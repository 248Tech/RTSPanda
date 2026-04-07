# RTSPanda v0.1.1 Hotfix.5

**Date:** April 6, 2026  
**Type:** Production hotfix (stream relay / HLS playback)

## Summary

This hotfix restores reliable browser playback for RTSP cameras whose URLs contain query strings (e.g. `?channel=2&subtype=0`), prevents mediamtx from staying down after a failed full reload, defaults HLS to **fMP4** so `index.m3u8` is served for typical Dahua-style RTSP sources, and tunes the web HLS player for mediamtx’s playlist behavior.

## Problem Context

- **YAML:** Unquoted `source` lines in generated `mediamtx.yml` break when the RTSP URL contains `&` (YAML anchor syntax). VLC uses the raw string; mediamtx read a corrupted URL.
- **Template:** Inside `paths` `range`, `{{.Profile.RTSPTransport}}` referred to the wrong context and could fail config render on full reload after **Reset streams**, stopping mediamtx without a successful restart.
- **HLS:** MediaMTX defaulted to **`lowLatency`** HLS for this deployment profile; some RTSP pulls returned **404** on `.../index.m3u8` while the path was otherwise healthy.
- **Player:** hls.js low-latency mode and short timeouts did not match on-demand RTSP + fMP4 HLS well.

## What Changed

### 1) Quoted RTSP `source` in mediamtx config

- Template helper `quoted` uses `strconv.Quote` for `source: ...` in [`backend/internal/streams/mediamtx.go`](backend/internal/streams/mediamtx.go).
- Mirror: [`mediamtx/mediamtx.yml.tmpl`](mediamtx/mediamtx.yml.tmpl).

### 2) Fix `rtspTransport` in camera path block

- Use `{{$.Profile.RTSPTransport}}` inside `range` (not `{{.Profile...}}` on `cameraEntry`).

### 3) Safer full reload (`Reset all streams`)

- Validate/write config **before** stopping mediamtx; abort reload on render failure so the relay keeps running.
- On `startProcess` failure after reload, schedule retry instead of leaving the watchdog in a bad state.

File: [`backend/internal/streams/manager.go`](backend/internal/streams/manager.go).

### 4) Configurable HLS variant; default `fmp4`

- Emit `hlsVariant` in generated YAML; default **`fmp4`** (override with `MEDIAMTX_HLS_VARIANT`: `mpegts` | `fmp4` | `lowLatency`).
- Stream debug profile includes `hls_variant`.

Files: [`backend/internal/streams/mediamtx.go`](backend/internal/streams/mediamtx.go), [`backend/internal/streams/debug.go`](backend/internal/streams/debug.go), [`docker-compose.yml`](docker-compose.yml).

### 5) Web player tuning

- `lowLatencyMode: false`, longer `manifestLoadingTimeOut` / `levelLoadingTimeOut` / `fragLoadingTimeOut` for on-demand RTSP.

File: [`frontend/src/components/VideoPlayer.tsx`](frontend/src/components/VideoPlayer.tsx).

### 6) Regression test

- [`backend/internal/streams/mediamtx_config_test.go`](backend/internal/streams/mediamtx_config_test.go) — asserts quoted `source` for URLs containing `&`.

## Validation Performed

- `go test ./...` in `backend/`
- `npm run build` in `frontend/` (recommended before release)
- Docker: `docker compose build rtspanda` and smoke-test `GET /hls/camera-<id>/index.m3u8` → **200** with `#EXTM3U` when mediamtx is up

## Upgrade Notes

1. Rebuild and redeploy the application image so the new binary and embedded frontend are used.
2. Defaults set **`MEDIAMTX_HLS_VARIANT=fmp4`** in Compose; only set `lowLatency` if you explicitly need LL-HLS and have verified manifest availability.
3. After upgrading, use **`GET /api/v1/cameras/{id}/stream/debug`** if playback fails: check `mediamtx_api_reachable`, `hls_probe_http_status`, and `profile.hls_variant`.

## Known Limitations

- Very high-resolution streams (e.g. 4K) may still stress client decode; consider substream URLs (`subtype=1`) on the camera if browsers struggle.
