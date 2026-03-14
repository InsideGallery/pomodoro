# Pomodoro Timer - Development Guide

## Project Overview

A single-binary Pomodoro timer desktop application written in Go using Ebiten (ebitengine).
Transparent, undecorated window with Dark/Light themes. System tray integration.

## Tech Stack

- **Language**: Go 1.24+
- **GUI Framework**: Ebiten v2.9+ (`github.com/hajimehoshi/ebiten/v2`)
- **ECS**: `github.com/InsideGallery/core` (System, Entity, Component, Registry)
- **Spatial**: `github.com/InsideGallery/game-core` (RTree, shapes, GJK/EPA)
- **Audio**: Ebiten audio/mp3
- **Text Rendering**: `ebiten/v2/text/v2` (vector-quality, HiDPI-aware)
- **Vector Graphics**: `ebiten/v2/vector` for rounded rects, arcs, icons
- **System Tray**: `github.com/getlantern/systray`
- **Resources**: All assets embedded via `//go:embed`
- **Config**: JSON file at `~/.config/pomodoro/config.json`

### Build Dependencies (Linux)

```bash
sudo apt-get install -y libayatana-appindicator3-dev
```

## Architecture: ECS + Scene-Based Design

Following the same ECS and scene patterns as `InsideGallery/detective`.
Each module is a self-contained Scene with its own ECS (Systems, Entities, Components).
The app is a thin shell that delegates to a Scene Manager.

### Core Libraries

From `InsideGallery/core`:
```go
ecs.System    — interface { Update(ctx context.Context) error }
ecs.Component — interface { Update(ctx context.Context) error }
ecs.Entity    — interface { GetID() uint64 }
ecs.BaseEntity — id + atomic version

registry.Registry[G, I, V] — thread-safe multi-group entity storage
  .Add(group, id, entity)
  .Get(group, id)
  .Remove(group, id)
  .Iterator(groups...) chan V
  .GetValues(group) []V
```

From `InsideGallery/game-core`:
```go
shapes.Spatial  — interface { Point1(), Center(), Bounds() Box, Move() }
shapes.Collide  — Spatial + Support(d Point) Point
rtree.RTree     — spatial index: Insert, Delete, Collision, NearestNeighbor
gjkepa2d.GJKEPA — collision detection via support functions
```

### Extended System Interface (same as detective)

```go
// internal/core/system.go
type System interface {
    ecs.System                                              // Update(ctx) error
    Draw(ctx context.Context, screen *ebiten.Image)         // render to world
}

type SystemWindow interface {
    System
    ScreenDraw(ctx context.Context, screen *ebiten.Image)   // render to screen (UI overlay)
}
```

### Scene Interface

```go
// internal/scene/scene.go
type Scene interface {
    Name() string
    Init(ctx context.Context)
    Load() error                                            // entering this scene
    Unload() error                                          // leaving this scene
    Update() error
    Draw(screen *ebiten.Image)
    Layout(outsideWidth, outsideHeight int) (int, int)
}
```

### BaseScene (shared ECS infrastructure)

Every scene embeds BaseScene and gets Systems + Registry + RTree for free.

```go
// internal/scene/base.go
type BaseScene struct {
    context.Context
    *core.Systems
    *registry.Registry[string, uint64, any]
    *rtree.RTree
    *event.Bus
}
```

BaseScene.Update() iterates all systems in order, calling Update(ctx).
BaseScene.Draw() iterates all systems, calling Draw(ctx, screen),
then calls ScreenDraw() on SystemWindow systems for UI overlays.

### Scene Manager

```go
// internal/scene/manager.go
type Manager struct {
    scenes  map[string]Scene
    current Scene
}

func (m *Manager) Add(ctx, scenes...)     // registers + calls Init()
func (m *Manager) SwitchSceneTo(name)     // calls Load()/Unload()
func (m *Manager) Scene() Scene           // current active scene
```

### App (thin shell)

```go
// internal/app/app.go
type Game struct {
    manager *scene.Manager
    bus     *event.Bus
    // window management: dragging, tray, HiDPI
}

func (g *Game) Update() error { return g.manager.Scene().Update() }
func (g *Game) Draw(s)        { g.manager.Scene().Draw(s) }
func (g *Game) Layout(w, h)   { return g.manager.Scene().Layout(w, h) }
```

App handles only: window dragging, system tray, HiDPI scaling, scene switching
based on events. All game/UI logic lives in scenes.

### Directory Structure

