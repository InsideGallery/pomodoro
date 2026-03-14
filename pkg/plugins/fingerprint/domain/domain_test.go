package domain

import (
	"testing"
)

func TestEncodeTile(t *testing.T) {
	val := EncodeTile(5, 10, 2, 200)

	// x=5 in byte 0, y=10 in byte 1, rotation=2 in byte 2, content=200 in byte 3
	if val&0xFF != 5 {
		t.Fatal("x byte wrong")
	}

	if (val>>8)&0xFF != 10 {
		t.Fatal("y byte wrong")
	}

	if DecodeRotation(val) != 2 {
		t.Fatal("rotation byte wrong")
	}

	if (val>>24)&0xFF != 200 {
		t.Fatal("content byte wrong")
	}
}

func TestTileRotation(t *testing.T) {
	tile := Tile{Content: 42, X: 3, Y: 5}
	tile.Recompute(0)

	if tile.Rotation() != 0 {
		t.Fatal("expected rotation 0")
	}

	tile.Rotate()

	if tile.Rotation() != 1 {
		t.Fatalf("expected rotation 1, got %d", tile.Rotation())
	}

	tile.Rotate()
	tile.Rotate()
	tile.Rotate()

	if tile.Rotation() != 0 {
		t.Fatalf("expected rotation 0 after full cycle, got %d", tile.Rotation())
	}
}

func TestTileValueChangesWithRotation(t *testing.T) {
	tile := Tile{Content: 42, X: 3, Y: 5}
	tile.Recompute(0)
	val0 := tile.Value

	tile.Rotate()
	val1 := tile.Value

	if val0 == val1 {
		t.Fatal("tile value should change with rotation")
	}
}

func TestTileIsCorrect(t *testing.T) {
	tile := Tile{Content: 42, X: 3, Y: 5}
	tile.Recompute(0)

	if !tile.IsCorrect(3, 5) {
		t.Fatal("should be correct at original position with rotation 0")
	}

	tile.Rotate()

	if tile.IsCorrect(3, 5) {
		t.Fatal("should not be correct after rotation")
	}
}

func TestFingerprintHashDeterministic(t *testing.T) {
	gen := NewPuzzleGenerator(42)
	fp1 := gen.GenerateSolvedFingerprint(3, 3, ColorGreen)

	gen2 := NewPuzzleGenerator(42)
	fp2 := gen2.GenerateSolvedFingerprint(3, 3, ColorGreen)

	if fp1.Hash() != fp2.Hash() {
		t.Fatal("same seed should produce same hash")
	}

	if len(fp1.Hash()) != 9 {
		t.Fatalf("hash should be 9 digits, got %d chars", len(fp1.Hash()))
	}
}

func TestFingerprintHashChangesWithRotation(t *testing.T) {
	gen := NewPuzzleGenerator(42)
	fp := gen.GenerateSolvedFingerprint(3, 3, ColorGreen)
	correctHash := fp.Hash()

	// Rotate one tile — hash should change
	fp.Tiles[0].Rotate()
	wrongHash := fp.Hash()

	if correctHash == wrongHash {
		t.Fatal("hash should change when a tile is rotated")
	}
}

func TestFingerprintUniqueID(t *testing.T) {
	gen := NewPuzzleGenerator(42)
	fp := gen.GenerateSolvedFingerprint(3, 3, ColorGreen)
	id := fp.UniqueID()

	if id[0] != '2' || id[1] != '-' {
		t.Fatalf("expected ID starting with '2-', got %s", id)
	}

	if len(id) != 11 {
		t.Fatalf("expected ID length 11, got %d", len(id))
	}
}

func TestPuzzlePreComputedHash(t *testing.T) {
	gen := NewPuzzleGenerator(42)
	solved := gen.GenerateSolvedFingerprint(4, 4, ColorYellow)

	result := gen.GeneratePuzzle(solved, 5)

	// Pre-computed hash should match the solved fingerprint
	if result.TargetID != solved.UniqueID() {
		t.Fatal("target ID should match solved fingerprint")
	}

	// Puzzle has holes
	if result.Puzzle.EmptyCount() != 5 {
		t.Fatalf("expected 5 empty, got %d", result.Puzzle.EmptyCount())
	}

	// Pieces are rotated (not at rotation 0)
	allRotZero := true

	for _, p := range result.Pieces {
		if p.Rotation() != 0 {
			allRotZero = false

			break
		}
	}

	if allRotZero {
		t.Fatal("pieces should be rotated away from 0")
	}

	// Place pieces back correctly → hash should match
	for _, piece := range result.Pieces {
		tile := result.Puzzle.TileAt(piece.X, piece.Y)
		if tile == nil {
			t.Fatalf("tile position (%d,%d) out of bounds", piece.X, piece.Y)
		}

		// Reset to correct rotation
		tile.Content = piece.Content
		tile.Recompute(0)
	}

	if result.Puzzle.UniqueID() != result.TargetID {
		t.Fatalf("placing pieces correctly should reproduce target ID\ngot:  %s\nwant: %s",
			result.Puzzle.UniqueID(), result.TargetID)
	}
}

func TestCaseSubmitCorrect(t *testing.T) {
	db := NewDatabase()
	gen := NewPuzzleGenerator(42)

	_, solved := gen.GeneratePerson(db, "Jane Smith", "avatar_1", 3, 3)
	targetID := solved.UniqueID()

	result := gen.GeneratePuzzle(solved, 3)

	// Fix all pieces
	for _, piece := range result.Pieces {
		tile := result.Puzzle.TileAt(piece.X, piece.Y)
		tile.Content = piece.Content
		tile.Recompute(0)
	}

	c := NewCase(1, result.Puzzle)
	person, correct := c.Submit(db, targetID)

	if !correct {
		t.Fatal("should be correct")
	}

	if person == nil || person.Name != "Jane Smith" {
		t.Fatal("should find Jane Smith")
	}
}

func TestCaseSubmitWrong(t *testing.T) {
	db := NewDatabase()
	gen := NewPuzzleGenerator(42)

	_, solved := gen.GeneratePerson(db, "John Doe", "avatar_2", 3, 3)

	result := gen.GeneratePuzzle(solved, 3)

	// Don't fix pieces — submit with holes
	c := NewCase(1, result.Puzzle)
	person, correct := c.Submit(db, result.TargetID)

	if correct {
		t.Fatal("should not be correct with missing tiles")
	}

	if person != nil {
		t.Fatal("wrong hash should not find anyone")
	}
}

func TestDatabaseLookup(t *testing.T) {
	db := NewDatabase()

	db.Add(&Person{Name: "Alice", FingerprintID: "2-123456789"})

	if db.Lookup("2-123456789") == nil {
		t.Fatal("should find Alice")
	}

	if db.Lookup("1-000000000") != nil {
		t.Fatal("should not find unknown")
	}

	if db.Size() != 1 {
		t.Fatal("size should be 1")
	}
}
