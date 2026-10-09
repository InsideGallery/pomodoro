# Building the App and Plugins

This guide covers building the Pomodoro timer and writing, building, and installing external
plugins (`.so` files).

## 1. Build the app

You need Go 1.25 or later (see `go.mod`) and a C compiler, because Ebitengine uses cgo on Linux
and macOS.

### Linux

```bash
sudo apt install libx11-dev libgl1-mesa-dev libxcursor-dev libxrandr-dev \
  libxinerama-dev libxi-dev libasound2-dev libayatana-appindicator3-dev libxxf86vm-dev
make build          # -> build/pomodoro
make appimage       # -> build/pomodoro-<version>-linux-amd64.AppImage
```

### macOS

```bash
xcode-select --install
CGO_ENABLED=1 go build -o build/pomodoro ./services/pomodoro/cmd/pomodoro/
```

### Windows

```bash
go build -ldflags "-H windowsgui" -o build/pomodoro.exe ./services/pomodoro/cmd/pomodoro/
```

The entry point is `services/pomodoro/cmd/pomodoro/`. The release workflow
(`.github/workflows/release.yml`) builds the same package for linux, windows, and darwin on amd64
and arm64.

## 2. How plugins work

There are two kinds of plugin:

| Kind | Where it lives | Platforms |
|---|---|---|
| Built-in | `pkg/plugins/<name>`, registered in `services/pomodoro/internal/builtin/plugins.go` | All |
| External `.so` | `plugins/<name>/main.go`, built with `-buildmode=plugin` | Linux and macOS only |

At startup, `pkg/app` scans `~/.config/pomodoro/plugins/` for `*.so` files. It opens each file
and looks up an exported `Plugin` symbol. It then registers every scene that the plugin returns.

Every plugin implements `pluggable.Module` (`pkg/pluggable/contract.go`):

```go
type Module interface {
	Name() string                                                    // unique id
	Scenes(bus *event.Bus, switchScene SceneSwitcher) []scene.Scene  // scenes it adds
	TrayItems() map[string]string                                    // label -> scene name
	ConfigKey() string                                               // enable/disable key in config.json
	DefaultEnabled() bool
}
```

## 3. Write an external plugin

1. Create `plugins/<name>/main.go`. Start from `plugins/example/main.go`.
2. Put `//go:build plugin` on the first line. This keeps the file out of `go build ./...`,
   `go test ./...`, and lint.
3. Use `package main` and export the module as a package-level variable named exactly `Plugin`:

   ```go
   //go:build plugin

   package main

   var Plugin pluggable.Module = &myPlugin{} //nolint:gochecknoglobals // plugin contract
   ```

4. Return your scenes from `Scenes`. Each scene implements `scene.Scene`: `Name`, `Init`, `Load`,
   `Unload`, `Update`, `Draw`, and `Layout`. Embed `*scene.BaseScene` and create it in `Init`.
5. Switch scenes with the `switchScene` callback, for example `switchScene("timer")` to return to
   the main timer. Subscribe to timer events through `bus`. See `pkg/plugins/minigame` for a scene
   that reacts to break events.

Put reusable logic in `pkg/plugins/<name>` and keep `plugins/<name>/main.go` as a thin wrapper, the
same way `minigame`, `lockscreen`, and `metrics` do. The wrapper can then also be registered as a
built-in.

### Scenes you can switch to

`switchScene(name)` takes a scene name. The Pomodoro app registers these scenes:

| Scene name | Defined in | What it is |
|---|---|---|
| `timer` | `services/pomodoro/internal/modules/timer` (`timer.SceneName`) | The main app window: the timer, its controls, and the settings button. **Use this to return to the main app.** |
| `settings` | `services/pomodoro/internal/modules/settings` | The Settings screen. Its back button returns to `timer`. |
| `mini` | `services/pomodoro/internal/modules/mini` | The compact mini window. It resizes the window to the mini size in `Load`. |
| `minigame` | `pkg/plugins/minigame` | Built-in plugin: Button Hunt on a fullscreen window during short breaks. |
| `lockscreen` | `pkg/plugins/lockscreen` | Built-in plugin: a fullscreen lock during long breaks. |
| `metrics` | `pkg/plugins/metrics` | Built-in plugin: usage statistics, opened from the tray item "Metrics". |

The `services/pomodoro/internal/...` packages are internal, so a plugin can't import their
`SceneName` constants. Pass the name as a string literal instead, for example `"timer"`.

When you switch scenes, the scene manager calls `Load()` on the new scene first and then
`Unload()` on the previous one. If an unknown name is passed, the app logs a warning and stays on
the current scene.

### Returning to the main app

