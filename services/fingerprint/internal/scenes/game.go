package scenes

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"

	"github.com/InsideGallery/core/memory/registry"
	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/InsideGallery/game-core/rtree"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/lafriks/go-tiled"

	"github.com/InsideGallery/pomodoro/pkg/core"
	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/systems"
	"github.com/InsideGallery/pomodoro/pkg/tilemap"
	"github.com/InsideGallery/pomodoro/pkg/ui"
	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
	"github.com/InsideGallery/pomodoro/services/fingerprint/internal/fsystems"
)

func statFile(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

const GameSceneName = "fingerprint_game"

// GameScene is the ECS-driven scene for the fingerprint game.
// It holds infrastructure (BaseScene, InputSystem, image caches) but NO game state.
// All mutable game state lives in components.GameData inside the Registry.
type GameScene struct {
	*scene.BaseScene

	// Infrastructure (not game state — these are framework handles)
	input *systems.InputSystem
	tmap  *tilemap.Map

	// Image caches (lazily loaded, shared across systems via accessor)
	targetImages     map[int]*FingerprintImages
	targetGreyImages map[int]*FingerprintImages
	allImages        map[string]*FingerprintImages
	avatarCache      map[string]*ebiten.Image

	// Scaling (computed from Layout, read-only for systems)
	scaleX, scaleY float64
	offsetX        float64
	width, height  int
}

func NewGameScene() *GameScene {
	return &GameScene{}
}

func (s *GameScene) Name() string { return GameSceneName }

func (s *GameScene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
	s.input = systems.NewInputSystem(s.RTree)

	// Create the singleton game state entity with GameData component
	gameEntity := &c.Entity{
		State:    &c.State{Current: c.StateLoading},
		GameData: &c.GameData{HoldingPiece: -1, AssetsDir: FindFingerprintAssetsDir()},
	}

	if err := s.Registry.Add(c.GroupGameState, 0, gameEntity); err != nil {
		slog.Error("create game state entity", "error", err)
	}

	// Create cursor entity with full-screen bounds (updated when TMX loads)
	cursorEntity := &c.Entity{Cursor: &c.Cursor{
		RoomMaxX: 3840, // generous default, refined after TMX load
		RoomMaxY: 2160,
	}}
	if err := s.Registry.Add(c.GroupCursor, 0, cursorEntity); err != nil {
		slog.Error("create cursor entity", "error", err)
	}

	// Register ECS systems in execution order
	dragDrop := fsystems.NewDragDropSystem(s)
	render := fsystems.NewRenderSystem(s, dragDrop)

	s.Systems.Add("state", fsystems.NewStateSystem(s))
	s.Systems.Add("cursor", fsystems.NewCursorSystem(s))
	s.Systems.Add("input", s.input)
	s.Systems.Add("scroll", fsystems.NewScrollSystem(s))
	s.Systems.Add("dragdrop", dragDrop)
	s.Systems.Add("camera", fsystems.NewCameraSystem(s))
	s.Systems.Add("render", render)

	slog.Info("ECS systems registered", "count", len(s.Systems.Get()))
}

func (s *GameScene) Load() error {
	if mon := ebiten.Monitor(); mon != nil {
		mw, mh := mon.Size()
		scale := mon.DeviceScaleFactor()
		s.width = int(float64(mw) * scale)
		s.height = int(float64(mh) * scale)
	}

	s.targetImages = make(map[int]*FingerprintImages)
	s.targetGreyImages = make(map[int]*FingerprintImages)
	s.allImages = make(map[string]*FingerprintImages)
	s.avatarCache = make(map[string]*ebiten.Image)

	ebiten.SetFullscreen(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	return nil
}

func (s *GameScene) Unload() error {
	ebiten.SetCursorMode(ebiten.CursorModeVisible)

	return nil
}

// Update delegates to BaseScene which iterates all systems in order.
func (s *GameScene) Update() error {
	return s.BaseScene.Update()
}

// Draw delegates to BaseScene which iterates all systems' Draw methods.
func (s *GameScene) Draw(screen *ebiten.Image) {
	s.BaseScene.Draw(screen)
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
			mapW := float64(s.tmap.MapPixelWidth())
			mapH := float64(s.tmap.MapPixelHeight())
			uniformScale := float64(h) / mapH
			s.scaleX = uniformScale
			s.scaleY = uniformScale
			s.offsetX = (float64(w) - mapW*uniformScale) / 2
		}
	}

	return w, h
}

