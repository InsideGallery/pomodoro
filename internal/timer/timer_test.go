package timer

import (
	"testing"
	"time"
)

func testConfig() Config {
	return Config{
		FocusDuration:     25 * time.Second,
		BreakDuration:     5 * time.Second,
		LongBreakDuration: 15 * time.Second,
		RoundsBeforeLong:  4,
		AutoStart:         false,
	}
}

func TestNewTimer(t *testing.T) {
	tm := New(testConfig())
	if tm.State() != StateIdle {
		t.Fatalf("expected Idle, got %s", tm.State())
	}

	if tm.Round() != 0 {
		t.Fatalf("expected round 0, got %d", tm.Round())
	}

	if tm.PendingNext() != StateFocus {
		t.Fatalf("expected pending Focus, got %s", tm.PendingNext())
	}
}

func TestStartTransition(t *testing.T) {
	tm := New(testConfig())
	now := time.Now()
	tm.Start(now)

	if tm.State() != StateFocus {
		t.Fatalf("expected Focus, got %s", tm.State())
	}
}

func TestStartIgnoredWhenNotIdle(t *testing.T) {
	tm := New(testConfig())
	now := time.Now()
	tm.Start(now)
	tm.Start(now) // should be no-op

	if tm.State() != StateFocus {
		t.Fatalf("expected Focus, got %s", tm.State())
	}
}

func TestRemaining(t *testing.T) {
	tm := New(testConfig())
	now := time.Now()
	tm.Start(now)

	rem := tm.Remaining(now.Add(10 * time.Second))
	if rem != 15*time.Second {
		t.Fatalf("expected 15s remaining, got %s", rem)
	}
}

func TestProgress(t *testing.T) {
	tm := New(testConfig())
	now := time.Now()
	tm.Start(now)

	p := tm.Progress(now)
	if p != 0 {
		t.Fatalf("expected progress 0, got %f", p)
	}

	p = tm.Progress(now.Add(25 * time.Second))
	if p != 1 {
		t.Fatalf("expected progress 1, got %f", p)
	}
}

func TestUpdateCompletes(t *testing.T) {
	var completed State

	tm := New(testConfig())
	tm.OnComplete = func(s State) { completed = s }
	now := time.Now()
	tm.Start(now)
	tm.Update(now.Add(26 * time.Second))

	if completed != StateFocus {
		t.Fatalf("expected OnComplete with Focus, got %s", completed)
	}

	if tm.State() != StateIdle {
		t.Fatalf("expected Idle after complete (autostart off), got %s", tm.State())
	}

	if tm.Round() != 1 {
		t.Fatalf("expected round 1, got %d", tm.Round())
	}
}

func TestPendingNextAfterFocusCompletes(t *testing.T) {
	tm := New(testConfig())
	now := time.Now()
	tm.Start(now)
	tm.Update(now.Add(26 * time.Second))
	// After focus completes without autostart, pending should be Break
	if tm.PendingNext() != StateBreak {
		t.Fatalf("expected pending Break, got %s", tm.PendingNext())
	}
	// Start should begin Break, not Focus
	tm.Start(now.Add(30 * time.Second))

	if tm.State() != StateBreak {
		t.Fatalf("expected Break after start, got %s", tm.State())
	}
}

func TestPendingNextAfterBreakCompletes(t *testing.T) {
	cfg := testConfig()
	cfg.AutoStart = true
	tm := New(cfg)
	now := time.Now()
	tm.Start(now)
	// Focus completes -> auto-start break
	now = now.Add(26 * time.Second)
	tm.Update(now)

	if tm.State() != StateBreak {
		t.Fatalf("expected Break, got %s", tm.State())
	}
	// Now disable autostart and let break complete
	cfg.AutoStart = false
	tm.SetConfig(cfg)

	now = now.Add(6 * time.Second)
	tm.Update(now)

	if tm.State() != StateIdle {
		t.Fatalf("expected Idle after break, got %s", tm.State())
	}

	if tm.PendingNext() != StateFocus {
		t.Fatalf("expected pending Focus after break, got %s", tm.PendingNext())
	}
}

func TestAutoStart(t *testing.T) {
	cfg := testConfig()
	cfg.AutoStart = true
	tm := New(cfg)
	now := time.Now()
	tm.Start(now)
	tm.Update(now.Add(26 * time.Second))

	if tm.State() != StateBreak {
		t.Fatalf("expected Break after auto-start, got %s", tm.State())
	}
}

