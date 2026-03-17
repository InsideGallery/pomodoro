package fsystems

import (
	"context"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
	"github.com/InsideGallery/pomodoro/pkg/tilemap"
	"github.com/InsideGallery/pomodoro/pkg/ui"
	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
)

// RenderSystem draws all entities. Implements both Draw (world space) and ScreenDraw (UI).
type RenderSystem struct {
	scene    SceneAccessor
	dragdrop *DragDropSystem
}

func NewRenderSystem(scene SceneAccessor, dragdrop *DragDropSystem) *RenderSystem {
	return &RenderSystem{scene: scene, dragdrop: dragdrop}
}

func (s *RenderSystem) Update(_ context.Context) error { return nil }

// Draw renders world-space content (puzzle grid, pieces) with camera transform.
func (s *RenderSystem) Draw(_ context.Context, screen *ebiten.Image) {
	reg := s.scene.GetRegistry()
	screen.Fill(color.RGBA{A: 0xFF})

	val, err := reg.Get(c.GroupGameState, 0)
	if err != nil {
		return
	}

	entity, ok := val.(*c.Entity)
	if !ok || entity.State == nil {
		return
	}

	state := entity.State.Current

	switch state {
	case c.StateLoading:
		s.drawLoading(screen, reg)
	case c.StateDisabled:
		s.drawDisabled(screen, reg, entity.State.BootTick)
	case c.StateEnabled:
		s.drawLayers(screen, "enabled")
		s.drawEnabledButtons(screen)
	case c.StateApplicationLayout:
		s.drawLayers(screen, "enabled")
		s.drawLayers(screen, "application-layout")
		s.drawAppContent(screen, reg)
	case c.StateApplicationNet:
		s.drawLayers(screen, "enabled")
		s.drawLayers(screen, "application-net-layout")
		s.drawPuzzleContent(screen, reg)
		s.drawResultOverlay(screen, reg)
		s.drawHeldPiece(screen, reg)
	}

	// Cursor always on top (after boot)
	if state >= c.StateEnabled {
		s.drawCursor(screen, reg)
	}
}

// ScreenDraw is called after Draw for UI overlays in screen space.
func (s *RenderSystem) ScreenDraw(_ context.Context, _ *ebiten.Image) {}

// --- Loading ---

func (s *RenderSystem) drawLoading(screen *ebiten.Image, reg RegType) {
	w, h := s.scene.GetScreenSize()
	cx := float64(w) / 2
	cy := float64(h) / 2

	ui.DrawTextCentered(screen, "Loading...", ui.Face(true, 14),
		cx, cy-40, color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF})

	val, err := reg.Get(c.GroupGameState, 0)
	if err != nil {
		return
	}

	entity, ok := val.(*c.Entity)
	if !ok || entity.GameData == nil {
		return
	}

	gd := entity.GameData
	barW := 300.0
	barH := 12.0
	barX := cx - barW/2
	barY := cy - barH/2

	ui.DrawRoundedRect(screen, float32(barX), float32(barY),
		float32(barW), float32(barH), 4,
		color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF})

	fillW := barW * gd.LoadProgress
	if fillW > 0 {
		ui.DrawRoundedRect(screen, float32(barX), float32(barY),
			float32(fillW), float32(barH), 4,
			color.RGBA{R: 0x2E, G: 0x86, B: 0x8E, A: 0xFF})
	}

	if gd.LoadStatus != "" {
		ui.DrawTextCentered(screen, gd.LoadStatus, ui.Face(false, 8),
			cx, cy+20, color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF})
	}
}

// --- Disabled (boot animation) ---

func (s *RenderSystem) drawDisabled(screen *ebiten.Image, _ RegType, bootTick int) {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	co := CoordsFromScene(s.scene)
	progress := float64(bootTick) / 90.0

	if progress > 1 {
		progress = 1
	}

	if progress < 0.5 {
		s.drawImageLayer(screen, tmap, "disabled", 1.0, co)
	} else {
		fade := (progress - 0.5) * 2
		s.drawImageLayer(screen, tmap, "disabled", 1.0-fade, co)
		s.drawImageLayer(screen, tmap, "enabled", fade, co)
	}
}

