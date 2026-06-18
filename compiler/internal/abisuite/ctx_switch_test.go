package abisuite

import (
	"reflect"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestCtxSwitchObjectSmokesUseBackendCallbacks(t *testing.T) {
	var calls []string
	deps := CtxSwitchDeps{
		BuildX86Object: func(funcs []ir.IRFunc) (CtxSwitchObject, error) {
			requireCtxSwitchSmokeIR(t, funcs, "__tetra_x86_ctx_switch_smoke")
			calls = append(calls, "x86:"+funcs[0].Name)
			code := []byte{0x90}
			code = append(code, ctxSwitchI386Stub()...)
			code = append(code, []byte{0x31, 0xC0, 0x50}...)
			return CtxSwitchObject{Target: "linux-x86", Code: code}, nil
		},
		BuildX32Object: func(funcs []ir.IRFunc) (CtxSwitchObject, error) {
			requireCtxSwitchSmokeIR(t, funcs, "__tetra_x32_ctx_switch_smoke")
			calls = append(calls, "x32:"+funcs[0].Name)
			code := []byte{0x90}
			code = append(code, ctxSwitchX32SysVStub()...)
			code = append(code, []byte{0x31, 0xC0, 0x50}...)
			return CtxSwitchObject{Target: "linux-x32", Code: code}, nil
		},
	}

	if err := CheckX86CtxSwitchObjectSmoke(deps); err != nil {
		t.Fatalf("CheckX86CtxSwitchObjectSmoke: %v", err)
	}
	if err := CheckX32CtxSwitchObjectSmoke(deps); err != nil {
		t.Fatalf("CheckX32CtxSwitchObjectSmoke: %v", err)
	}
	wantCalls := []string{"x86:__tetra_x86_ctx_switch_smoke", "x32:__tetra_x32_ctx_switch_smoke"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("backend calls = %#v, want %#v", calls, wantCalls)
	}
}

func requireCtxSwitchSmokeIR(t *testing.T, funcs []ir.IRFunc, name string) {
	t.Helper()
	if len(funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(funcs))
	}
	fn := funcs[0]
	if fn.Name != name {
		t.Fatalf("func name = %q, want %q", fn.Name, name)
	}
	if fn.ReturnSlots != 1 {
		t.Fatalf("return slots = %d, want 1", fn.ReturnSlots)
	}
	gotKinds := make([]ir.IRInstrKind, 0, len(fn.Instrs))
	for _, instr := range fn.Instrs {
		gotKinds = append(gotKinds, instr.Kind)
	}
	wantKinds := []ir.IRInstrKind{
		ir.IRConstI32,
		ir.IRConstI32,
		ir.IRConstI32,
		ir.IRCtxSwitch,
		ir.IRReturn,
	}
	if !reflect.DeepEqual(gotKinds, wantKinds) {
		t.Fatalf("instruction kinds = %#v, want %#v", gotKinds, wantKinds)
	}
}
