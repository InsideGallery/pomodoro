// Package resources provides a unified resource manager for embedded and disk assets.
// Core resources (fonts, sounds, icons) come from embed.FS.
// Plugin resources (maps, sprites, images) can be loaded from disk.
// Supports async loading with progress tracking for preloader scenes.
package resources

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
)

// Kind identifies the type of resource.
type Kind string

const (
	KindImage Kind = "image"
	KindRaw   Kind = "raw"
)

// Manager manages resources from multiple sources (embedded, disk).
// Thread-safe for concurrent access and async loading.
type Manager struct {
	mu    sync.RWMutex
	cache map[string]any

	// Async loading state
	total   atomic.Int64
	loaded  atomic.Int64
	loading atomic.Bool
}

// NewManager creates a new resource manager.
func NewManager() *Manager {
	return &Manager{
		cache: make(map[string]any),
	}
}

// Get retrieves a cached resource by key.
func (m *Manager) Get(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.cache[key]

	return v, ok
}

// GetImage retrieves a cached *ebiten.Image by key.
func (m *Manager) GetImage(key string) (*ebiten.Image, bool) {
	v, ok := m.Get(key)
	if !ok {
		return nil, false
	}

	img, ok := v.(*ebiten.Image)

	return img, ok
}

// Set stores a resource in the cache.
func (m *Manager) Set(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache[key] = value
}

// LoadImageFromFS loads a PNG image from an fs.FS and caches it.
func (m *Manager) LoadImageFromFS(fsys fs.FS, path, cacheKey string) error {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}

	eImg := ebiten.NewImageFromImage(img)
	m.Set(cacheKey, eImg)

	return nil
}

// LoadRawFromFS loads raw bytes from an fs.FS and caches them.
func (m *Manager) LoadRawFromFS(fsys fs.FS, path, cacheKey string) error {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	m.Set(cacheKey, data)

	return nil
}

// LoadImageFromBytes decodes a PNG from bytes and caches it.
func (m *Manager) LoadImageFromBytes(data []byte, cacheKey string) error {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	m.Set(cacheKey, ebiten.NewImageFromImage(img))

	return nil
}

// --- Async loading ---

// LoadTask represents a single resource to load asynchronously.
type LoadTask struct {
	Key  string
	Load func() (any, error) // returns the resource value
}

// LoadAsync starts loading resources in the background.
// Check Progress() and IsLoading() for status.
func (m *Manager) LoadAsync(tasks []LoadTask) {
	m.total.Store(int64(len(tasks)))
	m.loaded.Store(0)
	m.loading.Store(true)

	go func() {
		defer m.loading.Store(false)

		for _, task := range tasks {
			val, err := task.Load()
			if err != nil {
				slog.Warn("resource load", "key", task.Key, "error", err)
			} else {
				m.Set(task.Key, val)
			}

			m.loaded.Add(1)
		}
	}()
}

// Progress returns (loaded, total) for async loading.
func (m *Manager) Progress() (int, int) {
	return int(m.loaded.Load()), int(m.total.Load())
}

// IsLoading returns true while async loading is in progress.
func (m *Manager) IsLoading() bool {
	return m.loading.Load()
}

// Clear removes all cached resources.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache = make(map[string]any)
}
