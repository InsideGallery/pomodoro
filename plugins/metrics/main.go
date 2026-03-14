//go:build plugin

package main

import (
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	mt "github.com/InsideGallery/pomodoro/pkg/plugins/metrics"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

var Plugin pluggable.Module = &wrapper{} //nolint:gochecknoglobals // plugin contract

type wrapper struct{}

func (w *wrapper) Name() string                 { return "metrics" }
func (w *wrapper) ConfigKey() string            { return "metrics_enabled" }
func (w *wrapper) DefaultEnabled() bool         { return false }
func (w *wrapper) TrayItems() map[string]string { return map[string]string{"Metrics": "metrics"} }

func (w *wrapper) Scenes(bus *event.Bus, switchScene pluggable.SceneSwitcher) []scene.Scene {
	return []scene.Scene{
		mt.NewScene(bus, func() { switchScene("timer") }),
	}
}
