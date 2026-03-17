// Package mobile provides the Android entry point for Fingerprint Lab.
// Build: ebitenmobile bind -target android -javapkg com.insidegallery.fingerprint ./services/fingerprint/mobile/
package mobile

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/InsideGallery/pomodoro/pkg/platform"
	scenes "github.com/InsideGallery/pomodoro/services/fingerprint/internal/scenes"
)

func init() {
	mobile.SetGame(&mobileGame{})
}

// SetStorageDir is called from Java/Kotlin to set the app's internal storage path.
func SetStorageDir(dir string) {
	platform.SetDataDir(dir)
}

type mobileGame struct {
	game *scenes.GameScene
	init bool
}

func (g *mobileGame) Update() error {
	if !g.init {
		g.init = true
		g.game = scenes.NewGameScene()
		g.game.Init(context.Background())

		if err := g.game.Load(); err != nil {
			return err
		}
	}

	return g.game.Update()
}

func (g *mobileGame) Draw(screen *ebiten.Image) {
	if g.game != nil {
		g.game.Draw(screen)
	}
}

func (g *mobileGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	if g.game != nil {
		return g.game.Layout(outsideWidth, outsideHeight)
	}

	return outsideWidth, outsideHeight
}
