# RTSPanda — TODO

Last updated: 2026-03-09 (audited)

---

## In Progress

*(nothing)*

---

## Ready for Cursor

### TASK-011 — Dockerfile + docker-compose

- **Description:** Multi-stage Docker build + compose file
- **Purpose:** Production deployment path
- **Files to create:**
  - `Dockerfile`
  - `docker-compose.yml`
  - `.dockerignore`
- **Dockerfile stages:**
  1. Node — build frontend
  2. Go — build binary with embedded frontend
  3. Alpine runtime — minimal image with mediamtx binary
- **Acceptance criteria:**
  - `docker compose up` starts the app on port 8080
  - SQLite persists across restarts via volume mount
  - Image size under 150MB
- **Dependencies:** TASK-010
- **Next tool:** Cursor

---

## Ready for Claude (Planning)

### TASK-P02 — Feature Spec: Notification System (Phase 2)

- **Description:** Plan the notification architecture for Discord/email alerts
- **Purpose:** Future-proof the event pipeline
- **Output:** `AI/FEATURES/NOTIFICATIONS.md`
- **Next tool:** Claude (when Phase 2 begins)

---

## Ready for Aider

*(none yet — will populate as codebase grows)*

---

## Done

### TASK-010 — Embed frontend into Go binary ✓

- **Description:** Bundle built React app in the Go binary via `embed`
- **Implemented:**
  - `backend/internal/api/static.go` — `//go:embed web`; serve files from `web` with SPA fallback (unknown paths → index.html); cache headers for `assets/`; fallback HTML when frontend not built
  - `backend/internal/api/web/.gitkeep` — keeps `web` in repo for `go:embed`; build copies `frontend/dist` into `web`
  - `backend/internal/api/router.go` — register static handler on `/` (after API and `/hls/`)
  - `Makefile` (repo root) — build frontend, copy to `backend/internal/api/web`, build Go binary
  - `build.ps1` (repo root) — Windows: same steps, outputs `backend\rtspanda.exe`
  - `.gitignore` — `backend/internal/api/web/*` with `!.gitkeep` so built assets are not committed
- **Acceptance criteria:** Full build (make build or .\build.ps1) produces a single binary that serves API, HLS proxy, and SPA; no external static files required at runtime.

---

### TASK-008 — Frontend: Video player (HLS) ✓

- **Description:** hls.js video player + single camera view page
- **Implemented:**
  - `frontend/src/components/VideoPlayer.tsx` — hlsUrl prop; hls.js with Safari native HLS fallback; loading/playing/error states; Retry re-loads source; waiting/playing events for buffering
  - `frontend/src/pages/CameraView.tsx` — fetches camera + stream via `getCamera(id)` and `getStreamInfo(id)`; renders VideoPlayer, camera name, RTSP URL, StatusBadge; Back + Settings
  - `frontend/src/App.tsx` — renders `CameraView` at `/cameras/:id` (placeholder removed)
  - `frontend/vite.config.ts` — dev proxy `/api` and `/hls` → `http://localhost:8080`
  - `hls.js` added as dependency
- **Acceptance criteria:** Open camera from dashboard → single-camera view; player uses HLS URL from backend; loading and error states; Safari native HLS fallback; `npm run build` passes.

---

### TASK-009 — Frontend: Settings / Camera management UI ✓

