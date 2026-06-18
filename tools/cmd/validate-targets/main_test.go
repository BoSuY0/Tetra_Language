package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tetra_language/internal/toon"
)

func TestReadTargetsReportDefaultsToTargetsCommand(t *testing.T) {
	raw, err := readTargetsReport("")
	if err != nil {
		t.Fatalf("read default targets report: %v", err)
	}
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate default targets report: %v", err)
	}
}

func TestValidateTargetsReportAcceptsTOON(t *testing.T) {
	rawJSON, err := readTargetsReport("")
	if err != nil {
		t.Fatalf("read default targets report: %v", err)
	}
	rawTOON, err := toon.ConvertJSONToTOON(rawJSON, toon.Options{Deterministic: true, Strict: true})
	if err != nil {
		t.Fatalf("json->toon: %v", err)
	}
	if err := validateTargetsReport(rawTOON); err != nil {
		t.Fatalf("validate targets TOON: %v\n%s", err, rawTOON)
	}
}

func TestReadTargetsReportReportsTargetsCommandFailure(t *testing.T) {
	old := runTargetsCommand
	runTargetsCommand = func() ([]byte, error) {
		return []byte("runner failed"), errors.New("exit 127")
	}
	defer func() { runTargetsCommand = old }()

	_, err := readTargetsReport("")
	if err == nil || !strings.Contains(err.Error(), "runner failed") ||
		!strings.Contains(err.Error(), "exit 127") {
		t.Fatalf("unexpected default targets command error: %v", err)
	}
}

func TestValidateTargetsReportAcceptsExpectedShape(t *testing.T) {
	raw := targetsReportJSON(defaultTargetsForTest(true))
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate targets: %v", err)
	}
}

func TestValidateTargetsReportAcceptsMissingWASIRunner(t *testing.T) {
	raw := targetsReportJSON(defaultTargetsForTest(false))
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate targets without WASI runner: %v", err)
	}
}

func TestValidateTargetsReportRejectsLinuxHostNativeMarkedUnsupported(t *testing.T) {
	targets := defaultTargetsForTest(true)
	targets[0].RunSupported = false
	targets[0].RunUnsupportedReason = "linux-x64 cannot run on host linux/amd64"
	raw := targetsReportJSON(targets)
	err := validateTargetsReport(raw)
	if err == nil {
		t.Fatalf("expected linux host-native run_supported=false failure")
	}
	if !strings.Contains(err.Error(), "linux-x64") ||
		!strings.Contains(err.Error(), "run_supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTargetsReportRejectsWrongOrder(t *testing.T) {
	raw := targetsReportJSONWithLists(
		[]string{"windows-x64", "linux-x64", "macos-x64", "wasm32-wasi", "wasm32-web"},
		[]string{},
		[]string{},
		[]targetReportEntry{},
	)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected wrong-order failure")
	}
}

func TestValidateRunContractAcceptsHostProbedBuildOnly(t *testing.T) {
	entry := targetReportEntry{
		Triple:       "linux-x32",
		Status:       "build_only",
		BuildOnly:    true,
		RunMode:      "host_probed",
		RunSupported: false,
		RunUnsupportedReason: ("host linux/amd64 does not support Linux x32 ABI execution; no " +
			"host fallback is allowed; probe command: tetra test --diagnostics=json -" +
			"-target x32 --format=json <runner-smoke.tetra>"),
	}
	if err := validateRunContract(entry); err != nil {
		t.Fatalf("validate host-probed run contract: %v", err)
	}
	entry.RunSupported = true
	entry.RunUnsupportedReason = ""
	if err := validateRunContract(entry); err != nil {
		t.Fatalf("validate supported host-probed run contract: %v", err)
	}
	entry.BuildOnly = false
	if err := validateRunContract(entry); err == nil ||
		!strings.Contains(err.Error(), "build-only") {
		t.Fatalf("expected non-build-only host-probed failure, got %v", err)
	}
}

func TestValidateRunContractRejectsHostProbedReasonWithoutProbeCommand(t *testing.T) {
	entry := targetReportEntry{
		Triple:       "linux-x32",
		Status:       "build_only",
		BuildOnly:    true,
		RunMode:      "host_probed",
		RunSupported: false,
		RunUnsupportedReason: ("host does not support Linux x32 ABI execution; no host fallback " +
			"is allowed"),
	}
	err := validateRunContract(entry)
	if err == nil || !strings.Contains(err.Error(), "probe command") {
		t.Fatalf("expected missing probe command failure, got %v", err)
	}
}

