// common/messages/embed.go

// Package messages embeds the S99Locale backend catalogs and renders command and page text.
package messages

import (
	"embed"
	"encoding/json"
	"os"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// DefaultLanguage is used for a language with no catalog.
const DefaultLanguage = "en"

// Supported lists the catalogs this binary carries.
var Supported = []string{"en", "pl"}

//go:embed locales/*.json
var files embed.FS

// bundle is built once; parsing per message would cost more than rendering.
var bundle = newBundle()

func newBundle() *i18n.Bundle {
	built := i18n.NewBundle(language.English)
	built.RegisterUnmarshalFunc("json", json.Unmarshal)
	for _, code := range Supported {
		if _, err := built.LoadMessageFileFS(files, "locales/"+code+".json"); err != nil {
			panic("messages: load " + code + ": " + err.Error())
		}
	}
	return built
}

// Localizer renders messages in lang, falling back to the default language.
func Localizer(lang string) *i18n.Localizer {
	return i18n.NewLocalizer(bundle, Normalize(lang), DefaultLanguage)
}

// Normalize maps a tag such as pl_PL.UTF-8 to a supported catalog, or the default.
func Normalize(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	for _, code := range Supported {
		if tag == code || strings.HasPrefix(tag, code+"_") || strings.HasPrefix(tag, code+"-") || strings.HasPrefix(tag, code+".") {
			return code
		}
	}
	return DefaultLanguage
}

// LanguageFromEnv is QR2GIT_LANG, then the shell locale, then the default; no private module is linked.
func LanguageFromEnv() string {
	for _, name := range []string{"QR2GIT_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(name); value != "" && value != "C" && value != "POSIX" {
			return Normalize(value)
		}
	}
	return DefaultLanguage
}
