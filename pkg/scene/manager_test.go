package scene

import (
	"context"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type testScene struct {
	name      string
	inited    bool
	loaded    bool
	unloaded  bool
	updateErr error
}

func (s *testScene) Name() string               { return s.name }
func (s *testScene) Init(_ context.Context)     { s.inited = true }
func (s *testScene) Load() error                { s.loaded = true; return nil }
func (s *testScene) Unload() error              { s.unloaded = true; return nil }
func (s *testScene) Update() error              { return s.updateErr }
func (s *testScene) Draw(_ *ebiten.Image)       {}
func (s *testScene) Layout(_, _ int) (int, int) { return 0, 0 }

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("expected non-nil manager")
	}

	if m.Scene() != nil {
		t.Fatal("expected nil current scene")
	}
}

func TestAddCallsInit(t *testing.T) {
	m := NewManager()
	sc := &testScene{name: "test"}

	m.Add(context.Background(), sc)

	if !sc.inited {
		t.Fatal("expected Init to be called")
	}
}

func TestSwitchSceneTo(t *testing.T) {
	m := NewManager()
	sc := &testScene{name: "timer"}

	m.Add(context.Background(), sc)

	if err := m.SwitchSceneTo("timer"); err != nil {
		t.Fatalf("switch: %v", err)
	}

	if m.Scene() != sc {
		t.Fatal("expected current scene to be timer")
	}

	if !sc.loaded {
		t.Fatal("expected Load to be called")
	}
}

func TestSwitchSceneUnloadsPrevious(t *testing.T) {
	m := NewManager()
	sc1 := &testScene{name: "timer"}
	sc2 := &testScene{name: "settings"}

	m.Add(context.Background(), sc1, sc2)

	if err := m.SwitchSceneTo("timer"); err != nil {
		t.Fatal(err)
	}

	if err := m.SwitchSceneTo("settings"); err != nil {
		t.Fatal(err)
	}

	if !sc1.unloaded {
		t.Fatal("expected previous scene to be unloaded")
	}

	if m.Scene() != sc2 {
		t.Fatal("expected current scene to be settings")
	}
}

func TestSwitchSceneNotFound(t *testing.T) {
	m := NewManager()

	err := m.SwitchSceneTo("nonexistent")
	if err != ErrSceneNotFound {
		t.Fatalf("expected ErrSceneNotFound, got %v", err)
	}
}

func TestMultipleScenes(t *testing.T) {
	m := NewManager()

	m.Add(context.Background(),
		&testScene{name: "timer"},
		&testScene{name: "settings"},
		&testScene{name: "minigame"},
	)

	for _, name := range []string{"timer", "settings", "minigame", "timer"} {
		if err := m.SwitchSceneTo(name); err != nil {
			t.Fatalf("switch to %s: %v", name, err)
		}

		if m.Scene().Name() != name {
			t.Fatalf("expected %s, got %s", name, m.Scene().Name())
		}
	}
}
