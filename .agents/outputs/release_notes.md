# RTSPanda v0.0.7 Release Notes

## Highlights
- Added token-based authentication across backend and frontend, including login/session handling and protected API routes.
- Added initial automated test foundation for backend, frontend, and AI worker.
- Improved Raspberry Pi deployment with architecture-aware container builds and helper scripts.
- Resolved streaming configuration drift and documented latency/stability tuning controls.
- Expanded AI worker runtime safeguards for Pi CPU-only environments.

## Included Areas
- Security/auth: backend/internal/auth/*, backend/internal/api/router.go, frontend/src/auth/*
- Streaming/runtime: backend/internal/streams/mediamtx.go, mediamtx/mediamtx.yml.tmpl, docker-compose.yml
- Deployment: Dockerfile, ai_worker/Dockerfile, scripts/pi-*.sh
- AI runtime: ai_worker/app/main.py, ai_worker/requirements.txt
- Testing/docs: docs/testing-strategy.md, frontend/vitest.config.ts, ai_worker/tests/*

## Validation Checklist
- docker compose config -q
- cd backend && go test ./internal/...
- cd frontend && npm run test -- --config vitest.config.ts
- cd ai_worker && python -m pytest -q

## Upgrade Notes
- If authentication is enabled, define AUTH_TOKEN before startup.
- For Raspberry Pi, prefer 64-bit OS and use ./scripts/pi-up.sh for first deploy.
