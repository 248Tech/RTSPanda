# Feature Spec: Android No-Docker Pi-Mode + Remote YOLO

**Status:** Planning complete — ready for Aider/Cursor implementation
**Last updated:** 2026-03-22
**Owner (planning):** Claude
**Owner (implementation):** Aider / Cursor
**Parent initiative:** Platform expansion — extended hardware targets

---

## Problem Statement

RTSPanda Pi-mode currently assumes Docker availability. Android devices (phones, tablets, TV sticks, dedicated NVR hardware running Android) cannot run Docker in standard environments. Android Termux provides a native Linux userspace without requiring root on many devices, enabling Go binaries, mediamtx, and FFmpeg to run directly — but existing deployment instructions have no Android path.

Additionally, Android hardware is prone to thermal throttling under sustained computational load. Continuous FFmpeg frame extraction on a device running mediamtx, SQLite, and the RTSPanda Go binary can cause sustained thermal stress that degrades stream reliability and reduces hardware lifespan.

**Goals:**

1. Define Android (Termux, no Docker) as an officially supported deployment method using Pi-mode behavior.
2. Define a 2-node topology (Android + remote YOLO worker server) covering the majority of cases.
3. Define a 3-node topology (Android + intermediary Pi + remote YOLO worker server) for high-camera-count or thermally-constrained deployments.
4. Define a thermal policy with explicit temperature bands, per-band behavior, and operator-visible signals.
5. Define a decision gate that tells operators when 3-node is required.

**Non-goals:**

- Docker support on Android (out of scope — not viable in Termux without root).
- Local YOLO inference on Android (blocked by DEC-018; thermal constraints apply equally).
- Local vision models on Android (blocked by DEC-020 reasoning — 4–8 GB RAM requirement not practical).
- RTSPanda mobile app (blocked by AI/CURRENT_FOCUS.md "Out of Scope" clause).

---

## Architecture

### 2-Node Topology

**Use when:** Camera count ≤ 3, resolution ≤ 1080p, sample interval ≥ 15 s, device temperature under sustained load typically stays below 55°C.

```
┌──────────────────────────────────────┐     LAN      ┌──────────────────────────────┐
│  Android Device (Termux, no Docker)  │◄────────────►│  AI Server (x86, Docker)     │
│                                      │              │                              │
│  rtspanda (Go binary)                │   HTTP POST  │  ai-worker (FastAPI ONNX)    │
│  ├── mediamtx subprocess             │ ──frames──►  │  port 8090                   │
│  ├── SQLite (DATA_DIR)               │              │                              │
│  ├── FFmpeg (frame capture)          │              └──────────────────────────────┘
│  ├── Snapshot AI (Pi mode)           │
│  └── React UI (embedded)            │
│                                      │
│  RTSPANDA_MODE=pi                    │
│  AI_WORKER_URL=http://<server>:8090  │
└──────────────────────────────────────┘
         ▲             ▲
         │ RTSP        │ RTSP
  Camera 1          Camera 2-3
```

**Data flow:**

```
Camera RTSP
  ↓ (mediamtx subprocess, Android)
HLS segments → Browser UI (hls.js, port 8080)
  ↓ (detection sampler goroutine, per-camera interval)
FFmpeg single-frame extract → temp JPEG
  ↓ (HTTP POST multipart)
remote ai-worker /detect (server LAN)
  ↓
Detection response (labels, confidence, bboxes)
  ↓
Persist detection_events (Android SQLite)
  ↓
Discord alert (if configured)
```

### 3-Node Topology

**Use when:** Decision gate criteria are met (see section below). The intermediary Pi offloads FFmpeg frame-capture and HTTP forwarding duties from Android, reducing Android sustained CPU load and thermal output.

