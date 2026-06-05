package opt

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLoopCanonicalizationPassHoistsStableLenAndCanonicalizesLeMinusOne(t *testing.T) {
	prog := loopCanonicalizationProgram(ir.IRCmpLeI32, true)

	report, err := NewManager().Run(prog, LoopCanonicalizationPass())
	if err != nil {
		t.Fatalf("Run LoopCanonicalizationPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 5 {
		t.Fatalf("LocalSlots = %d, want new hoisted len local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{"store_local local:4", "load_local local:4", "cmp_lt_i32", "proof:proof:while:i:xs:1:1"} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	for _, forbidden := range []string{"sub_i32", "cmp_le_i32"} {
		if strings.Contains(after, forbidden) {
			t.Fatalf("after dump still contains %q:\n%s", forbidden, after)
		}
	}
	row := report.Passes[0]
	if row.Name != "loop-canonicalization" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "canonicalized", "stable_len_le_minus_one_to_lt") {
		t.Fatalf("decisions missing canonicalization evidence: %#v", row.Decisions)
	}
}

func TestLoopCanonicalizationPassHoistsStableLenForLessThanLoop(t *testing.T) {
	prog := loopCanonicalizationProgram(ir.IRCmpLtI32, true)

	report, err := NewManager().Run(prog, LoopCanonicalizationPass())
	if err != nil {
		t.Fatalf("Run LoopCanonicalizationPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if !strings.Contains(after, "store_local local:4") || !strings.Contains(after, "load_local local:4") {
		t.Fatalf("after dump missing hoisted len local:\n%s", after)
	}
	if strings.Contains(after, "cmp_le_i32") || strings.Contains(after, "sub_i32") {
		t.Fatalf("less-than loop unexpectedly contains <= canonicalization remnants:\n%s", after)
	}
	if !hasDecision(report.Passes[0].Decisions, "hoisted", "stable_len_load") {
		t.Fatalf("decisions missing hoist evidence: %#v", report.Passes[0].Decisions)
	}
}

func TestLoopCanonicalizationPassRejectsUnsafeLoopShapes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ir.IRProgram)
		reason string
	}{
		{
			name: "missing proof",
			mutate: func(p *ir.IRProgram) {
				for i := range p.Funcs[0].Instrs {
					if p.Funcs[0].Instrs[i].Kind == ir.IRIndexLoadI32Unchecked {
						p.Funcs[0].Instrs[i].Kind = ir.IRIndexLoadI32
						p.Funcs[0].Instrs[i].ProofID = ""
					}
				}
			},
			reason: "missing_while_bounds_proof",
		},
		{
			name: "call in loop",
			mutate: func(p *ir.IRProgram) {
				insertAt := 9
				p.Funcs[0].Instrs = append(p.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{{Kind: ir.IRCall, Name: "touch", ArgSlots: 0, RetSlots: 0}}, p.Funcs[0].Instrs[insertAt:]...)...)
			},
			reason: "loop_has_unknown_mutation",
		},
		{
			name: "len store in loop",
			mutate: func(p *ir.IRProgram) {
				insertAt := 9
				p.Funcs[0].Instrs = append(p.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{{Kind: ir.IRConstI32, Imm: 9}, {Kind: ir.IRStoreLocal, Local: 2}}, p.Funcs[0].Instrs[insertAt:]...)...)
			},
			reason: "loop_stores_len_local",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := loopCanonicalizationProgram(ir.IRCmpLtI32, true)
			tc.mutate(prog)
			report, err := NewManager().Run(prog, LoopCanonicalizationPass())
			if err != nil {
				t.Fatalf("Run LoopCanonicalizationPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if strings.Contains(after, "store_local local:4") {
				t.Fatalf("unsafe loop was hoisted:\n%s", after)
			}
			if !hasDecision(report.Passes[0].Decisions, "not_hoisted", tc.reason) {
				t.Fatalf("decisions missing %q: %#v", tc.reason, report.Passes[0].Decisions)
			}
		})
	}
}

func loopCanonicalizationProgram(cmp ir.IRInstrKind, withProof bool) *ir.IRProgram {
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 0},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 3},
		{Kind: ir.IRLabel, Label: 1},
		{Kind: ir.IRLoadLocal, Local: 0},
		{Kind: ir.IRLoadLocal, Local: 2},
	}
	if cmp == ir.IRCmpLeI32 {
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRConstI32, Imm: 1}, ir.IRInstr{Kind: ir.IRSubI32})
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: cmp},
		ir.IRInstr{Kind: ir.IRJmpIfZero, Label: 2},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 2},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRIndexLoadI32Unchecked, ProofID: proofID(withProof)},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 3},
		ir.IRInstr{Kind: ir.IRAddI32},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 3},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRAddI32},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRJmp, Label: 1},
		ir.IRInstr{Kind: ir.IRLabel, Label: 2},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 3},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  3,
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs:      instrs,
		}},
	}
}

func proofID(enabled bool) string {
	if !enabled {
		return ""
	}
	return "proof:while:i:xs:1:1"
}

func hasDecision(decisions []PassDecision, action string, reason string) bool {
	for _, decision := range decisions {
		if decision.Action == action && decision.Reason == reason {
			return true
		}
	}
	return false
}
