# RTSPanda — Handoff

## Latest Handoff: 2026-03-14 — v0.0.3 Reliability + Discord Trigger Expansion

### Summary

This handoff captures release work for `v0.0.3`:

- Fixed detector reliability failures in Docker deployments.
- Added verbose YOLO/detector logging for troubleshooting.
- Expanded per-camera Discord trigger/media controls.
- Added manual Discord media actions from camera view.
- Updated user-facing docs to YOLO-first alerting language.

---

### Key Issues Resolved

1. Detection worker failures from FFmpeg option incompatibility:
- Older FFmpeg builds rejected `-rw_timeout`.
- Added fallback logic: `rw_timeout` -> `timeout` -> no timeout option.

2. Detector request failures to `ai-worker`:
- Added detector URL fallback list in backend client.
- Improved request failure aggregation/logging across fallback URLs.

3. Docker AI worker startup crashes:
- Added missing runtime libs in AI worker image (`libxcb1`, GL libs, etc.).

4. Multipart detection upload compatibility:
- Explicit image content type now set on detector multipart form part.

---

### Feature Additions

- New camera config fields (migration `005_discord_triggers.sql`):
  - `discord_trigger_on_detection`
  - `discord_trigger_on_interval`
  - `discord_screenshot_interval_seconds`
  - `discord_include_motion_clip`
  - `discord_motion_clip_seconds`
  - `discord_record_format`
  - `discord_record_duration_seconds`
- Manual endpoints:
  - `POST /api/v1/cameras/{id}/discord/screenshot`
  - `POST /api/v1/cameras/{id}/discord/record`
- Camera view buttons:
  - `Screenshot to Discord`
  - `Record to Discord`
- Notifier media generation fallback:
  - `webm`, `webp`, `gif`

---

### Files Updated In This Release

- `ai_worker/Dockerfile`
- `ai_worker/app/main.py`
- `backend/cmd/rtspanda/main.go`
- `backend/internal/api/detections.go`
- `backend/internal/api/router.go`
- `backend/internal/cameras/model.go`
- `backend/internal/cameras/repository.go`
- `backend/internal/cameras/service.go`
- `backend/internal/db/migrations/005_discord_triggers.sql` (new)
- `backend/internal/detections/capture.go`
- `backend/internal/detections/client.go`
- `backend/internal/detections/manager.go`
- `backend/internal/notifications/discord.go`
- `frontend/src/api/cameras.ts`
- `frontend/src/api/detections.ts`
- `frontend/src/components/CameraForm.tsx`
- `frontend/src/pages/CameraView.tsx`
- `frontend/src/pages/Settings.tsx`
- `README.md`
- `human/USER_GUIDE.md`

---

### Verification Snapshot

- Backend compile/tests for changed packages: pass.
- Frontend build: pass.
- Docker services healthy together:
  - `rtspanda` up
  - `rtspanda-ai-worker` up
- Manual API verification:
  - test detection endpoint works
  - manual Discord screenshot/record endpoint works

---

### Remaining Risks / Next Work

1. Add retention cleanup for snapshots/events.
2. Add detection history pagination/filtering.
3. Add Discord retry/backoff and failed-delivery visibility.
4. Add integration tests around migration `005` and notifier media modes.
