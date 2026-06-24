package testall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func shellScript(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}

func testAllFakeRepo(t *testing.T, failFmt bool) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "ci"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "release", "v1_0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "release", "post_v0_4"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(
		filepath.Join(repoRoot(t), "scripts", "ci", "test-all.sh"),
		filepath.Join(root, "scripts", "ci", "test-all.sh"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "scripts", "dev", "bootstrap.sh"),
		[]byte("#!/usr/bin/env bash\nset -euo pipefail\ncp ./tetra ./t\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "scripts", "ci", "test.sh"),
		[]byte("#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "scripts", "release", "v1_0", "wasi-smoke.sh"),
		shellScript(
			"#!/usr/bin/env bash",
			"set -euo pipefail",
			`report=""`,
			`while [[ $# -gt 0 ]]; do`,
			`  case "$1" in`,
			`    --report) report="$2"; shift 2 ;;`,
			`    *) shift ;;`,
			`  esac`,
			`done`,
			`mkdir -p "$(dirname "$report")"`,
			`printf '{"status":"pass","cases":[]}\n' >"$report"`,
		),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "scripts", "release", "v1_0", "web-smoke.sh"),
		shellScript(
			"#!/usr/bin/env bash",
			"set -euo pipefail",
			`report=""`,
			`while [[ $# -gt 0 ]]; do`,
			`  case "$1" in`,
			`    --report) report="$2"; shift 2 ;;`,
			`    *) shift ;;`,
			`  esac`,
			`done`,
			`if [[ "${TETRA_FAKE_SKIP_WEB_UI_SMOKE_REPORT:-}" == "1" ]]; then`,
			`  exit 0`,
			`fi`,
			`mkdir -p "$(dirname "$report")"`,
			`printf '{"status":"pass","ui_schema":"tetra.ui.bundle.v1","cases":[]}\n' >"$report"`,
		),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "scripts", "release", "v1_0", "api-diff.sh"),
		shellScript(
			"#!/usr/bin/env bash",
			"set -euo pipefail",
			`report_dir=""`,
			`while [[ $# -gt 0 ]]; do`,
			`  case "$1" in`,
			`    --report-dir) report_dir="$2"; shift 2 ;;`,
			`    *) shift ;;`,
			`  esac`,
			`done`,
			`mkdir -p "$report_dir"`,
			`cat >"$report_dir/api-diff.json" <<'JSON'`,
			`{"review":{"status":"clean"},"diff":{"added":[],"removed":[],"changed":[]}}`,
			`JSON`,
		),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "scripts", "release", "post_v0_4", "memory-100-prod-stable-gate.sh"),
		shellScript(
			"#!/usr/bin/env bash",
			"set -euo pipefail",
			`report_dir=""`,
			`while [[ $# -gt 0 ]]; do`,
			`  case "$1" in`,
			`    --report-dir) report_dir="$2"; shift 2 ;;`,
			`    *) shift ;;`,
			`  esac`,
			`done`,
			`mkdir -p -- "$report_dir"`,
			`cat >"$report_dir/memory-100-prod-stable-manifest.json" <<'JSON'`,
			`{"schema":"tetra.memory-100.prod-stable.v1","status":"pass"}`,
			`JSON`,
			`cat >"$report_dir/artifact-hashes.json" <<'JSON'`,
			`{"schema":"tetra.artifact-hashes.v1","artifacts":[]}`,
			`JSON`,
		),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "docs", "generated", "manifest.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goScript := `#!/usr/bin/env bash
set -euo pipefail
fake_go_original_argv=("$@")
fake_go_script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P 2>/dev/null || true)"
fake_go_repo_root=""
if [[ -n "$fake_go_script_dir" ]]; then
  fake_go_repo_root="$(cd "$fake_go_script_dir/.." && pwd -P 2>/dev/null || true)"
fi
fake_go_trace_file=""
if [[ -n "$fake_go_repo_root" ]]; then
  mkdir -p "$fake_go_repo_root/.test-all-trace" 2>/dev/null || true
  fake_go_trace_file="$(mktemp "$fake_go_repo_root/.test-all-trace/go.XXXXXX.trace" 2>/dev/null || true)"