```
cmd/
  pomodoro/main.go                  -- Entry point, systray, ebiten.RunGameWithOptions
internal/
  core/                             -- Extended ECS for Ebiten (follows detective pattern)
    system.go                       -- System, SystemWindow interfaces
    systems.go                      -- Systems container (ordered, named, thread-safe)
    systems_test.go
  scene/                            -- Scene infrastructure
    scene.go                        -- Scene interface
    base.go                         -- BaseScene (Systems + Registry + RTree + Bus)
    base_test.go
    manager.go                      -- SceneManager (Add, SwitchSceneTo, Scene)
    manager_test.go
  app/                              -- Ebiten Game shell
    app.go                          -- Thin: delegates to SceneManager
  event/                            -- Event bus for cross-scene communication
    bus.go                          -- Bus, Subscribe, Publish (existing)
    types.go                        -- Event types (existing)
  timer/                            -- Domain: pure timer state machine (existing)
  config/                           -- Domain: JSON persistence (existing)
  audio/                            -- Audio manager (existing)
  ui/                               -- Low-level drawing primitives (existing)
  tray/                             -- System tray integration (existing)
  platform/                         -- Platform-specific window ops (existing)
  ecs/                              -- Shared ECS building blocks
    components/                     -- Reusable component types
      position.go                   -- Position {X, Y float64}
      clickable.go                  -- Clickable zone (shapes.Spatial for RTree)
      renderable.go                 -- Visual appearance (color, radius, shape type)
      text.go                       -- Dynamic text content (label, face, color)
    entities/                       -- Reusable entity types (if any)
  modules/                          -- Each module = one scene with own ECS
    timer/                          -- Timer scene (main screen)
      scene.go                      -- Embeds BaseScene, registers systems
      systems/                      -- Timer-specific systems
        render.go                   -- Draws progress ring, buttons, labels
        input.go                    -- Button click handling
        tick.go                     -- Timer tick updates
    settings/                       -- Settings scene
      scene.go
      systems/
        render.go
        input.go
        scroll.go
    minigame/                       -- Button Hunt scene (fullscreen transparent)
      scene.go
      game.go                       -- Pure game logic (existing, no Ebiten)
      game_test.go                  -- Game logic tests (existing)
      systems/
        render.go                   -- Draws targets, HUD, game-over
        input.go                    -- Click -> hit detection
        spawn.go                    -- Batch spawning on last target
    lockscreen/                     -- Lock screen scene (fullscreen opaque)
      scene.go
      lock.go                       -- Pure lock logic (existing, no Ebiten)
      lock_test.go                  -- Lock logic tests (existing)
      systems/
        render.go                   -- Progress bar, countdown, message
        tick.go                     -- Checks completion
pkg/
  systems/                          -- Reusable systems (importable across projects)
    input.go                        -- Generic click detection via RTree
    input_test.go
    debug.go                        -- FPS/TPS overlay
    debug_test.go
assets/
  embed.go                          -- //go:embed directives
  fonts/NotoSans-{Regular,Bold}.ttf
  sounds/{tick,alarm}.mp3
```

### How ECS Works in Each Scene

Example: MinigameScene

```
Entities (in Registry):
  group "target" -> Target entities (Position + Renderable + Clickable)
  group "hud"    -> HUD entity (Position + Text)
  group "score"  -> Score entity (Text)

Components on a Target entity:
  - Position {X: 450, Y: 230}
  - Clickable {Spatial: shapes.NewSphere(Point, radius)} -> inserted into RTree
  - Renderable {Color: purple, Radius: 15, Shape: Circle}
  - Alive {Value: true}

Systems (execution order):
  1. SpawnSystem    — checks alive count, spawns batches
  2. InputSystem    — mouse click -> RTree.Collision() -> find hit entity -> mark dead
  3. RenderSystem   — iterates "target" group, draws alive entities
  4. HUDSystem      — draws score, time, ESC hint (SystemWindow.ScreenDraw)
```

Adding a new entity type (e.g., power-ups) = new component + entities in registry.
No system code changes unless new behavior is needed.

### Cross-Scene Communication

Scenes communicate through the shared event.Bus:

```
Timer domain fires events -> Bus -> Scenes react
  FocusCompleted -> MinigameScene.Load() (if enabled)
  LongBreakStarted -> LockscreenScene.Load()
  BreakCompleted -> back to TimerScene
  ConfigChanged -> all scenes update their state
```

Scene switching is triggered by event handlers in app.go calling
manager.SwitchSceneTo(). Scenes never import each other.

### Reusable Systems (pkg/systems/)

Systems that work across any scene:

**InputSystem** — generic click detection:
- Takes RTree from BaseScene
- On mouse click, queries RTree.Collision() with click point
- Finds entities with Clickable component
- Calls entity's OnClick handler
- Reusable: timer buttons, settings widgets, minigame targets all use the same system

**DebugSystem** — FPS/TPS overlay:
- Implements SystemWindow (ScreenDraw)
- Shows performance stats on any scene

Module-specific systems go in `modules/<name>/systems/` when they contain
logic unique to that module (e.g., SpawnSystem for minigame).

