# RTSPanda — Phase 1 Verification Plan

Last updated: 2026-03-09

This document is a runnable checklist. Each section has copy-pasteable commands.
Run sections in order — later sections depend on state created by earlier ones.

---

## Prerequisites

### Tools needed
- `curl` (or any REST client — HTTPie, Bruno, Postman)
- A browser (Chrome or Firefox recommended; Safari for native HLS test)
- `jq` (optional, for pretty-printing curl output)

### For streaming tests only
- mediamtx binary at `mediamtx/mediamtx.exe` (Windows) or `mediamtx/mediamtx` (Linux)
- A real RTSP camera **or** a software RTSP source (see §5 for ffmpeg test source)

### Start the backend
```bash
cd backend
DATA_DIR=./data PORT=8080 go run ./cmd/rtspanda
```

Expected startup log (without mediamtx binary):
```
streams: WARNING — mediamtx binary not found...
streams: streaming disabled...
RTSPanda listening on :8080 (data: ./data)
```

Expected startup log (with mediamtx binary):
```
RTSPanda listening on :8080 (data: ./data)
```
mediamtx output will interleave. That's normal.

---

## 1. Backend API Smoke Tests

Run these with the server on port 8080. All should pass with zero cameras configured.

### 1.1 Health check
```bash
curl -s http://localhost:8080/api/v1/health
```
**Expected:** `{"status":"ok"}`

### 1.2 List cameras (empty)
```bash
curl -s http://localhost:8080/api/v1/cameras
```
**Expected:** `[]` (not `null`)

### 1.3 404 on unknown camera
```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/cameras/doesnotexist
```
**Expected:** `404`

### 1.4 404 on stream for unknown camera
```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/cameras/doesnotexist/stream
```
**Expected:** `404`

### 1.5 Server stays up after bad requests
Send several malformed requests and confirm health still returns 200:
```bash
curl -s -X POST http://localhost:8080/api/v1/cameras -d "not json"
curl -s http://localhost:8080/api/v1/health
```
**Expected:** health still returns `{"status":"ok"}`

---

## 2. Camera CRUD Test Cases

Set a variable for convenience. Adjust shell syntax for your OS.
```bash
BASE=http://localhost:8080/api/v1
```

### 2.1 Create — happy path
```bash
curl -s -X POST $BASE/cameras \
  -H "Content-Type: application/json" \
  -d '{"name":"Front Door","rtsp_url":"rtsp://192.168.1.100/stream1"}'
```
**Expected:**
- HTTP 201
- Body has `id` (UUID format: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)
- `enabled: true` by default
- `position: 0`
- `created_at` and `updated_at` are ISO timestamps

Capture the ID:
```bash
ID=$(curl -s -X POST $BASE/cameras \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Camera","rtsp_url":"rtsp://10.0.0.1/ch0"}' | \
  grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Camera ID: $ID"
```

### 2.2 Create — explicit enabled=false
```bash
curl -s -X POST $BASE/cameras \
  -H "Content-Type: application/json" \
  -d '{"name":"Garage","rtsp_url":"rtsp://10.0.0.5/live","enabled":false}'
```
**Expected:** `"enabled":false` in response

### 2.3 Create — validation: missing name
```bash
curl -s -X POST $BASE/cameras \
  -H "Content-Type: application/json" \
  -d '{"rtsp_url":"rtsp://10.0.0.1/ch0"}'
```
**Expected:** HTTP 400, body contains `"error"` key with message including "name"

### 2.4 Create — validation: missing rtsp_url
```bash
curl -s -X POST $BASE/cameras \
  -H "Content-Type: application/json" \
  -d '{"name":"Test"}'
```
**Expected:** HTTP 400, message includes "rtsp_url"

### 2.5 Create — validation: empty body
```bash
curl -s -X POST $BASE/cameras -H "Content-Type: application/json" -d '{}'
```
**Expected:** HTTP 400

### 2.6 Create — invalid JSON
```bash
curl -s -X POST $BASE/cameras -H "Content-Type: application/json" -d 'oops'
```
**Expected:** HTTP 400, `{"error":"invalid JSON"}`