// --- Layer rendering helpers ---

func (s *RenderSystem) drawLayers(screen *ebiten.Image, prefix string) {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	co := CoordsFromScene(s.scene)
	s.drawImageLayer(screen, tmap, prefix, 1.0, co)
	s.drawTileLayerScaled(screen, tmap, prefix, co)
}

func (s *RenderSystem) drawImageLayer(screen *ebiten.Image, tmap *tilemap.Map, name string, alpha float64, co Coords) {
	layer := tmap.FindImageLayer(name)
	if layer == nil {
		return
	}

	img := tmap.ImageLayerImage(layer)
	if img == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(co.Scale, co.Scale)
	op.GeoM.Translate(co.OffsetX, co.OffsetY)

	if alpha < 1.0 {
		op.ColorScale.Scale(float32(alpha), float32(alpha), float32(alpha), float32(alpha))
	}

	screen.DrawImage(img, op)
}

func (s *RenderSystem) drawTileLayerScaled(screen *ebiten.Image, tmap *tilemap.Map, name string, co Coords) {
	layer := tmap.FindTileLayer(name)
	if layer == nil {
		return
	}

	tmap.DrawTileLayer(screen, layer, co.Scale, co.Scale, co.OffsetX, co.OffsetY)
}

// --- Enabled state buttons ---

