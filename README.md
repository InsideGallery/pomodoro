# Pomodoro

A minimal, distraction-free Pomodoro timer built with Go and [Ebiten](https://ebitengine.org/).

Single binary, no runtime dependencies, transparent borderless window with system tray integration.

![icon](packaging/pomodoro.png)

## Features

- **Focus / Break / Long Break** cycle with configurable durations
- **Dark and Light themes** with adjustable window transparency
- **System tray** — close to tray, restore on click
- **Mini mode** — compact floating timer
- **HiDPI support** — crisp rendering on high-density displays
- **Audio** — tick sound during focus, alarm on completion (adjustable volume)
- **Keyboard shortcuts** — Space (start/pause), R (reset), S (settings), Escape (back)
- **AppImage** packaging for Linux

## Install

### From source

Requires Go 1.25+ and C compiler.

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

### AppImage (Linux)

```bash
make appimage
# Output: build/pomodoro-<version>-x86_64.AppImage
```

### System install (Linux)

```bash
sudo make install
# Installs binary, .desktop file, and icon
```

## Configuration

Settings are stored in `~/.config/pomodoro/config.json`:

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

All settings are adjustable from the in-app settings screen (press S or click the gear icon).

## Building

```bash
make build      # Build binary
make test       # Run tests
make appimage   # Build Linux AppImage
make icon       # Regenerate app icon
make clean      # Remove build artifacts
```

## License

See [LICENSE](LICENSE).
