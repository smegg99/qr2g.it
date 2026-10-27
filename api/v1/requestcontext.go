// api/v1/requestcontext.go

package api

import (
	"context"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/api/v1/gen"
)

// requestScoped passes each operation the request's own context, not gin's pooled
// one that database/sql reads after return. Methods are explicit so new ones fail to compile.
type requestScoped struct {
	next gen.StrictServerInterface
}

// requestContext is the request's context behind a gin one.
func requestContext(ctx context.Context) context.Context {
	if c, ok := ctx.(*gin.Context); ok {
		return c.Request.Context()
	}
	return ctx
}

func (r requestScoped) GetAutomationUsage(ctx context.Context, request gen.GetAutomationUsageRequestObject) (gen.GetAutomationUsageResponseObject, error) {
	return r.next.GetAutomationUsage(requestContext(ctx), request)
}

func (r requestScoped) RevealContact(ctx context.Context, request gen.RevealContactRequestObject) (gen.RevealContactResponseObject, error) {
	return r.next.RevealContact(requestContext(ctx), request)
}

func (r requestScoped) ListTopRepositories(ctx context.Context, request gen.ListTopRepositoriesRequestObject) (gen.ListTopRepositoriesResponseObject, error) {
	return r.next.ListTopRepositories(requestContext(ctx), request)
}

func (r requestScoped) GetCoverage(ctx context.Context, request gen.GetCoverageRequestObject) (gen.GetCoverageResponseObject, error) {
	return r.next.GetCoverage(requestContext(ctx), request)
}

func (r requestScoped) ListLanguages(ctx context.Context, request gen.ListLanguagesRequestObject) (gen.ListLanguagesResponseObject, error) {
	return r.next.ListLanguages(requestContext(ctx), request)
}

func (r requestScoped) CreateLink(ctx context.Context, request gen.CreateLinkRequestObject) (gen.CreateLinkResponseObject, error) {
	return r.next.CreateLink(requestContext(ctx), request)
}

func (r requestScoped) CreateLinks(ctx context.Context, request gen.CreateLinksRequestObject) (gen.CreateLinksResponseObject, error) {
	return r.next.CreateLinks(requestContext(ctx), request)
}

func (r requestScoped) LookupLink(ctx context.Context, request gen.LookupLinkRequestObject) (gen.LookupLinkResponseObject, error) {
	return r.next.LookupLink(requestContext(ctx), request)
}

func (r requestScoped) GetLink(ctx context.Context, request gen.GetLinkRequestObject) (gen.GetLinkResponseObject, error) {
	return r.next.GetLink(requestContext(ctx), request)
}

func (r requestScoped) RenderLinkAsset(ctx context.Context, request gen.RenderLinkAssetRequestObject) (gen.RenderLinkAssetResponseObject, error) {
	return r.next.RenderLinkAsset(requestContext(ctx), request)
}

func (r requestScoped) RefreshLinkPolicy(ctx context.Context, request gen.RefreshLinkPolicyRequestObject) (gen.RefreshLinkPolicyResponseObject, error) {
	return r.next.RefreshLinkPolicy(requestContext(ctx), request)
}

func (r requestScoped) ResolveLink(ctx context.Context, request gen.ResolveLinkRequestObject) (gen.ResolveLinkResponseObject, error) {
	return r.next.ResolveLink(requestContext(ctx), request)
}

func (r requestScoped) GetLinkStats(ctx context.Context, request gen.GetLinkStatsRequestObject) (gen.GetLinkStatsResponseObject, error) {
	return r.next.GetLinkStats(requestContext(ctx), request)
}

func (r requestScoped) ListRepositories(ctx context.Context, request gen.ListRepositoriesRequestObject) (gen.ListRepositoriesResponseObject, error) {
	return r.next.ListRepositories(requestContext(ctx), request)
}

func (r requestScoped) GetRepository(ctx context.Context, request gen.GetRepositoryRequestObject) (gen.GetRepositoryResponseObject, error) {
	return r.next.GetRepository(requestContext(ctx), request)
}
