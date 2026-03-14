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
		mx, _ := ebiten.CursorPosition()

		if mx > s.width-int(ui.S(50)) {
			s.onDone()

			return nil
		}

		if mx < int(ui.S(44)) && s.timer != nil {
			if fn := s.timer.OnStartPause(); fn != nil {
				fn()
			}
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
	ppW := ui.S(36)
	ppX := ui.S(8)
	ppY := ui.S(8)
	ppH := h - ui.S(16)

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

	ui.DrawText(screen, timerText, timerFace, float64(w)/2-tw/2, float64(h/2)-ui.Sf(10), ui.ColorTextPrimary)

	// Expand button
	btnW := ui.S(36)
	btnX := w - ui.S(8) - btnW
	btnY := ui.S(8)
	btnH := h - ui.S(16)

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
