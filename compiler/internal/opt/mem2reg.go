package opt

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

type mem2regState struct {
	decisions []PassDecision
}

type mem2regProducerKind string

const (
	mem2regProducerSingleValue                          mem2regProducerKind = "single_value"
	mem2regProducerComparisonExpression                 mem2regProducerKind = "comparison_expression"
	mem2regProducerSafeConstUnaryNegExpression          mem2regProducerKind = "safe_const_unary_neg_expression"
	mem2regProducerSafeKnownLocalUnaryNegExpression     mem2regProducerKind = "safe_known_local_unary_neg_expression"
	mem2regProducerSafeConstArithmeticExpression        mem2regProducerKind = "safe_const_arithmetic_expression"
	mem2regProducerSafeKnownLocalArithmeticExpression   mem2regProducerKind = "safe_known_local_arithmetic_expression"
	mem2regProducerSafeConstDenominatorDivModExpression mem2regProducerKind = "safe_const_denominator_divmod_expression"
	mem2regProducerSafeKnownLocalDivModExpression       mem2regProducerKind = "safe_known_local_divmod_expression"
)

type mem2regProducer struct {
	Kind         mem2regProducerKind
	Instrs       []ir.IRInstr
	SourceLocals map[int]struct{}
}

func Mem2RegPass() Pass {
	state := &mem2regState{}
	return Pass{
		Name:                      "mem2reg-single-assignment",
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
		ReportOutput:              "mem2reg-single-assignment.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       state.run,
		Decisions:                 state.reportDecisions,
	}
}

func (s *mem2regState) run(prog *ir.IRProgram) error {
	if prog == nil {
		return fmt.Errorf("mem2reg-single-assignment: missing IR program")
	}
	s.decisions = nil
	for i := range prog.Funcs {
		fn := &prog.Funcs[i]
		if fn.Policy.HasBudget || fn.Policy.HasConsent {
			s.decisions = append(s.decisions, PassDecision{Action: "not_promoted", Caller: fn.Name, Site: 0, Reason: "policy_guarded_function"})
			continue
		}
		if hasMem2RegControlFlow(fn.Instrs) {
			if hasAdjacentStoreLoad(fn.Instrs) {
				s.decisions = append(s.decisions, PassDecision{Action: "not_promoted", Caller: fn.Name, Site: 0, Reason: "control_flow_function"})
			}
			continue
		}
		fn.Instrs = s.rewriteFunc(fn.Name, fn.ParamSlots, fn.Instrs)
	}
	return nil
}

func (s *mem2regState) reportDecisions() []PassDecision {
	return append([]PassDecision(nil), s.decisions...)
}

