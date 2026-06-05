package compiler

import (
	"reflect"
	"strings"
	"testing"
)

func TestManifestRuntimeABIIncludesFullRequiredSymbolSets(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}

	assertSymbolSequence(t, "actors_required_symbols", manifest.RuntimeABI.ActorsRequiredSymbols, requiredActorRuntimeSymbols())
	assertSymbolSequence(t, "actor_state_required_symbols", manifest.RuntimeABI.ActorStateRequiredSymbols, requiredActorStateRuntimeSymbols())
	assertSymbolSequence(t, "task_required_symbols", manifest.RuntimeABI.TaskRequiredSymbols, requiredTaskRuntimeSymbols())
	assertSymbolSequence(t, "task_group_required_symbols", manifest.RuntimeABI.TaskGroupRequiredSymbols, requiredTaskGroupRuntimeSymbols())
	assertSymbolSequence(t, "typed_task_required_symbols", manifest.RuntimeABI.TypedTaskRequiredSymbols, requiredTypedTaskRuntimeSymbols(8))
	assertSymbolSequence(t, "time_required_symbols", manifest.RuntimeABI.TimeRequiredSymbols, requiredTimeRuntimeSymbols())
	assertSymbolSequence(t, "filesystem_required_symbols", manifest.RuntimeABI.FilesystemRequiredSymbols, requiredFilesystemRuntimeSymbols())
	assertSymbolSequence(t, "net_required_symbols", manifest.RuntimeABI.NetRequiredSymbols, requiredNetRuntimeSymbols())
	assertSymbolSequence(t, "surface_required_symbols", manifest.RuntimeABI.SurfaceRequiredSymbols, requiredSurfaceRuntimeSymbols())
}

func TestManifestIncludesLinuxNativePromotionGateMetadata(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	byTriple := map[string]TargetManifest{}
	for _, target := range manifest.Targets {
		byTriple[target.Triple] = target
	}
	for _, tc := range []struct {
		triple        string
		runtimeStatus string
		stdlibStatus  string
		ffiStatus     string
		artifact      string
	}{
		{triple: "linux-x64", runtimeStatus: "production", stdlibStatus: "production", ffiStatus: "scalar_object_smokes_partial", artifact: "linux-x64-runner.json"},
		{triple: "linux-x86", runtimeStatus: "partial_build_only", stdlibStatus: "partial_build_only", ffiStatus: "ilp32_scalar_object_smokes_partial", artifact: "linux-x86-runner.json"},
		{triple: "linux-x32", runtimeStatus: "partial_build_only", stdlibStatus: "partial_build_only", ffiStatus: "ilp32_scalar_object_smokes_partial", artifact: "linux-x32-runner.json"},
	} {
		got := byTriple[tc.triple]
		if got.Triple == "" {
			t.Fatalf("manifest missing target %s", tc.triple)
		}
		if got.RuntimeStatus != tc.runtimeStatus || got.StdlibStatus != tc.stdlibStatus || got.FFIStatus != tc.ffiStatus {
			t.Fatalf("%s promotion metadata = runtime:%q stdlib:%q ffi:%q", tc.triple, got.RuntimeStatus, got.StdlibStatus, got.FFIStatus)
		}
		if got.RunnerProbeCommand == "" || got.ReleaseGate != "scripts/release/post_v0_4/linux-native-targets-smoke.sh" || !stringSliceContains(got.EvidenceArtifacts, tc.artifact) {
			t.Fatalf("%s evidence metadata = runner:%q gate:%q artifacts:%#v", tc.triple, got.RunnerProbeCommand, got.ReleaseGate, got.EvidenceArtifacts)
		}
	}
}

