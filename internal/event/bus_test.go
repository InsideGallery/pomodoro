package event

import (
	"sync"
	"testing"
	"time"
)

func TestNewBus(t *testing.T) {
	b := NewBus()
	if b == nil {
		t.Fatal("expected non-nil bus")
	}
}

func TestSubscribeAndPublish(t *testing.T) {
	b := NewBus()

	var received Event

	b.Subscribe(FocusStarted, func(e Event) {
		received = e
	})

	now := time.Now()
	b.Publish(Event{Type: FocusStarted, Time: now})

	if received.Type != FocusStarted {
		t.Fatalf("expected FocusStarted, got %s", received.Type)
	}

	if !received.Time.Equal(now) {
		t.Fatalf("expected time %v, got %v", now, received.Time)
	}
}

func TestPublishNoSubscribers(_ *testing.T) {
	b := NewBus()
	// Should not panic
	b.Publish(Event{Type: FocusStarted, Time: time.Now()})
}

func TestMultipleSubscribers(t *testing.T) {
	b := NewBus()
	count := 0

	b.Subscribe(BreakStarted, func(_ Event) { count++ })
	b.Subscribe(BreakStarted, func(_ Event) { count++ })
	b.Subscribe(BreakStarted, func(_ Event) { count++ })

	b.Publish(Event{Type: BreakStarted, Time: time.Now()})

	if count != 3 {
		t.Fatalf("expected 3 handlers called, got %d", count)
	}
}

func TestSubscribersIsolatedByType(t *testing.T) {
	b := NewBus()
	focusCount := 0
	breakCount := 0

	b.Subscribe(FocusStarted, func(_ Event) { focusCount++ })
	b.Subscribe(BreakStarted, func(_ Event) { breakCount++ })

	b.Publish(Event{Type: FocusStarted, Time: time.Now()})

	if focusCount != 1 {
		t.Fatalf("expected focusCount 1, got %d", focusCount)
	}

	if breakCount != 0 {
		t.Fatalf("expected breakCount 0, got %d", breakCount)
	}
}

func TestAllEventTypes(t *testing.T) {
	types := []Type{
		FocusStarted, FocusCompleted,
		BreakStarted, BreakCompleted,
		LongBreakStarted, LongBreakCompleted,
		Paused, Resumed, Reset, Tick,
	}

	b := NewBus()
	received := make(map[Type]bool)

	for _, et := range types {
		et := et
		b.Subscribe(et, func(e Event) {
			received[e.Type] = true
		})
	}

	for _, et := range types {
		b.Publish(Event{Type: et, Time: time.Now()})
	}

	for _, et := range types {
		if !received[et] {
			t.Errorf("event type %s not received", et)
		}
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		t    Type
		want string
	}{
		{FocusStarted, "FocusStarted"},
		{FocusCompleted, "FocusCompleted"},
		{BreakStarted, "BreakStarted"},
		{BreakCompleted, "BreakCompleted"},
		{LongBreakStarted, "LongBreakStarted"},
		{LongBreakCompleted, "LongBreakCompleted"},
		{Paused, "Paused"},
		{Resumed, "Resumed"},
		{Reset, "Reset"},
		{Tick, "Tick"},
		{Type(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("Type(%d).String() = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestConcurrentPublish(t *testing.T) {
	b := NewBus()

	var mu sync.Mutex

	count := 0

	b.Subscribe(Tick, func(_ Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			b.Publish(Event{Type: Tick, Time: time.Now()})
		}()
	}

	wg.Wait()

	if count != 100 {
		t.Fatalf("expected 100, got %d", count)
	}
}

func TestConcurrentSubscribeAndPublish(_ *testing.T) {
	b := NewBus()

	var wg sync.WaitGroup

	// Subscribe concurrently
	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			b.Subscribe(FocusStarted, func(_ Event) {})
		}()
	}

	// Publish concurrently
	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			b.Publish(Event{Type: FocusStarted, Time: time.Now()})
		}()
	}

	wg.Wait()
}
