# Pomodoro: Lightweight Productivity Timer

A minimalist, cross-platform Pomodoro timer built with Go and the [Ebiten](https://ebitengine.org/) game engine for butter-smooth, hardware-accelerated performance — no Electron overhead.

![Go Version](https://img.shields.io/github/go-mod/go-version/InsideGallery/pomodoro)
![License](https://img.shields.io/github/license/InsideGallery/pomodoro)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20windows%20%7C%20macos-lightgrey)

---

## Interface

| Dark Mode | Light Mode |
|:---:|:---:|
| <img src="assets/screenshots/Screenshot%20from%202026-03-13%2018-08-19.png" width="300"> | <img src="assets/screenshots/Screenshot%20from%202026-03-13%2018-08-35.png" width="300"> |

| Dark Settings | Light Settings |
|:---:|:---:|
| <img src="assets/screenshots/Screenshot%20from%202026-03-13%2018-08-25.png" width="300"> | <img src="assets/screenshots/Screenshot%20from%202026-03-13%2018-08-32.png" width="300"> |

| Mini Mode (Dark) | Mini Mode (Light) |
|:---:|:---:|
| <img src="assets/screenshots/Screenshot%20from%202026-03-13%2018-07-35.png" width="250"> | <img src="assets/screenshots/Screenshot%20from%202026-03-13%2018-08-39.png" width="250"> |

---

## Key Features

- **Game Engine Powered** — Leveraging Ebiten for hardware-accelerated rendering at 60 FPS
- **ECS Architecture** — Entity-Component-System with Scene Manager, RTree spatial indexing, event-driven communication
- **Plugin System** — Features as plugins: compiled-in (Windows) or `.so` runtime loading (Linux/macOS)
- **Dual Modes** — Full mode and compact mini-mode overlay
- **Mini-Game** — Button Hunt during breaks: find targets on transparent fullscreen overlay
- **Lock Screen** — Fullscreen lock during long breaks (soft-lock with ESC x3 exit)
- **Usage Metrics** — Track focus hours, break time, games played (total/monthly/weekly)
- **Fully Customizable** — Focus/break times, dark/light themes, sound volumes, transparency
- **System Tray** — Close to tray, dynamic menu items from plugins
- **Cross-Platform** — Linux, macOS, Windows. Single binary, all plugins compiled in.
- **HiDPI Support** — Crisp vector rendering on high-density displays

---

## Architecture

```
internal/                          -- Core (always compiled in)
  app/                             -- Ebiten game shell
  modules/timer/                   -- Timer scene + systems
  modules/settings/                -- Settings scene (dynamic plugin toggles)
  modules/mini/                    -- Mini mode scene
  builtin/                         -- Registers all built-in plugins
  timer/                           -- Pure Go timer state machine
  audio/                           -- Audio manager

pkg/                               -- Public API (importable by external plugins)
  scene/                           -- Scene, BaseScene, SceneManager
  event/                           -- Event Bus
  core/                            -- ECS System interfaces
  systems/                         -- InputSystem (RTree), DebugSystem
  config/                          -- Config persistence
  ui/                              -- Drawing primitives, widgets
  platform/                        -- Window management
  pluggable/                       -- Plugin contract + loader
  plugins/                         -- Plugin logic packages (shared by .so and builtin)
    minigame/                      -- Button Hunt game logic + scene
    lockscreen/                    -- Lock screen logic + scene
    metrics/                       -- Metrics store + scene

plugins/                           -- .so plugin entry points (Linux/macOS only)
  minigame/main.go                 -- Thin wrapper importing pkg/plugins/minigame
  lockscreen/main.go               -- Thin wrapper importing pkg/plugins/lockscreen
  metrics/main.go                  -- Thin wrapper importing pkg/plugins/metrics
  example/main.go                  -- Example plugin template
```

**How plugins work:**

| Platform | Plugin mode | How |
|----------|-----------|-----|
| **Linux** | Runtime `.so` OR compiled-in | `make plugins` builds .so; builtin always available |
| **macOS** | Runtime `.dylib` OR compiled-in | Same as Linux |
| **Windows** | Compiled-in only | Go's plugin package not supported; all plugins compiled in |

All plugins are compiled into the binary by default (via `internal/builtin/`).
On Linux/macOS, external `.so` plugins from `~/.config/pomodoro/plugins/` can
override or extend the built-in ones.

---

## Installation

### Download Binary

Grab the latest executable from [Releases](https://github.com/InsideGallery/pomodoro/releases).

### Build from Source

Requires Go 1.25+ and a C compiler.

**Linux**:
```bash
sudo apt install libx11-dev libgl1-mesa-dev libxcursor-dev libxrandr-dev \
  libxinerama-dev libxi-dev libasound2-dev libayatana-appindicator3-dev libxxf86vm-dev
make build
```

**Windows / macOS**:
```bash
go build -o pomodoro ./cmd/pomodoro/
```

---

## Configuration

Settings in `~/.config/pomodoro/config.json` (or press **S** / click gear icon).

| Setting | Default |
|---|---|
| Focus duration | 25 min |
| Break duration | 5 min |
| Long break duration | 15 min |
| Rounds before long break | 4 |
| Auto-start next session | off |
| Tick sound | on, 50% volume |
| Alarm volume | 80% |
| Theme | dark |
| Transparency | 10% |

Plugin toggles (Mini-Game, Lock Screen, Metrics) appear dynamically based on loaded plugins.

---

## Building

```bash
make build          # Build binary (all plugins compiled in)
make plugins        # Build .so plugins for Linux/macOS runtime loading
make test           # Run tests
make lint           # Run golangci-lint
make coverage       # Run test coverage
make appimage       # Build Linux AppImage
make clean          # Remove build artifacts
```

---

## Writing Plugins

Plugins implement the `pluggable.Module` interface:

```go
type Module interface {
    Name() string
    Scenes(bus *event.Bus, switchScene SceneSwitcher) []scene.Scene
    TrayItems() map[string]string
    ConfigKey() string
    DefaultEnabled() bool
}
```

See `plugins/example/main.go` for a minimal template.

---

## Roadmap

- [ ] Tiled-based UI layouts (.tmx maps for data-driven UI)
- [ ] Camera system from detective project
- [ ] D-Bus notifications plugin
- [ ] Custom sound file support
- [ ] Task list plugin

---

## License

[Apache License 2.0](LICENSE)
