package main

import (
	"bytes"
	"chip8-wails/chip8"
	"chip8-wails/internal/roms"      // NEW
	"chip8-wails/internal/settings" // NEW
	"context"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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
	ctx             context.Context
	cpu             *chip8.Chip8
	frontendReady   chan struct{}
	cpuSpeed        time.Duration // Use time.Duration for clarity
	logBuffer       []string
	logMutex        sync.Mutex
	mu              sync.RWMutex
	isPaused        bool
	isDebugging     bool
	wailsInfo       WailsInfo
	romLoaded       []byte
	settings        settings.Settings // Use the new settings struct
	settingsManager *settings.Manager // NEW
	romLoader       *roms.Loader      // NEW
}

// NewApp creates a new App application struct
func NewApp() *App {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Failed to get user config dir: %v", err)
	}
	appConfigDir := filepath.Join(configDir, "chip8-wails")
	settingsPath := filepath.Join(appConfigDir, "settings.json")

	return &App{
		cpu:             chip8.New(),
		frontendReady:   make(chan struct{}),
		logBuffer:       make([]string, 0, 100),
		isPaused:        true,
		settingsManager: settings.NewManager(settingsPath),
		romLoader:       roms.NewLoader("./roms"),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load settings on startup using the manager
	loadedSettings, err := a.settingsManager.Load()
	if err != nil {
		log.Fatalf("FATAL: Could not load or create settings: %v", err)
	}
	a.settings = loadedSettings
	a.appendLog("Settings loaded successfully.")
	a.SetClockSpeed(a.settings.ClockSpeed)

	go a.runEmulator()
}

// SaveSettings now delegates to the manager
func (a *App) SaveSettings(newSettings settings.Settings) error {
	a.appendLog("Saving settings...")

	if err := a.settingsManager.Save(newSettings); err != nil {
		a.appendLog(fmt.Sprintf("Failed to write settings file: %v", err))
		return err
	}

	a.settings = newSettings
	a.SetClockSpeed(newSettings.ClockSpeed)

	a.appendLog("Settings saved successfully.")
	return nil
}

// GetInitialState returns the state and settings from our app struct
func (a *App) GetInitialState() map[string]interface{} {
	a.appendLog("Frontend connected, providing initial state.")
	return map[string]interface{}{
		"cpuState": a.cpu.GetState(),
		"settings": a.settings,
	}
}

// loadROMFromData is the internal helper that now just deals with the CPU
func (a *App) loadROMFromData(data []byte, romName string) {
	a.cpu.Reset()
	if err := a.cpu.LoadROM(data); err != nil {
		errMsg := fmt.Sprintf("Error loading ROM data %s: %v", romName, err)
		a.appendLog(errMsg)
		// Optionally emit an error event to the frontend
		return
	}

	a.mu.Lock()
	a.romLoaded = data
	a.isPaused = false
	a.cpu.IsRunning = true
	a.mu.Unlock()

	statusMsg := fmt.Sprintf("Status: Running | ROM: %s", romName)
	runtime.EventsEmit(a.ctx, "statusUpdate", statusMsg)
	a.appendLog(statusMsg)

	// Emit pause state update
	runtime.EventsEmit(a.ctx, "pauseUpdate", false)
}

// LoadROMFromFile delegates to the loader and then loads the data
func (a *App) LoadROMFromFile() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Load CHIP-8 ROM",
		Filters: []runtime.FileFilter{{DisplayName: "CHIP-8 ROMs (*.ch8, *.c8)", Pattern: "*.ch8;*.c8"}},
	})
	if err != nil || selection == "" {
		return "", err
	}
	return a.LoadROMByPath(selection)
}

// LoadROMByPath delegates to the loader
func (a *App) LoadROMByPath(path string) (string, error) {
	a.appendLog(fmt.Sprintf("Attempting to load ROM from path: %s", path))
	data, err := a.romLoader.LoadFromPath(path)
	if err != nil {
		a.appendLog(err.Error())
		return "", err
	}
	romName := filepath.Base(path)
	a.loadROMFromData(data, romName)
	return romName, nil
}

