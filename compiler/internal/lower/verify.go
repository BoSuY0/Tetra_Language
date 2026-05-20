package lower

import (
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/runtimeabi"
)

// IR verifier invariants:
//   - program main metadata names an existing function and function names are unique;
//   - function slot metadata is non-negative and parameters fit inside locals;
//   - branch labels are non-negative and every branch target names a label in
//     the same function;
//   - all control-flow paths entering an instruction agree on stack height;
//   - each instruction has enough input stack slots and leaves a non-negative stack;
//   - local loads/stores reference slots inside IRFunc.LocalSlots;
//   - global loads/stores reference non-negative lowered data slots;
//   - returns see exactly IRFunc.ReturnSlots values on the stack;
//   - calls declare non-negative argument and return slot counts and match
//     known in-program function signatures and known runtime ABI signatures;
//   - policy-protected functions carry the expected budget/consent guard
//     shape before backend codegen.
//
// The verifier is intentionally target-neutral: semantic type safety has already
// been checked, and new IRInstrKind values must update stackEffect here before
// they can reach backend codegen.
func VerifyProgram(prog *ir.IRProgram) error {
	if prog == nil {
		return irVerifierError("ir verifier: missing program")
	}
	if len(prog.Funcs) > 0 {
		if prog.MainIndex < 0 || prog.MainIndex >= len(prog.Funcs) {
			return irVerifierError("ir verifier: main index %d out of bounds (funcs=%d)", prog.MainIndex, len(prog.Funcs))
		}
		if prog.MainName == "" {
			return irVerifierError("ir verifier: missing main name")
		}
		if got := prog.Funcs[prog.MainIndex].Name; got != prog.MainName {
			return irVerifierError("ir verifier: main metadata mismatch: index %d names %q, want %q", prog.MainIndex, got, prog.MainName)
		}
	}
	funcSigs := make(map[string]ir.IRFunc, len(prog.Funcs))
	for _, fn := range prog.Funcs {
		if fn.Name == "" {
			return irVerifierError("ir verifier: function with empty name")
		}
		if _, exists := funcSigs[fn.Name]; exists {
			return irVerifierError("ir verifier: duplicate function name %q", fn.Name)
		}
		funcSigs[fn.Name] = fn
	}
	for _, fn := range prog.Funcs {
		if err := VerifyFunc(fn); err != nil {
			return err
		}
		if err := verifyKnownCallSignatures(fn, funcSigs); err != nil {
			return err
		}
	}
	return nil
}

