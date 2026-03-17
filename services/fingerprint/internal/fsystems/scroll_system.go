package fsystems

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/tilemap"
	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
)

// ScrollSystem handles mouse wheel scrolling for scrollable areas.
// Reads/writes GameData directly from Registry.
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

	co := CoordsFromScene(s.scene)
	cur := GetCursor(reg)

	if cur == nil {
		return nil
	}

	fcx, fcy := float64(cur.X), float64(cur.Y)

	// Check cases list area
	if obj := tilemap.FindObject(og, "list-of-cases"); obj != nil {
		sx, sy, sw, sh := co.MapRect(obj.X, obj.Y, obj.Width, obj.Height)

		if fcx >= sx && fcx <= sx+sw && fcy >= sy && fcy <= sy+sh {
			gd.CasesScroll += scrollDelta(wy)
			clampScroll(&gd.CasesScroll, len(gd.Cases), sh, co.MapToScreenSize(45))

			return nil
		}
	}

	// Check description area
	if obj := tilemap.FindObject(og, "description"); obj != nil {
		sx, sy, sw, sh := co.MapRect(obj.X, obj.Y, obj.Width, obj.Height)

		if fcx >= sx && fcx <= sx+sw && fcy >= sy && fcy <= sy+sh {
			gd.DescScroll += scrollDelta(wy)

			if gd.DescScroll < 0 {
				gd.DescScroll = 0
			}

			return nil
		}
	}

	// Default: scroll names list
	gd.NamesScroll += scrollDelta(wy)

	if gd.SelectedCase >= 0 && gd.SelectedCase < len(gd.Cases) {
		if obj := tilemap.FindObject(og, "fingerprints-user-names"); obj != nil {
			sh := co.MapToScreenSize(obj.Height)
			clampScroll(&gd.NamesScroll, len(gd.Cases[gd.SelectedCase].Puzzles), sh, co.MapToScreenSize(50))
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
