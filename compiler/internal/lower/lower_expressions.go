package lower

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowercallables "tetra_language/compiler/internal/lower/callables"
	lowerconstructors "tetra_language/compiler/internal/lower/constructors"
	lowerexpressions "tetra_language/compiler/internal/lower/expressions"
	lowerlets "tetra_language/compiler/internal/lower/lets"
	lowerrangeproof "tetra_language/compiler/internal/lower/rangeproof"
	corerangeproof "tetra_language/compiler/internal/rangeproof"
	"tetra_language/compiler/internal/semantics"
)

// ---- lower_constructors.go ----

func lowerIndexStoreKind(
	elemType string,
	types map[string]*semantics.TypeInfo,
) (ir.IRInstrKind, bool) {
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
				return "", fmt.Errorf(
					"%s: recv_typed expects one explicit type argument",
					frontend.FormatPos(e.At),
				)
			}
			return e.TypeArgs[0].Name, nil
		}
		if e.Name == "core.send_typed" {
			return "i32", nil
		}
		if e.Name == "core.task_spawn_i32_typed" || e.Name == "core.task_spawn_group_i32_typed" {
			if len(e.TypeArgs) != 1 || e.TypeArgs[0].Name == "" {
				return "", fmt.Errorf(
					"%s: task_spawn_i32_typed missing resolved error type",
					frontend.FormatPos(e.At),
				)
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

func (l *lowerer) lowerStructConstructorCall(
	e *frontend.CallExpr,
	functionFields map[string]semantics.FunctionFieldInfo,
) (int, bool, error) {
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
		return 0, true, fmt.Errorf(
			"%s: wrong field count for '%s'",
			frontend.FormatPos(e.At),
			e.Name,
		)
	}

	argByLabel := make(map[string]frontend.Expr, len(e.Args))
	for i, label := range e.ArgLabels {
		if _, exists := argByLabel[label]; exists {
			return 0, true, fmt.Errorf(
				"%s: duplicate field '%s'",
				frontend.FormatPos(e.Args[i].Pos()),
				label,
			)
		}
		argByLabel[label] = e.Args[i]
	}
	for label, expr := range argByLabel {
		if _, ok := info.FieldMap[label]; !ok {
			return 0, true, fmt.Errorf(
				"%s: unknown field '%s'",
				frontend.FormatPos(expr.Pos()),
				label,
			)
		}
	}

	total := 0
	for _, field := range info.Fields {
		expr, ok := argByLabel[field.Name]
		if !ok {
			return 0, true, fmt.Errorf(
				"%s: missing field '%s'",
				frontend.FormatPos(e.At),
				field.Name,
			)
		}
		slots := 0
		if field.FunctionTypeValue {
			if closure, ok := expr.(*frontend.ClosureExpr); ok {
				if fieldInfo, ok := functionFields[field.Name]; ok &&
					fieldInfo.FunctionHandleValue {
					slots = l.emitCallableHandleValue(
						fieldInfo.FunctionValue,
						fieldInfo.FunctionCaptures,
						closure.At,
					)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else if envLocals := l.closureEnvLocalsUnbounded(
					closure.Captures,
				); len(
					envLocals,
				) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else {
					slots = l.emitFunctionSymbolValue(
						l.closureSymbolName(closure),
						l.closureEnvLocals(closure.Captures),
						closure.At,
					)
				}
			} else if id, ok := expr.(*frontend.IdentExpr); ok {
				if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" {
					slots = l.emitFunctionSymbolValue(
						source.FunctionValue,
						l.capturedClosureEnvLocals(source),
						expr.Pos(),
					)
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
			return 0, true, fmt.Errorf(
				"%s: slot mismatch for field '%s'",
				frontend.FormatPos(expr.Pos()),
				field.Name,
			)
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

func (l *lowerer) lowerStructLiteralExpr(
	e *frontend.StructLitExpr,
	functionFields map[string]semantics.FunctionFieldInfo,
) (int, error) {
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
				if fieldInfo, ok := functionFields[field.Name]; ok &&
					fieldInfo.FunctionHandleValue {
					slots = l.emitCallableHandleValue(
						fieldInfo.FunctionValue,
						fieldInfo.FunctionCaptures,
						closure.At,
					)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else if envLocals := l.closureEnvLocalsUnbounded(
					closure.Captures,
				); len(
					envLocals,
				) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else {
					slots = l.emitFunctionSymbolValue(
						l.closureSymbolName(closure),
						l.closureEnvLocals(closure.Captures),
						closure.At,
					)
				}
			} else if id, ok := expr.(*frontend.IdentExpr); ok {
				if source, ok := l.locals[id.Name]; ok && source.FunctionTypeValue {
					for slot := 0; slot < source.SlotCount; slot++ {
						l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: source.Base + slot, Pos: expr.Pos()})
					}
					slots = source.SlotCount
				} else if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" {
					slots = l.emitFunctionSymbolValue(
						source.FunctionValue,
						l.capturedClosureEnvLocals(source),
						expr.Pos(),
					)
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
			return 0, fmt.Errorf(
				"%s: slot mismatch for field '%s'",
				frontend.FormatPos(e.At),
				field.Name,
			)
		}
		total += slots
	}
	return total, nil
}

func (l *lowerer) lowerEnumCaseConstructorCall(
	e *frontend.CallExpr,
	enumPayloadFunctions map[string]semantics.FunctionFieldInfo,
) (int, bool, error) {
	typeName, caseInfo, ok := l.resolveEnumCaseConstructor(e)
	if !ok {
		return 0, false, nil
	}
	info, ok := l.types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		return 0, true, fmt.Errorf("%s: unknown enum type '%s'", frontend.FormatPos(e.At), typeName)
	}
	if len(e.Args) != len(caseInfo.PayloadTypes) {
		return 0, true, fmt.Errorf(
			"%s: enum case '%s.%s' expects %d payload argument(s), got %d",
			frontend.FormatPos(e.At),
			typeName,
			caseInfo.Name,
			len(caseInfo.PayloadTypes),
			len(e.Args),
		)
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: caseInfo.Ordinal, Pos: e.At})
	payloadSlots := 0
	for i, arg := range e.Args {
		slots := 0
		if i < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[i] {
			if closure, ok := arg.(*frontend.ClosureExpr); ok {
				if payloadInfo, ok := enumPayloadFunctions[enumPayloadTargetKey(caseInfo.Ordinal, i)]; ok &&
					payloadInfo.FunctionHandleValue {
					slots = l.emitCallableHandleValue(
						payloadInfo.FunctionValue,
						payloadInfo.FunctionCaptures,
						closure.At,
					)
					l.emitZeroSlots(caseInfo.PayloadSlots[i]-slots, closure.At)
					slots = caseInfo.PayloadSlots[i]
				} else if envLocals := l.closureEnvLocalsUnbounded(
					closure.Captures,
				); len(
					envLocals,
				) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
					l.emitZeroSlots(caseInfo.PayloadSlots[i]-slots, closure.At)
					slots = caseInfo.PayloadSlots[i]
				} else {
					slots = l.emitFunctionSymbolValue(
						l.closureSymbolName(closure),
						l.closureEnvLocals(closure.Captures),
						closure.At,
					)
				}
			} else if id, ok := arg.(*frontend.IdentExpr); ok {
				if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" {
					slots = l.emitFunctionSymbolValue(
						source.FunctionValue,
						l.capturedClosureEnvLocals(source),
						arg.Pos(),
					)
				}
			} else if call, ok := arg.(*frontend.CallExpr); ok {
				if payloadInfo, ok := enumPayloadFunctions[enumPayloadTargetKey(
					caseInfo.Ordinal,
					i,
				)]; ok && payloadInfo.FunctionHandleValue {
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
			return 0, true, fmt.Errorf(
				"%s: enum case '%s.%s' payload %d slot mismatch",
				frontend.FormatPos(arg.Pos()),
				typeName,
				caseInfo.Name,
				i+1,
			)
		}
		payloadSlots += slots
	}
	padding := info.SlotCount - 1 - payloadSlots
	if padding < 0 {
		return 0, true, fmt.Errorf(
			"%s: enum case '%s.%s' payload layout exceeds enum layout",
			frontend.FormatPos(e.At),
			typeName,
			caseInfo.Name,
		)
	}
	l.emitZeroSlots(padding, e.At)
	return info.SlotCount, true, nil
}

func (l *lowerer) resolveEnumCaseConstructor(
	e *frontend.CallExpr,
) (string, semantics.EnumCaseInfo, bool) {
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

func findUniqueEnumByShortNameInLower(
	shortName string,
	types map[string]*semantics.TypeInfo,
) (string, *semantics.TypeInfo, bool) {
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

// ---- lower_expr.go ----

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
					return 0, fmt.Errorf(
						"%s: generic function symbol '%s' cannot be lowered as a callable value in this MVP",
						frontend.FormatPos(e.At),
						e.Name,
					)
				}
				return l.emitFunctionSymbolValue(e.Name, nil, e.At), nil
			}
			if field, ok := l.actorState[e.Name]; ok {
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(field.Slot), Pos: e.At})
				l.emit(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "__tetra_actor_state_load",
						ArgSlots: 1,
						RetSlots: 1,
						Pos:      e.At,
					},
				)
				return 1, nil
			}
			return 0, fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(e.At), e.Name)
		}
		if info.ActorField {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(info.ActorFieldSlot), Pos: e.At})
			l.emit(
				ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "__tetra_actor_state_load",
					ArgSlots: 1,
					RetSlots: 1,
					Pos:      e.At,
				},
			)
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
		callSuccessSlots, callErrorSlots, callCompact, err := throwingLayout(
			sig.ReturnType,
			sig.ThrowsType,
			l.types,
		)
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
			propagatedErrorSlots, convErr = l.emitConvertedThrowFromScratch(
				sig.ThrowsType,
				l.throwsType,
				e.At,
			)
			if convErr != nil {
				return 0, convErr
			}
			if propagatedErrorSlots != 1 {
				return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(e.At))
			}
		} else {
			l.emitZeroSlots(l.throwSuccessSlots, e.At)
			var convErr error
			propagatedErrorSlots, convErr = l.emitConvertedThrowFromScratch(
				sig.ThrowsType,
				l.throwsType,
				e.At,
			)
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
			return 0, lowerUnsupportedError(
				e.At,
				"unsupported unary operator '%s'",
				frontend.TokenName(e.Op),
			)
		}
	case *frontend.BinaryExpr:
		if (e.Op == frontend.TokenEqEq || e.Op == frontend.TokenBangEq) && (isNoneExpr(
			e.Left,
		) || isNoneExpr(
			e.Right,
		)) {
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
			return 0, lowerUnsupportedError(
				e.At,
				"unsupported binary operator '%s'",
				frontend.TokenName(e.Op),
			)
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
			l.emit(
				ir.IRInstr{Kind: ir.IRSymAddr, Name: l.closureSymbolName(closure), Pos: closure.At},
			)
			return 1, nil
		}
	}
	if expectedType == "task.i32" {
		if actualType, err := l.inferExprType(expr); err == nil &&
			semantics.IsTypedTaskHandleTypeName(actualType) {
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

// ---- lower_expr_calls.go ----

func (l *lowerer) lowerCallExpr(e *frontend.CallExpr) (int, error) {
	if slots, ok, err := l.lowerEnumCaseConstructorCall(e, nil); ok {
		return slots, err
	}
	if slots, ok, err := l.lowerStructConstructorCall(e, nil); ok {
		return slots, err
	}
	if fieldInfo, base, ok, err := l.functionFieldCallSource(e.Name, e.At); err != nil {
		return 0, err
	} else if ok {
		return l.lowerStoredFunctionCall(e, fieldInfo, base)
	}
	if local, ok := l.locals[e.Name]; ok && local.FunctionTypeValue {
		if local.FunctionHandleValue {
			return l.lowerFunctionTypedParamCall(e, local)
		}
		if local.FunctionValue != "" && !local.Mutable {
			return l.lowerStoredFunctionCall(e, semantics.FunctionFieldInfo{
				FunctionValue:          local.FunctionValue,
				FunctionParamTypes:     append([]string(nil), local.FunctionParamTypes...),
				FunctionParamOwnership: append([]string(nil), local.FunctionParamOwnership...),
				FunctionReturnType:     local.FunctionReturnType,
				FunctionThrowsType:     local.FunctionThrowsType,
			}, local.Base)
		}
		return l.lowerFunctionTypedParamCall(e, local)
	}
	if global, ok := l.globals[e.Name]; ok && global.FunctionTypeValue {
		l.emitGlobalFunctionValueInitIfNeeded(global, e.At)
		return l.lowerGlobalStoredFunctionCall(e, global)
	}
	e = lowerCallExprWithBuiltinAlias(e)
	if slots, ok, err := l.lowerRawOffsetCall(e); ok {
		return slots, err
	}
	if slots, ok, err := l.lowerPtrAddValueCall(e); ok {
		return slots, err
	}
	if slots, ok, err := l.lowerAtomicBuiltinCall(e); ok {
		return slots, err
	}
	switch e.Name {
	case "core.surface_open":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_open", 4)
	case "core.surface_close":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_close", 1)
	case "core.surface_poll_event_kind":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_kind", 1)
	case "core.surface_poll_event_x":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_x", 1)
	case "core.surface_poll_event_y":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_y", 1)
	case "core.surface_poll_event_button":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_button", 1)
	case "core.surface_poll_event_into":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_into", 3)
	case "core.surface_poll_event_text_len":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_text_len", 1)
	case "core.surface_poll_event_text_into":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_event_text_into", 3)
	case "core.surface_clipboard_write_text":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_clipboard_write_text", 3)
	case "core.surface_clipboard_read_text_into":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_clipboard_read_text_into", 3)
	case "core.surface_poll_composition_into":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_poll_composition_into", 3)
	case "core.surface_begin_frame":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_begin_frame", 1)
	case "core.surface_present_rgba":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_present_rgba", 6)
	case "core.surface_now_ms":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_now_ms", 0)
	case "core.surface_request_redraw":
		return l.lowerSurfaceRuntimeCall(e, "__tetra_surface_request_redraw", 1)
	case "core.spawn":
		if len(e.Args) != 1 {
			return 0, fmt.Errorf("%s: spawn expects 1 argument", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf("%s: spawn expects a string literal", frontend.FormatPos(e.At))
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf("%s: spawn expects a non-empty name", frontend.FormatPos(e.At))
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(name))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_spawn",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.spawn_remote":
		if len(e.Args) != 2 {
			return 0, fmt.Errorf("%s: spawn_remote expects 2 arguments", frontend.FormatPos(e.At))
		}
		nodeSlots, err := l.lowerExpr(e.Args[0])
		if err != nil {
			return 0, err
		}
		if nodeSlots != 1 {
			return 0, fmt.Errorf(
				"%s: spawn_remote expects a 1-slot node id",
				frontend.FormatPos(e.Args[0].Pos()),
			)
		}
		lit, ok := e.Args[1].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf(
				"%s: spawn_remote expects a string literal",
				frontend.FormatPos(e.At),
			)
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf(
				"%s: spawn_remote expects a non-empty name",
				frontend.FormatPos(e.At),
			)
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(name))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_spawn_remote",
				ArgSlots: 2,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.task_spawn_i32":
		if len(e.Args) != 1 {
			return 0, fmt.Errorf("%s: task_spawn_i32 expects 1 argument", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf(
				"%s: task_spawn_i32 expects a string literal",
				frontend.FormatPos(e.At),
			)
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf(
				"%s: task_spawn_i32 expects a non-empty name",
				frontend.FormatPos(e.At),
			)
		}
		sig, ok := l.funcs[name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
		}
		if sig.ReturnSlots != 1 {
			return 0, fmt.Errorf(
				"%s: task_spawn_i32 target must return 1 slot",
				frontend.FormatPos(e.At),
			)
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(name))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_spawn_i32",
				ArgSlots: 1,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		return 2, nil
	case "core.task_spawn_i32_typed":
		if len(e.TypeArgs) != 1 {
			return 0, fmt.Errorf(
				"%s: task_spawn_i32_typed expects one explicit error type argument",
				frontend.FormatPos(e.At),
			)
		}
		errorType := e.TypeArgs[0].Name
		if errorType == "" {
			return 0, fmt.Errorf(
				"%s: task_spawn_i32_typed missing resolved error type",
				frontend.FormatPos(e.At),
			)
		}
		_, handleInfo, err := semantics.EnsureTypedTaskHandleType(errorType, l.types)
		if err != nil {
			return 0, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
		}
		if len(e.Args) != 1 {
			return 0, fmt.Errorf(
				"%s: task_spawn_i32_typed expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf(
				"%s: task_spawn_i32_typed expects a string literal",
				frontend.FormatPos(e.At),
			)
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf(
				"%s: task_spawn_i32_typed expects a non-empty name",
				frontend.FormatPos(e.At),
			)
		}
		sig, ok := l.funcs[name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
		}
		if handleInfo.SlotCount <= 4 {
			if sig.ReturnSlots != handleInfo.SlotCount {
				return 0, fmt.Errorf(
					"%s: task_spawn_i32_typed target return slot mismatch",
					frontend.FormatPos(e.At),
				)
			}
		} else if sig.ReturnType != "i32" {
			return 0, fmt.Errorf(
				"%s: task_spawn_i32_typed staged mode requires target return type i32",
				frontend.FormatPos(e.At),
			)
		}
		wrapperName := typedTaskWrapperName(name, errorType)
		h := fnv.New32a()
		_, _ = h.Write([]byte(wrapperName))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_spawn_i32",
				ArgSlots: 1,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		if handleInfo.SlotCount > 2 {
			statusLocal := l.allocScratchSlots(1)
			handleLocal := l.allocScratchSlots(1)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: handleLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: handleLocal, Pos: e.At})
			l.emitZeroSlots(handleInfo.SlotCount-2, e.At)
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: e.At})
		}
		return handleInfo.SlotCount, nil
	case "core.task_spawn_group_i32_typed":
		if len(e.TypeArgs) != 1 {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32_typed expects one explicit error type argument",
				frontend.FormatPos(e.At),
			)
		}
		errorType := e.TypeArgs[0].Name
		if errorType == "" {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32_typed missing resolved error type",
				frontend.FormatPos(e.At),
			)
		}
		_, handleInfo, err := semantics.EnsureTypedTaskHandleType(errorType, l.types)
		if err != nil {
			return 0, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
		}
		if len(e.Args) != 2 {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32_typed expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		groupSlots, err := l.lowerExpr(e.Args[0])
		if err != nil {
			return 0, err
		}
		if groupSlots != 1 {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32_typed expects a 1-slot task.group handle",
				frontend.FormatPos(e.At),
			)
		}
		groupLocal := l.allocScratchSlots(1)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: groupLocal, Pos: e.At})
		lit, ok := e.Args[1].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32_typed expects a string literal worker name",
				frontend.FormatPos(e.At),
			)
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32_typed expects a non-empty name",
				frontend.FormatPos(e.At),
			)
		}
		sig, ok := l.funcs[name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
		}
		if handleInfo.SlotCount <= 4 {
			if sig.ReturnSlots != handleInfo.SlotCount {
				return 0, fmt.Errorf(
					"%s: task_spawn_group_i32_typed target return slot mismatch",
					frontend.FormatPos(e.At),
				)
			}
		} else if sig.ReturnType != "i32" {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32_typed staged mode requires target return type i32",
				frontend.FormatPos(e.At),
			)
		}

		activeLabel := l.newLabel()
		endLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: activeLabel, Pos: e.At})
		l.emitZeroSlots(handleInfo.SlotCount-1, e.At)
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: activeLabel, Pos: e.At})
		wrapperName := typedTaskWrapperName(name, errorType)
		h := fnv.New32a()
		_, _ = h.Write([]byte(wrapperName))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_spawn_group_i32",
				ArgSlots: 2,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		if handleInfo.SlotCount > 2 {
			statusLocal := l.allocScratchSlots(1)
			handleLocal := l.allocScratchSlots(1)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: handleLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: handleLocal, Pos: e.At})
			l.emitZeroSlots(handleInfo.SlotCount-2, e.At)
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
		return handleInfo.SlotCount, nil
	case "core.task_spawn_group_i32":
		if len(e.Args) != 2 {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32 expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		groupSlots, err := l.lowerExpr(e.Args[0])
		if err != nil {
			return 0, err
		}
		if groupSlots != 1 {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32 expects a 1-slot task.group handle",
				frontend.FormatPos(e.At),
			)
		}
		groupLocal := l.allocScratchSlots(1)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: groupLocal, Pos: e.At})
		lit, ok := e.Args[1].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32 expects a string literal worker name",
				frontend.FormatPos(e.At),
			)
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32 expects a non-empty name",
				frontend.FormatPos(e.At),
			)
		}
		sig, ok := l.funcs[name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
		}
		if sig.ReturnSlots != 1 {
			return 0, fmt.Errorf(
				"%s: task_spawn_group_i32 target must return 1 slot",
				frontend.FormatPos(e.At),
			)
		}

		activeLabel := l.newLabel()
		endLabel := l.newLabel()
		// group == 0 => canceled handle
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: activeLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: activeLabel, Pos: e.At})
		h := fnv.New32a()
		_, _ = h.Write([]byte(name))
		id := int32(h.Sum32())
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_spawn_group_i32",
				ArgSlots: 2,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
		return 2, nil
	case "core.recv":
		if len(e.Args) != 0 {
			return 0, fmt.Errorf("%s: recv expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_recv",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.recv_msg":
		if len(e.Args) != 0 {
			return 0, fmt.Errorf("%s: recv_msg expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_recv_msg",
				ArgSlots: 0,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		return 2, nil
	case "core.recv_typed":
		if len(e.Args) != 0 {
			return 0, fmt.Errorf("%s: recv_typed expects 0 arguments", frontend.FormatPos(e.At))
		}
		if len(e.TypeArgs) != 1 {
			return 0, fmt.Errorf(
				"%s: recv_typed expects one explicit type argument",
				frontend.FormatPos(e.At),
			)
		}
		msgType := e.TypeArgs[0].Name
		info, ok := l.types[msgType]
		if !ok || info.Kind != semantics.TypeEnum {
			return 0, fmt.Errorf(
				"%s: recv_typed expects an enum type argument",
				frontend.FormatPos(e.At),
			)
		}
		base := l.allocScratchSlots(info.SlotCount)
		tagBase := typedActorMessageTagBase(msgType)
		nonNegativeLabel := l.newLabel()
		mismatchLabel := l.newLabel()
		endLabel := l.newLabel()
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_recv_begin",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: tagBase, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRSubI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base, Pos: e.At})

		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nonNegativeLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: mismatchLabel, Pos: e.At})

		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nonNegativeLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(len(info.EnumCases)), Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: mismatchLabel, Pos: e.At})
		for slot := 0; slot < info.SlotCount-1; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot), Pos: e.At})
			l.emit(
				ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "__tetra_actor_recv_slot",
					ArgSlots: 1,
					RetSlots: 1,
					Pos:      e.At,
				},
			)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base + 1 + slot, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: mismatchLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base, Pos: e.At})
		for slot := 0; slot < info.SlotCount-1; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: -1, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base + 1 + slot, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
		for slot := 0; slot < info.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base + slot, Pos: e.At})
		}
		return info.SlotCount, nil
	case "core.send_typed":
		if len(e.Args) != 2 {
			return 0, fmt.Errorf("%s: send_typed expects 2 arguments", frontend.FormatPos(e.At))
		}
		targetSlots, err := l.lowerExpr(e.Args[0])
		if err != nil {
			return 0, err
		}
		if targetSlots != 1 {
			return 0, fmt.Errorf(
				"%s: send_typed expects actor target",
				frontend.FormatPos(e.Args[0].Pos()),
			)
		}
		targetLocal := l.allocScratchSlots(1)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: targetLocal, Pos: e.At})
		msgType, err := l.inferExprType(e.Args[1])
		if err != nil {
			return 0, err
		}
		info, ok := l.types[msgType]
		if !ok || info.Kind != semantics.TypeEnum {
			return 0, fmt.Errorf(
				"%s: send_typed expects an enum message",
				frontend.FormatPos(e.Args[1].Pos()),
			)
		}
		msgBase := l.allocScratchSlots(info.SlotCount)
		msgSlots, err := l.lowerExpr(e.Args[1])
		if err != nil {
			return 0, err
		}
		if msgSlots != info.SlotCount {
			return 0, fmt.Errorf(
				"%s: send_typed message slot mismatch",
				frontend.FormatPos(e.Args[1].Pos()),
			)
		}
		for slot := info.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: msgBase + slot, Pos: e.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: targetLocal, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: msgBase, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: typedActorMessageTagBase(msgType), Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(info.SlotCount - 1), Pos: e.At})
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_send_begin",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		beginResult := l.allocScratchSlots(1)
		beginFailedLabel := l.newLabel()
		endLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: beginResult, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: beginResult, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: beginFailedLabel, Pos: e.At})
		discard := l.ensureDiscardLocal()
		for slot := 0; slot < info.SlotCount-1; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot), Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: msgBase + 1 + slot, Pos: e.At})
			l.emit(
				ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "__tetra_actor_send_slot",
					ArgSlots: 2,
					RetSlots: 1,
					Pos:      e.At,
				},
			)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_send_commit",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: beginFailedLabel, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: beginResult, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
		return 1, nil
	case "core.self":
		if len(e.Args) != 0 {
			return 0, fmt.Errorf("%s: self expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_self",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.sender":
		if len(e.Args) != 0 {
			return 0, fmt.Errorf("%s: sender expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_sender",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.sym_addr":
		if len(e.Args) != 1 {
			return 0, fmt.Errorf("%s: sym_addr expects 1 argument", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return 0, fmt.Errorf("%s: sym_addr expects a string literal", frontend.FormatPos(e.At))
		}
		name := string(lit.Value)
		if name == "" {
			return 0, fmt.Errorf(
				"%s: sym_addr expects a non-empty symbol name",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: name, Pos: e.At})
		return 1, nil
	}
	total := 0
	callSig, hasCallSig := l.funcs[e.Name]
	for i, arg := range e.Args {
		var slots int
		var err error
		if hasCallSig && i < len(callSig.ParamFunctionTypes) && callSig.ParamFunctionTypes[i] {
			slots, err = l.lowerFunctionTypedArgument(arg)
		} else if hasCallSig && i < len(callSig.ParamTypes) {
			slots, err = l.lowerExprAs(arg, callSig.ParamTypes[i])
		} else {
			slots, err = l.lowerExpr(arg)
		}
		if err != nil {
			return 0, err
		}
		total += slots
	}
	if hasCallSig {
		l.invalidateWhileRangeProofsForInoutArgs(e.Args, callSig.ParamOwnership)
	}
	switch e.Name {
	case "core.cap_io":
		if total != 0 {
			return 0, fmt.Errorf("%s: cap_io expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCapIO, Pos: e.At})
		return 1, nil
	case "core.cap_mem":
		if total != 0 {
			return 0, fmt.Errorf("%s: cap_mem expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCapMem, Pos: e.At})
		return 1, nil
	case "core.alloc_bytes":
		if total != 1 {
			return 0, fmt.Errorf("%s: alloc_bytes expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRAllocBytes, Pos: e.At})
		return 1, nil
	case "core.make_u8":
		if total != 1 {
			return 0, fmt.Errorf("%s: make_u8 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMakeSliceU8, Pos: e.At})
		return 2, nil
	case "core.make_u16":
		if total != 1 {
			return 0, fmt.Errorf("%s: make_u16 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMakeSliceU16, Pos: e.At})
		return 2, nil
	case "core.make_i32":
		if total != 1 {
			return 0, fmt.Errorf("%s: make_i32 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMakeSliceI32, Pos: e.At})
		return 2, nil
	case "core.make_bool":
		if total != 1 {
			return 0, fmt.Errorf("%s: make_bool expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMakeSliceI32, Pos: e.At})
		return 2, nil
	case "core.raw_slice_u8_from_parts",
		"core.raw_slice_u16_from_parts",
		"core.raw_slice_i32_from_parts",
		"core.raw_slice_bool_from_parts":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: %s expects ptr, length, and cap.mem arguments",
				frontend.FormatPos(e.At),
				e.Name,
			)
		}
		l.emit(
			ir.IRInstr{Kind: ir.IRRawSliceFromParts, Imm: rawSliceElementShift(e.Name), Pos: e.At},
		)
		return 2, nil
	case "core.slice_borrow_u8",
		"core.slice_borrow_u16",
		"core.slice_borrow_i32",
		"core.slice_borrow_bool",
		"core.string_borrow":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: %s expects one view source argument",
				frontend.FormatPos(e.At),
				e.Name,
			)
		}
		return 2, nil
	case "core.slice_copy_u8",
		"core.slice_copy_u16",
		"core.slice_copy_i32",
		"core.slice_copy_bool",
		"core.string_copy":
		return l.lowerCopyBuiltinFromStack(e.Name, total, e.At)
	case "core.slice_copy_into_u8",
		"core.slice_copy_into_u16",
		"core.slice_copy_into_i32",
		"core.slice_copy_into_bool",
		"core.string_copy_into":
		return l.lowerCopyIntoBuiltinFromStack(e.Name, total, e.At)
	case "core.slice_window_u8",
		"core.slice_window_u16",
		"core.slice_window_i32",
		"core.slice_window_bool",
		"core.string_window":
		if total != 4 {
			return 0, fmt.Errorf(
				"%s: %s expects view source, start, and count arguments",
				frontend.FormatPos(e.At),
				e.Name,
			)
		}
		shift, ok := sliceViewElementShift(e.Name)
		if !ok {
			return 0, lowerUnsupportedError(e.At, "unsupported view window builtin '%s'", e.Name)
		}
		l.emit(ir.IRInstr{Kind: ir.IRSliceWindow, Imm: shift, Pos: e.At})
		return 2, nil
	case "core.slice_prefix_u8",
		"core.slice_prefix_u16",
		"core.slice_prefix_i32",
		"core.slice_prefix_bool",
		"core.string_prefix":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: %s expects view source and count arguments",
				frontend.FormatPos(e.At),
				e.Name,
			)
		}
		shift, ok := sliceViewElementShift(e.Name)
		if !ok {
			return 0, lowerUnsupportedError(e.At, "unsupported view prefix builtin '%s'", e.Name)
		}
		l.emit(ir.IRInstr{Kind: ir.IRSlicePrefix, Imm: shift, Pos: e.At})
		return 2, nil
	case "core.slice_suffix_u8",
		"core.slice_suffix_u16",
		"core.slice_suffix_i32",
		"core.slice_suffix_bool",
		"core.string_suffix":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: %s expects view source and start argument",
				frontend.FormatPos(e.At),
				e.Name,
			)
		}
		shift, ok := sliceViewElementShift(e.Name)
		if !ok {
			return 0, lowerUnsupportedError(e.At, "unsupported view suffix builtin '%s'", e.Name)
		}
		l.emit(ir.IRInstr{Kind: ir.IRSliceSuffix, Imm: shift, Pos: e.At})
		return 2, nil
	case "core.island_new":
		if total != 1 {
			return 0, fmt.Errorf("%s: island_new expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandNew, Pos: e.At})
		return 1, nil
	case "core.island_make_u8":
		if total != 2 {
			return 0, fmt.Errorf("%s: island_make_u8 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind: ir.IRIslandMakeSliceU8,
				Name: l.allocationNameForBuiltinCall(e.Name, e.At, allocplan.StorageExplicitIsland),
				Pos:  e.At,
			},
		)
		return 2, nil
	case "core.island_make_u16":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: island_make_u16 expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind: ir.IRIslandMakeSliceU16,
				Name: l.allocationNameForBuiltinCall(e.Name, e.At, allocplan.StorageExplicitIsland),
				Pos:  e.At,
			},
		)
		return 2, nil
	case "core.island_make_i32":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: island_make_i32 expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind: ir.IRIslandMakeSliceI32,
				Name: l.allocationNameForBuiltinCall(e.Name, e.At, allocplan.StorageExplicitIsland),
				Pos:  e.At,
			},
		)
		return 2, nil
	case "core.island_make_bool":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: island_make_bool expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind: ir.IRIslandMakeSliceI32,
				Name: l.allocationNameForBuiltinCall(e.Name, e.At, allocplan.StorageExplicitIsland),
				Pos:  e.At,
			},
		)
		return 2, nil
	case "core.island_reset":
		if total != 1 {
			return 0, fmt.Errorf("%s: island_reset expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandReset, Pos: e.At})
		return 1, nil
	case "core.mmio_read_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: mmio_read_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMmioReadI32, Pos: e.At})
		return 1, nil
	case "core.mmio_write_i32":
		if total != 3 {
			return 0, fmt.Errorf("%s: mmio_write_i32 expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMmioWriteI32, Pos: e.At})
		return 1, nil
	case "core.fs_exists":
		if total != 3 {
			return 0, fmt.Errorf("%s: fs_exists expects 3 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_fs_exists",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_socket_tcp4":
		if total != 1 {
			return 0, fmt.Errorf(
				"%s: net_socket_tcp4 expects 1 argument slot",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_socket_tcp4",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_bind_tcp4_loopback":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: net_bind_tcp4_loopback expects 3 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_bind_tcp4_loopback",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_connect_tcp4_loopback":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: net_connect_tcp4_loopback expects 3 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_connect_tcp4_loopback",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_listen":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: net_listen expects 3 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_listen",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_accept4":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: net_accept4 expects 3 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_accept4",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_read":
		if total != 6 {
			return 0, fmt.Errorf("%s: net_read expects 6 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_read",
				ArgSlots: 6,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_recv":
		if total != 6 {
			return 0, fmt.Errorf("%s: net_recv expects 6 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_recv",
				ArgSlots: 6,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_write":
		if total != 6 {
			return 0, fmt.Errorf("%s: net_write expects 6 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_write",
				ArgSlots: 6,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_send":
		if total != 6 {
			return 0, fmt.Errorf("%s: net_send expects 6 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_send",
				ArgSlots: 6,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_epoll_create":
		if total != 1 {
			return 0, fmt.Errorf(
				"%s: net_epoll_create expects 1 argument slot",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_epoll_create",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_epoll_ctl_add_read":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: net_epoll_ctl_add_read expects 3 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_epoll_ctl_add_read",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_epoll_ctl_add_read_write":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: net_epoll_ctl_add_read_write expects 3 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_epoll_ctl_add_read_write",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_epoll_ctl_mod_read":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: net_epoll_ctl_mod_read expects 3 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_epoll_ctl_mod_read",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_epoll_ctl_mod_read_write":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: net_epoll_ctl_mod_read_write expects 3 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_epoll_ctl_mod_read_write",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_epoll_ctl_delete":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: net_epoll_ctl_delete expects 3 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_epoll_ctl_delete",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_epoll_wait_one":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: net_epoll_wait_one expects 3 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_epoll_wait_one",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_epoll_wait_one_into":
		if total != 5 {
			return 0, fmt.Errorf(
				"%s: net_epoll_wait_one_into expects 5 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_epoll_wait_one_into",
				ArgSlots: 5,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_set_nonblocking":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: net_set_nonblocking expects 2 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_set_nonblocking",
				ArgSlots: 2,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_set_reuseport":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: net_set_reuseport expects 2 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_set_reuseport",
				ArgSlots: 2,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_set_tcp_nodelay":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: net_set_tcp_nodelay expects 2 argument slots",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_set_tcp_nodelay",
				ArgSlots: 2,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.net_close":
		if total != 2 {
			return 0, fmt.Errorf("%s: net_close expects 2 argument slots", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_net_close",
				ArgSlots: 2,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.load_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: load_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemReadI32, Pos: e.At})
		return 1, nil
	case "core.store_i32":
		if total != 3 {
			return 0, fmt.Errorf("%s: store_i32 expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemWriteI32, Pos: e.At})
		return 1, nil
	case "core.load_u8":
		if total != 2 {
			return 0, fmt.Errorf("%s: load_u8 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemReadU8, Pos: e.At})
		return 1, nil
	case "core.store_u8":
		if total != 3 {
			return 0, fmt.Errorf("%s: store_u8 expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemWriteU8, Pos: e.At})
		return 1, nil
	case "core.load_ptr":
		if total != 2 {
			return 0, fmt.Errorf("%s: load_ptr expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemReadPtr, Pos: e.At})
		return 1, nil
	case "core.store_ptr":
		if total != 3 {
			return 0, fmt.Errorf("%s: store_ptr expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemWritePtr, Pos: e.At})
		return 1, nil
	case "core.store_arch_ptr":
		if total != 3 {
			return 0, fmt.Errorf("%s: store_arch_ptr expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRMemWriteArchPtr, Pos: e.At})
		return 1, nil
	case "core.ptr_add":
		if total != 3 {
			return 0, fmt.Errorf("%s: ptr_add expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRPtrAdd, Pos: e.At})
		return 1, nil
	case "core.ctx_switch":
		if total != 3 {
			return 0, fmt.Errorf("%s: ctx_switch expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCtxSwitch, Pos: e.At})
		return 1, nil
	case "core.consent_token":
		if total != 0 {
			return 0, fmt.Errorf("%s: consent_token expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: consentTokenRuntimeSentinel, Pos: e.At})
		return 1, nil
	case "core.secret_seal_i32":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: secret_seal_i32 expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		// Keep the first argument (secret payload) and consume the token.
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRMulI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
		return 1, nil
	case "core.secret_unseal_i32":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: secret_unseal_i32 expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		// Keep the first argument (sealed payload) and consume the token.
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRMulI32, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
		return 1, nil
	case "core.task_group_open":
		if total != 0 {
			return 0, fmt.Errorf(
				"%s: task_group_open expects 0 arguments",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_group_open",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.time_now_ms":
		if total != 0 {
			return 0, fmt.Errorf("%s: time_now_ms expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_time_now_ms",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.sleep_ms":
		if total != 1 {
			return 0, fmt.Errorf("%s: sleep_ms expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_sleep_ms",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.sleep_until":
		if total != 1 {
			return 0, fmt.Errorf("%s: sleep_until expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_sleep_until_ms",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.deadline_ms":
		if total != 1 {
			return 0, fmt.Errorf("%s: deadline_ms expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_deadline_ms",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.timer_ready":
		if total != 1 {
			return 0, fmt.Errorf("%s: timer_ready expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_timer_ready_ms",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.yield":
		if total != 0 {
			return 0, fmt.Errorf("%s: yield expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_yield_now",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.task_group_close":
		if total != 1 {
			return 0, fmt.Errorf(
				"%s: task_group_close expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_group_close",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.task_group_cancel":
		if total != 1 {
			return 0, fmt.Errorf(
				"%s: task_group_cancel expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_group_cancel",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.task_group_current":
		if total != 0 {
			return 0, fmt.Errorf(
				"%s: task_group_current expects 0 arguments",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_group_current",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.task_group_status":
		if total != 1 {
			return 0, fmt.Errorf(
				"%s: task_group_status expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_group_status",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.task_is_canceled":
		if total != 0 {
			return 0, fmt.Errorf(
				"%s: task_is_canceled expects 0 arguments",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_is_canceled",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.task_checkpoint":
		if total != 0 {
			return 0, fmt.Errorf(
				"%s: task_checkpoint expects 0 arguments",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_checkpoint",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.task_join_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: task_join_i32 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_join_i32",
				ArgSlots: 2,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.task_join_i32_typed", "core.task_join_group_i32_typed":
		return 0, fmt.Errorf("%s: task_join_i32_typed requires try", frontend.FormatPos(e.At))
	case "core.task_join_result_i32":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: task_join_result_i32 expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_join_result_i32",
				ArgSlots: 2,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		return 2, nil
	case "core.task_join_until_i32":
		if total != 3 {
			return 0, fmt.Errorf(
				"%s: task_join_until_i32 expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_join_until_i32",
				ArgSlots: 3,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		return 2, nil
	case "core.task_poll_i32":
		if total != 2 {
			return 0, fmt.Errorf("%s: task_poll_i32 expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_poll_i32",
				ArgSlots: 2,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		return 2, nil
	case "core.select2_i32":
		if total != 3 {
			return 0, fmt.Errorf("%s: select2_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_join_until_i32",
				ArgSlots: 3,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		return 2, nil
	case "core.actor_dispatch":
		if total != 1 {
			return 0, fmt.Errorf("%s: actor_dispatch expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_dispatch",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.actor_main_entry_id":
		if total != 0 {
			return 0, fmt.Errorf(
				"%s: actor_main_entry_id expects 0 arguments",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_main_entry_id",
				ArgSlots: 0,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.actor_node_connect":
		if total != 2 {
			return 0, fmt.Errorf(
				"%s: actor_node_connect expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_node_connect",
				ArgSlots: 2,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.actor_node_status":
		if total != 1 {
			return 0, fmt.Errorf(
				"%s: actor_node_status expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_node_status",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.send":
		if total != 2 {
			return 0, fmt.Errorf("%s: send expects 2 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_send",
				ArgSlots: 2,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.send_msg":
		if total != 3 {
			return 0, fmt.Errorf("%s: send_msg expects 3 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_send_msg",
				ArgSlots: 3,
				RetSlots: 1,
				Pos:      e.At,
			},
		)
		return 1, nil
	case "core.recv_poll":
		if total != 0 {
			return 0, fmt.Errorf("%s: recv_poll expects 0 arguments", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_recv_poll",
				ArgSlots: 0,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		return 2, nil
	case "core.recv_until":
		if total != 1 {
			return 0, fmt.Errorf("%s: recv_until expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_recv_until",
				ArgSlots: 1,
				RetSlots: 2,
				Pos:      e.At,
			},
		)
		return 2, nil
	case "core.recv_msg_until":
		if total != 1 {
			return 0, fmt.Errorf("%s: recv_msg_until expects 1 argument", frontend.FormatPos(e.At))
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_actor_recv_msg_until",
				ArgSlots: 1,
				RetSlots: 3,
				Pos:      e.At,
			},
		)
		return 3, nil
	default:
		sig, ok := l.funcs[e.Name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), e.Name)
		}
		writebacks := []inoutWriteback(nil)
		if sig.ThrowsType == "" {
			var err error
			writebacks, err = l.collectInoutWritebacks(e.Args, sig.ParamOwnership)
			if err != nil {
				return 0, err
			}
		}
		abiReturnSlots := sig.ReturnSlots + inoutWritebackSlotCount(writebacks)
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     e.Name,
				ArgSlots: total,
				RetSlots: abiReturnSlots,
				Pos:      e.At,
			},
		)
		l.emitInoutWritebacks(writebacks, e.At)
		return sig.ReturnSlots, nil
	}
}

// ---- lower_lets_match.go ----

func (l *lowerer) ensureDiscardLocal() int {
	if l.discardLocal >= 0 {
		return l.discardLocal
	}
	l.discardLocal = l.localSlots
	l.localSlots++
	return l.discardLocal
}

func (l *lowerer) allocScratchSlots(slots int) int {
	base := l.localSlots
	l.localSlots += slots
	return base
}

func (l *lowerer) lowerUnusedCopyLet(
	name string,
	info semantics.LocalInfo,
	expr frontend.Expr,
	pos frontend.Position,
) (bool, int, error) {
	if !l.stackAllocationLowering || info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 1 {
		return false, 0, nil
	}
	if _, ok := freshCopyBuiltinElement(call.Name); !ok {
		return false, 0, nil
	}
	alloc, ok := l.allocationPlan[name]
	if !ok || alloc.ActualLoweringStorage != allocplan.StorageEliminated ||
		alloc.LoweringStatus != "eliminated_unused_copy" {
		return false, 0, nil
	}
	sourceSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return false, 0, err
	}
	if sourceSlots != 2 {
		return false, 0, fmt.Errorf(
			"%s: %s expects one view source argument",
			frontend.FormatPos(pos),
			call.Name,
		)
	}
	srcLen := l.allocScratchSlots(1)
	srcPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	return true, 2, nil
}

func (l *lowerer) lowerScalarReplacementLet(
	name string,
	info semantics.LocalInfo,
	expr frontend.Expr,
	pos frontend.Position,
) (bool, int, error) {
	if !l.stackAllocationLowering || info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 1 {
		return false, 0, nil
	}
	elem, isMake := stackAllocationElementByBuiltin(call.Name)
	if !isMake {
		var ok bool
		elem, ok = freshCopyBuiltinElement(call.Name)
		if !ok {
			return false, 0, nil
		}
	}
	isCopy := !isMake
	alloc, ok := l.allocationPlan[name]
	if !ok || alloc.ActualLoweringStorage != allocplan.StorageEliminated ||
		alloc.LoweringStatus != "scalar_replacement" {
		return false, 0, nil
	}
	if alloc.ElementSize <= 0 || alloc.ByteSize <= 0 || alloc.ByteSize%alloc.ElementSize != 0 {
		return false, 0, nil
	}
	length := int64(alloc.ByteSize / alloc.ElementSize)
	if isMake {
		var known bool
		length, known = evalConstInt64ForAllocation(call.Args[0])
		if !known {
			return false, 0, nil
		}
	}
	if length <= 0 || length > int64(alloc.ByteSize) {
		return false, 0, nil
	}
	if alloc.ElementSize <= 0 || int(length)*alloc.ElementSize != alloc.ByteSize {
		return false, 0, nil
	}
	loadKind, ok := lowerIndexLoadKind(elem, l.types)
	if !ok {
		return false, 0, nil
	}
	srcPtr, srcLen := -1, -1
	if isCopy {
		sourceSlots, err := l.lowerExpr(call.Args[0])
		if err != nil {
			return false, 0, err
		}
		if sourceSlots != 2 {
			return false, 0, fmt.Errorf(
				"%s: %s expects one view source argument",
				frontend.FormatPos(pos),
				call.Name,
			)
		}
		srcLen = l.allocScratchSlots(1)
		srcPtr = l.allocScratchSlots(1)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})
	}
	elementBase := l.allocScratchSlots(int(length))
	for i := int64(0); i < length; i++ {
		if isCopy {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcPtr, Pos: pos})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(i), Pos: pos})
			l.emit(ir.IRInstr{Kind: loadKind, Pos: pos})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: elementBase + int(i), Pos: pos})
	}
	l.scalarSlices[name] = scalarSliceLocal{
		elemType:    elem,
		length:      length,
		elementBase: elementBase,
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(length), Pos: pos})
	return true, 2, nil
}

func (l *lowerer) lowerScalarIndexStore(
	index *frontend.IndexExpr,
	value frontend.Expr,
	pos frontend.Position,
) (bool, error) {
	meta, indexValue, ok, err := l.scalarSliceIndex(index)
	if err != nil || !ok {
		return ok, err
	}
	slots, err := l.lowerExprAs(value, meta.elemType)
	if err != nil {
		return true, err
	}
	if slots != 1 {
		return true, fmt.Errorf(
			"%s: scalar-replaced slice store expects single-slot element",
			frontend.FormatPos(pos),
		)
	}
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: meta.elementBase + int(indexValue), Pos: pos})
	return true, nil
}

func (l *lowerer) lowerScalarIndexLoad(index *frontend.IndexExpr) (bool, int, error) {
	meta, indexValue, ok, err := l.scalarSliceIndex(index)
	if err != nil || !ok {
		return ok, 0, err
	}
	l.emit(
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: meta.elementBase + int(indexValue), Pos: index.At},
	)
	return true, 1, nil
}

func (l *lowerer) scalarSliceIndex(
	index *frontend.IndexExpr,
) (scalarSliceLocal, int64, bool, error) {
	if index == nil {
		return scalarSliceLocal{}, 0, false, nil
	}
	base, ok := index.Base.(*frontend.IdentExpr)
	if !ok || base == nil {
		return scalarSliceLocal{}, 0, false, nil
	}
	meta, ok := l.scalarSlices[base.Name]
	if !ok {
		return scalarSliceLocal{}, 0, false, nil
	}
	indexValue, known := evalConstInt64ForAllocation(index.Index)
	if !known {
		return scalarSliceLocal{}, 0, true, fmt.Errorf(
			"%s: scalar-replaced slice '%s' has non-constant index after allocation planning",
			frontend.FormatPos(index.At),
			base.Name,
		)
	}
	if indexValue < 0 || indexValue >= meta.length {
		return scalarSliceLocal{}, 0, true, fmt.Errorf(
			"%s: scalar-replaced slice '%s' has out-of-range constant index %d",
			frontend.FormatPos(index.At),
			base.Name,
			indexValue,
		)
	}
	return meta, indexValue, true, nil
}

func (l *lowerer) lowerFunctionTempRegionCopyLet(
	name string,
	info semantics.LocalInfo,
	expr frontend.Expr,
	pos frontend.Position,
) (bool, int, error) {
	if !l.functionTempRegionLowering || info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 1 {
		return false, 0, nil
	}
	elem, ok := copyBuiltinElement(call.Name)
	if !ok {
		return false, 0, nil
	}
	alloc, ok := l.allocationPlan[name]
	if !ok || alloc.ActualLoweringStorage != allocplan.StorageFunctionTempRegion {
		return false, 0, nil
	}
	_, loadKind, storeKind, ok := copyElementIRKinds(elem, l.types)
	if !ok {
		return false, 0, nil
	}
	regionKind, ok := regionSliceKindByElement(elem)
	if !ok {
		return false, 0, nil
	}
	sourceSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return false, 0, err
	}
	if sourceSlots != 2 {
		return false, 0, fmt.Errorf(
			"%s: %s expects one view source argument",
			frontend.FormatPos(pos),
			call.Name,
		)
	}
	srcLen := l.allocScratchSlots(1)
	srcPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})

	l.ensureFunctionTempRegion(pos)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: regionKind, Name: name, Pos: pos})
	dstLen := l.allocScratchSlots(1)
	dstPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstPtr, Pos: pos})

	l.emitCopyLoop(
		srcPtr,
		srcLen,
		dstPtr,
		dstLen,
		loadKind,
		storeKind,
		copyLoopBoundsProofID(call.Name, call.At),
		pos,
	)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstLen, Pos: pos})
	return true, 2, nil
}

func (l *lowerer) lowerExplicitIslandAllocationLet(
	name string,
	info semantics.LocalInfo,
	expr frontend.Expr,
	pos frontend.Position,
) (bool, int, error) {
	if info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 2 {
		return false, 0, nil
	}
	kind, ok := islandSliceKindByBuiltin(call.Name)
	if !ok {
		return false, 0, nil
	}
	alloc, ok := l.allocationPlan[name]
	if !ok || alloc.ActualLoweringStorage != allocplan.StorageExplicitIsland {
		return false, 0, nil
	}
	islandSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return false, 0, err
	}
	if islandSlots != 1 {
		return false, 0, fmt.Errorf(
			"%s: %s expects island handle argument",
			frontend.FormatPos(pos),
			call.Name,
		)
	}
	lengthSlots, err := l.lowerExpr(call.Args[1])
	if err != nil {
		return false, 0, err
	}
	if lengthSlots != 1 {
		return false, 0, fmt.Errorf(
			"%s: %s expects length argument",
			frontend.FormatPos(pos),
			call.Name,
		)
	}
	l.emit(ir.IRInstr{Kind: kind, Name: name, Pos: pos})
	return true, 2, nil
}

func (l *lowerer) ensureFunctionTempRegion(pos frontend.Position) {
	if l.functionTempRegionEntered {
		return
	}
	l.emit(ir.IRInstr{Kind: ir.IRRegionEnter, Pos: pos})
	l.functionTempRegionEntered = true
}

func (l *lowerer) emitFunctionTempRegionReset(pos frontend.Position) {
	if !l.functionTempRegionEntered {
		return
	}
	l.emit(ir.IRInstr{Kind: ir.IRRegionReset, Pos: pos})
}

func (l *lowerer) lowerStackCopyLet(
	name string,
	info semantics.LocalInfo,
	expr frontend.Expr,
	pos frontend.Position,
) (bool, int, error) {
	if !l.stackAllocationLowering || info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 1 {
		return false, 0, nil
	}
	elem, ok := copyBuiltinElement(call.Name)
	if !ok {
		return false, 0, nil
	}
	alloc, ok := l.allocationPlan[name]
	if !ok || alloc.ActualLoweringStorage != allocplan.StorageStack || alloc.ByteSize <= 0 ||
		alloc.ElementSize <= 0 {
		return false, 0, nil
	}
	_, loadKind, storeKind, ok := copyElementIRKinds(elem, l.types)
	if !ok {
		return false, 0, nil
	}
	stackKind, ok := stackSliceKindByElement(elem)
	if !ok {
		return false, 0, nil
	}
	sourceSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return false, 0, err
	}
	if sourceSlots != 2 {
		return false, 0, fmt.Errorf(
			"%s: %s expects one view source argument",
			frontend.FormatPos(pos),
			call.Name,
		)
	}
	srcLen := l.allocScratchSlots(1)
	srcPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})

	backingSlots := (alloc.ByteSize + 7) / 8
	backingBase := l.allocScratchSlots(backingSlots)
	logicalLen := alloc.ByteSize / alloc.ElementSize
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(
		ir.IRInstr{
			Kind:     stackKind,
			Local:    backingBase,
			ArgSlots: backingSlots,
			Imm:      int32(logicalLen),
			Name:     name,
			Pos:      pos,
		},
	)
	dstLen := l.allocScratchSlots(1)
	dstPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstPtr, Pos: pos})

	l.emitCopyLoop(
		srcPtr,
		srcLen,
		dstPtr,
		dstLen,
		loadKind,
		storeKind,
		copyLoopBoundsProofID(name, pos),
		pos,
	)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstLen, Pos: pos})
	return true, 2, nil
}

func (l *lowerer) lowerStackAllocationLet(
	name string,
	info semantics.LocalInfo,
	expr frontend.Expr,
	pos frontend.Position,
) (bool, int, error) {
	if !l.stackAllocationLowering || info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 1 {
		return false, 0, nil
	}
	alloc, ok := l.allocationPlan[name]
	if !ok {
		return false, 0, nil
	}
	if alloc.ActualLoweringStorage != allocplan.StorageStack &&
		alloc.ActualLoweringStorage != allocplan.StorageEliminated {
		return false, 0, nil
	}
	length, known := l.proofConstIntValue(call.Args[0])
	if !known {
		return false, 0, nil
	}
	kind, ok := stackSliceKindByBuiltin(call.Name)
	if !ok {
		return false, 0, nil
	}
	lengthSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return false, 0, err
	}
	if lengthSlots != 1 {
		return false, 0, fmt.Errorf("%s: allocation length must be i32", frontend.FormatPos(pos))
	}
	if alloc.ActualLoweringStorage == allocplan.StorageEliminated {
		if length != 0 {
			return false, 0, fmt.Errorf(
				"%s: eliminated allocation %q has non-zero length %d",
				frontend.FormatPos(pos),
				name,
				length,
			)
		}
		l.emit(ir.IRInstr{Kind: kind, Local: -1, ArgSlots: 0, Imm: 0, Name: name, Pos: pos})
		return true, 2, nil
	}
	if length <= 0 || alloc.ByteSize <= 0 {
		return false, 0, nil
	}
	backingSlots := (alloc.ByteSize + 7) / 8
	backingBase := l.allocScratchSlots(backingSlots)
	l.emit(
		ir.IRInstr{
			Kind:     kind,
			Local:    backingBase,
			ArgSlots: backingSlots,
			Imm:      int32(length),
			Name:     name,
			Pos:      pos,
		},
	)
	return true, 2, nil
}

func stackSliceKindByBuiltin(name string) (ir.IRInstrKind, bool) {
	return lowerlets.StackSliceKindByBuiltin(name)
}

func stackAllocationElementByBuiltin(name string) (string, bool) {
	return lowerlets.StackAllocationElementByBuiltin(name)
}

func stackSliceKindByElement(elem string) (ir.IRInstrKind, bool) {
	return lowerlets.StackSliceKindByElement(elem)
}

func regionSliceKindByElement(elem string) (ir.IRInstrKind, bool) {
	return lowerlets.RegionSliceKindByElement(elem)
}

func islandSliceKindByBuiltin(name string) (ir.IRInstrKind, bool) {
	return lowerlets.IslandSliceKindByBuiltin(name)
}

func (l *lowerer) allocationNameForBuiltinCall(
	name string,
	pos frontend.Position,
	storage allocplan.StorageClass,
) string {
	if len(l.allocationPlan) == 0 {
		return ""
	}
	source := frontend.FormatPos(pos)
	for id, alloc := range l.allocationPlan {
		if alloc.Builtin == name && alloc.Source == source &&
			alloc.ActualLoweringStorage == storage {
			return id
		}
	}
	return ""
}

func (l *lowerer) lowerMatchExpr(e *frontend.MatchExpr) (int, error) {
	info, ok := l.locals[e.ScrutineeLocal]
	if !ok {
		return 0, fmt.Errorf(
			"%s: unknown match expression scrutinee local",
			frontend.FormatPos(e.At),
		)
	}
	resultInfo, ok := l.locals[e.ResultLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown match expression result local", frontend.FormatPos(e.At))
	}
	valueSlots, err := l.lowerExpr(e.Value)
	if err != nil {
		return 0, err
	}
	if valueSlots != info.SlotCount {
		return 0, fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(e.At))
	}
	for i := info.SlotCount - 1; i >= 0; i-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: e.At})
	}
	endLabel := l.newLabel()
	defaultLabel := -1
	caseLabels := make([]int, len(e.Cases))
	guardFailLabels := make([]int, len(e.Cases))
	scrutTypeInfo, scrutTypeOK := l.types[info.TypeName]
	for i, c := range e.Cases {
		guardFailLabels[i] = endLabel
		caseLabels[i] = l.newLabel()
		if c.Default {
			defaultLabel = caseLabels[i]
			continue
		}
		nextLabel := l.newLabel()
		guardFailLabels[i] = nextLabel
		if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeOptional {
			if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				l.emit(
					ir.IRInstr{
						Kind:  ir.IRLoadLocal,
						Local: info.Base + info.SlotCount - 1,
						Pos:   c.At,
					},
				)
				l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
				continue
			}
			if !isNoneExpr(c.Pattern) {
				return 0, fmt.Errorf(
					"%s: optional match supports only 'none', 'some(name)', and '_' patterns",
					frontend.FormatPos(c.At),
				)
			}
			l.emit(
				ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At},
			)
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: c.At})
		} else if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeEnum {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
			switch pat := c.Pattern.(type) {
			case *frontend.FieldAccessExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum match pattern was not resolved", frontend.FormatPos(c.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			case *frontend.EnumCasePatternExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum match pattern was not resolved", frontend.FormatPos(c.At))
				}
				if err := l.validateEnumPatternLayout(pat, info); err != nil {
					return 0, err
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			default:
				return 0, fmt.Errorf(
					"%s: enum match supports enum case patterns and '_'",
					frontend.FormatPos(c.At),
				)
			}
		} else {
			if info.SlotCount != 1 {
				return 0, fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
			patSlots, err := l.lowerExpr(c.Pattern)
			if err != nil {
				return 0, err
			}
			if patSlots != 1 {
				return 0, fmt.Errorf("%s: match pattern slot mismatch", frontend.FormatPos(c.At))
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
	}
	if defaultLabel >= 0 {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: defaultLabel, Pos: e.At})
	} else {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
	}
	for i, c := range e.Cases {
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: caseLabels[i], Pos: c.At})
		if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			bindInfo, ok := l.locals[some.Name]
			if !ok {
				return 0, fmt.Errorf(
					"%s: unknown some binding '%s'",
					frontend.FormatPos(some.At),
					some.Name,
				)
			}
			if bindInfo.SlotCount != info.SlotCount-1 {
				return 0, fmt.Errorf(
					"%s: optional some binding slot mismatch",
					frontend.FormatPos(some.At),
				)
			}
			for slot := 0; slot < bindInfo.SlotCount; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + slot, Pos: some.At})
			}
			for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: some.At})
			}
		}
		if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
			if err := l.emitIfLetPatternBindings(enumPat, info); err != nil {
				return 0, err
			}
		}
		if c.Guard != nil {
			slots, err := l.lowerExpr(c.Guard)
			if err != nil {
				return 0, err
			}
			if slots != 1 {
				return 0, fmt.Errorf(
					"%s: match guard must be single-slot",
					frontend.FormatPos(c.Guard.Pos()),
				)
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: guardFailLabels[i], Pos: c.Guard.Pos()})
		}
		slots, err := l.lowerExprAs(c.Value, e.ResultType)
		if err != nil {
			return 0, err
		}
		if slots != resultInfo.SlotCount {
			return 0, fmt.Errorf(
				"%s: match expression result slot mismatch",
				frontend.FormatPos(c.At),
			)
		}
		for slot := resultInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultInfo.Base + slot, Pos: c.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: c.At})
	}
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
	for slot := 0; slot < resultInfo.SlotCount; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultInfo.Base + slot, Pos: e.At})
	}
	return resultInfo.SlotCount, nil
}

func (l *lowerer) lowerCatchExpr(e *frontend.CatchExpr) (int, error) {
	call, ok := e.Call.(*frontend.CallExpr)
	if !ok {
		return 0, fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
	}
	errorInfo, ok := l.locals[e.ErrorLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown catch error local", frontend.FormatPos(e.At))
	}
	resultInfo, ok := l.locals[e.ResultLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown catch result local", frontend.FormatPos(e.At))
	}
	call = lowerCallExprWithBuiltinAlias(call)
	var callSuccessSlots int
	var callErrorSlots int
	var callCompact bool
	var expectedReturnSlots int
	if isTypedTaskJoinCall(call.Name) {
		if len(call.TypeArgs) != 1 || call.TypeArgs[0].Name == "" {
			return 0, fmt.Errorf(
				"%s: task_join_i32_typed missing resolved error type",
				frontend.FormatPos(call.At),
			)
		}
		errorInfo, ok := l.types[call.TypeArgs[0].Name]
		if !ok || errorInfo.Kind != semantics.TypeEnum {
			return 0, fmt.Errorf(
				"%s: typed task error argument must be an enum",
				frontend.FormatPos(call.TypeArgs[0].At),
			)
		}
		_, handleInfo, err := semantics.EnsureTypedTaskHandleType(call.TypeArgs[0].Name, l.types)
		if err != nil {
			return 0, fmt.Errorf("%s: %v", frontend.FormatPos(call.TypeArgs[0].At), err)
		}
		callSuccessSlots = 1
		callErrorSlots = errorInfo.SlotCount
		callCompact = errorInfo.SlotCount == 1
		expectedReturnSlots = handleInfo.SlotCount
	} else {
		sig, ok := l.funcs[call.Name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
		}
		if sig.ThrowsType == "" {
			return 0, fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
		}
		var err error
		callSuccessSlots, callErrorSlots, callCompact, err = throwingLayout(
			sig.ReturnType,
			sig.ThrowsType,
			l.types,
		)
		if err != nil {
			return 0, err
		}
		expectedReturnSlots = sig.ReturnSlots
	}
	if callSuccessSlots != resultInfo.SlotCount || callErrorSlots != errorInfo.SlotCount {
		return 0, fmt.Errorf("%s: catch slot mismatch", frontend.FormatPos(e.At))
	}
	var slots int
	var err error
	if isTypedTaskJoinCall(call.Name) {
		slots, err = l.lowerTypedTaskJoinForCatch(call, e.At)
	} else {
		slots, err = l.lowerExpr(call)
	}
	if err != nil {
		return 0, err
	}
	if slots != expectedReturnSlots {
		return 0, fmt.Errorf("%s: catch call result slot mismatch", frontend.FormatPos(e.At))
	}

	successLabel := l.newLabel()
	endLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: successLabel, Pos: e.At})

	if callCompact {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: errorInfo.Base, Pos: e.At})
	} else {
		for slot := callErrorSlots - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: errorInfo.Base + slot, Pos: e.At})
		}
		discard := l.ensureDiscardLocal()
		for slot := 0; slot < callSuccessSlots; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
		}
	}

	defaultLabel := -1
	caseLabels := make([]int, len(e.Cases))
	guardFailLabels := make([]int, len(e.Cases))
	errorTypeInfo, errorTypeOK := l.types[errorInfo.TypeName]
	for i, c := range e.Cases {
		guardFailLabels[i] = endLabel
		caseLabels[i] = l.newLabel()
		if c.Default {
			defaultLabel = caseLabels[i]
			continue
		}
		nextLabel := l.newLabel()
		guardFailLabels[i] = nextLabel
		if errorTypeOK && errorTypeInfo.Kind == semantics.TypeOptional {
			if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				l.emit(
					ir.IRInstr{
						Kind:  ir.IRLoadLocal,
						Local: errorInfo.Base + errorInfo.SlotCount - 1,
						Pos:   c.At,
					},
				)
				l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
				continue
			}
			if !isNoneExpr(c.Pattern) {
				return 0, fmt.Errorf(
					"%s: optional catch supports only 'none', 'some(name)', and '_' patterns",
					frontend.FormatPos(c.At),
				)
			}
			l.emit(
				ir.IRInstr{
					Kind:  ir.IRLoadLocal,
					Local: errorInfo.Base + errorInfo.SlotCount - 1,
					Pos:   c.At,
				},
			)
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: c.At})
		} else if errorTypeOK && errorTypeInfo.Kind == semantics.TypeEnum {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errorInfo.Base, Pos: c.At})
			switch pat := c.Pattern.(type) {
			case *frontend.FieldAccessExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum catch pattern was not resolved", frontend.FormatPos(c.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			case *frontend.EnumCasePatternExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum catch pattern was not resolved", frontend.FormatPos(c.At))
				}
				if err := l.validateEnumPatternLayout(pat, errorInfo); err != nil {
					return 0, err
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			default:
				return 0, fmt.Errorf(
					"%s: enum catch supports enum case patterns and '_'",
					frontend.FormatPos(c.At),
				)
			}
		} else {
			if errorInfo.SlotCount != 1 {
				return 0, fmt.Errorf("%s: catch error slot mismatch", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errorInfo.Base, Pos: c.At})
			patSlots, err := l.lowerExpr(c.Pattern)
			if err != nil {
				return 0, err
			}
			if patSlots != 1 {
				return 0, fmt.Errorf("%s: catch pattern slot mismatch", frontend.FormatPos(c.At))
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
	}
	if defaultLabel >= 0 {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: defaultLabel, Pos: e.At})
	} else {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
	}
	for i, c := range e.Cases {
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: caseLabels[i], Pos: c.At})
		if err := l.emitIfLetPatternBindings(c.Pattern, errorInfo); err != nil {
			return 0, err
		}
		if c.Guard != nil {
			slots, err := l.lowerExpr(c.Guard)
			if err != nil {
				return 0, err
			}
			if slots != 1 {
				return 0, fmt.Errorf(
					"%s: catch guard must be single-slot",
					frontend.FormatPos(c.Guard.Pos()),
				)
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: guardFailLabels[i], Pos: c.Guard.Pos()})
		}
		slots, err := l.lowerExprAs(c.Value, e.ResultType)
		if err != nil {
			return 0, err
		}
		if slots != resultInfo.SlotCount {
			return 0, fmt.Errorf(
				"%s: catch expression result slot mismatch",
				frontend.FormatPos(c.At),
			)
		}
		for slot := resultInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultInfo.Base + slot, Pos: c.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: c.At})
	}

	successEntrySlots := callSuccessSlots
	if !callCompact {
		successEntrySlots += callErrorSlots
	}
	l.emitZeroSlots(successEntrySlots, e.At)
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: successLabel, Pos: e.At})
	if !callCompact {
		discard := l.ensureDiscardLocal()
		for slot := 0; slot < callErrorSlots; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
		}
	}
	for slot := resultInfo.SlotCount - 1; slot >= 0; slot-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultInfo.Base + slot, Pos: e.At})
	}
	l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
	for slot := 0; slot < resultInfo.SlotCount; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultInfo.Base + slot, Pos: e.At})
	}
	return resultInfo.SlotCount, nil
}

