package opt

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLICMPureInvariantPassHoistsPureComparisonInsideProofLoop(t *testing.T) {
	prog := licmInvariantProgram()

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 6 {
		t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:5",
		"load_local local:5",
		"cmp_gt_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	if strings.Count(after, "cmp_gt_i32") != 1 {
		t.Fatalf("after dump should keep only the hoisted invariant comparison:\n%s", after)
	}
	row := report.Passes[0]
	if row.Name != "licm-pure-invariant" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "hoisted", "pure_invariant_comparison") {
		t.Fatalf("decisions missing LICM evidence: %#v", row.Decisions)
	}
}

func TestLICMPureInvariantPassHoistsPureArithmeticInsideProofLoop(t *testing.T) {
	prog := licmInvariantProgram()
	prog.Funcs[0].Instrs[15].Imm = 7
	prog.Funcs[0].Instrs[16].Kind = ir.IRAddI32

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 6 {
		t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:5",
		"load_local local:5",
		"add_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	if strings.Count(after, "const_i32 7") != 1 {
		t.Fatalf("after dump should keep only the hoisted arithmetic constant:\n%s", after)
	}
	if !hoistedBeforeLoopLabel(after, "store_local local:5") {
		t.Fatalf("arithmetic invariant was not hoisted before the loop label:\n%s", after)
	}
	row := report.Passes[0]
	if row.Name != "licm-pure-invariant" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "hoisted", "pure_invariant_arithmetic") {
		t.Fatalf("decisions missing arithmetic LICM evidence: %#v", row.Decisions)
	}
}

func TestLICMPureInvariantPassHoistsPureSubArithmeticInsideProofLoop(t *testing.T) {
	prog := licmInvariantProgram()
	prog.Funcs[0].Instrs[15].Imm = 7
	prog.Funcs[0].Instrs[16].Kind = ir.IRSubI32

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 6 {
		t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:5",
		"load_local local:5",
		"sub_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	if strings.Count(after, "sub_i32") != 1 {
		t.Fatalf("after dump should keep only the hoisted subtraction:\n%s", after)
	}
	if !hoistedBeforeLoopLabel(after, "store_local local:5") {
		t.Fatalf("subtraction invariant was not hoisted before the loop label:\n%s", after)
	}
	row := report.Passes[0]
	if row.Name != "licm-pure-invariant" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "hoisted", "pure_invariant_arithmetic") {
		t.Fatalf("decisions missing subtraction LICM evidence: %#v", row.Decisions)
	}
}

func TestLICMPureInvariantPassHoistsSafeDivArithmeticInsideProofLoop(t *testing.T) {
	prog := licmInvariantProgram()
	prog.Funcs[0].Instrs[15].Imm = 3
	prog.Funcs[0].Instrs[16].Kind = ir.IRDivI32

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 6 {
		t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:5",
		"load_local local:5",
		"div_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	if strings.Count(after, "div_i32") != 1 {
		t.Fatalf("after dump should keep only the hoisted division:\n%s", after)
	}
	if !hoistedBeforeLoopLabel(after, "store_local local:5") {
		t.Fatalf("division invariant was not hoisted before the loop label:\n%s", after)
	}
	row := report.Passes[0]
	if row.Name != "licm-pure-invariant" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "hoisted", "pure_invariant_safe_division") {
		t.Fatalf("decisions missing safe-division LICM evidence: %#v", row.Decisions)
	}
}