fi
trace_command="${1:-}"
trace_package=""
trace_list_mode=0
trace_list_pattern=""
trace_selected_list_case=""
trace_list_result=""
trace_emitted_line_count=0
trace_target_host_report_env_present=0
if [[ -n "${TETRA_WINDOWS_UI_RUNTIME_REPORT:-}" ||
  -n "${TETRA_MACOS_UI_RUNTIME_REPORT:-}" ]]; then
  trace_target_host_report_env_present=1
fi
fake_go_cwd_relative() {
  local cwd
  cwd="$(pwd -P 2>/dev/null || true)"
  if [[ -n "$fake_go_repo_root" && "$cwd" == "$fake_go_repo_root" ]]; then
    printf '.'
  elif [[ -n "$fake_go_repo_root" && "$cwd" == "$fake_go_repo_root/"* ]]; then
    printf '%s' "${cwd#"$fake_go_repo_root/"}"
  else
    printf '<outside-fake-repo>'
  fi
}
fake_go_argv() {
  local first=1
  local arg
  for arg in "$@"; do
    if [[ "$first" == 0 ]]; then
      printf ' '
    fi
    first=0
    printf '%q' "$(fake_go_sanitize_arg "$arg")"
  done
}
fake_go_sanitize_arg() {
  local arg="$1"
  local upper="${arg^^}"
  case "$upper" in
    *AUTHORIZATION*|*CREDENTIAL*|*PASSWORD*|*SECRET*|*TOKEN*)
      printf '<redacted>'
      return 0
      ;;
  esac
  if [[ -n "$fake_go_repo_root" && "$arg" == "$fake_go_repo_root" ]]; then
    printf '<fake-repo-path>'
  elif [[ -n "$fake_go_repo_root" && "$arg" == "$fake_go_repo_root/"* ]]; then
    printf '<fake-repo-path>'
  else
    printf '%s' "$arg"
  fi
}
fake_go_write_trace() {
  local exit_code="$1"
  shift || true
  [[ -n "$fake_go_trace_file" ]] || return 0
  {
    printf 'schema=test_all_fake_go_trace_v1\n'
    printf 'pid=%s\n' "$$"
    printf 'ppid=%s\n' "$PPID"
    printf 'cwd_relative_to_fake_repo=%s\n' "$(fake_go_cwd_relative)"
    printf 'argv='
    fake_go_argv "$@"
    printf '\n'
    printf 'command=%s\n' "$trace_command"
    printf 'package=%s\n' "$trace_package"
    printf 'list_mode=%s\n' "$trace_list_mode"
    printf 'list_pattern=%s\n' "$trace_list_pattern"
    printf 'skip_unsafe_present=%s\n' "$([[ -n "${TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST:-}" ]] && printf 1 || printf 0)"
    printf 'skip_bounds_present=%s\n' "$([[ -n "${TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST:-}" ]] && printf 1 || printf 0)"
    printf 'skip_host_leak_present=%s\n' "$([[ -n "${TETRA_FAKE_SKIP_HOST_LEAK_LIST:-}" ]] && printf 1 || printf 0)"
    printf 'skip_memory_fuzz_present=%s\n' "$([[ -n "${TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST:-}" ]] && printf 1 || printf 0)"
    printf 'skip_ram_contract_present=%s\n' "$([[ -n "${TETRA_FAKE_SKIP_RAM_CONTRACT_LIST:-}" ]] && printf 1 || printf 0)"
    printf 'target_host_report_env_present=%s\n' "$trace_target_host_report_env_present"
    printf 'selected_list_case=%s\n' "$trace_selected_list_case"
    printf 'list_result=%s\n' "$trace_list_result"
    printf 'emitted_line_count=%s\n' "$trace_emitted_line_count"
    printf 'exit_code=%s\n' "$exit_code"
  } >"$fake_go_trace_file" 2>/dev/null || true
}
trap 'fake_go_write_trace "$?" "${fake_go_original_argv[@]}"' EXIT
emit_fake_go_list() {
  trace_selected_list_case="$1"
  shift
  trace_list_result=normal
  trace_emitted_line_count="$#"
  printf '%s\n' "$@"
}
if [[ -n "${TETRA_FAKE_GO_LOG:-}" ]]; then
  printf '%s\n' "$*" >>"$TETRA_FAKE_GO_LOG"
