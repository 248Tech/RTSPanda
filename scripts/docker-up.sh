#!/usr/bin/env sh
set -eu
docker compose up --build -d
echo "RTSPanda is running at http://localhost:8080"
