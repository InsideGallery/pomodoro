package systems

import (
	"image/color"

	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// SliderEntity represents a slider widget in the Registry.
type SliderEntity struct {
	Label       string
	Min, Max    float64
	Value       float64
	OnChange    func(float64)
	FormatValue func(float64) string
	TrackColor  color.Color
	KnobColor   color.Color

	// Layout (set by render system)
	X, Y, W, H float32
}

// ToggleEntity represents a toggle switch in the Registry.
type ToggleEntity struct {
	Label    string
	Value    bool
	OnChange func(bool)
	OnColor  color.Color
	OffColor color.Color

	// Layout (set by render system)
	X, Y, W, H float32
}

// SectionLabel is a section title in the settings.
type SectionLabel struct {
	Text  string
	Color color.RGBA

	// Layout (set by render system)
	Y float32
}

// SettingsButton is a button in the settings (Back, Reset).
type SettingsButton struct {
	Label      string
	OnClick    func()
	Color      color.Color
	HoverColor color.Color
	TextColor  color.Color
	IconDraw   any // func(*ebiten.Image, float32, float32, float32, color.Color)
	Face       *textv2.GoTextFace
	Fixed      bool // if true, not affected by scroll

	// Layout + state
	X, Y, W, H float32
	Hovered    bool
}
