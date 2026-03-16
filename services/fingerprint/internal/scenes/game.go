package scenes

import (
	"context"
	"fmt"
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

	state        GameState
	bootTick     int
	selectedCase int // 0-2, which case is selected in app-layout
	cases        []*domain.CaseConfig

	// Puzzle state
	holdingPiece int // index in tray (-1 = none)

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

	// Generate cases from DB
	s.cases = domain.GenerateCases(s.db, 99)
	s.selectedCase = 0
	s.holdingPiece = -1

	s.loadGameState()

	slog.Info("cases generated", "count", len(s.cases))

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

		// Camera zoom: Z = zoom in, X = zoom out, C = reset
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			s.Camera.ZoomFactor += 5
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyX) {
			s.Camera.ZoomFactor -= 5
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyC) {
			s.Camera.Reset()
		}

		// Mouse wheel: rotate held piece
		_, wy := ebiten.Wheel()
		if wy != 0 && s.holdingPiece >= 0 && s.selectedCase >= 0 {
			c := s.cases[s.selectedCase]
			if s.holdingPiece < len(c.TrayPieces) {
				c.TrayPieces[s.holdingPiece].Rotation = (c.TrayPieces[s.holdingPiece].Rotation + 1) % 4
			}
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
		s.drawImageLayer(screen, "enabled")
		s.drawImageLayer(screen, "application-layout")
		s.drawTileLayer(screen, "application-layout")
		s.drawAppContent(screen)

	case StateApplicationNet:
		s.drawImageLayer(screen, "enabled")
		s.drawImageLayer(screen, "application-net-layout")
		s.drawTileLayer(screen, "application-net-layout")
		s.drawPuzzleContent(screen)
	}

	// Held piece follows cursor
	if s.holdingPiece >= 0 && s.state == StateApplicationNet && s.selectedCase >= 0 {
		c := s.cases[s.selectedCase]
		if s.holdingPiece < len(c.TrayPieces) {
			tp := c.TrayPieces[s.holdingPiece]
			mx, my := ebiten.CursorPosition()

			clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x90}
			if tp.IsDecoy {
				clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0x90}
			}

			sz := 30.0
			ui.DrawRoundedRect(screen, float32(float64(mx)-sz/2), float32(float64(my)-sz/2),
				float32(sz), float32(sz), 3, clr)

			face := ui.Face(false, 7)
			ui.DrawText(screen, fmt.Sprintf("R%d", tp.Rotation), face,
				float64(mx)-8, float64(my)-4, color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
		}
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

var enabledLogOnce bool //nolint:gochecknoglobals // debug

