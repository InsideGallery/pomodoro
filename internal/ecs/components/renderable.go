package components

import "image/color"

// ShapeType defines what shape to draw.
type ShapeType int

const (
	ShapeCircle ShapeType = iota
	ShapeRect
	ShapeRoundedRect
)

// Renderable defines the visual appearance of an entity.
type Renderable struct {
	Shape  ShapeType
	Color  color.RGBA
	Width  float64
	Height float64
	Radius float64
}
