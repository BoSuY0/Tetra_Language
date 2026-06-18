package lower

import (
	"fmt"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func (l *lowerer) ensureCallableScratchBase(slots int) int {
	if slots <= 0 {
		return -1
	}
	base := l.localSlots
	l.localSlots += slots
	return base
}

func (l *lowerer) emitFunctionSymbolValue(name string, envLocals []int, pos frontend.Position) int {
	l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: name, Pos: pos})
	for i := 0; i < semantics.FnPtrEnvSlotCount; i++ {
		if i < len(envLocals) && envLocals[i] >= 0 {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: envLocals[i], Pos: pos})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		}
	}
	return semantics.FnPtrSlotCount
}

func (l *lowerer) emitCallableHandleValue(name string, captures []frontend.ClosureCapture, pos frontend.Position) int {
	envLocals := l.closureEnvLocalsUnbounded(captures)
	envPtrLocal := l.allocScratchSlots(1)
	discardLocal := l.ensureDiscardLocal()
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(len(envLocals) * 8), Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRAllocBytes, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: envPtrLocal, Pos: pos})
	for slot, local := range envLocals {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: envPtrLocal, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot * 8), Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: local, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRCapMem, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRMemWritePtrOffset, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discardLocal, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: name, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: envPtrLocal, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(len(envLocals)), Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	return semantics.CallableHandleSlotCount
}

func (l *lowerer) capturedClosureEnvLocals(info semantics.LocalInfo) []int {
	return l.closureEnvLocals(info.FunctionCaptures)
}

func (l *lowerer) closureEnvLocals(captures []frontend.ClosureCapture) []int {
	envLocals := make([]int, 0, semantics.FnPtrEnvSlotCount)
	for _, capture := range captures {
		captured, ok := l.locals[capture.Name]
		if !ok {
			return nil
		}
		for slot := 0; slot < captured.SlotCount; slot++ {
			if len(envLocals) >= semantics.FnPtrEnvSlotCount {
				return nil
			}
			envLocals = append(envLocals, captured.Base+slot)
		}
	}
	return envLocals
}

func (l *lowerer) closureEnvLocalsUnbounded(captures []frontend.ClosureCapture) []int {
	var envLocals []int
	for _, capture := range captures {
		captured, ok := l.locals[capture.Name]
		if !ok {
			return nil
		}
		for slot := 0; slot < captured.SlotCount; slot++ {
			envLocals = append(envLocals, captured.Base+slot)
		}
	}
	return envLocals
}

func (l *lowerer) lowerFunctionTypedLocalAssignmentValue(value frontend.Expr, target semantics.LocalInfo, pos frontend.Position) (int, error) {
	if closure, ok := value.(*frontend.ClosureExpr); ok {
		if envLocals := l.closureEnvLocalsUnbounded(closure.Captures); len(envLocals) > semantics.FnPtrEnvSlotCount {
			slots := l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
			if target.SlotCount < slots {
				return 0, fmt.Errorf("%s: callable handle assignment requires %d slots, target has %d", frontend.FormatPos(pos), slots, target.SlotCount)
			}
			l.emitZeroSlots(target.SlotCount-slots, pos)
			return target.SlotCount, nil
		}
		return l.emitFunctionSymbolValue(l.closureSymbolName(closure), l.closureEnvLocals(closure.Captures), closure.At), nil
	}
	if target, ok := functionTypedGlobalFieldTargetFromExpr(value, l.globals); ok {
		return l.emitFunctionSymbolValue(target, nil, value.Pos()), nil
	}
	if call, ok := value.(*frontend.CallExpr); ok {
		if sig, ok := l.funcs[call.Name]; ok && sig.ReturnFunctionHandleValue {
			slots, err := l.lowerExpr(value)
			if err != nil {
				return 0, err
			}
			if slots < target.SlotCount {
				l.emitZeroSlots(target.SlotCount-slots, value.Pos())
				return target.SlotCount, nil
			}
			return slots, nil
		}
	}
	if id, ok := value.(*frontend.IdentExpr); ok {
		if local, ok := l.locals[id.Name]; ok && local.FunctionTypeValue {
			for slot := 0; slot < local.SlotCount; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: local.Base + slot, Pos: value.Pos()})
			}
			desiredSlots := semantics.FnPtrSlotCount
			if target.FunctionHandleValue {
				desiredSlots = target.SlotCount
			}
			if local.SlotCount < desiredSlots {
				l.emitZeroSlots(desiredSlots-local.SlotCount, value.Pos())
				return desiredSlots, nil
			}
			return local.SlotCount, nil
		}
		if local, ok := l.locals[id.Name]; ok && !local.FunctionTypeValue && local.FunctionValue != "" {
			if target.FunctionHandleValue || local.FunctionHandleValue || len(l.closureEnvLocalsUnbounded(local.FunctionCaptures)) > semantics.FnPtrEnvSlotCount {
				slots := l.emitCallableHandleValue(local.FunctionValue, local.FunctionCaptures, value.Pos())
				if slots < target.SlotCount {
					l.emitZeroSlots(target.SlotCount-slots, value.Pos())
					return target.SlotCount, nil
				}
				return slots, nil
			}
			return l.emitFunctionSymbolValue(local.FunctionValue, l.capturedClosureEnvLocals(local), value.Pos()), nil
		}
	}
	return l.lowerExprAs(value, target.TypeName)
}

