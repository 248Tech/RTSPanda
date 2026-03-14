# RTSPanda — Project Context

Last updated: 2026-03-14

## What This Project Is

RTSPanda is a self-hosted RTSP camera platform focused on fast browser viewing plus practical detection/alert workflows.

Current product scope includes:

- Live camera viewing via browser (HLS)
- Optional continuous recording to disk
- YOLOv8 frame-sampled detection pipeline
- Detection overlays and event history
- Discord webhook alerting with snapshot/clip media

Target users: homelab operators, self-hosters, developers, and small business operators.

---

## Core Goals

- Keep deployment simple (`docker compose up --build -d` works end-to-end)
- Keep runtime local-first (no required cloud account/services)
- Provide reliable YOLO detection + alerting without blocking live playback
- Keep camera config and behavior per-camera, not global-only
- Preserve low operational overhead for single-node installs

---

## High-Level Architecture

```
RTSP Cameras
    ↓
mediamtx (RTSP relay/transcode to HLS paths)
    ↓
Go backend (API, camera config, detection scheduler, notifications)
    ↓                     ↘
Browser UI                FastAPI ai-worker (YOLOv8 /detect, /health)
```

Key behavior:

- Browser does not connect directly to camera RTSP endpoints.
- Detection is async and queue-based (workers in backend call AI worker).
- Detection events persist to SQLite with snapshot paths and frame dimensions.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.26 |
| Frontend | React + Vite + TypeScript |
| Database | SQLite (`modernc.org/sqlite`) |
| Streaming | mediamtx |
| AI Worker | Python FastAPI + Ultralytics YOLOv8 |
| Media tooling | FFmpeg |
| Deployment | Docker + docker-compose |

---

## Current Status (v0.0.3)

Shipped and working:

- Camera CRUD + stream status + recordings
- Detection sampler + async worker queue
- YOLO API integration with test detection endpoint
- Live overlay + detection history UI
- Discord detection alerts + interval screenshot alerts
- Manual Discord screenshot/record actions
- Clip format fallback (`webm`, `webp`, `gif`)
- Legacy alert-rule APIs preserved for compatibility

Not done yet:

- Retention cleanup for snapshots/events
- Detection history pagination/filtering
- Discord retry/backoff and failure queue
- Auth layer

---

## Operating Constraints

- Backend remains a single deployable binary image layer in containerized use.
- SQLite remains the default and only bundled DB.
- No auth in current release; deployment assumes trusted LAN/VPN/proxy.
- Live view must not block on detector/notification failures.
