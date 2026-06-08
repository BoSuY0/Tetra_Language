#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
cd "$repo_root"

mode="full"
report_dir=""
keep_going=false
json_only=false

usage() {
  cat <<'USAGE'
Usage: bash scripts/ci/test-all.sh [--quick|--full|--stabilization] [--keep-going] [--json-only] [--report-dir DIR]

Modes:
  --quick          Run the fast stabilization gate for local iteration.
  --full           Run the full selected-release stabilization gate with logs and summaries.
  --stabilization  Run --full plus v0.3/v1.0-pre focused stabilization gates.

Output:
  --keep-going  Run remaining steps after a failure and exit 1 at the end.
  --json-only   Suppress progress logs and print summary JSON to stdout.

Artifacts:
  The script writes per-step logs plus summary.md and summary.json to DIR.
  summary.json records each step name, status, duration_seconds, exit_code, and log.
  It also includes top-level step_count and failed_count fields.
  release_artifact defaults to tetra.release.<version-slug>.test-all-summary.v1.
  If DIR is omitted, reports/test-all-<UTC timestamp> is used.

Exit codes:
  0  All selected checks passed.
  1  One or more checks failed.
  2  Usage/configuration error.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --quick)
      mode="quick"
      shift
      ;;
    --full)
      mode="full"
      shift
      ;;
    --stabilization)
      mode="stabilization"
      shift
      ;;
    --report-dir)
      if [[ $# -lt 2 ]]; then
        echo "--report-dir requires a directory" >&2
        exit 2
      fi
      report_dir="$2"
      shift 2
      ;;
    --keep-going)
      keep_going=true
      shift
      ;;
    --json-only)
      json_only=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

release_version="${TETRA_TEST_ALL_RELEASE_VERSION:-}"
if [[ -z "$release_version" ]]; then
  release_version="$(./tetra version)"
fi
release_slug="${release_version#v}"
release_slug="${release_slug//./_}"
release_artifact="${TETRA_TEST_ALL_RELEASE_ARTIFACT:-tetra.release.v${release_slug}.test-all-summary.v1}"

timestamp="$(date -u +%Y%m%d-%H%M%S)"
if [[ -z "$report_dir" ]]; then
  report_dir="reports/test-all-$timestamp"
fi

check_report_dir_fresh() {
  local find_report_dir="$report_dir"
  if [[ "$find_report_dir" == -* ]]; then
    find_report_dir="./$find_report_dir"
  fi

  if [[ -L "$find_report_dir" ]]; then
    if [[ -d "$find_report_dir" ]]; then
      local symlink_entry
      symlink_entry="$(find -H -- "$find_report_dir" -mindepth 1 -print -quit)"
      if [[ -n "$symlink_entry" ]]; then
        echo "refusing to reuse non-empty report directory: $report_dir" >&2
        echo "choose a fresh --report-dir so stale reports cannot be reused" >&2
        exit 2
      fi
    fi
    echo "refusing to use symlink report directory: $report_dir" >&2
    echo "choose a real fresh --report-dir so reports cannot escape the selected directory" >&2
    exit 2
  fi

  if [[ ( -e "$find_report_dir" || -L "$find_report_dir" ) && ! -d "$find_report_dir" ]]; then
    echo "refusing to use non-directory report path: $report_dir" >&2
    echo "choose a fresh --report-dir directory" >&2
    exit 2
  fi
  if [[ ! -d "$find_report_dir" ]]; then
    return 0
  fi
  local first_entry
  first_entry="$(find -H -- "$find_report_dir" -mindepth 1 -print -quit)"
  if [[ -n "$first_entry" ]]; then
    echo "refusing to reuse non-empty report directory: $report_dir" >&2
    echo "choose a fresh --report-dir so stale reports cannot be reused" >&2
    exit 2
  fi
}

check_report_dir_fresh

logs_dir="$report_dir/logs"
summary_md="$report_dir/summary.md"
summary_json="$report_dir/summary.json"
tmp_dir="$(mktemp -d)"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
step_count=0
failed_count=0

mkdir -p -- "$logs_dir"
: >"$tmp_dir/steps.md"
: >"$tmp_dir/steps.jsonl"

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

json_escape() {
  local s="${1-}"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  printf '%s' "$s"
}

