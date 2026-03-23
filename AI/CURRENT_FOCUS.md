# RTSPanda — Current Focus

Last updated: 2026-03-22

---

## Current Phase

**Phase: Android No-Docker + Remote YOLO + Thermal Policy**

This is the active sprint. Planning is complete. Implementation is ready for Aider.

### Initiative: Android No-Docker Pi-Mode + Remote YOLO

**Spec:** `AI/FEATURES/ANDROID_NO_DOCKER_REMOTE_YOLO.md`
**Decisions:** DEC-021, DEC-022, DEC-023 (added 2026-03-22)

Goal: Make Android (Termux, no Docker) an officially supported deployment platform with Pi-mode behavior, remote YOLO forwarding, an explicit thermal policy, and an optional 3-node topology with an intermediary Raspberry Pi.

**Four implementation phases (all ready for Aider):**

| Phase | Name | Description | Status |
|-------|------|-------------|--------|
| A | Android binary + startup script | `android-up.sh`, Termux build guide | Ready for Aider |
| B | Thermal monitor goroutine | `backend/internal/thermal/`, band detection, hysteresis | Ready for Aider |
| C | Detection throttle integration | Wire thermal bands into detection manager | Ready for Aider (depends on B) |
| D | 3-node operator tooling | `android-3node-hub.sh`, `pi-detection-relay.sh` | Ready for Aider |

---

## Previously Completed

**Phase: v0.1.0 Streaming Readiness Hardening** (completed 2026-03-20)
- Three deployment modes: `pi`, `standard`, `viewer`
- Snapshot Intelligence Engine (Pi mode)
- Stream orchestration hardening + readiness gating

**Phase: v0.0.6 Performance + Observability** (completed 2026-03-18)
- Multi-view, stream status cache, Prometheus metrics, system stats UI

---

## Top Priorities After Android Initiative

1. **TASK-AI-003** — Detection event retention + cleanup (unbounded disk growth)
2. **TASK-AI-006** — Discord delivery resilience (silent alert loss)
3. **TASK-EXP-001** — WebRTC + HLS fallback (sub-second latency for Pi/Android)
4. **TASK-AI-004** — Event filters and pagination

---

## Platform Expansion Backlog

| Task | Feature | Status |
|------|---------|--------|
| TASK-AND-A | Android binary + startup script | Ready for Aider |
| TASK-AND-B | Thermal monitor goroutine | Ready for Aider |
| TASK-AND-C | Detection throttle integration | Ready for Aider |
| TASK-AND-D | 3-node operator tooling | Ready for Aider |
| TASK-EXP-001 | WebRTC + HLS fallback | Ready for Claude |
| TASK-EXP-002 | ONVIF discovery + PTZ | Ready for Claude |
| TASK-EXP-003 | CEL rules engine + MQTT | Ready for Claude |
| TASK-EXP-004 | Auth proxy integration | Ready for Claude |
| TASK-PERF-002 | React Query + pagination | Ready for Cursor |

---

## Operational Risks

- Snapshot/event volume still unbounded without TASK-AI-003.
- Discord webhook failures still not retried (TASK-AI-006).
- No auth layer — LAN-first assumption still applies.
- `rss_bytes` is 0 on Windows (dev) — expected. Real values appear on Linux/Pi/Android.
- `sourceOnDemand` in mediamtx template — verify against ADR-006 before next mediamtx-related change.

---

## Out of Scope Right Now

- Custom YOLO training pipelines
- Mobile app (RTSPanda UI is browser-based; Android hosts the server, not an app)
- Multi-instance / HA deployment
- Config sync between Android and Pi nodes (deferred from Android initiative)
