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
// Stateless — all state read/written via GameData component in Registry.
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

	cur := GetCursor(reg)
	if cur == nil {
		return nil
	}

	cx, cy := float64(cur.X), float64(cur.Y)

	// Mouse wheel: rotate held piece
	_, wy := ebiten.Wheel()
	if wy != 0 && gd.HoldingPiece >= 0 && gd.HoldingPiece < len(puzzle.TrayPieces) {
		if wy > 0 {
			puzzle.TrayPieces[gd.HoldingPiece].Rotation = (puzzle.TrayPieces[gd.HoldingPiece].Rotation + 1) % domain.RotationSteps
		} else {
			puzzle.TrayPieces[gd.HoldingPiece].Rotation = (puzzle.TrayPieces[gd.HoldingPiece].Rotation + domain.RotationSteps - 1) % domain.RotationSteps
		}
	}

	// Mouse pressed → try pickup (skip if over a button zone)
	input := s.scene.GetInputSystem()
	overButton := input != nil && input.HasHoveredZone()

	if !overButton && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && gd.HoldingPiece < 0 {
		s.tryPickup(gd, puzzle, cx, cy)
	}

	// Touch pressed → try pickup
	if !overButton && gd.HoldingPiece < 0 {
		for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
			tx, ty := ebiten.TouchPosition(id)
			s.tryPickup(gd, puzzle, float64(tx), float64(ty))

			break
		}
	}

	// Released → try place
	mouseRel := gd.Dragging && gd.HoldingPiece >= 0 && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	touchRel := gd.Dragging && gd.HoldingPiece >= 0 && len(inpututil.AppendJustReleasedTouchIDs(nil)) > 0

	if mouseRel || touchRel {
		s.tryPlace(gd, puzzle, cx, cy)
	}

	return nil
}

func (s *DragDropSystem) Draw(_ context.Context, _ *ebiten.Image) {}

// HoldingPiece returns the currently held piece index (-1 if none).
func (s *DragDropSystem) HoldingPiece() int {
	gd := GetGameData(s.scene.GetRegistry())
	if gd == nil {
		return -1
	}

	return gd.HoldingPiece
}

// IsDragging returns true if a piece is being dragged.
func (s *DragDropSystem) IsDragging() bool {
	gd := GetGameData(s.scene.GetRegistry())
	if gd == nil {
		return false
	}

	return gd.Dragging
}

func (s *DragDropSystem) tryPickup(gd *c.GameData, puzzle *domain.PuzzleConfig, cx, cy float64) {
	co := CoordsFromScene(s.scene)
	cellSz := s.gridCellScreenSize()

	for i := range puzzle.TrayPieces {
		tp := &puzzle.TrayPieces[i]
		if tp.IsPlaced {
			continue
		}

		tx := co.MapToScreenX(tp.TrayX)
		ty := co.MapToScreenY(tp.TrayY)

		if cx >= tx && cx <= tx+cellSz && cy >= ty && cy <= ty+cellSz {
			gd.HoldingPiece = i
			gd.Dragging = true

			return
		}
	}

	// Check placed pieces on grid
	gpx, gpy, gcw, ok := s.puzzleGridScreen()
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

		if cx >= gx && cx <= gx+gcw && cy >= gy && cy <= gy+gcw {
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

func (s *DragDropSystem) tryPlace(gd *c.GameData, puzzle *domain.PuzzleConfig, cx, cy float64) {
	tp := &puzzle.TrayPieces[gd.HoldingPiece]
	placed := false

	gpx, gpy, gcw, gridOk := s.puzzleGridScreen()

	if gridOk {
		missingSet := make(map[int]bool)
		for _, idx := range puzzle.MissingIndices {
			missingSet[idx] = true
		}

		col := int((cx - gpx) / gcw)
		row := int((cy - gpy) / gcw)

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

					slog.Info("placed piece", "tray", gd.HoldingPiece, "x", col, "y", row)
					s.scene.SaveGameState()
				}
			}
		}
	}

	if !placed {
		co := CoordsFromScene(s.scene)
		tp.TrayX = co.ScreenToMapX(cx)
		tp.TrayY = co.ScreenToMapY(cy)
	}

	gd.HoldingPiece = -1
	gd.Dragging = false
}

func (s *DragDropSystem) gridCellScreenSize() float64 {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return 40
	}

	co := CoordsFromScene(s.scene)

	og := tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return 40
	}

	for _, obj := range og.Objects {
		if obj.Name == "puzzle" {
			return math.Min(co.MapToScreenSize(obj.Width), co.MapToScreenSize(obj.Height)) / 10
		}
	}

	return 40
}

func (s *DragDropSystem) puzzleGridScreen() (px, py, cellW float64, ok bool) {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return 0, 0, 0, false
	}

	co := CoordsFromScene(s.scene)

	og := tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return 0, 0, 0, false
	}

	for _, obj := range og.Objects {
		if obj.Name == "puzzle" {
			px = co.MapToScreenX(obj.X)
			py = co.MapToScreenY(obj.Y)
			pw := co.MapToScreenSize(obj.Width)
			ph := co.MapToScreenSize(obj.Height)
			side := math.Min(pw, ph)
			px += (pw - side) / 2
			py += (ph - side) / 2

			return px, py, side / 10, true
		}
	}

	return 0, 0, 0, false
}
