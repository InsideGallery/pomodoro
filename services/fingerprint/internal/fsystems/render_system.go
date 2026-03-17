package fsystems

import (
	"context"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
	"github.com/InsideGallery/pomodoro/pkg/tilemap"
	"github.com/InsideGallery/pomodoro/pkg/ui"
	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
)

// RenderSystem draws all entities in world (map) coordinates.
// BaseScene composites World to screen via Camera.WorldMatrix().
// Cursor drawn in ScreenDraw (screen space).
type RenderSystem struct {
	scene    SceneAccessor
	dragdrop *DragDropSystem
}

func NewRenderSystem(scene SceneAccessor, dragdrop *DragDropSystem) *RenderSystem {
	return &RenderSystem{scene: scene, dragdrop: dragdrop}
}

func (s *RenderSystem) Update(_ context.Context) error { return nil }

// Draw renders everything in world (map) coordinates.
func (s *RenderSystem) Draw(_ context.Context, world *ebiten.Image) {
	reg := s.scene.GetRegistry()
	state := GetState(reg)

	if state == nil {
		return
	}

	switch state.Current {
	case c.StateLoading:
		// Loading screen drawn in ScreenDraw (screen space)
	case c.StateDisabled:
		s.drawDisabled(world, state.BootTick)
	case c.StateEnabled:
		s.drawLayers(world, "enabled")
		s.drawEnabledButtons(world)
	case c.StateApplicationLayout:
		s.drawLayers(world, "enabled")
		s.drawLayers(world, "application-layout")
		s.drawAppContent(world, reg)
	case c.StateApplicationNet:
		s.drawLayers(world, "enabled")
		s.drawLayers(world, "application-net-layout")
		s.drawPuzzleContent(world, reg)
		s.drawResultOverlay(world, reg)
		s.drawHeldPiece(world, reg)
	}
}

// ScreenDraw draws UI overlays in screen space (no camera transform).
func (s *RenderSystem) ScreenDraw(_ context.Context, screen *ebiten.Image) {
	reg := s.scene.GetRegistry()
	state := GetState(reg)

	if state == nil {
		return
	}

	// Loading screen in screen space
	if state.Current == c.StateLoading {
		screen.Fill(color.RGBA{A: 0xFF})
		s.drawLoading(screen, reg)

		return
	}

	// Cursor always on top in screen space
	if state.Current >= c.StateEnabled {
		s.drawCursor(screen, reg)
	}
}

// --- Loading (screen space) ---

func (s *RenderSystem) drawLoading(screen *ebiten.Image, reg RegType) {
	w, h := s.scene.GetScreenSize()
	cx := float64(w) / 2
	cy := float64(h) / 2

	ui.DrawTextCentered(screen, "Loading...", ui.Face(true, 14),
		cx, cy-40, color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF})

	gd := GetGameData(reg)
	if gd == nil {
		return
	}

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

// --- Disabled (boot animation) — world space ---

func (s *RenderSystem) drawDisabled(world *ebiten.Image, bootTick int) {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	progress := float64(bootTick) / 90.0
	if progress > 1 {
		progress = 1
	}

	if progress < 0.5 {
		s.drawImageLayer(world, tmap, "disabled", 1.0)
	} else {
		fade := (progress - 0.5) * 2
		s.drawImageLayer(world, tmap, "disabled", 1.0-fade)
		s.drawImageLayer(world, tmap, "enabled", fade)
	}
}

// --- Layer rendering — world space (no scale needed!) ---

func (s *RenderSystem) drawLayers(world *ebiten.Image, prefix string) {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	s.drawImageLayer(world, tmap, prefix, 1.0)
	s.drawTileLayer(world, tmap, prefix)
}

func (s *RenderSystem) drawImageLayer(world *ebiten.Image, tmap *tilemap.Map, name string, alpha float64) {
	layer := tmap.FindImageLayer(name)
	if layer == nil {
		return
	}

	img := tmap.ImageLayerImage(layer)
	if img == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	// No scale — world image is map-sized, image layers are map-sized

	if alpha < 1.0 {
		op.ColorScale.Scale(float32(alpha), float32(alpha), float32(alpha), float32(alpha))
	}

	world.DrawImage(img, op)
}

func (s *RenderSystem) drawTileLayer(world *ebiten.Image, tmap *tilemap.Map, name string) {
	layer := tmap.FindTileLayer(name)
	if layer == nil {
		return
	}

	// Scale 1:1, offset 0 — tile layer coordinates ARE world coordinates
	tmap.DrawTileLayer(world, layer, 1.0, 1.0, 0, 0)
}

// --- Enabled state buttons — world space ---

