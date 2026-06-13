package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTestAllScriptInterface(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	script := filepath.Join(root, "scripts", "ci", "test-all.sh")

	if out, err := exec.Command("bash", "-n", script).CombinedOutput(); err != nil {
		t.Fatalf("bash -n failed: %v\n%s", err, string(out))
	}

	help := exec.Command("bash", script, "--help")
	help.Dir = root
	helpOut, err := help.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, string(helpOut))
	}
	for _, want := range []string{"--keep-going", "--json-only", "Exit codes", "--report-dir"} {
		if !strings.Contains(string(helpOut), want) {
			t.Fatalf("help missing %q:\n%s", want, string(helpOut))
		}
	}

	bad := exec.Command("bash", script, "--definitely-not-a-real-option")
	bad.Dir = root
	badOut, err := bad.CombinedOutput()
	if err == nil {
		t.Fatalf("invalid option unexpectedly succeeded:\n%s", string(badOut))
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("invalid option exit = %v, output:\n%s", err, string(badOut))
	}
}

func TestTestAllScriptKeepGoingJSONOnly(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	scriptRaw, err := os.ReadFile(filepath.Join(root, "scripts", "ci", "test-all.sh"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts", "ci"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "scripts", "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "scripts", "release", "post_v0_4"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "ci", "test-all.sh"), scriptRaw, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "ci", "test.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "dev", "bootstrap.sh"), []byte("#!/usr/bin/env bash\ncp ./tetra ./t\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "release", "post_v0_4", "memory-100-prod-stable-gate.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nreport_dir=\"\"\nwhile [[ $# -gt 0 ]]; do case \"$1\" in --report-dir) report_dir=\"$2\"; shift 2 ;; *) shift ;; esac; done\nmkdir -p -- \"$report_dir\"\nprintf '{\"schema\":\"tetra.memory-100.prod-stable.v1\",\"status\":\"pass\"}\\n' >\"$report_dir/memory-100-prod-stable-manifest.json\"\nprintf '{\"schema\":\"tetra.artifact-hashes.v1\",\"artifacts\":[]}\\n' >\"$report_dir/artifact-hashes.json\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(`#!/usr/bin/env bash
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
    printf '{"schema_version":"tetra.memory-fuzz.oracle.v1","scope":"memory_production_core_v1_mpc15"}\n' >"$report_dir/memory-fuzz-oracle.json"
    printf '# Memory Fuzz Short Summary\n\n- tier: Tier 1 short CI smoke\n- report: memory-fuzz-oracle.json\n' >"$report_dir/summary.md"
    printf '{"schema_version":"tetra.memory-fuzz-short.summary.v1","kind":"tier1_short_ci_smoke","tier":"tier1_short_ci_smoke","status":"pass","artifacts":{"oracle_report":"memory-fuzz-oracle.json","summary_md":"summary.md","summary_json":"summary.json"},"commands":[{"name":"memory-fuzz-short","command":"go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir <artifact-dir>","status":"pass"},{"name":"validate-memory-fuzz-oracle","command":"go run ./tools/cmd/validate-memory-fuzz-oracle --report <artifact-dir>/memory-fuzz-oracle.json --artifact-dir <artifact-dir>","status":"pass"}]}\n' >"$report_dir/summary.json"
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
    printf '{"schema_version":"tetra.ram-contract-fuzz-oracle.v1","observations":[],"summary":{"mutations":0,"rejected":0},"non_claims":["not a full formal proof"]}\n' >"$report_dir/ram-contract-fuzz-oracle.json"
    printf '{"schema_version":"tetra.ram-contract-report.v1","rows":[]}\n' >"$report_dir/ram-contract-report.json"
    printf '{"schema_version":"tetra.memory-grade-report.v1"}\n' >"$report_dir/memory-grade-report.json"
    printf '{"schema_version":"tetra.proof-store-summary.v1","proofs":[],"summary":{"proof_count":0,"proven":0,"conservative":0,"rejected":0,"unknown":0},"non_claims":["no full formal proof claim"]}\n' >"$report_dir/proof-store-summary.json"
    printf '{"schema_version":"tetra.validation-pipeline-coverage.v1","entries":[]}\n' >"$report_dir/validation-pipeline-coverage.json"
    printf '{"schema_version":"tetra.ram-blockers.v1","kind":"heap","rows":[]}\n' >"$report_dir/heap-blockers.json"
    printf '{"schema_version":"tetra.ram-blockers.v1","kind":"copy","rows":[]}\n' >"$report_dir/copy-blockers.json"
  fi
  exit 0
fi
if [[ "${1:-}" == "test" ]]; then
  pkg="${2:-}"
  shift 2 || true
  list_mode=false
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -list)
        list_mode=true
        shift 2 || true
        ;;
      -list=*)
        list_mode=true
        shift
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ "$list_mode" == true ]]; then
    case "$pkg" in
      ./compiler/internal/memoryfacts)
        printf '%s\n' TestMemoryFactsRejectsUnsafeUnknownToSafeKnown TestMemoryFactsRejectsDirectSafeBorrowedFromUnsafeUnknown TestMemoryFactsRejectsDirectSafeOwnedFromUnsafeUnknown TestMemoryFactsRejectsUnsafeUnknownNoAliasAndBoundsProofClaims TestMemoryFactsRejectsUnsafeCheckedGenericPromotions TestMemoryFactsRejectsUnsafeVerifiedRootGenericClaims TestMemoryFactsRejectsValidatedUnsafeUnknownTrustedStorage TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaims TestValidateMemoryReportRejectsUnsafeCheckedGenericPromotions TestValidateMemoryReportRejectsUnsafeVerifiedRootGenericClaims TestValidateMemoryReportRejectsValidatedUnsafeUnknownTrustedStorage TestMemoryIdealV6ProjectsBoundsProofFacts TestMemoryIdealV6ProjectsMissingProofRejection TestValidateMemoryReportRejectsV6BoundsRowsWithoutParent TestValidateMemoryReportRejectsBareBoundsCheckEliminatedWithoutProofID
        ;;
      ./tools/cmd/validate-memory-report)
        printf '%s\n' TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaim TestValidateMemoryReportRejectsUnsafeCheckedGenericPromotion TestValidateMemoryReportRejectsUnsafeUnknownZeroCost TestValidateMemoryReportRejectsUnsafeVerifiedRootGenericClaim TestValidateMemoryReportRejectsUnsafeUnknownTrustedStorage TestValidateMemoryReportRejectsV6BoundsRowsWithoutParent TestValidateMemoryReportRejectsBareBoundsCheckEliminatedWithoutProofID
        ;;
      ./compiler)
        printf '%s\n' TestMemoryFuzzOracleReportCoversMPC15CategoriesAndInvariants TestClassifyMemoryFuzzOracleObservation TestValidateMemoryFuzzOracleReportRejectsDrift TestMemoryFuzzOracleReportCoversV12ReleaseEvidence TestValidateMemoryFuzzOracleReportRejectsV12ReleaseEvidenceDrift TestBuildBoundsAndProofReportsShowWhileRangeReason
        ;;
      ./compiler/internal/validation)
        printf '%s\n' TestCheckBoundsProofsRejectsRemovedCheckWithoutProofID TestCheckBoundsProofsWithPLIRRejectsUnknownLiveProof TestValidateTranslationRejectsMissingProofIDAfterTransform
        ;;
      ./compiler/internal/plir)
        printf '%s\n' TestVerifierRejectsUnknownProofUse TestVerifierRejectsNonDominatingProofUse
        ;;
      ./compiler/internal/lower)
        printf '%s\n' TestForSliceLoopUsesProofTaggedUncheckedIndexLoad TestWhileLessThanLenUsesProofTaggedUncheckedIndexLoad TestCopyLoopSourceLoadUsesProofTaggedUncheckedIndexLoad
        ;;
      ./tools/cmd/validate-memory-fuzz-oracle)
        printf '%s\n' TestValidateMemoryFuzzOracleReportFileAcceptsCompilerReport TestValidateMemoryFuzzOracleReportFileAcceptsTier1ArtifactBundle TestValidateMemoryFuzzOracleReportFileRejectsInvalidReport TestValidateMemoryFuzzOracleReportFileRejectsMissingV12ReleaseEvidence TestValidateMemoryFuzzOracleReportFileRejectsMissingArtifactSummary TestValidateMemoryFuzzOracleReportFileRejectsMissingValidatorProvenance
        ;;
      ./tools/cmd/memory-fuzz-short)
        printf '%s\n' TestRunMemoryFuzzShortWritesValidatedArtifacts TestRunMemoryFuzzShortRejectsUnsupportedTier TestRunMemoryFuzzShortRejectsStaleReportDir
        ;;
      ./tools/cmd/ram-contract-fuzz-short)
        printf '%s\n' TestRunRAMContractFuzzShortWritesValidatedArtifacts TestRunRAMContractFuzzShortRejectsStaleReportDir
        ;;
      ./tools/cmd/validate-ram-contract-fuzz-oracle)
        printf '%s\n' TestValidateRAMContractFuzzOracleAcceptsArtifactBundle TestValidateRAMContractFuzzOracleRejectsMissingReport
        ;;
      ./tools/cmd/validate-ram-contract-report)
        printf '%s\n' TestValidateRAMContractReportFileAcceptsCompilerReport TestValidateRAMContractReportRejectsMissingBlocker
        ;;
      ./compiler/internal/ramcontract)
        printf '%s\n' TestRAMContractFromAllocPlanTracksRowsAndBlockers TestRAMContractRejectsMissingBlockerExplanation TestRAMContractEnforcementFailsForHeap
        ;;
      ./cli/internal/actornet)
        printf '%s\n' TestBrokerCloseWithoutCancelStopsServeWatcher TestBrokerRoutesFramesBetweenLoopbackNodesAndWritesReport TestBrokerReportsNodeDownForMissingDestination
        ;;
    esac
    exit 0
  fi
  if [[ "$pkg" == "./compiler/..." ]]; then
    exit 1
  fi
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "rev-parse" && "${2:-}" == "HEAD" ]]; then
  echo "e2c19b8ee276158f8eb2c54cf61e11bd84952893"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tetra"), []byte(`#!/usr/bin/env bash