func (l *lowerer) lowerFunctionTypedArgument(value frontend.Expr) (int, error) {
	if closure, ok := value.(*frontend.ClosureExpr); ok {
		if envLocals := l.closureEnvLocalsUnbounded(closure.Captures); len(envLocals) > semantics.FnPtrEnvSlotCount {
			slots := l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
			l.emitZeroSlots(semantics.FnPtrSlotCount-slots, closure.At)
			return semantics.FnPtrSlotCount, nil
		}
		return l.emitFunctionSymbolValue(l.closureSymbolName(closure), l.closureEnvLocals(closure.Captures), closure.At), nil
	}
	if target, ok := functionTypedGlobalFieldTargetFromExpr(value, l.globals); ok {
		return l.emitFunctionSymbolValue(target, nil, value.Pos()), nil
	}
	if call, ok := value.(*frontend.CallExpr); ok {
		if sig, ok := l.funcs[call.Name]; ok && sig.ReturnFunctionHandleValue {
			slots, err := l.lowerExpr(value)
			if err != nil {
				return 0, err
			}
			if slots < semantics.FnPtrSlotCount {
				l.emitZeroSlots(semantics.FnPtrSlotCount-slots, value.Pos())
				return semantics.FnPtrSlotCount, nil
			}
			return slots, nil
		}
	}
	if id, ok := value.(*frontend.IdentExpr); ok {
		if local, ok := l.locals[id.Name]; ok && local.FunctionTypeValue {
			for slot := 0; slot < local.SlotCount; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: local.Base + slot, Pos: value.Pos()})
			}
			if local.SlotCount < semantics.FnPtrSlotCount {
				l.emitZeroSlots(semantics.FnPtrSlotCount-local.SlotCount, value.Pos())
				return semantics.FnPtrSlotCount, nil
			}
			return local.SlotCount, nil
		}
		if local, ok := l.locals[id.Name]; ok && !local.FunctionTypeValue && local.FunctionValue != "" {
			if local.FunctionHandleValue || len(l.closureEnvLocalsUnbounded(local.FunctionCaptures)) > semantics.FnPtrEnvSlotCount {
				slots := l.emitCallableHandleValue(local.FunctionValue, local.FunctionCaptures, value.Pos())
				l.emitZeroSlots(semantics.FnPtrSlotCount-slots, value.Pos())
				return semantics.FnPtrSlotCount, nil
			}
			return l.emitFunctionSymbolValue(local.FunctionValue, l.capturedClosureEnvLocals(local), value.Pos()), nil
		}
	}
	return l.lowerExpr(value)
}

