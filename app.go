package main

import (
	"bytes"
	"chip8-wails/chip8"
	"chip8-wails/internal/roms"
	"chip8-wails/internal/settings"
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

const debugUpdateInterval = time.Millisecond * 100 // ~10Hz throttle (1000ms / 100ms = 10 updates/sec)

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

type App struct {
	ctx                 context.Context
	cpu                 *chip8.Chip8
	frontendReady       chan struct{}
	cpuSpeed            time.Duration
	logBuffer           []string
	logMutex            sync.Mutex
	mu                  sync.RWMutex
	isPaused            bool
	isDebugging         bool
	wailsInfo           WailsInfo
	romLoaded           []byte
	settings            settings.Settings
	settingsManager     *settings.Manager
	romLoader           *roms.Loader
	lastDebugUpdateTime time.Time
}

/*
NewApp creates a new App instance, initializing configuration and dependencies.
*/
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

var frontendReadyOnce sync.Once

// FrontendReady signals that the frontend is ready to receive events.
func (a *App) FrontendReady() {
	frontendReadyOnce.Do(func() {
		close(a.frontendReady)
	})
}

/*
startup initializes the application context, loads settings, and starts the emulator loop.
*/
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	loadedSettings, err := a.settingsManager.Load()
	if err != nil {
		log.Fatalf("FATAL: Could not load or create settings: %v", err)
	}
	a.mu.Lock()
	a.settings = loadedSettings
	a.mu.Unlock()
	a.appendLog("Settings loaded successfully.")
	a.SetClockSpeed(loadedSettings.ClockSpeed)
	go a.runEmulator()
}

/*
runEmulator is the main loop for the emulator, handling CPU cycles, timers, and event emission.
*/
func (a *App) runEmulator() {
	<-a.frontendReady
	log.Println("Frontend is ready, starting emulation loop.")

	a.mu.RLock()
	speed := a.settings.ClockSpeed
	a.mu.RUnlock()

	if speed <= 0 {
		speed = 700
		a.appendLog(fmt.Sprintf("Warning: Invalid clock speed detected, falling back to %d Hz", speed))
	}

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
			if currentSpeed > 0 && int(time.Second/a.cpuSpeed) != currentSpeed {
				cpuTicker.Reset(time.Second / time.Duration(currentSpeed))
			}
			a.mu.RLock()
			isRunning := !a.isPaused
			a.mu.RUnlock()
			if isRunning {
				a.cpu.EmulateCycle()
			}

		case <-timerTicker.C:
			a.mu.Lock()
			isRunning := !a.isPaused
			isDebugging := a.isDebugging
			soundTimer := a.cpu.SoundTimer
			drawFlag := a.cpu.DrawFlag

			if isRunning {
				a.cpu.UpdateTimers()
				if soundTimer > 0 {
					a.emit("playBeep")
				}
			}

			if isDebugging && time.Since(a.lastDebugUpdateTime) >= debugUpdateInterval {
				state := a.cpu.GetState()
				a.lastDebugUpdateTime = time.Now()
				a.mu.Unlock()
				a.emit("debugUpdate", state)
			} else {
				a.mu.Unlock()
			}

			if drawFlag {
				displayData := base64.StdEncoding.EncodeToString(a.cpu.Display[:])
				a.emit("displayUpdate", displayData)
				a.cpu.ClearDrawFlag()
			}
		}
	}
}

/*
SaveSettings persists new settings and updates the emulator's configuration.
*/
func (a *App) SaveSettings(newSettings settings.Settings) error {
	a.appendLog("Saving settings...")
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.settingsManager.Save(newSettings); err != nil {
		a.appendLog(fmt.Sprintf("Failed to write settings file: %v", err))
		return err
	}
	a.settings = newSettings
	a.setClockSpeedInternal(newSettings.ClockSpeed)
	a.appendLog("Settings saved successfully.")
	return nil
}

/*
GetInitialState returns the current CPU state and settings for the frontend.
*/
func (a *App) GetInitialState() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	a.appendLog("Frontend connected, providing initial state.")
	return map[string]interface{}{
		"cpuState": a.cpu.GetState(),
		"settings": a.settings,
	}
}

/*
loadROMFromData loads a ROM into the emulator and updates state.
*/
func (a *App) loadROMFromData(data []byte, romName string) {
	a.cpu.Reset()
	if err := a.cpu.LoadROM(data); err != nil {
		errMsg := fmt.Sprintf("Error loading ROM data %s: %v", romName, err)
		a.appendLog(errMsg)
		return
	}
	a.mu.Lock()
	a.romLoaded = data
	a.isPaused = false
	a.cpu.IsRunning = true
	a.mu.Unlock()
	statusMsg := fmt.Sprintf("Status: Running | ROM: %s", romName)
	a.emit("statusUpdate", statusMsg)
	a.appendLog(statusMsg)
	a.emit("pauseUpdate", false)
}

/*
LoadROMFromFile opens a file dialog for the user to select a ROM file and loads it.
*/
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

/*
LoadROMByPath loads a ROM from a given file path.
*/
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

/*
LoadROM loads a ROM by name from the ROMs directory.
*/
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

/*
GetROMs returns a list of available ROMs.
*/
func (a *App) GetROMs() ([]string, error) {
	return a.romLoader.List()
}

