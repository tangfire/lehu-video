#!/usr/bin/env bash

set -o pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/find-request-logs.sh <request_id|trace_id|keyword> [since]
  scripts/find-request-logs.sh <request_id|trace_id|keyword> --since 2h --tail 50000 --context 2

Examples:
  scripts/find-request-logs.sh 1716960000000000000
  scripts/find-request-logs.sh 8f0d2c0b3e7d4c1aa2c4f2e4a0c9b123 --since 30m
  scripts/find-request-logs.sh "/v1/campus/posts" --since 2h --all-containers

Options:
  --since <duration>       Docker log time window, default: 24h
  --tail <lines>           Max lines read per container, default: 50000
  --context, -C <lines>    Lines before/after each match, default: 2
  --all-containers         Search every Docker container, not only campus/compose containers
  --stopped                Include stopped containers
  -h, --help               Show this help
EOF
}

since="${SINCE:-24h}"
tail_lines="${LOG_TAIL:-50000}"
context_lines="${LOG_CONTEXT:-2}"
all_containers=0
include_stopped=0
positionals=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --since)
      since="${2:-}"
      shift 2
      ;;
    --tail)
      tail_lines="${2:-}"
      shift 2
      ;;
    --context|-C)
      context_lines="${2:-}"
      shift 2
      ;;
    --all-containers|--all)
      all_containers=1
      shift
      ;;
    --stopped)
      include_stopped=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      positionals+=("$1")
      shift
      ;;
  esac
done

query="${positionals[0]:-}"
if [[ ${#positionals[@]} -ge 2 ]]; then
  since="${positionals[1]}"
fi

if [[ -z "$query" ]]; then
  usage
  exit 1
fi

if [[ -z "$since" ]]; then
  echo "ERROR: --since cannot be empty" >&2
  exit 1
fi

if ! [[ "$tail_lines" =~ ^[0-9]+$ ]]; then
  echo "ERROR: --tail must be a number" >&2
  exit 1
fi

if ! [[ "$context_lines" =~ ^[0-9]+$ ]]; then
  echo "ERROR: --context must be a number" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker command not found" >&2
  exit 1
fi

docker_container_names() {
  local docker_ps_args=(ps --format '{{.Names}}')
  if [[ "$include_stopped" -eq 1 ]]; then
    docker_ps_args=(ps -a --format '{{.Names}}')
  fi

  if [[ "$all_containers" -eq 1 ]]; then
    docker "${docker_ps_args[@]}"
    return
  fi

  {
    if [[ "$include_stopped" -eq 1 ]]; then
      docker compose ps -a -q 2>/dev/null
    else
      docker compose ps -q 2>/dev/null
    fi | while IFS= read -r container_id; do
      [[ -n "$container_id" ]] || continue
      docker inspect --format '{{.Name}}' "$container_id" 2>/dev/null | sed 's#^/##'
    done

    docker "${docker_ps_args[@]}" 2>/dev/null | grep -E '^(campus-|lehu-)' || true
  } | awk 'NF && !seen[$0]++'
}

containers=()
while IFS= read -r container_name; do
  [[ -n "$container_name" ]] || continue
  containers+=("$container_name")
done < <(docker_container_names)

if [[ ${#containers[@]} -eq 0 ]]; then
  echo "No campus/compose containers found. Try --all-containers if the service name is not prefixed with campus-."
  exit 1
fi

echo "Searching ${#containers[@]} container(s) for: ${query}"
echo "Window: since=${since}, tail=${tail_lines}, context=${context_lines}"
echo

matches=0
all_matches_file="$(mktemp)"
trap 'rm -f "$all_matches_file" "$container_matches_file"' EXIT

grep_args=(-F -C "$context_lines" -e "$query")
if [[ "$query" =~ ^status[=:][[:space:]]*([0-9]{3})$ ]]; then
  status_code="${BASH_REMATCH[1]}"
  grep_args+=(-e "\"status\":${status_code}" -e "\"status\": ${status_code}")
fi

for container in "${containers[@]}"; do
  container_matches_file="$(mktemp)"
  if docker logs --since "$since" --tail "$tail_lines" "$container" 2>&1 | grep "${grep_args[@]}" >"$container_matches_file"; then
    matches=$((matches + 1))
    {
      printf '========== %s ==========\n' "$container"
      cat "$container_matches_file"
      printf '\n'
    } | tee -a "$all_matches_file"
  fi
  rm -f "$container_matches_file"
done

if [[ "$matches" -eq 0 ]]; then
  echo "No matches found."
  echo "Tip: request_id usually appears in campus-api. For downstream base/core/RAG logs, search the trace_id from the matched API log line."
  exit 1
fi

trace_ids="$(
  {
    grep -Eo '"trace(_id|\.id)"[[:space:]]*:[[:space:]]*"[0-9a-fA-F]{16,32}"' "$all_matches_file" \
      | sed -E 's/.*"([0-9a-fA-F]{16,32})"/\1/'
    grep -Eo 'trace(_id|\.id)=["]?[0-9a-fA-F]{16,32}["]?' "$all_matches_file" \
      | sed -E 's/.*=["]?([0-9a-fA-F]{16,32})["]?/\1/'
  } | awk 'NF && !seen[$0]++'
)"

if [[ -n "$trace_ids" ]]; then
  echo "Detected trace_id value(s) you can search across service containers:"
  while IFS= read -r trace_id; do
    [[ -n "$trace_id" ]] || continue
    echo "  make logs-trace TID=${trace_id} SINCE=${since}"
  done <<<"$trace_ids"
fi