func (s *RenderSystem) drawEnabledButtons(world *ebiten.Image) {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	og := tmap.FindObjectGroup("enabled")
	if og == nil {
		return
	}

	if quitObj := tilemap.FindObject(og, "button-quit-os"); quitObj != nil {
		// Draw at map coordinates directly
		ui.DrawRoundedRect(world, float32(quitObj.X), float32(quitObj.Y), 200, 50, 4,
			color.RGBA{R: 0xCC, G: 0x33, B: 0x33, A: 0xCC})
		ui.DrawTextCentered(world, "QUIT", ui.Face(true, 22),
			quitObj.X+100, quitObj.Y+10,
			color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	}
}

// --- Application layout content — world space ---

func (s *RenderSystem) drawAppContent(world *ebiten.Image, reg RegType) { //nolint:gocyclo // UI
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	og := tmap.FindObjectGroup("application-layout")
	if og == nil {
		return
	}

	faceList := ui.Face(true, 16)
	faceBtn := ui.Face(true, 16)
	white := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	textClr := color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF}

	gd := GetGameData(reg)
	if gd == nil {
		return
	}

	// Scrollable case list
	if casesObj := tilemap.FindObject(og, "list-of-cases"); casesObj != nil {
		x, y, w, h := casesObj.X, casesObj.Y, casesObj.Width, casesObj.Height
		rowH := 90.0 // map units

		for i := gd.CasesScroll; i < len(gd.Cases); i++ {
			ry := y + float64(i-gd.CasesScroll)*rowH
			if ry+rowH > y+h {
				break
			}

			btnClr := color.RGBA{R: 0x3A, G: 0x5A, B: 0x5A, A: 0x80}
			if i == gd.SelectedCase {
				btnClr = color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x60}
			}

			ui.DrawRoundedRect(world, float32(x+2), float32(ry+2), float32(w-4), float32(rowH-4), 4, btnClr)

			solved := 0
			for _, p := range gd.Cases[i].Puzzles {
				if p.Solved || p.Failed {
					solved++
				}
			}

			label := fmt.Sprintf("%s (%d/%d)", gd.Cases[i].Name, solved, len(gd.Cases[i].Puzzles))
			ui.DrawText(world, label, faceList, x+12, ry+16, textClr)
		}

		s.drawScrollbar(world, x, y, w, h, rowH, len(gd.Cases), gd.CasesScroll)
	}

	// Scrollable puzzle list
	if namesObj := tilemap.FindObject(og, "fingerprints-user-names"); namesObj != nil && gd.SelectedCase >= 0 && gd.SelectedCase < len(gd.Cases) {
		x, y, w, h := namesObj.X, namesObj.Y, namesObj.Width, namesObj.Height
		rowH := 100.0
		cs := gd.Cases[gd.SelectedCase]

		for i := gd.NamesScroll; i < len(cs.Puzzles); i++ {
			ry := y + float64(i-gd.NamesScroll)*rowH
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

			if i == gd.SelectedPuzzle {
				btnClr = color.RGBA{R: 0x2E, G: 0x86, B: 0x8E, A: 0xCC}
				txtClr = white
			}

			label := fmt.Sprintf("%d. %s", i+1, name)
			ui.DrawRoundedRect(world, float32(x+2), float32(ry+2), float32(w-4), float32(rowH-4), 4, btnClr)
			ui.DrawText(world, label, faceList, x+12, ry+16, txtClr)
		}

		s.drawScrollbar(world, x, y, w, h, rowH, len(cs.Puzzles), gd.NamesScroll)
	}

	// Avatar
	if avatarObj := tilemap.FindObject(og, "avatar"); avatarObj != nil {
		puzzle := CurrentPuzzle(gd)
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
				op.GeoM.Scale(avatarObj.Width/iw, avatarObj.Height/ih)
				op.GeoM.Translate(avatarObj.X, avatarObj.Y)
				world.DrawImage(avatarImg, op)
			} else {
				ui.DrawRoundedRect(world, float32(avatarObj.X), float32(avatarObj.Y),
					float32(avatarObj.Width), float32(avatarObj.Height), 8,
					color.RGBA{R: 0xD5, G: 0xF2, B: 0xF1, A: 0xFF})
				ui.DrawTextCentered(world, "?", ui.Face(true, 48),
					avatarObj.X+avatarObj.Width/2, avatarObj.Y+avatarObj.Height/2-20, textClr)
			}
		}
	}

	// Description
	if descObj := tilemap.FindObject(og, "description"); descObj != nil {
		puzzle := CurrentPuzzle(gd)
		if puzzle != nil {
			var descText string

			switch {
			case puzzle.Solved:
				descText = domain.SolvedDescription(gd.SelectedCase, gd.SelectedPuzzle, puzzle.TargetRecord.PersonName)
			case puzzle.Failed:
				descText = domain.NoMatchDescription(gd.SelectedCase, gd.SelectedPuzzle)
			default:
				descText = domain.UnsolvedDescription(gd.SelectedCase, gd.SelectedPuzzle)
			}

			drawWrappedText(world, descText, descObj.X+8, descObj.Y+8,
				descObj.Width-16, descObj.Height-16, gd.DescScroll, textClr)
		}
	}

	// Programmatic buttons
	if btnObj := tilemap.FindObject(og, "play-puzzle"); btnObj != nil {
		ui.DrawRoundedRect(world, float32(btnObj.X), float32(btnObj.Y),
			float32(btnObj.Width), float32(btnObj.Height), 8,
			color.RGBA{R: 0x2E, G: 0x86, B: 0x8E, A: 0xDD})
		ui.DrawTextCentered(world, "OPEN PUZZLE", faceBtn,
			btnObj.X+btnObj.Width/2, btnObj.Y+btnObj.Height/2-12, white)
	}

	if btnObj := tilemap.FindObject(og, "regenerate-puzzles"); btnObj != nil {
		ui.DrawRoundedRect(world, float32(btnObj.X), float32(btnObj.Y),
			float32(btnObj.Width), float32(btnObj.Height), 8,
			color.RGBA{R: 0x8E, G: 0x44, B: 0x2E, A: 0xDD})
		ui.DrawTextCentered(world, "REGENERATE", faceBtn,
			btnObj.X+btnObj.Width/2, btnObj.Y+btnObj.Height/2-12, white)
	}
}