write_summary() {
  local status="$1"
  local ended_at
  ended_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  {
    echo "# Tetra $release_version Test Report"
    echo
    echo "- mode: \`$mode\`"
    echo "- status: \`$status\`"
    echo "- started_at: \`$started_at\`"
    echo "- ended_at: \`$ended_at\`"
    echo "- step_count: \`$step_count\`"
    echo "- failed_count: \`$failed_count\`"
    echo "- release_version: \`$release_version\`"
    echo "- release_artifact: \`$release_artifact\`"
    echo
    echo "## Steps"
    echo
    cat -- "$tmp_dir/steps.md"
  } >"$summary_md"

  {
    echo "{"
    printf '  "mode": "%s",\n' "$(json_escape "$mode")"
    printf '  "status": "%s",\n' "$(json_escape "$status")"
    printf '  "started_at": "%s",\n' "$(json_escape "$started_at")"
    printf '  "ended_at": "%s",\n' "$(json_escape "$ended_at")"
    printf '  "step_count": %s,\n' "$step_count"
    printf '  "failed_count": %s,\n' "$failed_count"
    printf '  "release_version": "%s",\n' "$(json_escape "$release_version")"
    printf '  "release_artifact": "%s",\n' "$(json_escape "$release_artifact")"
    echo '  "steps": ['
    awk 'NR > 1 { printf ",\n" } { printf "    %s", $0 } END { if (NR > 0) printf "\n" }' "$tmp_dir/steps.jsonl"
    echo '  ]'
    echo "}"
  } >"$summary_json"
}

validate_summary() {
  go run ./tools/cmd/validate-test-all-summary --summary "$summary_json" --report-dir "$report_dir"
}

validate_summary_best_effort() {
  if ! validate_summary; then
    printf 'warning: summary validation failed; preserving original test failure\n' >&2
  fi
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

slugify() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//'
}

run_step() {
  local name="$1"
  shift
  step_count=$((step_count + 1))

  local step_id
  local slug
  local log_rel
  local log_path
  local command
  step_id="$(printf '%02d' "$step_count")"
  slug="$(slugify "$name")"
  log_rel="logs/${step_id}-${slug}.log"
  log_path="$report_dir/$log_rel"
  command="$*"

  if [[ "$json_only" != true ]]; then
    printf '== [%s] %s ==\n' "$step_id" "$name"
  fi
  local start_s
  local end_s
  start_s="$(date +%s)"

  if "$@" >"$log_path" 2>&1; then
    end_s="$(date +%s)"
    record_step "$name" "pass" "$((end_s - start_s))" 0 "$log_rel" "$command"
    if [[ "$json_only" != true ]]; then
      printf '   pass (%ss)\n' "$((end_s - start_s))"
    fi
  else
    local exit_code="$?"
    end_s="$(date +%s)"
    record_step "$name" "fail" "$((end_s - start_s))" "$exit_code" "$log_rel" "$command"
    failed_count=$((failed_count + 1))
    if [[ "$json_only" != true ]]; then
      printf '   fail (%ss). Last log lines:\n' "$((end_s - start_s))" >&2
      tail -n 80 "$log_path" >&2 || true
    fi
    if [[ "$keep_going" != true ]]; then
      write_summary "fail"
      validate_summary_best_effort
      if [[ "$json_only" == true ]]; then
        cat -- "$summary_json"
      else
        printf '\nFull report: %s\n' "$summary_md" >&2
      fi
      exit 1
    fi
  fi
}

check_working_tree_whitespace() {
  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git diff --check
  else
    echo "not a git work tree; skipping whitespace audit"
  fi
}

check_release_version() {
  local version
  version="$(./tetra version)"
  if [[ "$version" != "$release_version" ]]; then
    echo "expected $release_version, got $version" >&2
    exit 1
  fi
  echo "$version"
}

check_short_alias_version() {
  local version
  local short_version
  version="$(./tetra version)"
  short_version="$(./t version)"
  if [[ "$short_version" != "$version" ]]; then
    echo "expected ./t version to match ./tetra version ($version), got $short_version" >&2
    return 1
  fi
  echo "$short_version"
}

check_test_json() {
  ./tetra test --report=json examples >"$report_dir/tetra-test-report.json"
  test -s "$report_dir/tetra-test-report.json"
  go run ./tools/cmd/validate-test-report --report "$report_dir/tetra-test-report.json"
}

check_tetra_doc() {
  ./tetra doc examples >"$report_dir/tetra-docs.md"
  go run ./tools/cmd/validate-api-docs --docs "$report_dir/tetra-docs.md"
}

check_json_diagnostic_case() {
  local name="$1"
  local contains="$2"
  local source="$tmp_dir/$name.tetra"
  local stdout="$tmp_dir/$name.out"
  local diagnostic="$report_dir/$name.json"
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
  check_json_diagnostic_case "planned-actor-diagnostic" "actor declarations currently support state fields and func methods only" <<'TETRA'
actor Worker:
    return 0
TETRA

  local wasm_out="$tmp_dir/hello.wasm"
  ./tetra build --target wasm32-wasi -o "$wasm_out" examples/hello.tetra >"$tmp_dir/wasm-target-build.out" 2>"$report_dir/wasm-target-build.err"
  test -s "$wasm_out"
  test "$(od -An -tx1 -N4 "$wasm_out" | tr -d ' \n')" = "0061736d"
}

check_targets_report() {
  ./tetra targets --format=json >"$report_dir/targets.json"
  go run ./tools/cmd/validate-targets --report "$report_dir/targets.json"
}

check_doctor_report() {
  ./tetra doctor --format=json >"$report_dir/doctor.json"
  go run ./tools/cmd/validate-doctor --report "$report_dir/doctor.json"
}

