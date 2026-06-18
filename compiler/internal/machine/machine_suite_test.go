package machine

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"strings"
	"testing"
	"tetra_language/compiler/internal/ir"
)

// ---- ir_test.go ----

func TestLivenessComputesLoopCarriedLiveSets(t *testing.T) {
	fn := SumToLoopFunction()
	live, err := AnalyzeLiveness(fn)
	if err != nil {
		t.Fatalf("AnalyzeLiveness: %v", err)
	}
	loop := live.Blocks["loop"]
	for _, want := range []VReg{"i", "n", "total"} {
		if !containsReg(loop.LiveIn, want) {
			t.Fatalf("loop live-in = %v, missing %s", loop.LiveIn, want)
		}
	}
	for _, want := range []VReg{"i", "n", "total"} {
		if !containsReg(loop.LiveOut, want) {
			t.Fatalf("loop live-out = %v, missing %s", loop.LiveOut, want)
		}
	}
}

func TestLinearScanAssignsRegistersAndSpillsOnlyWhenPressured(t *testing.T) {
	intervals := []Interval{
		{Reg: "a", Start: 0, End: 9},
		{Reg: "b", Start: 1, End: 7},
		{Reg: "c", Start: 2, End: 3},
	}
	wide, err := LinearScan(intervals, []PhysReg{"r1", "r2", "r3"})
	if err != nil {
		t.Fatalf("LinearScan wide: %v", err)
	}
	if len(wide.Spills) != 0 {
		t.Fatalf("wide allocation spills = %v, want none", wide.Spills)
	}
	tight, err := LinearScan(intervals, []PhysReg{"r1", "r2"})
	if err != nil {
		t.Fatalf("LinearScan tight: %v", err)
	}
	if len(tight.Spills) != 1 {
		t.Fatalf("tight allocation spills = %v, want one spill", tight.Spills)
	}
	if _, ok := tight.Assignments["c"]; !ok {
		t.Fatalf(
			"short interval c should stay in a register: assignments=%v spills=%v",
			tight.Assignments,
			tight.Spills,
		)
	}
}

func TestSumToMachineIRHotLoopHasNoPushPopChurn(t *testing.T) {
	fn := SumToLoopFunction()
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	text := FormatFunction(fn)
	for _, forbidden := range []string{string(OpPush), string(OpPop)} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("hot loop machine IR contains stack churn opcode %q:\n%s", forbidden, text)
		}
	}
	for _, want := range []string{"loop:", "add", "inc", "cmp", "return"} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine IR missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(fn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf("sum_to should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
	}
}

func TestVerifierRejectsUndefinedVRegUse(t *testing.T) {
	fn := Function{
		Name: "bad",
		Blocks: []Block{{
			Name: "entry",
			Instrs: []Instr{
				{Op: OpReturn, Uses: []VReg{"missing"}},
			},
		}},
	}
	if err := VerifyFunction(fn); err == nil || !strings.Contains(err.Error(), "undefined vreg") {
		t.Fatalf("VerifyFunction error = %v, want undefined vreg rejection", err)
	}
}

func TestVerifierRejectsBlockWithoutTerminator(t *testing.T) {
	fn := Function{
		Name: "bad",
		Blocks: []Block{{
			Name: "entry",
			Instrs: []Instr{
				{Op: OpMov, Defs: []VReg{"x"}, Imm: 1},
			},
		}},
	}
	if err := VerifyFunction(fn); err == nil ||
		!strings.Contains(err.Error(), "missing terminator") {
		t.Fatalf("VerifyFunction error = %v, want missing terminator rejection", err)
	}
}

func TestVerifierRejectsUnknownBranchTarget(t *testing.T) {
	fn := Function{
		Name: "bad",
		Blocks: []Block{{
			Name:       "entry",
			Instrs:     []Instr{{Op: OpBranch, Target: "missing"}},
			Successors: []string{"missing"},
		}},
	}
	if err := VerifyFunction(fn); err == nil ||
		!strings.Contains(err.Error(), "unknown branch target") {
		t.Fatalf("VerifyFunction error = %v, want unknown branch target rejection", err)
	}
}

func TestVerifierRejectsCallWithoutABIClobbers(t *testing.T) {
	fn := Function{
		Name:   "bad",
		Params: []VReg{"arg"},
		Blocks: []Block{{
			Name: "entry",
			Instrs: []Instr{
				{Op: OpCall, Call: "callee", ABI: "sysv", Uses: []VReg{"arg"}, Defs: []VReg{"ret"}},
				{Op: OpReturn, Uses: []VReg{"ret"}},
			},
		}},
	}
	if err := VerifyFunction(fn); err == nil ||
		!strings.Contains(err.Error(), "missing clobber metadata") {
		t.Fatalf("VerifyFunction error = %v, want missing clobber metadata rejection", err)
	}
}

func TestVerifierAcceptsMemoryOpsAndCallWithABIClobbers(t *testing.T) {
	fn := Function{
		Name:   "ok",
		Params: []VReg{"addr"},
		Blocks: []Block{{
			Name: "entry",
			Instrs: []Instr{
				{Op: OpLoad, Defs: []VReg{"loaded"}, Uses: []VReg{"addr"}},
				{Op: OpStore, Uses: []VReg{"addr", "loaded"}},
				{
					Op:       OpCall,
					Call:     "callee",
					ABI:      "sysv",
					Clobbers: LinuxX64CallerSaved(),
					Uses:     []VReg{"loaded"},
					Defs:     []VReg{"ret"},
				},
				{Op: OpReturn, Uses: []VReg{"ret"}},
			},
		}},
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
}

func TestVerifyAllocationRejectsInvalidPhysRegAndSpillSlot(t *testing.T) {
	fn := Function{
		Name: "alloc",
		Blocks: []Block{{
			Name: "entry",
			Instrs: []Instr{
				{Op: OpMov, Defs: []VReg{"a"}, Imm: 1},
				{Op: OpMov, Defs: []VReg{"b"}, Imm: 2},
				{Op: OpAdd, Defs: []VReg{"c"}, Uses: []VReg{"a", "b"}},
				{Op: OpReturn, Uses: []VReg{"c"}},
			},
		}},
	}
	badPhys := Allocation{Assignments: map[VReg]PhysReg{"a": "not-a-reg"}}
	if err := VerifyAllocation(fn, badPhys, []PhysReg{"r1"}, 0); err == nil ||
		!strings.Contains(err.Error(), "invalid physreg") {
		t.Fatalf("VerifyAllocation bad physreg error = %v, want invalid physreg", err)
	}
	badSpill := Allocation{Spills: map[VReg]int{"a": 1}}
	if err := VerifyAllocation(fn, badSpill, []PhysReg{"r1"}, 1); err == nil ||
		!strings.Contains(err.Error(), "spill slot") {
		t.Fatalf("VerifyAllocation bad spill error = %v, want spill slot bounds", err)
	}
}

func TestFormatProgramProvidesStableMachineIRDump(t *testing.T) {
	text := FormatProgram(Program{Functions: []Function{SumToLoopFunction()}})
	for _, want := range []string{
		"program machine_ir",
		"func sum_to target:linux-x64",
		"loop:",
		"return uses:total",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatProgram missing %q:\n%s", want, text)
		}
	}
}

func TestScalarIntFunctionFromStackIRLowersSimpleAdd(t *testing.T) {
	fn := ir.IRFunc{
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
	}
	mfn, ok, err := ScalarIntFunctionFromStackIR(fn)
	if err != nil {
		t.Fatalf("ScalarIntFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntFunctionFromStackIR did not accept simple add")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{"func add", "params:local0,local1", "add defs:", "return uses:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine scalar dump missing %q:\n%s", want, text)
		}
	}
}

func TestScalarIntFunctionFromStackIRLowersDivMod(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "div_mod",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRModI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
	mfn, ok, err := ScalarIntFunctionFromStackIR(fn)
	if err != nil {
		t.Fatalf("ScalarIntFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntFunctionFromStackIR did not accept div/mod scalar function")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{"div defs:", "mod defs:", "add defs:", "return uses:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine scalar div/mod dump missing %q:\n%s", want, text)
		}
	}
}

