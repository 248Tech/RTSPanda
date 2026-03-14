# RTSPanda — Current Focus

Last updated: 2026-03-14

---

## Current Phase

**Phase: v0.0.3 release hardening complete, v0.0.4 reliability backlog next.**

v0.0.3 shipped:

- YOLO detector reliability fixes in Docker (`ai-worker` runtime deps + detector URL fallback attempts)
- FFmpeg RTSP timeout option compatibility fallback (`rw_timeout` -> `timeout` -> none)
- Verbose detector/YOLO logging across backend workers and AI worker
- Per-camera Discord trigger controls (detection + interval screenshot modes)
- Manual camera actions: `Screenshot to Discord` and `Record to Discord`
- Motion clip/media format handling (`webm`, `webp`, `gif`)
- UI wording updates to YOLO-first alerting with legacy rule compatibility

---

## Top Priorities

1. Add retention/cleanup policy for `detection_events` and `data/snapshots/detections`.
2. Add detection history pagination/filtering in API and camera view.
3. Add Discord delivery retry/backoff with clear failure visibility.
4. Add integration tests for migration `005` and Discord media/trigger paths.

---

## Operational Risks

- Snapshot/event volume can grow unbounded without retention.
- Discord webhook failures are logged but not retried yet.
- No auth layer yet (LAN-first assumption still required).

---

## Out of Scope Right Now

- WebRTC path migration
- Auth/SSO implementation
- Mobile app work
- Custom YOLO training pipelines
