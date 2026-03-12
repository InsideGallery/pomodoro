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
func DrawGradientArc(dst *ebiten.Image, cx, cy, outerR, innerR float32, startAngle, endAngle float64, startClr, endClr color.Color) {
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
