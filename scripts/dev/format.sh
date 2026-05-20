#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
cd "$repo_root"

usage() {
  cat <<'USAGE'
Usage: bash scripts/dev/format.sh

Formats tracked and untracked Go files in the repository with gofmt -w.
USAGE
}

if [[ $# -gt 0 ]]; then
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "format.sh: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
fi

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  git ls-files -z --cached --others --exclude-standard '*.go' | xargs -0 gofmt -w
else
  find . -name '*.go' -not -path './.gocache/*' -not -path './.cache/*' -print0 | xargs -0 gofmt -w
fi

echo "OK"