func TestScalarIntLoopFunctionFromStackIRLowersSumNLoop(t *testing.T) {
	mfn, ok, err := ScalarIntLoopFunctionFromStackIR(sumNStackIRFunc())
	if err != nil {
		t.Fatalf("ScalarIntLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntLoopFunctionFromStackIR did not accept sum_n loop")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func sum_n",
		"params:local0",
		"label1:",
		"cmp defs:",
		"branch_if uses:",
		"add defs:local2 uses:local2,local1",
		"inc defs:local1 uses:local1",
		"return uses:local2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine loop dump missing %q:\n%s", want, text)
		}
	}
	live, err := AnalyzeLiveness(mfn)
	if err != nil {
		t.Fatalf("AnalyzeLiveness: %v", err)
	}
	loop := live.Blocks["label1"]
	for _, want := range []VReg{"local0", "local1", "local2"} {
		if !containsReg(loop.LiveIn, want) || !containsReg(loop.LiveOut, want) {
			t.Fatalf("loop liveness = %+v, want %s live-in and live-out", loop, want)
		}
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"sum_n loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestScalarIntLoopFunctionFromStackIRLowersConstantStrideLoop(t *testing.T) {
	mfn, ok, err := ScalarIntLoopFunctionFromStackIR(sumStrideStackIRFunc(2))
	if err != nil {
		t.Fatalf("ScalarIntLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntLoopFunctionFromStackIR did not accept constant-stride loop")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func sum_stride",
		"params:local0",
		"label1:",
		"cmp defs:",
		"branch_if uses:",
		"add defs:local2 uses:local2,local1",
		"add defs:local1 uses:local1,t1",
		"return uses:local2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine constant-stride loop dump missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "inc defs:local1 uses:local1") {
		t.Fatalf("constant-stride loop should use explicit stride add, not inc:\n%s", text)
	}
	if !hasMachineImmDef(mfn, "t1", 2) {
		t.Fatalf("constant-stride loop missing step immediate 2 in machine IR: %+v", mfn)
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"constant-stride loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestScalarIntLoopFunctionFromStackIRRejectsInvalidConstantStrideLoop(t *testing.T) {
	for _, step := range []int32{0, -1, 128} {
		if _, ok, err := ScalarIntLoopFunctionFromStackIR(sumStrideStackIRFunc(step)); err != nil ||
			ok {
			t.Fatalf(
				"ScalarIntLoopFunctionFromStackIR step %d ok=%v err=%v, want fallback without error",
				step,
				ok,
				err,
			)
		}
	}
}

func TestScalarIntConstModuloLoopFunctionFromStackIRLowersIntegerLoopsBenchmark(t *testing.T) {
	mfn, ok, err := ScalarIntConstModuloLoopFunctionFromStackIR(integerLoopsBenchmarkStackIRFunc())
	if err != nil {
		t.Fatalf("ScalarIntConstModuloLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf(
			"ScalarIntConstModuloLoopFunctionFromStackIR did not accept integer_loops benchmark",
		)
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func p25.integer_loops.main target:scalar-int-const-modulo-loop",
		"mov defs:t1 ; literal loop bound",
		"mov defs:t2 ; literal modulo divisor",
		"mod defs:t3 uses:local0,t2",
		"add defs:local1 uses:local1,t3",
		"cmp defs:t5 uses:local1,t4 ; total >= 0",
		"branch_if uses:t5 -> label2",
		"return uses:t6",
		"return uses:t7",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine const-modulo loop dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"integer_loops benchmark loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestScalarIntConstModuloLoopFunctionFromStackIRRejectsNonBenchmarkShape(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.IRFunc)
	}{
		{
			name: "different_bound",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[6].Imm = 1000
			},
		},
		{
			name: "different_modulus",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[11].Imm = 5
			},
		},
		{
			name: "different_final_guard",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[23].Kind = ir.IRCmpGtI32
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := integerLoopsBenchmarkStackIRFunc()
			tc.mutate(&fn)
			if _, ok, err := ScalarIntConstModuloLoopFunctionFromStackIR(fn); err != nil || ok {
				t.Fatalf(
					"ScalarIntConstModuloLoopFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func TestScalarIntSumSquaresLoopFunctionFromStackIRLowersMulLoop(t *testing.T) {
	mfn, ok, err := ScalarIntSumSquaresLoopFunctionFromStackIR(sumSquaresStackIRFunc())
	if err != nil {
		t.Fatalf("ScalarIntSumSquaresLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntSumSquaresLoopFunctionFromStackIR did not accept sum_squares loop")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func sum_squares",
		"params:local0",
		"label1:",
		"cmp defs:",
		"branch_if uses:",
		"mul defs:",
		"add defs:local2 uses:local2,t1",
		"inc defs:local1 uses:local1",
		"return uses:local2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine sum-squares loop dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"sum_squares loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestScalarIntProductLoopFunctionFromStackIRLowersProductReductionLoop(t *testing.T) {
	mfn, ok, err := ScalarIntProductLoopFunctionFromStackIR(productStackIRFunc())
	if err != nil {
		t.Fatalf("ScalarIntProductLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntProductLoopFunctionFromStackIR did not accept product_n loop")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func product_n",
		"params:local0",
		"label1:",
		"cmp defs:",
		"branch_if uses:",
		"add defs:t2 uses:local1,t1",
		"mul defs:local2 uses:local2,t2",
		"inc defs:local1 uses:local1",
		"return uses:local2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine product loop dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"product_n loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestScalarIntMaxLoopFunctionFromStackIRLowersBranchyMaxReductionLoop(t *testing.T) {
	mfn, ok, err := ScalarIntMaxLoopFunctionFromStackIR(maxStackIRFunc())
	if err != nil {
		t.Fatalf("ScalarIntMaxLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntMaxLoopFunctionFromStackIR did not accept max_n loop")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func max_n",
		"params:local0",
		"label1:",
		"cmp defs:",
		"branch_if uses:",
		"branch -> label3",
		"mov defs:local1 uses:local2",
		"inc defs:local2 uses:local2",
		"return uses:local1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine max loop dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"max_n loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestScalarIntAffineLoopFunctionFromStackIRLowersScaleBiasLoop(t *testing.T) {
	mfn, ok, err := ScalarIntAffineLoopFunctionFromStackIR(sumAffineStackIRFunc(2, 1))
	if err != nil {
		t.Fatalf("ScalarIntAffineLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntAffineLoopFunctionFromStackIR did not accept sum_affine loop")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func sum_affine",
		"params:local0",
		"label1:",
		"cmp defs:",
		"branch_if uses:",
		"mul defs:",
		"add defs:",
		"add defs:local2",
		"inc defs:local1 uses:local1",
		"return uses:local2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine affine loop dump missing %q:\n%s", want, text)
		}
	}
	if !hasMachineImmDef(mfn, "t1", 2) {
		t.Fatalf("affine loop missing scale immediate 2 in machine IR: %+v", mfn)
	}
	if !hasMachineImmDef(mfn, "t2", 1) {
		t.Fatalf("affine loop missing bias immediate 1 in machine IR: %+v", mfn)
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"sum_affine loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestScalarIntAffineLoopFunctionFromStackIRRejectsInvalidConstants(t *testing.T) {
	for _, tc := range []struct {
		scale int32
		bias  int32
	}{
		{0, 1},
		{2, 0},
		{128, 1},
		{2, 128},
	} {
		if _, ok, err := ScalarIntAffineLoopFunctionFromStackIR(
			sumAffineStackIRFunc(tc.scale, tc.bias),
		); err != nil ||
			ok {
			t.Fatalf(
				("ScalarIntAffineLoopFunctionFromStackIR scale=%d bias=%d ok=%v " +
					"err=%v, want fallback without error"),
				tc.scale,
				tc.bias,
				ok,
				err,
			)
		}
	}
}

func TestScalarIntCountdownLoopFunctionFromStackIRLowersDescendingLoop(t *testing.T) {
	mfn, ok, err := ScalarIntCountdownLoopFunctionFromStackIR(countdownStackIRFunc())
	if err != nil {
		t.Fatalf("ScalarIntCountdownLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntCountdownLoopFunctionFromStackIR did not accept sum_countdown loop")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func sum_countdown",
		"params:local0",
		"label1:",
		"cmp defs:",
		"branch_if uses:",
		"add defs:local1 uses:local1,local0",
		"sub defs:local0 uses:local0,t2",
		"return uses:local1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine countdown loop dump missing %q:\n%s", want, text)
		}
	}
	live, err := AnalyzeLiveness(mfn)
	if err != nil {
		t.Fatalf("AnalyzeLiveness: %v", err)
	}
	loop := live.Blocks["label1"]
	for _, want := range []VReg{"local0", "local1"} {
		if !containsReg(loop.LiveIn, want) || !containsReg(loop.LiveOut, want) {
			t.Fatalf("countdown-loop liveness = %+v, want %s live-in and live-out", loop, want)
		}
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"sum_countdown loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestScalarI32SliceSumLoopFromStackIRRequiresProofTaggedUncheckedLoad(t *testing.T) {
	mfn, ok, err := ScalarI32SliceSumLoopFunctionFromStackIR(sliceSumStackIRFunc(true))
	if err != nil {
		t.Fatalf("ScalarI32SliceSumLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarI32SliceSumLoopFunctionFromStackIR did not accept proof-tagged slice sum")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func sum",
		"params:local0,local1",
		"index_load defs:",
		"proof:while:",
		"add defs:local2",
		"return uses:local2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine slice-sum dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"slice sum loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
	if _, ok, err := ScalarI32SliceSumLoopFunctionFromStackIR(
		sliceSumStackIRFunc(false),
	); err != nil ||
		ok {
		t.Fatalf("checked/no-proof slice sum ok=%v err=%v, want fallback without error", ok, err)
	}
}

func TestSliceSumMainPlanFromStackIRAcceptsExactP25SliceSumMain(t *testing.T) {
	plan, ok, err := SliceSumMainPlanFromStackIR(sliceSumMainStackIRFunc())
	if err != nil {
		t.Fatalf("SliceSumMainPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("SliceSumMainPlanFromStackIR did not accept exact p25.slice_sum.main")
	}
	if plan.Function.Name != "p25.slice_sum.main" || plan.Function.Target != "slice-sum-main" {
		t.Fatalf("slice_sum main machine identity = %#v", plan.Function)
	}
	if plan.Length != 4096 || plan.FillModulus != 97 || plan.RepeatCount != 64 ||
		plan.Step != 1 {
		t.Fatalf(
			"slice_sum main constants = length %d modulus %d repeats %d step %d, want 4096/97/64/1",
			plan.Length,
			plan.FillModulus,
			plan.RepeatCount,
			plan.Step,
		)
	}
	if plan.StoreProofID != "proof:while:i:xs:8:5" ||
		plan.LoadProofID != "proof:while:i:xs:15:9" ||
		plan.BoundsChecks != 2 {
		t.Fatalf(
			"slice_sum proof/bounds = store %q load %q checks %d, want exact removed store/load",
			plan.StoreProofID,
			plan.LoadProofID,
			plan.BoundsChecks,
		)
	}
	if _, ok, err := ScalarI32SliceSumLoopFunctionFromStackIR(sliceSumMainStackIRFunc()); err != nil ||
		ok {
		t.Fatalf("helper slice-sum recognizer ok=%v err=%v for main row, want strict fallback", ok, err)
	}
	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func p25.slice_sum.main target:slice-sum-main",
		"index_store",
		"proof:while:i:xs:8:5",
		"index_load",
		"proof:while:i:xs:15:9",
		"return uses:return_success",
		"return uses:return_failure",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("slice_sum main machine dump missing %q:\n%s", want, text)
		}
	}
	if err := VerifyFunction(plan.Function); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	intervals, err := BuildIntervals(plan.Function)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf("slice_sum main should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
	}
}

func TestSliceSumMainPlanFromStackIRRejectsNearMisses(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
	}{
		{
			name: "helper_like_slice_sum_shape",
			fn: func() ir.IRFunc {
				return sliceSumStackIRFunc(true)
			},
		},
		{
			name: "altered_length_constant",
			fn: func() ir.IRFunc {
				fn := sliceSumMainStackIRFunc()
				fn.Instrs[0].Imm = 4097
				return fn
			},
		},
		{
			name: "altered_fill_modulus",
			fn: func() ir.IRFunc {
				fn := sliceSumMainStackIRFunc()
				fn.Instrs[17].Imm = 96
				return fn
			},
		},
		{
			name: "altered_loop_order",
			fn: func() ir.IRFunc {
				fn := sliceSumMainStackIRFunc()
				fn.Instrs[35], fn.Instrs[37] = fn.Instrs[37], fn.Instrs[35]
				return fn
			},
		},
		{
			name: "missing_final_branch",
			fn: func() ir.IRFunc {
				fn := sliceSumMainStackIRFunc()
				fn.Instrs = fn.Instrs[:64]
				return fn
			},
		},
		{
			name: "missing_store_proof",
			fn: func() ir.IRFunc {
				fn := sliceSumMainStackIRFunc()
				fn.Instrs[19].ProofID = ""
				return fn
			},
		},
		{
			name: "checked_load_without_unchecked_proof",
			fn: func() ir.IRFunc {
				fn := sliceSumMainStackIRFunc()
				fn.Instrs[46].Kind = ir.IRIndexLoadI32
				fn.Instrs[46].ProofID = ""
				return fn
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok, err := SliceSumMainPlanFromStackIR(tc.fn()); err != nil || ok {
				t.Fatalf(
					"SliceSumMainPlanFromStackIR near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestMatrixMultiplyMainPlanFromStackIRAcceptsExactP25MatrixMultiplyMain(t *testing.T) {
	plan, ok, err := MatrixMultiplyMainPlanFromStackIR(matrixMultiplyMainStackIRFunc())
	if err != nil {
		t.Fatalf("MatrixMultiplyMainPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("MatrixMultiplyMainPlanFromStackIR did not accept exact p25.matrix_multiply.main")
	}
	if plan.Function.Name != "p25.matrix_multiply.main" ||
		plan.Function.Target != "matrix-multiply-main" {
		t.Fatalf("matrix_multiply main machine identity = %#v", plan.Function)
	}
	if plan.SliceLength != 9 || plan.Dimension != 3 || plan.RepeatCount != 2000 ||
		plan.Step != 1 {
		t.Fatalf(
			"matrix_multiply constants = length %d dimension %d repeats %d step %d, want 9/3/2000/1",
			plan.SliceLength,
			plan.Dimension,
			plan.RepeatCount,
			plan.Step,
		)
	}
	if plan.AFillProofID != "proof:while-const:i:a:10:9" ||
		plan.BFillProofID != "proof:while-const:i:b:11:9" ||
		plan.CFillProofID != "proof:while-const:i:c:12:9" ||
		plan.ARowKProofID != "proof:affine-const:row_k:a:24:38" ||
		plan.BKColProofID != "proof:affine-const:k_col:b:24:55" ||
		plan.CRowColProofID != "proof:affine-const:row_col:c:26:19" ||
		plan.CModuloProofID != "proof:modulo:modulo_const:c:29:37" ||
		plan.BoundsChecks != 7 {
		t.Fatalf("matrix_multiply proof/bounds mismatch: %#v", plan)
	}
	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func p25.matrix_multiply.main target:matrix-multiply-main",
		"index_store",
		"proof:while-const:i:a:10:9",
		"proof:while-const:i:b:11:9",
		"proof:while-const:i:c:12:9",
		"index_load",
		"proof:affine-const:row_k:a:24:38",
		"proof:affine-const:k_col:b:24:55",
		"proof:affine-const:row_col:c:26:19",
		"proof:modulo:modulo_const:c:29:37",
		"return uses:return_success",
		"return uses:return_failure",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("matrix_multiply main machine dump missing %q:\n%s", want, text)
		}
	}
	if err := VerifyFunction(plan.Function); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	if _, err := BuildIntervals(plan.Function); err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
}

func TestMatrixMultiplyMainPlanFromStackIRRejectsNearMisses(t *testing.T) {
	insertCall := func(fn ir.IRFunc) ir.IRFunc {
		instrs := append([]ir.IRInstr(nil), fn.Instrs[:71]...)
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRCall, Name: "helper", ArgSlots: 0, RetSlots: 0})
		instrs = append(instrs, fn.Instrs[71:]...)
		fn.Instrs = instrs
		return fn
	}
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
	}{
		{
			name: "altered_allocation_count",
			fn: func() ir.IRFunc {
				fn := matrixMultiplyMainStackIRFunc()
				fn.Instrs[5].ArgSlots = 6
				return fn
			},
		},
		{
			name: "changed_outer_loop_constant",
			fn: func() ir.IRFunc {
				fn := matrixMultiplyMainStackIRFunc()
				fn.Instrs[50].Imm = 1999
				return fn
			},
		},
		{
			name: "altered_loop_order",
			fn: func() ir.IRFunc {
				fn := matrixMultiplyMainStackIRFunc()
				fn.Instrs[53], fn.Instrs[55] = fn.Instrs[55], fn.Instrs[53]
				return fn
			},
		},
		{
			name: "proof_id_tampering",
			fn: func() ir.IRFunc {
				fn := matrixMultiplyMainStackIRFunc()
				fn.Instrs[84].ProofID = "proof:tampered"
				return fn
			},
		},
		{
			name: "inserted_call",
			fn: func() ir.IRFunc {
				return insertCall(matrixMultiplyMainStackIRFunc())
			},
		},
		{
			name: "missing_final_branch",
			fn: func() ir.IRFunc {
				fn := matrixMultiplyMainStackIRFunc()
				fn.Instrs = fn.Instrs[:141]
				return fn
			},
		},
		{
			name: "runtime_effect_shape",
			fn: func() ir.IRFunc {
				fn := matrixMultiplyMainStackIRFunc()
				fn.Instrs[1].Kind = ir.IRMakeSliceI32
				return fn
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok, err := MatrixMultiplyMainPlanFromStackIR(tc.fn()); err != nil || ok {
				t.Fatalf(
					"MatrixMultiplyMainPlanFromStackIR near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestRegionIslandAllocationMainPlanFromStackIRAcceptsExactP25RegionIslandMain(
	t *testing.T,
) {
	plan, ok, err := RegionIslandAllocationMainPlanFromStackIR(
		regionIslandAllocationMainStackIRFunc(),
	)
	if err != nil {
		t.Fatalf("RegionIslandAllocationMainPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf(
			"RegionIslandAllocationMainPlanFromStackIR did not accept exact " +
				"p25.region_island_allocation.main",
		)
	}
	if plan.Function.Name != "p25.region_island_allocation.main" ||
		plan.Function.Target != "region-island-allocation-main" {
		t.Fatalf("region_island_allocation machine identity = %#v", plan.Function)
	}
	if plan.LoopBound != 256 || plan.SliceLength != 16 || plan.IslandBudget != 256 ||
		plan.Step != 1 {
		t.Fatalf(
			"region_island_allocation constants = bound %d len %d budget %d step %d, want 256/16/256/1",
			plan.LoopBound,
			plan.SliceLength,
			plan.IslandBudget,
			plan.Step,
		)
	}
	for _, proofID := range []string{plan.StoreProofID, plan.LoadProofID} {
		if !strings.HasPrefix(proofID, "proof:allocation-zero:literal0:xs:") {
			t.Fatalf("region_island_allocation proof id = %q, want allocation literal-zero proof", proofID)
		}
	}
	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func p25.region_island_allocation.main target:region-island-allocation-main",
		"index_store",
		"index_load",
		"island cleanup before loop backedge",
		"return uses:return_success",
		"return uses:return_failure",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("region_island_allocation machine dump missing %q:\n%s", want, text)
		}
	}
	if err := VerifyFunction(plan.Function); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	intervals, err := BuildIntervals(plan.Function)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"region_island_allocation main should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestRegionIslandAllocationMainPlanFromStackIRRejectsNearMisses(t *testing.T) {
	insert := func(fn ir.IRFunc, at int, instr ir.IRInstr) ir.IRFunc {
		instrs := append([]ir.IRInstr(nil), fn.Instrs[:at]...)
		instrs = append(instrs, instr)
		instrs = append(instrs, fn.Instrs[at:]...)
		fn.Instrs = instrs
		return fn
	}
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
	}{
		{
			name: "extra_runtime_call",
			fn: func() ir.IRFunc {
				return insert(
					regionIslandAllocationMainStackIRFunc(),
					29,
					ir.IRInstr{Kind: ir.IRCall, Name: "runtime.unrelated", ArgSlots: 0, RetSlots: 0},
				)
			},
		},
		{
			name: "changed_island_allocation_length",
			fn: func() ir.IRFunc {
				fn := regionIslandAllocationMainStackIRFunc()
				fn.Instrs[13].Imm = 17
				return fn
			},
		},
		{
			name: "missing_island_cleanup",
			fn: func() ir.IRFunc {
				fn := regionIslandAllocationMainStackIRFunc()
				fn.Instrs[30].Kind = ir.IRConstI32
				return fn
			},
		},
		{
			name: "extra_island_op",
			fn: func() ir.IRFunc {
				return insert(
					regionIslandAllocationMainStackIRFunc(),
					30,
					ir.IRInstr{Kind: ir.IRIslandReset},
				)
			},
		},
		{
			name: "changed_function_name",
			fn: func() ir.IRFunc {
				fn := regionIslandAllocationMainStackIRFunc()
				fn.Name = "p25.region_island_allocation.helper"
				return fn
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok, err := RegionIslandAllocationMainPlanFromStackIR(tc.fn()); err != nil || ok {
				t.Fatalf(
					"RegionIslandAllocationMainPlanFromStackIR near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestAllocationLoopFunctionFromStackIRLowersP25AllocationMain(t *testing.T) {
	plan, ok, err := AllocationLoopPlanFromStackIR(allocationLoopStackIRFunc())
	if err != nil {
		t.Fatalf("AllocationLoopPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("AllocationLoopPlanFromStackIR did not accept p25 allocation loop")
	}
	if plan.LoopBound != 1024 || plan.SliceLength != 32 || plan.IndexConst != 0 || plan.Step != 1 {
		t.Fatalf(
			"allocation loop constants = bound %d slice %d index %d step %d, want 1024/32/0/1",
			plan.LoopBound,
			plan.SliceLength,
			plan.IndexConst,
			plan.Step,
		)
	}
	if plan.BoundsChecks != 2 {
		t.Fatalf(
			"allocation loop bounds checks = %d, want checked store + checked load",
			plan.BoundsChecks,
		)
	}
	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func p25.allocation.main target:allocation-loop",
		"index_store",
		"checked xs[0] = r",
		"index_load",
		"checked xs[0]",
		"checksum > 0",
		"return uses:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("allocation-loop machine dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(plan.Function)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"allocation loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestP56AllocationLoopFunctionFromStackIRAcceptsExactP55ProofTaggedShape(t *testing.T) {
	plan, ok, err := AllocationLoopPlanFromStackIR(allocationLoopP55ProofStackIRFunc())
	if err != nil {
		t.Fatalf("AllocationLoopPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf(
			"AllocationLoopPlanFromStackIR did not accept exact P55 proof-tagged allocation loop",
		)
	}
	if plan.LoopBound != 1024 || plan.SliceLength != 32 || plan.IndexConst != 0 || plan.Step != 1 {
		t.Fatalf(
			"allocation loop constants = bound %d slice %d index %d step %d, want 1024/32/0/1",
			plan.LoopBound,
			plan.SliceLength,
			plan.IndexConst,
			plan.Step,
		)
	}
	if plan.BoundsChecks != 2 {
		t.Fatalf(
			"allocation loop bounds checks = %d, want store + load machine shape",
			plan.BoundsChecks,
		)
	}
}

func TestAllocationLoopFunctionFromStackIRRejectsAlteredShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.IRFunc)
	}{
		{
			name: "different_function",
			mutate: func(fn *ir.IRFunc) {
				fn.Name = "p25.hash_table.main"
			},
		},
		{
			name: "different_loop_bound",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[6].Imm = 512
			},
		},
		{
			name: "different_slice_length",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[9].Imm = 16
				fn.Instrs[10].Imm = 16
			},
		},
		{
			name: "unchecked_load",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[22].Kind = ir.IRIndexLoadI32Unchecked
				fn.Instrs[22].ProofID = "proof:while:r:xs"
			},
		},
		{
			name: "proof_tagged_store_non_p55",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[17].ProofID = "proof:while:r:xs"
			},
		},
		{
			name: "p55_store_without_p55_load",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[17].ProofID = p56AllocationLoopStoreProofID
			},
		},
		{
			name: "p55_load_without_p55_store",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[22].Kind = ir.IRIndexLoadI32Unchecked
				fn.Instrs[22].ProofID = p56AllocationLoopLoadProofID
			},
		},
		{
			name: "different_final_guard",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[33].Kind = ir.IRCmpGeI32
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := allocationLoopStackIRFunc()
			tc.mutate(&fn)
			if _, ok, err := AllocationLoopFunctionFromStackIR(fn); err != nil || ok {
				t.Fatalf(
					"AllocationLoopFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func TestHashTableLookupFunctionFromStackIRLowersExactCallBoundaryShape(t *testing.T) {
	plan, ok, err := HashTableLookupPlanFromStackIR(hashTableLookupStackIRFunc())
	if err != nil {
		t.Fatalf("HashTableLookupPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("HashTableLookupPlanFromStackIR did not accept exact p25.hash_table.lookup shape")
	}
	if plan.KeysBaseLocal != 0 || plan.KeysLenLocal != 1 ||
		plan.ValuesBaseLocal != 2 || plan.ValuesLenLocal != 3 ||
		plan.BoundLocal != 4 || plan.KeyLocal != 5 || plan.IndexLocal != 6 {
		t.Fatalf("hash lookup locals = %#v, want keys(0,1) values(2,3) n=4 key=5 i=6", plan)
	}
	if plan.StartLabel != 0 || plan.EndLabel != 1 || plan.MissLabel != 2 || plan.Step != 1 ||
		plan.NotFoundReturn != 0 {
		t.Fatalf("hash lookup control constants = %#v, want labels 0/1/2 step=1 not_found=0", plan)
	}
	if !strings.HasPrefix(plan.KeysProofID, "proof:call-boundary:i:keys:") ||
		!strings.HasPrefix(plan.ValuesProofID, "proof:call-boundary:i:values:") {
		t.Fatalf(
			"hash lookup proof IDs = %q/%q, want call-boundary keys/values proofs",
			plan.KeysProofID,
			plan.ValuesProofID,
		)
	}

	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func p25.hash_table.lookup target:hash-table-lookup",
		"params:local0,local1,local2,local3,local4,local5",
		"index_load",
		"proof:call-boundary:i:keys:",
		"key match",
		"proof:call-boundary:i:values:",
		"return values[i]",
		"return not found",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("hash lookup machine dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(plan.Function)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"hash lookup should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestHashTableLookupFunctionFromStackIRRejectsNearMissShapes(t *testing.T) {
	t.Run("different_loop_branch_shape", func(t *testing.T) {
		if _, ok, err := HashTableLookupFunctionFromStackIR(
			hashTableLookupNoEarlyReturnStackIRFunc(),
		); err != nil ||
			ok {
			t.Fatalf(
				"HashTableLookupFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
				ok,
				err,
			)
		}
	})
	t.Run("checked_key_load", func(t *testing.T) {
		fn := hashTableLookupStackIRFunc()
		fn.Instrs[10].Kind = ir.IRIndexLoadI32
		fn.Instrs[10].ProofID = ""
		if _, ok, err := HashTableLookupFunctionFromStackIR(fn); err != nil || ok {
			t.Fatalf(
				"HashTableLookupFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
				ok,
				err,
			)
		}
	})
	t.Run("different_function_name", func(t *testing.T) {
		fn := hashTableLookupStackIRFunc()
		fn.Name = "p25.hash_table.lookup_probe"
		if _, ok, err := HashTableLookupFunctionFromStackIR(fn); err != nil || ok {
			t.Fatalf(
				"HashTableLookupFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
				ok,
				err,
			)
		}
	})
}

func TestHashTableMainFunctionFromStackIRLowersExactNativeSliceShape(t *testing.T) {
	plan, ok, err := HashTableMainPlanFromStackIR(hashTableMainStackIRFunc())
	if err != nil {
		t.Fatalf("HashTableMainPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("HashTableMainPlanFromStackIR did not accept exact p25.hash_table.main shape")
	}
	if plan.KeysPtrLocal != 1 || plan.KeysLenLocal != 2 ||
		plan.ValuesPtrLocal != 3 || plan.ValuesLenLocal != 4 ||
		plan.NLocal != 0 || plan.IndexLocal != 5 || plan.ChecksumLocal != 6 ||
		plan.QueryLocal != 7 || plan.KeyLocal != 8 {
		t.Fatalf("hash main locals = %#v, want n=0 keys(1,2) values(3,4) i=5 checksum=6 q=7 key=8", plan)
	}
	if plan.Length != 256 || plan.KeysBackingSlots != 128 || plan.ValuesBackingSlots != 128 ||
		plan.Step != 1 || plan.FillStartLabel != 0 || plan.FillEndLabel != 1 ||
		plan.QueryStartLabel != 2 || plan.QueryEndLabel != 3 || plan.FailureLabel != 4 {
		t.Fatalf("hash main constants = %#v, want exact p25 hash table main shape", plan)
	}
	if plan.CallName != "p25.hash_table.lookup" || plan.CallArgSlots != 6 ||
		plan.CallRetSlots != 1 {
		t.Fatalf(
			"hash main call = %q args=%d rets=%d, want lookup args=6 rets=1",
			plan.CallName,
			plan.CallArgSlots,
			plan.CallRetSlots,
		)
	}

	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func p25.hash_table.main target:hash-table-main",
		"keys stack ptr",
		"values stack ptr",
		"keys[i] = i * 2 + 1",
		"values[i] = i + 7",
		"key = q * 2 + 1",
		"call p25.hash_table.lookup",
		"checksum += lookup",
		"checksum > 0",
		"return uses:return_success",
		"return uses:return_failure",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("hash main machine dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(plan.Function)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if err := VerifyAllocation(
		plan.Function,
		alloc,
		LinuxX64CallerSaved(),
		len(alloc.Spills),
	); err != nil {
		t.Fatalf("VerifyAllocation: %v", err)
	}
}

func TestHashTableMainFunctionFromStackIRRejectsNearMissShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.IRFunc)
	}{
		{
			name: "altered_function_name",
			mutate: func(fn *ir.IRFunc) {
				fn.Name = "p25.hash_table.main_probe"
			},
		},
		{
			name: "missing_allocation",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs = append(fn.Instrs[:6], fn.Instrs[10:]...)
			},
		},
		{
			name: "extra_allocation",
			mutate: func(fn *ir.IRFunc) {
				extra := []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 4},
					{Kind: ir.IRStackSliceI32, Local: 265, ArgSlots: 2, Imm: 4, Name: "extra"},
					{Kind: ir.IRStoreLocal, Local: 2},
					{Kind: ir.IRStoreLocal, Local: 1},
				}
				fn.Instrs = append(append([]ir.IRInstr(nil), extra...), fn.Instrs...)
				fn.LocalSlots = 267
			},
		},
		{
			name: "altered_fill_loop_bound",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[14].Local = 7
			},
		},
		{
			name: "altered_query_loop_bound",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[45].Local = 5
			},
		},
		{
			name: "altered_call_target",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[61].Name = "p25.hash_table.lookup_inline"
			},
		},
		{
			name: "altered_call_arg_slots",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[61].ArgSlots = 5
			},
		},
		{
			name: "altered_call_return_slots",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[61].RetSlots = 2
			},
		},
		{
			name: "missing_final_checksum_branch",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs = fn.Instrs[:70]
			},
		},
		{
			name: "reordered_loops",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[12].Label = 2
				fn.Instrs[43].Label = 0
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := hashTableMainStackIRFunc()
			tc.mutate(&fn)
			if _, ok, err := HashTableMainFunctionFromStackIR(fn); err != nil || ok {
				t.Fatalf(
					"HashTableMainFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func TestPostgreSQLFrameTypeAtFunctionFromStackIRLowersExactU8SliceLoad(t *testing.T) {
	plan, ok, err := PostgreSQLFrameTypeAtPlanFromStackIR(postgresqlFrameTypeAtStackIRFunc())
	if err != nil {
		t.Fatalf("PostgreSQLFrameTypeAtPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("PostgreSQLFrameTypeAtPlanFromStackIR did not accept exact frame_type_at shape")
	}
	if plan.SrcBaseLocal != 0 || plan.SrcLenLocal != 1 || plan.OffsetLocal != 2 {
		t.Fatalf("frame_type_at locals = %#v, want src base=0 len=1 offset=2", plan)
	}
	if !strings.HasPrefix(plan.ProofID, "proof:helper-offset:") {
		t.Fatalf("frame_type_at proof ID = %q, want helper-offset proof", plan.ProofID)
	}

	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func p25.postgresql_single_multiple_update.frame_type_at target:postgresql-frame-type-at",
		"params:local0,local1,local2",
		"index_load defs:frame_type uses:local0,local1,local2",
		"proof:helper-offset:",
		"return src[offset]",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("frame_type_at machine dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(plan.Function)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf("frame_type_at should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
	}
}

func TestPostgreSQLFrameTypeAtFunctionFromStackIRRejectsNearMissShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.IRFunc)
	}{
		{
			name: "different_function_name",
			mutate: func(fn *ir.IRFunc) {
				fn.Name = "p25.postgresql_single_multiple_update.other_frame_type_at"
			},
		},
		{
			name: "non_u8_load",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[3].Kind = ir.IRIndexLoadI32Unchecked
			},
		},
		{
			name: "extra_control_flow",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs = append(
					fn.Instrs[:3],
					append([]ir.IRInstr{{Kind: ir.IRLabel, Label: 7}}, fn.Instrs[3:]...)...,
				)
			},
		},
		{
			name: "missing_unchecked_proof",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[3].ProofID = ""
			},
		},
		{
			name: "checked_load",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[3].Kind = ir.IRIndexLoadU8
				fn.Instrs[3].ProofID = ""
			},
		},
		{
			name: "extra_return_slot",
			mutate: func(fn *ir.IRFunc) {
				fn.ReturnSlots = 2
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := postgresqlFrameTypeAtStackIRFunc()
			tc.mutate(&fn)
			if _, ok, err := PostgreSQLFrameTypeAtFunctionFromStackIR(fn); err != nil || ok {
				t.Fatalf(
					"PostgreSQLFrameTypeAtFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func TestPostgreSQLInoutWriterFunctionFromStackIRAcceptsExactWriterShapes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		fn         ir.IRFunc
		storeCount int
		returnAdd  int32
	}{
		{
			name:       "write_i32_be_at",
			fn:         postgresqlInoutWriterI32StackIRFunc(),
			storeCount: 4,
			returnAdd:  4,
		},
		{
			name:       "write_i16_be_at",
			fn:         postgresqlInoutWriterI16StackIRFunc(),
			storeCount: 2,
			returnAdd:  2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, ok, err := PostgreSQLInoutWriterPlanFromStackIR(tc.fn)
			if err != nil {
				t.Fatalf("PostgreSQLInoutWriterPlanFromStackIR: %v", err)
			}
			if !ok {
				t.Fatalf("PostgreSQLInoutWriterPlanFromStackIR did not accept exact %s shape", tc.name)
			}
			if plan.StoreCount != tc.storeCount || plan.ReturnAddend != tc.returnAdd {
				t.Fatalf("writer plan = %#v, want stores=%d returnAdd=%d", plan, tc.storeCount, tc.returnAdd)
			}
			if plan.DstBaseLocal != 0 || plan.DstLenLocal != 1 ||
				plan.StartLocal != 2 || plan.ValueLocal != 3 {
				t.Fatalf("writer locals = %#v, want dst base=0 len=1 start=2 value=3", plan)
			}

			text := FormatFunction(plan.Function)
			for _, want := range []string{
				"target:postgresql-inout-writer",
				"params:local0,local1,local2,local3",
				"index_store",
				"return start +",
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("writer machine dump missing %q:\n%s", want, text)
				}
			}
			intervals, err := BuildIntervals(plan.Function)
			if err != nil {
				t.Fatalf("BuildIntervals: %v", err)
			}
			alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
			if err != nil {
				t.Fatalf("LinearScan: %v", err)
			}
			if len(alloc.Spills) != 0 {
				t.Fatalf("writer should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
			}
		})
	}
}

func TestPostgreSQLInoutWriterFunctionFromStackIRRejectsNearMissShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.IRFunc)
	}{
		{
			name: "different_function_name",
			mutate: func(fn *ir.IRFunc) {
				fn.Name = "p25.postgresql_single_multiple_update.other_write_i32_be_at"
			},
		},
		{
			name: "wrong_return_slots",
			mutate: func(fn *ir.IRFunc) {
				fn.ReturnSlots = 1
			},
		},
		{
			name: "missing_inout_writeback_shape",
			mutate: func(fn *ir.IRFunc) {
				fn.ParamSlots = 3
			},
		},
		{
			name: "extra_control_flow",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs = append(
					fn.Instrs[:4],
					append([]ir.IRInstr{{Kind: ir.IRLabel, Label: 7}}, fn.Instrs[4:]...)...,
				)
			},
		},
		{
			name: "checked_store_without_helper_proof",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[8].ProofID = ""
			},
		},
		{
			name: "generic_multi_slot_aggregate_return",
			mutate: func(fn *ir.IRFunc) {
				fn.Name = "slice_header_return"
				fn.ParamSlots = 0
				fn.LocalSlots = 0
				fn.ReturnSlots = 3
				fn.Instrs = []ir.IRInstr{{Kind: ir.IRReturn}}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := postgresqlInoutWriterI32StackIRFunc()
			tc.mutate(&fn)
			if _, ok, err := PostgreSQLInoutWriterFunctionFromStackIR(fn); err != nil || ok {
				t.Fatalf(
					"PostgreSQLInoutWriterFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func TestInoutWriterHelperSummaryPlanFromStackIRAcceptsExactWriterShapes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		helperName  string
		storeCount  int
		returnConst int32
	}{
		{
			name:        "json_write_message_object",
			helperName:  "p25.json_parse_stringify.write_message_object",
			storeCount:  27,
			returnConst: 27,
		},
		{
			name:        "http_write_plaintext_response",
			helperName:  "p25.http_plaintext_json.write_plaintext_response",
			storeCount:  24,
			returnConst: 24,
		},
		{
			name:        "http_write_json_response",
			helperName:  "p25.http_plaintext_json.write_json_response",
			storeCount:  21,
			returnConst: 21,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := inoutWriterHelperSummaryStackIRFunc(tc.helperName, tc.storeCount)
			plan, ok, err := InoutWriterHelperSummaryPlanFromStackIR(fn)
			if err != nil {
				t.Fatalf("InoutWriterHelperSummaryPlanFromStackIR: %v", err)
			}
			if !ok {
				t.Fatalf(
					"InoutWriterHelperSummaryPlanFromStackIR did not accept exact %s shape",
					tc.helperName,
				)
			}
			if plan.HelperName != tc.helperName || plan.StoreCount != tc.storeCount ||
				plan.ScalarReturnConst != tc.returnConst {
				t.Fatalf(
					"helper-summary plan facts = %#v, want helper=%q stores=%d return=%d",
					plan,
					tc.helperName,
					tc.storeCount,
					tc.returnConst,
				)
			}
			if plan.ParamSlots != 2 || plan.ReturnSlots != 3 ||
				plan.VisibleReturnSlots != 1 || plan.HiddenWritebackSlots != 2 {
				t.Fatalf("helper-summary ABI slots = %#v, want 2 params and 1+2 returns", plan)
			}
			if plan.DstBaseLocal != 0 || plan.DstLenLocal != 1 {
				t.Fatalf("helper-summary dst locals = %#v, want base=0 len=1", plan)
			}
			if len(plan.StoreIndexes) != tc.storeCount || len(plan.StoreValues) != tc.storeCount ||
				len(plan.ProofIDs) != tc.storeCount {
				t.Fatalf(
					"helper-summary recorded indexes/values/proofs = %d/%d/%d, want %d",
					len(plan.StoreIndexes),
					len(plan.StoreValues),
					len(plan.ProofIDs),
					tc.storeCount,
				)
			}
			for i := 0; i < tc.storeCount; i++ {
				if plan.StoreIndexes[i] != int32(i) {
					t.Fatalf("store index[%d] = %d, want %d", i, plan.StoreIndexes[i], i)
				}
				if plan.StoreValues[i] != int32(65+i%26) {
					t.Fatalf("store value[%d] = %d, want %d", i, plan.StoreValues[i], 65+i%26)
				}
				if !strings.HasPrefix(plan.ProofIDs[i], "proof:helper-summary:") {
					t.Fatalf("store proof[%d] = %q, want helper-summary proof", i, plan.ProofIDs[i])
				}
			}
		})
	}
}

func TestInoutWriterHelperSummaryPlanFromStackIRRejectsNearMissShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.IRFunc)
	}{
		{
			name: "wrong_name",
			mutate: func(fn *ir.IRFunc) {
				fn.Name = "p25.json_parse_stringify.write_other"
			},
		},
		{
			name: "wrong_param_slots",
			mutate: func(fn *ir.IRFunc) {
				fn.ParamSlots = 3
			},
		},
		{
			name: "wrong_return_slots",
			mutate: func(fn *ir.IRFunc) {
				fn.ReturnSlots = 2
			},
		},
		{
			name: "wrong_store_count",
			mutate: func(fn *ir.IRFunc) {
				*fn = inoutWriterHelperSummaryStackIRFunc(fn.Name, 26)
			},
		},
		{
			name: "missing_proof_tag",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[4].ProofID = ""
			},
		},
		{
			name: "helper_offset_proof_tag",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[4].ProofID = "proof:helper-offset:start:dst:15:5"
			},
		},
		{
			name: "dynamic_index",
			mutate: func(fn *ir.IRFunc) {
				fn.LocalSlots = 3
				fn.Instrs[2] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 2}
			},
		},
		{
			name: "non_u8_store",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[4].Kind = ir.IRIndexStoreI32
			},
		},
		{
			name: "extra_call",
			mutate: func(fn *ir.IRFunc) {
				returnAt := len(fn.Instrs) - 1
				fn.Instrs = append(
					fn.Instrs[:returnAt],
					append(
						[]ir.IRInstr{{Kind: ir.IRCall, Name: "side_effect", ArgSlots: 0, RetSlots: 0}},
						fn.Instrs[returnAt:]...,
					)...,
				)
			},
		},
		{
			name: "extra_label",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs = append(
					fn.Instrs[:5],
					append([]ir.IRInstr{{Kind: ir.IRLabel, Label: 7}}, fn.Instrs[5:]...)...,
				)
			},
		},
		{
			name: "extra_jump",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs = append(
					fn.Instrs[:5],
					append([]ir.IRInstr{{Kind: ir.IRJmp, Label: 7}}, fn.Instrs[5:]...)...,
				)
			},
		},
		{
			name: "wrong_scalar_return_constant",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[len(fn.Instrs)-4].Imm = 28
			},
		},
		{
			name: "non_constant_return",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[len(fn.Instrs)-4] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0}
			},
		},
		{
			name: "generic_aggregate_return",
			mutate: func(fn *ir.IRFunc) {
				fn.Name = "slice_header_return"
				fn.ParamSlots = 0
				fn.LocalSlots = 0
				fn.ReturnSlots = 3
				fn.Instrs = []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRReturn},
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := inoutWriterHelperSummaryStackIRFunc(
				"p25.json_parse_stringify.write_message_object",
				27,
			)
			tc.mutate(&fn)
			if _, ok, err := InoutWriterHelperSummaryPlanFromStackIR(fn); err != nil || ok {
				t.Fatalf(
					"InoutWriterHelperSummaryPlanFromStackIR ok=%v err=%v, want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func TestInoutWriterHelperSummaryFunctionFromStackIRBuildsExactWriterShapes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		helperName  string
		storeCount  int
		returnConst int32
	}{
		{
			name:        "json_write_message_object",
			helperName:  "p25.json_parse_stringify.write_message_object",
			storeCount:  27,
			returnConst: 27,
		},
		{
			name:        "http_write_plaintext_response",
			helperName:  "p25.http_plaintext_json.write_plaintext_response",
			storeCount:  24,
			returnConst: 24,
		},
		{
			name:        "http_write_json_response",
			helperName:  "p25.http_plaintext_json.write_json_response",
			storeCount:  21,
			returnConst: 21,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := inoutWriterHelperSummaryStackIRFunc(tc.helperName, tc.storeCount)
			plan, ok, err := InoutWriterHelperSummaryPlanFromStackIR(fn)
			if err != nil {
				t.Fatalf("InoutWriterHelperSummaryPlanFromStackIR: %v", err)
			}
			if !ok {
				t.Fatalf("InoutWriterHelperSummaryPlanFromStackIR did not accept %s", tc.helperName)
			}
			if plan.StoreCount != tc.storeCount ||
				plan.ScalarReturnConst != tc.returnConst ||
				len(plan.StoreIndexes) != tc.storeCount ||
				len(plan.StoreValues) != tc.storeCount ||
				len(plan.ProofIDs) != tc.storeCount {
				t.Fatalf("helper-summary plan = %#v, want stores=%d return=%d",
					plan,
					tc.storeCount,
					tc.returnConst,
				)
			}

			mfn, ok, err := InoutWriterHelperSummaryFunctionFromStackIR(fn)
			if err != nil {
				t.Fatalf("InoutWriterHelperSummaryFunctionFromStackIR: %v", err)
			}
			if !ok {
				t.Fatalf(
					"InoutWriterHelperSummaryFunctionFromStackIR did not accept exact %s shape",
					tc.helperName,
				)
			}
			if mfn.Name != tc.helperName || mfn.Target != "inout-writer-helper-summary" {
				t.Fatalf("helper-summary machine function identity = %#v", mfn)
			}
			if countMachineOp(mfn, OpIndexStore) != tc.storeCount {
				t.Fatalf(
					"helper-summary machine index_store count = %d, want %d",
					countMachineOp(mfn, OpIndexStore),
					tc.storeCount,
				)
			}
			if err := VerifyFunction(mfn); err != nil {
				t.Fatalf("VerifyFunction: %v", err)
			}
			text := FormatFunction(mfn)
			for _, want := range []string{
				"target:inout-writer-helper-summary",
				"params:local0,local1",
				"index_store",
				"proof:helper-summary:",
				fmt.Sprintf("const index %d", tc.storeCount-1),
				fmt.Sprintf("scalar return constant %d", tc.returnConst),
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("helper-summary machine dump missing %q:\n%s", want, text)
				}
			}
			intervals, err := BuildIntervals(mfn)
			if err != nil {
				t.Fatalf("BuildIntervals: %v", err)
			}
			if _, err := LinearScan(intervals, LinuxX64CallerSaved()); err != nil {
				t.Fatalf("LinearScan: %v", err)
			}
		})
	}
}

