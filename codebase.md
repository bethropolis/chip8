# Project Structure

```
chip8-wails/
├── chip8/
│   ├── chip8.go
│   └── chip8_test.go
├── frontend/
│   └── src/
│       ├── lib/
│       │   ├── DebugPanel.svelte
│       │   ├── LogViewer.svelte
│       │   ├── Notification.svelte
│       │   ├── ROMBrowser.svelte
│       │   └── SettingsModal.svelte
│       └── App.svelte
├── app.go
└── main.go
```

# Project Files

## File: `chip8/chip8.go`

```go
package chip8

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	DisplayWidth  = 64
	DisplayHeight = 32
	ProgramStart  = 0x200
	FontSetStart  = 0x50
)

// Chip8 represents the state of the CHIP-8 emulator
type Chip8 struct {
	Memory     [4096]byte
	Registers  [16]byte
	I          uint16
	PC         uint16
	Display    [DisplayWidth * DisplayHeight]byte
	DelayTimer byte
	SoundTimer byte
	Stack      [16]uint16
	SP         byte
	Keys       [16]bool
	DrawFlag   bool
	IsRunning  bool
	Breakpoints map[uint16]bool // New: Map to store breakpoint addresses
	randSource rand.Source
}

// FontSet contains the hexadecimal representations of the CHIP-8 font
var FontSet = []byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

// New creates and initializes a new Chip8 emulator
func New() *Chip8 {
	c := &Chip8{}
	c.Breakpoints = make(map[uint16]bool) // Initialize the map
	c.Reset()
	return c
}

// Reset initializes the Chip8 state to its default values
func (c *Chip8) Reset() {
	c.PC = ProgramStart
	c.I = 0
	c.SP = 0
	c.DelayTimer = 0
	c.SoundTimer = 0
	c.DrawFlag = false
	c.IsRunning = false

	// Clear memory, registers, display, and stack
	c.Memory = [4096]byte{}
	c.Registers = [16]byte{}
	c.Display = [DisplayWidth * DisplayHeight]byte{}
	c.Stack = [16]uint16{}
	c.Keys = [16]bool{}

	// Clear breakpoints on reset, but keep the map initialized
	for k := range c.Breakpoints {
		delete(c.Breakpoints, k)
	}

	// Load font set into memory
	for i := 0; i < len(FontSet); i++ {
		c.Memory[FontSetStart+i] = FontSet[i]
	}

	c.randSource = rand.NewSource(time.Now().UnixNano())
}

// LoadROM loads a ROM into the emulator's memory
func (c *Chip8) LoadROM(data []byte) error {
	if len(data) > len(c.Memory)-ProgramStart {
		return fmt.Errorf("ROM size %d exceeds available memory %d", len(data), len(c.Memory)-ProgramStart)
	}
	for i, b := range data {
		c.Memory[ProgramStart+i] = b
	}
	return nil
}

// EmulateCycle executes a single CHIP-8 CPU cycle
func (c *Chip8) EmulateCycle() {
	if !c.IsRunning {
		return
	}

	// Check for breakpoint at current PC
	if c.Breakpoints[c.PC] {
		c.IsRunning = false // Pause emulation
		return
	}

	// Fetch opcode
	opcode := uint16(c.Memory[c.PC])<<8 | uint16(c.Memory[c.PC+1])

	// Decode opcode parts
	vx := (opcode & 0x0F00) >> 8
	vy := (opcode & 0x00F0) >> 4
	nnn := opcode & 0x0FFF
	nn := byte(opcode & 0x00FF)
	n := byte(opcode & 0x000F)

	// Increment PC before execution (most common case)
	c.PC += 2

	switch opcode & 0xF000 {
	case 0x0000:
		switch opcode & 0x00FF {
		case 0x00E0: // CLS
			for i := range c.Display {
				c.Display[i] = 0
			}
			c.DrawFlag = true
		case 0x00EE: // RET
			c.SP--
			c.PC = c.Stack[c.SP]
		}
	case 0x1000: // JP addr
		c.PC = nnn
	case 0x2000: // CALL addr
		c.Stack[c.SP] = c.PC
		c.SP++
		c.PC = nnn
	case 0x3000: // SE Vx, byte
		if c.Registers[vx] == nn {
			c.PC += 2
		}
	case 0x4000: // SNE Vx, byte
		if c.Registers[vx] != nn {
			c.PC += 2
		}
	case 0x5000: // SE Vx, Vy
		if c.Registers[vx] == c.Registers[vy] {
			c.PC += 2
		}
	case 0x6000: // LD Vx, byte
		c.Registers[vx] = nn
	case 0x7000: // ADD Vx, byte
		c.Registers[vx] += nn
	case 0x8000:
		switch n {
		case 0x0: // LD Vx, Vy
			c.Registers[vx] = c.Registers[vy]
		case 0x1: // OR Vx, Vy
			c.Registers[vx] |= c.Registers[vy]
		case 0x2: // AND Vx, Vy
			c.Registers[vx] &= c.Registers[vy]
		case 0x3: // XOR Vx, Vy
			c.Registers[vx] ^= c.Registers[vy]
		case 0x4: // ADD Vx, Vy
			if uint16(c.Registers[vx])+uint16(c.Registers[vy]) > 255 {
				c.Registers[0xF] = 1
			} else {
				c.Registers[0xF] = 0
			}
			c.Registers[vx] += c.Registers[vy]
		case 0x5: // SUB Vx, Vy
			if c.Registers[vx] > c.Registers[vy] {
				c.Registers[0xF] = 1
			} else {
				c.Registers[0xF] = 0
			}
			c.Registers[vx] -= c.Registers[vy]
		case 0x6: // SHR Vx {, Vy}
			c.Registers[0xF] = c.Registers[vx] & 0x1
			c.Registers[vx] >>= 1
		case 0x7: // SUBN Vx, Vy
			if c.Registers[vy] > c.Registers[vx] {
				c.Registers[0xF] = 1
			} else {
				c.Registers[0xF] = 0
			}
			c.Registers[vx] = c.Registers[vy] - c.Registers[vx]
		case 0xE: // SHL Vx {, Vy}
			c.Registers[0xF] = c.Registers[vx] >> 7
			c.Registers[vx] <<= 1
		}
	case 0x9000: // SNE Vx, Vy
		if c.Registers[vx] != c.Registers[vy] {
			c.PC += 2
		}
	case 0xA000: // LD I, addr
		c.I = nnn
	case 0xB000: // JP V0, addr
		c.PC = nnn + uint16(c.Registers[0])
	case 0xC000: // RND Vx, byte
		r := rand.New(c.randSource)
		c.Registers[vx] = byte(r.Intn(256)) & nn
	case 0xD000: // DRW Vx, Vy, nibble
		xCoord := uint16(c.Registers[vx])
		yCoord := uint16(c.Registers[vy])
		height := uint16(n)
		c.Registers[0xF] = 0

		for yline := uint16(0); yline < height; yline++ {
			spriteByte := c.Memory[c.I+yline]
			for xline := uint16(0); xline < 8; xline++ {
				if (spriteByte & (0x80 >> xline)) != 0 {
					finalX := (xCoord + xline) % DisplayWidth
					finalY := (yCoord + yline) % DisplayHeight
					index := finalY*DisplayWidth + finalX

					if index < uint16(len(c.Display)) {
						if c.Display[index] == 1 {
							c.Registers[0xF] = 1
						}
						c.Display[index] ^= 1
					}
				}
			}
		}
		c.DrawFlag = true
	case 0xE000:
		switch nn {
		case 0x9E: // SKP Vx
			if c.Keys[c.Registers[vx]] {
				c.PC += 2
			}
		case 0xA1: // SKNP Vx
			if !c.Keys[c.Registers[vx]] {
				c.PC += 2
			}
		}
	case 0xF000:
		switch nn {
		case 0x07: // LD Vx, DT
			c.Registers[vx] = c.DelayTimer
		case 0x0A: // LD Vx, K
			keyPress := false
			for i, pressed := range c.Keys {
				if pressed {
					c.Registers[vx] = byte(i)
					keyPress = true
					break // Found a key, stop looking
				}
			}
			if !keyPress {
				c.PC -= 2 // Block by repeating this instruction
			}
		case 0x15: // LD DT, Vx
			c.DelayTimer = c.Registers[vx]
		case 0x18: // LD ST, Vx
			c.SoundTimer = c.Registers[vx]
		case 0x1E: // ADD I, Vx
			c.I += uint16(c.Registers[vx])
		case 0x29: // LD F, Vx
			c.I = uint16(c.Registers[vx])*5 + FontSetStart
		case 0x33: // LD B, Vx
			c.Memory[c.I] = c.Registers[vx] / 100
			c.Memory[c.I+1] = (c.Registers[vx] / 10) % 10
			c.Memory[c.I+2] = c.Registers[vx] % 10
		case 0x55: // LD [I], Vx
			for i := uint16(0); i <= vx; i++ {
				c.Memory[c.I+i] = c.Registers[i]
			}
			// Original interpreters incremented I after this operation. Many ROMs depend on this quirk.
			c.I += vx + 1
		case 0x65: // LD Vx, [I]
			for i := uint16(0); i <= vx; i++ {
				c.Registers[i] = c.Memory[c.I+i]
			}
			// Original interpreters also incremented I here.
			c.I += vx + 1
		}
	default:
		fmt.Printf("Unknown opcode: 0x%04X\n", opcode)
		}
}

// ClearDrawFlag resets the draw flag.
func (c *Chip8) ClearDrawFlag() {
	c.DrawFlag = false
}

// Disassemble takes an opcode and returns a human-readable string representation.
func Disassemble(opcode uint16) string {
	vx := (opcode & 0x0F00) >> 8
	vy := (opcode & 0x00F0) >> 4
	nnn := opcode & 0x0FFF
	nn := byte(opcode & 0x00FF)
	n := byte(opcode & 0x000F)

	switch opcode & 0xF000 {
	case 0x0000:
		switch opcode & 0x00FF {
		case 0x00E0:
			return fmt.Sprintf("0x%04X CLS", opcode)
		case 0x00EE:
			return fmt.Sprintf("0x%04X RET", opcode)
		default:
			return fmt.Sprintf("0x%04X SYS 0x%03X", opcode, nnn)
		}
	case 0x1000:
		return fmt.Sprintf("0x%04X JP 0x%03X", opcode, nnn)
	case 0x2000:
		return fmt.Sprintf("0x%04X CALL 0x%03X", opcode, nnn)
	case 0x3000:
		return fmt.Sprintf("0x%04X SE V%X, 0x%02X", opcode, vx, nn)
	case 0x4000:
		return fmt.Sprintf("0x%04X SNE V%X, 0x%02X", opcode, vx, nn)
	case 0x5000:
		return fmt.Sprintf("0x%04X SE V%X, V%X", opcode, vx, vy)
	case 0x6000:
		return fmt.Sprintf("0x%04X LD V%X, 0x%02X", opcode, vx, nn)
	case 0x7000:
		return fmt.Sprintf("0x%04X ADD V%X, 0x%02X", opcode, vx, nn)
	case 0x8000:
		switch n {
		case 0x0:
			return fmt.Sprintf("0x%04X LD V%X, V%X", opcode, vx, vy)
		case 0x1:
			return fmt.Sprintf("0x%04X OR V%X, V%X", opcode, vx, vy)
		case 0x2:
			return fmt.Sprintf("0x%04X AND V%X, V%X", opcode, vx, vy)
		case 0x3:
			return fmt.Sprintf("0x%04X XOR V%X, V%X", opcode, vx, vy)
		case 0x4:
			return fmt.Sprintf("0x%04X ADD V%X, V%X", opcode, vx, vy)
		case 0x5:
			return fmt.Sprintf("0x%04X SUB V%X, V%X", opcode, vx, vy)
		case 0x6:
			return fmt.Sprintf("0x%04X SHR V%X", opcode, vx)
		case 0x7:
			return fmt.Sprintf("0x%04X SUBN V%X, V%X", opcode, vx, vy)
		case 0xE:
			return fmt.Sprintf("0x%04X SHL V%X", opcode, vx)
		default:
			return fmt.Sprintf("0x%04X UNKNOWN 0x%04X", opcode, opcode)
		}
	case 0x9000:
		return fmt.Sprintf("0x%04X SNE V%X, V%X", opcode, vx, vy)
	case 0xA000:
		return fmt.Sprintf("0x%04X LD I, 0x%03X", opcode, nnn)
	case 0xB000:
		return fmt.Sprintf("0x%04X JP V0, 0x%03X", opcode, nnn)
	case 0xC000:
		return fmt.Sprintf("0x%04X RND V%X, 0x%02X", opcode, vx, nn)
	case 0xD000:
		return fmt.Sprintf("0x%04X DRW V%X, V%X, %d", opcode, vx, vy, n)
	case 0xE000:
		switch nn {
		case 0x9E:
			return fmt.Sprintf("0x%04X SKP V%X", opcode, vx)
		case 0xA1:
			return fmt.Sprintf("0x%04X SKNP V%X", opcode, vx)
		default:
			return fmt.Sprintf("0x%04X UNKNOWN 0x%04X", opcode, opcode)
		}
	case 0xF000:
		switch nn {
		case 0x07:
			return fmt.Sprintf("0x%04X LD V%X, DT", opcode, vx)
		case 0x0A:
			return fmt.Sprintf("0x%04X LD V%X, K", opcode, vx)
		case 0x15:
			return fmt.Sprintf("0x%04X LD DT, V%X", opcode, vx)
		case 0x18:
			return fmt.Sprintf("0x%04X LD ST, V%X", opcode, vx)
		case 0x1E:
			return fmt.Sprintf("0x%04X ADD I, V%X", opcode, vx)
		case 0x29:
			return fmt.Sprintf("0x%04X LD F, V%X", opcode, vx)
		case 0x33:
			return fmt.Sprintf("0x%04X LD B, V%X", opcode, vx)
		case 0x55:
			return fmt.Sprintf("0x%04X LD [I], V%X", opcode, vx)
		case 0x65:
			return fmt.Sprintf("0x%04X LD V%X, [I]", opcode, vx)
		default:
			return fmt.Sprintf("0x%04X UNKNOWN 0x%04X", opcode, opcode)
		}
	default:
		return fmt.Sprintf("0x%04X UNKNOWN 0x%04X", opcode, opcode)
	}
}
}

// GetState returns a snapshot of the CPU state for debugging.
func (c *Chip8) GetState() map[string]interface{} {
	disassembly := []string{}
	// Disassemble 10 instructions around the Program Counter for context
	for i := -4; i < 6; i++ {
		addr := int(c.PC) + (i * 2)
		if addr >= ProgramStart && addr < len(c.Memory)-1 {
			opcode := uint16(c.Memory[addr])<<8 | uint16(c.Memory[addr+1])
			line := fmt.Sprintf("0x%04X: %s", addr, Disassemble(opcode))
			if addr == int(c.PC) {
				line = "► " + line
			}
			disassembly = append(disassembly, line)
		}
	}

	// Create copies of register and stack arrays to avoid data races
	registersCopy := make([]byte, len(c.Registers))
	copy(registersCopy, c.Registers[:])
	stackCopy := make([]uint16, len(c.Stack))
	copy(stackCopy, c.Stack[:])

	return map[string]interface{}{
		"PC":          c.PC,
		"I":           c.I,
		"SP":          c.SP,
		"DelayTimer":  c.DelayTimer,
		"SoundTimer":  c.SoundTimer,
		"Registers":   registersCopy,
		"Stack":       stackCopy,
		"Disassembly": disassembly,
	}
}

```