func (l *lowerer) emitIfLetPatternCheck(
	pattern frontend.Expr,
	valueInfo semantics.LocalInfo,
	elseLabel int,
	pos frontend.Position,
) error {
	scrutTypeInfo, scrutTypeOK := l.types[valueInfo.TypeName]
	if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeOptional {
		if _, ok := pattern.(*frontend.SomePatternExpr); ok {
			l.emit(
				ir.IRInstr{
					Kind:  ir.IRLoadLocal,
					Local: valueInfo.Base + valueInfo.SlotCount - 1,
					Pos:   pos,
				},
			)
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: pos})
			return nil
		}
		if !isNoneExpr(pattern) {
			return fmt.Errorf(
				"%s: optional if let supports only 'none' and 'some(name)' patterns",
				frontend.FormatPos(pos),
			)
		}
		l.emit(
			ir.IRInstr{
				Kind:  ir.IRLoadLocal,
				Local: valueInfo.Base + valueInfo.SlotCount - 1,
				Pos:   pos,
			},
		)
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: pos})
		return nil
	}
	if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeEnum {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base, Pos: pos})
		switch pat := pattern.(type) {
		case *frontend.FieldAccessExpr:
			if pat.EnumType == "" {
				return fmt.Errorf("%s: enum if-let pattern was not resolved", frontend.FormatPos(pos))
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: pos})
		case *frontend.EnumCasePatternExpr:
			if pat.EnumType == "" {
				return fmt.Errorf("%s: enum if-let pattern was not resolved", frontend.FormatPos(pos))
			}
			if err := l.validateEnumPatternLayout(pat, valueInfo); err != nil {
				return err
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: pos})
		default:
			return fmt.Errorf("%s: enum if let supports enum case patterns", frontend.FormatPos(pos))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: pos})
		return nil
	}
	return fmt.Errorf("%s: if let pattern requires optional or enum value", frontend.FormatPos(pos))
}