check_docs_manifest() {
  go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
  go run ./tools/cmd/gen-manifest -o "$tmp_dir/manifest.json"
  go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json"
  diff -u docs/generated/manifest.json "$tmp_dir/manifest.json"
}

check_safety_readiness() {
  ./tetra features --format=json >"$report_dir/safety-features.json"
  go run ./tools/cmd/validate-safety-readiness \
    --features "$report_dir/safety-features.json" \
    --current-surface docs/spec/current_supported_surface.md \
    --ownership-spec docs/spec/ownership_v1.md \
    --effects-spec docs/spec/effects_capabilities_privacy_v1.md \
    --out "$report_dir/safety-readiness.json" || return 1
  go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Actor|Task|Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem' -count=1 || return 1
}

require_named_go_test_names() {
  local label="$1"
  shift
  local package="$1"
  local pattern="$2"
  shift 2
  local list_out
  if ! list_out="$(go test "$package" -list "$pattern")"; then
    return 1
  fi
  local name
  for name in "$@"; do
    if ! printf '%s\n' "$list_out" | grep -qx "$name"; then
      echo "missing required $label test: $package $name" >&2
      return 1
    fi
  done
}

require_go_test_names() {
  require_named_go_test_names "unsafe promotion blocker" "$@"
}

require_bounds_go_test_names() {
  require_named_go_test_names "bounds proof blocker" "$@"
}

require_memory_fuzz_go_test_names() {
  require_named_go_test_names "memory fuzz oracle gate" "$@"
}

require_host_leak_go_test_names() {
  require_named_go_test_names "host leak blocker" "$@"
}

check_unsafe_promotion_blockers() {
  require_go_test_names ./compiler/internal/memoryfacts 'UnsafeUnknown|UnsafeVerified|Promotion' \
    TestMemoryFactsRejectsUnsafeUnknownToSafeKnown \
    TestMemoryFactsRejectsDirectSafeBorrowedFromUnsafeUnknown \
    TestMemoryFactsRejectsDirectSafeOwnedFromUnsafeUnknown \
    TestMemoryFactsRejectsUnsafeUnknownNoAliasAndBoundsProofClaims \
    TestMemoryFactsRejectsUnsafeCheckedGenericPromotions \
    TestMemoryFactsRejectsUnsafeVerifiedRootGenericClaims \
    TestMemoryFactsRejectsValidatedUnsafeUnknownTrustedStorage \
    TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaims \
    TestValidateMemoryReportRejectsUnsafeCheckedGenericPromotions \
    TestValidateMemoryReportRejectsUnsafeVerifiedRootGenericClaims \
    TestValidateMemoryReportRejectsValidatedUnsafeUnknownTrustedStorage || return 1
  go test ./compiler/internal/memoryfacts -run 'UnsafeUnknown|UnsafeVerified|Promotion' -count=1 || return 1

  require_go_test_names ./tools/cmd/validate-memory-report 'Unsafe|Promotion|Optimization|TrustedStorage' \
    TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown \
    TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaim \
    TestValidateMemoryReportRejectsUnsafeCheckedGenericPromotion \
    TestValidateMemoryReportRejectsUnsafeUnknownZeroCost \
    TestValidateMemoryReportRejectsUnsafeVerifiedRootGenericClaim \
    TestValidateMemoryReportRejectsUnsafeUnknownTrustedStorage || return 1
  go test ./tools/cmd/validate-memory-report -run 'Unsafe|Promotion|Optimization|TrustedStorage' -count=1 || return 1

  require_go_test_names ./compiler 'Unsafe|Raw|MemoryFuzzOracle' \
    TestMemoryFuzzOracleReportCoversMPC15CategoriesAndInvariants \
    TestClassifyMemoryFuzzOracleObservation \
    TestValidateMemoryFuzzOracleReportRejectsDrift \
    TestMemoryFuzzOracleReportCoversV12ReleaseEvidence \
    TestValidateMemoryFuzzOracleReportRejectsV12ReleaseEvidenceDrift || return 1
  go test ./compiler -run 'Unsafe|Raw|MemoryFuzzOracle' -count=1 || return 1

  require_go_test_names ./tools/cmd/validate-memory-fuzz-oracle 'MemoryFuzzOracle|Unsafe|Promotion|Blocking' \
    TestValidateMemoryFuzzOracleReportFileAcceptsCompilerReport \
    TestValidateMemoryFuzzOracleReportFileRejectsInvalidReport \
    TestValidateMemoryFuzzOracleReportFileRejectsMissingV12ReleaseEvidence || return 1
  go test ./tools/cmd/validate-memory-fuzz-oracle -run 'MemoryFuzzOracle|Unsafe|Promotion|Blocking' -count=1
}

