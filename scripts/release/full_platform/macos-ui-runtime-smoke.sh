#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/full-platform-ui-runtime"
evidence_path="${TETRA_MACOS_UI_RUNTIME_EVIDENCE:-}"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/full_platform/macos-ui-runtime-smoke.sh [--report-dir DIR] [--evidence PATH]

Copies and validates a macOS target-host tetra.ui.platform.v1 runtime report.
On non-macOS hosts without --evidence it writes a blocked report and exits
non-zero; build-only, metadata-only, runtime-less, fake/mock/placeholder, and
startup_failure evidence never counts as production UI runtime evidence.
USAGE
}

host_triple() {
  local kernel
  local machine
  kernel="$(uname -s 2>/dev/null || printf unknown)"
  machine="$(uname -m 2>/dev/null || printf unknown)"
  case "${kernel}:${machine}" in
    Linux:x86_64|Linux:amd64)
      printf 'linux-x64'
      ;;
    Linux:i386|Linux:i686)
      printf 'linux-x86'
      ;;
    Darwin:x86_64|Darwin:amd64)
      printf 'macos-x64'
      ;;
    Darwin:arm64|Darwin:aarch64)
      printf 'macos-arm64'
      ;;
    MINGW*:x86_64|MINGW*:amd64|MSYS*:x86_64|MSYS*:amd64|CYGWIN*:x86_64|CYGWIN*:amd64)
      printf 'windows-x64'
      ;;
    *)
      printf 'unknown-%s-%s' "$kernel" "$machine" | tr '[:upper:]' '[:lower:]'
      ;;
  esac
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      report_dir="${2:-}"
      if [[ -z "$report_dir" ]]; then
        echo "error: --report-dir requires a value" >&2
        exit 2
      fi
      shift 2
      ;;
    --evidence)
      evidence_path="${2:-}"
      if [[ -z "$evidence_path" ]]; then
        echo "error: --evidence requires a value" >&2
        exit 2
      fi
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
mkdir -p "$report_dir"
report_path="$report_dir/macos-ui-runtime.json"

if [[ -n "$evidence_path" ]]; then
  cp -- "$evidence_path" "$report_path"
  go run ./tools/cmd/validate-macos-ui-runtime --report "$report_path"
  go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
  go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
  echo "macOS UI runtime target-host report: $report_path"
  exit 0
fi

host_triple="$(host_triple)"
generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
cat >"$report_path" <<JSON
{
  "schema": "tetra.ui.platform.v1",
  "generated_at": "${generated_at}",
  "status": "blocked",
  "target": "macos-x64",
  "host": "${host_triple}",
  "platform": "macos",
  "runtime": "platform-ui-macos-x64",
  "ui_schema": "tetra.ui.v1",
  "evidence_kind": "target-host-runtime",
  "source": "scripts/release/full_platform/macos-ui-runtime-smoke.sh",
  "blocker": "cannot collect production UI runtime evidence on this host; provide a macOS target-host runtime report via --evidence",
  "processes": [],
  "contracts": [],
  "widgets": [],
  "events": [],
  "cases": []
}
JSON

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
echo "macOS UI runtime blocked report: $report_path" >&2
echo "cannot collect production UI runtime evidence on this host" >&2
exit 1
