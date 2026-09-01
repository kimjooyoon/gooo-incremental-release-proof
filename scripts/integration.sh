#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -ne 3 ]]; then
  echo "usage: integration.sh REPOSITORY CONFORMANCE_OUTPUT OUTPUT_ROOT" >&2
  exit 64
fi
repository=$1
conformance=$2
output_root=$3
mkdir -p "$output_root"
count=0
for artifact in "$conformance"/*/generated/release-lock.json; do
  [[ -f "$artifact" ]] || continue
  case_id=$(basename "$(dirname "$(dirname "$artifact")")")
  jq -e --arg id "$case_id" '
    .schema == "gooo/incremental-release-proof/generated-lock/v1" and
    .scenario == $id and
    (.decision == "CLOSED" or .decision == "UNKNOWN" or .decision == "REFUTED") and
    .current.lock_root_digest != ""
  ' "$artifact" >/dev/null
  jq -n --arg id "$case_id" --arg digest "$(jq -r '.current.lock_root_digest' "$artifact")" \
    '{case_id:$id,status:"CLOSED",lock_root_digest:$digest}' >"$output_root/$case_id.json"
  count=$((count + 1))
done
test "$count" = 9
jq -n --argjson cases "$count" --arg repository "$repository" \
  '{schema:"gooo/incremental-release-proof/integration/v1",repository:$repository,generated_cases:$cases,decision:"CLOSED"}' >"$output_root/integration.json"
