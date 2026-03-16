package domain

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
)

// CornerIndices are the 12 corner pieces excluded from puzzle selection.
// 3 per corner (L-shape): top-left, top-right, bottom-left, bottom-right.
var CornerIndices = map[int]bool{ //nolint:gochecknoglobals // constant
	0: true, 1: true, 10: true, // top-left
	8: true, 9: true, 19: true, // top-right
	80: true, 90: true, 91: true, // bottom-left
	89: true, 98: true, 99: true, // bottom-right
}

// ValidPieceIndices are the 88 non-corner indices (100 - 12 corners).
var ValidPieceIndices []int //nolint:gochecknoglobals // computed once

func init() {
	for i := range 100 {
		if !CornerIndices[i] {
			ValidPieceIndices = append(ValidPieceIndices, i)
		}
	}
}

// Game configuration constants.
const (
	NumCases       = 50
	PuzzlesPerCase = 20
	TotalPuzzles   = NumCases * PuzzlesPerCase
	DecoyGroups    = 2 // number of fake fingerprint groups added to tray
)

// Difficulty levels.
const (
	DiffEasy   = 3  // 3 missing pieces
	DiffMedium = 6  // 6 missing pieces
	DiffHard   = 12 // 12 missing pieces + grey color
)

// CaseConfig holds the setup for one case with multiple puzzles.
type CaseConfig struct {
	ID      int
	Name    string          // case location name
	Puzzles []*PuzzleConfig // PuzzlesPerCase puzzles
}

// PuzzleConfig holds one puzzle within a case.
type PuzzleConfig struct {
	TargetRecord  *FingerprintRecord
	PiecesToSolve int  // how many pieces removed
	HideColor     bool // if true, show grey fingerprint, colored pieces in tray

	MissingIndices []int        // indices of removed pieces (0-99)
	DecoyPieces    []DecoyPiece // extra wrong pieces from other fingerprints
	TrayPieces     []TrayPiece  // all pieces in the tray (missing + decoys, shuffled)
	Solved         bool
	Failed         bool
}

// DecoyPiece is a wrong piece from a different fingerprint.
type DecoyPiece struct {
	SourceRecordID int
	SourceColor    string
	SourceVariant  int
	PieceIndex     int    // 0-99
	Value          uint32 // the decoy's uint32
}

// TrayPiece is a piece shown in the tray for the player to drag.
type TrayPiece struct {
	Value         uint32
	OriginalX     int // correct position (for correct pieces, -1 for decoy)
	OriginalY     int
	Rotation      int // current rotation (0-7 = 0°/45°/90°/.../315°)
	IsDecoy       bool
	DecoyColor    string // source fingerprint color (for decoy image lookup)
	DecoyVariant  int    // source fingerprint variant
	DecoyPieceIdx int    // index in source fingerprint (for decoy image)
	IsPlaced      bool
	PlacedX       int // where player placed it (-1 if not placed)
	PlacedY       int
	TrayX         float64 // free-form position in tray (map coordinates)
	TrayY         float64
}

// caseDifficulty returns pieces-to-solve and whether color is hidden for a case index.
// 50 cases: first 20 EASY, next 15 MEDIUM, last 15 HARD.
func caseDifficulty(caseIdx int) (pieces int, forceGrey bool) {
	switch {
	case caseIdx < 20:
		return DiffEasy, false
	case caseIdx < 35:
		return DiffMedium, false
	default:
		return DiffHard, true
	}
}

// GenerateCases creates cases × puzzles from the 256 fingerprint DB.
func GenerateCases(db *FingerprintDB, seed uint64) []*CaseConfig {
	rng := rand.New(rand.NewPCG(seed, seed^0xABCDEF01)) //nolint:gosec // game logic

	numCases := CaseCount()
	if numCases > NumCases {
		numCases = NumCases
	}

	cases := make([]*CaseConfig, numCases)

	for i := range numCases {
		c := &CaseConfig{ID: i + 1, Name: CaseNameFromStory(i)}
		pieces, forceGrey := caseDifficulty(i)

		for range PuzzlesPerCase {
			puzzle := generatePuzzle(db, rng, pieces, forceGrey)
			c.Puzzles = append(c.Puzzles, puzzle)
		}

		cases[i] = c
	}

	return cases
}

