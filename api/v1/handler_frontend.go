package api

import (
	"bytes"
	"io/fs"
	"mime"
	"net/http"
	pathpkg "path"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/internal/frontend"
)

var nuxtTurnstileSiteKeyPattern = regexp.MustCompile(`turnstileSiteKey:"[^"]*"`)

func (s *Server) serveFrontendFile(c *gin.Context, requestPath string) bool {
	if !frontend.Available() {
		return false
	}
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}

	name, ok := frontendFileName(requestPath)
	if !ok {
		return false
	}

	static := frontend.Static()
	info, err := fs.Stat(static, name)
	if err != nil || info.IsDir() {
		return false
	}
	b, err := fs.ReadFile(static, name)
	if err != nil {
		return false
	}
	b = injectFrontendRuntimeConfig(name, b)

	if ct := mime.TypeByExtension(pathpkg.Ext(name)); ct != "" {
		c.Header("Content-Type", ct)
	}
	if strings.HasPrefix(name, "_nuxt/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "public, max-age=300")
	}

	http.ServeContent(c.Writer, c.Request, name, info.ModTime(), bytes.NewReader(b))
	c.Abort()
	return true
}

func injectFrontendRuntimeConfig(name string, b []byte) []byte {
	if !strings.HasSuffix(name, ".html") {
		return b
	}
	siteKey := config.Get().Turnstile.SiteKey
	replacement := []byte("turnstileSiteKey:" + strconv.Quote(siteKey))
	return nuxtTurnstileSiteKeyPattern.ReplaceAll(b, replacement)
}

func frontendFileName(requestPath string) (string, bool) {
	name := strings.TrimPrefix(requestPath, "/")
	if name == "" {
		return "index.html", true
	}
	name = pathpkg.Clean(name)
	if name == "." || strings.HasPrefix(name, "../") || name == ".." || !fs.ValidPath(name) {
		return "", false
	}
	return name, true
}
