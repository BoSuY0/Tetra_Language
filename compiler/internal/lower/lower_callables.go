package lower

import (
	"fmt"
	"sort"
	"strings"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowercallables "tetra_language/compiler/internal/lower/callables"
	"tetra_language/compiler/internal/semantics"
)

// ---- callable_lowering.go ----

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

func (l *lowerer) emitCallableHandleValue(
	name string,
	captures []frontend.ClosureCapture,
	pos frontend.Position,
) int {
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

func (l *lowerer) lowerFunctionTypedLocalAssignmentValue(
	value frontend.Expr,
	target semantics.LocalInfo,
	pos frontend.Position,
) (int, error) {
	if closure, ok := value.(*frontend.ClosureExpr); ok {
		if envLocals := l.closureEnvLocalsUnbounded(closure.Captures); len(
			envLocals,
		) > semantics.FnPtrEnvSlotCount {
			slots := l.emitCallableHandleValue(
				l.closureSymbolName(closure),
				closure.Captures,
				closure.At,
			)
			if target.SlotCount < slots {
				return 0, fmt.Errorf(
					"%s: callable handle assignment requires %d slots, target has %d",
					frontend.FormatPos(pos),
					slots,
					target.SlotCount,
				)
			}
			l.emitZeroSlots(target.SlotCount-slots, pos)
			return target.SlotCount, nil
		}
		return l.emitFunctionSymbolValue(
			l.closureSymbolName(closure),
			l.closureEnvLocals(closure.Captures),
			closure.At,
		), nil
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
		if local, ok := l.locals[id.Name]; ok && !local.FunctionTypeValue &&
			local.FunctionValue != "" {
			if target.FunctionHandleValue || local.FunctionHandleValue ||
				len(
					l.closureEnvLocalsUnbounded(local.FunctionCaptures),
				) > semantics.FnPtrEnvSlotCount {
				slots := l.emitCallableHandleValue(
					local.FunctionValue,
					local.FunctionCaptures,
					value.Pos(),
				)
				if slots < target.SlotCount {
					l.emitZeroSlots(target.SlotCount-slots, value.Pos())
					return target.SlotCount, nil
				}
				return slots, nil
			}
			return l.emitFunctionSymbolValue(
				local.FunctionValue,
				l.capturedClosureEnvLocals(local),
				value.Pos(),
			), nil
		}
	}
	return l.lowerExprAs(value, target.TypeName)
}

func (l *lowerer) lowerFunctionTypedArgument(value frontend.Expr) (int, error) {
	if closure, ok := value.(*frontend.ClosureExpr); ok {
		if envLocals := l.closureEnvLocalsUnbounded(closure.Captures); len(
			envLocals,
		) > semantics.FnPtrEnvSlotCount {
			slots := l.emitCallableHandleValue(
				l.closureSymbolName(closure),
				closure.Captures,
				closure.At,
			)
			l.emitZeroSlots(semantics.FnPtrSlotCount-slots, closure.At)
			return semantics.FnPtrSlotCount, nil
		}
		return l.emitFunctionSymbolValue(
			l.closureSymbolName(closure),
			l.closureEnvLocals(closure.Captures),
			closure.At,
		), nil
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
		if local, ok := l.locals[id.Name]; ok && !local.FunctionTypeValue &&
			local.FunctionValue != "" {
			if local.FunctionHandleValue ||
				len(
					l.closureEnvLocalsUnbounded(local.FunctionCaptures),
				) > semantics.FnPtrEnvSlotCount {
				slots := l.emitCallableHandleValue(
					local.FunctionValue,
					local.FunctionCaptures,
					value.Pos(),
				)
				l.emitZeroSlots(semantics.FnPtrSlotCount-slots, value.Pos())
				return semantics.FnPtrSlotCount, nil
			}
			return l.emitFunctionSymbolValue(
				local.FunctionValue,
				l.capturedClosureEnvLocals(local),
				value.Pos(),
			), nil
		}
	}
	return l.lowerExpr(value)
}

func (l *lowerer) lowerFunctionTypedParamCall(
	e *frontend.CallExpr,
	local semantics.LocalInfo,
) (int, error) {
	targets := l.callableParamTargets[e.Name]
	if len(targets) == 0 {
		return 0, fmt.Errorf(
			("%s: function-typed parameter '%s' cannot be lowered as a direct " +
				"fnptr call without a known target set; pass a direct named function/" +
				"closure symbol at each call site or use supported function-typed " +
				"storage before dispatch"),
			frontend.FormatPos(e.At),
			e.Name,
		)
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
		return 0, fmt.Errorf(
			"%s: unknown callback return type '%s'",
			frontend.FormatPos(e.At),
			local.FunctionReturnType,
		)
	}
	expectedArgSlots := total
	expectedRetSlots := returnInfo.SlotCount
	if local.FunctionThrowsType != "" {
		_, errorSlots, _, err := throwingLayout(
			local.FunctionReturnType,
			local.FunctionThrowsType,
			l.types,
		)
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
		return l.lowerCallableHandleLocalCall(
			e,
			local,
			targets,
			total,
			argScratch,
			expectedRetSlots,
			abiRetSlots,
			writebacks,
		)
	}
	for _, target := range targets {
		sig, ok := l.funcs[target]
		if !ok {
			return 0, fmt.Errorf(
				"%s: unknown callback target '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if sig.ThrowsType != local.FunctionThrowsType {
			return 0, fmt.Errorf(
				"%s: callback target '%s' throws type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(e.At),
				target,
				local.FunctionThrowsType,
				sig.ThrowsType,
			)
		}
		captureSlots := sig.ParamSlots - expectedArgSlots
		if captureSlots < 0 {
			return 0, fmt.Errorf(
				("%s: callback target '%s' slot mismatch: expected %d..%d arg " +
					"slots with captured env slots, got %d"),
				frontend.FormatPos(e.At),
				target,
				expectedArgSlots,
				expectedArgSlots+semantics.FnPtrEnvSlotCount,
				sig.ParamSlots,
			)
		}
		if sig.ReturnSlots != expectedRetSlots {
			return 0, fmt.Errorf(
				"%s: callback target '%s' return slot mismatch: expected %d, got %d",
				frontend.FormatPos(e.At),
				target,
				expectedRetSlots,
				sig.ReturnSlots,
			)
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
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     target,
				ArgSlots: targetArgSlots,
				RetSlots: abiRetSlots,
				Pos:      e.At,
			},
		)
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
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     target,
				ArgSlots: targetArgSlots,
				RetSlots: abiRetSlots,
				Pos:      e.At,
			},
		)
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
		return 0, fmt.Errorf(
			"%s: callable handle '%s' requires a single stable target for current handle lowering",
			frontend.FormatPos(e.At),
			e.Name,
		)
	}
	target := targets[0]
	sig, ok := l.funcs[target]
	if !ok {
		return 0, fmt.Errorf("%s: unknown callback target '%s'", frontend.FormatPos(e.At), target)
	}
	if sig.ThrowsType != local.FunctionThrowsType {
		return 0, fmt.Errorf(
			"%s: callback target '%s' throws type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(e.At),
			target,
			local.FunctionThrowsType,
			sig.ThrowsType,
		)
	}
	hiddenSlots := sig.ParamSlots - total
	if hiddenSlots < 0 {
		return 0, fmt.Errorf(
			"%s: callback target '%s' slot mismatch: expected at least %d arg slots, got %d",
			frontend.FormatPos(e.At),
			target,
			total,
			sig.ParamSlots,
		)
	}
	if sig.ReturnSlots != expectedRetSlots {
		return 0, fmt.Errorf(
			"%s: callback target '%s' return slot mismatch: expected %d, got %d",
			frontend.FormatPos(e.At),
			target,
			expectedRetSlots,
			sig.ReturnSlots,
		)
	}
	for slot := 0; slot < total; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: argScratch + slot, Pos: e.At})
	}
	for slot := 0; slot < hiddenSlots; slot++ {
		l.emitCallableHandleEnvLoad(local.Base+1, slot, e.At)
	}
	l.emit(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     target,
			ArgSlots: sig.ParamSlots,
			RetSlots: abiRetSlots,
			Pos:      e.At,
		},
	)
	l.emitInoutWritebacks(writebacks, e.At)
	return expectedRetSlots, nil
}

func (l *lowerer) emitCallableHandleEnvLoad(envPtrLocal int, slot int, pos frontend.Position) {
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: envPtrLocal, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot * 8), Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCapMem, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRMemReadPtrOffset, Pos: pos})
}

