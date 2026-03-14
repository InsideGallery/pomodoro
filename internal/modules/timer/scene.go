package timer

import (
	"context"
	"image/color"
	"log/slog"
	"math"
	"time"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/internal/audio"
	tsystems "github.com/InsideGallery/pomodoro/internal/modules/timer/systems"
	"github.com/InsideGallery/pomodoro/internal/timer"
	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/ecs"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/systems"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const (
	SceneName    = "timer"
	WindowWidth  = 380
	WindowHeight = 560
)

type Scene struct {
	*scene.BaseScene

	tmr   *timer.Timer
	audio *audio.Manager
	bus   *event.Bus
	tick  *tsystems.TickSystem
	input *systems.InputSystem

	onSwitchScene func(string)
	onClose       func()
	onMini        func()

	width, height int
	entityIDSeq   uint64
}

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

func (s *Scene) OnStartPause() func() { return s.tick.OnStartPause }

func (s *Scene) TimerRemaining() time.Duration { return s.tmr.Remaining(time.Now()) }

func (s *Scene) TimerStateString() string { return s.tmr.State().String() }

func (s *Scene) TimerIsRunning() bool { return s.tmr.State().IsRunning() }

func (s *Scene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, s.bus)
	s.input = systems.NewInputSystem(s.RTree)

	s.initAudio()

	s.Systems.Add("input", s.input)
	s.Systems.Add("keyboard", &tsystems.KeyboardSystem{
		OnStartPause: s.tick.OnStartPause,
		OnReset:      s.tick.OnReset,
		OnSettings:   func() { s.onSwitchScene("settings") },
	})
	s.Systems.Add("tick", s.tick)
	s.Systems.Add("render", &tsystems.RenderSystem{
		Reg: s.Registry,
		Tmr: s.tmr,
	})
}

func (s *Scene) Load() error {
	if s.width == 0 || s.height == 0 {
		w, h := ebiten.WindowSize()
		scale := 1.0

		if m := ebiten.Monitor(); m != nil {
			scale = m.DeviceScaleFactor()
		}

		s.width = int(float64(w) * scale)
		s.height = int(float64(h) * scale)
	}

	s.createEntities()

	return nil
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
		s.createEntities() // re-layout
	}

	return w, h
}

func (s *Scene) nextID() uint64 {
	s.entityIDSeq++

	return s.entityIDSeq
}

