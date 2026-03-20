#!/usr/bin/env sh
# RTSPanda — Raspberry Pi deployment helper.
#
# Supported modes (PI_DEPLOYMENT_MODE):
#
#   pi          Viewer + stream relay + snapshot AI. No local AI worker.
#               This is the ONLY valid mode for Raspberry Pi hardware.
#
#   ai-worker   Standalone AI inference worker on a SECOND (server-class) machine.
#               Never run this on a Pi — it will exhaust RAM.
#
# Raspberry Pi does NOT support real-time YOLO inference.
# Do not attempt to run the ai-worker service on Pi hardware.
# Use snapshot AI (SNAPSHOT_AI_ENABLED=true) for cloud-based alert interpretation.
#
# Usage:
#   ./scripts/pi-up.sh                                  # Pi viewer only
#   SNAPSHOT_AI_ENABLED=true \
#   SNAPSHOT_AI_PROVIDER=claude \
#   SNAPSHOT_AI_API_KEY=sk-ant-... \
#   ./scripts/pi-up.sh                                  # Pi + snapshot AI alerts
#
# To connect Pi to a remote YOLO server (standard mode on server required):
#   AI_WORKER_URL=http://192.168.1.50:8090 ./scripts/pi-up.sh

set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
PI_DEPLOYMENT_MODE="${PI_DEPLOYMENT_MODE:-pi}"
APP_URL="${APP_URL:-http://127.0.0.1:8080}"
AI_WORKER_HEALTH_URL="${AI_WORKER_HEALTH_URL:-http://127.0.0.1:8090/health}"
COMPOSE_FILES="-f docker-compose.yml"
COMPOSE_PROFILE=""
COMPOSE_SERVICES=""

info() { printf "INFO  — %s\n" "$*"; }
warn() { printf "WARN  — %s\n" "$*"; }
fail() {
  printf "FAIL  — %s\n" "$*" >&2
  exit 1
}

# Refuse to run the AI worker on ARM to prevent RAM exhaustion.
check_ai_worker_on_arm() {
  arch="$(uname -m 2>/dev/null || true)"
  case "$arch" in
    arm*|aarch64)
      printf "\n"
      printf "════════════════════════════════════════════════════════════\n"
      printf "  BLOCKED: AI worker cannot run on Raspberry Pi / ARM.\n"
      printf "\n"
      printf "  Raspberry Pi does NOT support real-time YOLO inference.\n"
      printf "  Running the ai-worker service on Pi will exhaust RAM and\n"
      printf "  cause thermal throttling. It is not a supported path.\n"
      printf "\n"
      printf "  ✓ Pi mode (viewer + snapshot AI) is the correct choice.\n"
      printf "  ✓ Run the AI worker on a dedicated x86/GPU server.\n"
      printf "  ✓ Point Pi at it via: AI_WORKER_URL=http://<server>:8090\n"
      printf "════════════════════════════════════════════════════════════\n"
      printf "\n"
      fail "ai-worker mode is blocked on ARM hardware"
      ;;
  esac
}

