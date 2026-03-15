package main

import (
	"context"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/app"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/resources"
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

func setup(ctx context.Context, _ *event.Bus, manager *scene.Manager, switchScene func(string)) string {
	shared := resources.NewManager()

	loading := scenes.NewLoadingScene(switchScene, scenes.DesktopSceneName,
		func(_ *scene.BaseScene) { scenes.LoadResources(shared) })
	loading.SetResources(shared)

	desktop := scenes.NewDesktopScene(switchScene)
	desktop.SetResources(shared)

	appScene := scenes.NewAppScene(switchScene)
	appScene.SetResources(shared)

	manager.Add(ctx, loading, desktop, appScene)

	return scenes.LoadingSceneName
}
