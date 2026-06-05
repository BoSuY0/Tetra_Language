package opt

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

const (
	minInt32 = int64(-1 << 31)
	maxInt32 = int64(1<<31 - 1)
)

func BasicScalarPass() Pass {
	return Pass{
		Name:                      "basic-scalar",
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
		ReportOutput:              "basic-scalar.opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       runBasicScalarPass,
	}
}

func runBasicScalarPass(prog *ir.IRProgram) error {
	if prog == nil {
		return fmt.Errorf("basic-scalar: missing IR program")
	}
	for i := range prog.Funcs {
		fn := &prog.Funcs[i]
		if !basicScalarFuncEligible(*fn) {
			continue
		}
		instrs := append([]ir.IRInstr(nil), fn.Instrs...)
		for iter := 0; iter < 16; iter++ {
			changed := false
			var stepChanged bool
			instrs, stepChanged = foldConstantsAndAlgebra(instrs)
			changed = changed || stepChanged
			instrs, stepChanged = propagateLocalCopies(instrs)
			changed = changed || stepChanged
			instrs, stepChanged = eliminateCommonLocalExpressions(instrs)
			changed = changed || stepChanged
			instrs, stepChanged = eliminateSimpleDeadStores(instrs)
			changed = changed || stepChanged
			if !changed {
				fn.Instrs = instrs
				break
			}
			if iter == 15 {
				return fmt.Errorf("basic-scalar: %s did not converge", fn.Name)
			}
		}
	}
	return nil
}

func basicScalarFuncEligible(fn ir.IRFunc) bool {
	if fn.Policy.HasBudget || fn.Policy.HasConsent {
		return false
	}
	for i, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			return false
		case ir.IRReturn:
			return i == len(fn.Instrs)-1
		}
	}
	return true
}

func foldConstantsAndAlgebra(instrs []ir.IRInstr) ([]ir.IRInstr, bool) {
	out := make([]ir.IRInstr, 0, len(instrs))
	changed := false
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRNegI32:
			if len(out) >= 1 && out[len(out)-1].Kind == ir.IRConstI32 {
				if folded, ok := checkedNegI32(out[len(out)-1].Imm); ok {
					out[len(out)-1] = constInstr(instr, folded)
					changed = true
					continue
				}
			}
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32,
			ir.IRDivI32, ir.IRModI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
			ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			if len(out) >= 2 {
				left := out[len(out)-2]
				right := out[len(out)-1]
				if left.Kind == ir.IRConstI32 && right.Kind == ir.IRConstI32 {
					if folded, ok := foldConstBinaryI32(instr.Kind, left.Imm, right.Imm); ok {
						out = out[:len(out)-2]
						out = append(out, constInstr(instr, folded))
						changed = true
						continue
					}
				}
				if applyAlgebraicSimplification(&out, instr.Kind) {
					changed = true
					continue
				}
			}
		}
		out = append(out, instr)
	}
	return out, changed
}

func applyAlgebraicSimplification(out *[]ir.IRInstr, kind ir.IRInstrKind) bool {
	instrs := *out
	if len(instrs) < 2 {
		return false
	}
	leftIdx := len(instrs) - 2
	rightIdx := len(instrs) - 1
	left := instrs[leftIdx]
	right := instrs[rightIdx]

	switch kind {
	case ir.IRAddI32:
		if isConstI32(right, 0) {
			*out = instrs[:rightIdx]
			return true
		}
		if isConstI32(left, 0) && isSinglePureValue(right) {
			*out = append(instrs[:leftIdx], right)
			return true
		}
	case ir.IRSubI32:
		if isConstI32(right, 0) {
			*out = instrs[:rightIdx]
			return true
		}
	case ir.IRMulI32:
		if isConstI32(right, 1) {
			*out = instrs[:rightIdx]
			return true
		}
		if isConstI32(left, 1) && isSinglePureValue(right) {
			*out = append(instrs[:leftIdx], right)
			return true
		}
		if isConstI32(right, 0) && isSinglePureValue(left) {
			*out = append(instrs[:leftIdx], constInstr(right, 0))
			return true
		}
		if isConstI32(left, 0) && isSinglePureValue(right) {
			*out = append(instrs[:leftIdx], constInstr(left, 0))
			return true
		}
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		if sameSinglePureValue(left, right) {
			*out = append(instrs[:leftIdx], constInstr(right, sameValueComparisonResult(kind)))
			return true
		}
	}
	return false
}

