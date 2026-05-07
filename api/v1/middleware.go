// api/middleware.go

package api

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/common/ratelimit"
)

const (
	requestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
	clientIPKey     = "client_ip"
)

func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		c.Set(requestIDKey, id)
		c.Set(clientIPKey, s.extractClientIP(c))
		c.Writer.Header().Set(requestIDHeader, id)
		c.Next()
	}
}

func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, err any) {
		logger.Error().
			Interface("panic", err).
			Str("path", c.Request.URL.Path).
			Msg("panic recovered")
		writeJSONError(c, http.StatusInternalServerError, "internal", "internal server error")
	})
}

func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start)
		logger.Info().
			Str("request_id", getString(c, requestIDKey)).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Int("size", c.Writer.Size()).
			Dur("duration", dur).
			Str("ip", getString(c, clientIPKey)).
			Msg("http request")
	}
}

func (s *Server) bodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func (s *Server) rateLimitMiddleware(bucket *ratelimit.Bucket, label string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getString(c, clientIPKey)
		if !bucket.Allow(ip) {
			logger.Debug().Str("limiter", label).Str("ip", ip).Msg("rate limited")
			writeJSONError(c, http.StatusTooManyRequests, "rate_limited", "too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}

// extractClientIP returns the originating client IP based on the configured
// trusted proxy mode. We never trust X-Forwarded-For without explicit opt-in.
func (s *Server) extractClientIP(c *gin.Context) string {
	switch s.trustedProxyMode {
	case "cloudflare":
		if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
			return strings.TrimSpace(ip)
		}
		return remoteHost(c.Request.RemoteAddr)
	case "all":
		if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
			parts := strings.Split(ip, ",")
			return strings.TrimSpace(parts[0])
		}
		if ip := c.GetHeader("X-Real-IP"); ip != "" {
			return strings.TrimSpace(ip)
		}
		return remoteHost(c.Request.RemoteAddr)
	default:
		return remoteHost(c.Request.RemoteAddr)
	}
}

func remoteHost(addr string) string {
	if addr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func newRequestID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func getString(c *gin.Context, key string) string {
	v, ok := c.Get(key)
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
