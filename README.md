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
- **ECS Architecture** — Entity-Component-System design using [InsideGallery/core](https://github.com/InsideGallery/core) with Scene Manager, RTree spatial indexing, and event-driven communication
- **Dual Modes** — Switch between an immersive full mode and a non-intrusive minimalist overlay
- **Always on Top** — Mini mode stays visible over your IDE or browser
- **Mini-Game** — Button Hunt game during short breaks: find and click randomly placed targets on a transparent fullscreen overlay
- **Lock Screen** — Optional fullscreen lock during long breaks with soft-lock (always-on-top + focus reclaim, ESC x3 to exit)
- **Usage Metrics** — Track focus hours, break time, games played, sessions started (total/monthly/weekly)
- **Lightweight** — No Electron overhead. Small binary, minimal RAM
- **Fully Customizable** — Focus/break times, dark/light themes, sound volumes, transparency
- **Ambient Tick & Alarm** — Gentle tick keeps you aware; alarm brings you back
- **System Tray** — Close to tray, Show/Metrics/Quit from tray menu
- **Keyboard Shortcuts** — Space (start/pause), R (reset), S (settings), Escape (back)
- **Single Binary** — No runtime dependencies. Settings stored in plain JSON
- **HiDPI Support** — Crisp vector rendering on high-density displays
- **Plugin-Ready Architecture** — Modular scene system designed for future external .so plugins

---

## Architecture

Built on an ECS (Entity-Component-System) + Scene pattern:

- **Scenes**: Timer, Settings, Mini-Game, Lock Screen, Metrics — each self-contained with own lifecycle
- **Systems**: Ordered execution of Update/Draw per scene (InputSystem, TickSystem, RenderSystem)
- **RTree**: Spatial indexing for efficient click/collision detection via [InsideGallery/game-core](https://github.com/InsideGallery/game-core)
- **Event Bus**: Cross-scene communication without module coupling
- **Pure Domain Logic**: Timer state machine, game logic, metrics store — fully tested, no Ebiten dependency

```
cmd/pomodoro/          -- Entry point
internal/
  core/                -- ECS System interface (Update + Draw)
  scene/               -- Scene interface, BaseScene, SceneManager
  modules/
    timer/             -- Main timer scene with systems
    settings/          -- Settings scene
    minigame/          -- Button Hunt fullscreen game
    lockscreen/        -- Long break lock screen
    metrics/           -- Usage statistics
  timer/               -- Pure Go timer state machine (25 tests)
  config/              -- JSON persistence (11 tests)
  event/               -- Event bus (9 tests)
  ui/                  -- Drawing primitives, widget components
  audio/               -- Tick/alarm audio manager
pkg/
  systems/             -- Reusable ECS systems (InputSystem, DebugSystem)
```

---

## Why Go + Ebiten?

Most desktop timers are either bloated Electron wrappers or ugly CLI tools. This one is different.

Go delivers a small, fast, self-contained binary with no runtime dependencies. Ebiten provides GPU-accelerated rendering, so the UI stays buttery smooth — all in under 10 MB of RAM.

---

## Installation

### Download Binary

Grab the latest executable from [Releases](https://github.com/InsideGallery/pomodoro/releases).

### AppImage (Linux)

```bash
chmod +x pomodoro-*-x86_64.AppImage
./pomodoro-*-x86_64.AppImage
```

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

### System Install (Linux)

```bash
sudo make install
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
| Mini-Game on break | off |
| Lock long break | off |
| Usage metrics | off |
| Tick sound | on, 50% volume |
| Alarm volume | 80% |
| Theme | dark |
| Transparency | 10% |

---

## Building

```bash
make build      # Build binary
make test       # Run tests (100+ tests, 86%+ coverage)
make appimage   # Build Linux AppImage
make clean      # Remove build artifacts
```

---

## Roadmap

- [ ] Tiled-based UI layouts (.tmx maps for data-driven UI)
- [ ] External plugin modules (.so) for community extensions
- [ ] D-Bus notifications for Linux
- [ ] Custom sound file support
- [ ] Task list panel linked to focus sessions

---

## License

[Apache License 2.0](LICENSE)
