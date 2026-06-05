package opt

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestInlineSmallPurePassInlinesCallAndReportsDecision(t *testing.T) {
	prog := inlineAddProgram()

	report, err := NewManager().Run(prog, InlineSmallPurePass())
	if err != nil {
		t.Fatalf("Run InlineSmallPurePass: %v", err)
	}
	if len(report.Passes) != 1 {
		t.Fatalf("passes = %d, want 1", len(report.Passes))
	}
	row := report.Passes[0]
	if row.Name != "inline-small-pure" || row.ReportOutput != "inline-small-pure.opt.json" || !row.TranslationValidated {
		t.Fatalf("metadata row = %#v", row)
	}
	if !strings.Contains(row.BeforeDump, "call add args:2 rets:1") {
		t.Fatalf("before dump missing call:\n%s", row.BeforeDump)
	}
	mainAfter := dumpFuncAfter(t, row.AfterDump, "main")
	if strings.Contains(mainAfter, "call add") {
		t.Fatalf("main after dump still contains call:\n%s", mainAfter)
	}
	for _, want := range []string{"store_local local:1", "store_local local:0", "load_local local:0", "load_local local:1", "add_i32"} {
		if !strings.Contains(mainAfter, want) {
			t.Fatalf("main after dump missing %q:\n%s", want, mainAfter)
		}
	}
	decision := requireDecision(t, row.Decisions, "inlined", "main", "add")
	if decision.Reason != "small_pure" {
		t.Fatalf("inlined reason = %q, want small_pure", decision.Reason)
	}
}

func TestInlineSmallPurePassReportsNotInlinedReasons(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 1},
					{Kind: ir.IRCall, Name: "self", ArgSlots: 1, RetSlots: 1},
					{Kind: ir.IRCall, Name: "writer", ArgSlots: 0, RetSlots: 1},
					{Kind: ir.IRAddI32},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRCall, Name: "proofy", ArgSlots: 3, RetSlots: 1},
					{Kind: ir.IRAddI32},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "self",
				ParamSlots:  1,
				LocalSlots:  1,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadLocal, Local: 0},
					{Kind: ir.IRCall, Name: "self", ArgSlots: 1, RetSlots: 1},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "writer",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRStrLit, Str: []byte("x")},
					{Kind: ir.IRWrite},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "proofy",
				ParamSlots:  3,
				LocalSlots:  3,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadLocal, Local: 0},
					{Kind: ir.IRLoadLocal, Local: 1},
					{Kind: ir.IRLoadLocal, Local: 2},
					{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:test"},
					{Kind: ir.IRReturn},
				},
			},
		},
	}

	report, err := NewManager().Run(prog, InlineSmallPurePass())
	if err != nil {
		t.Fatalf("Run InlineSmallPurePass: %v", err)
	}
	row := report.Passes[0]
	if got := requireDecision(t, row.Decisions, "not_inlined", "self", "self").Reason; got != "recursive" {
		t.Fatalf("self recursive reason = %q, want recursive", got)
	}
	if got := requireDecision(t, row.Decisions, "not_inlined", "main", "self").Reason; got != "callee_contains_call" {
		t.Fatalf("main->self reason = %q, want callee_contains_call", got)
	}
	if got := requireDecision(t, row.Decisions, "not_inlined", "main", "writer").Reason; got != "unsupported_effect" {
		t.Fatalf("main->writer reason = %q, want unsupported_effect", got)
	}
	if got := requireDecision(t, row.Decisions, "not_inlined", "main", "proofy").Reason; got != "proof_sensitive" {
		t.Fatalf("main->proofy reason = %q, want proof_sensitive", got)
	}
}

func TestInlineSmallPurePassDifferentialExecution(t *testing.T) {
	before := inlineAddProgram()
	after := cloneProgram(before)
	report, err := NewManager().Run(after, InlineSmallPurePass())
	if err != nil {
		t.Fatalf("Run InlineSmallPurePass: %v", err)
	}
	if len(report.Passes[0].Decisions) == 0 {
		t.Fatalf("missing inline decisions")
	}

	beforeExit := runOptLinuxX64(t, before.Funcs, "before-inline-small-pure")
	afterExit := runOptLinuxX64(t, after.Funcs, "after-inline-small-pure")
	if beforeExit != afterExit {
		t.Fatalf("exit mismatch before=%d after=%d", beforeExit, afterExit)
	}
	if afterExit != 42 {
		t.Fatalf("optimized exit = %d, want 42", afterExit)
	}
}

func inlineAddProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 20},
					{Kind: ir.IRConstI32, Imm: 22},
					{Kind: ir.IRCall, Name: "add", ArgSlots: 2, RetSlots: 1},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "add",
				ParamSlots:  2,
				LocalSlots:  2,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadLocal, Local: 0},
					{Kind: ir.IRLoadLocal, Local: 1},
					{Kind: ir.IRAddI32},
					{Kind: ir.IRReturn},
				},
			},
		},
	}
}

func requireDecision(t *testing.T, decisions []PassDecision, action string, caller string, callee string) PassDecision {
	t.Helper()
	for _, decision := range decisions {
		if decision.Action == action && decision.Caller == caller && decision.Callee == callee {
			return decision
		}
	}
	t.Fatalf("missing decision action=%s caller=%s callee=%s in %#v", action, caller, callee, decisions)
	return PassDecision{}
}

func dumpFuncAfter(t *testing.T, dump string, name string) string {
	t.Helper()
	marker := "func " + name + " "
	start := strings.Index(dump, marker)
	if start < 0 {
		t.Fatalf("dump missing function %q:\n%s", name, dump)
	}
	rest := dump[start:]
	next := strings.Index(rest[len(marker):], "\nfunc ")
	if next < 0 {
		return rest
	}
	return rest[:len(marker)+next]
}
