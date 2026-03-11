# RTSPanda Enterprise Performance & Stack Audit

**Date:** 2025-03-09  
**Scope:** Backend (Go), frontend (React/Vite/TypeScript), database (SQLite), streaming (mediamtx), and end-to-end request flows.

---

## Executive Summary

RTSPanda is a small, focused stack suitable for single-node or small-scale deployments. This audit identifies performance bottlenecks, scalability limits, and stack choices that would need to be addressed for enterprise-scale deployment (high camera count, many concurrent users, HA, and observability).

| Category              | Rating   | Summary |
|-----------------------|----------|---------|
| Backend API           | Moderate | Simple and correct; no compression, no caching, stream-status scales poorly with N cameras. |
| Database              | Limited  | SQLite with single connection; correct for single process; no indexes for list ordering. |
| Streaming pipeline    | Moderate | mediamtx on-demand is efficient; full config reload on change; no status caching. |
| Frontend              | Moderate | N+1 stream-status calls, no code splitting, hls.js in main bundle, fixed polling. |
| Observability         | Low      | No metrics, tracing, or structured logging. |
| Scalability / HA      | Low      | Single process, single DB file, no horizontal scaling. |

---

## 1. Stack Overview

| Layer        | Technology              | Version / notes |
|-------------|--------------------------|------------------|
| Backend     | Go (net/http)            | go 1.26          |
| Database    | SQLite (modernc.org/sqlite) | WAL mode, foreign keys |
| Streaming   | mediamtx (subprocess)     | External binary, HLS on :8888, API on 127.0.0.1:9997 |
| Frontend    | React 19, Vite 7, TypeScript 5.9 | SPA, no SSR |
| Video client| hls.js                   | ^1.6.15          |
| Build       | go build / vite build    | Single binary; static assets |

**Data flow (simplified):** Browser → Go API (cameras CRUD, stream status) and → Go HLS proxy → mediamtx → HLS segments. Stream status is derived by Go calling mediamtx HTTP API (`/v3/paths/list`) per request.

---

## 2. Backend Performance

### 2.1 HTTP Layer

- **Compression:** No gzip or other response compression. JSON and HLS proxy traffic are uncompressed. **Impact:** Higher bandwidth and slower responses on slow links.
- **Timeouts:** ReadTimeout 15s, WriteTimeout 15s, IdleTimeout 60s are set — good for avoiding hung connections.
- **Connection handling:** Default `net/http` behavior; keep-alive is on. No explicit MaxConns or rate limiting.
- **Recommendation:** Add `compress/gzip` middleware for JSON (and optionally for HLS if not already compressed by mediamtx). Consider configurable request timeouts and max header/body limits.

### 2.2 Database (SQLite)

- **Connection pool:** `SetMaxOpenConns(1)` — correct for SQLite’s single-writer model. All DB access is effectively serialized per process.
- **WAL mode:** Enabled via connection string — good for concurrent reads during writes.
- **Migrations:** Run at startup from embed; no long-running migration strategy for large datasets.
- **Indexes:** Only the primary key on `cameras(id)`. `List()` uses `ORDER BY position, created_at` with no composite index. **Impact:** For large camera counts (e.g. thousands), list could do a full scan and sort.
- **Recommendation:** Add index `CREATE INDEX idx_cameras_order ON cameras(position, created_at);` for list performance. Document that SQLite is the right choice for single-node only; for multi-node or very high write throughput, plan for a different store.

### 2.3 Camera List API

- **Behavior:** Loads all cameras into memory and returns them. No pagination, filtering, or sorting parameters.
- **Impact:** Fine for hundreds of cameras; for very large N, memory and response size grow linearly; client and network pay the cost of one large JSON payload.
- **Recommendation:** Add optional pagination (limit/offset or cursor) and a composite index; consider field selection if payload size becomes an issue.

### 2.4 Stream Status API (Critical Path)

- **Behavior:** `GET /api/v1/cameras/{id}/stream` calls `StreamStatus(id)`, which:
  1. Performs a DB lookup for the camera.
  2. Calls mediamtx `GET http://127.0.0.1:9997/v3/paths/list` (full path list).
  3. Finds the matching path and returns status + HLS URL.
- **Problems:**
  - Every stream-status request triggers a new HTTP client and a full mediamtx API call. No connection reuse (new `http.Client` per call in `health.go`).
  - No caching. N dashboard cards → N API requests → N mediamtx calls, each returning the full path list. Redundant work and load on mediamtx.
