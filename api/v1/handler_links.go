// api/handler_links.go

package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/github"
	"smegg.me/qr2g.it/core/githuburl"
	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/repositories"
	"smegg.me/qr2g.it/core/slug"
	"smegg.me/qr2g.it/core/turnstile"
)

// CreateLinkRequest is the JSON body for POST /api/v1/links.
type CreateLinkRequest struct {
	URL            string `json:"url"`
	TurnstileToken string `json:"turnstileToken"`
}

// handleCreateLink implements POST /api/v1/links.
//
// @Summary      Create or reuse a short link
// @Description  Submits a public GitHub repository URL and returns its permanent short link.
// @Description  The same `node_id` always maps back to the same slug.
// @Tags         links
// @Accept       json
// @Produce      json
// @Param        body  body      CreateLinkRequest  true  "Public GitHub repository URL"
// @Success      200   {object}  LinkResponse       "Existing link returned"
// @Success      201   {object}  LinkResponse       "New link created"
// @Failure      400   {object}  ErrorResponse      "Invalid body or URL"
// @Failure      403   {object}  ErrorResponse      "Repository private or Turnstile failure"
// @Failure      404   {object}  ErrorResponse      "Repository not found"
// @Failure      429   {object}  ErrorResponse      "Per-IP rate limit reached"
// @Failure      503   {object}  ErrorResponse      "GitHub or Turnstile temporarily unavailable"
// @Router       /api/v1/links [post]
func (s *Server) handleCreateLink(c *gin.Context) {
	var req CreateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeJSONError(c, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	repo, err := githuburl.Parse(req.URL)
	if err != nil {
		writeJSONError(c, http.StatusBadRequest, "invalid_url", "submit a public GitHub repository URL")
		return
	}

	ip := getString(c, clientIPKey)
	if !s.createDaily.Allow(ip) {
		writeJSONError(c, http.StatusTooManyRequests, "rate_limited", "daily create limit reached")
		return
	}

	if err := s.verifyTurnstile(c, req.TurnstileToken, ip); err != nil {
		return
	}

	link, created, err := s.lookupAndStore(c.Request.Context(), repo)
	if err != nil {
		s.respondLookupError(c, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, s.buildLinkResponse(link))
}

// handleAPIGetLink returns metadata for an existing slug. No verification.
//
// @Summary      Get link metadata
// @Description  Returns stored link metadata without performing GitHub verification.
// @Tags         links
// @Produce      json
// @Param        slug  path      string         true  "Uppercase base36 slug, e.g. AB12"
// @Success      200   {object}  LinkResponse
// @Failure      404   {object}  ErrorResponse  "No such link"
// @Router       /api/v1/links/{slug} [get]
func (s *Server) handleAPIGetLink(c *gin.Context) {
	link, ok := s.loadLinkBySlug(c, c.Param("slug"))
	if !ok {
		return
	}
	c.JSON(http.StatusOK, s.buildLinkResponse(link))
}

// handleAPIResolveLink returns the resolution outcome as JSON without
// performing the redirect. Strict like the public redirect path.
//
// @Summary      Resolve a slug as JSON
// @Description  Verifies stale active links against GitHub and returns the post-verification state.
// @Description  Fails closed: a stale link is not advertised as active when verification fails.
// @Tags         links
// @Produce      json
// @Param        slug  path      string         true  "Uppercase base36 slug, e.g. AB12"
// @Success      200   {object}  LinkResponse   "Active link"
// @Success      410   {object}  LinkResponse   "Tombstoned (private or deleted upstream)"
// @Failure      404   {object}  ErrorResponse  "No such link"
// @Failure      503   {object}  ErrorResponse  "Verification temporarily unavailable"
// @Router       /api/v1/links/{slug}/resolve [get]
func (s *Server) handleAPIResolveLink(c *gin.Context) {
	link, ok := s.loadLinkBySlug(c, c.Param("slug"))
	if !ok {
		return
	}
	resolved, status, err := s.resolveOrVerify(c.Request.Context(), link)
	if err != nil {
		s.respondResolveError(c, err)
		return
	}
	c.JSON(status, s.buildLinkResponse(resolved))
}

func (s *Server) loadLinkBySlug(c *gin.Context, raw string) (models.Link, bool) {
	id, err := slug.Decode(raw)
	if err != nil {
		s.applyMissLimit(c)
		writeJSONError(c, http.StatusNotFound, "not_found", "no such link")
		return models.Link{}, false
	}
	link, err := repositories.GetLinkRepository().GetByID(c.Request.Context(), id)
	if errors.Is(err, repositories.ErrNotFound) {
		s.applyMissLimit(c)
		writeJSONError(c, http.StatusNotFound, "not_found", "no such link")
		return models.Link{}, false
	}
	if err != nil {
		logger.Error().Err(err).Int64("link_id", id).Msg("get link by id failed")
		writeJSONError(c, http.StatusInternalServerError, "internal", "internal server error")
		return models.Link{}, false
	}
	return link, true
}

func (s *Server) applyMissLimit(c *gin.Context) {
	ip := getString(c, clientIPKey)
	s.missBucket.Allow(ip)
}

// lookupAndStore validates the GitHub repo via REST and creates/reuses a
// permanent link row.
func (s *Server) lookupAndStore(ctx context.Context, repo githuburl.Repo) (models.Link, bool, error) {
	gh, err := github.Get().LookupRepoREST(ctx, repo.Owner, repo.Name)
	if err != nil {
		return models.Link{}, false, err
	}
	link, created, err := repositories.GetLinkRepository().CreateOrGet(ctx, repositories.CreateLinkParams{
		NodeID:     gh.NodeID,
		DatabaseID: gh.DatabaseID,
		FullName:   gh.FullName,
		URL:        gh.URL,
		FreshTTL:   3600,
	})
	if err != nil {
		return models.Link{}, false, err
	}
	return link, created, nil
}

// verifyTurnstile validates a token if Turnstile is enabled. Writes an
// error response and returns a non-nil error on failure.
func (s *Server) verifyTurnstile(c *gin.Context, token, ip string) error {
	t := turnstile.Get()
	if !t.IsEnabled() {
		return nil
	}
	err := t.Verify(c.Request.Context(), token, ip)
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, turnstile.ErrEmptyToken):
		writeJSONError(c, http.StatusForbidden, "turnstile_required", "turnstile token required")
	case errors.Is(err, turnstile.ErrVerifyFailed):
		writeJSONError(c, http.StatusForbidden, "turnstile_failed", "turnstile verification failed")
	default:
		logger.Warn().Err(err).Msg("turnstile error")
		writeJSONError(c, http.StatusServiceUnavailable, "turnstile_unavailable", "verification temporarily unavailable")
	}
	return err
}