### 2.7 Get by ID
```bash
curl -s $BASE/cameras/$ID
```
**Expected:** HTTP 200, same fields as the create response

### 2.8 List cameras (non-empty)
```bash
curl -s $BASE/cameras
```
**Expected:** JSON array, at least one element, each with all Camera fields

### 2.9 Update — rename
```bash
curl -s -X PUT $BASE/cameras/$ID \
  -H "Content-Type: application/json" \
  -d '{"name":"Front Door Renamed"}'
```
**Expected:** HTTP 200, `name` is updated, `updated_at` is newer than `created_at`, `rtsp_url` unchanged

### 2.10 Update — disable
```bash
curl -s -X PUT $BASE/cameras/$ID \
  -H "Content-Type: application/json" \
  -d '{"enabled":false}'
```
**Expected:** HTTP 200, `"enabled":false`

### 2.11 Update — re-enable
```bash
curl -s -X PUT $BASE/cameras/$ID \
  -H "Content-Type: application/json" \
  -d '{"enabled":true}'
```
**Expected:** HTTP 200, `"enabled":true`

### 2.12 Update — empty name rejected
```bash
curl -s -X PUT $BASE/cameras/$ID \
  -H "Content-Type: application/json" \
  -d '{"name":""}'
```
**Expected:** HTTP 400

### 2.13 Update — 404 on unknown ID
```bash
curl -s -X PUT $BASE/cameras/00000000-0000-0000-0000-000000000000 \
  -H "Content-Type: application/json" \
  -d '{"name":"Ghost"}'
```
**Expected:** HTTP 404

### 2.14 Delete
```bash
curl -s -o /dev/null -w "%{http_code}" -X DELETE $BASE/cameras/$ID
```
**Expected:** `204`

### 2.15 Get deleted camera
```bash
curl -s -o /dev/null -w "%{http_code}" $BASE/cameras/$ID
```
**Expected:** `404`

### 2.16 Delete — 404 on already-deleted
```bash
curl -s -o /dev/null -w "%{http_code}" -X DELETE $BASE/cameras/$ID
```
**Expected:** `404`

### 2.17 Persistence across restart
1. Create a camera (note its `id`)
2. Stop the server (`Ctrl+C`)
3. Start it again (`go run ./cmd/rtspanda`)
4. `curl -s $BASE/cameras/$ID`
**Expected:** Camera is still there with the same data

---

## 3. Stream Status API Tests

### 3.1 Stream status — mediamtx disabled
With no mediamtx binary:
```bash
ID=$(curl -s -X POST $BASE/cameras \
  -H "Content-Type: application/json" \
  -d '{"name":"Cam A","rtsp_url":"rtsp://10.0.0.1/ch0"}' | \
  grep -o '"id":"[^"]*"' | cut -d'"' -f4)
curl -s $BASE/cameras/$ID/stream
```
**Expected:**
```json
{"hls_url":"/hls/camera-<id>/index.m3u8","status":"offline"}
```

### 3.2 HLS proxy — returns 502 or error when mediamtx not running
```bash
curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/hls/camera-test/index.m3u8"
```
**Expected:** `502` (reverse proxy target unreachable — this is correct behaviour)

### 3.3 Stream status — with mediamtx running, camera not actively viewed
*(Requires mediamtx binary)*

Start server, add a camera with a valid RTSP URL. Query stream status without opening the player.
```bash
curl -s $BASE/cameras/$ID/stream
```
**Expected:** `"status":"offline"` or `"status":"connecting"` — not `"online"` until someone is watching (`sourceOnDemand` is working)

### 3.4 Stream status — `online` when actively playing
*(Requires mediamtx binary + working RTSP source)*

Open the camera view in the browser and poll status while it's playing:
```bash
watch -n 2 "curl -s $BASE/cameras/$ID/stream"
```
**Expected:** `"status":"online"` within a few seconds of the player connecting

