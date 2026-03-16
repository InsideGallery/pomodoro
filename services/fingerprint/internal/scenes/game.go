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
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
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
	StateLoading           GameState = iota // Loading assets
	StateDisabled                           // PC off
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

	state          GameState
	bootTick       int
	selectedCase   int // 0-9, which case is selected
	selectedPuzzle int // 0-99, which puzzle within selected case
	cases          []*domain.CaseConfig
	puzzleSeed     uint64

	// Puzzle state
	holdingPiece     int                           // index in tray (-1 = none)
	dragging         bool                          // true while mouse held on a piece
	showResult       int                           // 0=none, 1=success, 2=fail
	resultTick       int                           // frames to show result
	casesScroll      int                           // scroll offset for cases list
	namesScroll      int                           // scroll offset for fingerprint names list
	descScroll       int                           // scroll offset for description text
	targetImages     map[int]*FingerprintImages    // record ID → images (with rotation/mirror)
	targetGreyImages map[int]*FingerprintImages    // record ID → grey images
	allImages        map[string]*FingerprintImages // "color.variant" → images (for decoys)
	avatarCache      map[string]*ebiten.Image      // avatar filename → loaded image
	assetsDir        string

	// Loading progress
	loadProgress float64       // 0.0 → 1.0
	loadStatus   string        // current loading step text
	loadStep     int           // which deferred step we're on
	loadDone     chan struct{} // signals background work finished

	// Virtual cursor (delta-based, clamped to cursor-room without stickiness)
	cursorX, cursorY   int
	prevRawX, prevRawY int
	cursorInited       bool

	// Scaling: map is 4000×2176, screen may differ
	scaleX, scaleY float64

	width, height int
}

func NewGameScene() *GameScene {
	return &GameScene{}
}

// currentPuzzle returns the active puzzle, or nil.
func (s *GameScene) currentPuzzle() *domain.PuzzleConfig {
	if s.selectedCase < 0 || s.selectedCase >= len(s.cases) {
		return nil
	}

	c := s.cases[s.selectedCase]
	if s.selectedPuzzle < 0 || s.selectedPuzzle >= len(c.Puzzles) {
		return nil
	}

	return c.Puzzles[s.selectedPuzzle]
}

// currentPuzzleImages returns target and grey images for the active puzzle.
func (s *GameScene) currentPuzzleImages() (imgs, greyImgs *FingerprintImages) {
	p := s.currentPuzzle()
	if p == nil {
		return nil, nil
	}

	imgs = s.targetImages[p.TargetRecord.ID]

	if p.HideColor {
		greyImgs = s.targetGreyImages[p.TargetRecord.ID]
	}

	return
}

// firstUnsolvedPuzzle returns the index of first unsolved puzzle in a case, or 0 if all solved.
func firstUnsolvedPuzzle(c *domain.CaseConfig) int {
	for i, p := range c.Puzzles {
		if !p.Solved && !p.Failed {
			return i
		}
	}

	return 0
}

// cursorRoom returns the cursor-room bounds in screen coordinates.
func (s *GameScene) cursorRoom() (minX, minY, maxX, maxY int) {
	if s.tmap == nil {
		return 0, 0, s.width, s.height
	}

	mainOG := s.tmap.FindObjectGroup("main")
	if mainOG == nil {
		return 0, 0, s.width, s.height
	}

	room := tilemap.FindObject(mainOG, "cursor-room")
	if room == nil {
		return 0, 0, s.width, s.height
	}

	return int(room.X * s.scaleX), int(room.Y * s.scaleY),
		int((room.X + room.Width) * s.scaleX), int((room.Y + room.Height) * s.scaleY)
}

// updateCursor applies delta-based virtual cursor with hard clamping (no stickiness).
func (s *GameScene) updateCursor() {
	rawX, rawY := ebiten.CursorPosition()

	if !s.cursorInited {
		s.cursorX = rawX
		s.cursorY = rawY
		s.prevRawX = rawX
		s.prevRawY = rawY
		s.cursorInited = true

		return
	}

	dx := rawX - s.prevRawX
	dy := rawY - s.prevRawY
	s.prevRawX = rawX
	s.prevRawY = rawY

	s.cursorX += dx
	s.cursorY += dy

	minX, minY, maxX, maxY := s.cursorRoom()

	if s.cursorX < minX {
		s.cursorX = minX
	}

	if s.cursorX > maxX {
		s.cursorX = maxX
	}

	if s.cursorY < minY {
		s.cursorY = minY
	}

	if s.cursorY > maxY {
		s.cursorY = maxY
	}
}

// gridCellSize returns the puzzle cell size in screen pixels.
func (s *GameScene) gridCellSize() float64 {
	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return 40
	}

	puzzleObj := tilemap.FindObject(og, "puzzle")
	if puzzleObj == nil {
		return 40
	}

	pw := puzzleObj.Width * s.scaleX
	ph := puzzleObj.Height * s.scaleY
	side := pw

	if ph < side {
		side = ph
	}

	return side / 10
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

	// Start in loading state — heavy work deferred to first Update() frames
	s.state = StateLoading
	s.bootTick = 0
	s.holdingPiece = -1
	s.allImages = make(map[string]*FingerprintImages)
	s.targetImages = make(map[int]*FingerprintImages)
	s.targetGreyImages = make(map[int]*FingerprintImages)
	s.avatarCache = make(map[string]*ebiten.Image)
	s.assetsDir = FindFingerprintAssetsDir()

	ebiten.SetFullscreen(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	return nil
}