func VerifyFunc(fn ir.IRFunc) error {
	if fn.ParamSlots < 0 || fn.LocalSlots < 0 || fn.ReturnSlots < 0 {
		return irVerifierError("ir verifier: %s has negative slot metadata params=%d locals=%d returns=%d", fn.Name, fn.ParamSlots, fn.LocalSlots, fn.ReturnSlots)
	}
	if fn.ParamSlots > fn.LocalSlots {
		return irVerifierError("ir verifier: %s param slots %d exceed locals %d", fn.Name, fn.ParamSlots, fn.LocalSlots)
	}
	labels := make(map[int]int)
	for i, instr := range fn.Instrs {
		if instr.Kind != ir.IRLabel {
			continue
		}
		if instr.Label < 0 {
			return verifyError(fn, i, "negative label %d", instr.Label)
		}
		if _, exists := labels[instr.Label]; exists {
			return verifyError(fn, i, "duplicate label %d", instr.Label)
		}
		labels[instr.Label] = i
	}
	for i, instr := range fn.Instrs {
		if _, _, known := stackEffect(instr); !known {
			return verifyError(fn, i, "unknown instruction kind %d", instr.Kind)
		}
		switch instr.Kind {
		case ir.IRJmp, ir.IRJmpIfZero:
			if instr.Label < 0 {
				return verifyError(fn, i, "negative label %d", instr.Label)
			}
			if _, ok := labels[instr.Label]; !ok {
				return verifyError(fn, i, "unknown label %d", instr.Label)
			}
		case ir.IRLoadLocal, ir.IRStoreLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return verifyError(fn, i, "local slot %d out of bounds (locals=%d)", instr.Local, fn.LocalSlots)
			}
		case ir.IRLoadGlobal, ir.IRStoreGlobal:
			if instr.Local < 0 {
				return verifyError(fn, i, "global slot %d out of bounds", instr.Local)
			}
		case ir.IRCall:
			if instr.Name == "" {
				return verifyError(fn, i, "call is missing target name")
			}
			if instr.ArgSlots < 0 || instr.RetSlots < 0 {
				return verifyError(fn, i, "call %q has negative ABI slots args=%d rets=%d", instr.Name, instr.ArgSlots, instr.RetSlots)
			}
			if sig, ok := runtimeabi.SignatureForSymbol(instr.Name); ok && (instr.ArgSlots != sig.ParamSlots || instr.RetSlots != sig.ReturnSlots) {
				return verifyError(fn, i, "runtime call %q ABI mismatch args=%d rets=%d want args=%d rets=%d", instr.Name, instr.ArgSlots, instr.RetSlots, sig.ParamSlots, sig.ReturnSlots)
			}
		case ir.IRSymAddr:
			if instr.Name == "" {
				return verifyError(fn, i, "symbol address is missing name")
			}
		}
	}

	if err := verifyPolicyGuardMetadata(fn, labels); err != nil {
		return err
	}

	if len(fn.Instrs) == 0 {
		if fn.ReturnSlots != 0 {
			return irVerifierError("ir verifier: %s empty body cannot produce %d return slots", fn.Name, fn.ReturnSlots)
		}
		return nil
	}

	heights := make([]int, len(fn.Instrs))
	seen := make([]bool, len(fn.Instrs))
	work := []stackState{{idx: 0, height: 0}}
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			if cur.height != 0 {
				return irVerifierError("ir verifier: %s falls off end with stack height %d", fn.Name, cur.height)
			}
			continue
		}
		if seen[cur.idx] {
			if heights[cur.idx] != cur.height {
				return verifyError(fn, cur.idx, "inconsistent stack height: got %d, previously %d", cur.height, heights[cur.idx])
			}
			continue
		}
		seen[cur.idx] = true
		heights[cur.idx] = cur.height

		instr := fn.Instrs[cur.idx]
		pop, push, known := stackEffect(instr)
		if !known {
			return verifyError(fn, cur.idx, "unknown instruction kind %d", instr.Kind)
		}
		if cur.height < pop {
			return verifyError(fn, cur.idx, "stack underflow: need %d slots, have %d", pop, cur.height)
		}
		nextHeight := cur.height - pop + push
		if nextHeight < 0 {
			return verifyError(fn, cur.idx, "negative stack height %d", nextHeight)
		}

		switch instr.Kind {
		case ir.IRReturn:
			if cur.height != fn.ReturnSlots {
				return verifyError(fn, cur.idx, "return expects %d stack slots, have %d", fn.ReturnSlots, cur.height)
			}
		case ir.IRJmp:
			work = append(work, stackState{idx: labels[instr.Label], height: nextHeight})
		case ir.IRJmpIfZero:
			work = append(work, stackState{idx: labels[instr.Label], height: nextHeight})
			work = append(work, stackState{idx: cur.idx + 1, height: nextHeight})
		default:
			work = append(work, stackState{idx: cur.idx + 1, height: nextHeight})
		}
	}

	if err := verifyPolicyGuardShape(fn, labels, heights, seen); err != nil {
		return err
	}

	if err := verifyLinearEmitterStack(fn); err != nil {
		return err
	}

	return nil
}

type stackState struct {
	idx    int
	height int
}

func verifyError(fn ir.IRFunc, idx int, format string, args ...interface{}) error {
	pos := fn.Instrs[idx].Pos
	fullArgs := append([]interface{}{fn.Name, idx}, args...)
	return irVerifierErrorAt(pos, "ir verifier: %s instr %d: "+format, fullArgs...)
}

func verifyKnownCallSignatures(fn ir.IRFunc, funcSigs map[string]ir.IRFunc) error {
	for i, instr := range fn.Instrs {
		if instr.Kind != ir.IRCall {
			continue
		}
		target, ok := funcSigs[instr.Name]
		if !ok {
			continue
		}
		if instr.ArgSlots != target.ParamSlots || instr.RetSlots != target.ReturnSlots {
			return verifyError(fn, i, "call %q ABI mismatch args=%d rets=%d want args=%d rets=%d", instr.Name, instr.ArgSlots, instr.RetSlots, target.ParamSlots, target.ReturnSlots)
		}
	}
	return nil
}

