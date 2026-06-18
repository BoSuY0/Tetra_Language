#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
cd "$repo_root"

release_artifact="tetra.release.v0_4_0.bootstrap-binaries.v1"

usage() {
  cat << 'USAGE'
Usage: bash scripts/dev/bootstrap.sh

Builds local CLI binaries:
- ./tetra
- ./t
USAGE
}

if [[ $# -gt 0 ]]; then
  case "$1" in
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "bootstrap: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
fi

exe=""
case "$(go env GOOS)" in
  windows) exe=".exe" ;;
esac

go build -o "./tetra${exe}" ./cli/cmd/tetra
cp "./tetra${exe}" "./t${exe}"
echo "Built: ./tetra${exe} ./t${exe}"
echo "Artifact: $release_artifact"
