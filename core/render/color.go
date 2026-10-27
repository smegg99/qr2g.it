// core/render/color.go

package render

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
)

type rgb struct{ r, g, b uint8 }

var (
	white = rgb{0xff, 0xff, 0xff}
	black = rgb{0, 0, 0}
)

type palette struct{ bg, text, muted rgb }

func themeOf(t Theme) palette {
	if t == ThemeDark {
		return palette{bg: rgb{0x12, 0x12, 0x12}, text: rgb{0xf2, 0xf2, 0xf2}, muted: rgb{0xa8, 0xab, 0xb0}}
	}
	return palette{bg: white, text: rgb{0x11, 0x11, 0x11}, muted: rgb{0x5f, 0x63, 0x68}}
}

func parseHex(s string) (rgb, bool) {
	if len(s) != 7 || s[0] != '#' {
		return rgb{}, false
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return rgb{}, false
	}
	return rgb{uint8(v >> 16), uint8(v >> 8), uint8(v)}, true
}

func mustHex(s string) rgb {
	c, _ := parseHex(s)
	return c
}

func (c rgb) hex() string { return fmt.Sprintf("#%02x%02x%02x", c.r, c.g, c.b) }

func (c rgb) rgba() color.RGBA { return color.RGBA{c.r, c.g, c.b, 0xff} }

// luminance is the WCAG relative luminance.
func (c rgb) luminance() float64 {
	ch := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.r) + 0.7152*ch(c.g) + 0.0722*ch(c.b)
}

func contrast(a, b rgb) float64 {
	la, lb := a.luminance(), b.luminance()
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}
