# RTSPanda — Architecture Decisions

This file records key decisions made during planning. Before changing anything listed here, review the rationale. If a decision needs revisiting, add a "Revisit" note rather than silently overwriting.

---

## DEC-001 — mediamtx as stream relay (not FFmpeg directly)

**Decision:** Use mediamtx as the RTSP relay sidecar instead of spawning FFmpeg processes.

**Rationale:**
- mediamtx is a single Go binary — no system dependency hell
- Handles RTSP → HLS and RTSP → WebRTC natively
- Built-in on-demand stream activation (`sourceOnDemand`)
- Actively maintained, LAN-optimized
- Much simpler than writing FFmpeg process management
- Future WebRTC path is already in mediamtx — no rework needed

**Implications:**
- mediamtx binary must be bundled into the Docker image
- Go backend must manage mediamtx as a subprocess
- mediamtx config is generated dynamically by the backend

**Status:** Decided. Do not revisit without strong reason.

---

## DEC-002 — HLS for Phase 1, WebRTC for Phase 2

**Decision:** Phase 1 uses HLS via hls.js. WebRTC comes in Phase 2.

**Rationale:**
- HLS works in all browsers without any special setup
- HLS via mediamtx requires zero transcoding for compatible streams
- WebRTC requires signaling server implementation — adds complexity
- Phase 1 goal is "it works" not "minimum latency"
- mediamtx supports WebRTC so the upgrade path is clear

**Implications:**
- Latency in Phase 1 will be 2–6 seconds (HLS buffering)
- This is acceptable for a homelab camera viewer
- Phase 2 will add WebRTC for sub-second latency use cases

**Status:** Decided.

---

## DEC-003 — SQLite only (no Postgres)

**Decision:** SQLite is the only database for RTSPanda.

**Rationale:**
- Target users are homelabbers — they don't want to run a DB server
- Camera metadata is tiny — SQLite handles it trivially
- Single file = simple backup (just copy the file)
- One-command Docker deployment doesn't need external DB

**Implications:**
- No connection pooling needed
- Write concurrency is not a concern (low-traffic app)
- Backups are trivially simple

**Status:** Decided. May revisit only if multi-instance deployments become a goal.

---

## DEC-004 — No authentication in Phase 1

**Decision:** Phase 1 has no authentication. The app is LAN-first.

**Rationale:**
- Adding auth before core features work is premature complexity
- Target users run this on a private LAN or behind a VPN
- Auth complexity would slow down getting a working product
- Middleware stubs can be added so auth is easy to add in Phase 2

**Implications:**
- Do not expose Phase 1 directly to the public internet
- README must warn users about this clearly
- Route handlers should be wrapped in a middleware chain so auth can be injected later without handler changes

**Status:** Decided. Auth is explicitly Phase 2.

---

## DEC-005 — Frontend embedded in Go binary

**Decision:** The React frontend is embedded into the Go binary using `go:embed`.

**Rationale:**
- Single binary deployment is a core design goal
- Eliminates need for a separate static file server
- Simplifies Docker image structure
- Standard Go practice for single-binary web apps

**Implications:**
- Build process must: 1) build frontend, 2) build Go binary
- Hot reload in development uses Vite dev server + Go API separately
- Production = single binary serving everything on port 8080

**Status:** Decided.

---

## DEC-006 — `sourceOnDemand` enabled for all streams

**Decision:** All mediamtx stream paths use `sourceOnDemand: true`.

**Rationale:**
- Camera streams must not stay open when nobody is watching
- Keeps CPU/network usage at zero when the app is idle
- Respects camera resources (some cameras have connection limits)
- Aligns with "minimal resource usage" design goal

**Implications:**
- First viewer sees a 1–3 second startup delay (stream warming up)
- Stream closes ~10 seconds after last viewer disconnects
- This is expected and acceptable behavior

**Status:** Decided.

---

## DEC-007 — Modular monolith (not microservices)

**Decision:** RTSPanda is a modular monolith — one process, one binary.

**Rationale:**
- Target deployment is a homelab — not Kubernetes
- Microservices add operational complexity that contradicts the product's goals
- Go's package system provides sufficient modularity without process separation
- One process = one log stream, one health check, one restart

