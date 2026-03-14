package scene

import (
	"context"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// callbackScene tracks whether callbacks fire correctly through switch cycles.
type callbackScene struct {
	name       string
	loadCB     func()
	switchedTo int
}

func (s *callbackScene) Name() string           { return s.name }
func (s *callbackScene) Init(_ context.Context) {}
func (s *callbackScene) Load() error {
	if s.loadCB != nil {
		s.loadCB()
	}

	return nil
}

func (s *callbackScene) Unload() error              { return nil }
func (s *callbackScene) Update() error              { return nil }
func (s *callbackScene) Draw(_ *ebiten.Image)       {}
func (s *callbackScene) Layout(_, _ int) (int, int) { return 0, 0 }

func TestCallbackSurvivesMultipleSwitches(t *testing.T) {
	m := NewManager()

	switchCount := 0
	doSwitch := func(name string) {
		switchCount++
		_ = m.SwitchSceneTo(name)
	}

	timer := &callbackScene{name: "timer"}
	settings := &callbackScene{name: "settings"}

	// Simulate: timer's button callback calls doSwitch("settings")
	timer.loadCB = func() { timer.switchedTo++ }
	settings.loadCB = func() { settings.switchedTo++ }

	m.Add(context.Background(), timer, settings)

	// Initial switch to timer
	_ = m.SwitchSceneTo("timer")

	if timer.switchedTo != 1 {
		t.Fatalf("expected timer loadCB called once, got %d", timer.switchedTo)
	}

	// Switch timer → settings (simulating button click)
	doSwitch("settings")
	if settings.switchedTo != 1 {
		t.Fatal("settings loadCB not called")
	}

	// Switch settings → timer (simulating back button)
	doSwitch("timer")
	if timer.switchedTo != 2 {
		t.Fatalf("expected timer loadCB called twice, got %d", timer.switchedTo)
	}

	// Do 10 more round-trips
	for range 10 {
		doSwitch("settings")
		doSwitch("timer")
	}

	if timer.switchedTo != 12 {
		t.Fatalf("expected 12 timer loads, got %d", timer.switchedTo)
	}

	if settings.switchedTo != 11 {
		t.Fatalf("expected 11 settings loads, got %d", settings.switchedTo)
	}

	if switchCount != 22 {
		t.Fatalf("expected 22 switches, got %d", switchCount)
	}
}

func TestSceneSwitchFromCallbackDuringUpdate(t *testing.T) {
	// Simulates: during scene.Update(), a button callback triggers SwitchSceneTo
	m := NewManager()

	settingsLoads := 0

	settings := &callbackScene{
		name:   "settings",
		loadCB: func() { settingsLoads++ },
	}

	timer := &callbackScene{name: "timer"}

	m.Add(context.Background(), timer, settings)
	_ = m.SwitchSceneTo("timer")

	// Simulate: timer's Update runs, user clicks settings button
	current := m.Scene()
	if current.Name() != "timer" {
		t.Fatal("expected timer")
	}

	_ = current.Update()

	// Button callback fires → switch to settings
	if err := m.SwitchSceneTo("settings"); err != nil {
		t.Fatal(err)
	}

	if m.Scene().Name() != "settings" {
		t.Fatalf("expected settings, got %s", m.Scene().Name())
	}

	if settingsLoads != 1 {
		t.Fatalf("expected 1 settings load, got %d", settingsLoads)
	}

	// Next frame: settings is the current scene
	_ = m.Scene().Update()

	// Back button → switch to timer
	if err := m.SwitchSceneTo("timer"); err != nil {
		t.Fatal(err)
	}

	if m.Scene().Name() != "timer" {
		t.Fatalf("expected timer, got %s", m.Scene().Name())
	}
}
