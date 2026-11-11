# Changelog

Notable changes to this project. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/).

## [0.5.1] - 2026-11-11

### Added

- Every release archive comes with a build provenance attestation.
  `gh attestation verify <archive> -R pszypowicz/optiprime` checks that the
  archive was built by this repository's release workflow.

### Changed

- Homebrew installs come from the `optiprime` cask in `pszypowicz/tap`,
  which each release now updates. On the next `brew update`, Homebrew moves
  an installed `optiprime` formula to the cask, or prints the commands that
  do it.

## [0.5.0] - 2026-11-11

### Added

- Without a PAT, the tool uses your `az login` session for the Remote tab
  and the PR counts. It mints an Azure DevOps token for the Entra tenant
  that backs the org, which can differ from the tenant of your
  subscriptions, and renews the token before it expires. Without `az`, or
  without a login, the tool stays in git-only mode.
- `ADO_TOKEN` takes an Entra access token, and `ADO_TENANT_ID` names the
  tenant of the org to skip the lookup.
- An Entra access token in `AZURE_DEVOPS_EXT_PAT` is sent as a bearer token.

### Changed

- The git-only notice in the header says why the remote features are off.
- Dependencies are updated, including Bubble Tea 2.1 and Lip Gloss 2.1. A
  build from source needs Go 1.26.7 or later.

## [0.4.0] - 2026-10-08

### Fixed

- Inside tmux, the tool used the dark palette on a light terminal. The zebra
  rows then showed dark text on a near-black background. The tool now asks
  the terminal for its background color, and tmux 3.4 and later answer.
- The zebra rows, the cursor row, and the details panel set their own text
  color. If a terminal does not report its background, the text stays
  readable.

### Changed

- The TUI uses Bubble Tea v2 and Lip Gloss v2. A build from source needs
  Go 1.26 or later.

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

[0.5.1]: https://github.com/pszypowicz/optiprime/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/pszypowicz/optiprime/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/pszypowicz/optiprime/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/pszypowicz/optiprime/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/pszypowicz/optiprime/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/pszypowicz/optiprime/releases/tag/v0.1.0
