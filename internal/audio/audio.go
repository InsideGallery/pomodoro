package audio

import (
	"bytes"
	"io"
	"sync"

	"github.com/InsideGallery/pomodoro/assets"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

const sampleRate = 48000

type Manager struct {
	mu      sync.Mutex
	ctx     *audio.Context
	tick    *audio.Player
	alarm   *audio.Player
	tickBuf []byte
	alarmBuf []byte

	tickVolume  float64
	alarmVolume float64
	tickEnabled bool
}

func NewManager() (*Manager, error) {
	ctx := audio.NewContext(sampleRate)

	tickStream, err := mp3.DecodeF32(bytes.NewReader(assets.TickSound))
	if err != nil {
		return nil, err
	}
	tickBuf, err := io.ReadAll(tickStream)
	if err != nil {
		return nil, err
	}

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
		tickBuf:     tickBuf,
		alarmBuf:    alarmBuf,
		tickVolume:  0.5,
		alarmVolume: 0.8,
		tickEnabled: true,
	}, nil
}

func (m *Manager) PlayTick() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.tickEnabled || m.tickVolume == 0 {
		return
	}

	if m.tick == nil {
		m.tick = m.ctx.NewPlayerF32FromBytes(m.tickBuf)
		m.tick.SetVolume(m.tickVolume)
	}

	if !m.tick.IsPlaying() {
		m.tick.SetPosition(0) //nolint:errcheck
		m.tick.Play()
	}
}

func (m *Manager) StopTick() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tick != nil && m.tick.IsPlaying() {
		m.tick.Pause()
	}
}

func (m *Manager) UpdateTick() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tick == nil || !m.tickEnabled {
		return
	}

	// Loop: restart when finished
	if !m.tick.IsPlaying() && m.tickVolume > 0 {
		m.tick.SetPosition(0) //nolint:errcheck
		m.tick.Play()
	}
}

func (m *Manager) PlayAlarm() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.alarm == nil {
		m.alarm = m.ctx.NewPlayerF32FromBytes(m.alarmBuf)
	}

	m.alarm.SetVolume(m.alarmVolume)
	m.alarm.SetPosition(0) //nolint:errcheck
	m.alarm.Play()
}

func (m *Manager) SetTickVolume(v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tickVolume = clamp(v, 0, 1)
	if m.tick != nil {
		m.tick.SetVolume(m.tickVolume)
	}
}

func (m *Manager) SetAlarmVolume(v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.alarmVolume = clamp(v, 0, 1)
	if m.alarm != nil {
		m.alarm.SetVolume(m.alarmVolume)
	}
}

func (m *Manager) SetTickEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tickEnabled = enabled
	if !enabled && m.tick != nil && m.tick.IsPlaying() {
		m.tick.Pause()
	}
}

func (m *Manager) TickVolume() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tickVolume
}

func (m *Manager) AlarmVolume() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.alarmVolume
}

func (m *Manager) TickEnabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tickEnabled
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