func TestInoutWriterHelperSummaryFunctionFromStackIRRejectsNearMissShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.IRFunc)
	}{
		{
			name: "wrong_helper_name",
			mutate: func(fn *ir.IRFunc) {
				fn.Name = "p25.json_parse_stringify.write_other"
			},
		},
		{
			name: "wrong_param_slots",
			mutate: func(fn *ir.IRFunc) {
				fn.ParamSlots = 3
			},
		},
		{
			name: "wrong_return_slots",
			mutate: func(fn *ir.IRFunc) {
				fn.ReturnSlots = 2
			},
		},
		{
			name: "missing_proof_family",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[4].ProofID = ""
			},
		},
		{
			name: "wrong_proof_family",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[4].ProofID = "proof:helper-offset:start:dst:15:5"
			},
		},
		{
			name: "dynamic_index",
			mutate: func(fn *ir.IRFunc) {
				fn.LocalSlots = 3
				fn.Instrs[2] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 2}
			},
		},
		{
			name: "non_constant_store_value",
			mutate: func(fn *ir.IRFunc) {
				fn.LocalSlots = 3
				fn.Instrs[3] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 2}
			},
		},
		{
			name: "extra_call",
			mutate: func(fn *ir.IRFunc) {
				returnAt := len(fn.Instrs) - 1
				fn.Instrs = append(
					fn.Instrs[:returnAt],
					append(
						[]ir.IRInstr{{Kind: ir.IRCall, Name: "side_effect", ArgSlots: 0, RetSlots: 0}},
						fn.Instrs[returnAt:]...,
					)...,
				)
			},
		},
		{
			name: "extra_label",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs = append(
					fn.Instrs[:5],
					append([]ir.IRInstr{{Kind: ir.IRLabel, Label: 7}}, fn.Instrs[5:]...)...,
				)
			},
		},
		{
			name: "extra_jump",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs = append(
					fn.Instrs[:5],
					append([]ir.IRInstr{{Kind: ir.IRJmp, Label: 7}}, fn.Instrs[5:]...)...,
				)
			},
		},
		{
			name: "wrong_scalar_return_constant",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[len(fn.Instrs)-4].Imm = 28
			},
		},
		{
			name: "generic_aggregate_return",
			mutate: func(fn *ir.IRFunc) {
				fn.Name = "slice_header_return"
				fn.ParamSlots = 0
				fn.LocalSlots = 0
				fn.ReturnSlots = 3
				fn.Instrs = []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRReturn},
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := inoutWriterHelperSummaryStackIRFunc(
				"p25.json_parse_stringify.write_message_object",
				27,
			)
			tc.mutate(&fn)
			if _, ok, err := InoutWriterHelperSummaryFunctionFromStackIR(fn); err != nil || ok {
				t.Fatalf(
					"InoutWriterHelperSummaryFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func TestInoutWriterHelperSummaryCallerPlanFromStackIRAcceptsExactJSONCaller(t *testing.T) {
	fn := inoutWriterHelperSummaryJSONCallerStackIRFunc(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "p25.json_parse_stringify.write_message_object",
			ArgSlots: 2,
			RetSlots: 3,
		},
	)
	plan, ok, err := InoutWriterHelperSummaryCallerPlanFromStackIR(fn)
	if err != nil {
		t.Fatalf("InoutWriterHelperSummaryCallerPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("InoutWriterHelperSummaryCallerPlanFromStackIR did not accept exact JSON caller")
	}
	if plan.CallerName != "p25.json_parse_stringify.main" ||
		plan.ProofFamily != "helper-summary" ||
		plan.Family != "json" ||
		plan.CallCount != 1 {
		t.Fatalf("JSON caller plan facts = %#v, want exact helper-summary JSON caller", plan)
	}
	assertHelperSummaryCallerPlanCalls(t, plan, []helperSummaryCallerPlanCallWant{
		{
			helperName: "p25.json_parse_stringify.write_message_object",
			argSlots:   2,
			retSlots:   3,
			family:     "json",
		},
	})
	assertNoNativeHelperSummaryCallerClaim(t, plan)
}

func TestInoutWriterHelperSummaryCallerPlanFromStackIRAcceptsExactHTTPCaller(t *testing.T) {
	fn := inoutWriterHelperSummaryHTTPCallerStackIRFunc(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "p25.http_plaintext_json.write_plaintext_response",
			ArgSlots: 2,
			RetSlots: 3,
		},
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "p25.http_plaintext_json.write_json_response",
			ArgSlots: 2,
			RetSlots: 3,
		},
	)
	plan, ok, err := InoutWriterHelperSummaryCallerPlanFromStackIR(fn)
	if err != nil {
		t.Fatalf("InoutWriterHelperSummaryCallerPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("InoutWriterHelperSummaryCallerPlanFromStackIR did not accept exact HTTP caller")
	}
	if plan.CallerName != "p25.http_plaintext_json.main" ||
		plan.ProofFamily != "helper-summary" ||
		plan.Family != "http" ||
		plan.CallCount != 2 {
		t.Fatalf("HTTP caller plan facts = %#v, want exact helper-summary HTTP caller", plan)
	}
	assertHelperSummaryCallerPlanCalls(t, plan, []helperSummaryCallerPlanCallWant{
		{
			helperName: "p25.http_plaintext_json.write_plaintext_response",
			argSlots:   2,
			retSlots:   3,
			family:     "http",
		},
		{
			helperName: "p25.http_plaintext_json.write_json_response",
			argSlots:   2,
			retSlots:   3,
			family:     "http",
		},
	})
	assertNoNativeHelperSummaryCallerClaim(t, plan)
}

func TestInoutWriterHelperSummaryCallerPlanFromStackIRRejectsNearMissShapes(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
	}{
		{
			name: "wrong_caller_name",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
				fn.Name = "p25.json_parse_stringify.other"
				return fn
			},
		},
		{
			name: "wrong_helper_name",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_other",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "accepted_helper_wrong_arg_slots",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 1,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "accepted_helper_wrong_ret_slots",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 2,
					},
				)
			},
		},
		{
			name: "extra_ret_slots_call",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
				fn.Instrs = append(
					fn.Instrs[:len(fn.Instrs)-1],
					append(
						inoutWriterHelperSummaryCallerCallInstrs(
							0,
							1,
							ir.IRInstr{
								Kind:     ir.IRCall,
								Name:     "p25.json_parse_stringify.unverified_pair",
								ArgSlots: 2,
								RetSlots: 2,
							},
						),
						fn.Instrs[len(fn.Instrs)-1],
					)...,
				)
				return fn
			},
		},
		{
			name: "mixed_safe_unsafe_multi_slot_calls",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryHTTPCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.http_plaintext_json.write_plaintext_response",
						ArgSlots: 2,
						RetSlots: 3,
					},
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.http_plaintext_json.unverified_writer",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "generic_aggregate_call_sample",
			fn: func() ir.IRFunc {
				return ir.IRFunc{
					Name:        "aggregate_call_sample",
					ParamSlots:  0,
					LocalSlots:  2,
					ReturnSlots: 1,
					Instrs: inoutWriterHelperSummaryCallerCallInstrs(
						0,
						1,
						ir.IRInstr{
							Kind:     ir.IRCall,
							Name:     "slice_header_return",
							ArgSlots: 2,
							RetSlots: 3,
						},
					),
				}
			},
		},
		{
			name: "json_missing_helper",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryJSONCallerStackIRFunc()
			},
		},
		{
			name: "http_missing_plaintext_helper",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryHTTPCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.http_plaintext_json.write_json_response",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "http_missing_json_helper",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryHTTPCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.http_plaintext_json.write_plaintext_response",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "duplicate_helper_call",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "wrong_caller_return_slots",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
				fn.ReturnSlots = 2
				return fn
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok, err := InoutWriterHelperSummaryCallerPlanFromStackIR(tc.fn()); err != nil || ok {
				t.Fatalf(
					"InoutWriterHelperSummaryCallerPlanFromStackIR ok=%v err=%v, "+
						"want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func TestInoutWriterHelperSummaryCallerFunctionFromStackIRBuildsExactJSONCaller(t *testing.T) {
	fn := inoutWriterHelperSummaryExactJSONCallerStackIRFunc(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "p25.json_parse_stringify.write_message_object",
			ArgSlots: 2,
			RetSlots: 3,
		},
	)
	mfn, ok, err := InoutWriterHelperSummaryCallerFunctionFromStackIR(fn)
	if err != nil {
		t.Fatalf("InoutWriterHelperSummaryCallerFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("InoutWriterHelperSummaryCallerFunctionFromStackIR did not accept JSON caller")
	}
	if mfn.Name != "p25.json_parse_stringify.main" ||
		mfn.Target != "inout-writer-helper-summary-caller" {
		t.Fatalf("JSON caller machine identity = %#v", mfn)
	}
	if err := VerifyFunction(mfn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	assertHelperSummaryCallerMachineFacts(t, mfn, "json", []helperSummaryCallerPlanCallWant{
		{
			helperName: "p25.json_parse_stringify.write_message_object",
			argSlots:   2,
			retSlots:   3,
			family:     "json",
		},
	})
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	if _, err := LinearScan(intervals, LinuxX64CallerSaved()); err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
}

func TestInoutWriterHelperSummaryCallerFunctionFromStackIRBuildsExactHTTPCaller(t *testing.T) {
	fn := inoutWriterHelperSummaryExactHTTPCallerStackIRFunc(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "p25.http_plaintext_json.write_plaintext_response",
			ArgSlots: 2,
			RetSlots: 3,
		},
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "p25.http_plaintext_json.write_json_response",
			ArgSlots: 2,
			RetSlots: 3,
		},
	)
	mfn, ok, err := InoutWriterHelperSummaryCallerFunctionFromStackIR(fn)
	if err != nil {
		t.Fatalf("InoutWriterHelperSummaryCallerFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("InoutWriterHelperSummaryCallerFunctionFromStackIR did not accept HTTP caller")
	}
	if mfn.Name != "p25.http_plaintext_json.main" ||
		mfn.Target != "inout-writer-helper-summary-caller" {
		t.Fatalf("HTTP caller machine identity = %#v", mfn)
	}
	if err := VerifyFunction(mfn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	assertHelperSummaryCallerMachineFacts(t, mfn, "http", []helperSummaryCallerPlanCallWant{
		{
			helperName: "p25.http_plaintext_json.write_plaintext_response",
			argSlots:   2,
			retSlots:   3,
			family:     "http",
		},
		{
			helperName: "p25.http_plaintext_json.write_json_response",
			argSlots:   2,
			retSlots:   3,
			family:     "http",
		},
	})
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	if _, err := LinearScan(intervals, LinuxX64CallerSaved()); err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
}

func TestInoutWriterHelperSummaryCallerFunctionFromStackIRRejectsNearMissShapes(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
	}{
		{
			name: "wrong_caller_name",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryExactJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
				fn.Name = "p25.json_parse_stringify.other"
				return fn
			},
		},
		{
			name: "wrong_helper_name",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryExactJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_other",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "missing_helper_call",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryExactJSONCallerStackIRFunc()
			},
		},
		{
			name: "duplicate_helper_call",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryExactJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "accepted_helper_wrong_arg_slots",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryExactJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 1,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "accepted_helper_wrong_ret_slots",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryExactJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 2,
					},
				)
			},
		},
		{
			name: "mixed_safe_unsafe_multi_slot_calls",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryExactHTTPCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.http_plaintext_json.write_plaintext_response",
						ArgSlots: 2,
						RetSlots: 3,
					},
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.http_plaintext_json.unverified_writer",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "generic_aggregate_call_sample",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryExactJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "slice_header_return",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
				fn.Name = "aggregate_call_sample"
				return fn
			},
		},
		{
			name: "wrong_caller_return_slots",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryExactJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
				fn.ReturnSlots = 2
				return fn
			},
		},
		{
			name: "missing_failure_scalar_return",
			fn: func() ir.IRFunc {
				return inoutWriterHelperSummaryJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
			},
		},
		{
			name: "wrong_success_scalar_return",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryExactJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
				replaceConstBeforeReturn(&fn, 0, 2)
				return fn
			},
		},
		{
			name: "wrong_failure_scalar_return",
			fn: func() ir.IRFunc {
				fn := inoutWriterHelperSummaryExactJSONCallerStackIRFunc(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "p25.json_parse_stringify.write_message_object",
						ArgSlots: 2,
						RetSlots: 3,
					},
				)
				replaceConstBeforeReturn(&fn, 1, 2)
				return fn
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok, err := InoutWriterHelperSummaryCallerFunctionFromStackIR(tc.fn()); err != nil ||
				ok {
				t.Fatalf(
					"InoutWriterHelperSummaryCallerFunctionFromStackIR ok=%v err=%v, "+
						"want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func TestVectorI32x4SliceSumLoopFromStackIRUsesSafeUnalignedTailAndScalarFallback(t *testing.T) {
	plan, ok, err := VectorI32x4SliceSumLoopPlanFromStackIR(sliceSumStackIRFunc(true))
	if err != nil {
		t.Fatalf("VectorI32x4SliceSumLoopPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("VectorI32x4SliceSumLoopPlanFromStackIR did not accept proof-tagged slice sum")
	}
	if plan.LaneCount != 4 || !plan.SafeUnaligned || plan.TailHandling != "scalar_tail" ||
		plan.ScalarFallback != "scalar-i32-slice-sum" {
		t.Fatalf("vector plan facts = %#v, want i32x4 safe unaligned scalar tail/fallback", plan)
	}
	if plan.NoAliasRequirement != "not_required_read_only_reduction" {
		t.Fatalf(
			"noalias requirement = %q, want read-only reduction exemption",
			plan.NoAliasRequirement,
		)
	}
	if plan.ProofID == "" {
		t.Fatalf("vector plan missing proof id: %#v", plan)
	}
	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func sum",
		"target:vector-i32x4-slice-sum-plan",
		"vector_zero_i32x4 defs:vsum",
		"vector_can_load_i32x4 defs:vcmp uses:local1,local3",
		"vector_load_i32x4_unaligned defs:vchunk uses:local0,local1,local3",
		"vector_add_i32x4 defs:vsum uses:vsum,vchunk",
		"vector_horizontal_add_i32x4 defs:local2 uses:vsum",
		"tail_scalar_i32_sum defs:local2 uses:local2,local0,local1,local3",
		"proof:while:",
		"return uses:local2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("vector machine slice-sum dump missing %q:\n%s", want, text)
		}
	}
	if _, ok, err := VectorI32x4SliceSumLoopPlanFromStackIR(sliceSumStackIRFunc(false)); err != nil ||
		ok {
		t.Fatalf(
			"checked/no-proof vector slice sum ok=%v err=%v, want fallback without error",
			ok,
			err,
		)
	}
}

func TestParallelMapReduceMainPlanFromStackIRAcceptsExactTaskHandlePair(t *testing.T) {
	fn := parallelMapReduceMainStackIRFunc()
	plan, ok, err := ParallelMapReduceMainPlanFromStackIR(fn)
	if err != nil {
		t.Fatalf("ParallelMapReduceMainPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ParallelMapReduceMainPlanFromStackIR did not accept exact benchmark main")
	}
	if plan.Function.Name != "p25.parallel_map_reduce.main" ||
		plan.Function.Target != "parallel-map-reduce-main" {
		t.Fatalf("parallel map/reduce machine identity = %#v", plan.Function)
	}
	if plan.ExpectedTotal != 42 || plan.SuccessReturn != 0 {
		t.Fatalf("parallel map/reduce returns = %#v, want total 42 success 0", plan)
	}
	if len(plan.Spawns) != 3 || len(plan.Joins) != 3 {
		t.Fatalf("parallel map/reduce calls = spawns %#v joins %#v, want 3/3", plan.Spawns, plan.Joins)
	}
	for i, want := range []struct {
		worker string
		handle int
		status int
	}{
		{worker: "left_worker", handle: 0, status: 1},
		{worker: "mid_worker", handle: 2, status: 3},
		{worker: "right_worker", handle: 4, status: 5},
	} {
		spawn := plan.Spawns[i]
		join := plan.Joins[i]
		if spawn.Worker != want.worker ||
			spawn.EntryID != parallelMapReduceEntryIDForTest(want.worker) ||
			spawn.HandleLocal != want.handle ||
			spawn.StatusLocal != want.status {
			t.Fatalf("spawn[%d] = %#v, want worker/local pair %#v", i, spawn, want)
		}
		if join.Worker != want.worker ||
			join.HandleLocal != want.handle ||
			join.StatusLocal != want.status {
			t.Fatalf("join[%d] = %#v, want worker/local pair %#v", i, join, want)
		}
	}
	if err := VerifyFunction(plan.Function); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"target:parallel-map-reduce-main",
		"call __tetra_task_spawn_i32 defs:left.handle,left.status",
		"ret_slots=2",
		"call __tetra_task_join_i32 defs:left.value uses:left.handle,left.status",
		"return uses:zero",
		"return uses:total",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("parallel map/reduce machine dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(plan.Function)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	if _, err := LinearScan(intervals, LinuxX64CallerSaved()); err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
}

func TestParallelMapReduceMainPlanFromStackIRRejectsNearMisses(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
	}{
		{
			name: "different_runtime_call_ret_slots_2",
			fn: func() ir.IRFunc {
				fn := parallelMapReduceMainStackIRFunc()
				fn.Instrs[1].Name = "__tetra_task_poll_i32"
				return fn
			},
		},
		{
			name: "spawn_ret_slots_3",
			fn: func() ir.IRFunc {
				fn := parallelMapReduceMainStackIRFunc()
				fn.Instrs[1].RetSlots = 3
				return fn
			},
		},
		{
			name: "missing_right_join",
			fn: func() ir.IRFunc {
				fn := parallelMapReduceMainStackIRFunc()
				fn.Instrs = append(fn.Instrs[:19], fn.Instrs[22:]...)
				return fn
			},
		},
		{
			name: "extra_spawn",
			fn: func() ir.IRFunc {
				fn := parallelMapReduceMainStackIRFunc()
				extra := []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: parallelMapReduceEntryIDForTest("left_worker")},
					{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2},
					{Kind: ir.IRStoreLocal, Local: 8},
					{Kind: ir.IRStoreLocal, Local: 7},
				}
				fn.LocalSlots = 9
				fn.Instrs = append(append([]ir.IRInstr{}, extra...), fn.Instrs...)
				return fn
			},
		},
		{
			name: "unrelated_multi_return_aggregate_helper",
			fn: func() ir.IRFunc {
				return ir.IRFunc{
					Name:        "aggregate_call_sample",
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						{Kind: ir.IRCall, Name: "slice_header_return", ArgSlots: 0, RetSlots: 2},
						{Kind: ir.IRReturn},
					},
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok, err := ParallelMapReduceMainPlanFromStackIR(tc.fn()); err != nil || ok {
				t.Fatalf(
					"ParallelMapReduceMainPlanFromStackIR near-miss ok=%v err=%v, want strict fallback",
					ok,
					err,
				)
			}
		})
	}
}

