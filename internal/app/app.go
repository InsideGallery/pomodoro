package app

import (
	"context"
	"image/color"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/internal/modules/mini"
	"github.com/InsideGallery/pomodoro/internal/modules/settings"
	timerscene "github.com/InsideGallery/pomodoro/internal/modules/timer"
	"github.com/InsideGallery/pomodoro/internal/tray"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/platform"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const (
	DefaultWindowWidth  = 380
	DefaultWindowHeight = 560
	windowTitle         = "Pomodoro"
)

// Game is the Ebiten game shell. Pure window management — no domain logic.
type Game struct {
	bus     *event.Bus
	manager *scene.Manager

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

	// Core scenes (always loaded, not disableable)
	ts := timerscene.NewScene(g.bus, switchScene,
		func() { g.hideToTray() },
		func() { switchScene("mini") },
	)

	ss := settings.NewScene(g.bus, switchScene)
	mn := mini.NewScene(ts, switchToTimer)

	g.manager.Add(ctx, ts, ss, mn)

	// Load external plugins (minigame, lockscreen, metrics, and any user plugins)
	loader := pluggable.NewLoader(pluggable.DefaultPluginDir())
	_ = loader.Load()

	for _, mod := range loader.Modules() {
		scenes := mod.Scenes(g.bus)

		for _, sc := range scenes {
			g.manager.Add(ctx, sc)
		}

		// Register tray items from plugins
		for label, sceneName := range mod.TrayItems() {
			name := sceneName // capture for closure

			tray.AddMenuItem(label, func() {
				g.showFromTray()
				switchScene(name)
			})
		}
	}

	_ = g.manager.SwitchSceneTo("timer")

	// Tray icon updates via events
	g.subscribeTrayIconUpdates()

	g.initialized = true
}

func (g *Game) subscribeTrayIconUpdates() {
	setIcon := func(clr color.RGBA) {
		tray.UpdateIcon(tray.GenerateIcon(32, clr))
	}

	for _, et := range []event.Type{
		event.FocusStarted, event.BreakStarted, event.LongBreakStarted,
		event.Paused, event.Resumed, event.Reset,
		event.FocusCompleted, event.BreakCompleted, event.LongBreakCompleted,
	} {
		g.bus.Subscribe(et, func(e event.Event) {
			if state, ok := e.Data.(string); ok {
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
		dragH = g.height
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
