package systems

import (
	"context"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/internal/audio"
	"github.com/InsideGallery/pomodoro/internal/timer"
	"github.com/InsideGallery/pomodoro/pkg/event"
)

// TickSystem advances the timer each frame and publishes state-change events.
type TickSystem struct {
	Tmr       *timer.Timer
	Audio     *audio.Manager
	Bus       *event.Bus
	SaveState func()
}

func (s *TickSystem) Update(_ context.Context) error {
	now := time.Now()
	prevState := s.Tmr.State()
	s.Tmr.Update(now)
	curState := s.Tmr.State()

	// Auto-start: timer completed and immediately started next phase
	if curState.IsRunning() && curState != prevState && !prevState.IsRunning() {
		s.publishStarted(curState, now)
	}

	if s.Audio != nil && curState.IsRunning() {
		if curState != prevState {
			s.Audio.PlayTick()
		}

		s.Audio.UpdateTick()
	}

	if curState.IsRunning() {
		s.Bus.Publish(event.Event{Type: event.Tick, Time: now})
	}

	return nil
}

func (s *TickSystem) Draw(_ context.Context, _ *ebiten.Image) {}

func (s *TickSystem) OnStartPause() {
	now := time.Now()

	switch s.Tmr.State() {
	case timer.StateIdle:
		s.Tmr.Start(now)
		s.publishStarted(s.Tmr.State(), now)

		if s.Audio != nil {
			s.Audio.PlayTick()
		}
	case timer.StatePaused:
		s.Tmr.Resume(now)
		s.Bus.Publish(event.Event{Type: event.Resumed, Time: now, Data: s.Tmr.State().String()})

		if s.Audio != nil {
			s.Audio.PlayTick()
		}
	default:
		s.Tmr.Pause(now)
		s.Bus.Publish(event.Event{Type: event.Paused, Time: now, Data: "Paused"})

		if s.Audio != nil {
			s.Audio.StopTick()
		}
	}

	s.SaveState()
}

func (s *TickSystem) OnReset() {
	s.Tmr.Reset()
	s.Bus.Publish(event.Event{Type: event.Reset, Time: time.Now(), Data: "Idle"})

	if s.Audio != nil {
		s.Audio.StopTick()
	}

	s.SaveState()
}

func (s *TickSystem) OnSkip() {
	now := time.Now()
	s.Tmr.Skip(now)

	if s.Tmr.State().IsRunning() {
		s.publishStarted(s.Tmr.State(), now)

		if s.Audio != nil {
			s.Audio.PlayTick()
		}
	} else if s.Audio != nil {
		s.Audio.StopTick()
	}

	s.SaveState()
}

func (s *TickSystem) publishStarted(st timer.State, now time.Time) {
	var t event.Type

	switch st {
	case timer.StateFocus:
		t = event.FocusStarted
	case timer.StateBreak:
		t = event.BreakStarted
	case timer.StateLongBreak:
		t = event.LongBreakStarted
	default:
		return
	}

	s.Bus.Publish(event.Event{Type: t, Time: now, Data: st.String()})
}

func (s *TickSystem) PublishCompleted(st timer.State) {
	var t event.Type

	switch st {
	case timer.StateFocus:
		t = event.FocusCompleted
	case timer.StateBreak:
		t = event.BreakCompleted
	case timer.StateLongBreak:
		t = event.LongBreakCompleted
	default:
		return
	}

	s.Bus.Publish(event.Event{Type: t, Time: time.Now(), Data: "Idle"})
}