func (l *lowerer) lowerStoredFunctionCall(
	e *frontend.CallExpr,
	fieldInfo semantics.FunctionFieldInfo,
	fnptrBase int,
) (int, error) {
	targets := append([]string(nil), l.callableParamTargets[e.Name]...)
	if len(targets) == 0 && fieldInfo.FunctionValue != "" {
		targets = append(targets, fieldInfo.FunctionValue)
	}
	if len(targets) == 0 {
		return 0, fmt.Errorf(
			"%s: function-typed struct field '%s' has no stable target",
			frontend.FormatPos(e.At),
			e.Name,
		)
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
		return 0, fmt.Errorf(
			"%s: unknown callback return type '%s'",
			frontend.FormatPos(e.At),
			fieldInfo.FunctionReturnType,
		)
	}
	expectedReturnSlots := returnInfo.SlotCount
	if fieldInfo.FunctionThrowsType != "" {
		_, errorSlots, _, err := throwingLayout(
			fieldInfo.FunctionReturnType,
			fieldInfo.FunctionThrowsType,
			l.types,
		)
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
			return 0, fmt.Errorf(
				"%s: unknown callback target '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if sig.ThrowsType != fieldInfo.FunctionThrowsType {
			return 0, fmt.Errorf(
				"%s: callback target '%s' throws type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(e.At),
				target,
				fieldInfo.FunctionThrowsType,
				sig.ThrowsType,
			)
		}
		captureSlots := sig.ParamSlots - total
		if captureSlots < 0 {
			return 0, fmt.Errorf(
				("%s: callback target '%s' slot mismatch: expected %d..%d arg " +
					"slots with captured env slots, got %d"),
				frontend.FormatPos(e.At),
				target,
				total,
				total+semantics.FnPtrEnvSlotCount,
				sig.ParamSlots,
			)
		}
		if sig.ReturnSlots != expectedReturnSlots {
			return 0, fmt.Errorf(
				"%s: callback target '%s' return slot mismatch: expected %d, got %d",
				frontend.FormatPos(e.At),
				target,
				expectedReturnSlots,
				sig.ReturnSlots,
			)
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
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     target,
				ArgSlots: sig.ParamSlots,
				RetSlots: abiReturnSlots,
				Pos:      e.At,
			},
		)
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
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     target,
				ArgSlots: sig.ParamSlots,
				RetSlots: abiReturnSlots,
				Pos:      e.At,
			},
		)
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

func (l *lowerer) lowerGlobalStoredFunctionCall(
	e *frontend.CallExpr,
	global semantics.GlobalInfo,
) (int, error) {
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

func (l *lowerer) lowerCallableExplicitArgument(
	arg frontend.Expr,
	paramTypes []string,
	index int,
) (int, error) {
	if index < len(paramTypes) {
		return l.lowerExprAs(arg, paramTypes[index])
	}
	return l.lowerExpr(arg)
}

// ---- callable_target_edges.go ----

const (
	callableReturnFieldPrefix      = "$return."
	callableReturnFunctionLocal    = "$return.fn"
	callableReturnEnumPayloadLocal = "$return.enum"
)

type callableTargetEdge struct {
	callee      string
	param       string
	sourceFunc  string
	sourceParam string
}

type moduleGlobalTargetEdge struct {
	module      string
	global      string
	sourceFunc  string
	sourceParam string
}

type enumPayloadTargetEdge struct {
	destFunc         string
	destLocal        string
	destPayloadKey   string
	sourceFunc       string
	sourceLocal      string
	sourcePayloadKey string
}

type callableTargetAdder func(callee, paramName, targetSymbol string) bool
type callableTargetEdgeAdder func(callableTargetEdge)
type enumPayloadTargetEdgeAdder func(enumPayloadTargetEdge)

func addStructLiteralFieldEdgesForTargets(
	caller semantics.CheckedFunc,
	destFunc, destPrefix, structType string,
	value frontend.Expr,
	destFields map[string]semantics.FunctionFieldInfo,
	types map[string]*semantics.TypeInfo,
	funcs map[string]semantics.FuncSig,
	globals map[string]semantics.GlobalInfo,
	addTarget callableTargetAdder,
	addEdge callableTargetEdgeAdder,
) {
	lit, ok := value.(*frontend.StructLitExpr)
	if !ok || len(destFields) == 0 {
		return
	}
	typeInfo, ok := types[structType]
	if !ok || typeInfo.Kind != semantics.TypeStruct {
		return
	}
	for _, init := range lit.Fields {
		field, ok := typeInfo.FieldMap[init.Name]
		if !ok {
			continue
		}
		if field.FunctionTypeValue {
			if _, ok := destFields[init.Name]; ok {
				destFieldName := destPrefix + init.Name
				if target, ok := callableTargetFromAssignedExpr(init.Value, caller, funcs, globals); ok {
					addTarget(destFunc, destFieldName, target)
				}
				if sourceFieldName := functionTypedFieldNameFromExpr(init.Value); sourceFieldName != "" {
					addEdge(callableTargetEdge{
						callee:      destFunc,
						param:       destFieldName,
						sourceFunc:  caller.Name,
						sourceParam: sourceFieldName,
					})
				} else if id, ok := init.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && source.FunctionTypeValue {
						addEdge(callableTargetEdge{
							callee:      destFunc,
							param:       destFieldName,
							sourceFunc:  caller.Name,
							sourceParam: id.Name,
						})
					} else if source, exists := globals[id.Name]; exists && source.FunctionTypeValue {
						addEdge(callableTargetEdge{
							callee:      destFunc,
							param:       destFieldName,
							sourceFunc:  caller.Name,
							sourceParam: id.Name,
						})
					}
				}
				if call, ok := init.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, funcs); ok {
						if sourceSig, exists := funcs[resolved]; exists &&
							sourceSig.ReturnFunctionType {
							addEdge(callableTargetEdge{
								callee:      destFunc,
								param:       destFieldName,
								sourceFunc:  resolved,
								sourceParam: callableReturnFunctionLocal,
							})
						}
					}
				}
			}
		}
		nestedPrefix := init.Name + "."
		hasNestedDest := false
		for fieldName := range destFields {
			if strings.HasPrefix(fieldName, nestedPrefix) {
				hasNestedDest = true
				break
			}
		}
		if !hasNestedDest {
			continue
		}
		if call, ok := init.Value.(*frontend.CallExpr); ok {
			if resolved, ok := resolvedCallableFunctionName(call.Name, funcs); ok {
				if sourceSig, exists := funcs[resolved]; exists &&
					len(sourceSig.ReturnFunctionFields) > 0 {
					for fieldName, field := range destFields {
						if !strings.HasPrefix(fieldName, nestedPrefix) {
							continue
						}
						sourceField := strings.TrimPrefix(fieldName, nestedPrefix)
						if _, ok := sourceSig.ReturnFunctionFields[sourceField]; !ok {
							continue
						}
						destFieldName := destPrefix + fieldName
						if field.FunctionValue != "" {
							addTarget(destFunc, destFieldName, field.FunctionValue)
						}
						addEdge(callableTargetEdge{
							callee:      destFunc,
							param:       destFieldName,
							sourceFunc:  resolved,
							sourceParam: callableReturnFieldPrefix + sourceField,
						})
					}
				}
			}
		}
		sourcePrefix := functionTypedFieldNameFromExpr(init.Value)
		if sourcePrefix == "" {
			if id, ok := init.Value.(*frontend.IdentExpr); ok {
				sourcePrefix = id.Name
			}
		}
		if sourcePrefix != "" {
			for fieldName, field := range destFields {
				if !strings.HasPrefix(fieldName, nestedPrefix) {
					continue
				}
				sourceField := strings.TrimPrefix(fieldName, nestedPrefix)
				destFieldName := destPrefix + fieldName
				if field.FunctionValue != "" {
					addTarget(destFunc, destFieldName, field.FunctionValue)
				}
				addEdge(callableTargetEdge{
					callee:      destFunc,
					param:       destFieldName,
					sourceFunc:  caller.Name,
					sourceParam: sourcePrefix + "." + sourceField,
				})
			}
		}
		addStructLiteralFieldEdgesForTargets(
			caller,
			destFunc,
			destPrefix+nestedPrefix,
			field.TypeName,
			init.Value,
			trimFunctionFields(destFields, nestedPrefix),
			types,
			funcs,
			globals,
			addTarget,
			addEdge,
		)
	}
}

