#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

ENV_FILE="${ENV_FILE:-.env.production}"
if [ ! -f "${ENV_FILE}" ]; then
  ENV_FILE=".env.production.example"
fi

RUN_GO_TESTS="${RUN_GO_TESTS:-1}"
RUN_ADMIN_BUILD="${RUN_ADMIN_BUILD:-1}"
RUN_COMPOSE_CONFIG="${RUN_COMPOSE_CONFIG:-1}"
RUN_HEALTH_CHECK="${RUN_HEALTH_CHECK:-0}"
RUN_SMOKE="${RUN_SMOKE:-0}"

read_env() {
  local key="$1"
  local default_value="$2"
  if [ -f "${ENV_FILE}" ]; then
    awk -F= -v key="${key}" '
      $0 !~ /^[[:space:]]*#/ && $1 == key {
        sub(/^[^=]*=/, "")
        print
        found=1
        exit
      }
      END { if (!found) exit 1 }
    ' "${ENV_FILE}" 2>/dev/null || printf '%s' "${default_value}"
  else
    printf '%s' "${default_value}"
  fi
}

section() {
  printf '\n==> %s\n' "$1"
}

run() {
  printf '+ %s\n' "$*"
  "$@"
}

API_HOST_PORT="$(read_env API_HOST_PORT 18080)"
API_HEALTH_BASE="${API_HEALTH_BASE:-http://127.0.0.1:${API_HOST_PORT}}"
API_BASE="${API_BASE:-${API_HEALTH_BASE}/v1}"

section "release preflight"
printf 'env file: %s\n' "${ENV_FILE}"
printf 'api health base: %s\n' "${API_HEALTH_BASE}"
printf 'api smoke base: %s\n' "${API_BASE}"

if [ "${RUN_COMPOSE_CONFIG}" = "1" ]; then
  section "docker compose config"
  printf '+ docker compose config > /tmp/lehu-campus-compose.local.yml\n'
  docker compose config >/tmp/lehu-campus-compose.local.yml
  printf '+ docker compose --env-file %s -f docker-compose.yml -f docker-compose.prod.yml config > /tmp/lehu-campus-compose.prod.yml\n' "${ENV_FILE}"
  docker compose --env-file "${ENV_FILE}" -f docker-compose.yml -f docker-compose.prod.yml config >/tmp/lehu-campus-compose.prod.yml
fi

if [ "${RUN_GO_TESTS}" = "1" ]; then
  section "go tests"
  run go test ./...
fi

if [ "${RUN_ADMIN_BUILD}" = "1" ]; then
  section "admin build"
  run npm --prefix web/admin run build
fi

if [ "${RUN_HEALTH_CHECK}" = "1" ]; then
  section "running service health"
  run curl -fsS "${API_HEALTH_BASE}/healthz"
  printf '\n'
  run curl -fsS "${API_HEALTH_BASE}/readyz"
  printf '\n'
fi

if [ "${RUN_SMOKE}" = "1" ]; then
  section "smoke"
  API_BASE="${API_BASE}" run bash scripts/smoke.sh
fi

section "done"
printf 'release check passed\n'
