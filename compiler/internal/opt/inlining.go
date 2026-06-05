package opt

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

const inlineSmallPureMaxBodyInstrs = 8

type inlineSmallPureState struct {
	decisions []PassDecision
}

type inlineCandidate struct {
	fn         ir.IRFunc
	ok         bool
	reason     string
	bodyInstrs int
}

func InlineSmallPurePass() Pass {
	state := &inlineSmallPureState{}
	return Pass{
		Name:                      "inline-small-pure",
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
		ReportOutput:              "inline-small-pure.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       state.run,
		Decisions:                 state.reportDecisions,
	}
}

func (s *inlineSmallPureState) run(prog *ir.IRProgram) error {
	if prog == nil {
		return fmt.Errorf("inline-small-pure: missing IR program")
	}
	s.decisions = nil
	funcs := make(map[string]ir.IRFunc, len(prog.Funcs))
	candidates := make(map[string]inlineCandidate, len(prog.Funcs))
	for _, fn := range prog.Funcs {
		funcs[fn.Name] = fn
		candidates[fn.Name] = analyzeInlineCandidate(fn)
	}
	promoteInlineWrapperCandidates(funcs, candidates)
	for i := range prog.Funcs {
		fn := &prog.Funcs[i]
		localSlots := fn.LocalSlots
		instrs := append([]ir.IRInstr(nil), fn.Instrs...)
		for {
			next := make([]ir.IRInstr, 0, len(instrs))
			changed := false
			for site, instr := range instrs {
				replacement, ok := s.rewriteInlineCall(fn.Name, instr, site, funcs, candidates, &localSlots)
				if ok {
					next = append(next, replacement...)
					changed = true
					continue
				}
				next = append(next, instr)
			}
			instrs = next
			if !changed {
				break
			}
		}
		fn.LocalSlots = localSlots
		fn.Instrs = instrs
	}
	return nil
}

func (s *inlineSmallPureState) reportDecisions() []PassDecision {
	return append([]PassDecision(nil), s.decisions...)
}

func (s *inlineSmallPureState) rewriteInlineCall(caller string, instr ir.IRInstr, site int, funcs map[string]ir.IRFunc, candidates map[string]inlineCandidate, localSlots *int) ([]ir.IRInstr, bool) {
	if instr.Kind != ir.IRCall {
		return nil, false
	}
	target, ok := funcs[instr.Name]
	if !ok {
		s.notInlined(caller, instr.Name, site, "external_or_runtime")
		return nil, false
	}
	if caller == instr.Name {
		s.notInlined(caller, instr.Name, site, "recursive")
		return nil, false
	}
	candidate := candidates[target.Name]
	if !candidate.ok {
		s.notInlined(caller, instr.Name, site, candidate.reason)
		return nil, false
	}
	if instr.ArgSlots != target.ParamSlots || instr.RetSlots != target.ReturnSlots {
		s.notInlined(caller, instr.Name, site, "signature_mismatch")
		return nil, false
	}
	base := *localSlots
	*localSlots += target.LocalSlots
	replacement := make([]ir.IRInstr, 0, target.ParamSlots+len(target.Instrs)-1)
	for p := target.ParamSlots - 1; p >= 0; p-- {
		replacement = append(replacement, ir.IRInstr{Kind: ir.IRStoreLocal, Local: base + p, Pos: instr.Pos})
	}
	for _, calleeInstr := range target.Instrs[:len(target.Instrs)-1] {
		replacement = append(replacement, remapInlineLocal(calleeInstr, base))
	}
	s.inlined(caller, instr.Name, site, candidate.reason)
	return replacement, true
}

func (s *inlineSmallPureState) inlined(caller string, callee string, site int, reason string) {
	s.decisions = append(s.decisions, PassDecision{
		Action: "inlined",
		Caller: caller,
		Callee: callee,
		Site:   site,
		Reason: reason,
	})
}

func (s *inlineSmallPureState) notInlined(caller string, callee string, site int, reason string) {
	s.decisions = append(s.decisions, PassDecision{
		Action: "not_inlined",
		Caller: caller,
		Callee: callee,
		Site:   site,
		Reason: reason,
	})
}

