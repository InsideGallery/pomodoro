package fingerprint

import (
	"context"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/resources"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const LoadingSceneName = "fingerprint_loading"

// LoadingScene shows an animated preloader while resources load.
// Uses the loading 1-4 + 1а-4а frames as spritesheet animation.
type LoadingScene struct {
	*scene.BaseScene

	switchScene func(string)
	targetScene string                 // scene to switch to after loading
	loadFunc    func(*scene.BaseScene) // starts async resource loading

	frame      int
	frameTick  int
	sheet      *ebiten.Image
	frameW     int
	frameCount int

	width, height int
}

func NewLoadingScene(switchScene func(string), targetScene string, loadFunc func(*scene.BaseScene)) *LoadingScene {
	return &LoadingScene{
		switchScene: switchScene,
		targetScene: targetScene,
		loadFunc:    loadFunc,
	}
}

func (s *LoadingScene) Name() string { return LoadingSceneName }

// SetResources replaces the scene's resource manager with a shared one.
func (s *LoadingScene) SetResources(rm *resources.Manager) {
	if s.BaseScene != nil {
		s.BaseScene.Resources = rm
	}
}

func (s *LoadingScene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
}

func (s *LoadingScene) Load() error {
	if mon := ebiten.Monitor(); mon != nil {
		mw, mh := mon.Size()
		scale := mon.DeviceScaleFactor()
		s.width = int(float64(mw) * scale)
		s.height = int(float64(mh) * scale)
	}

	s.frame = 0
	s.frameTick = 0

	// Start async resource loading
	if s.loadFunc != nil {
		s.loadFunc(s.BaseScene)
	}

	return nil
}

func (s *LoadingScene) Unload() error { return nil }

func (s *LoadingScene) Update() error {
	// Animate
	s.frameTick++

	if s.frameTick >= 10 { // ~6 FPS animation
		s.frameTick = 0

		if s.frameCount > 0 {
			s.frame = (s.frame + 1) % s.frameCount
		}
	}

	// Wait at least 30 frames (0.5s) and until loading completes
	if s.frameTick == 0 && s.frame > 2 && !s.Resources.IsLoading() {
		s.switchScene(s.targetScene)
	}

	return nil
}

func (s *LoadingScene) Draw(screen *ebiten.Image) {
	w := float32(s.width)
	h := float32(s.height)

	// Dark background
	ui.DrawRoundedRect(screen, 0, 0, w, h, 0, color.RGBA{R: 0x0A, G: 0x0A, B: 0x12, A: 0xFF})

	// Draw animated frame if spritesheet is ready
	if s.sheet != nil && s.frameCount > 0 {
		frameH := float64(s.sheet.Bounds().Dy())
		cx := float64(w)/2 - float64(s.frameW)/2
		cy := float64(h)/2 - frameH/2

		DrawSpriteFrame(screen, s.sheet, s.frameW, s.frame, cx, cy)
	}

	// Progress bar
	loaded, total := s.Resources.Progress()

	if total > 0 {
		progress := float32(loaded) / float32(total)
		barW := ui.S(200)
		barH := ui.S(6)
		barX := (w - barW) / 2
		barY := h * 0.7

		ui.DrawRoundedRect(screen, barX, barY, barW, barH, ui.S(3),
			color.RGBA{R: 0x20, G: 0x20, B: 0x30, A: 0xFF})
		ui.DrawRoundedRect(screen, barX, barY, barW*progress, barH, ui.S(3),
			ui.ColorAccentBreak)

		// Progress text
		faceSmall := ui.Face(false, 11)
		text := fmt.Sprintf("Loading... %d/%d", loaded, total)
		ui.DrawTextCentered(screen, text, faceSmall, float64(w/2), float64(barY+barH+ui.S(12)),
			ui.ColorTextSecond)
	}

	// Title
	faceTitle := ui.Face(true, 18)
	ui.DrawTextCentered(screen, "Fingerprint Lab", faceTitle, float64(w/2), float64(h*0.25),
		ui.ColorTextPrimary)
}

func (s *LoadingScene) Layout(outsideWidth, outsideHeight int) (int, int) {
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

// SetSpriteSheet sets the animation spritesheet built from loading frames.
func (s *LoadingScene) SetSpriteSheet(sheet *ebiten.Image, frameW, frameCount int) {
	s.sheet = sheet
	s.frameW = frameW
	s.frameCount = frameCount
}
