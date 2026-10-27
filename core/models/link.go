// core/models/link.go

package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// State identifies the repository availability upstream on GitHub.
type State string

const (
	StateActive      State = "active"
	StatePrivate     State = "private"
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

// Moderation is an operator decision recorded after a report.
type Moderation string

const (
	ModerationNone   Moderation = ""
	ModerationUnlist Moderation = "unlist"
	ModerationRemove Moderation = "remove"
)

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

	Description   *string
	Language      *string
	Archived      bool
	DefaultBranch *string
	// Stars is GitHub's star count at the last successful read; nil before the first one.
	Stars *int64
	// Languages are the largest languages by GitHub's byte count, largest first.
	Languages []LanguageShare

	NoIndex          bool
	NoAnalytics      bool
	PolicyCheckedAt  *time.Time
	PolicyFailed     bool
	PolicyCheckAfter time.Time

	Moderation Moderation
}

// LanguageShare is one language's part of a repository's code, between 0 and 1.
type LanguageShare struct {
	Name  string  `json:"name"`
	Share float64 `json:"share"`
}

// IsActive reports whether the link is in active state.
func (l Link) IsActive() bool { return l.State == StateActive }

// IsFresh reports whether verify_after is still in the future.
func (l Link) IsFresh(now time.Time) bool { return now.Before(l.VerifyAfter) }

// Redirects reports whether the public redirect may send visitors to GitHub.
func (l Link) Redirects() bool { return l.IsActive() && l.Moderation != ModerationRemove }

// PublishesDetails reports whether cached repository details may be shown at all.
func (l Link) PublishesDetails() bool { return l.Redirects() }

// HidesDiscovery reports whether the owner or the operator withheld descriptive metadata.
func (l Link) HidesDiscovery() bool { return l.NoIndex || l.Moderation != ModerationNone }

// Listed reports whether the repository appears on public discovery surfaces.
func (l Link) Listed() bool {
	return l.Redirects() && l.Moderation == ModerationNone && l.PolicyCheckedAt != nil && !l.NoIndex
}

// Collects reports whether a filtered redirect may be counted now.
func (l Link) Collects() bool {
	return l.Redirects() && l.PolicyCheckedAt != nil && !l.NoAnalytics
}
