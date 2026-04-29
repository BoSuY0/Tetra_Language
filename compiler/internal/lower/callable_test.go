package lower

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerCallableFunctionValueEmitsSymAddrIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return 0
`, "main")

	if countInstr(fn.Instrs, ir.IRSymAddr, "add1") != 1 {
		t.Fatalf("function-typed binding did not emit one IRSymAddr(add1): %#v", fn.Instrs)
	}
}

func TestLowerCallableAliasEmitsResolvedSymAddrIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    let g: fn(Int) -> Int = f
    return 0
`, "main")

	if countInstr(fn.Instrs, ir.IRSymAddr, "add1") != 2 {
		t.Fatalf("function-typed alias did not lower to resolved IRSymAddr(add1) stores: %#v", fn.Instrs)
	}
}

func TestLowerCallableDirectNamedParamUsesKnownTargetIR(t *testing.T) {
	prog := lowerCallableProgram(t, `
func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(add1, 41)
`)
	apply := requireCallableFunc(t, prog, "apply")
	mainFn := requireCallableFunc(t, prog, "main")

	if countInstr(mainFn.Instrs, ir.IRSymAddr, "add1") != 1 {
		t.Fatalf("direct named callback argument did not lower to IRSymAddr(add1): %#v", mainFn.Instrs)
	}
	if countCall(apply.Instrs, "add1", 1, 1) != 1 {
		t.Fatalf("single-target callback body did not lower to direct IRCall(add1): %#v", apply.Instrs)
	}
	if countInstr(apply.Instrs, ir.IRSymAddr, "add1") != 0 {
		t.Fatalf("single-target callback body should not emit a dynamic branch IRSymAddr: %#v", apply.Instrs)
	}
}

func TestLowerCallableMultiTargetParamBranchesOnSymAddrIR(t *testing.T) {
	apply := lowerCallableFunc(t, `
func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let a: Int = apply(add1, 10)
    let b: Int = apply(add2, 20)
    return a + b
`, "apply")

	if countInstr(apply.Instrs, ir.IRSymAddr, "add1") != 1 || countInstr(apply.Instrs, ir.IRSymAddr, "add2") != 1 {
		t.Fatalf("multi-target callback body did not compare against both target symbols: %#v", apply.Instrs)
	}
	if countCallableKind(apply.Instrs, ir.IRCmpEqI32) < 2 || countCallableKind(apply.Instrs, ir.IRJmpIfZero) < 2 {
		t.Fatalf("multi-target callback body lacks symbol compare/branch sequence: %#v", apply.Instrs)
	}
	if countCall(apply.Instrs, "add1", 1, 1) != 1 || countCall(apply.Instrs, "add2", 1, 1) != 1 {
		t.Fatalf("multi-target callback body did not lower both direct target calls: %#v", apply.Instrs)
	}
}

func TestLowerCallableMultiTargetStringReturnSlotsIR(t *testing.T) {
	apply := lowerCallableFunc(t, `
func word1(x: Int) -> String:
    return "cat"

func word2(x: Int) -> String:
    return "zebra"

func apply(cb: fn(Int) -> String, x: Int) -> String:
    return cb(x)

func main() -> Int:
    let a: String = apply(word1, 0)
    let b: String = apply(word2, 0)
    return a.len + b.len
`, "apply")

	if apply.ReturnSlots != 2 {
		t.Fatalf("string-return callback apply ReturnSlots = %d, want 2", apply.ReturnSlots)
	}
	if countCall(apply.Instrs, "word1", 1, 2) != 1 || countCall(apply.Instrs, "word2", 1, 2) != 1 {
		t.Fatalf("string-return callback branches did not preserve two return slots: %#v", apply.Instrs)
	}
	if countStoresAfterCalls(apply.Instrs, map[string]bool{"word1": true, "word2": true}) < 4 {
		t.Fatalf("string-return callback branches did not store two result slots per target: %#v", apply.Instrs)
	}
}

func TestLowerCallableMultiTargetStructReturnSlotsIR(t *testing.T) {
	apply := lowerCallableFunc(t, `
struct Pair:
    x: Int
    y: Int

func pair1(x: Int) -> Pair:
    return Pair(x: x, y: 1)

func pair2(x: Int) -> Pair:
    return Pair(x: x, y: 2)

func apply(cb: fn(Int) -> Pair, x: Int) -> Pair:
    return cb(x)

func main() -> Int:
    let a: Pair = apply(pair1, 10)
    let b: Pair = apply(pair2, 20)
    return a.x + a.y + b.x + b.y
`, "apply")

	if apply.ReturnSlots != 2 {
		t.Fatalf("struct-return callback apply ReturnSlots = %d, want 2", apply.ReturnSlots)
	}
	if countCall(apply.Instrs, "pair1", 1, 2) != 1 || countCall(apply.Instrs, "pair2", 1, 2) != 1 {
		t.Fatalf("struct-return callback branches did not preserve two return slots: %#v", apply.Instrs)
	}
	if countStoresAfterCalls(apply.Instrs, map[string]bool{"pair1": true, "pair2": true}) < 4 {
		t.Fatalf("struct-return callback branches did not store two result slots per target: %#v", apply.Instrs)
	}
}

func TestLowerCallableUnknownTargetDiagnostic(t *testing.T) {
	checked := checkCallableProgram(t, `
func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return 0
`)
	var apply semantics.CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "apply" {
			apply = fn
			break
		}
	}
	if apply.Name == "" {
		t.Fatalf("apply function not found")
	}

	_, err := lowerCheckedFunc(
		apply,
		checked.Types,
		checked.FuncSigs,
		checked.GlobalsByModule[apply.Module],
		typedTaskStagedTarget{},
		map[string][]string{"cb": {"missing_callback_target"}},
	)
	if err == nil || !strings.Contains(err.Error(), "unknown callback target 'missing_callback_target'") {
		t.Fatalf("error = %v, want unknown callback target diagnostic", err)
	}
}

func lowerCallableFunc(t *testing.T, src string, name string) ir.IRFunc {
	t.Helper()
	return requireCallableFunc(t, lowerCallableProgram(t, src), name)
}

func lowerCallableProgram(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	checked := checkCallableProgram(t, src)
	prog, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	return prog
}

func checkCallableProgram(t *testing.T, src string) *semantics.CheckedProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return checked
}

func requireCallableFunc(t *testing.T, prog *ir.IRProgram, name string) ir.IRFunc {
	t.Helper()
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("function %q not found", name)
	return ir.IRFunc{}
}

func countInstr(instrs []ir.IRInstr, kind ir.IRInstrKind, name string) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind != kind {
			continue
		}
		if name != "" && instr.Name != name {
			continue
		}
		count++
	}
	return count
}

func countCallableKind(instrs []ir.IRInstr, kind ir.IRInstrKind) int {
	return countInstr(instrs, kind, "")
}

func countCall(instrs []ir.IRInstr, name string, argSlots int, retSlots int) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind == ir.IRCall && instr.Name == name && instr.ArgSlots == argSlots && instr.RetSlots == retSlots {
			count++
		}
	}
	return count
}

func countStoresAfterCalls(instrs []ir.IRInstr, names map[string]bool) int {
	count := 0
	for i, instr := range instrs {
		if instr.Kind != ir.IRCall || !names[instr.Name] {
			continue
		}
		for j := i + 1; j < len(instrs) && instrs[j].Kind == ir.IRStoreLocal; j++ {
			count++
		}
	}
	return count
}
