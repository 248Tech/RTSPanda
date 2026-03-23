# RTSPanda Cluster Mode

Cluster mode splits RTSP ingestion from AI inference:
- Raspberry Pi or primary host runs `rtspanda`
- A second machine runs the standalone `ai-worker`

The transport stays simple:
- RTSPanda captures frames locally with ffmpeg
- RTSPanda sends frames to the remote AI worker over HTTP

## Topology
### Pi / primary host
- Runs backend, frontend, mediamtx
- Keeps camera connectivity local
- Can record even when AI is unavailable

### AI host
- Runs the standalone `ai-worker`
- Needs no RTSP camera access
- Needs no mediamtx

## 1. Start the AI worker on the second machine

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker build ai-worker-standalone
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker up -d --no-build ai-worker-standalone
```

Health check:

```bash
curl -s http://127.0.0.1:8090/health
```

If the Pi will connect over the LAN, ensure the AI host firewall allows inbound TCP `8090`.

## 2. Point the Pi at the remote AI worker

On the Pi:

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
chmod +x ./scripts/pi-*.sh
export AI_WORKER_URL="http://192.168.1.50:8090"
./scripts/pi-up.sh
```

Equivalent manual command:

```bash
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile pi build rtspanda-pi
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile pi up -d --no-build rtspanda-pi
```

## 3. Verify the connection

From the Pi:

```bash
curl -s http://127.0.0.1:8080/api/v1/detections/health
```

Expected:
- `ai_mode` is `remote`
- `ai_worker_url` matches the configured remote worker
- `detector_url` resolves to the remote AI worker

## Failure Behavior
- If the remote AI worker is unavailable, RTSPanda continues running.
- Streaming, recording, and UI access remain available.
- Detection health becomes degraded until the remote worker returns.

## Model Provisioning for the AI Host
### Download at build time

```bash
export MODEL_SOURCE=remote
export YOLO_MODEL_NAME=yolo11n
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker build ai-worker-standalone
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker up -d --no-build ai-worker-standalone
```

### Use a local prebuilt model

```bash
cp /path/to/model.onnx ./model.onnx
export MODEL_SOURCE=local
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker build ai-worker-standalone
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker up -d --no-build ai-worker-standalone
```

You can also place the file at `./ai_worker/model/model.onnx`.

## Operational Notes
- Keep the AI host and Pi on a stable LAN.
- Prefer `yolo11n` for CPU-only deployments unless you have measured headroom.
- The default `docker compose up --build -d` workflow is unchanged and still runs the full local stack on a single machine.

---

## Intermediary Pi Pattern (Android Hub + Pi + AI Server)

When RTSPanda runs on an Android device (Termux, no Docker), sustained FFmpeg load can cause thermal throttling. An intermediary Raspberry Pi can absorb the frame-capture workload, leaving Android to handle only RTSP ingest, mediamtx relay, and HLS serving.

### When to use this pattern

Use 3-node when any two of these apply:

| Criterion | Threshold |
|-----------|-----------|
| Camera count | ≥ 4 cameras on Android |
| Resolution | Any camera ≥ 1080p with detection enabled |
| Sample interval | < 15 seconds |
| Sustained Android temp | ≥ 55°C for 10+ minutes at 2-node |

Single-criterion override (always 3-node):
- Android sustained temperature ≥ 65°C at 2-node
- Camera count ≥ 6 on Android

### 3-Node Topology

```
Android (viewer mode, mediamtx relay)
    ├── Ingest camera RTSP
    ├── Re-stream on port 8554: rtsp://<android-ip>:8554/<camera-name>
    └── Serve HLS UI on port 8080

Raspberry Pi (Pi-mode, detection relay)
    ├── mediamtx reads Android re-streams
    ├── FFmpeg frame extraction (offloaded from Android)
    └── AI_WORKER_URL → remote AI server port 8090

AI Server (x86, Docker)
    └── ai-worker standalone
```

### Step 1: Start Android as Hub

On the Android device in Termux:

```bash
export RTSPANDA_MODE=viewer
export DATA_DIR=~/RTSPanda/data
./rtspanda
```

Android mediamtx re-streams are accessible at:

```
rtsp://<android-ip>:8554/<camera-name>
```

The camera name is the URL-safe version of the camera name you added in the Android UI.

### Step 2: Start Pi as Detection Relay

```bash
cd ~/RTSPanda
export RTSPANDA_MODE=pi
export PORT=8081
export DATA_DIR=~/RTSPanda-relay/data
export AI_WORKER_URL=http://<ai-server-ip>:8090
export ANDROID_HUB_IP=192.168.1.100   # Android device IP
./scripts/pi-up.sh
```

Add cameras in the Pi UI using Android re-stream URLs:

```
rtsp://192.168.1.100:8554/camera-living-room
rtsp://192.168.1.100:8554/camera-front-door
```

### Step 3: AI Server

Identical to standard cluster mode — no changes needed.

### Operational Notes for 3-Node

- Android and Pi run separate RTSPanda instances with separate SQLite databases.
- Detection events are stored on the Pi. Android stores camera metadata for UI/viewing.
- Camera configurations must be manually kept in sync between Android and Pi.
- Pi dashboard (`http://<pi-ip>:8081`) shows detection events. Android dashboard (`http://<android-ip>:8080`) shows live streams.

See [Android No-Docker setup guide](./android-no-docker.md) for the full walkthrough.
