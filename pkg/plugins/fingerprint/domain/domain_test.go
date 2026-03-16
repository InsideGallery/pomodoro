package domain

import (
	"path/filepath"
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

func TestWrongPositionChangesHash(t *testing.T) {
	gen := NewPuzzleGenerator(99)
	solved := gen.GenerateSolvedFingerprint(4, 4, ColorGreen)
	correctHash := solved.Hash()

	// Swap two tiles — hash should change
	solved.Tiles[0], solved.Tiles[1] = solved.Tiles[1], solved.Tiles[0]
	swappedHash := solved.Hash()

	if correctHash == swappedHash {
		t.Fatal("swapping tiles should change hash")
	}
}

func TestWrongColorChangesUniqueID(t *testing.T) {
	gen := NewPuzzleGenerator(99)
	fp1 := gen.GenerateSolvedFingerprint(3, 3, ColorGreen)

	gen2 := NewPuzzleGenerator(99)
	fp2 := gen2.GenerateSolvedFingerprint(3, 3, ColorRed)

	// Same tiles but different color → different UniqueID
	if fp1.Hash() != fp2.Hash() {
		t.Fatal("same seed should produce same hash regardless of color")
	}

	if fp1.UniqueID() == fp2.UniqueID() {
		t.Fatal("different colors should produce different UniqueID")
	}
}

func TestLargeGrid10x10(t *testing.T) {
	gen := NewPuzzleGenerator(42)
	solved := gen.GenerateSolvedFingerprint(10, 10, ColorYellow)

	if len(solved.Tiles) != 100 {
		t.Fatalf("10x10 grid should have 100 tiles, got %d", len(solved.Tiles))
	}

	result := gen.GeneratePuzzle(solved, 30)

	if result.Puzzle.EmptyCount() != 30 {
		t.Fatalf("expected 30 empty, got %d", result.Puzzle.EmptyCount())
	}

	if len(result.Pieces) != 30 {
		t.Fatalf("expected 30 pieces, got %d", len(result.Pieces))
	}

	// Reconstruct solved from puzzle + pieces → hash must match
	for _, piece := range result.Pieces {
		tile := result.Puzzle.TileAt(piece.X, piece.Y)
		tile.Content = piece.Content
		tile.Recompute(0)
	}

	if result.Puzzle.UniqueID() != result.TargetID {
		t.Fatal("reconstructed 10x10 should match target ID")
	}
}

func TestMirroredPiecesProduceDifferentHash(t *testing.T) {
	gen := NewPuzzleGenerator(42)
	solved := gen.GenerateSolvedFingerprint(4, 4, ColorBlue)
	result := gen.GeneratePuzzle(solved, 4)

	mirrored := MirrorPieces(result.Pieces, 4)

	// Place mirrored pieces — should NOT match target hash
	for _, piece := range mirrored {
		tile := result.Puzzle.TileAt(piece.X, piece.Y)
		if tile == nil {
			continue // mirrored position might be out of bounds
		}

		tile.Content = piece.Content
		tile.Recompute(piece.Rotation())
	}

	if result.Puzzle.UniqueID() == result.TargetID {
		t.Fatal("mirrored pieces should NOT reproduce target hash")
	}
}

func TestEveryRotationProducesDifferentValue(t *testing.T) {
	tile := Tile{Content: 42, X: 3, Y: 5}
	values := make(map[uint32]bool)

	for rot := uint8(0); rot < 4; rot++ {
		tile.Recompute(rot)
		if values[tile.Value] {
			t.Fatalf("rotation %d produced duplicate value", rot)
		}

		values[tile.Value] = true
	}
}

func TestGenerateDB(t *testing.T) {
	db := GenerateDB(42)

	if len(db.Records) != 100 {
		t.Fatalf("expected 100 records, got %d", len(db.Records))
	}

	// All hashes should be unique
	hashes := make(map[string]bool)

	for _, r := range db.Records {
		if hashes[r.Hash] {
			t.Fatalf("duplicate hash: %s", r.Hash)
		}

		hashes[r.Hash] = true
	}

	// Each record should have 100 pieces
	for _, r := range db.Records {
		if len(r.Pieces) != 100 {
			t.Fatalf("record %d: expected 100 pieces, got %d", r.ID, len(r.Pieces))
		}
	}

	// Hash should start with color letter
	for _, r := range db.Records {
		letter := ColorLetter(r.Color)
		if r.Hash[:1] != letter {
			t.Fatalf("record %d: hash %s should start with %s", r.ID, r.Hash, letter)
		}
	}
}

func TestDBLookupByHash(t *testing.T) {
	db := GenerateDB(42)
	target := db.Records[0]

	found := db.LookupByHash(target.Hash)
	if found == nil {
		t.Fatal("should find record by hash")
	}

	if found.ID != target.ID {
		t.Fatalf("expected ID %d, got %d", target.ID, found.ID)
	}

	notFound := db.LookupByHash("XNOTEXIST")
	if notFound != nil {
		t.Fatal("should not find nonexistent hash")
	}
}

func TestDBSaveLoad(t *testing.T) {
	db := GenerateDB(42)
	path := filepath.Join(t.TempDir(), "test_db.json")

	if err := db.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadDB(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded.Records) != 100 {
		t.Fatalf("loaded %d records, expected 100", len(loaded.Records))
	}

	if loaded.Records[0].Hash != db.Records[0].Hash {
		t.Fatal("loaded hash doesn't match saved hash")
	}
}

func TestComputeHashDeterministic(t *testing.T) {
	pieces := make([]PieceRecord, 100)

	for i := range pieces {
		pieces[i] = PieceRecord{X: i % 10, Y: i / 10, Value: uint32(i + 1)}
	}

	h1 := ComputeHash(pieces)
	h2 := ComputeHash(pieces)

	if h1 != h2 {
		t.Fatal("same pieces should produce same hash")
	}

	// Change one piece → different hash
	pieces[50].Value = 99999

	h3 := ComputeHash(pieces)

	if h1 == h3 {
		t.Fatal("different pieces should produce different hash")
	}
}

func TestGenerateCases(t *testing.T) {
	db := GenerateDB(42)
	cases := GenerateCases(db, 99)

	if len(cases) != 3 {
		t.Fatalf("expected 3 cases, got %d", len(cases))
	}

	// Case 1: 4-8 pieces, Case 2: 8-12, Case 3: 12-16
	for i, c := range cases {
		if c.TargetRecord == nil {
			t.Fatalf("case %d: no target record", i)
		}

		if len(c.MissingIndices) != c.PiecesToSolve {
			t.Fatalf("case %d: missing %d != piecesToSolve %d",
				i, len(c.MissingIndices), c.PiecesToSolve)
		}

		// Tray has missing pieces + decoys
		correctPieces := 0
		decoyPieces := 0

		for _, tp := range c.TrayPieces {
			if tp.IsDecoy {
				decoyPieces++
			} else {
				correctPieces++
			}
		}

		if correctPieces != c.PiecesToSolve {
			t.Fatalf("case %d: correct pieces %d != %d", i, correctPieces, c.PiecesToSolve)
		}

		if decoyPieces == 0 {
			t.Fatalf("case %d: no decoy pieces", i)
		}
	}

	// All cases should use different target records
	ids := make(map[int]bool)

	for _, c := range cases {
		if ids[c.TargetRecord.ID] {
			t.Fatal("cases should use different records")
		}

		ids[c.TargetRecord.ID] = true
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
