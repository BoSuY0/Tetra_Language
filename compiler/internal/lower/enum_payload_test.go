package lower

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerEnumPayloadConstructorLayoutIR(t *testing.T) {
	fn := lowerMainForEnumPayloadTest(t, `
enum Pair:
    case both(Int, String)
    case empty

func main() -> Int:
    let pair: Pair = Pair.both(7, "xy")
    return 0
`)

	if !hasEnumConstructorSequence(fn.Instrs, []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRConstI32, Imm: 7},
		{Kind: ir.IRStrLit},
	}, 4) {
		t.Fatalf("constructor IR does not preserve tag + payload slot order: %#v", fn.Instrs)
	}
}

func TestLowerEnumPayloadConstructorZeroPadsWideNoPayloadCaseIR(t *testing.T) {
	fn := lowerMainForEnumPayloadTest(t, `
enum Wide:
    case data(Int, String)
    case empty

func main() -> Int:
    let wide: Wide = Wide.empty
    return 0
`)

	if !hasEnumConstructorSequence(fn.Instrs, []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRConstI32, Imm: 0},
	}, 4) {
		t.Fatalf("no-payload constructor IR does not preserve tag + zero padding: %#v", fn.Instrs)
	}
}

func TestLowerEnumPayloadSlotOrderInMatchIR(t *testing.T) {
	fn := lowerMainForEnumPayloadTest(t, `
enum Pair:
    case both(Int, String)

func main() -> Int:
    let pair: Pair = Pair.both(5, "xy")
    match pair:
    case Pair.both(code, text):
        return code
`)

	firstBindingLoad, secondBindingLoad := findFirstTwoPayloadBindingLoads(t, fn.Instrs)
	if secondBindingLoad.Local != firstBindingLoad.Local+1 {
		t.Fatalf("payload binding loads not contiguous/in declaration order: first=%#v second=%#v", firstBindingLoad, secondBindingLoad)
	}
}

func TestLowerMatchExpressionEnumPayloadIR(t *testing.T) {
	fn := lowerMainForEnumPayloadTest(t, `
enum Result:
    case ok(Int)
    case err(Int)

func main() -> Int:
    let result: Result = Result.ok(42)
    let score: Int = match result:
    case Result.ok(value):
        value
    case Result.err(code):
        code
    return score
`)

	if !hasInstructionPair(fn.Instrs, ir.IRCmpEqI32, ir.IRJmpIfZero) {
		t.Fatalf("match expression IR lacks compare/branch discriminator checks: %#v", fn.Instrs)
	}
	if countKind(fn.Instrs, ir.IRLabel) < 3 {
		t.Fatalf("match expression IR label count too low: %#v", fn.Instrs)
	}
}

func TestLowerIfLetEnumPayloadPatternIR(t *testing.T) {
	fn := lowerMainForEnumPayloadTest(t, `
enum Result:
    case ok(Int, String)
    case err(Int)

func main() -> Int:
    let result: Result = Result.ok(40, "xy")
    if let Result.ok(code, text) = result:
        return code + text.len
    else:
        return 0
`)

	if !hasInstructionPair(fn.Instrs, ir.IRCmpEqI32, ir.IRJmpIfZero) {
		t.Fatalf("if-let enum pattern IR lacks compare/branch discriminator checks: %#v", fn.Instrs)
	}
	firstBindingLoad, secondBindingLoad := findFirstTwoPayloadBindingLoads(t, fn.Instrs)
	if secondBindingLoad.Local != firstBindingLoad.Local+1 {
		t.Fatalf("if-let payload binding loads not contiguous/in declaration order: first=%#v second=%#v", firstBindingLoad, secondBindingLoad)
	}
}

func lowerMainForEnumPayloadTest(t *testing.T, src string) ir.IRFunc {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
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
	for _, fn := range irProg.Funcs {
		if fn.Name == "main" {
			return fn
		}
	}
	t.Fatalf("main function not found")
	return ir.IRFunc{}
}

func hasEnumConstructorSequence(instrs []ir.IRInstr, values []ir.IRInstr, storeSlots int) bool {
	for i := 0; i+len(values)+storeSlots <= len(instrs); i++ {
		matched := true
		for j, want := range values {
			got := instrs[i+j]
			if got.Kind != want.Kind {
				matched = false
				break
			}
			if want.Kind == ir.IRConstI32 && got.Imm != want.Imm {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		for j := 0; j < storeSlots; j++ {
			if instrs[i+len(values)+j].Kind != ir.IRStoreLocal {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func findFirstTwoPayloadBindingLoads(t *testing.T, instrs []ir.IRInstr) (ir.IRInstr, ir.IRInstr) {
	t.Helper()
	for i := 0; i+3 < len(instrs); i++ {
		if instrs[i].Kind == ir.IRLoadLocal && instrs[i+1].Kind == ir.IRStoreLocal && instrs[i+2].Kind == ir.IRLoadLocal && instrs[i+3].Kind == ir.IRLoadLocal {
			return instrs[i], instrs[i+2]
		}
	}
	t.Fatalf("payload binding load sequence not found: %#v", instrs)
	return ir.IRInstr{}, ir.IRInstr{}
}

func hasInstructionPair(instrs []ir.IRInstr, first, second ir.IRInstrKind) bool {
	for i := 0; i+1 < len(instrs); i++ {
		if instrs[i].Kind == first && instrs[i+1].Kind == second {
			return true
		}
	}
	return false
}

func countKind(instrs []ir.IRInstr, kind ir.IRInstrKind) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind == kind {
			count++
		}
	}
	return count
}
