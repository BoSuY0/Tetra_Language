package ssair

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/plir"
)

func TestVerifyFunctionRejectsMalformedSSA(t *testing.T) {
	fn := Function{
		Name:       "bad",
		ReturnType: TypeI32,
		Values: []Value{
			{ID: "x", Type: TypeI32},
		},
		Blocks: []Block{{
			ID:     "entry",
			Entry:  true,
			Instrs: []Instr{{ID: "add0", Kind: OpAddI32, Result: "sum", Args: []ValueID{"x", "missing"}}},
			Term:   Terminator{Kind: TermReturn, Value: "sum"},
		}},
	}
	err := VerifyFunction(fn)
	if err == nil || !strings.Contains(err.Error(), "unknown value") {
		t.Fatalf("VerifyFunction error = %v, want unknown value rejection", err)
	}
}

func TestVerifyFunctionRequiresCallEffectTokens(t *testing.T) {
	fn := Function{
		Name:       "bad_call",
		ReturnType: TypeI32,
		Values: []Value{
			{ID: "arg", Type: TypeI32},
			{ID: "ret", Type: TypeI32},
		},
		Blocks: []Block{{
			ID:     "entry",
			Entry:  true,
			Instrs: []Instr{{ID: "call0", Kind: OpCall, Result: "ret", Args: []ValueID{"arg"}, Call: "callee"}},
			Term:   Terminator{Kind: TermReturn, Value: "ret"},
		}},
	}
	err := VerifyFunction(fn)
	if err == nil || !strings.Contains(err.Error(), "call effect tokens") {
		t.Fatalf("VerifyFunction error = %v, want call effect token rejection", err)
	}
}

