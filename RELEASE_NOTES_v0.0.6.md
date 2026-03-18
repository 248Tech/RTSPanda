# RTSPanda v0.0.6 - Performance, Observability, System Monitoring, Multi-View UX

Tag: `v0.0.6`  
Diff: [v0.0.5...v0.0.6](https://github.com/248Tech/RTSPanda/compare/v0.0.5...v0.0.6)

---

## Highlights

- Dashboard stream status now uses a shared mediamtx path cache and batch status endpoint.
- API responses under `/api/` now support gzip compression.
- New observability surface: request logging, Prometheus-compatible `/metrics`, and system stats API.
- New readiness endpoint for deeper health checks: `GET /api/v1/health/ready`.
- Multi-view UX improvements: inline add-camera card + per-panel remove button.
- Settings now includes a live `System` tab for uptime, memory, goroutines, request count, and network counters.

---

## Backend and API Changes

- Added request body limit (256 KB) for camera create/update payloads:
  - `backend/internal/api/cameras.go`
- Router and middleware updates:
  - `backend/internal/api/router.go`
  - Added API gzip wrapping, request logging wrapper, `/metrics`, and system routes.
- New API endpoints:
  - `GET /api/v1/cameras/stream-status`
  - `GET /api/v1/system/stats`
  - `GET /api/v1/health/ready`
  - `GET /metrics`
- New observability implementation files:
  - `backend/internal/api/metrics.go`
  - `backend/internal/api/middleware.go`
  - `backend/internal/api/sysinfo.go`

---

## Streams and Performance

- Added stream path-list cache with 3-second TTL (shared across callers):
  - `backend/internal/streams/cache.go`
- Stream manager now exposes:
  - `StreamStatusMap(cameraIDs []string)`
  - `IsReady() bool`
- Stream keepalive now uses cached path list, reducing mediamtx API pressure.
- mediamtx config now enables native metrics:
  - `metrics: yes`
  - `metricsAddress: 127.0.0.1:9998`

---

## Database Changes

- Added migration:
  - `backend/internal/db/migrations/008_cameras_index.sql`
- Index added:
  - `idx_cameras_order` on `cameras(position, created_at)`

---

## Frontend Changes

- Dashboard now batch-loads camera list + stream status map in parallel:
  - `frontend/src/pages/Dashboard.tsx`
  - `frontend/src/api/cameras.ts`
  - `frontend/src/components/CameraGrid.tsx`
  - `frontend/src/components/CameraCard.tsx`
- App now uses route-level lazy loading and suspense fallback:
  - `frontend/src/App.tsx`
- Multi-view enhancements:
  - Inline add-camera card picker
  - Per-panel quick remove (`✕`)
  - `frontend/src/pages/MultiCameraView.tsx`
- New System tab + API integration:
  - `frontend/src/pages/Settings.tsx`
  - `frontend/src/api/sysinfo.ts`

---

## AI Docs and Filesystem Review

Reviewed and updated AI planning/handoff/docs to match this release:

- Updated:
  - `AI/CURRENT_FOCUS.md`
  - `AI/DECISIONS.md`
  - `AI/HANDOFF.md`
  - `AI/TODO.md`
- Added:
  - `AI/AGENTIC-PLATFORM-EXPANSION-GUIDE.md`

These docs now reflect:
- v0.0.6 shipped scope
- deferred platform expansion roadmap
- explicit risks and next-priority items

---

## User Documentation

- Updated `README.md` to v0.0.6 summary and compare link.
- Updated `human/USER_GUIDE.md` with system monitoring and metrics coverage.

---

## Validation

- Backend:
  - `cd backend && go build ./...` (pass)
- Frontend:
  - `cd frontend && npm run build` (pass)
