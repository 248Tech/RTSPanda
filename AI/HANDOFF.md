# RTSPanda — Handoff

## Latest Handoff: 2026-03-12 — Object Detection Foundation Implemented

### Summary

Implemented the first-priority AI/object-detection foundation only:
- FFmpeg frame sampling from configured RTSP cameras
- Async dispatch queue in Go backend
- Separate Python FastAPI YOLOv8 worker boundary
- Structured detection result handling
- Detection event + snapshot persistence
- Minimal detection API and health surface

No unrelated UI redesign or full notification/rules/tracking feature set was added.

### Files Changed

- `backend/cmd/rtspanda/main.go`
- `backend/internal/api/router.go`
- `backend/internal/api/cameras.go`
- `backend/internal/api/detections.go` (new)
- `backend/internal/cameras/model.go`
- `backend/internal/cameras/repository.go`
- `backend/internal/cameras/service.go`
- `backend/internal/db/migrations/003_detection_foundation.sql` (new)
- `backend/internal/detections/model.go` (new)
- `backend/internal/detections/repository.go` (new)
- `backend/internal/detections/client.go` (new)
- `backend/internal/detections/capture.go` (new)
- `backend/internal/detections/manager.go` (new)
- `ai_worker/app/main.py` (new)
- `ai_worker/app/__init__.py` (new)
- `ai_worker/requirements.txt` (new)
- `ai_worker/Dockerfile` (new)
- `docker-compose.yml`
- `Dockerfile`
- `README.md`
- `AI/TODO.md`
- `AI/DECISIONS.md`
- `AI/FEATURES/AI_IMAGE_INTERPRETATION.md` (new)

### What Works

- Camera schema supports optional per-camera detection sampling override (`detection_sample_seconds`).
- Backend starts detection manager with:
  - configurable global sample interval
  - async queue (`DETECTION_QUEUE_SIZE`)
  - detector worker concurrency (`DETECTION_WORKERS`)
- FFmpeg captures snapshots under `DATA_DIR/snapshots/detections/{camera_id}`.
- Snapshot jobs are dispatched asynchronously to `DETECTOR_URL`.
- Python worker returns structured YOLOv8 detections (`label`, `confidence`, `bbox`, `timestamp`, `camera_id`).
- Detections are persisted into SQLite `detection_events`.
- Detection snapshots are linked via `snapshot_path`.
- New API endpoints are available:
  - `GET /api/v1/detections/health`
  - `POST /api/v1/cameras/{id}/detections/test-frame`
  - `POST /api/v1/cameras/{id}/detections/test`
  - `GET /api/v1/detection-events`
  - `GET /api/v1/detection-events/{id}/snapshot`
- Docker compose now includes `ai-worker` and backend `DETECTOR_URL` wiring.
- Live viewer path remains independent from AI path.

### What Remains

- Real-camera integration smoke tests for detection cadence and event quality.
- Retention/cleanup policy for old detection snapshots/events.
- Event pagination/filtering improvements for future UI.
- Tracking/rules/notifications/UI layers on top of new foundation.

### Risks / Warnings

- `ffmpeg` must be available in runtime (`FFMPEG_BIN`), or scheduled sampling degrades.
- First `ai-worker` container build can be slow/heavy due YOLO/PyTorch dependencies.
- Worker-down state is tolerated (detection health degrades), but no retries/backoff policy beyond queue drop behavior exists yet.
- Existing repo had unrelated uncommitted changes before this implementation; they were preserved.

### Next Recommended Tool

Cursor (for end-to-end manual validation and any UI-safe verification tooling), then Aider for retention/pagination backend follow-ups.

### Suggested Next Prompt

“Run an end-to-end validation pass of the new detection foundation: start docker compose, add a test RTSP camera, call the test frame/test detection endpoints, confirm detection events + snapshots persist, and document any gaps with proposed minimal fixes.”