func propagateLocalCopies(instrs []ir.IRInstr) ([]ir.IRInstr, bool) {
	out := make([]ir.IRInstr, 0, len(instrs))
	copies := map[int]int{}
	changed := false
	for _, instr := range instrs {
		switch instr.Kind {
		case ir.IRLoadLocal:
			src := resolveLocalCopy(copies, instr.Local)
			if src != instr.Local {
				instr.Local = src
				changed = true
			}
			out = append(out, instr)
		case ir.IRStoreLocal:
			dst := instr.Local
			invalidateLocalCopies(copies, dst)
			if len(out) > 0 && out[len(out)-1].Kind == ir.IRLoadLocal {
				src := resolveLocalCopy(copies, out[len(out)-1].Local)
				if src != out[len(out)-1].Local {
					out[len(out)-1].Local = src
					changed = true
				}
				if src != dst {
					copies[dst] = src
				}
			}
			out = append(out, instr)
		default:
			if clearsCopyFacts(instr.Kind) {
				clearLocalCopies(copies)
			}
			out = append(out, instr)
		}
	}
	return out, changed
}

func eliminateSimpleDeadStores(instrs []ir.IRInstr) ([]ir.IRInstr, bool) {
	remove := make([]bool, len(instrs))
	live := map[int]bool{}
	changed := false
	for i := len(instrs) - 1; i >= 0; i-- {
		if remove[i] {
			continue
		}
		instr := instrs[i]
		switch instr.Kind {
		case ir.IRLoadLocal:
			live[instr.Local] = true
		case ir.IRStoreLocal:
			if !live[instr.Local] {
				if start := deadStoreProducerStart(instrs, remove, i); start >= 0 {
					for j := start; j <= i; j++ {
						remove[j] = true
					}
					changed = true
					i = start
					continue
				}
			}
			delete(live, instr.Local)
		}
	}
	if !changed {
		return instrs, false
	}
	out := make([]ir.IRInstr, 0, len(instrs))
	for i, instr := range instrs {
		if !remove[i] {
			out = append(out, instr)
		}
	}
	return out, true
}

func deadStoreProducerStart(instrs []ir.IRInstr, remove []bool, storeIndex int) int {
	if storeIndex > 0 && !remove[storeIndex-1] && isDeadStoreProducer(instrs[storeIndex-1]) {
		return storeIndex - 1
	}
	if start := safeKnownLocalUnaryNegDeadStoreProducerStart(instrs, remove, storeIndex); start >= 0 {
		return start
	}
	if storeIndex >= 3 {
		start := storeIndex - 3
		if remove[start] || remove[start+1] || remove[start+2] {
			return -1
		}
		op := instrs[storeIndex-1].Kind
		if isNonTrappingDeadStoreExpressionOp(op) &&
			isSinglePureValue(instrs[start]) &&
			isSinglePureValue(instrs[start+1]) {
			return start
		}
		if isSafeKnownConstArithmeticDeadStoreExpression(op, instrs[start], instrs[start+1], instrs, remove, start) {
			return start
		}
		if isSafeConstDenominatorDeadStoreExpression(op, instrs[start], instrs[start+1]) {
			return start
		}
	}
	return -1
}

func safeKnownLocalUnaryNegDeadStoreProducerStart(instrs []ir.IRInstr, remove []bool, storeIndex int) int {
	if storeIndex < 2 {
		return -1
	}
	start := storeIndex - 2
	if remove[start] || remove[start+1] {
		return -1
	}
	operand := instrs[start]
	op := instrs[start+1]
	if operand.Kind != ir.IRLoadLocal || op.Kind != ir.IRNegI32 {
		return -1
	}
	imm, ok := knownConstLocalBefore(instrs, remove, start, operand.Local)
	if !ok {
		return -1
	}
	if _, ok := checkedNegI32(imm); !ok {
		return -1
	}
	return start
}

func knownConstLocalBefore(instrs []ir.IRInstr, remove []bool, beforeIndex int, local int) (int32, bool) {
	known := map[int]int32{}
	for i := 0; i < beforeIndex; i++ {
		if remove[i] {
			continue
		}
		instr := instrs[i]
		switch instr.Kind {
		case ir.IRStoreLocal:
			delete(known, instr.Local)
			if i > 0 && !remove[i-1] && instrs[i-1].Kind == ir.IRConstI32 {
				known[instr.Local] = instrs[i-1].Imm
			}
		default:
			if clearsCopyFacts(instr.Kind) {
				known = map[int]int32{}
			}
		}
	}
	imm, ok := known[local]
	return imm, ok
}

