# RTSPanda — Handoff

## Latest Handoff: 2026-03-14 — RAM Overhaul: 4 GB Target

### Summary

Reduced total runtime memory from ~1–2 GB down to ~650–800 MB, making RTSPanda
comfortable on a 4 GB host with headroom for recordings and OS cache.

**Root cause:** The AI worker ran PyTorch/ultralytics at runtime (~600–1500 MB).
The fix: multi-stage Docker build exports the model to ONNX at build time;
the runtime image has only `onnxruntime` (no PyTorch, no ultralytics).

### Memory Budget (after)

| Component        | Before        | After         |
|------------------|---------------|---------------|
| AI worker        | 600–1500 MB   | 150–250 MB    |
| Go backend       | ~80 MB        | ~80 MB        |
| mediamtx         | ~150 MB       | ~100 MB       |
| OS/Docker        | ~400 MB       | ~400 MB       |
| **Total**        | **1.2–2.1 GB**| **730–830 MB**|

Docker memory caps: 512 MB each service (enforced via `deploy.resources.limits`).

### Files Changed

- `ai_worker/app/main.py` — Complete rewrite. Replaced `ultralytics.YOLO` with
  `onnxruntime.InferenceSession`. Implements letterbox preprocessing, raw ONNX
  output decoding, NMS, and un-letterbox postprocessing in pure numpy.
  Also limits onnxruntime thread pools (`intra=2, inter=1`) to avoid over-spawning.
- `ai_worker/requirements.txt` — Removed `ultralytics`. Added `onnxruntime==1.21.0`,
  `numpy>=1.24,<3`. No torch dependency at all.
- `ai_worker/Dockerfile` — Multi-stage build:
  - **Stage 1 (exporter):** `python:3.12-slim` + ultralytics → exports `yolov8n.onnx`.
    This stage is ~2 GB during build but is NOT shipped.
  - **Stage 2 (runtime):** `python:3.12-slim` + onnxruntime only. Final image ~200 MB.
  - Build arg `YOLO_MODEL_NAME` selects the model (default: `yolov8n`).
  - uvicorn forced to `--workers 1` (model is not safe to share across forks).
- `ai_worker/export_model.py` — New utility for non-Docker users to export a `.pt`
  model to `.onnx` manually.
- `docker-compose.yml` — Removed `YOLO_MODEL` (now baked into image). Reduced
  `YOLO_MAX_DETECTIONS` 500→100. Added `GOMEMLIMIT: 200MiB` for Go backend.
  Changed `DETECTION_WORKERS: 1`, `DETECTION_QUEUE_SIZE: 32`.
  Added `deploy.resources.limits.memory: 512m` to both services.
- `backend/internal/streams/mediamtx.go` — `hlsSegmentCount` 7→3.
  Reduces HLS buffer from 14 s to 6 s per active stream (~30 MB saved per camera).

### Breaking Changes

1. **Model is now baked into the image at build time.**
   - `YOLO_MODEL` env var no longer has any effect at runtime.
   - To change models: update `YOLO_MODEL_NAME` build arg in `docker-compose.yml`
     and rebuild (`docker compose build ai-worker`).
   - Supported: any YOLOv8 COCO model (`yolov8n`, `yolov8s`, `yolov8m`, etc.)
   - Note: larger models use more RAM; `yolov8n` is the right choice for 4 GB hosts.

2. **`YOLO_IMAGE_SIZE` env var is removed.** Inference size is determined by the
   exported model (640×640 for default yolov8n export).

3. **`YOLO_AGNOSTIC_NMS` env var is removed.** NMS is now always class-agnostic
   (one NMS pass over all detections, independent of class). This matches the most
   common use-case and simplifies the inference path.

### Verification

```bash
# Rebuild images (stage 1 downloads torch + exports model — takes a few minutes)
docker compose build

# Start and check health
docker compose up -d
docker compose logs ai-worker
# Should see: "ai_worker: model ready input=images infer_size=640 ..."

# Test detection endpoint
curl -X POST http://localhost:8090/detect -F "image=@/path/to/test.jpg"
```

---

## Previous Handoff: 2026-03-14 — UI Redesign "Operator Dark"

### Summary

Full frontend visual redesign. No backend changes. All functionality preserved.

**Design direction:** Dense, cinematic, operator-grade. Zinc-based dark palette. Left sidebar navigation. Pill-shaped status badges. Skeleton loading. Feature indicators on camera cards.

### Files Changed

- `frontend/tailwind.config.ts` — New zinc palette, Inter font, shadow tokens
- `frontend/src/index.css` — Inter font import, thin scrollbar style
- `frontend/src/App.tsx` — Left icon sidebar (56px fixed), active indicator, click-outside removed from old navbar
- `frontend/src/components/StatusBadge.tsx` — Pill badges with bg tint + ring; "Live" label
- `frontend/src/components/CameraCard.tsx` — Overlaid status badge, feature icon bar (record/YOLO/Discord), grid texture
- `frontend/src/components/Modal.tsx` — Backdrop blur, rounded-xl, modal shadow, click-outside-to-close
- `frontend/src/components/EmptyState.tsx` — SVG icon, dashed border container, no emoji
- `frontend/src/pages/Dashboard.tsx` — Skeleton loaders, active/total count pill, refined header

### Verification

- Frontend build: run `npm run build` in `frontend/`
- No backend changes — API contracts unchanged

---

## Previous Handoff: 2026-03-14 — v0.0.3 Reliability + Discord Trigger Expansion

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
