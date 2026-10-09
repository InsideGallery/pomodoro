package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/assets"
	"github.com/InsideGallery/pomodoro/pkg/app"
	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/platform"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
	"github.com/InsideGallery/pomodoro/services/pomodoro/internal/builtin"
	"github.com/InsideGallery/pomodoro/services/pomodoro/internal/modules/mini"
	"github.com/InsideGallery/pomodoro/services/pomodoro/internal/modules/settings"
	timerscene "github.com/InsideGallery/pomodoro/services/pomodoro/internal/modules/timer"
	"github.com/InsideGallery/pomodoro/services/pomodoro/internal/tray"
)

func main() {
	// System tray
	tray.SetIcon(tray.GenerateIcon(32, color.RGBA{R: 0x8B, G: 0x8B, B: 0x9E, A: 0xFF}))

	go tray.Run()

	game := app.New(app.Config{
		Width:          380,
		Height:         560,
		Title:          "Pomodoro",
		Transparent:    true,
		DragEnabled:    true,
		HandleWinClose: func() { platform.HideWindow("Pomodoro") },
		OnTick:         onTick,
		Setup:          setupPomodoro,
	})

	ebiten.SetWindowSize(380, 560)
	ebiten.SetWindowTitle("Pomodoro")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowDecorated(false)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowClosingHandled(true)

	if icon, err := png.Decode(bytes.NewReader(assets.AppIcon)); err == nil {
		ebiten.SetWindowIcon([]image.Image{icon})
	}

	op := &ebiten.RunGameOptions{ScreenTransparent: true}
	if err := ebiten.RunGameWithOptions(game, op); err != nil {
		log.Fatal(err)
	}
}

var (
	timerScene    *timerscene.Scene
	sceneManager  *scene.Manager
	trayIconState string
)

func onTick() error {
	if err := processTray(); err != nil {
		return err
	}

	if timerScene != nil {
		timerScene.Advance()
		syncTrayIcon(timerScene.TimerStateString())
	}

	return nil
}

func setupPomodoro(ctx context.Context, bus *event.Bus, manager *scene.Manager, switchScene func(string)) string {
	// Core scenes
	ts := timerscene.NewScene(bus, switchScene,
		func() { platform.HideWindow("Pomodoro") }, // X button → hide to tray
		func() { switchScene("mini") },
	)
	timerScene = ts
	sceneManager = manager

	mn := mini.NewScene(ts, func() {
		switchScene("timer")

		ebiten.SetWindowSize(380, 560)
	})

	// Plugins: minigame, lockscreen, metrics (NOT fingerprint)
	plugins := builtin.Modules()

	for _, mod := range plugins {
		scenes := mod.Scenes(bus, pluggable.SceneSwitcher(switchScene))

		for _, sc := range scenes {
			manager.Add(ctx, sc)
		}

		cfg := config.Load()

		for label, sceneName := range mod.TrayItems() {
			name := sceneName
			key := mod.ConfigKey()

			if cfg.PluginEnabled(key, mod.DefaultEnabled()) {
				tray.AddMenuItem(label, func() {
					switchScene(name)
				})
			}
		}
	}

	// Settings scene (receives plugins for dynamic toggles)
	ss := settings.NewScene(bus, switchScene, plugins)

	manager.Add(ctx, ts, ss, mn)

	return "timer"
}

func processTray() error {
	select {
	case action := <-tray.ActionCh:
		switch action {
		case tray.ActionShow:
			platform.ShowWindow("Pomodoro")
			platform.RaiseWindow("Pomodoro")

			// ShowWindow puts the window back on the taskbar; the mini window stays off it.
			if sceneManager != nil && sceneManager.Scene() != nil && sceneManager.Scene().Name() == mini.SceneName {
				platform.SetSkipTaskbar("Pomodoro", true)
			}
		case tray.ActionQuit:
			tray.Quit()
			os.Exit(0)
		}
	default:
	}

	return nil
}

// syncTrayIcon colours the tray icon after the timer state. It runs every frame,
// so the icon also follows a state restored at startup, which fires no event.
func syncTrayIcon(state string) {
	if state == trayIconState {
		return
	}

	trayIconState = state

	clr := color.RGBA{R: 0x8B, G: 0x8B, B: 0x9E, A: 0xFF}

	switch state {
	case "Focus":
		clr = color.RGBA{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF}
	case "Break":
		clr = color.RGBA{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF}
	case "Long Break":
		clr = color.RGBA{R: 0x81, G: 0xEC, B: 0xEC, A: 0xFF}
	case "Paused":
		clr = color.RGBA{R: 0xFF, G: 0xC1, B: 0x07, A: 0xFF}
	}

	tray.UpdateIcon(tray.GenerateIcon(32, clr))
}
