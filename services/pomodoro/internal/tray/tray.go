package tray

import (
	"sync"

	"github.com/getlantern/systray"
)

// Actions sent from tray to the app.
type Action int

const (
	ActionShow Action = iota
	ActionQuit
)

type menuItem struct {
	label   string
	onClick func()
}

var (
	ActionCh    = make(chan Action, 4) //nolint:gochecknoglobals // tray state
	icon        []byte                 //nolint:gochecknoglobals // tray state
	ready       bool                   //nolint:gochecknoglobals // tray state
	extraItems  []menuItem             //nolint:gochecknoglobals // plugin menu items
	extraItemMu sync.Mutex             //nolint:gochecknoglobals // tray state
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

// AddMenuItem registers a custom menu item.
// Can be called before or after Run — items are added dynamically.
func AddMenuItem(label string, onClick func()) {
	extraItemMu.Lock()
	defer extraItemMu.Unlock()

	extraItems = append(extraItems, menuItem{label: label, onClick: onClick})

	// If tray is already running, add the item live
	if ready {
		mi := systray.AddMenuItem(label, label)

		go func() {
			for range mi.ClickedCh {
				onClick()
			}
		}()
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

	// Add plugin-registered menu items
	extraItemMu.Lock()
	items := make([]menuItem, len(extraItems))
	copy(items, extraItems)
	extraItemMu.Unlock()

	var pluginChans []chan struct{}

	for _, item := range items {
		mi := systray.AddMenuItem(item.label, item.label)
		pluginChans = append(pluginChans, mi.ClickedCh)
	}

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

	// Listen for plugin menu item clicks
	for i, ch := range pluginChans {
		fn := items[i].onClick

		go func(c chan struct{}) {
			for range c {
				fn()
			}
		}(ch)
	}
}

func onExit() {}