func addStructLiteralEnumPayloadFieldEdgesForTargets(
	caller semantics.CheckedFunc,
	destFunc, destPrefix, structType string,
	value frontend.Expr,
	destFields map[string]semantics.FunctionFieldInfo,
	types map[string]*semantics.TypeInfo,
	funcs map[string]semantics.FuncSig,
	addTarget callableTargetAdder,
	addEdge callableTargetEdgeAdder,
	addEnumPayloadEdge enumPayloadTargetEdgeAdder,
) {
	lit, ok := value.(*frontend.StructLitExpr)
	if !ok || len(destFields) == 0 {
		return
	}
	typeInfo, ok := types[structType]
	if !ok || typeInfo.Kind != semantics.TypeStruct {
		return
	}
	for _, init := range lit.Fields {
		field, ok := typeInfo.FieldMap[init.Name]
		if !ok {
			continue
		}
		if info, ok := types[field.TypeName]; ok && info.Kind == semantics.TypeEnum {
			fieldPrefix := init.Name + "#"
			if payloads := enumPayloadTargetsFromExpr(init.Value, caller, funcs, types); len(
				payloads,
			) > 0 {
				for payloadKey, payload := range payloads {
					fieldName := fieldPrefix + payloadKey
					if _, ok := destFields[fieldName]; !ok {
						continue
					}
					if payload.FunctionValue != "" {
						addTarget(destFunc, destPrefix+fieldName, payload.FunctionValue)
					}
				}
			}
			if call, ok := init.Value.(*frontend.CallExpr); ok {
				if resolved, ok := resolvedCallableFunctionName(call.Name, funcs); ok {
					if sourceSig, exists := funcs[resolved]; exists &&
						len(sourceSig.ReturnEnumPayloadFunctions) > 0 {
						for fieldName := range destFields {
							if !strings.HasPrefix(fieldName, fieldPrefix) {
								continue
							}
							payloadKey := strings.TrimPrefix(fieldName, fieldPrefix)
							if _, ok := sourceSig.ReturnEnumPayloadFunctions[payloadKey]; !ok {
								continue
							}
							addEnumPayloadEdge(enumPayloadTargetEdge{
								destFunc:         destFunc,
								destLocal:        strings.TrimSuffix(destPrefix+init.Name, "."),
								destPayloadKey:   payloadKey,
								sourceFunc:       resolved,
								sourceLocal:      callableReturnEnumPayloadLocal,
								sourcePayloadKey: payloadKey,
							})
						}
					}
				}
			}
		}
		nestedPrefix := init.Name + "."
		hasNestedDest := false
		for fieldName := range destFields {
			if strings.HasPrefix(fieldName, nestedPrefix) {
				hasNestedDest = true
				break
			}
		}
		if !hasNestedDest {
			continue
		}
		if call, ok := init.Value.(*frontend.CallExpr); ok {
			if resolved, ok := resolvedCallableFunctionName(call.Name, funcs); ok {
				if sourceSig, exists := funcs[resolved]; exists &&
					len(sourceSig.ReturnEnumPayloadFields) > 0 {
					for fieldName, field := range destFields {
						if !strings.HasPrefix(fieldName, nestedPrefix) {
							continue
						}
						sourceField := strings.TrimPrefix(fieldName, nestedPrefix)
						if _, ok := sourceSig.ReturnEnumPayloadFields[sourceField]; !ok {
							continue
						}
						destFieldName := destPrefix + fieldName
						if field.FunctionValue != "" {
							addTarget(destFunc, destFieldName, field.FunctionValue)
						}
						addEdge(callableTargetEdge{
							callee:      destFunc,
							param:       destFieldName,
							sourceFunc:  resolved,
							sourceParam: callableReturnFieldPrefix + sourceField,
						})
					}
				}
			}
		}
		sourcePrefix := functionTypedFieldNameFromExpr(init.Value)
		if sourcePrefix == "" {
			if id, ok := init.Value.(*frontend.IdentExpr); ok {
				sourcePrefix = id.Name
			}
		}
		if sourcePrefix != "" {
			for fieldName, field := range destFields {
				if !strings.HasPrefix(fieldName, nestedPrefix) {
					continue
				}
				sourceField := strings.TrimPrefix(fieldName, nestedPrefix)
				destFieldName := destPrefix + fieldName
				if field.FunctionValue != "" {
					addTarget(destFunc, destFieldName, field.FunctionValue)
				}
				addEdge(callableTargetEdge{
					callee:      destFunc,
					param:       destFieldName,
					sourceFunc:  caller.Name,
					sourceParam: sourcePrefix + "." + sourceField,
				})
			}
		}
		addStructLiteralEnumPayloadFieldEdgesForTargets(
			caller,
			destFunc,
			destPrefix+nestedPrefix,
			field.TypeName,
			init.Value,
			trimFunctionFields(destFields, nestedPrefix),
			types,
			funcs,
			addTarget,
			addEdge,
			addEnumPayloadEdge,
		)
	}
}

// ---- callable_targets.go ----

func callableTargetFromAssignedExpr(
	expr frontend.Expr,
	caller semantics.CheckedFunc,
	funcs map[string]semantics.FuncSig,
	globals map[string]semantics.GlobalInfo,
) (string, bool) {
	return lowercallables.TargetFromAssignedExpr(expr, caller, funcs, globals)
}

func callableClosureTargetName(
	caller semantics.CheckedFunc,
	closure *frontend.ClosureExpr,
	funcs map[string]semantics.FuncSig,
) string {
	return lowercallables.ClosureTargetName(caller, closure, funcs)
}

func enumPayloadTargetsFromExpr(
	expr frontend.Expr,
	caller semantics.CheckedFunc,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
) map[string]semantics.FunctionFieldInfo {
	return lowercallables.EnumPayloadTargetsFromExpr(expr, caller, funcs, types)
}

func enumPayloadFieldTargetsFromExpr(
	expr frontend.Expr,
	caller semantics.CheckedFunc,
) map[string]semantics.FunctionFieldInfo {
	return lowercallables.EnumPayloadFieldTargetsFromExpr(expr, caller)
}

func enumPayloadTargetKey(ordinal int32, index int) string {
	return lowercallables.EnumPayloadTargetKey(ordinal, index)
}

func enumPayloadTargetInfo(
	caseInfo semantics.EnumCaseInfo,
	index int,
	target string,
) semantics.FunctionFieldInfo {
	return lowercallables.EnumPayloadTargetInfo(caseInfo, index, target)
}

func enumCaseConstructorInfoForTargets(
	call *frontend.CallExpr,
	types map[string]*semantics.TypeInfo,
) (string, semantics.EnumCaseInfo, bool) {
	return lowercallables.EnumCaseConstructorInfoForTargets(call, types)
}

func enumCasePatternInfoForTargets(
	pattern *frontend.EnumCasePatternExpr,
	types map[string]*semantics.TypeInfo,
) (semantics.EnumCaseInfo, bool) {
	return lowercallables.EnumCasePatternInfoForTargets(pattern, types)
}

func resolvedCallableFunctionName(name string, funcs map[string]semantics.FuncSig) (string, bool) {
	return lowercallables.ResolvedFunctionName(name, funcs)
}

func trimFunctionFields(
	fields map[string]semantics.FunctionFieldInfo,
	prefix string,
) map[string]semantics.FunctionFieldInfo {
	return lowercallables.TrimFunctionFields(fields, prefix)
}

func resolveFunctionFieldName(
	name string,
	locals map[string]semantics.LocalInfo,
) (semantics.FunctionFieldInfo, bool, error) {
	return lowercallables.ResolveFunctionFieldName(name, locals)
}

func functionTypedFieldNameFromExpr(expr frontend.Expr) string {
	return lowercallables.FunctionTypedFieldNameFromExpr(expr)
}

func functionFieldTargetFromExpr(
	expr frontend.Expr,
	locals map[string]semantics.LocalInfo,
) (string, bool) {
	return lowercallables.FunctionFieldTargetFromExpr(expr, locals)
}

func functionTypedGlobalFieldTargetFromExpr(
	expr frontend.Expr,
	globals map[string]semantics.GlobalInfo,
) (string, bool) {
	return lowercallables.FunctionTypedGlobalFieldTargetFromExpr(expr, globals)
}

func (l *lowerer) functionFieldCallSource(
	name string,
	pos frontend.Position,
) (semantics.FunctionFieldInfo, int, bool, error) {
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		if len(parts) < 2 {
			return semantics.FunctionFieldInfo{}, 0, false, nil
		}
	}
	local, ok := l.locals[parts[0]]
	if !ok || len(local.FunctionFields) == 0 {
		return semantics.FunctionFieldInfo{}, 0, false, nil
	}
	fieldPath := parts[1:]
	field, ok := local.FunctionFields[strings.Join(fieldPath, ".")]
	if !ok {
		return semantics.FunctionFieldInfo{}, 0, false, nil
	}
	_, slotCount, base, err := resolveFieldChainLower(
		local.TypeName,
		local.Base,
		fieldPath,
		l.types,
		pos,
	)
	if err != nil {
		return semantics.FunctionFieldInfo{}, 0, false, err
	}
	if slotCount != semantics.FnPtrSlotCount {
		return semantics.FunctionFieldInfo{}, 0, false, fmt.Errorf(
			"%s: function-typed struct field '%s' slot mismatch",
			frontend.FormatPos(pos),
			name,
		)
	}
	return field, base, true, nil
}

func importedFunctionTargetFromExpr(
	expr frontend.Expr,
	imports map[string]string,
	funcs map[string]semantics.FuncSig,
) (string, bool) {
	return lowercallables.ImportedFunctionTargetFromExpr(expr, imports, funcs)
}

// ---- callables.go ----