// --- SceneAccessor implementation ---
// Systems access scene data through these methods.
// GameData component is the canonical source for mutable game state.

func (s *GameScene) gameData() *c.GameData {
	val, err := s.Registry.Get(c.GroupGameState, 0)
	if err != nil {
		return nil
	}

	if entity, ok := val.(*c.Entity); ok && entity.GameData != nil {
		return entity.GameData
	}

	return nil
}

func (s *GameScene) GetRegistry() *registry.Registry[string, uint64, any] {
	return s.Registry
}

func (s *GameScene) GetRTree() *rtree.RTree {
	return s.RTree
}

func (s *GameScene) GetCamera() *core.Camera {
	return s.Camera
}

func (s *GameScene) GetInputSystem() *systems.InputSystem {
	return s.input
}

func (s *GameScene) GetTileMap() *tilemap.Map {
	return s.tmap
}

func (s *GameScene) SetTileMap(m *tilemap.Map) {
	s.tmap = m
}

func (s *GameScene) getDB() *domain.FingerprintDB {
	if gd := s.gameData(); gd != nil {
		return gd.DB
	}

	return nil
}

func (s *GameScene) setDB(db *domain.FingerprintDB) {
	if gd := s.gameData(); gd != nil {
		gd.DB = db
	}
}

func (s *GameScene) getCases() []*domain.CaseConfig {
	if gd := s.gameData(); gd != nil {
		return gd.Cases
	}

	return nil
}

func (s *GameScene) setCases(cases []*domain.CaseConfig) {
	if gd := s.gameData(); gd != nil {
		gd.Cases = cases
	}
}

func (s *GameScene) getSelectedCase() int {
	if gd := s.gameData(); gd != nil {
		return gd.SelectedCase
	}

	return 0
}

func (s *GameScene) setSelectedCase(i int) {
	if gd := s.gameData(); gd != nil {
		gd.SelectedCase = i
	}
}

func (s *GameScene) getSelectedPuzzle() int {
	if gd := s.gameData(); gd != nil {
		return gd.SelectedPuzzle
	}

	return 0
}

func (s *GameScene) setSelectedPuzzle(i int) {
	if gd := s.gameData(); gd != nil {
		gd.SelectedPuzzle = i
	}
}

func (s *GameScene) getPuzzleSeed() uint64 {
	if gd := s.gameData(); gd != nil {
		return gd.PuzzleSeed
	}

	return 0
}

func (s *GameScene) setPuzzleSeed(seed uint64) {
	if gd := s.gameData(); gd != nil {
		gd.PuzzleSeed = seed
	}
}

func (s *GameScene) getAssetsDir() string {
	if gd := s.gameData(); gd != nil {
		return gd.AssetsDir
	}

	return ""
}

func (s *GameScene) GetScaleX() float64  { return s.scaleX }
func (s *GameScene) GetScaleY() float64  { return s.scaleY }
func (s *GameScene) GetOffsetX() float64 { return s.offsetX }

func (s *GameScene) SetScale(scaleX, scaleY, offsetX float64) {
	s.scaleX = scaleX
	s.scaleY = scaleY
	s.offsetX = offsetX
}

func (s *GameScene) GetScreenSize() (int, int) { return s.width, s.height }

func (s *GameScene) SetCursorPos(x, y int) {
	// Written by CursorSystem, cursor entity is the source of truth
}

func (s *GameScene) GetCursorPos() (int, int) {
	val, err := s.Registry.Get(c.GroupCursor, 0)
	if err != nil {
		return 0, 0
	}

	if entity, ok := val.(*c.Entity); ok && entity.Cursor != nil {
		return entity.Cursor.X, entity.Cursor.Y
	}

	return 0, 0
}

func (s *GameScene) GetCasesScroll() int {
	if gd := s.gameData(); gd != nil {
		return gd.CasesScroll
	}

	return 0
}

func (s *GameScene) SetCasesScroll(v int) {
	if gd := s.gameData(); gd != nil {
		gd.CasesScroll = v
	}
}

func (s *GameScene) GetNamesScroll() int {
	if gd := s.gameData(); gd != nil {
		return gd.NamesScroll
	}

	return 0
}

func (s *GameScene) SetNamesScroll(v int) {
	if gd := s.gameData(); gd != nil {
		gd.NamesScroll = v
	}
}

func (s *GameScene) GetDescScroll() int {
	if gd := s.gameData(); gd != nil {
		return gd.DescScroll
	}

	return 0
}

