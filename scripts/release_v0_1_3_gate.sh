#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

report_dir=""
release_version="${TETRA_RELEASE_GATE_VERSION:-v0.1.3}"
release_slug="${release_version//./_}"
release_artifact="${TETRA_RELEASE_GATE_ARTIFACT:-tetra.release.v0_1_3.gate-report.v1}"
release_gate_command="${TETRA_RELEASE_GATE_COMMAND:-bash scripts/release_v0_1_3_gate.sh}"
actor_diagnostic_contains="${TETRA_RELEASE_GATE_ACTOR_DIAGNOSTIC_CONTAINS:-planned feature 'actor'}"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release_v0_1_3_gate.sh [--report-dir DIR]
Artifact mapping: tetra.release.v0_1_3.gate-report.v1
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 || -z "${2:-}" ]]; then
        echo "release_v0_1_3_gate: --report-dir requires a directory" >&2
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
      echo "release_v0_1_3_gate: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_dir" ]]; then
  report_dir="/tmp/release-${release_slug}-gate-$(date -u +%Y%m%d-%H%M%S)"
fi

logs_dir="$report_dir/logs"
summary_md="$report_dir/summary.md"
summary_json="$report_dir/summary.json"
artifacts_dir="$report_dir/artifacts"

mkdir -p "$logs_dir" "$artifacts_dir"

tmp_dir="$(mktemp -d)"
generated_state_before="$tmp_dir/generated-artifacts.before"
trap 'rm -rf "$tmp_dir"' EXIT

step_count=0
failed_count=0
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
: >"$tmp_dir/steps.md"
: >"$tmp_dir/steps.jsonl"

json_escape() {
  local s="${1-}"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  printf '%s' "$s"
}

slugify() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//'
}

record_step() {
  local name="$1"
  local status="$2"
  local seconds="$3"
  local exit_code="$4"
  local log_rel="$5"
  local command="$6"

  printf -- '- `%s`: `%s` in %ss, exit `%s`, command `%s` ([%s](%s))\n' "$name" "$status" "$seconds" "$exit_code" "$command" "$log_rel" "$log_rel" >>"$tmp_dir/steps.md"
  printf '{"name":"%s","status":"%s","duration_seconds":%s,"exit_code":%s,"command":"%s","log":"%s"}\n' \
    "$(json_escape "$name")" \
    "$(json_escape "$status")" \
    "$seconds" \
    "$exit_code" \
    "$(json_escape "$command")" \
    "$(json_escape "$log_rel")" >>"$tmp_dir/steps.jsonl"
}

run_step() {
  local name="$1"
  shift
  step_count=$((step_count + 1))

  local id
  local slug
  local log_rel
  local log_path
  local command
  local start_s
  local end_s

  id="$(printf '%02d' "$step_count")"
  slug="$(slugify "$name")"
  log_rel="logs/${id}-${slug}.log"
  log_path="$report_dir/$log_rel"
  command="$*"

  printf '== [%s] %s ==\n' "$id" "$name"
  start_s="$(date +%s)"

  if "$@" >"$log_path" 2>&1; then
    end_s="$(date +%s)"
    record_step "$name" "pass" "$((end_s - start_s))" 0 "$log_rel" "$command"
    printf '   pass (%ss)\n' "$((end_s - start_s))"
  else
    local rc="$?"
    end_s="$(date +%s)"
    record_step "$name" "fail" "$((end_s - start_s))" "$rc" "$log_rel" "$command"
    failed_count=$((failed_count + 1))
    printf '   fail (%ss), exit %s\n' "$((end_s - start_s))" "$rc"
    tail -n 60 "$log_path" >&2 || true
  fi
}

