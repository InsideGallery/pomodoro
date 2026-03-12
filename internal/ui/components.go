package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// DrawText draws text at (x, y) top-left with the given face and color.
func DrawText(dst *ebiten.Image, s string, face *textv2.GoTextFace, x, y float64, clr color.Color) {
	op := &textv2.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	textv2.Draw(dst, s, face, op)
}

// DrawTextCentered draws text horizontally centered at (cx, y).
func DrawTextCentered(dst *ebiten.Image, s string, face *textv2.GoTextFace, cx, y float64, clr color.Color) {
	w, _ := textv2.Measure(s, face, 0)
	DrawText(dst, s, face, cx-w/2, y, clr)
}

// MeasureText returns (width, height) of the given string.
func MeasureText(s string, face *textv2.GoTextFace) (float64, float64) {
	return textv2.Measure(s, face, 0)
}

// Button is a clickable rounded rectangle with a label.
type Button struct {
	X, Y, W, H float32
	Label      string
	Color      color.Color
	HoverColor color.Color
	TextColor  color.Color
	Face       *textv2.GoTextFace
	OnClick    func()
	IconDraw   func(dst *ebiten.Image, cx, cy, size float32, clr color.Color) // optional icon instead of label

	hovered bool
	pressed bool
}

func (b *Button) Update() {
	mx, my := ebiten.CursorPosition()
	b.hovered = b.hit(mx, my)

	if b.hovered && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		b.pressed = true
	}

	if b.pressed && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		b.pressed = false
		if b.hovered && b.OnClick != nil {
			b.OnClick()
		}
	}

	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		b.pressed = false
	}
}

func (b *Button) Draw(dst *ebiten.Image) {
	clr := b.Color
	if b.hovered {
		clr = b.HoverColor
	}

	if b.pressed {
		clr = colorBrighten(clr, 0.8)
	}

	DrawRoundedRect(dst, b.X, b.Y, b.W, b.H, RadiusButton, clr)

	if b.IconDraw != nil {
		b.IconDraw(dst, b.X+b.W/2, b.Y+b.H/2, b.H*0.5, b.TextColor)
		return
	}

	if b.Face != nil && b.Label != "" {
		tw, th := textv2.Measure(b.Label, b.Face, 0)
		tx := float64(b.X) + (float64(b.W)-tw)/2
		ty := float64(b.Y) + (float64(b.H)-th)/2
		DrawText(dst, b.Label, b.Face, tx, ty, b.TextColor)
	}
}

func (b *Button) hit(mx, my int) bool {
	r := image.Rect(int(b.X), int(b.Y), int(b.X+b.W), int(b.Y+b.H))
	return image.Pt(mx, my).In(r)
}

// Slider is a horizontal slider for float64 values.
type Slider struct {
	X, Y, W, H  float32
	Min, Max    float64
	Value       float64
	TrackColor  color.Color
	KnobColor   color.Color
	Label       string
	Face        *textv2.GoTextFace
	TextColor   color.Color
	OnChange    func(float64)
	FormatValue func(float64) string // custom value formatter

	dragging bool
}

func (s *Slider) Update() {
	mx, my := ebiten.CursorPosition()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if s.hitTrack(mx, my) {
			s.dragging = true
		}
	}

	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.dragging = false
	}

	if s.dragging {
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
	}
}

