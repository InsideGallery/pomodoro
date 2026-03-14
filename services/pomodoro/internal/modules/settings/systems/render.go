package systems

import (
	"context"
	"fmt"
	"image"
	"image/color"

	"github.com/InsideGallery/core/memory/registry"
	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/pkg/ui"
)

// RenderSystem draws all settings entities from the Registry.
type RenderSystem struct {
	Reg    *registry.Registry[string, uint64, any]
	Scroll *ScrollSystem
	Width  int
	Height int
}

func (s *RenderSystem) Update(_ context.Context) error { return nil }

func (s *RenderSystem) Draw(_ context.Context, screen *ebiten.Image) {
	w := float32(s.Width)
	h := float32(s.Height)
	pad := ui.S(24)
	cardW := w - pad*2
	cardY := ui.S(48)
	cardH := h - cardY - pad

	// Card background
	ui.DrawRoundedRect(screen, pad, cardY, cardW, cardH, ui.S(ui.RadiusCard), ui.ColorCardBg)
	ui.DrawRoundedRectStroke(screen, pad, cardY, cardW, cardH, ui.S(ui.RadiusCard), ui.S(1), ui.ColorCardBorder)

	// Fixed header: back button + title
	faceHeading := ui.Face(true, 16)
	ui.DrawTextCentered(screen, "Settings", faceHeading, float64(w/2), ui.Sf(16), ui.ColorTextPrimary)

	for btn := range s.Reg.Iterator("fixed_button") {
		b, ok := btn.(*SettingsButton)
		if !ok {
			continue
		}

		s.drawButton(screen, b)
	}

	// Clipped scrollable area
	contentTop := s.Scroll.ContentTop
	clipRect := image.Rect(
		int(pad+ui.S(1)), int(contentTop),
		int(w-pad-ui.S(1)), int(h-pad-ui.S(1)),
	)
	clip := screen.SubImage(clipRect).(*ebiten.Image)

	dy := contentTop - s.Scroll.ScrollY

	// Section labels
	faceSection := ui.Face(true, 10)
	titleX := float64(pad) + ui.Sf(12)

	for sec := range s.Reg.Iterator("section") {
		sl, ok := sec.(*SectionLabel)
		if !ok {
			continue
		}

		ui.DrawText(clip, sl.Text, faceSection, titleX, float64(sl.Y+dy), sl.Color)
	}

	// Sliders
	faceLabel := ui.Face(false, 12)

	for sl := range s.Reg.Iterator("slider") {
		slider, ok := sl.(*SliderEntity)
		if !ok {
			continue
		}

		s.drawSlider(clip, slider, dy, faceLabel)
	}

	// Toggles
	for tg := range s.Reg.Iterator("toggle") {
		toggle, ok := tg.(*ToggleEntity)
		if !ok {
			continue
		}

		s.drawToggle(clip, toggle, dy, faceLabel)
	}

	// Scrollable buttons (Reset Defaults)
	for btn := range s.Reg.Iterator("button") {
		b, ok := btn.(*SettingsButton)
		if !ok {
			continue
		}

		s.drawButtonOffset(clip, b, dy)
	}

	// Scroll indicator
	if mx := s.Scroll.maxScroll(); mx > 0 {
		visH := s.Scroll.ViewportH

		barH := visH * visH / s.Scroll.ContentH
		if barH < ui.S(20) {
			barH = ui.S(20)
		}

		barY := contentTop + (visH-barH)*(s.Scroll.ScrollY/mx)
		barX := w - pad - ui.S(4)

		ui.DrawRoundedRect(screen, barX, barY, ui.S(3), barH, ui.S(2), ui.ColorBorder)
	}
}

