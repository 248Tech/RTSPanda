# RTSPanda — Architecture

## System Overview

RTSPanda is a modular monolith. One binary, one config, one compose file — with three explicitly separated deployment modes.

---

## Deployment Modes

### Mode: Standard (server-class hardware)

```
┌──────────────────────────────────────────────────────────────────┐
│  Docker Container                                                │
│                                                                  │
│  ┌──────────────────┐    ┌──────────────────────┐               │
│  │   Go Backend     │◄───│      mediamtx         │               │
│  │  - REST API      │    │  - RTSP ingest        │               │
│  │  - Static files  │    │  - HLS output         │               │
│  │  - SQLite        │    └──────────────────────┘               │
│  │  - Detection mgr │                                            │
│  └────────┬─────────┘                                            │
│           │                                                      │
│  ┌────────▼──────────┐    ┌─────────────────────┐               │
│  │  React Frontend   │    │  FastAPI AI Worker   │               │
│  │  (embedded)       │    │  - YOLOv8 ONNX       │               │
│  └───────────────────┘    │  - /detect endpoint  │               │
│                           └──────────────────────┘               │
└──────────────────────────────────────────────────────────────────┘
           ▲
           │ HTTP (port 8080)
      Browser (hls.js)
```

**Capable hardware required.** GPU strongly recommended for 4+ cameras.

---

### Mode: Pi (Raspberry Pi / ARM)

```
┌──────────────────────────────────────────────────────────────────┐
│  Docker Container (arm64 / armv7)                                │
│                                                                  │
│  ┌──────────────────┐    ┌──────────────────────┐               │
│  │   Go Backend     │◄───│      mediamtx         │               │
│  │  - REST API      │    │  - RTSP ingest        │               │
│  │  - Static files  │    │  - HLS output         │               │
│  │  - SQLite        │    └──────────────────────┘               │
│  │  - Snapshot AI   │                                            │
│  └────────┬─────────┘                                            │
│           │                                                      │
│  ┌────────▼──────────┐    ┌──────────────────────────────────┐   │
│  │  React Frontend   │    │  Snapshot Intelligence Engine    │   │
│  │  (embedded)       │    │  - FFmpeg frame capture          │   │
│  └───────────────────┘    │  - Claude / OpenAI vision API   │   │
│                           │  - Structured event output       │   │
│                           └──────────────────────────────────┘   │
│                                                                  │
│  ✗ No YOLO AI worker  ✗ No real-time detection                  │
└──────────────────────────────────────────────────────────────────┘
           ▲
           │ HTTP (port 8080)
      Browser (hls.js)
```

**Raspberry Pi does NOT support real-time YOLO inference.** The Snapshot
Intelligence Engine is the Pi AI replacement. It captures frames at a configured
interval and sends them to an external vision API for interpretation.

---

### Mode: Viewer (desktop, no AI)

Identical to Standard but with the AI worker disabled and RTSPANDA_MODE=viewer.
Suitable for Windows/Linux desktops that want camera management without detection.

---

## Backend — Module Breakdown

### `cmd/rtspanda/main.go`

- Detect deployment mode (`RTSPANDA_MODE` or auto from `runtime.GOARCH`)
- Parse config (env vars)
- Start SQLite, run migrations
- Start mediamtx subprocess
- Start YOLO detection manager (Standard mode only)
- Start Snapshot Intelligence Engine (Pi mode only)
- Start HTTP server
- Handle graceful shutdown

### `internal/mode/`

- `mode.go` — Mode type, `Detect()`, `AIInferenceAllowed()`, `SnapshotAIAllowed()`, `LogBanner()`

### `internal/db/`

- `db.go` — Open SQLite connection, run migrations
- `migrations/` — Sequential SQL migration files

### `internal/cameras/`

- `model.go` — Camera struct
- `service.go` — Business logic for camera CRUD
- `repository.go` — DB queries, returns domain types

### `internal/streams/`

- `manager.go` — Lifecycle for active streams
- `mediamtx.go` — Manages mediamtx process + config generation
- `health.go` — Polls mediamtx for stream status (3-second cache)

### `internal/detections/`

- `manager.go` — Detection sampler + async worker queue (Standard mode)
- `ai_config.go` — AI mode resolution (local / remote)
- `capture.go` — FFmpeg single-frame extraction
- `client.go` — HTTP client to /detect endpoint
- `model.go` — Detection types (Detection, Event, Snapshot, Health)
- `repository.go` — SQLite event persistence

### `internal/snapshotai/`