- **Impact:** With 20 cameras, opening the dashboard causes 20 mediamtx API calls in quick succession. Latency and mediamtx load scale with N.
- **Recommendation:**
  - Cache mediamtx path list in Go (e.g. in-memory with TTL 2–5s) and derive all camera statuses from one list; expose a single “all stream statuses” endpoint or batch stream status in one response.
  - Use a shared `http.Client` with connection pooling for mediamtx API calls (and any other outbound HTTP).

---

## 3. Streaming Pipeline (mediamtx)

- **On-demand:** `sourceOnDemand: yes` and `sourceOnDemandCloseAfter: 10s` — streams are created on first request and closed when idle. Good for resource use when many cameras exist but few are watched.
- **Config reload:** On add/update/remove camera, the process either uses mediamtx API (add/delete path) or falls back to full restart + config rewrite. Restart path rewrites entire YAML and restarts the process — brief interruption.
- **Single process:** One mediamtx instance. For enterprise, consider multiple instances or clustering if mediamtx supports it, and a load balancer in front of HLS.
- **Recommendation:** Keep on-demand; document reload/restart behavior. If path list grows very large, consider splitting config or multiple mediamtx instances with a single LB.

---

## 4. Frontend Performance

### 4.1 Bundle and Loading

- **Structure:** Single SPA; Dashboard, Settings, and CameraView (and thus VideoPlayer + hls.js) are all in the same entry bundle. No route-based code splitting.
- **Impact:** hls.js is loaded and parsed on initial load even when the user never opens a camera view. Larger initial bundle and longer TTI.
- **Recommendation:** Use `React.lazy()` and `Suspense` for route-level chunks (e.g. Dashboard, Settings, CameraView). Lazy-load the route that contains `VideoPlayer` so hls.js is loaded only when entering a camera view. Consider dynamic `import()` for hls.js inside VideoPlayer.

### 4.2 API Usage and N+1 Pattern

- **Dashboard:** Fetches full camera list (`getCameras()`), then each `CameraCard` mounts and calls `getStreamInfo(camera.id)` once. So for N cameras: 1 list request + N stream-status requests, all on first paint.
- **No batching:** There is no batch stream-status endpoint; the frontend cannot get “status for all cameras” in one call.
- **Impact:** High request count and backend/mediamtx load when N is large; waterfall of N requests can delay status display.
- **Recommendation:** Add backend endpoint that returns list of cameras plus stream status for all (or a subset) in one response, using a cached path list. Frontend then calls one “dashboard” endpoint (or list + batch status) instead of 1 + N.

### 4.3 Polling

- **Dashboard:** Polls `getCameras()` every 30s. Full list refetch; no conditional GET (ETag/If-None-Match) or delta updates.
- **Impact:** Predictable but fixed load; every 30s every open dashboard triggers a full list reload. No freshness guarantee for stream status (status is fetched once per card on mount, not re-polled).
- **Recommendation:** Consider longer interval or conditional GET to reduce bandwidth. If stream status is cached on backend, add a lightweight “stream statuses” poll or WebSocket for live status without per-camera endpoints.

### 4.4 Rendering and State

- **Camera list:** Simple list; no virtualization. Fine for hundreds of items; for thousands, consider virtualized list (e.g. react-window).
- **CameraCard:** One effect per card for `getStreamInfo`; cancellation on unmount is implemented — good. No shared cache (e.g. React Query or SWR) for stream info, so remounting refetches.
- **Recommendation:** Introduce a client-side cache (e.g. React Query or SWR) for cameras and stream info with short stale time to avoid duplicate requests and to centralize refetch/polling logic.

---

## 5. HLS Proxy and Media Path

- **Implementation:** `httputil.NewSingleHostReverseProxy` to `127.0.0.1:8888` with `StripPrefix("/hls")`. No custom Director or buffer tuning.
- **Impact:** Adequate for low concurrency. Under high concurrency, consider buffer sizes and timeouts; ensure errors and timeouts are handled so the proxy does not hold connections indefinitely.
- **Recommendation:** Keep current design for single-node; for multi-node, put mediamtx (or HLS origin) behind a dedicated LB and optionally add caching (e.g. CDN) for segment URLs.

---

## 6. Concurrency and Resource Limits

