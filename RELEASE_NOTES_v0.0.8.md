# RTSPanda v0.0.8 Release Notes

## Headline
RTSPanda v0.0.8 turns the project into a more serious edge-video platform: the default single-machine workflow stays intact, while Raspberry Pi deployments gain a cleaner first-run path, deterministic ONNX-only AI builds, and an easy upgrade path to distributed inference.

## Highlights
- Added a Pi-first lightweight deployment mode for `rtspanda` without local AI-worker overhead.
- Added cluster mode so a Pi can ingest streams locally and send frames to a remote AI worker on a second machine.
- Removed AI-worker export fallback from Docker builds. The worker now uses prebuilt ONNX models only.
- Added additive Compose profiles for `full`, `pi`, and `ai-worker` without breaking the existing `docker compose up --build -d` flow.
- Rewrote setup and deployment docs to clearly support Standard, Pi Standalone, and Pi + AI topologies.

## What Changed
### Platform and Deployment
- `docker-compose.yml` now supports:
  - standard full-stack deployment
  - lightweight Pi deployment
  - standalone remote AI-worker deployment
- `scripts/pi-up.sh` now supports:
  - `PI_DEPLOYMENT_MODE=pi`
  - `PI_DEPLOYMENT_MODE=full`
  - `PI_DEPLOYMENT_MODE=ai-worker`
- `scripts/pi-preflight.sh` now checks deployment mode and model-source expectations more accurately for Docker-first Pi workflows.

### AI Runtime
- `ai_worker/Dockerfile` now resolves models deterministically:
  - local prebuilt ONNX file first
  - explicit `YOLO_MODEL_URL` second
  - named Ultralytics ONNX asset fallback last
- No PyTorch install path
- No `YOLO(...).export(...)`
- No runtime model conversion on ARM

### Backend Detection Routing
- Added additive AI-target resolution using:
  - `AI_MODE=local|remote`
  - `AI_WORKER_URL=http://<host>:8090`
  - `DETECTOR_URL` as the highest-precedence override
- Detection health now reports AI mode and remote worker context.

### Release Quality Improvements
- Refreshed the frontend lockfile so current tests/tooling install cleanly.
- Fixed clean-checkout frontend embed compilation behavior in the backend.

## Setup Paths
### Standard

```bash
docker compose up --build -d
```

### Pi Standalone

```bash
./scripts/pi-up.sh
```

### Pi + AI

AI host:

```bash
docker compose --profile ai-worker up --build -d ai-worker-standalone
```

Pi host:

```bash
export AI_WORKER_URL="http://192.168.1.50:8090"
./scripts/pi-up.sh
```

## Validation Checklist
- `docker compose config -q`
- `cd backend && go test ./internal/...`
- `cd frontend && npm run test -- --config vitest.config.ts`
- `cd ai_worker && python -m pytest -q`

## Upgrade Notes
- Existing standard users can continue using `docker compose up --build -d`.
- Pi users who want the old all-in-one behavior can use `PI_DEPLOYMENT_MODE=full ./scripts/pi-up.sh`.
- For custom ONNX assets, place `model.onnx` at repo root or `ai_worker/model/model.onnx`, then set `MODEL_SOURCE=local`.
