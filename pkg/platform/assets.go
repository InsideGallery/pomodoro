package platform

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// AssetFS is the filesystem used to load game assets.
// On desktop: the real filesystem. On mobile: embed.FS set at startup.
var assetFS fs.FS //nolint:gochecknoglobals // set once at startup

// SetAssetFS sets the embedded filesystem for mobile.
func SetAssetFS(f fs.FS) {
	assetFS = f
}

// OpenAsset opens a file from the asset filesystem.
// On mobile: reads from embed.FS.
// On desktop: falls back to real filesystem with path candidates.
func OpenAsset(name string) (io.ReadCloser, error) {
	if assetFS != nil {
		return assetFS.Open(name)
	}

	// Desktop: try relative paths
	candidates := []string{
		filepath.Join("assets", name),
		filepath.Join("..", "assets", name),
		filepath.Join("..", "..", "assets", name),
		name,
	}

	for _, p := range candidates {
		f, err := os.Open(p)
		if err == nil {
			return f, nil
		}
	}

	return nil, os.ErrNotExist
}

// ReadAsset reads the full contents of an asset file.
func ReadAsset(name string) ([]byte, error) {
	f, err := OpenAsset(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

// AssetExists checks if an asset path exists.
func AssetExists(name string) bool {
	f, err := OpenAsset(name)
	if err != nil {
		return false
	}

	f.Close()

	return true
}