func TestValidateLinuxNativePromotionMetadataRejectsMissingEvidenceArtifact(t *testing.T) {
	entry := targetReportEntry{
		Triple:        "linux-x86",
		RuntimeStatus: "partial_build_only",
		StdlibStatus:  "partial_build_only",
		FFIStatus:     "ilp32_scalar_object_smokes_partial",
		RunnerProbeCommand: ("tetra test --diagnostics=json --target x86 --format=json " +
			"<runner-smoke.tetra>"),
		ReleaseGate:       "scripts/release/post_v0_4/linux-native-targets-smoke.sh",
		EvidenceArtifacts: []string{"targets.json"},
	}
	err := validateLinuxNativePromotionMetadata(entry)
	if err == nil || !strings.Contains(err.Error(), "linux-x86-abi.json") {
		t.Fatalf("expected missing linux-x86 ABI artifact failure, got %v", err)
	}
}

func TestValidateLinuxNativeSyscallMetadataRejectsCollapsedX32Numbering(t *testing.T) {
	entry := targetReportEntry{
		Triple:              "linux-x32",
		SyscallInstruction:  "syscall",
		SyscallNumbering:    "x86_64",
		SyscallArgRegisters: []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"},
		SyscallErrorRange:   "-4095..-1",
	}
	err := validateLinuxNativeSyscallMetadata(entry)
	if err == nil || !strings.Contains(err.Error(), "x32_syscall_bit") {
		t.Fatalf("expected x32 syscall numbering failure, got %v", err)
	}
}

func TestValidateMemoryCapabilityClaimsRejectsInflatedRuntimeClaim(t *testing.T) {
	entry := targetReportEntry{
		Triple:                   "linux-x86",
		Status:                   "build_only",
		BuildOnly:                true,
		RunSupported:             false,
		MemoryBuild:              "yes",
		MemoryLower:              "yes",
		MemoryRun:                "yes",
		MemoryRawDiagnostics:     "partial",
		MemoryRegionLowering:     "partial",
		MemoryAlignmentSemantics: "partial",
		MemoryClaimLevel:         "production/host_runtime",
	}
	err := validateMemoryCapabilityClaims(entry)
	if err == nil || !strings.Contains(err.Error(), "runtime memory claim") {
		t.Fatalf("expected runtime memory claim rejection, got %v", err)
	}
}

func TestValidateMemoryCapabilityClaimsRejectsProductionRuntimeClaimWithoutLinuxX64Evidence(
	t *testing.T,
) {
	entry := validMemoryCapabilityTargetForTest("linux-x64")
	entry.EvidenceArtifacts = []string{"linux-x64-abi.json"}

	err := validateMemoryCapabilityClaims(entry)
	if err == nil || !strings.Contains(err.Error(), "linux-x64 runner/artifact evidence") {
		t.Fatalf("expected linux-x64 runner/artifact evidence rejection, got %v", err)
	}
}

func TestValidateMemoryCapabilityClaimsRejectsHostRuntimeClaimWithoutTargetEvidence(t *testing.T) {
	for _, triple := range []string{"windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"} {
		t.Run(triple, func(t *testing.T) {
			entry := validMemoryCapabilityTargetForTest(triple)
			entry.MemoryRun = "yes"
			entry.MemoryClaimLevel = "production/host_runtime"

			err := validateMemoryCapabilityClaims(entry)
			if err == nil || !strings.Contains(err.Error(), "target-host/runner evidence") {
				t.Fatalf("expected target-host/runner evidence rejection, got %v", err)
			}
		})
	}
}

func TestValidateMemoryCapabilityClaimsRejectsRawDiagnosticsWithoutEvidence(t *testing.T) {
	entry := validMemoryCapabilityTargetForTest("linux-x86")
	entry.MemoryRawDiagnostics = "yes"
	err := validateMemoryCapabilityClaims(entry)
	if err == nil || !strings.Contains(err.Error(), "raw diagnostics") {
		t.Fatalf("expected raw diagnostics evidence rejection, got %v", err)
	}
}

func TestValidateMemoryCapabilityClaimsRejectsRegionLoweringWithoutArtifact(t *testing.T) {
	entry := validMemoryCapabilityTargetForTest("linux-x64")
	entry.EvidenceArtifacts = []string{"linux-x64-runner.json"}
	err := validateMemoryCapabilityClaims(entry)
	if err == nil || !strings.Contains(err.Error(), "lowered artifact") {
		t.Fatalf("expected region lowering artifact rejection, got %v", err)
	}
}

