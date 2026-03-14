package main

import (
	"context"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/app"
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
	// Shared resource manager so loading scene's assets are visible to all scenes
	shared := resources.NewManager()

	loading := fingerprint.NewLoadingScene(switchScene, fingerprint.DesktopSceneName,
		func(_ *scene.BaseScene) { fingerprint.LoadResources(shared) })
	desktop := fingerprint.NewDesktopScene(switchScene)
	puzzle := fingerprint.NewPuzzleScene(switchScene, 0)

	// Wrap scenes with shared resources injected after Init
	manager.Add(ctx,
		&sharedResourceScene{Scene: loading, res: shared},
		&sharedResourceScene{Scene: desktop, res: shared},
		&sharedResourceScene{Scene: puzzle, res: shared},
	)

	return fingerprint.LoadingSceneName
}

// sharedResourceScene wraps a scene and injects a shared Resources after Init.
type sharedResourceScene struct {
	scene.Scene
	res *resources.Manager
}

func (s *sharedResourceScene) Init(ctx context.Context) {
	s.Scene.Init(ctx)

	// Replace the per-scene Resources with the shared one
	if bs, ok := s.Scene.(interface{ SetResources(*resources.Manager) }); ok {
		bs.SetResources(s.res)
	}
}
