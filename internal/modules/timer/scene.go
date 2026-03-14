package timer

import (
	"context"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/internal/audio"
	"github.com/InsideGallery/pomodoro/internal/config"
	tsystems "github.com/InsideGallery/pomodoro/internal/modules/timer/systems"
	"github.com/InsideGallery/pomodoro/internal/timer"
	"github.com/InsideGallery/pomodoro/internal/ui"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/systems"
)

const (
	SceneName    = "timer"
	WindowWidth  = 380
	WindowHeight = 560
)

// Scene is the main timer scene. It owns the timer domain, audio, and state persistence.
type Scene struct {
	*scene.BaseScene

	tmr   *timer.Timer
	audio *audio.Manager
	bus   *event.Bus
	tick  *tsystems.TickSystem
	input *systems.InputSystem

	screen ui.TimerScreen

	onSwitchScene func(string)
	onClose       func()
	onMini        func()

	width, height int
}

// NewScene creates the timer scene. It owns timer creation, audio, and state.
func NewScene(
	bus *event.Bus,
	onSwitchScene func(string),
	onClose func(),
	onMini func(),
) *Scene {
	cfg := config.Load()

	tmr := timer.New(timer.Config{
		FocusDuration:     cfg.FocusDuration(),
		BreakDuration:     cfg.BreakDuration(),
		LongBreakDuration: cfg.LongBreakDuration(),
		RoundsBeforeLong:  cfg.RoundsBeforeLong,
		AutoStart:         cfg.AutoStart,
	})

	st := config.LoadState()
	tmr.Restore(st.State, st.PrePause, st.PendingNext, st.Round, st.RemainingSec, time.Now())

	s := &Scene{
		tmr:           tmr,
		bus:           bus,
		onSwitchScene: onSwitchScene,
		onClose:       onClose,
		onMini:        onMini,
		width:         WindowWidth,
		height:        WindowHeight,
	}

	s.tick = &tsystems.TickSystem{
		Tmr:       tmr,
		Bus:       bus,
		SaveState: s.saveState,
	}

	tmr.OnComplete = func(completed timer.State) {
		if s.audio != nil {
			s.audio.StopTick()
			s.audio.PlayAlarm()
		}

		s.tick.PublishCompleted(completed)
		s.saveState()
	}

	// React to config changes
	bus.Subscribe(event.ConfigChanged, func(e event.Event) {
		if c, ok := e.Data.(config.Config); ok {
			tmr.SetConfig(timer.Config{
				FocusDuration:     c.FocusDuration(),
				BreakDuration:     c.BreakDuration(),
				LongBreakDuration: c.LongBreakDuration(),
				RoundsBeforeLong:  c.RoundsBeforeLong,
				AutoStart:         c.AutoStart,
			})
		}
	})

	return s
}

func (s *Scene) Name() string { return SceneName }

// OnStartPause returns the start/pause callback for use by mini mode.
func (s *Scene) OnStartPause() func() { return s.tick.OnStartPause }

// TimerRemaining returns the current timer remaining duration.
func (s *Scene) TimerRemaining() time.Duration { return s.tmr.Remaining(time.Now()) }

// TimerStateString returns the current timer state as a string.
func (s *Scene) TimerStateString() string { return s.tmr.State().String() }

// TimerIsRunning returns whether the timer is currently running.
func (s *Scene) TimerIsRunning() bool { return s.tmr.State().IsRunning() }

func (s *Scene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, s.bus)
	s.input = systems.NewInputSystem(s.RTree)

	s.initAudio()

	// Systems in execution order: input first, then keyboard, tick, render
	s.Systems.Add("input", s.input)
	s.Systems.Add("keyboard", &tsystems.KeyboardSystem{
		OnStartPause: s.tick.OnStartPause,
		OnReset:      s.tick.OnReset,
		OnSettings:   func() { s.onSwitchScene("settings") },
	})
	s.Systems.Add("tick", s.tick)
	s.Systems.Add("render", &tsystems.RenderSystem{
		Screen: &s.screen,
		Tmr:    s.tmr,
	})
}

func (s *Scene) Load() error {
	s.screen.Timer = s.tmr
	s.screen.OnStart = s.tick.OnStartPause
	s.screen.OnReset = s.tick.OnReset
	s.screen.OnSkip = s.tick.OnSkip
	s.screen.OnSettings = func() { s.onSwitchScene("settings") }
	s.screen.OnClose = s.onClose
	s.screen.OnMini = s.onMini
	s.screen.OnSetRound = func(r int) {
		s.tmr.SetRound(r)
		s.saveState()
	}
	s.screen.OnAdjustTime = func(rem time.Duration) {
		s.tmr.SetRemaining(rem, time.Now())
	}
	s.screen.Init(s.width, s.height)

	// Register button zones in RTree for centralized hit detection
	s.registerZones()

	return nil
}

func (s *Scene) registerZones() {
	s.input.ClearZones()

	// Mark buttons as RTree-managed (skip self-hit-detection)
	s.screen.BtnStart.SetManagedByRTree()
	s.screen.BtnReset.SetManagedByRTree()
	s.screen.BtnSkip.SetManagedByRTree()
	s.screen.BtnSettings.SetManagedByRTree()
	s.screen.BtnClose.SetManagedByRTree()
	s.screen.BtnMini.SetManagedByRTree()

	// Register zones via RTree
	s.input.AddZone(ui.ButtonZone(&s.screen.BtnStart))
	s.input.AddZone(ui.ButtonZone(&s.screen.BtnReset))
	s.input.AddZone(ui.ButtonZone(&s.screen.BtnSkip))
	s.input.AddZone(ui.ButtonZone(&s.screen.BtnSettings))
	s.input.AddZone(ui.ButtonZone(&s.screen.BtnClose))
	s.input.AddZone(ui.ButtonZone(&s.screen.BtnMini))
}

func (s *Scene) Unload() error { return nil }

func (s *Scene) Update() error { return s.BaseScene.Update() }

func (s *Scene) Draw(screen *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)
	r := ui.S(12)

	ui.DrawRoundedRect(screen, 0, 0, w, h, r, ui.ColorWindowBg)
	ui.DrawRoundedRectStroke(screen, 0, 0, w, h, r, ui.S(1), ui.ColorCardBorder)

	s.BaseScene.Draw(screen)
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

func (s *Scene) initAudio() {
	am, err := audio.NewManager()
	if err != nil {
		return
	}

	s.audio = am
	s.tick.Audio = am

	cfg := config.Load()
	am.SetTickVolume(cfg.TickVolume)
	am.SetAlarmVolume(cfg.AlarmVolume)
	am.SetTickEnabled(cfg.TickEnabled)

	s.bus.Subscribe(event.ConfigChanged, func(e event.Event) {
		if c, ok := e.Data.(config.Config); ok {
			am.SetTickVolume(c.TickVolume)
			am.SetAlarmVolume(c.AlarmVolume)
			am.SetTickEnabled(c.TickEnabled)
		}
	})
}

func (s *Scene) saveState() {
	st := config.LoadState() // preserve other modules' fields

	state, prePause, pendingNext, round, remainingSec := s.tmr.Snapshot(time.Now())
	st.Round = round
	st.PendingNext = pendingNext
	st.State = state
	st.PrePause = prePause
	st.RemainingSec = remainingSec

	_ = config.SaveState(st)
}
