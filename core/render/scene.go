// core/render/scene.go

package render

import (
	"image"
	"image/draw"
	"math"
	"strconv"
	"strings"

	"golang.org/x/image/vector"
	"rsc.io/qr"
)

// scene is the single drawing model behind every SVG and raster output.
type scene struct {
	w, h   int
	mmW    int
	mmH    int
	dpi    int
	title  string
	shapes []shape
}

type shape interface {
	svg(b *strings.Builder)
	paint(dst *image.RGBA)
}

func (sc *scene) add(s shape) { sc.shapes = append(sc.shapes, s) }

type segKind uint8

const (
	segMove segKind = iota
	segLine
	segQuad
	segCube
	segClose
)

type seg struct {
	k segKind
	p [3][2]float64
}

var segPoints = [...]int{segMove: 1, segLine: 1, segQuad: 2, segCube: 3, segClose: 0}

type path struct{ segs []seg }

func (p *path) moveTo(x, y float64) {
	p.segs = append(p.segs, seg{k: segMove, p: [3][2]float64{{x, y}}})
}
func (p *path) lineTo(x, y float64) {
	p.segs = append(p.segs, seg{k: segLine, p: [3][2]float64{{x, y}}})
}
func (p *path) close() { p.segs = append(p.segs, seg{k: segClose}) }

func (p *path) quadTo(x1, y1, x, y float64) {
	p.segs = append(p.segs, seg{k: segQuad, p: [3][2]float64{{x1, y1}, {x, y}}})
}

func (p *path) cubeTo(x1, y1, x2, y2, x, y float64) {
	p.segs = append(p.segs, seg{k: segCube, p: [3][2]float64{{x1, y1}, {x2, y2}, {x, y}}})
}

// roundRect adds a rectangle whose corners (tl, tr, br, bl) use circular arcs.
func (p *path) roundRect(x, y, w, h float64, r [4]float64) {
	const k = 0.5522847498
	tl, tr, br, bl := r[0], r[1], r[2], r[3]
	p.moveTo(x+tl, y)
	p.lineTo(x+w-tr, y)
	if tr > 0 {
		p.cubeTo(x+w-tr+k*tr, y, x+w, y+tr-k*tr, x+w, y+tr)
	}
	p.lineTo(x+w, y+h-br)
	if br > 0 {
		p.cubeTo(x+w, y+h-br+k*br, x+w-br+k*br, y+h, x+w-br, y+h)
	}
	p.lineTo(x+bl, y+h)
	if bl > 0 {
		p.cubeTo(x+bl-k*bl, y+h, x, y+h-bl+k*bl, x, y+h-bl)
	}
	p.lineTo(x, y+tl)
	if tl > 0 {
		p.cubeTo(x, y+tl-k*tl, x+tl-k*tl, y, x+tl, y)
	}
	p.close()
}

// fill paints a path with a solid colour using the non-zero rule.
type fill struct {
	p     path
	color rgb
}

func (f fill) svg(b *strings.Builder) {
	if len(f.p.segs) == 0 {
		return
	}
	b.WriteString(`<path fill="`)
	b.WriteString(f.color.hex())
	b.WriteString(`" d="`)
	letters := [...]string{segMove: "M", segLine: "L", segQuad: "Q", segCube: "C", segClose: "Z"}
	for _, s := range f.p.segs {
		b.WriteString(letters[s.k])
		if s.k == segClose {
			continue
		}
		for i := range segPoints[s.k] {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(num(s.p[i][0]))
			b.WriteByte(' ')
			b.WriteString(num(s.p[i][1]))
		}
	}
	b.WriteString(`"/>`)
}