check_bounds_proof_blockers() {
  require_bounds_go_test_names ./compiler/internal/validation 'Bounds|Proof|Unchecked' \
    TestCheckBoundsProofsRejectsRemovedCheckWithoutProofID \
    TestCheckBoundsProofsWithPLIRRejectsUnknownLiveProof \
    TestValidateTranslationRejectsMissingProofIDAfterTransform || return 1
  require_bounds_go_test_names ./compiler/internal/plir 'Bounds|Proof|Unchecked' \
    TestVerifierRejectsUnknownProofUse \
    TestVerifierRejectsNonDominatingProofUse || return 1
  require_bounds_go_test_names ./compiler/internal/lower 'Bounds|Proof|Unchecked' \
    TestForSliceLoopUsesProofTaggedUncheckedIndexLoad \
    TestWhileLessThanLenUsesProofTaggedUncheckedIndexLoad \
    TestCopyLoopSourceLoadUsesProofTaggedUncheckedIndexLoad || return 1
  go test ./compiler/internal/plir ./compiler/internal/lower ./compiler/internal/validation -run 'Bounds|Proof|Unchecked' -count=1 || return 1

  require_bounds_go_test_names ./compiler/internal/memoryfacts 'Bounds|Proof' \
    TestMemoryIdealV6ProjectsBoundsProofFacts \
    TestMemoryIdealV6ProjectsMissingProofRejection \
    TestValidateMemoryReportRejectsV6BoundsRowsWithoutParent \
    TestValidateMemoryReportRejectsBareBoundsCheckEliminatedWithoutProofID || return 1
  require_bounds_go_test_names ./tools/cmd/validate-memory-report 'Bounds|Proof' \
    TestValidateMemoryReportRejectsV6BoundsRowsWithoutParent \
    TestValidateMemoryReportRejectsBareBoundsCheckEliminatedWithoutProofID || return 1
  go test ./compiler/internal/memoryfacts ./tools/cmd/validate-memory-report -run 'Bounds|Proof' -count=1 || return 1

  require_bounds_go_test_names ./compiler 'Bounds|MemoryFuzzOracle' \
    TestMemoryFuzzOracleReportCoversMPC15CategoriesAndInvariants \
    TestMemoryFuzzOracleReportCoversV12ReleaseEvidence \
    TestValidateMemoryFuzzOracleReportRejectsV12ReleaseEvidenceDrift \
    TestBuildBoundsAndProofReportsShowWhileRangeReason || return 1
  go test ./compiler -run 'Bounds|MemoryFuzzOracle' -count=1
}

check_memory_fuzz_oracle_gate() {
  local fuzz_dir="$report_dir/memory-fuzz-tier1"

  require_memory_fuzz_go_test_names ./tools/cmd/memory-fuzz-short 'MemoryFuzzShort|Tier|ReportDir' \
    TestRunMemoryFuzzShortWritesValidatedArtifacts \
    TestRunMemoryFuzzShortRejectsUnsupportedTier \
    TestRunMemoryFuzzShortRejectsStaleReportDir || return 1
  require_memory_fuzz_go_test_names ./tools/cmd/validate-memory-fuzz-oracle 'MemoryFuzzOracle|Artifact|Provenance' \
    TestValidateMemoryFuzzOracleReportFileAcceptsCompilerReport \
    TestValidateMemoryFuzzOracleReportFileAcceptsTier1ArtifactBundle \
    TestValidateMemoryFuzzOracleReportFileRejectsInvalidReport \
    TestValidateMemoryFuzzOracleReportFileRejectsMissingV12ReleaseEvidence \
    TestValidateMemoryFuzzOracleReportFileRejectsMissingArtifactSummary \
    TestValidateMemoryFuzzOracleReportFileRejectsMissingValidatorProvenance || return 1
  require_memory_fuzz_go_test_names ./compiler 'MemoryFuzzOracle' \
    TestMemoryFuzzOracleReportCoversMPC15CategoriesAndInvariants \
    TestClassifyMemoryFuzzOracleObservation \
    TestValidateMemoryFuzzOracleReportRejectsDrift \
    TestMemoryFuzzOracleReportCoversV12ReleaseEvidence \
    TestValidateMemoryFuzzOracleReportRejectsV12ReleaseEvidenceDrift || return 1

  go test ./compiler -run MemoryFuzzOracle -count=1 || return 1
  go test ./tools/cmd/memory-fuzz-short ./tools/cmd/validate-memory-fuzz-oracle -count=1 || return 1
  go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir "$fuzz_dir" || return 1
  go run ./tools/cmd/validate-memory-fuzz-oracle --report "$fuzz_dir/memory-fuzz-oracle.json" --artifact-dir "$fuzz_dir" || return 1
  test -s "$fuzz_dir/memory-fuzz-oracle.json" || return 1
  test -s "$fuzz_dir/summary.md" || return 1
  test -s "$fuzz_dir/summary.json"
}

check_host_leak_blockers() {
  require_host_leak_go_test_names ./cli/internal/actornet 'Broker|Leak|CloseWithoutCancel' \
    TestBrokerCloseWithoutCancelStopsServeWatcher \
    TestBrokerRoutesFramesBetweenLoopbackNodesAndWritesReport \
    TestBrokerReportsNodeDownForMissingDestination || return 1
  go test ./cli/internal/actornet -run 'Broker|Leak|CloseWithoutCancel' -count=1
}