func isSafeKnownConstArithmeticDeadStoreExpression(kind ir.IRInstrKind, left ir.IRInstr, right ir.IRInstr, instrs []ir.IRInstr, remove []bool, beforeIndex int) bool {
	switch kind {
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32:
	default:
		return false
	}
	leftImm, ok := knownConstOperandBefore(left, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	rightImm, ok := knownConstOperandBefore(right, instrs, remove, beforeIndex)
	if !ok {
		return false
	}
	_, ok = foldConstBinaryI32(kind, leftImm, rightImm)
	return ok
}

func knownConstOperandBefore(instr ir.IRInstr, instrs []ir.IRInstr, remove []bool, beforeIndex int) (int32, bool) {
	switch instr.Kind {
	case ir.IRConstI32:
		return instr.Imm, true
	case ir.IRLoadLocal:
		return knownConstLocalBefore(instrs, remove, beforeIndex, instr.Local)
	default:
		return 0, false
	}
}

func isNonTrappingDeadStoreExpressionOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func isSafeConstDenominatorDeadStoreExpression(kind ir.IRInstrKind, left ir.IRInstr, right ir.IRInstr) bool {
	switch kind {
	case ir.IRDivI32, ir.IRModI32:
		return left.Kind == ir.IRLoadLocal &&
			right.Kind == ir.IRConstI32 &&
			right.Imm != 0 &&
			right.Imm != -1
	default:
		return false
	}
}

type localExprKey struct {
	Kind  ir.IRInstrKind
	Left  localExprOperand
	Right localExprOperand
}

type localUnaryExprKey struct {
	Kind    ir.IRInstrKind
	Operand localExprOperand
}

type localExprOperandKind int

const (
	localExprOperandLocal localExprOperandKind = iota
	localExprOperandConst
)

type localExprOperand struct {
	Kind  localExprOperandKind
	Local int
	Imm   int32
}

func eliminateCommonLocalExpressions(instrs []ir.IRInstr) ([]ir.IRInstr, bool) {
	out := make([]ir.IRInstr, 0, len(instrs))
	exprs := map[localExprKey]int{}
	unaryExprs := map[localUnaryExprKey]int{}
	knownConsts := map[int]int32{}
	changed := false
	for i := 0; i < len(instrs); i++ {
		if i+1 < len(instrs) {
			keys := localUnaryExprKeysFromInstrs(instrs[i+1].Kind, instrs[i], knownConsts)
			if len(keys) > 0 {
				if cachedLocal, ok := cachedLocalForUnaryExprKeys(unaryExprs, keys); ok {
					out = append(out, ir.IRInstr{Kind: ir.IRLoadLocal, Local: cachedLocal, Pos: instrs[i+1].Pos})
					i += 1
					changed = true
					continue
				}
				if i+2 < len(instrs) && instrs[i+2].Kind == ir.IRStoreLocal {
					storableKeys := storableLocalUnaryExprKeys(keys, instrs[i+2].Local)
					if len(storableKeys) > 0 {
						out = append(out, instrs[i], instrs[i+1], instrs[i+2])
						invalidateCachedExpressions(exprs, instrs[i+2].Local)
						invalidateCachedUnaryExpressions(unaryExprs, instrs[i+2].Local)
						updateKnownConstForExpressionStore(knownConsts, instrs[i+1].Kind, instrs[i], instrs[i], instrs[i+2].Local)
						for _, key := range storableKeys {
							unaryExprs[key] = instrs[i+2].Local
						}
						i += 2
						continue
					}
				}
			}
		}
		if i+2 < len(instrs) {
			keys := localExprKeysFromInstrs(instrs[i+2].Kind, instrs[i], instrs[i+1], knownConsts)
			if len(keys) > 0 {
				if cachedLocal, ok := cachedLocalForExprKeys(exprs, keys); ok {
					out = append(out, ir.IRInstr{Kind: ir.IRLoadLocal, Local: cachedLocal, Pos: instrs[i+2].Pos})
					i += 2
					changed = true
					continue
				}
				if i+3 < len(instrs) && instrs[i+3].Kind == ir.IRStoreLocal {
					storableKeys := storableLocalExprKeys(keys, instrs[i+3].Local)
					if len(storableKeys) > 0 {
						out = append(out, instrs[i], instrs[i+1], instrs[i+2], instrs[i+3])
						invalidateCachedExpressions(exprs, instrs[i+3].Local)
						invalidateCachedUnaryExpressions(unaryExprs, instrs[i+3].Local)
						updateKnownConstForExpressionStore(knownConsts, instrs[i+2].Kind, instrs[i], instrs[i+1], instrs[i+3].Local)
						for _, key := range storableKeys {
							exprs[key] = instrs[i+3].Local
						}
						i += 3
						continue
					}
				}
			}
		}
		instr := instrs[i]
		switch instr.Kind {
		case ir.IRStoreLocal:
			invalidateCachedExpressions(exprs, instr.Local)
			invalidateCachedUnaryExpressions(unaryExprs, instr.Local)
			updateKnownConstForStore(knownConsts, instrs, i)
		default:
			if clearsCopyFacts(instr.Kind) {
				clearCachedExpressions(exprs)
				clearCachedUnaryExpressions(unaryExprs)
				clearKnownLocalConsts(knownConsts)
			}
		}
		out = append(out, instr)
	}
	return out, changed
}

func localUnaryExprKeysFromInstrs(kind ir.IRInstrKind, operandInstr ir.IRInstr, knownConsts map[int]int32) []localUnaryExprKey {
	keys := []localUnaryExprKey{}
	if key, ok := localUnaryExprKeyFromInstrs(kind, operandInstr); ok {
		keys = append(keys, key)
	}
	if key, ok := knownConstUnaryExprKeyFromInstrs(kind, operandInstr, knownConsts); ok && !localUnaryExprKeysContain(keys, key) {
		keys = append(keys, key)
	}
	return keys
}

func cachedLocalForUnaryExprKeys(exprs map[localUnaryExprKey]int, keys []localUnaryExprKey) (int, bool) {
	for _, key := range keys {
		if cachedLocal, ok := exprs[key]; ok {
			return cachedLocal, true
		}
	}
	return 0, false
}

func storableLocalUnaryExprKeys(keys []localUnaryExprKey, storeLocal int) []localUnaryExprKey {
	out := make([]localUnaryExprKey, 0, len(keys))
	for _, key := range keys {
		if !localUnaryExprKeyUsesLocal(key, storeLocal) {
			out = append(out, key)
		}
	}
	return out
}

func localUnaryExprKeysContain(keys []localUnaryExprKey, needle localUnaryExprKey) bool {
	for _, key := range keys {
		if key == needle {
			return true
		}
	}
	return false
}

func knownConstUnaryExprKeyFromInstrs(kind ir.IRInstrKind, operandInstr ir.IRInstr, knownConsts map[int]int32) (localUnaryExprKey, bool) {
	if kind != ir.IRNegI32 {
		return localUnaryExprKey{}, false
	}
	operand, knownLocal, ok := knownConstExprOperand(operandInstr, knownConsts)
	if !ok || !knownLocal {
		return localUnaryExprKey{}, false
	}
	if _, ok := checkedNegI32(operand.Imm); !ok {
		return localUnaryExprKey{}, false
	}
	return localUnaryExprKey{Kind: kind, Operand: operand}, true
}

func localExprKeysFromInstrs(kind ir.IRInstrKind, leftInstr ir.IRInstr, rightInstr ir.IRInstr, knownConsts map[int]int32) []localExprKey {
	keys := []localExprKey{}
	if key, ok := localExprKeyFromInstrs(kind, leftInstr, rightInstr); ok {
		keys = append(keys, key)
	}
	if key, ok := knownConstBinaryExprKeyFromInstrs(kind, leftInstr, rightInstr, knownConsts); ok && !localExprKeysContain(keys, key) {
		keys = append(keys, key)
	}
	return keys
}

func cachedLocalForExprKeys(exprs map[localExprKey]int, keys []localExprKey) (int, bool) {
	for _, key := range keys {
		if cachedLocal, ok := exprs[key]; ok {
			return cachedLocal, true
		}
	}
	return 0, false
}

func storableLocalExprKeys(keys []localExprKey, storeLocal int) []localExprKey {
	out := make([]localExprKey, 0, len(keys))
	for _, key := range keys {
		if !localExprKeyUsesLocal(key, storeLocal) {
			out = append(out, key)
		}
	}
	return out
}

func localExprKeysContain(keys []localExprKey, needle localExprKey) bool {
	for _, key := range keys {
		if key == needle {
			return true
		}
	}
	return false
}

func knownConstBinaryExprKeyFromInstrs(kind ir.IRInstrKind, leftInstr ir.IRInstr, rightInstr ir.IRInstr, knownConsts map[int]int32) (localExprKey, bool) {
	switch kind {
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32,
		ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
	default:
		return localExprKey{}, false
	}
	left, leftKnownLocal, ok := knownConstExprOperand(leftInstr, knownConsts)
	if !ok {
		return localExprKey{}, false
	}
	right, rightKnownLocal, ok := knownConstExprOperand(rightInstr, knownConsts)
	if !ok || (!leftKnownLocal && !rightKnownLocal) {
		return localExprKey{}, false
	}
	if _, ok := foldConstBinaryI32(kind, left.Imm, right.Imm); !ok {
		return localExprKey{}, false
	}
	return canonicalLocalExprKey(kind, left, right), true
}

func knownConstExprOperand(instr ir.IRInstr, knownConsts map[int]int32) (localExprOperand, bool, bool) {
	switch instr.Kind {
	case ir.IRConstI32:
		return localExprOperand{Kind: localExprOperandConst, Imm: instr.Imm}, false, true
	case ir.IRLoadLocal:
		imm, ok := knownConsts[instr.Local]
		if !ok {
			return localExprOperand{}, false, false
		}
		return localExprOperand{Kind: localExprOperandConst, Imm: imm}, true, true
	default:
		return localExprOperand{}, false, false
	}
}

func updateKnownConstForStore(knownConsts map[int]int32, instrs []ir.IRInstr, storeIndex int) {
	store := instrs[storeIndex]
	delete(knownConsts, store.Local)
	if storeIndex > 0 && instrs[storeIndex-1].Kind == ir.IRConstI32 {
		knownConsts[store.Local] = instrs[storeIndex-1].Imm
	}
}

func updateKnownConstForExpressionStore(knownConsts map[int]int32, kind ir.IRInstrKind, leftInstr ir.IRInstr, rightInstr ir.IRInstr, storeLocal int) {
	delete(knownConsts, storeLocal)
	left, _, ok := knownConstExprOperand(leftInstr, knownConsts)
	if !ok {
		return
	}
	if kind == ir.IRNegI32 {
		if folded, ok := checkedNegI32(left.Imm); ok {
			knownConsts[storeLocal] = folded
		}
		return
	}
	right, _, ok := knownConstExprOperand(rightInstr, knownConsts)
	if !ok {
		return
	}
	if folded, ok := foldConstBinaryI32(kind, left.Imm, right.Imm); ok {
		knownConsts[storeLocal] = folded
	}
}

func localUnaryExprKeyFromInstrs(kind ir.IRInstrKind, operandInstr ir.IRInstr) (localUnaryExprKey, bool) {
	if kind != ir.IRNegI32 {
		return localUnaryExprKey{}, false
	}
	operand, ok := localExprOperandFromInstr(operandInstr)
	if !ok || operand.Kind != localExprOperandLocal {
		return localUnaryExprKey{}, false
	}
	return localUnaryExprKey{Kind: kind, Operand: operand}, true
}

func localExprKeyFromInstrs(kind ir.IRInstrKind, leftInstr ir.IRInstr, rightInstr ir.IRInstr) (localExprKey, bool) {
	if !isPureLocalBinaryOp(kind) {
		return localExprKey{}, false
	}
	left, ok := localExprOperandFromInstr(leftInstr)
	if !ok {
		return localExprKey{}, false
	}
	right, ok := localExprOperandFromInstr(rightInstr)
	if !ok {
		return localExprKey{}, false
	}
	if left.Kind != localExprOperandLocal && right.Kind != localExprOperandLocal {
		return localExprKey{}, false
	}
	if !localExprOperandsSafeForCSE(kind, left, right) {
		return localExprKey{}, false
	}
	return canonicalLocalExprKey(kind, left, right), true
}

func localExprOperandFromInstr(instr ir.IRInstr) (localExprOperand, bool) {
	switch instr.Kind {
	case ir.IRLoadLocal:
		return localExprOperand{Kind: localExprOperandLocal, Local: instr.Local}, true
	case ir.IRConstI32:
		return localExprOperand{Kind: localExprOperandConst, Imm: instr.Imm}, true
	default:
		return localExprOperand{}, false
	}
}

func canonicalLocalExprKey(kind ir.IRInstrKind, left localExprOperand, right localExprOperand) localExprKey {
	if isCommutativeLocalBinaryOp(kind) && compareLocalExprOperands(right, left) < 0 {
		left, right = right, left
	}
	if isMirroredComparisonLocalBinaryOp(kind) && compareLocalExprOperands(right, left) < 0 {
		left, right = right, left
		kind = mirroredComparisonLocalBinaryOp(kind)
	}
	return localExprKey{Kind: kind, Left: left, Right: right}
}

func compareLocalExprOperands(left localExprOperand, right localExprOperand) int {
	if left.Kind != right.Kind {
		if left.Kind < right.Kind {
			return -1
		}
		return 1
	}
	switch left.Kind {
	case localExprOperandLocal:
		return compareInt(left.Local, right.Local)
	case localExprOperandConst:
		return compareInt32(left.Imm, right.Imm)
	default:
		return 0
	}
}

func compareInt(left int, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareInt32(left int32, right int32) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func isPureLocalBinaryOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32,
		ir.IRDivI32, ir.IRModI32,
		ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func localExprOperandsSafeForCSE(kind ir.IRInstrKind, left localExprOperand, right localExprOperand) bool {
	switch kind {
	case ir.IRDivI32, ir.IRModI32:
		return left.Kind == localExprOperandLocal &&
			right.Kind == localExprOperandConst &&
			right.Imm != 0 &&
			right.Imm != -1
	default:
		return true
	}
}

func isCommutativeLocalBinaryOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRAddI32, ir.IRMulI32, ir.IRCmpEqI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func isMirroredComparisonLocalBinaryOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpLeI32, ir.IRCmpGeI32:
		return true
	default:
		return false
	}
}

