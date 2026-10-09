package audio

import (
	"bytes"
	"io"
	"log/slog"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"

	"github.com/InsideGallery/pomodoro/assets"
	"github.com/InsideGallery/pomodoro/services/pomodoro/internal/timer"
)

const sampleRate = 48000

// Settings is the audio part of the user configuration.
type Settings struct {
	LoopVolume, AlarmVolume float64
	LoopEnabled, Muted      bool
	BreakMelody             string
}

// Manager plays one streamed loop and the decoded alarm.
type Manager struct {
	mu       sync.Mutex
	ctx      *audio.Context
	alarm    *audio.Player
	alarmBuf []byte
	loop     *audio.Player
	loopFile string

	loopVolume  float64
	alarmVolume float64
	loopEnabled bool
	muted       bool
	breakMelody string
}

func NewManager() (*Manager, error) {
	ctx := audio.NewContext(sampleRate)

	alarmStream, err := mp3.DecodeF32(bytes.NewReader(assets.AlarmSound))
	if err != nil {
		return nil, err
	}

	alarmBuf, err := io.ReadAll(alarmStream)
	if err != nil {
		return nil, err
	}

	return &Manager{
		ctx:         ctx,
		alarmBuf:    alarmBuf,
		loopVolume:  0.5,
		alarmVolume: 0.8,
		loopEnabled: true,
		breakMelody: "tick.mp3",
	}, nil
}

// Apply stores the settings and applies them to the live players.
func (m *Manager) Apply(s Settings) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.loopVolume = clamp(s.LoopVolume, 0, 1)
	m.alarmVolume = clamp(s.AlarmVolume, 0, 1)
	m.loopEnabled = s.LoopEnabled
	m.muted = s.Muted
	m.breakMelody = s.BreakMelody

	if m.loop != nil {
		m.loop.SetVolume(m.loopVolume)
	}

	if m.alarm != nil {
		m.alarm.SetVolume(m.alarmVolume)

		if m.muted && m.alarm.IsPlaying() {
			m.alarm.Pause()
		}
	}
}

// SyncLoop makes the playing loop match the timer state.
func (m *Manager) SyncLoop(st timer.State) {
	m.mu.Lock()
	defer m.mu.Unlock()

	want := loopFor(st, m.muted, m.loopEnabled, m.breakMelody)
	if want == "" {
		m.closeLoop()

		return
	}

	if want != m.loopFile || m.loop == nil {
		m.closeLoop()
		m.startLoop(want)

		return
	}

	if !m.loop.IsPlaying() {
		m.loop.Play()
	}
}

func (m *Manager) closeLoop() {
	if m.loop != nil {
		m.loop.Pause()
		m.loop.Close() //nolint:errcheck
	}

	m.loop = nil
	m.loopFile = ""
}

func (m *Manager) startLoop(want string) {
	data, err := assets.Sounds.ReadFile("sounds/" + want)
	if err != nil {
		slog.Warn("audio loop", "file", want, "error", err)

		return
	}

	stream, err := mp3.DecodeF32(bytes.NewReader(data))
	if err != nil {
		slog.Warn("audio loop", "file", want, "error", err)

		return
	}

	looped := audio.NewInfiniteLoopF32(stream, stream.Length())

	p, err := m.ctx.NewPlayerF32(looped)
	if err != nil {
		slog.Warn("audio loop", "file", want, "error", err)

		return
	}

	p.SetVolume(m.loopVolume)
	p.Play()

	m.loop = p
	m.loopFile = want
}

func (m *Manager) PlayAlarm() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.muted {
		return
	}

	if m.alarm == nil {
		m.alarm = m.ctx.NewPlayerF32FromBytes(m.alarmBuf)
	}

	m.alarm.SetVolume(m.alarmVolume)
	m.alarm.SetPosition(0) //nolint:errcheck
	m.alarm.Play()
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}
