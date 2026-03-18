package api

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// ─── Counting ResponseWriter ───────────────────────────────────────────────────

// countingResponseWriter wraps http.ResponseWriter to capture the status code
// and count bytes written (used for metrics and logging).
type countingResponseWriter struct {
	http.ResponseWriter
	status int
	wrote  int64
}

func (c *countingResponseWriter) WriteHeader(code int) {
	c.status = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *countingResponseWriter) Write(b []byte) (int, error) {
	n, err := c.ResponseWriter.Write(b)
	c.wrote += int64(n)
	return n, err
}

func (c *countingResponseWriter) statusCode() int {
	if c.status == 0 {
		return http.StatusOK
	}
	return c.status
}

// ─── Logging + Metrics Middleware ─────────────────────────────────────────────

// loggingMiddleware logs each request (method, path, status, duration) and
// records metrics counters. It wraps the entire mux.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Count request body bytes for bandwidth tracking.
		var bodyBytes int64
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = &countingReader{ReadCloser: r.Body, n: &bodyBytes}
		}

		crw := &countingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(crw, r)

		durationMs := time.Since(start).Milliseconds()
		status := crw.statusCode()

		RecordRequest(status, durationMs)
		RecordNetworkBytes(bodyBytes, crw.wrote)

		// Only log API routes to avoid flooding logs with static asset hits.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("%s %s %d %dms", r.Method, r.URL.Path, status, durationMs)
		}
	})
}

// countingReader wraps an io.ReadCloser to count bytes read.
type countingReader struct {
	io.ReadCloser
	n *int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.ReadCloser.Read(p)
	*c.n += int64(n)
	return n, err
}

// ─── Gzip Middleware ───────────────────────────────────────────────────────────

// gzipMiddleware compresses JSON API responses when the client supports gzip.
// Applied only to the API mux — HLS and static assets are excluded.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // length unknown after compression
		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, gz: gz}, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.gz.Write(b)
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	g.ResponseWriter.WriteHeader(code)
}
