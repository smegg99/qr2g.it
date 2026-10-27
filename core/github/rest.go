// core/github/rest.go

package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ClientConfig is the static configuration for the REST client.
type ClientConfig struct {
	UserAgent      string
	APIVersion     string
	RESTBaseURL    string
	RequestTimeout time.Duration
	// Token authenticates public-data reads for a larger allowance; empty means anonymous.
	Token string
	// Reserve is the remaining allowance below which background work pauses.
	Reserve int
}

// Client makes calls to the public GitHub REST API.
type Client struct {
	cfg ClientConfig
	hc  *http.Client

	mu        sync.Mutex
	remaining int
	reset     time.Time
	known     bool
}

// newClient is the package-internal constructor. External callers use
// Initialize + Get.
func newClient(cfg ClientConfig) *Client {
	return &Client{cfg: cfg, hc: &http.Client{Timeout: cfg.RequestTimeout}}
}

// NewClient builds a client outside the global, for tests and tools.
func NewClient(cfg ClientConfig) *Client { return newClient(cfg) }

// Errors returned by the GitHub client. Callers map these to HTTP status
// codes and tombstone decisions.
var (
	ErrNotFound        = errors.New("github: repository not found")
	ErrPrivate         = errors.New("github: repository is private")
	ErrUnavailable     = errors.New("github: repository unavailable")
	ErrTransient       = errors.New("github: transient error")
	ErrRateLimited     = errors.New("github: rate limited")
	ErrAuth            = errors.New("github: authentication failed")
	ErrInvalidResponse = errors.New("github: invalid response")
)

// restRepository mirrors the subset of fields qr2g.it consumes from
// GET /repos/{owner}/{repo}.
type restRepository struct {
	NodeID        string `json:"node_id"`
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"`
	HTMLURL       string `json:"html_url"`
	Private       bool   `json:"private"`
	Archived      bool   `json:"archived"`
	Disabled      bool   `json:"disabled"`
	Description   string `json:"description"`
	Language      string `json:"language"`
	DefaultBranch string `json:"default_branch"`
	Stars         int64  `json:"stargazers_count"`
}

// LookupRepoREST resolves a repo by current owner/name via the REST API.
func (c *Client) LookupRepoREST(ctx context.Context, owner, repo string) (Repo, error) {
	return c.lookupREST(ctx, c.endpoint("repos", owner, repo))
}

// LookupRepoByDatabaseID resolves a repo by its GitHub database id.
func (c *Client) LookupRepoByDatabaseID(ctx context.Context, id int64) (Repo, error) {
	return c.lookupREST(ctx, c.endpoint("repositories", strconv.FormatInt(id, 10)))
}

func (c *Client) endpoint(segments ...string) string {
	escaped := make([]string, len(segments))
	for i, s := range segments {
		escaped[i] = url.PathEscape(s)
	}
	return strings.TrimRight(c.cfg.RESTBaseURL, "/") + "/" + strings.Join(escaped, "/")
}

func (c *Client) lookupREST(ctx context.Context, endpoint string) (Repo, error) {
	body, err := c.get(ctx, endpoint, "application/vnd.github+json", 256<<10)
	if err != nil {
		return Repo{}, err
	}
	var r restRepository
	if err := json.Unmarshal(body, &r); err != nil {
		return Repo{}, fmt.Errorf("%w: rest decode: %v", ErrInvalidResponse, err)
	}
	if r.NodeID == "" {
		return Repo{}, fmt.Errorf("%w: missing node_id", ErrInvalidResponse)
	}
	out := Repo{
		NodeID:        r.NodeID,
		DatabaseID:    r.ID,
		FullName:      r.FullName,
		URL:           r.HTMLURL,
		Private:       r.Private,
		Archived:      r.Archived,
		Disabled:      r.Disabled,
		Description:   strings.TrimSpace(r.Description),
		Language:      r.Language,
		DefaultBranch: r.DefaultBranch,
		Stars:         r.Stars,
	}
	if out.Private {
		return out, ErrPrivate
	}
	if out.Disabled {
		return out, ErrNotFound
	}
	return out, nil
}