- **Go:** One server process; goroutines per request (default). No explicit limit on concurrent requests or on mediamtx API calls triggered by them.
- **SQLite:** Single writer; concurrent reads allowed. No read timeouts or statement timeouts configured.
- **mediamtx:** Single subprocess; restart/reload is sequential. Watchdog blocks on restart (3–10s backoff), which is acceptable for single-instance.
- **Recommendation:** If you add caching and a batch status endpoint, consider a semaphore or limit on concurrent mediamtx API calls to avoid thundering herd during dashboard load. Document that the app is single-process and single-DB for HA planning.

---

## 7. Observability and Operability

- **Logging:** Standard `log` only; no log levels, structured fields, or request IDs. Errors are logged in streams layer; API layer does not log requests or status codes.
- **Metrics:** No Prometheus/StatsD or similar. No counters for request rate, latency, errors, or stream-status cache hit rate.
- **Tracing:** No distributed tracing. Hard to trace a request from frontend → API → DB/mediamtx.
- **Health:** `/api/v1/health` returns fixed `{"status":"ok"}`. No DB ping or mediamtx liveness check.
- **Recommendation:** Add structured logging (e.g. slog) with request ID and status. Expose `/metrics` (e.g. Prometheus) for request counts, latencies, and errors. Extend health to optionally check DB and mediamtx and return 503 if unhealthy. Consider OpenTelemetry for tracing if the stack grows.

---

## 8. Scalability and High Availability

- **Horizontal scaling:** Not supported. Single SQLite file and single mediamtx process; no shared storage or coordination. Running multiple Go instances would require a different DB and a strategy for mediamtx (e.g. one mediamtx per instance or external cluster).
- **Vertical scaling:** SQLite and single-process design benefit from a single strong node; no built-in sharding or partitioning.
- **Recommendation:** Document that the current design is for single-node/small deployment. For HA or scale-out, plan for: replace SQLite with a networked DB (e.g. PostgreSQL), optional read replicas, mediamtx clustering or LB, and stateless API instances with shared config/state.

---

## 9. Stack and Dependency Notes

- **Go 1.26:** Very recent; confirm support in deployment environment and CI.
- **Dependencies:** Few (uuid, sqlite); no framework. Low supply-chain and upgrade surface.
- **Frontend:** React 19, Vite 7, hls.js — modern stack. Lockfile and audit in CI recommended.
- **Build:** No multi-stage Docker or build optimization (e.g. stripping, minimal base image) reviewed here; recommend a minimal container image and reproducible build.

---

## 10. Prioritized Recommendations

| Priority | Area              | Action |
|----------|-------------------|--------|
| P0       | Stream status     | Cache mediamtx path list in backend; add batch “all stream statuses” (or list+status) endpoint; use single shared HTTP client for mediamtx. |
| P0       | Frontend N+1      | Use new batch endpoint for dashboard; avoid N per-card stream-status calls on load. |
| P1       | Backend           | Add gzip for JSON; add index on `(position, created_at)` for list; add shared `http.Client` for mediamtx. |
| P1       | Frontend          | Route-based code splitting; lazy-load CameraView and hls.js. |
| P1       | Observability     | Structured logging; basic metrics (request count, latency, errors); health check DB and mediamtx. |
| P2       | API               | Optional pagination and field selection for list cameras; request body size limit. |
| P2       | Frontend          | Client-side cache (e.g. React Query) for cameras and stream info; consider conditional GET or longer poll interval. |
| P2       | Scale/HA          | Document single-node limits; outline path to multi-node (DB, mediamtx, LB) if required. |

---

## 11. Summary Table

| Dimension           | Current state                    | Enterprise-ready target (high level)        |
|--------------------|-----------------------------------|---------------------------------------------|
| Stream status load  | N mediamtx calls per dashboard load | 1 cached path list + 1 batch API response  |
| List cameras       | Full scan, no index               | Indexed, optional pagination                 |
| API responses      | Uncompressed                      | Gzip for JSON (and HLS if needed)           |
| Frontend bundle    | Monolithic, hls.js upfront        | Code-split, lazy hls.js                     |
| Observability      | Log only                          | Logs + metrics + health depth + optional tracing |
| Scale / HA         | Single node                       | Documented; path to multi-node if needed    |

This audit is based on static code review and architecture analysis. Validate with load tests (concurrent users, N cameras, stream-status and list endpoints) and production-like runs.
