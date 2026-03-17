package scenes

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/InsideGallery/core/memory/registry"
	"github.com/InsideGallery/game-core/geometry/shapes"
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

const GameSceneName = "fingerprint_game"

// GameScene is the ECS-driven scene for the fingerprint game.
// Systems draw in world (map) coordinates to the World offscreen image.
// A base transform (scale + center) maps World to screen.
// Camera provides additional user zoom on top of the base transform.
type GameScene struct {
	*scene.BaseScene

	input *systems.InputSystem
	tmap  *tilemap.Map

	// Image caches
	targetImages     map[int]*FingerprintImages
	targetGreyImages map[int]*FingerprintImages
	allImages        map[string]*FingerprintImages
	avatarCache      map[string]*ebiten.Image

	width, height int

	// Base transform: world → screen (centered on TMX camera point)
	baseScale float64 // uniform scale factor
	baseOffX  float64 // X offset to center camera point on screen
	baseOffY  float64 // Y offset to center camera point on screen
}

func NewGameScene() *GameScene {
	return &GameScene{}
}

func (s *GameScene) Name() string { return GameSceneName }

func (s *GameScene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
	s.input = systems.NewInputSystem(s.RTree)

	// Create game state + cursor entities
	gameEntity := &c.Entity{
		State:    &c.State{Current: c.StateLoading},
		GameData: &c.GameData{HoldingPiece: -1, AssetsDir: FindFingerprintAssetsDir()},
	}
	_ = s.Registry.Add(c.GroupGameState, 0, gameEntity)

	cursorEntity := &c.Entity{Cursor: &c.Cursor{WorldMaxX: 4000, WorldMaxY: 2176}}
	_ = s.Registry.Add(c.GroupCursor, 0, cursorEntity)

	// Register ECS systems
	dragDrop := fsystems.NewDragDropSystem(s)
	render := fsystems.NewRenderSystem(s, dragDrop)

	s.Systems.Add("state", fsystems.NewStateSystem(s))
	s.Systems.Add("cursor", fsystems.NewCursorSystem(s))
	s.Systems.Add("input", s.input)
	s.Systems.Add("scroll", fsystems.NewScrollSystem(s))
	s.Systems.Add("dragdrop", dragDrop)
	s.Systems.Add("camera", fsystems.NewCameraSystem(s))
	s.Systems.Add("render", render)
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

func (s *GameScene) Update() error {
	return s.BaseScene.Update()
}

// Draw: systems draw to World in map coords, then we composite to screen
// with base transform (scale+center) and camera zoom.
func (s *GameScene) Draw(screen *ebiten.Image) {
	sysList := s.Systems.Get()

	if s.World == nil {
		// No world yet (loading) — systems draw to screen directly
		for _, sys := range sysList {
			sys.Draw(s.Ctx, screen)
		}

		for _, sys := range sysList {
			if w, ok := sys.(core.SystemWindow); ok {
				w.ScreenDraw(s.Ctx, screen)
			}
		}

		return
	}

	// Systems draw to World (map coordinates)
	s.World.Clear()

	for _, sys := range sysList {
		sys.Draw(s.Ctx, s.World)
	}

	// Composite World to screen: base transform + camera zoom
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(s.baseScale, s.baseScale)
	op.GeoM.Translate(s.baseOffX, s.baseOffY)

	// Camera zoom on top: zoom around screen center
	if s.Camera.ZoomFactor != 0 {
		zoomScale := math.Pow(1.01, s.Camera.ZoomFactor)
		cx := float64(s.width) / 2
		cy := float64(s.height) / 2
		op.GeoM.Translate(-cx, -cy)
		op.GeoM.Scale(zoomScale, zoomScale)
		op.GeoM.Translate(cx, cy)
	}

	screen.DrawImage(s.World, op)

	// ScreenDraw overlays (cursor) — screen space, no transform
	for _, sys := range sysList {
		if w, ok := sys.(core.SystemWindow); ok {
			w.ScreenDraw(s.Ctx, screen)
		}
	}
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
		s.updateBaseTransform()
	}

	return w, h
}

