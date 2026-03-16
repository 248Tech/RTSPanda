# RTSPanda v0.0.4 — ONNX Runtime, Stream Reliability, Integrations

This release reduces runtime memory usage, improves stream recovery, and adds new integrations for AI captions and external recording sync.

---

## Highlights

- `ai-worker` now runs ONNX Runtime + numpy instead of PyTorch/ultralytics at runtime.
- Multi-stage `ai_worker/Dockerfile` exports YOLOv8 ONNX during build and ships a lean runtime image.
- Stream reliability improvements:
  - keepalive checks against mediamtx API + HLS playlist
  - single-camera stream reset
  - full stream network reset
- New per-camera ignore zones:
  - polygon editor in camera view
  - normalized polygon validation in backend
  - migration `006_tracking_ignore_polygons.sql`
- New Integrations settings:
  - OpenAI Vision captions for Discord snapshot alerts
  - external recording sync to Local Server, Dropbox, Google Drive, OneDrive, Proton Drive
- New multi-camera page (up to 4 simultaneous streams) with batch screenshot and Discord actions.
- Added Chrome PiP extension package and install docs in `/downloads`.

---

## API Additions

- `GET /api/v1/settings`
- `PUT /api/v1/settings`
- `POST /api/v1/cameras/{id}/stream/reset`
- `POST /api/v1/streams/reset`

---

## Docs Added/Updated

- `README.md` (v0.0.4 summary + config additions)
- `docs/EXTERNAL_VIDEO_STORAGE.md`
- `RELEASE_NOTES_v0.0.4.md` (this file)

---

## Operational Notes

- `docker-compose.yml` now includes memory limits and lower detection worker/queue defaults for constrained hosts.
- `RCLONE_BIN` can be used to specify a non-default `rclone` path.
- OpenAI captions are opt-in and require an API key configured in Integrations.