func collectFunctionTypedParamTargets(
	checked *semantics.CheckedProgram,
	module string,
) map[string]map[string][]string {
	if checked == nil {
		return nil
	}
	const returnFieldPrefix = "$return."
	const returnFunctionLocal = "$return.fn"
	const returnEnumPayloadLocal = "$return.enum"
	funcsByName := make(map[string]semantics.CheckedFunc, len(checked.Funcs))
	for _, fn := range checked.Funcs {
		funcsByName[fn.Name] = fn
	}
	targetSets := map[string]map[string]map[string]bool{}
	var edges []callableTargetEdge
	var moduleGlobalEdges []moduleGlobalTargetEdge
	var enumPayloadEdges []enumPayloadTargetEdge

	addTarget := func(callee, paramName, targetSymbol string) bool {
		if callee == "" || paramName == "" || targetSymbol == "" {
			return false
		}
		if _, ok := targetSets[callee]; !ok {
			targetSets[callee] = map[string]map[string]bool{}
		}
		if _, ok := targetSets[callee][paramName]; !ok {
			targetSets[callee][paramName] = map[string]bool{}
		}
		if targetSets[callee][paramName][targetSymbol] {
			return false
		}
		targetSets[callee][paramName][targetSymbol] = true
		return true
	}

	addModuleGlobalTarget := func(moduleName, globalName, targetSymbol string) bool {
		changed := false
		for _, fn := range checked.Funcs {
			if fn.Module != moduleName {
				continue
			}
			if addTarget(fn.Name, globalName, targetSymbol) {
				changed = true
			}
		}
		return changed
	}

	enumPayloadSourceName := func(localName, payloadKey string) string {
		return "$enum." + localName + "." + payloadKey
	}

	enumPayloadTargetSets := map[string]map[string]map[string]map[string]bool{}
	addEnumPayloadTarget := func(funcName, localName, payloadKey, targetSymbol string) bool {
		if funcName == "" || localName == "" || payloadKey == "" || targetSymbol == "" {
			return false
		}
		if _, ok := enumPayloadTargetSets[funcName]; !ok {
			enumPayloadTargetSets[funcName] = map[string]map[string]map[string]bool{}
		}
		if _, ok := enumPayloadTargetSets[funcName][localName]; !ok {
			enumPayloadTargetSets[funcName][localName] = map[string]map[string]bool{}
		}
		if _, ok := enumPayloadTargetSets[funcName][localName][payloadKey]; !ok {
			enumPayloadTargetSets[funcName][localName][payloadKey] = map[string]bool{}
		}
		if enumPayloadTargetSets[funcName][localName][payloadKey][targetSymbol] {
			return false
		}
		enumPayloadTargetSets[funcName][localName][payloadKey][targetSymbol] = true
		addTarget(funcName, enumPayloadSourceName(localName, payloadKey), targetSymbol)
		return true
	}

	addEnumPayloadTargetsForLocal := func(caller semantics.CheckedFunc, localName string, payloads map[string]semantics.FunctionFieldInfo) {
		for payloadKey, payload := range payloads {
			if payload.FunctionValue != "" {
				addEnumPayloadTarget(caller.Name, localName, payloadKey, payload.FunctionValue)
			}
		}
	}

	addEnumPayloadFunctionReturnEdgesForLocal := func(caller semantics.CheckedFunc, localName string, value frontend.Expr) {
		call, ok := value.(*frontend.CallExpr)
		if !ok {
			return
		}
		_, caseInfo, ok := enumCaseConstructorInfoForTargets(call, checked.Types)
		if !ok {
			return
		}
		for i, arg := range call.Args {
			if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
				continue
			}
			argCall, ok := arg.(*frontend.CallExpr)
			if !ok {
				continue
			}
			resolved, ok := resolvedCallableFunctionName(argCall.Name, checked.FuncSigs)
			if !ok {
				continue
			}
			if sourceSig, exists := checked.FuncSigs[resolved]; exists &&
				sourceSig.ReturnFunctionType {
				edges = append(edges, callableTargetEdge{
					callee: caller.Name,
					param: enumPayloadSourceName(
						localName,
						enumPayloadTargetKey(caseInfo.Ordinal, i),
					),
					sourceFunc:  resolved,
					sourceParam: returnFunctionLocal,
				})
			}
		}
	}
	addEnumPayloadConstructorArgEdgesForLocal := func(caller semantics.CheckedFunc, localName string, value frontend.Expr) {
		call, ok := value.(*frontend.CallExpr)
		if !ok {
			return
		}
		_, caseInfo, ok := enumCaseConstructorInfoForTargets(call, checked.Types)
		if !ok {
			return
		}
		for i, arg := range call.Args {
			if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
				continue
			}
			payloadSource := enumPayloadSourceName(
				localName,
				enumPayloadTargetKey(caseInfo.Ordinal, i),
			)
			if target, ok := callableTargetFromAssignedExpr(
				arg,
				caller,
				checked.FuncSigs,
				checked.GlobalsByModule[caller.Module],
			); ok {
				addTarget(caller.Name, payloadSource, target)
			}
			if id, ok := arg.(*frontend.IdentExpr); ok {
				if source, exists := caller.Locals[id.Name]; exists && source.FunctionTypeValue {
					edges = append(edges, callableTargetEdge{
						callee:      caller.Name,
						param:       payloadSource,
						sourceFunc:  caller.Name,
						sourceParam: id.Name,
					})
				} else if source, exists := checked.GlobalsByModule[caller.Module][id.Name]; exists && source.FunctionTypeValue {
					edges = append(edges, callableTargetEdge{
						callee:      caller.Name,
						param:       payloadSource,
						sourceFunc:  caller.Name,
						sourceParam: id.Name,
					})
				}
				continue
			}
			if sourceFieldName := functionTypedFieldNameFromExpr(arg); sourceFieldName != "" {
				if _, sourceOK, _ := resolveFunctionFieldName(sourceFieldName, caller.Locals); sourceOK {
					edges = append(edges, callableTargetEdge{
						callee:      caller.Name,
						param:       payloadSource,
						sourceFunc:  caller.Name,
						sourceParam: sourceFieldName,
					})
				}
				continue
			}
			argCall, ok := arg.(*frontend.CallExpr)
			if !ok {
				continue
			}
			resolved, ok := resolvedCallableFunctionName(argCall.Name, checked.FuncSigs)
			if !ok {
				continue
			}
			if sourceSig, exists := checked.FuncSigs[resolved]; exists &&
				sourceSig.ReturnFunctionType {
				edges = append(edges, callableTargetEdge{
					callee:      caller.Name,
					param:       payloadSource,
					sourceFunc:  resolved,
					sourceParam: returnFunctionLocal,
				})
			}
		}
	}

	for _, fn := range checked.Funcs {
		if sig, ok := checked.FuncSigs[fn.Name]; ok {
			if sig.ReturnFunctionType && sig.ReturnFunctionSymbol != "" {
				addTarget(fn.Name, returnFunctionLocal, sig.ReturnFunctionSymbol)
			}
			for fieldName, field := range sig.ReturnFunctionFields {
				if field.FunctionValue != "" {
					addTarget(fn.Name, returnFieldPrefix+fieldName, field.FunctionValue)
				}
			}
			addEnumPayloadTargetsForLocal(
				fn,
				returnEnumPayloadLocal,
				sig.ReturnEnumPayloadFunctions,
			)
			for fieldName, field := range sig.ReturnEnumPayloadFields {
				if field.FunctionValue != "" {
					addTarget(fn.Name, returnFieldPrefix+fieldName, field.FunctionValue)
				}
			}
		}
		for name, local := range fn.Locals {
			if local.FunctionTypeValue && local.FunctionValue != "" {
				addTarget(fn.Name, name, local.FunctionValue)
			}
			for fieldName, field := range local.FunctionFields {
				if field.FunctionValue != "" {
					addTarget(fn.Name, name+"."+fieldName, field.FunctionValue)
				}
			}
			addEnumPayloadTargetsForLocal(fn, name, local.EnumPayloadFunctions)
		}
		for name, global := range checked.GlobalsByModule[fn.Module] {
			if global.FunctionTypeValue && global.FunctionValue != "" {
				addTarget(fn.Name, name, global.FunctionValue)
			}
		}
	}

	var walkExpr func(frontend.Expr, semantics.CheckedFunc)
	var walkStmt func(frontend.Stmt, semantics.CheckedFunc)

	addEdge := func(edge callableTargetEdge) {
		edges = append(edges, edge)
	}
	addEnumPayloadEdge := func(edge enumPayloadTargetEdge) {
		enumPayloadEdges = append(enumPayloadEdges, edge)
	}
	addStructLiteralFieldEdges := func(caller semantics.CheckedFunc, destFunc, destPrefix, structType string, value frontend.Expr, destFields map[string]semantics.FunctionFieldInfo) {
		addStructLiteralFieldEdgesForTargets(
			caller,
			destFunc,
			destPrefix,
			structType,
			value,
			destFields,
			checked.Types,
			checked.FuncSigs,
			checked.GlobalsByModule[caller.Module],
			addTarget,
			addEdge,
		)
	}
	addStructLiteralEnumPayloadFieldEdges := func(caller semantics.CheckedFunc, destFunc, destPrefix, structType string, value frontend.Expr, destFields map[string]semantics.FunctionFieldInfo) {
		addStructLiteralEnumPayloadFieldEdgesForTargets(
			caller,
			destFunc,
			destPrefix,
			structType,
			value,
			destFields,
			checked.Types,
			checked.FuncSigs,
			addTarget,
			addEdge,
			addEnumPayloadEdge,
		)
	}

	addCallTargets := func(call *frontend.CallExpr, caller semantics.CheckedFunc) {
		if local, ok := caller.Locals[call.Name]; ok && local.FunctionTypeValue &&
			local.FunctionValue != "" {
			addTarget(caller.Name, call.Name, local.FunctionValue)
		}
		resolved := call.Name
		if builtin, ok := semantics.ResolveBuiltinAlias(resolved); ok {
			resolved = builtin
		}
		calleeSig, ok := checked.FuncSigs[resolved]
		if !ok || len(calleeSig.ParamFunctionTypes) == 0 {
			return
		}
		callee, ok := funcsByName[resolved]
		if !ok || len(callee.Decl.Params) == 0 {
			return
		}
		for i, isFuncParam := range calleeSig.ParamFunctionTypes {
			if !isFuncParam || i >= len(call.Args) || i >= len(callee.Decl.Params) {
				continue
			}
			paramName := callee.Decl.Params[i].Name
			if paramName == "" {
				continue
			}
			if closure, ok := call.Args[i].(*frontend.ClosureExpr); ok {
				addTarget(
					resolved,
					paramName,
					callableClosureTargetName(caller, closure, checked.FuncSigs),
				)
				continue
			}
			if fieldName := functionTypedFieldNameFromExpr(call.Args[i]); fieldName != "" {
				if field, ok, _ := resolveFunctionFieldName(fieldName, caller.Locals); ok {
					if field.FunctionValue != "" {
						addTarget(resolved, paramName, field.FunctionValue)
					}
					edges = append(edges, callableTargetEdge{
						callee:      resolved,
						param:       paramName,
						sourceFunc:  caller.Name,
						sourceParam: fieldName,
					})
					continue
				}
				if global, ok := checked.GlobalsByModule[caller.Module][fieldName]; ok &&
					global.FunctionTypeValue {
					if global.FunctionValue != "" {
						addTarget(resolved, paramName, global.FunctionValue)
					}
					edges = append(edges, callableTargetEdge{
						callee:      resolved,
						param:       paramName,
						sourceFunc:  caller.Name,
						sourceParam: fieldName,
					})
					continue
				}
			}
			if argCall, ok := call.Args[i].(*frontend.CallExpr); ok {
				if source, ok := resolvedCallableFunctionName(argCall.Name, checked.FuncSigs); ok {
					if sourceSig, exists := checked.FuncSigs[source]; exists &&
						sourceSig.ReturnFunctionType {
						edges = append(edges, callableTargetEdge{
							callee:      resolved,
							param:       paramName,
							sourceFunc:  source,
							sourceParam: returnFunctionLocal,
						})
						continue
					}
				}
			}
			id, ok := call.Args[i].(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if local, ok := caller.Locals[id.Name]; ok {
				if !local.FunctionTypeValue && local.FunctionValue == "" {
					continue
				}
				if local.FunctionValue != "" {
					addTarget(resolved, paramName, local.FunctionValue)
				}
				edges = append(edges, callableTargetEdge{
					callee:      resolved,
					param:       paramName,
					sourceFunc:  caller.Name,
					sourceParam: id.Name,
				})
				continue
			}
			if global, ok := checked.GlobalsByModule[caller.Module][id.Name]; ok &&
				global.FunctionTypeValue {
				if global.FunctionValue != "" {
					addTarget(resolved, paramName, global.FunctionValue)
				}
				edges = append(edges, callableTargetEdge{
					callee:      resolved,
					param:       paramName,
					sourceFunc:  caller.Name,
					sourceParam: id.Name,
				})
				continue
			}
			if _, ok := checked.FuncSigs[id.Name]; ok {
				addTarget(resolved, paramName, id.Name)
			}
		}
		for i, param := range callee.Decl.Params {
			if i >= len(call.Args) {
				continue
			}
			paramLocal, ok := callee.Locals[param.Name]
			if !ok {
				continue
			}
			if len(paramLocal.FunctionFields) > 0 {
				sourceFields := map[string]semantics.FunctionFieldInfo(nil)
				sourcePrefix := ""
				if id, ok := call.Args[i].(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists &&
						len(source.FunctionFields) > 0 {
						sourceFields = source.FunctionFields
						sourcePrefix = id.Name + "."
					}
				}
				addStructLiteralFieldEdges(
					caller,
					resolved,
					param.Name+".",
					paramLocal.TypeName,
					call.Args[i],
					paramLocal.FunctionFields,
				)
				if len(sourceFields) > 0 {
					for fieldName := range paramLocal.FunctionFields {
						destFieldName := param.Name + "." + fieldName
						if source, ok := sourceFields[fieldName]; ok && source.FunctionValue != "" {
							addTarget(resolved, destFieldName, source.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      resolved,
							param:       destFieldName,
							sourceFunc:  caller.Name,
							sourceParam: sourcePrefix + fieldName,
						})
					}
				}
			}
			if len(paramLocal.EnumPayloadFunctions) > 0 {
				if payloads := enumPayloadTargetsFromExpr(
					call.Args[i],
					caller,
					checked.FuncSigs,
					checked.Types,
				); len(
					payloads,
				) > 0 {
					addEnumPayloadTargetsForLocal(
						semantics.CheckedFunc{Name: resolved},
						param.Name,
						payloads,
					)
				}
				if id, ok := call.Args[i].(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists &&
						len(source.EnumPayloadFunctions) > 0 {
						for payloadKey := range paramLocal.EnumPayloadFunctions {
							enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
								destFunc:         resolved,
								destLocal:        param.Name,
								destPayloadKey:   payloadKey,
								sourceFunc:       caller.Name,
								sourceLocal:      id.Name,
								sourcePayloadKey: payloadKey,
							})
						}
					}
				} else if argCall, ok := call.Args[i].(*frontend.CallExpr); ok {
					if sourceName, ok := resolvedCallableFunctionName(argCall.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[sourceName]; exists && len(
							sourceSig.ReturnEnumPayloadFunctions,
						) > 0 {
							for payloadKey := range paramLocal.EnumPayloadFunctions {
								enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
									destFunc:         resolved,
									destLocal:        param.Name,
									destPayloadKey:   payloadKey,
									sourceFunc:       sourceName,
									sourceLocal:      returnEnumPayloadLocal,
									sourcePayloadKey: payloadKey,
								})
							}
						}
					}
				}
			}
		}
	}

	walkExpr = func(expr frontend.Expr, caller semantics.CheckedFunc) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			addCallTargets(e, caller)
			for _, arg := range e.Args {
				walkExpr(arg, caller)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value, caller)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base, caller)
		case *frontend.IndexExpr:
			walkExpr(e.Base, caller)
			walkExpr(e.Index, caller)
		case *frontend.BinaryExpr:
			walkExpr(e.Left, caller)
			walkExpr(e.Right, caller)
		case *frontend.UnaryExpr:
			walkExpr(e.X, caller)
		case *frontend.TryExpr:
			walkExpr(e.X, caller)
		case *frontend.CatchExpr:
			walkExpr(e.Call, caller)
		case *frontend.AwaitExpr:
			walkExpr(e.X, caller)
		}
	}

	walkStmt = func(stmt frontend.Stmt, caller semantics.CheckedFunc) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value, caller)
		case *frontend.ExpectStmt:
			walkExpr(s.Cond, caller)
		case *frontend.ReturnStmt:
			if sig, ok := checked.FuncSigs[caller.Name]; ok && sig.ReturnFunctionType {
				if target, ok := callableTargetFromAssignedExpr(
					s.Value,
					caller,
					checked.FuncSigs,
					checked.GlobalsByModule[caller.Module],
				); ok {
					addTarget(caller.Name, returnFunctionLocal, target)
				}
				if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && source.FunctionTypeValue {
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       returnFunctionLocal,
							sourceFunc:  caller.Name,
							sourceParam: id.Name,
						})
					} else if source, exists := checked.GlobalsByModule[caller.Module][id.Name]; exists && source.FunctionTypeValue {
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       returnFunctionLocal,
							sourceFunc:  caller.Name,
							sourceParam: id.Name,
						})
					}
				} else if sourceFieldName := functionTypedFieldNameFromExpr(s.Value); sourceFieldName != "" {
					if _, sourceOK, _ := resolveFunctionFieldName(sourceFieldName, caller.Locals); sourceOK {
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       returnFunctionLocal,
							sourceFunc:  caller.Name,
							sourceParam: sourceFieldName,
						})
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && sourceSig.ReturnFunctionType {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       returnFunctionLocal,
								sourceFunc:  resolved,
								sourceParam: returnFunctionLocal,
							})
						}
					}
				}
			}
			if sig, ok := checked.FuncSigs[caller.Name]; ok && len(sig.ReturnFunctionFields) > 0 {
				if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
					for fieldName, field := range sig.ReturnFunctionFields {
						returnFieldName := returnFieldPrefix + fieldName
						if field.FunctionValue != "" {
							addTarget(caller.Name, returnFieldName, field.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       returnFieldName,
							sourceFunc:  caller.Name,
							sourceParam: sourcePrefix + "." + fieldName,
						})
					}
				} else if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.FunctionFields) > 0 {
						for fieldName, field := range sig.ReturnFunctionFields {
							returnFieldName := returnFieldPrefix + fieldName
							if field.FunctionValue != "" {
								addTarget(caller.Name, returnFieldName, field.FunctionValue)
							}
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       returnFieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name + "." + fieldName,
							})
						}
					}
				} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
					addStructLiteralFieldEdges(
						caller,
						caller.Name,
						returnFieldPrefix,
						sig.ReturnType,
						s.Value,
						sig.ReturnFunctionFields,
					)
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(
							sourceSig.ReturnFunctionFields,
						) > 0 {
							for fieldName, field := range sig.ReturnFunctionFields {
								returnFieldName := returnFieldPrefix + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, returnFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       returnFieldName,
									sourceFunc:  resolved,
									sourceParam: returnFieldPrefix + fieldName,
								})
							}
						}
					}
				}
			}
			if sig, ok := checked.FuncSigs[caller.Name]; ok && len(sig.ReturnEnumPayloadFunctions) > 0 {
				if payloads := enumPayloadTargetsFromExpr(
					s.Value,
					caller,
					checked.FuncSigs,
					checked.Types,
				); len(
					payloads,
				) > 0 {
					addEnumPayloadTargetsForLocal(caller, returnEnumPayloadLocal, payloads)
				}
				addEnumPayloadConstructorArgEdgesForLocal(caller, returnEnumPayloadLocal, s.Value)
				if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.EnumPayloadFunctions) > 0 {
						for payloadKey := range sig.ReturnEnumPayloadFunctions {
							enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
								destFunc:         caller.Name,
								destLocal:        returnEnumPayloadLocal,
								destPayloadKey:   payloadKey,
								sourceFunc:       caller.Name,
								sourceLocal:      id.Name,
								sourcePayloadKey: payloadKey,
							})
						}
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(
							sourceSig.ReturnEnumPayloadFunctions,
						) > 0 {
							for payloadKey := range sig.ReturnEnumPayloadFunctions {
								enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
									destFunc:         caller.Name,
									destLocal:        returnEnumPayloadLocal,
									destPayloadKey:   payloadKey,
									sourceFunc:       resolved,
									sourceLocal:      returnEnumPayloadLocal,
									sourcePayloadKey: payloadKey,
								})
							}
						}
					}
				}
			}
			if sig, ok := checked.FuncSigs[caller.Name]; ok && len(sig.ReturnEnumPayloadFields) > 0 {
				if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
					for fieldName, field := range sig.ReturnEnumPayloadFields {
						returnFieldName := returnFieldPrefix + fieldName
						if field.FunctionValue != "" {
							addTarget(caller.Name, returnFieldName, field.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       returnFieldName,
							sourceFunc:  caller.Name,
							sourceParam: sourcePrefix + "." + fieldName,
						})
					}
				} else if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.EnumPayloadFields) > 0 {
						for fieldName, field := range sig.ReturnEnumPayloadFields {
							returnFieldName := returnFieldPrefix + fieldName
							if field.FunctionValue != "" {
								addTarget(caller.Name, returnFieldName, field.FunctionValue)
							}
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       returnFieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name + "." + fieldName,
							})
						}
					}
				} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
					addStructLiteralEnumPayloadFieldEdges(
						caller,
						caller.Name,
						returnFieldPrefix,
						sig.ReturnType,
						s.Value,
						sig.ReturnEnumPayloadFields,
					)
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(
							sourceSig.ReturnEnumPayloadFields,
						) > 0 {
							for fieldName, field := range sig.ReturnEnumPayloadFields {
								returnFieldName := returnFieldPrefix + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, returnFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       returnFieldName,
									sourceFunc:  resolved,
									sourceParam: returnFieldPrefix + fieldName,
								})
							}
						}
					}
				}
			}
			walkExpr(s.Value, caller)
		case *frontend.ThrowStmt:
			walkExpr(s.Value, caller)
		case *frontend.LetStmt:
			if payloads := enumPayloadTargetsFromExpr(
				s.Value,
				caller,
				checked.FuncSigs,
				checked.Types,
			); len(
				payloads,
			) > 0 {
				addEnumPayloadTargetsForLocal(caller, s.Name, payloads)
				addEnumPayloadFunctionReturnEdgesForLocal(caller, s.Name, s.Value)
			}
			if dest, ok := caller.Locals[s.Name]; ok && len(dest.EnumPayloadFunctions) > 0 {
				addEnumPayloadConstructorArgEdgesForLocal(caller, s.Name, s.Value)
				if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.EnumPayloadFunctions) > 0 {
						for payloadKey := range dest.EnumPayloadFunctions {
							enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
								destFunc:         caller.Name,
								destLocal:        s.Name,
								destPayloadKey:   payloadKey,
								sourceFunc:       caller.Name,
								sourceLocal:      id.Name,
								sourcePayloadKey: payloadKey,
							})
						}
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(
							sourceSig.ReturnEnumPayloadFunctions,
						) > 0 {
							for payloadKey := range dest.EnumPayloadFunctions {
								enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
									destFunc:         caller.Name,
									destLocal:        s.Name,
									destPayloadKey:   payloadKey,
									sourceFunc:       resolved,
									sourceLocal:      returnEnumPayloadLocal,
									sourcePayloadKey: payloadKey,
								})
							}
						}
					}
				}
			}
			if dest, ok := caller.Locals[s.Name]; ok && dest.FunctionTypeValue {
				if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && source.FunctionTypeValue {
						if source.FunctionValue != "" {
							addTarget(caller.Name, s.Name, source.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       s.Name,
							sourceFunc:  caller.Name,
							sourceParam: id.Name,
						})
					}
				} else if fieldName := functionTypedFieldNameFromExpr(s.Value); fieldName != "" {
					if field, ok, _ := resolveFunctionFieldName(fieldName, caller.Locals); ok {
						if field.FunctionValue != "" {
							addTarget(caller.Name, s.Name, field.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       s.Name,
							sourceFunc:  caller.Name,
							sourceParam: fieldName,
						})
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && sourceSig.ReturnFunctionType {
							if sourceSig.ReturnFunctionSymbol != "" {
								addTarget(caller.Name, s.Name, sourceSig.ReturnFunctionSymbol)
							}
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       s.Name,
								sourceFunc:  resolved,
								sourceParam: returnFunctionLocal,
							})
						}
					}
				}
			}
			if dest, ok := caller.Locals[s.Name]; ok && len(dest.FunctionFields) > 0 {
				if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
					for fieldName, field := range dest.FunctionFields {
						destFieldName := s.Name + "." + fieldName
						if field.FunctionValue != "" {
							addTarget(caller.Name, destFieldName, field.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       destFieldName,
							sourceFunc:  caller.Name,
							sourceParam: sourcePrefix + "." + fieldName,
						})
					}
				} else if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.FunctionFields) > 0 {
						for fieldName, field := range dest.FunctionFields {
							destFieldName := s.Name + "." + fieldName
							if field.FunctionValue != "" {
								addTarget(caller.Name, destFieldName, field.FunctionValue)
							}
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       destFieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name + "." + fieldName,
							})
						}
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(
							sourceSig.ReturnFunctionFields,
						) > 0 {
							for fieldName, field := range dest.FunctionFields {
								destFieldName := s.Name + "." + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, destFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       destFieldName,
									sourceFunc:  resolved,
									sourceParam: returnFieldPrefix + fieldName,
								})
							}
						}
					}
				} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
					addStructLiteralFieldEdges(
						caller,
						caller.Name,
						s.Name+".",
						dest.TypeName,
						s.Value,
						dest.FunctionFields,
					)
				}
			}
			if dest, ok := caller.Locals[s.Name]; ok && len(dest.EnumPayloadFields) > 0 {
				if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
					for fieldName, field := range dest.EnumPayloadFields {
						destFieldName := s.Name + "." + fieldName
						if field.FunctionValue != "" {
							addTarget(caller.Name, destFieldName, field.FunctionValue)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       destFieldName,
							sourceFunc:  caller.Name,
							sourceParam: sourcePrefix + "." + fieldName,
						})
					}
				} else if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if source, exists := caller.Locals[id.Name]; exists && len(source.EnumPayloadFields) > 0 {
						for fieldName, field := range dest.EnumPayloadFields {
							destFieldName := s.Name + "." + fieldName
							if field.FunctionValue != "" {
								addTarget(caller.Name, destFieldName, field.FunctionValue)
							}
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       destFieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name + "." + fieldName,
							})
						}
					}
				} else if call, ok := s.Value.(*frontend.CallExpr); ok {
					if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
						if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(
							sourceSig.ReturnEnumPayloadFields,
						) > 0 {
							for fieldName, field := range dest.EnumPayloadFields {
								destFieldName := s.Name + "." + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, destFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       destFieldName,
									sourceFunc:  resolved,
									sourceParam: returnFieldPrefix + fieldName,
								})
							}
						}
					}
				} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
					addStructLiteralEnumPayloadFieldEdges(
						caller,
						caller.Name,
						s.Name+".",
						dest.TypeName,
						s.Value,
						dest.EnumPayloadFields,
					)
				}
			}
			walkExpr(s.Value, caller)
		case *frontend.AssignStmt:
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if local, exists := caller.Locals[id.Name]; exists && local.FunctionTypeValue {
					if target, ok := callableTargetFromAssignedExpr(
						s.Value,
						caller,
						checked.FuncSigs,
						checked.GlobalsByModule[caller.Module],
					); ok {
						addTarget(caller.Name, id.Name, target)
					}
					if valueID, ok := s.Value.(*frontend.IdentExpr); ok {
						if source, sourceExists := checked.GlobalsByModule[caller.Module][valueID.Name]; sourceExists && source.FunctionTypeValue {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       id.Name,
								sourceFunc:  caller.Name,
								sourceParam: valueID.Name,
							})
						}
					}
					if call, ok := s.Value.(*frontend.CallExpr); ok {
						if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
							if sourceSig, exists := checked.FuncSigs[resolved]; exists && sourceSig.ReturnFunctionType {
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       id.Name,
									sourceFunc:  resolved,
									sourceParam: returnFunctionLocal,
								})
							}
						}
					}
				}
				if global, exists := checked.GlobalsByModule[caller.Module][id.Name]; exists && global.FunctionTypeValue {
					if target, ok := callableTargetFromAssignedExpr(
						s.Value,
						caller,
						checked.FuncSigs,
						checked.GlobalsByModule[caller.Module],
					); ok {
						addTarget(caller.Name, id.Name, target)
						if global.Mutable {
							addModuleGlobalTarget(caller.Module, id.Name, target)
						}
					}
					if valueID, ok := s.Value.(*frontend.IdentExpr); ok {
						if source, sourceExists := caller.Locals[valueID.Name]; sourceExists && source.FunctionTypeValue {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       id.Name,
								sourceFunc:  caller.Name,
								sourceParam: valueID.Name,
							})
							if global.Mutable {
								moduleGlobalEdges = append(moduleGlobalEdges, moduleGlobalTargetEdge{
									module:      caller.Module,
									global:      id.Name,
									sourceFunc:  caller.Name,
									sourceParam: valueID.Name,
								})
							}
						}
					} else if sourceFieldName := functionTypedFieldNameFromExpr(s.Value); sourceFieldName != "" {
						if _, sourceOK, _ := resolveFunctionFieldName(sourceFieldName, caller.Locals); sourceOK {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       id.Name,
								sourceFunc:  caller.Name,
								sourceParam: sourceFieldName,
							})
							if global.Mutable {
								moduleGlobalEdges = append(moduleGlobalEdges, moduleGlobalTargetEdge{
									module:      caller.Module,
									global:      id.Name,
									sourceFunc:  caller.Name,
									sourceParam: sourceFieldName,
								})
							}
						}
					} else if call, ok := s.Value.(*frontend.CallExpr); ok {
						if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
							if sourceSig, sourceExists := checked.FuncSigs[resolved]; sourceExists && sourceSig.ReturnFunctionType {
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       id.Name,
									sourceFunc:  resolved,
									sourceParam: returnFunctionLocal,
								})
							}
						}
					}
				}
				if local, exists := caller.Locals[id.Name]; exists {
					if info, ok := checked.Types[local.TypeName]; ok && info.Kind == semantics.TypeEnum {
						if payloads := enumPayloadTargetsFromExpr(
							s.Value,
							caller,
							checked.FuncSigs,
							checked.Types,
						); len(
							payloads,
						) > 0 {
							addEnumPayloadTargetsForLocal(caller, id.Name, payloads)
							addEnumPayloadFunctionReturnEdgesForLocal(caller, id.Name, s.Value)
						}
						addEnumPayloadConstructorArgEdgesForLocal(caller, id.Name, s.Value)
						if idValue, ok := s.Value.(*frontend.IdentExpr); ok {
							if source, exists := caller.Locals[idValue.Name]; exists && len(
								source.EnumPayloadFunctions,
							) > 0 {
								for payloadKey := range local.EnumPayloadFunctions {
									enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
										destFunc:         caller.Name,
										destLocal:        id.Name,
										destPayloadKey:   payloadKey,
										sourceFunc:       caller.Name,
										sourceLocal:      idValue.Name,
										sourcePayloadKey: payloadKey,
									})
								}
							}
						} else if call, ok := s.Value.(*frontend.CallExpr); ok {
							if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
								if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(
									sourceSig.ReturnEnumPayloadFunctions,
								) > 0 {
									for payloadKey := range local.EnumPayloadFunctions {
										enumPayloadEdges = append(enumPayloadEdges, enumPayloadTargetEdge{
											destFunc:         caller.Name,
											destLocal:        id.Name,
											destPayloadKey:   payloadKey,
											sourceFunc:       resolved,
											sourceLocal:      returnEnumPayloadLocal,
											sourcePayloadKey: payloadKey,
										})
									}
								}
							}
						}
					}
					if info, ok := checked.Types[local.TypeName]; ok && info.Kind == semantics.TypeStruct && len(
						local.FunctionFields,
					) > 0 {
						if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
							for fieldName, field := range local.FunctionFields {
								destFieldName := id.Name + "." + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, destFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       destFieldName,
									sourceFunc:  caller.Name,
									sourceParam: sourcePrefix + "." + fieldName,
								})
							}
						} else if valueID, ok := s.Value.(*frontend.IdentExpr); ok {
							if source, exists := caller.Locals[valueID.Name]; exists && len(source.FunctionFields) > 0 {
								for fieldName, field := range local.FunctionFields {
									destFieldName := id.Name + "." + fieldName
									if field.FunctionValue != "" {
										addTarget(caller.Name, destFieldName, field.FunctionValue)
									}
									edges = append(edges, callableTargetEdge{
										callee:      caller.Name,
										param:       destFieldName,
										sourceFunc:  caller.Name,
										sourceParam: valueID.Name + "." + fieldName,
									})
								}
							}
						} else if call, ok := s.Value.(*frontend.CallExpr); ok {
							if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
								if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(
									sourceSig.ReturnFunctionFields,
								) > 0 {
									for fieldName, field := range local.FunctionFields {
										destFieldName := id.Name + "." + fieldName
										if field.FunctionValue != "" {
											addTarget(caller.Name, destFieldName, field.FunctionValue)
										}
										edges = append(edges, callableTargetEdge{
											callee:      caller.Name,
											param:       destFieldName,
											sourceFunc:  resolved,
											sourceParam: returnFieldPrefix + fieldName,
										})
									}
								}
							}
						} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
							addStructLiteralFieldEdges(
								caller,
								caller.Name,
								id.Name+".",
								local.TypeName,
								s.Value,
								local.FunctionFields,
							)
						}
					}
					if info, ok := checked.Types[local.TypeName]; ok && info.Kind == semantics.TypeStruct && len(
						local.EnumPayloadFields,
					) > 0 {
						if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
							for fieldName, field := range local.EnumPayloadFields {
								destFieldName := id.Name + "." + fieldName
								if field.FunctionValue != "" {
									addTarget(caller.Name, destFieldName, field.FunctionValue)
								}
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       destFieldName,
									sourceFunc:  caller.Name,
									sourceParam: sourcePrefix + "." + fieldName,
								})
							}
						} else if valueID, ok := s.Value.(*frontend.IdentExpr); ok {
							if source, exists := caller.Locals[valueID.Name]; exists && len(
								source.EnumPayloadFields,
							) > 0 {
								for fieldName, field := range local.EnumPayloadFields {
									destFieldName := id.Name + "." + fieldName
									if field.FunctionValue != "" {
										addTarget(caller.Name, destFieldName, field.FunctionValue)
									}
									edges = append(edges, callableTargetEdge{
										callee:      caller.Name,
										param:       destFieldName,
										sourceFunc:  caller.Name,
										sourceParam: valueID.Name + "." + fieldName,
									})
								}
							}
						} else if call, ok := s.Value.(*frontend.CallExpr); ok {
							if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
								if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(
									sourceSig.ReturnEnumPayloadFields,
								) > 0 {
									for fieldName, field := range local.EnumPayloadFields {
										destFieldName := id.Name + "." + fieldName
										if field.FunctionValue != "" {
											addTarget(caller.Name, destFieldName, field.FunctionValue)
										}
										edges = append(edges, callableTargetEdge{
											callee:      caller.Name,
											param:       destFieldName,
											sourceFunc:  resolved,
											sourceParam: returnFieldPrefix + fieldName,
										})
									}
								}
							}
						} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
							addStructLiteralEnumPayloadFieldEdges(
								caller,
								caller.Name,
								id.Name+".",
								local.TypeName,
								s.Value,
								local.EnumPayloadFields,
							)
						}
					}
				}
			} else if fieldName := functionTypedFieldNameFromExpr(s.Target); fieldName != "" {
				if _, ok, _ := resolveFunctionFieldName(fieldName, caller.Locals); ok {
					if target, ok := callableTargetFromAssignedExpr(
						s.Value,
						caller,
						checked.FuncSigs,
						checked.GlobalsByModule[caller.Module],
					); ok {
						addTarget(caller.Name, fieldName, target)
					}
					if id, ok := s.Value.(*frontend.IdentExpr); ok {
						if source, exists := caller.Locals[id.Name]; exists && source.FunctionTypeValue {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       fieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name,
							})
						} else if source, exists := checked.GlobalsByModule[caller.Module][id.Name]; exists && source.FunctionTypeValue {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       fieldName,
								sourceFunc:  caller.Name,
								sourceParam: id.Name,
							})
						}
					} else if sourceFieldName := functionTypedFieldNameFromExpr(s.Value); sourceFieldName != "" {
						if _, sourceOK, _ := resolveFunctionFieldName(sourceFieldName, caller.Locals); sourceOK {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       fieldName,
								sourceFunc:  caller.Name,
								sourceParam: sourceFieldName,
							})
						}
					} else if call, ok := s.Value.(*frontend.CallExpr); ok {
						if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
							if sourceSig, exists := checked.FuncSigs[resolved]; exists && sourceSig.ReturnFunctionType {
								edges = append(edges, callableTargetEdge{
									callee:      caller.Name,
									param:       fieldName,
									sourceFunc:  resolved,
									sourceParam: returnFunctionLocal,
								})
							}
						}
					}
				} else {
					parts := strings.Split(fieldName, ".")
					if len(parts) >= 2 {
						baseName := parts[0]
						fieldPath := parts[1:]
						if local, exists := caller.Locals[baseName]; exists && len(local.FunctionFields) > 0 {
							targetType, _, _, err := resolveFieldChainLower(
								local.TypeName,
								local.Base,
								fieldPath,
								checked.Types,
								s.Target.Pos(),
							)
							if err == nil {
								if info, ok := checked.Types[targetType]; ok && info.Kind == semantics.TypeStruct {
									fieldPrefix := strings.Join(fieldPath, ".") + "."
									destFields := trimFunctionFields(local.FunctionFields, fieldPrefix)
									if len(destFields) > 0 {
										destPrefix := fieldName + "."
										if sourcePrefix := functionTypedFieldNameFromExpr(s.Value); sourcePrefix != "" {
											for nestedName, field := range destFields {
												destFieldName := destPrefix + nestedName
												if field.FunctionValue != "" {
													addTarget(caller.Name, destFieldName, field.FunctionValue)
												}
												edges = append(edges, callableTargetEdge{
													callee:      caller.Name,
													param:       destFieldName,
													sourceFunc:  caller.Name,
													sourceParam: sourcePrefix + "." + nestedName,
												})
											}
										} else if valueID, ok := s.Value.(*frontend.IdentExpr); ok {
											if source, exists := caller.Locals[valueID.Name]; exists && len(
												source.FunctionFields,
											) > 0 {
												for nestedName, field := range destFields {
													destFieldName := destPrefix + nestedName
													if field.FunctionValue != "" {
														addTarget(caller.Name, destFieldName, field.FunctionValue)
													}
													edges = append(edges, callableTargetEdge{
														callee:      caller.Name,
														param:       destFieldName,
														sourceFunc:  caller.Name,
														sourceParam: valueID.Name + "." + nestedName,
													})
												}
											}
										} else if call, ok := s.Value.(*frontend.CallExpr); ok {
											if resolved, ok := resolvedCallableFunctionName(call.Name, checked.FuncSigs); ok {
												if sourceSig, exists := checked.FuncSigs[resolved]; exists && len(
													sourceSig.ReturnFunctionFields,
												) > 0 {
													for nestedName, field := range destFields {
														destFieldName := destPrefix + nestedName
														if field.FunctionValue != "" {
															addTarget(caller.Name, destFieldName, field.FunctionValue)
														}
														edges = append(edges, callableTargetEdge{
															callee:      caller.Name,
															param:       destFieldName,
															sourceFunc:  resolved,
															sourceParam: returnFieldPrefix + nestedName,
														})
													}
												}
											}
										} else if _, ok := s.Value.(*frontend.StructLitExpr); ok {
											addStructLiteralFieldEdges(
												caller,
												caller.Name,
												destPrefix,
												targetType,
												s.Value,
												destFields,
											)
										}
									}
								}
							}
						}
					}
				}
			}
			walkExpr(s.Target, caller)
			walkExpr(s.Value, caller)
		case *frontend.ExprStmt:
			walkExpr(s.Expr, caller)
		case *frontend.IfStmt:
			walkExpr(s.Cond, caller)
			for _, inner := range s.Then {
				walkStmt(inner, caller)
			}
			for _, inner := range s.Else {
				walkStmt(inner, caller)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value, caller)
			for _, inner := range s.Then {
				walkStmt(inner, caller)
			}
			for _, inner := range s.Else {
				walkStmt(inner, caller)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond, caller)
			for _, inner := range s.Body {
				walkStmt(inner, caller)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable, caller)
			} else {
				walkExpr(s.Start, caller)
				walkExpr(s.End, caller)
			}
			for _, inner := range s.Body {
				walkStmt(inner, caller)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value, caller)
			if id, ok := s.Value.(*frontend.IdentExpr); ok {
				for _, c := range s.Cases {
					enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr)
					if !ok {
						continue
					}
					caseInfo, ok := enumCasePatternInfoForTargets(enumPat, checked.Types)
					if !ok {
						continue
					}
					for i, binding := range enumPat.Bindings {
						if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
							continue
						}
						payloadKey := enumPayloadTargetKey(caseInfo.Ordinal, i)
						for target := range enumPayloadTargetSets[caller.Name][id.Name][payloadKey] {
							addTarget(caller.Name, binding, target)
						}
						edges = append(edges, callableTargetEdge{
							callee:      caller.Name,
							param:       binding,
							sourceFunc:  caller.Name,
							sourceParam: enumPayloadSourceName(id.Name, payloadKey),
						})
					}
				}
			} else if payloads := enumPayloadTargetsFromExpr(
				s.Value,
				caller,
				checked.FuncSigs,
				checked.Types,
			); len(
				payloads,
			) > 0 {
				sourcePrefix := functionTypedFieldNameFromExpr(s.Value)
				for _, c := range s.Cases {
					enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr)
					if !ok {
						continue
					}
					caseInfo, ok := enumCasePatternInfoForTargets(enumPat, checked.Types)
					if !ok {
						continue
					}
					for i, binding := range enumPat.Bindings {
						if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
							continue
						}
						payloadKey := enumPayloadTargetKey(caseInfo.Ordinal, i)
						payload, ok := payloads[payloadKey]
						if !ok {
							continue
						}
						if payload.FunctionValue != "" {
							addTarget(caller.Name, binding, payload.FunctionValue)
						}
						if sourcePrefix != "" {
							edges = append(edges, callableTargetEdge{
								callee:      caller.Name,
								param:       binding,
								sourceFunc:  caller.Name,
								sourceParam: sourcePrefix + "#" + payloadKey,
							})
						}
					}
				}
			}
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern, caller)
				}
				for _, inner := range c.Body {
					walkStmt(inner, caller)
				}
			}
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner, caller)
			}
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner, caller)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size, caller)
			for _, inner := range s.Body {
				walkStmt(inner, caller)
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value, caller)
		}
	}

	for _, fn := range checked.Funcs {
		if module != "" && fn.Module != module {
			continue
		}
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt, fn)
		}
	}

	for changed := true; changed; {
		changed = false
		for _, edge := range enumPayloadEdges {
			for target := range enumPayloadTargetSets[edge.sourceFunc][edge.sourceLocal][edge.sourcePayloadKey] {
				if addEnumPayloadTarget(
					edge.destFunc,
					edge.destLocal,
					edge.destPayloadKey,
					target,
				) {
					changed = true
				}
			}
			for target := range targetSets[edge.sourceFunc][enumPayloadSourceName(
				edge.sourceLocal,
				edge.sourcePayloadKey,
			)] {
				if addEnumPayloadTarget(
					edge.destFunc,
					edge.destLocal,
					edge.destPayloadKey,
					target,
				) {
					changed = true
				}
			}
		}
		for _, edge := range edges {
			for target := range targetSets[edge.sourceFunc][edge.sourceParam] {
				if addTarget(edge.callee, edge.param, target) {
					changed = true
				}
			}
		}
		for _, edge := range moduleGlobalEdges {
			for target := range targetSets[edge.sourceFunc][edge.sourceParam] {
				if addModuleGlobalTarget(edge.module, edge.global, target) {
					changed = true
				}
			}
		}
	}

	out := map[string]map[string][]string{}
	for funcName, params := range targetSets {
		out[funcName] = map[string][]string{}
		for paramName, symbols := range params {
			list := make([]string, 0, len(symbols))
			for symbol := range symbols {
				list = append(list, symbol)
			}
			sort.Strings(list)
			out[funcName][paramName] = list
		}
	}
	return out
}
