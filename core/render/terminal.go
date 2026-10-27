// core/render/terminal.go

package render

import (
	"strings"

	"rsc.io/qr"
)

// terminalMargin is the quiet zone of a terminal code in modules; terminals are read from a screen.
const terminalMargin = 2

// Terminal draws text as a QR code with Unicode half blocks, two module rows per line.
// By default light modules are drawn, which reads correctly on a dark terminal; inverted
// draws dark modules instead, for light terminals.
func Terminal(text string, inverted bool) (string, error) {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return "", fieldErr("url", CodeInvalid)
	}
	n := code.Size + 2*terminalMargin
	// drawn reports whether a module gets ink; outside the symbol is the light quiet zone.
	drawn := func(x, y int) bool {
		x, y = x-terminalMargin, y-terminalMargin
		dark := x >= 0 && y >= 0 && x < code.Size && y < code.Size && code.Black(x, y)
		return dark == inverted
	}
	var b strings.Builder
	for y := 0; y < n; y += 2 {
		for x := range n {
			top, bottom := drawn(x, y), y+1 < n && drawn(x, y+1)
			if y+1 >= n {
				bottom = !inverted
			}
			switch {
			case top && bottom:
				b.WriteString("█")
			case top:
				b.WriteString("▀")
			case bottom:
				b.WriteString("▄")
			default:
				b.WriteByte(' ')
			}
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}
