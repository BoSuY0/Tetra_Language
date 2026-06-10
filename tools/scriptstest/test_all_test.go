package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTestAllQuickJSONIncludesStepExitCodes(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" || len(summary.Steps) == 0 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary.StepCount != len(summary.Steps) || summary.FailedCount != 0 {
		t.Fatalf("summary counts = steps:%d failed:%d len:%d", summary.StepCount, summary.FailedCount, len(summary.Steps))
	}
	if summary.ReleaseVersion != "v0.4.0" {
		t.Fatalf("release_version = %q", summary.ReleaseVersion)
	}
	if summary.ReleaseArtifact != "tetra.release.v0_4_0.test-all-summary.v1" {
		t.Fatalf("release_artifact = %q", summary.ReleaseArtifact)
	}
	for _, step := range summary.Steps {
		if step.Status != "pass" || step.ExitCode == nil || *step.ExitCode != 0 {
			t.Fatalf("step missing pass exit code: %#v", step)
		}
		if step.Command == "" {
			t.Fatalf("step missing command: %#v", step)
		}
	}
}

func TestTestAllRunsUnsafePromotionBlockerSuite(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test-all script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "unsafe promotion blocker suite" check_unsafe_promotion_blockers`,
		`require_go_test_names ./compiler/internal/memoryfacts 'UnsafeUnknown|UnsafeVerified|Promotion'`,
		`TestMemoryFactsRejectsUnsafeUnknownToSafeKnown`,
		`TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaim`,
		`TestMemoryFuzzOracleReportCoversMPC15CategoriesAndInvariants`,
		`TestValidateMemoryFuzzOracleReportFileRejectsInvalidReport`,
		`go test ./compiler/internal/memoryfacts -run 'UnsafeUnknown|UnsafeVerified|Promotion' -count=1`,
		`go test ./compiler -run 'Unsafe|Raw|MemoryFuzzOracle' -count=1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("scripts/ci/test-all.sh missing unsafe promotion blocker requirement %q", want)
		}
	}
}

func TestTestAllQuickReportsUnsafePromotionBlockerSuite(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if !hasTestAllStep(summary, "unsafe promotion blocker suite") {
		t.Fatalf("summary missing passing unsafe promotion blocker suite step: %#v", summary.Steps)
	}
}

func TestTestAllQuickFailsWhenUnsafePromotionBlockerSuiteMissing(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST=1"}, "--quick", "--keep-going", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected unsafe promotion blocker suite failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary status/counts = %q/%d, want fail/1: %#v", summary.Status, summary.FailedCount, summary.Steps)
	}
	var sawUnsafePromotionFailure bool
	var unsafePromotionLog string
	for _, step := range summary.Steps {
		if step.Name == "unsafe promotion blocker suite" {
			sawUnsafePromotionFailure = true
			if step.Status != "fail" || step.ExitCode == nil || *step.ExitCode == 0 {
				t.Fatalf("unsafe promotion blocker step = %#v", step)
			}
			if !strings.Contains(step.Log, "unsafe-promotion-blocker-suite.log") {
				t.Fatalf("unsafe promotion blocker log path = %q", step.Log)
			}
			unsafePromotionLog = step.Log
		}
	}
	if !sawUnsafePromotionFailure {
		t.Fatalf("summary missing failing unsafe promotion blocker suite step: %#v", summary.Steps)
	}
	logPath := filepath.Join(reportDir, unsafePromotionLog)
	logRaw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read unsafe promotion blocker log: %v", err)
	}
	if !strings.Contains(string(logRaw), "missing required unsafe promotion blocker test") {
		t.Fatalf("unsafe promotion blocker log missing required-test failure:\n%s", logRaw)
	}
}

func TestTestAllRunsBoundsProofBlockerSuite(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test-all script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "bounds proof blocker suite" check_bounds_proof_blockers`,
		`require_bounds_go_test_names ./compiler/internal/validation 'Bounds|Proof|Unchecked'`,
		`TestCheckBoundsProofsRejectsRemovedCheckWithoutProofID`,
		`TestCheckBoundsProofsWithPLIRRejectsUnknownLiveProof`,
		`TestValidateMemoryReportRejectsBareBoundsCheckEliminatedWithoutProofID`,
		`TestMemoryFuzzOracleReportCoversV12ReleaseEvidence`,
		`go test ./compiler/internal/plir ./compiler/internal/lower ./compiler/internal/validation -run 'Bounds|Proof|Unchecked' -count=1`,
		`go test ./compiler -run 'Bounds|MemoryFuzzOracle' -count=1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("scripts/ci/test-all.sh missing bounds proof blocker requirement %q", want)
		}
	}
}

func TestTestAllQuickReportsBoundsProofBlockerSuite(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if !hasTestAllStep(summary, "bounds proof blocker suite") {
		t.Fatalf("summary missing passing bounds proof blocker suite step: %#v", summary.Steps)
	}
}

func TestTestAllQuickFailsWhenBoundsProofBlockerSuiteMissing(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST=1"}, "--quick", "--keep-going", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected bounds proof blocker suite failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary status/counts = %q/%d, want fail/1: %#v", summary.Status, summary.FailedCount, summary.Steps)
	}
	var sawBoundsProofFailure bool
	var boundsProofLog string
	for _, step := range summary.Steps {
		if step.Name == "bounds proof blocker suite" {
			sawBoundsProofFailure = true
			if step.Status != "fail" || step.ExitCode == nil || *step.ExitCode == 0 {
				t.Fatalf("bounds proof blocker step = %#v", step)
			}
			if !strings.Contains(step.Log, "bounds-proof-blocker-suite.log") {
				t.Fatalf("bounds proof blocker log path = %q", step.Log)
			}
			boundsProofLog = step.Log
		}
	}
	if !sawBoundsProofFailure {
		t.Fatalf("summary missing failing bounds proof blocker suite step: %#v", summary.Steps)
	}
	logRaw, err := os.ReadFile(filepath.Join(reportDir, boundsProofLog))
	if err != nil {
		t.Fatalf("read bounds proof blocker log: %v", err)
	}
	if !strings.Contains(string(logRaw), "missing required bounds proof blocker test") {
		t.Fatalf("bounds proof blocker log missing required-test failure:\n%s", logRaw)
	}
}

