package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type targetsReport struct {
	Supported []string            `json:"supported"`
	BuildOnly []string            `json:"build_only"`
	Planned   []string            `json:"planned"`
	Targets   []targetReportEntry `json:"targets"`
}

type targetReportEntry struct {
	Triple                   string   `json:"triple"`
	Status                   string   `json:"status"`
	OS                       string   `json:"os"`
	Arch                     string   `json:"arch"`
	ABI                      string   `json:"abi"`
	DataModel                string   `json:"data_model"`
	Format                   string   `json:"format"`
	ExeExt                   string   `json:"exe_ext"`
	BuildOnly                bool     `json:"build_only"`
	RunMode                  string   `json:"run_mode"`
	RunRunner                string   `json:"run_runner,omitempty"`
	RunSupported             bool     `json:"run_supported"`
	RunUnsupportedReason     string   `json:"run_unsupported_reason,omitempty"`
	UIRuntimeContract        string   `json:"ui_runtime_contract,omitempty"`
	UIRuntimeStatus          string   `json:"ui_runtime_status,omitempty"`
	UIRuntimeEvidence        string   `json:"ui_runtime_evidence,omitempty"`
	PointerWidthBits         int      `json:"pointer_width_bits"`
	RegisterWidthBits        int      `json:"register_width_bits"`
	NativeIntWidthBits       int      `json:"native_int_width_bits"`
	Endian                   string   `json:"endian"`
	StackAlignmentBytes      int      `json:"stack_alignment_bytes"`
	MaxAtomicWidthBits       int      `json:"max_atomic_width_bits"`
	AtomicWidthBits          []int    `json:"atomic_width_bits"`
	AtomicPointerWidthBits   int      `json:"atomic_pointer_width_bits"`
	UnsupportedReason        string   `json:"unsupported_reason,omitempty"`
	RuntimeStatus            string   `json:"runtime_status,omitempty"`
	StdlibStatus             string   `json:"stdlib_status,omitempty"`
	FFIStatus                string   `json:"ffi_status,omitempty"`
	MemoryBuild              string   `json:"memory_build"`
	MemoryLower              string   `json:"memory_lower"`
	MemoryRun                string   `json:"memory_run"`
	MemoryRawDiagnostics     string   `json:"memory_raw_diagnostics"`
	MemoryRegionLowering     string   `json:"memory_region_lowering"`
	MemoryAlignmentSemantics string   `json:"memory_alignment_semantics"`
	MemoryClaimLevel         string   `json:"memory_claim_level"`
	RunnerProbeCommand       string   `json:"runner_probe_command,omitempty"`
	ReleaseGate              string   `json:"release_gate,omitempty"`
	EvidenceArtifacts        []string `json:"evidence_artifacts,omitempty"`
	SyscallInstruction       string   `json:"syscall_instruction,omitempty"`
	SyscallNumbering         string   `json:"syscall_numbering,omitempty"`
	SyscallArgRegisters      []string `json:"syscall_arg_registers,omitempty"`
	SyscallErrorRange        string   `json:"syscall_error_range,omitempty"`
	SupportsDebugInfo        bool     `json:"supports_debug_info"`
	SupportsReleaseOptimize  bool     `json:"supports_release_optimize"`
}

