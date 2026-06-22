package semantics

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/runtimeabi"
)

// ---- actor_state_mvp_test.go ----

func TestCheckActorStateBuildsSlotMapping(t *testing.T) {
	src := []byte(`
actor Worker:
    var count: Int = 0
    val step: Int = 2
    const enabled: Bool = true
    func run() -> Int:
        if enabled:
            count = count + step
        return count

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	var run CheckedFunc
	found := false
	for _, fn := range checked.Funcs {
		if fn.Name == "Worker.run" {
			run = fn
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing checked function Worker.run")
	}
	if len(run.ActorState) != 3 {
		t.Fatalf("actor state count = %d, want 3", len(run.ActorState))
	}

	count := run.ActorState["count"]
	if count.Slot != 0 || !count.Mutable || count.Const || count.TypeName != "i32" ||
		count.Init != 0 {
		t.Fatalf("count field = %#v", count)
	}
	step := run.ActorState["step"]
	if step.Slot != 1 || step.Mutable || step.Const || step.TypeName != "i32" || step.Init != 2 {
		t.Fatalf("step field = %#v", step)
	}
	enabled := run.ActorState["enabled"]
	if enabled.Slot != 2 || enabled.Mutable || !enabled.Const || enabled.TypeName != "bool" ||
		enabled.Init != 1 {
		t.Fatalf("enabled field = %#v", enabled)
	}
}

func TestCheckActorStateRejectsUnsupportedType(t *testing.T) {
	src := []byte(`
actor Worker:
    val title: String = "worker"
    func run() -> Int:
        return 0

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected actor state type diagnostic")
	}
	if !strings.Contains(
		err.Error(),
		("actor state field 'title' type 'str' is not supported; " +
			"supported actor state field types are Int, Bool, UInt8, UInt16, and " +
			"task.error"),
	) {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), "MVP") {
		t.Fatalf("error = %v, want stable non-versioned diagnostic", err)
	}
}

func TestCheckActorStateSupportsExtendedScalarTypes(t *testing.T) {
	src := []byte(`
actor Worker:
    var err: task.error = 0
    val byteStep: UInt8 = 7
    const wide: UInt16 = 9
    func run() -> Int:
        err = err + 1
        return err + byteStep + wide

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	var run CheckedFunc
	found := false
	for _, fn := range checked.Funcs {
		if fn.Name == "Worker.run" {
			run = fn
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing checked function Worker.run")
	}
	if len(run.ActorState) != 3 {
		t.Fatalf("actor state count = %d, want 3", len(run.ActorState))
	}

	errField := run.ActorState["err"]
	if errField.Slot != 0 || !errField.Mutable || errField.Const ||
		errField.TypeName != "task.error" ||
		errField.Init != 0 {
		t.Fatalf("err field = %#v", errField)
	}
	byteStep := run.ActorState["byteStep"]
	if byteStep.Slot != 1 || byteStep.Mutable || byteStep.Const || byteStep.TypeName != "u8" ||
		byteStep.Init != 7 {
		t.Fatalf("byteStep field = %#v", byteStep)
	}
	wide := run.ActorState["wide"]
	if wide.Slot != 2 || wide.Mutable || !wide.Const || wide.TypeName != "u16" || wide.Init != 9 {
		t.Fatalf("wide field = %#v", wide)
	}
}

func TestCheckActorStateRejectsPtrType(t *testing.T) {
	src := []byte(`
actor Worker:
    val raw: ptr = 0
    func run() -> Int:
        return 0

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected actor state type diagnostic")
	}
	if !strings.Contains(
		err.Error(),
		("actor state field 'raw' type 'ptr' is not supported; supported " +
			"actor state field types are Int, Bool, UInt8, UInt16, and task.error"),
	) {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), "MVP") {
		t.Fatalf("error = %v, want stable non-versioned diagnostic", err)
	}
}

func TestCheckActorStateStableDiagnosticsMatrix(t *testing.T) {
	cases := []struct {
		name       string
		src        string
		want       string
		rejectText string
	}{
		{
			name: "unsupported field type",
			src: `
actor Worker:
    val title: String = "worker"
    func run() -> Int:
        return 0

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`,
			want: ("actor state field 'title' type 'str' is not supported; " +
				"supported actor state field types are Int, Bool, UInt8, UInt16, and " +
				"task.error"),
			rejectText: "MVP",
		},
		{
			name: "dynamic initializer",
			src: `
actor Worker:
    val count: Int = core.recv()
    func run() -> Int:
        return count

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`,
			want: ("actor state field 'count' initializer must be a compile-time " +
				"constant Int/Bool expression"),
			rejectText: "MVP",
		},
		{
			name: "missing initializer",
			src: `
actor Worker:
    var count: Int
    func run() -> Int:
        return count

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`,
			want:       "actor state field 'count' requires a compile-time constant initializer",
			rejectText: "MVP",
		},
		{
			name: "slot count",
			src: `
actor Worker:
    val s0: Int = 0
    val s1: Int = 1
    val s2: Int = 2
    val s3: Int = 3
    val s4: Int = 4
    val s5: Int = 5
    val s6: Int = 6
    val s7: Int = 7
    val s8: Int = 8
    func run() -> Int:
        return s0

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`,
			want:       "actor 'Worker' state supports at most 8 slots, got 9",
			rejectText: "MVP",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := frontend.Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			_, err = Check(prog)
			if err == nil {
				t.Fatalf("expected actor state diagnostic containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if tt.rejectText != "" && strings.Contains(err.Error(), tt.rejectText) {
				t.Fatalf("error = %v, rejected text %q should not appear", err, tt.rejectText)
			}
		})
	}
}

// ---- array_mvp_test.go ----

func TestEnsureTypeInfoArraySupportedSubset(t *testing.T) {
	types := baseTypes()
	tests := []struct {
		name string
		elem string
		len  int
	}{
		{name: "[1]i32", elem: "i32", len: 1},
		{name: "[2]bool", elem: "bool", len: 2},
		{name: "[3]u8", elem: "u8", len: 3},
		{name: "[4]u16", elem: "u16", len: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ensureTypeInfo(tt.name, types)
			if err != nil {
				t.Fatalf("ensureTypeInfo(%q): %v", tt.name, err)
			}
			if info.Kind != TypeArray {
				t.Fatalf("kind = %v, want TypeArray", info.Kind)
			}
			if info.ElemType != tt.elem || info.ArrayLen != tt.len {
				t.Fatalf(
					"array info = elem=%q len=%d, want elem=%q len=%d",
					info.ElemType,
					info.ArrayLen,
					tt.elem,
					tt.len,
				)
			}
			if info.SlotCount != 2 {
				t.Fatalf("slot count = %d, want 2", info.SlotCount)
			}
		})
	}
}

func TestEnsureTypeInfoArrayRejectsUnsupportedSubset(t *testing.T) {
	types := baseTypes()

	if _, err := ensureTypeInfo("[0]i32", types); err == nil ||
		!strings.Contains(err.Error(), "array size must be positive constant") {
		t.Fatalf("expected positive-size error, got: %v", err)
	}

	if _, err := ensureTypeInfo("[2]str", types); err == nil ||
		!strings.Contains(err.Error(), "array element type 'str' is not supported") {
		t.Fatalf("expected unsupported-element error, got: %v", err)
	}
}

func TestActorTypeAndBuiltinSignaturesUseRuntimeABIContractSlots(t *testing.T) {
	types := baseTypes()
	wantActorSlots := runtimeabi.ActorHandleABI().RefSlots
	actorInfo := types["actor"]
	if actorInfo == nil {
		t.Fatalf("missing builtin actor type")
	}
	if actorInfo.SlotCount != wantActorSlots {
		t.Fatalf("actor slot count = %d, want ABI ref slots %d", actorInfo.SlotCount, wantActorSlots)
	}

	sigs, err := builtinFuncSigs(types)
	if err != nil {
		t.Fatalf("builtinFuncSigs: %v", err)
	}
	tests := []struct {
		name        string
		paramSlots  int
		returnSlots int
	}{
		{name: "core.spawn", paramSlots: 2, returnSlots: wantActorSlots},
		{name: "core.spawn_remote", paramSlots: 3, returnSlots: wantActorSlots},
		{name: "core.send", paramSlots: wantActorSlots + 1, returnSlots: 1},
		{name: "core.send_msg", paramSlots: wantActorSlots + 2, returnSlots: 1},
		{name: "core.send_typed", paramSlots: wantActorSlots + 1, returnSlots: 1},
		{name: "core.self", paramSlots: 0, returnSlots: wantActorSlots},
		{name: "core.sender", paramSlots: 0, returnSlots: wantActorSlots},
		{name: "core.actor_ref_local", paramSlots: 2, returnSlots: wantActorSlots},
		{name: "core.actor_ref_slot", paramSlots: wantActorSlots, returnSlots: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, ok := sigs[tt.name]
			if !ok {
				t.Fatalf("missing builtin signature")
			}
			if sig.ParamSlots != tt.paramSlots || sig.ReturnSlots != tt.returnSlots {
				t.Fatalf(
					"signature slots = params=%d returns=%d, want params=%d returns=%d",
					sig.ParamSlots,
					sig.ReturnSlots,
					tt.paramSlots,
					tt.returnSlots,
				)
			}
		})
	}
}

func TestActorLifecycleTypesAndBuiltinSignaturesUseRuntimeABIContractSlots(t *testing.T) {
	types := baseTypes()
	wantActorSlots := runtimeabi.ActorHandleABI().RefSlots
	typeSlots := map[string]int{
		"actor.status":            1,
		"actor.status_result_raw": 2,
		"actor.exit_reason":       1,
		"actor.exit":              wantActorSlots + 1,
		"actor.wait_result":       2,
		"actor.monitor":           1,
		"actor.spawn_options":     1,
	}
	for name, wantSlots := range typeSlots {
		info := types[name]
		if info == nil {
			t.Fatalf("missing lifecycle type %s", name)
		}
		if info.SlotCount != wantSlots {
			t.Fatalf("%s slot count = %d, want %d", name, info.SlotCount, wantSlots)
		}
	}

	sigs, err := builtinFuncSigs(types)
	if err != nil {
		t.Fatalf("builtinFuncSigs: %v", err)
	}
	tests := []struct {
		name        string
		paramTypes  []string
		paramSlots  int
		returnSlots int
		returnType  string
	}{
		{name: "core.actor_status", paramTypes: []string{"actor"}, paramSlots: wantActorSlots, returnSlots: 1, returnType: "actor.status"},
		{name: "core.actor_status_raw", paramTypes: []string{"actor"}, paramSlots: wantActorSlots, returnSlots: 2, returnType: "actor.status_result_raw"},
		{name: "core.actor_wait", paramTypes: []string{"actor"}, paramSlots: wantActorSlots, returnSlots: 2, returnType: "actor.wait_result"},
		{name: "core.actor_wait_until", paramTypes: []string{"actor", "i32"}, paramSlots: wantActorSlots + 1, returnSlots: 2, returnType: "actor.wait_result"},
		{name: "core.actor_stop", paramTypes: []string{"actor", "actor.exit_reason"}, paramSlots: wantActorSlots + 1, returnSlots: 1, returnType: "i32"},
		{name: "core.actor_exit_reason", paramTypes: []string{"actor"}, paramSlots: wantActorSlots, returnSlots: 1, returnType: "actor.exit_reason"},
		{name: "core.actor_link", paramTypes: []string{"actor"}, paramSlots: wantActorSlots, returnSlots: 1, returnType: "i32"},
		{name: "core.actor_unlink", paramTypes: []string{"actor"}, paramSlots: wantActorSlots, returnSlots: 1, returnType: "i32"},
		{name: "core.actor_monitor", paramTypes: []string{"actor"}, paramSlots: wantActorSlots, returnSlots: 1, returnType: "actor.monitor"},
		{name: "core.actor_demonitor", paramTypes: []string{"actor.monitor"}, paramSlots: 1, returnSlots: 1, returnType: "i32"},
		{name: "core.actor_set_trap_exit", paramTypes: []string{"i32"}, paramSlots: 1, returnSlots: 1, returnType: "i32"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, ok := sigs[tt.name]
			if !ok {
				t.Fatalf("missing builtin signature")
			}
			if sig.ParamSlots != tt.paramSlots || sig.ReturnSlots != tt.returnSlots ||
				sig.ReturnType != tt.returnType {
				t.Fatalf(
					"signature = params:%d returns:%d type:%s, want params:%d returns:%d type:%s",
					sig.ParamSlots,
					sig.ReturnSlots,
					sig.ReturnType,
					tt.paramSlots,
					tt.returnSlots,
					tt.returnType,
				)
			}
			if !reflect.DeepEqual(sig.ParamTypes, tt.paramTypes) {
				t.Fatalf("%s param types = %#v, want %#v", tt.name, sig.ParamTypes, tt.paramTypes)
			}
			runtimeName := "__tetra_" + strings.TrimPrefix(tt.name, "core.")
			rtSig, ok := runtimeabi.SignatureForSymbol(runtimeName)
			if !ok {
				t.Fatalf("missing runtime ABI signature for %s", runtimeName)
			}
			if rtSig.ParamSlots != tt.paramSlots || rtSig.ReturnSlots != tt.returnSlots {
				t.Fatalf(
					"runtime signature = params:%d returns:%d, want params:%d returns:%d",
					rtSig.ParamSlots,
					rtSig.ReturnSlots,
					tt.paramSlots,
					tt.returnSlots,
				)
			}
		})
	}
}

