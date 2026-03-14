//go:build plugin

package main //nolint:revive // metrics is our domain name, not runtime/metrics

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	dir := t.TempDir()

	return NewStore(filepath.Join(dir, "metrics.json"))
}

func TestNewStore(t *testing.T) {
	s := newTestStore(t)
	if s == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestLoadMissingFileReturnsNil(t *testing.T) {
	s := newTestStore(t)

	if err := s.Load(); err != nil {
		t.Fatalf("load missing file: %v", err)
	}

	if len(s.Days) != 0 {
		t.Fatalf("expected 0 days, got %d", len(s.Days))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	s := newTestStore(t)

	s.RecordFocusStart()
	s.RecordFocusStart()
	s.RecordFocusDuration(1500)
	s.RecordBreakStart()
	s.RecordBreakDuration(300)
	s.RecordGameStart()
	s.RecordGameDuration(120)
	s.RecordRelaxedBreak()
	s.RecordLongBreakIgnored()

	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	s2 := NewStore(s.path)

	if err := s2.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(s2.Days) != 1 {
		t.Fatalf("expected 1 day, got %d", len(s2.Days))
	}

	d := s2.Days[0]
	if d.FocusStarted != 2 {
		t.Fatalf("expected 2 focus started, got %d", d.FocusStarted)
	}

	if d.FocusSec != 1500 {
		t.Fatalf("expected 1500 focus sec, got %f", d.FocusSec)
	}

	if d.BreaksStarted != 1 {
		t.Fatalf("expected 1 break started, got %d", d.BreaksStarted)
	}

	if d.GamesStarted != 1 {
		t.Fatalf("expected 1 game started, got %d", d.GamesStarted)
	}

	if d.RelaxedBreaks != 1 {
		t.Fatalf("expected 1 relaxed break, got %d", d.RelaxedBreaks)
	}

	if d.LongBreaksIgnored != 1 {
		t.Fatalf("expected 1 long break ignored, got %d", d.LongBreaksIgnored)
	}
}

func TestTotalAggregation(t *testing.T) {
	s := newTestStore(t)

	s.Days = []DailyMetrics{
		{Date: "2026-03-10", FocusSec: 3600, FocusStarted: 2, BreaksStarted: 1},
		{Date: "2026-03-11", FocusSec: 7200, FocusStarted: 3, GamesStarted: 1, GameSec: 300},
		{Date: "2026-03-12", FocusSec: 1800, LongBreaksIgnored: 2, RelaxedBreaks: 1},
	}

	total := s.Total()

	if total.FocusHours != 3.5 {
		t.Fatalf("expected 3.5 focus hours, got %f", total.FocusHours)
	}

	if total.FocusStarted != 5 {
		t.Fatalf("expected 5 focus started, got %d", total.FocusStarted)
	}

	if total.GamesStarted != 1 {
		t.Fatalf("expected 1 game started, got %d", total.GamesStarted)
	}

	if total.LongBreaksIgnored != 2 {
		t.Fatalf("expected 2 long breaks ignored, got %d", total.LongBreaksIgnored)
	}

	if total.RelaxedBreaks != 1 {
		t.Fatalf("expected 1 relaxed break, got %d", total.RelaxedBreaks)
	}
}

func TestWeeklyFilters(t *testing.T) {
	s := newTestStore(t)

	s.Days = []DailyMetrics{
		{Date: "2020-01-01", FocusStarted: 100}, // old data
		{Date: "2026-03-14", FocusStarted: 5},   // today
	}

	weekly := s.Weekly()
	if weekly.FocusStarted != 5 {
		t.Fatalf("expected 5 (only this week), got %d", weekly.FocusStarted)
	}
}

func TestMonthlyFilters(t *testing.T) {
	s := newTestStore(t)

	s.Days = []DailyMetrics{
		{Date: "2026-02-15", FocusStarted: 100}, // last month
		{Date: "2026-03-01", FocusStarted: 3},
		{Date: "2026-03-14", FocusStarted: 7},
	}

	monthly := s.Monthly()
	if monthly.FocusStarted != 10 {
		t.Fatalf("expected 10 (only this month), got %d", monthly.FocusStarted)
	}
}

func TestRecordMultipleDays(t *testing.T) {
	s := newTestStore(t)

	// Simulate two records on same day
	s.RecordFocusStart()
	s.RecordFocusStart()

	if len(s.Days) != 1 {
		t.Fatalf("expected 1 day entry, got %d", len(s.Days))
	}

	if s.Days[0].FocusStarted != 2 {
		t.Fatalf("expected 2 focus started, got %d", s.Days[0].FocusStarted)
	}
}

func TestDefaultPath(t *testing.T) {
	p := DefaultPath()
	if p == "" {
		t.Fatal("expected non-empty default path")
	}
}