func TestActorPingPongRuntimeCallPlanFromStackIRAcceptsExactScalarShapes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fn        ir.IRFunc
		wantPath  string
		wantCalls []string
	}{
		{
			name:     "pong",
			fn:       actorPingPongPongStackIRFunc(),
			wantPath: "machine-ir-actor-ping-pong-pong",
			wantCalls: []string{
				"call __tetra_actor_recv",
				"call __tetra_actor_sender",
				"call __tetra_actor_send",
			},
		},
		{
			name:     "main",
			fn:       actorPingPongMainStackIRFunc(),
			wantPath: "machine-ir-actor-ping-pong-main",
			wantCalls: []string{
				"call __tetra_actor_spawn",
				"call __tetra_actor_send",
				"call __tetra_actor_recv",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, ok, err := ActorPingPongRuntimeCallPlanFromStackIRWithCallABI(
				tc.fn,
				SysVCallABIInfo(),
			)
			if err != nil {
				t.Fatalf("ActorPingPongRuntimeCallPlanFromStackIRWithCallABI: %v", err)
			}
			if !ok {
				t.Fatalf("ActorPingPongRuntimeCallPlanFromStackIRWithCallABI rejected %s", tc.name)
			}
			if plan.Path != tc.wantPath || plan.Function.Target != "actor-ping-pong-runtime-call" {
				t.Fatalf("actor ping-pong plan = %#v, want path %q", plan, tc.wantPath)
			}
			text := FormatFunction(plan.Function)
			for _, want := range append(tc.wantCalls,
				"abi:sysv",
				"return uses:ret0",
				"return uses:ret1",
			) {
				if !strings.Contains(text, want) {
					t.Fatalf("actor ping-pong machine dump missing %q:\n%s", want, text)
				}
			}
			if err := VerifyFunction(plan.Function); err != nil {
				t.Fatalf("VerifyFunction: %v", err)
			}
			intervals, err := BuildIntervals(plan.Function)
			if err != nil {
				t.Fatalf("BuildIntervals: %v", err)
			}
			if _, err := LinearScan(intervals, LinuxX64CallerSaved()); err != nil {
				t.Fatalf("LinearScan: %v", err)
			}
		})
	}
}