func (s *Server) respondLookupError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, github.ErrNotFound):
		writeJSONError(c, http.StatusNotFound, "repo_not_found", "github repository not found")
	case errors.Is(err, github.ErrPrivate):
		writeJSONError(c, http.StatusForbidden, "repo_private", "repository is private; only public repositories are supported")
	case errors.Is(err, github.ErrAuth):
		logger.Warn().Err(err).Msg("github auth failed")
		writeJSONError(c, http.StatusServiceUnavailable, "github_auth", "github authentication failed")
	case errors.Is(err, github.ErrRateLimited):
		writeJSONError(c, http.StatusServiceUnavailable, "github_rate_limited", "github rate limit reached, try again shortly")
	case errors.Is(err, github.ErrTransient), errors.Is(err, github.ErrUnavailable), errors.Is(err, github.ErrInvalidResponse):
		writeJSONError(c, http.StatusServiceUnavailable, "github_unavailable", "github temporarily unavailable")
	default:
		logger.Error().Err(err).Msg("create link failed")
		writeJSONError(c, http.StatusInternalServerError, "internal", "internal server error")
	}
}

func (s *Server) respondResolveError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, github.ErrRateLimited),
		errors.Is(err, github.ErrTransient),
		errors.Is(err, github.ErrUnavailable),
		errors.Is(err, github.ErrAuth),
		errors.Is(err, github.ErrInvalidResponse):
		writeJSONError(c, http.StatusServiceUnavailable, "verification_failed", "could not verify github repository identity")
	default:
		logger.Error().Err(err).Msg("resolve failed")
		writeJSONError(c, http.StatusServiceUnavailable, "verification_failed", "could not verify github repository identity")
	}
}
