#!/usr/bin/env sh
# RTSPanda Android launcher (Termux, no Docker, no root).
#
# Usage:
#   ./scripts/android-up.sh
# Optional env:
#   DATA_DIR=/data/data/com.termux/files/home/.rtspanda/data
#   PORT=8080
#   RTSPANDA_BIN=./rtspanda

set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
RTSPANDA_BIN="${RTSPANDA_BIN:-$ROOT_DIR/rtspanda}"
DATA_DIR="${DATA_DIR:-$HOME/.rtspanda/data}"
PORT="${PORT:-8080}"

info() { printf "INFO  — %s\n" "$*"; }
warn() { printf "WARN  — %s\n" "$*"; }
fail() {
  printf "FAIL  — %s\n" "$*" >&2
  exit 1
}

[ -x "$RTSPANDA_BIN" ] || fail "rtspanda binary not found or not executable: $RTSPANDA_BIN"
mkdir -p "$DATA_DIR"

if command -v termux-wake-lock >/dev/null 2>&1; then
  if termux-wake-lock >/dev/null 2>&1; then
    info "termux-wake-lock enabled"
  else
    warn "termux-wake-lock command exists but failed; continuing"
  fi
else
  warn "termux-wake-lock not available; continuing without wake lock"
fi

export RTSPANDA_MODE="pi"
export DATA_DIR
export PORT

info "starting RTSPanda (mode=$RTSPANDA_MODE data_dir=$DATA_DIR port=$PORT)"
exec "$RTSPANDA_BIN"