func (s *RenderSystem) drawEnabledButtons(screen *ebiten.Image) {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	co := CoordsFromScene(s.scene)

	og := tmap.FindObjectGroup("enabled")
	if og == nil {
		return
	}

	if quitObj := tilemap.FindObject(og, "button-quit-os"); quitObj != nil {
		qx := co.MapToScreenX(quitObj.X)
		qy := co.MapToScreenY(quitObj.Y)
		qw := co.MapToScreenSize(200)
		qh := co.MapToScreenSize(50)

		ui.DrawRoundedRect(screen, float32(qx), float32(qy), float32(qw), float32(qh), 4,
			color.RGBA{R: 0xCC, G: 0x33, B: 0x33, A: 0xCC})
		ui.DrawTextCentered(screen, "QUIT", ui.Face(true, 11), qx+qw/2, qy+qh/2-8,
			color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	}
}

// --- Application layout content ---

func (s *RenderSystem) drawAppContent(screen *ebiten.Image, reg RegType) { //nolint:gocyclo // UI rendering
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	og := tmap.FindObjectGroup("application-layout")
	if og == nil {
		return
	}

	co := CoordsFromScene(s.scene)
	faceList := ui.Face(true, 9)
	faceBtn := ui.Face(true, 8)
	white := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	textClr := color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF}

	gd := GetGameData(reg)
	if gd == nil {
		return
	}

	cases := gd.Cases
	selectedCase := gd.SelectedCase
	selectedPuzzle := gd.SelectedPuzzle

	casesScroll := gd.CasesScroll

	if casesObj := tilemap.FindObject(og, "list-of-cases"); casesObj != nil {
		x := co.MapToScreenX(casesObj.X)
		y := co.MapToScreenY(casesObj.Y)
		w := co.MapToScreenSize(casesObj.Width)
		h := co.MapToScreenSize(casesObj.Height)
		rowH := co.MapToScreenSize(45)

		for i := casesScroll; i < len(cases); i++ {
			ry := y + float64(i-casesScroll)*rowH
			if ry+rowH > y+h {
				break
			}

			btnClr := color.RGBA{R: 0x3A, G: 0x5A, B: 0x5A, A: 0x80}
			if i == selectedCase {
				btnClr = color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x60}
			}

			ui.DrawRoundedRect(screen, float32(x+1), float32(ry+1), float32(w-2), float32(rowH-2), 2, btnClr)

			solved := 0
			for _, p := range cases[i].Puzzles {
				if p.Solved || p.Failed {
					solved++
				}
			}

			label := fmt.Sprintf("%s (%d/%d)", cases[i].Name, solved, len(cases[i].Puzzles))
			ui.DrawText(screen, label, faceList, x+6, ry+8, textClr)
		}

		s.drawScrollbar(screen, x, y, w, h, rowH, len(cases), casesScroll)
	}

	// Scrollable puzzle list
	namesScroll := gd.NamesScroll

	if namesObj := tilemap.FindObject(og, "fingerprints-user-names"); namesObj != nil && selectedCase >= 0 && selectedCase < len(cases) {
		x := co.MapToScreenX(namesObj.X)
		y := co.MapToScreenY(namesObj.Y)
		w := co.MapToScreenSize(namesObj.Width)
		h := co.MapToScreenSize(namesObj.Height)
		rowH := co.MapToScreenSize(50)
		cs := cases[selectedCase]

		for i := namesScroll; i < len(cs.Puzzles); i++ {
			ry := y + float64(i-namesScroll)*rowH
			if ry+rowH > y+h {
				break
			}

			p := cs.Puzzles[i]
			name := "Unknown"

			if p.Solved && p.TargetRecord != nil {
				name = p.TargetRecord.PersonName
			} else if p.Failed {
				name = "No match"
			}

			btnClr := color.RGBA{R: 0x3A, G: 0x5A, B: 0x5A, A: 0xAA}
			txtClr := textClr

			if i == selectedPuzzle {
				btnClr = color.RGBA{R: 0x2E, G: 0x86, B: 0x8E, A: 0xCC}
				txtClr = white
			}

			label := fmt.Sprintf("%d. %s", i+1, name)
			ui.DrawRoundedRect(screen, float32(x+1), float32(ry+1), float32(w-2), float32(rowH-2), 2, btnClr)
			ui.DrawText(screen, label, faceList, x+6, ry+8, txtClr)
		}

		s.drawScrollbar(screen, x, y, w, h, rowH, len(cs.Puzzles), namesScroll)
	}

	// Avatar
	if avatarObj := tilemap.FindObject(og, "avatar"); avatarObj != nil {
		ax := co.MapToScreenX(avatarObj.X)
		ay := co.MapToScreenY(avatarObj.Y)
		aw := co.MapToScreenSize(avatarObj.Width)
		ah := co.MapToScreenSize(avatarObj.Height)

		puzzle := s.currentPuzzle()
		if puzzle != nil {
			avatarFile := domain.UnknownAvatar
			if puzzle.Solved {
				avatarFile = domain.AvatarForRecord(puzzle.TargetRecord)
			}

			avatarImg := s.scene.GetAvatarImage(avatarFile)
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

	// Description
	descScroll := gd.DescScroll

	if descObj := tilemap.FindObject(og, "description"); descObj != nil {
		dx := co.MapToScreenX(descObj.X)
		dy := co.MapToScreenY(descObj.Y)
		dw := co.MapToScreenSize(descObj.Width)
		dh := co.MapToScreenSize(descObj.Height)

		puzzle := s.currentPuzzle()
		if puzzle != nil {
			var descText string

			switch {
			case puzzle.Solved:
				descText = domain.SolvedDescription(selectedCase, selectedPuzzle, puzzle.TargetRecord.PersonName)
			case puzzle.Failed:
				descText = domain.NoMatchDescription(selectedCase, selectedPuzzle)
			default:
				descText = domain.UnsolvedDescription(selectedCase, selectedPuzzle)
			}

			drawWrappedText(screen, descText, dx+4, dy+4, dw-8, dh-8, descScroll, textClr)
		}
	}

	// Programmatic buttons
	if btnObj := tilemap.FindObject(og, "play-puzzle"); btnObj != nil {
		bx := co.MapToScreenX(btnObj.X)
		by := co.MapToScreenY(btnObj.Y)
		bw := co.MapToScreenSize(btnObj.Width)
		bh := co.MapToScreenSize(btnObj.Height)

		ui.DrawRoundedRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 4,
			color.RGBA{R: 0x2E, G: 0x86, B: 0x8E, A: 0xDD})
		ui.DrawTextCentered(screen, "OPEN PUZZLE", faceBtn, bx+bw/2, by+bh/2-6, white)
	}

	if btnObj := tilemap.FindObject(og, "regenerate-puzzles"); btnObj != nil {
		bx := co.MapToScreenX(btnObj.X)
		by := co.MapToScreenY(btnObj.Y)
		bw := co.MapToScreenSize(btnObj.Width)
		bh := co.MapToScreenSize(btnObj.Height)

		ui.DrawRoundedRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 4,
			color.RGBA{R: 0x8E, G: 0x44, B: 0x2E, A: 0xDD})
		ui.DrawTextCentered(screen, "REGENERATE", faceBtn, bx+bw/2, by+bh/2-6, white)
	}
}