## File: `chip8/chip8_test.go`

```go
package chip8

import (
	"testing"
)

func TestNewChip8(t *testing.T) {
	c := New()

	if c.PC != ProgramStart {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", ProgramStart, c.PC)
	}
	if c.I != 0 {
		t.Errorf("Expected I to be 0, got 0x%X", c.I)
	}
	if c.SP != 0 {
		t.Errorf("Expected SP to be 0, got %d", c.SP)
	}

	// Check if font set is loaded
	for i := 0; i < len(FontSet); i++ {
		if c.Memory[FontSetStart+i] != FontSet[i] {
			t.Errorf("FontSet not loaded correctly at 0x%X", FontSetStart+i)
		}
	}
}

func TestLoadROM(t *testing.T) {
	c := New()
	romData := []byte{0x12, 0x34, 0x56, 0x78}
	err := c.LoadROM(romData)

	if err != nil {
		t.Fatalf("LoadROM failed: %v", err)
	}

	for i, b := range romData {
		if c.Memory[ProgramStart+i] != b {
			t.Errorf("ROM byte at 0x%X expected 0x%X, got 0x%X", ProgramStart+i, b, c.Memory[ProgramStart+i])
		}
	}

	// Test ROM too large
	largeROM := make([]byte, 4096-ProgramStart+1)
	err = c.LoadROM(largeROM)
	if err == nil {
		t.Error("Expected error for large ROM, got nil")
	}
}

func TestOpcode00E0(t *testing.T) {
	c := New()
	c.Display[0] = 1 // Set a pixel
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0x00
	c.Memory[ProgramStart+1] = 0xE0

	c.EmulateCycle()

	if c.Display[0] != 0 {
		t.Errorf("Display not cleared, pixel at 0 is %d", c.Display[0])
	}
	if !c.DrawFlag {
		t.Error("DrawFlag not set")
	}
	if c.PC != ProgramStart+2 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", ProgramStart+2, c.PC)
	}
}

func TestOpcode00EE(t *testing.T) {
	c := New()
	// In a real CALL, the PC would already be incremented.
	// So we push the return address, e.g., 0x300
	c.Stack[0] = 0x300
	c.SP = 1
	c.PC = ProgramStart // The subroutine is at 0x200
	c.Memory[ProgramStart] = 0x00
	c.Memory[ProgramStart+1] = 0xEE

	c.EmulateCycle()

	if c.SP != 0 {
		t.Errorf("Expected SP to be 0, got %d", c.SP)
	}
	// After RET, PC should be exactly the address on the stack.
	// The next cycle's PC+=2 will then execute the next instruction.
	if c.PC != 0x300 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", 0x300, c.PC)
	}
}

func TestOpcode1NNN(t *testing.T) {
	c := New()
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0x12
	c.Memory[ProgramStart+1] = 0x34

	c.EmulateCycle()

	if c.PC != 0x0234 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", 0x0234, c.PC)
	}
}

func TestOpcode6XNN(t *testing.T) {
	c := New()
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0x6A
	c.Memory[ProgramStart+1] = 0x55 // Set V[A] to 0x55

	c.EmulateCycle()

	if c.Registers[0xA] != 0x55 {
		t.Errorf("Expected V[A] to be 0x%X, got 0x%X", 0x55, c.Registers[0xA])
	}
	if c.PC != ProgramStart+2 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", ProgramStart+2, c.PC)
	}
}

func TestOpcode7XNN(t *testing.T) {
	c := New()
	c.Registers[0xB] = 0x10 // Set V[B] to 0x10
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0x7B
	c.Memory[ProgramStart+1] = 0x05 // Add 0x05 to V[B]

	c.EmulateCycle()

	if c.Registers[0xB] != 0x15 {
		t.Errorf("Expected V[B] to be 0x%X, got 0x%X", 0x15, c.Registers[0xB])
	}
	if c.PC != ProgramStart+2 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", ProgramStart+2, c.PC)
	}
}

func TestOpcodeANNN(t *testing.T) {
	c := New()
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0xA1
	c.Memory[ProgramStart+1] = 0x23 // Set I to 0x123

	c.EmulateCycle()

	if c.I != 0x0123 {
		t.Errorf("Expected I to be 0x%X, got 0x%X", 0x0123, c.I)
	}
	if c.PC != ProgramStart+2 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", ProgramStart+2, c.PC)
	}
}

func TestOpcodeDXYN(t *testing.T) {
	c := New()
	c.Registers[0x0] = 0   // VX = 0
	c.Registers[0x1] = 0   // VY = 0
	c.I = FontSetStart     // Point I to a font character (e.g., '0')
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0xD0
	c.Memory[ProgramStart+1] = 0x15 // Draw sprite at (V0, V1), height 5

	c.EmulateCycle()

	if !c.DrawFlag {
		t.Error("DrawFlag not set")
	}
	if c.PC != ProgramStart+2 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", ProgramStart+2, c.PC)
	}

	// Check a few pixels that should be set for font '0'
	// Font '0': 0xF0, 0x90, 0x90, 0x90, 0xF0
	// (0,0) should be set (first bit of 0xF0)
	if c.Display[0] != 1 {
		t.Errorf("Expected pixel (0,0) to be 1, got %d", c.Display[0])
	}

	// (0,1) should be set (first bit of 0x90, second row)
	if c.Display[1*DisplayWidth+0] != 1 {
		t.Errorf("Expected pixel (0,1) to be 1, got %d", c.Display[1*DisplayWidth+0])
	}

	// Test collision (VF should be set)
	c.Registers[0xF] = 0 // Clear VF
	c.PC = ProgramStart
	c.EmulateCycle() // Draw again, causing collision

	if c.Registers[0xF] != 1 {
		t.Errorf("Expected VF to be 1 after collision, got %d", c.Registers[0xF])
	}
}

```

