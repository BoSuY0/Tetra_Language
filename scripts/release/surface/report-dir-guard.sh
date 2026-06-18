#!/usr/bin/env bash
set -euo pipefail

surface_release_guard_reject() {
	local prefix="$1"
	local message="$2"
	echo "${prefix} ${message}" >&2
	exit 2
}

surface_release_guard_reject_unsafe() {
	local prefix="$1"
	local message="$2"
	echo "${prefix} refusing unsafe report directory: ${message}" >&2
	echo "${prefix} choose a fresh repo-relative --report-dir" >&2
	exit 2
}

surface_release_guard_reject_symlink() {
	local prefix="$1"
	local report_dir="$2"
	echo "${prefix} refusing to use symlink report directory: $report_dir" >&2
	echo "${prefix} choose a real fresh --report-dir" >&2
	exit 2
}

surface_release_require_fresh_report_dir() {
	local report_dir="$1"
	local repo_root="$2"
	local prefix="${3:-surface_release_gate:}"

	if [[ -z "$report_dir" ]]; then
		surface_release_guard_reject "$prefix" "--report-dir requires a value"
	fi
	if [[ "$report_dir" = /* ]]; then
		surface_release_guard_reject_unsafe "$prefix" "absolute paths are not accepted ($report_dir)"
	fi
	if [[ "$report_dir" == "." || "$report_dir" == "./" ]]; then
		surface_release_guard_reject_unsafe "$prefix" "repo root is not a release report directory"
	fi
	if [[ "$report_dir" == -* ]]; then
		surface_release_guard_reject_unsafe "$prefix" "dash-prefixed paths are not accepted ($report_dir)"
	fi

	local part=""
	local current="$repo_root"
	IFS='/' read -r -a report_parts <<<"$report_dir"
	for part in "${report_parts[@]}"; do
		if [[ -z "$part" || "$part" == "." ]]; then
			continue
		fi
		if [[ "$part" == ".." ]]; then
			surface_release_guard_reject_unsafe "$prefix" "parent traversal is not accepted ($report_dir)"
		fi
		current="$current/$part"
		if [[ -L "$current" ]]; then
			surface_release_guard_reject_symlink "$prefix" "$report_dir"
		fi
	done

	local report_path="$repo_root/$report_dir"
	if [[ -L "$report_path" ]]; then
		surface_release_guard_reject_symlink "$prefix" "$report_dir"
	fi
	if [[ -e "$report_path" && ! -d "$report_path" ]]; then
		surface_release_guard_reject "$prefix" "refusing to use non-directory report path: $report_dir"
	fi
	if [[ -d "$report_path" ]]; then
		local first_entry=""
		first_entry="$(find -- "$report_path" -mindepth 1 -print -quit)"
		if [[ -n "$first_entry" ]]; then
			echo "${prefix} refusing to reuse non-empty report directory: $report_dir" >&2
			echo "${prefix} choose a fresh --report-dir so stale reports cannot be reused" >&2
			exit 2
		fi
	fi

	mkdir -p "$report_path"
	(cd "$report_path" && pwd)
}