func TestActorLifecycleStatusIsNamedV1Enum(t *testing.T) {
	info := baseTypes()["actor.status"]
	if info == nil {
		t.Fatalf("missing actor.status type")
	}
	if info.Kind != TypeEnum {
		t.Fatalf("actor.status kind = %v, want enum", info.Kind)
	}
	want := runtimeabi.ActorLifecycleStatusNames()
	if len(info.EnumCases) != len(want) {
		t.Fatalf("actor.status cases = %#v, want %d cases", info.EnumCases, len(want))
	}
	for i, name := range want {
		got := info.EnumCases[i]
		if got.Name != name || got.Ordinal != int32(i) {
			t.Fatalf("actor.status case %d = %s/%d, want %s/%d", i, got.Name, got.Ordinal, name, i)
		}
		if _, ok := info.CaseMap[name]; !ok {
			t.Fatalf("actor.status missing case map entry %s", name)
		}
	}

	src := `func score(status: actor.status) -> Int:
    match status:
    case actor.status.starting:
        return 0
    case actor.status.ready:
        return 1
    case actor.status.running:
        return 2
    case actor.status.blocked:
        return 3
    case actor.status.sleeping:
        return 4
    case actor.status.waiting:
        return 5
    case actor.status.stopping:
        return 6
    case actor.status.exited_normal:
        return 7
    case actor.status.exited_error:
        return 8
    case actor.status.canceled:
        return 9
    case actor.status.restarting:
        return 10
    case actor.status.dead:
        return 11

func main() -> Int
uses actors:
    return score(core.actor_status(core.self()))
`
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestActorLifecycleBuiltinAliasesResolveToCoreNames(t *testing.T) {
	for _, name := range []string{
		"actor_status",
		"actor_wait",
		"actor_wait_until",
		"actor_stop",
		"actor_exit_reason",
		"actor_link",
		"actor_unlink",
		"actor_monitor",
		"actor_demonitor",
		"actor_set_trap_exit",
	} {
		got, ok := ResolveBuiltinAlias(name)
		want := "core." + name
		if !ok || got != want {
			t.Fatalf("ResolveBuiltinAlias(%q) = %q, %v; want %q, true", name, got, ok, want)
		}
	}
}

func TestActorSystemReceiveTypesAndBuiltinSignaturesUseRawContractSlots(t *testing.T) {
	types := baseTypes()
	typeSlots := map[string]int{
		"actor.monitor":         1,
		"actor.node":            2,
		"actor.system_recv_raw": 8,
	}
	for name, wantSlots := range typeSlots {
		info := types[name]
		if info == nil {
			t.Fatalf("missing system receive type %s", name)
		}
		if info.SlotCount != wantSlots {
			t.Fatalf("%s slot count = %d, want %d", name, info.SlotCount, wantSlots)
		}
	}

	raw := types["actor.system_recv_raw"]
	wantFields := []struct {
		name  string
		typ   string
		slots int
	}{
		{name: "status", typ: "i32", slots: 1},
		{name: "kind", typ: "i32", slots: 1},
		{name: "subject", typ: "actor", slots: 1},
		{name: "monitor", typ: "actor.monitor", slots: 1},
		{name: "node", typ: "actor.node", slots: 2},
		{name: "reason_kind", typ: "i32", slots: 1},
		{name: "reason_code", typ: "i32", slots: 1},
	}
	if raw.Kind != TypeStruct || raw.Repr != frontend.StructReprC {
		t.Fatalf("actor.system_recv_raw kind/repr = %v/%q, want repr(C) struct", raw.Kind, raw.Repr)
	}
	if len(raw.Fields) != len(wantFields) {
		t.Fatalf("actor.system_recv_raw fields = %#v, want %d fields", raw.Fields, len(wantFields))
	}
	for i, want := range wantFields {
		got := raw.Fields[i]
		if got.Name != want.name || got.TypeName != want.typ || got.SlotCount != want.slots {
			t.Fatalf(
				"raw field %d = {%s %s slots=%d}, want {%s %s slots=%d}",
				i,
				got.Name,
				got.TypeName,
				got.SlotCount,
				want.name,
				want.typ,
				want.slots,
			)
		}
	}

	sigs, err := builtinFuncSigs(types)
	if err != nil {
		t.Fatalf("builtinFuncSigs: %v", err)
	}
	tests := []struct {
		name       string
		params     []string
		paramSlots int
		effects    []string
	}{
		{name: "core.actor_recv_system", paramSlots: 0, effects: []string{"actors", "runtime"}},
		{name: "core.actor_recv_system_poll", paramSlots: 0, effects: []string{"actors"}},
		{name: "core.actor_recv_system_until", params: []string{"i32"}, paramSlots: 1, effects: []string{"actors", "runtime"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, ok := sigs[tt.name]
			if !ok {
				t.Fatalf("missing builtin signature")
			}
			if sig.ParamSlots != tt.paramSlots || sig.ReturnSlots != 8 ||
				sig.ReturnType != "actor.system_recv_raw" {
				t.Fatalf(
					"signature = params:%d returns:%d type:%s, want params:%d returns:%d type:actor.system_recv_raw",
					sig.ParamSlots,
					sig.ReturnSlots,
					sig.ReturnType,
					tt.paramSlots,
					8,
				)
			}
			if !reflect.DeepEqual(sig.ParamTypes, tt.params) {
				t.Fatalf("%s param types = %#v, want %#v", tt.name, sig.ParamTypes, tt.params)
			}
			if got := builtinEffects(tt.name); !reflect.DeepEqual(got, tt.effects) {
				t.Fatalf("%s effects = %#v, want %#v", tt.name, got, tt.effects)
			}
		})
	}
}

func TestActorSystemReceiveEffectsAndOpaqueConstructionDiagnostics(t *testing.T) {
	valid := `
func main() -> Int
uses actors, runtime:
    let blocking: actor.system_recv_raw = core.actor_recv_system()
    let poll: actor.system_recv_raw = core.actor_recv_system_poll()
    let timed: actor.system_recv_raw = core.actor_recv_system_until(10)
    return 0
`
	prog, err := frontend.Parse([]byte(valid))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check accepted system receive builtins with proper effects: %v", err)
	}

	missingRuntime := `
func main() -> Int
uses actors:
    let raw: actor.system_recv_raw = core.actor_recv_system()
    return raw.status
`
	if err := checkActorSystemReceiveError(t, missingRuntime); err == nil ||
		!strings.Contains(err.Error(), "uses effect 'runtime' but does not declare it") {
		t.Fatalf("blocking receive missing-runtime diagnostic = %v", err)
	}

	constructMonitor := `
func main() -> Int:
    let monitor: actor.monitor = actor.monitor{value: 1}
    return 0
`
	if err := checkActorSystemReceiveError(t, constructMonitor); err == nil ||
		!strings.Contains(err.Error(), "runtime-owned actor handle 'actor.monitor' cannot be constructed") {
		t.Fatalf("monitor construction diagnostic = %v", err)
	}

	constructRaw := `
func main() -> Int:
    let raw: actor.system_recv_raw = actor.system_recv_raw{status: 0, kind: 0, subject: core.self(), monitor: core.actor_monitor(core.self()), node: actor.node{id: 0, epoch: 0}, reason_kind: 0, reason_code: 0}
    return raw.status
`
	if err := checkActorSystemReceiveError(t, constructRaw); err == nil ||
		!strings.Contains(err.Error(), "runtime-owned actor handle 'actor.system_recv_raw' cannot be constructed") {
		t.Fatalf("raw construction diagnostic = %v", err)
	}
}

func TestLibCoreActorsSystemReceiveSurfaceChecksAgainstRawBuiltins(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	libPath := filepath.Join(root, "lib", "core", "actors", "actors.tetra")
	libSrc, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatalf("read lib.core.actors: %v", err)
	}
	libFile, err := frontend.ParseFile(libSrc, libPath)
	if err != nil {
		t.Fatalf("ParseFile(lib.core.actors): %v", err)
	}
	appSrc := []byte(`module app.main
import lib.core.actors as actors

func main() -> Int
uses actors, runtime:
    let poll: actors.SystemReceiveResult = actors.poll_system()
    let timed: actors.SystemReceiveResult = actors.recv_system_until(0)
    return 0
`)
	appFile, err := frontend.ParseFile(appSrc, "app/main.tetra")
	if err != nil {
		t.Fatalf("ParseFile(app): %v", err)
	}
	world := &module.World{
		EntryModule: appFile.Module,
		Files:       []*frontend.FileAST{libFile, appFile},
		ByModule: map[string]*frontend.FileAST{
			libFile.Module: libFile,
			appFile.Module: appFile,
		},
	}
	if _, err := CheckWorldOpt(world, CheckOptions{RequireMain: true}); err != nil {
		t.Fatalf("CheckWorldOpt: %v", err)
	}
}

func TestLibCoreActorsSystemReceiveResultPayloadsCanBePatternMatched(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	libPath := filepath.Join(root, "lib", "core", "actors", "actors.tetra")
	libSrc, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatalf("read lib.core.actors: %v", err)
	}
	libFile, err := frontend.ParseFile(libSrc, libPath)
	if err != nil {
		t.Fatalf("ParseFile(lib.core.actors): %v", err)
	}
	appSrc := []byte(`module app.main
import lib.core.actors as actors

func score(result: actors.SystemReceiveResult) -> Int:
    match result:
    case actors.SystemReceiveResult.message(message):
        match message:
        case actors.SystemMessage.exit(exit_peer, exit_reason):
            return 10
        case actors.SystemMessage.down(down_monitor, down_peer, down_reason):
            return 20
        case actors.SystemMessage.node_down(down_node, node_reason):
            return 30
    case actors.SystemReceiveResult.empty:
        return 0
    case actors.SystemReceiveResult.timeout:
        return 1
    case actors.SystemReceiveResult.canceled:
        return 2
    case actors.SystemReceiveResult.runtime_closed:
        return 3
    case actors.SystemReceiveResult.invalid_state(code):
        return code

func main() -> Int
uses actors, runtime:
    let poll: actors.SystemReceiveResult = actors.poll_system()
    let timed: actors.SystemReceiveResult = actors.recv_system_until(0)
    return score(poll) + score(timed)
`)
	appFile, err := frontend.ParseFile(appSrc, "app/main.tetra")
	if err != nil {
		t.Fatalf("ParseFile(app): %v", err)
	}
	world := &module.World{
		EntryModule: appFile.Module,
		Files:       []*frontend.FileAST{libFile, appFile},
		ByModule: map[string]*frontend.FileAST{
			libFile.Module: libFile,
			appFile.Module: appFile,
		},
	}
	if _, err := CheckWorldOpt(world, CheckOptions{RequireMain: true}); err != nil {
		t.Fatalf("CheckWorldOpt: %v", err)
	}
}

func TestLibCoreActorsLifecycleSurfaceChecksAgainstRawBuiltins(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	libPath := filepath.Join(root, "lib", "core", "actors", "actors.tetra")
	libSrc, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatalf("read lib.core.actors: %v", err)
	}
	libFile, err := frontend.ParseFile(libSrc, libPath)
	if err != nil {
		t.Fatalf("ParseFile(lib.core.actors): %v", err)
	}
	appSrc := []byte(`module app.main
import lib.core.actors as actors

func score_status(result: actors.StatusResult) -> Int:
    match result:
    case actors.StatusResult.ok(status):
        match status:
        case actors.ActorStatus.starting:
            return 0
        case actors.ActorStatus.ready:
            return 1
        case actors.ActorStatus.running:
            return 2
        case actors.ActorStatus.blocked:
            return 3
        case actors.ActorStatus.sleeping:
            return 4
        case actors.ActorStatus.waiting:
            return 5
        case actors.ActorStatus.stopping:
            return 6
        case actors.ActorStatus.exited_normal:
            return 7
        case actors.ActorStatus.exited_error(code):
            return 80 + code
        case actors.ActorStatus.canceled:
            return 9
        case actors.ActorStatus.restarting:
            return 10
        case actors.ActorStatus.dead:
            return 11
        case actors.ActorStatus.unknown(code):
            return 100 + code
    case actors.StatusResult.invalid:
        return 200
    case actors.StatusResult.stale:
        return 201
    case actors.StatusResult.node_down:
        return 202

func score_wait(result: actors.WaitResult) -> Int:
    match result:
    case actors.WaitResult.exited(reason):
        match reason:
        case actors.ExitReason.normal:
            return 0
        case actors.ExitReason.shutdown(code):
            return 10 + code
        case actors.ExitReason.error(code):
            return 20 + code
        case actors.ExitReason.canceled:
            return 30
        case actors.ExitReason.killed:
            return 40
        case actors.ExitReason.node_down(code):
            return 50 + code
        case actors.ExitReason.protocol_error(code):
            return 60 + code
        case actors.ExitReason.runtime_error(code):
            return 70 + code
        case actors.ExitReason.unknown(kind, code):
            return 80 + kind + code
    case actors.WaitResult.timeout:
        return 100
    case actors.WaitResult.canceled:
        return 101
    case actors.WaitResult.invalid:
        return 102
    case actors.WaitResult.stale:
        return 103
    case actors.WaitResult.node_down:
        return 104

func score_stop(result: actors.StopResult) -> Int:
    match result:
    case actors.StopResult.requested:
        return 0
    case actors.StopResult.already_exited(reason):
        return 10
    case actors.StopResult.invalid:
        return 20
    case actors.StopResult.stale:
        return 21
    case actors.StopResult.node_down:
        return 22

func score_link(result: actors.LinkResult) -> Int:
    match result:
    case actors.LinkResult.linked:
        return 0
    case actors.LinkResult.already_linked:
        return 1
    case actors.LinkResult.target_exited(reason):
        return 2
    case actors.LinkResult.resource_exhausted:
        return 3
    case actors.LinkResult.invalid:
        return 4
    case actors.LinkResult.stale:
        return 5
    case actors.LinkResult.node_down:
        return 6

func score_monitor(result: actors.MonitorResult) -> Int:
    match result:
    case actors.MonitorResult.monitoring(reference):
        return 0
    case actors.MonitorResult.target_already_exited(reference):
        return 1
    case actors.MonitorResult.resource_exhausted:
        return 2
    case actors.MonitorResult.invalid:
        return 3
    case actors.MonitorResult.stale:
        return 4
    case actors.MonitorResult.node_down:
        return 5

func main() -> Int
uses actors, runtime:
    let self_ref: actor = core.self()
    let status: actors.StatusResult = actors.status(self_ref)
    let waited: actors.WaitResult = actors.wait_until(self_ref, 0)
    let stopped: actors.StopResult = actors.stop(self_ref, actors.ExitReason.normal)
    let linked: actors.LinkResult = actors.link(self_ref)
    let unlinked: Bool = actors.unlink(self_ref)
    let monitored: actors.MonitorResult = actors.monitor(self_ref)
    let trapped: Bool = actors.set_trap_exit(true)
    let _score: Int = score_status(status) + score_wait(waited) + score_stop(stopped) + score_link(linked) + score_monitor(monitored)
    return 0
`)
	appFile, err := frontend.ParseFile(appSrc, "app/main.tetra")
	if err != nil {
		t.Fatalf("ParseFile(app): %v", err)
	}
	world := &module.World{
		EntryModule: appFile.Module,
		Files:       []*frontend.FileAST{libFile, appFile},
		ByModule: map[string]*frontend.FileAST{
			libFile.Module: libFile,
			appFile.Module: appFile,
		},
	}
	if _, err := CheckWorldOpt(world, CheckOptions{RequireMain: true}); err != nil {
		t.Fatalf("CheckWorldOpt: %v", err)
	}
}