func (s *GameScene) SetDescScroll(v int) {
	if gd := s.gameData(); gd != nil {
		gd.DescScroll = v
	}
}

func (s *GameScene) GetHoldingPiece() int {
	if gd := s.gameData(); gd != nil {
		return gd.HoldingPiece
	}

	return -1
}

func (s *GameScene) SetHoldingPiece(v int) {
	if gd := s.gameData(); gd != nil {
		gd.HoldingPiece = v
	}
}

func (s *GameScene) GetDragging() bool {
	if gd := s.gameData(); gd != nil {
		return gd.Dragging
	}

	return false
}

func (s *GameScene) SetDragging(v bool) {
	if gd := s.gameData(); gd != nil {
		gd.Dragging = v
	}
}

func (s *GameScene) GetShowResult() int {
	if gd := s.gameData(); gd != nil {
		return gd.ShowResult
	}

	return 0
}

func (s *GameScene) SetShowResult(v int) {
	if gd := s.gameData(); gd != nil {
		gd.ShowResult = v
	}
}

func (s *GameScene) GetResultTick() int {
	if gd := s.gameData(); gd != nil {
		return gd.ResultTick
	}

	return 0
}

func (s *GameScene) SetResultTick(v int) {
	if gd := s.gameData(); gd != nil {
		gd.ResultTick = v
	}
}

// --- Game state persistence ---

func (s *GameScene) LoadGameState() {
	gd := s.gameData()
	if gd == nil || gd.Cases == nil {
		return
	}

	save, err := domain.LoadGame(domain.DefaultSavePath())
	if err != nil {
		return
	}

	for _, cs := range save.Cases {
		if cs.CaseIndex < 0 || cs.CaseIndex >= len(gd.Cases) {
			continue
		}

		caseConfig := gd.Cases[cs.CaseIndex]

		for pi, ps := range cs.Puzzles {
			if pi >= len(caseConfig.Puzzles) {
				break
			}

			p := caseConfig.Puzzles[pi]
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

		caseConfig.ID = cs.CaseIndex + 1
	}

	if gd.SelectedCase >= 0 && gd.SelectedCase < len(gd.Cases) {
		gd.SelectedPuzzle = firstUnsolvedPuzzle(gd.Cases[gd.SelectedCase])
	}

	slog.Info("game state loaded")
}

func (s *GameScene) SaveGameState() {
	gd := s.gameData()
	if gd == nil || gd.Cases == nil {
		return
	}

	save := &domain.GameSave{}

	for i, caseConfig := range gd.Cases {
		cs := domain.CaseSave{
			CaseIndex:    i,
			ActivePuzzle: firstUnsolvedPuzzle(caseConfig),
		}

		for _, p := range caseConfig.Puzzles {
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

// --- Zone registration (called by StateSystem on transitions) ---

func (s *GameScene) RegisterEnabledZones() {
	s.input.ClearZones()
	co := s.coords()

	og := s.tmap.FindObjectGroup("enabled")
	if og == nil {
		return
	}

	for _, obj := range og.Objects {
		scaledSpatial := s.scaleBox(obj)

		switch obj.Name {
		case "button-run-fingerprint":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					slog.Info("opening fingerprint app")
					s.setECSState(c.StateApplicationLayout)
					s.RegisterAppLayoutZones()
				},
			})
		case "button-quit-os":
			qx, qy := co.MapToScreenX(obj.X), co.MapToScreenY(obj.Y)
			qw, qh := co.MapToScreenSize(200), co.MapToScreenSize(50)

			s.input.AddZone(&systems.Zone{
				Spatial: shapes.NewBox(shapes.NewPoint(qx, qy), qw, qh),
				OnClick: func() {
					ebiten.SetCursorMode(ebiten.CursorModeVisible)
					os.Exit(0)
				},
			})
		}
	}
}

func (s *GameScene) RegisterAppLayoutZones() {
	s.input.ClearZones()
	co := s.coords()

	og := s.tmap.FindObjectGroup("application-layout")
	if og == nil {
		return
	}

	for _, obj := range og.Objects {
		scaledSpatial := s.scaleBox(obj)

		switch obj.Name {
		case "exit":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					s.setECSState(c.StateEnabled)
					s.RegisterEnabledZones()
				},
			})
		case "play-puzzle":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					s.EnsureCurrentPuzzleImages()
					s.InitTrayPositions()
					s.setECSState(c.StateApplicationNet)

					if gd := s.gameData(); gd != nil {
						gd.HoldingPiece = -1
						gd.Dragging = false
					}

					s.RegisterPuzzleZones()
				},
			})
		case "regenerate-puzzles":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					s.regenerateCases()
				},
			})
		case "list-of-cases":
			objCopy := obj

			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					gd := s.gameData()
					if gd == nil {
						return
					}

					cx, _ := s.GetCursorPos()
					_ = cx
					_, cy := s.GetCursorPos()
					relY := float64(cy) - co.MapToScreenY(objCopy.Y)
					rowH := co.MapToScreenSize(45)
					idx := int(relY/rowH) + gd.CasesScroll

					if idx >= 0 && idx < len(gd.Cases) {
						gd.SelectedCase = idx
						gd.SelectedPuzzle = firstUnsolvedPuzzle(gd.Cases[idx])
						gd.NamesScroll = 0
						gd.DescScroll = 0
					}
				},
			})
		case "fingerprints-user-names":
			objCopy := obj

			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					gd := s.gameData()
					if gd == nil {
						return
					}

					_, cy := s.GetCursorPos()
					relY := float64(cy) - co.MapToScreenY(objCopy.Y)
					rowH := co.MapToScreenSize(50)
					idx := int(relY/rowH) + gd.NamesScroll

					if gd.SelectedCase >= 0 && gd.SelectedCase < len(gd.Cases) {
						cs := gd.Cases[gd.SelectedCase]
						if idx >= 0 && idx < len(cs.Puzzles) {
							gd.SelectedPuzzle = idx
							gd.DescScroll = 0
						}
					}
				},
			})
		}
	}
}

