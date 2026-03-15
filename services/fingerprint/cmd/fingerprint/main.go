package main

import (
	"context"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/app"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	scenes "github.com/InsideGallery/pomodoro/services/fingerprint/internal/scenes"
)

func main() {
	game := app.New(app.Config{
		Width:     1920,
		Height:    1080,
		Title:     "Fingerprint Lab",
		Decorated: true,
		Setup:     setup,
	})

	ebiten.SetWindowTitle("Fingerprint Lab")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetFullscreen(true)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

func setup(ctx context.Context, _ *event.Bus, manager *scene.Manager, _ func(string)) string {
	// Single TMX-driven scene with state machine
	game := scenes.NewGameScene()
	manager.Add(ctx, game)

	return scenes.GameSceneName
}
