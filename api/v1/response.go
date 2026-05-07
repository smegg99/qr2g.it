// api/response.go

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/slug"
)

// LinkResponse is the JSON shape returned by API endpoints describing a link.
type LinkResponse struct {
	URL              string `json:"url"`
	Slug             string `json:"slug"`
	ID               int64  `json:"id"`
	InitialFullName  string `json:"initialFullName"`
	CurrentFullName  string `json:"currentFullName"`
	CurrentURL       string `json:"currentUrl"`
	State            string `json:"state"`
	StateReason      string `json:"stateReason,omitempty"`
	VerifiedAt       string `json:"verifiedAt,omitempty"`
	VerifyAfter      string `json:"verifyAfter,omitempty"`
	CreatedAt        string `json:"createdAt"`
	GithubNodeID     string `json:"githubNodeId"`
	GithubDatabaseID int64  `json:"githubDatabaseId"`
}

// ErrorResponse is the JSON shape of an error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (s *Server) buildLinkResponse(l models.Link) LinkResponse {
	sl, _ := slug.Encode(l.ID)
	return LinkResponse{
		URL:              s.canonicalShortURL(sl),
		Slug:             sl,
		ID:               l.ID,
		InitialFullName:  l.InitialFullName,
		CurrentFullName:  l.CurrentFullName,
		CurrentURL:       l.CurrentURL,
		State:            string(l.State),
		StateReason:      derefString(l.StateReason),
		VerifiedAt:       l.VerifiedAt.UTC().Format("2006-01-02T15:04:05Z"),
		VerifyAfter:      l.VerifyAfter.UTC().Format("2006-01-02T15:04:05Z"),
		CreatedAt:        l.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		GithubNodeID:     l.GithubNodeID,
		GithubDatabaseID: l.GithubDatabaseID,
	}
}

func (s *Server) canonicalShortURL(sl string) string {
	return s.canonicalBaseURL + "/" + sl
}

func writeJSONError(c *gin.Context, status int, code, msg string) {
	c.JSON(status, ErrorResponse{Error: code, Message: msg})
}

func writeRedirect(c *gin.Context, location string) {
	c.Writer.Header().Set("Location", location)
	c.Writer.Header().Set("Cache-Control", "no-store, private")
	c.Writer.Header().Set("Pragma", "no-cache")
	c.Writer.Header().Set("X-Robots-Tag", "noindex, nofollow")
	c.Writer.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
	c.Writer.WriteHeader(http.StatusFound)
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
