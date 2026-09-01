# Release policy

Changes after the bootstrap commit are PR-first. The PR workflow is the
authority for the implementation; after it succeeds, the PR is merged and the
post-main workflow is required to succeed before a release is dispatched.

The release workflow accepts the exact merged SHA and the successful post-main
verification run. It refuses tag and published-release reuse. It creates an
annotated tag once, creates a draft release before assets, uses the create
response release ID immediately, and lists releases only to recover an
existing exact draft. The draft upload URL is stripped of `{?name,label}` and
must be an `uploads.github.com` URL. Source and evidence assets are uploaded
and digest-checked before the lock asset is assembled; publication happens
only after all asset checks pass.

Failed runs, tags, drafts, and releases are evidence and are never deleted,
moved, regenerated, or overwritten.
