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
```

## MVP Scope

- Discover Git repositories under a root directory.
- Show current branch and dirty state per repository.
- Resolve target branches such as `develop`, `master`, `main`, and `latest-release`.
- Prefer dry-run and explicit safety checks for risky operations.
- Skip dirty repositories by default.
- Sync clean repositories with `fetch --prune`, checkout/tracking branch setup, and `pull --ff-only`.

## Safety

Start with `--dry-run` to inspect the plan. Running `sync` without `--dry-run` executes Git commands in every clean repository found under the root path.

## Development

Contributors need the Go SDK installed locally.

```text
go test ./...
go run ./cmd/omp-git-fleet scan --root .
go run ./cmd/omp-git-fleet status --root .
go run ./cmd/omp-git-fleet sync --root . --target develop --dry-run
```

## Non-Goals For The First Version

- Bulk commit.
- Bulk push.
- Pull request creation.
- Automatic branch deletion.

## License

MIT
