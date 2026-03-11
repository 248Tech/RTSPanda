# Phase-Based Agentic Development Performance Plan

**Source:** [performance-audit.md](./performance-audit.md)  
**Purpose:** Phased, agent-executable performance work. Each phase is self-contained with clear inputs, outputs, and acceptance criteria for agent or developer handoff.

---

## How to Use This Plan

- **Phases are sequential** unless marked otherwise; complete Phase N before starting N+1 (or follow dependency notes).
- **Per phase:** Execute the tasks in order; run verification steps; mark phase complete and record any deviations in the AI folder or handoff doc.
- **Agent context:** When starting a phase, load this file plus the listed "Artifacts to read" and the referenced audit sections. After the phase, update "Phase status" at the bottom.

---

## Phase 1 — Backend: Stream Status Cache & Batch Endpoint (P0)

**Goal:** Eliminate N mediamtx API calls per dashboard load. One cached path list + one batch API response.

**Audit reference:** §2.4 Stream Status API, §10 P0 (Stream status).

### 1.1 Scope

| Area | Change |
|------|--------|
| Backend | Shared `http.Client` for mediamtx API; in-memory cache of path list (TTL 2–5s); new endpoint returning cameras + stream statuses in one response. |
| API | New `GET /api/v1/cameras?stream_status=true` or new `GET /api/v1/stream-status` returning map of camera_id → status (and optionally hls_url). Keep existing `GET /api/v1/cameras/{id}/stream` for single-camera view; it should use the same cache and client. |

### 1.2 Agent Tasks

1. **Shared HTTP client for mediamtx**
   - **Where:** `backend/internal/streams/` (e.g. a new `client.go` or inside `mediamtx.go`).
   - **Do:** Create a package-level or manager-level `*http.Client` with reasonable timeout (e.g. 5s), reuse for all mediamtx API calls. Replace every `&http.Client{...}` in the streams package with this client.
   - **Files:** `mediamtx.go`, `health.go` (StreamStatus), any other callers of mediamtx API.

2. **Path list cache**
   - **Where:** `backend/internal/streams/` (e.g. `cache.go` or inside the existing Manager).
   - **Do:** Add an in-memory cache that stores the result of `GET /v3/paths/list` (parsed into a slice or map). TTL: 2–5 seconds. Use a mutex or sync.RWMutex. Expose a function like `GetCachedPathList(ctx) (pathsResponse, error)` that returns cached data if fresh, else fetches, updates cache, returns.
   - **Do:** Invalidate or allow cache to expire on config change (e.g. when `OnCameraAdded`, `OnCameraRemoved`, `OnCameraUpdated` are called, or simply rely on short TTL).
   - **Files:** New or existing in `streams/`.

3. **StreamStatus to use cache and shared client**
   - **Where:** `backend/internal/streams/health.go` (StreamStatus).
   - **Do:** Change `StreamStatus(cameraID)` to call `GetCachedPathList` (and shared client) instead of creating a new client and doing a fresh GET. Derive status for the given cameraID from the cached list.
   - **Files:** `health.go`.

4. **Batch endpoint: list cameras with stream statuses**
   - **Option A (preferred):** New endpoint `GET /api/v1/cameras/stream-status` that returns `{ "camera_id": { "status": "online"|"offline"|"connecting", "hls_url": "/hls/camera-{id}/index.m3u8" }, ... }`.
   - **Option B:** Extend `GET /api/v1/cameras` with query `?stream_status=true` and add a `stream_status` (or `stream_statuses`) field to the response.
   - **Do:** Implement one option. Backend must: list cameras (existing repo), fetch path list once via cache, build status map, return combined response. Reuse shared client and same cache as step 3.
   - **Files:** `backend/internal/api/router.go`, new handler in `api/` (e.g. `streams.go` or `cameras.go`), and `streams` package interface if needed (e.g. `CachedPathList()` or `StreamStatusMap(cameraIDs []string)`).

5. **Single-camera stream endpoint**
   - **Where:** `handleGetStream` and `StreamStatus(id)`.
   - **Do:** Ensure `GET /api/v1/cameras/{id}/stream` still works and uses the cached path list + shared client (already done if StreamStatus uses cache). No change to response shape.

### 1.3 Acceptance Criteria