## File: `frontend/src/lib/DebugPanel.svelte`

```svelte
<script>
    import { onMount, onDestroy } from 'svelte';
    import { GetMemory, GetLogs, SetBreakpoint, ClearBreakpoint } from '../wailsjs/go/main/App';
    import { EventsOn } from '../wailsjs/runtime/runtime.js';
    import LogViewer from './LogViewer.svelte';

    export let debugState;

    let memoryData = new Uint8Array(256);
    let memoryOffset = 0x200; // Start at program memory
    let memoryLimit = 256;
    let memoryUpdateInterval;

    async function fetchMemoryView() {
        if (!debugState.PC) return; // Don't fetch if no ROM is loaded
        const data = await GetMemory(memoryOffset, memoryLimit);
        // data is base64, needs decoding
        memoryData = new Uint8Array(atob(data).split('').map(char => char.charCodeAt(0)));
    }

    onMount(() => {
        // Fetch memory periodically for the viewer
        memoryUpdateInterval = setInterval(fetchMemoryView, 200);
        // Debug state is now pushed from App.svelte, no need for EventsOn here.
    });

    onDestroy(() => {
        clearInterval(memoryUpdateInterval);
    });

    function formatByte(byte) {
        return byte.toString(16).padStart(2, '0').toUpperCase();
    }

    function formatAddress(address) {
        return '0x' + address.toString(16).padStart(4, '0').toUpperCase();
    }

    function handleMemoryScroll(event) {
        const target = event.target;
        // Simple scroll: just move by a fixed amount
        if (event.deltaY > 0) {
            memoryOffset += 16;
        } else {
            memoryOffset -= 16;
        }

        if (memoryOffset < 0) memoryOffset = 0;
        if (memoryOffset > 4096 - memoryLimit) memoryOffset = 4096 - memoryLimit;

        // Prevent page scroll
        event.preventDefault();
        fetchMemoryView(); // Fetch new view on scroll
    }

    async function toggleBreakpoint(address) {
        if (debugState.Breakpoints && debugState.Breakpoints[address]) {
            await ClearBreakpoint(address);
        } else {
            await SetBreakpoint(address);
        }
    }
</script>

<div class="grid grid-cols-1 md:grid-cols-3 gap-4 p-4 overflow-y-auto h-full">
    <!-- CPU Registers -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700">
        <h3 class="font-bold text-lg mb-2">CPU Registers</h3>
        <div class="grid grid-cols-2 gap-x-4 text-sm font-mono">
            {#each { length: 8 } as _, i}
                <span>V{i.toString(16).toUpperCase()}: {`0x${debugState.Registers?.[i]?.toString(16).padStart(2, "0").toUpperCase() ?? "00"}`}</span>
                <span>V{(i + 8).toString(16).toUpperCase()}: {`0x${debugState.Registers?.[i + 8]?.toString(16).padStart(2, "0").toUpperCase() ?? "00"}`}</span>
            {/each}
        </div>
    </div>

    <!-- System State -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700">
        <h3 class="font-bold text-lg mb-2">System State</h3>
        <div class="text-sm font-mono">
            <p>PC: {`0x${debugState.PC?.toString(16).padStart(4, "0").toUpperCase() ?? "0000"}`}</p>
            <p>I: {`0x${debugState.I?.toString(16).padStart(4, "0").toUpperCase() ?? "0000"}`}</p>
            <p>SP: {`0x${debugState.SP?.toString(16).padStart(2, "0").toUpperCase() ?? "00"}`}</p>
            <p>Delay Timer: {debugState.DelayTimer ?? "0"}</p>
            <p>Sound Timer: {debugState.SoundTimer ?? "0"}</p>
        </div>
    </div>

    <!-- Stack -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700">
        <h3 class="font-bold text-lg mb-2">Stack</h3>
        <pre class="text-sm overflow-y-auto h-24 bg-slate-800 p-2 rounded-md border border-slate-700 font-mono">
            {#each debugState.Stack || [] as value, i}
                <div class:text-cyan-400={i === (debugState.SP > 0 ? debugState.SP -1 : 0)}>Stack[{i.toString(16).toUpperCase()}]: 0x{value.toString(16).padStart(4, "0").toUpperCase()}</div>
            {/each}
        </pre>
    </div>

    <!-- Disassembly -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700 md:col-span-1">
        <h3 class="font-bold text-lg mb-2">Disassembly</h3>
        <pre class="text-xs leading-tight overflow-y-auto bg-slate-800 p-2 rounded-md border border-slate-700 h-64 font-mono">
            {#each debugState.Disassembly || [] as line}
                {@const address = parseInt(line.split(":")[0].replace("► ", ""), 16)}
                <div
                    class:text-cyan-400={line.startsWith("►")}
                    class:bg-red-700={debugState.Breakpoints && debugState.Breakpoints[address]}
                    on:click={() => toggleBreakpoint(address)}
                    class="cursor-pointer hover:bg-gray-600"
                >{line}</div>
            {/each}
        </pre>
    </div>

    <!-- Memory Viewer -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700 md:col-span-2">
        <h3 class="font-bold text-lg mb-2">Memory Viewer (scroll to navigate)</h3>
        <div class="text-sm overflow-hidden h-64 bg-slate-800 p-2 rounded-md border border-slate-700 font-mono" on:wheel={handleMemoryScroll}>
            {#each Array(Math.ceil(memoryData.length / 16)) as _, rowIdx}
                <div class="flex">
                    <span class="text-gray-500 mr-2">{formatAddress(memoryOffset + rowIdx * 16)}:</span>
                    {#each Array(16) as _, colIdx}
                        {@const byte = memoryData[rowIdx * 16 + colIdx]}
                        <span class="mr-1">{byte !== undefined ? formatByte(byte) : "--"}</span>
                    {/each}
                </div>
            {/each}
        </div>
    </div>

    <!-- Logs -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700 col-span-3">
        <h3 class="font-bold text-lg mb-2">Application Logs</h3>
        <LogViewer />
    </div>
</div>
```

