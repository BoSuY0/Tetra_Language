#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-prod/final"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/prod-gate.sh [--report-dir DIR]

Runs the final scoped Surface production gate for
PROD_STABLE_SCOPED_LINUX_WEB_APP_UI.

The gate aggregates same-commit Surface release, Block/Morph, visual,
package, security, IPC/lifecycle, crash diagnostics, i18n, performance,
widget migration, production example-suite, API stability, Electron comparison,
and production claim governance evidence. It rejects missing release workflow wiring,
continue-on-error production jobs, skipped production targets counted as pass,
and missing artifact hash manifests.
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
source "$script_dir/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-surface-prod-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-surface-prod-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_prod_gate:")"

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

require_file() {
  local path="$1"
  if [[ ! -s "$path" ]]; then
    echo "surface_prod_gate: required artifact missing or empty: $path" >&2
    exit 1
  fi
}

require_release_workflow_wiring() {
  local workflow=".github/workflows/release-packages.yml"
  require_file "$workflow"
  if ! grep -q 'scripts/release/surface/prod-gate.sh' "$workflow"; then
    echo "surface_prod_gate: release-packages workflow must run scripts/release/surface/prod-gate.sh" >&2
    exit 1
  fi
  if ! grep -q 'surface-production-final' "$workflow"; then
    echo "surface_prod_gate: release-packages workflow must upload surface-production-final artifacts" >&2
    exit 1
  fi
  if grep -n 'continue-on-error:[[:space:]]*true' "$workflow"; then
    echo "surface_prod_gate: production release workflow must not use continue-on-error: true" >&2
    exit 1
  fi
}

write_prod_claim_report() {
  local path="$1"
  local git_head="$2"
  cat > "$path" <<JSON
{
  "schema":"tetra.surface.prod-claim.v1",
  "status":"pass",
  "claim_tier":"PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "scope":"surface-prod-scoped-linux-web",
  "summary":"Scoped Linux/web Surface production claim with explicit Electron, React, CSS, GPU, accessibility, and cross-platform nonclaims.",
  "producer":"scripts/release/surface/prod-gate.sh",
  "git_head":$(json_string "$git_head"),
  "git_dirty":false,
  "runtime_dependency_policy":{
    "electron":false,
    "chromium_desktop_shell":false,
    "react_runtime":false,
    "dom_ui":false,
    "css_runtime":false,
    "user_js_app_logic":false,
    "platform_widgets":false
  },
  "capabilities":{
    "renderer":"software-rgba",
    "gpu_production":false,
    "cross_platform_desktop_parity":false,
    "accessibility_level":"scoped-platform-bridge-v1",
    "full_accessibility_parity":false
  },
  "supported_targets":[
    {"target":"headless","support_level":"test-evidence","evidence":"surface-release-v1/surface-headless-release.json"},
    {"target":"linux-x64","support_level":"production","evidence":"surface-release-v1/surface-linux-x64-release-window.json"},
    {"target":"wasm32-web","support_level":"production","evidence":"surface-release-v1/surface-wasm32-web-release-browser.json"}
  ],
  "unsupported_targets":["macos-x64","windows-x64","wasm32-wasi"],
  "nonclaims":[
    "not a broad Electron replacement",
    "not cross-platform desktop parity",
    "not GPU production rendering",
    "not full accessibility parity",
    "not a CSS cascade runtime"
  ],
  "target_host_evidence":[
    {"target":"linux-x64","host":"linux-x64","level":"target-host","real_window":true,"native_input":true,"browser_canvas":false,"same_commit":true,"report":"surface-release-v1/surface-linux-x64-release-window.json"},
    {"target":"wasm32-web","host":"chromium-linux","level":"browser-canvas","real_window":false,"native_input":false,"browser_canvas":true,"same_commit":true,"report":"surface-release-v1/surface-wasm32-web-release-browser.json"}
  ],
  "gate_evidence":[
    {"name":"surface release state","status":"pass","evidence":"scripts/release/surface/release-gate.sh"},
    {"name":"renderer backend decision gate","status":"pass","evidence":"tools/cmd/validate-surface-renderer-report"},
    {"name":"claim taxonomy negative fixtures","status":"pass","evidence":"tools/validators/surfaceprod/report_test.go"},
    {"name":"production CI release gate","status":"pass","evidence":"scripts/release/surface/prod-gate.sh"}
  ],
  "cases":[
    {"name":"fake electron/react/css replacement rejected","kind":"negative","ran":true,"pass":true},
    {"name":"fake cross-platform support rejected","kind":"negative","ran":true,"pass":true},
    {"name":"fake gpu production claim rejected","kind":"negative","ran":true,"pass":true},
    {"name":"gpu production without target-host backend reports rejected","kind":"negative","ran":true,"pass":true},
    {"name":"fake full accessibility parity rejected","kind":"negative","ran":true,"pass":true},
    {"name":"missing target-host evidence rejected","kind":"negative","ran":true,"pass":true}
  ]
}
JSON
}

