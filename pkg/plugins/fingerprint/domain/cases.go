package domain

import (
	"math/rand/v2"
)

// CaseConfig holds the setup for one puzzle case.
type CaseConfig struct {
	ID            int
	Name          string // case location name
	TargetRecord  *FingerprintRecord
	PiecesToSolve int  // how many pieces removed
	HideColor     bool // if true, show grey fingerprint, colored pieces in tray

	// Generated puzzle data
	MissingIndices []int        // indices of removed pieces (0-99)
	DecoyPieces    []DecoyPiece // extra wrong pieces from other fingerprints
	TrayPieces     []TrayPiece  // all pieces in the tray (missing + decoys, shuffled)
}

// DecoyPiece is a wrong piece from a different fingerprint.
type DecoyPiece struct {
	SourceRecordID int
	PieceIndex     int    // 0-99
	Value          uint32 // the decoy's uint32
}

// TrayPiece is a piece shown in the tray for the player to drag.
type TrayPiece struct {
	Value     uint32
	OriginalX int // correct position (for correct pieces)
	OriginalY int
	Rotation  int // random initial rotation (0/1/2/3 = 0°/90°/180°/270°)
	IsDecoy   bool
	IsPlaced  bool // true after player places it on the grid
	PlacedX   int  // where player placed it (-1 if not placed)
	PlacedY   int
}

// GenerateCases creates 3 cases with increasing difficulty.
func GenerateCases(db *FingerprintDB, seed uint64) []*CaseConfig {
	rng := rand.New(rand.NewPCG(seed, seed^0xABCDEF01)) //nolint:gosec // game logic

	caseNames := []string{"MOTEL", "CAR WASH", "EDEN"}
	difficulties := [][2]int{{4, 8}, {8, 12}, {12, 16}} // min, max pieces to solve

	cases := make([]*CaseConfig, 3)

	usedRecords := make(map[int]bool)

	for i := range 3 {
		// Pick a target record (not already used)
		var target *FingerprintRecord

		for target == nil {
			idx := rng.IntN(len(db.Records))
			if !usedRecords[idx] {
				usedRecords[idx] = true
				target = &db.Records[idx]
			}
		}

		// Determine pieces to solve
		minP, maxP := difficulties[i][0], difficulties[i][1]
		piecesToSolve := minP + rng.IntN(maxP-minP+1)

		// Select which pieces to remove
		perm := rng.Perm(100)
		missingIndices := perm[:piecesToSolve]

		// Decide color visibility
		hideColor := rng.IntN(2) == 1

		// Generate decoy pieces from OTHER fingerprints
		var decoys []DecoyPiece

		for _, rec := range db.Records {
			if rec.ID == target.ID {
				continue
			}

			// Take 5 random pieces from this other fingerprint
			decoyPerm := rng.Perm(100)

			for j := range 5 {
				if j >= len(decoyPerm) {
					break
				}

				decoys = append(decoys, DecoyPiece{
					SourceRecordID: rec.ID,
					PieceIndex:     decoyPerm[j],
					Value:          rec.Pieces[decoyPerm[j]].Value,
				})
			}

			// Limit total decoys (5 per other variant of same color, ~15 total)
			if len(decoys) >= 15 {
				break
			}
		}

		// Build tray: missing pieces (correct) + decoys, shuffled
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
				Value:     d.Value,
				OriginalX: -1,
				OriginalY: -1,
				Rotation:  rng.IntN(4),
				IsDecoy:   true,
				PlacedX:   -1,
				PlacedY:   -1,
			})
		}

		// Shuffle tray
		rng.Shuffle(len(tray), func(a, b int) {
			tray[a], tray[b] = tray[b], tray[a]
		})

		cases[i] = &CaseConfig{
			ID:             i + 1,
			Name:           caseNames[i],
			TargetRecord:   target,
			PiecesToSolve:  piecesToSolve,
			HideColor:      hideColor,
			MissingIndices: missingIndices,
			DecoyPieces:    decoys,
			TrayPieces:     tray,
		}
	}

	return cases
}
