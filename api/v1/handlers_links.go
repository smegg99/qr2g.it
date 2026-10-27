// api/v1/handlers_links.go

package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/smegg99/s99ab/api/apierr"

	"smegg.me/qr2g.it/api/v1/gen"
	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/githuburl"
	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/repositories"
	"smegg.me/qr2g.it/core/slug"
	"smegg.me/qr2g.it/core/turnstile"
	"smegg.me/qr2g.it/core/verifier"
)

// handlers implements the generated contract on top of the server's services.
type handlers struct{ *Server }

var _ gen.StrictServerInterface = handlers{}

// CreateLink verifies a repository and creates or reuses its canonical link.
func (h handlers) CreateLink(ctx context.Context, req gen.CreateLinkRequestObject) (gen.CreateLinkResponseObject, error) {
	if req.Body == nil {
		return nil, fail(kindInvalidBody)
	}
	repo, err := githuburl.Parse(req.Body.Url)
	if err != nil {
		return nil, fail(kindInvalidURL)
	}
	if _, ok := tokenFrom(ctx); !ok {
		ip := clientIPFrom(ctx)
		if ok, after := h.createDaily.Allow(ip); !ok {
			return nil, limited(kindRateLimited, after)
		}
		token := ""
		if req.Body.TurnstileToken != nil {
			token = *req.Body.TurnstileToken
		}
		if err := h.verifyTurnstile(ctx, token, ip, turnstile.ActionCreateLink); err != nil {
			return nil, err
		}
	}
	link, created, err := h.lookupAndStore(ctx, repo)
	if err != nil {
		return nil, err
	}
	if created {
		return gen.CreateLink201JSONResponse(h.wireLink(link)), nil
	}
	return gen.CreateLink200JSONResponse(h.wireLink(link)), nil
}

// CreateLinks processes up to 25 repositories independently for an automation token.
func (h handlers) CreateLinks(ctx context.Context, req gen.CreateLinksRequestObject) (gen.CreateLinksResponseObject, error) {
	if _, ok := tokenFrom(ctx); !ok {
		return nil, fail(kindTokenRequired)
	}
	if req.Body == nil || len(req.Body.Repositories) == 0 || len(req.Body.Repositories) > batchMax {
		return nil, fail(kindInvalidBody)
	}
	// The whole batch answers within the write timeout; an entry still waiting on GitHub fails as retryable.
	ctx, cancel := context.WithTimeout(ctx, batchDeadline)
	defer cancel()
	results := make([]gen.BatchItem, len(req.Body.Repositories))
	slots := make(chan struct{}, batchParallel)
	var wg sync.WaitGroup
	for i, input := range req.Body.Repositories {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			results[i] = h.batchItem(ctx, input)
		})
	}
	wg.Wait()
	return gen.CreateLinks200JSONResponse{Results: results}, nil
}

const (
	batchMax      = 25
	batchParallel = 4
	batchDeadline = 10 * time.Second
)

func (h handlers) batchItem(ctx context.Context, input string) gen.BatchItem {
	item := gen.BatchItem{Input: input, Outcome: gen.Failed}
	repo, err := parseRepository(input)
	if err == nil && ctx.Err() != nil {
		err = failWrap(ctx.Err(), kindGithubUnavailable)
	}
	if err == nil {
		var link models.Link
		var created bool
		link, created, err = h.lookupAndStore(ctx, repo)
		if err == nil {
			wire := h.wireLink(link)
			item.Link, item.Outcome = &wire, gen.Existing
			if created {
				item.Outcome = gen.Created
			}
			return item
		}
	}
	var e *apierr.Error
	if !errors.As(err, &e) {
		e = fail(apierr.Internal)
	}
	item.Error = &gen.Error{Code: e.Kind.Code, Message: e.Message}
	return item
}

// parseRepository accepts owner/name or a GitHub URL.
func parseRepository(input string) (githuburl.Repo, error) {
	input = strings.TrimSpace(input)
	if strings.Contains(input, "://") {
		if repo, err := githuburl.Parse(input); err == nil {
			return repo, nil
		}
		return githuburl.Repo{}, fail(kindInvalidURL)
	}
	owner, name, ok := strings.Cut(strings.TrimPrefix(input, "github.com/"), "/")
	if !ok {
		return githuburl.Repo{}, fail(kindInvalidRepository)
	}
	repo, err := githuburl.ParseOwnerRepo(owner, name)
	if err != nil {
		return githuburl.Repo{}, fail(kindInvalidRepository)
	}
	return repo, nil
}

// LookupLink reads an existing link by its current name without creating one.
func (h handlers) LookupLink(ctx context.Context, req gen.LookupLinkRequestObject) (gen.LookupLinkResponseObject, error) {
	repo, err := parseRepository(req.Params.Repository)
	if err != nil {
		return nil, fail(kindInvalidRepository)
	}
	link, err := h.Links.GetByCurrentFullName(ctx, repo.FullName())
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, fail(kindLinkNotFound)
	}
	if err != nil {
		return nil, failWrap(err, apierr.Internal)
	}
	if !link.PublishesDetails() {
		return nil, fail(kindLinkNotFound)
	}
	return gen.LookupLink200JSONResponse(h.wireLink(link)), nil
}

