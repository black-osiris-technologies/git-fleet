# Git Fleet

> Safe, repeatable Git and GitFlow operations across every repository in your workspace.

[![CI](https://github.com/black-osiris-technologies/git-fleet/actions/workflows/ci.yml/badge.svg)](https://github.com/black-osiris-technologies/git-fleet/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/black-osiris-technologies/git-fleet)](https://github.com/black-osiris-technologies/git-fleet/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](go.mod)

Git Fleet coordinates the same guarded operation across many local Git repositories. It discovers repositories below a root directory, reports what it can safely do, and then synchronizes branches or performs release operations without destructive resets or hidden history rewrites.

It is designed for teams that use a GitFlow-style model with permanent integration/production branches and versioned release branches, especially when dozens of repositories need to move together.

## What Git Fleet provides

- **Fleet-wide discovery and status** across a directory tree.
- **Safe synchronization** with dirty-worktree protection and fast-forward-only pulls.
- **Active release selection** with `latest-release` and `previous-release`, resolved independently per repository.
- **Release branch creation** from an authoritative `origin` state.
- **Optional release tagging** with stable SemVer patch sequences.
- **GitHub pull-request promotion** into production and integration branches.
- **GitFlow release finishing** through merge commits, never squash merges.
- **Dry-run-first workflows** for every mutating command.
- **Machine-readable JSON**, bounded parallelism, deterministic output ordering, and CI-friendly non-zero failure exits.
- **Cross-platform distribution** as a single Go binary.

## Requirements and assumptions

### Required for all commands

- `git` must be available on `PATH`.
- Repositories must be normal Git working trees discoverable below `--root`.
- The canonical remote is currently expected to be named **`origin`**.

### Required for GitHub PR commands

`release-pr`, `release-merge`, and `release-finish` call the GitHub CLI (`gh`). For those commands you also need:

- `gh` installed and available on `PATH`;
- an authenticated GitHub CLI session (`gh auth status`);
- permission to create or merge pull requests in the target repositories;
- branch protection/review requirements satisfied before a merge can complete.

`scan`, `status`, `sync`, `release-start`, and `release-tag` do not require `gh`.

### Release naming conventions

Automatic release resolution intentionally supports a constrained convention so every command can understand branches and tags created by every other command.

**Release branches** must use one of these families:

```text
release-2.4
release-2.4.0
release/2.4
release/2.4.0
```

In other words: `release-` or `release/`, followed by at least `MAJOR.MINOR`, with optional additional numeric components.

**Stable release tags** must be SemVer triples with an optional `v` prefix:

```text
v2.4.0
2.4.0
```

Pre-release/build forms such as `v2.4.0-rc.1` or `v2.4.0+build.7` are not used when deriving the next release line or patch sequence.

Custom `--branch-format` and `--tag-format` values are validated. Git Fleet refuses a format that would create names its later automatic commands could not resolve.

## Install

Git Fleet ships as a single binary. Go is only required when building from source.

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/black-osiris-technologies/git-fleet/master/scripts/install.sh | sh
```

The installer uses `/usr/local/bin` when writable and otherwise falls back to `~/.local/bin`.

Optional environment variables:

```bash
GIT_FLEET_VERSION=v0.3.1 GIT_FLEET_INSTALL_DIR="$HOME/.local/bin" sh install.sh
```

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/black-osiris-technologies/git-fleet/master/scripts/install.ps1 | iex
```

The installer places the binary under `%LOCALAPPDATA%\git-fleet\bin` and adds that directory to the user `PATH`.

### Manual binary download

Download the archive for your OS/architecture from the [latest GitHub release](https://github.com/black-osiris-technologies/git-fleet/releases/latest), extract it, and place `git-fleet` on your `PATH`.

Release assets also include `checksums.txt`.

### Linux packages

```bash
sudo dpkg -i git-fleet_*_linux_amd64.deb   # Debian / Ubuntu
sudo rpm -i git-fleet_*_linux_amd64.rpm    # Fedora / RHEL
```

### Build/install from source

Requires Go 1.22+:

```bash
go install github.com/black-osiris-technologies/git-fleet/cmd/git-fleet@latest
```

`go install` writes to `$(go env GOPATH)/bin`, which may need to be added to your `PATH`.

## Five-minute quick start

Assume all repositories live somewhere below `~/code`:

```bash
# Confirm the installed binary.
git-fleet version

# See what Git Fleet discovers.
git-fleet scan --root ~/code

# Check current branch / worktree state.
git-fleet status --root ~/code

# Preview synchronization first.
git-fleet sync --root ~/code --target develop --dry-run

# Apply it after reviewing the plan.
git-fleet sync --root ~/code --target develop
```

The default root is the current directory (`.`), so `--root` can be omitted when you run Git Fleet from the workspace root.

## Repository discovery

`scan` recursively finds standard Git working trees under `--root` and returns them in stable path order.

```bash
git-fleet scan --root ~/code
git-fleet scan --root ~/code --json
```

To avoid expensive or irrelevant traversal, Git Fleet skips common generated/tooling directories such as:

```text
node_modules  vendor  dist  build  target  .idea  .vscode  .tmp
```

## Status

```bash
git-fleet status --root ~/code
git-fleet status --root ~/code --jobs 4
git-fleet status --root ~/code --json
```

Text output reports the current branch and whether the working tree is `clean` or `dirty`.

If one or more repositories cannot be inspected, their errors are still printed and the process exits non-zero. A dirty repository by itself is not an error; it is valid status information.

## Synchronizing repositories

`sync` fetches and updates each **clean** repository to the requested target. Dirty repositories are skipped rather than modified.

### Explicit branch

```bash
git-fleet sync --root ~/code --target develop --dry-run
git-fleet sync --root ~/code --target develop

git-fleet sync --root ~/code --target master
git-fleet sync --root ~/code --target main
```

Any explicit branch name can be supplied. If it exists locally, Git Fleet uses the local branch; otherwise it can create a tracking branch from `origin/<name>`.

### Latest active release per repository

```bash
git-fleet sync --root ~/code --target latest-release --dry-run
git-fleet sync --root ~/code --target latest-release
```

`latest-release` selects the numerically highest release branch visible on `origin` independently for every repository.

Example:

```text
repo-a: origin/release-2.8, origin/release-2.9   -> release-2.9
repo-b: origin/release-4.1                      -> release-4.1
```

A local-only release branch is never considered an active automatic release.

### Previous active release per repository

```bash
git-fleet sync --root ~/code --target previous-release --dry-run
git-fleet sync --root ~/code --target previous-release
```

`previous-release` selects the second-highest active release on `origin` for each repository. A repository with fewer than two active release branches is reported as `SKIPPED`.

This is useful when two release trains are being maintained in parallel and you need to switch the fleet between the current and previous line without hard-coding repository-specific versions.

### What a real sync does

For a clean repository, execution refreshes refs with:

```text
git fetch --prune --prune-tags --tags
```

It then checks out or creates the target tracking branch and runs:

```text
git pull --ff-only
```

Important consequences:

- deleted remote-tracking branches are pruned;
- tags deleted from `origin` are pruned locally;
- **local-only tags can therefore be deleted** by synchronization;
- non-fast-forward pulls fail instead of creating an implicit merge;
- dirty working trees are skipped before mutation.

Treat `origin` as authoritative for the tag namespace when using fleet synchronization.

## Release lifecycle

A typical GitFlow release across a fleet looks like this:

```bash
# 1. Make sure integration branches are current.
git-fleet sync --root ~/code --target develop --dry-run
git-fleet sync --root ~/code --target develop

# 2. Cut the next release line in each repository.
git-fleet release-start --root ~/code --dry-run
git-fleet release-start --root ~/code

# 3. Work/synchronize on the active release line as needed.
git-fleet sync --root ~/code --target latest-release

# 4. Optional: cut immutable patch tags if Git Fleet owns tagging.
git-fleet release-tag --root ~/code --dry-run
git-fleet release-tag --root ~/code

# 5. Create promotion PRs into both permanent branches.
git-fleet release-pr --root ~/code --from latest-release --to master --dry-run
git-fleet release-pr --root ~/code --from latest-release --to master

git-fleet release-pr --root ~/code --from latest-release --to develop --dry-run
git-fleet release-pr --root ~/code --from latest-release --to develop

# 6. After reviews/checks allow it, merge both PRs with merge commits.
git-fleet release-finish --root ~/code --from latest-release --dry-run
git-fleet release-finish --root ~/code --from latest-release
```

Use `--from previous-release` anywhere a release selector is accepted when the operation must target the previous active release train instead.

### `release-start`

`release-start` creates the next release branch from `origin/develop` by default.

```bash
# Highest stable origin tag v2.3.5 -> release-2.4
git-fleet release-start --root ~/code --dry-run
git-fleet release-start --root ~/code

# Next major line: v2.3.5 -> release-3.0
git-fleet release-start --root ~/code --major

# Repository has no stable tags yet: seed the first release explicitly.
git-fleet release-start --root ~/code --version 1.0

# Cut from another integration branch.
git-fleet release-start --root ~/code --base main

# Supported alternate release naming family.
git-fleet release-start --root ~/code --branch-format 'release/{major}.{minor}.{patch}'
```

Behavior:

- the latest **stable tag on `origin`** determines the next release version;
- default bump is minor; `--major` starts the next major line;
- patch releases stay on an existing release branch rather than creating a new line;
- dry-run reads branches/tags directly from `origin` using `git ls-remote`, so stale local refs cannot affect planning;
- real execution uses `fetch --prune --prune-tags --tags` first;
- a local-only branch with the same target name is preserved and ignored;
- the new branch is created directly on `origin` from `origin/<base>` without changing the repository's current checkout;
- creation uses a create-only force-with-lease so a concurrent actor cannot have its newly created release branch silently advanced;
- an existing target branch on `origin` is `SKIPPED`, making retries safe.

See [docs/release-start-remote-authority.md](docs/release-start-remote-authority.md) for the remote-authority rationale.

### `release-tag`

`release-tag` creates the next stable patch tag on a release line and pushes it to `origin`.

```bash
# release-2.4 with no 2.4 tags -> v2.4.0
# next run -> v2.4.1, then v2.4.2, ...
git-fleet release-tag --root ~/code --dry-run
git-fleet release-tag --root ~/code

# Explicit release line.
git-fleet release-tag --root ~/code --from release-2.4

# Previous active release line.
git-fleet release-tag --root ~/code --from previous-release

# Pin an exact patch on the selected MAJOR.MINOR line.
git-fleet release-tag --root ~/code --from release-2.4 --version 2.4.3

# Stable SemVer without the leading v.
git-fleet release-tag --root ~/code --tag-format '{major}.{minor}.{patch}'
```

Behavior:

- branch and tag discovery is authoritative to `origin`, including dry-run;
- deleted/stale local tags do not advance the patch sequence;
- real execution prunes remote-tracking refs and stale local tags before tagging;
- tags from other MAJOR.MINOR lines do not affect the selected line;
- the tag points at the fetched `origin/<release-branch>` tip, not an arbitrary local commit;
- tags are annotated; `--message` overrides the default `Release <tag>` annotation;
- allowed tag formats are `v{major}.{minor}.{patch}` and `{major}.{minor}.{patch}` so future release discovery stays compatible.

If Jenkins, GitHub Actions, or another pipeline already owns tagging and artifact promotion, keep that pipeline authoritative and treat `release-tag` as optional.

### `release-pr`

`release-pr` creates a GitHub pull request from a release branch to a target branch.

```bash
git-fleet release-pr --root ~/code --from latest-release --to master --dry-run
git-fleet release-pr --root ~/code --from latest-release --to master

git-fleet release-pr --root ~/code --from latest-release --to develop

git-fleet release-pr --root ~/code --from previous-release --to master

git-fleet release-pr --root ~/code --from release-2.4 --to master
```

The command uses `gh pr create`. Existing PRs are reported as `SKIPPED` rather than duplicated.

Release PR descriptions explicitly require merge commits rather than squash merges.

### `release-merge`

Use `release-merge` when you want to merge one existing release PR into one target branch.

```bash
git-fleet release-merge --root ~/code --from latest-release --to master --dry-run
git-fleet release-merge --root ~/code --from latest-release --to master
```

Only the `merge` method is supported:

```bash
git-fleet release-merge --root ~/code --merge-method merge
```

Squash/rebase release promotion is deliberately rejected because the release history is expected to remain explicit.

### `release-finish`

`release-finish` completes the normal GitFlow promotion by merging the existing release PRs into **both** permanent branches.

```bash
git-fleet release-finish --root ~/code --from latest-release --dry-run
git-fleet release-finish --root ~/code --from latest-release

# Delete the remote release branch only after both merges succeed in this run.
git-fleet release-finish --root ~/code --from latest-release --delete-branch

# Non-default permanent branch names.
git-fleet release-finish --root ~/code --master main --develop develop
```

Behavior:

- merges happen through GitHub PRs, so repository protection/review rules remain in force;
- merge commits are used for both targets;
- a missing open PR is `SKIPPED`, which makes reruns safe after a partial completion;
- `--delete-branch` deletes the release branch only when both target merges succeeded during that invocation;
- a merge or `gh` failure is `FAILED` and produces a non-zero process exit.

## Dry-run behavior

Use `--dry-run` before any command that can mutate repositories or remotes:

```bash
git-fleet sync --root ~/code --target develop --dry-run
git-fleet release-start --root ~/code --dry-run
git-fleet release-tag --root ~/code --dry-run
git-fleet release-pr --root ~/code --from latest-release --to master --dry-run
git-fleet release-merge --root ~/code --from latest-release --to master --dry-run
git-fleet release-finish --root ~/code --from latest-release --dry-run
```

Dry-run never performs the final mutation. Commands may still read local Git state or query `origin`/GitHub to build an accurate plan.

## Result states and exit codes

Fleet action commands emit one result per repository.

| State | Meaning | Process failure? |
| --- | --- | --- |
| `PLANNED` / `READY` | Dry-run operation is valid. | No |
| `DONE` | Operation completed. | No |
| `SKIPPED` | Repository was intentionally not changed (dirty tree, missing applicable release, already completed state, etc.). | No |
| `FAILED` | Git/GitHub/configuration operation failed. | **Yes** |

Git Fleet prints the complete fleet report even when some repositories fail. After the report is emitted, any `FAILED` repository causes a non-zero process exit.

`status` follows the same CI principle: inspection errors are printed and cause a non-zero exit, while a successfully inspected dirty repository does not.

This makes shell/CI usage predictable:

```bash
if ! git-fleet sync --root ~/code --target develop --json > fleet-result.json; then
  echo "At least one repository failed to synchronize"
fi
```

## JSON output

Use `--json` for automation:

```bash
git-fleet scan --root ~/code --json
git-fleet status --root ~/code --json
git-fleet sync --root ~/code --target latest-release --dry-run --json
git-fleet release-start --root ~/code --dry-run --json
```

Action commands emit a document shaped like:

```json
{
  "results": [
    {
      "repo": "/workspace/service-a",
      "action": "DONE",
      "message": "synced develop"
    }
  ],
  "summary": {
    "total": 1,
    "done": 1,
    "failed": 0,
    "skipped": 0,
    "ready": 0
  }
}
```

The exact summary keys follow the command (`ready` for sync dry-run, `planned` for release dry-runs, and so on).

A failed fleet still prints valid JSON before returning a non-zero exit code, allowing CI to preserve the detailed result as an artifact.

## Parallelism

`status`, `sync`, and all `release-*` commands support `--jobs`.

```bash
git-fleet sync --root ~/code --target develop --jobs 1
git-fleet sync --root ~/code --target develop --jobs 16
```

Default concurrency is `8` repositories.

Repositories are independent working trees, so operations can run concurrently. Results are stored by discovery index and printed in deterministic path order rather than completion order.

Use `--jobs 1` when debugging or when external infrastructure should be exercised sequentially.

## Safety model

Git Fleet favors a stopped/skipped operation over an ambiguous mutation.

- Dirty repositories are skipped before mutating operations.
- Synchronization uses `pull --ff-only`; it does not manufacture merge commits.
- `sync`, `release-start`, and `release-tag` can prune local tags that no longer exist on `origin`; do not use local-only tags as durable unpublished state when running those commands.
- Automatic `latest-release` / `previous-release` selection ignores local-only release branches.
- `release-start` creates the remote branch from the authoritative origin base and does not check it out locally.
- Incompatible branch/tag formats are rejected before they can break later automatic release resolution.
- Release promotion uses GitHub pull requests and merge commits.
- Squash/rebase release merges are rejected by `release-merge`.
- `release-finish --delete-branch` deletes only after both promotion merges succeed in the same invocation.
- Git Fleet does not perform destructive resets, bulk commits, or silent force updates of existing release branches.
- Failed repository operations are visible in output and produce a non-zero process exit.

Backups, repository permissions, protected branches, required reviews, and CI checks remain the responsibility of the repository owner.

## Common troubleshooting

### `dirty worktree on <branch>`

The repository is intentionally skipped. Commit, stash, or discard the local changes yourself, then rerun Git Fleet.

Git Fleet will not automatically stash or reset work.

### `previous-release requires at least 2 active release branches on origin`

That repository has only one (or zero) active release lines. Use `latest-release`, an explicit branch, or create/restore the required release line.

### `no eligible release tag found; pass --version to seed the first release`

The repository has no stable `X.Y.Z` / `vX.Y.Z` tag on `origin` from which `release-start` can derive the next line.

Example:

```bash
git-fleet release-start --root ~/code --version 1.0
```

### Incompatible branch/tag format

Use release branches in the `release-X.Y[...]` or `release/X.Y[...]` family and stable tags as `vX.Y.Z` or `X.Y.Z`.

Formats that create names outside those conventions are rejected because `latest-release`, `previous-release`, `release-tag`, and later release operations must be able to resolve them consistently.

### `gh` authentication or permission failures

Check:

```bash
gh auth status
```

Then confirm the authenticated account has access to the repositories and satisfies any protected-branch/review requirements.

### A tag disappeared locally after sync/release operations

`sync`, `release-start`, and `release-tag` intentionally use tag pruning. If a tag does not exist on `origin`, it can be removed locally.

Push important tags before running those commands, or do not use local-only tags as unpublished work markers.

## Command summary

| Command | Purpose | Mutates state without `--dry-run`? | Needs `gh`? |
| --- | --- | ---: | ---: |
| `scan` | Discover repositories below a root. | No | No |
| `status` | Report branch and worktree state. | No | No |
| `sync` | Fetch, checkout/create target branches, fast-forward pull. | Yes | No |
| `release-start` | Create the next release line from an origin base branch. | Yes | No |
| `release-tag` | Create/push the next patch tag on a release line. | Yes | No |
| `release-pr` | Create release promotion pull requests. | Yes | Yes |
| `release-merge` | Merge one open release PR with a merge commit. | Yes | Yes |
| `release-finish` | Merge release PRs into production + integration branches and optionally delete the release branch. | Yes | Yes |
| `version` | Print version/build metadata. | No | No |

Run command-specific help at any time:

```bash
git-fleet sync --help
git-fleet release-start --help
git-fleet release-tag --help
git-fleet release-pr --help
git-fleet release-merge --help
git-fleet release-finish --help
```

## Project status

Git Fleet is under active development and has not reached v1.0. Interfaces may still evolve, but safety and explicit behavior are treated as compatibility requirements.

See:

- [CHANGELOG.md](CHANGELOG.md) for shipped and unreleased changes;
- [GitHub Issues](https://github.com/black-osiris-technologies/git-fleet/issues) for bugs and planned work;
- [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidance;
- [SECURITY.md](SECURITY.md) for security reporting.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/git-fleet
```

The CI workflow runs the same checks for feature/defect branches and pull requests.

## License

Released under the [MIT License](LICENSE).
