# RTSPanda - Agentic Platform Expansion Guide (Claude)

Status: Ready for Claude execution  
Last updated: 2026-03-18

---

## Goal

Implement these capabilities with minimal ambiguity and safe rollout:

1. WebRTC live mode with automatic HLS fallback
2. ONVIF discovery and PTZ controls
3. Rules engine using CEL with MQTT output
4. Auth proxy integration
5. Prometheus observability

This guide is written for Claude as the planning + implementation coordinator.

---

## Execution Protocol (Claude)

1. Execute one phase at a time in the order below.
2. Keep each phase in a separate branch/PR.
3. Do not mix schema changes from multiple phases in one migration file.
4. After each phase:
   - Update `AI/TODO.md` status.
   - Add notes and risks to `AI/HANDOFF.md`.
   - Add/adjust architecture decisions in `AI/DECISIONS.md` if behavior changes.
5. Every phase must pass:
   - `cd backend && go test ./...`
   - `cd frontend && npm run build`

---

## Phase Order and Dependencies

| Phase | Feature | Depends on |
|------|---------|------------|
| 1 | Prometheus observability | none |
| 2 | WebRTC + HLS fallback | 1 (recommended), none (hard) |
| 3 | ONVIF discovery + PTZ | 1 |
| 4 | Rules engine (CEL) + MQTT | 1, 3 (camera metadata helps), 2 optional |
| 5 | Auth proxy integration | 1 |

Rationale: observability first makes every later phase easier to debug.

---

## Phase 1 - Prometheus Observability

### Scope

- Export backend metrics at `/metrics`.
- Enable mediamtx metrics and expose them in a scrape-friendly way.
- Add request, stream, detection, and notification metrics.

### Backend Tasks

1. Add Prometheus library:
   - `backend/go.mod`
   - Use `github.com/prometheus/client_golang/prometheus` + `promhttp`.
2. Add metrics package:
   - New `backend/internal/metrics/metrics.go`
   - Counters/histograms/gauges:
     - `http_requests_total{method,route,status}`
     - `http_request_duration_seconds{method,route}`
     - `stream_status_total{status}`
     - `detection_jobs_total{result}`
     - `detection_queue_depth`
     - `discord_webhooks_total{result,source}`
3. Add HTTP middleware for request metrics:
   - `backend/internal/api/router.go` (or new middleware file in `internal/api`)
4. Register `/metrics` route:
   - `backend/internal/api/router.go`
5. Instrument key flow points:
   - `backend/internal/streams/health.go`
   - `backend/internal/detections/manager.go`
   - `backend/internal/notifications/discord.go`
6. Enable mediamtx metrics in generated config:
   - `backend/internal/streams/mediamtx.go`
   - Set:
     - `metrics: yes`
     - `metricsAddress: 127.0.0.1:9998`
7. Add optional proxy endpoint for mediamtx metrics if needed by deployment:
   - `GET /api/v1/metrics/mediamtx` proxied to `127.0.0.1:9998/metrics`

### Acceptance Criteria

- `/metrics` exposes valid Prometheus text format.
- Request counters and durations change with API traffic.
- Detection and Discord metrics increment in expected code paths.
- mediamtx metrics are reachable (directly or via proxy endpoint).

### Verification

- `curl http://localhost:8080/metrics`
- Trigger stream and detection activity, confirm counters move.
- If proxy endpoint is added: `curl http://localhost:8080/api/v1/metrics/mediamtx`

---

## Phase 2 - WebRTC Live Mode with HLS Fallback

### Scope

- Add WebRTC stream URLs for each camera.
- Use mediamtx WHEP endpoint for low-latency playback.
- Automatically fall back to HLS if WebRTC negotiation fails.

Reference behavior from MediaMTX docs:
- WebRTC URL: `http://<host>:8889/<path>`
- WHEP URL: `http://<host>:8889/<path>/whep`

### API Contract Changes

Update stream response to include both protocols:

