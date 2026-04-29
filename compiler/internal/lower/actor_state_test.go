package lower

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerActorStateUsesRuntimeLoadStore(t *testing.T) {
	src := []byte(`
actor Counter:
    var count: Int = 1
    val enabled: Bool = true
    func run() -> Int:
        if enabled:
            count = count + 1
        return count

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Counter.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	run := findIRFuncByName(t, irProg.Funcs, "Counter.run")
	if !hasIRCallName(run, "__tetra_actor_state_load") {
		t.Fatalf("Counter.run is missing __tetra_actor_state_load call: %#v", run.Instrs)
	}
	if !hasIRCallName(run, "__tetra_actor_state_store") {
		t.Fatalf("Counter.run is missing __tetra_actor_state_store call: %#v", run.Instrs)
	}
}

func TestLowerActorStateExtendedScalarsUseRuntimeLoadStore(t *testing.T) {
	src := []byte(`
actor Counter:
    var err: task.error = 0
    var step: UInt8 = 1
    const boost: UInt16 = 2
    func run() -> Int:
        err = err + 1
        step = step + 1
        return err + step + boost

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Counter.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	run := findIRFuncByName(t, irProg.Funcs, "Counter.run")
	if !hasIRCallName(run, "__tetra_actor_state_load") {
		t.Fatalf("Counter.run is missing __tetra_actor_state_load call: %#v", run.Instrs)
	}
	if !hasIRCallName(run, "__tetra_actor_state_store") {
		t.Fatalf("Counter.run is missing __tetra_actor_state_store call: %#v", run.Instrs)
	}
}

func findIRFuncByName(t *testing.T, funcs []ir.IRFunc, name string) ir.IRFunc {
	t.Helper()
	for _, fn := range funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing IR function %q", name)
	return ir.IRFunc{}
}

func hasIRCallName(fn ir.IRFunc, name string) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			return true
		}
	}
	return false
}
