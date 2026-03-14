//go:build plugin

package main

import (
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

// Plugin is the exported symbol the loader looks for.
var Plugin pluggable.Module = &minigamePlugin{} //nolint:gochecknoglobals // plugin contract

type minigamePlugin struct{}

func (p *minigamePlugin) Name() string                 { return "minigame" }
func (p *minigamePlugin) ConfigKey() string            { return "minigame_enabled" }
func (p *minigamePlugin) DefaultEnabled() bool         { return false }
func (p *minigamePlugin) TrayItems() map[string]string { return nil }

func (p *minigamePlugin) Scenes(bus *event.Bus) []scene.Scene {
	switchToSelf := func() {
		// Plugin scenes activate themselves via event subscription.
		// The scene switching is handled internally.
	}

	return []scene.Scene{
		NewScene(bus, switchToSelf, func() {}),
	}
}