// loadDeferred runs heavy loading in background goroutines.
// Returns true when all loading is complete.
func (s *GameScene) loadDeferred() bool {
	// If a background job is running, wait for it
	if s.loadDone != nil {
		select {
		case <-s.loadDone:
			s.loadDone = nil
			s.loadStep++
		default:
			// Still working — animate progress bar
			if s.loadProgress < 0.95 {
				s.loadProgress += 0.003
			}

			return false
		}
	}

	switch s.loadStep {
	case 0:
		// Step 0: load stories + kick off TMX loading in background
		s.loadStatus = "Loading scene assets..."
		s.loadProgress = 0.1

		if s.assetsDir != "" {
			if err := domain.LoadStories(s.assetsDir); err != nil {
				slog.Warn("load stories", "error", err)
			} else {
				slog.Info("stories loaded", "cases", domain.CaseCount())
			}
		}
		s.loadDone = make(chan struct{})

		go func() {
			defer close(s.loadDone)

			tmxPath := FindTMXPath()
			if tmxPath == "" {
				slog.Error("fingerprint.tmx not found")

				return
			}

			m, err := tilemap.Load(tmxPath)
			if err != nil {
				slog.Error("load tmx", "error", err)

				return
			}

			s.tmap = m

			slog.Info("tmx loaded")
		}()

	case 1:
		// TMX loaded — compute scaling
		if s.tmap != nil {
			mapW := float64(s.tmap.MapPixelWidth())
			mapH := float64(s.tmap.MapPixelHeight())
			s.scaleX = float64(s.width) / mapW
			s.scaleY = float64(s.height) / mapH
		}

		// Step 1: kick off DB loading in background
		s.loadStatus = "Loading fingerprint database..."
		s.loadProgress = 0.6
		s.loadDone = make(chan struct{})

		go func() {
			defer close(s.loadDone)

			dbPath := domain.DefaultDBPath()

			db, dbErr := domain.LoadDB(dbPath)
			if dbErr != nil {
				slog.Info("generating fingerprint DB")

				db = domain.GenerateDB(42)
				if err := db.Save(dbPath); err != nil {
					slog.Warn("save db", "error", err)
				}
			}

			s.db = db
		}()

	case 2:
		// Step 2: generate/load puzzles in background
		s.loadStatus = "Generating puzzles..."
		s.loadProgress = 0.8
		s.loadDone = make(chan struct{})

		go func() {
			defer close(s.loadDone)

			puzzlesPath := domain.DefaultPuzzlesPath()
			loadedCases, loadedSeed, loadErr := domain.LoadPuzzles(puzzlesPath, s.db)

			if loadErr != nil {
				s.puzzleSeed = 99
				s.cases = domain.GenerateCases(s.db, s.puzzleSeed)

				if err := domain.SavePuzzles(s.cases, s.puzzleSeed, puzzlesPath); err != nil {
					slog.Warn("save puzzles", "error", err)
				}
			} else {
				s.puzzleSeed = loadedSeed
				s.cases = loadedCases
			}
		}()

	case 3:
		// All done
		s.loadStatus = "Ready"
		s.loadProgress = 1.0
		s.selectedCase = 0
		s.loadGameState()

		slog.Info("game ready", "cases", len(s.cases))

		return true
	}

	return false
}

func (s *GameScene) Unload() error {
	ebiten.SetCursorMode(ebiten.CursorModeVisible)

	return nil
}

