#!/usr/bin/env bash
set -euo pipefail

report_dir=""
source_path="examples/surface/runtime/surface_window_counter.tetra"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--report-dir)
		report_dir="$2"
		shift 2
		;;
	--source)
		source_path="$2"
		shift 2
		;;
	-h | --help)
		cat <<'USAGE'
usage: surface-linux-x64-native-host-smoke.sh --report-dir DIR [--source PATH]

Runs the real linux-x64 Wayland native Surface host path. While the window is
open, click the counter, press a key, then close the window. Missing real
pointer/key/close evidence makes the final validator fail.
USAGE
		exit 0
		;;
	*)
		echo "error: unknown argument $1" >&2
		exit 2
		;;
	esac
done

if [[ -z "$report_dir" ]]; then
	echo "error: --report-dir is required" >&2
	exit 2
fi
if [[ -z "${WAYLAND_DISPLAY:-}" || -z "${XDG_RUNTIME_DIR:-}" ]]; then
	printf '%s%s\n' \
		"BLOCKED: WAYLAND_DISPLAY and XDG_RUNTIME_DIR are required for native " \
		"Surface host evidence" >&2
	exit 3
fi

artifact_dir="$report_dir/artifacts"
runtime_report="$report_dir/surface-linux-x64-native-host.json"
artifact_hashes="$report_dir/artifact-hashes.json"
host_report="$artifact_dir/surface-host-report.json"
app_bin="$artifact_dir/surface-window-counter"
host_bin="$artifact_dir/tetra-surface-host-wayland"

mkdir -p "$artifact_dir"

go build -buildvcs=false -o "$host_bin" ./tools/cmd/tetra-surface-host

echo "Native Surface host gate: click the counter, press a key, then close the Wayland window." >&2
set +e
TETRA_SURFACE_HOST_BIN="$host_bin" \
	go run -buildvcs=false ./cli/cmd/tetra run \
	--target linux-x64 \
	--surface-host wayland \
	--surface-host-report "$host_report" \
	-o "$app_bin" \
	"$source_path"
app_exit=$?
set -e

go run -buildvcs=false ./tools/cmd/surface-native-host-report \
	--report "$runtime_report" \
	--source "$source_path" \
	--artifact-dir "$artifact_dir" \
	--component-app "$app_bin" \
	--host-binary "$host_bin" \
	--host-report "$host_report" \
	--app-exit-code "$app_exit" \
	--host-exit-code 0 \
	--build-command "tetra build --target linux-x64 $source_path -o $app_bin" \
	--app-command "$app_bin --surface-host wayland" \
	--host-command "$host_bin --backend wayland --socket <private> --report $host_report"

go run -buildvcs=false ./tools/cmd/validate-surface-runtime \
	--release linux-x64-native-host \
	--report "$runtime_report"

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$artifact_hashes"

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes \
	--manifest "$artifact_hashes"

echo "Surface linux-x64 native host runtime report: $runtime_report"
echo "Surface linux-x64 native host artifact hashes: $artifact_hashes"
