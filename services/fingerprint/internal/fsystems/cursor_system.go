package fsystems

import (
	"context"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// CursorSystem updates cursor screen position from mouse/touch input.
// No clamping — in fullscreen the screen edge IS the boundary.
// World-space clamping happens in ScreenToWorld if needed.
type CursorSystem struct {
	scene SceneAccessor
}

func NewCursorSystem(scene SceneAccessor) *CursorSystem {
	return &CursorSystem{scene: scene}
}

func (s *CursorSystem) Update(_ context.Context) error {
	cur := GetCursor(s.scene.GetRegistry())
	if cur == nil {
		return nil
	}

	rawX, rawY := s.readRawInput()
	cur.X = rawX
	cur.Y = rawY

	// InputSystem zones are in base-transform screen coords (no camera zoom).
	// Undo camera zoom from raw cursor so clicks match zone positions.
	inputX, inputY := rawX, rawY
	cam := s.scene.GetCamera()

	if cam != nil && cam.ZoomFactor != 0 {
		sw, sh := s.scene.GetScreenSize()
		cx := float64(sw) / 2
		cy := float64(sh) / 2
		zoomScale := math.Pow(1.01, cam.ZoomFactor)
		inputX = int((float64(rawX)-cx)/zoomScale + cx)
		inputY = int((float64(rawY)-cy)/zoomScale + cy)
	}

	input := s.scene.GetInputSystem()
	if input != nil {
		input.CursorOverride = &[2]int{inputX, inputY}
	}

	s.scene.SetCursorPos(cur.X, cur.Y)

	return nil
}

func (s *CursorSystem) Draw(_ context.Context, _ *ebiten.Image) {}

func (s *CursorSystem) readRawInput() (int, int) {
	touches := ebiten.AppendTouchIDs(nil)
	if len(touches) > 0 {
		return ebiten.TouchPosition(touches[0])
	}

	return ebiten.CursorPosition()
}
