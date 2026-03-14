package systems

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/ui"
)

// ScrollSystem manages scroll state for the settings scene.
type ScrollSystem struct {
	ScrollY    float32
	ContentH   float32
	ViewportH  float32
	ContentTop float32 // Y where scrollable area starts
}

func (s *ScrollSystem) Update(_ context.Context) error {
	_, wy := ebiten.Wheel()
	if wy != 0 {
		s.ScrollY -= float32(wy) * ui.S(30)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		s.ScrollY += ui.S(30)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		s.ScrollY -= ui.S(30)
	}

	if s.ScrollY < 0 {
		s.ScrollY = 0
	}

	if mx := s.maxScroll(); s.ScrollY > mx {
		s.ScrollY = mx
	}

	return nil
}

func (s *ScrollSystem) Draw(_ context.Context, _ *ebiten.Image) {}

// Offset returns the scroll offset for InputSystem (contentTop - scrollY).
func (s *ScrollSystem) Offset() float64 {
	return float64(s.ContentTop - s.ScrollY)
}

func (s *ScrollSystem) maxScroll() float32 {
	m := s.ContentH - s.ViewportH

	if m < 0 {
		return 0
	}

	return m
}
