package scenes

import (
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"

	_ "image/png" // register PNG decoder

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
)

const (
	puzzleSize = 690 // fingerprint scaled to 690×690
	cellSize   = 69  // 690 / 10
)

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

	// Scale to 690×690
	scaled := scaleImage(srcImg, puzzleSize, puzzleSize)

	// Apply rotation
	rotated := rotateImage(scaled, rec.Rotation)

	// Apply mirror
	var final image.Image = rotated

	if rec.Mirrored {
		final = mirrorImage(rotated)
	}

	// Convert to ebiten
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

	scaled := scaleImage(srcImg, puzzleSize, puzzleSize)
	rotated := rotateImage(scaled, rotation)

	var final image.Image = rotated

	if mirrored {
		final = mirrorImage(rotated)
	}

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

func loadStdImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)

	return img, err
}

func scaleImage(src image.Image, w, h int) *image.RGBA {
	srcB := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	scaleX := float64(srcB.Dx()) / float64(w)
	scaleY := float64(srcB.Dy()) / float64(h)

	for y := range h {
		for x := range w {
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)

			if srcX >= srcB.Dx() {
				srcX = srcB.Dx() - 1
			}

			if srcY >= srcB.Dy() {
				srcY = srcB.Dy() - 1
			}

			dst.Set(x, y, src.At(srcB.Min.X+srcX, srcB.Min.Y+srcY))
		}
	}

	return dst
}

func rotateImage(src *image.RGBA, degrees int) *image.RGBA {
	degrees = ((degrees % 360) + 360) % 360
	if degrees == 0 {
		return src
	}

	w := src.Bounds().Dx()
	h := src.Bounds().Dy()

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

	return src
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
