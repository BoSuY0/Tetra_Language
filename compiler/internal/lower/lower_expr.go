package lower

import (
	"fmt"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowerexpressions "tetra_language/compiler/internal/lower/expressions"
	"tetra_language/compiler/internal/semantics"
)

func (l *lowerer) lowerExpr(expr frontend.Expr) (int, error) {
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		return l.lowerMatchExpr(e)
	case *frontend.CatchExpr:
		return l.lowerCatchExpr(e)
	case *frontend.NumberExpr:
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: e.Value, Pos: e.At})
		return 1, nil
	case *frontend.BoolLitExpr:
		if e.Value {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		}
		return 1, nil
	case *frontend.NoneLitExpr:
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		return 2, nil
	case *frontend.StringLitExpr:
		l.emit(ir.IRInstr{Kind: ir.IRStrLit, Str: e.Value, Pos: e.At})
		return 2, nil
	case *frontend.IdentExpr:
		info, ok := l.locals[e.Name]
		if !ok {
			if g, ok := l.globals[e.Name]; ok {
				if g.FunctionTypeValue && g.FunctionValue != "" {
					if g.Mutable {
						l.emitGlobalFunctionValueInitIfNeeded(g, e.At)
						slotCount := gSlotCount(g.TypeName, l.types)
						for i := 0; i < slotCount; i++ {
							l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex + i, Pos: e.At})
						}
						return slotCount, nil
					}
					return l.emitFunctionSymbolValue(g.FunctionValue, nil, e.At), nil
				}
				if g.TypeName == "str" && g.HasStringLiteralInit {
					l.emitGlobalStringLiteralInitIfNeeded(g, e.At)
				}
				l.emitGlobalArrayBackingsInitIfNeeded(g, e.At)
				slotCount := gSlotCount(g.TypeName, l.types)
				for i := 0; i < slotCount; i++ {
					l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex + i, Pos: e.At})
				}
				return slotCount, nil
			}
			if sig, ok := l.funcs[e.Name]; ok {
				if sig.Generic {
					return 0, fmt.Errorf("%s: generic function symbol '%s' cannot be lowered as a callable value in this MVP", frontend.FormatPos(e.At), e.Name)
				}
				return l.emitFunctionSymbolValue(e.Name, nil, e.At), nil
			}
			if field, ok := l.actorState[e.Name]; ok {
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(field.Slot), Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_state_load", ArgSlots: 1, RetSlots: 1, Pos: e.At})
				return 1, nil
			}
			return 0, fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(e.At), e.Name)
		}
		if info.ActorField {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(info.ActorFieldSlot), Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_state_load", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		}
		for i := 0; i < info.SlotCount; i++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + i, Pos: e.At})
		}
		return info.SlotCount, nil
	case *frontend.FieldAccessExpr:
		if e.EnumType != "" {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: e.EnumOrdinal, Pos: e.At})
			info, ok := l.types[e.EnumType]
			if !ok {
				return 0, fmt.Errorf("%s: unknown enum type '%s'", frontend.FormatPos(e.At), e.EnumType)
			}
			l.emitZeroSlots(info.SlotCount-1, e.At)
			return info.SlotCount, nil
		}
		target, err := l.resolveLValue(e)
		if err != nil {
			return 0, err
		}
		if target.Global {
			if g, ok := l.globals[target.Name]; ok {
				if g.TypeName == "str" && g.HasStringLiteralInit {
					if !l.preparedStringFields[target.Name] {
						l.emitGlobalStringLiteralInitIfNeeded(g, e.At)
					}
				}
				l.emitGlobalArrayBackingsInitIfNeeded(g, e.At)
			}
			for i := 0; i < target.SlotCount; i++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: target.Base + i, Pos: e.At})
			}
			return target.SlotCount, nil
		}
		for i := 0; i < target.SlotCount; i++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: target.Base + i, Pos: e.At})
		}
		return target.SlotCount, nil
	case *frontend.IndexExpr:
		if lowered, slots, err := l.lowerScalarIndexLoad(e); lowered || err != nil {
			return slots, err
		}
		elemType, err := l.indexElemType(e.Base)
		if err != nil {
			return 0, err
		}
		baseSlots, err := l.lowerExpr(e.Base)
		if err != nil {
			return 0, err
		}
		if baseSlots != 2 {
			return 0, fmt.Errorf("%s: index base slot mismatch", frontend.FormatPos(e.At))
		}
		idxSlots, err := l.lowerExpr(e.Index)
		if err != nil {
			return 0, err
		}
		if idxSlots != 1 {
			return 0, fmt.Errorf("%s: index must be i32", frontend.FormatPos(e.At))
		}
		loadKind, ok := lowerIndexLoadKind(elemType, l.types)
		if !ok {
			return 0, lowerUnsupportedError(e.At, "unsupported index element type '%s'", elemType)
		}
		if proofID, ok := l.activeWhileProofForIndex(e); ok {
			l.emit(ir.IRInstr{Kind: uncheckedIndexLoadKind(loadKind), ProofID: proofID, Pos: e.At})
		} else {
			l.emit(ir.IRInstr{Kind: loadKind, Pos: e.At})
		}
		return 1, nil
	case *frontend.StructLitExpr:
		return l.lowerStructLiteralExpr(e, nil)
	case *frontend.TryExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			if await, awaitOK := e.X.(*frontend.AwaitExpr); awaitOK {
				call, ok = await.X.(*frontend.CallExpr)
			}
		}
		if !ok {
			return 0, fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		call = lowerCallExprWithBuiltinAlias(call)
		var dynamicFunctionValueSig *semantics.FuncSig
		if local, ok := l.locals[call.Name]; ok && local.FunctionTypeValue && local.FunctionThrowsType != "" {
			dynamicFunctionValueSig = &semantics.FuncSig{
				ReturnType: local.FunctionReturnType,
				ThrowsType: local.FunctionThrowsType,
			}
		} else if local, ok := l.locals[call.Name]; ok && local.FunctionTypeValue && local.FunctionValue != "" {
			call = lowerCallExprWithName(call, local.FunctionValue)
		} else if fieldInfo, _, ok, err := l.functionFieldCallSource(call.Name, call.At); err != nil {
			return 0, err
		} else if ok && fieldInfo.FunctionThrowsType != "" {
			dynamicFunctionValueSig = &semantics.FuncSig{
				ReturnType: fieldInfo.FunctionReturnType,
				ThrowsType: fieldInfo.FunctionThrowsType,
			}
		} else if global, ok := l.globals[call.Name]; ok && global.FunctionTypeValue && global.FunctionThrowsType != "" {
			dynamicFunctionValueSig = &semantics.FuncSig{
				ReturnType: global.FunctionReturnType,
				ThrowsType: global.FunctionThrowsType,
			}
		}
		if isTypedTaskJoinCall(call.Name) {
			return l.lowerTypedTaskJoin(call, e.At)
		}
		var sig semantics.FuncSig
		if dynamicFunctionValueSig != nil {
			sig = *dynamicFunctionValueSig
		} else {
			var ok bool
			sig, ok = l.funcs[call.Name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
			}
		}
		if sig.ThrowsType == "" {
			return 0, fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		callSuccessSlots, callErrorSlots, callCompact, err := throwingLayout(sig.ReturnType, sig.ThrowsType, l.types)
		if err != nil {
			return 0, err
		}
		expectedReturnSlots := sig.ReturnSlots
		if expectedReturnSlots == 0 {
			expectedReturnSlots = throwingReturnSlotCount(callSuccessSlots, callErrorSlots)
		}
		slots, err := l.lowerExpr(call)
		if err != nil {
			return 0, err
		}
		if slots != expectedReturnSlots {
			return 0, fmt.Errorf("%s: try result slot mismatch", frontend.FormatPos(e.At))
		}
		okLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: okLabel, Pos: e.At})

		if callCompact {
			if l.throwErrorSlots < 1 {
				return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase, Pos: e.At})
		} else {
			if callErrorSlots > l.throwErrorSlots {
				return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(e.At))
			}
			for slot := callErrorSlots - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase + slot, Pos: e.At})
			}
			for slot := 0; slot < callSuccessSlots; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase, Pos: e.At})
			}
		}

		propagatedErrorSlots := 0
		if l.throwCompact {
			var convErr error
			propagatedErrorSlots, convErr = l.emitConvertedThrowFromScratch(sig.ThrowsType, l.throwsType, e.At)
			if convErr != nil {
				return 0, convErr
			}
			if propagatedErrorSlots != 1 {
				return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(e.At))
			}
		} else {
			l.emitZeroSlots(l.throwSuccessSlots, e.At)
			var convErr error
			propagatedErrorSlots, convErr = l.emitConvertedThrowFromScratch(sig.ThrowsType, l.throwsType, e.At)
			if convErr != nil {
				return 0, convErr
			}
			if propagatedErrorSlots != l.throwErrorSlots {
				return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(e.At))
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
		l.emitCleanup(e.At)
		l.emitFunctionTempRegionReset(e.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: e.At})

		// The x64 emitter tracks stack depth linearly. This unreachable padding
		// mirrors the success-entry stack depth at okLabel.
		successEntrySlots := callSuccessSlots
		if !callCompact {
			successEntrySlots += callErrorSlots
		}
		l.emitZeroSlots(successEntrySlots, e.At)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: okLabel, Pos: e.At})

		if !callCompact {
			for slot := 0; slot < callErrorSlots; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase, Pos: e.At})
			}
		}
		return callSuccessSlots, nil
	case *frontend.AwaitExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return 0, fmt.Errorf("%s: await expects an async function call", frontend.FormatPos(e.At))
		}
		return l.lowerExpr(call)
	case *frontend.CallExpr:
		return l.lowerCallExpr(e)
	case *frontend.ClosureExpr:
		l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: l.closureSymbolName(e), Pos: e.At})
		return 1, nil
	case *frontend.UnaryExpr:
		slots, err := l.lowerExpr(e.X)
		if err != nil {
			return 0, err
		}
		if slots != 1 {
			return 0, fmt.Errorf("%s: unary operand must be i32", frontend.FormatPos(e.At))
		}
		switch e.Op {
		case frontend.TokenMinus:
			l.emit(ir.IRInstr{Kind: ir.IRNegI32, Pos: e.At})
			return 1, nil
		case frontend.TokenBang:
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
			return 1, nil
		default:
			return 0, lowerUnsupportedError(e.At, "unsupported unary operator '%s'", frontend.TokenName(e.Op))
		}
	case *frontend.BinaryExpr:
		if (e.Op == frontend.TokenEqEq || e.Op == frontend.TokenBangEq) && (isNoneExpr(e.Left) || isNoneExpr(e.Right)) {
			var value frontend.Expr
			if isNoneExpr(e.Left) {
				value = e.Right
			} else {
				value = e.Left
			}
			if err := l.lowerOptionalTag(value); err != nil {
				return 0, err
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			if e.Op == frontend.TokenEqEq {
				l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
			} else {
				l.emit(ir.IRInstr{Kind: ir.IRCmpNeI32, Pos: e.At})
			}
			return 1, nil
		}
		// Short-circuit &&
		if e.Op == frontend.TokenAmpAmp {
			resultLocal := l.allocScratchSlots(1)
			leftSlots, err := l.lowerExpr(e.Left)
			if err != nil {
				return 0, err
			}
			if leftSlots != 1 {
				return 0, fmt.Errorf("%s: && operand must be i32", frontend.FormatPos(e.At))
			}
			falseLabel := l.newLabel()
			endLabel := l.newLabel()
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: falseLabel, Pos: e.At})
			rightSlots, err := l.lowerExpr(e.Right)
			if err != nil {
				return 0, err
			}
			if rightSlots != 1 {
				return 0, fmt.Errorf("%s: && operand must be i32", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: falseLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultLocal, Pos: e.At})
			return 1, nil
		}

		// Short-circuit ||
		if e.Op == frontend.TokenPipePipe {
			resultLocal := l.allocScratchSlots(1)
			leftSlots, err := l.lowerExpr(e.Left)
			if err != nil {
				return 0, err
			}
			if leftSlots != 1 {
				return 0, fmt.Errorf("%s: || operand must be i32", frontend.FormatPos(e.At))
			}
			tryRightLabel := l.newLabel()
			endLabel := l.newLabel()
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: tryRightLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: tryRightLabel, Pos: e.At})
			rightSlots, err := l.lowerExpr(e.Right)
			if err != nil {
				return 0, err
			}
			if rightSlots != 1 {
				return 0, fmt.Errorf("%s: || operand must be i32", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultLocal, Pos: e.At})
			return 1, nil
		}

		leftSlots, err := l.lowerExpr(e.Left)
		if err != nil {
			return 0, err
		}
		rightSlots, err := l.lowerExpr(e.Right)
		if err != nil {
			return 0, err
		}
		if leftSlots != 1 || rightSlots != 1 {
			return 0, fmt.Errorf("%s: binary operands must be i32", frontend.FormatPos(e.At))
		}
		switch e.Op {
		case frontend.TokenPlus:
			l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
		case frontend.TokenMinus:
			l.emit(ir.IRInstr{Kind: ir.IRSubI32, Pos: e.At})
		case frontend.TokenStar:
			l.emit(ir.IRInstr{Kind: ir.IRMulI32, Pos: e.At})
		case frontend.TokenSlash:
			l.emit(ir.IRInstr{Kind: ir.IRDivI32, Pos: e.At})
		case frontend.TokenPercent:
			l.emit(ir.IRInstr{Kind: ir.IRModI32, Pos: e.At})
		case frontend.TokenEqEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		case frontend.TokenBangEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpNeI32, Pos: e.At})
		case frontend.TokenLess:
			l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: e.At})
		case frontend.TokenLessEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpLeI32, Pos: e.At})
		case frontend.TokenGreater:
			l.emit(ir.IRInstr{Kind: ir.IRCmpGtI32, Pos: e.At})
		case frontend.TokenGreaterEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpGeI32, Pos: e.At})
		default:
			return 0, lowerUnsupportedError(e.At, "unsupported binary operator '%s'", frontend.TokenName(e.Op))
		}
		return 1, nil
	default:
		return 0, lowerUnsupportedError(expr.Pos(), "unsupported expression kind %T", expr)
	}
}