## File: `frontend/src/lib/LogViewer.svelte`

```svelte
<script>
    import { onMount, onDestroy } from 'svelte';
    import { GetLogs } from '../wailsjs/go/main/App';

    let logs = [];
    let intervalId;
    let logViewerElement;

    async function fetchLogs() {
        logs = await GetLogs();
        // Scroll to bottom on new logs
        if (logViewerElement) {
            logViewerElement.scrollTop = logViewerElement.scrollHeight;
        }
    }

    onMount(() => {
        fetchLogs();
        intervalId = setInterval(fetchLogs, 500); // Fetch logs every 500ms
    });

    onDestroy(() => {
        clearInterval(intervalId);
    });
</script>

<div class="bg-slate-800 p-2 rounded-md border border-slate-700 font-mono text-xs overflow-y-scroll h-64" bind:this={logViewerElement}>
    {#each logs as log}
        <div>{log}</div>
    {/each}
</div>

<style>
    /* Add any specific styles for the log viewer here */
</style>

```

## File: `frontend/src/lib/Notification.svelte`

```svelte
<script>
    import { fade } from 'svelte/transition';
    import { createEventDispatcher } from 'svelte';

    export let message;
    export let type = 'info'; // 'info', 'success', 'warning', 'error'
    export let duration = 3000; // milliseconds

    const dispatch = createEventDispatcher();
    let timeout;

    function dismiss() {
        clearTimeout(timeout);
        dispatch('dismiss');
    }

    $: if (message) {
        clearTimeout(timeout);
        timeout = setTimeout(dismiss, duration);
    }

    let bgColorClass;
    $: {
        switch (type) {
            case 'success':
                bgColorClass = 'bg-green-500';
                break;
            case 'warning':
                bgColorClass = 'bg-yellow-500';
                break;
            case 'error':
                bgColorClass = 'bg-red-500';
                break;
            case 'info':
            default:
                bgColorClass = 'bg-blue-500';
        }
    }
</script>

{#if message}
    <div
        in:fade={{ duration: 150 }}
        out:fade={{ duration: 150 }}
        class="fixed bottom-4 right-4 p-4 rounded-lg shadow-lg text-white flex items-center space-x-3 z-50 {bgColorClass}"
        role="alert"
    >
        <span>{message}</span>
        <button on:click={dismiss} class="ml-auto text-white opacity-75 hover:opacity-100">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
            </svg>
        </button>
    </div>
{/if}

```

## File: `frontend/src/lib/ROMBrowser.svelte`

```svelte
<script>
    import { onMount } from 'svelte';
    import { GetROMs, LoadROM } from '../wailsjs/go/main/App';
    import { showNotification } from '../App.svelte'; // Assuming showNotification is exported
    import { Play } from 'lucide-svelte'; // New import for Play icon

    let roms = [];
    let selectedROM = '';

    async function fetchROMs() {
        try {
            roms = await GetROMs();
            if (roms.length === 0) {
                showNotification("No ROMs found in ./roms directory.", "warning");
            }
        } catch (error) {
            showNotification(`Failed to load ROM list: ${error.message}`, "error");
            console.error("Error fetching ROMs:", error);
        }
    }

    async function handleLoadSelectedROM() {
        if (selectedROM) {
            try {
                // LoadROM expects the full path, but GetROMs only returns the name.
                // We need to pass the full path to LoadROM.
                // For now, assuming ROMs are in a known 'roms' subdirectory relative to the app executable.
                // A more robust solution would involve passing the full path from Go or letting the user select.
                await LoadROM(selectedROM); // This will need adjustment if LoadROM expects full path
                showNotification(`ROM loaded: ${selectedROM}`, "success");
            } catch (error) {
                showNotification(`Failed to load ROM: ${error.message}`, "error");
                console.error("Error loading selected ROM:", error);
            }
        } else {
            showNotification("Please select a ROM first.", "warning");
        }
    }

    onMount(() => {
        fetchROMs();
    });
</script>

<div class="bg-gray-800 p-4 rounded-lg shadow-md">
    <h3 class="text-xl font-semibold mb-3 text-center text-cyan-400">ROM Browser</h3>
    <div class="mb-4">
        <select bind:value={selectedROM} class="w-full p-2 rounded-md bg-gray-700 border border-gray-600 text-gray-200">
            <option value="">-- Select a ROM --</option>
            {#each roms as rom}
                <option value={rom}>{rom}</option>
            {/each}
        </select>
    </div>
    <button
        on:click={handleLoadSelectedROM}
        class="w-full flex items-center justify-center space-x-2 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-md transition-colors duration-200"
    >
        <Play size={18} />
        <span>Load Selected ROM</span>
    </button>
</div>

```

## File: `frontend/src/lib/SettingsModal.svelte`

