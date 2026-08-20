#!/usr/bin/env bash
# Single source of truth for repo checks. The pre-commit hook and the CI
# workflow both run this script, so a passing hook means a passing PR.
#
# Usage: .github/scripts/checks.sh

set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

go_files=$(git ls-files -- '*.go')
if [ -n "$go_files" ]; then
    # shellcheck disable=SC2086 # word splitting is intended for the file list
    unformatted=$(gofmt -l $go_files)
    if [ -n "$unformatted" ]; then
        echo "gofmt would rewrite these files:" >&2
        echo "$unformatted" >&2
        echo "run: gofmt -w $unformatted" >&2
        exit 1
    fi
fi

go vet ./...
go mod tidy -diff
go build ./...
go test ./...
