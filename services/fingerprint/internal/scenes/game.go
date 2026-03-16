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
	selectedCase int // 0-2, which case is selected
	cases        []*domain.CaseConfig

	// Puzzle state
	holdingPiece int                           // index in tray (-1 = none)
	showResult   int                           // 0=none, 1=success, 2=fail
	resultTick   int                           // frames to show result
	caseImages   [3]*FingerprintImages         // cut fingerprint images per case
	greyImages   [3]*FingerprintImages         // grey versions (when color hidden)
	allImages    map[string]*FingerprintImages // "color.variant" → images (for decoys)
	assetsDir    string

	// Scaling: map is 4000×2176, screen may differ
	scaleX, scaleY float64

	width, height int
}

func NewGameScene() *GameScene {
	return &GameScene{}
}

// currentPuzzle returns the active puzzle, or nil.

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

	// Load ALL fingerprint images (for target + decoy pieces)
	s.assetsDir = FindFingerprintAssetsDir()
	s.allImages = make(map[string]*FingerprintImages)

	if s.assetsDir != "" {
		// Load all color/variant combinations
		colors := []string{"green", "red", "yellow", "blue"}

		for _, clr := range colors {
			for v := 1; v <= 4; v++ {
				rec := &domain.FingerprintRecord{Color: clr, Variant: v, Rotation: 0, Mirrored: false}

				key := fmt.Sprintf("%s.%d", clr, v)

				imgs, err := LoadFingerprintImages(s.assetsDir, rec)
				if err != nil {
					slog.Warn("load fingerprint", "key", key, "error", err)
				} else {
					s.allImages[key] = imgs
				}
			}
		}

		// Load grey variants
		for v := 1; v <= 4; v++ {
			key := fmt.Sprintf("grey.%d", v)
			imgs, err := LoadGreyFingerprintImages(s.assetsDir, v, 0, false)
			if err != nil {
				slog.Warn("load grey", "key", key, "error", err)
			} else {
				s.allImages[key] = imgs
			}
		}

		// Assign case images
		for i, c := range s.cases {
			key := fmt.Sprintf("%s.%d", c.Puzzles[0].TargetRecord.Color, c.Puzzles[0].TargetRecord.Variant)

			s.caseImages[i] = s.allImages[key]

			if c.Puzzles[0].HideColor {
				greyKey := fmt.Sprintf("grey.%d", c.Puzzles[0].TargetRecord.Variant)
				s.greyImages[i] = s.allImages[greyKey]
			}
		}

		slog.Info("fingerprint images loaded", "total", len(s.allImages))
	}

	slog.Info("cases generated", "count", len(s.cases), "assetsDir", s.assetsDir)

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
		// Result display timer
		if s.resultTick > 0 {
			s.resultTick--
			if s.resultTick == 0 {
				s.showResult = 0
			}
		}

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
			if s.holdingPiece < len(c.Puzzles[0].TrayPieces) {
				c.Puzzles[0].TrayPieces[s.holdingPiece].Rotation = (c.Puzzles[0].TrayPieces[s.holdingPiece].Rotation + 1) % 4
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

	// Result overlay (success/fail)
	if s.showResult > 0 && s.state == StateApplicationNet {
		if s.showResult == 1 {
			s.drawTileLayer(screen, "application-net-layout-success")
		} else {
			s.drawTileLayer(screen, "application-net-layout-fail")
		}
	}

	// Held piece follows cursor — draw the actual rotated piece image
	if s.holdingPiece >= 0 && s.state == StateApplicationNet && s.selectedCase >= 0 {
		c := s.cases[s.selectedCase]
		if s.holdingPiece < len(c.Puzzles[0].TrayPieces) {
			tp := c.Puzzles[0].TrayPieces[s.holdingPiece]
			mx, my := ebiten.CursorPosition()

			cImgs := s.caseImages[s.selectedCase]
			pieceImg := s.getPieceImage(tp, cImgs)
			sz := 40.0

			if pieceImg != nil {
				s.drawRotatedPiece(screen, pieceImg, float64(mx)-sz/2, float64(my)-sz/2, sz, tp.Rotation)
			} else {
				clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x90}
				if tp.IsDecoy {
					clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0x90}
				}

				ui.DrawRoundedRect(screen, float32(float64(mx)-sz/2), float32(float64(my)-sz/2),
					float32(sz), float32(sz), 3, clr)
			}
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

	// Draw Quit button (no design asset — draw programmatically)
	og := s.tmap.FindObjectGroup("enabled")
	if og != nil {
		if quitObj := tilemap.FindObject(og, "button-quit-os"); quitObj != nil {
			qx := quitObj.X * s.scaleX
			qy := quitObj.Y * s.scaleY

			face := ui.Face(true, 9)
			ui.DrawRoundedRect(screen, float32(qx), float32(qy), float32(60*s.scaleX), float32(24*s.scaleY), 3,
				color.RGBA{R: 0xCC, G: 0x33, B: 0x33, A: 0xCC})

			ui.DrawText(screen, "Quit", face, qx+8*s.scaleX, qy+6*s.scaleY,
				color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
		}
	}

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
	cursorImg := s.tmap.GetImage("ui/cursor.png")
	if cursorImg == nil {
		return
	}

	mx, my := ebiten.CursorPosition()

	// Clamp cursor to cursor-room (from TMX main objectgroup)
	mainOG := s.tmap.FindObjectGroup("main")
	if mainOG != nil {
		if room := tilemap.FindObject(mainOG, "cursor-room"); room != nil {
			minX := int(room.X * s.scaleX)
			minY := int(room.Y * s.scaleY)
			maxX := int((room.X + room.Width) * s.scaleX)
			maxY := int((room.Y + room.Height) * s.scaleY)

			if mx < minX {
				mx = minX
			}

			if mx > maxX {
				mx = maxX
			}

			if my < minY {
				my = minY
			}

			if my > maxY {
				my = maxY
			}
		}
	}

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

	faceLabel := ui.Face(true, 9)
	faceSmall := ui.Face(false, 7)

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

			if c.Puzzles[0].TargetRecord != nil {
				hash := c.Puzzles[0].TargetRecord.Hash
				if len(hash) > 12 {
					hash = hash[:12] + "..."
				}

				name = hash
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
			avatarSrc := fmt.Sprintf("avatars/%s.jpg", c.Puzzles[0].TargetRecord.AvatarKey)
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
			rec := c.Puzzles[0].TargetRecord

			lines := []string{
				fmt.Sprintf("Case: %s", c.Name),
				fmt.Sprintf("Difficulty: %d pieces", c.Puzzles[0].PiecesToSolve),
				fmt.Sprintf("Color: %s", rec.Color),
				fmt.Sprintf("Variant: %d", rec.Variant),
			}

			if c.Puzzles[0].HideColor {
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

func (s *GameScene) drawPuzzleContent(screen *ebiten.Image) { //nolint:gocyclo // puzzle rendering
	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return
	}

	faceHash := ui.Face(false, 10)

	// Draw hash
	if hashObj := tilemap.FindObject(og, "hash"); hashObj != nil {
		hx := hashObj.X * s.scaleX
		hy := hashObj.Y * s.scaleY

		if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
			hashText := s.computeCurrentHash()

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

		if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
			c := s.cases[s.selectedCase]
			caseIdx := s.selectedCase

			missingSet := make(map[int]bool)

			for _, idx := range c.Puzzles[0].MissingIndices {
				missingSet[idx] = true
			}

			// Get the fingerprint images for this case
			imgs := s.caseImages[caseIdx]
			greyImgs := s.greyImages[caseIdx]

			// Draw pre-filled pieces (not missing)
			for idx := range 100 {
				if missingSet[idx] {
					continue
				}

				col := idx % 10
				row := idx / 10
				cx := px + float64(col)*cellW
				cy := py + float64(row)*cellH

				// Use grey images if color is hidden, otherwise colored
				var pieceImg *ebiten.Image

				if c.Puzzles[0].HideColor && greyImgs != nil {
					pieceImg = greyImgs.Pieces[idx]
				} else if imgs != nil {
					pieceImg = imgs.Pieces[idx]
				}

				if pieceImg != nil {
					op := &ebiten.DrawImageOptions{}
					iw := float64(pieceImg.Bounds().Dx())
					op.GeoM.Scale(cellW/iw, cellH/iw)
					op.GeoM.Translate(cx, cy)
					screen.DrawImage(pieceImg, op)
				}
			}

			// Track placed pieces
			placedAt := make(map[int]int)

			for ti, tp := range c.Puzzles[0].TrayPieces {
				if tp.IsPlaced {
					gIdx := tp.PlacedY*10 + tp.PlacedX
					placedAt[gIdx] = ti
				}
			}

			// Draw missing slots (empty or with placed piece)
			for _, idx := range c.Puzzles[0].MissingIndices {
				col := idx % 10
				row := idx / 10
				cx := px + float64(col)*cellW
				cy := py + float64(row)*cellH

				if ti, ok := placedAt[idx]; ok {
					tp := c.Puzzles[0].TrayPieces[ti]
					pieceImg := s.getPieceImage(tp, imgs)

					if pieceImg != nil {
						s.drawRotatedPiece(screen, pieceImg, cx, cy, cellW, tp.Rotation)
					} else {
						clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0xCC}
						if tp.IsDecoy {
							clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0xCC}
						}

						ui.DrawRoundedRect(screen, float32(cx+1), float32(cy+1),
							float32(cellW-2), float32(cellH-2), 1, clr)
					}
				} else {
					// Empty slot highlight
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

			caseIdx := s.selectedCase
			cImgs := s.caseImages[caseIdx]

			for i, tp := range c.Puzzles[0].TrayPieces {
				if tp.IsPlaced {
					continue
				}

				col := i % 3
				row := i / 3
				tx := px + float64(col)*pieceSize
				ty := py + float64(row)*pieceSize

				// Get the piece image (correct or decoy)
				pieceImg := s.getPieceImage(tp, cImgs)
				if pieceImg != nil {
					s.drawRotatedPiece(screen, pieceImg, tx, ty, pieceSize, tp.Rotation)
				} else {
					clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0xCC}
					if tp.IsDecoy {
						clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0xCC}
					}

					ui.DrawRoundedRect(screen, float32(tx+1), float32(ty+1),
						float32(pieceSize-2), float32(pieceSize-2), 2, clr)
				}

				// Highlight if this piece is being held
				if i == s.holdingPiece {
					ui.DrawRoundedRect(screen, float32(tx), float32(ty),
						float32(pieceSize), float32(pieceSize), 2,
						color.RGBA{R: 0xFF, G: 0xFF, B: 0x00, A: 0x60})
				}
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

				for i := range c.Puzzles[0].TrayPieces {
					if c.Puzzles[0].TrayPieces[i].IsPlaced {
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
		case "hash":
			// Send button: positioned near the hash area (from tile layer at 1760,1408)
			sendX := 1760.0 * s.scaleX
			sendY := 1408.0 * s.scaleY
			sendW := 200.0 * s.scaleX
			sendH := 50.0 * s.scaleY

			s.input.AddZone(&systems.Zone{
				Spatial: shapes.NewBox(shapes.NewPoint(sendX, sendY), sendW, sendH),
				OnClick: func() {
					s.submitPuzzle()
				},
			})
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

				for _, idx := range c.Puzzles[0].MissingIndices {
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
							// If clicking a cell that already has a placed piece, pick it up
							for ti := range c.Puzzles[0].TrayPieces {
								tp := &c.Puzzles[0].TrayPieces[ti]
								if tp.IsPlaced && tp.PlacedY*10+tp.PlacedX == gIdx {
									tp.IsPlaced = false
									tp.PlacedX = -1
									tp.PlacedY = -1
									s.holdingPiece = ti

									slog.Info("picked up placed piece", "tray", ti, "grid", gIdx)

									return
								}
							}

							// Place the held piece
							if s.holdingPiece < 0 || s.holdingPiece >= len(c.Puzzles[0].TrayPieces) {
								return
							}

							tp := &c.Puzzles[0].TrayPieces[s.holdingPiece]
							tp.IsPlaced = true
							tp.PlacedX = gIdx % 10
							tp.PlacedY = gIdx / 10

							slog.Info("placed piece", "tray", s.holdingPiece,
								"grid", gIdx, "x", tp.PlacedX, "y", tp.PlacedY)

							s.holdingPiece = -1
							s.saveGameState()
						},
					})
				}
			}
		}
	}
}