// updateBaseTransform computes the base scale and offset.
// Centers on the TMX "camera" point if present, otherwise map center.
func (s *GameScene) updateBaseTransform() {
	if s.tmap == nil || s.height == 0 {
		return
	}

	mapH := float64(s.tmap.MapPixelHeight())
	s.baseScale = float64(s.height) / mapH

	// Find camera center from TMX (or default to map center)
	camX := float64(s.tmap.MapPixelWidth()) / 2
	camY := mapH / 2

	mainOG := s.tmap.FindObjectGroup("main")
	if mainOG != nil {
		if camObj := tilemap.FindObject(mainOG, "camera"); camObj != nil {
			camX = camObj.X
			camY = camObj.Y
		}
	}

	// Offset so that TMX camera point maps to screen center
	s.baseOffX = float64(s.width)/2 - camX*s.baseScale
	s.baseOffY = float64(s.height)/2 - camY*s.baseScale
}

// SetupWorld creates World offscreen image and computes base transform + cursor room.
func (s *GameScene) SetupWorld() {
	if s.tmap == nil {
		return
	}

	s.World = ebiten.NewImage(s.tmap.MapPixelWidth(), s.tmap.MapPixelHeight())
	s.updateBaseTransform()

	// Update cursor room bounds in screen space
	mainOG := s.tmap.FindObjectGroup("main")
	if mainOG == nil {
		return
	}

	room := tilemap.FindObject(mainOG, "cursor-room")
	if room == nil {
		return
	}

	cur := fsystems.GetCursor(s.Registry)
	if cur != nil {
		// Store in world (map) coords — CursorSystem clamps in world space
		cur.WorldMinX = room.X
		cur.WorldMinY = room.Y
		cur.WorldMaxX = room.X + room.Width
		cur.WorldMaxY = room.Y + room.Height
	}
}

// ScreenToWorld converts screen coordinates to world (map) coordinates.
func (s *GameScene) ScreenToWorld(screenX, screenY float64) (float64, float64) {
	// Undo camera zoom
	if s.Camera.ZoomFactor != 0 {
		zoomScale := math.Pow(1.01, s.Camera.ZoomFactor)
		cx := float64(s.width) / 2
		cy := float64(s.height) / 2
		screenX = (screenX-cx)/zoomScale + cx
		screenY = (screenY-cy)/zoomScale + cy
	}

	// Undo base transform
	return (screenX - s.baseOffX) / s.baseScale, (screenY - s.baseOffY) / s.baseScale
}

func (s *GameScene) GetBaseZoom() float64 { return 0 }
func (s *GameScene) ResetCameraZoom()     { s.Camera.ZoomFactor = 0 }

// --- SceneAccessor implementation ---

func (s *GameScene) gameData() *c.GameData                                { return fsystems.GetGameData(s.Registry) }
func (s *GameScene) GetRegistry() *registry.Registry[string, uint64, any] { return s.Registry }
func (s *GameScene) GetCamera() *core.Camera                              { return s.Camera }
func (s *GameScene) GetInputSystem() *systems.InputSystem                 { return s.input }
func (s *GameScene) GetTileMap() *tilemap.Map                             { return s.tmap }
func (s *GameScene) SetTileMap(m *tilemap.Map)                            { s.tmap = m }
func (s *GameScene) GetScreenSize() (int, int)                            { return s.width, s.height }
func (s *GameScene) SetCursorPos(_, _ int)                                {}
func (s *GameScene) RequestQuit()                                         { QuitGame() }

