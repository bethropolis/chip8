# Project Structure

```
chip8-wails/
├── chip8/
│   ├── chip8.go
│   └── chip8_test.go
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

// UpdateTimers decrements the delay and sound timers if they are greater than 0.
func (c *Chip8) UpdateTimers() {
	if c.DelayTimer > 0 {
		c.DelayTimer--
	}
	if c.SoundTimer > 0 {
		c.SoundTimer--
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

// Struct to parse wails.json (This should be in one place, main.go is fine)
type WailsInfo struct {
	Info struct {
		ProductName string `json:"productName"`
		Version     string `json:"version"`
		Description string `json:"description"`
		ProjectURL  string `json:"projectURL"`
	} `json:"info"`
	Author struct {
		Name string `json:"name"`
	} `json:"author"`
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
	isDebugging   bool // To track if the debug panel is active
	wailsInfo     WailsInfo
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

var frontendReadyOnce sync.Once

func (a *App) FrontendReady() {
	frontendReadyOnce.Do(func() {
		close(a.frontendReady)
	})
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
				a.cpu.UpdateTimers()
			}

			// --- OPTIMIZATION ---
			// Only push updates if the debug panel is active
			if a.isDebugging {
				runtime.EventsEmit(a.ctx, "debugUpdate", a.cpu.GetState())
			}

			// The display update is separate and should always happen if the draw flag is set
			if a.cpu.DrawFlag {
				displayData := base64.StdEncoding.EncodeToString(a.cpu.Display[:])
				runtime.EventsEmit(a.ctx, "displayUpdate", displayData)
				a.cpu.ClearDrawFlag()
			}
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
		log.Printf("ERROR: Failed to unmarshal settings.json: %v", err) // Added log
		// Handle case of corrupted JSON
		a.settings = Settings{
			ClockSpeed:     700,
			DisplayColor:   "#33FF00",
			ScanlineEffect: false,
			KeyMap:         DefaultKeyMap(),
		}
	} else {
		a.appendLog("Settings loaded successfully.")
		log.Printf("DEBUG: Settings loaded: %+v", a.settings) // Added log
	}

	// Apply the loaded clock speed
	a.SetClockSpeed(a.settings.ClockSpeed)
}

// SaveSettings is a new bindable method to save settings from the frontend.
func (a *App) SaveSettings(settings Settings) error {
	a.appendLog("Saving settings...")
	log.Printf("DEBUG: Saving settings: %+v", settings) // Added log
	a.settings = settings                               // Update the app's internal state

	// Apply the new clock speed immediately
	a.SetClockSpeed(settings.ClockSpeed)

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		a.appendLog(fmt.Sprintf("Failed to marshal settings: %v", err))
		log.Printf("ERROR: Failed to marshal settings: %v", err) // Added log
		return err
	}

	err = ioutil.WriteFile(a.settingsPath, data, 0644)
	if err != nil {
		a.appendLog(fmt.Sprintf("Failed to write settings file: %v", err))
		log.Printf("ERROR: Failed to write settings file: %v", err) // Added log
		return err
	}

	a.appendLog("Settings saved successfully.")
	log.Printf("DEBUG: Settings saved to %s", a.settingsPath) // Added log
	return nil
}

// GetInitialState now needs to include settings
func (a *App) GetInitialState() map[string]interface{} {
	a.appendLog("Frontend connected, providing initial state and settings.")
	log.Printf("DEBUG: Sending initial state: cpuState=%+v, settings=%+v", a.cpu.GetState(), a.settings) // Added log
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

// --- NEW BINDABLE METHODS ---

// StartDebugUpdates is called by the frontend when the debug tab is shown.
func (a *App) StartDebugUpdates() {
	a.appendLog("Debug view activated. Starting debug updates.")
	a.isDebugging = true
}

// StopDebugUpdates is called by the frontend when the debug tab is hidden.
func (a *App) StopDebugUpdates() {
	a.appendLog("Debug view deactivated. Stopping debug updates.")
	a.isDebugging = false
}

// ShowAboutDialog constructs and displays a detailed about dialog.
func (a *App) ShowAboutDialog() {
	if a.ctx == nil {
		return
	}
	message := fmt.Sprintf(`%s
Version: %s

%s

Developed by: %s`,
		a.wailsInfo.Info.ProductName,
		a.wailsInfo.Info.Version,
		a.wailsInfo.Info.Description,
		a.wailsInfo.Author.Name,
	)
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   fmt.Sprintf("About %s", a.wailsInfo.Info.ProductName),
		Message: message,
	})
}

// OpenGitHubLink opens the project's GitHub repository in the default browser.
func (a *App) OpenGitHubLink() {
	if a.ctx == nil || a.wailsInfo.Info.ProjectURL == "" {
		return
	}
	runtime.BrowserOpenURL(a.ctx, a.wailsInfo.Info.ProjectURL)
}

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
	"encoding/json" // Import the JSON package
	"log"           // Import log

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

// --- NEW: Embed wails.json to access app info ---
//
//go:embed wails.json
var wailsJSON []byte

func main() {
	app := NewApp()

	var wailsInfo WailsInfo // Using the struct defined in app.go

	err := json.Unmarshal(wailsJSON, &wailsInfo)
	if err != nil {
		log.Fatalf("Failed to parse wails.json: %v", err)
	}
	app.wailsInfo = wailsInfo // Assign the parsed info

	// Create application with options
	err = wails.Run(&options.App{
		Title:     wailsInfo.Info.ProductName, // Use ProductName for the title
		Width:     1280,
		Height:    800,
		Frameless: true, // Frameless window
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
				menu.Text("Load ROM...", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
					go app.LoadROMFromFile()
				}),
				// --- NEW MENU ITEM ---
				menu.Text("Save State", keys.CmdOrCtrl("s"), func(_ *menu.CallbackData) {
					runtime.EventsEmit(app.ctx, "menu:savestate")
				}),
				menu.Separator(),
				menu.Text("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
					runtime.Quit(app.ctx)
				}),
			)),
			menu.SubMenu("Emulation", menu.NewMenuFromItems(
				menu.Text("Pause/Resume", keys.CmdOrCtrl("p"), func(_ *menu.CallbackData) {
					runtime.EventsEmit(app.ctx, "menu:pause")
				}),
				// --- NEW MENU ITEMS ---
				menu.Text("Soft Reset", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
					runtime.EventsEmit(app.ctx, "menu:softreset")
				}),
				menu.Text("Hard Reset", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
					runtime.EventsEmit(app.ctx, "menu:hardreset")
				}),
			)),
			menu.SubMenu("Help", menu.NewMenuFromItems(
				// --- NEW MENU ITEM ---
				menu.Text("Visit GitHub", nil, func(_ *menu.CallbackData) {
					app.OpenGitHubLink()
				}),
				menu.Separator(),

				menu.Text("About", nil, func(_ *menu.CallbackData) {
					app.ShowAboutDialog()
				}),
			)),
		),
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

```

