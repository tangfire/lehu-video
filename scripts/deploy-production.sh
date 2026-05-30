#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

DEPLOY_BRANCH="${DEPLOY_BRANCH:-campus-estation-cleanup}"
DEPLOY_REMOTE="${DEPLOY_REMOTE:-origin}"
ENV_FILE="${ENV_FILE:-.env.production}"

section() {
  printf '\n==> %s\n' "$1"
}

run() {
  printf '+ %s\n' "$*"
  "$@"
}

if [ ! -f "${ENV_FILE}" ]; then
  echo "missing ${ENV_FILE}; production deploy refuses to use example env" >&2
  exit 1
fi

section "fetch ${DEPLOY_REMOTE}/${DEPLOY_BRANCH}"
run git fetch "${DEPLOY_REMOTE}" "${DEPLOY_BRANCH}"
run git checkout "${DEPLOY_BRANCH}"
run git reset --hard "${DEPLOY_REMOTE}/${DEPLOY_BRANCH}"

section "pre-deploy check"
run env ENV_FILE="${ENV_FILE}" RUN_GO_TESTS=0 RUN_ADMIN_BUILD=0 RUN_PYTHON_TESTS=0 RUN_HEALTH_CHECK=0 bash scripts/release-check.sh

section "compose deploy"
run docker compose --env-file "${ENV_FILE}" -f docker-compose.yml -f docker-compose.prod.yml up -d --build

section "post-deploy health"
run env ENV_FILE="${ENV_FILE}" RUN_GO_TESTS=0 RUN_ADMIN_BUILD=0 RUN_PYTHON_TESTS=0 RUN_HEALTH_CHECK=1 bash scripts/release-check.sh

section "containers"
run docker compose --env-file "${ENV_FILE}" -f docker-compose.yml -f docker-compose.prod.yml ps

section "done"
printf 'deployed %s/%s\n' "${DEPLOY_REMOTE}" "${DEPLOY_BRANCH}"
