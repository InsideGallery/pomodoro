package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// GenerateIcon creates a simple circular timer icon as PNG bytes.
func GenerateIcon(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 1

	purple := color.RGBA{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF}
	dark := color.RGBA{R: 0x14, G: 0x14, B: 0x19, A: 0xFF}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= r {
				if dist >= r-2 {
					img.Set(x, y, purple)
				} else {
					img.Set(x, y, dark)
				}
			}
		}
	}

	// Draw a small clock hand
	for i := 0; i < int(r*0.6); i++ {
		hx := int(cx) + 0
		hy := int(cy) - i
		if hy >= 0 && hy < size {
			img.Set(hx, hy, purple)
			if hx+1 < size {
				img.Set(hx+1, hy, purple)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
