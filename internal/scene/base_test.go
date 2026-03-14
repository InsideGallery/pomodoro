package scene

import (
	"context"
	"testing"

	"github.com/InsideGallery/pomodoro/internal/event"
)

func TestNewBaseScene(t *testing.T) {
	bus := event.NewBus()
	b := NewBaseScene(context.Background(), bus)

	if b.Systems == nil {
		t.Fatal("expected non-nil Systems")
	}

	if b.Registry == nil {
		t.Fatal("expected non-nil Registry")
	}

	if b.RTree == nil {
		t.Fatal("expected non-nil RTree")
	}

	if b.Bus != bus {
		t.Fatal("expected Bus to match")
	}
}

func TestBaseSceneUpdateEmpty(t *testing.T) {
	bus := event.NewBus()
	b := NewBaseScene(context.Background(), bus)

	if err := b.Update(); err != nil {
		t.Fatalf("update on empty scene should not error: %v", err)
	}
}

func TestBaseSceneRegistryOperations(t *testing.T) {
	bus := event.NewBus()
	b := NewBaseScene(context.Background(), bus)

	// Add entity to registry
	if err := b.Registry.Add("targets", 1, "entity1"); err != nil {
		t.Fatalf("registry add: %v", err)
	}

	val, err := b.Registry.Get("targets", 1)
	if err != nil {
		t.Fatalf("registry get: %v", err)
	}

	if val != "entity1" {
		t.Fatalf("expected entity1, got %v", val)
	}
}

func TestBaseSceneEventBus(t *testing.T) {
	bus := event.NewBus()
	b := NewBaseScene(context.Background(), bus)

	received := false

	b.Bus.Subscribe(event.FocusStarted, func(_ event.Event) {
		received = true
	})

	bus.Publish(event.Event{Type: event.FocusStarted})

	if !received {
		t.Fatal("scene should receive events through bus")
	}
}
