package settings

import (
	"context"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/systems"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const SceneName = "settings"

// Scene is the settings scene with RTree-based input.
type Scene struct {
	*scene.BaseScene

	screen ui.SettingsScreen
	input  *systems.InputSystem
	cfg    config.Config
	bus    *event.Bus

	onSwitchScene func(string)

	width, height int
	pendingReinit bool
}

func NewScene(bus *event.Bus, onSwitchScene func(string)) *Scene {
	return &Scene{
		cfg:           config.Load(),
		bus:           bus,
		onSwitchScene: onSwitchScene,
	}
}

func (s *Scene) Name() string { return SceneName }

func (s *Scene) publishConfig() {
	s.bus.Publish(event.Event{Type: event.ConfigChanged, Data: s.cfg})
}

func (s *Scene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, s.bus)
	s.input = systems.NewInputSystem(s.RTree)

	s.screen.Cfg = &s.cfg
	s.screen.OnBack = func() {
		s.publishConfig()
		s.onSwitchScene("timer")
	}
	s.screen.OnTickVolumeChange = func(_ float64) { s.publishConfig() }
	s.screen.OnAlarmVolumeChange = func(_ float64) { s.publishConfig() }
	s.screen.OnTickEnabledChange = func(_ bool) { s.publishConfig() }
	s.screen.OnMinigameChange = func(_ bool) { s.publishConfig() }
	s.screen.OnThemeChange = func(theme string) {
		ui.SetTheme(ui.ThemeID(theme))
		ui.ApplyTransparency(s.cfg.Transparency)
		s.pendingReinit = true
		s.publishConfig()
	}
	s.screen.OnTransparencyChange = func(t float64) {
		ui.ApplyTransparency(t)
	}
	s.screen.OnResetDefaults = func() {
		def := config.Default()
		def.Theme = s.cfg.Theme
		def.Transparency = s.cfg.Transparency
		s.cfg = def

		_ = config.Save(s.cfg) //nolint:errcheck // best effort

		s.publishConfig()
		s.pendingReinit = true
	}
}

func (s *Scene) Load() error {
	s.cfg = config.Load()
	s.screen.Cfg = &s.cfg

	// On first load, width/height may be 0 (Layout not yet called).
	// Use fallback from Ebiten's current window size.
	if s.width == 0 || s.height == 0 {
		w, h := ebiten.WindowSize()
		scale := 1.0

		if m := ebiten.Monitor(); m != nil {
			scale = m.DeviceScaleFactor()
		}

		s.width = int(float64(w) * scale)
		s.height = int(float64(h) * scale)
	}

	s.screen.Init(s.width, s.height)
	s.registerZones()

	return nil
}

func (s *Scene) Unload() error { return nil }

func (s *Scene) Update() error {
	if s.pendingReinit {
		s.pendingReinit = false
		s.screen.Cfg = &s.cfg
		s.screen.Relayout()
		s.registerZones()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.publishConfig()
		s.onSwitchScene("timer")

		return nil
	}

	// Handle scroll, then update InputSystem offset
	s.screen.HandleScroll()
	s.input.SetScrollOffset(s.screen.ScrollOffset())

	// Run InputSystem for click/drag/hover detection
	if err := s.input.Update(s.Ctx); err != nil {
		return err
	}

	return nil
}

func (s *Scene) Draw(screen *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)
	r := ui.S(12)

	ui.DrawRoundedRect(screen, 0, 0, w, h, r, ui.ColorWindowBg)
	ui.DrawRoundedRectStroke(screen, 0, 0, w, h, r, ui.S(1), ui.ColorCardBorder)

	s.screen.Draw(screen)
}

func (s *Scene) Layout(outsideWidth, outsideHeight int) (int, int) {
	scale := 1.0
	if m := ebiten.Monitor(); m != nil {
		scale = m.DeviceScaleFactor()
	}

	ui.UIScale = scale

	w := int(math.Ceil(float64(outsideWidth) * scale))
	h := int(math.Ceil(float64(outsideHeight) * scale))

	if w != s.width || h != s.height {
		s.width = w
		s.height = h
		s.screen.Resize(w, h)
		s.registerZones()
	}

	return w, h
}

func (s *Scene) registerZones() {
	s.input.ClearZones()

	for _, z := range s.screen.Zones() {
		s.input.AddZone(z)
	}
}
