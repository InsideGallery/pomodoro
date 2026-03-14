package lockscreen

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/internal/ui"
)

// Screen implements ui.Screen for the lock screen overlay.
type Screen struct {
	module *Module
	width  int
	height int
}

func (s *Screen) Init(w, h int) {
	s.width = w
	s.height = h
}

func (s *Screen) Resize(w, h int) {
	s.width = w
	s.height = h
}

func (s *Screen) Update() {
	now := time.Now()

	if s.module.lock.Complete(now) {
		s.module.lock.Stop()
	}
	// ESC is intentionally ignored — the lock cannot be dismissed
}

func (s *Screen) Draw(dst *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)
	now := time.Now()

	// Opaque dark background
	ui.DrawRoundedRect(dst, 0, 0, w, h, 0, color.RGBA{R: 0x10, G: 0x10, B: 0x18, A: 0xF0})

	faceTitle := ui.Face(true, 32)
	faceTime := ui.Face(true, 48)
	faceHint := ui.Face(false, 14)

	// Title
	ui.DrawText(dst, "Long Break", faceTitle,
		float64(w/2)-ui.Sf(80), float64(h/2)-ui.Sf(100),
		ui.ColorTextPrimary)

	// Time remaining
	rem := s.module.lock.Remaining(now)
	totalSecs := int(rem.Seconds())
	mins := totalSecs / 60
	secs := totalSecs % 60
	timeText := fmt.Sprintf("%02d:%02d", mins, secs)

	ui.DrawText(dst, timeText, faceTime,
		float64(w/2)-ui.Sf(70), float64(h/2)-ui.Sf(20),
		ui.ColorAccentBreak)

	// Progress bar
	barW := ui.S(300)
	barH := ui.S(8)
	barX := (w - barW) / 2
	barY := h/2 + ui.S(40)
	progress := s.module.lock.Progress(now)

	ui.DrawRoundedRect(dst, barX, barY, barW, barH, ui.S(4),
		color.RGBA{R: 0x30, G: 0x30, B: 0x40, A: 0xFF})
	ui.DrawRoundedRect(dst, barX, barY, barW*float32(progress), barH, ui.S(4),
		ui.ColorAccentBreak)

	// Message
	ui.DrawText(dst, "Relax and rest your eyes", faceHint,
		float64(w/2)-ui.Sf(90), float64(h/2)+ui.Sf(80),
		ui.ColorTextSecond)
}
