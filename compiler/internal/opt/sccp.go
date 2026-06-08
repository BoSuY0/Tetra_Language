package opt

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

type sccpState struct {
	decisions []PassDecision
}

func SCCPPass() Pass {
	state := &sccpState{}
	return Pass{
		Name:                      "sccp-constant-branch",
		InputKind:                 IRKindStack,
		OutputKind:                IRKindStack,
		InputVerifier:             VerifierLowerVerifyProgram,
		OutputVerifier:            VerifierLowerVerifyProgram,
		RequiredFacts:             []Fact{FactIRVerified},
		PreservedFacts:            []Fact{FactBoundsProofs},
		InvalidatedFacts:          []Fact{FactLiveness},
		ProofRule:                 ProofRulePreserveBoundsInvalidateLiveness,
		ValidationStrategy:        ValidationTranslation,
		TranslationValidationHook: TranslationHookValidateTranslation,
		ReportOutput:              "sccp-constant-branch.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       state.run,
		Decisions:                 state.reportDecisions,
	}
}

func (s *sccpState) run(prog *ir.IRProgram) error {
	if prog == nil {
		return fmt.Errorf("sccp-constant-branch: missing IR program")
	}
	s.decisions = nil
	for i := range prog.Funcs {
		fn := &prog.Funcs[i]
		if fn.Policy.HasBudget || fn.Policy.HasConsent {
			s.decisions = append(s.decisions, PassDecision{Action: "not_folded", Caller: fn.Name, Site: 0, Reason: "policy_guarded_function"})
			continue
		}
		fn.Instrs = s.rewriteFunc(fn.Name, fn.Instrs)
	}
	return nil
}

func (s *sccpState) reportDecisions() []PassDecision {
	return append([]PassDecision(nil), s.decisions...)
}

