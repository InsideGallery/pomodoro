package settings

import (
	"context"
	"fmt"
	"image/color"
	"math"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	ssystems "github.com/InsideGallery/pomodoro/internal/modules/settings/systems"
	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/logger"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/systems"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const SceneName = "settings"

type Scene struct {
	*scene.BaseScene

	input    *systems.InputSystem
	scroll   *ssystems.ScrollSystem
	renderer *ssystems.RenderSystem
	cfg      config.Config
	bus      *event.Bus

	onSwitchScene func(string)
	plugins       []pluggable.Module

	width, height int
	pendingReinit bool
	entityIDSeq   uint64
}

func NewScene(bus *event.Bus, onSwitchScene func(string), plugins []pluggable.Module) *Scene {
	return &Scene{
		cfg:           config.Load(),
		bus:           bus,
		onSwitchScene: onSwitchScene,
		plugins:       plugins,
	}
}

func (s *Scene) Name() string { return SceneName }

func (s *Scene) nextID() uint64 {
	s.entityIDSeq++

	return s.entityIDSeq
}

func (s *Scene) publishConfig() {
	s.bus.Publish(event.Event{Type: event.ConfigChanged, Data: s.cfg})
}

func (s *Scene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, s.bus)
	s.input = systems.NewInputSystem(s.RTree)
	s.scroll = &ssystems.ScrollSystem{ContentTop: ui.S(58)}

	s.renderer = &ssystems.RenderSystem{
		Reg:    s.Registry,
		Scroll: s.scroll,
	}

	s.Systems.Add("scroll", s.scroll)
	s.Systems.Add("input", s.input)
	s.Systems.Add("render", s.renderer)
}

func (s *Scene) Load() error {
	s.cfg = config.Load()

	if s.width == 0 || s.height == 0 {
		w, h := ebiten.WindowSize()
		scale := 1.0

		if m := ebiten.Monitor(); m != nil {
			scale = m.DeviceScaleFactor()
		}

		s.width = int(float64(w) * scale)
		s.height = int(float64(h) * scale)
	}

	s.renderer.Width = s.width
	s.renderer.Height = s.height
	s.scroll.ViewportH = float32(s.height) - ui.S(48) - ui.S(24) // cardH
	s.scroll.ScrollY = 0

	s.createEntities()

	return nil
}

func (s *Scene) Unload() error { return nil }

func (s *Scene) Update() error {
	if s.pendingReinit {
		s.pendingReinit = false
		s.cfg = config.Load()
		s.createEntities()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.publishConfig()
		s.onSwitchScene("timer")

		return nil
	}

	// Update scroll, then set offset for InputSystem
	if err := s.scroll.Update(s.Ctx); err != nil {
		return err
	}

	s.input.SetScrollOffset(s.scroll.Offset())

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

	s.renderer.Draw(s.Ctx, screen)
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
		s.renderer.Width = w
		s.renderer.Height = h
		s.scroll.ViewportH = float32(h) - ui.S(48) - ui.S(24)
		s.createEntities()
	}

	return w, h
}