func (s *GameScene) Update() error {
	if s.state == StateLoading {
		slog.Info("loading tick", "step", s.loadStep, "progress", fmt.Sprintf("%.0f%%", s.loadProgress*100), "status", s.loadStatus)

		if s.loadDeferred() {
			s.state = StateDisabled
			s.bootTick = 0
		}

		return nil
	}

	if s.tmap == nil {
		return nil
	}

	s.updateCursor()
	s.input.CursorOverride = &[2]int{s.cursorX, s.cursorY}

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

		// Mouse wheel scrolls whichever area the cursor is over
		_, wy := ebiten.Wheel()
		if wy != 0 {
			og := s.tmap.FindObjectGroup("application-layout")
			cx, cy := float64(s.cursorX), float64(s.cursorY)
			handled := false

			if og != nil {
				// Check cases list
				if !handled {
					if obj := tilemap.FindObject(og, "list-of-cases"); obj != nil {
						rx, ry := obj.X*s.scaleX, obj.Y*s.scaleY
						rw, rh := obj.Width*s.scaleX, obj.Height*s.scaleY

						if cx >= rx && cx <= rx+rw && cy >= ry && cy <= ry+rh {
							if wy > 0 {
								s.casesScroll--
							} else {
								s.casesScroll++
							}

							if s.casesScroll < 0 {
								s.casesScroll = 0
							}

							handled = true
						}
					}
				}

				// Check description area
				if !handled {
					if obj := tilemap.FindObject(og, "description"); obj != nil {
						rx, ry := obj.X*s.scaleX, obj.Y*s.scaleY
						rw, rh := obj.Width*s.scaleX, obj.Height*s.scaleY

						if cx >= rx && cx <= rx+rw && cy >= ry && cy <= ry+rh {
							if wy > 0 {
								s.descScroll--
							} else {
								s.descScroll++
							}

							if s.descScroll < 0 {
								s.descScroll = 0
							}

							handled = true
						}
					}
				}

				// Default: scroll names list
				if !handled {
					if wy > 0 {
						s.namesScroll--
					} else {
						s.namesScroll++
					}

					if s.namesScroll < 0 {
						s.namesScroll = 0
						s.descScroll = 0
					}
				}
			}
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
			s.holdingPiece = -1
			s.dragging = false
			s.registerAppLayoutZones()
		}

		// Camera zoom
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			s.Camera.ZoomFactor += 5
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyX) {
			s.Camera.ZoomFactor -= 5
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyC) {
			s.Camera.Reset()
		}

		// Mouse wheel: rotate held piece (up = clockwise, down = counter-clockwise)
		_, wy := ebiten.Wheel()
		if wy != 0 && s.holdingPiece >= 0 {
			if p := s.currentPuzzle(); p != nil && s.holdingPiece < len(p.TrayPieces) {
				if wy > 0 {
					p.TrayPieces[s.holdingPiece].Rotation = (p.TrayPieces[s.holdingPiece].Rotation + 1) % domain.RotationSteps
				} else {
					p.TrayPieces[s.holdingPiece].Rotation = (p.TrayPieces[s.holdingPiece].Rotation + domain.RotationSteps - 1) % domain.RotationSteps
				}
			}
		}

		// Drag-and-drop
		s.updateDragDrop()

		if err := s.input.Update(s.Ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *GameScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{A: 0xFF})

	if s.state == StateLoading || s.tmap == nil {
		cx := float64(s.width) / 2
		cy := float64(s.height) / 2

		// Title
		ui.DrawTextCentered(screen, "Loading...", ui.Face(true, 14),
			cx, cy-40, color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF})

		// Progress bar background
		barW := 300.0
		barH := 12.0
		barX := cx - barW/2
		barY := cy - barH/2

		ui.DrawRoundedRect(screen, float32(barX), float32(barY),
			float32(barW), float32(barH), 4,
			color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF})

		// Progress bar fill
		fillW := barW * s.loadProgress
		if fillW > 0 {
			ui.DrawRoundedRect(screen, float32(barX), float32(barY),
				float32(fillW), float32(barH), 4,
				color.RGBA{R: 0x2E, G: 0x86, B: 0x8E, A: 0xFF})
		}

		// Status text
		if s.loadStatus != "" {
			ui.DrawTextCentered(screen, s.loadStatus, ui.Face(false, 8),
				cx, cy+20, color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF})
		}

		return
	}

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
	if s.holdingPiece >= 0 && s.dragging && s.state == StateApplicationNet {
		if p := s.currentPuzzle(); p != nil && s.holdingPiece < len(p.TrayPieces) {
			tp := p.TrayPieces[s.holdingPiece]
			mx, my := s.cursorX, s.cursorY

			imgs, _ := s.currentPuzzleImages()
			pieceImg := s.getPieceImage(tp, imgs)
			sz := s.gridCellSize()

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
			qw := 200.0 * s.scaleX
			qh := 50.0 * s.scaleY

			face := ui.Face(true, 11)
			ui.DrawRoundedRect(screen, float32(qx), float32(qy), float32(qw), float32(qh), 4,
				color.RGBA{R: 0xCC, G: 0x33, B: 0x33, A: 0xCC})

			ui.DrawTextCentered(screen, "QUIT", face, qx+qw/2, qy+qh/2-8,
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

	op := &ebiten.DrawImageOptions{}
	cw := float64(cursorImg.Bounds().Dx())
	cursorScale := 32.0 / cw
	op.GeoM.Scale(cursorScale, cursorScale)
	op.GeoM.Translate(float64(s.cursorX), float64(s.cursorY))
	screen.DrawImage(cursorImg, op)
}

// --- Step 4: Application layout dynamic content ---

func (s *GameScene) drawAppContent(screen *ebiten.Image) {
	og := s.tmap.FindObjectGroup("application-layout")
	if og == nil {
		return
	}

	faceList := ui.Face(true, 9)
	faceBtn := ui.Face(true, 8)
	white := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	textClr := color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF}

	// Draw scrollable case names in list-of-cases room
	if casesObj := tilemap.FindObject(og, "list-of-cases"); casesObj != nil {
		x := casesObj.X * s.scaleX
		y := casesObj.Y * s.scaleY
		w := casesObj.Width * s.scaleX
		h := casesObj.Height * s.scaleY
		rowH := 45.0 * s.scaleY

		maxScroll := len(s.cases) - int(h/rowH)
		if maxScroll < 0 {
			maxScroll = 0
		}

		if s.casesScroll > maxScroll {
			s.casesScroll = maxScroll
		}

		if s.casesScroll < 0 {
			s.casesScroll = 0
		}

		for i := s.casesScroll; i < len(s.cases); i++ {
			ry := y + float64(i-s.casesScroll)*rowH
			if ry+rowH > y+h {
				break
			}

			c := s.cases[i]

			btnClr := color.RGBA{R: 0x3A, G: 0x5A, B: 0x5A, A: 0x80}
			if i == s.selectedCase {
				btnClr = color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x60}
			}

			ui.DrawRoundedRect(screen, float32(x+1), float32(ry+1), float32(w-2), float32(rowH-2), 2, btnClr)

			solved := 0
			for _, p := range c.Puzzles {
				if p.Solved || p.Failed {
					solved++
				}
			}

			label := fmt.Sprintf("%s (%d/%d)", c.Name, solved, len(c.Puzzles))
			ui.DrawText(screen, label, faceList, x+6, ry+8, textClr)
		}

		// Scrollbar
		if maxScroll > 0 {
			sbH := h * h / (float64(len(s.cases)) * rowH)
			sbY := y + float64(s.casesScroll)/float64(maxScroll)*(h-sbH)
			ui.DrawRoundedRect(screen, float32(x+w-4), float32(sbY), 3, float32(sbH), 1,
				color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80})
		}
	}

	// Draw scrollable fingerprint puzzle list for selected case
	if namesObj := tilemap.FindObject(og, "fingerprints-user-names"); namesObj != nil && s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
		x := namesObj.X * s.scaleX
		y := namesObj.Y * s.scaleY
		w := namesObj.Width * s.scaleX
		h := namesObj.Height * s.scaleY
		rowH := 50.0 * s.scaleY
		c := s.cases[s.selectedCase]

		// Clamp scroll
		maxScroll := len(c.Puzzles) - int(h/rowH)
		if maxScroll < 0 {
			maxScroll = 0
		}

		if s.namesScroll > maxScroll {
			s.namesScroll = maxScroll
		}

		if s.namesScroll < 0 {
			s.namesScroll = 0
		}

		for i := s.namesScroll; i < len(c.Puzzles); i++ {
			ry := y + float64(i-s.namesScroll)*rowH
			if ry+rowH > y+h {
				break
			}

			p := c.Puzzles[i]
			name := "Unknown"

			if p.Solved && p.TargetRecord != nil {
				name = p.TargetRecord.PersonName
			} else if p.Failed {
				name = "No match"
			}

			btnClr := color.RGBA{R: 0x3A, G: 0x5A, B: 0x5A, A: 0xAA}
			txtClr := textClr

			if i == s.selectedPuzzle {
				btnClr = color.RGBA{R: 0x2E, G: 0x86, B: 0x8E, A: 0xCC}
				txtClr = white
			}

			label := fmt.Sprintf("%d. %s", i+1, name)
			ui.DrawRoundedRect(screen, float32(x+1), float32(ry+1), float32(w-2), float32(rowH-2), 2, btnClr)
			ui.DrawText(screen, label, faceList, x+6, ry+8, txtClr)
		}

		// Scrollbar indicator
		if maxScroll > 0 {
			sbH := h * h / (float64(len(c.Puzzles)) * rowH)
			sbY := y + float64(s.namesScroll)/float64(maxScroll)*(h-sbH)
			ui.DrawRoundedRect(screen, float32(x+w-4), float32(sbY), 3, float32(sbH), 1,
				color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80})
		}
	}

	// Draw avatar for selected case's current puzzle
	if avatarObj := tilemap.FindObject(og, "avatar"); avatarObj != nil {
		ax := avatarObj.X * s.scaleX
		ay := avatarObj.Y * s.scaleY
		aw := avatarObj.Width * s.scaleX
		ah := avatarObj.Height * s.scaleY

		if p := s.currentPuzzle(); p != nil {
			avatarFile := domain.UnknownAvatar
			if p.Solved {
				avatarFile = domain.AvatarForRecord(p.TargetRecord)
			}

			avatarImg := s.getAvatar(avatarFile)

			if avatarImg != nil {
				op := &ebiten.DrawImageOptions{}
				iw := float64(avatarImg.Bounds().Dx())
				ih := float64(avatarImg.Bounds().Dy())
				op.GeoM.Scale(aw/iw, ah/ih)
				op.GeoM.Translate(ax, ay)
				screen.DrawImage(avatarImg, op)
			} else {
				ui.DrawRoundedRect(screen, float32(ax), float32(ay), float32(aw), float32(ah), 4,
					color.RGBA{R: 0xD5, G: 0xF2, B: 0xF1, A: 0xFF})
				ui.DrawTextCentered(screen, "?", ui.Face(true, 24), ax+aw/2, ay+ah/2-10, textClr)
			}
		}
	}

	// Draw description — story-based narrative (word-wrapped, scrollable, clipped to box)
	if descObj := tilemap.FindObject(og, "description"); descObj != nil {
		dx := descObj.X * s.scaleX
		dy := descObj.Y * s.scaleY
		dw := descObj.Width * s.scaleX
		dh := descObj.Height * s.scaleY

		if p := s.currentPuzzle(); p != nil {
			var descText string

			switch {
			case p.Solved:
				descText = domain.SolvedDescription(s.selectedCase, s.selectedPuzzle, p.TargetRecord.PersonName)
			case p.Failed:
				descText = domain.NoMatchDescription(s.selectedCase, s.selectedPuzzle)
			default:
				descText = domain.UnsolvedDescription(s.selectedCase, s.selectedPuzzle)
			}

			s.drawWrappedText(screen, descText, dx+4, dy+4, dw-8, dh-8, s.descScroll, textClr)
		}
	}

	// Draw play-puzzle button (programmatic)
	if btnObj := tilemap.FindObject(og, "play-puzzle"); btnObj != nil {
		bx := btnObj.X * s.scaleX
		by := btnObj.Y * s.scaleY
		bw := btnObj.Width * s.scaleX
		bh := btnObj.Height * s.scaleY

		ui.DrawRoundedRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 4,
			color.RGBA{R: 0x2E, G: 0x86, B: 0x8E, A: 0xDD})
		ui.DrawTextCentered(screen, "OPEN PUZZLE", faceBtn, bx+bw/2, by+bh/2-6, white)
	}

	// Draw regenerate-puzzles button (programmatic)
	if btnObj := tilemap.FindObject(og, "regenerate-puzzles"); btnObj != nil {
		bx := btnObj.X * s.scaleX
		by := btnObj.Y * s.scaleY
		bw := btnObj.Width * s.scaleX
		bh := btnObj.Height * s.scaleY

		ui.DrawRoundedRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 4,
			color.RGBA{R: 0x8E, G: 0x44, B: 0x2E, A: 0xDD})
		ui.DrawTextCentered(screen, "REGENERATE", faceBtn, bx+bw/2, by+bh/2-6, white)
	}
}