func TestPauseResume(t *testing.T) {
	tm := New(testConfig())
	now := time.Now()
	tm.Start(now)

	pauseAt := now.Add(10 * time.Second)
	tm.Pause(pauseAt)

	if tm.State() != StatePaused {
		t.Fatalf("expected Paused, got %s", tm.State())
	}

	rem := tm.Remaining(pauseAt)
	if rem != 15*time.Second {
		t.Fatalf("expected 15s remaining when paused, got %s", rem)
	}

	// remaining should not change while paused
	rem2 := tm.Remaining(pauseAt.Add(5 * time.Second))
	if rem2 != 15*time.Second {
		t.Fatalf("remaining should not change while paused, got %s", rem2)
	}

	resumeAt := pauseAt.Add(30 * time.Second)
	tm.Resume(resumeAt)

	if tm.State() != StateFocus {
		t.Fatalf("expected Focus after resume, got %s", tm.State())
	}

	rem3 := tm.Remaining(resumeAt)
	if rem3 != 15*time.Second {
		t.Fatalf("expected 15s after resume, got %s", rem3)
	}
}

func TestReset(t *testing.T) {
	cfg := testConfig()
	cfg.AutoStart = true
	tm := New(cfg)
	now := time.Now()
	tm.Start(now)
	tm.Update(now.Add(26 * time.Second)) // complete focus, auto-start break
	tm.Reset()

	if tm.State() != StateIdle {
		t.Fatalf("expected Idle after reset, got %s", tm.State())
	}

	if tm.Round() != 0 {
		t.Fatalf("expected round 0 after reset, got %d", tm.Round())
	}

	if tm.PendingNext() != StateFocus {
		t.Fatalf("expected pending Focus after reset, got %s", tm.PendingNext())
	}
}

func TestLongBreakAfterNRounds(t *testing.T) {
	cfg := testConfig()
	cfg.AutoStart = true
	cfg.RoundsBeforeLong = 2
	tm := New(cfg)
	now := time.Now()

	// Round 1: focus -> break
	tm.Start(now)
	now = now.Add(26 * time.Second)
	tm.Update(now)

	if tm.State() != StateBreak {
		t.Fatalf("expected Break after round 1, got %s", tm.State())
	}

	// Break completes -> focus
	now = now.Add(6 * time.Second)
	tm.Update(now)

	if tm.State() != StateFocus {
		t.Fatalf("expected Focus after break, got %s", tm.State())
	}

	// Round 2: focus -> long break
	now = now.Add(26 * time.Second)
	tm.Update(now)

	if tm.State() != StateLongBreak {
		t.Fatalf("expected LongBreak after round 2, got %s", tm.State())
	}
}

func TestSkipFocusStartsBreak(t *testing.T) {
	tm := New(testConfig())
	now := time.Now()
	tm.Start(now)
	tm.Skip(now.Add(5 * time.Second))
	// Skip focus without autostart -> idle with pending Break
	if tm.State() != StateIdle {
		t.Fatalf("expected Idle after skip, got %s", tm.State())
	}

	if tm.PendingNext() != StateBreak {
		t.Fatalf("expected pending Break after skip, got %s", tm.PendingNext())
	}

	if tm.Round() != 1 {
		t.Fatalf("expected round incremented on skip, got %d", tm.Round())
	}
	// Now Start should begin Break
	tm.Start(now.Add(10 * time.Second))

	if tm.State() != StateBreak {
		t.Fatalf("expected Break started, got %s", tm.State())
	}
}

func TestSkipWhilePaused(t *testing.T) {
	cfg := testConfig()
	cfg.AutoStart = true
	tm := New(cfg)
	now := time.Now()
	tm.Start(now)
	tm.Pause(now.Add(5 * time.Second))
	tm.Skip(now.Add(10 * time.Second))

	if tm.State() != StateBreak {
		t.Fatalf("expected Break after skip-while-paused with autostart, got %s", tm.State())
	}
}

func TestPauseIgnoredWhenIdle(t *testing.T) {
	tm := New(testConfig())
	tm.Pause(time.Now())

	if tm.State() != StateIdle {
		t.Fatalf("pause on idle should be no-op, got %s", tm.State())
	}
}

func TestResumeIgnoredWhenNotPaused(t *testing.T) {
	tm := New(testConfig())
	now := time.Now()
	tm.Start(now)
	tm.Resume(now) // should be no-op

	if tm.State() != StateFocus {
		t.Fatalf("resume on non-paused should be no-op, got %s", tm.State())
	}
}

func TestFullCycleNoAutoStart(t *testing.T) {
	tm := New(testConfig())
	now := time.Now()

	// Start focus
	tm.Start(now)

	if tm.State() != StateFocus {
		t.Fatalf("expected Focus, got %s", tm.State())
	}

	// Focus completes
	now = now.Add(26 * time.Second)
	tm.Update(now)

	if tm.State() != StateIdle {
		t.Fatalf("expected Idle, got %s", tm.State())
	}

	// Start break
	now = now.Add(time.Second)
	tm.Start(now)

	if tm.State() != StateBreak {
		t.Fatalf("expected Break, got %s", tm.State())
	}

	// Break completes
	now = now.Add(6 * time.Second)
	tm.Update(now)

	if tm.State() != StateIdle {
		t.Fatalf("expected Idle after break, got %s", tm.State())
	}

	// Start next focus
	now = now.Add(time.Second)
	tm.Start(now)

	if tm.State() != StateFocus {
		t.Fatalf("expected Focus again, got %s", tm.State())
	}
}
