# RTSPanda — Handoff

## Latest Handoff: 2026-03-13 — YOLOv8 UI + Per-Camera Tracking + Discord Rich Alerts

### Summary

Completed the end-to-end AI UX and alerting layer on top of the detection foundation:

- Per-camera YOLOv8 tracking settings in camera config
- Live detection overlays rendered in the camera player UI
- Detection event/history panel with snapshot previews
- Per-camera Discord rich-media webhook alerts with cooldown
- Detection frame dimension persistence for accurate overlay scaling

This turn shipped both backend data/model/pipeline updates and frontend UI integration.

---

### Files Changed

- `ai_worker/app/main.py`
- `backend/cmd/rtspanda/main.go`
- `backend/internal/cameras/model.go`
- `backend/internal/cameras/repository.go`
- `backend/internal/cameras/service.go`
- `backend/internal/db/migrations/004_camera_tracking_discord.sql` (new)
- `backend/internal/detections/model.go`
- `backend/internal/detections/repository.go`
- `backend/internal/detections/manager.go`
- `backend/internal/notifications/discord.go` (new)
- `frontend/src/api/cameras.ts`
- `frontend/src/api/detections.ts` (new)
- `frontend/src/components/CameraForm.tsx`
- `frontend/src/components/VideoPlayer.tsx`
- `frontend/src/pages/CameraView.tsx`
- `frontend/src/pages/Settings.tsx`

---

### What Works

- Camera model now supports:
  - `tracking_enabled`
  - `tracking_min_confidence`
  - `tracking_labels`
  - `discord_alerts_enabled`
  - `discord_webhook_url`
  - `discord_mention`
  - `discord_cooldown_seconds`
- Detection samplers only run for cameras that are both `enabled` and `tracking_enabled`.
- Detection filtering now happens per camera:
  - confidence threshold
  - optional label allow-list
- Detection events now persist `frame_width` and `frame_height`.
- Camera view UI supports:
  - quick tracking toggle
  - run-test-detection action
  - live overlay toggle
  - event history panel with grouped snapshots
- Discord notifier sends rich embeds with attached snapshot media and bbox/confidence details.
- Discord alert cooldown is enforced per camera.

---

### Verification

- `go build ./...` (backend): pass
- `npm run lint` (frontend): pass
- `npm run build` (frontend): pass
- `.\build.ps1` (embed + binary build): pass

---

### Known Constraints

- Current pipeline is frame-sampling detection, not persistent multi-object track IDs.
- Discord sends one message per sampled event batch (subject to cooldown), not per-object rate limiting.
- No retention/cleanup policy yet for detection snapshots/events.
- Frontend bundle remains large; Vite chunk warning still present.

---

### Recommended Next Steps

1. Add retention policy for `data/snapshots/detections` and old `detection_events`.
2. Add pagination/date-range filters for detection history API and UI.
3. Add optional object-level suppression windows (e.g. same label cooldown).
4. Add integration tests for migration `004` and detection filtering/Discord notify path.
