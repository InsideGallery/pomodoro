// Package entities provides assemblage (factory) functions that create
// entity groups in the Registry for each game state.
package entities

import (
	"log/slog"

	"github.com/InsideGallery/core/memory/registry"

	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
	"github.com/InsideGallery/pomodoro/pkg/tilemap"
	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
)

// RegType is the Registry type used by the game.
type RegType = *registry.Registry[string, uint64, any]

// nextID is a simple global counter for entity IDs.
var entityCounter uint64 //nolint:gochecknoglobals // simple ID gen

func nextID() uint64 {
	entityCounter++

	return entityCounter
}

// CleanStateGroups removes all state-specific entity groups.
// Called before entering a new state.
func CleanStateGroups(reg RegType) {
	groups := []string{
		c.GroupImageLayer, c.GroupTileLayer, c.GroupButton,
		c.GroupCaseList, c.GroupPuzzleList, c.GroupAvatar,
		c.GroupDescription, c.GroupPuzzleGrid, c.GroupGridCell,
		c.GroupTrayPiece, c.GroupHeldPiece, c.GroupHashDisplay,
		c.GroupResultOverlay,
	}

	for _, g := range groups {
		grp := reg.GetGroup(g)
		if grp != nil {
			grp.Truncate()
		}
	}
}

// CreateCursorEntity creates the singleton cursor entity if not present.
func CreateCursorEntity(reg RegType, roomMinX, roomMinY, roomMaxX, roomMaxY int) {
	if _, err := reg.Get(c.GroupCursor, 0); err == nil {
		return // already exists
	}

	entity := &c.Entity{
		Cursor: &c.Cursor{
			RoomMinX: roomMinX,
			RoomMinY: roomMinY,
			RoomMaxX: roomMaxX,
			RoomMaxY: roomMaxY,
		},
	}

	if err := reg.Add(c.GroupCursor, 0, entity); err != nil {
		slog.Warn("create cursor entity", "error", err)
	}
}

// CreateProgressEntity creates the loading progress bar entity.
func CreateProgressEntity(reg RegType) {
	entity := &c.Entity{
		Progress: &c.ProgressBar{},
	}

	if err := reg.Add(c.GroupProgress, 0, entity); err != nil {
		slog.Warn("create progress entity", "error", err)
	}
}

// CreateAppLayoutEntities creates entities for the application layout state.
func CreateAppLayoutEntities(reg RegType, cases []*domain.CaseConfig, selectedCase int) {
	CleanStateGroups(reg)

	// Case list scrollable
	caseListEntity := &c.Entity{
		Scrollable: &c.Scrollable{
			Scroll:     0,
			RowH:       45,
			TotalItems: len(cases),
		},
		Transform: &c.Transform{}, // filled by RenderSystem from TMX
	}

	if err := reg.Add(c.GroupCaseList, 0, caseListEntity); err != nil {
		slog.Warn("create case list", "error", err)
	}

	// Puzzle list scrollable
	puzzleCount := 0
	if selectedCase >= 0 && selectedCase < len(cases) {
		puzzleCount = len(cases[selectedCase].Puzzles)
	}

	puzzleListEntity := &c.Entity{
		Scrollable: &c.Scrollable{
			Scroll:     0,
			RowH:       50,
			TotalItems: puzzleCount,
		},
		Transform: &c.Transform{},
	}

	if err := reg.Add(c.GroupPuzzleList, 0, puzzleListEntity); err != nil {
		slog.Warn("create puzzle list", "error", err)
	}

	// Description text block
	descEntity := &c.Entity{
		TextBlock: &c.TextBlock{
			FontSize: 9,
			Color:    c.DefaultTextColor(),
		},
		Transform: &c.Transform{},
	}

	if err := reg.Add(c.GroupDescription, 0, descEntity); err != nil {
		slog.Warn("create description", "error", err)
	}

	// Avatar
	avatarEntity := &c.Entity{
		Avatar:    &c.Avatar{Filename: domain.UnknownAvatar},
		Transform: &c.Transform{},
	}

	if err := reg.Add(c.GroupAvatar, 0, avatarEntity); err != nil {
		slog.Warn("create avatar", "error", err)
	}
}

// CreatePuzzleEntities creates entities for the puzzle workspace state.
func CreatePuzzleEntities(reg RegType, puzzle *domain.PuzzleConfig, tmap *tilemap.Map, scaleX, scaleY float64) {
	CleanStateGroups(reg)

	// Puzzle grid entity
	gridEntity := &c.Entity{
		PuzzleGrid: &c.PuzzleGrid{Puzzle: puzzle},
		Transform:  &c.Transform{},
	}

	if err := reg.Add(c.GroupPuzzleGrid, 0, gridEntity); err != nil {
		slog.Warn("create puzzle grid", "error", err)
	}

	// Hash display entity
	hashEntity := &c.Entity{
		TextBlock: &c.TextBlock{FontSize: 10, Color: c.DefaultTextColor()},
		Transform: &c.Transform{},
	}

	if err := reg.Add(c.GroupHashDisplay, 0, hashEntity); err != nil {
		slog.Warn("create hash display", "error", err)
	}

	// Tray pieces — initialize positions if needed
	initTrayPositions(puzzle, tmap)

	for i := range puzzle.TrayPieces {
		tp := &puzzle.TrayPieces[i]

		pieceEntity := &c.Entity{
			PuzzlePiece: &c.PuzzlePiece{TrayIdx: i, IsDecoy: tp.IsDecoy},
			Draggable:   &c.Draggable{},
			Transform: &c.Transform{
				X: tp.TrayX,
				Y: tp.TrayY,
			},
		}

		if err := reg.Add(c.GroupTrayPiece, nextID(), pieceEntity); err != nil {
			slog.Warn("create tray piece", "error", err)
		}
	}
}

// initTrayPositions assigns scattered positions to tray pieces within pieces rooms.
func initTrayPositions(puzzle *domain.PuzzleConfig, tmap *tilemap.Map) {
	if tmap == nil {
		return
	}

	og := tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return
	}

	// Collect pieces rectangles
	var rects []struct{ x, y, w, h float64 }

	for _, obj := range og.Objects {
		if obj.Name == "pieces" {
			rects = append(rects, struct{ x, y, w, h float64 }{
				x: obj.X, y: obj.Y, w: obj.Width, h: obj.Height,
			})
		}
	}

	if len(rects) == 0 {
		return
	}

	// Grid cell size in map coords
	cellMap := 68.0

	for _, obj := range og.Objects {
		if obj.Name == "puzzle" {
			side := obj.Width
			if obj.Height < side {
				side = obj.Height
			}

			cellMap = side / 10

			break
		}
	}

	for i := range puzzle.TrayPieces {
		tp := &puzzle.TrayPieces[i]
		if tp.IsPlaced || (tp.TrayX != 0 && tp.TrayY != 0) {
			continue
		}

		r := rects[i%len(rects)]
		tp.TrayX = r.x + float64(i%3)*cellMap + float64(i%2)*10
		tp.TrayY = r.y + float64(i/3)*cellMap + float64(i%2)*5

		if tp.TrayX+cellMap > r.x+r.w {
			tp.TrayX = r.x + r.w - cellMap
		}

		if tp.TrayY+cellMap > r.y+r.h {
			tp.TrayY = r.y + r.h - cellMap
		}
	}
}