func (s *GameScene) RegisterPuzzleZones() {
	s.input.ClearZones()

	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return
	}

	puzzle := s.currentPuzzle()
	if puzzle == nil {
		return
	}

	for _, obj := range og.Objects {
		scaledSpatial := s.scaleBox(obj)

		switch obj.Name {
		case "back":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					s.setECSState(c.StateApplicationLayout)
					s.RegisterAppLayoutZones()
				},
			})
		case "exit":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					s.setECSState(c.StateEnabled)
					s.RegisterEnabledZones()
				},
			})
		case "button-send-puzzle":
			s.input.AddZone(&systems.Zone{
				Spatial: scaledSpatial,
				OnClick: func() {
					s.submitPuzzle()
				},
			})
		}
	}
}

// --- Puzzle logic ---

func (s *GameScene) currentPuzzle() *domain.PuzzleConfig {
	gd := s.gameData()
	if gd == nil || gd.Cases == nil {
		return nil
	}

	if gd.SelectedCase < 0 || gd.SelectedCase >= len(gd.Cases) {
		return nil
	}

	cs := gd.Cases[gd.SelectedCase]
	if gd.SelectedPuzzle < 0 || gd.SelectedPuzzle >= len(cs.Puzzles) {
		return nil
	}

	return cs.Puzzles[gd.SelectedPuzzle]
}

func (s *GameScene) submitPuzzle() {
	p := s.currentPuzzle()
	if p == nil {
		return
	}

	gd := s.gameData()
	if gd == nil {
		return
	}

	var found *domain.FingerprintRecord

	if p.HideColor {
		hashNum := s.computeCurrentHashNum(p)

		for _, letter := range []string{"G", "R", "Y", "B"} {
			candidate := fmt.Sprintf("%s%d", letter, hashNum)
			if rec := gd.DB.LookupByHash(candidate); rec != nil {
				found = rec
				p.HideColor = false

				slog.Info("SUBMIT: color revealed!", "color", rec.Color)

				break
			}
		}
	} else {
		currentHash := s.computeCurrentHash(p)
		found = gd.DB.LookupByHash(currentHash)
	}

	if found != nil {
		slog.Info("SUBMIT: person found!", "name", found.PersonName)

		p.Solved = true
		p.Failed = false
		gd.ShowResult = 1
	} else {
		slog.Info("SUBMIT: no match")

		p.Failed = true
		gd.ShowResult = 2
	}

	gd.ResultTick = 180
	s.SaveGameState()
}

func (s *GameScene) computeCurrentHash(p *domain.PuzzleConfig) string {
	pieces := s.buildPieceGrid(p)
	colorLetter := domain.ColorLetter(p.TargetRecord.Color)

	if p.HideColor {
		colorLetter = "?"
	}

	return fmt.Sprintf("%s%d", colorLetter, domain.ComputeHash(pieces))
}

func (s *GameScene) computeCurrentHashNum(p *domain.PuzzleConfig) uint64 {
	return domain.ComputeHash(s.buildPieceGrid(p))
}

