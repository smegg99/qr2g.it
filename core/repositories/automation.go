// core/repositories/automation.go

package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrQuotaExceeded means the token used its daily quota.
var ErrQuotaExceeded = errors.New("repositories: automation quota exceeded")

// AutomationRepository owns automation tokens and their daily usage.
type AutomationRepository struct{ db *sql.DB }

// NewAutomationRepository binds an AutomationRepository to db.
func NewAutomationRepository(db *sql.DB) *AutomationRepository { return &AutomationRepository{db: db} }

// Token is a stored automation token; the secret itself is never stored.
type Token struct {
	ID         int64
	Name       string
	DailyQuota int
	CreatedAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt *time.Time
}

const tokenColumns = `id, name, daily_quota, created_at, revoked_at, last_used_at`

func scanToken(r row) (Token, error) {
	var t Token
	err := r.Scan(&t.ID, &t.Name, &t.DailyQuota, &t.CreatedAt, &t.RevokedAt, &t.LastUsedAt)
	return t, err
}

// Create stores a token hash and returns the row.
func (r *AutomationRepository) Create(ctx context.Context, name, hash string, quota int) (Token, error) {
	t, err := scanToken(r.db.QueryRowContext(ctx, `
        INSERT INTO automation_tokens (name, token_hash, daily_quota) VALUES (?1, ?2, ?3)
        RETURNING `+tokenColumns, name, hash, quota))
	if err != nil {
		return Token{}, fmt.Errorf("create token: %w", err)
	}
	return t, nil
}

// GetActiveByHash finds an unrevoked token by its hash.
func (r *AutomationRepository) GetActiveByHash(ctx context.Context, hash string) (Token, error) {
	t, err := scanToken(r.db.QueryRowContext(ctx,
		`SELECT `+tokenColumns+` FROM automation_tokens WHERE token_hash = ?1 AND revoked_at IS NULL`, hash))
	return t, mapErr(err)
}

// List returns every token, newest first.
func (r *AutomationRepository) List(ctx context.Context) ([]Token, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+tokenColumns+` FROM automation_tokens ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Token
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Revoke disables a token permanently.
func (r *AutomationRepository) Revoke(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `
        UPDATE automation_tokens SET revoked_at = strftime('%Y-%m-%d %H:%M:%f', 'now')
        WHERE id = ?1 AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Consume adds n requests to today's usage unless that would exceed quota; it returns the new total.
func (r *AutomationRepository) Consume(ctx context.Context, id int64, n, quota int) (int, error) {
	var used int
	err := r.db.QueryRowContext(ctx, `
        INSERT INTO automation_usage (token_id, day, requests)
        SELECT ?1, strftime('%Y-%m-%d', 'now'), ?2 WHERE ?2 <= ?3
        ON CONFLICT (token_id, day) DO UPDATE SET requests = requests + excluded.requests
            WHERE requests + excluded.requests <= ?3
        RETURNING requests`, id, n, quota).Scan(&used)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrQuotaExceeded
	}
	if err != nil {
		return 0, fmt.Errorf("consume quota: %w", err)
	}
	_, _ = r.db.ExecContext(ctx,
		`UPDATE automation_tokens SET last_used_at = strftime('%Y-%m-%d %H:%M:%f', 'now') WHERE id = ?1`, id)
	return used, nil
}

// UsedToday returns today's request count.
func (r *AutomationRepository) UsedToday(ctx context.Context, id int64) (int, error) {
	var used int
	err := r.db.QueryRowContext(ctx, `
        SELECT COALESCE((SELECT requests FROM automation_usage
            WHERE token_id = ?1 AND day = strftime('%Y-%m-%d', 'now')), 0)`, id).Scan(&used)
	return used, err
}
