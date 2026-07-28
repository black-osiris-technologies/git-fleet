# Changelog

All notable changes are documented in this file. The project follows [Semantic Versioning](https://semver.org/) and the structure from [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

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

[Unreleased]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/black-osiris-technologies/git-fleet/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/black-osiris-technologies/git-fleet/releases/tag/v0.1.0
