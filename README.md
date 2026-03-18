# RTSPanda

**Watch your RTSP cameras in the browser. One app. No cloud. Runs on your machine.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/248Tech/RTSPanda?sort=semver)](https://github.com/248Tech/RTSPanda/releases)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61dafb?logo=react)](https://react.dev/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-2ea44f)](https://github.com/248Tech/RTSPanda)

RTSPanda is a small app you run on your PC or server. You add your camera URLs, open a browser, and watch live—no account, no subscription, no data sent to the cloud. You can also record, take screenshots, run per-camera YOLOv8 tracking, view live overlays + event history, and send rich Discord alerts powered by YOLO or Frigate.

---

## Demo

![RTSPanda demo screenshot](demo/Demo.png)

---

## What is new (v0.0.6)

- **Performance:** stream-status path cache + batch endpoint (`GET /api/v1/cameras/stream-status`) removes dashboard N+1 stream polling calls.
- **Faster frontend load:** route-level code splitting (`React.lazy` + `Suspense`) drops initial JS bundle from ~831 kB to ~202 kB.
- **Observability:** new request logging middleware, Prometheus-compatible `/metrics`, mediamtx metrics on `127.0.0.1:9998`, and system stats endpoint `GET /api/v1/system/stats`.
- **Health and readiness:** new deep-check endpoint `GET /api/v1/health/ready` validates DB connectivity and stream manager readiness.
- **UI/UX updates:** multi-view now supports inline **Add Camera** card picker and quick remove (`✕`) per panel; Settings includes a live **System** monitoring tab.
- **Hardening:** 256 KB request-body limit on camera create/update and DB index migration `008_cameras_index.sql` for camera ordering/query speed.
- **AI docs + planning:** `AI/` documentation refreshed for v0.0.6 handoff/status plus new platform expansion implementation guide.

Release details: [RELEASE_NOTES_v0.0.6.md](RELEASE_NOTES_v0.0.6.md)
Diff: [v0.0.5...v0.0.6](https://github.com/248Tech/RTSPanda/compare/v0.0.5...v0.0.6)

---

## What you need

- **A computer** (Windows, macOS, or Linux)
- **Your camera’s RTSP URL** (looks like `rtsp://admin:password@192.168.1.64:554/stream`)
- **To build once:** Git, Go, and Node.js (we’ll show you how to install them on Windows below)

*Optional:* the **mediamtx** program is what actually pulls the video from your camera. If you don’t add it, RTSPanda still runs and you can add cameras—they’ll just show “offline” until you drop in mediamtx.

---

## Quick Start (any OS)

**1. Get the code**

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
```

**2. Get mediamtx (so you can see video)**

- Go to [mediamtx releases](https://github.com/bluenviron/mediamtx/releases) and download the zip for your system.
- Put the `mediamtx` (or `mediamtx.exe` on Windows) file inside the `mediamtx` folder in RTSPanda.

**3. Build**

- **Windows:** open PowerShell in the RTSPanda folder and run:  
  `.\build.ps1`
- **Mac/Linux:** run:  
  `make build`

**4. Run**

- **Windows:**  
  `.\backend\rtspanda.exe`
- **Mac/Linux:**  
  `./backend/rtspanda`

**5. Open your browser**

Go to **http://localhost:8080**. Click **Settings** → **Cameras** → **Add Camera**, enter a name and your RTSP URL, then go back to the dashboard and click the camera to watch.

Need vendor setup help? Open the **Guides** page in the sidebar for Lorex RTSP lookup, Lorex port-forwarding notes, and Tailscale setup.

---

## One-line Docker setup

Use this if you want the fastest deploy path without local Go/Node setup.

```bash
git clone https://github.com/248Tech/RTSPanda.git && cd RTSPanda && docker compose up --build -d
```

Then open **http://localhost:8080**.

This compose setup starts both services: `rtspanda` (Go backend) and `ai-worker` (FastAPI + YOLOv8).
First build can take longer because Python AI dependencies are large.

Stop it:

```bash
docker compose down
```

Data persists in `./data` on your host.

Windows one-click helper (auto-starts Docker Desktop if needed):

```powershell
.\scripts\docker-up.ps1
```

---

## Windows: one-line install (PowerShell 7)

One command to clone the repo, install Git/Go/Node (if missing), and build. Paste into PowerShell 7:

```powershell
git clone https://github.com/248Tech/RTSPanda.git RTSPanda; cd RTSPanda; .\scripts\install-windows.ps1
```

To have the script also download mediamtx for you:

```powershell
git clone https://github.com/248Tech/RTSPanda.git RTSPanda; cd RTSPanda; .\scripts\install-windows.ps1 -DownloadMediamtx
```

**From CMD** (requires Git and PowerShell 7 installed):

```cmd
git clone https://github.com/248Tech/RTSPanda.git RTSPanda && cd RTSPanda && pwsh -NoProfile -File scripts\install-windows.ps1
```

*Requires:* For the one-liners you need **PowerShell 7** and **Git** (so we can clone). If Git isn’t installed: [git-scm.com](https://git-scm.com/download/win) or `winget install Git.Git`. The script will try to install Go and Node via `winget` if they’re missing.

When the install finishes, run:

```powershell
.\backend\rtspanda.exe
```

and open **http://localhost:8080**.

---

## Windows: PowerShell 7 step-by-step (power users)

Full control: install every dependency yourself, then build and run.

### 1. Install PowerShell 7 (if you’re still on Windows PowerShell 5)

```powershell
winget install Microsoft.PowerShell --accept-package-agreements
```

Close and reopen your terminal; use “PowerShell 7” or `pwsh` so the rest of the commands run in PS7.

### 2. Install Git

```powershell
winget install Git.Git --accept-package-agreements
```

Close and reopen the terminal so `git` is on your PATH.

### 3. Install Go

```powershell
winget install GoLang.Go --accept-package-agreements
```

Again, reopen the terminal so `go` is available.

### 4. Install Node.js (LTS)

```powershell
winget install OpenJS.NodeJS.LTS --accept-package-agreements
```

Reopen the terminal so `node` and `npm` are on your PATH.

### 5. Clone RTSPanda

```powershell
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
```

### 6. (Optional) Download mediamtx

Download the Windows zip from [mediamtx releases](https://github.com/bluenviron/mediamtx/releases) (e.g. `mediamtx_v*_windows_amd64.zip`), unzip it, and copy `mediamtx.exe` into the `mediamtx` folder inside RTSPanda:

```powershell
New-Item -ItemType Directory -Force -Path mediamtx
# Then copy mediamtx.exe from your Downloads into .\mediamtx\
```

Or use the install script to do it for you:

```powershell
.\scripts\install-windows.ps1 -DownloadMediamtx -SkipBuild
```

(Use `-SkipBuild` only if you already built and just want mediamtx.)

### 7. Build RTSPanda

```powershell
.\build.ps1
```

You should see the frontend build, then the Go build. The result is `backend\rtspanda.exe`.

### 8. Run RTSPanda

```powershell
.\backend\rtspanda.exe
```

You should see something like: `RTSPanda listening on :8080 (data: ./data)`.

### 9. Use it

Open **http://localhost:8080** in your browser. Add a camera in **Settings → Cameras**, then click it on the dashboard to watch the stream.

---

## What RTSPanda can do

| Feature | What it means |
|--------|----------------|
| **Live view** | Dashboard with all cameras; click one for full-screen live video. |
| **On-demand** | The app only connects to a camera when someone is watching. |
| **Recording** | Turn on “Record to disk” per camera; get 1-hour MP4 files you can browse and download in the app. |
| **Screenshots** | While watching, hover over the video and click to save a PNG. |
| **YOLOv8 tracking UI** | Configure tracking per camera and run test detections directly in camera view. |
| **Live overlays + history** | Show bounding boxes on live video and browse grouped detection snapshots/events. |
| **Discord alerts + triggers** | Send rich webhook alerts with configurable detection/interval triggers, media options, cooldown, and mention per camera. |
| **Manual Discord media** | Send instant screenshot or clip from camera view with one click. |
| **Legacy alert-rule API** | Optional compatibility webhooks remain available for external automation flows. |
| **REST API** | Manage cameras, get stream status, list recordings, and trigger alerts from code or scripts. |

---

## Configuration

You can change behaviour with environment variables (no config file needed):

| Variable | Default | What it does |
|----------|---------|----------------|
| `PORT` | `8080` | Port the web server uses. |
| `DATA_DIR` | `./data` | Where the database and recordings are stored. |
| `MEDIAMTX_BIN` | auto | Full path to `mediamtx.exe` if it’s not in the `mediamtx` folder. |
| `RCLONE_BIN` | `rclone` | rclone binary path for cloud video storage sync (Dropbox/Drive/OneDrive/Proton Drive). |
| `FFMPEG_BIN` | `ffmpeg` | FFmpeg path for frame capture used by object detection sampling. |
| `DETECTOR_URL` | `http://127.0.0.1:8090` | URL of the async detector worker (`/detect`, `/health`). |
| `FRIGATE_BASE_URL` | unset | Optional Frigate base URL (example `http://frigate:5000`) used to fetch event snapshots for Frigate-powered Discord alerts. |
| `DETECTION_SAMPLE_INTERVAL_SECONDS` | `30` | Global sample interval for camera frame capture. |
| `DETECTION_WORKERS` | `2` | Concurrent async detection worker requests from backend queue. |
| `DETECTION_QUEUE_SIZE` | `128` | Max queued snapshots waiting for detector service. |
| `DISCORD_MOTION_CLIP_SECONDS` | `4` | Default motion-clip length used when camera-specific value is missing. |

AI worker (YOLO) tuning variables:

| Variable | Default | What it does |
|----------|---------|----------------|
| `YOLO_MODEL_PATH` | `/model/yolov8n.onnx` | ONNX model path used by `ai-worker` runtime. |
| `YOLO_CONFIDENCE` | `0.25` | Base confidence threshold before backend camera filters are applied. Lower for more detections. |
| `YOLO_IOU` | `0.45` | NMS IoU threshold. Lower values suppress overlapping boxes more aggressively. |
| `YOLO_MAX_DETECTIONS` | `100` | Maximum objects returned per frame. |
| `YOLO_LOG_LEVEL` | `INFO` | AI worker logging verbosity (`DEBUG`, `INFO`, `WARNING`, etc). |

**Example (different port and data folder):**

```powershell
$env:PORT = "9000"; $env:DATA_DIR = "C:\rtspanda-data"; .\backend\rtspanda.exe
```

### External Video Storage

RTSPanda can auto-sync recordings to:

- Local Server (NAS/SMB/NFS path)
- Dropbox
- Google Drive
- OneDrive
- Proton Drive

Setup guide: [docs/EXTERNAL_VIDEO_STORAGE.md](docs/EXTERNAL_VIDEO_STORAGE.md)

---

## REST API (for scripts and power users)

Everything the web UI does can be done over HTTP. Base URL: **http://localhost:8080/api/v1**

| What | Method | Path |
|------|--------|------|
| List cameras | `GET` | `/cameras` |
| Add camera | `POST` | `/cameras` |
| Get one camera | `GET` | `/cameras/{id}` |
| Update camera | `PUT` | `/cameras/{id}` |
| Delete camera | `DELETE` | `/cameras/{id}` |
| Stream status + HLS URL | `GET` | `/cameras/{id}/stream` |
| List recordings | `GET` | `/cameras/{id}/recordings` |
| Download recording | `GET` | `/cameras/{id}/recordings/{filename}` |
| Alert rules | `GET` / `POST` | `/cameras/{id}/alerts` |
| Send alert event (webhook) | `POST` | `/alerts/{id}/events` |
| Detection health | `GET` | `/detections/health` |
| Trigger test frame capture | `POST` | `/cameras/{id}/detections/test-frame` |
| Trigger test detection | `POST` | `/cameras/{id}/detections/test` |
| Send screenshot to Discord | `POST` | `/cameras/{id}/discord/screenshot` |
| Send recording to Discord | `POST` | `/cameras/{id}/discord/record` |
| Ingest Frigate detection event | `POST` | `/frigate/events` |
| List recent detection events | `GET` | `/detection-events` |
| Get snapshot for event | `GET` | `/detection-events/{id}/snapshot` |
| Health check | `GET` | `/health` |

Example: add a camera with PowerShell:

```powershell
Invoke-RestMethod -Method POST -Uri "http://localhost:8080/api/v1/cameras" -ContentType "application/json" -Body '{"name":"Front Door","rtsp_url":"rtsp://admin:password@192.168.1.10:554/stream","enabled":true}'
```

More examples and details: [human/USER_GUIDE.md](human/USER_GUIDE.md).

---

## Security

- RTSPanda has **no login screen**. Use it on a trusted network, behind a VPN, or behind a reverse proxy (e.g. nginx with password).
- Camera passwords are stored in the SQLite database. Keep the `data` folder (and `DATA_DIR`) only readable by people you trust.
- Don’t expose the app directly to the internet without something in front of it (proxy, VPN, etc.).

---

## Development

- **Run backend only:** `cd backend; go run ./cmd/rtspanda`
- **Run frontend with hot reload:** `cd frontend; npm install; npm run dev` (then open http://localhost:5173; API and HLS are proxied to the backend.)
- **Lint:** `cd backend; go vet ./...` and `cd frontend; npm run lint`

Full guide, RTSP URL tips, and troubleshooting: [human/USER_GUIDE.md](human/USER_GUIDE.md).

---

## Project layout

```
RTSPanda/
├── backend/          # Go server and embedded web UI
├── frontend/         # React app (built and embedded into backend)
├── mediamtx/         # Put mediamtx.exe (or mediamtx) here
├── Dockerfile        # Docker image build
├── docker-compose.yml# One-command container run
├── scripts/          # install-windows.ps1, dev helpers
├── human/            # User guide
├── build.ps1         # Windows build
└── Makefile          # Mac/Linux build
```

---

## Quickstart (Folder Map + Run Modes)

This section mirrors `quickstart.md` for an in-README onboarding path.

### 1) Repository map (what matters first)

| Path | Purpose | Key files |
|---|---|---|
| `backend/` | Go API server, stream orchestration, DB migrations, embedded web UI host | `cmd/rtspanda/main.go`, `internal/api/router.go`, `go.mod` |
| `frontend/` | React 19 + Vite UI | `src/`, `package.json`, `vite.config.ts` |
| `ai_worker/` | Python FastAPI + ONNX Runtime detector service (YOLO) | `app/main.py`, `requirements.txt`, `Dockerfile` |
| `scripts/` | Local helper scripts (Windows + Docker + smoke checks) | `install-windows.ps1`, `docker-up.ps1`, `api-smoke.ps1` |
| `data/` | Runtime state (SQLite DB, recordings, snapshots, mediamtx config) | `rtspanda.db`, `recordings/`, `snapshots/` |
| `mediamtx/` | Place `mediamtx` binary here for local non-Docker streaming | `mediamtx.yml.tmpl` (template in repo) |
| `docs/`, `human/` | Product/setup docs and user guide | `docs/RTSP_SETUP.md`, `human/USER_GUIDE.md` |
| `docker-compose.yml`, `Dockerfile` | Containerized deployment path | starts `rtspanda` + `ai-worker` |
| `build.ps1`, `Makefile` | Local build pipeline (frontend build -> embed -> Go build) | produces `backend/rtspanda(.exe)` |

Notes:
- `backend/internal/api/web/` is build output (frontend files copied for embedding).
- `frontend/node_modules/` and `frontend/dist/` are local build artifacts.

### 2) Fastest way to run (recommended)

#### Docker Compose

```bash
docker compose up --build -d
```

Open: <http://localhost:8080>

Stop:

```bash
docker compose down
```

Windows helper (starts Docker Desktop if needed):

```powershell
.\scripts\docker-up.ps1
```

### 3) Local native run (no Docker)

Prereqs:
- Go 1.26+
- Node.js 18+ (repo currently uses Vite 7 / React 19)
- Optional but needed for real playback: `mediamtx` binary in `mediamtx/`

#### Windows

```powershell
cd frontend
npm install
cd ..
.\build.ps1
.\backend\rtspanda.exe
```

#### macOS / Linux

```bash
cd frontend && npm install && cd ..
make build
./backend/rtspanda
```

Open: <http://localhost:8080>

### 4) Development mode (hot reload UI)

Terminal 1 (backend):

```bash
cd backend
go run ./cmd/rtspanda
```

Terminal 2 (frontend):

```bash
cd frontend
npm install
npm run dev
```

Open: <http://localhost:5173>  
(`vite.config.ts` proxies `/api` and `/hls` to `http://localhost:8080`)

### 5) Quick health check

```powershell
.\scripts\api-smoke.ps1
```

This verifies health + basic camera CRUD endpoints against the running server.

### 6) Common gotchas

- If video stays offline locally, check `mediamtx` binary exists (`mediamtx/mediamtx.exe` on Windows or `mediamtx/mediamtx` on macOS/Linux).
- If backend serves "Frontend not built", rebuild so assets are copied into `backend/internal/api/web/`.
- Docker mode already wires detector service (`ai-worker`) through `DETECTOR_URL=http://ai-worker:8090`.

---

## License

MIT. See [LICENSE](LICENSE).

**RTSPanda** — self-hosted, no cloud, no fuss.
