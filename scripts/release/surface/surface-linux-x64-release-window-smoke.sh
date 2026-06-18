#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-linux-x64-release-window-smoke.sh [--report-dir DIR]

Runs the linux-x64 Tetra Surface release-window smoke.
The gate builds examples/surface/release/surface_release_form.tetra, opens a real Linux window
through the wayland-shm-rgba-release-v1 backend, records
linux-x64-release-window-v1 evidence for pointer/key/text/resize/close input,
clipboard, composition, and accessibility bridge probes, rejects memfd starter
promotion, and rejects old real-window smoke promotion as release evidence.
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
report_path="$report_dir/surface-linux-x64-release-window.json"

surface_linux_x64_release_window_active_pid=""

surface_linux_x64_release_window_cleanup() {
	if [[ -n "${surface_linux_x64_release_window_active_pid:-}" ]] &&
		kill -0 "$surface_linux_x64_release_window_active_pid" 2>/dev/null; then
		kill -TERM "$surface_linux_x64_release_window_active_pid" 2>/dev/null || true
		sleep 1
		kill -KILL "$surface_linux_x64_release_window_active_pid" 2>/dev/null || true
	fi
}

trap surface_linux_x64_release_window_cleanup EXIT

surface_linux_x64_release_window_json_escape() {
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
	escaped_reason="$(surface_linux_x64_release_window_json_escape "$reason")"
	cat >"$report_path" <<JSON
{
  "schema": "tetra.surface.runtime.v1",
  "status": "blocked",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "surface-linux-x64",
  "source": "examples/surface/release/surface_release_form.tetra",
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

surface_linux_x64_release_window_run() {
	local label="$1"
	local blocked_on_failure="$2"
	shift 2

	if ! command -v timeout >/dev/null 2>&1; then
		local reason="missing timeout command for bounded linux-x64 release-window smoke"
		if [[ "$blocked_on_failure" == "yes" ]]; then
			write_blocked_report "$reason"
			echo "Surface linux-x64 release-window blocked: $reason" >&2
			echo "Surface linux-x64 release-window blocked report: $report_path" >&2
		else
			echo "error: $reason" >&2
		fi
		exit 1
	fi

	local timeout_seconds="${SURFACE_LINUX_RELEASE_WINDOW_TIMEOUT_SECONDS:-120}"
	if ! [[ "$timeout_seconds" =~ ^[0-9]+$ ]] || [[ "$timeout_seconds" -le 0 ]]; then
		echo "error: SURFACE_LINUX_RELEASE_WINDOW_TIMEOUT_SECONDS must be a positive integer" >&2
		exit 2
	fi

	local log_path="$report_dir/surface-linux-x64-release-window-${label}.log"
	set +e
	timeout --kill-after=5s "${timeout_seconds}s" "$@" >"$log_path" 2>&1 &
	surface_linux_x64_release_window_active_pid=$!
	wait "$surface_linux_x64_release_window_active_pid"
	local status=$?
	surface_linux_x64_release_window_active_pid=""
	set -e

	if [[ "$status" -ne 0 ]]; then
		local reason
		if [[ "$status" -eq 124 || "$status" -eq 137 ]]; then
			reason="$label timed out after ${timeout_seconds}s; see $log_path"
		else
			reason="$label failed with exit $status; see $log_path"
		fi
		if [[ "$blocked_on_failure" == "yes" ]]; then
			write_blocked_report "$reason"
			echo "Surface linux-x64 release-window blocked: $reason" >&2
			echo "Surface linux-x64 release-window blocked report: $report_path" >&2
		else
			echo "error: $reason" >&2
		fi
		tail -n 80 "$log_path" >&2 || true
		exit "$status"
	fi
}

surface_linux_x64_release_window_run_unlogged() {
	local label="$1"
	shift

	if ! command -v timeout >/dev/null 2>&1; then
		echo "error: missing timeout command for bounded linux-x64 release-window smoke" >&2
		exit 1
	fi

	local timeout_seconds="${SURFACE_LINUX_RELEASE_WINDOW_TIMEOUT_SECONDS:-120}"
	if ! [[ "$timeout_seconds" =~ ^[0-9]+$ ]] || [[ "$timeout_seconds" -le 0 ]]; then
		echo "error: SURFACE_LINUX_RELEASE_WINDOW_TIMEOUT_SECONDS must be a positive integer" >&2
		exit 2
	fi

	set +e
	timeout --kill-after=5s "${timeout_seconds}s" "$@" &
	surface_linux_x64_release_window_active_pid=$!
	wait "$surface_linux_x64_release_window_active_pid"
	local status=$?
	surface_linux_x64_release_window_active_pid=""
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

if [[ -z "${WAYLAND_DISPLAY:-}" && -z "${DISPLAY:-}" ]]; then
	reason="missing WAYLAND_DISPLAY or DISPLAY for linux-x64 release-window target host"
	write_blocked_report "$reason"
	echo "Surface linux-x64 release-window blocked: $reason" >&2
	echo "Surface linux-x64 release-window blocked report: $report_path" >&2
	exit 1
fi

surface_linux_x64_release_window_run "runtime-smoke" "yes" go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-release-window --source examples/surface/release/surface_release_form.tetra --report "$report_path"
surface_linux_x64_release_window_run "validate-runtime" "no" go run ./tools/cmd/validate-surface-runtime --report "$report_path" --release surface-v1
surface_linux_x64_release_window_run "write-artifact-hashes" "no" go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
surface_linux_x64_release_window_run_unlogged "validate-artifact-hashes" go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface linux-x64 release-window runtime smoke report: $report_path"
echo "Surface linux-x64 release-window artifact hashes: $report_dir/artifact-hashes.json"