func TestActorPingPongRuntimeCallPlanFromStackIRRejectsNearMisses(t *testing.T) {
	insert := func(fn ir.IRFunc, at int, instr ir.IRInstr) ir.IRFunc {
		instrs := append([]ir.IRInstr(nil), fn.Instrs[:at]...)
		instrs = append(instrs, instr)
		instrs = append(instrs, fn.Instrs[at:]...)
		fn.Instrs = instrs
		return fn
	}
	for _, tc := range []struct {
		name string
		fn   func() ir.IRFunc
	}{
		{
			name: "pong_extra_runtime_call",
			fn: func() ir.IRFunc {
				return insert(
					actorPingPongPongStackIRFunc(),
					2,
					ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_poll", ArgSlots: 0, RetSlots: 2},
				)
			},
		},
		{
			name: "pong_typed_message_send",
			fn: func() ir.IRFunc {
				fn := actorPingPongPongStackIRFunc()
				fn.Instrs[8] = ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "__tetra_actor_send_msg",
					ArgSlots: 3,
					RetSlots: 1,
				}
				return fn
			},
		},
		{
			name: "pong_non_scalar_return",
			fn: func() ir.IRFunc {
				fn := actorPingPongPongStackIRFunc()
				fn.ReturnSlots = 2
				return fn
			},
		},
		{
			name: "main_recv_multi_slot",
			fn: func() ir.IRFunc {
				fn := actorPingPongMainStackIRFunc()
				fn.Instrs[8] = ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "__tetra_actor_recv_msg",
					ArgSlots: 0,
					RetSlots: 2,
				}
				return fn
			},
		},
		{
			name: "main_missing_success_branch",
			fn: func() ir.IRFunc {
				fn := actorPingPongMainStackIRFunc()
				fn.Instrs[12].Kind = ir.IRReturn
				return fn
			},
		},
		{
			name: "main_different_compare_literal",
			fn: func() ir.IRFunc {
				fn := actorPingPongMainStackIRFunc()
				fn.Instrs[10].Imm = 43
				return fn
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok, err := ActorPingPongRuntimeCallPlanFromStackIRWithCallABI(
				tc.fn(),
				SysVCallABIInfo(),
			); err != nil || ok {
				t.Fatalf(
					("ActorPingPongRuntimeCallPlanFromStackIRWithCallABI near-miss " +
						"ok=%v err=%v, want strict fallback"),
					ok,
					err,
				)
			}
		})
	}
}

