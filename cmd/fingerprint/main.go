package main

import (
	"context"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/internal/app"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint"
	"github.com/InsideGallery/pomodoro/pkg/resources"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

func main() {
	game := app.New(app.Config{
		Width:     1920,
		Height:    1080,
		Title:     "Fingerprint Lab",
		Decorated: true,
		Setup:     setupFingerprint,
	})

	ebiten.SetWindowTitle("Fingerprint Lab")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetFullscreen(true)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

func setupFingerprint(ctx context.Context, _ *event.Bus, manager *scene.Manager, switchScene func(string)) string {
	// Scenes: loading → desktop → puzzle
	desktop := fingerprint.NewDesktopScene(switchScene)
	puzzle := fingerprint.NewPuzzleScene(switchScene, 0) // no break timer — standalone game

	loading := fingerprint.NewLoadingScene(
		switchScene,
		fingerprint.DesktopSceneName,
		func(base *scene.BaseScene) {
			loadFingerprintResources(base.Resources)
		},
	)

	manager.Add(ctx, loading, desktop, puzzle)

	return fingerprint.LoadingSceneName
}

func loadFingerprintResources(rm *resources.Manager) {
	// Delegate to the fingerprint package's resource loader
	fingerprint.LoadResources(rm)
}