func mirroredComparisonLocalBinaryOp(kind ir.IRInstrKind) ir.IRInstrKind {
	switch kind {
	case ir.IRCmpLtI32:
		return ir.IRCmpGtI32
	case ir.IRCmpGtI32:
		return ir.IRCmpLtI32
	case ir.IRCmpLeI32:
		return ir.IRCmpGeI32
	case ir.IRCmpGeI32:
		return ir.IRCmpLeI32
	default:
		return kind
	}
}

func invalidateCachedExpressions(exprs map[localExprKey]int, local int) {
	for key, cachedLocal := range exprs {
		if cachedLocal == local || localExprKeyUsesLocal(key, local) {
			delete(exprs, key)
		}
	}
}

func localExprKeyUsesLocal(key localExprKey, local int) bool {
	return localExprOperandUsesLocal(key.Left, local) || localExprOperandUsesLocal(key.Right, local)
}

func invalidateCachedUnaryExpressions(exprs map[localUnaryExprKey]int, local int) {
	for key, cachedLocal := range exprs {
		if cachedLocal == local || localUnaryExprKeyUsesLocal(key, local) {
			delete(exprs, key)
		}
	}
}

func localUnaryExprKeyUsesLocal(key localUnaryExprKey, local int) bool {
	return localExprOperandUsesLocal(key.Operand, local)
}