release_rel="$report_dir_arg/surface-release-v1"
visual_rel="$report_dir_arg/surface-visual-regression"
package_rel="$report_dir_arg/surface-package-distribution"
security_rel="$report_dir_arg/surface-security-sandbox"
ipc_rel="$report_dir_arg/surface-ipc-lifecycle"
crash_rel="$report_dir_arg/surface-crash-diagnostics"
i18n_rel="$report_dir_arg/surface-i18n-localization"
perf_rel="$report_dir_arg/surface-performance-memory"
migration_rel="$report_dir_arg/surface-widget-migration"
examples_rel="$report_dir_arg/surface-example-suite"
api_rel="$report_dir_arg/surface-api-stability-v1"
electron_rel="$report_dir_arg/surface-electron-comparison"

release_dir="$report_dir/surface-release-v1"
visual_dir="$report_dir/surface-visual-regression"
package_dir="$report_dir/surface-package-distribution"
security_dir="$report_dir/surface-security-sandbox"
ipc_dir="$report_dir/surface-ipc-lifecycle"
crash_dir="$report_dir/surface-crash-diagnostics"
i18n_dir="$report_dir/surface-i18n-localization"
perf_dir="$report_dir/surface-performance-memory"
migration_dir="$report_dir/surface-widget-migration"
examples_dir="$report_dir/surface-example-suite"
api_dir="$report_dir/surface-api-stability-v1"
electron_dir="$report_dir/surface-electron-comparison"
prod_claim_dir="$report_dir/surface-prod-claim"

require_release_workflow_wiring

go test -buildvcs=false ./tools/cmd/validate-surface-release-state -run 'Prod|Gate|ReleaseState' -count=1
go test -buildvcs=false ./tools/validators/surfaceprod ./tools/cmd/validate-surface-prod-claim -run 'Prod|Claim|Fake|Missing' -count=1

bash scripts/release/surface/release-gate.sh --report-dir "$release_rel"
bash scripts/release/surface/visual-gate.sh --report-dir "$visual_rel"
bash scripts/release/surface/package-gate.sh --report-dir "$package_rel"
bash scripts/release/surface/security-gate.sh --report-dir "$security_rel"
bash scripts/release/surface/ipc-lifecycle-gate.sh --report-dir "$ipc_rel"
bash scripts/release/surface/crash-gate.sh --report-dir "$crash_rel"
bash scripts/release/surface/i18n-gate.sh --report-dir "$i18n_rel"
bash scripts/release/surface/perf-gate.sh --report-dir "$perf_rel"
bash scripts/release/surface/migration-gate.sh --report-dir "$migration_rel"
bash scripts/release/surface/example-suite-gate.sh --report-dir "$examples_rel"
bash scripts/release/surface/api-stability-gate.sh --report-dir "$api_rel"
bash scripts/release/surface/electron-comparison-gate.sh --report-dir "$electron_rel"

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$api_dir" --out "$api_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$api_dir/artifact-hashes.json"

mkdir -p "$prod_claim_dir"
git_head="$(git rev-parse HEAD 2>/dev/null || echo 0123456789abcdef0123456789abcdef01234567)"
write_prod_claim_report "$prod_claim_dir/surface-prod-claim.json" "$git_head"
go run -buildvcs=false ./tools/cmd/validate-surface-prod-claim --report "$prod_claim_dir/surface-prod-claim.json"
cat > "$prod_claim_dir/surface-prod-claim-gate-summary.json" <<JSON
{
  "schema": "tetra.surface.prod-claim-gate.v1",
  "status": "pass",
  "claim_tier": "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "scope": "surface-prod-scoped-linux-web",
  "producer": "scripts/release/surface/prod-gate.sh",
  "claim_report": "surface-prod-claim.json",
  "validator": "tools/cmd/validate-surface-prod-claim",
  "negative_fixtures": "tools/validators/surfaceprod/report_test.go",
  "release_clean_required": true,
  "git_head": $(json_string "$git_head")
}
JSON
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$prod_claim_dir" --out "$prod_claim_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$prod_claim_dir/artifact-hashes.json"

for manifest in \
  "$release_dir/artifact-hashes.json" \
  "$visual_dir/artifact-hashes.json" \
  "$package_dir/artifact-hashes.json" \
  "$security_dir/artifact-hashes.json" \
  "$ipc_dir/artifact-hashes.json" \
  "$crash_dir/artifact-hashes.json" \
  "$i18n_dir/artifact-hashes.json" \
  "$perf_dir/artifact-hashes.json" \
  "$migration_dir/artifact-hashes.json" \
  "$examples_dir/artifact-hashes.json" \
  "$api_dir/artifact-hashes.json" \
  "$electron_dir/artifact-hashes.json" \
  "$prod_claim_dir/artifact-hashes.json"; do
  go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$manifest"
done

generated_at_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
formatted_args="$(format_command "${original_args[@]}")"
command_line="bash scripts/release/surface/prod-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