func TestLICMPureInvariantPassHoistsSafeModArithmeticInsideProofLoop(t *testing.T) {
	prog := licmInvariantProgram()
	prog.Funcs[0].Instrs[15].Imm = 3
	prog.Funcs[0].Instrs[16].Kind = ir.IRModI32

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	if prog.Funcs[0].LocalSlots != 6 {
		t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
	}
	after := report.Passes[0].AfterDump
	for _, want := range []string{
		"store_local local:5",
		"load_local local:5",
		"mod_i32",
		"proof:proof:while:i:xs:1:1",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("after dump missing %q:\n%s", want, after)
		}
	}
	if strings.Count(after, "mod_i32") != 1 {
		t.Fatalf("after dump should keep only the hoisted modulo:\n%s", after)
	}
	if !hoistedBeforeLoopLabel(after, "store_local local:5") {
		t.Fatalf("modulo invariant was not hoisted before the loop label:\n%s", after)
	}
	row := report.Passes[0]
	if row.Name != "licm-pure-invariant" || !row.TranslationValidated {
		t.Fatalf("pass row = %#v", row)
	}
	if !hasDecision(row.Decisions, "hoisted", "pure_invariant_safe_modulo") {
		t.Fatalf("decisions missing safe-modulo LICM evidence: %#v", row.Decisions)
	}
}

func TestLICMPureInvariantPassHoistsSafeKnownLocalDivModInsideProofLoop(t *testing.T) {
	cases := []struct {
		name   string
		kind   ir.IRInstrKind
		op     string
		reason string
	}{
		{name: "division", kind: ir.IRDivI32, op: "div_i32", reason: "pure_invariant_safe_known_local_division"},
		{name: "modulo", kind: ir.IRModI32, op: "mod_i32", reason: "pure_invariant_safe_known_local_modulo"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := licmKnownLocalDivModInvariantProgram(tc.kind, 3)

			report, err := NewManager().Run(prog, LICMPureInvariantPass())
			if err != nil {
				t.Fatalf("Run LICMPureInvariantPass: %v", err)
			}
			if prog.Funcs[0].LocalSlots != 7 {
				t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
			}
			after := report.Passes[0].AfterDump
			for _, want := range []string{
				"store_local local:6",
				"load_local local:6",
				"load_local local:5",
				tc.op,
				"proof:proof:while:i:xs:1:1",
			} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			if strings.Count(after, tc.op) != 1 {
				t.Fatalf("after dump should keep only the hoisted known-local %s:\n%s", tc.op, after)
			}
			if !hoistedBeforeLoopLabel(after, "store_local local:6") {
				t.Fatalf("known-local %s invariant was not hoisted before the loop label:\n%s", tc.op, after)
			}
			if !hasDecision(report.Passes[0].Decisions, "hoisted", tc.reason) {
				t.Fatalf("decisions missing safe known-local %s LICM evidence: %#v", tc.op, report.Passes[0].Decisions)
			}
		})
	}
}

func TestLICMPureInvariantPassHoistsKnownLocalArithmeticInsideProofLoop(t *testing.T) {
	cases := []struct {
		name   string
		kind   ir.IRInstrKind
		op     string
		reason string
	}{
		{name: "addition", kind: ir.IRAddI32, op: "add_i32", reason: "pure_invariant_known_local_arithmetic"},
		{name: "subtraction", kind: ir.IRSubI32, op: "sub_i32", reason: "pure_invariant_known_local_arithmetic"},
		{name: "multiplication", kind: ir.IRMulI32, op: "mul_i32", reason: "pure_invariant_known_local_arithmetic"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := licmKnownLocalArithmeticInvariantProgram(tc.kind, 7)

			report, err := NewManager().Run(prog, LICMPureInvariantPass())
			if err != nil {
				t.Fatalf("Run LICMPureInvariantPass: %v", err)
			}
			if prog.Funcs[0].LocalSlots != 7 {
				t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
			}
			after := report.Passes[0].AfterDump
			for _, want := range []string{
				"store_local local:6",
				"load_local local:6",
				"load_local local:5",
				tc.op,
				"proof:proof:while:i:xs:1:1",
			} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			if strings.Count(after, "load_local local:5") != 1 {
				t.Fatalf("after dump should load the known-local arithmetic operand only in the hoisted expression:\n%s", after)
			}
			if !hoistedBeforeLoopLabel(after, "store_local local:6") {
				t.Fatalf("known-local %s invariant was not hoisted before the loop label:\n%s", tc.op, after)
			}
			if !hasDecision(report.Passes[0].Decisions, "hoisted", tc.reason) {
				t.Fatalf("decisions missing known-local %s LICM evidence: %#v", tc.op, report.Passes[0].Decisions)
			}
		})
	}
}

