# RTSPanda — Handoff

## Latest Handoff: 2026-03-18 — v0.0.6 Performance + Observability + System Monitor

### Summary

This session delivered three categories of work targeting a **4 GB Raspberry Pi** host:

1. **UI fixes** — Multi-view Add Camera card with picker dropdown; ✕ remove button per panel
2. **Performance overhaul** — Eliminated N+1 mediamtx API calls, gzip compression, code splitting (76% JS bundle reduction), DB index, request body limits
3. **Observability + system monitoring** — Prometheus `/metrics`, system stats API, Settings → System tab, extended health check

All changes compile clean: `go build ./...` and `npm run build` both pass.

---

### Bugs Fixed

**mediamtx connection refused (`127.0.0.1:8888`):**
- Root cause: `mediamtx/` directory existed but contained only `mediamtx.yml.tmpl` — no `mediamtx.exe` binary.
- `Manager` correctly entered disabled mode, but the HLS reverse proxy in `router.go` was always registered unconditionally, so every `/hls/` request hit nothing.
- Fix is operational (not code): place `mediamtx.exe` from [bluenviron/mediamtx v1.12.3](https://github.com/bluenviron/mediamtx/releases/tag/v1.12.3) into `mediamtx/`.

---

### UI Changes

**`frontend/src/pages/MultiCameraView.tsx`**
- Added `showPicker` state and `pickerRef` for the add card.
- Added `addCamera()` callback (adds to selectedIds, closes picker).
- Added click-outside handler to close picker on outside click.
- Renders a `+` card (dashed border, `aspect-video` height) after selected camera panels whenever slots remain (`selectedIds.length < 4`) and unselected cameras exist. Clicking opens an inline dropdown of available cameras.
- Added `✕` remove button to each camera panel header — removes camera from view without using the checkbox panel.

---

### Performance Changes (from AGENTIC-PERFORMANCE-PLAN.md)

**Phase 1 — Stream status cache (P0)**
- New `backend/internal/streams/cache.go`: `pathListCache` with 3-second TTL, RW-mutex, stale-on-error fallback, `invalidate()` for immediate refresh.
- `health.go` rewritten: `StreamStatus()` and new `StreamStatusMap()` both use the cache — N cameras = 1 mediamtx API call.
- `manager.go`: `pathCache` field added to `Manager`; `invalidate()` called on every `OnCameraAdded/Removed/Updated`; keepalive's `listPaths` call replaced with `m.pathCache.get()`.
- New `IsReady() bool` method on Manager (used by health/ready endpoint).

**Phase 1 — Batch status endpoint (P0)**
- `GET /api/v1/cameras/stream-status` added to `streams.go`: fetches camera list from DB, calls `StreamStatusMap()` once, returns `{ camera_id: { status, hls_url } }` map.
- `StreamStatusMap(cameraIDs []string)` added to `StreamManager` interface in `router.go`.

**Phase 2 — Frontend batch usage (P0)**
- `cameras.ts`: added `getStreamStatusMap()` calling the new endpoint.
- `Dashboard.tsx`: now does `Promise.all([getCameras(), getStreamStatusMap()])` — one load = 2 parallel API calls total (was 1 + N serial).
- `CameraCard.tsx`: added `initialStatus?: StreamStatus` prop. When provided, skips mount API call; only polls after mount. Falls back to per-card fetch when no initial status (e.g. in Settings).
- `CameraGrid.tsx`: accepts `statusMap?: StreamStatusMap`, passes `initialStatus` to each card.

**Phase 3 — Gzip, index, limits (P1)**
- `middleware.go`: `gzipMiddleware` wraps `/api/` mux only. Compresses JSON responses with `gzip.BestSpeed` when client sends `Accept-Encoding: gzip`. HLS proxy and static assets excluded.
- `008_cameras_index.sql`: `CREATE INDEX IF NOT EXISTS idx_cameras_order ON cameras(position, created_at)`.
- `cameras.go`: `MaxBytesReader(w, r.Body, 256*1024)` on `handleCreateCamera` and `handleUpdateCamera`.

**Phase 4 — Frontend code splitting (P1)**
- `App.tsx`: all 5 page components wrapped in `React.lazy(() => import(...))` + `<Suspense fallback={<PageSpinner />}>`.
- Initial JS bundle: **831 kB → 202 kB** (76% reduction). hls.js (529 kB) only loads when opening a camera view.
- Separate chunks produced: Dashboard, CameraView, MultiCameraView, Settings, Guides, VideoPlayer/hls.js, Modal, StatusBadge.

---

### Observability Changes (from AGENTIC-PERFORMANCE-PLAN.md Phase 5 + AGENTIC-PLATFORM-EXPANSION-GUIDE.md Phase 1)

**Request logging middleware** (`middleware.go`)
- `loggingMiddleware` wraps entire mux. Logs method, path, status, duration for `/api/` routes.
- `countingResponseWriter` captures status code and response bytes.
- `countingReader` counts request body bytes.
- All counts flow into `appMetrics` atomic counters.

**Prometheus-compatible `/metrics` endpoint** (`metrics.go`)
- Zero external dependencies — pure stdlib + `sync/atomic`.
- Exposes: `rtspanda_http_requests_total`, `_2xx/4xx/5xx`, `_avg_duration_ms`, `rtspanda_network_bytes_in/out`, `rtspanda_stream_health_checks_total`, `rtspanda_discord_webhooks_total`.
- Text exposition format compatible with Prometheus scrapers.

**mediamtx native metrics** (`mediamtx.go` config template)
- Added `metrics: yes` and `metricsAddress: 127.0.0.1:9998` to generated config.
- mediamtx exposes its own Prometheus metrics at `http://127.0.0.1:9998/metrics` when running.

**System stats API** (`sysinfo.go` + `api/sysinfo.ts`)
- `GET /api/v1/system/stats` returns: `uptime_seconds`, `goroutines`, `heap_alloc_bytes`, `heap_sys_bytes`, `rss_bytes` (VmRSS from `/proc/self/status` on Linux; 0 on Windows), `network_bytes_in/out`, `http_requests_total`, `goos`, `goarch`, `num_cpu`.
- RSS gives actual physical RAM usage on Raspberry Pi.

**Extended health check** (`sysinfo.go`)
- `GET /api/v1/health/ready`: pings DB (`PingContext`), probes mediamtx via cache, returns 503 + detail JSON if DB is down.
- Original `GET /api/v1/health` (liveness) unchanged — still returns `{status: ok}` always.

**Settings → System tab** (`Settings.tsx`)
- New `SystemPanel` component: auto-refreshes every 5 seconds.
- Displays: uptime, RSS/heap memory, goroutines, HTTP requests, network in/out.
- RAM usage bar: shows % of 4 GB (Pi-targeted).
- Platform line: `linux/arm64 · 4 CPUs`.

---

### Router/Interface Changes (`router.go`)

- `StreamManager` interface extended with `StreamStatusMap()` and `IsReady()`.
- `DBPinger` interface added; `DBPingerFunc` adapter for `*sql.DB.PingContext`.
- `server` struct gains `db DBPinger` field.
- `NewRouter` gains `db DBPinger` parameter — **breaking change to call site in `main.go`** (updated).
- Route layout: `/metrics` (uncompressed) → gzip wrapped `/api/` mux → `/hls/` proxy → static SPA.
- All routes wrapped with `loggingMiddleware` outermost.

---

### Files Changed

**New files:**
- `backend/internal/streams/cache.go`
- `backend/internal/api/metrics.go`
- `backend/internal/api/middleware.go`
- `backend/internal/api/sysinfo.go`
- `backend/internal/db/migrations/008_cameras_index.sql`
- `frontend/src/api/sysinfo.ts`

**Modified files:**
- `backend/internal/streams/health.go` — use cache, add StreamStatusMap
- `backend/internal/streams/manager.go` — add pathCache, IsReady, invalidation
- `backend/internal/streams/mediamtx.go` — add metrics config
- `backend/internal/api/streams.go` — add handleStreamStatusAll
- `backend/internal/api/cameras.go` — add MaxBytesReader
- `backend/internal/api/router.go` — full rewrite: new interface, middleware, routes
- `backend/cmd/rtspanda/main.go` — pass db to NewRouter
- `frontend/src/api/cameras.ts` — add getStreamStatusMap
- `frontend/src/pages/Dashboard.tsx` — batch fetch, pass statusMap
- `frontend/src/components/CameraCard.tsx` — accept initialStatus
- `frontend/src/components/CameraGrid.tsx` — accept statusMap
- `frontend/src/App.tsx` — React.lazy + Suspense
- `frontend/src/pages/Settings.tsx` — System tab + SystemPanel
- `frontend/src/pages/MultiCameraView.tsx` — Add Camera card + remove button

---

### What Was NOT Implemented (deferred to future sessions)

From `AGENTIC-PLATFORM-EXPANSION-GUIDE.md`:
- **Phase 2 — WebRTC + WHEP**: Requires WHEP frontend client (`RTCPeerConnection`), mediamtx WebRTC config, proxy route, and fallback logic. Estimated 2–3 days.
- **Phase 3 — ONVIF discovery + PTZ**: Requires `github.com/use-go/onvif`, WS-Discovery, new DB migration, PTZ UI. Estimated 3–5 days.
- **Phase 4 — CEL rules engine + MQTT**: Requires `github.com/google/cel-go`, `paho.mqtt.golang`, new tables (009 migration), rules UI. Estimated 3–5 days.
- **Phase 5 — Auth proxy**: Env-based middleware, CIDR validation, `/api/v1/me`, example compose. Estimated 1 day.

From `AGENTIC-PERFORMANCE-PLAN.md`:
- **Phase 6 — React Query / SWR pagination**: Estimated 2 days. Can be done independently.
- **Phase 7 — Scalability docs**: Estimated 2 hours.

---

### Risks / Notes for Next Agent

1. **mediamtx metrics port conflict**: Port 9998 could conflict if something else is bound there. Document as configurable future improvement.
2. **`sourceOnDemand` in mediamtx.go template is `no`**: Despite ADR-006 saying it should be `yes`, the current template has `sourceOnDemand: no`. This pre-dates this session — verify intent before changing.
3. **Windows vs Linux RSS**: `rss_bytes` is always 0 on Windows (dev machine). The Pi will show real values. Frontend correctly displays heap as fallback.
4. **Initial bundle warning**: `VideoPlayer` chunk is 529 kB (hls.js). Can be reduced with dynamic `import('hls.js')` inside VideoPlayer — deferred.
5. **`handleStreamStatusAll` route order**: `GET /api/v1/cameras/stream-status` is registered before `GET /api/v1/cameras/{id}` in the mux — Go 1.22+ pattern matching will handle this correctly since the specific literal path wins.

---

## Previous Handoff: 2026-03-14 — RAM Overhaul: 4 GB Target

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
