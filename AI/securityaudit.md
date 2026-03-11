# RTSPanda Security Audit

**Date:** 2025-03-09  
**Scope:** Full codebase (backend Go, frontend React/TypeScript, config, and data handling)

---

## 1. File Structure Overview

```
RTSPanda/
├── backend/                 # Go API + SQLite + mediamtx integration
│   ├── cmd/rtspanda/main.go
│   ├── internal/
│   │   ├── api/              # HTTP handlers (cameras, streams, router)
│   │   ├── cameras/          # Model, repository, service
│   │   ├── db/               # SQLite open + migrations
│   │   └── streams/          # mediamtx process manager, health, config
│   └── data/                 # Runtime DB (gitignored)
├── frontend/                 # Vite + React + TypeScript
│   └── src/                  # App, pages, components, api client
├── AI/                       # Project docs (this audit)
├── docs/
├── .gitignore
└── .claude/settings.local.json
```

---

## 2. Findings Summary

| Severity   | Count | Description |
|-----------|-------|-------------|
| High      | 2     | No authentication; YAML/config injection via camera ID/URL |
| Medium    | 4     | Error leakage; no rate limiting; no request size limit; ID format not validated |
| Low/Info  | 4     | CORS; HLS path traversal; DB file permissions; dependency hygiene |

---

## 3. High Severity

### 3.1 No Authentication or Authorization

- **Location:** All API routes in `backend/internal/api/router.go`.
- **Issue:** Every endpoint is unauthenticated: health, list/create/get/update/delete cameras, stream status. Any party that can reach the server can add/change/delete cameras and read all config and stream URLs (including RTSP URLs that may contain credentials).
- **Recommendation:** Add authentication (e.g. API key, session, or OAuth) and enforce it on all non-health endpoints. Restrict health to minimal info if exposed publicly.

### 3.2 YAML / Config Injection via Camera ID and RTSP URL

- **Location:** `backend/internal/streams/mediamtx.go` (config template and API path building).
- **Issue:** Camera `id` and `rtsp_url` are interpolated into the mediamtx config and API paths without sanitization. Go `text/template` does not escape. Example: if `id` is `"x\n  injected_path:\n    source: rtsp://evil"`, the generated YAML can add unintended paths. Similarly, `rtsp_url` can contain newlines/colons and break or extend the config.
- **Recommendation:**
  - Validate camera `id` format (e.g. UUID only) in API and service layer; reject non-matching path parameters.
  - Validate/sanitize `rtsp_url` (e.g. single line, allowed scheme `rtsp://`, no control characters, length limit). Consider allowlisting characters for mediamtx path names derived from `id`.

---

## 4. Medium Severity

### 4.1 Internal Error Messages Exposed to Client

- **Location:** `backend/internal/api/cameras.go` (and similar handlers).
- **Issue:** On non-validated errors, handlers call `writeError(w, http.StatusInternalServerError, err.Error())`, exposing internal details (e.g. SQL or file errors) to the client.
- **Recommendation:** Log the full error server-side; return a generic message to the client (e.g. "An error occurred") and use a stable error code or correlation ID for support.

### 4.2 No Rate Limiting

- **Location:** HTTP server in `backend/cmd/rtspanda/main.go` and `backend/internal/api/router.go`.
- **Issue:** No per-IP or per-key rate limiting. Enables brute-force, DoS, or abuse of camera create/update/delete and stream endpoints.
- **Recommendation:** Add rate limiting (e.g. per-IP and/or per-auth identity) for API routes, with stricter limits for mutating operations.

### 4.3 No Request Body Size Limit

- **Location:** Camera create/update handlers that decode JSON from `r.Body`.
- **Issue:** `json.NewDecoder(r.Body).Decode(...)` reads the body without a size limit. Very large payloads can exhaust memory.
- **Recommendation:** Use `http.MaxBytesReader` (or similar) to limit request body size (e.g. 64KB–256KB for camera JSON).

