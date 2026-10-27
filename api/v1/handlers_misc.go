// api/v1/handlers_misc.go

package api

import (
	"context"
	"errors"
	"time"

	"github.com/smegg99/s99ab/api/apierr"

	"smegg.me/qr2g.it/api/v1/gen"
	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/automation"
	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/policy"
	"smegg.me/qr2g.it/core/turnstile"
)

// RefreshLinkPolicy re-reads availability and owner markers now, at most once per cooldown.
func (h handlers) RefreshLinkPolicy(ctx context.Context, req gen.RefreshLinkPolicyRequestObject) (gen.RefreshLinkPolicyResponseObject, error) {
	link, err := h.loadLink(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if link.Moderation == models.ModerationRemove {
		return nil, fail(kindLinkUnavailable)
	}
	updated, err := h.Policy.Refresh(ctx, link)
	var cooldown *policy.CooldownError
	switch {
	case errors.As(err, &cooldown):
		return nil, limited(kindRateLimited, cooldown.RetryAfter)
	case err != nil && updated.IsActive():
		return nil, failWrap(err, kindGithubUnavailable)
	case !updated.Redirects():
		return nil, fail(kindLinkUnavailable)
	}
	return gen.RefreshLinkPolicy200JSONResponse(h.wireLink(updated)), nil
}

// RevealContact returns the main address after a verified check and the catch-all otherwise.
func (h handlers) RevealContact(ctx context.Context, req gen.RevealContactRequestObject) (gen.RevealContactResponseObject, error) {
	token := ""
	if req.Body != nil && req.Body.TurnstileToken != nil {
		token = *req.Body.TurnstileToken
	}
	email := h.contactCatchall
	if !h.Turnstile.IsEnabled() {
		email = h.contactEmail
	} else if err := h.Turnstile.Verify(ctx, token, clientIPFrom(ctx), turnstile.ActionContact); err == nil {
		email = h.contactEmail
	} else {
		// A failed check is not proof of a bot; the catch-all still reaches a person.
		logger.Debug().Err(err).Msg("contact check not verified; revealing catch-all")
	}
	if email == "" {
		logger.Warn().Msg("contact address not configured")
		return nil, fail(kindContactUnavailable)
	}
	return gen.RevealContact200JSONResponse{Email: email}, nil
}

// GetAutomationUsage reports the token's daily quota without charging it.
func (h handlers) GetAutomationUsage(ctx context.Context, _ gen.GetAutomationUsageRequestObject) (gen.GetAutomationUsageResponseObject, error) {
	token, ok := tokenFrom(ctx)
	if !ok {
		return nil, fail(kindTokenRequired)
	}
	used, err := h.Automation.UsedToday(ctx, token)
	if err != nil {
		return nil, failWrap(err, apierr.Internal)
	}
	return gen.GetAutomationUsage200JSONResponse{
		Name:       token.Name,
		DailyQuota: token.DailyQuota,
		UsedToday:  used,
		ResetsAt:   automation.NextReset(time.Now()),
	}, nil
}
