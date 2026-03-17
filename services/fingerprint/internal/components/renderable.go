package components

// Renderable holds visual properties applied during rendering.
type Renderable struct {
	Rotation float64 // radians
	Alpha    float64 // 0.0 = invisible, 1.0 = opaque
	ZOrder   int     // higher = drawn later (on top)
}
