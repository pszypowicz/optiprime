# Contributing

## Development setup

Requirements: Go (see `go.mod` for the minimum version), `git`, `ssh`.

```sh
git clone git@github.com:pszypowicz/optiprime-sync.git
cd optiprime-sync
go build -o optiprime-sync .
```

## Running tests

```sh
go test ./...
```

Run the same checks CI runs:

```sh
go vet ./...
test -z "$(gofmt -l .)"
go test ./...
```

## Pre-commit hook

A hook at `.githooks/pre-commit` runs `gofmt` against staged Go files, then
`go vet` and `go test` over the whole module. Enable it once per clone:

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
