// api/v1/pages.go

package api

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/common/messages"
	"smegg.me/qr2g.it/internal/frontend"
)

// pageMeta is what crawlers and link previews read from the initial HTML.
type pageMeta struct {
	Title       string
	Description string
	NoIndex     bool
}

var (
	titleTag       = regexp.MustCompile(`(?s)<title>.*?</title>`)
	descriptionTag = regexp.MustCompile(`<meta name="description"[^>]*>`)
)

// withMeta replaces the shell's title and adds escaped description, Open Graph and robots tags.
func withMeta(page []byte, m pageMeta) []byte {
	var head strings.Builder
	tag := func(attr, key, value string) {
		if value != "" {
			fmt.Fprintf(&head, `<meta %s="%s" content="%s">`, attr, key, html.EscapeString(value))
		}
	}
	tag("name", "description", m.Description)
	tag("property", "og:title", m.Title)
	tag("property", "og:description", m.Description)
	tag("property", "og:site_name", "qr2g.it")
	if m.NoIndex {
		tag("name", "robots", "noindex")
	}
	if m.Title != "" {
		page = titleTag.ReplaceAll(page, []byte("<title>"+html.EscapeString(m.Title)+"</title>"))
	}
	if m.Description != "" {
		page = descriptionTag.ReplaceAll(page, nil)
	}
	return bytes.Replace(page, []byte("</head>"), []byte(head.String()+"</head>"), 1)
}

// serveShell answers with the SPA shell, which routes the path on the client.
func (s *Server) serveShell(c *gin.Context, status int, m pageMeta) bool {
	if !frontend.Available() {
		return false
	}
	page, err := fs.ReadFile(frontend.Static(), "200.html")
	if err != nil {
		return false
	}
	page = withMeta(injectFrontendRuntimeConfig("200.html", page), m)
	c.Header("Cache-Control", "no-cache")
	if m.NoIndex {
		c.Header("X-Robots-Tag", "noindex")
	}
	c.Data(status, "text/html; charset=utf-8", page)
	c.Abort()
	return true
}

func acceptsHTML(c *gin.Context) bool {
	return strings.Contains(c.GetHeader("Accept"), "text/html")
}

func (s *Server) handleRoot(c *gin.Context) {
	if s.serveFrontendFile(c, "/") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"service": "qr2g.it", "docs": s.publicBaseURL + "/guide.md"})
}

// handleSitePage serves a generated site page or static document by its own route.
func (s *Server) handleSitePage(c *gin.Context) {
	if s.serveFrontendFile(c, c.Request.URL.Path) {
		return
	}
	if !s.serveShell(c, http.StatusOK, pageMeta{Title: "qr2g.it", Description: messages.PageDescription(s.say)}) {
		c.AbortWithStatus(http.StatusNotFound)
	}
}

func (s *Server) handleRobots(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, "User-agent: *\nDisallow: /api/\nAllow: /api/v1/links/*/asset\n\nSitemap: %s/sitemap.xml\n", s.publicBaseURL)
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// handleSitemap lists the site pages; nothing else is published.
func (s *Server) handleSitemap(c *gin.Context) {
	out := sitemap{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, p := range []string{"/", "/docs"} {
		out.URLs = append(out.URLs, sitemapURL{Loc: s.publicBaseURL + p})
	}
	body, err := xml.Marshal(out)
	if err != nil {
		respond(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", append([]byte(xml.Header), body...))
}

func (s *Server) handleFavicon(c *gin.Context) {
	if s.serveFrontendFile(c, "/favicon.svg") {
		return
	}
	c.AbortWithStatus(http.StatusNotFound)
}