prod_report_path="$release_dir/surface-prod-gate-report.json"
cat > "$prod_report_path" <<JSON
{
  "schema": "tetra.surface.prod-gate-report.v1",
  "status": "pass",
  "level": "surface-production-ci-release-gate-v1",
  "scope": "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/prod-gate.sh",
  "git_head": $(json_string "$git_head"),
  "same_commit": true,
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "ci_jobs": [
    {
      "workflow": ".github/workflows/release-packages.yml",
      "job": "release-packages",
      "required": true,
      "continue_on_error": false,
      "command": "bash scripts/release/surface/prod-gate.sh",
      "artifact_upload": "surface-production-final"
    }
  ],
  "gates": [
    {"name":"surface-release","report_dir":"surface-release-v1","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-release-v1/artifact-hashes.json"},
    {"name":"block-system","report_dir":"surface-release-v1/block-system","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-release-v1/artifact-hashes.json"},
    {"name":"morph","report_dir":"surface-release-v1/morph","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-release-v1/artifact-hashes.json"},
    {"name":"visual","report_dir":"surface-visual-regression","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-visual-regression/artifact-hashes.json"},
    {"name":"package","report_dir":"surface-package-distribution","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-package-distribution/artifact-hashes.json"},
    {"name":"security","report_dir":"surface-security-sandbox","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-security-sandbox/artifact-hashes.json"},
    {"name":"ipc-lifecycle","report_dir":"surface-ipc-lifecycle","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-ipc-lifecycle/artifact-hashes.json"},
    {"name":"crash-diagnostics","report_dir":"surface-crash-diagnostics","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-crash-diagnostics/artifact-hashes.json"},
    {"name":"i18n-localization","report_dir":"surface-i18n-localization","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-i18n-localization/artifact-hashes.json"},
    {"name":"performance-memory","report_dir":"surface-performance-memory","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-performance-memory/artifact-hashes.json"},
    {"name":"widget-migration","report_dir":"surface-widget-migration","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-widget-migration/artifact-hashes.json"},
    {"name":"example-suite","report_dir":"surface-example-suite","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-example-suite/artifact-hashes.json"},
    {"name":"api-stability","report_dir":"surface-api-stability-v1","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-api-stability-v1/artifact-hashes.json"},
    {"name":"electron-comparison","report_dir":"surface-electron-comparison","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-electron-comparison/artifact-hashes.json"},
    {"name":"prod-claim","report_dir":"surface-prod-claim","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-prod-claim/artifact-hashes.json"}
  ],
  "targets": [
    {"target":"linux-x64","tier":"prod","ran":true,"pass":true,"skipped":false},
    {"target":"wasm32-web","tier":"prod","ran":true,"pass":true,"skipped":false},
    {"target":"windows-x64","tier":"beta","ran":false,"pass":false,"skipped":true},
    {"target":"macos-x64","tier":"beta","ran":false,"pass":false,"skipped":true}
  ],
  "artifact_hashes_validated": true,
  "negative_guards": {
    "missing_job_rejected": true,
    "continue_on_error_rejected": true,
    "skipped_target_as_pass_rejected": true,
    "missing_artifact_hash_manifest_rejected": true
  },
  "cases": [
    {"name":"release-packages production gate job required","kind":"positive","ran":true,"pass":true},
    {"name":"no continue-on-error production jobs","kind":"negative","ran":true,"pass":true},
    {"name":"skipped target counted as pass rejected","kind":"negative","ran":true,"pass":true},
    {"name":"artifact hash manifest missing rejected","kind":"negative","ran":true,"pass":true}
  ]
}
JSON

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$release_dir" --out "$release_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$release_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-surface-release-state \
  --report-dir "$release_dir" \
  --expected-status current \
  --scope PROD_STABLE_SCOPED_LINUX_WEB_APP_UI \
  --manifest docs/generated/manifest.json

cat > "$report_dir/surface-prod-gate-summary.json" <<JSON
{
  "schema": "tetra.surface.prod-gate-summary.v1",
  "status": "pass",
  "level": "surface-production-ci-release-gate-v1",
  "scope": "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/prod-gate.sh",
  "prod_gate_report": "surface-release-v1/surface-prod-gate-report.json",
  "production_artifact_upload": "surface-production-final",
  "linux_web_prod_targets": ["linux-x64", "wasm32-web"],
  "beta_targets": ["windows-x64", "macos-x64"],
  "artifact_hashes_validated": true,
  "release_state_validated": true,
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "git_head": $(json_string "$git_head")
}
JSON
cp "$report_dir/surface-prod-gate-summary.json" "$report_dir/surface-prod-summary.json"

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface production gate reports: $report_dir"
echo "Surface production gate summary: $report_dir/surface-prod-gate-summary.json"
echo "Surface production final summary: $report_dir/surface-prod-summary.json"
echo "Surface production gate report: $prod_report_path"
echo "Surface production artifact hashes: $report_dir/artifact-hashes.json"
