package core

import (
	"math"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/hajimehoshi/ebiten/v2"
)

// Camera provides world-space transformations: pan, zoom, rotate.
// Scenes can use Camera to render world content with transformations,
// then overlay UI in screen space via SystemWindow.ScreenDraw().
type Camera struct {
	Position   shapes.Point
	ViewPort   shapes.Point
	ZoomFactor float64
	Rotation   float64 // degrees
}

// NewCamera creates a camera at the given position.
func NewCamera(pos shapes.Point) *Camera {
	return &Camera{
		Position: pos,
	}
}

// SetViewPort sets the viewport dimensions (usually from Layout).
func (c *Camera) SetViewPort(w, h float64) {
	c.ViewPort = shapes.NewPoint(w, h)
}

// WorldMatrix returns the transformation matrix from world to screen space.
func (c *Camera) WorldMatrix() ebiten.GeoM {
	m := ebiten.GeoM{}

	m.Translate(-c.Position.Coordinate(0), -c.Position.Coordinate(1))

	vpCX := c.ViewPort.Coordinate(0) * 0.5
	vpCY := c.ViewPort.Coordinate(1) * 0.5

	m.Translate(-vpCX, -vpCY)
	m.Scale(
		math.Pow(1.01, c.ZoomFactor),
		math.Pow(1.01, c.ZoomFactor),
	)
	m.Rotate(c.Rotation * 2 * math.Pi / 360)
	m.Translate(vpCX, vpCY)

	return m
}

// ScreenToWorld converts screen coordinates to world coordinates.
func (c *Camera) ScreenToWorld(screenX, screenY float64) (float64, float64) {
	inv := c.WorldMatrix()
	if inv.IsInvertible() {
		inv.Invert()

		return inv.Apply(screenX, screenY)
	}

	return math.NaN(), math.NaN()
}

// Reset restores default position, zoom, and rotation.
func (c *Camera) Reset() {
	c.Position = shapes.NewPoint()
	c.ZoomFactor = 0
	c.Rotation = 0
}