func TestValidateMemoryCapabilityClaimsRejectsAlignmentWithoutTargetABI(t *testing.T) {
	entry := validMemoryCapabilityTargetForTest("linux-x64")
	entry.ABI = ""
	err := validateMemoryCapabilityClaims(entry)
	if err == nil || !strings.Contains(err.Error(), "target-specific ABI") {
		t.Fatalf("expected alignment ABI evidence rejection, got %v", err)
	}
}

func targetsReportJSON(targets []targetReportEntry) []byte {
	return targetsReportJSONWithLists(
		[]string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"},
		[]string{"linux-x86", "linux-x32"},
		[]string{},
		targets,
	)
}

func targetsReportJSONWithLists(
	supported []string,
	buildOnly []string,
	planned []string,
	targets []targetReportEntry,
) []byte {
	return mustTargetJSON(targetsReport{
		Supported: supported,
		BuildOnly: buildOnly,
		Planned:   planned,
		Targets:   targets,
	})
}

func mustTargetJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func defaultTargetsForTest(wasiRunnerAvailable bool) []targetReportEntry {
	return []targetReportEntry{
		linuxX64TargetForTest(),
		hostRequiredTargetForTest("windows-x64"),
		hostRequiredTargetForTest("macos-x64"),
		wasiTargetForTest(wasiRunnerAvailable),
		webTargetForTest(),
		linuxBuildOnlyTargetForTest("linux-x86"),
		linuxBuildOnlyTargetForTest("linux-x32"),
	}
}

func linuxX64TargetForTest() targetReportEntry {
	entry := validMemoryCapabilityTargetForTest("linux-x64")
	entry.Format = "elf"
	entry.RunMode = "host_native"
	entry.RunnerProbeCommand = "tetra test --target x64 --format=json <runner-smoke.tetra>"
	entry.ReleaseGate = "scripts/release/post_v0_4/linux-native-targets-smoke.sh"
	entry.EvidenceArtifacts = linuxNativeEvidenceArtifacts("linux-x64")
	entry.SyscallInstruction = "syscall"
	entry.SyscallNumbering = "x86_64"
	entry.SyscallArgRegisters = []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"}
	entry.SyscallErrorRange = "-4095..-1"
	entry.SupportsDebugInfo = true
	entry.SupportsReleaseOptimize = true
	return entry
}

func hostRequiredTargetForTest(triple string) targetReportEntry {
	entry := validMemoryCapabilityTargetForTest(triple)
	entry.Format = map[string]string{
		"windows-x64": "pe",
		"macos-x64":   "macho",
	}[triple]
	entry.ExeExt = map[string]string{"windows-x64": ".exe"}[triple]
	entry.RunUnsupportedReason = triple + " cannot run on host linux/amd64"
	entry.SupportsDebugInfo = true
	entry.SupportsReleaseOptimize = true
	return entry
}

func wasiTargetForTest(runnerAvailable bool) targetReportEntry {
	entry := validMemoryCapabilityTargetForTest("wasm32-wasi")
	entry.Format = "wasm"
	entry.ExeExt = ".wasm"
	entry.SupportsReleaseOptimize = true
	if runnerAvailable {
		entry.RunSupported = true
		entry.RunRunner = "wasmtime"
		entry.RunUnsupportedReason = ""
	} else {
		entry.RunUnsupportedReason = "cannot run target wasm32-wasi: missing WASI runner"
		entry.RunUnsupportedReason += ": need wasmtime or node"
	}
	return entry
}

func webTargetForTest() targetReportEntry {
	entry := validMemoryCapabilityTargetForTest("wasm32-web")
	entry.Format = "wasm"
	entry.ExeExt = ".wasm"
	entry.RunUnsupportedReason = "web runner unavailable: chromium-compatible executable not found"
	entry.SupportsReleaseOptimize = true
	return entry
}

