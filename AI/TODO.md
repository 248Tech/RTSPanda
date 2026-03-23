# RTSPanda — TODO

Last updated: 2026-03-18

---

## Done

### TASK-PERF-001 — Performance + Observability Overhaul (v0.0.6) ✓

- **Completed:** 2026-03-18
- **Description:** Raspberry Pi 4 / 4 GB performance pass and full observability layer.
- **Changes:**
  - Stream status 3-second path list cache — N cameras = 1 mediamtx API call (was N)
  - `GET /api/v1/cameras/stream-status` batch endpoint
  - Gzip compression on all `/api/` JSON responses
  - DB index on `cameras(position, created_at)` — migration `008`
  - 256 KB request body limits on camera create/update
  - React.lazy + Suspense code splitting — initial JS bundle 831 kB → 202 kB (76% reduction)
  - `loggingMiddleware` — every API request logged with method/path/status/duration
  - `/metrics` Prometheus-compatible endpoint (stdlib only, no external library)
  - mediamtx native metrics enabled (`metrics: yes`, port 9998)
  - `GET /api/v1/system/stats` — RTSPanda process uptime, goroutines, heap, RSS, bandwidth
  - `GET /api/v1/health/ready` — extended health check with DB ping + mediamtx probe
  - Settings → System tab with live-refreshing RAM/CPU/network stats panel
  - Multi-view: `+` Add Camera card with picker dropdown; `✕` remove button per panel

### TASK-UI-002 — Multi-View UX Improvements (v0.0.6) ✓

- **Completed:** 2026-03-18
- **Description:** Multi-view grid add/remove camera UX and "Add Camera" button fix.
- **Changes:**
  - `+` Add Camera card in grid (dashed border, aspect-video, picker dropdown for unselected cameras)
  - `✕` remove button on each camera panel header (remove from view without checkbox panel)
  - Click-outside handler closes picker
  - "Add Camera" button in Settings → Cameras always visible (was disappearing after first camera)

---

### TASK-MEM-001 — RAM Overhaul: 4 GB Target ✓

- **Completed:** 2026-03-14
- **Description:** Reduced runtime memory footprint to support 4 GB hosts.
- **Root cause:** PyTorch/ultralytics in the AI worker consumed 600–1500 MB at runtime.
- **Changes:**
  - `ai_worker/app/main.py`: onnxruntime-based inference, pure numpy pre/postprocessing
  - `ai_worker/requirements.txt`: removed ultralytics; added onnxruntime==1.21.0
  - `ai_worker/Dockerfile`: multi-stage build — PyTorch only in builder, not runtime
  - `ai_worker/export_model.py`: export helper for non-Docker usage
  - `docker-compose.yml`: memory limits (512m each), GOMEMLIMIT=200MiB, DETECTION_WORKERS=1
  - `backend/internal/streams/mediamtx.go`: hlsSegmentCount 7→3
- **Result:** AI worker 600–1500 MB → 150–250 MB. Total cluster ~730–830 MB.

### TASK-UI-001 — UI Redesign: "Operator Dark" ✓

- **Completed:** 2026-03-14
- **Description:** Full frontend visual redesign for a more modern, information-dense look.
- **Changes:**
  - `tailwind.config.ts`: New zinc-based palette (zinc-950/900/800), blue-600 accent, Inter font family, modal/glow shadows
  - `index.css`: Inter font import via Google Fonts, custom thin scrollbar styling
  - `App.tsx`: Fixed left icon sidebar (56px) replaces top navbar; active nav indicator; icon-only with tooltips
  - `StatusBadge.tsx`: Pill-shaped badges with tinted bg + ring instead of bare dot+text; "Live" label for online
  - `CameraCard.tsx`: Status badge overlaid on thumbnail (top-right); feature indicator icons (recording=red, YOLO=violet, Discord=indigo); subtle grid texture; hover ring
  - `Modal.tsx`: `backdrop-blur-sm` overlay, `rounded-xl`, `shadow-modal`, click-outside-to-close
  - `EmptyState.tsx`: SVG camera icon in rounded box replaces emoji; "+" icon in CTA button
  - `Dashboard.tsx`: Skeleton loading cards (no spinner), "X/Y active" pill counter, refined multi-view button

