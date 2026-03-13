# 🍅 Pomodoro: Lightweight Productivity Timer

A minimalist, cross-platform Pomodoro timer built with Go and the [Ebiten](https://ebitengine.org/) game engine for butter-smooth, hardware-accelerated performance — no Electron overhead.

![Go Version](https://img.shields.io/github/go-mod/go-version/InsideGallery/pomodoro)
![License](https://img.shields.io/github/license/InsideGallery/pomodoro)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20windows%20%7C%20macos-lightgrey)

---

## 📸 Interface

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

## ✨ Key Features

- 🚀 **Game Engine Powered** — Leveraging Ebiten for hardware-accelerated rendering at 60 FPS
- 💎 **Dual Modes** — Switch between an immersive full mode and a non-intrusive minimalist overlay
- 📌 **Always on Top** — Mini mode stays visible over your IDE or browser, keeping you focused
- 🍃 **Lightweight** — No Electron overhead. Small binary size and minimal RAM usage
- 🎨 **Fully Customizable** — Adjustable focus/break times, dark/light themes, sound volumes, and window transparency
- 🔊 **Ambient Tick & Alarm** — Gentle tick keeps you aware of time; alarm brings you back. Independent volume controls for each
- 🖥️ **System Tray** — Close to tray, restore on click, keeps running in the background
- ⌨️ **Keyboard Shortcuts** — Space (start/pause), R (reset), S (settings), M (mini mode), Escape (back)
- 📦 **Single Binary** — No runtime dependencies. One file, zero setup. Settings stored in plain JSON
- 🖼️ **HiDPI Support** — Crisp vector rendering on high-density displays

---

## 🤔 Why Go + Ebiten?

Most desktop timers are either bloated Electron wrappers or ugly CLI tools. This one is different.

Go delivers a small, fast, self-contained binary with no runtime dependencies. Ebiten provides GPU-accelerated rendering, so the UI stays buttery smooth even during heavy system loads — all in under 10 MB of RAM.

The result: a native-feeling desktop app that starts instantly, barely touches your CPU, and looks gorgeous doing it.

---

## 📥 Installation

### Download Binary

Grab the latest executable for your platform from the [Releases](https://github.com/InsideGallery/pomodoro/releases) page.

### AppImage (Linux)

```bash
# Download from Releases, then:
chmod +x pomodoro-*-x86_64.AppImage
./pomodoro-*-x86_64.AppImage
```

### Build from Source

Requires Go 1.25+ and a C compiler.

**Linux** (needs X11 and audio dev libraries):
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
# Installs binary, .desktop file, and icon
```

---

## ⚙️ Configuration

Settings are stored in `~/.config/pomodoro/config.json` and can be adjusted from the in-app settings screen (press **S** or click the gear icon).

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

---

## 🛠️ Building

```bash
make build      # Build binary
make test       # Run tests
make appimage   # Build Linux AppImage
make icon       # Regenerate app icon
make clean      # Remove build artifacts
```

---

## 📋 Roadmap

- [ ] D-Bus notifications for Linux
- [ ] Custom sound file support
- [ ] Pomodoro session statistics and history

---

## 📄 License

[Apache License 2.0](LICENSE)
