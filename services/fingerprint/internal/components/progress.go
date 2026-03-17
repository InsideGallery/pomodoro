package components

// ProgressBar holds loading screen progress state.
type ProgressBar struct {
	Progress float64 // 0.0 → 1.0
	Status   string  // current loading step text
	Step     int     // which deferred loading step
	Done     bool    // all loading complete
}