func (s *Scene) createEntities() {
	// Clear previous entities and RTree zones
	s.input.ClearZones()

	for _, key := range s.Registry.GetKeys() {
		s.Registry.TruncateGroup(key)
	}

	w := float32(s.width)
	h := float32(s.height)
	pad := ui.S(24)
	cardY := ui.S(48)
	cardH := h - cardY - pad
	cardW := w - pad*2
	iconS := ui.S(32)

	faceTimer := ui.Face(true, 56)
	faceMode := ui.Face(true, 13)
	faceSmall := ui.Face(false, 12)
	faceBtn := ui.Face(true, 13)

	// --- Title bar buttons ---
	s.addButton("button", w-pad-iconS, ui.S(10), iconS, iconS,
		color.RGBA{}, color.RGBA{R: 0xFF, G: 0x6B, B: 0x6B, A: 0x30},
		ui.ColorTextSecond, "", nil, ui.DrawCloseIcon, s.onClose)

	s.addButton("button", w-pad-iconS*2-ui.S(8), ui.S(10), iconS, iconS,
		color.RGBA{}, ui.ColorBgTertiary,
		ui.ColorTextSecond, "", nil, ui.DrawMinimizeIcon, s.onMini)

	s.addButton("button", w-pad-iconS*3-ui.S(16), ui.S(10), iconS, iconS,
		color.RGBA{}, ui.ColorBgTertiary,
		ui.ColorTextSecond, "", nil, ui.DrawSettingsIcon,
		func() { s.onSwitchScene("settings") })

	// --- Control buttons ---
	btnW := ui.S(96)
	btnH := ui.S(40)
	gap := ui.S(10)
	totalW := btnW*3 + gap*2
	startX := (w - totalW) / 2
	btnY := h - pad - btnH - ui.S(16)

	s.addButton("button", startX, btnY, btnW, btnH,
		ui.ColorAccentSuccess, ui.ColorAccentSuccess,
		ui.ColorBgPrimary, "Focus", faceBtn, nil, s.tick.OnStartPause)

	s.addButton("button", startX+btnW+gap, btnY, btnW, btnH,
		ui.ColorBgTertiary, ui.ColorBorder,
		ui.ColorTextPrimary, "Skip", faceBtn, nil, s.tick.OnSkip)

	s.addButton("button", startX+(btnW+gap)*2, btnY, btnW, btnH,
		ui.ColorBgTertiary, ui.ColorBgTertiary,
		ui.ColorAccentDanger, "Reset", faceBtn, nil, s.tick.OnReset)

	// --- Ring ---
	ringCX := float64(w / 2)
	ringCY := float64(cardY + cardH*0.38)
	maxR := math.Min(float64(cardW)*0.28, float64(cardH)*0.26)

	ring := &tsystems.RingEntityData{CX: ringCX, CY: ringCY, OuterR: maxR, TrackColor: ui.ColorBgTertiary}

	if err := s.Registry.Add("ring", s.nextID(), ring); err == nil {
		// Ring drag zone
		s.input.AddZone(&systems.Zone{
			Spatial:     box(ringCX-maxR, ringCY-maxR, maxR*2, maxR*2),
			OnDragStart: func() {},
			OnDrag: func(mx, my int) {
				state := s.tmr.State()
				if !state.IsRunning() && state != timer.StatePaused {
					return
				}

				dx := float64(mx) - ringCX
				dy := float64(my) - ringCY
				angle := math.Atan2(dy, dx)
				progress := (angle + math.Pi/2) / (2 * math.Pi)

				if progress < 0 {
					progress++
				}

				total := s.tmr.TotalDuration()
				rem := time.Duration(float64(total) * (1 - progress))

				if rem < time.Second {
					rem = time.Second
				}

				s.tmr.SetRemaining(rem, time.Now())
			},
			OnDragEnd: func() {},
		})
	}

	// --- Mode label ---
	modeLabel := &ecs.ModeLabelEntity{
		Pos:  ecs.Position{X: ringCX, Y: ringCY - maxR*0.38},
		Face: faceMode,
		ColorFunc: func() color.Color {
			return s.accentForState(s.tmr.State())
		},
		TextFunc: func() string {
			state := s.tmr.State()
			if state == timer.StateIdle {
				return s.tmr.PendingNext().String()
			}

			if state == timer.StatePaused {
				return "Paused"
			}

			return state.String()
		},
	}

	if err := s.Registry.Add("mode_label", s.nextID(), modeLabel); err != nil {
		slog.Warn("registry add", "group", "mode_label", "error", err)
	}

	// --- Timer text ---
	timerText := &ecs.TimerTextEntity{
		Pos:       ecs.Position{X: ringCX, Y: ringCY + maxR*0.08},
		Face:      faceTimer,
		Color:     ui.ColorTextPrimary,
		Remaining: func() time.Duration { return s.tmr.Remaining(time.Now()) },
	}

	if err := s.Registry.Add("timer_text", s.nextID(), timerText); err != nil {
		slog.Warn("registry add", "group", "timer_text", "error", err)
	}

	// --- Hint ---
	hint := &ecs.HintEntity{
		Pos:   ecs.Position{X: ringCX, Y: ringCY + maxR + ui.Sf(18)},
		Face:  faceSmall,
		Color: ui.ColorTextSecond,
		TextFunc: func() string {
			if s.tmr.State() != timer.StateIdle {
				return ""
			}

			switch s.tmr.PendingNext() {
			case timer.StateBreak:
				return "Time for a break"
			case timer.StateLongBreak:
				return "Time for a long break"
			default:
				return "Ready to focus"
			}
		},
	}

	if err := s.Registry.Add("hint", s.nextID(), hint); err != nil {
		slog.Warn("registry add", "group", "hint", "error", err)
	}

	// --- Round dots ---
	dotY := ringCY + maxR + ui.Sf(44)
	dots := &tsystems.RoundDotsEntityData{CX: ringCX, CY: dotY}

	if err := s.Registry.Add("round_dots", s.nextID(), dots); err == nil {
		// Create clickable zones for each dot
		total := s.tmr.Config().RoundsBeforeLong
		if total > 0 {
			dotR := ui.S(4)
			dotGap := ui.S(12)
			dotTotalW := float64(total)*float64(dotR*2) + float64(total-1)*float64(dotGap)
			dotStartX := ringCX - dotTotalW/2 + float64(dotR)

			for i := range total {
				idx := i
				cx := dotStartX + float64(i)*(float64(dotR*2+dotGap))
				hitR := float64(dotR + ui.S(8))

				s.input.AddZone(&systems.Zone{
					Spatial: box(cx-hitR, dotY-hitR, hitR*2, hitR*2),
					OnClick: func() {
						state := s.tmr.State()
						if state == timer.StateIdle || state == timer.StatePaused {
							s.tmr.SetRound(idx)
							s.saveState()
						}
					},
				})
			}
		}
	}
}

func (s *Scene) addButton(
	group string,
	x, y, w, h float32,
	clr, hoverClr color.Color,
	textClr color.Color,
	label string,
	face *textv2.GoTextFace,
	iconDraw func(*ebiten.Image, float32, float32, float32, color.Color),
	onClick func(),
) {
	btn := &ecs.ButtonEntity{
		Pos:        ecs.Position{X: float64(x), Y: float64(y)},
		Size:       ecs.Size{W: float64(w), H: float64(h)},
		Color:      clr,
		HoverColor: hoverClr,
		TextColor:  textClr,
		Label:      label,
		Face:       face,
		OnClick:    onClick,
	}

	if iconDraw != nil {
		btn.IconDraw = iconDraw
	}

	id := s.nextID()

	if err := s.Registry.Add(group, id, btn); err != nil {
		return
	}

	zone := &systems.Zone{
		Spatial: box(float64(x), float64(y), float64(w), float64(h)),
		OnClick: onClick,
		OnHover: func(hovered bool) { btn.Hovered = hovered },
	}

	s.input.AddZone(zone)
}

func (s *Scene) accentForState(st timer.State) color.Color {
	switch st {
	case timer.StateFocus:
		return ui.ColorAccentFocus
	case timer.StateBreak:
		return ui.ColorAccentBreak
	case timer.StateLongBreak:
		return ui.ColorGradBreakEnd
	default:
		return ui.ColorTextSecond
	}
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
	st := config.LoadState()

	state, prePause, pendingNext, round, remainingSec := s.tmr.Snapshot(time.Now())
	st.Round = round
	st.PendingNext = pendingNext
	st.State = state
	st.PrePause = prePause
	st.RemainingSec = remainingSec

	if err := config.SaveState(st); err != nil {
		slog.Warn("save state", "error", err)
	}
}

func box(x, y, w, h float64) shapes.Spatial { //nolint:ireturn // returns spatial for RTree
	return shapes.NewBox(shapes.NewPoint(x, y), w, h)
}
