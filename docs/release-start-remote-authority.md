# release-start remote authority

`release-start` treats `origin` as the source of truth for release history and release-branch existence.

- Dry-run reads branches and tags directly from `origin` with `git ls-remote`; it does not mutate local refs.
- Real execution fetches with `--prune --prune-tags --tags` before creating a release.
- Tags deleted on `origin` are not considered when deriving the next release line.
- A same-named release branch that exists only locally does not block release creation and is never deleted or pushed automatically.
- The new release branch is created directly from `origin/<base>` on the remote. A create-only force-with-lease prevents a race from advancing a branch created by another actor.

This keeps stale local release state from affecting `release-start` while preserving local-only work for manual inspection or cleanup.
