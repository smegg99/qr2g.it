// core/models/alias.go

package models

import "time"

// Alias maps a previously-canonical github_node_id back to the current link
// id. Used when GitHub migrates node ids so old identities continue to
// resolve to the same qr2g.it link.
type Alias struct {
	NodeID      string
	LinkID      int64
	FirstSeenAt time.Time
	LastSeenAt  time.Time
}
