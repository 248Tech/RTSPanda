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
docker compose --profile ai-worker up --build -d ai-worker-standalone
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
docker compose --profile pi up --build -d rtspanda-pi
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
export YOLO_MODEL_NAME=yolov8n
docker compose --profile ai-worker up --build -d ai-worker-standalone
```

### Use a local prebuilt model

```bash
cp /path/to/model.onnx ./model.onnx
export MODEL_SOURCE=local
docker compose --profile ai-worker up --build -d ai-worker-standalone
```

You can also place the file at `./ai_worker/model/model.onnx`.

## Operational Notes
- Keep the AI host and Pi on a stable LAN.
- Prefer `yolov8n` for CPU-only deployments unless you have measured headroom.
- The default `docker compose up --build -d` workflow is unchanged and still runs the full local stack on a single machine.