---

## Android No-Docker Initiative (Active Sprint)

**Spec:** `AI/FEATURES/ANDROID_NO_DOCKER_REMOTE_YOLO.md`
**Decisions locked:** DEC-021, DEC-022, DEC-023

---

### TASK-AND-A — Android binary + startup script

- **Description:** Make `rtspanda` binary work in Termux without Docker dependency. Write `scripts/android-up.sh`.
- **Owner:** Aider
- **Dependencies:** None
- **Deliverables:**
  - `scripts/android-up.sh` — sets env, launches binary, handles Termux path conventions, runs `termux-wake-lock`
  - Verified `GOARCH=arm64 GOOS=linux` binary works in Termux
  - mediamtx ARM64 download documented
  - FFmpeg Termux package confirmed compatible with existing `detections/capture.go` args
- **Acceptance criteria:**
  - `./scripts/android-up.sh` starts RTSPanda Pi-mode on Termux
  - `/api/v1/health` returns 200
  - `/api/v1/health/ready` passes DB + mediamtx probes
  - One camera can be added and stream viewed from LAN browser
- **Next tool:** Aider

---

### TASK-AND-B — Thermal monitor goroutine

- **Description:** Implement `backend/internal/thermal/` package with four-band detection, hysteresis timers, and event pub/sub.
- **Owner:** Aider
- **Dependencies:** None (independent of TASK-AND-A)
- **Deliverables:**
  - `backend/internal/thermal/monitor.go` — reads `/sys/class/thermal/`, CPU-load fallback, disabled mode
  - `ThermalBand` enum: Normal, Warm, Hot, Critical
  - Hysteresis logic: 5-min cool-down for Hot/Critical transitions, 3-min for Warm
  - `Subscribe(chan ThermalBandEvent)` interface for detection manager integration
  - Thermal monitor starts only on arm64 + Pi-mode (or `THERMAL_MONITOR_ENABLED=true`)
  - `GET /api/v1/system/stats` gains `thermal_band` field
- **Acceptance criteria:**
  - Simulated band changes emit correct log lines (WARN/ERROR/CRITICAL)
  - `thermal_band` field appears in system stats response
  - Discord alert fires on Hot entry when camera has webhook
- **Next tool:** Aider

---

### TASK-AND-C — Detection throttle integration

- **Description:** Wire `ThermalBandEvent` from thermal monitor into the detection sampler manager.
- **Owner:** Aider
- **Dependencies:** TASK-AND-B (thermal monitor package must exist)
- **Deliverables:**
  - Detection manager subscribes to `ThermalBandEvent` channel
  - Warm: sample interval floor → max(configured, 30 s)
  - Hot: pause all detection goroutines, drain in-flight FFmpeg captures
  - Critical: drain detection goroutines, block new stream open requests
  - Recovery: log resume event; auto-resume only if `THERMAL_AUTO_RESUME=true`
  - `GET /api/v1/detections/health` gains `status: suspended_thermal` + `sample_interval_floor` fields
- **Acceptance criteria:**
  - At simulated Warm: detection health shows `sample_interval_floor: 30`
  - At simulated Hot: detection health shows `status: suspended_thermal`
  - At simulated Critical: all detection stopped; streaming still functional
  - After recovery with `THERMAL_AUTO_RESUME=false`: detection remains suspended until restart
- **Next tool:** Aider

---

### TASK-AND-D — 3-Node operator tooling

- **Description:** Write startup scripts and verify end-to-end 3-node flow (Android hub + Pi relay + AI server).
- **Owner:** Aider
- **Dependencies:** TASK-AND-A
- **Deliverables:**
  - `scripts/android-3node-hub.sh` — starts RTSPanda in viewer mode; exposes mediamtx RTSP re-streams on port 8554
  - `scripts/pi-detection-relay.sh` — starts RTSPanda Pi-mode on Pi; camera URLs point to Android re-streams; accepts `ANDROID_HUB_IP` env var
  - Documents camera URL format: `rtsp://<android-ip>:8554/<camera-name>`
- **Acceptance criteria:**
  - Following `docs/android-no-docker.md` 3-node section reaches working state
  - Two browser sessions: Android UI (`http://<android>:8080`) and Pi UI (`http://<pi>:8081`) both accessible
  - Pi detection events appear in Pi dashboard