```
┌─────────────────────────────┐
│  Android Device (Termux)    │
│                             │
│  rtspanda (Go binary)       │
│  ├── mediamtx subprocess    │  Re-streams RTSP → LAN
│  ├── SQLite (DATA_DIR)      │  (port 8554 RTSP re-stream)
│  ├── React UI (embedded)    │
│  └── RTSPANDA_MODE=viewer   │  ← Detection disabled on Android
└─────────────────────────────┘
         ▲             ▲
         │ RTSP        │ RTSP
   Camera 1       Camera 2-N
         │
         ▼ RTSP re-stream (mediamtx → LAN port 8554)
┌──────────────────────────────────┐
│  Intermediary Raspberry Pi       │
│                                  │
│  rtspanda (Go binary)            │
│  ├── mediamtx subprocess         │ ← reads Android mediamtx re-streams
│  ├── SQLite (own DATA_DIR)       │
│  ├── FFmpeg (frame capture)      │
│  ├── Snapshot AI (Pi mode)       │
│  └── RTSPANDA_MODE=pi            │
│      AI_WORKER_URL=http://server │
└──────────────────────────────────┘
                │ HTTP POST frames
                ▼
┌──────────────────────────────┐
│  AI Server (x86, Docker)     │
│  ai-worker FastAPI ONNX      │
│  port 8090                   │
└──────────────────────────────┘
```

**Data flow (3-node):**

```
Camera RTSP
  ↓ Android mediamtx ingest
HLS/RTSP re-stream (Android LAN port 8554 or 8888)
  ↓ Pi mediamtx reads re-stream as source camera
Pi FFmpeg frame extract → temp JPEG
  ↓ HTTP POST
remote ai-worker /detect
  ↓
Detection events → Pi SQLite + Pi Discord alerts
```

**Operator note:** In 3-node mode Android and Pi run separate RTSPanda instances with separate databases. The Pi database holds detection events; Android holds camera metadata used for UI and viewing. Camera configs must be maintained in sync manually or via shared env/script. A future Config Sync feature (out of scope here) could automate this.

---

## Decision Gate: When 3-Node Is Required

The 3-node topology is the recommended deployment when **any two** of the following hard criteria apply simultaneously:

| Criterion | Threshold | Reason |
|-----------|-----------|--------|
| Camera count | ≥ 4 cameras | Each camera adds a steady-state FFmpeg goroutine; 4+ creates sustained thermal pressure |
| Resolution | Any camera ≥ 1080p with detection enabled | 1080p JPEG encode/decode adds 30–60% per-frame CPU vs 720p |
| Detection sample interval | < 15 seconds global or per-camera | Below 15 s, FFmpeg spawns overlap with each other on single-core devices |
| Sustained temperature | ≥ 55°C for 10+ continuous minutes at 2-node | Device has already shown thermal pressure under current load |
| Available Pi | Pi on same LAN with < 60% CPU at idle | Worthwhile only if a capable intermediary is available |

**Single criterion override (always 3-node regardless):**

- Sustained temperature reaches ≥ 65°C (Critical band) at 2-node — operator MUST add intermediary Pi or reduce detection load.
- Camera count ≥ 6 on a single Android device.

---

## Thermal Policy

### Temperature Bands and Actions

| Band | Temperature | Detection Behavior | Stream Behavior | Operator Signal |
|------|-------------|-------------------|-----------------|-----------------|
| Normal | < 45°C | Full operation — configured sample intervals respected | Full HLS serving | None |
| Warm | 45–54°C | Sample interval floor raised to max(configured, 30 s) | Full HLS serving | Log WARN: `thermal: warm band — sample interval floored to 30s` |
| Hot | 55–64°C | All detection sampling suspended. Snapshot AI paused. | Full HLS serving continues | Log ERROR: `thermal: hot band — detection suspended` + Discord alert if webhook configured |
| Critical | ≥ 65°C | Detection suspended. mediamtx kept alive for active viewers only. All FFmpeg capture goroutines drained. | Active HLS streams maintained; new stream opens blocked | Log CRITICAL: `thermal: critical band — viewer-only fallback` + Discord alert |

### Hysteresis and Recovery Rules

To prevent mode flapping, recovery requires sustained cool-down:

| Recovery Path | Requirement Before Resuming |
|---------------|------------------------------|
| Critical → Hot | Temperature < 60°C sustained for 5 minutes |
| Hot → Warm | Temperature < 50°C sustained for 5 minutes |
| Warm → Normal | Temperature < 42°C sustained for 3 minutes |

Detection does not auto-resume after Hot or Critical band exit without a configurable `THERMAL_AUTO_RESUME=true` env var (default: `false`). When `false`, operator must issue a health endpoint trigger or restart to resume detection after a Hot/Critical event. This prevents silent re-entry into a thermal spiral.

