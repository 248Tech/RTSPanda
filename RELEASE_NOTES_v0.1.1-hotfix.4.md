# RTSPanda v0.1.1 Hotfix.4

**Date:** March 23, 2026  
**Commit:** `827121b`  
**Type:** Production hotfix (Pi playback diagnostics)

## Summary

This hotfix adds deep stream diagnostics for Raspberry Pi playback regressions where camera APIs are healthy but HLS playback fails with manifest `404` responses.

The key addition is a new debug endpoint that returns a full per-camera snapshot of stream state from RTSPanda and mediamtx, including runtime path data, config path data, and direct HLS probe results.

## Problem Context

Observed production behavior on Pi mode:

- `GET /api/v1/health` returns `200`
- `GET /api/v1/cameras/{id}/stream` returns `200`
- Browser player shows `Initializing` / `Network error`
- HLS proxy logs show:
  - `hls: upstream status path=/camera-<id>/index.m3u8 status=404`

This indicates mediamtx is not serving the HLS manifest for that path at request time.

## What Changed

### 1) New debug API endpoint

Added:

- `GET /api/v1/cameras/{id}/stream/debug`

Files:

- `backend/internal/api/router.go`
- `backend/internal/api/streams.go`
- `backend/internal/streams/debug.go`

This endpoint returns a single payload containing:

- App-level stream status (`app_stream_status`)
- Effective stream profile (`source_on_demand`, `hls_always_remux`, `rtsp_transport`, etc.)
- Path list snapshot from mediamtx `/v3/paths/list`
- Direct mediamtx runtime path read from `/v3/paths/get/{name}`
- Direct mediamtx config path read from `/v3/config/paths/get/{name}`
- Direct HLS probe (`hls_probe_http_status`, error, body prefix)

### 2) Batch stream status HLS URL alignment

In batch `GET /api/v1/cameras/stream-status`, `hls_url` is now returned for enabled cameras (matching single-camera stream endpoint behavior), not only when status is `online`.

File:

- `backend/internal/api/streams.go`

### 3) Lifecycle logging on path add

Added explicit log when a mediamtx path is successfully created:

- `streams: mediamtx path added name=camera-<id> camera=<id>`

File:

- `backend/internal/streams/manager.go`

## API Usage

### Request

```http
GET /api/v1/cameras/{camera_id}/stream/debug
```

### Diagnostic interpretation

- `mediamtx_paths_get_http_status = 404`:
  - path is not present in mediamtx runtime
- `mediamtx_paths_get_http_status = 200` + `hls_probe_http_status = 404`:
  - path exists, but HLS muxer/playlist is not available
- `path_list_error` populated:
  - mediamtx API list call is failing (connectivity or API issue)

## Validation Performed

- `go test ./...` passed in `backend/`
- Endpoint wiring and interface updates compile cleanly
- Debug payload includes both runtime and config mediamtx state for rapid triage

## Known Limitations

- This hotfix improves observability and triage speed; it does not by itself guarantee HLS playback recovery.
- Root-cause remediation for persistent HLS `404` remains dependent on diagnosis from the new debug payload.

## Upgrade Notes

After updating, use the new debug endpoint immediately when playback fails:

1. Open camera page and reproduce playback issue.
2. Call `/api/v1/cameras/{id}/stream/debug`.
3. Correlate endpoint response with `hls: upstream ... status=404` logs.
4. Adjust stream profile/runtime settings based on actual mediamtx state.
