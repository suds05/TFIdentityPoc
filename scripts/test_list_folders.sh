#!/usr/bin/env bash
# Smoke test GET /v1/teams/{teamId}/folders (stack must already be up).
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

Runs list-folders smoke tests per README.md verification scenarios.
Requires the POC stack to be running and seed data loaded.

Environment:
  JWT_SECRET        From ${ROOT_DIR}/.env if present (default: poc-dev-secret)
  STORAGE_TIER1_URL Default: http://localhost:8081
  STORAGE_TIER2_URL Default: http://localhost:8082
EOF
}

STORAGE_TIER1_URL="${STORAGE_TIER1_URL:-http://localhost:8081}"
STORAGE_TIER2_URL="${STORAGE_TIER2_URL:-http://localhost:8082}"

expect_status() {
  local label=$1
  local want=$2
  local got=$3
  local body=${4:-}
  if [[ "$got" != "$want" ]]; then
    die "${label}: expected ${want}, got ${got}; body: ${body}"
  fi
  printf '  %s: %s OK' "$label" "$want"
  if [[ -n "$body" && "$want" == "200" ]]; then
    printf '  %s' "$body"
  fi
  echo
}

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
  local token status body

  log "list folders API smoke test"
  token=$(go run "${ROOT_DIR}/scripts/mint_jwt.go" "$secret")

  # Member, correct tier (engineering on tier 1)
  body=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer ${token}" \
    "${STORAGE_TIER1_URL}/v1/teams/engineering/folders")
  status=$(tail -n1 <<<"$body")
  body=$(sed '$d' <<<"$body")
  expect_status "tier1 engineering (member)" "200" "$status" "$body"
  if [[ "$body" != *code* ]]; then
    die "tier1 engineering: expected folder data, got: ${body}"
  fi

  # Not a member (qa — user is not on qa team)
  status=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer ${token}" \
    "${STORAGE_TIER1_URL}/v1/teams/qa/folders")
  expect_status "tier1 qa (not a member)" "403" "$status"

  # Member, wrong tier (marketing on tier 1)
  status=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer ${token}" \
    "${STORAGE_TIER1_URL}/v1/teams/marketing/folders")
  expect_status "tier1 marketing (wrong tier)" "404" "$status"

  # Member, correct tier (marketing on tier 2)
  body=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer ${token}" \
    "${STORAGE_TIER2_URL}/v1/teams/marketing/folders")
  status=$(tail -n1 <<<"$body")
  body=$(sed '$d' <<<"$body")
  expect_status "tier2 marketing (member)" "200" "$status" "$body"
  if [[ "$body" != *campaigns* ]]; then
    die "tier2 marketing: expected folder data, got: ${body}"
  fi

  # No auth
  status=$(curl -s -o /dev/null -w "%{http_code}" \
    "${STORAGE_TIER1_URL}/v1/teams/engineering/folders")
  expect_status "tier1 no auth" "401" "$status"
}

main "$@"