func (s *sccpState) rewriteFunc(fnName string, instrs []ir.IRInstr) []ir.IRInstr {
	out := make([]ir.IRInstr, 0, len(instrs))
	knownLocals := map[int]int32{}
	knownZeroLocals := map[int]bool{}
	knownNonZeroLocals := map[int]bool{}
	constStack := make([]knownStackValue, 0)
	labelIncoming := countLabelIncoming(instrs)
	labelIndexes := indexLabels(instrs)
	pendingLabelFacts := map[int]map[int]int32{}
	pendingLabelZeroFacts := map[int]map[int]bool{}
	pendingLabelNonZeroFacts := map[int]map[int]bool{}
	for i := 0; i < len(instrs); i++ {
		if i+1 < len(instrs) && instrs[i].Kind == ir.IRConstI32 && instrs[i+1].Kind == ir.IRJmpIfZero {
			branch := instrs[i+1]
			if instrs[i].Imm == 0 {
				out = append(out, ir.IRInstr{Kind: ir.IRJmp, Label: branch.Label, Pos: branch.Pos})
				s.decisions = append(s.decisions, PassDecision{Action: "folded_const_zero_branch", Caller: fnName, Site: i, Reason: "constant_condition"})
				s.propagateKnownLocalsThroughFoldedZeroBranch(fnName, instrs, labelIndexes, labelIncoming, knownLocals, i+1, branch.Label, pendingLabelFacts)
				clearKnownStack(&constStack)
				next, pruned := skipFallthroughUntilLabel(instrs, i+2)
				if pruned > 0 {
					s.decisions = append(s.decisions, PassDecision{Action: "pruned_unreachable_fallthrough", Caller: fnName, Site: i + 2, Reason: "constant_branch_reachability"})
				}
				i = next - 1
				continue
			} else {
				s.decisions = append(s.decisions, PassDecision{Action: "folded_const_nonzero_fallthrough", Caller: fnName, Site: i, Reason: "constant_condition"})
				s.propagateKnownLocalsThroughFoldedNonzeroFallthrough(fnName, instrs, labelIncoming, knownLocals, i+1, i+2, pendingLabelFacts)
				clearKnownStack(&constStack)
			}
			i++
			continue
		}
		if i+1 < len(instrs) && instrs[i].Kind == ir.IRLoadLocal && instrs[i+1].Kind == ir.IRJmpIfZero {
			branch := instrs[i+1]
			local := instrs[i].Local
			if value, ok := knownLocals[local]; ok {
				if value == 0 {
					out = append(out, ir.IRInstr{Kind: ir.IRJmp, Label: branch.Label, Pos: branch.Pos})
					s.decisions = append(s.decisions, PassDecision{Action: "folded_known_local_zero_branch", Caller: fnName, Site: i, Reason: "constant_local_condition"})
					s.propagateKnownLocalsThroughFoldedZeroBranch(fnName, instrs, labelIndexes, labelIncoming, knownLocals, i+1, branch.Label, pendingLabelFacts)
					clearKnownStack(&constStack)
					next, pruned := skipFallthroughUntilLabel(instrs, i+2)
					if pruned > 0 {
						s.decisions = append(s.decisions, PassDecision{Action: "pruned_unreachable_fallthrough", Caller: fnName, Site: i + 2, Reason: "constant_branch_reachability"})
					}
					i = next - 1
					continue
				}
				s.decisions = append(s.decisions, PassDecision{Action: "folded_known_local_nonzero_fallthrough", Caller: fnName, Site: i, Reason: "constant_local_condition"})
				s.propagateKnownLocalsThroughFoldedNonzeroFallthrough(fnName, instrs, labelIncoming, knownLocals, i+1, i+2, pendingLabelFacts)
				clearKnownStack(&constStack)
				i++
				continue
			}
			if knownZeroLocals[local] {
				out = append(out, ir.IRInstr{Kind: ir.IRJmp, Label: branch.Label, Pos: branch.Pos})
				s.decisions = append(s.decisions, PassDecision{Action: "folded_path_local_zero_branch", Caller: fnName, Site: i, Reason: "path_local_condition"})
				s.propagatePathLocalZeroThroughFoldedZeroBranch(fnName, instrs, labelIndexes, labelIncoming, local, i+1, branch.Label, pendingLabelZeroFacts)
				clearKnownStack(&constStack)
				next, pruned := skipFallthroughUntilLabel(instrs, i+2)
				if pruned > 0 {
					s.decisions = append(s.decisions, PassDecision{Action: "pruned_unreachable_fallthrough", Caller: fnName, Site: i + 2, Reason: "constant_branch_reachability"})
				}
				i = next - 1
				continue
			}
			if knownNonZeroLocals[local] {
				s.decisions = append(s.decisions, PassDecision{Action: "folded_path_local_nonzero_fallthrough", Caller: fnName, Site: i, Reason: "path_local_condition"})
				s.propagatePathLocalNonZeroThroughFoldedNonzeroFallthrough(fnName, instrs, labelIncoming, local, i+1, i+2, pendingLabelNonZeroFacts)
				clearKnownStack(&constStack)
				i++
				continue
			}
			out = append(out, instrs[i], branch)
			s.decisions = append(s.decisions, PassDecision{Action: "not_folded", Caller: fnName, Site: i + 1, Reason: "dynamic_condition"})
			s.propagatePathLocalZeroThroughDynamicBranchTarget(fnName, instrs, labelIndexes, labelIncoming, local, i+1, branch.Label, pendingLabelZeroFacts)
			setKnownLocalNonZero(knownLocals, knownZeroLocals, knownNonZeroLocals, local)
			s.decisions = append(s.decisions, PassDecision{Action: "derived_path_local_nonzero_fallthrough", Caller: fnName, Site: i + 1, Reason: "dynamic_branch_fallthrough"})
			clearKnownStack(&constStack)
			i++
			continue
		}
		if i+1 < len(instrs) && i >= 1 && instrs[i+1].Kind == ir.IRJmpIfZero && isPureLocalUnaryOp(instrs[i].Kind) {
			operand, operandOK := knownBranchOperandConst(instrs[i-1], knownLocals)
			value, folded := foldConstUnaryI32(instrs[i].Kind, operand)
			if operandOK && folded && len(out) >= 1 {
				branch := instrs[i+1]
				out = out[:len(out)-1]
				if value == 0 {
					out = append(out, ir.IRInstr{Kind: ir.IRJmp, Label: branch.Label, Pos: branch.Pos})
					s.decisions = append(s.decisions, PassDecision{Action: "folded_const_unary_expr_zero_branch", Caller: fnName, Site: i - 1, Reason: "constant_unary_expression_condition"})
					s.propagateKnownLocalsThroughFoldedZeroBranch(fnName, instrs, labelIndexes, labelIncoming, knownLocals, i+1, branch.Label, pendingLabelFacts)
					clearKnownStack(&constStack)
					next, pruned := skipFallthroughUntilLabel(instrs, i+2)
					if pruned > 0 {
						s.decisions = append(s.decisions, PassDecision{Action: "pruned_unreachable_fallthrough", Caller: fnName, Site: i + 2, Reason: "constant_branch_reachability"})
					}
					i = next - 1
					continue
				}
				s.decisions = append(s.decisions, PassDecision{Action: "folded_const_unary_expr_nonzero_fallthrough", Caller: fnName, Site: i - 1, Reason: "constant_unary_expression_condition"})
				s.propagateKnownLocalsThroughFoldedNonzeroFallthrough(fnName, instrs, labelIncoming, knownLocals, i+1, i+2, pendingLabelFacts)
				clearKnownStack(&constStack)
				i++
				continue
			}
		}
		if i+1 < len(instrs) && i >= 2 && instrs[i+1].Kind == ir.IRJmpIfZero && isPureLocalBinaryOp(instrs[i].Kind) {
			left, leftOK := knownBranchOperandConst(instrs[i-2], knownLocals)
			right, rightOK := knownBranchOperandConst(instrs[i-1], knownLocals)
			value, folded := foldConstBinaryI32(instrs[i].Kind, left, right)
			if leftOK && rightOK && folded && len(out) >= 2 {
				branch := instrs[i+1]
				out = out[:len(out)-2]
				if value == 0 {
					out = append(out, ir.IRInstr{Kind: ir.IRJmp, Label: branch.Label, Pos: branch.Pos})
					s.decisions = append(s.decisions, PassDecision{Action: "folded_const_expr_zero_branch", Caller: fnName, Site: i - 2, Reason: "constant_expression_condition"})
					s.propagateKnownLocalsThroughFoldedZeroBranch(fnName, instrs, labelIndexes, labelIncoming, knownLocals, i+1, branch.Label, pendingLabelFacts)
					clearKnownStack(&constStack)
					next, pruned := skipFallthroughUntilLabel(instrs, i+2)
					if pruned > 0 {
						s.decisions = append(s.decisions, PassDecision{Action: "pruned_unreachable_fallthrough", Caller: fnName, Site: i + 2, Reason: "constant_branch_reachability"})
					}
					i = next - 1
					continue
				}
				s.decisions = append(s.decisions, PassDecision{Action: "folded_const_expr_nonzero_fallthrough", Caller: fnName, Site: i - 2, Reason: "constant_expression_condition"})
				s.propagateKnownLocalsThroughFoldedNonzeroFallthrough(fnName, instrs, labelIncoming, knownLocals, i+1, i+2, pendingLabelFacts)
				clearKnownStack(&constStack)
				i++
				continue
			}
		}
		if i+1 < len(instrs) && i >= 2 && instrs[i+1].Kind == ir.IRJmpIfZero {
			if fact, ok := zeroComparisonLocalFact(instrs, i); ok {
				branch := instrs[i+1]
				out = append(out, instrs[i], branch)
				s.decisions = append(s.decisions, PassDecision{Action: "not_folded", Caller: fnName, Site: i + 1, Reason: "dynamic_condition"})
				s.propagateComparisonPathLocalThroughDynamicBranchTarget(
					fnName,
					instrs,
					labelIndexes,
					labelIncoming,
					fact,
					i+1,
					branch.Label,
					pendingLabelZeroFacts,
					pendingLabelNonZeroFacts,
				)
				if fact.FallthroughZero {
					setKnownLocalZero(knownLocals, knownZeroLocals, knownNonZeroLocals, fact.Local)
					s.decisions = append(s.decisions, PassDecision{Action: "derived_comparison_path_local_zero_fallthrough", Caller: fnName, Site: i + 1, Reason: fact.FallthroughReason + "_fallthrough"})
				} else {
					setKnownLocalNonZero(knownLocals, knownZeroLocals, knownNonZeroLocals, fact.Local)
					s.decisions = append(s.decisions, PassDecision{Action: "derived_comparison_path_local_nonzero_fallthrough", Caller: fnName, Site: i + 1, Reason: fact.FallthroughReason + "_fallthrough"})
				}
				clearKnownStack(&constStack)
				i++
				continue
			}
		}
		if i+1 < len(instrs) && instrs[i+1].Kind == ir.IRJmpIfZero {
			s.decisions = append(s.decisions, PassDecision{Action: "not_folded", Caller: fnName, Site: i + 1, Reason: "dynamic_condition"})
		}
		instr := instrs[i]
		switch instr.Kind {
		case ir.IRConstI32:
			pushKnownStackConst(&constStack, instr.Imm)
		case ir.IRLoadLocal:
			if value, ok := knownLocals[instr.Local]; ok {
				pushKnownStackConst(&constStack, value)
			} else {
				pushUnknownStackValue(&constStack)
			}
		case ir.IRStoreLocal:
			if value, ok := popKnownStackConst(&constStack); ok {
				setKnownLocalConst(knownLocals, knownZeroLocals, knownNonZeroLocals, instr.Local, value)
				s.decisions = append(s.decisions, PassDecision{Action: "tracked_known_local_store", Caller: fnName, Site: i, Reason: "constant_stack_store"})
			} else {
				deleteKnownLocalFact(knownLocals, knownZeroLocals, knownNonZeroLocals, instr.Local)
			}
		case ir.IRNegI32:
			operand, ok := popKnownStackConst(&constStack)
			if !ok {
				pushUnknownStackValue(&constStack)
			} else if value, folded := foldConstUnaryI32(instr.Kind, operand); folded {
				pushKnownStackConst(&constStack, value)
			} else {
				pushUnknownStackValue(&constStack)
			}
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			right, rightOK := popKnownStackConst(&constStack)
			left, leftOK := popKnownStackConst(&constStack)
			if leftOK && rightOK {
				if value, folded := foldConstBinaryI32(instr.Kind, left, right); folded {
					pushKnownStackConst(&constStack, value)
				} else {
					pushUnknownStackValue(&constStack)
				}
			} else {
				pushUnknownStackValue(&constStack)
			}
		case ir.IRJmp:
			if reason, ok := knownLocalSinglePredecessorLabelReason(instrs, labelIndexes, labelIncoming, i, instr.Label); ok && len(knownLocals) > 0 {
				pendingLabelFacts[instr.Label] = cloneKnownLocalConsts(knownLocals)
				s.decisions = append(s.decisions, PassDecision{Action: "propagated_known_local_single_predecessor", Caller: fnName, Site: i, Reason: reason})
			}
			clearKnownLocalFacts(knownLocals, knownZeroLocals, knownNonZeroLocals)
			clearKnownStack(&constStack)
		case ir.IRLabel:
			clearKnownLocalFacts(knownLocals, knownZeroLocals, knownNonZeroLocals)
			clearKnownStack(&constStack)
			if facts, ok := pendingLabelFacts[instr.Label]; ok {
				applyKnownLocalConsts(knownLocals, knownZeroLocals, knownNonZeroLocals, facts)
			}
			if facts, ok := pendingLabelZeroFacts[instr.Label]; ok {
				applyKnownLocalZeros(knownLocals, knownZeroLocals, knownNonZeroLocals, facts)
			}
			if facts, ok := pendingLabelNonZeroFacts[instr.Label]; ok {
				applyKnownLocalNonZeros(knownLocals, knownZeroLocals, knownNonZeroLocals, facts)
			}
			delete(pendingLabelFacts, instr.Label)
			delete(pendingLabelZeroFacts, instr.Label)
			delete(pendingLabelNonZeroFacts, instr.Label)
		case ir.IRJmpIfZero:
			clearKnownLocalFacts(knownLocals, knownZeroLocals, knownNonZeroLocals)
			clearKnownStack(&constStack)
		default:
			if clearsCopyFacts(instr.Kind) {
				clearKnownLocalFacts(knownLocals, knownZeroLocals, knownNonZeroLocals)
				clearKnownStack(&constStack)
			} else {
				updateKnownStackForOpaqueInstr(&constStack, instr)
			}
		}
		out = append(out, instr)
	}
	return out
}

