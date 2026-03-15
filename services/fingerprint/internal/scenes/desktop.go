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

// DesktopScene renders the retro CRT monitor desktop.
// Shows the "Muldrow Police Department" wallpaper and "Fingerprinting" app icon.
type DesktopScene struct {
	*scene.BaseScene

	input       *systems.InputSystem
	switchScene func(string)

	// Resources loaded by preloader
	bgImage   *ebiten.Image // CRT monitor background (fullscreen)
	wallpaper *ebiten.Image // desktop wallpaper
	appIcon   *ebiten.Image // Fingerprinting app icon
	cursor    *ebiten.Image // custom cursor

	enabled  bool // false during boot animation, true when interactive
	bootTick int

	width, height int
}

func NewDesktopScene(switchScene func(string)) *DesktopScene {
	return &DesktopScene{
		switchScene: switchScene,
	}
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

	// Pull loaded resources
	s.bgImage, _ = s.Resources.GetImage("bg_static")
	s.wallpaper, _ = s.Resources.GetImage("wallpaper")
	s.appIcon, _ = s.Resources.GetImage("app_icon")
	s.cursor, _ = s.Resources.GetImage("cursor")

	slog.Info("desktop resources",
		"bg", s.bgImage != nil,
		"wallpaper", s.wallpaper != nil,
		"icon", s.appIcon != nil,
		"cursor", s.cursor != nil)

	s.registerIcon()

	ebiten.SetFullscreen(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden) // we draw custom cursor

	return nil
}

func (s *DesktopScene) Unload() error {
	ebiten.SetCursorMode(ebiten.CursorModeVisible)

	return nil
}

func (s *DesktopScene) registerIcon() {
	s.input.ClearZones()

	if s.appIcon == nil {
		return
	}

	// Icon position: top-left area of the CRT screen
	iconW := 80.0
	iconH := 80.0
	iconX := float64(s.width)*0.28 + 30
	iconY := float64(s.height)*0.15 + 30

	s.input.AddZone(&systems.Zone{
		Spatial: shapes.NewBox(shapes.NewPoint(iconX, iconY), iconW, iconH),
		OnClick: func() {
			if s.enabled {
				s.switchScene(AppSceneName)
			}
		},
	})
}

func (s *DesktopScene) Update() error {
	// Boot animation: screen turns on over ~60 frames (1 second)
	if !s.enabled {
		s.bootTick++

		if s.bootTick > 60 {
			s.enabled = true
		}

		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		ebiten.SetFullscreen(false)
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
		os.Exit(0)
	}

	if err := s.input.Update(s.Ctx); err != nil {
		return err
	}

	return nil
}

func (s *DesktopScene) Draw(screen *ebiten.Image) {
	w := float64(s.width)
	h := float64(s.height)

	// CRT monitor background (fills entire screen)
	if s.bgImage != nil {
		op := &ebiten.DrawImageOptions{}
		bw := float64(s.bgImage.Bounds().Dx())
		bh := float64(s.bgImage.Bounds().Dy())
		op.GeoM.Scale(w/bw, h/bh)
		screen.DrawImage(s.bgImage, op)
	} else {
		screen.Fill(color.RGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xFF})
	}

	// Boot animation: screen brightness fades in
	brightness := 1.0

	if !s.enabled {
		brightness = float64(s.bootTick) / 60.0
		if brightness > 1 {
			brightness = 1
		}
	}

	// Screen content area (inside the CRT bezel)
	// The CRT screen area is roughly centered, ~60% of the image
	screenX := w * 0.22
	screenY := h * 0.08
	screenW := w * 0.56
	screenH := h * 0.78

	// Draw wallpaper on the CRT screen area
	if s.wallpaper != nil && brightness > 0.01 {
		op := &ebiten.DrawImageOptions{}
		ww := float64(s.wallpaper.Bounds().Dx())
		wh := float64(s.wallpaper.Bounds().Dy())
		op.GeoM.Scale(screenW/ww, screenH/wh)
		op.GeoM.Translate(screenX, screenY)
		op.ColorScale.Scale(float32(brightness), float32(brightness), float32(brightness), 1)
		screen.DrawImage(s.wallpaper, op)
	}

	// Draw app icon
	if s.appIcon != nil && s.enabled {
		iconX := screenX + 30
		iconY := screenY + 30
		op := &ebiten.DrawImageOptions{}
		iw := float64(s.appIcon.Bounds().Dx())

		scale := 80.0 / iw
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(iconX, iconY)
		screen.DrawImage(s.appIcon, op)

		// Label under icon
		face := ui.Face(false, 10)
		ui.DrawTextCentered(screen, "Fingerprinting", face, iconX+40, iconY+85, ui.ColorTextPrimary)
	}

	// Custom cursor (drawn last, on top)
	if s.cursor != nil && s.enabled {
		mx, my := ebiten.CursorPosition()
		op := &ebiten.DrawImageOptions{}
		cw := float64(s.cursor.Bounds().Dx())

		cursorScale := 32.0 / cw
		op.GeoM.Scale(cursorScale, cursorScale)
		op.GeoM.Translate(float64(mx), float64(my))
		screen.DrawImage(s.cursor, op)
	}
}

func (s *DesktopScene) Layout(outsideWidth, outsideHeight int) (int, int) {
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
