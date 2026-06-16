package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/semantics"
)

func TestCanonicalSelfHostRuntimeSources(t *testing.T) {
	tests := []struct {
		path       string
		wantModule string
	}{
		{filepath.Join("..", "__rt", "actors_sysv.tetra"), "__rt.actors_sysv"},
		{filepath.Join("..", "__rt", "actors_i386.tetra"), "__rt.actors_i386"},
		{filepath.Join("..", "__rt", "actors_win64.tetra"), "__rt.actors_win64"},
		{filepath.Join("selfhostrt", "actors_sysv.tetra"), "__rt.actors_sysv"},
		{filepath.Join("selfhostrt", "actors_i386.tetra"), "__rt.actors_i386"},
		{filepath.Join("selfhostrt", "actors_win64.tetra"), "__rt.actors_win64"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			raw, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("read runtime source: %v", err)
			}
			file, err := frontend.ParseFile(raw, tt.path)
			if err != nil {
				t.Fatalf("parse runtime source: %v", err)
			}
			if file.Module != tt.wantModule {
				t.Fatalf("module = %q, want %q", file.Module, tt.wantModule)
			}
		})
	}
}

func TestSelfHostRuntimeObjectsExportRequiredSymbols(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		target string
	}{
		{"sysv-linux", filepath.Join("..", "__rt", "actors_sysv.tetra"), "linux-x64"},
		{"sysv-macos", filepath.Join("..", "__rt", "actors_sysv.tetra"), "macos-x64"},
		{"sysv-linux-x32", filepath.Join("..", "__rt", "actors_sysv.tetra"), "linux-x32"},
		{"win64", filepath.Join("..", "__rt", "actors_win64.tetra"), "windows-x64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			objPath := filepath.Join(tmp, "runtime.tobj")
			if _, err := BuildFileWithStatsOpt(tt.src, objPath, tt.target, BuildOptions{Emit: EmitLibrary}); err != nil {
				t.Fatalf("build runtime object: %v", err)
			}
			obj, err := ReadObject(objPath)
			if err != nil {
				t.Fatalf("read runtime object: %v", err)
			}
			required := append(requiredActorRuntimeSymbols(), requiredTimeRuntimeSymbols()...)
			required = append(required, requiredActorStateRuntimeSymbols()...)
			required = append(required, requiredTypedTaskRuntimeSymbols(8)...)
			assertObjectHasSymbols(t, obj, required...)
		})
	}
}

func TestRequiredTimeRuntimeSymbols(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredTimeRuntimeSymbols() {
		got[name] = struct{}{}
	}

	for _, name := range []string{
		"__tetra_time_now_ms",
		"__tetra_sleep_ms",
		"__tetra_sleep_until_ms",
		"__tetra_deadline_ms",
		"__tetra_timer_ready_ms",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required time runtime symbols missing %q", name)
		}
	}
}

func TestRequiredActorRuntimeSymbolsIncludeTaggedMessageABI(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredActorRuntimeSymbols() {
		got[name] = struct{}{}
	}

	for _, name := range []string{
		"__tetra_actor_send_msg",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_poll",
		"__tetra_actor_recv_until",
		"__tetra_actor_send_begin",
		"__tetra_actor_send_slot",
		"__tetra_actor_send_commit",
		"__tetra_actor_recv_msg_until",
		"__tetra_actor_recv_begin",
		"__tetra_actor_recv_slot",
		"__tetra_actor_recv_count",
		"__tetra_actor_yield_now",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required actor runtime symbols missing tagged message ABI symbol %q", name)
		}
	}
}

func TestRequiredActorStateRuntimeSymbols(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredActorStateRuntimeSymbols() {
		got[name] = struct{}{}
	}
	for _, name := range []string{
		"__tetra_actor_state_load",
		"__tetra_actor_state_store",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required actor-state runtime symbols missing %q", name)
		}
	}
}

