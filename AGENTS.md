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

**Domain model** (pkg/plugins/fingerprint/domain/, 11 tests):
- Tile: uint32 = EncodeTile(x, y, rotation, content)
- Fingerprint: color uint8 + tiles → 9-digit SHA256 hash
- Person + Database: lookup by fingerprint UniqueID
- Case: unsolved/solved/failed, submit against DB
- PuzzleGenerator: solved → remove tiles → puzzle + pieces (rotated)

**Assets** (assets/external/fingerprint/):
- CRT monitor background (Фон), desktop wallpaper (робочий стіл фон)
- Screen brightness variants (підвищена/понижена яскравість)
- Window frame (рама), workspace with grid (Робоче поле)
- Custom cursor (курсор.png), highlighter (підсвітка)
- 3-column app layout (Вікно вибору відбитка) with wireframe (Трасування колонок)
- 16 colored fingerprints (4 colors × 4 variants, each 100 pieces) + 4 grey
- 5 suspect avatars, UI buttons with hover states
- Loading animation 8 frames, success/fail stamps
- Design: Sylfaen 44pt codes, Georgia 72+43pt text, #d5f2f1 #ffffff #4d4b4b

**Scene flow:**
1. Loading — animated preloader, async resource loading
2. Desktop — CRT with boot animation, app icon clickable
3. App — 3-column: cases | codes | suspect profile
4. Puzzle — 10×10 grid, place/rotate pieces, choose color, submit

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
