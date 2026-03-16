package scenes

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"path/filepath"

	_ "image/png" // register PNG decoder

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
)

const (
	cropSize   = 480 // centered crop from source image
	puzzleSize = 690 // upscaled for puzzle grid
	cellSize   = 69  // 690 / 10
)

// largeCrop is the crop size for non-90° angles to ensure full coverage after rotation.
// For 45° rotation of a square s, inscribed square = s/sqrt(2).
// We need: inscribed >= puzzleSize, so s >= puzzleSize*sqrt(2) ≈ 976.
const largeCrop = 980

// FingerprintImages holds the cut pieces for a fingerprint.
type FingerprintImages struct {
	Pieces [100]*ebiten.Image // 10×10 grid of 69×69 images
	Full   *ebiten.Image      // full 690×690 image
}

// LoadFingerprintImages loads, scales, rotates, mirrors, and cuts a fingerprint.
func LoadFingerprintImages(assetsDir string, rec *domain.FingerprintRecord) (*FingerprintImages, error) {
	// Load the fingerprint image
	filename := fmt.Sprintf("%s.%d.png", rec.Color, rec.Variant)
	path := filepath.Join(assetsDir, "fingerprints", filename)

	srcImg, err := loadStdImage(path)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", path, err)
	}

	final := prepareFingerprint(srcImg, rec.Rotation, rec.Mirrored)
	fullImg := ebiten.NewImageFromImage(final)

	// Cut into 10×10 grid
	fi := &FingerprintImages{Full: fullImg}

	for y := range 10 {
		for x := range 10 {
			idx := y*10 + x
			rect := image.Rect(x*cellSize, y*cellSize, (x+1)*cellSize, (y+1)*cellSize)
			piece := image.NewRGBA(image.Rect(0, 0, cellSize, cellSize))

			draw.Draw(piece, piece.Bounds(), final, rect.Min, draw.Src)
			fi.Pieces[idx] = ebiten.NewImageFromImage(piece)
		}
	}

	return fi, nil
}

// LoadGreyFingerprintImages loads the grey version of a fingerprint.
func LoadGreyFingerprintImages(assetsDir string, variant int, rotation int, mirrored bool) (*FingerprintImages, error) {
	filename := fmt.Sprintf("grey.%d.png", variant)
	path := filepath.Join(assetsDir, "fingerprints", filename)

	srcImg, err := loadStdImage(path)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", path, err)
	}

	final := prepareFingerprint(srcImg, rotation, mirrored)
	fullImg := ebiten.NewImageFromImage(final)
	fi := &FingerprintImages{Full: fullImg}

	for y := range 10 {
		for x := range 10 {
			idx := y*10 + x
			rect := image.Rect(x*cellSize, y*cellSize, (x+1)*cellSize, (y+1)*cellSize)
			piece := image.NewRGBA(image.Rect(0, 0, cellSize, cellSize))

			draw.Draw(piece, piece.Bounds(), final, rect.Min, draw.Src)
			fi.Pieces[idx] = ebiten.NewImageFromImage(piece)
		}
	}

	return fi, nil
}

// cropCentered crops a centered cropW×cropH rectangle from the image.
func cropCentered(src image.Image, cropW, cropH int) *image.RGBA {
	b := src.Bounds()
	x0 := b.Min.X + (b.Dx()-cropW)/2
	y0 := b.Min.Y + (b.Dy()-cropH)/2

	cropped := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	draw.Draw(cropped, cropped.Bounds(), src, image.Pt(x0, y0), draw.Src)

	return cropped
}

