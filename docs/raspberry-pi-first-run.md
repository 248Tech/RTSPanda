# RTSPanda Raspberry Pi First Run

This is the shortest path to a successful Pi deployment.

## Lightweight Pi mode

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
chmod +x ./scripts/pi-*.sh
./scripts/pi-preflight.sh
./scripts/pi-up.sh
```

This starts:
- backend
- embedded frontend
- mediamtx

If `AI_WORKER_URL` is unset, detection stays degraded until a remote AI worker is added.

## Verify

```bash
docker compose ps
curl -s http://127.0.0.1:8080/api/v1/health
curl -s http://127.0.0.1:8080/api/v1/health/ready
curl -s http://127.0.0.1:8080/api/v1/detections/health
```

## If you want local AI on the Pi

```bash
PI_DEPLOYMENT_MODE=full ./scripts/pi-up.sh
```

## If you want remote AI on another machine

```bash
export AI_WORKER_URL="http://192.168.1.50:8090"
./scripts/pi-up.sh
```

For the full guide, see [raspberry-pi.md](./raspberry-pi.md).
