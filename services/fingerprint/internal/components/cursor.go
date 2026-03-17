package components

// Cursor holds the virtual cursor state.
// Delta-based to prevent stickiness at room edges.
type Cursor struct {
	X, Y     int
	PrevRawX int
	PrevRawY int
	Inited   bool
	RoomMinX int
	RoomMinY int
	RoomMaxX int
	RoomMaxY int
}
