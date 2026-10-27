// core/render/layout.go

package render

import (
	"math"
	"strings"

	"rsc.io/qr"
)

// symbol is an encoded QR code with its quiet zone.
type symbol struct {
	code   *qr.Code
	margin int
	spec   Spec
}

func encode(text string, s Spec) (symbol, error) {
	level := map[ECLevel]qr.Level{ECLow: qr.L, ECMedium: qr.M, ECQuartile: qr.Q, ECHigh: qr.H}[s.ErrorCorrection]
	code, err := qr.Encode(text, level)
	if err != nil {
		return symbol{}, fieldErr("url", CodeInvalid)
	}
	// From version 7 an alignment pattern sits at the centre, where the logo goes.
	if s.Logo && (code.Size-17)/4 >= 7 {
		return symbol{}, fieldErr("logo", CodeUnsupported)
	}
	return symbol{code: code, margin: *s.Margin, spec: s}, nil
}

// hole is the side in modules of the centre square kept free for the logo; zero without one.
func (q symbol) hole() int {
	if !q.spec.Logo {
		return 0
	}
	n := q.code.Size
	side := int(float64(n) * logoShare)
	// Matching the symbol's parity centres the square on whole modules; rounding down keeps it decodable.
	if side%2 != n%2 {
		side--
	}
	return side
}

// symbol adds the modules with their top-left module at (x, y), and the logo when requested.
func (sc *scene) symbol(q symbol, x, y, px int) {
	color := mustHex(q.spec.Foreground)
	hole := q.hole()
	sc.add(modules{x: x, y: y, px: px, code: q.code, hole: hole, color: color})
	if hole == 0 {
		return
	}
	// The fox in the QR colour, centred in the cleared square.
	at := float64(q.code.Size-hole)/2 + logoPad
	side := (float64(hole) - 2*logoPad) * float64(px)
	sc.add(fill{p: hyperfox.fit(float64(x)+at*float64(px), float64(y)+at*float64(px), side), color: color})
}

func (q symbol) total() int { return q.code.Size + 2*q.margin }

// radiusModules is the tile corner radius in modules; 100 percent equals twice the quiet zone.
func (q symbol) radiusModules() float64 {
	if q.spec.Transparent {
		return 0
	}
	return 2 * float64(q.margin) * float64(q.spec.Radius) / MaxRadius
}

// pad is the extra tile border that keeps the whole quiet zone inside the rounded corners.
func (q symbol) pad(px int) int {
	r := q.radiusModules() * float64(px)
	if r == 0 {
		return 0
	}
	return int(math.Ceil(r*(1-1/math.Sqrt2))) + 1
}

// fit returns the largest whole-pixel module size and the outer tile edge that fit target.
func (q symbol) fit(target float64) (px, edge int) {
	px = int(target / (float64(q.total()) + 2*q.radiusModules()*(1-1/math.Sqrt2)))
	for px > 0 && float64(px*q.total()+2*q.pad(px)) > target {
		px--
	}
	return px, px*q.total() + 2*q.pad(px)
}

// tile adds the rounded quiet-zone background (unless transparent) and the modules with the tile's outer corner at (x, y).
func (sc *scene) tile(q symbol, x, y, px int) {
	pad := q.pad(px)
	if !q.spec.Transparent {
		edge := float64(px*q.total() + 2*pad)
		r := q.radiusModules() * float64(px)
		sc.add(box{x: float64(x), y: float64(y), w: edge, h: edge, radius: [4]float64{r, r, r, r}, color: mustHex(q.spec.Background)})
	}
	sc.symbol(q, x+pad+q.margin*px, y+pad+q.margin*px, px)
}

func layoutQR(c Content, q symbol) (*scene, error) {
	n := q.spec.Size
	px, edge := q.fit(float64(n))
	if px < 1 {
		return nil, fieldErr("size", CodeOutOfRange)
	}
	off := (n - edge) / 2
	sc := &scene{w: n, h: n, title: titleOf(c)}
	if !q.spec.Transparent {
		r := q.radiusModules() * float64(px)
		sc.add(box{w: float64(n), h: float64(n), radius: [4]float64{r, r, r, r}, color: mustHex(q.spec.Background)})
	}
	at := off + q.pad(px) + q.margin*px
	sc.symbol(q, at, at, px)
	return sc, nil
}

