package hub

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var distFS embed.FS

var indexPlaceholder = []byte("__TRACELOG_URL_PREFIX__")

// NewSPAHandler serves the embedded dashboard; urlPathPrefix is "" or "/tracelog" (normalized).
func NewSPAHandler(urlPathPrefix string) http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("embedded dist not found: " + err.Error())
	}

	rawIndex, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		panic("embedded index.html: " + err.Error())
	}

	replacement := []byte(urlPathPrefix)
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		rel := strings.TrimPrefix(path, "/")
		if rel == "" {
			rel = "index.html"
		}

		if rel != "index.html" {
			if _, statErr := fs.Stat(sub, rel); statErr == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		} else {
			// Explicit / or /index.html
			serveIndex(w, rawIndex, replacement)
			return
		}

		// SPA fallback: no such static file
		serveIndex(w, rawIndex, replacement)
	})
}

func serveIndex(w http.ResponseWriter, rawIndex, replacement []byte) {
	body := bytes.ReplaceAll(rawIndex, indexPlaceholder, replacement)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(body)
}