func verifyLinearEmitterStack(fn ir.IRFunc) error {
	height := 0
	for i, instr := range fn.Instrs {
		pop, push, known := stackEffect(instr)
		if !known {
			return verifyError(fn, i, "unknown instruction kind %d", instr.Kind)
		}
		if instr.Kind == ir.IRReturn {
			if height < fn.ReturnSlots {
				return verifyError(fn, i, "linear return underflow: need %d stack slots, have %d", fn.ReturnSlots, height)
			}
			height = 0
			continue
		}
		if height < pop {
			return verifyError(fn, i, "linear stack underflow: need %d slots, have %d", pop, height)
		}
		height = height - pop + push
		if height < 0 {
			return verifyError(fn, i, "linear negative stack height %d", height)
		}
	}
	return nil
}

func verifyPolicyGuardMetadata(fn ir.IRFunc, labels map[int]int) error {
	policy := fn.Policy
	if !policy.HasBudget && !policy.HasConsent {
		return nil
	}
	if policy.FailLabel < 0 {
		return irVerifierError("ir verifier: %s policy guard missing failure label", fn.Name)
	}
	if _, ok := labels[policy.FailLabel]; !ok {
		return irVerifierError("ir verifier: %s policy guard failure label %d is not defined", fn.Name, policy.FailLabel)
	}
	if policy.HasBudget {
		if policy.BudgetLocal < 0 || policy.BudgetLocal >= fn.LocalSlots {
			return irVerifierError("ir verifier: %s policy budget local %d out of bounds (locals=%d)", fn.Name, policy.BudgetLocal, fn.LocalSlots)
		}
	}
	if policy.HasConsent {
		if policy.ConsentLocal < 0 || policy.ConsentLocal >= fn.LocalSlots {
			return irVerifierError("ir verifier: %s policy consent local %d out of bounds (locals=%d)", fn.Name, policy.ConsentLocal, fn.LocalSlots)
		}
		if policy.ConsentLocal >= fn.ParamSlots {
			return irVerifierError("ir verifier: %s policy consent local %d is not a parameter slot (params=%d)", fn.Name, policy.ConsentLocal, fn.ParamSlots)
		}
	}
	return nil
}

func verifyPolicyGuardShape(fn ir.IRFunc, labels map[int]int, heights []int, seen []bool) error {
	policy := fn.Policy
	if !policy.HasBudget && !policy.HasConsent {
		return nil
	}

	next := 0
	if policy.HasBudget {
		if !matchesBudgetInitializerAt(fn, next, policy) {
			return policyShapeError(fn, next, "malformed budget initializer")
		}
		next += 2
	}
	if policy.HasConsent {
		if !matchesConsentGuardAt(fn, next, policy) {
			return policyShapeError(fn, next, "malformed consent guard")
		}
		next += 4
	}

	if !policy.HasBudget {
		return nil
	}
	failIdx := labels[policy.FailLabel]
	for i, instr := range fn.Instrs {
		if i >= failIdx {
			break
		}
		cost, ok := budgetChargeForInstr(instr.Kind)
		if !ok {
			continue
		}
		if matchesBudgetGuardBefore(fn, i, 0, policy, cost) {
			continue
		}
		if seen[i] && heights[i] > 0 && matchesBudgetGuardBefore(fn, i, heights[i], policy, cost) {
			continue
		}
		return verifyError(fn, i, "missing budget guard before charged instruction")
	}
	return nil
}

func policyShapeError(fn ir.IRFunc, idx int, message string) error {
	if idx >= 0 && idx < len(fn.Instrs) {
		return verifyError(fn, idx, message)
	}
	return irVerifierError("ir verifier: %s %s", fn.Name, message)
}

func matchesBudgetInitializerAt(fn ir.IRFunc, idx int, policy ir.IRPolicy) bool {
	return idx+1 < len(fn.Instrs) &&
		fn.Instrs[idx].Kind == ir.IRConstI32 &&
		fn.Instrs[idx].Imm == policy.Budget &&
		fn.Instrs[idx+1].Kind == ir.IRStoreLocal &&
		fn.Instrs[idx+1].Local == policy.BudgetLocal
}

func matchesConsentGuardAt(fn ir.IRFunc, idx int, policy ir.IRPolicy) bool {
	return idx+3 < len(fn.Instrs) &&
		fn.Instrs[idx].Kind == ir.IRLoadLocal &&
		fn.Instrs[idx].Local == policy.ConsentLocal &&
		fn.Instrs[idx+1].Kind == ir.IRConstI32 &&
		fn.Instrs[idx+1].Imm == consentTokenRuntimeSentinel &&
		fn.Instrs[idx+2].Kind == ir.IRCmpEqI32 &&
		fn.Instrs[idx+3].Kind == ir.IRJmpIfZero &&
		fn.Instrs[idx+3].Label == policy.FailLabel
}

