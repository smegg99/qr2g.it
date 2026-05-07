// core/github/types.go

package github

import "time"

// Repo is the verified subset of GitHub repository identity facts used by
// qr2g.it. NodeID is the permanent (migratable) identity; DatabaseID is
// supporting metadata.
type Repo struct {
	NodeID     string
	DatabaseID int64
	FullName   string
	URL        string
	Private    bool
	Archived   bool
	Disabled   bool
}

// installationTokenResponse mirrors the JSON returned by the GitHub
// /app/installations/{id}/access_tokens endpoint.
type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