func TestTestAllRunsMemoryFuzzOracleGate(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test-all script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "memory fuzz oracle artifact gate" check_memory_fuzz_oracle_gate`,
		`local fuzz_dir="$report_dir/memory-fuzz-tier1"`,
		`go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir "$fuzz_dir"`,
		`go run ./tools/cmd/validate-memory-fuzz-oracle --report "$fuzz_dir/memory-fuzz-oracle.json" --artifact-dir "$fuzz_dir"`,
		`require_memory_fuzz_go_test_names ./tools/cmd/memory-fuzz-short 'MemoryFuzzShort|Tier|ReportDir'`,
		`TestRunMemoryFuzzShortWritesValidatedArtifacts`,
		`TestRunMemoryFuzzShortRejectsStaleReportDir`,
		`TestValidateMemoryFuzzOracleReportFileAcceptsTier1ArtifactBundle`,
		`TestValidateMemoryFuzzOracleReportFileRejectsMissingArtifactSummary`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("scripts/ci/test-all.sh missing memory fuzz oracle gate requirement %q", want)
		}
	}
}

func TestTestAllQuickReportsMemoryFuzzOracleGate(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if !hasTestAllStep(summary, "memory fuzz oracle artifact gate") {
		t.Fatalf("summary missing passing memory fuzz oracle artifact gate step: %#v", summary.Steps)
	}
}

func TestTestAllRunsRAMContractFuzzOracleGate(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test-all script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "RAM contract fuzz oracle artifact gate" check_ram_contract_fuzz_oracle_gate`,
		`local fuzz_dir="$report_dir/ram-contract-fuzz"`,
		`go run ./tools/cmd/ram-contract-fuzz-short --report-dir "$fuzz_dir"`,
		`go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report "$fuzz_dir/ram-contract-fuzz-oracle.json" --artifact-dir "$fuzz_dir"`,
		`go run ./tools/cmd/validate-ram-contract-report --report "$fuzz_dir/ram-contract-report.json"`,
		`TestRunRAMContractFuzzShortWritesValidatedArtifacts`,
		`TestValidateRAMContractFuzzOracleAcceptsArtifactBundle`,
		`TestRAMContractEnforcementFailsForHeap`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("scripts/ci/test-all.sh missing RAM contract fuzz oracle gate requirement %q", want)
		}
	}
}

func TestTestAllQuickReportsRAMContractFuzzOracleGate(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if !hasTestAllStep(summary, "RAM contract fuzz oracle artifact gate") {
		t.Fatalf("summary missing passing RAM contract fuzz oracle artifact gate step: %#v", summary.Steps)
	}
}

func TestTestAllQuickFailsWhenRAMContractFuzzOracleGateTestsMissing(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_SKIP_RAM_CONTRACT_LIST=1"}, "--quick", "--keep-going", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected RAM contract fuzz oracle gate failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary status/counts = %q/%d, want fail/1: %#v", summary.Status, summary.FailedCount, summary.Steps)
	}
	var fuzzLog string
	for _, step := range summary.Steps {
		if step.Name == "RAM contract fuzz oracle artifact gate" {
			if step.Status != "fail" || step.ExitCode == nil || *step.ExitCode == 0 {
				t.Fatalf("RAM contract fuzz oracle gate step = %#v", step)
			}
			fuzzLog = step.Log
		}
	}
	if fuzzLog == "" {
		t.Fatalf("summary missing failing RAM contract fuzz oracle artifact gate step: %#v", summary.Steps)
	}
	logRaw, err := os.ReadFile(filepath.Join(reportDir, fuzzLog))
	if err != nil {
		t.Fatalf("read RAM contract fuzz oracle gate log: %v", err)
	}
	if !strings.Contains(string(logRaw), "missing required RAM contract fuzz oracle gate test") {
		t.Fatalf("RAM contract fuzz oracle gate log missing required-test failure:\n%s", logRaw)
	}
}

func TestTestAllQuickFailsWhenMemoryFuzzOracleGateTestsMissing(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST=1"}, "--quick", "--keep-going", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected memory fuzz oracle gate failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary status/counts = %q/%d, want fail/1: %#v", summary.Status, summary.FailedCount, summary.Steps)
	}
	var fuzzLog string
	for _, step := range summary.Steps {
		if step.Name == "memory fuzz oracle artifact gate" {
			if step.Status != "fail" || step.ExitCode == nil || *step.ExitCode == 0 {
				t.Fatalf("memory fuzz oracle gate step = %#v", step)
			}
			fuzzLog = step.Log
		}
	}
	if fuzzLog == "" {
		t.Fatalf("summary missing failing memory fuzz oracle artifact gate step: %#v", summary.Steps)
	}
	logRaw, err := os.ReadFile(filepath.Join(reportDir, fuzzLog))
	if err != nil {
		t.Fatalf("read memory fuzz oracle gate log: %v", err)
	}
	if !strings.Contains(string(logRaw), "missing required memory fuzz oracle gate test") {
		t.Fatalf("memory fuzz oracle gate log missing required-test failure:\n%s", logRaw)
	}
}

func TestTestAllRunsHostLeakBlockerSuite(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test-all script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "host leak blocker suite" check_host_leak_blockers`,
		`require_host_leak_go_test_names ./cli/internal/actornet 'Broker|Leak|CloseWithoutCancel'`,
		`TestBrokerCloseWithoutCancelStopsServeWatcher`,
		`go test ./cli/internal/actornet -run 'Broker|Leak|CloseWithoutCancel' -count=1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("scripts/ci/test-all.sh missing host leak blocker requirement %q", want)
		}
	}
}

func TestTestAllQuickReportsHostLeakBlockerSuite(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if !hasTestAllStep(summary, "host leak blocker suite") {
		t.Fatalf("summary missing passing host leak blocker suite step: %#v", summary.Steps)
	}
}

func TestTestAllQuickFailsWhenHostLeakBlockerSuiteMissing(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_SKIP_HOST_LEAK_LIST=1"}, "--quick", "--keep-going", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected host leak blocker suite failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary status/counts = %q/%d, want fail/1: %#v", summary.Status, summary.FailedCount, summary.Steps)
	}
	var sawHostLeakFailure bool
	var hostLeakLog string
	for _, step := range summary.Steps {
		if step.Name == "host leak blocker suite" {
			sawHostLeakFailure = true
			if step.Status != "fail" || step.ExitCode == nil || *step.ExitCode == 0 {
				t.Fatalf("host leak blocker step = %#v", step)
			}
			if !strings.Contains(step.Log, "host-leak-blocker-suite.log") {
				t.Fatalf("host leak blocker log path = %q", step.Log)
			}
			hostLeakLog = step.Log
		}
	}
	if !sawHostLeakFailure {
		t.Fatalf("summary missing failing host leak blocker suite step: %#v", summary.Steps)
	}
	logRaw, err := os.ReadFile(filepath.Join(reportDir, hostLeakLog))
	if err != nil {
		t.Fatalf("read host leak blocker log: %v", err)
	}
	if !strings.Contains(string(logRaw), "missing required host leak blocker test") {
		t.Fatalf("host leak blocker log missing required-test failure:\n%s", logRaw)
	}
}

