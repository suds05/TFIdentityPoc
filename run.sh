#!/usr/bin/env bash
################################################################
# 
# Copyright 2026 Sudhakar Narayanamurthy. All rights reserved.
# Licensed under the Apache License, Version 2.0 (the "License")
# 
# The 'Do everything' script.
# 1. Builds and starts the POC Docker Compose stack.
# 2. Seeds MongoDB with test data.
# 3. Runs API smoke tests for the discover and list folders APIs.
#
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

log() { printf '==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not installed"
}

wait_for_url() {
  local url=$1
  local name=$2
  local attempts=${3:-60}
  local i=1
  while (( i <= attempts )); do
    if curl -sf "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
    (( i++ )) || true
  done
  die "timed out waiting for $name at $url"
}

require_cmd docker
require_cmd curl

if ! docker compose version >/dev/null 2>&1; then
  die "docker compose is required (Docker Compose V2 plugin)"
fi

if [[ ! -f .env ]]; then
  log "creating .env from .env.example"
  cp .env.example .env
fi

# shellcheck disable=SC1091
set -a
source .env
set +a

log "building images"
docker compose build

log "starting services"
if docker compose up -d --wait 2>/dev/null; then
  :
else
  # Older compose without --wait: start then poll health endpoints
  docker compose up -d
  wait_for_url "http://localhost:8080/health" "global tier"
  wait_for_url "http://localhost:8081/health" "storage tier 1"
  wait_for_url "http://localhost:8082/health" "storage tier 2"
fi

if [[ -x "${ROOT_DIR}/scripts/run_mongosh.sh" ]]; then
  log "seeding and verifying test data"
  "${ROOT_DIR}/scripts/run_mongosh.sh" seed_test_data.js verify_test_data.js
fi

log "health checks"
printf '  global:          '
curl -sf "http://localhost:8080/health"
echo
printf '  storage tier 1:  '
curl -sf "http://localhost:8081/health"
echo
printf '  storage tier 2:  '
curl -sf "http://localhost:8082/health"
echo
log "stack is up"
echo

log "testing discover API"
"${ROOT_DIR}/scripts/test_discover_api.sh"
echo

log "testing list folders API"
"${ROOT_DIR}/scripts/test_list_folders.sh"
echo

log "All good!"