**Implications:**
- mediamtx runs as a subprocess, not a separate container
- All future features (notifications, AI) will be modules within the monolith
- If scale ever demands it, modules can be extracted later — but not prematurely

**Status:** Decided.

---

## DEC-008 — AI detector runs as optional sidecar worker (HTTP boundary)

**Decision:** Keep RTSPanda as the main modular monolith, but run YOLO inference in a lightweight Python worker reachable over HTTP.

**Rationale:**
- Ultralytics/YOLO ecosystem is Python-native and faster to ship for v1 detection.
- Keeps Go backend simple and focused on orchestration/persistence.
- Clear boundary allows future swap/replacement of detector implementation.
- Inference failures remain isolated from live stream handling.

**Implications:**
- Backend dispatches frames asynchronously to `DETECTOR_URL`.
- Docker compose includes `ai-worker` service by default.
- If worker is down, detection health degrades but streaming continues.

**Status:** Decided for detection foundation.

---

## DEC-009 — Frame sampling uses FFmpeg pull from camera RTSP (not viewer pipeline taps)

**Decision:** Use FFmpeg single-frame extraction from configured camera RTSP URLs for initial detection sampling.

**Rationale:**
- Fastest path to a reliable sampling primitive.
- Does not couple detection to HLS segment internals.
- Preserves separation from viewer path and avoids browser-side impacts.

**Implications:**
- Requires `ffmpeg` availability (`FFMPEG_BIN`) in runtime environments.
- Camera-level override via `detection_sample_seconds`; global fallback via env.
- Sampling failures are logged and do not block stream serving.

**Status:** Decided for v1 foundation.

---

## DEC-010 — Persist events per detection; retain snapshots only when detections exist

**Decision:** Insert one DB event row per detected object and keep captured snapshots only for frames that produced detections.

**Rationale:**
- Avoids continuously storing non-actionable frames.
- Keeps storage growth controlled while preserving evidence for real events.
- Provides a simple event model for future rules/notifications/UI timelines.

**Implications:**
- `detection_events` stores label/confidence/bbox/snapshot path/raw payload.
- Empty-detection frames are deleted after inference.
- Future tracking and notification layers will build on this schema.

**Status:** Decided.

---

## DEC-011 — Per-camera tracking controls live on camera entity (not separate rule object)

**Decision:** Store core AI tracking controls directly on each camera record.

**Rationale:**
- Tracking enable/disable and thresholds are operational camera settings.
- Keeps scheduling/filtering path fast and simple in detection manager.
- Avoids introducing rules-engine complexity before retention/pagination hardening.

**Implications:**
- Camera API now owns fields such as `tracking_enabled`, confidence, and label filters.
- Detection sampler lifecycle responds immediately to camera update events.
- Future rules engine can still layer on top without moving core sampling controls.

**Status:** Decided.

---

## DEC-013 — Stream status uses a short-lived in-process cache (not per-request mediamtx calls)

**Decision:** All stream status queries (per-camera and batch) go through a 3-second TTL in-memory cache of the mediamtx `/v3/paths/list` response.

**Rationale:**
- Dashboard previously made N mediamtx API calls per load (one per camera). On a Pi with 4+ cameras this added measurable latency and CPU overhead.
- mediamtx path list is cheap to fetch and changes infrequently (only on camera add/remove or reconnect).
- 3-second TTL gives near-real-time status while capping mediamtx round-trips to at most one per 3 seconds across all callers.

**Implications:**
- Cache is invalidated immediately on `OnCameraAdded`, `OnCameraRemoved`, `OnCameraUpdated`.
- On mediamtx API error, cache returns stale data rather than reporting all cameras offline (fail-open).
- Both `StreamStatus(id)` (single) and `StreamStatusMap(ids)` (batch) share the same cache object.

**Status:** Decided. TTL is a constant in `cache.go` — adjust if needed.

---

## DEC-014 — Prometheus metrics use stdlib atomic counters (no external client library)

**Decision:** `/metrics` is implemented with `sync/atomic` counters exposed in Prometheus text format. No `github.com/prometheus/client_golang` dependency.