func (s *GameScene) drawEnabled(screen *ebiten.Image) {
	s.drawImageLayer(screen, "enabled")
	s.drawTileLayer(screen, "enabled")

	if !enabledLogOnce {
		enabledLogOnce = true

		layer := s.tmap.FindTileLayer("enabled")
		if layer == nil {
			slog.Warn("enabled tile layer not found")
		} else {
			nonZero := 0

			for _, t := range layer.Tiles {
				if !t.IsNil() {
					nonZero++
				}
			}

			slog.Info("enabled tile layer", "tiles", len(layer.Tiles), "nonZero", nonZero)
		}
	}
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

// --- Step 4: Application layout dynamic content ---

func (s *GameScene) drawAppContent(screen *ebiten.Image) {
	og := s.tmap.FindObjectGroup("application-layout")
	if og == nil {
		return
	}

	faceLabel := ui.Face(true, 11)
	faceSmall := ui.Face(false, 9)

	// Draw case names in list-of-cases room
	if casesObj := tilemap.FindObject(og, "list-of-cases"); casesObj != nil {
		x := casesObj.X * s.scaleX
		y := casesObj.Y * s.scaleY
		w := casesObj.Width * s.scaleX
		rowH := casesObj.Height * s.scaleY / float64(len(s.cases))

		for i, c := range s.cases {
			ry := y + float64(i)*rowH

			// Highlight selected case
			if i == s.selectedCase {
				ui.DrawRoundedRect(screen, float32(x+2), float32(ry+2), float32(w-4), float32(rowH-4), 2,
					color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x60})
			}

			ui.DrawText(screen, c.Name, faceLabel, x+8, ry+rowH*0.3,
				color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})
		}
	}

	// Draw fingerprint user names
	if namesObj := tilemap.FindObject(og, "fingerprints-user-names"); namesObj != nil {
		x := namesObj.X * s.scaleX
		y := namesObj.Y * s.scaleY
		rowH := namesObj.Height * s.scaleY / float64(len(s.cases))

		for i, c := range s.cases {
			ry := y + float64(i)*rowH
			name := "Unknown ?"

			if c.TargetRecord != nil {
				// Show person name only if case is solved (for now always show hash)
				name = c.TargetRecord.Hash
			}

			clr := color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF}
			if i == s.selectedCase {
				clr = color.RGBA{R: 0x00, G: 0x80, B: 0x80, A: 0xFF}
			}

			ui.DrawText(screen, name, faceSmall, x+4, ry+rowH*0.3, clr)
		}
	}

	// Draw avatar for selected case
	if avatarObj := tilemap.FindObject(og, "avatar"); avatarObj != nil {
		ax := avatarObj.X * s.scaleX
		ay := avatarObj.Y * s.scaleY
		aw := avatarObj.Width * s.scaleX
		ah := avatarObj.Height * s.scaleY

		if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
			c := s.cases[s.selectedCase]
			avatarSrc := fmt.Sprintf("avatars/%s.jpg", c.TargetRecord.AvatarKey)
			avatarImg := s.tmap.GetImage(avatarSrc)

			if avatarImg != nil {
				op := &ebiten.DrawImageOptions{}
				iw := float64(avatarImg.Bounds().Dx())
				ih := float64(avatarImg.Bounds().Dy())
				op.GeoM.Scale(aw/iw, ah/ih)
				op.GeoM.Translate(ax, ay)
				screen.DrawImage(avatarImg, op)
			} else {
				// Placeholder
				ui.DrawRoundedRect(screen, float32(ax), float32(ay), float32(aw), float32(ah), 4,
					color.RGBA{R: 0xD5, G: 0xF2, B: 0xF1, A: 0xFF})

				ui.DrawTextCentered(screen, "?", ui.Face(true, 24), ax+aw/2, ay+ah/2-10,
					color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})
			}
		}
	}

	// Draw description for selected case
	if descObj := tilemap.FindObject(og, "description"); descObj != nil {
		dx := descObj.X * s.scaleX
		dy := descObj.Y * s.scaleY

		if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
			c := s.cases[s.selectedCase]
			rec := c.TargetRecord

			lines := []string{
				fmt.Sprintf("Case: %s", c.Name),
				fmt.Sprintf("Difficulty: %d pieces", c.PiecesToSolve),
				fmt.Sprintf("Color: %s", rec.Color),
				fmt.Sprintf("Variant: %d", rec.Variant),
			}

			if c.HideColor {
				lines = append(lines, "Color: HIDDEN (grey)")
			}

			for i, line := range lines {
				ui.DrawText(screen, line, faceSmall, dx+4, dy+float64(i)*16,
					color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})
			}
		}
	}
}

// --- Step 5: Puzzle workspace content ---

