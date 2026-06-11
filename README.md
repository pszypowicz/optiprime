# optiprime-sync

Terminal UI for keeping a directory of Azure DevOps repositories in sync.

Scans the **current working directory** for git repos, fetches them all in parallel,
shows which are fast-forward-updatable to `origin/<default>`, and lets you update
them in a batch. A second tab lists every repo in the configured ADO project and
lets you SSH-clone the ones you don't have locally.

## Model

The working directory IS the scope. Every immediate subfolder that is a git repo
is treated as one local copy of an ADO repo (matched by folder name).

## Requirements

- `git`, `ssh` on PATH
- SSH key registered with ADO (this tool always clones via `sshUrl`)
- An ADO Personal Access Token with **Code (read)** scope

No `az` CLI dependency - repos are listed via the ADO REST API directly.

## Environment

| Variable                   | Required | Purpose                                                                                                                                                                                      |
| -------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ADO_ORG`                  | yes      | ADO organisation name (the bit between `dev.azure.com/` and the project)                                                                                                                     |
| `ADO_PROJECT`              | yes      | ADO project name                                                                                                                                                                             |
| `AZURE_DEVOPS_EXT_PAT`     | yes      | PAT with Code (read) scope - used as the HTTP basic-auth password                                                                                                                            |
| `OPTIPRIME_SYNC_GIT_TRACE` | no       | Set to `1` to run fetch/clone with `GIT_TRACE=1` and `ssh -v`, recording the full output in the error log - use to diagnose connection failures (agent unreachable, auth rejected, host key) |

Config-file support is not wired up yet - env is the only source.

## Install

```sh
go install github.com/pszypowicz/optiprime-sync@latest
```

or build from source:

```sh
git clone <this repo>
cd optiprime-sync
go build -o optiprime-sync .
```

## Usage

```sh
cd ~/Developer/dev.azure.com/<org>/<project>
optiprime-sync
```

### Keys

**Local tab**

| Key               | Action                                                                                                                                                    |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `j`/`k` or arrows | move cursor                                                                                                                                               |
| `space`           | toggle selection                                                                                                                                          |
| `a`               | select every repo that is safe to update in one keystroke                                                                                                 |
| `n`               | deselect all                                                                                                                                              |
| `u`               | update every selected repo: fast-forward on the default branch, or switch to default and ff when a feature branch's work is already in `origin/<default>` |
| `l`               | launch `lazygit` inside the hovered repo (re-fetches on exit)                                                                                             |
| `tab`             | switch to Remote                                                                                                                                          |
| `r`               | rescan + re-fetch                                                                                                                                         |
| `q`               | quit                                                                                                                                                      |

### Status glyphs

| glyph       | meaning                                                                    |
| ----------- | -------------------------------------------------------------------------- |
| `↑N↓M`      | commits ahead / behind upstream (or `origin/<default>` if no upstream set) |
| `+N`        | staged changes                                                             |
| `~N`        | unstaged tracked changes                                                   |
| `!N`        | merge conflicts                                                            |
| `?N`        | untracked files                                                            |
| `⚑N`        | stash entries                                                              |
| `[MERGING]` | in-progress `merge`, `rebase`, `cherry-pick`, `revert`, `bisect`, or `am`  |
| `[N PR]`    | `N` open PRs authored by the signed-in user on this repo                   |

### State column

| state                   | meaning                                                                          |
| ----------------------- | -------------------------------------------------------------------------------- |
| `ff-ready`              | on default branch, clean, and behind - one keystroke to fast-forward             |
| `merged → switch & ff`  | on a feature branch whose work is already in `origin/<default>` - safe to switch |
| `merged (dirty)`        | work is merged upstream but the tree is dirty, so the switch is not offered      |
| `up-to-date`            | at the tip of upstream / default                                                 |
| `diverged`              | local has commits upstream doesn't and vice versa                                |
| `ahead`                 | local has commits not in upstream                                                |
| `behind (other branch)` | on a non-default branch; default is behind but your branch is independent        |
| `behind (dirty)`        | default is behind but the tree is dirty, so ff is refused                        |
| `archived upstream`     | repo exists locally and in ADO but ADO marked it disabled                        |
| `not in ADO`            | local folder has no matching ADO repo (renamed or deleted upstream)              |

**Remote tab**

| Key               | Action                                             |
| ----------------- | -------------------------------------------------- |
| `j`/`k` or arrows | move cursor                                        |
| `enter`           | clone highlighted repo via SSH into the scope root |
| `tab`             | switch to Local                                    |
| `r`               | re-list ADO + rescan locals                        |
| `q`               | quit                                               |

## Stack

- **TUI**: [`bubbletea`](https://github.com/charmbracelet/bubbletea) (MVU) + `bubbles` (spinner) + `lipgloss` (styling)
- **Git**: shelled out (`os/exec`) - matches local git config, credentials, and hooks exactly
- **ADO**: direct REST call (`GET /_apis/git/repositories?api-version=7.1`) with PAT basic auth

`bubbletea` was picked over `tview` because each parallel `git fetch` can return
an independent `tea.Msg`, rendering incrementally as results arrive. `tview`'s
callback model makes that shape of update awkward.

## Concurrency

Up to 8 git operations run in parallel (bounded by a semaphore) so a 40-repo scope
doesn't fork 40 `git fetch` processes at once.

## Limitations

- Only immediate subfolders are scanned (no recursion).
- `git submodule` and repos with non-`origin` remotes are not handled specially.
- Feature branches are shown but never auto-selected - `u` only ever touches
  repos sitting on the default branch.
