# omp-git-fleet

Safe bulk Git operations for developers who keep many local repositories.

## Status

Early development. The first goal is a small CLI that can discover local Git repositories, report their status, and plan safe sync operations.

## Planned Commands

```text
omp-git-fleet scan   --root <path>
omp-git-fleet status --root <path>
omp-git-fleet sync   --root <path> --target develop --dry-run
```

## MVP Scope

- Discover Git repositories under a root directory.
- Show current branch and dirty state per repository.
- Resolve target branches such as `develop`, `master`, `main`, and later `latest-release`.
- Prefer dry-run and explicit safety checks for risky operations.
- Skip dirty repositories by default.

## Development

Contributors need the Go SDK installed locally.

```text
go test ./...
go run ./cmd/omp-git-fleet scan --root .
go run ./cmd/omp-git-fleet status --root .
```

## Non-Goals For The First Version

- Bulk commit.
- Bulk push.
- Pull request creation.
- Automatic branch deletion.

## License

MIT