func layoutPresentation(c Content, q symbol) (*scene, error) {
	s := q.spec
	w, h := Dimensions(s)
	k := float64(w) / DefaultPresentationWidth
	th := themeOf(s.Theme)
	sc := &scene{w: w, h: h, title: titleOf(c)}
	sc.add(box{w: float64(w), h: float64(h), color: th.bg})

	pad := 96 * k
	px, edge := q.fit(0.72 * float64(h))
	if px < 1 {
		return nil, fieldErr("size", CodeOutOfRange)
	}
	tx, ty := w-int(pad)-edge, (h-edge)/2
	sc.tile(q, tx, ty, px)

	colW := float64(tx) - pad - 80*k
	addr := style{fontMono, 56 * k}
	addrAscent, _ := addr.metrics()
	sc.block(addr, addr.wrap(c.Address, colW, 1), pad, float64(h)-pad-addrAscent, colW, addr.size, alignLeft, accentOr(s, th.text))
	limit := float64(h) - pad - addrAscent - 64*k

	owner := style{fontRegular, 40 * k}
	name := style{fontBold, 96 * k}
	caption := style{fontRegular, 44 * k}
	ownerLines := owner.wrap(c.Owner, colW, 1)
	nameMax, captionMax := 2, 3
	for {
		nameLines := name.wrap(c.Name, colW, nameMax)
		captionLines := caption.wrap(s.Caption, colW, captionMax)
		need := float64(len(ownerLines))*(owner.size*1.25+16*k) + float64(len(nameLines))*name.size*1.12
		if len(captionLines) > 0 {
			need += 36*k + float64(len(captionLines))*caption.size*1.3
		}
		if pad+need <= limit || (nameMax == 1 && captionMax == 0 && len(ownerLines) == 0) {
			y := pad
			if len(ownerLines) > 0 {
				y = sc.block(owner, ownerLines, pad, y, colW, owner.size*1.25, alignLeft, th.muted) + 16*k
			}
			y = sc.block(name, nameLines, pad, y, colW, name.size*1.12, alignLeft, th.text)
			sc.block(caption, captionLines, pad, y+36*k, colW, caption.size*1.3, alignLeft, th.text)
			return sc, nil
		}
		switch {
		case captionMax > 0:
			captionMax--
		case nameMax > 1:
			nameMax--
		default:
			ownerLines = nil
		}
	}
}

func layoutPoster(c Content, q symbol) (*scene, error) {
	s := q.spec
	w, h := Dimensions(s)
	k := float64(w) / DefaultPosterWidth
	th := themeOf(s.Theme)
	sc := &scene{w: w, h: h, title: titleOf(c)}
	sc.add(box{w: float64(w), h: float64(h), color: th.bg})

	pad := 80 * k
	inner := float64(w) - 2*pad
	y := pad
	if s.Accent != "" {
		sc.add(box{x: pad, y: y, w: 120 * k, h: 8 * k, color: mustHex(s.Accent)})
		y += 48 * k
	}
	name := style{fontBold, 84 * k}
	body := style{fontRegular, 38 * k}
	text := s.Caption
	if text == "" {
		text = c.Description
	}
	addr := style{fontMono, 64 * k}
	addrAscent, _ := addr.metrics()
	addrTop := float64(h) - pad - addrAscent
	minTile := 0.5 * inner
	nameMax, bodyMax := 2, 4
	for {
		nameLines := name.wrap(c.Name, inner, nameMax)
		bodyLines := body.wrap(text, inner, bodyMax)
		bottom := y + float64(len(nameLines))*name.size*1.12
		if len(bodyLines) > 0 {
			bottom += 20*k + float64(len(bodyLines))*body.size*1.35
		}
		avail := addrTop - 56*k - (bottom + 56*k)
		if avail >= minTile || (nameMax == 1 && bodyMax == 0) {
			yy := sc.block(name, nameLines, pad, y, inner, name.size*1.12, alignLeft, th.text)
			if len(bodyLines) > 0 {
				sc.block(body, bodyLines, pad, yy+20*k, inner, body.size*1.35, alignLeft, th.muted)
			}
			px, edge := q.fit(math.Min(inner, avail))
			if px < 1 {
				return nil, fieldErr("size", CodeOutOfRange)
			}
			sc.tile(q, (w-edge)/2, int(bottom+56*k+(avail-float64(edge))/2), px)
			sc.block(addr, addr.wrap(c.Address, inner, 1), pad, addrTop, inner, addr.size, alignCenter, accentOr(s, th.text))
			return sc, nil
		}
		if bodyMax > 0 {
			bodyMax--
		} else {
			nameMax--
		}
	}
}

