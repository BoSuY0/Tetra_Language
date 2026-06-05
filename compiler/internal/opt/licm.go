package opt

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

type licmPureInvariantState struct {
	decisions []PassDecision
}

type invariantExpressionCandidate struct {
	start  int
	local  int
	reason string
}

func LICMPureInvariantPass() Pass {
	state := &licmPureInvariantState{}
	return Pass{
		Name:                      "licm-pure-invariant",
		InputKind:                 IRKindStack,
		OutputKind:                IRKindStack,
		InputVerifier:             VerifierLowerVerifyProgram,
		OutputVerifier:            VerifierLowerVerifyProgram,
		RequiredFacts:             []Fact{FactIRVerified, FactBoundsProofs},
		PreservedFacts:            []Fact{FactBoundsProofs},
		InvalidatedFacts:          []Fact{FactLiveness},
		ProofRule:                 ProofRulePreserveBoundsInvalidateLiveness,
		ValidationStrategy:        ValidationTranslation,
		TranslationValidationHook: TranslationHookValidateTranslation,
		ReportOutput:              "licm-pure-invariant.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       state.run,
		Decisions:                 state.reportDecisions,
	}
}

func (s *licmPureInvariantState) run(prog *ir.IRProgram) error {
	if prog == nil {
		return fmt.Errorf("licm-pure-invariant: missing IR program")
	}
	s.decisions = nil
	for i := range prog.Funcs {
		s.rewriteFunc(&prog.Funcs[i])
	}
	return nil
}

func (s *licmPureInvariantState) reportDecisions() []PassDecision {
	return append([]PassDecision(nil), s.decisions...)
}

func (s *licmPureInvariantState) rewriteFunc(fn *ir.IRFunc) {
	instrs := fn.Instrs
	out := make([]ir.IRInstr, 0, len(instrs))
	for i := 0; i < len(instrs); {
		candidate, reason, ok := analyzeSimpleLoop(instrs, i)
		if !ok {
			out = append(out, instrs[i])
			i++
			continue
		}
		if reason != "" {
			s.decisions = append(s.decisions, PassDecision{Action: "not_hoisted", Caller: fn.Name, Site: i, Reason: reason})
			out = append(out, instrs[i])
			i++
			continue
		}
		invariant, reason, ok := findPureInvariantExpression(instrs, candidate)
		if !ok {
			s.decisions = append(s.decisions, PassDecision{Action: "not_hoisted", Caller: fn.Name, Site: i, Reason: reason})
			out = append(out, instrs[i])
			i++
			continue
		}
		hoistedLocal := fn.LocalSlots
		fn.LocalSlots++
		out = append(out, hoistInvariantExpression(instrs, invariant, hoistedLocal)...)
		out = append(out, replaceInvariantExpression(instrs[candidate.labelIndex:candidate.backJump+1], invariant.start-candidate.labelIndex, hoistedLocal)...)
		s.decisions = append(s.decisions, PassDecision{Action: "hoisted", Caller: fn.Name, Site: invariant.start, Reason: invariant.reason})
		i = candidate.backJump + 1
	}
	fn.Instrs = out
}

func findPureInvariantExpression(instrs []ir.IRInstr, loop simpleLoopCandidate) (invariantExpressionCandidate, string, bool) {
	bodyStart := loop.condJump + 1
	bodyEnd := loop.backJump
	for i := bodyStart; i+2 < bodyEnd; i++ {
		if instrs[i].Kind != ir.IRLoadLocal {
			continue
		}
		switch instrs[i+1].Kind {
		case ir.IRConstI32:
			reason, blockReason, ok := pureInvariantExpressionDecision(instrs[i+2].Kind, instrs[i+1].Imm)
			if blockReason != "" {
				return invariantExpressionCandidate{}, blockReason, false
			}
			if !ok {
				continue
			}
			local := instrs[i].Local
			if reason := invariantOperandBlockReason(instrs[bodyStart:bodyEnd], loop, local); reason != "" {
				return invariantExpressionCandidate{}, reason, false
			}
			return invariantExpressionCandidate{start: i, local: local, reason: reason}, "", true
		case ir.IRLoadLocal:
			reason, blockReason, ok := pureKnownLocalInvariantExpressionDecision(instrs[i+2].Kind, instrs[i].Local, instrs[i+1].Local, instrs, loop.labelIndex)
			if blockReason != "" {
				return invariantExpressionCandidate{}, blockReason, false
			}
			if !ok {
				continue
			}
			for _, local := range []int{instrs[i].Local, instrs[i+1].Local} {
				if reason := invariantOperandBlockReason(instrs[bodyStart:bodyEnd], loop, local); reason != "" {
					return invariantExpressionCandidate{}, reason, false
				}
			}
			return invariantExpressionCandidate{start: i, local: instrs[i].Local, reason: reason}, "", true
		}
	}
	return invariantExpressionCandidate{}, "no_pure_invariant_expression", false
}

