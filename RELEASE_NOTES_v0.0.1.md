# RTSPanda v0.0.1 — Initial Release

**Self-hosted RTSP camera viewer. One binary. Zero cloud. Full control.**

This is the first production-ready release of RTSPanda: a single executable that turns any RTSP camera into a browser-viewable live stream with recording, screenshots, and a clean REST API for integrations.

---

## What's in this release

### Core experience
- **Single binary** — Go backend with embedded React frontend. No separate web server, no container required. Run one executable and open a browser.
- **Live HLS streaming** — Cameras are relayed through [mediamtx](https://github.com/bluenviron/mediamtx) (embedded as a child process). RTSP → HLS conversion happens on your machine; browsers get a standard, compatible stream.
- **On-demand connections** — Streams open only when someone is watching and close after idle. No unnecessary load on cameras or network.
- **Dashboard & single-camera view** — Grid of all cameras with status (Online / Connecting / Offline); click through to full-width live view with native video controls.

### Recording & capture
- **Per-camera recording** — Enable "Record to disk" in Settings. mediamtx writes 1-hour MP4 segments to disk; browse, download, and delete from the UI.
- **Screenshots** — One-click PNG capture from the live stream.

### Integrations
- **REST API** — Full CRUD for cameras, stream status and HLS URLs, recordings, and alert rules. See the [README](https://github.com/248Tech/RTSPanda#rest-api-summary) for endpoints.
- **Alert rules & webhooks** — Define rules per camera (connectivity, motion, object_detection). External scripts or AI systems POST events to `POST /api/v1/alerts/{rule_id}/events`. RTSPanda stores rules and event history; you own the detection logic.

### Operations
- **SQLite** — Single-file database, WAL mode. No external DB setup.
- **Configuration** — Environment variables only: `PORT`, `DATA_DIR`, `MEDIAMTX_BIN`. No config files required.
- **Graceful degradation** — If mediamtx is not present, the app still runs; cameras show offline and the UI remains fully usable.

---

## Requirements

- **Go 1.22+** and **Node.js 18+** for building from source.
- **mediamtx** binary (optional): [download](https://github.com/bluenviron/mediamtx/releases) and place in `mediamtx/` or set `MEDIAMTX_BIN`. Without it, streams appear offline but the app runs.

---

## Quick start

```bash
git clone https://github.com/248Tech/RTSPanda.git
cd RTSPanda
# Place mediamtx binary in mediamtx/ (see README)
make build          # or .\build.ps1 on Windows
./backend/rtspanda  # or .\backend\rtspanda.exe on Windows
```

Open **http://localhost:8080**, add a camera in **Settings → Cameras**, and start watching.

---

## Documentation

- **[README](https://github.com/248Tech/RTSPanda)** — Quick start, features, API summary, configuration, architecture.
- **[User Guide](https://github.com/248Tech/RTSPanda/blob/master/human/USER_GUIDE.md)** — Installation, first camera, recording, alerts, RTSP URL tips, troubleshooting, security.

---

## Security note

This release has **no built-in authentication**. Intended for use on a trusted network, behind a VPN, or behind a reverse proxy with auth. Do not expose directly to the public internet without additional protection.

---

## License

MIT. See [LICENSE](https://github.com/248Tech/RTSPanda/blob/master/LICENSE).

---

We’re excited to ship this. Feedback and contributions are welcome — open an issue or PR on GitHub.

**248Tech**
