#!/usr/bin/env sh
# Stop RTSPanda Compose stack on Raspberry Pi.

set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

if ! command -v docker >/dev/null 2>&1; then
  printf "docker not found; nothing to stop\n"
  exit 0
fi

docker compose down
