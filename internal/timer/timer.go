package timer

import "time"

type State uint8

const (
	StateIdle State = iota
	StateFocus
	StateBreak
	StateLongBreak
	StatePaused
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateFocus:
		return "Focus"
	case StateBreak:
		return "Break"
	case StateLongBreak:
		return "Long Break"
	case StatePaused:
		return "Paused"
	default:
		return "Unknown"
	}
}

func (s State) IsRunning() bool {
	return s == StateFocus || s == StateBreak || s == StateLongBreak
}

type Config struct {
	FocusDuration     time.Duration
	BreakDuration     time.Duration
	LongBreakDuration time.Duration
	RoundsBeforeLong  int
	AutoStart         bool
}

func DefaultConfig() Config {
	return Config{
		FocusDuration:     25 * time.Minute,
		BreakDuration:     5 * time.Minute,
		LongBreakDuration: 15 * time.Minute,
		RoundsBeforeLong:  4,
		AutoStart:         false,
	}
}

type Timer struct {
	cfg Config

	state       State
	prePause    State // state before pause, for resume
	pendingNext State // what Start() will begin (Break after focus, Focus after break)
	startedAt   time.Time
	remaining   time.Duration // set on pause
	round       int           // completed focus rounds in current cycle
	OnComplete  func(completed State)
}

func New(cfg Config) *Timer {
	return &Timer{
		cfg:         cfg,
		state:       StateIdle,
		pendingNext: StateFocus,
	}
}

func (t *Timer) State() State       { return t.state }
func (t *Timer) Round() int         { return t.round }
func (t *Timer) Config() Config     { return t.cfg }
func (t *Timer) PendingNext() State { return t.pendingNext }
func (t *Timer) SetConfig(c Config) { t.cfg = c }

func (t *Timer) duration() time.Duration {
	switch t.state {
	case StateFocus:
		return t.cfg.FocusDuration
	case StateBreak:
		return t.cfg.BreakDuration
	case StateLongBreak:
		return t.cfg.LongBreakDuration
	default:
		return 0
	}
}

func (t *Timer) Remaining(now time.Time) time.Duration {
	switch t.state {
	case StatePaused:
		return t.remaining
	case StateFocus, StateBreak, StateLongBreak:
		r := t.duration() - now.Sub(t.startedAt)
		if r < 0 {
			return 0
		}

		return r
	default:
		return 0
	}
}

func (t *Timer) TotalDuration() time.Duration {
	if t.state == StatePaused {
		return t.durationForState(t.prePause)
	}

	if t.state == StateIdle {
		return t.durationForState(t.pendingNext)
	}

	return t.duration()
}

func (t *Timer) durationForState(s State) time.Duration {
	switch s {
	case StateFocus:
		return t.cfg.FocusDuration
	case StateBreak:
		return t.cfg.BreakDuration
	case StateLongBreak:
		return t.cfg.LongBreakDuration
	default:
		return 0
	}
}

func (t *Timer) Progress(now time.Time) float64 {
	total := t.TotalDuration()
	if total == 0 {
		return 0
	}

	rem := t.Remaining(now)

	p := 1.0 - float64(rem)/float64(total)
	if p < 0 {
		return 0
	}

	if p > 1 {
		return 1
	}

	return p
}

// Start begins the pending next state (Focus initially, then Break/LongBreak after focus).
func (t *Timer) Start(now time.Time) {
	if t.state != StateIdle {
		return
	}

	t.state = t.pendingNext
	t.startedAt = now
	t.remaining = 0
}

func (t *Timer) Pause(now time.Time) {
	if !t.state.IsRunning() {
		return
	}

	t.remaining = t.Remaining(now)
	t.prePause = t.state
	t.state = StatePaused
}

func (t *Timer) Resume(now time.Time) {
	if t.state != StatePaused {
		return
	}

	t.state = t.prePause
	t.startedAt = now.Add(-t.duration() + t.remaining)
	t.remaining = 0
}

func (t *Timer) Reset() {
	t.state = StateIdle
	t.round = 0
	t.remaining = 0
	t.pendingNext = StateFocus
}

func (t *Timer) Skip(now time.Time) {
	if t.state == StateIdle {
		return
	}

	current := t.state
	if t.state == StatePaused {
		current = t.prePause
	}

	t.complete(current, now)
}

func (t *Timer) Update(now time.Time) {
	if !t.state.IsRunning() {
		return
	}

	if t.Remaining(now) <= 0 {
		t.complete(t.state, now)
	}
}

func (t *Timer) complete(completed State, now time.Time) {
	if completed == StateFocus {
		t.round++
	}

	if t.OnComplete != nil {
		t.OnComplete(completed)
	}

	next := t.nextState(completed)
	if t.cfg.AutoStart && next != StateIdle {
		t.state = next
		t.startedAt = now
		t.remaining = 0
		t.pendingNext = StateFocus // default for after this auto-started phase
	} else {
		t.state = StateIdle
		t.pendingNext = next
	}
}

func (t *Timer) nextState(completed State) State {
	switch completed {
	case StateFocus:
		if t.round >= t.cfg.RoundsBeforeLong {
			t.round = 0
			return StateLongBreak
		}

		return StateBreak
	case StateBreak, StateLongBreak:
		return StateFocus
	default:
		return StateIdle
	}
}