func enumPayloadSlotCount(
	pat *frontend.EnumCasePatternExpr,
	fallbackBindings map[string]semantics.LocalInfo,
) (int, error) {
	if pat == nil {
		return 0, nil
	}
	if len(pat.PayloadSlots) > 0 {
		if len(pat.PayloadSlots) != len(pat.Bindings) {
			return 0, fmt.Errorf(
				"%s: enum payload pattern slot metadata mismatch",
				frontend.FormatPos(pat.At),
			)
		}
		total := 0
		for _, slots := range pat.PayloadSlots {
			if slots <= 0 {
				return 0, fmt.Errorf(
					"%s: enum payload pattern slot metadata mismatch",
					frontend.FormatPos(pat.At),
				)
			}
			total += slots
		}
		return total, nil
	}
	total := 0
	for _, binding := range pat.Bindings {
		bindInfo, ok := fallbackBindings[binding]
		if !ok {
			return 0, fmt.Errorf(
				"%s: unknown enum payload binding '%s'",
				frontend.FormatPos(pat.At),
				binding,
			)
		}
		if bindInfo.SlotCount <= 0 {
			return 0, fmt.Errorf(
				"%s: enum payload binding '%s' slot mismatch",
				frontend.FormatPos(pat.At),
				binding,
			)
		}
		total += bindInfo.SlotCount
	}
	return total, nil
}