- [ ] All mediamtx API calls use a single shared `http.Client`.
- [ ] Path list is cached with TTL 2–5s; cache is used by `StreamStatus` and by the new batch endpoint.
- [ ] New batch endpoint exists and returns all cameras’ stream status (and hls_url) in one response.
- [ ] With N cameras, one dashboard load triggers **1** mediamtx path-list call (plus list cameras from DB), not N.
- [ ] Existing `GET /api/v1/cameras/{id}/stream` unchanged in contract and uses cache.

### 1.4 Artifacts to Read

- `backend/internal/streams/mediamtx.go` (apiDo, apiBase)
- `backend/internal/streams/health.go` (StreamStatus, pathsResponse)
- `backend/internal/api/router.go`, `streams.go`, `cameras.go`
- `performance-audit.md` §2.4

### 1.5 Dependencies / Notes

- None. First phase.
- If you add a new route, document it in API docs or README if the project has them.

---

## Phase 2 — Frontend: Use Batch Stream Status (P0)

**Goal:** Dashboard makes one (or two) API calls instead of 1 + N; no per-card `getStreamInfo` on load.

**Audit reference:** §4.2 API Usage and N+1 Pattern, §10 P0 (Frontend N+1).

### 2.1 Scope

| Area | Change |
|------|--------|
| Frontend | Dashboard (or data layer) calls new batch endpoint; passes stream status (and hls_url) into each CameraCard. CameraCard no longer calls `getStreamInfo` on mount for grid view. Single-camera view can still use `getStreamInfo(id)` for one camera. |

### 2.2 Agent Tasks

1. **API client: batch stream status**
   - **Where:** `frontend/src/api/cameras.ts` (or equivalent).
   - **Do:** Add function `getStreamStatusMap(): Promise<Record<string, { status: StreamStatus; hls_url: string }>>` (or match backend response shape). Call new backend batch endpoint (e.g. `GET /api/v1/cameras/stream-status` or whatever was implemented in Phase 1).
   - **Files:** `frontend/src/api/cameras.ts`.

2. **Dashboard: fetch list + stream status together**
   - **Where:** `frontend/src/pages/Dashboard.tsx`.
   - **Do:** On load, either: (A) call `getCameras()` then `getStreamStatusMap()`, or (B) if backend returns list+status in one response, call one endpoint and derive list + map. Pass `streamStatusMap` (or per-camera status + hls_url) into `CameraGrid`/`CameraCard`.
   - **Files:** `Dashboard.tsx`.

3. **CameraGrid / CameraCard: accept status from parent**
   - **Where:** `frontend/src/components/CameraGrid.tsx`, `CameraCard.tsx`.
   - **Do:** Add optional props to `CameraCard`: e.g. `initialStatus?: StreamStatus`, `initialHlsUrl?: string` (or a single `streamInfo?: { status, hls_url }`). When provided, CameraCard does **not** call `getStreamInfo(camera.id)` in useEffect; it uses the provided status (and shows placeholder/icon as today). When not provided (e.g. single-camera view), keep existing behavior: fetch stream info on mount.
   - **Do:** CameraGrid receives the status map from Dashboard and passes the right slice to each card.
   - **Files:** `CameraGrid.tsx`, `CameraCard.tsx`.

4. **Single-camera view**
   - **Where:** `frontend/src/pages/CameraView.tsx`.
   - **Do:** Keep using `getStreamInfo(cameraId)` (or getCamera + getStreamInfo) for the detail view. No change required if backend single-camera endpoint still works.

### 2.3 Acceptance Criteria

- [ ] Opening the dashboard results in at most 2 API calls (list cameras + batch stream status), or 1 if backend combines them.
- [ ] No `getStreamInfo(camera.id)` calls from CameraCard when status is supplied by parent (dashboard grid).
- [ ] Camera cards still show correct status (online/offline/connecting) and behavior.
- [ ] CameraView (single camera) still loads and shows stream using existing `getStreamInfo`.

### 2.4 Artifacts to Read

- `frontend/src/pages/Dashboard.tsx`
- `frontend/src/components/CameraGrid.tsx`, `CameraCard.tsx`
- `frontend/src/api/cameras.ts`
- Phase 1 batch endpoint contract (route and response shape).

### 2.5 Dependencies

- **Requires Phase 1** (batch endpoint and backend contract).

---

## Phase 3 — Backend: Gzip, Index, Request Limits (P1)