// --- Puzzle workspace — world space ---

func (s *RenderSystem) drawPuzzleContent(world *ebiten.Image, _ RegType) { //nolint:gocyclo // puzzle
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	og := tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return
	}

	puzzle := CurrentPuzzle(GetGameData(s.scene.GetRegistry()))
	if puzzle == nil {
		return
	}

	faceHash := ui.Face(false, 18)

	// Hash
	if hashObj := tilemap.FindObject(og, "hash"); hashObj != nil {
		hashText := s.computeCurrentHash(puzzle)
		ui.DrawText(world, hashText, faceHash, hashObj.X+8, hashObj.Y+8,
			color.RGBA{R: 0x4D, G: 0x4B, B: 0x4B, A: 0xFF})
	}

	// Puzzle grid
	if puzzleObj := tilemap.FindObject(og, "puzzle"); puzzleObj != nil {
		px, py := puzzleObj.X, puzzleObj.Y
		pw, ph := puzzleObj.Width, puzzleObj.Height
		side := math.Min(pw, ph)

		px += (pw - side) / 2
		py += (ph - side) / 2
		cellW := side / 10

		// Grid lines
		for i := range 11 {
			lx := float32(px + float64(i)*cellW)
			ui.DrawRoundedRect(world, lx, float32(py), 1, float32(side), 0,
				color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x40})

			ly := float32(py + float64(i)*cellW)
			ui.DrawRoundedRect(world, float32(px), ly, float32(side), 1, 0,
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
				op.GeoM.Scale(cellW/iw, cellW/iw)
				op.GeoM.Translate(cx, cy)
				world.DrawImage(pieceImg, op)
			}
		}

		// Placed + empty slots
		placedAt := make(map[int]int)
		for ti, tp := range puzzle.TrayPieces {
			if tp.IsPlaced {
				placedAt[tp.PlacedY*10+tp.PlacedX] = ti
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
					drawRotatedPiece(world, pieceImg, cx, cy, cellW, tp.Rotation)
				} else {
					clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0xCC}
					if tp.IsDecoy {
						clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0xCC}
					}

					ui.DrawRoundedRect(world, float32(cx+1), float32(cy+1),
						float32(cellW-2), float32(cellW-2), 2, clr)
				}
			} else {
				ui.DrawRoundedRect(world, float32(cx+1), float32(cy+1),
					float32(cellW-2), float32(cellW-2), 2,
					color.RGBA{R: 0xFF, G: 0xA0, B: 0x00, A: 0x30})
			}
		}
	}

	// Tray pieces at map coordinates
	cellSz := s.gridCellMapSize()
	holdIdx := s.dragdrop.HoldingPiece()
	isDragging := s.dragdrop.IsDragging()

	for i, tp := range puzzle.TrayPieces {
		if tp.IsPlaced || (isDragging && i == holdIdx) {
			continue
		}

		pieceImg := s.getPieceImage(puzzle, tp)
		if pieceImg != nil {
			drawRotatedPiece(world, pieceImg, tp.TrayX, tp.TrayY, cellSz, tp.Rotation)
		} else {
			clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0xCC}
			if tp.IsDecoy {
				clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0xCC}
			}

			ui.DrawRoundedRect(world, float32(tp.TrayX+1), float32(tp.TrayY+1),
				float32(cellSz-2), float32(cellSz-2), 4, clr)
		}
	}
}