func TestVectorU8x16CopyLoopFromStackIRRequiresRangeNoAliasSafeUnalignedTailAndFallback(
	t *testing.T,
) {
	plan, ok, err := VectorU8x16CopyLoopPlanFromStackIR(copyU8StackIRFunc(true))
	if err != nil {
		t.Fatalf("VectorU8x16CopyLoopPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("VectorU8x16CopyLoopPlanFromStackIR did not accept proof-tagged copy []u8")
	}
	if plan.LaneCount != 16 || !plan.SafeUnaligned {
		t.Fatalf(
			"copy vector lane/safe-unaligned = %d/%v, want 16/true",
			plan.LaneCount,
			plan.SafeUnaligned,
		)
	}
	if plan.TailHandling != "scalar_tail" || plan.ScalarFallback == "" {
		t.Fatalf(
			"copy vector tail/fallback = %q/%q, want scalar_tail plus fallback",
			plan.TailHandling,
			plan.ScalarFallback,
		)
	}
	if plan.NoAliasRequirement != "required_source_dest_disjoint_owned_copy_result" {
		t.Fatalf("copy vector noalias requirement = %q", plan.NoAliasRequirement)
	}
	if !strings.HasPrefix(plan.ProofID, "proof:copy-loop:") {
		t.Fatalf("copy vector proof id = %q, want copy-loop proof", plan.ProofID)
	}
	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func copy_u8 target:vector-u8x16-copy-plan",
		"vector_can_copy_u8x16",
		"vector_load_u8x16_unaligned",
		"vector_store_u8x16_unaligned",
		"tail_scalar_u8_copy",
		"safe unaligned u8x16 copy load/store",
		"source/dest disjoint owned copy result",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("copy vector machine dump missing %q:\n%s", want, text)
		}
	}
	if _, ok, err := VectorU8x16CopyLoopPlanFromStackIR(copyU8StackIRFunc(false)); err != nil ||
		ok {
		t.Fatalf(
			"checked/no-proof vector copy []u8 ok=%v err=%v, want fallback without error",
			ok,
			err,
		)
	}
}

func TestVectorI32x4MapAddConstFromStackIRRequiresRangeSafeUnalignedTailAndFallback(t *testing.T) {
	plan, ok, err := VectorI32x4MapAddConstPlanFromStackIR(mapAddI32StackIRFunc(true))
	if err != nil {
		t.Fatalf("VectorI32x4MapAddConstPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("VectorI32x4MapAddConstPlanFromStackIR did not accept proof-tagged map []i32")
	}
	if plan.LaneCount != 4 || !plan.SafeUnaligned {
		t.Fatalf(
			"map vector lane/safe-unaligned = %d/%v, want 4/true",
			plan.LaneCount,
			plan.SafeUnaligned,
		)
	}
	if plan.Addend != 1 {
		t.Fatalf("map vector addend = %d, want 1", plan.Addend)
	}
	if plan.TailHandling != "scalar_tail" || plan.ScalarFallback == "" {
		t.Fatalf(
			"map vector tail/fallback = %q/%q, want scalar_tail plus fallback",
			plan.TailHandling,
			plan.ScalarFallback,
		)
	}
	if plan.NoAliasRequirement != "not_required_single_mutable_slice_in_place" {
		t.Fatalf("map vector noalias requirement = %q", plan.NoAliasRequirement)
	}
	if !strings.HasPrefix(plan.ProofID, "proof:map-loop:") {
		t.Fatalf("map vector proof id = %q, want map-loop proof", plan.ProofID)
	}
	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func map_i32_add1 target:vector-i32x4-map-add-const-plan",
		"vector_splat_i32x4 defs:vadd",
		"vector_can_map_i32x4",
		"vector_load_i32x4_unaligned",
		"vector_add_i32x4",
		"vector_store_i32x4_unaligned",
		"tail_scalar_i32_map",
		"safe unaligned i32x4 map load/store",
		"single mutable slice in-place map",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("map vector machine dump missing %q:\n%s", want, text)
		}
	}
	if _, ok, err := VectorI32x4MapAddConstPlanFromStackIR(mapAddI32StackIRFunc(false)); err != nil ||
		ok {
		t.Fatalf(
			"checked/no-proof vector map []i32 ok=%v err=%v, want fallback without error",
			ok,
			err,
		)
	}
}

