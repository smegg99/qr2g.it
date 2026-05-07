// api/routes.go

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "smegg.me/qr2g.it/docs"
)

func (s *Server) registerRoutes() {
	s.engine.Use(s.requestIDMiddleware())
	s.engine.Use(s.recoveryMiddleware())
	s.engine.Use(s.loggingMiddleware())

	s.engine.GET("/", s.handleRoot)
	s.engine.GET("/health", s.handleHealth)
	s.engine.GET("/ready", s.handleReady)
	s.engine.GET("/robots.txt", s.handleRobots)
	s.engine.GET("/favicon.ico", s.handleFavicon)

	s.engine.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := s.engine.Group("/api/v1")
	{
		v1.POST("/links",
			s.bodyLimitMiddleware(s.maxBodyBytes),
			s.rateLimitMiddleware(s.createBucket, "create"),
			s.handleCreateLink,
		)
		v1.GET("/links/:slug", s.handleAPIGetLink)
		v1.GET("/links/:slug/resolve", s.handleAPIResolveLink)
	}

	s.engine.NoRoute(s.handleNoRoute)
	s.engine.NoMethod(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusMethodNotAllowed)
	})
}
