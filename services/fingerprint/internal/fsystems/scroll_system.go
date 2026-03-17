package fsystems

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/tilemap"
	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
)

// ScrollSystem handles mouse wheel scrolling. Converts cursor to world coords.
type ScrollSystem struct {
	scene SceneAccessor
}

func NewScrollSystem(scene SceneAccessor) *ScrollSystem {
	return &ScrollSystem{scene: scene}
}

func (s *ScrollSystem) Update(_ context.Context) error {
	reg := s.scene.GetRegistry()
	state := GetState(reg)

	if state == nil || state.Current != c.StateApplicationLayout {
		return nil
	}

	_, wy := ebiten.Wheel()
	if wy == 0 {
		return nil
	}

	gd := GetGameData(reg)
	if gd == nil {
		return nil
	}

	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return nil
	}

	og := tmap.FindObjectGroup("application-layout")
	if og == nil {
		return nil
	}

	// Convert screen cursor to world coords
	cur := GetCursor(reg)
	if cur == nil {
		return nil
	}

	wx, worldY := s.scene.ScreenToWorld(float64(cur.X), float64(cur.Y))

	// Check cases list
	if obj := tilemap.FindObject(og, "list-of-cases"); obj != nil {
		if wx >= obj.X && wx <= obj.X+obj.Width && worldY >= obj.Y && worldY <= obj.Y+obj.Height {
			gd.CasesScroll += scrollDelta(wy)
			clampScroll(&gd.CasesScroll, len(gd.Cases), obj.Height, 90)

			return nil
		}
	}

	// Check description
	if obj := tilemap.FindObject(og, "description"); obj != nil {
		if wx >= obj.X && wx <= obj.X+obj.Width && worldY >= obj.Y && worldY <= obj.Y+obj.Height {
			gd.DescScroll += scrollDelta(wy)

			if gd.DescScroll < 0 {
				gd.DescScroll = 0
			}

			return nil
		}
	}

	// Default: scroll names
	gd.NamesScroll += scrollDelta(wy)

	if gd.SelectedCase >= 0 && gd.SelectedCase < len(gd.Cases) {
		if obj := tilemap.FindObject(og, "fingerprints-user-names"); obj != nil {
			clampScroll(&gd.NamesScroll, len(gd.Cases[gd.SelectedCase].Puzzles), obj.Height, 100)
		}
	}

	return nil
}

func (s *ScrollSystem) Draw(_ context.Context, _ *ebiten.Image) {}

func scrollDelta(wy float64) int {
	if wy > 0 {
		return -1
	}

	return 1
}

func clampScroll(scroll *int, totalItems int, areaH, rowH float64) {
	if *scroll < 0 {
		*scroll = 0
	}

	maxScroll := totalItems - int(areaH/rowH)
	if maxScroll < 0 {
		maxScroll = 0
	}

	if *scroll > maxScroll {
		*scroll = maxScroll
	}
}
