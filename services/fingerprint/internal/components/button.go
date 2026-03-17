package components

import "image/color"

// Button is a clickable UI element with label text.
type Button struct {
	Label    string
	BgColor  color.RGBA
	TxtColor color.RGBA
	FontBold bool
	FontSize float64
	OnClick  string // event name emitted on click (not a callback — systems handle it)
}
