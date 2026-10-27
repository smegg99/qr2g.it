// api/v1/middleware.go

package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smegg99/s99ab/api/apierr"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/common/ratelimit"
)

type clientIPContextKey struct{}

// clientIPFrom returns the client IP the request middleware resolved.
func clientIPFrom(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPContextKey{}).(string)
	return ip
}

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
		ip := s.extractClientIP(c)
		c.Set(requestIDKey, id)
		c.Set(clientIPKey, ip)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), clientIPContextKey{}, ip))
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
		apierr.Respond(c, fail(apierr.Internal))
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

// rateLimitMiddleware applies a per-IP budget; automation tokens have their own quota instead.
func (s *Server) rateLimitMiddleware(bucket *ratelimit.Bucket, label string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := tokenFrom(c.Request.Context()); ok {
			c.Next()
			return
		}
		ip := getString(c, clientIPKey)
		if ok, after := bucket.Allow(ip); !ok {
			logger.Debug().Str("limiter", label).Str("ip", ip).Msg("rate limited")
			respond(c, limited(kindRateLimited, after))
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
