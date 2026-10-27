// core/policy/markers.go

// Package policy reads repository owners' opt-out markers and keeps them current.
package policy

import (
	"regexp"
	"strings"
)

// Marker files in the default branch root. Their content is ignored.
const (
	FileNoIndex     = ".qr2git-noindex"
	FileNoAnalytics = ".qr2git-noanalytics"
)

// Markers are the restrictions an owner set; either form of a restriction wins.
type Markers struct {
	NoIndex     bool
	NoAnalytics bool
}

// Or combines two marker sources.
func (m Markers) Or(o Markers) Markers {
	return Markers{NoIndex: m.NoIndex || o.NoIndex, NoAnalytics: m.NoAnalytics || o.NoAnalytics}
}

// FromRoot reads the marker files from root file names.
func FromRoot(names []string) Markers {
	var m Markers
	for _, name := range names {
		switch name {
		case FileNoIndex:
			m.NoIndex = true
		case FileNoAnalytics:
			m.NoAnalytics = true
		}
	}
	return m
}

var (
	directive = regexp.MustCompile(`(?i)^ {0,3}<!--\s*qr2git:\s*(noindex|noanalytics)\s*-->\s*$`)
	fence     = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
)

// FromReadme finds standalone directive comments outside fenced and indented code.
func FromReadme(text string) Markers {
	var m Markers
	var open string
	for line := range strings.SplitSeq(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if f := fence.FindStringSubmatch(line); f != nil {
			switch {
			case open == "":
				open = f[1]
			case f[1][0] == open[0] && len(f[1]) >= len(open) && strings.TrimSpace(line[strings.Index(line, f[1])+len(f[1]):]) == "":
				open = ""
			}
			continue
		}
		if open != "" {
			continue
		}
		d := directive.FindStringSubmatch(line)
		if d == nil {
			continue
		}
		switch strings.ToLower(d[1]) {
		case "noindex":
			m.NoIndex = true
		case "noanalytics":
			m.NoAnalytics = true
		}
	}
	return m
}
