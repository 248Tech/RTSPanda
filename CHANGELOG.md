# Changelog

## v0.0.8 - 2026-03-19

### Added
- Pi-first deployment architecture with a lightweight `rtspanda-pi` service and standalone `ai-worker-standalone` profile.
- Remote AI mode via `AI_MODE` and `AI_WORKER_URL`, allowing Raspberry Pi ingest nodes to forward frames to a second machine.
- New docs for [raspberry-pi.md](./docs/raspberry-pi.md) and [cluster-mode.md](./docs/cluster-mode.md).
- Backend tests for AI mode resolution and remote/local detector targeting.
- AI worker tests covering `MODEL_SOURCE`, missing local model behavior, and health reporting.

### Changed
- `ai_worker/Dockerfile` is now deterministic and ONNX-only: no `ultralytics` install, no export fallback, no runtime model conversion.
- AI worker runtime now prefers local models, reports `model_source`, and fails or degrades explicitly when a model is missing.
- `docker-compose.yml` now supports additive `full`, `pi`, and `ai-worker` profiles while keeping `docker compose up --build -d` unchanged.
- Pi helper scripts were refactored for three clear modes: lightweight Pi, full local stack, and standalone AI worker.
- README was rewritten with employer-friendly architecture framing and full setup guides for Standard, Pi Standalone, and Pi + AI deployments.
- Frontend lockfile was refreshed so the current frontend test toolchain installs cleanly.
- Backend frontend-embed behavior now compiles from a clean checkout without requiring prebuilt web assets.

## v0.0.7 - 2026-03-19

### Added
- Token-based authentication flow across backend and frontend (login/session + protected API routes).
- Initial automated test baseline for backend, frontend, and AI worker.
- Raspberry Pi deployment helpers (scripts/pi-preflight.sh, scripts/pi-up.sh, scripts/pi-down.sh).
- New operational docs for streaming tuning, Pi deployment, and AI compatibility.

### Changed
- Multi-arch container build flow to remove hard amd64 assumptions.
- mediamtx runtime defaults aligned to reduce configuration drift and improve latency/stability balance.
- AI worker runtime controls expanded for Pi-safe CPU-only inference and fallback behavior.
- README regenerated with consolidated setup, usage, and architecture guidance.

### Notes
- This release contains coordinated changes across auth, streaming, AI, tests, and deployment. Run a full post-release smoke test on target hardware.
