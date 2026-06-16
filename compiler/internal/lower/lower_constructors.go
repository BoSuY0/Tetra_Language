package lower

import (
	"fmt"
	"strings"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowercallables "tetra_language/compiler/internal/lower/callables"
	lowerconstructors "tetra_language/compiler/internal/lower/constructors"
	"tetra_language/compiler/internal/semantics"
)

func lowerIndexStoreKind(elemType string, types map[string]*semantics.TypeInfo) (ir.IRInstrKind, bool) {
	return lowerconstructors.IndexStoreKind(elemType, types)
}

func (l *lowerer) inferExprType(expr frontend.Expr) (string, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", nil
	case *frontend.BoolLitExpr:
		return "bool", nil
	case *frontend.NoneLitExpr:
		return "none", nil
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.IdentExpr:
		info, ok := l.locals[e.Name]
		if !ok {
			if g, ok := l.globals[e.Name]; ok {
				return g.TypeName, nil
			}
			if field, ok := l.actorState[e.Name]; ok {
				return field.TypeName, nil
			}
			return "", fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(e.At), e.Name)
		}
		if info.ActorField {
			return info.TypeName, nil
		}
		return info.TypeName, nil
	case *frontend.FieldAccessExpr:
		if e.EnumType != "" {
			return e.EnumType, nil
		}
		_, targetType, err := semantics.ResolveFieldAccessType(e, l.locals, l.globals, l.types)
		if err != nil {
			return "", err
		}
		return targetType, nil
	case *frontend.IndexExpr:
		elem, err := l.indexElemType(e.Base)
		if err != nil {
			return "", err
		}
		return elem, nil
	case *frontend.StructLitExpr:
		return e.Type.Name, nil
	case *frontend.CallExpr:
		if typeName, _, ok := l.resolveEnumCaseConstructor(e); ok {
			return typeName, nil
		}
		if tname, ok, err := l.inferStructConstructorCallType(e); ok {
			return tname, err
		}
		if fieldInfo, _, ok, err := l.functionFieldCallSource(e.Name, e.At); err != nil {
			return "", err
		} else if ok {
			return fieldInfo.FunctionReturnType, nil
		}
		if local, ok := l.locals[e.Name]; ok && local.FunctionTypeValue {
			return local.FunctionReturnType, nil
		}
		if global, ok := l.globals[e.Name]; ok && global.FunctionTypeValue {
			return global.FunctionReturnType, nil
		}
		e = lowerCallExprWithBuiltinAlias(e)
		if e.Name == "core.recv_typed" {
			if len(e.TypeArgs) != 1 {
				return "", fmt.Errorf("%s: recv_typed expects one explicit type argument", frontend.FormatPos(e.At))
			}
			return e.TypeArgs[0].Name, nil
		}
		if e.Name == "core.send_typed" {
			return "i32", nil
		}
		if e.Name == "core.task_spawn_i32_typed" || e.Name == "core.task_spawn_group_i32_typed" {
			if len(e.TypeArgs) != 1 || e.TypeArgs[0].Name == "" {
				return "", fmt.Errorf("%s: task_spawn_i32_typed missing resolved error type", frontend.FormatPos(e.At))
			}
			return semantics.TypedTaskHandleTypeName(e.TypeArgs[0].Name, l.types), nil
		}
		if isTypedTaskJoinCall(e.Name) {
			return "i32", nil
		}
		sig, ok := l.funcs[e.Name]
		if !ok {
			return "", fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), e.Name)
		}
		return sig.ReturnType, nil
	case *frontend.ClosureExpr:
		return "ptr", nil
	case *frontend.TryExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			if await, awaitOK := e.X.(*frontend.AwaitExpr); awaitOK {
				call, ok = await.X.(*frontend.CallExpr)
			}
		}
		if !ok {
			return "", fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		call = lowerCallExprWithBuiltinAlias(call)
		sig, ok := l.funcs[call.Name]
		if !ok {
			return "", fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
		}
		return sig.ReturnType, nil
	case *frontend.CatchExpr:
		return e.ResultType, nil
	case *frontend.AwaitExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return "", fmt.Errorf("%s: await expects an async function call", frontend.FormatPos(e.At))
		}
		call = lowerCallExprWithBuiltinAlias(call)
		sig, ok := l.funcs[call.Name]
		if !ok {
			return "", fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
		}
		return sig.ReturnType, nil
	case *frontend.UnaryExpr:
		if e.Op == frontend.TokenBang {
			return "bool", nil
		}
		return "i32", nil
	case *frontend.BinaryExpr:
		return "i32", nil
	default:
		return "", lowerUnsupportedError(expr.Pos(), "unsupported expression kind %T", expr)
	}
}

