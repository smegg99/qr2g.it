// core/github/types.go

package github

// Repo is the verified subset of GitHub repository facts used by qr2g.it.
// NodeID is the permanent (migratable) identity; DatabaseID is supporting metadata.
type Repo struct {
	NodeID        string
	DatabaseID    int64
	FullName      string
	URL           string
	Private       bool
	Archived      bool
	Disabled      bool
	Description   string
	Language      string
	DefaultBranch string
	Stars         int64
}
