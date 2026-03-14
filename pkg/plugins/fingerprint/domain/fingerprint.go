package domain

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// FingerprintColor represents the color type of a fingerprint (uint8).
type FingerprintColor uint8

const (
	ColorGrey   FingerprintColor = 0 // unassigned — player must choose
	ColorYellow FingerprintColor = 1
	ColorGreen  FingerprintColor = 2
	ColorRed    FingerprintColor = 3
)

// Fingerprint represents a fingerprint composed of tiles with a color.
// The unique ID is derived from: color byte + 9-digit hash of all tile values.
type Fingerprint struct {
	Color FingerprintColor
	Tiles []Tile // all tiles in grid order
	GridW int    // grid width (columns)
	GridH int    // grid height (rows)
}

// NewFingerprint creates a fingerprint with the given grid dimensions.
func NewFingerprint(gridW, gridH int) *Fingerprint {
	tiles := make([]Tile, gridW*gridH)

	for y := range gridH {
		for x := range gridW {
			tiles[y*gridW+x] = Tile{X: x, Y: y}
		}
	}

	return &Fingerprint{Tiles: tiles, GridW: gridW, GridH: gridH}
}

// TileAt returns the tile at grid position (x, y).
func (f *Fingerprint) TileAt(x, y int) *Tile {
	if x < 0 || x >= f.GridW || y < 0 || y >= f.GridH {
		return nil
	}

	return &f.Tiles[y*f.GridW+x]
}

// TileValues returns all tile values in grid order (including 0 for empty).
func (f *Fingerprint) TileValues() []uint16 {
	vals := make([]uint16, len(f.Tiles))

	for i, t := range f.Tiles {
		vals[i] = t.Value
	}

	return vals
}

// EmptyCount returns how many tiles are missing (value == 0).
func (f *Fingerprint) EmptyCount() int {
	count := 0

	for _, t := range f.Tiles {
		if t.IsEmpty() {
			count++
		}
	}

	return count
}

// Hash computes the 9-digit hash from all non-zero tile values.
// The hash is deterministic: same tile values → same hash.
func (f *Fingerprint) Hash() string {
	h := sha256.New()

	for _, t := range f.Tiles {
		b := make([]byte, 2)
		binary.LittleEndian.PutUint16(b, t.Value)
		h.Write(b)
	}

	sum := h.Sum(nil)

	// Take first 4.5 bytes → convert to 9-digit decimal
	val := binary.BigEndian.Uint64(sum[:8]) % 1_000_000_000

	return fmt.Sprintf("%09d", val)
}

// UniqueID returns the full fingerprint identifier: color byte + 9-digit hash.
// Example: "2-384729105" (green, hash 384729105)
func (f *Fingerprint) UniqueID() string {
	return fmt.Sprintf("%d-%s", f.Color, f.Hash())
}

// IsComplete returns true if all tiles are placed (no empty slots).
func (f *Fingerprint) IsComplete() bool {
	return f.EmptyCount() == 0
}
