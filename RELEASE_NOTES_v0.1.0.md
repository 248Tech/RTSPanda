# RTSPanda v0.1.0 Release Notes

## Headline
RTSPanda v0.1.0 hardens stream orchestration and readiness behavior for production camera fleets, and ships a full operator walkthrough that documents every supported setup method.

## Highlights
- mediamtx orchestration now defaults to proactive stream startup (`sourceOnDemand=false`).
- Stream API now returns `initializing` until HLS is truly reachable.
- Keepalive logic now uses grace and backoff windows instead of tight path repair loops.
- README is now a production-grade setup and operations guide covering all deployment methods.

## What Changed

### Streaming Pipeline Stability
- Enforced proactive path startup and retained TCP RTSP transport defaults.
- Added idempotent path sync:
  - no path recreation if current source already matches
  - update only when path is missing or source changed
- Reduced reload churn:
  - keepalive repair waits for unhealthy grace period
  - repeat repairs respect backoff window
  - full reload remains reserved for repeated mediamtx API failures

### Stream Readiness Gate
- `/api/v1/cameras/:id/stream` now withholds `hls_url` until stream status is `online`.
- New status value `initializing` indicates stream path exists but playlist is not yet playable.
- Dashboard and camera views updated to handle `initializing` cleanly without premature playback attempts.

### Documentation
- Rewrote README into a full production walkthrough:
  - Standard full stack
  - Pi mode
  - Pi + remote worker
  - Standalone AI worker host
  - Viewer mode (binary and Docker)
  - Source development setup
  - hardening, validation, troubleshooting, and upgrade guidance

## Validation
- `cd backend && go test ./...`
- `cd frontend && npm run build`

## Upgrade Notes
- Existing installs can pull and redeploy normally.
- No breaking route changes; stream endpoint now intentionally returns empty `hls_url` until playable.
