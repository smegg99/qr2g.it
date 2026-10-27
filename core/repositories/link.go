// core/repositories/link.go

package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
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

// NewLinkRepository binds a LinkRepository to db.
func NewLinkRepository(db *sql.DB) *LinkRepository { return &LinkRepository{db: db} }

// queryer is satisfied by *sql.DB and *sql.Tx.
type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

const linkSelect = `
    SELECT l.id, l.github_node_id, l.github_database_id,
        l.initial_full_name, l.current_full_name, l.current_url,
        l.last_successful_full_name, l.last_successful_url,
        l.state, l.state_reason, l.state_changed_at,
        l.created_at, l.verified_at, l.verify_after,
        l.fail_count, l.last_error,
        l.description, l.language, l.archived, l.default_branch, l.stars, l.languages,
        l.noindex, l.noanalytics, l.policy_checked_at, l.policy_failed, l.policy_check_after,
        COALESCE(m.action, '')
    FROM links l LEFT JOIN moderation m ON m.link_id = l.id`

func scanLink(r row) (models.Link, error) {
	var l models.Link
	var moderation string
	var languages sql.NullString
	err := r.Scan(
		&l.ID, &l.GithubNodeID, &l.GithubDatabaseID,
		&l.InitialFullName, &l.CurrentFullName, &l.CurrentURL,
		&l.LastSuccessfulFullName, &l.LastSuccessfulURL,
		&l.State, &l.StateReason, &l.StateChangedAt,
		&l.CreatedAt, &l.VerifiedAt, &l.VerifyAfter,
		&l.FailCount, &l.LastError,
		&l.Description, &l.Language, &l.Archived, &l.DefaultBranch, &l.Stars, &languages,
		&l.NoIndex, &l.NoAnalytics, &l.PolicyCheckedAt, &l.PolicyFailed, &l.PolicyCheckAfter,
		&moderation,
	)
	l.Moderation = models.Moderation(moderation)
	if err == nil && languages.Valid {
		// A malformed value only loses the colours; the link itself stays readable.
		if jerr := json.Unmarshal([]byte(languages.String), &l.Languages); jerr != nil {
			logger.Warn().Int64("link_id", l.ID).Err(jerr).Msg("stored languages unreadable")
			l.Languages = nil
		}
	}
	return l, err
}