func TestTestAllWorkflowLivesInCIEntryPoint(t *testing.T) {
	root := repoRoot(t)
	ciPath := filepath.Join(root, "scripts", "ci", "test-all.sh")
	assertLegacyFileRemoved(t, "scripts/test_all.sh", "scripts/ci/test-all.sh")
	ciRaw, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read ci test-all: %v", err)
	}
	ciText := string(ciRaw)
	assertNoLegacyMention(t, ciText, "scripts/test_all.sh", "scripts/ci/test-all.sh help")
	cmd := exec.Command("bash", ciPath, "--help")
	cmd.Dir = root
	helpOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scripts/ci/test-all.sh --help failed: %v\n%s", err, helpOut)
	}
	helpText := string(helpOut)
	assertNoLegacyMention(t, helpText, "scripts/test_all.sh", "scripts/ci/test-all.sh --help")
	for _, want := range []string{
		"Usage: bash scripts/ci/test-all.sh",
		"--quick",
		"--full",
		"--stabilization",
		"release_artifact defaults",
	} {
		if !strings.Contains(helpText, want) {
			t.Fatalf("scripts/ci/test-all.sh --help missing %q", want)
		}
	}
	reportParent := t.TempDir()
	reportDir := filepath.Join(reportParent, "test-all-help-report")
	cmd = exec.Command("bash", ciPath, "--report-dir", reportDir, "--help")
	cmd.Dir = root
	helpOut, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scripts/ci/test-all.sh --report-dir DIR --help failed: %v\n%s", err, helpOut)
	}
	assertNoLegacyMention(t, string(helpOut), "scripts/test_all.sh", "scripts/ci/test-all.sh --report-dir DIR --help")
	if _, err := os.Stat(reportDir); !os.IsNotExist(err) {
		t.Fatalf("scripts/ci/test-all.sh --help must not create report dir %s: %v", reportDir, err)
	}
	for _, want := range []string{
		"Usage: bash scripts/ci/test-all.sh",
		"run_step \"bootstrap\"",
		"write_summary()",
		"validate-test-all-summary",
	} {
		if !strings.Contains(ciText, want) {
			t.Fatalf("scripts/ci/test-all.sh missing %q", want)
		}
	}
}

func TestTestAllReleaseArtifactFollowsReleaseVersion(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{
		"TETRA_TEST_ALL_RELEASE_VERSION=v0.3.0",
		"TETRA_FAKE_TETRA_VERSION=v0.3.0",
	}, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.ReleaseVersion != "v0.3.0" {
		t.Fatalf("release_version = %q", summary.ReleaseVersion)
	}
	if summary.ReleaseArtifact != "tetra.release.v0_3_0.test-all-summary.v1" {
		t.Fatalf("release_artifact = %q", summary.ReleaseArtifact)
	}
}

func TestTestAllReleaseArtifactCanBeOverridden(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{
		"TETRA_TEST_ALL_RELEASE_VERSION=v0.3.0",
		"TETRA_TEST_ALL_RELEASE_ARTIFACT=tetra.release.custom.test-all-summary.v1",
		"TETRA_FAKE_TETRA_VERSION=v0.3.0",
	}, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.ReleaseArtifact != "tetra.release.custom.test-all-summary.v1" {
		t.Fatalf("release_artifact = %q", summary.ReleaseArtifact)
	}
}

func TestTestAllRejectsMalformedReportDirFlag(t *testing.T) {
	root := testAllFakeRepo(t, false)
	out, err := runTestAll(t, root, nil, "--report-dir")
	if err == nil {
		t.Fatalf("expected malformed --report-dir failure\n%s", out)
	}
	if !strings.Contains(string(out), "--report-dir requires a directory") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestTestAllRejectsExistingReportArtifacts(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	if !strings.Contains(string(out), "refusing to reuse non-empty report directory") {
		t.Fatalf("unexpected stale report-dir output:\n%s", out)
	}
}

func assertTestAllRejectsNonDirectoryReportPath(t *testing.T, root, reportPath, label string) {
	t.Helper()
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportPath)
	if err == nil {
		t.Fatalf("expected %s report path rejection\n%s", label, out)
	}
	assertExitCode(t, err, 2, string(out))
	output := string(out)
	if !strings.Contains(output, "refusing to use non-directory report path: "+reportPath) {
		t.Fatalf("unexpected %s report path output:\n%s", label, out)
	}
	if !strings.Contains(output, "choose a fresh --report-dir directory") {
		t.Fatalf("%s report path output missing remediation:\n%s", label, out)
	}
	if strings.Contains(output, "mkdir:") {
		t.Fatalf("%s report path should fail before mkdir:\n%s", label, out)
	}
}