func generatePuzzle(db *FingerprintDB, rng *rand.Rand, piecesToSolve int, forceGrey bool) *PuzzleConfig {
	target := &db.Records[rng.IntN(len(db.Records))]

	if piecesToSolve > len(ValidPieceIndices) {
		piecesToSolve = len(ValidPieceIndices)
	}

	// Shuffle valid indices and take the first piecesToSolve
	validCopy := make([]int, len(ValidPieceIndices))
	copy(validCopy, ValidPieceIndices)

	rng.Shuffle(len(validCopy), func(a, b int) {
		validCopy[a], validCopy[b] = validCopy[b], validCopy[a]
	})

	missingIndices := validCopy[:piecesToSolve]

	hideColor := forceGrey
	if !forceGrey {
		hideColor = rng.IntN(2) == 1
	}

	// 2 fake groups, each with exactly piecesToSolve pieces from a distinct fingerprint.
	// Total tray = piecesToSolve (real) + 2 * piecesToSolve (fake) = 3 * piecesToSolve.
	decoys := pickDecoys(db, rng, target, piecesToSolve)

	// Build tray: missing pieces + decoys, all with random rotation
	var tray []TrayPiece

	for _, idx := range missingIndices {
		p := target.Pieces[idx]

		tray = append(tray, TrayPiece{
			Value:     p.Value,
			OriginalX: p.X,
			OriginalY: p.Y,
			Rotation:  rng.IntN(RotationSteps),
			IsDecoy:   false,
			PlacedX:   -1,
			PlacedY:   -1,
		})
	}

	for _, d := range decoys {
		tray = append(tray, TrayPiece{
			Value:         d.Value,
			OriginalX:     -1,
			OriginalY:     -1,
			Rotation:      rng.IntN(RotationSteps),
			IsDecoy:       true,
			DecoyColor:    d.SourceColor,
			DecoyVariant:  d.SourceVariant,
			DecoyPieceIdx: d.PieceIndex,
			PlacedX:       -1,
			PlacedY:       -1,
		})
	}

	rng.Shuffle(len(tray), func(a, b int) {
		tray[a], tray[b] = tray[b], tray[a]
	})

	return &PuzzleConfig{
		TargetRecord:   target,
		PiecesToSolve:  piecesToSolve,
		HideColor:      hideColor,
		MissingIndices: missingIndices,
		DecoyPieces:    decoys,
		TrayPieces:     tray,
	}
}

// pickDecoys selects DecoyGroups (2) fake groups, each with exactly N pieces
// from one distinct fingerprint record (different variant from target, any color).
func pickDecoys(db *FingerprintDB, rng *rand.Rand, target *FingerprintRecord, n int) []DecoyPiece {
	// Collect candidates: different variant than target
	var candidates []FingerprintRecord

	for _, rec := range db.Records {
		if rec.ID != target.ID && rec.Variant != target.Variant {
			candidates = append(candidates, rec)
		}
	}

	rng.Shuffle(len(candidates), func(a, b int) {
		candidates[a], candidates[b] = candidates[b], candidates[a]
	})

	// Pick first DecoyGroups distinct records
	if len(candidates) > DecoyGroups {
		candidates = candidates[:DecoyGroups]
	}

	// From each source, pick exactly N random valid pieces
	var decoys []DecoyPiece

	for _, src := range candidates {
		validCopy := make([]int, len(ValidPieceIndices))
		copy(validCopy, ValidPieceIndices)

		rng.Shuffle(len(validCopy), func(a, b int) {
			validCopy[a], validCopy[b] = validCopy[b], validCopy[a]
		})

		count := n
		if count > len(validCopy) {
			count = len(validCopy)
		}

		for _, idx := range validCopy[:count] {
			decoys = append(decoys, DecoyPiece{
				SourceRecordID: src.ID,
				SourceColor:    src.Color,
				SourceVariant:  src.Variant,
				PieceIndex:     idx,
				Value:          src.Pieces[idx].Value,
			})
		}
	}

	return decoys
}

