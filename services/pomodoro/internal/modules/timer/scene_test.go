package timer

import (
	"testing"

	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/event"
)

func TestToggleMute(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	bus := event.NewBus()

	var got []bool

	bus.Subscribe(event.ConfigChanged, func(e event.Event) {
		if c, ok := e.Data.(config.Config); ok {
			got = append(got, c.Muted)
		}
	})

	s := NewScene(bus, func(string) {}, func() {}, func() {})

	for i, want := range []bool{true, false} {
		s.ToggleMute()

		if s.Muted() != want {
			t.Fatalf("step %d: Muted() = %v, want %v", i, s.Muted(), want)
		}

		if config.Load().Muted != want {
			t.Fatalf("step %d: saved Muted = %v, want %v", i, config.Load().Muted, want)
		}

		if len(got) != i+1 || got[i] != want {
			t.Fatalf("step %d: events = %v", i, got)
		}
	}
}