func (l *lowerer) validateEnumPatternLayout(
	pattern frontend.Expr,
	valueInfo semantics.LocalInfo,
) error {
	enumPat, ok := pattern.(*frontend.EnumCasePatternExpr)
	if !ok {
		return nil
	}
	payloadSlots, err := enumPayloadSlotCount(enumPat, l.locals)
	if err != nil {
		return err
	}
	if payloadSlots > valueInfo.SlotCount-1 {
		return fmt.Errorf(
			"%s: enum payload pattern exceeds value layout",
			frontend.FormatPos(enumPat.At),
		)
	}
	return nil
}

func (l *lowerer) emitIfLetPatternBindings(
	pattern frontend.Expr,
	valueInfo semantics.LocalInfo,
) error {
	if some, ok := pattern.(*frontend.SomePatternExpr); ok {
		bindInfo, ok := l.locals[some.Name]
		if !ok {
			return fmt.Errorf(
				"%s: unknown some binding '%s'",
				frontend.FormatPos(some.At),
				some.Name,
			)
		}
		for slot := 0; slot < bindInfo.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + slot, Pos: some.At})
		}
		for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: some.At})
		}
	}
	if enumPat, ok := pattern.(*frontend.EnumCasePatternExpr); ok {
		payloadOffset := 1
		for i, binding := range enumPat.Bindings {
			bindInfo, ok := l.locals[binding]
			if !ok {
				return fmt.Errorf(
					"%s: unknown enum payload binding '%s'",
					frontend.FormatPos(enumPat.At),
					binding,
				)
			}
			wantSlots := bindInfo.SlotCount
			if i < len(enumPat.PayloadSlots) {
				wantSlots = enumPat.PayloadSlots[i]
			}
			if bindInfo.SlotCount != wantSlots {
				return fmt.Errorf(
					"%s: enum payload binding '%s' slot mismatch",
					frontend.FormatPos(enumPat.At),
					binding,
				)
			}
			for slot := 0; slot < bindInfo.SlotCount; slot++ {
				l.emit(
					ir.IRInstr{
						Kind:  ir.IRLoadLocal,
						Local: valueInfo.Base + payloadOffset + slot,
						Pos:   enumPat.At,
					},
				)
			}
			for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
				l.emit(
					ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: enumPat.At},
				)
			}
			payloadOffset += wantSlots
		}
	}
	return nil
}

func rawPtrAddCall(expr frontend.Expr) (*frontend.CallExpr, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok {
		return nil, false
	}
	name := call.Name
	if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = builtin
	}
	if name != "core.ptr_add" {
		return nil, false
	}
	return call, true
}

func (l *lowerer) rawPtrOffsetAliasFromExpr(expr frontend.Expr) (rawPtrOffsetLocal, bool) {
	call, ok := rawPtrAddCall(expr)
	if !ok || len(call.Args) != 3 {
		return rawPtrOffsetLocal{}, false
	}
	base, ok := call.Args[0].(*frontend.IdentExpr)
	if !ok {
		return rawPtrOffsetLocal{}, false
	}
	baseInfo, ok := l.locals[base.Name]
	if !ok || baseInfo.SlotCount != 1 {
		return rawPtrOffsetLocal{}, false
	}
	alias := rawPtrOffsetLocal{BaseLocal: baseInfo.Base, OffsetLocal: -1}
	if prior, ok := l.rawPtrOffsetLocals[baseInfo.Base]; ok {
		alias = prior
	}
	switch offset := call.Args[1].(type) {
	case *frontend.NumberExpr:
		if alias.HasOffsetImm {
			alias.OffsetImm += offset.Value
		} else if alias.OffsetLocal < 0 {
			alias.OffsetImm = offset.Value
			alias.HasOffsetImm = true
		} else {
			return rawPtrOffsetLocal{}, false
		}
	case *frontend.IdentExpr:
		if alias.HasOffsetImm || alias.OffsetLocal >= 0 {
			return rawPtrOffsetLocal{}, false
		}
		offsetInfo, ok := l.locals[offset.Name]
		if !ok || offsetInfo.SlotCount != 1 {
			return rawPtrOffsetLocal{}, false
		}
		alias.OffsetLocal = offsetInfo.Base
	default:
		return rawPtrOffsetLocal{}, false
	}
	return alias, true
}

func (l *lowerer) rememberRawPtrOffsetAlias(local int, expr frontend.Expr) {
	l.clearRawPtrOffsetAliasesForLocal(local)
	alias, ok := l.rawPtrOffsetAliasFromExpr(expr)
	if !ok {
		delete(l.rawPtrOffsetLocals, local)
		return
	}
	l.rawPtrOffsetLocals[local] = alias
}

func (l *lowerer) clearRawPtrOffsetAliasesForLocal(local int) {
	delete(l.rawPtrOffsetLocals, local)
	for aliasLocal, alias := range l.rawPtrOffsetLocals {
		if alias.BaseLocal == local || (!alias.HasOffsetImm && alias.OffsetLocal == local) {
			delete(l.rawPtrOffsetLocals, aliasLocal)
		}
	}
}

func (l *lowerer) lowerRawOffsetAlias(alias rawPtrOffsetLocal, pos frontend.Position) {
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: alias.BaseLocal, Pos: pos})
	if alias.HasOffsetImm {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: alias.OffsetImm, Pos: pos})
		return
	}
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: alias.OffsetLocal, Pos: pos})
}

func (l *lowerer) lowerRawOffsetAddress(expr frontend.Expr, pos frontend.Position) (bool, error) {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if info, ok := l.locals[id.Name]; ok {
			if alias, ok := l.rawPtrOffsetLocals[info.Base]; ok {
				l.lowerRawOffsetAlias(alias, pos)
				return true, nil
			}
		}
	}
	call, ok := rawPtrAddCall(expr)
	if !ok {
		return false, nil
	}
	if len(call.Args) != 3 {
		return true, fmt.Errorf("%s: ptr_add expects 3 arguments", frontend.FormatPos(call.At))
	}
	baseSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return true, err
	}
	if baseSlots != 1 {
		return true, fmt.Errorf(
			"%s: ptr_add expects a 1-slot base pointer",
			frontend.FormatPos(call.Args[0].Pos()),
		)
	}
	offsetSlots, err := l.lowerExpr(call.Args[1])
	if err != nil {
		return true, err
	}
	if offsetSlots != 1 {
		return true, fmt.Errorf(
			"%s: ptr_add expects a 1-slot offset",
			frontend.FormatPos(call.Args[1].Pos()),
		)
	}
	memSlots, err := l.lowerExpr(call.Args[2])
	if err != nil {
		return true, err
	}
	if memSlots != 1 {
		return true, fmt.Errorf(
			"%s: ptr_add expects a 1-slot memory capability",
			frontend.FormatPos(call.Args[2].Pos()),
		)
	}
	discard := l.ensureDiscardLocal()
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	return true, nil
}

