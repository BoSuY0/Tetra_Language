package main

import (
	"strings"
	"testing"
)

func TestValidateTargetsReportAcceptsExpectedShape(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":["linux-x86","linux-x32"],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":true,"runtime_status":"production","stdlib_status":"production","ffi_status":"scalar_object_smokes_partial","memory_build":"yes","memory_lower":"yes","memory_run":"yes","memory_raw_diagnostics":"yes","memory_region_lowering":"yes/partial","memory_alignment_semantics":"yes","memory_claim_level":"production/host_runtime","runner_probe_command":"tetra test --target x64 --format=json <runner-smoke.tetra>","release_gate":"scripts/release/post_v0_4/linux-native-targets-smoke.sh","evidence_artifacts":["targets.json","linux-x64-abi.json","linux-x64-atomic-stress.json","linux-x64-fuzz.json","linux-x64-runner.json","linux-native-targets-brutal.json","artifact-hashes.json"],"syscall_instruction":"syscall","syscall_numbering":"x86_64","syscall_arg_registers":["rax","rdi","rsi","rdx","r10","r8","r9"],"syscall_error_range":"-4095..-1","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","memory_build":"yes","memory_lower":"yes","memory_run":"host-required","memory_raw_diagnostics":"host-required","memory_region_lowering":"host-required","memory_alignment_semantics":"host-required","memory_claim_level":"build_lower_only unless run","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","memory_build":"yes","memory_lower":"yes","memory_run":"host-required","memory_raw_diagnostics":"host-required","memory_region_lowering":"host-required","memory_alignment_semantics":"host-required","memory_claim_level":"build_lower_only unless run","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","memory_build":"yes","memory_lower":"yes","memory_run":"runner-smoke if available","memory_raw_diagnostics":"safe-only","memory_region_lowering":"limited","memory_alignment_semantics":"wasm rules","memory_claim_level":"artifact/runtime tiered","run_runner":"wasmtime","run_supported":true,"supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","memory_build":"yes","memory_lower":"yes","memory_run":"browser-smoke if available","memory_raw_diagnostics":"safe-only","memory_region_lowering":"limited","memory_alignment_semantics":"wasm rules","memory_claim_level":"artifact/runtime tiered","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"linux-x86","status":"build_only","os":"linux","arch":"x86","abi":"i386-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host linux/amd64 does not support Linux i386 execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x86 --format=json <runner-smoke.tetra>","runtime_status":"partial_build_only","stdlib_status":"partial_build_only","ffi_status":"ilp32_scalar_object_smokes_partial","memory_build":"yes","memory_lower":"yes","memory_run":"no/host-dependent","memory_raw_diagnostics":"partial","memory_region_lowering":"partial","memory_alignment_semantics":"partial","memory_claim_level":"build_lower_only","runner_probe_command":"tetra test --diagnostics=json --target x86 --format=json <runner-smoke.tetra>","release_gate":"scripts/release/post_v0_4/linux-native-targets-smoke.sh","evidence_artifacts":["targets.json","linux-x86-abi.json","linux-x86-atomic-stress.json","linux-x86-fuzz.json","linux-x86-runner.json","linux-native-targets-brutal.json","artifact-hashes.json"],"syscall_instruction":"int 0x80","syscall_numbering":"i386","syscall_arg_registers":["eax","ebx","ecx","edx","esi","edi","ebp"],"syscall_error_range":"-4095..-1","supports_debug_info":false,"supports_release_optimize":false},
    {"triple":"linux-x32","status":"build_only","os":"linux","arch":"x64","abi":"x32-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host linux/amd64 does not support Linux x32 ABI execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>","runtime_status":"partial_build_only","stdlib_status":"partial_build_only","ffi_status":"ilp32_scalar_object_smokes_partial","memory_build":"yes","memory_lower":"yes","memory_run":"no/host-dependent","memory_raw_diagnostics":"partial","memory_region_lowering":"partial","memory_alignment_semantics":"special","memory_claim_level":"build_lower_only","runner_probe_command":"tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>","release_gate":"scripts/release/post_v0_4/linux-native-targets-smoke.sh","evidence_artifacts":["targets.json","linux-x32-abi.json","linux-x32-atomic-stress.json","linux-x32-fuzz.json","linux-x32-runner.json","linux-native-targets-brutal.json","artifact-hashes.json"],"syscall_instruction":"syscall","syscall_numbering":"x32_syscall_bit","syscall_arg_registers":["rax","rdi","rsi","rdx","r10","r8","r9"],"syscall_error_range":"-4095..-1","supports_debug_info":false,"supports_release_optimize":false}
  ]
}`)
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate targets: %v", err)
	}
}

func TestValidateTargetsReportAcceptsMissingWASIRunner(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":["linux-x86","linux-x32"],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":true,"runtime_status":"production","stdlib_status":"production","ffi_status":"scalar_object_smokes_partial","memory_build":"yes","memory_lower":"yes","memory_run":"yes","memory_raw_diagnostics":"yes","memory_region_lowering":"yes/partial","memory_alignment_semantics":"yes","memory_claim_level":"production/host_runtime","runner_probe_command":"tetra test --target x64 --format=json <runner-smoke.tetra>","release_gate":"scripts/release/post_v0_4/linux-native-targets-smoke.sh","evidence_artifacts":["targets.json","linux-x64-abi.json","linux-x64-atomic-stress.json","linux-x64-fuzz.json","linux-x64-runner.json","linux-native-targets-brutal.json","artifact-hashes.json"],"syscall_instruction":"syscall","syscall_numbering":"x86_64","syscall_arg_registers":["rax","rdi","rsi","rdx","r10","r8","r9"],"syscall_error_range":"-4095..-1","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","memory_build":"yes","memory_lower":"yes","memory_run":"host-required","memory_raw_diagnostics":"host-required","memory_region_lowering":"host-required","memory_alignment_semantics":"host-required","memory_claim_level":"build_lower_only unless run","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","memory_build":"yes","memory_lower":"yes","memory_run":"host-required","memory_raw_diagnostics":"host-required","memory_region_lowering":"host-required","memory_alignment_semantics":"host-required","memory_claim_level":"build_lower_only unless run","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","memory_build":"yes","memory_lower":"yes","memory_run":"runner-smoke if available","memory_raw_diagnostics":"safe-only","memory_region_lowering":"limited","memory_alignment_semantics":"wasm rules","memory_claim_level":"artifact/runtime tiered","run_supported":false,"run_unsupported_reason":"cannot run target wasm32-wasi: missing WASI runner: need wasmtime or node","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","memory_build":"yes","memory_lower":"yes","memory_run":"browser-smoke if available","memory_raw_diagnostics":"safe-only","memory_region_lowering":"limited","memory_alignment_semantics":"wasm rules","memory_claim_level":"artifact/runtime tiered","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"linux-x86","status":"build_only","os":"linux","arch":"x86","abi":"i386-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host linux/amd64 does not support Linux i386 execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x86 --format=json <runner-smoke.tetra>","runtime_status":"partial_build_only","stdlib_status":"partial_build_only","ffi_status":"ilp32_scalar_object_smokes_partial","memory_build":"yes","memory_lower":"yes","memory_run":"no/host-dependent","memory_raw_diagnostics":"partial","memory_region_lowering":"partial","memory_alignment_semantics":"partial","memory_claim_level":"build_lower_only","runner_probe_command":"tetra test --diagnostics=json --target x86 --format=json <runner-smoke.tetra>","release_gate":"scripts/release/post_v0_4/linux-native-targets-smoke.sh","evidence_artifacts":["targets.json","linux-x86-abi.json","linux-x86-atomic-stress.json","linux-x86-fuzz.json","linux-x86-runner.json","linux-native-targets-brutal.json","artifact-hashes.json"],"syscall_instruction":"int 0x80","syscall_numbering":"i386","syscall_arg_registers":["eax","ebx","ecx","edx","esi","edi","ebp"],"syscall_error_range":"-4095..-1","supports_debug_info":false,"supports_release_optimize":false},
    {"triple":"linux-x32","status":"build_only","os":"linux","arch":"x64","abi":"x32-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host linux/amd64 does not support Linux x32 ABI execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>","runtime_status":"partial_build_only","stdlib_status":"partial_build_only","ffi_status":"ilp32_scalar_object_smokes_partial","memory_build":"yes","memory_lower":"yes","memory_run":"no/host-dependent","memory_raw_diagnostics":"partial","memory_region_lowering":"partial","memory_alignment_semantics":"special","memory_claim_level":"build_lower_only","runner_probe_command":"tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>","release_gate":"scripts/release/post_v0_4/linux-native-targets-smoke.sh","evidence_artifacts":["targets.json","linux-x32-abi.json","linux-x32-atomic-stress.json","linux-x32-fuzz.json","linux-x32-runner.json","linux-native-targets-brutal.json","artifact-hashes.json"],"syscall_instruction":"syscall","syscall_numbering":"x32_syscall_bit","syscall_arg_registers":["rax","rdi","rsi","rdx","r10","r8","r9"],"syscall_error_range":"-4095..-1","supports_debug_info":false,"supports_release_optimize":false}
  ]
}`)
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate targets without WASI runner: %v", err)
	}
}