func analyzeInlineCandidate(fn ir.IRFunc) inlineCandidate {
	candidate := inlineCandidate{fn: fn}
	if fn.Policy.HasBudget || fn.Policy.HasConsent {
		candidate.reason = "unsupported_effect"
		return candidate
	}
	if fn.ReturnSlots != 1 {
		candidate.reason = "unsupported_return_slots"
		return candidate
	}
	if len(fn.Instrs) == 0 || fn.Instrs[len(fn.Instrs)-1].Kind != ir.IRReturn {
		candidate.reason = "control_flow"
		return candidate
	}
	if len(fn.Instrs)-1 > inlineSmallPureMaxBodyInstrs {
		candidate.reason = "not_small"
		return candidate
	}
	candidate.bodyInstrs = len(fn.Instrs) - 1
	for _, instr := range fn.Instrs[:len(fn.Instrs)-1] {
		if inlineInstrTouchesProof(instr) {
			candidate.reason = "proof_sensitive"
			return candidate
		}
		switch instr.Kind {
		case ir.IRConstI32, ir.IRLoadLocal, ir.IRStoreLocal,
			ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRNegI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
			ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		case ir.IRCall:
			candidate.reason = "callee_contains_call"
			return candidate
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			candidate.reason = "control_flow"
			return candidate
		default:
			candidate.reason = "unsupported_effect"
			return candidate
		}
	}
	candidate.ok = true
	candidate.reason = "small_pure"
	return candidate
}

func promoteInlineWrapperCandidates(funcs map[string]ir.IRFunc, candidates map[string]inlineCandidate) {
	for changed := true; changed; {
		changed = false
		for name, fn := range funcs {
			if candidates[name].ok {
				continue
			}
			candidate, ok := analyzeInlineWrapperCandidate(fn, candidates)
			if !ok {
				continue
			}
			candidates[name] = candidate
			changed = true
		}
	}
}

func analyzeInlineWrapperCandidate(fn ir.IRFunc, candidates map[string]inlineCandidate) (inlineCandidate, bool) {
	candidate := inlineCandidate{fn: fn}
	if fn.Policy.HasBudget || fn.Policy.HasConsent {
		candidate.reason = "unsupported_effect"
		return candidate, false
	}
	if fn.ReturnSlots != 1 {
		candidate.reason = "unsupported_return_slots"
		return candidate, false
	}
	if len(fn.Instrs) == 0 || fn.Instrs[len(fn.Instrs)-1].Kind != ir.IRReturn {
		candidate.reason = "control_flow"
		return candidate, false
	}
	bodyInstrs := 0
	hasInlineCall := false
	for _, instr := range fn.Instrs[:len(fn.Instrs)-1] {
		if inlineInstrTouchesProof(instr) {
			candidate.reason = "proof_sensitive"
			return candidate, false
		}
		switch instr.Kind {
		case ir.IRConstI32, ir.IRLoadLocal, ir.IRStoreLocal,
			ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRNegI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
			ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			bodyInstrs++
		case ir.IRCall:
			callee, ok := candidates[instr.Name]
			if !ok || !callee.ok {
				candidate.reason = "callee_contains_call"
				return candidate, false
			}
			bodyInstrs += callee.bodyInstrs
			hasInlineCall = true
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			candidate.reason = "control_flow"
			return candidate, false
		default:
			candidate.reason = "unsupported_effect"
			return candidate, false
		}
	}
	if !hasInlineCall {
		return candidate, false
	}
	if bodyInstrs > inlineSmallPureMaxBodyInstrs {
		candidate.reason = "not_small"
		return candidate, false
	}
	candidate.ok = true
	candidate.reason = "small_pure_wrapper"
	candidate.bodyInstrs = bodyInstrs
	return candidate, true
}

func inlineInstrTouchesProof(instr ir.IRInstr) bool {
	if instr.ProofID != "" {
		return true
	}
	switch instr.Kind {
	case ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
		return true
	default:
		return false
	}
}

func remapInlineLocal(instr ir.IRInstr, base int) ir.IRInstr {
	switch instr.Kind {
	case ir.IRLoadLocal, ir.IRStoreLocal:
		instr.Local += base
	}
	return instr
}
