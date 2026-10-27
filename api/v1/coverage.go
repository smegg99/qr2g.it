// api/v1/coverage.go

package api

import (
	"context"
	"sync"
	"time"

	"github.com/smegg99/s99ab/api/apierr"

	"smegg.me/qr2g.it/api/v1/gen"
	"smegg.me/qr2g.it/common/logger"
)

const (
	coverageMaxAge  = 12 * time.Hour
	coverageTimeout = 5 * time.Second
)

// coverageCache holds GitHub's public repository count; one search request every 12 hours at most.
type coverageCache struct {
	mu         sync.Mutex
	total      int64
	countedAt  time.Time
	refreshing bool
}

// coverageTotal returns the cached count at once and asks GitHub in the background when it is old;
// a request never waits for GitHub.
func (s *Server) coverageTotal() (int64, time.Time) {
	c := &s.coverage
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.countedAt) > coverageMaxAge && !c.refreshing {
		c.refreshing = true
		go s.refreshCoverage()
	}
	return c.total, c.countedAt
}

// refreshCoverage asks GitHub for its public repository count; the server starts one as it starts.
func (s *Server) refreshCoverage() {
	ctx, cancel := context.WithTimeout(s.base, coverageTimeout)
	defer cancel()
	n, err := s.GitHub.PublicRepositoryCount(ctx)
	c := &s.coverage
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refreshing = false
	if err != nil {
		logger.Warn().Err(err).Msg("github repository count failed")
		return
	}
	// GitHub stops counting at a different point each time; as a lower bound, the largest answer is the closest.
	c.total, c.countedAt = max(c.total, n), time.Now().UTC()
}

// GetCoverage counts mapped repositories next to GitHub's public repository count.
func (h handlers) GetCoverage(ctx context.Context, _ gen.GetCoverageRequestObject) (gen.GetCoverageResponseObject, error) {
	mapped, err := h.Links.CountActive(ctx)
	if err != nil {
		return nil, failWrap(err, apierr.Internal)
	}
	out := gen.GetCoverage200JSONResponse{Mapped: int(mapped)}
	if total, at := h.coverageTotal(); total > 0 {
		out.PublicRepositories = &total
		out.CountedAt = &at
	}
	return out, nil
}
