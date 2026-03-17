package fsystems

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/core"
	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
)

// CameraSystem handles zoom via keyboard (desktop) or pinch (mobile).
// Only active during StateApplicationNet.
type CameraSystem struct {
	scene SceneAccessor
}

func NewCameraSystem(scene SceneAccessor) *CameraSystem {
	return &CameraSystem{scene: scene}
}

func (s *CameraSystem) Update(_ context.Context) error {
	reg := s.scene.GetRegistry()

	// Only zoom in puzzle workspace
	val, err := reg.Get(c.GroupGameState, 0)
	if err != nil {
		return nil
	}

	entity, ok := val.(*c.Entity)
	if !ok || entity.State == nil || entity.State.Current != c.StateApplicationNet {
		return nil
	}

	cam := s.scene.GetCamera()
	if cam == nil {
		return nil
	}

	// Set viewport
	w, h := s.scene.GetScreenSize()
	cam.SetViewPort(float64(w), float64(h))

	// Desktop: Z = zoom in, X = zoom out, C = reset
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		cam.ZoomFactor += 5
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		cam.ZoomFactor -= 5
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		cam.Reset()
	}

	// Mobile: pinch zoom (two-finger distance change)
	touches := ebiten.AppendTouchIDs(nil)
	if len(touches) >= 2 {
		s.handlePinchZoom(cam, touches)
	}

	return nil
}

func (s *CameraSystem) Draw(_ context.Context, _ *ebiten.Image) {}

func (s *CameraSystem) handlePinchZoom(cam *core.Camera, touches []ebiten.TouchID) {
	// TODO: implement pinch zoom for mobile
	_ = cam
	_ = touches
}