```json
{
  "status": "online",
  "hls_url": "/hls/camera-<id>/index.m3u8",
  "webrtc_url": "/webrtc/camera-<id>/whep",
  "preferred_protocol": "webrtc"
}
```

`preferred_protocol` default can be `webrtc` for low latency, with fallback logic in frontend.

### Backend Tasks

1. Enable WebRTC listener in mediamtx generated config:
   - `backend/internal/streams/mediamtx.go`
   - Add:
     - `webrtc: yes`
     - `webrtcAddress: :8889`
     - `webrtcLocalUDPAddress: :8189`
2. Add reverse proxy for WebRTC HTTP handshake path:
   - `backend/internal/api/router.go`
   - Proxy `/webrtc/` to `http://127.0.0.1:8889/`
3. Extend stream API response:
   - `backend/internal/api/streams.go`
   - Return `webrtc_url` and `preferred_protocol`.
4. Optional runtime settings for network edge cases:
   - Add settings keys in `internal/settings` for:
     - `webrtc_additional_hosts`
     - `webrtc_ice_servers_json`
   - Wire into mediamtx config generator.

### Frontend Tasks

1. Extend API types:
   - `frontend/src/api/cameras.ts`
   - `StreamInfo` includes `webrtc_url` and `preferred_protocol`.
2. Add `WebRTCPlayer` component:
   - New `frontend/src/components/WebRTCPlayer.tsx`
   - Implement WHEP negotiation with `RTCPeerConnection`.
3. Update `VideoPlayer` orchestration:
   - `frontend/src/components/VideoPlayer.tsx`
   - Try WebRTC first when available.
   - Fall back to HLS on timeout/failure.
4. Update camera view wiring:
   - `frontend/src/pages/CameraView.tsx`
   - Pass both URLs and show active transport badge (`WebRTC` or `HLS`).

### Fallback Rules (Required)

- If WebRTC setup fails within 2-4 seconds, switch to HLS automatically.
- If WebRTC disconnects repeatedly, cool down and keep HLS for session.
- User should not need manual intervention.

### Acceptance Criteria

- Camera view uses WebRTC when available and stable.
- Playback falls back to HLS automatically when WebRTC fails.
- Existing HLS behavior still works for all browsers.
- No regression to screenshot, overlays, and camera controls.

### Verification

- Browser dev tools confirm WebRTC transport in happy path.
- Block UDP or break ICE config, confirm automatic HLS fallback.
- Run manual camera viewing tests on Chrome + Safari/Firefox.

---

## Phase 3 - ONVIF Discovery + PTZ

### Scope

- Discover ONVIF devices on LAN.
- Bootstrap camera entries from discovered streams.
- Add PTZ controls in camera view for ONVIF-capable cameras.

Suggested Go library: `github.com/use-go/onvif`.

### Data Model Changes

Add migration `008_onvif_ptz.sql`:

- `cameras.onvif_enabled` INTEGER NOT NULL DEFAULT 0
- `cameras.onvif_xaddr` TEXT NOT NULL DEFAULT ''
- `cameras.onvif_username` TEXT NOT NULL DEFAULT ''
- `cameras.onvif_password` TEXT NOT NULL DEFAULT ''
- `cameras.onvif_profile_token` TEXT NOT NULL DEFAULT ''
- `cameras.ptz_enabled` INTEGER NOT NULL DEFAULT 0

### Backend Tasks

1. Add ONVIF module:
   - New package `backend/internal/onvif/`
   - `discovery.go`: WS-Discovery scan with timeout
   - `profiles.go`: fetch profiles + stream URI
   - `ptz.go`: continuous move, stop, presets
2. Add API routes:
   - `POST /api/v1/onvif/discover`
   - `POST /api/v1/onvif/probe`
   - `POST /api/v1/cameras/{id}/ptz/move`
   - `POST /api/v1/cameras/{id}/ptz/stop`
   - Optional presets:
     - `GET /api/v1/cameras/{id}/ptz/presets`
     - `POST /api/v1/cameras/{id}/ptz/presets/{preset}/goto`
