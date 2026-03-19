# Testing Strategy

## Goals

- Catch regressions early with fast, deterministic tests.
- Keep local developer feedback under 2 minutes for default test commands.
- Separate unit, integration, and smoke concerns so failures are easy to triage.
- Establish a CI path that can grow without making every PR slow.

## Guiding Principles

- Prefer pure-function and boundary tests first (lower brittleness).
- Mock external systems at unit level; use real dependencies in integration level.
- Keep smoke checks focused on service availability and one core user flow.
- Add tests with each bug fix or feature in the closest layer that can catch it.

## Layered Strategy By Subsystem

### Backend (Go)

Tooling:

- `go test` (stdlib `testing` package)
- `httptest` for HTTP handler tests
- temporary SQLite databases for repository/service integration tests

Recommended layers:

1. Unit (`backend/internal/**/_test.go`)
- Test helpers, transformations, validation, and in-memory logic.
- No filesystem/network/process dependencies.

2. Integration (`backend/internal/**/_test.go`, build tag optional)
- Exercise repository + service with temp DB and real migrations.
- Exercise API handlers with `httptest` and real JSON payloads.

3. Smoke (`scripts/api-smoke.ps1` and future shell equivalent)
- Run against a started app instance.
- Validate `/health`, camera CRUD happy path, and stream-status endpoint response shape.

### Frontend (React + Vite)

Tooling:

- Vitest (`frontend/vitest.config.ts`)
- Node environment for API/client utility tests (fast baseline)
- Later: jsdom + React Testing Library for component behavior

Recommended layers:

1. Unit (`frontend/src/**/*.test.ts`)
- API wrappers, formatters, query builders, state helpers.
- Mock `fetch`; assert request method/headers/body and error mapping.

2. Integration (`frontend/src/**/*.test.tsx`)
- Render page-level flows with mocked API via MSW.
- Cover loading, empty, error, and success states.

3. Smoke (optional CI job)
- Build app and run a minimal browser script (Playwright) against a running backend.

### AI Worker (FastAPI + ONNX)

Tooling:

- `pytest`
- monkeypatch/stubs for `onnxruntime` in unit tests
- FastAPI `TestClient` in integration tests

Recommended layers:

1. Unit (`ai_worker/tests/unit_*.py` or `ai_worker/tests/test_*.py`)
- Image preprocessing, NMS logic, response shaping, threshold handling.
- Avoid model file dependency by stubbing runtime/session objects.

2. Integration (`ai_worker/tests/integration_*.py`)
- `/health` and `/detect` request/response validation with generated images.
- Use a controlled fake inference output.

3. Smoke (compose)
- `docker compose up` then verify AI worker `/health` and backend detector health endpoint.

## Test Structure Convention

- Backend: `backend/internal/<domain>/*_test.go`
- Frontend: `frontend/src/<domain>/*.test.ts(x)`
- AI worker: `ai_worker/tests/test_*.py`

Naming:

- Use behavior-oriented names (`TestBufferKeepsLastNLines`, `test_nms_filters_overlapping_boxes`).
- Keep each test focused on one behavior.

## Staged Rollout

### Stage 0 (now)

- Add minimal runnable tests in all three subsystems.
- Add local test commands and baseline config.

### Stage 1 (next 1-2 sprints)

- Backend: add repository+service integration tests for cameras/settings/detections.
- Frontend: add integration tests for `Dashboard` and `CameraView` with mocked network.
- AI worker: add `/detect` endpoint tests for empty upload, invalid mime, valid image response shape.

### Stage 2 (stabilization)

- Add coverage reporting per subsystem and enforce floor on changed files.
- Add flaky-test guardrails (timeouts, deterministic fixtures, no live-network tests).

### Stage 3 (pre-release hardening)

- Add smoke tests in CI for compose deployment and key API contracts.
- Run nightly extended integration suite (non-blocking on regular PRs).

## CI Suggestions

Run separate jobs so failures are isolated:

1. `backend-test`
- `cd backend && go test ./...`

2. `frontend-test`
- `cd frontend && npm ci && npm run test`

3. `ai-worker-test`
- `cd ai_worker && python -m pip install -r requirements.txt -r requirements-dev.txt && pytest`

4. `smoke` (optional required on main branch)
- Start services (compose or local)
- Run health/API smoke checks

Suggested policy:

- PR required: backend/frontend/ai unit tests
- Main/nightly: integration + smoke

## Baseline Tests Added In This Iteration

- Backend: `backend/internal/logs/buffer_test.go`
- Frontend: `frontend/src/api/recordings.test.ts`, `frontend/src/api/settings.test.ts`
- AI worker: `ai_worker/tests/test_main_helpers.py`

These tests are intentionally low-coupling and fast to establish a reliable base before broader integration coverage.
