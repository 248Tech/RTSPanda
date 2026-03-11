# RTSPanda

**Self-hosted RTSP camera viewer — single binary, zero cloud, full control.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/github/go-mod/go-version/248Tech/RTSPanda?filename=backend%2Fgo.mod)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61dafb?logo=react)](https://react.dev/)

RTSPanda turns any RTSP camera into a browser-ready live view with one executable. No SaaS, no account, no data leaving your network. Add cameras via a clean web UI, watch live HLS streams, capture screenshots, record to disk, and plug in AI or motion detectors via webhooks.

---

## Why RTSPanda?

| You want… | RTSPanda gives you |
|-----------|--------------------|
| **Privacy** | Everything runs on your machine. No telemetry, no cloud. |
| **Simplicity** | One binary. One port. SQLite. No containers required. |
| **Compatibility** | Works with any RTSP camera (Hikvision, Dahua, Reolink, Amcrest, Axis, Tapo, etc.). |
| **Efficiency** | Streams are on-demand — cameras are only connected when someone is watching. |
| **Extensibility** | REST API + alert webhooks ready for motion detection, object detection, or custom automation. |

---

## Quick Start

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| [Go](https://go.dev/dl/) | 1.22+ | Build the backend |
| [Node.js](https://nodejs.org/) | 18+ | Build the frontend (dev) |
| [mediamtx](https://github.com/bluenviron/mediamtx/releases) | latest | RTSP → HLS relay (optional; app runs without it, streams show offline) |

### 1. Clone

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
```

### 2. Get mediamtx (for live video)

Download the [mediamtx release](https://github.com/bluenviron/mediamtx/releases) for your OS and place the binary in the repo:

```
RTSPanda/mediamtx/
├── mediamtx        # Linux / macOS
└── mediamtx.exe    # Windows
```

Or set `MEDIAMTX_BIN` to the path of your mediamtx binary.

### 3. Build

**Linux / macOS / Git Bash:**
```bash
make build
# → backend/rtspanda
```

**Windows PowerShell:**
```powershell
.\build.ps1
# → backend\rtspanda.exe
```

### 4. Run

```bash
./backend/rtspanda          # Linux / macOS
.\backend\rtspanda.exe     # Windows
```

Open **http://localhost:8080**. Add a camera in **Settings → Cameras**, then watch it from the dashboard.

---

## Features

### Live streaming

- **Dashboard** — Grid of all cameras with status (Online / Connecting / Offline).
- **Single-camera view** — Full-width HLS player with native controls (play, volume, fullscreen).
- **On-demand** — mediamtx connects to a camera only when someone is watching; connections close after idle.

### Recording

- Enable **Record to disk** per camera in Settings.
- mediamtx writes 1-hour MP4 segments to `{DATA_DIR}/recordings/camera-{id}/`.
- Browse, download, and delete recordings from the camera detail page.

### Screenshots

- One-click PNG capture from the live stream (button appears on hover over the video).

### AI & alerts

- Define **alert rules** per camera (connectivity, motion, object_detection).
- External scripts or AI systems POST events to `POST /api/v1/alerts/{rule_id}/events`.
- RTSPanda stores rules and event history; detection logic stays in your stack.

### REST API

- **Cameras** — List, create, get, update, delete; stream status + HLS URL per camera.
- **Recordings** — List, download, delete per camera.
- **Alerts** — CRUD rules, list events, webhook for event ingestion.
- **Health** — `GET /api/v1/health` → `{"status":"ok"}`.

Full API details are in the [README API section](#rest-api) and in [human/USER_GUIDE.md](human/USER_GUIDE.md).

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATA_DIR` | `./data` | Database and recordings directory |
| `MEDIAMTX_BIN` | auto-detect | Path to mediamtx binary |

Example:

```bash
DATA_DIR=/var/lib/rtspanda PORT=9000 ./backend/rtspanda
```

---

## REST API (summary)

Base path: **`/api/v1`**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/cameras` | List cameras |
| `POST` | `/cameras` | Create camera |
| `GET` | `/cameras/{id}` | Get camera |
| `PUT` | `/cameras/{id}` | Update camera |
| `DELETE` | `/cameras/{id}` | Delete camera |
| `GET` | `/cameras/{id}/stream` | Stream status + HLS URL |
| `GET` | `/cameras/{id}/recordings` | List recordings |
| `GET` | `/cameras/{id}/recordings/{filename}` | Download recording |
| `DELETE` | `/cameras/{id}/recordings/{filename}` | Delete recording |
| `GET` | `/cameras/{id}/alerts` | List alert rules |
| `POST` | `/cameras/{id}/alerts` | Create alert rule |
| `GET` | `/alerts/{id}/events` | List events for a rule |
| `POST` | `/alerts/{id}/events` | Ingest alert event (webhook) |
| `GET` | `/health` | Health check |

Camera payload example:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Front Door",
  "rtsp_url": "rtsp://admin:password@192.168.1.10:554/stream",
  "enabled": true,
  "record_enabled": false,
  "position": 0,
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

---

## Architecture

- **Single binary** — Go server embeds the built React SPA (`go:embed`). No separate web server.
- **mediamtx** — Runs as a child process; Go manages config, lifecycle, and API (add/remove paths).
- **SQLite** — WAL mode, single file. No external database.
- **On-demand streams** — `sourceOnDemand: true`; no camera connection until a viewer requests the stream.

```
Browser ←→ Go server (HTTP + HLS proxy) ←→ mediamtx (RTSP → HLS) ←→ RTSP cameras
                ↓
           SQLite (cameras, alerts, recordings metadata)
```

---

## Security

- **No built-in auth** in this release. Use a reverse proxy (e.g. nginx with basic auth), VPN (Tailscale, WireGuard), or deploy on a private network.
- **RTSP credentials** are stored in plain text in SQLite. Restrict filesystem access to `DATA_DIR` on multi-user systems.
- Do not expose RTSPanda directly to the public internet without additional protection.

---

## Development

- **Backend only:** `cd backend && go run ./cmd/rtspanda`
- **Frontend dev server:** `cd frontend && npm install && npm run dev` (proxies `/api` and `/hls` to backend)
- **Lint:** `cd backend && go vet ./...`; `cd frontend && npm run lint`

See [human/USER_GUIDE.md](human/USER_GUIDE.md) for detailed setup, RTSP URL tips, and troubleshooting.

---

## Project layout

```
RTSPanda/
├── backend/
│   ├── cmd/rtspanda/          # Entry point
│   └── internal/
│       ├── api/                # HTTP router, handlers, embedded SPA
│       ├── cameras/            # Camera domain (service, repo)
│       ├── alerts/             # Alert rules and events
│       ├── recordings/        # Recording file service
│       ├── streams/            # mediamtx process + config
│       └── db/                 # SQLite and migrations
├── frontend/                   # React + Vite + TypeScript + Tailwind
│   └── src/
│       ├── api/                # Typed API client
│       ├── components/        # UI components (VideoPlayer, CameraCard, etc.)
│       └── pages/              # Dashboard, CameraView, Settings
├── mediamtx/                   # Place mediamtx binary here
├── human/                      # User guide and docs
├── Makefile                    # Unix build
└── build.ps1                   # Windows build
```

---

## License

MIT. See [LICENSE](LICENSE).

---

**RTSPanda** — self-hosted, no cloud, no fuss.
