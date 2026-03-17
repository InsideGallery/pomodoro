// Package mobile provides the Android entry point for Fingerprint Lab.
// Build: ebitenmobile bind -target android -javapkg com.insidegallery.fingerprint -o fingerprint.aar ./services/fingerprint/mobile/
package mobile

import (
	"context"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/InsideGallery/pomodoro/assets"
	"github.com/InsideGallery/pomodoro/pkg/platform"
	scenes "github.com/InsideGallery/pomodoro/services/fingerprint/internal/scenes"
)

func init() {
	// Set up embedded assets for mobile (no filesystem access)
	platform.SetAssetFS(assets.FingerprintTilesets)

	// Use a merged FS that combines all embedded asset dirs
	platform.SetAssetFS(mergedFS{})

	mobile.SetGame(&mobileGame{})
}

// SetStorageDir is called from Java/Kotlin to set writable storage path.
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

// mergedFS combines all embedded fingerprint asset filesystems.
type mergedFS struct{}

func (mergedFS) Open(name string) (fs.File, error) {
	// Try each embedded FS in order
	if f, err := assets.FingerprintImages.Open(name); err == nil {
		return f, nil
	}

	if f, err := assets.FingerprintAvatars.Open(name); err == nil {
		return f, nil
	}

	if f, err := assets.FingerprintUI.Open(name); err == nil {
		return f, nil
	}

	if f, err := assets.FingerprintTilesets.Open(name); err == nil {
		return f, nil
	}

	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
