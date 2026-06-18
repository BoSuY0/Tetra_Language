#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_path="reports/surface-prod/surface-prod-claim.json"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/prod-claim-gate.sh [--report FILE]

Validates a tetra.surface.prod-claim.v1 report. This gate is claim governance:
it rejects broad Electron/React/CSS replacement, fake cross-platform, fake GPU,
fake full accessibility, missing renderer backend decision gate evidence, and
missing target-host evidence claims.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report)
      if [[ $# -lt 2 ]]; then
        echo "error: --report requires a value" >&2
        usage >&2
        exit 2
      fi
      report_path="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

cd "$repo_root"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-surface-prod-claim"
fi
mkdir -p "$GOCACHE"

go run -buildvcs=false ./tools/cmd/validate-surface-prod-claim --report "$report_path"
