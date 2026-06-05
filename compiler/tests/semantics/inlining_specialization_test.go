package compiler_test

import (
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/opt"
)

func TestP17KnownEnumPayloadMatchFoldsAfterSCCP(t *testing.T) {
	src := []byte(`
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	before := findIRFunc(t, irProg.Funcs, "main")
	beforeCmps := countIRKind(before, ir.IRCmpEqI32)
	if beforeCmps < 2 {
		t.Fatalf("pre-optimization enum match lacks discriminator comparisons: %#v", before.Instrs)
	}

	report, err := opt.NewManager().Run(irProg, opt.SCCPPass())
	if err != nil {
		t.Fatalf("SCCPPass: %v", err)
	}
	if len(report.Passes) != 1 || !report.Passes[0].TranslationValidated {
		t.Fatalf("SCCP report lacks translation validation evidence: %#v", report.Passes)
	}
	if !hasPassDecision(report.Passes[0].Decisions, "folded_const_expr_nonzero_fallthrough", "main", "constant_expression_condition") {
		t.Fatalf("missing known enum case fold decision: %#v", report.Passes[0].Decisions)
	}
	after := findIRFunc(t, irProg.Funcs, "main")
	if afterCmps := countIRKind(after, ir.IRCmpEqI32); afterCmps >= beforeCmps {
		t.Fatalf("known enum case discriminator compare was not folded: before=%d after=%d instrs=%#v", beforeCmps, afterCmps, after.Instrs)
	}
}

func TestP17ProvenSomeOptionalMatchFoldsAfterSCCP(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let value: Int? = 42
    let score: Int = match value:
    case some(x):
        x
    case none:
        0
    return score
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	before := findIRFunc(t, irProg.Funcs, "main")
	beforeBranches := countIRKind(before, ir.IRJmpIfZero)
	if beforeBranches < 2 {
		t.Fatalf("pre-optimization optional match lacks expected branches: %#v", before.Instrs)
	}

	report, err := opt.NewManager().Run(irProg, opt.SCCPPass())
	if err != nil {
		t.Fatalf("SCCPPass: %v", err)
	}
	if len(report.Passes) != 1 || !report.Passes[0].TranslationValidated {
		t.Fatalf("SCCP report lacks translation validation evidence: %#v", report.Passes)
	}
	if !hasPassDecision(report.Passes[0].Decisions, "tracked_known_local_store", "main", "constant_stack_store") {
		t.Fatalf("missing optional tag store tracking decision: %#v", report.Passes[0].Decisions)
	}
	if !hasPassDecision(report.Passes[0].Decisions, "folded_known_local_nonzero_fallthrough", "main", "constant_local_condition") {
		t.Fatalf("missing proven-some optional branch fold decision: %#v", report.Passes[0].Decisions)
	}
	after := findIRFunc(t, irProg.Funcs, "main")
	if afterBranches := countIRKind(after, ir.IRJmpIfZero); afterBranches >= beforeBranches {
		t.Fatalf("proven-some optional branch was not folded: before=%d after=%d instrs=%#v", beforeBranches, afterBranches, after.Instrs)
	}
}

func TestP17StaticExtensionCallInlinesAfterSmallPure(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int
    y: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y

func main() -> Int:
    let v: Vec2 = Vec2(x: 40, y: 2)
    return Vec2.sum(v)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	before := findIRFunc(t, irProg.Funcs, "main")
	if calls := countIRCallName(before, "Vec2.sum"); calls != 1 {
		t.Fatalf("pre-optimization main calls Vec2.sum %d times, want 1: %#v", calls, before.Instrs)
	}

	report, err := opt.NewManager().Run(irProg, opt.InlineSmallPurePass())
	if err != nil {
		t.Fatalf("InlineSmallPurePass: %v", err)
	}
	if len(report.Passes) != 1 || !report.Passes[0].TranslationValidated {
		t.Fatalf("inline report lacks translation validation evidence: %#v", report.Passes)
	}
	if !hasPassDecisionCallee(report.Passes[0].Decisions, "inlined", "main", "Vec2.sum", "small_pure") {
		t.Fatalf("missing static extension inline decision: %#v", report.Passes[0].Decisions)
	}
	after := findIRFunc(t, irProg.Funcs, "main")
	if calls := countIRCallName(after, "Vec2.sum"); calls != 0 {
		t.Fatalf("static extension call was not inlined: remaining=%d instrs=%#v", calls, after.Instrs)
	}
	if adds := countIRKind(after, ir.IRAddI32); adds == 0 {
		t.Fatalf("inlined extension method body missing add_i32: %#v", after.Instrs)
	}
}

func TestP17StaticProtocolConformanceCallInlinesAfterSmallPure(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int
    y: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x + self.y

impl Vec2: Renderable

func main() -> Int:
    let v: Vec2 = Vec2(x: 40, y: 2)
    return Vec2.draw(v)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Impls) != 1 {
		t.Fatalf("pre-check program has %d impls, want 1", len(prog.Impls))
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.FuncSigs["Vec2.draw"]; !ok {
		t.Fatalf("missing statically checked conformance method signature: %#v", checked.FuncSigs)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	before := findIRFunc(t, irProg.Funcs, "main")
	if calls := countIRCallName(before, "Vec2.draw"); calls != 1 {
		t.Fatalf("pre-optimization main calls Vec2.draw %d times, want 1: %#v", calls, before.Instrs)
	}

	report, err := opt.NewManager().Run(irProg, opt.InlineSmallPurePass())
	if err != nil {
		t.Fatalf("InlineSmallPurePass: %v", err)
	}
	if len(report.Passes) != 1 || !report.Passes[0].TranslationValidated {
		t.Fatalf("inline report lacks translation validation evidence: %#v", report.Passes)
	}
	if !hasPassDecisionCallee(report.Passes[0].Decisions, "inlined", "main", "Vec2.draw", "small_pure") {
		t.Fatalf("missing static protocol conformance inline decision: %#v", report.Passes[0].Decisions)
	}
	after := findIRFunc(t, irProg.Funcs, "main")
	if calls := countIRCallName(after, "Vec2.draw"); calls != 0 {
		t.Fatalf("static protocol conformance call was not inlined: remaining=%d instrs=%#v", calls, after.Instrs)
	}
	if adds := countIRKind(after, ir.IRAddI32); adds == 0 {
		t.Fatalf("inlined conformance method body missing add_i32: %#v", after.Instrs)
	}
}

func countIRKind(fn compiler.IRFunc, kind ir.IRInstrKind) int {
	count := 0
	for _, instr := range fn.Instrs {
		if instr.Kind == kind {
			count++
		}
	}
	return count
}

func countIRCallName(fn compiler.IRFunc, name string) int {
	count := 0
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			count++
		}
	}
	return count
}

func hasPassDecision(decisions []opt.PassDecision, action string, caller string, reason string) bool {
	for _, decision := range decisions {
		if decision.Action == action && decision.Caller == caller && decision.Reason == reason {
			return true
		}
	}
	return false
}

func hasPassDecisionCallee(decisions []opt.PassDecision, action string, caller string, callee string, reason string) bool {
	for _, decision := range decisions {
		if decision.Action == action && decision.Caller == caller && decision.Callee == callee && decision.Reason == reason {
			return true
		}
	}
	return false
}
