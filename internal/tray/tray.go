package tray

import (
	"github.com/getlantern/systray"
)

// Actions sent from tray to the app.
type Action int

const (
	ActionShow Action = iota
	ActionMetrics
	ActionQuit
)

var (
	ActionCh = make(chan Action, 4)
	icon     []byte
	ready    bool
)

// SetIcon sets the tray icon data (PNG bytes) before Run.
func SetIcon(data []byte) {
	icon = data
}

// UpdateIcon changes the tray icon at runtime.
func UpdateIcon(data []byte) {
	icon = data

	if ready {
		systray.SetIcon(data)
	}
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
	ready = true

	if len(icon) > 0 {
		systray.SetIcon(icon)
	}

	systray.SetTitle("Pomodoro")
	systray.SetTooltip("Pomodoro Timer")

	mShow := systray.AddMenuItem("Show", "Show the timer window")
	mMetrics := systray.AddMenuItem("Metrics", "Show usage statistics")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				ActionCh <- ActionShow
			case <-mMetrics.ClickedCh:
				ActionCh <- ActionMetrics
			case <-mQuit.ClickedCh:
				ActionCh <- ActionQuit
			}
		}
	}()
}

func onExit() {}