// scaleBox converts TMX object from world to screen coords for RTree.
func (s *GameScene) scaleBox(obj *tiled.Object) shapes.Spatial { //nolint:ireturn
	return shapes.NewBox(
		shapes.NewPoint(obj.X*s.baseScale+s.baseOffX, obj.Y*s.baseScale+s.baseOffY),
		obj.Width*s.baseScale, obj.Height*s.baseScale,
	)
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

// --- Zone registration ---

func (s *GameScene) RegisterEnabledZones() {
	s.input.ClearZones()

	if s.tmap == nil {
		return
	}

	og := s.tmap.FindObjectGroup("enabled")
	if og == nil {
		return
	}

	for _, obj := range og.Objects {
		scaled := s.scaleBox(obj)

		switch obj.Name {
		case "button-run-fingerprint":
			s.input.AddZone(&systems.Zone{
				Spatial: scaled,
				OnClick: func() {
					s.setECSState(c.StateApplicationLayout)
					s.RegisterAppLayoutZones()
				},
			})
		case "button-quit-os":
			s.input.AddZone(&systems.Zone{
				Spatial: shapes.NewBox(
					shapes.NewPoint(obj.X*s.baseScale+s.baseOffX, obj.Y*s.baseScale+s.baseOffY),
					200*s.baseScale, 50*s.baseScale,
				),
				OnClick: func() {
					ebiten.SetCursorMode(ebiten.CursorModeVisible)
					QuitGame()
				},
			})
		}
	}
}

func (s *GameScene) RegisterAppLayoutZones() {
	s.input.ClearZones()

	if s.tmap == nil {
		return
	}

	og := s.tmap.FindObjectGroup("application-layout")
	if og == nil {
		return
	}

	for _, obj := range og.Objects {
		scaled := s.scaleBox(obj)

		switch obj.Name {
		case "exit":
			s.input.AddZone(&systems.Zone{Spatial: scaled, OnClick: func() {
				s.setECSState(c.StateEnabled)
				s.RegisterEnabledZones()
			}})
		case "play-puzzle":
			s.input.AddZone(&systems.Zone{Spatial: scaled, OnClick: func() {
				s.EnsureCurrentPuzzleImages()
				s.InitTrayPositions()
				s.setECSState(c.StateApplicationNet)

				if gd := s.gameData(); gd != nil {
					gd.HoldingPiece = -1
					gd.Dragging = false
				}

				s.RegisterPuzzleZones()
			}})
		case "regenerate-puzzles":
			s.input.AddZone(&systems.Zone{Spatial: scaled, OnClick: func() { s.regenerateCases() }})
		case "list-of-cases":
			objCopy := obj
			s.input.AddZone(&systems.Zone{Spatial: scaled, OnClick: func() {
				gd := s.gameData()
				cur := fsystems.GetCursor(s.Registry)

				if gd == nil || cur == nil {
					return
				}

				_, wy := s.ScreenToWorld(float64(cur.X), float64(cur.Y))
				idx := int((wy-objCopy.Y)/90) + gd.CasesScroll

				if idx >= 0 && idx < len(gd.Cases) {
					gd.SelectedCase = idx
					gd.SelectedPuzzle = firstUnsolvedPuzzle(gd.Cases[idx])
					gd.NamesScroll = 0
					gd.DescScroll = 0
				}
			}})
		case "fingerprints-user-names":
			objCopy := obj
			s.input.AddZone(&systems.Zone{Spatial: scaled, OnClick: func() {
				gd := s.gameData()
				cur := fsystems.GetCursor(s.Registry)

				if gd == nil || cur == nil {
					return
				}

				_, wy := s.ScreenToWorld(float64(cur.X), float64(cur.Y))
				idx := int((wy-objCopy.Y)/100) + gd.NamesScroll

				if gd.SelectedCase >= 0 && gd.SelectedCase < len(gd.Cases) {
					cs := gd.Cases[gd.SelectedCase]
					if idx >= 0 && idx < len(cs.Puzzles) {
						gd.SelectedPuzzle = idx
						gd.DescScroll = 0
					}
				}
			}})
		}
	}
}

func (s *GameScene) RegisterPuzzleZones() {
	s.input.ClearZones()

	if s.tmap == nil {
		return
	}

	og := s.tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return
	}

	for _, obj := range og.Objects {
		scaled := s.scaleBox(obj)

		switch obj.Name {
		case "back":
			s.input.AddZone(&systems.Zone{Spatial: scaled, OnClick: func() {
				s.setECSState(c.StateApplicationLayout)
				s.RegisterAppLayoutZones()
			}})
		case "exit":
			s.input.AddZone(&systems.Zone{Spatial: scaled, OnClick: func() {
				s.setECSState(c.StateEnabled)
				s.RegisterEnabledZones()
			}})
		case "button-send-puzzle":
			s.input.AddZone(&systems.Zone{Spatial: scaled, OnClick: func() { s.submitPuzzle() }})
		}
	}
}

// --- Puzzle logic ---

func (s *GameScene) currentPuzzle() *domain.PuzzleConfig {
	return fsystems.CurrentPuzzle(s.gameData())
}

func (s *GameScene) submitPuzzle() {
	p := s.currentPuzzle()
	gd := s.gameData()

	if p == nil || gd == nil {
		return
	}

	var found *domain.FingerprintRecord

	if p.HideColor {
		hashNum := domain.ComputeHash(s.buildPieceGrid(p))

		for _, letter := range []string{"G", "R", "Y", "B"} {
			if rec := gd.DB.LookupByHash(fmt.Sprintf("%s%d", letter, hashNum)); rec != nil {
				found = rec
				p.HideColor = false

				break
			}
		}
	} else {
		pieces := s.buildPieceGrid(p)
		hash := fmt.Sprintf("%s%d", domain.ColorLetter(p.TargetRecord.Color), domain.ComputeHash(pieces))
		found = gd.DB.LookupByHash(hash)
	}

	if found != nil {
		p.Solved = true
		p.Failed = false
		gd.ShowResult = 1
	} else {
		p.Failed = true
		gd.ShowResult = 2
	}

	gd.ResultTick = 180
	s.SaveGameState()
}

