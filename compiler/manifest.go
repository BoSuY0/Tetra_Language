package compiler

import (
	"fmt"

	"tetra_language/compiler/internal/formats"
	"tetra_language/compiler/internal/semantics"
	ctarget "tetra_language/compiler/target"
)

type Manifest struct {
	CompilerVersion string            `json:"compiler_version"`
	Formats         []FormatManifest  `json:"formats"`
	Targets         []TargetManifest  `json:"targets"`
	Builtins        []BuiltinManifest `json:"builtins"`
	RuntimeABI      RuntimeManifest   `json:"runtime_abi"`
	Features        []FeatureInfo     `json:"features"`
}

type FormatManifest = formats.Info

type TargetManifest struct {
	Triple                   string   `json:"triple"`
	Status                   string   `json:"status"`
	OS                       string   `json:"os"`
	Arch                     string   `json:"arch"`
	ABI                      string   `json:"abi"`
	DataModel                string   `json:"data_model"`
	Format                   string   `json:"format"`
	ExeExt                   string   `json:"exe_ext"`
	CollectImports           bool     `json:"collect_imports"`
	RunMode                  string   `json:"run_mode"`
	UIRuntimeContract        string   `json:"ui_runtime_contract,omitempty"`
	UIRuntimeStatus          string   `json:"ui_runtime_status"`
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

type BuiltinManifest struct {
	Name          string   `json:"name"`
	Aliases       []string `json:"aliases,omitempty"`
	ParamTypes    []string `json:"param_types,omitempty"`
	ReturnType    string   `json:"return_type"`
	Effects       []string `json:"effects,omitempty"`
	UnsafePolicy  string   `json:"unsafe_policy"`
	UnsafeDetails string   `json:"unsafe_details,omitempty"`
}

type RuntimeManifest struct {
	ReservedPrefix            string   `json:"reserved_prefix"`
	ActorsSupportedTargets    []string `json:"actors_supported_targets"`
	ActorsRequiredSymbols     []string `json:"actors_required_symbols"`
	ActorStateRequiredSymbols []string `json:"actor_state_required_symbols"`
	TaskRequiredSymbols       []string `json:"task_required_symbols"`
	TaskGroupRequiredSymbols  []string `json:"task_group_required_symbols"`
	TypedTaskRequiredSymbols  []string `json:"typed_task_required_symbols"`
	TimeRequiredSymbols       []string `json:"time_required_symbols,omitempty"`
	FilesystemRequiredSymbols []string `json:"filesystem_required_symbols,omitempty"`
	NetRequiredSymbols        []string `json:"net_required_symbols,omitempty"`
	SurfaceRequiredSymbols    []string `json:"surface_required_symbols,omitempty"`
	ActorsProgramGlueSymbols  []string `json:"actors_program_glue_symbols"`
}

func GetManifest() (Manifest, error) {
	builtins, err := semantics.DescribeBuiltins()
	if err != nil {
		return Manifest{}, err
	}
	builtinOut := make([]BuiltinManifest, 0, len(builtins))
	for _, b := range builtins {
		builtinOut = append(builtinOut, BuiltinManifest{
			Name:          b.Name,
			Aliases:       append([]string(nil), b.Aliases...),
			ParamTypes:    append([]string(nil), b.ParamTypes...),
			ReturnType:    b.ReturnType,
			Effects:       append([]string(nil), b.Effects...),
			UnsafePolicy:  b.UnsafePolicy,
			UnsafeDetails: b.UnsafeDetails,
		})
	}

	targets := ctarget.AllBuildable()
	targetOut := make([]TargetManifest, 0, len(targets))
	for _, t := range targets {
		targetOut = append(targetOut, TargetManifest{
			Triple:                   t.Triple,
			Status:                   fmt.Sprint(t.Status),
			OS:                       fmt.Sprint(t.OS),
			Arch:                     fmt.Sprint(t.Arch),
			ABI:                      fmt.Sprint(t.ABI),
			DataModel:                fmt.Sprint(t.DataModel),
			Format:                   fmt.Sprint(t.Format),
			ExeExt:                   t.ExeExt,
			CollectImports:           t.CollectImports,
			RunMode:                  fmt.Sprint(t.RunMode),
			UIRuntimeContract:        ctarget.UIRuntimeContract(t.Triple),
			UIRuntimeStatus:          ctarget.UIRuntimeStatus(t.Triple),
			UIRuntimeEvidence:        ctarget.UIRuntimeEvidence(t.Triple),
			PointerWidthBits:         t.PointerWidthBits,
			RegisterWidthBits:        t.RegisterWidthBits,
			NativeIntWidthBits:       t.NativeIntWidthBits,
			Endian:                   fmt.Sprint(t.Endian),
			StackAlignmentBytes:      t.StackAlignmentBytes,
			MaxAtomicWidthBits:       t.MaxAtomicWidthBits,
			AtomicWidthBits:          t.AtomicWidthBits(),
			AtomicPointerWidthBits:   manifestAtomicPointerWidthBits(t),
			UnsupportedReason:        t.UnsupportedReason,
			RuntimeStatus:            t.RuntimeStatus,
			StdlibStatus:             t.StdlibStatus,
			FFIStatus:                t.FFIStatus,
			MemoryBuild:              t.MemoryBuild,
			MemoryLower:              t.MemoryLower,
			MemoryRun:                t.MemoryRun,
			MemoryRawDiagnostics:     t.MemoryRawDiagnostics,
			MemoryRegionLowering:     t.MemoryRegionLowering,
			MemoryAlignmentSemantics: t.MemoryAlignmentSemantics,
			MemoryClaimLevel:         t.MemoryClaimLevel,
			RunnerProbeCommand:       t.RunnerProbeCommand,
			ReleaseGate:              t.ReleaseGate,
			EvidenceArtifacts:        append([]string(nil), t.EvidenceArtifacts...),
			SyscallInstruction:       t.SyscallInstruction,
			SyscallNumbering:         t.SyscallNumbering,
			SyscallArgRegisters:      append([]string(nil), t.SyscallArgRegisters...),
			SyscallErrorRange:        t.SyscallErrorRange,
			SupportsDebugInfo:        t.SupportsDebugInfo,
			SupportsReleaseOptimize:  t.SupportsReleaseOptimize,
		})
	}

	return Manifest{
		CompilerVersion: Version(),
		Formats:         formats.All(),
		Targets:         targetOut,
		Builtins:        builtinOut,
		RuntimeABI: RuntimeManifest{
			ReservedPrefix:            "__tetra_",
			ActorsSupportedTargets:    []string{"linux-x64", "macos-x64", "windows-x64"},
			ActorsRequiredSymbols:     requiredActorRuntimeSymbols(),
			ActorStateRequiredSymbols: requiredActorStateRuntimeSymbols(),
			TaskRequiredSymbols:       requiredTaskRuntimeSymbols(),
			TaskGroupRequiredSymbols:  requiredTaskGroupRuntimeSymbols(),
			TypedTaskRequiredSymbols:  requiredTypedTaskRuntimeSymbols(8),
			TimeRequiredSymbols:       requiredTimeRuntimeSymbols(),
			FilesystemRequiredSymbols: requiredFilesystemRuntimeSymbols(),
			NetRequiredSymbols:        requiredNetRuntimeSymbols(),
			SurfaceRequiredSymbols:    requiredSurfaceRuntimeSymbols(),
			ActorsProgramGlueSymbols: []string{
				"__tetra_actor_dispatch",
				"__tetra_actor_main_entry_id",
			},
		},
		Features: FeatureRegistry(),
	}, nil
}

func manifestAtomicPointerWidthBits(t ctarget.Target) int {
	layout, err := t.AtomicPointerLayout()
	if err != nil {
		return 0
	}
	return layout.WidthBits
}
