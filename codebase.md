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
│       │   ├── EmulatorView.svelte
│       │   ├── Header.svelte
│       │   ├── LogViewer.svelte
│       │   ├── Notification.svelte
│       │   ├── ROMBrowser.svelte
│       │   ├── SettingsModal.svelte
│       │   ├── clickOutside.js
│       │   └── stores.js
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
	Memory      [4096]byte
	Registers   [16]byte
	I           uint16
	PC          uint16
	Display     [DisplayWidth * DisplayHeight]byte
	DelayTimer  byte
	SoundTimer  byte
	Stack       [16]uint16
	SP          byte
	Keys        [16]bool
	DrawFlag    bool
	IsRunning   bool
	Breakpoints map[uint16]bool // Map to store breakpoint addresses
	randSource  rand.Source
}

// FontSet (keep as is)
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
	if c.Breakpoints == nil {
		c.Breakpoints = make(map[uint16]bool)
	} else {
		for k := range c.Breakpoints {
			delete(c.Breakpoints, k)
		}
	}

	// Load font set into memory
	for i := 0; i < len(FontSet); i++ {
		c.Memory[FontSetStart+i] = FontSet[i]
	}

	c.randSource = rand.NewSource(time.Now().UnixNano())
}

// LoadROM (keep as is)
func (c *Chip8) LoadROM(data []byte) error {
	if len(data) > len(c.Memory)-ProgramStart {
		return fmt.Errorf("ROM size %d exceeds available memory %d", len(data), len(c.Memory)-ProgramStart)
	}
	for i, b := range data {
		c.Memory[ProgramStart+i] = b
	}
	return nil
}

// EmulateCycle (keep as is)
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
	// ... (all opcode cases remain the same)
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

// Disassemble (keep as is, but remove the extra '}' that was causing the error)
func Disassemble(opcode uint16) string {
	vx := (opcode & 0x0F00) >> 8
	vy := (opcode & 0x00F0) >> 4
	nnn := opcode & 0x0FFF
	nn := byte(opcode & 0x00FF)
	n := byte(opcode & 0x000F)

	switch opcode & 0xF000 {
	// ... (all cases remain the same)
	case 0x0000:
		switch opcode & 0x00FF {
		case 0x00E0:
			return fmt.Sprintf("CLS") // Removed opcode prefix for cleaner look
		case 0x00EE:
			return fmt.Sprintf("RET")
		default:
			return fmt.Sprintf("SYS 0x%03X", nnn)
		}
	case 0x1000:
		return fmt.Sprintf("JP 0x%03X", nnn)
	case 0x2000:
		return fmt.Sprintf("CALL 0x%03X", nnn)
	case 0x3000:
		return fmt.Sprintf("SE V%X, 0x%02X", vx, nn)
	case 0x4000:
		return fmt.Sprintf("SNE V%X, 0x%02X", vx, nn)
	case 0x5000:
		return fmt.Sprintf("SE V%X, V%X", vx, vy)
	case 0x6000:
		return fmt.Sprintf("LD V%X, 0x%02X", vx, nn)
	case 0x7000:
		return fmt.Sprintf("ADD V%X, 0x%02X", vx, nn)
	case 0x8000:
		switch n {
		case 0x0:
			return fmt.Sprintf("LD V%X, V%X", vx, vy)
		case 0x1:
			return fmt.Sprintf("OR V%X, V%X", vx, vy)
		case 0x2:
			return fmt.Sprintf("AND V%X, V%X", vx, vy)
		case 0x3:
			return fmt.Sprintf("XOR V%X, V%X", vx, vy)
		case 0x4:
			return fmt.Sprintf("ADD V%X, V%X", vx, vy)
		case 0x5:
			return fmt.Sprintf("SUB V%X, V%X", vx, vy)
		case 0x6:
			return fmt.Sprintf("SHR V%X", vx)
		case 0x7:
			return fmt.Sprintf("SUBN V%X, V%X", vx, vy)
		case 0xE:
			return fmt.Sprintf("SHL V%X", vx)
		default:
			return fmt.Sprintf("UNKNOWN 8xx%X", n)
		}
	case 0x9000:
		return fmt.Sprintf("SNE V%X, V%X", vx, vy)
	case 0xA000:
		return fmt.Sprintf("LD I, 0x%03X", nnn)
	case 0xB000:
		return fmt.Sprintf("JP V0, 0x%03X", nnn)
	case 0xC000:
		return fmt.Sprintf("RND V%X, 0x%02X", vx, nn)
	case 0xD000:
		return fmt.Sprintf("DRW V%X, V%X, %d", vx, vy, n)
	case 0xE000:
		switch nn {
		case 0x9E:
			return fmt.Sprintf("SKP V%X", vx)
		case 0xA1:
			return fmt.Sprintf("SKNP V%X", vx)
		default:
			return fmt.Sprintf("UNKNOWN Ex%02X", nn)
		}
	case 0xF000:
		switch nn {
		case 0x07:
			return fmt.Sprintf("LD V%X, DT", vx)
		case 0x0A:
			return fmt.Sprintf("LD V%X, K", vx)
		case 0x15:
			return fmt.Sprintf("LD DT, V%X", vx)
		case 0x18:
			return fmt.Sprintf("LD ST, V%X", vx)
		case 0x1E:
			return fmt.Sprintf("ADD I, V%X", vx)
		case 0x29:
			return fmt.Sprintf("LD F, V%X", vx)
		case 0x33:
			return fmt.Sprintf("LD B, V%X", vx)
		case 0x55:
			return fmt.Sprintf("LD [I], V%X", vx)
		case 0x65:
			return fmt.Sprintf("LD V%X, [I]", vx)
		default:
			return fmt.Sprintf("UNKNOWN Fx%02X", nn)
		}
	default:
		return fmt.Sprintf("UNKNOWN %04X", opcode)
	}
	// NO extra brace here
}

