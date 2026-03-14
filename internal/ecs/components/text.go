package components

import (
	"image/color"

	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Text is a dynamic text label component.
type Text struct {
	Content string
	Face    *textv2.GoTextFace
	Color   color.RGBA
}
