package scene

import (
	"context"

	"github.com/InsideGallery/core/memory/registry"
	"github.com/InsideGallery/game-core/rtree"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/internal/core"
	"github.com/InsideGallery/pomodoro/internal/event"
)

// BaseScene provides shared ECS infrastructure for all scenes.
// Embed this in concrete scenes to get Systems, Registry, RTree, and Bus.
type BaseScene struct {
	Ctx      context.Context
	Systems  *core.Systems
	Registry *registry.Registry[string, uint64, any]
	RTree    *rtree.RTree
	Bus      *event.Bus
}

// NewBaseScene creates a BaseScene with all ECS infrastructure initialized.
func NewBaseScene(ctx context.Context, bus *event.Bus) *BaseScene {
	return &BaseScene{
		Ctx:      ctx,
		Systems:  core.NewSystems(),
		Registry: registry.NewRegistry[string, uint64, any](),
		RTree:    rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption),
		Bus:      bus,
	}
}

// Update iterates all systems in order, calling Update(ctx).
func (b *BaseScene) Update() error {
	for _, sys := range b.Systems.Get() {
		if err := sys.Update(b.Ctx); err != nil {
			return err
		}
	}

	return nil
}

// Draw iterates all systems, calling Draw(ctx, screen),
// then calls ScreenDraw() on SystemWindow systems for UI overlays.
func (b *BaseScene) Draw(screen *ebiten.Image) {
	systems := b.Systems.Get()

	for _, sys := range systems {
		sys.Draw(b.Ctx, screen)
	}

	for _, sys := range systems {
		if w, ok := sys.(core.SystemWindow); ok {
			w.ScreenDraw(b.Ctx, screen)
		}
	}
}
