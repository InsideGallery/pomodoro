package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// GenerateIcon creates a circular timer icon with the given ring color.
func GenerateIcon(size int, ringClr color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 1

	dark := color.RGBA{R: 0x14, G: 0x14, B: 0x19, A: 0xFF}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy

			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= r {
				if dist >= r-2 {
					img.Set(x, y, ringClr)
				} else {
					img.Set(x, y, dark)
				}
			}
		}
	}

	// Draw a small clock hand
	for i := 0; i < int(r*0.6); i++ {
		hx := int(cx)

		hy := int(cy) - i
		if hy >= 0 && hy < size {
			img.Set(hx, hy, ringClr)

			if hx+1 < size {
				img.Set(hx+1, hy, ringClr)
			}
		}
	}

	var buf bytes.Buffer

	_ = png.Encode(&buf, img)

	return buf.Bytes()
}
