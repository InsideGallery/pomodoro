package fingerprint

import (
	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

// Plugin implements pluggable.Module for the fingerprint puzzle.
type Plugin struct{}

func (p *Plugin) Name() string                 { return "fingerprint" }
func (p *Plugin) ConfigKey() string            { return "fingerprint_enabled" }
func (p *Plugin) DefaultEnabled() bool         { return false }
func (p *Plugin) TrayItems() map[string]string { return nil }

func (p *Plugin) Scenes(bus *event.Bus, switchScene pluggable.SceneSwitcher) []scene.Scene {
	cfg := config.Load()

	puzzle := NewPuzzleScene(
		func(name string) { switchScene(name) },
		cfg.BreakDuration(),
	)

	// Self-activate on break start (if enabled)
	bus.Subscribe(event.BreakStarted, func(_ event.Event) {
		if cfg.PluginEnabled("fingerprint_enabled", false) {
			switchScene(PuzzleSceneName)
		}
	})

	bus.Subscribe(event.ConfigChanged, func(e event.Event) {
		if c, ok := e.Data.(config.Config); ok {
			cfg = c
		}
	})

	return []scene.Scene{puzzle}
}