func (s *GameScene) drawPuzzleContent(screen *ebiten.Image) {
	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return
	}

	faceHash := ui.Face(false, 10)
	faceSmall := ui.Face(false, 8)

	// Draw hash
	if hashObj := tilemap.FindObject(og, "hash"); hashObj != nil {
		hx := hashObj.X * s.scaleX
		hy := hashObj.Y * s.scaleY

		if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
			c := s.cases[s.selectedCase]
			hashText := c.TargetRecord.Hash

			ui.DrawText(screen, hashText, faceHash, hx+4, hy+4,
				color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})
		}
	}

	// Draw puzzle grid area
	if puzzleObj := tilemap.FindObject(og, "puzzle"); puzzleObj != nil {
		px := puzzleObj.X * s.scaleX
		py := puzzleObj.Y * s.scaleY
		pw := puzzleObj.Width * s.scaleX
		ph := puzzleObj.Height * s.scaleY
		cellW := pw / 10
		cellH := ph / 10

		// Draw grid lines
		for i := range 11 {
			x := float32(px + float64(i)*cellW)
			ui.DrawRoundedRect(screen, x, float32(py), 1, float32(ph), 0,
				color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x40})

			y := float32(py + float64(i)*cellH)
			ui.DrawRoundedRect(screen, float32(px), y, float32(pw), 1, 0,
				color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x40})
		}

		// Draw piece indices (placeholder — real images come from fingerprint cutting)
		if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
			c := s.cases[s.selectedCase]
			missingSet := make(map[int]bool)

			for _, idx := range c.MissingIndices {
				missingSet[idx] = true
			}

			for idx := range 100 {
				if missingSet[idx] {
					continue // missing piece — empty slot
				}

				col := idx % 10
				row := idx / 10
				cx := px + float64(col)*cellW + cellW*0.3
				cy := py + float64(row)*cellH + cellH*0.3

				ui.DrawText(screen, fmt.Sprintf("%d", idx+1), faceSmall, cx, cy,
					color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x80})
			}

			// Track which grid cells have been filled by placed pieces
			placedAt := make(map[int]int) // gridIdx → trayIdx

			for ti, tp := range c.TrayPieces {
				if tp.IsPlaced {
					gIdx := tp.PlacedY*10 + tp.PlacedX
					placedAt[gIdx] = ti
				}
			}

			// Draw missing slots
			for _, idx := range c.MissingIndices {
				col := idx % 10
				row := idx / 10
				cx := px + float64(col)*cellW
				cy := py + float64(row)*cellH

				if ti, ok := placedAt[idx]; ok {
					// Piece placed here
					tp := c.TrayPieces[ti]
					clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0xCC}

					if tp.IsDecoy {
						clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0xCC}
					}

					ui.DrawRoundedRect(screen, float32(cx+1), float32(cy+1),
						float32(cellW-2), float32(cellH-2), 1, clr)
				} else {
					// Empty missing slot
					ui.DrawRoundedRect(screen, float32(cx+1), float32(cy+1),
						float32(cellW-2), float32(cellH-2), 1,
						color.RGBA{R: 0xFF, G: 0xA0, B: 0x00, A: 0x30})
				}
			}
		}
	}

	// Draw pieces tray
	if piecesObj := tilemap.FindObject(og, "pieces"); piecesObj != nil {
		px := piecesObj.X * s.scaleX
		py := piecesObj.Y * s.scaleY
		pw := piecesObj.Width * s.scaleX

		if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
			c := s.cases[s.selectedCase]
			pieceSize := pw / 3 // 3 columns in tray

			for i, tp := range c.TrayPieces {
				if tp.IsPlaced {
					continue
				}

				col := i % 3
				row := i / 3
				tx := px + float64(col)*pieceSize
				ty := py + float64(row)*pieceSize

				// Color based on decoy or correct
				clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0xCC}
				if tp.IsDecoy {
					clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0xCC}
				}

				ui.DrawRoundedRect(screen, float32(tx+1), float32(ty+1),
					float32(pieceSize-2), float32(pieceSize-2), 2, clr)

				// Show piece index
				ui.DrawText(screen, fmt.Sprintf("R%d", tp.Rotation), faceSmall,
					tx+4, ty+4, color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
			}
		}
	}
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
					slog.Info("opening puzzle workspace", "case", s.selectedCase)

					s.state = StateApplicationNet
					s.holdingPiece = -1
					s.registerPuzzleZones()
				},
			})
		case "list-of-cases":
			// Create clickable zones for each case within the room
			rx := obj.X * s.scaleX
			ry := obj.Y * s.scaleY
			rw := obj.Width * s.scaleX
			rowH := obj.Height * s.scaleY / float64(len(s.cases))

			for i := range s.cases {
				idx := i
				cy := ry + float64(i)*rowH

				s.input.AddZone(&systems.Zone{
					Spatial: shapes.NewBox(shapes.NewPoint(rx, cy), rw, rowH),
					OnClick: func() {
						s.selectedCase = idx
					},
				})
			}
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
		case "pieces":
			// Clickable piece tray — pick up piece
			if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
				c := s.cases[s.selectedCase]
				px := obj.X * s.scaleX
				py := obj.Y * s.scaleY
				pw := obj.Width * s.scaleX
				pieceSize := pw / 3

				for i := range c.TrayPieces {
					if c.TrayPieces[i].IsPlaced {
						continue
					}

					idx := i
					col := i % 3
					row := i / 3
					tx := px + float64(col)*pieceSize
					ty := py + float64(row)*pieceSize

					s.input.AddZone(&systems.Zone{
						Spatial: shapes.NewBox(shapes.NewPoint(tx, ty), pieceSize, pieceSize),
						OnClick: func() {
							if s.holdingPiece == idx {
								s.holdingPiece = -1 // drop
							} else {
								s.holdingPiece = idx // pick up
								slog.Info("picked up piece", "index", idx)
							}
						},
					})
				}
			}
		case "puzzle":
			// Click grid cell — place held piece
			if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
				c := s.cases[s.selectedCase]
				px := obj.X * s.scaleX
				py := obj.Y * s.scaleY
				pw := obj.Width * s.scaleX
				ph := obj.Height * s.scaleY
				cellW := pw / 10
				cellH := ph / 10

				missingSet := make(map[int]bool)

				for _, idx := range c.MissingIndices {
					missingSet[idx] = true
				}

				for gridIdx := range 100 {
					if !missingSet[gridIdx] {
						continue // only missing slots are clickable
					}

					gIdx := gridIdx
					col := gridIdx % 10
					row := gridIdx / 10
					cx := px + float64(col)*cellW
					cy := py + float64(row)*cellH

					s.input.AddZone(&systems.Zone{
						Spatial: shapes.NewBox(shapes.NewPoint(cx, cy), cellW, cellH),
						OnClick: func() {
							if s.holdingPiece < 0 || s.holdingPiece >= len(c.TrayPieces) {
								return
							}

							tp := &c.TrayPieces[s.holdingPiece]
							tp.IsPlaced = true
							tp.PlacedX = gIdx % 10
							tp.PlacedY = gIdx / 10

							slog.Info("placed piece", "tray", s.holdingPiece,
								"grid", gIdx, "x", tp.PlacedX, "y", tp.PlacedY)

							s.holdingPiece = -1

							// Save state after each placement
							s.saveGameState()
						},
					})
				}
			}
		}
	}
}

