package fingerprint

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

	"github.com/InsideGallery/pomodoro/pkg/resources"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/systems"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const (
	PuzzleSceneName = "fingerprint_puzzle"
	candidateCount  = 6
	fpSize          = 128
)

// CandidateEntity is a fingerprint candidate in the puzzle grid.
type CandidateEntity struct {
	X, Y    float64
	W, H    float64
	Image   *ebiten.Image
	IsMatch bool
	Index   int
	Hovered bool
}

// PuzzleScene is the fingerprint matching puzzle.
type PuzzleScene struct {
	*scene.BaseScene

	input       *systems.InputSystem
	switchScene func(string)

	target     Fingerprint
	candidates []Fingerprint
	matchIdx   int

	score     int
	solved    bool
	wrong     bool
	wrongIdx  int
	startedAt time.Time
	breakDur  time.Duration

	width, height int
	entityIDSeq   uint64
}

func NewPuzzleScene(switchScene func(string), breakDur time.Duration) *PuzzleScene {
	return &PuzzleScene{
		switchScene: switchScene,
		breakDur:    breakDur,
	}
}

func (s *PuzzleScene) Name() string { return PuzzleSceneName }

func (s *PuzzleScene) SetResources(rm *resources.Manager) {
	if s.BaseScene != nil {
		s.BaseScene.Resources = rm
	}
}

func (s *PuzzleScene) nextID() uint64 {
	s.entityIDSeq++

	return s.entityIDSeq
}

func (s *PuzzleScene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
	s.input = systems.NewInputSystem(s.RTree)

	s.Systems.Add("input", s.input)
}

func (s *PuzzleScene) Load() error {
	if mon := ebiten.Monitor(); mon != nil {
		mw, mh := mon.Size()
		scale := mon.DeviceScaleFactor()
		s.width = int(float64(mw) * scale)
		s.height = int(float64(mh) * scale)
	}

	s.startedAt = time.Now()
	s.solved = false
	s.wrong = false
	s.score = 0

	s.newPuzzle()

	ebiten.SetFullscreen(true)

	return nil
}

func (s *PuzzleScene) Unload() error {
	ebiten.SetFullscreen(false)

	return nil
}

func (s *PuzzleScene) newPuzzle() {
	seed := uint64(time.Now().UnixNano())
	s.target = Generate(seed, fpSize)
	s.matchIdx, s.candidates = GenerateSet(seed, candidateCount, fpSize)
	s.solved = false
	s.wrong = false

	// Cache generated images as Ebiten images in Resources
	s.Resources.Set("fp_target", ebiten.NewImageFromImage(s.target.Image))

	for i, c := range s.candidates {
		key := fmt.Sprintf("fp_candidate_%d", i)
		s.Resources.Set(key, ebiten.NewImageFromImage(c.Image))
	}

	s.createEntities()
}

func (s *PuzzleScene) createEntities() {
	s.input.ClearZones()

	for _, key := range s.Registry.GetKeys() {
		s.Registry.TruncateGroup(key)
	}

	// Layout: target on left, candidates in 2x3 grid on right
	w := float32(s.width)
	h := float32(s.height)

	cardSize := float64(h) * 0.25
	gridCols := 3
	gridRows := 2
	gap := float64(20)
	gridW := float64(gridCols)*cardSize + float64(gridCols-1)*gap
	gridH := float64(gridRows)*cardSize + float64(gridRows-1)*gap
	gridX := float64(w)*0.55 - gridW/2
	gridY := float64(h)/2 - gridH/2

	for i := range candidateCount {
		col := i % gridCols
		row := i / gridCols

		cx := gridX + float64(col)*(cardSize+gap)
		cy := gridY + float64(row)*(cardSize+gap)

		ce := &CandidateEntity{
			X: cx, Y: cy, W: cardSize, H: cardSize,
			IsMatch: i == s.matchIdx,
			Index:   i,
		}

		if img, ok := s.Resources.GetImage(fmt.Sprintf("fp_candidate_%d", i)); ok {
			ce.Image = img
		}

		id := s.nextID()
		if err := s.Registry.Add("candidate", id, ce); err != nil {
			slog.Warn("registry add", "group", "candidate", "error", err)

			continue
		}

		idx := i

		s.input.AddZone(&systems.Zone{
			Spatial: shapes.NewBox(shapes.NewPoint(cx, cy), cardSize, cardSize),
			OnHover: func(hovered bool) { ce.Hovered = hovered },
			OnClick: func() {
				if s.solved {
					return
				}

				if idx == s.matchIdx {
					s.solved = true
					s.score++
				} else {
					s.wrong = true
					s.wrongIdx = idx
				}
			},
		})
	}
}

