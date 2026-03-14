# Pomodoro Timer - Development Guide

## Architecture: ECS + Scene + Plugin

Every scene has its own ECS (Systems, Registry, RTree). Every interactive element
is an entity with components. No widget self-detection — all input via RTree.

### Core Principles

1. **Every UI element is an entity** in the Registry with typed components
2. **Systems process entities** — InputSystem queries RTree, RenderSystem draws from Registry
3. **No widget Update()** — widgets are pure rendering structs. All behavior lives in systems.
4. **RTree for ALL input** — buttons, sliders, toggles, ring drag, dot clicks. No manual hit checks.
5. **Each scene = mini ECS app** with own Systems + Registry + RTree + Bus
6. **Plugins are modules** compiled into the binary. Each provides scenes with own ECS.
7. **Never ignore errors**. Log or handle every error.

### Entity-Component Model

```go
// Components (pure data, no methods beyond accessors)
type Position struct { X, Y float64 }
type Size struct { W, H float64 }
type Clickable struct { OnClick func() }
type Draggable struct { OnDrag func(mx, my int); OnDragEnd func() }
type Hoverable struct { Hovered bool; OnHover func(bool) }
type Visual struct {
    Shape    ShapeType  // Circle, Rect, RoundedRect, Arc, Ring
    Color    color.RGBA
    Radius   float64
}
type Label struct {
    Text  string
    Face  *textv2.GoTextFace
    Color color.RGBA
    Align TextAlign  // Left, Center, Right
}
type SliderState struct { Min, Max, Value float64; OnChange func(float64) }
type ToggleState struct { Value bool; OnColor, OffColor color.RGBA; OnChange func(bool) }
type RingProgress struct { Progress float64; Width float64; StartColor, EndColor color.RGBA }
type RoundDots struct { Total, Completed int; DotRadius float64; Color, InactiveColor color.RGBA }
type TimerDisplay struct {} // marker: this entity shows the timer text
```

### Entity Groups in Registry

```go
registry.Registry[string, uint64, any]

// Timer scene groups:
"button"        → Start, Reset, Skip, Settings, Close, Mini buttons
"ring"          → Progress ring entity
"timer_text"    → MM:SS display entity
"mode_label"    → "Focus" / "Break" / "Paused" label
"round_dots"    → Round indicator dots
"hint"          → "Ready to focus" hint text

// Settings scene groups:
"slider"        → Focus, Break, LongBreak, Rounds, TickVol, AlarmVol, Transparency
"toggle"        → TickSound, AutoStart, Theme, + dynamic plugin toggles
"button"        → Back, Reset Defaults
"label"         → Section titles (TIMER, SOUND, APPEARANCE)

// Minigame scene groups:
"target"        → Game targets (Position + Visual + Clickable)
"hud"           → Score, Best, Time labels
```

### Systems (execution order per scene)

```
Timer scene:
  1. InputSystem        — RTree queries for buttons. Ring drag via DragZone. Dot clicks via Clickable.
  2. KeyboardSystem     — Space, R, S shortcuts
  3. TickSystem         — timer.Update(), event publishing, audio
  4. RenderSystem       — iterates Registry groups, draws each entity by components

Settings scene:
  1. ScrollSystem       — handles wheel/arrow scroll, updates scroll offset
  2. InputSystem        — RTree queries with scroll offset for all widgets
  3. RenderSystem       — iterates Registry groups, draws widgets

Minigame scene:
  1. InputSystem        — RTree queries for targets (smallest radius = highest priority)
  2. SpawnSystem        — checks alive count, spawns batches
  3. TimerSystem        — checks game over
  4. RenderSystem       — draws targets, HUD

Lockscreen scene:
  1. LockSystem         — checks completion, handles ESC×3
  2. RenderSystem       — draws countdown, progress bar
```

### RTree for Ring Drag

The ring is registered as a ring-shaped zone (annular region). On drag start,
the DragHandler receives mouse X/Y and converts to angle → progress using
the ring's center and radius (stored in the entity's RingProgress component).

```go
// Ring drag zone: register as a large box covering the ring area.
// DragHandler does the angular math.
ringZone := &Zone{
    Spatial: shapes.NewBox(centerX-outerR, centerY-outerR, outerR*2, outerR*2),
    OnDragStart: func() { /* check if click is near ring band */ },
    OnDrag: func(mx, my int) {
        angle := math.Atan2(float64(my)-centerY, float64(mx)-centerX)
        progress := (angle + math.Pi/2) / (2 * math.Pi)
        // update timer remaining from progress
    },
}
```

### RTree for Round Dots

Each dot is a separate entity with Position + Clickable. Dots are re-created
in the RenderSystem when round count changes. Their Clickable.OnClick calls
SetRound(i).

### Removing Old Code

The following will be deleted when ECS is fully implemented:

```
pkg/ui/screen_timer.go      → replaced by timer scene's systems + entities
pkg/ui/screen_settings.go   → replaced by settings scene's systems + entities
pkg/ui/components.go         → Button/Slider/Toggle structs become entity templates
                               Draw functions become part of RenderSystem
                               Hit detection removed entirely (RTree only)
```

Drawing primitives stay in `pkg/ui/draw.go` and `pkg/ui/theme.go` — used by RenderSystems.

### BaseScene provides (every scene gets for free)

```
Systems   *core.Systems                          — ordered named systems
Registry  *registry.Registry[string, uint64, any] — entity storage by group
RTree     *rtree.RTree                            — spatial index for input
Bus       *event.Bus                              — cross-scene events
Camera    *core.Camera                            — pan, zoom, rotate, WorldMatrix
Resources *resources.Manager                      — embedded + disk asset cache with async loading
```