func TestManifestIncludesMemoryCapabilityMatrixMetadata(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	byTriple := map[string]TargetManifest{}
	for _, target := range manifest.Targets {
		byTriple[target.Triple] = target
	}
	for _, tc := range []struct {
		triple             string
		run                string
		rawDiagnostics     string
		regionLowering     string
		alignmentSemantics string
		claimLevel         string
	}{
		{"linux-x64", "yes", "yes", "yes/partial", "yes", "production/host_runtime"},
		{"linux-x86", "no/host-dependent", "partial", "partial", "partial", "build_lower_only"},
		{"linux-x32", "no/host-dependent", "partial", "partial", "special", "build_lower_only"},
		{"macos-x64", "host-required", "host-required", "host-required", "host-required", "build_lower_only unless run"},
		{"windows-x64", "host-required", "host-required", "host-required", "host-required", "build_lower_only unless run"},
		{"wasm32-wasi", "runner-smoke if available", "safe-only", "limited", "wasm rules", "artifact/runtime tiered"},
		{"wasm32-web", "browser-smoke if available", "safe-only", "limited", "wasm rules", "artifact/runtime tiered"},
	} {
		got := byTriple[tc.triple]
		if got.MemoryBuild != "yes" || got.MemoryLower != "yes" || got.MemoryRun != tc.run ||
			got.MemoryRawDiagnostics != tc.rawDiagnostics || got.MemoryRegionLowering != tc.regionLowering ||
			got.MemoryAlignmentSemantics != tc.alignmentSemantics || got.MemoryClaimLevel != tc.claimLevel {
			t.Fatalf("%s memory capability metadata = build:%q lower:%q run:%q raw:%q region:%q alignment:%q claim:%q",
				tc.triple, got.MemoryBuild, got.MemoryLower, got.MemoryRun, got.MemoryRawDiagnostics,
				got.MemoryRegionLowering, got.MemoryAlignmentSemantics, got.MemoryClaimLevel)
		}
	}
}

func TestManifestIncludesLinuxNativeSyscallPackMetadata(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	byTriple := map[string]TargetManifest{}
	for _, target := range manifest.Targets {
		byTriple[target.Triple] = target
	}
	for _, tc := range []struct {
		triple      string
		instruction string
		numbering   string
		registers   []string
	}{
		{triple: "linux-x64", instruction: "syscall", numbering: "x86_64", registers: []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"}},
		{triple: "linux-x86", instruction: "int 0x80", numbering: "i386", registers: []string{"eax", "ebx", "ecx", "edx", "esi", "edi", "ebp"}},
		{triple: "linux-x32", instruction: "syscall", numbering: "x32_syscall_bit", registers: []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"}},
	} {
		got := byTriple[tc.triple]
		if got.SyscallInstruction != tc.instruction || got.SyscallNumbering != tc.numbering || got.SyscallErrorRange != "-4095..-1" || !reflect.DeepEqual(got.SyscallArgRegisters, tc.registers) {
			t.Fatalf("%s syscall metadata = instruction:%q numbering:%q regs:%#v error:%q", tc.triple, got.SyscallInstruction, got.SyscallNumbering, got.SyscallArgRegisters, got.SyscallErrorRange)
		}
	}
}

func assertSymbolSequence(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}

func stringSliceContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestManifestBuiltinsExposeStableUnsafePoliciesForPublicSurface(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}

	byName := map[string]BuiltinManifest{}
	for _, builtin := range manifest.Builtins {
		byName[builtin.Name] = builtin
	}

	for name, wantEffects := range map[string]string{
		"core.cap_io":         "capability,io",
		"core.cap_mem":        "capability,mem",
		"core.load_i32":       "mem",
		"core.store_i32":      "mem",
		"core.mmio_read_i32":  "io,mmio",
		"core.mmio_write_i32": "io,mmio",
	} {
		got, ok := byName[name]
		if !ok {
			t.Fatalf("manifest missing builtin %s", name)
		}
		if got.UnsafePolicy != "always" {
			t.Fatalf("%s unsafe_policy = %q, want always", name, got.UnsafePolicy)
		}
		if got.UnsafeDetails != "" {
			t.Fatalf("%s unsafe_details = %q, want empty", name, got.UnsafeDetails)
		}
		if strings.Join(got.Effects, ",") != wantEffects {
			t.Fatalf("%s effects = %q, want %q", name, strings.Join(got.Effects, ","), wantEffects)
		}
	}

	const wantConditionalUnsafeDetails = "requires unsafe when the island argument is not a scoped island variable"
	for _, name := range []string{
		"core.island_make_u8",
		"core.island_make_u16",
		"core.island_make_i32",
		"core.island_make_bool",
	} {
		got, ok := byName[name]
		if !ok {
			t.Fatalf("manifest missing builtin %s", name)
		}
		if got.UnsafePolicy != "conditional" {
			t.Fatalf("%s unsafe_policy = %q, want conditional", name, got.UnsafePolicy)
		}
		if got.UnsafeDetails != wantConditionalUnsafeDetails {
			t.Fatalf("%s unsafe_details = %q, want %q", name, got.UnsafeDetails, wantConditionalUnsafeDetails)
		}
	}
}