func localExprOperandUsesLocal(operand localExprOperand, local int) bool {
	return operand.Kind == localExprOperandLocal && operand.Local == local
}

func clearCachedExpressions(exprs map[localExprKey]int) {
	for key := range exprs {
		delete(exprs, key)
	}
}

func clearCachedUnaryExpressions(exprs map[localUnaryExprKey]int) {
	for key := range exprs {
		delete(exprs, key)
	}
}

func foldConstBinaryI32(kind ir.IRInstrKind, left int32, right int32) (int32, bool) {
	switch kind {
	case ir.IRAddI32:
		return checkedI32(int64(left) + int64(right))
	case ir.IRSubI32:
		return checkedI32(int64(left) - int64(right))
	case ir.IRMulI32:
		return checkedI32(int64(left) * int64(right))
	case ir.IRDivI32:
		if right == 0 || right == -1 {
			return 0, false
		}
		return left / right, true
	case ir.IRModI32:
		if right == 0 || right == -1 {
			return 0, false
		}
		return left % right, true
	case ir.IRCmpEqI32:
		return boolI32(left == right), true
	case ir.IRCmpLtI32:
		return boolI32(left < right), true
	case ir.IRCmpGtI32:
		return boolI32(left > right), true
	case ir.IRCmpGeI32:
		return boolI32(left >= right), true
	case ir.IRCmpLeI32:
		return boolI32(left <= right), true
	case ir.IRCmpNeI32:
		return boolI32(left != right), true
	default:
		return 0, false
	}
}

