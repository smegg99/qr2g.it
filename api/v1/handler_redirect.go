// api/v1/handler_redirect.go

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/slug"
)

// handleSlugRedirect implements GET /:SLUG. Verifies stale links before
// redirecting; never redirects after verification failure.
func (s *Server) handleSlugRedirect(c *gin.Context, raw string) {
	ip := getString(c, clientIPKey)
	if _, err := slug.Decode(raw); err != nil {
		s.missBucket.Allow(ip)
		s.serveMissing(c, http.StatusNotFound)
		return
	}
	if ok, after := s.resolveBucket.Allow(ip); !ok {
		respond(c, limited(kindRateLimited, after))
		return
	}
	ctx := c.Request.Context()
	link, err := s.loadLink(ctx, raw)
	if err != nil {
		s.serveMissing(c, http.StatusNotFound)
		return
	}
	resolved, status, err := s.resolveOrVerify(ctx, link)
	if err != nil {
		respond(c, failWrap(err, kindVerificationFailed))
		return
	}
	if status == http.StatusGone {
		s.serveMissing(c, http.StatusGone)
		return
	}
	if resolved.Collects() && countableRedirect(c) {
		s.Recorder.Record(resolved.ID)
	}
	writeRedirect(c, resolved.CurrentURL)
}

// serveMissing answers browsers with the site, which explains the state, and other clients with the envelope.
func (s *Server) serveMissing(c *gin.Context, status int) {
	if acceptsHTML(c) && s.serveShell(c, status, pageMeta{NoIndex: true}) {
		return
	}
	kind := kindNotFound
	if status == http.StatusGone {
		kind = kindLinkUnavailable
	}
	respond(c, fail(kind))
}

// handleInfoPage implements GET /:SLUG+. Returns JSON metadata without
// verification or redirect.
func (s *Server) handleInfoPage(c *gin.Context, raw string) {
	if ok, after := s.resolveBucket.Allow(getString(c, clientIPKey)); !ok {
		respond(c, limited(kindRateLimited, after))
		return
	}
	link, err := s.loadLink(c.Request.Context(), raw)
	if err != nil {
		respond(c, err)
		return
	}
	c.JSON(http.StatusOK, s.wireLink(link))
}

func (s *Server) redirectToSlug(c *gin.Context, id int64) {
	sl, err := slug.Encode(id)
	if err != nil {
		logger.Error().Err(err).Int64("id", id).Msg("encode slug failed")
		respond(c, err)
		return
	}
	writeRedirect(c, s.publicBaseURL+"/"+sl)
}

func writeRedirect(c *gin.Context, location string) {
	c.Writer.Header().Set("Location", location)
	c.Writer.Header().Set("Cache-Control", "no-store, private")
	c.Writer.Header().Set("Pragma", "no-cache")
	c.Writer.Header().Set("X-Robots-Tag", "noindex, nofollow")
	c.Writer.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
	c.Writer.WriteHeader(http.StatusFound)
}
