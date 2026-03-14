//go:build plugin

// Example plugin demonstrating the plugin contract.
// Build: go build -buildmode=plugin -o example.so ./plugins/example/
// Install: cp example.so ~/.config/pomodoro/plugins/
package main

import (
	"context"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

// Plugin is the exported symbol that the loader looks for.
var Plugin pluggable.Module = &examplePlugin{} //nolint:gochecknoglobals // required by plugin contract

type examplePlugin struct{}

func (p *examplePlugin) Name() string                 { return "example" }
func (p *examplePlugin) ConfigKey() string            { return "example_enabled" }
func (p *examplePlugin) DefaultEnabled() bool         { return false }
func (p *examplePlugin) TrayItems() map[string]string { return nil }

func (p *examplePlugin) Scenes(bus *event.Bus) []scene.Scene {
	return []scene.Scene{&exampleScene{bus: bus}}
}

// exampleScene is a minimal scene that shows a colored screen with instructions.
type exampleScene struct {
	*scene.BaseScene
	bus           *event.Bus
	width, height int
}

func (s *exampleScene) Name() string { return "example" }

func (s *exampleScene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, s.bus)
}

func (s *exampleScene) Load() error   { return nil }
func (s *exampleScene) Unload() error { return nil }

func (s *exampleScene) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		// Plugins don't know how to switch scenes directly.
		// They publish an event; the host app handles routing.
		s.bus.Publish(event.Event{Type: event.Reset})
	}

	return nil
}

func (s *exampleScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0x2D, G: 0x1B, B: 0x69, A: 0xFF})
}

func (s *exampleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	w := int(math.Ceil(float64(outsideWidth)))
	h := int(math.Ceil(float64(outsideHeight)))
	s.width = w
	s.height = h

	return w, h
}
