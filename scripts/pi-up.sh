#!/usr/bin/env sh
# One-command Raspberry Pi deployment path for RTSPanda.

set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
APP_URL="${APP_URL:-http://127.0.0.1:8080}"

info() { printf "INFO - %s\n" "$*"; }
warn() { printf "WARN - %s\n" "$*"; }
fail() {
  printf "FAIL - %s\n" "$*" >&2
  exit 1
}

run_preflight() {
  preflight="$ROOT_DIR/scripts/pi-preflight.sh"
  if [ ! -f "$preflight" ]; then
    warn "preflight script not found; continuing without it"
    return
  fi
  info "running preflight checks"
  sh "$preflight" || fail "preflight failed; fix issues above and retry"
}

require_docker() {
  command -v docker >/dev/null 2>&1 || fail "docker is required"
  docker compose version >/dev/null 2>&1 || fail "docker compose plugin is required"
  docker info >/dev/null 2>&1 || fail "docker daemon is not reachable"
}

compose_up() {
  cd "$ROOT_DIR"
  info "validating docker compose configuration"
  docker compose config -q || fail "docker compose config validation failed"

  info "building and starting rtspanda + ai-worker"
  docker compose up --build -d || fail "docker compose up failed"
  docker compose ps
}

wait_for_api() {
  attempts=45
  while [ "$attempts" -gt 0 ]; do
    if curl -fsS "$APP_URL/api/v1/health" >/dev/null 2>&1; then
      return 0
    fi
    attempts=$((attempts - 1))
    sleep 2
  done
  return 1
}

post_checks() {
  info "checking API health endpoints"
  if wait_for_api; then
    curl -fsS "$APP_URL/api/v1/health" || true
    printf "\n"
  else
    warn "api health endpoint did not become ready within timeout"
  fi

  if ! curl -fsS "$APP_URL/api/v1/health/ready"; then
    warn "readiness endpoint is not fully healthy yet (may be normal during first boot)"
  fi
  printf "\n"

  if ! curl -fsS "$APP_URL/api/v1/detections/health"; then
    warn "detection health endpoint is degraded or unavailable"
  fi
  printf "\n"
}

print_next_steps() {
  info "RTSPanda deployment command completed"
  printf "Open: %s\n" "$APP_URL"
  printf "Logs: docker compose logs -f rtspanda ai-worker\n"
  printf "Stop: docker compose down\n"
}

run_preflight
require_docker
compose_up
post_checks
print_next_steps