```svelte
<script>
    import { createEventDispatcher } from "svelte";

    export let showModal;
    export let currentClockSpeed;
    export let currentDisplayColor;
    export let currentScanlineEffect;
    export let currentDisplayScale;
    export let currentKeyMap; // New prop for key remapping

    const dispatch = createEventDispatcher();

    let newClockSpeed = currentClockSpeed;
    let newDisplayColor = currentDisplayColor;
    let newScanlineEffect = currentScanlineEffect;
    let newDisplayScale = currentDisplayScale;
    let newKeyMap = { ...currentKeyMap }; // Clone the current key map

    let remappingKey = null; // Stores the CHIP-8 key being remapped

    function closeModal() {
        showModal = false;
    }

    function saveSettings() {
        dispatch("save", {
            clockSpeed: newClockSpeed,
            displayColor: newDisplayColor,
            scanlineEffect: newScanlineEffect,
            displayScale: newDisplayScale,
            keyMap: newKeyMap, // Dispatch the new key map
        });
        closeModal();
    }

    function startRemap(event, chip8Key) {
        remappingKey = chip8Key;
        event.target.value = ""; // Clear input for new key press
        event.target.placeholder = "Press a key...";
        window.addEventListener("keydown", handleRemapKeyDown, { once: true });
    }

    function handleRemapKeyDown(event) {
        event.preventDefault();
        if (remappingKey !== null) {
            newKeyMap[remappingKey] = event.key.toLowerCase();
            remappingKey = null;
        }
    }

    function endRemap(event) {
        if (remappingKey !== null) {
            // If user blurs without pressing a key, revert to original
            newKeyMap[remappingKey] = currentKeyMap[remappingKey];
            remappingKey = null;
        }
        event.target.placeholder = "";
    }
</script>

{#if showModal}
    <div
        class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
    >
        <div
            class="bg-[#34495e] p-6 rounded-lg shadow-xl border border-gray-700 w-96"
        >
            <h2 class="text-2xl font-bold mb-4 text-center text-cyan-400">
                Settings
            </h2>

            <div class="mb-4">
                <label
                    for="clockSpeed"
                    class="block text-gray-300 text-sm font-bold mb-2"
                    >CPU Clock Speed (Hz):</label
                >
                <input
                    type="range"
                    id="clockSpeed"
                    min="100"
                    max="2000"
                    step="50"
                    bind:value={newClockSpeed}
                    class="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700"
                />
                <div class="text-center text-gray-400 text-sm mt-1">
                    {newClockSpeed} Hz
                </div>
            </div>

            <div class="mb-4">
                <label
                    for="speedPresets"
                    class="block text-gray-300 text-sm font-bold mb-2"
                    >Speed Presets:</label
                >
                <div id="speedPresets" class="mt-2 flex flex-wrap gap-2">
                    <label class="inline-flex items-center">
                        <input
                            type="radio"
                            class="form-radio"
                            name="speedPreset"
                            value={700}
                            bind:group={newClockSpeed}
                        />
                        <span class="ml-2 text-gray-300">Original (700Hz)</span>
                    </label>
                    <label class="inline-flex items-center">
                        <input
                            type="radio"
                            class="form-radio"
                            name="speedPreset"
                            value={1400}
                            bind:group={newClockSpeed}
                        />
                        <span class="ml-2 text-gray-300">Fast (1400Hz)</span>
                    </label>
                    <label class="inline-flex items-center">
                        <input
                            type="radio"
                            class="form-radio"
                            name="speedPreset"
                            value={2000}
                            bind:group={newClockSpeed}
                        />
                        <span class="ml-2 text-gray-300">Turbo (2000Hz)</span>
                    </label>
                </div>
            </div>

            <div class="mb-4">
                <label
                    for="displayColor"
                    class="block text-gray-300 text-sm font-bold mb-2"
                    >Display Color:</label
                >
                <div id="displayColor" class="mt-2">
                    <label class="inline-flex items-center mr-4">
                        <input
                            type="radio"
                            class="form-radio"
                            name="displayColor"
                            value="#33FF00"
                            bind:group={newDisplayColor}
                        />
                        <span class="ml-2 text-gray-300">Classic Green</span>
                    </label>
                    <label class="inline-flex items-center mr-4">
                        <input
                            type="radio"
                            class="form-radio"
                            name="displayColor"
                            value="#FFFFFF"
                            bind:group={newDisplayColor}
                        />
                        <span class="ml-2 text-gray-300">White</span>
                    </label>
                    <label class="inline-flex items-center">
                        <input
                            type="radio"
                            class="form-radio"
                            name="displayColor"
                            value="#FFBF00"
                            bind:group={newDisplayColor}
                        />
                        <span class="ml-2 text-gray-300">Amber</span>
                    </label>
                </div>
            </div>

            <div class="mb-4">
                <label
                    for="displayScaling"
                    class="block text-gray-300 text-sm font-bold mb-2"
                    >Display Scaling:</label
                >
                <div id="displayScaling" class="mt-2">
                    <div class="mb-4">
                        <label
                            class="block text-gray-300 text-sm font-bold mb-2"
                            >Display Color:</label
                        >
                        <div class="mt-2">
                            <label class="inline-flex items-center mr-4">
                                <input
                                    type="radio"
                                    class="form-radio"
                                    name="displayColor"
                                    value="#33FF00"
                                    bind:group={newDisplayColor}
                                />
                                <span class="ml-2 text-gray-300"
                                    >Classic Green</span
                                >
                            </label>
                            <label class="inline-flex items-center mr-4">
                                <input
                                    type="radio"
                                    class="form-radio"
                                    name="displayColor"
                                    value="#FFFFFF"
                                    bind:group={newDisplayColor}
                                />
                                <span class="ml-2 text-gray-300">White</span>
                            </label>
                            <label class="inline-flex items-center">
                                <input
                                    type="radio"
                                    class="form-radio"
                                    name="displayColor"
                                    value="#FFBF00"
                                    bind:group={newDisplayColor}
                                />
                                <span class="ml-2 text-gray-300">Amber</span>
                            </label>
                        </div>
                    </div>

                    <div class="mb-4">
                        <label class="inline-flex items-center">
                            <input
                                type="checkbox"
                                class="form-checkbox"
                                bind:checked={newScanlineEffect}
                            />
                            <span class="ml-2 text-gray-300"
                                >Enable Scanline Effect</span
                            >
                        </label>
                    </div>

                    <div class="mb-4">
                        <h3 class="text-lg font-bold mb-2 text-cyan-400">
                            Key Remapping
                        </h3>
                        <p class="text-sm text-gray-400 mb-2">
                            Click a CHIP-8 key and then press the desired
                            keyboard key.
                        </p>
                        <div class="grid grid-cols-4 gap-2 text-center">
                            {#each Object.entries(newKeyMap) as [chip8Key, keyboardKey]}
                                <div
                                    class="bg-gray-700 p-2 rounded-md border border-gray-600"
                                >
                                    <span class="text-gray-300"
                                        >{chip8Key.toUpperCase()}:</span
                                    >
                                    <input
                                        type="text"
                                        value={keyboardKey}
                                        on:focus={(e) =>
                                            startRemap(e, chip8Key)}
                                        on:blur={endRemap}
                                        class="w-full bg-gray-600 text-white text-center rounded-sm mt-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                        readonly
                                    />
                                </div>
                            {/each}
                        </div>
                    </div>

                    <div class="flex justify-end gap-3">
                        <button
                            on:click={closeModal}
                            class="bg-gray-600 hover:bg-gray-700 text-white font-bold py-2 px-4 rounded-md transition-colors"
                        >
                            Cancel
                        </button>
                        <button
                            on:click={saveSettings}
                            class="bg-green-600 hover:bg-green-700 text-white font-bold py-2 px-4 rounded-md transition-colors"
                        >
                            Save
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
{/if}

<style>
    /* Add any specific styles for the modal here if needed */
</style>

```

## File: `frontend/src/App.svelte`

