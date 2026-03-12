package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	FocusMinutes     int     `json:"focus_minutes"`
	BreakMinutes     int     `json:"break_minutes"`
	LongBreakMinutes int     `json:"long_break_minutes"`
	RoundsBeforeLong int     `json:"rounds_before_long"`
	AutoStart        bool    `json:"auto_start"`
	TickVolume       float64 `json:"tick_volume"`
	AlarmVolume      float64 `json:"alarm_volume"`
	TickEnabled      bool    `json:"tick_enabled"`
	Theme            string  `json:"theme"`
	Transparency     float64 `json:"transparency"`
}

func (c Config) FocusDuration() time.Duration { return time.Duration(c.FocusMinutes) * time.Minute }
func (c Config) BreakDuration() time.Duration { return time.Duration(c.BreakMinutes) * time.Minute }
func (c Config) LongBreakDuration() time.Duration {
	return time.Duration(c.LongBreakMinutes) * time.Minute
}

func Default() Config {
	return Config{
		FocusMinutes:     25,
		BreakMinutes:     5,
		LongBreakMinutes: 15,
		RoundsBeforeLong: 4,
		AutoStart:        false,
		TickVolume:       0.5,
		AlarmVolume:      0.8,
		TickEnabled:      true,
		Theme:            "dark",
		Transparency:     0.10,
	}
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "pomodoro"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.json"), nil
}

func Load() Config {
	p, err := configPath()
	if err != nil {
		return Default()
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return Default()
	}

	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default()
	}

	// Migrate old nanosecond-based config: any value > 1440 (24h in minutes)
	// is likely nanoseconds from the old format.
	if cfg.FocusMinutes > 1440 {
		cfg.FocusMinutes = int(time.Duration(cfg.FocusMinutes) / time.Minute)
	}

	if cfg.BreakMinutes > 1440 {
		cfg.BreakMinutes = int(time.Duration(cfg.BreakMinutes) / time.Minute)
	}

	if cfg.LongBreakMinutes > 1440 {
		cfg.LongBreakMinutes = int(time.Duration(cfg.LongBreakMinutes) / time.Minute)
	}

	return cfg
}

func Save(cfg Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, data, 0o600)
}

// TimerState represents the persisted timer state between restarts.
type TimerState struct {
	Round        int     `json:"round"`
	PendingNext  string  `json:"pending_next"`
	State        string  `json:"state"`
	PrePause     string  `json:"pre_pause"`
	RemainingSec float64 `json:"remaining_sec"`
}

func statePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "state.json"), nil
}

func LoadState() TimerState {
	p, err := statePath()
	if err != nil {
		return TimerState{}
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return TimerState{}
	}

	var s TimerState
	if err := json.Unmarshal(data, &s); err != nil {
		return TimerState{}
	}

	return s
}

func SaveState(s TimerState) error {
	p, err := statePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, data, 0o600)
}
