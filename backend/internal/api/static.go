package api

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// web is populated at build time: copy frontend/dist contents into backend/internal/api/web,
// then run go build. See Makefile or build script at repo root.
//
// Include the tracked .gitkeep placeholder so clean checkouts still compile
// before frontend assets are copied into web/.
//go:embed all:web
var webEmbed embed.FS

// fallbackHTML is served when index.html is not in the embed (e.g. go build without copying frontend).
const fallbackHTML = `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>RTSPanda</title></head><body><p>Frontend not built.</p><p>Run <code>make build</code> (or build frontend then copy <code>frontend/dist</code> to <code>backend/internal/api/web</code>) and rebuild the binary.</p></body></html>`

func staticHandler() (http.Handler, error) {
	root, err := fs.Sub(webEmbed, "web")
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "" || p == "/" {
			p = "index.html"
		} else {
			p = strings.TrimPrefix(p, "/")
		}
		// Security: no path traversal
		if strings.Contains(p, "..") {
			http.NotFound(w, r)
			return
		}
		f, err := root.Open(p)
		if err != nil {
			// SPA fallback: any non-file route serves index.html
			index, ierr := fs.ReadFile(root, "index.html")
			if ierr != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(fallbackHTML))
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(index)
			return
		}
		defer f.Close()
		stat, err := f.Stat()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if stat.IsDir() {
			http.NotFound(w, r)
			return
		}
		content, err := fs.ReadFile(root, p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Cache static assets (Vite adds hashes to filenames)
		if strings.HasPrefix(p, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		w.Header().Set("Content-Type", contentType(p))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}), nil
}

func contentType(name string) string {
	switch path.Ext(name) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}
