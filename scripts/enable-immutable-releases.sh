#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -lt 1 || "$#" -gt 2 ]]; then
  echo "usage: enable-immutable-releases.sh OWNER/REPOSITORY [RECEIPT_PATH]" >&2
  exit 64
fi
repository=$1
receipt_path=${2:-}
settings=$(gh api -X PUT "repos/$repository/immutable-releases")
test "$(jq -r '.enabled' <<<"$settings")" = true
settings_digest="sha256:$(printf '%s' "$settings" | sha256sum | awk '{print $1}')"
receipt=$(jq -S -n --arg schema "gooo/incremental-release-proof/operator-immutable-policy-receipt/v1" --arg repository "$repository" --arg digest "$settings_digest" --argjson settings "$settings" \
  '{schema:$schema,repository:$repository,enabled:true,immutable:true,digest:$digest,operator_api_calls:1,settings:$settings}')
if [[ -n "$receipt_path" ]]; then
  mkdir -p "$(dirname "$receipt_path")"
  printf '%s\n' "$receipt" >"$receipt_path"
else
  printf '%s\n' "$receipt"
fi
