package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

func checkExprWithEffects(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, int, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", regionNone, nil
	case *frontend.BoolLitExpr:
		return "bool", regionNone, nil
	case *frontend.NoneLitExpr:
		return "none", regionNone, nil
	case *frontend.StringLitExpr:
		return "str", regionNone, nil
	case *frontend.MatchExpr:
		resultType, err := checkMatchExpr(e, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		return resultType, regionNone, nil
	case *frontend.CatchExpr:
		resultType, err := checkCatchExpr(e, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		return resultType, regionNone, nil
	case *frontend.IdentExpr:
		if info, ok := locals[e.Name]; ok && info.ActorField {
			return info.TypeName, regionNone, nil
		}
		if err := state.checkNotConsumed(e.Name, e.At); err != nil {
			return "", regionNone, err
		}
		if err := state.checkResourceNotFinalized(e.Name, e.At); err != nil {
			return "", regionNone, err
		}
		if state.resourceUnknown(e.Name) {
			return "", regionNone, fmt.Errorf("%s: ambiguous resource provenance for '%s' after control-flow merge", frontend.FormatPos(e.At), e.Name)
		}
		if err := checkLocalScope(e.Name, state, e.At); err != nil {
			return "", regionNone, err
		}
		info, ok := locals[e.Name]
		if !ok {
			if g, ok := globals[e.Name]; ok {
				if analysis != nil && g.Mutable {
					analysis.touchesMutableGlobals = true
				}
				if err := checkResourceTreeUsable(e.Name, g.TypeName, types, state, e.At); err != nil {
					return "", regionNone, err
				}
				return g.TypeName, regionNone, nil
			}
			if state != nil {
				if field, ok := state.actorStateFields[e.Name]; ok {
					return field.TypeName, regionNone, nil
				}
			}
			return "", regionNone, fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(e.At), e.Name)
		}
		if info.GenericFunctionValue {
			return "", regionNone, fmt.Errorf("%s: generic closure '%s' cannot be used as a pointer value; only let-bound direct local calls with inferable concrete arguments are supported in this MVP", frontend.FormatPos(e.At), e.Name)
		}
		if info.FunctionTypeValue && info.FunctionValue != "" {
			return "", regionNone, fmt.Errorf("%s: function value '%s' cannot escape as a first-class value in this MVP; only direct local calls are supported", frontend.FormatPos(e.At), e.Name)
		}
		if len(info.FunctionCaptures) > 0 {
			return "", regionNone, fmt.Errorf("%s: capturing closure '%s' cannot be used as a pointer value; only direct local calls are supported in this MVP", frontend.FormatPos(e.At), e.Name)
		}
		if err := checkResourceTreeUsable(e.Name, info.TypeName, types, state, e.At); err != nil {
			return "", regionNone, err
		}
		if regionID, ok := state.regionVars[e.Name]; ok {
			if regionID == regionUnknown {
				if state.unknownVars[e.Name] {
					if conflict, ok := state.unknownConflicts[e.Name]; ok {
						return "", regionNone, fmt.Errorf(
							"%s: ambiguous region for '%s' after control-flow merge (%s: %s, %s: %s); hint: assign to a fresh variable in each branch and use it after the merge",
							frontend.FormatPos(e.At),
							e.Name,
							conflict.leftLabel,
							formatRegionID(state, conflict.leftRegion),
							conflict.rightLabel,
							formatRegionID(state, conflict.rightRegion),
						)
					}
					return "", regionNone, fmt.Errorf(
						"%s: ambiguous region for '%s' after control-flow merge; hint: reassign it to a single region before use",
						frontend.FormatPos(e.At),
						e.Name,
					)
				}
				return "", regionNone, fmt.Errorf("%s: ambiguous region for '%s'", frontend.FormatPos(e.At), e.Name)
			}
			if !state.isScopeActive(regionID) {
				return "", regionNone, fmt.Errorf("%s: slice from scoped island is out of scope", frontend.FormatPos(e.At))
			}
			return info.TypeName, regionID, nil
		}
		return info.TypeName, regionNone, nil
	case *frontend.FieldAccessExpr:
		if typeName, _, ok, err := resolveEnumCaseExpr(e, locals, globals, types, module, imports); ok || err != nil {
			if err != nil {
				return "", regionNone, err
			}
			return typeName, regionNone, nil
		}
		targetInfo, targetType, err := ResolveFieldAccessType(e, locals, globals, types)
		if err != nil {
			return "", regionNone, err
		}
		baseType, baseRegion, err := checkExprWithEffects(e.Base, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		if baseType == "" {
			return "", regionNone, fmt.Errorf("%s: invalid field access base", frontend.FormatPos(e.At))
		}
		if !targetInfo.Global {
			if err := checkLocalScope(targetInfo.Name, state, e.At); err != nil {
				return "", regionNone, err
			}
		}
		if isResourceHandleType(targetType) {
			source, err := resourceSourceForExpr(e, funcs, module, imports, state)
			if err != nil {
				return "", regionNone, err
			}
			if source.ambiguous {
				return "", regionNone, fmt.Errorf("%s: resource expression mixes resource provenance", frontend.FormatPos(e.At))
			}
			path, _ := resourcePathForExpr(e)
			if source.unknown {
				return "", regionNone, fmt.Errorf("%s: ambiguous resource provenance for '%s' after control-flow merge", frontend.FormatPos(e.At), path)
			}
			if source.known {
				if err := state.checkNotConsumed(source.name, e.At); err != nil {
					return "", regionNone, err
				}
				if err := state.checkResourceNotFinalized(source.name, e.At); err != nil {
					return "", regionNone, err
				}
			}
		}
		if typeMayContainRegion(targetType, types) && baseRegion != regionNone {
			return targetType, baseRegion, nil
		}
		return targetType, regionNone, nil
	case *frontend.IndexExpr:
		baseType, _, err := checkExprWithEffects(e.Base, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		indexType, _, err := checkExprWithEffects(e.Index, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		if !isInt32Like(indexType) {
			return "", regionNone, fmt.Errorf("%s: index must be i32/u8", frontend.FormatPos(e.At))
		}
		info, err := ensureTypeInfo(baseType, types)
		if err != nil {
			return "", regionNone, err
		}
		switch info.Kind {
		case TypeStr:
			return "u8", regionNone, nil
		case TypeSlice:
			return info.ElemType, regionNone, nil
		case TypeArray:
			return info.ElemType, regionNone, nil
		default:
			return "", regionNone, fmt.Errorf("%s: cannot index '%s'", frontend.FormatPos(e.At), baseType)
		}
	case *frontend.CallExpr:
		if enumType, caseInfo, ok, err := resolveEnumCaseConstructorCall(e, types, module, imports); ok || err != nil {
			if err != nil {
				return "", regionNone, err
			}
			if len(e.ArgLabels) > 0 {
				for _, label := range e.ArgLabels {
					if label != "" {
						return "", regionNone, fmt.Errorf("%s: enum case payload arguments do not use labels", frontend.FormatPos(e.At))
					}
				}
			}
			if len(caseInfo.PayloadTypes) == 0 {
				return "", regionNone, fmt.Errorf("%s: enum case '%s.%s' has no payload; use '%s.%s'", frontend.FormatPos(e.At), displayTypeName(enumType, module), caseInfo.Name, displayTypeName(enumType, module), caseInfo.Name)
			}
			if len(e.Args) != len(caseInfo.PayloadTypes) {
				return "", regionNone, fmt.Errorf("%s: enum case '%s.%s' expects %d payload argument(s), got %d", frontend.FormatPos(e.At), displayTypeName(enumType, module), caseInfo.Name, len(caseInfo.PayloadTypes), len(e.Args))
			}
			payloadRegion := regionNone
			for i, arg := range e.Args {
				argType, argRegion, err := checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return "", regionNone, err
				}
				if !typesCompatibleWithNullPtr(caseInfo.PayloadTypes[i], argType, arg) {
					return "", regionNone, fmt.Errorf("%s: enum case '%s.%s' payload %d expects '%s', got '%s'", frontend.FormatPos(arg.Pos()), displayTypeName(enumType, module), caseInfo.Name, i+1, caseInfo.PayloadTypes[i], argType)
				}
				consumeIslandSourceLocals(arg, caseInfo.PayloadTypes[i], locals, types, module, imports, state)
				payloadRegion = joinRegion(payloadRegion, argRegion)
				if payloadRegion == regionUnknown {
					return "", regionNone, fmt.Errorf("%s: enum case '%s.%s' mixes values from different regions", frontend.FormatPos(arg.Pos()), displayTypeName(enumType, module), caseInfo.Name)
				}
			}
			e.ResolvedType = enumType
			return enumType, payloadRegion, nil
		}
		return checkCallExprWithEffects(e, locals, globals, funcs, types, module, imports, state, effects, analysis)
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			if len(e.Decl.TypeParams) > 0 {
				if name, pos, ok := firstCapture(collectClosureCaptures(e.Decl, locals)); ok {
					return "", regionNone, fmt.Errorf("%s: generic closure literals do not support captures in this MVP (captured '%s')", frontend.FormatPos(pos), name)
				}
				return "ptr", regionNone, nil
			}
			if name, pos, ok := firstCapture(collectClosureCaptures(e.Decl, locals)); ok {
				return "", regionNone, fmt.Errorf("%s: capturing closure literal captures '%s' but is not let-bound; %s", frontend.FormatPos(pos), name, closureLiteralDirectCallCaptureText())
			}
		}
		return "ptr", regionNone, nil
	case *frontend.TryExpr:
		if state.throwType == "" {
			return "", regionNone, fmt.Errorf("%s: try is only allowed in throwing functions", frontend.FormatPos(e.At))
		}
		call, ok := e.X.(*frontend.CallExpr)
		isTryAwait := false
		if !ok {
			if await, awaitOK := e.X.(*frontend.AwaitExpr); awaitOK {
				if !state.async {
					return "", regionNone, fmt.Errorf("%s: await is only allowed in async functions", frontend.FormatPos(await.At))
				}
				call, ok = await.X.(*frontend.CallExpr)
				isTryAwait = ok
			}
		}
		if !ok {
			return "", regionNone, fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		state.allowThrowDepth++
		state.allowThrowCall = call
		if isTryAwait {
			state.allowAwaitDepth++
			state.allowAwaitCall = call
		}
		tname, regionID, err := checkCallExprWithEffects(call, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if isTryAwait {
			state.allowAwaitDepth--
			state.allowAwaitCall = nil
		}
		state.allowThrowDepth--
		state.allowThrowCall = nil
		return tname, regionID, err
	case *frontend.AwaitExpr:
		if !state.async {
			return "", regionNone, fmt.Errorf("%s: await is only allowed in async functions", frontend.FormatPos(e.At))
		}
		if tryExpr, ok := e.X.(*frontend.TryExpr); ok {
			if call, callOK := tryExpr.X.(*frontend.CallExpr); callOK {
				return "", regionNone, fmt.Errorf("%s: use 'try await %s()' for async typed-error propagation", frontend.FormatPos(e.At), call.Name)
			}
			return "", regionNone, fmt.Errorf("%s: use 'try await <call>()' for async typed-error propagation", frontend.FormatPos(e.At))
		}
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return "", regionNone, fmt.Errorf("%s: await expects an async function call", frontend.FormatPos(e.At))
		}
		state.allowAwaitDepth++
		state.allowAwaitCall = call
		tname, regionID, err := checkCallExprWithEffects(call, locals, globals, funcs, types, module, imports, state, effects, analysis)
		state.allowAwaitDepth--
		state.allowAwaitCall = nil
		return tname, regionID, err
	case *frontend.StructLitExpr:
		resolved, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return "", regionNone, err
		}
		e.Type.Name = resolved
		info, err := ensureTypeInfo(resolved, types)
		if err != nil {
			return "", regionNone, fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		if info.Kind != TypeStruct {
			return "", regionNone, fmt.Errorf("%s: '%s' is not a struct", frontend.FormatPos(e.At), resolved)
		}
		seen := make(map[string]frontend.StructFieldInit, len(e.Fields))
		for _, field := range e.Fields {
			if _, exists := info.FieldMap[field.Name]; !exists {
				return "", regionNone, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(field.At), field.Name)
			}
			if _, exists := seen[field.Name]; exists {
				return "", regionNone, fmt.Errorf("%s: duplicate field '%s'", frontend.FormatPos(field.At), field.Name)
			}
			seen[field.Name] = field
		}
		structRegion := regionNone
		for _, field := range info.Fields {
			init, ok := seen[field.Name]
			if !ok {
				return "", regionNone, fmt.Errorf("%s: missing field '%s'", frontend.FormatPos(e.At), field.Name)
			}
			valType, valRegion, err := checkExprWithEffects(init.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return "", regionNone, err
			}
			if !typesCompatibleWithNullPtr(field.TypeName, valType, init.Value) {
				return "", regionNone, fmt.Errorf("%s: type mismatch for field '%s'", frontend.FormatPos(init.At), field.Name)
			}
			consumeIslandSourceLocals(init.Value, field.TypeName, locals, types, module, imports, state)
			structRegion = joinRegion(structRegion, valRegion)
			if structRegion == regionUnknown {
				return "", regionNone, fmt.Errorf("%s: struct literal mixes values from different regions", frontend.FormatPos(init.At))
			}
		}
		return resolved, structRegion, nil
	case *frontend.UnaryExpr:
		xtype, _, err := checkExprWithEffects(e.X, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		switch e.Op {
		case frontend.TokenMinus:
			if !isInt32Like(xtype) {
				return "", regionNone, fmt.Errorf("%s: unary '-' expects i32/u8", frontend.FormatPos(e.At))
			}
			return "i32", regionNone, nil
		case frontend.TokenBang:
			if !isConditionType(xtype) {
				return "", regionNone, fmt.Errorf("%s: unary '!' expects bool or i32/u8", frontend.FormatPos(e.At))
			}
			return "bool", regionNone, nil
		default:
			return "", regionNone, fmt.Errorf("%s: unsupported unary operator", frontend.FormatPos(e.At))
		}
	case *frontend.BinaryExpr:
		ltype, _, err := checkExprWithEffects(e.Left, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		rtype, _, err := checkExprWithEffects(e.Right, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		switch e.Op {
		case frontend.TokenPlus, frontend.TokenMinus, frontend.TokenStar, frontend.TokenSlash, frontend.TokenPercent:
			if !isInt32Like(ltype) || !isInt32Like(rtype) {
				return "", regionNone, fmt.Errorf("%s: arithmetic operators require i32/u8", frontend.FormatPos(e.At))
			}
			return "i32", regionNone, nil
		case frontend.TokenLess, frontend.TokenGreater, frontend.TokenGreaterEq, frontend.TokenLessEq:
			if !isInt32Like(ltype) || !isInt32Like(rtype) {
				return "", regionNone, fmt.Errorf("%s: relational operators require i32/u8", frontend.FormatPos(e.At))
			}
			return "bool", regionNone, nil
		case frontend.TokenEqEq, frontend.TokenBangEq:
			if !comparableTypes(ltype, rtype, types) {
				return "", regionNone, fmt.Errorf("%s: cannot compare '%s' and '%s'", frontend.FormatPos(e.At), ltype, rtype)
			}
			return "bool", regionNone, nil
		case frontend.TokenAmpAmp, frontend.TokenPipePipe:
			if ltype != "bool" || rtype != "bool" {
				return "", regionNone, fmt.Errorf("%s: logical operators require bool", frontend.FormatPos(e.At))
			}
			return "bool", regionNone, nil
		default:
			return "", regionNone, fmt.Errorf("%s: unsupported binary operator", frontend.FormatPos(e.At))
		}
	default:
		return "", regionNone, fmt.Errorf("%s: unsupported expression", frontend.FormatPos(expr.Pos()))
	}
}

func checkMatchExpr(
	e *frontend.MatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, error) {
	scrutType, _, err := checkExprWithEffects(e.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
	if err != nil {
		return "", err
	}
	scrutInfo, scrutInfoOK := types[scrutType]
	if !isInt32Like(scrutType) {
		if !scrutInfoOK || (scrutInfo.Kind != TypeEnum && scrutInfo.Kind != TypeOptional) {
			return "", fmt.Errorf("%s: match value must be enum or i32/u8", frontend.FormatPos(e.At))
		}
	}
	if e.ResultType == "" {
		resultType, err := inferMatchExprType(e, locals, globals, funcs, types, module, imports)
		if err != nil {
			return "", fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		e.ResultType = resultType
	}
	if err := reportBarePayloadMatchExprPatterns(e, scrutType, locals, globals, funcs, types, module, imports); err != nil {
		return "", err
	}
	if !matchExprHasCompleteOptionalPatterns(e) && !matchExprHasCompleteEnumPatterns(e, locals, globals, funcs, types, module, imports) {
		hasDefault := false
		for _, c := range e.Cases {
			if c.Default {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			return "", fmt.Errorf("%s: match expression must be exhaustive", frontend.FormatPos(e.At))
		}
	}
	scrutineeResourcePath := e.ScrutineeLocal
	if scrutineeResourcePath != "" {
		if err := bindResourceTreeFromExpr(scrutineeResourcePath, scrutType, e.Value, funcs, types, module, imports, state); err != nil {
			return "", err
		}
	} else if path, ok := resourcePathForExpr(e.Value); ok {
		scrutineeResourcePath = path
	}
	seenDefault := false
	seenPatterns := map[string]frontend.Position{}
	caseScopes := state.matchExprScopes[e]
	beforeVars := copyRegionVars(state.regionVars)
	beforeFlow := snapshotFlow(state)
	var mergedVars map[string]int
	var mergedFlow flowSnapshot
	mergedSet := false
	for i, c := range e.Cases {
		if seenDefault {
			return "", fmt.Errorf("%s: match default must be last", frontend.FormatPos(c.At))
		}
		state.regionVars = copyRegionVars(beforeVars)
		restoreFlow(state, beforeFlow)
		if c.Default {
			seenDefault = true
		} else {
			patType := ""
			if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				if !scrutInfoOK || scrutInfo.Kind != TypeOptional {
					return "", fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(some.At))
				}
				patType = optionalSomePatternType
			} else if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
				caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
				if err != nil {
					return "", err
				}
				if !found {
					return "", fmt.Errorf("%s: unknown enum pattern '%s.%s'", frontend.FormatPos(enumPat.At), enumPat.TypeName, enumPat.CaseName)
				}
				if err := validateEnumCasePatternPayload(enumPat, caseType, caseInfo, module); err != nil {
					return "", err
				}
				patType = caseType
			} else {
				var err error
				patType, _, err = checkExprWithEffects(c.Pattern, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return "", err
				}
			}
			if scrutInfoOK && scrutInfo.Kind == TypeOptional && patType != "none" && patType != optionalSomePatternType {
				return "", fmt.Errorf("%s: optional match supports only 'none', 'some(name)', and '_' patterns", frontend.FormatPos(c.At))
			}
			if !matchPatternCompatible(scrutType, patType, types) {
				return "", fmt.Errorf("%s: match pattern type mismatch: expected '%s', got '%s'", frontend.FormatPos(c.At), scrutType, patType)
			}
			if c.Guard == nil {
				if key := matchPatternKey(c.Pattern, patType); key != "" {
					if first, exists := seenPatterns[key]; exists {
						return "", fmt.Errorf("%s: duplicate match pattern (first at %s)", frontend.FormatPos(c.At), frontend.FormatPos(first))
					}
					seenPatterns[key] = c.At
				}
			}
		}
		caseScopeID := regionNone
		if i < len(caseScopes) {
			caseScopeID = caseScopes[i]
		}
		err := withActiveScope(state, caseScopeID, func() error {
			if err := bindPatternResourceLocals(c.Pattern, "", scrutineeResourcePath, scrutType, types, module, imports, state); err != nil {
				return err
			}
			if c.Guard != nil {
				guardType, _, err := checkExprWithEffects(c.Guard, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
				if guardType != "bool" {
					return fmt.Errorf("%s: match guard must be Bool", frontend.FormatPos(c.Guard.Pos()))
				}
			}
			armType, _, err := checkExprWithEffects(c.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(e.ResultType, armType, c.Value) {
				return fmt.Errorf("%s: match expression case type mismatch: expected '%s', got '%s'", frontend.FormatPos(c.At), e.ResultType, armType)
			}
			if e.ResultLocal != "" {
				if err := bindResourceTreeFromExpr(e.ResultLocal, e.ResultType, c.Value, funcs, types, module, imports, state); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		caseVars := copyRegionVars(state.regionVars)
		caseFlow := snapshotFlow(state)
		if !mergedSet {
			mergedVars = caseVars
			mergedFlow = caseFlow
			mergedSet = true
		} else {
			state.regionVars = mergeRegionVars(mergedVars, caseVars)
			mergeFlow(state, mergedFlow, caseFlow)
			mergedVars = copyRegionVars(state.regionVars)
			mergedFlow = snapshotFlow(state)
		}
	}
	if mergedSet {
		state.regionVars = mergedVars
		restoreFlow(state, mergedFlow)
	}
	return e.ResultType, nil
}

func checkCatchExpr(
	e *frontend.CatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, error) {
	call, ok := e.Call.(*frontend.CallExpr)
	if !ok {
		return "", fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
	}
	state.allowCatchDepth++
	state.allowCatchCall = call
	successType, _, err := checkCallExprWithEffects(call, locals, globals, funcs, types, module, imports, state, effects, analysis)
	state.allowCatchDepth--
	state.allowCatchCall = nil
	if err != nil {
		return "", err
	}
	if call.Name == "core.task_join_i32_typed" || call.Name == "core.task_join_group_i32_typed" {
		if len(call.TypeArgs) != 1 || call.TypeArgs[0].Name == "" {
			return "", fmt.Errorf("%s: task_join_i32_typed missing resolved error type", frontend.FormatPos(call.At))
		}
		e.ErrorType = call.TypeArgs[0].Name
	} else {
		sig, ok := funcs[call.Name]
		if !ok || sig.ThrowsType == "" {
			return "", fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
		}
		e.ErrorType = sig.ThrowsType
	}
	if e.ResultType == "" {
		e.ResultType = successType
	}
	if err := reportBarePayloadCatchExprPatterns(e, locals, globals, funcs, types, module, imports); err != nil {
		return "", err
	}
	if !catchExprHasCompleteOptionalPatterns(e, e.ErrorType, types) && !catchExprHasCompleteEnumPatterns(e, e.ErrorType, locals, globals, funcs, types, module, imports) {
		hasDefault := false
		for _, c := range e.Cases {
			if c.Default {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			return "", fmt.Errorf("%s: catch expression must be exhaustive", frontend.FormatPos(e.At))
		}
	}
	seenDefault := false
	seenPatterns := map[string]frontend.Position{}
	caseScopes := state.catchExprScopes[e]
	beforeVars := copyRegionVars(state.regionVars)
	beforeFlow := snapshotFlow(state)
	mergedVars := copyRegionVars(beforeVars)
	mergedFlow := beforeFlow
	for i, c := range e.Cases {
		if seenDefault {
			return "", fmt.Errorf("%s: catch default must be last", frontend.FormatPos(c.At))
		}
		state.regionVars = copyRegionVars(beforeVars)
		restoreFlow(state, beforeFlow)
		if c.Default {
			seenDefault = true
		} else {
			patType, err := catchPatternType(c.Pattern, e.ErrorType, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return "", err
			}
			if !matchPatternCompatible(e.ErrorType, patType, types) {
				return "", fmt.Errorf("%s: catch pattern type mismatch: expected '%s', got '%s'", frontend.FormatPos(c.At), e.ErrorType, patType)
			}
			if c.Guard == nil {
				if key := matchPatternKey(c.Pattern, patType); key != "" {
					if first, exists := seenPatterns[key]; exists {
						return "", fmt.Errorf("%s: duplicate catch pattern (first at %s)", frontend.FormatPos(c.At), frontend.FormatPos(first))
					}
					seenPatterns[key] = c.At
				}
			}
		}
		caseScopeID := regionNone
		if i < len(caseScopes) {
			caseScopeID = caseScopes[i]
		}
		err := withActiveScope(state, caseScopeID, func() error {
			if c.Guard != nil {
				guardType, _, err := checkExprWithEffects(c.Guard, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
				if guardType != "bool" {
					return fmt.Errorf("%s: catch guard must be Bool", frontend.FormatPos(c.Guard.Pos()))
				}
			}
			armType, _, err := checkExprWithEffects(c.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(e.ResultType, armType, c.Value) {
				return fmt.Errorf("%s: catch expression case type mismatch: expected '%s', got '%s'", frontend.FormatPos(c.At), e.ResultType, armType)
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		caseVars := copyRegionVars(state.regionVars)
		caseFlow := snapshotFlow(state)
		state.regionVars = mergeRegionVars(mergedVars, caseVars)
		mergeFlow(state, mergedFlow, caseFlow)
		mergedVars = copyRegionVars(state.regionVars)
		mergedFlow = snapshotFlow(state)
	}
	state.regionVars = mergedVars
	restoreFlow(state, mergedFlow)
	return e.ResultType, nil
}

func checkCallExprWithEffects(
	e *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, int, error) {
	callerSig, hasCallerSig := currentCallerSignature(effects, funcs)
	resolved := ""
	isBuiltin := false
	if local, ok := locals[e.Name]; ok {
		if local.FunctionValue == "" {
			if !local.FunctionTypeValue {
				return "", regionNone, fmt.Errorf("%s: function value '%s' is not callable in this MVP; only local closure literals are supported", frontend.FormatPos(e.At), e.Name)
			}
			if len(e.TypeArgs) > 0 {
				return "", regionNone, fmt.Errorf("%s: explicit type arguments are not supported for function-typed callback values in this MVP", frontend.FormatPos(e.At))
			}
			if len(e.Args) != len(local.FunctionParamTypes) {
				return "", regionNone, fmt.Errorf("%s: wrong argument count for callback '%s'", frontend.FormatPos(e.At), e.Name)
			}
			if len(e.ArgLabels) > 0 {
				return "", regionNone, fmt.Errorf("%s: argument labels are not supported for callback '%s' in this MVP", frontend.FormatPos(e.At), e.Name)
			}
			for i, arg := range e.Args {
				argType, _, err := checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return "", regionNone, err
				}
				if !typesCompatibleWithNullPtr(local.FunctionParamTypes[i], argType, arg) {
					return "", regionNone, fmt.Errorf("%s: type mismatch for callback '%s' arg %d", frontend.FormatPos(arg.Pos()), e.Name, i+1)
				}
			}
			if err := effects.requireAll(e.At, local.FunctionEffects); err != nil {
				return "", regionNone, err
			}
			if hasCallerSig && hasStrictSemanticCallClauses(callerSig) {
				return "", regionNone, fmt.Errorf(
					"%s: function-typed callback '%s' has unknown target and cannot be called under semantic clause '%s' in this MVP",
					frontend.FormatPos(e.At),
					e.Name,
					firstStrictSemanticCallClause(callerSig),
				)
			}
			return local.FunctionReturnType, regionNone, nil
		}
		if local.GenericFunctionValue {
			return "", regionNone, fmt.Errorf("%s: generic closure '%s' is only supported for let-bound direct local calls with inferable concrete arguments in this MVP", frontend.FormatPos(e.At), e.Name)
		}
		if err := appendClosureCaptureArgs(e, local); err != nil {
			return "", regionNone, err
		}
		resolved = local.FunctionValue
		e.Name = resolved
	} else if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
		resolved = builtin
		isBuiltin = true
	} else if _, ok := funcs[e.Name]; ok {
		resolved = e.Name
	} else {
		var err error
		resolved, err = resolveCallName(e.Name, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
	}
	if resolved == "core.send_typed" || resolved == "core.recv_typed" {
		return checkTypedActorBuiltin(e, resolved, locals, globals, funcs, types, module, imports, state, effects, analysis)
	}
	if resolved == "core.task_spawn_i32_typed" || resolved == "core.task_spawn_group_i32_typed" ||
		resolved == "core.task_join_i32_typed" || resolved == "core.task_join_group_i32_typed" {
		return checkTypedTaskBuiltin(e, resolved, locals, globals, funcs, types, module, imports, state, effects, analysis)
	}
	sig, ok := funcs[resolved]
	if !ok {
		if ctorType, ctorRegion, handled, err := checkStructConstructorCallWithEffects(e, locals, globals, funcs, types, module, imports, state, effects, analysis); handled {
			return ctorType, ctorRegion, err
		}
		return "", regionNone, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), resolved)
	}
	if err := ensureFuncVisible(resolved, sig, module, e.At); err != nil {
		return "", regionNone, err
	}
	if sig.Generic {
		return "", regionNone, fmt.Errorf("%s: generic function '%s' could not be monomorphized; use inferable value arguments", frontend.FormatPos(e.At), e.Name)
	}
	if len(e.TypeArgs) > 0 {
		return "", regionNone, fmt.Errorf("%s: explicit type arguments are only supported for recv_typed", frontend.FormatPos(e.At))
	}
	if hasCallerSig {
		if err := validateCallAgainstSemanticClauses(callerSig, sig, resolved, e.At); err != nil {
			return "", regionNone, err
		}
	}
	if analysis != nil && !isBuiltin && sig.TouchesMutableGlobals {
		analysis.touchesMutableGlobals = true
	}
	isTryCall := state != nil && state.allowThrowDepth > 0 && state.allowThrowCall == e
	isCatchCall := state != nil && state.allowCatchDepth > 0 && state.allowCatchCall == e
	if sig.ThrowsType != "" {
		if !isTryCall && !isCatchCall {
			return "", regionNone, fmt.Errorf("%s: call to throwing function '%s' requires try", frontend.FormatPos(e.At), resolved)
		}
		if isTryCall && state.throwType == "" {
			return "", regionNone, fmt.Errorf("%s: try is only allowed in throwing functions", frontend.FormatPos(e.At))
		}
		if isTryCall && !typesCompatibleWithNullPtr(state.throwType, sig.ThrowsType, e) {
			return "", regionNone, fmt.Errorf("%s: thrown error type mismatch: expected '%s', got '%s'", frontend.FormatPos(e.At), state.throwType, sig.ThrowsType)
		}
	} else if isTryCall {
		return "", regionNone, fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
	} else if isCatchCall {
		return "", regionNone, fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
	}
	isAwaitCall := state != nil && state.allowAwaitDepth > 0 && state.allowAwaitCall == e
	if sig.Async {
		if !isAwaitCall {
			return "", regionNone, fmt.Errorf("%s: call to async function '%s' requires await", frontend.FormatPos(e.At), resolved)
		}
		if !state.async {
			return "", regionNone, fmt.Errorf("%s: await is only allowed in async functions", frontend.FormatPos(e.At))
		}
	} else if isAwaitCall {
		return "", regionNone, fmt.Errorf("%s: await expects an async function call", frontend.FormatPos(e.At))
	}
	if (resolved == "core.actor_dispatch" || resolved == "core.actor_main_entry_id") && !strings.HasPrefix(module, "__") {
		return "", regionNone, fmt.Errorf("%s: '%s' is reserved for internal runtime modules", frontend.FormatPos(e.At), resolved)
	}
	if len(e.Args) != len(sig.ParamTypes) {
		return "", regionNone, fmt.Errorf("%s: wrong argument count for '%s'", frontend.FormatPos(e.At), resolved)
	}
	if len(e.ArgLabels) > 0 {
		if len(e.ArgLabels) != len(e.Args) {
			return "", regionNone, fmt.Errorf("%s: internal error: call argument labels are inconsistent", frontend.FormatPos(e.At))
		}
		if len(sig.ParamNames) != len(e.Args) {
			return "", regionNone, fmt.Errorf("%s: argument labels are not supported for '%s'", frontend.FormatPos(e.At), resolved)
		}
		for i, label := range e.ArgLabels {
			if label == "" {
				return "", regionNone, fmt.Errorf("%s: cannot mix labeled and unlabeled arguments in call to '%s'", frontend.FormatPos(e.Args[i].Pos()), resolved)
			}
			if sig.ParamNames[i] == "" || label != sig.ParamNames[i] {
				return "", regionNone, fmt.Errorf("%s: argument label mismatch for '%s': expected '%s', got '%s'", frontend.FormatPos(e.Args[i].Pos()), resolved, sig.ParamNames[i], label)
			}
		}
	}
	argRegions := make([]int, len(e.Args))
	consumeArgs := make([]string, len(e.Args))
	consumeArgTypes := make([]string, len(e.Args))
	consumeArgPositions := make(map[string]frontend.Position)
	borrowArgs := make(map[string]frontend.Position)
	inoutArgs := make(map[string]frontend.Position)
	for i, arg := range e.Args {
		argType := ""
		argRegion := regionNone
		callbackParam := i < len(sig.ParamFunctionTypes) && sig.ParamFunctionTypes[i]
		if callbackParam {
			id, ok := arg.(*frontend.IdentExpr)
			if !ok {
				return "", regionNone, fmt.Errorf("%s: callback argument for '%s' must be a symbol-backed local function value or direct named function/closure symbol in this MVP", frontend.FormatPos(arg.Pos()), resolved)
			}
			callbackType, callbackSymbol, err := resolveCallbackArgumentType(id, resolved, sig, i, locals, funcs, module, imports, hasCallerSig && hasStrictSemanticCallClauses(callerSig))
			if err != nil {
				return "", regionNone, err
			}
			if hasCallerSig {
				if callbackSymbol == "" {
					if hasStrictSemanticCallClauses(callerSig) {
						return "", regionNone, fmt.Errorf(
							"%s: callback argument for '%s' has unknown target and is not allowed under semantic clause '%s' in this MVP",
							frontend.FormatPos(arg.Pos()),
							resolved,
							firstStrictSemanticCallClause(callerSig),
						)
					}
				} else if callbackSig, ok := funcs[callbackSymbol]; ok {
					if err := validateCallAgainstSemanticClauses(callerSig, callbackSig, callbackSymbol, arg.Pos()); err != nil {
						return "", regionNone, err
					}
				}
			}
			if callbackSymbol != "" {
				if callbackSig, ok := funcs[callbackSymbol]; ok {
					if err := effects.requireAll(arg.Pos(), callbackSig.Effects); err != nil {
						return "", regionNone, err
					}
				}
				// Keep lowered target collection deterministic across modules/import aliases.
				id.Name = callbackSymbol
			}
			argType = callbackType
		} else {
			var err error
			argType, argRegion, err = checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return "", regionNone, err
			}
		}
		if !typesCompatibleWithNullPtr(sig.ParamTypes[i], argType, arg) {
			return "", regionNone, fmt.Errorf("%s: type mismatch for '%s' arg %d", frontend.FormatPos(arg.Pos()), resolved, i+1)
		}
		if err := checkResourceCallArg(resolved, sig.ParamTypes[i], arg, funcs, module, imports, state); err != nil {
			return "", regionNone, err
		}
		paramOwnership := ""
		if i < len(sig.ParamOwnership) {
			paramOwnership = sig.ParamOwnership[i]
		}
		if paramOwnership == "" {
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return "", regionNone, fmt.Errorf("%s: borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of '%s'", frontend.FormatPos(arg.Pos()), borrowedName, i+1, resolved)
			}
		}
		if paramOwnership == "consume" {
			id, ok := arg.(*frontend.IdentExpr)
			if !ok {
				return "", regionNone, fmt.Errorf("%s: consume argument for '%s' must be a local value", frontend.FormatPos(arg.Pos()), resolved)
			}
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return "", regionNone, fmt.Errorf("%s: borrowed value derived from '%s' cannot be consumed by '%s'", frontend.FormatPos(arg.Pos()), borrowedName, resolved)
			}
			if firstPos, exists := inoutArgs[id.Name]; exists {
				return "", regionNone, fmt.Errorf("%s: consumed argument '%s' aliases inout argument in call to '%s' (inout at %s)", frontend.FormatPos(arg.Pos()), id.Name, resolved, frontend.FormatPos(firstPos))
			}
			consumeArgs[i] = id.Name
			consumeArgTypes[i] = argType
			consumeArgPositions[id.Name] = arg.Pos()
		}
		if paramOwnership == "borrow" {
			id, ok := arg.(*frontend.IdentExpr)
			if ok {
				if firstPos, exists := inoutArgs[id.Name]; exists {
					return "", regionNone, fmt.Errorf("%s: borrowed argument '%s' aliases inout argument in call to '%s' (inout at %s)", frontend.FormatPos(arg.Pos()), id.Name, resolved, frontend.FormatPos(firstPos))
				}
				borrowArgs[id.Name] = arg.Pos()
			}
		}
		if paramOwnership == "inout" {
			id, ok := arg.(*frontend.IdentExpr)
			if !ok {
				return "", regionNone, fmt.Errorf("%s: inout argument for '%s' must be a mutable local value", frontend.FormatPos(arg.Pos()), resolved)
			}
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return "", regionNone, fmt.Errorf("%s: borrowed value derived from '%s' cannot be passed as inout to '%s'", frontend.FormatPos(arg.Pos()), borrowedName, resolved)
			}
			local, ok := locals[id.Name]
			if !ok || !local.Mutable {
				return "", regionNone, fmt.Errorf("%s: inout argument '%s' for '%s' must be mutable", frontend.FormatPos(arg.Pos()), id.Name, resolved)
			}
			if firstPos, exists := inoutArgs[id.Name]; exists {
				return "", regionNone, fmt.Errorf("%s: inout argument '%s' used more than once in call to '%s' (first at %s)", frontend.FormatPos(arg.Pos()), id.Name, resolved, frontend.FormatPos(firstPos))
			}
			if firstPos, exists := borrowArgs[id.Name]; exists {
				return "", regionNone, fmt.Errorf("%s: inout argument '%s' aliases borrowed argument in call to '%s' (borrow at %s)", frontend.FormatPos(arg.Pos()), id.Name, resolved, frontend.FormatPos(firstPos))
			}
			if firstPos, exists := consumeArgPositions[id.Name]; exists {
				return "", regionNone, fmt.Errorf("%s: inout argument '%s' aliases consumed argument in call to '%s' (consume at %s)", frontend.FormatPos(arg.Pos()), id.Name, resolved, frontend.FormatPos(firstPos))
			}
			inoutArgs[id.Name] = arg.Pos()
		}
		argRegions[i] = argRegion
	}
	for i, name := range consumeArgs {
		if name == "" {
			continue
		}
		for j := 0; j < i; j++ {
			if consumeArgs[j] == name {
				return "", regionNone, fmt.Errorf("%s: value '%s' consumed more than once in call to '%s'", frontend.FormatPos(e.Args[i].Pos()), name, resolved)
			}
			if resourceValuesAlias(consumeArgs[j], consumeArgTypes[j], name, consumeArgTypes[i], types, state) {
				return "", regionNone, fmt.Errorf("%s: value '%s' consumed more than once in call to '%s'", frontend.FormatPos(e.Args[i].Pos()), name, resolved)
			}
		}
		markConsumedResourceValue(name, consumeArgTypes[i], types, state, e.Args[i].Pos())
	}
	markCallFinalizedResources(resolved, e, funcs, module, imports, state)
	if resolved == "core.spawn" {
		if len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf("%s: spawn expects 1 argument", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return "", regionNone, fmt.Errorf("%s: spawn expects a string literal", frontend.FormatPos(e.At))
		}
		raw := string(lit.Value)
		if raw == "" {
			return "", regionNone, fmt.Errorf("%s: spawn expects a non-empty name", frontend.FormatPos(e.At))
		}
		target, err := resolveCallName(raw, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf("%s: spawn target must be a user function, got '%s'", frontend.FormatPos(e.At), target)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), target)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			return "", regionNone, fmt.Errorf("%s: spawn target must have shape fun %s(): i32", frontend.FormatPos(e.At), target)
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf("%s: spawn target must be synchronous", frontend.FormatPos(e.At))
		}
		if targetSig.ThrowsType != "" {
			return "", regionNone, fmt.Errorf("%s: spawn target must not throw", frontend.FormatPos(e.At))
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf("%s: spawn target '%s' touches mutable global state and cannot cross actor boundary", frontend.FormatPos(e.At), target)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf("%s: spawn target '%s' is not sendable across actor boundary", frontend.FormatPos(e.At), target)
		}
		lit.Value = []byte(target)
	}
	if resolved == "core.task_spawn_i32" {
		if len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32 expects 1 argument", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32 expects a string literal", frontend.FormatPos(e.At))
		}
		raw := string(lit.Value)
		if raw == "" {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32 expects a non-empty name", frontend.FormatPos(e.At))
		}
		target, err := resolveCallName(raw, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32 target must be a user function, got '%s'", frontend.FormatPos(e.At), target)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), target)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32 target must have shape func %s() -> i32", frontend.FormatPos(e.At), target)
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32 target must be synchronous", frontend.FormatPos(e.At))
		}
		if targetSig.ThrowsType != "" {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32 target must not throw", frontend.FormatPos(e.At))
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32 target '%s' touches mutable global state and cannot cross task boundary", frontend.FormatPos(e.At), target)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32 target '%s' is not sendable across task boundary", frontend.FormatPos(e.At), target)
		}
		lit.Value = []byte(target)
	}
	if resolved == "core.task_spawn_group_i32" {
		if len(e.Args) != 2 {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32 expects 2 arguments", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[1].(*frontend.StringLitExpr)
		if !ok {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32 expects a string literal worker name", frontend.FormatPos(e.At))
		}
		raw := string(lit.Value)
		if raw == "" {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32 expects a non-empty name", frontend.FormatPos(e.At))
		}
		target, err := resolveCallName(raw, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32 target must be a user function, got '%s'", frontend.FormatPos(e.At), target)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), target)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32 target must have shape func %s() -> i32", frontend.FormatPos(e.At), target)
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32 target must be synchronous", frontend.FormatPos(e.At))
		}
		if targetSig.ThrowsType != "" {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32 target must not throw", frontend.FormatPos(e.At))
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32 target '%s' touches mutable global state and cannot cross task boundary", frontend.FormatPos(e.At), target)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32 target '%s' is not sendable across task boundary", frontend.FormatPos(e.At), target)
		}
		lit.Value = []byte(target)
	}
	if resolved == "core.sym_addr" {
		if len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf("%s: sym_addr expects 1 argument", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return "", regionNone, fmt.Errorf("%s: sym_addr expects a string literal", frontend.FormatPos(e.At))
		}
		if len(lit.Value) == 0 {
			return "", regionNone, fmt.Errorf("%s: sym_addr expects a non-empty symbol name", frontend.FormatPos(e.At))
		}
	}
	if (resolved == "core.island_make_u8" || resolved == "core.island_make_u16" || resolved == "core.island_make_i32" || resolved == "core.island_make_bool") && len(argRegions) > 0 && argRegions[0] == regionUnknown {
		return "", regionNone, fmt.Errorf("%s: ambiguous region for '%s' argument", frontend.FormatPos(e.At), resolved)
	}
	if err := effects.requireAll(e.At, sig.Effects); err != nil {
		return "", regionNone, err
	}
	if builtinNeedsUnsafe(resolved, argRegions) && !state.inUnsafe() {
		return "", regionNone, fmt.Errorf("%s: '%s' is only allowed in unsafe blocks", frontend.FormatPos(e.At), resolved)
	}
	if permission, attenuatedEffect := builtinCapsulePermission(resolved); permission != "" {
		if err := effects.requireCapsulePermission(e.At, permission, attenuatedEffect); err != nil {
			return "", regionNone, err
		}
	}
	e.Name = resolved
	regionID := regionNone
	if sig.ReturnRegionParam >= 0 {
		if sig.ReturnRegionParam >= len(argRegions) {
			return "", regionNone, fmt.Errorf("%s: invalid region signature for '%s'", frontend.FormatPos(e.At), resolved)
		}
		regionID = argRegions[sig.ReturnRegionParam]
		if regionID == regionUnknown {
			return "", regionNone, fmt.Errorf("%s: ambiguous region for '%s' return", frontend.FormatPos(e.At), resolved)
		}
	}
	return sig.ReturnType, regionID, nil
}

var noblockForbiddenCallEffects = []string{"actors", "control", "io", "link", "mmio", "runtime"}

var realtimeForbiddenCallEffects = []string{"actors", "alloc", "control", "io", "link", "mmio", "runtime"}

func currentCallerSignature(effects *effectContext, funcs map[string]FuncSig) (FuncSig, bool) {
	if effects == nil || effects.funcName == "" {
		return FuncSig{}, false
	}
	sig, ok := funcs[effects.funcName]
	if !ok {
		return FuncSig{}, false
	}
	return sig, true
}

func validateCallAgainstSemanticClauses(callerSig FuncSig, calleeSig FuncSig, calleeName string, pos frontend.Position) error {
	if callerSig.HasRealtime {
		if blocked := firstFuncSigForbiddenEffect(calleeSig, realtimeForbiddenCallEffects); blocked != "" {
			return fmt.Errorf("%s: semantic clause 'realtime' forbids call to '%s' because it is not realtime-safe (effect '%s')", frontend.FormatPos(pos), calleeName, blocked)
		}
	}
	if callerSig.HasNoAlloc && funcSigHasEffect(calleeSig, "alloc") {
		return fmt.Errorf("%s: semantic clause 'noalloc' forbids call to '%s' because it may allocate", frontend.FormatPos(pos), calleeName)
	}
	if callerSig.HasNoBlock {
		if blocked := firstFuncSigForbiddenEffect(calleeSig, noblockForbiddenCallEffects); blocked != "" {
			return fmt.Errorf("%s: semantic clause 'noblock' forbids call to '%s' because it may block (effect '%s')", frontend.FormatPos(pos), calleeName, blocked)
		}
	}
	return nil
}

func validateCallbackClauseCompatibility(
	callbackSig FuncSig,
	calleeSig FuncSig,
	calleeName string,
	pos frontend.Position,
	callbackName string,
) error {
	if calleeSig.HasRealtime {
		if blocked := firstFuncSigForbiddenEffect(callbackSig, realtimeForbiddenCallEffects); blocked != "" {
			return fmt.Errorf("%s: callback function symbol '%s' is not realtime-safe (effect '%s') for callee '%s'", frontend.FormatPos(pos), callbackName, blocked, calleeName)
		}
	}
	if calleeSig.HasNoAlloc && funcSigHasEffect(callbackSig, "alloc") {
		return fmt.Errorf("%s: callback function symbol '%s' may allocate but callee '%s' has semantic clause 'noalloc'", frontend.FormatPos(pos), callbackName, calleeName)
	}
	if calleeSig.HasNoBlock {
		if blocked := firstFuncSigForbiddenEffect(callbackSig, noblockForbiddenCallEffects); blocked != "" {
			return fmt.Errorf("%s: callback function symbol '%s' may block (effect '%s') but callee '%s' has semantic clause 'noblock'", frontend.FormatPos(pos), callbackName, blocked, calleeName)
		}
	}
	return nil
}

func funcSigHasEffect(sig FuncSig, effect string) bool {
	for _, name := range sig.Effects {
		if name == effect {
			return true
		}
	}
	return false
}

func firstFuncSigForbiddenEffect(sig FuncSig, forbidden []string) string {
	effects := effectSet(sig.Effects)
	return firstForbiddenEffect(effects, forbidden)
}

func hasStrictSemanticCallClauses(sig FuncSig) bool {
	return sig.HasNoAlloc || sig.HasNoBlock || sig.HasRealtime
}

func firstStrictSemanticCallClause(sig FuncSig) string {
	if sig.HasRealtime {
		return "realtime"
	}
	if sig.HasNoAlloc {
		return "noalloc"
	}
	if sig.HasNoBlock {
		return "noblock"
	}
	return ""
}

func resolveCallbackArgumentType(
	arg *frontend.IdentExpr,
	calleeName string,
	calleeSig FuncSig,
	paramIndex int,
	locals map[string]LocalInfo,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	deferEffectValidation bool,
) (string, string, error) {
	if localInfo, ok := locals[arg.Name]; ok {
		if !localInfo.FunctionTypeValue || localInfo.FunctionValue == "" {
			return "", "", fmt.Errorf("%s: callback argument must be a symbol-backed local function value or direct named function/closure symbol in this MVP", frontend.FormatPos(arg.Pos()))
		}
		if localInfo.GenericFunctionValue {
			return "", "", fmt.Errorf("%s: generic function symbol '%s' is not supported for callback argument in this MVP", frontend.FormatPos(arg.Pos()), arg.Name)
		}
		if len(localInfo.FunctionCaptures) > 0 {
			return "", "", unsupportedCallbackCaptureError(arg.Pos(), arg.Name)
		}
		localSig, ok := funcs[localInfo.FunctionValue]
		if !ok {
			return "", "", fmt.Errorf("%s: unknown callback function symbol '%s'", frontend.FormatPos(arg.Pos()), localInfo.FunctionValue)
		}
		if err := validateCallbackSignature(localSig, calleeSig, paramIndex, arg.At, arg.Name, deferEffectValidation); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(localSig, calleeSig, calleeName, arg.At, arg.Name); err != nil {
			return "", "", err
		}
		return localInfo.TypeName, localInfo.FunctionValue, nil
	}

	resolved, err := resolveCheckedCallName(arg.Name, funcs, module, imports, arg.At)
	if err != nil {
		return "", "", fmt.Errorf("%s: callback argument must be a symbol-backed local function value or direct named function/closure symbol in this MVP", frontend.FormatPos(arg.Pos()))
	}
	sig, ok := funcs[resolved]
	if !ok {
		return "", "", fmt.Errorf("%s: callback argument must be a symbol-backed local function value or direct named function/closure symbol in this MVP", frontend.FormatPos(arg.Pos()))
	}
	if err := ensureFuncVisible(resolved, sig, module, arg.At); err != nil {
		return "", "", err
	}
	if sig.Generic {
		return "", "", fmt.Errorf("%s: generic function symbol '%s' is not supported for callback argument in this MVP", frontend.FormatPos(arg.Pos()), arg.Name)
	}
	if sig.ThrowsType != "" {
		return "", "", fmt.Errorf("%s: throwing function symbol '%s' is not supported for callback argument in this MVP", frontend.FormatPos(arg.Pos()), arg.Name)
	}
	if err := validateCallbackSignature(sig, calleeSig, paramIndex, arg.At, arg.Name, deferEffectValidation); err != nil {
		return "", "", err
	}
	if err := validateCallbackClauseCompatibility(sig, calleeSig, calleeName, arg.At, arg.Name); err != nil {
		return "", "", err
	}
	return "ptr", resolved, nil
}

func unsupportedCallbackCaptureError(pos frontend.Position, rawName string) error {
	return fmt.Errorf("%s: callback argument '%s' captures local values; captured function values cannot be passed as callback arguments in this MVP", frontend.FormatPos(pos), rawName)
}

func validateCallbackSignature(
	callbackSig FuncSig,
	calleeSig FuncSig,
	paramIndex int,
	pos frontend.Position,
	rawName string,
	deferEffectValidation bool,
) error {
	if paramIndex >= len(calleeSig.ParamFunctionParams) || paramIndex >= len(calleeSig.ParamFunctionReturns) {
		return nil
	}
	wantParams := calleeSig.ParamFunctionParams[paramIndex]
	wantReturn := calleeSig.ParamFunctionReturns[paramIndex]
	wantEffects := []string(nil)
	if paramIndex < len(calleeSig.ParamFunctionEffects) {
		wantEffects = calleeSig.ParamFunctionEffects[paramIndex]
	}
	if len(wantParams) != len(callbackSig.ParamTypes) {
		return fmt.Errorf("%s: callback function symbol '%s' has incompatible parameter count: expected %d, got %d", frontend.FormatPos(pos), rawName, len(wantParams), len(callbackSig.ParamTypes))
	}
	for i := range wantParams {
		if wantParams[i] != callbackSig.ParamTypes[i] {
			return fmt.Errorf("%s: callback function symbol '%s' parameter %d type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), rawName, i+1, wantParams[i], callbackSig.ParamTypes[i])
		}
	}
	if wantReturn != "" && wantReturn != callbackSig.ReturnType {
		return fmt.Errorf("%s: callback function symbol '%s' return type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), rawName, wantReturn, callbackSig.ReturnType)
	}
	if !deferEffectValidation {
		if err := validateFunctionTypeCallableEffects(wantEffects, callbackSig.Effects, pos, "callback function symbol", rawName); err != nil {
			return err
		}
	}
	return nil
}

func checkTypedActorBuiltin(
	e *frontend.CallExpr,
	resolved string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, int, error) {
	if err := effects.requireAll(e.At, []string{"actors"}); err != nil {
		return "", regionNone, err
	}
	switch resolved {
	case "core.send_typed":
		if len(e.TypeArgs) != 0 {
			return "", regionNone, fmt.Errorf("%s: send_typed does not accept explicit type arguments", frontend.FormatPos(e.At))
		}
		if len(e.Args) != 2 {
			return "", regionNone, fmt.Errorf("%s: send_typed expects 2 arguments", frontend.FormatPos(e.At))
		}
		targetType, _, err := checkExprWithEffects(e.Args[0], locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		if targetType != "actor" {
			return "", regionNone, fmt.Errorf("%s: type mismatch for 'core.send_typed' arg 1", frontend.FormatPos(e.Args[0].Pos()))
		}
		msgType, _, err := checkExprWithEffects(e.Args[1], locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		info, ok := types[msgType]
		if !ok || info.Kind != TypeEnum {
			return "", regionNone, fmt.Errorf("%s: send_typed expects an enum message", frontend.FormatPos(e.Args[1].Pos()))
		}
		if err := validateTypedActorMessageType(msgType, types, map[string]bool{}); err != nil {
			return "", regionNone, fmt.Errorf("%s: %v", frontend.FormatPos(e.Args[1].Pos()), err)
		}
		if err := consumeTypedActorTransferPayloads(e.Args[1], msgType, locals, types, module, imports, state); err != nil {
			return "", regionNone, err
		}
		e.Name = resolved
		return "i32", regionNone, nil
	case "core.recv_typed":
		if len(e.Args) != 0 {
			return "", regionNone, fmt.Errorf("%s: recv_typed expects 0 arguments", frontend.FormatPos(e.At))
		}
		if len(e.TypeArgs) != 1 {
			return "", regionNone, fmt.Errorf("%s: recv_typed expects one explicit type argument", frontend.FormatPos(e.At))
		}
		typeName, err := resolveTypeName(&e.TypeArgs[0], module, imports)
		if err != nil {
			return "", regionNone, err
		}
		e.TypeArgs[0].Name = typeName
		info, ok := types[typeName]
		if !ok || info.Kind != TypeEnum {
			return "", regionNone, fmt.Errorf("%s: recv_typed expects an enum type argument", frontend.FormatPos(e.TypeArgs[0].At))
		}
		if err := validateTypedActorMessageType(typeName, types, map[string]bool{}); err != nil {
			return "", regionNone, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
		}
		e.Name = resolved
		return typeName, regionNone, nil
	default:
		return "", regionNone, fmt.Errorf("%s: unknown typed actor builtin '%s'", frontend.FormatPos(e.At), resolved)
	}
}

func checkTypedTaskBuiltin(
	e *frontend.CallExpr,
	resolved string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, int, error) {
	if err := effects.requireAll(e.At, []string{"runtime"}); err != nil {
		return "", regionNone, err
	}
	if len(e.TypeArgs) != 1 {
		return "", regionNone, fmt.Errorf("%s: %s expects one explicit error type argument", frontend.FormatPos(e.At), resolved)
	}
	errorType, err := resolveTypeName(&e.TypeArgs[0], module, imports)
	if err != nil {
		return "", regionNone, err
	}
	e.TypeArgs[0].Name = errorType
	if err := validateTypedTaskErrorType(errorType, types, e.TypeArgs[0].At); err != nil {
		return "", regionNone, err
	}
	handleType, handleInfo, err := EnsureTypedTaskHandleType(errorType, types)
	if err != nil {
		return "", regionNone, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
	}

	switch resolved {
	case "core.task_spawn_i32_typed", "core.task_spawn_group_i32_typed":
		if resolved == "core.task_spawn_i32_typed" && len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32_typed expects 1 argument", frontend.FormatPos(e.At))
		}
		if resolved == "core.task_spawn_group_i32_typed" && len(e.Args) != 2 {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32_typed expects 2 arguments", frontend.FormatPos(e.At))
		}
		workerArg := 0
		if resolved == "core.task_spawn_group_i32_typed" {
			groupType, _, err := checkExprWithEffects(e.Args[0], locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return "", regionNone, err
			}
			if groupType != "task.group" {
				return "", regionNone, fmt.Errorf("%s: type mismatch for 'core.task_spawn_group_i32_typed' arg 1", frontend.FormatPos(e.Args[0].Pos()))
			}
			workerArg = 1
		}
		lit, ok := e.Args[workerArg].(*frontend.StringLitExpr)
		if !ok {
			if resolved == "core.task_spawn_group_i32_typed" {
				return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32_typed expects a string literal worker name", frontend.FormatPos(e.At))
			}
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32_typed expects a string literal", frontend.FormatPos(e.At))
		}
		raw := string(lit.Value)
		if raw == "" {
			if resolved == "core.task_spawn_group_i32_typed" {
				return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32_typed expects a non-empty name", frontend.FormatPos(e.At))
			}
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32_typed expects a non-empty name", frontend.FormatPos(e.At))
		}
		target, err := resolveCallName(raw, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf("%s: %s target must be a user function, got '%s'", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), target)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			if handleInfo.SlotCount > 4 {
				return "", regionNone, fmt.Errorf("%s: %s target must have shape func %s() -> i32", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target)
			}
			return "", regionNone, fmt.Errorf("%s: %s target must have shape func %s() -> i32 throws %s", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target, displayTypeName(errorType, module))
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf("%s: %s target must be synchronous", frontend.FormatPos(e.At), taskTypedSpawnName(resolved))
		}
		if handleInfo.SlotCount > 4 {
			if targetSig.ThrowsType != "" && targetSig.ThrowsType != errorType {
				return "", regionNone, fmt.Errorf("%s: %s target must throw '%s' (or be non-throwing in staged mode)", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), displayTypeName(errorType, module))
			}
		} else if targetSig.ThrowsType != errorType {
			return "", regionNone, fmt.Errorf("%s: %s target must throw '%s'", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), displayTypeName(errorType, module))
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf("%s: %s target '%s' touches mutable global state and cannot cross task boundary", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf("%s: %s target '%s' is not sendable across task boundary", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target)
		}
		lit.Value = []byte(target)
		e.Name = resolved
		return handleType, regionNone, nil
	case "core.task_join_i32_typed", "core.task_join_group_i32_typed":
		if len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf("%s: task_join_i32_typed expects 1 argument", frontend.FormatPos(e.At))
		}
		argType, _, err := checkExprWithEffects(e.Args[0], locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		if argType != handleType {
			return "", regionNone, fmt.Errorf("%s: type mismatch for '%s' arg 1: expected '%s', got '%s'", frontend.FormatPos(e.Args[0].Pos()), resolved, handleType, argType)
		}
		isTryCall := state != nil && state.allowThrowDepth > 0 && state.allowThrowCall == e
		isCatchCall := state != nil && state.allowCatchDepth > 0 && state.allowCatchCall == e
		if !isTryCall && !isCatchCall {
			return "", regionNone, fmt.Errorf("%s: call to throwing function '%s' requires try", frontend.FormatPos(e.At), resolved)
		}
		if isTryCall && state.throwType == "" {
			return "", regionNone, fmt.Errorf("%s: try is only allowed in throwing functions", frontend.FormatPos(e.At))
		}
		if isTryCall && state.throwType != errorType {
			return "", regionNone, fmt.Errorf("%s: thrown error type mismatch: expected '%s', got '%s'", frontend.FormatPos(e.At), state.throwType, errorType)
		}
		markTaskHandleJoined(e.Args[0], funcs, module, imports, state)
		e.Name = resolved
		return "i32", regionNone, nil
	default:
		return "", regionNone, fmt.Errorf("%s: unknown typed task builtin '%s'", frontend.FormatPos(e.At), resolved)
	}
}

func checkResourceCallArg(
	resolved string,
	paramType string,
	arg frontend.Expr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if paramType != "task.group" {
		return nil
	}
	source, err := resourceSourceForExpr(arg, funcs, module, imports, state)
	if err != nil {
		return err
	}
	if source.ambiguous {
		return fmt.Errorf("%s: resource expression mixes resource provenance", frontend.FormatPos(arg.Pos()))
	}
	if source.unknown {
		name, _ := resourcePathForExpr(arg)
		if name == "" {
			name = "<resource>"
		}
		return fmt.Errorf("%s: ambiguous resource provenance for '%s' after control-flow merge", frontend.FormatPos(arg.Pos()), name)
	}
	if !source.known {
		return nil
	}
	if resolved == "core.task_group_status" {
		return state.checkResourceFinalizationAllowed(source.name, arg.Pos(), "closed")
	}
	return state.checkResourceFinalizationAllowed(source.name, arg.Pos())
}

func markCallFinalizedResources(
	resolved string,
	call *frontend.CallExpr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	state *regionState,
) {
	if state == nil || call == nil || len(call.Args) == 0 {
		return
	}
	switch resolved {
	case "core.task_join_i32", "core.task_join_result_i32":
		markTaskHandleJoined(call.Args[0], funcs, module, imports, state)
	case "core.task_group_close":
		if source, err := resourceSourceForExpr(call.Args[0], funcs, module, imports, state); err == nil && source.known {
			state.markResourceFinalized(source.name, "closed", call.Args[0].Pos())
		}
	}
}

func markTaskHandleJoined(arg frontend.Expr, funcs map[string]FuncSig, module string, imports map[string]string, state *regionState) {
	if state == nil {
		return
	}
	if source, err := resourceSourceForExpr(arg, funcs, module, imports, state); err == nil && source.known {
		state.markResourceFinalized(source.name, "joined", arg.Pos())
	}
}

func borrowedOwnerForRegion(regionID int, state *regionState) (string, bool) {
	if state == nil || regionID == regionNone || regionID == regionUnknown {
		return "", false
	}
	if borrowedName, borrowed := state.borrowedParamOwner(regionID); borrowed {
		return borrowedName, true
	}
	return "", false
}

func borrowedOwnerFromExpr(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, funcs map[string]FuncSig, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState, effects *effectContext, analysis *functionAnalysisState) (string, bool, error) {
	_, regionID, err := checkExprWithEffects(expr, locals, globals, funcs, types, module, imports, state, effects, analysis)
	if err != nil {
		return "", false, err
	}
	borrowedName, borrowed := borrowedOwnerForRegion(regionID, state)
	return borrowedName, borrowed, nil
}

func checkBorrowedEscape(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, funcs map[string]FuncSig, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState, effects *effectContext, analysis *functionAnalysisState, format func(string) error) error {
	if expr == nil {
		return nil
	}
	if borrowedName, borrowed, err := borrowedOwnerFromExpr(expr, locals, globals, funcs, types, module, imports, state, effects, analysis); err != nil {
		return err
	} else if borrowed {
		return format(borrowedName)
	}
	if lit, ok := expr.(*frontend.StructLitExpr); ok {
		for _, field := range lit.Fields {
			if err := checkBorrowedEscape(field.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkBorrowedInoutEscape(expr frontend.Expr, targetName string, pos frontend.Position, locals map[string]LocalInfo, globals map[string]GlobalInfo, funcs map[string]FuncSig, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState, effects *effectContext, analysis *functionAnalysisState) error {
	return checkBorrowedEscape(expr, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
		return fmt.Errorf("%s: borrowed local '%s' cannot escape via inout assignment to '%s'", frontend.FormatPos(pos), borrowedName, targetName)
	})
}

func markConsumedResourceValue(name string, typeName string, types map[string]*TypeInfo, state *regionState, pos frontend.Position) {
	if state == nil || name == "" {
		return
	}
	if !typeContainsResourceHandle(typeName, types) {
		state.markConsumed(name, pos)
		return
	}
	markConsumedResourcePath(name, typeName, types, state, pos)
}

func markConsumedResourcePath(prefix string, typeName string, types map[string]*TypeInfo, state *regionState, pos frontend.Position) {
	if state == nil || prefix == "" {
		return
	}
	marked := false
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		path := joinResourcePath(prefix, leaf)
		if _, ok := state.resourceID(path); ok {
			state.markConsumed(path, pos)
			marked = true
		}
	}
	if !marked {
		state.markConsumed(prefix, pos)
	}
}

func markConsumedResourceExpr(
	expr frontend.Expr,
	typeName string,
	locals map[string]LocalInfo,
	types map[string]*TypeInfo,
	state *regionState,
) bool {
	if state == nil || expr == nil {
		return false
	}
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if _, exists := locals[id.Name]; !exists {
			return false
		}
		markConsumedResourceValue(id.Name, typeName, types, state, id.At)
		return true
	}
	if path, ok := resourcePathForExpr(expr); ok {
		markConsumedResourcePath(path, typeName, types, state, expr.Pos())
		return true
	}
	return false
}

func resourceValuesAlias(leftName string, leftType string, rightName string, rightType string, types map[string]*TypeInfo, state *regionState) bool {
	if state == nil {
		return false
	}
	leftIDs := resourceIDsForValue(leftName, leftType, types, state)
	if len(leftIDs) == 0 {
		return false
	}
	for id := range resourceIDsForValue(rightName, rightType, types, state) {
		if leftIDs[id] {
			return true
		}
	}
	return false
}

func resourceIDsForValue(name string, typeName string, types map[string]*TypeInfo, state *regionState) map[int]bool {
	ids := make(map[int]bool)
	if state == nil || name == "" {
		return ids
	}
	if typeContainsResourceHandle(typeName, types) {
		for _, leaf := range resourceLeafPaths(typeName, types, "") {
			if id, ok := state.resourceID(joinResourcePath(name, leaf)); ok {
				ids[id] = true
			}
		}
	}
	if id, ok := state.resourceID(name); ok {
		ids[id] = true
	}
	return ids
}

func checkResourceTreeUsable(name string, typeName string, types map[string]*TypeInfo, state *regionState, pos frontend.Position) error {
	if state == nil || name == "" || !typeContainsResourceHandle(typeName, types) {
		return nil
	}
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		path := joinResourcePath(name, leaf)
		source := resourceSourceForPath(path, state)
		if source.unknown {
			return fmt.Errorf("%s: ambiguous resource provenance for '%s' after control-flow merge", frontend.FormatPos(pos), path)
		}
		if !source.known {
			continue
		}
		if err := state.checkNotConsumed(source.name, pos); err != nil {
			return err
		}
		if err := state.checkResourceNotFinalized(source.name, pos); err != nil {
			return err
		}
	}
	return nil
}

func validateTypedTaskErrorType(typeName string, types map[string]*TypeInfo, pos frontend.Position) error {
	info, ok := types[typeName]
	if !ok {
		return fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), typeName)
	}
	if info.Kind != TypeEnum {
		return fmt.Errorf("%s: typed task error argument must be an enum", frontend.FormatPos(pos))
	}
	return nil
}

