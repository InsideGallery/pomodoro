package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// DrawText draws text at (x, y) top-left with the given face and color.
func DrawText(dst *ebiten.Image, s string, face *textv2.GoTextFace, x, y float64, clr color.Color) {
	op := &textv2.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	textv2.Draw(dst, s, face, op)
}

// DrawTextCentered draws text horizontally centered at (cx, y).
func DrawTextCentered(dst *ebiten.Image, s string, face *textv2.GoTextFace, cx, y float64, clr color.Color) {
	w, _ := textv2.Measure(s, face, 0)
	DrawText(dst, s, face, cx-w/2, y, clr)
}

// MeasureText returns (width, height) of the given string.
func MeasureText(s string, face *textv2.GoTextFace) (float64, float64) {
	return textv2.Measure(s, face, 0)
}
