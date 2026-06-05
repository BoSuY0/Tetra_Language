package machine

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
)

type ScalarU8CopyLoopPlan struct {
	Function     Function
	DstBaseLocal int
	SrcBaseLocal int
	LenLocal     int
	IndexLocal   int
	StartLabel   int
	EndLabel     int
	ProofID      string
}

type VectorU8x16CopyLoopPlan struct {
	Function           Function
	ScalarPlan         ScalarU8CopyLoopPlan
	LaneCount          int
	SafeUnaligned      bool
	TailHandling       string
	ScalarFallback     string
	NoAliasRequirement string
	ProofID            string
}

func ScalarU8CopyLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarU8CopyLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarU8CopyLoopPlanFromStackIR(fn ir.IRFunc) (ScalarU8CopyLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 3 || fn.LocalSlots < 4 {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 23 {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	startLabel := in[2].Label
	if in[2].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if !isLoad(in[3], indexLocal) || !isLoad(in[4], 2) || in[5].Kind != ir.IRCmpLtI32 || in[6].Kind != ir.IRJmpIfZero {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	endLabel := in[6].Label
	if endLabel < 0 {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if !isLoad(in[7], 0) || !isLoad(in[8], 2) || !isLoad(in[9], indexLocal) ||
		!isLoad(in[10], 1) || !isLoad(in[11], 2) || !isLoad(in[12], indexLocal) {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if in[13].Kind != ir.IRIndexLoadU8Unchecked || !strings.HasPrefix(in[13].ProofID, "proof:copy-loop:") {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if in[14].Kind != ir.IRIndexStoreU8 {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if !isLoad(in[15], indexLocal) || in[16].Kind != ir.IRConstI32 || in[16].Imm != 1 || in[17].Kind != ir.IRAddI32 || !isStore(in[18], indexLocal) {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if in[19].Kind != ir.IRJmp || in[19].Label != startLabel || in[20].Kind != ir.IRLabel || in[20].Label != endLabel || in[21].Kind != ir.IRConstI32 || in[21].Imm != 0 || in[22].Kind != ir.IRReturn {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarU8CopyLoopPlan{}, true, err
	}
	if indexLocal < fn.ParamSlots {
		return ScalarU8CopyLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	elem := VReg("t1")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-u8-copy",
		Params: []VReg{local(0), local(1), local(2)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(indexLocal), local(2)}, Note: "index < len"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpIndexLoad, Defs: []VReg{elem}, Uses: []VReg{local(1), local(2), local(indexLocal)}, Note: in[13].ProofID},
					{Op: OpIndexStore, Uses: []VReg{local(0), local(2), local(indexLocal), elem}, Note: in[13].ProofID + "; store uses same range proof"},
					{Op: OpInc, Defs: []VReg{local(indexLocal)}, Uses: []VReg{local(indexLocal)}, Note: "index++"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{"zero"}, Note: "returns 0"},
				},
			},
		},
	}
	out.Blocks[0].Instrs = append([]Instr{{Op: OpMov, Defs: []VReg{"zero"}, Imm: 0, Note: "return code = 0"}}, out.Blocks[0].Instrs...)
	if err := VerifyFunction(out); err != nil {
		return ScalarU8CopyLoopPlan{}, true, err
	}
	return ScalarU8CopyLoopPlan{
		Function:     out,
		DstBaseLocal: 0,
		SrcBaseLocal: 1,
		LenLocal:     2,
		IndexLocal:   indexLocal,
		StartLabel:   startLabel,
		EndLabel:     endLabel,
		ProofID:      in[13].ProofID,
	}, true, nil
}

func VectorU8x16CopyLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := VectorU8x16CopyLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func VectorU8x16CopyLoopPlanFromStackIR(fn ir.IRFunc) (VectorU8x16CopyLoopPlan, bool, error) {
	scalar, ok, err := ScalarU8CopyLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return VectorU8x16CopyLoopPlan{}, ok, err
	}

	local := func(slot int) VReg { return VReg("local" + itoa(slot)) }
	lane := VReg("vlane")
	cmp := VReg("vcmp")
	vchunk := VReg("vchunk")
	loopName := scalarLoopLabelName(scalar.StartLabel)
	tailName := "vector_tail"
	exitName := scalarLoopLabelName(scalar.EndLabel)
	proofNote := scalar.ProofID + "; safe unaligned u8x16 copy load/store; source/dest disjoint owned copy result"

	out := Function{
		Name:   fn.Name,
		Target: "vector-u8x16-copy-plan",
		Params: []VReg{local(0), local(1), local(2)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(scalar.IndexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpMov, Defs: []VReg{lane}, Imm: 16, Note: "u8x16 lane count"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpVectorCanCopyU8x16, Defs: []VReg{cmp}, Uses: []VReg{local(2), local(scalar.IndexLocal)}, Note: "index + 16 <= len"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: tailName, Note: "if fewer than sixteen bytes remain"},
					{Op: OpVectorLoadU8x16Unaligned, Defs: []VReg{vchunk}, Uses: []VReg{local(1), local(2), local(scalar.IndexLocal)}, Note: proofNote},
					{Op: OpVectorStoreU8x16Unaligned, Uses: []VReg{local(0), local(2), local(scalar.IndexLocal), vchunk}, Note: proofNote},
					{Op: OpAdd, Defs: []VReg{local(scalar.IndexLocal)}, Uses: []VReg{local(scalar.IndexLocal), lane}, Note: "index += 16"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, tailName},
			},
			{
				Name: tailName,
				Instrs: []Instr{
					{Op: OpTailScalarU8Copy, Uses: []VReg{local(0), local(1), local(2), local(scalar.IndexLocal)}, Note: scalar.ProofID + "; scalar tail handles len % 16"},
					{Op: OpBranch, Target: exitName},
				},
				Successors: []string{exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{"zero"}, Note: "returns 0"},
				},
			},
		},
	}
	out.Blocks[0].Instrs = append([]Instr{{Op: OpMov, Defs: []VReg{"zero"}, Imm: 0, Note: "return code = 0"}}, out.Blocks[0].Instrs...)
	if err := VerifyFunction(out); err != nil {
		return VectorU8x16CopyLoopPlan{}, true, err
	}
	return VectorU8x16CopyLoopPlan{
		Function:           out,
		ScalarPlan:         scalar,
		LaneCount:          16,
		SafeUnaligned:      true,
		TailHandling:       "scalar_tail",
		ScalarFallback:     scalar.Function.Target,
		NoAliasRequirement: "required_source_dest_disjoint_owned_copy_result",
		ProofID:            scalar.ProofID,
	}, true, nil
}
