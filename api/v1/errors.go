// api/v1/errors.go

package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smegg99/s99ab/api/apierr"

	"smegg.me/qr2g.it/api/v1/gen"
	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/github"
	"smegg.me/qr2g.it/core/turnstile"
)

// Stable error codes and statuses. The contract's error responses list them.
var (
	kindInvalidBody         = apierr.Kind{Code: "invalid_body", Status: http.StatusBadRequest}
	kindInvalidRequest      = apierr.Kind{Code: "invalid_request", Status: http.StatusBadRequest}
	kindInvalidURL          = apierr.Kind{Code: "invalid_url", Status: http.StatusBadRequest}
	kindInvalidRepository   = apierr.Kind{Code: "invalid_repository", Status: http.StatusBadRequest}
	kindInvalidDesign       = apierr.Kind{Code: "invalid_design", Status: http.StatusBadRequest}
	kindTokenRequired       = apierr.Kind{Code: "token_required", Status: http.StatusUnauthorized}
	kindTokenInvalid        = apierr.Kind{Code: "token_invalid", Status: http.StatusUnauthorized}
	kindRepoPrivate         = apierr.Kind{Code: "repo_private", Status: http.StatusForbidden}
	kindTurnstileRequired   = apierr.Kind{Code: "turnstile_required", Status: http.StatusForbidden}
	kindTurnstileFailed     = apierr.Kind{Code: "turnstile_failed", Status: http.StatusForbidden}
	kindNotFound            = apierr.NotFound
	kindLinkNotFound        = apierr.Kind{Code: "link_not_found", Status: http.StatusNotFound}
	kindRepoNotFound        = apierr.Kind{Code: "repo_not_found", Status: http.StatusNotFound}
	kindNotListed           = apierr.Kind{Code: "not_listed", Status: http.StatusNotFound}
	kindLinkUnavailable     = apierr.Kind{Code: "link_unavailable", Status: http.StatusGone}
	kindRateLimited         = apierr.RateLimited
	kindQuotaExceeded       = apierr.Kind{Code: "quota_exceeded", Status: http.StatusTooManyRequests}
	kindGithubUnavailable   = apierr.Kind{Code: "github_unavailable", Status: http.StatusServiceUnavailable}
	kindGithubRateLimited   = apierr.Kind{Code: "github_rate_limited", Status: http.StatusServiceUnavailable}
	kindVerificationFailed  = apierr.Kind{Code: "verification_failed", Status: http.StatusServiceUnavailable}
	kindTurnstileUnavailble = apierr.Kind{Code: "turnstile_unavailable", Status: http.StatusServiceUnavailable}
	kindContactUnavailable  = apierr.Kind{Code: "contact_unavailable", Status: http.StatusServiceUnavailable}
)

// fail builds an error whose message is the frontend i18n key for its code.
func fail(k apierr.Kind) *apierr.Error { return apierr.New(k, "errors."+k.Code) }

// failWrap is fail with a logged cause.
func failWrap(cause error, k apierr.Kind) *apierr.Error {
	return apierr.Wrap(cause, k, "errors."+k.Code)
}

// retryError carries how long a limited caller should wait.
type retryError struct {
	err   *apierr.Error
	after time.Duration
}

func (e *retryError) Error() string { return e.err.Error() }
func (e *retryError) Unwrap() error { return e.err }

// limited is a 429 with Retry-After.
func limited(k apierr.Kind, after time.Duration) error {
	return &retryError{err: fail(k), after: max(after, time.Second)}
}

// respond writes any error as the envelope, adding Retry-After when known.
func respond(c *gin.Context, err error) {
	var r *retryError
	if errors.As(err, &r) {
		c.Header("Retry-After", strconv.Itoa(int((r.after+time.Second-1)/time.Second)))
	}
	var e *apierr.Error
	if errors.As(err, &e) && e.Kind.Status >= http.StatusInternalServerError {
		logger.Warn().Err(err).Str("path", c.Request.URL.Path).Msg("request failed")
	}
	apierr.Respond(c, err)
}

// strictOptions make the generated boundary's own failures answer the envelope.
func strictOptions() gen.StrictGinServerOptions {
	return gen.StrictGinServerOptions{
		RequestErrorHandlerFunc:  func(c *gin.Context, err error) { respond(c, failWrap(err, kindInvalidBody)) },
		HandlerErrorFunc:         respond,
		ResponseErrorHandlerFunc: respond,
	}
}

// bindError answers a path or query value the wrapper could not bind, such as margin=abc.
func bindError(c *gin.Context, err error, _ int) { respond(c, failWrap(err, kindInvalidRequest)) }

// lookupError maps a GitHub lookup failure during creation; a rate limit carries GitHub's reset time.
// githubDown reports a GitHub failure that says nothing about the repository itself.
func githubDown(err error) bool {
	return errors.Is(err, github.ErrRateLimited) || errors.Is(err, github.ErrTransient) || errors.Is(err, github.ErrUnavailable)
}

func (s *Server) lookupError(err error) error {
	switch {
	case errors.Is(err, github.ErrNotFound):
		return fail(kindRepoNotFound)
	case errors.Is(err, github.ErrPrivate):
		return fail(kindRepoPrivate)
	case errors.Is(err, github.ErrRateLimited):
		return &retryError{err: failWrap(err, kindGithubRateLimited), after: s.GitHub.RetryAfter(time.Now())}
	case errors.Is(err, github.ErrAuth), errors.Is(err, github.ErrTransient),
		errors.Is(err, github.ErrUnavailable), errors.Is(err, github.ErrInvalidResponse):
		return failWrap(err, kindGithubUnavailable)
	default:
		return failWrap(err, apierr.Internal)
	}
}

// turnstileError maps a verification failure; nil means verified.
func turnstileError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, turnstile.ErrEmptyToken):
		return fail(kindTurnstileRequired)
	case errors.Is(err, turnstile.ErrVerifyFailed):
		return failWrap(err, kindTurnstileFailed)
	default:
		return failWrap(err, kindTurnstileUnavailble)
	}
}
