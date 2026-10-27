// core/render/render_test.go

package render

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"rsc.io/qr"
)

const testURL = "HTTPS://QR2G.IT/2"

var contents = map[string]Content{
	"plain": {URL: testURL, Address: "qr2g.it/2", Owner: "smegg99", Name: "Goptivum", Description: "Timetable viewer for Optivum schedules."},
	"polish": {
		URL: testURL, Address: "qr2g.it/2", Owner: "zażółć-gęślą-jaźń",
		Name:        "Źdźbło-żółwia-łąkowego-z-bardzo-długą-nazwą-repozytorium-które-się-nie-mieści",
		Description: strings.Repeat("Zażółć gęślą jaźń, łąka pełna źdźbeł i ćwiczeń. ", 12),
	},
	"empty-description": {URL: testURL, Address: "qr2g.it/2", Name: "x"},
}

func decode(t *testing.T, img image.Image) string {
	t.Helper()
	flat := image.NewRGBA(img.Bounds())
	draw.Draw(flat, flat.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), img, img.Bounds().Min, draw.Over)
	bmp, err := gozxing.NewBinaryBitmapFromImage(flat)
	if err != nil {
		t.Fatal(err)
	}
	res, err := qrcode.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return res.GetText()
}

func render(t *testing.T, c Content, s Spec) Asset {
	t.Helper()
	a, err := Render(c, s)
	if err != nil {
		t.Fatalf("render %+v: %v", s, err)
	}
	return a
}

func TestRenderDecodes(t *testing.T) {
	presets := []Spec{
		{Preset: PresetQR, Size: 512},
		{Preset: PresetQR, Size: 512, Radius: 100, Transparent: true},
		{Preset: PresetQR, Size: 512, Logo: true},
		{Preset: PresetQR, Size: 512, Logo: true, Foreground: "#0b3d91", Background: "#fffaf0", Radius: 40},
		{Preset: PresetQR, Size: 512, Logo: true, Transparent: true},
		{Preset: PresetPresentation, Size: 960, Logo: true},
		{Preset: PresetPresentation, Size: 960},
		{Preset: PresetPresentation, Size: 960, Theme: ThemeDark, Accent: "#7aa2ff", Caption: "Scan to see the code"},
		{Preset: PresetPoster, Size: 600, Accent: "#b3261e", Radius: 60},
		{Preset: PresetLabel, Caption: "Bench 3, port A"},
	}
	for name, c := range contents {
		for _, base := range presets {
			for _, f := range []Format{FormatPNG, FormatJPG, FormatSVG} {
				s := base
				s.Format = f
				if s.Transparent && f == FormatJPG {
					continue
				}
				t.Run(name+"/"+string(s.Preset)+"/"+string(f)+"/logo="+strconv.FormatBool(s.Logo), func(t *testing.T) {
					a := render(t, c, s)
					if len(a.Body) == 0 || a.Width <= 0 || a.Height <= 0 {
						t.Fatalf("empty asset %+v", a)
					}
					switch f {
					case FormatPNG:
						img, err := png.Decode(bytes.NewReader(a.Body))
						if err != nil {
							t.Fatal(err)
						}
						checkSize(t, img, a)
						if got := decode(t, img); got != c.URL {
							t.Fatalf("decoded %q", got)
						}
					case FormatJPG:
						img, err := jpeg.Decode(bytes.NewReader(a.Body))
						if err != nil {
							t.Fatal(err)
						}
						checkSize(t, img, a)
						if got := decode(t, img); got != c.URL {
							t.Fatalf("decoded %q", got)
						}
					case FormatSVG:
						checkSVG(t, a, c.URL)
					}
				})
			}
		}
	}
}

func checkSize(t *testing.T, img image.Image, a Asset) {
	t.Helper()
	if b := img.Bounds(); b.Dx() != a.Width || b.Dy() != a.Height {
		t.Fatalf("image %v, asset %dx%d", b, a.Width, a.Height)
	}
}

var svgSize = regexp.MustCompile(`^<svg xmlns="http://www.w3.org/2000/svg" width="([^"]+)" height="([^"]+)" viewBox="0 0 (\d+) (\d+)">`)

