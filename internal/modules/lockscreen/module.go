package lockscreen

import (
	"time"

	"github.com/InsideGallery/pomodoro/internal/ui"
	"github.com/InsideGallery/pomodoro/pkg/event"
)

// Module implements the module.Module interface for the long-break lock screen.
type Module struct {
	lock     Lock
	screen   Screen
	breakDur time.Duration
}

// NewModule creates a lock screen module with the given long break duration.
func NewModule(breakDur time.Duration) *Module {
	return &Module{breakDur: breakDur}
}

func (m *Module) ID() string    { return "lockscreen" }
func (m *Module) Enabled() bool { return true }
func (m *Module) Active() bool  { return m.lock.Active() }
func (m *Module) Lock() *Lock   { return &m.lock }

func (m *Module) Init(bus *event.Bus) {
	bus.Subscribe(event.LongBreakStarted, func(_ event.Event) {
		m.lock.Start(m.breakDur, time.Now())
	})

	bus.Subscribe(event.LongBreakCompleted, func(_ event.Event) {
		m.lock.Stop()
	})

	bus.Subscribe(event.Reset, func(_ event.Event) {
		m.lock.Stop()
	})
}

// Screen returns the Ebiten screen for this module.
func (m *Module) Screen() ui.Screen { //nolint:ireturn // required by screenProvider interface
	m.screen.module = m

	return &m.screen
}

// SetBreakDur updates the break duration (e.g. when config changes).
func (m *Module) SetBreakDur(d time.Duration) {
	m.breakDur = d
}
