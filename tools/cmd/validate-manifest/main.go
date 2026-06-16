package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ctarget "tetra_language/compiler/target"
	"tetra_language/tools/internal/reportdecode"
)

type manifestEnvelope struct {
	CompilerVersion string             `json:"compiler_version"`
	FormatsRaw      json.RawMessage    `json:"formats,omitempty"`
	TargetsRaw      json.RawMessage    `json:"targets"`
	BuiltinsRaw     json.RawMessage    `json:"builtins"`
	RuntimeABI      runtimeABIManifest `json:"runtime_abi"`
	FeaturesRaw     json.RawMessage    `json:"features"`
	Formats         []formatManifest
	Targets         []targetManifest
	Builtins        []builtinManifest
	Features        []featureManifest
}

type formatManifest struct {
	Name        string `json:"name"`
	Extension   string `json:"extension,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Primary     bool   `json:"primary,omitempty"`
	Legacy      bool   `json:"legacy,omitempty"`
}

type targetManifest struct {
	Triple                   string   `json:"triple"`
	Status                   string   `json:"status,omitempty"`
	OS                       string   `json:"os"`
	Arch                     string   `json:"arch"`
	ABI                      string   `json:"abi"`
	DataModel                string   `json:"data_model,omitempty"`
	Format                   string   `json:"format"`
	ExeExt                   string   `json:"exe_ext"`
	CollectImports           bool     `json:"collect_imports"`
	RunMode                  string   `json:"run_mode,omitempty"`
	UIRuntimeContract        string   `json:"ui_runtime_contract,omitempty"`
	UIRuntimeStatus          string   `json:"ui_runtime_status,omitempty"`
	UIRuntimeEvidence        string   `json:"ui_runtime_evidence,omitempty"`
	PointerWidthBits         int      `json:"pointer_width_bits,omitempty"`
	RegisterWidthBits        int      `json:"register_width_bits,omitempty"`
	NativeIntWidthBits       int      `json:"native_int_width_bits,omitempty"`
	Endian                   string   `json:"endian,omitempty"`
	StackAlignmentBytes      int      `json:"stack_alignment_bytes,omitempty"`
	MaxAtomicWidthBits       int      `json:"max_atomic_width_bits,omitempty"`
	AtomicWidthBits          []int    `json:"atomic_width_bits,omitempty"`
	AtomicPointerWidthBits   int      `json:"atomic_pointer_width_bits,omitempty"`
	UnsupportedReason        string   `json:"unsupported_reason,omitempty"`
	RuntimeStatus            string   `json:"runtime_status,omitempty"`
	StdlibStatus             string   `json:"stdlib_status,omitempty"`
	FFIStatus                string   `json:"ffi_status,omitempty"`
	MemoryBuild              string   `json:"memory_build,omitempty"`
	MemoryLower              string   `json:"memory_lower,omitempty"`
	MemoryRun                string   `json:"memory_run,omitempty"`
	MemoryRawDiagnostics     string   `json:"memory_raw_diagnostics,omitempty"`
	MemoryRegionLowering     string   `json:"memory_region_lowering,omitempty"`
	MemoryAlignmentSemantics string   `json:"memory_alignment_semantics,omitempty"`
	MemoryClaimLevel         string   `json:"memory_claim_level,omitempty"`
	RunnerProbeCommand       string   `json:"runner_probe_command,omitempty"`
	ReleaseGate              string   `json:"release_gate,omitempty"`
	EvidenceArtifacts        []string `json:"evidence_artifacts,omitempty"`
	SyscallInstruction       string   `json:"syscall_instruction,omitempty"`
	SyscallNumbering         string   `json:"syscall_numbering,omitempty"`
	SyscallArgRegisters      []string `json:"syscall_arg_registers,omitempty"`
	SyscallErrorRange        string   `json:"syscall_error_range,omitempty"`
	SupportsDebugInfo        bool     `json:"supports_debug_info,omitempty"`
	SupportsReleaseOptimize  bool     `json:"supports_release_optimize,omitempty"`
}

type builtinManifest struct {
	Name          string   `json:"name"`
	Aliases       []string `json:"aliases,omitempty"`
	ParamTypes    []string `json:"param_types,omitempty"`
	ReturnType    string   `json:"return_type"`
	Effects       []string `json:"effects,omitempty"`
	UnsafePolicy  string   `json:"unsafe_policy"`
	UnsafeDetails string   `json:"unsafe_details,omitempty"`
}

type runtimeABIManifest struct {
	ReservedPrefix            string   `json:"reserved_prefix"`
	ActorsSupportedTargets    []string `json:"actors_supported_targets"`
	ActorsRequiredSymbols     []string `json:"actors_required_symbols"`
	ActorStateRequiredSymbols []string `json:"actor_state_required_symbols,omitempty"`
	TaskRequiredSymbols       []string `json:"task_required_symbols,omitempty"`
	TaskGroupRequiredSymbols  []string `json:"task_group_required_symbols,omitempty"`
	TypedTaskRequiredSymbols  []string `json:"typed_task_required_symbols,omitempty"`
	TimeRequiredSymbols       []string `json:"time_required_symbols,omitempty"`
	FilesystemRequiredSymbols []string `json:"filesystem_required_symbols,omitempty"`
	NetRequiredSymbols        []string `json:"net_required_symbols,omitempty"`
	SurfaceRequiredSymbols    []string `json:"surface_required_symbols,omitempty"`
	ActorsProgramGlueSymbols  []string `json:"actors_program_glue_symbols"`
}

type featureManifest struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Since     string   `json:"since,omitempty"`
	Scope     string   `json:"scope"`
	Stability string   `json:"stability"`
	Docs      []string `json:"docs"`
}

const manifestArtifact = "tetra.release.v0_4_0.manifest-json.v1"

func main() {
	var manifestPath string
	var reportFormat string
	flag.StringVar(&manifestPath, "manifest", "", "path to generated manifest JSON")
	flag.StringVar(&reportFormat, "format", "auto", "manifest format: auto, json, or toon")
	flag.Parse()

	if manifestPath == "" {
		fmt.Fprintln(os.Stderr, "error: --manifest is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateManifestFormat(raw, reportFormat); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateManifest(raw []byte) error {
	return validateManifestFormat(raw, "auto")
}

func validateManifestFormat(raw []byte, format string) error {
	var manifest manifestEnvelope
	if err := reportdecode.DecodeStrictFormat(raw, format, &manifest); err != nil {
		return err
	}
	if manifest.CompilerVersion == "" {
		return fmt.Errorf("compiler_version is required")
	}
	if len(bytes.TrimSpace(manifest.FormatsRaw)) > 0 {
		if err := unmarshalArray(manifest.FormatsRaw, "formats", &manifest.Formats); err != nil {
			return err
		}
		if err := validateFormats(manifest.Formats); err != nil {
			return err
		}
	}
	if err := unmarshalArray(manifest.TargetsRaw, "targets", &manifest.Targets); err != nil {
		return err
	}
	if err := unmarshalArray(manifest.BuiltinsRaw, "builtins", &manifest.Builtins); err != nil {
		return err
	}
	if len(manifest.Targets) == 0 {
		return fmt.Errorf("targets must not be empty")
	}
	if len(manifest.Builtins) == 0 {
		return fmt.Errorf("builtins must not be empty")
	}
	if !isSortedStrings(extractBuiltinNames(manifest.Builtins)) {
		return fmt.Errorf("builtins must be sorted by name for deterministic manifest output")
	}
	targets := map[string]bool{}
	var targetTriples []string
	for _, target := range manifest.Targets {
		if err := validateTarget(target); err != nil {
			return err
		}
		if targets[target.Triple] {
			return fmt.Errorf("duplicate target %s", target.Triple)
		}
		targets[target.Triple] = true
		targetTriples = append(targetTriples, target.Triple)
	}
	supportedTargets := ctarget.SupportedTriples()
	wantTargets := append([]string{}, ctarget.SupportedTriples()...)
	wantTargets = append(wantTargets, ctarget.BuildOnlyTriples()...)
	if !sameStringSet(targetTriples, supportedTargets) && !sameStringSet(targetTriples, wantTargets) {
		return fmt.Errorf("targets got %s want %s", strings.Join(sortedStrings(targetTriples), ", "), strings.Join(sortedStrings(wantTargets), ", "))
	}
	if !sameStringSequence(targetTriples, supportedTargets) && !sameStringSequence(targetTriples, wantTargets) {
		return fmt.Errorf("targets must follow buildable target order: got %s want %s", strings.Join(targetTriples, ", "), strings.Join(wantTargets, ", "))
	}
	builtins := map[string]bool{}
	for _, builtin := range manifest.Builtins {
		if err := validateBuiltin(builtin); err != nil {
			return err
		}
		if builtins[builtin.Name] {
			return fmt.Errorf("duplicate builtin %s", builtin.Name)
		}
		builtins[builtin.Name] = true
	}
	if err := validateRuntimeABI(manifest.RuntimeABI, targets); err != nil {
		return err
	}
	if err := unmarshalArray(manifest.FeaturesRaw, "features", &manifest.Features); err != nil {
		return err
	}
	if featureHasStatus(manifest.Features, "targets.wasm-artifact-preflight", "current") {
		for _, triple := range ctarget.WASMTriples() {
			if !targets[triple] {
				return fmt.Errorf("targets.wasm-artifact-preflight is current but targets missing %s", triple)
			}
		}
	}
	return validateFeatures(manifest.Features)
}

func featureHasStatus(features []featureManifest, id string, status string) bool {
	for _, feature := range features {
		if feature.ID == id && feature.Status == status {
			return true
		}
	}
	return false
}

func validateFormats(formats []formatManifest) error {
	if len(formats) == 0 {
		return fmt.Errorf("formats must not be empty")
	}
	required := map[string]string{
		".t4":        "source",
		".tetra":     "source",
		".tdx":       "todex-fragment",
		".t4s":       "offline-seed",
		".t4i":       "interface",
		".t4p":       "proof",
		".t4r":       "replay",
		".t4q":       "quest",
		".tneed":     "needmap",
		"Tetra.lock": "semantic-lock",
	}
	officialOrder := []string{".t4", ".tetra", ".tdx", ".t4s", ".t4i", ".t4p", ".t4r", ".t4q", ".tneed", "Tetra.lock"}
	seen := map[string]bool{}
	var order []string
	for _, format := range formats {
		if format.Name == "" {
			return fmt.Errorf("format missing name")
		}
		if format.Role == "" {
			return fmt.Errorf("format %s missing role", format.Name)
		}
		if format.Description == "" {
			return fmt.Errorf("format %s missing description", format.Name)
		}
		key := format.Extension
		if key == "" {
			key = format.FileName
		}
		if key == "" {
			return fmt.Errorf("format %s missing extension or file_name", format.Name)
		}
		if format.Extension != "" && format.FileName != "" {
			return fmt.Errorf("format %s must not set both extension and file_name", format.Name)
		}
		if seen[key] {
			return fmt.Errorf("duplicate format %s", key)
		}
		seen[key] = true
		order = append(order, key)
		if wantRole, ok := required[key]; ok && format.Role != wantRole {
			return fmt.Errorf("format %s role = %s, want %s", key, format.Role, wantRole)
		}
		switch key {
		case ".t4":
			if !format.Primary || format.Legacy {
				return fmt.Errorf(".t4 must be primary source format")
			}
		case ".tetra":
			if !format.Legacy || format.Primary {
				return fmt.Errorf(".tetra must be legacy source format")
			}
		}
	}
	for _, key := range officialOrder {
		if !seen[key] {
			return fmt.Errorf("formats missing %s", key)
		}
	}
	if len(order) >= len(officialOrder) && !sameStringSequence(order[:len(officialOrder)], officialOrder) {
		return fmt.Errorf("formats must start with official T4 order: got %s want %s", strings.Join(order[:len(officialOrder)], ", "), strings.Join(officialOrder, ", "))
	}
	return nil
}

func unmarshalArray[T any](raw json.RawMessage, field string, out *[]T) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("%s must be an array", field)
	}
	if bytes.Equal(trimmed, []byte("null")) || trimmed[0] != '[' {
		return fmt.Errorf("%s must be an array, not null", field)
	}
	if err := decodeStrictJSON(trimmed, out); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func validateTarget(target targetManifest) error {
	if target.Triple == "" {
		return fmt.Errorf("target missing triple")
	}
	if target.OS == "" {
		return fmt.Errorf("target %s missing os", target.Triple)
	}
	if target.Arch == "" {
		return fmt.Errorf("target %s missing arch", target.Triple)
	}
	if target.ABI == "" {
		return fmt.Errorf("target %s missing abi", target.Triple)
	}
	if target.Format == "" {
		return fmt.Errorf("target %s missing format", target.Triple)
	}
	if tgt, err := ctarget.Parse(target.Triple); err == nil {
		if target.Status != "" && target.Status != tgt.Status.String() {
			return fmt.Errorf("target %s status = %s, want %s", target.Triple, target.Status, tgt.Status.String())
		}
		if target.DataModel != "" && target.DataModel != tgt.DataModel.String() {
			return fmt.Errorf("target %s data_model = %s, want %s", target.Triple, target.DataModel, tgt.DataModel.String())
		}
		if target.RunMode != "" && target.RunMode != tgt.RunMode.String() {
			return fmt.Errorf("target %s run_mode = %s, want %s", target.Triple, target.RunMode, tgt.RunMode.String())
		}
		if target.UIRuntimeStatus != "" && target.UIRuntimeStatus != ctarget.UIRuntimeStatus(target.Triple) {
			return fmt.Errorf("target %s ui_runtime_status = %s, want %s", target.Triple, target.UIRuntimeStatus, ctarget.UIRuntimeStatus(target.Triple))
		}
		if target.UIRuntimeContract != "" && target.UIRuntimeContract != ctarget.UIRuntimeContract(target.Triple) {
			return fmt.Errorf("target %s ui_runtime_contract = %s, want %s", target.Triple, target.UIRuntimeContract, ctarget.UIRuntimeContract(target.Triple))
		}
		if target.PointerWidthBits != 0 && target.PointerWidthBits != tgt.PointerWidthBits {
			return fmt.Errorf("target %s pointer_width_bits = %d, want %d", target.Triple, target.PointerWidthBits, tgt.PointerWidthBits)
		}
		if target.RegisterWidthBits != 0 && target.RegisterWidthBits != tgt.RegisterWidthBits {
			return fmt.Errorf("target %s register_width_bits = %d, want %d", target.Triple, target.RegisterWidthBits, tgt.RegisterWidthBits)
		}
		if target.NativeIntWidthBits != 0 && target.NativeIntWidthBits != tgt.NativeIntWidthBits {
			return fmt.Errorf("target %s native_int_width_bits = %d, want %d", target.Triple, target.NativeIntWidthBits, tgt.NativeIntWidthBits)
		}
		if target.Endian != "" && target.Endian != tgt.Endian.String() {
			return fmt.Errorf("target %s endian = %s, want %s", target.Triple, target.Endian, tgt.Endian.String())
		}
		if target.StackAlignmentBytes != 0 && target.StackAlignmentBytes != tgt.StackAlignmentBytes {
			return fmt.Errorf("target %s stack_alignment_bytes = %d, want %d", target.Triple, target.StackAlignmentBytes, tgt.StackAlignmentBytes)
		}
		if target.MaxAtomicWidthBits != 0 && target.MaxAtomicWidthBits != tgt.MaxAtomicWidthBits {
			return fmt.Errorf("target %s max_atomic_width_bits = %d, want %d", target.Triple, target.MaxAtomicWidthBits, tgt.MaxAtomicWidthBits)
		}
		if len(target.AtomicWidthBits) > 0 && !sameInts(target.AtomicWidthBits, tgt.AtomicWidthBits()) {
			return fmt.Errorf("target %s atomic_width_bits = %v, want %v", target.Triple, target.AtomicWidthBits, tgt.AtomicWidthBits())
		}
		if target.AtomicPointerWidthBits != 0 {
			ptr, err := tgt.AtomicPointerLayout()
			if err != nil {
				return fmt.Errorf("target %s atomic_pointer_width_bits unsupported: %v", target.Triple, err)
			}
			if target.AtomicPointerWidthBits != ptr.WidthBits {
				return fmt.Errorf("target %s atomic_pointer_width_bits = %d, want %d", target.Triple, target.AtomicPointerWidthBits, ptr.WidthBits)
			}
		}
		if target.UnsupportedReason != "" && target.UnsupportedReason != tgt.UnsupportedReason {
			return fmt.Errorf("target %s unsupported_reason = %q, want %q", target.Triple, target.UnsupportedReason, tgt.UnsupportedReason)
		}
		if target.RuntimeStatus != "" && target.RuntimeStatus != tgt.RuntimeStatus {
			return fmt.Errorf("target %s runtime_status = %s, want %s", target.Triple, target.RuntimeStatus, tgt.RuntimeStatus)
		}
		if target.StdlibStatus != "" && target.StdlibStatus != tgt.StdlibStatus {
			return fmt.Errorf("target %s stdlib_status = %s, want %s", target.Triple, target.StdlibStatus, tgt.StdlibStatus)
		}
		if target.FFIStatus != "" && target.FFIStatus != tgt.FFIStatus {
			return fmt.Errorf("target %s ffi_status = %s, want %s", target.Triple, target.FFIStatus, tgt.FFIStatus)
		}
		if target.RunnerProbeCommand != "" && target.RunnerProbeCommand != tgt.RunnerProbeCommand {
			return fmt.Errorf("target %s runner_probe_command = %q, want %q", target.Triple, target.RunnerProbeCommand, tgt.RunnerProbeCommand)
		}
		if target.ReleaseGate != "" && target.ReleaseGate != tgt.ReleaseGate {
			return fmt.Errorf("target %s release_gate = %q, want %q", target.Triple, target.ReleaseGate, tgt.ReleaseGate)
		}
		if len(target.EvidenceArtifacts) > 0 && !sameStringSequence(target.EvidenceArtifacts, tgt.EvidenceArtifacts) {
			return fmt.Errorf("target %s evidence_artifacts = %s, want %s", target.Triple, strings.Join(target.EvidenceArtifacts, ", "), strings.Join(tgt.EvidenceArtifacts, ", "))
		}
		if err := validateTargetMemoryCapabilities(target, tgt); err != nil {
			return err
		}
		if target.SyscallInstruction != "" && target.SyscallInstruction != tgt.SyscallInstruction {
			return fmt.Errorf("target %s syscall_instruction = %q, want %q", target.Triple, target.SyscallInstruction, tgt.SyscallInstruction)
		}
		if target.SyscallNumbering != "" && target.SyscallNumbering != tgt.SyscallNumbering {
			return fmt.Errorf("target %s syscall_numbering = %q, want %q", target.Triple, target.SyscallNumbering, tgt.SyscallNumbering)
		}
		if len(target.SyscallArgRegisters) > 0 && !sameStringSequence(target.SyscallArgRegisters, tgt.SyscallArgRegisters) {
			return fmt.Errorf("target %s syscall_arg_registers = %s, want %s", target.Triple, strings.Join(target.SyscallArgRegisters, ", "), strings.Join(tgt.SyscallArgRegisters, ", "))
		}
		if target.SyscallErrorRange != "" && target.SyscallErrorRange != tgt.SyscallErrorRange {
			return fmt.Errorf("target %s syscall_error_range = %q, want %q", target.Triple, target.SyscallErrorRange, tgt.SyscallErrorRange)
		}
	}
	return nil
}

func validateTargetMemoryCapabilities(target targetManifest, tgt ctarget.Target) error {
	if !targetHasMemoryCapabilityFields(target) && target.Status == "" {
		return nil
	}
	if tgt.Status == ctarget.StatusBuildOnly && (target.MemoryRun == "yes" || target.MemoryClaimLevel == "production/host_runtime") {
		return fmt.Errorf("target %s runtime memory claim requires target runtime evidence, but target is build-only", target.Triple)
	}
	if target.MemoryRawDiagnostics == "yes" && target.Triple != "linux-x64" {
		return fmt.Errorf("target %s raw diagnostics claim requires raw diagnostics evidence", target.Triple)
	}
	if (target.MemoryRegionLowering == "yes" || target.MemoryRegionLowering == "yes/partial" || target.MemoryRegionLowering == "partial") &&
		(target.Triple == "linux-x64" || target.Triple == "linux-x86" || target.Triple == "linux-x32") &&
		!containsString(target.EvidenceArtifacts, target.Triple+"-abi.json") {
		return fmt.Errorf("target %s region lowering claim requires lowered artifact evidence", target.Triple)
	}
	if target.MemoryAlignmentSemantics == "yes" && (target.ABI == "" || target.DataModel == "") {
		return fmt.Errorf("target %s alignment claim requires target-specific ABI evidence", target.Triple)
	}
	if target.MemoryBuild != tgt.MemoryBuild {
		return fmt.Errorf("target %s memory_build = %s, want %s", target.Triple, target.MemoryBuild, tgt.MemoryBuild)
	}
	if target.MemoryLower != tgt.MemoryLower {
		return fmt.Errorf("target %s memory_lower = %s, want %s", target.Triple, target.MemoryLower, tgt.MemoryLower)
	}
	if target.MemoryRun != tgt.MemoryRun {
		return fmt.Errorf("target %s memory_run = %s, want %s", target.Triple, target.MemoryRun, tgt.MemoryRun)
	}
	if target.MemoryRawDiagnostics != tgt.MemoryRawDiagnostics {
		return fmt.Errorf("target %s memory_raw_diagnostics = %s, want %s", target.Triple, target.MemoryRawDiagnostics, tgt.MemoryRawDiagnostics)
	}
	if target.MemoryRegionLowering != tgt.MemoryRegionLowering {
		return fmt.Errorf("target %s memory_region_lowering = %s, want %s", target.Triple, target.MemoryRegionLowering, tgt.MemoryRegionLowering)
	}
	if target.MemoryAlignmentSemantics != tgt.MemoryAlignmentSemantics {
		return fmt.Errorf("target %s memory_alignment_semantics = %s, want %s", target.Triple, target.MemoryAlignmentSemantics, tgt.MemoryAlignmentSemantics)
	}
	if target.MemoryClaimLevel != tgt.MemoryClaimLevel {
		return fmt.Errorf("target %s memory_claim_level = %s, want %s", target.Triple, target.MemoryClaimLevel, tgt.MemoryClaimLevel)
	}
	return nil
}

func targetHasMemoryCapabilityFields(target targetManifest) bool {
	return target.MemoryBuild != "" ||
		target.MemoryLower != "" ||
		target.MemoryRun != "" ||
		target.MemoryRawDiagnostics != "" ||
		target.MemoryRegionLowering != "" ||
		target.MemoryAlignmentSemantics != "" ||
		target.MemoryClaimLevel != ""
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

func validateBuiltin(builtin builtinManifest) error {
	if builtin.Name == "" {
		return fmt.Errorf("builtin missing name")
	}
	if builtin.ReturnType == "" {
		return fmt.Errorf("builtin %s missing return_type", builtin.Name)
	}
	switch builtin.UnsafePolicy {
	case "never", "always", "conditional":
	default:
		return fmt.Errorf("builtin %s invalid unsafe_policy %q", builtin.Name, builtin.UnsafePolicy)
	}
	if builtin.UnsafePolicy == "conditional" && builtin.UnsafeDetails == "" {
		return fmt.Errorf("builtin %s conditional unsafe_policy requires unsafe_details", builtin.Name)
	}
	for _, effect := range builtin.Effects {
		if effect == "" {
			return fmt.Errorf("builtin %s has empty effect", builtin.Name)
		}
	}
	return nil
}

func validateFeatures(features []featureManifest) error {
	if len(features) == 0 {
		return fmt.Errorf("features must not be empty")
	}
	allowedStatus := map[string]bool{
		"current":              true,
		"experimental":         true,
		"release_candidate":    true,
		"unsupported":          true,
		"legacy_compatibility": true,
		"planned":              true,
		"post-v1":              true,
	}
	requiredStatus := map[string]bool{"current": false, "planned": false, "post-v1": false}
	requiredIDs := map[string]string{
		"cli.core":                                "current",
		"language.flow":                           "current",
		"language.generics-mvp":                   "current",
		"language.protocol-conformance-mvp":       "current",
		"language.callable-mvp":                   "current",
		"language.callable-level1":                "current",
		"targets.wasm-artifact-preflight":         "current",
		"stdlib.experimental-mirrors":             "current",
		"language.enum-payload-match":             "current",
		"language.protocol-bound-generics-static": "current",
		"language.ownership-markers-mvp":          "current",
		"language.resource-lifetime-mvp":          "current",
		"actors.task-transfer-safety":             "current",
		"language.lifetime-ssa":                   "current",
		"safety.production-core":                  "current",
		"language.callable-level2":                "current",
		"compiler.ram-contracts":                  "current",
		"ui.metadata-v1":                          "current",
		"ui.toolkit-core":                         "current",
		"wasm.runtime-execution":                  "current",
		"language.full-v1-guarantees":             "planned",
		"eco.distributed-network":                 "post-v1",
		"language.full-first-class-callables":     "current",
	}
	seen := map[string]string{}
	featureByID := map[string]featureManifest{}
	for _, feature := range features {
		if feature.ID == "" {
			return fmt.Errorf("feature missing id")
		}
		if feature.Name == "" || feature.Scope == "" || feature.Stability == "" {
			return fmt.Errorf("feature %s missing name, scope, or stability", feature.ID)
		}
		if !allowedStatus[feature.Status] {
			return fmt.Errorf("feature %s invalid status %q", feature.ID, feature.Status)
		}
		if seenStatus, ok := seen[feature.ID]; ok {
			return fmt.Errorf("duplicate feature %s (%s and %s)", feature.ID, seenStatus, feature.Status)
		}
		seen[feature.ID] = feature.Status
		featureByID[feature.ID] = feature
		requiredStatus[feature.Status] = true
		if feature.Status == "current" && feature.Since == "" {
			return fmt.Errorf("current feature %s missing since", feature.ID)
		}
		if len(feature.Docs) == 0 {
			return fmt.Errorf("feature %s missing docs", feature.ID)
		}
		if claims := forbiddenPersistentObjectMemoryClaims(feature.Scope + " " + feature.Stability); len(claims) > 0 {
			return fmt.Errorf("feature %s forbidden persistent/object memory claim %q", feature.ID, strings.Join(claims, ", "))
		}
		if claims := forbiddenMemory100ManifestClaims(feature.Scope + " " + feature.Stability); len(claims) > 0 {
			return fmt.Errorf("feature %s forbidden Memory100 claim %q", feature.ID, strings.Join(claims, ", "))
		}
		for _, doc := range feature.Docs {
			docPath := filepath.ToSlash(doc)
			if doc == "" || filepath.IsAbs(doc) || strings.Contains(docPath, "..") {
				return fmt.Errorf("feature %s invalid doc reference %q", feature.ID, doc)
			}
			if !strings.HasPrefix(docPath, "docs/") || !strings.HasSuffix(docPath, ".md") {
				return fmt.Errorf("feature %s doc reference %q must point at docs/*.md", feature.ID, doc)
			}
		}
	}
	for status, present := range requiredStatus {
		if !present {
			return fmt.Errorf("features missing %s status", status)
		}
	}
	for id, wantStatus := range requiredIDs {
		if gotStatus, ok := seen[id]; !ok {
			return fmt.Errorf("features missing %s", id)
		} else if gotStatus != wantStatus {
			return fmt.Errorf("feature %s status = %s, want %s", id, gotStatus, wantStatus)
		}
	}
	if err := validateFeatureTruthBoundaries(featureByID); err != nil {
		return err
	}
	if err := validateSurfaceFeatureRows(featureByID); err != nil {
		return err
	}
	return nil
}

func validateFeatureTruthBoundaries(features map[string]featureManifest) error {
	checks := map[string][]string{
		"language.generics-mvp": {
			"statically monomorphized",
			"no runtime generic values or dynamic dispatch",
			"generic structs",
			"future/post-v1",
		},
		"language.protocol-conformance-mvp": {
			"checked statically",
			"generic requirement signature shape",
			"no witness tables",
			"dynamic dispatch remain post-v1",
		},
		"language.ownership-markers-mvp": {
			"conservative borrow/inout/consume marker checks",
			"use-after-consume",
			"borrow escape diagnostics",
			"not a full SSA lifetime solver",
		},
		"language.resource-lifetime-mvp": {
			"conservative resource finalization checks",
			"task handles",
			"island handles",
			"double-use",
			"ambiguous provenance",
			"not a full SSA lifetime solver",
		},
		"actors.task-transfer-safety": {
			"conservative actor/task ownership transfer checks",
			"worker entrypoints",
			"use-after-transfer diagnostics",
			"conservative local MVP",
			"distributed actors",
		},
		"language.lifetime-ssa": {
			"production SSA-like local lifetime join analysis",
			"ownership consume state",
			"resource finalization state",
			"maybe-consumed diagnostics",
			"richer interprocedural lifetime proofs",
		},
		"safety.production-core": {
			"production local safety model",
			"ownership/lifetime/borrow/consume/inout",
			"resource finalization",
			"callable escape diagnostics",
			"effects/capabilities/privacy/consent/budget",
			"unsafe boundaries",
			"actor/task transfer safety",
			"pointer/MMIO/memory capability gates",
			"memory cost model",
			"memory fuzz oracle",
			"memory production final audit",
			"validate-island-proof",
			"--islands-debug sanitizer smoke",
			"island-proof-fuzz-summary",
			"leak/resource finalization evidence",
			"integrated Memory/Islands/Surface release gate",
			"memory-islands-surface-production-manifest.json",
			"artifact-hashes.json",
			"no Memory 100% claim",
			"no production object memory",
			"production persistent memory claim",
			"explicit diagnostics",
		},
		"compiler.ram-contracts": {
			"RAM Contract Compiler report evidence",
			"tetra.ram-contract-report.v1",
			"tetra.memory-grade-report.v1",
			"tetra.proof-store-summary.v1",
			"tetra.validation-pipeline-coverage.v1",
			"heap-blockers.json",
			"copy-blockers.json",
			"ram-contract-fuzz-oracle.json",
			"--emit-ram-contract-report",
			"--fail-if-heap",
			"--fail-if-copy",
			"--fail-if-unbounded",
			"--memory-budget",
			"--ram-contract",
			"TETRA4100",
			"validate-ram-contract-release",
			"ram-contract-linux-x64-smoke.sh",
			"no zero heap for all programs claim",
			"no zero-copy for all programs claim",
			"no full formal proof claim",
			"no all-target RAM parity claim",
			"no production object memory claim",
			"no production persistent memory claim",
			"no performance claim",
		},
		"language.enum-payload-match": {
			"positional enum payload constructors",
			"match/catch/if-let",
			"exhaustive unguarded enum match/catch",
			"nested destructuring patterns",
			"guard expansion remain future/post-v1",
		},
		"language.protocol-bound-generics-static": {
			"validated statically during monomorphization",
			"same-module and cross-module impl conformance",
			"visibility diagnostics",
			"calling protocol requirements through generic bounds",
			"dynamic dispatch remain unsupported",
		},
		"ui.toolkit-core": {
			"production platform-independent UI Toolkit Core contract",
			"tetra.ui.toolkit.v1",
			"widget model",
			"layout model",
			"accessibility model",
			"event dispatch",
			"state binding/update",
			"runtime trace artifacts",
			"metadata-only",
			"runtime-less",
			"native-shell sidecar-only",
			"web-only",
			"GTK/Qt/OS platform backend production",
			"full cross-platform UI",
		},
	}
	docChecks := map[string][]string{
		"language.generics-mvp":                   {"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"},
		"language.protocol-conformance-mvp":       {"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"},
		"language.ownership-markers-mvp":          {"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		"language.resource-lifetime-mvp":          {"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		"actors.task-transfer-safety":             {"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		"language.lifetime-ssa":                   {"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		"safety.production-core":                  {"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/effects_capabilities_privacy_v1.md", "docs/spec/unsafe.md", "docs/spec/memory_report_schema_v1.md", "docs/spec/islands.md", "docs/design/memory_production_core_v1.md", "docs/design/memory_cost_model.md", "docs/audits/memory-fuzz-oracle-v1.md", "docs/audits/memory-production-core-v1-final.md", "docs/audits/memory-production-core-v1-artifact-map.md", "docs/audits/memory-production-core-v1-nonclaims.md", "docs/release/memory_islands_surface_scope.md", "docs/audits/memory-ideal-vslice-v0-baseline.md", "docs/audits/memory-ideal-vslice-v0-correlation.md", "docs/audits/memory-ideal-vslice-v0-final.md", "docs/audits/memory-ideal-vslice-v1-correlation.md", "docs/audits/memory-ideal-vslice-v1-final.md", "docs/audits/memory-ideal-vslice-v2-correlation.md", "docs/audits/memory-ideal-vslice-v2-final.md", "docs/audits/memory-ideal-vslice-v3-correlation.md", "docs/audits/memory-ideal-vslice-v3-final.md"},
		"compiler.ram-contracts":                  {"docs/design/ram_contract_compiler.md", "docs/spec/ram_contract_report_schema.md", "docs/user/ram_contracts.md", "docs/audits/ram-contract-compiler-readiness.md", "docs/audits/ram-contract-compiler-handoff.md"},
		"language.enum-payload-match":             {"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v0_3_scope.md"},
		"language.protocol-bound-generics-static": {"docs/spec/current_supported_surface.md", "docs/spec/v0_3_scope.md", "docs/spec/flow_syntax_v1.md"},
		"ui.toolkit-core":                         {"docs/spec/current_supported_surface.md", "docs/spec/ui_toolkit_core.md", "docs/spec/ui_v0.4.0.md"},
	}
	for id, required := range checks {
		feature, ok := features[id]
		if !ok {
			return fmt.Errorf("features missing %s", id)
		}
		haystack := feature.Scope + " " + feature.Stability
		for _, want := range required {
			if !strings.Contains(haystack, want) {
				return fmt.Errorf("feature %s missing truth-boundary phrase %q", id, want)
			}
		}
		for _, doc := range docChecks[id] {
			if !containsString(feature.Docs, doc) {
				return fmt.Errorf("feature %s missing doc reference %s", id, doc)
			}
		}
	}
	return nil
}

func validateSurfaceFeatureRows(features map[string]featureManifest) error {
	if _, ok := features["ui.surface-core"]; !ok {
		return nil
	}
	requiredStatus := map[string]string{
		"ui.surface-core":             "current",
		"ui.surface-block-system":     "experimental",
		"ui.surface-morph-capsule":    "experimental",
		"ui.surface-headless":         "current",
		"ui.surface-linux-x64":        "current",
		"ui.surface-web-wasm":         "current",
		"ui.surface-component-model":  "current",
		"ui.surface-toolkit-v1":       "current",
		"ui.surface-text-input-v1":    "current",
		"ui.surface-accessibility-v1": "current",
		"ui.surface-macos-x64":        "unsupported",
		"ui.surface-windows-x64":      "unsupported",
		"ui.surface-wasm32-wasi":      "unsupported",
	}
	docChecks := map[string][]string{
		"ui.surface-core":             {"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md"},
		"ui.surface-block-system":     {"docs/spec/current_supported_surface.md", "docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md", "docs/release/surface_v1_release_contract.md", "docs/release/surface_v1_release_notes.md", "docs/release/surface_v1_release_audit.md"},
		"ui.surface-morph-capsule":    {"docs/spec/surface_morph.md", "docs/spec/current_supported_surface.md", "docs/user/surface_guide.md", "docs/user/examples_index.md", "docs/user/standard_library_guide.md", "docs/release/surface_v1_release_contract.md", "docs/release/surface_v1_release_notes.md"},
		"ui.surface-headless":         {"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md"},
		"ui.surface-linux-x64":        {"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md"},
		"ui.surface-web-wasm":         {"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md"},
		"ui.surface-component-model":  {"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md"},
		"ui.surface-toolkit-v1":       {"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md"},
		"ui.surface-text-input-v1":    {"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md"},
		"ui.surface-accessibility-v1": {"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md"},
		"ui.surface-macos-x64":        {"docs/spec/surface_v1.md", "docs/release/surface_v1_release_contract.md"},
		"ui.surface-windows-x64":      {"docs/spec/surface_v1.md", "docs/release/surface_v1_release_contract.md"},
		"ui.surface-wasm32-wasi":      {"docs/spec/surface_v1.md", "docs/release/surface_v1_release_contract.md"},
	}
	phraseChecks := map[string][]string{
		"ui.surface-core":             {"surface-v1-linux-web", "headless, linux-x64 real-window, and wasm32-web browser-canvas", "unsupported or future work"},
		"ui.surface-block-system":     {"Block-first Surface architecture", "core Surface primitive", "recipes/compatibility", "tetra.surface.block-system.gate.v1", "block_system.memory_budget", "reports/surface-block/p18-budget", "same-commit target evidence", "not current", "not production support", "no production Block claim"},
		"ui.surface-morph-capsule":    {"Morph Capsule", "lib.core.morph", "expands into Block", "tetra.surface.morph.gate.v1", "deterministic headless", "not Surface v1 production support", "does not add core widget primitives"},
		"ui.surface-headless":         {"release evidence target", "not as an end-user platform claim"},
		"ui.surface-linux-x64":        {"linux-x64-release-window-v1", "linux-app-shell-subset-v1", "electron-feature-ledger-v1", "tray/notification/file-picker/dialog support", "broad Electron shell parity"},
		"ui.surface-web-wasm":         {"wasm32-web-browser-canvas-release-v1", "DOM host canvas only", "DOM-authored app UI trees", "Node-only promotion"},
		"ui.surface-component-model":  {"component-tree-api release subset", "dynamic trait-object", "witness-table"},
		"ui.surface-toolkit-v1":       {"production-widgets-v1", "no magical widgets, platform widgets, DOM UI, user JS"},
		"ui.surface-text-input-v1":    {"production-text-input-v1", "rich text"},
		"ui.surface-accessibility-v1": {"platform-bridge-v1", "screen-reader claims"},
		"ui.surface-macos-x64":        {"unsupported for Surface v1", "no production target evidence", "tetra.surface.target-host-status.v1", "UNSUPPORTED", "build-only"},
		"ui.surface-windows-x64":      {"unsupported for Surface v1", "no production target evidence", "tetra.surface.target-host-status.v1", "UNSUPPORTED", "build-only"},
		"ui.surface-wasm32-wasi":      {"unsupported for Surface v1", "no production target evidence"},
	}
	for id, wantStatus := range requiredStatus {
		feature, ok := features[id]
		if !ok {
			return fmt.Errorf("features missing %s", id)
		}
		if feature.Status != wantStatus {
			return fmt.Errorf("feature %s status = %s, want %s", id, feature.Status, wantStatus)
		}
		haystack := feature.Scope + " " + feature.Stability
		for _, want := range phraseChecks[id] {
			if !strings.Contains(haystack, want) {
				return fmt.Errorf("feature %s missing Surface scope phrase %q", id, want)
			}
		}
		for _, doc := range docChecks[id] {
			if !containsString(feature.Docs, doc) {
				return fmt.Errorf("feature %s missing doc reference %s", id, doc)
			}
		}
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func forbiddenPersistentObjectMemoryClaims(text string) []string {
	lower := strings.ToLower(text)
	var claims []string
	for _, phrase := range []string{
		"object memory",
		"persistent memory",
		"persistent/object memory",
		"object/persistent memory",
		"production object memory",
		"object memory production",
		"production persistent memory",
		"persistent memory production",
		"todium",
		"memoryfield",
		"memoryruntime",
		"memoryeval",
		"false memory",
		"stale memory",
		"wal-backed object memory",
		"wal backed object memory",
		"fts-backed object memory",
		"fts backed object memory",
		"vacuum-backed object memory",
		"retention-backed object memory",
	} {
		searchFrom := 0
		for {
			index := strings.Index(lower[searchFrom:], phrase)
			if index < 0 {
				break
			}
			absolute := searchFrom + index
			clause := claimClauseAround(lower, absolute, len(phrase), 260)
			if !explicitNonClaimContext(clause) && persistentObjectMemoryClaimContext(phrase, clause) {
				claims = append(claims, phrase)
			}
			searchFrom = absolute + len(phrase)
		}
	}
	sort.Strings(claims)
	return compactStrings(claims)
}

func forbiddenMemory100ManifestClaims(text string) []string {
	lower := strings.ToLower(text)
	var claims []string
	for _, phrase := range []string{
		"memory 100%",
		"memory 100 percent",
		"memory is 100% ready",
		"memory 100% ready",
		"full formal proof",
		"formal proof of memory safety",
		"all-target memory parity",
		"all target memory parity",
		"all targets memory-stable",
		"all targets memory stable",
		"no leaks",
		"no memory leaks",
		"leak-free",
		"leak free",
		"leak freedom",
	} {
		searchFrom := 0
		for {
			index := strings.Index(lower[searchFrom:], phrase)
			if index < 0 {
				break
			}
			absolute := searchFrom + index
			clause := claimClauseAround(lower, absolute, len(phrase), 260)
			if !explicitMemory100NonClaimContext(clause) {
				claims = append(claims, phrase)
			}
			searchFrom = absolute + len(phrase)
		}
	}
	sort.Strings(claims)
	return compactStrings(claims)
}

func persistentObjectMemoryClaimContext(phrase string, clause string) bool {
	switch phrase {
	case "object memory", "persistent memory", "persistent/object memory", "object/persistent memory":
		for _, qualifier := range []string{
			"production",
			"prod_ready",
			"release-ready",
			"release ready",
			"supported",
			"current",
			"ships",
			"backed by",
		} {
			if strings.Contains(clause, qualifier) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func explicitMemory100NonClaimContext(lower string) bool {
	if explicitNonClaimContext(lower) {
		return true
	}
	normalized := strings.NewReplacer(`"`, "", "`", "", "'", "").Replace(lower)
	for _, marker := range []string{
		"rejects",
		"rejected",
		"rejection",
		"fake",
		"not a public",
		"not a full",
		"not full",
		"no memory 100",
		"no full",
		"no all-target",
		"no all target",
		"no target parity",
		"no leak-free",
		"no leak free",
		"or full formal proof",
	} {
		if strings.Contains(lower, marker) || strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func explicitNonClaimContext(lower string) bool {
	normalized := strings.NewReplacer(`"`, "", "`", "", "'", "").Replace(lower)
	for _, marker := range []string{
		"does not claim",
		"do not claim",
		"does not prove",
		"do not prove",
		"does not promote",
		"do not promote",
		"not production",
		"no production",
		"makes no",
		"non-goal",
		"non goal",
		"non-claim",
		"nonclaim",
		"out of scope",
		"not included",
		"does not include",
		"absent",
		"no todium",
		"no memoryfield",
		"without",
		"forbid",
		"forbidden",
	} {
		if strings.Contains(lower, marker) || strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func claimClauseAround(text string, index int, length int, maxSide int) string {
	start := index
	for start > 0 && !claimClauseBoundary(text[start-1]) {
		start--
		if index-start >= maxSide {
			break
		}
	}
	end := index + length
	for end < len(text) && !claimClauseBoundary(text[end]) {
		end++
		if end-(index+length) >= maxSide {
			break
		}
	}
	return text[start:end]
}

func claimClauseBoundary(b byte) bool {
	return b == '\n' || b == '.' || b == '!' || b == '?' || b == ';'
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:0]
	var previous string
	for _, value := range values {
		if value == previous {
			continue
		}
		out = append(out, value)
		previous = value
	}
	return out
}

func validateRuntimeABI(abi runtimeABIManifest, targets map[string]bool) error {
	if abi.ReservedPrefix == "" {
		return fmt.Errorf("runtime_abi.reserved_prefix is required")
	}
	if len(abi.ActorsSupportedTargets) == 0 {
		return fmt.Errorf("actors_supported_targets must not be empty")
	}
	if len(abi.ActorsRequiredSymbols) == 0 {
		return fmt.Errorf("actors_required_symbols must not be empty")
	}
	if len(abi.ActorsProgramGlueSymbols) == 0 {
		return fmt.Errorf("actors_program_glue_symbols must not be empty")
	}
	for _, target := range abi.ActorsSupportedTargets {
		if target == "" {
			return fmt.Errorf("actors_supported_targets contains empty target")
		}
		if !targets[target] {
			return fmt.Errorf("actors_supported_targets references unknown target %s", target)
		}
	}
	if !sameStringSet(abi.ActorsSupportedTargets, ctarget.ActorRuntimeTriples()) {
		return fmt.Errorf("actors_supported_targets got %s want %s", strings.Join(sortedStrings(abi.ActorsSupportedTargets), ", "), strings.Join(sortedStrings(ctarget.ActorRuntimeTriples()), ", "))
	}
	requiredRuntimeSymbols := []string{
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_send_slot",
		"__tetra_actor_send_commit",
		"__tetra_actor_recv",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_poll",
		"__tetra_actor_recv_until",
		"__tetra_actor_recv_msg_until",
		"__tetra_actor_recv_begin",
		"__tetra_actor_recv_slot",
		"__tetra_actor_recv_count",
		"__tetra_actor_self",
		"__tetra_actor_sender",
		"__tetra_actor_yield_now",
	}
	if !sameStringSet(abi.ActorsRequiredSymbols, requiredRuntimeSymbols) {
		return fmt.Errorf("actors_required_symbols got %s want %s", strings.Join(sortedStrings(abi.ActorsRequiredSymbols), ", "), strings.Join(sortedStrings(requiredRuntimeSymbols), ", "))
	}
	requiredTimeSymbols := []string{
		"__tetra_time_now_ms",
		"__tetra_sleep_ms",
		"__tetra_sleep_until_ms",
		"__tetra_deadline_ms",
		"__tetra_timer_ready_ms",
	}
	requiredFilesystemSymbols := []string{
		"__tetra_fs_exists",
	}
	requiredSurfaceSymbols := []string{
		"__tetra_surface_open",
		"__tetra_surface_close",
		"__tetra_surface_poll_event_kind",
		"__tetra_surface_poll_event_x",
		"__tetra_surface_poll_event_y",
		"__tetra_surface_poll_event_button",
		"__tetra_surface_poll_event_into",
		"__tetra_surface_poll_event_text_len",
		"__tetra_surface_poll_event_text_into",
		"__tetra_surface_clipboard_write_text",
		"__tetra_surface_clipboard_read_text_into",
		"__tetra_surface_poll_composition_into",
		"__tetra_surface_begin_frame",
		"__tetra_surface_present_rgba",
		"__tetra_surface_now_ms",
		"__tetra_surface_request_redraw",
	}
	requiredActorStateSymbols := []string{
		"__tetra_actor_state_load",
		"__tetra_actor_state_store",
	}
	requiredTaskSymbols := []string{
		"__tetra_task_spawn_i32",
		"__tetra_task_join_i32",
		"__tetra_task_join_result_i32",
		"__tetra_task_join_until_i32",
		"__tetra_task_poll_i32",
		"__tetra_task_is_canceled",
		"__tetra_task_checkpoint",
	}
	requiredTaskGroupSymbols := []string{
		"__tetra_task_group_open",
		"__tetra_task_group_close",
		"__tetra_task_group_cancel",
		"__tetra_task_group_current",
		"__tetra_task_group_status",
		"__tetra_task_spawn_group_i32",
	}
	requiredTypedTaskSymbols := []string{
		"__tetra_task_result_begin",
		"__tetra_task_result_slot",
		"__tetra_task_result_get",
		"__tetra_task_join_typed_2",
		"__tetra_task_join_typed_3",
		"__tetra_task_join_typed_4",
		"__tetra_task_join_typed_5",
		"__tetra_task_join_typed_6",
		"__tetra_task_join_typed_7",
		"__tetra_task_join_typed_8",
	}
	if len(abi.TimeRequiredSymbols) == 0 {
		return fmt.Errorf("time_required_symbols must not be empty")
	}
	if !sameStringSet(abi.TimeRequiredSymbols, requiredTimeSymbols) {
		return fmt.Errorf("time_required_symbols got %s want %s", strings.Join(sortedStrings(abi.TimeRequiredSymbols), ", "), strings.Join(sortedStrings(requiredTimeSymbols), ", "))
	}
	if len(abi.FilesystemRequiredSymbols) == 0 {
		return fmt.Errorf("filesystem_required_symbols must not be empty")
	}
	if !sameStringSet(abi.FilesystemRequiredSymbols, requiredFilesystemSymbols) {
		return fmt.Errorf("filesystem_required_symbols got %s want %s", strings.Join(sortedStrings(abi.FilesystemRequiredSymbols), ", "), strings.Join(sortedStrings(requiredFilesystemSymbols), ", "))
	}
	if len(abi.SurfaceRequiredSymbols) == 0 {
		return fmt.Errorf("surface_required_symbols must not be empty")
	}
	if !sameStringSet(abi.SurfaceRequiredSymbols, requiredSurfaceSymbols) {
		return fmt.Errorf("surface_required_symbols got %s want %s", strings.Join(sortedStrings(abi.SurfaceRequiredSymbols), ", "), strings.Join(sortedStrings(requiredSurfaceSymbols), ", "))
	}
	if len(abi.ActorStateRequiredSymbols) == 0 {
		return fmt.Errorf("actor_state_required_symbols must not be empty")
	}
	if !sameStringSet(abi.ActorStateRequiredSymbols, requiredActorStateSymbols) {
		return fmt.Errorf("actor_state_required_symbols got %s want %s", strings.Join(sortedStrings(abi.ActorStateRequiredSymbols), ", "), strings.Join(sortedStrings(requiredActorStateSymbols), ", "))
	}
	if len(abi.TaskRequiredSymbols) == 0 {
		return fmt.Errorf("task_required_symbols must not be empty")
	}
	if !sameStringSet(abi.TaskRequiredSymbols, requiredTaskSymbols) {
		return fmt.Errorf("task_required_symbols got %s want %s", strings.Join(sortedStrings(abi.TaskRequiredSymbols), ", "), strings.Join(sortedStrings(requiredTaskSymbols), ", "))
	}
	if len(abi.TaskGroupRequiredSymbols) == 0 {
		return fmt.Errorf("task_group_required_symbols must not be empty")
	}
	if !sameStringSet(abi.TaskGroupRequiredSymbols, requiredTaskGroupSymbols) {
		return fmt.Errorf("task_group_required_symbols got %s want %s", strings.Join(sortedStrings(abi.TaskGroupRequiredSymbols), ", "), strings.Join(sortedStrings(requiredTaskGroupSymbols), ", "))
	}
	if len(abi.TypedTaskRequiredSymbols) == 0 {
		return fmt.Errorf("typed_task_required_symbols must not be empty")
	}
	if !sameStringSet(abi.TypedTaskRequiredSymbols, requiredTypedTaskSymbols) {
		return fmt.Errorf("typed_task_required_symbols got %s want %s", strings.Join(sortedStrings(abi.TypedTaskRequiredSymbols), ", "), strings.Join(sortedStrings(requiredTypedTaskSymbols), ", "))
	}
	requiredGlueSymbols := []string{
		"__tetra_actor_dispatch",
		"__tetra_actor_main_entry_id",
	}
	if !sameStringSet(abi.ActorsProgramGlueSymbols, requiredGlueSymbols) {
		return fmt.Errorf("actors_program_glue_symbols got %s want %s", strings.Join(sortedStrings(abi.ActorsProgramGlueSymbols), ", "), strings.Join(sortedStrings(requiredGlueSymbols), ", "))
	}
	allSymbols := append([]string{}, abi.ActorsRequiredSymbols...)
	allSymbols = append(allSymbols, abi.ActorStateRequiredSymbols...)
	allSymbols = append(allSymbols, abi.TaskRequiredSymbols...)
	allSymbols = append(allSymbols, abi.TaskGroupRequiredSymbols...)
	allSymbols = append(allSymbols, abi.TypedTaskRequiredSymbols...)
	allSymbols = append(allSymbols, abi.TimeRequiredSymbols...)
	allSymbols = append(allSymbols, abi.FilesystemRequiredSymbols...)
	allSymbols = append(allSymbols, abi.NetRequiredSymbols...)
	allSymbols = append(allSymbols, abi.SurfaceRequiredSymbols...)
	allSymbols = append(allSymbols, abi.ActorsProgramGlueSymbols...)
	for _, symbol := range allSymbols {
		if symbol == "" {
			return fmt.Errorf("runtime_abi contains empty symbol")
		}
		if len(symbol) < len(abi.ReservedPrefix) || symbol[:len(abi.ReservedPrefix)] != abi.ReservedPrefix {
			return fmt.Errorf("runtime symbol %s does not use reserved prefix %s", symbol, abi.ReservedPrefix)
		}
	}
	return nil
}

func sameStringSet(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func isSortedStrings(values []string) bool {
	return sort.StringsAreSorted(values)
}

func extractTargetTriples(targets []targetManifest) []string {
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		out = append(out, target.Triple)
	}
	return out
}

func extractBuiltinNames(builtins []builtinManifest) []string {
	out := make([]string, 0, len(builtins))
	for _, builtin := range builtins {
		out = append(out, builtin.Name)
	}
	return out
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
