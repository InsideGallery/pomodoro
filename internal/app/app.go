package app

import (
	"context"
	"fmt"
	"image/color"
	"math"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/internal/event"
	"github.com/InsideGallery/pomodoro/internal/modules/lockscreen"
	"github.com/InsideGallery/pomodoro/internal/modules/metrics"
	"github.com/InsideGallery/pomodoro/internal/modules/minigame"
	"github.com/InsideGallery/pomodoro/internal/modules/settings"
	timerscene "github.com/InsideGallery/pomodoro/internal/modules/timer"
	"github.com/InsideGallery/pomodoro/internal/platform"
	"github.com/InsideGallery/pomodoro/internal/scene"
	"github.com/InsideGallery/pomodoro/internal/tray"
	"github.com/InsideGallery/pomodoro/internal/ui"
)

const (
	DefaultWindowWidth  = 380
	DefaultWindowHeight = 560
	MiniWindowWidth     = 220
	MiniWindowHeight    = 60
	windowTitle         = "Pomodoro"
)

// timerStatus provides timer info for mini mode (defined at consumer, not module).
type timerStatus interface {
	TimerRemaining() time.Duration
	TimerIsRunning() bool
	TimerStateString() string
	OnStartPause() func()
}

// enabledChecker checks if a feature is enabled (defined at consumer).
type enabledChecker interface {
	IsEnabled() bool
}

// Game is the Ebiten game shell. It only handles window/tray concerns
// and delegates all logic to the SceneManager. No concrete scene types stored.
type Game struct {
	bus     *event.Bus
	manager *scene.Manager

	// Interface references — app doesn't know concrete types
	timer   timerStatus
	metrics enabledChecker

	// Window-level state (not scene-specific)
	dragging    bool
	dragOffsetX int
	dragOffsetY int
	hidden      bool
	miniMode    bool
	faceMini    *textv2.GoTextFace

	width, height int
	initialized   bool
}

func New() *Game {
	return &Game{
		bus:     event.NewBus(),
		manager: scene.NewManager(),
		width:   DefaultWindowWidth,
		height:  DefaultWindowHeight,
	}
}

func (g *Game) initApp() {
	ctx := context.Background()

	switchScene := func(name string) { _ = g.manager.SwitchSceneTo(name) }

	switchToTimer := func() {
		switchScene("timer")

		ebiten.SetWindowSize(DefaultWindowWidth, DefaultWindowHeight)
	}

	// Create scenes — composition root is the only place that knows concrete types.
	// After creation, app.go only holds interface references.
	ts := timerscene.NewScene(g.bus, switchScene,
		func() { g.hideToTray() },
		func() { g.enterMini() },
	)

	ss := settings.NewScene(g.bus, switchScene)
	ms := minigame.NewScene(g.bus, func() { switchScene("minigame") }, switchToTimer)
	ls := lockscreen.NewScene(g.bus, func() { switchScene("lockscreen") }, switchToTimer)
	mt := metrics.NewScene(g.bus, switchToTimer)

	g.manager.Add(ctx, ts, ss, ms, ls, mt)
	_ = g.manager.SwitchSceneTo("timer")

	// Store only interfaces — no concrete types after this point
	g.timer = ts
	g.metrics = mt

	// Tray icon updates via events
	g.subscribeTrayIconUpdates()

	g.faceMini = ui.Face(true, 12)
	g.initialized = true
}

func (g *Game) subscribeTrayIconUpdates() {
	setIcon := func(clr color.RGBA) {
		tray.UpdateIcon(tray.GenerateIcon(32, clr))
	}

	iconForState := func() {
		if g.timer == nil {
			return
		}

		switch g.timer.TimerStateString() {
		case "Focus":
			setIcon(color.RGBA{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF})
		case "Break":
			setIcon(color.RGBA{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF})
		case "Long Break":
			setIcon(color.RGBA{R: 0x81, G: 0xEC, B: 0xEC, A: 0xFF})
		case "Paused":
			setIcon(color.RGBA{R: 0xFF, G: 0xC1, B: 0x07, A: 0xFF})
		default:
			setIcon(color.RGBA{R: 0x8B, G: 0x8B, B: 0x9E, A: 0xFF})
		}
	}

	g.bus.Subscribe(event.FocusStarted, func(_ event.Event) { iconForState() })
	g.bus.Subscribe(event.BreakStarted, func(_ event.Event) { iconForState() })
	g.bus.Subscribe(event.LongBreakStarted, func(_ event.Event) { iconForState() })
	g.bus.Subscribe(event.Paused, func(_ event.Event) { iconForState() })
	g.bus.Subscribe(event.Resumed, func(_ event.Event) { iconForState() })
	g.bus.Subscribe(event.Reset, func(_ event.Event) { iconForState() })
	g.bus.Subscribe(event.FocusCompleted, func(_ event.Event) { iconForState() })
	g.bus.Subscribe(event.BreakCompleted, func(_ event.Event) { iconForState() })
	g.bus.Subscribe(event.LongBreakCompleted, func(_ event.Event) { iconForState() })
}

// --- Window-level concerns ---