func (s *RenderSystem) drawResultOverlay(world *ebiten.Image, reg RegType) {
	gd := GetGameData(reg)
	if gd == nil || gd.ShowResult == 0 {
		return
	}

	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	if gd.ShowResult == 1 {
		s.drawTileLayer(world, tmap, "application-net-layout-success")
	} else {
		s.drawTileLayer(world, tmap, "application-net-layout-fail")
	}
}

func (s *RenderSystem) drawHeldPiece(world *ebiten.Image, reg RegType) {
	if !s.dragdrop.IsDragging() || s.dragdrop.HoldingPiece() < 0 {
		return
	}

	puzzle := CurrentPuzzle(GetGameData(reg))
	if puzzle == nil || s.dragdrop.HoldingPiece() >= len(puzzle.TrayPieces) {
		return
	}

	tp := puzzle.TrayPieces[s.dragdrop.HoldingPiece()]

	// Convert screen cursor to world coordinates
	cur := GetCursor(reg)
	if cur == nil {
		return
	}

	wx, wy := s.scene.ScreenToWorld(float64(cur.X), float64(cur.Y))
	sz := s.gridCellMapSize()

	pieceImg := s.getPieceImage(puzzle, tp)
	if pieceImg != nil {
		drawRotatedPiece(world, pieceImg, wx-sz/2, wy-sz/2, sz, tp.Rotation)
	} else {
		clr := color.RGBA{R: 0x4D, G: 0x8B, B: 0x8B, A: 0x90}
		if tp.IsDecoy {
			clr = color.RGBA{R: 0x8B, G: 0x4D, B: 0x4D, A: 0x90}
		}

		ui.DrawRoundedRect(world, float32(wx-sz/2), float32(wy-sz/2),
			float32(sz), float32(sz), 4, clr)
	}
}

// --- Cursor (screen space) ---

func (s *RenderSystem) drawCursor(screen *ebiten.Image, reg RegType) {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return
	}

	cursorImg := tmap.GetImage("ui/cursor.png")
	if cursorImg == nil {
		return
	}

	cur := GetCursor(reg)
	if cur == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	cw := float64(cursorImg.Bounds().Dx())
	cursorScale := 32.0 / cw
	op.GeoM.Scale(cursorScale, cursorScale)
	op.GeoM.Translate(float64(cur.X), float64(cur.Y))
	screen.DrawImage(cursorImg, op)
}

// --- Helpers ---

func (s *RenderSystem) gridCellMapSize() float64 {
	tmap := s.scene.GetTileMap()
	if tmap == nil {
		return 68
	}

	og := tmap.FindObjectGroup("application-net-layout")
	if og == nil {
		return 68
	}

	for _, obj := range og.Objects {
		if obj.Name == "puzzle" {
			return math.Min(obj.Width, obj.Height) / 10
		}
	}

	return 68
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

	return fmt.Sprintf("%s%d", colorLetter, domain.ComputeHash(pieces))
}

func (s *RenderSystem) drawScrollbar(world *ebiten.Image, x, y, w, h, rowH float64, totalItems, scroll int) {
	maxScroll := totalItems - int(h/rowH)
	if maxScroll <= 0 {
		return
	}

	sbH := h * h / (float64(totalItems) * rowH)
	sbY := y + float64(scroll)/float64(maxScroll)*(h-sbH)
	ui.DrawRoundedRect(world, float32(x+w-6), float32(sbY), 5, float32(sbH), 2,
		color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80})
}

// --- Standalone drawing helpers ---

func drawRotatedPiece(dst *ebiten.Image, img *ebiten.Image, x, y, size float64, rotation int) {
	iw := float64(img.Bounds().Dx())
	scale := size / iw

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-iw/2, -iw/2)

	angle := float64(rotation%domain.RotationSteps) * math.Pi / 4
	if angle != 0 {
		op.GeoM.Rotate(angle)
	}

	op.GeoM.Translate(iw/2, iw/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)

	dst.DrawImage(img, op)
}

func drawWrappedText(dst *ebiten.Image, text string, x, y, maxW, maxH float64, scroll int, clr color.Color) {
	face := ui.Face(false, 18)
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

		ui.DrawText(dst, line, face, x, ty, clr)
		drawn++
	}
}
