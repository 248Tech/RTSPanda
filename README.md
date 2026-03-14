# RTSPanda

**Watch your RTSP cameras in the browser. One app. No cloud. Runs on your machine.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/248Tech/RTSPanda?sort=semver)](https://github.com/248Tech/RTSPanda/releases)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61dafb?logo=react)](https://react.dev/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-2ea44f)](https://github.com/248Tech/RTSPanda)

RTSPanda is a small app you run on your PC or server. You add your camera URLs, open a browser, and watch live—no account, no subscription, no data sent to the cloud. You can also record, take screenshots, run per-camera YOLOv8 tracking, view live overlays + event history, and send rich Discord alerts.

---

## Demo

![RTSPanda demo screenshot](demo/Demo.png)

---

## What is new (v0.0.3)

- Fixes detector reliability in Docker by stabilizing `ai-worker` runtime dependencies and improving detector URL fallback behavior.
- Adds verbose YOLO request/detection logging in backend and AI worker for faster troubleshooting.
- Adds per-camera Discord trigger controls: detection toggle, interval screenshots, interval seconds, clip include toggle, and clip duration.
- Adds manual Discord actions in camera view: `Screenshot to Discord` and `Record to Discord` (default 60s, configurable format/duration).
- Adds richer Discord media generation with `webm`, `webp`, and `gif` format fallback.
- Rewords settings from generic AI wording to YOLO-focused wording, while retaining legacy alert-rule webhook APIs for compatibility.

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
| `FFMPEG_BIN` | `ffmpeg` | FFmpeg path for frame capture used by object detection sampling. |
| `DETECTOR_URL` | `http://127.0.0.1:8090` | URL of the async detector worker (`/detect`, `/health`). |
| `DETECTION_SAMPLE_INTERVAL_SECONDS` | `30` | Global sample interval for camera frame capture. |
| `DETECTION_WORKERS` | `2` | Concurrent async detection worker requests from backend queue. |
| `DETECTION_QUEUE_SIZE` | `128` | Max queued snapshots waiting for detector service. |
| `DISCORD_MOTION_CLIP_SECONDS` | `4` | Default motion-clip length used when camera-specific value is missing. |

**Example (different port and data folder):**

```powershell
$env:PORT = "9000"; $env:DATA_DIR = "C:\rtspanda-data"; .\backend\rtspanda.exe
```

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

## License

MIT. See [LICENSE](LICENSE).

**RTSPanda** — self-hosted, no cloud, no fuss.
