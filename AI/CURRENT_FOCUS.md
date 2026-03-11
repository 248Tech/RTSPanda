# RTSPanda — Current Focus

Last updated: 2026-03-09

---

## What We Are Working On Right Now

**Phase: Single-binary done — next: Docker**

TASK-010 (embed frontend into Go binary) is complete. The build copies `frontend/dist` into `backend/internal/api/web`, and the binary embeds and serves the SPA. All non-API, non-`/hls/` routes serve the React app (SPA fallback). Use **`make build`** (Unix/macOS/Git Bash) or **`.\build.ps1`** (Windows) from the repo root to produce the binary.

**Next milestone:** TASK-011 — Dockerfile + docker-compose for production deployment.

---

## Progress at a Glance

| Task | Description | Status |
|------|-------------|--------|
| TASK-001 | Go backend scaffold | ✓ Done |
| TASK-002 | React + Vite + TS scaffold | ✓ Done |
| TASK-003 | SQLite schema + migrations | ✓ Done |
| TASK-004 | Camera REST API (CRUD) | ✓ Done |
| TASK-005 | mediamtx subprocess management | ✓ Done |
| TASK-006 | Stream status API endpoint | ✓ Done |
| TASK-007 | Camera dashboard UI | ✓ Done |
| TASK-008 | hls.js video player + CameraView page | ✓ Done |
| TASK-009 | Settings / camera management UI | ✓ Done |
| TASK-010 | Embed frontend into Go binary | ✓ Done |
| TASK-011 | Dockerfile + docker-compose | ⬜ Ready |

---

## Highest Priority Tasks (In Order)

1. **TASK-011** — Dockerfile + docker-compose — multi-stage build, mediamtx in image, volume for SQLite

---

## Current Blockers

- **mediamtx binary** — For live streams, place the mediamtx binary at `mediamtx/mediamtx` (or `.exe` on Windows) or set `MEDIAMTX_BIN`. The app runs without it (streams show offline).

- **VCS stamping** — If `go build` fails with VCS errors, use `go build -buildvcs=false` or run from a proper git clone.

---

## Do Not Work On Right Now

- Notification system (Phase 2)
- AI image analysis (Phase 2)
- Motion detection (Phase 2)
- WebRTC (Phase 2)
- Authentication (Phase 2)
- Mobile app (Phase 3)

Focus on TASK-011 (Docker).

---

## Definition of Phase 1 Complete

- [x] User can open the app at `http://localhost:8080` (run the built binary)
- [x] User can add an RTSP camera URL via the Settings page
- [x] Camera appears in the dashboard grid with a status indicator
- [x] User can click a camera and watch a live stream via HLS
- [ ] App runs via `docker compose up` from a single compose file
- [ ] SQLite data persists across container restarts