fi
if [[ "${TETRA_FAKE_FORBID_TARGET_HOST_REPORT_ENV:-}" == "1" ]]; then
  if [[ -n "${TETRA_WINDOWS_UI_RUNTIME_REPORT:-}" ||
    -n "${TETRA_MACOS_UI_RUNTIME_REPORT:-}" ]]; then
    echo "target-host report env leaked into fake go" >&2
    exit 44
  fi
fi
emit_tetra_api_metadata() {
  printf '%s' '<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1",'
  printf '%s' '"api_hash":"sha256:ede46e5e34948c25f6ec38b0b963a2d8d42f5aa'
  printf '%s' '09071128581ee08271e966459",'
  printf '%s\n\n' '"module_count":1,"entry_count":1} -->'
}
if [[ "${1:-}" == "run" ]] &&
  [[ "${2:-}" == "./tools/cmd/validate-test-all-summary" ]] &&
  [[ "${TETRA_FAIL_SUMMARY_VALIDATOR:-}" == "1" ]]; then
  echo "summary validator unavailable" >&2
  exit 23
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/json-to-toon" ]]; then
  out=""
  shift 2
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --out)
        out="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -n "$out" ]]; then
    mkdir -p "$(dirname "$out")"
    printf 'status: pass\n' >"$out"
  fi
  exit 0
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/gen-manifest" ]]; then
  out=""
  shift 2
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -o)
        out="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -n "$out" ]]; then
    printf '{}\n' >"$out"
  fi
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/gen-docs" ]]; then
  printf '%s\n' '# Generated Tetra API Docs' ''
  emit_tetra_api_metadata
  printf '%s\n' '## examples' '' '### Functions' ''
  printf '%b\n' '- \x60func main() -> Int\x60'
  exit 0
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/validate-safety-readiness" ]]; then
  if [[ "${TETRA_FAIL_SAFETY_READINESS:-}" == "1" ]]; then
    echo "safety readiness failed" >&2
    exit 19
  fi
  out=""
  shift 2
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --out)
        out="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -n "$out" ]]; then
    mkdir -p "$(dirname "$out")"
    cat >"$out" <<'JSON'
{"schema":"tetra.safety-readiness.v1","status":"pass","version":"v0.4.0","required_features":[]}
JSON
  fi
  exit 0
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/memory-fuzz-short" ]]; then
  report_dir=""
  shift 2
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --report-dir)
        report_dir="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -n "$report_dir" ]]; then
    mkdir -p -- "$report_dir"
    cat >"$report_dir/memory-fuzz-oracle.json" <<'JSON'
{"schema_version":"tetra.memory-fuzz.oracle.v1","scope":"memory_production_core_v1_mpc15"}
JSON
    cat >"$report_dir/summary.md" <<'MD'
# Memory Fuzz Short Summary

