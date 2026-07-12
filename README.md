# Git Fleet

> Safe, repeatable Git operations across every repository in your workspace.

[![CI](https://github.com/black-osiris-technologies/git-fleet/actions/workflows/ci.yml/badge.svg)](https://github.com/black-osiris-technologies/git-fleet/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/black-osiris-technologies/git-fleet)](https://github.com/black-osiris-technologies/git-fleet/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](go.mod)

Git Fleet scans a directory of local repositories, explains what it would change, and performs guarded sync or release operations only where it is safe to proceed. It is built for developers who maintain many projects and want one predictable command instead of repetitive shell loops.

## Why Git Fleet

- **Safety first:** dirty repositories are skipped and risky operations are explicit.
- **Plan before execution:** use `--dry-run` to inspect every intended command.
- **GitFlow aware:** resolve `develop`, `master`, `main`, or the latest release branch.
- **Automation friendly:** stable exit behavior and focused commands for scripts and CI.
- **Cross-platform:** a single Go binary with no runtime dependencies.

## Quick Start

Requirements: Git and Go 1.22 or newer.

```bash
go install github.com/black-osiris-technologies/git-fleet/cmd/git-fleet@latest

git-fleet scan --root ~/code
git-fleet status --root ~/code
git-fleet sync --root ~/code --target develop --dry-run
```

Review the dry-run output before removing `--dry-run`:

```bash
git-fleet sync --root ~/code --target develop
```

## Commands

| Command | Purpose |
| --- | --- |
| `scan` | Discover Git repositories below a root directory. |
| `status` | Report branch and working-tree state for every repository. |
| `sync` | Fetch, switch, and fast-forward clean repositories to a target branch. |
| `release-pr` | Create release promotion pull requests using explicit source and target branches. |
| `release-merge` | Merge a release pull request with a normal merge commit. |

Examples:

```bash
git-fleet sync --root . --target develop --dry-run
git-fleet release-pr --root . --from latest-release --to master --dry-run
git-fleet release-merge --root . --from latest-release --to master --merge-method merge --dry-run
```

Run `git-fleet <command> --help` for command-specific options.

## Safety Model

Git Fleet intentionally favors a stopped operation over an ambiguous mutation.

- Dirty repositories are skipped by default.
- Sync uses `fetch --prune` and `pull --ff-only`.
- Destructive resets, bulk commits, bulk pushes, and automatic branch deletion are out of scope.
- Release promotions preserve merge history; squash release merges are rejected.
- Dry-run output is available for operations that can change repository state.

Always keep independent backups for important work. A coordination tool cannot replace repository access controls or a recovery plan.

## Project Status

Git Fleet is under active development. The current release covers repository discovery, status reporting, guarded synchronization, and GitFlow release pull requests. Interfaces may evolve before v1.0.

See [CHANGELOG.md](CHANGELOG.md) for shipped changes and the [issue tracker](https://github.com/black-osiris-technologies/git-fleet/issues) for planned work.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/git-fleet
```

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md), the [Code of Conduct](CODE_OF_CONDUCT.md), and [SECURITY.md](SECURITY.md) before opening a pull request.

## License

Released under the [MIT License](LICENSE).
