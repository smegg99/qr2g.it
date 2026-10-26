// core/models/link.go

package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// State identifies the repository availability.
type State string

const (
	StateActive      State = "active"
	StatePrivate     State = "private"
	StateDeleted     State = "deleted"
	StateUnavailable State = "unavailable"
)

// Scan implements sql.Scanner.
func (s *State) Scan(src any) error {
	switch v := src.(type) {
	case string:
		*s = State(v)
	case []byte:
		*s = State(v)
	case nil:
		*s = ""
	default:
		return fmt.Errorf("models: cannot scan %T into State", src)
	}
	return nil
}

// Value implements driver.Valuer.
func (s State) Value() (driver.Value, error) { return string(s), nil }

// Link is the canonical row in the links table. The id is the slug source
// (uppercase base36(id) is the public slug) and the github_node_id is the
// permanent GitHub identity that survives renames and ownership transfers.
type Link struct {
	ID                     int64
	GithubNodeID           string
	GithubDatabaseID       int64
	InitialFullName        string
	CurrentFullName        string
	CurrentURL             string
	LastSuccessfulFullName *string
	LastSuccessfulURL      *string
	State                  State
	StateReason            *string
	StateChangedAt         *time.Time
	CreatedAt              time.Time
	VerifiedAt             time.Time
	VerifyAfter            time.Time
	FailCount              int32
	LastError              *string
}

// IsActive reports whether the link is in active state.
func (l Link) IsActive() bool { return l.State == StateActive }

// IsFresh reports whether verify_after is still in the future.
func (l Link) IsFresh(now time.Time) bool { return now.Before(l.VerifyAfter) }
