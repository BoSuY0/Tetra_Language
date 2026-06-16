#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE' >&2
Usage: bash scripts/release/surface/surface-docs-claims-gate.sh [--report-dir DIR]

Validates Surface documentation and release wording against the Surface claim
scanner. When --report-dir is provided, same-commit Morph rendered beauty
evidence from that directory may satisfy gated beauty/quality claims.
USAGE
}

report_dir=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 ]]; then
        echo "error: --report-dir requires a value" >&2
        usage
        exit 2
      fi
      report_dir="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
cd "$repo_root"

cmd=(go run ./tools/cmd/validate-surface-claims --root "$repo_root")
if [[ -n "$report_dir" ]]; then
  cmd+=(--report-dir "$report_dir")
fi

"${cmd[@]}"
