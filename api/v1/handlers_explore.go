// api/v1/handlers_explore.go

package api

import (
	"context"
	"errors"
	"time"

	"github.com/smegg99/s99ab/api/apierr"

	"smegg.me/qr2g.it/api/v1/gen"
	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/repositories"
	"smegg.me/qr2g.it/core/slug"
)

const (
	exploreDefaultLimit = 24
	exploreMaxLimit     = 50
	topDefaultLimit     = 3
	topMaxLimit         = 10
)

// ListRepositories searches listed repositories.
func (h handlers) ListRepositories(ctx context.Context, req gen.ListRepositoriesRequestObject) (gen.ListRepositoriesResponseObject, error) {
	p := req.Params
	page := max(int(deref(p.Page)), 1)
	limit := deref(p.Limit)
	if limit <= 0 {
		limit = exploreDefaultLimit
	}
	limit = min(limit, exploreMaxLimit)
	sort := repositories.SortRecent
	switch deref(p.Sort) {
	case gen.Popular7:
		sort = repositories.SortPopular7
	case gen.Popular30:
		sort = repositories.SortPopular30
	case "", gen.Recent:
	default:
		return nil, fail(kindInvalidRequest)
	}
	rows, total, err := h.Stats.ListListed(ctx, repositories.ExploreQuery{
		Search:   string(deref(p.Q)),
		Language: deref(p.Language),
		Sort:     sort,
		Limit:    limit,
		Offset:   (page - 1) * limit,
	})
	if err != nil {
		return nil, failWrap(err, apierr.Internal)
	}
	out := gen.ListRepositories200JSONResponse{Total: total, Items: make([]gen.ExploreItem, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, h.exploreItem(r))
	}
	return out, nil
}

// loadListed reads a link that is eligible for public discovery.
func (s *Server) loadListed(ctx context.Context, raw string) (models.Link, error) {
	link, err := s.loadLink(ctx, raw)
	if err != nil {
		return models.Link{}, err
	}
	if !link.Listed() {
		return models.Link{}, fail(kindNotListed)
	}
	return link, nil
}

// listedRow builds the discovery row for one listed link.
func (s *Server) listedRow(ctx context.Context, link models.Link) (repositories.ExploreRow, error) {
	row := repositories.ExploreRow{
		LinkID:      link.ID,
		FullName:    link.CurrentFullName,
		Description: deref(link.Description),
		Language:    deref(link.Language),
		Archived:    link.Archived,
		CreatedOn:   link.CreatedAt.UTC().Format(time.DateOnly),
		NoAnalytics: link.NoAnalytics,
	}
	days, err := s.Stats.Daily(ctx, link.ID, dayOffset(time.Now(), -29))
	if err != nil {
		return row, err
	}
	week := dayOffset(time.Now(), -6)
	for _, d := range days {
		row.Redirects30 += d.Redirects
		if d.Day >= week {
			row.Redirects7 += d.Redirects
		}
	}
	return row, nil
}

