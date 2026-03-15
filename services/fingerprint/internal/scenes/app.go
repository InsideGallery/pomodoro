package scenes

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"math"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
	"github.com/InsideGallery/pomodoro/pkg/resources"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/systems"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const AppSceneName = "fingerprint_app"

// CaseEntity represents a case in the left column.
type CaseEntity struct {
	Name    string
	CaseRef *domain.Case
	Solved  bool
	Y       float64
	Hovered bool
}

// AppScene is the main 3-column forensic application.
type AppScene struct {
	*scene.BaseScene

	input       *systems.InputSystem
	switchScene func(string)

	db       *domain.Database
	gen      *domain.PuzzleGenerator
	cases    []*domain.Case
	selected int // index of selected case, -1 = none

	// Solved fingerprints stored for verification
	solvedPrints []*domain.Fingerprint

	cursor *ebiten.Image

	width, height int
	entityIDSeq   uint64
}

func NewAppScene(switchScene func(string)) *AppScene {
	return &AppScene{
		switchScene: switchScene,
		selected:    -1,
	}
}

func (s *AppScene) Name() string { return AppSceneName }

func (s *AppScene) SetResources(rm *resources.Manager) {
	if s.BaseScene != nil {
		s.BaseScene.Resources = rm
	}
}

func (s *AppScene) nextID() uint64 {
	s.entityIDSeq++

	return s.entityIDSeq
}

func (s *AppScene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
	s.input = systems.NewInputSystem(s.RTree)
}

func (s *AppScene) Load() error {
	if mon := ebiten.Monitor(); mon != nil {
		mw, mh := mon.Size()
		scale := mon.DeviceScaleFactor()
		s.width = int(float64(mw) * scale)
		s.height = int(float64(mh) * scale)
	}

	if img, ok := s.Resources.GetImage("cursor"); ok {
		s.cursor = img
	}

	s.selected = -1
	s.generateCases()
	s.createEntities()

	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	return nil
}

func (s *AppScene) Unload() error {
	ebiten.SetCursorMode(ebiten.CursorModeVisible)

	return nil
}

func (s *AppScene) generateCases() {
	s.db = domain.NewDatabase()
	s.gen = domain.NewPuzzleGenerator(42)
	s.cases = nil
	s.solvedPrints = nil

	// Generate suspects and cases
	names := []struct{ name, avatar, location string }{
		{"O'Connel, Thomas", "avatar_1", "MOTEL"},
		{"Moretti, Isabella", "avatar_2", "CAR WASH"},
		{"Blackwood, James", "avatar_3", "EDEN"},
		{"McQueen, Sarah", "avatar_4", "HOUSE MCQUEEN"},
	}

	for i, n := range names {
		_, solved := s.gen.GeneratePerson(s.db, n.name, n.avatar, 10, 10)
		s.solvedPrints = append(s.solvedPrints, solved)

		result := s.gen.GeneratePuzzle(solved, 30)

		c := domain.NewCase(i+1, result.Puzzle)
		s.cases = append(s.cases, c)
	}
}

func (s *AppScene) createEntities() {
	s.input.ClearZones()

	for _, key := range s.Registry.GetKeys() {
		s.Registry.TruncateGroup(key)
	}

	w := float32(s.width)
	h := float32(s.height)

	// Screen area inside CRT (approximate percentages from the background image)
	screenX := w * 0.22
	screenY := h * 0.10
	screenW := w * 0.56
	screenH := h * 0.76

	// Title bar
	titleH := screenH * 0.06
	contentY := screenY + titleH
	contentH := screenH - titleH

	// 3-column layout
	col1W := screenW * 0.30
	col1X := screenX

	// Exit button (top-right of title bar)
	exitW := float64(40)
	exitH := float64(titleH * 0.8)
	exitX := float64(screenX+screenW) - exitW - 5
	exitY := float64(screenY) + 3

	s.input.AddZone(&systems.Zone{
		Spatial: shapes.NewBox(shapes.NewPoint(exitX, exitY), exitW, exitH),
		OnClick: func() { s.switchScene(DesktopSceneName) },
	})

	// Case list buttons (left column)
	locations := []string{"MOTEL", "CAR WASH", "EDEN", "HOUSE MCQUEEN"}
	caseRowH := contentH * 0.08
	caseStartY := contentY + contentH*0.05

	for i, loc := range locations {
		idx := i
		cy := float64(caseStartY + float32(i)*caseRowH)

		ce := &CaseEntity{
			Name: loc,
			Y:    cy,
		}

		if idx < len(s.cases) {
			ce.CaseRef = s.cases[idx]
			ce.Solved = s.cases[idx].IsSolved()
		}

		if err := s.Registry.Add("case", s.nextID(), ce); err != nil {
			slog.Warn("registry add", "group", "case", "error", err)
		}

		s.input.AddZone(&systems.Zone{
			Spatial: shapes.NewBox(shapes.NewPoint(float64(col1X)+5, cy), float64(col1W)-10, float64(caseRowH)-4),
			OnHover: func(h bool) { ce.Hovered = h },
			OnClick: func() { s.selected = idx },
		})
	}
}

