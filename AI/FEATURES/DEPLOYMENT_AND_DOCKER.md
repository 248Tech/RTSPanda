# Feature Spec: Deployment — TASK-010 + TASK-011

Status: Spec complete — Ready for implementation
Last updated: 2026-03-09

---

## Overview

Phase 1 ships as a single Docker container:

- One Go binary (serves HTTP on 8080, manages mediamtx)
- React frontend **embedded inside the binary** via `go:embed`
- mediamtx binary **copied into the image** at `/usr/local/bin/mediamtx`
- SQLite database on a **host-mounted volume** at `/data`
- mediamtx HLS output on port 8888 — **internal only**, reverse-proxied by Go

```
docker compose up
       │
       ▼
┌─────────────────────────────────────────────┐
│  alpine container                           │
│                                             │
│  /usr/local/bin/rtspanda  ← Go binary       │
│    ├─ embedded frontend/dist (go:embed)     │
│    ├─ spawns: /usr/local/bin/mediamtx       │
│    └─ opens: /data/rtspanda.db              │
│                                             │
│  port 8080 ─────────────────────────────── │── browser
│  port 8888  (HLS, internal only)            │
└─────────────────────────────────────────────┘
         │
    /data/ ← volume mount (SQLite persists here)
```

---

## TASK-010: Embed Frontend into Go Binary

### The `go:embed` Path Constraint

`go:embed` can only reference paths **inside or below** the directory containing the `.go` file. It cannot traverse `../` upward. Since `static.go` lives at `backend/internal/api/static.go`, it can only embed paths rooted in `backend/internal/api/`.

**Convention:** a gitignored directory `backend/internal/api/dist/` is the build-time bridge. The build script (Makefile or Dockerfile) places the Vite output there before `go build` runs.

```
Build step:
  1. cd frontend && npm run build       → produces frontend/dist/
  2. cp -r frontend/dist/* backend/internal/api/dist/
  3. cd backend && go build ./cmd/rtspanda
  4. (optional cleanup: rm -rf backend/internal/api/dist/)
```

In Dockerfile this becomes a `COPY --from=frontend-builder` line (see §TASK-011).

Add to `.gitignore`:
```
backend/internal/api/dist/
```

### `backend/internal/api/static.go` — Implementation

```go
package api

import (
    "embed"
    "io/fs"
    "net/http"
)

//go:embed dist
var distFS embed.FS

// staticHandler returns a handler that serves the embedded React app.
// All requests not matched by the API mux fall through to this.
// For SPA routing: any path that doesn't correspond to an asset serves index.html.
func staticHandler() http.Handler {
    sub, err := fs.Sub(distFS, "dist")
    if err != nil {
        panic(err)
    }
    fileServer := http.FileServer(http.FS(sub))
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Try to serve the file. If it doesn't exist, serve index.html.
        f, err := sub.Open(r.URL.Path)
        if err != nil {
            // SPA fallback: let React Router handle the path
            r2 := r.Clone(r.Context())
            r2.URL.Path = "/"
            fileServer.ServeHTTP(w, r2)
            return
        }
        f.Close()
        fileServer.ServeHTTP(w, r)
    })
}
```

### Router Changes (NewRouter)

Add the static catch-all **last** in the mux. All `/api/` and `/hls/` routes must be registered first:

```go
// At the end of NewRouter, after all API routes:
mux.Handle("/", staticHandler())
```

**Order matters.** Go's `http.ServeMux` uses longest-prefix matching for paths with trailing slashes. Since `/api/` and `/hls/` are more specific, they win. `/` catches everything else.

### Cache Headers for Static Assets

Assets in `dist/assets/` have content-hashed filenames (Vite default). They can be cached aggressively. `index.html` must not be cached. Implement in `staticHandler`:

```go
// If path starts with /assets/, set long cache
w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

// For index.html (and the SPA fallback):
w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
```

### Makefile

Create `Makefile` at the repo root. This is the canonical build entry point for both local and CI:

```makefile
.PHONY: build clean dev

BINARY=backend/rtspanda

build: frontend-build
	@cp -r frontend/dist backend/internal/api/dist
	cd backend && CGO_ENABLED=0 go build -ldflags="-s -w" -o rtspanda ./cmd/rtspanda
	@rm -rf backend/internal/api/dist

frontend-build:
	cd frontend && npm ci && npm run build

clean:
	rm -f backend/rtspanda
	rm -rf backend/internal/api/dist
	rm -rf frontend/dist

dev:
	@echo "Start Go: cd backend && go run ./cmd/rtspanda"
	@echo "Start Vite: cd frontend && npm run dev"
```

Notes on the build flags:
- `CGO_ENABLED=0` — `modernc.org/sqlite` is pure Go. No C compiler needed. Enables cross-compilation.
- `-ldflags="-s -w"` — strips debug info and DWARF. Typical reduction: 30–40% smaller binary.
- For Docker cross-compile: add `GOOS=linux GOARCH=amd64` before `go build`.

