package domain

import (
	"math/rand/v2"
)

// CaseConfig holds the setup for one case with multiple puzzles.
type CaseConfig struct {
	ID      int
	Name    string          // case location name
	Puzzles []*PuzzleConfig // 5 puzzles per case
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
	Rotation      int // current rotation (0/1/2/3 = 0°/90°/180°/270°)
	IsDecoy       bool
	DecoyColor    string // source fingerprint color (for decoy image lookup)
	DecoyVariant  int    // source fingerprint variant
	DecoyPieceIdx int    // index in source fingerprint (for decoy image)
	IsPlaced      bool
	PlacedX       int // where player placed it (-1 if not placed)
	PlacedY       int
}

// GenerateCases creates 3 cases, each with 5 puzzles of increasing difficulty.
func GenerateCases(db *FingerprintDB, seed uint64) []*CaseConfig {
	rng := rand.New(rand.NewPCG(seed, seed^0xABCDEF01)) //nolint:gosec // game logic

	caseNames := []string{"MOTEL", "CAR WASH", "EDEN"}
	difficulties := [][2]int{{4, 8}, {8, 12}, {12, 16}}

	cases := make([]*CaseConfig, 3)
	usedRecords := make(map[int]bool)

	for i := range 3 {
		c := &CaseConfig{ID: i + 1, Name: caseNames[i]}

		for p := range 5 {
			puzzle := generatePuzzle(db, rng, difficulties[i], usedRecords, p)
			c.Puzzles = append(c.Puzzles, puzzle)
		}

		cases[i] = c
	}

	return cases
}

func generatePuzzle(db *FingerprintDB, rng *rand.Rand, diff [2]int, used map[int]bool, _ int) *PuzzleConfig {
	var target *FingerprintRecord

	for target == nil {
		idx := rng.IntN(len(db.Records))
		if !used[idx] {
			used[idx] = true
			target = &db.Records[idx]
		}
	}

	minP, maxP := diff[0], diff[1]
	piecesToSolve := minP + rng.IntN(maxP-minP+1)
	perm := rng.Perm(100)
	missingIndices := perm[:piecesToSolve]
	hideColor := rng.IntN(2) == 1

	var decoys []DecoyPiece

	for _, rec := range db.Records {
		if rec.ID == target.ID {
			continue
		}

		decoyPerm := rng.Perm(100)

		for j := range 5 {
			if j >= len(decoyPerm) {
				break
			}

			decoys = append(decoys, DecoyPiece{
				SourceRecordID: rec.ID,
				SourceColor:    rec.Color,
				SourceVariant:  rec.Variant,
				PieceIndex:     decoyPerm[j],
				Value:          rec.Pieces[decoyPerm[j]].Value,
			})
		}

		if len(decoys) >= 15 {
			break
		}
	}

	var tray []TrayPiece

	for _, idx := range missingIndices {
		p := target.Pieces[idx]

		tray = append(tray, TrayPiece{
			Value:     p.Value,
			OriginalX: p.X,
			OriginalY: p.Y,
			Rotation:  rng.IntN(4),
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
			Rotation:      rng.IntN(4),
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
