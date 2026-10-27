// common/messages/messages_test.go

package messages

import "testing"

func TestRendersBothLanguagesWithPlaceholders(t *testing.T) {
	en := CliModerateNotSlug(Localizer("en"), CliModerateNotSlugParams{Value: "x"})
	pl := CliModerateNotSlug(Localizer("pl_PL.UTF-8"), CliModerateNotSlugParams{Value: "x"})
	if en != "x is not a slug" || pl != "x nie jest identyfikatorem linku" {
		t.Fatalf("rendered %q and %q", en, pl)
	}
	if got := CliServe(Localizer("de")); got != "Serve the website, API and redirects" {
		t.Fatalf("unsupported language: %q", got)
	}
}

func TestLanguageFromEnv(t *testing.T) {
	for _, c := range []struct{ app, lang, want string }{
		{"", "pl_PL.UTF-8", "pl"},
		{"en", "pl_PL.UTF-8", "en"},
		{"", "C", "en"},
		{"", "fr_FR.UTF-8", "en"},
	} {
		t.Setenv("QR2GIT_LANG", c.app)
		t.Setenv("LC_ALL", "")
		t.Setenv("LC_MESSAGES", "")
		t.Setenv("LANG", c.lang)
		if got := LanguageFromEnv(); got != c.want {
			t.Fatalf("%+v: got %s", c, got)
		}
	}
}
