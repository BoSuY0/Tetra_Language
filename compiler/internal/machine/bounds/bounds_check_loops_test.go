package bounds

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

func TestBoundsCheckLoopsFunctionFromStackIRLowersExactBenchmarkMain(t *testing.T) {
	plan, ok, err := BoundsCheckLoopsPlanFromStackIR(boundsCheckLoopsStackIRFunc())
	if err != nil {
		t.Fatalf("BoundsCheckLoopsPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("BoundsCheckLoopsPlanFromStackIR did not accept p25 bounds_check_loops main")
	}
	if plan.Function.Target != "bounds-check-loops" {
		t.Fatalf("bounds-check-loops target = %q, want bounds-check-loops", plan.Function.Target)
	}
	if plan.SliceLength != 4096 || plan.SliceBytes != 16384 ||
		plan.FillModulus != 97 || plan.HotLoopBound != 200000 ||
		plan.IndexMultiplier != 17 ||
		plan.SuccessReturn != 0 || plan.FailureReturn != 1 {
		t.Fatalf("bounds-check-loops constants = %#v, want exact benchmark constants", plan)
	}
	if err := machine.VerifyFunction(plan.Function); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	text := machine.FormatFunction(plan.Function)
	for _, want := range []string{
		"func p25.bounds_check_loops.main target:bounds-check-loops",
		"index_store",
		"fill xs[i] = i % 97",
		"mod",
		"idx = (i * 17) % 4096",
		"index_load",
		"proof:modulo:",
		"total >= 0",
		"return uses:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("bounds-check-loops machine dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := machine.BuildIntervals(plan.Function)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := machine.LinearScan(intervals, machine.LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"bounds-check-loops should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestBoundsCheckLoopsFunctionFromStackIRRejectsAlteredShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.IRFunc)
	}{
		{
			name: "different_slice_length",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[0].Imm = 4095
				fn.Instrs[3].Imm = 4095
			},
		},
		{
			name: "different_fill_modulus",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[17].Imm = 96
			},
		},
		{
			name: "different_hot_loop_bound",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[32].Imm = 199999
			},
		},
		{
			name: "different_index_multiplier",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[36].Imm = 19
			},
		},
		{
			name: "proofless_modulo_index_load",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[45].Kind = ir.IRIndexLoadI32
				fn.Instrs[45].ProofID = ""
			},
		},
		{
			name: "altered_final_guard",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[56].Kind = ir.IRCmpGtI32
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := boundsCheckLoopsStackIRFunc()
			tc.mutate(&fn)
			if _, ok, err := BoundsCheckLoopsFunctionFromStackIR(fn); err != nil || ok {
				t.Fatalf(
					"BoundsCheckLoopsFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func boundsCheckLoopsStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.bounds_check_loops.main",
		ExportName:  "main",
		LocalSlots:  2054,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4096},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRStackSliceI32, Local: 6, ArgSlots: 2048, Imm: 4096, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 97},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while:i:xs:8:5"},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 200000},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 17},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRModI32},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:modulo:idx:xs:14:33"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGeI32},
			{Kind: ir.IRJmpIfZero, Label: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 4},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}
