// core/render/spec.go

package render

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Preset selects the asset layout.
type Preset string

// Format selects the output encoding.
type Format string

// Theme selects card colours for presentation and poster presets.
type Theme string

// ECLevel is the QR error correction level.
type ECLevel string

const (
	PresetQR           Preset = "qr"
	PresetPresentation Preset = "presentation"
	PresetPoster       Preset = "poster"
	PresetLabel        Preset = "label"
	PresetBadge        Preset = "badge"

	FormatSVG Format = "svg"
	FormatPNG Format = "png"
	FormatJPG Format = "jpg"

	ThemeLight Theme = "light"
	ThemeDark  Theme = "dark"

	ECLow      ECLevel = "L"
	ECMedium   ECLevel = "M"
	ECQuartile ECLevel = "Q"
	ECHigh     ECLevel = "H"
)

// Limits and defaults shared with the API layer.
const (
	MinMargin     = 4
	MaxMargin     = 16
	DefaultMargin = 4
	MaxRadius     = 100

	MaxCaptionRunes = 140

	MinQRSize     = 128
	MaxQRSize     = 4000
	DefaultQRSize = 1024

	MinPresentationWidth     = 960
	MaxPresentationWidth     = 3840
	DefaultPresentationWidth = 1920

	MinPosterWidth     = 600
	MaxPosterWidth     = 3000
	DefaultPosterWidth = 1200

	MinLabelMM           = 20
	MaxLabelMM           = 200
	DefaultLabelWidthMM  = 40
	DefaultLabelHeightMM = 50
	MinDPI               = 150
	MaxDPI               = 1200
	DefaultDPI           = 300

	MaxPixels = 16_000_000

	MinQRContrast     = 4.5
	MinAccentContrast = 3.0

	DefaultForeground = "#111111"
	DefaultBackground = "#ffffff"
)

// Stable FieldError codes.
const (
	CodeInvalid     = "invalid"
	CodeOutOfRange  = "out_of_range"
	CodeLowContrast = "low_contrast"
	CodeUnsupported = "unsupported"
)

// Spec is the complete design specification of one asset.
type Spec struct {
	Preset          Preset
	Format          Format
	Theme           Theme
	Foreground      string
	Background      string
	Accent          string
	Caption         string
	ErrorCorrection ECLevel
	Margin          int
	Radius          int
	Size            int
	Transparent     bool
	Logo            bool
	LabelWidthMM    int
	LabelHeightMM   int
	DPI             int
}

// Content is the repository data placed on an asset.
type Content struct {
	URL         string
	Address     string
	Owner       string
	Name        string
	Description string
}

// Asset is one rendered file.
type Asset struct {
	Body        []byte
	ContentType string
	Width       int
	Height      int
	WidthMM     int
	HeightMM    int
	DPI         int
}

// FieldError reports the first invalid Spec or Content field.
type FieldError struct {
	Field string
	Code  string
}

func (e *FieldError) Error() string { return "render: " + e.Field + " " + e.Code }

func fieldErr(field, code string) *FieldError { return &FieldError{Field: field, Code: code} }

// Normalize applies defaults, validates every field and clears fields the preset ignores.
func Normalize(s Spec) (Spec, error) {
	if err := normalizeEnums(&s); err != nil {
		return s, err
	}
	if err := normalizeColors(&s); err != nil {
		return s, err
	}
	if err := normalizeCaption(&s); err != nil {
		return s, err
	}
	if s.Margin == 0 {
		s.Margin = DefaultMargin
	}
	if s.Margin < MinMargin || s.Margin > MaxMargin {
		return s, fieldErr("margin", CodeOutOfRange)
	}
	if s.Radius < 0 || s.Radius > MaxRadius {
		return s, fieldErr("radius", CodeOutOfRange)
	}
	if s.Transparent && (s.Preset != PresetQR || s.Format == FormatJPG) {
		return s, fieldErr("transparent", CodeUnsupported)
	}
	if s.Transparent {
		s.Radius = 0
	}
	if s.Preset == PresetBadge && s.Format == FormatJPG {
		return s, fieldErr("format", CodeUnsupported)
	}
	if err := normalizeSize(&s); err != nil {
		return s, err
	}
	// The cleared centre costs codewords that only H reliably recovers.
	if s.Logo && s.ErrorCorrection != ECHigh {
		return s, fieldErr("errorCorrection", CodeUnsupported)
	}
	return s, checkContrast(s)
}

func normalizeEnums(s *Spec) error {
	switch s.Preset {
	case "":
		s.Preset = PresetQR
	case PresetQR, PresetPresentation, PresetPoster, PresetLabel, PresetBadge:
	default:
		return fieldErr("preset", CodeInvalid)
	}
	switch s.Format {
	case "":
		s.Format = FormatPNG
	case FormatSVG, FormatPNG, FormatJPG:
	default:
		return fieldErr("format", CodeInvalid)
	}
	switch s.Theme {
	case "":
		s.Theme = ThemeLight
	case ThemeLight, ThemeDark:
	default:
		return fieldErr("theme", CodeInvalid)
	}
	switch s.ErrorCorrection {
	case "":
		s.ErrorCorrection = ECMedium
		if s.Logo {
			s.ErrorCorrection = ECHigh
		}
	case ECLow, ECMedium, ECQuartile, ECHigh:
	default:
		return fieldErr("errorCorrection", CodeInvalid)
	}
	return nil
}

