// Package domain contains the pure game logic for the fingerprint puzzle.
// No Ebiten imports — fully testable.
package domain

// Tile represents a single piece of a fingerprint.
// Value 0 = missing (needs to be placed by the player).
// Rotation is 0-3 (0°, 90°, 180°, 270°).
type Tile struct {
	Value    uint16
	Rotation uint8 // 0-3
	X, Y     int   // grid position
}

// Rotate cycles the tile rotation clockwise (0→1→2→3→0).
func (t *Tile) Rotate() {
	t.Rotation = (t.Rotation + 1) % 4
}

// IsEmpty returns true if this tile slot is missing (needs a piece).
func (t *Tile) IsEmpty() bool {
	return t.Value == 0
}
