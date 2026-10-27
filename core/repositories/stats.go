// core/repositories/stats.go

package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// StatsRepository owns the redirect aggregates and public discovery queries.
type StatsRepository struct{ db *sql.DB }

// NewStatsRepository binds a StatsRepository to db.
func NewStatsRepository(db *sql.DB) *StatsRepository { return &StatsRepository{db: db} }

// DayKey identifies one link's aggregate for one UTC day (YYYY-MM-DD).
type DayKey struct {
	LinkID int64
	Day    string
}

// RedirectBatch is buffered redirect data: counts per link and day, and each link's latest redirect.
type RedirectBatch struct {
	Counts map[DayKey]int
	Latest map[int64]time.Time
}

// AddRedirects adds a batch in one transaction; each upsert is atomic per row.
func (r *StatsRepository) AddRedirects(ctx context.Context, b RedirectBatch) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for k, n := range b.Counts {
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO redirect_daily (link_id, day, redirects) VALUES (?1, ?2, ?3)
            ON CONFLICT (link_id, day) DO UPDATE SET redirects = redirects + excluded.redirects`,
			k.LinkID, k.Day, n); err != nil {
			return fmt.Errorf("add redirects %d %s: %w", k.LinkID, k.Day, err)
		}
	}
	for id, at := range b.Latest {
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO redirect_latest (link_id, at) VALUES (?1, ?2)
            ON CONFLICT (link_id) DO UPDATE SET at = max(at, excluded.at)`,
			id, at.UTC().Format(sqliteTime)); err != nil {
			return fmt.Errorf("latest redirect %d: %w", id, err)
		}
	}
	return tx.Commit()
}

// Totals is a link's filtered redirects since collection started and the time of the latest one.
func (r *StatsRepository) Totals(ctx context.Context, linkID int64) (int, *time.Time, error) {
	var total int
	var last sql.NullTime
	err := r.db.QueryRowContext(ctx, `
        SELECT COALESCE((SELECT sum(redirects) FROM redirect_daily WHERE link_id = ?1), 0),
            (SELECT at FROM redirect_latest WHERE link_id = ?1)`, linkID).Scan(&total, &last)
	if err != nil || !last.Valid {
		return total, nil, err
	}
	return total, &last.Time, nil
}

// sqliteTime matches strftime('%Y-%m-%d %H:%M:%f') so stored times compare as text in max().
const sqliteTime = "2006-01-02 15:04:05.000"

// TopRow is one listed repository ranked by its recent filtered redirects.
type TopRow struct {
	LinkID      int64
	FullName    string
	Description string
	Language    string
	Total       int
	Redirects30 int
	LastAt      *time.Time
}