func taskTypedSpawnName(resolved string) string {
	if resolved == "core.task_spawn_group_i32_typed" {
		return "task_spawn_group_i32_typed"
	}
	return "task_spawn_i32_typed"
}

func validateTypedActorMessageType(typeName string, types map[string]*TypeInfo, visiting map[string]bool) error {
	info, ok := types[typeName]
	if !ok {
		return fmt.Errorf("unknown type '%s'", typeName)
	}
	switch info.Kind {
	case TypeI32, TypeU8, TypeBool, TypeIsland:
		return nil
	case TypeEnum:
		if info.SlotCount-1 > 8 {
			return fmt.Errorf("typed actor message payload supports at most 8 value slots, got %d for '%s'", info.SlotCount-1, typeName)
		}
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if err := validateTypedActorMessageType(payload, types, visiting); err != nil {
					return err
				}
			}
		}
		return nil
	case TypeStruct:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if err := validateTypedActorMessageType(field.TypeName, types, visiting); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("typed actor message payload must be value-only, got '%s'", typeName)
	}
}

func typedActorTypeContainsIsland(typeName string, types map[string]*TypeInfo, visiting map[string]bool) bool {
	info, ok := types[typeName]
	if !ok {
		return false
	}
	switch info.Kind {
	case TypeIsland:
		return true
	case TypeEnum:
		if visiting[typeName] {
			return false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if typedActorTypeContainsIsland(payload, types, visiting) {
					return true
				}
			}
		}
	case TypeStruct:
		if visiting[typeName] {
			return false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if typedActorTypeContainsIsland(field.TypeName, types, visiting) {
				return true
			}
		}
	}
	return false
}

