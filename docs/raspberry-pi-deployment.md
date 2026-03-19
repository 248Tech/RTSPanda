# RTSPanda Raspberry Pi Deployment (Docker Compose)

This guide is the Pi-native Docker deployment path for RTSPanda.

Architecture on Pi:

- `rtspanda` container: Go backend + embedded frontend + mediamtx binary
- `ai-worker` container: FastAPI + ONNX Runtime detector

## Prerequisites

- Raspberry Pi OS Bookworm 64-bit (`aarch64`)
- Raspberry Pi 4/5 recommended (4 GB RAM minimum)
- Docker Engine + Docker Compose plugin installed and running

## 1) Clone and prepare

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
chmod +x ./scripts/pi-*.sh
```

## 2) One-command run (recommended)

```bash
./scripts/pi-up.sh
```

What this does:

- runs `scripts/pi-preflight.sh`
- validates compose config
- builds and starts `rtspanda` + `ai-worker`
- runs health checks (`/health`, `/health/ready`, `/detections/health`)

## 3) Operational checks

Container state:

```bash
docker compose ps
```

Backend health:

```bash
curl -s http://127.0.0.1:8080/api/v1/health
curl -s http://127.0.0.1:8080/api/v1/health/ready
```

Detector health:

```bash
curl -s http://127.0.0.1:8080/api/v1/detections/health
```

Expected behavior:

- `/api/v1/health` returns `{"status":"ok"}`
- `/api/v1/health/ready` becomes healthy after DB + stream manager init
- `/api/v1/detections/health` may be degraded briefly during first model load

## 4) Logs and lifecycle

Follow logs:

```bash
docker compose logs -f rtspanda ai-worker
```

Restart:

```bash
docker compose restart
```

Stop:

```bash
./scripts/pi-down.sh
```

## 5) Security (recommended before remote/network exposure)

By default, Compose sets `AUTH_ENABLED=false` for first-run convenience.
To enforce authentication:

```bash
export AUTH_ENABLED=true
export AUTH_TOKEN='replace-with-a-long-random-token'
./scripts/pi-up.sh
```

## 6) Model override (optional)

Default model is `yolov8n` ONNX. Override at build time:

```bash
export YOLO_MODEL_NAME=yolov8s
./scripts/pi-up.sh
```

If your network blocks GitHub assets, provide a direct ONNX URL:

```bash
export YOLO_MODEL_URL='https://your-mirror/yolov8n.onnx'
./scripts/pi-up.sh
```

## 7) Upgrade flow

```bash
git pull
./scripts/pi-up.sh
```

This rebuilds images with the latest code and restarts the stack.
