package ui

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/internal/timer"
)

type TimerScreen struct {
	Timer       *timer.Timer
	BtnStart    Button
	BtnReset    Button
	BtnSkip     Button
	BtnSettings Button
	BtnClose    Button

	OnStart      func()
	OnReset      func()
	OnSkip       func()
	OnSettings   func()
	OnClose      func()
	OnMini       func()
	OnSetRound   func(int)
	OnAdjustTime func(time.Duration)

	BtnMini Button

	faceTimer *textv2.GoTextFace
	faceMode  *textv2.GoTextFace
	faceSmall *textv2.GoTextFace
	faceBtn   *textv2.GoTextFace

	// Ring geometry (computed in Draw, used by Update for drag detection)
	ringCX, ringCY float32
	outerR         float32
	ringW          float32
	ringDragging   bool

	// Dot geometry
	dotCenters []float32 // X positions of each dot
	dotY       float32
	dotR       float32

	initialized bool
	width       int
	height      int
}

func (s *TimerScreen) Init(w, h int) {
	s.width = w
	s.height = h
	s.faceTimer = Face(true, 56)
	s.faceMode = Face(true, 13)
	s.faceSmall = Face(false, 12)
	s.faceBtn = Face(true, 13)
	s.layoutButtons()
	s.initialized = true
}

func (s *TimerScreen) layoutButtons() {
	w := float32(s.width)
	h := float32(s.height)
	pad := S(24)
	iconS := S(32)

	// Close button — top right
	s.BtnClose = Button{
		X: w - pad - iconS, Y: S(10), W: iconS, H: iconS,
		Color: color.RGBA{}, HoverColor: color.RGBA{R: 0xFF, G: 0x6B, B: 0x6B, A: 0x30},
		TextColor: ColorTextSecond,
		IconDraw:  DrawCloseIcon,
		OnClick:   s.OnClose,
	}

	// Mini button (minimize to small view)
	s.BtnMini = Button{
		X: w - pad - iconS*2 - S(8), Y: S(10), W: iconS, H: iconS,
		Color: color.RGBA{}, HoverColor: ColorBgTertiary,
		TextColor: ColorTextSecond,
		IconDraw:  DrawMinimizeIcon,
		OnClick:   s.OnMini,
	}

	// Settings button
	s.BtnSettings = Button{
		X: w - pad - iconS*3 - S(16), Y: S(10), W: iconS, H: iconS,
		Color: color.RGBA{}, HoverColor: ColorBgTertiary,
		TextColor: ColorTextSecond,
		IconDraw:  DrawSettingsIcon,
		OnClick:   s.OnSettings,
	}

	// Control buttons — centered with proper bottom margin
	btnW := S(96)
	btnH := S(40)
	gap := S(10)
	totalW := btnW*3 + gap*2
	startX := (w - totalW) / 2
	btnY := h - pad - btnH - S(16) // extra 16 margin from card bottom

	s.BtnStart = Button{
		X: startX, Y: btnY, W: btnW, H: btnH,
		Label: "Start", Face: s.faceBtn,
		Color: ColorAccentSuccess, HoverColor: colorBrighten(ColorAccentSuccess, 1.2),
		TextColor: ColorBgPrimary,
		OnClick:   s.OnStart,
	}
	s.BtnSkip = Button{
		X: startX + btnW + gap, Y: btnY, W: btnW, H: btnH,
		Label: "Skip", Face: s.faceBtn,
		Color: ColorBgTertiary, HoverColor: ColorBorder,
		TextColor: ColorTextPrimary,
		OnClick:   s.OnSkip,
	}
	s.BtnReset = Button{
		X: startX + (btnW+gap)*2, Y: btnY, W: btnW, H: btnH,
		Label: "Reset", Face: s.faceBtn,
		Color: ColorBgTertiary, HoverColor: colorBrighten(ColorAccentDanger, 0.4),
		TextColor: ColorAccentDanger,
		OnClick:   s.OnReset,
	}
}

func (s *TimerScreen) Update() {
	if !s.initialized {
		return
	}

	s.updateStartButton()
	s.updateRingDrag()
	s.updateDotClick()
	s.BtnClose.Update()
	s.BtnMini.Update()
	s.BtnSettings.Update()
	s.BtnStart.Update()
	s.BtnSkip.Update()
	s.BtnReset.Update()
}