type knownStackValue struct {
	value int32
	known bool
}

func pushKnownStackConst(stack *[]knownStackValue, value int32) {
	*stack = append(*stack, knownStackValue{value: value, known: true})
}

func pushUnknownStackValue(stack *[]knownStackValue) {
	*stack = append(*stack, knownStackValue{})
}

func popKnownStackConst(stack *[]knownStackValue) (int32, bool) {
	if len(*stack) == 0 {
		return 0, false
	}
	value := (*stack)[len(*stack)-1]
	*stack = (*stack)[:len(*stack)-1]
	return value.value, value.known
}

func clearKnownStack(stack *[]knownStackValue) {
	*stack = (*stack)[:0]
}

func updateKnownStackForOpaqueInstr(stack *[]knownStackValue, instr ir.IRInstr) {
	switch instr.Kind {
	case ir.IRStrLit, ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32,
		ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32,
		ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32,
		ir.IRRawSliceFromParts, ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
		clearKnownStack(stack)
	case ir.IRCall, ir.IRWrite, ir.IRReturn, ir.IRAllocBytes, ir.IRRegionEnter, ir.IRRegionReset,
		ir.IRIslandNew, ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32,
		ir.IRIslandFree, ir.IRIslandReset, ir.IRCapIO, ir.IRCapMem, ir.IRMemReadI32, ir.IRMemWriteI32,
		ir.IRMemReadU8, ir.IRMemWriteU8, ir.IRMemReadPtr, ir.IRMemWritePtr,
		ir.IRMemWriteArchPtr, ir.IRMemReadI32Offset, ir.IRMemWriteI32Offset,
		ir.IRMemReadU8Offset, ir.IRMemWriteU8Offset, ir.IRMemReadPtrOffset,
		ir.IRMemWritePtrOffset, ir.IRMemWriteArchPtrOffset, ir.IRPtrAdd,
		ir.IRMmioReadI32, ir.IRMmioWriteI32, ir.IRSymAddr, ir.IRCtxSwitch,
		ir.IRAtomicLoadPtr, ir.IRAtomicStorePtr, ir.IRAtomicExchangePtr,
		ir.IRAtomicFetchAddPtr, ir.IRAtomicFetchSubPtr, ir.IRAtomicFetchAndPtr,
		ir.IRAtomicFetchOrPtr, ir.IRAtomicFetchXorPtr, ir.IRAtomicCompareExchangePtr,
		ir.IRAtomicFenceSeqCst, ir.IRAtomicFenceRelaxed, ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease, ir.IRAtomicFenceAcqRel, ir.IRAtomicLoadI32,
		ir.IRAtomicStoreI32, ir.IRAtomicExchangeI32, ir.IRAtomicCompareExchangeI32,
		ir.IRAtomicFetchAddI32, ir.IRAtomicFetchSubI32, ir.IRAtomicFetchAndI32,
		ir.IRAtomicFetchOrI32, ir.IRAtomicFetchXorI32, ir.IRAtomicLoadI64,
		ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64, ir.IRAtomicCompareExchangeI64,
		ir.IRAtomicFetchAddI64, ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64, ir.IRAtomicFetchXorI64, ir.IRAtomicLoadI8,
		ir.IRAtomicStoreI8, ir.IRAtomicExchangeI8, ir.IRAtomicCompareExchangeI8,
		ir.IRAtomicFetchAddI8, ir.IRAtomicFetchSubI8, ir.IRAtomicFetchAndI8,
		ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8, ir.IRAtomicLoadI16,
		ir.IRAtomicStoreI16, ir.IRAtomicExchangeI16, ir.IRAtomicCompareExchangeI16,
		ir.IRAtomicFetchAddI16, ir.IRAtomicFetchSubI16, ir.IRAtomicFetchAndI16,
		ir.IRAtomicFetchOrI16, ir.IRAtomicFetchXorI16:
		clearKnownStack(stack)
	default:
		clearKnownStack(stack)
	}
}