// LoadROM loads from the browser list, delegating to the loader
func (a *App) LoadROM(romName string) error {
	a.appendLog(fmt.Sprintf("Attempting to load ROM from browser: %s", romName))
	data, err := a.romLoader.LoadFromDir(romName)
	if err != nil {
		a.appendLog(err.Error())
		return err
	}
	a.loadROMFromData(data, romName)
	return nil
}

// GetROMs delegates to the loader
func (a *App) GetROMs() ([]string, error) {
	return a.romLoader.List()
}

// SoftReset re-uses the stored romLoaded data
func (a *App) SoftReset() error {
	a.mu.RLock()
	romToLoad := a.romLoaded
	a.mu.RUnlock()

	if romToLoad == nil {
		return fmt.Errorf("no ROM loaded to soft reset")
	}

	romName := "previously loaded ROM" // We don't have the name, but that's ok
	a.loadROMFromData(romToLoad, romName)
	a.appendLog("Soft reset complete.")
	return nil
}

// The rest of your `app.go` file (runEmulator, logging, HardReset, TogglePause, state saving, etc.)
// can remain largely the same, as they are part of the core application coordination logic.

// ... (paste the rest of your app.go functions here, they should work with the refactored struct)
// e.g., appendLog, runEmulator, HardReset, TogglePause, KeyDown/Up, etc.
// ...
func (a *App) runEmulator() {
	<-a.frontendReady
	a.mu.RLock()
	speed := a.settings.ClockSpeed
	a.mu.RUnlock()

	cpuTicker := time.NewTicker(time.Second / time.Duration(speed))
	timerTicker := time.NewTicker(time.Second / 60)
	defer cpuTicker.Stop()
	defer timerTicker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-cpuTicker.C:
			a.mu.RLock()
			currentSpeed := a.settings.ClockSpeed
			a.mu.RUnlock()
			if int(time.Second/a.cpuSpeed) != currentSpeed {
				cpuTicker.Reset(time.Second / time.Duration(currentSpeed))
			}

			a.mu.RLock()
			isRunning := !a.isPaused
			a.mu.RUnlock()
			if isRunning {
				a.cpu.EmulateCycle()
			}
		case <-timerTicker.C:
			a.mu.RLock()
			isRunning := !a.isPaused
			isDebugging := a.isDebugging
			a.mu.RUnlock()
			if isRunning {
				a.cpu.UpdateTimers()
				if a.cpu.SoundTimer > 0 {
					runtime.EventsEmit(a.ctx, "playBeep")
				}
			}
			if isDebugging {
				runtime.EventsEmit(a.ctx, "debugUpdate", a.cpu.GetState())
			}
			if a.cpu.DrawFlag {
				displayData := base64.StdEncoding.EncodeToString(a.cpu.Display[:])
				runtime.EventsEmit(a.ctx, "displayUpdate", displayData)
				a.cpu.ClearDrawFlag()
			}
		}
	}
}

func (a *App) appendLog(msg string) {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()
	log.Println(msg)
	if len(a.logBuffer) >= 100 {
		a.logBuffer = a.logBuffer[1:]
	}
	a.logBuffer = append(a.logBuffer, time.Now().Format("15:04:05")+" | "+msg)
}