- tier: Tier 1 short CI smoke
- report: memory-fuzz-oracle.json
MD
    memory_fuzz_cmd="go run ./tools/cmd/memory-fuzz-short"
    memory_fuzz_cmd="$memory_fuzz_cmd --tier 1 --report-dir <artifact-dir>"
    validate_fuzz_cmd="go run ./tools/cmd/validate-memory-fuzz-oracle"
    validate_fuzz_cmd="$validate_fuzz_cmd --report <artifact-dir>/memory-fuzz-oracle.json"
    validate_fuzz_cmd="$validate_fuzz_cmd --artifact-dir <artifact-dir>"
    cat >"$report_dir/summary.json" <<JSON
{
  "schema_version": "tetra.memory-fuzz-short.summary.v1",
  "kind": "tier1_short_ci_smoke",
  "tier": "tier1_short_ci_smoke",
  "status": "pass",
  "artifacts": {
    "oracle_report": "memory-fuzz-oracle.json",
    "summary_md": "summary.md",
    "summary_json": "summary.json"
  },
  "commands": [
    {
      "name": "memory-fuzz-short",
      "command": "$memory_fuzz_cmd",
      "status": "pass"
    },
    {
      "name": "validate-memory-fuzz-oracle",
      "command": "$validate_fuzz_cmd",
      "status": "pass"
    }
  ]
}
JSON
  fi
  exit 0
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/ram-contract-fuzz-short" ]]; then
  report_dir=""
  shift 2
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --report-dir)
        report_dir="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -n "$report_dir" ]]; then
    mkdir -p -- "$report_dir"
    cat >"$report_dir/ram-contract-fuzz-oracle.json" <<JSON
{"schema":"tetra.ram-contract-fuzz-oracle.v1","status":"pass","artifact_dir":"$report_dir"}
JSON
    cat >"$report_dir/ram-contract-report.json" <<'JSON'
{
  "schema_version": "tetra.ram-contract-report.v1",
  "entrypoint": "main",
  "target": "linux-x64",
  "git_head": "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "rows": [],
  "blockers": [],
  "summary": {
    "total_heap_bytes": 0,
    "total_copy_bytes": 0,
    "unbounded_rows": 0,
    "heap_blockers": 0,
    "copy_blockers": 0
  }
}
JSON
    cat >"$report_dir/memory-grade-report.json" <<'JSON'
{
  "schema_version": "tetra.memory-grade-report.v1",
  "grade": "M0",
  "reasons": ["no heap or copy blockers"],
  "ram_contract_report": "ram-contract-report.json"
}
JSON
    cat >"$report_dir/proof-store-summary.json" <<'JSON'
{"schema_version":"tetra.proof-store-summary.v1","proofs":[],"invalid_references":[]}
JSON
    cat >"$report_dir/validation-pipeline-coverage.json" <<'JSON'
{
  "schema_version": "tetra.validation-pipeline-coverage.v1",
  "stages": [{"name":"ram-contract","status":"pass"}]
}
JSON
    cat >"$report_dir/heap-blockers.json" <<'JSON'
{"schema_version":"tetra.ram-blockers.v1","kind":"heap","blockers":[]}
JSON
    cat >"$report_dir/copy-blockers.json" <<'JSON'
{"schema_version":"tetra.ram-blockers.v1","kind":"copy","blockers":[]}
JSON
  fi
  exit 0
