package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestHome(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("HOME", dir)

	return dir
}

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.FocusMinutes != 25 {
		t.Fatalf("expected 25, got %d", cfg.FocusMinutes)
	}

	if cfg.BreakMinutes != 5 {
		t.Fatalf("expected 5, got %d", cfg.BreakMinutes)
	}

	if cfg.LongBreakMinutes != 15 {
		t.Fatalf("expected 15, got %d", cfg.LongBreakMinutes)
	}

	if cfg.RoundsBeforeLong != 4 {
		t.Fatalf("expected 4, got %d", cfg.RoundsBeforeLong)
	}

	if cfg.AutoStart {
		t.Fatal("expected AutoStart false")
	}

	if cfg.Theme != "dark" {
		t.Fatalf("expected dark, got %s", cfg.Theme)
	}
}

func TestDurations(t *testing.T) {
	cfg := Default()

	if cfg.FocusDuration().Minutes() != 25 {
		t.Fatalf("expected 25m, got %s", cfg.FocusDuration())
	}

	if cfg.BreakDuration().Minutes() != 5 {
		t.Fatalf("expected 5m, got %s", cfg.BreakDuration())
	}

	if cfg.LongBreakDuration().Minutes() != 15 {
		t.Fatalf("expected 15m, got %s", cfg.LongBreakDuration())
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	setupTestHome(t)

	cfg := Default()
	cfg.FocusMinutes = 30
	cfg.Theme = "light"
	cfg.TickVolume = 0.7

	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded := Load()

	if loaded.FocusMinutes != 30 {
		t.Fatalf("expected 30, got %d", loaded.FocusMinutes)
	}

	if loaded.Theme != "light" {
		t.Fatalf("expected light, got %s", loaded.Theme)
	}

	if loaded.TickVolume != 0.7 {
		t.Fatalf("expected 0.7, got %f", loaded.TickVolume)
	}
}

func TestLoadNonExistentReturnsDefault(t *testing.T) {
	setupTestHome(t)

	cfg := Load()
	def := Default()

	if cfg.FocusMinutes != def.FocusMinutes {
		t.Fatalf("expected default %d, got %d", def.FocusMinutes, cfg.FocusMinutes)
	}

	if cfg.Theme != def.Theme {
		t.Fatalf("expected default %s, got %s", def.Theme, cfg.Theme)
	}
}

func TestLoadCorruptedReturnsDefault(t *testing.T) {
	home := setupTestHome(t)

	dir := filepath.Join(home, ".config", "pomodoro")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{invalid json"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	def := Default()

	if cfg.FocusMinutes != def.FocusMinutes {
		t.Fatalf("expected default %d, got %d", def.FocusMinutes, cfg.FocusMinutes)
	}
}

func TestMigrationNanoseconds(t *testing.T) {
	home := setupTestHome(t)

	dir := filepath.Join(home, ".config", "pomodoro")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Old format stored nanoseconds as int (25 minutes = 1,500,000,000,000 ns)
	data := `{"focus_minutes":1500000000000,"break_minutes":300000000000,"long_break_minutes":900000000000}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Load()

	if cfg.FocusMinutes != 25 {
		t.Fatalf("expected migrated focus 25, got %d", cfg.FocusMinutes)
	}

	if cfg.BreakMinutes != 5 {
		t.Fatalf("expected migrated break 5, got %d", cfg.BreakMinutes)
	}

	if cfg.LongBreakMinutes != 15 {
		t.Fatalf("expected migrated long break 15, got %d", cfg.LongBreakMinutes)
	}
}

func TestSaveLoadStateRoundTrip(t *testing.T) {
	setupTestHome(t)

	st := TimerState{
		Round:        3,
		PendingNext:  "Break",
		State:        "Focus",
		PrePause:     "Idle",
		RemainingSec: 142.5,
	}

	if err := SaveState(st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	loaded := LoadState()

	if loaded.Round != 3 {
		t.Fatalf("expected round 3, got %d", loaded.Round)
	}

	if loaded.PendingNext != "Break" {
		t.Fatalf("expected Break, got %s", loaded.PendingNext)
	}

	if loaded.State != "Focus" {
		t.Fatalf("expected Focus, got %s", loaded.State)
	}

	if loaded.RemainingSec != 142.5 {
		t.Fatalf("expected 142.5, got %f", loaded.RemainingSec)
	}
}

func TestLoadStateNonExistent(t *testing.T) {
	setupTestHome(t)

	st := LoadState()

	if st.Round != 0 {
		t.Fatalf("expected round 0, got %d", st.Round)
	}

	if st.State != "" {
		t.Fatalf("expected empty state, got %s", st.State)
	}
}

func TestLoadStateCorrupted(t *testing.T) {
	home := setupTestHome(t)

	dir := filepath.Join(home, ".config", "pomodoro")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	st := LoadState()

	if st.Round != 0 {
		t.Fatalf("expected round 0, got %d", st.Round)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	setupTestHome(t)

	cfg := Default()
	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify file exists
	p, _ := configPath()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}
}

func TestSaveStateCreatesDirectory(t *testing.T) {
	setupTestHome(t)

	if err := SaveState(TimerState{Round: 1}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	p, _ := statePath()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Fatal("state file was not created")
	}
}
