#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-release-v1"
original_args=("$@")

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/release-gate.sh [--report-dir DIR]

Runs the final Tetra Surface v1 release gate for surface-v1-linux-web.
It requires headless release evidence, linux-x64 real-window release evidence,
wasm32-web browser-canvas release evidence, strict Surface v1 validators,
artifact hash integrity, and docs/generated manifest state.

Surface v1 release gate must fail, not skip, when Chromium-compatible browser,
Linux Wayland/display, accessibility probe, or clipboard harness evidence is
unavailable.
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
gate_contract="scripts/release/surface/contracts/surface-release-v1.json"
gate_contract_id="surface-release-v1"
source "$script_dir/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
	export GOCACHE="$repo_root/.cache/go-build-surface-release"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
	export GOTMPDIR="$repo_root/.cache/go-tmp-surface-release"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"
report_dir_arg="${report_dir%/}"
report_dir="$report_dir_arg"
report_dir="$(
	surface_release_require_fresh_report_dir \
		"$report_dir" \
		"$repo_root" \
		"surface_release_gate:"
)"

if [[ "${TETRA_RUN_GATE_CONTRACT_EXEC:-}" != "1" ]]; then
	go run ./tools/cmd/run-gate \
		--contract "$gate_contract" \
		--report-dir "$report_dir_arg"
	exit $?
fi

if [[ "${TETRA_RUN_GATE_CONTRACT_ID:-}" != "$gate_contract_id" ]]; then
	echo "surface_release_gate: refusing guarded execution for contract" \
		"${TETRA_RUN_GATE_CONTRACT_ID:-<unset>}; want $gate_contract_id" >&2
	exit 2
fi
if [[ -n "${TETRA_RUN_GATE_REPORT_DIR:-}" &&
	"${TETRA_RUN_GATE_REPORT_DIR:-}" != "$report_dir_arg" ]]; then
	echo "surface_release_gate: refusing guarded execution for report dir" \
		"${TETRA_RUN_GATE_REPORT_DIR}; want $report_dir_arg" >&2
	exit 2
fi

go run ./tools/cmd/run-gate \
	--contract "$gate_contract" \
	--report-dir "$report_dir_arg" \
	--dry-run >/dev/null
block_system_report_dir="$report_dir_arg/block-system"
morph_report_dir="$report_dir_arg/morph"

format_command() {
	local formatted=""
	local quoted=""
	local arg
	for arg in "$@"; do
		printf -v quoted "%q" "$arg"
		if [[ -z "$formatted" ]]; then
			formatted="$quoted"
		else
			formatted+=" $quoted"
		fi
	done
	printf "%s" "$formatted"
}

json_string() {
	local value="$1"
	value="${value//\\/\\\\}"
	value="${value//\"/\\\"}"
	value="${value//$'\n'/\\n}"
	value="${value//$'\r'/\\r}"
	value="${value//$'\t'/\\t}"
	printf '"%s"' "$value"
}

surface_gate_scripts=(
	scripts/release/surface/surface-headless-release-smoke.sh
	scripts/release/surface/surface-headless-release-text-input-smoke.sh
	scripts/release/surface/surface-headless-release-toolkit-smoke.sh
	scripts/release/surface/surface-headless-release-accessibility-smoke.sh
	scripts/release/surface/surface-headless-app-model-smoke.sh
	scripts/release/surface/surface-linux-x64-release-window-smoke.sh
	scripts/release/surface/surface-linux-x64-release-app-shell-smoke.sh
	scripts/release/surface/surface-dev-workflow-smoke.sh
	scripts/release/surface/surface-inspector-smoke.sh
	scripts/release/surface/surface-template-smoke.sh
	scripts/release/surface/surface-reference-apps-smoke.sh
	scripts/release/surface/surface-package-smoke.sh
	scripts/release/surface/surface-crash-report-smoke.sh
	scripts/release/surface/surface-i18n-smoke.sh
	scripts/release/surface/surface-widget-migration-smoke.sh
	scripts/release/surface/surface-linux-x64-release-text-input-smoke.sh
	scripts/release/surface/surface-linux-x64-release-toolkit-smoke.sh
	scripts/release/surface/surface-linux-x64-release-accessibility-smoke.sh
	scripts/release/surface/surface-wasm32-web-release-browser-smoke.sh
	scripts/release/surface/surface-wasm32-web-release-text-input-smoke.sh
	scripts/release/surface/surface-wasm32-web-release-toolkit-smoke.sh
	scripts/release/surface/surface-wasm32-web-release-accessibility-smoke.sh
	scripts/release/surface/block-system-gate.sh
	scripts/release/surface/morph-gate.sh
)
surface_gate_report_dirs=(
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir_arg"
	"$report_dir_arg"
	"$report_dir_arg"
	"$report_dir_arg"
	"$report_dir_arg"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$report_dir"
	"$block_system_report_dir"
	"$morph_report_dir"
)