func TestLICMPureInvariantPassRejectsKnownLocalArithmeticWhenOperandMutatesInLoop(t *testing.T) {
	prog := licmKnownLocalArithmeticInvariantProgram(ir.IRMulI32, 7)
	insertAt := 16
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 5},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[insertAt:]...)...)

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if strings.Contains(after, "store_local local:6") {
		t.Fatalf("mutating known-local arithmetic operand invariant expression was hoisted:\n%s", after)
	}
	if !hasDecision(report.Passes[0].Decisions, "not_hoisted", "loop_stores_invariant_operand") {
		t.Fatalf("decisions missing known-local arithmetic operand mutation rejection: %#v", report.Passes[0].Decisions)
	}
}

func TestLICMPureInvariantPassHoistsKnownLocalLeftArithmeticInsideProofLoop(t *testing.T) {
	cases := []struct {
		name string
		kind ir.IRInstrKind
		op   string
	}{
		{name: "addition", kind: ir.IRAddI32, op: "add_i32"},
		{name: "subtraction", kind: ir.IRSubI32, op: "sub_i32"},
		{name: "multiplication", kind: ir.IRMulI32, op: "mul_i32"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := licmKnownLocalLeftArithmeticInvariantProgram(tc.kind, 7)

			report, err := NewManager().Run(prog, LICMPureInvariantPass())
			if err != nil {
				t.Fatalf("Run LICMPureInvariantPass: %v", err)
			}
			if prog.Funcs[0].LocalSlots != 7 {
				t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
			}
			after := report.Passes[0].AfterDump
			for _, want := range []string{
				"store_local local:6",
				"load_local local:6",
				"load_local local:5",
				tc.op,
				"proof:proof:while:i:xs:1:1",
			} {
				if !strings.Contains(after, want) {
					t.Fatalf("after dump missing %q:\n%s", want, after)
				}
			}
			if strings.Count(after, "load_local local:5") != 1 {
				t.Fatalf("after dump should load the known-local left operand only in the hoisted expression:\n%s", after)
			}
			if !hoistedBeforeLoopLabel(after, "store_local local:6") {
				t.Fatalf("known-local left %s invariant was not hoisted before the loop label:\n%s", tc.op, after)
			}
			if !hasDecision(report.Passes[0].Decisions, "hoisted", "pure_invariant_known_local_arithmetic") {
				t.Fatalf("decisions missing known-local left %s LICM evidence: %#v", tc.op, report.Passes[0].Decisions)
			}
		})
	}
}

func TestLICMPureInvariantPassRejectsKnownLocalLeftArithmeticWhenOperandMutatesInLoop(t *testing.T) {
	prog := licmKnownLocalLeftArithmeticInvariantProgram(ir.IRSubI32, 7)
	insertAt := 16
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 5},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[insertAt:]...)...)

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if strings.Contains(after, "store_local local:6") {
		t.Fatalf("mutating known-local left arithmetic operand invariant expression was hoisted:\n%s", after)
	}
	if !hasDecision(report.Passes[0].Decisions, "not_hoisted", "loop_stores_invariant_operand") {
		t.Fatalf("decisions missing known-local left arithmetic operand mutation rejection: %#v", report.Passes[0].Decisions)
	}
}

