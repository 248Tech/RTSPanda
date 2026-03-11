# RTSPanda — Project Context

## What Is This

RTSPanda is a lightning-fast, lightweight, self-hosted RTSP camera viewer.

It is not an NVR. It is not a recording system. It is the fastest path from RTSP stream to browser.

Target users: homelab enthusiasts, developers, small businesses, security camera hobbyists.

---

## Core Goals

- View RTSP cameras instantly in a browser
- Minimal CPU/memory usage
- Simple self-hosting via Docker
- No cloud dependencies
- Clean, fast UI
- Open source

---

## What It Is NOT (Phase 1)

- Not a recording system
- Not a motion detection system
- Not a notification system
- Not a cloud product

These are future phases. Do not over-engineer Phase 1 toward them. Build clean extension points but do not implement them yet.

---

## Tech Stack

| Layer       | Technology                          |
|-------------|-------------------------------------|
| Backend     | Go (stdlib + minimal deps)          |
| Frontend    | React + Vite + TypeScript           |
| Database    | SQLite (via `modernc.org/sqlite`)   |
| Streaming   | mediamtx (stream relay sidecar)     |
| Player      | hls.js (Phase 1) / WebRTC (Phase 2) |
| Deployment  | Docker, docker-compose              |

---

## Streaming Model

```
RTSP Camera
    ↓
mediamtx (relay sidecar — handles RTSP → HLS/WebRTC)
    ↓
Go backend (manages mediamtx, serves API + HLS files)
    ↓
Browser (hls.js player — never connects directly to camera)
```

The browser must never connect directly to the RTSP camera. All streams are relayed.

---

## Constraints

- Go backend must be a single binary
- Frontend must be embedded into the Go binary for production
- Docker image must be small (target: < 150MB)
- SQLite only — no Postgres/MySQL for Phase 1
- mediamtx is allowed as a bundled dependency (it is a single binary)
- FFmpeg is optional and discouraged unless absolutely necessary
- No authentication in Phase 1 (LAN-first product; auth is Phase 2)

---

## Coding Standards

### Go
- Use standard library where possible
- Avoid unnecessary abstraction
- Errors returned, not panicked
- Flat package structure preferred: `internal/cameras`, `internal/streams`, `internal/db`
- SQL via raw queries with `database/sql` (no ORM)

### TypeScript / React
- Functional components only
- No class components
- Use `fetch` or a thin wrapper — no Axios
- Tailwind CSS for styling
- No Redux — use React context or Zustand if state management needed

### General
- Do not add features that are not in the current TODO
- Do not add logging frameworks before core features work
- Keep dependencies minimal and justified

---

## Project Status

**Phase: Initial Setup / Architecture Planning**

Nothing is implemented yet. Planning phase is active.

---

## Future Features (Planned, Not Now)

- Discord / Email notifications
- Snapshot storage
- Clip export
- Motion detection
- AI image interpretation (async only — must never affect live view)
- Android mobile app
- Event pipeline

---

## Repository Structure (Target)

```
RTSPanda/
├── AI/                        ← AI coordination files (this folder)
├── backend/
│   ├── cmd/rtspanda/          ← main entrypoint
│   ├── internal/
│   │   ├── cameras/           ← camera CRUD + model
│   │   ├── streams/           ← stream lifecycle manager
│   │   ├── db/                ← SQLite setup + migrations
│   │   └── api/               ← HTTP handlers
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   └── api/               ← typed API client
│   ├── vite.config.ts
│   └── package.json
├── mediamtx/                  ← mediamtx config template
├── Dockerfile
├── docker-compose.yml
├── install.sh
└── README.md
```
