// api/v1/routes.go

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smegg99/s99ab/api/apierr"
	"github.com/smegg99/s99ab/api/apihealth"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"smegg.me/qr2g.it/api/v1/gen"
	"smegg.me/qr2g.it/contracts"
)

const batchBodyBytes = 32 << 10

func (s *Server) registerRoutes() {
	e := s.engine
	e.Use(s.requestIDMiddleware(), s.recoveryMiddleware(), s.loggingMiddleware())

	apihealth.Register(e, apihealth.Options{
		Version: s.Version,
		Checks:  []apihealth.Check{{Name: "database", Fn: s.Ready}},
	})

	e.GET("/", s.handleRoot)
	e.GET("/robots.txt", s.handleRobots)
	e.GET("/sitemap.xml", s.handleSitemap)
	e.GET("/favicon.ico", s.handleFavicon)
	// Explicit routes keep site pages out of the slug and owner/repo dispatcher.
	for _, page := range []string{"/docs", "/guide.md", "/llms.txt"} {
		e.GET(page, s.handleSitePage)
		e.HEAD(page, s.handleSitePage)
	}

	e.GET("/swagger", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/swagger/index.html") })
	e.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api/v1/openapi.yaml")))

	v1 := e.Group("/api/v1")
	v1.GET("/openapi.yaml", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", contracts.Bundle)
	})

	w := gen.ServerInterfaceWrapper{
		Handler:      gen.NewStrictHandlerWithOptions(requestScoped{next: handlers{s}}, nil, strictOptions()),
		ErrorHandler: bindError,
	}
	auth := s.automationAuth(chargeOne)
	sized := s.bodyLimitMiddleware(s.maxBodyBytes)
	read := s.rateLimitMiddleware(s.readBucket, "read")
	resolve := s.rateLimitMiddleware(s.resolveBucket, "resolve")

	v1.POST("/links", sized, auth, s.rateLimitMiddleware(s.createBucket, "create"), w.CreateLink)
	v1.POST("/links/batch", s.bodyLimitMiddleware(batchBodyBytes), s.automationAuth(chargeBatch), w.CreateLinks)
	v1.GET("/links/lookup", auth, read, w.LookupLink)
	v1.GET("/links/:slug", auth, resolve, w.GetLink)
	v1.GET("/links/:slug/resolve", auth, resolve, w.ResolveLink)
	v1.POST("/links/:slug/policy/refresh", auth, s.rateLimitMiddleware(s.refreshBucket, "refresh"), w.RefreshLinkPolicy)
	v1.GET("/links/:slug/asset", auth, s.rateLimitMiddleware(s.assetBucket, "asset"), w.RenderLinkAsset)
	v1.GET("/links/:slug/stats", auth, read, w.GetLinkStats)
	v1.GET("/repositories", auth, read, w.ListRepositories)
	v1.GET("/repositories/top", auth, read, w.ListTopRepositories)
	v1.GET("/repositories/:slug", auth, read, w.GetRepository)
	v1.GET("/languages", auth, read, w.ListLanguages)
	v1.GET("/coverage", auth, read, w.GetCoverage)
	v1.POST("/contact/reveal", sized, s.rateLimitMiddleware(s.contactBucket, "contact"), w.RevealContact)
	v1.GET("/automation/usage", s.automationAuth(chargeNone), w.GetAutomationUsage)

	e.NoRoute(s.handleNoRoute)
	e.NoMethod(func(c *gin.Context) { apierr.Respond(c, fail(kindNotFound)) })
}
