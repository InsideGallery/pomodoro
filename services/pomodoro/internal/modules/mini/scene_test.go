package mini

import (
	"testing"

	"github.com/InsideGallery/pomodoro/pkg/ui"
)

func TestHit(t *testing.T) {
	ui.UIScale = 1

	cases := []struct {
		x, y int
		want miniButton
	}{
		{20, 30, btnPlay},
		{150, 30, btnMute},
		{190, 30, btnExpand},
		{100, 30, btnNone},
		{150, 2, btnNone},
	}

	for _, c := range cases {
		if got := hit(220, 60, c.x, c.y); got != c.want {
			t.Errorf("hit(%d,%d) = %d, want %d", c.x, c.y, got, c.want)
		}
	}
}

func TestLayout(t *testing.T) {
	ui.UIScale = 1

	play, mute, expand := layout(220, 60)
	if mute.X != 132 || expand.X != 176 {
		t.Errorf("mute.X=%v expand.X=%v, want 132 and 176", mute.X, expand.X)
	}

	if c := (play.X + play.W + mute.X) / 2; c != 88 {
		t.Errorf("time centre = %v, want 88", c)
	}
}