func consumeIslandSourceLocals(
	expr frontend.Expr,
	typeName string,
	locals map[string]LocalInfo,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) {
	if !typedActorTypeContainsIsland(typeName, types, map[string]bool{}) {
		return
	}
	info, ok := types[typeName]
	if !ok {
		return
	}
	switch info.Kind {
	case TypeIsland:
		markConsumedResourceExpr(expr, typeName, locals, types, state)
	case TypeStruct:
		if lit, ok := expr.(*frontend.StructLitExpr); ok {
			for _, field := range lit.Fields {
				fieldInfo, ok := info.FieldMap[field.Name]
				if !ok {
					continue
				}
				consumeIslandSourceLocals(field.Value, fieldInfo.TypeName, locals, types, module, imports, state)
			}
			return
		}
		markConsumedResourceExpr(expr, typeName, locals, types, state)
	case TypeEnum:
		if call, ok := expr.(*frontend.CallExpr); ok {
			_, caseInfo, found, err := resolveEnumCaseConstructorCall(call, types, module, imports)
			if err == nil && found {
				for i, arg := range call.Args {
					if i >= len(caseInfo.PayloadTypes) {
						break
					}
					consumeIslandSourceLocals(arg, caseInfo.PayloadTypes[i], locals, types, module, imports, state)
				}
				return
			}
		}
		markConsumedResourceExpr(expr, typeName, locals, types, state)
	}
}

