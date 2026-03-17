package components

import "github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"

// GameData holds ALL mutable game state as a single component.
// This is the canonical source of truth — no state lives on GameScene.
// Systems read and write this component.
type GameData struct {
	// Core game data
	DB         *domain.FingerprintDB
	Cases      []*domain.CaseConfig
	PuzzleSeed uint64

	// Selection
	SelectedCase   int
	SelectedPuzzle int

	// Scroll offsets
	CasesScroll int
	NamesScroll int
	DescScroll  int

	// Drag-drop state
	HoldingPiece int // -1 = none
	Dragging     bool

	// Result overlay
	ShowResult int // 0=none, 1=success, 2=fail
	ResultTick int // frames remaining

	// Loading
	LoadProgress float64
	LoadStatus   string
	LoadStep     int
	LoadDone     chan struct{}

	// Assets directory
	AssetsDir string
}