func TestLibCoreActorsSystemMessageCannotUseOrdinaryActorMailbox(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	libPath := filepath.Join(root, "lib", "core", "actors", "actors.tetra")
	libSrc, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatalf("read lib.core.actors: %v", err)
	}
	libFile, err := frontend.ParseFile(libSrc, libPath)
	if err != nil {
		t.Fatalf("ParseFile(lib.core.actors): %v", err)
	}
	appSrc := []byte(`module app.main
import lib.core.actors as actors

func main() -> Int
uses actors:
    let msg: actors.SystemMessage = actors.SystemMessage.exit(core.self(), actors.ExitReason.normal)
    return core.send_typed(core.self(), msg)
`)
	appFile, err := frontend.ParseFile(appSrc, "app/main.tetra")
	if err != nil {
		t.Fatalf("ParseFile(app): %v", err)
	}
	world := &module.World{
		EntryModule: appFile.Module,
		Files:       []*frontend.FileAST{libFile, appFile},
		ByModule: map[string]*frontend.FileAST{
			libFile.Module: libFile,
			appFile.Module: appFile,
		},
	}
	_, err = CheckWorldOpt(world, CheckOptions{RequireMain: true})
	if err == nil {
		t.Fatalf("expected system-message ordinary mailbox diagnostic")
	}
	if !strings.Contains(err.Error(), "runtime system messages cannot be sent through the ordinary actor mailbox") {
		t.Fatalf("error = %v", err)
	}
}

func TestLibCoreActorsSystemMessageCannotUseOrdinaryTypedReceive(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	libPath := filepath.Join(root, "lib", "core", "actors", "actors.tetra")
	libSrc, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatalf("read lib.core.actors: %v", err)
	}
	libFile, err := frontend.ParseFile(libSrc, libPath)
	if err != nil {
		t.Fatalf("ParseFile(lib.core.actors): %v", err)
	}
	appSrc := []byte(`module app.main
import lib.core.actors as actors

func main() -> Int
uses actors:
    let msg: actors.SystemMessage = core.recv_typed<actors.SystemMessage>()
    return 0
`)
	appFile, err := frontend.ParseFile(appSrc, "app/main.tetra")
	if err != nil {
		t.Fatalf("ParseFile(app): %v", err)
	}
	world := &module.World{
		EntryModule: appFile.Module,
		Files:       []*frontend.FileAST{libFile, appFile},
		ByModule: map[string]*frontend.FileAST{
			libFile.Module: libFile,
			appFile.Module: appFile,
		},
	}
	_, err = CheckWorldOpt(world, CheckOptions{RequireMain: true})
	if err == nil {
		t.Fatalf("expected system-message ordinary typed receive diagnostic")
	}
	if !strings.Contains(err.Error(), "runtime system messages cannot be sent through the ordinary actor mailbox") {
		t.Fatalf("error = %v", err)
	}
}

func TestUserDefinedSystemMessageCanUseOrdinaryTypedActorMailbox(t *testing.T) {
	src := `module main

enum SystemMessage:
    case ping(Int)

func main() -> Int
uses actors:
    let sent: Int = core.send_typed(core.self(), SystemMessage.ping(7))
    let received: SystemMessage = core.recv_typed<SystemMessage>()
    match received:
    case SystemMessage.ping(value):
        return sent + value
`
	file, err := frontend.ParseFile([]byte(src), "main.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: file.Module,
		Files:       []*frontend.FileAST{file},
		ByModule: map[string]*frontend.FileAST{
			file.Module: file,
		},
	}
	if _, err := CheckWorldOpt(world, CheckOptions{RequireMain: true}); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func checkActorSystemReceiveError(t *testing.T, src string) error {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected Check error")
	}
	return err
}

func TestEnsureTypeInfoRejectsTargetLayoutOnlyNativeIntegers(t *testing.T) {
	types := baseTypes()
	for _, name := range []string{
		"usize",
		"isize",
		"size_t",
		"ssize_t",
		"native_int",
		"native_uint",
		"c_long",
		"c_ulong",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ensureTypeInfo(name, types)
			if err == nil {
				t.Fatalf(
					("ensureTypeInfo(%q) succeeded; target-layout-only scalar must " +
						"not become a source type implicitly"),
					name,
				)
			}
			for _, want := range []string{
				"target-layout scalar type '" + name + "'",
				"not supported in source-level Tetra yet",
				"native-int/codegen support",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("ensureTypeInfo(%q) error = %v, want substring %q", name, err, want)
				}
			}
		})
	}
}

func TestEnsureTypeInfoAcceptsILP32NativeIntegerAliasesWhenEnabled(t *testing.T) {
	types := baseTypes()
	addILP32NativeScalarTypes(types)
	for _, name := range []string{
		"usize",
		"isize",
		"size_t",
		"ssize_t",
		"native_int",
		"native_uint",
		"c_long",
		"c_ulong",
	} {
		t.Run(name, func(t *testing.T) {
			info, err := ensureTypeInfo(name, types)
			if err != nil {
				t.Fatalf("ensureTypeInfo(%q): %v", name, err)
			}
			if info.Kind != TypeI32 || info.SlotCount != 1 || !info.Public {
				t.Fatalf("type info = %#v, want public 1-slot TypeI32 alias", info)
			}
		})
	}
}

func TestEnsureTypeInfoAcceptsILP32PointerAliasesWhenEnabled(t *testing.T) {
	for _, name := range []string{"rawptr", "nullable_ptr", "ref"} {
		t.Run(name, func(t *testing.T) {
			types := baseTypes()
			if _, err := ensureTypeInfo(name, types); err == nil {
				t.Fatalf("ensureTypeInfo(%s) succeeded before ILP32 target scalar enablement", name)
			}

			addILP32NativeScalarTypes(types)
			info, err := ensureTypeInfo(name, types)
			if err != nil {
				t.Fatalf("ensureTypeInfo(%s): %v", name, err)
			}
			if info.Kind != TypePtr || info.SlotCount != 1 || !info.Public {
				t.Fatalf("type info = %#v, want public 1-slot TypePtr alias", info)
			}
		})
	}
}

// ---- callable_escape_test.go ----

func TestClassifyCallableEscapeUsesFnptrForBoundedLocalSnapshot(t *testing.T) {
	kind, handle, err := classifyCallableEscape(callableBoundaryReturn, []frontend.ClosureCapture{
		{Name: "base", Type: frontend.TypeRef{Name: "i32"}},
	}, baseTypes())
	if err != nil {
		t.Fatalf("classifyCallableEscape: %v", err)
	}
	if kind != CallableEscapeLocalSnapshot || handle {
		t.Fatalf(
			"classification = (%q, %v), want (%q, false)",
			kind,
			handle,
			CallableEscapeLocalSnapshot,
		)
	}
}

func TestClassifyCallableEscapeUsesHandleForOversizedReturn(t *testing.T) {
	captures := make([]frontend.ClosureCapture, 0, FnPtrEnvSlotCount+1)
	for i := 0; i < FnPtrEnvSlotCount+1; i++ {
		captures = append(captures, frontend.ClosureCapture{
			Name: "capture",
			Type: frontend.TypeRef{Name: "i32"},
		})
	}

	kind, handle, err := classifyCallableEscape(callableBoundaryReturn, captures, baseTypes())
	if err != nil {
		t.Fatalf("classifyCallableEscape: %v", err)
	}
	if kind != CallableEscapeHeap || !handle {
		t.Fatalf("classification = (%q, %v), want (%q, true)", kind, handle, CallableEscapeHeap)
	}
}

