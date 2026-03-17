package components

// GameState represents the current scene state.
type GameState int

const (
	StateLoading           GameState = iota // Loading assets
	StateDisabled                           // PC off / boot animation
	StateEnabled                            // Desktop with app icon
	StateApplicationLayout                  // Case selection UI
	StateApplicationNet                     // Puzzle workspace
)

// State holds the game state machine data.
type State struct {
	Current  GameState
	BootTick int // frames since boot animation started
}
