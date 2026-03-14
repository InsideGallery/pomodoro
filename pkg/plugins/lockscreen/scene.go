package lockscreen

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/platform"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const (
	SceneName      = "lockscreen"
	windowTitle    = "Pomodoro"
	escTapsToExit  = 3
	escResetWindow = 2 * time.Second
)

// LabelEntity is a text label in the Registry.
type LabelEntity struct {
	GetX, GetY func() float64
	FaceSize   float64
	Bold       bool
	Color      func() color.Color
	Text       func() string
}

// ProgressEntity is a progress bar in the Registry.
type ProgressEntity struct {
	X, Y, W, H float32
	Progress   func() float64
	TrackColor color.RGBA
	FillColor  color.Color
}

type Scene struct {
	*scene.BaseScene

	lock     Lock
	breakDur time.Duration
	locked   bool
	onDone   func()

	escCount int
	lastEsc  time.Time

	width, height int
	entityIDSeq   uint64
}

func NewScene(bus *event.Bus, switchToSelf func(), onDone func()) *Scene {
	cfg := config.Load()

	s := &Scene{
		breakDur: cfg.LongBreakDuration(),
		locked:   cfg.PluginEnabled("lock_break_screen", false),
		onDone:   onDone,
	}

	bus.Subscribe(event.LongBreakStarted, func(_ event.Event) {
		if s.locked {
			switchToSelf()
		}
	})

	bus.Subscribe(event.ConfigChanged, func(e event.Event) {
		if c, ok := e.Data.(config.Config); ok {
			s.breakDur = c.LongBreakDuration()
			s.locked = c.PluginEnabled("lock_break_screen", false)
		}
	})

	return s
}

func (s *Scene) Name() string    { return SceneName }
func (s *Scene) IsEnabled() bool { return s.locked }

func (s *Scene) nextID() uint64 {
	s.entityIDSeq++

	return s.entityIDSeq
}

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
	s.createEntities()

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

	if !ebiten.IsFocused() {
		platform.RaiseWindow(windowTitle)
	}

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

func (s *Scene) createEntities() {
	for _, key := range s.Registry.GetKeys() {
		s.Registry.TruncateGroup(key)
	}

	w := float32(s.width)
	h := float32(s.height)

	if err := s.Registry.Add("label", s.nextID(), &LabelEntity{
		GetX:     func() float64 { return float64(w/2) - ui.Sf(80) },
		GetY:     func() float64 { return float64(h/2) - ui.Sf(100) },
		FaceSize: 32, Bold: true,
		Color: func() color.Color { return ui.ColorTextPrimary },
		Text:  func() string { return "Long Break" },
	}); err != nil {
		slog.Warn("registry add", "group", "label", "error", err)
	}

	if err := s.Registry.Add("label", s.nextID(), &LabelEntity{
		GetX:     func() float64 { return float64(w/2) - ui.Sf(70) },
		GetY:     func() float64 { return float64(h/2) - ui.Sf(20) },
		FaceSize: 48, Bold: true,
		Color: func() color.Color { return ui.ColorAccentBreak },
		Text: func() string {
			rem := s.lock.Remaining(time.Now())
			totalSecs := int(rem.Seconds())

			return fmt.Sprintf("%02d:%02d", totalSecs/60, totalSecs%60)
		},
	}); err != nil {
		slog.Warn("registry add", "group", "label", "error", err)
	}

	barW := ui.S(300)
	barH := ui.S(8)
	barX := (w - barW) / 2
	barY := h/2 + ui.S(40)

	if err := s.Registry.Add("progress", s.nextID(), &ProgressEntity{
		X: barX, Y: barY, W: barW, H: barH,
		Progress:   func() float64 { return s.lock.Progress(time.Now()) },
		TrackColor: color.RGBA{R: 0x30, G: 0x30, B: 0x40, A: 0xFF},
		FillColor:  ui.ColorAccentBreak,
	}); err != nil {
		slog.Warn("registry add", "group", "progress", "error", err)
	}

	if err := s.Registry.Add("label", s.nextID(), &LabelEntity{
		GetX:     func() float64 { return float64(w/2) - ui.Sf(120) },
		GetY:     func() float64 { return float64(h/2) + ui.Sf(80) },
		FaceSize: 14, Bold: false,
		Color: func() color.Color { return ui.ColorTextSecond },
		Text: func() string {
			if s.locked {
				return "Relax and rest your eyes  (ESC×3 to unlock)"
			}

			return "Relax and rest your eyes"
		},
	}); err != nil {
		slog.Warn("registry add", "group", "label", "error", err)
	}
}

func (s *Scene) Draw(screen *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)

	ui.DrawRoundedRect(screen, 0, 0, w, h, 0, color.RGBA{R: 0x10, G: 0x10, B: 0x18, A: 0xF0})

	for le := range s.Registry.Iterator("label") {
		l, ok := le.(*LabelEntity)
		if !ok {
			continue
		}

		face := ui.Face(l.Bold, l.FaceSize)
		ui.DrawText(screen, l.Text(), face, l.GetX(), l.GetY(), l.Color())
	}

	for pe := range s.Registry.Iterator("progress") {
		p, ok := pe.(*ProgressEntity)
		if !ok {
			continue
		}

		ui.DrawRoundedRect(screen, p.X, p.Y, p.W, p.H, ui.S(4), p.TrackColor)
		ui.DrawRoundedRect(screen, p.X, p.Y, p.W*float32(p.Progress()), p.H, ui.S(4), p.FillColor)
	}
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
