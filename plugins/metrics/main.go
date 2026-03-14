//go:build plugin

package main

import (
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

// Plugin is the exported symbol the loader looks for.
var Plugin pluggable.Module = &metricsPlugin{} //nolint:gochecknoglobals // plugin contract

type metricsPlugin struct{}

func (p *metricsPlugin) Name() string                 { return "metrics" }
func (p *metricsPlugin) ConfigKey() string            { return "metrics_enabled" }
func (p *metricsPlugin) DefaultEnabled() bool         { return false }
func (p *metricsPlugin) TrayItems() map[string]string { return map[string]string{"Metrics": "metrics"} }

func (p *metricsPlugin) Scenes(bus *event.Bus) []scene.Scene {
	return []scene.Scene{
		NewScene(bus, func() {}),
	}
}