func (l *lowerer) lowerSurfaceRuntimeCall(
	e *frontend.CallExpr,
	runtimeName string,
	expectedArgSlots int,
) (int, error) {
	total := 0
	for _, arg := range e.Args {
		slots, err := l.lowerExpr(arg)
		if err != nil {
			return 0, err
		}
		total += slots
	}
	if total != expectedArgSlots {
		return 0, fmt.Errorf(
			"%s: %s lowered %d argument slots, want %d",
			frontend.FormatPos(e.At),
			e.Name,
			total,
			expectedArgSlots,
		)
	}
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: runtimeName, ArgSlots: total, RetSlots: 1, Pos: e.At})
	return 1, nil
}

func (l *lowerer) lowerRawOffsetCall(e *frontend.CallExpr) (int, bool, error) {
	switch e.Name {
	case "core.load_i32", "core.load_u8", "core.load_ptr":
		if len(e.Args) != 2 {
			return 0, true, fmt.Errorf(
				"%s: %s expects 2 arguments",
				frontend.FormatPos(e.At),
				strings.TrimPrefix(e.Name, "core."),
			)
		}
		ok, err := l.lowerRawOffsetAddress(e.Args[0], e.At)
		if err != nil || !ok {
			return 0, ok, err
		}
		memSlots, err := l.lowerExpr(e.Args[1])
		if err != nil {
			return 0, true, err
		}
		if memSlots != 1 {
			return 0, true, fmt.Errorf(
				"%s: %s expects a 1-slot memory capability",
				frontend.FormatPos(e.Args[1].Pos()),
				strings.TrimPrefix(e.Name, "core."),
			)
		}
		switch e.Name {
		case "core.load_i32":
			l.emit(ir.IRInstr{Kind: ir.IRMemReadI32Offset, Pos: e.At})
		case "core.load_u8":
			l.emit(ir.IRInstr{Kind: ir.IRMemReadU8Offset, Pos: e.At})
		default:
			l.emit(ir.IRInstr{Kind: ir.IRMemReadPtrOffset, Pos: e.At})
		}
		return 1, true, nil
	case "core.store_i32", "core.store_u8", "core.store_ptr", "core.store_arch_ptr":
		if len(e.Args) != 3 {
			return 0, true, fmt.Errorf(
				"%s: %s expects 3 arguments",
				frontend.FormatPos(e.At),
				strings.TrimPrefix(e.Name, "core."),
			)
		}
		ok, err := l.lowerRawOffsetAddress(e.Args[0], e.At)
		if err != nil || !ok {
			return 0, ok, err
		}
		valueSlots, err := l.lowerExpr(e.Args[1])
		if err != nil {
			return 0, true, err
		}
		if valueSlots != 1 {
			return 0, true, fmt.Errorf(
				"%s: %s expects a 1-slot value",
				frontend.FormatPos(e.Args[1].Pos()),
				strings.TrimPrefix(e.Name, "core."),
			)
		}
		memSlots, err := l.lowerExpr(e.Args[2])
		if err != nil {
			return 0, true, err
		}
		if memSlots != 1 {
			return 0, true, fmt.Errorf(
				"%s: %s expects a 1-slot memory capability",
				frontend.FormatPos(e.Args[2].Pos()),
				strings.TrimPrefix(e.Name, "core."),
			)
		}
		switch e.Name {
		case "core.store_i32":
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteI32Offset, Pos: e.At})
		case "core.store_u8":
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteU8Offset, Pos: e.At})
		case "core.store_arch_ptr":
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteArchPtrOffset, Pos: e.At})
		default:
			l.emit(ir.IRInstr{Kind: ir.IRMemWritePtrOffset, Pos: e.At})
		}
		return 1, true, nil
	default:
		return 0, false, nil
	}
}

func (l *lowerer) lowerPtrAddValueCall(e *frontend.CallExpr) (int, bool, error) {
	if e.Name != "core.ptr_add" {
		return 0, false, nil
	}
	if len(e.Args) != 3 {
		return 0, true, fmt.Errorf("%s: ptr_add expects 3 arguments", frontend.FormatPos(e.At))
	}
	alias, ok := l.rawPtrOffsetAliasFromExpr(e)
	if !ok {
		return 0, false, nil
	}
	l.lowerRawOffsetAlias(alias, e.At)
	memSlots, err := l.lowerExpr(e.Args[2])
	if err != nil {
		return 0, true, err
	}
	if memSlots != 1 {
		return 0, true, fmt.Errorf(
			"%s: ptr_add expects a 1-slot memory capability",
			frontend.FormatPos(e.Args[2].Pos()),
		)
	}
	l.emit(ir.IRInstr{Kind: ir.IRPtrAdd, Pos: e.At})
	return 1, true, nil
}

// ---- lower_lvalues_copy.go ----

func (l *lowerer) emitGlobalStringLiteralInitIfNeeded(
	g semantics.GlobalInfo,
	pos frontend.Position,
) {
	if g.TypeName != "str" || !g.HasStringLiteralInit {
		return
	}
	readyLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: readyLabel, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStrLit, Str: g.StringLiteralInit, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex + 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: readyLabel, Pos: pos})
}

func (l *lowerer) emitGlobalArrayBackingsInitIfNeeded(
	g semantics.GlobalInfo,
	pos frontend.Position,
) {
	for _, backing := range g.ArrayBackings {
		byteLen := globalArrayBackingByteLen(backing.ElemType, backing.Len, l.types)
		if byteLen <= 0 {
			continue
		}
		ptrSlot := g.DataIndex + backing.HeaderOffset
		lenSlot := ptrSlot + 1
		readyLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: ptrSlot, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: readyLabel, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStrLit, Str: make([]byte, byteLen), Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.ensureDiscardLocal(), Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: ptrSlot, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(backing.Len), Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: lenSlot, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: readyLabel, Pos: pos})
	}
}

func globalArrayBackingByteLen(elemType string, n int, types map[string]*semantics.TypeInfo) int {
	if n <= 0 {
		return 0
	}
	switch elemType {
	case "u8":
		return n
	case "u16":
		return n * 2
	case "i32", "c_int", "c_uint", "bool",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return n * 4
	}
	if info, ok := types[elemType]; ok && info.Kind == semantics.TypeStruct && info.SlotCount == 1 {
		return n * 4
	}
	return 0
}

func (l *lowerer) emitGlobalFunctionValueInitIfNeeded(
	g semantics.GlobalInfo,
	pos frontend.Position,
) {
	if !g.FunctionTypeValue || !g.Mutable || g.FunctionValue == "" {
		return
	}
	readyLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: readyLabel, Pos: pos})
	slots := l.emitFunctionSymbolValue(g.FunctionValue, nil, pos)
	for i := slots - 1; i >= 0; i-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex + i, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: readyLabel, Pos: pos})
}

type lvalueInfo struct {
	Base      int
	SlotCount int
	TypeName  string
	Name      string
	Global    bool
}

func (l *lowerer) resolveLValue(expr frontend.Expr) (lvalueInfo, error) {
	baseName, fields, pos, ok := splitFieldPathLower(expr)
	if !ok {
		return lvalueInfo{}, fmt.Errorf("%s: invalid assignment target", frontend.FormatPos(pos))
	}
	info, ok := l.locals[baseName]
	if !ok {
		if g, ok := l.globals[baseName]; ok {
			targetType, slotCount, offset, err := resolveFieldChainLower(
				g.TypeName,
				g.DataIndex,
				fields,
				l.types,
				pos,
			)
			if err != nil {
				return lvalueInfo{}, err
			}
			if _, ok := l.types[targetType]; !ok {
				return lvalueInfo{}, fmt.Errorf(
					"%s: unknown type '%s'",
					frontend.FormatPos(pos),
					targetType,
				)
			}
			return lvalueInfo{
				Base:      offset,
				SlotCount: slotCount,
				TypeName:  targetType,
				Name:      baseName,
				Global:    true,
			}, nil
		}
		return lvalueInfo{}, fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(pos), baseName)
	}
	targetType, slotCount, offset, err := resolveFieldChainLower(
		info.TypeName,
		info.Base,
		fields,
		l.types,
		pos,
	)
	if err != nil {
		return lvalueInfo{}, err
	}
	if _, ok := l.types[targetType]; !ok {
		return lvalueInfo{}, fmt.Errorf(
			"%s: unknown type '%s'",
			frontend.FormatPos(pos),
			targetType,
		)
	}
	return lvalueInfo{Base: offset, SlotCount: slotCount, TypeName: targetType, Name: baseName}, nil
}

func splitFieldPathLower(expr frontend.Expr) (string, []string, frontend.Position, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, nil, e.At, true
	case *frontend.FieldAccessExpr:
		baseName, fields, pos, ok := splitFieldPathLower(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, e.Field)
		return baseName, fields, e.At, true
	default:
		return "", nil, expr.Pos(), false
	}
}

func resolveFieldChainLower(
	typeName string,
	baseOffset int,
	fields []string,
	types map[string]*semantics.TypeInfo,
	pos frontend.Position,
) (string, int, int, error) {
	offset := baseOffset
	current := typeName
	for _, field := range fields {
		info, ok := types[current]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
		}
		if info.Kind != semantics.TypeStruct && info.Kind != semantics.TypeSlice &&
			info.Kind != semantics.TypeArray &&
			info.Kind != semantics.TypeStr {
			return "", 0, 0, fmt.Errorf(
				"%s: '%s' is not a struct",
				frontend.FormatPos(pos),
				current,
			)
		}
		fieldInfo, ok := info.FieldMap[field]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(pos), field)
		}
		offset += fieldInfo.Offset
		current = fieldInfo.TypeName
	}
	info, ok := types[current]
	if !ok {
		return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
	}
	return current, info.SlotCount, offset, nil
}

func isNoneExpr(expr frontend.Expr) bool {
	_, ok := expr.(*frontend.NoneLitExpr)
	return ok
}

func (l *lowerer) lowerOptionalTag(expr frontend.Expr) error {
	if isNoneExpr(expr) {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: expr.Pos()})
		return nil
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		info, ok := l.locals[e.Name]
		if !ok {
			return fmt.Errorf(
				"%s: optional comparison to none requires a stored optional value",
				frontend.FormatPos(e.At),
			)
		}
		typeInfo, ok := l.types[info.TypeName]
		if !ok || typeInfo.Kind != semantics.TypeOptional {
			return fmt.Errorf(
				"%s: optional comparison to none requires optional value",
				frontend.FormatPos(e.At),
			)
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + typeInfo.SlotCount - 1, Pos: e.At})
		return nil
	case *frontend.FieldAccessExpr:
		target, err := l.resolveLValue(e)
		if err != nil {
			return err
		}
		tname, err := l.inferExprType(e)
		if err != nil {
			return err
		}
		typeInfo, ok := l.types[tname]
		if !ok || typeInfo.Kind != semantics.TypeOptional {
			return fmt.Errorf(
				"%s: optional comparison to none requires optional value",
				frontend.FormatPos(e.At),
			)
		}
		kind := ir.IRLoadLocal
		if target.Global {
			kind = ir.IRLoadGlobal
		}
		l.emit(ir.IRInstr{Kind: kind, Local: target.Base + typeInfo.SlotCount - 1, Pos: e.At})
		return nil
	default:
		return fmt.Errorf(
			"%s: optional comparison to none requires a stored optional value",
			frontend.FormatPos(expr.Pos()),
		)
	}
}

func (l *lowerer) indexElemType(base frontend.Expr) (string, error) {
	baseType, err := l.inferExprType(base)
	if err != nil {
		return "", err
	}
	info, ok := l.types[baseType]
	if !ok {
		return "", fmt.Errorf("unknown type '%s'", baseType)
	}
	switch info.Kind {
	case semantics.TypeStr:
		return "u8", nil
	case semantics.TypeSlice:
		return info.ElemType, nil
	case semantics.TypeArray:
		return info.ElemType, nil
	default:
		return "", fmt.Errorf("%s: cannot index '%s'", frontend.FormatPos(base.Pos()), baseType)
	}
}

func lowerIndexLoadKind(
	elemType string,
	types map[string]*semantics.TypeInfo,
) (ir.IRInstrKind, bool) {
	return lowerexpressions.IndexLoadKind(elemType, types)
}

func uncheckedIndexLoadKind(kind ir.IRInstrKind) ir.IRInstrKind {
	return lowerexpressions.UncheckedIndexLoadKind(kind)
}

func sliceViewElementShift(name string) (int32, bool) {
	if strings.HasPrefix(name, "core.string_") {
		return 0, true
	}
	parts := strings.Split(name, "_")
	if len(parts) == 0 {
		return 0, false
	}
	switch parts[len(parts)-1] {
	case "u8":
		return 0, true
	case "u16":
		return 1, true
	case "i32", "bool":
		return 2, true
	default:
		return 0, false
	}
}

func (l *lowerer) lowerCopyBuiltinFromStack(
	name string,
	total int,
	pos frontend.Position,
) (int, error) {
	if total != 2 {
		return 0, fmt.Errorf(
			"%s: %s expects one view source argument",
			frontend.FormatPos(pos),
			name,
		)
	}
	elem, ok := copyBuiltinElement(name)
	if !ok {
		return 0, lowerUnsupportedError(pos, "unsupported copy builtin '%s'", name)
	}
	makeKind, loadKind, storeKind, ok := copyElementIRKinds(elem, l.types)
	if !ok {
		return 0, lowerUnsupportedError(pos, "unsupported copy element type '%s'", elem)
	}
	srcLen := l.allocScratchSlots(1)
	srcPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: makeKind, Pos: pos})
	dstLen := l.allocScratchSlots(1)
	dstPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstPtr, Pos: pos})

	l.emitCopyLoop(
		srcPtr,
		srcLen,
		dstPtr,
		dstLen,
		loadKind,
		storeKind,
		copyLoopBoundsProofID(name, pos),
		pos,
	)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstLen, Pos: pos})
	return 2, nil
}

func (l *lowerer) lowerCopyIntoBuiltinFromStack(
	name string,
	total int,
	pos frontend.Position,
) (int, error) {
	if total != 4 {
		return 0, fmt.Errorf(
			"%s: %s expects source and destination view arguments",
			frontend.FormatPos(pos),
			name,
		)
	}
	elem, ok := copyBuiltinElement(name)
	if !ok {
		return 0, lowerUnsupportedError(pos, "unsupported copy_into builtin '%s'", name)
	}
	_, loadKind, storeKind, ok := copyElementIRKinds(elem, l.types)
	if !ok {
		return 0, lowerUnsupportedError(pos, "unsupported copy_into element type '%s'", elem)
	}
	shift, ok := copyElementShift(elem)
	if !ok {
		return 0, lowerUnsupportedError(pos, "unsupported copy_into element shift for '%s'", elem)
	}
	dstLen := l.allocScratchSlots(1)
	dstPtr := l.allocScratchSlots(1)
	srcLen := l.allocScratchSlots(1)
	srcPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRSlicePrefix, Imm: shift, Pos: pos})
	checkedDstLen := l.allocScratchSlots(1)
	checkedDstPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: checkedDstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: checkedDstPtr, Pos: pos})

	l.emitCopyLoop(
		srcPtr,
		srcLen,
		checkedDstPtr,
		checkedDstLen,
		loadKind,
		storeKind,
		copyLoopBoundsProofID(name, pos),
		pos,
	)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	return 1, nil
}

func (l *lowerer) emitCopyLoop(
	srcPtr, srcLen, dstPtr, dstLen int,
	loadKind, storeKind ir.IRInstrKind,
	proofID string,
	pos frontend.Position,
) {
	index := l.allocScratchSlots(1)
	value := l.allocScratchSlots(1)
	startLabel := l.newLabel()
	endLabel := l.newLabel()

	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: index, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: index, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: index, Pos: pos})
	if proofID != "" {
		l.emit(ir.IRInstr{Kind: uncheckedIndexLoadKind(loadKind), ProofID: proofID, Pos: pos})
	} else {
		l.emit(ir.IRInstr{Kind: loadKind, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: value, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: index, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: value, Pos: pos})
	l.emit(ir.IRInstr{Kind: storeKind, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: index, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: index, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: pos})
}

func copyBuiltinElement(name string) (string, bool) {
	if name == "core.string_copy" || name == "core.string_copy_into" {
		return "u8", true
	}
	for _, prefix := range []string{"core.slice_copy_into_", "core.slice_copy_"} {
		if strings.HasPrefix(name, prefix) {
			elem := strings.TrimPrefix(name, prefix)
			switch elem {
			case "u8", "u16", "i32", "bool":
				return elem, true
			}
		}
	}
	return "", false
}

func freshCopyBuiltinElement(name string) (string, bool) {
	if name == "core.string_copy_into" || strings.HasPrefix(name, "core.slice_copy_into_") {
		return "", false
	}
	return copyBuiltinElement(name)
}

func copyElementIRKinds(
	elem string,
	types map[string]*semantics.TypeInfo,
) (ir.IRInstrKind, ir.IRInstrKind, ir.IRInstrKind, bool) {
	makeKind := ir.IRMakeSliceI32
	switch elem {
	case "u8":
		makeKind = ir.IRMakeSliceU8
	case "u16":
		makeKind = ir.IRMakeSliceU16
	case "i32", "bool":
		makeKind = ir.IRMakeSliceI32
	default:
		return 0, 0, 0, false
	}
	loadKind, ok := lowerIndexLoadKind(elem, types)
	if !ok {
		return 0, 0, 0, false
	}
	storeKind, ok := lowerIndexStoreKind(elem, types)
	if !ok {
		return 0, 0, 0, false
	}
	return makeKind, loadKind, storeKind, true
}

func copyElementShift(elem string) (int32, bool) {
	switch elem {
	case "u8":
		return 0, true
	case "u16":
		return 1, true
	case "i32", "bool":
		return 2, true
	default:
		return 0, false
	}
}

func staticInvalidCollectionIterable(expr frontend.Expr) bool {
	return staticInvalidAllocationIterable(expr) || staticInvalidStringViewIterable(expr)
}

func staticInvalidAllocationIterable(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	elemSize, ok := allocationElementSizeByBuiltin(name)
	if !ok {
		if target, aliasOK := semantics.ResolveBuiltinAlias(name); aliasOK {
			name = target
			elemSize, ok = allocationElementSizeByBuiltin(name)
		}
	}
	if !ok {
		return false
	}
	lengthArgIndex := 0
	if strings.HasPrefix(name, "core.island_make_") {
		lengthArgIndex = 1
	}
	if lengthArgIndex >= len(call.Args) {
		return false
	}
	length, known := evalConstInt64ForAllocation(call.Args[lengthArgIndex])
	if !known {
		return false
	}
	if length < 0 {
		return true
	}
	return elemSize > 0 && length*int64(elemSize) > 2147483647
}

