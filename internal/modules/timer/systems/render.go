package systems

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/internal/timer"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

// RenderSystem draws the timer UI. It wraps the existing ui.TimerScreen
// for rendering, keeping visual output identical while the architecture is ECS.
type RenderSystem struct {
	Screen *ui.TimerScreen
	Tmr    *timer.Timer
}

// Update only updates the start button label/color based on timer state.
// Hit detection is handled by InputSystem, not widget self-detection.
func (s *RenderSystem) Update(_ context.Context) error {
	s.Screen.UpdateStartButton()

	return nil
}

func (s *RenderSystem) Draw(_ context.Context, screen *ebiten.Image) {
	s.Screen.Draw(screen)
}
