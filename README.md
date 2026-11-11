# optiprime

Terminal UI for keeping a directory of Azure DevOps repositories in sync.

Scans the **current working directory** for git repos, fetches them all in parallel,
shows which are fast-forward-updatable to `origin/<default>`, and lets you update
them in a batch. A second tab lists every repo in the configured ADO project and
lets you SSH-clone the ones you don't have locally.

## Model

The working directory IS the scope. Every immediate subfolder that is a git repo
is treated as one local copy of an ADO repo (matched by folder name).

The tool is a thin layer on top of the repos. Git owns all protocol and auth
handling for fetch and clone, with your normal git config, ssh agent, and
credential helpers. The ADO credential is used only for the ADO REST API,
which git cannot serve: the repo list and the PR counts. The credential is
optional - without it the tool runs in git-only mode and turns those two
features off.

## Requirements

- `git`, `ssh` on PATH
- SSH key registered with ADO (this tool clones via `sshUrl`)
- Optional, for the Remote tab and the PR counts: an ADO Personal Access
  Token with **Code (read)** scope, or an `az login` session whose account
  can read the org

The `az` CLI is optional. The tool calls the ADO REST API directly and runs
`az` only to mint an access token when neither a PAT nor `ADO_TOKEN` is set.

## Environment

| Variable               | Required | Purpose                                                                                                                                                                                      |
| ---------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `AZURE_DEVOPS_EXT_PAT` | no       | PAT with Code (read) scope - used as the HTTP basic-auth password for the REST API. An Entra access token here is sent as a bearer token. Takes precedence over `ADO_TOKEN` and `az login`  |
| `ADO_TOKEN`            | no       | Entra access token for Azure DevOps, sent as a bearer token. Used when `AZURE_DEVOPS_EXT_PAT` is unset, and takes precedence over `az login`                                                 |
| `ADO_TENANT_ID`        | no       | GUID of the Entra tenant that backs the org. Skips the tenant lookup when the tool mints a token from `az login`                                                                             |
| `ADO_ORG`              | no       | Override for the ADO organization name. When unset, the tool derives it from the `origin` remotes of the repos in scope                                                                      |
| `ADO_PROJECT`          | no       | Override for the ADO project name. Derived the same way when unset                                                                                                                           |
| `OPTIPRIME_GIT_TRACE`  | no       | Set to `1` to run fetch/clone with `GIT_TRACE=1` and `ssh -v`, recording the full output in the error log - use to diagnose connection failures (agent unreachable, auth rejected, host key) |

No variable is required. Without a credential the tool starts in git-only
mode: scan, fetch, fast-forward, and lazygit work as normal, while the Remote
tab, the PR counts, and the `archived upstream` / `not in ADO` states are
off. The header shows a notice with the reason, and the Remote tab reads
`Remote (off)`.

With a credential, the org and the project come from the `origin` remotes of
the repos in the scope directory. Both HTTPS remotes
(`https://dev.azure.com/<org>/<project>/_git/<repo>`) and SSH remotes
(`git@ssh.dev.azure.com:v3/<org>/<project>/<repo>`) are recognized. With a
PAT or `ADO_TOKEN`, the tool exits with an error if the remotes disagree, or
if the scope contains no ADO remote and no override is set. With `az login`
the same case starts the tool in git-only mode.

### Authentication

The REST features use the first credential available:

1. `AZURE_DEVOPS_EXT_PAT`
2. `ADO_TOKEN`
3. A token minted from your `az login` session

For `az login`, the tool looks up the Entra tenant that backs the org in the
`X-VSS-ResourceTenant` header of an anonymous request to
`<org>/_apis/connectionData`. It then runs
`az account get-access-token --resource 499b84ac-1321-427f-aa17-267ca6975798 --tenant <org tenant>`.
The GUID is the public Azure DevOps resource. The tenant matters because the
org can sit in another tenant than your Azure subscriptions. If your account
is a member or guest of that tenant, your existing login is enough. Tokens
expire after about an hour, so the tool mints a new one shortly before
expiry, and once more if ADO rejects a token.

If `az` is not installed or not logged in, the tool runs in git-only mode,
the same as without a PAT. Other `az` failures show on the Remote tab with
the command that fixes them. For example, a tenant that requires an
interactive sign-in needs
`az login --tenant <org tenant> --allow-no-subscriptions --scope 499b84ac-1321-427f-aa17-267ca6975798/.default`.
Over SSH the suggested command uses `--use-device-code`.

Config-file support is not wired up yet - env is the only source.

## Install

With Homebrew, on macOS or Linux:

```sh
brew install --cask pszypowicz/tap/optiprime
```

The binary is not notarized, so the cask removes the macOS quarantine flag
after install. Without that step, macOS would refuse to run it.

From the [releases](https://github.com/pszypowicz/optiprime/releases), each
archive has a build provenance attestation. To check that an archive was
built by this repository's release workflow:

```sh
gh attestation verify optiprime_<version>_<os>_<arch>.tar.gz -R pszypowicz/optiprime
```

With Go:

```sh
go install github.com/pszypowicz/optiprime@latest
```

or build from source:

```sh
git clone <this repo>
cd optiprime
go build -o optiprime .
```

## Usage

```sh
cd ~/Developer/dev.azure.com/<org>/<project>
optiprime
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
- **ADO**: direct REST call (`GET /_apis/git/repositories?api-version=7.1`) with PAT basic auth, or a bearer token from `ADO_TOKEN` or `az login`

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
