//go:build plugin

package main

import (
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

// Plugin is the exported symbol the loader looks for.
var Plugin pluggable.Module = &lockscreenPlugin{} //nolint:gochecknoglobals // plugin contract

type lockscreenPlugin struct{}

func (p *lockscreenPlugin) Name() string                 { return "lockscreen" }
func (p *lockscreenPlugin) ConfigKey() string            { return "lock_break_screen" }
func (p *lockscreenPlugin) DefaultEnabled() bool         { return false }
func (p *lockscreenPlugin) TrayItems() map[string]string { return nil }

func (p *lockscreenPlugin) Scenes(bus *event.Bus) []scene.Scene {
	return []scene.Scene{
		NewScene(bus, func() {}, func() {}),
	}
}