// --- Step 5: Puzzle workspace content ---

func (s *GameScene) drawPuzzleContent(screen *ebiten.Image) { //nolint:gocyclo // puzzle rendering
	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return
	}

	p := s.currentPuzzle()
	if p == nil {
		return
	}

	imgs, greyImgs := s.currentPuzzleImages()
	faceHash := ui.Face(false, 10)

	// Draw hash
	if hashObj := tilemap.FindObject(og, "hash"); hashObj != nil {
		hx := hashObj.X * s.scaleX
		hy := hashObj.Y * s.scaleY
		hashText := s.computeCurrentHash()

		ui.DrawText(screen, hashText, faceHash, hx+4, hy+4,
			color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})
	}

	// Draw puzzle grid area (forced square using smaller dimension)
	if puzzleObj := tilemap.FindObject(og, "puzzle"); puzzleObj != nil {
		px := puzzleObj.X * s.scaleX
		py := puzzleObj.Y * s.scaleY
		pw := puzzleObj.Width * s.scaleX
		ph := puzzleObj.Height * s.scaleY

		side := pw
		if ph < side {
			side = ph
		}

		px += (pw - side) / 2
		py += (ph - side) / 2
		cellW := side / 10
		cellH := cellW

		// Draw grid lines
		for i := range 11 {
			x := float32(px + float64(i)*cellW)
			ui.DrawRoundedRect(screen, x, float32(py), 1, float32(side), 0,
				color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x40})

			y := float32(py + float64(i)*cellH)
			ui.DrawRoundedRect(screen, float32(px), y, float32(side), 1, 0,
				color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x40})
		}

		missingSet := make(map[int]bool)
		for _, idx := range p.MissingIndices {
			missingSet[idx] = true
		}

		// Draw pre-filled pieces (not missing)
		for idx := range 100 {
			if missingSet[idx] {
				continue
			}

			col := idx % 10
			row := idx / 10
			cx := px + float64(col)*cellW
			cy := py + float64(row)*cellH

			var pieceImg *ebiten.Image

			if p.HideColor && greyImgs != nil {
				pieceImg = greyImgs.Pieces[idx]
			} else if imgs != nil {
				pieceImg = imgs.Pieces[idx]
			}

			if pieceImg != nil {
				op := &ebiten.DrawImageOptions{}
				iw := float64(pieceImg.Bounds().Dx())
				ih := float64(pieceImg.Bounds().Dy())
				op.GeoM.Scale(cellW/iw, cellH/ih)
				op.GeoM.Translate(cx, cy)
				screen.DrawImage(pieceImg, op)
			}
		}

		// Track placed pieces
		placedAt := make(map[int]int)
		for ti, tp := range p.TrayPieces {
			if tp.IsPlaced {
				gIdx := tp.PlacedY*10 + tp.PlacedX
				placedAt[gIdx] = ti
			}
		}

		// Draw missing slots (empty or with placed piece)
		for _, idx := range p.MissingIndices {
			col := idx % 10
			row := idx / 10
			cx := px + float64(col)*cellW
			cy := py + float64(row)*cellH

			if ti, ok := placedAt[idx]; ok {
				tp := p.TrayPieces[ti]
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
				ui.DrawRoundedRect(screen, float32(cx+1), float32(cy+1),
					float32(cellW-2), float32(cellH-2), 1,
					color.RGBA{R: 0xFF, G: 0xA0, B: 0x00, A: 0x30})
			}
		}
	}

	// Draw tray pieces at their free-form TrayX/TrayY positions (same size as grid cells)
	cellSz := s.gridCellSize()

	for i, tp := range p.TrayPieces {
		if tp.IsPlaced || (s.dragging && i == s.holdingPiece) {
			continue
		}

		tx := tp.TrayX * s.scaleX
		ty := tp.TrayY * s.scaleY

		pieceImg := s.getPieceImage(tp, imgs)
		if pieceImg != nil {
			s.drawRotatedPiece(screen, pieceImg, tx, ty, cellSz, tp.Rotation)
		} else {
			clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0xCC}
			if tp.IsDecoy {
				clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0xCC}
			}

			ui.DrawRoundedRect(screen, float32(tx+1), float32(ty+1),
				float32(cellSz-2), float32(cellSz-2), 2, clr)
		}
	}
}

