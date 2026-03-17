package scene

import (
	"context"

	"github.com/InsideGallery/core/memory/registry"
	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/InsideGallery/game-core/rtree"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/core"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/resources"
)

// BaseScene provides shared ECS infrastructure for all scenes.
// Embed this in concrete scenes to get Systems, Registry, RTree, Bus, Camera, Resources.
type BaseScene struct {
	Ctx       context.Context
	Systems   *core.Systems
	Registry  *registry.Registry[string, uint64, any]
	RTree     *rtree.RTree
	Bus       *event.Bus
	Camera    *core.Camera
	Resources *resources.Manager
	World     *ebiten.Image // offscreen World — systems draw here in world coords
}

// NewBaseScene creates a BaseScene with all ECS infrastructure initialized.
func NewBaseScene(ctx context.Context, bus *event.Bus) *BaseScene {
	return &BaseScene{
		Ctx:       ctx,
		Systems:   core.NewSystems(),
		Registry:  registry.NewRegistry[string, uint64, any](),
		RTree:     rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption),
		Bus:       bus,
		Camera:    core.NewCamera(shapes.NewPoint()),
		Resources: resources.NewManager(),
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

// Draw: systems draw to World (world coords), then World composited to screen
// via Camera.WorldMatrix(). ScreenDraw overlays go directly to screen.
// If World is nil, draws directly to screen (no camera transform).
func (b *BaseScene) Draw(screen *ebiten.Image) {
	systems := b.Systems.Get()

	if b.World != nil {
		b.World.Clear()

		for _, sys := range systems {
			sys.Draw(b.Ctx, b.World)
		}

		screen.DrawImage(b.World, &ebiten.DrawImageOptions{
			GeoM: b.Camera.WorldMatrix(),
		})
	} else {
		for _, sys := range systems {
			sys.Draw(b.Ctx, screen)
		}
	}

	// UI overlays in screen space (cursor, debug)
	for _, sys := range systems {
		if w, ok := sys.(core.SystemWindow); ok {
			w.ScreenDraw(b.Ctx, screen)
		}
	}
}