func TestClassifyCallableEscapeRejectsMutableEscapingCapture(t *testing.T) {
	captures := make([]frontend.ClosureCapture, 0, FnPtrEnvSlotCount+1)
	for i := 0; i < FnPtrEnvSlotCount+1; i++ {
		captures = append(captures, frontend.ClosureCapture{
			Name:    "total",
			Type:    frontend.TypeRef{Name: "i32"},
			Mutable: i == 0,
		})
	}

	_, _, err := classifyCallableEscape(callableBoundaryGlobal, captures, baseTypes())
	if err == nil {
		t.Fatalf("expected mutable capture escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestClassifyCallableEscapeRejectsResourceCaptureAcrossThreadBoundary(t *testing.T) {
	_, _, err := classifyCallableEscape(callableBoundaryThread, []frontend.ClosureCapture{
		{Name: "raw", Type: frontend.TypeRef{Name: "ptr"}},
	}, baseTypes())
	if err == nil {
		t.Fatalf("expected resource capture escape diagnostic")
	}
	want := "escaped function value captures local 'raw' of type 'ptr'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestClassifyCallableEscapeRejectsMutableCaptureAcrossThreadBoundary(t *testing.T) {
	_, _, err := classifyCallableEscape(callableBoundaryThread, []frontend.ClosureCapture{
		{Name: "total", Type: frontend.TypeRef{Name: "i32"}, Mutable: true},
	}, baseTypes())
	if err == nil {
		t.Fatalf("expected mutable capture thread escape diagnostic")
	}
	want := "thread-escaped function value captures mutable local 'total'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

// ---- distributed_actor_runtime_test.go ----

func TestCheckDistributedActorRuntimeBuiltins(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(2, 5010)
    let peer: actor = core.spawn_remote(2, "worker")
    let sent: Int = core.send(peer, connected)
    return core.actor_node_status(2) + sent
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestCheckDistributedActorRuntimeBuiltinsRequireRuntimeEffect(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors:
    let _peer: actor = core.spawn_remote(2, "worker")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected runtime effect diagnostic")
	}
	if !strings.Contains(err.Error(), "uses effect 'runtime'") {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckSpawnRemoteRejectsNonLiteralTarget(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let name: str = "worker"
    let _peer: actor = core.spawn_remote(2, name)
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected spawn_remote target diagnostic")
	}
	if !strings.Contains(err.Error(), "spawn_remote expects a string literal") {
		t.Fatalf("error = %v", err)
	}
}

// ---- enum_payload_fields_test.go ----

func TestReturnedStructEnumPayloadFieldSignatureMetadata(t *testing.T) {
	src := []byte(`module app.main

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

func makeBox(cb: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(cb))

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let box: Box = makeBox(add1)
    let choice: MaybeCallback = box.choice
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`)
	file, err := frontend.ParseFile(src, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: "app.main",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"app.main": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	sig := checked.FuncSigs["app.main.makeBox"]
	if len(sig.ReturnEnumPayloadFields) == 0 {
		t.Fatalf("ReturnEnumPayloadFields is empty")
	}
	field, ok := sig.ReturnEnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("ReturnEnumPayloadFields = %#v, want choice#0:0", sig.ReturnEnumPayloadFields)
	}
	if field.FunctionParamName != "cb" {
		t.Fatalf("FunctionParamName = %q, want cb", field.FunctionParamName)
	}
}

func TestReturnedStructEnumPayloadFieldCallSiteCaptureMetadata(t *testing.T) {
	callbacksSrc := []byte(`module lib.callbacks

pub func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb
`)
	mainSrc := []byte(`module app.main
import lib.callbacks as callbacks

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

func makeBox(cb: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(cb))

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let box: Box = makeBox(callbacks.identity(captured))
    let choice: MaybeCallback = box.choice
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`)
	callbacks, err := frontend.ParseFile(callbacksSrc, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("ParseFile callbacks: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule: "app.main",
		Files:       []*frontend.FileAST{callbacks, main},
		ByModule: map[string]*frontend.FileAST{
			"lib.callbacks": callbacks,
			"app.main":      main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	identitySig := checked.FuncSigs["lib.callbacks.identity"]
	if identitySig.ReturnFunctionParamName != "cb" {
		t.Fatalf(
			"identity ReturnFunctionParamName = %q, want cb",
			identitySig.ReturnFunctionParamName,
		)
	}
	var mainFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			mainFunc = fn
			break
		}
	}
	box := mainFunc.Locals["box"]
	field, ok := box.EnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("box EnumPayloadFields = %#v, want choice#0:0", box.EnumPayloadFields)
	}
	if len(field.FunctionEscapeCaptures) == 0 && len(field.FunctionCaptures) == 0 {
		t.Fatalf("box.choice payload has no captures: %#v", field)
	}
	if field.FunctionValue == "" {
		t.Fatalf("box.choice payload has no function value: %#v", field)
	}
	choice := mainFunc.Locals["choice"]
	payload, ok := choice.EnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("choice EnumPayloadFunctions = %#v, want 0:0", choice.EnumPayloadFunctions)
	}
	if len(payload.FunctionEscapeCaptures) == 0 && len(payload.FunctionCaptures) == 0 {
		t.Fatalf("choice payload has no captures: %#v", payload)
	}
	if payload.FunctionValue == "" {
		t.Fatalf("choice payload has no function value: %#v", payload)
	}
	bound := mainFunc.Locals["cb"]
	if len(bound.FunctionEscapeCaptures) == 0 && len(bound.FunctionCaptures) == 0 {
		t.Fatalf("pattern cb has no captures: %#v", bound)
	}
	if bound.FunctionValue == "" {
		t.Fatalf("pattern cb has no function value: %#v", bound)
	}
}

func TestImportedReturnedStructEnumPayloadDirectClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    ))
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let box: pack.Box = pack.makeBox()
    match box.choice:
    case pack.MaybeCallback.some(local):
        cb = local
        return 0
    case pack.MaybeCallback.empty:
        return 0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule: "app.main",
		Files:       []*frontend.FileAST{pack, main},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeBox := checked.FuncSigs["lib.pack.makeBox"]
	field, ok := makeBox.ReturnEnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf(
			"makeBox ReturnEnumPayloadFields = %#v, want choice#0:0",
			makeBox.ReturnEnumPayloadFields,
		)
	}
	if !field.FunctionReturnSnapshotAlias || len(field.FunctionEscapeCaptures) == 0 ||
		field.FunctionParamName != "" {
		t.Fatalf("makeBox returned payload metadata = %#v, want direct snapshot", field)
	}
	var mainFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			mainFunc = fn
			break
		}
	}
	box := mainFunc.Locals["box"]
	field, ok = box.EnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("box EnumPayloadFields = %#v, want choice#0:0", box.EnumPayloadFields)
	}
	if !field.FunctionReturnSnapshotAlias || len(field.FunctionEscapeCaptures) == 0 ||
		field.FunctionParamName != "" {
		t.Fatalf("box returned payload metadata = %#v, want direct snapshot", field)
	}
	bound := mainFunc.Locals["local"]
	if !bound.FunctionReturnSnapshotAlias || len(bound.FunctionEscapeCaptures) == 0 ||
		bound.FunctionParamName != "" {
		t.Fatalf("pattern local metadata = %#v, want direct snapshot", bound)
	}
}

func TestImportedReturnedEnumPayloadDirectClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let choice: pack.MaybeCallback = pack.makeChoice()
    match choice:
    case pack.MaybeCallback.some(local):
        cb = local
        return 0
    case pack.MaybeCallback.empty:
        return 0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule: "app.main",
		Files:       []*frontend.FileAST{pack, main},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeChoice := checked.FuncSigs["lib.pack.makeChoice"]
	payload, ok := makeChoice.ReturnEnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf(
			"makeChoice ReturnEnumPayloadFunctions = %#v, want 0:0",
			makeChoice.ReturnEnumPayloadFunctions,
		)
	}
	if !payload.FunctionReturnSnapshotAlias || len(payload.FunctionEscapeCaptures) == 0 ||
		payload.FunctionParamName != "" {
		t.Fatalf("makeChoice returned payload metadata = %#v, want direct snapshot", payload)
	}
	var mainFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			mainFunc = fn
			break
		}
	}
	choice := mainFunc.Locals["choice"]
	payload, ok = choice.EnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("choice EnumPayloadFunctions = %#v, want 0:0", choice.EnumPayloadFunctions)
	}
	if !payload.FunctionReturnSnapshotAlias || len(payload.FunctionEscapeCaptures) == 0 ||
		payload.FunctionParamName != "" {
		t.Fatalf("choice returned payload metadata = %#v, want direct snapshot", payload)
	}
	bound := mainFunc.Locals["local"]
	if !bound.FunctionReturnSnapshotAlias || len(bound.FunctionEscapeCaptures) == 0 ||
		bound.FunctionParamName != "" {
		t.Fatalf("pattern local metadata = %#v, want direct snapshot", bound)
	}
}

func TestInterfaceReturnedStructEnumPayloadInlineClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    return Box(choice: MaybeCallback.some(fn(p0: Int) -> Int = 0))
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

func main() -> Int:
    let box: pack.Box = pack.makeBox()
    match box.choice:
    case pack.MaybeCallback.some(local):
        return local(41)
    case pack.MaybeCallback.empty:
        return 0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4i")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule:      "app.main",
		Files:            []*frontend.FileAST{pack, main},
		InterfaceModules: map[string]bool{"lib.pack": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeBox := checked.FuncSigs["lib.pack.makeBox"]
	field, ok := makeBox.ReturnEnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf(
			"makeBox ReturnEnumPayloadFields = %#v, want choice#0:0",
			makeBox.ReturnEnumPayloadFields,
		)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" {
		t.Fatalf("makeBox returned payload metadata = %#v, want synthetic closure target", field)
	}
	var mainFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			mainFunc = fn
			break
		}
	}
	box := mainFunc.Locals["box"]
	field, ok = box.EnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("box EnumPayloadFields = %#v, want choice#0:0", box.EnumPayloadFields)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" {
		t.Fatalf("box returned payload metadata = %#v, want synthetic closure target", field)
	}
	bound := mainFunc.Locals["local"]
	if bound.FunctionValue == "" || bound.FunctionParamName != "" {
		t.Fatalf("pattern local metadata = %#v, want synthetic closure target", bound)
	}
}

func TestInterfaceReturnedStructEnumPayloadInlineThrowingClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    return Box(choice: MaybeCallback.some(fn(p0: Int) -> Int throws Boom = 0))
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

func caller() -> Int throws pack.Boom:
    let box: pack.Box = pack.makeBox()
    match box.choice:
    case pack.MaybeCallback.some(local):
        return try local(41)
    case pack.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case pack.Boom.bad:
        0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4i")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule:      "app.main",
		Files:            []*frontend.FileAST{pack, main},
		InterfaceModules: map[string]bool{"lib.pack": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeBox := checked.FuncSigs["lib.pack.makeBox"]
	field, ok := makeBox.ReturnEnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf(
			"makeBox ReturnEnumPayloadFields = %#v, want choice#0:0",
			makeBox.ReturnEnumPayloadFields,
		)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" ||
		field.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf(
			"makeBox returned payload metadata = %#v, want synthetic throwing closure target",
			field,
		)
	}
	var callerFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.caller" {
			callerFunc = fn
			break
		}
	}
	box := callerFunc.Locals["box"]
	field, ok = box.EnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("box EnumPayloadFields = %#v, want choice#0:0", box.EnumPayloadFields)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" ||
		field.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf(
			"box returned payload metadata = %#v, want synthetic throwing closure target",
			field,
		)
	}
	bound := callerFunc.Locals["local"]
	if bound.FunctionValue == "" || bound.FunctionParamName != "" ||
		bound.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf("pattern local metadata = %#v, want synthetic throwing closure target", bound)
	}
}

func TestInterfaceReturnedStructFieldInlineThrowingClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    return Holder(cb: fn(p0: Int) -> Int throws Boom = 0)
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

func caller() -> Int throws pack.Boom:
    let holder: pack.Holder = pack.makeHolder()
    return try holder.cb(41)

func main() -> Int:
    return catch caller():
    case pack.Boom.bad:
        0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4i")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule:      "app.main",
		Files:            []*frontend.FileAST{pack, main},
		InterfaceModules: map[string]bool{"lib.pack": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeHolder := checked.FuncSigs["lib.pack.makeHolder"]
	field, ok := makeHolder.ReturnFunctionFields["cb"]
	if !ok {
		t.Fatalf("makeHolder ReturnFunctionFields = %#v, want cb", makeHolder.ReturnFunctionFields)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" ||
		field.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf(
			"makeHolder returned field metadata = %#v, want synthetic throwing closure target",
			field,
		)
	}
	var callerFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.caller" {
			callerFunc = fn
			break
		}
	}
	holder := callerFunc.Locals["holder"]
	field, ok = holder.FunctionFields["cb"]
	if !ok {
		t.Fatalf("holder FunctionFields = %#v, want cb", holder.FunctionFields)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" ||
		field.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf(
			"holder returned field metadata = %#v, want synthetic throwing closure target",
			field,
		)
	}
}

func TestInterfaceReturnedEnumPayloadInlineClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    return MaybeCallback.some(fn(p0: Int) -> Int = 0)
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

func main() -> Int:
    let choice: pack.MaybeCallback = pack.makeChoice()
    match choice:
    case pack.MaybeCallback.some(local):
        return local(41)
    case pack.MaybeCallback.empty:
        return 0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4i")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule:      "app.main",
		Files:            []*frontend.FileAST{pack, main},
		InterfaceModules: map[string]bool{"lib.pack": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeChoice := checked.FuncSigs["lib.pack.makeChoice"]
	payload, ok := makeChoice.ReturnEnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf(
			"makeChoice ReturnEnumPayloadFunctions = %#v, want 0:0",
			makeChoice.ReturnEnumPayloadFunctions,
		)
	}
	if payload.FunctionValue == "" || payload.FunctionParamName != "" {
		t.Fatalf(
			"makeChoice returned payload metadata = %#v, want synthetic closure target",
			payload,
		)
	}
	var mainFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			mainFunc = fn
			break
		}
	}
	choice := mainFunc.Locals["choice"]
	payload, ok = choice.EnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("choice EnumPayloadFunctions = %#v, want 0:0", choice.EnumPayloadFunctions)
	}
	if payload.FunctionValue == "" || payload.FunctionParamName != "" {
		t.Fatalf("choice returned payload metadata = %#v, want synthetic closure target", payload)
	}
	bound := mainFunc.Locals["local"]
	if bound.FunctionValue == "" || bound.FunctionParamName != "" {
		t.Fatalf("pattern local metadata = %#v, want synthetic closure target", bound)
	}
}

func TestInterfaceReturnedEnumPayloadInlineThrowingClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    return MaybeCallback.some(fn(p0: Int) -> Int throws Boom = 0)
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

func caller() -> Int throws pack.Boom:
    let choice: pack.MaybeCallback = pack.makeChoice()
    match choice:
    case pack.MaybeCallback.some(local):
        return try local(41)
    case pack.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case pack.Boom.bad:
        0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4i")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule:      "app.main",
		Files:            []*frontend.FileAST{pack, main},
		InterfaceModules: map[string]bool{"lib.pack": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeChoice := checked.FuncSigs["lib.pack.makeChoice"]
	payload, ok := makeChoice.ReturnEnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf(
			"makeChoice ReturnEnumPayloadFunctions = %#v, want 0:0",
			makeChoice.ReturnEnumPayloadFunctions,
		)
	}
	if payload.FunctionValue == "" || payload.FunctionParamName != "" ||
		payload.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf(
			"makeChoice returned payload metadata = %#v, want synthetic throwing closure target",
			payload,
		)
	}
	var callerFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.caller" {
			callerFunc = fn
			break
		}
	}
	choice := callerFunc.Locals["choice"]
	payload, ok = choice.EnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("choice EnumPayloadFunctions = %#v, want 0:0", choice.EnumPayloadFunctions)
	}
	if payload.FunctionValue == "" || payload.FunctionParamName != "" ||
		payload.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf(
			"choice returned payload metadata = %#v, want synthetic throwing closure target",
			payload,
		)
	}
	bound := callerFunc.Locals["local"]
	if bound.FunctionValue == "" || bound.FunctionParamName != "" ||
		bound.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf("pattern local metadata = %#v, want synthetic throwing closure target", bound)
	}
}

// ---- iflet_pattern_test.go ----

func TestIfLetSomePatternBindsOptionalPayload(t *testing.T) {
	checked := checkIfLetPatternSource(t, `
func unwrap(value: Int?) -> Int:
    if let some(x) = value:
        return x
    else:
        return 0

func main() -> Int:
    return unwrap(1)
`)
	if got := checked.Funcs[0].Locals["x"].TypeName; got != "i32" {
		t.Fatalf("some binding type = %q, want i32", got)
	}
}

