# RTSPanda — Project Context

Last updated: 2026-03-20

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

## Deployment Modes

RTSPanda has three explicitly separated deployment modes:

| Mode | Hardware | AI | Command |
|------|----------|----|---------|
| `pi` | Raspberry Pi / ARM | Snapshot AI (Claude/OpenAI) | `./scripts/pi-up.sh` |
| `standard` | Server (x86/GPU) | YOLO (ONNX) | `docker compose up --build -d` |
| `viewer` | Desktop | None | `RTSPANDA_MODE=viewer` |

**Raspberry Pi does NOT support real-time YOLO inference.** Pi uses the Snapshot
Intelligence Engine: interval frame capture → Claude/OpenAI vision API → events.

Set `RTSPANDA_MODE` to override auto-detection (default: `pi` on ARM, `standard` on x86).

## High-Level Architecture

**Standard mode:**
```
RTSP Cameras → mediamtx → Go backend → Browser UI
                                ↓
                       Detection scheduler (YOLO)
                                ↓
                     FastAPI ai-worker (/detect)
```

**Pi mode:**
```
RTSP Cameras → mediamtx → Go backend → Browser UI
                                ↓
                    Snapshot Intelligence Engine
                    (interval capture + FFmpeg)
                                ↓
                  Claude / OpenAI vision API (HTTPS)
```

Key behavior:

- Browser does not connect directly to camera RTSP endpoints.
- YOLO detection is async and queue-based (Standard mode only).
- Snapshot AI is interval-based with external API round-trip (Pi mode only).
- Detection events use identical schema regardless of source.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.26 |
| Frontend | React + Vite + TypeScript |
| Database | SQLite (`modernc.org/sqlite`) |
| Streaming | mediamtx |
| AI Worker (Standard) | Python FastAPI + ONNX Runtime (YOLOv8) |
| Snapshot AI (Pi) | Claude / OpenAI vision API (Go HTTP clients) |
| Media tooling | FFmpeg |
| Deployment | Docker + docker-compose |

---

## Current Status (v0.1.0)

Shipped and working:

- Camera CRUD + stream status + recordings
- Detection sampler + async worker queue (YOLO — Standard mode)
- Snapshot Intelligence Engine (Pi mode — Claude/OpenAI vision API)
- Three deployment modes: `pi`, `standard`, `viewer` (RTSPANDA_MODE)
- YOLO API integration with test detection endpoint
- Live overlay + detection history UI
- Discord detection alerts + interval screenshot alerts
- Manual Discord screenshot/record actions
- Clip format fallback (`webm`, `webp`, `gif`)
- Legacy alert-rule APIs preserved for compatibility
- Performance + observability (v0.0.6): stream status cache, Prometheus metrics, 76% JS bundle reduction
- Multi-view UI, Operator Dark theme (v0.0.6)
- Deployment modes + snapshot AI rollout (v0.0.9)
- Stream orchestration hardening + readiness gating (v0.1.0)

Not done yet:

- Retention cleanup for snapshots/events
- Detection history pagination/filtering
- Discord retry/backoff and failure queue
- Auth layer
- WebRTC streaming (Phase 2)
- Snapshot AI settings UI (currently env-var only)

---

## Operating Constraints

- Backend remains a single deployable binary image layer in containerized use.
- SQLite remains the default and only bundled DB.
- No auth in current release; deployment assumes trusted LAN/VPN/proxy.
- Live view must not block on detector/notification failures.
