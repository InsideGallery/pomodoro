package minigame

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/InsideGallery/pomodoro/internal/ui"
)

// Screen implements ui.Screen for the Button Hunt mini-game.
type Screen struct {
	module      *Module
	width       int
	height      int
	initialized bool
}

func (s *Screen) Init(w, h int) {
	s.width = w
	s.height = h
	s.initialized = true

	if s.module.active && !s.module.gameOver {
		s.module.game.Start(w, h, s.module.BestScore(), s.module.BreakDur(), time.Now())
	}
}

func (s *Screen) Resize(w, h int) {
	s.width = w
	s.height = h
}

func (s *Screen) Update() {
	if s.module.gameOver {
		// Any click or ESC dismisses the game-over screen
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
			inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.module.Dismiss()
		}

		return
	}

	now := time.Now()

	if s.module.game.IsOver(now) {
		s.module.finish()

		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.module.Dismiss()

		return
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		s.module.game.Click(float64(mx), float64(my))
	}
}

func (s *Screen) Draw(dst *ebiten.Image) {
	// Fully transparent background — targets float directly over the desktop
	if s.module.gameOver {
		s.drawGameOver(dst)

		return
	}

	s.drawTargets(dst)
	s.drawHUD(dst)
}

func (s *Screen) drawTargets(dst *ebiten.Image) {
	palette := []color.RGBA{
		{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF}, // purple
		{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF}, // teal
		{R: 0xFD, G: 0x79, B: 0x72, A: 0xFF}, // coral
		{R: 0xFD, G: 0xCB, B: 0x6E, A: 0xFF}, // yellow
		{R: 0x55, G: 0xEF, B: 0xC4, A: 0xFF}, // mint
		{R: 0xA2, G: 0x9B, B: 0xFE, A: 0xFF}, // lavender
		{R: 0xFF, G: 0x77, B: 0x75, A: 0xFF}, // salmon
		{R: 0x74, G: 0xB9, B: 0xFF, A: 0xFF}, // sky blue
		{R: 0xFF, G: 0x92, B: 0x50, A: 0xFF}, // orange
		{R: 0x00, G: 0xD2, B: 0xD3, A: 0xFF}, // cyan
	}

	for i, t := range s.module.game.Targets {
		if !t.Alive {
			continue
		}

		clr := palette[i%len(palette)]

		// Border
		border := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xBB}
		vector.FillCircle(dst, float32(t.X), float32(t.Y), float32(t.Radius+2), border, true)
		// Fill
		vector.FillCircle(dst, float32(t.X), float32(t.Y), float32(t.Radius), clr, true)
	}
}

func (s *Screen) drawHUD(dst *ebiten.Image) {
	now := time.Now()

	rem := s.module.game.Remaining(now)
	totalSecs := int(rem.Seconds())
	mins := totalSecs / 60
	secs := totalSecs % 60

	// HUD card in top-right corner
	cardW := ui.S(160)
	cardH := ui.S(80)
	cardX := float32(s.width) - cardW - ui.S(16)
	cardY := ui.S(16)

	ui.DrawRoundedRect(dst, cardX, cardY, cardW, cardH, ui.S(8),
		color.RGBA{R: 0x18, G: 0x18, B: 0x22, A: 0xDD})

	face := ui.Face(true, 14)
	faceSmall := ui.Face(false, 11)

	scoreText := fmt.Sprintf("Score: %d", s.module.game.Score)
	ui.DrawText(dst, scoreText, face, float64(cardX+ui.S(12)), float64(cardY+ui.S(16)), ui.ColorTextPrimary)

	bestText := fmt.Sprintf("Best: %d", s.module.game.BestScore)
	ui.DrawText(dst, bestText, faceSmall, float64(cardX+ui.S(12)), float64(cardY+ui.S(36)), ui.ColorTextSecond)

	timeText := fmt.Sprintf("%02d:%02d", mins, secs)
	ui.DrawText(dst, timeText, face, float64(cardX+ui.S(12)), float64(cardY+ui.S(56)), ui.ColorTextPrimary)

	escText := "ESC to close"
	ui.DrawText(dst, escText, faceSmall, float64(cardX+cardW-ui.S(90)), float64(cardY+ui.S(56)), ui.ColorTextSecond)
}

func (s *Screen) drawGameOver(dst *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)

	// Dark overlay
	ui.DrawRoundedRect(dst, 0, 0, w, h, 0, color.RGBA{R: 0x10, G: 0x10, B: 0x18, A: 0xDD})

	// Result card
	cardW := ui.S(300)
	cardH := ui.S(200)
	cardX := (w - cardW) / 2
	cardY := (h - cardH) / 2

	ui.DrawRoundedRect(dst, cardX, cardY, cardW, cardH, ui.S(12),
		color.RGBA{R: 0x20, G: 0x20, B: 0x30, A: 0xFF})

	faceTitle := ui.Face(true, 24)
	faceScore := ui.Face(true, 18)
	faceHint := ui.Face(false, 12)

	if s.module.game.BeatRecord() {
		ui.DrawText(dst, "New Record!", faceTitle,
			float64(cardX+ui.S(60)), float64(cardY+ui.S(40)),
			color.RGBA{R: 0xFD, G: 0xCB, B: 0x6E, A: 0xFF})
	} else {
		ui.DrawText(dst, "Time's Up!", faceTitle,
			float64(cardX+ui.S(70)), float64(cardY+ui.S(40)),
			ui.ColorTextPrimary)
	}

	scoreText := fmt.Sprintf("Score: %d", s.module.game.Score)
	ui.DrawText(dst, scoreText, faceScore,
		float64(cardX+ui.S(100)), float64(cardY+ui.S(90)),
		ui.ColorTextPrimary)

	bestText := fmt.Sprintf("Best: %d", max(s.module.game.Score, s.module.game.BestScore))
	ui.DrawText(dst, bestText, faceScore,
		float64(cardX+ui.S(105)), float64(cardY+ui.S(120)),
		ui.ColorTextSecond)

	ui.DrawText(dst, "Click or ESC to continue", faceHint,
		float64(cardX+ui.S(65)), float64(cardY+ui.S(170)),
		ui.ColorTextSecond)
}