func consumeTypedActorTransferPayloads(
	expr frontend.Expr,
	typeName string,
	locals map[string]LocalInfo,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if !typedActorTypeContainsIsland(typeName, types, map[string]bool{}) {
		return nil
	}
	info, ok := types[typeName]
	if !ok {
		return fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(expr.Pos()), typeName)
	}
	switch info.Kind {
	case TypeIsland:
		if markConsumedResourceExpr(expr, typeName, locals, types, state) {
			return nil
		}
		id, ok := expr.(*frontend.IdentExpr)
		if !ok {
			return fmt.Errorf("%s: island transfer payload must be a local value", frontend.FormatPos(expr.Pos()))
		}
		if _, ok := locals[id.Name]; !ok {
			return fmt.Errorf("%s: island transfer payload '%s' must be a local value", frontend.FormatPos(id.At), id.Name)
		}
		markConsumedResourceValue(id.Name, typeName, types, state, id.At)
		return nil
	case TypeStruct:
		if lit, ok := expr.(*frontend.StructLitExpr); ok {
			for _, field := range lit.Fields {
				fieldInfo, ok := info.FieldMap[field.Name]
				if !ok {
					continue
				}
				if err := consumeTypedActorTransferPayloads(field.Value, fieldInfo.TypeName, locals, types, module, imports, state); err != nil {
					return err
				}
			}
			return nil
		}
		if markConsumedResourceExpr(expr, typeName, locals, types, state) {
			return nil
		}
		id, ok := expr.(*frontend.IdentExpr)
		if !ok {
			return fmt.Errorf("%s: island-containing struct transfer payload must be a local value", frontend.FormatPos(expr.Pos()))
		}
		if _, ok := locals[id.Name]; !ok {
			return fmt.Errorf("%s: island-containing transfer payload '%s' must be a local value", frontend.FormatPos(id.At), id.Name)
		}
		markConsumedResourceValue(id.Name, typeName, types, state, id.At)
		return nil
	case TypeEnum:
		if call, ok := expr.(*frontend.CallExpr); ok {
			_, caseInfo, found, err := resolveEnumCaseConstructorCall(call, types, module, imports)
			if err != nil {
				return err
			}
			if found {
				for i, arg := range call.Args {
					if i >= len(caseInfo.PayloadTypes) {
						break
					}
					if err := consumeTypedActorTransferPayloads(arg, caseInfo.PayloadTypes[i], locals, types, module, imports, state); err != nil {
						return err
					}
				}
				return nil
			}
		}
		if markConsumedResourceExpr(expr, typeName, locals, types, state) {
			return nil
		}
		id, ok := expr.(*frontend.IdentExpr)
		if !ok {
			return fmt.Errorf("%s: island-containing enum transfer payload must be a local value", frontend.FormatPos(expr.Pos()))
		}
		if _, ok := locals[id.Name]; !ok {
			return fmt.Errorf("%s: island-containing transfer payload '%s' must be a local value", frontend.FormatPos(id.At), id.Name)
		}
		markConsumedResourceValue(id.Name, typeName, types, state, id.At)
		return nil
	default:
		return nil
	}
}

