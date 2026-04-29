package lower

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerBoolSliceBuiltinsUseI32LayoutIR(t *testing.T) {
	src := []byte(`
func main() -> Int
uses alloc, islands, mem:
    var xs: []bool = make_bool(2)
    xs[0] = true
    xs[1] = false
    island(64) as isl:
        var ys: []bool = core.island_make_bool(isl, 1)
        ys[0] = xs[0]
    if xs[0]:
        return 1
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

	var mainFn *ir.IRFunc
	for i := range irProg.Funcs {
		if irProg.Funcs[i].Name == "main" {
			mainFn = &irProg.Funcs[i]
			break
		}
	}
	if mainFn == nil {
		t.Fatalf("main function not found in IR output")
	}

	makeI32Count := 0
	islandMakeI32Count := 0
	for _, instr := range mainFn.Instrs {
		switch instr.Kind {
		case ir.IRMakeSliceI32:
			makeI32Count++
		case ir.IRIslandMakeSliceI32:
			islandMakeI32Count++
		}
	}

	if makeI32Count == 0 {
		t.Fatalf("expected IRMakeSliceI32 for make_bool")
	}
	if islandMakeI32Count == 0 {
		t.Fatalf("expected IRIslandMakeSliceI32 for island_make_bool")
	}
}