### Operator-Visible Signals

| Signal | Delivery Method |
|--------|----------------|
| Band change log lines | `rtspanda` stdout (all bands) |
| Detection suspended | `GET /api/v1/detections/health` → `status: suspended_thermal` |
| Current band | `GET /api/v1/system/stats` → `thermal_band: "hot"` (future field) |
| Discord alert | Sent on Hot or Critical entry if camera has webhook configured |
| Recovery notice | Log line when detection resumes (with band + temperature) |

### Temperature Source

Android does not expose `/proc/` thermal zone data in Termux without root. The thermal monitor reads from:

1. `/sys/class/thermal/thermal_zone*/temp` — available in many Termux environments without root (read-only).
2. Fallback: CPU load proxy — if `thermal_zone` is unreadable, use CPU load average as a thermal proxy (> 2.0 load on a 4-core device ≈ Hot equivalent behavior).
3. If neither is readable: thermal policy runs in disabled mode (log warning). All detection proceeds normally, and operator must monitor temperature manually.

Temperature is sampled every 30 seconds by a background goroutine in the Pi-mode detection subsystem.

---

## Runtime Assumptions

| Assumption | Rationale |
|-----------|-----------|
| Termux installed with `pkg install golang ffmpeg` | Go and FFmpeg must be available as native binaries |
| mediamtx binary available at `mediamtx/mediamtx` or `MEDIAMTX_BIN` | Same rule as all other platforms |
| Android device is on stable LAN or Wi-Fi | RTSP and HTTP frame forwarding require reliable connectivity |
| Android Wi-Fi not suspended by power management | Operator must disable Wi-Fi sleep for RTSPanda to remain reachable |
| `DATA_DIR` writable by Termux user | Termux home dir (`~/`) or sdcard path with write permissions |
| Remote ai-worker reachable on LAN port 8090 | Firewall on server must allow inbound 8090 from Android device |
| FFmpeg available on PATH | `pkg install ffmpeg` in Termux; or set `FFMPEG_BIN` env |
| No privileged ports | RTSPanda binds 8080 (or `PORT`); no ports < 1024 |

---

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Android Wi-Fi sleep kills streams | High | Operator must configure Wi-Fi lock; documented in setup guide |
| Thermal policy can't read temp (no `/sys/class/thermal`) | Medium | CPU-load proxy fallback defined; thermal disabled mode logged |
| Two-database inconsistency in 3-node (Android + Pi separate DBs) | Medium | Camera configs must be manually synced; documented limitation. Future: config sync feature. |
| mediamtx re-stream port 8554 may conflict with other apps on LAN | Low | Port configurable via `MEDIAMTX_RTSP_PORT` env; documented |
| Android battery optimizations kill background processes | High | Operator must whitelist Termux from battery optimization; documented in setup guide |
| Pi reads stale streams from Android if Android restarts | Medium | Pi mediamtx sourceOnDemand handles reconnect; acceptable delay |
| No persistent service manager in Termux | Medium | Operator must use `termux-services` or a `tmux`/`screen` session; startup script provided |

---

## Implementation Phases

### Phase A: Android Binary Build and Startup Script (Aider)

**Goal:** Make `rtspanda` binary work in Termux without any Docker dependency.

Tasks:
- Verify Go cross-compilation to `GOARCH=arm64 GOOS=linux` produces a working binary in Termux.
- Write `scripts/android-up.sh` — sets env vars, launches `./rtspanda`, handles Termux path conventions.
- Confirm mediamtx binary (ARM64) download path and document it.
- Confirm FFmpeg Termux package works with existing `detections/capture.go` FFmpeg args.
- Test `RTSPANDA_MODE=pi` on ARM64 Android with remote `AI_WORKER_URL`.

**Acceptance criteria for Phase A:**
- `./scripts/android-up.sh` starts RTSPanda in Pi-mode on a Termux device.
- `/api/v1/health` returns 200.
- `/api/v1/health/ready` passes DB + mediamtx probes.
- One camera can be added and stream viewed in browser.

### Phase B: Thermal Monitor Goroutine (Aider)

**Goal:** Implement thermal band detection and policy enforcement in `backend/internal/thermal/`.

