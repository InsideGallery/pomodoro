package ui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawRoundedRect draws a filled rounded rectangle.
func DrawRoundedRect(dst *ebiten.Image, x, y, w, h, radius float32, clr color.Color) {
	if w <= 0 || h <= 0 {
		return
	}

	var p vector.Path
	roundedRectPath(&p, x, y, w, h, radius)

	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)

	r, g, b, a := colorToFloat32(clr)
	for i := range vs {
		vs[i].ColorR = r
		vs[i].ColorG = g
		vs[i].ColorB = b
		vs[i].ColorA = a
	}

	dst.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	})
}

// DrawRoundedRectStroke draws a rounded rectangle outline.
func DrawRoundedRectStroke(dst *ebiten.Image, x, y, w, h, radius, strokeWidth float32, clr color.Color) {
	if w <= 0 || h <= 0 {
		return
	}

	var p vector.Path
	roundedRectPath(&p, x, y, w, h, radius)

	so := &vector.StrokeOptions{
		Width:    strokeWidth,
		LineJoin: vector.LineJoinRound,
	}
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, so)

	r, g, b, a := colorToFloat32(clr)
	for i := range vs {
		vs[i].ColorR = r
		vs[i].ColorG = g
		vs[i].ColorB = b
		vs[i].ColorA = a
	}

	dst.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	})
}

// DrawArc draws a filled arc (for progress rings).
func DrawArc(dst *ebiten.Image, cx, cy, outerR, innerR float32, startAngle, endAngle float64, clr color.Color) {
	if endAngle <= startAngle {
		return
	}

	segments := int(math.Ceil((endAngle - startAngle) / (math.Pi / 32)))
	if segments < 2 {
		segments = 2
	}

	step := (endAngle - startAngle) / float64(segments)
	r, g, b, a := colorToFloat32(clr)

	vertices := make([]ebiten.Vertex, 0, (segments+1)*2)
	indices := make([]uint16, 0, segments*6)

	for i := 0; i <= segments; i++ {
		angle := startAngle + float64(i)*step
		cos := float32(math.Cos(angle))
		sin := float32(math.Sin(angle))

		vertices = append(vertices,
			ebiten.Vertex{
				DstX: cx + outerR*cos, DstY: cy + outerR*sin,
				SrcX: 0.5, SrcY: 0.5,
				ColorR: r, ColorG: g, ColorB: b, ColorA: a,
			},
			ebiten.Vertex{
				DstX: cx + innerR*cos, DstY: cy + innerR*sin,
				SrcX: 0.5, SrcY: 0.5,
				ColorR: r, ColorG: g, ColorB: b, ColorA: a,
			},
		)
	}

	for i := 0; i < segments; i++ {
		base := uint16(i * 2)
		indices = append(indices,
			base, base+1, base+2,
			base+1, base+2, base+3,
		)
	}

	dst.DrawTriangles(vertices, indices, whitePixel(), &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	})
}

// DrawGradientArc draws an arc with color gradient from startClr to endClr.
func DrawGradientArc(dst *ebiten.Image, cx, cy, outerR, innerR float32,
	startAngle, endAngle float64, startClr, endClr color.Color,
) {
	if endAngle <= startAngle {
		return
	}

	segments := int(math.Ceil((endAngle - startAngle) / (math.Pi / 32)))
	if segments < 2 {
		segments = 2
	}

	step := (endAngle - startAngle) / float64(segments)
	sr, sg, sb, sa := colorToFloat32(startClr)
	er, eg, eb, ea := colorToFloat32(endClr)

	vertices := make([]ebiten.Vertex, 0, (segments+1)*2)
	indices := make([]uint16, 0, segments*6)

	for i := 0; i <= segments; i++ {
		t := float32(i) / float32(segments)
		angle := startAngle + float64(i)*step
		cos := float32(math.Cos(angle))
		sin := float32(math.Sin(angle))

		cr := sr + (er-sr)*t
		cg := sg + (eg-sg)*t
		cb := sb + (eb-sb)*t
		ca := sa + (ea-sa)*t

		vertices = append(vertices,
			ebiten.Vertex{
				DstX: cx + outerR*cos, DstY: cy + outerR*sin,
				SrcX: 0.5, SrcY: 0.5,
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			},
			ebiten.Vertex{
				DstX: cx + innerR*cos, DstY: cy + innerR*sin,
				SrcX: 0.5, SrcY: 0.5,
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			},
		)
	}

	for i := 0; i < segments; i++ {
		base := uint16(i * 2)
		indices = append(indices,
			base, base+1, base+2,
			base+1, base+2, base+3,
		)
	}

	dst.DrawTriangles(vertices, indices, whitePixel(), &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	})
}

