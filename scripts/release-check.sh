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
RUN_PYTHON_TESTS="${RUN_PYTHON_TESTS:-1}"
RUN_COMPOSE_CONFIG="${RUN_COMPOSE_CONFIG:-1}"
RUN_HEALTH_CHECK="${RUN_HEALTH_CHECK:-0}"
RUN_SMOKE="${RUN_SMOKE:-0}"
PYTHON_DOCKER_IMAGE="${PYTHON_DOCKER_IMAGE:-python:3.12-slim}"
LOCAL_ENV_FILE="${LOCAL_ENV_FILE:-.env.local.example}"

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

is_example_env() {
  [ "$(basename "${ENV_FILE}")" = ".env.production.example" ]
}

validate_production_env() {
  if [ ! -f "${ENV_FILE}" ] || is_example_env; then
    return 0
  fi

  section "production env guard"
  local failed=0
  local placeholders
  placeholders="$(
    awk '
      /^[[:space:]]*#/ || /^[[:space:]]*$/ { next }
      /^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*=/ && ($0 ~ /change-me/ || $0 ~ /example\.com/) {
        print FNR ":" $0
      }
    ' "${ENV_FILE}"
  )"
  if [ -n "${placeholders}" ]; then
    echo "ERROR: ${ENV_FILE} still contains placeholder values:" >&2
    printf '%s\n' "${placeholders}" >&2
    failed=1
  fi

  local required_keys=(
    REDIS_PASSWORD
    LEHU_JWT_SECRET
    GRAFANA_ADMIN_PASSWORD
    LEHU_MYSQL_DSN
    COS_SECRET_ID
    COS_SECRET_KEY
    COS_REGION
    COS_BUCKET
    COS_PUBLIC_CDN_BASE_URL
    LEHU_CAMPUS_ADMIN_USER_IDS
    WECHAT_APP_ID
    WECHAT_APP_SECRET
    CAMPUS_AGENT_INTERNAL_TOKEN
    ADMIN_API_BASE_URL
    LEHU_PUBLIC_API_BASE_URL
    LEHU_ADMIN_ROOT_URL
    GRAFANA_ROOT_URL
    LEHU_ALERT_WEBHOOK_TOKEN
    LEHU_ALERT_FEISHU_WEBHOOK
  )
  local key value
  for key in "${required_keys[@]}"; do
    value="$(read_env "${key}" "")"
    if [ -z "${value//[[:space:]]/}" ]; then
      echo "ERROR: ${ENV_FILE} missing required ${key}" >&2
      failed=1
    fi
  done

  for key in LEHU_CAMPUS_ADMIN_ALLOW_ALL LEHU_WECHAT_MOCK_LOGIN LEHU_ENABLE_LEGACY_UPLOAD; do
    value="$(read_env "${key}" "false")"
    value="$(printf '%s' "${value}" | tr '[:upper:]' '[:lower:]')"
    if [ "${value}" = "true" ]; then
      echo "ERROR: ${key}=true is not allowed in production" >&2
      failed=1
    fi
  done

  value="$(read_env "LEHU_FEISHU_CARD_CALLBACK_ENABLED" "true")"
  value="$(printf '%s' "${value}" | tr '[:upper:]' '[:lower:]')"
  if [ "${value}" != "false" ] && [ -z "$(read_env "LEHU_FEISHU_CARD_VERIFY_TOKEN" "" | tr -d '[:space:]')" ]; then
    echo "ERROR: LEHU_FEISHU_CARD_VERIFY_TOKEN is required when Feishu card callback is enabled" >&2
    failed=1
  fi

  if [ "${failed}" -ne 0 ]; then
    exit 1
  fi
}

python_bin() {
  if [ -n "${PYTHON_BIN:-}" ]; then
    printf '%s\n' "${PYTHON_BIN}"
    return 0
  fi
  if command -v python3.12 >/dev/null 2>&1; then
    command -v python3.12
    return 0
  fi
  printf ''
}

python_minor_version() {
  "$1" - <<'PY'
import sys
print(f"{sys.version_info.major}.{sys.version_info.minor}")
PY
}

run_python_suite_docker() {
  local name="$1"
  local dir="$2"
  if ! command -v docker >/dev/null 2>&1; then
    echo "ERROR: python3.12 or Docker is required for ${name} tests" >&2
    exit 1
  fi
  printf '+ docker run --rm -v %s:/work -w /work %s sh -c python tests for %s\n' "${ROOT_DIR}" "${PYTHON_DOCKER_IMAGE}" "${name}"
  docker run --rm -v "${ROOT_DIR}:/work" -w /work "${PYTHON_DOCKER_IMAGE}" sh -c \
    "python -m pip install --disable-pip-version-check --root-user-action=ignore --no-input -r '${dir}/requirements.txt' && cd '${dir}' && python -m unittest test_main.py"
}

run_python_suite() {
  local name="$1"
  local dir="$2"
  local py
  py="$(python_bin)"
  if [ -z "${py}" ]; then
    run_python_suite_docker "${name}" "${dir}"
    return 0
  fi
  local version
  version="$(python_minor_version "${py}")"
  if [ "${version}" != "3.12" ]; then
    echo "ERROR: ${name} tests require Python 3.12, got ${version} from ${py}" >&2
    echo "Set PYTHON_BIN to a Python 3.12 executable, or unset it to use the Docker fallback." >&2
    exit 1
  fi
  local venv_dir="${TMPDIR:-/tmp}/lehu-campus-${name}-venv"
  rm -rf "${venv_dir}"
  run "${py}" -m venv "${venv_dir}"
  run "${venv_dir}/bin/python" -m pip install --disable-pip-version-check --no-input -r "${dir}/requirements.txt"
  (cd "${dir}" && run "${venv_dir}/bin/python" -m unittest test_main.py)
}

API_HOST_PORT="$(read_env API_HOST_PORT 18080)"
API_HEALTH_BASE="${API_HEALTH_BASE:-http://127.0.0.1:${API_HOST_PORT}}"
API_BASE="${API_BASE:-${API_HEALTH_BASE}/v1}"

section "release preflight"
printf 'env file: %s\n' "${ENV_FILE}"
printf 'api health base: %s\n' "${API_HEALTH_BASE}"
printf 'api smoke base: %s\n' "${API_BASE}"

validate_production_env

if [ "${RUN_COMPOSE_CONFIG}" = "1" ]; then
  section "docker compose config"
  printf '+ docker compose config > /tmp/lehu-campus-compose.local.yml\n'
  docker compose config >/tmp/lehu-campus-compose.local.yml
  if [ -f "${LOCAL_ENV_FILE}" ]; then
    printf '+ docker compose --env-file %s config > /tmp/lehu-campus-compose.local-env.yml\n' "${LOCAL_ENV_FILE}"
    docker compose --env-file "${LOCAL_ENV_FILE}" config >/tmp/lehu-campus-compose.local-env.yml
  fi
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

if [ "${RUN_PYTHON_TESTS}" = "1" ]; then
  section "python tests"
  run_python_suite "campus-rag" "campus-rag"
  run_python_suite "campus-agent" "campus-agent"
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
  run env API_BASE="${API_BASE}" bash scripts/smoke.sh
fi

section "done"
printf 'release check passed\n'
