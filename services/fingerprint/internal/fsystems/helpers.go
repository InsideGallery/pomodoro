package fsystems

import (
	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
)

// GetGameData reads the singleton GameData component from Registry.
func GetGameData(reg RegType) *c.GameData {
	val, err := reg.Get(c.GroupGameState, 0)
	if err != nil {
		return nil
	}

	if entity, ok := val.(*c.Entity); ok {
		return entity.GameData
	}

	return nil
}

// GetState reads the singleton State component from Registry.
func GetState(reg RegType) *c.State {
	val, err := reg.Get(c.GroupGameState, 0)
	if err != nil {
		return nil
	}

	if entity, ok := val.(*c.Entity); ok {
		return entity.State
	}

	return nil
}

// GetCursor reads the singleton Cursor component from Registry.
func GetCursor(reg RegType) *c.Cursor {
	val, err := reg.Get(c.GroupCursor, 0)
	if err != nil {
		return nil
	}

	if entity, ok := val.(*c.Entity); ok {
		return entity.Cursor
	}

	return nil
}

// CurrentPuzzle returns the active puzzle from GameData, or nil.
func CurrentPuzzle(gd *c.GameData) *domain.PuzzleConfig {
	if gd == nil || gd.Cases == nil {
		return nil
	}

	if gd.SelectedCase < 0 || gd.SelectedCase >= len(gd.Cases) {
		return nil
	}

	cs := gd.Cases[gd.SelectedCase]
	if gd.SelectedPuzzle < 0 || gd.SelectedPuzzle >= len(cs.Puzzles) {
		return nil
	}

	return cs.Puzzles[gd.SelectedPuzzle]
}
