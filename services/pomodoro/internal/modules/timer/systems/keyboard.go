package systems

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// KeyboardSystem handles keyboard shortcuts for the timer scene.
type KeyboardSystem struct {
	OnStartPause func()
	OnReset      func()
	OnSettings   func()
}

func (s *KeyboardSystem) Update(_ context.Context) error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && s.OnStartPause != nil {
		s.OnStartPause()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) && s.OnReset != nil {
		s.OnReset()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyS) && s.OnSettings != nil {
		s.OnSettings()
	}

	return nil
}

func (s *KeyboardSystem) Draw(_ context.Context, _ *ebiten.Image) {}