func checkSVG(t *testing.T, a Asset, url string) {
	t.Helper()
	m := svgSize.FindStringSubmatch(string(a.Body))
	if m == nil {
		t.Fatalf("svg header: %.120s", a.Body)
	}
	wantW, wantH := itoa(a.Width), itoa(a.Height)
	if a.WidthMM > 0 {
		wantW, wantH = itoa(a.WidthMM)+"mm", itoa(a.HeightMM)+"mm"
	}
	if m[1] != wantW || m[2] != wantH || m[3] != itoa(a.Width) || m[4] != itoa(a.Height) {
		t.Fatalf("svg size %v, asset %+v", m[1:], a)
	}
	for _, banned := range []string{"<text", "<style", "<script", "href", "font-family", "<image"} {
		if strings.Contains(string(a.Body), banned) {
			t.Fatalf("svg contains %q", banned)
		}
	}
	tool, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Log("rsvg-convert not installed; SVG raster decode skipped")
		return
	}
	dir := t.TempDir()
	in, out := filepath.Join(dir, "a.svg"), filepath.Join(dir, "a.png")
	if err := os.WriteFile(in, a.Body, 0o600); err != nil {
		t.Fatal(err)
	}
	if b, err := exec.Command(tool, "-w", "1000", "-b", "white", "-o", out, in).CombinedOutput(); err != nil {
		t.Fatalf("rsvg-convert: %v %s", err, b)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if got := decode(t, img); got != url {
		t.Fatalf("svg decoded %q", got)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestBadge(t *testing.T) {
	for _, f := range []Format{FormatSVG, FormatPNG} {
		a := render(t, contents["plain"], Spec{Preset: PresetBadge, Format: f})
		if a.Height != badgeHeight || a.Width < 40 {
			t.Fatalf("badge %dx%d", a.Width, a.Height)
		}
	}
	if _, err := Render(contents["plain"], Spec{Preset: PresetBadge, Format: FormatJPG}); !isField(err, "format", CodeUnsupported) {
		t.Fatalf("badge jpg: %v", err)
	}
}

func TestQuietZoneAndRadius(t *testing.T) {
	c := contents["plain"]
	near := func(a, b color.RGBA) bool {
		d := func(x, y uint8) bool { return int(x)-int(y) <= 2 && int(y)-int(x) <= 2 }
		return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
	}
	for _, radius := range []int{0, 50, 100} {
		s := mustNormalize(t, Spec{Preset: PresetQR, Format: FormatPNG, Size: 500, Radius: radius})
		img, err := png.Decode(bytes.NewReader(render(t, c, s).Body))
		if err != nil {
			t.Fatal(err)
		}
		q, _ := encode(c.URL, s)
		px, edge := q.fit(500)
		start := (500-edge)/2 + q.pad(px) + q.margin*px
		end := start + q.code.Size*px
		fg, bg := mustHex(s.Foreground).rgba(), mustHex(s.Background).rgba()
		at := func(x, y int) color.RGBA { return color.RGBAModel.Convert(img.At(x, y)).(color.RGBA) }
		for y := start - 4*px; y < end+4*px; y++ {
			for x := start - 4*px; x < end+4*px; x++ {
				inside := x >= start && x < end && y >= start && y < end
				if !inside && !near(at(x, y), bg) {
					t.Fatalf("radius %d: quiet zone pixel (%d,%d) = %v", radius, x, y, at(x, y))
				}
			}
		}
		for _, p := range [][2]int{{start, start}, {end - 1, start}, {start, end - 1}} {
			if at(p[0], p[1]) != fg {
				t.Fatalf("radius %d: finder corner (%d,%d) = %v", radius, p[0], p[1], at(p[0], p[1]))
			}
		}
		if got := decode(t, img); got != c.URL {
			t.Fatalf("radius %d decoded %q", radius, got)
		}
	}
}

func TestLabelPhysicalSize(t *testing.T) {
	c := contents["plain"]
	a := render(t, c, Spec{Preset: PresetLabel, Format: FormatPNG, LabelWidthMM: 30, LabelHeightMM: 40, DPI: 600})
	if a.WidthMM != 30 || a.HeightMM != 40 || a.DPI != 600 || a.Width != 709 || a.Height != 945 {
		t.Fatalf("label asset %+v", a)
	}
	i := bytes.Index(a.Body, []byte("pHYs"))
	if i < 0 {
		t.Fatal("pHYs chunk missing")
	}
	ppm := binary.BigEndian.Uint32(a.Body[i+4:])
	if ppm != 23622 || a.Body[i+12] != 1 {
		t.Fatalf("pHYs ppm %d unit %d", ppm, a.Body[i+12])
	}
	if _, err := png.Decode(bytes.NewReader(a.Body)); err != nil {
		t.Fatalf("png with pHYs invalid: %v", err)
	}
	svg := render(t, c, Spec{Preset: PresetLabel, Format: FormatSVG})
	if !bytes.Contains(svg.Body, []byte(`width="40mm" height="50mm"`)) {
		t.Fatalf("label svg header %.120s", svg.Body)
	}
}

func TestSameScene(t *testing.T) {
	for _, s := range []Spec{{Preset: PresetPresentation, Size: 1280}, {Preset: PresetPoster, Size: 900}, {Preset: PresetQR, Size: 700}} {
		s.Format = FormatSVG
		svg := render(t, contents["polish"], s)
		s.Format = FormatPNG
		pngA := render(t, contents["polish"], s)
		w, h := Dimensions(mustNormalize(t, s))
		if svg.Width != pngA.Width || svg.Height != pngA.Height || pngA.Width != w || pngA.Height != h {
			t.Fatalf("%s: svg %dx%d png %dx%d dims %dx%d", s.Preset, svg.Width, svg.Height, pngA.Width, pngA.Height, w, h)
		}
	}
}

func TestDeterministic(t *testing.T) {
	for _, f := range []Format{FormatSVG, FormatPNG} {
		s := Spec{Preset: PresetPresentation, Format: f, Size: 960, Caption: "Ąę"}
		if !bytes.Equal(render(t, contents["polish"], s).Body, render(t, contents["polish"], s).Body) {
			t.Fatalf("%s output differs between runs", f)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		name  string
		spec  Spec
		field string
		code  string
	}{
		{"bad colour", Spec{Foreground: "#12345"}, "foreground", CodeInvalid},
		{"named colour", Spec{Background: "white"}, "background", CodeInvalid},
		{"low contrast", Spec{Foreground: "#777777", Background: "#888888"}, "foreground", CodeLowContrast},
		{"accent contrast", Spec{Preset: PresetPresentation, Accent: "#fafafa"}, "accent", CodeLowContrast},
		{"margin negative", Spec{Margin: ptr(-1)}, "margin", CodeOutOfRange},
		{"margin high", Spec{Margin: ptr(17)}, "margin", CodeOutOfRange},
		{"radius", Spec{Radius: 101}, "radius", CodeOutOfRange},
		{"caption long", Spec{Caption: strings.Repeat("ż", MaxCaptionRunes+1)}, "caption", CodeOutOfRange},
		{"caption control", Spec{Caption: "a\nb"}, "caption", CodeInvalid},
		{"transparent jpg", Spec{Transparent: true, Format: FormatJPG}, "transparent", CodeUnsupported},
		{"transparent card", Spec{Transparent: true, Preset: PresetPoster}, "transparent", CodeUnsupported},
		{"qr size", Spec{Size: 4096}, "size", CodeOutOfRange},
		{"presentation size", Spec{Preset: PresetPresentation, Size: 500}, "size", CodeOutOfRange},
		{"poster size", Spec{Preset: PresetPoster, Size: 3001}, "size", CodeOutOfRange},
		{"label mm", Spec{Preset: PresetLabel, LabelWidthMM: 10}, "labelWidthMm", CodeOutOfRange},
		{"label pixels", Spec{Preset: PresetLabel, LabelWidthMM: 200, LabelHeightMM: 200, DPI: 1200}, "dpi", CodeOutOfRange},
		{"preset", Spec{Preset: "slide"}, "preset", CodeInvalid},
		{"format", Spec{Format: "gif"}, "format", CodeInvalid},
		{"theme", Spec{Theme: "blue"}, "theme", CodeInvalid},
		{"ec", Spec{ErrorCorrection: "X"}, "errorCorrection", CodeInvalid},
		{"logo low ec", Spec{Logo: true, ErrorCorrection: ECLow}, "errorCorrection", CodeUnsupported},
		{"logo medium ec", Spec{Logo: true, ErrorCorrection: ECMedium}, "errorCorrection", CodeUnsupported},
		{"logo quartile ec", Spec{Logo: true, ErrorCorrection: ECQuartile}, "errorCorrection", CodeUnsupported},
	}
	for _, tc := range cases {
		if _, err := Normalize(tc.spec); !isField(err, tc.field, tc.code) {
			t.Errorf("%s: got %v, want %s %s", tc.name, err, tc.field, tc.code)
		}
	}
	if _, err := Normalize(Spec{Foreground: "#AABBCC", Background: "#FFFFFF"}); err == nil {
		t.Fatal("light foreground accepted")
	}
	got, err := Normalize(Spec{Foreground: "#0A0B0C", Caption: " x "})
	want := Spec{Preset: PresetQR, Format: FormatPNG, Theme: ThemeLight, Foreground: "#0a0b0c", Background: DefaultBackground,
		ErrorCorrection: ECMedium, Size: DefaultQRSize}
	if err != nil || got.Margin == nil || *got.Margin != DefaultMargin {
		t.Fatalf("default margin: %+v %v", got, err)
	}
	got.Margin = nil
	if got != want {
		t.Fatalf("defaults: %+v", got)
	}
	if bare := mustNormalize(t, Spec{Margin: ptr(0)}); *bare.Margin != 0 {
		t.Fatalf("an explicit zero quiet zone became %d", *bare.Margin)
	}
	if got := mustNormalize(t, Spec{Logo: true}); got.ErrorCorrection != ECHigh {
		t.Fatalf("logo default ec: %+v", got)
	}
	if got := mustNormalize(t, Spec{Preset: PresetBadge, Logo: true}); got.Logo || got.ErrorCorrection != ECMedium {
		t.Fatalf("badge logo: %+v", got)
	}
	label := mustNormalize(t, Spec{Preset: PresetLabel, Foreground: "#123456", Accent: "#ff0000", Radius: 50})
	if label.Foreground != "#000000" || label.Background != "#ffffff" || label.Accent != "" || label.Radius != 0 ||
		label.LabelWidthMM != DefaultLabelWidthMM || label.DPI != DefaultDPI {
		t.Fatalf("label defaults: %+v", label)
	}
	if _, err := Render(Content{URL: testURL, Address: "qr2g.it/2"}, Spec{Preset: PresetPoster}); !isField(err, "name", CodeInvalid) {
		t.Fatalf("missing name: %v", err)
	}
}

func mustNormalize(t *testing.T, s Spec) Spec {
	t.Helper()
	n, err := Normalize(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func isField(err error, field, code string) bool {
	var fe *FieldError
	return errors.As(err, &fe) && fe.Field == field && fe.Code == code
}

// Real addresses are lowercase byte-mode text, a larger symbol than testURL.
func TestLogoDecodesRealAddresses(t *testing.T) {
	for _, url := range []string{
		"https://qr2g.it/2", "https://qr2g.it/ZZZZZZ", "https://qr2g.it/1ABCDEFGHJKL",
		"https://qr2git.example.org/1ABCDEFGHJKL", "https://links.example.org/qr2git/1ABCDEFGHJKL",
	} {
		for _, size := range []int{256, 600, 2048} {
			a := render(t, Content{URL: url, Address: strings.TrimPrefix(url, "https://")}, Spec{Size: size, Logo: true})
			img, err := png.Decode(bytes.NewReader(a.Body))
			if err != nil {
				t.Fatal(err)
			}
			if got := decode(t, img); got != url {
				t.Fatalf("%s at %d px: decoded %q", url, size, got)
			}
		}
	}
}

// Swapped colours give a light-on-dark code; inverting the pixels back must decode it.
func TestInvertedDecodes(t *testing.T) {
	a := render(t, Content{URL: "https://qr2g.it/2", Address: "qr2g.it/2"}, Spec{Size: 600, Logo: true, Foreground: "#ffffff", Background: "#111111"})
	img, err := png.Decode(bytes.NewReader(a.Body))
	if err != nil {
		t.Fatal(err)
	}
	flipped := image.NewRGBA(img.Bounds())
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			r, g, b, _ := img.At(x, y).RGBA()
			flipped.Set(x, y, color.RGBA{uint8(255 - r>>8), uint8(255 - g>>8), uint8(255 - b>>8), 255})
		}
	}
	if got := decode(t, flipped); got != "https://qr2g.it/2" {
		t.Fatalf("decoded %q", got)
	}
}

// The fox is drawn in the QR colour over a cleared centre, and only when asked for.
func TestLogoClearsCentre(t *testing.T) {
	spec := Spec{Format: FormatSVG, Foreground: "#0b3d91", Background: "#fffaf0"}
	without := string(render(t, contents["plain"], spec).Body)
	spec.Logo = true
	with := string(render(t, contents["plain"], spec).Body)
	if strings.Count(with, `<path fill="#0b3d91"`) != 1 || strings.Contains(without, `<path fill="#0b3d91"`) {
		t.Fatal("fox path missing or drawn without the logo")
	}
}

var svgPathData = regexp.MustCompile(`<path\b[^>]*?\sd="([^"]*)"`)

func TestLogoMatchesAsset(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "assets", "qr2g.it_hyperfox.svg"))
	if err != nil {
		t.Fatal(err)
	}
	found := svgPathData.FindAllStringSubmatch(string(raw), -1)
	if len(found) != len(hyperfoxPaths) {
		t.Fatalf("asset has %d paths, renderer %d", len(found), len(hyperfoxPaths))
	}
	for i, m := range found {
		if m[1] != hyperfoxPaths[i] {
			t.Fatalf("path %d differs from the asset", i)
		}
		var p path
		if err := parseSVGPath(&p, m[1]); err != nil {
			t.Fatal(err)
		}
	}
	if w, h := hyperfox.maxX-hyperfox.minX, hyperfox.maxY-hyperfox.minY; w <= 0 || h <= 0 {
		t.Fatalf("logo bounds %vx%v", w, h)
	}
}

// Reading the half blocks back gives the symbol's modules, in both terminal modes.
func TestTerminalMatchesSymbol(t *testing.T) {
	code, err := qr.Encode(testURL, qr.M)
	if err != nil {
		t.Fatal(err)
	}
	for _, inverted := range []bool{false, true} {
		out, err := Terminal(testURL, inverted)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		n := code.Size + 2*terminalMargin
		if len(lines) != (n+1)/2 {
			t.Fatalf("%d lines for %d modules", len(lines), n)
		}
		for row, line := range lines {
			cells := []rune(line)
			if len(cells) != n {
				t.Fatalf("line %d has %d cells, want %d", row, len(cells), n)
			}
			for x, c := range cells {
				top := c == '█' || c == '▀'
				bottom := c == '█' || c == '▄'
				for i, ink := range []bool{top, bottom} {
					y := 2*row + i - terminalMargin
					mx := x - terminalMargin
					if y >= code.Size || y < 0 || mx < 0 || mx >= code.Size {
						continue
					}
					if dark := code.Black(mx, y); (dark == inverted) != ink {
						t.Fatalf("inverted=%v module %d,%d: ink %v, dark %v", inverted, mx, y, ink, dark)
					}
				}
			}
		}
	}
}

// A code without a quiet zone still decodes once it sits on a light surface, which then acts as one.
func TestZeroQuietZoneDecodesOnLightSurface(t *testing.T) {
	for _, logo := range []bool{false, true} {
		a := render(t, Content{URL: "https://qr2g.it/Q", Address: "qr2g.it/Q"}, Spec{Size: 600, Margin: ptr(0), Logo: logo})
		img, err := png.Decode(bytes.NewReader(a.Body))
		if err != nil {
			t.Fatal(err)
		}
		page := image.NewRGBA(image.Rect(0, 0, 700, 700))
		draw.Draw(page, page.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
		draw.Draw(page, image.Rect(50, 50, 650, 650), img, image.Point{}, draw.Src)
		if got := decode(t, page); got != "https://qr2g.it/Q" {
			t.Fatalf("logo=%v: decoded %q", logo, got)
		}
	}
}
