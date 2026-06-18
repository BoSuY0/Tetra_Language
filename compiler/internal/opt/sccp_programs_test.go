package opt

import "tetra_language/compiler/internal/ir"

func constantZeroBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func singlePredecessorKnownLocalBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRJmp, Label: 1},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func mergeLabelKnownLocalBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRJmp, Label: 1},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func forwardSinglePredecessorKnownLocalBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRJmp, Label: 1},
				{Kind: ir.IRConstI32, Imm: 11},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func forwardFallthroughPredecessorKnownLocalBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRJmp, Label: 2},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRJmpIfZero, Label: 3},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 3},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func foldedZeroBranchSinglePredecessorKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func foldedZeroBranchFallthroughTargetKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func foldedNonzeroFallthroughOnlyLabelKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 9},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 9},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func foldedNonzeroFallthroughExplicitIncomingLabelKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 9},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 9},
				{Kind: ir.IRJmp, Label: 1},
			},
		}},
	}
}

func dynamicZeroTargetPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicNonzeroFallthroughPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 13},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicZeroFallthroughTargetPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicEqZeroFallthroughPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCmpEqI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 13},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicEqZeroTargetNonzeroPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCmpEqI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicNeZeroFallthroughPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCmpNeI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 13},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicNeZeroTargetZeroPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCmpNeI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func dynamicComparisonFallthroughTargetPathKnownLocalProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCmpNeI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func knownLocalLessThanBranchProgram(localValue int32, compareImm int32) *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: localValue},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: compareImm},
				{Kind: ir.IRCmpLtI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func unaryNegBranchProgram(imm int32) *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: imm},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func storedUnaryNegBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRNegI32},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func storedConstantExpressionBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRSubI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func storedDynamicExpressionBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ParamSlots:  1,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRSubI32},
				{Kind: ir.IRStoreLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func knownLocalZeroBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			LocalSlots:  1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 99},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func constantNonZeroBranchProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 42},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func countDecisions(decisions []PassDecision, action string, reason string) int {
	count := 0
	for _, decision := range decisions {
		if decision.Action == action && decision.Reason == reason {
			count++
		}
	}
	return count
}
