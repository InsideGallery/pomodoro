package fingerprint

import (
	"image"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// TileCut represents a single cut piece from a fingerprint image.
type TileCut struct {
	Image    *ebiten.Image
	GridX    int // position in the grid
	GridY    int
	Rotation int // 0-3 (applied by player)
}

// CutTiles splits a source image into a gridW x gridH grid of tiles.
// Each tile is a rectangular sub-image. No pre-cut files needed.
func CutTiles(src *ebiten.Image, gridW, gridH int) []TileCut {
	bounds := src.Bounds()
	tileW := bounds.Dx() / gridW
	tileH := bounds.Dy() / gridH

	// Convert ebiten.Image to standard image for sub-imaging
	srcStd := src

	tiles := make([]TileCut, 0, gridW*gridH)

	for y := range gridH {
		for x := range gridW {
			rect := image.Rect(x*tileW, y*tileH, (x+1)*tileW, (y+1)*tileH)

			tileImg := ebiten.NewImage(tileW, tileH)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(-rect.Min.X), float64(-rect.Min.Y))
			tileImg.DrawImage(srcStd, op)

			tiles = append(tiles, TileCut{
				Image: tileImg,
				GridX: x,
				GridY: y,
			})
		}
	}

	return tiles
}

// CutTilesFromStd splits a standard image.Image into tiles as *ebiten.Image.
func CutTilesFromStd(src image.Image, gridW, gridH int) []TileCut {
	bounds := src.Bounds()
	tileW := bounds.Dx() / gridW
	tileH := bounds.Dy() / gridH

	tiles := make([]TileCut, 0, gridW*gridH)

	for y := range gridH {
		for x := range gridW {
			rect := image.Rect(
				bounds.Min.X+x*tileW, bounds.Min.Y+y*tileH,
				bounds.Min.X+(x+1)*tileW, bounds.Min.Y+(y+1)*tileH,
			)

			tileRGBA := image.NewRGBA(image.Rect(0, 0, tileW, tileH))
			draw.Draw(tileRGBA, tileRGBA.Bounds(), src, rect.Min, draw.Src)

			tiles = append(tiles, TileCut{
				Image: ebiten.NewImageFromImage(tileRGBA),
				GridX: x,
				GridY: y,
			})
		}
	}

	return tiles
}

// RotateTileImage returns a new image rotated by 90° * rotation clockwise.
func RotateTileImage(src *ebiten.Image, rotation int) *ebiten.Image {
	rotation %= 4
	if rotation == 0 {
		return src
	}

	w := src.Bounds().Dx()
	h := src.Bounds().Dy()

	var dstW, dstH int

	switch rotation {
	case 1, 3:
		dstW, dstH = h, w
	default:
		dstW, dstH = w, h
	}

	dst := ebiten.NewImage(dstW, dstH)
	op := &ebiten.DrawImageOptions{}

	// Move origin to center, rotate, move back
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)

	switch rotation {
	case 1:
		op.GeoM.Rotate(1.5707963) // 90°
	case 2:
		op.GeoM.Rotate(3.1415926) // 180°
	case 3:
		op.GeoM.Rotate(4.7123889) // 270°
	}

	op.GeoM.Translate(float64(dstW)/2, float64(dstH)/2)
	dst.DrawImage(src, op)

	return dst
}

// BuildSpriteSheet combines multiple images into a single horizontal spritesheet.
// Returns the sheet image and the width of each frame.
func BuildSpriteSheet(frames []*ebiten.Image) (*ebiten.Image, int) {
	if len(frames) == 0 {
		return nil, 0
	}

	frameW := frames[0].Bounds().Dx()
	frameH := frames[0].Bounds().Dy()
	sheetW := frameW * len(frames)

	sheet := ebiten.NewImage(sheetW, frameH)

	for i, f := range frames {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(i*frameW), 0)
		sheet.DrawImage(f, op)
	}

	return sheet, frameW
}

// DrawSpriteFrame draws a single frame from a horizontal spritesheet.
func DrawSpriteFrame(dst, sheet *ebiten.Image, frameW, frameIdx int, x, y float64) {
	sx := frameIdx * frameW
	rect := image.Rect(sx, 0, sx+frameW, sheet.Bounds().Dy())
	frame := sheet.SubImage(rect).(*ebiten.Image)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	dst.DrawImage(frame, op)
}
