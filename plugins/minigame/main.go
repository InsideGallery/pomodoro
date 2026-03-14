//go:build plugin

package main

import (
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

var Plugin pluggable.Module = &minigamePlugin{} //nolint:gochecknoglobals // plugin contract

type minigamePlugin struct{}

func (p *minigamePlugin) Name() string                 { return "minigame" }
func (p *minigamePlugin) ConfigKey() string            { return "minigame_enabled" }
func (p *minigamePlugin) DefaultEnabled() bool         { return false }
func (p *minigamePlugin) TrayItems() map[string]string { return nil }

func (p *minigamePlugin) Scenes(bus *event.Bus, switchScene pluggable.SceneSwitcher) []scene.Scene {
	return []scene.Scene{
		NewScene(bus, func() { switchScene("minigame") }, func() { switchScene("timer") }),
	}
}
