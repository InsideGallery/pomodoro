package app

import (
	"context"
	"image/color"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/internal/event"
	"github.com/InsideGallery/pomodoro/internal/modules/lockscreen"
	"github.com/InsideGallery/pomodoro/internal/modules/metrics"
	"github.com/InsideGallery/pomodoro/internal/modules/mini"
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
	windowTitle         = "Pomodoro"
)

// enabledChecker checks if a feature is enabled (defined at consumer).
type enabledChecker interface {
	IsEnabled() bool
}

// Game is the Ebiten game shell. Pure window management — no domain logic.
type Game struct {
	bus     *event.Bus
	manager *scene.Manager

	// Interface reference for tray metrics check
	metrics enabledChecker

	// Window dragging (generic, works for any undecorated window)
	dragging    bool
	dragOffsetX int
	dragOffsetY int
	hidden      bool

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

	// Create scenes — composition root (only place that knows concrete types)
	ts := timerscene.NewScene(g.bus, switchScene,
		func() { g.hideToTray() },
		func() { switchScene("mini") },
	)

	ss := settings.NewScene(g.bus, switchScene)
	ms := minigame.NewScene(g.bus, func() { switchScene("minigame") }, switchToTimer)
	ls := lockscreen.NewScene(g.bus, func() { switchScene("lockscreen") }, switchToTimer)
	mt := metrics.NewScene(g.bus, switchToTimer)
	mn := mini.NewScene(ts, switchToTimer) // timer scene implements TimerProvider

	g.manager.Add(ctx, ts, ss, ms, ls, mt, mn)
	_ = g.manager.SwitchSceneTo("timer")

	// Store only interface for tray check
	g.metrics = mt

	// Tray icon updates via events (no timer dependency)
	g.subscribeTrayIconUpdates()

	g.initialized = true
}

func (g *Game) subscribeTrayIconUpdates() {
	setIcon := func(clr color.RGBA) {
		tray.UpdateIcon(tray.GenerateIcon(32, clr))
	}

	iconForState := func(state string) {
		switch state {
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

	// Use event Data to carry state string for tray icon
	for _, et := range []event.Type{
		event.FocusStarted, event.BreakStarted, event.LongBreakStarted,
		event.Paused, event.Resumed, event.Reset,
		event.FocusCompleted, event.BreakCompleted, event.LongBreakCompleted,
	} {
		et := et
		g.bus.Subscribe(et, func(e event.Event) {
			if state, ok := e.Data.(string); ok {
				iconForState(state)
			}
		})
	}
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

		ebiten.SetWindowSize(DefaultWindowWidth, DefaultWindowHeight)
	}

	platform.RaiseWindow(windowTitle)
}

func (g *Game) updateDrag() {
	mx, my := ebiten.CursorPosition()

	cur := g.manager.Scene()
	isMini := cur != nil && cur.Name() == "mini"

	dragH := int(ui.S(48))
	if isMini {
		dragH = g.height // entire mini window is draggable
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && my < dragH {
		btnZone := !isMini && mx >= g.width-int(ui.S(140))
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

	current := g.manager.Scene()
	if current != nil {
		return current.Update()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
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
