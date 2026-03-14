# Pomodoro Timer - Development Guide

## Architecture

### Core Principles

1. **Core is always loaded**: Timer, Settings, Mini mode — these are internal modules,
   not disableable, compiled into the binary.
2. **Features are plugins**: Minigame, Lockscreen, Metrics — compiled as `.so` files,
   loaded at runtime from `~/.config/pomodoro/plugins/`. User can add/remove.
3. **Settings are dynamic**: Plugin toggles appear ONLY for loaded plugins.
   Each plugin declares its ConfigKey; the settings scene auto-generates toggles.
4. **pkg/ is the public API**: Anything in `pkg/` can be imported by external plugins.
   Anything in `internal/` is private to the core app.
5. **Scenes are autonomous**: Each scene owns its lifecycle (Init/Load/Unload/Update/Draw).
   Scenes communicate only through the event bus. No scene imports another.
6. **Never ignore errors**: Always handle or log errors. No `_ = doSomething()`.

### Input Strategy

**RTree InputSystem** is used in scenes with many static-position widgets (Settings).
The InputSystem queries RTree.Collision() for hit detection, supports hover/click/drag.
For scrollable content, `SetScrollOffset()` converts mouse Y to content-space.

**Widget self-detection** is used in scenes with specialized input (Timer).
Ring drag uses angular math from mouse position. Dot clicks use positions computed
in Draw(). These don't map to static RTree zones. The TimerScreen.Update() handles
all input directly.

**Rule**: Use RTree where zones are static. Use widget self-detection where input
requires runtime geometry (ring drag, scrollable dynamic positions).

### Directory Structure

```
cmd/pomodoro/                       -- Entry point
internal/                           -- Private to the app (not importable by plugins)
  app/app.go                        -- Ebiten Game shell: bus, manager, drag, tray
  modules/
    timer/                          -- Core: timer scene + systems
      scene.go                      -- Owns timer.Timer, audio, state persistence
      systems/tick.go               -- Timer domain updates, event publishing
      systems/render.go             -- Delegates to TimerScreen.Update()/Draw()
      systems/keyboard.go           -- Space/R/S shortcuts
    settings/                       -- Core: settings scene with RTree InputSystem
      scene.go                      -- Zones for all widgets, scroll offset
    mini/                           -- Core: mini mode scene
      scene.go                      -- Compact timer overlay, always-on-top
  timer/                            -- Pure Go timer state machine (25 tests)
  audio/                            -- Audio manager (tick/alarm)
  tray/                             -- System tray (dynamic menu items from plugins)
  ecs/components/                   -- Shared ECS components (Position, Clickable, etc.)
pkg/                                -- Public API (importable by plugins)
  scene/                            -- Scene interface, BaseScene, SceneManager
  event/                            -- Event Bus, Event types (Data any)
  core/                             -- System, SystemWindow, Systems container
  systems/                          -- InputSystem (RTree + scroll), DebugSystem
  config/                           -- Config persistence (JSON)
  ui/                               -- Drawing primitives, widget components
  platform/                         -- Window management (X11, macOS, Windows)
  pluggable/                        -- Plugin contract (Module interface, Loader)
plugins/                            -- External plugins (compiled as .so)
  example/                          -- Minimal example plugin
  minigame/                         -- Button Hunt game (break activity)
  lockscreen/                       -- Long break lock screen (ESC×3 exit)
  metrics/                          -- Usage statistics (total/monthly/weekly)
```

### Plugin Contract

```go
// pkg/pluggable/contract.go
type SceneSwitcher func(name string)

type Module interface {
    Name() string
    Scenes(bus *event.Bus, switchScene SceneSwitcher) []scene.Scene
    TrayItems() map[string]string     // label → scene name
    ConfigKey() string                // e.g. "minigame_enabled"
    DefaultEnabled() bool
}
```

Each `.so` plugin exports `var Plugin pluggable.Module = &myPlugin{}`.

### Plugin Lifecycle

```
1. App starts, creates bus + SceneManager
2. Core scenes registered (timer, settings, mini)
3. Loader scans ~/.config/pomodoro/plugins/*.so
4. For each .so: Open → Lookup("Plugin") → Module
5. Module.Scenes(bus, switchScene) → register with SceneManager
6. Module.TrayItems() → register with tray
7. Module.ConfigKey() → settings scene creates toggle
8. App runs — plugins activate via event subscriptions
```

### Build Commands

```bash
make build          # Build core app (no plugins)
make plugins        # Build .so plugins to ~/.config/pomodoro/plugins/
make test           # Run tests
make lint           # Run golangci-lint
make coverage       # Run test coverage check
make build plugins  # Build everything
```

### Event Types

```
FocusStarted, FocusCompleted       -- timer state
BreakStarted, BreakCompleted       -- short break
LongBreakStarted, LongBreakCompleted  -- long break
Paused, Resumed, Reset             -- user actions
Tick                               -- every frame while running
ConfigChanged                      -- settings changed (Data: config.Config)
```

Events carry `Data any` — used for state string (tray icon) and config (settings).

## TODO

### Immediate

- [ ] Settings toggles must be dynamic (generated from loaded plugins, not hardcoded)
- [ ] Remove hardcoded MinigameToggle/LockBreakToggle/MetricsToggle from screen_settings.go
- [ ] Settings scene queries plugin loader for available plugins and creates toggles
- [ ] Error handling: replace all `_ = ...` with proper logging

### Next

- [ ] Extract pkg/app/ — generic Ebiten game shell reusable across projects
- [ ] Move initApp composition to separate file (cmd/pomodoro/main.go or internal/app/init.go)
- [ ] Clean up dead code (internal/ecs/components/ if unused)

### Future

- [ ] Tiled-based UI (.tmx for layout, RTree for click zones)
- [ ] Timer scene migrated to full entity-based rendering
- [ ] Process-based plugins (gRPC/Unix socket) for cross-platform

## Code Conventions

- Pure domain logic: `internal/timer/`, `pkg/config/` — no Ebiten imports
- Pure game logic: `plugins/*/game.go`, `plugins/*/lock.go` — no Ebiten imports
- Systems: behavior. Components: data. Entities: IDs in Registry.
- Reusable systems: `pkg/systems/`. Module-specific: `modules/*/systems/`.
- Scenes never import other scenes — event.Bus only.
- Test coverage >= 70% on non-Ebiten code.
- Never ignore errors. Log or handle.