**Rationale:**
- The Prometheus client library adds ~5 MB to the binary and significant import graph complexity.
- RTSPanda's metric surface is small: request counts, durations, network bytes, stream checks, Discord calls.
- Prometheus text format is a simple line protocol trivially hand-written with `fmt.Fprintf`.
- Keeps the Go binary small (important for Pi SD card / RAM constraints).

**Implications:**
- Histograms are not implemented — only counters and a single avg duration gauge.
- If richer metric cardinality (per-route labels, per-camera stream gauges) is needed later, add the Prometheus library then.
- mediamtx exposes its own full Prometheus metrics at port 9998 — scrapers can hit both endpoints.

**Status:** Decided. Add library only if label cardinality or histogram precision becomes a clear need.

---

## DEC-015 — System stats use /proc/self/status for RSS on Linux; runtime.ReadMemStats for heap everywhere

**Decision:** `GET /api/v1/system/stats` reads physical RAM from `/proc/self/status` (VmRSS) on Linux (the primary deploy target = Raspberry Pi). On Windows (dev), it falls back to Go heap stats with `rss_bytes: 0`.

**Rationale:**
- `runtime.ReadMemStats()` reports Go heap, not total process RSS. On a Pi you want to know total memory consumption including CGo, SQLite pages, and OS buffers.
- `/proc/self/status` is stable, zero-dependency, and available on all Linux kernels.
- The Windows fallback is acceptable because monitoring is primarily a Pi production concern.

**Implications:**
- `rss_bytes` is 0 in local Windows development. Frontend shows heap as fallback (labeled clearly).
- RAM bar in the UI is hardcoded to 4 GB denominator — matches the target Raspberry Pi 4 SKU.

**Status:** Decided. Adjust 4 GB denominator if the target hardware changes.

---

## DEC-016 — Frontend routes are lazy-loaded; hls.js is excluded from initial bundle

**Decision:** All page-level components are wrapped in `React.lazy` + `Suspense`. This splits the single 831 kB bundle into per-route chunks, keeping the initial load at ~202 kB.

**Rationale:**
- hls.js alone is ~500 kB. Loading it on the dashboard or settings page where no video plays wastes bandwidth and parse time (especially on Pi-served connections).
- Code splitting is a one-time change with no ongoing maintenance cost.
- `<PageSpinner>` Suspense fallback provides a smooth loading experience.

**Implications:**
- First navigation to each page incurs a small chunk fetch (cached after first visit).
- Any code added to page components is automatically tree-shaken into that page's chunk.
- If a new shared component is added to `App.tsx` directly, it will be in the initial bundle — keep `App.tsx` lean.

**Status:** Decided.

---

---

## DEC-017 — Three deployment modes as a first-class runtime concept

**Decision:** RTSPanda has exactly three deployment modes (`pi`, `standard`, `viewer`), resolved at startup via `RTSPANDA_MODE` or auto-detected from `runtime.GOARCH`. Each mode gates specific subsystems.

**Rationale:**
- Prior architecture had implicit behavior changes via individual env vars (`AI_MODE`, `AI_WORKER_URL`), which left room for misconfiguration and unclear capability expectations.
- A single mode enum makes the deployment surface explicit and testable.
- ARM auto-detection removes the most common Pi misconfiguration (running standard mode on Pi).

**Mode capabilities:**

| Mode | YOLO detection | Snapshot AI | Notes |
|------|---------------|-------------|-------|
| `pi` | ✗ disabled | ✔ optional | Default on ARM |
| `standard` | ✔ enabled | ✗ | Default on x86 |
| `viewer` | ✗ disabled | ✗ | No AI of any kind |

**Implications:**
- `internal/mode/mode.go` is the authority on mode resolution.
- `main.go` consults `deployMode.AIInferenceAllowed()` and `deployMode.SnapshotAIAllowed()` before starting each subsystem.
- Forcing `RTSPANDA_MODE=standard` on ARM emits a warning but does not block startup (users may have capable ARM servers).

**Status:** Decided. Do not add new per-feature env vars that circumvent mode gating.

---

## DEC-018 — Raspberry Pi is a viewer and snapshot AI node, not a YOLO inference host

**Decision:** Raspberry Pi is explicitly unsupported as a real-time YOLO inference host. This constraint is enforced in code, documentation, scripts, and error messages. There is no "experimental AI on Pi" path.