func knownBranchOperandConst(instr ir.IRInstr, knownLocals map[int]int32) (int32, bool) {
	switch instr.Kind {
	case ir.IRConstI32:
		return instr.Imm, true
	case ir.IRLoadLocal:
		value, ok := knownLocals[instr.Local]
		return value, ok
	default:
		return 0, false
	}
}

type zeroComparisonFact struct {
	Local             int
	TargetZero        bool
	TargetReason      string
	FallthroughZero   bool
	FallthroughReason string
}

func zeroComparisonLocalFact(instrs []ir.IRInstr, cmpIndex int) (zeroComparisonFact, bool) {
	if cmpIndex < 2 {
		return zeroComparisonFact{}, false
	}
	load := instrs[cmpIndex-2]
	zero := instrs[cmpIndex-1]
	if load.Kind != ir.IRLoadLocal || zero.Kind != ir.IRConstI32 || zero.Imm != 0 {
		return zeroComparisonFact{}, false
	}
	switch instrs[cmpIndex].Kind {
	case ir.IRCmpEqI32:
		return zeroComparisonFact{
			Local:             load.Local,
			TargetZero:        false,
			TargetReason:      "eq_zero_false",
			FallthroughZero:   true,
			FallthroughReason: "eq_zero_true",
		}, true
	case ir.IRCmpNeI32:
		return zeroComparisonFact{
			Local:             load.Local,
			TargetZero:        true,
			TargetReason:      "ne_zero_false",
			FallthroughZero:   false,
			FallthroughReason: "ne_zero_true",
		}, true
	default:
		return zeroComparisonFact{}, false
	}
}

