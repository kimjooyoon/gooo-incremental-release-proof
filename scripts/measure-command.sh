#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -lt 4 ]]; then
  echo "usage: measure-command.sh OUTPUT_JSON STAGE LOG COMMAND..." >&2
  exit 64
fi

output=$1
stage=$2
log=$3
shift 3
time_output=$(mktemp)
trap 'rm -f "$time_output"' EXIT
status=0
if /usr/bin/time -f 'wall_seconds=%e\npeak_rss_kib=%M' -o "$time_output" "$@" >"$log" 2>&1; then
  status=0
else
  status=$?
fi
wall_seconds=$(awk -F= '$1 == "wall_seconds" {print $2}' "$time_output")
peak_rss_kib=$(awk -F= '$1 == "peak_rss_kib" {print $2}' "$time_output")
if [[ -n "${wall_seconds:-}" ]]; then
  wall_ms=$(awk -v seconds="$wall_seconds" 'BEGIN { printf "%d", (seconds * 1000) + 0.5 }')
else
  wall_ms=null
fi
if [[ -n "${peak_rss_kib:-}" ]]; then
  rss_json=$peak_rss_kib
else
  rss_json=null
fi
jq -n --arg stage "$stage" --argjson exit_code "$status" --argjson wall_ms "${wall_ms:-null}" --argjson peak_rss_kib "${rss_json:-null}" \
  '{stage:$stage,exit_code:$exit_code,wall_ms:$wall_ms,peak_rss_kib:$peak_rss_kib,measurement_status:(if ($wall_ms == null or $peak_rss_kib == null) then "UNKNOWN" else "CLOSED" end)}' >"$output"
cat "$log"
exit "$status"