func (s *GameScene) buildPieceGrid(p *domain.PuzzleConfig) []domain.PieceRecord {
	pieces := make([]domain.PieceRecord, 100)
	copy(pieces, p.TargetRecord.Pieces)

	for _, idx := range p.MissingIndices {
		pieces[idx] = domain.PieceRecord{X: idx % 10, Y: idx / 10}
	}

	for _, tp := range p.TrayPieces {
		if tp.IsPlaced {
			gIdx := tp.PlacedY*10 + tp.PlacedX
			if gIdx >= 0 && gIdx < 100 {
				pieces[gIdx] = domain.PieceRecord{X: tp.PlacedX, Y: tp.PlacedY, Value: tp.Value}
			}
		}
	}

	return pieces
}

func (s *GameScene) regenerateCases() {
	gd := s.gameData()
	if gd == nil || gd.DB == nil {
		return
	}

	gd.PuzzleSeed ^= 0xFACE
	gd.Cases = domain.GenerateCases(gd.DB, gd.PuzzleSeed)
	gd.SelectedCase = 0
	gd.SelectedPuzzle = 0
	gd.HoldingPiece = -1
	gd.NamesScroll = 0
	gd.DescScroll = 0

	_ = domain.SavePuzzles(gd.Cases, gd.PuzzleSeed, domain.DefaultPuzzlesPath())
	s.RegisterAppLayoutZones()
}

// --- Persistence ---

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

		for pi, ps := range cs.Puzzles {
			if pi >= len(gd.Cases[cs.CaseIndex].Puzzles) {
				break
			}

			p := gd.Cases[cs.CaseIndex].Puzzles[pi]
			p.Solved = ps.Solved
			p.Failed = ps.Failed

			for _, pp := range ps.PlacedPieces {
				if pp.TrayIndex >= 0 && pp.TrayIndex < len(p.TrayPieces) {
					tp := &p.TrayPieces[pp.TrayIndex]
					tp.IsPlaced = pp.GridX >= 0 && pp.GridY >= 0
					tp.PlacedX = pp.GridX
					tp.PlacedY = pp.GridY
					tp.Rotation = pp.Rotation
					tp.TrayX = pp.TrayX
					tp.TrayY = pp.TrayY
				}
			}
		}
	}

	if gd.SelectedCase >= 0 && gd.SelectedCase < len(gd.Cases) {
		gd.SelectedPuzzle = firstUnsolvedPuzzle(gd.Cases[gd.SelectedCase])
	}
}

func (s *GameScene) SaveGameState() {
	gd := s.gameData()
	if gd == nil || gd.Cases == nil {
		return
	}

	save := &domain.GameSave{}

	for i, cs := range gd.Cases {
		caseSave := domain.CaseSave{CaseIndex: i, ActivePuzzle: firstUnsolvedPuzzle(cs)}

		for _, p := range cs.Puzzles {
			ps := domain.PuzzleSave{Solved: p.Solved, Failed: p.Failed}

			for j, tp := range p.TrayPieces {
				if tp.IsPlaced || tp.TrayX != 0 || tp.TrayY != 0 || tp.Rotation != 0 {
					ps.PlacedPieces = append(ps.PlacedPieces, domain.PlacedSave{
						TrayIndex: j, GridX: tp.PlacedX, GridY: tp.PlacedY,
						Rotation: tp.Rotation, TrayX: tp.TrayX, TrayY: tp.TrayY,
					})
				}
			}

			caseSave.Puzzles = append(caseSave.Puzzles, ps)
		}

		save.Cases = append(save.Cases, caseSave)
	}

	_ = domain.SaveGame(save, domain.DefaultSavePath())
}

// --- Image access ---