func pureInvariantExpressionDecision(kind ir.IRInstrKind, constant int32) (reason string, blockReason string, ok bool) {
	switch kind {
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return "pure_invariant_comparison", "", true
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32:
		return "pure_invariant_arithmetic", "", true
	case ir.IRDivI32:
		if constant == 0 || constant == -1 {
			return "", "unsafe_division_denominator", false
		}
		return "pure_invariant_safe_division", "", true
	case ir.IRModI32:
		if constant == 0 || constant == -1 {
			return "", "unsafe_modulo_denominator", false
		}
		return "pure_invariant_safe_modulo", "", true
	default:
		return "", "", false
	}
}

func pureKnownLocalInvariantExpressionDecision(kind ir.IRInstrKind, leftLocal int, rightLocal int, instrs []ir.IRInstr, labelIndex int) (reason string, blockReason string, ok bool) {
	_, leftKnown := knownConstLocalInStraightLinePreheader(instrs, labelIndex, leftLocal)
	right, known := knownConstLocalInStraightLinePreheader(instrs, labelIndex, rightLocal)
	switch kind {
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		if leftKnown || known {
			return "pure_invariant_known_local_comparison", "", true
		}
		return "", "", false
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32:
		if leftKnown || known {
			return "pure_invariant_known_local_arithmetic", "", true
		}
		return "", "", false
	case ir.IRDivI32:
		if !known {
			return "", "", false
		}
		if right == 0 || right == -1 {
			return "", "unsafe_known_local_division_denominator", false
		}
		return "pure_invariant_safe_known_local_division", "", true
	case ir.IRModI32:
		if !known {
			return "", "", false
		}
		if right == 0 || right == -1 {
			return "", "unsafe_known_local_modulo_denominator", false
		}
		return "pure_invariant_safe_known_local_modulo", "", true
	default:
		return "", "", false
	}
}

func knownConstLocalInStraightLinePreheader(instrs []ir.IRInstr, labelIndex int, local int) (int32, bool) {
	for i := labelIndex - 1; i >= 0; i-- {
		instr := instrs[i]
		switch instr.Kind {
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			return 0, false
		case ir.IRStoreLocal:
			if instr.Local != local {
				continue
			}
			if i > 0 && instrs[i-1].Kind == ir.IRConstI32 {
				return instrs[i-1].Imm, true
			}
			return 0, false
		default:
			if clearsCopyFacts(instr.Kind) {
				return 0, false
			}
		}
	}
	return 0, false
}

func invariantOperandBlockReason(loopBody []ir.IRInstr, loop simpleLoopCandidate, local int) string {
	if local == loop.indexLocal {
		return "variant_loop_index_operand"
	}
	if localStoredInRange(loopBody, local) {
		return "loop_stores_invariant_operand"
	}
	return ""
}

func localStoredInRange(instrs []ir.IRInstr, local int) bool {
	for _, instr := range instrs {
		if instr.Kind == ir.IRStoreLocal && instr.Local == local {
			return true
		}
	}
	return false
}

func hoistInvariantExpression(instrs []ir.IRInstr, invariant invariantExpressionCandidate, hoistedLocal int) []ir.IRInstr {
	return []ir.IRInstr{
		instrs[invariant.start],
		instrs[invariant.start+1],
		instrs[invariant.start+2],
		{Kind: ir.IRStoreLocal, Local: hoistedLocal, Pos: instrs[invariant.start+2].Pos},
	}
}

func replaceInvariantExpression(loop []ir.IRInstr, relativeStart int, hoistedLocal int) []ir.IRInstr {
	out := make([]ir.IRInstr, 0, len(loop)-2)
	for i := 0; i < len(loop); i++ {
		if i == relativeStart {
			out = append(out, ir.IRInstr{Kind: ir.IRLoadLocal, Local: hoistedLocal, Pos: loop[i].Pos})
			i += 2
			continue
		}
		out = append(out, loop[i])
	}
	return out
}
