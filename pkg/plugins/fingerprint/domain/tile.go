package domain

// RotationSteps is the number of discrete rotation angles (every 45°).
const RotationSteps = 8

// Tile represents a single piece of a fingerprint.
// Value encodes (x, y, rotation, content) as uint32.
// Rotation 0-7 → 0°, 45°, 90°, 135°, 180°, 225°, 270°, 315°.
// Value 0 = missing (needs to be placed by the player).
type Tile struct {
	Value   uint32
	X, Y    int   // current grid position
	Content uint8 // the tile's unique pattern data (immutable)
}

// EncodeTile computes the uint32 value from position, rotation, and content.
// x: byte 0, y: byte 1, rotation: byte 2, content: byte 3.
// Only the correct (x, y, rotation=0) produces the value that matches the solved hash.
func EncodeTile(x, y, rotation, content uint8) uint32 {
	return uint32(x) | uint32(y)<<8 | uint32(rotation)<<16 | uint32(content)<<24
}

// DecodeRotation extracts the rotation byte from a tile value.
func DecodeRotation(value uint32) uint8 {
	return uint8((value >> 16) & 0xFF)
}

// Recompute recalculates the tile's Value from its current position and rotation.
func (t *Tile) Recompute(rotation uint8) {
	t.Value = EncodeTile(uint8(t.X), uint8(t.Y), rotation, t.Content)
}

// Rotate cycles rotation clockwise (45° step) and recomputes Value.
func (t *Tile) Rotate() {
	rot := DecodeRotation(t.Value)
	rot = (rot + 1) % RotationSteps
	t.Recompute(rot)
}

// Rotation returns the current rotation (0-7).
func (t *Tile) Rotation() uint8 {
	return DecodeRotation(t.Value)
}

// RotationDegrees returns rotation in degrees (0, 45, 90, ..., 315).
func RotationDegrees(step int) int {
	return (step % RotationSteps) * 45
}

// IsEmpty returns true if this tile slot is missing.
func (t *Tile) IsEmpty() bool {
	return t.Value == 0
}

// IsCorrect returns true if the tile is at its correct position with rotation 0.
func (t *Tile) IsCorrect(correctX, correctY int) bool {
	return t.X == correctX && t.Y == correctY && t.Rotation() == 0
}