func (l *lowerer) lowerStructConstructorCall(e *frontend.CallExpr, functionFields map[string]semantics.FunctionFieldInfo) (int, bool, error) {
	if len(e.Args) == 0 || len(e.ArgLabels) != len(e.Args) {
		return 0, false, nil
	}
	for _, label := range e.ArgLabels {
		if label == "" {
			return 0, false, nil
		}
	}

	info, ok := l.types[e.Name]
	if !ok || info.Kind != semantics.TypeStruct {
		return 0, false, nil
	}
	if len(e.Args) != len(info.Fields) {
		return 0, true, fmt.Errorf("%s: wrong field count for '%s'", frontend.FormatPos(e.At), e.Name)
	}

	argByLabel := make(map[string]frontend.Expr, len(e.Args))
	for i, label := range e.ArgLabels {
		if _, exists := argByLabel[label]; exists {
			return 0, true, fmt.Errorf("%s: duplicate field '%s'", frontend.FormatPos(e.Args[i].Pos()), label)
		}
		argByLabel[label] = e.Args[i]
	}
	for label, expr := range argByLabel {
		if _, ok := info.FieldMap[label]; !ok {
			return 0, true, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(expr.Pos()), label)
		}
	}

	total := 0
	for _, field := range info.Fields {
		expr, ok := argByLabel[field.Name]
		if !ok {
			return 0, true, fmt.Errorf("%s: missing field '%s'", frontend.FormatPos(e.At), field.Name)
		}
		slots := 0
		if field.FunctionTypeValue {
			if closure, ok := expr.(*frontend.ClosureExpr); ok {
				if fieldInfo, ok := functionFields[field.Name]; ok && fieldInfo.FunctionHandleValue {
					slots = l.emitCallableHandleValue(fieldInfo.FunctionValue, fieldInfo.FunctionCaptures, closure.At)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else if envLocals := l.closureEnvLocalsUnbounded(closure.Captures); len(envLocals) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else {
					slots = l.emitFunctionSymbolValue(l.closureSymbolName(closure), l.closureEnvLocals(closure.Captures), closure.At)
				}
			} else if id, ok := expr.(*frontend.IdentExpr); ok {
				if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" {
					slots = l.emitFunctionSymbolValue(source.FunctionValue, l.capturedClosureEnvLocals(source), expr.Pos())
				}
			} else if call, ok := expr.(*frontend.CallExpr); ok {
				if fieldInfo, ok := functionFields[field.Name]; ok && fieldInfo.FunctionHandleValue {
					var err error
					slots, err = l.lowerExpr(call)
					if err != nil {
						return 0, true, err
					}
					if slots < field.SlotCount {
						l.emitZeroSlots(field.SlotCount-slots, call.Pos())
						slots = field.SlotCount
					}
				}
			} else if copied, ok, err := l.emitFunctionFieldValueFromExpr(expr); err != nil {
				return 0, true, err
			} else if ok {
				slots = copied
			} else if target, ok := functionFieldTargetFromExpr(expr, l.locals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			} else if target, ok := functionTypedGlobalFieldTargetFromExpr(expr, l.globals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			} else if target, ok := importedFunctionTargetFromExpr(expr, l.imports, l.funcs); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			}
		}
		if slots == 0 {
			var err error
			if field.FunctionTypeValue {
				slots, err = l.lowerExprAs(expr, field.TypeName)
			} else {
				slots, err = l.lowerExprAs(expr, field.TypeName)
			}
			if err != nil {
				return 0, true, err
			}
		}
		if slots != field.SlotCount {
			return 0, true, fmt.Errorf("%s: slot mismatch for field '%s'", frontend.FormatPos(expr.Pos()), field.Name)
		}
		total += slots
	}
	return total, true, nil
}

func (l *lowerer) emitFunctionFieldValueFromExpr(expr frontend.Expr) (int, bool, error) {
	name := functionTypedFieldNameFromExpr(expr)
	if name == "" {
		return 0, false, nil
	}
	_, base, ok, err := l.functionFieldCallSource(name, expr.Pos())
	if err != nil || !ok {
		return 0, ok, err
	}
	for slot := 0; slot < semantics.FnPtrSlotCount; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base + slot, Pos: expr.Pos()})
	}
	return semantics.FnPtrSlotCount, true, nil
}

