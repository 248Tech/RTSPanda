# Feature Spec: AI Image Interpretation (Phase 1 Foundation)

Status: Implemented foundation
Last updated: 2026-03-12

---

## Implemented Scope

RTSPanda now has a first-stage async object-detection pipeline:

1. Go backend samples frames from configured RTSP cameras with FFmpeg.
2. Samples are queued internally and processed asynchronously.
3. A separate Python FastAPI worker runs YOLOv8 inference.
4. Structured detections are returned and persisted as detection events.
5. Detection snapshots are stored on disk and linked to event records.

This is foundation infrastructure only. No rules engine, tracking, or user notifications are included.

---

## Architecture

```
RTSP Camera
   │
   ├─(independent live path)────► mediamtx ───► HLS ───► Browser
   │
   └─(AI sample path)───────────► FFmpeg frame capture (Go)
                                  │
                                  ▼
                            Async in-memory queue (Go)
                                  │
                                  ▼
                        Python AI worker (FastAPI + YOLOv8)
                                  │
                                  ▼
                      detection_events (SQLite) + snapshots on disk
```

Key rule preserved: AI inference is not in the viewer/live stream path.

---

## Backend Components

- `backend/internal/detections/manager.go`
  - Camera sampler lifecycle (`OnCameraAdded/Updated/Removed`)
  - Async queue and worker goroutines
  - Detector HTTP calls
  - Event persistence and snapshot retention policy
- `backend/internal/detections/capture.go`
  - FFmpeg single-frame extraction from camera RTSP URL
- `backend/internal/detections/client.go`
  - HTTP client for detector `/detect` and `/health`
- `backend/internal/detections/repository.go`
  - SQLite CRUD for `detection_events`

---

## Worker Components

- `ai_worker/app/main.py`
  - `GET /health`
  - `POST /detect` (multipart image upload + optional `camera_id`/`timestamp`)
  - YOLO model loaded once on startup via Ultralytics
- `ai_worker/Dockerfile`
  - Containerized detector service for compose deployment

---

## Storage

- Snapshot root: `DATA_DIR/snapshots/detections/{camera_id}/`
- SQLite events table: `detection_events`
  - `id`
  - `camera_id`
  - `object_label`
  - `confidence`
  - `bbox_json`
  - `snapshot_path`
  - `raw_payload` (optional)
  - `created_at`

Snapshots are kept only for frames with detections (non-detection frames are deleted).

---

## API Surface (Go backend)

- `GET /api/v1/detections/health`
- `POST /api/v1/cameras/{id}/detections/test-frame`
- `POST /api/v1/cameras/{id}/detections/test`
- `GET /api/v1/detection-events?limit=100&camera_id=...`
- `GET /api/v1/detection-events/{id}/snapshot`

---

## Config

- `FFMPEG_BIN` (default `ffmpeg`)
- `DETECTOR_URL` (default `http://127.0.0.1:8090`)
- `DETECTION_SAMPLE_INTERVAL_SECONDS` (default `30`)
- `DETECTION_WORKERS` (default `2`)
- `DETECTION_QUEUE_SIZE` (default `128`)
- camera-level override: `detection_sample_seconds` (nullable)

---

## Non-Goals (Still Out of Scope)

- Notifications (Discord/email/push)
- Motion zones or tracking
- Rules engine / suppression windows
- Live overlay rendering
- Clip generation tied to detections
- Custom model training
