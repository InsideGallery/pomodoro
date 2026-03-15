package scenes

import "github.com/hajimehoshi/ebiten/v2"

// CRT screen area as percentage of the 8328x4320 background image.
const (
	CRTLeft   = 0.265
	CRTTop    = 0.095
	CRTRight  = 0.735
	CRTBottom = 0.875
)

// drawFit draws an image scaled to fit within w×h, preserving aspect ratio, centered.
func drawFit(dst, src *ebiten.Image, w, h, alpha float64) {
	if src == nil || alpha <= 0 {
		return
	}

	bw := float64(src.Bounds().Dx())
	bh := float64(src.Bounds().Dy())

	scale := w / bw
	if h/bh < scale {
		scale = h / bh
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate((w-bw*scale)/2, (h-bh*scale)/2)

	if alpha < 1 {
		op.ColorScale.Scale(float32(alpha), float32(alpha), float32(alpha), float32(alpha))
	}

	dst.DrawImage(src, op)
}
