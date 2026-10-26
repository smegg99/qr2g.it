// core/repositories/link.go

package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/models"
)

// LinkRepository owns reads and writes against the links table.
type LinkRepository struct{ db *sql.DB }

// GetLinkRepository returns a LinkRepository bound to the global datastore.
func GetLinkRepository() *LinkRepository { return &LinkRepository{db: datastore.Get()} }

const linkColumns = `
    id, github_node_id, github_database_id,
    initial_full_name, current_full_name, current_url,
    last_successful_full_name, last_successful_url,
    state, state_reason, state_changed_at,
    created_at, verified_at, verify_after,
    fail_count, last_error`

func scanLink(r row) (models.Link, error) {
	var l models.Link
	err := r.Scan(
		&l.ID,
		&l.GithubNodeID,
		&l.GithubDatabaseID,
		&l.InitialFullName,
		&l.CurrentFullName,
		&l.CurrentURL,
		&l.LastSuccessfulFullName,
		&l.LastSuccessfulURL,
		&l.State,
		&l.StateReason,
		&l.StateChangedAt,
		&l.CreatedAt,
		&l.VerifiedAt,
		&l.VerifyAfter,
		&l.FailCount,
		&l.LastError,
	)
	return l, err
}

// CreateLinkParams holds the inputs needed to create a new link row.
type CreateLinkParams struct {
	NodeID     string
	DatabaseID int64
	FullName   string
	URL        string
	FreshTTL   int // seconds until first verification
}

// CreateOrGet stores the link and alias atomically, reporting whether the link is new.
func (r *LinkRepository) CreateOrGet(ctx context.Context, p CreateLinkParams) (link models.Link, created bool, err error) {
	logger.Debug().
		Str("node_id", p.NodeID).
		Str("full_name", p.FullName).
		Msg("create or get link")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Link{}, false, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	insertSQL := `
        INSERT INTO links (
            github_node_id, github_database_id,
            initial_full_name, current_full_name, current_url,
            verify_after
        ) VALUES (
            ?1, ?2, ?3, ?3, ?4,
            strftime('%Y-%m-%d %H:%M:%f', 'now', ?5 || ' seconds')
        )
        ON CONFLICT (github_node_id) DO NOTHING
        RETURNING ` + linkColumns

	link, err = scanLink(tx.QueryRowContext(ctx, insertSQL, p.NodeID, p.DatabaseID, p.FullName, p.URL, p.FreshTTL))
	switch {
	case err == nil:
		created = true
	case errors.Is(err, sql.ErrNoRows):
		link, err = scanLink(tx.QueryRowContext(ctx,
			`SELECT `+linkColumns+` FROM links WHERE github_node_id = ?1`,
			p.NodeID,
		))
		if err != nil {
			return models.Link{}, false, wrapErr("get existing", mapErr(err))
		}
	default:
		return models.Link{}, false, fmt.Errorf("insert: %w", err)
	}

	if _, err = tx.ExecContext(ctx, upsertAliasSQL, p.NodeID, link.ID); err != nil {
		return models.Link{}, false, fmt.Errorf("upsert alias: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return models.Link{}, false, fmt.Errorf("commit: %w", err)
	}

	logger.Debug().
		Int64("id", link.ID).
		Bool("created", created).
		Msg("link ready")
	return link, created, nil
}

// GetByID fetches a link by primary id.
func (r *LinkRepository) GetByID(ctx context.Context, id int64) (models.Link, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+linkColumns+` FROM links WHERE id = ?1`, id)
	l, err := scanLink(row)
	return l, mapErr(err)
}

// GetByNodeID resolves the current GitHub node ID or a recorded alias.
func (r *LinkRepository) GetByNodeID(ctx context.Context, nodeID string) (models.Link, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+linkColumns+` FROM links WHERE github_node_id = ?1`, nodeID)
	l, err := scanLink(row)
	if err == nil {
		return l, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return models.Link{}, err
	}
	row = r.db.QueryRowContext(ctx, `
        SELECT `+linkColumns+` FROM links
        WHERE id = (SELECT link_id FROM github_node_id_aliases WHERE node_id = ?1)
    `, nodeID)
	l, err = scanLink(row)
	return l, mapErr(err)
}

// GetByCurrentFullName finds a repository name without regard to case.
func (r *LinkRepository) GetByCurrentFullName(ctx context.Context, fullName string) (models.Link, error) {
	row := r.db.QueryRowContext(ctx, `
        SELECT `+linkColumns+` FROM links
        WHERE LOWER(current_full_name) = LOWER(?1)
        LIMIT 1
    `, fullName)
	l, err := scanLink(row)
	return l, mapErr(err)
}

// VerificationUpdate carries fresh GitHub identity facts.
type VerificationUpdate struct {
	NodeID      string
	FullName    string
	URL         string
	FreshTTL    int // seconds until next verify_after
	JitterRange int // seconds of additional random jitter
}

