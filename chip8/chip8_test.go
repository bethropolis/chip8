package chip8

import (
	"testing"
)

/*
TestNewChip8 verifies that a new Chip8 instance is initialized correctly,
including registers, program counter, stack pointer, and font set.
*/
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

	for i := 0; i < len(FontSet); i++ {
		if c.Memory[FontSetStart+i] != FontSet[i] {
			t.Errorf("FontSet not loaded correctly at 0x%X", FontSetStart+i)
		}
	}
}

/*
TestLoadROM checks that loading a ROM places its bytes in the correct memory locations,
and that an error is returned if the ROM is too large.
*/
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

	largeROM := make([]byte, 4096-ProgramStart+1)
	err = c.LoadROM(largeROM)
	if err == nil {
		t.Error("Expected error for large ROM, got nil")
	}
}

/*
TestOpcode00E0 verifies that the CLS opcode clears the display and sets the draw flag.
*/
func TestOpcode00E0(t *testing.T) {
	c := New()
	c.Display[0] = 1
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

/*
TestOpcode00EE checks that the RET opcode pops the stack and sets the PC correctly.
*/
func TestOpcode00EE(t *testing.T) {
	c := New()
	c.Stack[0] = 0x300
	c.SP = 1
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0x00
	c.Memory[ProgramStart+1] = 0xEE

	c.EmulateCycle()

	if c.SP != 0 {
		t.Errorf("Expected SP to be 0, got %d", c.SP)
	}
	if c.PC != 0x300 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", 0x300, c.PC)
	}
}

/*
TestOpcode1NNN checks that the JP opcode sets the PC to the correct address.
*/
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

/*
TestOpcode6XNN checks that the LD Vx, byte opcode sets the register correctly.
*/
func TestOpcode6XNN(t *testing.T) {
	c := New()
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0x6A
	c.Memory[ProgramStart+1] = 0x55

	c.EmulateCycle()

	if c.Registers[0xA] != 0x55 {
		t.Errorf("Expected V[A] to be 0x%X, got 0x%X", 0x55, c.Registers[0xA])
	}
	if c.PC != ProgramStart+2 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", ProgramStart+2, c.PC)
	}
}

/*
TestOpcode7XNN checks that the ADD Vx, byte opcode adds the value to the register.
*/
func TestOpcode7XNN(t *testing.T) {
	c := New()
	c.Registers[0xB] = 0x10
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0x7B
	c.Memory[ProgramStart+1] = 0x05

	c.EmulateCycle()

	if c.Registers[0xB] != 0x15 {
		t.Errorf("Expected V[B] to be 0x%X, got 0x%X", 0x15, c.Registers[0xB])
	}
	if c.PC != ProgramStart+2 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", ProgramStart+2, c.PC)
	}
}

/*
TestOpcodeANNN checks that the LD I, addr opcode sets the I register correctly.
*/
func TestOpcodeANNN(t *testing.T) {
	c := New()
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0xA1
	c.Memory[ProgramStart+1] = 0x23

	c.EmulateCycle()

	if c.I != 0x0123 {
		t.Errorf("Expected I to be 0x%X, got 0x%X", 0x0123, c.I)
	}
	if c.PC != ProgramStart+2 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", ProgramStart+2, c.PC)
	}
}

/*
TestOpcodeDXYN checks that the DRW Vx, Vy, nibble opcode draws the sprite,
sets the draw flag, and detects pixel collisions (setting VF).
*/
func TestOpcodeDXYN(t *testing.T) {
	c := New()
	c.Registers[0x0] = 0
	c.Registers[0x1] = 0
	c.I = FontSetStart
	c.PC = ProgramStart
	c.Memory[ProgramStart] = 0xD0
	c.Memory[ProgramStart+1] = 0x15

	c.EmulateCycle()

	if !c.DrawFlag {
		t.Error("DrawFlag not set")
	}
	if c.PC != ProgramStart+2 {
		t.Errorf("Expected PC to be 0x%X, got 0x%X", ProgramStart+2, c.PC)
	}

	if c.Display[0] != 1 {
		t.Errorf("Expected pixel (0,0) to be 1, got %d", c.Display[0])
	}

	if c.Display[1*DisplayWidth+0] != 1 {
		t.Errorf("Expected pixel (0,1) to be 1, got %d", c.Display[1*DisplayWidth+0])
	}

	c.Registers[0xF] = 0
	c.PC = ProgramStart
	c.EmulateCycle()

	if c.Registers[0xF] != 1 {
		t.Errorf("Expected VF to be 1 after collision, got %d", c.Registers[0xF])
	}
}
