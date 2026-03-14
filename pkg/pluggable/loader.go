package pluggable

import (
	"fmt"
	"os"
	"path/filepath"
	goplugin "plugin"
	"strings"

	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

// Loader discovers and loads plugin .so files.
type Loader struct {
	pluginDir string
	modules   []Module
}

// NewLoader creates a loader that scans the given directory for .so files.
func NewLoader(pluginDir string) *Loader {
	return &Loader{pluginDir: pluginDir}
}

// DefaultPluginDir returns ~/.config/pomodoro/plugins/
func DefaultPluginDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", "pomodoro", "plugins")
}

// Load scans the plugin directory and loads all .so files.
// Each .so must export a `Plugin` symbol implementing Module.
func (l *Loader) Load() error {
	entries, err := os.ReadDir(l.pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no plugins dir = no plugins, not an error
		}

		return fmt.Errorf("read plugin dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".so") {
			continue
		}

		path := filepath.Join(l.pluginDir, entry.Name())

		mod, err := loadPlugin(path)
		if err != nil {
			continue // skip broken plugins silently
		}

		l.modules = append(l.modules, mod)
	}

	return nil
}

// Modules returns all successfully loaded plugin modules.
func (l *Loader) Modules() []Module {
	return l.modules
}

// RegisterAll registers all plugin scenes with the SceneManager.
func (l *Loader) RegisterAll(bus *event.Bus, manager *scene.Manager, switchScene SceneSwitcher) {
	for _, mod := range l.modules {
		for _, sc := range mod.Scenes(bus, switchScene) {
			manager.Add(nil, sc) //nolint:staticcheck // nil ctx OK, scenes handle it
		}
	}
}

func loadPlugin(path string) (Module, error) { //nolint:ireturn // returns interface by design
	p, err := goplugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open plugin %s: %w", path, err)
	}

	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("lookup Plugin in %s: %w", path, err)
	}

	mod, ok := sym.(*Module)
	if !ok {
		return nil, fmt.Errorf("plugin symbol in %s is not *plugin.Module", path)
	}

	return *mod, nil
}
