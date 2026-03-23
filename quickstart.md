# RTSPanda Quickstart

This file is a short, practical guide based on the current repository layout and scripts.

## 1) Repository map (what matters first)

| Path | Purpose | Key files |
|---|---|---|
| `backend/` | Go API server, stream orchestration, DB migrations, embedded web UI host | `cmd/rtspanda/main.go`, `internal/api/router.go`, `go.mod` |
| `frontend/` | React 19 + Vite UI | `src/`, `package.json`, `vite.config.ts` |
| `ai_worker/` | Python FastAPI + ONNX Runtime detector service (YOLO) | `app/main.py`, `requirements.txt`, `Dockerfile` |
| `scripts/` | Helper scripts for Docker, Pi, and Android (Termux) | `install-windows.ps1`, `docker-up.ps1`, `pi-up.sh`, `android-up.sh` |
| `data/` | Runtime state (SQLite DB, recordings, snapshots, mediamtx config) | `rtspanda.db`, `recordings/`, `snapshots/` |
| `mediamtx/` | Place `mediamtx` binary here for local non-Docker streaming | `mediamtx.yml.tmpl` (template in repo) |
| `docs/`, `human/` | Product/setup docs and user guide | `docs/RTSP_SETUP.md`, `human/USER_GUIDE.md` |
| `docker-compose.yml`, `Dockerfile` | Containerized deployment path | starts `rtspanda` + `ai-worker` |
| `build.ps1`, `Makefile` | Local build pipeline (frontend build -> embed -> Go build) | produces `backend/rtspanda(.exe)` |

Notes:
- `backend/internal/api/web/` is build output (frontend files copied for embedding).
- `frontend/node_modules/` and `frontend/dist/` are local build artifacts.

## 2) Fastest way to run (recommended)

### Docker Compose

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

## 3) Android no-Docker quick start (Termux + remote YOLO)

Use this path when running RTSPanda natively on Android (no Docker, no root).

On AI server (Docker host):

```bash
docker compose -f docker-compose.yml -f docker-compose.standalone.yml \
  --profile ai-worker up -d --no-build ai-worker-standalone
curl -s http://127.0.0.1:8090/health
```

On Android (Termux):

```bash
pkg update && pkg upgrade -y
pkg install -y golang ffmpeg wget
git clone https://github.com/248Tech/RTSPanda.git ~/RTSPanda
cd ~/RTSPanda/backend && go build -o ../rtspanda ./cmd/rtspanda

# Put mediamtx binary at ~/RTSPanda/mediamtx/mediamtx (ARM64 Linux build)
mkdir -p ~/RTSPanda/mediamtx

cd ~/RTSPanda
export AI_WORKER_URL=http://<ai-server-ip>:8090
./scripts/android-up.sh
```

Validate:

```bash
curl -s http://127.0.0.1:8080/api/v1/health
curl -s http://127.0.0.1:8080/api/v1/health/ready
curl -s http://127.0.0.1:8080/api/v1/detections/health
```

## 4) Local native run (no Docker)

Prereqs:
- Go 1.26+
- Node.js 18+ (repo currently uses Vite 7 / React 19)
- Optional but needed for real playback: `mediamtx` binary in `mediamtx/`

### Windows

```powershell
cd frontend
npm install
cd ..
.\build.ps1
.\backend\rtspanda.exe
```

### macOS / Linux

```bash
cd frontend && npm install && cd ..
make build
./backend/rtspanda
```

Open: <http://localhost:8080>

## 5) Development mode (hot reload UI)

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

## 6) Quick health check

```powershell
.\scripts\api-smoke.ps1
```

This verifies health + basic camera CRUD endpoints against the running server.

## 7) Common gotchas

- If video stays offline locally, check `mediamtx` binary exists (`mediamtx/mediamtx.exe` on Windows or `mediamtx/mediamtx` on macOS/Linux).
- If backend serves "Frontend not built", rebuild so assets are copied into `backend/internal/api/web/`.
- Docker mode already wires detector service (`ai-worker`) through `DETECTOR_URL=http://ai-worker:8090`.
- Android mode uses `scripts/android-up.sh` (sets `RTSPANDA_MODE=pi`, configures `DATA_DIR`, and runs `termux-wake-lock` if available).