func TestVectorU8x16MemsetZeroHelperFromStackIRRequiresRangeSafeUnalignedTailAndFallback(
	t *testing.T,
) {
	plan, ok, err := VectorU8x16MemsetZeroPlanFromStackIR(memsetZeroU8StackIRFunc(true))
	if err != nil {
		t.Fatalf("VectorU8x16MemsetZeroPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf(
			"VectorU8x16MemsetZeroPlanFromStackIR did not accept proof-tagged memset_zero_u8 helper",
		)
	}
	if plan.LaneCount != 16 || !plan.SafeUnaligned {
		t.Fatalf(
			"memset vector lane/safe-unaligned = %d/%v, want 16/true",
			plan.LaneCount,
			plan.SafeUnaligned,
		)
	}
	if plan.FillValue != 0 {
		t.Fatalf("memset vector fill value = %d, want zero-fill helper", plan.FillValue)
	}
	if plan.TailHandling != "scalar_tail" || plan.ScalarFallback == "" {
		t.Fatalf(
			"memset vector tail/fallback = %q/%q, want scalar_tail plus fallback",
			plan.TailHandling,
			plan.ScalarFallback,
		)
	}
	if plan.NoAliasRequirement != "not_required_single_mutable_slice_in_place_zero_fill" {
		t.Fatalf("memset vector noalias requirement = %q", plan.NoAliasRequirement)
	}
	if !strings.HasPrefix(plan.ProofID, "proof:memset-loop:") {
		t.Fatalf("memset vector proof id = %q, want memset-loop proof", plan.ProofID)
	}
	text := FormatFunction(plan.Function)
	for _, want := range []string{
		"func memset_zero_u8 target:vector-u8x16-memset-zero-plan",
		"vector_zero_u8x16 defs:vzero",
		"vector_can_memset_u8x16",
		"vector_store_u8x16_unaligned",
		"tail_scalar_u8_memset",
		"safe unaligned u8x16 zero-fill store",
		"single mutable slice zero-fill helper",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("memset vector machine dump missing %q:\n%s", want, text)
		}
	}
	if _, ok, err := VectorU8x16MemsetZeroPlanFromStackIR(
		memsetZeroU8StackIRFunc(false),
	); err != nil ||
		ok {
		t.Fatalf(
			"checked/no-proof vector memset_zero_u8 ok=%v err=%v, want fallback without error",
			ok,
			err,
		)
	}
}

func TestScalarI32SliceSumLoopFromStackIRLowersProofTaggedConstantStride(t *testing.T) {
	mfn, ok, err := ScalarI32SliceSumLoopFunctionFromStackIR(sliceSumStrideStackIRFunc(true, 2))
	if err != nil {
		t.Fatalf("ScalarI32SliceSumLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf(
			"ScalarI32SliceSumLoopFunctionFromStackIR did not accept proof-tagged constant-stride slice sum",
		)
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func sum_stride",
		"params:local0,local1",
		"index_load defs:",
		"proof:while:",
		"add defs:local2",
		"add defs:local3 uses:local3,t2",
		"return uses:local2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine slice-stride-sum dump missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "inc defs:local3 uses:local3") {
		t.Fatalf("constant-stride slice sum should use explicit stride add, not inc:\n%s", text)
	}
	if !hasMachineImmDef(mfn, "t2", 2) {
		t.Fatalf("constant-stride slice sum missing step immediate 2 in machine IR: %+v", mfn)
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"slice stride sum loop should fit in linux-x64 caller-saved registers, spills=%v",
			alloc.Spills,
		)
	}
}

func TestScalarI32SliceSumLoopFromStackIRRejectsInvalidConstantStride(t *testing.T) {
	for _, step := range []int32{0, -1, 128} {
		if _, ok, err := ScalarI32SliceSumLoopFunctionFromStackIR(
			sliceSumStrideStackIRFunc(true, step),
		); err != nil ||
			ok {
			t.Fatalf(
				"ScalarI32SliceSumLoopFunctionFromStackIR step %d ok=%v err=%v, want fallback without error",
				step,
				ok,
				err,
			)
		}
	}
	if _, ok, err := ScalarI32SliceSumLoopFunctionFromStackIR(
		sliceSumStrideStackIRFunc(false, 2),
	); err != nil ||
		ok {
		t.Fatalf(
			"checked/no-proof slice stride sum ok=%v err=%v, want fallback without error",
			ok,
			err,
		)
	}
}

func TestScalarIntFunctionFromStackIRLowersNestedCallsWithABIClobbers(t *testing.T) {
	mfn, ok, err := ScalarIntFunctionFromStackIR(nestedCallStackIRFunc())
	if err != nil {
		t.Fatalf("ScalarIntFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntFunctionFromStackIR did not accept scalar nested calls")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func main",
		"call inc",
		"abi:sysv",
		"clobbers:rax,rcx,rdx,rsi,rdi,r8,r9,r10,r11",
		"return uses:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine call dump missing %q:\n%s", want, text)
		}
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if len(alloc.Spills) != 0 {
		t.Fatalf(
			"nested scalar calls should keep virtual values in the call-safe scratch model, spills=%v",
			alloc.Spills,
		)
	}
}

func TestScalarIntFunctionFromStackIRFallsBackForMultiSlotCallReturns(t *testing.T) {
	_, ok, err := ScalarIntFunctionFromStackIR(ir.IRFunc{
		Name:        "slice_return_caller",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRCall, Name: "slice_return", ArgSlots: 0, RetSlots: 2},
			{Kind: ir.IRReturn},
		},
	})
	if err != nil {
		t.Fatalf("ScalarIntFunctionFromStackIR: %v", err)
	}
	if ok {
		t.Fatalf(
			("multi-slot call returns must stay on the stack fallback until " +
				"slice/String representation is verified"),
		)
	}
}

func TestScalarIntCallLoopFunctionFromStackIRLowersCallWithABIClobbers(t *testing.T) {
	mfn, ok, err := ScalarIntCallLoopFunctionFromStackIR(sumCallLoopStackIRFunc())
	if err != nil {
		t.Fatalf("ScalarIntCallLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarIntCallLoopFunctionFromStackIR did not accept call loop")
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func sum_call",
		"call inc",
		"abi:sysv",
		"clobbers:rax,rcx,rdx,rsi,rdi,r8,r9,r10,r11",
		"add defs:local2",
		"return uses:local2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine call-loop dump missing %q:\n%s", want, text)
		}
	}
	live, err := AnalyzeLiveness(mfn)
	if err != nil {
		t.Fatalf("AnalyzeLiveness: %v", err)
	}
	loop := live.Blocks["label1"]
	for _, want := range []VReg{"local0", "local1", "local2"} {
		if !containsReg(loop.LiveIn, want) || !containsReg(loop.LiveOut, want) {
			t.Fatalf("call-loop liveness = %+v, want %s live-in and live-out", loop, want)
		}
	}
}

func TestScalarIntCallLoopFunctionFromStackIRLowersCompileTimeEqualityTail(t *testing.T) {
	mfn, ok, err := ScalarIntCallLoopFunctionFromStackIR(compileTimeBenchmarkMainStackIRFunc())
	if err != nil {
		t.Fatalf("ScalarIntCallLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf(
			"ScalarIntCallLoopFunctionFromStackIR did not accept compile_time equality-tail call loop",
		)
	}
	text := FormatFunction(mfn)
	for _, want := range []string{
		"func p25.compile_time.main target:scalar-int-call-loop",
		"call p25.compile_time.f2",
		"abi:sysv",
		"clobbers:rax,rcx,rdx,rsi,rdi,r8,r9,r10,r11",
		"loop bound constant",
		"total == 0",
		"return 1",
		"return 0",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("machine compile_time call-loop dump missing %q:\n%s", want, text)
		}
	}
	if err := VerifyFunction(mfn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	intervals, err := BuildIntervals(mfn)
	if err != nil {
		t.Fatalf("BuildIntervals: %v", err)
	}
	alloc, err := LinearScan(intervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan: %v", err)
	}
	if err := VerifyAllocation(mfn, alloc, LinuxX64CallerSaved(), len(alloc.Spills)); err != nil {
		t.Fatalf("VerifyAllocation: %v", err)
	}
}

func TestScalarIntCallLoopFunctionFromStackIRRejectsAlteredCompileTimeEqualityTail(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.IRFunc)
	}{
		{
			name: "altered_loop_bound",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[6].Imm = 199999
			},
		},
		{
			name: "altered_final_compare",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[22].Kind = ir.IRCmpGeI32
			},
		},
		{
			name: "altered_equal_return",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[24].Imm = 0
			},
		},
		{
			name: "altered_fallthrough_return",
			mutate: func(fn *ir.IRFunc) {
				fn.Instrs[27].Imm = 1
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := compileTimeBenchmarkMainStackIRFunc()
			tc.mutate(&fn)
			if _, ok, err := ScalarIntCallLoopFunctionFromStackIR(fn); err != nil || ok {
				t.Fatalf(
					"ScalarIntCallLoopFunctionFromStackIR ok=%v err=%v, want strict fallback without error",
					ok,
					err,
				)
			}
		})
	}
}

func TestRecursionBenchmarkFunctionFromStackIRLowersFibAndMain(t *testing.T) {
	fib, ok, err := RecursionFibFunctionFromStackIRWithCallABI(
		recursionFibStackIRFunc(),
		SysVCallABIInfo(),
	)
	if err != nil {
		t.Fatalf("RecursionFibFunctionFromStackIRWithCallABI: %v", err)
	}
	if !ok {
		t.Fatalf("RecursionFibFunctionFromStackIRWithCallABI did not accept exact fib")
	}
	fibText := FormatFunction(fib)
	for _, want := range []string{
		"func p25.recursion.fib target:recursion-fib",
		"cmp defs:",
		"call p25.recursion.fib",
		"abi:sysv",
		"clobbers:rax,rcx,rdx,rsi,rdi,r8,r9,r10,r11",
		"add defs:",
		"return uses:",
	} {
		if !strings.Contains(fibText, want) {
			t.Fatalf("machine recursion fib dump missing %q:\n%s", want, fibText)
		}
	}
	if err := VerifyFunction(fib); err != nil {
		t.Fatalf("VerifyFunction fib: %v", err)
	}
	fibIntervals, err := BuildIntervals(fib)
	if err != nil {
		t.Fatalf("BuildIntervals fib: %v", err)
	}
	fibAlloc, err := LinearScan(fibIntervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan fib: %v", err)
	}
	if err := VerifyAllocation(
		fib,
		fibAlloc,
		LinuxX64CallerSaved(),
		len(fibAlloc.Spills),
	); err != nil {
		t.Fatalf("VerifyAllocation fib: %v", err)
	}

	mainFn, ok, err := RecursionMainFunctionFromStackIRWithCallABI(
		recursionMainStackIRFunc(),
		SysVCallABIInfo(),
	)
	if err != nil {
		t.Fatalf("RecursionMainFunctionFromStackIRWithCallABI: %v", err)
	}
	if !ok {
		t.Fatalf("RecursionMainFunctionFromStackIRWithCallABI did not accept exact recursion main")
	}
	mainText := FormatFunction(mainFn)
	for _, want := range []string{
		"func p25.recursion.main target:recursion-main-loop",
		"mov defs:call_arg ; fib(10)",
		"call p25.recursion.fib",
		"cmp defs:",
		"return uses:ret0",
		"return uses:ret1",
	} {
		if !strings.Contains(mainText, want) {
			t.Fatalf("machine recursion main dump missing %q:\n%s", want, mainText)
		}
	}
	if err := VerifyFunction(mainFn); err != nil {
		t.Fatalf("VerifyFunction main: %v", err)
	}
	mainIntervals, err := BuildIntervals(mainFn)
	if err != nil {
		t.Fatalf("BuildIntervals main: %v", err)
	}
	mainAlloc, err := LinearScan(mainIntervals, LinuxX64CallerSaved())
	if err != nil {
		t.Fatalf("LinearScan main: %v", err)
	}
	if err := VerifyAllocation(
		mainFn,
		mainAlloc,
		LinuxX64CallerSaved(),
		len(mainAlloc.Spills),
	); err != nil {
		t.Fatalf("VerifyAllocation main: %v", err)
	}
}

func TestRecursionBenchmarkFunctionFromStackIRRejectsAlteredShapes(t *testing.T) {
	t.Run("altered_fib_base_case", func(t *testing.T) {
		fn := recursionFibStackIRFunc()
		fn.Instrs[1].Imm = 3
		if _, ok, err := RecursionFibFunctionFromStackIRWithCallABI(fn, SysVCallABIInfo()); err != nil ||
			ok {
			t.Fatalf(
				"RecursionFibFunctionFromStackIRWithCallABI ok=%v err=%v, want strict fallback without error",
				ok,
				err,
			)
		}
	})
	t.Run("altered_main_loop_bound", func(t *testing.T) {
		fn := recursionMainStackIRFunc()
		fn.Instrs[6].Imm = 41
		if _, ok, err := RecursionMainFunctionFromStackIRWithCallABI(fn, SysVCallABIInfo()); err != nil ||
			ok {
			t.Fatalf(
				"RecursionMainFunctionFromStackIRWithCallABI ok=%v err=%v, want strict fallback without error",
				ok,
				err,
			)
		}
	})
	t.Run("altered_main_success_value", func(t *testing.T) {
		fn := recursionMainStackIRFunc()
		fn.Instrs[21].Imm = 2199
		if _, ok, err := RecursionMainFunctionFromStackIRWithCallABI(fn, SysVCallABIInfo()); err != nil ||
			ok {
			t.Fatalf(
				"RecursionMainFunctionFromStackIRWithCallABI ok=%v err=%v, want strict fallback without error",
				ok,
				err,
			)
		}
	})
}

func hasMachineImmDef(fn Function, def VReg, imm int64) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Imm != imm {
				continue
			}
			for _, got := range instr.Defs {
				if got == def {
					return true
				}
			}
		}
	}
	return false
}

func sumNStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_n",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func sumStrideStackIRFunc(step int32) ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_stride",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: step},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func integerLoopsBenchmarkStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.integer_loops.main",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 200000},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRModI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGeI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func sumSquaresStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_squares",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func productStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "product_n",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func maxStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "max_n",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func sumAffineStackIRFunc(scale int32, bias int32) ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_affine",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: scale},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRConstI32, Imm: bias},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func countdownStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_countdown",
		ParamSlots:  1,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func nestedCallStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 40},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func sumCallLoopStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_call",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func compileTimeBenchmarkMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.compile_time.main",
		ExportName:  "main",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 200000},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCall, Name: "p25.compile_time.f2", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func recursionFibStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.recursion.fib",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRCall, Name: "p25.recursion.fib", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRCall, Name: "p25.recursion.fib", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func recursionMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.recursion.main",
		ExportName:  "main",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 40},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 10},
			{Kind: ir.IRCall, Name: "p25.recursion.fib", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 2200},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func sliceSumStackIRFunc(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadI32
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadI32Unchecked
		proofID = "proof:while:i:xs:1:1"
	}
	return ir.IRFunc{
		Name:        "sum",
		ParamSlots:  2,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func sliceSumMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.slice_sum.main",
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
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 4},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 5},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:i:xs:15:9"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 4},
			{Kind: ir.IRLabel, Label: 5},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 6},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func matrixMultiplyMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.matrix_multiply.main",
		ExportName:  "main",
		LocalSlots:  28,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRStackSliceI32, Local: 13, ArgSlots: 5, Imm: 9, Name: "a"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRStackSliceI32, Local: 18, ArgSlots: 5, Imm: 9, Name: "b"},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRStackSliceI32, Local: 23, ArgSlots: 5, Imm: 9, Name: "c"},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:a:10:9"},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:b:11:9"},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:c:12:9"},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 8},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRConstI32, Imm: 2000},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 9},
			{Kind: ir.IRLabel, Label: 4},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 5},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 10},
			{Kind: ir.IRLabel, Label: 6},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 7},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 12},
			{Kind: ir.IRLabel, Label: 8},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 9},
			{Kind: ir.IRLoadLocal, Local: 12},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:affine-const:row_k:a:24:38"},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:affine-const:k_col:b:24:55"},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 12},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 11},
			{Kind: ir.IRJmp, Label: 8},
			{Kind: ir.IRLabel, Label: 9},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 12},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:affine-const:row_col:c:26:19"},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 10},
			{Kind: ir.IRJmp, Label: 6},
			{Kind: ir.IRLabel, Label: 7},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 9},
			{Kind: ir.IRJmp, Label: 4},
			{Kind: ir.IRLabel, Label: 5},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:modulo:modulo_const:c:29:37"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 8},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 10},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 10},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func regionIslandAllocationMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.region_island_allocation.main",
		ExportName:  "main",
		LocalSlots:  5,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 16},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:allocation-zero:literal0:xs:9:13"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:allocation-zero:literal0:xs:10:35"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

const (
	p56AllocationLoopStoreProofID = "proof:allocation-zero:literal0:xs:9:9"
	p56AllocationLoopLoadProofID  = "proof:allocation-zero:literal0:xs:10:33"
)

func allocationLoopStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.allocation.main",
		ExportName:  "main",
		LocalSlots:  20,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1024},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 32},
			{Kind: ir.IRStackSliceI32, Local: 4, ArgSlots: 16, Imm: 32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRIndexStoreI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func allocationLoopP55ProofStackIRFunc() ir.IRFunc {
	fn := allocationLoopStackIRFunc()
	fn.Instrs[17].ProofID = p56AllocationLoopStoreProofID
	fn.Instrs[22].Kind = ir.IRIndexLoadI32Unchecked
	fn.Instrs[22].ProofID = p56AllocationLoopLoadProofID
	return fn
}

func sliceSumStrideStackIRFunc(proof bool, step int32) ir.IRFunc {
	fn := sliceSumStackIRFunc(proof)
	fn.Name = "sum_stride"
	fn.Instrs[17].Imm = step
	return fn
}

func copyU8StackIRFunc(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadU8
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadU8Unchecked
		proofID = "proof:copy-loop:u8:1:1"
	}
	return ir.IRFunc{
		Name:        "copy_u8",
		ParamSlots:  3,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func mapAddI32StackIRFunc(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadI32
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadI32Unchecked
		proofID = "proof:map-loop:i32:1:1"
	}
	return ir.IRFunc{
		Name:        "map_i32_add1",
		ParamSlots:  2,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRIndexStoreI32},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func hashTableLookupStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.hash_table.lookup",
		ParamSlots:  6,
		LocalSlots:  7,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:call-boundary:i:keys:7:16"},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:call-boundary:i:values:8:26"},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func hashTableLookupNoEarlyReturnStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.hash_table.lookup",
		ParamSlots:  6,
		LocalSlots:  8,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:call-boundary:i:keys:7:16"},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:call-boundary:i:values:8:26"},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRReturn},
		},
	}
}

func hashTableMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.hash_table.main",
		ExportName:  "main",
		LocalSlots:  265,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRStackSliceI32, Local: 9, ArgSlots: 128, Imm: 256, Name: "keys"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRStackSliceI32, Local: 137, ArgSlots: 128, Imm: 256, Name: "values"},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:keys:19:9"},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while:i:values:18:5"},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 8},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRCall, Name: "p25.hash_table.lookup", ArgSlots: 6, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 4},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func postgresqlFrameTypeAtStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.postgresql_single_multiple_update.frame_type_at",
		ParamSlots:  3,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRIndexLoadU8Unchecked, ProofID: "proof:helper-offset:offset:src:4:16"},
			{Kind: ir.IRReturn},
		},
	}
}

func postgresqlInoutWriterI32StackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.postgresql_single_multiple_update.write_i32_be_at",
		ParamSlots:  4,
		LocalSlots:  4,
		ReturnSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 16777216},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start:dst:15:5"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 65536},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start+1:dst:16:5"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start+2:dst:17:5"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start+3:dst:18:5"},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func postgresqlInoutWriterI16StackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.postgresql_single_multiple_update.write_i16_be_at",
		ParamSlots:  4,
		LocalSlots:  4,
		ReturnSlots: 3,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start:dst:23:5"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:start+1:dst:24:5"},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func inoutWriterHelperSummaryStackIRFunc(name string, storeCount int) ir.IRFunc {
	instrs := make([]ir.IRInstr, 0, storeCount*5+4)
	for i := 0; i < storeCount; i++ {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(i)},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(65 + i%26)},
			ir.IRInstr{
				Kind:    ir.IRIndexStoreU8,
				ProofID: "proof:helper-summary:const-index:dst",
			},
		)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(storeCount)},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{
		Name:        name,
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 3,
		Instrs:      instrs,
	}
}

func parallelMapReduceMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.parallel_map_reduce.main",
		LocalSlots:  7,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: parallelMapReduceEntryIDForTest("left_worker")},
			{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: parallelMapReduceEntryIDForTest("mid_worker")},
			{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: parallelMapReduceEntryIDForTest("right_worker")},
			{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCall, Name: "__tetra_task_join_i32", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRCall, Name: "__tetra_task_join_i32", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRCall, Name: "__tetra_task_join_i32", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRReturn},
		},
	}
}

func actorPingPongPongStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "pong",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRCall, Name: "__tetra_actor_recv", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 41},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRCall, Name: "__tetra_actor_sender", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRCall, Name: "__tetra_actor_send", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func actorPingPongMainStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: actorPingPongEntryIDForTest("pong")},
			{Kind: ir.IRCall, Name: "__tetra_actor_spawn", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 41},
			{Kind: ir.IRCall, Name: "__tetra_actor_send", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRCall, Name: "__tetra_actor_recv", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func actorPingPongEntryIDForTest(name string) int32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return int32(h.Sum32())
}

func parallelMapReduceEntryIDForTest(name string) int32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte("p25.parallel_map_reduce." + name))
	return int32(h.Sum32())
}

func countMachineOp(fn Function, op Opcode) int {
	count := 0
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == op {
				count++
			}
		}
	}
	return count
}

type helperSummaryCallerPlanCallWant struct {
	helperName string
	argSlots   int
	retSlots   int
	family     string
}

func assertHelperSummaryCallerPlanCalls(
	t *testing.T,
	plan InoutWriterHelperSummaryCallerPlan,
	want []helperSummaryCallerPlanCallWant,
) {
	t.Helper()
	if len(plan.AcceptedHelperCalls) != len(want) {
		t.Fatalf("accepted helper calls = %#v, want %d calls", plan.AcceptedHelperCalls, len(want))
	}
	for i, call := range plan.AcceptedHelperCalls {
		if call.HelperName != want[i].helperName ||
			call.ArgSlots != want[i].argSlots ||
			call.RetSlots != want[i].retSlots ||
			call.Family != want[i].family {
			t.Fatalf("accepted helper call[%d] = %#v, want %#v", i, call, want[i])
		}
	}
}

func assertHelperSummaryCallerMachineFacts(
	t *testing.T,
	fn Function,
	family string,
	want []helperSummaryCallerPlanCallWant,
) {
	t.Helper()
	text := FormatFunction(fn)
	common := []string{
		"target:inout-writer-helper-summary-caller",
		"proof_family=helper-summary",
		fmt.Sprintf("family=%s", family),
		fmt.Sprintf("call_count=%d", len(want)),
		"scalar success return 0",
		"scalar failure return 1",
	}
	for _, value := range common {
		if !strings.Contains(text, value) {
			t.Fatalf("helper-summary caller machine dump missing %q:\n%s", value, text)
		}
	}
	for i, call := range want {
		for _, value := range []string{
			fmt.Sprintf("helper call %d %s", i, call.helperName),
			fmt.Sprintf("arg_slots=%d", call.argSlots),
			fmt.Sprintf("ret_slots=%d", call.retSlots),
			fmt.Sprintf("family=%s", call.family),
		} {
			if !strings.Contains(text, value) {
				t.Fatalf("helper-summary caller machine dump missing %q:\n%s", value, text)
			}
		}
	}
	for _, forbidden := range []string{"abi:sysv", "abi:win64", "native/register"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("helper-summary caller machine dump contains forbidden claim %q:\n%s", forbidden, text)
		}
	}
}

func assertNoNativeHelperSummaryCallerClaim(
	t *testing.T,
	plan InoutWriterHelperSummaryCallerPlan,
) {
	t.Helper()
	typ := reflect.TypeOf(plan)
	for _, forbidden := range []string{"Function", "Target", "BackendPath"} {
		if _, ok := typ.FieldByName(forbidden); ok {
			t.Fatalf("caller plan must not record native/backend claim field %q: %#v", forbidden, plan)
		}
	}
}

func inoutWriterHelperSummaryExactJSONCallerStackIRFunc(calls ...ir.IRInstr) ir.IRFunc {
	return withHelperSummaryCallerScalarReturnShape(
		inoutWriterHelperSummaryJSONCallerStackIRFunc(calls...),
	)
}

func inoutWriterHelperSummaryExactHTTPCallerStackIRFunc(calls ...ir.IRInstr) ir.IRFunc {
	return withHelperSummaryCallerScalarReturnShape(
		inoutWriterHelperSummaryHTTPCallerStackIRFunc(calls...),
	)
}

func withHelperSummaryCallerScalarReturnShape(fn ir.IRFunc) ir.IRFunc {
	if len(fn.Instrs) >= 2 &&
		fn.Instrs[len(fn.Instrs)-2].Kind == ir.IRConstI32 &&
		fn.Instrs[len(fn.Instrs)-1].Kind == ir.IRReturn {
		fn.Instrs = fn.Instrs[:len(fn.Instrs)-2]
	}
	fn.Instrs = append(fn.Instrs,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRJmpIfZero, Label: 99},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRReturn},
		ir.IRInstr{Kind: ir.IRLabel, Label: 99},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return fn
}

func replaceConstBeforeReturn(fn *ir.IRFunc, old int32, newValue int32) {
	for i := 1; i < len(fn.Instrs); i++ {
		if fn.Instrs[i].Kind != ir.IRReturn ||
			fn.Instrs[i-1].Kind != ir.IRConstI32 ||
			fn.Instrs[i-1].Imm != old {
			continue
		}
		fn.Instrs[i-1].Imm = newValue
		return
	}
}

func inoutWriterHelperSummaryJSONCallerStackIRFunc(calls ...ir.IRInstr) ir.IRFunc {
	return inoutWriterHelperSummaryCallerStackIRFunc(
		"p25.json_parse_stringify.main",
		[]int{0, 1},
		calls...,
	)
}

func inoutWriterHelperSummaryHTTPCallerStackIRFunc(calls ...ir.IRInstr) ir.IRFunc {
	return inoutWriterHelperSummaryCallerStackIRFunc(
		"p25.http_plaintext_json.main",
		[]int{0, 1, 2, 3},
		calls...,
	)
}

func inoutWriterHelperSummaryCallerStackIRFunc(
	name string,
	callLocals []int,
	calls ...ir.IRInstr,
) ir.IRFunc {
	instrs := make([]ir.IRInstr, 0, len(calls)*6+2)
	for i, call := range calls {
		base := 0
		if len(callLocals) >= (i+1)*2 {
			base = callLocals[i*2]
		}
		length := base + 1
		if len(callLocals) >= (i+1)*2 {
			length = callLocals[i*2+1]
		}
		instrs = append(instrs, inoutWriterHelperSummaryCallerCallInstrs(base, length, call)...)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{
		Name:        name,
		ParamSlots:  0,
		LocalSlots:  5,
		ReturnSlots: 1,
		Instrs:      instrs,
	}
}

func inoutWriterHelperSummaryCallerCallInstrs(
	baseLocal int,
	lenLocal int,
	call ir.IRInstr,
) []ir.IRInstr {
	return []ir.IRInstr{
		{Kind: ir.IRLoadLocal, Local: baseLocal},
		{Kind: ir.IRLoadLocal, Local: lenLocal},
		call,
		{Kind: ir.IRStoreLocal, Local: lenLocal},
		{Kind: ir.IRStoreLocal, Local: baseLocal},
		{Kind: ir.IRStoreLocal, Local: 4},
	}
}

func memsetZeroU8StackIRFunc(proof bool) ir.IRFunc {
	proofID := ""
	if proof {
		proofID = "proof:memset-loop:u8:zero:1:1"
	}
	return ir.IRFunc{
		Name:        "memset_zero_u8",
		ParamSlots:  2,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexStoreU8, ProofID: proofID},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}
