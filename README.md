# RTSPanda

RTSPanda is a self-hosted, local-first RTSP camera platform built for fast browser viewing, practical recording workflows, and modular AI detection that can run either on one machine or across a lightweight Pi + remote worker topology.

**Project tags:** `rtsp` `video-surveillance` `computer-vision` `golang` `react` `fastapi` `onnxruntime` `raspberry-pi` `docker-compose` `homelab`

## Why RTSPanda
- Fast browser playback through `mediamtx` and HLS, without exposing camera RTSP feeds directly to the browser
- Local-first architecture with no required cloud dependency for core operation
- Modular AI pipeline: run detection locally or forward frames to a remote AI worker
- Raspberry Pi-friendly deployment path with deterministic ONNX-only AI builds
- Production-minded runtime choices: health checks, resource caps, queue-based detection, graceful degradation, and multi-arch Docker support

## Engineering Highlights
- **Backend:** Go 1.26 API, camera orchestration, detection scheduling, stream management, alerts, recordings, and SQLite-backed state
- **Frontend:** React + TypeScript + Vite SPA embedded into the backend for a clean single-app deployment
- **AI worker:** FastAPI + ONNX Runtime inference service with Pi-aware throttling and explicit degraded-mode behavior
- **Streaming layer:** `mediamtx` for RTSP ingest and HLS output
- **Deployment model:** single-machine standard stack, Pi-only streaming node, or Pi + remote AI worker cluster

## Architecture

```text
RTSP Cameras
    ↓
mediamtx
    ↓
Go backend API + embedded frontend
    ↓
Detection scheduler / queue
    ↓
FastAPI AI worker (local or remote)
```

In the default workflow, `rtspanda` and `ai-worker` run together. In cluster mode, the Pi keeps RTSP ingest, playback, and recording local while forwarding sampled frames to a second machine for inference.

## Choose Your Setup

| Mode | Best For | Command |
|------|----------|---------|
| Standard | x86, laptops, desktops, single-node homelab installs | `docker compose up --build -d` |
| Pi Standalone | Raspberry Pi running UI + streaming only | `./scripts/pi-up.sh` |
| Pi + AI | Raspberry Pi for ingest/UI plus remote AI worker | `AI_WORKER_URL=http://<host>:8090 ./scripts/pi-up.sh` |

## Full Setup Guide
### 1. Standard Mode

Best for users who want the existing single-machine experience with the least setup.

Requirements:
- Docker Engine
- Docker Compose plugin

Setup:

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

Open:
- `http://localhost:8080`

Optional explicit full-profile alias:

```bash
docker compose --profile full up --build -d stack-full
```

### 2. Pi Standalone

Best for Raspberry Pi 4/5 users who want reliable first-run streaming, recording, and UI without paying the cost of building or running the AI worker locally.

Requirements:
- Raspberry Pi OS 64-bit recommended
- Docker Engine
- Docker Compose plugin

Setup:

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
chmod +x ./scripts/pi-*.sh
./scripts/pi-preflight.sh
./scripts/pi-up.sh
```

Equivalent manual command:

```bash
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile pi build rtspanda-pi
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile pi up -d --no-build rtspanda-pi
```

Behavior:
- `rtspanda-pi` runs backend + embedded frontend + `mediamtx`
- no local AI worker is started
- detections stay degraded until `AI_WORKER_URL` or `DETECTOR_URL` is configured

Verify:

```bash
curl -s http://127.0.0.1:8080/api/v1/health
curl -s http://127.0.0.1:8080/api/v1/detections/health
```

### 3. Pi + AI

Best for a distributed deployment where the Pi handles cameras and the second machine handles inference.

#### Step A: start the AI worker on the second machine

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker build ai-worker-standalone
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker up -d --no-build ai-worker-standalone
```

Verify the worker:

```bash
curl -s http://127.0.0.1:8090/health
```

#### Step B: point the Pi at the AI worker

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
chmod +x ./scripts/pi-*.sh
export AI_WORKER_URL="http://192.168.1.50:8090"
./scripts/pi-up.sh
```

Equivalent manual command on the Pi:

```bash
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile pi build rtspanda-pi
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile pi up -d --no-build rtspanda-pi
```

Verify from the Pi:

```bash
curl -s http://127.0.0.1:8080/api/v1/detections/health
```

Expected detection-health fields:
- `ai_mode` = `remote`
- `ai_worker_url` = your remote AI worker
- `detector_url` = resolved detector target

## Model Setup
The Docker AI worker is ONNX-only and never exports or converts models at runtime.

### Default remote model download

```bash
export MODEL_SOURCE=remote
export YOLO_MODEL_NAME=yolo11n
export YOLO_MODEL_RELEASE=v8.3.0
```

Optional mirror:

```bash
export YOLO_MODEL_URL="https://your-mirror.example/yolo11n.onnx"
```

### Local prebuilt model

Place a prebuilt ONNX file at either location before building:

```text
./model.onnx
./ai_worker/model/model.onnx
```

Then build with:

```bash
export MODEL_SOURCE=local
docker compose up --build -d
```

Runtime mount target for custom deployments:

```text
/model/model.onnx
```

## Configuration Highlights
- `AI_MODE=local|remote`
- `AI_WORKER_URL=http://<host>:8090`
- `DETECTOR_URL=http://<custom-detector>:8090`
- `MODEL_SOURCE=local|remote`
- `MODEL_PATH=/model/model.onnx`
- `YOLO_MODEL_URL`, `YOLO_MODEL_NAME`, `YOLO_MODEL_RELEASE`
- `MEDIAMTX_*` tuning for stream latency and on-demand behavior
- `AUTH_ENABLED`, `AUTH_TOKEN` for protected deployments

## Documentation
- [Raspberry Pi guide](./docs/raspberry-pi.md)
- [Cluster mode guide](./docs/cluster-mode.md)
- [Raspberry Pi first run](./docs/raspberry-pi-first-run.md)
- [Raspberry Pi deployment](./docs/raspberry-pi-deployment.md)
- [Streaming tuning](./docs/streaming-tuning.md)
- [AI Pi compatibility](./docs/ai-pi-compatibility.md)
- [Testing strategy](./docs/testing-strategy.md)

## Development and Validation

```bash
cd backend && go test ./internal/...
cd frontend && npm run test -- --config vitest.config.ts
cd ai_worker && python -m pytest -q
```

## Who It’s For
- Homelab operators who want local camera control and simple deployment
- Developers interested in Go + React + FastAPI + ONNX Runtime systems
- Small teams or operators who want a modular edge-video stack with a credible Pi story

## License
MIT (`LICENSE`).
