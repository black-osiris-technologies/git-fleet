# Git Fleet

> Safe, repeatable Git and GitFlow operations across every repository in your workspace.

[![CI](https://github.com/black-osiris-technologies/git-fleet/actions/workflows/ci.yml/badge.svg)](https://github.com/black-osiris-technologies/git-fleet/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/black-osiris-technologies/git-fleet)](https://github.com/black-osiris-technologies/git-fleet/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](go.mod)

Git Fleet coordinates guarded Git operations across many local repositories. It discovers repositories below a workspace root, reports what it can safely do, and then synchronizes branches or runs a GitFlow-style release lifecycle without destructive resets or hidden history rewrites.

The tool is designed for teams that maintain many repositories with a canonical `origin`, permanent integration/production branches such as `develop` and `master`, and versioned release branches.

## Highlights

- Fleet-wide repository discovery and status.
- Dirty-worktree protection before mutating operations.
- Dry-run support for every mutating command.
- Fast-forward-only synchronization against `origin`.
- `latest-release` and `previous-release` selection per repository.
- Release creation from authoritative remote state.
- Optional stable SemVer patch tagging.
- GitHub pull-request promotion with merge commits.
- Guarded release-branch deletion after promotion.
- JSON output, bounded parallelism, deterministic result ordering, and non-zero failure exits.
- Cross-platform single-binary distribution.

## Requirements and operating assumptions

### Required for all commands

- `git` must be available on `PATH`.
- Repositories must be normal Git working trees below the chosen `--root`.
- The canonical remote is currently expected to be named **`origin`**.

Git Fleet deliberately treats `origin` as authoritative for automatic release selection and for release/tag existence checks.

### Required for GitHub PR commands

`release-pr`, `release-merge`, and `release-finish` call the GitHub REST API directly. The GitHub CLI (`gh`) is **not** required and does not need to be installed or present on `PATH`.

For GitHub.com, set one of these environment variables before running a mutating PR command:

- `GH_TOKEN` (preferred);
- `GITHUB_TOKEN`.

For GitHub Enterprise Server, set `GH_ENTERPRISE_TOKEN` or `GITHUB_ENTERPRISE_TOKEN`. Git Fleet deliberately does **not** fall back to `GH_TOKEN` / `GITHUB_TOKEN` for non-`github.com` hosts, so a GitHub.com credential cannot be sent to an arbitrary `origin` host.

The token must have access to the target repository and permission to create or merge pull requests. For a fine-grained personal access token, grant the repository **Pull requests: write** permission. Repository review, status-check, and branch-protection requirements still apply; Git Fleet does not bypass them.

Git Fleet reads the token from the environment for the current process and does not persist it. The repository owner/name and GitHub host are derived from the canonical `origin` remote. Standard GitHub HTTPS/SSH remotes and GitHub Enterprise Server remotes are supported. Explicit ports on GitHub Enterprise HTTPS remotes are preserved for API requests. Plaintext `http://` origins are rejected for authenticated GitHub API operations so bearer tokens are never sent without TLS.

Real PR commands validate that a supported GitHub `origin` and a token are configured before performing Git mutations. `--dry-run` remains non-mutating and does not call the GitHub API, so it does not validate token correctness or repository API permissions.

`scan`, `status`, `sync`, `release-start`, and `release-tag` do not require a GitHub API token.

### Release naming conventions

Automatic release resolution intentionally uses a constrained naming family so a branch or tag created by one Git Fleet command remains understandable to later commands.

Supported release branch shapes include:

```text
release-2.4
release-2.4.0
release/2.4
release/2.4.0
```

The branch must begin with `release-` or `release/` and then contain dot-separated numeric version components. `--branch-format` is validated after rendering and is rejected if it produces a name outside that family.

Stable release tags must be SemVer triples with an optional `v` prefix:

```text
v2.4.0
2.4.0
```

`release-tag --tag-format` therefore accepts only:

```text
v{major}.{minor}.{patch}
{major}.{minor}.{patch}
```

Pre-release/build forms such as `v2.4.0-rc.1` and `v2.4.0+build.7` are not eligible when Git Fleet derives the next release line or patch sequence.

## Installation

Git Fleet ships as a single binary. Go is only required when building from source.

### Linux / macOS install script

```bash
curl -fsSL https://raw.githubusercontent.com/black-osiris-technologies/git-fleet/master/scripts/install.sh | sh
```

The installer uses `/usr/local/bin` when writable and otherwise falls back to `~/.local/bin`.

You can pin a release or install directory with environment variables supported by the script.

### Windows PowerShell install script

```powershell
irm https://raw.githubusercontent.com/black-osiris-technologies/git-fleet/master/scripts/install.ps1 | iex
```

The installer places the binary under `%LOCALAPPDATA%\git-fleet\bin` and adds that directory to the user `PATH`.

### Manual binary download

Download the archive for your OS/architecture from the [latest GitHub release](https://github.com/black-osiris-technologies/git-fleet/releases/latest), extract it, and place `git-fleet` on your `PATH`.

Release assets include `checksums.txt`.

### Linux packages

```bash
sudo dpkg -i git-fleet_*_linux_amd64.deb   # Debian / Ubuntu
sudo rpm -i git-fleet_*_linux_amd64.rpm    # Fedora / RHEL
```

### Install from source

Requires Go 1.22+:

```bash
go install github.com/black-osiris-technologies/git-fleet/cmd/git-fleet@latest
```

`go install` writes to `$(go env GOPATH)/bin`, which may need to be added to your `PATH`.

## Five-minute quick start

Assume your repositories live below `~/code`:

```bash
# Verify the binary.
git-fleet version

# See what will be managed.
git-fleet scan --root ~/code

# Inspect local state.
git-fleet status --root ~/code

# Preview synchronization.
git-fleet sync --root ~/code --target develop --dry-run

# Apply after reviewing the plan.
git-fleet sync --root ~/code --target develop
```

The default root is the current directory (`.`), so `--root` can be omitted when running from the workspace root.

## Command overview

| Command | Purpose |
| --- | --- |
| `scan` | Discover Git repositories below a root directory. |
| `status` | Report current branch and dirty/clean state. |
| `sync` | Fetch, switch, and fast-forward clean repositories to a target branch. |
| `release-start` | Create the next release branch from an integration branch. |
| `release-tag` | Create and push the next stable patch tag on a release line. |
| `release-pr` | Create a GitHub release-promotion pull request. |
| `release-merge` | Merge one open release PR with a merge commit. |
| `release-finish` | Merge a release into both permanent branches and optionally delete the release branch. |
| `version` | Print version/build metadata. |

Run command-specific help at any time:

```bash
git-fleet <command> --help
```

## Repository discovery

`scan` recursively finds standard Git working trees and returns them in stable path order.

```bash
git-fleet scan --root ~/code
git-fleet scan --root ~/code --json
```

Git Fleet skips common generated/tooling directories while traversing:

```text
node_modules  vendor  dist  build  target  .idea  .vscode  .tmp
```

## Status

```bash
git-fleet status --root ~/code
git-fleet status --root ~/code --jobs 4
git-fleet status --root ~/code --json
```

A dirty worktree is valid status information, not an error. If a repository cannot be inspected, the error is still printed and the process exits non-zero.

## Synchronizing repositories

`sync` updates each **clean** repository to a requested target. Dirty repositories are reported as `SKIPPED` and are not changed.

### Explicit branch

```bash
git-fleet sync --root ~/code --target develop --dry-run
git-fleet sync --root ~/code --target develop

git-fleet sync --root ~/code --target master
git-fleet sync --root ~/code --target main
```

Any explicit branch name can be supplied.

If the branch already exists locally and also exists on `origin`, Git Fleet keeps the local checkout but fast-forwards it explicitly against `origin/<branch>`; it does not depend on whatever upstream happens to be configured locally.

If the branch exists only on `origin`, Git Fleet creates a local tracking branch.

A legacy explicit local-only branch can still fall back to its configured upstream, but automatic release selectors never use local-only release branches. For predictable fleet operation, keep synchronized targets on `origin`.

### Latest active release

```bash
git-fleet sync --root ~/code --target latest-release --dry-run
git-fleet sync --root ~/code --target latest-release
```

`latest-release` selects the numerically highest supported release branch currently present on `origin`, independently for every repository.

Example:

```text
repo-a: origin/release-2.8, origin/release-2.9  -> release-2.9
repo-b: origin/release-4.1                     -> release-4.1
```

Local-only release branches are ignored for automatic release selection.

### Previous active release

```bash
git-fleet sync --root ~/code --target previous-release --dry-run
git-fleet sync --root ~/code --target previous-release
```

`previous-release` selects the second-highest active release branch on `origin` per repository. A repository with fewer than two active release branches is `SKIPPED`.

This is useful when two release trains are maintained in parallel.

### Sync mechanics

Dry-run reads live branch names from `origin` with `git ls-remote`, so a deleted remote branch is not kept alive merely by a stale local remote-tracking ref.

Real execution first runs the equivalent of:

```text
git fetch --prune --prune-tags --tags
```

For an origin-backed target, the selected local branch is then updated with a fast-forward-only merge from the explicit remote-tracking ref:

```text
git merge --ff-only origin/<branch>
```

This has several deliberate consequences:

- deleted remote-tracking branches are pruned;
- tags deleted from `origin` are pruned locally;
- **local-only tags can therefore be deleted**;
- Git Fleet never creates an implicit merge during synchronization;
- a divergent branch fails instead of being rewritten;
- local upstream configuration cannot redirect an origin-backed sync to another remote.

Treat `origin` as authoritative for the tag namespace when using `sync`.

## Release lifecycle

A typical fleet-wide release looks like this:

```bash
# 1. Synchronize integration branches.
git-fleet sync --root ~/code --target develop --dry-run
git-fleet sync --root ~/code --target develop

# 2. Cut the next release line.
git-fleet release-start --root ~/code --dry-run
git-fleet release-start --root ~/code

# 3. Work/synchronize on the active release line.
git-fleet sync --root ~/code --target latest-release

# 4. Optional: tag the release line if Git Fleet owns tagging.
git-fleet release-tag --root ~/code --dry-run
git-fleet release-tag --root ~/code

# 5. Create promotion PRs into both permanent branches.
git-fleet release-pr --root ~/code --from latest-release --to master --dry-run
git-fleet release-pr --root ~/code --from latest-release --to master

git-fleet release-pr --root ~/code --from latest-release --to develop --dry-run
git-fleet release-pr --root ~/code --from latest-release --to develop

# 6. After review/checks, merge both PRs.
git-fleet release-finish --root ~/code --from latest-release --dry-run
git-fleet release-finish --root ~/code --from latest-release
```

Use `--from previous-release` when an operation must target the previous active release train.

## `release-start`

`release-start` creates the next release branch from `origin/develop` by default.

```bash
# Highest stable origin tag v2.3.5 -> release-2.4
git-fleet release-start --root ~/code --dry-run
git-fleet release-start --root ~/code

# Start the next major line: v2.3.5 -> release-3.0
git-fleet release-start --root ~/code --major

# Seed a repository that has no stable release tags yet.
git-fleet release-start --root ~/code --version 1.0

# Use another integration branch.
git-fleet release-start --root ~/code --base main

# Supported alternate release naming shape.
git-fleet release-start --root ~/code --branch-format 'release/{major}.{minor}.{patch}'
```

Behavior:

- the highest stable tag currently on `origin` determines the next release version;
- minor is the default bump; `--major` starts the next major line;
- patch releases remain on an existing release line;
- dry-run reads branches/tags directly from `origin` without mutating local refs;
- real execution prunes stale remote-tracking refs and tags first;
- a local-only branch with the same target name is preserved and ignored;
- the release branch is created directly on `origin` from the fetched `origin/<base>`;
- a create-only force-with-lease prevents a concurrent actor's new branch from being advanced accidentally;
- an existing target branch on `origin` is `SKIPPED`.

See [docs/release-start-remote-authority.md](docs/release-start-remote-authority.md) for the remote-authority rationale.

## `release-tag`

`release-tag` creates the next stable patch tag on a release line.

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

# Stable SemVer without v.
git-fleet release-tag --root ~/code --tag-format '{major}.{minor}.{patch}'
```

Behavior:

- branch and tag discovery is authoritative to live `origin`, including dry-run;
- stale/deleted local tags do not advance the remote patch sequence;
- real execution uses `fetch --prune --prune-tags --tags` first;
- only tags on the selected MAJOR.MINOR line affect its next patch;
- the annotated tag points at the fetched `origin/<release-branch>` tip;
- publication uses a create-only tag lease, so an identically named tag created concurrently is never overwritten;
- if tag publication fails, the local tag created by that invocation is removed best-effort to keep retries clean;
- `--message` overrides the default `Release <tag>` annotation;
- supported tag formats are `v{major}.{minor}.{patch}` and `{major}.{minor}.{patch}`.

If Jenkins, GitHub Actions, or another delivery pipeline already owns immutable artifact tagging, keep that pipeline authoritative and treat `release-tag` as optional.

## `release-pr`

`release-pr` creates a GitHub pull request from a release branch into a target branch.

```bash
git-fleet release-pr --root ~/code --from latest-release --to master --dry-run
git-fleet release-pr --root ~/code --from latest-release --to master

git-fleet release-pr --root ~/code --from latest-release --to develop

git-fleet release-pr --root ~/code --from previous-release --to master

git-fleet release-pr --root ~/code --from release-2.4 --to master
```

Behavior:

- automatic selectors resolve against live `origin`;
- target existence is checked against live `origin`;
- when the source already exists on `origin`, a same-named local branch is **not pushed**, preventing stale local state from changing the PR source;
- an explicit local-only source can be published intentionally, but publication is create-only with a lease so a concurrently created remote branch is not advanced;
- the GitHub REST API is used directly to detect/create the pull request; no `gh` executable is required;
- an already-existing PR is `SKIPPED` rather than duplicated;
- the generated PR text instructs maintainers to use merge commits rather than squash.

## `release-merge`

`release-merge` merges one existing release PR into one target branch.

```bash
git-fleet release-merge --root ~/code --from latest-release --to master --dry-run
git-fleet release-merge --root ~/code --from latest-release --to master
```

Only the merge-commit method is supported:

```bash
git-fleet release-merge --root ~/code --merge-method merge
```

The source release branch must exist on `origin`; a local-only release branch is not considered mergeable through GitHub PRs.

`release-merge` uses the GitHub REST API to find the open PR and merges it with GitHub's `merge` method. A missing open PR is `SKIPPED`.

## `release-finish`

`release-finish` completes normal GitFlow promotion by merging the release into **both** permanent branches.

```bash
git-fleet release-finish --root ~/code --from latest-release --dry-run
git-fleet release-finish --root ~/code --from latest-release

# Delete the remote release branch after both merges succeed.
git-fleet release-finish --root ~/code --from latest-release --delete-branch

# Non-default permanent branch names.
git-fleet release-finish --root ~/code --master main --develop develop
```

Behavior:

- the source release must exist on `origin`;
- merges happen through existing GitHub PRs, preserving repository protection/review rules;
- merge commits are used for both targets;
- a missing open PR is `SKIPPED`, making partial reruns safe;
- `--delete-branch` is attempted only when both target merges succeeded during that invocation;
- before deletion, Git Fleet compares the current remote release SHA with the fetched SHA;
- if the branch advanced during the operation, deletion is refused and the repository is `FAILED` rather than deleting new work;
- deletion itself uses a force-with-lease;
- if GitHub already auto-deleted the merged release branch, that is accepted as the desired final state rather than reported as failure.

## Dry-run semantics

Use `--dry-run` before any mutating command:

```bash
git-fleet sync --root ~/code --target develop --dry-run
git-fleet release-start --root ~/code --dry-run
git-fleet release-tag --root ~/code --dry-run
git-fleet release-pr --root ~/code --from latest-release --to master --dry-run
git-fleet release-merge --root ~/code --from latest-release --to master --dry-run
git-fleet release-finish --root ~/code --from latest-release --dry-run
```

Dry-run does not create branches/tags, change the checkout, push refs, create PRs, merge PRs, or delete branches.

Where remote truth determines the operation, Git Fleet uses read-only `git ls-remote` queries so stale remote-tracking refs do not silently drive planning.

For PR merge/finish commands, dry-run validates source/target ref selection but does not claim an open PR exists; actual open-PR discovery happens during execution through the GitHub REST API. Dry-run does not require or validate a GitHub API token.

## Result states and exit codes

Fleet action commands emit one result per repository:

- `PLANNED` / `READY` — dry-run can proceed.
- `DONE` — requested work completed.
- `SKIPPED` — safe no-op or repository not applicable; this is not a process failure by itself.
- `FAILED` — operational/configuration failure requiring attention.

Examples of `SKIPPED` conditions include:

- dirty worktree on a mutating command;
- fewer than two active releases for `previous-release`;
- target release branch already exists during `release-start`;
- no open PR to merge;
- an already-existing release PR or tag.

Examples of `FAILED` conditions include:

- inability to query/fetch/push `origin`;
- invalid release/tag configuration;
- non-fast-forward synchronization;
- GitHub API authentication, permission, or request failures;
- a guarded branch deletion refused because the branch advanced.

The complete fleet report is printed first. If any repository is `FAILED`, the command then returns a non-zero process exit. `SKIPPED` alone keeps exit code 0.

`status` follows the same principle: dirty is informational, while a repository inspection error makes the process exit non-zero.

## JSON output

Action commands support `--json`:

```bash
git-fleet sync --root ~/code --target latest-release --dry-run --json
git-fleet release-start --root ~/code --dry-run --json
```

The shape is:

```json
{
  "results": [
    {
      "repo": "/code/service-a",
      "action": "DONE",
      "message": "..."
    }
  ],
  "summary": {
    "total": 1,
    "done": 1,
    "failed": 0
  }
}
```

Summary keys vary with the command (`ready`, `planned`, `done`, `skipped`, `failed`).

`scan --json` returns a `repos` array. `status --json` returns a `results` array with branch/dirty/error fields.

If JSON output contains one or more failed repository operations, Git Fleet still returns a non-zero exit after writing the JSON document.

## Parallelism

`status`, `sync`, and release commands support `--jobs N`.

```bash
git-fleet sync --root ~/code --target develop --jobs 4
git-fleet release-start --root ~/code --jobs 1 --dry-run
```

The default is 8 concurrent repositories. Repositories are independent working trees, and output remains in deterministic path order regardless of completion order.

Use `--jobs 1` when debugging or when external infrastructure should be accessed sequentially.

## Safety model

Git Fleet intentionally prefers a stopped operation over an ambiguous mutation.

- Dirty repositories are skipped by mutating commands.
- Automatic release selection is based on active branches on `origin`, not local-only branches.
- Dry-run consults live remote state for operations whose meaning depends on `origin`.
- Synchronization never resets a branch and never creates an implicit merge.
- Origin-backed sync explicitly fast-forwards from `origin/<branch>` instead of trusting arbitrary local upstream configuration.
- Tag pruning is explicit and documented; local-only tags may be removed.
- Release branch creation is create-only and lease guarded.
- Tag publication is create-only and lease guarded.
- Existing remote PR source branches are never replaced by same-named local state.
- Release promotion uses merge commits, never squash/rebase.
- Optional release-branch deletion is protected against concurrent branch advancement.
- Git Fleet does not perform destructive resets, bulk commits, or force-push branch rewrites.

Git Fleet is a coordination tool, not a replacement for repository access controls, backups, protected branches, CI, or review policy.

## Troubleshooting

### A repository is `SKIPPED` because it is dirty

Inspect it directly:

```bash
git -C /path/to/repo status
```

Commit, stash, or otherwise resolve the working-tree changes yourself. Git Fleet does not stash or reset user work automatically.

### `previous-release` says there are fewer than two active releases

The repository needs at least two supported release branches on `origin`. Local-only branches do not count.

```bash
git ls-remote --heads origin 'refs/heads/release*'
```

### A tag disappeared locally after sync/release operations

`sync`, `release-start`, and `release-tag` may use `--prune-tags`. A tag that does not exist on `origin` can therefore be removed locally. Do not rely on unpushed local tags when using these fleet operations.

### GitHub PR commands fail

The PR commands do not use the GitHub CLI. Ensure a token is available to the Git Fleet process:

```bash
# bash / zsh
export GH_TOKEN="<token>"
```

```powershell
# PowerShell
$env:GH_TOKEN = "<token>"
```

`GITHUB_TOKEN` is also accepted for GitHub.com. On GitHub Enterprise Server, use `GH_ENTERPRISE_TOKEN` or `GITHUB_ENTERPRISE_TOKEN`; generic GitHub.com token variables are intentionally ignored for enterprise hosts.

For a fine-grained token, verify that the target repository is included and **Pull requests: write** is granted. API errors are reported by repository with the HTTP status and GitHub message. Also verify branch-protection and required-review/status-check rules.

### A release branch was not deleted after `release-finish --delete-branch`

Git Fleet deliberately refuses deletion if the live release branch advanced after the operation began. Inspect the branch and merge any new work before retrying.

### A custom branch/tag format is rejected

The rendered release branch and stable tags must remain inside Git Fleet's automatic resolution conventions. Use `git-fleet release-start --help` and `git-fleet release-tag --help` for the supported forms.

## Project status

Git Fleet is under active development and remains pre-1.0. Interfaces can still evolve as safety rules and release workflows are hardened.

See [CHANGELOG.md](CHANGELOG.md) for shipped and unreleased changes, and the [issue tracker](https://github.com/black-osiris-technologies/git-fleet/issues) for planned work.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/git-fleet
```

The GitHub Actions CI workflow runs the same validation on supported branches and pull requests.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md), the [Code of Conduct](CODE_OF_CONDUCT.md), and [SECURITY.md](SECURITY.md) before opening a pull request.

## License

Released under the [MIT License](LICENSE).
