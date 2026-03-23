# RTSPanda on Android (Termux, No Docker)

This guide covers running RTSPanda natively on an Android device using Termux — without Docker and without root. Detection can be provided by a remote AI server on your LAN.

RTSPanda uses Pi-mode behavior on Android: the Go binary runs mediamtx as a subprocess, captures frames with FFmpeg, and forwards detections to a remote YOLO worker on your LAN. Snapshot AI (Claude/OpenAI) also works.

---

## Before You Start

**Android requirements:**
- Android 9+ (arm64 strongly recommended; armv7 is untested)
- Termux (install from F-Droid — Play Store version is outdated)
- Device connected to your LAN via Wi-Fi (not mobile data)
- Wi-Fi sleep disabled (see step 3)
- Battery optimization disabled for Termux (see step 3)

**Network requirements:**
- Stable LAN access to your RTSP cameras
- If using remote YOLO: your AI server reachable on LAN port 8090

**Not supported on Android:**
- Docker (Termux cannot run Docker containers without root)
- Local YOLO inference (thermal and memory constraints — same restriction as Raspberry Pi)
- Local vision models (4–8 GB RAM requirement; not viable on mobile hardware)

---

## Topology Options

### Option A: 2-Node (Android + Remote AI Server)

Recommended for: 1–3 cameras, mild ambient conditions, sample interval ≥ 15 s.

```
Android (RTSPanda Pi-mode, Termux) ──frames──► AI Server (ai-worker Docker, LAN)
       ▲           ▲           ▲
   Camera 1    Camera 2    Camera 3
```

Android handles: RTSP ingest, HLS serving, UI, FFmpeg frame capture, frame forwarding.

### Option B: 3-Node (Android Hub + Intermediary Pi + Remote AI Server)

Recommended for: 4+ cameras, 1080p+ streams, or if Android shows sustained heat (> 55°C for 10+ minutes at 2-node).

```
Android (viewer/stream hub) ── RTSP re-stream ──► Raspberry Pi (detection relay)
       ▲           ▲                                      │ HTTP frames
   Camera 1    Camera 2-N                                 ▼
                                               AI Server (ai-worker Docker)
```

Android handles: RTSP ingest, mediamtx relay, HLS serving, UI only.
Pi handles: frame capture (FFmpeg), detection forwarding — keeping Android cool.

