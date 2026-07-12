# Contributing to Git Fleet

Thank you for helping make multi-repository Git operations safer.

## Before You Start

- Search existing issues and pull requests.
- Open an issue before large behavioral or architectural changes.
- Never include credentials, private repository data, or sensitive logs.

## Development Workflow

1. Branch from the latest `develop` using `feature/<issue-id>` or `defect/<issue-id>`.
2. Keep the change focused and add tests for behavior changes.
3. Run the required checks locally.
4. Open a pull request into `develop` and complete the template.

```bash
go test ./...
go vet ./...
go build ./cmd/git-fleet
```

Release branches are promoted to `master` with normal merge commits and backmerged into `develop`. Release promotions must not be squashed.

## Engineering Expectations

- Preserve dry-run support for state-changing operations.
- Prefer explicit failure over guessing when repository state is ambiguous.
- Avoid destructive Git operations unless the behavior is guarded, documented, and thoroughly tested.
- Keep user-facing output actionable and deterministic.
- Update documentation when commands, safety behavior, or workflows change.

By participating, you agree to follow our [Code of Conduct](CODE_OF_CONDUCT.md).
