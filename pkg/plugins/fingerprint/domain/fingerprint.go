package domain

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// FingerprintColor represents the color type of a fingerprint.
type FingerprintColor uint8

const (
	ColorGrey   FingerprintColor = 0 // unassigned — player must choose
	ColorYellow FingerprintColor = 1
	ColorGreen  FingerprintColor = 2
	ColorRed    FingerprintColor = 3
)

// MaxGridSize is the maximum supported grid dimension.
// x and y are encoded as uint8, so max 255.
const MaxGridSize = 255

// Fingerprint represents a fingerprint composed of tiles with a color.
// UniqueID = color byte + 9-digit hash of all tile uint32 values.
// Only 1 exact combination of (color, all tiles at correct positions + rotation 0)
// produces the pre-computed correct hash.
type Fingerprint struct {
	Color FingerprintColor
	Tiles []Tile
	GridW int
	GridH int
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

// Hash computes the 9-digit hash from all tile uint32 values.
// Deterministic: same tile values in same order → same hash.
func (f *Fingerprint) Hash() string {
	h := sha256.New()

	for _, t := range f.Tiles {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, t.Value)
		h.Write(b)
	}

	sum := h.Sum(nil)
	val := binary.BigEndian.Uint64(sum[:8]) % 1_000_000_000

	return fmt.Sprintf("%09d", val)
}

// UniqueID returns the full fingerprint identifier: color byte + 9-digit hash.
func (f *Fingerprint) UniqueID() string {
	return fmt.Sprintf("%d-%s", f.Color, f.Hash())
}

// IsComplete returns true if all tiles are placed (no empty slots).
func (f *Fingerprint) IsComplete() bool {
	return f.EmptyCount() == 0
}
