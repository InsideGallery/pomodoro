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
    timer_test.go               -- 20 unit tests
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

## Architecture Evolution: Modular Event-Driven Design

The app is evolving beyond a simple timer. Planned features (mini-games during breaks,
lock screen on long breaks, task lists, statistics) require an architecture that keeps
new functionality isolated from the stable core.

### Principles

1. **Domain logic stays pure** -- no Ebiten imports in `timer/`, `config/`, `event/`, `module/`
2. **Modules are self-contained** -- each feature lives in its own package under `internal/modules/`
3. **Communication via events** -- modules react to typed events, never call each other directly
4. **Screens are pluggable** -- any module can provide a `Screen` implementation
5. **Test coverage >= 70%** -- every new package ships with tests; coverage gate enforced in CI

### Target Architecture

```
internal/
  event/                          -- Event bus (generic, reusable)
    bus.go                        -- Bus struct, Subscribe(), Publish()
    bus_test.go
    types.go                      -- EventType enum, Event struct
  module/                         -- Module system (generic, reusable)
    module.go                     -- Module interface, Registry
    module_test.go
    screen.go                     -- Screen interface
  timer/                          -- Domain: timer state machine (existing, unchanged)
  config/                         -- Domain: persistence (existing)
  app/                            -- Ebiten game loop, thin router
    app.go                        -- Holds registry, broadcasts events, delegates to active screen
  ui/                             -- Rendering primitives, shared components (existing)
  audio/                          -- Audio manager (existing)
  modules/                        -- Feature modules (each is independent)
    minigame/                     -- Optional mini-game during short breaks
      module.go                   -- Implements Module interface
      screen.go                   -- Game screen (Ebiten)
      game.go                     -- Game logic (pure, testable)
      game_test.go
    lockscreen/                   -- Required lock during long breaks
      module.go
      screen.go
      lock.go                     -- Lock logic (pure, testable)
      lock_test.go
    notifications/                -- Desktop notifications on state changes
      module.go
    stats/                        -- Session history and statistics
      module.go
      store.go                    -- Persistence (pure, testable)
      store_test.go
      screen.go
```

### Interfaces

```go
// internal/event/types.go
type EventType int
const (
    EventFocusStarted EventType = iota
    EventFocusCompleted
    EventBreakStarted
    EventBreakCompleted
    EventLongBreakStarted
    EventLongBreakCompleted
    EventPaused
    EventResumed
    EventReset
    EventTick               // fired every Update() while running
)

type Event struct {
    Type EventType
    Time time.Time
}

// internal/event/bus.go
type Handler func(Event)
type Bus struct { ... }
func (b *Bus) Subscribe(t EventType, h Handler)
func (b *Bus) Publish(e Event)

// internal/module/screen.go
type Screen interface {
    Init(w, h int)
    Update()
    Draw(screen *ebiten.Image)
    Resize(w, h int)
}

// internal/module/module.go
type Module interface {
    ID() string
    Init(bus *event.Bus)
    Enabled() bool
    Screen() Screen           // nil if module has no screen
    Priority() int            // higher = checked first for screen activation
}

type Registry struct { ... }
func (r *Registry) Register(m Module)
func (r *Registry) ActiveScreen() Screen  // highest-priority module with non-nil active screen
func (r *Registry) Modules() []Module
```

### Implementation Phases

Each phase is a standalone PR. No phase changes existing user-visible behavior.
Tests must pass and coverage must meet the phase target before merging.

#### Phase 1: Foundation (coverage target: 60%)

Goal: Add testable infrastructure without changing behavior.

1. **Add `internal/config/` tests** -- Load/Save/LoadState/SaveState round-trips,
   defaults, migration of nanosecond values. This alone lifts coverage significantly
   since `config` is currently at 0%.
2. **Create `internal/event/`** -- Event types enum, Event struct, Bus with
   Subscribe/Publish. Pure Go, no Ebiten. 90%+ coverage.
3. **Create `internal/module/screen.go`** -- Screen interface only (extract from
   what TimerScreen/SettingsScreen already implement implicitly).
4. **Update `.testcoverage.yml`** -- Raise total threshold to 60%.

Deliverables: event bus tested and ready, config fully tested, Screen interface defined.

#### Phase 2: Module System (coverage target: 65%)

Goal: Add module registry; wire events into app.go alongside existing callbacks.

1. **Create `internal/module/module.go`** -- Module interface + Registry.
   Pure Go, fully tested.
2. **Wire event bus into `app.go`** -- Publish events from timer state transitions
   (alongside existing OnComplete callback, not replacing it yet).
3. **Refactor existing screens** -- TimerScreen and SettingsScreen satisfy Screen
   interface (may need minor signature adjustments).
4. **Update `.testcoverage.yml`** -- Raise total threshold to 65%.

Deliverables: modules can register and receive events; existing behavior unchanged.

#### Phase 3: First Modules (coverage target: 70%)

Goal: Ship mini-game and lock screen as isolated modules.

1. **`internal/modules/minigame/`** -- Activates on `EventBreakStarted`.
   Shows a simple game (e.g., click-the-circles, memory match). Skippable.
   Game logic in pure Go with tests; screen uses Ebiten.
2. **`internal/modules/lockscreen/`** -- Activates on `EventLongBreakStarted`.
   Fullscreen overlay, cannot be skipped until break completes.
   Lock logic in pure Go with tests.
3. **Module config** -- Each module can declare settings; settings screen
   discovers them from the registry (future-proof, not required for Phase 3).
4. **Update `.testcoverage.yml`** -- Raise total threshold to 70%.

Deliverables: two working modules, each independently testable, 70% coverage met.

#### Phase 4: Extended Modules (coverage target: 70%+)

Goal: Build on the module system for additional features.

- **`internal/modules/notifications/`** -- Desktop notifications via D-Bus/libnotify.
- **`internal/modules/stats/`** -- Persist completed sessions, show history screen.
- **`internal/modules/tasks/`** -- Task list panel linked to focus sessions.

### Rules for Module Development

- A module MUST NOT import another module's package.
- A module communicates only through the event bus.
- Game/business logic MUST be in pure Go files (no Ebiten imports), with tests.
- Screen/rendering code MAY use Ebiten but SHOULD be thin.
- Each module MUST have a `*_test.go` covering its logic at >= 70%.
- Modules are registered in `app.New()` -- commenting out a Register call
  cleanly disables the feature with zero side effects.

## Future Features

- Task list panel
- Platform integrations (sync tasks, upload time-spent)
- Desktop notifications
- Statistics/history view
- Flatpak packaging
- GitHub Actions CI (tests + AppImage artifact)
