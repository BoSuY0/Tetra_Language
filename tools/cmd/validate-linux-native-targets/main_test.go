package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestValidateLinuxNativeTargetsAcceptsStrictFamilyEvidence(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", targetsReportWithHostProbeRunSupported(t, "linux-x86", "linux-x32"))
	abiX64 := writeTestFile(t, dir, "linux-x64-abi.json", targetABIReport("linux-x64"))
	atomicX64 := writeTestFile(t, dir, "linux-x64-atomic-stress.json", targetAtomicReport("linux-x64"))
	fuzzX64 := writeTestFile(t, dir, "linux-x64-fuzz.json", targetFuzzReport("linux-x64"))
	abiX86 := writeTestFile(t, dir, "linux-x86-abi.json", targetABIReport("linux-x86"))
	atomicX86 := writeTestFile(t, dir, "linux-x86-atomic-stress.json", targetAtomicReport("linux-x86"))
	fuzzX86 := writeTestFile(t, dir, "linux-x86-fuzz.json", targetFuzzReport("linux-x86"))
	abiX32 := writeTestFile(t, dir, "linux-x32-abi.json", targetABIReport("linux-x32"))
	atomicX32 := writeTestFile(t, dir, "linux-x32-atomic-stress.json", targetAtomicReport("linux-x32"))
	fuzzX32 := writeTestFile(t, dir, "linux-x32-fuzz.json", targetFuzzReport("linux-x32"))
	brutal := writeTestFile(t, dir, "linux-native-targets-brutal.json", suiteReport(requiredBrutalNames()...))
	runnerX64 := writeTestFile(t, dir, "linux-x64-runner.json", runnerReport("linux-x64", "runner"))
	runnerX86 := writeTestFile(t, dir, "linux-x86-runner.json", runnerReport("linux-x86", "runner"))
	runnerX32 := writeTestFile(t, dir, "linux-x32-runner.json", runnerReport("linux-x32", "runner"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		BrutalReport:           brutal,
		ArtifactHashesManifest: hashes,
		TargetReports: []targetReportInput{
			{Triple: "linux-x64", ABIReport: abiX64, AtomicReport: atomicX64, FuzzReport: fuzzX64},
			{Triple: "linux-x86", ABIReport: abiX86, AtomicReport: atomicX86, FuzzReport: fuzzX86},
			{Triple: "linux-x32", ABIReport: abiX32, AtomicReport: atomicX32, FuzzReport: fuzzX32},
		},
		RunnerReports: []targetRunnerInput{
			{Triple: "linux-x64", Report: runnerX64},
			{Triple: "linux-x86", Report: runnerX86},
			{Triple: "linux-x32", Report: runnerX32},
		},
	})
	if err != nil {
		t.Fatalf("validateLinuxNativeTargets failed: %v", err)
	}
}

func TestReadTargetsReportAcceptsMemoryCapabilityFields(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())

	report, err := readTargetsReport(targetsPath)
	if err != nil {
		t.Fatalf("read targets report with memory capability fields: %v", err)
	}
	for _, entry := range report.Targets {
		if entry.Triple == "linux-x64" && entry.MemoryClaimLevel != "production/host_runtime" {
			t.Fatalf("linux-x64 memory claim level = %q", entry.MemoryClaimLevel)
		}
	}
}

func TestValidateLinuxNativeMemoryCapabilityMetadataRejectsBuildOnlyRuntimeClaim(t *testing.T) {
	entry := targetReportEntry{
		Triple:            "linux-x86",
		BuildOnly:         true,
		MemoryBuild:       "yes",
		MemoryLower:       "yes",
		MemoryRun:         "yes",
		MemoryClaimLevel:  "production/host_runtime",
		EvidenceArtifacts: []string{"linux-x86-abi.json", "linux-x86-runner.json"},
	}
	issues := validateLinuxNativeMemoryCapabilityMetadata(entry)
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, "; "), "runtime memory claim") {
		t.Fatalf("expected build-only runtime memory claim rejection, got %v", issues)
	}
}