func TestLICMPureInvariantPassHoistsKnownLocalComparisonInsideProofLoop(t *testing.T) {
	ops := []struct {
		name string
		kind ir.IRInstrKind
		op   string
	}{
		{name: "eq", kind: ir.IRCmpEqI32, op: "cmp_eq_i32"},
		{name: "lt", kind: ir.IRCmpLtI32, op: "cmp_lt_i32"},
		{name: "gt", kind: ir.IRCmpGtI32, op: "cmp_gt_i32"},
		{name: "ge", kind: ir.IRCmpGeI32, op: "cmp_ge_i32"},
		{name: "le", kind: ir.IRCmpLeI32, op: "cmp_le_i32"},
		{name: "ne", kind: ir.IRCmpNeI32, op: "cmp_ne_i32"},
	}
	positions := []struct {
		name        string
		knownOnLeft bool
	}{
		{name: "known-left", knownOnLeft: true},
		{name: "known-right", knownOnLeft: false},
	}

	for _, pos := range positions {
		for _, op := range ops {
			t.Run(pos.name+"-"+op.name, func(t *testing.T) {
				prog := licmKnownLocalComparisonInvariantProgram(op.kind, 7, pos.knownOnLeft)

				report, err := NewManager().Run(prog, LICMPureInvariantPass())
				if err != nil {
					t.Fatalf("Run LICMPureInvariantPass: %v", err)
				}
				if prog.Funcs[0].LocalSlots != 7 {
					t.Fatalf("LocalSlots = %d, want new hoisted invariant local", prog.Funcs[0].LocalSlots)
				}
				after := report.Passes[0].AfterDump
				for _, want := range []string{
					"store_local local:6",
					"load_local local:6",
					"load_local local:5",
					op.op,
					"proof:proof:while:i:xs:1:1",
				} {
					if !strings.Contains(after, want) {
						t.Fatalf("after dump missing %q:\n%s", want, after)
					}
				}
				if strings.Count(after, "load_local local:5") != 1 {
					t.Fatalf("after dump should load the known-local comparison operand only in the hoisted expression:\n%s", after)
				}
				if !hoistedBeforeLoopLabel(after, "store_local local:6") {
					t.Fatalf("known-local %s comparison invariant was not hoisted before the loop label:\n%s", op.op, after)
				}
				if !hasDecision(report.Passes[0].Decisions, "hoisted", "pure_invariant_known_local_comparison") {
					t.Fatalf("decisions missing known-local comparison LICM evidence: %#v", report.Passes[0].Decisions)
				}
			})
		}
	}
}

func TestLICMPureInvariantPassRejectsKnownLocalComparisonWhenOperandMutatesInLoop(t *testing.T) {
	prog := licmKnownLocalComparisonInvariantProgram(ir.IRCmpLtI32, 7, false)
	insertAt := 16
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 5},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[insertAt:]...)...)

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if strings.Contains(after, "store_local local:6") {
		t.Fatalf("mutating known-local comparison operand invariant expression was hoisted:\n%s", after)
	}
	if !hasDecision(report.Passes[0].Decisions, "not_hoisted", "loop_stores_invariant_operand") {
		t.Fatalf("decisions missing known-local comparison operand mutation rejection: %#v", report.Passes[0].Decisions)
	}
}

func TestLICMPureInvariantPassRejectsUnsafeKnownLocalDivModInsideProofLoop(t *testing.T) {
	cases := []struct {
		name   string
		kind   ir.IRInstrKind
		op     string
		denom  int32
		reason string
	}{
		{name: "division by zero", kind: ir.IRDivI32, op: "div_i32", denom: 0, reason: "unsafe_known_local_division_denominator"},
		{name: "division by minus one", kind: ir.IRDivI32, op: "div_i32", denom: -1, reason: "unsafe_known_local_division_denominator"},
		{name: "modulo by zero", kind: ir.IRModI32, op: "mod_i32", denom: 0, reason: "unsafe_known_local_modulo_denominator"},
		{name: "modulo by minus one", kind: ir.IRModI32, op: "mod_i32", denom: -1, reason: "unsafe_known_local_modulo_denominator"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := licmKnownLocalDivModInvariantProgram(tc.kind, tc.denom)

			report, err := NewManager().Run(prog, LICMPureInvariantPass())
			if err != nil {
				t.Fatalf("Run LICMPureInvariantPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if strings.Contains(after, "store_local local:6") {
				t.Fatalf("unsafe known-local %s invariant expression was hoisted:\n%s", tc.op, after)
			}
			if strings.Count(after, tc.op) != 1 {
				t.Fatalf("unsafe known-local %s expression should remain in loop:\n%s", tc.op, after)
			}
			if !hasDecision(report.Passes[0].Decisions, "not_hoisted", tc.reason) {
				t.Fatalf("decisions missing %q: %#v", tc.reason, report.Passes[0].Decisions)
			}
		})
	}
}

