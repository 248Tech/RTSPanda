# RTSPanda — Handoff

---

## Latest Handoff: 2026-03-09 — Backend Complete, Dashboard UI Done

### Summary

The entire Go backend is implemented and smoke-tested. The React frontend has a working dashboard, routing, and all camera-grid components. The next work is the HLS video player (TASK-008) and the Settings/camera management UI (TASK-009).

### What Is Done

**Backend (all complete, `go build ./...` clean):**
- `backend/internal/db/` — SQLite connection, WAL mode, embed.FS migration runner
- `backend/internal/cameras/` — Camera model, repo (raw SQL), service (UUID, validation)
- `backend/internal/api/` — Full CRUD handlers + stream status endpoint + HLS reverse proxy
- `backend/internal/streams/` — mediamtx subprocess manager, watchdog, config generation, stream status polling
- `backend/cmd/rtspanda/main.go` — wires everything; graceful shutdown; `DATA_DIR` / `PORT` env vars
- `.gitignore` — covers mediamtx binary, data dir, dist

**Frontend (component layer complete, `npm run build` clean):**
- `src/App.tsx` — Custom `usePath` router hook, Navbar, page routing (`/`, `/settings`, `/cameras/:id`), `CameraViewPlaceholder` stub for TASK-008
- `src/pages/Dashboard.tsx` — Fetches cameras on load, polls every 30s, loading/error/empty states
- `src/components/CameraGrid.tsx` — Responsive grid (1/2/3/4 col), renders CameraCard
- `src/components/CameraCard.tsx` — Fetches stream status per card, 16:9 placeholder thumbnail, click-to-navigate
- `src/components/StatusBadge.tsx` — `online` / `offline` / `connecting` states with correct Tailwind tokens
- `src/components/EmptyState.tsx` — "No cameras configured" with Add Camera CTA
- `src/api/cameras.ts` — Typed API client for all endpoints
- `tailwind.config.ts` — Custom design tokens (`base`, `card`, `accent`, `status-*`, `text-*`)

**Not yet done (stubs):**
- `src/pages/Settings.tsx` — Stub only (heading text); TASK-009
- No `VideoPlayer.tsx` / `CameraView.tsx` — TASK-008
- No `CameraForm.tsx` — TASK-009

### Open Issues

1. **No Vite dev proxy** — Add to `frontend/vite.config.ts`:
   ```ts
   server: { proxy: { '/api': 'http://localhost:8080', '/hls': 'http://localhost:8080' } }
   ```
   Without this, `npm run dev` can't reach the Go API. Not blocking production (Go embeds the frontend), but blocking local dev testing.

2. **mediamtx binary not installed** — Streaming is in graceful-disabled mode. Download the binary for the target platform and place at `mediamtx/mediamtx[.exe]`, or set `MEDIAMTX_BIN` env var. Releases: https://github.com/bluenviron/mediamtx/releases

3. **`backend/data/rtspanda.db` on disk** — Leftover from smoke tests. Covered by `.gitignore` but present locally. Safe to delete.

4. **`Settings.tsx` uses raw Tailwind colors** (`slate-200`, `slate-500`) instead of design tokens. Acceptable since TASK-009 will rewrite the file entirely.

### Resolved (Previously Outstanding)

- ✓ Go version — 1.26 (verified)
- ✓ UUID library — `github.com/google/uuid` confirmed
- ✓ PORT env var — implemented
- ✓ mediamtx subprocess management — watchdog + graceful shutdown + disabled mode
- ✓ Frontend CORS — handled by embedded binary in production; dev proxy is the only open gap

### Next Recommended Work

**TASK-008 and TASK-009 can be started in parallel.**

- **TASK-008** — Install `hls.js` (`npm install hls.js`), create `VideoPlayer.tsx` + `CameraView.tsx`, replace `CameraViewPlaceholder` in `App.tsx`. See `AI/UXDesign/DASHBOARD_UX.md` for layout spec.
- **TASK-009** — Implement `Settings.tsx` (camera list with edit/delete), `CameraForm.tsx` (modal with name + RTSP URL + enabled toggle). Use `addCamera`, `updateCamera`, `deleteCamera` from `src/api/cameras.ts`.

After both: **TASK-010** (embed frontend into Go binary) then **TASK-011** (Docker).

---

## Previous Handoffs

### 2026-03-09 — Initial Planning Complete

Architecture planning complete. All 7 architecture decisions made. 11 implementation tasks decomposed. AI coordination files bootstrapped. No code existed yet.