- `manager.go` — Interval-based snapshot capture → vision API → event emit (Pi mode)
- `providers.go` — OpenAI and Claude vision API clients

### `internal/notifications/`

- `discord.go` — Discord webhook dispatch (detection events, manual snapshots/clips)
- `openai_caption.go` — Optional OpenAI caption provider for Discord messages

### `internal/api/`

- `router.go` — HTTP routes (all modes share the same router)
- `cameras.go` — Camera REST handlers
- `streams.go` — Stream status + HLS URL endpoint
- `static.go` — Serves embedded React frontend

---

## Database Schema

### `cameras` table

```sql
CREATE TABLE cameras (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    rtsp_url    TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
    -- plus: detection controls, Discord settings, recording config
);
```

### `settings` table

```sql
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

### `detection_events` table

Used by both YOLO pipeline (Standard) and Snapshot AI (Pi). Fields are identical.

```sql
CREATE TABLE detection_events (
    id            TEXT PRIMARY KEY,
    camera_id     TEXT NOT NULL,
    object_label  TEXT NOT NULL,
    confidence    REAL NOT NULL,
    bbox_json     TEXT NOT NULL,
    snapshot_path TEXT NOT NULL,
    frame_width   INTEGER,
    frame_height  INTEGER,
    raw_payload   TEXT,
    created_at    DATETIME NOT NULL
);
```

---

## REST API

Base path: `/api/v1`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/cameras` | List all cameras |
| POST | `/api/v1/cameras` | Add camera |
| GET | `/api/v1/cameras/:id` | Get camera |
| PUT | `/api/v1/cameras/:id` | Update camera |
| DELETE | `/api/v1/cameras/:id` | Delete camera |
| GET | `/api/v1/cameras/:id/stream` | Stream URL + status |
| GET | `/api/v1/detections/health` | Detection subsystem health |
| GET | `/api/v1/health` | Liveness |
| GET | `/api/v1/health/ready` | Readiness |
| GET | `/api/v1/system/stats` | Process stats |
| GET | `/metrics` | Prometheus metrics |

---

## Streaming Pipeline

1. User adds camera RTSP URL via API
2. Backend writes mediamtx config for that camera
3. mediamtx ingests RTSP → outputs HLS segments (on-demand only)
4. Frontend fetches HLS URL from `/api/v1/cameras/:id/stream`
5. hls.js player loads the `.m3u8` playlist
6. Browser plays video — no direct camera connection

`sourceOnDemand: true` is locked — streams must not stay open when nobody is watching.

---

## Detection Pipelines

### Standard Mode (YOLO)

```
Camera RTSP URL
    ↓ (FFmpeg single-frame extract, per camera sample interval)
Detection queue (async goroutine workers)
    ↓ (HTTP POST to /detect)
FastAPI AI worker (ONNX Runtime)
    ↓
Detection response (labels, confidence, bboxes)
    ↓
Persist to detection_events
    ↓
Discord alert (if camera has webhook + trigger configured)
```

### Pi Mode (Snapshot AI)

```
Camera RTSP URL
    ↓ (FFmpeg single-frame extract, per SNAPSHOT_AI_INTERVAL_SECONDS)
Base64 encode JPEG
    ↓ (HTTPS POST with image + prompt)
Claude / OpenAI vision API
    ↓
Structured JSON response {detected, label, confidence, summary}
    ↓
Persist to detection_events (same schema as YOLO events)
    ↓
Discord alert (if camera has webhook + trigger configured)
```

Both pipelines emit identical event records and identical Discord alert formats. The
UI cannot and does not need to distinguish between them.

---

## Frontend Architecture

- Custom `usePath` hook routing (no react-router-dom)
- React.lazy code splitting per page (202 kB initial bundle)
- Tailwind dark theme (zinc-950 base, blue-600 accent)
- API client wrappers in `src/api/`

Pages: Dashboard, CameraView, MultiCameraView, Settings, Guides

---

---

## Android No-Docker Architecture

Android (Termux) is a supported RTSPanda deployment target using Pi-mode behavior. No Docker. Go binary, mediamtx, and FFmpeg run as native ARM64 processes.

### Mode: Android 2-Node (Termux + Remote AI Worker)

