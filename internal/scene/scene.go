package scene

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"
)

// Scene is a self-contained screen with its own ECS.
type Scene interface {
	Name() string
	Init(ctx context.Context)
	Load() error
	Unload() error
	Update() error
	Draw(screen *ebiten.Image)
	Layout(outsideWidth, outsideHeight int) (int, int)
}
