package machine

import (
	"tetra_language/compiler/internal/ir"
)

type VectorI32x4SliceSumLoopPlan struct {
	Function           Function
	ScalarPlan         ScalarI32SliceSumLoopPlan
	LaneCount          int
	SafeUnaligned      bool
	TailHandling       string
	ScalarFallback     string
	NoAliasRequirement string
	ProofID            string
}

func VectorI32x4SliceSumLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := VectorI32x4SliceSumLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func VectorI32x4SliceSumLoopPlanFromStackIR(fn ir.IRFunc) (VectorI32x4SliceSumLoopPlan, bool, error) {
	scalar, ok, err := ScalarI32SliceSumLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return VectorI32x4SliceSumLoopPlan{}, ok, err
	}
	if scalar.Step != 1 {
		return VectorI32x4SliceSumLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg("local" + itoa(slot)) }
	lane := VReg("vlane")
	cmp := VReg("vcmp")
	vsum := VReg("vsum")
	vchunk := VReg("vchunk")
	loopName := scalarLoopLabelName(scalar.StartLabel)
	tailName := "vector_tail"
	exitName := scalarLoopLabelName(scalar.EndLabel)
	proofNote := scalar.ProofID + "; safe unaligned i32x4 vector load"

	out := Function{
		Name:   fn.Name,
		Target: "vector-i32x4-slice-sum-plan",
		Params: []VReg{local(0), local(1)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(scalar.TotalLocal)}, Imm: 0, Note: "total = 0"},
					{Op: OpMov, Defs: []VReg{local(scalar.IndexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpMov, Defs: []VReg{lane}, Imm: 4, Note: "i32x4 lane count"},
					{Op: OpVectorZeroI32x4, Defs: []VReg{vsum}, Note: "vector accumulator = 0"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpVectorCanLoadI32x4, Defs: []VReg{cmp}, Uses: []VReg{local(1), local(scalar.IndexLocal)}, Note: "index + 4 <= len"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: tailName, Note: "if fewer than four elements remain"},
					{Op: OpVectorLoadI32x4Unaligned, Defs: []VReg{vchunk}, Uses: []VReg{local(0), local(1), local(scalar.IndexLocal)}, Note: proofNote},
					{Op: OpVectorAddI32x4, Defs: []VReg{vsum}, Uses: []VReg{vsum, vchunk}, Note: "vector total += xs[index:index+4]"},
					{Op: OpAdd, Defs: []VReg{local(scalar.IndexLocal)}, Uses: []VReg{local(scalar.IndexLocal), lane}, Note: "index += 4"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, tailName},
			},
			{
				Name: tailName,
				Instrs: []Instr{
					{Op: OpVectorHorizontalAddI32x4, Defs: []VReg{local(scalar.TotalLocal)}, Uses: []VReg{vsum}, Note: "horizontal reduce vector accumulator"},
					{Op: OpTailScalarI32Sum, Defs: []VReg{local(scalar.TotalLocal)}, Uses: []VReg{local(scalar.TotalLocal), local(0), local(1), local(scalar.IndexLocal)}, Note: scalar.ProofID + "; scalar tail handles len % 4"},
					{Op: OpBranch, Target: exitName},
				},
				Successors: []string{exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(scalar.TotalLocal)}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return VectorI32x4SliceSumLoopPlan{}, true, err
	}
	return VectorI32x4SliceSumLoopPlan{
		Function:           out,
		ScalarPlan:         scalar,
		LaneCount:          4,
		SafeUnaligned:      true,
		TailHandling:       "scalar_tail",
		ScalarFallback:     scalar.Function.Target,
		NoAliasRequirement: "not_required_read_only_reduction",
		ProofID:            scalar.ProofID,
	}, true, nil
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	n := v
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