func TestIfLetNonePatternAcceptsOptionalValue(t *testing.T) {
	checkIfLetPatternSource(t, `
func score(value: Int?) -> Int:
    if let none = value:
        return 7
    else:
        return 1

func main() -> Int:
    return score(none)
`)
}

func TestIfLetEnumPayloadPatternBindsPayloads(t *testing.T) {
	checked := checkIfLetPatternSource(t, `
enum Result:
    case ok(Int, String)
    case err(Int)

func score(value: Result) -> Int:
    if let Result.ok(code, text) = value:
        return code + text.len
    else:
        return 0

func main() -> Int:
    return score(Result.ok(1, "x"))
`)
	fn := checked.FuncSigs["score"]
	if fn.ParamSlots != 4 {
		t.Fatalf("score param slots = %d, want 4", fn.ParamSlots)
	}
	locals := checked.Funcs[0].Locals
	if got := locals["code"].TypeName; got != "i32" {
		t.Fatalf("code binding type = %q, want i32", got)
	}
	if got := locals["text"].TypeName; got != "str" {
		t.Fatalf("text binding type = %q, want str", got)
	}
}

func TestIfLetPatternRejectsNonOptionalAndNonEnumValue(t *testing.T) {
	err := checkIfLetPatternError(t, `
func main() -> Int:
    if let some(x) = 1:
        return x
    else:
        return 0
`)
	if !strings.Contains(err.Error(), "if let pattern requires optional or enum value") {
		t.Fatalf("error = %v", err)
	}
}

func TestIfLetOptionalPatternRejectsLiteralPattern(t *testing.T) {
	err := checkIfLetPatternError(t, `
func main() -> Int:
    let value: Int? = 1
    if let 1 = value:
        return 1
    else:
        return 0
`)
	if !strings.Contains(err.Error(), "optional if let supports only 'none'") {
		t.Fatalf("error = %v", err)
	}
}

func checkIfLetPatternSource(t *testing.T, src string) *CheckedProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return checked
}

func checkIfLetPatternError(t *testing.T, src string) error {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected Check error")
	}
	return err
}

// ---- layout_repr_test.go ----

func TestCheckPreservesStructRepresentationMetadata(t *testing.T) {
	prog, err := frontend.Parse([]byte(`
repr(C) struct Header:
    tag: c_int
    ptr: ptr

struct Packet:
    bytes: []u8

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.Types["Header"].Repr; got != frontend.StructReprC {
		t.Fatalf("Header repr = %q, want %q", got, frontend.StructReprC)
	}
	if got := checked.Types["Packet"].Repr; got != frontend.StructReprDefault {
		t.Fatalf("Packet repr = %q, want %q", got, frontend.StructReprDefault)
	}
}

func TestExportedDefaultStructRequiresExplicitRepr(t *testing.T) {
	prog, err := frontend.Parse([]byte(`
struct Pair:
    lo: c_int
    hi: c_int

@export("ffi_pair_c")
func ffi_pair(pair: Pair) -> c_int:
    return pair.lo

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("Check accepted exported default-layout struct")
	}
	for _, want := range []string{
		"exported function 'ffi_pair'",
		"parameter 'pair'",
		"type 'Pair'",
		"requires explicit repr(C)",
		"default Tetra layout is compiler-owned",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("diagnostic = %v, want substring %q", err, want)
		}
	}
}