func (g *Game) hideToTray() {
	g.hidden = true

	platform.HideWindow(windowTitle)
}

func (g *Game) showFromTray() {
	g.hidden = false

	platform.ShowWindow(windowTitle)

	if cur := g.manager.Scene(); cur != nil && cur.Name() != "timer" {
		_ = g.manager.SwitchSceneTo("timer")
	}

	platform.RaiseWindow(windowTitle)
}

func (g *Game) enterMini() {
	g.miniMode = true

	ebiten.SetWindowSize(MiniWindowWidth, MiniWindowHeight)
	platform.SetAlwaysOnTop(windowTitle, true)
}

func (g *Game) exitMini() {
	platform.SetAlwaysOnTop(windowTitle, false)

	g.miniMode = false

	ebiten.SetWindowSize(DefaultWindowWidth, DefaultWindowHeight)
}

func (g *Game) updateDrag() {
	mx, my := ebiten.CursorPosition()

	dragH := int(ui.S(48))
	if g.miniMode {
		dragH = g.height
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && my < dragH {
		btnZone := !g.miniMode && mx >= g.width-int(ui.S(140))
		if !btnZone {
			g.dragging = true
			g.dragOffsetX = mx
			g.dragOffsetY = my
		}
	}

	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.dragging = false
	}

	if g.dragging {
		wx, wy := ebiten.WindowPosition()
		dx := mx - g.dragOffsetX
		dy := my - g.dragOffsetY
		scale := ui.UIScale

		ebiten.SetWindowPosition(wx+int(float64(dx)/scale), wy+int(float64(dy)/scale))
	}
}

func (g *Game) processTrayActions() {
	select {
	case action := <-tray.ActionCh:
		switch action {
		case tray.ActionShow:
			g.showFromTray()
		case tray.ActionMetrics:
			g.showFromTray()

			if g.metrics != nil && g.metrics.IsEnabled() {
				_ = g.manager.SwitchSceneTo("metrics")
			}
		case tray.ActionQuit:
			tray.Quit()
			os.Exit(0)
		}
	default:
	}
}

// --- Ebiten Game interface ---

func (g *Game) Update() error {
	if !g.initialized {
		g.initApp()
	}

	if ebiten.IsWindowBeingClosed() {
		g.hideToTray()
	}

	g.processTrayActions()
	g.updateDrag()

	if g.miniMode {
		g.updateMini()

		return nil
	}

	current := g.manager.Scene()
	if current != nil {
		return current.Update()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.miniMode {
		g.drawMini(screen)

		return
	}

	current := g.manager.Scene()
	if current != nil {
		current.Draw(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	current := g.manager.Scene()
	if current != nil {
		w, h := current.Layout(outsideWidth, outsideHeight)
		g.width = w
		g.height = h

		return w, h
	}

	scale := 1.0
	if m := ebiten.Monitor(); m != nil {
		scale = m.DeviceScaleFactor()
	}

	ui.UIScale = scale

	w := int(math.Ceil(float64(outsideWidth) * scale))
	h := int(math.Ceil(float64(outsideHeight) * scale))
	g.width = w
	g.height = h

	return w, h
}

// --- Mini mode ---

func (g *Game) updateMini() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.exitMini()

		return
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, _ := ebiten.CursorPosition()

		if mx > g.width-int(ui.S(50)) {
			g.exitMini()

			return
		}

		if mx < int(ui.S(44)) && g.timer != nil {
			if fn := g.timer.OnStartPause(); fn != nil {
				fn()
			}
		}
	}
}

func (g *Game) drawMini(screen *ebiten.Image) {
	w := float32(g.width)
	h := float32(g.height)

	ui.DrawRoundedRect(screen, 0, 0, w, h, ui.S(8), ui.ColorWindowBg)
	ui.DrawRoundedRectStroke(screen, 0, 0, w, h, ui.S(8), ui.S(1), ui.ColorCardBorder)

	ppW := ui.S(36)
	ppX := ui.S(8)
	ppY := ui.S(8)
	ppH := h - ui.S(16)

	ui.DrawRoundedRect(screen, ppX, ppY, ppW, ppH, ui.S(6), ui.ColorBgTertiary)

	if g.timer != nil && g.timer.TimerIsRunning() {
		ui.DrawPauseIcon(screen, ppX+ppW/2, ppY+ppH/2, ui.S(18), ui.ColorTextPrimary)
	} else {
		ui.DrawPlayIcon(screen, ppX+ppW/2, ppY+ppH/2, ui.S(18), ui.ColorTextPrimary)
	}

	var rem time.Duration
	if g.timer != nil {
		rem = g.timer.TimerRemaining()
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

	btnW := ui.S(36)
	btnX := w - ui.S(8) - btnW
	btnY := ui.S(8)
	btnH := h - ui.S(16)

	ui.DrawRoundedRect(screen, btnX, btnY, btnW, btnH, ui.S(6), ui.ColorBgTertiary)
	ui.DrawExpandIcon(screen, btnX+btnW/2, btnY+btnH/2, ui.S(16), ui.ColorTextPrimary)
}
