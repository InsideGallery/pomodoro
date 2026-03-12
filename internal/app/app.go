package app

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/internal/audio"
	"github.com/InsideGallery/pomodoro/internal/config"
	"github.com/InsideGallery/pomodoro/internal/platform"
	"github.com/InsideGallery/pomodoro/internal/timer"
	"github.com/InsideGallery/pomodoro/internal/tray"
	"github.com/InsideGallery/pomodoro/internal/ui"
)

const (
	DefaultWindowWidth  = 380
	DefaultWindowHeight = 560
	MiniWindowWidth     = 220
	MiniWindowHeight    = 60
)

type screen uint8

const (
	screenTimer screen = iota
	screenSettings
	screenMini
)

type Game struct {
	cfg   config.Config
	tmr   *timer.Timer
	audio *audio.Manager

	timerScreen    ui.TimerScreen
	settingsScreen ui.SettingsScreen
	activeScreen   screen
	prevScreen     screen // screen before mini mode

	// Window dragging
	dragging    bool
	dragOffsetX int
	dragOffsetY int

	// Mini mode
	faceMini *textv2.GoTextFace

	// Hidden to tray state
	hidden bool

	// Tray icon state tracking
	lastTrayState timer.State

	width, height         int
	initialized           bool
	pendingSettingsReinit bool
}

func New() *Game {
	cfg := config.Load()
	ui.SetTheme(ui.ThemeID(cfg.Theme))
	ui.ApplyTransparency(cfg.Transparency)

	tmr := timer.New(timer.Config{
		FocusDuration:     cfg.FocusDuration(),
		BreakDuration:     cfg.BreakDuration(),
		LongBreakDuration: cfg.LongBreakDuration(),
		RoundsBeforeLong:  cfg.RoundsBeforeLong,
		AutoStart:         cfg.AutoStart,
	})

	// Restore persisted timer state
	st := config.LoadState()
	tmr.Restore(st.State, st.PrePause, st.PendingNext, st.Round, st.RemainingSec, time.Now())

	g := &Game{
		cfg:           cfg,
		tmr:           tmr,
		activeScreen:  screenTimer,
		width:         DefaultWindowWidth,
		height:        DefaultWindowHeight,
		lastTrayState: timer.StateIdle,
	}

	tmr.OnComplete = func(_ timer.State) {
		if g.audio != nil {
			g.audio.StopTick()
			g.audio.PlayAlarm()
		}

		g.saveState()
	}

	return g
}

func (g *Game) initAudio() {
	am, err := audio.NewManager()
	if err != nil {
		return
	}

	g.audio = am
	g.audio.SetTickVolume(g.cfg.TickVolume)
	g.audio.SetAlarmVolume(g.cfg.AlarmVolume)
	g.audio.SetTickEnabled(g.cfg.TickEnabled)
}

func (g *Game) initScreens() {
	g.timerScreen.Timer = g.tmr
	g.timerScreen.OnStart = g.onStartPause
	g.timerScreen.OnReset = g.onReset
	g.timerScreen.OnSkip = g.onSkip
	g.timerScreen.OnSettings = func() { g.showSettings() }
	g.timerScreen.OnClose = func() { g.hideToTray() }
	g.timerScreen.OnMini = func() { g.enterMini() }
	g.timerScreen.OnSetRound = func(r int) {
		g.tmr.SetRound(r)
		g.saveState()
	}
	g.timerScreen.OnAdjustTime = func(rem time.Duration) {
		g.tmr.SetRemaining(rem, time.Now())
	}
	g.timerScreen.Init(g.width, g.height)

	g.settingsScreen.Cfg = &g.cfg
	g.settingsScreen.OnBack = func() {
		g.applyConfig()
		g.activeScreen = screenTimer
	}
	g.settingsScreen.OnTickVolumeChange = func(v float64) {
		if g.audio != nil {
			g.audio.SetTickVolume(v)
		}
	}
	g.settingsScreen.OnAlarmVolumeChange = func(v float64) {
		if g.audio != nil {
			g.audio.SetAlarmVolume(v)
		}
	}
	g.settingsScreen.OnTickEnabledChange = func(v bool) {
		if g.audio != nil {
			g.audio.SetTickEnabled(v)
		}
	}
	g.settingsScreen.OnThemeChange = func(theme string) {
		ui.SetTheme(ui.ThemeID(theme))
		ui.ApplyTransparency(g.cfg.Transparency)
		g.timerScreen.Init(g.width, g.height)
		// Defer settings reinit to next frame (avoid shift/unshift corruption)
		g.pendingSettingsReinit = true
	}
	g.settingsScreen.OnTransparencyChange = func(t float64) {
		ui.ApplyTransparency(t)
	}
	g.settingsScreen.OnResetDefaults = func() {
		def := config.Default()
		def.Theme = g.cfg.Theme
		def.Transparency = g.cfg.Transparency
		g.cfg = def
		_ = config.Save(g.cfg)
		g.applyConfig()

		if g.audio != nil {
			g.audio.SetTickVolume(g.cfg.TickVolume)
			g.audio.SetAlarmVolume(g.cfg.AlarmVolume)
			g.audio.SetTickEnabled(g.cfg.TickEnabled)
		}
		// Defer the re-init to next frame to avoid corrupting
		// widget Y positions mid-Update (shift/unshift cycle).
		g.pendingSettingsReinit = true
	}
	g.settingsScreen.Init(g.width, g.height)
}

