// api/v1/automation.go

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smegg99/s99ab/api/apierr"

	"smegg.me/qr2g.it/core/automation"
	"smegg.me/qr2g.it/core/repositories"
)

type tokenContextKey struct{}

// tokenFrom returns the automation token that authenticated the request.
func tokenFrom(ctx context.Context) (repositories.Token, bool) {
	t, ok := ctx.Value(tokenContextKey{}).(repositories.Token)
	return t, ok
}

// charge is how many quota units an authenticated request costs.
type charge func(c *gin.Context) (int, error)

func chargeOne(*gin.Context) (int, error)  { return 1, nil }
func chargeNone(*gin.Context) (int, error) { return 0, nil }

// chargeBatch costs one unit per repository in the body, which stays readable for the handler.
func chargeBatch(c *gin.Context) (int, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return 0, failWrap(err, kindInvalidBody)
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	var req struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return 0, failWrap(err, kindInvalidBody)
	}
	// An oversized batch is refused before it can spend quota.
	if len(req.Repositories) == 0 || len(req.Repositories) > batchMax {
		return 0, fail(kindInvalidBody)
	}
	return len(req.Repositories), nil
}

// automationAuth authenticates an optional bearer token and charges its daily quota.
// A request without Authorization continues anonymously; a bad token is refused.
func (s *Server) automationAuth(cost charge) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}
		raw, ok := strings.CutPrefix(header, "Bearer ")
		if !ok {
			respond(c, fail(kindTokenInvalid))
			return
		}
		ctx := c.Request.Context()
		token, err := s.Automation.Authenticate(ctx, strings.TrimSpace(raw))
		if errors.Is(err, automation.ErrInvalid) {
			respond(c, fail(kindTokenInvalid))
			return
		}
		if err != nil {
			respond(c, failWrap(err, apierr.Internal))
			return
		}
		units, err := cost(c)
		if err != nil {
			respond(c, err)
			return
		}
		now := time.Now()
		reset := automation.NextReset(now)
		used, err := s.Automation.UsedToday(ctx, token)
		if units > 0 {
			used, err = s.Automation.Consume(ctx, token, units)
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(token.DailyQuota))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		if errors.Is(err, automation.ErrQuotaExceeded) {
			c.Header("X-RateLimit-Remaining", "0")
			respond(c, limited(kindQuotaExceeded, reset.Sub(now)))
			return
		}
		if err != nil {
			respond(c, failWrap(err, apierr.Internal))
			return
		}
		c.Header("X-RateLimit-Remaining", strconv.Itoa(max(token.DailyQuota-used, 0)))
		c.Request = c.Request.WithContext(context.WithValue(ctx, tokenContextKey{}, token))
		c.Next()
	}
}