See the [3-Node Setup](#3-node-setup-android--pi--ai-server) section below.

---

## 2-Node Setup (Android + AI Server)

### Step 1: Prepare the AI Server

On your AI server (x86 Linux machine, Docker required):

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
docker compose -f docker-compose.yml -f docker-compose.standalone.yml \
  --profile ai-worker build ai-worker-standalone
docker compose -f docker-compose.yml -f docker-compose.standalone.yml \
  --profile ai-worker up -d --no-build ai-worker-standalone
```

Verify:

```bash
curl -s http://127.0.0.1:8090/health
# Expected: {"status":"ok","model_loaded":true}
```

Allow inbound port 8090 from your Android device on the server firewall.

### Step 2: Install Termux Dependencies

In Termux on your Android device:

```bash
pkg update && pkg upgrade -y
pkg install -y golang ffmpeg wget
```

Verify:

```bash
go version      # go1.21+ expected
ffmpeg -version # any recent build
```

### Step 3: Android Power and Connectivity

**Disable Wi-Fi sleep:** Settings → Wi-Fi → Advanced → Keep Wi-Fi on during sleep → Always.

**Disable battery optimization for Termux:** Settings → Apps → Termux → Battery → Unrestricted (or "Don't optimize").

**Keep screen on (optional but helps during first run):** Use a screen lock timeout of Never or use `termux-wake-lock` in your startup script.

### Step 4: Download RTSPanda and mediamtx

```bash
# Clone the repo
git clone https://github.com/248Tech/RTSPanda.git ~/RTSPanda
cd ~/RTSPanda/backend

# Build the binary (native Go compile — no cross-compilation needed in Termux)
go build -o ~/RTSPanda/rtspanda ./cmd/rtspanda

# Download mediamtx ARM64 binary (adjust version as needed)
mkdir -p ~/RTSPanda/mediamtx
wget -O /tmp/mediamtx.tar.gz \
  https://github.com/bluenviron/mediamtx/releases/download/v1.12.3/mediamtx_v1.12.3_linux_arm64v8.tar.gz
tar -xzf /tmp/mediamtx.tar.gz -C ~/RTSPanda/mediamtx mediamtx
chmod +x ~/RTSPanda/mediamtx/mediamtx
```

### Step 5: Configure and Start

```bash
cd ~/RTSPanda

# Required
export RTSPANDA_MODE=pi
export DATA_DIR=~/RTSPanda/data
export AI_WORKER_URL=http://192.168.1.50:8090   # your AI server IP

# Optional — Snapshot AI (Claude or OpenAI) in addition to remote YOLO
# export SNAPSHOT_AI_ENABLED=true
# export SNAPSHOT_AI_PROVIDER=claude
# export SNAPSHOT_AI_API_KEY=sk-ant-...
# export SNAPSHOT_AI_INTERVAL_SECONDS=60

# Start (use tmux or screen to keep alive)
./scripts/android-up.sh
```

Or run directly without the script:

```bash
./rtspanda
```

### Step 6: Verify

From any browser on your LAN:

```
http://<android-ip>:8080
```

Health checks:

```bash
curl -s http://127.0.0.1:8080/api/v1/health
curl -s http://127.0.0.1:8080/api/v1/health/ready
curl -s http://127.0.0.1:8080/api/v1/detections/health
# ai_mode should be "remote"
```

---

## 3-Node Setup (Android + Pi + AI Server)

Use this when any two of the following apply:

- 4+ cameras on Android
- Any camera at 1080p+ with detection enabled
- Detection sample interval < 15 seconds
- Sustained Android temperature ≥ 55°C for 10+ minutes at 2-node
- A Raspberry Pi is available on the same LAN

### 3-Node: Android (Hub)

Android runs in viewer mode — only RTSP ingest, mediamtx relay, and HLS serving. No FFmpeg detection work.

```bash
cd ~/RTSPanda

export RTSPANDA_MODE=viewer
export DATA_DIR=~/RTSPanda/data
export PORT=8080

./rtspanda
```

Android's mediamtx will expose RTSP re-streams on port 8554:

```
rtsp://<android-ip>:8554/<camera-name>
```

Add your cameras in the Android UI as usual. The Pi will read from these re-streams.

### 3-Node: Intermediary Raspberry Pi (Detection Relay)

On the Pi, use RTSPanda in Pi-mode. Configure camera RTSP URLs to point to Android's mediamtx re-streams instead of the physical cameras directly.

```bash
cd ~/RTSPanda
chmod +x ./scripts/pi-*.sh

export RTSPANDA_MODE=pi
export PORT=8081                              # avoid conflict if Pi also serves UI
export DATA_DIR=~/RTSPanda-relay/data
export AI_WORKER_URL=http://192.168.1.50:8090 # AI server

./scripts/pi-up.sh
```

Then in the Pi UI (`http://<pi-ip>:8081`), add cameras using the Android re-stream URLs:

```
rtsp://192.168.1.100:8554/camera-living-room
rtsp://192.168.1.100:8554/camera-front-door
```

The Pi handles all FFmpeg frame extraction and forwards detections to the server. Detection events appear in the Pi dashboard.

### 3-Node: AI Server

Identical to 2-node. No changes needed.

---

## Thermal Operation

RTSPanda includes a thermal monitor for Android Pi-mode deployments. It reads temperature from `/sys/class/thermal/thermal_zone*/temp` (available without root in most Android kernels). If thermal data is unreadable, it falls back to CPU load as a proxy.

### Temperature Bands

| Band | Temperature | Current Behavior |
|------|-------------|-------------|
| Normal | < 45°C | Normal operation |
| Warm | 45–54°C | Warm state reported |
| Hot | 55–64°C | Hot state reported + Discord system alert on entry |
| Critical | ≥ 65°C | Critical state reported |

### Recovery

Band downshifts require sustained cool-down (hysteresis) to prevent flapping:

- Critical → Hot: 5 minutes
- Hot → Warm: 5 minutes
- Warm → Normal: 3 minutes

`THERMAL_AUTO_RESUME=false` remains the default. This flag is reserved for future thermal-aware detection throttling behavior.

### Operator Signals

```bash
# Check current thermal band
curl -s http://127.0.0.1:8080/api/v1/system/stats | grep thermal_band

# Logs
# Look for WARN/ERROR/CRITICAL thermal transition logs
```

A Discord system alert is sent on Hot-band entry if any camera has a Discord webhook configured.

---

## Configuration Reference (Android-specific)

| Variable | Default | Notes |
|---------|---------|------|
| `RTSPANDA_MODE` | auto (pi on ARM) | Set `pi` explicitly for detection; `viewer` for 3-node hub |
| `AI_WORKER_URL` | empty | Set to your remote YOLO server `http://<ip>:8090` |
| `DATA_DIR` | `./data` | Use absolute path under Termux home, e.g. `~/RTSPanda/data` |
| `FFMPEG_BIN` | `ffmpeg` (PATH) | Override if Termux FFmpeg is in a non-standard location |
| `MEDIAMTX_BIN` | `mediamtx/mediamtx` | Path to mediamtx ARM64 binary |
| `PORT` | `8080` | HTTP port for RTSPanda UI and API |
| `THERMAL_MONITOR_ENABLED` | `false` (auto-on arm64+pi) | Set `true` to force-enable on other modes/platforms |
| `THERMAL_AUTO_RESUME` | `false` | Reserved for future thermal auto-resume behavior |
| `SNAPSHOT_AI_ENABLED` | `false` | Enable Claude/OpenAI snapshot AI alongside remote YOLO |

---

## Keeping RTSPanda Running (Session Management)

Termux has no systemd. Use one of:

### tmux (recommended)

```bash
pkg install tmux
tmux new-session -d -s rtspanda 'cd ~/RTSPanda && ./scripts/android-up.sh'
```

Reattach:

```bash
tmux attach -t rtspanda
```

### termux-services

```bash
pkg install termux-services
# Create a run script
mkdir -p $PREFIX/var/service/rtspanda/log
cat > $PREFIX/var/service/rtspanda/run << 'EOF'
#!/data/data/com.termux/files/usr/bin/sh
exec ~/RTSPanda/scripts/android-up.sh 2>&1
EOF
chmod +x $PREFIX/var/service/rtspanda/run
sv-enable rtspanda
```

---

## Troubleshooting

### Stream stuck in `initializing`

- Verify RTSP URL with: `ffplay rtsp://camera-ip/stream`
- Check if mediamtx binary is executable: `ls -la ~/RTSPanda/mediamtx/mediamtx`
- Review RTSPanda logs for mediamtx errors

### Detection not firing

- Confirm remote ai-worker is reachable: `curl -s http://<server>:8090/health`
- Check `AI_WORKER_URL` is set: `echo $AI_WORKER_URL`
- Confirm server firewall allows port 8090 inbound

### High temperature / detection suspended

- Check thermal band: `curl -s http://127.0.0.1:8080/api/v1/system/stats`
- Increase sample interval to reduce FFmpeg load
- Consider 3-node topology if sustained ≥ 55°C

### RTSPanda killed by Android

- Disable battery optimization for Termux (see Step 3)
- Use `termux-wake-lock` at start of your run script: add `termux-wake-lock` as first line

### Port 8080 in use

- Change `PORT=8090` or another free port
- `ss -tulnp | grep 8080` to identify what is using the port

### Can't connect from browser on LAN

- Android firewall: no action needed (Android does not block inbound by default)
- Confirm Android and browser device are on the same Wi-Fi network (not guest VLAN isolation)
- Confirm `PORT` matches what you're browsing to

---

## Rollback

To return to a previous version:

```bash
cd ~/RTSPanda
git fetch
git checkout v0.1.0   # or whichever version to roll back to
cd backend && go build -o ../rtspanda ./cmd/rtspanda
```

Data (`DATA_DIR`) is not touched by a rollback. SQLite migrations are forward-only — ensure the older version is compatible with the current migration level before rolling back.

---

## Related Docs

- [Cluster Mode (Pi + remote AI)](./cluster-mode.md)
- [Raspberry Pi setup](./raspberry-pi.md)
- [Streaming tuning](./streaming-tuning.md)