/*
SoftReset reloads the currently loaded ROM, if any.
*/
func (a *App) SoftReset() error {
	a.mu.RLock()
	romToLoad := a.romLoaded
	a.mu.RUnlock()
	if romToLoad == nil {
		return fmt.Errorf("no ROM loaded to soft reset")
	}
	a.loadROMFromData(romToLoad, "previously loaded ROM")
	a.appendLog("Soft reset complete.")
	return nil
}

/*
HardReset resets the emulator state and clears the loaded ROM.
*/
func (a *App) HardReset() {
	a.mu.Lock()
	a.isPaused = true
	a.cpu.Reset()
	a.romLoaded = nil
	a.mu.Unlock()
	statusMsg := "Status: Hard Reset | ROM cleared."
	a.appendLog(statusMsg)
	a.emit("statusUpdate", statusMsg)
	a.emit("pauseUpdate", true)
	a.emit("displayUpdate", base64.StdEncoding.EncodeToString(a.cpu.Display[:]))
	a.emit("debugUpdate", a.cpu.GetState())
}

/*
TogglePause toggles the paused state of the emulator.
*/
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
	a.emit("pauseUpdate", isPausedNow)
	return isPausedNow
}

/*
GetMemory returns a base64-encoded slice of memory from the emulator.
*/
func (a *App) GetMemory(offset, limit int) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	memLen := len(a.cpu.Memory)
	if offset < 0 || limit <= 0 || offset >= memLen {
		return ""
	}
	if offset+limit > memLen {
		limit = memLen - offset
	}
	return base64.StdEncoding.EncodeToString(a.cpu.Memory[offset : offset+limit])
}

/*
SetClockSpeed updates the emulator's clock speed.
*/
func (a *App) SetClockSpeed(speed int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.setClockSpeedInternal(speed)
}

func (a *App) setClockSpeedInternal(speed int) {
	if speed > 0 {
		a.cpuSpeed = time.Second / time.Duration(speed)
		if a.settings.ClockSpeed != speed {
			a.settings.ClockSpeed = speed
		}
		a.emit("clockSpeedUpdate", speed)
		a.appendLog(fmt.Sprintf("Clock speed set to %d Hz", speed))
	}
}

/*
emit sends an event to the frontend if the context is available.
*/
func (a *App) emit(eventName string, data ...interface{}) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, eventName, data...)
}

/*
appendLog adds a message to the log buffer and prints it.
*/
func (a *App) appendLog(msg string) {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()
	log.Println(msg)
	if len(a.logBuffer) >= 100 {
		a.logBuffer = a.logBuffer[1:]
	}
	a.logBuffer = append(a.logBuffer, time.Now().Format("15:04:05")+" | "+msg)
}

/*
GetLogs returns a copy of the current log buffer.
*/
func (a *App) GetLogs() []string {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()
	logsCopy := make([]string, len(a.logBuffer))
	copy(logsCopy, a.logBuffer)
	return logsCopy
}

/*
KeyDown sets the specified key as pressed.
*/
func (a *App) KeyDown(key int) {
	if key >= 0 && key < 16 {
		a.cpu.Keys[key] = true
	}
}

/*
KeyUp sets the specified key as released.
*/
func (a *App) KeyUp(key int) {
	if key >= 0 && key < 16 {
		a.cpu.Keys[key] = false
	}
}

/*
StartDebugUpdates enables debug state updates.
*/
func (a *App) StartDebugUpdates() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isDebugging = true
	a.appendLog("Debug view activated.")
}

/*
StopDebugUpdates disables debug state updates.
*/
func (a *App) StopDebugUpdates() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isDebugging = false
	a.appendLog("Debug view deactivated.")
}

/*
LoadStateFromFile loads a saved emulator state from a file.
*/
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
	if err := gob.NewDecoder(bytes.NewBuffer(data)).Decode(&loadedCPU); err != nil {
		return fmt.Errorf("failed to decode CPU state: %w", err)
	}
	a.cpu = &loadedCPU
	a.appendLog("State loaded successfully. Forcing UI refresh.")
	a.emit("displayUpdate", base64.StdEncoding.EncodeToString(a.cpu.Display[:]))
	a.emit("debugUpdate", a.cpu.GetState())
	a.emit("pauseUpdate", true)
	return nil
}

/*
SaveStateToFile saves the current emulator state to a file.
*/
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

/*
SetBreakpoint sets a breakpoint at the given address.
*/
func (a *App) SetBreakpoint(address uint16) {
	if a.cpu != nil {
		a.mu.Lock()
		a.cpu.Breakpoints[address] = true
		a.mu.Unlock()
		a.appendLog(fmt.Sprintf("Breakpoint set at 0x%04X", address))
	}
}

/*
ClearBreakpoint removes a breakpoint at the given address.
*/
func (a *App) ClearBreakpoint(address uint16) {
	if a.cpu != nil {
		a.mu.Lock()
		delete(a.cpu.Breakpoints, address)
		a.mu.Unlock()
		a.appendLog(fmt.Sprintf("Breakpoint cleared at 0x%04X", address))
	}
}

/*
ShowAboutDialog displays an about dialog with application information.
*/
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

/*
OpenGitHubLink opens the project's GitHub URL in the browser.
*/
func (a *App) OpenGitHubLink() {
	if a.ctx == nil || a.wailsInfo.Info.ProjectURL == "" {
		return
	}
	runtime.BrowserOpenURL(a.ctx, a.wailsInfo.Info.ProjectURL)
}

/*
SaveScreenshot saves a base64-encoded PNG screenshot to a file.
*/
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
