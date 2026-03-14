package fingerprint

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/config"
	"github.com/InsideGallery/pomodoro/pkg/event"
	"github.com/InsideGallery/pomodoro/pkg/pluggable"
	"github.com/InsideGallery/pomodoro/pkg/resources"
	"github.com/InsideGallery/pomodoro/pkg/scene"
)

// Plugin implements pluggable.Module for the fingerprint puzzle.
type Plugin struct{}

func (p *Plugin) Name() string         { return "fingerprint" }
func (p *Plugin) ConfigKey() string    { return "fingerprint_enabled" }
func (p *Plugin) DefaultEnabled() bool { return false }
func (p *Plugin) TrayItems() map[string]string {
	return map[string]string{"Fingerprint Lab": LoadingSceneName}
}

func (p *Plugin) Scenes(bus *event.Bus, switchScene pluggable.SceneSwitcher) []scene.Scene {
	cfg := config.Load()

	puzzle := NewPuzzleScene(
		func(name string) { switchScene(name) },
		cfg.BreakDuration(),
	)

	loading := NewLoadingScene(
		func(name string) { switchScene(name) },
		PuzzleSceneName,
		func(base *scene.BaseScene) {
			LoadResources(base.Resources)
		},
	)

	// Self-activate on break start (if enabled)
	bus.Subscribe(event.BreakStarted, func(_ event.Event) {
		c := config.Load()
		if c.PluginEnabled("fingerprint_enabled", false) {
			switchScene(LoadingSceneName)
		}
	})

	bus.Subscribe(event.ConfigChanged, func(e event.Event) {
		if c, ok := e.Data.(config.Config); ok {
			cfg = c
		}
	})

	return []scene.Scene{loading, puzzle}
}

// LoadResources loads all fingerprint assets asynchronously.
func LoadResources(rm *resources.Manager) {
	assetsDir := findAssetsDir()
	if assetsDir == "" {
		return
	}

	var tasks []resources.LoadTask

	// Desktop scene resources
	desktopFiles := map[string]string{
		"bg_static":   "Фон (не анімований).png",
		"bg_bright":   "екран (підвищена яскраввість).png",
		"bg_dim":      "екран (понижена яскравість).png",
		"wallpaper":   "робочий стіл (фон).png",
		"cursor":      "курсор.png",
		"app_frame":   "рама.png",
		"workspace":   "Робоче поле Дактилоскопії.png",
		"grid":        "Робоче поле Дактилоскопії (сітка 0-9).png",
		"highlighter": "Відбитки/highlighter.png",
	}

	for key, file := range desktopFiles {
		k, f := key, file

		tasks = append(tasks, resources.LoadTask{
			Key: k,
			Load: func() (any, error) {
				return loadImage(filepath.Join(assetsDir, f))
			},
		})
	}

	// Load avatars (1-5.jpg)
	for i := 1; i <= 5; i++ {
		idx := i

		tasks = append(tasks, resources.LoadTask{
			Key: fmt.Sprintf("avatar_%d", idx),
			Load: func() (any, error) {
				return loadImage(filepath.Join(assetsDir, fmt.Sprintf("%d.jpg", idx)))
			},
		})
	}

	// Load UI buttons
	buttons := []string{
		"place button.png", "code button.png", "send button.png",
		"success button.png", "fail button.png", "fingerprinting.png",
	}

	for _, btn := range buttons {
		name := btn

		tasks = append(tasks, resources.LoadTask{
			Key: "ui_" + name,
			Load: func() (any, error) {
				return loadImage(filepath.Join(assetsDir, name))
			},
		})
	}

	// Load base fingerprint images (full, for programmatic cutting)
	colors := []string{"green", "blue", "red", "yellow"}

	for _, clr := range colors {
		for variant := 1; variant <= 4; variant++ {
			c, v := clr, variant

			tasks = append(tasks, resources.LoadTask{
				Key: fmt.Sprintf("fp_%s_%d", c, v),
				Load: func() (any, error) {
					dir := filepath.Join(assetsDir, "Відбитки", "шматочки пазлу", fmt.Sprintf("%s %d", c, v))
					path := filepath.Join(dir, fmt.Sprintf("%s%d centered.png", string(c[0]), v))

					if _, err := os.Stat(path); err != nil {
						// Try alternate naming
						path = filepath.Join(dir, fmt.Sprintf("%s%d.png", string(c[0]), v))
					}

					return loadImage(path)
				},
			})
		}
	}

	// Load loading animation frames
	for i := 1; i <= 4; i++ {
		idx := i

		tasks = append(tasks, resources.LoadTask{
			Key: fmt.Sprintf("loading_%d", idx),
			Load: func() (any, error) {
				return loadImage(filepath.Join(assetsDir, fmt.Sprintf("loading %d.png", idx)))
			},
		})

		tasks = append(tasks, resources.LoadTask{
			Key: fmt.Sprintf("loading_%da", idx),
			Load: func() (any, error) {
				return loadImage(filepath.Join(assetsDir, fmt.Sprintf("loading %dа.png", idx)))
			},
		})
	}

	rm.LoadAsync(tasks)
}

func loadImage(path string) (*ebiten.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	return ebiten.NewImageFromImage(img), nil
}

func findAssetsDir() string {
	// Check common locations
	candidates := []string{
		"assets/external/fingerprint",
		"../assets/external/fingerprint",
		filepath.Join(os.Getenv("HOME"), ".config", "pomodoro", "assets", "fingerprint"),
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	return ""
}