select_mode() {
  case "$PI_DEPLOYMENT_MODE" in
    pi)
      COMPOSE_FILES="$COMPOSE_FILES -f docker-compose.standalone.yml"
      COMPOSE_PROFILE="--profile pi"
      COMPOSE_SERVICES="rtspanda-pi"
      printf "\n"
      info "Pi mode selected: viewer + stream relay + snapshot AI (no local YOLO)"
      if [ -n "${SNAPSHOT_AI_ENABLED:-}" ] && [ "$SNAPSHOT_AI_ENABLED" = "true" ]; then
        info "Snapshot AI enabled (provider: ${SNAPSHOT_AI_PROVIDER:-claude})"
        info "Alert threshold: ${SNAPSHOT_AI_THRESHOLD:-medium}"
      else
        info "Snapshot AI disabled. Set SNAPSHOT_AI_ENABLED=true to enable cloud-based alerts."
      fi
      if [ -n "${AI_WORKER_URL:-}" ]; then
        info "Remote YOLO worker configured: $AI_WORKER_URL"
        info "Note: Pi will forward detection jobs to the remote server."
      fi
      printf "\n"
      ;;
    ai-worker)
      check_ai_worker_on_arm
      COMPOSE_FILES="$COMPOSE_FILES -f docker-compose.standalone.yml"
      COMPOSE_PROFILE="--profile ai-worker"
      COMPOSE_SERVICES="ai-worker-standalone"
      info "Standalone AI worker mode (must run on a server-class machine)"
      ;;
    full)
      # Explicitly blocked. This mode attempted to run AI locally on Pi.
      printf "\n"
      printf "════════════════════════════════════════════════════════════\n"
      printf "  BLOCKED: 'full' mode has been removed.\n"
      printf "\n"
      printf "  Running the YOLO AI worker locally on Raspberry Pi is NOT\n"
      printf "  a supported deployment path. It exhausts RAM (512 MB+ for\n"
      printf "  ONNX inference alone) and causes thermal throttling.\n"
      printf "\n"
      printf "  Correct paths:\n"
      printf "    Pi viewer only:  PI_DEPLOYMENT_MODE=pi\n"
      printf "    Pi + cloud AI:   SNAPSHOT_AI_ENABLED=true PI_DEPLOYMENT_MODE=pi\n"
      printf "    Pi + remote AI:  AI_WORKER_URL=http://<server>:8090 PI_DEPLOYMENT_MODE=pi\n"
      printf "    Full server:     docker compose up --build -d (on a server)\n"
      printf "════════════════════════════════════════════════════════════\n"
      printf "\n"
      fail "unsupported PI_DEPLOYMENT_MODE=full (see above)"
      ;;
    *)
      fail "unknown PI_DEPLOYMENT_MODE=$PI_DEPLOYMENT_MODE (valid: pi, ai-worker)"
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
  # shellcheck disable=SC2086
  docker compose $COMPOSE_FILES $COMPOSE_PROFILE config -q || fail "docker compose config validation failed"

  info "building $COMPOSE_SERVICES"
  # shellcheck disable=SC2086
  docker compose $COMPOSE_FILES $COMPOSE_PROFILE build $COMPOSE_SERVICES || fail "docker compose build failed"

  info "starting $COMPOSE_SERVICES"
  # shellcheck disable=SC2086
  docker compose $COMPOSE_FILES $COMPOSE_PROFILE up -d --no-build $COMPOSE_SERVICES || fail "docker compose up failed"

  # shellcheck disable=SC2086
  docker compose $COMPOSE_FILES ps
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
        warn "API health endpoint did not become ready within timeout"
      fi

      if ! curl -fsS "$APP_URL/api/v1/health/ready" >/dev/null 2>&1; then
        warn "readiness endpoint not yet healthy (normal on first boot)"
      fi
      printf "\n"

      # Detection health on Pi will show degraded unless a remote worker is set.
      # This is expected and correct behavior.
      curl -fsS "$APP_URL/api/v1/detections/health" 2>/dev/null || true
      printf "\n"
      ;;
  esac
}

print_next_steps() {
  case "$PI_DEPLOYMENT_MODE" in
    ai-worker)
      info "Standalone AI worker deployment complete"
      printf "Health:   %s\n" "$AI_WORKER_HEALTH_URL"
      printf "Logs:     docker compose %s logs -f ai-worker-standalone\n" "$COMPOSE_FILES"
      ;;
    pi)
      printf "\n"
      info "Pi deployment complete"
      printf "Open:     %s\n" "$APP_URL"
      printf "Logs:     docker compose %s logs -f rtspanda-pi\n" "$COMPOSE_FILES"
      printf "Stop:     docker compose %s down\n" "$COMPOSE_FILES"
      printf "\n"
      printf "AI options:\n"
      printf "  Snapshot AI (cloud):  SNAPSHOT_AI_ENABLED=true SNAPSHOT_AI_PROVIDER=claude SNAPSHOT_AI_API_KEY=... \\\n"
      printf "                        %s/scripts/pi-up.sh\n" "$ROOT_DIR"
      printf "  Remote YOLO worker:   AI_WORKER_URL=http://<server>:8090 \\\n"
      printf "                        %s/scripts/pi-up.sh\n" "$ROOT_DIR"
      printf "\n"
      printf "Note: Real-time YOLO inference is NOT available on Raspberry Pi.\n"
      ;;
  esac
}

select_mode
run_preflight
require_docker
compose_up
post_checks
print_next_steps