**Goal:** Reduce payload size, speed up list query, and protect server from oversized bodies.

**Audit reference:** §2.1 HTTP Layer, §2.2 Database, §10 P1 (Backend).

### 3.1 Scope

| Area | Change |
|------|--------|
| Backend | Gzip compression for JSON responses; DB index for list ordering; request body size limit for JSON endpoints. |

### 3.2 Agent Tasks

1. **Gzip middleware for JSON**
   - **Where:** `backend/internal/api/` (e.g. new `middleware.go` or in `router.go`).
   - **Do:** Wrap the API mux (or specific JSON handlers) with a middleware that uses `compress/gzip`. Writer: only compress when `Content-Type` is `application/json` and client sends `Accept-Encoding: gzip`. Do not gzip HLS proxy responses (leave `/hls/` unwrapped) unless explicitly desired.
   - **Files:** New middleware file or `router.go`.

2. **DB index for list ordering**
   - **Where:** `backend/internal/db/migrations/`.
   - **Do:** Add a new migration file (e.g. `002_cameras_list_index.sql`) containing `CREATE INDEX IF NOT EXISTS idx_cameras_order ON cameras(position, created_at);`. Ensure migrations run on startup (already in place); no code change except new file.
   - **Files:** `backend/internal/db/migrations/002_cameras_list_index.sql` (or next number).

3. **Request body size limit**
   - **Where:** Handlers that read `r.Body` (create/update camera).
   - **Do:** Wrap `r.Body` with `http.MaxBytesReader(w, r.Body, maxBytes)` before decoding JSON. Choose a reasonable limit (e.g. 256KB or 64KB). Use a constant. Return 413 or 400 with clear message if limit exceeded.
   - **Files:** `backend/internal/api/cameras.go` (handleCreateCamera, handleUpdateCamera).

### 3.3 Acceptance Criteria

- [ ] JSON API responses are gzip when client sends `Accept-Encoding: gzip`; HLS proxy unchanged.
- [ ] Index exists: `idx_cameras_order` on `(position, created_at)`.
- [ ] Request body for POST/PUT camera is limited; oversized body returns 413 or 400.

### 3.4 Artifacts to Read

- `backend/internal/api/router.go`, `cameras.go`
- `backend/internal/db/db.go`, `migrations/001_initial.sql`
- `performance-audit.md` §2.1, §2.2

### 3.5 Dependencies

- None. Can run in parallel with Phase 2 after Phase 1.

---

## Phase 4 — Frontend: Route Code Splitting & Lazy hls.js (P1)

**Goal:** Reduce initial bundle size; load CameraView and hls.js only when user opens a camera.

**Audit reference:** §4.1 Bundle and Loading, §10 P1 (Frontend).

### 4.1 Scope

| Area | Change |
|------|--------|
| Frontend | Lazy-load route components (Dashboard, Settings, CameraView) with `React.lazy` + `Suspense`; ensure VideoPlayer/hls.js are only loaded when CameraView is loaded (e.g. dynamic import for hls.js in VideoPlayer if needed). |

### 4.2 Agent Tasks

1. **Lazy route components**
   - **Where:** `frontend/src/App.tsx`.
   - **Do:** Replace static imports of `Dashboard`, `Settings`, `CameraView` with `React.lazy(() => import(...))`. Wrap route content in `<Suspense fallback={...}>` with a simple loading indicator (e.g. spinner or skeleton). Keep same route logic and props.
   - **Files:** `App.tsx`.

2. **Lazy-load hls.js in VideoPlayer (optional but recommended)**
   - **Where:** `frontend/src/components/VideoPlayer.tsx`.
   - **Do:** Use dynamic `import('hls.js')` when the player actually needs it (e.g. when `hlsUrl` is set and Hls.isSupported would be used). Ensure Hls is awaited before use; handle loading/error state. This keeps hls.js out of the Dashboard/Settings chunk.
   - **Files:** `VideoPlayer.tsx`.

3. **Verify chunks**
   - **Do:** Run `npm run build` (or `vite build`) and confirm separate chunks for Dashboard, Settings, and CameraView (and that a chunk containing hls.js is only loaded when entering camera view). Document or add a short note in plan if chunk names differ.

### 4.3 Acceptance Criteria

