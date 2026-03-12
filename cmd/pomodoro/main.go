package main

import (
	"log"

	"github.com/InsideGallery/pomodoro/internal/app"
	"github.com/InsideGallery/pomodoro/internal/tray"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// Start system tray in background
	tray.SetIcon(tray.GenerateIcon(32))
	go tray.Run()

	game := app.New()

	ebiten.SetWindowSize(app.DefaultWindowWidth, app.DefaultWindowHeight)
	ebiten.SetWindowTitle("Pomodoro")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowDecorated(false)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowClosingHandled(true)

	op := &ebiten.RunGameOptions{
		ScreenTransparent: true,
	}
	if err := ebiten.RunGameWithOptions(game, op); err != nil {
		log.Fatal(err)
	}
}
