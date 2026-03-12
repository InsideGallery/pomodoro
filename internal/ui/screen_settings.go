package ui

import (
	"fmt"
	"image"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/internal/config"
)

type SettingsScreen struct {
	Cfg *config.Config

	FocusSlider      Slider
	BreakSlider      Slider
	LongBreakSlider  Slider
	RoundsSlider     Slider
	TickVolSlider    Slider
	AlarmVolSlider   Slider
	TickToggle       Toggle
	AutoStartToggle  Toggle
	ThemeToggle      Toggle
	TransparencySldr Slider
	BtnBack          Button
	BtnReset         Button

	OnBack               func()
	OnResetDefaults      func()
	OnThemeChange        func(string)
	OnTransparencyChange func(float64)
	OnTickVolumeChange   func(float64)
	OnAlarmVolumeChange  func(float64)
	OnTickEnabledChange  func(bool)

	faceHeading *textv2.GoTextFace
	faceLabel   *textv2.GoTextFace
	faceSection *textv2.GoTextFace

	scrollY     float32
	contentH    float32 // total height of scrollable content
	initialized bool
	width       int
	height      int

	// section title Y positions (content-relative, starting from 0)
	timerTitleY  float32
	soundTitleY  float32
	appearTitleY float32
}

func (s *SettingsScreen) Init(w, h int) {
	s.width = w
	s.height = h
	s.faceHeading = Face(true, 16)
	s.faceLabel = Face(false, 12)
	s.faceSection = Face(true, 10)
	s.scrollY = 0
	s.layout()
	s.initialized = true
}

// Relayout rebuilds widgets with current config/colors but preserves scroll position.
func (s *SettingsScreen) Relayout() {
	oldScroll := s.scrollY
	s.faceHeading = Face(true, 16)
	s.faceLabel = Face(false, 12)
	s.faceSection = Face(true, 10)
	s.layout()

	s.scrollY = oldScroll
	if mx := s.maxScroll(); s.scrollY > mx {
		s.scrollY = mx
	}
}