// GetLink returns stored link metadata without a GitHub call.
func (h handlers) GetLink(ctx context.Context, req gen.GetLinkRequestObject) (gen.GetLinkResponseObject, error) {
	link, err := h.loadLink(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.GetLink200JSONResponse(h.wireLink(link)), nil
}

// ResolveLink verifies a stale link and answers the redirect decision as JSON.
func (h handlers) ResolveLink(ctx context.Context, req gen.ResolveLinkRequestObject) (gen.ResolveLinkResponseObject, error) {
	link, err := h.loadLink(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	resolved, status, err := h.resolveOrVerify(ctx, link)
	if err != nil {
		return nil, failWrap(err, kindVerificationFailed)
	}
	if status == http.StatusGone {
		return gen.ResolveLink410JSONResponse(h.wireLink(resolved)), nil
	}
	return gen.ResolveLink200JSONResponse(h.wireLink(resolved)), nil
}

// loadLink decodes a slug and reads its link; unknown slugs also spend the miss budget.
func (s *Server) loadLink(ctx context.Context, raw string) (models.Link, error) {
	id, err := slug.Decode(raw)
	if err != nil {
		s.missBucket.Allow(clientIPFrom(ctx))
		return models.Link{}, fail(kindNotFound)
	}
	link, err := s.Links.GetByID(ctx, id)
	if errors.Is(err, repositories.ErrNotFound) {
		s.missBucket.Allow(clientIPFrom(ctx))
		return models.Link{}, fail(kindNotFound)
	}
	if err != nil {
		return models.Link{}, failWrap(err, apierr.Internal)
	}
	return link, nil
}

// lookupAndStore verifies the repository on GitHub, then creates or refreshes its link.
// languagesTimeout bounds the extra GitHub read for the language colours on a create.
const languagesTimeout = 3 * time.Second

func (s *Server) lookupAndStore(ctx context.Context, repo githuburl.Repo) (models.Link, bool, error) {
	gh, err := s.GitHub.LookupRepoREST(ctx, repo.Owner, repo.Name)
	if err != nil {
		// While GitHub cannot answer, a repository that already redirects keeps its stored link.
		if githubDown(err) {
			if link, gerr := s.Links.GetByCurrentFullName(ctx, repo.FullName()); gerr == nil && link.Redirects() {
				return link, false, nil
			}
		}
		return models.Link{}, false, s.lookupError(err)
	}
	fresh := int(s.Verifier.FreshTTL().Seconds())
	link, created, err := s.Links.CreateOrGet(ctx, repositories.CreateLinkParams{
		NodeID:     gh.NodeID,
		DatabaseID: gh.DatabaseID,
		FullName:   gh.FullName,
		URL:        gh.URL,
		FreshTTL:   fresh,
		Metadata:   verifier.Metadata(gh),
	})
	if err != nil {
		return models.Link{}, false, failWrap(err, apierr.Internal)
	}
	if !created {
		// The lookup just proved the repository public, so stored facts and state catch up.
		link, err = s.Links.ApplySuccessfulVerification(ctx, link.ID, repositories.VerificationUpdate{
			NodeID: gh.NodeID, FullName: gh.FullName, URL: gh.URL, FreshTTL: fresh, JitterRange: 1,
			Metadata: verifier.Metadata(gh),
		})
		if err != nil {
			return models.Link{}, false, failWrap(err, apierr.Internal)
		}
	}
	if link.Moderation == models.ModerationRemove {
		return models.Link{}, false, fail(kindLinkUnavailable)
	}
	if link.PolicyCheckedAt == nil && s.Policy.Enabled() {
		go s.firstPolicyCheck(link)
	}
	languages, cancel := context.WithTimeout(ctx, languagesTimeout)
	defer cancel()
	return s.Verifier.RefreshLanguages(languages, link), created, nil
}

// firstPolicyCheck reads the owner's markers soon after a link appears.
func (s *Server) firstPolicyCheck(link models.Link) {
	ctx, cancel := context.WithTimeout(s.base, time.Minute)
	defer cancel()
	if _, err := s.Policy.Check(ctx, link); err != nil {
		logger.Debug().Int64("link_id", link.ID).Err(err).Msg("first policy check failed; the scheduler retries")
	}
}

// verifyTurnstile checks a browser token when Turnstile is enabled.
func (s *Server) verifyTurnstile(ctx context.Context, token, ip, action string) error {
	if !s.Turnstile.IsEnabled() {
		return nil
	}
	return turnstileError(s.Turnstile.Verify(ctx, token, ip, action))
}

// resolveOrVerify decides whether a link redirects, verifying it with GitHub when stale.
// A link that stopped redirecting is rechecked on schedule so it can recover.
func (s *Server) resolveOrVerify(ctx context.Context, link models.Link) (models.Link, int, error) {
	if link.Moderation == models.ModerationRemove {
		return link, http.StatusGone, nil
	}
	if time.Now().Before(link.VerifyAfter) {
		if link.IsActive() {
			return link, http.StatusOK, nil
		}
		return link, http.StatusGone, nil
	}
	verified, err := s.Verifier.VerifyOnce(ctx, link)
	if !verified.IsActive() {
		return verified, http.StatusGone, nil
	}
	if err != nil {
		return link, 0, err
	}
	return verified, http.StatusOK, nil
}