### TASK-010 Acceptance Criteria

- [ ] `make build` produces `backend/rtspanda` (single binary, no external files)
- [ ] Running `./backend/rtspanda` serves `http://localhost:8080` — React app loads
- [ ] `GET /api/v1/health` returns 200 (API not shadowed by static handler)
- [ ] `GET /hls/...` reverse-proxies (not shadowed)
- [ ] Direct browser navigation to `/settings` serves `index.html`, not 404
- [ ] Direct navigation to `/cameras/<id>` serves `index.html`, not 404
- [ ] Browser reload at any SPA route works
- [ ] Assets in `/assets/` have long-lived `Cache-Control` headers
- [ ] `index.html` has `no-cache` header
- [ ] Binary size: `ls -lh backend/rtspanda` — target < 50MB

---

## TASK-011: Dockerfile + docker-compose

### Multi-Stage Dockerfile Design

Three stages. Each stage has a single job.

```
Stage 1 (frontend):  node:24-alpine  → produces frontend/dist/
Stage 2 (go-build):  golang:1.26-alpine → produces /rtspanda binary
Stage 3 (runtime):   alpine:3.x       → final image
```

**Why not two stages?** The mediamtx binary can be downloaded in a dedicated fourth stage or baked into stage 3. See §mediamtx in Docker below.

### Stage 1: Frontend Builder

```dockerfile
FROM node:24-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build
# Output: /app/dist/
```

Use `npm ci` (not `npm install`) — reproducible, uses lockfile exactly.

### Stage 2: Go Builder

```dockerfile
FROM golang:1.26-alpine AS go-builder
WORKDIR /app/backend

# Cache go module downloads as a separate layer
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source
COPY backend/ .

# Copy frontend build output to the expected embed path
COPY --from=frontend-builder /app/dist ./internal/api/dist

# Build: pure Go, no CGO needed (modernc.org/sqlite)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /rtspanda ./cmd/rtspanda
```

The `go mod download` layer is separate from source copy so it's cached across code-only changes.

### Stage 3 (Option A): Download mediamtx at build time

**Recommended.** The Dockerfile downloads the correct Linux binary from GitHub Releases. No binary needs to be in the developer's working tree for Docker builds.

```dockerfile
FROM alpine:3.21 AS mediamtx-downloader
ARG MEDIAMTX_VERSION=v1.12.3
RUN apk add --no-cache curl tar && \
    curl -fsSL "https://github.com/bluenviron/mediamtx/releases/download/${MEDIAMTX_VERSION}/mediamtx_${MEDIAMTX_VERSION}_linux_amd64.tar.gz" \
    | tar -xz -C /usr/local/bin mediamtx && \
    chmod +x /usr/local/bin/mediamtx
```

Then the runtime stage copies from this.

### Stage 4: Runtime Image

```dockerfile
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && \
    mkdir -p /data

COPY --from=go-builder /rtspanda /usr/local/bin/rtspanda
COPY --from=mediamtx-downloader /usr/local/bin/mediamtx /usr/local/bin/mediamtx

ENV DATA_DIR=/data
ENV PORT=8080
ENV MEDIAMTX_BIN=/usr/local/bin/mediamtx

VOLUME /data
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/rtspanda"]
```

Why `ca-certificates`: mediamtx may need to verify TLS certificates for RTSPS (RTSP over TLS) sources. `tzdata` is needed if any timestamps need to reflect local time.

**Do not EXPOSE 8888.** The HLS port is internal only. Port 8080 is the single external interface.

### Stage 3 (Option B): Local mediamtx binary (air-gapped builds)

For environments without internet access at build time. Developer places the Linux binary at `mediamtx/mediamtx-linux-amd64` before running `docker build`.

```dockerfile
COPY mediamtx/mediamtx-linux-amd64 /usr/local/bin/mediamtx
RUN chmod +x /usr/local/bin/mediamtx
```

Add to `.gitignore`:
```
mediamtx/mediamtx-linux-amd64
```

The spec recommends Option A (download) as default. Option B is documented for special cases.

### docker-compose.yml

```yaml
services:
  rtspanda:
    build:
      context: .
      args:
        MEDIAMTX_VERSION: v1.12.3
    image: rtspanda:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - DATA_DIR=/data
      - PORT=8080
    restart: unless-stopped
```

**Volume mount:** `./data:/data` — the SQLite database and generated mediamtx config live here. The directory is created on the host automatically by Docker. SQLite writes happen to `/data/rtspanda.db` inside the container.

**Do not** use a named volume (`data:`) by default. A host-bind mount makes the database trivially accessible for backup (`cp ./data/rtspanda.db ./backup.db`).

### .dockerignore