// piecesRects returns all "pieces" rectangles from the net-layout object group (map coords).
func (s *GameScene) piecesRects() []struct{ x, y, w, h float64 } {
	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return nil
	}

	var rects []struct{ x, y, w, h float64 }

	for _, obj := range og.Objects {
		if obj.Name == "pieces" {
			rects = append(rects, struct{ x, y, w, h float64 }{
				x: obj.X, y: obj.Y, w: obj.Width, h: obj.Height,
			})
		}
	}

	return rects
}

// initTrayPositions assigns random (TrayX, TrayY) to each non-placed tray piece.
func (s *GameScene) initTrayPositions(p *domain.PuzzleConfig) {
	rects := s.piecesRects()
	if len(rects) == 0 {
		return
	}

	cellMap := s.gridCellSizeMap()

	for i := range p.TrayPieces {
		tp := &p.TrayPieces[i]
		if tp.IsPlaced || (tp.TrayX != 0 && tp.TrayY != 0) {
			continue // already has a position
		}

		// Pick a random rect and random position within it
		r := rects[i%len(rects)]
		tp.TrayX = r.x + float64(i%3)*cellMap + float64(i%2)*10
		tp.TrayY = r.y + float64(i/3)*cellMap + float64(i%2)*5

		// Clamp within rect
		if tp.TrayX+cellMap > r.x+r.w {
			tp.TrayX = r.x + r.w - cellMap
		}

		if tp.TrayY+cellMap > r.y+r.h {
			tp.TrayY = r.y + r.h - cellMap
		}
	}
}

// gridCellSizeMap returns the puzzle cell size in map coordinates.
func (s *GameScene) gridCellSizeMap() float64 {
	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return 68
	}

	puzzleObj := tilemap.FindObject(og, "puzzle")
	if puzzleObj == nil {
		return 68
	}

	side := puzzleObj.Width
	if puzzleObj.Height < side {
		side = puzzleObj.Height
	}

	return side / 10
}

// isInsidePiecesRoom checks if screen coords (sx, sy) are inside any pieces rectangle.
func (s *GameScene) isInsidePiecesRoom(sx, sy float64) bool {
	for _, r := range s.piecesRects() {
		rx := r.x * s.scaleX
		ry := r.y * s.scaleY
		rw := r.w * s.scaleX
		rh := r.h * s.scaleY

		if sx >= rx && sx <= rx+rw && sy >= ry && sy <= ry+rh {
			return true
		}
	}

	return false
}

