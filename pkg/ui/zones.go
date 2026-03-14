package ui

import (
	"github.com/InsideGallery/game-core/geometry/shapes"

	"github.com/InsideGallery/pomodoro/pkg/systems"
)

// ButtonZone creates an InputSystem Zone for a Button.
// The zone handles hover state and click-on-release behavior.
func ButtonZone(b *Button) *systems.Zone {
	return &systems.Zone{
		Spatial: shapes.NewBox(
			shapes.NewPoint(float64(b.X), float64(b.Y)),
			float64(b.W), float64(b.H),
		),
		OnClick: b.OnClick,
		OnHover: func(hovered bool) { b.hovered = hovered },
	}
}

// ToggleZone creates an InputSystem Zone for a Toggle.
// The hit area includes the label area (200px to the left).
func ToggleZone(t *Toggle) *systems.Zone {
	lx := float64(t.X) - 200
	if lx < 0 {
		lx = 0
	}

	w := float64(t.X+t.W) - lx

	return &systems.Zone{
		Spatial: shapes.NewBox(
			shapes.NewPoint(lx, float64(t.Y)-4),
			w, float64(t.H)+8,
		),
		OnClick: func() {
			t.Value = !t.Value
			if t.OnChange != nil {
				t.OnChange(t.Value)
			}
		},
	}
}

// SliderZone creates an InputSystem Zone for a Slider.
// Supports drag behavior for adjusting the value.
func SliderZone(s *Slider) *systems.Zone {
	pad := float64(12)

	return &systems.Zone{
		Spatial: shapes.NewBox(
			shapes.NewPoint(float64(s.X)-pad, float64(s.Y)-pad),
			float64(s.W)+pad*2, float64(s.H)+pad*2,
		),
		OnDragStart: func() {
			s.dragging = true
			s.dragMoved = false
		},
		OnDrag: func(mx int) {
			s.dragging = true

			if !s.dragMoved {
				s.dragMoved = true
			}

			t := float64(float32(mx)-s.X) / float64(s.W)
			if t < 0 {
				t = 0
			}

			if t > 1 {
				t = 1
			}

			newVal := s.Min + t*(s.Max-s.Min)
			if newVal != s.Value {
				s.Value = newVal
				if s.OnChange != nil {
					s.OnChange(s.Value)
				}
			}
		},
		OnDragEnd: func() {
			s.dragging = false
			s.dragMoved = false
		},
	}
}
