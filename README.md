# RTSPanda

RTSPanda is a self-hosted RTSP camera platform for fast browser viewing, recording workflows, and modular AI detection. It runs as a single Go binary with an embedded React frontend, a mediamtx stream relay subprocess, and an optional Python AI worker — all wired together in a single docker-compose file.

**Project tags:** `rtsp` `video-surveillance` `computer-vision` `golang` `react` `fastapi` `onnxruntime` `raspberry-pi` `docker-compose` `homelab`

---

## Deployment Modes

RTSPanda has three explicitly separated deployment modes. Each mode has a clear hardware target, a defined capability set, and hard constraints on what it does and does not support.

| Mode | Hardware Target | AI Type | Real-Time Detection | Primary Use Case |
|------|----------------|---------|---------------------|-----------------|
| **Pi Mode** | Raspberry Pi, phones, ARM | Snapshot AI (Claude / OpenAI) | ✗ No | Edge viewer + interval alerts |
| **Standard** | Dedicated server (x86, GPU) | YOLO (ONNX / GPU) | ✔ Yes | Full system, all features |
| **Viewer** | Windows / Linux desktop | None | ✗ No | Monitoring without AI |

**Raspberry Pi does NOT support real-time AI detection.**
Pi is supported as a viewer node and as a snapshot-based alert node via external vision APIs (Claude, ChatGPT). There is no YOLO inference path on Pi — not experimental, not degraded, not optional. It does not work and is not a goal.

---

## Mode 1 — Pi Mode

**Purpose:** Ultra-lightweight deployment for Raspberry Pi, phones (Termux), and other low-power ARM devices.

**Capabilities:**
- RTSP stream viewing via HLS
- Web UI access
- Stream relay via mediamtx
- Optional snapshot AI alerts (Claude or OpenAI vision)
- Optional interval screenshot capture

**AI on Pi — How It Works:**

Since Pi cannot run YOLO inference, RTSPanda uses the **Snapshot Intelligence Engine**: frames are captured at a configurable interval, sent to an external vision AI API (Claude or OpenAI), and the structured response is turned into detection events and Discord alerts.

Example alert:
```
@Frosty: Amazon driver detected in driveway (confidence: medium)
```

Configuration:
```bash
SNAPSHOT_AI_ENABLED=true
SNAPSHOT_AI_PROVIDER=claude          # or: openai
SNAPSHOT_AI_API_KEY=sk-ant-...
SNAPSHOT_AI_INTERVAL_SECONDS=15
SNAPSHOT_AI_PROMPT="Detect delivery drivers, people, or vehicles near a house."
SNAPSHOT_AI_THRESHOLD=medium         # low | medium | high
```

**Constraints:**
- No real-time detection (interval-based only)
- Latency depends on external API round-trip (typically 1–5 seconds)
- Not suitable for continuous object tracking
- Requires an API key for Claude or OpenAI

**Setup:**

Requirements: Raspberry Pi OS 64-bit recommended, Docker Engine, Docker Compose plugin.

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
chmod +x ./scripts/pi-*.sh
./scripts/pi-preflight.sh
./scripts/pi-up.sh
```

With snapshot AI:
```bash
export SNAPSHOT_AI_ENABLED=true
export SNAPSHOT_AI_PROVIDER=claude
export SNAPSHOT_AI_API_KEY=sk-ant-...
./scripts/pi-up.sh
```

Verify:
```bash
curl -s http://127.0.0.1:8080/api/v1/health
```

Open: `http://localhost:8080`

---

## Mode 2 — Standard Mode

**Purpose:** Full RTSPanda deployment with real-time YOLO detection. Requires a dedicated server — GPU strongly recommended for multi-camera deployments.

**Capabilities:**
- Live RTSP ingest + HLS playback
- Real-time YOLOv8 detection (ONNX Runtime, GPU-accelerated)
- Multi-camera support with per-camera detection controls
- Low-latency Discord alerts with snapshot/clip attachments
- Object confidence filtering, label filtering, ignore zones
- Continuous recording + external storage sync (rclone)
- OpenAI caption integration for Discord alerts

**Setup:**

Requirements: Docker Engine, Docker Compose plugin. GPU optional but recommended for 4+ cameras.

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
docker compose up --build -d
```

Verify:
```bash
curl -s http://127.0.0.1:8080/api/v1/health
curl -s http://127.0.0.1:8080/api/v1/health/ready
curl -s http://127.0.0.1:8080/api/v1/detections/health
```

Open: `http://localhost:8080`

---

## Mode 3 — Viewer Mode

**Purpose:** Simple desktop deployment for monitoring with no AI dependency and no Docker required.

**Capabilities:**
- Local RTSP viewing
- Web UI dashboard
- Optional continuous recording
- No AI worker, no detection pipeline

**Setup:**

Set `RTSPANDA_MODE=viewer` and run the binary directly:

```bash
RTSPANDA_MODE=viewer DATA_DIR=./data ./rtspanda
```

Or with Docker (no AI worker):
```bash
RTSPANDA_MODE=viewer docker compose up --build -d rtspanda
```

---

