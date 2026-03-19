#!/usr/bin/env sh
# RTSPanda Raspberry Pi preflight checks.
# Safe and non-destructive: this script only reads local state and prints guidance.

set -u

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
FAIL_COUNT=0
WARN_COUNT=0

pass() { printf "OK   - %s\n" "$*"; }
warn() {
  printf "WARN - %s\n" "$*"
  WARN_COUNT=$((WARN_COUNT + 1))
}
fail() {
  printf "FAIL - %s\n" "$*"
  FAIL_COUNT=$((FAIL_COUNT + 1))
}

have_cmd() { command -v "$1" >/dev/null 2>&1; }

check_cmd_required() {
  cmd="$1"
  help="$2"
  if have_cmd "$cmd"; then
    pass "found '$cmd' ($(command -v "$cmd"))"
  else
    fail "missing '$cmd' ($help)"
  fi
}

check_cmd_optional() {
  cmd="$1"
  help="$2"
  if have_cmd "$cmd"; then
    pass "found '$cmd' ($(command -v "$cmd"))"
  else
    warn "missing '$cmd' ($help)"
  fi
}

check_go_version() {
  if ! have_cmd go; then
    return
  fi

  go_ver="$(go env GOVERSION 2>/dev/null || go version | awk '{print $3}')"
  major="$(printf "%s" "$go_ver" | sed -E 's/^go([0-9]+)\..*$/\1/')"
  minor="$(printf "%s" "$go_ver" | sed -E 's/^go[0-9]+\.([0-9]+).*/\1/')"

  if [ -z "$major" ] || [ -z "$minor" ]; then
    warn "could not parse Go version '$go_ver' (need Go >= 1.26 for this repo)"
    return
  fi

  if [ "$major" -gt 1 ] || { [ "$major" -eq 1 ] && [ "$minor" -ge 26 ]; }; then
    pass "Go version is compatible ($go_ver)"
  else
    fail "Go version is too old ($go_ver). Install Go >= 1.26."
  fi
}

check_node_version() {
  if ! have_cmd node; then
    return
  fi

  node_ver="$(node -v 2>/dev/null || true)"
  major="$(printf "%s" "$node_ver" | sed -E 's/^v([0-9]+).*/\1/')"
  if [ -z "$major" ]; then
    warn "could not parse Node.js version '$node_ver' (need Node >= 18)"
    return
  fi

  if [ "$major" -ge 18 ]; then
    pass "Node.js version is compatible ($node_ver)"
  else
    fail "Node.js version is too old ($node_ver). Install Node.js >= 18."
  fi
}

check_architecture() {
  os_name="$(uname -s 2>/dev/null || echo unknown)"
  arch_name="$(uname -m 2>/dev/null || echo unknown)"
  bits="$(getconf LONG_BIT 2>/dev/null || echo unknown)"
  pass "host detected: ${os_name}/${arch_name} (${bits}-bit userspace)"

  if [ "$os_name" != "Linux" ]; then
    warn "this script is intended for Raspberry Pi Linux hosts"
  fi

  case "$arch_name" in
    aarch64|arm64)
      pass "ARM64 detected (recommended for RTSPanda + AI worker)"
      ;;
    armv7l|armv6l|armhf)
      warn "32-bit ARM detected (${arch_name}); AI worker (onnxruntime) is likely unavailable on armv7/armv6"
      ;;
    *)
      warn "non-Pi architecture detected (${arch_name}); Pi-specific caveats may not apply"
      ;;
  esac
}

check_mediamtx() {
  local_bin="$ROOT_DIR/mediamtx/mediamtx"
  if [ -x "$local_bin" ]; then
    pass "mediamtx binary found at '$local_bin'"
    return
  fi

  if have_cmd mediamtx; then
    pass "mediamtx found on PATH ($(command -v mediamtx))"
    return
  fi

  warn "mediamtx binary not found; app will run but camera streams stay offline until mediamtx is installed"
}