// puzzleGridInfo returns the grid origin and cell size in screen coordinates.
func (s *GameScene) puzzleGridInfo() (px, py, cellW float64, ok bool) {
	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return 0, 0, 0, false
	}

	puzzleObj := tilemap.FindObject(og, "puzzle")
	if puzzleObj == nil {
		return 0, 0, 0, false
	}

	px = puzzleObj.X * s.scaleX
	py = puzzleObj.Y * s.scaleY
	pw := puzzleObj.Width * s.scaleX
	ph := puzzleObj.Height * s.scaleY

	side := pw
	if ph < side {
		side = ph
	}

	px += (pw - side) / 2
	py += (ph - side) / 2
	cellW = side / 10

	return px, py, cellW, true
}

// updateDragDrop handles mouse press → drag → release for puzzle pieces.
func (s *GameScene) updateDragDrop() { //nolint:gocyclo // drag-drop state machine
	p := s.currentPuzzle()
	if p == nil {
		return
	}

	cx, cy := float64(s.cursorX), float64(s.cursorY)

	// Mouse just pressed → try to pick up a piece
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && s.holdingPiece < 0 {
		cellSz := s.gridCellSize()

		// Check tray pieces
		for i := range p.TrayPieces {
			tp := &p.TrayPieces[i]
			if tp.IsPlaced {
				continue
			}

			tx := tp.TrayX * s.scaleX
			ty := tp.TrayY * s.scaleY

			if cx >= tx && cx <= tx+cellSz && cy >= ty && cy <= ty+cellSz {
				s.holdingPiece = i
				s.dragging = true

				return
			}
		}

		// Check placed pieces on grid
		gpx, gpy, gcw, gridOk := s.puzzleGridInfo()
		if gridOk {
			missingSet := make(map[int]bool)
			for _, idx := range p.MissingIndices {
				missingSet[idx] = true
			}

			for ti := range p.TrayPieces {
				tp := &p.TrayPieces[ti]
				if !tp.IsPlaced {
					continue
				}

				gx := gpx + float64(tp.PlacedX)*gcw
				gy := gpy + float64(tp.PlacedY)*gcw

				if cx >= gx && cx <= gx+gcw && cy >= gy && cy <= gy+gcw {
					tp.IsPlaced = false
					tp.PlacedX = -1
					tp.PlacedY = -1
					s.holdingPiece = ti
					s.dragging = true

					slog.Info("picked up placed piece", "tray", ti)

					return
				}
			}
		}
	}

	// Mouse released while dragging → try to place
	if s.dragging && s.holdingPiece >= 0 && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		tp := &p.TrayPieces[s.holdingPiece]
		placed := false

		// Check if over a missing grid cell
		gpx, gpy, gcw, gridOk := s.puzzleGridInfo()

		if gridOk {
			missingSet := make(map[int]bool)
			for _, idx := range p.MissingIndices {
				missingSet[idx] = true
			}

			col := int((cx - gpx) / gcw)
			row := int((cy - gpy) / gcw)

			if col >= 0 && col < 10 && row >= 0 && row < 10 {
				gIdx := row*10 + col

				if missingSet[gIdx] {
					// Check no other piece already there
					occupied := false
					for _, other := range p.TrayPieces {
						if other.IsPlaced && other.PlacedX == col && other.PlacedY == row {
							occupied = true

							break
						}
					}

					if !occupied {
						tp.IsPlaced = true
						tp.PlacedX = col
						tp.PlacedY = row
						placed = true

						slog.Info("placed piece", "tray", s.holdingPiece, "x", col, "y", row)

						s.saveGameState()
					}
				}
			}
		}

		// If not placed on grid, drop back at cursor position within drag-and-drop-zone
		if !placed {
			mapX := cx / s.scaleX
			mapY := cy / s.scaleY
			cellMap := s.gridCellSizeMap()

			// Clamp to drag-and-drop-zone (the full working area)
			netOG := s.tmap.FindObjectGroup("application-net-layout")
			if dz := tilemap.FindObject(netOG, "drag-and-drop-zone"); dz != nil {
				if mapX < dz.X {
					mapX = dz.X
				}

				if mapX > dz.X+dz.Width-cellMap {
					mapX = dz.X + dz.Width - cellMap
				}

				if mapY < dz.Y {
					mapY = dz.Y
				}

				if mapY > dz.Y+dz.Height-cellMap {
					mapY = dz.Y + dz.Height - cellMap
				}
			}

			tp.TrayX = mapX
			tp.TrayY = mapY
		}

		s.holdingPiece = -1
		s.dragging = false
	}
}

// wrapText splits text into lines that fit within maxW pixels using the given face.
func wrapText(text string, face *textv2.GoTextFace, maxW float64) []string {
	var lines []string

	// Split on explicit newlines first
	for _, paragraph := range splitLines(text) {
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}

		words := splitWords(paragraph)
		current := ""

		for _, word := range words {
			test := current
			if test != "" {
				test += " "
			}

			test += word

			w, _ := ui.MeasureText(test, face)
			if w > maxW && current != "" {
				lines = append(lines, current)
				current = word
			} else {
				current = test
			}
		}

		if current != "" {
			lines = append(lines, current)
		}
	}

	return lines
}

func splitLines(s string) []string {
	var lines []string

	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	lines = append(lines, s[start:])

	return lines
}

