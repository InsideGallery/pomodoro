package systems

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/internal/timer"
	"github.com/InsideGallery/pomodoro/internal/ui"
)

// RenderSystem draws the timer UI. It wraps the existing ui.TimerScreen
// for rendering, keeping visual output identical while the architecture is ECS.
type RenderSystem struct {
	Screen *ui.TimerScreen
	Tmr    *timer.Timer
}

func (s *RenderSystem) Update(_ context.Context) error {
	s.Screen.Update()

	return nil
}

func (s *RenderSystem) Draw(_ context.Context, screen *ebiten.Image) {
	s.Screen.Draw(screen)
}
