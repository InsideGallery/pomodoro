package domain

import (
	"math/rand/v2"
)

// PuzzleGenerator creates puzzles dynamically.
type PuzzleGenerator struct {
	rng *rand.Rand
}

// NewPuzzleGenerator creates a generator with the given seed.
func NewPuzzleGenerator(seed uint64) *PuzzleGenerator {
	return &PuzzleGenerator{
		rng: rand.New(rand.NewPCG(seed, seed^0xCAFEBABE)), //nolint:gosec // game logic
	}
}

// GenerateSolvedFingerprint creates a fully-filled fingerprint.
// This is the "answer" — the complete fingerprint before removing pieces.
func (g *PuzzleGenerator) GenerateSolvedFingerprint(gridW, gridH int, clr FingerprintColor) *Fingerprint {
	fp := NewFingerprint(gridW, gridH)
	fp.Color = clr

	for i := range fp.Tiles {
		fp.Tiles[i].Value = uint16(g.rng.IntN(65534) + 1) // 1-65535, never 0
	}

	return fp
}

// GeneratePuzzle creates a puzzle by removing some tiles from a solved fingerprint.
// Returns: the puzzle (with holes), the removed pieces, and the target fingerprint ID.
func (g *PuzzleGenerator) GeneratePuzzle(solved *Fingerprint, removeCount int) (*Fingerprint, []Tile, string) {
	targetID := solved.UniqueID()

	// Clone the solved fingerprint as the puzzle
	puzzle := NewFingerprint(solved.GridW, solved.GridH)
	puzzle.Color = solved.Color

	for i := range solved.Tiles {
		puzzle.Tiles[i] = solved.Tiles[i]
	}

	// Select random tiles to remove
	indices := g.rng.Perm(len(puzzle.Tiles))

	if removeCount > len(indices) {
		removeCount = len(indices)
	}

	removed := make([]Tile, 0, removeCount)

	for _, idx := range indices[:removeCount] {
		removed = append(removed, puzzle.Tiles[idx])
		puzzle.Tiles[idx].Value = 0 // empty slot
	}

	// Shuffle removed pieces and randomize their rotation
	g.rng.Shuffle(len(removed), func(i, j int) {
		removed[i], removed[j] = removed[j], removed[i]
	})

	for i := range removed {
		removed[i].Rotation = uint8(g.rng.IntN(4))
	}

	return puzzle, removed, targetID
}

// GeneratePerson creates a random person with a fingerprint in the database.
func (g *PuzzleGenerator) GeneratePerson(db *Database, name, avatarKey string, gridW, gridH int) *Person {
	clr := FingerprintColor(g.rng.IntN(3) + 1) // 1-3 (not grey)
	solved := g.GenerateSolvedFingerprint(gridW, gridH, clr)

	person := &Person{
		Name:          name,
		AvatarKey:     avatarKey,
		FingerprintID: solved.UniqueID(),
	}

	db.Add(person)

	return person
}

// MirrorPieces creates mirrored variants of pieces for extra difficulty.
func MirrorPieces(pieces []Tile) []Tile {
	mirrored := make([]Tile, len(pieces))

	for i, p := range pieces {
		mirrored[i] = p
		mirrored[i].X = -p.X - 1 // mirror horizontally
		mirrored[i].Rotation = (p.Rotation + 2) % 4
	}

	return mirrored
}
