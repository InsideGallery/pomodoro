package components

import "github.com/hajimehoshi/ebiten/v2"

// Sprite holds an image reference for rendering.
type Sprite struct {
	Image   *ebiten.Image
	Visible bool
}