// RootNames lists file names in the root of the repository's current default branch.
// Reading by database id keeps a rename or a new repository at the old name from answering.
func (c *Client) RootNames(ctx context.Context, databaseID int64) ([]string, error) {
	endpoint := c.endpoint("repositories", strconv.FormatInt(databaseID, 10), "contents") + "/"
	body, err := c.get(ctx, endpoint, "application/vnd.github+json", 1<<20)
	if err != nil {
		return nil, err
	}
	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("%w: contents decode: %v", ErrInvalidResponse, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Type == "file" {
			names = append(names, e.Name)
		}
	}
	return names, nil
}

// Readme returns the raw README of the current default branch, or "" when there is none.
func (c *Client) Readme(ctx context.Context, databaseID int64) (string, error) {
	endpoint := c.endpoint("repositories", strconv.FormatInt(databaseID, 10), "readme")
	body, err := c.get(ctx, endpoint, "application/vnd.github.raw+json", 1<<20)
	if errors.Is(err, ErrNotFound) {
		return "", nil
	}
	return string(body), err
}

// Languages returns bytes of code per language for the repository's default branch.
func (c *Client) Languages(ctx context.Context, databaseID int64) (map[string]int64, error) {
	body, err := c.get(ctx, c.endpoint("repositories", strconv.FormatInt(databaseID, 10), "languages"), "application/vnd.github+json", 64<<10)
	if err != nil {
		return nil, err
	}
	var out map[string]int64
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("%w: languages decode: %v", ErrInvalidResponse, err)
	}
	return out, nil
}

// PublicRepositoryCount is GitHub's search count of public repositories, forks included.
// GitHub stops counting early on a query this broad, so the number is a lower bound.
func (c *Client) PublicRepositoryCount(ctx context.Context) (int64, error) {
	endpoint := strings.TrimRight(c.cfg.RESTBaseURL, "/") + "/search/repositories?per_page=1&q=" + url.QueryEscape("is:public fork:true")
	body, err := c.get(ctx, endpoint, "application/vnd.github+json", 256<<10)
	if err != nil {
		return 0, err
	}
	var r struct {
		TotalCount int64 `json:"total_count"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, fmt.Errorf("%w: search decode: %v", ErrInvalidResponse, err)
	}
	if r.TotalCount <= 0 {
		return 0, fmt.Errorf("%w: empty repository count", ErrInvalidResponse)
	}
	return r.TotalCount, nil
}

func (c *Client) get(ctx context.Context, endpoint, accept string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build rest req: %w", err)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", c.cfg.APIVersion)
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransient, err)
	}
	defer resp.Body.Close()
	c.observe(resp)
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrTransient, err)
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		return body, nil
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrNotFound
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		if isRateLimit(resp) {
			return nil, ErrRateLimited
		}
		return nil, fmt.Errorf("%w: http %d", ErrAuth, resp.StatusCode)
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case resp.StatusCode >= 500:
		return nil, fmt.Errorf("%w: http %d", ErrTransient, resp.StatusCode)
	default:
		return nil, fmt.Errorf("%w: http %d", ErrUnavailable, resp.StatusCode)
	}
}

// observe records the primary rate limit GitHub reports; search has its own allowance and is ignored.
func (c *Client) observe(resp *http.Response) {
	if res := resp.Header.Get("X-RateLimit-Resource"); res != "" && res != "core" {
		return
	}
	remaining, err1 := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
	reset, err2 := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64)
	if err1 != nil || err2 != nil {
		return
	}
	c.mu.Lock()
	c.remaining, c.reset, c.known = remaining, time.Unix(reset, 0), true
	c.mu.Unlock()
}

// BackgroundAllowed reports whether background work may spend the remaining allowance.
func (c *Client) BackgroundAllowed(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.known || now.After(c.reset) || c.remaining > c.cfg.Reserve
}

// RetryAfter is how long until GitHub's reported allowance resets, or a minute when unknown.
func (c *Client) RetryAfter(now time.Time) time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.known && c.reset.After(now) {
		return c.reset.Sub(now)
	}
	return time.Minute
}

// Authenticated reports whether reads use a token.
func (c *Client) Authenticated() bool { return c.cfg.Token != "" }

func isRateLimit(resp *http.Response) bool {
	return resp.Header.Get("X-RateLimit-Remaining") == "0" || resp.Header.Get("Retry-After") != ""
}
