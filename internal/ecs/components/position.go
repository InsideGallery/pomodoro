package components

import "github.com/InsideGallery/game-core/geometry/shapes"

// Position is a 2D position component.
type Position struct {
	X, Y float64
}

// ToPoint converts to a shapes.Point for spatial operations.
func (p Position) ToPoint() shapes.Point {
	return shapes.NewPoint(p.X, p.Y)
}
