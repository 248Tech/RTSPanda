# RTSPanda v0.0.5 — Frigate Alert Provider + In-App Guides

Tag: `v0.0.5`  
Diff: [v0.0.4...v0.0.5](https://github.com/248Tech/RTSPanda/compare/v0.0.4...v0.0.5)

---

## Highlights

- Added **provider-based Discord detection alerts** per camera:
  - `yolo` (RTSPanda built-in YOLOv8)
  - `frigate` (external Frigate event ingestion)
- Added **Frigate webhook endpoint**:
  - `POST /api/v1/frigate/events`
- Added new in-app **Guides** page with practical setup walkthroughs:
  - Lorex NVR port-forwarding
  - Tailscale setup
  - Lorex RTSP URL retrieval from login/web UI flow
- Added **Support the Developer** links to `https://248tech.com/donate`.

---

## Backend / API Updates

- Added Frigate ingestion handler:
  - `backend/internal/api/frigate.go`
  - Parses Frigate webhook payloads and forwards detection alerts to matching cameras.
  - Matches cameras by provider `frigate` and `frigate_camera_name` (or camera name fallback).
  - Ignores non-`new` Frigate event types.
- Registered new route:
  - `POST /api/v1/frigate/events`
- Extended notifier API contract to support external detection event dispatch.
- Updated Discord notification flow:
  - Added source-aware detection notifications (`YOLOv8` vs `Frigate` labels in payloads).
  - Ensured YOLO-origin notifications are skipped for cameras configured with provider `frigate`.
- Added optional Frigate snapshot fetch support via env:
  - `FRIGATE_BASE_URL`
  - If configured, event snapshots are fetched and attached to Discord alerts when available.

---

## Database / Camera Model Updates

- Added migration:
  - `backend/internal/db/migrations/007_discord_detection_provider.sql`
- New camera fields:
  - `discord_detection_provider` (default `yolo`)
  - `frigate_camera_name` (default empty)
- Updated camera model, service validation, and repository persistence/scan logic:
  - `backend/internal/cameras/model.go`
  - `backend/internal/cameras/service.go`
  - `backend/internal/cameras/repository.go`

---

## Frontend Updates

- Added new page:
  - `frontend/src/pages/Guides.tsx`
- Added sidebar navigation route for Guides:
  - `frontend/src/App.tsx`
- Added donate/support links in UI:
  - Sidebar support icon
  - Guides page support button
- Extended camera API types for provider fields:
  - `frontend/src/api/cameras.ts`
- Updated camera setup form:
  - Provider selection (`YOLOv8` or `Frigate`)
  - Optional `Frigate camera name`
  - Source-aware trigger wording
  - `frontend/src/components/CameraForm.tsx`
- Updated settings and camera badges to reflect provider-aware alerting:
  - `frontend/src/pages/Settings.tsx`
  - `frontend/src/pages/CameraView.tsx`

---

## Documentation Updates

- Updated README release section and feature/config/API notes:
  - `README.md`
- Updated user guide for provider-based alerts and Frigate endpoint/env:
  - `human/USER_GUIDE.md`

---

## Validation

- Backend checks:
  - `go test ./...` (pass)
- Frontend checks:
  - `npm run build` (pass)