func staticInvalidStringViewIterable(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	if target, aliasOK := semantics.ResolveBuiltinAlias(name); aliasOK {
		name = target
	}
	if !strings.HasPrefix(name, "core.string_") {
		return false
	}
	sourceLen, knownLen := staticStringByteLen(callArg(call, 0))
	if !knownLen {
		return false
	}
	switch name {
	case "core.string_window":
		if len(call.Args) != 3 {
			return false
		}
		start, startKnown := evalConstInt64ForAllocation(call.Args[1])
		count, countKnown := evalConstInt64ForAllocation(call.Args[2])
		if !startKnown || !countKnown {
			return false
		}
		return start < 0 || count < 0 || start > sourceLen || count > sourceLen-start
	case "core.string_prefix":
		if len(call.Args) != 2 {
			return false
		}
		count, known := evalConstInt64ForAllocation(call.Args[1])
		if !known {
			return false
		}
		return count < 0 || count > sourceLen
	case "core.string_suffix":
		if len(call.Args) != 2 {
			return false
		}
		start, known := evalConstInt64ForAllocation(call.Args[1])
		if !known {
			return false
		}
		return start < 0 || start > sourceLen
	default:
		return false
	}
}

func callArg(call *frontend.CallExpr, index int) frontend.Expr {
	if call == nil || index < 0 || index >= len(call.Args) {
		return nil
	}
	return call.Args[index]
}

func staticStringByteLen(expr frontend.Expr) (int64, bool) {
	lit, ok := expr.(*frontend.StringLitExpr)
	if !ok || lit == nil {
		return 0, false
	}
	return int64(len(lit.Value)), true
}

func allocationElementSizeByBuiltin(name string) (int, bool) {
	return lowerlets.AllocationElementSizeByBuiltin(name)
}

func evalConstInt64ForAllocation(expr frontend.Expr) (int64, bool) {
	switch e := expr.(type) {
	case nil:
		return 0, false
	case *frontend.NumberExpr:
		return int64(e.Value), true
	case *frontend.UnaryExpr:
		v, ok := evalConstInt64ForAllocation(e.X)
		if !ok {
			return 0, false
		}
		if e.Op == frontend.TokenMinus {
			return -v, true
		}
		return 0, false
	case *frontend.BinaryExpr:
		left, ok := evalConstInt64ForAllocation(e.Left)
		if !ok {
			return 0, false
		}
		right, ok := evalConstInt64ForAllocation(e.Right)
		if !ok {
			return 0, false
		}
		switch e.Op {
		case frontend.TokenPlus:
			return left + right, true
		case frontend.TokenMinus:
			return left - right, true
		case frontend.TokenStar:
			return left * right, true
		case frontend.TokenSlash:
			if right == 0 {
				return 0, false
			}
			return left / right, true
		case frontend.TokenPercent:
			if right == 0 {
				return 0, false
			}
			return left % right, true
		default:
			return 0, false
		}
	default:
		return 0, false
	}
}

// ---- lower_rangeproof.go ----

const moduloRangeProofKeyPrefix = "\x00modulo-range:"
const moduloRangeProofFieldSep = "\x00"
const allocationConstLengthKeyPrefix = "\x00allocation-const-length:"
const nonNegativeWhileProofBase = "\x00non-negative"
const constLoopAllocationLengthProofBase = "\x00const-loop-allocation-length"
const constLoopAllocationLengthFieldSep = "\x00"
const nonNegativeConstLoopProofBase = "\x00non-negative-const-loop"
const affineConstExtentProofBase = "\x00affine-const-extent"
const affineConstExtentProofSetBase = "\x00affine-const-extent-set"

type affineConstExtentProof struct {
	baseName   string
	leftName   string
	rightName  string
	stride     int64
	leftUpper  int64
	rightUpper int64
	length     int64
	siteLine   int
	siteCol    int
}

type moduloRangeProof struct {
	baseName    string
	divisorName string
	proofID     string
}

func (l *lowerer) whileRangeProof(stmt *frontend.WhileStmt) (whileRangeProof, bool) {
	if proof, ok := l.whileLenRangeProof(stmt); ok {
		return proof, true
	}
	if proof, ok := l.callBoundaryRangeProof(stmt); ok {
		return proof, true
	}
	if proof, ok := l.constLoopAllocationLengthProof(stmt); ok {
		return proof, true
	}
	if proof, ok := l.nonNegativeWhileRangeProof(stmt); ok {
		return proof, true
	}
	return whileRangeProof{}, false
}

func (l *lowerer) whileLenRangeProof(stmt *frontend.WhileStmt) (whileRangeProof, bool) {
	indexName, baseName, ok := l.whileRangeCondition(stmt.Cond)
	if !ok {
		return whileRangeProof{}, false
	}
	if !l.zeroLocals[indexName] {
		return whileRangeProof{}, false
	}
	if !l.whileBodyHasUnitIncrement(stmt.Body, indexName) {
		return whileRangeProof{}, false
	}
	if l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] {
		return whileRangeProof{}, false
	}
	return whileRangeProof{
		indexName: indexName,
		baseName:  baseName,
		proofID:   whileBoundsProofID(indexName, baseName, stmt.At),
		active:    true,
	}, true
}

func (l *lowerer) callBoundaryRangeProof(stmt *frontend.WhileStmt) (whileRangeProof, bool) {
	indexName, upperName, ok := callBoundaryLoopCondition(stmt.Cond)
	if !ok {
		return whileRangeProof{}, false
	}
	if !l.zeroLocals[indexName] {
		return whileRangeProof{}, false
	}
	if !l.whileBodyHasExactlyOneUnitIncrement(stmt.Body, indexName) {
		return whileRangeProof{}, false
	}
	bases := l.callBoundaryLenProof.BasesForUpper(upperName)
	if len(bases) == 0 {
		return whileRangeProof{}, false
	}
	eligible := map[string]bool{}
	for _, base := range bases {
		if base == "" || l.externalSliceLocals[base] || l.invalidSliceLocals[base] {
			continue
		}
		eligible[base] = true
	}
	if len(eligible) == 0 {
		return whileRangeProof{}, false
	}
	return whileRangeProof{
		indexName:         indexName,
		baseName:          "call-boundary:" + upperName,
		proofID:           rangeBoundsProofID("call-boundary", indexName, "lookup", stmt.At),
		callBoundaryBases: eligible,
		active:            true,
	}, true
}

func callBoundaryLoopCondition(cond frontend.Expr) (string, string, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenLess {
		return "", "", false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil || left.Name == "" {
		return "", "", false
	}
	right, ok := bin.Right.(*frontend.IdentExpr)
	if !ok || right == nil || right.Name == "" {
		return "", "", false
	}
	return left.Name, right.Name, true
}

func (l *lowerer) constLoopAllocationLengthProof(stmt *frontend.WhileStmt) (whileRangeProof, bool) {
	indexName, upper, ok := l.constLoopCondition(stmt.Cond)
	if !ok || upper <= 0 {
		return whileRangeProof{}, false
	}
	if !l.zeroLocals[indexName] {
		return whileRangeProof{}, false
	}
	if !l.whileBodyHasExactlyOneUnitIncrement(stmt.Body, indexName) {
		return whileRangeProof{}, false
	}
	bases := l.constLoopDirectIndexStoreBases(stmt.Body, indexName)
	if len(bases) == 0 {
		return whileRangeProof{}, false
	}
	eligible := make([]string, 0, len(bases))
	for _, baseName := range bases {
		if l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] {
			continue
		}
		length, ok := l.allocationConstLengthForBase(baseName)
		if !ok || length != upper {
			continue
		}
		eligible = append(eligible, baseName)
	}
	if len(eligible) == 0 {
		return whileRangeProof{}, false
	}
	return whileRangeProof{
		indexName: indexName,
		baseName:  encodeConstLoopAllocationLengthProof(upper, eligible),
		proofID:   rangeBoundsProofID("while-const-loop", indexName, "allocation", stmt.At),
		active:    true,
	}, true
}

func (l *lowerer) constLoopCondition(cond frontend.Expr) (string, int64, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenLess {
		return "", 0, false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil || left.Name == "" {
		return "", 0, false
	}
	upper, ok := l.proofConstIntValue(bin.Right)
	if !ok {
		return "", 0, false
	}
	return left.Name, upper, true
}

func (l *lowerer) constLoopDirectIndexStoreBases(stmts []frontend.Stmt, indexName string) []string {
	seen := map[string]bool{}
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		index, ok := assign.Target.(*frontend.IndexExpr)
		if !ok || index == nil {
			continue
		}
		if simpleExprPath(index.Index) != indexName {
			continue
		}
		baseName := simpleExprPath(index.Base)
		if baseName != "" {
			seen[baseName] = true
		}
	}
	out := make([]string, 0, len(seen))
	for baseName := range seen {
		out = append(out, baseName)
	}
	sort.Strings(out)
	return out
}

func (l *lowerer) whileBodyHasExactlyOneUnitIncrement(
	stmts []frontend.Stmt,
	indexName string,
) bool {
	found := false
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if !ok || target.Name != indexName {
			continue
		}
		if found || !l.isUnitIncrementExpr(assign.Value, indexName) {
			return false
		}
		found = true
	}
	return found
}

func (l *lowerer) nonNegativeWhileRangeProof(stmt *frontend.WhileStmt) (whileRangeProof, bool) {
	bin, ok := stmt.Cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenLess {
		return whileRangeProof{}, false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil || left.Name == "" {
		return whileRangeProof{}, false
	}
	if !l.zeroLocals[left.Name] {
		return whileRangeProof{}, false
	}
	upper, ok := l.proofConstIntValue(bin.Right)
	if !ok || upper <= 0 {
		return whileRangeProof{}, false
	}
	if l.whileBodyHasExactlyOneUnitIncrement(stmt.Body, left.Name) {
		if proof, ok := l.affineConstExtentProof(stmt, left.Name, upper); ok {
			return proof, true
		}
		return whileRangeProof{
			indexName: left.Name,
			baseName:  encodeNonNegativeConstLoopProofBase(upper),
			proofID:   rangeBoundsProofID("while-nonnegative", left.Name, "nonnegative", stmt.At),
			active:    true,
		}, true
	}
	if !l.whileBodyOnlyMutatesIndexByUnitIncrement(stmt.Body, left.Name) {
		return whileRangeProof{}, false
	}
	return whileRangeProof{
		indexName: left.Name,
		baseName:  nonNegativeWhileProofBase,
		proofID:   rangeBoundsProofID("while-nonnegative", left.Name, "nonnegative", stmt.At),
		active:    true,
	}, true
}

func (l *lowerer) ifRangeProof(stmt *frontend.IfStmt) (whileRangeProof, bool) {
	indexName, baseName, ok := branchRangeCondition(stmt.Cond)
	if !ok {
		indexName, baseName, ok = whileRangeCondition(stmt.Cond)
		if !ok || !l.zeroLocals[indexName] {
			return whileRangeProof{}, false
		}
	}
	if l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] {
		return whileRangeProof{}, false
	}
	return whileRangeProof{
		indexName: indexName,
		baseName:  baseName,
		proofID:   ifBoundsProofID(indexName, baseName, stmt.At),
		active:    true,
	}, true
}

func (l *lowerer) pushWhileRangeProof(proof whileRangeProof) {
	l.whileRangeProofs = append(l.whileRangeProofs, proof)
}

func (l *lowerer) popWhileRangeProof() {
	l.whileRangeProofs = l.whileRangeProofs[:len(l.whileRangeProofs)-1]
}

func (l *lowerer) invalidateWhileRangeProofForLocal(name string) {
	for i := range l.whileRangeProofs {
		if lowerProofPathMatchesMutation(l.whileRangeProofs[i].indexName, name) ||
			lowerProofPathMatchesMutation(l.whileRangeProofs[i].baseName, name) ||
			callBoundaryProofContainsBase(l.whileRangeProofs[i], name) ||
			constLoopAllocationLengthProofContainsBase(l.whileRangeProofs[i].baseName, name) {
			l.whileRangeProofs[i].active = false
			continue
		}
		if updated, ok := removeAffineConstExtentProofsForLocal(
			l.whileRangeProofs[i].baseName,
			name,
		); ok {
			if updated == "" {
				l.whileRangeProofs[i].active = false
				continue
			}
			l.whileRangeProofs[i].baseName = updated
		}
	}
	l.forgetAllocationConstLengthForBase(name)
	l.forgetModuloRangeProofForLocal(name)
}

func (l *lowerer) invalidateWhileRangeProofsForInoutArgs(args []frontend.Expr, ownership []string) {
	if len(args) == 0 || len(ownership) == 0 {
		return
	}
	for i, owner := range ownership {
		if owner != "inout" {
			continue
		}
		if i >= len(args) {
			break
		}
		path := simpleExprPath(args[i])
		if path == "" {
			continue
		}
		l.invalidateWhileRangeProofForLocal(path)
	}
}

func lowerProofPathMatchesMutation(proofPath string, mutatedPath string) bool {
	return lowerrangeproof.PathMatchesMutation(proofPath, mutatedPath)
}

func (l *lowerer) activeWhileProofForIndex(index *frontend.IndexExpr) (string, bool) {
	baseName := simpleExprPath(index.Base)
	if baseName == "" {
		return "", false
	}
	if proofID, ok := l.activeHelperOffsetProofForIndex(baseName, index); ok {
		return proofID, true
	}
	if proofID, ok := l.activeHelperSummaryProofForIndex(baseName, index); ok {
		return proofID, true
	}
	if proofID, ok := l.activeAllocationLiteralZeroProofForIndex(baseName, index); ok {
		return proofID, true
	}
	if proofID, ok := l.activeAffineConstExtentProofForIndex(baseName, index); ok {
		return proofID, true
	}
	indexName := simpleExprPath(index.Index)
	if indexName != "" {
		for i := len(l.whileRangeProofs) - 1; i >= 0; i-- {
			proof := l.whileRangeProofs[i]
			if proof.active && proof.baseName == baseName && proof.indexName == indexName {
				return proof.proofID, true
			}
			if proof.active && proof.indexName == indexName {
				if proof.callBoundaryBases[baseName] && !l.externalSliceLocals[baseName] &&
					!l.invalidSliceLocals[baseName] {
					return rangeBoundsProofID(
						"call-boundary",
						proof.indexName,
						baseName,
						index.Pos(),
					), true
				}
				if proofID, ok := l.activeConstLoopAllocationLengthProofForIndex(
					proof,
					baseName,
					index.Pos(),
				); ok {
					return proofID, true
				}
				if proofID, ok := l.activeSameConstAllocationLengthProofForIndex(
					proof,
					baseName,
					index.Pos(),
				); ok {
					return proofID, true
				}
			}
		}
		if proofID, ok := l.activeModuloRangeProofForIndex(baseName, indexName); ok {
			return proofID, true
		}
	}
	if proofID, ok := l.activeModuloConstProofForIndex(baseName, index.Index); ok {
		return proofID, true
	}
	return "", false
}

func (l *lowerer) activeHelperOffsetProofForIndex(
	baseName string,
	index *frontend.IndexExpr,
) (string, bool) {
	if baseName == "" || index == nil || l.helperOffsetProof.Empty() ||
		baseName != l.helperOffsetProof.ParamName {
		return "", false
	}
	access, ok := l.helperOffsetProof.AccessForIndex(index.Index, "")
	if !ok {
		return "", false
	}
	return corerangeproof.HelperOffsetBoundsProofID(baseName, access.ActualIndex, index.Pos()), true
}

func (l *lowerer) activeHelperSummaryProofForIndex(
	baseName string,
	index *frontend.IndexExpr,
) (string, bool) {
	if baseName == "" || index == nil || l.helperSummaryProof.Empty() ||
		baseName != l.helperSummaryProof.ParamName {
		return "", false
	}
	indexValue, ok := evalConstInt64ForAllocation(index.Index)
	if !ok || indexValue < 0 {
		return "", false
	}
	if _, ok := l.helperSummaryProof.StoreForIndex(indexValue); !ok ||
		indexValue >= l.helperSummaryProof.Length {
		return "", false
	}
	return corerangeproof.HelperSummaryBoundsProofID(baseName, indexValue, index.Pos()), true
}

func (l *lowerer) activeAllocationLiteralZeroProofForIndex(
	baseName string,
	index *frontend.IndexExpr,
) (string, bool) {
	if baseName == "" || index == nil || !isZeroLiteral(index.Index) {
		return "", false
	}
	info, ok := l.locals[baseName]
	if !ok || !strings.HasPrefix(info.TypeName, "[]") {
		return "", false
	}
	if l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] {
		return "", false
	}
	length, ok := l.allocationConstLengthForBase(baseName)
	if !ok || length <= 0 {
		return "", false
	}
	return allocationLiteralZeroBoundsProofID(baseName, index.Pos()), true
}

func (l *lowerer) activeConstLoopAllocationLengthProofForIndex(
	proof whileRangeProof,
	baseName string,
	pos frontend.Position,
) (string, bool) {
	upper, bases, ok := decodeConstLoopAllocationLengthProof(proof.baseName)
	if !ok || baseName == "" || l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] {
		return "", false
	}
	if !bases[baseName] {
		return "", false
	}
	length, ok := l.allocationConstLengthForBase(baseName)
	if !ok || length != upper {
		return "", false
	}
	return rangeBoundsProofID("while-const", proof.indexName, baseName, pos), true
}

func (l *lowerer) activeSameConstAllocationLengthProofForIndex(
	proof whileRangeProof,
	baseName string,
	pos frontend.Position,
) (string, bool) {
	if baseName == "" || proof.baseName == "" || proof.baseName == baseName {
		return "", false
	}
	if l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] ||
		l.externalSliceLocals[proof.baseName] || l.invalidSliceLocals[proof.baseName] {
		return "", false
	}
	length, ok := l.allocationConstLengthForBase(baseName)
	if !ok {
		return "", false
	}
	proofLength, ok := l.allocationConstLengthForBase(proof.baseName)
	if !ok || proofLength != length {
		return "", false
	}
	return rangeBoundsProofID("while-const", proof.indexName, baseName, pos), true
}

func (l *lowerer) activeModuloConstProofForIndex(
	baseName string,
	expr frontend.Expr,
) (string, bool) {
	if baseName == "" || l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] {
		return "", false
	}
	numerator, divisorValue, ok := l.moduloConstDivisor(expr)
	if !ok || divisorValue <= 0 {
		return "", false
	}
	length, ok := l.allocationConstLengthForBase(baseName)
	if !ok || length != divisorValue {
		return "", false
	}
	if !l.exprKnownNonNegative(numerator) {
		return "", false
	}
	return moduloBoundsProofID(moduloConstProofIndexName(expr), baseName, expr.Pos()), true
}

func (l *lowerer) activeModuloRangeProofForIndex(baseName string, indexName string) (string, bool) {
	if baseName == "" || indexName == "" {
		return "", false
	}
	proof, ok := l.moduloRangeProofForLocal(indexName)
	if !ok || proof.baseName != baseName {
		return "", false
	}
	if l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] {
		return "", false
	}
	if currentBase := l.lenBoundLocals[proof.divisorName]; currentBase != baseName {
		return "", false
	}
	return proof.proofID, true
}

