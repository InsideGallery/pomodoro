# Pomodoro Timer - Development Guide

## Project Overview

A single-binary Pomodoro timer desktop application written in Go using Ebiten (ebitengine).
Transparent, undecorated window with Dark/Light themes. System tray integration.
Inspired by the dark fintech dashboard UI style from Dribbble (shot 27171922).

## Tech Stack

- **Language**: Go 1.24+
- **GUI Framework**: Ebiten v2.9+ (`github.com/hajimehoshi/ebiten/v2`)
- **Audio**: Ebiten audio/mp3 (`ebiten/v2/audio`, `ebiten/v2/audio/mp3`)
- **Text Rendering**: `ebiten/v2/text/v2` (vector-quality, HiDPI-aware)
- **Vector Graphics**: `ebiten/v2/vector` for rounded rects, arcs, icons
- **System Tray**: `github.com/getlantern/systray` (requires `libayatana-appindicator3-dev` at build time)
- **Resources**: All assets embedded via `//go:embed` for single-binary distribution
- **Config**: JSON file at `~/.config/pomodoro/config.json`

### Build Dependencies (Linux)

```bash
sudo apt-get install -y libayatana-appindicator3-dev
```

Runtime library `libayatana-appindicator3-1` is pre-installed on most Linux desktops (GNOME, Cinnamon, KDE).

## Architecture

```
cmd/
  pomodoro/main.go              -- Entry point, systray init, ebiten.RunGameWithOptions
  genicon/main.go               -- Tool to generate app icon PNG
internal/
  app/app.go                    -- Game struct, Update/Draw/Layout, screen management,
                                   window dragging, hide-to-tray, mini mode, HiDPI scaling
  timer/
    timer.go                    -- Pure Go timer (states, pendingNext, round counter)
    timer_test.go               -- 17 unit tests
  ui/
    theme.go                    -- UIScale, S()/Sf(), SetTheme(), ApplyTransparency(), Face()
    draw.go                     -- Vector primitives (rounded rect, arc, gradient arc, circle)
                                   Icon drawing (DrawCloseIcon, DrawSettingsIcon,
                                   DrawMinimizeIcon, DrawBackIcon)
    components.go               -- Button (with IconDraw), Slider, Toggle, DrawText helpers
    screen_timer.go             -- Timer display: progress ring, round dots, control buttons
    screen_settings.go          -- Scrollable settings: sliders, toggles, theme, transparency
  audio/audio.go                -- Mutex-safe audio manager (tick loop, alarm, volume)
  config/config.go              -- Config struct + JSON persistence
  tray/
    tray.go                     -- System tray integration (Show/Quit actions)
    icon.go                     -- Programmatic tray icon generation
assets/
  embed.go                      -- //go:embed directives
  fonts/NotoSans-{Regular,Bold}.ttf
  sounds/{tick,alarm}.mp3
packaging/
  pomodoro.desktop              -- FreeDesktop .desktop file
  pomodoro.png                  -- 256x256 app icon
Makefile                        -- build, test, appimage, install, clean
```

## Window Modes

| Mode | Size | Description |
|------|------|-------------|
| Normal | 380x560 | Full timer + controls |
| Mini | 220x60 | Compact timer bar, click expand to restore |
| Hidden | 1x1 off-screen | Hidden to tray, invisible in taskbar |

## Design System

### Themes
Two built-in themes: **Dark** and **Light**. Switched via `SetTheme()` which reassigns
package-level color variables. `ApplyTransparency()` adjusts alpha on window/card/border
colors after theme switch.

### HiDPI Rendering
- `Layout()` returns `outsideWidth * deviceScaleFactor` for native-resolution rendering
- `S(v)` / `Sf(v)` scale logical values by `UIScale`
- `Face(bold, size)` creates fonts at `size * UIScale`
- All fixed dimensions (padding, button sizes, radii) use `S()` in layout code

### Colors (Dark Theme)
- **Window Bg**: `#101018` semi-transparent
- **Card Bg**: `#181822` semi-transparent
- **Text Primary**: `#F0F0F5`
- **Text Secondary**: `#8B8B9E`
- **Accent Focus**: `#6C5CE7` (purple)
- **Accent Break**: `#00CEC9` (teal)

### Colors (Light Theme)
- **Window Bg**: `#F2F3F7` semi-transparent
- **Card Bg**: `#FFFFFF` semi-transparent
- **Text Primary**: `#1A1B25`
- **Text Secondary**: `#6B6D7B`
- **Accents**: Slightly deeper versions of dark theme accents

## Timer States

```
IDLE -> FOCUS -> (auto/manual) -> BREAK -> (auto/manual) -> FOCUS -> ...
                                  LONG_BREAK (every N rounds)
Any state -> PAUSED -> resume to previous state
Any state -> IDLE (reset)
```

States: `Idle`, `Focus`, `Break`, `LongBreak`, `Paused`

`pendingNext` field remembers what `Start()` should begin next (Break after Focus, etc.).

## Key Features

- **Timer**: Focus/Break/LongBreak with configurable durations and round counter
- **Audio**: Tick-tock loop during active timer, alarm on completion, independent volume controls
- **Themes**: Dark / Light toggle with semi-transparent glass effect
- **Transparency**: Configurable 10%-90%, affects both themes
- **System Tray**: Tray icon with Show/Quit menu; X button hides to tray
- **Mini Mode**: Compact 220x60 timer bar
- **Settings**: Scrollable panel with sliders, toggles, Reset Defaults button
- **Keyboard**: Space (start/pause), R (reset), S (settings), Escape (back/exit mini)
- **Window**: Undecorated, transparent, draggable title bar, rounded corners
- **HiDPI**: Native resolution rendering on high-DPI displays
- **Single Binary**: All fonts, sounds, icons embedded

## Build & Run

```bash
make build           # Build binary to build/pomodoro
make test            # Run all tests
make appimage        # Build AppImage (downloads appimagetool on first run)
make install         # Install to /usr/local (requires sudo)
make clean           # Remove build artifacts

go run ./cmd/pomodoro/   # Run directly
```

## Settings (persisted to JSON)

| Setting | Range | Default |
|---------|-------|---------|
| Focus Duration | 1-60 min | 25 min |
| Short Break | 1-30 min | 5 min |
| Long Break | 1-60 min | 15 min |
| Rounds Before Long | 1-10 | 4 |
| Tick Volume | 0-100% | 50% |
| Alarm Volume | 0-100% | 80% |
| Tick Sound | on/off | on |
| Auto-Start Next | on/off | off |
| Theme | dark/light | dark |
| Transparency | 10-90% | 10% |

## Code Conventions

- Timer logic in `internal/timer/` — pure Go, no Ebiten dependency, fully testable
- All Ebiten-specific code in `internal/app/` and `internal/ui/`
- All resources accessed through `assets/embed.go`
- Vector icons drawn with `vector.Path` — no unicode/font icon dependencies
- Callbacks set BEFORE `Init()` so `layout()` can wire them into widgets
- Settings screen uses `shiftToScreen()`/`shiftToContent()` for scroll offset;
  never call `Init()` inside the shift cycle — use `Relayout()` or defer to next frame
- `colorToFloat32()` produces premultiplied alpha for Ebiten vertex colors

## Future Features

- Task list panel
- Platform integrations (sync tasks, upload time-spent)
- Desktop notifications
- Statistics/history view
- Flatpak packaging
- GitHub Actions CI (tests + AppImage artifact)