func (s *PuzzleScene) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switchScene("timer")

		return nil
	}

	// Camera zoom with scroll wheel
	_, wy := ebiten.Wheel()
	if wy != 0 {
		s.Camera.ZoomFactor += wy * 3
	}

	// After solving, click/space goes to next puzzle
	if s.solved {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
			inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			s.newPuzzle()
		}

		return nil
	}

	// Clear wrong feedback after a moment
	if s.wrong {
		s.wrong = false
	}

	if err := s.input.Update(s.Ctx); err != nil {
		return err
	}

	return nil
}

func (s *PuzzleScene) Draw(screen *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)

	// Dark background
	ui.DrawRoundedRect(screen, 0, 0, w, h, 0, color.RGBA{R: 0x12, G: 0x12, B: 0x1A, A: 0xF8})

	// Title
	faceTitle := ui.Face(true, 20)
	ui.DrawTextCentered(screen, "Match the Fingerprint", faceTitle,
		float64(w/2), ui.Sf(20), ui.ColorTextPrimary)

	// Timer
	faceSmall := ui.Face(false, 12)
	rem := s.breakDur - time.Since(s.startedAt)

	if rem < 0 {
		rem = 0
	}

	totalSecs := int(rem.Seconds())
	timeText := fmt.Sprintf("Time: %02d:%02d  Score: %d", totalSecs/60, totalSecs%60, s.score)
	ui.DrawText(screen, timeText, faceSmall, ui.Sf(20), ui.Sf(20), ui.ColorTextSecond)

	// ESC hint
	ui.DrawText(screen, "ESC to exit  |  Scroll to zoom", faceSmall,
		float64(w)-ui.Sf(220), ui.Sf(20), ui.ColorTextSecond)

	// Target fingerprint (left side)
	targetSize := float64(h) * 0.35
	targetX := float64(w)*0.2 - targetSize/2
	targetY := float64(h)/2 - targetSize/2

	ui.DrawRoundedRect(screen, float32(targetX-4), float32(targetY-4),
		float32(targetSize+8), float32(targetSize+8), ui.S(8),
		color.RGBA{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF})

	if img, ok := s.Resources.GetImage("fp_target"); ok {
		op := &ebiten.DrawImageOptions{}
		scaleX := targetSize / float64(fpSize)
		scaleY := targetSize / float64(fpSize)
		op.GeoM.Scale(scaleX, scaleY)
		op.GeoM.Translate(targetX, targetY)
		screen.DrawImage(img, op)
	}

	faceLabel := ui.Face(true, 14)
	ui.DrawTextCentered(screen, "Target", faceLabel,
		float64(w)*0.2, targetY-ui.Sf(16), ui.ColorAccentFocus)

	// Draw candidates from Registry
	for ce := range s.Registry.Iterator("candidate") {
		c, ok := ce.(*CandidateEntity)
		if !ok {
			continue
		}

		// Border color based on state
		borderClr := color.RGBA{R: 0x30, G: 0x30, B: 0x40, A: 0xFF}

		switch {
		case s.solved && c.IsMatch:
			borderClr = color.RGBA{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF} // teal = correct
		case s.wrong && c.Index == s.wrongIdx:
			borderClr = color.RGBA{R: 0xFF, G: 0x6B, B: 0x6B, A: 0xFF} // red = wrong
		case c.Hovered:
			borderClr = color.RGBA{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF} // purple = hover
		}

		ui.DrawRoundedRect(screen, float32(c.X-3), float32(c.Y-3),
			float32(c.W+6), float32(c.H+6), ui.S(6), borderClr)

		if c.Image != nil {
			op := &ebiten.DrawImageOptions{}
			scaleX := c.W / float64(fpSize)
			scaleY := c.H / float64(fpSize)
			op.GeoM.Scale(scaleX, scaleY)
			op.GeoM.Translate(c.X, c.Y)
			screen.DrawImage(c.Image, op)
		}
	}

	// Solved overlay
	if s.solved {
		faceBig := ui.Face(true, 28)
		ui.DrawTextCentered(screen, "Correct! Click for next puzzle", faceBig,
			float64(w/2), float64(h)-ui.Sf(50),
			color.RGBA{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF})
	}
}

func (s *PuzzleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
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
