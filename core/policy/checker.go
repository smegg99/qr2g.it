// core/policy/checker.go

package policy

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/github"
	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/repositories"
)

// Identity verifies a link's GitHub identity and availability before its markers are read.
type Identity interface {
	VerifyOnce(ctx context.Context, l models.Link) (models.Link, error)
}

// Source reads marker locations from GitHub.
type Source interface {
	RootNames(ctx context.Context, databaseID int64) ([]string, error)
	Readme(ctx context.Context, databaseID int64) (string, error)
	BackgroundAllowed(now time.Time) bool
}

// Config sets the checker's schedule and budget.
type Config struct {
	Enabled         bool
	Interval        time.Duration
	BatchSize       int
	FreshTTL        time.Duration
	FailureBackoff  time.Duration
	RefreshCooldown time.Duration
	CheckTimeout    time.Duration
}

// CooldownError means a link was refreshed too recently.
type CooldownError struct{ RetryAfter time.Duration }

func (e *CooldownError) Error() string {
	return fmt.Sprintf("policy: refreshed recently, retry after %s", e.RetryAfter)
}

// Checker keeps owner markers current with periodic and requested checks.
type Checker struct {
	cfg      Config
	source   Source
	identity Identity
	links    *repositories.LinkRepository
	now      func() time.Time
	sf       singleflight.Group

	mu        sync.Mutex
	refreshed map[int64]time.Time
}

// NewChecker wires a Checker.
func NewChecker(cfg Config, source Source, identity Identity, links *repositories.LinkRepository) *Checker {
	if cfg.CheckTimeout <= 0 {
		cfg.CheckTimeout = 30 * time.Second
	}
	return &Checker{cfg: cfg, source: source, identity: identity, links: links, now: time.Now, refreshed: map[int64]time.Time{}}
}

// Enabled reports whether scheduled and automatic checks run.
func (c *Checker) Enabled() bool { return c.cfg.Enabled }

// Run checks due links until ctx is done.
func (c *Checker) Run(ctx context.Context) error {
	if !c.cfg.Enabled {
		logger.Info().Msg("policy checker disabled")
		<-ctx.Done()
		return nil
	}
	t := time.NewTicker(c.cfg.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			c.tick(ctx)
		}
	}
}

func (c *Checker) tick(ctx context.Context) {
	if !c.source.BackgroundAllowed(c.now()) {
		logger.Debug().Msg("policy tick skipped: github allowance reserved for requests")
		return
	}
	due, err := c.links.ClaimPolicyDue(ctx, c.cfg.BatchSize, int(c.cfg.FailureBackoff.Seconds()))
	if err != nil {
		logger.Error().Err(err).Msg("policy claim failed")
		return
	}
	for _, l := range due {
		if _, err := c.Check(ctx, l); err != nil {
			logger.Debug().Int64("link_id", l.ID).Err(err).Msg("policy check failed")
		}
	}
}

// Refresh checks a link now unless it was refreshed within the cooldown.
func (c *Checker) Refresh(ctx context.Context, l models.Link) (models.Link, error) {
	c.mu.Lock()
	now := c.now()
	if last, ok := c.refreshed[l.ID]; ok && now.Sub(last) < c.cfg.RefreshCooldown {
		c.mu.Unlock()
		return l, &CooldownError{RetryAfter: c.cfg.RefreshCooldown - now.Sub(last)}
	}
	c.refreshed[l.ID] = now
	for id, at := range c.refreshed {
		if now.Sub(at) >= c.cfg.RefreshCooldown {
			delete(c.refreshed, id)
		}
	}
	c.mu.Unlock()
	return c.run(ctx, l, true)
}

// Check reads markers, verifying identity first unless a recent verification still holds.
func (c *Checker) Check(ctx context.Context, l models.Link) (models.Link, error) {
	return c.run(ctx, l, false)
}

// run coalesces concurrent checks of one link into one result.
func (c *Checker) run(ctx context.Context, l models.Link, reverify bool) (models.Link, error) {
	res, err, _ := c.sf.Do(strconv.FormatInt(l.ID, 10), func() (any, error) {
		shared, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.cfg.CheckTimeout)
		defer cancel()
		return c.check(shared, l, reverify)
	})
	if updated, ok := res.(models.Link); ok {
		return updated, err
	}
	return l, err
}

func (c *Checker) check(ctx context.Context, l models.Link, reverify bool) (models.Link, error) {
	verified, err := l, error(nil)
	if reverify || !l.IsActive() || !l.IsFresh(c.now()) {
		verified, err = c.identity.VerifyOnce(ctx, l)
	}
	if err == nil && !verified.IsActive() {
		return verified, nil
	}
	if err != nil {
		if !verified.IsActive() && l.IsActive() {
			return verified, nil
		}
		return c.fail(ctx, verified, fmt.Errorf("verify identity: %w", err))
	}

	// A root listing that cannot be read, including a 404, is a failed check, never "no markers".
	names, err := c.source.RootNames(ctx, verified.GithubDatabaseID)
	if err != nil {
		return c.fail(ctx, verified, fmt.Errorf("list root: %w", err))
	}
	readme, err := c.source.Readme(ctx, verified.GithubDatabaseID)
	if err != nil {
		return c.fail(ctx, verified, fmt.Errorf("read readme: %w", err))
	}
	m := FromRoot(names).Or(FromReadme(readme))
	updated, err := c.links.ApplyPolicy(ctx, verified.ID, repositories.PolicyUpdate{
		NoIndex:     m.NoIndex,
		NoAnalytics: m.NoAnalytics,
		FreshTTL:    int(c.cfg.FreshTTL.Seconds()),
	})
	if err != nil {
		return verified, fmt.Errorf("apply policy: %w", err)
	}
	if updated.NoIndex != l.NoIndex || updated.NoAnalytics != l.NoAnalytics || l.PolicyCheckedAt == nil {
		logger.Info().Int64("link_id", l.ID).Bool("noindex", m.NoIndex).Bool("noanalytics", m.NoAnalytics).
			Msg("repository policy updated")
	}
	return updated, nil
}

// fail keeps the previous restrictions and postpones the next attempt.
func (c *Checker) fail(ctx context.Context, l models.Link, cause error) (models.Link, error) {
	updated, err := c.links.ApplyPolicyFailure(ctx, l.ID, int(c.cfg.FailureBackoff.Seconds()))
	if err != nil {
		return l, errors.Join(cause, fmt.Errorf("record failure: %w", err))
	}
	return updated, cause
}

// ensure the GitHub client satisfies Source.
var _ Source = (*github.Client)(nil)
