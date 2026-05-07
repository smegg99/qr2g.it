// common/breaker/breaker.go

package breaker

import (
	"errors"
	"sync"
	"time"
)

// State of the circuit.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// ErrOpen is returned by Allow when the circuit is open.
var ErrOpen = errors.New("breaker: open")

// Config tunes a Breaker.
type Config struct {
	Threshold int           // consecutive failures before opening
	OpenFor   time.Duration // how long to stay open before half-open trial
}

// Breaker is safe for concurrent use.
type Breaker struct {
	cfg Config

	mu          sync.Mutex
	state       State
	failures    int
	openedAt    time.Time
	halfOpenSet bool
}

// New constructs a Breaker. Threshold <= 0 falls back to 5; OpenFor <= 0
// falls back to 30s.
func New(cfg Config) *Breaker {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 5
	}
	if cfg.OpenFor <= 0 {
		cfg.OpenFor = 30 * time.Second
	}
	return &Breaker{cfg: cfg, state: StateClosed}
}

// Allow returns nil if a call may proceed, or ErrOpen if the circuit is
// open. If the circuit is half-open this returns nil for the first caller
// only; subsequent callers see ErrOpen until the trial finishes.
func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Since(b.openedAt) < b.cfg.OpenFor {
			return ErrOpen
		}
		b.state = StateHalfOpen
		b.halfOpenSet = true
		return nil
	case StateHalfOpen:
		if b.halfOpenSet {
			return ErrOpen
		}
		b.halfOpenSet = true
		return nil
	}
	return nil
}

// OnSuccess records a successful call.
func (b *Breaker) OnSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.state = StateClosed
	b.halfOpenSet = false
}

// OnFailure records a failed call and may open the circuit.
func (b *Breaker) OnFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.state == StateHalfOpen {
		b.state = StateOpen
		b.openedAt = time.Now()
		b.halfOpenSet = false
		return
	}
	if b.failures >= b.cfg.Threshold {
		b.state = StateOpen
		b.openedAt = time.Now()
	}
}

// State returns a snapshot of the current state.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}