// DrawCircle draws a filled circle.
func DrawCircle(dst *ebiten.Image, cx, cy, radius float32, clr color.Color) {
	DrawArc(dst, cx, cy, radius, 0, 0, 2*math.Pi, clr)
}

// DrawSettingsIcon draws a three-line settings/hamburger icon.
func DrawSettingsIcon(dst *ebiten.Image, cx, cy, size float32, clr color.Color) {
	lineW := size * 0.7
	lineH := size * 0.09
	gap := size * 0.25
	x := cx - lineW/2

	for i := -1; i <= 1; i++ {
		ly := cy + float32(i)*gap - lineH/2
		DrawRoundedRect(dst, x, ly, lineW, lineH, lineH/2, clr)
		// Small circle on each line (equalizer style)
		dotR := lineH * 1.5
		dotX := cx + float32(i)*size*0.15
		DrawCircle(dst, dotX, ly+lineH/2, dotR, clr)
	}
}

// DrawMinimizeIcon draws a horizontal line (minimize).
func DrawMinimizeIcon(dst *ebiten.Image, cx, cy, size float32, clr color.Color) {
	half := size * 0.4
	sw := size * 0.12

	var p vector.Path
	p.MoveTo(cx-half, cy)
	p.LineTo(cx+half, cy)

	so := &vector.StrokeOptions{Width: sw, LineCap: vector.LineCapRound}
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, so)

	r, g, b, a := colorToFloat32(clr)
	for i := range vs {
		vs[i].ColorR = r
		vs[i].ColorG = g
		vs[i].ColorB = b
		vs[i].ColorA = a
	}

	dst.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