3. Extend camera model/service/repo:
   - `backend/internal/cameras/model.go`
   - `backend/internal/cameras/service.go`
   - `backend/internal/cameras/repository.go`
4. Add PTZ safety constraints:
   - Command rate limit per camera
   - Force `stop` on button release or timeout

### Frontend Tasks

1. Add discovery UI in Settings:
   - `frontend/src/pages/Settings.tsx`
   - Button: "Discover ONVIF Cameras"
   - Result list with "Add Camera"
2. Add ONVIF fields in camera form:
   - `frontend/src/components/CameraForm.tsx`
3. Add PTZ controls in camera view:
   - `frontend/src/pages/CameraView.tsx`
   - Pan/Tilt arrows + zoom in/out + stop
   - Only shown when `ptz_enabled` is true

### Acceptance Criteria

- Discovery returns devices on local network with profile stream URIs.
- User can import discovered camera into RTSPanda with one flow.
- PTZ commands work for supported cameras and fail safely otherwise.
- Non-PTZ cameras show no PTZ controls.

### Verification

- Test with at least one ONVIF camera (or ONVIF simulator).
- Validate move/stop latency and no stuck movement.
- Confirm camera CRUD remains compatible for non-ONVIF cameras.

---

## Phase 4 - Rules Engine (CEL) + MQTT Output

### Scope

- Add programmable event rules using CEL expressions.
- Trigger MQTT publishes when rules match.
- Keep detection and live view non-blocking on failures.

Suggested library:
- CEL: `github.com/google/cel-go`
- MQTT client: `github.com/eclipse/paho.mqtt.golang`

### Rule Model (Initial)

Add migration `009_automation_rules.sql` with:

- `automation_rules`
  - `id`, `camera_id`, `name`, `enabled`
  - `event_type` (`detection`, `connectivity`, `manual`)
  - `cel_expression` TEXT NOT NULL
  - `cooldown_seconds` INTEGER NOT NULL DEFAULT 0
  - `mqtt_topic` TEXT NOT NULL
  - `mqtt_payload_template` TEXT NOT NULL
  - timestamps
- `automation_rule_runs`
  - `id`, `rule_id`, `camera_id`, `matched`, `error`, `created_at`

Keep legacy `alert_rules` untouched for compatibility.

### Event Context (CEL Input)

Provide consistent JSON-like variables for CEL:

```text
camera.id
camera.name
event.type
event.label
event.confidence
event.count
time.hour
time.weekday
```

Example expressions:

- `event.type == "detection" && event.label == "person" && event.confidence > 0.7`
- `time.hour >= 22 || time.hour <= 6`

### Backend Tasks

1. Add rules package:
   - New `backend/internal/rules/`
   - compile/cache CEL programs
   - evaluate rules against event context
2. Add MQTT notifier package:
   - New `backend/internal/mqtt/`
   - connection manager with retry/backoff
   - publish with QoS/retain from settings
3. Add MQTT settings:
   - Extend `settings` model/service/repo with:
     - `mqtt_enabled`
     - `mqtt_broker_url`
     - `mqtt_client_id`
     - `mqtt_username`
     - `mqtt_password`
     - `mqtt_topic_prefix`
4. Wire rule evaluation into detection pipeline:
   - `backend/internal/detections/manager.go`
   - Evaluate asynchronously after event persistence.
5. Add rules API:
   - CRUD endpoints under `/api/v1/cameras/{id}/automation-rules`
   - dry-run endpoint: `POST /api/v1/automation-rules/test`

### Frontend Tasks

1. Add Automation tab in Settings:
   - `frontend/src/pages/Settings.tsx`
2. Add rule editor form:
   - `frontend/src/components/` new form component
   - fields for expression, cooldown, MQTT topic/payload
3. Add MQTT settings form:
   - provider config + connection test button

### Non-Blocking Guarantees