func (l *lowerer) lowerFunctionTypedParamCall(e *frontend.CallExpr, local semantics.LocalInfo) (int, error) {
	targets := l.callableParamTargets[e.Name]
	if len(targets) == 0 {
		return 0, fmt.Errorf("%s: function-typed parameter '%s' cannot be lowered as a direct fnptr call without a known target set; pass a direct named function/closure symbol at each call site or use supported function-typed storage before dispatch", frontend.FormatPos(e.At), e.Name)
	}
	total := 0
	for i, arg := range e.Args {
		slots, err := l.lowerCallableExplicitArgument(arg, local.FunctionParamTypes, i)
		if err != nil {
			return 0, err
		}
		total += slots
	}
	argScratch := l.ensureCallableScratchBase(total)
	for slot := total - 1; slot >= 0; slot-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: argScratch + slot, Pos: e.At})
	}
	returnInfo, ok := l.types[local.FunctionReturnType]
	if !ok {
		return 0, fmt.Errorf("%s: unknown callback return type '%s'", frontend.FormatPos(e.At), local.FunctionReturnType)
	}
	expectedArgSlots := total
	expectedRetSlots := returnInfo.SlotCount
	if local.FunctionThrowsType != "" {
		_, errorSlots, _, err := throwingLayout(local.FunctionReturnType, local.FunctionThrowsType, l.types)
		if err != nil {
			return 0, err
		}
		expectedRetSlots = throwingReturnSlotCount(returnInfo.SlotCount, errorSlots)
	}
	writebacks := []inoutWriteback(nil)
	if local.FunctionThrowsType == "" {
		var err error
		writebacks, err = l.collectInoutWritebacks(e.Args, local.FunctionParamOwnership)
		if err != nil {
			return 0, err
		}
	}
	l.invalidateWhileRangeProofsForInoutArgs(e.Args, local.FunctionParamOwnership)
	abiRetSlots := expectedRetSlots + inoutWritebackSlotCount(writebacks)
	if local.FunctionHandleValue {
		return l.lowerCallableHandleLocalCall(e, local, targets, total, argScratch, expectedRetSlots, abiRetSlots, writebacks)
	}
	for _, target := range targets {
		sig, ok := l.funcs[target]
		if !ok {
			return 0, fmt.Errorf("%s: unknown callback target '%s'", frontend.FormatPos(e.At), target)
		}
		if sig.ThrowsType != local.FunctionThrowsType {
			return 0, fmt.Errorf("%s: callback target '%s' throws type mismatch: expected '%s', got '%s'", frontend.FormatPos(e.At), target, local.FunctionThrowsType, sig.ThrowsType)
		}
		captureSlots := sig.ParamSlots - expectedArgSlots
		if captureSlots < 0 {
			return 0, fmt.Errorf("%s: callback target '%s' slot mismatch: expected %d..%d arg slots with captured env slots, got %d", frontend.FormatPos(e.At), target, expectedArgSlots, expectedArgSlots+semantics.FnPtrEnvSlotCount, sig.ParamSlots)
		}
		if sig.ReturnSlots != expectedRetSlots {
			return 0, fmt.Errorf("%s: callback target '%s' return slot mismatch: expected %d, got %d", frontend.FormatPos(e.At), target, expectedRetSlots, sig.ReturnSlots)
		}
	}
	if len(targets) == 1 {
		target := targets[0]
		for slot := 0; slot < total; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: argScratch + slot, Pos: e.At})
		}
		targetArgSlots := total
		if sig := l.funcs[target]; sig.ParamSlots > total {
			hiddenSlots := sig.ParamSlots - total
			for slot := 0; slot < hiddenSlots; slot++ {
				if hiddenSlots > semantics.FnPtrEnvSlotCount {
					l.emitCallableHandleEnvLoad(local.Base+1, slot, e.At)
				} else {
					l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: local.Base + 1 + slot, Pos: e.At})
				}
			}
			targetArgSlots = sig.ParamSlots
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: target, ArgSlots: targetArgSlots, RetSlots: abiRetSlots, Pos: e.At})
		l.emitInoutWritebacks(writebacks, e.At)
		return expectedRetSlots, nil
	}

	resultScratch := l.ensureCallableScratchBase(expectedRetSlots)
	endLabel := l.newLabel()
	for _, target := range targets {
		nextLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: local.Base, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: target, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: e.At})
		for slot := 0; slot < total; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: argScratch + slot, Pos: e.At})
		}
		targetArgSlots := total
		if sig := l.funcs[target]; sig.ParamSlots > total {
			hiddenSlots := sig.ParamSlots - total
			for slot := 0; slot < hiddenSlots; slot++ {
				if hiddenSlots > semantics.FnPtrEnvSlotCount {
					l.emitCallableHandleEnvLoad(local.Base+1, slot, e.At)
				} else {
					l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: local.Base + 1 + slot, Pos: e.At})
				}
			}
			targetArgSlots = sig.ParamSlots
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: target, ArgSlots: targetArgSlots, RetSlots: abiRetSlots, Pos: e.At})
		l.emitInoutWritebacks(writebacks, e.At)
		for slot := expectedRetSlots - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultScratch + slot, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: e.At})
	}
	l.emitZeroSlots(expectedRetSlots, e.At)
	for slot := expectedRetSlots - 1; slot >= 0; slot-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultScratch + slot, Pos: e.At})
	}
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
	for slot := 0; slot < expectedRetSlots; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultScratch + slot, Pos: e.At})
	}
	return expectedRetSlots, nil
}

