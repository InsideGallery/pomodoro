package systems

import (
	"context"
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/InsideGallery/core/memory/registry"
	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/internal/timer"
	"github.com/InsideGallery/pomodoro/pkg/ecs"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

// RenderSystem draws all timer entities from the Registry.
type RenderSystem struct {
	Reg *registry.Registry[string, uint64, any]
	Tmr *timer.Timer
}

func (s *RenderSystem) Update(_ context.Context) error {
	s.updateStartButton()

	return nil
}

func (s *RenderSystem) Draw(_ context.Context, screen *ebiten.Image) {
	now := time.Now()
	w := float32(screen.Bounds().Dx())
	pad := ui.S(24)
	cardW := w - pad*2
	cardY := ui.S(48)
	cardH := float32(screen.Bounds().Dy()) - cardY - pad

	// Card background
	ui.DrawRoundedRect(screen, pad, cardY, cardW, cardH, ui.S(ui.RadiusCard), ui.ColorCardBg)
	ui.DrawRoundedRectStroke(screen, pad, cardY, cardW, cardH, ui.S(ui.RadiusCard), ui.S(1), ui.ColorCardBorder)

	// Draw buttons
	for btn := range s.Reg.Iterator("button") {
		b, ok := btn.(*ecs.ButtonEntity)
		if !ok {
			continue
		}

		clr := b.Color
		if b.Hovered {
			clr = b.HoverColor
		}

		bx, by, bw, bh := float32(b.Pos.X), float32(b.Pos.Y), float32(b.Size.W), float32(b.Size.H)
		ui.DrawRoundedRect(screen, bx, by, bw, bh, ui.RadiusButton, clr)

		if b.IconDraw != nil {
			if iconFn, ok := b.IconDraw.(func(*ebiten.Image, float32, float32, float32, color.Color)); ok {
				iconFn(screen, bx+bw/2, by+bh/2, bh*0.5, b.TextColor)
			}
		} else if b.Face != nil && b.Label != "" {
			tw, th := textv2.Measure(b.Label, b.Face, 0)
			tx := float64(bx) + (float64(bw)-tw)/2
			ty := float64(by) + (float64(bh)-th)/2

			ui.DrawText(screen, b.Label, b.Face, tx, ty, b.TextColor)
		}
	}

	// Draw ring
	for ring := range s.Reg.Iterator("ring") {
		rp, ok := ring.(*RingEntityData)
		if !ok {
			continue
		}

		innerR := float32(rp.OuterR) - ui.S(14)
		ui.DrawArc(screen, float32(rp.CX), float32(rp.CY), float32(rp.OuterR), innerR, 0, 2*math.Pi, rp.TrackColor)

		progress := s.Tmr.Progress(now)
		if progress > 0 {
			startAngle := -math.Pi / 2
			endAngle := startAngle + progress*2*math.Pi
			startClr, endClr := s.gradientForState(s.Tmr.State())

			ui.DrawGradientArc(screen,
				float32(rp.CX), float32(rp.CY), float32(rp.OuterR), innerR,
				startAngle, endAngle, startClr, endClr)

			capMidR := float32(rp.OuterR) - ui.S(14)/2
			capX := float32(rp.CX) + capMidR*float32(math.Cos(endAngle))
			capY := float32(rp.CY) + capMidR*float32(math.Sin(endAngle))

			ui.DrawCircle(screen, capX, capY, ui.S(14)/2, endClr)
		}
	}

	// Draw mode label
	for ml := range s.Reg.Iterator("mode_label") {
		m, ok := ml.(*ecs.ModeLabelEntity)
		if !ok {
			continue
		}

		if m.Face != nil {
			ui.DrawTextCentered(screen, m.TextFunc(), m.Face, m.Pos.X, m.Pos.Y, m.ColorFunc())
		}
	}

	// Draw timer text
	for tt := range s.Reg.Iterator("timer_text") {
		t, ok := tt.(*ecs.TimerTextEntity)
		if !ok {
			continue
		}

		rem := t.Remaining()
		if rem < 0 {
			rem = 0
		}

		totalSecs := int(rem.Seconds())
		mins := totalSecs / 60
		secs := totalSecs % 60
		text := fmt.Sprintf("%02d:%02d", mins, secs)

		if t.Face != nil {
			tw, th := textv2.Measure(text, t.Face, 0)
			ui.DrawText(screen, text, t.Face, t.Pos.X-tw/2, t.Pos.Y-th/2, t.Color)
		}
	}

	// Draw hint
	for h := range s.Reg.Iterator("hint") {
		hint, ok := h.(*ecs.HintEntity)
		if !ok {
			continue
		}

		text := hint.TextFunc()
		if text != "" && hint.Face != nil {
			ui.DrawTextCentered(screen, text, hint.Face, hint.Pos.X, hint.Pos.Y, hint.Color)
		}
	}

	// Draw round dots
	for d := range s.Reg.Iterator("round_dots") {
		dots, ok := d.(*RoundDotsEntityData)
		if !ok {
			continue
		}

		s.drawDots(screen, dots)
	}
}

func (s *RenderSystem) drawDots(screen *ebiten.Image, dots *RoundDotsEntityData) {
	total := s.Tmr.Config().RoundsBeforeLong
	completed := s.Tmr.Round()

	if total <= 0 {
		return
	}

	r := ui.S(4)
	gap := ui.S(12)
	totalW := float32(total)*r*2 + float32(total-1)*gap
	startX := float32(dots.CX) - totalW/2 + r
	accentClr := s.accentForState(s.Tmr.State())

	for i := range total {
		x := startX + float32(i)*(r*2+gap)

		if i < completed {
			ui.DrawCircle(screen, x, float32(dots.CY), r, accentClr)
		} else {
			ui.DrawCircle(screen, x, float32(dots.CY), r, ui.ColorBorder)
		}
	}
}

func (s *RenderSystem) updateStartButton() {
	for btn := range s.Reg.Iterator("button") {
		b, ok := btn.(*ecs.ButtonEntity)
		if !ok || b.Label == "" {
			continue
		}

		// Only update the start/pause/resume button
		switch b.Label {
		case "Focus", "Break", "Resume", "Pause":
		default:
			if b.Label != "Start" {
				continue
			}
		}

		state := s.Tmr.State()

		switch state {
		case timer.StateIdle:
			pending := s.Tmr.PendingNext()

			switch pending {
			case timer.StateBreak, timer.StateLongBreak:
				b.Label = "Break"
				b.Color = ui.ColorAccentBreak
				b.HoverColor = ui.ColorAccentBreak
				b.TextColor = ui.ColorBgPrimary
			default:
				b.Label = "Focus"
				b.Color = ui.ColorAccentSuccess
				b.HoverColor = ui.ColorAccentSuccess
				b.TextColor = ui.ColorBgPrimary
			}
		case timer.StatePaused:
			b.Label = "Resume"
			b.Color = ui.ColorAccentSuccess
			b.HoverColor = ui.ColorAccentSuccess
			b.TextColor = ui.ColorBgPrimary
		default:
			b.Label = "Pause"
			b.Color = ui.ColorAccentFocus
			b.HoverColor = ui.ColorAccentFocus
			b.TextColor = ui.ColorTextPrimary
		}
	}
}

func (s *RenderSystem) accentForState(st timer.State) color.Color {
	switch st {
	case timer.StateFocus:
		return ui.ColorAccentFocus
	case timer.StateBreak:
		return ui.ColorAccentBreak
	case timer.StateLongBreak:
		return ui.ColorGradBreakEnd
	default:
		return ui.ColorTextSecond
	}
}

func (s *RenderSystem) gradientForState(st timer.State) (color.Color, color.Color) {
	switch st {
	case timer.StateFocus:
		return ui.ColorAccentFocus, ui.ColorGradFocusEnd
	case timer.StateBreak, timer.StateLongBreak:
		return ui.ColorAccentBreak, ui.ColorGradBreakEnd
	default:
		return ui.ColorAccentFocus, ui.ColorGradFocusEnd
	}
}
