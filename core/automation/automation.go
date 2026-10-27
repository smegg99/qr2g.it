// core/automation/automation.go

// Package automation issues and checks quota-limited tokens for agents and scripts.
package automation

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"smegg.me/qr2g.it/core/repositories"
)

// Prefix marks qr2g.it automation tokens so they are recognisable in leaks.
const Prefix = "q2g_"

var (
	ErrInvalid       = errors.New("automation: invalid or revoked token")
	ErrQuotaExceeded = repositories.ErrQuotaExceeded
)

// Service owns token issue, authentication and quota accounting.
type Service struct {
	tokens *repositories.AutomationRepository
}

// NewService wires a Service.
func NewService(tokens *repositories.AutomationRepository) *Service { return &Service{tokens: tokens} }

// Hash is the stored form of a token.
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Issue creates a token and returns its only plaintext copy.
func (s *Service) Issue(ctx context.Context, name string, dailyQuota int) (string, repositories.Token, error) {
	name = strings.TrimSpace(name)
	if name == "" || dailyQuota < 1 {
		return "", repositories.Token{}, errors.New("automation: a name and a positive daily quota are required")
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", repositories.Token{}, fmt.Errorf("automation: random: %w", err)
	}
	raw := Prefix + base64.RawURLEncoding.EncodeToString(secret[:])
	t, err := s.tokens.Create(ctx, name, Hash(raw), dailyQuota)
	return raw, t, err
}

// Authenticate resolves a presented token.
func (s *Service) Authenticate(ctx context.Context, raw string) (repositories.Token, error) {
	if !strings.HasPrefix(raw, Prefix) || len(raw) > 128 {
		return repositories.Token{}, ErrInvalid
	}
	t, err := s.tokens.GetActiveByHash(ctx, Hash(raw))
	if errors.Is(err, repositories.ErrNotFound) {
		return repositories.Token{}, ErrInvalid
	}
	return t, err
}

// Consume charges n requests to today's quota and returns the total used today.
func (s *Service) Consume(ctx context.Context, t repositories.Token, n int) (int, error) {
	return s.tokens.Consume(ctx, t.ID, n, t.DailyQuota)
}

// UsedToday reads today's usage without charging it.
func (s *Service) UsedToday(ctx context.Context, t repositories.Token) (int, error) {
	return s.tokens.UsedToday(ctx, t.ID)
}

// Revoke disables a token permanently.
func (s *Service) Revoke(ctx context.Context, id int64) error { return s.tokens.Revoke(ctx, id) }

// List returns every token without secrets.
func (s *Service) List(ctx context.Context) ([]repositories.Token, error) { return s.tokens.List(ctx) }

// NextReset is the start of the next UTC day, when quotas reset.
func NextReset(now time.Time) time.Time {
	y, m, d := now.UTC().Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC)
}
