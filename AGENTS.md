# Development Guide

## Monorepo Structure

Two independent products sharing a common framework:

```
services/
  pomodoro/                         — Pomodoro productivity timer
    cmd/pomodoro/                   — Entry point (tray, transparent window, drag)
    cmd/genicon/                    — Icon generator tool
    internal/
      timer/                        — Pure Go timer state machine (25 tests)
      audio/                        — Audio manager (tick/alarm)
      tray/                         — System tray (dynamic menu items)
      builtin/                      — Compiled-in plugin registration
      modules/
        timer/                      — Timer scene (full ECS, 6 entity groups)
        settings/                   — Settings scene (full ECS, 5 entity groups)
        mini/                       — Mini mode scene

  fingerprint/                      — Fingerprint Lab forensic puzzle game
    cmd/fingerprint/                — Entry point (fullscreen, opaque, custom cursor)

pkg/                                — Shared framework (importable by all products + plugins)
  app/                              — Generic Ebiten game shell (Config + SetupFunc)
  scene/                            — Scene, BaseScene, SceneManager
  event/                            — Event Bus, Event types (Data any)
  core/                             — System, SystemWindow, Systems, Camera
  systems/                          — InputSystem (RTree + scroll + drag), DebugSystem
  config/                           — Config persistence (JSON)
  ui/                               — Drawing primitives (draw.go, theme.go, text.go)
  platform/                         — Window management (X11/macOS/Windows)
  pluggable/                        — Plugin contract (Module, Loader, SceneSwitcher)
  resources/                        — Resource manager (async loading, cache, progress)
  ecs/                              — Shared entity component types
  plugins/
    minigame/                       — Button Hunt break game
    lockscreen/                     — Long break lock screen
    metrics/                        — Usage statistics
    fingerprint/                    — Fingerprint puzzle (domain, scenes, tile cutter)
      domain/                       — Pure game logic (tile, fingerprint, person, case, puzzle gen)
```

## Architecture

### ECS + Scene + Plugin

- Every UI element = entity in Registry with typed components
- Systems process entities (InputSystem, RenderSystem, ScrollSystem)
- All input via RTree — no manual coordinate checks
- Each scene has own Systems + Registry + RTree + Bus + Camera + Resources (via BaseScene)
- Scenes communicate via event.Bus only, never import each other

### App Shell (pkg/app/)

Generic Ebiten game shell. Each product configures via SetupFunc:

```go
app.New(app.Config{
    Width: 380, Height: 560,
    DragEnabled: true,
    Setup: func(ctx, bus, manager, switchScene) string {
        // create scenes, register plugins, return initial scene name
    },
})
```

### Products

**Pomodoro** (services/pomodoro/):
- Transparent, undecorated, draggable window
- System tray with Show/Quit + plugin menu items
- Timer → Settings → Mini mode scene flow
- Plugins: minigame, lockscreen, metrics (compiled-in + .so)

**Fingerprint Lab** (services/fingerprint/):
- Fullscreen, opaque, decorated
- No tray
- Custom cursor
- Loading → Desktop (CRT monitor + boot animation) → App → Puzzle
- Shared resources across scenes via SetResources()

### Input Strategy

- RTree InputSystem: settings (static zones + scroll offset)
- RTree InputSystem: minigame (priority by radius)
- RTree InputSystem: fingerprint puzzle (candidate zones)
- Widget self-detection: timer (ring drag = angular math)
- Rule: RTree for static zones, self-detection for runtime geometry

### Fingerprint Lab Game Design

**TMX-driven**: `fingerprint.tmx` is the source of truth for all layout.
Single scene with state machine (disabled → enabled → app → puzzle).
See `Fingerprint.md` for complete implementation guide.

**Domain** (pkg/plugins/fingerprint/domain/, 16 tests):
- Tile uint32 from (x,y), CRC64 hash, color letter prefix
- Person DB with 100 pre-generated records (db.json)
- Case: 3 hardcoded, difficulty scaling (4-16 missing pieces)
- Decoy pieces from other fingerprint variants

**pkg/tilemap/** — shared TMX loader (reused from detective patterns)

**7 implementation steps** — see Fingerprint.md

## Build Commands

```bash
make build              # Pomodoro timer
make build-fingerprint  # Fingerprint Lab
make build-all          # Both
make test / lint / coverage
```

## Code Conventions

- Every interactive element = entity in Registry
- All input via RTree — no manual coordinate checks
- Systems contain behavior, Components contain data
- Products share pkg/, keep internal/ independent
- Test coverage >= 70% on logic code
- Never ignore errors — use log/slog
- KISS = efficient by design, simple to maintain
