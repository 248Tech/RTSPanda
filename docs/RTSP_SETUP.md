# RTSP Source Setup

RTSPanda needs two things for real playback testing:

1. A reachable RTSP source
2. A working `mediamtx` binary

## Recommended Development Setup

### Option A: Real IP camera

Use a real RTSP camera URL, for example:

```text
rtsp://username:password@camera-host:554/stream1
```

Best when:

- you want true end-to-end validation
- you need realistic stream behavior

### Option B: Temporary test source through mediamtx

If you already have an RTSP, RTMP, or SRT source elsewhere, mediamtx can relay it. This is useful when a real camera exists but you want a stable local endpoint.

### Option C: Local synthetic source

Use OBS, FFmpeg, or another broadcaster to publish a local RTSP-compatible stream into mediamtx for testing. This is the easiest fallback when a real camera is not available.

Example workflow:

1. Download `mediamtx` and place `mediamtx.exe` in `mediamtx/`
2. Start mediamtx manually once to verify the binary works
3. Publish a local test stream into mediamtx with your preferred tool
4. Add that RTSP URL to RTSPanda through the API or Settings page

## mediamtx Binary Resolution

RTSPanda looks for the binary in this order:

1. `MEDIAMTX_BIN`
2. `mediamtx/mediamtx.exe`
3. `PATH`

## What Works Without mediamtx

The following still work if mediamtx is missing:

- backend startup
- database migrations
- camera CRUD
- stream status endpoint shape

The following do not work without mediamtx plus a real source:

- live HLS playlist generation
- actual browser playback
- validation of reconnect behavior

## Suggested Validation Flow

1. Start the backend
2. Add one enabled camera through the API
3. Confirm `GET /api/v1/cameras`
4. Confirm `GET /api/v1/cameras/{id}/stream`
5. Open the returned HLS URL once frontend playback is implemented

## Open Decisions

- Which mediamtx version should be used for local dev and Docker
- Whether the team wants to commit only config templates or also provide a download helper script