write_summary() {
  local status="$1"
  local ended_at
  ended_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  {
    echo "# Tetra $release_version Release Gate Report"
    echo
    echo "- status: \`$status\`"
    echo "- release_version: \`$release_version\`"
    echo "- release_artifact: \`$release_artifact\`"
    echo "- release_gate_command: \`$release_gate_command\`"
    echo "- started_at: \`$started_at\`"
    echo "- ended_at: \`$ended_at\`"
    echo "- step_count: \`$step_count\`"
    echo "- failed_count: \`$failed_count\`"
    echo "- report_dir: \`$report_dir\`"
    echo
    echo "## Steps"
    echo
    cat "$tmp_dir/steps.md"
  } >"$summary_md"

  {
    echo "{"
    printf '  "status": "%s",\n' "$(json_escape "$status")"
    printf '  "release_version": "%s",\n' "$(json_escape "$release_version")"
    printf '  "release_artifact": "%s",\n' "$(json_escape "$release_artifact")"
    printf '  "release_gate_command": "%s",\n' "$(json_escape "$release_gate_command")"
    printf '  "started_at": "%s",\n' "$(json_escape "$started_at")"
    printf '  "ended_at": "%s",\n' "$(json_escape "$ended_at")"
    printf '  "step_count": %s,\n' "$step_count"
    printf '  "failed_count": %s,\n' "$failed_count"
    printf '  "report_dir": "%s",\n' "$(json_escape "$report_dir")"
    echo '  "steps": ['
    awk 'NR > 1 { printf ",\n" } { printf "    %s", $0 } END { if (NR > 0) printf "\n" }' "$tmp_dir/steps.jsonl"
    echo '  ]'
    echo "}"
  } >"$summary_json"
}

check_release_version() {
  local version
  version="$(./tetra version 2>/dev/null || true)"
  if [[ "$version" != "$release_version" ]]; then
    echo "expected $release_version version, got '${version:-<missing>}'" >&2
    return 1
  fi
}

check_short_alias_version() {
  local version
  local short_version
  version="$(./tetra version 2>/dev/null || true)"
  short_version="$(./t version 2>/dev/null || true)"
  if [[ "$short_version" != "$version" ]]; then
    echo "expected ./t version to match ./tetra version ($version), got '${short_version:-<missing>}'" >&2
    return 1
  fi
}

check_docs_manifest() {
  go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
  go run ./tools/cmd/gen-manifest -o "$artifacts_dir/manifest.json"
  go run ./tools/cmd/validate-manifest --manifest "$artifacts_dir/manifest.json"
  diff -u docs/generated/manifest.json "$artifacts_dir/manifest.json"
}

check_api_diff() {
  bash scripts/release_v1_0_api_diff.sh --report-dir "$artifacts_dir/api-diff" --baseline docs/baselines/api-diff-baseline.v1alpha1.json --enforce no-change
}

check_performance_regression_artifact() {
  local src="docs/generated/v1_0/performance-regression.json"
  local dst="$artifacts_dir/performance-regression.json"
  if [[ ! -f "$src" ]]; then
    echo "missing performance artifact $src" >&2
    return 1
  fi
  cp "$src" "$dst"
  go run ./tools/cmd/validate-performance-report --report "$dst"
}

check_repro_build() {
  bash scripts/release_v1_0_repro.sh --report "$artifacts_dir/reproducible-build.json"
}

check_binary_size_thresholds() {
  bash scripts/release_v1_0_binary_size.sh --report "$artifacts_dir/binary-size-thresholds.json"
}

check_security_review_signoff() {
  local signoff_path="${TETRA_SECURITY_REVIEW_SIGNOFF:-$artifacts_dir/security-review.md}"
  bash scripts/release_v1_0_security_review.sh --signoff "$signoff_path"
  cp "$signoff_path" "$artifacts_dir/security-review.md"
}

capture_generated_artifact_state() {
  local out="$1"
  {
    echo "## git status"
    git status --porcelain --untracked-files=no -- docs/generated docs/baselines
    echo "## git diff"
    git diff --binary -- docs/generated docs/baselines
  } >"$out"
}

check_generated_artifact_churn() {
  local after="$tmp_dir/generated-artifacts.after"
  capture_generated_artifact_state "$after"
  diff -u "$generated_state_before" "$after"
}

check_release_state() {
  go run ./tools/cmd/validate-release-state --expected-version "$release_version" --format=json --report-dir "$report_dir" >"$artifacts_dir/release-state.json"
  go run ./tools/cmd/validate-release-state --expected-version "$release_version" --format=text --report-dir "$report_dir" >"$artifacts_dir/release-state.txt"
}