// Top returns up to limit listed repositories that collect statistics and had redirects in the
// last 30 UTC days, most redirected first.
func (r *StatsRepository) Top(ctx context.Context, limit int) ([]TopRow, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT * FROM (
            SELECT l.id, l.current_full_name, COALESCE(l.description, ''), COALESCE(l.language, ''),
                COALESCE((SELECT sum(redirects) FROM redirect_daily d WHERE d.link_id = l.id), 0) AS total,
                COALESCE((SELECT sum(redirects) FROM redirect_daily d
                    WHERE d.link_id = l.id AND d.day >= strftime('%Y-%m-%d', 'now', '-29 days')), 0) AS r30,
                (SELECT at FROM redirect_latest x WHERE x.link_id = l.id) AS last_at
            FROM links l LEFT JOIN moderation m ON m.link_id = l.id
            WHERE `+listedWhere+` AND l.noanalytics = 0
        ) WHERE r30 > 0
        ORDER BY r30 DESC, total DESC, id
        LIMIT ?1`, limit)
	if err != nil {
		return nil, fmt.Errorf("top listed: %w", err)
	}
	defer rows.Close()
	var out []TopRow
	for rows.Next() {
		var t TopRow
		var last sql.NullTime
		if err := rows.Scan(&t.LinkID, &t.FullName, &t.Description, &t.Language, &t.Total, &t.Redirects30, &last); err != nil {
			return nil, err
		}
		if last.Valid {
			t.LastAt = &last.Time
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CollectionStartedOn returns the UTC day redirect collection began.
func (r *StatsRepository) CollectionStartedOn(ctx context.Context) (string, error) {
	var day string
	err := r.db.QueryRowContext(ctx,
		`SELECT value FROM service_facts WHERE key = 'redirect_collection_started_on'`).Scan(&day)
	return day, mapErr(err)
}

// DayCount is one stored aggregate.
type DayCount struct {
	Day       string
	Redirects int
}

// Daily returns stored days for a link on or after fromDay, oldest first.
func (r *StatsRepository) Daily(ctx context.Context, linkID int64, fromDay string) ([]DayCount, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT day, redirects FROM redirect_daily
        WHERE link_id = ?1 AND day >= ?2 ORDER BY day`, linkID, fromDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DayCount
	for rows.Next() {
		var d DayCount
		if err := rows.Scan(&d.Day, &d.Redirects); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Pause is a period without collection; EndedOn is empty while it continues.
type Pause struct {
	StartedOn string
	EndedOn   string
	Reason    string
}

// Pauses returns a link's collection pauses, oldest first.
func (r *StatsRepository) Pauses(ctx context.Context, linkID int64) ([]Pause, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT started_on, COALESCE(ended_on, ''), reason FROM analytics_pauses
        WHERE link_id = ?1 ORDER BY started_on, id`, linkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Pause
	for rows.Next() {
		var p Pause
		if err := rows.Scan(&p.StartedOn, &p.EndedOn, &p.Reason); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// listedWhere is the single eligibility rule for every public discovery surface.
const listedWhere = `l.state = 'active' AND m.link_id IS NULL AND l.policy_checked_at IS NOT NULL AND l.noindex = 0`

// ExploreSort orders listed repositories.
type ExploreSort string

const (
	SortRecent    ExploreSort = "recent"
	SortPopular7  ExploreSort = "popular7"
	SortPopular30 ExploreSort = "popular30"
)

// ExploreQuery filters listed repositories.
type ExploreQuery struct {
	Search   string
	Language string
	Sort     ExploreSort
	Limit    int
	Offset   int
}

// ExploreRow is one listed repository with its recent totals.
type ExploreRow struct {
	LinkID      int64
	FullName    string
	Description string
	Language    string
	Archived    bool
	CreatedOn   string
	NoAnalytics bool
	Redirects7  int
	Redirects30 int
}

// escapeLike makes user text literal inside a LIKE pattern using '\' as escape.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// ListListed returns one page of listed repositories and the total that match.
func (r *StatsRepository) ListListed(ctx context.Context, q ExploreQuery) ([]ExploreRow, int, error) {
	where := listedWhere
	args := []any{}
	if s := strings.ToLower(strings.TrimSpace(q.Search)); s != "" {
		args = append(args, "%"+escapeLike(s)+"%")
		where += fmt.Sprintf(` AND (LOWER(l.current_full_name) LIKE ?%[1]d ESCAPE '\' OR LOWER(COALESCE(l.description, '')) LIKE ?%[1]d ESCAPE '\')`, len(args))
	}
	if q.Language != "" {
		args = append(args, q.Language)
		where += fmt.Sprintf(` AND l.language = ?%d COLLATE NOCASE`, len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM links l LEFT JOIN moderation m ON m.link_id = l.id WHERE `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count listed: %w", err)
	}

	order := `l.created_at DESC, l.id DESC`
	switch q.Sort {
	case SortPopular7:
		order = `CASE WHEN l.noanalytics = 1 THEN 0 ELSE r7 END DESC, l.created_at DESC, l.id DESC`
	case SortPopular30:
		order = `CASE WHEN l.noanalytics = 1 THEN 0 ELSE r30 END DESC, l.created_at DESC, l.id DESC`
	}
	args = append(args, q.Limit, q.Offset)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
        SELECT l.id, l.current_full_name, COALESCE(l.description, ''), COALESCE(l.language, ''),
            l.archived, strftime('%%Y-%%m-%%d', l.created_at), l.noanalytics,
            COALESCE((SELECT sum(redirects) FROM redirect_daily d
                WHERE d.link_id = l.id AND d.day >= strftime('%%Y-%%m-%%d', 'now', '-6 days')), 0) AS r7,
            COALESCE((SELECT sum(redirects) FROM redirect_daily d
                WHERE d.link_id = l.id AND d.day >= strftime('%%Y-%%m-%%d', 'now', '-29 days')), 0) AS r30
        FROM links l LEFT JOIN moderation m ON m.link_id = l.id
        WHERE %s ORDER BY %s LIMIT ?%d OFFSET ?%d`, where, order, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list listed: %w", err)
	}
	defer rows.Close()
	var out []ExploreRow
	for rows.Next() {
		var e ExploreRow
		if err := rows.Scan(&e.LinkID, &e.FullName, &e.Description, &e.Language,
			&e.Archived, &e.CreatedOn, &e.NoAnalytics, &e.Redirects7, &e.Redirects30); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

// LanguageCount is one language among listed repositories.
type LanguageCount struct {
	Name  string
	Count int
}

// Languages counts listed repositories per primary language, most common first.
func (r *StatsRepository) Languages(ctx context.Context) ([]LanguageCount, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT l.language, count(*) FROM links l LEFT JOIN moderation m ON m.link_id = l.id
        WHERE `+listedWhere+` AND l.language IS NOT NULL AND l.language <> ''
        GROUP BY l.language ORDER BY count(*) DESC, l.language`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LanguageCount
	for rows.Next() {
		var lc LanguageCount
		if err := rows.Scan(&lc.Name, &lc.Count); err != nil {
			return nil, err
		}
		out = append(out, lc)
	}
	return out, rows.Err()
}