func (s *SettingsScreen) layout() {
	w := float32(s.width)
	pad := S(24)
	sliderW := w - pad*2 - S(32)
	sliderX := pad + S(16)
	sliderH := S(20)
	rowH := S(44)
	toggleRowH := S(36)
	sectionGap := S(20)
	titleGap := S(30) // space between section title and first control

	// Back button (fixed, not scrolled)
	s.BtnBack = Button{
		X: pad, Y: S(10), W: S(32), H: S(32),
		Color: ColorBgTertiary, HoverColor: ColorBorder,
		TextColor: ColorTextPrimary,
		IconDraw:  DrawBackIcon, OnClick: s.OnBack,
	}

	// All Y values below are content-relative (0 = top of scrollable area).
	// They get shifted to screen space by adding contentTop() - scrollY.
	y := float32(0)

	// --- TIMER ---
	s.timerTitleY = y
	y += titleGap

	minFmt := func(v float64) string { return fmt.Sprintf("%d min", int(math.Round(v))) }
	intFmt := func(v float64) string { return fmt.Sprintf("%d", int(math.Round(v))) }

	s.FocusSlider = Slider{
		X: sliderX, Y: y, W: sliderW, H: sliderH,
		Min: 1, Max: 60, Value: float64(s.Cfg.FocusDuration / time.Minute),
		TrackColor: ColorSliderTrack, KnobColor: ColorAccentFocus,
		Label: "Focus Duration", Face: s.faceLabel, TextColor: ColorTextSecond,
		FormatValue: minFmt,
		OnChange: func(v float64) {
			s.Cfg.FocusDuration = time.Duration(math.Round(v)) * time.Minute
			s.save()
		},
	}
	y += rowH
	s.BreakSlider = Slider{
		X: sliderX, Y: y, W: sliderW, H: sliderH,
		Min: 1, Max: 30, Value: float64(s.Cfg.BreakDuration / time.Minute),
		TrackColor: ColorSliderTrack, KnobColor: ColorAccentBreak,
		Label: "Short Break", Face: s.faceLabel, TextColor: ColorTextSecond,
		FormatValue: minFmt,
		OnChange: func(v float64) {
			s.Cfg.BreakDuration = time.Duration(math.Round(v)) * time.Minute
			s.save()
		},
	}
	y += rowH
	s.LongBreakSlider = Slider{
		X: sliderX, Y: y, W: sliderW, H: sliderH,
		Min: 1, Max: 60, Value: float64(s.Cfg.LongBreakDuration / time.Minute),
		TrackColor: ColorSliderTrack, KnobColor: ColorGradBreakEnd,
		Label: "Long Break", Face: s.faceLabel, TextColor: ColorTextSecond,
		FormatValue: minFmt,
		OnChange: func(v float64) {
			s.Cfg.LongBreakDuration = time.Duration(math.Round(v)) * time.Minute
			s.save()
		},
	}
	y += rowH
	s.RoundsSlider = Slider{
		X: sliderX, Y: y, W: sliderW, H: sliderH,
		Min: 1, Max: 10, Value: float64(s.Cfg.RoundsBeforeLong),
		TrackColor: ColorSliderTrack, KnobColor: ColorAccentFocus,
		Label: "Rounds Before Long Break", Face: s.faceLabel, TextColor: ColorTextSecond,
		FormatValue: intFmt,
		OnChange: func(v float64) {
			s.Cfg.RoundsBeforeLong = int(math.Round(v))
			s.save()
		},
	}

	// --- SOUND ---
	y += rowH + sectionGap
	s.soundTitleY = y
	y += titleGap

	s.TickVolSlider = Slider{
		X: sliderX, Y: y, W: sliderW, H: sliderH,
		Min: 0, Max: 1, Value: s.Cfg.TickVolume,
		TrackColor: ColorSliderTrack, KnobColor: ColorAccentSuccess,
		Label: "Tick Volume", Face: s.faceLabel, TextColor: ColorTextSecond,
		OnChange: func(v float64) {
			s.Cfg.TickVolume = v
			if s.OnTickVolumeChange != nil {
				s.OnTickVolumeChange(v)
			}

			s.save()
		},
	}
	y += rowH
	s.AlarmVolSlider = Slider{
		X: sliderX, Y: y, W: sliderW, H: sliderH,
		Min: 0, Max: 1, Value: s.Cfg.AlarmVolume,
		TrackColor: ColorSliderTrack, KnobColor: ColorAccentDanger,
		Label: "Alarm Volume", Face: s.faceLabel, TextColor: ColorTextSecond,
		OnChange: func(v float64) {
			s.Cfg.AlarmVolume = v
			if s.OnAlarmVolumeChange != nil {
				s.OnAlarmVolumeChange(v)
			}

			s.save()
		},
	}

	// --- APPEARANCE ---
	y += rowH + sectionGap
	s.appearTitleY = y
	y += titleGap

	toggleW := S(44)
	toggleH := S(24)
	toggleX := w - pad - toggleW - S(16)

	s.TickToggle = Toggle{
		X: toggleX, Y: y, W: toggleW, H: toggleH,
		Value:   s.Cfg.TickEnabled,
		OnColor: ColorAccentSuccess, OffColor: ColorToggleOff,
		KnobColor: ColorTextPrimary,
		Label:     "Tick Sound", Face: s.faceLabel, TextColor: ColorTextSecond,
		OnChange: func(v bool) {
			s.Cfg.TickEnabled = v
			if s.OnTickEnabledChange != nil {
				s.OnTickEnabledChange(v)
			}

			s.save()
		},
	}
	y += toggleRowH
	s.AutoStartToggle = Toggle{
		X: toggleX, Y: y, W: toggleW, H: toggleH,
		Value:   s.Cfg.AutoStart,
		OnColor: ColorAccentSuccess, OffColor: ColorToggleOff,
		KnobColor: ColorTextPrimary,
		Label:     "Auto-Start Next", Face: s.faceLabel, TextColor: ColorTextSecond,
		OnChange: func(v bool) {
			s.Cfg.AutoStart = v
			s.save()
		},
	}
	y += toggleRowH
	s.ThemeToggle = Toggle{
		X: toggleX, Y: y, W: toggleW, H: toggleH,
		Value:   s.Cfg.Theme == "light",
		OnColor: ColorAccentFocus, OffColor: ColorToggleOff,
		KnobColor: ColorTextPrimary,
		Label:     "Light Theme", Face: s.faceLabel, TextColor: ColorTextSecond,
		OnChange: func(v bool) {
			if v {
				s.Cfg.Theme = "light"
			} else {
				s.Cfg.Theme = "dark"
			}

			if s.OnThemeChange != nil {
				s.OnThemeChange(s.Cfg.Theme)
			}

			s.save()
		},
	}
	y += rowH
	pctFmt := func(v float64) string { return fmt.Sprintf("%.0f%%", v*100) }
	s.TransparencySldr = Slider{
		X: sliderX, Y: y, W: sliderW, H: sliderH,
		Min: 0.10, Max: 0.90, Value: s.Cfg.Transparency,
		TrackColor: ColorSliderTrack, KnobColor: ColorAccentFocus,
		Label: "Transparency", Face: s.faceLabel, TextColor: ColorTextSecond,
		FormatValue: pctFmt,
		OnChange: func(v float64) {
			s.Cfg.Transparency = v
			if s.OnTransparencyChange != nil {
				s.OnTransparencyChange(v)
			}

			s.save()
		},
	}

	// Reset button
	y += rowH + sectionGap
	resetW := S(140)
	resetH := S(34)
	resetX := (w - resetW) / 2
	s.BtnReset = Button{
		X: resetX, Y: y, W: resetW, H: resetH,
		Label: "Reset Defaults", Face: s.faceLabel,
		Color: ColorBgTertiary, HoverColor: colorBrighten(ColorAccentDanger, 0.4),
		TextColor: ColorAccentDanger,
		OnClick:   s.OnResetDefaults,
	}
	y += resetH + S(16)
	s.contentH = y
}

