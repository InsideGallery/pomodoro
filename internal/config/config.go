package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	FocusDuration     time.Duration `json:"focus_duration"`
	BreakDuration     time.Duration `json:"break_duration"`
	LongBreakDuration time.Duration `json:"long_break_duration"`
	RoundsBeforeLong  int           `json:"rounds_before_long"`
	AutoStart         bool          `json:"auto_start"`
	TickVolume        float64       `json:"tick_volume"`
	AlarmVolume       float64       `json:"alarm_volume"`
	TickEnabled       bool          `json:"tick_enabled"`
	Theme             string        `json:"theme"`
	Transparency      float64       `json:"transparency"`
}

func Default() Config {
	return Config{
		FocusDuration:     25 * time.Minute,
		BreakDuration:     5 * time.Minute,
		LongBreakDuration: 15 * time.Minute,
		RoundsBeforeLong:  4,
		AutoStart:         false,
		TickVolume:        0.5,
		AlarmVolume:       0.8,
		TickEnabled:       true,
		Theme:             "dark",
		Transparency:      0.10,
	}
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pomodoro", "config.json"), nil
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

	return os.WriteFile(p, data, 0o644)
}
