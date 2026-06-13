#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-linux-x64-release-app-shell-smoke.sh [--report-dir DIR]

Runs the linux-x64 Tetra Surface app-shell subset smoke.
The report validates tetra.surface.linux-app-shell.v1
linux-app-shell-subset-v1 evidence for the electron feature ledger:
multi-window notes, lifecycle open/close/reopen, resize/DPI/cursors,
clipboard, IME, accessibility, scoped crash/error reporting adapters,
file dialog, file picker, notification, and tray blocked-pass evidence, and
surface-security-permission-v1 default-deny filesystem/network policy,
capability-checked IPC/process boundaries, local hashed asset/font/image
safety, surface-performance-budget-v1 startup/frame/memory/cache/framebuffer
local budget evidence with no faster-than-Electron claim, and rejection of
GTK/Qt/native widget UI substitutes.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 ]]; then
        echo "error: --report-dir requires a value" >&2
        usage >&2
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
      usage >&2
      exit 2
      ;;
  esac
done

cd "$repo_root"
mkdir -p "$report_dir"
report_dir="$(cd "$report_dir" && pwd)"
report_path="$report_dir/surface-linux-x64-release-app-shell.json"

json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/ }"
  value="${value//$'\r'/ }"
  printf '%s' "$value"
}

write_blocked_report() {
  local reason="$1"
  local escaped_reason
  escaped_reason="$(json_escape "$reason")"
  cat >"$report_path" <<JSON
{
  "schema": "tetra.surface.runtime.v1",
  "status": "blocked",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "surface-linux-x64",
  "source": "examples/surface_linux_app_shell_notes.tetra",
  "production_claim": false,
  "blocked_reason": "$escaped_reason",
  "host_evidence": {
    "level": "linux-x64-release-window-v1",
    "backend": "wayland-shm-rgba-release-v1",
    "framebuffer": false,
    "real_window": false,
    "native_input": false,
    "text_input": false,
    "clipboard": false,
    "composition": false,
    "accessibility_bridge": false
  }
}
JSON
}

if [[ -z "${WAYLAND_DISPLAY:-}" && -z "${DISPLAY:-}" ]]; then
  reason="missing WAYLAND_DISPLAY or DISPLAY for bounded linux-x64 app-shell target-host smoke"
  write_blocked_report "$reason"
  echo "Surface linux-x64 app-shell blocked: $reason" >&2
  echo "Surface linux-x64 app-shell blocked report: $report_path" >&2
  exit 1
fi

go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-release-app-shell --source examples/surface_linux_app_shell_notes.tetra --report "$report_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path" --release linux-app-shell
go run ./tools/cmd/validate-surface-security-report --report "$report_path"
go run ./tools/cmd/validate-surface-performance-budget --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface linux-x64 app-shell runtime smoke report: $report_path"
echo "Surface linux-x64 app-shell artifact hashes: $report_dir/artifact-hashes.json"
