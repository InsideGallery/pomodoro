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
- **Dual Modes** — Full mode and compact mini-mode overlay
- **Mini-Game** — Button Hunt during breaks: find targets on transparent fullscreen
- **Lock Screen** — Optional fullscreen lock during long breaks
- **Usage Metrics** — Track focus hours, break time, games played
- **Fully Customizable** — Focus/break times, dark/light themes, sound volumes, transparency
- **System Tray** — Close to tray, dynamic menu items from plugins
- **Cross-Platform** — Linux, macOS, Windows. Single binary.
- **HiDPI Support** — Crisp vector rendering on high-density displays

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
go build -o pomodoro ./services/pomodoro/cmd/pomodoro/
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

Plugin toggles appear dynamically based on loaded plugins.

---

## Building

```bash
make build      # Build binary
make test       # Run tests
make lint       # Run linter
make appimage   # Build Linux AppImage
make clean      # Remove build artifacts
```

---

## License

[Apache License 2.0](LICENSE)
