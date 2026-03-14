package fingerprint

import (
	"image"
	"image/color"
	"testing"
)

func TestCutTilesFromStd(t *testing.T) {
	// Create a 100x100 test image with distinct quadrants
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Top-left red, top-right green, bottom-left blue, bottom-right white
	for y := range 100 {
		for x := range 100 {
			switch {
			case x < 50 && y < 50:
				img.Set(x, y, color.RGBA{R: 255, A: 255})
			case x >= 50 && y < 50:
				img.Set(x, y, color.RGBA{G: 255, A: 255})
			case x < 50 && y >= 50:
				img.Set(x, y, color.RGBA{B: 255, A: 255})
			default:
				img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}

	tiles := CutTilesFromStd(img, 2, 2)

	if len(tiles) != 4 {
		t.Fatalf("expected 4 tiles, got %d", len(tiles))
	}

	// Check grid positions
	if tiles[0].GridX != 0 || tiles[0].GridY != 0 {
		t.Fatal("tile 0 should be at (0,0)")
	}

	if tiles[1].GridX != 1 || tiles[1].GridY != 0 {
		t.Fatal("tile 1 should be at (1,0)")
	}

	if tiles[2].GridX != 0 || tiles[2].GridY != 1 {
		t.Fatal("tile 2 should be at (0,1)")
	}

	if tiles[3].GridX != 1 || tiles[3].GridY != 1 {
		t.Fatal("tile 3 should be at (1,1)")
	}

	// Each tile should be 50x50
	for i, tile := range tiles {
		b := tile.Image.Bounds()
		if b.Dx() != 50 || b.Dy() != 50 {
			t.Fatalf("tile %d: expected 50x50, got %dx%d", i, b.Dx(), b.Dy())
		}
	}
}

func TestCutTilesGridSizes(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 120))

	tiles3x3 := CutTilesFromStd(img, 3, 3)
	if len(tiles3x3) != 9 {
		t.Fatalf("3x3: expected 9 tiles, got %d", len(tiles3x3))
	}

	tiles10x10 := CutTilesFromStd(img, 10, 10)
	if len(tiles10x10) != 100 {
		t.Fatalf("10x10: expected 100 tiles, got %d", len(tiles10x10))
	}
}

func TestRotateTileImage(t *testing.T) {
	// 20x10 image
	src := image.NewRGBA(image.Rect(0, 0, 20, 10))

	tiles := CutTilesFromStd(src, 1, 1)
	if len(tiles) != 1 {
		t.Fatal("expected 1 tile")
	}

	// Rotate 90° — should be 10x20
	rot := RotateTileImage(tiles[0].Image, 1)
	if rot.Bounds().Dx() != 10 || rot.Bounds().Dy() != 20 {
		t.Fatalf("90° rotation of 20x10 should be 10x20, got %dx%d",
			rot.Bounds().Dx(), rot.Bounds().Dy())
	}

	// Rotate 0° — should be same
	same := RotateTileImage(tiles[0].Image, 0)
	if same != tiles[0].Image {
		t.Fatal("0° rotation should return same image")
	}
}