func (g *Game) onStartPause() {
	now := time.Now()

	switch g.tmr.State() {
	case timer.StateIdle:
		g.tmr.Start(now)
		g.startTick()
	case timer.StatePaused:
		g.tmr.Resume(now)
		g.startTick()
	default:
		g.tmr.Pause(now)
		g.stopTick()
	}

	g.saveState()
}

func (g *Game) onReset() {
	g.tmr.Reset()
	g.stopTick()
	g.saveState()
}

func (g *Game) onSkip() {
	g.tmr.Skip(time.Now())

	if g.tmr.State().IsRunning() {
		g.startTick()
	} else {
		g.stopTick()
	}

	g.saveState()
}

func (g *Game) startTick() {
	if g.audio != nil {
		g.audio.PlayTick()
	}
}

func (g *Game) stopTick() {
	if g.audio != nil {
		g.audio.StopTick()
	}
}

func (g *Game) showSettings() {
	g.settingsScreen.Cfg = &g.cfg
	g.settingsScreen.Init(g.width, g.height)
	g.activeScreen = screenSettings
}

func (g *Game) applyConfig() {
	g.tmr.SetConfig(timer.Config{
		FocusDuration:     g.cfg.FocusDuration(),
		BreakDuration:     g.cfg.BreakDuration(),
		LongBreakDuration: g.cfg.LongBreakDuration(),
		RoundsBeforeLong:  g.cfg.RoundsBeforeLong,
		AutoStart:         g.cfg.AutoStart,
	})
}

const windowTitle = "Pomodoro"

func (g *Game) hideToTray() {
	if g.hidden {
		return
	}

	g.hidden = true

	platform.HideWindow(windowTitle)
}

func (g *Game) showFromTray() {
	g.hidden = false

	platform.ShowWindow(windowTitle)
}

func (g *Game) enterMini() {
	g.prevScreen = g.activeScreen
	g.activeScreen = screenMini

	ebiten.SetWindowSize(MiniWindowWidth, MiniWindowHeight)
	platform.SetAlwaysOnTop(windowTitle, true)
}

func (g *Game) exitMini() {
	platform.SetAlwaysOnTop(windowTitle, false)

	g.activeScreen = g.prevScreen
	if g.activeScreen == screenMini {
		g.activeScreen = screenTimer
	}

	ebiten.SetWindowSize(DefaultWindowWidth, DefaultWindowHeight)
}

func (g *Game) updateDrag() {
	mx, my := ebiten.CursorPosition()

	dragH := int(ui.S(48))
	if g.activeScreen == screenMini {
		dragH = g.height // entire mini window is draggable
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && my < dragH {
		g.dragging = true
		g.dragOffsetX = mx
		g.dragOffsetY = my
	}

	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.dragging = false
	}

	if g.dragging {
		wx, wy := ebiten.WindowPosition()
		dx := mx - g.dragOffsetX
		dy := my - g.dragOffsetY
		scale := ui.UIScale
		ebiten.SetWindowPosition(wx+int(float64(dx)/scale), wy+int(float64(dy)/scale))
	}
}

func (g *Game) processTrayActions() {
	select {
	case action := <-tray.ActionCh:
		switch action {
		case tray.ActionShow:
			g.showFromTray()
		case tray.ActionQuit:
			g.saveState()
			tray.Quit()
			os.Exit(0)
		}
	default:
	}
}

func (g *Game) updateTrayIcon() {
	st := g.tmr.State()
	if st == g.lastTrayState {
		return
	}

	g.lastTrayState = st

	var clr color.RGBA

	switch st {
	case timer.StateFocus:
		clr = color.RGBA{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF} // purple
	case timer.StateBreak:
		clr = color.RGBA{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF} // teal
	case timer.StateLongBreak:
		clr = color.RGBA{R: 0x81, G: 0xEC, B: 0xEC, A: 0xFF} // light teal
	case timer.StatePaused:
		clr = color.RGBA{R: 0xFF, G: 0xC1, B: 0x07, A: 0xFF} // yellow
	default:
		clr = color.RGBA{R: 0x8B, G: 0x8B, B: 0x9E, A: 0xFF} // gray
	}

	tray.UpdateIcon(tray.GenerateIcon(32, clr))
}

func (g *Game) saveState() {
	state, prePause, pendingNext, round, remainingSec := g.tmr.Snapshot(time.Now())
	_ = config.SaveState(config.TimerState{
		Round:        round,
		PendingNext:  pendingNext,
		State:        state,
		PrePause:     prePause,
		RemainingSec: remainingSec,
	})
}

