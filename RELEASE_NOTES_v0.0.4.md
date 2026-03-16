# RTSPanda v0.0.4 - YOLO UI Polishing, Security, UI, Reliability, Connectivity

Tag: `v0.0.4`  
Diff: [v0.0.3...v0.0.4](https://github.com/248Tech/RTSPanda/compare/v0.0.3...v0.0.4)

## YOLO UI Polishing

- Added per-camera ignore-zone editing using polygon drawing from live camera frames.
- Added ignore-zone rendering in camera view video overlays.
- Improved YOLO camera configuration UX and error guidance in forms.
- Improved detection feedback in camera view actions and status messages.

## Security Updates

- Added strict backend validation for ignore-zone polygon structure and coordinates.
- Added validation and normalization for integrations settings input.
- Added centralized OpenAI settings service with key-set/clear handling.
- Added provider-specific validation rules for external storage configuration.

## UI Updates

- Added multi-camera view (up to 4 cameras) with batch screenshot and Discord actions.
- Added `Settings -> Integrations` UI for OpenAI captions and external storage sync.
- Updated app shell navigation to include multi-view and extension download access.
- Added dashboard `Reset Network` control for stream recovery workflows.
- Added bundled Chrome PiP extension package and install instructions.

## Reliability Updates

- Migrated AI worker runtime from PyTorch/ultralytics to ONNX Runtime + numpy.
- Added multi-stage AI worker Docker build with ONNX export at build time.
- Added stream keepalive health checks (mediamtx API + HLS reachability).
- Added per-camera stream reset and full-stream reload API endpoints.
- Added async manual Discord recording trigger flow with improved backend logging.
- Added conservative Docker memory/runtime defaults for constrained hosts.

## Connectivity

- Added external video storage sync service for:
  - Local Server (NAS/SMB/NFS path)
  - Dropbox (rclone)
  - Google Drive (rclone)
  - OneDrive (rclone)
  - Proton Drive (rclone)
- Added app settings API endpoints:
  - `GET /api/v1/settings`
  - `PUT /api/v1/settings`
- Added stream control API endpoints:
  - `POST /api/v1/cameras/{id}/stream/reset`
  - `POST /api/v1/streams/reset`

## Docs Updated

- `README.md` (v0.0.4 summary + config updates)
- `docs/EXTERNAL_VIDEO_STORAGE.md`
- `extensions/chrome-rtspanda-pip/README.md`
- `frontend/public/downloads/rtspanda-chrome-pip-extension-install.md`
