# RTSPanda — TODO

Last updated: 2026-03-12

---

## In Progress

### TASK-AI-002 — Validate detector pipeline on real RTSP cameras

- **Description:** End-to-end smoke test against at least one real camera and confirm snapshot/event creation cadence.
- **Purpose:** Verify FFmpeg + YOLO worker behavior in practical network conditions.
- **Acceptance criteria:**
  - Sampled frames captured at configured interval.
  - `POST /api/v1/cameras/{id}/detections/test` returns structured detections.
  - `GET /api/v1/detection-events` returns persisted events with valid snapshot links.
- **Next tool:** Cursor or Aider

---

## Ready for Aider

### TASK-AI-003 — Detection event retention + cleanup policy

- **Description:** Add configurable retention/cleanup for old detection snapshots/events.
- **Purpose:** Prevent unbounded disk growth while preserving recent evidence.
- **Dependencies:** TASK-AI-001
- **Next tool:** Aider

### TASK-AI-004 — Event filters and pagination

- **Description:** Expand detection event APIs with pagination cursors and richer filters.
- **Purpose:** Prepare scalable UI/event timeline consumption.
- **Dependencies:** TASK-AI-001
- **Next tool:** Aider

---

## Ready for Claude (Planning)

### TASK-AI-P01 — Object tracking architecture (post-foundation)

- **Description:** Plan lightweight multi-frame tracking strategy layered on current detection events.
- **Output:** `AI/FEATURES/OBJECT_TRACKING.md`
- **Dependencies:** TASK-AI-001
- **Next tool:** Claude

### TASK-AI-P02 — AI rules engine spec

- **Description:** Define rules model (object filters, confidence thresholds, schedules, suppression windows).
- **Output:** `AI/FEATURES/AI_RULES_ENGINE.md`
- **Dependencies:** TASK-AI-001
- **Next tool:** Claude

### TASK-AI-P03 — Notification architecture spec

- **Description:** Design notifications pipeline (Discord/email) on top of detection events.
- **Output:** `AI/FEATURES/NOTIFICATIONS.md`
- **Dependencies:** TASK-AI-001
- **Next tool:** Claude

### TASK-AI-P04 — Detection UI integration spec

- **Description:** Plan frontend event timeline/snapshot viewer and testing UX for detection health.
- **Output:** `AI/UXDesign/DETECTION_EVENTS_UX.md`
- **Dependencies:** TASK-AI-001
- **Next tool:** Claude

---

## Ready for Cursor

### TASK-AI-005 — Minimal detection events UI wiring

- **Description:** Add non-invasive UI surfaces to list detection events and preview snapshots.
- **Purpose:** Make persisted detection scaffolding visible for manual verification.
- **Dependencies:** TASK-AI-004
- **Next tool:** Cursor

---

## Done

### TASK-AI-001 — Object detection foundation (Phase 1) ✓

- **Implemented:**
  - FFmpeg-based frame sampling pipeline with per-camera override (`detection_sample_seconds`) and global interval env fallback.
  - Async detector boundary with queue + worker goroutines in Go.
  - Python FastAPI YOLOv8 worker (`ai_worker`) with `/detect` + `/health`.
  - SQLite `detection_events` table + repository scaffolding.
  - Snapshot persistence under `DATA_DIR/snapshots/detections/{camera_id}`.
  - Backend APIs:
    - `POST /api/v1/cameras/{id}/detections/test-frame`
    - `POST /api/v1/cameras/{id}/detections/test`
    - `GET /api/v1/detection-events`
    - `GET /api/v1/detection-events/{id}/snapshot`
    - `GET /api/v1/detections/health`
  - Docker compose wiring for optional `ai-worker` service.
- **Notes:** Streaming/viewer path remains decoupled from detection pipeline.
