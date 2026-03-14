//go:build plugin

package main

import (
	"context"
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/platform"
	"github.com/InsideGallery/pomodoro/pkg/ui"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

const (
	SceneName      = "lockscreen"
	windowTitle    = "Pomodoro"
	escTapsToExit  = 3
	escResetWindow = 2 * time.Second // ESC taps must happen within this window
)

// Scene is the long-break lock screen scene.
type Scene struct {
	*scene.BaseScene

	lock     Lock
	breakDur time.Duration
	locked   bool // if true, require ESC×3 to exit; if false, auto-exit on complete
	onDone   func()

	escCount int
	lastEsc  time.Time

	width, height int
}

func NewScene(bus *event.Bus, switchToSelf func(), onDone func()) *Scene {
	cfg := config.Load()

	s := &Scene{
		breakDur: cfg.LongBreakDuration(),
		locked:   cfg.LockBreakScreen,
		onDone:   onDone,
	}

	// Self-activate on long break start if enabled
	bus.Subscribe(event.LongBreakStarted, func(_ event.Event) {
		if s.locked {
			switchToSelf()
		}
	})

	bus.Subscribe(event.ConfigChanged, func(e event.Event) {
		if c, ok := e.Data.(config.Config); ok {
			s.breakDur = c.LongBreakDuration()
			s.locked = c.LockBreakScreen
		}
	})

	return s
}

func (s *Scene) Name() string    { return SceneName }
func (s *Scene) IsEnabled() bool { return s.locked }

func (s *Scene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
}

func (s *Scene) Load() error {
	if mon := ebiten.Monitor(); mon != nil {
		mw, mh := mon.Size()
		scale := mon.DeviceScaleFactor()
		s.width = int(float64(mw) * scale)
		s.height = int(float64(mh) * scale)
	}

	s.lock.Start(s.breakDur, time.Now())
	s.escCount = 0

	ebiten.SetFullscreen(true)

	if s.locked {
		platform.SetAlwaysOnTop(windowTitle, true)
	}

	return nil
}

func (s *Scene) Unload() error {
	s.lock.Stop()

	if s.locked {
		platform.SetAlwaysOnTop(windowTitle, false)
	}

	ebiten.SetFullscreen(false)

	return nil
}

func (s *Scene) Update() error {
	now := time.Now()

	if s.lock.Complete(now) {
		s.lock.Stop()
		s.onDone()

		return nil
	}

	if !s.locked {
		return nil
	}

	// Soft-lock: reclaim focus if window lost it (e.g. Alt+Tab)
	if !ebiten.IsFocused() {
		platform.RaiseWindow(windowTitle)
	}

	// Locked mode: ESC×3 within 2 seconds to force-exit
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if now.Sub(s.lastEsc) > escResetWindow {
			s.escCount = 0
		}

		s.escCount++
		s.lastEsc = now

		if s.escCount >= escTapsToExit {
			s.lock.Stop()
			s.onDone()
		}
	}

	return nil
}

func (s *Scene) Draw(screen *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)
	now := time.Now()

	// Opaque dark background
	ui.DrawRoundedRect(screen, 0, 0, w, h, 0, color.RGBA{R: 0x10, G: 0x10, B: 0x18, A: 0xF0})

	faceTitle := ui.Face(true, 32)
	faceTime := ui.Face(true, 48)
	faceHint := ui.Face(false, 14)

	ui.DrawText(screen, "Long Break", faceTitle,
		float64(w/2)-ui.Sf(80), float64(h/2)-ui.Sf(100),
		ui.ColorTextPrimary)

	rem := s.lock.Remaining(now)
	totalSecs := int(rem.Seconds())
	mins := totalSecs / 60
	secs := totalSecs % 60
	timeText := fmt.Sprintf("%02d:%02d", mins, secs)

	ui.DrawText(screen, timeText, faceTime,
		float64(w/2)-ui.Sf(70), float64(h/2)-ui.Sf(20),
		ui.ColorAccentBreak)

	barW := ui.S(300)
	barH := ui.S(8)
	barX := (w - barW) / 2
	barY := h/2 + ui.S(40)
	progress := s.lock.Progress(now)

	ui.DrawRoundedRect(screen, barX, barY, barW, barH, ui.S(4),
		color.RGBA{R: 0x30, G: 0x30, B: 0x40, A: 0xFF})
	ui.DrawRoundedRect(screen, barX, barY, barW*float32(progress), barH, ui.S(4),
		ui.ColorAccentBreak)

	hintText := "Relax and rest your eyes"
	if s.locked {
		hintText = "Relax and rest your eyes  (ESC×3 to unlock)"
	}

	ui.DrawText(screen, hintText, faceHint,
		float64(w/2)-ui.Sf(120), float64(h/2)+ui.Sf(80),
		ui.ColorTextSecond)
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