func (l *lowerer) closureSymbolName(closure *frontend.ClosureExpr) string {
	if closure == nil || closure.Name == "" {
		return ""
	}
	if _, ok := l.funcs[closure.Name]; ok {
		return closure.Name
	}
	if l.module != "" {
		qualified := l.module + "." + closure.Name
		if _, ok := l.funcs[qualified]; ok {
			return qualified
		}
	}
	return closure.Name
}

func (l *lowerer) lowerExprAs(expr frontend.Expr, expectedType string) (int, error) {
	if expectedType == "ptr" {
		if closure, ok := expr.(*frontend.ClosureExpr); ok {
			l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: l.closureSymbolName(closure), Pos: closure.At})
			return 1, nil
		}
	}
	if expectedType == "task.i32" {
		if actualType, err := l.inferExprType(expr); err == nil && semantics.IsTypedTaskHandleTypeName(actualType) {
			return l.lowerTypedTaskPublicHandle(expr)
		}
	}
	expectedInfo, ok := l.types[expectedType]
	if !ok || expectedInfo.Kind != semantics.TypeOptional {
		return l.lowerExpr(expr)
	}
	actualType, err := l.inferExprType(expr)
	if err != nil {
		return 0, err
	}
	if actualType == expectedType {
		return l.lowerExpr(expr)
	}
	if actualType == "none" {
		l.emitZeroSlots(expectedInfo.SlotCount-1, expr.Pos())
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: expr.Pos()})
		return expectedInfo.SlotCount, nil
	}
	if !l.optionalPayloadSlotCompatible(expectedInfo.ElemType, actualType) {
		return l.lowerExpr(expr)
	}
	slots, err := l.lowerExprAs(expr, expectedInfo.ElemType)
	if err != nil {
		return 0, err
	}
	if slots != expectedInfo.SlotCount-1 {
		return 0, fmt.Errorf("%s: optional payload slot mismatch", frontend.FormatPos(expr.Pos()))
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: expr.Pos()})
	return expectedInfo.SlotCount, nil
}

