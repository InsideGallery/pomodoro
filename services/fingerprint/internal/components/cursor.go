package components

// Cursor holds the virtual cursor state.
// Position is in screen space. Room bounds are in WORLD (map) space.
// CursorSystem clamps by converting screen→world, clamping, converting back.
type Cursor struct {
	X, Y int // screen space

	// Cursor room bounds in WORLD (map) coordinates — from TMX
	WorldMinX float64
	WorldMinY float64
	WorldMaxX float64
	WorldMaxY float64
}
