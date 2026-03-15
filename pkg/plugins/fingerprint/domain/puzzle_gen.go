package domain

import (
	"math/rand/v2"
)

// PuzzleGenerator creates puzzles dynamically.
type PuzzleGenerator struct {
	rng *rand.Rand
}

func NewPuzzleGenerator(seed uint64) *PuzzleGenerator {
	return &PuzzleGenerator{
		rng: rand.New(rand.NewPCG(seed, seed^0xCAFEBABE)), //nolint:gosec // game logic
	}
}

// GenerateSolvedFingerprint creates a fully-filled fingerprint where every tile
// has its correct uint32 value: EncodeTile(x, y, rotation=0, content).
// The content byte is random per tile (from the generator's RNG).
func (g *PuzzleGenerator) GenerateSolvedFingerprint(gridW, gridH int, clr FingerprintColor) *Fingerprint {
	fp := NewFingerprint(gridW, gridH)
	fp.Color = clr

	for i := range fp.Tiles {
		content := uint8(g.rng.IntN(255) + 1) // 1-255, never 0
		fp.Tiles[i].Content = content
		fp.Tiles[i].Value = EncodeTile(uint8(fp.Tiles[i].X), uint8(fp.Tiles[i].Y), 0, content)
	}

	return fp
}

// PuzzleResult holds the generated puzzle data.
type PuzzleResult struct {
	Puzzle     *Fingerprint // partially filled (holes where pieces removed)
	Pieces     []Tile       // removed pieces (shuffled, random rotation)
	TargetID   string       // pre-computed correct UniqueID
	TargetHash string       // pre-computed correct hash (without color)
}

// GeneratePuzzle creates a puzzle by removing tiles from a solved fingerprint.
// The correct hash is pre-computed from the solved state.
// Only placing all pieces back at correct (x, y) with rotation 0 reproduces the hash.
func (g *PuzzleGenerator) GeneratePuzzle(solved *Fingerprint, removeCount int) PuzzleResult {
	// Pre-compute the correct answer BEFORE modifying anything
	targetID := solved.UniqueID()
	targetHash := solved.Hash()

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
		// Save the piece with its correct data
		piece := puzzle.Tiles[idx]
		removed = append(removed, piece)

		// Clear the slot in the puzzle
		puzzle.Tiles[idx].Value = 0
	}

	// Shuffle removed pieces
	g.rng.Shuffle(len(removed), func(i, j int) {
		removed[i], removed[j] = removed[j], removed[i]
	})

	// Randomize rotation of removed pieces (makes them wrong until player fixes)
	for i := range removed {
		rotations := g.rng.IntN(3) + 1 // 1-3 rotations (never 0 = already correct)

		for range rotations {
			removed[i].Rotate()
		}
	}

	return PuzzleResult{
		Puzzle:     puzzle,
		Pieces:     removed,
		TargetID:   targetID,
		TargetHash: targetHash,
	}
}

// GeneratePerson creates a person with a unique fingerprint and registers in the DB.
func (g *PuzzleGenerator) GeneratePerson(
	db *Database, name, avatarKey string, gridW, gridH int,
) (*Person, *Fingerprint) {
	clr := FingerprintColor(g.rng.IntN(4) + 1) // 1-4 (yellow, green, red, blue)
	solved := g.GenerateSolvedFingerprint(gridW, gridH, clr)

	person := &Person{
		Name:          name,
		AvatarKey:     avatarKey,
		FingerprintID: solved.UniqueID(),
	}

	db.Add(person)

	return person, solved
}

// MirrorPieces creates horizontally mirrored variants of pieces.
func MirrorPieces(pieces []Tile, gridW int) []Tile {
	mirrored := make([]Tile, len(pieces))

	for i, p := range pieces {
		mirrored[i] = p
		mirrored[i].X = gridW - 1 - p.X
		mirrored[i].Recompute((p.Rotation() + 2) % 4)
	}

	return mirrored
}
