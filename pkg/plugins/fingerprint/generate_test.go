package fingerprint

import (
	"testing"
)

func TestGenerateDeterministic(t *testing.T) {
	fp1 := Generate(42, 128)
	fp2 := Generate(42, 128)

	if fp1.Pattern != fp2.Pattern {
		t.Fatal("same seed should produce same pattern")
	}

	if fp1.Seed != fp2.Seed {
		t.Fatal("seed should match")
	}

	// Compare a sample of pixels
	for y := 10; y < 50; y++ {
		for x := 10; x < 50; x++ {
			r1, g1, b1, _ := fp1.Image.At(x, y).RGBA()
			r2, g2, b2, _ := fp2.Image.At(x, y).RGBA()

			if r1 != r2 || g1 != g2 || b1 != b2 {
				t.Fatalf("pixel mismatch at (%d,%d): same seed should produce identical image", x, y)
			}
		}
	}
}

func TestGenerateDifferentSeeds(t *testing.T) {
	fp1 := Generate(42, 128)
	fp2 := Generate(99, 128)

	// Different seeds should produce different images (statistically)
	diffCount := 0

	for y := 10; y < 100; y++ {
		for x := 10; x < 100; x++ {
			r1, _, _, _ := fp1.Image.At(x, y).RGBA()
			r2, _, _, _ := fp2.Image.At(x, y).RGBA()

			if r1 != r2 {
				diffCount++
			}
		}
	}

	if diffCount == 0 {
		t.Fatal("different seeds should produce different images")
	}
}

func TestGenerateSetContainsMatch(t *testing.T) {
	matchIdx, results := GenerateSet(42, 6, 128)

	if matchIdx < 0 || matchIdx >= 6 {
		t.Fatalf("match index %d out of range [0,6)", matchIdx)
	}

	if len(results) != 6 {
		t.Fatalf("expected 6 results, got %d", len(results))
	}

	// The target and the match should have the same seed
	target := Generate(42, 128)

	if results[matchIdx].Seed != target.Seed {
		t.Fatal("match should have same seed as target")
	}
}

func TestGenerateImageSize(t *testing.T) {
	fp := Generate(1, 200)

	bounds := fp.Image.Bounds()
	if bounds.Dx() != 200 || bounds.Dy() != 200 {
		t.Fatalf("expected 200x200, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestAllPatternTypes(t *testing.T) {
	patterns := make(map[PatternType]bool)

	// Try many seeds to hit all pattern types
	for seed := uint64(0); seed < 100; seed++ {
		fp := Generate(seed, 64)
		patterns[fp.Pattern] = true
	}

	if len(patterns) < 3 {
		t.Fatalf("expected 3 pattern types, got %d", len(patterns))
	}
}
