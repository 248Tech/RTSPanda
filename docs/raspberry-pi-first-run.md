# RTSPanda Raspberry Pi First Run

This guide is optimized for the first successful run on a Raspberry Pi.

For full operations (security hardening, upgrades, model overrides), see:

- [raspberry-pi-deployment.md](./raspberry-pi-deployment.md)

## OS and Hardware Assumptions

- Raspberry Pi OS Bookworm 64-bit (`aarch64`) recommended.
- Raspberry Pi 4/5 with at least 4 GB RAM recommended.
- 32-bit Pi OS (`armv7`) is not recommended for the AI worker.

## 0) Install base packages

```bash
sudo apt update
sudo apt install -y git curl ca-certificates docker.io docker-compose-plugin
sudo systemctl enable --now docker
```

## 1) Clone and run preflight

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
chmod +x ./scripts/pi-*.sh
./scripts/pi-preflight.sh
```

`pi-preflight.sh` is read-only and reports readiness/caveats.

## 2) One-command first run

```bash
./scripts/pi-up.sh
```

This command runs preflight, validates compose config, builds images, starts containers, and performs health checks.

## 3) Verify

```bash
docker compose ps
curl -s http://127.0.0.1:8080/api/v1/health
curl -s http://127.0.0.1:8080/api/v1/health/ready
curl -s http://127.0.0.1:8080/api/v1/detections/health
```

Expected:

- `/api/v1/health` returns `{"status":"ok"}`
- `/api/v1/health/ready` becomes healthy after startup
- `/api/v1/detections/health` may be briefly degraded during first model load

Open [http://localhost:8080](http://localhost:8080).

## 4) Stop and logs

```bash
docker compose logs -f rtspanda ai-worker
./scripts/pi-down.sh
```

## 5) Notes for first run

- Docker Compose on Pi is supported in this repo state (multi-arch build path enabled).
- `rtspanda` container includes backend + embedded frontend + bundled mediamtx binary.
- Auth defaults to disabled in compose for first-run convenience (`AUTH_ENABLED=false`).
- Set `AUTH_ENABLED=true` and `AUTH_TOKEN=<token>` before running `pi-up.sh` if you need authenticated access immediately.