func TestTestAllRejectsExistingReportPathFile(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportPath := filepath.Join(root, "report-file")
	if err := os.WriteFile(reportPath, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertTestAllRejectsNonDirectoryReportPath(t, root, reportPath, "regular-file")
}

func TestTestAllRejectsDanglingReportDirSymlink(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportPath := filepath.Join(root, "report-link")
	if err := os.Symlink(filepath.Join(root, "missing-report-target"), reportPath); err != nil {
		t.Fatalf("create dangling report-dir symlink: %v", err)
	}
	assertTestAllRejectsSymlinkReportPath(t, root, reportPath, "dangling symlink")
}

func TestTestAllRejectsReportDirSymlinkToFile(t *testing.T) {
	root := testAllFakeRepo(t, false)
	targetPath := filepath.Join(root, "report-file-target")
	if err := os.WriteFile(targetPath, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "report-link")
	if err := os.Symlink(targetPath, reportPath); err != nil {
		t.Fatalf("create report-dir symlink to file: %v", err)
	}
	assertTestAllRejectsSymlinkReportPath(t, root, reportPath, "file symlink")
}

func assertTestAllRejectsSymlinkReportPath(t *testing.T, root, reportPath, label string) {
	t.Helper()
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportPath)
	if err == nil {
		t.Fatalf("expected %s report path rejection\n%s", label, out)
	}
	assertExitCode(t, err, 2, string(out))
	output := string(out)
	if !strings.Contains(output, "refusing to use symlink report directory: "+reportPath) {
		t.Fatalf("unexpected %s report path output:\n%s", label, out)
	}
	if !strings.Contains(output, "choose a real fresh --report-dir") {
		t.Fatalf("%s report path output missing remediation:\n%s", label, out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func TestTestAllRejectsSymlinkToExistingReportArtifacts(t *testing.T) {
	root := testAllFakeRepo(t, false)
	targetDir := filepath.Join(root, "stale-target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(root, "report-link")
	if err := os.Symlink(targetDir, reportDir); err != nil {
		t.Fatalf("create report-dir symlink: %v", err)
	}
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_GO_LOG=" + goLog}, "--quick", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected stale symlinked report-dir rejection\n%s", out)
	}
	assertExitCode(t, err, 2, string(out))
	if !strings.Contains(string(out), "refusing to reuse non-empty report directory") {
		t.Fatalf("unexpected stale symlinked report-dir output:\n%s", out)
	}
	if _, err := os.Stat(goLog); err == nil {
		t.Fatalf("fake go ran for stale symlinked report-dir\noutput:\n%s", out)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat fake go marker: %v", err)
	}
}

func TestTestAllAllowsExistingEmptyReportDir(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick with empty report-dir failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" {
		t.Fatalf("summary status = %q", summary.Status)
	}
}

func TestTestAllAllowsDashPrefixedFreshReportDir(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := "-fresh-report"
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_GO_LOG=" + goLog}, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick with dash-prefixed fresh report-dir failed: %v\n%s", err, out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" {
		t.Fatalf("summary status = %q", summary.Status)
	}
	if _, err := os.Stat(filepath.Join(root, reportDir, "summary.json")); err != nil {
		t.Fatalf("expected summary.json in dash-prefixed report-dir: %v", err)
	}
	if _, err := os.Stat(goLog); err != nil {
		t.Fatalf("expected fake go to run for fresh report-dir: %v", err)
	}
}

func TestTestAllRejectsSymlinkToEmptyReportDirBeforeExecution(t *testing.T) {
	root := testAllFakeRepo(t, false)
	targetDir := filepath.Join(root, "empty-target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(root, "report-link")
	if err := os.Symlink(targetDir, reportDir); err != nil {
		t.Fatalf("create report-dir symlink: %v", err)
	}
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_GO_LOG=" + goLog}, "--quick", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected symlink report-dir rejection\n%s", out)
	}
	assertExitCode(t, err, 2, string(out))
	output := string(out)
	if !strings.Contains(output, "refusing to use symlink report directory: "+reportDir) {
		t.Fatalf("unexpected symlink report-dir output:\n%s", out)
	}
	if !strings.Contains(output, "choose a real fresh --report-dir") {
		t.Fatalf("symlink report-dir output missing remediation:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(goLog); err == nil {
		t.Fatalf("fake go ran for symlink report-dir\noutput:\n%s", out)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat fake go marker: %v", err)
	}
}

