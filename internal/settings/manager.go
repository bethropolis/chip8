package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Settings struct {
	ClockSpeed     int            `json:"clockSpeed"`
	DisplayColor   string         `json:"displayColor"`
	ScanlineEffect bool           `json:"scanlineEffect"`
	KeyMap         map[string]int `json:"keyMap"`
	PixelScale     int            `json:"pixelScale"`
	RomsPath       string         `json:"romsPath"`
}

/*
DefaultSettings returns a new Settings object with default values.
*/
func DefaultSettings() Settings {
	return Settings{
		ClockSpeed:     700,
		DisplayColor:   "#33FF00",
		ScanlineEffect: false,
		PixelScale:     10,
		RomsPath:       "./roms",
		KeyMap: map[string]int{
			"1": 0x1, "2": 0x2, "3": 0x3, "4": 0xc,
			"q": 0x4, "w": 0x5, "e": 0x6, "r": 0xd,
			"a": 0x7, "s": 0x8, "d": 0x9, "f": 0xe,
			"z": 0xa, "x": 0x0, "c": 0xb, "v": 0xf,
		},
	}
}

type Manager struct {
	path string
}

/*
NewManager creates a new settings Manager.
*/
func NewManager(path string) *Manager {
	return &Manager{path: path}
}

/*
Load reads settings from the file system. If the file doesn't exist,
it creates one with default settings.
*/
func (m *Manager) Load() (Settings, error) {
	data, err := ioutil.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			s := DefaultSettings()
			err := m.Save(s)
			return s, err
		}
		return Settings{}, fmt.Errorf("failed to read settings file: %w", err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		fmt.Printf("Warning: could not parse settings.json, falling back to defaults: %v\n", err)
		s = DefaultSettings() // Use defaults if parsing fails
	}
	// Ensure new fields have default values if loading old settings file
	if s.PixelScale == 0 {
		s.PixelScale = 10
	}
	if s.RomsPath == "" {
		s.RomsPath = "./roms"
	}
	return s, nil
}

/*
Save writes the given settings to the file system.
*/
func (m *Manager) Save(s Settings) error {
	configDir := filepath.Dir(m.path)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("could not create config directory: %w", err)
		}
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	return ioutil.WriteFile(m.path, data, 0644)
}
