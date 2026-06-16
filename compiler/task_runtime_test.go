package compiler

import (
	"strconv"
	"strings"
	"testing"

	"tetra_language/compiler/internal/runtimeabi"
)

func TestSelectRuntimeModeStabilizationMatrix(t *testing.T) {
	for _, tc := range []struct {
		name      string
		requested RuntimeMode
		usage     runtimeUsageProfile
		want      RuntimeMode
		wantErr   string
	}{
		{
			name:      "auto_actor_only_uses_selfhost",
			requested: RuntimeAuto,
			want:      RuntimeSelfHost,
		},
		{
			name:      "auto_multi_spawn_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{actorSpawnCount: 2},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_actor_state_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{actorStateUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_task_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{tasksUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_task_group_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{taskGroupsUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_typed_task_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 4},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_staged_typed_task_slots_use_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 8},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_time_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{timeRuntimeUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_filesystem_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{filesystemUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_networking_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{netUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "explicit_selfhost_actor_only_allowed",
			requested: RuntimeSelfHost,
			want:      RuntimeSelfHost,
		},
		{
			name:      "explicit_selfhost_rejects_typed_tasks",
			requested: RuntimeSelfHost,
			usage:     runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 4},
			wantErr:   "self-host runtime does not support typed task handles",
		},
		{
			name:      "explicit_selfhost_rejects_multi_spawn",
			requested: RuntimeSelfHost,
			usage:     runtimeUsageProfile{actorSpawnCount: 2},
			wantErr:   "self-host runtime supports at most one spawned actor",
		},
		{
			name:      "explicit_builtin_allowed",
			requested: RuntimeBuiltin,
			usage:     runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 8, timeRuntimeUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "invalid_runtime_rejected",
			requested: RuntimeMode(99),
			wantErr:   "unsupported runtime mode: 99",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := selectRuntimeMode(tc.requested, tc.usage)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want contains %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("selectRuntimeMode returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("selectRuntimeMode = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRuntimeModeForLinuxX32AutoUsesSelfHostWhenSupported(t *testing.T) {
	usage := runtimeUsageProfile{timeRuntimeUsed: true}
	got, err := runtimeModeForNativeTarget("linux-x32", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 auto runtime mode = %v, want self-host for supported usage", got)
	}

	got, err = runtimeModeForNativeTarget("linux-x32", RuntimeBuiltin, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("explicit builtin runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeBuiltin {
		t.Fatalf("x32 explicit builtin runtime mode = %v, want builtin to preserve explicit diagnostic", got)
	}

	got, err = runtimeModeForNativeTarget("linux-x64", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x64 runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeBuiltin {
		t.Fatalf("x64 auto runtime mode = %v, want existing builtin preference", got)
	}

	usage = runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 4, actorSpawnCount: 1}
	got, err = runtimeModeForNativeTarget("linux-x32", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x32 typed task runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 typed task auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x32", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x32 explicit self-host typed task selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 explicit self-host typed task runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 4, actorSpawnCount: 2}
	got, err = runtimeModeForNativeTarget("linux-x32", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x32 multi-spawn typed task runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 multi-spawn typed task auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x32", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x32 explicit self-host multi-spawn typed task selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 explicit self-host multi-spawn typed task runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{taskGroupsUsed: true, actorSpawnCount: 1}
	got, err = runtimeModeForNativeTarget("linux-x32", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x32 task-group runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 task-group auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x32", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x32 explicit self-host task-group selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 explicit self-host task-group runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{taskGroupsUsed: true, typedTasksUsed: true, typedTaskMaxSlots: 4, actorSpawnCount: 1}
	got, err = runtimeModeForNativeTarget("linux-x32", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x32 typed task-group runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 typed task-group auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x32", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x32 explicit self-host typed task-group selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 explicit self-host typed task-group runtime mode = %v, want self-host", got)
	}
}

func TestTargetSpecificSelfHostMultiSpawnOverrideStaysLinuxILP32Only(t *testing.T) {
	usage := runtimeUsageProfile{actorSpawnCount: 2}
	for _, target := range []string{"linux-x64", "macos-x64", "windows-x64"} {
		if got, err := selectRuntimeModeForNativeTarget(target, RuntimeSelfHost, usage); err == nil || !strings.Contains(err.Error(), "self-host runtime supports at most one spawned actor") {
			t.Fatalf("%s explicit self-host two-spawn selection = mode %v err %v, want generic one-spawn diagnostic", target, got, err)
		}
	}
}

func TestRuntimeModeForLinuxX86AutoUsesSelfHostWhenSupported(t *testing.T) {
	usage := runtimeUsageProfile{taskGroupsUsed: true, actorSpawnCount: 1}
	got, err := runtimeModeForNativeTarget("linux-x86", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x86 task-group runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 task-group auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x86", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x86 explicit self-host task-group selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 explicit self-host task-group runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 4, actorSpawnCount: 1}
	got, err = runtimeModeForNativeTarget("linux-x86", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x86 typed task runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 typed task auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x86", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x86 explicit self-host typed task selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 explicit self-host typed task runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 8, actorSpawnCount: 1}
	got, err = runtimeModeForNativeTarget("linux-x86", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x86 staged typed task runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 staged typed task auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x86", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x86 explicit self-host staged typed task selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 explicit self-host staged typed task runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{taskGroupsUsed: true, actorSpawnCount: 2}
	got, err = runtimeModeForNativeTarget("linux-x86", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x86 multi-spawn task-group runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 multi-spawn task-group auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x86", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x86 explicit self-host multi-spawn task-group selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 explicit self-host multi-spawn task-group runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{taskGroupsUsed: true, typedTasksUsed: true, typedTaskMaxSlots: 4, actorSpawnCount: 1}
	got, err = runtimeModeForNativeTarget("linux-x86", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x86 typed task-group runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 typed task-group auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x86", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x86 explicit self-host typed task-group selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 explicit self-host typed task-group runtime mode = %v, want self-host", got)
	}
}

func TestNativeRuntimeCapabilityTableDocumentsCurrentLinuxFamily(t *testing.T) {
	x64 := nativeRuntimeCapabilitiesForTarget("linux-x64")
	if !x64.actors || !x64.actorState || !x64.tasks || !x64.taskGroups || !x64.typedTasks || !x64.time || !x64.filesystem || !x64.networking || !x64.surface || !x64.distributedActors || x64.maxActorSpawns != unlimitedActorSpawns || !x64.builtinRuntime || !x64.selfHostActorsRuntime {
		t.Fatalf("linux-x64 runtime capabilities = %#v", x64)
	}

	x32 := nativeRuntimeCapabilitiesForTarget("linux-x32")
	if !x32.actors || !x32.actorState || !x32.tasks || !x32.taskGroups || !x32.typedTasks || !x32.time || !x32.filesystem || !x32.networking || x32.surface || x32.distributedActors || x32.maxActorSpawns != 2 || x32.maxTypedTaskSlots != 8 || x32.builtinRuntime || !x32.selfHostActorsRuntime {
		t.Fatalf("linux-x32 runtime capabilities = %#v", x32)
	}

	x86 := nativeRuntimeCapabilitiesForTarget("linux-x86")
	if !x86.actors || !x86.actorState || !x86.tasks || !x86.taskGroups || !x86.typedTasks || !x86.time || !x86.timeOnlyWithoutScheduler || !x86.filesystem || !x86.networking || x86.surface || x86.distributedActors || x86.maxActorSpawns != 2 || x86.maxTypedTaskSlots != 8 || x86.builtinRuntime || !x86.selfHostActorsRuntime || !x86.selfHostTimeRuntime {
		t.Fatalf("linux-x86 runtime capabilities = %#v", x86)
	}
}

func TestTaskSpawnI32CollectsRuntimeEntry(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    if task.value != 1:
        return 60 + task.value
    if task.error != 0:
        return 70 + task.error
    return core.task_join_i32(task)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	used, entries, _, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collectActorEntries: %v", err)
	}
	if !used {
		t.Fatalf("task runtime was not collected")
	}
	if len(entries) != 2 || entries[0] != "main" || entries[1] != "worker" {
		t.Fatalf("entries = %#v, want [main worker]", entries)
	}
}

func TestRequiredTypedTaskRuntimeSymbolsSupportStagedSlotsUpToEight(t *testing.T) {
	base := map[string]struct{}{}
	for _, name := range requiredTypedTaskRuntimeSymbols(4) {
		base[name] = struct{}{}
	}
	for _, slots := range []int{2, 3, 4} {
		name := "__tetra_task_join_typed_" + strconv.Itoa(slots)
		if _, ok := base[name]; !ok {
			t.Fatalf("missing required typed task runtime symbol %q", name)
		}
	}
	if _, ok := base["__tetra_task_result_get"]; ok {
		t.Fatalf("non-staged symbol set should not require __tetra_task_result_get")
	}

	got := map[string]struct{}{}
	for _, name := range requiredTypedTaskRuntimeSymbols(8) {
		got[name] = struct{}{}
	}
	for _, slots := range []int{2, 3, 4, 5, 6, 7, 8} {
		name := "__tetra_task_join_typed_" + strconv.Itoa(slots)
		if _, ok := got[name]; !ok {
			t.Fatalf("missing required typed task runtime symbol %q", name)
		}
	}
	if _, ok := got["__tetra_task_result_get"]; !ok {
		t.Fatalf("missing required staged typed task runtime symbol %q", "__tetra_task_result_get")
	}
}

func TestRequiredTypedTaskRuntimeSymbolsClampToSupportedABIEnvelope(t *testing.T) {
	low := requiredTypedTaskRuntimeSymbols(0)
	if got, want := strings.Join(low, ","), strings.Join(requiredTypedTaskRuntimeSymbols(2), ","); got != want {
		t.Fatalf("low slot clamp = %q, want %q", got, want)
	}
	for _, forbidden := range []string{"__tetra_task_join_typed_0", "__tetra_task_join_typed_1", "__tetra_task_result_get"} {
		if containsString(low, forbidden) {
			t.Fatalf("low slot ABI unexpectedly contains %q: %#v", forbidden, low)
		}
	}

	high := requiredTypedTaskRuntimeSymbols(99)
	if got, want := strings.Join(high, ","), strings.Join(requiredTypedTaskRuntimeSymbols(8), ","); got != want {
		t.Fatalf("high slot clamp = %q, want %q", got, want)
	}
	for _, required := range []string{"__tetra_task_join_typed_8", "__tetra_task_result_get"} {
		if !containsString(high, required) {
			t.Fatalf("high slot ABI missing %q: %#v", required, high)
		}
	}
	if containsString(high, "__tetra_task_join_typed_9") {
		t.Fatalf("high slot ABI exceeded supported envelope: %#v", high)
	}
}

func TestValidateTypedTaskRuntimeObjectRejectsMissingStagedSymbols(t *testing.T) {
	obj := &Object{}
	for _, name := range requiredTypedTaskRuntimeSymbols(4) {
		obj.Symbols = append(obj.Symbols, Symbol{Name: name})
	}
	if err := validateTypedTaskRuntimeObject(obj, 8); err == nil {
		t.Fatalf("expected missing staged typed task runtime symbol failure")
	} else if !strings.Contains(err.Error(), "__tetra_task_result_get") {
		t.Fatalf("unexpected typed task runtime validation error: %v", err)
	}
}

func TestValidateTaskRuntimeObjectChecksSignatureMetadata(t *testing.T) {
	t.Run("correct metadata passes", func(t *testing.T) {
		obj := runtimeObjectWithTaskRuntimeSignatures()
		if err := validateTaskRuntimeObject(obj); err != nil {
			t.Fatalf("validate task runtime object: %v", err)
		}
	})

	t.Run("wrong arity fails", func(t *testing.T) {
		obj := runtimeObjectWithTaskRuntimeSignatures()
		replaceRuntimeSymbolSignature(obj, "__tetra_task_join_i32", 1, 1)
		err := validateTaskRuntimeObject(obj)
		if err == nil {
			t.Fatalf("expected wrong arity failure")
		}
		if !strings.Contains(err.Error(), "runtime object symbol '__tetra_task_join_i32' signature mismatch") ||
			!strings.Contains(err.Error(), "params=1 want=2") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wrong return slot count fails", func(t *testing.T) {
		obj := runtimeObjectWithTaskRuntimeSignatures()
		replaceRuntimeSymbolSignature(obj, "__tetra_task_join_result_i32", 2, 1)
		err := validateTaskRuntimeObject(obj)
		if err == nil {
			t.Fatalf("expected wrong return slot failure")
		}
		if !strings.Contains(err.Error(), "runtime object symbol '__tetra_task_join_result_i32' signature mismatch") ||
			!strings.Contains(err.Error(), "returns=1 want=2") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("legacy symbols without metadata remain compatible", func(t *testing.T) {
		obj := &Object{}
		for _, name := range requiredTaskRuntimeSymbols() {
			obj.Symbols = append(obj.Symbols, Symbol{Name: name})
		}
		if err := validateTaskRuntimeObject(obj); err != nil {
			t.Fatalf("legacy task runtime object should remain compatible: %v", err)
		}
	})
}

func runtimeObjectWithTaskRuntimeSignatures() *Object {
	obj := &Object{}
	for _, name := range requiredTaskRuntimeSymbols() {
		sig, ok := runtimeObjectSignature(name)
		if !ok {
			panic("missing task runtime signature for " + name)
		}
		obj.Symbols = append(obj.Symbols, Symbol{
			Name:         name,
			HasSignature: true,
			ParamSlots:   sig.paramSlots,
			ReturnSlots:  sig.returnSlots,
		})
	}
	return obj
}

func TestRuntimeObjectSignatureUsesSharedRuntimeABI(t *testing.T) {
	names := append([]string{}, requiredActorRuntimeSymbols()...)
	names = append(names, requiredActorStateRuntimeSymbols()...)
	names = append(names, requiredDistributedActorRuntimeSymbols()...)
	names = append(names, requiredTaskRuntimeSymbols()...)
	names = append(names, requiredTaskGroupRuntimeSymbols()...)
	names = append(names, requiredTypedTaskRuntimeSymbols(8)...)
	names = append(names, requiredTimeRuntimeSymbols()...)
	names = append(names, requiredFilesystemRuntimeSymbols()...)
	names = append(names, requiredSurfaceRuntimeSymbols()...)

	for _, name := range names {
		objectSig, ok := runtimeObjectSignature(name)
		if !ok {
			t.Fatalf("missing runtime object signature for %q", name)
		}
		sharedSig, ok := runtimeabi.SignatureForSymbol(name)
		if !ok {
			t.Fatalf("missing shared runtime ABI signature for %q", name)
		}
		if sharedSig.ParamSlots != objectSig.paramSlots || sharedSig.ReturnSlots != objectSig.returnSlots {
			t.Fatalf("%s ABI mismatch: shared params=%d returns=%d object params=%d returns=%d", name, sharedSig.ParamSlots, sharedSig.ReturnSlots, objectSig.paramSlots, objectSig.returnSlots)
		}
	}
}

func replaceRuntimeSymbolSignature(obj *Object, name string, paramSlots int, returnSlots int) {
	for i := range obj.Symbols {
		if obj.Symbols[i].Name == name {
			obj.Symbols[i].HasSignature = true
			obj.Symbols[i].ParamSlots = paramSlots
			obj.Symbols[i].ReturnSlots = returnSlots
			return
		}
	}
	panic("missing runtime symbol " + name)
}

func TestRequiredTaskRuntimeSymbolsIncludeDeadlineAndCancellationABI(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredTaskRuntimeSymbols() {
		got[name] = struct{}{}
	}
	for _, name := range []string{
		"__tetra_task_join_until_i32",
		"__tetra_task_poll_i32",
		"__tetra_task_is_canceled",
		"__tetra_task_checkpoint",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required task runtime symbols missing deadline/cancellation ABI symbol %q", name)
		}
	}
}

func TestRequiredTaskGroupRuntimeSymbolsIncludeCancellationABI(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredTaskGroupRuntimeSymbols() {
		got[name] = struct{}{}
	}
	for _, name := range []string{
		"__tetra_task_group_open",
		"__tetra_task_group_close",
		"__tetra_task_group_cancel",
		"__tetra_task_group_current",
		"__tetra_task_group_status",
		"__tetra_task_spawn_group_i32",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required task group runtime symbols missing ABI symbol %q", name)
		}
	}
}

func TestValidateTaskGroupRuntimeObjectRejectsMissingCancellationSymbols(t *testing.T) {
	obj := &Object{}
	for _, name := range requiredTaskGroupRuntimeSymbols() {
		if name == "__tetra_task_group_cancel" {
			continue
		}
		obj.Symbols = append(obj.Symbols, Symbol{Name: name})
	}
	if err := validateTaskGroupRuntimeObject(obj); err == nil {
		t.Fatalf("expected missing task group cancellation symbol failure")
	} else if !strings.Contains(err.Error(), "__tetra_task_group_cancel") {
		t.Fatalf("unexpected task group runtime validation error: %v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
