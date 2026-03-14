package minigame

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"math"
	"time"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/systems"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const SceneName = "minigame"

// TargetEntity is a clickable target in the Registry.
type TargetEntity struct {
	X, Y   float64
	Radius float64
	Alive  bool
	Color  color.RGBA
}

type Scene struct {
	*scene.BaseScene

	input     *systems.InputSystem
	game      Game
	enabled   bool
	gameOver  bool
	bestScore int
	breakDur  time.Duration
	onSave    func(int)
	onDone    func()

	width, height int
	entityIDSeq   uint64
}

func NewScene(bus *event.Bus, switchToSelf func(), onDone func()) *Scene {
	cfg := config.Load()
	st := config.LoadState()

	s := &Scene{
		enabled:   cfg.PluginEnabled("minigame_enabled", false),
		bestScore: st.MinigameBestScore,
		breakDur:  cfg.BreakDuration(),
		onSave: func(best int) {
			loadedSt := config.LoadState()

			loadedSt.MinigameBestScore = best
			if err := config.SaveState(loadedSt); err != nil {
				slog.Warn("save state", "error", err)
			}
		},
		onDone: onDone,
	}

	bus.Subscribe(event.BreakStarted, func(_ event.Event) {
		if s.enabled {
			switchToSelf()
		}
	})

	bus.Subscribe(event.ConfigChanged, func(e event.Event) {
		if c, ok := e.Data.(config.Config); ok {
			s.enabled = c.PluginEnabled("minigame_enabled", false)
			s.breakDur = c.BreakDuration()
		}
	})

	return s
}

func (s *Scene) Name() string   { return SceneName }
func (s *Scene) IsActive() bool { return s.enabled }

func (s *Scene) nextID() uint64 {
	s.entityIDSeq++

	return s.entityIDSeq
}

func (s *Scene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
	s.input = systems.NewInputSystem(s.RTree)

	s.Systems.Add("input", s.input)
}

func (s *Scene) Load() error {
	s.gameOver = false

	if mon := ebiten.Monitor(); mon != nil {
		mw, mh := mon.Size()
		scale := mon.DeviceScaleFactor()
		s.width = int(float64(mw) * scale)
		s.height = int(float64(mh) * scale)
	}

	st := config.LoadState()
	s.bestScore = st.MinigameBestScore
	s.game.Start(s.width, s.height, s.bestScore, s.breakDur, time.Now())

	s.syncTargetsToRegistry()

	ebiten.SetFullscreen(true)

	return nil
}

func (s *Scene) Unload() error {
	ebiten.SetFullscreen(false)

	return nil
}

func (s *Scene) Update() error {
	if s.gameOver {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
			inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.onDone()
		}

		return nil
	}

	now := time.Now()

	if s.game.IsOver(now) {
		s.finish()

		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.saveIfRecord()
		s.onDone()

		return nil
	}

	// InputSystem handles click detection via RTree
	if err := s.input.Update(s.Ctx); err != nil {
		return err
	}

	// Check if we need to spawn more targets (alive count <= 1)
	alive := 0

	for _, t := range s.game.Targets {
		if t.Alive {
			alive++
		}
	}

	if alive <= 1 && len(s.game.Targets) > 0 {
		s.game.SpawnBatch()
		s.syncTargetsToRegistry()
	}

	return nil
}

func (s *Scene) finish() {
	s.gameOver = true
	s.saveIfRecord()
}

func (s *Scene) saveIfRecord() {
	if s.game.BeatRecord() && s.onSave != nil {
		s.bestScore = s.game.Score
		s.onSave(s.bestScore)
	}
}

// syncTargetsToRegistry creates entities and RTree zones from game targets.
func (s *Scene) syncTargetsToRegistry() {
	s.input.ClearZones()

	for _, key := range s.Registry.GetKeys() {
		s.Registry.TruncateGroup(key)
	}

	for i, t := range s.game.Targets {
		if !t.Alive {
			continue
		}

		clr := palette[i%len(palette)]
		te := &TargetEntity{X: t.X, Y: t.Y, Radius: t.Radius, Alive: true, Color: clr}
		idx := i

		id := s.nextID()
		if err := s.Registry.Add("target", id, te); err != nil {
			slog.Warn("registry add", "group", "target", "error", err)

			continue
		}

		// Smaller targets get higher priority (lower number = checked first)
		s.input.AddZone(&systems.Zone{
			Spatial:  shapes.NewSphere(shapes.NewPoint(t.X, t.Y), t.Radius),
			Priority: int(t.Radius), // smallest radius = highest priority
			OnClick: func() {
				s.game.Targets[idx].Alive = false
				te.Alive = false
				s.game.Score++
			},
		})
	}
}

