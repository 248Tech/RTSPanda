# RTSPanda AI Context

## Architecture Overview
RTSPanda is a multi-service video monitoring system with four primary runtime layers:

1. Backend API (`backend/`, Go)
- Exposes REST API endpoints for health, cameras, streams, settings, detections, alerts, recordings, logs, and metrics.
- Manages SQLite-backed app state, stream lifecycle, and detection orchestration.
- Spawns/coordinates `mediamtx` for RTSP/HLS handling.

2. Frontend UI (`frontend/`, React + TypeScript + Vite)
- Single-page application for camera management, stream viewing, settings, and operational controls.
- Uses API modules under `frontend/src/api/*` and route/page modules under `frontend/src/pages/*`.

3. AI Worker (`ai_worker/`, FastAPI + ONNX Runtime)
- Accepts frames, runs object detection, and returns normalized detection results.
- Includes model/runtime configuration via environment variables and health endpointing.

4. Streaming Layer (`mediamtx/` + backend stream manager)
- `mediamtx` serves stream paths and HLS output.
- Backend stream manager tracks camera path status and health, and applies runtime stream policies.

## Key Components
- API router and middleware: `backend/internal/api/router.go`
- Stream orchestration: `backend/internal/streams/manager.go`, `backend/internal/streams/mediamtx.go`
- Camera domain: `backend/internal/api/cameras.go`, `backend/internal/cameras/*`
- Metrics/system endpoints: `backend/internal/api/metrics.go`, `backend/internal/api/sysinfo.go`
- AI inference service: `ai_worker/app/main.py`
- Container/runtime orchestration: `Dockerfile`, `ai_worker/Dockerfile`, `docker-compose.yml`
- Frontend application shell: `frontend/src/main.tsx`, `frontend/src/App.tsx`

## Recent Changes (Diff-Based: `git diff HEAD~1`)
The current diff includes substantial platform and runtime expansion:

### Security and Access
- Added token-based authentication subsystem in backend (`backend/internal/auth/*`).
- Added frontend auth context/login flow (`frontend/src/auth/*`, app boot gating in `App.tsx`/`main.tsx`).
- Router now supports auth bootstrap/session endpoints and protected API behavior.

### Streaming and Runtime Stability
- Resolved mediamtx config drift by aligning runtime/template/compose defaults.
- Introduced tunable stream defaults (on-demand, HLS segment/part controls) and tuning documentation.
- Added stream/cache/health improvements in backend stream modules.

### Raspberry Pi and Multi-Arch Deployment
- Root `Dockerfile` updated for target-arch-aware builds and mediamtx binary selection.
- `ai_worker/Dockerfile` improved with model download/fallback export flow.
- `docker-compose.yml` adjusted for Pi-oriented defaults, resource limits, and health checks.
- Added helper scripts: `scripts/pi-preflight.sh`, `scripts/pi-up.sh`, `scripts/pi-down.sh`.

### AI Worker Compatibility and Guardrails
- `ai_worker/app/main.py` gained Pi-aware modes, fallback behavior, request pacing, upload limits, model-size checks, and enhanced health reporting.
- `ai_worker/requirements.txt` updated for architecture-sensitive compatibility behavior.

### Test Foundation
- Added backend first tests (`backend/internal/logs/buffer_test.go`, auth tests).
- Added frontend vitest configuration and API tests.
- Added AI worker pytest config and helper tests.
- Added strategy documentation for staged testing rollout.

### Documentation and Operational Guidance
- New operational docs for Pi deployment, streaming tuning, and AI Pi compatibility.
- README updated to cover auth/testing/streaming/Pi setup changes.

## Current Repository State Notes
- Working tree is dirty with coordinated multi-area updates (security, tests, deployment, AI tuning, docs).
- Several updates touch shared integration points (`README.md`, `docker-compose.yml`, `backend/internal/api/router.go`) and should be validated together before release publication.