func (a *App) HardReset() {
	a.mu.Lock()
	a.isPaused = true
	a.cpu.Reset()
	a.romLoaded = nil
	a.mu.Unlock()
	statusMsg := "Status: Hard Reset | ROM cleared."
	a.appendLog(statusMsg)
	runtime.EventsEmit(a.ctx, "statusUpdate", statusMsg)
	runtime.EventsEmit(a.ctx, "pauseUpdate", true)
	displayData := base64.StdEncoding.EncodeToString(a.cpu.Display[:])
	runtime.EventsEmit(a.ctx, "displayUpdate", displayData)
	runtime.EventsEmit(a.ctx, "debugUpdate", a.cpu.GetState())
}
func (a *App) TogglePause() bool {
	a.mu.Lock()
	a.isPaused = !a.isPaused
	a.cpu.IsRunning = !a.isPaused
	isPausedNow := a.isPaused
	a.mu.Unlock()
	if isPausedNow {
		a.appendLog("Emulation Paused.")
	} else {
		a.appendLog("Emulation Resumed.")
	}
	runtime.EventsEmit(a.ctx, "pauseUpdate", isPausedNow)
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
func (a *App) SetBreakpoint(address uint16) {
	if a.cpu != nil {
		a.cpu.Breakpoints[address] = true
		a.appendLog(fmt.Sprintf("Breakpoint set at 0x%04X", address))
	}
}
func (a *App) ClearBreakpoint(address uint16) {
	if a.cpu != nil {
		delete(a.cpu.Breakpoints, address)
		a.appendLog(fmt.Sprintf("Breakpoint cleared at 0x%04X", address))
	}
}
func (a *App) StartDebugUpdates() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isDebugging = true
	a.appendLog("Debug view activated.")
}
func (a *App) StopDebugUpdates() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isDebugging = false
	a.appendLog("Debug view deactivated.")
}
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
func (a *App) OpenGitHubLink() {
	if a.ctx == nil || a.wailsInfo.Info.ProjectURL == "" {
		return
	}
	runtime.BrowserOpenURL(a.ctx, a.wailsInfo.Info.ProjectURL)
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
	return base64.StdEncoding.EncodeToString(mem[offset : offset+limit])
}
func (a *App) SetClockSpeed(speed int) {
	if speed > 0 {
		a.mu.Lock()
		a.cpuSpeed = time.Second / time.Duration(speed)
		if a.settings.ClockSpeed != speed {
			a.settings.ClockSpeed = speed
		}
		a.mu.Unlock()
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
	a.appendLog(fmt.Sprintf("Screenshot saved to: %s", selection))
	return nil
}
func (a *App) SaveStateToFile() error {
	a.mu.Lock()
	a.isPaused = true
	a.cpu.IsRunning = false
	a.mu.Unlock()

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(a.cpu); err != nil {
		return fmt.Errorf("failed to encode CPU state: %w", err)
	}
	selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save CHIP-8 State",
		Filters:         []runtime.FileFilter{{DisplayName: "CHIP-8 State (*.ch8state)", Pattern: "*.ch8state"}},
		DefaultFilename: "chip8_state.ch8state",
	})
	if err != nil || selection == "" {
		return err
	}
	if err := ioutil.WriteFile(selection, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	a.appendLog(fmt.Sprintf("State saved to: %s", selection))
	return nil
}
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

	a.mu.Lock()
	defer a.mu.Unlock()
	a.isPaused = true
	a.cpu.IsRunning = false

	var loadedCPU chip8.Chip8
	if err := gob.NewEncoder(bytes.NewBuffer(data)).Encode(&loadedCPU); err != nil {
		return fmt.Errorf("failed to decode CPU state: %w", err)
	}
	a.cpu = &loadedCPU

	a.appendLog("State loaded successfully. Forcing UI refresh.")
	displayData := base64.StdEncoding.EncodeToString(a.cpu.Display[:])
	runtime.EventsEmit(a.ctx, "displayUpdate", displayData)
	runtime.EventsEmit(a.ctx, "debugUpdate", a.cpu.GetState())
	runtime.EventsEmit(a.ctx, "pauseUpdate", true)
	return nil
}
func (a *App) GetLogs() []string {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()
	logsCopy := make([]string, len(a.logBuffer))
	copy(logsCopy, a.logBuffer)
	return logsCopy
}