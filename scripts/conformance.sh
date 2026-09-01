#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -ne 3 ]]; then
  echo "usage: conformance.sh REPOSITORY BINARY OUTPUT" >&2
  exit 64
fi
repository=$1
binary=$2
output=$3
before=$(git -C "$repository" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
mkdir -p "$output"
toolchain="$(go env GOVERSION)/$(go env GOOS)/$(go env GOARCH)"
runner_material="${RUNNER_OS:-unknown}|${RUNNER_ARCH:-unknown}|${ImageOS:-unknown}|${ImageVersion:-unknown}"
runner_digest="sha256:$(printf '%s' "$runner_material" | sha256sum | awk '{print $1}')"

"$binary" conformance \
  --meta "$repository/.gooo/incremental-release-proof.gooo" \
  --corpus "$repository/fixtures/corpus.json" \
  --root "$repository" \
  --out "$output/cases" \
  --toolchain "$toolchain" \
  --runner-digest "$runner_digest"

jq -e '
  .schema == "gooo/incremental-release-proof/conformance/v1" and
  .decision == "CLOSED" and
  .summary == {total_cases:9,closed:3,unknown:3,refuted:3,tests_total:9,tests_selected:9,tests_executed:7,tests_reused:2,tests_failed:3,tests_unknown:3} and
  .authority.repository_writes == 0 and
  .authority.output_scope == "CALLER_OWNED_TEMP_OUTPUT_ONLY" and
  ([.cases[] | select(.id == "parent-checkpoint-proven") | .decision] == ["CLOSED"]) and
  ([.cases[] | select(.id == "changed-release-evidence-verified") | .decision] == ["CLOSED"]) and
  ([.cases[] | select(.id == "deterministic-replay") | .decision] == ["CLOSED"]) and
  ([.cases[] | select(.id == "parent-checkpoint-missing") | .decision] == ["UNKNOWN"]) and
  ([.cases[] | select(.id == "parent-checkpoint-stale") | .decision] == ["UNKNOWN"]) and
  ([.cases[] | select(.id == "improvement-pair-missing") | .decision] == ["UNKNOWN"]) and
  ([.cases[] | select(.id == "parent-digest-contradiction") | .decision] == ["REFUTED"]) and
  ([.cases[] | select(.id == "changed-asset-digest-mismatch") | .decision] == ["REFUTED"]) and
  ([.cases[] | select(.id == "chain-discontinuity") | .decision] == ["REFUTED"]) and
  ([.cases[] | select(.id == "parent-checkpoint-proven") | .remote_queries.historical_queries_avoided] == [48]) and
  ([.cases[] | .unknowns[] | (.stage != "" and .step != "" and .reason != "" and .unknown_class != "" and .next_operation != "" and (.blocked_by|length) > 0)] | all) and
  ([.cases[] | select(.decision == "CLOSED") | .historical_remote_survival.status] | all(. == "UNKNOWN")) and
  ([.cases[] | select(.improvement.status == "CLOSED") | .improvement.exact_pair] | all) and
  ([.. | objects | keys[]? | select(test("score|percentage|average|estimate"; "i"))] | length) == 0
' "$output/cases/conformance.json" >/dev/null

after=$(git -C "$repository" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"
