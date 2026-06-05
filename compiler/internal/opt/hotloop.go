package opt

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
	"tetra_language/compiler/internal/ssair"
)

type HotLoopShapeReport struct {
	SchemaVersion string            `json:"schema_version"`
	Rows          []HotLoopShapeRow `json:"rows"`
	NonClaims     []string          `json:"non_claims"`
}

type HotLoopShapeRow struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	RegisterPath  bool     `json:"register_path"`
	SSAVerified   bool     `json:"ssa_verified"`
	MachinePath   string   `json:"machine_path"`
	MachineTarget string   `json:"machine_target,omitempty"`
	SpillFree     bool     `json:"spill_free"`
	StackChurnOps int      `json:"stack_churn_ops"`
	RequiredOps   []string `json:"required_ops,omitempty"`
	ProofID       string   `json:"proof_id,omitempty"`
	CallABI       string   `json:"call_abi,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	Evidence      string   `json:"evidence"`
	Boundary      string   `json:"boundary"`
}

type hotLoopMachineLowerer func(ir.IRFunc) (machine.Function, bool, error)

func CoreHotLoopShapeEvidence() (HotLoopShapeReport, error) {
	rows := []HotLoopShapeRow{}
	scalar, err := hotLoopRegisterRow(
		"scalar-sum-loop",
		"scalar sum_n loop",
		hotLoopScalarSumFunc(),
		"machine-ir-loop",
		machine.ScalarIntLoopFunctionFromStackIR,
		"compiler/internal/machine/scalar_loop.go; compiler/internal/machine/ir_test.go::TestScalarIntLoopFunctionFromStackIRLowersSumNLoop",
		"canonical scalar i32 sum loop only; this is shape evidence, not throughput parity",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, scalar)

	stride, err := hotLoopRegisterRow(
		"scalar-stride-sum-loop",
		"scalar constant-stride sum loop",
		hotLoopScalarStrideFunc(),
		"machine-ir-stride-loop",
		machine.ScalarIntLoopFunctionFromStackIR,
		"compiler/internal/machine/scalar_loop.go; compiler/internal/machine/ir_test.go::TestScalarIntLoopFunctionFromStackIRLowersConstantStrideLoop",
		"canonical scalar i32 sum loop with positive constant stride 2..127 only; this is shape evidence, not throughput parity",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, stride)

	squares, err := hotLoopRegisterRow(
		"scalar-sum-squares-loop",
		"scalar sum_squares loop",
		hotLoopSumSquaresFunc(),
		"machine-ir-sum-squares-loop",
		machine.ScalarIntSumSquaresLoopFunctionFromStackIR,
		"compiler/internal/machine/scalar_sum_squares_loop.go; compiler/internal/machine/ir_test.go::TestScalarIntSumSquaresLoopFunctionFromStackIRLowersMulLoop",
		"canonical scalar i32 sum-of-squares loop only; this is shape evidence, not throughput parity",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, squares)

	product, err := hotLoopRegisterRow(
		"scalar-product-loop",
		"scalar product reduction loop",
		hotLoopProductFunc(),
		"machine-ir-product-loop",
		machine.ScalarIntProductLoopFunctionFromStackIR,
		"compiler/internal/machine/scalar_product_loop.go; compiler/internal/machine/ir_test.go::TestScalarIntProductLoopFunctionFromStackIRLowersProductReductionLoop",
		"canonical scalar i32 product reduction loop with product *= index + 1 only; this is shape evidence, not throughput or overflow-safety parity",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, product)

	max, err := hotLoopRegisterRow(
		"scalar-max-loop",
		"scalar max reduction loop",
		hotLoopMaxFunc(),
		"machine-ir-max-loop",
		machine.ScalarIntMaxLoopFunctionFromStackIR,
		"compiler/internal/machine/scalar_max_loop.go; compiler/internal/machine/ir_test.go::TestScalarIntMaxLoopFunctionFromStackIRLowersBranchyMaxReductionLoop",
		"canonical scalar i32 max reduction loop with branchy max update only; this is shape evidence, not throughput, general min/max, or overflow-safety parity",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, max)

	affine, err := hotLoopRegisterRow(
		"scalar-affine-sum-loop",
		"scalar affine sum loop",
		hotLoopScalarAffineFunc(),
		"machine-ir-affine-loop",
		machine.ScalarIntAffineLoopFunctionFromStackIR,
		"compiler/internal/machine/scalar_affine_loop.go; compiler/internal/machine/ir_test.go::TestScalarIntAffineLoopFunctionFromStackIRLowersScaleBiasLoop",
		"canonical scalar i32 affine sum loop with positive compile-time scale and bias 1..127 only; this is shape evidence, not throughput parity",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, affine)

	countdown, err := hotLoopRegisterRow(
		"scalar-countdown-loop",
		"scalar countdown loop",
		hotLoopCountdownFunc(),
		"machine-ir-countdown-loop",
		machine.ScalarIntCountdownLoopFunctionFromStackIR,
		"compiler/internal/machine/scalar_countdown_loop.go; compiler/internal/machine/ir_test.go::TestScalarIntCountdownLoopFunctionFromStackIRLowersDescendingLoop",
		"canonical scalar i32 countdown sum loop only; this is shape evidence, not throughput parity",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, countdown)

	slice, err := hotLoopRegisterRow(
		"proof-slice-sum-loop",
		"proof-tagged i32 slice sum loop",
		hotLoopSliceSumFunc(true),
		"machine-ir-slice-sum",
		machine.ScalarI32SliceSumLoopFunctionFromStackIR,
		"compiler/internal/machine/scalar_slice_sum.go; compiler/internal/machine/ir_test.go::TestScalarI32SliceSumLoopFromStackIRRequiresProofTaggedUncheckedLoad",
		"proof-tagged i32 slice sum loop only; checked/no-proof index loads stay out of this register-shape claim",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, slice)

	sliceStride, err := hotLoopRegisterRow(
		"proof-slice-stride-sum-loop",
		"proof-tagged i32 slice constant-stride sum loop",
		hotLoopSliceStrideSumFunc(true),
		"machine-ir-slice-stride-sum",
		machine.ScalarI32SliceSumLoopFunctionFromStackIR,
		"compiler/internal/machine/scalar_slice_sum.go; compiler/internal/machine/ir_test.go::TestScalarI32SliceSumLoopFromStackIRLowersProofTaggedConstantStride",
		"proof-tagged i32 slice sum loop with positive compile-time stride 2..127 only; checked/no-proof index loads and invalid strides stay out of this register-shape claim",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, sliceStride)

	call, err := hotLoopRegisterRow(
		"call-sum-loop",
		"scalar call loop",
		hotLoopCallSumFunc(),
		"machine-ir-call-loop",
		machine.ScalarIntCallLoopFunctionFromStackIR,
		"compiler/internal/machine/scalar_call_loop.go; compiler/internal/machine/ir_test.go::TestScalarIntCallLoopFunctionFromStackIRLowersCallWithABIClobbers",
		"single-i32 argument/result call loop with target ABI clobber metadata only",
	)
	if err != nil {
		return HotLoopShapeReport{}, err
	}
	rows = append(rows, call)

	rows = append(rows, hotLoopCheckedSliceFallbackRow())

	return HotLoopShapeReport{
		SchemaVersion: "tetra.optimizer.hot_loop_shape.v1",
		Rows:          rows,
		NonClaims: []string{
			"no C/Rust -O1/-O2 performance parity claim",
			"no general vectorization claim",
			"no unproven bounds-check removal claim",
		},
	}, nil
}

func hotLoopRegisterRow(id string, name string, fn ir.IRFunc, path string, lower hotLoopMachineLowerer, evidence string, boundary string) (HotLoopShapeRow, error) {
	ssaVerified, err := hotLoopSSAVerified(fn)
	if err != nil {
		return HotLoopShapeRow{}, err
	}
	mfn, ok, err := lower(fn)
	if err != nil {
		return HotLoopShapeRow{}, err
	}
	if !ok {
		return HotLoopShapeRow{}, fmt.Errorf("hot-loop shape: %s did not match %s", fn.Name, path)
	}
	if err := machine.VerifyFunction(mfn); err != nil {
		return HotLoopShapeRow{}, err
	}
	intervals, err := machine.BuildIntervals(mfn)
	if err != nil {
		return HotLoopShapeRow{}, err
	}
	alloc, err := machine.LinearScan(intervals, machine.LinuxX64CallerSaved())
	if err != nil {
		return HotLoopShapeRow{}, err
	}
	if alloc.Spills == nil {
		alloc.Spills = map[machine.VReg]int{}
	}
	if err := machine.VerifyAllocation(mfn, alloc, machine.LinuxX64CallerSaved(), len(alloc.Spills)); err != nil {
		return HotLoopShapeRow{}, err
	}
	return HotLoopShapeRow{
		ID:            id,
		Name:          name,
		RegisterPath:  true,
		SSAVerified:   ssaVerified,
		MachinePath:   path,
		MachineTarget: mfn.Target,
		SpillFree:     len(alloc.Spills) == 0,
		StackChurnOps: hotLoopStackChurnOps(mfn),
		RequiredOps:   hotLoopOps(mfn),
		ProofID:       hotLoopProofID(mfn),
		CallABI:       hotLoopCallABI(mfn),
		Evidence:      evidence,
		Boundary:      boundary,
	}, nil
}

func hotLoopCheckedSliceFallbackRow() HotLoopShapeRow {
	fn := hotLoopSliceSumFunc(false)
	ssaVerified, _ := hotLoopSSAVerified(fn)
	_, ok, err := machine.ScalarI32SliceSumLoopFunctionFromStackIR(fn)
	reason := "proof_tag_required_for_slice_sum_register_shape"
	if err != nil {
		reason = "machine_shape_error"
	} else if ok {
		reason = "unexpected_register_shape"
	}
	return HotLoopShapeRow{
		ID:           "checked-slice-sum-fallback",
		Name:         "checked i32 slice sum fallback",
		RegisterPath: false,
		SSAVerified:  ssaVerified,
		MachinePath:  "stack-fallback",
		Reason:       reason,
		Evidence:     "compiler/internal/machine/ir_test.go::TestScalarI32SliceSumLoopFromStackIRRequiresProofTaggedUncheckedLoad",
		Boundary:     "slice-sum register shape requires proof-tagged unchecked load; checked/no-proof shape remains stack fallback",
	}
}

func hotLoopSliceStrideSumFunc(proof bool) ir.IRFunc {
	fn := hotLoopSliceSumFunc(proof)
	fn.Name = "sum_stride"
	fn.Instrs[17].Imm = 2
	return fn
}

func hotLoopSSAVerified(fn ir.IRFunc) (bool, error) {
	ssaFn, ok, err := ssair.FromStackIRFunction(fn)
	if err != nil || !ok {
		return false, err
	}
	if err := ssair.VerifyFunction(ssaFn); err != nil {
		return false, err
	}
	return true, nil
}

func hotLoopOps(fn machine.Function) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			op := string(instr.Op)
			if op == "" || seen[op] {
				continue
			}
			seen[op] = true
			out = append(out, op)
		}
	}
	return out
}

func hotLoopStackChurnOps(fn machine.Function) int {
	count := 0
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch instr.Op {
			case machine.OpSpill, machine.OpReload, machine.OpPush, machine.OpPop:
				count++
			}
		}
	}
	return count
}

func hotLoopProofID(fn machine.Function) string {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if strings.HasPrefix(instr.Note, "proof:") {
				return instr.Note
			}
		}
	}
	return ""
}

func hotLoopCallABI(fn machine.Function) string {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == machine.OpCall {
				return instr.ABI
			}
		}
	}
	return ""
}

func hotLoopScalarSumFunc() ir.IRFunc {
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

func hotLoopScalarStrideFunc() ir.IRFunc {
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
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func hotLoopSumSquaresFunc() ir.IRFunc {
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

func hotLoopProductFunc() ir.IRFunc {
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

func hotLoopMaxFunc() ir.IRFunc {
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

func hotLoopScalarAffineFunc() ir.IRFunc {
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
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRConstI32, Imm: 1},
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

func hotLoopCountdownFunc() ir.IRFunc {
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

func hotLoopCallSumFunc() ir.IRFunc {
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

func hotLoopSliceSumFunc(proof bool) ir.IRFunc {
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