func (s *RenderSystem) drawButton(screen *ebiten.Image, b *SettingsButton) {
	clr := b.Color
	if b.Hovered {
		clr = b.HoverColor
	}

	ui.DrawRoundedRect(screen, b.X, b.Y, b.W, b.H, ui.RadiusButton, clr)

	if b.IconDraw != nil {
		if fn, ok := b.IconDraw.(func(*ebiten.Image, float32, float32, float32, color.Color)); ok { //nolint:lll // type assertion
			fn(screen, b.X+b.W/2, b.Y+b.H/2, b.H*0.5, b.TextColor)
		}
	} else if b.Face != nil && b.Label != "" {
		tw, th := textv2.Measure(b.Label, b.Face, 0)
		tx := float64(b.X) + (float64(b.W)-tw)/2
		ty := float64(b.Y) + (float64(b.H)-th)/2

		ui.DrawText(screen, b.Label, b.Face, tx, ty, b.TextColor)
	}
}

func (s *RenderSystem) drawButtonOffset(clip *ebiten.Image, b *SettingsButton, dy float32) {
	clr := b.Color
	if b.Hovered {
		clr = b.HoverColor
	}

	ui.DrawRoundedRect(clip, b.X, b.Y+dy, b.W, b.H, ui.RadiusButton, clr)

	if b.Face != nil && b.Label != "" {
		tw, th := textv2.Measure(b.Label, b.Face, 0)
		tx := float64(b.X) + (float64(b.W)-tw)/2
		ty := float64(b.Y+dy) + (float64(b.H)-th)/2

		ui.DrawText(clip, b.Label, b.Face, tx, ty, b.TextColor)
	}
}

func (s *RenderSystem) drawSlider(clip *ebiten.Image, sl *SliderEntity, dy float32, face *textv2.GoTextFace) {
	x, y, w, h := sl.X, sl.Y+dy, sl.W, sl.H
	trackH := h * 0.35
	trackY := y + (h-trackH)/2

	// Label
	ui.DrawText(clip, sl.Label, face, float64(x), float64(y)-16, ui.ColorTextSecond)

	// Value text
	valStr := fmt.Sprintf("%.0f%%", (sl.Value-sl.Min)/(sl.Max-sl.Min)*100)
	if sl.FormatValue != nil {
		valStr = sl.FormatValue(sl.Value)
	}

	vw, _ := textv2.Measure(valStr, face, 0)
	ui.DrawText(clip, valStr, face, float64(x+w)-vw, float64(y)-16, ui.ColorTextSecond)

	// Track
	ui.DrawRoundedRect(clip, x, trackY, w, trackH, trackH/2, sl.TrackColor)

	// Filled portion
	t := float32(0)
	if sl.Max > sl.Min {
		t = float32((sl.Value - sl.Min) / (sl.Max - sl.Min))
	}

	filledW := w * t
	if filledW > trackH {
		ui.DrawRoundedRect(clip, x, trackY, filledW, trackH, trackH/2, sl.KnobColor)
	}

	// Knob
	knobR := h * 0.3
	knobX := x + filledW
	knobY := y + h/2

	ui.DrawCircle(clip, knobX, knobY, knobR+1, ui.ColorBgPrimary)
	ui.DrawCircle(clip, knobX, knobY, knobR, sl.KnobColor)
}

func (s *RenderSystem) drawToggle(clip *ebiten.Image, tg *ToggleEntity, dy float32, face *textv2.GoTextFace) {
	x, y, w, h := tg.X, tg.Y+dy, tg.W, tg.H

	// Label to the left
	_, th := textv2.Measure(tg.Label, face, 0)

	lx := float64(x) - 200
	if lx < 0 {
		lx = 0
	}

	ly := float64(y) + (float64(h)-th)/2
	ui.DrawText(clip, tg.Label, face, lx, ly, ui.ColorTextSecond)

	// Track
	trackColor := tg.OffColor
	if tg.Value {
		trackColor = tg.OnColor
	}

	ui.DrawRoundedRect(clip, x, y, w, h, h/2, trackColor)

	// Knob
	knobR := h * 0.38
	knobX := x + knobR + 3

	if tg.Value {
		knobX = x + w - knobR - 3
	}

	knobY := y + h/2

	ui.DrawCircle(clip, knobX, knobY, knobR, ui.ColorTextPrimary)
}
