#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -ne 8 ]]; then
  echo "usage: ci-metrics.sh REPOSITORY OUTPUT CONFORMANCE COMPILE BUILD TEST CONFORMANCE_STAGE INTEGRATION" >&2
  exit 64
fi
repo_root=$1
output=$2
conformance_report=$3
compile=$4
build=$5
test_stage=$6
conformance_stage=$7
integration=$8
conformance_root=$(dirname "$conformance_report")

sum_lines() {
  local suffix=$1 total=0 file lines
  while IFS= read -r -d '' file; do
    lines=$(wc -l <"$file" | tr -d ' ')
    total=$((total + lines))
  done < <(find "$repo_root" -path "$repo_root/.git" -prune -o -type f -name "$suffix" ! -path "$repo_root/README.md" -print0)
  echo "$total"
}

count_files() {
  local suffix=$1
  find "$repo_root" -path "$repo_root/.git" -prune -o -type f -name "$suffix" ! -path "$repo_root/README.md" -print | wc -l | tr -d ' '
}

read_stage() {
  if [[ -f "$1" ]]; then
    jq -c '{wall_ms,peak_rss_kib,measurement_status}' "$1"
  else
    printf '%s\n' '{"wall_ms":null,"peak_rss_kib":null,"measurement_status":"UNKNOWN"}'
  fi
}

go_files=$(count_files '*.go')
gooo_files=$(count_files '*.gooo')
go_lines=$(sum_lines '*.go')
gooo_lines=$(sum_lines '*.gooo')
regular_files=$(find "$repo_root" -path "$repo_root/.git" -prune -o -type f ! -path "$repo_root/README.md" -print | wc -l | tr -d ' ')
descendant_dirs=$(find "$repo_root" -mindepth 1 -path "$repo_root/.git" -prune -o -type d -print | wc -l | tr -d ' ')
generated_files=$(find "$conformance_root" -type f -path '*/generated/*' -print | wc -l | tr -d ' ')
generated_bytes=$(find "$conformance_root" -type f -path '*/generated/*' -exec wc -c {} + | awk 'END {print $1 + 0}')
repository_writes=$(git -C "$repo_root" status --porcelain=v1 | wc -l | tr -d ' ')
toolchain=$(go env GOVERSION)
runner_material="${RUNNER_OS:-unknown}|${RUNNER_ARCH:-unknown}|${ImageOS:-unknown}|${ImageVersion:-unknown}"
runner_digest="sha256:$(printf '%s' "$runner_material" | sha256sum | awk '{print $1}')"

jq -n \
  --arg schema "gooo/incremental-release-proof/ci-evidence/v1" \
  --arg go_version "$toolchain" --arg runner_digest "$runner_digest" \
  --argjson go_files "$go_files" --argjson gooo_files "$gooo_files" \
  --argjson go_lines "$go_lines" --argjson gooo_lines "$gooo_lines" \
  --argjson regular_files "$regular_files" --argjson descendant_dirs "$descendant_dirs" \
  --argjson generated_files "$generated_files" --argjson generated_bytes "$generated_bytes" \
  --argjson repository_writes "$repository_writes" \
  --argjson compile "$(read_stage "$compile")" --argjson build "$(read_stage "$build")" \
  --argjson test_stage "$(read_stage "$test_stage")" --argjson conformance_stage "$(read_stage "$conformance_stage")" \
  --argjson integration "$(read_stage "$integration")" --slurpfile report "$conformance_report" \
  '{schema:$schema,verification_authority:"GITHUB_ACTIONS",go_version:$go_version,runner_digest:$runner_digest,
    root_readme_excluded:true,repository_writes:$repository_writes,
    inventory:{go_files:$go_files,gooo_files:$gooo_files,go_physical_lines:$go_lines,gooo_physical_lines:$gooo_lines,descendant_dirs:$descendant_dirs,regular_files:$regular_files},
    generated:{files:$generated_files,bytes:$generated_bytes},
    stages:{compile:$compile,build:$build,test:$test_stage,conformance:$conformance_stage,integration:$integration},
    tests:($report[0].summary|{total:.tests_total,selected:.tests_selected,executed:.tests_executed,reused:.tests_reused,failed:.tests_failed,unknown:.tests_unknown}),
    cases:($report[0].cases|map({id,decision,primary_decision,historical_remote_survival,improvement,remote_queries})),
    authority:($report[0].authority|. + {local_verification_executions:0,cross_project_required_gates:0,operator_api_calls_in_actions:0,automatic_commit:0,automatic_push:0,automatic_merge:0,automatic_release:0})
  }' >"$output"