func TestFromStackIRScalarFunctionProducesTypedSSA(t *testing.T) {
	fn, ok, err := FromStackIRFunction(ir.IRFunc{
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
	})
	if err != nil {
		t.Fatalf("FromStackIRFunction: %v", err)
	}
	if !ok {
		t.Fatal("scalar add should be accepted by SSA bridge")
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	if fn.ReturnType != TypeI32 || len(fn.Blocks) != 1 || fn.Blocks[0].Term.Kind != TermReturn {
		t.Fatalf("scalar SSA shape = %+v", fn)
	}
	if !hasInstrKind(fn, OpAddI32) {
		t.Fatalf("scalar SSA missing add op: %+v", fn)
	}
}

func TestFromStackIRScalarLoopUsesBlockParams(t *testing.T) {
	fn, ok, err := FromStackIRFunction(sumNStackIRFunc())
	if err != nil {
		t.Fatalf("FromStackIRFunction: %v", err)
	}
	if !ok {
		t.Fatal("scalar loop should be accepted by SSA bridge")
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	loop := blockByID(fn, "loop")
	if len(loop.Params) < 2 {
		t.Fatalf("loop params = %+v, want index/total phi-style params", loop.Params)
	}
	if !hasTermTarget(fn, "loop") {
		t.Fatalf("SSA loop does not branch back to loop: %+v", fn)
	}
}

func TestFromStackIRScalarConstantStrideLoopUsesBlockParamsAndStep(t *testing.T) {
	fn, ok, err := FromStackIRFunction(sumStrideStackIRFunc(2))
	if err != nil {
		t.Fatalf("FromStackIRFunction: %v", err)
	}
	if !ok {
		t.Fatal("scalar constant-stride loop should be accepted by SSA bridge")
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	loop := blockByID(fn, "loop")
	if len(loop.Params) < 2 {
		t.Fatalf("loop params = %+v, want index/total phi-style params", loop.Params)
	}
	if !hasConstValue(fn, "step", 2) {
		t.Fatalf("constant-stride SSA missing step const: %+v", fn)
	}
	if !hasTermTarget(fn, "loop") {
		t.Fatalf("SSA constant-stride loop does not branch back to loop: %+v", fn)
	}
}

func TestFromStackIRScalarConstantStrideLoopRejectsInvalidStep(t *testing.T) {
	for _, step := range []int32{0, -1, 128} {
		if _, ok, err := FromStackIRFunction(sumStrideStackIRFunc(step)); err != nil || ok {
			t.Fatalf("FromStackIRFunction step %d ok=%v err=%v, want fallback without error", step, ok, err)
		}
	}
}

func TestFromStackIRScalarSumSquaresLoopUsesBlockParamsAndMul(t *testing.T) {
	fn, ok, err := FromStackIRFunction(sumSquaresStackIRFunc())
	if err != nil {
		t.Fatalf("FromStackIRFunction: %v", err)
	}
	if !ok {
		t.Fatal("scalar sum-squares loop should be accepted by SSA bridge")
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	loop := blockByID(fn, "loop")
	if len(loop.Params) < 2 {
		t.Fatalf("loop params = %+v, want index/total phi-style params", loop.Params)
	}
	if !hasInstrKind(fn, OpMulI32) {
		t.Fatalf("sum-squares SSA missing mul op: %+v", fn)
	}
	if !hasTermTarget(fn, "loop") {
		t.Fatalf("SSA sum-squares loop does not branch back to loop: %+v", fn)
	}
}

func TestFromStackIRScalarProductLoopUsesBlockParamsAndMul(t *testing.T) {
	fn, ok, err := FromStackIRFunction(productStackIRFunc())
	if err != nil {
		t.Fatalf("FromStackIRFunction: %v", err)
	}
	if !ok {
		t.Fatal("scalar product loop should be accepted by SSA bridge")
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	loop := blockByID(fn, "loop")
	if len(loop.Params) < 2 {
		t.Fatalf("loop params = %+v, want index/product phi-style params", loop.Params)
	}
	if !hasInstrKind(fn, OpMulI32) || !hasInstrKind(fn, OpAddI32) {
		t.Fatalf("product SSA missing mul/add ops: %+v", fn)
	}
	if !hasTermTarget(fn, "loop") {
		t.Fatalf("SSA product loop does not branch back to loop: %+v", fn)
	}
}

func TestFromStackIRScalarMaxLoopUsesBranchyBlockParams(t *testing.T) {
	fn, ok, err := FromStackIRFunction(maxStackIRFunc())
	if err != nil {
		t.Fatalf("FromStackIRFunction: %v", err)
	}
	if !ok {
		t.Fatal("scalar max loop should be accepted by SSA bridge")
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	loop := blockByID(fn, "loop")
	if len(loop.Params) < 2 {
		t.Fatalf("loop params = %+v, want index/max phi-style params", loop.Params)
	}
	body := blockByID(fn, "body")
	if len(body.Params) < 2 || body.Term.Kind != TermCondBr || body.Term.IfTrue != "update" || body.Term.IfFalse != "keep" {
		t.Fatalf("body block = %+v, want conditional update/keep branch with params", body)
	}
	if !hasInstrKind(fn, OpCmpGtI32) {
		t.Fatalf("max SSA missing cmp_gt op: %+v", fn)
	}
	if !hasTermTarget(fn, "loop") {
		t.Fatalf("SSA max loop does not branch back to loop: %+v", fn)
	}
}

func TestFromStackIRScalarAffineLoopUsesBlockParamsAndScaleBias(t *testing.T) {
	fn, ok, err := FromStackIRFunction(sumAffineStackIRFunc(2, 1))
	if err != nil {
		t.Fatalf("FromStackIRFunction: %v", err)
	}
	if !ok {
		t.Fatal("scalar affine loop should be accepted by SSA bridge")
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	loop := blockByID(fn, "loop")
	if len(loop.Params) < 2 {
		t.Fatalf("loop params = %+v, want index/total phi-style params", loop.Params)
	}
	if !hasInstrKind(fn, OpMulI32) || !hasInstrKind(fn, OpAddI32) {
		t.Fatalf("affine SSA missing mul/add ops: %+v", fn)
	}
	if !hasConstValue(fn, "scale", 2) {
		t.Fatalf("affine SSA missing scale const: %+v", fn)
	}
	if !hasConstValue(fn, "bias", 1) {
		t.Fatalf("affine SSA missing bias const: %+v", fn)
	}
	if !hasTermTarget(fn, "loop") {
		t.Fatalf("SSA affine loop does not branch back to loop: %+v", fn)
	}
}

func TestFromStackIRScalarAffineLoopRejectsInvalidConstants(t *testing.T) {
	for _, tc := range []struct {
		scale int32
		bias  int32
	}{
		{0, 1},
		{2, 0},
		{128, 1},
		{2, 128},
	} {
		if _, ok, err := FromStackIRFunction(sumAffineStackIRFunc(tc.scale, tc.bias)); err != nil || ok {
			t.Fatalf("FromStackIRFunction scale=%d bias=%d ok=%v err=%v, want fallback without error", tc.scale, tc.bias, ok, err)
		}
	}
}

func TestFromStackIRScalarCountdownLoopUsesBlockParamsAndSub(t *testing.T) {
	fn, ok, err := FromStackIRFunction(countdownStackIRFunc())
	if err != nil {
		t.Fatalf("FromStackIRFunction: %v", err)
	}
	if !ok {
		t.Fatal("scalar countdown loop should be accepted by SSA bridge")
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	loop := blockByID(fn, "loop")
	if len(loop.Params) < 2 {
		t.Fatalf("loop params = %+v, want countdown/total phi-style params", loop.Params)
	}
	if !hasInstrKind(fn, OpSubI32) {
		t.Fatalf("countdown SSA missing sub op: %+v", fn)
	}
	if !hasTermTarget(fn, "loop") {
		t.Fatalf("SSA countdown loop does not branch back to loop: %+v", fn)
	}
}

func TestFromStackIRSliceSumCarriesMemoryEffectAndProof(t *testing.T) {
	fn, ok, err := FromStackIRFunction(sliceSumStackIRFunc(true))
	if err != nil {
		t.Fatalf("FromStackIRFunction: %v", err)
	}
	if !ok {
		t.Fatal("proof-tagged slice sum should be accepted by SSA bridge")
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	load := instrByKind(fn, OpIndexLoadI32)
	if load.ID == "" || load.ProofID == "" || load.EffectIn == "" || load.EffectOut == "" {
		t.Fatalf("slice sum SSA load = %+v, want proof id and memory effect tokens", load)
	}
}

func TestFromStackIRSliceSumConstantStrideCarriesMemoryEffectProofAndStep(t *testing.T) {
	fn, ok, err := FromStackIRFunction(sliceSumStrideStackIRFunc(true, 2))
	if err != nil {
		t.Fatalf("FromStackIRFunction: %v", err)
	}
	if !ok {
		t.Fatal("proof-tagged constant-stride slice sum should be accepted by SSA bridge")
	}
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	load := instrByKind(fn, OpIndexLoadI32)
	if load.ID == "" || load.ProofID == "" || load.EffectIn == "" || load.EffectOut == "" {
		t.Fatalf("slice stride SSA load = %+v, want proof id and memory effect tokens", load)
	}
	if !hasConstValue(fn, "step", 2) {
		t.Fatalf("slice stride SSA missing step const: %+v", fn)
	}
}

func TestFromStackIRSliceSumConstantStrideRejectsInvalidStep(t *testing.T) {
	for _, step := range []int32{0, -1, 128} {
		if _, ok, err := FromStackIRFunction(sliceSumStrideStackIRFunc(true, step)); err != nil || ok {
			t.Fatalf("FromStackIRFunction step %d ok=%v err=%v, want fallback without error", step, ok, err)
		}
	}
	if _, ok, err := FromStackIRFunction(sliceSumStrideStackIRFunc(false, 2)); err != nil || ok {
		t.Fatalf("checked/no-proof slice stride ok=%v err=%v, want fallback without error", ok, err)
	}
}

func TestFromPLIRProgramChainsCallEffects(t *testing.T) {
	prog, err := FromPLIR(&plir.Program{Funcs: []plir.Function{{
		Name: "plir_call",
		Values: []plir.Value{
			{ID: "arg", Type: "Int"},
			{ID: "ret", Type: "Int"},
		},
		Ops: []plir.Operation{
			{ID: "op0", Kind: plir.OpCall, Inputs: []string{"arg"}, Outputs: []string{"ret"}, Note: "callee"},
			{ID: "op1", Kind: plir.OpReturn, Inputs: []string{"ret"}},
		},
	}}})
	if err != nil {
		t.Fatalf("FromPLIR: %v", err)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("SSA funcs = %d, want 1", len(prog.Funcs))
	}
	fn := prog.Funcs[0]
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	call := instrByKind(fn, OpCall)
	if call.EffectIn == "" || call.EffectOut == "" {
		t.Fatalf("PLIR call SSA = %+v, want chained effect tokens", call)
	}
}

func TestFromPLIRProgramLowersSliceIndexLoadWithMemoryEffect(t *testing.T) {
	prog, err := FromPLIR(&plir.Program{Funcs: []plir.Function{{
		Name: "plir_slice",
		Values: []plir.Value{
			{ID: "xs", Type: "[]i32"},
			{ID: "len", Type: "Int"},
			{ID: "i", Type: "Int"},
			{ID: "elem", Type: "Int"},
		},
		Ops: []plir.Operation{
			{ID: "op0", Kind: plir.OpIndexLoad, Inputs: []string{"xs", "len", "i"}, Outputs: []string{"elem"}, Note: "proof:while:test"},
			{ID: "op1", Kind: plir.OpReturn, Inputs: []string{"elem"}},
		},
	}}})
	if err != nil {
		t.Fatalf("FromPLIR: %v", err)
	}
	fn := prog.Funcs[0]
	if err := VerifyFunction(fn); err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
	load := instrByKind(fn, OpIndexLoadI32)
	if load.ID == "" || load.EffectIn == "" || load.EffectOut == "" {
		t.Fatalf("PLIR index load SSA = %+v, want memory effect chain", load)
	}
}

func hasInstrKind(fn Function, kind OpKind) bool {
	return instrByKind(fn, kind).ID != ""
}

func instrByKind(fn Function, kind OpKind) Instr {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Kind == kind {
				return instr
			}
		}
	}
	return Instr{}
}

func blockByID(fn Function, id string) Block {
	for _, block := range fn.Blocks {
		if block.ID == id {
			return block
		}
	}
	return Block{}
}

func hasTermTarget(fn Function, target string) bool {
	for _, block := range fn.Blocks {
		if block.Term.Target == target || block.Term.IfTrue == target || block.Term.IfFalse == target {
			return true
		}
	}
	return false
}

func hasConstValue(fn Function, result ValueID, imm int32) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Kind == OpConstI32 && instr.Result == result && instr.Imm == imm {
				return true
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

func sliceSumStackIRFunc(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadI32
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadI32Unchecked
		proofID = "proof:while:test"
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