check_performance_report() {
  local report="docs/generated/v1_0/performance-regression.json"
  if [[ ! -f "$report" ]]; then
    echo "performance report not found at $report; skipping in compatibility mode"
    return 0
  fi
  go run ./tools/cmd/validate-performance-report --report "$report"
}

check_techempower_reports() {
  local found=false
  local report
  local reports=(
    "docs/benchmarks/techempower_local_smoke_skip_db_report.json"
    "docs/benchmarks/techempower_scram_single_query_local_report.json"
    "docs/benchmarks/techempower_scram_single_query_matrix_local_report.json"
    "docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json"
  )
  for report in "${reports[@]}"; do
    if [[ ! -f "$report" ]]; then
      continue
    fi
    found=true
    if [[ "$report" == "docs/benchmarks/techempower_local_smoke_skip_db_report.json" ]]; then
      go run ./tools/cmd/validate-techempower-report --report "$report" --allow-skip-db
    else
      go run ./tools/cmd/validate-techempower-report --report "$report"
    fi
  done
  if [[ "$found" != true ]]; then
    echo "techempower reports not found; skipping in compatibility mode"
  fi
}

check_lsp_stdio() {
  local lsp_init
  local lsp_open
  local lsp_symbols
  local lsp_hover
  local lsp_completion
  local lsp_definition
  local lsp_references
  local lsp_rename
  local lsp_formatting
  local lsp_change
  local lsp_code_action
  local lsp_shutdown
  local lsp_exit
  lsp_init='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
  lsp_open='{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n  return answer\n"}}}'
  lsp_symbols='{"jsonrpc":"2.0","id":2,"method":"textDocument/documentSymbol","params":{"textDocument":{"uri":"file:///sample.tetra"}}}'
  lsp_hover='{"jsonrpc":"2.0","id":3,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":0,"character":6}}}'
  lsp_completion='{"jsonrpc":"2.0","id":4,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":9}}}'
  lsp_definition='{"jsonrpc":"2.0","id":5,"method":"textDocument/definition","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":9}}}'
  lsp_references='{"jsonrpc":"2.0","id":6,"method":"textDocument/references","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":9},"context":{"includeDeclaration":true}}}'
  lsp_rename='{"jsonrpc":"2.0","id":7,"method":"textDocument/rename","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":9},"newName":"value"}}'
  lsp_formatting='{"jsonrpc":"2.0","id":8,"method":"textDocument/formatting","params":{"textDocument":{"uri":"file:///sample.tetra"},"options":{"tabSize":4,"insertSpaces":true}}}'
  lsp_change='{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///sample.tetra","version":2},"contentChanges":[{"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    print(\"x\")\n    return answer\n"}]}}'
  lsp_code_action='{"jsonrpc":"2.0","id":9,"method":"textDocument/codeAction","params":{"textDocument":{"uri":"file:///sample.tetra"},"range":{"start":{"line":3,"character":4},"end":{"line":3,"character":9}},"context":{"diagnostics":[{"range":{"start":{"line":3,"character":4},"end":{"line":3,"character":9}},"severity":1,"code":"TETRA2001","source":"tetra","message":"function '\''main'\'' uses effect '\''io'\'' but does not declare it"}]}}}'
  lsp_shutdown='{"jsonrpc":"2.0","id":10,"method":"shutdown","params":{}}'
  lsp_exit='{"jsonrpc":"2.0","method":"exit","params":{}}'

  {
    for body in "$lsp_init" "$lsp_open" "$lsp_symbols" "$lsp_hover" "$lsp_completion" "$lsp_definition" "$lsp_references" "$lsp_rename" "$lsp_formatting" "$lsp_change" "$lsp_code_action" "$lsp_shutdown" "$lsp_exit"; do
      printf 'Content-Length: %s\r\n\r\n%s' "$(printf '%s' "$body" | wc -c)" "$body"
    done
  } | ./tetra lsp --stdio >"$tmp_dir/lsp-stdio.out"

  go run ./tools/cmd/validate-lsp-stdio --transcript "$tmp_dir/lsp-stdio.out"
  grep -q '"capabilities"' "$tmp_dir/lsp-stdio.out"
  grep -q '"textDocument/publishDiagnostics"' "$tmp_dir/lsp-stdio.out"
}

check_lsp_smoke() {
  ./tetra lsp --stdio-smoke examples/flow_hello.tetra >"$report_dir/lsp-smoke.json"
  go run ./tools/cmd/validate-lsp-smoke --report "$report_dir/lsp-smoke.json"
}

run_tetra_smoke_target() {
  local target="$1"
  local run_binaries="$2"
  local report_path="$3"
  ./tetra smoke --target "$target" --run="$run_binaries" --report "$report_path"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report_path"
}

