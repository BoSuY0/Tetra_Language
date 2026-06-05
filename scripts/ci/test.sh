#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
cd "$repo_root"

release_version="${TETRA_TEST_RELEASE_VERSION:-}"
if [[ -z "$release_version" ]]; then
  if [[ -x ./tetra ]]; then
    release_version="$(./tetra version)"
  else
    release_version="v0.4.0"
  fi
fi
release_slug="${release_version#v}"
release_slug="${release_slug//./_}"
release_artifact="${TETRA_TEST_RELEASE_ARTIFACT:-tetra.release.v${release_slug}.go-test-suite.v1}"

usage() {
  cat <<'USAGE'
Usage: bash scripts/ci/test.sh [--frontend-focused]

Runs the canonical Go test suite:
- gofmt -l formatting gate for tracked and untracked Go files
- go test ./compiler/... -count=1
- go test ./cli/... -count=1
- go test ./tools/... -count=1

Focused targets:
- --frontend-focused:
  - go test ./compiler/internal/frontend ./compiler -run 'Lex|Parser|Flow|Diagnostic|Stabilization' -count=1
  - go test ./compiler/internal/backend/wasm32_web -count=1
USAGE
}

mode="canonical"
if [[ $# -gt 0 ]]; then
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --frontend-focused)
      mode="frontend-focused"
      shift
      ;;
    *)
      echo "test.sh: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
fi

if [[ $# -gt 0 ]]; then
  echo "test.sh: unexpected argument $1" >&2
  usage >&2
  exit 2
fi

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  unformatted="$(
    git ls-files -z --cached --others --exclude-standard '*.go' |
      while IFS= read -r -d '' path; do
        if [[ -f "$path" ]]; then
          printf '%s\0' "$path"
        fi
      done |
      xargs -0 gofmt -l
  )"
else
  unformatted="$(find . -name '*.go' -not -path './.gocache/*' -not -path './.cache/*' -print0 | xargs -0 gofmt -l)"
fi

if [[ -n "$unformatted" ]]; then
  echo "test.sh: Go files need formatting:" >&2
  printf '%s\n' "$unformatted" >&2
  echo "Run gofmt on the listed files before retrying." >&2
  exit 1
fi

case "$mode" in
  canonical)
    go test ./compiler/... -count=1
    go test ./cli/... -count=1
    go test ./tools/... -count=1
    ;;
  frontend-focused)
    go test ./compiler/internal/frontend ./compiler -run 'Lex|Parser|Flow|Diagnostic|Stabilization' -count=1
    go test ./compiler/internal/backend/wasm32_web -count=1
    ;;
esac

echo "OK"
echo "Artifact: $release_artifact"
