#!/usr/bin/env bash
set -Eeuo pipefail

root=${1:-.}
test -f "$root/.gooo/incremental-release-proof.gooo"
test -f "$root/fixtures/corpus.json"
test "$(grep -c '^  scenario ' "$root/.gooo/incremental-release-proof.gooo")" = 9
test "$(grep -c '^  activity ' "$root/.gooo/incremental-release-proof.gooo")" = 12
test "$(jq '.checkpoints["parent-48"].releases | length' "$root/fixtures/corpus.json")" = 48
jq -e '.schema == "gooo/incremental-release-proof/corpus/v1" and (.cases|length) == 9' "$root/fixtures/corpus.json" >/dev/null
grep -q 'precedence "REFUTED" "UNKNOWN" "CLOSED"' "$root/.gooo/incremental-release-proof.gooo"
grep -q 'unknown_fields "stage" "step" "reason" "unknown_class" "next_operation" "blocked_by"' "$root/.gooo/incremental-release-proof.gooo"
grep -q 'root_readme "excluded"' "$root/.gooo/incremental-release-proof.gooo"
grep -q 'repository_writes "0"' "$root/.gooo/incremental-release-proof.gooo"