func (s *GameScene) buildPieceGrid(p *domain.PuzzleConfig) []domain.PieceRecord {
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

	return pieces
}

func (s *GameScene) regenerateCases() {
	gd := s.gameData()
	if gd == nil || gd.DB == nil {
		return
	}

	gd.PuzzleSeed = gd.PuzzleSeed ^ 0xFACE
	gd.Cases = domain.GenerateCases(gd.DB, gd.PuzzleSeed)
	gd.SelectedCase = 0
	gd.SelectedPuzzle = 0
	gd.HoldingPiece = -1
	gd.NamesScroll = 0
	gd.DescScroll = 0

	if err := domain.SavePuzzles(gd.Cases, gd.PuzzleSeed, domain.DefaultPuzzlesPath()); err != nil {
		slog.Warn("save puzzles", "error", err)
	}

	s.RegisterAppLayoutZones()
	slog.Info("puzzles regenerated", "seed", gd.PuzzleSeed)
}

// --- Image access (lazy-loaded) ---

func (s *GameScene) EnsureCurrentPuzzleImages() {
	p := s.currentPuzzle()
	if p == nil {
		return
	}

	assetsDir := s.getAssetsDir()
	if assetsDir == "" {
		return
	}

	rec := p.TargetRecord

	if _, ok := s.targetImages[rec.ID]; !ok {
		imgs, err := LoadFingerprintImages(assetsDir, rec)
		if err != nil {
			slog.Warn("load target", "id", rec.ID, "error", err)
		} else {
			s.targetImages[rec.ID] = imgs
		}
	}

	if p.HideColor {
		if _, ok := s.targetGreyImages[rec.ID]; !ok {
			greyImgs, err := LoadGreyFingerprintImages(assetsDir, rec.Variant, rec.Rotation, rec.Mirrored)
			if err != nil {
				slog.Warn("load grey target", "id", rec.ID, "error", err)
			} else {
				s.targetGreyImages[rec.ID] = greyImgs
			}
		}
	}
}

func (s *GameScene) InitTrayPositions() {
	p := s.currentPuzzle()
	if p == nil || s.tmap == nil {
		return
	}

	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return
	}

	var rects []struct{ x, y, w, h float64 }

	for _, obj := range og.Objects {
		if obj.Name == "pieces" {
			rects = append(rects, struct{ x, y, w, h float64 }{x: obj.X, y: obj.Y, w: obj.Width, h: obj.Height})
		}
	}

	if len(rects) == 0 {
		return
	}

	cellMap := 68.0

	for _, obj := range og.Objects {
		if obj.Name == "puzzle" {
			side := obj.Width
			if obj.Height < side {
				side = obj.Height
			}

			cellMap = side / 10

			break
		}
	}

	for i := range p.TrayPieces {
		tp := &p.TrayPieces[i]
		if tp.IsPlaced || (tp.TrayX != 0 && tp.TrayY != 0) {
			continue
		}

		r := rects[i%len(rects)]
		tp.TrayX = r.x + float64(i%3)*cellMap + float64(i%2)*10
		tp.TrayY = r.y + float64(i/3)*cellMap + float64(i%2)*5

		if tp.TrayX+cellMap > r.x+r.w {
			tp.TrayX = r.x + r.w - cellMap
		}

		if tp.TrayY+cellMap > r.y+r.h {
			tp.TrayY = r.y + r.h - cellMap
		}
	}
}

func (s *GameScene) GetTargetImages(recordID int) *ebiten.Image {
	s.ensureTargetImages(recordID)

	if imgs, ok := s.targetImages[recordID]; ok && imgs != nil {
		return imgs.Full
	}

	return nil
}

func (s *GameScene) GetTargetPieceImage(recordID, pieceIdx int) *ebiten.Image {
	s.ensureTargetImages(recordID)

	if imgs, ok := s.targetImages[recordID]; ok && imgs != nil && pieceIdx >= 0 && pieceIdx < 100 {
		return imgs.Pieces[pieceIdx]
	}

	return nil
}

func (s *GameScene) GetGreyPieceImage(recordID, pieceIdx int) *ebiten.Image {
	s.ensureGreyImages(recordID)

	if imgs, ok := s.targetGreyImages[recordID]; ok && imgs != nil && pieceIdx >= 0 && pieceIdx < 100 {
		return imgs.Pieces[pieceIdx]
	}

	return nil
}