func (s *TimerScreen) updateRingDrag() {
	if s.outerR == 0 {
		return
	}

	mx, my := ebiten.CursorPosition()
	fmx := float32(mx)
	fmy := float32(my)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		dx := fmx - s.ringCX
		dy := fmy - s.ringCY
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		innerR := s.outerR - s.ringW

		// Click must be near the ring band
		if dist >= innerR-S(10) && dist <= s.outerR+S(10) {
			state := s.Timer.State()
			if state.IsRunning() || state == timer.StatePaused {
				s.ringDragging = true
			}
		}
	}

	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.ringDragging = false
	}

	if s.ringDragging {
		dx := float64(fmx - s.ringCX)
		dy := float64(fmy - s.ringCY)
		angle := math.Atan2(dy, dx)

		// Convert from math angle to progress (top = 0, clockwise)
		progress := (angle + math.Pi/2) / (2 * math.Pi)
		if progress < 0 {
			progress++
		}

		total := s.Timer.TotalDuration()
		rem := time.Duration(float64(total) * (1 - progress))

		if rem < time.Second {
			rem = time.Second
		}

		if s.OnAdjustTime != nil {
			s.OnAdjustTime(rem)
		}
	}
}

func (s *TimerScreen) updateDotClick() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(s.dotCenters) == 0 {
		return
	}

	// Only allow changing round when idle
	if s.Timer.State() != timer.StateIdle {
		return
	}

	mx, my := ebiten.CursorPosition()
	fmx := float32(mx)
	fmy := float32(my)

	hitR := s.dotR + S(8) // generous hit area
	if fmy < s.dotY-hitR || fmy > s.dotY+hitR {
		return
	}

	for i, cx := range s.dotCenters {
		if fmx >= cx-hitR && fmx <= cx+hitR {
			if s.OnSetRound != nil {
				s.OnSetRound(i)
			}

			return
		}
	}
}

func (s *TimerScreen) updateStartButton() {
	state := s.Timer.State()

	switch state {
	case timer.StateIdle:
		pending := s.Timer.PendingNext()

		switch pending {
		case timer.StateBreak, timer.StateLongBreak:
			s.BtnStart.Label = "Break"
			s.BtnStart.Color = ColorAccentBreak
			s.BtnStart.HoverColor = colorBrighten(ColorAccentBreak, 1.2)
			s.BtnStart.TextColor = ColorBgPrimary
		default:
			s.BtnStart.Label = "Focus"
			s.BtnStart.Color = ColorAccentSuccess
			s.BtnStart.HoverColor = colorBrighten(ColorAccentSuccess, 1.2)
			s.BtnStart.TextColor = ColorBgPrimary
		}
	case timer.StatePaused:
		s.BtnStart.Label = "Resume"
		s.BtnStart.Color = ColorAccentSuccess
		s.BtnStart.HoverColor = colorBrighten(ColorAccentSuccess, 1.2)
		s.BtnStart.TextColor = ColorBgPrimary
	default:
		s.BtnStart.Label = "Pause"
		s.BtnStart.Color = colorWithAlpha(ColorAccentFocus, 0.8)
		s.BtnStart.HoverColor = ColorAccentFocus
		s.BtnStart.TextColor = ColorTextPrimary
	}
}