// prepareFingerprint crops, scales, rotates and mirrors a source image into a 690×690 result.
// For 45° angles, crops larger to ensure full coverage after rotation.
func prepareFingerprint(srcImg image.Image, degrees int, mirrored bool) *image.RGBA {
	degrees = ((degrees % 360) + 360) % 360
	is45 := degrees%90 != 0

	// For 45° angles, crop larger from source so rotation doesn't leave gaps
	crop := cropSize
	if is45 {
		crop = largeCrop
	}

	cropped := cropCentered(srcImg, crop, crop)

	// Upscale to working size
	workSize := puzzleSize
	if is45 {
		workSize = int(math.Ceil(float64(puzzleSize) * math.Sqrt2))
	}

	upscaled := scaleImage(cropped, workSize, workSize)

	// Rotate
	rotated := rotateImage(upscaled, degrees)

	// Mirror
	var result image.Image = rotated
	if mirrored {
		result = mirrorImage(rotated)
	}

	// Crop center puzzleSize×puzzleSize from the (possibly larger) result
	return cropCentered(result, puzzleSize, puzzleSize)
}

// scaleImage resizes src to dstW×dstH using nearest-neighbor.
func scaleImage(src *image.RGBA, dstW, dstH int) *image.RGBA {
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))

	for y := range dstH {
		for x := range dstW {
			sx := x * srcW / dstW
			sy := y * srcH / dstH
			dst.Set(x, y, src.At(sx, sy))
		}
	}

	return dst
}

func loadStdImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)

	return img, err
}

func rotateImage(src *image.RGBA, degrees int) *image.RGBA {
	degrees = ((degrees % 360) + 360) % 360
	if degrees == 0 {
		return src
	}

	w := src.Bounds().Dx()
	h := src.Bounds().Dy()

	// Fast path for 90° increments
	switch degrees {
	case 90:
		dst := image.NewRGBA(image.Rect(0, 0, h, w))

		for y := range h {
			for x := range w {
				dst.Set(h-1-y, x, src.At(x, y))
			}
		}

		return dst
	case 180:
		dst := image.NewRGBA(image.Rect(0, 0, w, h))

		for y := range h {
			for x := range w {
				dst.Set(w-1-x, h-1-y, src.At(x, y))
			}
		}

		return dst
	case 270:
		dst := image.NewRGBA(image.Rect(0, 0, h, w))

		for y := range h {
			for x := range w {
				dst.Set(y, w-1-x, src.At(x, y))
			}
		}

		return dst
	}

	// General rotation for arbitrary angles (45°, 135°, etc.)
	rad := float64(degrees) * math.Pi / 180.0
	cosA := math.Cos(rad)
	sinA := math.Sin(rad)

	newW := int(math.Ceil(math.Abs(float64(w)*cosA) + math.Abs(float64(h)*sinA)))
	newH := int(math.Ceil(math.Abs(float64(w)*sinA) + math.Abs(float64(h)*cosA)))

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	cx := float64(w) / 2
	cy := float64(h) / 2
	ncx := float64(newW) / 2
	ncy := float64(newH) / 2

	for dy := range newH {
		for dx := range newW {
			fx := float64(dx) - ncx
			fy := float64(dy) - ncy
			sx := cosA*fx + sinA*fy + cx
			sy := -sinA*fx + cosA*fy + cy

			if sx >= 0 && sx < float64(w)-1 && sy >= 0 && sy < float64(h)-1 {
				dst.Set(dx, dy, src.At(int(sx), int(sy)))
			}
		}
	}

	return dst
}

func mirrorImage(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	for y := range b.Dy() {
		for x := range b.Dx() {
			dst.Set(b.Dx()-1-x, y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}

	return dst
}

// FindFingerprintAssetsDir finds the fingerprints directory.
func FindFingerprintAssetsDir() string {
	candidates := []string{
		"assets/external/fingerprint",
		"../assets/external/fingerprint",
		"../../assets/external/fingerprint",
	}

	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "assets", "external", "fingerprint"),
			filepath.Join(dir, "..", "assets", "external", "fingerprint"),
			filepath.Join(dir, "..", "..", "assets", "external", "fingerprint"),
		)
	}

	for _, p := range candidates {
		if _, err := os.Stat(filepath.Join(p, "fingerprints")); err == nil {
			return p
		}
	}

	return ""
}
