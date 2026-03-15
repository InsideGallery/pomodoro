package scenes

import (
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/resources"
)

// LoadResources loads all fingerprint game assets asynchronously.
func LoadResources(rm *resources.Manager) {
	assetsDir := FindAssetsDir()
	if assetsDir == "" {
		slog.Warn("fingerprint assets dir not found")

		return
	}

	slog.Info("fingerprint assets", "dir", assetsDir)

	var tasks []resources.LoadTask

	// Small UI resources (load first — fast)
	smallFiles := map[string]string{
		"cursor":      "курсор.png",
		"app_frame":   "рама.png",
		"highlighter": "Відбитки/highlighter.png",
	}

	for key, file := range smallFiles {
		k, f := key, file

		tasks = append(tasks, resources.LoadTask{
			Key:  k,
			Load: func() (any, error) { return loadImage(filepath.Join(assetsDir, f)) },
		})
	}

	// Avatars (1-5.jpg)
	for i := 1; i <= 5; i++ {
		idx := i

		tasks = append(tasks, resources.LoadTask{
			Key:  fmt.Sprintf("avatar_%d", idx),
			Load: func() (any, error) { return loadImage(filepath.Join(assetsDir, fmt.Sprintf("%d.jpg", idx))) },
		})
	}

	// UI buttons
	buttons := []struct{ key, file string }{
		{"btn_place", "place button.png"},
		{"btn_place_hover", "place button - активовано.png"},
		{"btn_code", "code button.png"},
		{"btn_code_hover", "code button - активовано.png"},
		{"btn_send", "send button.png"},
		{"btn_send_hover", "send button-  активовано.png"},
		{"btn_back_hover", "back - активовано.png"},
		{"btn_exit_hover", "exit - активовано.png"},
		{"stamp_success", "success button.png"},
		{"stamp_fail", "fail button.png"},
		{"app_icon", "fingerprinting.png"},
	}

	for _, btn := range buttons {
		b := btn

		tasks = append(tasks, resources.LoadTask{
			Key:  b.key,
			Load: func() (any, error) { return loadImage(filepath.Join(assetsDir, b.file)) },
		})
	}

	// Base fingerprint images (full, for programmatic cutting)
	colorDirs := []string{"green", "blue", "red", "yellow"}

	for _, clr := range colorDirs {
		for variant := 1; variant <= 4; variant++ {
			c, v := clr, variant
			key := fmt.Sprintf("fp_%s_%d", c, v)
			dir := filepath.Join(assetsDir, "Відбитки", "шматочки пазлу", fmt.Sprintf("%s %d", c, v))

			tasks = append(tasks, resources.LoadTask{
				Key: key,
				Load: func() (any, error) {
					return loadCenteredImage(dir)
				},
			})
		}
	}

	// Grey fingerprints
	greyDir := filepath.Join(assetsDir, "Відбитки", "шматочки пазлу", "grey")

	for i := 1; i <= 4; i++ {
		idx := i

		tasks = append(tasks, resources.LoadTask{
			Key: fmt.Sprintf("fp_grey_%d", idx),
			Load: func() (any, error) {
				return loadImage(filepath.Join(greyDir, fmt.Sprintf("G%d centered.png", idx)))
			},
		})
	}

	// Loading animation frames
	for i := 1; i <= 4; i++ {
		idx := i

		tasks = append(tasks, resources.LoadTask{
			Key:  fmt.Sprintf("loading_%d", idx),
			Load: func() (any, error) { return loadImage(filepath.Join(assetsDir, fmt.Sprintf("loading %d.png", idx))) },
		})

		tasks = append(tasks, resources.LoadTask{
			Key:  fmt.Sprintf("loading_%da", idx),
			Load: func() (any, error) { return loadImage(filepath.Join(assetsDir, fmt.Sprintf("loading %dа.png", idx))) },
		})
	}

	// Large images loaded LAST (can take seconds to decode 86MB PNGs)
	// These are downscaled to 1920x1080 max for performance
	largeFiles := map[string]string{
		"bg_static":  "Фон (не анімований).png",
		"bg_bright":  "екран (підвищена яскраввість).png",
		"bg_dim":     "екран (понижена яскравість).png",
		"wallpaper":  "робочий стіл (фон).png",
		"workspace":  "Робоче поле Дактилоскопії.png",
		"grid":       "Робоче поле Дактилоскопії (сітка 0-9).png",
		"app_window": "Вікно вибору відбитка.png",
		"app_full":   "Вікно вибору відбитка (повне).png",
	}

	for key, file := range largeFiles {
		k, f := key, file

		tasks = append(tasks, resources.LoadTask{
			Key: k,
			Load: func() (any, error) {
				return loadAndScale(filepath.Join(assetsDir, f), 1920, 1080)
			},
		})
	}

	slog.Info("starting async load", "tasks", len(tasks))
	rm.LoadAsync(tasks)
}

// loadCenteredImage finds and loads the "*centered*" file in a directory.
func loadCenteredImage(dir string) (*ebiten.Image, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.Contains(strings.ToLower(name), "centered") {
			return loadImage(filepath.Join(dir, name))
		}
	}

	return nil, fmt.Errorf("no centered image found in %s", dir)
}

// loadAndScale loads an image and scales it down to fit within maxW x maxH.
func loadAndScale(path string, maxW, maxH int) (*ebiten.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Only downscale if larger than max
	if w <= maxW && h <= maxH {
		return ebiten.NewImageFromImage(img), nil
	}

	// Calculate scale factor
	scaleX := float64(maxW) / float64(w)
	scaleY := float64(maxH) / float64(h)
	scale := scaleX

	if scaleY < scale {
		scale = scaleY
	}

	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)

	// Use Ebiten to scale (GPU-accelerated)
	src := ebiten.NewImageFromImage(img)
	dst := ebiten.NewImage(newW, newH)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	dst.DrawImage(src, op)

	return dst, nil
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

// FindAssetsDir locates the fingerprint assets directory.
func FindAssetsDir() string {
	candidates := []string{
		"assets/external/fingerprint",
		"../assets/external/fingerprint",
		"../../assets/external/fingerprint",
		filepath.Join(os.Getenv("HOME"), ".config", "pomodoro", "assets", "fingerprint"),
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "assets", "external", "fingerprint"),
			filepath.Join(exeDir, "..", "assets", "external", "fingerprint"),
			filepath.Join(exeDir, "..", "..", "assets", "external", "fingerprint"),
		)
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	return ""
}
