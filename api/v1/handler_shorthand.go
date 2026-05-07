// api/handler_shorthand.go

package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/githuburl"
	"smegg.me/qr2g.it/core/repositories"
	"smegg.me/qr2g.it/core/slug"
	"smegg.me/qr2g.it/core/turnstile"
)

// cliUserAgentPrefixes are User-Agent prefixes that opt the client into the
// plain-text response shape. Browsers fall through to the redirect path.
var cliUserAgentPrefixes = []string{
	"curl/",
	"Wget/", "wget/",
	"HTTPie/", "httpie/",
	"python-requests/",
	"Go-http-client/",
	"PowerShell/", "WindowsPowerShell/",
}

// handleShorthand implements GET /:owner/:repo. Resolves an existing link
// or creates a new one from a fresh GitHub lookup. Browsers get a 302 to
// /:SLUG; CLI clients (curl, wget, ...) get the public short URL as
// plain text so `curl qr2g.it/owner/repo` prints the link directly. The
// CLI shape uses the lowercase public base URL (humans paste this), not
// the uppercase canonical form that lives in QR codes.
func (s *Server) handleShorthand(c *gin.Context, owner, repo string) {
	ip := getString(c, clientIPKey)
	wantsText := wantsPlainText(c)

	repoRef, err := githuburl.ParseOwnerRepo(owner, repo)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Cheap path: known by current_full_name → straight respond.
	if link, err := repositories.GetLinkRepository().
		GetByCurrentFullName(c.Request.Context(), repoRef.FullName()); err == nil {
		s.respondShorthand(c, link.ID, wantsText)
		return
	} else if !errors.Is(err, repositories.ErrNotFound) {
		logger.Error().Err(err).Msg("shorthand: get by full name failed")
		writeJSONError(c, http.StatusInternalServerError, "internal", "internal server error")
		return
	}

	// Cold path requires GitHub lookup. Apply create-class limits.
	if !s.shorthandBucket.Allow(ip) || !s.shorthandHourly.Allow(ip) {
		writeJSONError(c, http.StatusTooManyRequests, "rate_limited", "shorthand create limit reached")
		return
	}

	if s.requireShorthandTurnstile && turnstile.Get().IsEnabled() {
		token := c.Query("cf-turnstile-response")
		if token == "" {
			token = c.GetHeader("X-Turnstile-Token")
		}
		if err := s.verifyTurnstile(c, token, ip); err != nil {
			return
		}
	}

	link, _, err := s.lookupAndStore(c.Request.Context(), repoRef)
	if err != nil {
		s.respondLookupError(c, err)
		return
	}
	s.respondShorthand(c, link.ID, wantsText)
}

func (s *Server) respondShorthand(c *gin.Context, id int64, wantsText bool) {
	if !wantsText {
		s.redirectToSlug(c, id)
		return
	}
	sl, err := slug.Encode(id)
	if err != nil {
		logger.Error().Err(err).Int64("id", id).Msg("encode slug failed")
		writeJSONError(c, http.StatusInternalServerError, "internal", "internal server error")
		return
	}
	c.Writer.Header().Set("Cache-Control", "no-store, private")
	c.Writer.Header().Set("X-Robots-Tag", "noindex, nofollow")
	c.String(http.StatusOK, "%s\n", s.publicBaseURL+"/"+sl)
}

// wantsPlainText returns true when the client is a CLI tool or has explicitly
// asked for a plain-text response via Accept or query string. The detection is
// conservative: browsers send Accept headers prefixed with text/html and never
// match these UA prefixes.
func wantsPlainText(c *gin.Context) bool {
	switch c.Query("format") {
	case "text", "txt", "plain":
		return true
	}
	if c.Query("text") != "" {
		return true
	}

	accept := c.GetHeader("Accept")
	if accept != "" && !strings.Contains(accept, "text/html") &&
		(strings.Contains(accept, "text/plain") || strings.Contains(accept, "text/*")) {
		return true
	}

	ua := c.GetHeader("User-Agent")
	for _, p := range cliUserAgentPrefixes {
		if strings.HasPrefix(ua, p) {
			return true
		}
	}
	return false
}
