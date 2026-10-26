// core/repositories/alias.go

package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/models"
)

// AliasRepository owns reads and writes against github_node_id_aliases.
type AliasRepository struct{ db *sql.DB }

// GetAliasRepository returns an AliasRepository bound to the global datastore.
func GetAliasRepository() *AliasRepository { return &AliasRepository{db: datastore.Get()} }

// upsertAliasSQL is shared with LinkRepository for transactional alias
// touches inside link writes.
const upsertAliasSQL = `
    INSERT INTO github_node_id_aliases (node_id, link_id)
    VALUES (?1, ?2)
    ON CONFLICT (node_id) DO UPDATE SET last_seen_at = strftime('%Y-%m-%d %H:%M:%f', 'now')`

// Upsert records that nodeID has been seen pointing at linkID.
func (r *AliasRepository) Upsert(ctx context.Context, nodeID string, linkID int64) error {
	if _, err := r.db.ExecContext(ctx, upsertAliasSQL, nodeID, linkID); err != nil {
		return fmt.Errorf("upsert alias %s: %w", nodeID, err)
	}
	return nil
}

// GetByNodeID returns the alias row for nodeID.
func (r *AliasRepository) GetByNodeID(ctx context.Context, nodeID string) (models.Alias, error) {
	var a models.Alias
	err := r.db.QueryRowContext(ctx, `
        SELECT node_id, link_id, first_seen_at, last_seen_at
        FROM github_node_id_aliases
        WHERE node_id = ?1
    `, nodeID).Scan(&a.NodeID, &a.LinkID, &a.FirstSeenAt, &a.LastSeenAt)
	return a, mapErr(err)
}