for i in "${!surface_gate_scripts[@]}"; do
	bash "${surface_gate_scripts[$i]}" --report-dir "${surface_gate_report_dirs[$i]}"
done

summary_path="$report_dir/surface-release-summary.json"
git_head="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
git_dirty=false
if ! git diff --quiet 2>/dev/null ||
	! git diff --cached --quiet 2>/dev/null ||
	[[ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]]; then
	git_dirty=true
fi
version="$(go list -m 2>/dev/null || echo tetra_language)"
host_os="$(go env GOOS 2>/dev/null || uname -s)"
host_arch="$(go env GOARCH 2>/dev/null || uname -m)"
generated_at_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
formatted_args="$(format_command "${original_args[@]}")"
command_line="bash scripts/release/surface/release-gate.sh"
if [[ -n "$formatted_args" ]]; then
	command_line+=" $formatted_args"
fi
macos_target_host_status_path="$report_dir/surface-macos-x64-target-host-status.json"
windows_target_host_status_path="$report_dir/surface-windows-x64-target-host-status.json"

cat >"$macos_target_host_status_path" <<JSON
{
  "schema": "tetra.surface.target-host-status.v1",
  "target": "macos-x64",
  "status": "unsupported",
  "tier": "UNSUPPORTED",
  "release_scope": "surface-v1-linux-web",
  "source": "scripts/release/surface/release-gate.sh",
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "reason": "macOS target-host Surface v1 evidence is missing; build-only evidence is excluded",
  "production_claim": false,
  "experimental": false,
  "target_host_evidence": false,
  "build_only_evidence": false,
  "build_only_promotion": false,
  "linux_substitute": false,
  "ci_artifact_required": true,
  "required_evidence": {
    "real_window": false,
    "native_input": false,
    "clipboard": false,
    "dpi_scaling": false,
    "accessibility_snapshot": false,
    "app_shell": false
  },
  "unsupported_claims": [
    "macos-real-window-surface",
    "macos-production-surface-nonclaim",
    "macos-target-host-runtime",
    "build-only-macos-surface-runtime",
    "linux-substitute-macos-surface-runtime"
  ],
  "negative_guards": {
    "no_linux_substitute": true,
    "no_build_only_promotion": true,
    "no_production_claim": true,
    "no_docs_only_evidence": true,
    "no_copied_report": true,
    "ci_artifact_required": true
  }
}
JSON

cat >"$windows_target_host_status_path" <<JSON
{
  "schema": "tetra.surface.target-host-status.v1",
  "target": "windows-x64",
  "status": "unsupported",
  "tier": "UNSUPPORTED",
  "release_scope": "surface-v1-linux-web",
  "source": "scripts/release/surface/release-gate.sh",
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "reason": "Windows target-host Surface v1 evidence is missing; build-only evidence is excluded",
  "production_claim": false,
  "experimental": false,
  "target_host_evidence": false,
  "build_only_evidence": false,
  "build_only_promotion": false,
  "linux_substitute": false,
  "ci_artifact_required": true,
  "required_evidence": {
    "real_window": false,
    "native_input": false,
    "clipboard": false,
    "dpi_scaling": false,
    "accessibility_snapshot": false,
    "app_shell": false
  },
  "unsupported_claims": [
    "windows-real-window-surface",
    "windows-production-surface-nonclaim",
    "windows-target-host-runtime",
    "build-only-windows-surface-runtime",
    "linux-substitute-windows-surface-runtime"
  ],
  "negative_guards": {
    "no_linux_substitute": true,
    "no_build_only_promotion": true,
    "no_production_claim": true,
    "no_docs_only_evidence": true,
    "no_copied_report": true,
    "ci_artifact_required": true
  }
}
JSON

