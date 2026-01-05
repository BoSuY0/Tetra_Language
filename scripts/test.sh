#!/usr/bin/env bash
set -euo pipefail

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  gofmt -w $(git ls-files '*.go')
else
  gofmt -w $(find . -name '*.go' -not -path './.gocache/*' -not -path './.cache/*')
fi

go test ./compiler/...
go test ./cli/...
go test ./tools/...

echo "OK"