func linuxBuildOnlyTargetForTest(triple string) targetReportEntry {
	entry := validMemoryCapabilityTargetForTest(triple)
	entry.Format = "elf"
	entry.RunMode = "host_probed"
	targetArg := map[string]string{"linux-x86": "x86", "linux-x32": "x32"}[triple]
	entry.RunnerProbeCommand = "tetra test --diagnostics=json --target " + targetArg
	entry.RunnerProbeCommand += " --format=json <runner-smoke.tetra>"
	entry.RunUnsupportedReason = "host linux/amd64 does not support "
	if triple == "linux-x86" {
		entry.RunUnsupportedReason += "Linux i386 execution"
	} else {
		entry.RunUnsupportedReason += "Linux x32 ABI execution"
	}
	entry.RunUnsupportedReason += "; no host fallback is allowed; probe command: "
	entry.RunUnsupportedReason += entry.RunnerProbeCommand
	entry.ReleaseGate = "scripts/release/post_v0_4/linux-native-targets-smoke.sh"
	entry.EvidenceArtifacts = linuxNativeEvidenceArtifacts(triple)
	if triple == "linux-x86" {
		entry.SyscallInstruction = "int 0x80"
		entry.SyscallNumbering = "i386"
		entry.SyscallArgRegisters = []string{"eax", "ebx", "ecx", "edx", "esi", "edi", "ebp"}
	} else {
		entry.SyscallInstruction = "syscall"
		entry.SyscallNumbering = "x32_syscall_bit"
		entry.SyscallArgRegisters = []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"}
	}
	entry.SyscallErrorRange = "-4095..-1"
	return entry
}

func linuxNativeEvidenceArtifacts(triple string) []string {
	return []string{
		"targets.json",
		triple + "-abi.json",
		triple + "-atomic-stress.json",
		triple + "-fuzz.json",
		triple + "-runner.json",
		"linux-native-targets-brutal.json",
		"artifact-hashes.json",
	}
}

func validMemoryCapabilityTargetForTest(triple string) targetReportEntry {
	switch triple {
	case "linux-x64":
		return targetReportEntry{
			Triple:                   "linux-x64",
			Status:                   "supported",
			OS:                       "linux",
			Arch:                     "x64",
			ABI:                      "sysv",
			DataModel:                "lp64",
			BuildOnly:                false,
			RunSupported:             true,
			RuntimeStatus:            "production",
			StdlibStatus:             "production",
			FFIStatus:                "scalar_object_smokes_partial",
			EvidenceArtifacts:        []string{"linux-x64-abi.json", "linux-x64-runner.json"},
			MemoryBuild:              "yes",
			MemoryLower:              "yes",
			MemoryRun:                "yes",
			MemoryRawDiagnostics:     "yes",
			MemoryRegionLowering:     "yes/partial",
			MemoryAlignmentSemantics: "yes",
			MemoryClaimLevel:         "production/host_runtime",
		}
	case "linux-x86":
		return targetReportEntry{
			Triple:                   "linux-x86",
			Status:                   "build_only",
			OS:                       "linux",
			Arch:                     "x86",
			ABI:                      "i386-sysv",
			DataModel:                "ilp32",
			BuildOnly:                true,
			RunSupported:             false,
			RuntimeStatus:            "partial_build_only",
			StdlibStatus:             "partial_build_only",
			FFIStatus:                "ilp32_scalar_object_smokes_partial",
			EvidenceArtifacts:        []string{"linux-x86-abi.json", "linux-x86-runner.json"},
			MemoryBuild:              "yes",
			MemoryLower:              "yes",
			MemoryRun:                "no/host-dependent",
			MemoryRawDiagnostics:     "partial",
			MemoryRegionLowering:     "partial",
			MemoryAlignmentSemantics: "partial",
			MemoryClaimLevel:         "build_lower_only",
		}
	case "linux-x32":
		return targetReportEntry{
			Triple:                   "linux-x32",
			Status:                   "build_only",
			OS:                       "linux",
			Arch:                     "x64",
			ABI:                      "x32-sysv",
			DataModel:                "x32",
			BuildOnly:                true,
			RunSupported:             false,
			RuntimeStatus:            "partial_build_only",
			StdlibStatus:             "partial_build_only",
			FFIStatus:                "ilp32_scalar_object_smokes_partial",
			EvidenceArtifacts:        []string{"linux-x32-abi.json", "linux-x32-runner.json"},
			MemoryBuild:              "yes",
			MemoryLower:              "yes",
			MemoryRun:                "no/host-dependent",
			MemoryRawDiagnostics:     "partial",
			MemoryRegionLowering:     "partial",
			MemoryAlignmentSemantics: "special",
			MemoryClaimLevel:         "build_lower_only",
		}
	case "windows-x64":
		return targetReportEntry{
			Triple:                   "windows-x64",
			Status:                   "supported",
			OS:                       "windows",
			Arch:                     "x64",
			ABI:                      "win64",
			DataModel:                "llp64",
			BuildOnly:                false,
			RunMode:                  "host_native",
			RunSupported:             false,
			MemoryBuild:              "yes",
			MemoryLower:              "yes",
			MemoryRun:                "host-required",
			MemoryRawDiagnostics:     "host-required",
			MemoryRegionLowering:     "host-required",
			MemoryAlignmentSemantics: "host-required",
			MemoryClaimLevel:         "build_lower_only unless run",
		}
	case "macos-x64":
		return targetReportEntry{
			Triple:                   "macos-x64",
			Status:                   "supported",
			OS:                       "macos",
			Arch:                     "x64",
			ABI:                      "sysv",
			DataModel:                "lp64",
			BuildOnly:                false,
			RunMode:                  "host_native",
			RunSupported:             false,
			MemoryBuild:              "yes",
			MemoryLower:              "yes",
			MemoryRun:                "host-required",
			MemoryRawDiagnostics:     "host-required",
			MemoryRegionLowering:     "host-required",
			MemoryAlignmentSemantics: "host-required",
			MemoryClaimLevel:         "build_lower_only unless run",
		}
	case "wasm32-wasi":
		return targetReportEntry{
			Triple:                   "wasm32-wasi",
			Status:                   "supported",
			OS:                       "wasi",
			Arch:                     "wasm32",
			ABI:                      "wasi",
			DataModel:                "ilp32",
			BuildOnly:                false,
			RunMode:                  "wasi_runner",
			RunSupported:             false,
			MemoryBuild:              "yes",
			MemoryLower:              "yes",
			MemoryRun:                "runner-smoke if available",
			MemoryRawDiagnostics:     "safe-only",
			MemoryRegionLowering:     "limited",
			MemoryAlignmentSemantics: "wasm rules",
			MemoryClaimLevel:         "artifact/runtime tiered",
		}
	case "wasm32-web":
		return targetReportEntry{
			Triple:                   "wasm32-web",
			Status:                   "supported",
			OS:                       "web",
			Arch:                     "wasm32",
			ABI:                      "web",
			DataModel:                "ilp32",
			BuildOnly:                false,
			RunMode:                  "web_runner",
			RunSupported:             false,
			MemoryBuild:              "yes",
			MemoryLower:              "yes",
			MemoryRun:                "browser-smoke if available",
			MemoryRawDiagnostics:     "safe-only",
			MemoryRegionLowering:     "limited",
			MemoryAlignmentSemantics: "wasm rules",
			MemoryClaimLevel:         "artifact/runtime tiered",
		}
	default:
		panic("unsupported test target " + triple)
	}
}

