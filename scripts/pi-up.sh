#!/usr/bin/env sh
# Raspberry Pi deployment helper for RTSPanda.

set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
PI_DEPLOYMENT_MODE="${PI_DEPLOYMENT_MODE:-pi}"
APP_URL="${APP_URL:-http://127.0.0.1:8080}"
AI_WORKER_HEALTH_URL="${AI_WORKER_HEALTH_URL:-http://127.0.0.1:8090/health}"
COMPOSE_PROFILE=""
COMPOSE_SERVICES=""

info() { printf "INFO - %s\n" "$*"; }
warn() { printf "WARN - %s\n" "$*"; }
fail() {
  printf "FAIL - %s\n" "$*" >&2
  exit 1
}

select_mode() {
  case "$PI_DEPLOYMENT_MODE" in
    pi)
      COMPOSE_PROFILE="--profile pi"
      COMPOSE_SERVICES="rtspanda-pi"
      if [ -z "${AI_WORKER_URL:-}" ] && [ -z "${DETECTOR_URL:-}" ]; then
        warn "AI_WORKER_URL/DETECTOR_URL not set; Pi mode will run RTSP + UI only until a remote AI worker is configured"
      fi
      ;;
    full)
      COMPOSE_SERVICES="rtspanda ai-worker"
      ;;
    ai-worker)
      COMPOSE_PROFILE="--profile ai-worker"
      COMPOSE_SERVICES="ai-worker-standalone"
      ;;
    *)
      fail "unsupported PI_DEPLOYMENT_MODE=$PI_DEPLOYMENT_MODE (expected: pi, full, ai-worker)"
      ;;
  esac
}

run_preflight() {
  preflight="$ROOT_DIR/scripts/pi-preflight.sh"
  if [ ! -f "$preflight" ]; then
    warn "preflight script not found; continuing without it"
    return
  fi
  info "running preflight checks"
  PI_DEPLOYMENT_MODE="$PI_DEPLOYMENT_MODE" sh "$preflight" || fail "preflight failed; fix issues above and retry"
}

require_docker() {
  command -v docker >/dev/null 2>&1 || fail "docker is required"
  docker compose version >/dev/null 2>&1 || fail "docker compose plugin is required"
  docker info >/dev/null 2>&1 || fail "docker daemon is not reachable"
}

compose_up() {
  cd "$ROOT_DIR"
  info "validating docker compose configuration"
  if [ -n "$COMPOSE_PROFILE" ]; then
    # shellcheck disable=SC2086
    docker compose $COMPOSE_PROFILE config -q || fail "docker compose config validation failed"
    info "building and starting $COMPOSE_SERVICES"
    # shellcheck disable=SC2086
    docker compose $COMPOSE_PROFILE up --build -d $COMPOSE_SERVICES || fail "docker compose up failed"
  else
    docker compose config -q || fail "docker compose config validation failed"
    info "building and starting $COMPOSE_SERVICES"
    # shellcheck disable=SC2086
    docker compose up --build -d $COMPOSE_SERVICES || fail "docker compose up failed"
  fi
  docker compose ps
}

wait_for_url() {
  url="$1"
  attempts=45
  while [ "$attempts" -gt 0 ]; do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    attempts=$((attempts - 1))
    sleep 2
  done
  return 1
}

post_checks() {
  case "$PI_DEPLOYMENT_MODE" in
    ai-worker)
      info "checking AI worker health endpoint"
      if wait_for_url "$AI_WORKER_HEALTH_URL"; then
        curl -fsS "$AI_WORKER_HEALTH_URL" || true
        printf "\n"
      else
        warn "AI worker health endpoint did not become ready within timeout"
      fi
      ;;
    *)
      info "checking API health endpoints"
      if wait_for_url "$APP_URL/api/v1/health"; then
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
      ;;
  esac
}

print_next_steps() {
  case "$PI_DEPLOYMENT_MODE" in
    ai-worker)
      info "standalone AI worker deployment completed"
      printf "Health: %s\n" "$AI_WORKER_HEALTH_URL"
      printf "Logs: docker compose logs -f ai-worker-standalone\n"
      ;;
    pi)
      info "Pi lightweight deployment completed"
      printf "Open: %s\n" "$APP_URL"
      printf "Logs: docker compose logs -f rtspanda-pi\n"
      ;;
    full)
      info "full Pi deployment completed"
      printf "Open: %s\n" "$APP_URL"
      printf "Logs: docker compose logs -f rtspanda ai-worker\n"
      ;;
  esac
  printf "Stop: docker compose down\n"
}

select_mode
run_preflight
require_docker
compose_up
post_checks
print_next_steps