func normalizeColors(s *Spec) error {
	for _, f := range []struct {
		name string
		v    *string
		def  string
	}{
		{"foreground", &s.Foreground, DefaultForeground},
		{"background", &s.Background, DefaultBackground},
		{"accent", &s.Accent, ""},
	} {
		if *f.v == "" {
			*f.v = f.def
			continue
		}
		c, ok := parseHex(*f.v)
		if !ok {
			return fieldErr(f.name, CodeInvalid)
		}
		*f.v = c.hex()
	}
	return nil
}

func normalizeCaption(s *Spec) error {
	if !utf8.ValidString(s.Caption) {
		return fieldErr("caption", CodeInvalid)
	}
	s.Caption = strings.TrimSpace(s.Caption)
	for _, r := range s.Caption {
		if unicode.IsControl(r) {
			return fieldErr("caption", CodeInvalid)
		}
	}
	if utf8.RuneCountInString(s.Caption) > MaxCaptionRunes {
		return fieldErr("caption", CodeOutOfRange)
	}
	return nil
}

func normalizeSize(s *Spec) error {
	sized := func(min, max, def int) error {
		if s.Size == 0 {
			s.Size = def
		}
		if s.Size < min || s.Size > max {
			return fieldErr("size", CodeOutOfRange)
		}
		return nil
	}
	switch s.Preset {
	case PresetQR:
		s.Theme, s.Accent, s.Caption = ThemeLight, "", ""
		s.LabelWidthMM, s.LabelHeightMM, s.DPI = 0, 0, 0
		return sized(MinQRSize, MaxQRSize, DefaultQRSize)
	case PresetPresentation:
		s.LabelWidthMM, s.LabelHeightMM, s.DPI = 0, 0, 0
		return sized(MinPresentationWidth, MaxPresentationWidth, DefaultPresentationWidth)
	case PresetPoster:
		s.LabelWidthMM, s.LabelHeightMM, s.DPI = 0, 0, 0
		return sized(MinPosterWidth, MaxPosterWidth, DefaultPosterWidth)
	case PresetLabel:
		s.Size, s.Radius, s.Theme, s.Accent = 0, 0, ThemeLight, ""
		s.Foreground, s.Background = "#000000", "#ffffff"
		return normalizeLabel(s)
	default:
		s.Size, s.Caption, s.Theme = 0, "", ThemeLight
		s.Foreground, s.Background = DefaultForeground, DefaultBackground
		s.Margin, s.Radius, s.ErrorCorrection, s.Logo = DefaultMargin, 0, ECMedium, false
		s.LabelWidthMM, s.LabelHeightMM, s.DPI = 0, 0, 0
		return nil
	}
}

func normalizeLabel(s *Spec) error {
	if s.LabelWidthMM == 0 {
		s.LabelWidthMM = DefaultLabelWidthMM
	}
	if s.LabelHeightMM == 0 {
		s.LabelHeightMM = DefaultLabelHeightMM
	}
	if s.DPI == 0 {
		s.DPI = DefaultDPI
	}
	if s.LabelWidthMM < MinLabelMM || s.LabelWidthMM > MaxLabelMM {
		return fieldErr("labelWidthMm", CodeOutOfRange)
	}
	if s.LabelHeightMM < MinLabelMM || s.LabelHeightMM > MaxLabelMM {
		return fieldErr("labelHeightMm", CodeOutOfRange)
	}
	if s.DPI < MinDPI || s.DPI > MaxDPI {
		return fieldErr("dpi", CodeOutOfRange)
	}
	w, h := labelPixels(*s)
	if w*h > MaxPixels {
		return fieldErr("dpi", CodeOutOfRange)
	}
	return nil
}

func checkContrast(s Spec) error {
	fg, _ := parseHex(s.Foreground)
	bg, _ := parseHex(s.Background)
	// Light-on-dark codes are allowed; phone camera apps read them, so only contrast is enforced.
	if s.Preset != PresetBadge && contrast(fg, bg) < MinQRContrast {
		return fieldErr("foreground", CodeLowContrast)
	}
	if s.Accent == "" {
		return nil
	}
	accent, _ := parseHex(s.Accent)
	against := themeOf(s.Theme).bg
	if s.Preset == PresetBadge {
		against = white
	}
	if contrast(accent, against) < MinAccentContrast {
		return fieldErr("accent", CodeLowContrast)
	}
	return nil
}

// Dimensions reports the pixel size a normalized spec renders at, without the badge whose width depends on text.
func Dimensions(s Spec) (width, height int) {
	switch s.Preset {
	case PresetQR:
		return s.Size, s.Size
	case PresetPresentation:
		return s.Size, roundInt(float64(s.Size) * 9 / 16)
	case PresetPoster:
		return s.Size, roundInt(float64(s.Size) * 4 / 3)
	case PresetLabel:
		return labelPixels(s)
	default:
		return 0, badgeHeight
	}
}

func labelPixels(s Spec) (int, int) {
	px := func(mm int) int { return roundInt(float64(mm) * float64(s.DPI) / 25.4) }
	return px(s.LabelWidthMM), px(s.LabelHeightMM)
}
