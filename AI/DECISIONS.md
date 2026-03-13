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
