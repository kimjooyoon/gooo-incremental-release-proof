# Incremental release proof protocol v1

The `.gooo` declaration is the source of meaning. It owns the fixed
denominator, evidence boundary, status precedence, UNKNOWN tuple, activity
mapping, reuse/reverification rule, and measurement contract. Go parses the
declaration, evaluates the fixture observations, and generates caller-owned
JSON evidence.

## Checkpoint shape

A parent checkpoint is an immutable summary of release evidence. Every release
record preserves the immutable release ID, tag object and peeled commit, the CI
run/job/artifact identity, and source/evidence asset IDs, sizes, and SHA-256
digests. The checkpoint stores a Merkle root over the ordered release records
and a lock root over the checkpoint identity, previous root, Merkle root, and
release IDs.

The next run verifies the parent checkpoint's own roots and immutable status,
then checks the root link and complete remote evidence only for new or changed
releases. The unchanged historical set is reused as a proof-carrying boundary;
a cache hit, matching hash, or prior success is never sufficient by itself.
For a 48-release parent, the fixture records 48 avoided historical queries and
one changed-release query.

## Status and scope

Known digest, tag, asset, or chain contradictions are `REFUTED`. A missing,
stale, ambiguous, or unbounded parent is `UNKNOWN`. Every UNKNOWN has the six
fields declared by `.gooo`. Status precedence is `REFUTED > UNKNOWN > CLOSED`.

The primary incremental-lock decision is separate from the
`historical_remote_survival` claim. Reusing a checkpoint does not claim that
all earlier releases still exist remotely today. When that has not been
proven, the claim remains `UNKNOWN` with a next operation to re-query every
prior release.

Speed and memory claims are `CLOSED` only for an exact integer before/after
pair whose scenario, source, contract, fixture, toolchain, and runner
identities all match. Otherwise the improvement claim is `UNKNOWN`.
