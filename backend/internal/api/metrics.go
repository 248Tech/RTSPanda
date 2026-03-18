package api

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// appMetrics holds all application-level counters. Values are safe for
// concurrent access via sync/atomic. No external Prometheus library needed.
var appMetrics struct {
	httpRequestsTotal    atomic.Int64
	httpRequests2xx      atomic.Int64
	httpRequests4xx      atomic.Int64
	httpRequests5xx      atomic.Int64
	httpRequestDurationMs atomic.Int64 // sum of durations in ms (for avg)

	networkBytesIn  atomic.Int64
	networkBytesOut atomic.Int64

	streamHealthChecks atomic.Int64
	discordWebhooks    atomic.Int64
}

// RecordRequest records one completed HTTP request.
func RecordRequest(status int, durationMs int64) {
	appMetrics.httpRequestsTotal.Add(1)
	appMetrics.httpRequestDurationMs.Add(durationMs)
	switch {
	case status >= 500:
		appMetrics.httpRequests5xx.Add(1)
	case status >= 400:
		appMetrics.httpRequests4xx.Add(1)
	default:
		appMetrics.httpRequests2xx.Add(1)
	}
}

// RecordNetworkBytes records bytes transferred over HTTP.
func RecordNetworkBytes(in, out int64) {
	if in > 0 {
		appMetrics.networkBytesIn.Add(in)
	}
	if out > 0 {
		appMetrics.networkBytesOut.Add(out)
	}
}

// RecordStreamHealthCheck increments the health check counter.
func RecordStreamHealthCheck() {
	appMetrics.streamHealthChecks.Add(1)
}

// RecordDiscordWebhook increments the Discord webhook counter.
func RecordDiscordWebhook() {
	appMetrics.discordWebhooks.Add(1)
}

// handleMetrics serves a Prometheus-compatible text exposition.
// GET /metrics
func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	total := appMetrics.httpRequestsTotal.Load()
	avgMs := int64(0)
	if total > 0 {
		avgMs = appMetrics.httpRequestDurationMs.Load() / total
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP rtspanda_http_requests_total Total HTTP requests handled\n")
	fmt.Fprintf(w, "# TYPE rtspanda_http_requests_total counter\n")
	fmt.Fprintf(w, "rtspanda_http_requests_total %d\n", total)

	fmt.Fprintf(w, "# HELP rtspanda_http_requests_2xx HTTP 2xx responses\n")
	fmt.Fprintf(w, "# TYPE rtspanda_http_requests_2xx counter\n")
	fmt.Fprintf(w, "rtspanda_http_requests_2xx %d\n", appMetrics.httpRequests2xx.Load())

	fmt.Fprintf(w, "# HELP rtspanda_http_requests_4xx HTTP 4xx responses\n")
	fmt.Fprintf(w, "# TYPE rtspanda_http_requests_4xx counter\n")
	fmt.Fprintf(w, "rtspanda_http_requests_4xx %d\n", appMetrics.httpRequests4xx.Load())

	fmt.Fprintf(w, "# HELP rtspanda_http_requests_5xx HTTP 5xx responses\n")
	fmt.Fprintf(w, "# TYPE rtspanda_http_requests_5xx counter\n")
	fmt.Fprintf(w, "rtspanda_http_requests_5xx %d\n", appMetrics.httpRequests5xx.Load())

	fmt.Fprintf(w, "# HELP rtspanda_http_avg_duration_ms Average HTTP request duration in milliseconds\n")
	fmt.Fprintf(w, "# TYPE rtspanda_http_avg_duration_ms gauge\n")
	fmt.Fprintf(w, "rtspanda_http_avg_duration_ms %d\n", avgMs)

	fmt.Fprintf(w, "# HELP rtspanda_network_bytes_in Total HTTP request body bytes received\n")
	fmt.Fprintf(w, "# TYPE rtspanda_network_bytes_in counter\n")
	fmt.Fprintf(w, "rtspanda_network_bytes_in %d\n", appMetrics.networkBytesIn.Load())

	fmt.Fprintf(w, "# HELP rtspanda_network_bytes_out Total HTTP response bytes sent\n")
	fmt.Fprintf(w, "# TYPE rtspanda_network_bytes_out counter\n")
	fmt.Fprintf(w, "rtspanda_network_bytes_out %d\n", appMetrics.networkBytesOut.Load())

	fmt.Fprintf(w, "# HELP rtspanda_stream_health_checks_total Total stream keepalive health checks\n")
	fmt.Fprintf(w, "# TYPE rtspanda_stream_health_checks_total counter\n")
	fmt.Fprintf(w, "rtspanda_stream_health_checks_total %d\n", appMetrics.streamHealthChecks.Load())

	fmt.Fprintf(w, "# HELP rtspanda_discord_webhooks_total Total Discord webhook calls\n")
	fmt.Fprintf(w, "# TYPE rtspanda_discord_webhooks_total counter\n")
	fmt.Fprintf(w, "rtspanda_discord_webhooks_total %d\n", appMetrics.discordWebhooks.Load())
}
