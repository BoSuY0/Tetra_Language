#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/full-platform-ui-runtime"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/full_platform/ui-runtime-gate.sh [--report-dir DIR]

Runs the full-platform UI runtime promotion gate. This gate is intentionally
strict: Windows and macOS must provide real target-host runtime reports, not
build-only, metadata-only, runtime-less, fake/mock/placeholder, startup_failure,
or remote-blocked evidence.
USAGE
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

require_fresh_report_dir() {
  if [[ -d "$report_dir" ]] && find "$report_dir" -mindepth 1 -maxdepth 1 | grep -q .; then
    echo "full-platform UI runtime gate: --report-dir requires a fresh empty report directory: $report_dir" >&2
    exit 2
  fi
}

require_fresh_report_dir
mkdir -p "$report_dir"

failures=()

record_failure() {
  local name="$1"
  local code="$2"
  failures+=("${name} (exit ${code})")
  echo "full-platform UI runtime gate: ${name} failed with exit ${code}" >&2
}

run_required_step() {
  local name="$1"
  shift
  set +e
  "$@"
  local code=$?
  set -e
  if [[ "$code" -ne 0 ]]; then
    record_failure "$name" "$code"
  fi
}

go test ./compiler/... ./cli/... ./tools/... -count=1
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
./tetra targets --format=json >"$report_dir/targets.json"
go run ./tools/cmd/validate-targets --report "$report_dir/targets.json"

run_required_step "native UI linux-x64 smoke" bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir "$report_dir"
run_required_step "native UI linux-x64 validator" go run ./tools/cmd/validate-native-ui-runtime --report "$report_dir/native-ui-linux-x64.json"

run_required_step "UI production runtime linux-x64 smoke" bash scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh --report-dir "$report_dir"
run_required_step "UI production runtime linux-x64 validator" go run ./tools/cmd/validate-ui-production-runtime --report "$report_dir/ui-production-runtime-linux-x64.json"

run_required_step "windows UI runtime smoke" bash scripts/release/full_platform/windows-ui-runtime-smoke.sh --report-dir "$report_dir"
run_required_step "windows UI runtime validator" go run ./tools/cmd/validate-windows-ui-runtime --report "$report_dir/windows-ui-runtime.json"

run_required_step "macOS UI runtime smoke" bash scripts/release/full_platform/macos-ui-runtime-smoke.sh --report-dir "$report_dir"
run_required_step "macOS UI runtime validator" go run ./tools/cmd/validate-macos-ui-runtime --report "$report_dir/macos-ui-runtime.json"

run_required_step "web UI runtime smoke" bash scripts/release/v1_0/web-smoke.sh --report "$report_dir/web-smoke.json"
run_required_step "web UI runtime validator" go run ./tools/cmd/validate-web-ui-smoke --report "$report_dir/web-smoke.json"

run_required_step "cross-platform UI runtime validator" go run ./tools/cmd/validate-cross-platform-ui-runtime \
  --linux "$report_dir/ui-production-runtime-linux-x64.json" \
  --windows "$report_dir/windows-ui-runtime.json" \
  --macos "$report_dir/macos-ui-runtime.json" \
  --web "$report_dir/web-smoke.json"

run_required_step "artifact hash manifest write" go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
run_required_step "artifact hash manifest validate" go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

if [[ "${#failures[@]}" -ne 0 ]]; then
  echo "full-platform UI runtime gate failed:" >&2
  for failure in "${failures[@]}"; do
    echo " - ${failure}" >&2
  done
  exit 1
fi

cat >"$report_dir/full-platform-ui-runtime-gate.json" <<'JSON'
{
  "schema": "tetra.ui.full-platform-runtime-gate.v1",
  "status": "pass",
  "contract": "tetra.ui.platform.v1"
}
JSON

run_required_step "final artifact hash manifest write" go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
run_required_step "final artifact hash manifest validate" go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

if [[ "${#failures[@]}" -ne 0 ]]; then
  echo "full-platform UI runtime gate failed:" >&2
  for failure in "${failures[@]}"; do
    echo " - ${failure}" >&2
  done
  exit 1
fi

echo "Full-platform UI runtime gate report directory: $report_dir"
