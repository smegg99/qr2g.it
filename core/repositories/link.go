// core/repositories/link.go

package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/models"
)

// LinkRepository owns reads and writes against the links table.
type LinkRepository struct{}

// GetLinkRepository returns a LinkRepository bound to the global datastore.
func GetLinkRepository() *LinkRepository { return &LinkRepository{} }

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

// CreateOrGet atomically inserts a new link by github_node_id, or returns
// the existing one if the node id is already known. The alias table is
// upserted in the same transaction so the node id always resolves.
// created reports whether a fresh row was inserted.
func (r *LinkRepository) CreateOrGet(ctx context.Context, p CreateLinkParams) (link models.Link, created bool, err error) {
	logger.Debug().
		Str("node_id", p.NodeID).
		Str("full_name", p.FullName).
		Msg("create or get link")

	tx, err := datastore.Get().Begin(ctx)
	if err != nil {
		return models.Link{}, false, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	insertSQL := `
        INSERT INTO links (
            github_node_id, github_database_id,
            initial_full_name, current_full_name, current_url,
            verify_after
        ) VALUES (
            $1, $2, $3, $3, $4,
            now() + make_interval(secs => $5)
        )
        ON CONFLICT (github_node_id) DO NOTHING
        RETURNING ` + linkColumns

	link, err = scanLink(tx.QueryRow(ctx, insertSQL, p.NodeID, p.DatabaseID, p.FullName, p.URL, p.FreshTTL))
	switch {
	case err == nil:
		created = true
	case errors.Is(err, pgx.ErrNoRows):
		link, err = scanLink(tx.QueryRow(ctx,
			`SELECT `+linkColumns+` FROM links WHERE github_node_id = $1`,
			p.NodeID,
		))
		if err != nil {
			return models.Link{}, false, wrapErr("get existing", mapErr(err))
		}
	default:
		return models.Link{}, false, fmt.Errorf("insert: %w", err)
	}

	if _, err = tx.Exec(ctx, upsertAliasSQL, p.NodeID, link.ID); err != nil {
		return models.Link{}, false, fmt.Errorf("upsert alias: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
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
	row := datastore.Get().QueryRow(ctx, `SELECT `+linkColumns+` FROM links WHERE id = $1`, id)
	l, err := scanLink(row)
	return l, mapErr(err)
}

// GetByNodeID fetches a link by current github_node_id, falling back to the
// alias table for migrated ids.
func (r *LinkRepository) GetByNodeID(ctx context.Context, nodeID string) (models.Link, error) {
	row := datastore.Get().QueryRow(ctx, `SELECT `+linkColumns+` FROM links WHERE github_node_id = $1`, nodeID)
	l, err := scanLink(row)
	if err == nil {
		return l, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return models.Link{}, err
	}
	row = datastore.Get().QueryRow(ctx, `
        SELECT `+linkColumns+` FROM links
        WHERE id = (SELECT link_id FROM github_node_id_aliases WHERE node_id = $1)
    `, nodeID)
	l, err = scanLink(row)
	return l, mapErr(err)
}

// GetByCurrentFullName looks up a link by current_full_name (case-insensitive).
// Used by the /:owner/:repo shorthand handler.
func (r *LinkRepository) GetByCurrentFullName(ctx context.Context, fullName string) (models.Link, error) {
	row := datastore.Get().QueryRow(ctx, `
        SELECT `+linkColumns+` FROM links
        WHERE LOWER(current_full_name) = LOWER($1)
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

// ApplySuccessfulVerification updates a link with fresh GitHub data. If the
// node id changed (post-migration), the previous node id is recorded as an
// alias and the canonical node id is rotated to the new value.
func (r *LinkRepository) ApplySuccessfulVerification(ctx context.Context, id int64, u VerificationUpdate) (models.Link, error) {
	tx, err := datastore.Get().Begin(ctx)
	if err != nil {
		return models.Link{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := scanLink(tx.QueryRow(ctx,
		`SELECT `+linkColumns+` FROM links WHERE id = $1 FOR UPDATE`, id,
	))
	if err != nil {
		return models.Link{}, wrapErr("lock link", mapErr(err))
	}

	previousNodeID := current.GithubNodeID
	if u.NodeID != "" && u.NodeID != previousNodeID {
		if _, err := tx.Exec(ctx,
			`UPDATE links SET github_node_id = $2 WHERE id = $1`,
			id, u.NodeID,
		); err != nil {
			return models.Link{}, fmt.Errorf("rotate node_id: %w", err)
		}
		if _, err := tx.Exec(ctx, upsertAliasSQL, u.NodeID, id); err != nil {
			return models.Link{}, fmt.Errorf("upsert alias new: %w", err)
		}
		if _, err := tx.Exec(ctx, upsertAliasSQL, previousNodeID, id); err != nil {
			return models.Link{}, fmt.Errorf("upsert alias old: %w", err)
		}
		logger.Info().
			Int64("link_id", id).
			Str("old_node_id", previousNodeID).
			Str("new_node_id", u.NodeID).
			Msg("github node id rotated")
	} else {
		if _, err := tx.Exec(ctx, upsertAliasSQL, previousNodeID, id); err != nil {
			return models.Link{}, fmt.Errorf("touch alias: %w", err)
		}
	}

	updated, err := scanLink(tx.QueryRow(ctx, `
        UPDATE links SET
            current_full_name         = $2,
            current_url               = $3,
            last_successful_full_name = $2,
            last_successful_url       = $3,
            state                     = 'active',
            state_reason              = NULL,
            state_changed_at          = CASE WHEN state <> 'active' THEN now() ELSE state_changed_at END,
            verified_at               = now(),
            verify_after              = now() + make_interval(secs => $4 + floor(random() * GREATEST($5, 1))::int),
            fail_count                = 0,
            last_error                = NULL
        WHERE id = $1
        RETURNING `+linkColumns,
		id, u.FullName, u.URL, u.FreshTTL, u.JitterRange,
	))
	if err != nil {
		return models.Link{}, fmt.Errorf("update link: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Link{}, fmt.Errorf("commit: %w", err)
	}
	return updated, nil
}

// FailureUpdate carries information about a transient verification failure.
type FailureUpdate struct {
	Message     string
	BackoffSecs int
}

// ApplyTransientFailure records a non-terminal failure: increments
// fail_count, pushes verify_after out by BackoffSecs, leaves state as-is.
func (r *LinkRepository) ApplyTransientFailure(ctx context.Context, id int64, f FailureUpdate) (models.Link, error) {
	row := datastore.Get().QueryRow(ctx, `
        UPDATE links SET
            fail_count   = fail_count + 1,
            last_error   = $2,
            verify_after = now() + make_interval(secs => $3)
        WHERE id = $1
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
	row := datastore.Get().QueryRow(ctx, `
        UPDATE links SET
            state            = $2,
            state_reason     = $3,
            state_changed_at = CASE WHEN state <> $2 THEN now() ELSE state_changed_at END,
            verify_after     = now() + make_interval(secs => $4),
            last_error       = $3
        WHERE id = $1
        RETURNING `+linkColumns,
		id, t.State, t.Reason, t.Backoff,
	)
	l, err := scanLink(row)
	return l, mapErr(err)
}

// ClaimStale atomically claims up to limit active links that are due for
// verification, pushing their verify_after out by leaseSecs so other workers
// skip them. Returns the claimed rows.
func (r *LinkRepository) ClaimStale(ctx context.Context, limit, leaseSecs int) ([]models.Link, error) {
	rows, err := datastore.Get().Query(ctx, `
        WITH claimed AS (
            SELECT id FROM links
            WHERE state = 'active' AND verify_after <= now()
            ORDER BY verify_after
            LIMIT $1
            FOR UPDATE SKIP LOCKED
        )
        UPDATE links
        SET verify_after = now() + make_interval(secs => $2)
        WHERE id IN (SELECT id FROM claimed)
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
	err := datastore.Get().QueryRow(ctx, `SELECT count(*) FROM links WHERE state = 'active'`).Scan(&n)
	return n, err
}
