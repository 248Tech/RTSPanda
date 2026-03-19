# Changelog

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