func (f fill) paint(dst *image.RGBA) {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, s := range f.p.segs {
		for i := range segPoints[s.k] {
			minX, maxX = math.Min(minX, s.p[i][0]), math.Max(maxX, s.p[i][0])
			minY, maxY = math.Min(minY, s.p[i][1]), math.Max(maxY, s.p[i][1])
		}
	}
	r := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX)), int(math.Ceil(maxY))).Intersect(dst.Bounds())
	if r.Empty() {
		return
	}
	ox, oy := float64(r.Min.X), float64(r.Min.Y)
	z := vector.NewRasterizer(r.Dx(), r.Dy())
	open := false
	for _, s := range f.p.segs {
		pt := func(i int) (float32, float32) { return float32(s.p[i][0] - ox), float32(s.p[i][1] - oy) }
		switch s.k {
		case segMove:
			if open {
				z.ClosePath()
			}
			z.MoveTo(pt(0))
			open = true
		case segLine:
			z.LineTo(pt(0))
		case segQuad:
			x1, y1 := pt(0)
			x, y := pt(1)
			z.QuadTo(x1, y1, x, y)
		case segCube:
			x1, y1 := pt(0)
			x2, y2 := pt(1)
			x, y := pt(2)
			z.CubeTo(x1, y1, x2, y2, x, y)
		case segClose:
			z.ClosePath()
			open = false
		}
	}
	if open {
		z.ClosePath()
	}
	z.Draw(dst, r, image.NewUniform(f.color.rgba()), image.Point{})
}

// box is an axis-aligned rectangle with optional rounded corners.
type box struct {
	x, y, w, h float64
	radius     [4]float64
	color      rgb
}

func (bx box) rounded() bool { return bx.radius != [4]float64{} }

func (bx box) svg(b *strings.Builder) {
	if bx.rounded() && (bx.radius[0] != bx.radius[1] || bx.radius[0] != bx.radius[2] || bx.radius[0] != bx.radius[3]) {
		bx.asFill().svg(b)
		return
	}
	b.WriteString(`<rect x="` + num(bx.x) + `" y="` + num(bx.y) + `" width="` + num(bx.w) + `" height="` + num(bx.h) + `"`)
	if bx.rounded() {
		b.WriteString(` rx="` + num(bx.radius[0]) + `"`)
	}
	b.WriteString(` fill="` + bx.color.hex() + `"/>`)
}

func (bx box) paint(dst *image.RGBA) {
	integral := bx.x == math.Trunc(bx.x) && bx.y == math.Trunc(bx.y) && bx.w == math.Trunc(bx.w) && bx.h == math.Trunc(bx.h)
	if bx.rounded() || !integral {
		bx.asFill().paint(dst)
		return
	}
	r := image.Rect(int(bx.x), int(bx.y), int(bx.x+bx.w), int(bx.y+bx.h))
	draw.Draw(dst, r, image.NewUniform(bx.color.rgba()), image.Point{}, draw.Src)
}

func (bx box) asFill() fill {
	var p path
	p.roundRect(bx.x, bx.y, bx.w, bx.h, bx.radius)
	return fill{p: p, color: bx.color}
}

// modules draws QR modules as integer-pixel squares starting at the symbol origin.
type modules struct {
	x, y, px int
	code     *qr.Code
	hole     int
	color    rgb
}

func (m modules) runs(fn func(x, y, n int)) {
	lo := (m.code.Size - m.hole) / 2
	hi := lo + m.hole
	cleared := func(x, y int) bool { return x >= lo && x < hi && y >= lo && y < hi }
	for y := range m.code.Size {
		start := -1
		for x := 0; x <= m.code.Size; x++ {
			dark := x < m.code.Size && m.code.Black(x, y) && !cleared(x, y)
			if dark && start < 0 {
				start = x
			}
			if !dark && start >= 0 {
				fn(start, y, x-start)
				start = -1
			}
		}
	}
}

func (m modules) svg(b *strings.Builder) {
	b.WriteString(`<path shape-rendering="crispEdges" fill="` + m.color.hex() + `" transform="translate(` +
		strconv.Itoa(m.x) + " " + strconv.Itoa(m.y) + ") scale(" + strconv.Itoa(m.px) + `)" d="`)
	m.runs(func(x, y, n int) {
		b.WriteString("M" + strconv.Itoa(x) + " " + strconv.Itoa(y) + "h" + strconv.Itoa(n) + "v1h-" + strconv.Itoa(n) + "z")
	})
	b.WriteString(`"/>`)
}

func (m modules) paint(dst *image.RGBA) {
	src := image.NewUniform(m.color.rgba())
	m.runs(func(x, y, n int) {
		r := image.Rect(m.x+x*m.px, m.y+y*m.px, m.x+(x+n)*m.px, m.y+(y+1)*m.px)
		draw.Draw(dst, r, src, image.Point{}, draw.Src)
	})
}

func num(v float64) string {
	v = math.Round(v*100) / 100
	if v == 0 {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func roundInt(v float64) int { return int(math.Round(v)) }
