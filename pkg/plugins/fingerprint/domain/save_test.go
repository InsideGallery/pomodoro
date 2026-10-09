package domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadGameRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "save.json")

	want := &GameSave{Cases: []CaseSave{{
		CaseIndex:    2,
		ActivePuzzle: 1,
		Puzzles: []PuzzleSave{{
			Solved:       true,
			PlacedPieces: []PlacedSave{{TrayIndex: 3, GridX: 4, GridY: 5, Rotation: 6, TrayX: 1.5, TrayY: 2.5}},
		}},
	}}}

	if err := SaveGame(want, path); err != nil {
		t.Fatalf("SaveGame: %v", err)
	}

	got, err := LoadGame(path)
	if err != nil {
		t.Fatalf("LoadGame: %v", err)
	}

	if len(got.Cases) != 1 || got.Cases[0].CaseIndex != 2 || got.Cases[0].ActivePuzzle != 1 {
		t.Fatalf("cases = %+v", got.Cases)
	}

	p := got.Cases[0].Puzzles[0].PlacedPieces[0]
	if p != want.Cases[0].Puzzles[0].PlacedPieces[0] {
		t.Fatalf("placed piece = %+v", p)
	}
}

func TestLoadGameMissingFile(t *testing.T) {
	if _, err := LoadGame(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestLoadGameInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadGame(path); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestDefaultSavePath(t *testing.T) {
	if p := DefaultSavePath(); !strings.HasSuffix(p, filepath.Join("fingerprint", "save.json")) {
		t.Fatalf("DefaultSavePath() = %q", p)
	}
}
