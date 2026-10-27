// common/ratelimit/ratelimit.go
package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter holds a per-key token bucket plus a last-touch timestamp for
// eviction.
type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Bucket is a per-key set of rate.Limiters with eviction.
type Bucket struct {
	rate    rate.Limit
	burst   int
	idleTTL time.Duration

	mu      sync.Mutex
	entries map[string]*entry
}

// NewBucket returns a Bucket configured to allow rps requests per second
// with burst capacity. Idle keys are evicted after idleTTL.
func NewBucket(rps float64, burst int, idleTTL time.Duration) *Bucket {
	if burst <= 0 {
		burst = 1
	}
	if idleTTL <= 0 {
		idleTTL = 10 * time.Minute
	}
	return &Bucket{
		rate:    rate.Limit(rps),
		burst:   burst,
		idleTTL: idleTTL,
		entries: make(map[string]*entry),
	}
}

// Allow consumes one token if available. When the caller is over budget for
// key it returns false and how long until a token is available.
func (b *Bucket) Allow(key string) (bool, time.Duration) {
	if b == nil {
		return true, 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	e, ok := b.entries[key]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(b.rate, b.burst)}
		b.entries[key] = e
	}
	now := time.Now()
	e.lastSeen = now
	r := e.limiter.ReserveN(now, 1)
	if delay := r.DelayFrom(now); delay > 0 {
		r.CancelAt(now)
		return false, delay
	}
	return true, 0
}

// Reset clears the bucket for testing.
func (b *Bucket) Reset() {
	b.mu.Lock()
	b.entries = make(map[string]*entry)
	b.mu.Unlock()
}

// Cleanup removes idle entries. Run periodically from a goroutine.
func (b *Bucket) Cleanup() {
	if b == nil {
		return
	}
	cutoff := time.Now().Add(-b.idleTTL)
	b.mu.Lock()
	for k, e := range b.entries {
		if e.lastSeen.Before(cutoff) {
			delete(b.entries, k)
		}
	}
	b.mu.Unlock()
}

// RunCleanup runs Cleanup at interval until ctx is done.
func (b *Bucket) RunCleanup(ctx context.Context, interval time.Duration) {
	if b == nil {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.Cleanup()
		}
	}
}

// DailyCounter is a small per-key calendar-day counter, used for the
// 50/day-per-IP create limit.
type DailyCounter struct {
	limit int

	mu       sync.Mutex
	day      int // YYYYMMDD in UTC
	counters map[string]int
}

// NewDailyCounter returns a counter with the given per-day limit.
func NewDailyCounter(limit int) *DailyCounter {
	return &DailyCounter{
		limit:    limit,
		counters: make(map[string]int),
	}
}

// Allow increments the per-day count for key. Over the limit it returns false
// and the time until the next UTC day.
func (d *DailyCounter) Allow(key string) (bool, time.Duration) {
	if d == nil || d.limit <= 0 {
		return true, 0
	}
	now := time.Now().UTC()
	day := now.Year()*10000 + int(now.Month())*100 + now.Day()

	d.mu.Lock()
	defer d.mu.Unlock()
	if day != d.day {
		d.day = day
		d.counters = make(map[string]int)
	}
	if d.counters[key] >= d.limit {
		y, m, dd := now.Date()
		return false, time.Date(y, m, dd+1, 0, 0, 0, 0, time.UTC).Sub(now)
	}
	d.counters[key]++
	return true, 0
}

// HourlyCounter is the same idea but bucketed per hour, used for the
// shorthand create limit.
type HourlyCounter struct {
	limit int

	mu       sync.Mutex
	hour     int64
	counters map[string]int
}

// NewHourlyCounter returns an hourly-bucketed counter.
func NewHourlyCounter(limit int) *HourlyCounter {
	return &HourlyCounter{
		limit:    limit,
		counters: make(map[string]int),
	}
}

// Allow increments the per-hour count for key. Over the limit it returns
// false and the time until the next hour.
func (h *HourlyCounter) Allow(key string) (bool, time.Duration) {
	if h == nil || h.limit <= 0 {
		return true, 0
	}
	now := time.Now().UTC()
	hour := now.Unix() / 3600

	h.mu.Lock()
	defer h.mu.Unlock()
	if hour != h.hour {
		h.hour = hour
		h.counters = make(map[string]int)
	}
	if h.counters[key] >= h.limit {
		return false, time.Unix((hour+1)*3600, 0).Sub(now)
	}
	h.counters[key]++
	return true, 0
}