check_docker() {
  if ! have_cmd docker; then
    warn "docker not found (native Pi run is still supported)"
    return
  fi

  pass "docker found ($(command -v docker))"

  if docker info >/dev/null 2>&1; then
    pass "docker daemon is reachable"
  else
    warn "docker daemon is not reachable (start docker or fix user permissions)"
  fi

  if docker compose version >/dev/null 2>&1; then
    pass "docker compose plugin is available"
  else
    warn "docker compose plugin missing (install 'docker-compose-plugin')"
  fi

  if have_cmd systemctl; then
    docker_state="$(systemctl is-active docker 2>/dev/null || true)"
    if [ "$docker_state" = "active" ]; then
      pass "docker service is active"
    else
      warn "docker service state: ${docker_state:-unknown}"
    fi
  fi
}

check_repo_container_support() {
  dockerfile="$ROOT_DIR/Dockerfile"
  composefile="$ROOT_DIR/docker-compose.yml"

  if [ ! -f "$dockerfile" ]; then
    warn "Dockerfile not found at repo root"
  else
    if grep -q "GOARCH=amd64" "$dockerfile" || grep -q "linux_amd64" "$dockerfile"; then
      warn "Dockerfile still contains amd64 pinning; Pi builds may fail without further changes"
    else
      pass "Dockerfile does not appear amd64-pinned"
    fi

    if grep -q "TARGETARCH" "$dockerfile"; then
      pass "Dockerfile includes target-arch aware build logic"
    else
      warn "Dockerfile does not appear to use target-arch args (TARGETARCH)"
    fi
  fi

  if [ -f "$composefile" ]; then
    if grep -q "ai-worker:" "$composefile" && grep -q "rtspanda:" "$composefile"; then
      pass "docker-compose.yml includes rtspanda + ai-worker services"
    else
      warn "docker-compose.yml is missing expected services (rtspanda and/or ai-worker)"
    fi
  else
    fail "docker-compose.yml not found at repo root"
  fi

  if have_cmd docker && docker compose version >/dev/null 2>&1; then
    if (cd "$ROOT_DIR" && docker compose config -q >/dev/null 2>&1); then
      pass "docker compose config validates"
    else
      warn "docker compose config check failed (inspect docker-compose.yml and .env values)"
    fi
  fi
}

check_pi_scripts() {
  if [ -f "$ROOT_DIR/scripts/pi-up.sh" ]; then
    pass "pi-up helper script exists"
  else
    warn "scripts/pi-up.sh missing (one-command Pi startup helper)"
  fi
}

check_data_dir() {
  if [ -d "$ROOT_DIR/data" ]; then
    if [ -w "$ROOT_DIR/data" ]; then
      pass "data directory is writable ($ROOT_DIR/data)"
    else
      fail "data directory exists but is not writable ($ROOT_DIR/data)"
    fi
  else
    warn "data directory does not exist yet (it will be created by runtime)"
  fi
}

printf "RTSPanda Pi preflight\n"
printf "repo: %s\n\n" "$ROOT_DIR"

check_architecture
check_cmd_required git "install git first"
check_cmd_required curl "install curl first"
check_cmd_required make "install build-essential/make first"
check_cmd_required go "install Go >= 1.26"
check_cmd_required node "install Node.js >= 18"
check_cmd_required npm "install npm with Node.js"
check_cmd_optional ffmpeg "install ffmpeg for frame capture/detections"
check_go_version
check_node_version
check_mediamtx
check_docker
check_repo_container_support
check_pi_scripts
check_data_dir

printf "\n"
if [ "$FAIL_COUNT" -eq 0 ]; then
  printf "Preflight result: PASS (%s warning(s)).\n" "$WARN_COUNT"
else
  printf "Preflight result: FAIL (%s failure(s), %s warning(s)).\n" "$FAIL_COUNT" "$WARN_COUNT"
fi

printf "Next: follow docs/raspberry-pi-deployment.md\n"

if [ "$FAIL_COUNT" -ne 0 ]; then
  exit 1
fi