func checkStructConstructorCallWithEffects(
	e *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, int, bool, error) {
	if len(e.Args) == 0 || len(e.ArgLabels) != len(e.Args) {
		return "", regionNone, false, nil
	}
	for _, label := range e.ArgLabels {
		if label == "" {
			return "", regionNone, false, nil
		}
	}

	typeRef := frontend.TypeRef{At: e.At, Kind: frontend.TypeRefNamed, Name: e.Name}
	resolvedType, err := resolveTypeName(&typeRef, module, imports)
	if err != nil {
		return "", regionNone, false, nil
	}
	info, ok := types[resolvedType]
	if !ok || info.Kind != TypeStruct {
		return "", regionNone, false, nil
	}
	if err := ensureTypeVisible(resolvedType, info, module, e.At); err != nil {
		return "", regionNone, true, err
	}
	if len(e.Args) != len(info.Fields) {
		return "", regionNone, true, fmt.Errorf("%s: wrong field count for '%s'", frontend.FormatPos(e.At), resolvedType)
	}

	argByLabel := make(map[string]frontend.Expr, len(e.Args))
	for i, label := range e.ArgLabels {
		if _, exists := argByLabel[label]; exists {
			return "", regionNone, true, fmt.Errorf("%s: duplicate field '%s'", frontend.FormatPos(e.Args[i].Pos()), label)
		}
		argByLabel[label] = e.Args[i]
	}
	for label, expr := range argByLabel {
		if _, ok := info.FieldMap[label]; !ok {
			return "", regionNone, true, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(expr.Pos()), label)
		}
	}

	orderedArgs := make([]frontend.Expr, 0, len(info.Fields))
	orderedLabels := make([]string, 0, len(info.Fields))
	structRegion := regionNone
	for _, field := range info.Fields {
		arg, ok := argByLabel[field.Name]
		if !ok {
			return "", regionNone, true, fmt.Errorf("%s: missing field '%s'", frontend.FormatPos(e.At), field.Name)
		}
		valType, valRegion, err := checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, true, err
		}
		if !typesCompatibleWithNullPtr(field.TypeName, valType, arg) {
			return "", regionNone, true, fmt.Errorf("%s: type mismatch for field '%s'", frontend.FormatPos(arg.Pos()), field.Name)
		}
		consumeIslandSourceLocals(arg, field.TypeName, locals, types, module, imports, state)
		structRegion = joinRegion(structRegion, valRegion)
		if structRegion == regionUnknown {
			return "", regionNone, true, fmt.Errorf("%s: struct constructor mixes values from different regions", frontend.FormatPos(arg.Pos()))
		}
		orderedArgs = append(orderedArgs, arg)
		orderedLabels = append(orderedLabels, field.Name)
	}

	e.Name = resolvedType
	e.Args = orderedArgs
	e.ArgLabels = orderedLabels
	return resolvedType, structRegion, true, nil
}