func layoutLabel(c Content, q symbol) (*scene, error) {
	s := q.spec
	w, h := Dimensions(s)
	short := math.Min(float64(w), float64(h))
	pad := math.Round(0.06 * short)
	inner := float64(w) - 2*pad
	sc := &scene{w: w, h: h, mmW: s.LabelWidthMM, mmH: s.LabelHeightMM, dpi: s.DPI, title: titleOf(c)}
	sc.add(box{w: float64(w), h: float64(h), color: white})

	addr := style{fontMono, 100}
	addr.size = math.Min(0.1*float64(h), inner*100/math.Max(addr.width(c.Address), 1))
	text := addr.size * 1.2
	caption := style{fontRegular, addr.size * 0.75}
	captionLines := caption.wrap(s.Caption, inner, 1)
	if len(captionLines) > 0 {
		text += caption.size * 1.25
	}
	gap := math.Round(0.04 * short)
	px, edge := q.fit(math.Min(inner, float64(h)-2*pad-gap-text))
	if px < 1 {
		return nil, fieldErr("labelHeightMm", CodeOutOfRange)
	}
	ty := int(pad + (float64(h)-2*pad-gap-text-float64(edge))/2)
	sc.tile(q, (w-edge)/2, ty, px)
	y := sc.block(addr, addr.wrap(c.Address, inner, 1), pad, float64(ty+edge)+gap, inner, addr.size*1.2, alignCenter, black)
	sc.block(caption, captionLines, pad, y, inner, caption.size*1.25, alignCenter, black)
	return sc, nil
}

const badgeHeight = 20

func layoutBadge(c Content, s Spec) *scene {
	host, slug, ok := strings.Cut(c.Address, "/")
	if !ok || slug == "" {
		host, slug = "qr2g.it", c.Address
	}
	st := style{fontRegular, 11}
	lw := math.Ceil(st.width(host) + 12)
	rw := math.Ceil(st.width(slug) + 12)
	sc := &scene{w: int(lw + rw), h: badgeHeight, title: c.Address}
	right := mustHex("#2f6feb")
	if s.Accent != "" {
		right = mustHex(s.Accent)
	}
	sc.add(box{w: lw, h: badgeHeight, radius: [4]float64{3, 0, 0, 3}, color: mustHex("#555555")})
	sc.add(box{x: lw, w: rw, h: badgeHeight, radius: [4]float64{0, 3, 3, 0}, color: right})
	ascent, _ := st.metrics()
	top := 14 - ascent
	sc.block(st, []string{host}, 0, top, lw, badgeHeight, alignCenter, white)
	sc.block(st, []string{slug}, lw, top, rw, badgeHeight, alignCenter, white)
	return sc
}

func accentOr(s Spec, fallback rgb) rgb {
	if s.Accent == "" {
		return fallback
	}
	return mustHex(s.Accent)
}

func titleOf(c Content) string {
	if c.Name != "" {
		return c.Name + " - " + c.Address
	}
	return c.Address
}
