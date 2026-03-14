//go:build plugin

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DailyMetrics holds aggregated metrics for a single day.
type DailyMetrics struct {
	Date              string  `json:"date"` // YYYY-MM-DD
	FocusSec          float64 `json:"focus_sec"`
	BreakSec          float64 `json:"break_sec"`
	GameSec           float64 `json:"game_sec"`
	FocusStarted      int     `json:"focus_started"`
	BreaksStarted     int     `json:"breaks_started"`
	GamesStarted      int     `json:"games_started"`
	LongBreaksIgnored int     `json:"long_breaks_ignored"`
	RelaxedBreaks     int     `json:"relaxed_breaks"` // breaks completed without skip
}

// Summary holds aggregated metrics for a time period.
type Summary struct {
	FocusHours        float64
	BreakHours        float64
	GameHours         float64
	FocusStarted      int
	BreaksStarted     int
	GamesStarted      int
	LongBreaksIgnored int
	RelaxedBreaks     int
}

// Store persists daily metrics to a JSON file.
type Store struct {
	mu   sync.Mutex
	Days []DailyMetrics `json:"days"`
	path string
}

// NewStore creates a store backed by the given file path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// DefaultPath returns the default metrics file path.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", "pomodoro", "metrics.json")
}

// maxReasonableSec is the maximum reasonable duration per day (48 hours).
const maxReasonableSec = 48 * 3600

// Load reads metrics from disk and sanitizes corrupted values.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		s.Days = nil

		return nil //nolint:nilerr // missing file is not an error
	}

	if err := json.Unmarshal(data, &s.Days); err != nil {
		return err
	}

	// Sanitize corrupted entries (e.g., from zero-time elapsed bugs)
	for i := range s.Days {
		if s.Days[i].FocusSec > maxReasonableSec {
			s.Days[i].FocusSec = 0
		}

		if s.Days[i].BreakSec > maxReasonableSec {
			s.Days[i].BreakSec = 0
		}

		if s.Days[i].GameSec > maxReasonableSec {
			s.Days[i].GameSec = 0
		}
	}

	return nil
}

// Save writes metrics to disk.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.Days, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o600)
}

// today returns or creates the DailyMetrics for today.
func (s *Store) today() *DailyMetrics {
	todayStr := time.Now().Format("2006-01-02")

	if len(s.Days) > 0 && s.Days[len(s.Days)-1].Date == todayStr {
		return &s.Days[len(s.Days)-1]
	}

	s.Days = append(s.Days, DailyMetrics{Date: todayStr})

	return &s.Days[len(s.Days)-1]
}

// RecordFocusStart increments the focus started counter.
func (s *Store) RecordFocusStart() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.today().FocusStarted++
}

// RecordFocusDuration adds focus time in seconds.
func (s *Store) RecordFocusDuration(secs float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.today().FocusSec += secs
}

// RecordBreakStart increments the break started counter.
func (s *Store) RecordBreakStart() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.today().BreaksStarted++
}

// RecordBreakDuration adds break time in seconds.
func (s *Store) RecordBreakDuration(secs float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.today().BreakSec += secs
}

// RecordRelaxedBreak increments breaks completed without skip.
func (s *Store) RecordRelaxedBreak() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.today().RelaxedBreaks++
}

// RecordGameStart increments the games started counter.
func (s *Store) RecordGameStart() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.today().GamesStarted++
}

// RecordGameDuration adds game time in seconds.
func (s *Store) RecordGameDuration(secs float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.today().GameSec += secs
}

// RecordLongBreakIgnored increments the long break ignored counter.
func (s *Store) RecordLongBreakIgnored() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.today().LongBreaksIgnored++
}

// Total returns aggregated metrics for all time.
func (s *Store) Total() Summary {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.aggregate(s.Days)
}

// Weekly returns aggregated metrics for the current week (Mon-Sun).
func (s *Store) Weekly() Summary {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	weekday := now.Weekday()

	if weekday == time.Sunday {
		weekday = 7
	}

	weekStart := now.AddDate(0, 0, -int(weekday-time.Monday)).Format("2006-01-02")

	var filtered []DailyMetrics

	for _, d := range s.Days {
		if d.Date >= weekStart {
			filtered = append(filtered, d)
		}
	}

	return s.aggregate(filtered)
}

// Monthly returns aggregated metrics for the current month.
func (s *Store) Monthly() Summary {
	s.mu.Lock()
	defer s.mu.Unlock()

	monthStart := time.Now().Format("2006-01") + "-01"

	var filtered []DailyMetrics

	for _, d := range s.Days {
		if d.Date >= monthStart {
			filtered = append(filtered, d)
		}
	}

	return s.aggregate(filtered)
}

func (s *Store) aggregate(days []DailyMetrics) Summary {
	var sum Summary

	for _, d := range days {
		sum.FocusHours += d.FocusSec / 3600
		sum.BreakHours += d.BreakSec / 3600
		sum.GameHours += d.GameSec / 3600
		sum.FocusStarted += d.FocusStarted
		sum.BreaksStarted += d.BreaksStarted
		sum.GamesStarted += d.GamesStarted
		sum.LongBreaksIgnored += d.LongBreaksIgnored
		sum.RelaxedBreaks += d.RelaxedBreaks
	}

	return sum
}