// GetState returns a snapshot of the CPU state for debugging.
func (c *Chip8) GetState() map[string]interface{} {
	disassembly := []string{}
	// Disassemble instructions around the Program Counter for context
	// Let's show a bit more context, maybe 20 lines
	for i := -10; i < 10; i++ {
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

	// Create copies of arrays to avoid data races
	registersCopy := make([]byte, len(c.Registers))
	copy(registersCopy, c.Registers[:])
	stackCopy := make([]uint16, len(c.Stack))
	copy(stackCopy, c.Stack[:])
	// *** FIX: Also create a copy of the breakpoints map ***
	breakpointsCopy := make(map[uint16]bool)
	for k, v := range c.Breakpoints {
		breakpointsCopy[k] = v
	}

	return map[string]interface{}{
		"PC":          c.PC,
		"I":           c.I,
		"SP":          c.SP,
		"DelayTimer":  c.DelayTimer,
		"SoundTimer":  c.SoundTimer,
		"Registers":   registersCopy,
		"Stack":       stackCopy,
		"Disassembly": disassembly,
		"Breakpoints": breakpointsCopy, // *** FIX: Add breakpoints to state ***
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
        try {
            const data = await GetMemory(memoryOffset, memoryLimit);
            if (data) {
                memoryData = new Uint8Array(atob(data).split('').map(char => char.charCodeAt(0)));
            }
        } catch (error) {
            console.error("Failed to fetch memory view:", error);
        }
    }

    onMount(() => {
        memoryUpdateInterval = setInterval(fetchMemoryView, 200);
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
        if (event.deltaY > 0) {
            memoryOffset += 16;
        } else {
            memoryOffset -= 16;
        }

        if (memoryOffset < 0) memoryOffset = 0;
        if (memoryOffset > 4096 - memoryLimit) memoryOffset = 4096 - memoryLimit;

        event.preventDefault();
        fetchMemoryView();
    }

    async function toggleBreakpoint(address) {
        if (debugState.Breakpoints && debugState.Breakpoints[address]) {
            await ClearBreakpoint(address);
        } else {
            await SetBreakpoint(address);
        }
    }
</script>

<div class="grid grid-cols-1 lg:grid-cols-3 gap-3 p-3 h-full overflow-y-auto bg-gray-900 text-gray-300 font-sans">
    
    <!-- Left Column -->
    <div class="lg:col-span-1 flex flex-col space-y-3">
        <!-- CPU State -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700">
            <h3 class="font-semibold text-md mb-2 text-gray-400">CPU State</h3>
            <div class="grid grid-cols-2 gap-x-4 text-sm font-mono">
                <p>PC: <span class="text-cyan-400">{formatAddress(debugState.PC ?? 0)}</span></p>
                <p>I: <span class="text-cyan-400">{formatAddress(debugState.I ?? 0)}</span></p>
                <p>SP: <span class="text-cyan-400">{formatAddress(debugState.SP ?? 0)}</span></p>
            </div>
        </div>

        <!-- Timers -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700">
            <h3 class="font-semibold text-md mb-2 text-gray-400">Timers</h3>
            <div class="grid grid-cols-2 gap-x-4 text-sm font-mono">
                <p>Delay: <span class="text-green-400">{debugState.DelayTimer ?? 0}</span></p>
                <p>Sound: <span class="text-green-400">{debugState.SoundTimer ?? 0}</span></p>
            </div>
        </div>

        <!-- Registers -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700">
            <h3 class="font-semibold text-md mb-2 text-gray-400">Registers</h3>
            <div class="grid grid-cols-4 gap-x-2 gap-y-1 text-sm font-mono">
                {#each { length: 16 } as _, i}
                    <span>V{i.toString(16).toUpperCase()}: <span class="text-yellow-400">{`0x${debugState.Registers?.[i]?.toString(16).padStart(2, "0").toUpperCase() ?? "00"}`}</span></span>
                {/each}
            </div>
        </div>

        <!-- Stack -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700">
            <h3 class="font-semibold text-md mb-2 text-gray-400">Stack</h3>
            <pre class="text-xs overflow-y-auto h-28 bg-gray-900 p-2 rounded-md border border-gray-700 font-mono">
                {#each debugState.Stack || [] as value, i}
                    <div class:text-cyan-300={i === (debugState.SP > 0 ? debugState.SP -1 : 0)} class:font-bold={i === (debugState.SP > 0 ? debugState.SP -1 : 0)}>Stack[{i.toString(16).toUpperCase()}]: {formatAddress(value)}</div>
                {/each}
            </pre>
        </div>
    </div>

    <!-- Middle Column -->
    <div class="lg:col-span-1 flex flex-col space-y-3">
        <!-- Disassembly -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700 flex-grow flex flex-col">
            <h3 class="font-semibold text-md mb-2 text-gray-400">Disassembly</h3>
            <pre class="text-xs leading-snug overflow-y-auto bg-gray-900 p-2 rounded-md border border-gray-700 flex-grow font-mono">
                {#each debugState.Disassembly || [] as line}
                    {@const address = parseInt(line.split(":")[0].replace("► ", ""), 16)}
                    <div
                        class="cursor-pointer hover:bg-gray-700 px-1 rounded-sm"
                        class:text-cyan-300={line.startsWith("►")}
                        class:font-bold={line.startsWith("►")}
                        class:bg-red-800={debugState.Breakpoints && debugState.Breakpoints[address]}
                        class:hover:bg-red-700={debugState.Breakpoints && debugState.Breakpoints[address]}
                        on:click={() => toggleBreakpoint(address)}
                        title="Click to toggle breakpoint"
                    >{line}</div>
                {/each}
            </pre>
        </div>
    </div>

    <!-- Right Column -->
    <div class="lg:col-span-1 flex flex-col space-y-3">
        <!-- Memory Viewer -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700 flex-grow flex flex-col">
            <h3 class="font-semibold text-md mb-2 text-gray-400">Memory Viewer</h3>
            <div class="text-xs overflow-y-auto bg-gray-900 p-2 rounded-md border border-gray-700 flex-grow font-mono" on:wheel={handleMemoryScroll}>
                {#each Array(Math.ceil(memoryData.length / 16)) as _, rowIdx}
                    <div class="flex whitespace-pre">
                        <span class="text-gray-500 mr-2">{formatAddress(memoryOffset + rowIdx * 16)}:</span>
                        <div class="flex-grow grid grid-cols-16">
                            {#each Array(16) as _, colIdx}
                                {@const byte = memoryData[rowIdx * 16 + colIdx]}
                                <span class="mr-1">{byte !== undefined ? formatByte(byte) : "--"}</span>
                            {/each}
                        </div>
                    </div>
                {/each}
            </div>
        </div>
    </div>

    <!-- Logs (Full Width) -->
    <div class="lg:col-span-3 bg-gray-800 p-3 rounded-md border border-gray-700">
        <h3 class="font-semibold text-md mb-2 text-gray-400">Application Logs</h3>
        <LogViewer />
    </div>
</div>
```

## File: `frontend/src/lib/EmulatorView.svelte`

```svelte
<script>
    import {
        Camera,
        Pause,
        Play,
        RotateCcw,
        Save,
        Upload,
    } from "lucide-svelte";
    import { createEventDispatcher, onDestroy, onMount } from "svelte";
    import { settings, showNotification } from "./stores.js";
    import Gamepad from "svelte-gamepad";
    import {
        HardReset,
        KeyDown,
        KeyUp,
        LoadROM,
        LoadStateFromFile,
        SaveScreenshot,
        SaveState,
        SaveStateToFile,
        SoftReset,
        TogglePause,
    } from "../wailsjs/go/main/App.js";
    import { EventsOn } from "../wailsjs/runtime/runtime.js";
    import { clickOutside } from "./clickOutside.js";
    import ROMBrowser from "./ROMBrowser.svelte";

    const dispatch = createEventDispatcher();


    // --- State from the store ---
    $: keyMap = $settings.keyMap;
    $: currentDisplayColor = $settings.displayColor;
    $: currentScanlineEffect = $settings.scanlineEffect;

    let canvasElement;
    let isPaused = true;
    let currentDisplayBuffer = new Uint8Array(64 * 32);
    let showResetOptions = false;

    /** @type {{hex: number, key: string, keyboardKey: string}[]} */
    const keypadLayout = [
        { hex: 0x1, key: "1", keyboardKey: "1" },
        { hex: 0x2, key: "2", keyboardKey: "2" },
        { hex: 0x3, key: "3", keyboardKey: "3" },
        { hex: 0xc, key: "C", keyboardKey: "4" },
        { hex: 0x4, key: "4", keyboardKey: "Q" },
        { hex: 0x5, key: "5", keyboardKey: "W" },
        { hex: 0x6, key: "6", keyboardKey: "E" },
        { hex: 0xd, key: "D", keyboardKey: "R" },
        { hex: 0x7, key: "7", keyboardKey: "A" },
        { hex: 0x8, key: "8", keyboardKey: "S" },
        { hex: 0x9, key: "9", keyboardKey: "D" },
        { hex: 0xe, key: "E", keyboardKey: "F" },
        { hex: 0xa, key: "A", keyboardKey: "Z" },
        { hex: 0x0, key: "0", keyboardKey: "X" },
        { hex: 0xb, key: "B", keyboardKey: "C" },
        { hex: 0xf, key: "F", keyboardKey: "V" },
    ];

    /** @type {Record<string, number>} */
    const gamepadMap = {
        A: 0x5,
        B: 0x6,
        X: 0x8,
        Y: 0x9,
        DpadUp: 0x2,
        DpadDown: 0x8,
        DpadLeft: 0x7,
        DpadRight: 0x9,
    };

    let pressedKeys = {};

    const SCALE = 10;
    const DISPLAY_WIDTH = 64;
    const DISPLAY_HEIGHT = 32;

    let audioContext;
    let oscillator;
    let animationFrameId;

    /** Play a short beep using Web Audio API. */
    function playBeep() {
        if (!audioContext) {
            audioContext = new (window.AudioContext ||
                window.webkitAudioContext)();
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

    /**
     * Draw the CHIP-8 display buffer to the canvas.
     * @param {HTMLCanvasElement} canvas
     * @param {Uint8Array} displayBuffer
     */
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
        EventsOn("wails:file-drop", handleFileDrop);
        EventsOn("menu:pause", handleTogglePause);

        EventsOn("displayUpdate", (base64DisplayBuffer) => {
            if (animationFrameId) cancelAnimationFrame(animationFrameId);
            animationFrameId = requestAnimationFrame(() => {
                const binaryString = atob(base64DisplayBuffer);
                const bytes = new Uint8Array(binaryString.length);
                for (let i = 0; i < binaryString.length; i++) {
                    bytes[i] = binaryString.charCodeAt(i);
                }
                currentDisplayBuffer = bytes;
                drawDisplay(canvasElement, currentDisplayBuffer);
            });
        });

        EventsOn("playBeep", playBeep);

        drawDisplay(
            canvasElement,
            new Uint8Array(DISPLAY_WIDTH * DISPLAY_HEIGHT),
        );
    });

    $: if (canvasElement) {
        drawDisplay(canvasElement, currentDisplayBuffer);
    }

    let reverseKeyMap = {};
    $: {
        reverseKeyMap = {};
        for (const [keyboardKey, chip8Key] of Object.entries($settings.keyMap)) {
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

    /** Toggle emulator pause state. */
    async function handleTogglePause() {
        isPaused = await TogglePause();
    }

    /** Save a screenshot of the current canvas. */
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

    /** Save emulator state to file. */
    async function handleSaveState() {
        try {
            const state = await SaveState();
            await SaveStateToFile(state);
            showNotification("Emulator state saved!", "success");
        } catch (error) {
            showNotification(`Failed to save state: ${error}`, "error");
        }
    }

    /** Load emulator state from file. */
    async function handleLoadState() {
        try {
            await LoadStateFromFile();
            showNotification("Emulator state loaded!", "success");
        } catch (error) {
            showNotification(`Failed to load state: ${error}`, "error");
        }
    }

    function toggleResetOptions() {
        showResetOptions = !showResetOptions;
    }

    /** Perform a soft reset (reload ROM). */
    async function handleSoftReset() {
        try {
            await SoftReset();
            isPaused = false;
            showNotification("Soft reset complete! ROM reloaded.", "success");
        } catch (error) {
            showNotification(`Soft reset failed: ${error}`, "error");
        }
        showResetOptions = false;
    }

    /** Perform a hard reset (clear ROM). */
    async function handleHardReset() {
        try {
            await HardReset();
            isPaused = true;
            showNotification("Hard reset complete! ROM cleared.", "info");
        } catch (error) {
            showNotification(`Hard reset failed: ${error}`, "error");
        }
        showResetOptions = false;
    }

    /**
     * Handle file drop event for loading ROMs.
     * @param {any} event
     */
    async function handleFileDrop(event) {
        if (event.data.length > 0) {
            const romName = event.data[0].split("/").pop();
            try {
                await LoadROM(romName);
                showNotification(`Successfully loaded ${romName}`, "success");
            } catch (error) {
                showNotification(`Failed to load ROM: ${error}`, "error");
            }
        }
    }

    /** @param {CustomEvent} e */
    function onGamepadConnected(e) {
        showNotification(
            `Gamepad ${e.detail.gamepadIndex + 1} connected.`,
            "success",
        );
    }

    /** @param {CustomEvent} e */
    function onGamepadDisconnected(e) {
        showNotification(
            `Gamepad ${e.detail.gamepadIndex + 1} disconnected.`,
            "warning",
        );
    }

    /** @param {CustomEvent} e */
    function handleGamepadButton(e) {
        const chip8Key = gamepadMap[e.type];
        if (chip8Key !== undefined) {
            if (e.detail.pressed) {
                handleKeypadPress(chip8Key);
            } else {
                handleKeypadRelease(chip8Key);
            }
        }
    }

    /**
     * Handle keypad press for CHIP-8 key.
     * @param {number} key
     */
    function handleKeypadPress(key) {
        KeyDown(key);
        pressedKeys = { ...pressedKeys, [key]: true };
    }

    /**
     * Handle keypad release for CHIP-8 key.
     * @param {number} key
     */
    function handleKeypadRelease(key) {
        KeyUp(key);
        pressedKeys = { ...pressedKeys, [key]: false };
    }
</script>

<Gamepad
    gamepadIndex={0}
    on:Connected={onGamepadConnected}
    on:Disconnected={onGamepadDisconnected}
    on:A={handleGamepadButton}
    on:B={handleGamepadButton}
    on:X={handleGamepadButton}
    on:Y={handleGamepadButton}
    on:DpadUp={handleGamepadButton}
    on:DpadDown={handleGamepadButton}
    on:DpadLeft={handleGamepadButton}
    on:DpadRight={handleGamepadButton}
/>

<div
    class="flex flex-col md:flex-row h-full p-3 space-y-3 md:space-y-0 md:space-x-3"
>
    <section
        class="flex-grow flex items-center justify-center bg-gray-900 rounded-md shadow-inner p-3"
    >
        <canvas
            bind:this={canvasElement}
            width={DISPLAY_WIDTH * SCALE}
            height={DISPLAY_HEIGHT * SCALE}
            class="border border-gray-700 rounded-sm"
        ></canvas>
    </section>
    <aside class="flex-none w-full md:w-72 flex flex-col space-y-3">
        <ROMBrowser />
        <div class="bg-gray-900 p-3 rounded-md shadow-inner">
            <h2 class="text-lg font-semibold mb-2 text-center text-gray-400">
                Controls
            </h2>
            <div class="grid grid-cols-2 gap-2">
                <div
                    class="relative inline-block text-left"
                    use:clickOutside={() => (showResetOptions = false)}
                >
                    <button
                        on:click={toggleResetOptions}
                        class="flex items-center justify-center space-x-2 bg-yellow-600 hover:bg-yellow-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 w-full text-sm"
                        title="Reset Options"
                    >
                        <RotateCcw size={16} />
                        <span>Reset</span>
                    </button>
                    {#if showResetOptions}
                        <div
                            class="origin-top-right absolute right-0 mt-1 w-full rounded-md shadow-lg bg-gray-700 ring-1 ring-black ring-opacity-5 focus:outline-none z-10"
                        >
                            <div class="py-1">
                                <button
                                    on:click={handleSoftReset}
                                    class="block w-full text-left px-3 py-1 text-sm text-gray-200 hover:bg-gray-600"
                                    >Soft Reset</button
                                >
                                <button
                                    on:click={handleHardReset}
                                    class="block w-full text-left px-3 py-1 text-sm text-gray-200 hover:bg-gray-600"
                                    >Hard Reset</button
                                >
                            </div>
                        </div>
                    {/if}
                </div>
                <button
                    on:click={handleTogglePause}
                    class="flex items-center justify-center space-x-2 bg-green-600 hover:bg-green-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm"
                    title={isPaused
                        ? "Resume emulation (Ctrl+P)"
                        : "Pause emulation (Ctrl+P)"}
                >
                    {#if isPaused}<Play size={16} /><span>Resume</span
                        >{:else}<Pause size={16} /><span>Pause</span>{/if}
                </button>
                <button
                    on:click={handleScreenshot}
                    class="flex items-center justify-center space-x-2 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 col-span-2 text-sm"
                    title="Take a screenshot"
                    ><Camera size={16} /><span>Screenshot</span></button
                >
                <button
                    on:click={handleSaveState}
                    class="flex items-center justify-center space-x-2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm"
                    title="Save State"
                    ><Save size={16} /><span>Save State</span></button
                >
                <button
                    on:click={handleLoadState}
                    class="flex items-center justify-center space-x-2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm"
                    title="Load State (Ctrl+O)"
                    ><Upload size={16} /><span>Load State</span></button
                >
            </div>
        </div>
        <div class="bg-gray-900 p-3 rounded-md shadow-inner">
            <h2 class="text-lg font-semibold mb-2 text-center text-gray-400">
                CHIP-8 Keypad
            </h2>
            <div class="grid grid-cols-4 gap-2 text-center font-mono">
                {#each keypadLayout as { hex, key, keyboardKey } (hex)}
                    <button
                        on:mousedown={() => handleKeypadPress(hex)}
                        on:mouseup={() => handleKeypadRelease(hex)}
                        on:mouseleave={() => handleKeypadRelease(hex)}
                        class="p-2 rounded-md border text-lg font-bold flex flex-col items-center justify-center aspect-square transition-all duration-100 focus:outline-none"
                        class:bg-blue-500={pressedKeys[hex]}
                        class:border-blue-400={pressedKeys[hex]}
                        class:text-white={pressedKeys[hex]}
                        class:bg-gray-700={!pressedKeys[hex]}
                        class:border-gray-600={!pressedKeys[hex]}
                        class:hover:bg-gray-600={!pressedKeys[hex]}
                        title={`CHIP-8 Key: ${hex.toString(16).toUpperCase()}`}
                    >
                        <span class="text-xl">{key}</span>
                        <span class="text-xs text-gray-400 mt-1"
                            >{keyboardKey}</span
                        >
                    </button>
                {/each}
            </div>
        </div>
    </aside>
</div>

```

## File: `frontend/src/lib/Header.svelte`

```svelte
<script>
    import {
        WindowMinimise,
        WindowMaximise,
        WindowUnmaximise,
        WindowIsMaximised,
        Quit,
    } from "../wailsjs/runtime/runtime.js";
    import { Settings, Minimize, Maximize, X, Copy } from "lucide-svelte";
    import appIcon from "../assets/appicon.svg";
    import { createEventDispatcher, onMount } from "svelte";

    const dispatch = createEventDispatcher();

    export let currentTab;

    let isMaximized = false;

    onMount(async () => {
        isMaximized = await WindowIsMaximised();
    });

    async function toggleMaximize() {
        if (await WindowIsMaximised()) {
            WindowUnmaximise();
        } else {
            WindowMaximise();
        }
        isMaximized = !isMaximized;
    }

    function openSettings() {
        dispatch("openSettings");
    }
</script>

<header
    style="--wails-draggable:drag"
    class="flex-none bg-gray-900 text-gray-200 shadow-md z-20 flex items-center justify-between pr-2"
>
    <div class="flex items-center">
        <!-- App Icon and Title -->
        <div class="p-2 flex items-center space-x-2">
            <img src={appIcon} alt="App Icon" class="h-5 w-5" />
            <h1 class="text-md font-semibold text-gray-300">CHIP-8 Emulator</h1>
        </div>
        <!-- Tabs -->
        <nav class="flex space-x-1">
            <button
                on:click={() => (currentTab = "emulator")}
                class="px-3 py-1 rounded-md text-sm font-medium transition-colors duration-200"
                class:bg-gray-700={currentTab === "emulator"}
                class:text-white={currentTab === "emulator"}
                class:text-gray-400={currentTab !== "emulator"}
                class:hover:bg-gray-700={currentTab !== "emulator"}
                class:hover:text-white={currentTab !== "emulator"}
                >Emulator</button
            >
            <button
                on:click={() => (currentTab = "debug")}
                class="px-3 py-1 rounded-md text-sm font-medium transition-colors duration-200"
                class:bg-gray-700={currentTab === "debug"}
                class:text-white={currentTab === "debug"}
                class:text-gray-400={currentTab !== "debug"}
                class:hover:bg-gray-700={currentTab !== "debug"}
                class:hover:text-white={currentTab !== "debug"}
                >Debug</button
            >
        </nav>
    </div>

    <!-- Window Controls -->
    <div class="flex items-center space-x-1">
        <button
            on:click={openSettings}
            class="p-2 rounded-md hover:bg-gray-700 transition-colors duration-200"
            title="Settings"
        >
            <Settings size={16} />
        </button>
        <button
            on:click={WindowMinimise}
            class="p-2 rounded-md hover:bg-gray-700 transition-colors duration-200"
            title="Minimize"
        >
            <Minimize size={16} />
        </button>
        <button
            on:click={toggleMaximize}
            class="p-2 rounded-md hover:bg-gray-700 transition-colors duration-200"
            title={isMaximized ? "Restore" : "Maximize"}
        >
            {#if isMaximized}
                <Copy size={16} />
            {:else}
                <Maximize size={16} />
            {/if}
        </button>
        <button
            on:click={Quit}
            class="p-2 rounded-md hover:bg-red-600 transition-colors duration-200"
            title="Quit"
        >
            <X size={16} />
        </button>
    </div>
</header>

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
    import { fade } from "svelte/transition";
    import { notification } from "./stores.js";

    let timeout;

    function dismiss() {
        clearTimeout(timeout);
        notification.set({ ...$notification, show: false });
    }

    let bgColorClass;
    $: {
        if ($notification.show && $notification.message) {
            clearTimeout(timeout);
            timeout = setTimeout(dismiss, 3000);
        }
        switch ($notification.type) {
            case "success":
                bgColorClass = "bg-green-500";
                break;
            case "warning":
                bgColorClass = "bg-yellow-500";
                break;
            case "error":
                bgColorClass = "bg-red-500";
                break;
            case "info":
            default:
                bgColorClass = "bg-blue-500";
        }
    }
</script>

{#if $notification.show && $notification.message}
    <div
        in:fade={{ duration: 150 }}
        out:fade={{ duration: 150 }}
        class="fixed bottom-4 right-4 p-4 rounded-lg shadow-lg text-white flex items-center space-x-3 z-50 {bgColorClass}"
        role="alert"
    >
        <span>{$notification.message}</span>
        <button
            on:click={dismiss}
            class="ml-auto text-white opacity-75 hover:opacity-100"
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5"
                viewBox="0 0 20 20"
                fill="currentColor"
            >
                <path
                    fill-rule="evenodd"
                    d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                    clip-rule="evenodd"
                />
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
    import { showNotification } from './stores.js';
    import { Play } from 'lucide-svelte';

    let roms = [];
    let selectedROM = '';

    async function fetchROMs() {
        try {
            const result = await GetROMs();
            roms = result || [];

            if (roms.length === 0) {
                showNotification("No ROMs found in ./roms directory.", "warning");
            }
        } catch (error) {
            showNotification(`Failed to load ROM list: ${error}`, "error");
            console.error("Error fetching ROMs:", error);
            roms = [];
        }
    }

    async function handleLoadSelectedROM() {
        if (selectedROM) {
            try {
                await LoadROM(selectedROM);
                showNotification(`ROM loaded: ${selectedROM}`, "success");
            } catch (error) {
                showNotification(`Failed to load ROM: ${error}`, "error");
            }
        } else {
            showNotification("Please select a ROM first.", "warning");
        }
    }

    onMount(fetchROMs);
</script>

<style>
    select {
        -webkit-appearance: none;
        -moz-appearance: none;
        appearance: none;
        background-image: url('data:image/svg+xml;charset=US-ASCII,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22292.4%22%20height%3D%22292.4%22%3E%3Cpath%20fill%3D%22%23CBD5E0%22%20d%3D%22M287%2069.4a17.6%2017.6%200%200%200-13-5.4H18.4c-5%200-9.3%201.8-12.9%205.4A17.6%2017.6%200%200%200%200%2082.2c0%205%201.8%209.3%205.4%2012.9l128%20127.9c3.6%203.6%207.8%205.4%2012.8%205.4s9.2-1.8%2012.8-5.4L287%2095c3.5-3.5%205.4-7.8%205.4-12.8%200-5-1.9-9.2-5.5-12.8z%22%2F%3E%3C%2Fsvg%3E');
        background-repeat: no-repeat;
        background-position: right 0.7rem center;
        background-size: 0.65em auto;
        padding-right: 2.5rem;
    }
</style>

<div class="bg-gray-900 p-3 rounded-md shadow-inner">
    <h3 class="text-lg font-semibold mb-2 text-center text-gray-400">ROM Browser</h3>
    <div class="mb-2">
        <select bind:value={selectedROM} class="w-full p-2 rounded-md bg-gray-700 border border-gray-600 text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500">
            <option value="">-- Select a ROM --</option>
            {#each roms as rom}
                <option value={rom}>{rom}</option>
            {/each}
        </select>
    </div>
    <button
        on:click={handleLoadSelectedROM}
        class="w-full flex items-center justify-center space-x-2 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm"
        title="Load Selected ROM"
    >
        <Play size={16} />
        <span>Load ROM</span>
    </button>
</div>
```

## File: `frontend/src/lib/SettingsModal.svelte`

```svelte
<script>
    import { settings, updateAndSaveSettings } from "./stores.js";

    export let showModal;

    // --- Tab State ---
    let activeTab = "appearance"; // 'appearance', 'emulation', 'keybindings'



    let remappingKey = null; // Stores the CHIP-8 key (a number, 0-15) being remapped

    // --- Derived State for Keybinding View ---
    let keybindings = [];
    $: {
        // --- Local State for Settings ---
    if (showModal) {
        localSettings = { ...$settings, keyMap: { ...$settings.keyMap } };
    }
        // Create a display-friendly array
        keybindings = Array.from({ length: 16 }, (_, i) => {
            const chip8Hex = i;
            let keyboardKey = "N/A";
            for (const k in localSettings.keyMap) {
                if (localSettings.keyMap[k] === chip8Hex) {
                    keyboardKey = k;
                    break;
                }
            }
            return { chip8Key: chip8Hex, keyboardKey: keyboardKey };
        });
    }

    function closeModal() {
        showModal = false;
    }

    async function saveSettings() {
        await updateAndSaveSettings(localSettings);
        closeModal();
    }

    // --- Key Remapping Logic ---
    function startRemap(event, chip8KeyToRemap) {
        remappingKey = [REDACTED_generic-api-key];
        event.target.value = "Press key...";
        window.addEventListener("keydown", handleRemapKeyDown, { once: true });
    }

    function handleRemapKeyDown(event) {
        event.preventDefault();
        if (remappingKey === null) return;

        const newKeyboardKey = event.key.toLowerCase();

        // Find the keyboard key that is currently mapped to the chip8 key we are remapping
        let oldKeyboardKey = Object.keys(localSettings.keyMap).find(
            (k) => localSettings.keyMap[k] === remappingKey,
        );

        // Find which chip8 key (if any) is currently using the new keyboard key
        const conflictingChip8Key = localSettings.keyMap[newKeyboardKey];

        // Create a new map to avoid weird reactivity issues
        const updatedKeyMap = { ...localSettings.keyMap };

        // 1. Remove the old mapping for the key we are changing
        if (oldKeyboardKey) {
            delete updatedKeyMap[oldKeyboardKey];
        }

        // 2. If the new key was already in use by another chip8 key, we need to handle it.
        //    Let's swap them: the conflicting chip8 key will now be mapped to the old keyboard key.
        if (conflictingChip8Key !== undefined && oldKeyboardKey) {
            updatedKeyMap[oldKeyboardKey] = conflictingChip8Key;
        } else if (conflictingChip8Key !== undefined) {
            // If there was no old key to swap to, the conflicting mapping is simply removed.
            delete updatedKeyMap[newKeyboardKey];
        }

        // 3. Set the new mapping
        updatedKeyMap[newKeyboardKey] = remappingKey;

        // 4. Update the local state (replace object for reactivity)
        localSettings = {
            ...localSettings,
            keyMap: updatedKeyMap,
        };
        remappingKey = null;
    }

    function endRemap(event, chip8Key) {
        if (remappingKey !== null) {
            // Find the original keyboard key to revert to
            let originalKeyboardKey = "N/A";
            for (const k in $settings.keyMap) {
                if ($settings.keyMap[k] === chip8Key) {
                    originalKeyboardKey = k;
                    break;
                }
            }
            event.target.value = originalKeyboardKey.toUpperCase();
            remappingKey = null;
        }
    }
</script>

{#if showModal}
    <div
        class="fixed inset-0 bg-black bg-opacity-70 flex items-center justify-center z-50 transition-opacity"
    >
        <div
            class="bg-gray-800 p-5 rounded-lg shadow-2xl border border-gray-700 w-full max-w-2xl"
        >
            <h2 class="text-xl font-semibold mb-4 text-center text-gray-200">
                Settings
            </h2>
            <div class="flex space-x-1">
                <!-- Sidebar -->
                <div class="w-1/4 bg-gray-900 p-3 rounded-l-md">
                    <nav class="space-y-1">
                        <button
                            on:click={() => (activeTab = "appearance")}
                            class="w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors duration-150"
                            class:bg-gray-700={activeTab === "appearance"}
                            class:text-white={activeTab === "appearance"}
                            class:text-gray-400={activeTab !== "appearance"}
                            class:hover:bg-gray-700={activeTab !== "appearance"}
                            class:hover:text-white={activeTab !== "appearance"}
                            >Appearance</button
                        >
                        <button
                            on:click={() => (activeTab = "emulation")}
                            class="w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors duration-150"
                            class:bg-gray-700={activeTab === "emulation"}
                            class:text-white={activeTab === "emulation"}
                            class:text-gray-400={activeTab !== "emulation"}
                            class:hover:bg-gray-700={activeTab !== "emulation"}
                            class:hover:text-white={activeTab !== "emulation"}
                            >Emulation</button
                        >
                        <button
                            on:click={() => (activeTab = "keybindings")}
                            class="w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors duration-150"
                            class:bg-gray-700={activeTab === "keybindings"}
                            class:text-white={activeTab === "keybindings"}
                            class:text-gray-400={activeTab !== "keybindings"}
                            class:hover:bg-gray-700={activeTab !==
                                "keybindings"}
                            class:hover:text-white={activeTab !== "keybindings"}
                            >Keybindings</button
                        >
                    </nav>
                </div>
                <!-- Content -->
                <div class="w-3/4 bg-gray-800 p-4 rounded-r-md">
                    <div class="min-h-[300px]">
                        {#if activeTab === "appearance"}
                            <div class="space-y-5">
                                <h3 class="text-lg font-semibold text-gray-300">
                                    Display
                                </h3>
                                <div>
                                    <label
                                        class="block text-gray-400 text-sm font-medium mb-2"
                                        >Pixel Color</label
                                    >
                                    <div class="flex flex-wrap gap-4">
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-green-500 focus:ring-green-500"
                                                name="displayColor"
                                                value="#33FF00"
                                                bind:group={
                                                    localSettings.displayColor
                                                }
                                            /><span class="ml-2"
                                                >Classic Green</span
                                            ></label
                                        >
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500"
                                                name="displayColor"
                                                value="#FFFFFF"
                                                bind:group={
                                                    localSettings.displayColor
                                                }
                                            /><span class="ml-2">White</span
                                            ></label
                                        >
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-yellow-500 focus:ring-yellow-500"
                                                name="displayColor"
                                                value="#FFBF00"
                                                bind:group={
                                                    localSettings.displayColor
                                                }
                                            /><span class="ml-2">Amber</span
                                            ></label
                                        >
                                    </div>
                                </div>
                                <div>
                                    <label class="inline-flex items-center"
                                        ><input
                                            type="checkbox"
                                            class="form-checkbox bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500"
                                            bind:checked={
                                                localSettings.scanlineEffect
                                            }
                                        /><span class="ml-2 text-gray-300"
                                            >Enable Scanline Effect</span
                                        ></label
                                    >
                                </div>
                            </div>
                        {/if}
                        {#if activeTab === "emulation"}
                            <div class="space-y-6">
                                <h3 class="text-lg font-semibold text-gray-300">
                                    Performance
                                </h3>
                                <div>
                                    <label
                                        for="clockSpeed"
                                        class="block text-gray-400 text-sm font-medium mb-2"
                                        >CPU Clock Speed: {localSettings.clockSpeed}
                                        Hz</label
                                    >
                                    <input
                                        type="range"
                                        id="clockSpeed"
                                        min="100"
                                        max="2000"
                                        step="50"
                                        bind:value={localSettings.clockSpeed}
                                        class="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer"
                                    />
                                </div>
                                <div>
                                    <label
                                        class="block text-gray-400 text-sm font-medium mb-2"
                                        >Speed Presets</label
                                    >
                                    <div class="flex flex-wrap gap-4">
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500"
                                                value={700}
                                                bind:group={
                                                    localSettings.clockSpeed
                                                }
                                            /><span class="ml-2"
                                                >Original (700Hz)</span
                                            ></label
                                        >
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500"
                                                value={1400}
                                                bind:group={
                                                    localSettings.clockSpeed
                                                }
                                            /><span class="ml-2"
                                                >Fast (1400Hz)</span
                                            ></label
                                        >
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500"
                                                value={2000}
                                                bind:group={
                                                    localSettings.clockSpeed
                                                }
                                            /><span class="ml-2"
                                                >Turbo (2000Hz)</span
                                            ></label
                                        >
                                    </div>
                                </div>
                            </div>
                        {/if}
                        {#if activeTab === "keybindings"}
                            <div>
                                <h3 class="text-lg font-semibold text-gray-300">
                                    Key Remapping
                                </h3>
                                <p class="text-sm text-gray-400 mb-3">
                                    Click a key, then press the desired keyboard
                                    key to rebind.
                                </p>
                                <div
                                    class="grid grid-cols-4 gap-3 text-center font-mono"
                                >
                                    {#each keybindings as binding (binding.chip8Key)}
                                        <div
                                            class="bg-gray-700 p-2 rounded-md border border-gray-600"
                                        >
                                            <span
                                                class="font-bold text-gray-300"
                                                >{binding.chip8Key
                                                    .toString(16)
                                                    .toUpperCase()}</span
                                            >
                                            <input
                                                type="text"
                                                value={(
                                                    binding.keyboardKey || ""
                                                ).toUpperCase()}
                                                on:focus={(e) =>
                                                    startRemap(
                                                        e,
                                                        binding.chip8Key,
                                                    )}
                                                on:blur={(e) =>
                                                    endRemap(
                                                        e,
                                                        binding.chip8Key,
                                                    )}
                                                class="w-full bg-gray-600 text-white text-center rounded-sm mt-1 focus:outline-none focus:ring-2 focus:ring-blue-500 p-1 cursor-pointer"
                                                readonly
                                            />
                                        </div>
                                    {/each}
                                </div>
                            </div>
                        {/if}
                    </div>
                </div>
            </div>
            <!-- Action Buttons -->
            <div
                class="flex justify-end gap-3 mt-4 border-t border-gray-700 pt-4"
            >
                <button
                    on:click={closeModal}
                    class="bg-gray-600 hover:bg-gray-500 text-white font-medium py-2 px-4 rounded-md transition-colors text-sm"
                    >Cancel</button
                >
                <button
                    on:click={saveSettings}
                    class="bg-blue-600 hover:bg-blue-500 text-white font-medium py-2 px-4 rounded-md transition-colors text-sm"
                    >Save & Close</button
                >
            </div>
        </div>
    </div>
{/if}

```

## File: `frontend/src/lib/clickOutside.js`

```javascript
/**
 * Svelte action: dispatches a 'click_outside' event when a click occurs outside the given node.
 * @param {HTMLElement} node - The element to detect outside clicks for.
 * @returns {{ destroy(): void }}
 */
export function clickOutside(node) {
  const handleClick = (event) => {
    if (node && !node.contains(event.target) && !event.defaultPrevented) {
      node.dispatchEvent(new CustomEvent("click_outside", node));
    }
  };

  document.addEventListener("click", handleClick, true);

  return {
    destroy() {
      document.removeEventListener("click", handleClick, true);
    },
  };
}

```

## File: `frontend/src/lib/stores.js`

```javascript
import { writable } from "svelte/store";
import { SaveSettings } from "../wailsjs/go/main/App.js";

/**
 * Svelte store for notification state.
 * @type {import("svelte/store").Writable<{message: string, type: string, show: boolean}>}
 */
export const notification = writable({
  message: "",
  type: "info",
  show: false,
});

/**
 * Show a notification with a message, type, and duration.
 * @param {string} message
 * @param {string} [type="info"]
 * @param {number} [duration=3000]
 */
export function showNotification(message, type = "info", duration = 3000) {
  notification.set({ message, type, show: true });
  setTimeout(() => {
    notification.update((n) => ({ ...n, show: false }));
  }, duration);
}

/**
 * Default emulator settings.
 * @type {{
 *   clockSpeed: number,
 *   displayColor: string,
 *   scanlineEffect: boolean,
 *   keyMap: Record<string|number, number>
 * }}
 */
const defaultSettings = {
  clockSpeed: 700,
  displayColor: "#33FF00",
  scanlineEffect: false,
  keyMap: {
    1: 0x1,
    2: 0x2,
    3: 0x3,
    4: 0xc,
    q: 0x4,
    w: 0x5,
    e: 0x6,
    r: 0xd,
    a: 0x7,
    s: 0x8,
    d: 0x9,
    f: 0xe,
    z: 0xa,
    x: 0x0,
    c: 0xb,
    v: 0xf,
  },
};

/**
 * Svelte store for emulator settings.
 * @type {import("svelte/store").Writable<typeof defaultSettings>}
 */
export const settings = writable(defaultSettings);

/**
 * Save settings to both the store and the Go backend.
 * @param {typeof defaultSettings} newSettings
 * @returns {Promise<void>}
 */
export async function updateAndSaveSettings(newSettings) {
  try {
    await SaveSettings(newSettings);
    settings.set(newSettings);
    showNotification("Settings saved successfully!", "success");
  } catch (error) {
    showNotification(`Failed to save settings: ${error}`, "error");
    console.error("Settings save error:", error);
  }
}

```

## File: `frontend/src/App.svelte`

```svelte
<script>
    import { onMount } from "svelte";
    import { EventsOn } from "./wailsjs/runtime/runtime.js";
    import { FrontendReady, GetInitialState } from "./wailsjs/go/main/App.js";
    import { settings } from "./lib/stores.js";
    import SettingsModal from "./lib/SettingsModal.svelte";
    import DebugPanel from "./lib/DebugPanel.svelte";
    import Notification from "./lib/Notification.svelte";
    import Header from "./lib/Header.svelte";
    import EmulatorView from "./lib/EmulatorView.svelte";

    /**
     * @typedef {Object} DebugState
     * @property {number[]} Registers
     * @property {any[]} Disassembly
     * @property {number[]} Stack
     * @property {Object} Breakpoints
     * @property {number} PC
     * @property {number} I
     * @property {number} SP
     * @property {number} DelayTimer
     * @property {number} SoundTimer
     */

    /** @type {DebugState} */
    let debugState = {
        Registers: Array(16).fill(0),
        Disassembly: [],
        Stack: Array(16).fill(0),
        Breakpoints: {},
        PC: 0,
        I: 0,
        SP: 0,
        DelayTimer: 0,
        SoundTimer: 0,
    };
    let statusMessage = "Status: Idle | ROM: None";
    let showSettingsModal = false;
    let currentTab = "emulator";

    onMount(async () => {
        EventsOn("debugUpdate", (newState) => {
            debugState = newState;
        });

        EventsOn("statusUpdate", (newStatus) => {
            statusMessage = newStatus;
        });

        await FrontendReady();

        const initialState = await GetInitialState();
        if (initialState.cpuState) {
            debugState = initialState.cpuState;
        }
        if (initialState.settings) {
            settings.set(initialState.settings);
        }
    });

    /** Open the settings modal. */
    function openSettings() {
        showSettingsModal = true;
    }

</script>
<div
    class="flex flex-col h-screen bg-gray-800 text-gray-200 font-sans antialiased"
>
    <Header bind:currentTab on:openSettings={openSettings} />

    <!-- Main Content Area -->
    <main class="flex-grow overflow-hidden">
        {#if currentTab === "emulator"}
            <EmulatorView />
        {:else if currentTab === "debug"}
            <DebugPanel bind:debugState />
        {/if}
    </main>
    <footer
        class="flex-none bg-gray-900 text-gray-400 text-xs text-center py-2 shadow-inner border-t border-gray-800"
    >
        {statusMessage}
    </footer>
    <Notification />
</div>
{#if showSettingsModal}
    <SettingsModal bind:showModal={showSettingsModal} />
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
	"encoding/json"
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

// Settings defines the user-configurable options for the emulator.
type Settings struct {
	ClockSpeed     int            `json:"clockSpeed"`
	DisplayColor   string         `json:"displayColor"`
	ScanlineEffect bool           `json:"scanlineEffect"`
	KeyMap         map[string]int `json:"keyMap"`
}

// DefaultKeyMap returns the default keyboard to CHIP-8 key mappings.
func DefaultKeyMap() map[string]int {
	return map[string]int{
		"1": 0x1, "2": 0x2, "3": 0x3, "4": 0xc,
		"q": 0x4, "w": 0x5, "e": 0x6, "r": 0xd,
		"a": 0x7, "s": 0x8, "d": 0x9, "f": 0xe,
		"z": 0xa, "x": 0x0, "c": 0xb, "v": 0xf,
	}
}

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
	settings      Settings
	settingsPath  string
}

// NewApp creates a new App application struct
func NewApp() *App {
	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Failed to get user config dir: %v", err)
	}
	appConfigDir := filepath.Join(configDir, "chip8-wails")

	return &App{
		cpu:           chip8.New(),
		frontendReady: make(chan struct{}),
		logBuffer:     make([]string, 0, 100),
		isPaused:      true,
		settingsPath:  filepath.Join(appConfigDir, "settings.json"),
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

	// Load settings on startup
	a.loadSettings()

	// Start the main emulation loop
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
			cpuTicker.Reset(a.cpuSpeed)

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

// loadSettings reads settings from disk or creates a default file.
func (a *App) loadSettings() {
	// Ensure the config directory exists
	configDir := filepath.Dir(a.settingsPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)
	}

	data, err := ioutil.ReadFile(a.settingsPath)
	if err != nil {
		// If file doesn't exist, create it with defaults
		a.appendLog("Settings file not found, creating with defaults.")
		a.settings = Settings{
			ClockSpeed:     700,
			DisplayColor:   "#33FF00",
			ScanlineEffect: false,
			KeyMap:         DefaultKeyMap(),
		}
		// Save the new default settings
		a.SaveSettings(a.settings)
		return
	}

	// If file exists, unmarshal it
	if err := json.Unmarshal(data, &a.settings); err != nil {
		a.appendLog(fmt.Sprintf("Error reading settings.json: %v. Using defaults.", err))
		// Handle case of corrupted JSON
		a.settings = Settings{
			ClockSpeed:     700,
			DisplayColor:   "#33FF00",
			ScanlineEffect: false,
			KeyMap:         DefaultKeyMap(),
		}
	} else {
		a.appendLog("Settings loaded successfully.")
	}

	// Apply the loaded clock speed
	a.SetClockSpeed(a.settings.ClockSpeed)
}

// SaveSettings is a new bindable method to save settings from the frontend.
func (a *App) SaveSettings(settings Settings) error {
	a.appendLog("Saving settings...")
	a.settings = settings // Update the app's internal state

	// Apply the new clock speed immediately
	a.SetClockSpeed(settings.ClockSpeed)

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		a.appendLog(fmt.Sprintf("Failed to marshal settings: %v", err))
		return err
	}

	err = ioutil.WriteFile(a.settingsPath, data, 0644)
	if err != nil {
		a.appendLog(fmt.Sprintf("Failed to write settings file: %v", err))
		return err
	}

	a.appendLog("Settings saved successfully.")
	return nil
}

// GetInitialState now needs to include settings
func (a *App) GetInitialState() map[string]interface{} {
	a.appendLog("Frontend connected, providing initial state and settings.")
	return map[string]interface{}{
		"cpuState": a.cpu.GetState(),
		"settings": a.settings,
	}
}

func (a *App) GetDisplay() []byte {
	// This function might not be needed if we push updates, but it's good to have.
	// We return a safe copy.
	displayCopy := make([]byte, len(a.cpu.Display))
	copy(displayCopy, a.cpu.Display[:])
	return displayCopy
}

// LoadROMFromFile opens a file dialog and loads the selected ROM.
func (a *App) LoadROMFromFile() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Load CHIP-8 ROM",
		Filters: []runtime.FileFilter{{DisplayName: "CHIP-8 ROMs (*.ch8, *.c8)", Pattern: "*.ch8;*.c8"}},
	})
	if err != nil || selection == "" {
		return "", err // User cancelled or error
	}

	data, err := ioutil.ReadFile(selection)
	if err != nil {
		errMsg := fmt.Sprintf("Error reading ROM file %s: %v", selection, err)
		a.appendLog(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	romName := filepath.Base(selection)
	a.loadROMFromData(data, romName)

	return romName, nil
}

// Internal helper to avoid code duplication
func (a *App) loadROMFromData(data []byte, romName string) error {
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

// Modify the existing LoadROM to use the helper
func (a *App) LoadROM(romName string) error {
	a.appendLog(fmt.Sprintf("Attempting to load ROM from browser: %s", romName))
	romPath := filepath.Join("roms", romName)
	data, err := ioutil.ReadFile(romPath)
	if err != nil {
		errMsg := fmt.Sprintf("Error reading ROM file %s: %v", romName, err)
		a.appendLog(errMsg)
		return fmt.Errorf(errMsg)
	}
	return a.loadROMFromData(data, romName)
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
func (a *App) PlayBeep() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "playBeep")
	}
}

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
// This is the corrected version.
func (a *App) LoadState(state []byte) error {
	buf := bytes.NewBuffer(state)
	dec := gob.NewDecoder(buf)
	var loadedCPU chip8.Chip8
	if err := dec.Decode(&loadedCPU); err != nil {
		return fmt.Errorf("failed to decode CPU state: %w", err)
	}

	a.cpu = &loadedCPU // The lock in LoadStateFromFile handles safety

	statusMsg := "Emulator state loaded."
	a.appendLog(statusMsg)
	runtime.EventsEmit(a.ctx, "statusUpdate", statusMsg)
	return nil
}

// LoadStateFromFile opens an open dialog, reads a state file, and loads it into the emulator.
// This is the corrected version.
func (a *App) LoadStateFromFile() error {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Load CHIP-8 State",
		Filters: []runtime.FileFilter{{DisplayName: "CHIP-8 State (*.ch8state)", Pattern: "*.ch8state"}},
	})
	if err != nil || selection == "" {
		return err // User cancelled or error
	}

	data, err := ioutil.ReadFile(selection)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// Pause emulation while loading
	a.pauseMutex.Lock()
	defer a.pauseMutex.Unlock()
	a.isPaused = true
	a.cpu.IsRunning = false

	err = a.LoadState(data)

	// After loading, force a UI refresh
	if err == nil {
		a.appendLog("State loaded successfully. Forcing UI refresh.")
		displayData := base64.StdEncoding.EncodeToString(a.cpu.Display[:])
		runtime.EventsEmit(a.ctx, "displayUpdate", displayData)
		runtime.EventsEmit(a.ctx, "debugUpdate", a.cpu.GetState())
	}
	return err
}

func (a *App) GetLogs() []string {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()
	// Return a copy to avoid data races if the buffer is modified while
	// the frontend is processing it.
	logsCopy := make([]string, len(a.logBuffer))
	copy(logsCopy, a.logBuffer)
	return logsCopy
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
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Create an instance of the app structure.
	// NewApp() now correctly initializes the CPU core.
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "chip8-wails",
		Width:            1280,
		Height:           800,
		Frameless:        true, // Frameless window
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 44, G: 62, B: 80, A: 1}, // Matches bg-[#2c3e50]
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			Icon: icon,
		},
		Menu: menu.NewMenuFromItems(
			menu.SubMenu("File", menu.NewMenuFromItems(
								menu.Text("Load ROM", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
					go app.LoadROMFromFile() // Run in a goroutine to not block the UI
				}),
				menu.Separator(),
				menu.Text("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
					if app.ctx != nil {
						runtime.Quit(app.ctx)
					}
				}),
			)),
			menu.SubMenu("Emulation", menu.NewMenuFromItems(
				menu.Text("Pause/Resume", keys.CmdOrCtrl("p"), func(_ *menu.CallbackData) {
					if app.ctx != nil {
						runtime.EventsEmit(app.ctx, "menu:pause")
					}
				}),
			)),
			menu.SubMenu("Help", menu.NewMenuFromItems(
				menu.Text("About", nil, func(_ *menu.CallbackData) {
					if app.ctx != nil {
						runtime.MessageDialog(app.ctx, runtime.MessageDialogOptions{
							Type:    runtime.InfoDialog,
							Title:   "About CHIP-8 Emulator",
							Message: "A CHIP-8 emulator built with Wails and Svelte.",
						})
					}
				}),
			)),
		),
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

```

