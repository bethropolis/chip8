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
