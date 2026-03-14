package scene

import (
	"context"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// lifecycleScene tracks every lifecycle call for verification.
type lifecycleScene struct {
	name    string
	initN   int
	loadN   int
	unloadN int
	updateN int
	drawN   int //nolint:unused // tracked for completeness
}

func (s *lifecycleScene) Name() string               { return s.name }
func (s *lifecycleScene) Init(_ context.Context)     { s.initN++ }
func (s *lifecycleScene) Load() error                { s.loadN++; return nil }
func (s *lifecycleScene) Unload() error              { s.unloadN++; return nil }
func (s *lifecycleScene) Update() error              { s.updateN++; return nil }
func (s *lifecycleScene) Draw(_ *ebiten.Image)       {}
func (s *lifecycleScene) Layout(_, _ int) (int, int) { return 0, 0 }

func TestSceneSwitchLifecycle(t *testing.T) {
	m := NewManager()
	timer := &lifecycleScene{name: "timer"}
	settings := &lifecycleScene{name: "settings"}

	m.Add(context.Background(), timer, settings)

	// Init is called once per scene on Add
	if timer.initN != 1 {
		t.Fatalf("timer init: expected 1, got %d", timer.initN)
	}

	if settings.initN != 1 {
		t.Fatalf("settings init: expected 1, got %d", settings.initN)
	}

	// Switch to timer
	if err := m.SwitchSceneTo("timer"); err != nil {
		t.Fatal(err)
	}

	if timer.loadN != 1 {
		t.Fatalf("timer load: expected 1, got %d", timer.loadN)
	}

	if m.Scene().Name() != "timer" {
		t.Fatalf("expected timer, got %s", m.Scene().Name())
	}

	// Switch to settings
	if err := m.SwitchSceneTo("settings"); err != nil {
		t.Fatal(err)
	}

	if settings.loadN != 1 {
		t.Fatalf("settings load: expected 1, got %d", settings.loadN)
	}

	if timer.unloadN != 1 {
		t.Fatalf("timer unload: expected 1, got %d", timer.unloadN)
	}

	if m.Scene().Name() != "settings" {
		t.Fatalf("expected settings, got %s", m.Scene().Name())
	}

	// Switch back to timer
	if err := m.SwitchSceneTo("timer"); err != nil {
		t.Fatal(err)
	}

	if timer.loadN != 2 {
		t.Fatalf("timer load: expected 2, got %d", timer.loadN)
	}

	if settings.unloadN != 1 {
		t.Fatalf("settings unload: expected 1, got %d", settings.unloadN)
	}

	if m.Scene().Name() != "timer" {
		t.Fatalf("expected timer, got %s", m.Scene().Name())
	}
}

func TestRapidSceneSwitching(t *testing.T) {
	m := NewManager()
	scenes := make([]*lifecycleScene, 4)

	for i := range scenes {
		names := []string{"timer", "settings", "minigame", "lockscreen"}
		scenes[i] = &lifecycleScene{name: names[i]}
	}

	m.Add(context.Background(), scenes[0], scenes[1], scenes[2], scenes[3])

	// Rapid switching 20 times
	order := []string{
		"timer", "settings", "timer", "minigame", "timer", "settings",
		"lockscreen", "timer", "settings", "timer", "minigame", "lockscreen",
		"timer", "settings", "timer", "minigame", "timer", "settings", "timer", "timer",
	}

	for _, name := range order {
		if err := m.SwitchSceneTo(name); err != nil {
			t.Fatalf("switch to %s: %v", name, err)
		}

		if m.Scene().Name() != name {
			t.Fatalf("expected %s, got %s", name, m.Scene().Name())
		}
	}

	// Verify total loads for timer (it's switched TO the most)
	if scenes[0].loadN < 5 {
		t.Fatalf("timer should have been loaded many times, got %d", scenes[0].loadN)
	}
}

func TestSwitchToSameScene(t *testing.T) {
	m := NewManager()
	sc := &lifecycleScene{name: "timer"}

	m.Add(context.Background(), sc)

	if err := m.SwitchSceneTo("timer"); err != nil {
		t.Fatal(err)
	}

	// Switch to the same scene again
	if err := m.SwitchSceneTo("timer"); err != nil {
		t.Fatal(err)
	}

	// Load should be called twice, unload once (unloads previous which is itself)
	if sc.loadN != 2 {
		t.Fatalf("expected 2 loads, got %d", sc.loadN)
	}

	if sc.unloadN != 1 {
		t.Fatalf("expected 1 unload, got %d", sc.unloadN)
	}
}

func TestSwitchDuringUpdate(t *testing.T) {
	// Simulates: scene's Update() triggers a scene switch
	m := NewManager()

	settings := &lifecycleScene{name: "settings"}
	timer := &lifecycleScene{name: "timer"}

	m.Add(context.Background(), timer, settings)
	_ = m.SwitchSceneTo("timer")

	// Simulate what happens when a button click in Update triggers a switch
	current := m.Scene()
	_ = current.Update() // timer's Update runs

	// During the "Update", the switch is requested
	if err := m.SwitchSceneTo("settings"); err != nil {
		t.Fatal(err)
	}

	// Now the current scene should be settings
	if m.Scene().Name() != "settings" {
		t.Fatalf("expected settings after mid-update switch, got %s", m.Scene().Name())
	}

	// The timer should have been unloaded
	if timer.unloadN != 1 {
		t.Fatalf("timer should be unloaded, got %d", timer.unloadN)
	}
}