### 3.5 Stream status — returns `offline` for disabled camera
```bash
curl -s -X PUT $BASE/cameras/$ID -H "Content-Type: application/json" -d '{"enabled":false}'
curl -s $BASE/cameras/$ID/stream
```
**Expected:** `"status":"offline"` (mediamtx path removed; sourceOnDemand won't trigger)

---

## 4. mediamtx Subprocess Tests

*(All require the mediamtx binary installed)*

### 4.1 mediamtx starts with Go app
Check process list after starting server:
```bash
# Linux/macOS
ps aux | grep mediamtx | grep -v grep

# Windows (PowerShell)
Get-Process mediamtx -ErrorAction SilentlyContinue
```
**Expected:** mediamtx process is running as a child

### 4.2 Config file written to DATA_DIR
```bash
cat ./data/mediamtx.yml
```
**Expected:** Valid YAML with `api: yes`, `hlsAddress: :8888`, `sourceOnDemand: yes` for each enabled camera

### 4.3 Camera added → appears in mediamtx config / API
After adding a camera via API, check mediamtx's path list:
```bash
curl -s http://127.0.0.1:9997/v3/paths/list
```
**Expected:** `items` array contains `camera-<id>` path

### 4.4 Camera deleted → removed from mediamtx
Delete a camera, then:
```bash
curl -s http://127.0.0.1:9997/v3/paths/list
```
**Expected:** `camera-<id>` path no longer present

### 4.5 Watchdog restarts mediamtx on crash
With server running, kill mediamtx manually:
```bash
# Linux/macOS
pkill mediamtx

# Windows (PowerShell)
Stop-Process -Name mediamtx -Force
```
Wait 4 seconds. Check process list again.
**Expected:** mediamtx restarts automatically; server log shows "mediamtx exited... restarting in 3s"

### 4.6 mediamtx stops on Go app exit
Stop the Go server with `Ctrl+C`. Check process list.
**Expected:** mediamtx process is gone within a few seconds; no orphan process

---

## 5. Frontend Visual + Interaction Tests

Start the server with the frontend via `npm run dev` (after adding Vite proxy to `vite.config.ts`),
or use the embedded build via `go run ./cmd/rtspanda` after `npm run build`.

### 5.1 Dashboard — empty state
Open `http://localhost:8080` (or `http://localhost:5173` in dev mode) with no cameras configured.
- [ ] Dark background, panda navbar visible
- [ ] "No cameras configured" message shown
- [ ] "Add Camera" button present
- [ ] Clicking "Add Camera" navigates to `/settings`

### 5.2 Dashboard — camera grid
Add 1–5 cameras via Settings, then go back to dashboard.
- [ ] Each camera shows a card with name and status badge
- [ ] Status badge shows "Offline" (if no stream) or "Online"/"Connecting"
- [ ] Grid is 1 column on narrow viewport (< 640px)
- [ ] Grid is 2 columns on tablet (640–1024px)
- [ ] Grid is 3 columns on desktop (1024–1280px)
- [ ] Grid is 4 columns on wide screen (> 1280px)
- [ ] Resize browser — layout reflows without breaking

### 5.3 Dashboard — keyboard navigation
- [ ] Tab through all camera cards
- [ ] Focus ring visible on each card
- [ ] Enter/Space on a focused card navigates to that camera

### 5.4 Dashboard — 30-second poll
- [ ] Add a camera via API directly (curl) while the dashboard is open
- [ ] After ≤ 30s, the new camera appears without a page reload

### 5.5 Settings — camera list
Navigate to `/settings`.
- [ ] All cameras are listed with name, RTSP URL, and enabled/disabled status
- [ ] Edit and Delete buttons present on each row
- [ ] "Add Camera" button in header

### 5.6 Settings — add camera (happy path)
Click "+ Add Camera":
- [ ] Modal opens with Name, RTSP URL fields and Enabled toggle
- [ ] Submit with valid name and RTSP URL
- [ ] Modal closes, new camera appears in list immediately
- [ ] Navigate to dashboard — new camera card appears

### 5.7 Settings — add camera (validation)
Open the Add Camera modal:
- [ ] Submit with empty Name → error shown, modal stays open
- [ ] Submit with empty RTSP URL → error shown
- [ ] Fix fields and submit → succeeds

### 5.8 Settings — edit camera
Click Edit on an existing camera:
- [ ] Modal opens with existing values pre-filled
- [ ] Change name and save
- [ ] Updated name appears in the camera list immediately
- [ ] Toggle enabled/disabled → updates in list

### 5.9 Settings — delete camera
Click Delete on a camera:
- [ ] Confirmation dialog shown with camera name
- [ ] Cancel → dialog closes, camera still present
- [ ] Confirm → camera removed from list
- [ ] Navigate to dashboard → deleted camera no longer shown

### 5.10 Settings — error state
With server stopped, open Settings:
- [ ] Error message shown ("Could not load cameras")
- [ ] App does not crash

### 5.11 Camera view — VideoPlayer wired up
*(Requires TASK-008 to wire `CameraViewPlaceholder` → real `CameraView` in `App.tsx`)*

Click a camera card on the dashboard:
- [ ] Navigates to `/cameras/<id>`
- [ ] Camera name shown
- [ ] VideoPlayer renders in 16:9 aspect ratio
- [ ] Loading spinner shown while stream starts
- [ ] Browser Back button returns to dashboard

### 5.12 Camera view — stream playback
*(Requires mediamtx binary + working RTSP source)*
- [ ] Video starts playing within ~5 seconds
- [ ] Video is smooth (no continuous buffering)
- [ ] Controls (play/pause/volume) are accessible
- [ ] Leaving the page for 10+ seconds — confirm `sourceOnDemand` closes the RTSP connection (check mediamtx logs)
- [ ] Return to camera view — stream reconnects

### 5.13 Camera view — error state
Point a camera at an invalid RTSP URL (e.g. `rtsp://localhost:9999/dead`):
- [ ] Player shows error overlay ("Network error" or "Playback error")
- [ ] Retry button present and clickable
- [ ] Clicking Retry reloads the stream

### 5.14 Safari / native HLS fallback
*(If you have Safari available)*
- [ ] Open camera view in Safari
- [ ] Video plays via native HLS (no hls.js involved)
- [ ] Check devtools — no hls.js errors

---

## 6. End-to-End Test with Real RTSP Camera

### 6a. Using a real IP camera

1. **Start server with mediamtx binary**
   ```bash
   cd backend && DATA_DIR=./data go run ./cmd/rtspanda
   ```

2. **Add camera via Settings UI** (or curl):
   ```bash
   curl -s -X POST http://localhost:8080/api/v1/cameras \
     -H "Content-Type: application/json" \
     -d '{"name":"Living Room","rtsp_url":"rtsp://<user>:<pass>@<camera-ip>:554/stream1"}'
   ```

3. **Verify stream status goes connecting then online:**
   ```bash
   watch -n 2 "curl -s http://localhost:8080/api/v1/cameras/<ID>/stream"
   ```

4. **Open camera view in browser** → video plays

5. **Close browser tab** → wait 15 seconds → check mediamtx logs confirm source disconnected

6. **Re-open camera view** → stream reconnects within ~5 seconds

### 6b. Using ffmpeg as a software RTSP source (no real camera needed)

Install ffmpeg, then create a looping RTSP stream:
```bash
ffmpeg -re -f lavfi -i "testsrc=size=1280x720:rate=25" \
  -f lavfi -i anullsrc \
  -c:v libx264 -preset ultrafast -tune zerolatency \
  -c:a aac \
  -f rtsp rtsp://localhost:8554/test
```

Add the camera with URL `rtsp://localhost:8554/test`.
mediamtx will relay this to HLS. Open in the browser — you should see the ffmpeg test pattern video.

### 6c. Acceptance gate

All of the following must be true before Phase 1 is considered shippable:

- [ ] Can open `http://localhost:8080` in a browser
- [ ] Can add a camera via the Settings page
- [ ] Camera appears on the dashboard grid with a status badge
- [ ] Status badge transitions to "Online" when stream is active
- [ ] Can click a camera card and watch live video
- [ ] Closing the video tab disconnects the RTSP source (check mediamtx logs for "sourceOnDemandCloseAfter")
- [ ] App survives a server restart — cameras persist, mediamtx restarts cleanly
- [ ] No orphan mediamtx process after server exit

---

## 7. TASK-010 Regression Checklist (Embed Frontend)

Run after `npm run build` + `go build` produces the single binary.

### Build verification
```bash
cd frontend && npm run build        # produces frontend/dist/
cd ../backend && go build -o rtspanda ./cmd/rtspanda
ls -lh rtspanda                     # check binary size: should be < 50MB
```

### Runtime verification (no separate frontend server)
```bash
DATA_DIR=./data ./rtspanda
```

- [ ] `http://localhost:8080` serves the React app (not 404)
- [ ] CSS and JS assets load (check Network tab in devtools — no 404s on assets)
- [ ] `http://localhost:8080/settings` renders Settings page (deep link works via SPA fallback)
- [ ] `http://localhost:8080/cameras/<id>` renders camera view (deep link works)
- [ ] `http://localhost:8080/api/v1/health` still returns `{"status":"ok"}` (API not shadowed by static handler)
- [ ] `http://localhost:8080/hls/` still reverse-proxies to mediamtx (not shadowed)
- [ ] Page reload at `/settings` works (not 404 — requires SPA catch-all in `static.go`)
- [ ] Browser history: back/forward navigate correctly
- [ ] Static assets have cache headers (`Cache-Control: public, max-age=...` on `.js`/`.css`, no-cache on `index.html`)

---

## 8. TASK-011 Regression Checklist (Docker)

Run after `Dockerfile` and `docker-compose.yml` are implemented.

### Build
```bash
docker compose build
docker images | grep rtspanda    # check image size: should be < 150MB
```

### First run
```bash
docker compose up
```
- [ ] Container starts without errors
- [ ] `http://localhost:8080/api/v1/health` returns 200
- [ ] `http://localhost:8080` serves the frontend
- [ ] mediamtx starts inside the container (check `docker compose logs`)

### SQLite persistence
```bash
# Add a camera
curl -s -X POST http://localhost:8080/api/v1/cameras \
  -H "Content-Type: application/json" \
  -d '{"name":"Persist Test","rtsp_url":"rtsp://test/stream"}'

# Stop and restart
docker compose down
docker compose up

# Camera should still exist
curl -s http://localhost:8080/api/v1/cameras
```
- [ ] Camera created before restart is present after restart
- [ ] `./data/rtspanda.db` exists on the host (volume mount working)

### Image size
```bash
docker images rtspanda --format "{{.Size}}"
```
- [ ] Under 150MB

### Environment variable config
```bash
docker compose run --rm -e PORT=9090 rtspanda
curl -s http://localhost:9090/api/v1/health
```
- [ ] `PORT` env var changes the listen port

### mediamtx binary in image
```bash
docker compose exec rtspanda ls -lh /usr/local/bin/mediamtx
```
- [ ] Binary present and executable in the container

### Graceful shutdown
```bash
docker compose stop   # sends SIGTERM
```
- [ ] Container exits within 10 seconds (no SIGKILL needed)
- [ ] No `shutdown error:` line in logs

---

## Appendix: Quick Reference

### Run server
```bash
cd backend && DATA_DIR=./data PORT=8080 go run ./cmd/rtspanda
```

### Create a test camera
```bash
curl -s -X POST http://localhost:8080/api/v1/cameras \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","rtsp_url":"rtsp://10.0.0.1/stream"}' | jq
```

### mediamtx internal API (when binary running)
```bash
curl -s http://127.0.0.1:9997/v3/paths/list | jq '.items[].name'
```

### Delete all test cameras (reset state)
```bash
for id in $(curl -s http://localhost:8080/api/v1/cameras | grep -o '"id":"[^"]*"' | cut -d'"' -f4); do
  curl -s -X DELETE http://localhost:8080/api/v1/cameras/$id
  echo "Deleted $id"
done
```

### Delete test data directory
```bash
rm -rf backend/data
```
