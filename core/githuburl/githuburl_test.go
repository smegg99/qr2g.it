package githuburl

import "testing"

func TestParseAccepts(t *testing.T) {
	cases := []struct {
		in   string
		want Repo
	}{
		{"https://github.com/owner/repo", Repo{"owner", "repo"}},
		{"https://github.com/owner/repo/", Repo{"owner", "repo"}},
		{"https://github.com/owner/repo.git", Repo{"owner", "repo"}},
		{"https://GITHUB.COM/owner/repo", Repo{"owner", "repo"}},
		{"HTTPS://github.com/owner/repo", Repo{"owner", "repo"}},
		{"https://github.com/torvalds/linux", Repo{"torvalds", "linux"}},
		{"https://github.com/golang/go.git", Repo{"golang", "go"}},
		{"https://github.com/some-org/some.repo_name-1", Repo{"some-org", "some.repo_name-1"}},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q) err=%v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Parse(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestParseRejects(t *testing.T) {
	bad := []string{
		"",
		"  ",
		"http://github.com/owner/repo",
		"ftp://github.com/owner/repo",
		"https://gitlab.com/owner/repo",
		"https://github.io/owner/repo",
		"https://github.com.evil.com/owner/repo",
		"https://evil.com/github.com/owner/repo",
		"https://user:pass@github.com/owner/repo",
		"https://github.com:8443/owner/repo",
		"https://github.com/owner/repo?x=1",
		"https://github.com/owner/repo#frag",
		"https://github.com/owner/repo/issues/1",
		"https://github.com/owner/repo/tree/main",
		"https://github.com/owner",
		"https://github.com/",
		"https://github.com",
		"https://github.com/-bad/repo",
		"https://github.com/bad-/repo",
		"https://github.com/bad--owner/repo",
		"https://github.com/owner/.",
		"https://github.com/owner/..",
		"https://github.com/api/repo",
		"https://github.com/settings/repo",
		"https://github.com/owner/" + longString(101),
		"https://github.com/" + longString(40) + "/repo",
		"not a url at all",
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) expected error", s)
		}
	}
}

func TestParseOwnerRepo(t *testing.T) {
	if r, err := ParseOwnerRepo("torvalds", "linux"); err != nil || r.FullName() != "torvalds/linux" {
		t.Errorf("got %+v err=%v", r, err)
	}
	if _, err := ParseOwnerRepo("api", "linux"); err == nil {
		t.Errorf("expected reserved owner rejection")
	}
	if r, err := ParseOwnerRepo("torvalds", "linux.git"); err != nil || r.Name != "linux" {
		t.Errorf("got %+v err=%v", r, err)
	}
}

func longString(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = 'a'
	}
	return string(out)
}