func (s *TimerScreen) Draw(screen *ebiten.Image) {
	if !s.initialized {
		return
	}

	now := time.Now()
	state := s.Timer.State()
	w := float32(s.width)
	h := float32(s.height)
	pad := S(24)
	cardW := w - pad*2

	// --- Main card ---
	cardY := S(48)
	cardH := h - cardY - pad
	DrawRoundedRect(screen, pad, cardY, cardW, cardH, S(RadiusCard), ColorCardBg)
	DrawRoundedRectStroke(screen, pad, cardY, cardW, cardH, S(RadiusCard), S(1), ColorCardBorder)

	// Top bar icons
	s.BtnClose.Draw(screen)
	s.BtnMini.Draw(screen)
	s.BtnSettings.Draw(screen)

	// --- Progress ring ---
	s.ringCX = w / 2
	s.ringCY = cardY + cardH*0.38
	maxR := math.Min(float64(cardW)*0.28, float64(cardH)*0.26)
	s.outerR = float32(maxR)
	s.ringW = S(14)
	innerR := s.outerR - s.ringW

	// Ring track
	DrawArc(screen, s.ringCX, s.ringCY, s.outerR, innerR, 0, 2*math.Pi, ColorBgTertiary)

	// Ring progress
	progress := s.Timer.Progress(now)
	accentClr := s.accentForState(state)

	if progress > 0 {
		startAngle := -math.Pi / 2
		endAngle := startAngle + progress*2*math.Pi
		startClr, endClr := s.gradientForState(state)
		DrawGradientArc(screen, s.ringCX, s.ringCY, s.outerR, innerR, startAngle, endAngle, startClr, endClr)

		capMidR := s.outerR - s.ringW/2
		capX := s.ringCX + capMidR*float32(math.Cos(endAngle))
		capY := s.ringCY + capMidR*float32(math.Sin(endAngle))
		DrawCircle(screen, capX, capY, s.ringW/2, endClr)
	}

	// --- Mode label inside ring ---
	displayState := state
	if state == timer.StateIdle {
		displayState = s.Timer.PendingNext()
	}

	modeText := displayState.String()
	if state == timer.StatePaused {
		modeText = "Paused"
	}

	if s.faceMode != nil {
		DrawTextCentered(screen, modeText, s.faceMode,
			float64(s.ringCX), float64(s.ringCY)-float64(s.outerR)*0.38, accentClr)
	}

	// --- Timer digits ---
	rem := s.Timer.Remaining(now)
	if rem < 0 {
		rem = 0
	}

	totalSecs := int(rem.Seconds())
	mins := totalSecs / 60
	secs := totalSecs % 60
	timerText := fmt.Sprintf("%02d:%02d", mins, secs)

	if s.faceTimer != nil {
		tw, th := textv2.Measure(timerText, s.faceTimer, 0)
		tx := float64(s.ringCX) - tw/2
		ty := float64(s.ringCY) - th/2 + float64(s.outerR)*0.08
		DrawText(screen, timerText, s.faceTimer, tx, ty, ColorTextPrimary)
	}

	// --- Hint below ring ---
	if state == timer.StateIdle {
		pending := s.Timer.PendingNext()

		var hintText string

		switch pending {
		case timer.StateBreak:
			hintText = "Time for a break"
		case timer.StateLongBreak:
			hintText = "Time for a long break"
		default:
			hintText = "Ready to focus"
		}

		if s.faceSmall != nil {
			DrawTextCentered(screen, hintText, s.faceSmall,
				float64(s.ringCX), float64(s.ringCY)+float64(s.outerR)+Sf(18), ColorTextSecond)
		}
	}

	// --- Round dots ---
	cfg := s.Timer.Config()
	s.dotY = s.ringCY + s.outerR + S(44)
	s.dotR = S(4)
	s.drawRoundDots(screen, s.ringCX, s.dotY, cfg.RoundsBeforeLong, s.Timer.Round(), accentClr)

	// --- Buttons ---
	s.BtnStart.Draw(screen)
	s.BtnSkip.Draw(screen)
	s.BtnReset.Draw(screen)
}

func (s *TimerScreen) drawRoundDots(screen *ebiten.Image, cx, cy float32, total, completed int, accentClr color.Color) {
	if total <= 0 {
		return
	}

	gap := S(12)
	totalW := float32(total)*s.dotR*2 + float32(total-1)*gap
	startX := cx - totalW/2 + s.dotR

	s.dotCenters = make([]float32, total)

	for i := 0; i < total; i++ {
		x := startX + float32(i)*(s.dotR*2+gap)
		s.dotCenters[i] = x

		if i < completed {
			DrawCircle(screen, x, cy, s.dotR, accentClr)
		} else {
			DrawCircle(screen, x, cy, s.dotR, ColorBorder)
		}
	}
}

func (s *TimerScreen) Resize(w, h int) {
	s.width = w
	s.height = h
	s.faceTimer = Face(true, 56)
	s.faceMode = Face(true, 13)
	s.faceSmall = Face(false, 12)
	s.faceBtn = Face(true, 13)
	s.layoutButtons()
}

func (s *TimerScreen) accentForState(st timer.State) color.Color {
	switch st {
	case timer.StateFocus:
		return ColorAccentFocus
	case timer.StateBreak:
		return ColorAccentBreak
	case timer.StateLongBreak:
		return ColorGradBreakEnd
	case timer.StatePaused:
		return ColorTextSecond
	default:
		return ColorTextSecond
	}
}

func (s *TimerScreen) gradientForState(st timer.State) (color.Color, color.Color) {
	switch st {
	case timer.StateFocus:
		return ColorAccentFocus, ColorGradFocusEnd
	case timer.StateBreak:
		return ColorAccentBreak, ColorGradBreakEnd
	case timer.StateLongBreak:
		return ColorAccentBreak, ColorGradBreakEnd
	default:
		return ColorAccentFocus, ColorGradFocusEnd
	}
}