func (l *lowerer) rememberRangeMetadataForLocal(name string, expr frontend.Expr) {
	l.forgetLenBoundsForBase(name)
	l.forgetAllocationConstLengthForBase(name)
	l.forgetModuloRangeProofForLocal(name)
	if value, ok := l.proofConstIntValue(expr); ok {
		l.zeroLocals[name] = value == 0
		l.constIntLocals[name] = value
	} else {
		l.zeroLocals[name] = isZeroLiteral(expr)
		delete(l.constIntLocals, name)
	}
	if base := lenFieldBaseName(expr); base != "" {
		l.lenBoundLocals[name] = base
	} else {
		delete(l.lenBoundLocals, name)
	}
	if lengthName, ok := l.allocationLengthBoundLocal(expr); ok {
		l.lenBoundLocals[lengthName] = name
	}
	if length, ok := l.allocationConstLength(expr); ok {
		l.constIntLocals[allocationConstLengthKey(name)] = length
	}
	l.rememberModuloRangeProofForLocal(name, expr)
	l.externalSliceLocals[name] = l.exprHasExternalSliceProvenance(expr)
	l.invalidSliceLocals[name] = l.exprIsInvalidSliceView(expr)
}

func (l *lowerer) forgetLenBoundsForBase(baseName string) {
	if baseName == "" {
		return
	}
	l.forgetAllocationConstLengthForBase(baseName)
	for name, base := range l.lenBoundLocals {
		if base == baseName {
			delete(l.lenBoundLocals, name)
			continue
		}
		if proof, ok := decodeModuloRangeProof(base); ok && proof.baseName == baseName {
			delete(l.lenBoundLocals, name)
		}
	}
}

func (l *lowerer) rememberModuloRangeProofForLocal(name string, expr frontend.Expr) {
	delete(l.lenBoundLocals, moduloRangeProofKey(name))
	info, ok := l.locals[name]
	if !ok || info.Mutable || info.SlotCount != 1 {
		return
	}
	numerator, divisorName, ok := moduloDivisorName(expr)
	if !ok {
		return
	}
	divisorInfo, ok := l.locals[divisorName]
	if !ok || divisorInfo.Mutable {
		return
	}
	divisorValue, ok := l.proofConstIntValue(&frontend.IdentExpr{Name: divisorName})
	if !ok || divisorValue <= 0 {
		return
	}
	baseName := l.lenBoundLocals[divisorName]
	if baseName == "" || l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] {
		return
	}
	if !l.exprKnownNonNegative(numerator) {
		return
	}
	l.lenBoundLocals[moduloRangeProofKey(name)] = encodeModuloRangeProof(moduloRangeProof{
		baseName:    baseName,
		divisorName: divisorName,
		proofID:     moduloBoundsProofID(name, baseName, expr.Pos()),
	})
}

func (l *lowerer) moduloRangeProofForLocal(name string) (moduloRangeProof, bool) {
	encoded, ok := l.lenBoundLocals[moduloRangeProofKey(name)]
	if !ok {
		return moduloRangeProof{}, false
	}
	return decodeModuloRangeProof(encoded)
}

func (l *lowerer) forgetModuloRangeProofForLocal(name string) {
	if name == "" {
		return
	}
	delete(l.lenBoundLocals, moduloRangeProofKey(name))
	for key, encoded := range l.lenBoundLocals {
		proof, ok := decodeModuloRangeProof(encoded)
		if !ok {
			continue
		}
		if lowerProofPathMatchesMutation(proof.baseName, name) ||
			lowerProofPathMatchesMutation(proof.divisorName, name) {
			delete(l.lenBoundLocals, key)
		}
	}
}

func (l *lowerer) exprKnownNonNegative(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return e != nil && e.Value >= 0
	case *frontend.IdentExpr:
		if e == nil || e.Name == "" {
			return false
		}
		if value, ok := l.proofConstIntValue(e); ok {
			return value >= 0
		}
		return l.activeNonNegativeWhileProofForLocal(e.Name)
	case *frontend.BinaryExpr:
		if e == nil {
			return false
		}
		switch e.Op {
		case frontend.TokenPlus, frontend.TokenStar:
			return l.exprKnownNonNegative(e.Left) && l.exprKnownNonNegative(e.Right)
		default:
			return false
		}
	default:
		return false
	}
}

func (l *lowerer) activeNonNegativeWhileProofForLocal(name string) bool {
	for i := len(l.whileRangeProofs) - 1; i >= 0; i-- {
		proof := l.whileRangeProofs[i]
		if proof.active && isNonNegativeWhileProofBase(proof.baseName) && proof.indexName == name {
			return true
		}
		if proof.active && proof.indexName == name {
			if _, _, ok := decodeConstLoopAllocationLengthProof(proof.baseName); ok {
				return true
			}
		}
		if proof.active && affineConstExtentProofContainsLocal(proof.baseName, name) {
			return true
		}
	}
	return false
}

func callBoundaryProofContainsBase(proof whileRangeProof, baseName string) bool {
	if baseName == "" || len(proof.callBoundaryBases) == 0 {
		return false
	}
	return proof.callBoundaryBases[baseName]
}

func (l *lowerer) affineConstExtentProof(
	stmt *frontend.WhileStmt,
	rightName string,
	rightUpper int64,
) (whileRangeProof, bool) {
	candidates := l.affineConstExtentCandidates(stmt, rightName, rightUpper)
	if len(candidates) == 0 {
		return whileRangeProof{}, false
	}
	return whileRangeProof{
		indexName: rightName,
		baseName:  encodeAffineConstExtentProofs(candidates),
		proofID:   affineConstExtentBoundsProofID(candidates[0]),
		active:    true,
	}, true
}

func (l *lowerer) affineConstExtentCandidates(
	stmt *frontend.WhileStmt,
	rightName string,
	rightUpper int64,
) []affineConstExtentProof {
	if rightUpper != 3 {
		return nil
	}
	var candidates []affineConstExtentProof
	switch rightName {
	case "col":
		if candidate, ok := l.affineConstExtentStoreCandidate(stmt.Body, rightName, rightUpper); ok {
			candidates = append(candidates, candidate)
		}
		if candidate, ok := l.affineConstExtentBLoadCandidate(stmt.Body, rightName, rightUpper); ok {
			candidates = append(candidates, candidate)
		}
	case "k":
		if candidate, ok := l.affineConstExtentLoadCandidate(stmt.Body, rightName, rightUpper); ok {
			candidates = append(candidates, candidate)
		}
	default:
		return nil
	}
	return candidates
}

func (l *lowerer) affineConstExtentStoreCandidate(
	stmts []frontend.Stmt,
	rightName string,
	rightUpper int64,
) (affineConstExtentProof, bool) {
	var out affineConstExtentProof
	found := false
	for idx, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		index, ok := assign.Target.(*frontend.IndexExpr)
		if !ok || index == nil {
			continue
		}
		baseName := simpleExprPath(index.Base)
		if baseName != "c" {
			continue
		}
		leftName, stride, matchedRight, ok := affineConstExtentIndexParts(index.Index)
		if !ok || leftName != "row" || matchedRight != rightName || stride != 3 || rightUpper != 3 {
			continue
		}
		leftUpper, ok := l.activeExactConstLoopUpperForLocal(leftName)
		if !ok || leftUpper != 3 {
			return affineConstExtentProof{}, false
		}
		length, ok := l.allocationConstLengthForBase(baseName)
		if !ok || length != 9 || leftUpper*stride != length || rightUpper != stride {
			return affineConstExtentProof{}, false
		}
		if affineConstExtentPrefixInvalidated(stmts[:idx], baseName, leftName, rightName) {
			return affineConstExtentProof{}, false
		}
		if found {
			return affineConstExtentProof{}, false
		}
		out = affineConstExtentProof{
			baseName:   baseName,
			leftName:   leftName,
			rightName:  rightName,
			stride:     stride,
			leftUpper:  leftUpper,
			rightUpper: rightUpper,
			length:     length,
			siteLine:   index.Pos().Line,
			siteCol:    index.Pos().Col,
		}
		found = true
	}
	return out, found
}

func (l *lowerer) affineConstExtentLoadCandidate(
	stmts []frontend.Stmt,
	rightName string,
	rightUpper int64,
) (affineConstExtentProof, bool) {
	var out affineConstExtentProof
	found := false
	for idx, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		loads := affineConstExtentLoadIndexes(assign.Value)
		for _, index := range loads {
			baseName := simpleExprPath(index.Base)
			if baseName != "a" {
				continue
			}
			leftName, stride, matchedRight, ok := affineConstExtentIndexParts(index.Index)
			if !ok || leftName != "row" || matchedRight != rightName || stride != 3 ||
				rightUpper != 3 {
				continue
			}
			leftUpper, ok := l.activeExactConstLoopUpperForLocal(leftName)
			if !ok || leftUpper != 3 {
				return affineConstExtentProof{}, false
			}
			length, ok := l.allocationConstLengthForBase(baseName)
			if !ok || length != 9 || leftUpper*stride != length || rightUpper != stride {
				return affineConstExtentProof{}, false
			}
			if affineConstExtentPrefixInvalidated(stmts[:idx], baseName, leftName, rightName) {
				return affineConstExtentProof{}, false
			}
			if found {
				return affineConstExtentProof{}, false
			}
			out = affineConstExtentProof{
				baseName:   baseName,
				leftName:   leftName,
				rightName:  rightName,
				stride:     stride,
				leftUpper:  leftUpper,
				rightUpper: rightUpper,
				length:     length,
				siteLine:   index.Pos().Line,
				siteCol:    index.Pos().Col,
			}
			found = true
		}
	}
	return out, found
}

func (l *lowerer) affineConstExtentBLoadCandidate(
	stmts []frontend.Stmt,
	rightName string,
	rightUpper int64,
) (affineConstExtentProof, bool) {
	if rightName != "col" || rightUpper != 3 {
		return affineConstExtentProof{}, false
	}
	var out affineConstExtentProof
	found := false
	for outerIdx, stmt := range stmts {
		nested, ok := stmt.(*frontend.WhileStmt)
		if !ok || nested == nil {
			continue
		}
		leftName, leftUpper, ok := l.constLoopCondition(nested.Cond)
		if !ok || leftName != "k" || leftUpper != 3 {
			continue
		}
		if !affineConstExtentPrefixDeclaresZeroLocal(stmts[:outerIdx], leftName) {
			return affineConstExtentProof{}, false
		}
		if !l.whileBodyHasExactlyOneUnitIncrement(nested.Body, leftName) {
			return affineConstExtentProof{}, false
		}
		for innerIdx, nestedStmt := range nested.Body {
			assign, ok := nestedStmt.(*frontend.AssignStmt)
			if !ok || assign == nil {
				continue
			}
			for _, index := range affineConstExtentLoadIndexes(assign.Value) {
				baseName := simpleExprPath(index.Base)
				if baseName != "b" {
					continue
				}
				matchedLeft, stride, matchedRight, ok := affineConstExtentIndexParts(index.Index)
				if !ok || matchedLeft != leftName || matchedRight != rightName || stride != 3 {
					continue
				}
				length, ok := l.allocationConstLengthForBase(baseName)
				if !ok || length != 9 || leftUpper*stride != length || rightUpper != stride {
					return affineConstExtentProof{}, false
				}
				if affineConstExtentPrefixInvalidated(
					stmts[:outerIdx],
					baseName,
					leftName,
					rightName,
				) ||
					affineConstExtentPrefixInvalidated(
						nested.Body[:innerIdx],
						baseName,
						leftName,
						rightName,
					) {
					return affineConstExtentProof{}, false
				}
				if found {
					return affineConstExtentProof{}, false
				}
				out = affineConstExtentProof{
					baseName:   baseName,
					leftName:   leftName,
					rightName:  rightName,
					stride:     stride,
					leftUpper:  leftUpper,
					rightUpper: rightUpper,
					length:     length,
					siteLine:   index.Pos().Line,
					siteCol:    index.Pos().Col,
				}
				found = true
			}
		}
	}
	return out, found
}

func affineConstExtentPrefixDeclaresZeroLocal(stmts []frontend.Stmt, name string) bool {
	for _, stmt := range stmts {
		let, ok := stmt.(*frontend.LetStmt)
		if !ok || let == nil || let.Name != name {
			continue
		}
		return isZeroLiteral(let.Value)
	}
	return false
}

func (l *lowerer) activeExactConstLoopUpperForLocal(name string) (int64, bool) {
	for i := len(l.whileRangeProofs) - 1; i >= 0; i-- {
		proof := l.whileRangeProofs[i]
		if !proof.active || proof.indexName != name {
			continue
		}
		if upper, ok := decodeNonNegativeConstLoopProofBase(proof.baseName); ok {
			return upper, true
		}
	}
	return 0, false
}

func (l *lowerer) activeAffineConstExtentProofForIndex(
	baseName string,
	index *frontend.IndexExpr,
) (string, bool) {
	if baseName == "" || index == nil || l.externalSliceLocals[baseName] ||
		l.invalidSliceLocals[baseName] {
		return "", false
	}
	leftName, stride, rightName, ok := affineConstExtentIndexParts(index.Index)
	if !ok {
		return "", false
	}
	for i := len(l.whileRangeProofs) - 1; i >= 0; i-- {
		proof := l.whileRangeProofs[i]
		if !proof.active {
			continue
		}
		affines, ok := decodeAffineConstExtentProofs(proof.baseName)
		if !ok {
			continue
		}
		for _, affine := range affines {
			if affine.baseName != baseName ||
				affine.leftName != leftName ||
				affine.rightName != rightName ||
				affine.stride != stride ||
				affine.siteLine != index.Pos().Line ||
				affine.siteCol != index.Pos().Col {
				continue
			}
			length, ok := l.allocationConstLengthForBase(baseName)
			if !ok || length != affine.length {
				return "", false
			}
			return affineConstExtentBoundsProofID(affine), true
		}
	}
	return "", false
}

func affineConstExtentIndexParts(expr frontend.Expr) (string, int64, string, bool) {
	add, ok := expr.(*frontend.BinaryExpr)
	if !ok || add == nil || add.Op != frontend.TokenPlus {
		return "", 0, "", false
	}
	mul, ok := add.Left.(*frontend.BinaryExpr)
	if !ok || mul == nil || mul.Op != frontend.TokenStar {
		return "", 0, "", false
	}
	left, ok := mul.Left.(*frontend.IdentExpr)
	if !ok || left == nil || left.Name == "" {
		return "", 0, "", false
	}
	strideExpr, ok := mul.Right.(*frontend.NumberExpr)
	if !ok || strideExpr == nil || strideExpr.Value <= 0 {
		return "", 0, "", false
	}
	right, ok := add.Right.(*frontend.IdentExpr)
	if !ok || right == nil || right.Name == "" {
		return "", 0, "", false
	}
	return left.Name, int64(strideExpr.Value), right.Name, true
}

func affineConstExtentLoadIndexes(expr frontend.Expr) []*frontend.IndexExpr {
	var out []*frontend.IndexExpr
	var walk func(frontend.Expr)
	walk = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.IndexExpr:
			if e != nil {
				out = append(out, e)
			}
		case *frontend.BinaryExpr:
			if e != nil {
				walk(e.Left)
				walk(e.Right)
			}
		case *frontend.UnaryExpr:
			if e != nil {
				walk(e.X)
			}
		case *frontend.CallExpr:
			if e != nil {
				for _, arg := range e.Args {
					walk(arg)
				}
			}
		case *frontend.FieldAccessExpr:
			if e != nil {
				walk(e.Base)
			}
		}
	}
	walk(expr)
	return out
}

func affineConstExtentPrefixInvalidated(stmts []frontend.Stmt, names ...string) bool {
	seen := map[string]bool{}
	for _, name := range names {
		if name != "" {
			seen[name] = true
		}
	}
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if ok && target != nil && seen[target.Name] {
			return true
		}
	}
	return false
}

func encodeConstLoopAllocationLengthProof(upper int64, bases []string) string {
	return constLoopAllocationLengthProofBase + constLoopAllocationLengthFieldSep +
		strconv.FormatInt(upper, 10) + constLoopAllocationLengthFieldSep +
		strings.Join(bases, constLoopAllocationLengthFieldSep)
}

func decodeConstLoopAllocationLengthProof(encoded string) (int64, map[string]bool, bool) {
	if !strings.HasPrefix(
		encoded,
		constLoopAllocationLengthProofBase+constLoopAllocationLengthFieldSep,
	) {
		return 0, nil, false
	}
	rest := strings.TrimPrefix(
		encoded,
		constLoopAllocationLengthProofBase+constLoopAllocationLengthFieldSep,
	)
	upperText, basesText, ok := strings.Cut(rest, constLoopAllocationLengthFieldSep)
	if !ok {
		return 0, nil, false
	}
	upper, err := strconv.ParseInt(upperText, 10, 64)
	if err != nil || upper <= 0 {
		return 0, nil, false
	}
	bases := map[string]bool{}
	for _, base := range strings.Split(basesText, constLoopAllocationLengthFieldSep) {
		if base != "" {
			bases[base] = true
		}
	}
	if len(bases) == 0 {
		return 0, nil, false
	}
	return upper, bases, true
}

func constLoopAllocationLengthProofContainsBase(encoded string, baseName string) bool {
	if baseName == "" {
		return false
	}
	_, bases, ok := decodeConstLoopAllocationLengthProof(encoded)
	return ok && bases[baseName]
}

func encodeNonNegativeConstLoopProofBase(upper int64) string {
	return nonNegativeConstLoopProofBase + constLoopAllocationLengthFieldSep + strconv.FormatInt(
		upper,
		10,
	)
}

func decodeNonNegativeConstLoopProofBase(encoded string) (int64, bool) {
	prefix := nonNegativeConstLoopProofBase + constLoopAllocationLengthFieldSep
	if !strings.HasPrefix(encoded, prefix) {
		return 0, false
	}
	upper, err := strconv.ParseInt(strings.TrimPrefix(encoded, prefix), 10, 64)
	if err != nil || upper <= 0 {
		return 0, false
	}
	return upper, true
}

func isNonNegativeWhileProofBase(encoded string) bool {
	if encoded == nonNegativeWhileProofBase {
		return true
	}
	_, ok := decodeNonNegativeConstLoopProofBase(encoded)
	return ok
}

func encodeAffineConstExtentProof(proof affineConstExtentProof) string {
	return affineConstExtentProofBase + constLoopAllocationLengthFieldSep + strings.Join([]string{
		proof.baseName,
		proof.leftName,
		proof.rightName,
		strconv.FormatInt(proof.stride, 10),
		strconv.FormatInt(proof.leftUpper, 10),
		strconv.FormatInt(proof.rightUpper, 10),
		strconv.FormatInt(proof.length, 10),
		strconv.Itoa(proof.siteLine),
		strconv.Itoa(proof.siteCol),
	}, constLoopAllocationLengthFieldSep)
}

func encodeAffineConstExtentProofs(proofs []affineConstExtentProof) string {
	if len(proofs) == 1 {
		return encodeAffineConstExtentProof(proofs[0])
	}
	parts := []string{strconv.Itoa(len(proofs))}
	for _, proof := range proofs {
		parts = append(parts,
			proof.baseName,
			proof.leftName,
			proof.rightName,
			strconv.FormatInt(proof.stride, 10),
			strconv.FormatInt(proof.leftUpper, 10),
			strconv.FormatInt(proof.rightUpper, 10),
			strconv.FormatInt(proof.length, 10),
			strconv.Itoa(proof.siteLine),
			strconv.Itoa(proof.siteCol),
		)
	}
	return affineConstExtentProofSetBase + constLoopAllocationLengthFieldSep + strings.Join(
		parts,
		constLoopAllocationLengthFieldSep,
	)
}

