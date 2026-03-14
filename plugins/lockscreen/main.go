//go:build plugin

package main

import (
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	ls "github.com/InsideGallery/pomodoro/pkg/plugins/lockscreen"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

var Plugin pluggable.Module = &wrapper{} //nolint:gochecknoglobals // plugin contract

type wrapper struct{}

func (w *wrapper) Name() string                 { return "lockscreen" }
func (w *wrapper) ConfigKey() string            { return "lock_break_screen" }
func (w *wrapper) DefaultEnabled() bool         { return false }
func (w *wrapper) TrayItems() map[string]string { return nil }

func (w *wrapper) Scenes(bus *event.Bus, switchScene pluggable.SceneSwitcher) []scene.Scene {
	return []scene.Scene{
		ls.NewScene(bus, func() { switchScene("lockscreen") }, func() { switchScene("timer") }),
	}
}
