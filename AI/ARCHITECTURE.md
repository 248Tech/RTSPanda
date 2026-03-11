# RTSPanda — Architecture

## System Overview

RTSPanda is a modular monolith. One binary, one config, one compose file.

```
┌─────────────────────────────────────────────────────┐
│                    Docker Container                  │
│                                                      │
│  ┌──────────────────┐    ┌──────────────────────┐   │
│  │   Go Backend     │    │      mediamtx         │   │
│  │                  │◄───┤  (stream relay)       │   │
│  │  - REST API      │    │  - RTSP ingest        │   │
│  │  - Static files  │    │  - HLS output         │   │
│  │  - Stream mgmt   │    │  - WebRTC (future)    │   │
│  │  - SQLite        │    └──────────────────────┘   │
│  └────────┬─────────┘                               │
│           │                                          │
│  ┌────────▼─────────┐                               │
│  │  React Frontend  │                               │
│  │  (embedded)      │                               │
│  └──────────────────┘                               │
└─────────────────────────────────────────────────────┘
         ▲
         │ HTTP (port 8080)
         │
    Browser (hls.js)
```

---

## Backend — Module Breakdown

### `cmd/rtspanda/main.go`

- Parse config (env vars + optional config file)
- Start SQLite
- Run DB migrations
- Start mediamtx subprocess
- Start HTTP server
- Handle graceful shutdown

### `internal/db/`

- `db.go` — open SQLite connection, run migrations
- `migrations/` — sequential SQL migration files
- `cameras.go` — camera DB queries (CRUD)
- `settings.go` — settings key/value queries

### `internal/cameras/`

- `model.go` — Camera struct
- `service.go` — business logic for camera CRUD
- `repository.go` — wraps DB queries, returns domain types

### `internal/streams/`

- `manager.go` — lifecycle for active streams
- `mediamtx.go` — manages mediamtx process + config generation
- `health.go` — polls mediamtx for stream status

### `internal/api/`

- `router.go` — sets up all HTTP routes
- `cameras.go` — camera REST handlers
- `streams.go` — stream status + HLS URL endpoint
- `static.go` — serves embedded React frontend

---

## Database Schema

### `cameras` table

```sql
CREATE TABLE cameras (
    id          TEXT PRIMARY KEY,          -- UUID
    name        TEXT NOT NULL,
    rtsp_url    TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    position    INTEGER NOT NULL DEFAULT 0, -- grid sort order
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);
```

### `settings` table

```sql
CREATE TABLE settings (
    key     TEXT PRIMARY KEY,
    value   TEXT NOT NULL
);
```

---

## REST API Design

Base path: `/api/v1`

| Method | Path                    | Description              |
|--------|-------------------------|--------------------------|
| GET    | `/api/v1/cameras`       | List all cameras         |
| POST   | `/api/v1/cameras`       | Add camera               |
| GET    | `/api/v1/cameras/:id`   | Get camera               |
| PUT    | `/api/v1/cameras/:id`   | Update camera            |
| DELETE | `/api/v1/cameras/:id`   | Delete camera            |
| GET    | `/api/v1/cameras/:id/stream` | Get stream URL + status |
| GET    | `/api/v1/health`        | Health check             |

Static files: all non-`/api/` requests → serve embedded React app.

---

## Streaming Pipeline Detail

### Phase 1: HLS via mediamtx

1. User adds camera RTSP URL via API
2. Backend writes mediamtx config entry for that camera
3. mediamtx ingests RTSP stream and outputs HLS segments
4. Frontend fetches HLS URL from `/api/v1/cameras/:id/stream`
5. hls.js player loads the `.m3u8` playlist
6. Browser plays video — no direct camera connection

### mediamtx Configuration (generated per camera)

```yaml
paths:
  camera-{id}:
    source: rtsp://user:pass@camera-ip:554/stream
    sourceOnDemand: true       # only connect when someone is watching
    sourceOnDemandCloseAfter: 10s
```

`sourceOnDemand: true` is critical — streams must not stay open when nobody is watching.

### Phase 2: WebRTC (future)

Replace HLS with WebRTC signaling via pion/webrtc embedded in Go.
Lower latency (~100-500ms vs 2-6s for HLS).

---

## Frontend Architecture

### Pages

- `/` — Camera dashboard grid (all cameras)
- `/cameras/:id` — Single camera full view
- `/settings` — Add/edit/remove cameras

### Components

- `CameraGrid` — responsive grid of camera cards
- `CameraCard` — thumbnail + name + status indicator
- `VideoPlayer` — hls.js wrapper component
- `CameraForm` — add/edit camera modal
- `StatusBadge` — online/offline/connecting indicator

### State Management

- Camera list: React context or Zustand store
- Stream state: local component state + polling
- No global state library needed for Phase 1

### API Client

Thin typed wrapper in `src/api/`:

```typescript
// src/api/cameras.ts
export async function getCameras(): Promise<Camera[]>
export async function addCamera(data: CreateCameraInput): Promise<Camera>
export async function deleteCamera(id: string): Promise<void>
export async function getStreamUrl(id: string): Promise<StreamInfo>
```

---

## Deployment

### Docker

- Multi-stage Dockerfile
- Stage 1: Build Go binary (with embedded frontend)
- Stage 2: Minimal runtime image (alpine or distroless)
- mediamtx binary bundled into the image

### docker-compose.yml

```yaml
services:
  rtspanda:
    image: rtspanda:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data     # SQLite database
    environment:
      - DATA_DIR=/data
```

### install.sh

One-command install:

```bash
curl -fsSL https://raw.githubusercontent.com/.../install.sh | bash
```

---

## Important Rules

1. Browser never touches camera directly
2. mediamtx runs as a subprocess managed by Go — not a separate container
3. Streams only open when actively viewed (`sourceOnDemand`)
4. SQLite DB stored in a mounted volume — survives restarts
5. Frontend is embedded in the Go binary — no separate file serving needed
6. No auth in Phase 1 — design for it (middleware stubs OK) but don't implement
7. Keep Go binary under 50MB, Docker image under 150MB
