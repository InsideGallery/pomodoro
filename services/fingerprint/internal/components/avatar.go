package components

import "github.com/hajimehoshi/ebiten/v2"

// Avatar holds a character portrait image.
type Avatar struct {
	Filename string
	Image    *ebiten.Image // lazily loaded, nil until first render
}