func (s *Slider) Draw(dst *ebiten.Image) {
	trackH := s.H * 0.35
	trackY := s.Y + (s.H-trackH)/2

	// Label on the left above
	if s.Face != nil && s.Label != "" {
		DrawText(dst, s.Label, s.Face, float64(s.X), float64(s.Y)-16, s.TextColor)
	}

	// Value on the right above
	if s.Face != nil {
		valStr := s.formatVal()
		vw, _ := textv2.Measure(valStr, s.Face, 0)
		DrawText(dst, valStr, s.Face, float64(s.X+s.W)-vw, float64(s.Y)-16, s.TextColor)
	}

	// Track background
	DrawRoundedRect(dst, s.X, trackY, s.W, trackH, trackH/2, s.TrackColor)

	// Filled portion
	t := float32(0)
	if s.Max > s.Min {
		t = float32((s.Value - s.Min) / (s.Max - s.Min))
	}

	filledW := s.W * t
	if filledW > trackH {
		DrawRoundedRect(dst, s.X, trackY, filledW, trackH, trackH/2, s.KnobColor)
	}

	// Knob
	knobR := s.H * 0.3
	knobX := s.X + filledW
	knobY := s.Y + s.H/2
	DrawCircle(dst, knobX, knobY, knobR+1, ColorBgPrimary) // border
	DrawCircle(dst, knobX, knobY, knobR, s.KnobColor)
}

func (s *Slider) formatVal() string {
	if s.FormatValue != nil {
		return s.FormatValue(s.Value)
	}

	return fmt.Sprintf("%.0f%%", (s.Value-s.Min)/(s.Max-s.Min)*100)
}

func (s *Slider) hitTrack(mx, my int) bool {
	pad := float32(12)
	r := image.Rect(int(s.X-pad), int(s.Y-pad), int(s.X+s.W+pad), int(s.Y+s.H+pad))

	return image.Pt(mx, my).In(r)
}

// Toggle is an on/off switch.
type Toggle struct {
	X, Y, W, H float32
	Value      bool
	OnColor    color.Color
	OffColor   color.Color
	KnobColor  color.Color
	Label      string
	Face       *textv2.GoTextFace
	TextColor  color.Color
	OnChange   func(bool)
}

func (t *Toggle) Update() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		// Hit area includes the label
		lx := int(t.X) - 200
		if lx < 0 {
			lx = 0
		}

		r := image.Rect(lx, int(t.Y)-4, int(t.X+t.W), int(t.Y+t.H)+4)
		if image.Pt(mx, my).In(r) {
			t.Value = !t.Value
			if t.OnChange != nil {
				t.OnChange(t.Value)
			}
		}
	}
}

func (t *Toggle) Draw(dst *ebiten.Image) {
	// Label to the left
	if t.Face != nil && t.Label != "" {
		_, th := textv2.Measure(t.Label, t.Face, 0)

		lx := float64(t.X) - 200
		if lx < 0 {
			lx = 0
		}

		ly := float64(t.Y) + (float64(t.H)-th)/2
		DrawText(dst, t.Label, t.Face, lx, ly, t.TextColor)
	}

	trackColor := t.OffColor
	if t.Value {
		trackColor = t.OnColor
	}

	DrawRoundedRect(dst, t.X, t.Y, t.W, t.H, t.H/2, trackColor)

	knobR := t.H * 0.38

	knobX := t.X + knobR + 3
	if t.Value {
		knobX = t.X + t.W - knobR - 3
	}

	knobY := t.Y + t.H/2
	DrawCircle(dst, knobX, knobY, knobR, t.KnobColor)
}

// DurationSlider is a slider that displays value in minutes.
type DurationSlider struct {
	Slider
}

func (d *DurationSlider) Minutes() int {
	return int(math.Round(d.Value))
}

func colorBrighten(c color.Color, factor float64) color.Color {
	r, g, b, a := c.RGBA()

	return color.RGBA{
		R: clampByte(float64(r>>8) * factor),
		G: clampByte(float64(g>>8) * factor),
		B: clampByte(float64(b>>8) * factor),
		A: uint8(a >> 8),
	}
}

func colorWithAlpha(c color.Color, alpha float64) color.Color {
	r, g, b, _ := c.RGBA()

	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: clampByte(alpha * 255),
	}
}

func clampByte(v float64) uint8 {
	if v > 255 {
		return 255
	}

	if v < 0 {
		return 0
	}

	return uint8(v)
}
