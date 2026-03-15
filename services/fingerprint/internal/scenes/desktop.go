package scenes

import (
	"context"
	"image/color"
	"log/slog"
	"math"
	"os"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/resources"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/pkg/systems"
	"github.com/InsideGallery/pomodoro/pkg/ui"
)

const DesktopSceneName = "fingerprint_desktop"

// CRT screen area as percentage of the 8328x4320 background image.
// Measured from the actual asset: the lit screen area inside the monitor bezel.
const (
	crtLeft   = 0.265 // left edge of screen area
	crtTop    = 0.095 // top edge
	crtRight  = 0.735 // right edge
	crtBottom = 0.875 // bottom edge
)

type DesktopScene struct {
	*scene.BaseScene

	input       *systems.InputSystem
	switchScene func(string)

	bgDim     *ebiten.Image // powered-off screen
	bgBright  *ebiten.Image // powered-on screen
	wallpaper *ebiten.Image
	appIcon   *ebiten.Image
	cursor    *ebiten.Image

	enabled  bool // interactive after boot animation
	bootTick int

	// Computed layout
	bgScale    float64
	bgOffsetX  float64
	bgOffsetY  float64
	screenRect [4]float64 // x, y, w, h of CRT screen area in window coords

	width, height int
}

func NewDesktopScene(switchScene func(string)) *DesktopScene {
	return &DesktopScene{switchScene: switchScene}
}

func (s *DesktopScene) Name() string { return DesktopSceneName }

func (s *DesktopScene) SetResources(rm *resources.Manager) {
	if s.BaseScene != nil {
		s.BaseScene.Resources = rm
	}
}

func (s *DesktopScene) Init(ctx context.Context) {
	s.BaseScene = scene.NewBaseScene(ctx, nil)
	s.input = systems.NewInputSystem(s.RTree)
}

