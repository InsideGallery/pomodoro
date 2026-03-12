package tray

import (
	"github.com/getlantern/systray"
)

// Actions sent from tray to the app.
type Action int

const (
	ActionShow Action = iota
	ActionQuit
)

var (
	ActionCh = make(chan Action, 4)
	icon     []byte
)

// SetIcon sets the tray icon data (PNG bytes).
func SetIcon(data []byte) {
	icon = data
}

// Run starts the systray. Call from a goroutine — it blocks.
func Run() {
	systray.Run(onReady, onExit)
}

// Quit requests systray shutdown.
func Quit() {
	systray.Quit()
}

func onReady() {
	if len(icon) > 0 {
		systray.SetIcon(icon)
	}
	systray.SetTitle("Pomodoro")
	systray.SetTooltip("Pomodoro Timer")

	mShow := systray.AddMenuItem("Show", "Show the timer window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				ActionCh <- ActionShow
			case <-mQuit.ClickedCh:
				ActionCh <- ActionQuit
			}
		}
	}()
}

func onExit() {}
