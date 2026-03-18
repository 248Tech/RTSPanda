# RTSPanda — Current Focus

Last updated: 2026-03-18

---

## Current Phase

**Phase: v0.0.6 performance + observability complete. Preparing release.**

v0.0.6 deliverables shipped:

- Multi-view `+` Add Camera card with picker; `✕` remove button per panel
- Stream status path list cache — N cameras = 1 mediamtx call (was N)
- `GET /api/v1/cameras/stream-status` batch endpoint
- Gzip compression on all `/api/` JSON responses
- DB index `008_cameras_index.sql` on `cameras(position, created_at)`
- 256 KB request body limit on camera create/update
- React.lazy code splitting — initial JS bundle 831 kB → 202 kB (76% drop)
- Request logging middleware + atomic metrics counters
- `/metrics` Prometheus-compatible endpoint (stdlib only)
- mediamtx native metrics at port 9998 (via generated config)
- `GET /api/v1/system/stats` — process uptime, RAM (RSS on Linux), goroutines, bandwidth
- `GET /api/v1/health/ready` — DB + mediamtx deep health check
- Settings → System tab with 5-second auto-refresh stats panel and RAM bar

---

## Top Priorities for v0.0.7

1. **WebRTC + HLS fallback** (TASK-EXP-001): biggest Pi performance win — sub-second latency, no HLS buffering. Spec in `AI/AGENTIC-PLATFORM-EXPANSION-GUIDE.md` Phase 2.
2. **Detection event retention + cleanup** (TASK-AI-003): snapshot disk growth is unbounded — must ship before wide adoption.
3. **Detection history pagination** (TASK-AI-004): camera page gets slow on long-running deployments.
4. **Discord delivery retry/backoff** (TASK-AI-006): silent alert loss on transient failures.

---

## Platform Expansion Backlog (from AGENTIC-PLATFORM-EXPANSION-GUIDE.md)

| Task | Feature | Status |
|------|---------|--------|
| TASK-EXP-001 | WebRTC + HLS fallback | Ready for Claude |
| TASK-EXP-002 | ONVIF discovery + PTZ | Ready for Claude |
| TASK-EXP-003 | CEL rules engine + MQTT | Ready for Claude |
| TASK-EXP-004 | Auth proxy integration | Ready for Claude |
| TASK-PERF-002 | React Query + pagination | Ready for Cursor |

See `AI/AGENTIC-PLATFORM-EXPANSION-GUIDE.md` for full specs.
See `AI/AGENTIC-PERFORMANCE-PLAN.md` for remaining performance phases (6–7).

---

## Operational Risks

- Snapshot/event volume still unbounded without TASK-AI-003.
- Discord webhook failures still not retried (TASK-AI-006).
- No auth layer — LAN-first assumption still applies.
- `rss_bytes` is 0 on Windows (dev) — expected. Real values appear on Linux/Pi.
- `sourceOnDemand` in mediamtx template is currently `no` — verify against ADR-006 before next mediamtx-related change.

---

## Out of Scope Right Now

- Custom YOLO training pipelines
- Mobile app
- Multi-instance / HA deployment
