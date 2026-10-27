// core/verifier/verifier.go

package verifier

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"

	"smegg.me/qr2g.it/common/breaker"
	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/github"
	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/repositories"
)

// Verifier handles both background and request-time GitHub verification.
type Verifier struct {
	enabled        bool
	interval       time.Duration
	batchSize      int
	concurrency    int
	freshTTL       time.Duration
	jitter         time.Duration
	maxFailCount   int
	backoffBase    time.Duration
	backoffMax     time.Duration
	leaseSeconds   int
	requestTimeout time.Duration

	gh      *github.Client
	links   *repositories.LinkRepository
	limiter *rate.Limiter
	breaker *breaker.Breaker
	sf      singleflight.Group
}

var (
	instance *Verifier
	initOnce sync.Once
)

// Errors callers may want to test against.
var ErrCircuitOpen = errors.New("verifier: github circuit open")

func seconds(n int64) time.Duration {
	if n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// Initialize builds the global Verifier from config.
func Initialize() error {
	initOnce.Do(func() {
		v := config.Get().Verifier
		rl := config.Get().Ratelimit

		instance = &Verifier{
			enabled:        v.Enabled,
			interval:       seconds(v.IntervalSeconds),
			batchSize:      max1(int(v.BatchSize)),
			concurrency:    max1(int(v.Concurrency)),
			freshTTL:       seconds(v.FreshTTLSeconds),
			jitter:         seconds(v.StaleJitterSeconds),
			maxFailCount:   max1(int(v.MaxFailCount)),
			backoffBase:    seconds(v.BackoffBaseSeconds),
			backoffMax:     seconds(v.BackoffMaxSeconds),
			leaseSeconds:   60,
			requestTimeout: seconds(v.RequestTimeoutSeconds),

			gh:    github.Get(),
			links: repositories.GetLinkRepository(),
			limiter: rate.NewLimiter(
				rate.Limit(rl.GithubPerSec),
				int(rl.GithubBurst),
			),
			breaker: breaker.New(breaker.Config{
				Threshold: 5,
				OpenFor:   30 * time.Second,
			}),
		}

		logger.Info().
			Bool("enabled", instance.enabled).
			Dur("interval", instance.interval).
			Int("batch_size", instance.batchSize).
			Int("concurrency", instance.concurrency).
			Msg("verifier initialised")
	})
	return nil
}

// Get returns the global Verifier. Panics if Initialize has not run.
func Get() *Verifier {
	if instance == nil {
		panic("verifier: not initialised, call Initialize first")
	}
	return instance
}

// Run blocks until ctx is done, polling the database for stale active links
// and verifying them with bounded concurrency.
func (v *Verifier) Run(ctx context.Context) error {
	if !v.enabled {
		logger.Info().Msg("verifier disabled")
		<-ctx.Done()
		return nil
	}

	logger.Info().Dur("interval", v.interval).Msg("verifier started")

	t := time.NewTicker(v.interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := v.tick(ctx); err != nil {
				logger.Error().Err(err).Msg("verifier tick failed")
			}
		}
	}
}

func (v *Verifier) tick(ctx context.Context) error {
	if !v.gh.BackgroundAllowed(time.Now()) {
		logger.Debug().Msg("verifier tick skipped: github allowance reserved for requests")
		return nil
	}
	links, err := v.links.ClaimStale(ctx, v.batchSize, v.leaseSeconds)
	if err != nil {
		return fmt.Errorf("claim stale: %w", err)
	}
	if len(links) == 0 {
		return nil
	}

	logger.Debug().Int("count", len(links)).Msg("verifier batch claimed")

	sem := make(chan struct{}, v.concurrency)
	var wg sync.WaitGroup
	for _, l := range links {
		wg.Add(1)
		sem <- struct{}{}
		go func(l models.Link) {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := v.VerifyOnce(ctx, l); err != nil {
				logger.Debug().
					Int64("link_id", l.ID).
					Err(err).
					Msg("verify failed")
			}
		}(l)
	}
	wg.Wait()
	return nil
}

// VerifyOnce coalesces concurrent verification of the same link id via
// singleflight. The returned Link reflects the stored state after the attempt,
// including a tombstone recorded together with its cause.
func (v *Verifier) VerifyOnce(ctx context.Context, l models.Link) (models.Link, error) {
	key := strconv.FormatInt(l.ID, 10)
	res, err, _ := v.sf.Do(key, func() (any, error) {
		return v.verify(ctx, l)
	})
	if updated, ok := res.(models.Link); ok && updated.ID == l.ID {
		return updated, err
	}
	return l, err
}

func (v *Verifier) verify(ctx context.Context, l models.Link) (models.Link, error) {
	if err := v.breaker.Allow(); err != nil {
		return l, ErrCircuitOpen
	}

	if v.limiter != nil {
		if err := v.limiter.Wait(ctx); err != nil {
			return l, fmt.Errorf("github wait: %w", err)
		}
	}

	callCtx := ctx
	if v.requestTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, v.requestTimeout)
		defer cancel()
	}

	repo, err := v.lookup(callCtx, l)
	if err == nil {
		v.breaker.OnSuccess()
		return v.applySuccess(ctx, l, repo)
	}

	switch {
	case errors.Is(err, github.ErrPrivate):
		v.breaker.OnSuccess()
		return v.applyTombstoneOrFail(ctx, l, models.StatePrivate, "repository is private", err)

	case errors.Is(err, github.ErrNotFound):
		v.breaker.OnSuccess()
		return v.applyTombstoneOrFail(ctx, l, models.StateUnavailable, "repository is no longer publicly available", err)

	case errors.Is(err, github.ErrRateLimited),
		errors.Is(err, github.ErrAuth),
		errors.Is(err, github.ErrTransient),
		errors.Is(err, github.ErrUnavailable),
		errors.Is(err, github.ErrInvalidResponse):
		v.breaker.OnFailure()
		return v.applyTransient(ctx, l, err)

	default:
		v.breaker.OnFailure()
		return v.applyTransient(ctx, l, err)
	}
}