// SetLanguages stores a repository's language shares, largest first.
func (r *LinkRepository) SetLanguages(ctx context.Context, id int64, shares []models.LanguageShare) error {
	raw, err := json.Marshal(shares)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE links SET languages = ?2 WHERE id = ?1`, id, string(raw))
	return err
}

func getByID(ctx context.Context, q queryer, id int64) (models.Link, error) {
	l, err := scanLink(q.QueryRowContext(ctx, linkSelect+` WHERE l.id = ?1`, id))
	return l, mapErr(err)
}

// Metadata carries optional repository facts read with the identity.
type Metadata struct {
	Description   string
	Language      string
	Archived      bool
	DefaultBranch string
	Stars         int64
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// CreateLinkParams holds the inputs needed to create a new link row.
type CreateLinkParams struct {
	NodeID     string
	DatabaseID int64
	FullName   string
	URL        string
	FreshTTL   int // seconds until first verification
	Metadata   Metadata
}

// CreateOrGet stores the link and alias atomically, reporting whether the link is new.
func (r *LinkRepository) CreateOrGet(ctx context.Context, p CreateLinkParams) (link models.Link, created bool, err error) {
	logger.Debug().Str("node_id", p.NodeID).Str("full_name", p.FullName).Msg("create or get link")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Link{}, false, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// The transaction takes the write lock first, so the lookup and insert are atomic.
	// The database id also matches a row whose node id GitHub has since migrated.
	var existing sql.NullInt64
	err = tx.QueryRowContext(ctx, `
        SELECT COALESCE(
            (SELECT id FROM links WHERE github_node_id = ?1),
            (SELECT link_id FROM github_node_id_aliases WHERE node_id = ?1),
            (SELECT id FROM links WHERE github_database_id = ?2 ORDER BY id LIMIT 1))`,
		p.NodeID, p.DatabaseID).Scan(&existing)
	if err != nil {
		return models.Link{}, false, fmt.Errorf("find existing: %w", err)
	}
	id := existing.Int64
	if !existing.Valid {
		if err = tx.QueryRowContext(ctx, `
            INSERT INTO links (
                github_node_id, github_database_id,
                initial_full_name, current_full_name, current_url,
                verify_after, description, language, archived, default_branch, stars
            ) VALUES (
                ?1, ?2, ?3, ?3, ?4,
                strftime('%Y-%m-%d %H:%M:%f', 'now', ?5 || ' seconds'), ?6, ?7, ?8, ?9, ?10
            )
            RETURNING id`,
			p.NodeID, p.DatabaseID, p.FullName, p.URL, p.FreshTTL,
			nullable(p.Metadata.Description), nullable(p.Metadata.Language), p.Metadata.Archived, nullable(p.Metadata.DefaultBranch),
			p.Metadata.Stars,
		).Scan(&id); err != nil {
			return models.Link{}, false, fmt.Errorf("insert: %w", err)
		}
		created = true
	}

	if _, err = tx.ExecContext(ctx, upsertAliasSQL, p.NodeID, id); err != nil {
		return models.Link{}, false, fmt.Errorf("upsert alias: %w", err)
	}
	if link, err = getByID(ctx, tx, id); err != nil {
		return models.Link{}, false, wrapErr("read link", err)
	}
	if err = tx.Commit(); err != nil {
		return models.Link{}, false, fmt.Errorf("commit: %w", err)
	}

	logger.Debug().Int64("id", link.ID).Bool("created", created).Msg("link ready")
	return link, created, nil
}

// GetByID fetches a link by primary id.
func (r *LinkRepository) GetByID(ctx context.Context, id int64) (models.Link, error) {
	return getByID(ctx, r.db, id)
}

// GetByNodeID resolves the current GitHub node ID or a recorded alias.
func (r *LinkRepository) GetByNodeID(ctx context.Context, nodeID string) (models.Link, error) {
	l, err := scanLink(r.db.QueryRowContext(ctx, linkSelect+`
        WHERE l.id = COALESCE((SELECT id FROM links WHERE github_node_id = ?1),
            (SELECT link_id FROM github_node_id_aliases WHERE node_id = ?1))`, nodeID))
	return l, mapErr(err)
}

// GetByCurrentFullName finds a repository name without regard to case.
func (r *LinkRepository) GetByCurrentFullName(ctx context.Context, fullName string) (models.Link, error) {
	l, err := scanLink(r.db.QueryRowContext(ctx,
		linkSelect+` WHERE LOWER(l.current_full_name) = LOWER(?1) ORDER BY (l.state = 'active') DESC, l.id DESC LIMIT 1`, fullName))
	return l, mapErr(err)
}

// VerificationUpdate carries fresh GitHub identity facts.
type VerificationUpdate struct {
	NodeID      string
	FullName    string
	URL         string
	FreshTTL    int // seconds until next verify_after
	JitterRange int // seconds of additional random jitter
	Metadata    Metadata
}

// ApplySuccessfulVerification refreshes repository facts and records changed node IDs as aliases.
func (r *LinkRepository) ApplySuccessfulVerification(ctx context.Context, id int64, u VerificationUpdate) (models.Link, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Link{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := getByID(ctx, tx, id)
	if err != nil {
		return models.Link{}, wrapErr("lock link", err)
	}

	previousNodeID := current.GithubNodeID
	if u.NodeID != "" && u.NodeID != previousNodeID {
		if _, err := tx.ExecContext(ctx, `UPDATE links SET github_node_id = ?2 WHERE id = ?1`, id, u.NodeID); err != nil {
			return models.Link{}, fmt.Errorf("rotate node_id: %w", err)
		}
		if _, err := tx.ExecContext(ctx, upsertAliasSQL, u.NodeID, id); err != nil {
			return models.Link{}, fmt.Errorf("upsert alias new: %w", err)
		}
		logger.Info().Int64("link_id", id).Str("old_node_id", previousNodeID).Str("new_node_id", u.NodeID).
			Msg("github node id rotated")
	}
	if _, err := tx.ExecContext(ctx, upsertAliasSQL, previousNodeID, id); err != nil {
		return models.Link{}, fmt.Errorf("touch alias: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
        UPDATE links SET
            current_full_name         = ?2,
            current_url               = ?3,
            last_successful_full_name = ?2,
            last_successful_url       = ?3,
            state                     = 'active',
            state_reason              = NULL,
            state_changed_at          = CASE WHEN state <> 'active' THEN strftime('%Y-%m-%d %H:%M:%f', 'now') ELSE state_changed_at END,
            policy_check_after        = CASE WHEN state <> 'active' THEN strftime('%Y-%m-%d %H:%M:%f', 'now') ELSE policy_check_after END,
            policy_checked_at         = CASE WHEN state <> 'active' THEN NULL ELSE policy_checked_at END,
            verified_at               = strftime('%Y-%m-%d %H:%M:%f', 'now'),
            verify_after              = strftime('%Y-%m-%d %H:%M:%f', 'now', ?4 || ' seconds'),
            fail_count                = 0,
            last_error                = NULL,
            description               = ?5,
            language                  = ?6,
            archived                  = ?7,
            default_branch            = ?8,
            stars                     = ?9
        WHERE id = ?1`,
		id, u.FullName, u.URL, u.FreshTTL+rand.IntN(max(u.JitterRange, 1)),
		nullable(u.Metadata.Description), nullable(u.Metadata.Language), u.Metadata.Archived, nullable(u.Metadata.DefaultBranch),
		u.Metadata.Stars,
	); err != nil {
		return models.Link{}, fmt.Errorf("update link: %w", err)
	}
	updated, err := getByID(ctx, tx, id)
	if err != nil {
		return models.Link{}, wrapErr("read link", err)
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
	if _, err := r.db.ExecContext(ctx, `
        UPDATE links SET
            fail_count   = fail_count + 1,
            last_error   = ?2,
            verify_after = strftime('%Y-%m-%d %H:%M:%f', 'now', ?3 || ' seconds')
        WHERE id = ?1`, id, f.Message, f.BackoffSecs); err != nil {
		return models.Link{}, fmt.Errorf("record failure: %w", err)
	}
	return r.GetByID(ctx, id)
}

