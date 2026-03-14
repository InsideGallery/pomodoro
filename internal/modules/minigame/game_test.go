package minigame

import (
	"testing"
	"time"
)

func newTestGame() *Game {
	g := &Game{}
	g.Start(1920, 1080, 50, 5*time.Minute, time.Now())

	return g
}

func TestSpawnBatchGenerates10Targets(t *testing.T) {
	g := newTestGame()

	alive := g.AliveCount()
	if alive != 10 {
		t.Fatalf("expected 10 alive targets, got %d", alive)
	}
}

func TestSpawnBatchSizeDistribution(t *testing.T) {
	g := newTestGame()

	tiny, small, medium, large := 0, 0, 0, 0

	for _, tgt := range g.Targets {
		switch {
		case tgt.Radius >= 4 && tgt.Radius <= 10:
			tiny++
		case tgt.Radius >= 11 && tgt.Radius <= 30:
			small++
		case tgt.Radius >= 31 && tgt.Radius <= 60:
			medium++
		case tgt.Radius >= 61 && tgt.Radius <= 120:
			large++
		default:
			t.Fatalf("target radius %f out of expected ranges", tgt.Radius)
		}
	}

	if tiny != 4 {
		t.Fatalf("expected 4 tiny, got %d", tiny)
	}

	if small != 3 {
		t.Fatalf("expected 3 small, got %d", small)
	}

	if medium != 2 {
		t.Fatalf("expected 2 medium, got %d", medium)
	}

	if large != 1 {
		t.Fatalf("expected 1 large, got %d", large)
	}
}

func TestClickHitsTarget(t *testing.T) {
	g := &Game{}
	g.screenW = 1920
	g.screenH = 1080
	g.breakDur = 5 * time.Minute
	g.startedAt = time.Now()
	g.Targets = []Target{
		{X: 100, Y: 100, Radius: 20, Alive: true},
		{X: 500, Y: 500, Radius: 30, Alive: true},
	}

	hit := g.Click(105, 100)
	if !hit {
		t.Fatal("expected hit")
	}

	if g.Score != 1 {
		t.Fatalf("expected score 1, got %d", g.Score)
	}

	if g.Targets[0].Alive {
		t.Fatal("expected target 0 to be dead")
	}

	if !g.Targets[1].Alive {
		t.Fatal("expected target 1 to still be alive")
	}
}

func TestClickMissesTarget(t *testing.T) {
	g := &Game{}
	g.Targets = []Target{
		{X: 100, Y: 100, Radius: 10, Alive: true},
	}

	hit := g.Click(200, 200)
	if hit {
		t.Fatal("expected miss")
	}

	if g.Score != 0 {
		t.Fatalf("expected score 0, got %d", g.Score)
	}
}

func TestClickSmallestTargetFirst(t *testing.T) {
	g := &Game{}
	g.screenW = 1920
	g.screenH = 1080
	g.breakDur = 5 * time.Minute
	g.startedAt = time.Now()

	// Two overlapping targets at the same center, different sizes
	g.Targets = []Target{
		{X: 100, Y: 100, Radius: 50, Alive: true},
		{X: 100, Y: 100, Radius: 10, Alive: true},
		{X: 800, Y: 800, Radius: 20, Alive: true}, // extra to prevent spawn
	}

	hit := g.Click(100, 100)
	if !hit {
		t.Fatal("expected hit")
	}

	// Smaller target (index 1) should be hit
	if g.Targets[1].Alive {
		t.Fatal("expected smaller target to be hit first")
	}

	if !g.Targets[0].Alive {
		t.Fatal("expected larger target to still be alive")
	}
}

func TestSpawnOnLastTarget(t *testing.T) {
	g := &Game{}
	g.screenW = 1920
	g.screenH = 1080
	g.breakDur = 5 * time.Minute
	g.startedAt = time.Now()

	// Start with 2 targets
	g.Targets = []Target{
		{X: 100, Y: 100, Radius: 20, Alive: true},
		{X: 500, Y: 500, Radius: 20, Alive: true},
	}

	// Kill first one — now 1 alive, should trigger SpawnBatch
	g.Click(100, 100)

	// Should have spawned 10 more + the 1 remaining = 11 alive
	alive := g.AliveCount()
	if alive != 11 {
		t.Fatalf("expected 11 alive after spawn (10 new + 1 existing), got %d", alive)
	}
}

func TestBeatRecord(t *testing.T) {
	g := &Game{BestScore: 10}

	g.Score = 5
	if g.BeatRecord() {
		t.Fatal("score 5 should not beat record 10")
	}

	g.Score = 10
	if g.BeatRecord() {
		t.Fatal("score 10 should not beat record 10 (must exceed)")
	}

	g.Score = 11
	if !g.BeatRecord() {
		t.Fatal("score 11 should beat record 10")
	}
}

func TestIsOver(t *testing.T) {
	now := time.Now()
	g := &Game{breakDur: 5 * time.Minute, startedAt: now}

	if g.IsOver(now.Add(4 * time.Minute)) {
		t.Fatal("should not be over at 4 minutes")
	}

	if !g.IsOver(now.Add(5 * time.Minute)) {
		t.Fatal("should be over at 5 minutes")
	}

	if !g.IsOver(now.Add(6 * time.Minute)) {
		t.Fatal("should be over at 6 minutes")
	}
}

func TestTargetsWithinBounds(t *testing.T) {
	g := newTestGame()

	for i, tgt := range g.Targets {
		if tgt.X-tgt.Radius < 0 || tgt.X+tgt.Radius > float64(g.screenW) {
			t.Fatalf("target %d X out of bounds: x=%f r=%f screenW=%d", i, tgt.X, tgt.Radius, g.screenW)
		}

		if tgt.Y-tgt.Radius < 0 || tgt.Y+tgt.Radius > float64(g.screenH) {
			t.Fatalf("target %d Y out of bounds: y=%f r=%f screenH=%d", i, tgt.Y, tgt.Radius, g.screenH)
		}
	}
}

func TestRemaining(t *testing.T) {
	now := time.Now()
	g := &Game{breakDur: 5 * time.Minute, startedAt: now}

	rem := g.Remaining(now.Add(3 * time.Minute))
	if rem != 2*time.Minute {
		t.Fatalf("expected 2m remaining, got %s", rem)
	}

	rem = g.Remaining(now.Add(6 * time.Minute))
	if rem != 0 {
		t.Fatalf("expected 0 remaining, got %s", rem)
	}
}

func TestStartResetsState(t *testing.T) {
	g := &Game{Score: 42}
	g.Targets = []Target{{X: 1, Y: 1, Radius: 5, Alive: true}}

	g.Start(1920, 1080, 100, 5*time.Minute, time.Now())

	if g.Score != 0 {
		t.Fatalf("expected score 0 after start, got %d", g.Score)
	}

	if g.BestScore != 100 {
		t.Fatalf("expected best score 100, got %d", g.BestScore)
	}

	if g.AliveCount() != 10 {
		t.Fatalf("expected 10 targets after start, got %d", g.AliveCount())
	}
}