func (l *lowerer) lowerCallableHandleLocalCall(
	e *frontend.CallExpr,
	local semantics.LocalInfo,
	targets []string,
	total int,
	argScratch int,
	expectedRetSlots int,
	abiRetSlots int,
	writebacks []inoutWriteback,
) (int, error) {
	if len(targets) != 1 {
		return 0, fmt.Errorf("%s: callable handle '%s' requires a single stable target for current handle lowering", frontend.FormatPos(e.At), e.Name)
	}
	target := targets[0]
	sig, ok := l.funcs[target]
	if !ok {
		return 0, fmt.Errorf("%s: unknown callback target '%s'", frontend.FormatPos(e.At), target)
	}
	if sig.ThrowsType != local.FunctionThrowsType {
		return 0, fmt.Errorf("%s: callback target '%s' throws type mismatch: expected '%s', got '%s'", frontend.FormatPos(e.At), target, local.FunctionThrowsType, sig.ThrowsType)
	}
	hiddenSlots := sig.ParamSlots - total
	if hiddenSlots < 0 {
		return 0, fmt.Errorf("%s: callback target '%s' slot mismatch: expected at least %d arg slots, got %d", frontend.FormatPos(e.At), target, total, sig.ParamSlots)
	}
	if sig.ReturnSlots != expectedRetSlots {
		return 0, fmt.Errorf("%s: callback target '%s' return slot mismatch: expected %d, got %d", frontend.FormatPos(e.At), target, expectedRetSlots, sig.ReturnSlots)
	}
	for slot := 0; slot < total; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: argScratch + slot, Pos: e.At})
	}
	for slot := 0; slot < hiddenSlots; slot++ {
		l.emitCallableHandleEnvLoad(local.Base+1, slot, e.At)
	}
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: target, ArgSlots: sig.ParamSlots, RetSlots: abiRetSlots, Pos: e.At})
	l.emitInoutWritebacks(writebacks, e.At)
	return expectedRetSlots, nil
}

func (l *lowerer) emitCallableHandleEnvLoad(envPtrLocal int, slot int, pos frontend.Position) {
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: envPtrLocal, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot * 8), Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCapMem, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRMemReadPtrOffset, Pos: pos})
}

