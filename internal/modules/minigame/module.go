package minigame

import (
	"time"

	"github.com/InsideGallery/pomodoro/internal/event"
	"github.com/InsideGallery/pomodoro/internal/ui"
)

// Module implements the module.Module interface for the Button Hunt mini-game.
type Module struct {
	enabled   bool
	active    bool
	gameOver  bool
	game      Game
	screen    Screen
	breakDur  time.Duration
	bestScore int
	onSave    func(bestScore int) // callback to persist best score
}

// NewModule creates a mini-game module.
func NewModule(enabled bool, bestScore int, breakDur time.Duration, onSave func(int)) *Module {
	return &Module{
		enabled:   enabled,
		bestScore: bestScore,
		breakDur:  breakDur,
		onSave:    onSave,
	}
}

func (m *Module) ID() string        { return "minigame" }
func (m *Module) Enabled() bool     { return m.enabled }
func (m *Module) SetEnabled(v bool) { m.enabled = v }
func (m *Module) Active() bool      { return m.active }
func (m *Module) GameOver() bool    { return m.gameOver }
func (m *Module) Game() *Game       { return &m.game }

func (m *Module) Init(bus *event.Bus) {
	bus.Subscribe(event.BreakStarted, func(_ event.Event) {
		if !m.enabled {
			return
		}

		m.activate()
	})

	bus.Subscribe(event.BreakCompleted, func(_ event.Event) {
		m.finish()
	})

	bus.Subscribe(event.Reset, func(_ event.Event) {
		m.deactivate()
	})
}

func (m *Module) activate() {
	m.active = true
	m.gameOver = false
	// Actual game.Start() is called from screen.Init() once we know screen dimensions
}

func (m *Module) finish() {
	if !m.active {
		return
	}

	m.gameOver = true

	if m.game.BeatRecord() && m.onSave != nil {
		m.bestScore = m.game.Score
		m.onSave(m.bestScore)
	}
}

func (m *Module) deactivate() {
	m.active = false
	m.gameOver = false
}

// Dismiss closes the game-over or ESC screen and returns to the timer.
func (m *Module) Dismiss() {
	m.deactivate()
}

// Screen returns the Ebiten screen for this module.
func (m *Module) Screen() ui.Screen { //nolint:ireturn // required by screenProvider interface
	m.screen.module = m
	return &m.screen
}

// BreakDur returns the configured break duration for starting the game.
func (m *Module) BreakDur() time.Duration { return m.breakDur }

// BestScore returns the current best score.
func (m *Module) BestScore() int { return m.bestScore }