- **Next tool:** Aider

---

### TASK-AND-E — Documentation alignment

- **Description:** Ensure all docs accurately reflect Android no-Docker path and remove conflicting guidance.
- **Owner:** Claude / Cursor
- **Dependencies:** TASK-AND-A (scripts must exist before docs reference them)
- **Deliverables:**
  - `docs/android-no-docker.md` — operator walkthrough (written in this planning session, may need script path updates after TASK-AND-A)
  - `docs/cluster-mode.md` — intermediary Pi pattern section (written in this planning session)
  - `docs/raspberry-pi.md` — remove conflicting "full local AI on Pi" guidance; align with DEC-018
  - `README.md` — Android no-Docker row in setup matrix
- **Acceptance criteria:**
  - No docs reference Docker for Android deployment
  - No docs suggest local YOLO/AI on Pi or Android as a supported path
  - README setup matrix includes Android row with link to `docs/android-no-docker.md`
- **Next tool:** Cursor (for README/docs prose); Claude (for architecture alignment)

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

## Ready for Claude (Implementation — Platform Expansion)

### TASK-EXP-001 — WebRTC live mode with HLS fallback

- **Spec:** `AI/AGENTIC-PLATFORM-EXPANSION-GUIDE.md` Phase 2
- **Description:** Add WHEP WebRTC playback via mediamtx, automatic HLS fallback on failure.
- **Backend:** Enable `webrtc: yes` in mediamtx config; proxy `/webrtc/` → port 8889; extend stream API with `webrtc_url` + `preferred_protocol`.
- **Frontend:** `WebRTCPlayer.tsx` using `RTCPeerConnection` + WHEP negotiation; `VideoPlayer.tsx` tries WebRTC first, falls back to HLS within 2–4 s.
- **Pi benefit:** Sub-second latency; no HLS buffering overhead.
- **Estimated effort:** 2–3 days.
- **Next tool:** Claude (implementation)

### TASK-EXP-002 — ONVIF discovery + PTZ controls

- **Spec:** `AI/AGENTIC-PLATFORM-EXPANSION-GUIDE.md` Phase 3
- **Description:** WS-Discovery scan, import discovered cameras, PTZ pan/tilt/zoom UI.
- **Backend:** `backend/internal/onvif/` package; migration `009_onvif_ptz.sql`; PTZ API routes.
- **Frontend:** Discover button in Settings; PTZ overlay in CameraView.
- **Estimated effort:** 3–5 days.
- **Next tool:** Claude (implementation)

### TASK-EXP-003 — Rules engine (CEL) + MQTT output

- **Spec:** `AI/AGENTIC-PLATFORM-EXPANSION-GUIDE.md` Phase 4
- **Description:** Programmable event rules using CEL expressions; MQTT publish on match.
- **Backend:** `github.com/google/cel-go`, `paho.mqtt.golang`; migration `009_automation_rules.sql`; rules CRUD API.
- **Frontend:** Automation tab in Settings; rule editor form with expression field.
- **Estimated effort:** 3–5 days.
- **Next tool:** Claude (implementation)

### TASK-EXP-004 — Auth proxy integration

- **Spec:** `AI/AGENTIC-PLATFORM-EXPANSION-GUIDE.md` Phase 5
- **Description:** Trusted forward-auth proxy support (oauth2-proxy / Authelia style).
- **Backend:** `backend/internal/api/authproxy.go`; env vars `AUTH_PROXY_ENABLED`, `AUTH_PROXY_TRUSTED_CIDRS`, etc.; `GET /api/v1/me`.
- **Artifacts:** `docker-compose.auth.yml`, `docs/AUTH_PROXY.md`.
- **Estimated effort:** 1 day.
- **Next tool:** Claude (implementation)

### TASK-PERF-002 — React Query / SWR + pagination

- **Spec:** `AI/AGENTIC-PERFORMANCE-PLAN.md` Phase 6
- **Description:** Replace manual fetch/polling with React Query or SWR; add cursor pagination to `GET /api/v1/cameras`.
- **Estimated effort:** 2 days.
- **Next tool:** Cursor

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
