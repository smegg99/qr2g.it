// api/v1/handler_shorthand.go

package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/smegg99/s99ab/api/apierr"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/githuburl"
	"smegg.me/qr2g.it/core/render"
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

// handleShorthand implements GET /:owner/:repo; CLI clients may add ?qr for a terminal QR code. Resolves an existing link
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
	ctx := c.Request.Context()
	if link, err := s.Links.GetByCurrentFullName(ctx, repoRef.FullName()); err == nil {
		s.respondShorthand(c, link.ID, wantsText)
		return
	} else if !errors.Is(err, repositories.ErrNotFound) {
		logger.Error().Err(err).Msg("shorthand: get by full name failed")
		respond(c, failWrap(err, apierr.Internal))
		return
	}

	// Cold path requires GitHub lookup and may create a link. Apply create-class limits.
	if ok, after := s.shorthandBucket.Allow(ip); !ok {
		respond(c, limited(kindRateLimited, after))
		return
	}
	if ok, after := s.shorthandHourly.Allow(ip); !ok {
		respond(c, limited(kindRateLimited, after))
		return
	}

	if s.requireShorthandTurnstile {
		token := c.Query("cf-turnstile-response")
		if token == "" {
			token = c.GetHeader("X-Turnstile-Token")
		}
		if err := s.verifyTurnstile(ctx, token, ip, turnstile.ActionCreateLink); err != nil {
			respond(c, err)
			return
		}
	}

	link, _, err := s.lookupAndStore(ctx, repoRef)
	if err != nil {
		respond(c, err)
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
		respond(c, failWrap(err, apierr.Internal))
		return
	}
	c.Writer.Header().Set("Cache-Control", "no-store, private")
	c.Writer.Header().Set("X-Robots-Tag", "noindex, nofollow")
	// ?qr draws the code above the link; ?qr=invert is for terminals with a light background.
	mode, wantsQR := c.GetQuery("qr")
	if !wantsQR {
		c.String(http.StatusOK, "%s\n", s.publicBaseURL+"/"+sl)
		return
	}
	code, err := render.Terminal(s.canonicalShortURL(sl), mode == "invert")
	if err != nil {
		respond(c, failWrap(err, apierr.Internal))
		return
	}
	c.String(http.StatusOK, "%s\n%s\n", code, s.publicBaseURL+"/"+sl)
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