func checkedNegI32(v int32) (int32, bool) {
	if int64(v) == minInt32 {
		return 0, false
	}
	return -v, true
}

func checkedI32(v int64) (int32, bool) {
	if v < minInt32 || v > maxInt32 {
		return 0, false
	}
	return int32(v), true
}

func boolI32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func constInstr(from ir.IRInstr, v int32) ir.IRInstr {
	return ir.IRInstr{Kind: ir.IRConstI32, Imm: v, Pos: from.Pos}
}

func isConstI32(instr ir.IRInstr, v int32) bool {
	return instr.Kind == ir.IRConstI32 && instr.Imm == v
}

func isSinglePureValue(instr ir.IRInstr) bool {
	switch instr.Kind {
	case ir.IRConstI32, ir.IRLoadLocal:
		return true
	default:
		return false
	}
}

func sameSinglePureValue(left ir.IRInstr, right ir.IRInstr) bool {
	if left.Kind != right.Kind || !isSinglePureValue(left) {
		return false
	}
	switch left.Kind {
	case ir.IRConstI32:
		return left.Imm == right.Imm
	case ir.IRLoadLocal:
		return left.Local == right.Local
	default:
		return false
	}
}

func sameValueComparisonResult(kind ir.IRInstrKind) int32 {
	switch kind {
	case ir.IRCmpEqI32, ir.IRCmpGeI32, ir.IRCmpLeI32:
		return 1
	default:
		return 0
	}
}

