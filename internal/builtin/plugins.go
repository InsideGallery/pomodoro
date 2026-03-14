// Package builtin provides compiled-in versions of all plugins.
// Used on Windows (no .so support) and for full builds (make build-full).
package builtin

import (
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	ls "github.com/InsideGallery/pomodoro/pkg/plugins/lockscreen"
	mt "github.com/InsideGallery/pomodoro/pkg/plugins/metrics"
	mg "github.com/InsideGallery/pomodoro/pkg/plugins/minigame"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

// Modules returns all built-in plugin modules.
func Modules() []pluggable.Module {
	return []pluggable.Module{
		&minigamePlugin{},
		&lockscreenPlugin{},
		&metricsPlugin{},
	}
}

type minigamePlugin struct{}

func (p *minigamePlugin) Name() string                 { return "minigame" }
func (p *minigamePlugin) ConfigKey() string            { return "minigame_enabled" }
func (p *minigamePlugin) DefaultEnabled() bool         { return false }
func (p *minigamePlugin) TrayItems() map[string]string { return nil }

func (p *minigamePlugin) Scenes(bus *event.Bus, sw pluggable.SceneSwitcher) []scene.Scene {
	return []scene.Scene{mg.NewScene(bus, func() { sw("minigame") }, func() { sw("timer") })}
}

type lockscreenPlugin struct{}

func (p *lockscreenPlugin) Name() string                 { return "lockscreen" }
func (p *lockscreenPlugin) ConfigKey() string            { return "lock_break_screen" }
func (p *lockscreenPlugin) DefaultEnabled() bool         { return false }
func (p *lockscreenPlugin) TrayItems() map[string]string { return nil }

func (p *lockscreenPlugin) Scenes(bus *event.Bus, sw pluggable.SceneSwitcher) []scene.Scene {
	return []scene.Scene{ls.NewScene(bus, func() { sw("lockscreen") }, func() { sw("timer") })}
}

type metricsPlugin struct{}

func (p *metricsPlugin) Name() string                 { return "metrics" }
func (p *metricsPlugin) ConfigKey() string            { return "metrics_enabled" }
func (p *metricsPlugin) DefaultEnabled() bool         { return false }
func (p *metricsPlugin) TrayItems() map[string]string { return map[string]string{"Metrics": "metrics"} }

func (p *metricsPlugin) Scenes(bus *event.Bus, sw pluggable.SceneSwitcher) []scene.Scene {
	return []scene.Scene{mt.NewScene(bus, func() { sw("timer") })}
}
