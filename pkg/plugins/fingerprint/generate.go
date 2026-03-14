package fingerprint

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"
)

// PatternType defines the fingerprint pattern.
type PatternType int

const (
	PatternWhorl PatternType = iota
	PatternLoop
	PatternArch
)

// Fingerprint holds a generated fingerprint image and its pattern metadata.
type Fingerprint struct {
	Image   *image.RGBA
	Pattern PatternType
	Seed    uint64
	Size    int
}

// Generate creates a procedural fingerprint image from a seed.
// Same seed = same fingerprint (deterministic for matching).
func Generate(seed uint64, size int) Fingerprint {
	rng := rand.New(rand.NewPCG(seed, seed^0xDEADBEEF)) //nolint:gosec // game visuals
	pattern := PatternType(rng.IntN(3))

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Background — slightly off-white like paper
	bg := color.RGBA{R: 0xF5, G: 0xF0, B: 0xE8, A: 0xFF}

	for y := range size {
		for x := range size {
			img.Set(x, y, bg)
		}
	}

	cx, cy := float64(size)/2, float64(size)/2
	radius := float64(size) * 0.4

	// Draw fingerprint ridges
	ridgeColor := color.RGBA{R: 0x4A, G: 0x3A, B: 0x2A, A: 0xCC}

	switch pattern {
	case PatternWhorl:
		drawWhorl(img, cx, cy, radius, ridgeColor, rng)
	case PatternLoop:
		drawLoop(img, cx, cy, radius, ridgeColor, rng)
	case PatternArch:
		drawArch(img, cx, cy, radius, ridgeColor, rng)
	}

	// Add noise for realism
	addNoise(img, rng)

	return Fingerprint{Image: img, Pattern: pattern, Seed: seed, Size: size}
}

// GenerateSet creates n fingerprints with one guaranteed match to the target.
// Returns target index and the set.
func GenerateSet(targetSeed uint64, count, size int) (int, []Fingerprint) {
	rng := rand.New(rand.NewPCG(targetSeed+999, 0)) //nolint:gosec // game visuals
	target := Generate(targetSeed, size)

	results := make([]Fingerprint, count)
	matchIdx := rng.IntN(count)

	for i := range count {
		if i == matchIdx {
			results[i] = target
		} else {
			// Different seed = different fingerprint
			results[i] = Generate(targetSeed+uint64(i*7919)+1, size)
		}
	}

	return matchIdx, results
}

func drawWhorl(img *image.RGBA, cx, cy, radius float64, clr color.RGBA, rng *rand.Rand) {
	spirals := 5 + rng.IntN(4)
	offset := rng.Float64() * math.Pi

	for s := range spirals {
		angleOff := float64(s) * (2 * math.Pi / float64(spirals))

		for t := 0.0; t < radius*0.9; t += 0.5 {
			angle := offset + angleOff + t*0.15
			r := t
			x := cx + r*math.Cos(angle)
			y := cy + r*math.Sin(angle)

			drawThickPoint(img, x, y, 1.2, clr)
		}
	}
}

func drawLoop(img *image.RGBA, cx, cy, radius float64, clr color.RGBA, rng *rand.Rand) {
	loops := 6 + rng.IntN(4)
	skew := (rng.Float64() - 0.5) * 0.3

	for l := range loops {
		r := radius * float64(l+1) / float64(loops)

		for angle := -math.Pi * 0.8; angle < math.Pi*0.8; angle += 0.05 {
			x := cx + r*math.Cos(angle)*(1+skew)
			y := cy + r*math.Sin(angle)*0.6

			drawThickPoint(img, x, y, 1.0, clr)
		}
	}
}

func drawArch(img *image.RGBA, cx, cy, radius float64, clr color.RGBA, rng *rand.Rand) {
	arches := 7 + rng.IntN(4)
	tilt := (rng.Float64() - 0.5) * 0.2

	for a := range arches {
		r := radius * float64(a+1) / float64(arches)

		for angle := -math.Pi * 0.9; angle < math.Pi*0.9; angle += 0.04 {
			x := cx + r*math.Cos(angle+tilt)
			y := cy - r*math.Sin(angle)*0.5 + float64(a)*3

			drawThickPoint(img, x, y, 1.0, clr)
		}
	}
}

func drawThickPoint(img *image.RGBA, x, y, thickness float64, clr color.RGBA) {
	bounds := img.Bounds()
	t := int(math.Ceil(thickness))

	for dy := -t; dy <= t; dy++ {
		for dx := -t; dx <= t; dx++ {
			px := int(x) + dx
			py := int(y) + dy

			if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
				if float64(dx*dx+dy*dy) <= thickness*thickness {
					img.Set(px, py, clr)
				}
			}
		}
	}
}

func addNoise(img *image.RGBA, rng *rand.Rand) {
	bounds := img.Bounds()

	for range bounds.Dx() * bounds.Dy() / 20 {
		x := rng.IntN(bounds.Dx())
		y := rng.IntN(bounds.Dy())
		r, g, b, a := img.At(x, y).RGBA()

		noise := rng.IntN(20) - 10

		nr := clamp(int(r>>8) + noise)
		ng := clamp(int(g>>8) + noise)
		nb := clamp(int(b>>8) + noise)

		img.Set(x, y, color.RGBA{R: uint8(nr), G: uint8(ng), B: uint8(nb), A: uint8(a >> 8)})
	}
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}

	if v > 255 {
		return 255
	}

	return v
}
