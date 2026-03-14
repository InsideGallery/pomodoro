package lockscreen

import "time"

// Lock holds the pure state for a long-break lock screen session.
type Lock struct {
	startedAt time.Time
	breakDur  time.Duration
	active    bool
}

// Start activates the lock screen for the given break duration.
func (l *Lock) Start(breakDur time.Duration, now time.Time) {
	l.startedAt = now
	l.breakDur = breakDur
	l.active = true
}

// Active returns whether the lock screen is currently active.
func (l *Lock) Active() bool { return l.active }

// Complete returns true when the break duration has elapsed.
func (l *Lock) Complete(now time.Time) bool {
	return l.active && now.Sub(l.startedAt) >= l.breakDur
}

// Remaining returns time left in the lock.
func (l *Lock) Remaining(now time.Time) time.Duration {
	rem := l.breakDur - now.Sub(l.startedAt)
	if rem < 0 {
		return 0
	}

	return rem
}

// Progress returns 0.0 to 1.0 indicating how far through the break we are.
func (l *Lock) Progress(now time.Time) float64 {
	if l.breakDur == 0 {
		return 0
	}

	p := float64(now.Sub(l.startedAt)) / float64(l.breakDur)

	if p < 0 {
		return 0
	}

	if p > 1 {
		return 1
	}

	return p
}

// Stop deactivates the lock screen.
func (l *Lock) Stop() {
	l.active = false
}
