# RTSPanda Human Context

## Plain-Language Summary
RTSPanda has moved from a basic camera monitoring stack to a more production-ready setup. The biggest changes add authentication, first-pass automated tests, Raspberry Pi deployment support, and better stream/AI behavior under limited hardware resources.

## What Changed Recently
1. Security
- API/UI access now supports token-based authentication with login/session handling.

2. Streaming Reliability
- Streaming defaults were cleaned up to remove conflicting settings.
- HLS and on-demand behavior is now more consistent and easier to tune.

3. Raspberry Pi Readiness
- Docker build and compose flow now target ARM/Pi scenarios more directly.
- New helper scripts provide preflight checks and one-command startup/shutdown.

4. AI Worker Practicality
- AI worker now has fallback/degraded modes and guardrails for low-power devices.
- Additional runtime limits reduce overload risk on Pi hardware.

5. Test Baseline
- Initial backend/frontend/AI tests and configs were added to establish a foundation.

## Impact of Changes
### Positive Impact
- Better default security posture (with configurable auth behavior).
- Easier deployment path on Raspberry Pi.
- More stable streaming defaults and clearer tuning knobs.
- Early automated tests reduce regression risk.

### Operational Impact
- Environment configuration is now more important (auth token + AI/stream tuning variables).
- First deploys should include explicit validation of health endpoints and stream behavior.
- Release preparation now requires checking multiple related areas (README, compose, changelog, versioning) together.

### Remaining Risks
- Some integration points changed simultaneously and need one consolidated smoke test pass before publishing.
- AI performance on smaller Pi devices can still be constrained; fallback modes should be used when needed.