// contentTop: screen Y where the scrollable area starts.
func (s *SettingsScreen) contentTop() float32 { return S(58) }

// visibleH: height of the visible scroll viewport.
func (s *SettingsScreen) visibleH() float32 {
	return float32(s.height) - s.contentTop() - S(24)
}

func (s *SettingsScreen) maxScroll() float32 {
	m := s.contentH - s.visibleH()
	if m < 0 {
		return 0
	}

	return m
}

// shiftToScreen shifts all content-relative Y positions to screen space.
func (s *SettingsScreen) shiftToScreen() {
	dy := s.contentTop() - s.scrollY
	s.FocusSlider.Y += dy
	s.BreakSlider.Y += dy
	s.LongBreakSlider.Y += dy
	s.RoundsSlider.Y += dy
	s.TickVolSlider.Y += dy
	s.AlarmVolSlider.Y += dy
	s.TickToggle.Y += dy
	s.AutoStartToggle.Y += dy
	s.ThemeToggle.Y += dy
	s.TransparencySldr.Y += dy
	s.BtnReset.Y += dy
	s.timerTitleY += dy
	s.soundTitleY += dy
	s.appearTitleY += dy
}

// shiftToContent restores content-relative Y positions.
func (s *SettingsScreen) shiftToContent() {
	dy := -(s.contentTop() - s.scrollY)
	s.FocusSlider.Y += dy
	s.BreakSlider.Y += dy
	s.LongBreakSlider.Y += dy
	s.RoundsSlider.Y += dy
	s.TickVolSlider.Y += dy
	s.AlarmVolSlider.Y += dy
	s.TickToggle.Y += dy
	s.AutoStartToggle.Y += dy
	s.ThemeToggle.Y += dy
	s.TransparencySldr.Y += dy
	s.BtnReset.Y += dy
	s.timerTitleY += dy
	s.soundTitleY += dy
	s.appearTitleY += dy
}