fi
if [[ "${1:-}" == "test" ]]; then
  pkg="${2:-}"
  shift 2 || true
  list_mode=false
  list_pattern=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -list)
        list_mode=true
        list_pattern="${2:-}"
        shift 2 || true
        ;;
      -list=*)
        list_mode=true
        list_pattern="${1#-list=}"
        shift
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ "$list_mode" == true ]]; then
    trace_package="$pkg"
    trace_list_mode=1
    trace_list_pattern="$list_pattern"
    if [[ "${TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST:-}" == "1" ]] &&
      [[ "$list_pattern" == *Unsafe* ]]; then
      trace_selected_list_case=skip_unsafe_control
      trace_list_result=skipped_by_explicit_control
      exit 0
    fi
    if [[ "${TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST:-}" == "1" ]]; then
      case "$list_pattern" in
        *Bounds*|*Proof*|*Unchecked*)
          trace_selected_list_case=skip_bounds_control
          trace_list_result=skipped_by_explicit_control
          exit 0
          ;;
      esac
    fi
    if [[ "${TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST:-}" == "1" ]] &&
      [[ "$pkg" == "./tools/cmd/memory-fuzz-short" ]]; then
      trace_selected_list_case=skip_memory_fuzz_control
      trace_list_result=skipped_by_explicit_control
      exit 0
    fi
		if [[ "${TETRA_FAKE_SKIP_HOST_LEAK_LIST:-}" == "1" && "$pkg" == "./cli/internal/actornet" ]]; then
			trace_selected_list_case=skip_host_leak_control
			trace_list_result=skipped_by_explicit_control
			exit 0
		fi
    if [[ "${TETRA_FAKE_SKIP_RAM_CONTRACT_LIST:-}" == "1" ]] &&
      [[ "$pkg" == "./tools/cmd/ram-contract-fuzz-short" ]]; then
      trace_selected_list_case=skip_ram_contract_control
      trace_list_result=skipped_by_explicit_control
      exit 0
    fi
    case "$pkg" in
      ./compiler/internal/memoryfacts)
        emit_fake_go_list "$pkg" \
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
          TestValidateMemoryReportRejectsValidatedUnsafeUnknownTrustedStorage \
          TestMemoryIdealV6ProjectsBoundsProofFacts \
          TestMemoryIdealV6ProjectsMissingProofRejection \
          TestValidateMemoryReportRejectsV6BoundsRowsWithoutParent \
          TestValidateMemoryReportRejectsBareBoundsCheckEliminatedWithoutProofID
        ;;
      ./compiler/cmd/validate-memory-report)
        emit_fake_go_list "$pkg" \
          TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown \
          TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaim \
          TestValidateMemoryReportRejectsUnsafeCheckedGenericPromotion \
          TestValidateMemoryReportRejectsUnsafeUnknownZeroCost \
          TestValidateMemoryReportRejectsUnsafeVerifiedRootGenericClaim \
          TestValidateMemoryReportRejectsUnsafeUnknownTrustedStorage \
          TestValidateMemoryReportRejectsV6BoundsRowsWithoutParent \
          TestValidateMemoryReportRejectsBareBoundsCheckEliminatedWithoutProofID
        ;;
      ./compiler)
        emit_fake_go_list "$pkg" \
          TestMemoryFuzzOracleReportCoversMPC15CategoriesAndInvariants \
          TestClassifyMemoryFuzzOracleObservation \
          TestValidateMemoryFuzzOracleReportRejectsDrift \
          TestMemoryFuzzOracleReportCoversV12ReleaseEvidence \
          TestValidateMemoryFuzzOracleReportRejectsV12ReleaseEvidenceDrift \
          TestBuildBoundsAndProofReportsShowWhileRangeReason
        ;;
      ./compiler/internal/validation)
        emit_fake_go_list "$pkg" \
          TestCheckBoundsProofsRejectsRemovedCheckWithoutProofID \
          TestCheckBoundsProofsWithPLIRRejectsUnknownLiveProof \
          TestValidateTranslationRejectsMissingProofIDAfterTransform
        ;;
      ./compiler/internal/plir)
        emit_fake_go_list "$pkg" \
          TestVerifierRejectsUnknownProofUse \
          TestVerifierRejectsNonDominatingProofUse
        ;;
      ./compiler/internal/lower)
        emit_fake_go_list "$pkg" \
          TestForSliceLoopUsesProofTaggedUncheckedIndexLoad \
          TestWhileLessThanLenUsesProofTaggedUncheckedIndexLoad \
          TestCopyLoopSourceLoadUsesProofTaggedUncheckedIndexLoad
        ;;
      ./tools/cmd/validate-memory-fuzz-oracle)
        emit_fake_go_list "$pkg" \
          TestValidateMemoryFuzzOracleReportFileAcceptsCompilerReport \
          TestValidateMemoryFuzzOracleReportFileAcceptsTier1ArtifactBundle \
          TestValidateMemoryFuzzOracleReportFileRejectsInvalidReport \
          TestValidateMemoryFuzzOracleReportFileRejectsMissingV12ReleaseEvidence \
          TestValidateMemoryFuzzOracleReportFileRejectsMissingArtifactSummary \
          TestValidateMemoryFuzzOracleReportFileRejectsMissingValidatorProvenance
        ;;
      ./tools/cmd/memory-fuzz-short)
        emit_fake_go_list "$pkg" \
          TestRunMemoryFuzzShortWritesValidatedArtifacts \
          TestRunMemoryFuzzShortRejectsUnsupportedTier \
          TestRunMemoryFuzzShortRejectsStaleReportDir
        ;;
      ./tools/cmd/ram-contract-fuzz-short)
        emit_fake_go_list "$pkg" \
          TestRunRAMContractFuzzShortWritesValidatedArtifacts \
          TestRunRAMContractFuzzShortRejectsStaleReportDir
        ;;
      ./tools/cmd/validate-ram-contract-fuzz-oracle)
        emit_fake_go_list "$pkg" \
          TestValidateRAMContractFuzzOracleAcceptsArtifactBundle \
          TestValidateRAMContractFuzzOracleRejectsMissingReport
        ;;
      ./tools/cmd/validate-ram-contract-report)
        emit_fake_go_list "$pkg" \
          TestValidateRAMContractReportFileAcceptsCompilerReport \
          TestValidateRAMContractReportRejectsMissingBlocker
        ;;
      ./compiler/internal/ramcontract)
        emit_fake_go_list "$pkg" \
          TestRAMContractFromAllocPlanTracksRowsAndBlockers \
          TestRAMContractRejectsMissingBlockerExplanation \
          TestRAMContractEnforcementFailsForHeap
        ;;
      ./cli/internal/actornet)
        emit_fake_go_list "$pkg" \
          TestBrokerCloseReopenWithoutGoroutineLeak \
          TestBrokerCloseWithoutCancelStopsServeWatcher \
          TestBrokerRoutesFramesBetweenLoopbackNodesAndWritesReport \
          TestBrokerReportsNodeDownForMissingDestination
        ;;
    esac
    if [[ -z "$trace_list_result" ]]; then
      trace_selected_list_case=unknown_package
      trace_list_result=unknown_package
      trace_emitted_line_count=0
    fi
    exit 0
  fi
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(goScript), 0o755); err != nil {
		t.Fatal(err)
	}
	gitScript := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "rev-parse" && "${2:-}" == "HEAD" ]]; then
  echo "e2c19b8ee276158f8eb2c54cf61e11bd84952893"
  exit 0