func reportBarePayloadMatchExprPatterns(
	e *frontend.MatchExpr,
	scrutType string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	if e == nil {
		return nil
	}
	info, ok := types[scrutType]
	if !ok || info.Kind != TypeEnum {
		return nil
	}
	for i := range e.Cases {
		c := &e.Cases[i]
		if c.Default || c.Guard != nil {
			continue
		}
		caseName, ok := bareEnumPatternCaseName(c.Pattern, scrutType, module, imports)
		if !ok {
			continue
		}
		caseInfo, ok := info.CaseMap[caseName]
		if !ok || len(caseInfo.PayloadTypes) == 0 {
			continue
		}
		return barePayloadRequiredDiagnostic(c.At, scrutType, caseName, len(caseInfo.PayloadTypes), module)
	}
	return nil
}

func reportBarePayloadCatchExprPatterns(
	e *frontend.CatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	if e == nil {
		return nil
	}
	info, ok := types[e.ErrorType]
	if !ok || info.Kind != TypeEnum {
		return nil
	}
	for i := range e.Cases {
		c := &e.Cases[i]
		if c.Default || c.Guard != nil {
			continue
		}
		caseName, ok := bareEnumPatternCaseName(c.Pattern, e.ErrorType, module, imports)
		if !ok {
			continue
		}
		caseInfo, ok := info.CaseMap[caseName]
		if !ok || len(caseInfo.PayloadTypes) == 0 {
			continue
		}
		return barePayloadRequiredDiagnostic(c.At, e.ErrorType, caseName, len(caseInfo.PayloadTypes), module)
	}
	return nil
}

