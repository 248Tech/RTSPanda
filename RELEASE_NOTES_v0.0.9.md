# RTSPanda v0.0.9 Release Notes

## Headline
RTSPanda v0.0.9 formalizes deployment modes as a runtime contract and adds a dedicated Snapshot Intelligence Engine for Raspberry Pi deployments.

## Highlights
- Added first-class runtime modes: `pi`, `standard`, and `viewer`.
- Added Pi Snapshot AI pipeline with Claude/OpenAI providers and structured detection event output.
- Added strict Pi deployment guardrails: local YOLO AI-worker paths are explicitly blocked on ARM.
- Rewrote architecture and README docs around mode capabilities and constraints.

## What Changed
### Runtime and Backend
- Added `internal/mode` for mode detection, capability gating, and startup banners.
- Updated startup flow in `backend/cmd/rtspanda/main.go`:
  - YOLO detection manager starts only in mode-capable environments.
  - Pi and Viewer still expose detection endpoints through degraded manager handles.
  - Snapshot AI manager starts in Pi-capable mode when enabled.
- Added `CaptureFrameToPath` export in `internal/detections/capture.go` for snapshot pipeline reuse.

### Pi AI and Deployment
- Added `internal/snapshotai/manager.go` and `internal/snapshotai/providers.go` for interval frame analysis through external vision APIs.
- Updated `docker-compose.yml`:
  - Explicit `RTSPANDA_MODE` defaults.
  - Pi profile Snapshot AI environment variables.
- Updated `scripts/pi-up.sh`:
  - Removed unsupported `full` Pi path.
  - Added ARM checks to block standalone AI-worker mode on Pi.
  - Improved user-facing deployment and AI option guidance.

### Documentation
- Reworked `README.md` to explain three deployment modes and expected behavior.
- Expanded AI planning docs (`AI/ARCHITECTURE.md`, `AI/DECISIONS.md`, `AI/PROJECT_CONTEXT.md`) to align implementation with mode decisions.

## Validation
- `cd backend && go test ./...`

## Upgrade Notes
- Existing `docker compose up --build -d` server installs remain supported and map to Standard mode.
- Pi installs should use `PI_DEPLOYMENT_MODE=pi` with optional Snapshot AI env vars.
- Remote AI-worker deployments remain supported via separate server-class hosts.
