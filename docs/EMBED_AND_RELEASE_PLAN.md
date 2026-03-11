# Embed And Release Plan

This document prepares TASK-010 and supports TASK-011.

## Goal

Produce a single Go binary that:

- serves the API
- serves the built frontend
- proxies HLS traffic to mediamtx

Then package that binary in Docker with a persistent data volume.

## Proposed TASK-010 Flow

### Build order

1. Install frontend dependencies
2. Run `npm run build` in `frontend/`
3. Embed `frontend/dist` into the Go binary
4. Build the backend binary

### Backend changes

Create a static file handler in `backend/internal/api/static.go` that:

- uses `embed.FS`
- serves hashed assets directly
- serves `index.html` for non-API SPA routes
- leaves `/api/` and `/hls/` untouched

### Routing expectations

- `/api/*` -> Go API handlers
- `/hls/*` -> reverse proxy to mediamtx
- all other paths -> embedded frontend assets

### Cache guidance

- hashed JS/CSS assets: long-lived cache headers
- `index.html`: short cache or no-store

## Proposed TASK-011 Flow

### Multi-stage Docker build

Stage 1:

- Node image
- install deps
- build frontend

Stage 2:

- Go image
- copy backend sources and built frontend assets
- compile Go binary with embedded frontend

Stage 3:

- lightweight runtime image
- copy RTSPanda binary
- copy mediamtx binary if bundling it
- expose app port

## Runtime Data

Persist:

- SQLite DB
- generated mediamtx config

Suggested volume target:

```text
/app/data
```

## Environment Variables

- `PORT`
- `DATA_DIR`
- `MEDIAMTX_BIN`

## Known Risks

- Windows dev paths differ from Linux container paths
- embedding fails if frontend build is skipped
- final image size depends heavily on how mediamtx is bundled
- Docker path for mediamtx must match `MEDIAMTX_BIN` or local binary resolution logic

## Recommended Next Implementation Notes

- Add a build script that fails fast if `frontend/dist` is missing
- Ensure SPA fallback does not intercept `/api/` or `/hls/`
- Keep local dev frontend separate until embed flow is stable
