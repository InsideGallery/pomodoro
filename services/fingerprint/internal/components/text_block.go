package components

import "image/color"

// TextBlock holds multi-line text for word-wrapped rendering.
type TextBlock struct {
	Text     string
	FontSize float64
	FontBold bool
	Color    color.RGBA
	Scroll   int // line scroll offset
}
