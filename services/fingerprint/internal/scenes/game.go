package scenes

import (
	"context"
	"image/color"
	"log/slog"
	"math"
	"os"
	"path/filepath"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/lafriks/go-tiled"

	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/systems"
	"github.com/InsideGallery/pomodoro/pkg/tilemap"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const GameSceneName = "fingerprint_game"

// GameState represents the current layer visibility state.
type GameState int

const (
	StateDisabled          GameState = iota // PC off
	StateEnabled                            // Desktop with icon
	StateApplicationLayout                  // Case selection (3-column)
	StateApplicationNet                     // Puzzle workspace
)

// GameScene is the single TMX-driven scene for the fingerprint game.
// All UI is defined in fingerprint.tmx. State changes toggle layer visibility.
type GameScene struct {
	*scene.BaseScene

	input *systems.InputSystem
	tmap  *tilemap.Map
	db    *domain.FingerprintDB

	state    GameState
	bootTick int

	// Scaling: map is 4000×2176, screen may differ
	scaleX, scaleY float64

	width, height int
}

func NewGameScene() *GameScene {
	return &GameScene{}
}

func (s *GameScene) Name() string { return GameSceneName }

func (s *GameScene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
	s.input = systems.NewInputSystem(s.RTree)
}

func (s *GameScene) Load() error {
	if mon := ebiten.Monitor(); mon != nil {
		mw, mh := mon.Size()
		scale := mon.DeviceScaleFactor()
		s.width = int(float64(mw) * scale)
		s.height = int(float64(mh) * scale)
	}

	// Load TMX
	tmxPath := FindTMXPath()
	if tmxPath == "" {
		slog.Error("fingerprint.tmx not found")

		return nil
	}

	var err error

	s.tmap, err = tilemap.Load(tmxPath)
	if err != nil {
		slog.Error("load tmx", "error", err)

		return nil
	}

	slog.Info("tmx loaded",
		"size", s.tmap.MapPixelWidth(), "x", s.tmap.MapPixelHeight(),
		"layers", len(s.tmap.Layers),
		"imageLayers", len(s.tmap.ImageLayers),
		"objectGroups", len(s.tmap.ObjectGroups),
		"tilesets", len(s.tmap.Tilesets))

	// Load or generate fingerprint DB
	dbPath := domain.DefaultDBPath()

	var dbErr error

	s.db, dbErr = domain.LoadDB(dbPath)
	if dbErr != nil {
		slog.Info("generating fingerprint DB (first run)")

		s.db = domain.GenerateDB(42)
		if err := s.db.Save(dbPath); err != nil {
			slog.Warn("save db", "error", err)
		} else {
			slog.Info("fingerprint DB saved", "path", dbPath, "records", len(s.db.Records))
		}
	} else {
		slog.Info("fingerprint DB loaded", "records", len(s.db.Records))
	}

	// Compute scaling
	mapW := float64(s.tmap.MapPixelWidth())
	mapH := float64(s.tmap.MapPixelHeight())
	s.scaleX = float64(s.width) / mapW
	s.scaleY = float64(s.height) / mapH

	// Start in disabled state (PC off)
	s.state = StateDisabled
	s.bootTick = 0

	ebiten.SetFullscreen(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	return nil
}

func (s *GameScene) Unload() error {
	ebiten.SetCursorMode(ebiten.CursorModeVisible)

	return nil
}

func (s *GameScene) Update() error {
	if s.tmap == nil {
		return nil
	}

	switch s.state {
	case StateDisabled:
		s.bootTick++
		// After 90 frames (~1.5s), transition to enabled
		if s.bootTick > 90 {
			s.state = StateEnabled
			s.registerEnabledZones()
		}

	case StateEnabled:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
			os.Exit(0)
		}

		if err := s.input.Update(s.Ctx); err != nil {
			return err
		}

	case StateApplicationLayout:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.state = StateEnabled
			s.registerEnabledZones()
		}

		if err := s.input.Update(s.Ctx); err != nil {
			return err
		}

	case StateApplicationNet:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.state = StateApplicationLayout
			s.registerAppLayoutZones()
		}

		if err := s.input.Update(s.Ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *GameScene) Draw(screen *ebiten.Image) {
	if s.tmap == nil {
		screen.Fill(color.RGBA{A: 0xFF})

		return
	}

	screen.Fill(color.RGBA{A: 0xFF})

	switch s.state {
	case StateDisabled:
		s.drawDisabled(screen)

	case StateEnabled:
		s.drawEnabled(screen)

	case StateApplicationLayout:
		// Keep enabled background
		s.drawImageLayer(screen, "enabled")
		// Draw app layout on top
		s.drawImageLayer(screen, "application-layout")
		s.drawTileLayer(screen, "application-layout")

	case StateApplicationNet:
		// Keep enabled background
		s.drawImageLayer(screen, "enabled")
		// Draw puzzle workspace
		s.drawImageLayer(screen, "application-net-layout")
		s.drawTileLayer(screen, "application-net-layout")
	}

	// Custom cursor (always on top, after boot)
	if s.state >= StateEnabled {
		s.drawCursor(screen)
	}
}

func (s *GameScene) drawDisabled(screen *ebiten.Image) {
	// Cross-fade from disabled to enabled
	progress := float64(s.bootTick) / 90.0
	if progress > 1 {
		progress = 1
	}

	if progress < 0.5 {
		s.drawImageLayer(screen, "disabled")
	} else {
		fade := (progress - 0.5) * 2

		s.drawImageLayerAlpha(screen, "disabled", 1.0-fade)
		s.drawImageLayerAlpha(screen, "enabled", fade)
	}
}

func (s *GameScene) drawEnabled(screen *ebiten.Image) {
	s.drawImageLayer(screen, "enabled")
	s.drawTileLayer(screen, "enabled")
}

func (s *GameScene) drawImageLayer(screen *ebiten.Image, name string) {
	s.drawImageLayerAlpha(screen, name, 1.0)
}

func (s *GameScene) drawImageLayerAlpha(screen *ebiten.Image, name string, alpha float64) {
	layer := s.tmap.FindImageLayer(name)
	if layer == nil {
		return
	}

	img := s.tmap.ImageLayerImage(layer)
	if img == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(s.scaleX, s.scaleY)

	if alpha < 1.0 {
		op.ColorScale.Scale(float32(alpha), float32(alpha), float32(alpha), float32(alpha))
	}

	screen.DrawImage(img, op)
}

func (s *GameScene) drawTileLayer(screen *ebiten.Image, name string) {
	layer := s.tmap.FindTileLayer(name)
	if layer == nil {
		return
	}

	s.tmap.DrawTileLayer(screen, layer, s.scaleX, s.scaleY, 0, 0)
}

func (s *GameScene) drawCursor(screen *ebiten.Image) {
	// Find cursor image from buttons tileset (tile id=3)
	cursorImg := s.tmap.GetImage("ui/cursor.png")
	if cursorImg == nil {
		return
	}

	mx, my := ebiten.CursorPosition()
	op := &ebiten.DrawImageOptions{}

	cw := float64(cursorImg.Bounds().Dx())
	cursorScale := 32.0 / cw
	op.GeoM.Scale(cursorScale, cursorScale)
	op.GeoM.Translate(float64(mx), float64(my))
	screen.DrawImage(cursorImg, op)
}

func (s *GameScene) registerEnabledZones() {
	s.input.ClearZones()

	og := s.tmap.FindObjectGroup("enabled")
	if og == nil {
		slog.Warn("enabled object group not found")

		return
	}

	for _, obj := range og.Objects {
		spatial := tilemap.ObjectToSpatial(obj)

		// Scale spatial to screen coordinates
		scaledSpatial := scaleBox(obj, s.scaleX, s.scaleY)

		switch obj.Name {
		case "button-run-fingerprint":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					slog.Info("opening fingerprint app")

					s.state = StateApplicationLayout
					s.registerAppLayoutZones()
				},
			})
		case "button-quit-os":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					ebiten.SetCursorMode(ebiten.CursorModeVisible)
					os.Exit(0)
				},
			})
		default:
			slog.Debug("unhandled enabled object", "name", obj.Name, "spatial", spatial)
		}
	}
}