fi
if [[ "${1:-}" == "diff" && "${2:-}" == "--check" ]]; then
  exit 0
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(gitScript), 0o755); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/usr/bin/env bash
set -euo pipefail
emit_tetra_api_metadata() {
  printf '%s' '<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1",'
  printf '%s' '"api_hash":"sha256:ede46e5e34948c25f6ec38b0b963a2d8d42f5aa'
  printf '%s' '09071128581ee08271e966459",'
  printf '%s\n\n' '"module_count":1,"entry_count":1} -->'
}
emit_diag() {
  printf '{"code":"%s","message":"%s","severity":"error"}\n' "$1" "$2" >&2
}
cmd="${1:-}"
shift || true
	case "$cmd" in
	  version)
	    echo "${TETRA_FAKE_TETRA_VERSION:-v0.4.0}"
	    ;;
  fmt)
    if [[ "${TETRA_FAIL_FMT:-}" == "1" ]]; then
      echo "format mismatch" >&2
      exit 7
    fi
    ;;
  test)
    for arg in "$@"; do
      if [[ "$arg" == "--report=json" ]]; then
        echo '{"total":0,"passed":0,"failed":0,"files":[],"results":[]}'
        exit 0
      fi
    done
    ;;
  check)
    for arg in "$@"; do
      if [[ "$arg" == "--diagnostics=json" ]]; then
        case "$*" in
          *missing-effect-diagnostic.tetra*)
            emit_diag "TETRA2001" "function main uses effect 'io' but does not declare it"
            ;;
          *tabs-diagnostic.tetra*)
            emit_diag "TETRA0001" "tabs are not supported in Flow indentation"
            ;;
          *planned-actor-diagnostic.tetra*)
            actor_msg="actor declarations currently support state fields"
            actor_msg="$actor_msg and func methods only"
            emit_diag "TETRA0001" "$actor_msg"
            ;;
          *)
            emit_diag "TETRA2001" "unknown function missing_call"
            ;;
        esac
        exit 1
      fi
    done
    ;;
  doc)
    printf '%s\n' '# Tetra API Docs' ''
    emit_tetra_api_metadata
    printf '%s\n' '## examples' '' '### Functions' ''
    printf '%b\n' '- \x60func main() -> Int\x60'
    ;;
  build)
    out=""
    prev=""
    for arg in "$@"; do
      if [[ "$prev" == "-o" ]]; then
        out="$arg"
      fi
      prev="$arg"
    done
    if [[ -n "$out" ]]; then
      mkdir -p "$(dirname "$out")"
      printf '\x00\x61\x73\x6d\x01\x00\x00\x00' >"$out"
    fi
    ;;
  targets)
    cat <<'JSON'
{
  "supported": ["linux-x64","windows-x64","macos-x64"],
  "build_only": ["wasm32-wasi","wasm32-web"],
  "planned": []
}
JSON
    ;;
  features)
    printf '{"schema":"tetra.features.v1","version":"v0.4.0","features":[]}\n'
    ;;
  doctor)
    if [[ "${TETRA_FAKE_ZERO_DOCTOR_REPORT:-}" == "1" ]]; then
      exit 0
    fi
    cat <<'JSON'
{
  "status": "pass",
  "checks": [
    {"name":"version","status":"pass"},
    {"name":"supported targets","status":"pass"},
    {"name":"build-only targets","status":"pass"},
    {"name":"planned targets","status":"pass"},
    {"name":"repo root","status":"pass"},
    {"name":"__rt/actors_sysv.tetra","status":"pass"},
    {"name":"__rt/actors_win64.tetra","status":"pass"},
    {"name":"compiler/selfhostrt/actors_sysv.tetra","status":"pass"},
    {"name":"compiler/selfhostrt/actors_win64.tetra","status":"pass"},
    {"name":"examples/flow/flow_hello.tetra","status":"pass"},
    {"name":"docs/generated/manifest.json","status":"pass"},
    {"name":"docs manifest version","status":"pass"},
    {"name":"docs manifest surface","status":"pass"},
    {"name":"smoke sources","status":"pass"},
    {"name":"runtime exports","status":"pass"}
  ]
}
JSON
    ;;
  smoke)
    report=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --report)
          report="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ -n "$report" && "${TETRA_FAKE_SMOKE_REPORT_FAIL:-}" == "1" ]]; then
      echo "smoke report failed" >&2
      exit 23
    fi
    if [[ -n "$report" ]]; then
      printf '{"target":"linux-x64","cases":[]}\n' >"$report"
    fi
    echo "Smoke linux-x64: 0/0 passed"
    ;;
  lsp)
    if [[ "${1:-}" == "--stdio" ]]; then
      cat >/dev/null
      printf '{"result":{"capabilities":{}}}\n'
      printf '{"method":"textDocument/publishDiagnostics","params":{"diagnostics":[]}}\n'
    else
      printf '{"diagnostics":[]}\n'
    fi
    ;;
  eco)
    sub="${1:-}"
    shift || true
    case "$sub" in
      verify)
        lock=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --lock)
              lock="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        if [[ -n "$lock" ]]; then
          mkdir -p "$(dirname "$lock")"
          cat >"$lock" <<'JSON'
{
  "schema": "tetra.eco.lock.v1",
  "manifest_schema": "tetra.capsule.v1",
  "permissions_model": "tetra.eco.permissions.v1",
  "graph_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "path": "Tetra.capsule",
      "targets": ["linux-x64"],
      "permissions": ["io"]
    }
  ]
}
JSON
        fi
        ;;
      seed)
        action="${1:-}"
        shift || true
        case "$action" in
          export)
            out="tetra.seed.json"
            while [[ $# -gt 0 ]]; do
              case "$1" in
                --out)
                  out="$2"
                  shift 2
                  ;;
                *)
                  shift
                  ;;
              esac
            done
            mkdir -p "$(dirname "$out")"
            printf '{}\n' >"$out"
            ;;
          import)
            lock=""
            capsules_dir=""
            while [[ $# -gt 0 ]]; do
              case "$1" in
                --lock)
                  lock="$2"
                  shift 2
                  ;;
                --capsules-dir)
                  capsules_dir="$2"
                  shift 2
                  ;;
                *)
                  shift
                  ;;
              esac
            done
            if [[ -n "$lock" ]]; then
              mkdir -p "$(dirname "$lock")"
              printf '{}\n' >"$lock"
            fi
            if [[ -n "$capsules_dir" ]]; then
              mkdir -p "$capsules_dir"
              cat >"$capsules_dir/App.capsule" <<'CAPSULE'
