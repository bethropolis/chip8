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
