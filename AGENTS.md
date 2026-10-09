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
    internal/
      scenes/                       — GameScene (TMX-driven, state machine)
        game.go                     — Scene, zone registration, puzzle logic, persistence
        fingerprint_images.go       — Image pipeline (crop, rotate, mirror, cut 10x10)
        quit_desktop.go             — Desktop quit handler
        quit_mobile.go              — Mobile quit handler
      fsystems/                     — ECS systems
        accessor.go                 — SceneAccessor interface
        state_system.go             — State machine, loading, transitions
        cursor_system.go            — Virtual cursor (delta-based)
        scroll_system.go            — Mouse wheel scrolling with clamping
        dragdrop_system.go          — Piece pickup, rotation, placement
        camera_system.go            — Zoom controls
        render_system.go            — All drawing (layers, UI, puzzle grid)
        text_helpers.go             — Text wrapping utilities
        helpers.go                  — Registry access helpers
      components/                   — Pure data components
        entity.go                   — Entity container
        groups.go                   — Group constants
        state.go                    — Game states enum
        game_data.go                — Game data (DB, cases, selections, loading)
        cursor.go                   — Cursor position + bounds
      entities/                     — Assemblage factories

pkg/                                — Shared framework (importable by all products + plugins)
  app/                              — Generic Ebiten game shell (Config + SetupFunc)
  scene/                            — Scene, BaseScene, SceneManager
  event/                            — Event Bus, Event types (Data any)
  core/                             — System, SystemWindow, Systems, Camera
  systems/                          — InputSystem (RTree + zones), DebugSystem
  config/                           — Config persistence (JSON)
  ui/                               — Drawing primitives (rounded rect, polygon, text, icons)
  tilemap/                          — TMX loader (go-tiled), ObjectBounds, PolygonCentroid
  platform/                         — Window management (X11/macOS/Windows), AssetFS
  pluggable/                        — Plugin contract (Module, Loader, SceneSwitcher)
  resources/                        — Resource manager (async loading, cache, progress)
  ecs/                              — Shared entity component types
  plugins/
    minigame/                       — Button Hunt break game
    lockscreen/                     — Long break lock screen
    metrics/                        — Usage statistics
    fingerprint/                    — Fingerprint puzzle domain logic
      domain/                       — Pure game logic (tile, db, cases, save, story)
        tile.go                     — Tile uint32 encoding, 8-rotation system
        db.go                       — 256 fingerprint records, CRC64 hashing
        cases.go                    — 50 cases x 20 puzzles, decoy generation
        save.go                     — Game save/load (placed pieces, solved/failed)
        story.go                    — Story system, character-avatar mapping
        puzzle_gen.go               — Legacy puzzle generator
        domain_test.go              — 25+ tests

assets/external/fingerprint/        — Game assets
  fingerprint.tmx                   — TMX map (4000x2176, all UI positions)
  stories.json                      — 50 case narratives
  avatars/                          — Character portraits
  fingerprints/                     — {color}.{variant}.png, grey.{variant}.png
  background/                       — Large background PNGs
```

## Architecture

### ECS + Scene + Plugin

See `ecs.md` for full ECS rules and patterns.

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
- No tray, custom cursor
- TMX-driven single scene with state machine
- Loading → Desktop → App → Puzzle
- State: Loading → Disabled → Enabled → ApplicationLayout → ApplicationNet

### Fingerprint Game Design

**TMX-driven**: `fingerprint.tmx` is source of truth for all layout.
Single scene with state machine. See `fingerprint.md` memory file for complete details.

**Domain** (pkg/plugins/fingerprint/domain/, 25+ tests):
- Tile uint32 from (x,y), CRC64 hash, color letter prefix
- Person DB with 256 records (4 colors x 4 variants x 8 rotations x 2 mirror)
- 50 cases with 20 puzzles each, difficulty scaling (3-12 missing pieces)
- Decoy pieces from other fingerprint variants (deterministic RNG)

**System execution order**: State → Cursor → Input → Scroll → DragDrop → Camera → Render

### Input Strategy

- RTree InputSystem with zones: settings, minigame, fingerprint puzzle
- Virtual cursor (delta-based) for fingerprint game
- CursorOverride for custom cursor position
- All zone registration via TMX object groups

### Tilemap

`pkg/tilemap/` — shared TMX loader with:
- FindObjectGroup, FindImageLayer, FindTileLayer
- ObjectBounds (rect + polygon), ObjectPolygonPoints, PolygonCentroid
- ObjectToSpatial (polygon/polyline/box → shapes.Spatial)
- DrawImageLayer, DrawTileLayer

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
- KISS = efficient by design, simple to maintain, smart architecture