```
┌──────────────────────────────────────────────┐        ┌────────────────────────────┐
│  Android Device (Termux, no Docker)          │        │  AI Server (x86, Docker)   │
│                                              │  LAN   │                            │
│  rtspanda (Go binary, RTSPANDA_MODE=pi)      │        │  ai-worker (FastAPI ONNX)  │
│  ├── mediamtx subprocess                     │◄──────►│  port 8090                 │
│  ├── SQLite (DATA_DIR)                       │        │                            │
│  ├── FFmpeg (frame capture)                  │        └────────────────────────────┘
│  ├── Snapshot AI optional (Pi mode)          │
│  └── React UI (embedded, port 8080)         │
│                                              │
│  Detection: FFmpeg → HTTP frames → server    │
└──────────────────────────────────────────────┘
         ▲         ▲         ▲
         │ RTSP    │ RTSP    │ RTSP
     Camera 1  Camera 2  Camera 3
```

### Mode: Android 3-Node (Thermally Constrained — Android Hub + Intermediary Pi + AI Server)

Use when: camera count ≥ 4, resolution ≥ 1080p, sample interval < 15 s, or sustained temperature ≥ 55°C at 2-node.

```
┌────────────────────────────────────────┐
│  Android Device (Termux)               │
│  RTSPANDA_MODE=viewer                  │
│  ├── mediamtx subprocess               │
│  │   └── RTSP re-stream LAN port 8554  │
│  ├── SQLite (camera metadata)          │
│  └── React UI (embedded, port 8080)   │
└────────────────────────────────────────┘
         ▲          ▲
         │ RTSP     │ RTSP
     Camera 1   Camera 2-N
         │
         ▼  RTSP re-stream  rtsp://<android>:8554/<name>
┌────────────────────────────────────────────┐
│  Intermediary Raspberry Pi                 │
│  RTSPANDA_MODE=pi                          │
│  ├── mediamtx subprocess                   │
│  ├── SQLite (detection events)             │
│  ├── FFmpeg (frame capture from re-streams)│
│  └── AI_WORKER_URL=http://<server>:8090    │
└────────────────────────────────────────────┘
                    │ HTTP frames
                    ▼
         ┌──────────────────────────┐
         │  AI Server (x86, Docker) │
         │  ai-worker port 8090     │
         └──────────────────────────┘
```

**3-node data flows:**

- Android mediamtx ingests camera RTSP and re-streams on port 8554 (LAN-accessible).
- Pi mediamtx reads Android re-streams as its camera sources.
- Pi FFmpeg extracts frames, dispatches to remote YOLO server.
- Detection events persist in Pi SQLite; Discord alerts fire from Pi.
- Android and Pi have separate databases. Camera configs must be manually synchronized.

### Thermal Policy (Android Pi-mode)

A background goroutine in `backend/internal/thermal/` samples device temperature and enforces band-based policy:

```
Temperature → ThermalMonitor goroutine
                    │
          ┌─────────▼──────────┐
          │  Band Detection     │
          │  + Hysteresis Timer │
          └─────────┬──────────┘
                    │ ThermalBandEvent
         ┌──────────▼──────────┐
         │  Detection Manager  │
         │  (subscribe + act)  │
         └─────────────────────┘
```

| Band | Temp | Detection Action |
|------|------|-----------------|
| Normal | < 45°C | Full operation |
| Warm | 45–54°C | Sample interval floor → 30 s |
| Hot | 55–64°C | Detection suspended |
| Critical | ≥ 65°C | Detection suspended; new stream opens blocked |

Recovery requires sustained cool-down (5 minutes at target threshold) before band downgrade.

### Key Rules for Android

1. Android does NOT run YOLO locally (DEC-018 and DEC-023).
2. Android does NOT run Docker (DEC-021).
3. In 3-node, Android is always the stream hub (viewer mode); Pi is always the detection relay.
4. Thermal monitor reads `/sys/class/thermal/thermal_zone*/temp` or falls back to CPU load proxy.
5. `THERMAL_AUTO_RESUME=false` by default — operator must explicitly re-enable detection after Hot/Critical.

---

## Deployment

### docker-compose.yml profiles

| Profile | Services | Mode |
|---------|----------|------|
| (default) | rtspanda + ai-worker | Standard |
| pi | rtspanda-pi | Pi |
| ai-worker | ai-worker-standalone | Remote AI worker (server) |

### Important rules

1. Browser never touches camera directly
2. mediamtx runs as a subprocess managed by Go — not a separate container
3. Streams only open when actively viewed (`sourceOnDemand`)
4. SQLite stored in mounted volume — survives restarts
5. Frontend embedded in Go binary
6. **Raspberry Pi does NOT support real-time YOLO inference — ever**
7. Go binary < 50 MB, Docker image < 150 MB