func TestTestAllRejectsDashPrefixedExistingReportArtifacts(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := "-stale-report"
	if err := os.MkdirAll(filepath.Join(root, reportDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, reportDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected dash-prefixed stale report-dir rejection\n%s", out)
	}
	assertExitCode(t, err, 2, string(out))
	if !strings.Contains(string(out), "refusing to reuse non-empty report directory: "+reportDir) {
		t.Fatalf("unexpected dash-prefixed stale report-dir output:\n%s", out)
	}
	if strings.Contains(string(out), "find:") {
		t.Fatalf("dash-prefixed report-dir should not be parsed as a find option:\n%s", out)
	}
}

func TestTestAllRejectsDashPrefixedSymlinkToExistingReportArtifacts(t *testing.T) {
	root := testAllFakeRepo(t, false)
	targetDir := filepath.Join(root, "stale-target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reportDir := "-stale-report-link"
	if err := os.Symlink(targetDir, filepath.Join(root, reportDir)); err != nil {
		t.Fatalf("create dash-prefixed report-dir symlink: %v", err)
	}
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected dash-prefixed stale symlink report-dir rejection\n%s", out)
	}
	assertExitCode(t, err, 2, string(out))
	if !strings.Contains(string(out), "refusing to reuse non-empty report directory: "+reportDir) {
		t.Fatalf("unexpected dash-prefixed stale symlink report-dir output:\n%s", out)
	}
	if strings.Contains(string(out), "find:") {
		t.Fatalf("dash-prefixed symlink report-dir should not be parsed as a find option:\n%s", out)
	}
}

func TestTestAllRunsFromNestedWorkingDirectory(t *testing.T) {
	root := testAllFakeRepo(t, false)
	nested := filepath.Join(root, "sub", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := runTestAllFromWorkingDir(t, root, nested, nil, "--quick", "--json-only", "--report-dir", filepath.Join(root, "report"))
	if err != nil {
		t.Fatalf("test_all run from nested dir failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" {
		t.Fatalf("summary status = %q", summary.Status)
	}
}

func TestTestAllKeepGoingJSONRecordsFailingExitCode(t *testing.T) {
	root := testAllFakeRepo(t, true)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{"TETRA_FAIL_FMT=1"}, "--quick", "--keep-going", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected failing test_all run\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" {
		t.Fatalf("status = %q, want fail", summary.Status)
	}
	if summary.StepCount != len(summary.Steps) || summary.FailedCount != 1 {
		t.Fatalf("summary counts = steps:%d failed:%d len:%d", summary.StepCount, summary.FailedCount, len(summary.Steps))
	}
	var sawFmt bool
	var sawLaterStep bool
	for _, step := range summary.Steps {
		if step.Name == testAllFormatterStepName {
			sawFmt = true
			if step.Status != "fail" || step.ExitCode == nil || *step.ExitCode != 7 {
				t.Fatalf("formatter step = %#v", step)
			}
		}
		if step.Name == "host smoke linux-x64" && step.Status == "pass" && step.ExitCode != nil && *step.ExitCode == 0 {
			sawLaterStep = true
		}
	}
	if !sawFmt || !sawLaterStep {
		t.Fatalf("summary did not record failing and later steps: %#v", summary.Steps)
	}
}

func TestTestAllFailurePathPreservesSummaryWhenSummaryValidatorFails(t *testing.T) {
	root := testAllFakeRepo(t, true)
	reportDir := filepath.Join(root, "report")
	stdout, stderr, err := runTestAllSplit(t, root, []string{
		"TETRA_FAIL_FMT=1",
		"TETRA_FAIL_SUMMARY_VALIDATOR=1",
	}, "--quick", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected failing test_all run\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(string(stderr), "warning: summary validation failed; preserving original test failure") {
		t.Fatalf("stderr missing best-effort validator warning:\n%s", stderr)
	}
	summary := decodeTestAllSummary(t, stdout)
	if summary.Status != "fail" {
		t.Fatalf("status = %q, want fail", summary.Status)
	}
	if summary.StepCount != len(summary.Steps) || summary.FailedCount != 1 {
		t.Fatalf("summary counts = steps:%d failed:%d len:%d", summary.StepCount, summary.FailedCount, len(summary.Steps))
	}
	if got := len(summary.Steps); got == 0 {
		t.Fatalf("summary has no steps")
	}
	step := summary.Steps[len(summary.Steps)-1]
	if step.Name != testAllFormatterStepName || step.Status != "fail" || step.ExitCode == nil || *step.ExitCode != 7 {
		t.Fatalf("last failing step = %#v", step)
	}
}

func TestReleaseV06GateValidatesCrossTargetSmokeReports(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, report := range []string{"linux-smoke.json", "macos-smoke.json", "windows-smoke.json"} {
		want := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/` + report + `"`
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing smoke report validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesHostSmokeReport(t *testing.T) {
	assertLegacyFileRemoved(t, "scripts/release_v0_6_gate.sh", "scripts/release/v0_6/gate.sh directly")
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	run := `./tetra smoke --target linux-x64 --run=true --report "$tmp_dir/host-smoke.json"`
	if !strings.Contains(text, run) {
		t.Fatalf("release gate missing host smoke report command %q", run)
	}
	validate := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/host-smoke.json"`
	if !strings.Contains(text, validate) {
		t.Fatalf("release gate missing host smoke report validation %q", validate)
	}
}

func TestReleaseV05GateValidatesJSONReports(t *testing.T) {
	root := repoRoot(t)
	assertLegacyFileRemoved(t, "scripts/release_v0_5_gate.sh", "scripts/release/v0_5/gate.sh directly")
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "v0_5", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.5 release gate: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, `repo_root="$(cd "$script_dir/../../.." && pwd)"`) {
		t.Fatalf("v0.5 release gate should resolve the repo root from its versioned script path")
	}
	generatedManifest := `go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json"`
	if !strings.Contains(text, generatedManifest) {
		t.Fatalf("v0.5 release gate missing generated manifest validation %q", generatedManifest)
	}
	canonicalManifest := `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
	if !strings.Contains(text, canonicalManifest) {
		t.Fatalf("v0.5 release gate missing canonical manifest validation %q", canonicalManifest)
	}
	testReport := `go run ./tools/cmd/validate-test-report --report "$tmp_dir/tetra-test-report.json"`
	if !strings.Contains(text, testReport) {
		t.Fatalf("v0.5 release gate missing test report validation %q", testReport)
	}
	lspReport := `go run ./tools/cmd/validate-lsp-smoke --report "$tmp_dir/lsp.json"`
	if !strings.Contains(text, lspReport) {
		t.Fatalf("v0.5 release gate missing LSP smoke validation %q", lspReport)
	}
	apiDocs := `go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/api-docs.md"`
	if !strings.Contains(text, apiDocs) {
		t.Fatalf("v0.5 release gate missing API docs validation %q", apiDocs)
	}
	ecoLock := `go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"`
	if !strings.Contains(text, ecoLock) {
		t.Fatalf("v0.5 release gate missing Eco lock validation %q", ecoLock)
	}
	ecoVault := `go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"`
	if !strings.Contains(text, ecoVault) {
		t.Fatalf("v0.5 release gate missing Eco vault validation %q", ecoVault)
	}
	hostSmoke := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/host-smoke.json"`
	if !strings.Contains(text, hostSmoke) {
		t.Fatalf("v0.5 release gate missing host smoke validation %q", hostSmoke)
	}
	for _, report := range []string{"linux-smoke.json", "macos-smoke.json", "windows-smoke.json"} {
		want := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/` + report + `"`
		if !strings.Contains(text, want) {
			t.Fatalf("v0.5 release gate missing smoke report validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesTestRunnerJSONReport(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-test-report --report "$tmp_dir/tetra-test-report.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing test report validation %q", want)
	}
}

func TestReleaseV06GateValidatesDocsManifests(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json"`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing manifest validation %q", want)
		}
	}
}

func TestReleaseV06GateChecksShortAliasVersion(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`short_version="$(./t version)"`,
		`expected ./t version to match ./tetra version`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing short alias version check %q", want)
		}
	}
}

func TestReleaseV06GateValidatesLSPSmokeJSONReport(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-lsp-smoke --report "$tmp_dir/lsp-smoke.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing LSP smoke validation %q", want)
	}
}

func TestReleaseV06GateValidatesLSPStdioTranscript(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-lsp-stdio --transcript "$tmp_dir/lsp-stdio.out"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing LSP stdio validation %q", want)
	}
}

func TestReleaseV06GateValidatesGeneratedAPIDocs(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/api-docs.md"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing API docs validation %q", want)
	}
}

func TestReleaseV06GateFormatterCoversRuntimeSources(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `./tetra fmt --check examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing runtime formatter coverage %q", want)
	}
}

func TestReleaseV06GateScansFlowOnlySources(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Flow-only scan %q", want)
	}
}

func TestReleaseV06GateValidatesTargetsJSONReport(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra targets --format=json >"$tmp_dir/targets.json"`,
		`go run ./tools/cmd/validate-targets --report "$tmp_dir/targets.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing targets validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesDoctorJSONReport(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra doctor --format=json >"$tmp_dir/doctor.json"`,
		`go run ./tools/cmd/validate-doctor --report "$tmp_dir/doctor.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing doctor validation %q", want)
		}
	}
}

func TestReleaseV06GateRunsCheckAndDocCommands(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra check examples/flow_hello.tetra`,
		`./tetra doc examples >"$tmp_dir/tetra-docs.md"`,
		`go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/tetra-docs.md"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing CLI command coverage %q", want)
		}
	}
}

func TestReleaseV06GateValidatesJSONDiagnosticShape(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`check_json_diagnostic_case "invalid-diagnostic" "unknown function"`,
		`check_json_diagnostic_case "missing-effect-diagnostic" "uses effect 'io'"`,
		`check_json_diagnostic_case "tabs-diagnostic" "tabs are not supported"`,
		`check_json_diagnostic_case "planned-actor-diagnostic" "planned feature 'actor'"`,
		`--require-position`,
		`./tetra build --target wasm32-wasi -o "$wasm_out" examples/hello.tetra`,
		`test "$(od -An -tx1 -N4 "$wasm_out" | tr -d ' \n')" = "0061736d"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing JSON diagnostic validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesSmokeListJSONReport(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra smoke --list --format=json >"$tmp_dir/smoke-list.json"`,
		`go run ./tools/cmd/validate-smoke-list --report "$tmp_dir/smoke-list.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing smoke list validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesEcoLock(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Eco lock validation %q", want)
	}
}

