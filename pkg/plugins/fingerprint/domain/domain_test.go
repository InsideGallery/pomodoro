package domain

import (
	"testing"
)

func TestTileRotation(t *testing.T) {
	tile := Tile{Value: 100, Rotation: 0}

	tile.Rotate()
	if tile.Rotation != 1 {
		t.Fatalf("expected rotation 1, got %d", tile.Rotation)
	}

	tile.Rotate()
	tile.Rotate()
	tile.Rotate()

	if tile.Rotation != 0 {
		t.Fatalf("expected rotation 0 after full cycle, got %d", tile.Rotation)
	}
}

func TestTileEmpty(t *testing.T) {
	empty := Tile{Value: 0}
	if !empty.IsEmpty() {
		t.Fatal("value 0 should be empty")
	}

	full := Tile{Value: 42}
	if full.IsEmpty() {
		t.Fatal("value 42 should not be empty")
	}
}

func TestFingerprintCreation(t *testing.T) {
	fp := NewFingerprint(4, 3)

	if len(fp.Tiles) != 12 {
		t.Fatalf("expected 12 tiles, got %d", len(fp.Tiles))
	}

	if fp.GridW != 4 || fp.GridH != 3 {
		t.Fatal("wrong grid dimensions")
	}

	// All tiles should be empty initially
	if fp.EmptyCount() != 12 {
		t.Fatalf("expected 12 empty, got %d", fp.EmptyCount())
	}
}

func TestFingerprintTileAt(t *testing.T) {
	fp := NewFingerprint(3, 3)
	fp.Tiles[4].Value = 999 // position (1,1)

	tile := fp.TileAt(1, 1)
	if tile == nil {
		t.Fatal("expected tile at (1,1)")
	}

	if tile.Value != 999 {
		t.Fatalf("expected value 999, got %d", tile.Value)
	}

	// Out of bounds
	if fp.TileAt(-1, 0) != nil {
		t.Fatal("expected nil for out of bounds")
	}
}

func TestFingerprintHashDeterministic(t *testing.T) {
	fp1 := NewFingerprint(3, 3)
	fp2 := NewFingerprint(3, 3)

	// Set same values
	for i := range fp1.Tiles {
		fp1.Tiles[i].Value = uint16(i + 1)
		fp2.Tiles[i].Value = uint16(i + 1)
	}

	if fp1.Hash() != fp2.Hash() {
		t.Fatal("same tiles should produce same hash")
	}

	if len(fp1.Hash()) != 9 {
		t.Fatalf("hash should be 9 digits, got %d", len(fp1.Hash()))
	}
}

func TestFingerprintHashDiffers(t *testing.T) {
	fp1 := NewFingerprint(3, 3)
	fp2 := NewFingerprint(3, 3)

	for i := range fp1.Tiles {
		fp1.Tiles[i].Value = uint16(i + 1)
		fp2.Tiles[i].Value = uint16(i + 100)
	}

	if fp1.Hash() == fp2.Hash() {
		t.Fatal("different tiles should produce different hash")
	}
}

func TestFingerprintUniqueID(t *testing.T) {
	fp := NewFingerprint(3, 3)
	fp.Color = ColorGreen

	for i := range fp.Tiles {
		fp.Tiles[i].Value = uint16(i + 1)
	}

	id := fp.UniqueID()

	// Should start with color "2-"
	if id[0] != '2' || id[1] != '-' {
		t.Fatalf("expected ID starting with '2-', got %s", id)
	}

	if len(id) != 11 { // "2-" + 9 digits
		t.Fatalf("expected ID length 11, got %d", len(id))
	}
}

func TestFingerprintComplete(t *testing.T) {
	fp := NewFingerprint(2, 2)

	if fp.IsComplete() {
		t.Fatal("empty fingerprint should not be complete")
	}

	for i := range fp.Tiles {
		fp.Tiles[i].Value = uint16(i + 1)
	}

	if !fp.IsComplete() {
		t.Fatal("all tiles filled should be complete")
	}
}

func TestDatabaseLookup(t *testing.T) {
	db := NewDatabase()

	person := &Person{
		Name:          "John Doe",
		FingerprintID: "2-123456789",
	}
	db.Add(person)

	found := db.Lookup("2-123456789")
	if found == nil || found.Name != "John Doe" {
		t.Fatal("should find person by fingerprint ID")
	}

	notFound := db.Lookup("1-000000000")
	if notFound != nil {
		t.Fatal("should return nil for unknown ID")
	}
}

func TestCaseSubmit(t *testing.T) {
	db := NewDatabase()

	// Create a "solved" fingerprint and register the person
	gen := NewPuzzleGenerator(42)
	solved := gen.GenerateSolvedFingerprint(3, 3, ColorGreen)
	targetID := solved.UniqueID()

	db.Add(&Person{
		Name:          "Jane Smith",
		FingerprintID: targetID,
	})

	// Create a puzzle from the solved fingerprint
	puzzle, _, _ := gen.GeneratePuzzle(solved, 3)

	// Fill in the missing tiles correctly
	for i := range puzzle.Tiles {
		if puzzle.Tiles[i].IsEmpty() {
			puzzle.Tiles[i].Value = solved.Tiles[i].Value
		}
	}

	c := NewCase(1, puzzle)
	person, correct := c.Submit(db, targetID)

	if !correct {
		t.Fatal("should be correct when tiles match")
	}

	if person == nil || person.Name != "Jane Smith" {
		t.Fatal("should find the person")
	}

	if !c.IsSolved() {
		t.Fatal("case should be solved")
	}
}

func TestCaseSubmitWrong(t *testing.T) {
	db := NewDatabase()
	gen := NewPuzzleGenerator(42)

	solved := gen.GenerateSolvedFingerprint(3, 3, ColorRed)
	targetID := solved.UniqueID()

	puzzle, _, _ := gen.GeneratePuzzle(solved, 3)

	// Don't fill tiles correctly — leave them empty
	c := NewCase(1, puzzle)
	person, correct := c.Submit(db, targetID)

	if correct {
		t.Fatal("should not be correct with missing tiles")
	}

	if person != nil {
		t.Fatal("should not find a person with wrong fingerprint")
	}

	if c.Status != CaseFailed {
		t.Fatal("case should be failed")
	}
}

func TestPuzzleGeneratorRemovesTiles(t *testing.T) {
	gen := NewPuzzleGenerator(42)
	solved := gen.GenerateSolvedFingerprint(4, 4, ColorYellow)

	puzzle, removed, targetID := gen.GeneratePuzzle(solved, 5)

	if puzzle.EmptyCount() != 5 {
		t.Fatalf("expected 5 empty tiles, got %d", puzzle.EmptyCount())
	}

	if len(removed) != 5 {
		t.Fatalf("expected 5 removed pieces, got %d", len(removed))
	}

	if targetID == "" {
		t.Fatal("target ID should not be empty")
	}

	// Removed pieces should have non-zero values
	for _, p := range removed {
		if p.Value == 0 {
			t.Fatal("removed piece should have non-zero value")
		}
	}
}

func TestGeneratePersonInDatabase(t *testing.T) {
	db := NewDatabase()
	gen := NewPuzzleGenerator(42)

	person := gen.GeneratePerson(db, "Alice", "avatar_1", 3, 3)

	if person.Name != "Alice" {
		t.Fatal("wrong name")
	}

	if db.Size() != 1 {
		t.Fatal("database should have 1 person")
	}

	found := db.Lookup(person.FingerprintID)
	if found == nil {
		t.Fatal("should find person in database")
	}
}