// --- Puzzle workspace content ---

func (s *RenderSystem) drawPuzzleContent(screen *ebiten.Image, _ RegType) { //nolint:gocyclo // puzzle rendering
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	og := tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return
	}

	puzzle := s.currentPuzzle()
	if puzzle == nil {
		return
	}

	co := CoordsFromScene(s.scene)
	faceHash := ui.Face(false, 10)

	// Hash display
	if hashObj := tilemap.FindObject(og, "hash"); hashObj != nil {
		hx := co.MapToScreenX(hashObj.X)
		hy := co.MapToScreenY(hashObj.Y)
		hashText := s.computeCurrentHash(puzzle)

		ui.DrawText(screen, hashText, faceHash, hx+4, hy+4,
			color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})
	}

	// Puzzle grid
	if puzzleObj := tilemap.FindObject(og, "puzzle"); puzzleObj != nil {
		px := co.MapToScreenX(puzzleObj.X)
		py := co.MapToScreenY(puzzleObj.Y)
		pw := co.MapToScreenSize(puzzleObj.Width)
		ph := co.MapToScreenSize(puzzleObj.Height)
		side := math.Min(pw, ph)

		px += (pw - side) / 2
		py += (ph - side) / 2
		cellW := side / 10

		// Grid lines
		for i := range 11 {
			lx := float32(px + float64(i)*cellW)
			ui.DrawRoundedRect(screen, lx, float32(py), 1, float32(side), 0,
				color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x40})

			ly := float32(py + float64(i)*cellW)
			ui.DrawRoundedRect(screen, float32(px), ly, float32(side), 1, 0,
				color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x40})
		}

		missingSet := make(map[int]bool)
		for _, idx := range puzzle.MissingIndices {
			missingSet[idx] = true
		}

		// Pre-filled pieces
		for idx := range 100 {
			if missingSet[idx] {
				continue
			}

			col := idx % 10
			row := idx / 10
			cx := px + float64(col)*cellW
			cy := py + float64(row)*cellW

			var pieceImg *ebiten.Image
			if puzzle.HideColor {
				pieceImg = s.scene.GetGreyPieceImage(puzzle.TargetRecord.ID, idx)
			}

			if pieceImg == nil {
				pieceImg = s.scene.GetTargetPieceImage(puzzle.TargetRecord.ID, idx)
			}

			if pieceImg != nil {
				op := &ebiten.DrawImageOptions{}
				iw := float64(pieceImg.Bounds().Dx())
				ih := float64(pieceImg.Bounds().Dy())
				op.GeoM.Scale(cellW/iw, cellW/ih)
				op.GeoM.Translate(cx, cy)
				screen.DrawImage(pieceImg, op)
			}
		}

		// Placed pieces + empty slots
		placedAt := make(map[int]int)
		for ti, tp := range puzzle.TrayPieces {
			if tp.IsPlaced {
				gIdx := tp.PlacedY*10 + tp.PlacedX
				placedAt[gIdx] = ti
			}
		}

		for _, idx := range puzzle.MissingIndices {
			col := idx % 10
			row := idx / 10
			cx := px + float64(col)*cellW
			cy := py + float64(row)*cellW

			if ti, ok := placedAt[idx]; ok {
				tp := puzzle.TrayPieces[ti]
				pieceImg := s.getPieceImage(puzzle, tp)

				if pieceImg != nil {
					drawRotatedPiece(screen, pieceImg, cx, cy, cellW, tp.Rotation)
				} else {
					clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0xCC}
					if tp.IsDecoy {
						clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0xCC}
					}

					ui.DrawRoundedRect(screen, float32(cx+1), float32(cy+1),
						float32(cellW-2), float32(cellW-2), 1, clr)
				}
			} else {
				ui.DrawRoundedRect(screen, float32(cx+1), float32(cy+1),
					float32(cellW-2), float32(cellW-2), 1,
					color.RGBA{R: 0xFF, G: 0xA0, B: 0x00, A: 0x30})
			}
		}
	}

	// Tray pieces
	cellSz := s.gridCellScreenSize(co)
	holdIdx := s.dragdrop.HoldingPiece()
	isDragging := s.dragdrop.IsDragging()

	for i, tp := range puzzle.TrayPieces {
		if tp.IsPlaced || (isDragging && i == holdIdx) {
			continue
		}

		tx := co.MapToScreenX(tp.TrayX)
		ty := co.MapToScreenY(tp.TrayY)

		pieceImg := s.getPieceImage(puzzle, tp)
		if pieceImg != nil {
			drawRotatedPiece(screen, pieceImg, tx, ty, cellSz, tp.Rotation)
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

// --- Result overlay ---

func (s *RenderSystem) drawResultOverlay(screen *ebiten.Image, reg RegType) {
	gd := GetGameData(reg)
	if gd == nil {
		return
	}

	showResult := gd.ShowResult
	if showResult == 0 {
		return
	}

	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	co := CoordsFromScene(s.scene)

	if showResult == 1 {
		s.drawTileLayerScaled(screen, tmap, "application-net-layout-success", co)
	} else {
		s.drawTileLayerScaled(screen, tmap, "application-net-layout-fail", co)
	}
}

// --- Held piece ---

func (s *RenderSystem) drawHeldPiece(screen *ebiten.Image, reg RegType) {
	if !s.dragdrop.IsDragging() || s.dragdrop.HoldingPiece() < 0 {
		return
	}

	puzzle := s.currentPuzzle()
	if puzzle == nil || s.dragdrop.HoldingPiece() >= len(puzzle.TrayPieces) {
		return
	}

	tp := puzzle.TrayPieces[s.dragdrop.HoldingPiece()]
	cx, cy := s.getCursorScreenPos(reg)
	sz := s.gridCellScreenSize(CoordsFromScene(s.scene))

	pieceImg := s.getPieceImage(puzzle, tp)
	if pieceImg != nil {
		drawRotatedPiece(screen, pieceImg, cx-sz/2, cy-sz/2, sz, tp.Rotation)
	} else {
		clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x90}
		if tp.IsDecoy {
			clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0x90}
		}

		ui.DrawRoundedRect(screen, float32(cx-sz/2), float32(cy-sz/2),
			float32(sz), float32(sz), 3, clr)
	}
}