Critical for build speed and correctness. Without it, the Docker build context includes `node_modules/`, `frontend/dist/`, and other large directories:

```
# Node
frontend/node_modules/
frontend/dist/

# Go build artifacts
backend/rtspanda
backend/data/
backend/internal/api/dist/

# Development runtime data
data/

# mediamtx binaries
mediamtx/mediamtx
mediamtx/mediamtx.exe

# Version control
.git/
.gitignore

# AI coordination (no need in image)
AI/

# Editor
.vscode/
.idea/
```

### TASK-011 Acceptance Criteria

- [ ] `docker compose build` succeeds from a clean checkout (no pre-built artifacts needed)
- [ ] `docker compose up` starts the container
- [ ] `curl http://localhost:8080/api/v1/health` returns `{"status":"ok"}`
- [ ] `http://localhost:8080` serves the React app
- [ ] `http://localhost:8080/settings` deep link works (SPA fallback)
- [ ] `docker images rtspanda --format "{{.Size}}"` — under 150MB
- [ ] Add a camera, `docker compose down`, `docker compose up` — camera still exists
- [ ] `./data/rtspanda.db` is present on the host after first run
- [ ] `docker compose logs` shows mediamtx starting
- [ ] `docker compose stop` exits cleanly within 10 seconds (no SIGKILL)
- [ ] No port 8888 exposed to the host (only 8080)

---

## Environment Variables

All configuration is via environment variables. No config file in Phase 1.

| Variable | Default (binary) | Default (Docker) | Description |
|----------|-----------------|------------------|-------------|
| `PORT` | `8080` | `8080` | HTTP listen port |
| `DATA_DIR` | `./data` | `/data` | SQLite + mediamtx.yml location |
| `MEDIAMTX_BIN` | *(not set)* | `/usr/local/bin/mediamtx` | Absolute path to mediamtx binary |

`findBinary()` resolution order (when `MEDIAMTX_BIN` not set):
1. `MEDIAMTX_BIN` env var
2. `./mediamtx/mediamtx[.exe]` relative to working directory
3. `mediamtx` on `$PATH`
4. Graceful disabled mode — streaming offline, app continues

In Docker, `MEDIAMTX_BIN=/usr/local/bin/mediamtx` is set in the image. The relative-path lookup (`./mediamtx/...`) will never find anything in a container — that's fine.

---

## Local Dev vs Production

| Aspect | Local Dev | Production (Docker) |
|--------|-----------|---------------------|
| Frontend | `npm run dev` on :5173, Vite HMR | Embedded in binary via `go:embed` |
| API proxy | Vite `server.proxy` → `:8080` | No proxy needed — same origin |
| Go server | `go run ./cmd/rtspanda` on :8080 | `./rtspanda` (compiled) |
| mediamtx | `mediamtx/mediamtx.exe` (Windows) | `/usr/local/bin/mediamtx` (Linux) |
| SQLite | `./data/rtspanda.db` | `/data/rtspanda.db` |
| HMR | Yes (Vite) | No |
| Binary size | N/A | < 50MB |
| Image size | N/A | < 150MB |

**Vite dev proxy** (already configured in `frontend/vite.config.ts`):
```ts
server: {
  proxy: {
    '/api': 'http://localhost:8080',
    '/hls': 'http://localhost:8080',
  }
}
```
This makes `npm run dev` at port 5173 forward API and HLS requests to the Go server. It is only for developer convenience and has no production effect.

---

## Known Risks

### 1. mediamtx Linux binary on Windows dev machines

**Risk:** Developer runs `docker compose build` from Windows. The `mediamtx/` directory may contain `mediamtx.exe` (the Windows dev binary) but the Docker build needs a Linux binary.

**Mitigation:** Use Option A (download in Dockerfile) — the build downloads the Linux binary from GitHub, completely independent of what's in `mediamtx/`. The Windows `.exe` is irrelevant to the Docker build.

If using Option B (local binary), the developer must explicitly download and place `mediamtx/mediamtx-linux-amd64` before running `docker build`. Document this clearly in the README.

### 2. mediamtx static linking vs Alpine musl

**Risk:** The mediamtx pre-built binary for Linux is typically compiled against glibc. Alpine uses musl libc. If the binary is dynamically linked against glibc, it will segfault on Alpine.

**Mitigation:** Check the mediamtx release page — the `linux_amd64` binary should be statically linked. If it is not, either:
- Switch the runtime image from `alpine:3.x` to `debian:bookworm-slim` (~40MB larger, but has glibc)
- Or use `alpine:3.x` with `compat-libs` (`apk add libc6-compat`)

**Test:** After building the image, run `docker compose exec rtspanda mediamtx --version`. If it outputs a version, linking is fine. If it crashes, switch to Debian slim.

### 3. `go:embed dist` failing when `dist/` is empty

