# Streaming Tuning (mediamtx)

This document defines the default RTSPanda streaming profile and tuning knobs for latency, stability, and scaling.

## Audit Summary

Before this update, streaming config had drift across files:

- Runtime template in `backend/internal/streams/mediamtx.go` used:
  - `sourceOnDemand: no`
  - `hlsAlwaysRemux: yes`
  - `hlsSegmentCount: 7`
- Reference template in `mediamtx/mediamtx.yml.tmpl` and internal notes expected:
  - `sourceOnDemand: yes`
  - `hlsAlwaysRemux: no`
  - `hlsSegmentCount: 3`

That mismatch changed behavior in production-like runs: higher steady RTSP fan-out, larger HLS buffers, and conflicting memory assumptions.

## Default Profile (Now Canonical)

Source of truth: `backend/internal/streams/mediamtx.go` (generated at runtime to `DATA_DIR/mediamtx.yml`).

Global defaults:

- `MEDIAMTX_SOURCE_ON_DEMAND=true`
- `MEDIAMTX_SOURCE_ON_DEMAND_CLOSE_AFTER=10s`
- `MEDIAMTX_HLS_ALWAYS_REMUX=false`
- `MEDIAMTX_HLS_SEGMENT_COUNT=3`
- `MEDIAMTX_HLS_SEGMENT_DURATION=2s`
- `MEDIAMTX_HLS_PART_DURATION=200ms`
- Per-path transport defaults remain:
  - `rtspTransport: tcp`
  - `rtspAnyPort: yes`

## Why These Defaults

Latency:

- `hlsSegmentCount=3` + `hlsPartDuration=200ms` keeps playback startup and live edge tighter than a larger segment window.

Stability:

- `rtspTransport=tcp` avoids common UDP RTP breakage in Docker/NAT/camera-vendor networks.
- `hlsSegmentDuration=2s` is conservative enough to reduce segment churn while remaining responsive.

Scaling:

- `sourceOnDemand=true` prevents opening RTSP sessions for cameras nobody is watching.
- `hlsAlwaysRemux=false` avoids continuous remux work for idle paths.

## Tuning Knobs

All values are optional environment variables on the backend process (`rtspanda` service in compose).

| Variable | Default | Allowed / Guidance | Tradeoff |
|---|---:|---|---|
| `MEDIAMTX_SOURCE_ON_DEMAND` | `true` | `true`/`false` | `false` gives fastest instant viewing, but scales poorly with many cameras. |
| `MEDIAMTX_SOURCE_ON_DEMAND_CLOSE_AFTER` | `10s` | `1s` to `5m` | Higher reduces reconnect churn; lower releases camera sessions faster. |
| `MEDIAMTX_HLS_ALWAYS_REMUX` | `false` | `true`/`false` | `true` reduces first-view delay but keeps idle remux work active. |
| `MEDIAMTX_HLS_SEGMENT_COUNT` | `3` | `2` to `10` | Higher smooths bad networks but increases memory and latency. |
| `MEDIAMTX_HLS_SEGMENT_DURATION` | `2s` | `1s` to `10s` | Lower can reduce latency but increases CPU/IO churn. |
| `MEDIAMTX_HLS_PART_DURATION` | `200ms` | `100ms` to `2s`, must be `< segment duration` | Lower can improve live edge but increases part cadence overhead. |

If an env value is invalid, RTSPanda logs a warning and falls back to safe defaults.

## Verification Steps

### 1) Validate generated config

After start, verify `DATA_DIR/mediamtx.yml` matches expected values.

Docker:

```bash
docker exec rtspanda sh -lc "sed -n '1,160p' /data/mediamtx.yml"
```

Local run:

```powershell
Get-Content .\data\mediamtx.yml
```

### 2) Validate runtime API path settings

Check one path through mediamtx API:

```bash
curl -s http://127.0.0.1:9997/v3/config/get | jq '.paths | to_entries[] | select(.key|startswith("camera-")) | .value | {sourceOnDemand, sourceOnDemandCloseAfter, rtspTransport, rtspAnyPort, record}'
```

### 3) Validate observability

- Backend startup logs now print the resolved streaming profile once.
- mediamtx metrics endpoint should answer on `127.0.0.1:9998/metrics`.

### 4) Basic behavior checks

- Open one camera: stream should become online and HLS playlist should load.
- Close viewer tab for >10s: camera path should stop pulling source when `sourceOnDemand=true`.
- Reopen viewer: stream should recover without manual reset.

## First-Run Safety

- Defaults are conservative and do not require additional secrets or files.
- Invalid env overrides do not crash startup; they degrade to defaults with explicit warnings.
- The same profile is used for both initial YAML render and runtime path add/update API calls, preventing drift over process lifetime.