// DrawExpandIcon draws a maximize/expand icon (square with outward arrow).
func DrawExpandIcon(dst *ebiten.Image, cx, cy, size float32, clr color.Color) {
	half := size * 0.4
	sw := size * 0.12

	// Square outline
	var p vector.Path
	p.MoveTo(cx-half, cy-half)
	p.LineTo(cx+half, cy-half)
	p.LineTo(cx+half, cy+half)
	p.LineTo(cx-half, cy+half)
	p.Close()

	so := &vector.StrokeOptions{Width: sw, LineJoin: vector.LineJoinRound}
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, so)

	r, g, b, a := colorToFloat32(clr)
	for i := range vs {
		vs[i].ColorR = r
		vs[i].ColorG = g
		vs[i].ColorB = b
		vs[i].ColorA = a
	}

	dst.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{AntiAlias: true})

	// Diagonal arrow (bottom-left to top-right)
	var p2 vector.Path

	ar := half * 0.5
	p2.MoveTo(cx-ar, cy+ar)
	p2.LineTo(cx+ar, cy-ar)
	// Arrowhead
	p2.MoveTo(cx+ar, cy-ar)
	p2.LineTo(cx+ar-half*0.35, cy-ar)
	p2.MoveTo(cx+ar, cy-ar)
	p2.LineTo(cx+ar, cy-ar+half*0.35)

	so2 := &vector.StrokeOptions{Width: sw, LineCap: vector.LineCapRound}

	vs2, is2 := p2.AppendVerticesAndIndicesForStroke(nil, nil, so2)
	for i := range vs2 {
		vs2[i].ColorR = r
		vs2[i].ColorG = g
		vs2[i].ColorB = b
		vs2[i].ColorA = a
	}

	dst.DrawTriangles(vs2, is2, whitePixel(), &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

// DrawPlayIcon draws a play triangle (pointing right).
func DrawPlayIcon(dst *ebiten.Image, cx, cy, size float32, clr color.Color) {
	half := size * 0.4

	var p vector.Path

	p.MoveTo(cx-half*0.6, cy-half)
	p.LineTo(cx+half, cy)
	p.LineTo(cx-half*0.6, cy+half)
	p.Close()

	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)

	r, g, b, a := colorToFloat32(clr)
	for i := range vs {
		vs[i].ColorR = r
		vs[i].ColorG = g
		vs[i].ColorB = b
		vs[i].ColorA = a
	}

	dst.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

// DrawPauseIcon draws two vertical bars.
func DrawPauseIcon(dst *ebiten.Image, cx, cy, size float32, clr color.Color) {
	barW := size * 0.2
	barH := size * 0.7
	gap := size * 0.15

	DrawRoundedRect(dst, cx-gap-barW, cy-barH/2, barW, barH, barW/4, clr)
	DrawRoundedRect(dst, cx+gap, cy-barH/2, barW, barH, barW/4, clr)
}

// DrawBackIcon draws a left-arrow back icon.
func DrawBackIcon(dst *ebiten.Image, cx, cy, size float32, clr color.Color) {
	half := size * 0.45
	sw := size * 0.14

	var p vector.Path
	// Arrow head: < shape
	p.MoveTo(cx+half*0.3, cy-half)
	p.LineTo(cx-half*0.3, cy)
	p.LineTo(cx+half*0.3, cy+half)

	so := &vector.StrokeOptions{
		Width:    sw,
		LineJoin: vector.LineJoinRound,
		LineCap:  vector.LineCapRound,
	}
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, so)

	r, g, b, a := colorToFloat32(clr)
	for i := range vs {
		vs[i].ColorR = r
		vs[i].ColorG = g
		vs[i].ColorB = b
		vs[i].ColorA = a
	}

	dst.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	})
}

// DrawCloseIcon draws an X close icon.
func DrawCloseIcon(dst *ebiten.Image, cx, cy, size float32, clr color.Color) {
	half := size / 2
	sw := size * 0.12

	var p vector.Path
	p.MoveTo(cx-half, cy-half)
	p.LineTo(cx+half, cy+half)
	p.MoveTo(cx+half, cy-half)
	p.LineTo(cx-half, cy+half)

	so := &vector.StrokeOptions{
		Width:    sw,
		LineJoin: vector.LineJoinRound,
		LineCap:  vector.LineCapRound,
	}
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, so)

	r, g, b, a := colorToFloat32(clr)
	for i := range vs {
		vs[i].ColorR = r
		vs[i].ColorG = g
		vs[i].ColorB = b
		vs[i].ColorA = a
	}

	dst.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	})
}

// drawSpeakerBase draws the speaker body and cone on the left half of the box.
func drawSpeakerBase(dst *ebiten.Image, cx, cy, size float32, clr color.Color) {
	left := cx - size/2
	bodyW := size * 0.18
	bodyH := size * 0.3

	DrawFilledPolygon(dst, [][2]float32{
		{left, cy - bodyH/2},
		{left + bodyW, cy - bodyH/2},
		{left + bodyW, cy + bodyH/2},
		{left, cy + bodyH/2},
	}, clr)

	coneW := size * 0.26
	coneH := size * 0.7
	DrawFilledPolygon(dst, [][2]float32{
		{left + bodyW, cy - bodyH/2},
		{left + bodyW + coneW, cy - coneH/2},
		{left + bodyW + coneW, cy + coneH/2},
		{left + bodyW, cy + bodyH/2},
	}, clr)
}