func (s *Scene) createEntities() {
	s.input.ClearZones()

	for _, key := range s.Registry.GetKeys() {
		s.Registry.TruncateGroup(key)
	}

	w := float32(s.width)
	pad := ui.S(24)
	cardW := w - pad*2
	faceBtn := ui.Face(true, 13)

	// Fixed back button (not scrollable)
	back := &ssystems.SettingsButton{
		Label: "",
		X:     pad, Y: ui.S(10), W: ui.S(32), H: ui.S(32),
		Color:      color.RGBA{},
		HoverColor: ui.ColorBgTertiary,
		TextColor:  ui.ColorTextSecond,
		IconDraw:   ui.DrawBackIcon,
		Fixed:      true,
		OnClick: func() {
			s.publishConfig()
			s.onSwitchScene("timer")
		},
	}

	if err := s.Registry.Add("fixed_button", s.nextID(), back); err != nil {
		logger.Warn("registry add", "group", "fixed_button", "error", err)
	}

	// Back button zone (no scroll offset — fixed position)
	s.input.AddZone(&systems.Zone{
		Spatial: shapes.NewBox(shapes.NewPoint(float64(back.X), float64(back.Y)), float64(back.W), float64(back.H)),
		OnClick: back.OnClick,
		OnHover: func(h bool) { back.Hovered = h },
	})

	// --- Scrollable content (Y=0 = top of scroll area) ---
	y := float32(0)
	sliderW := cardW - ui.S(32)
	sliderX := pad + ui.S(16)
	sliderH := ui.S(28)
	sliderGap := ui.S(48)
	toggleW := ui.S(44)
	toggleH := ui.S(24)
	toggleX := pad + cardW - ui.S(16) - toggleW
	toggleRowH := ui.S(36)

	// Section: TIMER
	s.addSection("TIMER", ui.ColorAccentFocus, y)
	y += ui.S(24)

	s.addSlider("Focus", sliderX, y, sliderW, sliderH, 1, 60, float64(s.cfg.FocusMinutes),
		func(v float64) string { return fmt.Sprintf("%.0f min", v) },
		func(v float64) {
			s.cfg.FocusMinutes = int(math.Round(v))
			s.save()
			s.publishConfig()
		})
	y += sliderGap

	s.addSlider("Short Break", sliderX, y, sliderW, sliderH, 1, 30, float64(s.cfg.BreakMinutes),
		func(v float64) string { return fmt.Sprintf("%.0f min", v) },
		func(v float64) {
			s.cfg.BreakMinutes = int(math.Round(v))
			s.save()
			s.publishConfig()
		})
	y += sliderGap

	s.addSlider("Long Break", sliderX, y, sliderW, sliderH, 1, 60, float64(s.cfg.LongBreakMinutes),
		func(v float64) string { return fmt.Sprintf("%.0f min", v) },
		func(v float64) {
			s.cfg.LongBreakMinutes = int(math.Round(v))
			s.save()
			s.publishConfig()
		})
	y += sliderGap

	s.addSlider("Rounds", sliderX, y, sliderW, sliderH, 1, 10, float64(s.cfg.RoundsBeforeLong),
		func(v float64) string { return fmt.Sprintf("%.0f", v) },
		func(v float64) {
			s.cfg.RoundsBeforeLong = int(math.Round(v))
			s.save()
			s.publishConfig()
		})
	y += sliderGap

	// Section: SOUND
	s.addSection("SOUND", ui.ColorAccentSuccess, y)
	y += ui.S(24)

	s.addSlider("Tick Volume", sliderX, y, sliderW, sliderH, 0, 1, s.cfg.TickVolume,
		nil,
		func(v float64) {
			s.cfg.TickVolume = v
			s.save()
			s.publishConfig()
		})
	y += sliderGap

	s.addSlider("Alarm Volume", sliderX, y, sliderW, sliderH, 0, 1, s.cfg.AlarmVolume,
		nil,
		func(v float64) {
			s.cfg.AlarmVolume = v
			s.save()
			s.publishConfig()
		})
	y += sliderGap

	s.addToggle("Tick Sound", toggleX, y, toggleW, toggleH, s.cfg.TickEnabled,
		ui.ColorAccentSuccess, ui.ColorToggleOff,
		func(v bool) { s.cfg.TickEnabled = v; s.save(); s.publishConfig() })
	y += toggleRowH

	s.addToggle("Auto-Start Next", toggleX, y, toggleW, toggleH, s.cfg.AutoStart,
		ui.ColorAccentSuccess, ui.ColorToggleOff,
		func(v bool) { s.cfg.AutoStart = v; s.save(); s.publishConfig() })
	y += toggleRowH

	// Plugin toggles
	for _, mod := range s.plugins {
		key := mod.ConfigKey()

		s.addToggle(mod.Name(), toggleX, y, toggleW, toggleH,
			s.cfg.PluginEnabled(key, mod.DefaultEnabled()),
			ui.ColorAccentBreak, ui.ColorToggleOff,
			func(v bool) { s.cfg.SetPlugin(key, v); s.save(); s.publishConfig() })
		y += toggleRowH
	}

	// Section: APPEARANCE
	s.addSection("APPEARANCE", ui.ColorAccentFocus, y)
	y += ui.S(24)

	s.addToggle("Light Theme", toggleX, y, toggleW, toggleH, s.cfg.Theme == "light",
		ui.ColorAccentFocus, ui.ColorToggleOff,
		func(v bool) {
			if v {
				s.cfg.Theme = "light"
			} else {
				s.cfg.Theme = "dark"
			}

			ui.SetTheme(ui.ThemeID(s.cfg.Theme))
			ui.ApplyTransparency(s.cfg.Transparency)
			s.save()
			s.publishConfig()
			s.pendingReinit = true
		})
	y += toggleRowH

	s.addSlider("Transparency", sliderX, y, sliderW, sliderH, 0.1, 0.9, s.cfg.Transparency,
		func(v float64) string { return fmt.Sprintf("%.0f%%", v*100) },
		func(v float64) {
			s.cfg.Transparency = v
			ui.ApplyTransparency(v)
			s.save()
		})
	y += sliderGap

	// Reset Defaults button
	resetBtn := &ssystems.SettingsButton{
		Label:      "Reset Defaults",
		Face:       faceBtn,
		X:          (w - ui.S(140)) / 2,
		Y:          y,
		W:          ui.S(140),
		H:          ui.S(36),
		Color:      ui.ColorBgTertiary,
		HoverColor: ui.ColorBorder,
		TextColor:  ui.ColorAccentDanger,
		OnClick: func() {
			def := config.Default()
			def.Theme = s.cfg.Theme
			def.Transparency = s.cfg.Transparency
			s.cfg = def
			s.save()
			s.publishConfig()
			s.pendingReinit = true
		},
	}

	if err := s.Registry.Add("button", s.nextID(), resetBtn); err == nil {
		s.input.AddZone(&systems.Zone{
			Spatial: shapes.NewBox(shapes.NewPoint(float64(resetBtn.X), float64(resetBtn.Y)),
				float64(resetBtn.W), float64(resetBtn.H)),
			OnClick: resetBtn.OnClick,
			OnHover: func(h bool) { resetBtn.Hovered = h },
		})
	}

	y += ui.S(48)

	s.scroll.ContentH = y
}

