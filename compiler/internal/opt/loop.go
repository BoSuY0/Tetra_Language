package opt

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
)

type loopCanonicalizationState struct {
	decisions []PassDecision
}

type simpleLoopCandidate struct {
	labelIndex   int
	condJump     int
	backJump     int
	indexLocal   int
	lenLocal     int
	canonicalize bool
}

func LoopCanonicalizationPass() Pass {
	state := &loopCanonicalizationState{}
	return Pass{
		Name:                      "loop-canonicalization",
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
		ReportOutput:              "loop-canonicalization.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       state.run,
		Decisions:                 state.reportDecisions,
	}
}

func (s *loopCanonicalizationState) run(prog *ir.IRProgram) error {
	if prog == nil {
		return fmt.Errorf("loop-canonicalization: missing IR program")
	}
	s.decisions = nil
	for i := range prog.Funcs {
		s.rewriteFunc(&prog.Funcs[i])
	}
	return nil
}

func (s *loopCanonicalizationState) reportDecisions() []PassDecision {
	return append([]PassDecision(nil), s.decisions...)
}

func (s *loopCanonicalizationState) rewriteFunc(fn *ir.IRFunc) {
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
		hoistedLocal := fn.LocalSlots
		fn.LocalSlots++
		out = append(out, rewriteSimpleLoop(instrs[candidate.labelIndex:candidate.backJump+1], candidate, hoistedLocal)...)
		action := "hoisted"
		decisionReason := "stable_len_load"
		if candidate.canonicalize {
			action = "canonicalized"
			decisionReason = "stable_len_le_minus_one_to_lt"
		}
		s.decisions = append(s.decisions, PassDecision{Action: action, Caller: fn.Name, Site: i, Reason: decisionReason})
		i = candidate.backJump + 1
	}
	fn.Instrs = out
}

func analyzeSimpleLoop(instrs []ir.IRInstr, labelIndex int) (simpleLoopCandidate, string, bool) {
	if labelIndex < 0 || labelIndex >= len(instrs) || instrs[labelIndex].Kind != ir.IRLabel {
		return simpleLoopCandidate{}, "", false
	}
	label := instrs[labelIndex].Label
	backJump := -1
	for i := labelIndex + 1; i < len(instrs); i++ {
		if instrs[i].Kind == ir.IRJmp && instrs[i].Label == label {
			backJump = i
			break
		}
	}
	if backJump < 0 {
		return simpleLoopCandidate{}, "", false
	}
	candidate, ok := matchLoopCondition(instrs, labelIndex, backJump)
	if !ok {
		return simpleLoopCandidate{}, "", false
	}
	if !loopHasWhileProofLoad(instrs[candidate.condJump+1 : backJump]) {
		return candidate, "missing_while_bounds_proof", true
	}
	if reason := loopMutationReason(instrs[candidate.condJump+1:backJump], candidate.lenLocal); reason != "" {
		return candidate, reason, true
	}
	return candidate, "", true
}

func matchLoopCondition(instrs []ir.IRInstr, labelIndex int, backJump int) (simpleLoopCandidate, bool) {
	if labelIndex+4 < len(instrs) && labelIndex+4 < backJump &&
		instrs[labelIndex+1].Kind == ir.IRLoadLocal &&
		instrs[labelIndex+2].Kind == ir.IRLoadLocal &&
		instrs[labelIndex+3].Kind == ir.IRCmpLtI32 &&
		instrs[labelIndex+4].Kind == ir.IRJmpIfZero {
		return simpleLoopCandidate{
			labelIndex: labelIndex,
			condJump:   labelIndex + 4,
			backJump:   backJump,
			indexLocal: instrs[labelIndex+1].Local,
			lenLocal:   instrs[labelIndex+2].Local,
		}, true
	}
	if labelIndex+6 < len(instrs) && labelIndex+6 < backJump &&
		instrs[labelIndex+1].Kind == ir.IRLoadLocal &&
		instrs[labelIndex+2].Kind == ir.IRLoadLocal &&
		instrs[labelIndex+3].Kind == ir.IRConstI32 &&
		instrs[labelIndex+3].Imm == 1 &&
		instrs[labelIndex+4].Kind == ir.IRSubI32 &&
		instrs[labelIndex+5].Kind == ir.IRCmpLeI32 &&
		instrs[labelIndex+6].Kind == ir.IRJmpIfZero {
		return simpleLoopCandidate{
			labelIndex:   labelIndex,
			condJump:     labelIndex + 6,
			backJump:     backJump,
			indexLocal:   instrs[labelIndex+1].Local,
			lenLocal:     instrs[labelIndex+2].Local,
			canonicalize: true,
		}, true
	}
	return simpleLoopCandidate{}, false
}

func rewriteSimpleLoop(loop []ir.IRInstr, candidate simpleLoopCandidate, hoistedLocal int) []ir.IRInstr {
	if len(loop) == 0 {
		return nil
	}
	lenLoad := loop[2]
	out := []ir.IRInstr{
		{Kind: ir.IRLoadLocal, Local: candidate.lenLocal, Pos: lenLoad.Pos},
		{Kind: ir.IRStoreLocal, Local: hoistedLocal, Pos: lenLoad.Pos},
	}
	if candidate.canonicalize {
		label := loop[0]
		indexLoad := loop[1]
		hoistedLenLoad := loop[2]
		hoistedLenLoad.Local = hoistedLocal
		cmp := loop[5]
		cmp.Kind = ir.IRCmpLtI32
		out = append(out, label, indexLoad, hoistedLenLoad, cmp, loop[6])
		out = append(out, replaceLoopLenLoads(loop[7:], candidate.lenLocal, hoistedLocal)...)
		return out
	}
	out = append(out, replaceLoopLenLoads(loop, candidate.lenLocal, hoistedLocal)...)
	return out
}

func replaceLoopLenLoads(instrs []ir.IRInstr, lenLocal int, hoistedLocal int) []ir.IRInstr {
	out := append([]ir.IRInstr(nil), instrs...)
	for i := range out {
		if out[i].Kind == ir.IRLoadLocal && out[i].Local == lenLocal {
			out[i].Local = hoistedLocal
		}
	}
	return out
}

func loopHasWhileProofLoad(instrs []ir.IRInstr) bool {
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
			if strings.HasPrefix(instr.ProofID, "proof:while:") {
				return true
			}
		}
	}
	return false
}

func loopMutationReason(instrs []ir.IRInstr, lenLocal int) string {
	for _, instr := range instrs {
		if instr.Kind == ir.IRStoreLocal && instr.Local == lenLocal {
			return "loop_stores_len_local"
		}
		if loopHasUnknownMutation(instr.Kind) {
			return "loop_has_unknown_mutation"
		}
	}
	return ""
}

func loopHasUnknownMutation(kind ir.IRInstrKind) bool {
	if clearsCopyFacts(kind) {
		return true
	}
	switch kind {
	case ir.IRWrite, ir.IRStrLit,
		ir.IRAllocBytes, ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32,
		ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32,
		ir.IRRawSliceFromParts, ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix,
		ir.IRIslandNew, ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16,
		ir.IRIslandMakeSliceI32, ir.IRIslandFree, ir.IRIslandReset:
		return true
	default:
		return false
	}
}