Always give the user a way back to `timer`, usually with Esc and when the plugin's work is done.
In `Unload`, undo any window changes the scene made. The timer keeps counting while your scene is
on screen, because the app advances it every frame whatever scene is active.

```go
func (s *myScene) Load() error {
	ebiten.SetFullscreen(true) // optional: take over the screen
	return nil
}

func (s *myScene) Unload() error {
	ebiten.SetFullscreen(false) // restore the window before the timer comes back
	return nil
}

func (s *myScene) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switchScene("timer") // back to the main app
	}
	return nil
}
```

Switching to `timer` doesn't resize the window. The full size (380×560) is restored only by the
mini window's own exit button. If your scene opens while the mini window is showing, for example
on a `BreakStarted` event, then after `switchScene("timer")` the timer is drawn in the small mini
window. To prevent this, restore the size yourself before returning, with
`ebiten.SetWindowSize(380, 560)`, or return to `"mini"` if that is where the user was. Switching to
`mini` resizes the window in its own `Load`.

### Opening your scene automatically

An external plugin has no tray item or Settings toggle (see section 6), so it opens its scene by
listening to timer events on the bus. This is the same pattern the built-in `minigame` plugin uses:

```go
func (p *myPlugin) Scenes(bus *event.Bus, switchScene pluggable.SceneSwitcher) []scene.Scene {
	bus.Subscribe(event.BreakStarted, func(_ event.Event) {
		switchScene("myscene")
	})
	return []scene.Scene{&myScene{bus: bus, switchScene: switchScene}}
}
```

The events are defined in `pkg/event/types.go`: `FocusStarted`, `FocusCompleted`, `BreakStarted`,
`BreakCompleted`, `LongBreakStarted`, `LongBreakCompleted`, `Paused`, `Resumed`, `Reset`, `Tick`,
and `ConfigChanged`. `ConfigChanged` carries the full `config.Config` (`pkg/config`) in
`Event.Data`. Read it to follow user settings, such as `c.BreakDuration()` or
`c.PluginEnabled("<key>", false)`.

## 4. Build and install plugins

```bash
make plugins
```

This builds every `plugins/*/main.go` with:

```bash
go build -buildmode=plugin -tags plugin -o ~/.config/pomodoro/plugins/<name>.so ./plugins/<name>/
```

To build a single plugin:

```bash
mkdir -p ~/.config/pomodoro/plugins
go build -buildmode=plugin -tags plugin -o ~/.config/pomodoro/plugins/myplugin.so ./plugins/myplugin/
```

Then restart the app.

### Compatibility rules (Go `plugin` package)

A `.so` loads only into an app binary built:

- with the **same Go toolchain version**;
- from the **same versions of every shared package**, which means the same checkout of this
  repository and the same `go.sum`;
- with the **same `-trimpath` and `-tags` choices** where they affect shared packages;
- with cgo enabled (the default on Linux and macOS).

In practice, build the app (`make build`) and the plugins (`make plugins`) from the same checkout
on the same machine. A plugin built locally will usually **not** load into a binary downloaded from
GitHub Releases.

Windows has no Go `plugin` support. On Windows, a plugin must be compiled in as a built-in (see
section 5).

## 5. Make a plugin built-in

1. Move the logic into `pkg/plugins/<name>`.
2. Add a wrapper type and append it to `Modules()` in
   `services/pomodoro/internal/builtin/plugins.go`.
3. Rebuild the app.

Built-in plugins get a toggle in Settings, which is saved under `ConfigKey()` in
`~/.config/pomodoro/config.json`. When enabled, they also get a tray menu item for each
`TrayItems()` entry.

## 6. Current limitations

- **External plugins get no Settings toggle or tray items.** Only built-in modules are passed to
  the settings scene and the tray. The scenes of a `.so` plugin are registered, but the plugin must
  make itself reachable, for example by switching scenes in response to a bus event.
- **Names must be unique.** Scenes are stored by name. Built-ins are registered after `.so`
  plugins, so a `.so` scene named `minigame`, `lockscreen`, or `metrics` is replaced by the
  built-in scene with that name. Building those three plugins as `.so` files has no effect in the
  Pomodoro app.
- **Load errors are silent.** A `.so` file that fails to open, has no `Plugin` symbol, or has the
  wrong symbol type is skipped without a log line. If a plugin doesn't show up, check the
  compatibility rules above. You can open the file directly to see the error:

  ```go
  _, err := plugin.Open(os.ExpandEnv("$HOME/.config/pomodoro/plugins/myplugin.so"))
  fmt.Println(err)
  ```

## 7. Remove plugins

```bash
rm ~/.config/pomodoro/plugins/<name>.so
```
