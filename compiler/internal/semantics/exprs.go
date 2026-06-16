package semantics

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
)

func markMutableFunctionTypedGlobalSource(expr frontend.Expr, globals map[string]GlobalInfo, analysis *functionAnalysisState) {
	if analysis == nil {
		return
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok {
		return
	}
	global, ok := globals[id.Name]
	if ok && global.Mutable && global.FunctionTypeValue {
		analysis.touchesMutableGlobals = true
	}
}

func markFunctionTargetMutableGlobalUse(sig FuncSig, analysis *functionAnalysisState) {
	if analysis != nil && sig.TouchesMutableGlobals {
		analysis.touchesMutableGlobals = true
	}
}

func markFunctionTypedReturnCallMutableGlobalUse(
	callSig FuncSig,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	analysis *functionAnalysisState,
) error {
	if analysis == nil {
		return nil
	}
	if callSig.ReturnFunctionTouchesMutableGlobals {
		analysis.touchesMutableGlobals = true
	}
	if callSig.ReturnFunctionSymbol != "" {
		if targetSig, ok := funcs[callSig.ReturnFunctionSymbol]; ok {
			markFunctionTargetMutableGlobalUse(targetSig, analysis)
		}
	}
	if callSig.ReturnFunctionParamName == "" {
		return nil
	}
	returnInfo, found, err := functionTypedReturnParamRefMetadata(callSig, callSig.ReturnFunctionParamName, call, locals, globals, funcs, types, module, imports)
	if err != nil || !found || returnInfo.FunctionValue == "" {
		return err
	}
	if targetSig, ok := funcs[returnInfo.FunctionValue]; ok {
		markFunctionTargetMutableGlobalUse(targetSig, analysis)
	}
	return nil
}

// checkExprWithEffects returns both the semantic type name and the region id
// carried by expressions that produce region-backed resources. Callers rely on
// the region result for ownership/resource diagnostics, so this function should
// stay conservative when an expression has ambiguous provenance.
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
		if err := analysis.checkSurfaceFramePixelsUsable(e.Name, e.At); err != nil {
			return "", regionNone, err
		}
		if err := state.checkNotConsumed(e.Name, e.At); err != nil {
			return "", regionNone, err
		}
		if err := state.checkResourceNotFinalized(e.Name, e.At); err != nil {
			return "", regionNone, err
		}
		if state.resourceUnknown(e.Name) {
			return "", regionNone, ownershipDiagnosticf(e.At, "ambiguous resource provenance for '%s' after control-flow merge", e.Name)
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
			return "", regionNone, unsupportedGenericClosurePointerEscapeError(e.At, e.Name)
		}
		if info.FunctionTypeValue && info.FunctionValue != "" {
			return "", regionNone, unsupportedFunctionValueEscapeError(e.At, e.Name)
		}
		if len(info.FunctionCaptures) > 0 {
			return "", regionNone, unsupportedCapturingClosurePointerEscapeError(e.At, e.Name)
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
				return "", regionNone, lifetimeDiagnosticf(e.At, "slice from scoped island is out of scope")
			}
			if err := state.checkBorrowedRegionAfterAwait(regionID, e.Name, e.At); err != nil {
				return "", regionNone, err
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
		if path, ok := canonicalOwnershipAccessPath(e); ok {
			if err := state.checkNotConsumed(path, e.At); err != nil {
				return "", regionNone, err
			}
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
				return "", regionNone, ownershipDiagnosticf(e.At, "resource expression mixes resource provenance")
			}
			path, _ := resourcePathForExpr(e)
			if source.unknown {
				return "", regionNone, ownershipDiagnosticf(e.At, "ambiguous resource provenance for '%s' after control-flow merge", path)
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
		if typeMayContainRegion(targetType, types) {
			if path, ok := resourcePathForExpr(e); ok {
				if regionID, found := state.regionVars[path]; found {
					if err := checkRegionUsable(regionID, path, e.At, state); err != nil {
						return "", regionNone, err
					}
					return targetType, regionID, nil
				}
			}
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
			payloadTree := make(map[string]int)
			for i, arg := range e.Args {
				argType := ""
				argRegion := regionNone
				if i < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[i] {
					markMutableFunctionTypedGlobalSource(arg, globals, analysis)
					if _, err := validateFunctionTypeEnumPayloadBinding(enumType, caseInfo, i, arg, locals, globals, funcs, types, module, imports); err != nil {
						return "", regionNone, err
					}
					argType = caseInfo.PayloadTypes[i]
				} else {
					var err error
					argType, argRegion, err = checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
					if err != nil {
						return "", regionNone, err
					}
				}
				if !typesCompatibleWithNullPtr(caseInfo.PayloadTypes[i], argType, arg) {
					return "", regionNone, fmt.Errorf("%s: enum case '%s.%s' payload %d expects '%s', got '%s'", frontend.FormatPos(arg.Pos()), displayTypeName(enumType, module), caseInfo.Name, i+1, caseInfo.PayloadTypes[i], argType)
				}
				if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
					consumeIslandSourceLocals(arg, caseInfo.PayloadTypes[i], locals, types, module, imports, state)
					appendRegionTree(payloadTree, resourceEnumPayloadPath("", caseInfo.Ordinal, i), caseInfo.PayloadTypes[i], arg, argRegion, types, state)
				}
			}
			e.ResolvedType = enumType
			state.setExprRegionTree(e, payloadTree)
			return enumType, constructorRegionFromTree(payloadTree), nil
		}
		return checkCallExprWithEffects(e, locals, globals, funcs, types, module, imports, state, effects, analysis)
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			if len(e.Decl.TypeParams) > 0 {
				if name, pos, ok := firstCapture(collectClosureCaptures(e.Decl, locals)); ok {
					return "", regionNone, unsupportedGenericClosureCaptureError(pos, name)
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
		awaitPos := e.At
		if !ok {
			if await, awaitOK := e.X.(*frontend.AwaitExpr); awaitOK {
				if !state.async {
					return "", regionNone, fmt.Errorf("%s: await is only allowed in async functions", frontend.FormatPos(await.At))
				}
				call, ok = await.X.(*frontend.CallExpr)
				isTryAwait = ok
				awaitPos = await.At
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
			if err == nil {
				state.invalidateBorrowedRegionsAfterAwait(awaitPos)
			}
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
		if err == nil {
			state.invalidateBorrowedRegionsAfterAwait(e.At)
		}
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
		fieldTree := make(map[string]int)
		for _, field := range info.Fields {
			init, ok := seen[field.Name]
			if !ok {
				return "", regionNone, fmt.Errorf("%s: missing field '%s'", frontend.FormatPos(e.At), field.Name)
			}
			if field.FunctionTypeValue {
				markMutableFunctionTypedGlobalSource(init.Value, globals, analysis)
				if _, err := validateFunctionTypeStructFieldBinding(resolved, field, init.Value, locals, globals, funcs, types, module, imports); err != nil {
					return "", regionNone, err
				}
				continue
			}
			valType, valRegion, err := checkExprWithEffects(init.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return "", regionNone, err
			}
			if !typesCompatibleWithNullPtr(field.TypeName, valType, init.Value) {
				return "", regionNone, fmt.Errorf("%s: type mismatch for field '%s'", frontend.FormatPos(init.At), field.Name)
			}
			consumeIslandSourceLocals(init.Value, field.TypeName, locals, types, module, imports, state)
			appendRegionTree(fieldTree, field.Name, field.TypeName, init.Value, valRegion, types, state)
		}
		state.setExprRegionTree(e, fieldTree)
		return resolved, constructorRegionFromTree(fieldTree), nil
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
	scrutType, scrutRegion, err := checkExprWithEffects(e.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
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
			if c.Default && c.Guard == nil {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			return "", fmt.Errorf("%s: match expression must be exhaustive", frontend.FormatPos(e.At))
		}
	}
	scrutineeResourcePath := e.ScrutineeLocal
	scrutineeOwnershipPath := scrutineeResourcePath
	if path, ok := resourcePathForExpr(e.Value); ok {
		scrutineeOwnershipPath = path
	}
	if scrutineeResourcePath != "" {
		if err := bindResourceTreeFromExpr(scrutineeResourcePath, scrutType, e.Value, funcs, types, module, imports, state); err != nil {
			return "", err
		}
		bindRegionTreeFromExpr(scrutineeResourcePath, scrutType, e.Value, scrutRegion, types, state)
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
			if err := bindPatternOwnershipAliases(c.Pattern, "", scrutineeOwnershipPath, scrutType, types, module, imports, state); err != nil {
				return err
			}
			if err := bindPatternResourceLocals(c.Pattern, "", scrutineeResourcePath, scrutType, types, module, imports, state); err != nil {
				return err
			}
			if err := bindPatternRegionLocals(c.Pattern, "", scrutineeResourcePath, scrutType, types, module, imports, state); err != nil {
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
			armType, armRegion, err := checkExprWithEffects(c.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
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
				bindRegionTreeFromExpr(e.ResultLocal, e.ResultType, c.Value, armRegion, types, state)
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
			mergeFlowWithLabels(state, mergedFlow, caseFlow, "previous cases", fmt.Sprintf("case %d", i+1))
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
	var catchSig FuncSig
	catchSigOK := false
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
		catchSig = sig
		catchSigOK = true
	}
	if e.ResultType == "" {
		e.ResultType = successType
	}
	if catchSigOK {
		if err := bindCatchErrorResourceSummary(e.ErrorLocal, call, catchSig, funcs, types, module, imports, state); err != nil {
			return "", err
		}
	}
	if err := reportBarePayloadCatchExprPatterns(e, locals, globals, funcs, types, module, imports); err != nil {
		return "", err
	}
	if !catchExprHasCompleteOptionalPatterns(e, e.ErrorType, types) && !catchExprHasCompleteEnumPatterns(e, e.ErrorType, locals, globals, funcs, types, module, imports) {
		hasDefault := false
		for _, c := range e.Cases {
			if c.Default && c.Guard == nil {
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
			if err := bindPatternOwnershipAliases(c.Pattern, "", e.ErrorLocal, e.ErrorType, types, module, imports, state); err != nil {
				return err
			}
			if err := bindPatternBorrowedPtrAliases(c.Pattern, "", e.ErrorLocal, e.ErrorType, types, module, imports, state); err != nil {
				return err
			}
			if err := bindPatternResourceLocals(c.Pattern, "", e.ErrorLocal, e.ErrorType, types, module, imports, state); err != nil {
				return err
			}
			if err := bindPatternRegionLocals(c.Pattern, "", e.ErrorLocal, e.ErrorType, types, module, imports, state); err != nil {
				return err
			}
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
		mergeFlowWithLabels(state, mergedFlow, caseFlow, "previous cases", fmt.Sprintf("case %d", i+1))
		mergedVars = copyRegionVars(state.regionVars)
		mergedFlow = snapshotFlow(state)
	}
	state.regionVars = mergedVars
	restoreFlow(state, mergedFlow)
	return e.ResultType, nil
}

// ownershipArgRef records canonical call-argument access paths in source order.