- **Description:** UI for adding, editing, and removing cameras
- **Implemented:**
  - `frontend/src/pages/Settings.tsx` — list cameras, Add/Edit/Delete with modals; loading, empty, error states
  - `frontend/src/components/CameraForm.tsx` — Name, RTSP URL, Enabled toggle; create/edit modes; validation (required name, required URL, rtsp:// prefix)
  - `frontend/src/components/Modal.tsx` — overlay + panel, Escape/click-outside to close
  - `frontend/src/components/ConfirmDialog.tsx` — title, message, confirm/cancel (danger variant for delete)
- **Acceptance criteria:** Add camera → appears in UI; edit name/URL/enabled; delete with confirmation; `npm run build` passes. Navigation Dashboard ↔ Settings already wired in TASK-007.

---

### TASK-P01 — UX Design: Camera Dashboard ✓

- `AI/UXDesign/DASHBOARD_UX.md` created — color palette, grid breakpoints, component breakdown, interaction model, scope limits

---

### TASK-007 — Frontend: Camera dashboard ✓

- **Description:** Build the main dashboard showing all cameras in a grid
- **Implemented:**
  - `frontend/src/App.tsx` — app shell, navbar, minimal path-based routing
  - `frontend/src/pages/Dashboard.tsx` — fetch cameras on load, 30s poll, empty state or grid
  - `frontend/src/components/CameraGrid.tsx` — responsive grid (1/2/3/4 cols)
  - `frontend/src/components/CameraCard.tsx` — name, status badge, 16:9 placeholder, click → `/cameras/:id`
  - `frontend/src/components/StatusBadge.tsx` — online/offline/connecting
  - `frontend/src/components/EmptyState.tsx` — "No cameras" + Add Camera → Settings
  - Tailwind theme extended with UX palette (base, card, status colors, accent)
- **Acceptance criteria:** Dashboard renders camera cards from API; empty state when no cameras; status badge reflects online/offline/connecting; card click navigates to single-camera placeholder (TASK-008 for full view).

---

### TASK-005 — mediamtx integration ✓

- `backend/internal/streams/mediamtx.go` — `proc` struct: config generation, start/stop, API add/remove path, `findBinary()`
- `backend/internal/streams/manager.go` — `Manager`: Start/Stop/OnCamera*, watchdog goroutine, API-first with reload fallback
- `backend/internal/streams/health.go` — `StreamStatus()` polls `/v3/paths/list`
- `mediamtx/mediamtx.yml.tmpl` — reference template
- `.gitignore` — ignores mediamtx binary, data dir, dist
- Binary not found → graceful disabled mode (app runs, streaming offline)
- `sourceOnDemand: yes` on all paths ✓

---

### TASK-006 — Stream status API endpoint ✓

- `backend/internal/api/streams.go` — `GET /api/v1/cameras/{id}/stream` → `{hls_url, status}`
- `backend/internal/api/router.go` — `StreamManager` interface, HLS reverse proxy (`/hls/` → `:8888`), new route
- `backend/internal/api/cameras.go` — create/update/delete now notify stream manager
- `backend/cmd/rtspanda/main.go` — creates Manager, starts with camera list, graceful cancel
- Smoke tested: stream endpoint returns `{hls_url, status: "offline"}` ✓, 404 on missing camera ✓

---

### TASK-003 — SQLite schema + migrations ✓

- `backend/internal/db/db.go` — `db.Open()`, WAL mode, `_migrations` table, idempotent runner via `embed.FS`
- `backend/internal/db/migrations/001_initial.sql` — `cameras` + `settings` tables
- Verified: DB file created on start, migrations idempotent across restarts

---

### TASK-004 — Camera REST API (CRUD) ✓

- `backend/internal/cameras/model.go` — `Camera`, `CreateInput`, `UpdateInput`, sentinel errors
- `backend/internal/cameras/repository.go` — SQL CRUD, returns `[]Camera{}` (never nil)
- `backend/internal/cameras/service.go` — UUID generation, validation, business logic
- `backend/internal/api/cameras.go` — all 5 handlers, 404/400/500 mapped correctly
- `backend/internal/api/router.go` — `CameraService` interface, Go 1.22 pattern routing
- `backend/cmd/rtspanda/main.go` — wires DB → repo → service → router, `DATA_DIR` env
- Smoke tested: health ✓, POST camera ✓, GET list ✓, UUIDs ✓, 404 ✓

---

### TASK-001 — Initialize Go backend project ✓

- `backend/go.mod` — module `github.com/rtspanda/rtspanda`, Go 1.26
- `backend/cmd/rtspanda/main.go` — HTTP server on :8080, graceful shutdown
- `backend/internal/api/router.go` — `GET /api/v1/health` → `{"status":"ok"}`
- `backend/internal/db/db.go` — stub (implemented in TASK-003)
- `go build ./...` passes clean

---

### TASK-002 — Initialize React + Vite + TypeScript frontend ✓

- `frontend/` — Vite react-ts scaffold (Vite 7, Node 24)
- `frontend/tailwind.config.ts` — content paths configured
- `frontend/src/index.css` — Tailwind directives added
- `frontend/src/api/cameras.ts` — fully typed API client (all endpoints)
- `frontend/src/pages/Dashboard.tsx` — stub page
- `frontend/src/pages/Settings.tsx` — stub page
- `npm run build` passes, `dist/` produced, Tailwind CSS included