func (l *lowerer) lowerStructLiteralExpr(e *frontend.StructLitExpr, functionFields map[string]semantics.FunctionFieldInfo) (int, error) {
	info, ok := l.types[e.Type.Name]
	if !ok {
		return 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(e.At), e.Type.Name)
	}
	fieldMap := make(map[string]frontend.Expr, len(e.Fields))
	for _, field := range e.Fields {
		fieldMap[field.Name] = field.Value
	}
	total := 0
	for _, field := range info.Fields {
		expr, ok := fieldMap[field.Name]
		if !ok {
			return 0, fmt.Errorf("%s: missing field '%s'", frontend.FormatPos(e.At), field.Name)
		}
		slots := 0
		if field.FunctionTypeValue {
			if closure, ok := expr.(*frontend.ClosureExpr); ok {
				if fieldInfo, ok := functionFields[field.Name]; ok && fieldInfo.FunctionHandleValue {
					slots = l.emitCallableHandleValue(fieldInfo.FunctionValue, fieldInfo.FunctionCaptures, closure.At)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else if envLocals := l.closureEnvLocalsUnbounded(closure.Captures); len(envLocals) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else {
					slots = l.emitFunctionSymbolValue(l.closureSymbolName(closure), l.closureEnvLocals(closure.Captures), closure.At)
				}
			} else if id, ok := expr.(*frontend.IdentExpr); ok {
				if source, ok := l.locals[id.Name]; ok && source.FunctionTypeValue {
					for slot := 0; slot < source.SlotCount; slot++ {
						l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: source.Base + slot, Pos: expr.Pos()})
					}
					slots = source.SlotCount
				} else if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" {
					slots = l.emitFunctionSymbolValue(source.FunctionValue, l.capturedClosureEnvLocals(source), expr.Pos())
				} else if _, ok := l.funcs[id.Name]; ok {
					slots = l.emitFunctionSymbolValue(id.Name, nil, expr.Pos())
				}
			} else if call, ok := expr.(*frontend.CallExpr); ok {
				if fieldInfo, ok := functionFields[field.Name]; ok && fieldInfo.FunctionHandleValue {
					var err error
					slots, err = l.lowerExpr(call)
					if err != nil {
						return 0, err
					}
					if slots < field.SlotCount {
						l.emitZeroSlots(field.SlotCount-slots, call.Pos())
						slots = field.SlotCount
					}
				}
			} else if copied, ok, err := l.emitFunctionFieldValueFromExpr(expr); err != nil {
				return 0, err
			} else if ok {
				slots = copied
			} else if target, ok := functionFieldTargetFromExpr(expr, l.locals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			} else if target, ok := functionTypedGlobalFieldTargetFromExpr(expr, l.globals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			} else if target, ok := importedFunctionTargetFromExpr(expr, l.imports, l.funcs); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			}
		}
		if slots == 0 {
			var err error
			if field.FunctionTypeValue {
				slots, err = l.lowerExprAs(expr, field.TypeName)
			} else {
				slots, err = l.lowerExprAs(expr, field.TypeName)
			}
			if err != nil {
				return 0, err
			}
		}
		if slots != field.SlotCount {
			return 0, fmt.Errorf("%s: slot mismatch for field '%s'", frontend.FormatPos(e.At), field.Name)
		}
		total += slots
	}
	return total, nil
}

