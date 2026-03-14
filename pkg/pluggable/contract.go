package pluggable

import (
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

// SceneSwitcher switches to a named scene. Provided by the host app.
type SceneSwitcher func(name string)

// Module is the contract every plugin must satisfy.
// External plugins implement this interface and export it as `var Plugin Module`.
type Module interface {
	// Name returns the unique plugin identifier (used for config keys, logging).
	Name() string

	// Scenes returns the scenes this plugin provides.
	// bus: for event subscriptions.
	// switchScene: for switching to any scene by name (including own scenes).
	Scenes(bus *event.Bus, switchScene SceneSwitcher) []scene.Scene

	// TrayItems returns optional tray menu items.
	// Key = display label, Value = scene name to switch to when clicked.
	TrayItems() map[string]string

	// ConfigKey returns the config key for the enable/disable toggle.
	ConfigKey() string

	// DefaultEnabled returns whether the plugin is enabled by default.
	DefaultEnabled() bool
}
