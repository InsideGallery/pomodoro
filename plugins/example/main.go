//go:build plugin

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

var Plugin pluggable.Module = &examplePlugin{} //nolint:gochecknoglobals // plugin contract

type examplePlugin struct{}

func (p *examplePlugin) Name() string                 { return "example" }
func (p *examplePlugin) ConfigKey() string            { return "example_enabled" }
func (p *examplePlugin) DefaultEnabled() bool         { return false }
func (p *examplePlugin) TrayItems() map[string]string { return nil }

func (p *examplePlugin) Scenes(bus *event.Bus, switchScene pluggable.SceneSwitcher) []scene.Scene {
	return []scene.Scene{&exampleScene{bus: bus, switchScene: switchScene}}
}

type exampleScene struct {
	*scene.BaseScene
	bus           *event.Bus
	switchScene   pluggable.SceneSwitcher
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
		s.switchScene("timer")
	}

	return nil
}

func (s *exampleScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0x2D, G: 0x1B, B: 0x69, A: 0xFF})
}

func (s *exampleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	s.width = outsideWidth
	s.height = outsideHeight

	return int(math.Ceil(float64(outsideWidth))), int(math.Ceil(float64(outsideHeight)))
}