func (l *lowerer) lowerEnumCaseConstructorCall(e *frontend.CallExpr, enumPayloadFunctions map[string]semantics.FunctionFieldInfo) (int, bool, error) {
	typeName, caseInfo, ok := l.resolveEnumCaseConstructor(e)
	if !ok {
		return 0, false, nil
	}
	info, ok := l.types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		return 0, true, fmt.Errorf("%s: unknown enum type '%s'", frontend.FormatPos(e.At), typeName)
	}
	if len(e.Args) != len(caseInfo.PayloadTypes) {
		return 0, true, fmt.Errorf("%s: enum case '%s.%s' expects %d payload argument(s), got %d", frontend.FormatPos(e.At), typeName, caseInfo.Name, len(caseInfo.PayloadTypes), len(e.Args))
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: caseInfo.Ordinal, Pos: e.At})
	payloadSlots := 0
	for i, arg := range e.Args {
		slots := 0
		if i < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[i] {
			if closure, ok := arg.(*frontend.ClosureExpr); ok {
				if payloadInfo, ok := enumPayloadFunctions[enumPayloadTargetKey(caseInfo.Ordinal, i)]; ok && payloadInfo.FunctionHandleValue {
					slots = l.emitCallableHandleValue(payloadInfo.FunctionValue, payloadInfo.FunctionCaptures, closure.At)
					l.emitZeroSlots(caseInfo.PayloadSlots[i]-slots, closure.At)
					slots = caseInfo.PayloadSlots[i]
				} else if envLocals := l.closureEnvLocalsUnbounded(closure.Captures); len(envLocals) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
					l.emitZeroSlots(caseInfo.PayloadSlots[i]-slots, closure.At)
					slots = caseInfo.PayloadSlots[i]
				} else {
					slots = l.emitFunctionSymbolValue(l.closureSymbolName(closure), l.closureEnvLocals(closure.Captures), closure.At)
				}
			} else if id, ok := arg.(*frontend.IdentExpr); ok {
				if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" {
					slots = l.emitFunctionSymbolValue(source.FunctionValue, l.capturedClosureEnvLocals(source), arg.Pos())
				}
			} else if call, ok := arg.(*frontend.CallExpr); ok {
				if payloadInfo, ok := enumPayloadFunctions[enumPayloadTargetKey(caseInfo.Ordinal, i)]; ok && payloadInfo.FunctionHandleValue {
					var err error
					slots, err = l.lowerExpr(call)
					if err != nil {
						return 0, true, err
					}
					if slots < caseInfo.PayloadSlots[i] {
						l.emitZeroSlots(caseInfo.PayloadSlots[i]-slots, call.Pos())
						slots = caseInfo.PayloadSlots[i]
					}
				}
			} else if copied, ok, err := l.emitFunctionFieldValueFromExpr(arg); err != nil {
				return 0, true, err
			} else if ok {
				slots = copied
			} else if target, ok := functionFieldTargetFromExpr(arg, l.locals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, arg.Pos())
			} else if target, ok := functionTypedGlobalFieldTargetFromExpr(arg, l.globals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, arg.Pos())
			} else if target, ok := importedFunctionTargetFromExpr(arg, l.imports, l.funcs); ok {
				slots = l.emitFunctionSymbolValue(target, nil, arg.Pos())
			}
		}
		if slots == 0 {
			var err error
			if i < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[i] {
				slots, err = l.lowerExprAs(arg, caseInfo.PayloadTypes[i])
			} else {
				slots, err = l.lowerExprAs(arg, caseInfo.PayloadTypes[i])
			}
			if err != nil {
				return 0, true, err
			}
		}
		want := caseInfo.PayloadSlots[i]
		if slots != want {
			return 0, true, fmt.Errorf("%s: enum case '%s.%s' payload %d slot mismatch", frontend.FormatPos(arg.Pos()), typeName, caseInfo.Name, i+1)
		}
		payloadSlots += slots
	}
	padding := info.SlotCount - 1 - payloadSlots
	if padding < 0 {
		return 0, true, fmt.Errorf("%s: enum case '%s.%s' payload layout exceeds enum layout", frontend.FormatPos(e.At), typeName, caseInfo.Name)
	}
	l.emitZeroSlots(padding, e.At)
	return info.SlotCount, true, nil
}

func (l *lowerer) resolveEnumCaseConstructor(e *frontend.CallExpr) (string, semantics.EnumCaseInfo, bool) {
	if e.ResolvedType != "" {
		parts := strings.Split(e.Name, ".")
		if len(parts) >= 2 {
			caseName := parts[len(parts)-1]
			if info, ok := l.types[e.ResolvedType]; ok && info.Kind == semantics.TypeEnum {
				if caseInfo, ok := info.CaseMap[caseName]; ok {
					return e.ResolvedType, caseInfo, true
				}
			}
		}
	}
	parts := strings.Split(e.Name, ".")
	if len(parts) < 2 {
		return "", semantics.EnumCaseInfo{}, false
	}
	typeName := strings.Join(parts[:len(parts)-1], ".")
	caseName := parts[len(parts)-1]
	info, ok := l.types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		if altName, altInfo, found := findUniqueEnumByShortNameInLower(typeName, l.types); found {
			typeName = altName
			info = altInfo
		} else {
			return "", semantics.EnumCaseInfo{}, false
		}
	}
	caseInfo, ok := info.CaseMap[caseName]
	if !ok {
		return "", semantics.EnumCaseInfo{}, false
	}
	return typeName, caseInfo, true
}

func findUniqueEnumByShortNameInLower(shortName string, types map[string]*semantics.TypeInfo) (string, *semantics.TypeInfo, bool) {
	return lowercallables.FindUniqueEnumByShortName(shortName, types)
}

func (l *lowerer) inferStructConstructorCallType(e *frontend.CallExpr) (string, bool, error) {
	if len(e.Args) == 0 || len(e.ArgLabels) != len(e.Args) {
		return "", false, nil
	}
	for _, label := range e.ArgLabels {
		if label == "" {
			return "", false, nil
		}
	}
	info, ok := l.types[e.Name]
	if !ok || info.Kind != semantics.TypeStruct {
		return "", false, nil
	}
	return e.Name, true, nil
}