func TestReleaseV06GateValidatesEcoUnpack(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Eco unpack validation %q", want)
	}
}

func TestReleaseV06GateValidatesEcoVault(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Eco vault validation %q", want)
	}
}

func TestTestAllValidatesTestRunnerJSONReport(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-test-report --report "$report_dir/tetra-test-report.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing test report validation %q", want)
	}
}

func TestTestAllValidatesSummaryArtifacts(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-test-all-summary --summary "$summary_json" --report-dir "$report_dir"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing summary validation %q", want)
	}
}

func TestTestAllTopLevelGoTestBypassesCache(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `run_step "go test all packages" env -u TETRA_TEST_ALL_RELEASE_VERSION -u TETRA_TEST_ALL_RELEASE_ARTIFACT -u TETRA_SECURITY_REVIEW_SIGNOFF go test ./compiler/... ./cli/... ./tools/... -count=1`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all top-level go test should bypass cache with %q", want)
	}
	wantRepo := `run_step "repo test script" env -u TETRA_TEST_ALL_RELEASE_VERSION -u TETRA_TEST_ALL_RELEASE_ARTIFACT -u TETRA_SECURITY_REVIEW_SIGNOFF bash scripts/ci/test.sh`
	if !strings.Contains(string(raw), wantRepo) {
		t.Fatalf("test_all repo script should clear release signoff env with %q", wantRepo)
	}
}

func TestTestAllChecksShortAliasVersion(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "short alias version" check_short_alias_version`,
		`short_version="$(./t version)"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing short alias version check %q", want)
		}
	}
}

func TestTestAllValidatesHostSmokeReport(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	run := `run_tetra_smoke_target "linux-x64" "true" "$report_dir/host-smoke.json"`
	if !strings.Contains(text, run) {
		t.Fatalf("test_all missing host smoke report command %q", run)
	}
	validate := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report_path"`
	if !strings.Contains(text, validate) {
		t.Fatalf("test_all missing host smoke report validation %q", validate)
	}
	if strings.Contains(text, `release_smoke_cases=`) {
		t.Fatalf("test_all should not keep a local release smoke case array")
	}
}

func TestTestAllFormatterCoversRuntimeSources(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `run_step "formatter check examples lib runtime" ./tetra fmt --check examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing runtime formatter coverage %q", want)
	}
}

func TestTestAllScansFlowOnlySources(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `run_step "flow-only source scan" go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Flow-only scan %q", want)
	}
}

func TestTestAllValidatesTargetsJSONReport(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "targets json report" check_targets_report`,
		`./tetra targets --format=json >"$report_dir/targets.json"`,
		`go run ./tools/cmd/validate-targets --report "$report_dir/targets.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing targets validation %q", want)
		}
	}
}

func TestTestAllValidatesDoctorJSONReport(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "doctor json report" check_doctor_report`,
		`./tetra doctor --format=json >"$report_dir/doctor.json"`,
		`go run ./tools/cmd/validate-doctor --report "$report_dir/doctor.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing doctor validation %q", want)
		}
	}
}

func TestTestAllRunsCheckAndDocCommands(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "tetra check flow hello" ./tetra check examples/flow_hello.tetra`,
		`run_step "tetra doc examples" check_tetra_doc`,
		`./tetra doc examples >"$report_dir/tetra-docs.md"`,
		`go run ./tools/cmd/validate-api-docs --docs "$report_dir/tetra-docs.md"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing CLI command coverage %q", want)
		}
	}
}

func TestTestAllFullRunsSafetyReadinessGate(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "safety readiness evidence" check_safety_readiness`,
		`go run ./tools/cmd/validate-safety-readiness`,
		`--out "$report_dir/safety-readiness.json"`,
		`go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Actor|Task|Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem' -count=1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing safety readiness coverage %q", want)
		}
	}
}

func TestTestAllFullFailsOnSafetyReadinessValidatorFailure(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{"TETRA_FAIL_SAFETY_READINESS=1"}, "--full", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected full gate to fail on safety readiness validator failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	var sawSafetyFailure bool
	for _, step := range summary.Steps {
		if step.Name == "safety readiness evidence" && step.Status == "fail" {
			sawSafetyFailure = true
		}
		if step.Name == "tooling summary aggregation" {
			t.Fatalf("tooling summary should not run after safety readiness failure without keep-going: %#v", summary.Steps)
		}
	}
	if !sawSafetyFailure {
		t.Fatalf("summary missing failing safety readiness step: %#v", summary.Steps)
	}
	logRaw, err := os.ReadFile(filepath.Join(reportDir, testAllStepLog(t, summary, "safety readiness evidence")))
	if err != nil {
		t.Fatalf("read safety readiness log: %v", err)
	}
	if !strings.Contains(string(logRaw), "safety readiness failed") {
		t.Fatalf("safety readiness log missing validator failure detail:\n%s", logRaw)
	}
}

func TestTestAllFullRunsOwnershipAuditGate(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "ownership production audit" go run ./tools/cmd/validate-ownership-audit --audit docs/release/ownership_production_audit.md --expected-status achieved`,
		`docs/release/ownership_production_audit.md`,
		`--expected-status achieved`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing ownership audit coverage %q", want)
		}
	}
}

func TestTestAllValidatesJSONDiagnosticShape(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "json diagnostic shape" check_json_diagnostic`,
		`check_json_diagnostic_case "invalid-diagnostic" "unknown function"`,
		`check_json_diagnostic_case "missing-effect-diagnostic" "uses effect 'io'"`,
		`check_json_diagnostic_case "tabs-diagnostic" "tabs are not supported"`,
		`check_json_diagnostic_case "planned-actor-diagnostic" "actor declarations currently support state fields and func methods only"`,
		`--require-position`,
		`./tetra build --target wasm32-wasi -o "$wasm_out" examples/hello.tetra`,
		`test "$(od -An -tx1 -N4 "$wasm_out" | tr -d ' \n')" = "0061736d"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing JSON diagnostic validation %q", want)
		}
	}
}

func TestTestAllValidatesSmokeListJSONReport(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "smoke list json report" check_smoke_list`,
		`./tetra smoke --list --target linux-x64 --format=json >"$report_dir/smoke-list.json"`,
		`go run ./tools/cmd/validate-smoke-list --report "$report_dir/smoke-list.json" --examples-root examples`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing smoke list validation %q", want)
		}
	}
}

