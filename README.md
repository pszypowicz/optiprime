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

| Variable               | Required | Purpose                                                                  |
| ---------------------- | -------- | ------------------------------------------------------------------------ |
| `ADO_ORG`              | yes      | ADO organisation name (the bit between `dev.azure.com/` and the project) |
| `ADO_PROJECT`          | yes      | ADO project name                                                         |
| `AZURE_DEVOPS_EXT_PAT` | yes      | PAT with Code (read) scope - used as the HTTP basic-auth password        |

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

| Key               | Action                                                        |
| ----------------- | ------------------------------------------------------------- |
| `j`/`k` or arrows | move cursor                                                   |
| `space`           | toggle selection                                              |
| `a`               | select all ff-ready (on default branch, clean, behind)        |
| `n`               | deselect all                                                  |
| `u`               | fast-forward every selected repo to `origin/<default>`        |
| `l`               | launch `lazygit` inside the hovered repo (re-fetches on exit) |
| `tab`             | switch to Remote                                              |
| `r`               | rescan + re-fetch                                             |
| `q`               | quit                                                          |

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