```svelte
<script>
    import { onMount } from "svelte";
    import { EventsOn } from "./wailsjs/runtime/runtime.js";
    import {
        Reset,
        TogglePause,
        KeyDown,
        KeyUp,
        FrontendReady,
        GetInitialState,
        SetClockSpeed,
        SaveScreenshot, // New import
        SaveState, // New import
        SaveStateToFile, // New import
        LoadStateFromFile, // New import
        SoftReset, // New import
        HardReset, // New import
    } from "./wailsjs/go/main/App.js";
    import SettingsModal from "./lib/SettingsModal.svelte";
    import DebugPanel from "./lib/DebugPanel.svelte";
    import Notification from "./lib/Notification.svelte";
    import ROMBrowser from "./lib/ROMBrowser.svelte";
    import {
        Settings,
        RotateCcw,
        Play,
        Pause,
        Camera,
        Save,
        Upload,
    } from "lucide-svelte";

    // --- UI Elements & State ---
    let canvasElement;
    let debugState = {
        Registers: Array(16).fill(0),
        Disassembly: [],
        Stack: Array(16).fill(0),
        PC: 0, I: 0, SP: 0,
        DelayTimer: 0, SoundTimer: 0,
    };
    let statusMessage = "Status: Idle | ROM: None";
    let isPaused = true;
    let showSettingsModal = false;
    let currentClockSpeed = 700;
    let currentDisplayColor = "#33FF00";
    let currentScanlineEffect = false;
    let currentDisplayScale = 1;
    let currentTab = "emulator";
    let currentDisplayBuffer = new Uint8Array(DISPLAY_WIDTH * DISPLAY_HEIGHT); // Store current display state

    let notificationMessage = "";
    let notificationType = "info";
    let showResetOptions = false; // New state for reset options dropdown

    // --- Display Constants ---
    const SCALE = 10;
    const DISPLAY_WIDTH = 64;
    const DISPLAY_HEIGHT = 32;

    // --- Keypad Mapping ---
    // Make keyMap mutable for remapping
    let keyMap = {
        "1": 0x1, "2": 0x2, "3": 0x3, "4": 0xc,
        q: 0x4, w: 0x5, e: 0x6, r: 0xd,
        a: 0x7, s: 0x8, d: 0x9, f: 0xe,
        z: 0xa, x: 0x0, c: 0xb, v: 0xf,
    };
    let pressedKeys = {};

    // --- Audio ---
    let audioContext;
    let oscillator;

    function playBeep() {
        if (!audioContext) {
            audioContext = new (window.AudioContext || window.webkitAudioContext)();
        }
        if (oscillator) {
            oscillator.stop();
            oscillator.disconnect();
        }
        oscillator = audioContext.createOscillator();
        oscillator.type = "sine";
        oscillator.frequency.setValueAtTime(440, audioContext.currentTime);
        oscillator.connect(audioContext.destination);
        oscillator.start();
        oscillator.stop(audioContext.currentTime + 0.1);
    }

    function drawDisplay(canvas, displayBuffer) {
        if (!canvas || !displayBuffer) return;
        const ctx = canvas.getContext("2d");
        if (!ctx) return;

        ctx.fillStyle = "#000000";
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        ctx.fillStyle = currentDisplayColor;
        for (let y = 0; y < DISPLAY_HEIGHT; y++) {
            for (let x = 0; x < DISPLAY_WIDTH; x++) {
                if (displayBuffer[y * DISPLAY_WIDTH + x]) {
                    ctx.fillRect(x * SCALE, y * SCALE, SCALE, SCALE);
                }
            }
        }

        if (currentScanlineEffect) {
            ctx.fillStyle = "rgba(0, 0, 0, 0.3)";
            for (let y = 0; y < DISPLAY_HEIGHT; y += 2) {
                ctx.fillRect(0, y * SCALE, canvas.width, SCALE);
            }
        }
    }

    onMount(async () => {
        // 1. Setup listeners FIRST
        EventsOn("displayUpdate", (base64DisplayBuffer) => {
            if (animationFrameId) cancelAnimationFrame(animationFrameId);
            animationFrameId = requestAnimationFrame(() => {
                const binaryString = atob(base64DisplayBuffer);
                const len = binaryString.length;
                const bytes = new Uint8Array(len);
                for (let i = 0; i < len; i++) {
                    bytes[i] = binaryString.charCodeAt(i);
                }
                drawDisplay(canvasElement, bytes);
                currentDisplayBuffer = bytes;
            });
        });

        EventsOn("debugUpdate", (newState) => {
            debugState = newState;
        });

        EventsOn("statusUpdate", (newStatus) => {
            statusMessage = newStatus;
        });

        EventsOn("clockSpeedUpdate", (speed) => {
            currentClockSpeed = speed;
        });

        EventsOn("playBeep", playBeep);

        // 2. THEN, tell backend we are ready
        await FrontendReady();

        // 3. FINALLY, pull the initial state to populate the UI
        const initialState = await GetInitialState();
        debugState = initialState;
        currentDisplayBuffer = new Uint8Array(DISPLAY_WIDTH * DISPLAY_HEIGHT); // Start with a blank buffer
        drawDisplay(canvasElement, currentDisplayBuffer);
    });

    // Reactive redraw when canvas element is available and tab is emulator
    let animationFrameId; // Declare animationFrameId here
    $: if (canvasElement && currentTab === "emulator") {
        drawDisplay(canvasElement, currentDisplayBuffer);
    }

    // Reverse map for finding CHIP-8 key from keyboard key
    let reverseKeyMap = {};
    $: {
        reverseKeyMap = {};
        for(const [keyboardKey, chip8Key] of Object.entries(keyMap)){
            reverseKeyMap[keyboardKey] = chip8Key;
        }
    }

    window.addEventListener("keydown", (e) => {
        const key = e.key.toLowerCase();
        const chip8Key = reverseKeyMap[key];
        if (chip8Key !== undefined) {
            e.preventDefault();
            KeyDown(chip8Key);
            pressedKeys = { ...pressedKeys, [chip8Key]: true };
        }
    });

    window.addEventListener("keyup", (e) => {
        const key = e.key.toLowerCase();
        const chip8Key = reverseKeyMap[key];
        if (chip8Key !== undefined) {
            e.preventDefault();
            KeyUp(chip8Key);
            pressedKeys = { ...pressedKeys, [chip8Key]: false };
        }
    });

    async function handleReset() {
        await Reset();
        isPaused = true;
        showNotification("Emulator reset!", "info");
    }

    function toggleResetOptions() {
        showResetOptions = !showResetOptions;
    }

    async function handleSoftReset() {
        try {
            await SoftReset();
            isPaused = false;
            showNotification("Soft reset complete! ROM reloaded.", "success");
        } catch (error) {
            showNotification(`Soft reset failed: ${error.message}`, "error");
            console.error("Soft reset error:", error);
        }
        showResetOptions = false;
    }

    async function handleHardReset() {
        try {
            await HardReset();
            isPaused = true;
            showNotification("Hard reset complete! ROM cleared.", "info");
        } catch (error) {
            showNotification(`Hard reset failed: ${error.message}`, "error");
            console.error("Hard reset error:", error);
        }
        showResetOptions = false;
    }

    async function handleTogglePause() {
        isPaused = await TogglePause();
    }

    function openSettings() {
        showSettingsModal = true;
    }

    async function handleSaveSettings(event) {
        const { clockSpeed, displayColor, scanlineEffect, keyMap: newKeyMap } = event.detail;
        await SetClockSpeed(clockSpeed);
        currentClockSpeed = clockSpeed;
        currentDisplayColor = displayColor;
        currentScanlineEffect = scanlineEffect;
        keyMap = newKeyMap; // Update keymap
    }

    async function handleScreenshot() {
        if (!canvasElement) {
            showNotification("Canvas not available for screenshot.", "error");
            return;
        }
        try {
            const dataURL = canvasElement.toDataURL("image/png");
            const base64Data = dataURL.split(",")[1];
            await SaveScreenshot(base64Data);
            showNotification("Screenshot saved!", "success");
        } catch (error) {
            showNotification(`Failed to save screenshot: ${error}`, "error");
        }
    }

    async function handleSaveState() {
        try {
            const state = await SaveState();
            await SaveStateToFile(state);
            showNotification("Emulator state saved!", "success");
        } catch (error) {
            showNotification(`Failed to save state: ${error}`, "error");
        }
    }

    async function handleLoadState() {
        try {
            await LoadStateFromFile();
            showNotification("Emulator state loaded!", "success");
        } catch (error) {
            showNotification(`Failed to load state: ${error}`, "error");
        }
    }

    export function showNotification(message, type = "info") {
        notificationMessage = message;
        notificationType = type;
    }

    function dismissNotification() {
        notificationMessage = "";
    }
</script>

<div class="flex flex-col h-screen bg-gray-900 text-gray-100 font-sans antialiased">
    <!-- Top Bar -->
    <header class="flex-none bg-gray-800 text-gray-100 shadow-lg z-10">
        <div class="container mx-auto px-4 py-3 flex items-center justify-between">
            <div class="flex items-center space-x-4">
                <h1 class="text-2xl font-bold text-cyan-400">CHIP-8 Emulator</h1>
                <nav class="flex space-x-2">
                    <button on:click={() => (currentTab = "emulator")}
                        class="px-4 py-2 rounded-md text-sm font-medium transition-colors duration-200"
                        class:bg-blue-600={currentTab === "emulator"}
                        class:hover:bg-blue-700={currentTab === "emulator"}
                        class:text-white={currentTab === "emulator"}
                        class:text-gray-300={currentTab !== "emulator"}
                        class:hover:text-white={currentTab !== "emulator"}>Emulator</button>
                    <button on:click={() => (currentTab = "debug")}
                        class="px-4 py-2 rounded-md text-sm font-medium transition-colors duration-200"
                        class:bg-blue-600={currentTab === "debug"}
                        class:hover:bg-blue-700={currentTab === "debug"}
                        class:text-white={currentTab === "debug"}
                        class:text-gray-300={currentTab !== "debug"}
                        class:hover:text-white={currentTab !== "debug"}>Debug</button>
                </nav>
            </div>
            <button on:click={openSettings} class="p-2 rounded-full hover:bg-gray-700 transition-colors duration-200" title="Settings">
                <Settings size={20} />
            </button>
        </div>
    </header>

    <!-- Main Content Area -->
    <main class="flex-grow overflow-hidden">
        {#if currentTab === "emulator"}
            <div class="flex flex-col md:flex-row h-full p-4 space-y-4 md:space-y-0 md:space-x-4">
                <section class="flex-grow flex items-center justify-center bg-gray-800 rounded-lg shadow-md p-4">
                    <canvas bind:this={canvasElement} width={DISPLAY_WIDTH * SCALE} height={DISPLAY_HEIGHT * SCALE} class="border-2 border-cyan-500 rounded-md"></canvas>
                </section>
                <aside class="flex-none w-full md:w-80 flex flex-col space-y-4">
                    <ROMBrowser />
                    <div class="bg-gray-800 p-4 rounded-lg shadow-md">
                        <h2 class="text-xl font-semibold mb-3 text-center text-cyan-400">Controls</h2>
                        <div class="grid grid-cols-2 gap-3">
                                                        <button
                                on:click={handleReset}
                                class="flex items-center justify-center space-x-2 bg-yellow-500 hover:bg-yellow-600 text-white font-medium py-2 px-4 rounded-md transition-colors duration-200"
                                title="Reset the emulator state"
                            >
                                <RotateCcw size={18} />
                                <span>Reset</span>
                            </button>
                            <div class="relative inline-block text-left w-full">
                                <button
                                    on:click={toggleResetOptions}
                                    class="flex items-center justify-center space-x-2 bg-yellow-500 hover:bg-yellow-600 text-white font-medium py-2 px-4 rounded-md transition-colors duration-200 w-full"
                                    title="Reset Options"
                                >
                                    <RotateCcw size={18} />
                                    <span>Reset Options</span>
                                </button>
                                {#if showResetOptions}
                                    <div class="origin-top-right absolute right-0 mt-2 w-56 rounded-md shadow-lg bg-gray-700 ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
                                        <div class="py-1" role="menu" aria-orientation="vertical" aria-labelledby="options-menu">
                                            <button
                                                on:click={handleSoftReset}
                                                class="block w-full text-left px-4 py-2 text-sm text-gray-200 hover:bg-gray-600 hover:text-white"
                                                role="menuitem"
                                            >Soft Reset (Reload ROM)</button>
                                            <button
                                                on:click={handleHardReset}
                                                class="block w-full text-left px-4 py-2 text-sm text-gray-200 hover:bg-gray-600 hover:text-white"
                                                role="menuitem"
                                            >Hard Reset (Clear All)</button>
                                        </div>
                                    </div>
                                {/if}
                            </div>
                            <button on:click={handleTogglePause} class="flex items-center justify-center space-x-2 bg-green-600 hover:bg-green-700 text-white font-medium py-2 px-4 rounded-md transition-colors duration-200" title={isPaused ? "Resume emulation" : "Pause emulation"}>
                                {#if isPaused}<Play size={18} /><span>Resume</span>{:else}<Pause size={18} /><span>Pause</span>{/if}
                            </button>
                            <button on:click={handleScreenshot} class="flex items-center justify-center space-x-2 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-md transition-colors duration-200" title="Take a screenshot of the display"><Camera size={18} /><span>Screenshot</span></button>
                            <button on:click={handleSaveState} class="flex items-center justify-center space-x-2 bg-purple-600 hover:bg-purple-700 text-white font-medium py-2 px-4 rounded-md transition-colors duration-200" title="Save current emulator state"><Save size={18} /><span>Save State</span></button>
                             <button on:click={handleLoadState} class="flex items-center justify-center space-x-2 bg-purple-600 hover:bg-purple-700 text-white font-medium py-2 px-4 rounded-md transition-colors duration-200" title="Load emulator state from file"><Upload size={18} /><span>Load State</span></button>
                        </div>
                    </div>
                    <div class="bg-gray-800 p-4 rounded-lg shadow-md">
                        <h2 class="text-xl font-semibold mb-3 text-center text-cyan-400">CHIP-8 Keypad</h2>
                        <div class="grid grid-cols-4 gap-2 text-center">
                            {#each [1, 2, 3, 0xC, 4, 5, 6, 0xD, 7, 8, 9, 0xE, 0xA, 0, 0xB, 0xF] as chip8Key}
                                <div class="bg-gray-700 p-3 rounded-md border border-gray-600 text-lg font-bold flex items-center justify-center aspect-square transition-colors duration-100" class:bg-blue-500={pressedKeys[chip8Key]} class:border-blue-400={pressedKeys[chip8Key]}>
                                    {chip8Key.toString(16).toUpperCase()}
                                </div>
                            {/each}
                        </div>
                        <p class="text-xs text-center mt-3 text-gray-400">Keys: 1-4, Q-R, A-F, Z-V</p>
                    </div>
                </aside>
            </div>
        {:else if currentTab === "debug"}
            <DebugPanel bind:debugState />
        {/if}
    </main>
    <footer class="flex-none bg-gray-800 text-gray-300 text-sm text-center py-3 shadow-inner">{statusMessage}</footer>
    <Notification message={notificationMessage} type={notificationType} on:dismiss={dismissNotification}/>
</div>
{#if showSettingsModal}
    <SettingsModal bind:showModal={showSettingsModal} currentClockSpeed={currentClockSpeed} currentDisplayColor={currentDisplayColor} currentScanlineEffect={currentScanlineEffect} currentDisplayScale={currentDisplayScale} currentKeyMap={keyMap} on:save={handleSaveSettings} />
{/if}
```

