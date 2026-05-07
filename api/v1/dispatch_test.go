// api/dispatch_test.go

package api

import "testing"

func TestLooksLikeOwnerRepo(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"smegg99/qr2g.it", true},
		{"torvalds/linux", true},
		{"a/b", true},
		{"smegg99", false},
		{"smegg99/", false},
		{"/smegg99/qr2g.it", false},
		{"smegg99/qr2g.it/extra", false},
		{"", false},
	}
	for _, c := range cases {
		got := looksLikeOwnerRepo(c.in)
		if got != c.want {
			t.Errorf("looksLikeOwnerRepo(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestReservedSegments(t *testing.T) {
	mustBeReserved := []string{
		"api", "health", "ready", "robots.txt", "favicon.ico",
		".well-known", "assets", "_nuxt", "static", "public",
	}
	for _, s := range mustBeReserved {
		if _, ok := reservedSegments[s]; !ok {
			t.Errorf("expected %q in reservedSegments", s)
		}
	}
	mustNotBeReserved := []string{
		"ABC123", "torvalds", "smegg99", "Z", "1",
	}
	for _, s := range mustNotBeReserved {
		if _, ok := reservedSegments[s]; ok {
			t.Errorf("did not expect %q in reservedSegments", s)
		}
	}
}