func previousKnownStackConst(out []ir.IRInstr, knownLocals map[int]int32) (int32, bool) {
	if len(out) == 0 {
		return 0, false
	}
	prev := out[len(out)-1]
	switch prev.Kind {
	case ir.IRConstI32:
		return prev.Imm, true
	case ir.IRLoadLocal:
		value, ok := knownLocals[prev.Local]
		return value, ok
	default:
		if len(out) >= 2 && isPureLocalUnaryOp(prev.Kind) {
			operand, operandOK := knownBranchOperandConst(out[len(out)-2], knownLocals)
			if operandOK {
				return foldConstUnaryI32(prev.Kind, operand)
			}
		}
		if len(out) >= 3 && isPureLocalBinaryOp(prev.Kind) {
			left, leftOK := knownBranchOperandConst(out[len(out)-3], knownLocals)
			right, rightOK := knownBranchOperandConst(out[len(out)-2], knownLocals)
			if leftOK && rightOK {
				return foldConstBinaryI32(prev.Kind, left, right)
			}
		}
		return 0, false
	}
}

func isPureLocalUnaryOp(kind ir.IRInstrKind) bool {
	return kind == ir.IRNegI32
}

func foldConstUnaryI32(kind ir.IRInstrKind, value int32) (int32, bool) {
	switch kind {
	case ir.IRNegI32:
		return checkedNegI32(value)
	default:
		return 0, false
	}
}

