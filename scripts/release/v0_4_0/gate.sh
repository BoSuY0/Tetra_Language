#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
cd "$repo_root"

report_dir=""
require_clean=0
release_version="v0.4.0"
release_artifact="tetra.release.v0_4_0.gate-report.v1"
release_gate_command="bash scripts/release/v0_4_0/gate.sh"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/v0_4_0/gate.sh [--report-dir DIR] [--require-clean]

Notes:
- This gate validates the scoped Linux-x64 v0.4.0 production objective.
- EcoNet, non-Linux target runtimes, WASM target runtimes, and full v1.0
  language guarantees are outside this production claim.
- It requires ./tetra version == v0.4.0 and ./t version parity before expensive
  release evidence can run.
- Artifact mapping: tetra.release.v0_4_0.gate-report.v1
- The first implemented step is a readiness preflight over manifest, feature
  registry, target runtime status, and v0.4.0 scope decisions.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 || -z "${2:-}" ]]; then
        echo "release_v0_4_0_gate: --report-dir requires a directory" >&2
        usage >&2
        exit 2
      fi
      report_dir="$2"
      shift 2
      ;;
    --require-clean)
      require_clean=1
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "release_v0_4_0_gate: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_dir" ]]; then
  report_dir="reports/release-v0.4.0-gate-$(date -u +%Y%m%d-%H%M%S)"
fi

check_report_dir_fresh() {
  if [[ (-e "$report_dir" || -L "$report_dir") && ! -d "$report_dir" ]]; then
    echo "release_v0_4_0_gate: refusing to use non-directory report path: $report_dir" >&2
    echo "release_v0_4_0_gate: choose a fresh --report-dir directory" >&2
    exit 2
  fi
  if [[ ! -d "$report_dir" ]]; then
    return 0
  fi
  local first_entry
  local find_report_dir="$report_dir"
  if [[ "$find_report_dir" == -* ]]; then
    find_report_dir="./$find_report_dir"
  fi
  first_entry="$(find -H "$find_report_dir" -mindepth 1 -print -quit)"
  if [[ -n "$first_entry" ]]; then
    echo "release_v0_4_0_gate: refusing to reuse non-empty report directory: $report_dir" >&2
    echo "release_v0_4_0_gate: choose a fresh --report-dir so stale reports cannot be reused" >&2
    exit 2
  fi
}

check_report_dir_fresh

json_escape() {
  local s="${1-}"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  printf '%s' "$s"
}

