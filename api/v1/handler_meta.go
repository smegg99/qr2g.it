// api/handler_meta.go

package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/core/datastore"
)

func (s *Server) handleRoot(c *gin.Context) {
	if s.serveFrontendFile(c, "/") {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"service":     s.serviceName,
		"description": "permanent QR identity layer for public GitHub repositories",
		"docs":        s.publicBaseURL,
		"submit":      "POST " + s.publicBaseURL + "/api/v1/links",
	})
}

// handleHealth implements GET /health.
//
// @Summary      Liveness probe
// @Tags         meta
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleReady implements GET /ready.
//
// @Summary      Readiness probe
// @Description  Pings SQLite with a 2s timeout.
// @Tags         meta
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Router       /ready [get]
func (s *Server) handleReady(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := datastore.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"reason": "database unavailable",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (s *Server) handleRobots(c *gin.Context) {
	if s.serveFrontendFile(c, c.Request.URL.Path) {
		return
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
}

func (s *Server) handleFavicon(c *gin.Context) {
	if s.serveFrontendFile(c, c.Request.URL.Path) {
		return
	}
	c.AbortWithStatus(http.StatusNotFound)
}