func (s *GameScene) EnsureCurrentPuzzleImages() {
	p := s.currentPuzzle()
	gd := s.gameData()

	if p == nil || gd == nil || gd.AssetsDir == "" {
		return
	}

	rec := p.TargetRecord

	if _, ok := s.targetImages[rec.ID]; !ok {
		imgs, _ := LoadFingerprintImages(gd.AssetsDir, rec)
		s.targetImages[rec.ID] = imgs
	}

	if p.HideColor {
		if _, ok := s.targetGreyImages[rec.ID]; !ok {
			imgs, _ := LoadGreyFingerprintImages(gd.AssetsDir, rec.Variant, rec.Rotation, rec.Mirrored)
			s.targetGreyImages[rec.ID] = imgs
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
			rects = append(rects, struct{ x, y, w, h float64 }{obj.X, obj.Y, obj.Width, obj.Height})
		}
	}

	if len(rects) == 0 {
		return
	}

	cellMap := 68.0

	for _, obj := range og.Objects {
		if obj.Name == "puzzle" {
			cellMap = math.Min(obj.Width, obj.Height) / 10

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

func (s *GameScene) GetTargetPieceImage(recordID, pieceIdx int) *ebiten.Image {
	s.ensureTargetLoaded(recordID)

	if imgs := s.targetImages[recordID]; imgs != nil && pieceIdx >= 0 && pieceIdx < 100 {
		return imgs.Pieces[pieceIdx]
	}

	return nil
}

func (s *GameScene) GetGreyPieceImage(recordID, pieceIdx int) *ebiten.Image {
	s.ensureGreyLoaded(recordID)

	if imgs := s.targetGreyImages[recordID]; imgs != nil && pieceIdx >= 0 && pieceIdx < 100 {
		return imgs.Pieces[pieceIdx]
	}

	return nil
}

func (s *GameScene) GetDecoyPieceImage(clr string, variant, rotation int, mirrored bool, pieceIdx int) *ebiten.Image {
	key := fmt.Sprintf("%s.%d.r%d.m%v", clr, variant, rotation, mirrored)

	if _, ok := s.allImages[key]; !ok {
		gd := s.gameData()
		if gd == nil || gd.AssetsDir == "" {
			return nil
		}

		imgs, _ := LoadFingerprintImages(gd.AssetsDir, &domain.FingerprintRecord{
			Color: clr, Variant: variant, Rotation: rotation, Mirrored: mirrored,
		})
		s.allImages[key] = imgs
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

	gd := s.gameData()
	if gd == nil || gd.AssetsDir == "" {
		return nil
	}

	img, err := loadStdImage(filepath.Join(gd.AssetsDir, "avatars", filename))
	if err != nil {
		s.avatarCache[filename] = nil

		return nil
	}

	eImg := ebiten.NewImageFromImage(img)
	s.avatarCache[filename] = eImg

	return eImg
}

func (s *GameScene) ensureTargetLoaded(id int) {
	if _, ok := s.targetImages[id]; ok {
		return
	}

	gd := s.gameData()
	if gd == nil || gd.AssetsDir == "" || gd.DB == nil {
		return
	}

	for i := range gd.DB.Records {
		if gd.DB.Records[i].ID == id {
			imgs, _ := LoadFingerprintImages(gd.AssetsDir, &gd.DB.Records[i])
			s.targetImages[id] = imgs

			return
		}
	}
}

func (s *GameScene) ensureGreyLoaded(id int) {
	if _, ok := s.targetGreyImages[id]; ok {
		return
	}

	gd := s.gameData()
	if gd == nil || gd.AssetsDir == "" || gd.DB == nil {
		return
	}

	for i := range gd.DB.Records {
		if gd.DB.Records[i].ID == id {
			rec := &gd.DB.Records[i]
			imgs, _ := LoadGreyFingerprintImages(gd.AssetsDir, rec.Variant, rec.Rotation, rec.Mirrored)
			s.targetGreyImages[id] = imgs

			return
		}
	}
}

// --- Helpers ---

func firstUnsolvedPuzzle(cs *domain.CaseConfig) int {
	for i, p := range cs.Puzzles {
		if !p.Solved && !p.Failed {
			return i
		}
	}

	return 0
}

// FindTMXPath returns the TMX path as a real filesystem path.
func FindTMXPath() string {
	for _, p := range []string{
		"external/fingerprint/fingerprint.tmx",
		"assets/external/fingerprint/fingerprint.tmx",
		"../assets/external/fingerprint/fingerprint.tmx",
		"../../assets/external/fingerprint/fingerprint.tmx",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// FindFingerprintAssetsDir returns the assets directory as a real filesystem path.
func FindFingerprintAssetsDir() string {
	for _, p := range []string{
		"external/fingerprint",
		"assets/external/fingerprint",
		"../assets/external/fingerprint",
		"../../assets/external/fingerprint",
	} {
		if _, err := os.Stat(filepath.Join(p, "fingerprints")); err == nil {
			return p
		}
	}

	return ""
}
