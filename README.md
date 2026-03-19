# RTSPanda

Self-hosted RTSP camera monitoring with browser playback, recording, AI detection, and stream orchestration.

## Overview
RTSPanda combines:
- A Go backend API for camera, stream, detection, alert, and recording workflows.
- A React frontend for dashboard and operations.
- A Python AI worker (FastAPI + ONNX Runtime) for object detection.
- mediamtx for RTSP ingestion and HLS output.

Primary goals:
- No cloud dependency for core operation.
- Local-first deployment (native or Docker Compose).
- Practical controls for low-power hosts, including Raspberry Pi.

## Installation and Setup
### Prerequisites
- Docker + Docker Compose plugin (recommended path)
- Or native toolchain: Go 1.26+, Node.js 18+, Python 3.12+
- RTSP camera URLs

### Recommended: Docker Compose
```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
chmod +x ./scripts/pi-*.sh
./scripts/pi-up.sh
```

If you are not on Pi and want the plain compose path:
```bash
docker compose up --build -d
```

Stop services:
```bash
./scripts/pi-down.sh
# or
docker compose down
```

### Native Build (without Docker)
Windows:
```powershell
.\build.ps1
.\backend\rtspanda.exe
```

macOS/Linux:
```bash
make build
./backend/rtspanda
```

### Authentication Setup
Auth middleware is enabled by default in backend config.
Set token before startup when auth is enabled:

```bash
export AUTH_ENABLED=true
export AUTH_TOKEN="replace-with-long-random-token"
```

Windows PowerShell:
```powershell
$env:AUTH_ENABLED="true"
$env:AUTH_TOKEN="replace-with-long-random-token"
```

## Usage
### Start and Access
- Open `http://localhost:8080`
- Add cameras in Settings
- View streams in dashboard/multi-view

### Core Operations
- Camera CRUD and stream status via API/UI
- Per-camera recording management
- Detection events and snapshots
- Discord notification/test actions

### Health and Diagnostics
```bash
curl -s http://127.0.0.1:8080/api/v1/health
curl -s http://127.0.0.1:8080/api/v1/health/ready
curl -s http://127.0.0.1:8080/api/v1/detections/health
```

### AI Worker on Raspberry Pi
Suggested baseline environment tuning:
```bash
YOLO_PI_MODE=on
YOLO_ORT_INTRA_THREADS=1
YOLO_ORT_INTER_THREADS=1
YOLO_MIN_REQUEST_INTERVAL_MS=750
YOLO_BUSY_POLICY=drop
YOLO_MAX_SOURCE_SIDE=1280
```

Fallback when model/runtime is too heavy:
```bash
YOLO_MODEL_REQUIRED=false
YOLO_FALLBACK_MODE=empty
YOLO_ENABLE_MODEL=false
```

## Architecture Summary
### Backend (`backend/`)
- Entry: `backend/cmd/rtspanda/main.go`
- API/router: `backend/internal/api/router.go`
- Stream manager: `backend/internal/streams/*`
- Auth module: `backend/internal/auth/*`

### Frontend (`frontend/`)
- Entry: `frontend/src/main.tsx`
- App shell: `frontend/src/App.tsx`
- Auth flow: `frontend/src/auth/*`

### AI Worker (`ai_worker/`)
- Entry: `ai_worker/app/main.py`
- Runtime config via `YOLO_*` environment variables
- Tests under `ai_worker/tests/`

### Streaming (`mediamtx/`)
- Template: `mediamtx/mediamtx.yml.tmpl`
- Runtime tuning documented in `docs/streaming-tuning.md`

## Configuration Highlights
- `AUTH_ENABLED`, `AUTH_TOKEN`, `AUTH_COOKIE_*`
- `MEDIAMTX_SOURCE_ON_DEMAND`, `MEDIAMTX_HLS_*`
- `DETECTOR_URL`, `DETECTION_*`
- `YOLO_*` tuning variables for AI runtime behavior

## Testing
```bash
cd backend && go test ./internal/...
cd frontend && npm run test -- --config vitest.config.ts
cd ai_worker && python -m pytest -q
```

Testing strategy document:
- `docs/testing-strategy.md`

## Related Docs
- `docs/raspberry-pi-first-run.md`
- `docs/raspberry-pi-deployment.md`
- `docs/streaming-tuning.md`
- `docs/ai-pi-compatibility.md`
- `human/USER_GUIDE.md`

## License
MIT (`LICENSE`).
