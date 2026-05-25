#!/usr/bin/env bash
# //////////////////////////////////////////////////////////
# //
# // Copyright 2026 Sudhakar Narayanamurthy. All rights reserved.
# // Licensed under the Apache License, Version 2.0 (the "License")
# //
# // Runs mongosh scripts against local or Docker MongoDB (seed, verify, etc.).
# //
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

log() { printf '==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<EOF
Usage: $(basename "$0") [options] <script.js>...

Run mongosh script(s) against the local POC MongoDB.
Scripts are resolved under ${SCRIPT_DIR}/ when given as a bare name.

Options:
  -h, --help       Show this help
  --no-wait        Skip waiting for MongoDB to accept connections
  --eval EXPR      Run a mongosh expression (may be repeated)

Environment:
  MONGODB_URI      Default: mongodb://localhost:27017
  MONGOSH_CMD      Override mongosh invocation (e.g. custom wrapper)

Examples:
  $(basename "$0") seed_test_data.js
  $(basename "$0") --eval 'db.getSiblingDB("global").stats()'

Available scripts:
EOF
  local f
  for f in "${SCRIPT_DIR}"/*.js; do
    [[ -f "$f" ]] && printf '  %s\n' "$(basename "$f")"
  done
}

load_env() {
  if [[ -f "${ROOT_DIR}/.env" ]]; then
    # shellcheck disable=SC1091
    set -a
    source "${ROOT_DIR}/.env"
    set +a
  fi
  MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27017}"
}

resolve_script() {
  local path=$1
  if [[ -f "$path" ]]; then
    printf '%s\n' "$(cd "$(dirname "$path")" && pwd)/$(basename "$path")"
    return
  fi
  if [[ -f "${SCRIPT_DIR}/${path}" ]]; then
    printf '%s\n' "${SCRIPT_DIR}/${path}"
    return
  fi
  die "script not found: $path"
}

mongosh_exec() {
  if [[ -n "${MONGOSH_CMD:-}" ]]; then
    # shellcheck disable=SC2086
    $MONGOSH_CMD "$MONGODB_URI" --quiet "$@"
    return
  fi

  if command -v mongosh >/dev/null 2>&1; then
    mongosh "$MONGODB_URI" --quiet "$@"
    return
  fi

  if docker compose -f "${ROOT_DIR}/docker-compose.yml" ps mongo 2>/dev/null \
    | grep -q 'running\|healthy'; then
    docker compose -f "${ROOT_DIR}/docker-compose.yml" exec -T mongo \
      mongosh "mongodb://127.0.0.1:27017" --quiet "$@"
    return
  fi

  die "mongosh not found and mongo container is not running; install mongosh or start compose"
}

wait_for_mongo() {
  local attempts=${1:-30}
  local i=1
  while (( i <= attempts )); do
    if mongosh_exec --eval "db.adminCommand('ping')" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
    (( i++ )) || true
  done
  die "MongoDB is not reachable at $MONGODB_URI (is the stack up?)"
}

WAIT_FOR_MONGO=1
SCRIPTS=()
EVAL_EXPRS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --no-wait)
      WAIT_FOR_MONGO=0
      shift
      ;;
    --eval)
      [[ $# -ge 2 ]] || die "--eval requires an argument"
      EVAL_EXPRS+=("$2")
      shift 2
      ;;
    --)
      shift
      SCRIPTS+=("$@")
      break
      ;;
    -*)
      die "unknown option: $1 (try --help)"
      ;;
    *)
      SCRIPTS+=("$1")
      shift
      ;;
  esac
done

if [[ ${#SCRIPTS[@]} -eq 0 && ${#EVAL_EXPRS[@]} -eq 0 ]]; then
  usage >&2
  exit 1
fi

load_env

if (( WAIT_FOR_MONGO )); then
  log "waiting for MongoDB at $MONGODB_URI"
  wait_for_mongo
fi

for script in "${SCRIPTS[@]}"; do
  resolved="$(resolve_script "$script")"
  log "running $(basename "$resolved")"
  mongosh_exec --file "$resolved"
done

for expr in "${EVAL_EXPRS[@]}"; do
  log "running --eval"
  mongosh_exec --eval "$expr"
done