func barePayloadRequiredDiagnostic(pos frontend.Position, typeName string, caseName string, arity int, module string) error {
	if arity <= 0 {
		arity = 1
	}
	return fmt.Errorf("%s: enum case '%s.%s' carries %d payload value(s); use '%s.%s(%s)'", frontend.FormatPos(pos), displayTypeName(typeName, module), caseName, arity, displayTypeName(typeName, module), caseName, placeholderBindingList(arity))
}

func bareEnumPatternCaseName(pattern frontend.Expr, expectedType string, module string, imports map[string]string) (string, bool) {
	field, ok := pattern.(*frontend.FieldAccessExpr)
	if !ok {
		return "", false
	}
	baseName, fields, pos, ok := splitFieldPath(field.Base)
	if !ok {
		return "", false
	}
	parts := append([]string{baseName}, fields...)
	if len(parts) == 0 {
		return "", false
	}
	ref := frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed, Name: strings.Join(parts, ".")}
	typeName, err := resolveTypeName(&ref, module, imports)
	if err != nil || typeName != expectedType {
		return "", false
	}
	return field.Field, true
}

func comparableTypes(left, right string, types map[string]*TypeInfo) bool {
	if left == "none" || right == "none" {
		if left == "none" && right == "none" {
			return true
		}
		if _, ok := optionalElemName(left); ok && right == "none" {
			return true
		}
		if _, ok := optionalElemName(right); ok && left == "none" {
			return true
		}
		return false
	}
	if left == right {
		info, ok := types[left]
		if !ok {
			return false
		}
		switch info.Kind {
		case TypeI32, TypeU8, TypeBool, TypePtr, TypeIsland, TypeCap, TypeActor, TypeEnum:
			return info.SlotCount == 1
		default:
			return false
		}
	}
	if isInt32Like(left) && isInt32Like(right) {
		return true
	}
	leftInfo, leftOK := types[left]
	rightInfo, rightOK := types[right]
	if leftOK && rightOK && (leftInfo.Kind == TypeEnum || rightInfo.Kind == TypeEnum) {
		return false
	}
	return false
}