func splitWords(s string) []string {
	var words []string

	start := -1
	for i, c := range s {
		if c == ' ' {
			if start >= 0 {
				words = append(words, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}

	if start >= 0 {
		words = append(words, s[start:])
	}

	return words
}

// drawWrappedText word-wraps text to maxW, clips to maxH, supports scroll offset.
func (s *GameScene) drawWrappedText(screen *ebiten.Image, text string, x, y, maxW, maxH float64, scroll int, clr color.Color) {
	face := ui.Face(false, 9)
	lineH := face.Size * 1.5
	lines := wrapText(text, face, maxW)

	drawn := 0

	for i, line := range lines {
		if i < scroll {
			continue
		}

		ty := y + float64(drawn)*lineH
		if ty+lineH > y+maxH {
			break
		}

		ui.DrawText(screen, line, face, x, ty, clr)
		drawn++
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
			// Polygon object — Width/Height are 0, use same size as drawn button
			qx := obj.X * s.scaleX
			qy := obj.Y * s.scaleY
			qw := 200.0 * s.scaleX
			qh := 50.0 * s.scaleY

			s.input.AddZone(&systems.Zone{
				Spatial: shapes.NewBox(shapes.NewPoint(qx, qy), qw, qh),
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
					slog.Info("opening puzzle workspace", "case", s.selectedCase, "puzzle", s.selectedPuzzle)

					s.ensureCurrentPuzzleImages()

					if p := s.currentPuzzle(); p != nil {
						s.initTrayPositions(p)
					}

					s.state = StateApplicationNet
					s.holdingPiece = -1
					s.dragging = false
					s.registerPuzzleZones()
				},
			})
		case "regenerate-puzzles":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					slog.Info("regenerating puzzles")
					s.regenerateCases()
				},
			})
		case "list-of-cases":
			objCopy := obj

			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					my := s.cursorY
					ry := objCopy.Y * s.scaleY
					rowH := 45.0 * s.scaleY
					relY := float64(my) - ry
					idx := int(relY/rowH) + s.casesScroll

					if idx >= 0 && idx < len(s.cases) {
						s.selectedCase = idx
						s.selectedPuzzle = firstUnsolvedPuzzle(s.cases[idx])
						s.namesScroll = 0
						s.descScroll = 0
					}
				},
			})
		case "fingerprints-user-names":
			objCopy := obj // capture for closure

			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					my := s.cursorY
					ny := objCopy.Y * s.scaleY
					rowH := 50.0 * s.scaleY
					relY := float64(my) - float64(ny)
					idx := int(relY/rowH) + s.namesScroll

					if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
						c := s.cases[s.selectedCase]
						if idx >= 0 && idx < len(c.Puzzles) {
							s.selectedPuzzle = idx
							s.descScroll = 0
						}
					}
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

	p := s.currentPuzzle()
	if p == nil {
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
			// Piece interaction handled by updateDragDrop() — no zones needed
		case "button-send-puzzle":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					s.submitPuzzle()
				},
			})
		case "puzzle":
			// Grid placement handled by updateDragDrop()
		}
	}
}

