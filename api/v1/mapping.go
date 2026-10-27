// api/v1/mapping.go

package api

import (
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"smegg.me/qr2g.it/api/v1/gen"
	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/repositories"
	"smegg.me/qr2g.it/core/slug"
)

func (s *Server) canonicalShortURL(sl string) string { return s.canonicalBaseURL + "/" + sl }

// displayAddress is the short address people read and type, without a scheme.
func (s *Server) displayAddress(sl string) string {
	host := s.publicBaseURL
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	return host + "/" + sl
}

func date(t time.Time) openapi_types.Date { return openapi_types.Date{Time: t.UTC()} }

func optString(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

func splitFullName(full string) (string, string) {
	owner, name, _ := strings.Cut(full, "/")
	return owner, name
}

func wireState(l models.Link) gen.LinkState {
	switch {
	case l.Moderation == models.ModerationRemove:
		return gen.LinkStateRemoved
	case l.State == models.StateActive:
		return gen.LinkStateActive
	case l.State == models.StatePrivate:
		return gen.LinkStatePrivate
	default:
		return gen.LinkStateUnavailable
	}
}

func wirePolicy(l models.Link) gen.Policy {
	p := gen.Policy{LastCheckFailed: l.PolicyFailed, CheckedAt: l.PolicyCheckedAt}
	switch {
	case l.Moderation != models.ModerationNone:
		p.Listing, p.Analytics = gen.PolicyListingRemoved, gen.PolicyAnalyticsRemoved
	case !l.IsActive():
		p.Listing, p.Analytics = gen.PolicyListingUnavailable, gen.PolicyAnalyticsUnavailable
	case l.PolicyCheckedAt == nil:
		p.Listing, p.Analytics = gen.PolicyListingPending, gen.PolicyAnalyticsPending
	default:
		p.Listing, p.Analytics = gen.PolicyListingListed, gen.PolicyAnalyticsCollecting
		if l.NoIndex {
			p.Listing = gen.PolicyListingOwnerNoindex
		}
		if l.NoAnalytics {
			p.Analytics = gen.PolicyAnalyticsOwnerNoanalytics
		}
	}
	return p
}

// wireLink keeps the shipped flat shape and publishes repository facts only while the link redirects.
func (s *Server) wireLink(l models.Link) gen.Link {
	sl := slug.MustEncode(l.ID)
	y, m, d := l.CreatedAt.UTC().Date()
	w := gen.Link{
		Id:        l.ID,
		Slug:      sl,
		Url:       s.canonicalShortURL(sl),
		State:     wireState(l),
		CreatedAt: time.Date(y, m, d, 0, 0, 0, 0, time.UTC),
		Policy:    wirePolicy(l),
	}
	if !l.PublishesDetails() {
		if l.Moderation != models.ModerationRemove {
			w.StateReason = optString(l.StateReason)
		}
		return w
	}
	verified, after := l.VerifiedAt.UTC(), l.VerifyAfter.UTC()
	nodeID, databaseID, archived := l.GithubNodeID, l.GithubDatabaseID, l.Archived
	w.VerifiedAt, w.VerifyAfter = &verified, &after
	w.InitialFullName, w.CurrentFullName, w.CurrentUrl = &l.InitialFullName, &l.CurrentFullName, &l.CurrentURL
	w.GithubNodeId, w.GithubDatabaseId, w.Archived = &nodeID, &databaseID, &archived
	if !l.HidesDiscovery() {
		w.Description, w.Language, w.Stars = optString(l.Description), optString(l.Language), l.Stars
		if len(l.Languages) > 0 {
			languages := make([]gen.LanguageShare, len(l.Languages))
			for i, x := range l.Languages {
				languages[i] = gen.LanguageShare{Name: x.Name, Share: float32(x.Share)}
			}
			w.Languages = &languages
		}
	}
	return w
}

func (s *Server) exploreItem(r repositories.ExploreRow) gen.ExploreItem {
	sl := slug.MustEncode(r.LinkID)
	owner, name := splitFullName(r.FullName)
	created, _ := time.Parse(time.DateOnly, r.CreatedOn)
	item := gen.ExploreItem{
		Slug:         sl,
		Url:          s.canonicalShortURL(sl),
		Owner:        owner,
		Name:         name,
		FullName:     r.FullName,
		Archived:     r.Archived,
		FirstAddedOn: date(created),
		Description:  optString(&r.Description),
		Language:     optString(&r.Language),
	}
	if !r.NoAnalytics {
		item.Redirects7, item.Redirects30 = &r.Redirects7, &r.Redirects30
	}
	return item
}
