package ui

import (
	"bytes"
	"image/color"
	"math"

	"github.com/InsideGallery/pomodoro/assets"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// UIScale is the device scale factor for HiDPI rendering.
// Set from Layout via ebiten.Monitor().DeviceScaleFactor().
var UIScale float64 = 1.0

// S scales a logical value by the current device scale factor.
func S(v float32) float32 {
	return float32(math.Round(float64(v) * UIScale))
}

// Sf scales a float64 value.
func Sf(v float64) float64 {
	return math.Round(v * UIScale)
}

// ThemeID identifies a color theme.
type ThemeID string

const (
	ThemeDark  ThemeID = "dark"
	ThemeLight ThemeID = "light"
)

// Colors — all mutable, switched by SetTheme().
var (
	ColorBgPrimary     color.RGBA
	ColorBgSecondary   color.RGBA
	ColorBgTertiary    color.RGBA
	ColorBorder        color.RGBA
	ColorTextPrimary   color.RGBA
	ColorTextSecond    color.RGBA
	ColorAccentFocus   color.RGBA
	ColorAccentBreak   color.RGBA
	ColorAccentDanger  color.RGBA
	ColorAccentSuccess color.RGBA
	ColorGradFocusEnd  color.RGBA
	ColorGradBreakEnd  color.RGBA
	ColorSliderTrack   color.RGBA
	ColorToggleOff     color.RGBA
	ColorWindowBg      color.RGBA
	ColorCardBg        color.RGBA
	ColorCardBorder    color.RGBA
)

func init() {
	SetTheme(ThemeDark)
	initFonts()
}

// SetTheme switches all color variables to the given theme.
func SetTheme(id ThemeID) {
	switch id {
	case ThemeLight:
		ColorBgPrimary = color.RGBA{R: 0xF2, G: 0xF3, B: 0xF7, A: 0xFF}
		ColorBgSecondary = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
		ColorBgTertiary = color.RGBA{R: 0xE8, G: 0xE9, B: 0xEF, A: 0xFF}
		ColorBorder = color.RGBA{R: 0xD8, G: 0xDA, B: 0xE2, A: 0xFF}
		ColorTextPrimary = color.RGBA{R: 0x1A, G: 0x1B, B: 0x25, A: 0xFF}
		ColorTextSecond = color.RGBA{R: 0x6B, G: 0x6D, B: 0x7B, A: 0xFF}
		ColorAccentFocus = color.RGBA{R: 0x5B, G: 0x4C, B: 0xD6, A: 0xFF}
		ColorAccentBreak = color.RGBA{R: 0x00, G: 0xB0, B: 0xAB, A: 0xFF}
		ColorAccentDanger = color.RGBA{R: 0xE8, G: 0x4D, B: 0x4D, A: 0xFF}
		ColorAccentSuccess = color.RGBA{R: 0x00, G: 0xA0, B: 0x7E, A: 0xFF}
		ColorGradFocusEnd = color.RGBA{R: 0x8B, G: 0x7E, B: 0xF0, A: 0xFF}
		ColorGradBreakEnd = color.RGBA{R: 0x5C, G: 0xDB, B: 0xDB, A: 0xFF}
		ColorSliderTrack = color.RGBA{R: 0xDE, G: 0xDF, B: 0xE5, A: 0xFF}
		ColorToggleOff = color.RGBA{R: 0xC8, G: 0xCA, B: 0xD2, A: 0xFF}
		// Alpha set by ApplyTransparency after SetTheme
		ColorWindowBg = color.RGBA{R: 0xF2, G: 0xF3, B: 0xF7, A: 0xFF}
		ColorCardBg = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
		ColorCardBorder = color.RGBA{R: 0xD8, G: 0xDA, B: 0xE2, A: 0xFF}
	default: // dark
		ColorBgPrimary = color.RGBA{R: 0x0B, G: 0x0B, B: 0x0F, A: 0xFF}
		ColorBgSecondary = color.RGBA{R: 0x14, G: 0x14, B: 0x19, A: 0xFF}
		ColorBgTertiary = color.RGBA{R: 0x1C, G: 0x1C, B: 0x24, A: 0xFF}
		ColorBorder = color.RGBA{R: 0x2A, G: 0x2A, B: 0x35, A: 0xFF}
		ColorTextPrimary = color.RGBA{R: 0xF0, G: 0xF0, B: 0xF5, A: 0xFF}
		ColorTextSecond = color.RGBA{R: 0x8B, G: 0x8B, B: 0x9E, A: 0xFF}
		ColorAccentFocus = color.RGBA{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF}
		ColorAccentBreak = color.RGBA{R: 0x00, G: 0xCE, B: 0xC9, A: 0xFF}
		ColorAccentDanger = color.RGBA{R: 0xFF, G: 0x6B, B: 0x6B, A: 0xFF}
		ColorAccentSuccess = color.RGBA{R: 0x00, G: 0xB8, B: 0x94, A: 0xFF}
		ColorGradFocusEnd = color.RGBA{R: 0xA2, G: 0x9B, B: 0xFE, A: 0xFF}
		ColorGradBreakEnd = color.RGBA{R: 0x81, G: 0xEC, B: 0xEC, A: 0xFF}
		ColorSliderTrack = color.RGBA{R: 0x22, G: 0x22, B: 0x2E, A: 0xFF}
		ColorToggleOff = color.RGBA{R: 0x3A, G: 0x3A, B: 0x4A, A: 0xFF}
		// Alpha set by ApplyTransparency after SetTheme
		ColorWindowBg = color.RGBA{R: 0x10, G: 0x10, B: 0x18, A: 0xFF}
		ColorCardBg = color.RGBA{R: 0x18, G: 0x18, B: 0x22, A: 0xFF}
		ColorCardBorder = color.RGBA{R: 0x2A, G: 0x2A, B: 0x35, A: 0xFF}
	}
}

// Font sources
var (
	FontSourceRegular *textv2.GoTextFaceSource
	FontSourceBold    *textv2.GoTextFaceSource
)

func initFonts() {
	var err error
	FontSourceRegular, err = textv2.NewGoTextFaceSource(bytes.NewReader(assets.FontRegular))
	if err != nil {
		panic("failed to parse regular font: " + err.Error())
	}
	FontSourceBold, err = textv2.NewGoTextFaceSource(bytes.NewReader(assets.FontBold))
	if err != nil {
		panic("failed to parse bold font: " + err.Error())
	}
}

// Face creates a text face at the given logical size, scaled by UIScale.
func Face(bold bool, size float64) *textv2.GoTextFace {
	src := FontSourceRegular
	if bold {
		src = FontSourceBold
	}
	return &textv2.GoTextFace{
		Source: src,
		Size:   size * UIScale,
	}
}

// ApplyTransparency adjusts window and card alpha. t is 0.0 (opaque) to 1.0 (fully transparent).
func ApplyTransparency(t float64) {
	if t < 0.10 {
		t = 0.10
	}
	if t > 0.90 {
		t = 0.90
	}
	opaque := 1.0 - t
	winAlpha := uint8(opaque * 230)   // max ~90% even at 0% transparency setting
	cardAlpha := uint8(opaque * 240)  // slightly more opaque than window
	borderAlpha := uint8(opaque * 180)
	ColorWindowBg.A = winAlpha
	ColorCardBg.A = cardAlpha
	ColorCardBorder.A = borderAlpha
}

// Layout constants (logical, use S() to scale)
const (
	RadiusCard   = 12
	RadiusButton = 8
)
