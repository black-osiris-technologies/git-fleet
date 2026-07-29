# Changelog

All notable changes are documented in this file. The project follows [Semantic Versioning](https://semver.org/) and the structure from [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

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

[Unreleased]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.3.1...HEAD
[0.3.1]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/black-osiris-technologies/git-fleet/releases/tag/v0.1.0