func decodeAffineConstExtentProof(encoded string) (affineConstExtentProof, bool) {
	prefix := affineConstExtentProofBase + constLoopAllocationLengthFieldSep
	if !strings.HasPrefix(encoded, prefix) {
		return affineConstExtentProof{}, false
	}
	parts := strings.Split(strings.TrimPrefix(encoded, prefix), constLoopAllocationLengthFieldSep)
	if len(parts) != 9 {
		return affineConstExtentProof{}, false
	}
	stride, ok := parsePositiveInt64(parts[3])
	if !ok {
		return affineConstExtentProof{}, false
	}
	leftUpper, ok := parsePositiveInt64(parts[4])
	if !ok {
		return affineConstExtentProof{}, false
	}
	rightUpper, ok := parsePositiveInt64(parts[5])
	if !ok {
		return affineConstExtentProof{}, false
	}
	length, ok := parsePositiveInt64(parts[6])
	if !ok {
		return affineConstExtentProof{}, false
	}
	siteLine, err := strconv.Atoi(parts[7])
	if err != nil || siteLine <= 0 {
		return affineConstExtentProof{}, false
	}
	siteCol, err := strconv.Atoi(parts[8])
	if err != nil || siteCol <= 0 {
		return affineConstExtentProof{}, false
	}
	proof := affineConstExtentProof{
		baseName:   parts[0],
		leftName:   parts[1],
		rightName:  parts[2],
		stride:     stride,
		leftUpper:  leftUpper,
		rightUpper: rightUpper,
		length:     length,
		siteLine:   siteLine,
		siteCol:    siteCol,
	}
	if proof.baseName == "" || proof.leftName == "" || proof.rightName == "" {
		return affineConstExtentProof{}, false
	}
	return proof, true
}

func decodeAffineConstExtentProofs(encoded string) ([]affineConstExtentProof, bool) {
	if proof, ok := decodeAffineConstExtentProof(encoded); ok {
		return []affineConstExtentProof{proof}, true
	}
	prefix := affineConstExtentProofSetBase + constLoopAllocationLengthFieldSep
	if !strings.HasPrefix(encoded, prefix) {
		return nil, false
	}
	parts := strings.Split(strings.TrimPrefix(encoded, prefix), constLoopAllocationLengthFieldSep)
	if len(parts) < 1 {
		return nil, false
	}
	count, err := strconv.Atoi(parts[0])
	if err != nil || count <= 0 || len(parts) != 1+count*9 {
		return nil, false
	}
	proofs := make([]affineConstExtentProof, 0, count)
	for i := 0; i < count; i++ {
		offset := 1 + i*9
		stride, ok := parsePositiveInt64(parts[offset+3])
		if !ok {
			return nil, false
		}
		leftUpper, ok := parsePositiveInt64(parts[offset+4])
		if !ok {
			return nil, false
		}
		rightUpper, ok := parsePositiveInt64(parts[offset+5])
		if !ok {
			return nil, false
		}
		length, ok := parsePositiveInt64(parts[offset+6])
		if !ok {
			return nil, false
		}
		siteLine, err := strconv.Atoi(parts[offset+7])
		if err != nil || siteLine <= 0 {
			return nil, false
		}
		siteCol, err := strconv.Atoi(parts[offset+8])
		if err != nil || siteCol <= 0 {
			return nil, false
		}
		proof := affineConstExtentProof{
			baseName:   parts[offset],
			leftName:   parts[offset+1],
			rightName:  parts[offset+2],
			stride:     stride,
			leftUpper:  leftUpper,
			rightUpper: rightUpper,
			length:     length,
			siteLine:   siteLine,
			siteCol:    siteCol,
		}
		if proof.baseName == "" || proof.leftName == "" || proof.rightName == "" {
			return nil, false
		}
		proofs = append(proofs, proof)
	}
	return proofs, true
}

func parsePositiveInt64(text string) (int64, bool) {
	value, err := strconv.ParseInt(text, 10, 64)
	return value, err == nil && value > 0
}

func affineConstExtentProofContainsLocal(encoded string, name string) bool {
	if name == "" {
		return false
	}
	proofs, ok := decodeAffineConstExtentProofs(encoded)
	if !ok {
		return false
	}
	for _, proof := range proofs {
		if proof.baseName == name || proof.leftName == name || proof.rightName == name {
			return true
		}
	}
	return false
}

func removeAffineConstExtentProofsForLocal(encoded string, name string) (string, bool) {
	if name == "" {
		return encoded, false
	}
	proofs, ok := decodeAffineConstExtentProofs(encoded)
	if !ok {
		return encoded, false
	}
	filtered := proofs[:0]
	for _, proof := range proofs {
		if proof.baseName == name || proof.leftName == name || proof.rightName == name {
			continue
		}
		filtered = append(filtered, proof)
	}
	if len(filtered) == 0 {
		return "", true
	}
	return encodeAffineConstExtentProofs(filtered), true
}

func affineConstExtentBoundsProofID(proof affineConstExtentProof) string {
	return rangeBoundsProofID(
		"affine-const",
		proof.leftName+"_"+proof.rightName,
		proof.baseName,
		frontend.Position{Line: proof.siteLine, Col: proof.siteCol},
	)
}

func moduloRangeProofKey(name string) string {
	return moduloRangeProofKeyPrefix + name
}

func encodeModuloRangeProof(proof moduloRangeProof) string {
	return proof.baseName + moduloRangeProofFieldSep + proof.divisorName + moduloRangeProofFieldSep + proof.proofID
}

func decodeModuloRangeProof(encoded string) (moduloRangeProof, bool) {
	baseName, rest, ok := strings.Cut(encoded, moduloRangeProofFieldSep)
	if !ok {
		return moduloRangeProof{}, false
	}
	divisorName, proofID, ok := strings.Cut(rest, moduloRangeProofFieldSep)
	if !ok || baseName == "" || divisorName == "" || proofID == "" {
		return moduloRangeProof{}, false
	}
	return moduloRangeProof{baseName: baseName, divisorName: divisorName, proofID: proofID}, true
}

func (l *lowerer) allocationLengthBoundLocal(expr frontend.Expr) (string, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil || len(call.Args) != 1 {
		return "", false
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	switch name {
	case "core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool":
	default:
		return "", false
	}
	id, ok := call.Args[0].(*frontend.IdentExpr)
	if !ok || id == nil || id.Name == "" {
		return "", false
	}
	info, ok := l.locals[id.Name]
	if !ok || info.Mutable {
		return "", false
	}
	return id.Name, true
}

func (l *lowerer) allocationConstLength(expr frontend.Expr) (int64, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return 0, false
	}
	lengthExpr, ok := allocationConstLengthExpr(call)
	if !ok {
		return 0, false
	}
	length, ok := l.proofConstIntValue(lengthExpr)
	if !ok || length < 0 {
		return 0, false
	}
	return length, true
}

func (l *lowerer) allocationConstLengthForBase(baseName string) (int64, bool) {
	if baseName == "" {
		return 0, false
	}
	length, ok := l.constIntLocals[allocationConstLengthKey(baseName)]
	return length, ok
}

func (l *lowerer) forgetAllocationConstLengthForBase(baseName string) {
	if baseName == "" {
		return
	}
	delete(l.constIntLocals, allocationConstLengthKey(baseName))
}

func allocationConstLengthKey(baseName string) string {
	return allocationConstLengthKeyPrefix + baseName
}

func isMakeSliceCallName(name string) bool {
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	switch name {
	case "core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool",
		"core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool":
		return true
	default:
		return false
	}
}

func allocationConstLengthExpr(call *frontend.CallExpr) (frontend.Expr, bool) {
	if call == nil {
		return nil, false
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	switch name {
	case "core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool":
		if len(call.Args) != 1 {
			return nil, false
		}
		return call.Args[0], true
	case "core.island_make_u8",
		"core.island_make_u16",
		"core.island_make_i32",
		"core.island_make_bool":
		if len(call.Args) < 2 {
			return nil, false
		}
		return call.Args[1], true
	default:
		return nil, false
	}
}

type rangeMetadataState struct {
	zero     map[string]bool
	constInt map[string]int64
	lenBound map[string]string
	external map[string]bool
	invalid  map[string]bool
}

func (l *lowerer) snapshotRangeMetadata() rangeMetadataState {
	return rangeMetadataState{
		zero:     cloneLowerBoolMap(l.zeroLocals),
		constInt: cloneLowerInt64Map(l.constIntLocals),
		lenBound: cloneLowerStringMap(l.lenBoundLocals),
		external: cloneLowerBoolMap(l.externalSliceLocals),
		invalid:  cloneLowerBoolMap(l.invalidSliceLocals),
	}
}

func (l *lowerer) restoreRangeMetadata(state rangeMetadataState) {
	l.zeroLocals = cloneLowerBoolMap(state.zero)
	l.constIntLocals = cloneLowerInt64Map(state.constInt)
	l.lenBoundLocals = cloneLowerStringMap(state.lenBound)
	l.externalSliceLocals = cloneLowerBoolMap(state.external)
	l.invalidSliceLocals = cloneLowerBoolMap(state.invalid)
}

func (l *lowerer) mergeRangeMetadata(thenState rangeMetadataState, elseState rangeMetadataState) {
	keys := map[string]bool{}
	for key := range thenState.zero {
		keys[key] = true
	}
	for key := range elseState.zero {
		keys[key] = true
	}
	for key := range thenState.constInt {
		keys[key] = true
	}
	for key := range elseState.constInt {
		keys[key] = true
	}
	for key := range thenState.lenBound {
		keys[key] = true
	}
	for key := range elseState.lenBound {
		keys[key] = true
	}
	for key := range thenState.external {
		keys[key] = true
	}
	for key := range elseState.external {
		keys[key] = true
	}
	for key := range thenState.invalid {
		keys[key] = true
	}
	for key := range elseState.invalid {
		keys[key] = true
	}
	for key := range keys {
		l.zeroLocals[key] = thenState.zero[key] && elseState.zero[key]
		if thenValue, thenOK := thenState.constInt[key]; thenOK {
			if elseValue, elseOK := elseState.constInt[key]; elseOK && thenValue == elseValue {
				l.constIntLocals[key] = thenValue
			} else {
				delete(l.constIntLocals, key)
			}
		} else {
			delete(l.constIntLocals, key)
		}
		if thenValue, thenOK := thenState.lenBound[key]; thenOK {
			if elseValue, elseOK := elseState.lenBound[key]; elseOK && thenValue == elseValue {
				l.lenBoundLocals[key] = thenValue
			} else {
				delete(l.lenBoundLocals, key)
			}
		} else {
			delete(l.lenBoundLocals, key)
		}
		l.externalSliceLocals[key] = thenState.external[key] || elseState.external[key]
		l.invalidSliceLocals[key] = thenState.invalid[key] || elseState.invalid[key]
	}
}

func cloneLowerBoolMap(in map[string]bool) map[string]bool {
	return lowerrangeproof.CloneBoolMap(in)
}

func cloneLowerInt64Map(in map[string]int64) map[string]int64 {
	return lowerrangeproof.CloneInt64Map(in)
}

func cloneLowerStringMap(in map[string]string) map[string]string {
	return lowerrangeproof.CloneStringMap(in)
}

func (l *lowerer) whileRangeCondition(cond frontend.Expr) (string, string, bool) {
	indexName, baseName, _, ok := l.rangeFromCondition(cond)
	return indexName, baseName, ok
}

func (l *lowerer) rangeFromCondition(
	cond frontend.Expr,
) (string, string, corerangeproof.Range, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return "", "", corerangeproof.Range{}, false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil {
		return "", "", corerangeproof.Range{}, false
	}
	switch bin.Op {
	case frontend.TokenLess, frontend.TokenBangEq:
		base := l.lenBoundBaseName(bin.Right)
		if base == "" {
			return "", "", corerangeproof.Range{}, false
		}
		return left.Name, base, corerangeproof.LessThanLen(left.Name, base), true
	case frontend.TokenLessEq:
		base := lenMinusOneBaseName(bin.Right)
		if base == "" {
			return "", "", corerangeproof.Range{}, false
		}
		return left.Name, base, corerangeproof.LessEqualLenMinusOne(left.Name, base), true
	default:
		return "", "", corerangeproof.Range{}, false
	}
}

func staticRangeFromCondition(cond frontend.Expr) (string, string, corerangeproof.Range, bool) {
	return lowerrangeproof.StaticRangeFromCondition(cond)
}

func staticRangeCondition(cond frontend.Expr) (string, string, bool) {
	return lowerrangeproof.StaticRangeCondition(cond)
}

func whileRangeCondition(cond frontend.Expr) (string, string, bool) {
	return lowerrangeproof.WhileRangeCondition(cond)
}

func staticWhileRangeCondition(cond frontend.Expr) (string, string, bool) {
	return lowerrangeproof.StaticWhileRangeCondition(cond)
}

func branchRangeCondition(cond frontend.Expr) (string, string, bool) {
	return lowerrangeproof.BranchRangeCondition(cond)
}

func (l *lowerer) lenBoundBaseName(expr frontend.Expr) string {
	if base := lenFieldBaseName(expr); base != "" {
		return base
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return ""
	}
	return l.lenBoundLocals[id.Name]
}

func branchRangeConditionParts(lower frontend.Expr, upper frontend.Expr) (string, string, bool) {
	return lowerrangeproof.BranchRangeConditionParts(lower, upper)
}

func moduloDivisorName(expr frontend.Expr) (frontend.Expr, string, bool) {
	return lowerrangeproof.ModuloDivisorName(expr)
}

func (l *lowerer) moduloConstDivisor(expr frontend.Expr) (frontend.Expr, int64, bool) {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenPercent {
		return nil, 0, false
	}
	switch divisor := bin.Right.(type) {
	case *frontend.NumberExpr:
		if divisor == nil {
			return nil, 0, false
		}
		return bin.Left, int64(divisor.Value), true
	case *frontend.IdentExpr:
		if divisor == nil || divisor.Name == "" {
			return nil, 0, false
		}
		value, ok := l.proofConstIntValue(divisor)
		if !ok {
			return nil, 0, false
		}
		return bin.Left, value, true
	default:
		return nil, 0, false
	}
}

func moduloConstProofIndexName(expr frontend.Expr) string {
	return "modulo_const"
}

func nonNegativeGuardIndex(expr frontend.Expr) (string, bool) {
	return lowerrangeproof.NonNegativeGuardIndex(expr)
}

func isZeroNumber(expr frontend.Expr) bool {
	return lowerrangeproof.IsZeroNumber(expr)
}

func lenFieldBaseName(expr frontend.Expr) string {
	return lowerrangeproof.LenFieldBaseName(expr)
}

func lenMinusOneBaseName(expr frontend.Expr) string {
	return lowerrangeproof.LenMinusOneBaseName(expr)
}

func (l *lowerer) whileBodyHasUnitIncrement(stmts []frontend.Stmt, indexName string) bool {
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if !ok || target.Name != indexName {
			continue
		}
		if l.isUnitIncrementExpr(assign.Value, indexName) {
			return true
		}
	}
	return false
}

func (l *lowerer) whileBodyOnlyMutatesIndexByUnitIncrement(
	stmts []frontend.Stmt,
	indexName string,
) bool {
	found := false
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if !ok || target.Name != indexName {
			continue
		}
		if !l.isUnitIncrementExpr(assign.Value, indexName) {
			return false
		}
		found = true
	}
	return found
}

func (l *lowerer) isUnitIncrementExpr(expr frontend.Expr, indexName string) bool {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenPlus {
		return false
	}
	if left, ok := bin.Left.(*frontend.IdentExpr); ok && left.Name == indexName {
		return l.isUnitStepExpr(bin.Right)
	}
	if right, ok := bin.Right.(*frontend.IdentExpr); ok && right.Name == indexName {
		return l.isUnitStepExpr(bin.Left)
	}
	return false
}

func (l *lowerer) isUnitStepExpr(expr frontend.Expr) bool {
	if num, ok := expr.(*frontend.NumberExpr); ok && num != nil {
		return num.Value == 1
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return false
	}
	info, ok := l.locals[id.Name]
	if !ok || info.Mutable {
		return false
	}
	value, ok := l.constIntLocals[id.Name]
	return ok && value == 1
}

func (l *lowerer) proofConstIntValue(expr frontend.Expr) (int64, bool) {
	if value, ok := evalConstInt64ForAllocation(expr); ok {
		return value, true
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return 0, false
	}
	info, ok := l.locals[id.Name]
	if !ok || info.Mutable {
		return 0, false
	}
	value, ok := l.constIntLocals[id.Name]
	return value, ok
}

func isZeroLiteral(expr frontend.Expr) bool {
	return lowerrangeproof.IsZeroLiteral(expr)
}

func (l *lowerer) collectionIterableProofAllowed(expr frontend.Expr) bool {
	if expr == nil {
		return false
	}
	return !l.exprHasExternalSliceProvenance(expr) && !l.exprIsInvalidSliceView(expr)
}

func isRawSliceConstructor(expr frontend.Expr) bool {
	return lowerrangeproof.IsRawSliceConstructor(expr)
}

func rawSliceElementShift(name string) int32 {
	return lowerrangeproof.RawSliceElementShift(name)
}

func (l *lowerer) exprHasExternalSliceProvenance(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return l.externalSliceLocals[e.Name]
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = target
		}
		if isRawSliceConstructor(&frontend.CallExpr{Name: name}) {
			return true
		}
		if isSliceCopyBuiltinName(name) {
			return false
		}
		if isBorrowOrViewBuiltinName(name) {
			return len(e.Args) == 0 || l.exprHasExternalSliceProvenance(e.Args[0])
		}
	}
	return false
}

func (l *lowerer) exprIsInvalidSliceView(expr frontend.Expr) bool {
	if staticInvalidCollectionIterable(expr) {
		return true
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return l.invalidSliceLocals[e.Name]
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = target
		}
		if isSliceCopyBuiltinName(name) {
			return false
		}
		if isBorrowOrViewBuiltinName(name) {
			return len(e.Args) > 0 && l.exprIsInvalidSliceView(e.Args[0])
		}
	}
	return false
}

func isSliceCopyBuiltinName(name string) bool {
	return lowerrangeproof.IsSliceCopyBuiltinName(name)
}

func isBorrowOrViewBuiltinName(name string) bool {
	return lowerrangeproof.IsBorrowOrViewBuiltinName(name)
}

func simpleExprPath(expr frontend.Expr) string {
	return lowerrangeproof.SimpleExprPath(expr)
}

func whileBoundsProofID(indexName string, baseName string, pos frontend.Position) string {
	return lowerrangeproof.WhileBoundsProofID(indexName, baseName, pos)
}

func ifBoundsProofID(indexName string, baseName string, pos frontend.Position) string {
	return lowerrangeproof.IfBoundsProofID(indexName, baseName, pos)
}

func moduloBoundsProofID(indexName string, baseName string, pos frontend.Position) string {
	return lowerrangeproof.ModuloBoundsProofID(indexName, baseName, pos)
}

func rangeBoundsProofID(
	kind string,
	indexName string,
	baseName string,
	pos frontend.Position,
) string {
	return lowerrangeproof.RangeBoundsProofID(kind, indexName, baseName, pos)
}

func allocationLiteralZeroBoundsProofID(baseName string, pos frontend.Position) string {
	return rangeBoundsProofID("allocation-zero", "literal0", baseName, pos)
}

func copyLoopBoundsProofID(name string, pos frontend.Position) string {
	return lowerrangeproof.CopyLoopBoundsProofID(name, pos)
}

func forCollectionBoundsProofID(stmt *frontend.ForRangeStmt) string {
	return lowerrangeproof.ForCollectionBoundsProofID(stmt)
}

func isViewCollectionIterable(expr frontend.Expr) bool {
	return lowerrangeproof.IsViewCollectionIterable(expr)
}