func (s *Scene) addSection(text string, clr color.RGBA, y float32) {
	if err := s.Registry.Add("section", s.nextID(), &ssystems.SectionLabel{Text: text, Color: clr, Y: y}); err != nil {
		logger.Warn("registry add", "group", "section", "error", err)
	}
}

func (s *Scene) addSlider(label string, x, y, w, h float32, minV, maxV, value float64,
	format func(float64) string, onChange func(float64),
) {
	sl := &ssystems.SliderEntity{
		Label: label, X: x, Y: y, W: w, H: h,
		Min: minV, Max: maxV, Value: value,
		FormatValue: format, OnChange: onChange,
		TrackColor: ui.ColorSliderTrack, KnobColor: ui.ColorAccentBreak,
	}

	if err := s.Registry.Add("slider", s.nextID(), sl); err != nil {
		return
	}

	pad := float64(12)

	s.input.AddZone(&systems.Zone{
		Spatial: shapes.NewBox(
			shapes.NewPoint(float64(x)-pad, float64(y)-pad),
			float64(w)+pad*2, float64(h)+pad*2),
		OnDragStart: func() {},
		OnDrag: func(mx, _ int) {
			t := float64(float32(mx)-x) / float64(w)
			if t < 0 {
				t = 0
			}

			if t > 1 {
				t = 1
			}

			newVal := minV + t*(maxV-minV)
			if newVal != sl.Value {
				sl.Value = newVal
				if sl.OnChange != nil {
					sl.OnChange(newVal)
				}
			}
		},
		OnDragEnd: func() {},
	})
}

func (s *Scene) addToggle(label string, x, y, w, h float32, value bool,
	onColor, offColor color.Color, onChange func(bool),
) {
	tg := &ssystems.ToggleEntity{
		Label: label, X: x, Y: y, W: w, H: h,
		Value: value, OnChange: onChange,
		OnColor: onColor, OffColor: offColor,
	}

	if err := s.Registry.Add("toggle", s.nextID(), tg); err != nil {
		return
	}

	// Hit area includes label (200px to the left)
	lx := float64(x) - 200
	if lx < 0 {
		lx = 0
	}

	hitW := float64(x+w) - lx

	s.input.AddZone(&systems.Zone{
		Spatial: shapes.NewBox(shapes.NewPoint(lx, float64(y)-4), hitW, float64(h)+8),
		OnClick: func() {
			tg.Value = !tg.Value
			if tg.OnChange != nil {
				tg.OnChange(tg.Value)
			}
		},
	})
}

func (s *Scene) save() {
	if err := config.Save(s.cfg); err != nil {
		logger.Warn("save config", "error", err)
	}
}
