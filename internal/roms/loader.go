package roms

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Loader handles finding and loading ROM files.
type Loader struct {
	RomsDir string
}

// NewLoader creates a new ROM loader.
func NewLoader(romsDir string) *Loader {
	// Ensure the 'roms' directory exists
	if _, err := os.Stat(romsDir); os.IsNotExist(err) {
		os.Mkdir(romsDir, 0755)
	}
	return &Loader{RomsDir: romsDir}
}

// List returns a list of available ROM filenames in the roms directory.
func (l *Loader) List() ([]string, error) {
	files, err := ioutil.ReadDir(l.RomsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read ROMs directory: %w", err)
	}

	var romNames []string
	for _, file := range files {
		name := strings.ToLower(file.Name())
		if !file.IsDir() && (strings.HasSuffix(name, ".ch8") || strings.HasSuffix(name, ".c8")) {
			romNames = append(romNames, file.Name())
		}
	}
	return romNames, nil
}

// LoadFromDir loads a ROM by its filename from the configured ROMs directory.
func (l *Loader) LoadFromDir(filename string) ([]byte, error) {
	path := filepath.Join(l.RomsDir, filename)
	return l.LoadFromPath(path)
}

// LoadFromPath loads a ROM from an absolute or relative file path.
// It returns the ROM data.
func (l *Loader) LoadFromPath(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading ROM file %s: %w", path, err)
	}
	return data, nil
}
