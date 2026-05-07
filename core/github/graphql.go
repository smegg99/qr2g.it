// core/github/graphql.go

package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const resolveRepoQuery = `query ResolveRepo($id: ID!) {
  node(id: $id) {
    __typename
    ... on Repository {
      id
      databaseId
      nameWithOwner
      url
      isPrivate
      isArchived
      isDisabled
    }
  }
}`

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphqlError struct {
	Message string         `json:"message"`
	Type    string         `json:"type,omitempty"`
	Path    []any          `json:"path,omitempty"`
	Extras  map[string]any `json:"extensions,omitempty"`
}

type graphqlResolveRepoResponse struct {
	Data struct {
		Node *struct {
			TypeName      string `json:"__typename"`
			ID            string `json:"id"`
			DatabaseID    int64  `json:"databaseId"`
			NameWithOwner string `json:"nameWithOwner"`
			URL           string `json:"url"`
			IsPrivate     bool   `json:"isPrivate"`
			IsArchived    bool   `json:"isArchived"`
			IsDisabled    bool   `json:"isDisabled"`
		} `json:"node"`
	} `json:"data"`
	Errors []graphqlError `json:"errors,omitempty"`
}

// ResolveRepoByNodeID strictly verifies a repository identity by node id
// using GraphQL. The Repo it returns is the canonical, currently-true
// identity. Mapped error cases (private, missing) take priority over Repo.
func (c *Client) ResolveRepoByNodeID(ctx context.Context, nodeID string) (Repo, error) {
	if !c.cfg.UseAppAuth || c.app == nil || !c.app.IsConfigured() {
		return Repo{}, fmt.Errorf("%w: graphql requires app auth", ErrAuth)
	}

	body, err := json.Marshal(graphqlRequest{
		Query:     resolveRepoQuery,
		Variables: map[string]any{"id": nodeID},
	})
	if err != nil {
		return Repo{}, fmt.Errorf("graphql encode: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.GraphQLURL, bytes.NewReader(body))
	if err != nil {
		return Repo{}, fmt.Errorf("graphql req: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err := c.applyAuthAndHeaders(ctx, req); err != nil {
		return Repo{}, err
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return Repo{}, fmt.Errorf("%w: %v", ErrTransient, err)
	}
	defer resp.Body.Close()
	rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		if isRateLimit(resp) {
			return Repo{}, ErrRateLimited
		}
		c.app.Reset()
		return Repo{}, fmt.Errorf("%w: http %d", ErrAuth, resp.StatusCode)
	case resp.StatusCode == http.StatusTooManyRequests:
		return Repo{}, ErrRateLimited
	case resp.StatusCode >= 500:
		return Repo{}, fmt.Errorf("%w: http %d", ErrTransient, resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return Repo{}, fmt.Errorf("%w: http %d", ErrUnavailable, resp.StatusCode)
	}

	var out graphqlResolveRepoResponse
	if err := json.Unmarshal(rawBody, &out); err != nil {
		return Repo{}, fmt.Errorf("%w: graphql decode: %v", ErrInvalidResponse, err)
	}
	if len(out.Errors) > 0 {
		// GitHub returns errors for permission denied (private repos), rate
		// limits, etc. We treat any presence here conservatively as transient.
		first := out.Errors[0]
		if first.Type == "NOT_FOUND" {
			return Repo{}, ErrNotFound
		}
		if first.Type == "RATE_LIMITED" {
			return Repo{}, ErrRateLimited
		}
		return Repo{}, fmt.Errorf("%w: graphql error: %s", ErrTransient, first.Message)
	}
	if out.Data.Node == nil {
		return Repo{}, ErrNotFound
	}
	if out.Data.Node.TypeName != "Repository" {
		return Repo{}, fmt.Errorf("%w: not a repository", ErrInvalidResponse)
	}

	r := Repo{
		NodeID:     out.Data.Node.ID,
		DatabaseID: out.Data.Node.DatabaseID,
		FullName:   out.Data.Node.NameWithOwner,
		URL:        out.Data.Node.URL,
		Private:    out.Data.Node.IsPrivate,
		Archived:   out.Data.Node.IsArchived,
		Disabled:   out.Data.Node.IsDisabled,
	}
	if r.Private {
		return r, ErrPrivate
	}
	return r, nil
}
