// core/policy/markers_test.go

package policy

import "testing"

func TestFromReadme(t *testing.T) {
	cases := []struct {
		name string
		text string
		want Markers
	}{
		{"absent", "# Project\n\nNothing here.\n", Markers{}},
		{"noindex", "# Project\n<!-- qr2git: noindex -->\n", Markers{NoIndex: true}},
		{"noanalytics spacing and case", "<!--QR2GIT:NoAnalytics-->  \r\n", Markers{NoAnalytics: true}},
		{"both", "<!-- qr2git: noindex -->\ntext\n<!-- qr2git: noanalytics -->", Markers{NoIndex: true, NoAnalytics: true}},
		{"backtick fence example", "```html\n<!-- qr2git: noindex -->\n```\n", Markers{}},
		{"tilde fence example", "~~~\n<!-- qr2git: noanalytics -->\n~~~\n", Markers{}},
		{"longer closing fence", "````\n<!-- qr2git: noindex -->\n`````\n<!-- qr2git: noanalytics -->\n", Markers{NoAnalytics: true}},
		{"mismatched fence stays open", "```\n~~~\n<!-- qr2git: noindex -->\n", Markers{}},
		{"fence with info string does not close", "```\n``` js\n<!-- qr2git: noindex -->\n", Markers{}},
		{"indented code", "text\n\n    <!-- qr2git: noindex -->\n", Markers{}},
		{"inline code", "Add `<!-- qr2git: noindex -->` to opt out.\n", Markers{}},
		{"not standalone", "Hello <!-- qr2git: noindex -->\n", Markers{}},
		{"unknown directive", "<!-- qr2git: hide -->\n", Markers{}},
		{"marker after closed fence", "```\ncode\n```\n<!-- qr2git: noindex -->\n", Markers{NoIndex: true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FromReadme(c.text); got != c.want {
				t.Fatalf("FromReadme() = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestFromRootAndOr(t *testing.T) {
	root := FromRoot([]string{"README.md", FileNoAnalytics, ".qr2git-NOINDEX"})
	if root != (Markers{NoAnalytics: true}) {
		t.Fatalf("FromRoot() = %+v", root)
	}
	if got := root.Or(Markers{NoIndex: true}); got != (Markers{NoIndex: true, NoAnalytics: true}) {
		t.Fatalf("Or() = %+v", got)
	}
}
