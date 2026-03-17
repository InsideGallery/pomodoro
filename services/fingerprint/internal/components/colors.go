package components

import "image/color"

// DefaultTextColor returns the standard text color.
func DefaultTextColor() color.RGBA {
	return color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF}
}
