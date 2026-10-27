// core/render/text.go

package render

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var (
	fontsOnce                       sync.Once
	fontRegular, fontBold, fontMono *sfnt.Font
	fontsErr                        error
)

func loadFonts() error {
	fontsOnce.Do(func() {
		parse := func(name string, b []byte) *sfnt.Font {
			f, err := sfnt.Parse(b)
			if err != nil && fontsErr == nil {
				fontsErr = fmt.Errorf("render: parse %s font: %w", name, err)
			}
			return f
		}
		fontRegular = parse("regular", goregular.TTF)
		fontBold = parse("bold", gobold.TTF)
		fontMono = parse("mono", gomonobold.TTF)
	})
	return fontsErr
}

// style is one font at one pixel size.
type style struct {
	font *sfnt.Font
	size float64
}

func (st style) ppem() fixed.Int26_6 { return fixed.Int26_6(math.Round(st.size * 64)) }

func (st style) glyph(b *sfnt.Buffer, r rune) sfnt.GlyphIndex {
	if i, err := st.font.GlyphIndex(b, r); err == nil && i != 0 {
		return i
	}
	i, _ := st.font.GlyphIndex(b, '?')
	return i
}

// ascent and descent are positive distances from the baseline in pixels.
func (st style) metrics() (ascent, descent float64) {
	var b sfnt.Buffer
	m, err := st.font.Metrics(&b, st.ppem(), font.HintingNone)
	if err != nil {
		return st.size * 0.93, st.size * 0.22
	}
	return float64(m.Ascent) / 64, float64(m.Descent) / 64
}

// width measures text in pixels including kerning.
func (st style) width(text string) float64 {
	var b sfnt.Buffer
	var total fixed.Int26_6
	prev := sfnt.GlyphIndex(0)
	for _, r := range text {
		g := st.glyph(&b, r)
		if prev != 0 {
			if k, err := st.font.Kern(&b, prev, g, st.ppem(), font.HintingNone); err == nil {
				total += k
			}
		}
		adv, err := st.font.GlyphAdvance(&b, g, st.ppem(), font.HintingNone)
		if err == nil {
			total += adv
		}
		prev = g
	}
	return float64(total) / 64
}

// outline appends the glyph outlines of text with its baseline origin at (x, y).
func (st style) outline(p *path, text string, x, y float64) {
	var b sfnt.Buffer
	pen := fixed.Int26_6(0)
	prev := sfnt.GlyphIndex(0)
	at := func(v fixed.Point26_6) (float64, float64) {
		return x + float64(pen+v.X)/64, y + float64(v.Y)/64
	}
	for _, r := range text {
		g := st.glyph(&b, r)
		if prev != 0 {
			if k, err := st.font.Kern(&b, prev, g, st.ppem(), font.HintingNone); err == nil {
				pen += k
			}
		}
		segs, err := st.font.LoadGlyph(&b, g, st.ppem(), nil)
		if err == nil {
			started := false
			for _, s := range segs {
				switch s.Op {
				case sfnt.SegmentOpMoveTo:
					if started {
						p.close()
					}
					p.moveTo(at(s.Args[0]))
					started = true
				case sfnt.SegmentOpLineTo:
					p.lineTo(at(s.Args[0]))
				case sfnt.SegmentOpQuadTo:
					x1, y1 := at(s.Args[0])
					x2, y2 := at(s.Args[1])
					p.quadTo(x1, y1, x2, y2)
				case sfnt.SegmentOpCubeTo:
					x1, y1 := at(s.Args[0])
					x2, y2 := at(s.Args[1])
					x3, y3 := at(s.Args[2])
					p.cubeTo(x1, y1, x2, y2, x3, y3)
				}
			}
			if started {
				p.close()
			}
		}
		if adv, err := st.font.GlyphAdvance(&b, g, st.ppem(), font.HintingNone); err == nil {
			pen += adv
		}
		prev = g
	}
}

func (st style) ellipsis() string {
	var b sfnt.Buffer
	if i, err := st.font.GlyphIndex(&b, '\u2026'); err == nil && i != 0 {
		return "\u2026"
	}
	return "..."
}

// wrap breaks text into at most maxLines lines no wider than maxW, ellipsizing the last line when text remains.
func (st style) wrap(text string, maxW float64, maxLines int) []string {
	words := strings.Fields(text)
	if len(words) == 0 || maxLines <= 0 || maxW <= 0 {
		return nil
	}
	var lines []string
	cur := ""
	truncated := false
	for i, w := range words {
		if len(lines) >= maxLines {
			truncated = true
			break
		}
		cand := w
		if cur != "" {
			cand = cur + " " + w
		}
		if st.width(cand) <= maxW {
			cur = cand
			continue
		}
		if cur != "" {
			lines = append(lines, cur)
			cur = ""
		}
		for st.width(w) > maxW && len(lines) < maxLines {
			head, tail := st.fit(w, maxW)
			lines = append(lines, head)
			w = tail
		}
		if len(lines) >= maxLines {
			truncated = w != "" || i < len(words)-1
			break
		}
		cur = w
	}
	if cur != "" {
		if len(lines) < maxLines {
			lines = append(lines, cur)
		} else {
			truncated = true
		}
	}
	if truncated && len(lines) > 0 {
		lines[len(lines)-1] = st.ellipsize(lines[len(lines)-1], maxW)
	}
	return lines
}

// fit splits word into the longest rune prefix fitting maxW (at least one rune) and the rest.
func (st style) fit(word string, maxW float64) (string, string) {
	runes := []rune(word)
	n := 1
	for n < len(runes) && st.width(string(runes[:n+1])) <= maxW {
		n++
	}
	return string(runes[:n]), string(runes[n:])
}

func (st style) ellipsize(line string, maxW float64) string {
	mark := st.ellipsis()
	runes := []rune(strings.TrimRight(line, " "))
	for len(runes) > 0 && st.width(string(runes)+mark) > maxW {
		runes = []rune(strings.TrimRight(string(runes[:len(runes)-1]), " "))
	}
	return string(runes) + mark
}

type align int

const (
	alignLeft align = iota
	alignCenter
)

// block places lines from the top edge y inside [x, x+w] and returns the bottom edge.
func (sc *scene) block(st style, lines []string, x, y, w, lineHeight float64, a align, c rgb) float64 {
	if len(lines) == 0 {
		return y
	}
	ascent, _ := st.metrics()
	var p path
	for i, line := range lines {
		lx := x
		if a == alignCenter {
			lx = x + (w-st.width(line))/2
		}
		st.outline(&p, line, lx, y+ascent+float64(i)*lineHeight)
	}
	sc.add(fill{p: p, color: c})
	return y + float64(len(lines))*lineHeight
}