// ApplySuccessfulVerification refreshes repository facts and records changed node IDs as aliases.
func (r *LinkRepository) ApplySuccessfulVerification(ctx context.Context, id int64, u VerificationUpdate) (models.Link, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Link{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := scanLink(tx.QueryRowContext(ctx,
		`SELECT `+linkColumns+` FROM links WHERE id = ?1`, id,
	))
	if err != nil {
		return models.Link{}, wrapErr("lock link", mapErr(err))
	}

	previousNodeID := current.GithubNodeID
	if u.NodeID != "" && u.NodeID != previousNodeID {
		if _, err := tx.ExecContext(ctx,
			`UPDATE links SET github_node_id = ?2 WHERE id = ?1`,
			id, u.NodeID,
		); err != nil {
			return models.Link{}, fmt.Errorf("rotate node_id: %w", err)
		}
		if _, err := tx.ExecContext(ctx, upsertAliasSQL, u.NodeID, id); err != nil {
			return models.Link{}, fmt.Errorf("upsert alias new: %w", err)
		}
		if _, err := tx.ExecContext(ctx, upsertAliasSQL, previousNodeID, id); err != nil {
			return models.Link{}, fmt.Errorf("upsert alias old: %w", err)
		}
		logger.Info().
			Int64("link_id", id).
			Str("old_node_id", previousNodeID).
			Str("new_node_id", u.NodeID).
			Msg("github node id rotated")
	} else {
		if _, err := tx.ExecContext(ctx, upsertAliasSQL, previousNodeID, id); err != nil {
			return models.Link{}, fmt.Errorf("touch alias: %w", err)
		}
	}

	updated, err := scanLink(tx.QueryRowContext(ctx, `
        UPDATE links SET
            current_full_name         = ?2,
            current_url               = ?3,
            last_successful_full_name = ?2,
            last_successful_url       = ?3,
            state                     = 'active',
            state_reason              = NULL,
            state_changed_at          = CASE WHEN state <> 'active' THEN strftime('%Y-%m-%d %H:%M:%f', 'now') ELSE state_changed_at END,
            verified_at               = strftime('%Y-%m-%d %H:%M:%f', 'now'),
            verify_after              = strftime('%Y-%m-%d %H:%M:%f', 'now', ?4 || ' seconds'),
            fail_count                = 0,
            last_error                = NULL
        WHERE id = ?1
        RETURNING `+linkColumns,
		id, u.FullName, u.URL, u.FreshTTL+rand.IntN(max(u.JitterRange, 1)),
	))
	if err != nil {
		return models.Link{}, fmt.Errorf("update link: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return models.Link{}, fmt.Errorf("commit: %w", err)
	}
	return updated, nil
}

// FailureUpdate carries information about a transient verification failure.
type FailureUpdate struct {
	Message     string
	BackoffSecs int
}

// ApplyTransientFailure records a failure and postpones verification without changing link state.
func (r *LinkRepository) ApplyTransientFailure(ctx context.Context, id int64, f FailureUpdate) (models.Link, error) {
	row := r.db.QueryRowContext(ctx, `
        UPDATE links SET
            fail_count   = fail_count + 1,
            last_error   = ?2,
            verify_after = strftime('%Y-%m-%d %H:%M:%f', 'now', ?3 || ' seconds')
        WHERE id = ?1
        RETURNING `+linkColumns,
		id, f.Message, f.BackoffSecs,
	)
	l, err := scanLink(row)
	return l, mapErr(err)
}

// TombstoneUpdate carries a terminal state transition.
type TombstoneUpdate struct {
	State   models.State
	Reason  string
	Backoff int // seconds before we re-check
}

// ApplyTombstone moves a link into a terminal state with a reason.
func (r *LinkRepository) ApplyTombstone(ctx context.Context, id int64, t TombstoneUpdate) (models.Link, error) {
	row := r.db.QueryRowContext(ctx, `
        UPDATE links SET
            state            = ?2,
            state_reason     = ?3,
            state_changed_at = CASE WHEN state <> ?2 THEN strftime('%Y-%m-%d %H:%M:%f', 'now') ELSE state_changed_at END,
            verify_after     = strftime('%Y-%m-%d %H:%M:%f', 'now', ?4 || ' seconds'),
            last_error       = ?3
        WHERE id = ?1
        RETURNING `+linkColumns,
		id, t.State, t.Reason, t.Backoff,
	)
	l, err := scanLink(row)
	return l, mapErr(err)
}

// ClaimStale leases due active links so other workers skip them until the lease expires.
func (r *LinkRepository) ClaimStale(ctx context.Context, limit, leaseSecs int) ([]models.Link, error) {
	rows, err := r.db.QueryContext(ctx, `
        UPDATE links
        SET verify_after = strftime('%Y-%m-%d %H:%M:%f', 'now', ?2 || ' seconds')
        WHERE id IN (
            SELECT id FROM links
            WHERE state = 'active' AND verify_after <= strftime('%Y-%m-%d %H:%M:%f', 'now')
            ORDER BY verify_after
            LIMIT CAST(?1 AS INTEGER)
        )
        RETURNING `+linkColumns,
		limit, leaseSecs,
	)
	if err != nil {
		return nil, fmt.Errorf("claim: %w", err)
	}
	defer rows.Close()

	var out []models.Link
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CountActive returns the number of links in active state.
func (r *LinkRepository) CountActive(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM links WHERE state = 'active'`).Scan(&n)
	return n, err
}