func countLabelIncoming(instrs []ir.IRInstr) map[int]int {
	incoming := map[int]int{}
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRJmp, ir.IRJmpIfZero:
			incoming[instr.Label]++
		}
	}
	return incoming
}

func indexLabels(instrs []ir.IRInstr) map[int]int {
	indexes := map[int]int{}
	for i, instr := range instrs {
		if instr.Kind == ir.IRLabel {
			indexes[instr.Label] = i
		}
	}
	return indexes
}

func knownLocalSinglePredecessorLabelReason(instrs []ir.IRInstr, labelIndexes map[int]int, labelIncoming map[int]int, jumpIndex int, label int) (string, bool) {
	targetIndex, ok := labelIndexes[label]
	if !ok || targetIndex <= jumpIndex || labelIncoming[label] != 1 {
		return "", false
	}
	if !labelHasNoFallthroughPredecessor(instrs, targetIndex) {
		return "", false
	}
	if targetIndex == jumpIndex+1 {
		return "single_predecessor_label", true
	}
	return "forward_single_predecessor_jump", true
}

func (s *sccpState) propagateKnownLocalsThroughFoldedZeroBranch(
	fnName string,
	instrs []ir.IRInstr,
	labelIndexes map[int]int,
	labelIncoming map[int]int,
	knownLocals map[int]int32,
	branchIndex int,
	label int,
	pendingLabelFacts map[int]map[int]int32,
) {
	if len(knownLocals) == 0 {
		return
	}
	reason, ok := knownLocalSinglePredecessorLabelReason(instrs, labelIndexes, labelIncoming, branchIndex, label)
	if !ok {
		return
	}
	pendingLabelFacts[label] = cloneKnownLocalConsts(knownLocals)
	switch reason {
	case "single_predecessor_label":
		reason = "folded_zero_branch_single_predecessor_label"
	case "forward_single_predecessor_jump":
		reason = "folded_zero_branch_forward_single_predecessor_jump"
	}
	s.decisions = append(s.decisions, PassDecision{Action: "propagated_known_local_folded_zero_branch", Caller: fnName, Site: branchIndex, Reason: reason})
}

