package mini

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/pkg/platform"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const (
	SceneName = "mini"
	Width     = 220
	Height    = 60
	Title     = "Pomodoro"
)

// TimerProvider gives the mini scene access to timer status without knowing concrete types.
type TimerProvider interface {
	TimerRemaining() time.Duration
	TimerIsRunning() bool
	OnStartPause() func()
	Muted() bool
	ToggleMute()
}

type miniButton int

const (
	btnNone miniButton = iota
	btnPlay
	btnMute
	btnExpand
)

type rect struct{ X, Y, W, H float32 }

func (r rect) contains(x, y float32) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// layout returns the click rectangles of the three buttons for a window of size w by h.
func layout(w, h float32) (play, mute, expand rect) {
	bw := ui.S(36)
	gap := ui.S(8)
	y := ui.S(8)
	bh := h - ui.S(16)

	play = rect{X: gap, Y: y, W: bw, H: bh}
	expand = rect{X: w - gap - bw, Y: y, W: bw, H: bh}
	mute = rect{X: w - gap - bw - gap - bw, Y: y, W: bw, H: bh}

	return play, mute, expand
}

// hit returns the button under the point, or btnNone.
func hit(w, h float32, mx, my int) miniButton {
	play, mute, expand := layout(w, h)
	x, y := float32(mx), float32(my)

	switch {
	case play.contains(x, y):
		return btnPlay
	case mute.contains(x, y):
		return btnMute
	case expand.contains(x, y):
		return btnExpand
	}

	return btnNone
}

// Scene is the compact mini-mode overlay.
type Scene struct {
	*scene.BaseScene

	timer  TimerProvider
	onDone func() // called to switch back to timer scene

	width, height int
}

func NewScene(timer TimerProvider, onDone func()) *Scene {
	return &Scene{
		timer:  timer,
		onDone: onDone,
	}
}

func (s *Scene) Name() string { return SceneName }

func (s *Scene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
}

func (s *Scene) Load() error {
	ebiten.SetWindowSize(Width, Height)
	platform.SetAlwaysOnTop(Title, true)

	return nil
}

func (s *Scene) Unload() error {
	platform.SetAlwaysOnTop(Title, false)

	return nil
}

func (s *Scene) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.onDone()

		return nil
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()

		switch hit(float32(s.width), float32(s.height), mx, my) {
		case btnPlay:
			if s.timer != nil {
				if fn := s.timer.OnStartPause(); fn != nil {
					fn()
				}
			}
		case btnMute:
			if s.timer != nil {
				s.timer.ToggleMute()
			}
		case btnExpand:
			s.onDone()

			return nil
		case btnNone:
		}
	}

	return nil
}

func (s *Scene) Draw(screen *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)

	ui.DrawRoundedRect(screen, 0, 0, w, h, ui.S(8), ui.ColorWindowBg)
	ui.DrawRoundedRectStroke(screen, 0, 0, w, h, ui.S(8), ui.S(1), ui.ColorCardBorder)

	// Play/pause button
	play, mute, expand := layout(w, h)
	ppW, ppX, ppY, ppH := play.W, play.X, play.Y, play.H

	ui.DrawRoundedRect(screen, ppX, ppY, ppW, ppH, ui.S(6), ui.ColorBgTertiary)

	if s.timer != nil && s.timer.TimerIsRunning() {
		ui.DrawPauseIcon(screen, ppX+ppW/2, ppY+ppH/2, ui.S(18), ui.ColorTextPrimary)
	} else {
		ui.DrawPlayIcon(screen, ppX+ppW/2, ppY+ppH/2, ui.S(18), ui.ColorTextPrimary)
	}

	// Timer text
	var rem time.Duration
	if s.timer != nil {
		rem = s.timer.TimerRemaining()
	}

	if rem < 0 {
		rem = 0
	}

	totalSecs := int(rem.Seconds())
	mins := totalSecs / 60
	secs := totalSecs % 60
	timerText := fmt.Sprintf("%02d:%02d", mins, secs)

	timerFace := ui.Face(true, 18)
	tw, _ := textv2.Measure(timerText, timerFace, 0)

	centerX := float64((play.X + play.W + mute.X) / 2)

	ui.DrawText(screen, timerText, timerFace, centerX-tw/2, float64(h/2)-ui.Sf(10), ui.ColorTextPrimary)

	// Mute button
	ui.DrawRoundedRect(screen, mute.X, mute.Y, mute.W, mute.H, ui.S(6), ui.ColorBgTertiary)

	if s.timer != nil && s.timer.Muted() {
		ui.DrawMutedIcon(screen, mute.X+mute.W/2, mute.Y+mute.H/2, ui.S(16), ui.ColorTextPrimary)
	} else {
		ui.DrawSpeakerIcon(screen, mute.X+mute.W/2, mute.Y+mute.H/2, ui.S(16), ui.ColorTextPrimary)
	}

	// Expand button
	btnW, btnX, btnY, btnH := expand.W, expand.X, expand.Y, expand.H

	ui.DrawRoundedRect(screen, btnX, btnY, btnW, btnH, ui.S(6), ui.ColorBgTertiary)
	ui.DrawExpandIcon(screen, btnX+btnW/2, btnY+btnH/2, ui.S(16), ui.ColorTextPrimary)
}

func (s *Scene) Layout(outsideWidth, outsideHeight int) (int, int) {
	scale := 1.0
	if m := ebiten.Monitor(); m != nil {
		scale = m.DeviceScaleFactor()
	}

	ui.UIScale = scale

	w := int(math.Ceil(float64(outsideWidth) * scale))
	h := int(math.Ceil(float64(outsideHeight) * scale))
	s.width = w
	s.height = h

	return w, h
}