check_host_smoke() {
  run_tetra_smoke_target "linux-x64" "true" "$report_dir/host-smoke.json"
}

check_smoke_list() {
  ./tetra smoke --list --target linux-x64 --format=json >"$report_dir/smoke-list.json"
  go run ./tools/cmd/validate-smoke-list --report "$report_dir/smoke-list.json" --examples-root examples
}

check_generated_api_docs() {
  go run ./tools/cmd/gen-docs examples >"$report_dir/api-docs.md"
  go run ./tools/cmd/validate-api-docs --docs "$report_dir/api-docs.md"
}

check_eco_suite() {
  mkdir -p "$tmp_dir/project/src"
  cat >"$tmp_dir/project/Tetra.capsule" <<'CAPSULE'
manifest "tetra.capsule.v1"
capsule App:
  id "tetra://app"
  version "0.1.0"
  target "linux-x64"
  permission "io"
  dependency "tetra://core" "0.1.0"
CAPSULE
  cat >"$tmp_dir/Core.capsule" <<'CAPSULE'
manifest "tetra.capsule.v1"
capsule Core:
  id "tetra://core"
  version "0.1.0"
  target "linux-x64"
  permission "io"
CAPSULE
  cat >"$tmp_dir/project/src/main.tetra" <<'TETRA'
func main() -> Int:
    return 0
TETRA

  ./tetra eco verify --target linux-x64 --lock "$tmp_dir/tetra.lock.json" "$tmp_dir/project/Tetra.capsule" "$tmp_dir/Core.capsule"
  go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"
  ./tetra eco seed export --out "$tmp_dir/tetra.seed.json" "$tmp_dir/project/Tetra.capsule" "$tmp_dir/Core.capsule"
  go run ./tools/cmd/validate-eco-seed --seed "$tmp_dir/tetra.seed.json"
  ./tetra eco seed import --seed "$tmp_dir/tetra.seed.json" --lock "$tmp_dir/tetra.seed.lock.json" --capsules-dir "$tmp_dir/seed-capsules"
  go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.seed.lock.json"
  ./tetra eco needmap --lock "$tmp_dir/tetra.lock.json" -o "$tmp_dir/tetra.needmap.json"
  go run ./tools/cmd/validate-eco-needmap --needmap "$tmp_dir/tetra.needmap.json"
  ./tetra eco pack "$tmp_dir/project/Tetra.capsule" -o "$tmp_dir/single.todex"
  ./tetra eco pack --project "$tmp_dir/project/Tetra.capsule" -o "$tmp_dir/project.todex"
  ./tetra eco unpack "$tmp_dir/project.todex" -C "$tmp_dir/unpacked"
  go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"
  test -f "$tmp_dir/unpacked/src/main.tetra"
  ./tetra eco vault add --store "$tmp_dir/vault" --kind source examples/flow_hello.tetra
  ./tetra eco vault list --store "$tmp_dir/vault"
  ./tetra eco vault verify --store "$tmp_dir/vault"
  go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"
  ./tetra eco trust snapshot --lock "$tmp_dir/tetra.lock.json" --store "$tmp_dir/vault" -o "$tmp_dir/tetra.trust-snapshot.json"
  go run ./tools/cmd/validate-eco-trust --trust "$tmp_dir/tetra.trust-snapshot.json"
  ./tetra eco materialize "$tmp_dir/project.todex" --target linux-x64 --trust "$tmp_dir/tetra.trust-snapshot.json" -C "$tmp_dir/materialized"
  test -f "$tmp_dir/materialized/tetra.materialization.json"
  go run ./tools/cmd/validate-eco-materialization --materialization "$tmp_dir/materialized/tetra.materialization.json"
  ./tetra eco publish --package "$tmp_dir/project.todex" --registry "$tmp_dir/registry" --target linux-x64 --trust "$tmp_dir/tetra.trust-snapshot.json"
  go run ./tools/cmd/validate-eco-publish --registry "$tmp_dir/registry" --id tetra://app --version 0.1.0 --target linux-x64
  ./tetra eco download --id tetra://app --version 0.1.0 --target linux-x64 --registry "$tmp_dir/registry" -o "$tmp_dir/downloaded.todex"
  test -f "$tmp_dir/downloaded.todex"
  ./tetra eco tetrahub publish --package "$tmp_dir/project.todex" --store "$tmp_dir/tetrahub-beta" --target linux-x64 --trust "$tmp_dir/tetra.trust-snapshot.json"
  ./tetra eco tetrahub download --id tetra://app --version 0.1.0 --target linux-x64 --store "$tmp_dir/tetrahub-beta" -o "$tmp_dir/hub-downloaded.todex"
  test -f "$tmp_dir/hub-downloaded.todex"
}