func (s *mem2regState) rewriteFunc(fnName string, paramSlots int, instrs []ir.IRInstr) []ir.IRInstr {
	stores, loads := localUsageCounts(instrs)
	loadIndexes := localLoadIndexes(instrs)
	replacements := map[int][]ir.IRInstr{}
	out := make([]ir.IRInstr, 0, len(instrs))
	for i := 0; i < len(instrs); i++ {
		if replacement, ok := replacements[i]; ok {
			out = append(out, replacement...)
			continue
		}
		instr := instrs[i]
		if i+1 < len(instrs) && instr.Kind == ir.IRStoreLocal && instrs[i+1].Kind == ir.IRLoadLocal && instr.Local == instrs[i+1].Local {
			local := instr.Local
			if reason := mem2regPromotionBlockReason(local, paramSlots, stores, loads); reason != "" {
				s.decisions = append(s.decisions, PassDecision{Action: "not_promoted", Caller: fnName, Site: i, Reason: reason})
			} else {
				s.decisions = append(s.decisions, PassDecision{Action: "promoted_single_assignment_temp", Caller: fnName, Site: i, Reason: "single_store_single_load_adjacent"})
				i++
				continue
			}
		}
		if instr.Kind == ir.IRStoreLocal && i > 0 && !isAdjacentMem2RegStoreLoad(instrs, i) {
			local := instr.Local
			loadIndex, hasLoad := loadIndexes[local]
			if hasLoad && loadIndex > i+1 {
				if reason := mem2regPromotionBlockReason(local, paramSlots, stores, loads); reason != "" {
					s.decisions = append(s.decisions, PassDecision{Action: "not_promoted", Caller: fnName, Site: i, Reason: reason})
				} else if producer, ok := mem2regProducerBeforeStore(instrs, i); !ok {
					s.decisions = append(s.decisions, PassDecision{Action: "not_promoted", Caller: fnName, Site: i, Reason: "producer_not_available"})
				} else {
					if len(out) < len(producer.Instrs) || !sameMem2RegProducerSpan(out[len(out)-len(producer.Instrs):], producer.Instrs) {
						s.decisions = append(s.decisions, PassDecision{Action: "not_promoted", Caller: fnName, Site: i, Reason: "producer_not_available"})
					} else if reason := separatedMem2RegBlockReason(instrs[i+1:loadIndex], producer.SourceLocals, local); reason != "" {
						s.decisions = append(s.decisions, PassDecision{Action: "not_promoted", Caller: fnName, Site: i, Reason: reason})
					} else {
						out = out[:len(out)-len(producer.Instrs)]
						replacements[loadIndex] = append([]ir.IRInstr(nil), producer.Instrs...)
						reason := "single_store_single_load_stack_neutral"
						switch producer.Kind {
						case mem2regProducerComparisonExpression:
							reason = "single_store_single_load_stack_neutral_comparison_expression"
						case mem2regProducerSafeConstUnaryNegExpression:
							reason = "single_store_single_load_stack_neutral_safe_const_unary_neg_expression"
						case mem2regProducerSafeKnownLocalUnaryNegExpression:
							reason = "single_store_single_load_stack_neutral_safe_known_local_unary_neg_expression"
						case mem2regProducerSafeConstArithmeticExpression:
							reason = "single_store_single_load_stack_neutral_safe_const_arithmetic_expression"
						case mem2regProducerSafeKnownLocalArithmeticExpression:
							reason = "single_store_single_load_stack_neutral_safe_known_local_arithmetic_expression"
						case mem2regProducerSafeConstDenominatorDivModExpression:
							reason = "single_store_single_load_stack_neutral_safe_const_denominator_divmod_expression"
						case mem2regProducerSafeKnownLocalDivModExpression:
							reason = "single_store_single_load_stack_neutral_safe_known_local_divmod_expression"
						}
						s.decisions = append(s.decisions, PassDecision{Action: "promoted_single_assignment_temp", Caller: fnName, Site: i, Reason: reason})
						continue
					}
				}
			}
		}
		out = append(out, instr)
	}
	return out
}

func localUsageCounts(instrs []ir.IRInstr) (map[int]int, map[int]int) {
	stores := map[int]int{}
	loads := map[int]int{}
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRStoreLocal:
			stores[instr.Local]++
		case ir.IRLoadLocal:
			loads[instr.Local]++
		}
	}
	return stores, loads
}

func localLoadIndexes(instrs []ir.IRInstr) map[int]int {
	indexes := map[int]int{}
	for i, instr := range instrs {
		if instr.Kind == ir.IRLoadLocal {
			indexes[instr.Local] = i
		}
	}
	return indexes
}

func mem2regPromotionBlockReason(local int, paramSlots int, stores map[int]int, loads map[int]int) string {
	if local < paramSlots {
		return "param_slot"
	}
	if stores[local] != 1 {
		return "local_not_single_store"
	}
	if loads[local] != 1 {
		return "local_not_single_load"
	}
	return ""
}

func isAdjacentMem2RegStoreLoad(instrs []ir.IRInstr, index int) bool {
	return index+1 < len(instrs) &&
		instrs[index].Kind == ir.IRStoreLocal &&
		instrs[index+1].Kind == ir.IRLoadLocal &&
		instrs[index].Local == instrs[index+1].Local
}

func sameMem2RegProducerSpan(left, right []ir.IRInstr) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !sameMem2RegProducerInstr(left[i], right[i]) {
			return false
		}
	}
	return true
}

