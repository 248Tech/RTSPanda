# RTSPanda — User Guide

> Start to finish: installation, first camera, live streaming, recording, screenshots, YOLOv8 tracking, Frigate integration, and Discord alerts.

---

## Table of Contents

1. [What is RTSPanda?](#1-what-is-rtspanda)
2. [How it works (plain English)](#2-how-it-works-plain-english)
3. [What you need before you start](#3-what-you-need-before-you-start)
4. [Installation](#4-installation)
   - [Step 1 — Get the code](#step-1--get-the-code)
   - [Step 2 — Get the mediamtx binary](#step-2--get-the-mediamtx-binary)
   - [Step 3 — Build RTSPanda](#step-3--build-rtspanda)
   - [Step 4 — Run it](#step-4--run-it)
5. [Your first camera](#5-your-first-camera)
6. [Watching cameras](#6-watching-cameras)
   - [Dashboard](#dashboard)
   - [Single-camera view](#single-camera-view)
   - [Stream status explained](#stream-status-explained)
7. [Screenshots](#7-screenshots)
8. [Recording](#8-recording)
   - [Enabling recording](#enabling-recording)
   - [Browsing and downloading recordings](#browsing-and-downloading-recordings)
   - [Deleting recordings](#deleting-recordings)
9. [Detection and Discord Alerts](#9-detection-and-discord-alerts)
   - [YOLOv8 tracking per camera](#yolov8-tracking-per-camera)
   - [Live overlays in camera view](#live-overlays-in-camera-view)
   - [Detection event history panel](#detection-event-history-panel)
   - [Discord rich media alerts](#discord-rich-media-alerts)
   - [Manual Discord actions in camera view](#manual-discord-actions-in-camera-view)
   - [Legacy alert rules and webhooks (advanced)](#legacy-alert-rules-and-webhooks-advanced)
10. [Managing cameras](#10-managing-cameras)
11. [Finding your RTSP URL](#11-finding-your-rtsp-url)
12. [Testing without a real camera](#12-testing-without-a-real-camera)
13. [Environment variables](#13-environment-variables)
14. [Development mode](#14-development-mode)
15. [Troubleshooting](#15-troubleshooting)
16. [Security](#16-security)
17. [API quick reference](#17-api-quick-reference)

---

## 1. What is RTSPanda?

RTSPanda is a **self-hosted camera viewer**. You run it on your own computer or server, point it at your RTSP cameras, and watch them live in any browser — no cloud account, no subscription, no data leaving your network.

It ships as a **single file** that contains the entire web server and user interface. You run one command and open a browser.

**What it does today:**

- Shows a live grid of all your cameras
- Lets you click into a full-screen single-camera view
- Saves screenshot PNGs with one click while watching
- Records continuous MP4 segments to disk per camera (optional)
- Lets you browse, download, and delete recordings from the browser
- Runs YOLOv8 detection per camera with configurable thresholds and label filters
- Draws live bounding-box overlays on video playback
- Stores detection history with snapshot previews
- Sends rich Discord webhook alerts with snapshot/clip media and manual push actions (optional), using YOLO or Frigate as the detection source

**What it is not:**

- Not a cloud product — everything stays on your machine
- Not a full NVR — it is lightweight by design
- Not a full enterprise VMS/NVR platform

---

## 2. How it works (plain English)

```
Your RTSP camera
      │
      ▼
  mediamtx  ←── runs inside RTSPanda, manages stream conversion
      │
      ▼
  RTSPanda (Go server)  ←── serves the web UI and REST API
      │
      ▼
  Your browser  ←── watches via HLS (standard video protocol)
```

**The key point:** your browser never connects directly to your camera. RTSPanda's built-in relay (mediamtx) handles that connection and converts RTSP to a format the browser understands (HLS). This means:

- Cameras behind NAT or firewalls work fine as long as RTSPanda can reach them
- Browser compatibility is excellent — HLS works on Chrome, Firefox, Safari, Edge
- The camera only gets one connection regardless of how many browser tabs are open

**Stream latency:** HLS has 2–6 seconds of inherent buffering. This is normal and expected. It is a protocol trade-off in exchange for universal browser support.

**On-demand connections:** streams open the moment you click to watch and close about 10 seconds after you navigate away. If nobody is watching, no connection is made to your camera at all. This saves CPU, bandwidth, and camera connection slots.

---

## 3. What you need before you start

| Requirement | Minimum version | Where to get it |
|-------------|----------------|-----------------|
| Go | 1.26 | [golang.org/dl](https://golang.org/dl/) |
| Node.js | 18 | [nodejs.org](https://nodejs.org/) |
| npm | 9 | Comes with Node.js |
| Git | any | [git-scm.com](https://git-scm.com/) |
| mediamtx binary | latest | [github.com/bluenviron/mediamtx/releases](https://github.com/bluenviron/mediamtx/releases) |

**Do you need mediamtx to get started?** No — RTSPanda starts and runs without it. The web UI will load, you can add cameras, and everything works except actual video playback. If you just want to explore the interface, skip mediamtx for now.

**Operating systems:** Windows, macOS, and Linux are all supported.

---

## 4. Installation

### Step 1 — Get the code

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
```

### Step 2 — Get the mediamtx binary

Go to the [mediamtx releases page](https://github.com/bluenviron/mediamtx/releases) and download the archive for your platform:

| Platform | File to download |
|----------|-----------------|
| Windows 64-bit | `mediamtx_vX.X.X_windows_amd64.zip` |
| macOS Apple Silicon | `mediamtx_vX.X.X_darwin_arm64.tar.gz` |
| macOS Intel | `mediamtx_vX.X.X_darwin_amd64.tar.gz` |
| Linux 64-bit | `mediamtx_vX.X.X_linux_amd64.tar.gz` |

Extract the archive and place the binary inside the `mediamtx/` folder of the repo:

```
rtspanda/
└── mediamtx/
    ├── mediamtx        ← Linux / macOS
    └── mediamtx.exe    ← Windows
```

On Linux/macOS, make it executable:

```bash
chmod +x mediamtx/mediamtx
```

> **Alternative:** If mediamtx is already on your system `PATH`, RTSPanda will find it automatically. Or set the `MEDIAMTX_BIN` environment variable to its absolute path.

### Step 3 — Build RTSPanda

This compiles the React frontend and embeds it into the Go binary.

**Linux / macOS / Git Bash:**

```bash
make build
```

**Windows PowerShell:**

```powershell
.\build.ps1
```

The build takes 30–60 seconds. When done you will have:

```
backend/rtspanda        ← Linux / macOS
backend/rtspanda.exe    ← Windows
```

### Step 4 — Run it

```bash
./backend/rtspanda          # Linux / macOS
.\backend\rtspanda.exe      # Windows
```

You should see:

```
RTSPanda listening on :8080 (data: ./data)
```

Open [http://localhost:8080](http://localhost:8080) in your browser. The dashboard loads immediately.

Need vendor-specific setup help? Open the **Guides** page in the sidebar for Lorex RTSP steps, Lorex port-forwarding notes, and Tailscale remote-access setup.

> **Data storage:** RTSPanda creates a `data/` folder next to the binary. This holds the SQLite database (`rtspanda.db`) and all recordings. You can change the location with the `DATA_DIR` environment variable (see [Environment variables](#13-environment-variables)).

---

## 5. Your first camera

1. In the browser, click **Settings** (gear icon, top-right corner of the navbar).

2. You land on the **Cameras** tab. Click **+ Add Camera**.

3. Fill in the form:

   | Field | What to put here |
   |-------|-----------------|
   | **Name** | A friendly label, e.g. `Front Door` |
   | **RTSP URL** | The full RTSP address of your camera — see [Finding your RTSP URL](#11-finding-your-rtsp-url) |
   | **Enabled** | Leave ticked. Untick to pause the camera without deleting it. |
   | **Record to disk** | Tick this if you want RTSPanda to save continuous MP4 footage. |

4. Click **Add Camera**. The camera appears in the list immediately.

5. Click the RTSPanda logo or **Cameras** heading to go back to the dashboard.

6. Your camera card appears. The status badge will show **Connecting** briefly, then **Online** once mediamtx establishes the stream. If it shows **Offline**, see [Troubleshooting](#15-troubleshooting).

---

## 6. Watching cameras

### Dashboard

The dashboard is the main screen. It shows a grid of all your cameras:

- **1 column** on mobile
- **2 columns** on tablets
- **3 columns** on desktop
- **4 columns** on wide screens

Each card shows:
- A 16:9 placeholder area (the grid never loads live video — only the single-camera view does, to keep CPU usage low)
- The camera name (left) and stream status badge (right)

Click any card to open the single-camera view for that camera.

### Single-camera view

This is where live video plays. What you see:

- **Top bar:** a back arrow to return to the dashboard, and a Settings link
- **Video player:** full-width, 16:9, with native browser controls (play/pause, volume, fullscreen)
- **Camera info:** name, RTSP URL, and stream status badge below the player
- **Recordings panel:** list of recorded files for this camera (appears below the player)

The stream starts loading immediately when you open this view. Expect 2–6 seconds before video appears — this is normal HLS buffering.

### Stream status explained

| Badge | Colour | Meaning |
|-------|--------|---------|
| **Online** | Green | mediamtx has an active connection to the camera and is delivering video |
| **Connecting** | Amber (pulsing) | mediamtx is attempting to connect — wait a few seconds |
| **Offline** | Red | mediamtx cannot reach the camera, or recording is disabled |

**Offline does not always mean something is wrong.** If you have just added the camera and are not currently watching the camera view, the status will show Offline because the stream is on-demand and hasn't been opened yet. Navigate to the camera view and wait a few seconds — status will move to Connecting then Online.

---

## 7. Screenshots

While watching a live stream:

1. **Hover over the video.** A **Screenshot** button appears in the top-right corner of the player.
2. **Click Screenshot.** The current video frame is captured and downloaded as a PNG to your browser's default download folder.

The filename is formatted as: `{camera name}_{timestamp}.png`
Example: `Front_Door_2025-01-15T10-30-00.png`

> Screenshots are taken client-side from the video frame already in your browser. No round-trip to the server is needed. The button only appears when the stream is actively playing.

---

## 8. Recording

RTSPanda can record each camera's stream continuously to MP4 files on disk.

### Enabling recording

There are two ways:

**When adding a camera:**
Tick **Record to disk** in the Add Camera form.

**On an existing camera:**
1. Go to **Settings → Cameras**
2. Click **Edit** next to the camera
3. Tick **Record to disk**
4. Click **Save**

Once enabled, mediamtx writes MP4 segments to:

```
data/recordings/camera-{id}/YYYY-MM-DD_HH-MM-SS.mp4
```

Segments are 1 hour long. A new file is created automatically at each hour boundary.

> **Disk space:** A 1080p H.264 stream typically uses 500 MB–2 GB per hour depending on bitrate. Plan your storage accordingly. There is no automatic cleanup — delete files manually or via the UI.

### Browsing and downloading recordings

1. Navigate to any camera's single-camera view (click its card on the dashboard)
2. Scroll below the video player to the **Recordings** section
3. Recordings are listed newest first, showing filename, file size, and last-modified timestamp
4. Click the **download icon** (arrow) next to a recording to download it as an MP4 file

The downloaded file plays in any media player — VLC, QuickTime, Windows Media Player, etc.

### Deleting recordings

In the Recordings panel, click the **trash icon** next to any recording. You will see a spinner while the file is deleted, then it disappears from the list. Deletion is permanent.

---

## 9. Detection and Discord Alerts

### YOLOv8 tracking per camera

YOLOv8 tracking is configured per camera in **Settings → Cameras → Add/Edit**.

Tracking fields:

- **YOLOv8 Tracking: Enabled**
- **Sample interval (seconds)**: how often a frame is captured for inference
- **Minimum confidence (0–1)**: detections below this are ignored
- **Track labels (optional)**: comma-separated allow-list (for example `person, car`)

When tracking is enabled, RTSPanda samples frames with FFmpeg, sends them to the AI worker, and stores detections as events.

### Live overlays in camera view

In single-camera view:

- Click **Overlay On/Off** to show or hide live bounding boxes
- Overlays show the label and confidence for recent detections
- Overlay placement is scaled against source frame dimensions so boxes line up across aspect ratios

### Detection event history panel

In single-camera view, below the player, the **Detection Event History** panel shows:

- Grouped detection events by snapshot timestamp
- Snapshot preview images
- Labels and confidence chips for each detected object

Use **Refresh** to pull latest events immediately.

### Discord rich media alerts

Discord alerts are also configured per camera in **Settings → Cameras → Add/Edit**.

Fields:

- **Discord Rich Alerts: Enabled**
- **Webhook URL** (required when enabled)
- **Mention** (optional, e.g. `@here` or `<@123...>`)
- **Cooldown (seconds)** to avoid spam
- **Detection source** (`RTSPanda YOLOv8` or `Frigate`)
- **Frigate camera name** (optional override when Frigate source is selected)
- **Trigger on detections** (uses your selected source)
- **Trigger on interval screenshots**
- **Screenshot interval (seconds)** for interval trigger mode
- **Include motion clip on detection alerts**
- **Motion clip seconds**
- **Manual record format** (`webp`, `webm`, or `gif`)
- **Manual record seconds**

If you choose **Frigate**, configure Frigate to POST events to:

- `POST /api/v1/frigate/events`

Optional: set `FRIGATE_BASE_URL` in RTSPanda so snapshot media can be fetched and attached to Discord alerts automatically.

When active, detection batches (YOLO or Frigate) send a Discord embed with:

- Camera name and detection summary
- Confidence and bbox details
- Attached snapshot image (rich media)
- Optional clip attachment with format fallback (`webm` -> `webp` -> `gif`)

### Manual Discord actions in camera view

In single-camera view, you can trigger media pushes without waiting for a detection:

- **Screenshot to Discord**: captures one frame and sends it to the configured webhook.
- **Record to Discord**: records and sends a clip using the camera's configured format and duration.

Both buttons require the camera-level Discord webhook URL to be configured.

### Legacy alert rules and webhooks (advanced)

The **Alert Rules** tab keeps legacy alert-rule APIs for compatibility with external systems.
Primary alerting should now be configured in camera-level provider-based Discord settings.
If you already have an external workflow, you can keep using:

- `POST /api/v1/alerts/{id}/events`

for manual or third-party alert event ingestion.

---

## 10. Managing cameras

All camera management is in **Settings → Cameras**.

### Edit a camera

Click **Edit** on any camera row. You can change:
- Name
- RTSP URL
- Enabled / disabled state
- Record to disk on/off
- YOLOv8 tracking settings (toggle, interval, threshold, labels)
- Discord alert settings (webhook, mention, cooldown, trigger mode, interval screenshots, clip include/duration, record format/duration)

Changes take effect immediately. If you change the RTSP URL, mediamtx reconnects to the new address.

### Disable a camera without deleting it

Edit the camera and untick **Enabled**. The stream stops and the card shows Offline on the dashboard. Re-enable at any time by editing again.

### Delete a camera

Click **Delete** on a camera row. A confirmation dialog appears. Confirm to delete permanently.

Deleting a camera:
- Removes it from the database
- Stops the mediamtx stream
- Does **not** delete its recordings — those stay in `data/recordings/camera-{id}/` and must be deleted manually

### Reorder cameras

Camera order on the dashboard follows the `position` field. Currently this must be set via the API. A drag-and-drop reorder UI is planned.

```bash
curl -X PUT http://localhost:8080/api/v1/cameras/{id} \
  -H "Content-Type: application/json" \
  -d '{"position": 2}'
```

Lower numbers appear first.

---

## 11. Finding your RTSP URL

Every camera brand uses a slightly different URL format. Below are common patterns:

### Generic format

```
rtsp://username:password@camera-ip:port/stream-path
```

### Common camera brands

| Brand | Common URL format |
|-------|------------------|
| Hikvision | `rtsp://admin:password@192.168.1.64:554/Streaming/Channels/101` |
| Dahua | `rtsp://admin:password@192.168.1.64:554/cam/realmonitor?channel=1&subtype=0` |
| Reolink | `rtsp://admin:password@192.168.1.64:554/h264Preview_01_main` |
| Amcrest | `rtsp://admin:password@192.168.1.64:554/cam/realmonitor?channel=1&subtype=0` |
| Axis | `rtsp://root:password@192.168.1.64:554/axis-media/media.amp` |
| Hanwha/Samsung | `rtsp://admin:password@192.168.1.64:554/profile2/media.smp` |
| TP-Link Tapo | `rtsp://username:password@192.168.1.64:554/stream1` |
| Wyze | Requires third-party firmware (Wyze cameras do not natively support RTSP on stock firmware) |

### Finding your camera's IP

- Check your router's DHCP client list
- Use your camera's manufacturer app — the IP is usually shown in device settings
- Run a network scanner like `nmap -sn 192.168.1.0/24`

### Checking if RTSP is enabled

Some cameras have RTSP disabled by default. Look for:
- **RTSP** or **Video streaming** in your camera's web interface or app
- Port 554 — the default RTSP port

### Testing your RTSP URL before adding it to RTSPanda

Use VLC: open VLC → Media → Open Network Stream → paste the URL. If video plays in VLC, it will work in RTSPanda.

```bash
# Or with ffplay (part of FFmpeg)
ffplay rtsp://admin:password@192.168.1.64:554/stream
```

---

## 12. Testing without a real camera

If you don't have an RTSP camera yet but want to explore RTSPanda, you can publish a test stream using one of these methods:

### Option A — OBS Studio (easiest)

1. Download [OBS Studio](https://obsproject.com/)
2. Go to Settings → Stream → Service: Custom, Server: `rtsp://localhost:554/test`
3. Set OBS to output via the RTSP server built into newer OBS versions, or use the OBS-RTSP-Server plugin
4. In RTSPanda, add a camera with URL `rtsp://localhost:554/test`

### Option B — FFmpeg

If you have FFmpeg installed, generate a test pattern stream:

```bash
ffmpeg -re -f lavfi -i testsrc=size=1280x720:rate=25 \
  -vcodec libx264 -preset ultrafast \
  -f rtsp rtsp://localhost:8554/test
```

This publishes to mediamtx's RTSP input port (8554 by default). In RTSPanda, add:

```
rtsp://localhost:8554/test
```

### Option C — A public RTSP stream

Some public test streams exist online. Search for "public RTSP stream test" — note that these are not reliable long-term.

---

## 13. Environment variables

Set these before running the binary to customise behaviour. No config file is needed.

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port RTSPanda listens on |
| `DATA_DIR` | `./data` | Where to store the database and recordings |
| `MEDIAMTX_BIN` | auto | Explicit path to mediamtx binary |
| `FFMPEG_BIN` | `ffmpeg` | FFmpeg binary used for detection frame capture |
| `DETECTOR_URL` | `http://127.0.0.1:8090` | AI worker base URL |
| `FRIGATE_BASE_URL` | unset | Optional Frigate URL used to download snapshots when Frigate alerts are enabled |
| `DETECTION_SAMPLE_INTERVAL_SECONDS` | `30` | Global detection sample interval fallback |
| `DETECTION_WORKERS` | `2` | Detection worker concurrency |
| `DETECTION_QUEUE_SIZE` | `128` | In-memory detection queue capacity |
| `DISCORD_MOTION_CLIP_SECONDS` | `4` | Default clip seconds used when camera-level clip duration is unset |
| `YOLO_MODEL` | `yolov8n.pt` | AI worker model (worker container/env) |
| `YOLO_CONFIDENCE` | `0.25` | AI worker baseline confidence (before per-camera filtering) |

### Examples

**Change the port:**
```bash
PORT=9000 ./backend/rtspanda
```

**Store data on a separate drive:**
```bash
DATA_DIR=/mnt/recordings ./backend/rtspanda
```

**Use a mediamtx binary in a custom location:**
```bash
MEDIAMTX_BIN=/opt/mediamtx/mediamtx ./backend/rtspanda
```

**Point RTSPanda to a remote detector worker:**
```bash
DETECTOR_URL=http://10.0.0.50:8090 ./backend/rtspanda
```

**Tune global detection sampling fallback:**
```bash
DETECTION_SAMPLE_INTERVAL_SECONDS=10 DETECTION_WORKERS=4 ./backend/rtspanda
```

**Windows PowerShell equivalent:**
```powershell
$env:PORT = "9000"
$env:DATA_DIR = "D:\recordings"
.\backend\rtspanda.exe
```

### Running as a service (Linux systemd)

Create `/etc/systemd/system/rtspanda.service`:

```ini
[Unit]
Description=RTSPanda camera viewer
After=network.target

[Service]
Type=simple
ExecStart=/opt/rtspanda/rtspanda
Environment=DATA_DIR=/var/lib/rtspanda
Environment=PORT=8080
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now rtspanda
```

---

## 14. Development mode

If you are building on top of RTSPanda or modifying it, you can run the frontend and backend separately so changes hot-reload.

**Terminal 1 — backend:**

```bash
cd backend
go run ./cmd/rtspanda
# Runs on http://localhost:8080
```

**Terminal 2 — frontend dev server:**

```bash
cd frontend
npm install        # first time only
npm run dev
# Opens http://localhost:5173
# API requests (/api, /hls) are automatically proxied to :8080
```

Use `http://localhost:5173` during development. The frontend auto-reloads when you save a file.

**When you are done developing and want to test the embedded build:**

```bash
make build                   # Linux/macOS
.\build.ps1                  # Windows
./backend/rtspanda           # then run the single binary
```

---

## 15. Troubleshooting

### Camera shows Offline immediately after adding

- **Check the RTSP URL** — paste it into VLC to verify it works
- **Check camera is on the same network** — RTSPanda must be able to reach the camera's IP
- **Check port 554 is open** — some cameras block RTSP by default; enable it in the camera's settings
- **Check credentials** — a wrong username or password will cause an immediate offline

### Camera shows Connecting for a long time (more than 30 seconds)

- mediamtx is trying to connect but the camera isn't responding
- Try watching the camera view — the stream only opens on demand. If you are only on the dashboard, status is polled less frequently
- Check if mediamtx is running: look for it in Task Manager (Windows) or `ps aux | grep mediamtx` (Linux/macOS)
- Restart RTSPanda

### Video player shows an error

- **"Network error"** — the HLS stream failed mid-playback. Click Retry. If it keeps failing, the camera likely dropped the RTSP connection
- **"HLS not supported"** — very unlikely in modern browsers; try Chrome or Firefox
- **Blank black screen** — the stream may still be starting up; wait 10 seconds and retry

### RTSPanda starts but mediamtx is not found

You will see this in the terminal:

```
streams: WARNING — mediamtx binary not found
streams: streaming disabled
```

Fix: make sure the mediamtx binary is at `mediamtx/mediamtx` (or `mediamtx/mediamtx.exe` on Windows), or set `MEDIAMTX_BIN` to its path.

### Recordings are not being created

- Make sure **Record to disk** is enabled for the camera (Settings → Cameras → Edit)
- Make sure someone has watched the camera at least once — the stream must open before recording begins
- Check that `DATA_DIR` is writable by the process
- Recordings appear after the first hour boundary. If you enabled recording 30 minutes ago, you will not see a file yet — the current segment is still being written

### No detection overlays or history appear

- Confirm tracking is enabled for that camera (Settings → Cameras → Edit)
- In camera view, click **Run Test Detection** and check for a success message
- Check AI worker health: `GET /api/v1/detections/health`
- Verify `ffmpeg` is installed and reachable (`FFMPEG_BIN`)
- If using Docker Compose, ensure both `rtspanda` and `ai-worker` containers are up

### Detector request errors (`connection refused` / `no such host`)

If logs show errors like:

- `Post "http://ai-worker:8090/detect": connect: connection refused`
- `lookup ai-worker ... no such host`

check container health and networking first:

- `docker ps` and confirm both `rtspanda` and `rtspanda-ai-worker` are running
- `docker logs rtspanda-ai-worker` for Python startup/runtime errors
- `docker compose up --build -d` after dependency or Dockerfile changes

RTSPanda now supports detector URL fallback internally, but a stopped or crashing worker will still fail requests.

### Discord alerts are not arriving

- Verify camera-level **Discord Rich Alerts** is enabled
- Confirm webhook URL is valid and starts with `https://`
- Verify trigger mode: **Trigger on YOLO detections** and/or **Trigger on interval screenshots**
- Set cooldown to a lower number while testing
- Use **Run Test Detection** on a scene that reliably triggers detections
- Inspect server logs for webhook delivery errors
- Try **Screenshot to Discord** from camera view to validate webhook delivery independently of detection

### Build fails

**`npm run build` fails:**
- Make sure Node.js 18+ is installed: `node --version`
- Delete `frontend/node_modules` and run `npm install` again
- Check for TypeScript errors in the output

**`go build` fails:**
- Make sure Go 1.26+ is installed: `go version`
- Run `go mod tidy` in the `backend/` directory
- Check the error message — it will point to the file and line

### Port 8080 is already in use

```bash
PORT=8081 ./backend/rtspanda
```

Then open `http://localhost:8081`.

---

## 16. Security

**RTSPanda has no authentication in this release.**

This is intentional for the current version — the design is optimised for homelab and LAN use where the network itself is the security boundary.

**What this means:**

- Anyone who can reach port 8080 can view all cameras, download recordings, and modify settings
- Do not expose RTSPanda directly to the public internet without additional protection

**Recommended approaches for remote access:**

| Method | Difficulty | Notes |
|--------|-----------|-------|
| **VPN** (Tailscale, WireGuard) | Easy | RTSPanda stays on LAN; you connect via VPN |
| **Reverse proxy with auth** (Nginx, Caddy, Authelia) | Medium | Add HTTP basic auth or SSO in front |
| **SSH tunnel** | Easy | `ssh -L 8080:localhost:8080 user@server` |

**RTSP credentials:** your camera's RTSP URL (including username and password) is stored in plain text in the SQLite database at `DATA_DIR/rtspanda.db`. Protect that file's access permissions on multi-user systems.

---

## 17. API quick reference

All endpoints are under `/api/v1`. Responses are JSON.

### Cameras

```http
GET    /api/v1/cameras                    # list all cameras
POST   /api/v1/cameras                    # add camera
GET    /api/v1/cameras/{id}               # get camera
PUT    /api/v1/cameras/{id}               # update camera
DELETE /api/v1/cameras/{id}               # delete camera
GET    /api/v1/cameras/{id}/stream        # get HLS URL + status
```

**Add camera — request body:**
```json
{
  "name": "Front Door",
  "rtsp_url": "rtsp://admin:pass@192.168.1.10:554/stream",
  "enabled": true,
  "record_enabled": false,
  "tracking_enabled": true,
  "detection_sample_seconds": 15,
  "tracking_min_confidence": 0.45,
  "tracking_labels": ["person", "car"],
  "discord_alerts_enabled": true,
  "discord_webhook_url": "https://discord.com/api/webhooks/...",
  "discord_mention": "@here",
  "discord_cooldown_seconds": 90,
  "discord_trigger_on_detection": true,
  "discord_trigger_on_interval": false,
  "discord_screenshot_interval_seconds": 300,
  "discord_include_motion_clip": true,
  "discord_motion_clip_seconds": 4,
  "discord_record_format": "webp",
  "discord_record_duration_seconds": 60,
  "discord_detection_provider": "yolo",
  "frigate_camera_name": ""
}
```

**Camera object (response):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Front Door",
  "rtsp_url": "rtsp://admin:pass@192.168.1.10:554/stream",
  "enabled": true,
  "record_enabled": false,
  "detection_sample_seconds": 15,
  "tracking_enabled": true,
  "tracking_min_confidence": 0.45,
  "tracking_labels": ["person", "car"],
  "discord_alerts_enabled": true,
  "discord_webhook_url": "https://discord.com/api/webhooks/...",
  "discord_mention": "@here",
  "discord_cooldown_seconds": 90,
  "discord_trigger_on_detection": true,
  "discord_trigger_on_interval": false,
  "discord_screenshot_interval_seconds": 300,
  "discord_include_motion_clip": true,
  "discord_motion_clip_seconds": 4,
  "discord_record_format": "webp",
  "discord_record_duration_seconds": 60,
  "discord_detection_provider": "yolo",
  "frigate_camera_name": "",
  "position": 0,
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

**Stream status response:**
```json
{
  "hls_url": "/hls/camera-550e8400.../index.m3u8",
  "status": "online"
}
```

### Detection endpoints

```http
GET    /api/v1/detections/health
POST   /api/v1/cameras/{id}/detections/test-frame
POST   /api/v1/cameras/{id}/detections/test
POST   /api/v1/cameras/{id}/discord/screenshot
POST   /api/v1/cameras/{id}/discord/record
POST   /api/v1/frigate/events
GET    /api/v1/detection-events?limit=100&camera_id={id}
GET    /api/v1/detection-events/{id}/snapshot
```

**Detection event object (response):**
```json
{
  "id": "event-id",
  "camera_id": "camera-id",
  "object_label": "person",
  "confidence": 0.92,
  "bbox": {"x": 123, "y": 45, "width": 210, "height": 420},
  "snapshot_path": "data/snapshots/detections/camera-id/20260313T032200Z.jpg",
  "frame_width": 1920,
  "frame_height": 1080,
  "created_at": "2026-03-13T03:22:00Z"
}
```

### Recordings

```http
GET    /api/v1/cameras/{id}/recordings                   # list recordings
GET    /api/v1/cameras/{id}/recordings/{filename}        # download file
DELETE /api/v1/cameras/{id}/recordings/{filename}        # delete file
```

**List recordings response:**
```json
[
  {
    "filename": "2025-01-15_10-00-00.mp4",
    "camera_id": "550e8400...",
    "size_bytes": 1073741824,
    "created_at": "2025-01-15T11:00:00Z"
  }
]
```

### Legacy alert rules

```http
GET    /api/v1/cameras/{id}/alerts          # list rules for camera
POST   /api/v1/cameras/{id}/alerts          # create rule
PUT    /api/v1/alerts/{id}                  # update rule
DELETE /api/v1/alerts/{id}                  # delete rule
GET    /api/v1/alerts/{id}/events           # list events (last 50)
POST   /api/v1/alerts/{id}/events           # trigger event (webhook)
GET    /api/v1/cameras/{id}/alert-events    # all events for a camera
```

**Create rule — request body:**
```json
{
  "name": "Person at front door",
  "type": "object_detection",
  "enabled": true
}
```

Valid types: `motion`, `connectivity`, `object_detection`

**Trigger event — request body:**
```json
{
  "snapshot_path": "/path/to/frame.jpg",
  "metadata": "{\"label\":\"person\",\"confidence\":0.94}"
}
```

Both fields are optional.

### Health

```http
GET /api/v1/health   →   {"status": "ok"}
```

---

## Appendix: File locations reference

| What | Default location |
|------|-----------------|
| Database | `data/rtspanda.db` |
| mediamtx config (auto-generated) | `data/mediamtx.yml` |
| Recordings | `data/recordings/camera-{id}/` |
| mediamtx binary | `mediamtx/mediamtx` or `mediamtx/mediamtx.exe` |

All paths under `data/` are relative to wherever the binary is run from, or `DATA_DIR` if set.

---

*RTSPanda — self-hosted, no cloud, no fuss.*
