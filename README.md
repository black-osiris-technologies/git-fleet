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
- **Automation friendly:** `--json` output, `--jobs` parallelism, stable exit behavior, and focused commands for scripts and CI.
- **Cross-platform:** a single Go binary with no runtime dependencies.

## Install

Git Fleet ships as a single binary with no runtime dependencies. Only Git is
required to *use* it; Go is only needed if you build from source.

**Linux / macOS — install script**

```bash
curl -fsSL https://raw.githubusercontent.com/black-osiris-technologies/git-fleet/master/scripts/install.sh | sh
```

Installs to `/usr/local/bin` (falling back to `~/.local/bin`). Pin a version or
change the directory with `GIT_FLEET_VERSION` and `GIT_FLEET_INSTALL_DIR`.

**Windows — install script (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/black-osiris-technologies/git-fleet/master/scripts/install.ps1 | iex
```

Installs to `%LOCALAPPDATA%\git-fleet\bin` and adds it to your user `PATH`.

**Download a binary manually**

Grab the archive for your OS/arch from the [latest release](https://github.com/black-osiris-technologies/git-fleet/releases/latest),
extract it, and move `git-fleet` onto your `PATH`. Checksums are published as
`checksums.txt`.

**Linux packages**

`.deb` and `.rpm` packages are attached to each release:

```bash
sudo dpkg -i git-fleet_*_linux_amd64.deb   # Debian/Ubuntu
sudo rpm -i  git-fleet_*_linux_amd64.rpm   # Fedora/RHEL
```

**From source (requires Go 1.22+)**

```bash
go install github.com/black-osiris-technologies/git-fleet/cmd/git-fleet@latest
```

Note that `go install` places the binary in `$(go env GOPATH)/bin`, which may
not be on your `PATH`. The install scripts above avoid that.

## Quick Start

```bash
git-fleet version
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
| `release-start` | Cut the next release branch from `origin/develop`, deriving the version from the latest tag. |
| `release-tag` | Cut and push the next patch tag on a release line (`v2.4.0`, `v2.4.1`, …). |
| `release-pr` | Create release promotion pull requests using explicit source and target branches. |
| `release-merge` | Merge a release pull request with a normal merge commit. |
| `release-finish` | Merge a release into both `master` and `develop`, then optionally delete the branch. |
| `version` | Print the installed version, commit, and build date. |

Examples:

```bash
git-fleet sync --root . --target develop --dry-run
git-fleet release-start --root . --dry-run
git-fleet release-pr --root . --from latest-release --to master --dry-run
git-fleet release-merge --root . --from latest-release --to master --merge-method merge --dry-run
git-fleet release-finish --root . --from latest-release --dry-run
```

### Starting the next release line

`release-start` cuts a new release branch from `origin/develop` across the fleet.
The version is derived per repository from its highest stable tag, so repositories
at different versions each advance independently:

```bash
# Latest tag 2.3.5 -> creates release-2.4 (next minor, the default).
git-fleet release-start --root ~/code --dry-run
git-fleet release-start --root ~/code

# Start the next major line instead: latest tag 2.3.5 -> release-3.0.
git-fleet release-start --root ~/code --major

# Seed the first release for repositories that have no tags yet.
git-fleet release-start --root ~/code --version 1.0

# Use a three-part branch name (for example release/2.4.0).
git-fleet release-start --root ~/code --branch-format 'release/{major}.{minor}.{patch}'
```

Behavior worth knowing:

- **Minor by default.** The latest tag is bumped by minor (`2.3.5` -> `release-2.4`);
  pass `--major` for the next major line (`release-3.0`). Patch releases belong on an
  existing release branch, so `release-start` does not create them.
- **Idempotent.** If the target branch already exists it is reported as skipped and
  never recreated, so re-running a partially completed fleet operation is safe.
- **Pre-release tags are ignored** when choosing the latest version, and repositories
  with no eligible tag are skipped unless you pass `--version`.
- **Branch name is configurable** with `--branch-format` using `{major}`, `{minor}`,
  and `{patch}` (default `release-{major}.{minor}`).

### Tagging a release line

`release-tag` cuts the next patch tag on a release line and pushes it. Tags are
the immutable artifacts promoted through environments, so each stabilization
iteration cuts the next patch:

```bash
# On the release-2.4 line: v2.4.0 first, then v2.4.1, v2.4.2, ...
git-fleet release-tag --root ~/code --dry-run
git-fleet release-tag --root ~/code

# Target a specific release line rather than the latest.
git-fleet release-tag --root ~/code --from release-2.4

# Pin an exact version (must be on the resolved line).
git-fleet release-tag --root ~/code --version 2.4.3
```

Behavior worth knowing:

- **Continuous patch sequence.** The highest existing patch on the `MAJOR.MINOR`
  line is advanced by one (or `.0` when the line has no tags yet); tags on other
  lines never affect the sequence.
- **Tags the authoritative tip.** The tag is created on `origin/<release-branch>`
  after a pruning fetch, so a stabilized commit is promoted, never a stale local one.
- **Annotated tags** named `v{major}.{minor}.{patch}` by default (`--tag-format` to
  change), with an optional `--message`. A repository without a release branch on
  origin is skipped.
- **`release-tag` advances by design** — re-running cuts the next patch. Use
  `--dry-run` to confirm the target before pushing.

> **Note.** In a CI-driven release flow where a pipeline (for example Jenkins)
> owns tagging — building the production artifact, running tests and
> integrations, and tagging the validated commit so a single immutable artifact
> is promoted across environments — that pipeline stays canonical for tags.
> `release-tag` is then optional: use it for repositories without such a
> pipeline, or for ad-hoc tagging. `release-start`, `release-pr`, and
> `release-finish` complement a Jenkins-style flow without touching tags.

### Finishing a release

`release-finish` completes a GitFlow release by merging the release branch into
**both** the production and integration branches with merge commits, preserving
release history on both lines:

```bash
# Merge the open release PRs into master and develop (never squash).
git-fleet release-finish --root ~/code --dry-run
git-fleet release-finish --root ~/code

# Also delete the release branch on origin once both merges land.
git-fleet release-finish --root ~/code --delete-branch

# Non-default permanent branch names.
git-fleet release-finish --root ~/code --master main --develop develop
```

Behavior worth knowing:

- **Both permanent branches.** The release is merged into `master` and `develop`
  (configurable with `--master`/`--develop`), so neither line misses the release.
- **Through pull requests.** It merges the existing open release PRs (create them
  with `release-pr --to master` and `release-pr --to develop`), so branch
  protection and review are honored; it never merges locally.
- **Safe to re-run.** A branch with no open PR into a target is reported as
  skipped rather than failed, so a partially completed finish can be repeated.
- **Deletion is guarded.** `--delete-branch` removes the release branch on origin
  only after both merges succeed in the same run, so an unmerged branch is never
  deleted.

Run `git-fleet <command> --help` for command-specific options.

### Automation and scale

- **`--json`** switches any command to a structured JSON document (`{"results":[…],"summary":{…}}`,
  or `{"repos":[…]}` for `scan`) instead of tab-separated text, so output can be parsed
  reliably in CI:

  ```bash
  git-fleet status --root ~/code --json
  git-fleet release-start --root ~/code --dry-run --json
  ```

- **`--jobs N`** processes repositories in parallel (default 8) for `status`, `sync`, and
  the `release-*` commands. Repositories are independent working trees, so their git
  operations run concurrently; output stays in a stable, path-sorted order regardless of
  completion order. Use `--jobs 1` to force sequential processing.

## Safety Model

Git Fleet intentionally favors a stopped operation over an ambiguous mutation.

- Dirty repositories are skipped by default.
- Sync uses `fetch --prune` and `pull --ff-only`.
- `release-start` creates a branch without checking it out and skips any repository whose target branch already exists, so it never overwrites work or leaves repositories on a new branch.
- `release-tag` tags the fetched `origin` tip of a release branch and refuses to recreate a tag that already exists, so tags stay immutable.
- Destructive resets, bulk commits, bulk pushes, and automatic branch deletion are out of scope.
- Release promotions preserve merge history; squash release merges are rejected.
- `release-finish` merges through pull requests into both permanent branches and deletes a release branch only after both merges succeed, so it never bypasses review or removes unmerged work.
- Dry-run output is available for operations that can change repository state.

Always keep independent backups for important work. A coordination tool cannot replace repository access controls or a recovery plan.

## Project Status

Git Fleet is under active development. The current release covers repository discovery, status reporting, guarded synchronization, release branch creation, release tagging, and GitFlow release pull requests including finishing a release into both permanent branches. Interfaces may evolve before v1.0.

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
