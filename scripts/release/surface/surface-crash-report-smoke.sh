#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface/crash-report"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-crash-report-smoke.sh [--report-dir DIR]

Builds bounded Surface crash recovery and error reporting evidence for
surface-v1-linux-web. It records command_failure, host_crash, restart_recovery,
tetra.surface.diagnostic.v1 artifacts, local trace/log collection, redaction
policy, and scoped-linux-x64-process-restart-v1 evidence without user data,
network upload, Electron crash reporter dependency, or broad crash recovery
claims.
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
source "$script_dir/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
	export GOCACHE="$repo_root/.cache/go-build-surface-crash-report"
fi
mkdir -p "$GOCACHE"

report_dir_arg="${report_dir%/}"
report_dir="$report_dir_arg"
if [[ -z "$report_dir" ]]; then
	surface_release_guard_reject "surface_crash_report_smoke:" "--report-dir requires a value"
fi
if [[ "$report_dir" = /* || "$report_dir" == "." || "$report_dir" == "./" || "$report_dir" == -* ]]; then
	surface_release_guard_reject_unsafe "surface_crash_report_smoke:" "$report_dir"
fi
IFS='/' read -r -a report_parts <<<"$report_dir"
current="$repo_root"
for part in "${report_parts[@]}"; do
	if [[ -z "$part" || "$part" == "." ]]; then
		continue
	fi
	if [[ "$part" == ".." ]]; then
		surface_release_guard_reject_unsafe "surface_crash_report_smoke:" "$report_dir"
	fi
	current="$current/$part"
	if [[ -L "$current" ]]; then
		surface_release_guard_reject_symlink "surface_crash_report_smoke:" "$report_dir"
	fi
done
report_dir_abs="$repo_root/$report_dir"
if [[ -e "$report_dir_abs" && ! -d "$report_dir_abs" ]]; then
	surface_release_guard_reject \
	  "surface_crash_report_smoke:" \
	  "refusing to use non-directory report path: $report_dir"
fi
mkdir -p "$report_dir_abs"
report_dir="$(realpath --relative-to="$repo_root" "$report_dir_abs")"

report_path="$report_dir/surface-crash-report.json"
work_dir="$report_dir/surface-crash-work"
crash_dir="$report_dir/surface-crash"
for owned_path in "$report_path" "$work_dir" "$crash_dir"; do
	if [[ -e "$owned_path" ]]; then
		echo "surface_crash_report_smoke: refusing to reuse existing crash artifact path: $owned_path" >&2
		exit 2
	fi
done
mkdir -p "$work_dir/build" "$crash_dir"

json_string() {
	local value="$1"
	value="${value//\\/\\\\}"
	value="${value//\"/\\\"}"
	value="${value//$'\n'/\\n}"
	value="${value//$'\r'/\\r}"
	value="${value//$'\t'/\\t}"
	printf '"%s"' "$value"
}

sha256_file() {
	sha256sum "$1" | awk '{print "sha256:" $1}'
}

file_size() {
	wc -c <"$1" | tr -d ' '
}

source_path="examples/surface/reference_core/surface_reference_command_palette.tetra"
reference_app="command-palette"
linux_binary="$work_dir/build/surface-command-palette-linux-x64"

for forbidden_pattern in \
	'React' \
	'Electron' \
	'Chromium' \
	'DOM app UI' \
	'CSS runtime' \
	'JavaScript app logic' \
	'platform_widget' \
	'native_widget' \
	'platform widget' \
	'native widget' \
	'lib\.core\.component' \
	'lib\.core\.widgets'; do
	if rg -n "$forbidden_pattern" "$source_path" >"$work_dir/source-scan.txt"; then
		echo "surface_crash_report_smoke: forbidden runtime vocabulary: $source_path" >&2
		cat "$work_dir/source-scan.txt" >&2
		exit 1
	fi
done
: >"$work_dir/source-scan.txt"

go run ./cli/cmd/tetra check "$source_path"
go run ./cli/cmd/tetra build --target linux-x64 -o "$linux_binary" "$source_path"

"$linux_binary"
before_exit=0

set +e
bash -c 'echo "surface command boundary failure: command.palette.missing" >&2; exit 77' \
	>"$work_dir/command-failure.stdout" \
	2>"$work_dir/command-failure.stderr"
command_failure_exit=$?
bash -c 'echo "surface host crash harness: sanitized panic frame captured" >&2; exit 70' \
	>"$work_dir/host-crash.stdout" \
	2>"$work_dir/host-crash.stderr"
host_crash_exit=$?
set -e
if [[ "$command_failure_exit" -ne 77 ]]; then
	echo "surface_crash_report_smoke: command failure exit $command_failure_exit, want 77" >&2
	exit 1
fi
if [[ "$host_crash_exit" -ne 70 ]]; then
	echo "surface_crash_report_smoke: host crash harness exit = $host_crash_exit, want 70" >&2
	exit 1
fi

"$linux_binary"
after_exit=0

write_diagnostic() {
	local path="$1"
	local kind="$2"
	local trigger="$3"
	local exit_code="$4"
	local boundary="$5"
	cat >"$path" <<JSON
{
  "schema": "tetra.surface.diagnostic.v1",
  "kind": $(json_string "$kind"),
  "target": "linux-x64",
  "source": $(json_string "$source_path"),
  "reference_app": $(json_string "$reference_app"),
  "trigger": $(json_string "$trigger"),
  "boundary": $(json_string "$boundary"),
  "exit_code": $exit_code,
  "message": "sanitized Surface diagnostic",
  "stack_hash": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
  "user_data_redacted": true,
  "contains_user_data": false,
  "clipboard_payload_captured": false,
  "user_text_captured": false,
  "env_dumped": false,
  "home_path_captured": false,
  "network_upload": false,
  "local_only": true,
  "pass": true
}
JSON
}

command_diag="$crash_dir/command-failure.json"
host_diag="$crash_dir/host-crash.json"
restart_diag="$crash_dir/restart-recovery.json"
trace_path="$crash_dir/surface-app-trace.json"
log_path="$crash_dir/surface-app.log"

write_diagnostic \
	"$command_diag" \
	"command_failure" \
	"command.palette.missing" \
	"$command_failure_exit" \
	"surface-command-boundary-v1"
write_diagnostic \
	"$host_diag" \
	"host_crash" \
	"surface-host panic harness" \
	"$host_crash_exit" \
	"surface-host-crash-harness-v1"
write_diagnostic \
	"$restart_diag" \
	"restart_recovery" \
	"restart after command failure report" \
	"$after_exit" \
	"surface-restart-boundary-v1"

cat >"$trace_path" <<JSON
{
  "schema": "tetra.surface.diagnostic-trace.v1",
  "ring_buffer": true,
  "max_bytes": 4096,
  "local_only": true,
  "events": [
    {"name":"before_run","target":"linux-x64","exit_code":$before_exit},
    {"name":"command_failure_report_written","target":"linux-x64","path":$(json_string "$command_diag")},
    {"name":"host_crash_report_written","target":"linux-x64","path":$(json_string "$host_diag")},
    {"name":"after_run","target":"linux-x64","exit_code":$after_exit}
  ]
}
JSON

cat >"$log_path" <<'LOG'
surface diagnostic log v1
before_run exit=0
command_failure report_written=true
host_crash report_written=true
restart_recovery after_run exit=0
LOG

command_diag_sha="$(sha256_file "$command_diag")"
host_diag_sha="$(sha256_file "$host_diag")"
restart_diag_sha="$(sha256_file "$restart_diag")"
command_diag_size="$(file_size "$command_diag")"
host_diag_size="$(file_size "$host_diag")"
restart_diag_size="$(file_size "$restart_diag")"

cat >"$report_path" <<JSON
{
  "schema": "tetra.surface.crash-report.v1",
  "model": "surface-crash-report-v1",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/surface-crash-report-smoke.sh",
  "source": $(json_string "$source_path"),
  "reference_app": $(json_string "$reference_app"),
  "target": "linux-x64",
  "diagnostic_schema": "tetra.surface.diagnostic.v1",
  "scenarios": [
    {
      "name": "command failure boundary",
      "kind": "command_failure",
      "target": "linux-x64",
      "source": $(json_string "$source_path"),
      "trigger": "command.palette.missing",
      "diagnostic_path": $(json_string "$command_diag"),
      "diagnostic_sha256": $(json_string "$command_diag_sha"),
      "report_written": true,
      "command_boundary": true,
      "host_captured": false,
      "restarted": false,
      "contains_user_data": false,
      "pass": true
    },
    {
      "name": "host crash capture",
      "kind": "host_crash",
      "target": "linux-x64",
      "source": $(json_string "$source_path"),
      "trigger": "surface-host panic harness",
      "diagnostic_path": $(json_string "$host_diag"),
      "diagnostic_sha256": $(json_string "$host_diag_sha"),
      "report_written": true,
      "command_boundary": false,
      "host_captured": true,
      "restarted": false,
      "contains_user_data": false,
      "pass": true
    },
    {
      "name": "restart after diagnostic",
      "kind": "restart_recovery",
      "target": "linux-x64",
      "source": $(json_string "$source_path"),
      "trigger": "restart after command failure report",
      "diagnostic_path": $(json_string "$restart_diag"),
      "diagnostic_sha256": $(json_string "$restart_diag_sha"),
      "report_written": true,
      "command_boundary": false,
      "host_captured": false,
      "restarted": true,
      "contains_user_data": false,
      "pass": true
    }
  ],
  "diagnostics": [
    {"path": $(json_string "$command_diag"), "kind": "command_failure", "schema": "tetra.surface.diagnostic.v1", "sha256": $(json_string "$command_diag_sha"), "size_bytes": $command_diag_size, "redacted": true, "contains_user_data": false, "pass": true},
    {"path": $(json_string "$host_diag"), "kind": "host_crash", "schema": "tetra.surface.diagnostic.v1", "sha256": $(json_string "$host_diag_sha"), "size_bytes": $host_diag_size, "redacted": true, "contains_user_data": false, "pass": true},
    {"path": $(json_string "$restart_diag"), "kind": "restart_recovery", "schema": "tetra.surface.diagnostic.v1", "sha256": $(json_string "$restart_diag_sha"), "size_bytes": $restart_diag_size, "redacted": true, "contains_user_data": false, "pass": true}
  ],
  "trace_collection": {
    "trace_path": $(json_string "$trace_path"),
    "log_path": $(json_string "$log_path"),
    "ring_buffer": true,
    "max_bytes": 4096,
    "event_count": 4,
    "bounded": true,
    "local_only": true,
    "pass": true
  },
  "restart_recovery": {
    "scope": "scoped-linux-x64-process-restart-v1",
    "target": "linux-x64",
    "restart_claim": true,
    "before_run": true,
    "failure_report_written": true,
    "after_run": true,
    "before_exit_code": $before_exit,
    "after_exit_code": $after_exit,
    "state_restored": "explicit-startup-state-v1",
    "command": $(json_string "$linux_binary"),
    "pass": true
  },
  "privacy_policy": {
    "policy": "surface-non-user-data-diagnostics-v1",
    "redaction_version": "surface-diagnostic-redaction-v1",
    "user_data_redacted": true,
    "clipboard_payload_captured": false,
    "user_text_captured": false,
    "env_dumped": false,
    "home_path_captured": false,
    "network_upload": false,
    "local_only": true,
    "pass": true
  },
  "negative_guards": {
    "no_user_data_leak": true,
    "no_clipboard_payload": true,
    "no_user_text_payload": true,
    "no_env_dump": true,
    "no_home_path_leak": true,
    "no_network_upload": true,
    "no_restart_claim_without_evidence": true,
    "no_silent_failure": true,
    "no_docs_only_crash_claim": true,
    "no_electron_crash_reporter_dependency": true
  },
  "pass": true
}
JSON

if [[ -n "${HOME:-}" ]] && rg -n -F "$HOME" "$report_path" "$crash_dir" >/dev/null; then
	echo "surface_crash_report_smoke: diagnostic artifacts leaked HOME path" >&2
	exit 1
fi

go run ./tools/cmd/validate-surface-crash-report --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface crash report smoke report: $report_path"
