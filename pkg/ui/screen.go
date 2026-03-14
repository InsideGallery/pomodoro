package ui

import "github.com/hajimehoshi/ebiten/v2"

// Screen is the interface that all app screens implement.
type Screen interface {
	Init(w, h int)
	Update()
	Draw(dst *ebiten.Image)
	Resize(w, h int)
}

// Compile-time checks that existing screens satisfy the interface.
var (
	_ Screen = (*TimerScreen)(nil)
	_ Screen = (*SettingsScreen)(nil)
)
