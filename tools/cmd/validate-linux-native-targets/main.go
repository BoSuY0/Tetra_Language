package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type validateOptions struct {
	TargetsReport          string
	BrutalReport           string
	ArtifactHashesManifest string
	TargetReports          []targetReportInput
	RunnerReports          []targetRunnerInput
}

type targetReportInput struct {
	Triple       string
	ABIReport    string
	AtomicReport string
	FuzzReport   string
}

type targetReportFlags []targetReportInput

type targetRunnerInput struct {
	Triple string
	Report string
}

type targetRunnerFlags []targetRunnerInput

func (f *targetReportFlags) String() string {
	if f == nil {
		return ""
	}
	parts := make([]string, 0, len(*f))
	for _, item := range *f {
		parts = append(parts, item.Triple+":"+item.ABIReport+":"+item.AtomicReport+":"+item.FuzzReport)
	}
	return strings.Join(parts, ",")
}

func (f *targetReportFlags) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 4 {
		return fmt.Errorf("--target must be TRIPLE:ABI_REPORT:ATOMIC_REPORT:FUZZ_REPORT")
	}
	*f = append(*f, targetReportInput{
		Triple:       canonicalLinuxTarget(parts[0]),
		ABIReport:    parts[1],
		AtomicReport: parts[2],
		FuzzReport:   parts[3],
	})
	return nil
}

func (f *targetRunnerFlags) String() string {
	if f == nil {
		return ""
	}
	parts := make([]string, 0, len(*f))
	for _, item := range *f {
		parts = append(parts, item.Triple+":"+item.Report)
	}
	return strings.Join(parts, ",")
}

func (f *targetRunnerFlags) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("--runner must be TRIPLE:REPORT")
	}
	*f = append(*f, targetRunnerInput{
		Triple: canonicalLinuxTarget(parts[0]),
		Report: parts[1],
	})
	return nil
}

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

type testRunnerReport struct {
	Total      int                    `json:"total"`
	Passed     int                    `json:"passed"`
	Failed     int                    `json:"failed"`
	Target     string                 `json:"target,omitempty"`
	DurationMS int64                  `json:"duration_ms"`
	Files      []testRunnerFileReport `json:"files"`
	Results    []testRunnerResult     `json:"results"`
}

