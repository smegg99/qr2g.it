// core/githuburl/githuburl.go

package githuburl

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	Host          = "github.com"
	maxOwnerLen   = 39
	maxRepoLen    = 100
	repoExtension = ".git"
)

var (
	ErrEmpty       = errors.New("github url: empty")
	ErrScheme      = errors.New("github url: scheme must be https")
	ErrHost        = errors.New("github url: host must be github.com")
	ErrUserInfo    = errors.New("github url: userinfo not allowed")
	ErrQuery       = errors.New("github url: query not allowed")
	ErrFragment    = errors.New("github url: fragment not allowed")
	ErrPort        = errors.New("github url: port not allowed")
	ErrPath        = errors.New("github url: path must be /owner/repo")
	ErrOwner       = errors.New("github url: invalid owner")
	ErrRepo        = errors.New("github url: invalid repo")
	ErrReservedSeg = errors.New("github url: reserved path segment")
)

// Repo is a parsed reference to a GitHub repository (without verification).
type Repo struct {
	Owner string
	Name  string
}

// FullName returns "owner/name".
func (r Repo) FullName() string { return r.Owner + "/" + r.Name }

// reservedOwnerSegments are GitHub reserved owner-level paths that look like
// usernames but route to GitHub features and never to a user/org repo.
var reservedOwnerSegments = map[string]struct{}{
	"about":         {},
	"account":       {},
	"admin":         {},
	"api":           {},
	"apps":          {},
	"assets":        {},
	"blog":          {},
	"business":      {},
	"contact":       {},
	"dashboard":     {},
	"discussions":   {},
	"enterprise":    {},
	"events":        {},
	"explore":       {},
	"features":      {},
	"gist":          {},
	"home":          {},
	"issues":        {},
	"login":         {},
	"logout":        {},
	"marketplace":   {},
	"new":           {},
	"notifications": {},
	"organizations": {},
	"pricing":       {},
	"pulls":         {},
	"raw":           {},
	"readme":        {},
	"search":        {},
	"security":      {},
	"sessions":      {},
	"settings":      {},
	"signup":        {},
	"site":          {},
	"sponsors":      {},
	"stars":         {},
	"support":       {},
	"trending":      {},
	"tree":          {},
	"users":         {},
}

// Parse strictly parses a public GitHub repository URL.
func Parse(raw string) (Repo, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Repo{}, ErrEmpty
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Repo{}, fmt.Errorf("github url: %w", err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return Repo{}, ErrScheme
	}
	if u.User != nil {
		return Repo{}, ErrUserInfo
	}
	if u.Port() != "" {
		return Repo{}, ErrPort
	}
	if !strings.EqualFold(u.Hostname(), Host) {
		return Repo{}, ErrHost
	}
	if u.RawQuery != "" {
		return Repo{}, ErrQuery
	}
	if u.Fragment != "" || u.RawFragment != "" {
		return Repo{}, ErrFragment
	}
	return parsePath(u.Path)
}

// ParseOwnerRepo validates a raw "owner/repo" pair (no scheme, no host).
func ParseOwnerRepo(owner, repo string) (Repo, error) {
	if !validOwner(owner) {
		return Repo{}, ErrOwner
	}
	if _, reserved := reservedOwnerSegments[strings.ToLower(owner)]; reserved {
		return Repo{}, ErrReservedSeg
	}
	repo = strings.TrimSuffix(repo, repoExtension)
	if !validRepo(repo) {
		return Repo{}, ErrRepo
	}
	return Repo{Owner: owner, Name: repo}, nil
}

func parsePath(path string) (Repo, error) {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return Repo{}, ErrPath
	}
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return Repo{}, ErrPath
	}
	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], repoExtension)
	if owner == "" || repo == "" {
		return Repo{}, ErrPath
	}
	return ParseOwnerRepo(owner, repo)
}

// validOwner enforces GitHub username/org constraints:
//   - 1..39 chars
//   - alphanumeric or single hyphen
//   - cannot start or end with a hyphen
//   - cannot contain consecutive hyphens
func validOwner(s string) bool {
	if len(s) == 0 || len(s) > maxOwnerLen {
		return false
	}
	if s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-'
		if !ok {
			return false
		}
		if c == '-' && i+1 < len(s) && s[i+1] == '-' {
			return false
		}
	}
	return true
}

// validRepo enforces GitHub repository name constraints:
//   - 1..100 chars
//   - alphanumeric, hyphen, underscore, period
//   - not "." or ".."
func validRepo(s string) bool {
	if len(s) == 0 || len(s) > maxRepoLen {
		return false
	}
	if s == "." || s == ".." {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.'
		if !ok {
			return false
		}
	}
	return true
}