check_cross_target_smoke() {
  run_tetra_smoke_target "linux-x64" "false" "$tmp_dir/linux-smoke.json"
  run_tetra_smoke_target "macos-x64" "false" "$tmp_dir/macos-smoke.json"
  run_tetra_smoke_target "windows-x64" "false" "$tmp_dir/windows-smoke.json"
  run_tetra_smoke_target "wasm32-wasi" "false" "$report_dir/wasm32-wasi-artifact-smoke.json"
  run_tetra_smoke_target "wasm32-web" "false" "$report_dir/wasm32-web-artifact-smoke.json"
}

check_wasm_smoke_schema() {
  local target="$1"
  local report="$report_dir/$target-artifact-smoke.json"
  test -s "$report" || return 1
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report" || return 1
  if [[ "$target" == "wasm32-wasi" ]]; then
    go run ./tools/cmd/validate-wasi-smoke-report --mode artifact --report "$report" || return 1
  fi
}

write_tooling_summary() {
  local summary="$report_dir/tooling-summary.json"
  local summary_md="$report_dir/tooling-summary.md"
  local artifacts=(
    "targets.json"
    "doctor.json"
    "smoke-list.json"
    "tetra-test-report.json"
    "host-smoke.json"
    "safety-readiness.json"
    "lsp-smoke.json"
    "tetra-docs.md"
    "api-docs.md"
    "wasm32-wasi-artifact-smoke.json"
    "wasm32-web-artifact-smoke.json"
  )
  if [[ "$mode" == "stabilization" ]]; then
    artifacts+=(
      "wasi-smoke.json"
      "web-ui-smoke.json"
      "api-diff/api-diff.json"
    )
  fi
  local rel
  local failures=0

  {
    echo "{"
    echo '  "schema": "tetra.tooling-summary.v1alpha1",'
    printf '  "mode": "%s",\n' "$(json_escape "$mode")"
    printf '  "release_version": "%s",\n' "$(json_escape "$release_version")"
    printf '  "report_dir": "%s",\n' "$(json_escape "$report_dir")"
    echo '  "artifacts": ['
    local first=true
    local bytes
    for rel in "${artifacts[@]}"; do
      if [[ -f "$report_dir/$rel" ]]; then
        bytes="$(wc -c <"$report_dir/$rel" | tr -d ' ')"
      else
        bytes=0
      fi
      if [[ "$first" == true ]]; then
        first=false
      else
        echo ","
      fi
      if [[ -f "$report_dir/$rel" ]]; then
        printf '    {"path":"%s","required":true,"exists":true,"bytes":%s,"size_bytes":%s}' "$(json_escape "$rel")" "$bytes" "$bytes"
      else
        printf '    {"path":"%s","required":true,"exists":false,"bytes":0,"size_bytes":0}' "$(json_escape "$rel")"
      fi
    done
    echo
    echo '  ]'
    echo "}"
  } >"$summary"

  {
    echo "# Tooling Summary"
    echo
    echo "- schema: \`tetra.tooling-summary.v1alpha1\`"
    echo "- mode: \`$mode\`"
    echo "- release_version: \`$release_version\`"
    echo
    echo "## Artifacts"
    echo
    for rel in "${artifacts[@]}"; do
      if [[ -f "$report_dir/$rel" ]]; then
        bytes="$(wc -c <"$report_dir/$rel" | tr -d ' ')"
        echo "- \`$rel\`: required, exists:true, bytes:$bytes"
      else
        echo "- \`$rel\`: required, exists:false, bytes:0"
      fi
    done
  } >"$summary_md"

  for rel in "${artifacts[@]}"; do
    if [[ ! -f "$report_dir/$rel" ]]; then
      printf 'required artifact missing: %s\n' "$rel" >&2
      failures=$((failures + 1))
    elif [[ ! -s "$report_dir/$rel" ]]; then
      printf 'required artifact is zero-byte: %s\n' "$rel" >&2
      failures=$((failures + 1))
    fi
  done

  if [[ "$failures" -gt 0 ]]; then
    return 1
  fi
}

if [[ "$json_only" != true ]]; then
  printf 'Tetra %s test wrapper\n' "$release_version"
  printf 'mode: %s\n' "$mode"
  printf 'report_dir: %s\n\n' "$report_dir"
fi

run_step "go test all packages" env -u TETRA_TEST_ALL_RELEASE_VERSION -u TETRA_TEST_ALL_RELEASE_ARTIFACT -u TETRA_SECURITY_REVIEW_SIGNOFF go test ./compiler/... ./cli/... ./tools/... -count=1
run_step "unsafe promotion blocker suite" check_unsafe_promotion_blockers
run_step "bounds proof blocker suite" check_bounds_proof_blockers
run_step "memory fuzz oracle artifact gate" check_memory_fuzz_oracle_gate
run_step "host leak blocker suite" check_host_leak_blockers

if [[ "$mode" == "full" || "$mode" == "stabilization" ]]; then
  run_step "repo test script" env -u TETRA_TEST_ALL_RELEASE_VERSION -u TETRA_TEST_ALL_RELEASE_ARTIFACT -u TETRA_SECURITY_REVIEW_SIGNOFF bash scripts/ci/test.sh
fi