**Risk:** Running `go build` locally without first running `npm run build` + copying to `backend/internal/api/dist/` will fail with `//go:embed dist: cannot find matching files`.

**Mitigation:**
- The `Makefile` build target enforces the correct order.
- Add a clear error message in the Makefile if `dist/` is missing.
- Document that `go build` directly (without `make build`) will fail — devs should always use `make`.
- Add `backend/internal/api/dist/.gitkeep` with an empty placeholder? No — `go:embed` skips hidden files (dot files). Instead, document clearly.

**Alternative:** Use a build tag (`//go:build production`) to conditionally include the embed, allowing `go build` to work without the frontend for development. This adds complexity and is optional.

### 4. SPA deep links returning 404

**Risk:** Browser navigates directly to `/settings` or `/cameras/:id`. Without the catch-all handler, Go's mux returns 404 (no route registered for those paths).

**Mitigation:** The `staticHandler()` in `static.go` must be registered as `mux.Handle("/", ...)` (catch-all) and must serve `index.html` for any path that doesn't match a real file in `dist/`.

**Verify:** After embedding, load `http://localhost:8080/settings` directly — must serve the React app, not 404.

### 5. DATA_DIR writable in container

**Risk:** Container runs as root (Docker default). `/data` is created with root ownership. If the host bind mount at `./data` has permission issues on Linux hosts, writes will fail.

**Mitigation:** In the Dockerfile, create the `/data` directory and optionally run as a non-root user. For Phase 1, running as root inside the container is acceptable. Document that on Linux hosts, `./data` will be owned by root.

### 6. mediamtx HLS port 8888 conflict

**Risk:** If another process on the host is using port 8888, mediamtx will fail to start. In Docker, this is container-internal — no conflict with the host.

**For local dev:** If port 8888 is in use on the developer's machine, mediamtx startup will fail. The Go app will log the error and enter disabled mode. The stream manager's watchdog will keep retrying.

**Mitigation for later:** Make the HLS port configurable via `MEDIAMTX_HLS_PORT` env var. Not required for Phase 1.

### 7. Go module cache in Docker build

**Risk:** Without layer caching, every `docker build` re-downloads all Go dependencies. This is slow (30–60s).

**Mitigation:** The Dockerfile separates `COPY go.mod go.sum` + `RUN go mod download` from `COPY backend/` + `RUN go build`. This way, dependency downloads are cached as long as `go.mod` and `go.sum` don't change.

### 8. `npm ci` vs `npm install` in Docker

Use `npm ci` in the Docker frontend stage:
- Installs exact versions from `package-lock.json`
- Fails if `package-lock.json` is missing or inconsistent with `package.json`
- Faster than `npm install` in CI contexts (no resolution step)

### 9. Binary size budget

Current dependencies add up approximately as follows:

| Component | Approximate Size |
|-----------|-----------------|
| Go binary (stripped, with sqlite) | ~20–30MB |
| Embedded frontend (dist/) | ~1–3MB |
| Total binary | ~22–33MB (under 50MB target) |
| mediamtx binary | ~25–35MB |
| Alpine base + ca-certs | ~10MB |
| **Estimated total image** | **~60–80MB** (well under 150MB target) |

If the binary exceeds 50MB, investigate with `go tool nm` to identify large dependencies. `modernc.org/sqlite`'s precompiled C code is the most likely cause.

---

## File Layout After TASK-010 + TASK-011

```
RTSPanda/
├── Makefile                            ← TASK-010: build orchestration
├── Dockerfile                          ← TASK-011
├── docker-compose.yml                  ← TASK-011
├── .dockerignore                       ← TASK-011
├── .gitignore                          ← (already exists, add dist/ entry)
├── backend/
│   ├── internal/
│   │   └── api/
│   │       ├── static.go              ← TASK-010: go:embed + SPA handler
│   │       └── dist/                  ← TASK-010: gitignored, build artifact
│   └── ...
├── frontend/
│   └── dist/                          ← gitignored, output of npm run build
└── data/                              ← gitignored, runtime SQLite
```

---

## Implementation Order

TASK-010 must be done before TASK-011. There is no other dependency order.

1. **TASK-010**
   - Write `backend/internal/api/static.go` with `//go:embed dist` and SPA handler
   - Update `NewRouter` to register the catch-all
   - Write `Makefile` with build/clean/dev targets
   - Add `backend/internal/api/dist/` to `.gitignore`
   - Verify: `make build` → `./backend/rtspanda` → browser can reach all pages

2. **TASK-011**
   - Write `Dockerfile` (four stages: frontend, go-builder, mediamtx-downloader, runtime)
   - Write `docker-compose.yml`
   - Write `.dockerignore`
   - Verify: `docker compose build && docker compose up` → full stack works
   - Verify: SQLite persists across `docker compose down/up`
   - Verify image size < 150MB
