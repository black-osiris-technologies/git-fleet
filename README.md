# omp-git-fleet

Safe bulk Git operations for developers who keep many local repositories.

## Status

Early development. The CLI can discover local Git repositories, report their status, plan safe sync operations, and sync clean repositories.

## Commands

```text
omp-git-fleet scan   --root <path>
omp-git-fleet status --root <path>
omp-git-fleet sync   --root <path> --target develop --dry-run
omp-git-fleet sync   --root <path> --target develop
omp-git-fleet release-pr    --root <path> --from latest-release --to master --dry-run
omp-git-fleet release-merge --root <path> --from latest-release --to master --merge-method merge --dry-run
```

## MVP Scope

- Discover Git repositories under a root directory.
- Show current branch and dirty state per repository.
- Resolve target branches such as `develop`, `master`, `main`, and `latest-release`.
- Prefer dry-run and explicit safety checks for risky operations.
- Skip dirty repositories by default.
- Sync clean repositories with `fetch --prune`, checkout/tracking branch setup, and `pull --ff-only`.
- Create and merge release pull requests from `latest-release` to `master` or `develop`.

## Safety

Start with `--dry-run` to inspect the plan. Running `sync` without `--dry-run` executes Git commands in every clean repository found under the root path.

Release merges use normal merge commits only. Squash merges are intentionally not supported.

## Development

Contributors need the Go SDK installed locally.

```text
go test ./...
go run ./cmd/omp-git-fleet scan --root .
go run ./cmd/omp-git-fleet status --root .
go run ./cmd/omp-git-fleet sync --root . --target develop --dry-run
go run ./cmd/omp-git-fleet release-pr --root . --from latest-release --to master --dry-run
```

## Non-Goals For The First Version

- Bulk commit.
- Bulk push.
- Automatic branch deletion.

## License

MIT
