# Feature Spec: AI Image Interpretation

Status: Implemented (foundation + UI + Discord alerts)  
Last updated: 2026-03-13

---

## Implemented Scope

RTSPanda now includes a full first-pass object-detection product loop:

1. Per-camera scheduling and AI tracking controls
2. Async frame sampling + YOLOv8 inference pipeline
3. Detection event and snapshot persistence
4. Live overlay rendering in camera view
5. Detection event/history browsing in UI
6. Per-camera Discord rich-media webhook alerts

AI inference remains decoupled from live playback path.

---

## Architecture

```
RTSP Camera
   │
   ├─(live path)──────────────► mediamtx ───► HLS ───► Browser Player + Overlay
   │
   └─(AI sample path)─────────► FFmpeg capture (Go)
                                 │
                                 ▼
                           Async queue/workers (Go)
                                 │
                                 ▼
                       FastAPI worker + YOLOv8 (Python)
                                 │
                                 ▼
              detection_events + snapshots + optional Discord webhook
```

---

## Backend Components

- `backend/internal/cameras/*`
  - Stores per-camera AI and Discord settings
  - Validates confidence/webhook/cooldown inputs
- `backend/internal/detections/manager.go`
  - Starts samplers only when `tracking_enabled`
  - Filters detections by confidence and labels
  - Persists events with frame dimensions
  - Calls alert notifier hook
- `backend/internal/notifications/discord.go`
  - Sends rich embeds with attached snapshot
  - Supports mention text and per-camera cooldown
- `backend/internal/db/migrations/004_camera_tracking_discord.sql`
  - Adds new camera AI/Discord columns
  - Adds `frame_width` and `frame_height` to detection events

---

## Worker Components

- `ai_worker/app/main.py`
  - `GET /health`
  - `POST /detect`
  - Returns detections plus `image_width` and `image_height`

---

## Frontend Components

- `frontend/src/components/CameraForm.tsx`
  - Tracking config UI
  - Discord alert config UI
- `frontend/src/components/VideoPlayer.tsx`
  - Live overlay bounding boxes and labels
  - Scaled correctly using source frame dimensions
- `frontend/src/pages/CameraView.tsx`
  - Tracking toggle
  - Overlay toggle
  - Test detection action
  - Detection history panel with snapshot thumbnails

---

## API Surface

Existing detection endpoints (active):

- `GET /api/v1/detections/health`
- `POST /api/v1/cameras/{id}/detections/test-frame`
- `POST /api/v1/cameras/{id}/detections/test`
- `GET /api/v1/detection-events?limit=...&camera_id=...`
- `GET /api/v1/detection-events/{id}/snapshot`

Camera CRUD now accepts/returns:

- `tracking_enabled`
- `tracking_min_confidence`
- `tracking_labels`
- `discord_alerts_enabled`
- `discord_webhook_url`
- `discord_mention`
- `discord_cooldown_seconds`

---

## Runtime Config

- `FFMPEG_BIN`
- `DETECTOR_URL`
- `DETECTION_SAMPLE_INTERVAL_SECONDS`
- `DETECTION_WORKERS`
- `DETECTION_QUEUE_SIZE`
- `YOLO_MODEL` (worker)
- `YOLO_CONFIDENCE` (worker baseline)

---

## Non-Goals / Remaining Work

- Multi-frame object ID tracking (track IDs across frames)
- Retention/TTL for old detection snapshots and events
- Notification retry/backoff queueing
- Advanced suppression windows and rules engine integration