func (s *sccpState) propagateKnownLocalsThroughFoldedNonzeroFallthrough(
	fnName string,
	instrs []ir.IRInstr,
	labelIncoming map[int]int,
	knownLocals map[int]int32,
	branchIndex int,
	fallthroughIndex int,
	pendingLabelFacts map[int]map[int]int32,
) {
	if len(knownLocals) == 0 || fallthroughIndex >= len(instrs) {
		return
	}
	fallthroughInstr := instrs[fallthroughIndex]
	if fallthroughInstr.Kind != ir.IRLabel || labelIncoming[fallthroughInstr.Label] != 0 {
		return
	}
	pendingLabelFacts[fallthroughInstr.Label] = cloneKnownLocalConsts(knownLocals)
	s.decisions = append(s.decisions, PassDecision{Action: "propagated_known_local_folded_nonzero_fallthrough", Caller: fnName, Site: branchIndex, Reason: "folded_nonzero_fallthrough_label"})
}

func (s *sccpState) propagatePathLocalZeroThroughDynamicBranchTarget(
	fnName string,
	instrs []ir.IRInstr,
	labelIndexes map[int]int,
	labelIncoming map[int]int,
	local int,
	branchIndex int,
	label int,
	pendingLabelZeroFacts map[int]map[int]bool,
) {
	reason, ok := knownLocalSinglePredecessorLabelReason(instrs, labelIndexes, labelIncoming, branchIndex, label)
	if !ok {
		return
	}
	addPendingLocalBoolFact(pendingLabelZeroFacts, label, local)
	switch reason {
	case "single_predecessor_label":
		reason = "dynamic_zero_single_predecessor_label"
	case "forward_single_predecessor_jump":
		reason = "dynamic_zero_forward_single_predecessor_jump"
	}
	s.decisions = append(s.decisions, PassDecision{Action: "propagated_path_local_zero_target", Caller: fnName, Site: branchIndex, Reason: reason})
}

func (s *sccpState) propagatePathLocalZeroThroughFoldedZeroBranch(
	fnName string,
	instrs []ir.IRInstr,
	labelIndexes map[int]int,
	labelIncoming map[int]int,
	local int,
	branchIndex int,
	label int,
	pendingLabelZeroFacts map[int]map[int]bool,
) {
	reason, ok := knownLocalSinglePredecessorLabelReason(instrs, labelIndexes, labelIncoming, branchIndex, label)
	if !ok {
		return
	}
	addPendingLocalBoolFact(pendingLabelZeroFacts, label, local)
	switch reason {
	case "single_predecessor_label":
		reason = "path_zero_single_predecessor_label"
	case "forward_single_predecessor_jump":
		reason = "path_zero_forward_single_predecessor_jump"
	}
	s.decisions = append(s.decisions, PassDecision{Action: "propagated_path_local_zero_target", Caller: fnName, Site: branchIndex, Reason: reason})
}

func (s *sccpState) propagatePathLocalNonZeroThroughFoldedNonzeroFallthrough(
	fnName string,
	instrs []ir.IRInstr,
	labelIncoming map[int]int,
	local int,
	branchIndex int,
	fallthroughIndex int,
	pendingLabelNonZeroFacts map[int]map[int]bool,
) {
	if fallthroughIndex >= len(instrs) {
		return
	}
	fallthroughInstr := instrs[fallthroughIndex]
	if fallthroughInstr.Kind != ir.IRLabel || labelIncoming[fallthroughInstr.Label] != 0 {
		return
	}
	addPendingLocalBoolFact(pendingLabelNonZeroFacts, fallthroughInstr.Label, local)
	s.decisions = append(s.decisions, PassDecision{Action: "propagated_path_local_nonzero_fallthrough", Caller: fnName, Site: branchIndex, Reason: "path_nonzero_fallthrough_label"})
}

func (s *sccpState) propagateComparisonPathLocalThroughDynamicBranchTarget(
	fnName string,
	instrs []ir.IRInstr,
	labelIndexes map[int]int,
	labelIncoming map[int]int,
	fact zeroComparisonFact,
	branchIndex int,
	label int,
	pendingLabelZeroFacts map[int]map[int]bool,
	pendingLabelNonZeroFacts map[int]map[int]bool,
) {
	reason, ok := knownLocalSinglePredecessorLabelReason(instrs, labelIndexes, labelIncoming, branchIndex, label)
	if !ok {
		return
	}
	switch reason {
	case "single_predecessor_label":
		reason = fact.TargetReason + "_single_predecessor_label"
	case "forward_single_predecessor_jump":
		reason = fact.TargetReason + "_forward_single_predecessor_jump"
	default:
		reason = fact.TargetReason
	}
	if fact.TargetZero {
		addPendingLocalBoolFact(pendingLabelZeroFacts, label, fact.Local)
		s.decisions = append(s.decisions, PassDecision{Action: "propagated_comparison_path_local_zero_target", Caller: fnName, Site: branchIndex, Reason: reason})
		return
	}
	addPendingLocalBoolFact(pendingLabelNonZeroFacts, label, fact.Local)
	s.decisions = append(s.decisions, PassDecision{Action: "propagated_comparison_path_local_nonzero_target", Caller: fnName, Site: branchIndex, Reason: reason})
}