func TestLICMPureInvariantPassRejectsSafeKnownLocalDivModWhenDenominatorMutatesInLoop(t *testing.T) {
	prog := licmKnownLocalDivModInvariantProgram(ir.IRDivI32, 3)
	insertAt := 16
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 5},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[insertAt:]...)...)

	report, err := NewManager().Run(prog, LICMPureInvariantPass())
	if err != nil {
		t.Fatalf("Run LICMPureInvariantPass: %v", err)
	}
	after := report.Passes[0].AfterDump
	if strings.Contains(after, "store_local local:6") {
		t.Fatalf("mutating known-local denominator invariant expression was hoisted:\n%s", after)
	}
	if !hasDecision(report.Passes[0].Decisions, "not_hoisted", "loop_stores_invariant_operand") {
		t.Fatalf("decisions missing denominator mutation rejection: %#v", report.Passes[0].Decisions)
	}
}

func TestLICMPureInvariantPassRejectsVariantOrMutatedExpressions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ir.IRProgram)
		reason string
	}{
		{
			name: "loop index operand",
			mutate: func(p *ir.IRProgram) {
				// The candidate expression becomes `i > 0`, which is variant.
				p.Funcs[0].Instrs[14].Local = 0
			},
			reason: "variant_loop_index_operand",
		},
		{
			name: "stored invariant operand",
			mutate: func(p *ir.IRProgram) {
				insertAt := 14
				p.Funcs[0].Instrs = append(p.Funcs[0].Instrs[:insertAt], append([]ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 7},
					{Kind: ir.IRStoreLocal, Local: 4},
				}, p.Funcs[0].Instrs[insertAt:]...)...)
			},
			reason: "loop_stores_invariant_operand",
		},
		{
			name: "arithmetic loop index operand",
			mutate: func(p *ir.IRProgram) {
				p.Funcs[0].Instrs[14].Local = 0
				p.Funcs[0].Instrs[15].Imm = 7
				p.Funcs[0].Instrs[16].Kind = ir.IRAddI32
			},
			reason: "variant_loop_index_operand",
		},
		{
			name: "division by zero denominator",
			mutate: func(p *ir.IRProgram) {
				p.Funcs[0].Instrs[15].Imm = 0
				p.Funcs[0].Instrs[16].Kind = ir.IRDivI32
			},
			reason: "unsafe_division_denominator",
		},
		{
			name: "division by minus one denominator",
			mutate: func(p *ir.IRProgram) {
				p.Funcs[0].Instrs[15].Imm = -1
				p.Funcs[0].Instrs[16].Kind = ir.IRDivI32
			},
			reason: "unsafe_division_denominator",
		},
		{
			name: "modulo by zero denominator",
			mutate: func(p *ir.IRProgram) {
				p.Funcs[0].Instrs[15].Imm = 0
				p.Funcs[0].Instrs[16].Kind = ir.IRModI32
			},
			reason: "unsafe_modulo_denominator",
		},
		{
			name: "modulo by minus one denominator",
			mutate: func(p *ir.IRProgram) {
				p.Funcs[0].Instrs[15].Imm = -1
				p.Funcs[0].Instrs[16].Kind = ir.IRModI32
			},
			reason: "unsafe_modulo_denominator",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := licmInvariantProgram()
			tc.mutate(prog)
			report, err := NewManager().Run(prog, LICMPureInvariantPass())
			if err != nil {
				t.Fatalf("Run LICMPureInvariantPass: %v", err)
			}
			after := report.Passes[0].AfterDump
			if strings.Contains(after, "store_local local:5") {
				t.Fatalf("unsafe invariant expression was hoisted:\n%s", after)
			}
			if !hasDecision(report.Passes[0].Decisions, "not_hoisted", tc.reason) {
				t.Fatalf("decisions missing %q: %#v", tc.reason, report.Passes[0].Decisions)
			}
		})
	}
}