func (s *Scene) Draw(screen *ebiten.Image) {
	if s.gameOver {
		s.drawGameOver(screen)

		return
	}

	// Draw targets from Registry
	for te := range s.Registry.Iterator("target") {
		t, ok := te.(*TargetEntity)
		if !ok || !t.Alive {
			continue
		}

		border := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xBB}
		vector.FillCircle(screen, float32(t.X), float32(t.Y), float32(t.Radius+2), border, true)
		vector.FillCircle(screen, float32(t.X), float32(t.Y), float32(t.Radius), t.Color, true)
	}

	s.drawHUD(screen)
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

var palette = []color.RGBA{
	{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF},
	{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF},
	{R: 0xFD, G: 0x79, B: 0x72, A: 0xFF},
	{R: 0xFD, G: 0xCB, B: 0x6E, A: 0xFF},
	{R: 0x55, G: 0xEF, B: 0xC4, A: 0xFF},
	{R: 0xA2, G: 0x9B, B: 0xFE, A: 0xFF},
	{R: 0xFF, G: 0x77, B: 0x75, A: 0xFF},
	{R: 0x74, G: 0xB9, B: 0xFF, A: 0xFF},
	{R: 0xFF, G: 0x92, B: 0x50, A: 0xFF},
	{R: 0x00, G: 0xD2, B: 0xD3, A: 0xFF},
}

func (s *Scene) drawHUD(dst *ebiten.Image) {
	now := time.Now()
	rem := s.game.Remaining(now)
	totalSecs := int(rem.Seconds())
	mins := totalSecs / 60
	secs := totalSecs % 60

	cardW := ui.S(160)
	cardH := ui.S(80)
	cardX := float32(s.width) - cardW - ui.S(16)
	cardY := ui.S(16)

	ui.DrawRoundedRect(dst, cardX, cardY, cardW, cardH, ui.S(8),
		color.RGBA{R: 0x18, G: 0x18, B: 0x22, A: 0xDD})

	face := ui.Face(true, 14)
	faceSmall := ui.Face(false, 11)

	scoreText := fmt.Sprintf("Score: %d", s.game.Score)
	ui.DrawText(dst, scoreText, face, float64(cardX+ui.S(12)), float64(cardY+ui.S(16)), ui.ColorTextPrimary)

	best := max(s.game.Score, s.game.BestScore)
	bestText := fmt.Sprintf("Best: %d", best)
	ui.DrawText(dst, bestText, faceSmall, float64(cardX+ui.S(12)), float64(cardY+ui.S(36)), ui.ColorTextSecond)

	timeText := fmt.Sprintf("%02d:%02d", mins, secs)
	ui.DrawText(dst, timeText, face, float64(cardX+ui.S(12)), float64(cardY+ui.S(56)), ui.ColorTextPrimary)

	escText := "ESC to close"
	ui.DrawText(dst, escText, faceSmall, float64(cardX+cardW-ui.S(90)), float64(cardY+ui.S(56)), ui.ColorTextSecond)
}

func (s *Scene) drawGameOver(dst *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)

	ui.DrawRoundedRect(dst, 0, 0, w, h, 0, color.RGBA{R: 0x10, G: 0x10, B: 0x18, A: 0xDD})

	cardW := ui.S(300)
	cardH := ui.S(200)
	cardX := (w - cardW) / 2
	cardY := (h - cardH) / 2

	ui.DrawRoundedRect(dst, cardX, cardY, cardW, cardH, ui.S(12),
		color.RGBA{R: 0x20, G: 0x20, B: 0x30, A: 0xFF})

	faceTitle := ui.Face(true, 24)
	faceScore := ui.Face(true, 18)
	faceHint := ui.Face(false, 12)

	if s.game.BeatRecord() {
		ui.DrawText(dst, "New Record!", faceTitle,
			float64(cardX+ui.S(60)), float64(cardY+ui.S(40)),
			color.RGBA{R: 0xFD, G: 0xCB, B: 0x6E, A: 0xFF})
	} else {
		ui.DrawText(dst, "Time's Up!", faceTitle,
			float64(cardX+ui.S(70)), float64(cardY+ui.S(40)),
			ui.ColorTextPrimary)
	}

	scoreText := fmt.Sprintf("Score: %d", s.game.Score)
	ui.DrawText(dst, scoreText, faceScore,
		float64(cardX+ui.S(100)), float64(cardY+ui.S(90)),
		ui.ColorTextPrimary)

	bestText := fmt.Sprintf("Best: %d", max(s.game.Score, s.game.BestScore))
	ui.DrawText(dst, bestText, faceScore,
		float64(cardX+ui.S(105)), float64(cardY+ui.S(120)),
		ui.ColorTextSecond)

	ui.DrawText(dst, "Click or ESC to continue", faceHint,
		float64(cardX+ui.S(65)), float64(cardY+ui.S(170)),
		ui.ColorTextSecond)
}
