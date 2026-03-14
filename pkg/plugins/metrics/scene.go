package metrics

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const SceneName = "metrics"

// Scene displays usage statistics. It owns its own Store and event subscriptions.
type Scene struct {
	*scene.BaseScene

	store   *Store
	enabled bool
	onDone  func()

	tab int // 0=Total, 1=Monthly, 2=Weekly

	width, height int
}

// NewScene creates a self-contained metrics scene.
// It creates its own store and subscribes to events for recording.
func NewScene(bus *event.Bus, onDone func()) *Scene {
	cfg := config.Load()

	store := NewStore(DefaultPath())
	if err := store.Load(); err != nil {
		slog.Warn("metrics load", "error", err)
	}

	s := &Scene{
		store:   store,
		enabled: cfg.PluginEnabled("metrics_enabled", false),
		onDone:  onDone,
	}

	// Self-contained event recording
	var phaseStartedAt time.Time

	var gameStartedAt time.Time

	bus.Subscribe(event.FocusStarted, func(e event.Event) {
		phaseStartedAt = e.Time

		store.RecordFocusStart()
	})

	elapsedSince := func(t *time.Time) float64 {
		if t.IsZero() {
			return 0
		}

		elapsed := time.Since(*t).Seconds()
		*t = time.Time{} // reset to prevent double-counting

		return elapsed
	}

	bus.Subscribe(event.FocusCompleted, func(_ event.Event) {
		if secs := elapsedSince(&phaseStartedAt); secs > 0 {
			store.RecordFocusDuration(secs)
		}

		if err := store.Save(); err != nil {
			slog.Warn("metrics save", "error", err)
		}
	})

	bus.Subscribe(event.BreakStarted, func(e event.Event) {
		phaseStartedAt = e.Time

		store.RecordBreakStart()

		if cfg.PluginEnabled("minigame_enabled", false) {
			gameStartedAt = time.Now()

			store.RecordGameStart()
		}
	})

	bus.Subscribe(event.BreakCompleted, func(_ event.Event) {
		if secs := elapsedSince(&phaseStartedAt); secs > 0 {
			store.RecordBreakDuration(secs)
		}

		store.RecordRelaxedBreak()

		if secs := elapsedSince(&gameStartedAt); secs > 0 {
			store.RecordGameDuration(secs)
		}

		if err := store.Save(); err != nil {
			slog.Warn("metrics save", "error", err)
		}
	})

	bus.Subscribe(event.LongBreakStarted, func(e event.Event) {
		phaseStartedAt = e.Time

		store.RecordBreakStart()
	})

	bus.Subscribe(event.LongBreakCompleted, func(_ event.Event) {
		if secs := elapsedSince(&phaseStartedAt); secs > 0 {
			store.RecordBreakDuration(secs)
		}

		if err := store.Save(); err != nil {
			slog.Warn("metrics save", "error", err)
		}
	})

	bus.Subscribe(event.ConfigChanged, func(e event.Event) {
		if c, ok := e.Data.(config.Config); ok {
			s.enabled = c.PluginEnabled("metrics_enabled", false)
		}
	})

	return s
}

func (s *Scene) Name() string    { return SceneName }
func (s *Scene) IsEnabled() bool { return s.enabled }

func (s *Scene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
}

func (s *Scene) Load() error {
	if err := s.store.Load(); err != nil {
		slog.Warn("metrics load", "error", err)
	}

	s.tab = 0

	return nil
}

func (s *Scene) Unload() error { return nil }

func (s *Scene) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.onDone()

		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		s.tab = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		s.tab = 1
	}

	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		s.tab = 2
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && s.tab > 0 {
		s.tab--
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyRight) && s.tab < 2 {
		s.tab++
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		s.handleTabClick(float32(mx), float32(my))
	}

	return nil
}

func (s *Scene) handleTabClick(mx, my float32) {
	w := float32(s.width)
	pad := ui.S(24)
	tabW := (w - pad*2 - ui.S(8)*2) / 3
	tabY := ui.S(56)
	tabH := ui.S(32)

	if my < tabY || my > tabY+tabH {
		return
	}

	for i := range 3 {
		tabX := pad + float32(i)*(tabW+ui.S(8))
		if mx >= tabX && mx <= tabX+tabW {
			s.tab = i

			return
		}
	}
}