write_blocked_summary() {
  local step_name="$1"
  local command="$2"
  local reason="$3"
  local duration_seconds="$4"
  local exit_code="$5"
  local log_rel="$6"
  local ended_at
  local step_md
  ended_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf -v step_md -- \
    '- `%s`: `fail` in %ss, exit `%s`, command `%s` ([%s](%s))' \
    "$step_name" \
    "$duration_seconds" \
    "$exit_code" \
    "$command" \
    "$log_rel" \
    "$log_rel"

  cat > "$report_dir/summary.md" << SUMMARY
# Tetra $release_version Release Gate Report

- status: \`blocked\`
- release_version: \`$release_version\`
- release_artifact: \`$release_artifact\`
- release_gate_command: \`$release_gate_command\`
- started_at: \`$started_at\`
- ended_at: \`$ended_at\`
- step_count: \`1\`
- failed_count: \`1\`
- report_dir: \`$report_dir\`

## Steps

$step_md

## Blocker

$reason
SUMMARY

  cat > "$report_dir/summary.json" << JSON
{
  "status": "blocked",
  "release_version": "$(json_escape "$release_version")",
  "release_artifact": "$(json_escape "$release_artifact")",
  "release_gate_command": "$(json_escape "$release_gate_command")",
  "started_at": "$(json_escape "$started_at")",
  "ended_at": "$(json_escape "$ended_at")",
  "step_count": 1,
  "failed_count": 1,
  "report_dir": "$(json_escape "$report_dir")",
  "steps": [
    {
      "name": "$(json_escape "$step_name")",
      "status": "fail",
      "duration_seconds": $duration_seconds,
      "exit_code": $exit_code,
      "command": "$(json_escape "$command")",
      "log": "$(json_escape "$log_rel")"
    }
  ]
}
JSON
}

validate_blocked_summary() {
  go run ./tools/cmd/validate-release-gate-summary \
    --summary "$report_dir/summary.json" \
    --report-dir "$report_dir" \
    --expected-version "$release_version" \
    --expected-artifact "$release_artifact" \
    --expected-command "$release_gate_command"
}

write_blocked_release_state() {
  go run ./tools/cmd/validate-release-state \
    --expected-version "$release_version" \
    --format=json \
    --report-dir "$report_dir" > "$report_dir/artifacts/release-state.json" || true
  go run ./tools/cmd/validate-release-state \
    --expected-version "$release_version" \
    --format=text \
    --report-dir "$report_dir" > "$report_dir/artifacts/release-state.txt" || true
}

write_blocked_readiness_blockers() {
  local log_rel="$1"
  local detail
  detail="$(cat "$report_dir/$log_rel" 2> /dev/null || true)"
  cat > "$report_dir/artifacts/readiness-blockers.json" << JSON
{
  "schema": "tetra.release.v0_4_0.readiness-blockers.v1",
  "release_version": "$(json_escape "$release_version")",
  "artifact": "readiness-blockers.json",
  "source_log": "$(json_escape "$log_rel")",
  "blockers": [
    {
      "id": "readiness-preflight",
      "status": "blocked",
      "summary": "v0.4.0 readiness preflight failed",
      "detail": "$(json_escape "$detail")"
    }
  ]
}
JSON
  go run ./tools/cmd/validate-v0-4-readiness-blockers \
    --artifact "$report_dir/artifacts/readiness-blockers.json" \
    --report-dir "$report_dir" \
    --expected-version "$release_version"
}

write_blocked_residual_risks() {
  local log_rel="$1"
  local readiness_summary
  readiness_summary="v0.4.0 readiness preflight failed; production release "
  readiness_summary+="evidence cannot be collected until manifest, feature, "
  readiness_summary+="target, and scope-decision blockers are resolved."
  cat > "$report_dir/artifacts/residual-risks.json" << JSON
{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "$(json_escape "$release_version")",
  "artifact": "residual-risks.json",
  "risks": [
    {
      "id": "v0.4.0-readiness-preflight",
      "severity": "critical",
      "owner": "release-owner",
      "status": "blocked",
      "summary": "$(json_escape "$readiness_summary")",
      "evidence": "$(json_escape "$log_rel")"
    }
  ]
}
JSON
  go run ./tools/cmd/validate-residual-risks \
    --artifact "$report_dir/artifacts/residual-risks.json" \
    --expected-version "$release_version"
}

write_blocked_artifact_hashes() {
  go run ./tools/cmd/validate-artifact-hashes \
    --write \
    --root "$report_dir" \
    --out "$report_dir/artifact-hashes.json"
  go run ./tools/cmd/validate-artifact-hashes \
    --manifest "$report_dir/artifact-hashes.json"
}

check_tag_ready_clean_worktree() {
  local status
  if ! status="$(git status --porcelain --untracked-files=all 2>&1)"; then
    echo "release_v0_4_0_gate: tag-ready clean worktree check failed" >&2
    printf '%s\n' "$status" >&2
    return 1
  fi
  if [[ -n "$status" ]]; then
    echo "release_v0_4_0_gate: blocked: tag-ready clean worktree required (--require-clean)" >&2
    echo "release_v0_4_0_gate: git status --porcelain --untracked-files=all" \
      "reported dirty state:" >&2
    printf '%s\n' "$status" >&2
    return 1
  fi
}

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
steps_md="$tmp_dir/steps.md"
steps_jsonl="$tmp_dir/steps.jsonl"
: > "$steps_md"
: > "$steps_jsonl"
step_count=0
failed_count=0

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
  local step_format
  local step_json_format

  step_format='- `%s`: `%s` in %ss, exit `%s`, command `%s` ([%s](%s))\n'
  step_json_format='{"name":"%s","status":"%s","duration_seconds":%s,'
  step_json_format+='"exit_code":%s,"command":"%s","log":"%s"}\n'

  printf -- "$step_format" \
    "$name" \
    "$status" \
    "$seconds" \
    "$exit_code" \
    "$command" \
    "$log_rel" \
    "$log_rel" >> "$steps_md"
  printf "$step_json_format" \
    "$(json_escape "$name")" \
    "$(json_escape "$status")" \
    "$seconds" \
    "$exit_code" \
    "$(json_escape "$command")" \
    "$(json_escape "$log_rel")" >> "$steps_jsonl"
}

record_known_step() {
  local name="$1"
  local status="$2"
  local seconds="$3"
  local exit_code="$4"
  local log_rel="$5"
  local command="$6"
  step_count=$((step_count + 1))
  if [[ "$status" == "fail" ]]; then
    failed_count=$((failed_count + 1))
  fi
  record_step "$name" "$status" "$seconds" "$exit_code" "$log_rel" "$command"
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

  start_s="$(date +%s)"
  if "$@" > "$log_path" 2>&1; then
    end_s="$(date +%s)"
    record_step "$name" "pass" "$((end_s - start_s))" 0 "$log_rel" "$command"
    printf 'release_v0_4_0_gate: pass: %s\n' "$name"
  else
    local rc="$?"
    end_s="$(date +%s)"
    record_step "$name" "fail" "$((end_s - start_s))" "$rc" "$log_rel" "$command"
    failed_count=$((failed_count + 1))
    printf 'release_v0_4_0_gate: fail: %s (exit %s)\n' "$name" "$rc" >&2
    tail -n 60 "$log_path" >&2 || true
  fi
}

write_final_summary() {
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
    cat "$steps_md"
  } > "$report_dir/summary.md"

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
    awk '
      NR > 1 { printf ",\n" }
      { printf "    %s", $0 }
      END { if (NR > 0) printf "\n" }
    ' "$steps_jsonl"
    echo '  ]'
    echo "}"
  } > "$report_dir/summary.json"
}

validate_final_summary() {
  go run ./tools/cmd/validate-release-gate-summary \
    --summary "$report_dir/summary.json" \
    --report-dir "$report_dir" \
    --expected-version "$release_version" \
    --expected-artifact "$release_artifact" \
    --expected-command "$release_gate_command"
}

check_versions() {
  local tetra_version
  local short_version
  tetra_version="$(./tetra version 2> /dev/null || true)"
  short_version="$(./t version 2> /dev/null || true)"
  if [[ "$tetra_version" != "$release_version" ]]; then
    echo "expected ./tetra version $release_version, got ${tetra_version:-<missing>}" >&2
    return 1
  fi
  if [[ "$short_version" != "$release_version" ]]; then
    echo "expected ./t version $release_version, got ${short_version:-<missing>}" >&2
    return 1
  fi
}

check_go_test_packages() {
  env \
    -u TETRA_SECURITY_REVIEW_SIGNOFF \
    -u TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF \
    -u TETRA_MACOS_RUNTIME_SMOKE_REPORT \
    -u TETRA_WINDOWS_RUNTIME_SMOKE_REPORT \
    go test ./compiler/... ./cli/... ./tools/... -count=1
}

check_techempower_reports() {
  local report
  local reports=(
    "docs/benchmarks/techempower_local_smoke_skip_db_report.json"
    "docs/benchmarks/techempower_scram_single_query_local_report.json"
    "docs/benchmarks/techempower_scram_single_query_matrix_local_report.json"
    "docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json"
  )
  for report in "${reports[@]}"; do
    if [[ "$report" == "docs/benchmarks/techempower_local_smoke_skip_db_report.json" ]]; then
      go run ./tools/cmd/validate-techempower-report --report "$report" --allow-skip-db
    else
      go run ./tools/cmd/validate-techempower-report --report "$report"
    fi
  done
}

run_linux_host_smoke() {
  go run ./cli/cmd/tetra smoke \
    --target linux-x64 \
    --run=true \
    --report "$report_dir/artifacts/linux-host-smoke.json"
  cp -- "$report_dir/artifacts/linux-host-smoke.json" "$canonical_v04_report_dir/linux-host-smoke.json"
}

run_distributed_actor_smoke() {
  bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh \
    --report-dir "$report_dir/artifacts"
  cp -- "$report_dir/artifacts/distributed-actors-linux-x64.json" "$canonical_v04_report_dir/distributed-actors-linux-x64.json"
}

run_native_ui_smoke() {
  bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir "$report_dir/artifacts"
  cp -- "$report_dir/artifacts/native-ui-linux-x64.json" "$canonical_v04_report_dir/native-ui-linux-x64.json"
}

run_memory_production_smoke() {
  local memory_report_dir="$tmp_dir/memory-production"
  bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh \
    --report-dir "$memory_report_dir"
  cp -- \
    "$memory_report_dir/memory-production-linux-x64.json" \
    "$report_dir/artifacts/memory-production-linux-x64.json"
}

run_memory_core_v2_gate() {
  local memory_core_v2_report_dir="$tmp_dir/memory-core-v2"
  bash scripts/release/memory/memory-core-v2-gate.sh \
    --report-dir "$memory_core_v2_report_dir"
  cp -- \
    "$memory_core_v2_report_dir/memory-core-v2-evidence.json" \
    "$report_dir/artifacts/memory-core-v2-evidence.json"
}

run_parallel_production_smoke() {
  local parallel_report_dir="$tmp_dir/parallel-production"
  bash scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh \
    --report-dir "$parallel_report_dir"
  cp -- \
    "$parallel_report_dir/parallel-production-linux-x64.json" \
    "$report_dir/artifacts/parallel-production-linux-x64.json"
}

run_compiler_production_smoke() {
  local compiler_report_dir="$tmp_dir/compiler-production"
  bash scripts/release/post_v0_4/compiler-production-linux-x64-smoke.sh \
    --report-dir "$compiler_report_dir"
  cp -- \
    "$compiler_report_dir/compiler-production-linux-x64.json" \
    "$report_dir/artifacts/compiler-production-linux-x64.json"
}

check_readiness_final() {
  go run ./tools/cmd/validate-v0-4-readiness \
    --expected-version "$release_version" \
    --features "$report_dir/artifacts/features.json" \
    --targets "$report_dir/artifacts/targets.json" \
    --manifest docs/generated/manifest.json \
    --scope-decisions docs/release/v0_4/data/v0_4_0_scope_decisions.json
}

check_release_state() {
  go run ./tools/cmd/validate-v0-4-release-state \
    --expected-version "$release_version" \
    --format=json \
    --report-dir "$report_dir" > "$report_dir/artifacts/release-state.json"
  go run ./tools/cmd/validate-v0-4-release-state \
    --expected-version "$release_version" \
    --format=text \
    --report-dir "$report_dir" > "$report_dir/artifacts/release-state.txt"
}

check_security_review_signoff() {
  local signoff_path="${TETRA_SECURITY_REVIEW_SIGNOFF:-}"
  if [[ -z "$signoff_path" ]]; then
    cat > "$report_dir/artifacts/security-review.md" << EOF
# v0.4.0 Security Review Signoff

Decision: blocked
Reason: missing TETRA_SECURITY_REVIEW_SIGNOFF for the exact release candidate.

Create a signoff with:

\`\`\`sh
bash scripts/release/v0_4_0/security-review.sh --write-template <security-review.md>
\`\`\`
EOF
    echo "release_v0_4_0_gate: missing TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>" >&2
    return 1
  fi
  if [[ ! -f "$signoff_path" ]]; then
    echo "release_v0_4_0_gate: missing security review signoff artifact: $signoff_path" >&2
    return 1
  fi
  cp -- "$signoff_path" "$report_dir/artifacts/security-review.md"
  bash scripts/release/v0_4_0/security-review.sh \
    --signoff "$report_dir/artifacts/security-review.md"
}

write_security_review_hash() {
  local review_path="$report_dir/artifacts/security-review.md"
  local detached_hash_path="$report_dir/artifacts/security-review.md.sha256"
  if [[ ! -f "$review_path" ]]; then
    echo "release_v0_4_0_gate: cannot hash missing security review: $review_path" >&2
    return 1
  fi
  if command -v sha256sum > /dev/null 2>&1; then
    sha256sum "$review_path" |
      awk '{print "sha256:" $1 "  artifacts/security-review.md"}' \
        > "$detached_hash_path"
  else
    shasum -a 256 "$review_path" |
      awk '{print "sha256:" $1 "  artifacts/security-review.md"}' \
        > "$detached_hash_path"
  fi
}

write_artifact_hashes() {
  go run ./tools/cmd/validate-artifact-hashes \
    --write \
    --root "$report_dir" \
    --out "$report_dir/artifact-hashes.json"
  go run ./tools/cmd/validate-artifact-hashes \
    --manifest "$report_dir/artifact-hashes.json"
}

if [[ "$require_clean" -eq 1 ]]; then
  check_tag_ready_clean_worktree
fi

canonical_v04_report_dir="reports/v0.4.0"
mkdir -p "$report_dir/artifacts" "$report_dir/logs" "$canonical_v04_report_dir"
features_json="$report_dir/artifacts/features.json"
targets_json="$report_dir/artifacts/targets.json"
readiness_runtime_report_args=()

go run ./cli/cmd/tetra features --format=json > "$features_json"
go run ./cli/cmd/tetra targets --format=json > "$targets_json"
cp -- "$features_json" "$canonical_v04_report_dir/features.json"
cp -- "$targets_json" "$canonical_v04_report_dir/targets.json"

if [[ -n "${TETRA_MACOS_RUNTIME_SMOKE_REPORT:-}" ]]; then
  readiness_runtime_report_args+=(
    --runtime-report
    "macos-x64=$TETRA_MACOS_RUNTIME_SMOKE_REPORT"
  )
fi
if [[ -n "${TETRA_WINDOWS_RUNTIME_SMOKE_REPORT:-}" ]]; then
  readiness_runtime_report_args+=(
    --runtime-report
    "windows-x64=$TETRA_WINDOWS_RUNTIME_SMOKE_REPORT"
  )
fi

readiness_command="go run ./tools/cmd/validate-v0-4-readiness"
readiness_command+=" --expected-version $release_version"
readiness_command+=" --features $features_json"
readiness_command+=" --targets $targets_json"
readiness_command+=" --manifest docs/generated/manifest.json"
readiness_command+=" --scope-decisions docs/release/v0_4/data/v0_4_0_scope_decisions.json"
readiness_command+=" --allow-pending-release-artifacts"
if [[ -n "${TETRA_MACOS_RUNTIME_SMOKE_REPORT:-}" ]]; then
  readiness_command+=" --runtime-report macos-x64=$TETRA_MACOS_RUNTIME_SMOKE_REPORT"
fi
if [[ -n "${TETRA_WINDOWS_RUNTIME_SMOKE_REPORT:-}" ]]; then
  readiness_command+=" --runtime-report windows-x64=$TETRA_WINDOWS_RUNTIME_SMOKE_REPORT"
fi
readiness_log_rel="logs/01-readiness-preflight.log"
readiness_log_path="$report_dir/$readiness_log_rel"
readiness_start_s="$(date +%s)"

if go run ./tools/cmd/validate-v0-4-readiness \
  --expected-version "$release_version" \
  --features "$features_json" \
  --targets "$targets_json" \
  --manifest docs/generated/manifest.json \
  --scope-decisions docs/release/v0_4/data/v0_4_0_scope_decisions.json \
  --allow-pending-release-artifacts \
  "${readiness_runtime_report_args[@]}" > "$readiness_log_path" 2>&1; then
  readiness_rc=0
else
  readiness_rc=$?
fi
readiness_end_s="$(date +%s)"
readiness_duration="$((readiness_end_s - readiness_start_s))"

if [[ "$readiness_rc" -ne 0 ]]; then
  write_blocked_summary \
    "readiness preflight" \
    "$readiness_command" \
    "v0.4.0 readiness preflight failed" \
    "$readiness_duration" \
    "$readiness_rc" \
    "$readiness_log_rel"
  validate_blocked_summary
  write_blocked_release_state
  write_blocked_readiness_blockers "$readiness_log_rel"
  write_blocked_residual_risks "$readiness_log_rel"
  write_blocked_artifact_hashes
  cat "$readiness_log_path" >&2 || true
  echo "release_v0_4_0_gate: v0.4.0 readiness preflight failed" >&2
  echo "release_v0_4_0_gate: $release_artifact remains blocked" >&2
  echo "release_v0_4_0_gate: command: $release_gate_command" >&2
  exit 1
fi

record_known_step \
  "readiness preflight" \
  "pass" \
  "$readiness_duration" \
  0 \
  "$readiness_log_rel" \
  "$readiness_command"

echo "release_v0_4_0_gate: bootstrapping local binaries before version preflight" >&2
bash scripts/dev/bootstrap.sh >&2

run_step "version parity" check_versions
run_step "readiness validator tests" \
  go test \
    ./tools/cmd/validate-v0-4-readiness \
    ./tools/cmd/validate-v0-4-completion-audit \
    -count=1
run_step "docs verification" go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
run_step "techempower report schemas" check_techempower_reports
run_step "compiler cli tools baseline" check_go_test_packages
run_step "memory core v2 gate" run_memory_core_v2_gate
run_step "memory production linux x64 smoke" run_memory_production_smoke
run_step "validate memory production" \
  go run ./tools/cmd/validate-memory-production \
    --report "$report_dir/artifacts/memory-production-linux-x64.json"
run_step "parallel production linux x64 smoke" run_parallel_production_smoke
run_step "validate parallel production" \
  go run ./tools/cmd/validate-parallel-production \
    --report "$report_dir/artifacts/parallel-production-linux-x64.json"
run_step "compiler production linux x64 smoke" run_compiler_production_smoke
run_step "validate compiler production" \
  go run ./tools/cmd/validate-compiler-production \
    --report "$report_dir/artifacts/compiler-production-linux-x64.json"
run_step "linux host smoke" run_linux_host_smoke
run_step "distributed actors linux x64 smoke" run_distributed_actor_smoke
run_step "validate distributed actor runtime" \
  go run ./tools/cmd/validate-distributed-actor-runtime \
    --report "$report_dir/artifacts/distributed-actors-linux-x64.json"
run_step "native ui linux x64 smoke" run_native_ui_smoke
run_step "validate native ui runtime" \
  go run ./tools/cmd/validate-native-ui-runtime \
    --report "$report_dir/artifacts/native-ui-linux-x64.json"
run_step "readiness final" check_readiness_final
run_step "completion audit validation" \
  go run ./tools/cmd/validate-v0-4-completion-audit \
    --audit docs/release/v0_4/v0_4_0_completion_audit.md \
    --expected-status achieved
run_step "release state" check_release_state
run_step "security review signoff" check_security_review_signoff
run_step "security review detached hash" write_security_review_hash
run_step "diff check" git diff --check

if [[ "$failed_count" -eq 0 ]]; then
  write_final_summary "pass"
  write_artifact_hashes
  validate_final_summary
  echo "release_v0_4_0_gate: $release_artifact passed for $release_version"
  exit 0
fi

write_final_summary "blocked"
write_artifact_hashes || true
validate_final_summary || true
echo "release_v0_4_0_gate: $release_artifact blocked with $failed_count failing step(s)" >&2
exit 1
