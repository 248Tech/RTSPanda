# RTSPanda on Raspberry Pi

This guide covers the Pi-first deployment paths added for RTSPanda without changing the default x86/GPU workflow.

## Supported Modes
### 1. Lightweight Pi mode

Runs:
- `rtspanda-pi` only
- Go backend
- Embedded frontend
- mediamtx

Does not run:
- local AI worker
- model download/export during Pi startup

Start it:

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
chmod +x ./scripts/pi-*.sh
./scripts/pi-up.sh
```

This is equivalent to:

```bash
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile pi build rtspanda-pi
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile pi up -d --no-build rtspanda-pi
```

If `AI_WORKER_URL` is not set, the Pi stays usable for RTSP ingestion, playback, and recording while detection health remains degraded.

### 2. Pi with remote AI worker

Point the Pi at a second machine running the standalone AI worker:

```bash
export AI_WORKER_URL="http://192.168.1.50:8090"
./scripts/pi-up.sh
```

The Pi continues to:
- ingest RTSP streams
- capture frames with ffmpeg
- forward frames over HTTP to the remote AI worker

### 3. Full local stack on Pi (not recommended)

Running the full local YOLO AI worker on a Pi is not recommended and is not a supported configuration. Raspberry Pi does not support real-time YOLO inference (see DEC-018):

- YOLOv8n ONNX requires ~400–600 MB RAM at runtime, consuming nearly all headroom on a 4 GB Pi running the full stack.
- ONNX Runtime on arm64 runs at 3–8 FPS on Pi 4 CPU — not viable for real-time alerting.
- Thermal throttling degrades performance under sustained load.

**Use remote AI worker (Mode 2 above) instead.** If you have a second machine available on the LAN, the Pi + remote worker split is the correct architecture for Pi deployments.

For snap-based AI analysis on Pi without a remote worker, use Snapshot AI (Mode 1 with `SNAPSHOT_AI_ENABLED=true`).

## Model Options
The AI worker is ONNX-only in Docker.

### Remote model download

Default behavior:

```bash
export MODEL_SOURCE=remote
export YOLO_MODEL_NAME=yolo11n
export YOLO_MODEL_RELEASE=v8.3.0
```

Optional explicit URL:

```bash
export YOLO_MODEL_URL="https://your-mirror.example/yolo11n.onnx"
```

### Local model baked into the image

Place a prebuilt model at either location before building:

```text
./model.onnx
./ai_worker/model/model.onnx
```

Then set:

```bash
export MODEL_SOURCE=local
PI_DEPLOYMENT_MODE=full ./scripts/pi-up.sh
```

### Local model mounted at runtime

For native or custom Docker runs, mount a model at:

```text
/model/model.onnx
```

## Health Checks
Backend:

```bash
curl -s http://127.0.0.1:8080/api/v1/health
curl -s http://127.0.0.1:8080/api/v1/health/ready
curl -s http://127.0.0.1:8080/api/v1/detections/health
```

The detection health payload now includes:
- `ai_mode`
- `ai_worker_url`
- `detector_url`

## Performance Expectations
- Raspberry Pi 4/5 works best with the lightweight Pi mode.
- Remote AI mode gives the cleanest first-run result on Pi because the worker is not built or run locally.
- Full local-AI mode is viable on 64-bit Pi OS with enough RAM, but inference throughput is still CPU-limited.

## Limitations
- 32-bit Pi OS is not recommended for the AI worker.
- The Docker AI worker will not export models or install PyTorch as a fallback.
- In lightweight mode, detection remains degraded until `AI_WORKER_URL` or `DETECTOR_URL` is configured.

## Related Docs
- [Cluster Mode](./cluster-mode.md)
- [Raspberry Pi First Run](./raspberry-pi-first-run.md)
- [Raspberry Pi Deployment](./raspberry-pi-deployment.md)
