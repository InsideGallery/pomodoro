package components

import "github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"

// PuzzleGrid holds the puzzle workspace state.
type PuzzleGrid struct {
	Puzzle *domain.PuzzleConfig
	// CellSize computed at render time from Transform dimensions
}

// PuzzlePiece links a tray piece to its visual representation.
type PuzzlePiece struct {
	TrayIdx int // index into PuzzleConfig.TrayPieces
	IsDecoy bool
}
