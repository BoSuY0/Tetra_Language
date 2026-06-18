#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-wasm32-web-release-browser-smoke.sh [--report-dir DIR]

Runs the wasm32-web browser canvas Tetra Surface release browser smoke.
The gate builds examples/surface/release/surface_release_form.tetra, runs it in a real
Chromium-compatible browser canvas, validates wasm imports, records
wasm32-web-browser-canvas-release-v1 evidence with deterministic browser
clipboard/composition traces and browser accessibility snapshot/mirror, rejects
Node-only promotion, DOM visual UI, user JavaScript app logic, and legacy
sidecars, then validates artifact hashes.
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
	-h | --help)
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
report_path="$report_dir/surface-wasm32-web-release-browser.json"
wasm_path="$report_dir/surface-wasm32-web-release-browser-artifacts/surface-release-form.wasm"

surface_wasm32_web_browser_active_pid=""

surface_wasm32_web_browser_cleanup() {
	if [[ -n "${surface_wasm32_web_browser_active_pid:-}" ]] &&
		kill -0 "$surface_wasm32_web_browser_active_pid" 2>/dev/null; then
		kill -TERM "$surface_wasm32_web_browser_active_pid" 2>/dev/null || true
		sleep 1
		kill -KILL "$surface_wasm32_web_browser_active_pid" 2>/dev/null || true
	fi
}

trap surface_wasm32_web_browser_cleanup EXIT

surface_wasm32_web_browser_json_escape() {
	local value="$1"
	value="${value//\\/\\\\}"
	value="${value//\"/\\\"}"
	value="${value//$'\n'/ }"
	value="${value//$'\r'/ }"
	printf '%s' "$value"
}

surface_wasm32_web_browser_blocked_report() {
	local reason="$1"
	local escaped_reason
	escaped_reason="$(surface_wasm32_web_browser_json_escape "$reason")"
	cat >"$report_path" <<JSON
{
  "schema": "tetra.surface.runtime.v1",
  "status": "blocked",
  "target": "wasm32-web",
  "host": "$(go env GOHOSTOS 2>/dev/null || uname -s)",
  "runtime": "surface-wasm32-web",
  "surface_schema": "tetra.surface.v1",
  "host_abi": "tetra.surface.host-abi.v1",
  "host_evidence": {
    "level": "wasm32-web-browser-canvas-release-v1",
    "backend": "browser-canvas-rgba-accessible",
    "framebuffer": false,
    "real_window": false,
    "native_input": false,
    "browser_canvas": false,
    "browser_input": false,
    "browser_clipboard": false,
    "browser_composition": false,
    "browser_accessibility_snapshot": false,
    "browser_accessibility_mirror": false,
    "user_facing_platform_widgets": false
  },
  "source": "examples/surface/release/surface_release_form.tetra",
  "production_claim": false,
  "blocked_reason": "$escaped_reason"
}
JSON
}

surface_wasm32_web_browser_run() {
	local label="$1"
	local blocked_on_failure="$2"
	shift 2

	if ! command -v timeout >/dev/null 2>&1; then
		local reason="missing timeout command for bounded wasm32-web browser release smoke"
		if [[ "$blocked_on_failure" == "yes" ]]; then
			surface_wasm32_web_browser_blocked_report "$reason"
			echo "blocked: $reason" >&2
			echo "Surface wasm32-web browser-canvas release browser blocked report: $report_path" >&2
		else
			echo "error: $reason" >&2
		fi
		exit 1
	fi

	local timeout_seconds="${SURFACE_BROWSER_SMOKE_TIMEOUT_SECONDS:-120}"
	if ! [[ "$timeout_seconds" =~ ^[0-9]+$ ]] || [[ "$timeout_seconds" -le 0 ]]; then
		echo "error: SURFACE_BROWSER_SMOKE_TIMEOUT_SECONDS must be a positive integer" >&2
		exit 2
	fi

	local log_path="$report_dir/${label}.log"
	set +e
	timeout --kill-after=5s "${timeout_seconds}s" "$@" >"$log_path" 2>&1 &
	surface_wasm32_web_browser_active_pid=$!
	wait "$surface_wasm32_web_browser_active_pid"
	local status=$?
	surface_wasm32_web_browser_active_pid=""
	set -e

	if [[ "$status" -ne 0 ]]; then
		local reason
		if [[ "$status" -eq 124 || "$status" -eq 137 ]]; then
			reason="$label timed out after ${timeout_seconds}s; see $log_path"
		else
			reason="$label failed with exit $status; see $log_path"
		fi
		if [[ "$blocked_on_failure" == "yes" ]]; then
			surface_wasm32_web_browser_blocked_report "$reason"
			echo "blocked: $reason" >&2
			echo "Surface wasm32-web browser-canvas release browser blocked report: $report_path" >&2
		else
			echo "error: $reason" >&2
		fi
		exit "$status"
	fi
}

surface_wasm32_web_browser_run_unlogged() {
	local label="$1"
	shift

	if ! command -v timeout >/dev/null 2>&1; then
		echo "error: missing timeout command for bounded wasm32-web browser release smoke" >&2
		exit 1
	fi
	local timeout_seconds="${SURFACE_BROWSER_SMOKE_TIMEOUT_SECONDS:-120}"
	if ! [[ "$timeout_seconds" =~ ^[0-9]+$ ]] || [[ "$timeout_seconds" -le 0 ]]; then
		echo "error: SURFACE_BROWSER_SMOKE_TIMEOUT_SECONDS must be a positive integer" >&2
		exit 2
	fi

	set +e
	timeout --kill-after=5s "${timeout_seconds}s" "$@" &
	surface_wasm32_web_browser_active_pid=$!
	wait "$surface_wasm32_web_browser_active_pid"
	local status=$?
	surface_wasm32_web_browser_active_pid=""
	set -e

	if [[ "$status" -ne 0 ]]; then
		if [[ "$status" -eq 124 || "$status" -eq 137 ]]; then
			echo "error: $label timed out after ${timeout_seconds}s" >&2
		else
			echo "error: $label failed with exit $status" >&2
		fi
		exit "$status"
	fi
}

surface_wasm32_web_browser_run surface-runtime-smoke yes go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-release-browser --source examples/surface/release/surface_release_form.tetra --report "$report_path"
surface_wasm32_web_browser_run validate-wasm-imports no go run ./tools/cmd/validate-wasm-imports --target wasm32-web "$wasm_path"
surface_wasm32_web_browser_run validate-surface-runtime no go run ./tools/cmd/validate-surface-runtime --report "$report_path" --release wasm32-web-browser
surface_wasm32_web_browser_run write-artifact-hashes no go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
surface_wasm32_web_browser_run_unlogged validate-artifact-hashes go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface wasm32-web browser-canvas release browser runtime smoke report: $report_path"
echo "Surface wasm32-web browser-canvas release browser artifact hashes: $report_dir/artifact-hashes.json"