func TestValidateLinuxNativeTargetsRequiresArtifactHashManifest(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		TargetReports: []targetReportInput{{
			Triple:       "linux-x64",
			ABIReport:    writeTestFile(t, dir, "linux-x64-abi.json", targetABIReport("linux-x64")),
			AtomicReport: writeTestFile(t, dir, "linux-x64-atomic-stress.json", targetAtomicReport("linux-x64")),
			FuzzReport:   writeTestFile(t, dir, "linux-x64-fuzz.json", targetFuzzReport("linux-x64")),
		}},
		RunnerReports: []targetRunnerInput{
			{Triple: "linux-x64", Report: writeTestFile(t, dir, "linux-x64-runner.json", runnerReport("linux-x64", "runner"))},
		},
	})
	if err == nil {
		t.Fatalf("expected missing artifact hash manifest to fail")
	}
	if !strings.Contains(err.Error(), "--artifact-hashes is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRequiresBrutalReportForFullFamilyEvidence(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	abiX64 := writeTestFile(t, dir, "linux-x64-abi.json", targetABIReport("linux-x64"))
	atomicX64 := writeTestFile(t, dir, "linux-x64-atomic-stress.json", targetAtomicReport("linux-x64"))
	fuzzX64 := writeTestFile(t, dir, "linux-x64-fuzz.json", targetFuzzReport("linux-x64"))
	abiX86 := writeTestFile(t, dir, "linux-x86-abi.json", targetABIReport("linux-x86"))
	atomicX86 := writeTestFile(t, dir, "linux-x86-atomic-stress.json", targetAtomicReport("linux-x86"))
	fuzzX86 := writeTestFile(t, dir, "linux-x86-fuzz.json", targetFuzzReport("linux-x86"))
	abiX32 := writeTestFile(t, dir, "linux-x32-abi.json", targetABIReport("linux-x32"))
	atomicX32 := writeTestFile(t, dir, "linux-x32-atomic-stress.json", targetAtomicReport("linux-x32"))
	fuzzX32 := writeTestFile(t, dir, "linux-x32-fuzz.json", targetFuzzReport("linux-x32"))
	runnerX64 := writeTestFile(t, dir, "linux-x64-runner.json", runnerReport("linux-x64", "runner"))
	runnerX86 := writeTestFile(t, dir, "linux-x86-runner.json", runnerReport("linux-x86", "runner"))
	runnerX32 := writeTestFile(t, dir, "linux-x32-runner.json", runnerReport("linux-x32", "runner"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		TargetReports: []targetReportInput{
			{Triple: "linux-x64", ABIReport: abiX64, AtomicReport: atomicX64, FuzzReport: fuzzX64},
			{Triple: "linux-x86", ABIReport: abiX86, AtomicReport: atomicX86, FuzzReport: fuzzX86},
			{Triple: "linux-x32", ABIReport: abiX32, AtomicReport: atomicX32, FuzzReport: fuzzX32},
		},
		RunnerReports: []targetRunnerInput{
			{Triple: "linux-x64", Report: runnerX64},
			{Triple: "linux-x86", Report: runnerX86},
			{Triple: "linux-x32", Report: runnerX32},
		},
	})
	if err == nil {
		t.Fatalf("expected missing brutal report to fail for full-family evidence")
	}
	if !strings.Contains(err.Error(), "linux native brutal report is required for full-family evidence") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsRunnerReportForWrongTarget(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	abiPath := writeTestFile(t, dir, "linux-x86-abi.json", targetABIReport("linux-x86"))
	atomicPath := writeTestFile(t, dir, "linux-x86-atomic-stress.json", targetAtomicReport("linux-x86"))
	fuzzPath := writeTestFile(t, dir, "linux-x86-fuzz.json", targetFuzzReport("linux-x86"))
	runnerPath := writeTestFile(t, dir, "linux-x86-runner.json", suiteReportForTarget("linux-x64", "runner"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		TargetReports: []targetReportInput{{
			Triple:       "linux-x86",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
		RunnerReports: []targetRunnerInput{{Triple: "linux-x86", Report: runnerPath}},
	})
	if err == nil {
		t.Fatalf("expected wrong-target runner report to fail")
	}
	if !strings.Contains(err.Error(), `linux-x86 runner report target = "linux-x64", want linux-x86`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsSuiteReportForWrongTarget(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	abiPath := writeTestFile(t, dir, "linux-x86-abi.json", suiteReportForTarget("linux-x64", requiredABINames("linux-x86")...))
	atomicPath := writeTestFile(t, dir, "linux-x86-atomic-stress.json", suiteReportForTarget("linux-x86", "x86 atomic object matrix", "x86 pointer atomic object width", "x86 atomic concurrency stress oracle"))
	fuzzPath := writeTestFile(t, dir, "linux-x86-fuzz.json", suiteReportForTarget("linux-x86", "x86 layout fuzz", "x86 object signature fuzz"))
	runnerPath := writeTestFile(t, dir, "linux-x86-runner.json", runnerReport("linux-x86", "runner"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		TargetReports: []targetReportInput{{
			Triple:       "linux-x86",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
		RunnerReports: []targetRunnerInput{{Triple: "linux-x86", Report: runnerPath}},
	})
	if err == nil {
		t.Fatalf("expected wrong-target suite report to fail")
	}
	if !strings.Contains(err.Error(), `linux-x86 ABI report target = "linux-x64", want linux-x86`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsStaleArtifactHashManifest(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	abiPath := writeTestFile(t, dir, "linux-x64-abi.json", targetABIReport("linux-x64"))
	atomicPath := writeTestFile(t, dir, "linux-x64-atomic-stress.json", targetAtomicReport("linux-x64"))
	fuzzPath := writeTestFile(t, dir, "linux-x64-fuzz.json", targetFuzzReport("linux-x64"))
	runnerPath := writeTestFile(t, dir, "linux-x64-runner.json", runnerReport("linux-x64", "runner"))
	hashes := writeArtifactHashManifest(t, dir)
	if err := os.WriteFile(abiPath, []byte(suiteReportForTarget("linux-x64", "x64 target model")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		TargetReports: []targetReportInput{{
			Triple:       "linux-x64",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
		RunnerReports: []targetRunnerInput{{Triple: "linux-x64", Report: runnerPath}},
	})
	if err == nil {
		t.Fatalf("expected stale artifact hash manifest to fail")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch for linux-x64-abi.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsArtifactHashManifestThatDoesNotCoverEvidenceFiles(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	abiPath := writeTestFile(t, dir, "linux-x64-abi.json", targetABIReport("linux-x64"))
	atomicPath := writeTestFile(t, dir, "linux-x64-atomic-stress.json", targetAtomicReport("linux-x64"))
	fuzzPath := writeTestFile(t, dir, "linux-x64-fuzz.json", targetFuzzReport("linux-x64"))
	runnerPath := writeTestFile(t, dir, "linux-x64-runner.json", runnerReport("linux-x64", "runner"))

	hashDir := filepath.Join(dir, "hashes")
	if err := os.Mkdir(hashDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, hashDir, "unrelated.json", suiteReport("unrelated"))
	hashes := writeArtifactHashManifest(t, hashDir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		TargetReports: []targetReportInput{{
			Triple:       "linux-x64",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
		RunnerReports: []targetRunnerInput{{Triple: "linux-x64", Report: runnerPath}},
	})
	if err == nil {
		t.Fatalf("expected artifact hash manifest that does not cover evidence files to fail")
	}
	if !strings.Contains(err.Error(), "artifact hash manifest does not cover required evidence file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsABIReportMissingCurrentSuiteResults(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	legacySubsetABI := writeTestFile(t, dir, "linux-x86-abi.json", suiteReportForTarget("linux-x86",
		"x86 target model",
		"x86 pointer FFI object smoke",
		"x86 ref FFI null-return diagnostics",
		"x86 function-pointer FFI diagnostics",
		"x86 source native scalar diagnostics",
		"x86 stdout executable smoke",
		"x86 stdlib runtime boundary diagnostics",
		"x86 filesystem runtime smoke",
		"x86 filesystem scheduler composition smoke",
		"x86 time runtime smoke",
		"x86 single-actor self-host runtime smoke",
		"x86 single-task self-host runtime smoke",
		"x86 typed-task self-host runtime smoke",
		"x86 staged typed-task self-host runtime smoke",
		"x86 task-group self-host runtime smoke",
		"x86 typed-task-group self-host runtime smoke",
		"x86 actor-state self-host runtime smoke",
		"x86 ctx_switch object smoke",
		"x86 networking runtime boundary diagnostics",
		"x86 surface/distributed runtime boundary diagnostics",
		"x86 object ABI smoke",
		"x86 executable matrix smoke",
	))

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		TargetReports: []targetReportInput{{
			Triple:       "linux-x86",
			ABIReport:    legacySubsetABI,
			AtomicReport: writeTestFile(t, dir, "x86-atomic.json", targetAtomicReport("linux-x86")),
			FuzzReport:   writeTestFile(t, dir, "x86-fuzz.json", targetFuzzReport("linux-x86")),
		}},
		RunnerReports: []targetRunnerInput{
			{Triple: "linux-x86", Report: writeTestFile(t, dir, "linux-x86-runner.json", runnerDiagnostic("cannot run tests for target linux-x86: host does not support Linux i386 execution; no host fallback is allowed"))},
		},
	})
	if err == nil {
		t.Fatalf("expected legacy-subset x86 ABI report to fail")
	}
	if !strings.Contains(err.Error(), `linux-x86 ABI report missing required result "x86 i386 SysV classifier"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsX64BuildOnlyBoundaryDiagnostics(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	x64Names := append([]string{}, requiredABINames("linux-x64")...)
	x64Names = append(x64Names, "x64 target runtime boundary diagnostics")
	abiPath := writeTestFile(t, dir, "linux-x64-abi.json", suiteReportForTarget("linux-x64", x64Names...))
	atomicPath := writeTestFile(t, dir, "linux-x64-atomic-stress.json", targetAtomicReport("linux-x64"))
	fuzzPath := writeTestFile(t, dir, "linux-x64-fuzz.json", targetFuzzReport("linux-x64"))
	runnerPath := writeTestFile(t, dir, "linux-x64-runner.json", runnerReport("linux-x64", "runner"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		TargetReports: []targetReportInput{{
			Triple:       "linux-x64",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
		RunnerReports: []targetRunnerInput{{Triple: "linux-x64", Report: runnerPath}},
	})
	if err == nil {
		t.Fatalf("expected x64 ABI report with build-only boundary diagnostics to fail")
	}
	if !strings.Contains(err.Error(), `linux-x64 ABI report contains build-only boundary result "x64 target runtime boundary diagnostics"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsBrutalReportMissingPerTargetRequiredResults(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	abiX86 := writeTestFile(t, dir, "linux-x86-abi.json", targetABIReport("linux-x86"))
	atomicX86 := writeTestFile(t, dir, "linux-x86-atomic-stress.json", targetAtomicReport("linux-x86"))
	fuzzX86 := writeTestFile(t, dir, "linux-x86-fuzz.json", targetFuzzReport("linux-x86"))
	brutal := writeTestFile(t, dir, "linux-native-targets-brutal.json", suiteReport(
		"x86 target model",
		"x86 pointer FFI object smoke",
		"x86 filesystem runtime smoke",
		"x86 filesystem scheduler composition smoke",
		"x86 time runtime smoke",
		"x86 stdout executable smoke",
		"x86 typed-task self-host runtime smoke",
		"x86 staged typed-task self-host runtime smoke",
		"x86 task-group self-host runtime smoke",
		"x86 typed-task-group self-host runtime smoke",
		"x86 ctx_switch object smoke",
		"x86 surface/distributed runtime boundary diagnostics",
		"x86 networking runtime boundary diagnostics",
		"x64 target model",
		"x64 pointer FFI regression smoke",
		"x64 filesystem scheduler composition smoke",
		"x64 scheduler restriction regression smoke",
		"x32 target model",
		"x32 pointer FFI object smoke",
		"x32 time runtime smoke",
		"x32 filesystem runtime smoke",
		"x32 stdout executable smoke",
		"x32 filesystem scheduler composition smoke",
		"x32 single-actor self-host runtime smoke",
		"x32 single-task self-host runtime smoke",
		"x32 typed-task self-host runtime smoke",
		"x32 staged typed-task self-host runtime smoke",
		"x32 task-group self-host runtime smoke",
		"x32 typed-task-group self-host runtime smoke",
		"x32 actor-state self-host runtime smoke",
		"x32 ctx_switch object smoke",
		"x32 surface/distributed runtime boundary diagnostics",
		"x32 networking runtime boundary diagnostics",
		"x86 atomic object matrix",
		"x64 atomic object matrix",
		"x32 atomic object matrix",
		"x86 layout fuzz",
		"x64 layout fuzz",
		"x32 layout fuzz",
	))

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		BrutalReport:  brutal,
		TargetReports: []targetReportInput{
			{Triple: "linux-x86", ABIReport: abiX86, AtomicReport: atomicX86, FuzzReport: fuzzX86},
		},
		RunnerReports: []targetRunnerInput{
			{Triple: "linux-x86", Report: writeTestFile(t, dir, "linux-x86-runner.json", runnerDiagnostic("cannot run tests for target linux-x86: host does not support Linux i386 execution; no host fallback is allowed"))},
		},
	})
	if err == nil {
		t.Fatalf("expected incomplete brutal report to fail")
	}
	if !strings.Contains(err.Error(), `linux native brutal report missing required result "x86 i386 SysV classifier"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsX32CollapsedToX64(t *testing.T) {
	dir := t.TempDir()
	targets := strings.Replace(validTargetsReport(), `"abi":"x32-sysv","data_model":"x32"`, `"abi":"sysv","data_model":"lp64"`, 1)
	targets = strings.Replace(targets, `"pointer_width_bits":32,"register_width_bits":64,"native_int_width_bits":32`, `"pointer_width_bits":64,"register_width_bits":64,"native_int_width_bits":64`, 1)
	targetsPath := writeTestFile(t, dir, "targets.json", targets)
	report := writeTestFile(t, dir, "x32-abi.json", suiteReportForTarget("linux-x32", "x32 target model", "x32 object ABI smoke", "x32 executable matrix smoke"))

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		TargetReports: []targetReportInput{{Triple: "linux-x32", ABIReport: report, AtomicReport: report, FuzzReport: report}},
	})
	if err == nil {
		t.Fatalf("expected x32 collapse to fail")
	}
	if !strings.Contains(err.Error(), "linux-x32") || !strings.Contains(err.Error(), "x32-sysv") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsX32PlainX64SyscallPack(t *testing.T) {
	dir := t.TempDir()
	targets := strings.Replace(validTargetsReport(), `"syscall_numbering":"x32_syscall_bit"`, `"syscall_numbering":"x86_64"`, 1)
	targetsPath := writeTestFile(t, dir, "targets.json", targets)
	report := writeTestFile(t, dir, "x32-abi.json", suiteReportForTarget("linux-x32", "x32 target model", "x32 object ABI smoke", "x32 executable matrix smoke"))

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		TargetReports: []targetReportInput{{Triple: "linux-x32", ABIReport: report, AtomicReport: report, FuzzReport: report}},
	})
	if err == nil {
		t.Fatalf("expected x32 plain x64 syscall pack to fail")
	}
	if !strings.Contains(err.Error(), "linux-x32") || !strings.Contains(err.Error(), "x32_syscall_bit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsPrematureX86Promotion(t *testing.T) {
	dir := t.TempDir()
	targets := strings.Replace(validTargetsReport(), `"triple":"linux-x86","status":"build_only"`, `"triple":"linux-x86","status":"supported"`, 1)
	targets = strings.Replace(targets, `"build_only":true,"run_mode":"host_probed"`, `"build_only":false,"run_mode":"host_native"`, 1)
	targetsPath := writeTestFile(t, dir, "targets.json", targets)
	report := writeTestFile(t, dir, "x86-abi.json", suiteReportForTarget("linux-x86", "x86 target model", "x86 object ABI smoke", "x86 executable matrix smoke"))

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		TargetReports: []targetReportInput{{Triple: "linux-x86", ABIReport: report, AtomicReport: report, FuzzReport: report}},
	})
	if err == nil {
		t.Fatalf("expected premature x86 promotion to fail")
	}
	if !strings.Contains(err.Error(), "linux-x86") || !strings.Contains(err.Error(), "build_only") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsPaperEvidence(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	report := writeTestFile(t, dir, "x64-abi.json", `{
  "total":1,
  "passed":1,
  "failed":0,
  "target":"linux-x64",
  "duration_ms":1,
  "files":[{"filename":"tetra:x64-abi","total":1,"passed":1,"failed":0,"duration_ms":1}],
  "results":[{"name":"x64 fake skipped ABI smoke","filename":"tetra:x64-abi","index":0,"function_name":"__tetra_test_fake","exit_code":0,"passed":true,"duration_ms":1}]
}`)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		TargetReports: []targetReportInput{{Triple: "linux-x64", ABIReport: report, AtomicReport: report, FuzzReport: report}},
	})
	if err == nil {
		t.Fatalf("expected paper evidence to fail")
	}
	if !strings.Contains(err.Error(), "paper evidence") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsBrutalReportWithX64BuildOnlyBoundaryDiagnostics(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	abiX64 := writeTestFile(t, dir, "linux-x64-abi.json", targetABIReport("linux-x64"))
	atomicX64 := writeTestFile(t, dir, "linux-x64-atomic-stress.json", targetAtomicReport("linux-x64"))
	fuzzX64 := writeTestFile(t, dir, "linux-x64-fuzz.json", targetFuzzReport("linux-x64"))
	abiX86 := writeTestFile(t, dir, "linux-x86-abi.json", targetABIReport("linux-x86"))
	atomicX86 := writeTestFile(t, dir, "linux-x86-atomic-stress.json", targetAtomicReport("linux-x86"))
	fuzzX86 := writeTestFile(t, dir, "linux-x86-fuzz.json", targetFuzzReport("linux-x86"))
	abiX32 := writeTestFile(t, dir, "linux-x32-abi.json", targetABIReport("linux-x32"))
	atomicX32 := writeTestFile(t, dir, "linux-x32-atomic-stress.json", targetAtomicReport("linux-x32"))
	fuzzX32 := writeTestFile(t, dir, "linux-x32-fuzz.json", targetFuzzReport("linux-x32"))
	brutalNames := append([]string{}, requiredBrutalNames()...)
	brutalNames = append(brutalNames, "x64 target runtime boundary diagnostics")
	brutal := writeTestFile(t, dir, "linux-native-targets-brutal.json", suiteReport(brutalNames...))
	runnerX64 := writeTestFile(t, dir, "linux-x64-runner.json", runnerReport("linux-x64", "runner"))
	runnerX86 := writeTestFile(t, dir, "linux-x86-runner.json", runnerReport("linux-x86", "runner"))
	runnerX32 := writeTestFile(t, dir, "linux-x32-runner.json", runnerReport("linux-x32", "runner"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		BrutalReport:           brutal,
		ArtifactHashesManifest: hashes,
		TargetReports: []targetReportInput{
			{Triple: "linux-x64", ABIReport: abiX64, AtomicReport: atomicX64, FuzzReport: fuzzX64},
			{Triple: "linux-x86", ABIReport: abiX86, AtomicReport: atomicX86, FuzzReport: fuzzX86},
			{Triple: "linux-x32", ABIReport: abiX32, AtomicReport: atomicX32, FuzzReport: fuzzX32},
		},
		RunnerReports: []targetRunnerInput{
			{Triple: "linux-x64", Report: runnerX64},
			{Triple: "linux-x86", Report: runnerX86},
			{Triple: "linux-x32", Report: runnerX32},
		},
	})
	if err == nil {
		t.Fatalf("expected brutal report with x64 build-only boundary diagnostics to fail")
	}
	if !strings.Contains(err.Error(), `linux native brutal report contains build-only boundary result "x64 target runtime boundary diagnostics"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRequiresRunnerEvidenceForTargetReport(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		TargetReports: []targetReportInput{{
			Triple:       "linux-x64",
			ABIReport:    writeTestFile(t, dir, "x64-abi.json", targetABIReport("linux-x64")),
			AtomicReport: writeTestFile(t, dir, "x64-atomic.json", targetAtomicReport("linux-x64")),
			FuzzReport:   writeTestFile(t, dir, "x64-fuzz.json", targetFuzzReport("linux-x64")),
		}},
	})
	if err == nil {
		t.Fatalf("expected missing runner evidence to fail")
	}
	if !strings.Contains(err.Error(), "linux-x64 runner report is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsRunnerReportWithoutRunnerSmoke(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	report := writeTestFile(t, dir, "x64-runner.json", runnerReport("linux-x64", "not the runner smoke"))

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		RunnerReports: []targetRunnerInput{{Triple: "linux-x64", Report: report}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x64",
			ABIReport:    writeTestFile(t, dir, "x64-abi.json", targetABIReport("linux-x64")),
			AtomicReport: writeTestFile(t, dir, "x64-atomic.json", targetAtomicReport("linux-x64")),
			FuzzReport:   writeTestFile(t, dir, "x64-fuzz.json", targetFuzzReport("linux-x64")),
		}},
	})
	if err == nil {
		t.Fatalf("expected non-runner suite to fail")
	}
	if !strings.Contains(err.Error(), `missing required result "runner arithmetic"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsRunnerReportMissingRuntimeSmokeResults(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	abiPath := writeTestFile(t, dir, "linux-x86-abi.json", targetABIReport("linux-x86"))
	atomicPath := writeTestFile(t, dir, "linux-x86-atomic-stress.json", targetAtomicReport("linux-x86"))
	fuzzPath := writeTestFile(t, dir, "linux-x86-fuzz.json", targetFuzzReport("linux-x86"))
	runnerPath := writeTestFile(t, dir, "linux-x86-runner.json", suiteReportForTarget("linux-x86", "runner"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		RunnerReports:          []targetRunnerInput{{Triple: "linux-x86", Report: runnerPath}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x86",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
	})
	if err == nil {
		t.Fatalf("expected runner report missing runtime smoke results to fail")
	}
	if !strings.Contains(err.Error(), `missing required result "runner alloc memory"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsRunnerReportMissingTimeSmoke(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", targetsReportWithHostProbeRunSupported(t, "linux-x86"))
	names := withoutString(requiredRunnerNames("linux-x86"), "runner time")
	abiPath := writeTestFile(t, dir, "linux-x86-abi.json", targetABIReport("linux-x86"))
	atomicPath := writeTestFile(t, dir, "linux-x86-atomic-stress.json", targetAtomicReport("linux-x86"))
	fuzzPath := writeTestFile(t, dir, "linux-x86-fuzz.json", targetFuzzReport("linux-x86"))
	runnerPath := writeTestFile(t, dir, "linux-x86-runner.json", suiteReportForTarget("linux-x86", names...))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		RunnerReports:          []targetRunnerInput{{Triple: "linux-x86", Report: runnerPath}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x86",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
	})
	if err == nil {
		t.Fatalf("expected runner report missing time smoke to fail")
	}
	if !strings.Contains(err.Error(), `missing required result "runner time"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsRunnerReportMissingNetworkSocketSmoke(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", targetsReportWithHostProbeRunSupported(t, "linux-x32"))
	names := withoutString(requiredRunnerNames("linux-x32"), "runner network socket")
	abiPath := writeTestFile(t, dir, "linux-x32-abi.json", targetABIReport("linux-x32"))
	atomicPath := writeTestFile(t, dir, "linux-x32-atomic-stress.json", targetAtomicReport("linux-x32"))
	fuzzPath := writeTestFile(t, dir, "linux-x32-fuzz.json", targetFuzzReport("linux-x32"))
	runnerPath := writeTestFile(t, dir, "linux-x32-runner.json", suiteReportForTarget("linux-x32", names...))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		RunnerReports:          []targetRunnerInput{{Triple: "linux-x32", Report: runnerPath}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x32",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
	})
	if err == nil {
		t.Fatalf("expected runner report missing network socket smoke to fail")
	}
	if !strings.Contains(err.Error(), `missing required result "runner network socket"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsRunnerReportMissingNetworkOptionsSmoke(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", targetsReportWithHostProbeRunSupported(t, "linux-x86"))
	names := withoutString(requiredRunnerNames("linux-x86"), "runner network options")
	abiPath := writeTestFile(t, dir, "linux-x86-abi.json", targetABIReport("linux-x86"))
	atomicPath := writeTestFile(t, dir, "linux-x86-atomic-stress.json", targetAtomicReport("linux-x86"))
	fuzzPath := writeTestFile(t, dir, "linux-x86-fuzz.json", targetFuzzReport("linux-x86"))
	runnerPath := writeTestFile(t, dir, "linux-x86-runner.json", suiteReportForTarget("linux-x86", names...))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		RunnerReports:          []targetRunnerInput{{Triple: "linux-x86", Report: runnerPath}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x86",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
	})
	if err == nil {
		t.Fatalf("expected runner report missing network options smoke to fail")
	}
	if !strings.Contains(err.Error(), `missing required result "runner network options"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsAcceptsBuildOnlyNoHostFallbackRunnerDiagnostic(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	diagPath := writeTestFile(t, dir, "x32-runner-blocked.json", runnerDiagnostic("cannot run tests for target linux-x32: host does not support Linux x32 ABI execution; no host fallback is allowed"))
	abiPath := writeTestFile(t, dir, "x32-abi.json", targetABIReport("linux-x32"))
	atomicPath := writeTestFile(t, dir, "x32-atomic.json", targetAtomicReport("linux-x32"))
	fuzzPath := writeTestFile(t, dir, "x32-fuzz.json", targetFuzzReport("linux-x32"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		RunnerReports:          []targetRunnerInput{{Triple: "linux-x32", Report: diagPath}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x32",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
	})
	if err != nil {
		t.Fatalf("validateLinuxNativeTargets blocked runner diagnostic failed: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsNoHostRunnerDiagnosticWhenMetadataRunSupported(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", targetsReportWithHostProbeRunSupported(t, "linux-x32"))
	diagPath := writeTestFile(t, dir, "x32-runner-blocked.json", runnerDiagnostic("cannot run tests for target linux-x32: host does not support Linux x32 ABI execution; no host fallback is allowed"))
	abiPath := writeTestFile(t, dir, "x32-abi.json", targetABIReport("linux-x32"))
	atomicPath := writeTestFile(t, dir, "x32-atomic.json", targetAtomicReport("linux-x32"))
	fuzzPath := writeTestFile(t, dir, "x32-fuzz.json", targetFuzzReport("linux-x32"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		RunnerReports:          []targetRunnerInput{{Triple: "linux-x32", Report: diagPath}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x32",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
	})
	if err == nil {
		t.Fatalf("expected no-host runner diagnostic to fail when metadata says run_supported=true")
	}
	if !strings.Contains(err.Error(), "run_supported=true") || !strings.Contains(err.Error(), "no-host") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsPassingRunnerReportWhenMetadataRunUnsupported(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	runnerPath := writeTestFile(t, dir, "x86-runner.json", runnerReport("linux-x86", "runner"))
	abiPath := writeTestFile(t, dir, "x86-abi.json", targetABIReport("linux-x86"))
	atomicPath := writeTestFile(t, dir, "x86-atomic.json", targetAtomicReport("linux-x86"))
	fuzzPath := writeTestFile(t, dir, "x86-fuzz.json", targetFuzzReport("linux-x86"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		RunnerReports:          []targetRunnerInput{{Triple: "linux-x86", Report: runnerPath}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x86",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
	})
	if err == nil {
		t.Fatalf("expected passing runner report to fail when metadata says run_supported=false")
	}
	if !strings.Contains(err.Error(), "run_supported=false") || !strings.Contains(err.Error(), "passing runner") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsGenericBuildOnlyRunnerDiagnosticCode(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	diagPath := writeTestFile(t, dir, "x86-runner-blocked.json", `{"code":"TETRA0001","message":"cannot run tests for target linux-x86: host does not support Linux i386 execution; no host fallback is allowed","hint":"","severity":"error"}`)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		RunnerReports: []targetRunnerInput{{Triple: "linux-x86", Report: diagPath}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x86",
			ABIReport:    writeTestFile(t, dir, "x86-abi.json", targetABIReport("linux-x86")),
			AtomicReport: writeTestFile(t, dir, "x86-atomic.json", targetAtomicReport("linux-x86")),
			FuzzReport:   writeTestFile(t, dir, "x86-fuzz.json", targetFuzzReport("linux-x86")),
		}},
	})
	if err == nil {
		t.Fatalf("expected generic runner diagnostic code to fail")
	}
	if !strings.Contains(err.Error(), "TETRA3003") || !strings.Contains(err.Error(), "diagnostic code") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsRunnerDiagnosticWithoutProbeCommand(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	diagPath := writeTestFile(t, dir, "x32-runner-blocked.json", `{"code":"TETRA3003","message":"cannot run tests for target linux-x32: host does not support Linux x32 ABI execution; no host fallback is allowed","hint":"","severity":"error"}`)
	abiPath := writeTestFile(t, dir, "x32-abi.json", targetABIReport("linux-x32"))
	atomicPath := writeTestFile(t, dir, "x32-atomic.json", targetAtomicReport("linux-x32"))
	fuzzPath := writeTestFile(t, dir, "x32-fuzz.json", targetFuzzReport("linux-x32"))
	hashes := writeArtifactHashManifest(t, dir)

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport:          targetsPath,
		ArtifactHashesManifest: hashes,
		RunnerReports:          []targetRunnerInput{{Triple: "linux-x32", Report: diagPath}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x32",
			ABIReport:    abiPath,
			AtomicReport: atomicPath,
			FuzzReport:   fuzzPath,
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "probe command") {
		t.Fatalf("expected missing probe command failure, got %v", err)
	}
}

func TestValidateLinuxNativeTargetsRejectsLinuxX64BlockedRunnerDiagnostic(t *testing.T) {
	dir := t.TempDir()
	targetsPath := writeTestFile(t, dir, "targets.json", validTargetsReport())
	diagPath := writeTestFile(t, dir, "x64-runner-blocked.json", runnerDiagnostic("cannot run tests for target linux-x64: no host fallback is allowed"))

	err := validateLinuxNativeTargets(validateOptions{
		TargetsReport: targetsPath,
		RunnerReports: []targetRunnerInput{{Triple: "linux-x64", Report: diagPath}},
		TargetReports: []targetReportInput{{
			Triple:       "linux-x64",
			ABIReport:    writeTestFile(t, dir, "x64-abi.json", targetABIReport("linux-x64")),
			AtomicReport: writeTestFile(t, dir, "x64-atomic.json", targetAtomicReport("linux-x64")),
			FuzzReport:   writeTestFile(t, dir, "x64-fuzz.json", targetFuzzReport("linux-x64")),
		}},
	})
	if err == nil {
		t.Fatalf("expected blocked linux-x64 runner diagnostic to fail")
	}
	if !strings.Contains(err.Error(), "linux-x64") || !strings.Contains(err.Error(), "runner") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTestFile(t *testing.T, dir string, name string, text string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeArtifactHashManifest(t *testing.T, dir string) string {
	t.Helper()
	const manifestName = "artifact-hashes.json"
	type artifact struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
		Schema string `json:"schema,omitempty"`
	}
	var artifacts []artifact
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == manifestName {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		artifacts = append(artifacts, artifact{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
			Schema: testJSONSchema(raw, path),
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	manifest := struct {
		Schema    string     `json:"schema"`
		Root      string     `json:"root"`
		Artifacts []artifact `json:"artifacts"`
	}{
		Schema:    artifactHashManifestSchema,
		Root:      ".",
		Artifacts: artifacts,
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, manifestName)
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func testJSONSchema(raw []byte, path string) string {
	if filepath.Ext(path) != ".json" {
		return ""
	}
	var envelope struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	return envelope.Schema
}

func runnerReport(target string, name string) string {
	if name == "runner" {
		return suiteReportForTarget(target, requiredRunnerNames(target)...)
	}
	return suiteReportForTarget(target, name)
}

func targetABIReport(triple string) string {
	return suiteReportForTarget(triple, requiredABINames(triple)...)
}

func targetAtomicReport(triple string) string {
	return suiteReportForTarget(triple, requiredAtomicNames(triple)...)
}

func targetFuzzReport(triple string) string {
	return suiteReportForTarget(triple, requiredFuzzNames(triple)...)
}

func runnerDiagnostic(message string) string {
	if !strings.Contains(message, "probe command:") {
		switch {
		case strings.Contains(message, "linux-x86"):
			message += "; probe command: tetra test --diagnostics=json --target x86 --format=json <runner-smoke.tetra>"
		case strings.Contains(message, "linux-x32"):
			message += "; probe command: tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>"
		}
	}
	return `{"code":"TETRA3003","message":"` + message + `","hint":"","severity":"error"}`
}

func suiteReport(names ...string) string {
	return suiteReportForTarget("", names...)
}

func suiteReportForTarget(target string, names ...string) string {
	var b strings.Builder
	b.WriteString(`{"total":`)
	b.WriteString(strconv.Itoa(len(names)))
	b.WriteString(`,"passed":`)
	b.WriteString(strconv.Itoa(len(names)))
	if target != "" {
		b.WriteString(`,"target":"`)
		b.WriteString(target)
		b.WriteString(`"`)
	}
	b.WriteString(`,"failed":0,"duration_ms":1,"files":[{"filename":"suite","total":`)
	b.WriteString(strconv.Itoa(len(names)))
	b.WriteString(`,"passed":`)
	b.WriteString(strconv.Itoa(len(names)))
	b.WriteString(`,"failed":0,"duration_ms":1}],"results":[`)
	for i, name := range names {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"name":"`)
		b.WriteString(name)
		b.WriteString(`","filename":"suite","index":`)
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`,"function_name":"__tetra_test_case","exit_code":0,"passed":true,"duration_ms":1}`)
	}
	b.WriteString(`]}`)
	return b.String()
}

func validTargetsReport() string {
	return `{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":["linux-x86","linux-x32"],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","data_model":"lp64","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":true,"pointer_width_bits":64,"register_width_bits":64,"native_int_width_bits":64,"endian":"little","stack_alignment_bytes":16,"max_atomic_width_bits":64,"atomic_width_bits":[8,16,32,64],"atomic_pointer_width_bits":64,"runtime_status":"production","stdlib_status":"production","ffi_status":"scalar_object_smokes_partial","memory_build":"yes","memory_lower":"yes","memory_run":"yes","memory_raw_diagnostics":"yes","memory_region_lowering":"yes/partial","memory_alignment_semantics":"yes","memory_claim_level":"production/host_runtime","runner_probe_command":"tetra test --target x64 --format=json <runner-smoke.tetra>","release_gate":"scripts/release/post_v0_4/linux-native-targets-smoke.sh","evidence_artifacts":["targets.json","linux-x64-abi.json","linux-x64-atomic-stress.json","linux-x64-fuzz.json","linux-x64-runner.json","linux-native-targets-brutal.json","artifact-hashes.json"],"syscall_instruction":"syscall","syscall_numbering":"x86_64","syscall_arg_registers":["rax","rdi","rsi","rdx","r10","r8","r9"],"syscall_error_range":"-4095..-1","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","data_model":"llp64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","pointer_width_bits":64,"register_width_bits":64,"native_int_width_bits":64,"endian":"little","stack_alignment_bytes":16,"max_atomic_width_bits":64,"atomic_width_bits":[8,16,32,64],"atomic_pointer_width_bits":64,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","data_model":"lp64","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","pointer_width_bits":64,"register_width_bits":64,"native_int_width_bits":64,"endian":"little","stack_alignment_bytes":16,"max_atomic_width_bits":64,"atomic_width_bits":[8,16,32,64],"atomic_pointer_width_bits":64,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","data_model":"ilp32","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","run_supported":false,"run_unsupported_reason":"cannot run target wasm32-wasi: missing WASI runner: need wasmtime or node","pointer_width_bits":32,"register_width_bits":32,"native_int_width_bits":32,"endian":"little","stack_alignment_bytes":16,"max_atomic_width_bits":64,"atomic_width_bits":[8,16,32,64],"atomic_pointer_width_bits":32,"supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","data_model":"ilp32","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","pointer_width_bits":32,"register_width_bits":32,"native_int_width_bits":32,"endian":"little","stack_alignment_bytes":16,"max_atomic_width_bits":64,"atomic_width_bits":[8,16,32,64],"atomic_pointer_width_bits":32,"supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"linux-x86","status":"build_only","os":"linux","arch":"x86","abi":"i386-sysv","data_model":"ilp32","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host linux/amd64 does not support Linux i386 execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x86 --format=json <runner-smoke.tetra>","pointer_width_bits":32,"register_width_bits":32,"native_int_width_bits":32,"endian":"little","stack_alignment_bytes":16,"max_atomic_width_bits":32,"atomic_width_bits":[8,16,32],"atomic_pointer_width_bits":32,"unsupported_reason":"full linux-x86 runtime/stdlib/FFI support is not implemented yet","runtime_status":"partial_build_only","stdlib_status":"partial_build_only","ffi_status":"ilp32_scalar_object_smokes_partial","memory_build":"yes","memory_lower":"yes","memory_run":"no/host-dependent","memory_raw_diagnostics":"partial","memory_region_lowering":"partial","memory_alignment_semantics":"partial","memory_claim_level":"build_lower_only","runner_probe_command":"tetra test --diagnostics=json --target x86 --format=json <runner-smoke.tetra>","release_gate":"scripts/release/post_v0_4/linux-native-targets-smoke.sh","evidence_artifacts":["targets.json","linux-x86-abi.json","linux-x86-atomic-stress.json","linux-x86-fuzz.json","linux-x86-runner.json","linux-native-targets-brutal.json","artifact-hashes.json"],"syscall_instruction":"int 0x80","syscall_numbering":"i386","syscall_arg_registers":["eax","ebx","ecx","edx","esi","edi","ebp"],"syscall_error_range":"-4095..-1","supports_debug_info":false,"supports_release_optimize":false},
    {"triple":"linux-x32","status":"build_only","os":"linux","arch":"x64","abi":"x32-sysv","data_model":"x32","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host linux/amd64 does not support Linux x32 ABI execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>","pointer_width_bits":32,"register_width_bits":64,"native_int_width_bits":32,"endian":"little","stack_alignment_bytes":16,"max_atomic_width_bits":64,"atomic_width_bits":[8,16,32,64],"atomic_pointer_width_bits":32,"unsupported_reason":"full linux-x32 runtime/stdlib/FFI support is not implemented yet","runtime_status":"partial_build_only","stdlib_status":"partial_build_only","ffi_status":"ilp32_scalar_object_smokes_partial","memory_build":"yes","memory_lower":"yes","memory_run":"no/host-dependent","memory_raw_diagnostics":"partial","memory_region_lowering":"partial","memory_alignment_semantics":"special","memory_claim_level":"build_lower_only","runner_probe_command":"tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>","release_gate":"scripts/release/post_v0_4/linux-native-targets-smoke.sh","evidence_artifacts":["targets.json","linux-x32-abi.json","linux-x32-atomic-stress.json","linux-x32-fuzz.json","linux-x32-runner.json","linux-native-targets-brutal.json","artifact-hashes.json"],"syscall_instruction":"syscall","syscall_numbering":"x32_syscall_bit","syscall_arg_registers":["rax","rdi","rsi","rdx","r10","r8","r9"],"syscall_error_range":"-4095..-1","supports_debug_info":false,"supports_release_optimize":false}
  ]
}`
}

func targetsReportWithHostProbeRunSupported(t *testing.T, triples ...string) string {
	t.Helper()
	report := validTargetsReport()
	for _, triple := range triples {
		var old string
		switch triple {
		case "linux-x86":
			old = `"run_supported":false,"run_unsupported_reason":"host linux/amd64 does not support Linux i386 execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x86 --format=json <runner-smoke.tetra>","pointer_width_bits":32`
		case "linux-x32":
			old = `"run_supported":false,"run_unsupported_reason":"host linux/amd64 does not support Linux x32 ABI execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>","pointer_width_bits":32`
		default:
			t.Fatalf("unsupported host-probed target %s", triple)
		}
		next := strings.Replace(report, old, `"run_supported":true,"pointer_width_bits":32`, 1)
		if next == report {
			t.Fatalf("failed to rewrite %s run_supported metadata", triple)
		}
		report = next
	}
	return report
}

func withoutString(values []string, drop string) []string {
	var out []string
	for _, value := range values {
		if value != drop {
			out = append(out, value)
		}
	}
	return out
}