write_known_issues_artifact() {
  local gate_result="pass"
  local version
  local branch
  if [[ "$failed_count" -gt 0 ]]; then
    gate_result="blocked"
  fi
  version="$(./tetra version 2>/dev/null || echo '<unknown>')"
  branch="$(git branch --show-current 2>/dev/null || echo '<unknown>')"
  cat >"$artifacts_dir/known_issues.md" <<KNOWN_ISSUES
# Tetra $release_version Known Issues

Generated by \`$release_gate_command\`.

## Release

- Version: \`$version\`
- Candidate or patch branch: \`$branch\`
- Artifact archive: \`$report_dir\`
- Last release gate command: \`$release_gate_command --report-dir $report_dir\`
- Last release gate result: \`$gate_result\`

## Issues

| ID | Title | Component | User impact | Workaround | Release blocker? | Owner | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |

No known issues were emitted automatically by the release gate. Add reviewed rows
before release if blockers or accepted non-blockers are discovered.
KNOWN_ISSUES
}

check_artifact_hash_manifest() {
  go run ./tools/cmd/validate-artifact-hashes --write --root "$artifacts_dir" --out "$artifacts_dir/artifact-hashes.json"
  go run ./tools/cmd/validate-artifact-hashes --manifest "$artifacts_dir/artifact-hashes.json"
}

check_tetra_doc_output() {
  # Keep the canonical command text for script parity tests: ./tetra doc examples
  mapfile -t tracked_examples < <(git ls-files 'examples/*.tetra')
  if [[ "${#tracked_examples[@]}" -eq 0 ]]; then
    echo "no tracked examples found under examples/" >&2
    return 1
  fi
  ./tetra doc "${tracked_examples[@]}" >"$artifacts_dir/tetra-docs.md"
  go run ./tools/cmd/validate-api-docs --docs "$artifacts_dir/tetra-docs.md"
}

check_json_diagnostic_case() {
  local name="$1"
  local contains="$2"
  local source="$tmp_dir/$name.tetra"
  local stdout="$tmp_dir/$name.out"
  local diagnostic="$artifacts_dir/$name.json"
  shift 2
  cat >"$source"
  if ./tetra check --diagnostics=json "$source" >"$stdout" 2>"$diagnostic"; then
    echo "expected tetra check --diagnostics=json to fail for $name" >&2
    return 1
  fi
  test ! -s "$stdout"
  go run ./tools/cmd/validate-diagnostic --diagnostic "$diagnostic" --severity error --contains "$contains" --require-position
}

check_json_diagnostic() {
  check_json_diagnostic_case "invalid-diagnostic" "unknown function" <<'TETRA'
func main() -> Int:
    return missing_call()
TETRA
  check_json_diagnostic_case "missing-effect-diagnostic" "uses effect 'io'" <<'TETRA'
func main() -> Int:
    print("missing uses\n")
    return 0
TETRA
  check_json_diagnostic_case "tabs-diagnostic" "tabs are not supported" <<'TETRA'
func main() -> Int:
	return 0
TETRA
  check_json_diagnostic_case "planned-actor-diagnostic" "$actor_diagnostic_contains" <<'TETRA'
actor Worker:
    return 0
TETRA
}

check_wasi_runner_smoke() {
  bash scripts/release_v1_0_wasi_smoke.sh --report "$artifacts_dir/wasi-smoke.json"
}

check_web_ui_smoke() {
  bash scripts/release_v1_0_web_smoke.sh --report "$artifacts_dir/web-ui-smoke.json"
  go run ./tools/cmd/validate-web-ui-smoke --report "$artifacts_dir/web-ui-smoke.json"
}

echo "== $release_version preflight =="
echo "report_dir: $report_dir"

run_step "bootstrap tetra binaries" bash scripts/bootstrap.sh
run_step "version preflight ($release_version required)" check_release_version
if [[ "$failed_count" -gt 0 ]]; then
  write_summary "blocked"
  echo
  echo "$release_version release gate blocked at version preflight: expected $release_version before mandatory release checks" >&2
  echo "summary: $summary_md" >&2
  echo "json: $summary_json" >&2
  exit 1
fi
run_step "short alias version parity" check_short_alias_version
capture_generated_artifact_state "$generated_state_before"
run_step "go test packages" go test ./compiler/... ./cli/... ./tools/... -count=1
run_step "full stabilization wrapper" bash scripts/test_all.sh --full --keep-going --report-dir "$artifacts_dir/test-all"
run_step "flow-only source scan" go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
run_step "targets report validation" sh -c './tetra targets --format=json >"$1" && go run ./tools/cmd/validate-targets --report "$1"' sh "$artifacts_dir/targets.json"
run_step "doctor report validation" sh -c './tetra doctor --format=json >"$1" && go run ./tools/cmd/validate-doctor --report "$1"' sh "$artifacts_dir/doctor.json"
run_step "tetra check flow hello" ./tetra check examples/flow_hello.tetra
run_step "formatter check" ./tetra fmt --check examples lib __rt compiler/selfhostrt
run_step "tetra test examples json" sh -c './tetra test --report=json examples >"$1" && go run ./tools/cmd/validate-test-report --report "$1"' sh "$artifacts_dir/tetra-test-report.json"
run_step "docs manifest regenerate+validate" check_docs_manifest
run_step "docs verification and doctests" go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
run_step "tetra doc output validation" check_tetra_doc_output
run_step "json diagnostic shape" check_json_diagnostic
run_step "smoke list validation" sh -c './tetra smoke --list --format=json >"$1" && go run ./tools/cmd/validate-smoke-list --report "$1" --examples-root examples' sh "$artifacts_dir/smoke-list.json"
run_step "native host smoke linux-x64" sh -c './tetra smoke --target linux-x64 --run=true --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1"' sh "$artifacts_dir/host-smoke.json"

run_step "build-only smoke linux-x64" sh -c './tetra smoke --target linux-x64 --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1"' sh "$artifacts_dir/linux-smoke.json"
run_step "build-only smoke macos-x64" sh -c './tetra smoke --target macos-x64 --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1"' sh "$artifacts_dir/macos-smoke.json"
run_step "build-only smoke windows-x64" sh -c './tetra smoke --target windows-x64 --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1"' sh "$artifacts_dir/windows-smoke.json"
run_step "build-only smoke wasm32-wasi" sh -c './tetra smoke --target wasm32-wasi --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1"' sh "$artifacts_dir/wasm32-wasi-smoke.json"
run_step "build-only smoke wasm32-web" sh -c './tetra smoke --target wasm32-web --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1"' sh "$artifacts_dir/wasm32-web-smoke.json"

run_step "WASI runner smoke" check_wasi_runner_smoke
run_step "Web UI browser smoke" check_web_ui_smoke
run_step "security review signoff" check_security_review_signoff
run_step "API diff gate" check_api_diff
run_step "performance regression evidence" check_performance_regression_artifact
run_step "binary size thresholds" check_binary_size_thresholds
run_step "reproducible build proof" check_repro_build
run_step "eco verify command surface" sh -c 'test -x ./tetra && ./tetra eco verify --help >/dev/null'
run_step "known issues artifact" write_known_issues_artifact
run_step "artifact hash manifest" check_artifact_hash_manifest
run_step "generated artifact churn check" check_generated_artifact_churn

if [[ "$failed_count" -eq 0 ]]; then
  write_summary "pass"
else
  write_summary "blocked"
fi
run_step "release state audit" check_release_state
run_step "artifact hash manifest refresh" check_artifact_hash_manifest

if [[ "$failed_count" -gt 0 ]]; then
  write_summary "blocked"
  echo
  echo "$release_version release gate blocked: $failed_count step(s) failed" >&2
  echo "summary: $summary_md" >&2
  echo "json: $summary_json" >&2
  exit 1
fi

write_summary "pass"
echo
echo "$release_version release gate passed"
echo "summary: $summary_md"
echo "json: $summary_json"
