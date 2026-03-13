# RTSPanda — Current Focus

Last updated: 2026-03-13

---

## What We Are Working On Right Now

**Phase: AI detection UX + notifications shipped; hardening and scaling next.**

Latest completed milestone:

- Per-camera YOLOv8 tracking settings
- Live bounding-box overlays in camera view
- Detection event/history panel
- Discord rich-media alert delivery with per-camera cooldown

Current focus is now reliability and operational polish.

---

## Progress at a Glance

| Task | Description | Status |
|------|-------------|--------|
| TASK-001 | Go backend scaffold | Done |
| TASK-002 | React + Vite + TS scaffold | Done |
| TASK-003 | SQLite schema + migrations | Done |
| TASK-004 | Camera REST API (CRUD) | Done |
| TASK-005 | mediamtx subprocess management | Done |
| TASK-006 | Stream status API endpoint | Done |
| TASK-007 | Camera dashboard UI | Done |
| TASK-008 | hls.js player + camera view | Done |
| TASK-009 | Settings / camera management UI | Done |
| TASK-010 | Embed frontend into Go binary | Done |
| TASK-011 | Dockerfile + docker-compose | Done |
| TASK-012 | Detection foundation (async + worker + events) | Done |
| TASK-013 | YOLOv8 UI + per-camera tracking + Discord alerts | Done |

---

## Highest Priority Tasks (In Order)

1. **Retention policy** for `detection_events` and snapshot files.
2. **Detection history pagination/filtering** (API + UI).
3. **Alert delivery resilience** (retry/backoff + failure visibility).
4. **Automated integration coverage** for migration `004` and notifier path.

---

## Current Blockers

- None blocking feature delivery.
- Primary risk is operational growth (event/snapshot volume) without retention.

---

## Do Not Work On Right Now

- WebRTC stream path
- Auth/SSO layer
- Mobile app
- Custom model training workflows

Keep effort on reliability and scale-hardening of shipped AI functionality.