func (l *lowerer) lowerStoredFunctionCall(e *frontend.CallExpr, fieldInfo semantics.FunctionFieldInfo, fnptrBase int) (int, error) {
	targets := append([]string(nil), l.callableParamTargets[e.Name]...)
	if len(targets) == 0 && fieldInfo.FunctionValue != "" {
		targets = append(targets, fieldInfo.FunctionValue)
	}
	if len(targets) == 0 {
		return 0, fmt.Errorf("%s: function-typed struct field '%s' has no stable target", frontend.FormatPos(e.At), e.Name)
	}
	total := 0
	for i, arg := range e.Args {
		slots, err := l.lowerCallableExplicitArgument(arg, fieldInfo.FunctionParamTypes, i)
		if err != nil {
			return 0, err
		}
		total += slots
	}
	argScratch := l.ensureCallableScratchBase(total)
	for slot := total - 1; slot >= 0; slot-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: argScratch + slot, Pos: e.At})
	}
	returnInfo, ok := l.types[fieldInfo.FunctionReturnType]
	if !ok {
		return 0, fmt.Errorf("%s: unknown callback return type '%s'", frontend.FormatPos(e.At), fieldInfo.FunctionReturnType)
	}
	expectedReturnSlots := returnInfo.SlotCount
	if fieldInfo.FunctionThrowsType != "" {
		_, errorSlots, _, err := throwingLayout(fieldInfo.FunctionReturnType, fieldInfo.FunctionThrowsType, l.types)
		if err != nil {
			return 0, err
		}
		expectedReturnSlots = throwingReturnSlotCount(returnInfo.SlotCount, errorSlots)
	}
	writebacks := []inoutWriteback(nil)
	if fieldInfo.FunctionThrowsType == "" {
		var err error
		writebacks, err = l.collectInoutWritebacks(e.Args, fieldInfo.FunctionParamOwnership)
		if err != nil {
			return 0, err
		}
	}
	l.invalidateWhileRangeProofsForInoutArgs(e.Args, fieldInfo.FunctionParamOwnership)
	abiReturnSlots := expectedReturnSlots + inoutWritebackSlotCount(writebacks)
	for _, target := range targets {
		sig, ok := l.funcs[target]
		if !ok {
			return 0, fmt.Errorf("%s: unknown callback target '%s'", frontend.FormatPos(e.At), target)
		}
		if sig.ThrowsType != fieldInfo.FunctionThrowsType {
			return 0, fmt.Errorf("%s: callback target '%s' throws type mismatch: expected '%s', got '%s'", frontend.FormatPos(e.At), target, fieldInfo.FunctionThrowsType, sig.ThrowsType)
		}
		captureSlots := sig.ParamSlots - total
		if captureSlots < 0 {
			return 0, fmt.Errorf("%s: callback target '%s' slot mismatch: expected %d..%d arg slots with captured env slots, got %d", frontend.FormatPos(e.At), target, total, total+semantics.FnPtrEnvSlotCount, sig.ParamSlots)
		}
		if sig.ReturnSlots != expectedReturnSlots {
			return 0, fmt.Errorf("%s: callback target '%s' return slot mismatch: expected %d, got %d", frontend.FormatPos(e.At), target, expectedReturnSlots, sig.ReturnSlots)
		}
	}
	if len(targets) == 1 {
		target := targets[0]
		sig := l.funcs[target]
		for slot := 0; slot < total; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: argScratch + slot, Pos: e.At})
		}
		hiddenSlots := sig.ParamSlots - total
		for slot := 0; slot < hiddenSlots; slot++ {
			if hiddenSlots > semantics.FnPtrEnvSlotCount {
				l.emitCallableHandleEnvLoad(fnptrBase+1, slot, e.At)
			} else {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: fnptrBase + 1 + slot, Pos: e.At})
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: target, ArgSlots: sig.ParamSlots, RetSlots: abiReturnSlots, Pos: e.At})
		l.emitInoutWritebacks(writebacks, e.At)
		return expectedReturnSlots, nil
	}
	resultScratch := l.ensureCallableScratchBase(expectedReturnSlots)
	endLabel := l.newLabel()
	for _, target := range targets {
		sig := l.funcs[target]
		nextLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: fnptrBase, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: target, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: e.At})
		for slot := 0; slot < total; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: argScratch + slot, Pos: e.At})
		}
		hiddenSlots := sig.ParamSlots - total
		for slot := 0; slot < hiddenSlots; slot++ {
			if hiddenSlots > semantics.FnPtrEnvSlotCount {
				l.emitCallableHandleEnvLoad(fnptrBase+1, slot, e.At)
			} else {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: fnptrBase + 1 + slot, Pos: e.At})
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: target, ArgSlots: sig.ParamSlots, RetSlots: abiReturnSlots, Pos: e.At})
		l.emitInoutWritebacks(writebacks, e.At)
		for slot := expectedReturnSlots - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultScratch + slot, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: e.At})
	}
	l.emitZeroSlots(expectedReturnSlots, e.At)
	for slot := expectedReturnSlots - 1; slot >= 0; slot-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultScratch + slot, Pos: e.At})
	}
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
	for slot := 0; slot < expectedReturnSlots; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultScratch + slot, Pos: e.At})
	}
	return expectedReturnSlots, nil
}

func (l *lowerer) lowerGlobalStoredFunctionCall(e *frontend.CallExpr, global semantics.GlobalInfo) (int, error) {
	slotCount := gSlotCount(global.TypeName, l.types)
	fnptrBase := l.ensureCallableScratchBase(slotCount)
	for slot := 0; slot < slotCount; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: global.DataIndex + slot, Pos: e.At})
	}
	for slot := slotCount - 1; slot >= 0; slot-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: fnptrBase + slot, Pos: e.At})
	}
	return l.lowerStoredFunctionCall(e, semantics.FunctionFieldInfo{
		FunctionValue:          global.FunctionValue,
		FunctionParamTypes:     append([]string(nil), global.FunctionParamTypes...),
		FunctionParamOwnership: append([]string(nil), global.FunctionParamOwnership...),
		FunctionReturnType:     global.FunctionReturnType,
		FunctionThrowsType:     global.FunctionThrowsType,
	}, fnptrBase)
}

func (l *lowerer) lowerCallableExplicitArgument(arg frontend.Expr, paramTypes []string, index int) (int, error) {
	if index < len(paramTypes) {
		return l.lowerExprAs(arg, paramTypes[index])
	}
	return l.lowerExpr(arg)
}