Tasks:
- Create `backend/internal/thermal/monitor.go` — reads `/sys/class/thermal/thermal_zone*/temp`, falls back to CPU load, emits band changes.
- Define `ThermalBand` enum: Normal, Warm, Hot, Critical.
- Implement hysteresis logic with band-specific cool-down timers.
- Expose `GetCurrentBand() ThermalBand` and `Subscribe(chan ThermalBandEvent)` interface.
- Wire into `main.go`: thermal monitor starts only when `RTSPANDA_MODE=pi` and `THERMAL_MONITOR_ENABLED=true` (default true on arm64).

**Acceptance criteria for Phase B:**
- `GET /api/v1/system/stats` returns `thermal_band` field.
- Band transitions logged at correct levels (WARN/ERROR/CRITICAL).
- Discord alert fires on Hot entry when camera has webhook.

### Phase C: Detection Throttle Integration (Aider)

**Goal:** Wire thermal band changes into the detection sampler.

Tasks:
- Detection manager subscribes to `ThermalBandEvent` channel.
- On Warm: raise all per-camera sample intervals to max(configured, 30s).
- On Hot: pause all detection goroutines, drain in-flight FFmpeg captures.
- On Critical: drain detection goroutines, block new stream open requests.
- On recovery (threshold met): log resume event. Resume auto only if `THERMAL_AUTO_RESUME=true`.

**Acceptance criteria for Phase C:**
- At simulated Warm band: `GET /api/v1/detections/health` → `sample_interval_floor: 30`.
- At simulated Hot band: `status: suspended_thermal`.
- At simulated Critical band: all detection goroutines stopped; streaming still works.

### Phase D: 3-Node Operator Tooling (Aider)

**Goal:** Make 3-node setup operator-friendly.

Tasks:
- Write `scripts/android-3node-hub.sh` — starts RTSPanda in viewer mode on Android with mediamtx RTSP re-stream exposed on LAN.
- Write `scripts/pi-detection-relay.sh` — starts RTSPanda in Pi-mode on intermediary Pi with camera URLs pointing to Android mediamtx re-streams.
- Document camera URL format for re-streams: `rtsp://<android-ip>:8554/<camera-name>`.
- Add env var `ANDROID_HUB_IP` to the Pi relay script for easy config.

**Acceptance criteria for Phase D:**
- Operator can follow `docs/android-no-docker.md` 3-node section and reach a working state with two browser sessions: one viewing from Android UI, one viewing from Pi UI.
- Pi detection events visible in Pi dashboard.

### Phase E: Documentation and README (Claude / Cursor)

- `docs/android-no-docker.md` operator guide (see deliverables).
- `docs/cluster-mode.md` intermediary Pi section.
- `docs/raspberry-pi.md` alignment.
- `README.md` setup matrix row.

---

## Acceptance Criteria (Full Initiative)

- [ ] `scripts/android-up.sh` deploys RTSPanda in Pi-mode on Termux without Docker.
- [ ] Thermal monitor reads band from `/sys/class/thermal` or CPU-load proxy.
- [ ] Band transitions logged with correct severity; Discord alert fires on Hot entry.
- [ ] Detection suspended at Hot band; streaming continues unaffected.
- [ ] Decision gate for 3-node documented and referenced in setup guide.
- [ ] `docs/android-no-docker.md` passes operator walkthrough review.
- [ ] `README.md` setup matrix updated with Android row.
- [ ] 3-node scripts functional (`android-3node-hub.sh` + `pi-detection-relay.sh`).
- [ ] No YOLO inference paths added for Android (DEC-018 compliance).

---

## Open Questions for PM

1. **Camera config sync (3-node):** Should we scope a lightweight config-export/import CLI flag (`--export-cameras`, `--import-cameras`) in this initiative, or defer to a future Config Sync feature?
2. **Thermal monitor disabled mode:** When thermal data is unreadable, should the system emit a startup warning or silently proceed? (Current proposal: log warning, proceed.)
3. **`THERMAL_AUTO_RESUME` default:** Default `false` (safest, requires operator action) or `true` (more autonomous)? Current proposal: `false`.
4. **Re-stream port selection:** Should mediamtx RTSP re-stream port be 8554 (mediamtx default) or a different default to reduce conflict risk?
5. **Termux service integration:** Is `termux-services` startup integration in scope for Phase A, or is a `tmux`/`screen` wrapper acceptable?
