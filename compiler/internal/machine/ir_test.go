package machine

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

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
		t.Fatalf("short interval c should stay in a register: assignments=%v spills=%v", tight.Assignments, tight.Spills)
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
	if err := VerifyFunction(fn); err == nil || !strings.Contains(err.Error(), "missing terminator") {
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
	if err := VerifyFunction(fn); err == nil || !strings.Contains(err.Error(), "unknown branch target") {
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
	if err := VerifyFunction(fn); err == nil || !strings.Contains(err.Error(), "missing clobber metadata") {
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
				{Op: OpCall, Call: "callee", ABI: "sysv", Clobbers: LinuxX64CallerSaved(), Uses: []VReg{"loaded"}, Defs: []VReg{"ret"}},
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
	if err := VerifyAllocation(fn, badPhys, []PhysReg{"r1"}, 0); err == nil || !strings.Contains(err.Error(), "invalid physreg") {
		t.Fatalf("VerifyAllocation bad physreg error = %v, want invalid physreg", err)
	}
	badSpill := Allocation{Spills: map[VReg]int{"a": 1}}
	if err := VerifyAllocation(fn, badSpill, []PhysReg{"r1"}, 1); err == nil || !strings.Contains(err.Error(), "spill slot") {
		t.Fatalf("VerifyAllocation bad spill error = %v, want spill slot bounds", err)
	}
}

func TestFormatProgramProvidesStableMachineIRDump(t *testing.T) {
	text := FormatProgram(Program{Functions: []Function{SumToLoopFunction()}})
	for _, want := range []string{"program machine_ir", "func sum_to target:linux-x64", "loop:", "return uses:total"} {
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
		t.Fatalf("sum_n loop should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
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
		t.Fatalf("constant-stride loop should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
	}
}

func TestScalarIntLoopFunctionFromStackIRRejectsInvalidConstantStrideLoop(t *testing.T) {
	for _, step := range []int32{0, -1, 128} {
		if _, ok, err := ScalarIntLoopFunctionFromStackIR(sumStrideStackIRFunc(step)); err != nil || ok {
			t.Fatalf("ScalarIntLoopFunctionFromStackIR step %d ok=%v err=%v, want fallback without error", step, ok, err)
		}
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
		t.Fatalf("sum_squares loop should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
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
		t.Fatalf("product_n loop should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
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
		t.Fatalf("max_n loop should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
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
		t.Fatalf("sum_affine loop should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
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
		if _, ok, err := ScalarIntAffineLoopFunctionFromStackIR(sumAffineStackIRFunc(tc.scale, tc.bias)); err != nil || ok {
			t.Fatalf("ScalarIntAffineLoopFunctionFromStackIR scale=%d bias=%d ok=%v err=%v, want fallback without error", tc.scale, tc.bias, ok, err)
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
		t.Fatalf("sum_countdown loop should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
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
		t.Fatalf("slice sum loop should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
	}
	if _, ok, err := ScalarI32SliceSumLoopFunctionFromStackIR(sliceSumStackIRFunc(false)); err != nil || ok {
		t.Fatalf("checked/no-proof slice sum ok=%v err=%v, want fallback without error", ok, err)
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
	if plan.LaneCount != 4 || !plan.SafeUnaligned || plan.TailHandling != "scalar_tail" || plan.ScalarFallback != "scalar-i32-slice-sum" {
		t.Fatalf("vector plan facts = %#v, want i32x4 safe unaligned scalar tail/fallback", plan)
	}
	if plan.NoAliasRequirement != "not_required_read_only_reduction" {
		t.Fatalf("noalias requirement = %q, want read-only reduction exemption", plan.NoAliasRequirement)
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
	if _, ok, err := VectorI32x4SliceSumLoopPlanFromStackIR(sliceSumStackIRFunc(false)); err != nil || ok {
		t.Fatalf("checked/no-proof vector slice sum ok=%v err=%v, want fallback without error", ok, err)
	}
}

func TestVectorU8x16CopyLoopFromStackIRRequiresRangeNoAliasSafeUnalignedTailAndFallback(t *testing.T) {
	plan, ok, err := VectorU8x16CopyLoopPlanFromStackIR(copyU8StackIRFunc(true))
	if err != nil {
		t.Fatalf("VectorU8x16CopyLoopPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("VectorU8x16CopyLoopPlanFromStackIR did not accept proof-tagged copy []u8")
	}
	if plan.LaneCount != 16 || !plan.SafeUnaligned {
		t.Fatalf("copy vector lane/safe-unaligned = %d/%v, want 16/true", plan.LaneCount, plan.SafeUnaligned)
	}
	if plan.TailHandling != "scalar_tail" || plan.ScalarFallback == "" {
		t.Fatalf("copy vector tail/fallback = %q/%q, want scalar_tail plus fallback", plan.TailHandling, plan.ScalarFallback)
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
	if _, ok, err := VectorU8x16CopyLoopPlanFromStackIR(copyU8StackIRFunc(false)); err != nil || ok {
		t.Fatalf("checked/no-proof vector copy []u8 ok=%v err=%v, want fallback without error", ok, err)
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
		t.Fatalf("map vector lane/safe-unaligned = %d/%v, want 4/true", plan.LaneCount, plan.SafeUnaligned)
	}
	if plan.Addend != 1 {
		t.Fatalf("map vector addend = %d, want 1", plan.Addend)
	}
	if plan.TailHandling != "scalar_tail" || plan.ScalarFallback == "" {
		t.Fatalf("map vector tail/fallback = %q/%q, want scalar_tail plus fallback", plan.TailHandling, plan.ScalarFallback)
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
	if _, ok, err := VectorI32x4MapAddConstPlanFromStackIR(mapAddI32StackIRFunc(false)); err != nil || ok {
		t.Fatalf("checked/no-proof vector map []i32 ok=%v err=%v, want fallback without error", ok, err)
	}
}

func TestVectorU8x16MemsetZeroHelperFromStackIRRequiresRangeSafeUnalignedTailAndFallback(t *testing.T) {
	plan, ok, err := VectorU8x16MemsetZeroPlanFromStackIR(memsetZeroU8StackIRFunc(true))
	if err != nil {
		t.Fatalf("VectorU8x16MemsetZeroPlanFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("VectorU8x16MemsetZeroPlanFromStackIR did not accept proof-tagged memset_zero_u8 helper")
	}
	if plan.LaneCount != 16 || !plan.SafeUnaligned {
		t.Fatalf("memset vector lane/safe-unaligned = %d/%v, want 16/true", plan.LaneCount, plan.SafeUnaligned)
	}
	if plan.FillValue != 0 {
		t.Fatalf("memset vector fill value = %d, want zero-fill helper", plan.FillValue)
	}
	if plan.TailHandling != "scalar_tail" || plan.ScalarFallback == "" {
		t.Fatalf("memset vector tail/fallback = %q/%q, want scalar_tail plus fallback", plan.TailHandling, plan.ScalarFallback)
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
	if _, ok, err := VectorU8x16MemsetZeroPlanFromStackIR(memsetZeroU8StackIRFunc(false)); err != nil || ok {
		t.Fatalf("checked/no-proof vector memset_zero_u8 ok=%v err=%v, want fallback without error", ok, err)
	}
}

func TestScalarI32SliceSumLoopFromStackIRLowersProofTaggedConstantStride(t *testing.T) {
	mfn, ok, err := ScalarI32SliceSumLoopFunctionFromStackIR(sliceSumStrideStackIRFunc(true, 2))
	if err != nil {
		t.Fatalf("ScalarI32SliceSumLoopFunctionFromStackIR: %v", err)
	}
	if !ok {
		t.Fatalf("ScalarI32SliceSumLoopFunctionFromStackIR did not accept proof-tagged constant-stride slice sum")
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
		t.Fatalf("slice stride sum loop should fit in linux-x64 caller-saved registers, spills=%v", alloc.Spills)
	}
}

func TestScalarI32SliceSumLoopFromStackIRRejectsInvalidConstantStride(t *testing.T) {
	for _, step := range []int32{0, -1, 128} {
		if _, ok, err := ScalarI32SliceSumLoopFunctionFromStackIR(sliceSumStrideStackIRFunc(true, step)); err != nil || ok {
			t.Fatalf("ScalarI32SliceSumLoopFunctionFromStackIR step %d ok=%v err=%v, want fallback without error", step, ok, err)
		}
	}
	if _, ok, err := ScalarI32SliceSumLoopFunctionFromStackIR(sliceSumStrideStackIRFunc(false, 2)); err != nil || ok {
		t.Fatalf("checked/no-proof slice stride sum ok=%v err=%v, want fallback without error", ok, err)
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
		t.Fatalf("nested scalar calls should keep virtual values in the call-safe scratch model, spills=%v", alloc.Spills)
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
		t.Fatalf("multi-slot call returns must stay on the stack fallback until slice/String representation is verified")
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
