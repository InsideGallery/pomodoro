package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func main() {
	size := 256
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 4

	purple := color.RGBA{R: 0x6C, G: 0x5C, B: 0xE7, A: 0xFF}
	purpleLight := color.RGBA{R: 0xA2, G: 0x9B, B: 0xFE, A: 0xFF}
	dark := color.RGBA{R: 0x0B, G: 0x0B, B: 0x0F, A: 0xFF}
	darkInner := color.RGBA{R: 0x14, G: 0x14, B: 0x19, A: 0xFF}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist <= r {
				if dist >= r-8 {
					// Outer ring — gradient
					t := (dist - (r - 8)) / 8
					angle := math.Atan2(dy, dx)
					at := (angle + math.Pi) / (2 * math.Pi)
					c := lerpColor(purple, purpleLight, at)
					c = lerpColor(darkInner, c, t)
					img.Set(x, y, c)
				} else {
					img.Set(x, y, dark)
				}
			}
		}
	}

	// Clock hands
	drawLine(img, cx, cy, cx, cy-r*0.55, 6, purple)
	drawLine(img, cx, cy, cx+r*0.35, cy, 4, purpleLight)

	// Center dot
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx

			dy := float64(y) - cy
			if math.Sqrt(dx*dx+dy*dy) <= 6 {
				img.Set(x, y, purple)
			}
		}
	}

	path := "packaging/pomodoro.png"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 float64, width float64, clr color.RGBA) {
	steps := int(math.Max(math.Abs(x2-x1), math.Abs(y2-y1))) * 2
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		px := x1 + (x2-x1)*t
		py := y1 + (y2-y1)*t

		for dy := -width / 2; dy <= width/2; dy++ {
			for dx := -width / 2; dx <= width/2; dx++ {
				if dx*dx+dy*dy <= (width/2)*(width/2) {
					ix, iy := int(px+dx), int(py+dy)
					if ix >= 0 && ix < img.Bounds().Dx() && iy >= 0 && iy < img.Bounds().Dy() {
						img.Set(ix, iy, clr)
					}
				}
			}
		}
	}
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 0xFF,
	}
}
