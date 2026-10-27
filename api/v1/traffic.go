// api/v1/traffic.go

package api

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// automatedAgent matches identifiable crawlers, link previews, monitors and HTTP libraries.
// Filtering is imperfect: unidentified automation is still counted.
var automatedAgent = regexp.MustCompile(`(?i)(bot\b|bot/|crawl|spider|slurp|preview|fetch|scan|monitor|uptime|pingdom|` +
	`headless|lighthouse|facebookexternalhit|facebookcatalog|embedly|quora link|whatsapp|telegram|` +
	`skypeuripreview|vkshare|w3c_validator|curl/|wget/|httpie/|python-|go-http-client|okhttp|java/|` +
	`libwww|axios/|node-fetch|undici|postman|insomnia|powershell|aiohttp|httpx|scrapy|phantomjs|` +
	`chatgpt|gptbot|claude|anthropic|perplexity|bytespider|meta-externalagent)`)

// countableRedirect reports whether a final redirect counts as filtered usage.
func countableRedirect(c *gin.Context) bool {
	r := c.Request
	if r.Method != http.MethodGet {
		return false
	}
	if r.Header.Get("Authorization") != "" {
		return false
	}
	for _, h := range []string{"Purpose", "Sec-Purpose", "X-Purpose", "X-Moz"} {
		if strings.Contains(strings.ToLower(r.Header.Get(h)), "prefetch") ||
			strings.Contains(strings.ToLower(r.Header.Get(h)), "preview") {
			return false
		}
	}
	ua := r.Header.Get("User-Agent")
	return ua != "" && !automatedAgent.MatchString(ua)
}
