package frontend

import (
	"embed"
	"io/fs"
	"sync"
)

//go:embed all:dist
var embedded embed.FS

var (
	staticFS fs.FS
	fsOnce   sync.Once
)

// Static returns the embedded frontend filesystem rooted at dist.
func Static() fs.FS {
	fsOnce.Do(func() {
		sub, err := fs.Sub(embedded, "dist")
		if err != nil {
			staticFS = embedded
			return
		}
		staticFS = sub
	})
	return staticFS
}

// Available reports whether a real Nuxt static build has been embedded.
func Available() bool {
	info, err := fs.Stat(Static(), "index.html")
	return err == nil && !info.IsDir()
}