func TestValidateTargetsReportRejectsUnknownFields(t *testing.T) {
	raw := mustTargetJSON(map[string]any{
		"supported":  []string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi"},
		"build_only": []string{},
		"planned":    []string{},
		"targets":    []targetReportEntry{},
		"extra":      true,
	})
	if err := validateTargetsReport(raw); err == nil ||
		!strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown top-level field failure, got %v", err)
	}
	raw = mustTargetJSON(map[string]any{
		"supported":  []string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi"},
		"build_only": []string{},
		"planned":    []string{},
		"targets": []map[string]any{{
			"triple":                    "linux-x64",
			"status":                    "supported",
			"os":                        "linux",
			"arch":                      "x64",
			"abi":                       "sysv",
			"format":                    "elf",
			"build_only":                false,
			"run_supported":             true,
			"supports_debug_info":       true,
			"supports_release_optimize": true,
			"extra":                     true,
		}},
	})
	if err := validateTargetsReport(raw); err == nil ||
		!strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown nested field failure, got %v", err)
	}
}

func TestValidateTargetsReportRejectsDuplicate(t *testing.T) {
	if err := validateTargetList(
		"supported",
		[]string{"linux-x64", "linux-x64"},
		[]string{"linux-x64", "linux-x64"},
	); err == nil {
		t.Fatalf("expected duplicate failure")
	}
}

func TestValidateTargetsReportRejectsMissingMetadata(t *testing.T) {
	raw := mustTargetJSON(map[string]any{
		"supported":  []string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi"},
		"build_only": []string{},
		"planned":    []string{},
	})
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected missing metadata failure")
	}
}

func TestValidateTargetsReportRejectsWrongABI(t *testing.T) {
	targets := defaultTargetsForTest(false)
	targets = targets[:5]
	targets[0].ABI = "win64"
	raw := targetsReportJSONWithLists(
		[]string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"},
		[]string{},
		[]string{},
		targets,
	)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected wrong ABI failure")
	}
}

func TestValidateTargetsReportRejectsMissingStatus(t *testing.T) {
	targets := defaultTargetsForTest(false)
	targets = targets[:5]
	targets[0].Status = ""
	raw := targetsReportJSONWithLists(
		[]string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"},
		[]string{},
		[]string{},
		targets,
	)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected missing status failure")
	}
}
