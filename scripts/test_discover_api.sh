#!/usr/bin/env bash
# Smoke test GET /v1/discover on the global tier (stack must already be up).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

log() { printf '==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not installed"
}

load_env() {
  if [[ -f "${ROOT_DIR}/.env" ]]; then
    # shellcheck disable=SC1091
    set -a
    source "${ROOT_DIR}/.env"
    set +a
  fi
}

usage() {
  cat <<EOF
Usage: $(basename "$0")

Runs discover API smoke tests against http://localhost:8080/v1/discover.
Requires the POC stack to be running and seed data loaded.

Environment:
  JWT_SECRET     From ${ROOT_DIR}/.env if present (default: poc-dev-secret)
  GLOBAL_URL     Default: http://localhost:8080
EOF
}

GLOBAL_URL="${GLOBAL_URL:-http://localhost:8080}"

main() {
  case "${1:-}" in
    -h|--help)
      usage
      exit 0
      ;;
  esac

  require_cmd curl
  require_cmd go
  load_env

  local secret="${JWT_SECRET:-poc-dev-secret}"
  local status body token

  log "discover API smoke test (${GLOBAL_URL})"

  status=$(curl -s -o /dev/null -w "%{http_code}" "${GLOBAL_URL}/v1/discover")
  if [[ "$status" != "401" ]]; then
    die "discover without auth: expected 401, got ${status}"
  fi
  printf '  discover (no auth):     401 OK\n'

  token=$(go run "${ROOT_DIR}/scripts/mint_jwt.go" "$secret")
  body=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer ${token}" "${GLOBAL_URL}/v1/discover")
  status=$(tail -n1 <<<"$body")
  body=$(sed '$d' <<<"$body")
  if [[ "$status" != "200" ]]; then
    die "discover with token: expected 200, got ${status}; body: ${body}"
  fi
  if [[ "$body" != *engineering* ]] || [[ "$body" != *marketing* ]]; then
    die "discover response missing expected teams: ${body}"
  fi
  printf '  discover (usr_sudhakan): 200 OK  %s\n' "$body"

  token=$(go run "${ROOT_DIR}/scripts/mint_jwt.go" "$secret" usr_unknown)
  status=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer ${token}" "${GLOBAL_URL}/v1/discover")
  if [[ "$status" != "401" ]]; then
    die "discover (unknown user): expected 401, got ${status}"
  fi
  printf '  discover (usr_unknown): 401 OK\n'
}

main "$@"