type testRunnerFileReport struct {
	Filename   string `json:"filename"`
	Total      int    `json:"total"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	DurationMS int64  `json:"duration_ms"`
}

type testRunnerResult struct {
	Name         string `json:"name"`
	Filename     string `json:"filename"`
	Index        int    `json:"index"`
	FunctionName string `json:"function_name"`
	ExitCode     int    `json:"exit_code"`
	Passed       bool   `json:"passed"`
	DurationMS   int64  `json:"duration_ms"`
	Error        string `json:"error,omitempty"`
}

type diagnosticReport struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	Severity string `json:"severity"`
}

const targetRuntimeDiagnosticCode = "TETRA3003"
const artifactHashManifestSchema = "tetra.release-artifact-hashes.v1alpha1"

type artifactHashManifest struct {
	Schema    string                    `json:"schema"`
	Root      string                    `json:"root"`
	Artifacts []artifactHashManifestRow `json:"artifacts"`
}

type artifactHashManifestRow struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

func main() {
	var options validateOptions
	var targets targetReportFlags
	var runners targetRunnerFlags
	flag.StringVar(&options.TargetsReport, "targets", "", "path to tetra targets --format=json report")
	flag.StringVar(&options.BrutalReport, "brutal", "", "tetra test --all-targets --brutal --format=json or --report=json report; required when all Linux native family targets are provided")
	flag.StringVar(&options.ArtifactHashesManifest, "artifact-hashes", "", "path to artifact-hashes.json manifest for this Linux native evidence directory")
	flag.Var(&targets, "target", "target evidence as TRIPLE:ABI_REPORT:ATOMIC_REPORT:FUZZ_REPORT; repeatable")
	flag.Var(&runners, "runner", "runner evidence as TRIPLE:REPORT; repeatable; REPORT may be a passing test report or no-host-fallback diagnostic for build-only targets")
	flag.Parse()
	options.TargetReports = append(options.TargetReports, targets...)
	options.RunnerReports = append(options.RunnerReports, runners...)
	if err := validateLinuxNativeTargets(options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateLinuxNativeTargets(options validateOptions) error {
	var issues []string
	var inputIssues []string
	if strings.TrimSpace(options.TargetsReport) == "" {
		inputIssues = append(inputIssues, "--targets is required")
	}
	if len(options.TargetReports) == 0 {
		inputIssues = append(inputIssues, "at least one --target report set is required")
	}
	if len(inputIssues) > 0 {
		return errors.New(strings.Join(inputIssues, "; "))
	}
	if strings.TrimSpace(options.ArtifactHashesManifest) == "" {
		issues = append(issues, "--artifact-hashes is required")
	}

	targets, err := readTargetsReport(options.TargetsReport)
	if err != nil {
		return err
	}
	issues = append(issues, validateLinuxFamilyTopLevel(targets)...)
	byTriple := map[string]targetReportEntry{}
	for _, entry := range targets.Targets {
		if byTriple[entry.Triple].Triple != "" {
			issues = append(issues, fmt.Sprintf("duplicate target metadata for %s", entry.Triple))
		}
		byTriple[entry.Triple] = entry
	}
	seenInputs := map[string]bool{}
	for _, input := range options.TargetReports {
		triple := canonicalLinuxTarget(input.Triple)
		if !isLinuxNativeFamilyTarget(triple) {
			issues = append(issues, fmt.Sprintf("target %q is not in linux-x64/linux-x86/linux-x32 family", input.Triple))
			continue
		}
		if seenInputs[triple] {
			issues = append(issues, fmt.Sprintf("duplicate report set for %s", triple))
			continue
		}
		seenInputs[triple] = true
		entry, ok := byTriple[triple]
		if !ok {
			issues = append(issues, fmt.Sprintf("targets report missing %s", triple))
			continue
		}
		issues = append(issues, validateLinuxTargetMetadata(entry)...)
		issues = append(issues, validateSuiteReportForTarget(input.ABIReport, triple+" ABI report", requiredABINames(triple), triple)...)
		issues = append(issues, validateSuiteReportForTarget(input.AtomicReport, triple+" atomic report", requiredAtomicNames(triple), triple)...)
		issues = append(issues, validateSuiteReportForTarget(input.FuzzReport, triple+" fuzz report", requiredFuzzNames(triple), triple)...)
	}
	seenRunnerInputs := map[string]bool{}
	for _, input := range options.RunnerReports {
		triple := canonicalLinuxTarget(input.Triple)
		if !isLinuxNativeFamilyTarget(triple) {
			issues = append(issues, fmt.Sprintf("runner target %q is not in linux-x64/linux-x86/linux-x32 family", input.Triple))
			continue
		}
		if seenRunnerInputs[triple] {
			issues = append(issues, fmt.Sprintf("duplicate runner report for %s", triple))
			continue
		}
		seenRunnerInputs[triple] = true
		issues = append(issues, validateRunnerReport(input.Report, triple, byTriple[triple])...)
	}
	for triple := range seenInputs {
		if !seenRunnerInputs[triple] {
			issues = append(issues, fmt.Sprintf("%s runner report is required for target evidence", triple))
		}
	}
	for _, triple := range []string{"linux-x64", "linux-x86", "linux-x32"} {
		if entry, ok := byTriple[triple]; ok {
			issues = append(issues, validateLinuxTargetMetadata(entry)...)
		} else {
			issues = append(issues, fmt.Sprintf("targets report missing %s", triple))
		}
	}
	if fullLinuxFamilyEvidence(seenInputs) && strings.TrimSpace(options.BrutalReport) == "" {
		issues = append(issues, "linux native brutal report is required for full-family evidence")
	}
	if strings.TrimSpace(options.BrutalReport) != "" {
		issues = append(issues, validateSuiteReport(options.BrutalReport, "linux native brutal report", requiredBrutalNames())...)
		issues = append(issues, validateForbiddenSuiteResults(options.BrutalReport, "linux native brutal report", forbiddenSuiteResultNames("linux-x64"))...)
	}
	if strings.TrimSpace(options.ArtifactHashesManifest) != "" {
		issues = append(issues, validateArtifactHashManifest(options.ArtifactHashesManifest)...)
		issues = append(issues, validateArtifactHashManifestCoversEvidence(options.ArtifactHashesManifest, evidenceArtifactPaths(options))...)
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateRunnerReport(path string, triple string, entry targetReportEntry) []string {
	label := triple + " runner report"
	if strings.TrimSpace(path) == "" {
		return []string{label + " path is required"}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", label, err)}
	}
	issues := rejectPaperEvidence(raw, label)
	if isDiagnosticJSON(raw) {
		var diag diagnosticReport
		if err := decodeStrictJSON(raw, &diag); err != nil {
			return append(issues, fmt.Sprintf("%s: %v", label, err))
		}
		if entry.Triple != "" && entry.RunSupported {
			issues = append(issues, fmt.Sprintf("%s is a no-host diagnostic but %s metadata has run_supported=true", label, triple))
		}
		return append(issues, validateRunnerDiagnostic(diag, triple, label)...)
	}
	if entry.Triple != "" && entry.RunMode == "host_probed" && !entry.RunSupported {
		issues = append(issues, fmt.Sprintf("%s is a passing runner report but %s metadata has run_supported=false", label, triple))
	}
	return append(issues, validateSuiteReportForTarget(path, label, requiredRunnerNames(triple), triple)...)
}

func isDiagnosticJSON(raw []byte) bool {
	var probe struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	return strings.TrimSpace(probe.Code) != "" || strings.TrimSpace(probe.Message) != ""
}

func validateRunnerDiagnostic(diag diagnosticReport, triple string, label string) []string {
	var issues []string
	if triple == "linux-x64" {
		return []string{label + " cannot be a blocked diagnostic; linux-x64 must run as the baseline"}
	}
	if triple != "linux-x86" && triple != "linux-x32" {
		return []string{label + " diagnostic is only valid for linux-x86/linux-x32 host-probed targets"}
	}
	if diag.Code == "" {
		issues = append(issues, label+" diagnostic code is required")
	} else if diag.Code != targetRuntimeDiagnosticCode {
		issues = append(issues, fmt.Sprintf("%s diagnostic code = %q, want %s", label, diag.Code, targetRuntimeDiagnosticCode))
	}
	if diag.Severity != "error" {
		issues = append(issues, fmt.Sprintf("%s diagnostic severity = %q, want error", label, diag.Severity))
	}
	if !strings.Contains(diag.Message, triple) {
		issues = append(issues, fmt.Sprintf("%s diagnostic message must mention %s", label, triple))
	}
	if !strings.Contains(diag.Message, "no host fallback") {
		issues = append(issues, label+" diagnostic message must mention no host fallback")
	}
	if !strings.Contains(diag.Message, "host ") {
		issues = append(issues, label+" diagnostic message must include host identity")
	}
	if !strings.Contains(diag.Message, "probe command:") {
		issues = append(issues, label+" diagnostic message must include probe command")
	}
	if wantFlag := runnerProbeTargetFlag(triple); wantFlag != "" && !strings.Contains(diag.Message, wantFlag) {
		issues = append(issues, fmt.Sprintf("%s diagnostic message must include %s probe command", label, wantFlag))
	}
	return issues
}

func runnerProbeTargetFlag(triple string) string {
	switch triple {
	case "linux-x86":
		return "--target x86"
	case "linux-x32":
		return "--target x32"
	default:
		return ""
	}
}

func readTargetsReport(path string) (targetsReport, error) {
	var report targetsReport
	raw, err := os.ReadFile(path)
	if err != nil {
		return report, err
	}
	if err := decodeStrictJSON(raw, &report); err != nil {
		return report, fmt.Errorf("%s: %w", path, err)
	}
	return report, nil
}

func validateLinuxFamilyTopLevel(report targetsReport) []string {
	var issues []string
	if !containsString(report.Supported, "linux-x64") {
		issues = append(issues, "supported targets must include linux-x64")
	}
	for _, triple := range []string{"linux-x86", "linux-x32"} {
		if !containsString(report.BuildOnly, triple) {
			issues = append(issues, fmt.Sprintf("build_only targets must include %s until promotion evidence is real", triple))
		}
		if containsString(report.Supported, triple) {
			issues = append(issues, fmt.Sprintf("%s appears in supported targets without linux native promotion validator support; keep it build_only until runtime/stdlib/FFI/smoke evidence passes", triple))
		}
	}
	return issues
}

func validateLinuxTargetMetadata(entry targetReportEntry) []string {
	switch entry.Triple {
	case "linux-x64":
		issues := validateExpectedTarget(entry, expectedTarget{
			status: "supported", os: "linux", arch: "x64", abi: "sysv", dataModel: "lp64", format: "elf",
			buildOnly: false, runMode: "host_native", pointerBits: 64, registerBits: 64, nativeIntBits: 64,
			maxAtomicBits: 64, atomicBits: []int{8, 16, 32, 64}, atomicPointerBits: 64,
			debugInfo: true, releaseOptimize: true,
		})
		issues = append(issues, validateLinuxNativeMemoryCapabilityMetadata(entry)...)
		issues = append(issues, validateLinuxNativePromotionMetadata(entry)...)
		return append(issues, validateLinuxNativeSyscallMetadata(entry)...)
	case "linux-x86":
		issues := validateExpectedTarget(entry, expectedTarget{
			status: "build_only", os: "linux", arch: "x86", abi: "i386-sysv", dataModel: "ilp32", format: "elf",
			buildOnly: true, runMode: "host_probed", pointerBits: 32, registerBits: 32, nativeIntBits: 32,
			maxAtomicBits: 32, atomicBits: []int{8, 16, 32}, atomicPointerBits: 32,
			debugInfo: false, releaseOptimize: false,
		})
		issues = append(issues, validateLinuxNativeMemoryCapabilityMetadata(entry)...)
		issues = append(issues, validateLinuxNativePromotionMetadata(entry)...)
		issues = append(issues, validateLinuxNativeSyscallMetadata(entry)...)
		return append(issues, validateBuildOnlyLinuxReason(entry, "linux-x86")...)
	case "linux-x32":
		issues := validateExpectedTarget(entry, expectedTarget{
			status: "build_only", os: "linux", arch: "x64", abi: "x32-sysv", dataModel: "x32", format: "elf",
			buildOnly: true, runMode: "host_probed", pointerBits: 32, registerBits: 64, nativeIntBits: 32,
			maxAtomicBits: 64, atomicBits: []int{8, 16, 32, 64}, atomicPointerBits: 32,
			debugInfo: false, releaseOptimize: false,
		})
		issues = append(issues, validateLinuxNativeMemoryCapabilityMetadata(entry)...)
		issues = append(issues, validateLinuxNativePromotionMetadata(entry)...)
		issues = append(issues, validateLinuxNativeSyscallMetadata(entry)...)
		return append(issues, validateBuildOnlyLinuxReason(entry, "linux-x32")...)
	default:
		return nil
	}
}

type linuxNativeMemoryCapabilityExpectation struct {
	build              string
	lower              string
	run                string
	rawDiagnostics     string
	regionLowering     string
	alignmentSemantics string
	claimLevel         string
}

func validateLinuxNativeMemoryCapabilityMetadata(entry targetReportEntry) []string {
	want := map[string]linuxNativeMemoryCapabilityExpectation{
		"linux-x64": {"yes", "yes", "yes", "yes", "yes/partial", "yes", "production/host_runtime"},
		"linux-x86": {"yes", "yes", "no/host-dependent", "partial", "partial", "partial", "build_lower_only"},
		"linux-x32": {"yes", "yes", "no/host-dependent", "partial", "partial", "special", "build_lower_only"},
	}[entry.Triple]
	var issues []string
	if entry.BuildOnly && (entry.MemoryRun == "yes" || entry.MemoryClaimLevel == "production/host_runtime") {
		issues = append(issues, fmt.Sprintf("%s runtime memory claim requires target runtime evidence, but target is build-only", entry.Triple))
	}
	if entry.Triple == "linux-x64" && (entry.MemoryRun == "yes" || entry.MemoryClaimLevel == "production/host_runtime") {
		for _, artifact := range []string{"linux-x64-abi.json", "linux-x64-runner.json"} {
			if !containsString(entry.EvidenceArtifacts, artifact) {
				issues = append(issues, fmt.Sprintf("%s production runtime memory claim requires %s evidence", entry.Triple, artifact))
			}
		}
	}
	if entry.MemoryBuild != want.build {
		issues = append(issues, fmt.Sprintf("%s memory_build = %q, want %q", entry.Triple, entry.MemoryBuild, want.build))
	}
	if entry.MemoryLower != want.lower {
		issues = append(issues, fmt.Sprintf("%s memory_lower = %q, want %q", entry.Triple, entry.MemoryLower, want.lower))
	}
	if entry.MemoryRun != want.run {
		issues = append(issues, fmt.Sprintf("%s memory_run = %q, want %q", entry.Triple, entry.MemoryRun, want.run))
	}
	if entry.MemoryRawDiagnostics != want.rawDiagnostics {
		issues = append(issues, fmt.Sprintf("%s memory_raw_diagnostics = %q, want %q", entry.Triple, entry.MemoryRawDiagnostics, want.rawDiagnostics))
	}
	if entry.MemoryRegionLowering != want.regionLowering {
		issues = append(issues, fmt.Sprintf("%s memory_region_lowering = %q, want %q", entry.Triple, entry.MemoryRegionLowering, want.regionLowering))
	}
	if entry.MemoryAlignmentSemantics != want.alignmentSemantics {
		issues = append(issues, fmt.Sprintf("%s memory_alignment_semantics = %q, want %q", entry.Triple, entry.MemoryAlignmentSemantics, want.alignmentSemantics))
	}
	if entry.MemoryClaimLevel != want.claimLevel {
		issues = append(issues, fmt.Sprintf("%s memory_claim_level = %q, want %q", entry.Triple, entry.MemoryClaimLevel, want.claimLevel))
	}
	return issues
}

func validateLinuxNativePromotionMetadata(entry targetReportEntry) []string {
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
	var issues []string
	if entry.RuntimeStatus != wantRuntimeStatus[entry.Triple] {
		issues = append(issues, fmt.Sprintf("%s runtime_status = %q, want %q", entry.Triple, entry.RuntimeStatus, wantRuntimeStatus[entry.Triple]))
	}
	if entry.StdlibStatus != wantStdlibStatus[entry.Triple] {
		issues = append(issues, fmt.Sprintf("%s stdlib_status = %q, want %q", entry.Triple, entry.StdlibStatus, wantStdlibStatus[entry.Triple]))
	}
	if entry.FFIStatus != wantFFIStatus[entry.Triple] {
		issues = append(issues, fmt.Sprintf("%s ffi_status = %q, want %q", entry.Triple, entry.FFIStatus, wantFFIStatus[entry.Triple]))
	}
	if entry.ReleaseGate != "scripts/release/post_v0_4/linux-native-targets-smoke.sh" {
		issues = append(issues, fmt.Sprintf("%s release_gate = %q, want linux native smoke script", entry.Triple, entry.ReleaseGate))
	}
	if strings.TrimSpace(entry.RunnerProbeCommand) == "" {
		issues = append(issues, fmt.Sprintf("%s runner_probe_command is required", entry.Triple))
	}
	if entry.Triple == "linux-x86" && !strings.Contains(entry.RunnerProbeCommand, "--target x86") {
		issues = append(issues, fmt.Sprintf("%s runner_probe_command = %q, want x86 target probe", entry.Triple, entry.RunnerProbeCommand))
	}
	if entry.Triple == "linux-x32" && !strings.Contains(entry.RunnerProbeCommand, "--target x32") {
		issues = append(issues, fmt.Sprintf("%s runner_probe_command = %q, want x32 target probe", entry.Triple, entry.RunnerProbeCommand))
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
			issues = append(issues, fmt.Sprintf("%s evidence_artifacts missing %s", entry.Triple, artifact))
		}
	}
	return issues
}

func validateLinuxNativeSyscallMetadata(entry targetReportEntry) []string {
	type wantSyscall struct {
		instruction string
		numbering   string
		registers   []string
	}
	want := map[string]wantSyscall{
		"linux-x64": {instruction: "syscall", numbering: "x86_64", registers: []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"}},
		"linux-x86": {instruction: "int 0x80", numbering: "i386", registers: []string{"eax", "ebx", "ecx", "edx", "esi", "edi", "ebp"}},
		"linux-x32": {instruction: "syscall", numbering: "x32_syscall_bit", registers: []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"}},
	}[entry.Triple]
	var issues []string
	if entry.SyscallInstruction != want.instruction {
		issues = append(issues, fmt.Sprintf("%s syscall_instruction = %q, want %q", entry.Triple, entry.SyscallInstruction, want.instruction))
	}
	if entry.SyscallNumbering != want.numbering {
		issues = append(issues, fmt.Sprintf("%s syscall_numbering = %q, want %q", entry.Triple, entry.SyscallNumbering, want.numbering))
	}
	if !sameStringSequence(entry.SyscallArgRegisters, want.registers) {
		issues = append(issues, fmt.Sprintf("%s syscall_arg_registers = %v, want %v", entry.Triple, entry.SyscallArgRegisters, want.registers))
	}
	if entry.SyscallErrorRange != "-4095..-1" {
		issues = append(issues, fmt.Sprintf("%s syscall_error_range = %q, want -4095..-1", entry.Triple, entry.SyscallErrorRange))
	}
	return issues
}

type expectedTarget struct {
	status            string
	os                string
	arch              string
	abi               string
	dataModel         string
	format            string
	buildOnly         bool
	runMode           string
	pointerBits       int
	registerBits      int
	nativeIntBits     int
	maxAtomicBits     int
	atomicBits        []int
	atomicPointerBits int
	debugInfo         bool
	releaseOptimize   bool
}

func validateExpectedTarget(entry targetReportEntry, want expectedTarget) []string {
	var issues []string
	if entry.Status != want.status {
		issues = append(issues, fmt.Sprintf("%s status = %q, want %q", entry.Triple, entry.Status, want.status))
	}
	if entry.OS != want.os || entry.Arch != want.arch || entry.ABI != want.abi || entry.DataModel != want.dataModel || entry.Format != want.format {
		issues = append(issues, fmt.Sprintf("%s platform = os:%s arch:%s abi:%s data_model:%s format:%s, want os:%s arch:%s abi:%s data_model:%s format:%s",
			entry.Triple, entry.OS, entry.Arch, entry.ABI, entry.DataModel, entry.Format, want.os, want.arch, want.abi, want.dataModel, want.format))
	}
	if entry.BuildOnly != want.buildOnly {
		issues = append(issues, fmt.Sprintf("%s build_only = %v, want %v", entry.Triple, entry.BuildOnly, want.buildOnly))
	}
	if entry.RunMode != want.runMode {
		issues = append(issues, fmt.Sprintf("%s run_mode = %q, want %q", entry.Triple, entry.RunMode, want.runMode))
	}
	if entry.PointerWidthBits != want.pointerBits || entry.RegisterWidthBits != want.registerBits || entry.NativeIntWidthBits != want.nativeIntBits {
		issues = append(issues, fmt.Sprintf("%s widths = pointer:%d register:%d native_int:%d, want pointer:%d register:%d native_int:%d",
			entry.Triple, entry.PointerWidthBits, entry.RegisterWidthBits, entry.NativeIntWidthBits, want.pointerBits, want.registerBits, want.nativeIntBits))
	}
	if entry.Endian != "little" {
		issues = append(issues, fmt.Sprintf("%s endian = %q, want little", entry.Triple, entry.Endian))
	}
	if entry.StackAlignmentBytes != 16 {
		issues = append(issues, fmt.Sprintf("%s stack_alignment_bytes = %d, want 16", entry.Triple, entry.StackAlignmentBytes))
	}
	if entry.MaxAtomicWidthBits != want.maxAtomicBits {
		issues = append(issues, fmt.Sprintf("%s max_atomic_width_bits = %d, want %d", entry.Triple, entry.MaxAtomicWidthBits, want.maxAtomicBits))
	}
	if !sameInts(entry.AtomicWidthBits, want.atomicBits) {
		issues = append(issues, fmt.Sprintf("%s atomic_width_bits = %v, want %v", entry.Triple, entry.AtomicWidthBits, want.atomicBits))
	}
	if entry.AtomicPointerWidthBits != want.atomicPointerBits {
		issues = append(issues, fmt.Sprintf("%s atomic_pointer_width_bits = %d, want %d", entry.Triple, entry.AtomicPointerWidthBits, want.atomicPointerBits))
	}
	if entry.SupportsDebugInfo != want.debugInfo {
		issues = append(issues, fmt.Sprintf("%s supports_debug_info = %v, want %v", entry.Triple, entry.SupportsDebugInfo, want.debugInfo))
	}
	if entry.SupportsReleaseOptimize != want.releaseOptimize {
		issues = append(issues, fmt.Sprintf("%s supports_release_optimize = %v, want %v", entry.Triple, entry.SupportsReleaseOptimize, want.releaseOptimize))
	}
	if entry.RunSupported && entry.RunUnsupportedReason != "" {
		issues = append(issues, fmt.Sprintf("%s run_unsupported_reason must be empty when run_supported is true", entry.Triple))
	}
	if !entry.RunSupported && entry.RunMode == "host_probed" {
		if !strings.Contains(entry.RunUnsupportedReason, "no host fallback") {
			issues = append(issues, fmt.Sprintf("%s run_unsupported_reason must mention no host fallback", entry.Triple))
		}
		if !strings.Contains(entry.RunUnsupportedReason, "host ") {
			issues = append(issues, fmt.Sprintf("%s run_unsupported_reason must include host identity", entry.Triple))
		}
		if !strings.Contains(entry.RunUnsupportedReason, "probe command:") {
			issues = append(issues, fmt.Sprintf("%s run_unsupported_reason must include probe command", entry.Triple))
		}
		if entry.RunnerProbeCommand != "" && !strings.Contains(entry.RunUnsupportedReason, entry.RunnerProbeCommand) {
			issues = append(issues, fmt.Sprintf("%s run_unsupported_reason must include runner_probe_command %q", entry.Triple, entry.RunnerProbeCommand))
		}
	}
	if entry.Triple == "linux-x64" && !entry.RunSupported {
		issues = append(issues, "linux-x64 must be runnable for the linux native baseline smoke")
	}
	if entry.Triple == "linux-x64" && entry.RunUnsupportedReason != "" {
		issues = append(issues, "linux-x64 run_unsupported_reason must be empty for the linux native baseline smoke")
	}
	return issues
}

func validateBuildOnlyLinuxReason(entry targetReportEntry, triple string) []string {
	var issues []string
	if entry.Status != "build_only" || !entry.BuildOnly || entry.RunMode != "host_probed" {
		issues = append(issues, fmt.Sprintf("%s must stay build_only with host_probed run mode until full runtime/stdlib/FFI promotion evidence exists", triple))
	}
	reason := strings.ToLower(entry.UnsupportedReason)
	for _, want := range []string{"not implemented yet", "runtime", "stdlib", "ffi"} {
		if !strings.Contains(reason, want) {
			issues = append(issues, fmt.Sprintf("%s unsupported_reason must mention %s", triple, want))
		}
	}
	return issues
}

func validateSuiteReport(path string, label string, requiredNames []string) []string {
	var issues []string
	if strings.TrimSpace(path) == "" {
		return []string{label + " path is required"}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", label, err)}
	}
	issues = append(issues, rejectPaperEvidence(raw, label)...)
	var report testRunnerReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return append(issues, fmt.Sprintf("%s: %v", label, err))
	}
	if report.Total <= 0 {
		issues = append(issues, fmt.Sprintf("%s total = %d, want positive", label, report.Total))
	}
	if report.Passed != report.Total || report.Failed != 0 {
		issues = append(issues, fmt.Sprintf("%s pass/fail = %d/%d of %d, want all pass", label, report.Passed, report.Failed, report.Total))
	}
	if len(report.Files) == 0 {
		issues = append(issues, fmt.Sprintf("%s files must not be empty", label))
	}
	if len(report.Results) != report.Total {
		issues = append(issues, fmt.Sprintf("%s results count = %d, want total %d", label, len(report.Results), report.Total))
	}
	seenNames := map[string]bool{}
	for i, result := range report.Results {
		name := strings.TrimSpace(result.Name)
		if name == "" {
			issues = append(issues, fmt.Sprintf("%s result[%d] name is required", label, i))
			continue
		}
		seenNames[name] = true
		if strings.TrimSpace(result.Filename) == "" {
			issues = append(issues, fmt.Sprintf("%s result %s filename is required", label, name))
		}
		if !strings.HasPrefix(result.FunctionName, "__tetra_test_") {
			issues = append(issues, fmt.Sprintf("%s result %s function_name = %q, want __tetra_test_ prefix", label, name, result.FunctionName))
		}
		if !result.Passed {
			issues = append(issues, fmt.Sprintf("%s result %s did not pass", label, name))
		}
		if result.ExitCode != 0 {
			issues = append(issues, fmt.Sprintf("%s result %s exit_code = %d, want 0", label, name, result.ExitCode))
		}
		if strings.TrimSpace(result.Error) != "" {
			issues = append(issues, fmt.Sprintf("%s result %s error must be empty", label, name))
		}
	}
	for _, name := range requiredNames {
		if !seenNames[name] {
			issues = append(issues, fmt.Sprintf("%s missing required result %q", label, name))
		}
	}
	return issues
}

func validateSuiteReportForTarget(path string, label string, requiredNames []string, expectedTarget string) []string {
	issues := validateSuiteReport(path, label, requiredNames)
	raw, err := os.ReadFile(path)
	if err != nil {
		return issues
	}
	var report testRunnerReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return issues
	}
	if strings.TrimSpace(expectedTarget) != "" && report.Target != expectedTarget {
		issues = append(issues, fmt.Sprintf("%s target = %q, want %s", label, report.Target, expectedTarget))
	}
	issues = append(issues, validateForbiddenResultNames(report, label, forbiddenSuiteResultNames(expectedTarget))...)
	return issues
}

func validateForbiddenSuiteResults(path string, label string, forbiddenNames []string) []string {
	if len(forbiddenNames) == 0 {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var report testRunnerReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return nil
	}
	return validateForbiddenResultNames(report, label, forbiddenNames)
}

func validateForbiddenResultNames(report testRunnerReport, label string, forbiddenNames []string) []string {
	if len(forbiddenNames) == 0 {
		return nil
	}
	forbidden := map[string]bool{}
	for _, name := range forbiddenNames {
		forbidden[name] = true
	}
	var issues []string
	for _, result := range report.Results {
		name := strings.TrimSpace(result.Name)
		if forbidden[name] {
			issues = append(issues, fmt.Sprintf("%s contains build-only boundary result %q", label, name))
		}
	}
	return issues
}

func forbiddenSuiteResultNames(triple string) []string {
	switch triple {
	case "linux-x64":
		return []string{
			"x64 stdlib runtime boundary diagnostics",
			"x64 target runtime boundary diagnostics",
			"x64 networking runtime boundary diagnostics",
			"x64 surface/distributed runtime boundary diagnostics",
		}
	default:
		return nil
	}
}

func validateArtifactHashManifest(path string) []string {
	label := "artifact hash manifest"
	if strings.TrimSpace(path) == "" {
		return []string{"--artifact-hashes is required"}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", label, err)}
	}
	var manifest artifactHashManifest
	if err := decodeStrictJSON(raw, &manifest); err != nil {
		return []string{fmt.Sprintf("%s: %v", label, err)}
	}
	var issues []string
	if manifest.Schema != artifactHashManifestSchema {
		issues = append(issues, fmt.Sprintf("%s schema = %q, want %s", label, manifest.Schema, artifactHashManifestSchema))
	}
	if strings.TrimSpace(manifest.Root) == "" {
		issues = append(issues, label+" root must not be empty")
	} else if filepath.IsAbs(manifest.Root) || strings.Contains(filepath.ToSlash(manifest.Root), "..") {
		issues = append(issues, fmt.Sprintf("%s root = %q is unsafe", label, manifest.Root))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, label+" artifacts must not be empty")
	}
	if len(issues) > 0 {
		return issues
	}

	root := filepath.Join(filepath.Dir(path), filepath.FromSlash(manifest.Root))
	manifestRel, err := filepath.Rel(root, path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", label, err)}
	}
	manifestRel = filepath.ToSlash(manifestRel)
	seen := map[string]bool{}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		artifactPath := strings.TrimSpace(artifact.Path)
		if artifactPath == "" {
			issues = append(issues, label+" artifact missing path")
			continue
		}
		if filepath.IsAbs(artifactPath) || strings.Contains(filepath.ToSlash(artifactPath), "..") {
			issues = append(issues, fmt.Sprintf("%s artifact path %q is unsafe", label, artifactPath))
			continue
		}
		if lastPath != "" && artifactPath < lastPath {
			issues = append(issues, fmt.Sprintf("%s artifacts must be sorted by path: %s appears before %s", label, artifactPath, lastPath))
		}
		lastPath = artifactPath
		if seen[artifactPath] {
			issues = append(issues, fmt.Sprintf("%s duplicate artifact path %s", label, artifactPath))
			continue
		}
		seen[artifactPath] = true
		if artifact.Size < 0 {
			issues = append(issues, fmt.Sprintf("%s artifact %s has negative size", label, artifactPath))
		}
		if err := validateArtifactSHA256(artifact.SHA256, artifactPath); err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", label, err))
			continue
		}
		actual, err := hashArtifact(root, artifactPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s artifact %s: %v", label, artifactPath, err))
			continue
		}
		if actual.Size != artifact.Size {
			issues = append(issues, fmt.Sprintf("%s size mismatch for %s: got %d want %d", label, artifactPath, actual.Size, artifact.Size))
		}
		if actual.SHA256 != artifact.SHA256 {
			issues = append(issues, fmt.Sprintf("%s sha256 mismatch for %s: got %s want %s", label, artifactPath, actual.SHA256, artifact.SHA256))
		}
		if actual.Schema != artifact.Schema {
			issues = append(issues, fmt.Sprintf("%s schema mismatch for %s: got %q want %q", label, artifactPath, actual.Schema, artifact.Schema))
		}
	}
	actualPaths, err := listArtifactPaths(root, manifestRel)
	if err != nil {
		issues = append(issues, fmt.Sprintf("%s: %v", label, err))
		return issues
	}
	for _, actualPath := range actualPaths {
		if !seen[actualPath] {
			issues = append(issues, fmt.Sprintf("%s missing listed hash for artifact %s", label, actualPath))
		}
	}
	return issues
}

func evidenceArtifactPaths(options validateOptions) []string {
	paths := []string{options.TargetsReport}
	for _, input := range options.TargetReports {
		paths = append(paths, input.ABIReport, input.AtomicReport, input.FuzzReport)
	}
	for _, input := range options.RunnerReports {
		paths = append(paths, input.Report)
	}
	if strings.TrimSpace(options.BrutalReport) != "" {
		paths = append(paths, options.BrutalReport)
	}
	return paths
}

func validateArtifactHashManifestCoversEvidence(manifestPath string, evidencePaths []string) []string {
	label := "artifact hash manifest"
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil
	}
	var manifest artifactHashManifest
	if err := decodeStrictJSON(raw, &manifest); err != nil {
		return nil
	}
	if strings.TrimSpace(manifest.Root) == "" || filepath.IsAbs(manifest.Root) || strings.Contains(filepath.ToSlash(manifest.Root), "..") {
		return nil
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(manifest.Root)))
	root, err = filepath.Abs(root)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", label, err)}
	}
	seen := map[string]bool{}
	for _, artifact := range manifest.Artifacts {
		seen[filepath.ToSlash(strings.TrimSpace(artifact.Path))] = true
	}

	var issues []string
	for _, path := range evidencePaths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s does not cover required evidence file %s: %v", label, path, err))
			continue
		}
		rel, err := filepath.Rel(root, absPath)
		if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
			issues = append(issues, fmt.Sprintf("%s does not cover required evidence file %s", label, path))
			continue
		}
		rel = filepath.ToSlash(rel)
		if !seen[rel] {
			issues = append(issues, fmt.Sprintf("%s does not cover required evidence file %s", label, path))
		}
	}
	return issues
}

func validateArtifactSHA256(value string, path string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("artifact %s has invalid sha256 format %q", path, value)
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return fmt.Errorf("artifact %s sha256 must contain 64 hex chars", path)
	}
	for _, ch := range hexPart {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return fmt.Errorf("artifact %s sha256 has non-hex characters", path)
		}
	}
	return nil
}

func hashArtifact(root, rel string) (artifactHashManifestRow, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	file, err := os.Open(path)
	if err != nil {
		return artifactHashManifestRow{}, err
	}
	defer file.Close()
	h := sha256.New()
	size, err := io.Copy(h, file)
	if err != nil {
		return artifactHashManifestRow{}, err
	}
	return artifactHashManifestRow{
		Path:   filepath.ToSlash(rel),
		SHA256: "sha256:" + hex.EncodeToString(h.Sum(nil)),
		Size:   size,
		Schema: artifactJSONSchema(path),
	}, nil
}

func listArtifactPaths(root, manifestRel string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == manifestRel {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func artifactJSONSchema(path string) string {
	if filepath.Ext(path) != ".json" {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
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

func rejectPaperEvidence(raw []byte, label string) []string {
	lower := strings.ToLower(string(raw))
	forbidden := []string{"metadata-only", "docs-only", "report-only", "sidecar-only", "fake", "mock", "placeholder", "skipped", "skip-only"}
	var issues []string
	for _, marker := range forbidden {
		if strings.Contains(lower, marker) {
			issues = append(issues, fmt.Sprintf("%s contains paper evidence marker %q", label, marker))
		}
	}
	return issues
}

func requiredABINames(triple string) []string {
	switch triple {
	case "linux-x86":
		return []string{
			"x86 target model",
			"x86 i386 SysV classifier",
			"x86 varargs and sret ABI",
			"x86 pointer FFI object smoke",
			"x86 c_int FFI object smoke",
			"x86 c_uint FFI object smoke",
			"x86 ILP32 native/libc FFI object smoke",
			"x86 ref FFI null-return diagnostics",
			"x86 function-pointer FFI diagnostics",
			"x86 source native scalar diagnostics",
			"x86 stdout executable smoke",
			"x86 stderr fd runtime smoke",
			"x86 allocator executable smoke",
			"x86 allocator failure executable smoke",
			"x86 raw memory bounds executable smoke",
			"x86 raw pointer slot executable smoke",
			"x86 raw pointer offset slot executable smoke",
			"x86 island free executable smoke",
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
			"x86 target runtime boundary diagnostics",
			"x86 networking runtime boundary diagnostics",
			"x86 networking lifecycle runtime smoke",
			"x86 surface/distributed runtime boundary diagnostics",
			"x86 pointer atomic ABI width",
			"x86 object ABI smoke",
			"x86 atomic ABI object",
			"x86 executable matrix smoke",
		}
	case "linux-x64":
		return []string{
			"x64 target model",
			"x64 SysV classifier",
			"x64 SysV varargs and aggregates",
			"x64 source native scalar diagnostics",
			"x64 pointer FFI regression smoke",
			"x64 c_int FFI object smoke",
			"x64 c_uint FFI object smoke",
			"x64 filesystem scheduler composition smoke",
			"x64 networking runtime smoke",
			"x64 scheduler restriction regression smoke",
			"x64 pointer atomic ABI width",
			"x64 object ABI smoke",
			"x64 atomic ABI object",
			"x64 executable matrix smoke",
		}
	case "linux-x32":
		return []string{
			"x32 target model",
			"x32 SysV classifier",
			"x32 SysV varargs and aggregates",
			"x32 pointer FFI object smoke",
			"x32 c_int FFI object smoke",
			"x32 c_uint FFI object smoke",
			"x32 ILP32 native/libc FFI object smoke",
			"x32 ref FFI null-return diagnostics",
			"x32 function-pointer FFI diagnostics",
			"x32 source native scalar diagnostics",
			"x32 stdout executable smoke",
			"x32 stderr fd runtime smoke",
			"x32 allocator executable smoke",
			"x32 allocator failure executable smoke",
			"x32 raw memory bounds executable smoke",
			"x32 raw pointer slot executable smoke",
			"x32 raw pointer offset slot executable smoke",
			"x32 island free executable smoke",
			"x32 stdlib runtime boundary diagnostics",
			"x32 time runtime smoke",
			"x32 filesystem runtime smoke",
			"x32 filesystem scheduler composition smoke",
			"x32 single-actor self-host runtime smoke",
			"x32 single-task self-host runtime smoke",
			"x32 typed-task self-host runtime smoke",
			"x32 staged typed-task self-host runtime smoke",
			"x32 task-group self-host runtime smoke",
			"x32 typed-task-group self-host runtime smoke",
			"x32 actor-state self-host runtime smoke",
			"x32 ctx_switch object smoke",
			"x32 target runtime boundary diagnostics",
			"x32 networking runtime boundary diagnostics",
			"x32 networking lifecycle runtime smoke",
			"x32 surface/distributed runtime boundary diagnostics",
			"x32 pointer atomic ABI width",
			"x32 object ABI smoke",
			"x32 atomic ABI object",
			"x32 executable matrix smoke",
		}
	default:
		return nil
	}
}

func requiredAtomicNames(triple string) []string {
	prefix := targetPrefix(triple)
	if prefix == "" {
		return nil
	}
	return []string{prefix + " atomic object matrix", prefix + " pointer atomic object width", prefix + " atomic concurrency stress oracle"}
}

func requiredFuzzNames(triple string) []string {
	prefix := targetPrefix(triple)
	if prefix == "" {
		return nil
	}
	return []string{prefix + " layout fuzz", prefix + " object signature fuzz"}
}

func requiredRunnerNames(triple string) []string {
	switch triple {
	case "linux-x64", "linux-x86", "linux-x32":
		return []string{
			"runner arithmetic",
			"runner alloc memory",
			"runner filesystem",
			"runner stderr fd",
			"runner time",
			"runner network socket",
			"runner network options",
			"runner task join",
		}
	default:
		return nil
	}
}

func requiredBrutalNames() []string {
	triples := []string{"linux-x86", "linux-x64", "linux-x32"}
	var names []string
	for _, triple := range triples {
		names = append(names, requiredABINames(triple)...)
	}
	for _, triple := range triples {
		names = append(names, requiredAtomicNames(triple)...)
	}
	for _, triple := range triples {
		names = append(names, requiredFuzzNames(triple)...)
	}
	return names
}

func fullLinuxFamilyEvidence(seen map[string]bool) bool {
	for _, triple := range []string{"linux-x64", "linux-x86", "linux-x32"} {
		if !seen[triple] {
			return false
		}
	}
	return true
}

func targetPrefix(triple string) string {
	switch triple {
	case "linux-x86":
		return "x86"
	case "linux-x64":
		return "x64"
	case "linux-x32":
		return "x32"
	default:
		return ""
	}
}

func canonicalLinuxTarget(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "x86", "linux-x86":
		return "linux-x86"
	case "x64", "linux-x64":
		return "linux-x64"
	case "x32", "linux-x32":
		return "linux-x32"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func isLinuxNativeFamilyTarget(triple string) bool {
	return triple == "linux-x64" || triple == "linux-x86" || triple == "linux-x32"
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
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

func sameInts(a []int, b []int) bool {
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

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("JSON must contain a single document")
	}
	return nil
}
