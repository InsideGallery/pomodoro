package domain

import (
	"encoding/json"
	"fmt"
	"hash/crc64"
	"math/rand/v2"
	"os"
	"path/filepath"
)

// DB constants.
const (
	NumColors   = 4
	NumVariants = 4
	NumMirror   = 2
)

// MaxFingerprints returns the total number of unique fingerprints:
// 4 colors × 4 variants × 8 rotations × 2 mirror = 256.
func MaxFingerprints() int {
	return NumColors * NumVariants * RotationSteps * NumMirror
}

// FingerprintRecord is a pre-generated fingerprint in the DB.
type FingerprintRecord struct {
	ID       int    `json:"id"`
	Color    string `json:"color"`    // "green", "red", "yellow", "blue"
	Variant  int    `json:"variant"`  // 1-4
	Rotation int    `json:"rotation"` // 0, 45, 90, 135, 180, 225, 270, 315
	Mirrored bool   `json:"mirrored"`
	Hash     string `json:"hash"` // "{LETTER}{CRC64}"

	// Piece data for verification
	Pieces []PieceRecord `json:"pieces"` // 100 pieces (10×10)

	// Person data
	PersonName string `json:"person_name"`
	AvatarKey  string `json:"avatar_key"` // "1"-"5"
}

// PieceRecord stores the uint32 value for one grid cell.
type PieceRecord struct {
	X     int    `json:"x"`     // 0-9
	Y     int    `json:"y"`     // 0-9
	Value uint32 `json:"value"` // encoded from (x, y)
}

// FingerprintDB is the full database of pre-generated fingerprints.
type FingerprintDB struct {
	Records []FingerprintRecord `json:"records"`
}

var crcTable = crc64.MakeTable(crc64.ECMA) //nolint:gochecknoglobals // CRC table

// Colors is the ordered list of fingerprint colors.
var Colors = []string{"green", "red", "yellow", "blue"} //nolint:gochecknoglobals // constant

// ColorLetter maps color name to hash prefix letter.
func ColorLetter(color string) string {
	switch color {
	case "green":
		return "G"
	case "red":
		return "R"
	case "yellow":
		return "Y"
	case "blue":
		return "B"
	default:
		return "?"
	}
}

// ComputeHash computes CRC64 hash from piece uint32 values.
func ComputeHash(pieces []PieceRecord) uint64 {
	h := crc64.New(crcTable)

	for _, p := range pieces {
		b := make([]byte, 4)
		b[0] = byte(p.Value)
		b[1] = byte(p.Value >> 8)
		b[2] = byte(p.Value >> 16)
		b[3] = byte(p.Value >> 24)

		h.Write(b)
	}

	return h.Sum64()
}

// ComputeFullHash returns "{LETTER}{CRC64}" string.
func ComputeFullHash(color string, pieces []PieceRecord) string {
	return fmt.Sprintf("%s%d", ColorLetter(color), ComputeHash(pieces))
}

// GenerateDB creates 256 deterministic fingerprint records:
// 4 colors × 4 variants × 8 rotations × 2 mirror.
func GenerateDB(seed uint64) *FingerprintDB {
	rng := rand.New(rand.NewPCG(seed, seed^0xDEADBEEF)) //nolint:gosec // game logic

	db := &FingerprintDB{}
	id := 0

	for ci := range NumColors {
		clr := Colors[ci]

		for v := 1; v <= NumVariants; v++ {
			for rot := range RotationSteps {
				for _, mirror := range []bool{false, true} {
					id++

					pieces := make([]PieceRecord, 100)
					for y := range 10 {
						for x := range 10 {
							idx := y*10 + x
							content := uint8(rng.IntN(255) + 1)
							value := EncodeTile(uint8(x), uint8(y), 0, content)
							pieces[idx] = PieceRecord{X: x, Y: y, Value: value}
						}
					}

					hash := ComputeFullHash(clr, pieces)
					character := CharacterForRecord(&FingerprintRecord{ID: id})
					personName := character.Name
					avatarKey := character.Avatar

					db.Records = append(db.Records, FingerprintRecord{
						ID:         id,
						Color:      clr,
						Variant:    v,
						Rotation:   RotationDegrees(rot),
						Mirrored:   mirror,
						Hash:       hash,
						Pieces:     pieces,
						PersonName: personName,
						AvatarKey:  avatarKey,
					})
				}
			}
		}
	}

	return db
}

// Save writes the DB to a JSON file.
func (db *FingerprintDB) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	return os.WriteFile(path, data, 0o600)
}

// LoadDB reads the DB from a JSON file.
func LoadDB(path string) (*FingerprintDB, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var db FingerprintDB
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &db, nil
}

// LookupByHash finds a record by full hash string.
func (db *FingerprintDB) LookupByHash(hash string) *FingerprintRecord {
	for i := range db.Records {
		if db.Records[i].Hash == hash {
			return &db.Records[i]
		}
	}

	return nil
}

// DefaultDBPath returns the default path for db.json.
func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "db.json"
	}

	return filepath.Join(home, ".config", "pomodoro", "fingerprint", "db.json")
}

// DefaultPuzzlesPath returns the path for puzzles.json (regeneratable).
func DefaultPuzzlesPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "puzzles.json"
	}

	return filepath.Join(home, ".config", "pomodoro", "fingerprint", "puzzles.json")
}