// --- Cursor ---

func (s *RenderSystem) drawCursor(screen *ebiten.Image, reg RegType) {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	cursorImg := tmap.GetImage("ui/cursor.png")
	if cursorImg == nil {
		return
	}

	cx, cy := s.getCursorScreenPos(reg)

	op := &ebiten.DrawImageOptions{}
	cw := float64(cursorImg.Bounds().Dx())
	cursorScale := 32.0 / cw
	op.GeoM.Scale(cursorScale, cursorScale)
	op.GeoM.Translate(cx, cy)
	screen.DrawImage(cursorImg, op)
}

// --- Helpers ---

func (s *RenderSystem) currentPuzzle() *domain.PuzzleConfig {
	return CurrentPuzzle(GetGameData(s.scene.GetRegistry()))
}

func (s *RenderSystem) getCursorScreenPos(reg RegType) (float64, float64) {
	val, err := reg.Get(c.GroupCursor, 0)
	if err != nil {
		return 0, 0
	}

	entity, ok := val.(*c.Entity)
	if !ok || entity.Cursor == nil {
		return 0, 0
	}

	return float64(entity.Cursor.X), float64(entity.Cursor.Y)
}

func (s *RenderSystem) getScrollOffset(reg RegType, group string) int {
	val, err := reg.Get(group, 0)
	if err != nil {
		return 0
	}

	entity, ok := val.(*c.Entity)
	if !ok || entity.Scrollable == nil {
		return 0
	}

	return entity.Scrollable.Scroll
}