func (s *SettingsScreen) Update() {
	if !s.initialized {
		return
	}

	// Scroll
	_, wy := ebiten.Wheel()
	if wy != 0 {
		s.scrollY -= float32(wy) * S(30)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		s.scrollY += S(30)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		s.scrollY -= S(30)
	}

	if s.scrollY < 0 {
		s.scrollY = 0
	}

	if mx := s.maxScroll(); s.scrollY > mx {
		s.scrollY = mx
	}

	// Back button is fixed (not scrollable)
	s.BtnBack.Update()

	// Shift widgets to screen space for hit testing (same coords as Draw)
	s.shiftToScreen()

	s.FocusSlider.Update()
	s.BreakSlider.Update()
	s.LongBreakSlider.Update()
	s.RoundsSlider.Update()
	s.TickVolSlider.Update()
	s.AlarmVolSlider.Update()
	s.TickToggle.Update()
	s.AutoStartToggle.Update()
	s.ThemeToggle.Update()
	s.TransparencySldr.Update()
	s.BtnReset.Update()

	s.shiftToContent()
}

func (s *SettingsScreen) Draw(screen *ebiten.Image) {
	if !s.initialized {
		return
	}

	w := float32(s.width)
	h := float32(s.height)
	pad := S(24)
	cardW := w - pad*2

	// Main card background
	cardY := S(48)
	cardH := h - cardY - pad
	DrawRoundedRect(screen, pad, cardY, cardW, cardH, S(RadiusCard), ColorCardBg)
	DrawRoundedRectStroke(screen, pad, cardY, cardW, cardH, S(RadiusCard), S(1), ColorCardBorder)

	// Header (fixed, not clipped)
	s.BtnBack.Draw(screen)

	if s.faceHeading != nil {
		DrawTextCentered(screen, "Settings", s.faceHeading, float64(w/2), Sf(16), ColorTextPrimary)
	}

	// --- Clipped scrollable area ---
	// SubImage clips drawing to the card content area.
	clipRect := image.Rect(
		int(pad+S(1)), int(s.contentTop()),
		int(w-pad-S(1)), int(h-pad-S(1)),
	)
	clip := screen.SubImage(clipRect).(*ebiten.Image)

	// Shift widgets to screen space (same transform as Update)
	s.shiftToScreen()

	// Section titles
	titleX := float64(pad) + Sf(12)
	if s.faceSection != nil {
		DrawText(clip, "TIMER", s.faceSection, titleX, float64(s.timerTitleY), ColorAccentFocus)
		DrawText(clip, "SOUND", s.faceSection, titleX, float64(s.soundTitleY), ColorAccentSuccess)
		DrawText(clip, "APPEARANCE", s.faceSection, titleX, float64(s.appearTitleY), ColorAccentFocus)
	}

	// Section dividers
	DrawRoundedRect(clip, pad+S(8), s.soundTitleY-S(8), cardW-S(16), S(1), 0, ColorBorder)
	DrawRoundedRect(clip, pad+S(8), s.appearTitleY-S(8), cardW-S(16), S(1), 0, ColorBorder)

	// Widgets
	s.FocusSlider.Draw(clip)
	s.BreakSlider.Draw(clip)
	s.LongBreakSlider.Draw(clip)
	s.RoundsSlider.Draw(clip)
	s.TickVolSlider.Draw(clip)
	s.AlarmVolSlider.Draw(clip)
	s.TickToggle.Draw(clip)
	s.AutoStartToggle.Draw(clip)
	s.ThemeToggle.Draw(clip)
	s.TransparencySldr.Draw(clip)
	s.BtnReset.Draw(clip)

	s.shiftToContent()

	// Scroll indicator
	if s.maxScroll() > 0 {
		visH := s.visibleH()

		barH := visH * visH / s.contentH
		if barH < S(20) {
			barH = S(20)
		}

		barY := s.contentTop() + (visH-barH)*(s.scrollY/s.maxScroll())
		barX := w - pad - S(4)
		DrawRoundedRect(screen, barX, barY, S(3), barH, S(2), ColorBorder)
	}
}

func (s *SettingsScreen) Resize(w, h int) {
	s.width = w
	s.height = h
	s.faceHeading = Face(true, 16)
	s.faceLabel = Face(false, 12)
	s.faceSection = Face(true, 10)
	s.layout()
}

func (s *SettingsScreen) save() {
	_ = config.Save(*s.Cfg)
}
