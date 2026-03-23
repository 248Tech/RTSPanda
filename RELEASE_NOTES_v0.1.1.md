# RTSPanda v0.1.1 Release Notes

## Headline
RTSPanda v0.1.1 adds Android no-Docker startup support and introduces an arm64 Pi-mode thermal monitor with banded alerts and API visibility.

## Highlights
- Added `scripts/android-up.sh` for Termux-native startup (no Docker, no root).
- Added thermal monitor package with four thermal bands and hysteresis handling.
- Added `thermal_band` to `/api/v1/system/stats`.
- Added Discord system alert dispatch on Hot thermal band entry.
- Updated README and quickstart with Android and release-aligned setup guidance.

## What Changed

### Android startup path
- New script: `scripts/android-up.sh`
  - sets `RTSPANDA_MODE=pi`
  - sets/creates `DATA_DIR`
  - runs `termux-wake-lock` when available
  - launches `./rtspanda`

### Thermal monitoring
- New package: `backend/internal/thermal/monitor.go`
- Thermal bands:
  - Normal: `<45C`
  - Warm: `45-54C`
  - Hot: `55-64C`
  - Critical: `>=65C`
- Sensor source priority:
  1. `/sys/class/thermal/thermal_zone*/temp`
  2. `/proc/loadavg` proxy fallback
  3. disabled mode with warning
- Hysteresis:
  - Critical -> Hot: 5m
  - Hot -> Warm: 5m
  - Warm -> Normal: 3m

### Runtime integration
- `cmd/rtspanda/main.go` now starts thermal monitor when:
  - `GOARCH=arm64` and mode is `pi`, or
  - `THERMAL_MONITOR_ENABLED=true`
- `THERMAL_AUTO_RESUME` defaults to `false`.
- Thermal transition events are logged with severity:
  - Warm: WARN
  - Hot: ERROR
  - Critical: CRITICAL
- On first Hot-band entry, Discord system alerts are sent to cameras with configured webhook URLs.

### API change
- `GET /api/v1/system/stats` now includes:
  - `thermal_band` (`normal`, `warm`, `hot`, `critical`)

## Validation
- `cd backend && go test ./...`

## Upgrade Notes
- Existing deployments can update in place.
- Android/Termux users can use `scripts/android-up.sh` for consistent startup.
