package platform

import (
	"os"
	"path/filepath"
)

// dirPerm is the permission for directories the app creates: owner rwx, others rx.
const dirPerm = 0o755

// DataDir returns the application data directory.
// On desktop: ~/.config/pomodoro/
// On mobile: set by the app at startup via SetDataDir.
var dataDir string //nolint:gochecknoglobals // set once at startup

// SetDataDir sets the application data directory (called by mobile entry point).
func SetDataDir(dir string) {
	dataDir = dir
}

// GetDataDir returns the application data directory.
func GetDataDir() string {
	if dataDir != "" {
		return dataDir
	}

	// Desktop default
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	return filepath.Join(home, ".config", "pomodoro")
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, dirPerm)
}
