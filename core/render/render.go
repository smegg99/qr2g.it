// core/render/render.go

package render

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"hash/crc32"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"strconv"
	"strings"
)

// Render normalizes the spec, lays out one scene and encodes it in the requested format.
func Render(c Content, s Spec) (Asset, error) {
	s, err := Normalize(s)
	if err != nil {
		return Asset{}, err
	}
	if err := loadFonts(); err != nil {
		return Asset{}, err
	}
	c.URL, c.Address = strings.TrimSpace(c.URL), strings.TrimSpace(c.Address)
	c.Owner, c.Name = strings.TrimSpace(c.Owner), strings.TrimSpace(c.Name)
	c.Description = strings.TrimSpace(c.Description)
	if c.URL == "" {
		return Asset{}, fieldErr("url", CodeInvalid)
	}
	if c.Address == "" {
		return Asset{}, fieldErr("address", CodeInvalid)
	}
	if c.Name == "" && (s.Preset == PresetPresentation || s.Preset == PresetPoster) {
		return Asset{}, fieldErr("name", CodeInvalid)
	}
	sc, err := layout(c, s)
	if err != nil {
		return Asset{}, err
	}
	a := Asset{Width: sc.w, Height: sc.h, WidthMM: sc.mmW, HeightMM: sc.mmH, DPI: sc.dpi}
	switch s.Format {
	case FormatSVG:
		a.ContentType, a.Body = "image/svg+xml", sc.svg()
	case FormatPNG:
		a.ContentType = "image/png"
		a.Body, err = sc.png()
	default:
		a.ContentType = "image/jpeg"
		a.Body, err = sc.jpg()
	}
	return a, err
}

func layout(c Content, s Spec) (*scene, error) {
	if s.Preset == PresetBadge {
		return layoutBadge(c, s), nil
	}
	q, err := encode(c.URL, s)
	if err != nil {
		return nil, err
	}
	switch s.Preset {
	case PresetPresentation:
		return layoutPresentation(c, q)
	case PresetPoster:
		return layoutPoster(c, q)
	case PresetLabel:
		return layoutLabel(c, q)
	default:
		return layoutQR(c, q)
	}
}

func (sc *scene) svg() []byte {
	var b strings.Builder
	w, h := strconv.Itoa(sc.w), strconv.Itoa(sc.h)
	if sc.mmW > 0 {
		w, h = strconv.Itoa(sc.mmW)+"mm", strconv.Itoa(sc.mmH)+"mm"
	}
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="` + w + `" height="` + h +
		`" viewBox="0 0 ` + strconv.Itoa(sc.w) + " " + strconv.Itoa(sc.h) + `"><title>`)
	_ = xml.EscapeText(&b, []byte(sc.title))
	b.WriteString("</title>")
	for _, s := range sc.shapes {
		s.svg(&b)
	}
	b.WriteString("</svg>")
	return []byte(b.String())
}

func (sc *scene) raster() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, sc.w, sc.h))
	for _, s := range sc.shapes {
		s.paint(img)
	}
	return img
}

func (sc *scene) png() ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, sc.raster()); err != nil {
		return nil, fmt.Errorf("render: encode png: %w", err)
	}
	if sc.dpi == 0 {
		return buf.Bytes(), nil
	}
	return withPHYs(buf.Bytes(), sc.dpi), nil
}

func (sc *scene) jpg() ([]byte, error) {
	img := sc.raster()
	flat := image.NewRGBA(img.Bounds())
	draw.Draw(flat, flat.Bounds(), image.NewUniform(white.rgba()), image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), img, image.Point{}, draw.Over)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, flat, &jpeg.Options{Quality: 92}); err != nil {
		return nil, fmt.Errorf("render: encode jpeg: %w", err)
	}
	if sc.dpi == 0 {
		return buf.Bytes(), nil
	}
	return withJFIF(buf.Bytes(), sc.dpi), nil
}

// withPHYs inserts a pHYs chunk after IHDR so print software reads the intended DPI.
func withPHYs(b []byte, dpi int) []byte {
	const ihdrEnd = 8 + 4 + 4 + 13 + 4
	ppm := uint32(float64(dpi)/0.0254 + 0.5)
	chunk := make([]byte, 4+4+9+4)
	binary.BigEndian.PutUint32(chunk[0:], 9)
	copy(chunk[4:], "pHYs")
	binary.BigEndian.PutUint32(chunk[8:], ppm)
	binary.BigEndian.PutUint32(chunk[12:], ppm)
	chunk[16] = 1
	binary.BigEndian.PutUint32(chunk[17:], crc32.ChecksumIEEE(chunk[4:17]))
	out := make([]byte, 0, len(b)+len(chunk))
	out = append(out, b[:ihdrEnd]...)
	out = append(out, chunk...)
	return append(out, b[ihdrEnd:]...)
}

// withJFIF inserts a JFIF APP0 segment with the DPI after SOI; Go's encoder writes none.
func withJFIF(b []byte, dpi int) []byte {
	seg := []byte{0xff, 0xe0, 0, 16, 'J', 'F', 'I', 'F', 0, 1, 1, 1, 0, 0, 0, 0, 0, 0}
	binary.BigEndian.PutUint16(seg[12:], uint16(dpi))
	binary.BigEndian.PutUint16(seg[14:], uint16(dpi))
	out := make([]byte, 0, len(b)+len(seg))
	out = append(out, b[:2]...)
	out = append(out, seg...)
	return append(out, b[2:]...)
}
