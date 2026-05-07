// api/resolve.go

package api

import (
	"context"
	"net/http"
	"time"

	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/verifier"
)

// resolveOrVerify is the shared resolution flow used by both API resolve and
// public redirect. Returns a fresh link, an HTTP status (200 redirect-ready,
// 410 tombstoned), or an error on transient verification failure.
func (s *Server) resolveOrVerify(ctx context.Context, link models.Link) (models.Link, int, error) {
	if link.State != models.StateActive {
		return link, http.StatusGone, nil
	}
	if time.Now().Before(link.VerifyAfter) {
		return link, http.StatusOK, nil
	}
	verified, err := verifier.Get().VerifyOnce(ctx, link)
	if err != nil {
		return link, 0, err
	}
	if verified.State != models.StateActive {
		return verified, http.StatusGone, nil
	}
	return verified, http.StatusOK, nil
}
