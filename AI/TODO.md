# RTSPanda — TODO

Last updated: 2026-03-14

---

## In Progress

### TASK-AI-003 — Detection event retention + cleanup policy

- **Description:** Add configurable retention/cleanup for old detection snapshots/events.
- **Purpose:** Prevent unbounded disk growth while preserving recent evidence.
- **Acceptance criteria:**
  - Configurable max age for snapshots/events.
  - Scheduled cleanup job removes expired files and rows safely.
  - Cleanup failures are logged without crashing runtime.
- **Next tool:** Aider

---

## Ready for Aider

### TASK-AI-004 — Event filters and pagination

- **Description:** Expand detection event APIs with cursor/offset pagination and richer filters.
- **Purpose:** Prepare scalable timeline/history UX and lower payload sizes.
- **Dependencies:** TASK-AI-001, TASK-AI-002
- **Next tool:** Aider

### TASK-AI-006 — Discord delivery resilience

- **Description:** Add retry/backoff and failed-delivery visibility for Discord webhook sends.
- **Purpose:** Reduce silent alert loss on transient network failures.
- **Dependencies:** TASK-AI-002
- **Next tool:** Aider

### TASK-AI-008 — Detection startup orchestration hardening

- **Description:** Improve startup sequencing/health checks between `rtspanda` and `ai-worker`.
- **Purpose:** Reduce transient `connection refused` / DNS lookup failures during container startup.
- **Dependencies:** TASK-AI-001
- **Next tool:** Aider

---

## Ready for Claude (Planning)

### TASK-AI-P01 — Multi-frame tracking architecture

- **Description:** Plan lightweight object ID tracking across frames (beyond per-frame detections).
- **Output:** `AI/FEATURES/OBJECT_TRACKING.md`
- **Dependencies:** TASK-AI-002
- **Next tool:** Claude

### TASK-AI-P02 — Rules engine spec

- **Description:** Define rule model for object filters, schedules, suppression windows, and actions.
- **Output:** `AI/FEATURES/AI_RULES_ENGINE.md`
- **Dependencies:** TASK-AI-002
- **Next tool:** Claude

---

## Ready for Cursor

### TASK-AI-007 — Detection history UX scaling

- **Description:** Add timeline filters and lazy loading to camera event history panel.
- **Purpose:** Keep camera page performant on long-running deployments.
- **Dependencies:** TASK-AI-004
- **Next tool:** Cursor

---

## Done

### TASK-AI-001 — Object detection foundation (Phase 1) ✓

- FFmpeg-based frame sampling pipeline
- Async detector queue + workers in Go
- Python FastAPI YOLOv8 worker (`/detect`, `/health`)
- SQLite detection event storage + snapshot persistence
- Detection APIs and Docker wiring

### TASK-AI-002 — Detection UI + per-camera controls + Discord alerts ✓

- Per-camera settings:
  - tracking toggle
  - sample interval
  - confidence threshold
  - label filters
  - Discord webhook + mention + cooldown
- Live overlay rendering in camera view
- Detection event/history panel with snapshots
- Discord rich-media webhook notifications from detection pipeline

### TASK-AI-003A — v0.0.3 reliability + Discord trigger/media expansion ✓

- Detector URL fallback attempts with improved failure logs
- FFmpeg RTSP timeout compatibility fallback (`rw_timeout` handling)
- AI worker Docker runtime dependency fixes
- Verbose detector + YOLO request/result logging
- New per-camera Discord trigger/media fields + migration `005`
- Manual camera view actions: screenshot/record to Discord
