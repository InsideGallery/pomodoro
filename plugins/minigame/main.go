//go:build plugin

package main

import (
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	mg "github.com/InsideGallery/pomodoro/pkg/plugins/minigame"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

var Plugin pluggable.Module = &wrapper{} //nolint:gochecknoglobals // plugin contract

type wrapper struct{}

func (w *wrapper) Name() string                 { return "minigame" }
func (w *wrapper) ConfigKey() string            { return "minigame_enabled" }
func (w *wrapper) DefaultEnabled() bool         { return false }
func (w *wrapper) TrayItems() map[string]string { return nil }

func (w *wrapper) Scenes(bus *event.Bus, switchScene pluggable.SceneSwitcher) []scene.Scene {
	return []scene.Scene{
		mg.NewScene(bus, func() { switchScene("minigame") }, func() { switchScene("timer") }),
	}
}
