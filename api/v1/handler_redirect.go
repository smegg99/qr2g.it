// api/handler_redirect.go

package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/repositories"
	"smegg.me/qr2g.it/core/slug"
)

// handleSlugRedirect implements GET /:SLUG. Verifies stale links before
// redirecting; never redirects after verification failure.
func (s *Server) handleSlugRedirect(c *gin.Context, raw string) {
	ip := getString(c, clientIPKey)

	id, err := slug.Decode(raw)
	if err != nil {
		s.missBucket.Allow(ip)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if !s.resolveBucket.Allow(ip) {
		writeJSONError(c, http.StatusTooManyRequests, "rate_limited", "too many requests")
		c.Abort()
		return
	}

	link, err := repositories.GetLinkRepository().GetByID(c.Request.Context(), id)
	if errors.Is(err, repositories.ErrNotFound) {
		s.missBucket.Allow(ip)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if err != nil {
		logger.Error().Err(err).Int64("link_id", id).Msg("redirect: get link failed")
		writeJSONError(c, http.StatusInternalServerError, "internal", "internal server error")
		return
	}

	resolved, status, vErr := s.resolveOrVerify(c.Request.Context(), link)
	if vErr != nil {
		writeJSONError(c, http.StatusServiceUnavailable, "verification_failed",
			"could not verify github repository identity, try again shortly")
		return
	}

	if status == http.StatusGone {
		c.JSON(http.StatusGone, gin.H{
			"error":       "tombstoned",
			"slug":        raw,
			"state":       string(resolved.State),
			"stateReason": derefString(resolved.StateReason),
			"message":     "this qr2g.it link is no longer active",
		})
		return
	}

	writeRedirect(c, resolved.CurrentURL)
}

// handleInfoPage implements GET /:SLUG+. Returns JSON metadata without
// verification or redirect.
func (s *Server) handleInfoPage(c *gin.Context, raw string) {
	ip := getString(c, clientIPKey)
	if !s.resolveBucket.Allow(ip) {
		writeJSONError(c, http.StatusTooManyRequests, "rate_limited", "too many requests")
		return
	}

	id, err := slug.Decode(raw)
	if err != nil {
		s.missBucket.Allow(ip)
		writeJSONError(c, http.StatusNotFound, "not_found", "no such link")
		return
	}

	link, err := repositories.GetLinkRepository().GetByID(c.Request.Context(), id)
	if errors.Is(err, repositories.ErrNotFound) {
		s.missBucket.Allow(ip)
		writeJSONError(c, http.StatusNotFound, "not_found", "no such link")
		return
	}
	if err != nil {
		logger.Error().Err(err).Int64("link_id", id).Msg("info: get link failed")
		writeJSONError(c, http.StatusInternalServerError, "internal", "internal server error")
		return
	}
	c.JSON(http.StatusOK, s.buildLinkResponse(link))
}

func (s *Server) redirectToSlug(c *gin.Context, id int64) {
	sl, err := slug.Encode(id)
	if err != nil {
		logger.Error().Err(err).Int64("id", id).Msg("encode slug failed")
		writeJSONError(c, http.StatusInternalServerError, "internal", "internal server error")
		return
	}
	writeRedirect(c, s.publicBaseURL+"/"+sl)
}
