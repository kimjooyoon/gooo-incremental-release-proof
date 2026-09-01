# gooo-incremental-release-proof

`gooo-incremental-release-proof` implements proof-preserving incremental
release locks for Gooo. It avoids re-fetching a large unchanged release set by
carrying forward an immutable checkpoint, while requiring independent proof of
the parent roots and complete remote evidence for every new or changed
release.

The authoritative contract is
[`.gooo/incremental-release-proof.gooo`](.gooo/incremental-release-proof.gooo).
Go is only the evaluator, generator, and runtime. The fixed nine-case corpus
contains three `CLOSED`, three `UNKNOWN`, and three `REFUTED` cases, with 48
historical release records in the parent checkpoint:

| Class | Cases |
| --- | --- |
| `CLOSED` | parent checkpoint proven; changed release evidence verified; deterministic replay |
| `UNKNOWN` | missing parent; stale parent; missing exact measurement pair |
| `REFUTED` | parent digest contradiction; changed asset digest mismatch; chain discontinuity |

`REFUTED > UNKNOWN > CLOSED`. A cache hit, hash match, or prior success never
auto-closes a run. The historical remote-survival claim is emitted separately
as `UNKNOWN` when the current run did not re-query every prior release.

GitHub Actions on Go 1.27 is the verification authority. Its exact artifact
contains stage wall time and peak RSS, test counts, source inventory, generated
artifact counts/bytes, checkpoint roots, activity mapping, and the zero-write
authority record. Generated output is caller-owned temporary output only; the
root README is excluded from inventory.

See [`docs/protocol-v1.md`](docs/protocol-v1.md),
[`docs/ci-evidence-v1.md`](docs/ci-evidence-v1.md), and
[`docs/release-policy.md`](docs/release-policy.md).
