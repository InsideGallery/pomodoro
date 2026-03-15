package app

import (
	"context"
	"log/slog"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

// SetupFunc configures scenes and plugins for a specific product.
// Called once during init. Returns the name of the initial scene.
type SetupFunc func(ctx context.Context, bus *event.Bus, manager *scene.Manager, switchScene func(string)) string

// Config configures the app shell.
type Config struct {
	Width, Height  int
	Title          string
	Transparent    bool
	Decorated      bool
	DragEnabled    bool
	HandleWinClose func() // called when window X is clicked (nil = ignore)
	Setup          SetupFunc
}

// Game is the Ebiten game shell. Pure window management — no domain logic.
type Game struct {
	cfg     Config
	bus     *event.Bus
	manager *scene.Manager

	dragging    bool
	dragOffsetX int
	dragOffsetY int

	width, height int
	initialized   bool
}

func New(cfg Config) *Game {
	if cfg.Width == 0 {
		cfg.Width = 380
	}

	if cfg.Height == 0 {
		cfg.Height = 560
	}

	return &Game{
		cfg:     cfg,
		bus:     event.NewBus(),
		manager: scene.NewManager(),
		width:   cfg.Width,
		height:  cfg.Height,
	}
}

func (g *Game) Bus() *event.Bus         { return g.bus }
func (g *Game) Manager() *scene.Manager { return g.manager }

func (g *Game) initApp() {
	ctx := context.Background()

	switchScene := func(name string) {
		if err := g.manager.SwitchSceneTo(name); err != nil {
			slog.Warn("switch scene", "name", name, "error", err)
		}
	}

	// Load external .so plugins
	loader := pluggable.NewLoader(pluggable.DefaultPluginDir())
	if err := loader.Load(); err != nil {
		slog.Warn("load plugins", "error", err)
	}

	for _, mod := range loader.Modules() {
		scenes := mod.Scenes(g.bus, pluggable.SceneSwitcher(switchScene))

		for _, sc := range scenes {
			g.manager.Add(ctx, sc)
		}
	}

	// Product-specific setup
	initialScene := ""
	if g.cfg.Setup != nil {
		initialScene = g.cfg.Setup(ctx, g.bus, g.manager, switchScene)
	}

	if initialScene != "" {
		if err := g.manager.SwitchSceneTo(initialScene); err != nil {
			panic("failed to switch to initial scene " + initialScene + ": " + err.Error())
		}
	}

	g.initialized = true
}

func (g *Game) Update() error {
	if !g.initialized {
		g.initApp()
	}

	if ebiten.IsWindowBeingClosed() && g.cfg.HandleWinClose != nil {
		g.cfg.HandleWinClose()
	}

	if g.cfg.DragEnabled {
		g.updateDrag()
	}

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
