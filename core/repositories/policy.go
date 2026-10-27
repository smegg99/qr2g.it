// core/repositories/policy.go

package repositories

import (
	"context"
	"fmt"

	"smegg.me/qr2g.it/core/models"
)

// PolicyUpdate is the outcome of a successful owner policy check.
type PolicyUpdate struct {
	NoIndex     bool
	NoAnalytics bool
	FreshTTL    int // seconds until the next scheduled check
}

// ApplyPolicy stores confirmed markers and opens or closes the analytics pause.
func (r *LinkRepository) ApplyPolicy(ctx context.Context, id int64, u PolicyUpdate) (models.Link, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Link{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
        UPDATE links SET
            noindex            = ?2,
            noanalytics        = ?3,
            policy_checked_at  = strftime('%Y-%m-%d %H:%M:%f', 'now'),
            policy_failed      = 0,
            policy_check_after = strftime('%Y-%m-%d %H:%M:%f', 'now', ?4 || ' seconds')
        WHERE id = ?1`, id, u.NoIndex, u.NoAnalytics, u.FreshTTL); err != nil {
		return models.Link{}, fmt.Errorf("update policy: %w", err)
	}
	if u.NoAnalytics {
		_, err = tx.ExecContext(ctx, `
            INSERT INTO analytics_pauses (link_id, reason, started_on)
            SELECT ?1, 'owner_noanalytics', strftime('%Y-%m-%d', 'now')
            WHERE NOT EXISTS (SELECT 1 FROM analytics_pauses WHERE link_id = ?1 AND ended_on IS NULL)`, id)
	} else {
		_, err = tx.ExecContext(ctx, `
            UPDATE analytics_pauses SET ended_on = strftime('%Y-%m-%d', 'now')
            WHERE link_id = ?1 AND ended_on IS NULL`, id)
	}
	if err != nil {
		return models.Link{}, fmt.Errorf("analytics pause: %w", err)
	}
	link, err := getByID(ctx, tx, id)
	if err != nil {
		return models.Link{}, wrapErr("read link", err)
	}
	if err := tx.Commit(); err != nil {
		return models.Link{}, fmt.Errorf("commit: %w", err)
	}
	return link, nil
}

// ApplyPolicyFailure keeps every known restriction and postpones the next attempt.
func (r *LinkRepository) ApplyPolicyFailure(ctx context.Context, id int64, backoffSecs int) (models.Link, error) {
	if _, err := r.db.ExecContext(ctx, `
        UPDATE links SET
            policy_failed      = 1,
            policy_check_after = strftime('%Y-%m-%d %H:%M:%f', 'now', ?2 || ' seconds')
        WHERE id = ?1`, id, backoffSecs); err != nil {
		return models.Link{}, fmt.Errorf("record policy failure: %w", err)
	}
	return r.GetByID(ctx, id)
}

// SetModeration records or clears an operator decision for a link.
func (r *LinkRepository) SetModeration(ctx context.Context, id int64, action models.Moderation, note string) error {
	if action == models.ModerationNone {
		_, err := r.db.ExecContext(ctx, `DELETE FROM moderation WHERE link_id = ?1`, id)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO moderation (link_id, action, note) VALUES (?1, ?2, ?3)
        ON CONFLICT (link_id) DO UPDATE SET action = excluded.action, note = excluded.note,
            created_at = strftime('%Y-%m-%d %H:%M:%f', 'now')`, id, string(action), nullable(note))
	return err
}

// ModeratedLink is one operator decision.
type ModeratedLink struct {
	LinkID   int64
	FullName string
	Action   models.Moderation
	Note     string
}

// ListModeration returns every operator decision, newest first.
func (r *LinkRepository) ListModeration(ctx context.Context) ([]ModeratedLink, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT m.link_id, l.current_full_name, m.action, COALESCE(m.note, '')
        FROM moderation m JOIN links l ON l.id = m.link_id
        ORDER BY m.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModeratedLink
	for rows.Next() {
		var m ModeratedLink
		var action string
		if err := rows.Scan(&m.LinkID, &m.FullName, &action, &m.Note); err != nil {
			return nil, err
		}
		m.Action = models.Moderation(action)
		out = append(out, m)
	}
	return out, rows.Err()
}
