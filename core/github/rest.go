// core/github/rest.go

package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ClientConfig is the static configuration for both the REST and GraphQL
// client surfaces.
type ClientConfig struct {
	UserAgent      string
	APIVersion     string
	RESTBaseURL    string
	GraphQLURL     string
	RequestTimeout time.Duration
	UseAppAuth     bool
	UseGraphQL     bool
}

// Client makes authenticated calls to the GitHub REST and GraphQL APIs.
// When UseAppAuth is true and the App is configured, it sends an
// installation token as a Bearer credential. Otherwise it falls back to
// unauthenticated requests against public endpoints.
type Client struct {
	cfg ClientConfig
	hc  *http.Client
	app *app
}

// newClient is the package-internal constructor. External callers use
// Initialize + Get.
func newClient(cfg ClientConfig, a *app) *Client {
	hc := &http.Client{Timeout: cfg.RequestTimeout}
	return &Client{cfg: cfg, hc: hc, app: a}
}

// UsesAppAuth reports whether the client will send an installation token.
func (c *Client) UsesAppAuth() bool {
	return c.cfg.UseAppAuth && c.app != nil && c.app.IsConfigured()
}

// UsesGraphQL reports whether the client may use GraphQL for verification.
// GraphQL requires app auth.
func (c *Client) UsesGraphQL() bool {
	return c.cfg.UseGraphQL && c.UsesAppAuth()
}

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
	NodeID   string `json:"node_id"`
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
	Private  bool   `json:"private"`
	Archived bool   `json:"archived"`
	Disabled bool   `json:"disabled"`
	Message  string `json:"message,omitempty"`
}

// LookupRepoREST resolves a repo by current owner/name via the REST API.
func (c *Client) LookupRepoREST(ctx context.Context, owner, repo string) (Repo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s",
		strings.TrimRight(c.cfg.RESTBaseURL, "/"), owner, repo)
	return c.lookupREST(ctx, url)
}

// LookupRepoByDatabaseID resolves a repo by its GitHub database id. Used as
// the verification fallback when GitHub App auth is unavailable.
func (c *Client) LookupRepoByDatabaseID(ctx context.Context, id int64) (Repo, error) {
	url := fmt.Sprintf("%s/repositories/%d",
		strings.TrimRight(c.cfg.RESTBaseURL, "/"), id)
	return c.lookupREST(ctx, url)
}

func (c *Client) lookupREST(ctx context.Context, url string) (Repo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Repo{}, fmt.Errorf("build rest req: %w", err)
	}
	if err := c.applyAuthAndHeaders(ctx, req); err != nil {
		return Repo{}, err
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return Repo{}, fmt.Errorf("%w: %v", ErrTransient, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))

	switch resp.StatusCode {
	case http.StatusOK:
		var r restRepository
		if err := json.Unmarshal(body, &r); err != nil {
			return Repo{}, fmt.Errorf("%w: rest decode: %v", ErrInvalidResponse, err)
		}
		if r.NodeID == "" {
			return Repo{}, fmt.Errorf("%w: missing node_id", ErrInvalidResponse)
		}
		out := Repo{
			NodeID:     r.NodeID,
			DatabaseID: r.ID,
			FullName:   r.FullName,
			URL:        r.HTMLURL,
			Private:    r.Private,
			Archived:   r.Archived,
			Disabled:   r.Disabled,
		}
		if out.Private {
			return out, ErrPrivate
		}
		return out, nil

	case http.StatusNotFound:
		return Repo{}, ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		if isRateLimit(resp) {
			return Repo{}, ErrRateLimited
		}
		if c.app != nil {
			c.app.Reset()
		}
		return Repo{}, fmt.Errorf("%w: http %d", ErrAuth, resp.StatusCode)
	case http.StatusTooManyRequests:
		return Repo{}, ErrRateLimited
	default:
		if resp.StatusCode >= 500 {
			return Repo{}, fmt.Errorf("%w: http %d", ErrTransient, resp.StatusCode)
		}
		return Repo{}, fmt.Errorf("%w: http %d", ErrUnavailable, resp.StatusCode)
	}
}

func (c *Client) applyAuthAndHeaders(ctx context.Context, req *http.Request) error {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", c.cfg.APIVersion)
	req.Header.Set("X-Github-Next-Global-ID", "1")
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	if c.cfg.UseAppAuth && c.app != nil && c.app.IsConfigured() {
		tok, err := c.app.Token(ctx)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrAuth, err)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	return nil
}

func isRateLimit(resp *http.Response) bool {
	if resp.Header.Get("X-RateLimit-Remaining") == "0" {
		return true
	}
	if resp.Header.Get("Retry-After") != "" {
		return true
	}
	return false
}