// strokePath strokes the path with the given width and colour.
func strokePath(dst *ebiten.Image, p *vector.Path, width float32, clr color.Color) {
	so := &vector.StrokeOptions{
		Width:    width,
		LineJoin: vector.LineJoinRound,
		LineCap:  vector.LineCapRound,
	}
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, so)

	r, g, b, a := colorToFloat32(clr)
	for i := range vs {
		vs[i].ColorR = r
		vs[i].ColorG = g
		vs[i].ColorB = b
		vs[i].ColorA = a
	}

	dst.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

// DrawSpeakerIcon draws a speaker with sound waves.
func DrawSpeakerIcon(dst *ebiten.Image, cx, cy, size float32, clr color.Color) {
	drawSpeakerBase(dst, cx, cy, size, clr)

	x := cx + size*0.12

	var p vector.Path
	p.MoveTo(x, cy-size*0.14)
	p.LineTo(x+size*0.1, cy)
	p.LineTo(x, cy+size*0.14)
	p.MoveTo(x+size*0.12, cy-size*0.3)
	p.LineTo(x+size*0.27, cy)
	p.LineTo(x+size*0.12, cy+size*0.3)

	strokePath(dst, &p, size*0.1, clr)
}

// DrawMutedIcon draws a speaker crossed out with an X.
func DrawMutedIcon(dst *ebiten.Image, cx, cy, size float32, clr color.Color) {
	drawSpeakerBase(dst, cx, cy, size, clr)

	x := cx + size*0.3
	d := size * 0.16

	var p vector.Path
	p.MoveTo(x-d, cy-d)
	p.LineTo(x+d, cy+d)
	p.MoveTo(x+d, cy-d)
	p.LineTo(x-d, cy+d)

	strokePath(dst, &p, size*0.1, clr)
}

// DrawFilledPolygon draws a filled polygon from a list of (x,y) pairs.
func DrawFilledPolygon(dst *ebiten.Image, points [][2]float32, clr color.Color) {
	if len(points) < 3 {
		return
	}

	var p vector.Path
	p.MoveTo(points[0][0], points[0][1])

	for _, pt := range points[1:] {
		p.LineTo(pt[0], pt[1])
	}

	p.Close()

	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)

	r, g, b, a := colorToFloat32(clr)
	for i := range vs {
		vs[i].ColorR = r
		vs[i].ColorG = g
		vs[i].ColorB = b
		vs[i].ColorA = a
	}

	dst.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	})
}

func roundedRectPath(p *vector.Path, x, y, w, h, r float32) {
	if r > w/2 {
		r = w / 2
	}

	if r > h/2 {
		r = h / 2
	}

	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.ArcTo(x+w, y, x+w, y+r, r)
	p.LineTo(x+w, y+h-r)
	p.ArcTo(x+w, y+h, x+w-r, y+h, r)
	p.LineTo(x+r, y+h)
	p.ArcTo(x, y+h, x, y+h-r, r)
	p.LineTo(x, y+r)
	p.ArcTo(x, y, x+r, y, r)
	p.Close()
}

var whiteImg *ebiten.Image

func whitePixel() *ebiten.Image {
	if whiteImg == nil {
		whiteImg = ebiten.NewImage(3, 3)
		whiteImg.Fill(color.White)
	}

	return whiteImg
}

func colorToFloat32(clr color.Color) (r, g, b, a float32) {
	cr, cg, cb, ca := clr.RGBA()
	if ca == 0 {
		return 0, 0, 0, 0
	}

	a = float32(ca) / 0xFFFF
	r = float32(cr) / 0xFFFF * a
	g = float32(cg) / 0xFFFF * a
	b = float32(cb) / 0xFFFF * a

	return
}
