// core/stats/recorder.go

// Package stats aggregates filtered redirects per link and UTC day.
package stats

import (
	"context"
	"sync"
	"time"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/repositories"
)

// Store persists aggregated counts.
type Store interface {
	AddRedirects(ctx context.Context, b repositories.RedirectBatch) error
}

// Recorder buffers counts in memory and flushes them periodically, so a
// redirect never waits on or fails because of the statistics write.
type Recorder struct {
	store    Store
	interval time.Duration
	now      func() time.Time

	mu      sync.Mutex
	pending map[repositories.DayKey]int
	latest  map[int64]time.Time
}

// NewRecorder builds a Recorder flushing every interval.
func NewRecorder(store Store, interval time.Duration) *Recorder {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &Recorder{store: store, interval: interval, now: time.Now, pending: map[repositories.DayKey]int{}, latest: map[int64]time.Time{}}
}

// Record counts one filtered redirect for linkID now.
func (r *Recorder) Record(linkID int64) {
	now := r.now().UTC()
	key := repositories.DayKey{LinkID: linkID, Day: now.Format(time.DateOnly)}
	r.mu.Lock()
	r.pending[key]++
	if now.After(r.latest[linkID]) {
		r.latest[linkID] = now
	}
	r.mu.Unlock()
}

// Flush writes buffered counts; on failure they return to the buffer for the next attempt.
func (r *Recorder) Flush(ctx context.Context) error {
	r.mu.Lock()
	batch := repositories.RedirectBatch{Counts: r.pending, Latest: r.latest}
	r.pending, r.latest = map[repositories.DayKey]int{}, map[int64]time.Time{}
	r.mu.Unlock()
	if len(batch.Counts) == 0 {
		return nil
	}
	if err := r.store.AddRedirects(ctx, batch); err != nil {
		r.mu.Lock()
		for k, n := range batch.Counts {
			r.pending[k] += n
		}
		for id, at := range batch.Latest {
			if at.After(r.latest[id]) {
				r.latest[id] = at
			}
		}
		r.mu.Unlock()
		return err
	}
	return nil
}

// Run flushes until ctx is done, then flushes once more.
func (r *Recorder) Run(ctx context.Context) error {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			final, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			if err := r.Flush(final); err != nil {
				logger.Error().Err(err).Msg("final statistics flush failed")
			}
			return nil
		case <-t.C:
			if err := r.Flush(ctx); err != nil {
				logger.Warn().Err(err).Msg("statistics flush failed; counts kept for retry")
			}
		}
	}
}
