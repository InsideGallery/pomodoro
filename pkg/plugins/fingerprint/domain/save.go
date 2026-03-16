package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GameSave persists the game state across restarts.
type GameSave struct {
	Cases []CaseSave `json:"cases"`
}

// CaseSave stores one case's progress.
type CaseSave struct {
	CaseIndex    int          `json:"case_index"`
	ActivePuzzle int          `json:"active_puzzle"`
	Puzzles      []PuzzleSave `json:"puzzles"`
}

// PuzzleSave stores one puzzle's progress within a case.
type PuzzleSave struct {
	Solved       bool         `json:"solved"`
	Failed       bool         `json:"failed"`
	PlacedPieces []PlacedSave `json:"placed_pieces"`
}

// PlacedSave records where a tray piece was placed or positioned.
type PlacedSave struct {
	TrayIndex int     `json:"tray_index"`
	GridX     int     `json:"grid_x"`
	GridY     int     `json:"grid_y"`
	Rotation  int     `json:"rotation"`
	TrayX     float64 `json:"tray_x,omitempty"`
	TrayY     float64 `json:"tray_y,omitempty"`
}

// SaveGame writes game state to disk.
func SaveGame(save *GameSave, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	data, err := json.MarshalIndent(save, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	return os.WriteFile(path, data, 0o600)
}

// LoadGame reads game state from disk.
func LoadGame(path string) (*GameSave, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var save GameSave
	if err := json.Unmarshal(data, &save); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &save, nil
}

// DefaultSavePath returns the default path for save.json.
func DefaultSavePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "save.json"
	}

	return filepath.Join(home, ".config", "pomodoro", "fingerprint", "save.json")
}