### Camera System

Ported from detective project. Provides world-space transformations.
Scenes can render world content with Camera.WorldMatrix(), then overlay
UI in screen space via SystemWindow.ScreenDraw().

Plugins can manipulate the camera (zoom into puzzle details, pan across game board).

```go
camera.Position    // pan
camera.ZoomFactor  // zoom (1.01^factor)
camera.Rotation    // degrees
camera.WorldMatrix() → ebiten.GeoM
camera.ScreenToWorld(x, y) → worldX, worldY
```

### Resource Manager

Unified resource loading for embedded and disk assets:
- Core resources (fonts, sounds, icons) → from embed.FS (always available)
- Plugin resources (maps, sprites, puzzles) → loaded from disk
- Async loading with progress tracking → preloader scene during load
- Thread-safe cache → Get/Set/GetImage

```go
resources.LoadImageFromFS(embedFS, "fonts/icon.png", "icon")
resources.LoadAsync(tasks) // background loading
resources.Progress() → (loaded, total)
resources.IsLoading() → bool
```

Each plugin can load its own resources. During Load(), a preloader scene
shows progress. On Unload(), resources are released.

### Product Vision

This app grows into a **productivity suite** (like Lunatask):
- Pomodoro timer (core)
- Task manager with kanban/list views
- Break activities (mini-games, puzzles)
- Usage metrics and statistics
- Focus music / ambient sounds

Games during breaks:
- Button Hunt (current) — visual search on transparent fullscreen
- Fingerprint Puzzle (next) — match fingerprint patterns, uses Camera zoom
- More puzzle types via plugins

All features are plugins with own scenes, ECS, resources, and camera access.

### Implementation Plan

Phases 1-3: DONE (timer + settings full ECS, old screens deleted)

#### Phase 4: Minigame + Lockscreen ECS

1. Define all components in `pkg/ecs/components/`
2. Timer scene creates entities in Registry during Load()
3. RenderSystem iterates groups and draws by component type
4. InputSystem registers all entities with Clickable/Draggable in RTree
5. Ring drag via DragZone with angular math in handler
6. Dot clicks via individual Clickable entities
7. Delete TimerScreen struct — all logic in systems + entities
8. Tests: verify entity creation, system execution order

#### Phase 2: Settings Scene Full ECS

1. Settings scene creates slider/toggle/button entities in Registry
2. ScrollSystem manages offset, InputSystem uses SetScrollOffset()
3. Plugin toggles are entities created dynamically from loaded plugins
4. Back button entity in a "fixed" group (not affected by scroll)
5. RenderSystem draws all settings entities
6. Delete SettingsScreen struct

#### Phase 3: Clean Up
L
1. Remove `pkg/ui/screen_timer.go`, `pkg/ui/screen_settings.go`
2. Remove `pkg/ui/components.go` (Button/Slider/Toggle structs)
3. Remove `internal/ecs/components/` (old unused components)
4. Keep `pkg/ui/draw.go` + `pkg/ui/theme.go` (rendering primitives)
5. Keep `pkg/ui/zones.go` (zone creators for entity → RTree registration)
6. Update tests and coverage

#### Phase 4: Minigame + Lockscreen Full ECS

1. Minigame: targets as entities in Registry + RTree
2. Lockscreen: progress bar + labels as entities
3. Both use standard InputSystem + RenderSystem

#### Phase 5: Preloader Scene

1. Generic preloader scene that shows loading progress
2. Plugins call Resources.LoadAsync() during Load()
3. Preloader renders progress bar + animation
4. Transitions to target scene when loading completes

#### Phase 6: Fingerprint Puzzle Plugin

1. New game plugin for break activities
2. Uses Camera (zoom into fingerprint details)
3. Loads puzzle images from disk via Resources
4. Entities: fingerprint pieces, match zones, score
5. Systems: InputSystem for drag-match, RenderSystem with Camera transform

#### Phase 7: Task Manager Plugin

1. Task list scene (CRUD tasks)
2. Tasks linked to pomodoro sessions (focus on a task)
3. Kanban board view (columns: todo, in progress, done)
4. Entities: task cards, columns, drag zones
5. Persistence: tasks stored in JSON alongside config
6. Statistics: tasks completed per day/week

### What Stays

```
pkg/scene/         — Scene, BaseScene(Systems+Registry+RTree+Bus+Camera+Resources), SceneManager
pkg/event/         — Event Bus, types
pkg/core/          — System, SystemWindow, Systems container
pkg/systems/       — InputSystem (RTree zones + scroll), DebugSystem
pkg/config/        — Config persistence
pkg/ui/draw.go     — DrawRoundedRect, DrawArc, DrawCircle, etc.
pkg/ui/theme.go    — Colors, scaling (S/Sf), font Face()
pkg/ui/zones.go    — Zone creators (updated for entity-based components)
pkg/platform/      — Window management
pkg/pluggable/     — Plugin contract + loader
pkg/plugins/       — Plugin logic packages
internal/timer/    — Pure Go timer state machine
internal/builtin/  — Compiled-in plugin registration
internal/app/      — Thin Ebiten shell
internal/tray/     — System tray
internal/audio/    — Audio manager
```

### Code Conventions

- Every interactive element = entity in Registry with components
- All input via RTree — no manual coordinate checks anywhere
- Systems contain behavior, Components contain data
- Each scene manages its own Registry groups
- Plugins provide scenes; scenes provide entities + systems
- Drawing primitives in pkg/ui/draw.go — used by all RenderSystems
- Test coverage >= 70% on non-Ebiten code
- Never ignore errors