func (s *DesktopScene) Load() error {
	if mon := ebiten.Monitor(); mon != nil {
		mw, mh := mon.Size()
		scale := mon.DeviceScaleFactor()
		s.width = int(float64(mw) * scale)
		s.height = int(float64(mh) * scale)
	}

	s.enabled = false
	s.bootTick = 0

	s.bgDim, _ = s.Resources.GetImage("bg_dim")
	s.bgBright, _ = s.Resources.GetImage("bg_bright")
	s.wallpaper, _ = s.Resources.GetImage("wallpaper")
	s.appIcon, _ = s.Resources.GetImage("app_icon")
	s.cursor, _ = s.Resources.GetImage("cursor")

	slog.Info("desktop loaded",
		"dim", s.bgDim != nil, "bright", s.bgBright != nil,
		"wallpaper", s.wallpaper != nil, "icon", s.appIcon != nil,
		"cursor", s.cursor != nil)

	s.computeLayout()
	s.registerZones()

	ebiten.SetFullscreen(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	return nil
}

func (s *DesktopScene) Unload() error {
	ebiten.SetCursorMode(ebiten.CursorModeVisible)

	return nil
}

func (s *DesktopScene) computeLayout() {
	w := float64(s.width)
	h := float64(s.height)

	// Scale background to FIT (preserve aspect ratio, letterbox)
	bgW, bgH := 8328.0, 4320.0

	scaleX := w / bgW
	scaleY := h / bgH
	s.bgScale = scaleX

	if scaleY < scaleX {
		s.bgScale = scaleY
	}

	scaledW := bgW * s.bgScale
	scaledH := bgH * s.bgScale
	s.bgOffsetX = (w - scaledW) / 2
	s.bgOffsetY = (h - scaledH) / 2

	// CRT screen area in window coordinates
	s.screenRect = [4]float64{
		s.bgOffsetX + crtLeft*scaledW,
		s.bgOffsetY + crtTop*scaledH,
		(crtRight - crtLeft) * scaledW,
		(crtBottom - crtTop) * scaledH,
	}
}

func (s *DesktopScene) registerZones() {
	s.input.ClearZones()

	sx, sy, sw, sh := s.screenRect[0], s.screenRect[1], s.screenRect[2], s.screenRect[3]

	// App icon: top-left of screen area, with padding
	iconSize := sh * 0.12
	iconX := sx + sw*0.05
	iconY := sy + sh*0.05

	s.input.AddZone(&systems.Zone{
		Spatial: shapes.NewBox(shapes.NewPoint(iconX, iconY), iconSize, iconSize+20),
		OnClick: func() {
			if s.enabled {
				slog.Info("opening fingerprint app")
				s.switchScene(AppSceneName)
			}
		},
	})

	// Quit button: bottom-right of screen area
	quitW := sw * 0.08
	quitH := sh * 0.04
	quitX := sx + sw - quitW - sw*0.02
	quitY := sy + sh - quitH - sh*0.02

	s.input.AddZone(&systems.Zone{
		Spatial: shapes.NewBox(shapes.NewPoint(quitX, quitY), quitW, quitH),
		OnClick: func() {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
			os.Exit(0)
		},
	})
}

func (s *DesktopScene) Update() error {
	if !s.enabled {
		s.bootTick++

		if s.bootTick > 90 { // ~1.5 seconds boot animation
			s.enabled = true
		}

		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
		os.Exit(0)
	}

	if err := s.input.Update(s.Ctx); err != nil {
		return err
	}

	return nil
}

func (s *DesktopScene) Draw(screen *ebiten.Image) {
	// Black letterbox background
	screen.Fill(color.RGBA{A: 0xFF})

	bootProgress := float64(s.bootTick) / 90.0
	if bootProgress > 1 {
		bootProgress = 1
	}

	// Phase 1 (0-0.5): Show dim/off screen
	// Phase 2 (0.5-1.0): Fade in bright screen + wallpaper
	if bootProgress < 0.5 {
		// Show powered-off monitor
		s.drawBG(screen, s.bgDim, 1.0)
	} else {
		// Cross-fade to powered-on monitor
		fade := (bootProgress - 0.5) * 2 // 0→1

		s.drawBG(screen, s.bgDim, 1.0-fade)
		s.drawBG(screen, s.bgBright, fade)

		// Wallpaper fades in
		if s.wallpaper != nil && fade > 0.1 {
			sx, sy, sw, sh := s.screenRect[0], s.screenRect[1], s.screenRect[2], s.screenRect[3]
			op := &ebiten.DrawImageOptions{}
			ww := float64(s.wallpaper.Bounds().Dx())
			wh := float64(s.wallpaper.Bounds().Dy())
			op.GeoM.Scale(sw/ww, sh/wh)
			op.GeoM.Translate(sx, sy)
			op.ColorScale.Scale(float32(fade), float32(fade), float32(fade), 1)
			screen.DrawImage(s.wallpaper, op)
		}
	}

	// Desktop content (only when boot complete)
	if s.enabled {
		sx, sy, _, sh := s.screenRect[0], s.screenRect[1], s.screenRect[2], s.screenRect[3]

		// App icon
		if s.appIcon != nil {
			iconSize := sh * 0.12
			iconX := sx + s.screenRect[2]*0.05
			iconY := sy + sh*0.05

			op := &ebiten.DrawImageOptions{}
			iw := float64(s.appIcon.Bounds().Dx())
			scale := iconSize / iw
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(iconX, iconY)
			screen.DrawImage(s.appIcon, op)

			// Label
			face := ui.Face(false, 10)
			ui.DrawTextCentered(screen, "Fingerprinting", face,
				iconX+iconSize/2, iconY+iconSize+5, ui.ColorTextPrimary)
		}

		// Quit label (bottom-right)
		face := ui.Face(false, 11)
		quitX := sx + s.screenRect[2]*0.88
		quitY := sy + sh*0.93
		ui.DrawText(screen, "Quit", face, quitX, quitY, ui.ColorTextSecond)
	}

	// Custom cursor (always on top)
	if s.cursor != nil && s.enabled {
		mx, my := ebiten.CursorPosition()
		op := &ebiten.DrawImageOptions{}
		cw := float64(s.cursor.Bounds().Dx())
		scale := 32.0 / cw
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(float64(mx), float64(my))
		screen.DrawImage(s.cursor, op)
	}
}

func (s *DesktopScene) drawBG(screen *ebiten.Image, img *ebiten.Image, alpha float64) {
	if img == nil || alpha <= 0 {
		return
	}

	op := &ebiten.DrawImageOptions{}
	bw := float64(img.Bounds().Dx())
	bh := float64(img.Bounds().Dy())

	// Scale to fit, centered
	scale := float64(s.width) / bw
	scaleY := float64(s.height) / bh

	if scaleY < scale {
		scale = scaleY
	}

	op.GeoM.Scale(scale, scale)
	scaledW := bw * scale
	scaledH := bh * scale
	op.GeoM.Translate((float64(s.width)-scaledW)/2, (float64(s.height)-scaledH)/2)
	op.ColorScale.Scale(float32(alpha), float32(alpha), float32(alpha), float32(alpha))
	screen.DrawImage(img, op)
}

func (s *DesktopScene) Layout(outsideWidth, outsideHeight int) (int, int) {
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
		s.computeLayout()
		s.registerZones()
	}

	return w, h
}