## Domain Logic (unchanged)

### Timer States

```
IDLE -> FOCUS -> (auto/manual) -> BREAK -> (auto/manual) -> FOCUS -> ...
                                  LONG_BREAK (every N rounds)
Any state -> PAUSED -> resume to previous state
Any state -> IDLE (reset)
```

`internal/timer/` is pure Go, no Ebiten. Fully tested.
`internal/config/` is pure Go persistence. Fully tested.
`internal/modules/minigame/game.go` is pure game logic. Fully tested.
`internal/modules/lockscreen/lock.go` is pure lock logic. Fully tested.

### Mini-Game: Button Hunt

Visual search game during short breaks. Fullscreen, fully transparent background.
10 targets (4 tiny, 3 small, 2 medium, 1 large) spawn randomly.
Click to remove; new batch at 1 remaining. Best score persisted.
ESC to close. See game.go + game_test.go for full logic.

## Implementation Plan

### Phase 1: Core Infrastructure

1. Add `InsideGallery/core` and `InsideGallery/game-core` dependencies
2. Create `internal/core/` — System, SystemWindow, Systems (following detective)
3. Create `internal/scene/` — Scene interface, BaseScene, Manager
4. Create `internal/ecs/components/` — Position, Clickable, Renderable, Text
5. Create `pkg/systems/input.go` — generic RTree-based click detection
6. Tests for Systems, BaseScene, Manager, InputSystem

### Phase 2: Timer Scene

1. Create `internal/modules/timer/scene.go` — embeds BaseScene
2. Create timer-specific systems (render, input, tick)
3. Create entities for buttons, ring, labels
4. Wire into app.go via SceneManager
5. Verify identical behavior to current UI

### Phase 3: Settings Scene

1. Create `internal/modules/settings/scene.go`
2. Systems for rendering, input, scroll
3. Entities for sliders, toggles, labels
4. Scene switching: timer <-> settings

### Phase 4: Minigame + Lockscreen Scenes

1. Port minigame module.go/screen.go to scene + systems pattern
2. Port lockscreen module.go/screen.go to scene + systems pattern
3. Fullscreen management in scene Load/Unload
4. Event-driven scene switching

### Phase 5: Cleanup

1. Remove old `internal/module/`, `internal/ui/screen.go`
2. Remove old screen_timer.go, screen_settings.go (replaced by scenes)
3. Keep `internal/ui/` drawing primitives (still used by render systems)

## External Plugin Modules (.so)

Future: modules can be compiled as Go plugins (.so) and loaded at runtime.

### How It Works

```go
// Plugin interface (defined in a shared package)
type PluginModule interface {
    Scenes() []scene.Scene
}

// Each .so exports:
var Plugin PluginModule
```

App scans `~/.config/pomodoro/plugins/` at startup, loads each .so via
Go's `plugin` package, and registers the returned scenes with the Manager.

### Limitations

- Go plugins require Linux or macOS (no Windows support)
- Plugin and host must be compiled with the same Go version
- Shared dependencies must match exact versions
- Plugins cannot be unloaded once loaded

### Alternative: Process-Based Plugins

For cross-platform support, plugins can run as separate processes
communicating via Unix sockets or gRPC. Higher overhead but no
Go version coupling. This is a longer-term option.

### Plugin Development Flow

1. Developer creates a new Go module importing `pomodoro/pkg/systems`
2. Implements Scene(s) using BaseScene + own systems
3. Builds with `go build -buildmode=plugin`
4. Drops .so into plugins directory
5. Pomodoro discovers and loads it on next startup

## Future: Tiled-Based UI

UI layouts defined as .tmx maps in Tiled editor. Button zones as ObjectGroups
with collision shapes, loaded into RTree. Click detection becomes spatial query.
Theme switching = load different .tmx file.

This is planned after the ECS foundation is solid. The minigame scene will be
the first candidate (targets as map objects). Timer/Settings scenes will follow.

## Code Conventions

- Pure domain logic in `internal/timer/`, `internal/config/` — no Ebiten
- Pure game logic in `modules/*/game.go`, `modules/*/lock.go` — no Ebiten
- Systems contain behavior, Components contain data, Entities are IDs
- Reusable systems in `pkg/systems/`, module-specific in `modules/*/systems/`
- Scenes never import other scenes — communicate via event.Bus only
- Test coverage >= 70% on non-Ebiten code, enforced in CI
- Each scene is fully autonomous — Load/Unload manages its own lifecycle

## Build & Run

```bash
make build           # Build binary to build/pomodoro
make test            # Run all tests
make appimage        # Build AppImage
make install         # Install to /usr/local
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
| Mini-Game on Break | on/off | off |
| Theme | dark/light | dark |
| Transparency | 10-90% | 10% |
