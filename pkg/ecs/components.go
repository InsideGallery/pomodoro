// Package ecs defines components and entity types for the ECS architecture.
// Components are pure data. Systems process them. Entities compose components.
package ecs

import (
	"image/color"
	"time"

	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ShapeType defines what shape to draw.
type ShapeType int

const (
	ShapeNone ShapeType = iota
	ShapeCircle
	ShapeRect
	ShapeRoundedRect
)

// TextAlign defines text alignment.
type TextAlign int

const (
	AlignLeft TextAlign = iota
	AlignCenter
)

// Position is a 2D position.
type Position struct{ X, Y float64 }

// Size is a 2D dimension.
type Size struct{ W, H float64 }

// Visual defines how an entity is drawn.
type Visual struct {
	Shape  ShapeType
	Color  color.RGBA
	Radius float64
}

// Label is a text label.
type Label struct {
	Text  string
	Face  *textv2.GoTextFace
	Color color.RGBA
	Align TextAlign
}

// Clickable marks an entity as clickable via RTree.
type Clickable struct {
	OnClick func()
}

// Draggable marks an entity as draggable via RTree.
type Draggable struct {
	OnDragStart func()
	OnDrag      func(mx, my int)
	OnDragEnd   func()
}

// Hoverable tracks hover state for visual feedback.
type Hoverable struct {
	Hovered bool
}

// RingProgress represents a circular progress indicator.
type RingProgress struct {
	Progress   float64
	Width      float64
	StartColor color.RGBA
	EndColor   color.RGBA
	TrackColor color.RGBA
}

// RoundDots represents round indicator dots.
type RoundDots struct {
	Total         int
	Completed     int
	Radius        float64
	ActiveColor   color.Color
	InactiveColor color.RGBA
	OnClickDot    func(index int)
}

// ButtonEntity is a clickable button with label or icon.
type ButtonEntity struct {
	Pos        Position
	Size       Size
	Color      color.Color
	HoverColor color.Color
	TextColor  color.Color
	Label      string
	Face       *textv2.GoTextFace
	IconDraw   any // func(*ebiten.Image, float32, float32, float32, color.Color)
	OnClick    func()
	Hovered    bool
}

// TimerTextEntity displays MM:SS from a timer.
type TimerTextEntity struct {
	Pos       Position
	Face      *textv2.GoTextFace
	Color     color.RGBA
	Remaining func() time.Duration
}

// ModeLabelEntity displays the current timer mode.
type ModeLabelEntity struct {
	Pos       Position
	Face      *textv2.GoTextFace
	ColorFunc func() color.Color
	TextFunc  func() string
}

// HintEntity displays contextual hint text.
type HintEntity struct {
	Pos      Position
	Face     *textv2.GoTextFace
	Color    color.RGBA
	TextFunc func() string
}
