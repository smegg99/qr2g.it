// api/v1/handlers_assets.go

package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime"
	"regexp"
	"strings"

	"github.com/smegg99/s99ab/api/apierr"

	"smegg.me/qr2g.it/api/v1/gen"
	"smegg.me/qr2g.it/core/render"
	"smegg.me/qr2g.it/core/slug"
)

// RenderLinkAsset renders a preset with the same renderer the website previews.
func (h handlers) RenderLinkAsset(ctx context.Context, req gen.RenderLinkAssetRequestObject) (gen.RenderLinkAssetResponseObject, error) {
	link, err := h.loadLink(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if !link.Redirects() {
		return nil, fail(kindLinkUnavailable)
	}
	spec := assetSpec(req.Params)
	sl := slug.MustEncode(link.ID)
	owner, name := splitFullName(link.CurrentFullName)
	content := render.Content{
		URL:     h.canonicalShortURL(sl),
		Address: h.displayAddress(sl),
		Owner:   owner,
		Name:    name,
	}
	if link.Description != nil && !link.HidesDiscovery() {
		content.Description = *link.Description
	}
	asset, err := render.Render(content, spec)
	var field *render.FieldError
	if errors.As(err, &field) {
		return nil, apierr.Wrap(err, kindInvalidDesign, "errors.invalid_design").
			WithDetails(map[string]string{field.Field: field.Code})
	}
	if err != nil {
		return nil, failWrap(err, apierr.Internal)
	}

	disposition := "inline"
	if req.Params.Download != nil && *req.Params.Download {
		disposition = "attachment"
	}
	filename := assetFilename(name, string(spec.Preset), string(spec.Format))
	headers := gen.RenderLinkAsset200ResponseHeaders{
		// Image proxies such as GitHub's camo fetch README images from shared addresses; let them cache.
		CacheControl:       ptr("public, max-age=3600"),
		ContentDisposition: ptr(mime.FormatMediaType(disposition, map[string]string{"filename": filename})),
		XAssetWidth:        &asset.Width,
		XAssetHeight:       &asset.Height,
	}
	body, size := bytes.NewReader(asset.Body), int64(len(asset.Body))
	switch asset.ContentType {
	case "image/png":
		return gen.RenderLinkAsset200ImagepngResponse{Body: body, ContentLength: size, Headers: headers}, nil
	case "image/jpeg":
		return gen.RenderLinkAsset200ImagejpegResponse{Body: body, ContentLength: size, Headers: headers}, nil
	default:
		return gen.RenderLinkAsset200ImagesvgXmlResponse{Body: body, ContentLength: size, Headers: headers}, nil
	}
}

func ptr[T any](v T) *T { return &v }

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// assetSpec maps query parameters onto the renderer's specification; the renderer validates it.
func assetSpec(p gen.RenderLinkAssetParams) render.Spec {
	return render.Spec{
		Preset:          render.Preset(deref(p.Preset)),
		Format:          render.Format(deref(p.Format)),
		Theme:           render.Theme(deref(p.Theme)),
		Foreground:      deref(p.Fg),
		Background:      deref(p.Bg),
		Accent:          deref(p.Accent),
		Caption:         deref(p.Caption),
		ErrorCorrection: render.ECLevel(deref(p.Ec)),
		Margin:          p.Margin,
		Radius:          deref(p.Radius),
		Size:            deref(p.Size),
		Transparent:     deref(p.Transparent),
		Logo:            p.Logo == nil || *p.Logo,
		LabelWidthMM:    deref(p.LabelWidth),
		LabelHeightMM:   deref(p.LabelHeight),
		DPI:             deref(p.Dpi),
	}
}

var unsafeFilename = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// assetFilename names a download after the project and preset, e.g. Goptivum-presentation.png.
func assetFilename(name, preset, format string) string {
	base := strings.Trim(unsafeFilename.ReplaceAllString(name, "_"), "._")
	if base == "" {
		base = "qr2git"
	}
	if preset == "" {
		preset = string(render.PresetQR)
	}
	if format == "" {
		format = string(render.FormatSVG)
	}
	return fmt.Sprintf("%s-%s.%s", base, preset, format)
}
