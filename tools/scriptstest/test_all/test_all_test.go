package testall

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
		t.Fatalf(
			"summary counts = steps:%d failed:%d len:%d",
			summary.StepCount,
			summary.FailedCount,
			len(summary.Steps),
		)
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

func TestTestAllQuickWritesTOONSummaryMirror(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(
		t,
		root,
		nil,
		"--quick",
		"--json-only",
		"--report-dir",
		reportDir,
		"--report-format",
		"both",
	)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" || summary.StepCount == 0 {
		t.Fatalf("summary = %#v", summary)
	}
	if _, err := os.Stat(filepath.Join(reportDir, "summary.json")); err != nil {
		t.Fatalf("expected summary.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(reportDir, "summary.toon")); err != nil {
		t.Fatalf("expected summary.toon mirror: %v", err)
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
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST=1"},
		"--quick",
		"--keep-going",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf("expected unsafe promotion blocker suite failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf(
			"summary status/counts = %q/%d, want fail/1: %#v",
			summary.Status,
			summary.FailedCount,
			summary.Steps,
		)
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
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST=1"},
		"--quick",
		"--keep-going",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf("expected bounds proof blocker suite failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf(
			"summary status/counts = %q/%d, want fail/1: %#v",
			summary.Status,
			summary.FailedCount,
			summary.Steps,
		)
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
		t.Fatalf(
			"summary missing passing memory fuzz oracle artifact gate step: %#v",
			summary.Steps,
		)
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
			t.Fatalf(
				"scripts/ci/test-all.sh missing RAM contract fuzz oracle gate requirement %q",
				want,
			)
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
		t.Fatalf(
			"summary missing passing RAM contract fuzz oracle artifact gate step: %#v",
			summary.Steps,
		)
	}
}

func TestTestAllQuickFailsWhenRAMContractFuzzOracleGateTestsMissing(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_SKIP_RAM_CONTRACT_LIST=1"},
		"--quick",
		"--keep-going",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf("expected RAM contract fuzz oracle gate failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf(
			"summary status/counts = %q/%d, want fail/1: %#v",
			summary.Status,
			summary.FailedCount,
			summary.Steps,
		)
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
		t.Fatalf(
			"summary missing failing RAM contract fuzz oracle artifact gate step: %#v",
			summary.Steps,
		)
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
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST=1"},
		"--quick",
		"--keep-going",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf("expected memory fuzz oracle gate failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf(
			"summary status/counts = %q/%d, want fail/1: %#v",
			summary.Status,
			summary.FailedCount,
			summary.Steps,
		)
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
		t.Fatalf(
			"summary missing failing memory fuzz oracle artifact gate step: %#v",
			summary.Steps,
		)
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
		`TestBrokerCloseReopenWithoutGoroutineLeak`,
		`TestBrokerCloseWithoutCancelStopsServeWatcher`,
		`go test ./cli/internal/actornet -run 'Broker|Leak|CloseWithoutCancel' -count=1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("scripts/ci/test-all.sh missing host leak blocker requirement %q", want)
		}
	}
}

func TestTestAllRunsMemory100ProdStableGate(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test-all script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "Memory100 prod-stable gate" check_memory100_prod_stable_gate`,
		`local memory100_dir="$report_dir/memory-100-prod-stable"`,
		`env GOTELEMETRY=off GOCACHE="$repo_root/.cache/go-build-memory-100-test-all" GOTMPDIR="$repo_root/.cache/go-tmp-memory-100-test-all" bash scripts/release/post_v0_4/memory-100-prod-stable-gate.sh --report-dir "$memory100_dir"`,
		`test -s "$memory100_dir/memory-100-prod-stable-manifest.json"`,
		`test -s "$memory100_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-memory-100-prod-stable --report-dir "$memory100_dir" --current-git-head "$(git rev-parse HEAD)"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf(
				"scripts/ci/test-all.sh missing Memory100 prod-stable gate requirement %q",
				want,
			)
		}
	}
	for _, forbidden := range []string{"GOCACHE=/tmp", "GOTMPDIR=/tmp"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("scripts/ci/test-all.sh must not use tmpfs Go cache marker %q", forbidden)
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
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_SKIP_HOST_LEAK_LIST=1"},
		"--quick",
		"--keep-going",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf("expected host leak blocker suite failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf(
			"summary status/counts = %q/%d, want fail/1: %#v",
			summary.Status,
			summary.FailedCount,
			summary.Steps,
		)
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
		"--report-format",
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
	assertNoLegacyMention(
		t,
		string(helpOut),
		"scripts/test_all.sh",
		"scripts/ci/test-all.sh --report-dir DIR --help",
	)
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
	if err := os.WriteFile(
		filepath.Join(reportDir, "summary.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
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
	if err := os.WriteFile(
		filepath.Join(targetDir, "summary.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(root, "report-link")
	if err := os.Symlink(targetDir, reportDir); err != nil {
		t.Fatalf("create report-dir symlink: %v", err)
	}
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_GO_LOG=" + goLog},
		"--quick",
		"--json-only",
		"--report-dir",
		reportDir,
	)
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
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_GO_LOG=" + goLog},
		"--quick",
		"--json-only",
		"--report-dir",
		reportDir,
	)
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
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_GO_LOG=" + goLog},
		"--quick",
		"--json-only",
		"--report-dir",
		reportDir,
	)
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
	if err := os.WriteFile(
		filepath.Join(root, reportDir, "summary.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
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
	if err := os.WriteFile(
		filepath.Join(targetDir, "summary.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
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
	out, err := runTestAllFromWorkingDir(
		t,
		root,
		nested,
		nil,
		"--quick",
		"--json-only",
		"--report-dir",
		filepath.Join(root, "report"),
	)
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
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAIL_FMT=1"},
		"--quick",
		"--keep-going",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf("expected failing test_all run\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" {
		t.Fatalf("status = %q, want fail", summary.Status)
	}
	if summary.StepCount != len(summary.Steps) || summary.FailedCount != 1 {
		t.Fatalf(
			"summary counts = steps:%d failed:%d len:%d",
			summary.StepCount,
			summary.FailedCount,
			len(summary.Steps),
		)
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
		if step.Name == "host smoke linux-x64" && step.Status == "pass" && step.ExitCode != nil &&
			*step.ExitCode == 0 {
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
	if !strings.Contains(
		string(stderr),
		"warning: summary validation failed; preserving original test failure",
	) {
		t.Fatalf("stderr missing best-effort validator warning:\n%s", stderr)
	}
	summary := decodeTestAllSummary(t, stdout)
	if summary.Status != "fail" {
		t.Fatalf("status = %q, want fail", summary.Status)
	}
	if summary.StepCount != len(summary.Steps) || summary.FailedCount != 1 {
		t.Fatalf(
			"summary counts = steps:%d failed:%d len:%d",
			summary.StepCount,
			summary.FailedCount,
			len(summary.Steps),
		)
	}
	if got := len(summary.Steps); got == 0 {
		t.Fatalf("summary has no steps")
	}
	step := summary.Steps[len(summary.Steps)-1]
	if step.Name != testAllFormatterStepName || step.Status != "fail" || step.ExitCode == nil ||
		*step.ExitCode != 7 {
		t.Fatalf("last failing step = %#v", step)
	}
}
