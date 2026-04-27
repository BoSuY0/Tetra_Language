#!/usr/bin/env bash
set -euo pipefail

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  git ls-files -z '*.go' | xargs -0 gofmt -w
else
  find . -name '*.go' -not -path './.gocache/*' -not -path './.cache/*' -print0 | xargs -0 gofmt -w
fi

go test ./compiler/...
go test ./cli/...
go test ./tools/...

echo "OK"