run_step "bootstrap" bash scripts/dev/bootstrap.sh
run_step "version preflight" check_release_version
run_step "short alias version" check_short_alias_version
run_step "formatter check examples lib runtime" ./tetra fmt --check examples lib __rt compiler/selfhostrt
run_step "flow-only source scan" go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
run_step "targets json report" check_targets_report
run_step "doctor json report" check_doctor_report
run_step "tetra check flow hello" ./tetra check examples/flow_hello.tetra
run_step "json diagnostic shape" check_json_diagnostic
run_step "smoke list json report" check_smoke_list
run_step "tetra test examples" ./tetra test examples

if [[ "$mode" == "full" || "$mode" == "stabilization" ]]; then
  run_step "tetra test json report" check_test_json
fi

run_step "host smoke linux-x64" check_host_smoke

if [[ "$mode" == "full" || "$mode" == "stabilization" ]]; then
  run_step "docs manifest diff" check_docs_manifest
  run_step "docs verification" go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
  run_step "safety readiness evidence" check_safety_readiness
  run_step "ownership production audit" go run ./tools/cmd/validate-ownership-audit --audit docs/release/ownership_production_audit.md --expected-status achieved
  run_step "performance report schema" check_performance_report
  run_step "techempower report schemas" check_techempower_reports
  run_step "lsp stdio smoke" check_lsp_smoke
  run_step "lsp json-rpc stdio" check_lsp_stdio
  run_step "tetra doc examples" check_tetra_doc
  run_step "generated api docs" check_generated_api_docs
  run_step "eco graph bundle vault" check_eco_suite
  run_step "cross-target build smoke" check_cross_target_smoke
  run_step "wasm32-wasi smoke schema" check_wasm_smoke_schema "wasm32-wasi"
  run_step "wasm32-web smoke schema" check_wasm_smoke_schema "wasm32-web"
fi

if [[ "$mode" == "stabilization" ]]; then
  run_step "compiler pipeline focused gate" go test ./compiler/... -run 'Pipeline|Build|Target|Backend|Cache|Object|Link|Stats|Runtime|ABI' -count=1
  run_step "frontend callable focused gate" go test ./compiler/... -run 'Diagnostic|Parser|Frontend|Flow|Lexer|FunctionTyped|Callable|Closure|Type|Inference|Enum|Optional|Protocol|Extension|Module' -count=1
  run_step "safety runtime focused gate" go test ./compiler/... -run 'Ownership|Borrow|Lifetime|Island|Actor|Task|Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem|Async|Await|TypedError|Stress|SelfHost|Builtin' -count=1
  run_step "lowering ir focused gate" go test ./compiler/internal/lower ./compiler -run 'Lower|IR|Verify|Unsupported|Loop|Task|Actor|UI|Unsafe|Runtime|Island|Budget' -count=1
  run_step "wasm ui focused gate" go test ./compiler/... -run 'UI|View|State|Style|Accessibility|NativeShell|WASM|Web' -count=1
  run_step "eco dx focused gate" go test ./cli/... ./tools/... -run 'Eco|Project|Workspace|Permission|Capsule|Trust|Lock|Vault|Publish|Unpack|Materialize|Download|TetraHub' -count=1
  run_step "lsp docs validators focused gate" go test ./tools/cmd/validate-lsp-stdio/... ./tools/cmd/validate-lsp-smoke/... ./tools/cmd/validate-diagnostic/... ./tools/cmd/validate-api-docs/... ./tools/cmd/validate-wasi-smoke-report/... ./tools/cmd/verify-docs/... -count=1
  run_step "wasi runner smoke" bash scripts/release/v1_0/wasi-smoke.sh --report "$report_dir/wasi-smoke.json"
  run_step "web runtime browser smoke" bash scripts/release/v1_0/web-smoke.sh --report "$report_dir/web-ui-smoke.json"
  run_step "api diff no-change" bash scripts/release/v1_0/api-diff.sh --report-dir "$report_dir/api-diff" --baseline docs/baselines/api-diff-baseline.v1alpha1.json --enforce no-change
  run_step "working tree whitespace audit" check_working_tree_whitespace
fi

if [[ "$mode" == "full" || "$mode" == "stabilization" ]]; then
  run_step "tooling summary aggregation" write_tooling_summary
fi

if [[ "$failed_count" -gt 0 ]]; then
  write_summary "fail"
  validate_summary_best_effort
  if [[ "$json_only" == true ]]; then
    cat -- "$summary_json"
  else
    printf '\n%s %s check(s) failed.\n' "$failed_count" "$mode" >&2
    printf 'Summary: %s\n' "$summary_md" >&2
    printf 'JSON: %s\n' "$summary_json" >&2
  fi
  exit 1
fi

write_summary "pass"
validate_summary
if [[ "$json_only" == true ]]; then
  cat -- "$summary_json"
else
  printf '\nAll %s checks passed.\n' "$mode"
  printf 'Summary: %s\n' "$summary_md"
  printf 'JSON: %s\n' "$summary_json"
fi
