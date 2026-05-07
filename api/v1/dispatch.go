// api/dispatch.go

package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// reservedSegments are top-level path segments that must never be treated
// as slugs, owners, or shorthand. These are always 404 from the dispatcher.
var reservedSegments = map[string]struct{}{
	"api":         {},
	"health":      {},
	"ready":       {},
	"robots.txt":  {},
	"favicon.ico": {},
	".well-known": {},
	"assets":      {},
	"_nuxt":       {},
	"static":      {},
	"public":      {},
}

// handleNoRoute is Gin's NoRoute handler. It implements the slug, info-page,
// and owner/repo shorthand dispatch logic.
func (s *Server) handleNoRoute(c *gin.Context) {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		c.AbortWithStatus(http.StatusMethodNotAllowed)
		return
	}

	path := c.Request.URL.Path
	if path == "" || path == "/" {
		s.handleRoot(c)
		return
	}
	if s.serveFrontendFile(c, path) {
		return
	}

	rest := strings.TrimPrefix(path, "/")
	first := rest
	if i := strings.IndexByte(first, '/'); i >= 0 {
		first = first[:i]
	}
	if _, reserved := reservedSegments[first]; reserved {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Two-segment paths: owner/repo shorthand.
	if strings.Contains(rest, "/") {
		if !looksLikeOwnerRepo(rest) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		parts := strings.Split(rest, "/")
		s.handleShorthand(c, parts[0], parts[1])
		return
	}

	// Single-segment paths: SLUG or SLUG+.
	if base, ok := strings.CutSuffix(rest, "+"); ok {
		s.handleInfoPage(c, base)
		return
	}
	s.handleSlugRedirect(c, rest)
}

// looksLikeOwnerRepo reports whether path "owner/repo" is plausibly a
// GitHub-style 2-segment route.
func looksLikeOwnerRepo(rest string) bool {
	if rest == "" || strings.HasPrefix(rest, "/") {
		return false
	}
	if strings.Count(rest, "/") != 1 {
		return false
	}
	parts := strings.Split(rest, "/")
	return parts[0] != "" && parts[1] != ""
}