func TestValidateTargetsReportRejectsLinuxHostNativeMarkedUnsupported(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":["linux-x86","linux-x32"],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"linux-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","memory_build":"yes","memory_lower":"yes","memory_run":"host-required","memory_raw_diagnostics":"host-required","memory_region_lowering":"host-required","memory_alignment_semantics":"host-required","memory_claim_level":"build_lower_only unless run","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","memory_build":"yes","memory_lower":"yes","memory_run":"host-required","memory_raw_diagnostics":"host-required","memory_region_lowering":"host-required","memory_alignment_semantics":"host-required","memory_claim_level":"build_lower_only unless run","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","memory_build":"yes","memory_lower":"yes","memory_run":"runner-smoke if available","memory_raw_diagnostics":"safe-only","memory_region_lowering":"limited","memory_alignment_semantics":"wasm rules","memory_claim_level":"artifact/runtime tiered","run_runner":"wasmtime","run_supported":true,"supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","memory_build":"yes","memory_lower":"yes","memory_run":"browser-smoke if available","memory_raw_diagnostics":"safe-only","memory_region_lowering":"limited","memory_alignment_semantics":"wasm rules","memory_claim_level":"artifact/runtime tiered","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"linux-x86","status":"build_only","os":"linux","arch":"x86","abi":"i386-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host does not support Linux i386 execution; no host fallback is allowed","supports_debug_info":false,"supports_release_optimize":false},
    {"triple":"linux-x32","status":"build_only","os":"linux","arch":"x64","abi":"x32-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host does not support Linux x32 ABI execution; no host fallback is allowed","supports_debug_info":false,"supports_release_optimize":false}
  ]
}`)
	err := validateTargetsReport(raw)
	if err == nil {
		t.Fatalf("expected linux host-native run_supported=false failure")
	}
	if !strings.Contains(err.Error(), "linux-x64") || !strings.Contains(err.Error(), "run_supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTargetsReportRejectsWrongOrder(t *testing.T) {
	raw := []byte(`{"supported":["windows-x64","linux-x64","macos-x64","wasm32-wasi","wasm32-web"],"build_only":[],"planned":[],"targets":[]}`)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected wrong-order failure")
	}
}

func TestValidateRunContractAcceptsHostProbedBuildOnly(t *testing.T) {
	entry := targetReportEntry{
		Triple:               "linux-x32",
		Status:               "build_only",
		BuildOnly:            true,
		RunMode:              "host_probed",
		RunSupported:         false,
		RunUnsupportedReason: "host linux/amd64 does not support Linux x32 ABI execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>",
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
	if err := validateRunContract(entry); err == nil || !strings.Contains(err.Error(), "build-only") {
		t.Fatalf("expected non-build-only host-probed failure, got %v", err)
	}
}

func TestValidateRunContractRejectsHostProbedReasonWithoutProbeCommand(t *testing.T) {
	entry := targetReportEntry{
		Triple:               "linux-x32",
		Status:               "build_only",
		BuildOnly:            true,
		RunMode:              "host_probed",
		RunSupported:         false,
		RunUnsupportedReason: "host does not support Linux x32 ABI execution; no host fallback is allowed",
	}
	err := validateRunContract(entry)
	if err == nil || !strings.Contains(err.Error(), "probe command") {
		t.Fatalf("expected missing probe command failure, got %v", err)
	}
}

func TestValidateLinuxNativePromotionMetadataRejectsMissingEvidenceArtifact(t *testing.T) {
	entry := targetReportEntry{
		Triple:             "linux-x86",
		RuntimeStatus:      "partial_build_only",
		StdlibStatus:       "partial_build_only",
		FFIStatus:          "ilp32_scalar_object_smokes_partial",
		RunnerProbeCommand: "tetra test --diagnostics=json --target x86 --format=json <runner-smoke.tetra>",
		ReleaseGate:        "scripts/release/post_v0_4/linux-native-targets-smoke.sh",
		EvidenceArtifacts:  []string{"targets.json"},
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
			EvidenceArtifacts:        []string{"linux-x86-abi.json", "linux-x86-runner.json"},
			MemoryBuild:              "yes",
			MemoryLower:              "yes",
			MemoryRun:                "no/host-dependent",
			MemoryRawDiagnostics:     "partial",
			MemoryRegionLowering:     "partial",
			MemoryAlignmentSemantics: "partial",
			MemoryClaimLevel:         "build_lower_only",
		}
	default:
		panic("unsupported test target " + triple)
	}
}

func TestValidateTargetsReportRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],"build_only":[],"planned":[],"targets":[],"extra":true}`)
	if err := validateTargetsReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown top-level field failure, got %v", err)
	}
	raw = []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":[],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_supported":true,"supports_debug_info":true,"supports_release_optimize":true,"extra":true}
  ]
}`)
	if err := validateTargetsReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown nested field failure, got %v", err)
	}
}

func TestValidateTargetsReportRejectsDuplicate(t *testing.T) {
	if err := validateTargetList("supported", []string{"linux-x64", "linux-x64"}, []string{"linux-x64", "linux-x64"}); err == nil {
		t.Fatalf("expected duplicate failure")
	}
}

func TestValidateTargetsReportRejectsMissingMetadata(t *testing.T) {
	raw := []byte(`{"supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],"build_only":[],"planned":[]}`)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected missing metadata failure")
	}
}

func TestValidateTargetsReportRejectsWrongABI(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":[],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"win64","format":"elf","exe_ext":"","build_only":false,"run_supported":true,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_supported":false,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_supported":false,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_supported":false,"run_unsupported_reason":"cannot run target wasm32-wasi: missing WASI runner: need wasmtime or node","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true}
  ]
}`)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected wrong ABI failure")
	}
}

func TestValidateTargetsReportRejectsMissingStatus(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":[],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_supported":true,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_supported":false,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_supported":false,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_supported":false,"run_unsupported_reason":"cannot run target wasm32-wasi: missing WASI runner: need wasmtime or node","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true}
  ]
}`)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected missing status failure")
	}
}