func TestExportedReprCStructPassesExplicitReprGate(t *testing.T) {
	prog, err := frontend.Parse([]byte(`
repr(C) struct Pair:
    lo: c_int
    hi: c_int

@export("ffi_pair_c")
func ffi_pair(pair: Pair) -> c_int:
    return pair.lo

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check rejected repr(C) exported struct at explicit repr gate: %v", err)
	}
}

func TestBuiltinRuntimeABIAggregatesUseReprC(t *testing.T) {
	types := baseTypes()
	for _, name := range []string{
		"task.i32",
		"task.result_i32",
		"actor.msg",
		"actor.recv_result_i32",
		"actor.recv_msg_result",
	} {
		info := types[name]
		if info == nil {
			t.Fatalf("missing builtin type %s", name)
		}
		if got := info.Repr; got != frontend.StructReprC {
			t.Fatalf("%s repr = %q, want %q", name, got, frontend.StructReprC)
		}
	}
}

// ---- manifest_test.go ----

func TestManifestDescribeBuiltinsSortedAndAliasStable(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected non-empty builtin manifest")
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Name >= got[i].Name {
			t.Fatalf("manifest names not sorted: %q then %q", got[i-1].Name, got[i].Name)
		}
	}

	foundMakeU8 := false
	foundMakeU16 := false
	foundMakeBool := false
	foundIslandMakeBool := false
	foundRawSliceU8 := false
	for _, entry := range got {
		switch entry.Name {
		case "core.make_u8":
			foundMakeU8 = true
			if len(entry.Aliases) == 0 || entry.Aliases[0] != "make_u8" {
				t.Fatalf("core.make_u8 aliases = %#v, want leading make_u8", entry.Aliases)
			}
			if entry.ReturnType != "[]u8" {
				t.Fatalf("core.make_u8 return type = %q, want []u8", entry.ReturnType)
			}
		case "core.make_u16":
			foundMakeU16 = true
			if len(entry.Aliases) == 0 || entry.Aliases[0] != "make_u16" {
				t.Fatalf("core.make_u16 aliases = %#v, want leading make_u16", entry.Aliases)
			}
			if entry.ReturnType != "[]u16" {
				t.Fatalf("core.make_u16 return type = %q, want []u16", entry.ReturnType)
			}
		case "core.make_bool":
			foundMakeBool = true
			if len(entry.Aliases) == 0 || entry.Aliases[0] != "make_bool" {
				t.Fatalf("core.make_bool aliases = %#v, want leading make_bool", entry.Aliases)
			}
			if entry.ReturnType != "[]bool" {
				t.Fatalf("core.make_bool return type = %q, want []bool", entry.ReturnType)
			}
		case "core.island_make_bool":
			foundIslandMakeBool = true
			if len(entry.Aliases) == 0 || entry.Aliases[0] != "island_make_bool" {
				t.Fatalf(
					"core.island_make_bool aliases = %#v, want leading island_make_bool",
					entry.Aliases,
				)
			}
			if entry.ReturnType != "[]bool" {
				t.Fatalf("core.island_make_bool return type = %q, want []bool", entry.ReturnType)
			}
			if entry.UnsafePolicy != "conditional" {
				t.Fatalf(
					"core.island_make_bool unsafe policy = %q, want conditional",
					entry.UnsafePolicy,
				)
			}
		case "core.raw_slice_u8_from_parts":
			foundRawSliceU8 = true
			if !reflect.DeepEqual(entry.ParamTypes, []string{"ptr", "i32", "cap.mem"}) {
				t.Fatalf(
					"core.raw_slice_u8_from_parts params = %#v, want ptr/i32/cap.mem",
					entry.ParamTypes,
				)
			}
			if entry.ReturnType != "[]u8" {
				t.Fatalf(
					"core.raw_slice_u8_from_parts return type = %q, want []u8",
					entry.ReturnType,
				)
			}
			if entry.UnsafePolicy != "always" {
				t.Fatalf(
					"core.raw_slice_u8_from_parts unsafe policy = %q, want always",
					entry.UnsafePolicy,
				)
			}
		}
	}
	if !foundMakeU8 {
		t.Fatalf("missing core.make_u8 in manifest output")
	}
	if !foundMakeU16 {
		t.Fatalf("missing core.make_u16 in manifest output")
	}
	if !foundMakeBool {
		t.Fatalf("missing core.make_bool in manifest output")
	}
	if !foundIslandMakeBool {
		t.Fatalf("missing core.island_make_bool in manifest output")
	}
	if !foundRawSliceU8 {
		t.Fatalf("missing core.raw_slice_u8_from_parts in manifest output")
	}
}

func TestManifestValidationRejectsInvalidUnsafePolicy(t *testing.T) {
	err := validateBuiltinManifestEntry(BuiltinManifest{
		Name:         "core.fake",
		ReturnType:   "i32",
		UnsafePolicy: "sometimes",
	})
	if err == nil {
		t.Fatalf("expected unsafe policy validation error")
	}
}

func TestManifestValidationRejectsUnsortedEffectsOrAliases(t *testing.T) {
	tests := []BuiltinManifest{
		{
			Name:         "core.fake.aliases",
			ReturnType:   "i32",
			UnsafePolicy: "never",
			Aliases:      []string{"z", "a"},
		},
		{
			Name:         "core.fake.effects",
			ReturnType:   "i32",
			UnsafePolicy: "never",
			Effects:      []string{"runtime", "actors"},
		},
	}
	for _, tc := range tests {
		if err := validateBuiltinManifestEntry(tc); err == nil {
			t.Fatalf("expected validation error for %#v", tc)
		}
	}
}

func TestManifestValidationAcceptsWellFormedEntry(t *testing.T) {
	err := validateBuiltinManifestEntry(BuiltinManifest{
		Name:         "core.fake",
		ReturnType:   "i32",
		UnsafePolicy: "conditional",
		Aliases:      []string{"fake"},
		Effects:      []string{"mem", "runtime"},
	})
	if err != nil {
		t.Fatalf("validateBuiltinManifestEntry: %v", err)
	}
}

func TestManifestDescribeBuiltinsIncludesFilesystemExists(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}
	for _, entry := range got {
		if entry.Name != "core.fs_exists" {
			continue
		}
		if !reflect.DeepEqual(entry.ParamTypes, []string{"str", "cap.io"}) {
			t.Fatalf("core.fs_exists param types = %#v, want str, cap.io", entry.ParamTypes)
		}
		if entry.ReturnType != "bool" {
			t.Fatalf("core.fs_exists return type = %q, want bool", entry.ReturnType)
		}
		if strings.Join(entry.Effects, ",") != "io" {
			t.Fatalf("core.fs_exists effects = %q, want io", strings.Join(entry.Effects, ","))
		}
		if entry.UnsafePolicy != "never" {
			t.Fatalf("core.fs_exists unsafe policy = %q, want never", entry.UnsafePolicy)
		}
		return
	}
	t.Fatalf("manifest missing core.fs_exists")
}

func TestManifestDescribeBuiltinsIncludesAtomicSurface(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}
	byName := map[string]BuiltinManifest{}
	for _, entry := range got {
		byName[entry.Name] = entry
	}

	tests := []struct {
		name       string
		params     []string
		returnType string
	}{
		{
			name:       "core.atomic_load_i32_acquire",
			params:     []string{"ptr", "cap.mem"},
			returnType: "i32",
		},
		{
			name:       "core.atomic_store_i32_release",
			params:     []string{"ptr", "i32", "cap.mem"},
			returnType: "i32",
		},
		{
			name:       "core.atomic_compare_exchange_i32_acq_rel",
			params:     []string{"ptr", "i32", "i32", "cap.mem"},
			returnType: "i32",
		},
		{
			name:       "core.atomic_compare_exchange_weak_i32_seq_cst",
			params:     []string{"ptr", "i32", "i32", "cap.mem"},
			returnType: "i32",
		},
		{
			name:       "core.atomic_load_i64_acquire",
			params:     []string{"ptr", "cap.mem"},
			returnType: "i64",
		},
		{
			name:       "core.atomic_compare_exchange_weak_i64_seq_cst",
			params:     []string{"ptr", "i64", "i64", "cap.mem"},
			returnType: "i64",
		},
		{
			name:       "core.atomic_exchange_u8_seq_cst",
			params:     []string{"ptr", "u8", "cap.mem"},
			returnType: "u8",
		},
		{
			name:       "core.atomic_exchange_u16_seq_cst",
			params:     []string{"ptr", "u16", "cap.mem"},
			returnType: "u16",
		},
		{
			name:       "core.atomic_fetch_add_ptr_relaxed",
			params:     []string{"ptr", "ptr", "cap.mem"},
			returnType: "ptr",
		},
		{name: "core.atomic_fence_seq_cst", params: []string{"cap.mem"}, returnType: "i32"},
	}
	for _, tt := range tests {
		entry, ok := byName[tt.name]
		if !ok {
			t.Fatalf("manifest missing %s", tt.name)
		}
		if !reflect.DeepEqual(entry.ParamTypes, tt.params) {
			t.Fatalf("%s param types = %#v, want %#v", tt.name, entry.ParamTypes, tt.params)
		}
		if entry.ReturnType != tt.returnType {
			t.Fatalf("%s return type = %q, want %q", tt.name, entry.ReturnType, tt.returnType)
		}
		if strings.Join(entry.Effects, ",") != "mem" {
			t.Fatalf("%s effects = %q, want mem", tt.name, strings.Join(entry.Effects, ","))
		}
		if entry.UnsafePolicy != "always" {
			t.Fatalf("%s unsafe policy = %q, want always", tt.name, entry.UnsafePolicy)
		}
	}
}

func TestManifestDescribeBuiltinsIncludesSafeSliceViews(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}
	byName := map[string]BuiltinManifest{}
	for _, entry := range got {
		byName[entry.Name] = entry
	}

	for _, elem := range []string{"u8", "u16", "i32", "bool"} {
		for _, tc := range []struct {
			kind       string
			params     []string
			returnType string
			effects    []string
		}{
			{kind: "window", params: []string{"[]" + elem, "i32", "i32"}, returnType: "[]" + elem},
			{kind: "prefix", params: []string{"[]" + elem, "i32"}, returnType: "[]" + elem},
			{kind: "suffix", params: []string{"[]" + elem, "i32"}, returnType: "[]" + elem},
			{kind: "borrow", params: []string{"[]" + elem}, returnType: "[]" + elem},
			{
				kind:       "copy",
				params:     []string{"[]" + elem},
				returnType: "[]" + elem,
				effects:    []string{"alloc", "mem"},
			},
			{
				kind:       "copy_into",
				params:     []string{"[]" + elem, "[]" + elem},
				returnType: "i32",
				effects:    []string{"mem"},
			},
		} {
			name := "core.slice_" + tc.kind + "_" + elem
			entry, ok := byName[name]
			if !ok {
				t.Fatalf("manifest missing %s", name)
			}
			if !reflect.DeepEqual(entry.ParamTypes, tc.params) {
				t.Fatalf("%s param types = %#v, want %#v", name, entry.ParamTypes, tc.params)
			}
			if entry.ReturnType != tc.returnType {
				t.Fatalf("%s return type = %q, want %s", name, entry.ReturnType, tc.returnType)
			}
			if len(entry.Aliases) != 0 {
				t.Fatalf("%s aliases = %#v, want none", name, entry.Aliases)
			}
			if !reflect.DeepEqual(entry.Effects, tc.effects) {
				t.Fatalf("%s effects = %#v, want %#v", name, entry.Effects, tc.effects)
			}
			if entry.UnsafePolicy != "never" {
				t.Fatalf("%s unsafe policy = %q, want never", name, entry.UnsafePolicy)
			}
		}
	}

	for _, tc := range []struct {
		name       string
		params     []string
		returnType string
		effects    []string
	}{
		{name: "core.string_window", params: []string{"str", "i32", "i32"}, returnType: "str"},
		{name: "core.string_prefix", params: []string{"str", "i32"}, returnType: "str"},
		{name: "core.string_suffix", params: []string{"str", "i32"}, returnType: "str"},
		{name: "core.string_borrow", params: []string{"str"}, returnType: "str"},
		{
			name:       "core.string_copy",
			params:     []string{"str"},
			returnType: "str",
			effects:    []string{"alloc", "mem"},
		},
		{
			name:       "core.string_copy_into",
			params:     []string{"str", "[]u8"},
			returnType: "i32",
			effects:    []string{"mem"},
		},
	} {
		entry, ok := byName[tc.name]
		if !ok {
			t.Fatalf("manifest missing %s", tc.name)
		}
		if !reflect.DeepEqual(entry.ParamTypes, tc.params) {
			t.Fatalf("%s param types = %#v, want %#v", tc.name, entry.ParamTypes, tc.params)
		}
		if entry.ReturnType != tc.returnType {
			t.Fatalf("%s return type = %q, want %s", tc.name, entry.ReturnType, tc.returnType)
		}
		if len(entry.Aliases) != 0 {
			t.Fatalf("%s aliases = %#v, want none", tc.name, entry.Aliases)
		}
		if !reflect.DeepEqual(entry.Effects, tc.effects) {
			t.Fatalf("%s effects = %#v, want %#v", tc.name, entry.Effects, tc.effects)
		}
		if entry.UnsafePolicy != "never" {
			t.Fatalf("%s unsafe policy = %q, want never", tc.name, entry.UnsafePolicy)
		}
	}
}

func TestManifestDescribeBuiltinsIncludesNetSocketLifecycle(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}
	byName := map[string]BuiltinManifest{}
	for _, entry := range got {
		byName[entry.Name] = entry
	}
	tests := []struct {
		name       string
		params     []string
		returnType string
		effects    string
	}{
		{
			name:       "core.net_socket_tcp4",
			params:     []string{"cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_bind_tcp4_loopback",
			params:     []string{"i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_connect_tcp4_loopback",
			params:     []string{"i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_listen",
			params:     []string{"i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_accept4",
			params:     []string{"i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_read",
			params:     []string{"i32", "[]u8", "i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io,mem",
		},
		{
			name:       "core.net_recv",
			params:     []string{"i32", "[]u8", "i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io,mem",
		},
		{
			name:       "core.net_write",
			params:     []string{"i32", "[]u8", "i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io,mem",
		},
		{
			name:       "core.net_send",
			params:     []string{"i32", "[]u8", "i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io,mem",
		},
		{
			name:       "core.net_epoll_create",
			params:     []string{"cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_epoll_ctl_add_read",
			params:     []string{"i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_epoll_ctl_add_read_write",
			params:     []string{"i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_epoll_ctl_mod_read",
			params:     []string{"i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_epoll_ctl_mod_read_write",
			params:     []string{"i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_epoll_ctl_delete",
			params:     []string{"i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_epoll_wait_one",
			params:     []string{"i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_epoll_wait_one_into",
			params:     []string{"i32", "[]i32", "i32", "cap.io"},
			returnType: "i32",
			effects:    "io,mem",
		},
		{
			name:       "core.net_set_nonblocking",
			params:     []string{"i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_set_reuseport",
			params:     []string{"i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_set_tcp_nodelay",
			params:     []string{"i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
		{
			name:       "core.net_close",
			params:     []string{"i32", "cap.io"},
			returnType: "i32",
			effects:    "io",
		},
	}
	for _, tt := range tests {
		entry, ok := byName[tt.name]
		if !ok {
			t.Fatalf("manifest missing %s", tt.name)
		}
		if !reflect.DeepEqual(entry.ParamTypes, tt.params) {
			t.Fatalf("%s param types = %#v, want %#v", tt.name, entry.ParamTypes, tt.params)
		}
		if entry.ReturnType != tt.returnType {
			t.Fatalf("%s return type = %q, want %q", tt.name, entry.ReturnType, tt.returnType)
		}
		if strings.Join(entry.Effects, ",") != tt.effects {
			t.Fatalf(
				"%s effects = %q, want %q",
				tt.name,
				strings.Join(entry.Effects, ","),
				tt.effects,
			)
		}
		if entry.UnsafePolicy != "never" {
			t.Fatalf("%s unsafe policy = %q, want never", tt.name, entry.UnsafePolicy)
		}
	}
}

func TestManifestDriftProofAgainstBuiltinPolicySources(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}

	types := baseTypes()
	sigs, err := builtinFuncSigs(types)
	if err != nil {
		t.Fatalf("builtinFuncSigs: %v", err)
	}

	if len(got) != len(sigs) {
		t.Fatalf("manifest size = %d, builtin signatures = %d", len(got), len(sigs))
	}

	entriesByName := make(map[string]BuiltinManifest, len(got))
	for _, entry := range got {
		entriesByName[entry.Name] = entry
	}

	expectedAliasesByTarget := make(map[string][]string, len(sigs))
	for target := range sigs {
		short := strings.TrimPrefix(target, "core.")
		if resolved, ok := ResolveBuiltinAlias(short); ok {
			if resolved != target {
				t.Fatalf("ResolveBuiltinAlias(%q) = %q, want %q", short, resolved, target)
			}
			expectedAliasesByTarget[target] = append(expectedAliasesByTarget[target], short)
		}
	}
	for target, aliases := range expectedAliasesByTarget {
		sort.Strings(aliases)
		expectedAliasesByTarget[target] = aliases
	}

	for name, sig := range sigs {
		entry, ok := entriesByName[name]
		if !ok {
			t.Fatalf("manifest missing builtin %q", name)
		}
		if !reflect.DeepEqual(entry.ParamTypes, sig.ParamTypes) {
			t.Fatalf("%s param types = %#v, want %#v", name, entry.ParamTypes, sig.ParamTypes)
		}
		if entry.ReturnType != sig.ReturnType {
			t.Fatalf("%s return type = %q, want %q", name, entry.ReturnType, sig.ReturnType)
		}

		wantEffects := builtinEffects(name)
		if !reflect.DeepEqual(entry.Effects, wantEffects) {
			t.Fatalf("%s effects = %#v, want %#v", name, entry.Effects, wantEffects)
		}

		wantPolicy, wantDetails := expectedBuiltinUnsafePolicy(name)
		if entry.UnsafePolicy != wantPolicy {
			t.Fatalf("%s unsafe policy = %q, want %q", name, entry.UnsafePolicy, wantPolicy)
		}
		if entry.UnsafeDetails != wantDetails {
			t.Fatalf("%s unsafe details = %q, want %q", name, entry.UnsafeDetails, wantDetails)
		}

		wantAliases := expectedAliasesByTarget[name]
		if !reflect.DeepEqual(entry.Aliases, wantAliases) {
			t.Fatalf("%s aliases = %#v, want %#v", name, entry.Aliases, wantAliases)
		}
		for _, alias := range entry.Aliases {
			resolved, ok := ResolveBuiltinAlias(alias)
			if !ok || resolved != name {
				t.Fatalf(
					"%s alias %q resolves to (%q, %v), want (%q, true)",
					name,
					alias,
					resolved,
					ok,
					name,
				)
			}
		}
	}
}

func expectedBuiltinUnsafePolicy(name string) (policy string, details string) {
	switch name {
	case "core.island_make_u8",
		"core.island_make_u16",
		"core.island_make_i32",
		"core.island_make_bool":
		return "conditional", "requires unsafe when the island argument is not a scoped island variable"
	default:
		if builtinNeedsUnsafe(name, nil) {
			return "always", ""
		}
		return "never", ""
	}
}

// ---- match_expr_inference_test.go ----

func TestMatchExprInferenceBindsOptionalSomePayload(t *testing.T) {
	checked := checkMatchExprInferenceSource(t, `
func main() -> Int:
    let value: String? = "abcd"
    let score = match value:
    case some(text):
        text.len
    case none:
        0
    return score
`)
	if got := checked.Funcs[0].Locals["score"].TypeName; got != "i32" {
		t.Fatalf("score type = %q, want i32", got)
	}
	if got := checked.Funcs[0].Locals["text"].TypeName; got != "str" {
		t.Fatalf("text binding type = %q, want str", got)
	}
}

func TestMatchExprInferenceBindsEnumPayloads(t *testing.T) {
	checked := checkMatchExprInferenceSource(t, `
enum Result:
    case ok(Int, String)
    case err(Int)

func main() -> Int:
    let result: Result = Result.ok(40, "xy")
    let score = match result:
    case Result.ok(code, text):
        code + text.len
    case Result.err(errCode):
        errCode
    return score
`)
	locals := checked.Funcs[0].Locals
	if got := locals["score"].TypeName; got != "i32" {
		t.Fatalf("score type = %q, want i32", got)
	}
	if got := locals["code"].TypeName; got != "i32" {
		t.Fatalf("code binding type = %q, want i32", got)
	}
	if got := locals["errCode"].TypeName; got != "i32" {
		t.Fatalf("errCode binding type = %q, want i32", got)
	}
	if got := locals["text"].TypeName; got != "str" {
		t.Fatalf("text binding type = %q, want str", got)
	}
}

func TestMatchExprInferenceRejectsCaseTypeMismatch(t *testing.T) {
	err := checkMatchExprInferenceError(t, `
func main() -> Int:
    let value: Int? = 1
    let score = match value:
    case some(x):
        x
    case none:
        "bad"
    return score
`)
	if !strings.Contains(err.Error(), "match expression case type mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func checkMatchExprInferenceSource(t *testing.T, src string) *CheckedProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return checked
}

func checkMatchExprInferenceError(t *testing.T, src string) error {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected Check error")
	}
	return err
}

// ---- memory_boundary_handoff_test.go ----

func TestMemoryBoundaryHandoffAuditCoversP10PlanRows(t *testing.T) {
	report := MemoryBoundaryHandoffAudit()
	if err := ValidateMemoryBoundaryHandoffAudit(report); err != nil {
		t.Fatalf("ValidateMemoryBoundaryHandoffAudit failed: %v", err)
	}
	if report.SchemaVersion != "tetra.memory.boundary_handoff.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.FullActorRuntimeClaimed {
		t.Fatalf("P10 audit must not claim full actor runtime production")
	}
	if !hasBoundaryHandoffText(report.NonClaims, "full production actor runtime is not claimed") {
		t.Fatalf("nonclaims = %#v, want actor runtime nonclaim", report.NonClaims)
	}

	byID := map[MemoryBoundaryHandoffID]MemoryBoundaryHandoffRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
	}
	expected := []MemoryBoundaryHandoffID{
		MemoryBoundaryActorBorrowRejected,
		MemoryBoundaryTaskBorrowRejected,
		MemoryBoundaryRequestRegionScoped,
		MemoryBoundaryUnsafeSafeMessageRejected,
		MemoryBoundaryStaleEpochRejected,
		MemoryBoundaryIslandMoveLinear,
		MemoryBoundaryActorRuntimeNonClaim,
	}
	for _, id := range expected {
		row, ok := byID[id]
		if !ok {
			t.Fatalf("missing P10 boundary handoff row %q", id)
		}
		if row.Status != MemoryBoundaryImplementedNarrow {
			t.Fatalf("row %q status = %q, want %q", id, row.Status, MemoryBoundaryImplementedNarrow)
		}
		if row.Evidence == "" || row.Boundary == "" {
			t.Fatalf("row %q missing evidence/boundary: %#v", id, row)
		}
	}

	requireBoundaryHandoffFacts(
		t,
		byID[MemoryBoundaryActorBorrowRejected],
		"cannot send borrowed view across actor boundary",
		".copy()",
	)
	requireBoundaryHandoffFacts(
		t,
		byID[MemoryBoundaryTaskBorrowRejected],
		"typed task error payload must be sendable across task boundary",
	)
	requireBoundaryHandoffFacts(
		t,
		byID[MemoryBoundaryRequestRegionScoped],
		"RequestRegionScope",
		"TaskRegionScope",
		"reset",
	)
	requireBoundaryHandoffFacts(
		t,
		byID[MemoryBoundaryUnsafeSafeMessageRejected],
		"ptr",
		"cap.mem",
		"typed actor message payload must be value-only",
	)
	requireBoundaryHandoffFacts(
		t,
		byID[MemoryBoundaryStaleEpochRejected],
		"core.island_reset",
		"cannot use consumed value",
	)
	requireBoundaryHandoffFacts(
		t,
		byID[MemoryBoundaryIslandMoveLinear],
		"core.send_typed",
		"cannot use consumed value",
		"island",
	)
	requireBoundaryHandoffFacts(
		t,
		byID[MemoryBoundaryActorRuntimeNonClaim],
		"not a production actor runtime",
	)
}

func TestMemoryBoundaryHandoffAuditRejectsFakeScopeClaims(t *testing.T) {
	report := MemoryBoundaryHandoffAudit()

	fakeClaim := report
	fakeClaim.FullActorRuntimeClaimed = true
	if err := ValidateMemoryBoundaryHandoffAudit(fakeClaim); err == nil ||
		!strings.Contains(err.Error(), "full production actor runtime") {
		t.Fatalf("fake full actor runtime claim error = %v", err)
	}

	missingStale := cloneMemoryBoundaryHandoffReport(report)
	var rows []MemoryBoundaryHandoffRow
	for _, row := range missingStale.Rows {
		if row.ID != MemoryBoundaryStaleEpochRejected {
			rows = append(rows, row)
		}
	}
	missingStale.Rows = rows
	if err := ValidateMemoryBoundaryHandoffAudit(missingStale); err == nil ||
		!strings.Contains(err.Error(), "stale_epoch_rejected") {
		t.Fatalf("missing stale row error = %v", err)
	}

	noNonclaim := cloneMemoryBoundaryHandoffReport(report)
	noNonclaim.NonClaims = nil
	if err := ValidateMemoryBoundaryHandoffAudit(noNonclaim); err == nil ||
		!strings.Contains(err.Error(), "nonclaim") {
		t.Fatalf("missing nonclaim error = %v", err)
	}
}

func requireBoundaryHandoffFacts(t *testing.T, row MemoryBoundaryHandoffRow, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !hasBoundaryHandoffText(row.RequiredFacts, want) {
			t.Fatalf("row %q missing fact %q: %#v", row.ID, want, row.RequiredFacts)
		}
	}
}

func hasBoundaryHandoffText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func cloneMemoryBoundaryHandoffReport(
	report MemoryBoundaryHandoffReport,
) MemoryBoundaryHandoffReport {
	clone := report
	clone.Rows = append([]MemoryBoundaryHandoffRow(nil), report.Rows...)
	clone.NonClaims = append([]string(nil), report.NonClaims...)
	return clone
}

// ---- region_ownership_test.go ----

func TestCheckNoConsumedDescendantsCanonicalizesAliasPaths(t *testing.T) {
	t.Run("query_path_is_alias_to_consumed_parent", func(t *testing.T) {
		state := newRegionState(nil)
		state.bindOwnershipAlias("raw", "msg")
		state.markConsumed("msg", frontend.Position{})

		err := state.checkNoConsumedDescendants("raw", frontend.Position{})
		if err == nil {
			t.Fatalf("expected consumed value error, got nil")
		}
		if !strings.Contains(err.Error(), "'msg'") {
			t.Fatalf("error = %q, want canonical path 'msg'", err.Error())
		}
		if strings.Contains(err.Error(), "'raw'") {
			t.Fatalf("error = %q, should not use alias name 'raw'", err.Error())
		}
	})

	t.Run("query_nested_alias_resolves_to_canonical_descendant", func(t *testing.T) {
		state := newRegionState(nil)
		state.bindOwnershipAlias("raw", "msg")
		state.markConsumed("msg.$case0.payload0", frontend.Position{})

		err := state.checkNoConsumedDescendants("raw.$case0.payload0", frontend.Position{})
		if err == nil {
			t.Fatalf("expected consumed value error, got nil")
		}
		if !strings.Contains(err.Error(), "msg.$case0.payload0") {
			t.Fatalf("error = %q, want canonical path 'msg.$case0.payload0'", err.Error())
		}
		if strings.Contains(err.Error(), "raw.$case0.payload0") {
			t.Fatalf("error = %q, should not use alias path 'raw.$case0.payload0'", err.Error())
		}
	})
}

func TestCheckNotConsumedCanonicalizesAliasPaths(t *testing.T) {
	state := newRegionState(nil)
	state.bindOwnershipAlias("raw", "msg")
	state.markConsumed("msg", frontend.Position{
		File: "app/main.t4",
		Line: 12,
		Col:  3,
	})

	err := state.checkNotConsumed("raw", frontend.Position{})
	if err == nil {
		t.Fatalf("expected consumed value error, got nil")
	}
	if !strings.Contains(err.Error(), "'msg'") {
		t.Fatalf("error = %q, want canonical path 'msg'", err.Error())
	}
	if strings.Contains(err.Error(), "'raw'") {
		t.Fatalf("error = %q, should not use alias name 'raw'", err.Error())
	}
}

func TestCheckNotConsumedCanonicalizesAliasPathsWhenConsumedBeforeAlias(t *testing.T) {
	state := newRegionState(nil)
	state.markConsumed("raw", frontend.Position{
		File: "app/main.t4",
		Line: 7,
		Col:  2,
	})
	state.bindOwnershipAlias("raw", "msg")

	err := state.checkNotConsumed("raw", frontend.Position{})
	if err == nil {
		t.Fatalf("expected consumed value error, got nil")
	}
	if !strings.Contains(err.Error(), "msg") {
		t.Fatalf("error = %q, want canonical path 'msg'", err.Error())
	}
	if strings.Contains(err.Error(), "raw") {
		t.Fatalf("error = %q, should not use alias name 'raw'", err.Error())
	}
}

func TestCheckNotConsumedNestedAliasCanonicalizesPath(t *testing.T) {
	state := newRegionState(nil)
	state.bindOwnershipAlias("raw", "msg")
	state.markConsumed("msg.$case0.payload0", frontend.Position{})

	err := state.checkNotConsumed("raw.$case0.payload0", frontend.Position{})
	if err == nil {
		t.Fatalf("expected consumed value error, got nil")
	}
	if !strings.Contains(err.Error(), "msg.$case0.payload0") {
		t.Fatalf("error = %q, want canonical path 'msg.$case0.payload0'", err.Error())
	}
	if strings.Contains(err.Error(), "raw.$case0.payload0") {
		t.Fatalf("error = %q, should not use alias path 'raw.$case0.payload0'", err.Error())
	}
}

func TestClearConsumedTreeClearsAliasEquivalentPaths(t *testing.T) {
	state := newRegionState(nil)
	state.bindOwnershipAlias("raw", "msg")
	state.markConsumed("msg.$case0.payload0", frontend.Position{})
	state.markConsumed("msg", frontend.Position{})

	state.clearConsumedTree("raw.$case0.payload0")
	if _, ok := state.consumedVars["msg.$case0.payload0"]; ok {
		t.Fatalf("expected 'msg.$case0.payload0' to be cleared")
	}
	if _, ok := state.consumedVars["raw.$case0.payload0"]; ok {
		t.Fatalf("expected 'raw.$case0.payload0' to be cleared")
	}
	if _, ok := state.consumedVars["msg"]; !ok {
		t.Fatalf(
			"expected 'msg' to remain when clearing descendant only? got %v",
			state.consumedVars["msg"],
		)
	}
}

func TestClearConsumedTreeClearsAliasForConsumedBasePath(t *testing.T) {
	state := newRegionState(nil)
	state.bindOwnershipAlias("raw", "msg")
	state.markConsumed("raw", frontend.Position{})

	state.clearConsumedTree("msg")
	if _, ok := state.consumedVars["raw"]; ok {
		t.Fatalf("expected alias key 'raw' to be cleared")
	}
	if _, ok := state.consumedVars["msg"]; ok {
		t.Fatalf("expected canonical key 'msg' to be cleared")
	}
}

func TestMergeOwnershipAliasesIntersectsOnlyMatchingMappings(t *testing.T) {
	left := map[string]string{
		"common":   "root",
		"leftOnly": "msg",
	}
	right := map[string]string{
		"common":    "root",
		"rightOnly": "other",
	}
	merged := mergeOwnershipAliases(left, right)
	if got := len(merged); got != 1 {
		t.Fatalf("expected exactly one merged alias, got %d (%v)", got, merged)
	}
	if _, ok := merged["common"]; !ok {
		t.Fatalf("expected 'common' to remain after merge, got %v", merged)
	}
	if merged["common"] != "root" {
		t.Fatalf("expected 'common' to map to 'root', got %q", merged["common"])
	}
	if _, ok := merged["leftOnly"]; ok {
		t.Fatalf("did not expect 'leftOnly' in merged aliases: %v", merged)
	}
	if _, ok := merged["rightOnly"]; ok {
		t.Fatalf("did not expect 'rightOnly' in merged aliases: %v", merged)
	}
}

func TestMergeFlowWithLabelsIntersectsOwnershipAliases(t *testing.T) {
	state := newRegionState(nil)
	left := flowSnapshot{
		reachable: true,
		consumedVars: map[string]frontend.Position{
			"raw": {},
		},
		maybeConsumedVars: map[string]ownershipJoinConflict{},
		ownershipAliases: map[string]string{
			"common":   "root",
			"leftOnly": "msg",
		},
		borrowedPtrAliases: map[string]string{},
		consumedResources:  map[int]frontend.Position{},
		resourceVars:       map[string]int{},
		unknownResources:   map[int]bool{},
		finalizedResources: map[int]resourceFinalization{},
	}
	right := flowSnapshot{
		reachable: true,
		consumedVars: map[string]frontend.Position{
			"raw": {},
		},
		maybeConsumedVars: map[string]ownershipJoinConflict{},
		ownershipAliases: map[string]string{
			"common":    "root",
			"rightOnly": "other",
		},
		borrowedPtrAliases: map[string]string{},
		consumedResources:  map[int]frontend.Position{},
		resourceVars:       map[string]int{},
		unknownResources:   map[int]bool{},
		finalizedResources: map[int]resourceFinalization{},
	}

	mergeFlowWithLabels(state, left, right, "left", "right")
	if len(state.ownershipAliases) != 1 {
		t.Fatalf(
			"expected exactly one ownership alias after merge, got %d (%v)",
			len(state.ownershipAliases),
			state.ownershipAliases,
		)
	}
	if value, ok := state.ownershipAliases["common"]; !ok || value != "root" {
		t.Fatalf(
			"expected alias common->root, got %v (ok=%v)",
			state.ownershipAliases["common"],
			ok,
		)
	}
	if _, ok := state.ownershipAliases["leftOnly"]; ok {
		t.Fatalf("did not expect leftOnly alias after merge: %v", state.ownershipAliases)
	}
	if _, ok := state.ownershipAliases["rightOnly"]; ok {
		t.Fatalf("did not expect rightOnly alias after merge: %v", state.ownershipAliases)
	}
}

// ---- representation_metadata_test.go ----

func TestRepresentationMetadataRegistryCoversMemoryIdealV0Names(t *testing.T) {
	want := map[string]bool{
		"ptr":           true,
		"len":           true,
		"owner_id":      true,
		"region_id":     true,
		"provenance_id": true,
		"borrow_source": true,
		"storage_class": true,
		"unsafe_class":  true,
	}
	for _, field := range representationMetadataRegistry {
		if !want[field.Name] {
			t.Fatalf("unexpected representation metadata field %q", field.Name)
		}
		delete(want, field.Name)
		if field.AssignableInSafeCode {
			t.Fatalf("representation metadata field %q is assignable in safe code", field.Name)
		}
		if field.SourceFactKind != representationMetadataSourceFactKind {
			t.Fatalf(
				"representation metadata field %q source fact kind = %q",
				field.Name,
				field.SourceFactKind,
			)
		}
	}
	for name := range want {
		t.Fatalf("representation metadata registry missing %q", name)
	}
}

func TestRepresentationMetadataRegistryReservesMemoryIdealV0Names(t *testing.T) {
	for _, name := range []string{
		"ptr",
		"len",
		"owner_id",
		"region_id",
		"provenance_id",
		"borrow_source",
		"storage_class",
		"unsafe_class",
	} {
		if !isReservedRepresentationMetadataField(name) {
			t.Fatalf("%q is not reserved representation metadata", name)
		}
	}
}

func TestTypeModelDoesNotExposeSliceMetadataAsWritableField(t *testing.T) {
	types := baseTypes()
	info, err := ensureTypeInfo("[]u8", types)
	if err != nil {
		t.Fatalf("ensure []u8: %v", err)
	}
	for _, name := range []string{"ptr", "len"} {
		field, ok := info.FieldMap[name]
		if !ok {
			t.Fatalf("[]u8 metadata field %q missing from internal layout", name)
		}
		if field.UserAssignable {
			t.Fatalf("[]u8 metadata field %q is user-assignable", name)
		}
	}
}

func TestTypeModelDoesNotExposeStringMetadataAsWritableField(t *testing.T) {
	types := baseTypes()
	info := types["str"]
	if info == nil {
		t.Fatal("str type missing")
	}
	for _, name := range []string{"ptr", "len"} {
		field, ok := info.FieldMap[name]
		if !ok {
			t.Fatalf("str metadata field %q missing from internal layout", name)
		}
		if field.UserAssignable {
			t.Fatalf("str metadata field %q is user-assignable", name)
		}
	}
}

// ---- resolution_epic04_test.go ----

func TestResolutionModuleImportAliasResolvesCallAndType(t *testing.T) {
	pos := frontend.Position{File: "app/main.tetra", Line: 2, Col: 5}
	imports := map[string]string{"math": "engine.math"}

	callName, err := resolveCallName("math.add_one", "app.main", imports, pos)
	if err != nil {
		t.Fatalf("resolveCallName: %v", err)
	}
	if callName != "engine.math.add_one" {
		t.Fatalf("resolved call = %q, want engine.math.add_one", callName)
	}

	ref := frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed, Name: "math.Vec2"}
	typeName, err := resolveTypeName(&ref, "app.main", imports)
	if err != nil {
		t.Fatalf("resolveTypeName: %v", err)
	}
	if typeName != "engine.math.Vec2" {
		t.Fatalf("resolved type = %q, want engine.math.Vec2", typeName)
	}
}

func TestResolutionDiagnosticForInvalidAliasCallShape(t *testing.T) {
	pos := frontend.Position{File: "app/main.tetra", Line: 4, Col: 7}
	_, err := resolveCallName("math.", "app.main", map[string]string{"math": "engine.math"}, pos)
	if err == nil {
		t.Fatalf("expected alias call shape error")
	}
	if !strings.Contains(err.Error(), "app/main.tetra:4:7: expected 'math.<func>'") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolutionDisplayTextForEnumCaseUsesLocalTypeName(t *testing.T) {
	pos := frontend.Position{File: "app/main.tetra", Line: 6, Col: 10}
	types := map[string]*TypeInfo{
		"app.main.Color": {
			Name: "app.main.Color",
			Kind: TypeEnum,
			CaseMap: map[string]EnumCaseInfo{
				"Red": {Name: "Red", Ordinal: 0},
			},
		},
	}
	expr := &frontend.FieldAccessExpr{
		At:    pos,
		Base:  &frontend.IdentExpr{Name: "Color", At: pos},
		Field: "Blue",
	}

	_, _, ok, err := resolveEnumCaseExpr(expr, nil, nil, types, "app.main", nil)
	if !ok {
		t.Fatalf("expected enum resolution path")
	}
	if err == nil {
		t.Fatalf("expected unknown enum case error")
	}
	if !strings.Contains(err.Error(), "unknown enum case 'Blue' for 'Color'") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolutionImportAliasConflictWithTopLevelDeclaration(t *testing.T) {
	file := &frontend.FileAST{
		Imports: []frontend.ImportDecl{
			{
				Path:  "engine.math",
				Alias: "math",
				At:    frontend.Position{File: "app/main.tetra", Line: 2, Col: 1},
			},
		},
		Funcs: []*frontend.FuncDecl{
			{Name: "math", Pos: frontend.Position{File: "app/main.tetra", Line: 3, Col: 1}},
		},
	}

	_, err := collectImportAliases(file)
	if err == nil {
		t.Fatalf("expected alias conflict error")
	}
	if !strings.Contains(err.Error(), "import alias 'math' conflicts with declaration 'math'") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolutionImportAliasRequiredBoundary(t *testing.T) {
	file := &frontend.FileAST{
		Imports: []frontend.ImportDecl{
			{
				Path: "engine.math",
				At:   frontend.Position{File: "app/main.tetra", Line: 2, Col: 1},
			},
		},
	}

	_, err := collectImportAliases(file)
	if err == nil {
		t.Fatalf("expected alias required error")
	}
	if !strings.Contains(err.Error(), "app/main.tetra:2:1: import alias required") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolutionImportPathRequiredBoundary(t *testing.T) {
	file := &frontend.FileAST{
		Imports: []frontend.ImportDecl{
			{
				Path:  "",
				Alias: "math",
				At:    frontend.Position{File: "app/main.tetra", Line: 2, Col: 1},
			},
		},
	}

	_, err := collectImportAliases(file)
	if err == nil {
		t.Fatalf("expected import path required error")
	}
	if !strings.Contains(err.Error(), "app/main.tetra:2:1: import path required") {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckWorldRejectsDuplicateInterfaceFunctionParams(t *testing.T) {
	iface, err := frontend.ParseFile([]byte(`module lib.api

pub func dup(x: Int, x: Int) -> Int:
    return x
`), "lib/api.t4i")
	if err != nil {
		t.Fatalf("ParseFile iface: %v", err)
	}

	_, err = CheckWorldOpt(&module.World{
		EntryModule:      "lib.api",
		Files:            []*frontend.FileAST{iface},
		InterfaceModules: map[string]bool{"lib.api": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.api": iface,
		},
	}, CheckOptions{RequireMain: false})
	if err == nil {
		t.Fatalf("expected duplicate parameter error")
	}
	if !strings.Contains(err.Error(), "duplicate parameter 'x'") {
		t.Fatalf("error = %v", err)
	}
}

// ---- resolution_test.go ----

func TestResolveTypeNameUnsupportedPathsArePositioned(t *testing.T) {
	pos := frontend.Position{File: "bad_types.tetra", Line: 3, Col: 7}
	tests := []struct {
		name string
		ref  frontend.TypeRef
		want string
	}{
		{
			name: "unsupported kind",
			ref:  frontend.TypeRef{At: pos, Kind: frontend.TypeRefKind(99)},
			want: "bad_types.tetra:3:7: unsupported type reference kind 99",
		},
		{
			name: "missing slice element",
			ref:  frontend.TypeRef{At: pos, Kind: frontend.TypeRefSlice},
			want: "bad_types.tetra:3:7: missing slice element type",
		},
		{
			name: "missing array element",
			ref:  frontend.TypeRef{At: pos, Kind: frontend.TypeRefArray},
			want: "bad_types.tetra:3:7: missing array element type",
		},
		{
			name: "missing optional payload",
			ref:  frontend.TypeRef{At: pos, Kind: frontend.TypeRefOptional},
			want: "bad_types.tetra:3:7: missing optional payload type",
		},
		{
			name: "missing named type",
			ref:  frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed},
			want: "bad_types.tetra:3:7: missing type name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolveTypeName(&tt.ref, "main", nil)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestValidateGenericTypeRefUnsupportedKindIsActionable(t *testing.T) {
	err := validateGenericTypeRef(frontend.TypeRef{
		At:   frontend.Position{File: "generic.tetra", Line: 9, Col: 11},
		Kind: frontend.TypeRefKind(77),
	}, map[string]struct{}{"T": {}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(
		err.Error(),
		"generic.tetra:9:11: unsupported generic type reference kind 77",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveTypeNameFunctionTypeRefMVP(t *testing.T) {
	ref := frontend.TypeRef{
		At:   frontend.Position{File: "fn_types.tetra", Line: 2, Col: 9},
		Kind: frontend.TypeRefFunction,
		Params: []frontend.TypeRef{
			{
				At:   frontend.Position{File: "fn_types.tetra", Line: 2, Col: 12},
				Kind: frontend.TypeRefNamed,
				Name: "Int",
			},
			{
				At:   frontend.Position{File: "fn_types.tetra", Line: 2, Col: 17},
				Kind: frontend.TypeRefNamed,
				Name: "Bool",
			},
		},
		Return: &frontend.TypeRef{
			At:   frontend.Position{File: "fn_types.tetra", Line: 2, Col: 26},
			Kind: frontend.TypeRefNamed,
			Name: "UInt8",
		},
	}
	got, err := resolveTypeName(&ref, "main", nil)
	if err != nil {
		t.Fatalf("resolveTypeName: %v", err)
	}
	if got != "fnptr" {
		t.Fatalf("resolved = %q, want fnptr", got)
	}
}

func TestResolveTypeNameFunctionTypeRefRequiresReturn(t *testing.T) {
	_, err := resolveTypeName(&frontend.TypeRef{
		At:   frontend.Position{File: "fn_types.tetra", Line: 4, Col: 5},
		Kind: frontend.TypeRefFunction,
		Params: []frontend.TypeRef{
			{
				At:   frontend.Position{File: "fn_types.tetra", Line: 4, Col: 8},
				Kind: frontend.TypeRefNamed,
				Name: "Int",
			},
		},
	}, "main", nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "fn_types.tetra:4:5: missing function return type") {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckWorldAliasesImportedPublicFunctionTypedGlobals(t *testing.T) {
	lib, err := frontend.ParseFile([]byte(`module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`), "lib/math.t4")
	if err != nil {
		t.Fatalf("parse lib: %v", err)
	}
	app, err := frontend.ParseFile([]byte(`module app.main
import lib.math as math

func main() -> Int:
    return 0
`), "app/main.t4")
	if err != nil {
		t.Fatalf("parse app: %v", err)
	}
	selective, err := frontend.ParseFile([]byte(`module app.selective
import lib.math.{cb}

func probe() -> Int:
    return 0
`), "app/selective.t4")
	if err != nil {
		t.Fatalf("parse selective app: %v", err)
	}
	world := &module.World{
		EntryModule: app.Module,
		Files:       []*frontend.FileAST{lib, app, selective},
		ByModule: map[string]*frontend.FileAST{
			lib.Module:       lib,
			app.Module:       app,
			selective.Module: selective,
		},
	}

	checked, err := CheckWorldOpt(world, CheckOptions{RequireMain: false})
	if err != nil {
		t.Fatalf("CheckWorldOpt: %v", err)
	}
	globals := checked.GlobalsByModule[app.Module]
	for _, name := range []string{"math.cb", "lib.math.cb"} {
		global, ok := globals[name]
		if !ok {
			t.Fatalf("missing imported function-typed global alias %q in %#v", name, globals)
		}
		if !global.FunctionTypeValue || global.FunctionValue != "lib.math.add2" || global.Mutable {
			t.Fatalf(
				"alias %q = %#v, want immutable function-typed global backed by lib.math.add2",
				name,
				global,
			)
		}
	}
	selectiveGlobal, ok := checked.GlobalsByModule[selective.Module]["cb"]
	if !ok {
		t.Fatalf(
			"missing selective imported function-typed global alias cb in %#v",
			checked.GlobalsByModule[selective.Module],
		)
	}
	if !selectiveGlobal.FunctionTypeValue || selectiveGlobal.FunctionValue != "lib.math.add2" ||
		selectiveGlobal.Mutable {
		t.Fatalf(
			"selective alias cb = %#v, want immutable function-typed global backed by lib.math.add2",
			selectiveGlobal,
		)
	}
}

func TestCheckWorldAliasesImportedMutableFunctionTypedGlobalsAsBoundary(t *testing.T) {
	lib, err := frontend.ParseFile([]byte(`module lib.math

pub var cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`), "lib/math.t4")
	if err != nil {
		t.Fatalf("parse lib: %v", err)
	}
	app, err := frontend.ParseFile([]byte(`module app.main
import lib.math as math

func main() -> Int:
    return 0
`), "app/main.t4")
	if err != nil {
		t.Fatalf("parse app: %v", err)
	}
	selective, err := frontend.ParseFile([]byte(`module app.selective
import lib.math.{cb}

func probe() -> Int:
    return 0
`), "app/selective.t4")
	if err != nil {
		t.Fatalf("parse selective app: %v", err)
	}
	world := &module.World{
		EntryModule: app.Module,
		Files:       []*frontend.FileAST{lib, app, selective},
		ByModule: map[string]*frontend.FileAST{
			lib.Module:       lib,
			app.Module:       app,
			selective.Module: selective,
		},
	}

	checked, err := CheckWorldOpt(world, CheckOptions{RequireMain: false})
	if err != nil {
		t.Fatalf("CheckWorldOpt: %v", err)
	}
	globals := checked.GlobalsByModule[app.Module]
	for _, name := range []string{"math.cb", "lib.math.cb"} {
		global, ok := globals[name]
		if !ok {
			t.Fatalf(
				"missing imported mutable function-typed global alias %q in %#v",
				name,
				globals,
			)
		}
		if !global.FunctionTypeValue || !global.Mutable || global.FunctionValue != "" {
			t.Fatalf(
				"alias %q = %#v, want mutable function-typed boundary alias without static function value",
				name,
				global,
			)
		}
	}
	selectiveGlobal, ok := checked.GlobalsByModule[selective.Module]["cb"]
	if !ok {
		t.Fatalf(
			"missing selective imported mutable function-typed global alias cb in %#v",
			checked.GlobalsByModule[selective.Module],
		)
	}
	if !selectiveGlobal.FunctionTypeValue || !selectiveGlobal.Mutable ||
		selectiveGlobal.FunctionValue != "" {
		t.Fatalf(
			("selective alias cb = %#v, want mutable function-typed boundary " +
				"alias without static function value"),
			selectiveGlobal,
		)
	}
}
