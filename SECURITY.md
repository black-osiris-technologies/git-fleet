# Security Policy

## Supported Versions

Security fixes are applied to the latest published release. Pre-1.0 releases may receive fixes through a newer minor release rather than a patch branch.

## Reporting a Vulnerability

Do not open a public issue for suspected vulnerabilities. Use [GitHub private vulnerability reporting](https://github.com/black-osiris-technologies/git-fleet/security/advisories/new) with:

- affected command and version;
- reproduction steps or a minimal repository layout;
- expected and observed behavior;
- potential impact;
- suggested mitigation, if known.

Remove credentials, private remote URLs, and sensitive repository content from reports. We will acknowledge valid reports, assess impact, and coordinate disclosure after a fix is available.

## Security Model

Git Fleet executes Git and GitHub CLI commands across local repositories. Its security boundary depends on trusted local executables, repository permissions, and explicit operator intent. Dry-run behavior, dirty-worktree checks, fast-forward-only pulls, and non-destructive defaults are part of the security model.