cat >"$summary_path" <<JSON
{
  "schema": "tetra.surface.release.v1",
  "release_scope": "surface-v1-linux-web",
  "status": "current",
  "production_claim": true,
  "experimental": false,
  "producer": "scripts/release/surface/release-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "supported_targets": [
    "headless",
    "linux-x64",
    "wasm32-web"
  ],
  "runtime_targets": [
    "linux-x64",
    "wasm32-web"
  ],
  "test_targets": [
    "headless"
  ],
  "unsupported_targets": [
    "macos-x64",
    "windows-x64",
    "wasm32-wasi"
  ],
  "host_abi": "tetra.surface.host.v1",
  "toolkit": "production-widgets-v1",
  "text_input": "production-text-input-v1",
  "clipboard": "clipboard-text-v1",
  "ime": "composition-baseline-v1",
  "accessibility": "platform-bridge-v1",
  "app_model": "explicit-command-reducer-v1",
  "linux_app_shell": "linux-app-shell-subset-v1",
  "app_shell_features": "electron-feature-ledger-v1",
  "security_permissions": "surface-security-permission-v1",
  "performance_budget": "surface-performance-budget-v1",
  "developer_fast_loop": "surface-dev-workflow-v1",
  "inspector": "surface-inspector-v1",
  "project_templates": "surface-template-smoke-v1",
  "reference_apps": "surface-reference-app-suite-v1",
  "surface_package": "surface-package-v1",
  "crash_reporting": "surface-crash-report-v1",
  "i18n_localization": "surface-i18n-v1",
  "widget_migration": "surface-widget-migration-v1",
  "browser_surface": "browser-canvas-release-v1",
  "linux_surface": "linux-x64-release-window-v1",
  "block_system": "block-system",
  "block_system_gate": "tetra.surface.block-system.gate.v1",
  "morph": "morph-capsule",
  "morph_gate": "tetra.surface.morph.gate.v1",
  "artifact_hashes_validated": true,
  "legacy_sidecars": false,
  "dom_ui": false,
  "user_js": false,
  "platform_widgets": false
}
JSON

required_reports=(
	"surface-headless-release.json"
	"surface-headless-release-text-input.json"
	"surface-headless-release-toolkit.json"
	"surface-headless-release-accessibility.json"
	"surface-headless-app-model.json"
	"surface-linux-x64-release-window.json"
	"surface-linux-x64-release-app-shell.json"
	"surface-dev-workflow.json"
	"surface-inspector.json"
	"surface-template-smoke.json"
	"surface-reference-apps.json"
	"surface-package.json"
	"surface-crash-report.json"
	"surface-i18n.json"
	"surface-widget-migration.json"
	"surface-linux-x64-release-text-input.json"
	"surface-linux-x64-release-toolkit.json"
	"surface-linux-x64-release-accessibility.json"
	"surface-macos-x64-target-host-status.json"
	"surface-windows-x64-target-host-status.json"
	"surface-wasm32-web-release-browser.json"
	"surface-wasm32-web-release-text-input.json"
	"surface-wasm32-web-release-toolkit.json"
	"surface-wasm32-web-release-accessibility.json"
	"block-system/surface-block-system-gate-summary.json"
	"block-system/headless/surface-headless-block-system.json"
	"block-system/headless/surface-block-examples.json"
	"block-system/linux-x64-real-window/surface-block-system-linux-x64.json"
	"block-system/wasm32-web-browser-canvas/surface-block-system-wasm32-web.json"
	"morph/surface-morph-gate-summary.json"
	"morph/headless/surface-headless-morph.json"
	"surface-release-summary.json"
	"artifact-hashes.json"
)
for report in "${required_reports[@]}"; do
	if [[ "$report" == "artifact-hashes.json" ]]; then
		continue
	fi
	if [[ ! -s "$report_dir/$report" ]]; then
		echo "error: required Surface v1 release report missing or empty: $report_dir/$report" >&2
		exit 1
	fi
done

go run ./tools/cmd/validate-surface-runtime \
	--report "$report_dir/surface-release-summary.json" \
	--release surface-v1
go run ./tools/cmd/validate-surface-security-report \
	--report "$report_dir/surface-linux-x64-release-app-shell.json"
go run ./tools/cmd/validate-surface-performance-budget \
	--report "$report_dir/surface-linux-x64-release-app-shell.json"
go run ./tools/cmd/validate-surface-dev-workflow --report "$report_dir/surface-dev-workflow.json"
go run ./tools/cmd/validate-surface-inspector --report "$report_dir/surface-inspector.json"
go run ./tools/cmd/validate-surface-template-smoke \
	--report "$report_dir/surface-template-smoke.json"
go run ./tools/cmd/validate-surface-reference-apps \
	--report "$report_dir/surface-reference-apps.json"
go run ./tools/cmd/validate-surface-package --report "$report_dir/surface-package.json"
go run ./tools/cmd/validate-surface-crash-report --report "$report_dir/surface-crash-report.json"
go run ./tools/cmd/validate-surface-i18n --report "$report_dir/surface-i18n.json"
go run ./tools/cmd/validate-surface-widget-migration \
	--report "$report_dir/surface-widget-migration.json"
go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-surface-release-state \
	--report-dir "$report_dir" \
	--expected-status current \
	--scope surface-v1-linux-web \
	--manifest docs/generated/manifest.json
go run ./tools/cmd/validate-surface-claims --root "$repo_root" --report-dir "$report_dir"

echo "Surface v1 release gate reports: $report_dir"
echo "Surface v1 release gate summary: $report_dir/surface-release-summary.json"
echo "Surface v1 release gate artifact hashes: $report_dir/artifact-hashes.json"
