# Contributing

## Development setup

Requirements: Go (see `go.mod` for the minimum version), `git`, `ssh`.

```sh
git clone git@github.com:pszypowicz/optiprime.git
cd optiprime
go build -o optiprime .
```

## Running tests

```sh
go test ./...
```

Run the same checks CI runs:

```sh
.github/scripts/checks.sh
```

The script runs `gofmt`, `go vet`, `go mod tidy -diff`, `go build`, and
`go test` over the whole module.

## Pre-commit hook

A hook at `.githooks/pre-commit` runs `.github/scripts/checks.sh`, the same
script CI runs. It skips commits that touch no Go file. Enable it once per
clone:

```sh
git config core.hooksPath .githooks
```

Disable again with `git config --unset core.hooksPath`.

## Code style

- `gofmt` is enforced in CI; the pre-commit hook will catch drift locally.
- Keep PR titles action-oriented. They are the `What's Changed` bullets in
  release notes, so write them the way you want them to appear.

## Release process

Tags matching `v*` trigger the release workflow, which cross-compiles
linux/windows/darwin × amd64/arm64 archives via GoReleaser and publishes a
GitHub release with auto-generated notes.

```sh
git tag vX.Y.Z
git push origin vX.Y.Z
```