// getPieceImage returns the correct image for a tray piece (correct or decoy).
func (s *GameScene) getPieceImage(tp domain.TrayPiece, caseImgs *FingerprintImages) *ebiten.Image {
	if !tp.IsDecoy {
		// Correct piece from the target fingerprint
		if caseImgs != nil && tp.OriginalX >= 0 {
			origIdx := tp.OriginalY*10 + tp.OriginalX
			if origIdx >= 0 && origIdx < 100 {
				return caseImgs.Pieces[origIdx]
			}
		}

		return nil
	}

	// Decoy piece from a different fingerprint
	key := fmt.Sprintf("%s.%d", tp.DecoyColor, tp.DecoyVariant)

	if imgs, ok := s.allImages[key]; ok && tp.DecoyPieceIdx >= 0 && tp.DecoyPieceIdx < 100 {
		return imgs.Pieces[tp.DecoyPieceIdx]
	}

	return nil
}

// drawRotatedPiece draws a piece image with rotation at the given position and size.
func (s *GameScene) drawRotatedPiece(screen *ebiten.Image, img *ebiten.Image, x, y, size float64, rotation int) {
	iw := float64(img.Bounds().Dx())
	ih := float64(img.Bounds().Dy())
	scale := size / iw

	op := &ebiten.DrawImageOptions{}

	// Rotate around center
	op.GeoM.Translate(-iw/2, -ih/2)

	switch rotation % 4 {
	case 1:
		op.GeoM.Rotate(math.Pi / 2)
	case 2:
		op.GeoM.Rotate(math.Pi)
	case 3:
		op.GeoM.Rotate(3 * math.Pi / 2)
	}

	op.GeoM.Translate(iw/2, ih/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)

	screen.DrawImage(img, op)
}