func (g *Game) Update() error {
	if !g.initialized {
		g.initAudio()
		g.initScreens()
		g.faceMini = ui.Face(true, 12)
		g.initialized = true
	}

	// Deferred settings relayout (preserves scroll, safe outside shift/unshift cycle)
	if g.pendingSettingsReinit {
		g.pendingSettingsReinit = false
		g.settingsScreen.Cfg = &g.cfg
		g.settingsScreen.Relayout()
	}

	// Handle window close → hide to tray
	if ebiten.IsWindowBeingClosed() {
		g.saveState()
		g.hideToTray()
	}

	g.processTrayActions()
	g.updateDrag()
	g.updateTrayIcon()

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.activeScreen == screenTimer {
		g.onStartPause()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) && g.activeScreen == screenTimer {
		g.onReset()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyS) && g.activeScreen == screenTimer {
		g.showSettings()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch g.activeScreen {
		case screenSettings:
			g.applyConfig()
			g.activeScreen = screenTimer
		case screenMini:
			g.exitMini()
		}
	}

	prevState := g.tmr.State()
	g.tmr.Update(time.Now())
	curState := g.tmr.State()

	if g.audio != nil && curState.IsRunning() {
		// State changed (e.g. auto-start after completion) — restart tick
		if curState != prevState {
			g.startTick()
		}

		g.audio.UpdateTick()
	}

	switch g.activeScreen {
	case screenTimer:
		g.timerScreen.Update()
	case screenSettings:
		g.settingsScreen.Update()
	case screenMini:
		g.updateMini()
	}

	return nil
}

func (g *Game) updateMini() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, _ := ebiten.CursorPosition()

		// Right side: expand button area
		if mx > g.width-int(ui.S(50)) {
			g.exitMini()
			return
		}

		// Left side: play/pause button area
		if mx < int(ui.S(44)) {
			g.onStartPause()
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	w := float32(g.width)
	h := float32(g.height)

	r := ui.S(12)
	if g.activeScreen == screenMini {
		r = ui.S(8)
	}

	ui.DrawRoundedRect(screen, 0, 0, w, h, r, ui.ColorWindowBg)
	ui.DrawRoundedRectStroke(screen, 0, 0, w, h, r, ui.S(1), ui.ColorCardBorder)

	switch g.activeScreen {
	case screenTimer:
		g.timerScreen.Draw(screen)
	case screenSettings:
		g.settingsScreen.Draw(screen)
	case screenMini:
		g.drawMini(screen)
	}
}

func (g *Game) drawMini(screen *ebiten.Image) {
	w := float32(g.width)
	h := float32(g.height)

	// Play/pause button on the left
	ppW := ui.S(36)
	ppX := ui.S(8)
	ppY := ui.S(8)
	ppH := h - ui.S(16)
	ui.DrawRoundedRect(screen, ppX, ppY, ppW, ppH, ui.S(6), ui.ColorBgTertiary)

	state := g.tmr.State()
	if state.IsRunning() {
		ui.DrawPauseIcon(screen, ppX+ppW/2, ppY+ppH/2, ui.S(18), ui.ColorTextPrimary)
	} else {
		ui.DrawPlayIcon(screen, ppX+ppW/2, ppY+ppH/2, ui.S(18), ui.ColorTextPrimary)
	}

	// Timer text centered
	now := time.Now()

	rem := g.tmr.Remaining(now)
	if rem < 0 {
		rem = 0
	}

	totalSecs := int(rem.Seconds())
	mins := totalSecs / 60
	secs := totalSecs % 60
	timerText := fmt.Sprintf("%02d:%02d", mins, secs)

	timerFace := ui.Face(true, 18)
	tw, _ := textv2.Measure(timerText, timerFace, 0)
	ui.DrawText(screen, timerText, timerFace, float64(w)/2-tw/2, float64(h/2)-Sf(10), ui.ColorTextPrimary)

	// Expand button on the right
	btnW := ui.S(36)
	btnX := w - ui.S(8) - btnW
	btnY := ui.S(8)
	btnH := h - ui.S(16)
	ui.DrawRoundedRect(screen, btnX, btnY, btnW, btnH, ui.S(6), ui.ColorBgTertiary)
	ui.DrawExpandIcon(screen, btnX+btnW/2, btnY+btnH/2, ui.S(16), ui.ColorTextPrimary)
}

func Sf(v float64) float64 { return ui.Sf(v) }

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	scale := 1.0
	if m := ebiten.Monitor(); m != nil {
		scale = m.DeviceScaleFactor()
	}

	ui.UIScale = scale

	w := int(math.Ceil(float64(outsideWidth) * scale))
	h := int(math.Ceil(float64(outsideHeight) * scale))

	if w != g.width || h != g.height {
		g.width = w

		g.height = h
		if g.initialized {
			if g.activeScreen != screenMini {
				g.timerScreen.Resize(w, h)
				g.settingsScreen.Resize(w, h)
			}

			g.faceMini = ui.Face(true, 12)
		}
	}

	return w, h
}