func labelHasNoFallthroughPredecessor(instrs []ir.IRInstr, labelIndex int) bool {
	if labelIndex == 0 {
		return true
	}
	switch instrs[labelIndex-1].Kind {
	case ir.IRJmp, ir.IRReturn:
		return true
	default:
		return false
	}
}

func cloneKnownLocalConsts(knownLocals map[int]int32) map[int]int32 {
	out := make(map[int]int32, len(knownLocals))
	for local, value := range knownLocals {
		out[local] = value
	}
	return out
}

func addPendingLocalBoolFact(pending map[int]map[int]bool, label int, local int) {
	facts, ok := pending[label]
	if !ok {
		facts = map[int]bool{}
		pending[label] = facts
	}
	facts[local] = true
}

func applyKnownLocalConsts(knownLocals map[int]int32, knownZeroLocals map[int]bool, knownNonZeroLocals map[int]bool, facts map[int]int32) {
	for local, value := range facts {
		setKnownLocalConst(knownLocals, knownZeroLocals, knownNonZeroLocals, local, value)
	}
}

func applyKnownLocalZeros(knownLocals map[int]int32, knownZeroLocals map[int]bool, knownNonZeroLocals map[int]bool, facts map[int]bool) {
	for local := range facts {
		setKnownLocalZero(knownLocals, knownZeroLocals, knownNonZeroLocals, local)
	}
}

func applyKnownLocalNonZeros(knownLocals map[int]int32, knownZeroLocals map[int]bool, knownNonZeroLocals map[int]bool, facts map[int]bool) {
	for local := range facts {
		setKnownLocalNonZero(knownLocals, knownZeroLocals, knownNonZeroLocals, local)
	}
}

func setKnownLocalConst(knownLocals map[int]int32, knownZeroLocals map[int]bool, knownNonZeroLocals map[int]bool, local int, value int32) {
	knownLocals[local] = value
	delete(knownZeroLocals, local)
	delete(knownNonZeroLocals, local)
	if value == 0 {
		knownZeroLocals[local] = true
	} else {
		knownNonZeroLocals[local] = true
	}
}

func setKnownLocalZero(knownLocals map[int]int32, knownZeroLocals map[int]bool, knownNonZeroLocals map[int]bool, local int) {
	delete(knownLocals, local)
	knownZeroLocals[local] = true
	delete(knownNonZeroLocals, local)
}

func setKnownLocalNonZero(knownLocals map[int]int32, knownZeroLocals map[int]bool, knownNonZeroLocals map[int]bool, local int) {
	delete(knownLocals, local)
	delete(knownZeroLocals, local)
	knownNonZeroLocals[local] = true
}

func deleteKnownLocalFact(knownLocals map[int]int32, knownZeroLocals map[int]bool, knownNonZeroLocals map[int]bool, local int) {
	delete(knownLocals, local)
	delete(knownZeroLocals, local)
	delete(knownNonZeroLocals, local)
}

func clearKnownLocalConsts(knownLocals map[int]int32) {
	for local := range knownLocals {
		delete(knownLocals, local)
	}
}

func clearKnownLocalFacts(knownLocals map[int]int32, knownZeroLocals map[int]bool, knownNonZeroLocals map[int]bool) {
	for local := range knownLocals {
		delete(knownLocals, local)
	}
	for local := range knownZeroLocals {
		delete(knownZeroLocals, local)
	}
	for local := range knownNonZeroLocals {
		delete(knownNonZeroLocals, local)
	}
}

func skipFallthroughUntilLabel(instrs []ir.IRInstr, start int) (next int, pruned int) {
	for next = start; next < len(instrs); next++ {
		if instrs[next].Kind == ir.IRLabel {
			break
		}
		pruned++
	}
	return next, pruned
}