- [ ] Initial page load does not load CameraView or hls.js (verify via network tab or bundle analyzer).
- [ ] Navigating to `/`, `/settings`, `/cameras/:id` loads the corresponding lazy chunk and shows Suspense fallback until ready.
- [ ] All existing behavior (navigation, camera list, settings, camera view playback) still works.

### 4.4 Artifacts to Read

- `frontend/src/App.tsx`
- `frontend/src/components/VideoPlayer.tsx`
- `performance-audit.md` §4.1

### 4.5 Dependencies

- None. Can run in parallel with Phase 2/3.

---

## Phase 5 — Observability: Logging, Metrics, Health (P1)

**Goal:** Structured logs, basic request metrics, and health that checks DB and mediamtx.

**Audit reference:** §7 Observability and Operability, §10 P1 (Observability).

### 5.1 Scope

| Area | Change |
|------|--------|
| Backend | Structured logging (e.g. slog) with request ID; optional middleware to log method, path, status, duration; `/metrics` endpoint (Prometheus or simple counters); `/api/v1/health` extended to optionally check DB and mediamtx and return 503 if unhealthy. |

### 5.2 Agent Tasks

1. **Structured logging**
   - **Where:** Backend entry and API layer.
   - **Do:** Introduce `log/slog` (or similar). Add a request middleware that generates a request ID (or uses `X-Request-ID` if present), sets it in context and on response header, and logs after each request: method, path, status, duration, request_id. Use structured fields (key-value). Replace critical `log.Printf` in API/streams with slog where useful.
   - **Files:** New middleware or `router.go`, and any `main.go` / startup logging.

2. **Metrics endpoint**
   - **Where:** `backend/internal/api/` or a small `metrics` package.
   - **Do:** Add `/metrics` (or `/api/v1/metrics`) that returns Prometheus text format (or a simple JSON). At minimum: counter of requests by method and path (or path pattern) and status; optionally request duration histogram or summary. Use in-memory counters; no need for full Prometheus client library unless desired.
   - **Files:** New handler, register in router. Optionally middleware to record request count and duration.

3. **Health check depth**
   - **Where:** Health handler and dependencies.
   - **Do:** Extend `GET /api/v1/health` (or add `GET /api/v1/health/ready`) to: ping DB (e.g. `SELECT 1` or `Ping()`); optionally check mediamtx (e.g. GET to 127.0.0.1:9997). If any check fails, return 503 and body indicating which dependency failed. Keep 200 only when all are healthy. Document query param or path for “deep” check if you want to keep a lightweight liveness (e.g. `/health` = ok, `/health/ready` = deep).
   - **Files:** `router.go`, handler (e.g. in a small `health.go`), and pass DB and stream manager into health if needed.

### 5.3 Acceptance Criteria

- [ ] Each API request has a request ID in logs and optionally on response header.
- [ ] Logs are structured (key-value); at least method, path, status, duration, request_id after each request.
- [ ] `/metrics` returns request counts (and optionally latency) in Prometheus or simple format.
- [ ] Health endpoint can return 503 when DB or mediamtx is down; 200 when both are up.

### 5.4 Artifacts to Read

- `backend/cmd/rtspanda/main.go`
- `backend/internal/api/router.go`
- `backend/internal/db/db.go`
- `backend/internal/streams/` (mediamtx client for health check)
- `performance-audit.md` §7

### 5.5 Dependencies

- None. Can run after or in parallel with Phase 1–4.

---

## Phase 6 — Optional: Pagination & Client-Side Cache (P2)

**Goal:** Optional list pagination and a client cache (e.g. React Query or SWR) for cameras and stream info.

**Audit reference:** §2.3 Camera List API, §4.3 Polling, §4.4 Rendering and State, §10 P2.

### 6.1 Scope

| Area | Change |
|------|--------|
| Backend | Optional `limit`/`offset` (or `cursor`) query params on `GET /api/v1/cameras`; response shape supports pagination (e.g. `items` + `total` or `next_cursor`). |
| Frontend | Integrate React Query or SWR: cameras list and stream status (or batch) with short stale time; reuse cache across Dashboard and Settings; optional longer poll interval or conditional GET if backend supports. |

### 6.2 Agent Tasks

1. **Backend: optional pagination**
   - **Where:** `GET /api/v1/cameras` handler and repository.
   - **Do:** Accept query params e.g. `limit` (default all), `offset`. Repository: add `List(limit, offset int)` or equivalent; return total count if feasible (e.g. `SELECT COUNT(*)`). Response: `{ "items": [...], "total": N }` or similar. Keep default behavior (no params = return all) for backward compatibility.
   - **Files:** `api/cameras.go`, `cameras/repository.go`, possibly `cameras/service.go`.

