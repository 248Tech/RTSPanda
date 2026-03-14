# RTSPanda v0.0.3 — YOLO Reliability + Discord Media Controls

This release improves detector stability and expands Discord alerting/media workflows.

---

## Highlights

- Improved YOLO detector reliability in Docker deployments.
- Added verbose backend + AI worker logging around detection requests/results.
- Added per-camera Discord trigger controls for detection and interval screenshots.
- Added manual camera-view actions:
  - `Screenshot to Discord`
  - `Record to Discord`
- Added richer clip/media handling with format fallback:
  - `webm`, `webp`, `gif`
- Updated user/docs wording to YOLO-first alerting, while keeping legacy alert-rule APIs for compatibility.

---

## Detection Reliability Fixes

- Added FFmpeg RTSP timeout option fallback for older builds:
  - try `-rw_timeout`
  - fallback to `-timeout`
  - fallback to no timeout flag
- Added detector URL fallback behavior in backend detection client.
- Improved detector request failure aggregation/logging across fallback URLs.
- Added runtime dependencies to the AI worker image to prevent startup/runtime crashes in container environments.
- Ensured multipart image uploads include an explicit image content type expected by the detector.

---

## New Camera Discord Controls

Per-camera fields added:

- `discord_trigger_on_detection`
- `discord_trigger_on_interval`
- `discord_screenshot_interval_seconds`
- `discord_include_motion_clip`
- `discord_motion_clip_seconds`
- `discord_record_format`
- `discord_record_duration_seconds`

Database migration:

- `backend/internal/db/migrations/005_discord_triggers.sql`

---

## New API Endpoints

- `POST /api/v1/cameras/{id}/discord/screenshot`
- `POST /api/v1/cameras/{id}/discord/record`

---

## Notes

- Legacy alert-rule webhook APIs remain available for external workflows.
- Retention cleanup, delivery retry/backoff, and event pagination remain on the post-release backlog.