func (s *GameScene) GetDecoyPieceImage(clr string, variant, rotation int, mirrored bool, pieceIdx int) *ebiten.Image {
	key := fmt.Sprintf("%s.%d.r%d.m%v", clr, variant, rotation, mirrored)

	if _, ok := s.allImages[key]; !ok {
		assetsDir := s.getAssetsDir()
		if assetsDir == "" {
			return nil
		}

		rec := &domain.FingerprintRecord{Color: clr, Variant: variant, Rotation: rotation, Mirrored: mirrored}

		imgs, err := LoadFingerprintImages(assetsDir, rec)
		if err != nil {
			slog.Warn("lazy load decoy", "key", key, "error", err)
			s.allImages[key] = nil
		} else {
			s.allImages[key] = imgs
		}
	}

	if imgs := s.allImages[key]; imgs != nil && pieceIdx >= 0 && pieceIdx < 100 {
		return imgs.Pieces[pieceIdx]
	}

	return nil
}

func (s *GameScene) GetAvatarImage(filename string) *ebiten.Image {
	if img, ok := s.avatarCache[filename]; ok {
		return img
	}

	assetsDir := s.getAssetsDir()
	if assetsDir == "" {
		return nil
	}

	path := filepath.Join(assetsDir, "avatars", filename)

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

// --- Internal helpers ---

func (s *GameScene) ensureTargetImages(recordID int) {
	if _, ok := s.targetImages[recordID]; ok {
		return
	}

	assetsDir := s.getAssetsDir()
	db := s.getDB()

	if assetsDir == "" || db == nil {
		return
	}

	for i := range db.Records {
		rec := &db.Records[i]
		if rec.ID == recordID {
			imgs, err := LoadFingerprintImages(assetsDir, rec)
			if err != nil {
				slog.Warn("load target", "id", recordID, "error", err)
				s.targetImages[recordID] = nil
			} else {
				s.targetImages[recordID] = imgs
			}

			return
		}
	}
}

func (s *GameScene) ensureGreyImages(recordID int) {
	if _, ok := s.targetGreyImages[recordID]; ok {
		return
	}

	assetsDir := s.getAssetsDir()
	db := s.getDB()

	if assetsDir == "" || db == nil {
		return
	}

	for i := range db.Records {
		rec := &db.Records[i]
		if rec.ID == recordID {
			greyImgs, err := LoadGreyFingerprintImages(assetsDir, rec.Variant, rec.Rotation, rec.Mirrored)
			if err != nil {
				slog.Warn("load grey", "id", recordID, "error", err)
				s.targetGreyImages[recordID] = nil
			} else {
				s.targetGreyImages[recordID] = greyImgs
			}

			return
		}
	}
}

func firstUnsolvedPuzzle(cs *domain.CaseConfig) int {
	for i, p := range cs.Puzzles {
		if !p.Solved && !p.Failed {
			return i
		}
	}

	return 0
}

func (s *GameScene) scaleBox(obj *tiled.Object) shapes.Spatial { //nolint:ireturn // spatial for RTree
	sx, sy, sw, sh := s.coords().MapRect(obj.X, obj.Y, obj.Width, obj.Height)

	return shapes.NewBox(shapes.NewPoint(sx, sy), sw, sh)
}

func (s *GameScene) coords() fsystems.Coords {
	return fsystems.Coords{Scale: s.scaleX, OffsetX: s.offsetX}
}

func (s *GameScene) setECSState(state c.GameState) {
	val, err := s.Registry.Get(c.GroupGameState, 0)
	if err != nil {
		return
	}

	if entity, ok := val.(*c.Entity); ok && entity.State != nil {
		entity.State.Current = state
	}
}

// FindTMXPath locates the fingerprint.tmx file.
func FindTMXPath() string {
	candidates := []string{
		"assets/external/fingerprint/fingerprint.tmx",
		"../assets/external/fingerprint/fingerprint.tmx",
		"../../assets/external/fingerprint/fingerprint.tmx",
	}

	for _, p := range candidates {
		if info, err := statFile(p); err == nil && info != nil {
			return p
		}
	}

	return ""
}

// FindFingerprintAssetsDir finds the fingerprints directory.
func FindFingerprintAssetsDir() string {
	candidates := []string{
		"assets/external/fingerprint",
		"../assets/external/fingerprint",
		"../../assets/external/fingerprint",
	}

	for _, p := range candidates {
		if info, err := statFile(filepath.Join(p, "fingerprints")); err == nil && info != nil {
			return p
		}
	}

	return ""
}
