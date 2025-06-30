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
