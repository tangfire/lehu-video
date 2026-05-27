#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:18080/v1}"
MOBILE="${SMOKE_MOBILE:-1390000$(date +%H%M%S)}"
EMAIL="${SMOKE_EMAIL:-smoke$(date +%H%M%S)@lehu.local}"
PASSWORD="${SMOKE_PASSWORD:-12345678}"

echo "Smoke target: ${API_BASE}"

json_post() {
  local path="$1"
  local body="$2"
  curl -sS -X POST "${API_BASE}${path}" \
    -H 'Content-Type: application/json' \
    ${TOKEN:+-H "Authorization: Bearer ${TOKEN}"} \
    -d "${body}"
}

json_get() {
  local path="$1"
  curl -sS "${API_BASE}${path}" \
    ${TOKEN:+-H "Authorization: Bearer ${TOKEN}"}
}

code_resp="$(json_post /user/code '{}')"
code_id="$(printf '%s' "${code_resp}" | sed -n 's/.*"code_id":"\{0,1\}\([0-9]*\)"\{0,1\}.*/\1/p')"
if [ -z "${code_id}" ]; then
  echo "Failed to get verification code: ${code_resp}" >&2
  exit 1
fi

register_resp="$(json_post /user/register "{\"mobile\":\"${MOBILE}\",\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\",\"code_id\":\"${code_id}\",\"code\":\"888888\"}")"
if ! printf '%s' "${register_resp}" | grep -q '"code":0'; then
  echo "Register failed: ${register_resp}" >&2
  exit 1
fi

login_resp="$(json_post /user/login "{\"mobile\":\"${MOBILE}\",\"password\":\"${PASSWORD}\"}")"
TOKEN="$(printf '%s' "${login_resp}" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"
if [ -z "${TOKEN}" ]; then
  echo "Login failed: ${login_resp}" >&2
  exit 1
fi

feed_resp="$(json_post /video/feed '{"feed_num":"5","feed_type":1}')"
if ! printf '%s' "${feed_resp}" | grep -q '"code":0'; then
  echo "Feed failed: ${feed_resp}" >&2
  exit 1
fi

echo "Smoke passed: registered ${MOBILE}, login token received, feed reachable."
