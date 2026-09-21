# Changelog

All notable changes are documented in this file. The project follows [Semantic Versioning](https://semver.org/) and the structure from [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.4.1] - 2026-09-21

### Added

- Complete README usage guide covering requirements, `origin`/release naming
  conventions, `latest-release` and `previous-release`, the end-to-end GitFlow
  lifecycle, JSON/parallel operation, exit codes, safety behavior, and common
  troubleshooting.
- Regression coverage for remote-authoritative release selection, stale/deleted
  refs, tag pruning, create-only ref publication, guarded branch deletion,
  compatible release naming formats, and non-zero fleet failure reporting.

### Fixed

- `release-pr`, `release-merge`, and `release-finish`
  no longer require the external GitHub CLI (`gh`). Git Fleet now performs PR
  discovery, creation, and merge operations through the GitHub REST API while
  preserving merge-commit-only release promotion.
- GitHub PR mutations now preflight the configured `origin` and API token before
  changing repository state. Authentication is read from `GH_TOKEN` /
  `GITHUB_TOKEN` (or the GitHub Enterprise token variants) and is not persisted
  by Git Fleet.
- GitHub.com tokens are never reused for non-`github.com` origins; GitHub Enterprise hosts require explicitly enterprise-scoped token variables, preventing credentials from being sent to arbitrary origin hosts.
- GitHub Enterprise HTTPS origins with explicit ports now preserve that port when constructing REST API URLs; plaintext `http://` origins are rejected before authentication.

### Changed

- `sync` dry-run now reads live branch names from `origin`, and origin-backed
  synchronization fast-forwards explicitly from `origin/<branch>` rather than
  relying on arbitrary local upstream configuration.
- `release-tag` now treats `origin` as authoritative for both release-branch
  selection and patch-tag sequencing. Dry-run queries remote refs directly and
  real execution prunes stale local tags before resolving the next patch.
- Tag publication is create-only with a ref lease. A failed tag push removes the
  local tag created by that invocation best-effort so a retry can refetch remote
  state cleanly.
- Automatic `release-pr`, `release-merge`, and `release-finish` release selection
  now reads live `origin` state. Target-branch checks also use live remote state,
  avoiding stale remote-tracking refs during dry-run.
- `release-pr` no longer pushes a same-named local branch when the source already
  exists on `origin`. Explicit local-only sources are published create-only with
  a lease.
- `release-merge` and `release-finish` require their release source to exist on
  `origin`; a local-only branch cannot be mistaken for a GitHub PR source.
- `release-finish --delete-branch` now compares the live remote release SHA with
  the fetched SHA and deletes with a lease. A concurrently advanced branch is
  retained and reported as failed, while a branch already auto-deleted by GitHub
  is accepted as the desired final state.
- `--branch-format` and `--tag-format` are validated so Git Fleet cannot create
  release names that later automatic commands cannot resolve. Release branches
  must stay in the `release-X.Y[...]` / `release/X.Y[...]` families and release
  tags must remain stable SemVer triples with an optional `v` prefix.
- Fleet action commands now return a non-zero process exit after printing the
  complete report when any repository is `FAILED`. `status` follows the same
  rule for per-repository inspection errors; `SKIPPED` remains a successful
  process outcome.
- Operational failures while querying `origin` are reported as `FAILED`, while
  expected no-op conditions such as a missing applicable release remain
  `SKIPPED`.
- CLI help now documents `previous-release` anywhere a release selector is
  accepted and describes the supported release branch/tag format constraints.

## [0.4.0] - 2026-08-30

### Note

- The published `v0.4.0` tag points to the same source commit as the previous
  production line, so it contains no additional source changes relative to
  `v0.3.1`. The accumulated unreleased changes are published in `v0.4.1`.

## [0.3.1] - 2026-07-28

### Fixed

- `release-tag` tests configure a git identity on the test clone so they
  pass on a clean CI runner (no functional change to the binary).

## [0.3.0] - 2026-07-28

### Added

- `release-start` command that cuts the next release branch from `origin/develop`
  across the fleet. The version is derived per repository from its highest stable
  tag (minor bump by default, `--major` for the next major line), the branch name
  is configurable with `--branch-format`, and the operation is idempotent: a
  repository whose target branch already exists is skipped rather than recreated.
- `release-tag` command that cuts and pushes the next patch tag on a release line
  across the fleet. The patch number advances continuously per `MAJOR.MINOR` line
  (tags on other lines are ignored), the annotated tag is created on the
  authoritative `origin/<release-branch>` tip after a pruning fetch, the tag name
  is configurable with `--tag-format`, and `--version` pins an exact patch on the
  resolved line.
- `release-finish` command that completes a GitFlow release across the fleet by
  merging the release branch into both `master` and `develop` with merge commits
  (never squash), through the existing open pull requests so review and branch
  protection are honored. A target with no open PR is skipped rather than failed,
  and `--delete-branch` removes the release branch on origin only after both
  merges succeed. The permanent branch names are configurable with `--master`
  and `--develop`.

- `--json` flag on every command that emits a structured JSON document instead of
  tab-separated text (`{"results":[…],"summary":{…}}`, or `{"repos":[…]}` for `scan`),
  making the output reliable to parse in automation.
- `--jobs` flag on `status`, `sync`, and the `release-*` commands that processes
  repositories in parallel (default 8) while keeping output in a stable, path-sorted
  order. Use `--jobs 1` for sequential processing.

- Cross-platform distribution: a GoReleaser configuration and a release workflow
  triggered by pushing a `vX.Y.Z` tag that publish binaries for Linux, macOS, and
  Windows (amd64 and arm64), archives, `checksums.txt`, and `.deb`/`.rpm` packages
  to a GitHub Release. Install scripts (`scripts/install.sh`, `scripts/install.ps1`)
  download the right binary without a Go toolchain, and the Windows script adds
  the install directory to the user `PATH`.
- `version` command that prints the version, commit, and build date stamped into
  released binaries.

### Changed

- Refactored the single-target release merge into a shared helper reused by
  `release-merge` and `release-finish`.
- Documented binary, install-script, and Linux-package installation as the
  primary paths; `go install` is now presented as a from-source option.

## [0.2.1] - 2026-07-12

### Fixed

- Updated installation guidance now that the renamed Go module is available from a release tag.

## [0.2.0] - 2026-07-12

### Changed

- Renamed the project from `omp-git-fleet` to `git-fleet`.
- Expanded project documentation and contribution guidance.

## [0.1.0] - 2026-07-09

### Added

- Repository discovery and status reporting.
- Guarded synchronization with dry-run planning.
- GitFlow release pull request and merge commands.

[Unreleased]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/black-osiris-technologies/git-fleet/releases/tag/v0.1.0