// GetRepository reads one listed repository.
func (h handlers) GetRepository(ctx context.Context, req gen.GetRepositoryRequestObject) (gen.GetRepositoryResponseObject, error) {
	link, err := h.loadListed(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	row, err := h.listedRow(ctx, link)
	if err != nil {
		return nil, failWrap(err, apierr.Internal)
	}
	analytics := gen.RepositoryDetailAnalyticsCollecting
	if link.NoAnalytics {
		analytics = gen.RepositoryDetailAnalyticsOwnerNoanalytics
	}
	verified := link.VerifiedAt.UTC()
	return gen.GetRepository200JSONResponse{
		Item:            h.exploreItem(row),
		InitialFullName: link.InitialFullName,
		VerifiedAt:      &verified,
		Analytics:       analytics,
	}, nil
}

// ListLanguages counts listed repositories per language.
func (h handlers) ListLanguages(ctx context.Context, _ gen.ListLanguagesRequestObject) (gen.ListLanguagesResponseObject, error) {
	langs, err := h.Stats.Languages(ctx)
	if err != nil {
		return nil, failWrap(err, apierr.Internal)
	}
	out := gen.ListLanguages200JSONResponse{Items: make([]gen.LanguageCount, 0, len(langs))}
	for _, l := range langs {
		out.Items = append(out.Items, gen.LanguageCount{Count: l.Count, Name: l.Name})
	}
	return out, nil
}

func dayOffset(now time.Time, days int) string {
	return now.UTC().AddDate(0, 0, days).Format(time.DateOnly)
}

// ListTopRepositories ranks listed repositories by their recent filtered redirects.
func (h handlers) ListTopRepositories(ctx context.Context, req gen.ListTopRepositoriesRequestObject) (gen.ListTopRepositoriesResponseObject, error) {
	limit := deref(req.Params.Limit)
	if limit <= 0 {
		limit = topDefaultLimit
	}
	limit = min(limit, topMaxLimit)
	rows, err := h.Stats.Top(ctx, limit)
	if err != nil {
		return nil, failWrap(err, apierr.Internal)
	}
	out := gen.ListTopRepositories200JSONResponse{Items: make([]gen.TopRepository, 0, len(rows))}
	for _, r := range rows {
		w, err := h.statsWindow(ctx, r.LinkID)
		if err != nil {
			return nil, failWrap(err, apierr.Internal)
		}
		sl := slug.MustEncode(r.LinkID)
		item := gen.TopRepository{
			Slug:     sl,
			Url:      h.canonicalShortURL(sl),
			FullName: r.FullName,
			Total:    r.Total,
			Total30:  w.total30,
			Days:     w.days,
		}
		if r.Description != "" {
			item.Description = &r.Description
		}
		if r.Language != "" {
			item.Language = &r.Language
		}
		if r.LastAt != nil {
			at := r.LastAt.UTC()
			item.LastRedirectAt = &at
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

// statsWindow is a link's last 30 UTC days of filtered redirects, without days outside collection.
type statsWindow struct {
	started         string
	days            []gen.StatsDay
	pauses          []repositories.Pause
	total7, total30 int
}

func (s *Server) statsWindow(ctx context.Context, linkID int64) (statsWindow, error) {
	started, err := s.Stats.CollectionStartedOn(ctx)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return statsWindow{}, err
	}
	now := time.Now().UTC()
	from := max(dayOffset(now, -29), started)
	stored, err := s.Stats.Daily(ctx, linkID, from)
	if err != nil {
		return statsWindow{}, err
	}
	pauses, err := s.Stats.Pauses(ctx, linkID)
	if err != nil {
		return statsWindow{}, err
	}
	counts := make(map[string]int, len(stored))
	for _, d := range stored {
		counts[d.Day] = d.Redirects
	}
	w := statsWindow{started: started, days: []gen.StatsDay{}, pauses: pauses}
	week := dayOffset(now, -6)
	for day := from; day <= now.Format(time.DateOnly); day = nextDay(day) {
		if paused(day, pauses) {
			continue
		}
		n := counts[day]
		w.total30 += n
		if day >= week {
			w.total7 += n
		}
		t, _ := time.Parse(time.DateOnly, day)
		w.days = append(w.days, gen.StatsDay{Date: date(t), Redirects: n})
	}
	return w, nil
}

// GetLinkStats returns 30 days of filtered redirects for a listed repository.
func (h handlers) GetLinkStats(ctx context.Context, req gen.GetLinkStatsRequestObject) (gen.GetLinkStatsResponseObject, error) {
	link, err := h.loadListed(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	w, err := h.statsWindow(ctx, link.ID)
	if err != nil {
		return nil, failWrap(err, apierr.Internal)
	}
	total, last, err := h.Stats.Totals(ctx, link.ID)
	if err != nil {
		return nil, failWrap(err, apierr.Internal)
	}
	if last != nil {
		at := last.UTC()
		last = &at
	}
	startedOn, _ := time.Parse(time.DateOnly, w.started)
	out := gen.GetLinkStats200JSONResponse{
		Total:               total,
		LastRedirectAt:      last,
		Slug:                req.Slug,
		Metric:              gen.FilteredRedirects,
		CollectionStartedOn: date(startedOn),
		Collecting:          !link.NoAnalytics,
		Total7:              w.total7,
		Total30:             w.total30,
		Days:                w.days,
		Pauses:              []gen.StatsPause{},
	}
	for _, p := range w.pauses {
		fromDay, _ := time.Parse(time.DateOnly, p.StartedOn)
		pause := gen.StatsPause{From: date(fromDay), Reason: gen.StatsPauseReason(p.Reason)}
		if p.EndedOn != "" {
			to, _ := time.Parse(time.DateOnly, p.EndedOn)
			d := date(to)
			pause.To = &d
		}
		out.Pauses = append(out.Pauses, pause)
	}
	return out, nil
}

func nextDay(day string) string {
	t, _ := time.Parse(time.DateOnly, day)
	return t.AddDate(0, 0, 1).Format(time.DateOnly)
}

// paused reports whether collection was off for the whole day; start and end days are partial.
func paused(day string, pauses []repositories.Pause) bool {
	for _, p := range pauses {
		if day > p.StartedOn && (p.EndedOn == "" || day < p.EndedOn) {
			return true
		}
	}
	return false
}