func (s *GameScene) registerAppLayoutZones() {
	s.input.ClearZones()

	og := s.tmap.FindObjectGroup("application-layout")
	if og == nil {
		return
	}

	for _, obj := range og.Objects {
		scaledSpatial := scaleBox(obj, s.scaleX, s.scaleY)

		switch obj.Name {
		case "exit":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					s.state = StateEnabled
					s.registerEnabledZones()
				},
			})
		case "play-puzzle":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					slog.Info("opening puzzle workspace")

					s.state = StateApplicationNet
					s.registerPuzzleZones()
				},
			})
		}
	}
}

func (s *GameScene) registerPuzzleZones() {
	s.input.ClearZones()

	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return
	}

	for _, obj := range og.Objects {
		scaledSpatial := scaleBox(obj, s.scaleX, s.scaleY)

		switch obj.Name {
		case "back":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					s.state = StateApplicationLayout
					s.registerAppLayoutZones()
				},
			})
		case "exit":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					s.state = StateEnabled
					s.registerEnabledZones()
				},
			})
		}
	}
}

func scaleBox(obj *tiled.Object, sx, sy float64) shapes.Spatial { //nolint:ireturn // spatial for RTree
	return shapes.NewBox(
		shapes.NewPoint(obj.X*sx, obj.Y*sy),
		obj.Width*sx, obj.Height*sy,
	)
}

func (s *GameScene) Layout(outsideWidth, outsideHeight int) (int, int) {
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

		if s.tmap != nil {
			s.scaleX = float64(w) / float64(s.tmap.MapPixelWidth())
			s.scaleY = float64(h) / float64(s.tmap.MapPixelHeight())
		}
	}

	return w, h
}

// FindTMXPath locates the fingerprint.tmx file.
func FindTMXPath() string {
	candidates := []string{
		"assets/external/fingerprint/fingerprint.tmx",
		"../assets/external/fingerprint/fingerprint.tmx",
		"../../assets/external/fingerprint/fingerprint.tmx",
	}

	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "assets", "external", "fingerprint", "fingerprint.tmx"),
			filepath.Join(dir, "..", "assets", "external", "fingerprint", "fingerprint.tmx"),
			filepath.Join(dir, "..", "..", "assets", "external", "fingerprint", "fingerprint.tmx"),
		)
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}
