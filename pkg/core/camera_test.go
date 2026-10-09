package core

import (
	"math"
	"testing"

	"github.com/InsideGallery/game-core/geometry/shapes"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestCameraScreenToWorldIdentity(t *testing.T) {
	c := NewCamera(shapes.NewPoint(0, 0))
	c.SetViewPort(800, 600)

	x, y := c.ScreenToWorld(100, 200)
	if !near(x, 100) || !near(y, 200) {
		t.Fatalf("ScreenToWorld = (%v, %v), want (100, 200)", x, y)
	}
}

func TestCameraScreenToWorldPanned(t *testing.T) {
	c := NewCamera(shapes.NewPoint(50, 30))
	c.SetViewPort(800, 600)

	x, y := c.ScreenToWorld(0, 0)
	if !near(x, 50) || !near(y, 30) {
		t.Fatalf("ScreenToWorld = (%v, %v), want (50, 30)", x, y)
	}
}

func TestCameraZoomKeepsViewportCentre(t *testing.T) {
	c := NewCamera(shapes.NewPoint(0, 0))
	c.SetViewPort(800, 600)
	c.ZoomFactor = 50
	c.Rotation = 90

	x, y := c.ScreenToWorld(400, 300)
	if !near(x, 400) || !near(y, 300) {
		t.Fatalf("centre maps to (%v, %v), want (400, 300)", x, y)
	}
}

func TestCameraReset(t *testing.T) {
	c := NewCamera(shapes.NewPoint(10, 20))
	c.ZoomFactor = 5
	c.Rotation = 45

	c.Reset()

	if c.ZoomFactor != 0 || c.Rotation != 0 || c.Position.Coordinate(0) != 0 || c.Position.Coordinate(1) != 0 {
		t.Fatalf("after Reset: %+v", c)
	}
}