func (s *GameScene) saveGameState() {
	save := &domain.GameSave{}

	for i, c := range s.cases {
		cs := domain.CaseSave{CaseIndex: i}

		for j, tp := range c.TrayPieces {
			if tp.IsPlaced {
				cs.PlacedPieces = append(cs.PlacedPieces, domain.PlacedSave{
					TrayIndex: j,
					GridX:     tp.PlacedX,
					GridY:     tp.PlacedY,
					Rotation:  tp.Rotation,
				})
			}
		}

		save.Cases = append(save.Cases, cs)
	}

	if err := domain.SaveGame(save, domain.DefaultSavePath()); err != nil {
		slog.Warn("save game", "error", err)
	}
}

func (s *GameScene) loadGameState() {
	save, err := domain.LoadGame(domain.DefaultSavePath())
	if err != nil {
		return // no save file = fresh game
	}

	for _, cs := range save.Cases {
		if cs.CaseIndex < 0 || cs.CaseIndex >= len(s.cases) {
			continue
		}

		c := s.cases[cs.CaseIndex]

		for _, pp := range cs.PlacedPieces {
			if pp.TrayIndex < 0 || pp.TrayIndex >= len(c.TrayPieces) {
				continue
			}

			tp := &c.TrayPieces[pp.TrayIndex]
			tp.IsPlaced = true
			tp.PlacedX = pp.GridX
			tp.PlacedY = pp.GridY
			tp.Rotation = pp.Rotation
		}

		c.ID = cs.CaseIndex + 1
	}

	slog.Info("game state loaded")
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