// PuzzlesSave is the persisted puzzle configuration (regeneratable).
type PuzzlesSave struct {
	Seed  uint64      `json:"seed"`
	Cases []CaseSaved `json:"cases"`
}

// CaseSaved is a case in the puzzle save file.
type CaseSaved struct {
	Name    string        `json:"name"`
	Puzzles []PuzzleSaved `json:"puzzles"`
}

// PuzzleSaved is a minimal puzzle reference for persistence.
type PuzzleSaved struct {
	TargetID       int   `json:"target_id"`
	PiecesToSolve  int   `json:"pieces_to_solve"`
	HideColor      bool  `json:"hide_color"`
	MissingIndices []int `json:"missing_indices"`
}

// SavePuzzles writes puzzle configs to disk (minimal form).
func SavePuzzles(cases []*CaseConfig, seed uint64, path string) error {
	save := PuzzlesSave{Seed: seed}

	for _, c := range cases {
		cs := CaseSaved{Name: c.Name}

		for _, p := range c.Puzzles {
			cs.Puzzles = append(cs.Puzzles, PuzzleSaved{
				TargetID:       p.TargetRecord.ID,
				PiecesToSolve:  p.PiecesToSolve,
				HideColor:      p.HideColor,
				MissingIndices: p.MissingIndices,
			})
		}

		save.Cases = append(save.Cases, cs)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	data, err := json.Marshal(save)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	return os.WriteFile(path, data, 0o600)
}

// LoadPuzzles reads puzzle configs and reconstructs full CaseConfigs from DB.
func LoadPuzzles(path string, db *FingerprintDB) ([]*CaseConfig, uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("read: %w", err)
	}

	var save PuzzlesSave
	if err := json.Unmarshal(data, &save); err != nil {
		return nil, 0, fmt.Errorf("unmarshal: %w", err)
	}

	recByID := make(map[int]*FingerprintRecord, len(db.Records))
	for i := range db.Records {
		recByID[db.Records[i].ID] = &db.Records[i]
	}

	rng := rand.New(rand.NewPCG(save.Seed, save.Seed^0xABCDEF01)) //nolint:gosec // game logic

	var cases []*CaseConfig

	for ci, cs := range save.Cases {
		c := &CaseConfig{ID: ci + 1, Name: cs.Name}

		for _, ps := range cs.Puzzles {
			target := recByID[ps.TargetID]
			if target == nil {
				continue
			}

			puzzle := reconstructPuzzle(db, rng, target, ps)
			c.Puzzles = append(c.Puzzles, puzzle)
		}

		cases = append(cases, c)
	}

	return cases, save.Seed, nil
}

func reconstructPuzzle(db *FingerprintDB, rng *rand.Rand, target *FingerprintRecord, ps PuzzleSaved) *PuzzleConfig {
	decoys := pickDecoys(db, rng, target, ps.PiecesToSolve)

	var tray []TrayPiece

	for _, idx := range ps.MissingIndices {
		p := target.Pieces[idx]

		tray = append(tray, TrayPiece{
			Value:     p.Value,
			OriginalX: p.X,
			OriginalY: p.Y,
			Rotation:  rng.IntN(RotationSteps),
			IsDecoy:   false,
			PlacedX:   -1,
			PlacedY:   -1,
		})
	}

	for _, d := range decoys {
		tray = append(tray, TrayPiece{
			Value:         d.Value,
			OriginalX:     -1,
			OriginalY:     -1,
			Rotation:      rng.IntN(RotationSteps),
			IsDecoy:       true,
			DecoyColor:    d.SourceColor,
			DecoyVariant:  d.SourceVariant,
			DecoyPieceIdx: d.PieceIndex,
			PlacedX:       -1,
			PlacedY:       -1,
		})
	}

	rng.Shuffle(len(tray), func(a, b int) {
		tray[a], tray[b] = tray[b], tray[a]
	})

	return &PuzzleConfig{
		TargetRecord:   target,
		PiecesToSolve:  ps.PiecesToSolve,
		HideColor:      ps.HideColor,
		MissingIndices: ps.MissingIndices,
		DecoyPieces:    decoys,
		TrayPieces:     tray,
	}
}
