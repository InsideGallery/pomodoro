package fsystems

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"

	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
)

// CursorSystem updates the virtual cursor using delta-based movement.
// Clamps to cursor-room bounds to prevent stickiness.
type CursorSystem struct {
	scene SceneAccessor
}

func NewCursorSystem(scene SceneAccessor) *CursorSystem {
	return &CursorSystem{scene: scene}
}

func (s *CursorSystem) Update(_ context.Context) error {
	reg := s.scene.GetRegistry()

	val, err := reg.Get(c.GroupCursor, 0)
	if err != nil {
		return nil
	}

	entity, ok := val.(*c.Entity)
	if !ok || entity.Cursor == nil {
		return nil
	}

	cur := entity.Cursor
	rawX, rawY := s.readRawInput()

	if !cur.Inited {
		// Init to screen center, not raw position (which may be 0,0 on first frames)
		w, h := s.scene.GetScreenSize()
		cur.X = w / 2
		cur.Y = h / 2
		cur.PrevRawX = rawX
		cur.PrevRawY = rawY
		cur.Inited = true

		s.scene.SetCursorPos(cur.X, cur.Y)

		input := s.scene.GetInputSystem()
		if input != nil {
			input.CursorOverride = &[2]int{cur.X, cur.Y}
		}

		return nil
	}

	dx := rawX - cur.PrevRawX
	dy := rawY - cur.PrevRawY
	cur.PrevRawX = rawX
	cur.PrevRawY = rawY

	cur.X += dx
	cur.Y += dy

	// Clamp to room bounds
	if cur.X < cur.RoomMinX {
		cur.X = cur.RoomMinX
	}

	if cur.X > cur.RoomMaxX {
		cur.X = cur.RoomMaxX
	}

	if cur.Y < cur.RoomMinY {
		cur.Y = cur.RoomMinY
	}

	if cur.Y > cur.RoomMaxY {
		cur.Y = cur.RoomMaxY
	}

	// Feed to InputSystem and scene
	input := s.scene.GetInputSystem()
	if input != nil {
		input.CursorOverride = &[2]int{cur.X, cur.Y}
	}

	s.scene.SetCursorPos(cur.X, cur.Y)

	return nil
}

func (s *CursorSystem) Draw(_ context.Context, _ *ebiten.Image) {}

// readRawInput returns the raw cursor position from the OS.
// On desktop: mouse position. On mobile: touch position (see build-tagged override).
func (s *CursorSystem) readRawInput() (int, int) {
	// Touch takes priority if available
	touches := ebiten.AppendTouchIDs(nil)
	if len(touches) > 0 {
		return ebiten.TouchPosition(touches[0])
	}

	return ebiten.CursorPosition()
}