func hoistedBeforeLoopLabel(dump string, hoisted string) bool {
	hoistedIndex := strings.Index(dump, hoisted)
	labelIndex := strings.Index(dump, "label label:1")
	return hoistedIndex >= 0 && labelIndex >= 0 && hoistedIndex < labelIndex
}

func licmInvariantProgram() *ir.IRProgram {
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 0},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 3},
		{Kind: ir.IRLabel, Label: 1},
		{Kind: ir.IRLoadLocal, Local: 0},
		{Kind: ir.IRLoadLocal, Local: 2},
		{Kind: ir.IRCmpLtI32},
		{Kind: ir.IRJmpIfZero, Label: 2},
		{Kind: ir.IRLoadLocal, Local: 3},
		{Kind: ir.IRLoadLocal, Local: 1},
		{Kind: ir.IRLoadLocal, Local: 2},
		{Kind: ir.IRLoadLocal, Local: 0},
		{Kind: ir.IRIndexLoadI32Unchecked, ProofID: proofID(true)},
		{Kind: ir.IRLoadLocal, Local: 4},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRCmpGtI32},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 3},
		{Kind: ir.IRLoadLocal, Local: 0},
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 0},
		{Kind: ir.IRJmp, Label: 1},
		{Kind: ir.IRLabel, Label: 2},
		{Kind: ir.IRLoadLocal, Local: 3},
		{Kind: ir.IRReturn},
	}
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  5,
			LocalSlots:  5,
			ReturnSlots: 1,
			Instrs:      instrs,
		}},
	}
}

func licmKnownLocalDivModInvariantProgram(kind ir.IRInstrKind, denominator int32) *ir.IRProgram {
	prog := licmInvariantProgram()
	prog.Funcs[0].LocalSlots = 6
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:4], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: denominator},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[4:]...)...)
	prog.Funcs[0].Instrs[16] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 4}
	prog.Funcs[0].Instrs[17] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 5}
	prog.Funcs[0].Instrs[18] = ir.IRInstr{Kind: kind}
	return prog
}

func licmKnownLocalArithmeticInvariantProgram(kind ir.IRInstrKind, right int32) *ir.IRProgram {
	prog := licmInvariantProgram()
	prog.Funcs[0].LocalSlots = 6
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:4], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: right},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[4:]...)...)
	prog.Funcs[0].Instrs[16] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 4}
	prog.Funcs[0].Instrs[17] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 5}
	prog.Funcs[0].Instrs[18] = ir.IRInstr{Kind: kind}
	return prog
}

func licmKnownLocalLeftArithmeticInvariantProgram(kind ir.IRInstrKind, left int32) *ir.IRProgram {
	prog := licmInvariantProgram()
	prog.Funcs[0].LocalSlots = 6
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:4], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: left},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[4:]...)...)
	prog.Funcs[0].Instrs[16] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 5}
	prog.Funcs[0].Instrs[17] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 4}
	prog.Funcs[0].Instrs[18] = ir.IRInstr{Kind: kind}
	return prog
}

func licmKnownLocalComparisonInvariantProgram(kind ir.IRInstrKind, value int32, knownOnLeft bool) *ir.IRProgram {
	prog := licmInvariantProgram()
	prog.Funcs[0].LocalSlots = 6
	prog.Funcs[0].Instrs = append(prog.Funcs[0].Instrs[:4], append([]ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: value},
		{Kind: ir.IRStoreLocal, Local: 5},
	}, prog.Funcs[0].Instrs[4:]...)...)
	if knownOnLeft {
		prog.Funcs[0].Instrs[16] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 5}
		prog.Funcs[0].Instrs[17] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 4}
	} else {
		prog.Funcs[0].Instrs[16] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 4}
		prog.Funcs[0].Instrs[17] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 5}
	}
	prog.Funcs[0].Instrs[18] = ir.IRInstr{Kind: kind}
	return prog
}
