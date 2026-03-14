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
	"github.com/InsideGallery/pomodoro/internal/app"
	"github.com/InsideGallery/pomodoro/internal/builtin"
	"github.com/InsideGallery/pomodoro/internal/modules/mini"
	"github.com/InsideGallery/pomodoro/internal/modules/settings"
	timerscene "github.com/InsideGallery/pomodoro/internal/modules/timer"
	"github.com/InsideGallery/pomodoro/internal/tray"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

func main() {
	// System tray
	tray.SetIcon(tray.GenerateIcon(32, color.RGBA{R: 0x8B, G: 0x8B, B: 0x9E, A: 0xFF}))

	go tray.Run()

	game := app.New(app.Config{
		Width:       380,
		Height:      560,
		Title:       "Pomodoro",
		Transparent: true,
		DragEnabled: true,
		Setup:       setupPomodoro,
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

func setupPomodoro(ctx context.Context, bus *event.Bus, manager *scene.Manager, switchScene func(string)) string {
	// Core scenes
	ts := timerscene.NewScene(bus, switchScene,
		func() { os.Exit(0) }, // close → exit (tray handles show/hide)
		func() { switchScene("mini") },
	)

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

		for label, sceneName := range mod.TrayItems() {
			name := sceneName

			tray.AddMenuItem(label, func() {
				switchScene(name)
			})
		}
	}

	// Settings scene (receives plugins for dynamic toggles)
	ss := settings.NewScene(bus, switchScene, plugins)

	manager.Add(ctx, ts, ss, mn)

	// Tray icon updates via timer events
	subscribeTrayIconUpdates(bus)

	return "timer"
}

func subscribeTrayIconUpdates(bus *event.Bus) {
	setIcon := func(clr color.RGBA) {
		tray.UpdateIcon(tray.GenerateIcon(32, clr))
	}

	for _, et := range []event.Type{
		event.FocusStarted, event.BreakStarted, event.LongBreakStarted,
		event.Paused, event.Resumed, event.Reset,
		event.FocusCompleted, event.BreakCompleted, event.LongBreakCompleted,
	} {
		bus.Subscribe(et, func(e event.Event) {
			if state, ok := e.Data.(string); ok {
				switch state {
				case "Focus":
					setIcon(color.RGBA{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF})
				case "Break":
					setIcon(color.RGBA{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF})
				case "Long Break":
					setIcon(color.RGBA{R: 0x81, G: 0xEC, B: 0xEC, A: 0xFF})
				case "Paused":
					setIcon(color.RGBA{R: 0xFF, G: 0xC1, B: 0x07, A: 0xFF})
				default:
					setIcon(color.RGBA{R: 0x8B, G: 0x8B, B: 0x9E, A: 0xFF})
				}
			}
		})
	}
}
