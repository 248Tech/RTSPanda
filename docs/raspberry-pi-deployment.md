# RTSPanda Raspberry Pi Deployment

Use this document as the operational companion to [raspberry-pi.md](./raspberry-pi.md).

## Recommended Path
For the most reliable first run on Raspberry Pi:

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
chmod +x ./scripts/pi-*.sh
./scripts/pi-up.sh
```

This starts the lightweight Pi-safe deployment:
- `rtspanda-pi`
- backend + frontend + mediamtx
- remote-AI ready

## Alternate Paths
### Full local stack on Pi

```bash
PI_DEPLOYMENT_MODE=full ./scripts/pi-up.sh
```

### Standalone AI worker on another machine

```bash
PI_DEPLOYMENT_MODE=ai-worker ./scripts/pi-up.sh
```

Or directly:

```bash
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker build ai-worker-standalone
docker compose -f docker-compose.yml -f docker-compose.standalone.yml --profile ai-worker up -d --no-build ai-worker-standalone
```

## Upgrades

```bash
git pull
./scripts/pi-up.sh
```

For cluster mode, see [cluster-mode.md](./cluster-mode.md).
