package systems

import (
	"context"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// DebugSystem displays FPS/TPS overlay. Implements SystemWindow.
type DebugSystem struct {
	enabled bool
}

func NewDebugSystem(enabled bool) *DebugSystem {
	return &DebugSystem{enabled: enabled}
}

func (d *DebugSystem) SetEnabled(v bool) { d.enabled = v }

func (d *DebugSystem) Update(_ context.Context) error { return nil }

func (d *DebugSystem) Draw(_ context.Context, _ *ebiten.Image) {}

// ScreenDraw renders FPS/TPS in screen space (not world space).
func (d *DebugSystem) ScreenDraw(_ context.Context, screen *ebiten.Image) {
	if !d.enabled {
		return
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %.0f TPS: %.0f", ebiten.ActualFPS(), ebiten.ActualTPS()))
}
