package scene

import (
	"context"
	"errors"
	"sync"
)

var ErrSceneNotFound = errors.New("scene not found")

// Manager manages named scenes and switches between them.
type Manager struct {
	mu      sync.RWMutex
	scenes  map[string]Scene
	current Scene
}

func NewManager() *Manager {
	return &Manager{
		scenes: map[string]Scene{},
	}
}

// Add registers scenes and calls Init() on each.
func (m *Manager) Add(ctx context.Context, scenes ...Scene) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sc := range scenes {
		sc.Init(ctx)
		m.scenes[sc.Name()] = sc
	}
}

// SwitchSceneTo transitions to the named scene.
// Calls Load() on the new scene and Unload() on the previous.
func (m *Manager) SwitchSceneTo(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sc, ok := m.scenes[name]
	if !ok || sc == nil {
		return ErrSceneNotFound
	}

	if err := sc.Load(); err != nil {
		return err
	}

	prev := m.current
	m.current = sc

	if prev != nil {
		if err := prev.Unload(); err != nil {
			return err
		}
	}

	return nil
}

// Scene returns the current active scene.
// SceneByName returns a registered scene by name, or nil.
func (m *Manager) SceneByName(name string) Scene { //nolint:ireturn // returns interface by design
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.scenes[name]
}

func (m *Manager) Scene() Scene { //nolint:ireturn // manager returns interface by design
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.current
}