func TestActorGlueExportsProgramRuntimeSymbols(t *testing.T) {
	dispatchFn, err := buildActorDispatchFunc([]string{"main", "pong"}, nil)
	if err != nil {
		t.Fatalf("build dispatch: %v", err)
	}
	mainIDFn, err := buildActorMainEntryIDFunc("main")
	if err != nil {
		t.Fatalf("build main entry id: %v", err)
	}
	obj, err := CodegenObjectLinuxX64([]IRFunc{dispatchFn, mainIDFn})
	if err != nil {
		t.Fatalf("codegen glue object: %v", err)
	}
	assertObjectHasSymbols(t, obj, "__tetra_actor_dispatch", "__tetra_actor_main_entry_id")
}

func TestActorDispatchStateInitializationMatchesRuntimeStoreABI(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Funcs: []semantics.CheckedFunc{
			{
				Name: "Counter.run",
				ActorState: map[string]semantics.ActorStateField{
					"count": {Name: "count", Slot: 0, TypeName: "Int", Mutable: true, Init: 7},
				},
			},
		},
	}
	dispatchFn, err := buildActorDispatchFunc([]string{"Counter.run"}, checked)
	if err != nil {
		t.Fatalf("build dispatch: %v", err)
	}
	if err := lower.VerifyFunc(dispatchFn); err != nil {
		t.Fatalf("dispatch verifier: %v", err)
	}

	for _, instr := range dispatchFn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == "__tetra_actor_state_store" {
			if instr.ArgSlots != 2 || instr.RetSlots != 1 {
				t.Fatalf("state store ABI = args %d rets %d, want args 2 rets 1", instr.ArgSlots, instr.RetSlots)
			}
			return
		}
	}
	t.Fatalf("dispatch missing __tetra_actor_state_store call: %#v", dispatchFn.Instrs)
}

func TestGeneratedActorGlueIsVerifiedBeforeNativeCodegen(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Funcs: []semantics.CheckedFunc{
			{
				Name: "stateful",
				ActorState: map[string]semantics.ActorStateField{
					"count": {Name: "count", Slot: 0, TypeName: "Int", Mutable: true, Init: 1},
				},
			},
		},
	}

	codegenCalled := false
	native := nativeBuildTarget{
		triple: "linux-x64",
		backend: nativeExecutableBackend{
			actorRuntime: func(actorEntries []string) (*Object, error) {
				symbolNames := append([]string{}, requiredActorRuntimeSymbols()...)
				symbolNames = append(symbolNames, requiredActorStateRuntimeSymbols()...)
				symbolNames = append(symbolNames, "__tetra_actor_main_entry_id")
				symbols := make([]Symbol, 0, len(symbolNames))
				for _, name := range symbolNames {
					symbols = append(symbols, Symbol{Name: name})
				}
				return &Object{Symbols: symbols}, nil
			},
			link: func(outputPath string, objects []*Object, mainName string) error {
				return nil
			},
		},
		codegen: func(funcs []IRFunc, dataPrefix [][]byte) (*Object, error) {
			for _, fn := range funcs {
				if fn.Name == "__tetra_actor_dispatch" {
					codegenCalled = true
				}
			}
			return &Object{}, nil
		},
	}

	err := linkNativeExecutable(filepath.Join(t.TempDir(), "out"), native, BuildOptions{}, checked, nil, nil)
	if err == nil || !strings.Contains(err.Error(), `call is missing target name`) {
		t.Fatalf("linkNativeExecutable error = %v, want generated IR verifier error", err)
	}
	if codegenCalled {
		t.Fatalf("generated actor glue reached native codegen before verifier")
	}
}

func assertObjectHasSymbols(t *testing.T, obj *Object, names ...string) {
	t.Helper()
	symbols := make(map[string]struct{}, len(obj.Symbols))
	for _, sym := range obj.Symbols {
		symbols[sym.Name] = struct{}{}
	}
	for _, name := range names {
		if _, ok := symbols[name]; !ok {
			t.Fatalf("missing symbol %q", name)
		}
	}
}