## File: `app.go`

```go
package main

import (
	"bytes"
	"chip8-wails/chip8"
	"context"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx           context.Context
	cpu           *chip8.Chip8
	frontendReady chan struct{}
	cpuSpeed      time.Duration // Use time.Duration for clarity
	logBuffer     []string
	logMutex      sync.Mutex
	isPaused      bool
	pauseMutex    sync.Mutex
	romLoaded     []byte // Store the loaded ROM data for soft reset
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		cpu:           chip8.New(),
		frontendReady: make(chan struct{}),
		cpuSpeed:      time.Second / 700, // Default to 700Hz
		logBuffer:     make([]string, 0, 100),
		isPaused:      true, // Start paused
	}
}

func (a *App) appendLog(msg string) {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()
	log.Println(msg) // Also log to console for easier debugging
	if len(a.logBuffer) >= 100 {
		a.logBuffer = a.logBuffer[1:]
	}
	a.logBuffer = append(a.logBuffer, time.Now().Format("15:04:05")+" | "+msg)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Ensure the 'roms' directory exists
	if _, err := os.Stat("./roms"); os.IsNotExist(err) {
		os.Mkdir("./roms", 0755)
		a.appendLog("Created 'roms' directory. Please place your .ch8 files here.")
	}
	go a.runEmulator()
}

// --- Frontend Ready Signal ---

func (a *App) FrontendReady() {
	close(a.frontendReady)
}

// --- Main Emulator Loop ---

func (a *App) runEmulator() {
	<-a.frontendReady // Wait for the frontend to be ready

	// Create tickers ONCE, outside the loop
	cpuTicker := time.NewTicker(a.cpuSpeed)
	timerTicker := time.NewTicker(time.Second / 60) // 60Hz for timers/refresh
	defer cpuTicker.Stop()
	defer timerTicker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return

		case <-cpuTicker.C:
			// Check if speed has changed and update ticker if necessary
			// This is more efficient than recreating it every cycle.
			if cpuTicker.Reset(a.cpuSpeed) {
				// This branch is just for clarity; Reset handles the change.
			}

			a.pauseMutex.Lock()
			isRunning := !a.isPaused
			a.pauseMutex.Unlock()

			if isRunning {
				a.cpu.EmulateCycle()
			}

		case <-timerTicker.C:
			a.pauseMutex.Lock()
			isRunning := !a.isPaused
			a.pauseMutex.Unlock()

			if isRunning {
				// Handle timers at a consistent 60Hz
				if a.cpu.DelayTimer > 0 {
					a.cpu.DelayTimer--
				}
				if a.cpu.SoundTimer > 0 {
					// Beep only when timer is active
					a.PlayBeep()
					a.cpu.SoundTimer--
				}
			}

			// Push updates to the UI at a consistent 60Hz
			if a.cpu.DrawFlag {
				// Frontend expects a base64 string for display updates.
				displayData := base64.StdEncoding.EncodeToString(a.cpu.Display[:])
				runtime.EventsEmit(a.ctx, "displayUpdate", displayData)
				a.cpu.ClearDrawFlag()
			}
			runtime.EventsEmit(a.ctx, "debugUpdate", a.cpu.GetState())
		}
	}
}

// --- Go Functions Callable from Frontend ---

func (a *App) GetInitialState() map[string]interface{} {
	a.appendLog("Frontend connected, providing initial state.")
	return a.cpu.GetState()
}

func (a *App) GetDisplay() []byte {
	// This function might not be needed if we push updates, but it's good to have.
	// We return a safe copy.
	displayCopy := make([]byte, len(a.cpu.Display))
	copy(displayCopy, a.cpu.Display[:])
	return displayCopy
}

func (a *App) LoadROM(romName string) error {
	a.appendLog(fmt.Sprintf("Attempting to load ROM: %s", romName))
	romPath := filepath.Join("roms", romName)
	data, err := ioutil.ReadFile(romPath)
	if err != nil {
		errMsg := fmt.Sprintf("Error reading ROM file %s: %v", romName, err)
		a.appendLog(errMsg)
		return fmt.Errorf(errMsg)
	}

	a.cpu.Reset()
	if err := a.cpu.LoadROM(data); err != nil {
		errMsg := fmt.Sprintf("Error loading ROM data %s: %v", romName, err)
		a.appendLog(errMsg)
		return fmt.Errorf(errMsg)
	}

	a.romLoaded = data // Store the ROM data

	a.pauseMutex.Lock()
	a.isPaused = false
	a.cpu.IsRunning = true
	a.pauseMutex.Unlock()

	statusMsg := fmt.Sprintf("Status: Running | ROM: %s", romName)
	runtime.EventsEmit(a.ctx, "statusUpdate", statusMsg)
	a.appendLog(statusMsg)
	return nil
}

func (a *App) Reset() {
	a.pauseMutex.Lock()
	a.isPaused = true
	a.cpu.Reset()
	a.pauseMutex.Unlock()

	statusMsg := "Status: Reset | ROM: None"
	runtime.EventsEmit(a.ctx, "statusUpdate", statusMsg)
	a.appendLog(statusMsg)

	// Force push the cleared state to the UI
	displayData := base64.StdEncoding.EncodeToString(a.cpu.Display[:])
	runtime.EventsEmit(a.ctx, "displayUpdate", displayData)
	runtime.EventsEmit(a.ctx, "debugUpdate", a.cpu.GetState())
}

// SoftReset resets the CPU state and reloads the currently loaded ROM.
func (a *App) SoftReset() error {
	if a.romLoaded == nil {
		return fmt.Errorf("no ROM loaded to soft reset")
	}

	a.pauseMutex.Lock()
	a.isPaused = true
	a.cpu.Reset()
	if err := a.cpu.LoadROM(a.romLoaded); err != nil {
		a.pauseMutex.Unlock()
		return fmt.Errorf("failed to reload ROM during soft reset: %w", err)
	}
	a.cpu.IsRunning = true
	a.isPaused = false
	a.pauseMutex.Unlock()

	statusMsg := "Status: Soft Reset | ROM reloaded."
	a.appendLog(statusMsg)
	runtime.EventsEmit(a.ctx, "statusUpdate", statusMsg)

	// Force push the updated state to the UI
	displayData := base64.StdEncoding.EncodeToString(a.cpu.Display[:])
	runtime.EventsEmit(a.ctx, "displayUpdate", displayData)
	runtime.EventsEmit(a.ctx, "debugUpdate", a.cpu.GetState())

	return nil
}

// HardReset resets the CPU state and clears any loaded ROM.
func (a *App) HardReset() {
	a.pauseMutex.Lock()
	a.isPaused = true
	a.cpu.Reset()
	a.romLoaded = nil // Clear loaded ROM
	a.pauseMutex.Unlock()

	statusMsg := "Status: Hard Reset | ROM cleared."
	a.appendLog(statusMsg)
	runtime.EventsEmit(a.ctx, "statusUpdate", statusMsg)

	// Force push the cleared state to the UI
	displayData := base64.StdEncoding.EncodeToString(a.cpu.Display[:])
	runtime.EventsEmit(a.ctx, "displayUpdate", displayData)
	runtime.EventsEmit(a.ctx, "debugUpdate", a.cpu.GetState())
}

func (a *App) TogglePause() bool {
	a.pauseMutex.Lock()
	a.isPaused = !a.isPaused
	a.cpu.IsRunning = !a.isPaused
	isPausedNow := a.isPaused
	a.pauseMutex.Unlock()

	if isPausedNow {
		a.appendLog("Emulation Paused.")
	} else {
		a.appendLog("Emulation Resumed.")
	}
	return isPausedNow
}

func (a *App) KeyDown(key int) {
	if key >= 0 && key < 16 {
		a.cpu.Keys[key] = true
	}
}

func (a *App) KeyUp(key int) {
	if key >= 0 && key < 16 {
		a.cpu.Keys[key] = false
	}
}

// SetBreakpoint sets a breakpoint at the given address.
func (a *App) SetBreakpoint(address uint16) {
	if a.cpu != nil {
		a.cpu.Breakpoints[address] = true
		a.appendLog(fmt.Sprintf("Breakpoint set at 0x%04X", address))
	}
}

// ClearBreakpoint clears the breakpoint at the given address.
func (a *App) ClearBreakpoint(address uint16) {
	if a.cpu != nil {
		delete(a.cpu.Breakpoints, address)
		a.appendLog(fmt.Sprintf("Breakpoint cleared at 0x%04X", address))
	}
}

// PlayBeep sends a signal to the frontend to play a beep sound.

func (a *App) GetMemory(offset, limit int) string {
	mem := a.cpu.Memory[:]
	if offset < 0 {
		offset = 0
	}
	if offset >= len(mem) {
		return ""
	}
	if limit <= 0 || offset+limit > len(mem) {
		limit = len(mem) - offset
	}
	// Return as base64 as expected by the frontend
	return base64.StdEncoding.EncodeToString(mem[offset : offset+limit])
}

func (a *App) GetROMs() ([]string, error) {
	romsDir := "./roms"
	files, err := ioutil.ReadDir(romsDir)
	if err != nil {
		a.appendLog(fmt.Sprintf("Error reading ROMs directory: %v", err))
		return nil, fmt.Errorf("failed to read ROMs directory: %w", err)
	}

	var romNames []string
	for _, file := range files {
		// Filter for common CHIP-8 extensions
		if !file.IsDir() && (strings.HasSuffix(strings.ToLower(file.Name()), ".ch8") || strings.HasSuffix(strings.ToLower(file.Name()), ".c8")) {
			romNames = append(romNames, file.Name())
		}
	}
	return romNames, nil
}

func (a *App) SetClockSpeed(speed int) {
	if speed > 0 {
		a.cpuSpeed = time.Second / time.Duration(speed)
		runtime.EventsEmit(a.ctx, "clockSpeedUpdate", speed)
		a.appendLog(fmt.Sprintf("Clock speed set to %d Hz", speed))
	}
}

func (a *App) SaveScreenshot(data string) error {
	dec, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return fmt.Errorf("failed to decode base64: %w", err)
	}

	selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Screenshot",
		Filters:         []runtime.FileFilter{{DisplayName: "PNG Image (*.png)", Pattern: "*.png"}},
		DefaultFilename: "chip8_screenshot.png",
	})
	if err != nil || selection == "" {
		return err
	}

	if err := ioutil.WriteFile(selection, dec, 0644); err != nil {
		a.appendLog(fmt.Sprintf("Error saving screenshot: %v", err))
		return fmt.Errorf("failed to write file: %w", err)
	}

	statusMsg := fmt.Sprintf("Screenshot saved to: %s", selection)
	runtime.EventsEmit(a.ctx, "statusUpdate", statusMsg)
	a.appendLog(statusMsg)
	return nil
}

// SaveState returns the current state of the emulator as a gob-encoded byte array.
func (a *App) SaveState() ([]byte, error) {
	a.pauseMutex.Lock()
	a.isPaused = true
	a.cpu.IsRunning = false // Pause emulation before saving
	a.pauseMutex.Unlock()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(a.cpu); err != nil {
		return nil, fmt.Errorf("failed to encode CPU state: %w", err)
	}
	return buf.Bytes(), nil
}

// SaveStateToFile opens a save dialog and writes the provided state to a file.
func (a *App) SaveStateToFile(state []byte) error {
	selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save CHIP-8 State",
		Filters:         []runtime.FileFilter{{DisplayName: "CHIP-8 State (*.ch8state)", Pattern: "*.ch8state"}},
		DefaultFilename: "chip8_state.ch8state",
	})
	if err != nil || selection == "" {
		return err
	}

	if err := ioutil.WriteFile(selection, state, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	statusMsg := fmt.Sprintf("State saved to: %s", selection)
	a.appendLog(statusMsg)
	runtime.EventsEmit(a.ctx, "statusUpdate", statusMsg)
	return nil
}

// LoadState loads a gob-encoded state into the emulator.
func (a *App) LoadState(state []byte) error {
	buf := bytes.NewBuffer(state)
	dec := gob.NewDecoder(buf)
	var loadedCPU chip8.Chip8
	if err := dec.Decode(&loadedCPU); err != nil {
		return fmt.Errorf("failed to decode CPU state: %w", err)
	}

	a.pauseMutex.Lock()
	a.cpu = &loadedCPU
	a.isPaused = true
	a.cpu.IsRunning = false
	a.pauseMutex.Unlock()

	statusMsg := "Emulator state loaded."
	a.appendLog(statusMsg)
	runtime.EventsEmit(a.ctx, "statusUpdate", statusMsg)
	return nil
}

// LoadStateFromFile opens an open dialog, reads a state file, and loads it into the emulator.
func (a *App) LoadStateFromFile() error {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Load CHIP-8 State",
		Filters: []runtime.FileFilter{{DisplayName: "CHIP-8 State (*.ch8state)", Pattern: "*.ch8state"}},
	})
	if err != nil || selection == "" {
		return err
	}

	data, err := ioutil.ReadFile(selection)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	return a.LoadState(data)
}
```

## File: `main.go`

```go
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure.
	// NewApp() now correctly initializes the CPU core.
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "chip8-wails",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 44, G: 62, B: 80, A: 1}, // Matches bg-[#2c3e50]
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

```

