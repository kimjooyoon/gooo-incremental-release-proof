# GitHub Actions evidence v1

GitHub Actions is the only verification authority. The workflow runs with Go
1.27 and emits exact evidence from the CI runner: compile, build, test,
conformance, and integration `wall_ms` and `peak_rss_kib`; test
`total/selected/executed/reused/failed/unknown`; Go and Gooo file counts and
physical lines; descendant directories; regular files; and generated artifact
count and bytes. Missing observations are `null` and carry an UNKNOWN marker;
they are never replaced with zero.

The root `README.md` is excluded from the inventory. Conformance and
integration outputs are written only beneath the caller-owned runner temp
directory, and the source checkout must retain `repository_writes: 0`.

The CI artifact is the evidence boundary used by the release workflow. The
release workflow records the verification run, job, and artifact IDs in the
published lock asset and uses only the standard `github.token`.