func (l *lowerer) optionalPayloadSlotCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	if semantics.TypedTaskHandleTypesCompatible(expected, actual) {
		return true
	}
	if lowerInt32LikeType(expected) && lowerInt32LikeType(actual) {
		return true
	}
	if expectedInfo, ok := l.types[expected]; ok && expectedInfo.Kind == semantics.TypeOptional {
		return l.optionalPayloadSlotCompatible(expectedInfo.ElemType, actual)
	}
	return false
}

func (l *lowerer) lowerTypedTaskPublicHandle(expr frontend.Expr) (int, error) {
	slots, err := l.lowerExpr(expr)
	if err != nil {
		return 0, err
	}
	if slots == 2 {
		return slots, nil
	}
	if slots < 2 {
		return 0, fmt.Errorf("%s: typed task handle slot mismatch", frontend.FormatPos(expr.Pos()))
	}
	base := l.allocScratchSlots(slots)
	for slot := slots - 1; slot >= 0; slot-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base + slot, Pos: expr.Pos()})
	}
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: expr.Pos()})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base + slots - 1, Pos: expr.Pos()})
	return 2, nil
}

func lowerInt32LikeType(typeName string) bool {
	return lowerexpressions.Int32LikeType(typeName)
}

func gSlotCount(typeName string, types map[string]*semantics.TypeInfo) int {
	return lowerexpressions.SlotCount(typeName, types)
}
