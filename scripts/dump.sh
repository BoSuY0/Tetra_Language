#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

release_artifact="tetra.release.v0_2_0.project-dump.v1"

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  cat <<'USAGE'
Usage: bash scripts/dump.sh [dump-project flags...]

Wrapper around:
  go run ./tools/cmd/dump-project
USAGE
  exit 0
fi

go run ./tools/cmd/dump-project "$@"
echo "Artifact: $release_artifact"
