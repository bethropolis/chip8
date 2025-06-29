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