- If CEL evaluation fails, log and continue.
- If MQTT is down, queue/drop based on policy; never block stream/detection loop.
- Persist rule run outcome for debugging.

### Acceptance Criteria

- Rules can be created, listed, edited, deleted via API/UI.
- CEL expressions evaluate correctly against detection events.
- Matching rules publish MQTT messages with expected topic/payload.
- MQTT or CEL failures do not break live playback or detection writes.

### Verification

- Unit tests for CEL compile/eval and cooldown behavior.
- Integration test with Mosquitto container for end-to-end publish.
- Manual test from live detection event to MQTT subscriber output.

---

## Phase 5 - Auth Proxy Integration

### Scope

- Keep RTSPanda app auth-free internally.
- Add trusted auth proxy integration for protected deployments.
- Support oauth2-proxy / Authelia style forwarded identity headers.

This preserves LAN-first simplicity while enabling secure internet-facing setups.

### Backend Tasks

1. Add auth proxy middleware:
   - New `backend/internal/api/authproxy.go`
2. Env settings:
   - `AUTH_PROXY_ENABLED` (default `false`)
   - `AUTH_PROXY_TRUSTED_CIDRS`
   - `AUTH_PROXY_USER_HEADER` (default `X-Forwarded-User`)
   - `AUTH_PROXY_EMAIL_HEADER` (default `X-Forwarded-Email`)
   - `AUTH_PROXY_GROUPS_HEADER` (default `X-Forwarded-Groups`)
   - `AUTH_PROXY_REQUIRED_GROUP` (optional)
3. Middleware behavior when enabled:
   - Allow only requests coming from trusted proxy CIDRs.
   - Require configured user header.
   - Optionally require group membership.
4. Exemptions:
   - keep `/api/v1/health` reachable.
   - keep `/metrics` reachable only if explicitly desired (document behavior).
5. Add `GET /api/v1/me` for UI display/debug of proxied identity.

### Deployment Artifacts

1. Add example compose file:
   - `docker-compose.auth.yml` with `oauth2-proxy` + RTSPanda
2. Add docs:
   - `docs/AUTH_PROXY.md`
   - Include nginx/caddy example for forward-auth pattern.

### Frontend Tasks

1. Optional user indicator in top UI:
   - fetch `/api/v1/me`
   - display signed-in principal when available

### Acceptance Criteria

- With proxy enabled and trusted headers present, UI/API works.
- Direct requests bypassing proxy are rejected.
- Behavior is unchanged when proxy integration is disabled.

### Verification

- Positive test through proxy.
- Negative test direct-to-app request denied.
- Group-required scenario enforced correctly.

---

## Cross-Cutting Testing Matrix

Run at end of every phase and final integration:

1. Backend tests and vet:
   - `cd backend`
   - `go test ./...`
2. Frontend build:
   - `cd frontend`
   - `npm run build`
3. API smoke:
   - `./scripts/api-smoke.ps1`
4. Manual regression:
   - camera CRUD
   - stream playback
   - recordings list/download
   - detection test endpoint
   - Discord send actions

Final integrated test should include:

- WebRTC happy path and fallback path
- ONVIF discover + PTZ command
- CEL rule match + MQTT publish
- Auth proxy on/off behavior
- Prometheus scraping dashboard

---

## Rollout and Risk Controls

1. Feature flags / safe defaults:
   - Keep new features disabled by default where possible.
2. Backward compatibility:
   - Do not break existing HLS-only clients.
   - Keep legacy alert rule endpoints intact.
3. Migration safety:
   - additive schema changes only.
4. Performance guardrails:
   - no blocking network calls on main stream paths.
   - add timeouts/retries with bounded limits.

---

## Deliverables Checklist

- [ ] New doc references added where relevant (`README.md`, `human/USER_GUIDE.md`)
- [ ] Migrations `008`, `009` added and tested
- [ ] New APIs documented in README API section
- [ ] `AI/TODO.md` updated with phase tasks and status
- [ ] `AI/HANDOFF.md` updated with implementation notes/blockers