func (s *Scene) Draw(screen *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)
	pad := ui.S(24)
	r := ui.S(12)

	ui.DrawRoundedRect(screen, 0, 0, w, h, r, ui.ColorWindowBg)
	ui.DrawRoundedRectStroke(screen, 0, 0, w, h, r, ui.S(1), ui.ColorCardBorder)

	faceTitle := ui.Face(true, 16)
	ui.DrawText(screen, "Metrics", faceTitle, float64(pad), ui.Sf(20), ui.ColorTextPrimary)

	faceHint := ui.Face(false, 10)
	ui.DrawText(screen, "ESC to close", faceHint, float64(w-pad-ui.S(80)), ui.Sf(24), ui.ColorTextSecond)

	s.drawTabs(screen)

	var sum Summary

	switch s.tab {
	case 0:
		sum = s.store.Total()
	case 1:
		sum = s.store.Monthly()
	case 2:
		sum = s.store.Weekly()
	}

	cardY := ui.S(96)
	cardW := w - pad*2
	cardH := h - cardY - pad

	ui.DrawRoundedRect(screen, pad, cardY, cardW, cardH, ui.S(8), ui.ColorCardBg)
	ui.DrawRoundedRectStroke(screen, pad, cardY, cardW, cardH, ui.S(8), ui.S(1), ui.ColorCardBorder)

	s.drawStats(screen, sum, pad, cardY, cardW)
}

func (s *Scene) drawTabs(screen *ebiten.Image) {
	w := float32(s.width)
	pad := ui.S(24)
	tabW := (w - pad*2 - ui.S(8)*2) / 3
	tabH := ui.S(32)
	tabY := ui.S(56)

	labels := []string{"Total", "Monthly", "Weekly"}
	faceTab := ui.Face(true, 11)

	for i, label := range labels {
		tabX := pad + float32(i)*(tabW+ui.S(8))

		bgColor := ui.ColorBgTertiary
		textColor := ui.ColorTextSecond

		if i == s.tab {
			bgColor = ui.ColorAccentFocus
			textColor = ui.ColorTextPrimary
		}

		ui.DrawRoundedRect(screen, tabX, tabY, tabW, tabH, ui.S(6), bgColor)

		tw, _ := textv2.Measure(label, faceTab, 0)
		ui.DrawText(screen, label, faceTab,
			float64(tabX)+float64(tabW)/2-tw/2,
			float64(tabY)+ui.Sf(10), textColor)
	}
}

func (s *Scene) drawStats(screen *ebiten.Image, sum Summary, pad, cardY, cardW float32) {
	faceLabel := ui.Face(false, 12)
	faceValue := ui.Face(true, 14)

	x := pad + ui.S(16)
	y := cardY + ui.S(20)
	rowH := ui.S(36)

	rows := []struct {
		label string
		value string
		color color.RGBA
	}{
		{"Focus Time", formatHours(sum.FocusHours), ui.ColorAccentFocus},
		{"Break Time", formatHours(sum.BreakHours), colorTeal()},
		{"Game Time", formatHours(sum.GameHours), colorMint()},
		{"Focus Sessions", fmt.Sprintf("%d", sum.FocusStarted), ui.ColorAccentFocus},
		{"Breaks Taken", fmt.Sprintf("%d", sum.BreaksStarted), colorTeal()},
		{"Relaxed Breaks", fmt.Sprintf("%d", sum.RelaxedBreaks), colorTeal()},
		{"Games Played", fmt.Sprintf("%d", sum.GamesStarted), colorMint()},
		{"Long Breaks Skipped", fmt.Sprintf("%d", sum.LongBreaksIgnored), colorDanger()},
	}

	for _, row := range rows {
		ui.DrawText(screen, row.label, faceLabel, float64(x), float64(y), ui.ColorTextSecond)

		tw, _ := textv2.Measure(row.value, faceValue, 0)
		ui.DrawText(screen, row.value, faceValue,
			float64(pad+cardW-ui.S(16))-tw, float64(y), row.color)

		y += rowH
		ui.DrawRoundedRect(screen, pad+ui.S(8), y-ui.S(12), cardW-ui.S(16), ui.S(1), 0, ui.ColorBorder)
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

func formatHours(h float64) string {
	if h < 0.1 {
		return fmt.Sprintf("%.0fm", h*60)
	}

	return fmt.Sprintf("%.1fh", h)
}

func colorTeal() color.RGBA   { return color.RGBA{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF} }
func colorMint() color.RGBA   { return color.RGBA{R: 0x55, G: 0xEF, B: 0xC4, A: 0xFF} }
func colorDanger() color.RGBA { return color.RGBA{R: 0xFF, G: 0x6B, B: 0x6B, A: 0xFF} }