func main() {
	var path string
	flag.StringVar(&path, "report", "", "path to tetra targets --format=json output")
	flag.Parse()
	var raw []byte
	var err error
	if path == "" {
		raw, err = exec.Command("./tetra", "targets", "--format=json").Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to run ./tetra targets --format=json: %v\n", err)
			os.Exit(1)
		}
	} else {
		raw, err = os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := validateTargetsReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateTargetsReport(raw []byte) error {
	var report targetsReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return fmt.Errorf("invalid targets JSON: %w", err)
	}
	if err := validateTargetList("supported", report.Supported, []string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"}); err != nil {
		return err
	}
	if err := validateTargetList("build_only", report.BuildOnly, []string{"linux-x86", "linux-x32"}); err != nil {
		return err
	}
	if err := validateTargetList("planned", report.Planned, []string{}); err != nil {
		return err
	}
	if err := validateTargetMetadata(report.Targets); err != nil {
		return err
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func validateTargetList(name string, got []string, want []string) error {
	if len(got) != len(want) {
		return fmt.Errorf("%s target count = %d, want %d", name, len(got), len(want))
	}
	seen := map[string]bool{}
	for i, target := range got {
		if target != want[i] {
			return fmt.Errorf("%s target[%d] = %q, want %q", name, i, target, want[i])
		}
		if seen[target] {
			return fmt.Errorf("%s target %q is duplicated", name, target)
		}
		seen[target] = true
	}
	return nil
}

func validateTargetMetadata(got []targetReportEntry) error {
	want := []targetReportEntry{
		{Triple: "linux-x64", Status: "supported", OS: "linux", Arch: "x64", ABI: "sysv", Format: "elf", ExeExt: "", BuildOnly: false, RunMode: "host_native", SupportsDebugInfo: true, SupportsReleaseOptimize: true},
		{Triple: "windows-x64", Status: "supported", OS: "windows", Arch: "x64", ABI: "win64", Format: "pe", ExeExt: ".exe", BuildOnly: false, RunMode: "host_native", SupportsDebugInfo: true, SupportsReleaseOptimize: true},
		{Triple: "macos-x64", Status: "supported", OS: "macos", Arch: "x64", ABI: "sysv", Format: "macho", ExeExt: "", BuildOnly: false, RunMode: "host_native", SupportsDebugInfo: true, SupportsReleaseOptimize: true},
		{Triple: "wasm32-wasi", Status: "supported", OS: "wasi", Arch: "wasm32", ABI: "wasi", Format: "wasm", ExeExt: ".wasm", BuildOnly: false, RunMode: "wasi_runner", SupportsDebugInfo: false, SupportsReleaseOptimize: true},
		{Triple: "wasm32-web", Status: "supported", OS: "web", Arch: "wasm32", ABI: "web", Format: "wasm", ExeExt: ".wasm", BuildOnly: false, RunMode: "web_runner", SupportsDebugInfo: false, SupportsReleaseOptimize: true},
		{Triple: "linux-x86", Status: "build_only", OS: "linux", Arch: "x86", ABI: "i386-sysv", Format: "elf", ExeExt: "", BuildOnly: true, RunMode: "host_probed", SupportsDebugInfo: false, SupportsReleaseOptimize: false},
		{Triple: "linux-x32", Status: "build_only", OS: "linux", Arch: "x64", ABI: "x32-sysv", Format: "elf", ExeExt: "", BuildOnly: true, RunMode: "host_probed", SupportsDebugInfo: false, SupportsReleaseOptimize: false},
	}
	if len(got) != len(want) {
		return fmt.Errorf("target metadata count = %d, want %d", len(got), len(want))
	}
	seen := map[string]bool{}
	for i := range want {
		if seen[got[i].Triple] {
			return fmt.Errorf("target metadata %q is duplicated", got[i].Triple)
		}
		seen[got[i].Triple] = true
		if got[i].Triple != want[i].Triple {
			return fmt.Errorf("target metadata[%d].triple = %q, want %q", i, got[i].Triple, want[i].Triple)
		}
		if got[i].Status != want[i].Status {
			return fmt.Errorf("target metadata[%s].status = %q, want %q", got[i].Triple, got[i].Status, want[i].Status)
		}
		if got[i].OS != want[i].OS || got[i].Arch != want[i].Arch || got[i].ABI != want[i].ABI || got[i].Format != want[i].Format {
			return fmt.Errorf("target metadata[%s] platform = os:%s arch:%s abi:%s format:%s, want os:%s arch:%s abi:%s format:%s",
				got[i].Triple, got[i].OS, got[i].Arch, got[i].ABI, got[i].Format, want[i].OS, want[i].Arch, want[i].ABI, want[i].Format)
		}
		if got[i].ExeExt != want[i].ExeExt {
			return fmt.Errorf("target metadata[%s].exe_ext = %q, want %q", got[i].Triple, got[i].ExeExt, want[i].ExeExt)
		}
		if got[i].BuildOnly != want[i].BuildOnly {
			return fmt.Errorf("target metadata[%s].build_only = %v, want %v", got[i].Triple, got[i].BuildOnly, want[i].BuildOnly)
		}
		if got[i].RunMode != want[i].RunMode {
			return fmt.Errorf("target metadata[%s].run_mode = %q, want %q", got[i].Triple, got[i].RunMode, want[i].RunMode)
		}
		if err := validateRunContract(got[i]); err != nil {
			return err
		}
		if !got[i].RunSupported && got[i].RunUnsupportedReason == "" {
			return fmt.Errorf("target metadata[%s].run_unsupported_reason is required when run_supported is false", got[i].Triple)
		}
		if got[i].SupportsDebugInfo != want[i].SupportsDebugInfo {
			return fmt.Errorf("target metadata[%s].supports_debug_info = %v, want %v", got[i].Triple, got[i].SupportsDebugInfo, want[i].SupportsDebugInfo)
		}
		if got[i].SupportsReleaseOptimize != want[i].SupportsReleaseOptimize {
			return fmt.Errorf("target metadata[%s].supports_release_optimize = %v, want %v", got[i].Triple, got[i].SupportsReleaseOptimize, want[i].SupportsReleaseOptimize)
		}
		if err := validateUIRuntimeMetadata(got[i]); err != nil {
			return err
		}
		if err := validateLinuxNativePromotionMetadata(got[i]); err != nil {
			return err
		}
		if err := validateLinuxNativeSyscallMetadata(got[i]); err != nil {
			return err
		}
		if err := validateMemoryCapabilityClaims(got[i]); err != nil {
			return err
		}
	}
	return nil
}

func validateLinuxNativeSyscallMetadata(entry targetReportEntry) error {
	type wantSyscall struct {
		instruction string
		numbering   string
		registers   []string
	}
	want, ok := map[string]wantSyscall{
		"linux-x64": {instruction: "syscall", numbering: "x86_64", registers: []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"}},
		"linux-x86": {instruction: "int 0x80", numbering: "i386", registers: []string{"eax", "ebx", "ecx", "edx", "esi", "edi", "ebp"}},
		"linux-x32": {instruction: "syscall", numbering: "x32_syscall_bit", registers: []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"}},
	}[entry.Triple]
	if !ok {
		return nil
	}
	if entry.SyscallInstruction != want.instruction {
		return fmt.Errorf("target metadata[%s].syscall_instruction = %q, want %q", entry.Triple, entry.SyscallInstruction, want.instruction)
	}
	if entry.SyscallNumbering != want.numbering {
		return fmt.Errorf("target metadata[%s].syscall_numbering = %q, want %q", entry.Triple, entry.SyscallNumbering, want.numbering)
	}
	if !sameStringSequence(entry.SyscallArgRegisters, want.registers) {
		return fmt.Errorf("target metadata[%s].syscall_arg_registers = %v, want %v", entry.Triple, entry.SyscallArgRegisters, want.registers)
	}
	if entry.SyscallErrorRange != "-4095..-1" {
		return fmt.Errorf("target metadata[%s].syscall_error_range = %q, want -4095..-1", entry.Triple, entry.SyscallErrorRange)
	}
	return nil
}

type memoryCapabilityExpectation struct {
	build              string
	lower              string
	run                string
	rawDiagnostics     string
	regionLowering     string
	alignmentSemantics string
	claimLevel         string
}

func expectedMemoryCapability(triple string) (memoryCapabilityExpectation, bool) {
	switch triple {
	case "linux-x64":
		return memoryCapabilityExpectation{"yes", "yes", "yes", "yes", "yes/partial", "yes", "production/host_runtime"}, true
	case "linux-x86":
		return memoryCapabilityExpectation{"yes", "yes", "no/host-dependent", "partial", "partial", "partial", "build_lower_only"}, true
	case "linux-x32":
		return memoryCapabilityExpectation{"yes", "yes", "no/host-dependent", "partial", "partial", "special", "build_lower_only"}, true
	case "macos-x64", "windows-x64":
		return memoryCapabilityExpectation{"yes", "yes", "host-required", "host-required", "host-required", "host-required", "build_lower_only unless run"}, true
	case "wasm32-wasi":
		return memoryCapabilityExpectation{"yes", "yes", "runner-smoke if available", "safe-only", "limited", "wasm rules", "artifact/runtime tiered"}, true
	case "wasm32-web":
		return memoryCapabilityExpectation{"yes", "yes", "browser-smoke if available", "safe-only", "limited", "wasm rules", "artifact/runtime tiered"}, true
	default:
		return memoryCapabilityExpectation{}, false
	}
}

func validateMemoryCapabilityClaims(entry targetReportEntry) error {
	want, ok := expectedMemoryCapability(entry.Triple)
	if !ok {
		return nil
	}
	if entry.BuildOnly && (entry.MemoryRun == "yes" || entry.MemoryClaimLevel == "production/host_runtime") {
		return fmt.Errorf("target metadata[%s] runtime memory claim requires target runtime evidence, but target is build-only", entry.Triple)
	}
	if entry.MemoryRawDiagnostics == "yes" && !hasRawDiagnosticsEvidence(entry) {
		return fmt.Errorf("target metadata[%s] raw diagnostics claim requires raw diagnostics evidence", entry.Triple)
	}
	if (entry.MemoryRegionLowering == "yes" || entry.MemoryRegionLowering == "yes/partial" || entry.MemoryRegionLowering == "partial") && !hasLoweredArtifactEvidence(entry) {
		return fmt.Errorf("target metadata[%s] region lowering claim requires lowered artifact evidence", entry.Triple)
	}
	if requiresTargetABIForAlignment(entry.MemoryAlignmentSemantics) && !hasTargetSpecificABIEvidence(entry) {
		return fmt.Errorf("target metadata[%s] alignment claim requires target-specific ABI evidence", entry.Triple)
	}
	if entry.MemoryBuild != want.build {
		return fmt.Errorf("target metadata[%s].memory_build = %q, want %q", entry.Triple, entry.MemoryBuild, want.build)
	}
	if entry.MemoryLower != want.lower {
		return fmt.Errorf("target metadata[%s].memory_lower = %q, want %q", entry.Triple, entry.MemoryLower, want.lower)
	}
	if entry.MemoryRun != want.run {
		return fmt.Errorf("target metadata[%s].memory_run = %q, want %q", entry.Triple, entry.MemoryRun, want.run)
	}
	if entry.MemoryRawDiagnostics != want.rawDiagnostics {
		return fmt.Errorf("target metadata[%s].memory_raw_diagnostics = %q, want %q", entry.Triple, entry.MemoryRawDiagnostics, want.rawDiagnostics)
	}
	if entry.MemoryRegionLowering != want.regionLowering {
		return fmt.Errorf("target metadata[%s].memory_region_lowering = %q, want %q", entry.Triple, entry.MemoryRegionLowering, want.regionLowering)
	}
	if entry.MemoryAlignmentSemantics != want.alignmentSemantics {
		return fmt.Errorf("target metadata[%s].memory_alignment_semantics = %q, want %q", entry.Triple, entry.MemoryAlignmentSemantics, want.alignmentSemantics)
	}
	if entry.MemoryClaimLevel != want.claimLevel {
		return fmt.Errorf("target metadata[%s].memory_claim_level = %q, want %q", entry.Triple, entry.MemoryClaimLevel, want.claimLevel)
	}
	return nil
}

func hasRawDiagnosticsEvidence(entry targetReportEntry) bool {
	return entry.Triple == "linux-x64" &&
		entry.RunSupported &&
		entry.RuntimeStatus == "production" &&
		containsString(entry.EvidenceArtifacts, entry.Triple+"-runner.json")
}

func hasLoweredArtifactEvidence(entry targetReportEntry) bool {
	switch entry.Triple {
	case "linux-x64", "linux-x86", "linux-x32":
		return containsString(entry.EvidenceArtifacts, entry.Triple+"-abi.json")
	default:
		return entry.MemoryRegionLowering == "host-required" || entry.MemoryRegionLowering == "limited"
	}
}

func requiresTargetABIForAlignment(kind string) bool {
	switch kind {
	case "yes", "partial", "special", "wasm rules":
		return true
	default:
		return false
	}
}

func hasTargetSpecificABIEvidence(entry targetReportEntry) bool {
	if entry.ABI == "" {
		return false
	}
	switch entry.MemoryAlignmentSemantics {
	case "yes":
		return entry.Triple == "linux-x64" && entry.ABI == "sysv"
	case "partial":
		return entry.Triple == "linux-x86" && entry.ABI == "i386-sysv"
	case "special":
		return entry.Triple == "linux-x32" && entry.ABI == "x32-sysv"
	case "wasm rules":
		return entry.Triple == "wasm32-wasi" || entry.Triple == "wasm32-web"
	default:
		return true
	}
}

func validateLinuxNativePromotionMetadata(entry targetReportEntry) error {
	wantRuntimeStatus := map[string]string{
		"linux-x64": "production",
		"linux-x86": "partial_build_only",
		"linux-x32": "partial_build_only",
	}
	wantStdlibStatus := map[string]string{
		"linux-x64": "production",
		"linux-x86": "partial_build_only",
		"linux-x32": "partial_build_only",
	}
	wantFFIStatus := map[string]string{
		"linux-x64": "scalar_object_smokes_partial",
		"linux-x86": "ilp32_scalar_object_smokes_partial",
		"linux-x32": "ilp32_scalar_object_smokes_partial",
	}
	wantRuntime, ok := wantRuntimeStatus[entry.Triple]
	if !ok {
		return nil
	}
	if entry.RuntimeStatus != wantRuntime {
		return fmt.Errorf("target metadata[%s].runtime_status = %q, want %q", entry.Triple, entry.RuntimeStatus, wantRuntime)
	}
	if entry.StdlibStatus != wantStdlibStatus[entry.Triple] {
		return fmt.Errorf("target metadata[%s].stdlib_status = %q, want %q", entry.Triple, entry.StdlibStatus, wantStdlibStatus[entry.Triple])
	}
	if entry.FFIStatus != wantFFIStatus[entry.Triple] {
		return fmt.Errorf("target metadata[%s].ffi_status = %q, want %q", entry.Triple, entry.FFIStatus, wantFFIStatus[entry.Triple])
	}
	if entry.ReleaseGate != "scripts/release/post_v0_4/linux-native-targets-smoke.sh" {
		return fmt.Errorf("target metadata[%s].release_gate = %q, want linux native smoke script", entry.Triple, entry.ReleaseGate)
	}
	if entry.RunnerProbeCommand == "" {
		return fmt.Errorf("target metadata[%s].runner_probe_command is required", entry.Triple)
	}
	if entry.Triple == "linux-x86" && !strings.Contains(entry.RunnerProbeCommand, "--target x86") {
		return fmt.Errorf("target metadata[%s].runner_probe_command = %q, want x86 target probe", entry.Triple, entry.RunnerProbeCommand)
	}
	if entry.Triple == "linux-x32" && !strings.Contains(entry.RunnerProbeCommand, "--target x32") {
		return fmt.Errorf("target metadata[%s].runner_probe_command = %q, want x32 target probe", entry.Triple, entry.RunnerProbeCommand)
	}
	for _, artifact := range []string{
		"targets.json",
		entry.Triple + "-abi.json",
		entry.Triple + "-atomic-stress.json",
		entry.Triple + "-fuzz.json",
		entry.Triple + "-runner.json",
		"linux-native-targets-brutal.json",
		"artifact-hashes.json",
	} {
		if !containsString(entry.EvidenceArtifacts, artifact) {
			return fmt.Errorf("target metadata[%s].evidence_artifacts missing %s", entry.Triple, artifact)
		}
	}
	return nil
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func sameStringSequence(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func validateUIRuntimeMetadata(entry targetReportEntry) error {
	if entry.UIRuntimeStatus == "" {
		return nil
	}
	wantStatus := map[string]string{
		"linux-x64":   "production",
		"windows-x64": "requires_target_host_evidence",
		"macos-x64":   "requires_target_host_evidence",
		"wasm32-web":  "production",
		"wasm32-wasi": "unsupported",
		"linux-x86":   "unsupported",
		"linux-x32":   "unsupported",
	}[entry.Triple]
	if entry.UIRuntimeStatus != wantStatus {
		return fmt.Errorf("target metadata[%s].ui_runtime_status = %q, want %q", entry.Triple, entry.UIRuntimeStatus, wantStatus)
	}
	if entry.UIRuntimeStatus == "production" || entry.UIRuntimeStatus == "requires_target_host_evidence" {
		if entry.UIRuntimeContract != "tetra.ui.platform.v1" {
			return fmt.Errorf("target metadata[%s].ui_runtime_contract = %q, want tetra.ui.platform.v1", entry.Triple, entry.UIRuntimeContract)
		}
		if strings.TrimSpace(entry.UIRuntimeEvidence) == "" {
			return fmt.Errorf("target metadata[%s].ui_runtime_evidence is required", entry.Triple)
		}
	}
	if (entry.Triple == "windows-x64" || entry.Triple == "macos-x64") && strings.Contains(entry.UIRuntimeStatus, "production") {
		return fmt.Errorf("target metadata[%s] must not mark UI runtime production without target-host evidence", entry.Triple)
	}
	return nil
}

func validateRunContract(entry targetReportEntry) error {
	switch entry.RunMode {
	case "host_native":
		if entry.RunRunner != "" {
			return fmt.Errorf("target metadata[%s].run_runner = %q, want empty for host_native", entry.Triple, entry.RunRunner)
		}
		hostTriple, hostOK := validatorHostTriple()
		if hostOK && entry.Triple == hostTriple {
			if !entry.RunSupported {
				return fmt.Errorf("target metadata[%s].run_supported = false, want true on host %s/%s", entry.Triple, runtime.GOOS, runtime.GOARCH)
			}
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must be empty on matching host", entry.Triple)
			}
		} else if entry.RunSupported {
			return fmt.Errorf("target metadata[%s].run_supported = true, want false on host %s/%s", entry.Triple, runtime.GOOS, runtime.GOARCH)
		}
	case "wasi_runner":
		if entry.BuildOnly || entry.Triple != "wasm32-wasi" {
			return fmt.Errorf("target metadata[%s].run_mode wasi_runner is only valid for supported wasm32-wasi target", entry.Triple)
		}
		if entry.RunSupported {
			if entry.RunRunner != "wasmtime" && entry.RunRunner != "node-wasi" {
				return fmt.Errorf("target metadata[%s].run_runner = %q, want wasmtime or node-wasi when run_supported is true", entry.Triple, entry.RunRunner)
			}
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must be empty when WASI runner is available", entry.Triple)
			}
		} else {
			if entry.RunRunner != "" {
				return fmt.Errorf("target metadata[%s].run_runner = %q, want empty when WASI runner is unavailable", entry.Triple, entry.RunRunner)
			}
			if !strings.Contains(entry.RunUnsupportedReason, "missing WASI runner") {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must explain missing WASI runner", entry.Triple)
			}
		}
	case "host_probed":
		if !entry.BuildOnly {
			return fmt.Errorf("target metadata[%s].run_mode host_probed is only valid for build-only native targets", entry.Triple)
		}
		if entry.RunRunner != "" {
			return fmt.Errorf("target metadata[%s].run_runner = %q, want empty for host_probed", entry.Triple, entry.RunRunner)
		}
		if entry.RunSupported {
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must be empty when host probe succeeds", entry.Triple)
			}
		} else if !strings.Contains(entry.RunUnsupportedReason, "no host fallback") {
			return fmt.Errorf("target metadata[%s].run_unsupported_reason must explain host probe failure and no host fallback", entry.Triple)
		} else if !strings.Contains(entry.RunUnsupportedReason, "host ") {
			return fmt.Errorf("target metadata[%s].run_unsupported_reason must include host identity", entry.Triple)
		} else if !strings.Contains(entry.RunUnsupportedReason, "probe command:") {
			return fmt.Errorf("target metadata[%s].run_unsupported_reason must include the runner probe command", entry.Triple)
		} else if entry.RunnerProbeCommand != "" && !strings.Contains(entry.RunUnsupportedReason, entry.RunnerProbeCommand) {
			return fmt.Errorf("target metadata[%s].run_unsupported_reason must include runner_probe_command %q", entry.Triple, entry.RunnerProbeCommand)
		}
	case "web_runner":
		if entry.Triple != "wasm32-web" || entry.BuildOnly {
			return fmt.Errorf("target metadata[%s].run_mode web_runner is only valid for supported wasm32-web target", entry.Triple)
		}
		if entry.RunSupported {
			if entry.RunRunner == "" {
				return fmt.Errorf("target metadata[%s].run_runner is required when web runner is available", entry.Triple)
			}
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must be empty when web runner is available", entry.Triple)
			}
		} else {
			if entry.RunRunner != "" {
				return fmt.Errorf("target metadata[%s].run_runner = %q, want empty when web runner is unavailable", entry.Triple, entry.RunRunner)
			}
			if !strings.Contains(entry.RunUnsupportedReason, "web runner unavailable") && !strings.Contains(entry.RunUnsupportedReason, "browser runner unavailable") {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must explain missing web runner", entry.Triple)
			}
		}
	default:
		return fmt.Errorf("target metadata[%s].run_mode = %q, want host_native, host_probed, wasi_runner, or web_runner", entry.Triple, entry.RunMode)
	}
	return nil
}

func validatorHostTriple() (string, bool) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return "linux-x64", true
	case "windows/amd64":
		return "windows-x64", true
	case "darwin/amd64":
		return "macos-x64", true
	default:
		return "", false
	}
}
