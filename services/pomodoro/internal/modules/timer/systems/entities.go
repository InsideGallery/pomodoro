package systems

import "image/color"

// RingEntityData holds ring geometry for rendering + drag input.
type RingEntityData struct {
	CX, CY, OuterR float64
	TrackColor     color.RGBA
}

// RoundDotsEntityData holds dot center position.
type RoundDotsEntityData struct {
	CX, CY float64
}
