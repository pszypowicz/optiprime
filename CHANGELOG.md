# Changelog

Notable changes to this project. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/).

## [0.3.0] - 2026-08-20

### Changed

- `AZURE_DEVOPS_EXT_PAT` is now optional. Without it the tool starts in
  git-only mode: scan, fetch, fast-forward, and lazygit work as normal,
  while the ADO REST features are off - the Remote tab, the PR counts, and
  the `archived upstream` / `not in ADO` states. The header shows a notice,
  and the Remote tab reads `Remote (off)`.
- Without a PAT the org and the project are not resolved, so the tool also
  starts in directories whose repos have no ADO remotes.

## [0.2.0] - 2026-08-20

### Changed

- Renamed the project from `optiprime-sync` to `optiprime`. The rename covers
  the repository, the module path (`github.com/pszypowicz/optiprime`), the
  binary, the log directory, and the trace variable (now
  `OPTIPRIME_GIT_TRACE`).
- Only `AZURE_DEVOPS_EXT_PAT` is required in the environment. The org and the
  project derive from the `origin` remotes of the repos in scope. `ADO_ORG`
  and `ADO_PROJECT` remain as optional overrides. (#3)
- The error log moved to `~/Library/Logs/optiprime/` on macOS.

### Added

- Opt-in git/ssh trace mode via `OPTIPRIME_GIT_TRACE=1` for connection
  diagnostics.
- A shared check script at `.github/scripts/checks.sh`. The pre-commit hook
  and CI both run it, so a passing hook means a passing PR.

## [0.1.0] - 2026-04-20

Initial release: a terminal UI that keeps a directory of Azure DevOps repos
in sync. Parallel fetch, batch fast-forward updates, and SSH clone of repos
that are missing locally.

[0.3.0]: https://github.com/pszypowicz/optiprime/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/pszypowicz/optiprime/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/pszypowicz/optiprime/releases/tag/v0.1.0