2. **Frontend: React Query or SWR**
   - **Where:** Dashboard, Settings, API layer.
   - **Do:** Add dependency (e.g. `@tanstack/react-query` or `swr`). Create hooks: e.g. `useCameras()`, `useStreamStatusMap()` (or combined `useDashboardData()`). Use for list and batch stream status; set staleTime 5–30s. Replace manual `useState`/`useEffect` fetch and polling with query invalidation or refetchInterval. Ensure CameraCard and CameraView can still get data from cache when navigating.
   - **Files:** `frontend/src/api/` or `frontend/src/hooks/`, `Dashboard.tsx`, `Settings.tsx`, optionally `CameraView.tsx`.

3. **Polling / refetch**
   - **Do:** Use library’s refetch interval (e.g. 30s or 60s) for cameras; optionally refetch stream status map on same interval. Remove duplicate fetch logic from Dashboard and Settings.

### 6.3 Acceptance Criteria

- [ ] `GET /api/v1/cameras?limit=10&offset=0` returns paginated items and total (or next cursor).
- [ ] Frontend uses React Query or SWR for cameras and stream status; no duplicate fetches; shared cache across pages.
- [ ] Polling/refetch is centralized and configurable.

### 6.4 Artifacts to Read

- `backend/internal/api/cameras.go`, `cameras/repository.go`
- `frontend/src/pages/Dashboard.tsx`, `Settings.tsx`
- `performance-audit.md` §2.3, §4.3, §4.4

### 6.5 Dependencies

- Phase 1 and 2 done (batch endpoint and dashboard using it). Pagination can be done independently; client cache is most useful after batch endpoint exists.

---

## Phase 7 — Documentation: Single-Node Limits & Scale Path (P2)

**Goal:** Document current single-node design and a path to multi-node/HA for future work.

**Audit reference:** §8 Scalability and High Availability, §10 P2 (Scale/HA).

### 7.1 Scope

| Area | Change |
|------|--------|
| Docs | Add a short architecture/scale doc in repo (e.g. `docs/SCALABILITY.md` or section in existing ARCHITECTURE) describing single-node limits and options for multi-node (DB, mediamtx, LB). |

### 7.2 Agent Tasks

1. **Write scalability doc**
   - **Where:** `docs/SCALABILITY.md` or `AI/SCALABILITY.md` (or extend `AI/ARCHITECTURE.md` if it exists).
   - **Do:** Describe: current single-process, single-SQLite, single-mediamtx design; recommended deployment (one node, one binary); limits (no horizontal scale, single point of failure). Outline path to scale: replace SQLite with PostgreSQL (or similar), run multiple API instances, mediamtx clustering or one mediamtx per instance behind LB, shared config/state. No code changes required.
   - **Files:** New or updated doc in `docs/` or `AI/`.

### 7.3 Acceptance Criteria

- [ ] Document exists and states single-node design and its limits.
- [ ] Document describes at least one concrete path to multi-node or HA (DB, API, mediamtx, LB).

### 7.4 Dependencies

- None. Can be done any time.

---

## Phase Status (Update as you complete)

| Phase | Name                          | Status      | Completed date / notes |
|-------|-------------------------------|-------------|-------------------------|
| 1     | Backend stream status & batch | Not started | |
| 2     | Frontend batch usage          | Not started | |
| 3     | Backend gzip, index, limits   | Not started | |
| 4     | Frontend code splitting       | Not started | |
| 5     | Observability                 | Not started | |
| 6     | Pagination & client cache     | Not started | |
| 7     | Scalability documentation     | Not started | |

---

## Quick Reference: Phase → Audit Section

| Phase | Audit § | Priority |
|-------|---------|----------|
| 1     | 2.4, 10 P0 | P0 |
| 2     | 4.2, 10 P0 | P0 |
| 3     | 2.1, 2.2, 10 P1 | P1 |
| 4     | 4.1, 10 P1 | P1 |
| 5     | 7, 10 P1 | P1 |
| 6     | 2.3, 4.3, 4.4, 10 P2 | P2 |
| 7     | 8, 10 P2 | P2 |