### 4.4 Camera ID Path Parameter Not Validated

- **Location:** All handlers that use `r.PathValue("id")` in `backend/internal/api/`.
- **Issue:** The `id` from the URL is used as-is for DB lookups, stream status, and mediamtx path names. Non-UUID values (e.g. `../`, or strings with newlines) can lead to path traversal in HLS URLs and YAML injection as in 3.2.
- **Recommendation:** Validate that `id` matches the same format used at creation (e.g. UUID). Return 400 Bad Request for invalid format before calling service/repository.

---

## 5. Low Severity / Informational

### 5.1 CORS Not Configured

- **Location:** Backend does not set CORS headers.
- **Issue:** If the frontend is served from a different origin, browsers will block API calls unless CORS is configured.
- **Recommendation:** When deploying cross-origin, add explicit CORS middleware with a restricted allowlist of origins (and methods/headers) instead of `*`.

### 5.2 HLS Reverse Proxy Path Traversal

- **Location:** `backend/internal/api/router.go` (HLS proxy to mediamtx on 127.0.0.1:8888).
- **Issue:** Requests like `/hls/camera-../../../other/index.m3u8` are forwarded to mediamtx. Impact is limited to localhost and depends on mediamtx path handling; combined with 3.2/4.4, validating ID format reduces risk.
- **Recommendation:** Validate camera ID format so that only `camera-<uuid>`-style paths are valid; optionally normalize or reject paths containing `..` in the proxy.

### 5.3 Database File Permissions

- **Location:** `backend/internal/db/db.go` (data dir `0755`, DB file uses default from `os.WriteFile` in migrations; mediamtx config in `streams/mediamtx.go` is `0644`).
- **Issue:** Data directory is world-readable; DB and config file permissions should be reviewed for multi-user or shared-host deployments.
- **Recommendation:** Restrict data directory and DB file to the process user only (e.g. `0700` for data dir, `0600` for DB and sensitive config) where the OS allows.

### 5.4 Dependency and Build Hygiene

- **Location:** `backend/go.mod`, `frontend/package.json`.
- **Issue:** No automated check for known vulnerabilities (e.g. `go vet`, `npm audit`). Frontend uses `hls.js`, React, Vite, etc.; backend uses `modernc.org/sqlite`, `github.com/google/uuid`.
- **Recommendation:** Run `go vet` and `npm audit` (or similar) in CI; update dependencies when security advisories are published.

---

## 6. Positive Observations

- **SQL:** All database access uses parameterized queries (`?` placeholders); no string concatenation into SQL. SQL injection risk is low.
- **Command execution:** mediamtx is started with `exec.CommandContext` and a fixed config file path; no user input is passed as command arguments. Command injection risk is low.
- **Frontend:** No `dangerouslySetInnerHTML`, `eval`, or raw `innerHTML` usage found; React’s default escaping is in place. No use of `process.env` or `import.meta.env` for secrets in the audited frontend code.
- **Secrets:** No hardcoded passwords or API keys in the audited code. `.gitignore` excludes `backend/data/`, `data/`, and build artifacts; `.claude/settings.local.json` contains only tool permissions, no secrets.
- **Timeouts:** HTTP server and client timeouts are set (Read/Write/Idle in main.go; HTTP client timeouts in streams package), which helps limit resource exhaustion.

---

## 7. Recommendations Priority

1. **Immediate:** Add authentication/authorization for all non-health API endpoints.
2. **Immediate:** Validate camera `id` (e.g. UUID) and sanitize/validate `rtsp_url` to prevent YAML/config injection and path abuse.
3. **Short term:** Stop returning raw `err.Error()` to clients; add request body size limits and rate limiting.
4. **Ongoing:** Harden file permissions for data directory and DB; add CORS when using a separate frontend origin; run dependency audits in CI.

---

*This audit is based on a static review of the codebase and does not include dynamic testing or deployment configuration.*
