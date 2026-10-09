package audio

import (
	"testing"

	"github.com/InsideGallery/pomodoro/assets"
	"github.com/InsideGallery/pomodoro/services/pomodoro/internal/timer"
)

func TestLoopFor(t *testing.T) {
	states := []timer.State{timer.StateIdle, timer.StateFocus, timer.StateBreak, timer.StateLongBreak, timer.StatePaused}
	melodies := []string{"tick.mp3", "pause.mp3", "missing.mp3", ""}

	for _, st := range states {
		for _, muted := range []bool{false, true} {
			for _, enabled := range []bool{false, true} {
				for _, bm := range melodies {
					want := ""

					if !muted && enabled {
						switch st {
						case timer.StateFocus:
							want = "tick.mp3"
						case timer.StateBreak, timer.StateLongBreak:
							want = bm
							if bm == "missing.mp3" || bm == "" {
								want = "tick.mp3"
							}
						}
					}

					if got := loopFor(st, muted, enabled, bm); got != want {
						t.Errorf("loopFor(%v,%v,%v,%q)=%q want %q", st, muted, enabled, bm, got, want)
					}
				}
			}
		}
	}
}

func TestNextMelody(t *testing.T) {
	tests := []struct {
		file string
		step int
		want string
	}{
		{"tick.mp3", -1, "piano.mp3"},
		{"piano.mp3", 1, "tick.mp3"},
		{"unknown.mp3", 1, "pause.mp3"},
		{"tick.mp3", 1, "pause.mp3"},
	}

	for _, tc := range tests {
		if got := NextMelody(tc.file, tc.step).File; got != tc.want {
			t.Errorf("NextMelody(%q,%d)=%q want %q", tc.file, tc.step, got, tc.want)
		}
	}
}

func TestMelodiesAreEmbedded(t *testing.T) {
	for _, m := range Melodies {
		b, err := assets.Sounds.ReadFile("sounds/" + m.File)
		if err != nil || len(b) == 0 {
			t.Errorf("%s: err=%v len=%d", m.File, err, len(b))
		}
	}
}