func matchesBudgetGuardBefore(fn ir.IRFunc, chargedIdx int, preservedDepth int, policy ir.IRPolicy, cost int32) bool {
	loadStart := chargedIdx
	if preservedDepth > 0 {
		loadStart = chargedIdx - preservedDepth
		if loadStart < 0 {
			return false
		}
		base := fn.Instrs[loadStart].Local
		for i := 0; i < preservedDepth; i++ {
			instr := fn.Instrs[loadStart+i]
			if instr.Kind != ir.IRLoadLocal || instr.Local != base+i {
				return false
			}
		}
		storeStart := loadStart - budgetGuardInstrs - preservedDepth
		if storeStart < 0 {
			return false
		}
		for i := 0; i < preservedDepth; i++ {
			instr := fn.Instrs[storeStart+i]
			if instr.Kind != ir.IRStoreLocal || instr.Local != base+preservedDepth-1-i {
				return false
			}
		}
	}
	guardStart := loadStart - budgetGuardInstrs
	return matchesBudgetGuardAt(fn, guardStart, policy, cost)
}

const budgetGuardInstrs = 8

func matchesBudgetGuardAt(fn ir.IRFunc, idx int, policy ir.IRPolicy, cost int32) bool {
	return idx >= 0 &&
		idx+budgetGuardInstrs-1 < len(fn.Instrs) &&
		fn.Instrs[idx].Kind == ir.IRLoadLocal &&
		fn.Instrs[idx].Local == policy.BudgetLocal &&
		fn.Instrs[idx+1].Kind == ir.IRConstI32 &&
		fn.Instrs[idx+1].Imm == cost &&
		fn.Instrs[idx+2].Kind == ir.IRSubI32 &&
		fn.Instrs[idx+3].Kind == ir.IRStoreLocal &&
		fn.Instrs[idx+3].Local == policy.BudgetLocal &&
		fn.Instrs[idx+4].Kind == ir.IRLoadLocal &&
		fn.Instrs[idx+4].Local == policy.BudgetLocal &&
		fn.Instrs[idx+5].Kind == ir.IRConstI32 &&
		fn.Instrs[idx+5].Imm == 0 &&
		fn.Instrs[idx+6].Kind == ir.IRCmpGeI32 &&
		fn.Instrs[idx+7].Kind == ir.IRJmpIfZero &&
		fn.Instrs[idx+7].Label == policy.FailLabel
}

func stackEffect(instr ir.IRInstr) (pop int, push int, known bool) {
	switch instr.Kind {
	case ir.IRWrite:
		return 2, 0, true
	case ir.IRStrLit:
		return 0, 2, true
	case ir.IRConstI32, ir.IRLoadLocal, ir.IRLoadGlobal:
		return 0, 1, true
	case ir.IRStoreLocal, ir.IRStoreGlobal:
		return 1, 0, true
	case ir.IRAddI32, ir.IRSubI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
		ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return 2, 1, true
	case ir.IRNegI32:
		return 1, 1, true
	case ir.IRCall:
		return instr.ArgSlots, instr.RetSlots, true
	case ir.IRLabel, ir.IRJmp:
		return 0, 0, true
	case ir.IRJmpIfZero:
		return 1, 0, true
	case ir.IRReturn:
		return 0, 0, true
	case ir.IRAllocBytes, ir.IRIslandNew:
		return 1, 1, true
	case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32:
		return 1, 2, true
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16:
		return 3, 1, true
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		return 4, 0, true
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		return 2, 2, true
	case ir.IRIslandFree:
		return 1, 0, true
	case ir.IRCapIO, ir.IRCapMem, ir.IRSymAddr:
		return 0, 1, true
	case ir.IRMemReadI32, ir.IRMemReadU8, ir.IRMemReadPtr, ir.IRMmioReadI32:
		return 2, 1, true
	case ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRPtrAdd,
		ir.IRMmioWriteI32, ir.IRCtxSwitch:
		return 3, 1, true
	case ir.IRMemReadI32Offset, ir.IRMemReadU8Offset, ir.IRMemReadPtrOffset:
		return 3, 1, true
	case ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset:
		return 4, 1, true
	default:
		return 0, 0, false
	}
}