case "$1" in
  version) echo "v0.3.0"; exit 0 ;;
  fmt|test|smoke) exit 0 ;;
  check)
    for arg in "$@"; do
      if [[ "$arg" == "--diagnostics=json" ]]; then
        case "$*" in
          *missing-effect-diagnostic.tetra*) echo '{"code":"TETRA2001","message":"function main uses effect '\''io'\'' but does not declare it","severity":"error"}' >&2 ;;
          *tabs-diagnostic.tetra*) echo '{"code":"TETRA0001","message":"tabs are not supported in Flow indentation","severity":"error"}' >&2 ;;
          *planned-actor-diagnostic.tetra*) echo '{"code":"TETRA0001","message":"actor declarations currently support state fields and func methods only","severity":"error"}' >&2 ;;
          *) echo '{"code":"TETRA2001","message":"unknown function missing_call","severity":"error"}' >&2 ;;
        esac
        exit 1
      fi
    done
    exit 0
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
    exit 0
    ;;
  targets)
    echo '{"supported":["linux-x64","windows-x64","macos-x64"],"build_only":["wasm32-wasi","wasm32-web"],"planned":[]}'
    exit 0
    ;;
  doctor)
    echo '{"status":"pass","checks":[{"name":"version","status":"pass"},{"name":"supported targets","status":"pass"},{"name":"build-only targets","status":"pass"},{"name":"planned targets","status":"pass"},{"name":"repo root","status":"pass"},{"name":"__rt/actors_sysv.tetra","status":"pass"},{"name":"__rt/actors_win64.tetra","status":"pass"},{"name":"compiler/selfhostrt/actors_sysv.tetra","status":"pass"},{"name":"compiler/selfhostrt/actors_win64.tetra","status":"pass"},{"name":"examples/flow_hello.tetra","status":"pass"},{"name":"docs/generated/manifest.json","status":"pass"},{"name":"docs manifest version","status":"pass"},{"name":"docs manifest surface","status":"pass"},{"name":"smoke sources","status":"pass"},{"name":"runtime exports","status":"pass"},{"name":"target metadata","status":"pass"},{"name":"tooling commands","status":"pass"}]}'
    exit 0
    ;;
  *) exit 2 ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(dir, "report")
	cmd := exec.Command("bash", "scripts/ci/test-all.sh", "--quick", "--keep-going", "--json-only", "--report-dir", reportDir)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failing keep-going run, got success:\n%s", string(out))
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("exit = %v, output:\n%s", err, string(out))
	}

	var summary struct {
		Status string `json:"status"`
		Steps  []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(out, &summary); err != nil {
		t.Fatalf("summary JSON: %v\n%s", err, string(out))
	}
	if summary.Status != "fail" || len(summary.Steps) != 19 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary.Steps[0].Name != "go test all packages" || summary.Steps[0].Status != "fail" {
		t.Fatalf("first step = %#v", summary.Steps[0])
	}
	if summary.Steps[1].Name != "unsafe promotion blocker suite" || summary.Steps[1].Status != "pass" {
		t.Fatalf("unsafe promotion blocker step = %#v", summary.Steps[1])
	}
	if summary.Steps[2].Name != "bounds proof blocker suite" || summary.Steps[2].Status != "pass" {
		t.Fatalf("bounds proof blocker step = %#v", summary.Steps[2])
	}
	if summary.Steps[3].Name != "memory fuzz oracle artifact gate" || summary.Steps[3].Status != "pass" {
		t.Fatalf("memory fuzz oracle gate step = %#v", summary.Steps[3])
	}
	if summary.Steps[4].Name != "RAM contract fuzz oracle artifact gate" || summary.Steps[4].Status != "pass" {
		t.Fatalf("RAM contract fuzz oracle gate step = %#v", summary.Steps[4])
	}
	if summary.Steps[5].Name != "host leak blocker suite" || summary.Steps[5].Status != "pass" {
		t.Fatalf("host leak blocker step = %#v", summary.Steps[5])
	}
	if summary.Steps[6].Name != "Memory100 prod-stable gate" || summary.Steps[6].Status != "pass" {
		t.Fatalf("Memory100 prod-stable gate step = %#v", summary.Steps[6])
	}
	if summary.Steps[len(summary.Steps)-1].Name != "host smoke linux-x64" || summary.Steps[len(summary.Steps)-1].Status != "pass" {
		t.Fatalf("last step = %#v", summary.Steps[len(summary.Steps)-1])
	}
	if _, err := os.Stat(filepath.Join(reportDir, "summary.md")); err != nil {
		t.Fatalf("missing summary.md: %v", err)
	}
}
