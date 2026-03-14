package pluggable

import (
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

// Module is the contract every plugin must satisfy.
// External plugins implement this interface and export it as `var Plugin Module`.
type Module interface {
	// Name returns the unique plugin identifier (used for config keys, logging).
	Name() string

	// Scenes returns the scenes this plugin provides.
	// Each scene is registered with the SceneManager.
	// The bus is provided for event subscriptions.
	Scenes(bus *event.Bus) []scene.Scene

	// TrayItems returns optional tray menu items.
	// Key = display label, Value = scene name to switch to when clicked.
	// Return nil if the plugin doesn't need tray items.
	TrayItems() map[string]string

	// ConfigKey returns the config key for the enable/disable toggle.
	// The settings screen auto-generates a toggle for each plugin.
	// Example: "minigame_enabled" — stored in config.json.
	ConfigKey() string

	// DefaultEnabled returns whether the plugin is enabled by default.
	DefaultEnabled() bool
}