func (s *AppScene) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switchScene(DesktopSceneName)

		return nil
	}

	if err := s.input.Update(s.Ctx); err != nil {
		return err
	}

	return nil
}

var appLogOnce bool //nolint:gochecknoglobals // debug

func (s *AppScene) Draw(screen *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)

	if !appLogOnce {
		appLogOnce = true

		hasBG, _ := s.Resources.GetImage("bg_static")
		hasWin, _ := s.Resources.GetImage("app_window")

		slog.Info("app draw",
			"screen", fmt.Sprintf("%dx%d", screen.Bounds().Dx(), screen.Bounds().Dy()),
			"width", s.width, "height", s.height,
			"bg_static", hasBG != nil,
			"app_window", hasWin != nil,
			"cases", len(s.cases),
			"selected", s.selected,
		)
	}

	// CRT background (fit, preserve aspect ratio)
	screen.Fill(color.RGBA{A: 0xFF})

	if bg, ok := s.Resources.GetImage("bg_bright"); ok {
		drawFit(screen, bg, float64(w), float64(h), 1.0)
	} else if bg2, ok2 := s.Resources.GetImage("bg_static"); ok2 {
		drawFit(screen, bg2, float64(w), float64(h), 1.0)
	}

	// CRT screen area (same constants as desktop)
	bgW, bgH := 8328.0, 4320.0
	scaleX := float64(w) / bgW
	scaleY := float64(h) / bgH
	bgScale := scaleX

	if scaleY < scaleX {
		bgScale = scaleY
	}

	scaledW := bgW * bgScale
	scaledH := bgH * bgScale
	bgOffX := (float64(w) - scaledW) / 2
	bgOffY := (float64(h) - scaledH) / 2

	screenX := float32(bgOffX + CRTLeft*scaledW)
	screenY := float32(bgOffY + CRTTop*scaledH)
	screenW := float32((CRTRight - CRTLeft) * scaledW)
	screenH := float32((CRTBottom - CRTTop) * scaledH)
	titleH := screenH * 0.06

	// App window background
	if appWin, ok := s.Resources.GetImage("app_window"); ok {
		op := &ebiten.DrawImageOptions{}
		aw := float64(appWin.Bounds().Dx())
		ah := float64(appWin.Bounds().Dy())
		op.GeoM.Scale(float64(screenW)/aw, float64(screenH)/ah)
		op.GeoM.Translate(float64(screenX), float64(screenY))
		screen.DrawImage(appWin, op)
	} else {
		// Fallback: draw manually
		ui.DrawRoundedRect(screen, screenX, screenY, screenW, screenH, 4, color.RGBA{R: 0xD5, G: 0xF2, B: 0xF1, A: 0xFF})

		// Title bar
		ui.DrawRoundedRect(screen, screenX, screenY, screenW, titleH, 0, color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0xFF})
	}

	// Title text
	faceTitle := ui.Face(true, 14)
	ui.DrawTextCentered(screen, "FINGERPRINTING", faceTitle,
		float64(screenX+screenW/2), float64(screenY+titleH*0.25),
		color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})

	// Exit text
	faceSmall := ui.Face(false, 10)
	ui.DrawText(screen, "exit", faceSmall,
		float64(screenX+screenW-40), float64(screenY+titleH*0.3),
		color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})

	contentY := screenY + titleH

	// Column layout
	col1W := screenW * 0.30
	col2W := screenW * 0.30
	col1X := screenX
	col2X := screenX + col1W
	col3X := screenX + col1W + col2W
	col3W := screenW * 0.40

	// Column dividers
	ui.DrawRoundedRect(screen, col2X, contentY, 1, screenH-titleH, 0,
		color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x80})
	ui.DrawRoundedRect(screen, col3X, contentY, 1, screenH-titleH, 0,
		color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x80})

	// Draw cases (left column)
	faceCase := ui.Face(true, 12)

	for ce := range s.Registry.Iterator("case") {
		c, ok := ce.(*CaseEntity)
		if !ok {
			continue
		}

		rowClr := color.RGBA{R: 0xD5, G: 0xF2, B: 0xF1, A: 0x40}
		if c.Hovered {
			rowClr = color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x60}
		}

		if c.Solved {
			rowClr = color.RGBA{R: 0x00, G: 0xCE, B: 0xC9, A: 0x40}
		}

		ui.DrawRoundedRect(screen, col1X+4, float32(c.Y), col1W-8, screenH*0.07, 3, rowClr)
		ui.DrawText(screen, c.Name, faceCase, float64(col1X+12), c.Y+float64(screenH*0.02),
			color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})
	}

	// Middle column: fingerprint code(s) for selected case
	if s.selected >= 0 && s.selected < len(s.cases) {
		c := s.cases[s.selected]
		faceCode := ui.Face(false, 11)

		codeText := c.Puzzle.UniqueID()
		ui.DrawText(screen, codeText, faceCode,
			float64(col2X+8), float64(contentY+screenH*0.06),
			color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})

		// Status
		statusText := "UNSOLVED"
		statusClr := color.RGBA{R: 0xFF, G: 0x6B, B: 0x6B, A: 0xFF}

		if c.IsSolved() {
			statusText = "SOLVED"
			statusClr = color.RGBA{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF}
		}

		ui.DrawText(screen, statusText, faceCase,
			float64(col2X+8), float64(contentY+screenH*0.12),
			statusClr)
	}

	// Right column: suspect info
	if s.selected >= 0 && s.selected < len(s.cases) {
		// Avatar
		avatarKey := fmt.Sprintf("avatar_%d", s.selected+1)
		avatarSize := float64(col3W) * 0.5
		avatarX := float64(col3X) + float64(col3W)*0.25
		avatarY := float64(contentY) + 10

		if avatar, ok := s.Resources.GetImage(avatarKey); ok {
			op := &ebiten.DrawImageOptions{}
			aw := float64(avatar.Bounds().Dx())
			scale := avatarSize / aw
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(avatarX, avatarY)
			screen.DrawImage(avatar, op)
		} else {
			ui.DrawRoundedRect(screen, float32(avatarX), float32(avatarY),
				float32(avatarSize), float32(avatarSize), 4,
				color.RGBA{R: 0xD5, G: 0xF2, B: 0xF1, A: 0xFF})
			ui.DrawTextCentered(screen, "???", faceTitle,
				avatarX+avatarSize/2, avatarY+avatarSize/2-10,
				color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})
		}

		// Name
		nameY := avatarY + avatarSize + 10
		nameText := "???"

		c := s.cases[s.selected]
		if c.IsSolved() && c.MatchPerson != nil {
			nameText = fmt.Sprintf("Name: %s", c.MatchPerson.Name)
		}

		ui.DrawText(screen, nameText, faceCase,
			float64(col3X+8), nameY,
			color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})
	}

	// Custom cursor
	if s.cursor != nil {
		mx, my := ebiten.CursorPosition()
		op := &ebiten.DrawImageOptions{}
		cw := float64(s.cursor.Bounds().Dx())
		cursorScale := 32.0 / cw
		op.GeoM.Scale(cursorScale, cursorScale)
		op.GeoM.Translate(float64(mx), float64(my))
		screen.DrawImage(s.cursor, op)
	}
}

func (s *AppScene) Layout(outsideWidth, outsideHeight int) (int, int) {
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
