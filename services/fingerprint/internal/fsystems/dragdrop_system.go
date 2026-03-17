package fsystems

import (
	"context"
	"log/slog"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
)

// DragDropSystem handles picking up, rotating, and placing puzzle pieces.
// All coordinates in world (map) space. Screen cursor converted via Camera.
type DragDropSystem struct {
	scene SceneAccessor
}

func NewDragDropSystem(scene SceneAccessor) *DragDropSystem {
	return &DragDropSystem{scene: scene}
}

func (s *DragDropSystem) Update(_ context.Context) error {
	reg := s.scene.GetRegistry()
	state := GetState(reg)

	if state == nil || state.Current != c.StateApplicationNet {
		return nil
	}

	gd := GetGameData(reg)
	if gd == nil {
		return nil
	}

	puzzle := CurrentPuzzle(gd)
	if puzzle == nil {
		return nil
	}

	// Convert screen cursor to world coordinates
	cur := GetCursor(reg)
	if cur == nil {
		return nil
	}

	wx, wy := s.scene.ScreenToWorld(float64(cur.X), float64(cur.Y))

	// Mouse wheel: rotate held piece
	_, wheelY := ebiten.Wheel()
	if wheelY != 0 && gd.HoldingPiece >= 0 && gd.HoldingPiece < len(puzzle.TrayPieces) {
		if wheelY > 0 {
			puzzle.TrayPieces[gd.HoldingPiece].Rotation = (puzzle.TrayPieces[gd.HoldingPiece].Rotation + 1) % domain.RotationSteps
		} else {
			puzzle.TrayPieces[gd.HoldingPiece].Rotation = (puzzle.TrayPieces[gd.HoldingPiece].Rotation + domain.RotationSteps - 1) % domain.RotationSteps
		}
	}

	// Press → pickup (skip if over button zone)
	input := s.scene.GetInputSystem()
	overButton := input != nil && input.HasHoveredZone()

	if !overButton && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && gd.HoldingPiece < 0 {
		s.tryPickup(gd, puzzle, wx, wy)
	}

	if !overButton && gd.HoldingPiece < 0 {
		for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
			tx, ty := ebiten.TouchPosition(id)
			twx, twy := s.scene.ScreenToWorld(float64(tx), float64(ty))
			s.tryPickup(gd, puzzle, twx, twy)

			break
		}
	}

	// Release → place
	mouseRel := gd.Dragging && gd.HoldingPiece >= 0 && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	touchRel := gd.Dragging && gd.HoldingPiece >= 0 && len(inpututil.AppendJustReleasedTouchIDs(nil)) > 0

	if mouseRel || touchRel {
		s.tryPlace(gd, puzzle, wx, wy)
	}

	return nil
}

func (s *DragDropSystem) Draw(_ context.Context, _ *ebiten.Image) {}

func (s *DragDropSystem) HoldingPiece() int {
	gd := GetGameData(s.scene.GetRegistry())
	if gd == nil {
		return -1
	}

	return gd.HoldingPiece
}

func (s *DragDropSystem) IsDragging() bool {
	gd := GetGameData(s.scene.GetRegistry())

	return gd != nil && gd.Dragging
}

func (s *DragDropSystem) tryPickup(gd *c.GameData, puzzle *domain.PuzzleConfig, wx, wy float64) {
	cellSz := s.gridCellMapSize()

	// Check tray pieces (world coords)
	for i := range puzzle.TrayPieces {
		tp := &puzzle.TrayPieces[i]
		if tp.IsPlaced {
			continue
		}

		if wx >= tp.TrayX && wx <= tp.TrayX+cellSz && wy >= tp.TrayY && wy <= tp.TrayY+cellSz {
			gd.HoldingPiece = i
			gd.Dragging = true

			return
		}
	}

	// Check placed pieces on grid
	gpx, gpy, gcw, ok := s.puzzleGridWorld()
	if !ok {
		return
	}

	for ti := range puzzle.TrayPieces {
		tp := &puzzle.TrayPieces[ti]
		if !tp.IsPlaced {
			continue
		}

		gx := gpx + float64(tp.PlacedX)*gcw
		gy := gpy + float64(tp.PlacedY)*gcw

		if wx >= gx && wx <= gx+gcw && wy >= gy && wy <= gy+gcw {
			tp.IsPlaced = false
			tp.PlacedX = -1
			tp.PlacedY = -1
			gd.HoldingPiece = ti
			gd.Dragging = true

			slog.Info("picked up placed piece", "tray", ti)

			return
		}
	}
}

func (s *DragDropSystem) tryPlace(gd *c.GameData, puzzle *domain.PuzzleConfig, wx, wy float64) {
	tp := &puzzle.TrayPieces[gd.HoldingPiece]
	placed := false

	gpx, gpy, gcw, gridOk := s.puzzleGridWorld()

	if gridOk {
		missingSet := make(map[int]bool)
		for _, idx := range puzzle.MissingIndices {
			missingSet[idx] = true
		}

		col := int((wx - gpx) / gcw)
		row := int((wy - gpy) / gcw)

		if col >= 0 && col < 10 && row >= 0 && row < 10 {
			gIdx := row*10 + col

			if missingSet[gIdx] {
				occupied := false
				for _, other := range puzzle.TrayPieces {
					if other.IsPlaced && other.PlacedX == col && other.PlacedY == row {
						occupied = true

						break
					}
				}

				if !occupied {
					tp.IsPlaced = true
					tp.PlacedX = col
					tp.PlacedY = row
					placed = true
					s.scene.SaveGameState()
				}
			}
		}
	}

	if !placed {
		// Drop at world position
		tp.TrayX = wx
		tp.TrayY = wy
	}

	gd.HoldingPiece = -1
	gd.Dragging = false
}

func (s *DragDropSystem) gridCellMapSize() float64 {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return 68
	}

	og := tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return 68
	}

	for _, obj := range og.Objects {
		if obj.Name == "puzzle" {
			return math.Min(obj.Width, obj.Height) / 10
		}
	}

	return 68
}

func (s *DragDropSystem) puzzleGridWorld() (px, py, cellW float64, ok bool) {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return 0, 0, 0, false
	}

	og := tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return 0, 0, 0, false
	}

	for _, obj := range og.Objects {
		if obj.Name == "puzzle" {
			pw, ph := obj.Width, obj.Height
			side := math.Min(pw, ph)
			px = obj.X + (pw-side)/2
			py = obj.Y + (ph-side)/2

			return px, py, side / 10, true
		}
	}

	return 0, 0, 0, false
}