func (s *RenderSystem) getTextScrollOffset(reg RegType, group string) int {
	val, err := reg.Get(group, 0)
	if err != nil {
		return 0
	}

	entity, ok := val.(*c.Entity)
	if !ok || entity.TextBlock == nil {
		return 0
	}

	return entity.TextBlock.Scroll
}

func (s *RenderSystem) gridCellScreenSize(co Coords) float64 {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return 40
	}

	og := tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return 40
	}

	for _, obj := range og.Objects {
		if obj.Name == "puzzle" {
			pw := co.MapToScreenSize(obj.Width)
			ph := co.MapToScreenSize(obj.Height)

			return math.Min(pw, ph) / 10
		}
	}

	return 40
}

func (s *RenderSystem) getPieceImage(puzzle *domain.PuzzleConfig, tp domain.TrayPiece) *ebiten.Image {
	if !tp.IsDecoy {
		if tp.OriginalX >= 0 {
			origIdx := tp.OriginalY*10 + tp.OriginalX
			if origIdx >= 0 && origIdx < 100 {
				return s.scene.GetTargetPieceImage(puzzle.TargetRecord.ID, origIdx)
			}
		}

		return nil
	}

	rot := 0
	mirror := false

	if puzzle.TargetRecord != nil {
		rot = puzzle.TargetRecord.Rotation
		mirror = puzzle.TargetRecord.Mirrored
	}

	return s.scene.GetDecoyPieceImage(tp.DecoyColor, tp.DecoyVariant, rot, mirror, tp.DecoyPieceIdx)
}

func (s *RenderSystem) computeCurrentHash(puzzle *domain.PuzzleConfig) string {
	pieces := make([]domain.PieceRecord, 100)

	for i, piece := range puzzle.TargetRecord.Pieces {
		pieces[i] = piece
	}

	for _, idx := range puzzle.MissingIndices {
		pieces[idx] = domain.PieceRecord{X: idx % 10, Y: idx / 10, Value: 0}
	}

	for _, tp := range puzzle.TrayPieces {
		if !tp.IsPlaced {
			continue
		}

		gIdx := tp.PlacedY*10 + tp.PlacedX
		if gIdx >= 0 && gIdx < 100 {
			pieces[gIdx] = domain.PieceRecord{X: tp.PlacedX, Y: tp.PlacedY, Value: tp.Value}
		}
	}

	colorLetter := domain.ColorLetter(puzzle.TargetRecord.Color)
	if puzzle.HideColor {
		colorLetter = "?"
	}

	hash := domain.ComputeHash(pieces)

	return fmt.Sprintf("%s%d", colorLetter, hash)
}

func (s *RenderSystem) drawScrollbar(screen *ebiten.Image, x, y, w, h, rowH float64, totalItems, scroll int) {
	maxScroll := totalItems - int(h/rowH)
	if maxScroll <= 0 {
		return
	}

	sbH := h * h / (float64(totalItems) * rowH)
	sbY := y + float64(scroll)/float64(maxScroll)*(h-sbH)
	ui.DrawRoundedRect(screen, float32(x+w-4), float32(sbY), 3, float32(sbH), 1,
		color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80})
}

// --- Standalone drawing helpers (no receiver, reusable) ---

func drawRotatedPiece(screen *ebiten.Image, img *ebiten.Image, x, y, size float64, rotation int) {
	iw := float64(img.Bounds().Dx())
	ih := float64(img.Bounds().Dy())
	scale := size / iw

	op := &ebiten.DrawImageOptions{}

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

func drawWrappedText(screen *ebiten.Image, text string, x, y, maxW, maxH float64, scroll int, clr color.Color) {
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

func wrapText(text string, face *textv2.GoTextFace, maxW float64) []string {
	var lines []string

	for _, paragraph := range splitNewlines(text) {
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

func splitNewlines(s string) []string {
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
	for i, ch := range s {
		if ch == ' ' {
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
