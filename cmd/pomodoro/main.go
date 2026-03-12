package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/assets"
	"github.com/InsideGallery/pomodoro/internal/app"
	"github.com/InsideGallery/pomodoro/internal/tray"
)

func main() {
	// Start system tray in background
	tray.SetIcon(tray.GenerateIcon(32, color.RGBA{R: 0x8B, G: 0x8B, B: 0x9E, A: 0xFF}))

	go tray.Run()

	game := app.New()

	ebiten.SetWindowSize(app.DefaultWindowWidth, app.DefaultWindowHeight)
	ebiten.SetWindowTitle("Pomodoro")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowDecorated(false)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowClosingHandled(true)

	if icon, err := png.Decode(bytes.NewReader(assets.AppIcon)); err == nil {
		ebiten.SetWindowIcon([]image.Image{icon})
	}

	op := &ebiten.RunGameOptions{
		ScreenTransparent: true,
	}
	if err := ebiten.RunGameWithOptions(game, op); err != nil {
		log.Fatal(err)
	}
}