// TombstoneUpdate carries an availability loss.
type TombstoneUpdate struct {
	State   models.State
	Reason  string
	Backoff int // seconds before we re-check
}

// ApplyTombstone marks a link unavailable; the row and slug stay so the repository can recover.
func (r *LinkRepository) ApplyTombstone(ctx context.Context, id int64, t TombstoneUpdate) (models.Link, error) {
	if _, err := r.db.ExecContext(ctx, `
        UPDATE links SET
            state            = ?2,
            state_reason     = ?3,
            state_changed_at = CASE WHEN state <> ?2 THEN strftime('%Y-%m-%d %H:%M:%f', 'now') ELSE state_changed_at END,
            verify_after     = strftime('%Y-%m-%d %H:%M:%f', 'now', ?4 || ' seconds'),
            last_error       = ?3
        WHERE id = ?1`, id, t.State, t.Reason, t.Backoff); err != nil {
		return models.Link{}, fmt.Errorf("tombstone: %w", err)
	}
	return r.GetByID(ctx, id)
}

// ClaimStale leases due links of any state so other workers skip them until the lease expires.
func (r *LinkRepository) ClaimStale(ctx context.Context, limit, leaseSecs int) ([]models.Link, error) {
	return r.claim(ctx, `
        UPDATE links
        SET verify_after = strftime('%Y-%m-%d %H:%M:%f', 'now', ?2 || ' seconds')
        WHERE id IN (
            SELECT id FROM links
            WHERE verify_after <= strftime('%Y-%m-%d %H:%M:%f', 'now')
            ORDER BY verify_after
            LIMIT CAST(?1 AS INTEGER)
        )
        RETURNING id`, limit, leaseSecs)
}

// ClaimPolicyDue leases active links whose owner policy is due for a check.
func (r *LinkRepository) ClaimPolicyDue(ctx context.Context, limit, leaseSecs int) ([]models.Link, error) {
	return r.claim(ctx, `
        UPDATE links
        SET policy_check_after = strftime('%Y-%m-%d %H:%M:%f', 'now', ?2 || ' seconds')
        WHERE id IN (
            SELECT id FROM links
            WHERE state = 'active' AND policy_check_after <= strftime('%Y-%m-%d %H:%M:%f', 'now')
            ORDER BY policy_check_after
            LIMIT CAST(?1 AS INTEGER)
        )
        RETURNING id`, limit, leaseSecs)
}

func (r *LinkRepository) claim(ctx context.Context, query string, args ...any) ([]models.Link, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("claim: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]models.Link, 0, len(ids))
	for _, id := range ids {
		l, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("read claimed %d: %w", id, err)
		}
		out = append(out, l)
	}
	return out, nil
}

// CountActive returns the number of links in active state.
func (r *LinkRepository) CountActive(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM links WHERE state = 'active'`).Scan(&n)
	return n, err
}