**Rationale:**
- YOLOv8n ONNX inference requires ~400–600 MB RAM at runtime. A Pi 4 (4 GB) running the full stack (Go backend, mediamtx, 4 RTSP streams, SQLite) is already at its practical memory ceiling.
- ONNX Runtime on arm64 runs at 3–8 FPS on a Pi 4 CPU — not viable for real-time alerting.
- Thermal throttling degrades performance further under sustained load.
- Messaging "experimental AI on Pi" sets false expectations and produces user frustration at root causes that are fundamental, not fixable.

**Enforcement points:**
- `internal/mode/mode.go` — `AIInferenceAllowed()` returns false for `ModePi`
- `scripts/pi-up.sh` — `full` mode is blocked with an explicit error message
- `docker-compose.yml` — `rtspanda-pi` service sets `RTSPANDA_MODE=pi`
- `README.md` — contains a clear statement in the deployment section header

**Status:** Non-negotiable. Do not add YOLO inference paths for Pi hardware.

---

## DEC-019 — Snapshot Intelligence Engine as the Pi AI replacement

**Decision:** Pi mode's AI capability is the Snapshot Intelligence Engine: interval-based JPEG capture → external vision API (Claude or OpenAI) → structured events → Discord alerts.

**Rationale:**
- Cloud vision APIs (GPT-4o-mini, claude-haiku) handle complex scene understanding that YOLO cannot (e.g., "Amazon driver detected" vs "person detected").
- API latency (1–5 seconds) is acceptable for interval-based alerting — this is not a continuous tracking use case.
- Cost per alert is negligible for homelab usage (< $0.001 per call for haiku/mini).
- No GPU, no model downloads, no Python process — Pi handles only frame capture (FFmpeg) and HTTP dispatch.

**Output contract:**
- Emits `detection_events` rows with identical schema to YOLO events.
- Discord alerts use the same `NotifyExternalDetectionEvents` path with a `sourceLabel` of "Snapshot AI (Claude)" or "Snapshot AI (OpenAI)".
- The UI cannot and does not need to distinguish snapshot AI events from YOLO events.

**Constraints:**
- Not real-time. One API call per camera per interval tick.
- Not suitable for sub-second response or continuous tracking.
- Requires `SNAPSHOT_AI_ENABLED=true` and a valid `SNAPSHOT_AI_API_KEY`.
- Positioning: "smart alerting via AI interpretation, not real-time detection."

**Status:** Decided. Configuration via env vars; Settings UI integration is a follow-up task.

---

## DEC-020 — Snapshot AI uses external vision APIs, not a local model

**Decision:** The Snapshot Intelligence Engine sends frames to hosted APIs (Anthropic Claude or OpenAI). There is no local vision model for Pi.

**Rationale:**
- Local vision models capable of scene description (LLaVA, InternVL, etc.) require 4–8 GB RAM — not viable on Pi.
- External APIs are available on-demand, require no local resources, and produce higher-quality structured descriptions than YOLO labels.
- The cloud dependency is explicit and opt-in (requires API key). Core viewing and recording work without it.

**Provider support:**
- `claude` — Uses `claude-haiku-4-5-20251001` (fast, cheap, capable vision)
- `openai` — Uses `gpt-4o-mini` (comparable cost and capability)

**Implications:**
- Snapshot AI only activates if `SNAPSHOT_AI_ENABLED=true` and `SNAPSHOT_AI_API_KEY` is set.
- No API key = no AI on Pi (core viewer still works fully).
- Prompt is user-configurable (`SNAPSHOT_AI_PROMPT`) for property-specific detection.

**Status:** Decided. Do not add local model paths for Pi — they are not viable hardware targets.

---

## DEC-012 — Discord rich alerts are emitted from detection manager via notifier boundary

**Decision:** Send Discord alerts from backend detection pipeline through a notifier interface.

**Rationale:**
- Alerting should happen near event creation to guarantee payload consistency.
- Interface boundary keeps transport-specific logic out of detection core.
- Enables future notifier expansion (email/webhook bus) with minimal manager changes.

**Implications:**
- New `notifications` package with Discord webhook implementation.
- Per-camera cooldown applied server-side to reduce alert spam.
- Notification failures are logged but do not block detection persistence.

**Status:** Decided.