func sameMem2RegProducerInstr(left, right ir.IRInstr) bool {
	if left.Kind != right.Kind {
		return false
	}
	switch left.Kind {
	case ir.IRConstI32:
		return left.Imm == right.Imm
	case ir.IRLoadLocal:
		return left.Local == right.Local
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32,
		ir.IRNegI32, ir.IRAddI32, ir.IRSubI32, ir.IRMulI32,
		ir.IRDivI32, ir.IRModI32:
		return true
	default:
		return false
	}
}

func mem2regProducerBeforeStore(instrs []ir.IRInstr, storeIndex int) (mem2regProducer, bool) {
	if storeIndex <= 0 {
		return mem2regProducer{}, false
	}
	if isSingleMem2RegProducerValue(instrs[storeIndex-1]) {
		span := instrs[storeIndex-1 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSingleValue,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 2 &&
		isMem2RegSafeConstUnaryNegProducer(instrs[storeIndex-1], instrs[storeIndex-2]) {
		span := instrs[storeIndex-2 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeConstUnaryNegExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 2 &&
		isMem2RegSafeKnownLocalUnaryNegProducer(instrs[storeIndex-1], instrs[storeIndex-2], instrs, storeIndex-2) {
		span := instrs[storeIndex-2 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeKnownLocalUnaryNegExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 3 &&
		isMem2RegComparisonProducer(instrs[storeIndex-1]) &&
		isSingleMem2RegProducerValue(instrs[storeIndex-2]) &&
		isSingleMem2RegProducerValue(instrs[storeIndex-3]) {
		span := instrs[storeIndex-3 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerComparisonExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 3 &&
		isMem2RegSafeConstArithmeticProducer(instrs[storeIndex-1], instrs[storeIndex-3], instrs[storeIndex-2]) {
		span := instrs[storeIndex-3 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeConstArithmeticExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 3 &&
		isMem2RegSafeKnownLocalArithmeticProducer(instrs[storeIndex-1], instrs[storeIndex-3], instrs[storeIndex-2], instrs, storeIndex-3) {
		span := instrs[storeIndex-3 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeKnownLocalArithmeticExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 3 &&
		isMem2RegSafeConstDenominatorDivModProducer(instrs[storeIndex-1], instrs[storeIndex-3], instrs[storeIndex-2]) {
		span := instrs[storeIndex-3 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeConstDenominatorDivModExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	if storeIndex >= 3 &&
		isMem2RegSafeKnownLocalDivModProducer(instrs[storeIndex-1], instrs[storeIndex-3], instrs[storeIndex-2], instrs, storeIndex-3) {
		span := instrs[storeIndex-3 : storeIndex]
		return mem2regProducer{
			Kind:         mem2regProducerSafeKnownLocalDivModExpression,
			Instrs:       append([]ir.IRInstr(nil), span...),
			SourceLocals: mem2regProducerSourceLocals(span),
		}, true
	}
	return mem2regProducer{}, false
}

func isSingleMem2RegProducerValue(instr ir.IRInstr) bool {
	return instr.Kind == ir.IRConstI32 || instr.Kind == ir.IRLoadLocal
}

func isMem2RegComparisonProducer(instr ir.IRInstr) bool {
	switch instr.Kind {
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func isMem2RegSafeConstUnaryNegProducer(op ir.IRInstr, operand ir.IRInstr) bool {
	if op.Kind != ir.IRNegI32 || operand.Kind != ir.IRConstI32 {
		return false
	}
	_, ok := checkedNegI32(operand.Imm)
	return ok
}

func isMem2RegSafeKnownLocalUnaryNegProducer(op ir.IRInstr, operand ir.IRInstr, instrs []ir.IRInstr, beforeIndex int) bool {
	if op.Kind != ir.IRNegI32 || operand.Kind != ir.IRLoadLocal {
		return false
	}
	remove := make([]bool, len(instrs))
	imm, ok := knownConstLocalBefore(instrs, remove, beforeIndex, operand.Local)
	if !ok {
		return false
	}
	_, ok = checkedNegI32(imm)
	return ok
}

func isMem2RegSafeConstArithmeticProducer(op ir.IRInstr, left ir.IRInstr, right ir.IRInstr) bool {
	switch op.Kind {
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32:
		if left.Kind != ir.IRConstI32 || right.Kind != ir.IRConstI32 {
			return false
		}
		_, ok := foldConstBinaryI32(op.Kind, left.Imm, right.Imm)
		return ok
	default:
		return false
	}
}

func isMem2RegSafeKnownLocalArithmeticProducer(op ir.IRInstr, left ir.IRInstr, right ir.IRInstr, instrs []ir.IRInstr, beforeIndex int) bool {
	switch op.Kind {
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32:
	default:
		return false
	}
	if left.Kind != ir.IRLoadLocal && right.Kind != ir.IRLoadLocal {
		return false
	}
	remove := make([]bool, len(instrs))
	leftImm, ok := knownConstOperandBefore(left, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	rightImm, ok := knownConstOperandBefore(right, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	_, ok = foldConstBinaryI32(op.Kind, leftImm, rightImm)
	return ok
}

func isMem2RegSafeConstDenominatorDivModProducer(op ir.IRInstr, left ir.IRInstr, right ir.IRInstr) bool {
	switch op.Kind {
	case ir.IRDivI32, ir.IRModI32:
		return left.Kind == ir.IRLoadLocal && right.Kind == ir.IRConstI32 && right.Imm != 0 && right.Imm != -1
	default:
		return false
	}
}

func isMem2RegSafeKnownLocalDivModProducer(op ir.IRInstr, left ir.IRInstr, right ir.IRInstr, instrs []ir.IRInstr, beforeIndex int) bool {
	switch op.Kind {
	case ir.IRDivI32, ir.IRModI32:
	default:
		return false
	}
	if left.Kind != ir.IRLoadLocal && right.Kind != ir.IRLoadLocal {
		return false
	}
	remove := make([]bool, len(instrs))
	leftImm, ok := knownConstOperandBefore(left, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	rightImm, ok := knownConstOperandBefore(right, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	_, ok = foldConstBinaryI32(op.Kind, leftImm, rightImm)
	return ok
}

func mem2regProducerSourceLocals(instrs []ir.IRInstr) map[int]struct{} {
	locals := map[int]struct{}{}
	for _, instr := range instrs {
		if instr.Kind == ir.IRLoadLocal {
			locals[instr.Local] = struct{}{}
		}
	}
	return locals
}

func separatedMem2RegBlockReason(instrs []ir.IRInstr, sourceLocals map[int]struct{}, tempLocal int) string {
	depth := 0
	for _, instr := range instrs {
		if instr.Kind == ir.IRStoreLocal {
			if instr.Local == tempLocal {
				return "temp_local_modified_before_load"
			}
			if _, ok := sourceLocals[instr.Local]; ok {
				return "source_local_modified_before_load"
			}
		}
		pop, push, ok := mem2regStackEffect(instr)
		if !ok || depth < pop {
			return "intervening_not_stack_neutral"
		}
		depth += push - pop
	}
	if depth != 0 {
		return "intervening_not_stack_neutral"
	}
	return ""
}

func mem2regStackEffect(instr ir.IRInstr) (pop int, push int, ok bool) {
	switch instr.Kind {
	case ir.IRConstI32, ir.IRLoadLocal:
		return 0, 1, true
	case ir.IRStoreLocal:
		return 1, 0, true
	case ir.IRNegI32:
		return 1, 1, true
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32,
		ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return 2, 1, true
	default:
		return 0, 0, false
	}
}

func hasMem2RegControlFlow(instrs []ir.IRInstr) bool {
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			return true
		}
	}
	return false
}

func hasAdjacentStoreLoad(instrs []ir.IRInstr) bool {
	for i := 0; i+1 < len(instrs); i++ {
		if instrs[i].Kind == ir.IRStoreLocal && instrs[i+1].Kind == ir.IRLoadLocal && instrs[i].Local == instrs[i+1].Local {
			return true
		}
	}
	return false
}