// getPieceImage returns the correct image for a tray piece (correct or decoy).
// Decoy images are loaded with the target's rotation+mirror so they look visually consistent.
func (s *GameScene) getPieceImage(tp domain.TrayPiece, caseImgs *FingerprintImages) *ebiten.Image {
	if !tp.IsDecoy {
		if caseImgs != nil && tp.OriginalX >= 0 {
			origIdx := tp.OriginalY*10 + tp.OriginalX
			if origIdx >= 0 && origIdx < 100 {
				return caseImgs.Pieces[origIdx]
			}
		}

		return nil
	}

	// Decoy piece — lazy-load with the TARGET's rotation+mirror
	p := s.currentPuzzle()
	rot := 0
	mirror := false

	if p != nil && p.TargetRecord != nil {
		rot = p.TargetRecord.Rotation
		mirror = p.TargetRecord.Mirrored
	}

	key := fmt.Sprintf("%s.%d.r%d.m%v", tp.DecoyColor, tp.DecoyVariant, rot, mirror)

	if _, ok := s.allImages[key]; !ok && s.assetsDir != "" {
		rec := &domain.FingerprintRecord{
			Color: tp.DecoyColor, Variant: tp.DecoyVariant,
			Rotation: rot, Mirrored: mirror,
		}

		imgs, err := LoadFingerprintImages(s.assetsDir, rec)
		if err != nil {
			slog.Warn("lazy load decoy", "key", key, "error", err)
			s.allImages[key] = nil
		} else {
			s.allImages[key] = imgs
		}
	}

	if imgs := s.allImages[key]; imgs != nil && tp.DecoyPieceIdx >= 0 && tp.DecoyPieceIdx < 100 {
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

	// Rotate around center: rotation 0-7 → 0°, 45°, 90°, ..., 315°
	op.GeoM.Translate(-iw/2, -ih/2)

	angle := float64(rotation%domain.RotationSteps) * math.Pi / 4
	if angle != 0 {
		op.GeoM.Rotate(angle)
	}

	op.GeoM.Translate(iw/2, ih/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)

	screen.DrawImage(img, op)
}

// computeCurrentHash computes the live hash from placed pieces on the grid.
func (s *GameScene) computeCurrentHash() string {
	p := s.currentPuzzle()
	if p == nil {
		return "?"
	}

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

// computeCurrentHashNum returns just the CRC64 number (no color prefix).
func (s *GameScene) computeCurrentHashNum() uint64 {
	p := s.currentPuzzle()
	if p == nil {
		return 0
	}

	pieces := make([]domain.PieceRecord, 100)

	for i, piece := range p.TargetRecord.Pieces {
		pieces[i] = piece
	}

	for _, idx := range p.MissingIndices {
		pieces[idx] = domain.PieceRecord{X: idx % 10, Y: idx / 10, Value: 0}
	}

	for _, tp := range p.TrayPieces {
		if !tp.IsPlaced {
			continue
		}

		gIdx := tp.PlacedY*10 + tp.PlacedX
		if gIdx >= 0 && gIdx < 100 {
			pieces[gIdx] = domain.PieceRecord{X: tp.PlacedX, Y: tp.PlacedY, Value: tp.Value}
		}
	}

	return domain.ComputeHash(pieces)
}

func (s *GameScene) submitPuzzle() {
	p := s.currentPuzzle()
	if p == nil {
		return
	}

	var found *domain.FingerprintRecord

	if p.HideColor {
		// Grey puzzle: try all 4 color letters to find a match
		hashNum := s.computeCurrentHashNum()

		for _, letter := range []string{"G", "R", "Y", "B"} {
			candidate := fmt.Sprintf("%s%d", letter, hashNum)
			if rec := s.db.LookupByHash(candidate); rec != nil {
				found = rec
				// Reveal the color — switch from grey to actual color
				p.HideColor = false

				slog.Info("SUBMIT: color revealed!", "color", rec.Color, "hash", candidate)

				break
			}
		}
	} else {
		currentHash := s.computeCurrentHash()
		found = s.db.LookupByHash(currentHash)
	}

	if found != nil {
		slog.Info("SUBMIT: person found!", "name", found.PersonName)

		p.Solved = true
		p.Failed = false
		s.showResult = 1
	} else {
		slog.Info("SUBMIT: no match")

		p.Failed = true
		s.showResult = 2
	}

	s.resultTick = 180
	s.saveGameState()
}

func (s *GameScene) saveGameState() {
	save := &domain.GameSave{}

	for i, c := range s.cases {
		cs := domain.CaseSave{
			CaseIndex:    i,
			ActivePuzzle: firstUnsolvedPuzzle(c),
		}

		for _, p := range c.Puzzles {
			ps := domain.PuzzleSave{Solved: p.Solved, Failed: p.Failed}

			for j, tp := range p.TrayPieces {
				if tp.IsPlaced || tp.TrayX != 0 || tp.TrayY != 0 || tp.Rotation != 0 {
					ps.PlacedPieces = append(ps.PlacedPieces, domain.PlacedSave{
						TrayIndex: j,
						GridX:     tp.PlacedX,
						GridY:     tp.PlacedY,
						Rotation:  tp.Rotation,
						TrayX:     tp.TrayX,
						TrayY:     tp.TrayY,
					})
				}
			}

			cs.Puzzles = append(cs.Puzzles, ps)
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

		for pi, ps := range cs.Puzzles {
			if pi >= len(c.Puzzles) {
				break
			}

			p := c.Puzzles[pi]
			p.Solved = ps.Solved
			p.Failed = ps.Failed

			for _, pp := range ps.PlacedPieces {
				if pp.TrayIndex < 0 || pp.TrayIndex >= len(p.TrayPieces) {
					continue
				}

				tp := &p.TrayPieces[pp.TrayIndex]
				tp.IsPlaced = pp.GridX >= 0 && pp.GridY >= 0
				tp.PlacedX = pp.GridX
				tp.PlacedY = pp.GridY
				tp.Rotation = pp.Rotation
				tp.TrayX = pp.TrayX
				tp.TrayY = pp.TrayY
			}
		}

		c.ID = cs.CaseIndex + 1
	}

	// Restore selectedPuzzle to first unsolved for selected case
	if s.selectedCase >= 0 && s.selectedCase < len(s.cases) {
		s.selectedPuzzle = firstUnsolvedPuzzle(s.cases[s.selectedCase])
	}

	slog.Info("game state loaded")
}

func (s *GameScene) regenerateCases() {
	// New seed for variety (keep 256 fingerprints, regenerate puzzles only)
	s.puzzleSeed = s.puzzleSeed ^ uint64(s.bootTick) ^ 0xFACE
	s.cases = domain.GenerateCases(s.db, s.puzzleSeed)
	s.selectedCase = 0
	s.selectedPuzzle = 0
	s.holdingPiece = -1
	s.namesScroll = 0
	s.descScroll = 0

	// Save new puzzles
	if err := domain.SavePuzzles(s.cases, s.puzzleSeed, domain.DefaultPuzzlesPath()); err != nil {
		slog.Warn("save puzzles", "error", err)
	}

	// Target images loaded lazily when entering puzzle workspace

	// Delete old game save
	if err := os.Remove(domain.DefaultSavePath()); err != nil {
		slog.Debug("remove save", "error", err)
	}

	s.registerAppLayoutZones()
	slog.Info("puzzles regenerated", "seed", s.puzzleSeed)
}

// getAvatar loads and caches an avatar image from disk.
func (s *GameScene) getAvatar(filename string) *ebiten.Image {
	if img, ok := s.avatarCache[filename]; ok {
		return img
	}

	if s.assetsDir == "" {
		return nil
	}

	path := filepath.Join(s.assetsDir, "avatars", filename)

	img, err := loadStdImage(path)
	if err != nil {
		slog.Warn("load avatar", "file", filename, "error", err)
		s.avatarCache[filename] = nil

		return nil
	}

	eImg := ebiten.NewImageFromImage(img)
	s.avatarCache[filename] = eImg

	return eImg
}

// ensureCurrentPuzzleImages loads target images for the current puzzle if not already cached.
func (s *GameScene) ensureCurrentPuzzleImages() {
	if s.assetsDir == "" {
		return
	}

	p := s.currentPuzzle()
	if p == nil {
		return
	}

	rec := p.TargetRecord

	if _, ok := s.targetImages[rec.ID]; !ok {
		imgs, err := LoadFingerprintImages(s.assetsDir, rec)
		if err != nil {
			slog.Warn("load target", "id", rec.ID, "error", err)
		} else {
			s.targetImages[rec.ID] = imgs
		}
	}

	if p.HideColor {
		if _, ok := s.targetGreyImages[rec.ID]; !ok {
			greyImgs, err := LoadGreyFingerprintImages(s.assetsDir, rec.Variant, rec.Rotation, rec.Mirrored)
			if err != nil {
				slog.Warn("load grey target", "id", rec.ID, "error", err)
			} else {
				s.targetGreyImages[rec.ID] = greyImgs
			}
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
