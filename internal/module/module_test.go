package module

import (
	"testing"

	"github.com/InsideGallery/pomodoro/internal/event"
)

type stubModule struct {
	id          string
	enabled     bool
	initialized bool
	bus         *event.Bus
}

func (m *stubModule) ID() string { return m.id }

func (m *stubModule) Init(bus *event.Bus) {
	m.initialized = true
	m.bus = bus
}

func (m *stubModule) Enabled() bool { return m.enabled }

func TestNewRegistry(t *testing.T) {
	bus := event.NewBus()
	r := NewRegistry(bus)

	if r == nil {
		t.Fatal("expected non-nil registry")
	}

	if len(r.Modules()) != 0 {
		t.Fatalf("expected 0 modules, got %d", len(r.Modules()))
	}
}

func TestRegisterCallsInit(t *testing.T) {
	bus := event.NewBus()
	r := NewRegistry(bus)
	m := &stubModule{id: "test", enabled: true}

	r.Register(m)

	if !m.initialized {
		t.Fatal("expected Init to be called")
	}

	if m.bus != bus {
		t.Fatal("expected bus to be passed to Init")
	}
}

func TestModulesReturnsAll(t *testing.T) {
	bus := event.NewBus()
	r := NewRegistry(bus)

	r.Register(&stubModule{id: "a", enabled: true})
	r.Register(&stubModule{id: "b", enabled: false})
	r.Register(&stubModule{id: "c", enabled: true})

	mods := r.Modules()
	if len(mods) != 3 {
		t.Fatalf("expected 3 modules, got %d", len(mods))
	}

	if mods[0].ID() != "a" || mods[1].ID() != "b" || mods[2].ID() != "c" {
		t.Fatal("modules not in registration order")
	}
}

func TestByIDFound(t *testing.T) {
	bus := event.NewBus()
	r := NewRegistry(bus)

	r.Register(&stubModule{id: "alpha", enabled: true})
	r.Register(&stubModule{id: "beta", enabled: true})

	m := r.ByID("beta")
	if m == nil {
		t.Fatal("expected to find module 'beta'")
	}

	if m.ID() != "beta" {
		t.Fatalf("expected 'beta', got %q", m.ID())
	}
}

func TestByIDNotFound(t *testing.T) {
	bus := event.NewBus()
	r := NewRegistry(bus)

	r.Register(&stubModule{id: "alpha", enabled: true})

	m := r.ByID("nonexistent")
	if m != nil {
		t.Fatal("expected nil for nonexistent module")
	}
}

func TestModuleReceivesEvents(t *testing.T) {
	bus := event.NewBus()
	r := NewRegistry(bus)

	received := false
	m := &stubModule{id: "listener", enabled: true}

	r.Register(m)

	m.bus.Subscribe(event.FocusStarted, func(_ event.Event) {
		received = true
	})

	bus.Publish(event.Event{Type: event.FocusStarted})

	if !received {
		t.Fatal("module did not receive event through bus")
	}
}
