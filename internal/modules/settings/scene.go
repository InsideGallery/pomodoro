package settings

import (
	"context"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/internal/config"
	"github.com/InsideGallery/pomodoro/internal/ui"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

const SceneName = "settings"

// Scene is the settings scene. It owns a config copy and publishes ConfigChanged events.
type Scene struct {
	*scene.BaseScene

	screen ui.SettingsScreen
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
		_ = config.Save(s.cfg)
		s.publishConfig()
		s.pendingReinit = true
	}
}

func (s *Scene) Load() error {
	s.cfg = config.Load()
	s.screen.Cfg = &s.cfg
	s.screen.Init(s.width, s.height)

	return nil
}

func (s *Scene) Unload() error { return nil }

func (s *Scene) Update() error {
	if s.pendingReinit {
		s.pendingReinit = false
		s.screen.Cfg = &s.cfg
		s.screen.Relayout()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.publishConfig()
		s.onSwitchScene("timer")

		return nil
	}

	s.screen.Update()

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
	}

	return w, h
}
