// Package webui serves the built React app embedded in the binary.
package webui

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"strings"
)

func init() {
	// Go's built-in table doesn't know the web app manifest type.
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

// Index returns the built index.html, for pages the server fills in itself.
func Index() []byte {
	b, err := dist.ReadFile("dist/index.html")
	if err != nil {
		panic(err)
	}
	return b
}

//go:embed all:dist
var dist embed.FS

// Handler serves static files and falls back to index.html for app routes.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(sub, path); err != nil {
			r.URL.Path = "/"
		}
		switch {
		case strings.HasPrefix(path, "assets/"):
			// Hashed file names: a new build gets new names.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case path == "sw.js" || r.URL.Path == "/":
			// The service worker and the page must be checked on every visit
			// so updates reach installed copies of the app.
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}