## Distributed Topology — Pi + Remote AI Worker

Pi can forward detection jobs to a YOLO worker running on a separate server-class machine. This is optional and distinct from snapshot AI.

**Step A: start the AI worker on the server**

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker build ai-worker-standalone
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker up -d --no-build ai-worker-standalone
```

Verify:
```bash
curl -s http://127.0.0.1:8090/health
```

**Step B: point the Pi at the AI worker**

```bash
export AI_WORKER_URL="http://192.168.1.50:8090"
./scripts/pi-up.sh
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Standard Mode (server-class hardware)                          │
│                                                                 │
│  RTSP Cameras → mediamtx → Go backend → Browser (hls.js)       │
│                                ↓                                │
│                       Detection scheduler                        │
│                                ↓                                │
│                    FastAPI AI worker (YOLO)                      │
│                                ↓                                │
│                       Discord alerts                            │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  Pi Mode (Raspberry Pi / ARM)                                   │
│                                                                 │
│  RTSP Cameras → mediamtx → Go backend → Browser (hls.js)       │
│                                ↓                                │
│                    Snapshot Intelligence Engine                  │
│                     (interval frame capture)                    │
│                                ↓                                │
│              Claude / OpenAI Vision API (external)              │
│                                ↓                                │
│                       Discord alerts                            │
│                                                                 │
│  ✗ No YOLO worker    ✗ No real-time detection                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Mode Flag

Set `RTSPANDA_MODE` to control which subsystems start:

| Value | Effect |
|-------|--------|
| `standard` | YOLO detection worker enabled. Default on x86. |
| `pi` | YOLO disabled. Snapshot AI available. Default on ARM. |
| `viewer` | No AI of any kind. Viewer + recordings only. |

On ARM hardware, Pi mode is the automatic default. Forcing `RTSPANDA_MODE=standard` on a Pi will emit a warning and run in degraded state — YOLO inference on Pi is not viable.

---

## Model Setup (Standard Mode Only)

The AI worker is ONNX-only and never converts models at runtime.

**Remote download (default):**
```bash
export MODEL_SOURCE=remote
export YOLO_MODEL_NAME=yolo11n
export YOLO_MODEL_RELEASE=v8.3.0
```

**Local prebuilt model:**
```bash
# Place the ONNX file at either location before building:
./model.onnx
./ai_worker/model/model.onnx

export MODEL_SOURCE=local
docker compose up --build -d
```

---

## Configuration Reference

### Core
| Variable | Default | Description |
|----------|---------|-------------|
| `RTSPANDA_MODE` | auto | `pi`, `standard`, or `viewer` |
| `DATA_DIR` | `./data` | SQLite + snapshots + recordings |
| `PORT` | `8080` | HTTP listen port |

### AI Detection (Standard Mode)
| Variable | Default | Description |
|----------|---------|-------------|
| `AI_MODE` | `local` | `local` or `remote` |
| `AI_WORKER_URL` | — | Remote AI worker URL |
| `DETECTOR_URL` | — | Override detector endpoint directly |

### Snapshot AI (Pi Mode)
| Variable | Default | Description |
|----------|---------|-------------|
| `SNAPSHOT_AI_ENABLED` | `false` | Enable snapshot AI engine |
| `SNAPSHOT_AI_PROVIDER` | `claude` | `claude` or `openai` |
| `SNAPSHOT_AI_API_KEY` | — | API key for the selected provider |
| `SNAPSHOT_AI_INTERVAL_SECONDS` | `30` | Capture interval per camera |
| `SNAPSHOT_AI_PROMPT` | See defaults | Prompt sent with each frame |
| `SNAPSHOT_AI_THRESHOLD` | `medium` | Minimum confidence to alert: `low`, `medium`, `high` |

### Auth
| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_ENABLED` | `false` | Token-based auth |
| `AUTH_TOKEN` | — | Required when auth is enabled |

---

## Documentation
- [Raspberry Pi setup guide](./docs/raspberry-pi.md)
- [Cluster mode — Pi + remote AI worker](./docs/cluster-mode.md)
- [Streaming tuning](./docs/streaming-tuning.md)
- [Testing strategy](./docs/testing-strategy.md)

---

## Development

```bash
cd backend && go test ./internal/...
cd frontend && npm run test -- --config vitest.config.ts
cd ai_worker && python -m pytest -q
```

---

## Engineering Highlights

- **Backend:** Go 1.26 — camera orchestration, detection scheduling, stream management, SQLite state
- **Frontend:** React + TypeScript + Vite SPA embedded in the Go binary
- **AI worker:** FastAPI + ONNX Runtime (Standard mode). No PyTorch at runtime.
- **Snapshot AI:** Claude / OpenAI vision API clients in Go (Pi mode)
- **Streaming:** mediamtx subprocess for RTSP ingest and HLS output
- **Deployment:** Single-machine or distributed docker-compose

---

## Who It's For

- Homelab operators who want local camera control with a credible Pi story
- Developers building on Go + React + Python + ONNX systems
- Small teams who want modular edge-video without cloud vendor lock-in

---

## License

MIT (`LICENSE`).