func TestSmokeSourceSetsUseUnifiedRegistry(t *testing.T) {
	root := repoRoot(t)
	files := map[string][]string{
		"scripts/ci/test-all.sh": {
			`./tetra smoke --list --target linux-x64 --format=json >"$report_dir/smoke-list.json"`,
			`run_tetra_smoke_target "linux-x64" "true" "$report_dir/host-smoke.json"`,
			`run_tetra_smoke_target "wasm32-wasi" "false" "$report_dir/wasm32-wasi-artifact-smoke.json"`,
			`run_tetra_smoke_target "wasm32-web" "false" "$report_dir/wasm32-web-artifact-smoke.json"`,
		},
		"scripts/release/v1_0/wasi-smoke.sh": {
			`./tetra smoke --list --target wasm32-wasi --format=json >"$smoke_list"`,
			`smoke_source_for_case "$smoke_list" "dogfood_wasi"`,
			`smoke_source_for_case "$smoke_list" "ui_web_smoke"`,
		},
		"scripts/release/v1_0/web-smoke.sh": {
			`./tetra smoke --list --target wasm32-web --format=json >"$smoke_list"`,
			`smoke_source_for_case "$smoke_list" "dogfood_web_ui"`,
			`smoke_source_for_case "$smoke_list" "ui_web_smoke"`,
		},
	}
	for rel, wants := range files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(raw)
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing unified smoke registry contract %q", rel, want)
			}
		}
		for _, forbidden := range []string{
			`release_smoke_cases=`,
			`examples/projects/dogfood_wasi/src/main.tetra`,
			`examples/projects/dogfood_web_ui/src/main.tetra`,
			`examples/ui_web_smoke.tetra`,
			`examples/flow_struct_smoke.tetra`,
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s still hard-codes smoke source %q", rel, forbidden)
			}
		}
	}
}

func TestTestAllWASMSchemaChecksUseArtifactSmokeReports(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "ci", "test-all.sh"))
	if err != nil {
		t.Fatalf("read test-all.sh: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`local report="$report_dir/$target-artifact-smoke.json"`,
		`go run ./tools/cmd/validate-wasi-smoke-report --mode artifact --report "$report"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test-all WASM schema check missing %q", want)
		}
	}
}

func TestTestAllValidatesDocsManifests(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json"`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing manifest validation %q", want)
		}
	}
}

func TestTestAllFullValidatesPerformanceReportWhenPresent(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`check_performance_report()`,
		`go run ./tools/cmd/validate-performance-report --report "$report"`,
		`run_step "performance report schema" check_performance_report`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing performance report wiring %q", want)
		}
	}
}

func TestTestAllFullValidatesTechEmpowerReportsWhenPresent(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`check_techempower_reports()`,
		`docs/benchmarks/techempower_local_smoke_skip_db_report.json`,
		`docs/benchmarks/techempower_scram_single_query_local_report.json`,
		`docs/benchmarks/techempower_scram_single_query_matrix_local_report.json`,
		`docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json`,
		`go run ./tools/cmd/validate-techempower-report --report "$report"`,
		`go run ./tools/cmd/validate-techempower-report --report "$report" --allow-skip-db`,
		`run_step "techempower report schemas" check_techempower_reports`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing TechEmpower report wiring %q", want)
		}
	}
}

func TestTestAllFullRunsTechEmpowerReportValidationForPresentReports(t *testing.T) {
	root := testAllFakeRepo(t, false)
	for _, report := range []string{
		"docs/benchmarks/techempower_local_smoke_skip_db_report.json",
		"docs/benchmarks/techempower_scram_single_query_local_report.json",
		"docs/benchmarks/techempower_scram_single_query_matrix_local_report.json",
		"docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json",
	} {
		path := filepath.Join(root, report)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reportDir := filepath.Join(root, "report")
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_GO_LOG=" + goLog}, "--full", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all full failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if !hasTestAllStep(summary, "techempower report schemas") {
		t.Fatalf("full test_all summary missing TechEmpower report schema step: %#v", summary.Steps)
	}
	rawLog, err := os.ReadFile(goLog)
	if err != nil {
		t.Fatalf("read fake go log: %v", err)
	}
	log := string(rawLog)
	for _, want := range []string{
		`run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_local_smoke_skip_db_report.json --allow-skip-db`,
		`run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_local_report.json`,
		`run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_matrix_local_report.json`,
		`run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json`,
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("full gate missing TechEmpower validator call %q in log:\n%s", want, log)
		}
	}
}

func TestTestAllFullRunsDocsManifestDiffStep(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--full", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all full failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	for _, step := range summary.Steps {
		if step.Name == "docs manifest diff" && step.Status == "pass" {
			return
		}
	}
	t.Fatalf("full test_all summary missing passing docs manifest diff step: %#v", summary.Steps)
}

func TestTestAllValidatesLSPSmokeJSONReport(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `./tetra lsp --stdio-smoke examples/flow_hello.tetra >"$report_dir/lsp-smoke.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing persisted LSP smoke report %q", want)
	}
	validator := `go run ./tools/cmd/validate-lsp-smoke --report "$report_dir/lsp-smoke.json"`
	if !strings.Contains(string(raw), validator) {
		t.Fatalf("test_all missing LSP smoke validation %q", validator)
	}
}

func TestTestAllValidatesLSPStdioTranscript(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-lsp-stdio --transcript "$tmp_dir/lsp-stdio.out"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing LSP stdio validation %q", want)
	}
}

func TestTestAllValidatesGeneratedAPIDocs(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	gen := `go run ./tools/cmd/gen-docs examples >"$report_dir/api-docs.md"`
	if !strings.Contains(string(raw), gen) {
		t.Fatalf("test_all missing persisted generated API docs %q", gen)
	}
	validate := `go run ./tools/cmd/validate-api-docs --docs "$report_dir/api-docs.md"`
	if !strings.Contains(string(raw), validate) {
		t.Fatalf("test_all missing API docs validation %q", validate)
	}
}

func TestReleaseV012GateFormatterCoversRuntimeSources(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_1_2", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.2 release gate: %v", err)
	}
	want := `./tetra fmt --check examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("v0.1.2 release gate missing runtime formatter coverage %q", want)
	}
}

