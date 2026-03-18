package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var processStartTime = time.Now()

// SystemStats is the JSON payload for GET /api/v1/system/stats.
type SystemStats struct {
	UptimeSeconds     float64 `json:"uptime_seconds"`
	Goroutines        int     `json:"goroutines"`
	HeapAllocBytes    uint64  `json:"heap_alloc_bytes"`
	HeapSysBytes      uint64  `json:"heap_sys_bytes"`
	RSSBytes          uint64  `json:"rss_bytes"` // Linux-only; 0 elsewhere
	NetworkBytesIn    int64   `json:"network_bytes_in"`
	NetworkBytesOut   int64   `json:"network_bytes_out"`
	HTTPRequestsTotal int64   `json:"http_requests_total"`
	GOOS              string  `json:"goos"`
	GOARCH            string  `json:"goarch"`
	NumCPU            int     `json:"num_cpu"`
}

// handleSystemStats: GET /api/v1/system/stats
func handleSystemStats(w http.ResponseWriter, _ *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	stats := SystemStats{
		UptimeSeconds:     time.Since(processStartTime).Seconds(),
		Goroutines:        runtime.NumGoroutine(),
		HeapAllocBytes:    ms.HeapAlloc,
		HeapSysBytes:      ms.HeapSys,
		RSSBytes:          readRSS(),
		NetworkBytesIn:    appMetrics.networkBytesIn.Load(),
		NetworkBytesOut:   appMetrics.networkBytesOut.Load(),
		HTTPRequestsTotal: appMetrics.httpRequestsTotal.Load(),
		GOOS:              runtime.GOOS,
		GOARCH:            runtime.GOARCH,
		NumCPU:            runtime.NumCPU(),
	}
	writeJSON(w, http.StatusOK, stats)
}

// readRSS reads resident set size from /proc/self/status on Linux.
// Returns 0 on non-Linux platforms.
func readRSS() uint64 {
	if runtime.GOOS != "linux" {
		return 0
	}
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		parts := strings.Fields(line) // e.g. ["VmRSS:", "12345", "kB"]
		if len(parts) < 2 {
			return 0
		}
		kb, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return 0
		}
		return kb * 1024
	}
	return 0
}

// DBPinger is satisfied by *sql.DB (or *db.DB which embeds it).
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// handleHealthReady: GET /api/v1/health/ready
// Extended health check — validates DB and mediamtx reachability.
// Returns 503 when any critical dependency is down.
func (s *server) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	type check struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	type result struct {
		Healthy  bool             `json:"healthy"`
		Checks   map[string]check `json:"checks"`
		UptimeMs int64            `json:"uptime_ms"`
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]check{}

	// DB ping (required — no DB = completely broken)
	if err := s.db.PingContext(ctx); err != nil {
		checks["db"] = check{OK: false, Error: fmt.Sprintf("ping failed: %v", err)}
	} else {
		checks["db"] = check{OK: true}
	}

	// mediamtx API (informational — streaming may be disabled by design)
	if s.streams.IsReady() {
		checks["mediamtx"] = check{OK: true}
	} else {
		checks["mediamtx"] = check{OK: false, Error: "not reachable or streaming disabled"}
	}

	healthy := checks["db"].OK
	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, result{
		Healthy:  healthy,
		Checks:   checks,
		UptimeMs: time.Since(processStartTime).Milliseconds(),
	})
}