capsule App:
  id "tetra://app"
  version "0.1.0"
CAPSULE
            fi
            ;;
        esac
        ;;
      needmap)
        out="tetra.needmap.json"
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -o)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$(dirname "$out")"
        printf '{}\n' >"$out"
        ;;
      pack)
        out=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -o|--out)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        if [[ -n "$out" ]]; then
          mkdir -p "$(dirname "$out")"
          printf 'todex\n' >"$out"
        fi
        ;;
      unpack)
        out=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -C|--dir)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        if [[ -n "$out" ]]; then
          mkdir -p "$out/src"
          cat >"$out/Tetra.capsule" <<'CAPSULE'
capsule App:
  id "tetra://app"
  version "0.1.0"
  target "linux-x64"
CAPSULE
          printf 'func main() -> Int:\n    return 0\n' >"$out/src/main.tetra"
          cat >"$out/tetra.package.json" <<'JSON'
{
  "schema": "tetra.eco.package.v1",
  "compression": "gzip",
  "mtime_unix": 0,
  "file_count": 2,
  "files": [
    {
      "path": "Tetra.capsule",
      "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "size": 68
    },
    {
      "path": "src/main.tetra",
      "sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
      "size": 32
    }
  ]
}
JSON
        fi
        ;;
      vault)
        action="${1:-}"
        shift || true
        store=".tetra/todex-vault"
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --store)
              store="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$store/objects/sha256"
        printf '{}' >"$store/records.json"
        if [[ "$action" == "add" ]]; then
          printf '%s' "Vault added: sha256:"
          printf '%s' "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
          printf '%s\n' " source fixture"
        fi
        if [[ "$action" == "verify" ]]; then
          echo "Vault OK: 1 records"
        fi
        ;;
      trust)
        action="${1:-}"
        shift || true
        if [[ "$action" == "snapshot" ]]; then
          out="tetra.trust-snapshot.json"
          while [[ $# -gt 0 ]]; do
            case "$1" in
              -o)
                out="$2"
                shift 2
                ;;
              *)
                shift
                ;;
            esac
          done
          mkdir -p "$(dirname "$out")"
          printf '{}\n' >"$out"
        fi
        ;;
      materialize)
        out="."
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -C|--dir)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$out"
        printf '{}\n' >"$out/tetra.materialization.json"
        ;;
      publish)
        registry=".tetra/registry-beta"
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --registry)
              registry="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$registry/packages/tetra_app/0.1.0/linux-x64"
        printf 'todex\n' >"$registry/packages/tetra_app/0.1.0/linux-x64/package.todex"
        cat >"$registry/packages/tetra_app/0.1.0/linux-x64/metadata.json" <<'JSON'
{
  "schema": "tetra.eco.publish.v1beta",
  "channel": "beta",
  "hub": "local-beta",
  "published_at_unix": 0,
  "capsule": {
    "id": "tetra://app",
    "name": "App",
    "version": "0.1.0",
    "target": "linux-x64",
    "targets": ["linux-x64"],
    "permissions": ["io"]
  },
  "package": {
    "file": "package.todex",
    "size": 6,
    "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  },
  "downloads": [
    {
      "target": "linux-x64",
      "path": "packages/tetra_app/0.1.0/linux-x64/package.todex"
    }
  ]
}
JSON
        ;;
      download)
        out=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -o)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        if [[ -n "$out" ]]; then
          mkdir -p "$(dirname "$out")"
          printf 'todex\n' >"$out"
        fi
        ;;
      tetrahub)
        action="${1:-}"
        shift || true
        if [[ "$action" == "publish" ]]; then
          store=".tetra/tetrahub-beta"
          while [[ $# -gt 0 ]]; do
            case "$1" in
              --store)
                store="$2"
                shift 2
                ;;
              *)
                shift
                ;;
            esac
          done
          mkdir -p "$store/packages/tetra_app/0.1.0/linux-x64"
          printf 'todex\n' >"$store/packages/tetra_app/0.1.0/linux-x64/package.todex"
        elif [[ "$action" == "download" ]]; then
          out=""
          while [[ $# -gt 0 ]]; do
            case "$1" in
              -o)
                out="$2"
                shift 2
                ;;
              *)
                shift
                ;;
            esac
          done
          if [[ -n "$out" ]]; then
            mkdir -p "$(dirname "$out")"
            printf 'todex\n' >"$out"
          fi
        fi
        ;;
    esac
    ;;
  *)
    ;;
esac
`
	if failFmt && len(tetra) == 0 {
		t.Fatal("unreachable")
	}
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}