func TestTestAllValidatesEcoLock(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Eco lock validation %q", want)
	}
}

func TestTestAllValidatesEcoUnpack(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Eco unpack validation %q", want)
	}
}

func TestTestAllValidatesEcoVault(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Eco vault validation %q", want)
	}
}

func Test_test_all_validates_tool011_eco_reports(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	for _, want := range []string{
		`go run ./tools/cmd/validate-eco-seed --seed "$tmp_dir/tetra.seed.json"`,
		`go run ./tools/cmd/validate-eco-needmap --needmap "$tmp_dir/tetra.needmap.json"`,
		`go run ./tools/cmd/validate-eco-trust --trust "$tmp_dir/tetra.trust-snapshot.json"`,
		`go run ./tools/cmd/validate-eco-materialization --materialization "$tmp_dir/materialized/tetra.materialization.json"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("test_all missing TOOL-011 Eco validation %q", want)
		}
	}
}

func TestTestAllFullValidatesCrossTargetSmokeReports(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_GO_LOG=" + goLog}, "--full", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all full failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(goLog)
	if err != nil {
		t.Fatalf("read fake go log: %v", err)
	}
	log := string(raw)
	for _, report := range []string{"linux-smoke.json", "macos-smoke.json", "windows-smoke.json", "wasm32-wasi-artifact-smoke.json", "wasm32-web-artifact-smoke.json"} {
		want := "run ./tools/cmd/smoke-report-to-checklist --validate-only --report "
		if !strings.Contains(log, want) || !strings.Contains(log, report) {
			t.Fatalf("full gate missing validate-only call for %s in log:\n%s", report, log)
		}
	}
	if !strings.Contains(log, "run ./tools/cmd/validate-wasi-smoke-report --mode artifact --report ") ||
		!strings.Contains(log, "wasm32-wasi-artifact-smoke.json") {
		t.Fatalf("full gate missing strict WASI artifact/import report validation in log:\n%s", log)
	}
	summary := decodeTestAllSummary(t, out)
	for _, step := range []string{"wasm32-wasi smoke schema", "wasm32-web smoke schema"} {
		if !hasTestAllStep(summary, step) {
			t.Fatalf("full gate missing %q step: %#v", step, summary.Steps)
		}
	}
}

func TestTestAllWasmSchemaValidationUsesPersistedArtifactReports(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`local report="$report_dir/$target-artifact-smoke.json"`,
		`test -s "$report" || return 1`,
		`go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report" || return 1`,
		`go run ./tools/cmd/validate-wasi-smoke-report --mode artifact --report "$report" || return 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all wasm schema validation missing %q", want)
		}
	}
}

func TestTestAllFullAggregatesToolingSummary(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--full", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all full failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if !hasTestAllStep(summary, "tooling summary aggregation") {
		t.Fatalf("full gate missing tooling summary aggregation step: %#v", summary.Steps)
	}
	raw, err := os.ReadFile(filepath.Join(reportDir, "tooling-summary.json"))
	if err != nil {
		t.Fatalf("read tooling summary: %v", err)
	}
	for _, want := range []string{`"schema": "tetra.tooling-summary.v1alpha1"`, `"targets.json"`, `"doctor.json"`, `"safety-readiness.json"`, `"wasm32-wasi-artifact-smoke.json"`, `"wasm32-web-artifact-smoke.json"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("tooling summary missing %q:\n%s", want, raw)
		}
	}
}

func TestTestAllFullToolingSummaryFailsOnZeroByteRequiredArtifact(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_ZERO_DOCTOR_REPORT=1"}, "--full", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected full tooling summary to fail on zero-byte required artifact\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	var sawToolingFailure bool
	for _, step := range summary.Steps {
		if step.Name == "tooling summary aggregation" && step.Status == "fail" {
			sawToolingFailure = true
		}
	}
	if !sawToolingFailure {
		t.Fatalf("summary missing failing tooling summary step: %#v", summary.Steps)
	}
	logRaw, err := os.ReadFile(filepath.Join(reportDir, testAllStepLog(t, summary, "tooling summary aggregation")))
	if err != nil {
		t.Fatalf("read tooling summary log: %v", err)
	}
	log := string(logRaw)
	if !strings.Contains(log, "required artifact is zero-byte: doctor.json") {
		t.Fatalf("tooling summary log missing zero-byte artifact detail:\n%s", log)
	}
}

func TestTestAllStabilizationToolingSummaryRequiresFocusedArtifacts(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_SKIP_WEB_UI_SMOKE_REPORT=1"}, "--stabilization", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected stabilization tooling summary to fail on missing focused artifact\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	var sawToolingFailure bool
	for _, step := range summary.Steps {
		if step.Name == "tooling summary aggregation" && step.Status == "fail" {
			sawToolingFailure = true
		}
	}
	if !sawToolingFailure {
		t.Fatalf("summary missing failing tooling summary step: %#v", summary.Steps)
	}
	logRaw, err := os.ReadFile(filepath.Join(reportDir, testAllStepLog(t, summary, "tooling summary aggregation")))
	if err != nil {
		t.Fatalf("read tooling summary log: %v", err)
	}
	log := string(logRaw)
	if !strings.Contains(log, "required artifact missing: web-ui-smoke.json") {
		t.Fatalf("tooling summary log missing required artifact detail:\n%s", log)
	}
}

func TestTestAllStabilizationRunsFocusedGates(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_GO_LOG=" + goLog}, "--stabilization", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all stabilization failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" {
		t.Fatalf("summary status = %q", summary.Status)
	}
	if !hasTestAllStep(summary, "compiler pipeline focused gate") || !hasTestAllStep(summary, "api diff no-change") {
		t.Fatalf("stabilization summary missing focused gates: %#v", summary.Steps)
	}
	if !hasTestAllStep(summary, "wasi runner smoke") || !hasTestAllStep(summary, "web runtime browser smoke") {
		t.Fatalf("stabilization summary missing smoke gates: %#v", summary.Steps)
	}
	if summary.StepCount <= 24 {
		t.Fatalf("stabilization should extend full gate step count, got %d", summary.StepCount)
	}
	raw, err := os.ReadFile(goLog)
	if err != nil {
		t.Fatalf("read fake go log: %v", err)
	}
	log := string(raw)
	for _, want := range []string{
		"test ./tools/cmd/validate-lsp-stdio/...",
		"./tools/cmd/validate-api-docs/...",
		"./tools/cmd/validate-wasi-smoke-report/...",
		"./tools/cmd/verify-docs/...",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("stabilization missing validator go invocation %q in log:\n%s", want, log)
		}
	}
}
