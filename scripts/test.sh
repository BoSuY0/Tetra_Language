#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

release_artifact="tetra.release.v0_2_0.go-test-suite.v1"

usage() {
  cat <<'USAGE'
Usage: bash scripts/test.sh

Runs the canonical Go test suite:
- go test ./compiler/...
- go test ./cli/...
- go test ./tools/...
USAGE
}

if [[ $# -gt 0 ]]; then
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "test.sh: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
fi

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  git ls-files -z '*.go' | xargs -0 gofmt -w
else
  find . -name '*.go' -not -path './.gocache/*' -not -path './.cache/*' -print0 | xargs -0 gofmt -w
fi

go test ./compiler/...
go test ./cli/...
go test ./tools/...

echo "OK"
echo "Artifact: $release_artifact"