// computeCurrentHash computes the live hash from placed pieces on the grid.
func (s *GameScene) computeCurrentHash() string {
	c := s.cases[s.selectedCase]
	p := c.Puzzles[0]

	// Build grid: start with all correct pieces, then overlay placed tray pieces
	pieces := make([]domain.PieceRecord, 100)

	for i, piece := range p.TargetRecord.Pieces {
		pieces[i] = piece
	}

	// Zero out missing indices
	for _, idx := range p.MissingIndices {
		pieces[idx] = domain.PieceRecord{X: idx % 10, Y: idx / 10, Value: 0}
	}

	// Place tray pieces
	for _, tp := range p.TrayPieces {
		if !tp.IsPlaced {
			continue
		}

		gIdx := tp.PlacedY*10 + tp.PlacedX
		if gIdx >= 0 && gIdx < 100 {
			pieces[gIdx] = domain.PieceRecord{X: tp.PlacedX, Y: tp.PlacedY, Value: tp.Value}
		}
	}

	// Determine color letter
	colorLetter := domain.ColorLetter(p.TargetRecord.Color)
	if p.HideColor {
		colorLetter = "?"
	}

	hash := domain.ComputeHash(pieces)

	return fmt.Sprintf("%s%d", colorLetter, hash)
}

func (s *GameScene) submitPuzzle() {
	if s.selectedCase < 0 || s.selectedCase >= len(s.cases) {
		return
	}

	// Compute the ACTUAL hash from current grid state (not target)
	currentHash := s.computeCurrentHash()
	found := s.db.LookupByHash(currentHash)

	if found != nil {
		slog.Info("SUBMIT: person found!", "name", found.PersonName, "hash", currentHash)

		s.showResult = 1
	} else {
		slog.Info("SUBMIT: no match", "hash", currentHash)

		s.showResult = 2
	}

	s.resultTick = 180 // show for 3 seconds
	s.saveGameState()
}

func (s *GameScene) saveGameState() {
	save := &domain.GameSave{}

	for i, c := range s.cases {
		cs := domain.CaseSave{CaseIndex: i}

		for j, tp := range c.Puzzles[0].TrayPieces {
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
			if pp.TrayIndex < 0 || pp.TrayIndex >= len(c.Puzzles[0].TrayPieces) {
				continue
			}

			tp := &c.Puzzles[0].TrayPieces[pp.TrayIndex]
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
