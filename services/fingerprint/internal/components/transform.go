package components

// Transform holds position and size in map coordinates.
// Systems multiply by ScaleX/ScaleY to convert to screen pixels.
type Transform struct {
	X float64 // map X
	Y float64 // map Y
	W float64 // width in map units
	H float64 // height in map units
}