func (v *Verifier) lookup(ctx context.Context, l models.Link) (github.Repo, error) {
	return v.gh.LookupRepoByDatabaseID(ctx, l.GithubDatabaseID)
}

func (v *Verifier) applySuccess(ctx context.Context, l models.Link, repo github.Repo) (models.Link, error) {
	updated, err := v.links.ApplySuccessfulVerification(ctx, l.ID, repositories.VerificationUpdate{
		NodeID:      repo.NodeID,
		FullName:    repo.FullName,
		URL:         repo.URL,
		FreshTTL:    int(v.freshTTL.Seconds()),
		JitterRange: int(v.jitter.Seconds()),
		Metadata:    Metadata(repo),
	})
	if err != nil {
		return l, fmt.Errorf("apply success: %w", err)
	}
	if v.gh.BackgroundAllowed(time.Now()) {
		updated = v.RefreshLanguages(ctx, updated)
	}
	return updated, nil
}

// maxLanguages is how many of a repository's languages are kept; the rest is too small to show.
const maxLanguages = 6

// RefreshLanguages reads the repository's languages from GitHub and stores them. A failure keeps the
// stored shares, since the colours they drive are decoration.
func (v *Verifier) RefreshLanguages(ctx context.Context, l models.Link) models.Link {
	bytes, err := v.gh.Languages(ctx, l.GithubDatabaseID)
	if err != nil {
		logger.Debug().Int64("link_id", l.ID).Err(err).Msg("languages not refreshed")
		return l
	}
	shares := Shares(bytes)
	if err := v.links.SetLanguages(ctx, l.ID, shares); err != nil {
		logger.Warn().Int64("link_id", l.ID).Err(err).Msg("store languages failed")
		return l
	}
	l.Languages = shares
	return l
}

// Shares turns GitHub's bytes per language into the largest shares of the whole, largest first.
func Shares(bytes map[string]int64) []models.LanguageShare {
	var total int64
	for _, n := range bytes {
		total += max(n, 0)
	}
	out := make([]models.LanguageShare, 0, len(bytes))
	if total == 0 {
		return out
	}
	for name, n := range bytes {
		if n > 0 {
			out = append(out, models.LanguageShare{Name: name, Share: math.Round(float64(n)/float64(total)*1000) / 1000})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Share != out[j].Share {
			return out[i].Share > out[j].Share
		}
		return out[i].Name < out[j].Name
	})
	return out[:min(len(out), maxLanguages)]
}

// Metadata extracts the optional repository facts stored with a link.
func Metadata(repo github.Repo) repositories.Metadata {
	return repositories.Metadata{
		Description:   repo.Description,
		Language:      repo.Language,
		Archived:      repo.Archived,
		DefaultBranch: repo.DefaultBranch,
		Stars:         repo.Stars,
	}
}

// FreshTTL is how long a successful verification stays fresh.
func (v *Verifier) FreshTTL() time.Duration { return v.freshTTL }

func (v *Verifier) applyTombstoneOrFail(ctx context.Context, l models.Link, terminal models.State, reason string, cause error) (models.Link, error) {
	if int(l.FailCount)+1 < v.maxFailCount {
		return v.applyTransient(ctx, l, cause)
	}
	updated, terr := v.links.ApplyTombstone(ctx, l.ID, repositories.TombstoneUpdate{
		State:   terminal,
		Reason:  reason,
		Backoff: int((24 * time.Hour).Seconds()),
	})
	if terr != nil {
		return l, fmt.Errorf("tombstone: %w", terr)
	}
	logger.Info().
		Int64("link_id", l.ID).
		Str("state", string(terminal)).
		Str("reason", reason).
		Msg("link tombstoned")
	return updated, cause
}

func (v *Verifier) applyTransient(ctx context.Context, l models.Link, cause error) (models.Link, error) {
	backoff := v.backoff(int(l.FailCount) + 1)
	updated, terr := v.links.ApplyTransientFailure(ctx, l.ID, repositories.FailureUpdate{
		Message:     cause.Error(),
		BackoffSecs: int(backoff.Seconds()),
	})
	if terr != nil {
		return l, fmt.Errorf("apply failure: %w", terr)
	}
	return updated, cause
}

func (v *Verifier) backoff(failCount int) time.Duration {
	base := v.backoffBase.Seconds()
	factor := math.Pow(2, float64(failCount-1))
	secs := base * factor
	if secs > v.backoffMax.Seconds() {
		secs = v.backoffMax.Seconds()
	}
	jitter := rand.Float64() * 0.3 * secs
	return time.Duration((secs - jitter) * float64(time.Second))
}
