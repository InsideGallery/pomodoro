package lockscreen

import (
	"testing"
	"time"
)

func TestStartActivates(t *testing.T) {
	l := &Lock{}
	l.Start(15*time.Minute, time.Now())

	if !l.Active() {
		t.Fatal("expected active after start")
	}
}

func TestNotActiveByDefault(t *testing.T) {
	l := &Lock{}
	if l.Active() {
		t.Fatal("expected not active by default")
	}
}

func TestCompleteBeforeDuration(t *testing.T) {
	now := time.Now()
	l := &Lock{}
	l.Start(15*time.Minute, now)

	if l.Complete(now.Add(10 * time.Minute)) {
		t.Fatal("should not be complete at 10 minutes")
	}
}

func TestCompleteAtDuration(t *testing.T) {
	now := time.Now()
	l := &Lock{}
	l.Start(15*time.Minute, now)

	if !l.Complete(now.Add(15 * time.Minute)) {
		t.Fatal("should be complete at 15 minutes")
	}
}

func TestCompleteAfterDuration(t *testing.T) {
	now := time.Now()
	l := &Lock{}
	l.Start(15*time.Minute, now)

	if !l.Complete(now.Add(20 * time.Minute)) {
		t.Fatal("should be complete after 20 minutes")
	}
}

func TestCompleteWhenNotActive(t *testing.T) {
	l := &Lock{}

	if l.Complete(time.Now()) {
		t.Fatal("should not be complete when not active")
	}
}

func TestRemaining(t *testing.T) {
	now := time.Now()
	l := &Lock{}
	l.Start(15*time.Minute, now)

	rem := l.Remaining(now.Add(5 * time.Minute))
	if rem != 10*time.Minute {
		t.Fatalf("expected 10m remaining, got %s", rem)
	}

	rem = l.Remaining(now.Add(20 * time.Minute))
	if rem != 0 {
		t.Fatalf("expected 0 remaining, got %s", rem)
	}
}

func TestProgress(t *testing.T) {
	now := time.Now()
	l := &Lock{}
	l.Start(10*time.Minute, now)

	p := l.Progress(now)
	if p != 0 {
		t.Fatalf("expected progress 0, got %f", p)
	}

	p = l.Progress(now.Add(5 * time.Minute))
	if p != 0.5 {
		t.Fatalf("expected progress 0.5, got %f", p)
	}

	p = l.Progress(now.Add(10 * time.Minute))
	if p != 1 {
		t.Fatalf("expected progress 1.0, got %f", p)
	}

	p = l.Progress(now.Add(15 * time.Minute))
	if p != 1 {
		t.Fatalf("expected progress clamped to 1.0, got %f", p)
	}
}

func TestProgressZeroDuration(t *testing.T) {
	l := &Lock{}
	l.Start(0, time.Now())

	if l.Progress(time.Now()) != 0 {
		t.Fatal("expected 0 progress for zero duration")
	}
}

func TestStop(t *testing.T) {
	l := &Lock{}
	l.Start(15*time.Minute, time.Now())
	l.Stop()

	if l.Active() {
		t.Fatal("expected not active after stop")
	}
}
