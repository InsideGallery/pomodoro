package settings

import (
	"testing"

	"github.com/InsideGallery/pomodoro/pkg/config"
)

func TestResetConfigKeepsMute(t *testing.T) {
	cur := config.Default()
	cur.Muted = true
	cur.Theme = "light"
	cur.Transparency = 0.4
	cur.BreakMelody = "piano.mp3"
	cur.FocusMinutes = 40

	got := resetConfig(cur)

	if !got.Muted || got.Theme != "light" || got.Transparency != 0.4 {
		t.Fatalf("kept fields wrong: %+v", got)
	}

	if got.BreakMelody != "tick.mp3" || got.FocusMinutes != 25 {
		t.Fatalf("defaults wrong: %+v", got)
	}
}
