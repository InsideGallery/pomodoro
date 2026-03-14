package minigame

import (
	"math"
	"math/rand/v2"
	"time"
)

// Target is a clickable circle on screen.
type Target struct {
	X, Y   float64
	Radius float64
	Alive  bool
}

// sizeClass defines how many targets of a given radius range to spawn.
type sizeClass struct {
	MinR, MaxR float64
	Count      int
}

var defaultSizes = []sizeClass{
	{4, 10, 4},   // tiny
	{11, 30, 3},  // small
	{31, 60, 2},  // medium
	{61, 120, 1}, // large
}

// Game holds the pure state for a Button Hunt session.
type Game struct {
	Targets   []Target
	Score     int
	BestScore int

	screenW, screenH int
	startedAt        time.Time
	breakDur         time.Duration
}

// Start initializes a new game session.
func (g *Game) Start(screenW, screenH, bestScore int, breakDur time.Duration, now time.Time) {
	g.screenW = screenW
	g.screenH = screenH
	g.BestScore = bestScore
	g.breakDur = breakDur
	g.startedAt = now
	g.Score = 0
	g.Targets = nil
	g.SpawnBatch()
}

// SpawnBatch appends 10 new targets with the standard size distribution.
func (g *Game) SpawnBatch() {
	for _, sc := range defaultSizes {
		for range sc.Count {
			r := sc.MinR + rand.Float64()*(sc.MaxR-sc.MinR) //nolint:gosec // game visuals, not security

			// Keep targets fully within screen bounds
			x := r + rand.Float64()*math.Max(0, float64(g.screenW)-2*r) //nolint:gosec // game visuals
			y := r + rand.Float64()*math.Max(0, float64(g.screenH)-2*r) //nolint:gosec // game visuals

			g.Targets = append(g.Targets, Target{X: x, Y: y, Radius: r, Alive: true})
		}
	}
}

// Click checks if (x,y) hits an alive target. Smallest target wins on overlap.
// Returns true if a target was hit.
func (g *Game) Click(x, y float64) bool {
	bestIdx := -1
	bestRadius := math.MaxFloat64

	for i, t := range g.Targets {
		if !t.Alive {
			continue
		}

		dx := x - t.X
		dy := y - t.Y

		if dx*dx+dy*dy <= t.Radius*t.Radius && t.Radius < bestRadius {
			bestIdx = i
			bestRadius = t.Radius
		}
	}

	if bestIdx < 0 {
		return false
	}

	g.Targets[bestIdx].Alive = false
	g.Score++

	if g.AliveCount() <= 1 {
		g.SpawnBatch()
	}

	return true
}

// AliveCount returns the number of targets still alive.
func (g *Game) AliveCount() int {
	count := 0

	for _, t := range g.Targets {
		if t.Alive {
			count++
		}
	}

	return count
}

// IsOver returns true when the break duration has elapsed.
func (g *Game) IsOver(now time.Time) bool {
	return now.Sub(g.startedAt) >= g.breakDur
}

// BeatRecord returns true if the current score exceeds the previous best.
func (g *Game) BeatRecord() bool {
	return g.Score > g.BestScore
}

// Remaining returns time left in the game.
func (g *Game) Remaining(now time.Time) time.Duration {
	rem := g.breakDur - now.Sub(g.startedAt)
	if rem < 0 {
		return 0
	}

	return rem
}