func isDeadStoreProducer(instr ir.IRInstr) bool {
	return isSinglePureValue(instr)
}

func resolveLocalCopy(copies map[int]int, local int) int {
	seen := map[int]bool{}
	cur := local
	for {
		if seen[cur] {
			return cur
		}
		seen[cur] = true
		next, ok := copies[cur]
		if !ok {
			return cur
		}
		cur = next
	}
}

func invalidateLocalCopies(copies map[int]int, local int) {
	for dst, src := range copies {
		if dst == local || src == local {
			delete(copies, dst)
		}
	}
}

func clearLocalCopies(copies map[int]int) {
	for dst := range copies {
		delete(copies, dst)
	}
}

func clearsCopyFacts(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRCall, ir.IRStoreGlobal, ir.IRIndexStoreI32, ir.IRIndexStoreU8,
		ir.IRIndexStoreU16, ir.IRMemWriteI32, ir.IRMemWriteU8,
		ir.IRMemWritePtr, ir.IRMemWriteArchPtr, ir.IRMemWriteI32Offset,
		ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset,
		ir.IRMemWriteArchPtrOffset, ir.IRMmioWriteI32, ir.IRCtxSwitch,
		ir.IRAtomicStorePtr, ir.IRAtomicExchangePtr, ir.IRAtomicFetchAddPtr,
		ir.IRAtomicFetchSubPtr, ir.IRAtomicFetchAndPtr, ir.IRAtomicFetchOrPtr,
		ir.IRAtomicFetchXorPtr, ir.IRAtomicCompareExchangePtr,
		ir.IRAtomicStoreI32, ir.IRAtomicExchangeI32,
		ir.IRAtomicCompareExchangeI32, ir.IRAtomicFetchAddI32,
		ir.IRAtomicFetchSubI32, ir.IRAtomicFetchAndI32,
		ir.IRAtomicFetchOrI32, ir.IRAtomicFetchXorI32,
		ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64,
		ir.IRAtomicCompareExchangeI64, ir.IRAtomicFetchAddI64,
		ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64, ir.IRAtomicFetchXorI64,
		ir.IRAtomicStoreI8, ir.IRAtomicExchangeI8,
		ir.IRAtomicCompareExchangeI8, ir.IRAtomicFetchAddI8,
		ir.IRAtomicFetchSubI8, ir.IRAtomicFetchAndI8,
		ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8,
		ir.IRAtomicStoreI16, ir.IRAtomicExchangeI16,
		ir.IRAtomicCompareExchangeI16, ir.IRAtomicFetchAddI16,
		ir.IRAtomicFetchSubI16, ir.IRAtomicFetchAndI16,
		ir.IRAtomicFetchOrI16, ir.IRAtomicFetchXorI16:
		return true
	default:
		return false
	}
}
