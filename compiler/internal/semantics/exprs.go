package semantics

import (
	"fmt"
	"sort"
	"strings"

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
type ownershipArgRef struct {
	path string
	pos  frontend.Position
}

func canonicalOwnershipAccessPath(expr frontend.Expr) (string, bool) {
	base, fields, _, ok := splitOwnershipPath(expr)
	if !ok {
		return "", false
	}
	if len(fields) == 0 || base == "" {
		return base, len(fields) == 0
	}
	path := base
	for _, field := range fields {
		path = joinOwnershipPath(path, field)
	}
	return path, true
}

func splitOwnershipPath(expr frontend.Expr) (string, []string, frontend.Position, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, nil, e.At, true
	case *frontend.FieldAccessExpr:
		base, fields, pos, ok := splitOwnershipPath(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, e.Field)
		return base, fields, e.At, true
	case *frontend.IndexExpr:
		base, fields, pos, ok := splitOwnershipPath(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, splitOwnershipIndexSegment(e.Index))
		return base, fields, e.At, true
	default:
		return "", nil, expr.Pos(), false
	}
}

func splitOwnershipPathSegments(path string) []string {
	if path == "" {
		return nil
	}
	segments := make([]string, 0, 4)
	start := 0
	depth := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || (path[i] == '.' && depth == 0) {
			if i >= start {
				segments = append(segments, path[start:i])
			}
			start = i + 1
			continue
		}
		switch path[i] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		}
	}
	return segments
}

func ownershipPathSegmentsMatch(left string, right string) bool {
	if left == right {
		return true
	}
	if left == "[_]" || right == "[_]" {
		return true
	}
	return false
}

func ownershipPathPrefix(prefix string, path string) bool {
	if prefix == "" || path == "" {
		return false
	}
	prefixParts := splitOwnershipPathSegments(prefix)
	pathParts := splitOwnershipPathSegments(path)
	if len(prefixParts) == 0 || len(prefixParts) > len(pathParts) {
		return false
	}
	for i := 0; i < len(prefixParts); i++ {
		if !ownershipPathSegmentsMatch(prefixParts[i], pathParts[i]) {
			return false
		}
	}
	return true
}

func ownershipPathParent(prefix string) string {
	parts := splitOwnershipPathSegments(prefix)
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], ".")
}

func splitOwnershipIndexSegment(index frontend.Expr) string {
	switch i := index.(type) {
	case *frontend.NumberExpr:
		return fmt.Sprintf("[%d]", i.Value)
	case *frontend.IdentExpr:
		return "[" + i.Name + "]"
	default:
		return "[_]"
	}
}

func joinOwnershipPath(prefix string, segment string) string {
	if segment == "" {
		return prefix
	}
	if strings.HasPrefix(segment, "[") {
		return prefix + segment
	}
	if prefix == "" {
		return segment
	}
	return prefix + "." + segment
}

func consumeLocalArgumentName(expr frontend.Expr, callee string, callback bool, phraseOverride ...string) (string, error) {
	targetPhrase := fmt.Sprintf("'%s'", callee)
	if callback {
		targetPhrase = fmt.Sprintf("callback '%s'", callee)
	}
	if len(phraseOverride) > 0 && phraseOverride[0] != "" {
		targetPhrase = phraseOverride[0]
	}
	path, ok := canonicalOwnershipAccessPath(expr)
	if !ok {
		return "", ownershipDiagnosticf(expr.Pos(), "consume argument for %s must be a local value", targetPhrase)
	}
	return path, nil
}

func checkWholeOwnershipValueAvailable(expr frontend.Expr, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState) error {
	return checkWholeOwnershipValueAvailableForType(expr, "", types, module, imports, state)
}

func checkWholeOwnershipValueAvailableForType(expr frontend.Expr, expectedType string, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState) error {
	if expr == nil || state == nil {
		return nil
	}
	if path, ok := canonicalOwnershipAccessPath(expr); ok {
		if expectedType != "" && typeContainsResourceHandle(expectedType, types) {
			if err := state.checkNoConsumedProperDescendants(path, expr.Pos()); err != nil {
				return err
			}
		} else {
			if err := state.checkNoConsumedDescendants(path, expr.Pos()); err != nil {
				return err
			}
		}
	}
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		typeName, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return err
		}
		info, ok := types[typeName]
		if !ok || info.Kind != TypeStruct {
			return nil
		}
		for _, field := range e.Fields {
			fieldInfo, ok := info.FieldMap[field.Name]
			if !ok {
				continue
			}
			if err := checkWholeOwnershipValueAvailableForType(field.Value, fieldInfo.TypeName, types, module, imports, state); err != nil {
				return err
			}
		}
	case *frontend.CallExpr:
		if e.ResolvedType == "" {
			return nil
		}
		info, ok := types[e.ResolvedType]
		if !ok {
			return nil
		}
		switch info.Kind {
		case TypeStruct:
			for i, field := range info.Fields {
				if i >= len(e.Args) {
					break
				}
				if err := checkWholeOwnershipValueAvailableForType(e.Args[i], field.TypeName, types, module, imports, state); err != nil {
					return err
				}
			}
		case TypeEnum:
			_, caseInfo, found, err := resolveEnumCaseConstructorCall(e, types, module, imports)
			if err != nil {
				return err
			}
			if !found {
				return nil
			}
			for i, arg := range e.Args {
				if i >= len(caseInfo.PayloadTypes) {
					break
				}
				if err := checkWholeOwnershipValueAvailableForType(arg, caseInfo.PayloadTypes[i], types, module, imports, state); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func ownershipAccessPathsAlias(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	return ownershipPathPrefix(left, right) || ownershipPathPrefix(right, left)
}

func findOwnershipAlias(refs []ownershipArgRef, path string) (ownershipArgRef, bool) {
	for _, ref := range refs {
		if ownershipAccessPathsAlias(ref.path, path) {
			return ref, true
		}
	}
	return ownershipArgRef{}, false
}

// checkCallExprWithEffects intentionally keeps call validation in one ordered
// path: resolve local/builtin/imported targets, enforce semantic clauses and
// effects, validate async/throw context, then check arguments, ownership, and
// resource provenance before returning type and region metadata.
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
	if rewritten, err := rewriteSliceViewMethodCall(e, locals, globals, types); rewritten || err != nil {
		if err != nil {
			return "", regionNone, err
		}
	}
	resolved := ""
	isBuiltin := false
	preserveDynamicCallName := false
	functionTypedGlobalCallName := ""
	if fieldInfo, ok, err := resolveFunctionFieldCall(e.Name, locals); err != nil {
		return "", regionNone, err
	} else if ok {
		if analysis != nil && fieldInfo.FunctionTouchesMutableGlobals {
			analysis.touchesMutableGlobals = true
		}
		if len(e.TypeArgs) > 0 {
			return "", regionNone, unsupportedFunctionTypedExplicitTypeArgsError(e.At, fmt.Sprintf("function-typed struct field call '%s'", e.Name))
		}
		if len(e.Args) != len(fieldInfo.FunctionParamTypes) {
			return "", regionNone, fmt.Errorf("%s: wrong argument count for function-typed struct field call '%s'", frontend.FormatPos(e.At), e.Name)
		}
		if err := validateFunctionTypedValueCallLabels(e, "function-typed struct field call", e.Name); err != nil {
			return "", regionNone, err
		}
		if fieldInfo.FunctionValue != "" {
			targetSig, ok := funcs[fieldInfo.FunctionValue]
			if !ok {
				return "", regionNone, fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(e.At), fieldInfo.FunctionValue)
			}
			markFunctionTargetMutableGlobalUse(targetSig, analysis)
			if targetSig.Generic {
				return "", regionNone, fmt.Errorf("%s: generic function symbol '%s' is not supported for function-typed struct field call in this MVP", frontend.FormatPos(e.At), e.Name)
			}
			explicitSlots, err := functionParamSlotCount(fieldInfo.FunctionParamTypes, types)
			if err != nil {
				return "", regionNone, err
			}
			hiddenSlots := targetSig.ParamSlots - explicitSlots
			if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !fieldInfo.FunctionHandleValue) {
				return "", regionNone, unsupportedFunctionFieldCallCaptureError(e.At, e.Name, hiddenSlots)
			}
			if hasCallerSig {
				if err := validateCallAgainstSemanticClauseTarget(callerSig, targetSig, fmt.Sprintf("function-typed struct field call '%s'", e.Name), e.At); err != nil {
					return "", regionNone, err
				}
			}
		} else if hasCallerSig {
			if err := validateFunctionTypeCallableEffects(callerSig.Effects, fieldInfo.FunctionEffects, e.At, "function-typed struct field call", e.Name); err != nil {
				return "", regionNone, err
			}
		}
		consumeArgs := make([]string, len(e.Args))
		consumeArgTypes := make([]string, len(e.Args))
		consumeArgRefs := make([]ownershipArgRef, 0, len(e.Args))
		borrowArgs := make([]ownershipArgRef, 0, len(e.Args))
		inoutArgs := make([]ownershipArgRef, 0, len(e.Args))
		argRegions := make([]int, len(e.Args))
		for i, arg := range e.Args {
			argType, argRegion, err := checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return "", regionNone, err
			}
			argRegions[i] = argRegion
			if !typesCompatibleWithNullPtr(fieldInfo.FunctionParamTypes[i], argType, arg) {
				return "", regionNone, fmt.Errorf("%s: type mismatch for function-typed struct field call '%s' arg %d", frontend.FormatPos(arg.Pos()), e.Name, i+1)
			}
			paramOwnership := ownershipAt(fieldInfo.FunctionParamOwnership, i)
			if paramOwnership == "" {
				if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of function-typed struct field call '%s'", borrowedName, i+1, e.Name)
				}
				if argType == "ptr" {
					if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of function-typed struct field call '%s'", borrowedName, i+1, e.Name)
					}
				}
				if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
					if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
						return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of function-typed struct field call '%s'", borrowedName, i+1, e.Name)
					}); err != nil {
						return "", regionNone, err
					}
				}
				if path, ok := canonicalOwnershipAccessPath(arg); ok {
					if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
						return "", regionNone, err
					}
				}
			}
			if paramOwnership == "consume" {
				name, err := consumeLocalArgumentName(arg, e.Name, true)
				if err != nil {
					return "", regionNone, err
				}
				if err := state.checkNoConsumedDescendants(name, arg.Pos()); err != nil {
					return "", regionNone, err
				}
				if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by function-typed struct field call '%s'", borrowedName, e.Name)
				}
				if argType == "ptr" {
					if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by function-typed struct field call '%s'", borrowedName, e.Name)
					}
				}
				if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
					if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
						return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by function-typed struct field call '%s'", borrowedName, e.Name)
					}); err != nil {
						return "", regionNone, err
					}
				}
				path := name
				if first, exists := findOwnershipAlias(inoutArgs, path); exists {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "consumed argument '%s' aliases inout argument in function-typed struct field call '%s' (inout at %s)", path, e.Name, frontend.FormatPos(first.pos))
				}
				consumeArgs[i] = name
				consumeArgTypes[i] = argType
				consumeArgRefs = append(consumeArgRefs, ownershipArgRef{path: path, pos: arg.Pos()})
			}
			if paramOwnership == "borrow" {
				path, ok := canonicalOwnershipAccessPath(arg)
				if ok {
					if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
						return "", regionNone, err
					}
					if first, exists := findOwnershipAlias(inoutArgs, path); exists {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed argument '%s' aliases inout argument in function-typed struct field call '%s' (inout at %s)", path, e.Name, frontend.FormatPos(first.pos))
					}
					borrowArgs = append(borrowArgs, ownershipArgRef{path: path, pos: arg.Pos()})
				}
			}
			if paramOwnership == "inout" {
				path, ok := canonicalOwnershipAccessPath(arg)
				if !ok {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument for function-typed struct field call '%s' must be a mutable local value", e.Name)
				}
				if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to function-typed struct field call '%s'", borrowedName, e.Name)
				}
				if argType == "ptr" {
					if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to function-typed struct field call '%s'", borrowedName, e.Name)
					}
				}
				if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
					if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
						return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to function-typed struct field call '%s'", borrowedName, e.Name)
					}); err != nil {
						return "", regionNone, err
					}
				}
				if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
					return "", regionNone, err
				}
				targetInfo, _, err := resolveAssignTarget(arg, locals, globals, types)
				if err != nil || !targetInfo.Mutable || targetInfo.Global {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' for function-typed struct field call '%s' must be mutable", path, e.Name)
				}
				if first, exists := findOwnershipAlias(inoutArgs, path); exists {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' used more than once in function-typed struct field call '%s' (first at %s)", path, e.Name, frontend.FormatPos(first.pos))
				}
				if first, exists := findOwnershipAlias(borrowArgs, path); exists {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' aliases borrowed argument in function-typed struct field call '%s' (borrow at %s)", path, e.Name, frontend.FormatPos(first.pos))
				}
				if first, exists := findOwnershipAlias(consumeArgRefs, path); exists {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' aliases consumed argument in function-typed struct field call '%s' (consume at %s)", path, e.Name, frontend.FormatPos(first.pos))
				}
				inoutArgs = append(inoutArgs, ownershipArgRef{path: path, pos: arg.Pos()})
			}
		}
		for i, name := range consumeArgs {
			if name == "" {
				continue
			}
			for j := 0; j < i; j++ {
				if consumeArgs[j] == name {
					return "", regionNone, ownershipDiagnosticf(e.Args[i].Pos(), "value '%s' consumed more than once in function-typed struct field call '%s'", name, e.Name)
				}
				if resourceValuesAlias(consumeArgs[j], consumeArgTypes[j], name, consumeArgTypes[i], types, state) {
					return "", regionNone, ownershipDiagnosticf(e.Args[i].Pos(), "value '%s' consumed more than once in function-typed struct field call '%s'", name, e.Name)
				}
			}
			markConsumedResourceValue(name, consumeArgTypes[i], types, state, e.Args[i].Pos())
		}
		if err := effects.requireAll(e.At, fieldInfo.FunctionEffects); err != nil {
			return "", regionNone, err
		}
		if err := validateFunctionTypedThrowCall(fieldInfo.FunctionThrowsType, e, state); err != nil {
			return "", regionNone, err
		}
		return fieldInfo.FunctionReturnType, functionTypedBorrowReturnRegion(fieldInfo.FunctionReturnOwnership, fieldInfo.FunctionParamOwnership, argRegions, state), nil
	} else if local, ok := locals[e.Name]; ok {
		if analysis != nil && local.FunctionTouchesMutableGlobals {
			analysis.touchesMutableGlobals = true
		}
		if local.FunctionEnumPayload && local.FunctionValue != "" && local.FunctionTypeValue {
			targetSig, ok := funcs[local.FunctionValue]
			if !ok {
				return "", regionNone, fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(e.At), local.FunctionValue)
			}
			markFunctionTargetMutableGlobalUse(targetSig, analysis)
			explicitSlots, err := functionParamSlotCount(local.FunctionParamTypes, types)
			if err != nil {
				return "", regionNone, err
			}
			hiddenSlots := targetSig.ParamSlots - explicitSlots
			if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !local.FunctionHandleValue) {
				return "", regionNone, unsupportedEnumPayloadCallCaptureError(e.At, e.Name, hiddenSlots)
			}
		}
		if local.FunctionValue == "" || (local.FunctionTypeValue && len(local.FunctionCaptures) == 0 && local.SlotCount == FnPtrSlotCount) {
			if !local.FunctionTypeValue {
				return "", regionNone, unsupportedFunctionValueCallError(e.At, e.Name)
			}
			if len(local.FunctionCaptures) > 0 {
				return "", regionNone, fmt.Errorf("%s: function-typed callback '%s' captures local values; captured function values cannot be called through function type in this MVP", frontend.FormatPos(e.At), e.Name)
			}
			valueCallKind := "callback"
			valueCallPhrase := fmt.Sprintf("callback '%s'", e.Name)
			if local.FunctionEnumPayload {
				valueCallKind = "function-typed enum payload call"
				valueCallPhrase = fmt.Sprintf("function-typed enum payload call '%s'", e.Name)
			}
			if len(e.TypeArgs) > 0 {
				return "", regionNone, unsupportedFunctionTypedExplicitTypeArgsError(e.At, valueCallPhrase)
			}
			if len(e.Args) != len(local.FunctionParamTypes) {
				return "", regionNone, fmt.Errorf("%s: wrong argument count for %s", frontend.FormatPos(e.At), valueCallPhrase)
			}
			if err := validateFunctionTypedValueCallLabels(e, valueCallKind, e.Name); err != nil {
				return "", regionNone, err
			}
			consumeArgs := make([]string, len(e.Args))
			consumeArgTypes := make([]string, len(e.Args))
			consumeArgRefs := make([]ownershipArgRef, 0, len(e.Args))
			borrowArgs := make([]ownershipArgRef, 0, len(e.Args))
			inoutArgs := make([]ownershipArgRef, 0, len(e.Args))
			argRegions := make([]int, len(e.Args))
			for i, arg := range e.Args {
				argType, argRegion, err := checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return "", regionNone, err
				}
				argRegions[i] = argRegion
				if !typesCompatibleWithNullPtr(local.FunctionParamTypes[i], argType, arg) {
					return "", regionNone, fmt.Errorf("%s: type mismatch for %s arg %d", frontend.FormatPos(arg.Pos()), valueCallPhrase, i+1)
				}
				paramOwnership := ownershipAt(local.FunctionParamOwnership, i)
				if paramOwnership == "" {
					if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s", borrowedName, i+1, valueCallPhrase)
					}
					if argType == "ptr" {
						if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
							return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s", borrowedName, i+1, valueCallPhrase)
						}
					}
					if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
						if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
							return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s", borrowedName, i+1, valueCallPhrase)
						}); err != nil {
							return "", regionNone, err
						}
					}
					if path, ok := canonicalOwnershipAccessPath(arg); ok {
						if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
							return "", regionNone, err
						}
					}
				}
				if paramOwnership == "consume" {
					name, err := consumeLocalArgumentName(arg, e.Name, true)
					if err != nil {
						return "", regionNone, err
					}
					if err := state.checkNoConsumedDescendants(name, arg.Pos()); err != nil {
						return "", regionNone, err
					}
					if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by %s", borrowedName, valueCallPhrase)
					}
					if argType == "ptr" {
						if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
							return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by %s", borrowedName, valueCallPhrase)
						}
					}
					if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
						if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
							return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by %s", borrowedName, valueCallPhrase)
						}); err != nil {
							return "", regionNone, err
						}
					}
					path := name
					if first, exists := findOwnershipAlias(inoutArgs, path); exists {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "consumed argument '%s' aliases inout argument in %s (inout at %s)", path, valueCallPhrase, frontend.FormatPos(first.pos))
					}
					consumeArgs[i] = name
					consumeArgTypes[i] = argType
					consumeArgRefs = append(consumeArgRefs, ownershipArgRef{path: path, pos: arg.Pos()})
				}
				if paramOwnership == "borrow" {
					path, ok := canonicalOwnershipAccessPath(arg)
					if ok {
						if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
							return "", regionNone, err
						}
						if first, exists := findOwnershipAlias(inoutArgs, path); exists {
							return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed argument '%s' aliases inout argument in %s (inout at %s)", path, valueCallPhrase, frontend.FormatPos(first.pos))
						}
						borrowArgs = append(borrowArgs, ownershipArgRef{path: path, pos: arg.Pos()})
					}
				}
				if paramOwnership == "inout" {
					path, ok := canonicalOwnershipAccessPath(arg)
					if !ok {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument for %s must be a mutable local value", valueCallPhrase)
					}
					if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to %s", borrowedName, valueCallPhrase)
					}
					if argType == "ptr" {
						if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
							return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to %s", borrowedName, valueCallPhrase)
						}
					}
					if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
						if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
							return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to %s", borrowedName, valueCallPhrase)
						}); err != nil {
							return "", regionNone, err
						}
					}
					if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
						return "", regionNone, err
					}
					targetInfo, _, err := resolveAssignTarget(arg, locals, globals, types)
					if err != nil || !targetInfo.Mutable || targetInfo.Global {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' for %s must be mutable", path, valueCallPhrase)
					}
					if first, exists := findOwnershipAlias(inoutArgs, path); exists {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' used more than once in %s (first at %s)", path, valueCallPhrase, frontend.FormatPos(first.pos))
					}
					if first, exists := findOwnershipAlias(borrowArgs, path); exists {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' aliases borrowed argument in %s (borrow at %s)", path, valueCallPhrase, frontend.FormatPos(first.pos))
					}
					if first, exists := findOwnershipAlias(consumeArgRefs, path); exists {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' aliases consumed argument in %s (consume at %s)", path, valueCallPhrase, frontend.FormatPos(first.pos))
					}
					inoutArgs = append(inoutArgs, ownershipArgRef{path: path, pos: arg.Pos()})
				}
			}
			for i, name := range consumeArgs {
				if name == "" {
					continue
				}
				for j := 0; j < i; j++ {
					if consumeArgs[j] == name {
						return "", regionNone, ownershipDiagnosticf(e.Args[i].Pos(), "value '%s' consumed more than once in %s", name, valueCallPhrase)
					}
					if resourceValuesAlias(consumeArgs[j], consumeArgTypes[j], name, consumeArgTypes[i], types, state) {
						return "", regionNone, ownershipDiagnosticf(e.Args[i].Pos(), "value '%s' consumed more than once in %s", name, valueCallPhrase)
					}
				}
				markConsumedResourceValue(name, consumeArgTypes[i], types, state, e.Args[i].Pos())
			}
			if hasCallerSig && hasStrictSemanticCallClauses(callerSig) {
				if local.FunctionValue == "" {
					paramSlots, err := functionParamSlotCount(local.FunctionParamTypes, types)
					if err != nil {
						return "", regionNone, err
					}
					declaredSig := FuncSig{
						ParamTypes:     append([]string(nil), local.FunctionParamTypes...),
						ParamOwnership: append([]string(nil), local.FunctionParamOwnership...),
						ParamSlots:     paramSlots,
						ReturnType:     local.FunctionReturnType,
						Effects:        append([]string(nil), local.FunctionEffects...),
					}
					semanticCallPhrase := fmt.Sprintf("call to '%s'", e.Name)
					if local.FunctionEnumPayload {
						semanticCallPhrase = valueCallPhrase
					}
					if err := validateCallAgainstSemanticClauseTarget(callerSig, declaredSig, semanticCallPhrase, e.At); err != nil {
						return "", regionNone, err
					}
				} else {
					targetSig, ok := funcs[local.FunctionValue]
					if !ok {
						return "", regionNone, fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(e.At), local.FunctionValue)
					}
					semanticCallPhrase := fmt.Sprintf("call to callback '%s'", e.Name)
					if local.FunctionEnumPayload {
						semanticCallPhrase = valueCallPhrase
					}
					if err := validateCallAgainstSemanticClauseTarget(callerSig, targetSig, semanticCallPhrase, e.At); err != nil {
						return "", regionNone, err
					}
				}
			}
			if err := effects.requireAll(e.At, local.FunctionEffects); err != nil {
				return "", regionNone, err
			}
			if local.FunctionValue != "" {
				if targetSig, ok := funcs[local.FunctionValue]; ok {
					markFunctionTargetMutableGlobalUse(targetSig, analysis)
				}
			}
			if err := validateFunctionTypedThrowCall(local.FunctionThrowsType, e, state); err != nil {
				return "", regionNone, err
			}
			return local.FunctionReturnType, functionTypedBorrowReturnRegion(local.FunctionReturnOwnership, local.FunctionParamOwnership, argRegions, state), nil
		}
		if local.GenericFunctionValue {
			return "", regionNone, unsupportedGenericClosureDirectCallError(e.At, e.Name)
		}
		if local.FunctionTypeValue && local.FunctionReturnType != "" {
			targetSig, ok := funcs[local.FunctionValue]
			if !ok {
				return "", regionNone, fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(e.At), local.FunctionValue)
			}
			markFunctionTargetMutableGlobalUse(targetSig, analysis)
			explicitSlots, err := functionParamSlotCount(local.FunctionParamTypes, types)
			if err != nil {
				return "", regionNone, err
			}
			hiddenSlots := targetSig.ParamSlots - explicitSlots
			if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !local.FunctionHandleValue) {
				if local.FunctionEnumPayload {
					return "", regionNone, unsupportedEnumPayloadCallCaptureError(e.At, e.Name, hiddenSlots)
				} else {
					return "", regionNone, unsupportedFunctionTypedCallCaptureError(e.At, e.Name, hiddenSlots)
				}
			}
			if hiddenSlots > 0 || local.Mutable {
				valueCallPhrase := fmt.Sprintf("function-typed callback '%s'", e.Name)
				if len(e.TypeArgs) > 0 {
					return "", regionNone, unsupportedFunctionTypedExplicitTypeArgsError(e.At, valueCallPhrase)
				}
				if len(e.Args) != len(local.FunctionParamTypes) {
					return "", regionNone, fmt.Errorf("%s: wrong argument count for %s", frontend.FormatPos(e.At), valueCallPhrase)
				}
				if err := validateFunctionTypedValueCallLabels(e, "function-typed callback", e.Name); err != nil {
					return "", regionNone, err
				}
				if hasCallerSig {
					if err := validateCallAgainstSemanticClauseTarget(callerSig, targetSig, valueCallPhrase, e.At); err != nil {
						return "", regionNone, err
					}
				}
				for i, arg := range e.Args {
					argType, _, err := checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
					if err != nil {
						return "", regionNone, err
					}
					if !typesCompatibleWithNullPtr(local.FunctionParamTypes[i], argType, arg) {
						return "", regionNone, fmt.Errorf("%s: type mismatch for %s arg %d", frontend.FormatPos(arg.Pos()), valueCallPhrase, i+1)
					}
				}
				if err := effects.requireAll(e.At, local.FunctionEffects); err != nil {
					return "", regionNone, err
				}
				if err := validateFunctionTypedThrowCall(local.FunctionThrowsType, e, state); err != nil {
					return "", regionNone, err
				}
				return local.FunctionReturnType, regionNone, nil
			}
		}
		if err := appendClosureCaptureArgs(e, local); err != nil {
			return "", regionNone, err
		}
		resolved = local.FunctionValue
		e.Name = resolved
	} else if global, ok := globals[e.Name]; ok && global.FunctionTypeValue {
		if analysis != nil && global.Mutable {
			analysis.touchesMutableGlobals = true
		}
		if global.FunctionValue == "" {
			if global.Mutable {
				return "", regionNone, unsupportedImportedMutableFunctionTypedGlobalCallError(e.At, e.Name)
			}
			return "", regionNone, unsupportedFunctionTypedGlobalTargetError(e.At, e.Name)
		}
		targetSig, ok := funcs[global.FunctionValue]
		if !ok {
			return "", regionNone, fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(e.At), global.FunctionValue)
		}
		targetInfo := LocalInfo{
			TypeName:               global.TypeName,
			FunctionValue:          global.FunctionValue,
			FunctionTypeValue:      true,
			FunctionParamTypes:     append([]string(nil), global.FunctionParamTypes...),
			FunctionParamOwnership: append([]string(nil), global.FunctionParamOwnership...),
			FunctionReturnType:     global.FunctionReturnType,
			FunctionThrowsType:     global.FunctionThrowsType,
			FunctionEffects:        append([]string(nil), global.FunctionEffects...),
		}
		if err := validateFunctionInfoAssignable(e.Name, targetInfo, targetSig, e.At); err != nil {
			return "", regionNone, err
		}
		functionTypedGlobalCallName = e.Name
		resolved = global.FunctionValue
		if !global.Mutable {
			e.Name = resolved
		} else {
			preserveDynamicCallName = true
		}
	} else if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
		resolved = builtin
		isBuiltin = true
	} else if _, ok := funcs[e.Name]; ok {
		resolved = e.Name
	} else {
		var err error
		resolved, err = resolveKnownCallName(e.Name, funcs, module, imports, e.At)
		if err != nil {
			if diagnostic, ok := atomicBuiltinDiagnostic(e.Name); ok {
				return "", regionNone, fmt.Errorf("%s: %s", frontend.FormatPos(e.At), diagnostic)
			}
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
		if diagnostic, ok := atomicBuiltinDiagnostic(resolved); ok {
			return "", regionNone, fmt.Errorf("%s: %s", frontend.FormatPos(e.At), diagnostic)
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
		if functionTypedGlobalCallName != "" {
			return "", regionNone, unsupportedFunctionTypedExplicitTypeArgsError(e.At, fmt.Sprintf("function-typed global call '%s'", functionTypedGlobalCallName))
		}
		return "", regionNone, fmt.Errorf("%s: explicit type arguments are only supported for recv_typed", frontend.FormatPos(e.At))
	}
	if hasCallerSig {
		semanticCallPhrase := fmt.Sprintf("call to '%s'", resolved)
		if functionTypedGlobalCallName != "" {
			semanticCallPhrase = fmt.Sprintf("function-typed global call '%s'", functionTypedGlobalCallName)
		}
		if err := validateCallAgainstSemanticClauseTarget(callerSig, sig, semanticCallPhrase, e.At); err != nil {
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
	callTargetPhrase := fmt.Sprintf("'%s'", resolved)
	callActionPhrase := fmt.Sprintf("call to '%s'", resolved)
	if functionTypedGlobalCallName != "" {
		callTargetPhrase = fmt.Sprintf("function-typed global call '%s'", functionTypedGlobalCallName)
		callActionPhrase = callTargetPhrase
	}
	if len(e.Args) != len(sig.ParamTypes) {
		return "", regionNone, fmt.Errorf("%s: wrong argument count for %s", frontend.FormatPos(e.At), callTargetPhrase)
	}
	if len(e.ArgLabels) > 0 && functionTypedGlobalCallName != "" {
		if err := validateFunctionTypedValueCallLabels(e, "function-typed global call", functionTypedGlobalCallName); err != nil {
			return "", regionNone, err
		}
	} else if len(e.ArgLabels) > 0 {
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
	ownershipTargetPhrase := callTargetPhrase
	ownershipCallPhrase := callActionPhrase
	consumeTargetPhrase := ""
	if functionTypedGlobalCallName != "" {
		consumeTargetPhrase = ownershipTargetPhrase
	}
	argRegions := make([]int, len(e.Args))
	consumeArgs := make([]string, len(e.Args))
	consumeArgTypes := make([]string, len(e.Args))
	consumeArgRefs := make([]ownershipArgRef, 0, len(e.Args))
	borrowArgs := make([]ownershipArgRef, 0, len(e.Args))
	inoutArgs := make([]ownershipArgRef, 0, len(e.Args))
	for i, arg := range e.Args {
		argType := ""
		argRegion := regionNone
		callbackParam := i < len(sig.ParamFunctionTypes) && sig.ParamFunctionTypes[i]
		localCallbackArg := false
		globalCallbackArg := false
		fieldCallbackArg := false
		if callbackParam {
			if id, ok := arg.(*frontend.IdentExpr); ok {
				if _, exists := locals[id.Name]; exists {
					localCallbackArg = true
				}
				if global, exists := globals[id.Name]; exists && global.FunctionTypeValue {
					globalCallbackArg = true
				}
			} else if _, ok, err := resolveFunctionFieldArgument(arg, locals); err != nil {
				return "", regionNone, err
			} else if ok {
				fieldCallbackArg = true
			} else if fieldAccess, ok := arg.(*frontend.FieldAccessExpr); ok {
				if _, _, globalOK, err := resolveFunctionTypedGlobalFieldAccess(fieldAccess, globals, funcs); err != nil {
					return "", regionNone, err
				} else if globalOK {
					globalCallbackArg = true
				}
			}
			callbackType, callbackSymbol, err := resolveCallbackArgumentType(arg, resolved, sig, i, locals, globals, funcs, types, module, imports, state, effects, analysis, hasCallerSig && hasStrictSemanticCallClauses(callerSig))
			if err != nil {
				return "", regionNone, err
			}
			if hasCallerSig {
				if callbackSymbol == "" {
					if hasStrictSemanticCallClauses(callerSig) {
						return "", regionNone, unsupportedCallbackUnknownSemanticTargetError(arg.Pos(), resolved, firstStrictSemanticCallClause(callerSig))
					}
				} else if callbackSig, ok := funcs[callbackSymbol]; ok {
					callbackPhrase := fmt.Sprintf("call to '%s'", callbackSymbol)
					if localCallbackArg || globalCallbackArg || fieldCallbackArg {
						if name := callbackArgumentName(arg); name != "" {
							callbackPhrase = fmt.Sprintf("callback argument '%s'", name)
						}
					}
					if err := validateCallAgainstSemanticClauseTarget(callerSig, callbackSig, callbackPhrase, arg.Pos()); err != nil {
						return "", regionNone, err
					}
				}
			}
			if callbackSymbol != "" {
				if callbackSig, ok := funcs[callbackSymbol]; ok {
					if analysis != nil && callbackSig.TouchesMutableGlobals {
						analysis.touchesMutableGlobals = true
					}
					if err := effects.requireAll(arg.Pos(), callbackSig.Effects); err != nil {
						return "", regionNone, err
					}
				}
				if id, ok := arg.(*frontend.IdentExpr); ok && !localCallbackArg && !globalCallbackArg && !fieldCallbackArg {
					// Keep lowered target collection deterministic across modules/import aliases.
					id.Name = callbackSymbol
				}
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
			return "", regionNone, fmt.Errorf("%s: type mismatch for %s arg %d", frontend.FormatPos(arg.Pos()), callTargetPhrase, i+1)
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
				return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s", borrowedName, i+1, ownershipTargetPhrase)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s", borrowedName, i+1, ownershipTargetPhrase)
				}
			}
			if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
					return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s", borrowedName, i+1, ownershipTargetPhrase)
				}); err != nil {
					return "", regionNone, err
				}
			}
			if path, ok := canonicalOwnershipAccessPath(arg); ok {
				if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
					return "", regionNone, err
				}
			}
		}
		if paramOwnership == "consume" {
			name, err := consumeLocalArgumentName(arg, resolved, false, consumeTargetPhrase)
			if err != nil {
				return "", regionNone, err
			}
			if err := state.checkNoConsumedDescendants(name, arg.Pos()); err != nil {
				return "", regionNone, err
			}
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by %s", borrowedName, ownershipTargetPhrase)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by %s", borrowedName, ownershipTargetPhrase)
				}
			}
			if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
					return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by %s", borrowedName, ownershipTargetPhrase)
				}); err != nil {
					return "", regionNone, err
				}
			}
			path := name
			if first, exists := findOwnershipAlias(inoutArgs, path); exists {
				return "", regionNone, ownershipDiagnosticf(arg.Pos(), "consumed argument '%s' aliases inout argument in %s (inout at %s)", path, ownershipCallPhrase, frontend.FormatPos(first.pos))
			}
			consumeArgs[i] = name
			consumeArgTypes[i] = argType
			consumeArgRefs = append(consumeArgRefs, ownershipArgRef{path: path, pos: arg.Pos()})
		}
		if paramOwnership == "borrow" {
			path, ok := canonicalOwnershipAccessPath(arg)
			if ok {
				if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
					return "", regionNone, err
				}
				if first, exists := findOwnershipAlias(inoutArgs, path); exists {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed argument '%s' aliases inout argument in %s (inout at %s)", path, ownershipCallPhrase, frontend.FormatPos(first.pos))
				}
				borrowArgs = append(borrowArgs, ownershipArgRef{path: path, pos: arg.Pos()})
			}
		}
		if paramOwnership == "inout" {
			path, ok := canonicalOwnershipAccessPath(arg)
			if !ok {
				return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument for %s must be a mutable local value", ownershipTargetPhrase)
			}
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to %s", borrowedName, ownershipTargetPhrase)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to %s", borrowedName, ownershipTargetPhrase)
				}
			}
			if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
					return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to %s", borrowedName, ownershipTargetPhrase)
				}); err != nil {
					return "", regionNone, err
				}
			}
			if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
				return "", regionNone, err
			}
			targetInfo, _, err := resolveAssignTarget(arg, locals, globals, types)
			if err != nil || !targetInfo.Mutable || targetInfo.Global {
				return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' for %s must be mutable", path, ownershipTargetPhrase)
			}
			if first, exists := findOwnershipAlias(inoutArgs, path); exists {
				return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' used more than once in %s (first at %s)", path, ownershipCallPhrase, frontend.FormatPos(first.pos))
			}
			if first, exists := findOwnershipAlias(borrowArgs, path); exists {
				return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' aliases borrowed argument in %s (borrow at %s)", path, ownershipCallPhrase, frontend.FormatPos(first.pos))
			}
			if first, exists := findOwnershipAlias(consumeArgRefs, path); exists {
				return "", regionNone, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' aliases consumed argument in %s (consume at %s)", path, ownershipCallPhrase, frontend.FormatPos(first.pos))
			}
			inoutArgs = append(inoutArgs, ownershipArgRef{path: path, pos: arg.Pos()})
		}
		argRegions[i] = argRegion
	}
	for i, name := range consumeArgs {
		if name == "" {
			continue
		}
		for j := 0; j < i; j++ {
			if consumeArgs[j] == name {
				return "", regionNone, ownershipDiagnosticf(e.Args[i].Pos(), "value '%s' consumed more than once in %s", name, ownershipCallPhrase)
			}
			if resourceValuesAlias(consumeArgs[j], consumeArgTypes[j], name, consumeArgTypes[i], types, state) {
				return "", regionNone, ownershipDiagnosticf(e.Args[i].Pos(), "value '%s' consumed more than once in %s", name, ownershipCallPhrase)
			}
		}
		markConsumedResourceValue(name, consumeArgTypes[i], types, state, e.Args[i].Pos())
	}
	if handleArg, ok := surfaceHostABIHandleArgIndex(resolved); ok && handleArg < len(e.Args) {
		if owner, ok := surfaceHandleOwnerPathExprWithAnalysis(e.Args[handleArg], locals, globals, types, analysis); ok {
			if err := state.checkNotConsumed(owner, e.Args[handleArg].Pos()); err != nil {
				return "", regionNone, err
			}
		}
	}
	if resolved == "core.surface_close" && len(e.Args) > 0 {
		if owner, ok := surfaceHandleOwnerPathExprWithAnalysis(e.Args[0], locals, globals, types, analysis); ok {
			markConsumedResourceValue(owner, surfaceSurfaceTypeName, types, state, e.Args[0].Pos())
		}
	}
	if resolved == "core.surface_present_rgba" && len(e.Args) > 1 {
		if frameName, ok := surfaceFramePixelsSourceExpr(e.Args[1], locals, globals, types, analysis); ok && frameName != "" {
			if err := checkSurfacePresentFrameOwnerPath(frameName, analysis, state, e.Args[1].Pos()); err != nil {
				return "", regionNone, err
			}
			analysis.markSurfaceFramePresented(frameName, e.Args[1].Pos())
			markConsumedResourceValue(frameName, surfaceFrameTypeName, types, state, e.Args[1].Pos())
		}
	}
	if isSurfacePresentCallName(resolved) && len(e.Args) > 0 {
		if err := checkSurfacePresentFrameOwner(e.Args[0], analysis, state, e.Args[0].Pos()); err != nil {
			return "", regionNone, err
		}
		if frameName, ok := surfacePresentedFrameArg(e.Args[0]); ok {
			analysis.markSurfaceFramePresented(frameName, e.Args[0].Pos())
		}
	}
	markCallFinalizedResources(resolved, e, funcs, module, imports, state)
	if isTryCall {
		if err := recordTryCallThrowResourceSummary(e, sig, funcs, types, module, imports, state); err != nil {
			return "", regionNone, err
		}
	}
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
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
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
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf("%s: spawn target '%s' uses effect '%s' and cannot cross actor boundary", frontend.FormatPos(e.At), target, blocked)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf("%s: spawn target '%s' is not sendable across actor boundary", frontend.FormatPos(e.At), target)
		}
		lit.Value = []byte(target)
	}
	if resolved == "core.spawn_remote" {
		if len(e.Args) != 2 {
			return "", regionNone, fmt.Errorf("%s: spawn_remote expects 2 arguments", frontend.FormatPos(e.At))
		}
		lit, ok := e.Args[1].(*frontend.StringLitExpr)
		if !ok {
			return "", regionNone, fmt.Errorf("%s: spawn_remote expects a string literal", frontend.FormatPos(e.At))
		}
		raw := string(lit.Value)
		if raw == "" {
			return "", regionNone, fmt.Errorf("%s: spawn_remote expects a non-empty name", frontend.FormatPos(e.At))
		}
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf("%s: spawn_remote target must be a user function, got '%s'", frontend.FormatPos(e.At), target)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), target)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			return "", regionNone, fmt.Errorf("%s: spawn_remote target must have shape fun %s(): i32", frontend.FormatPos(e.At), target)
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf("%s: spawn_remote target must be synchronous", frontend.FormatPos(e.At))
		}
		if targetSig.ThrowsType != "" {
			return "", regionNone, fmt.Errorf("%s: spawn_remote target must not throw", frontend.FormatPos(e.At))
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf("%s: spawn_remote target '%s' touches mutable global state and cannot cross actor boundary", frontend.FormatPos(e.At), target)
		}
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf("%s: spawn_remote target '%s' uses effect '%s' and cannot cross actor boundary", frontend.FormatPos(e.At), target, blocked)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf("%s: spawn_remote target '%s' is not sendable across actor boundary", frontend.FormatPos(e.At), target)
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
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
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
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32 target '%s' uses effect '%s' and cannot cross task boundary", frontend.FormatPos(e.At), target, blocked)
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
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
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
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32 target '%s' uses effect '%s' and cannot cross task boundary", frontend.FormatPos(e.At), target, blocked)
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
		return "", regionNone, effectDiagnosticf(e.At, "'%s' is only allowed in unsafe blocks", resolved)
	}
	if permission, attenuatedEffect := builtinCapsulePermission(resolved); permission != "" {
		if err := effects.requireCapsulePermission(e.At, permission, attenuatedEffect); err != nil {
			return "", regionNone, err
		}
	}
	if !preserveDynamicCallName {
		e.Name = resolved
	}
	regionID := regionNone
	if isExplicitBorrowBuiltin(resolved) && len(e.Args) == 1 {
		owner := borrowOwnerNameFromExpr(e.Args[0])
		if len(argRegions) > 0 {
			if _, borrowed := state.borrowedParamOwner(argRegions[0]); borrowed {
				return sig.ReturnType, argRegions[0], nil
			}
		}
		regionID = state.bindExplicitBorrow(owner)
		return sig.ReturnType, regionID, nil
	}
	if len(sig.ReturnRegionSummary) > 0 {
		tree := make(map[string]int, len(sig.ReturnRegionSummary))
		for leaf, paramIndex := range sig.ReturnRegionSummary {
			if paramIndex < 0 || paramIndex >= len(argRegions) {
				return "", regionNone, fmt.Errorf("%s: invalid region signature for '%s'", frontend.FormatPos(e.At), resolved)
			}
			leafRegion := argRegions[paramIndex]
			if leafRegion == regionUnknown {
				return "", regionNone, fmt.Errorf("%s: ambiguous region for '%s' return", frontend.FormatPos(e.At), resolved)
			}
			if leafRegion != regionNone {
				tree[leaf] = leafRegion
			}
		}
		state.setExprRegionTree(e, tree)
		regionID = constructorRegionFromTree(tree)
	} else if sig.ReturnRegionParam >= 0 {
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

func functionTypedBorrowReturnRegion(returnOwnership string, paramOwnership []string, argRegions []int, state *regionState) int {
	if returnOwnership != "borrow" {
		return regionNone
	}
	regionID := regionNone
	seen := false
	for i, ownership := range paramOwnership {
		if ownership != "borrow" || i >= len(argRegions) {
			continue
		}
		argRegion := argRegions[i]
		if argRegion == regionNone {
			continue
		}
		if argRegion == regionUnknown {
			return state.bindExplicitBorrow("<borrow>")
		}
		if _, borrowed := borrowedOwnerForRegion(argRegion, state); !borrowed {
			continue
		}
		if !seen {
			regionID = argRegion
			seen = true
			continue
		}
		if regionID != argRegion {
			return state.bindExplicitBorrow("<borrow>")
		}
	}
	return regionID
}

func isExplicitBorrowBuiltin(name string) bool {
	if target, ok := ResolveBuiltinAlias(name); ok {
		name = target
	}
	if name == "core.string_borrow" {
		return true
	}
	return strings.HasPrefix(name, "core.slice_borrow_")
}

func isExplicitCopyBuiltin(name string) bool {
	if target, ok := ResolveBuiltinAlias(name); ok {
		name = target
	}
	if name == "core.string_copy" {
		return true
	}
	return strings.HasPrefix(name, "core.slice_copy_") && !strings.HasPrefix(name, "core.slice_copy_into_")
}

func borrowOwnerNameFromExpr(expr frontend.Expr) string {
	if expr == nil {
		return "<borrow>"
	}
	if path, ok := resourcePathForExpr(expr); ok && path != "" {
		return path
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		return borrowOwnerNameFromExpr(e.Base)
	case *frontend.IndexExpr:
		return borrowOwnerNameFromExpr(e.Base)
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := ResolveBuiltinAlias(name); ok {
			name = target
		}
		if isExplicitBorrowBuiltin(name) || isExplicitCopyBuiltin(name) {
			if len(e.Args) > 0 {
				return borrowOwnerNameFromExpr(e.Args[0])
			}
		}
		if _, _, ok := sliceViewElemFromBuiltin(name); ok && len(e.Args) > 0 {
			return borrowOwnerNameFromExpr(e.Args[0])
		}
	}
	return "<borrow>"
}

func validateFunctionTypedValueCallLabels(e *frontend.CallExpr, kind, name string) error {
	if len(e.ArgLabels) == 0 {
		return nil
	}
	if len(e.ArgLabels) != len(e.Args) {
		return fmt.Errorf("%s: internal error: call argument labels are inconsistent", frontend.FormatPos(e.At))
	}
	for i, label := range e.ArgLabels {
		if label == "" {
			return fmt.Errorf("%s: cannot mix labeled and unlabeled arguments in %s '%s'", frontend.FormatPos(e.Args[i].Pos()), kind, name)
		}
	}
	return nil
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
	return validateCallAgainstSemanticClauseTarget(callerSig, calleeSig, fmt.Sprintf("call to '%s'", calleeName), pos)
}

func validateCallAgainstSemanticClauseTarget(callerSig FuncSig, calleeSig FuncSig, calleePhrase string, pos frontend.Position) error {
	if err := validateBudgetedSemanticCallTarget(callerSig, calleeSig, calleePhrase, pos); err != nil {
		return err
	}
	if callerSig.HasRealtime {
		if blocked := firstFuncSigForbiddenEffect(calleeSig, realtimeForbiddenCallEffects); blocked != "" {
			return fmt.Errorf("%s: semantic clause 'realtime' forbids %s because it is not realtime-safe (effect '%s')", frontend.FormatPos(pos), calleePhrase, blocked)
		}
	}
	if callerSig.HasNoAlloc && funcSigHasEffect(calleeSig, "alloc") {
		return fmt.Errorf("%s: semantic clause 'noalloc' forbids %s because it may allocate", frontend.FormatPos(pos), calleePhrase)
	}
	if callerSig.HasNoBlock {
		if blocked := firstFuncSigForbiddenEffect(calleeSig, noblockForbiddenCallEffects); blocked != "" {
			return fmt.Errorf("%s: semantic clause 'noblock' forbids %s because it may block (effect '%s')", frontend.FormatPos(pos), calleePhrase, blocked)
		}
	}
	return nil
}

func validateBudgetedSemanticCallTarget(callerSig FuncSig, calleeSig FuncSig, calleePhrase string, pos frontend.Position) error {
	if !calleeSig.HasBudget {
		return nil
	}
	required := calleeSig.Budget
	if !callerSig.HasBudget {
		return budgetDiagnosticf(pos, "budget context for %s requires caller budget at least %d", calleePhrase, required)
	}
	if callerSig.Budget < required {
		return budgetDiagnosticf(pos, "budget context for %s requires caller budget at least %d, got %d", calleePhrase, required, callerSig.Budget)
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
	if err := validateBudgetedSemanticCallTarget(calleeSig, callbackSig, fmt.Sprintf("callback function symbol '%s' for callee '%s'", callbackName, calleeName), pos); err != nil {
		return err
	}
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

func actorTaskWorkerBoundaryEffect(sig FuncSig) string {
	for _, effect := range sig.Effects {
		switch effect {
		case "alloc", "capability", "control", "islands", "link", "mem", "mmio", "privacy":
			return effect
		}
	}
	return ""
}

func firstFuncSigForbiddenEffect(sig FuncSig, forbidden []string) string {
	effects := effectSet(sig.Effects)
	return firstForbiddenEffect(effects, forbidden)
}

func hasStrictSemanticCallClauses(sig FuncSig) bool {
	return sig.HasNoAlloc || sig.HasNoBlock || sig.HasRealtime || sig.HasBudget
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
	if sig.HasBudget {
		return "budget"
	}
	return ""
}

func resolveCallbackArgumentType(
	arg frontend.Expr,
	calleeName string,
	calleeSig FuncSig,
	paramIndex int,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
	deferEffectValidation bool,
) (string, string, error) {
	if closure, ok := arg.(*frontend.ClosureExpr); ok {
		targetInfo := functionParamLocalInfo(calleeSig, paramIndex)
		if err := validateFunctionTypedClosureAssignment("closure literal", targetInfo, closure, locals, funcs, types, module, imports, closure.At, "callback argument"); err != nil {
			return "", "", err
		}
		if len(closure.Captures) > 0 {
			captureSlots, err := functionCaptureSlotCount(closure.Captures, types)
			if err != nil {
				return "", "", err
			}
			if captureSlots > FnPtrEnvSlotCount {
				if _, _, err := classifyCallableEscape(callableBoundaryCallback, closure.Captures, types); err != nil {
					return "", "", err
				}
			}
		}
		callbackSig, ok := funcs[closure.Name]
		if !ok {
			callbackSig = FuncSig{
				ParamTypes:     append([]string(nil), targetInfo.FunctionParamTypes...),
				ParamOwnership: append([]string(nil), targetInfo.FunctionParamOwnership...),
				ReturnType:     targetInfo.FunctionReturnType,
				Effects:        append([]string(nil), targetInfo.FunctionEffects...),
			}
		}
		if !deferEffectValidation {
			if err := validateCallbackClauseCompatibility(callbackSig, calleeSig, calleeName, closure.At, callbackArgumentName(arg)); err != nil {
				return "", "", err
			}
		}
		return targetInfo.TypeName, closure.Name, nil
	}
	if call, ok := arg.(*frontend.CallExpr); ok {
		argType, _, err := checkExprWithEffects(call, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", "", err
		}
		callSig, ok := funcs[call.Name]
		if !ok || !callSig.ReturnFunctionType {
			return "", "", fmt.Errorf("%s: callback argument call '%s' does not return a function type", frontend.FormatPos(call.At), call.Name)
		}
		if err := markFunctionTypedReturnCallMutableGlobalUse(callSig, call, locals, globals, funcs, types, module, imports, analysis); err != nil {
			return "", "", err
		}
		returnedSig := FuncSig{
			ParamTypes:     append([]string(nil), callSig.ReturnFunctionParams...),
			ParamOwnership: append([]string(nil), callSig.ReturnFunctionParamOwnership...),
			ReturnType:     callSig.ReturnFunctionReturn,
			ThrowsType:     callSig.ReturnFunctionThrows,
			Effects:        append([]string(nil), callSig.ReturnFunctionEffects...),
		}
		if err := validateCallbackSignature(returnedSig, calleeSig, paramIndex, call.At, callbackArgumentName(call), deferEffectValidation); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(returnedSig, calleeSig, calleeName, call.At, callbackArgumentName(call)); err != nil {
			return "", "", err
		}
		return argType, "", nil
	}
	if fieldInfo, ok, err := resolveFunctionFieldArgument(arg, locals); err != nil {
		return "", "", err
	} else if ok {
		if analysis != nil && fieldInfo.FunctionTouchesMutableGlobals {
			analysis.touchesMutableGlobals = true
		}
		if fieldInfo.FunctionValue == "" {
			paramSlots, err := functionParamSlotCount(fieldInfo.FunctionParamTypes, types)
			if err != nil {
				return "", "", err
			}
			callbackSig := FuncSig{
				ParamTypes:     append([]string(nil), fieldInfo.FunctionParamTypes...),
				ParamOwnership: append([]string(nil), fieldInfo.FunctionParamOwnership...),
				ParamSlots:     paramSlots,
				ReturnType:     fieldInfo.FunctionReturnType,
				ThrowsType:     fieldInfo.FunctionThrowsType,
				Effects:        append([]string(nil), fieldInfo.FunctionEffects...),
			}
			if err := validateCallbackSignatureForField(callbackSig, fieldInfo, calleeSig, paramIndex, arg.Pos(), callbackArgumentName(arg), types, deferEffectValidation); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(callbackSig, calleeSig, calleeName, arg.Pos(), callbackArgumentName(arg)); err != nil {
				return "", "", err
			}
			return "fnptr", "", nil
		}
		localSig, ok := funcs[fieldInfo.FunctionValue]
		if !ok {
			return "", "", fmt.Errorf("%s: unknown callback function symbol '%s'", frontend.FormatPos(arg.Pos()), fieldInfo.FunctionValue)
		}
		if err := validateCallbackSignatureForField(localSig, fieldInfo, calleeSig, paramIndex, arg.Pos(), callbackArgumentName(arg), types, deferEffectValidation); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(localSig, calleeSig, calleeName, arg.Pos(), callbackArgumentName(arg)); err != nil {
			return "", "", err
		}
		return "fnptr", fieldInfo.FunctionValue, nil
	}
	if fieldAccess, ok := arg.(*frontend.FieldAccessExpr); ok {
		globalInfo, globalSig, globalOK, err := resolveFunctionTypedGlobalFieldAccess(fieldAccess, globals, funcs)
		if err != nil {
			return "", "", err
		}
		if globalOK {
			globalLocal := LocalInfo{
				TypeName:               globalInfo.TypeName,
				FunctionTypeValue:      true,
				FunctionValue:          globalInfo.FunctionValue,
				FunctionParamTypes:     append([]string(nil), globalInfo.FunctionParamTypes...),
				FunctionParamOwnership: append([]string(nil), globalInfo.FunctionParamOwnership...),
				FunctionReturnType:     globalInfo.FunctionReturnType,
				FunctionThrowsType:     globalInfo.FunctionThrowsType,
				FunctionEffects:        append([]string(nil), globalInfo.FunctionEffects...),
			}
			if err := validateCallbackSignatureForLocal(globalSig, globalLocal, calleeSig, paramIndex, arg.Pos(), callbackArgumentName(arg), types, deferEffectValidation); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(globalSig, calleeSig, calleeName, arg.Pos(), callbackArgumentName(arg)); err != nil {
				return "", "", err
			}
			return globalInfo.TypeName, globalInfo.FunctionValue, nil
		}
	}
	id, ok := arg.(*frontend.IdentExpr)
	if !ok {
		return "", "", unsupportedCallbackArgumentSourceError(arg.Pos(), calleeName)
	}
	if localInfo, ok := locals[id.Name]; ok {
		if analysis != nil && localInfo.FunctionTouchesMutableGlobals {
			analysis.touchesMutableGlobals = true
		}
		if !localInfo.FunctionTypeValue {
			if localInfo.FunctionValue == "" {
				return "", "", unsupportedCallbackArgumentSourceError(arg.Pos(), calleeName)
			}
			if localInfo.GenericFunctionValue {
				return "", "", unsupportedGenericCallbackSymbolError(arg.Pos(), id.Name)
			}
			localSig, ok := funcs[localInfo.FunctionValue]
			if !ok {
				return "", "", fmt.Errorf("%s: unknown callback function symbol '%s'", frontend.FormatPos(arg.Pos()), localInfo.FunctionValue)
			}
			if localSig.ThrowsType != "" && callbackExpectedThrowsType(calleeSig, paramIndex) == "" {
				return "", "", unsupportedThrowingCallbackSymbolError(arg.Pos(), id.Name)
			}
			targetInfo := functionParamLocalInfo(calleeSig, paramIndex)
			targetInfo.FunctionValue = localInfo.FunctionValue
			targetInfo.FunctionCaptures = append([]frontend.ClosureCapture(nil), localInfo.FunctionCaptures...)
			if len(targetInfo.FunctionCaptures) > 0 {
				captureSlots, err := functionCaptureSlotCount(targetInfo.FunctionCaptures, types)
				if err != nil {
					return "", "", err
				}
				if captureSlots > FnPtrEnvSlotCount {
					escapeKind, handleValue, err := classifyCallableEscape(callableBoundaryCallback, targetInfo.FunctionCaptures, types)
					if err != nil {
						return "", "", err
					}
					targetInfo.FunctionEscapeKind = escapeKind
					targetInfo.FunctionHandleValue = handleValue
				}
			}
			if err := validateCallbackSignatureForLocal(localSig, targetInfo, calleeSig, paramIndex, arg.Pos(), id.Name, types, deferEffectValidation); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(localSig, calleeSig, calleeName, arg.Pos(), id.Name); err != nil {
				return "", "", err
			}
			return targetInfo.TypeName, localInfo.FunctionValue, nil
		}
		if localInfo.FunctionValue == "" {
			paramSlots, err := functionParamSlotCount(localInfo.FunctionParamTypes, types)
			if err != nil {
				return "", "", err
			}
			callbackSig := FuncSig{
				ParamTypes:     append([]string(nil), localInfo.FunctionParamTypes...),
				ParamOwnership: append([]string(nil), localInfo.FunctionParamOwnership...),
				ParamSlots:     paramSlots,
				ReturnType:     localInfo.FunctionReturnType,
				ThrowsType:     localInfo.FunctionThrowsType,
				Effects:        append([]string(nil), localInfo.FunctionEffects...),
			}
			if err := validateCallbackSignatureForLocal(callbackSig, localInfo, calleeSig, paramIndex, arg.Pos(), id.Name, types, deferEffectValidation); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(callbackSig, calleeSig, calleeName, arg.Pos(), id.Name); err != nil {
				return "", "", err
			}
			return "fnptr", "", nil
		}
		if localInfo.GenericFunctionValue {
			return "", "", unsupportedGenericCallbackSymbolError(arg.Pos(), id.Name)
		}
		localSig, ok := funcs[localInfo.FunctionValue]
		if !ok {
			return "", "", fmt.Errorf("%s: unknown callback function symbol '%s'", frontend.FormatPos(arg.Pos()), localInfo.FunctionValue)
		}
		if err := validateCallbackSignatureForLocal(localSig, localInfo, calleeSig, paramIndex, arg.Pos(), id.Name, types, deferEffectValidation); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(localSig, calleeSig, calleeName, arg.Pos(), id.Name); err != nil {
			return "", "", err
		}
		return localInfo.TypeName, localInfo.FunctionValue, nil
	}
	if globalInfo, ok := globals[id.Name]; ok {
		if analysis != nil && globalInfo.Mutable {
			analysis.touchesMutableGlobals = true
		}
		if !globalInfo.FunctionTypeValue || globalInfo.FunctionValue == "" {
			return "", "", unsupportedCallbackArgumentSourceError(arg.Pos(), calleeName)
		}
		globalSig, ok := funcs[globalInfo.FunctionValue]
		if !ok {
			return "", "", fmt.Errorf("%s: unknown callback function symbol '%s'", frontend.FormatPos(arg.Pos()), globalInfo.FunctionValue)
		}
		globalLocal := LocalInfo{
			TypeName:               globalInfo.TypeName,
			FunctionTypeValue:      true,
			FunctionValue:          globalInfo.FunctionValue,
			FunctionParamTypes:     append([]string(nil), globalInfo.FunctionParamTypes...),
			FunctionParamOwnership: append([]string(nil), globalInfo.FunctionParamOwnership...),
			FunctionReturnType:     globalInfo.FunctionReturnType,
			FunctionThrowsType:     globalInfo.FunctionThrowsType,
			FunctionEffects:        append([]string(nil), globalInfo.FunctionEffects...),
		}
		if err := validateCallbackSignatureForLocal(globalSig, globalLocal, calleeSig, paramIndex, arg.Pos(), id.Name, types, deferEffectValidation); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(globalSig, calleeSig, calleeName, arg.Pos(), id.Name); err != nil {
			return "", "", err
		}
		return globalInfo.TypeName, globalInfo.FunctionValue, nil
	}

	resolved, err := resolveCheckedCallName(id.Name, funcs, module, imports, id.At)
	if err != nil {
		return "", "", unsupportedCallbackArgumentSourceError(arg.Pos(), calleeName)
	}
	sig, ok := funcs[resolved]
	if !ok {
		return "", "", unsupportedCallbackArgumentSourceError(arg.Pos(), calleeName)
	}
	if err := ensureFuncVisible(resolved, sig, module, id.At); err != nil {
		return "", "", err
	}
	if sig.Generic {
		return "", "", unsupportedGenericCallbackSymbolError(arg.Pos(), id.Name)
	}
	if sig.ThrowsType != "" && callbackExpectedThrowsType(calleeSig, paramIndex) == "" {
		return "", "", unsupportedThrowingCallbackSymbolError(arg.Pos(), id.Name)
	}
	if err := validateCallbackSignature(sig, calleeSig, paramIndex, arg.Pos(), id.Name, deferEffectValidation); err != nil {
		return "", "", err
	}
	if err := validateCallbackClauseCompatibility(sig, calleeSig, calleeName, arg.Pos(), id.Name); err != nil {
		return "", "", err
	}
	return "fnptr", resolved, nil
}

func unsupportedCallbackArgumentSourceError(pos frontend.Position, calleeName string) error {
	return fmt.Errorf(
		"%s: callback argument for '%s' must be a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		frontend.FormatPos(pos),
		calleeName,
	)
}

func unsupportedCallbackCaptureError(pos frontend.Position, rawName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf("%s: callback argument '%s' captures %d environment slots; captured callback arguments support at most %d fnptr environment slots within the supported fnptr ABI", frontend.FormatPos(pos), rawName, envSlots, FnPtrEnvSlotCount)
	}
	return fmt.Errorf("%s: callback argument '%s' captures local values; captured function values cannot be passed as callback arguments in this MVP; closure lifetime/ABI evidence is only available for local direct calls", frontend.FormatPos(pos), rawName)
}

func unsupportedClosureLiteralCallbackCaptureError(pos frontend.Position, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf("%s: callback argument 'closure literal' captures %d environment slots; captured callback arguments support at most %d fnptr environment slots within the supported fnptr ABI", frontend.FormatPos(pos), envSlots, FnPtrEnvSlotCount)
	}
	return fmt.Errorf("%s: callback argument 'closure literal' captures local values; captured function values cannot be passed as callback arguments in this MVP; closure lifetime/ABI evidence is only available for local direct calls", frontend.FormatPos(pos))
}

func unsupportedFunctionTypedCallCaptureError(pos frontend.Position, rawName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf("%s: function-typed callback '%s' captures %d environment slots; direct function-typed calls support at most %d fnptr environment slots within the supported fnptr ABI", frontend.FormatPos(pos), rawName, envSlots, FnPtrEnvSlotCount)
	}
	return fmt.Errorf("%s: function-typed callback '%s' has unsupported captured environment size", frontend.FormatPos(pos), rawName)
}

func unsupportedFunctionFieldCallCaptureError(pos frontend.Position, rawName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf("%s: function-typed struct field call '%s' captures %d environment slots; direct struct-field calls support at most %d fnptr environment slots within the supported fnptr ABI", frontend.FormatPos(pos), rawName, envSlots, FnPtrEnvSlotCount)
	}
	return fmt.Errorf("%s: function-typed struct field call '%s' has unsupported captured environment size", frontend.FormatPos(pos), rawName)
}

func unsupportedEnumPayloadCallCaptureError(pos frontend.Position, rawName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf("%s: function-typed enum payload binding '%s' captures %d environment slots; direct enum-payload calls support at most %d fnptr environment slots within the supported fnptr ABI", frontend.FormatPos(pos), rawName, envSlots, FnPtrEnvSlotCount)
	}
	return fmt.Errorf("%s: function-typed enum payload binding '%s' has unsupported captured environment size", frontend.FormatPos(pos), rawName)
}

func validateCallbackSignatureForLocal(
	callbackSig FuncSig,
	localInfo LocalInfo,
	calleeSig FuncSig,
	paramIndex int,
	pos frontend.Position,
	rawName string,
	types map[string]*TypeInfo,
	deferEffectValidation bool,
) error {
	if localInfo.FunctionReturnType != "" {
		explicitSlots, err := functionParamSlotCount(localInfo.FunctionParamTypes, types)
		if err != nil {
			return err
		}
		hiddenSlots := callbackSig.ParamSlots - explicitSlots
		if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !localInfo.FunctionHandleValue) {
			return unsupportedCallbackCaptureError(pos, rawName, hiddenSlots)
		}
		visibleSig := callbackSig
		visibleSig.ParamTypes = append([]string(nil), localInfo.FunctionParamTypes...)
		visibleSig.ParamOwnership = append([]string(nil), localInfo.FunctionParamOwnership...)
		visibleSig.ParamSlots = explicitSlots
		visibleSig.ReturnType = localInfo.FunctionReturnType
		return validateCallbackSignature(visibleSig, calleeSig, paramIndex, pos, rawName, deferEffectValidation)
	}
	if len(localInfo.FunctionCaptures) == 0 {
		return validateCallbackSignature(callbackSig, calleeSig, paramIndex, pos, rawName, deferEffectValidation)
	}
	captureSlots := 0
	for _, capture := range localInfo.FunctionCaptures {
		info, err := ensureTypeInfo(capture.Type.Name, types)
		if err != nil {
			return err
		}
		captureSlots += info.SlotCount
	}
	if captureSlots < 1 || captureSlots > FnPtrEnvSlotCount {
		return unsupportedCallbackCaptureError(pos, rawName, captureSlots)
	}
	trimmed := callbackSig
	trimmed.ParamTypes = append([]string(nil), localInfo.FunctionParamTypes...)
	trimmed.ParamOwnership = append([]string(nil), localInfo.FunctionParamOwnership...)
	trimmed.ParamSlots -= captureSlots
	return validateCallbackSignature(trimmed, calleeSig, paramIndex, pos, rawName, deferEffectValidation)
}

func validateCallbackSignatureForField(
	callbackSig FuncSig,
	fieldInfo FunctionFieldInfo,
	calleeSig FuncSig,
	paramIndex int,
	pos frontend.Position,
	rawName string,
	types map[string]*TypeInfo,
	deferEffectValidation bool,
) error {
	if fieldInfo.FunctionReturnType == "" {
		return validateCallbackSignature(callbackSig, calleeSig, paramIndex, pos, rawName, deferEffectValidation)
	}
	explicitSlots, err := functionParamSlotCount(fieldInfo.FunctionParamTypes, types)
	if err != nil {
		return err
	}
	hiddenSlots := callbackSig.ParamSlots - explicitSlots
	if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !fieldInfo.FunctionHandleValue) {
		return unsupportedCallbackCaptureError(pos, rawName, hiddenSlots)
	}
	visibleSig := callbackSig
	visibleSig.ParamTypes = append([]string(nil), fieldInfo.FunctionParamTypes...)
	visibleSig.ParamOwnership = append([]string(nil), fieldInfo.FunctionParamOwnership...)
	visibleSig.ParamSlots = explicitSlots
	visibleSig.ReturnType = fieldInfo.FunctionReturnType
	return validateCallbackSignature(visibleSig, calleeSig, paramIndex, pos, rawName, deferEffectValidation)
}

func functionParamSlotCount(typeNames []string, types map[string]*TypeInfo) (int, error) {
	slots := 0
	for _, typeName := range typeNames {
		info, err := ensureTypeInfo(typeName, types)
		if err != nil {
			return 0, err
		}
		slots += info.SlotCount
	}
	return slots, nil
}

func resolveFunctionFieldCall(name string, locals map[string]LocalInfo) (FunctionFieldInfo, bool, error) {
	if !strings.Contains(name, ".") {
		return FunctionFieldInfo{}, false, nil
	}
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return FunctionFieldInfo{}, false, nil
	}
	local, ok := locals[parts[0]]
	if !ok || len(local.FunctionFields) == 0 {
		return FunctionFieldInfo{}, false, nil
	}
	field, ok := local.FunctionFields[strings.Join(parts[1:], ".")]
	return field, ok, nil
}

func resolveFunctionFieldArgument(expr frontend.Expr, locals map[string]LocalInfo) (FunctionFieldInfo, bool, error) {
	name := callbackArgumentName(expr)
	if name == "" {
		return FunctionFieldInfo{}, false, nil
	}
	return resolveFunctionFieldCall(name, locals)
}

func callbackArgumentName(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := callbackArgumentName(e.Base)
		if base != "" && e.Field != "" {
			return base + "." + e.Field
		}
	case *frontend.CallExpr:
		if e.Name != "" {
			return e.Name + "()"
		}
	}
	return ""
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
	wantReturnOwnership := ""
	if paramIndex < len(calleeSig.ParamFunctionReturnOwnership) {
		wantReturnOwnership = calleeSig.ParamFunctionReturnOwnership[paramIndex]
	}
	wantThrows := callbackExpectedThrowsType(calleeSig, paramIndex)
	wantEffects := []string(nil)
	if paramIndex < len(calleeSig.ParamFunctionEffects) {
		wantEffects = calleeSig.ParamFunctionEffects[paramIndex]
	}
	if len(wantParams) != len(callbackSig.ParamTypes) {
		return fmt.Errorf("%s: callback function symbol '%s' has incompatible parameter count: expected %d, got %d", frontend.FormatPos(pos), rawName, len(wantParams), len(callbackSig.ParamTypes))
	}
	wantOwnership := []string(nil)
	if paramIndex < len(calleeSig.ParamFunctionOwnership) {
		wantOwnership = calleeSig.ParamFunctionOwnership[paramIndex]
	}
	if err := validateFunctionTypeParamOwnership(wantOwnership, callbackSig.ParamOwnership, len(wantParams), pos, "callback function symbol", rawName); err != nil {
		return err
	}
	for i := range wantParams {
		if wantParams[i] != callbackSig.ParamTypes[i] {
			return fmt.Errorf("%s: callback function symbol '%s' parameter %d type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), rawName, i+1, wantParams[i], callbackSig.ParamTypes[i])
		}
	}
	if wantReturn != "" && wantReturn != callbackSig.ReturnType {
		return fmt.Errorf("%s: callback function symbol '%s' return type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), rawName, wantReturn, callbackSig.ReturnType)
	}
	if wantReturnOwnership != callbackSig.ReturnOwnership {
		return fmt.Errorf("%s: callback function symbol '%s' return ownership mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), rawName, ownershipDisplay(wantReturnOwnership), ownershipDisplay(callbackSig.ReturnOwnership))
	}
	if wantThrows != callbackSig.ThrowsType {
		return fmt.Errorf("%s: callback function symbol '%s' throws type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), rawName, wantThrows, callbackSig.ThrowsType)
	}
	if !deferEffectValidation {
		if err := validateFunctionTypeCallableEffects(wantEffects, callbackSig.Effects, pos, "callback function symbol", rawName); err != nil {
			return err
		}
	}
	return nil
}

func callbackExpectedThrowsType(calleeSig FuncSig, paramIndex int) string {
	if paramIndex >= 0 && paramIndex < len(calleeSig.ParamFunctionThrows) {
		return calleeSig.ParamFunctionThrows[paramIndex]
	}
	return ""
}

func validateFunctionTypedThrowCall(throwsType string, e *frontend.CallExpr, state *regionState) error {
	isTryCall := state != nil && state.allowThrowDepth > 0 && state.allowThrowCall == e
	isCatchCall := state != nil && state.allowCatchDepth > 0 && state.allowCatchCall == e
	if throwsType == "" {
		if isTryCall {
			return fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		if isCatchCall {
			return fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
		}
		return nil
	}
	if !isTryCall && !isCatchCall {
		return fmt.Errorf("%s: call to throwing function '%s' requires try", frontend.FormatPos(e.At), e.Name)
	}
	if isTryCall && state.throwType == "" {
		return fmt.Errorf("%s: try is only allowed in throwing functions", frontend.FormatPos(e.At))
	}
	if isTryCall && !typesCompatibleWithNullPtr(state.throwType, throwsType, e) {
		return fmt.Errorf("%s: thrown error type mismatch: expected '%s', got '%s'", frontend.FormatPos(e.At), state.throwType, throwsType)
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
		transferOwners := actorTransferOwnerPayloads(e.Args[1], msgType, types, module, imports)
		if err := validateActorBoundaryPayloadExpr(e.Args[1], msgType, types, module, imports, state, transferOwners); err != nil {
			return "", regionNone, err
		}
		if err := checkBorrowedEscape(e.Args[1], locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
			return ownershipDiagnosticf(e.Args[1].Pos(), "cannot send borrowed view across actor boundary; use .copy() (borrowed value derived from '%s' cannot cross actor boundary without copy)", borrowedName)
		}); err != nil {
			return "", regionNone, err
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
			if err := checkResourceCallArg(resolved, "task.group", e.Args[0], funcs, module, imports, state); err != nil {
				return "", regionNone, err
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
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
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
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf("%s: %s target '%s' uses effect '%s' and cannot cross task boundary", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target, blocked)
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
		if !typesCompatibleWithNullPtr(handleType, argType, e.Args[0]) {
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
	if !isResourceHandleType(paramType) {
		return nil
	}
	source, err := resourceSourceForExpr(arg, funcs, module, imports, state)
	if err != nil {
		return err
	}
	if source.ambiguous {
		return ownershipDiagnosticf(arg.Pos(), "resource expression mixes resource provenance")
	}
	if source.unknown {
		name, _ := resourcePathForExpr(arg)
		if name == "" {
			name = "<resource>"
		}
		return ownershipDiagnosticf(arg.Pos(), "ambiguous resource provenance for '%s' after control-flow merge", name)
	}
	if !source.known {
		return nil
	}
	if paramType == "task.group" && resolved == "core.task_group_status" {
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
			state.markResourceFinalizedAliases(source.name, "closed", call.Args[0].Pos())
		}
	}
}

func markTaskHandleJoined(arg frontend.Expr, funcs map[string]FuncSig, module string, imports map[string]string, state *regionState) {
	if state == nil {
		return
	}
	if source, err := resourceSourceForExpr(arg, funcs, module, imports, state); err == nil && source.known {
		state.markResourceFinalizedAliases(source.name, "joined", arg.Pos())
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
	if explicitCopyResultExpr(expr) {
		return "", false, nil
	}
	tname, regionID, err := checkExprWithEffects(expr, locals, globals, funcs, types, module, imports, state, effects, analysis)
	if err != nil {
		return "", false, err
	}
	if explicitCopyResultExpr(expr) {
		return "", false, nil
	}
	borrowedName, borrowed := borrowedOwnerForRegion(regionID, state)
	if borrowed {
		return borrowedName, true, nil
	}
	if typeMayContainRegion(tname, types) {
		owners := map[string]struct{}{}
		for _, leafRegion := range regionTreeForExpr(tname, expr, regionID, types, state) {
			if borrowedName, borrowed := borrowedOwnerForRegion(leafRegion, state); borrowed {
				owners[borrowedName] = struct{}{}
			}
		}
		if len(owners) > 0 {
			names := make([]string, 0, len(owners))
			for name := range owners {
				names = append(names, name)
			}
			sort.Strings(names)
			return names[0], true, nil
		}
	}
	if tname == "ptr" {
		borrowedName, borrowed := borrowedPtrOwnerFromExpr(expr, state, nil)
		return borrowedName, borrowed, nil
	}
	if typeMayContainPtr(tname, types) {
		if borrowedName, borrowed := borrowedPtrOwnerFromExpr(expr, state, nil); borrowed {
			return borrowedName, true, nil
		}
	}
	return borrowedName, borrowed, nil
}

func explicitCopyResultExpr(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok {
		return false
	}
	if method, ok := syntheticViewMethodName(call.Name); ok {
		return method == "copy"
	}
	if _, method, ok := sliceViewMethodParts(call.Name); ok {
		return method == "copy"
	}
	name := call.Name
	if target, ok := ResolveBuiltinAlias(name); ok {
		name = target
	}
	return isExplicitCopyBuiltin(name)
}

func checkBorrowedEscape(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, funcs map[string]FuncSig, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState, effects *effectContext, analysis *functionAnalysisState, format func(string) error) error {
	if expr == nil {
		return nil
	}
	if handled, err := checkBorrowedEscapeAggregateConstructor(expr, locals, globals, funcs, types, module, imports, state, effects, analysis, format); handled || err != nil {
		return err
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
	if call, ok := expr.(*frontend.CallExpr); ok {
		if _, caseInfo, found, err := resolveEnumCaseConstructorCall(call, types, module, imports); err != nil {
			return err
		} else if found {
			for i, arg := range call.Args {
				if i >= len(caseInfo.PayloadTypes) {
					break
				}
				if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
					return err
				}
			}
		} else if len(call.Args) > 0 && len(call.ArgLabels) == len(call.Args) {
			allLabels := true
			for _, label := range call.ArgLabels {
				if label == "" {
					allLabels = false
					break
				}
			}
			typeRef := frontend.TypeRef{At: call.At, Kind: frontend.TypeRefNamed, Name: call.Name}
			resolvedType, err := resolveTypeName(&typeRef, module, imports)
			if err == nil && allLabels {
				if info, ok := types[resolvedType]; ok && info.Kind == TypeStruct {
					for _, arg := range call.Args {
						if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	if fieldAccess, ok := expr.(*frontend.FieldAccessExpr); ok {
		if _, _, enumCase, err := resolveEnumCaseExpr(fieldAccess, locals, globals, types, module, imports); err != nil {
			return err
		} else if enumCase {
			return nil
		}
		if err := checkBorrowedEscape(fieldAccess.Base, locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
			return err
		}
	}
	if idx, ok := expr.(*frontend.IndexExpr); ok {
		if err := checkBorrowedEscape(idx.Base, locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
			return err
		}
	}
	if match, ok := expr.(*frontend.MatchExpr); ok {
		for _, c := range match.Cases {
			if err := checkBorrowedEscape(c.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
				return err
			}
		}
	}
	if catch, ok := expr.(*frontend.CatchExpr); ok {
		for _, c := range catch.Cases {
			if err := checkBorrowedEscape(c.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
				return err
			}
		}
	}
	if _, ok := expr.(*frontend.TryExpr); ok {
		return nil
	}
	if _, ok := expr.(*frontend.AwaitExpr); ok {
		return nil
	}
	if unary, ok := expr.(*frontend.UnaryExpr); ok {
		return checkBorrowedEscape(unary.X, locals, globals, funcs, types, module, imports, state, effects, analysis, format)
	}
	if binary, ok := expr.(*frontend.BinaryExpr); ok {
		if err := checkBorrowedEscape(binary.Left, locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
			return err
		}
		return checkBorrowedEscape(binary.Right, locals, globals, funcs, types, module, imports, state, effects, analysis, format)
	}
	return nil
}

func checkBorrowedEscapeAggregateConstructor(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, funcs map[string]FuncSig, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState, effects *effectContext, analysis *functionAnalysisState, format func(string) error) (bool, error) {
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		typeName, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return true, err
		}
		info, ok := types[typeName]
		if !ok || info.Kind != TypeStruct {
			return false, nil
		}
		for _, field := range e.Fields {
			fieldInfo, ok := info.FieldMap[field.Name]
			if !ok || !borrowedEscapeShouldInspect(fieldInfo.TypeName, types) {
				continue
			}
			if err := checkBorrowedEscape(field.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
				return true, err
			}
		}
		return true, nil
	case *frontend.CallExpr:
		if e.ResolvedType == "" {
			return false, nil
		}
		info, ok := types[e.ResolvedType]
		if !ok {
			return false, nil
		}
		switch info.Kind {
		case TypeStruct:
			for i, field := range info.Fields {
				if i >= len(e.Args) || !borrowedEscapeShouldInspect(field.TypeName, types) {
					continue
				}
				if err := checkBorrowedEscape(e.Args[i], locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
					return true, err
				}
			}
			return true, nil
		case TypeEnum:
			_, caseInfo, found, err := resolveEnumCaseConstructorCall(e, types, module, imports)
			if err != nil {
				return true, err
			}
			if !found {
				return true, nil
			}
			for i, arg := range e.Args {
				if i >= len(caseInfo.PayloadTypes) || !borrowedEscapeShouldInspect(caseInfo.PayloadTypes[i], types) {
					continue
				}
				if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, format); err != nil {
					return true, err
				}
			}
			return true, nil
		}
	}
	return false, nil
}

func borrowedEscapeShouldInspect(typeName string, types map[string]*TypeInfo) bool {
	if typeMayContainPtr(typeName, types) {
		return true
	}
	return typeMayContainRegion(typeName, types) && !typeContainsResourceHandle(typeName, types)
}

func checkBorrowedInoutEscape(expr frontend.Expr, targetName string, pos frontend.Position, locals map[string]LocalInfo, globals map[string]GlobalInfo, funcs map[string]FuncSig, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState, effects *effectContext, analysis *functionAnalysisState) error {
	return checkBorrowedEscape(expr, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
		return lifetimeDiagnosticf(pos, "borrowed local '%s' cannot escape via inout assignment to '%s'", borrowedName, targetName)
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
			return ownershipDiagnosticf(pos, "ambiguous resource provenance for '%s' after control-flow merge", path)
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
	if reason := typeActorTaskSendabilityUnsafeReason(typeName, types, map[string]bool{}); reason != "" {
		return fmt.Errorf("%s: typed task error payload must be sendable across task boundary: %s", frontend.FormatPos(pos), reason)
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
	if surfaceType, ok := surfaceActorTaskBoundaryValueType(typeName, types); ok {
		return fmt.Errorf("surface value '%s' cannot cross actor/task boundary", surfaceType)
	}
	switch info.Kind {
	case TypeI32, TypeU8, TypeBool, TypeIsland:
		return nil
	case TypeEnum:
		if len(info.EnumCases) > 255 {
			return fmt.Errorf("typed actor message enum supports at most 255 cases, got %d for '%s'", len(info.EnumCases), typeName)
		}
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
	case TypeActor:
		return typedActorUnsupportedTransferTypeError("actor handle", typeName)
	case TypeCap:
		return typedActorUnsupportedTransferTypeError("capability handle", typeName)
	case TypePtr:
		return typedActorUnsupportedTransferTypeError("pointer handle", typeName)
	case TypeStr, TypeSlice:
		return nil
	case TypeArray:
		return typedActorUnsupportedTransferTypeError("array view", typeName)
	case TypeOptional:
		return typedActorUnsupportedTransferTypeError("optional wrapper", typeName)
	default:
		return typedActorUnsupportedTransferTypeError("non-value type", typeName)
	}
}

func validateActorBoundaryPayloadExpr(expr frontend.Expr, typeName string, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState, transferOwners map[string]bool) error {
	info, ok := types[typeName]
	if !ok {
		return fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(expr.Pos()), typeName)
	}
	switch info.Kind {
	case TypeStr, TypeSlice:
		if isExplicitCopyExpr(expr) {
			return nil
		}
		if info.Kind == TypeSlice {
			if owner := ownedRegionSliceOwnerForExpr(expr, state); owner != "" && transferOwners[owner] {
				return nil
			}
		}
		return ownershipDiagnosticf(expr.Pos(), "cannot send borrowed view across actor boundary; use .copy() (borrowed value derived from '%s' cannot cross actor boundary without copy)", borrowOwnerNameFromExpr(expr))
	case TypeStruct:
		for _, field := range structFieldExprs(expr, info) {
			if kind, _, directView := borrowedReturnDirectViewLabels(field.typeName, types); directView && !isExplicitCopyExpr(field.value) {
				return ownershipDiagnosticf(expr.Pos(), "aggregate '%s' contains borrowed %s field '%s' that cannot cross actor boundary", displayTypeName(typeName, module), kind, field.name)
			}
			if err := validateActorBoundaryPayloadExpr(field.value, field.typeName, types, module, imports, state, transferOwners); err != nil {
				return err
			}
		}
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
					if err := validateActorBoundaryPayloadExpr(arg, caseInfo.PayloadTypes[i], types, module, imports, state, transferOwners); err != nil {
						return err
					}
				}
			}
		}
	case TypeOptional:
		if call, ok := expr.(*frontend.CallExpr); ok {
			for _, arg := range call.Args {
				if err := validateActorBoundaryPayloadExpr(arg, info.ElemType, types, module, imports, state, transferOwners); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func actorTransferOwnerPayloads(expr frontend.Expr, typeName string, types map[string]*TypeInfo, module string, imports map[string]string) map[string]bool {
	owners := make(map[string]bool)
	collectActorTransferOwnerPayloads(expr, typeName, types, module, imports, owners)
	return owners
}

func collectActorTransferOwnerPayloads(expr frontend.Expr, typeName string, types map[string]*TypeInfo, module string, imports map[string]string, owners map[string]bool) {
	if expr == nil {
		return
	}
	info, ok := types[typeName]
	if !ok {
		return
	}
	switch info.Kind {
	case TypeIsland:
		if path, ok := resourcePathForExpr(expr); ok && path != "" {
			owners[path] = true
		}
	case TypeStruct:
		for _, field := range structFieldExprs(expr, info) {
			collectActorTransferOwnerPayloads(field.value, field.typeName, types, module, imports, owners)
		}
	case TypeEnum:
		call, ok := expr.(*frontend.CallExpr)
		if !ok {
			return
		}
		_, caseInfo, found, err := resolveEnumCaseConstructorCall(call, types, module, imports)
		if err != nil || !found {
			return
		}
		for i, arg := range call.Args {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			collectActorTransferOwnerPayloads(arg, caseInfo.PayloadTypes[i], types, module, imports, owners)
		}
	case TypeOptional:
		if call, ok := expr.(*frontend.CallExpr); ok {
			for _, arg := range call.Args {
				collectActorTransferOwnerPayloads(arg, info.ElemType, types, module, imports, owners)
			}
		}
	}
}

func isExplicitCopyExpr(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	return isExplicitCopyBuiltin(call.Name)
}

func typedActorUnsupportedTransferTypeError(category string, typeName string) error {
	return fmt.Errorf("typed actor message payload must be value-only, got %s '%s'", category, typeName)
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
	info, ok := types[typeName]
	if !ok {
		return fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(expr.Pos()), typeName)
	}
	if !typedActorTypeContainsIsland(typeName, types, map[string]bool{}) && info.Kind != TypeSlice {
		return nil
	}
	switch info.Kind {
	case TypeIsland:
		if markConsumedResourceExpr(expr, typeName, locals, types, state) {
			return nil
		}
		id, ok := expr.(*frontend.IdentExpr)
		if !ok {
			return ownershipDiagnosticf(expr.Pos(), "island transfer payload must be a local value")
		}
		if _, ok := locals[id.Name]; !ok {
			return ownershipDiagnosticf(id.At, "island transfer payload '%s' must be a local value", id.Name)
		}
		markConsumedResourceValue(id.Name, typeName, types, state, id.At)
		return nil
	case TypeSlice:
		if ownedRegionSliceOwnerForExpr(expr, state) == "" || isExplicitCopyExpr(expr) {
			return nil
		}
		if path, ok := resourcePathForExpr(expr); ok && path != "" {
			state.markConsumed(path, expr.Pos())
			return nil
		}
		if id, ok := expr.(*frontend.IdentExpr); ok {
			if _, exists := locals[id.Name]; !exists {
				return nil
			}
			state.markConsumed(id.Name, id.At)
		}
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
			return ownershipDiagnosticf(expr.Pos(), "island-containing struct transfer payload must be a local value")
		}
		if _, ok := locals[id.Name]; !ok {
			return ownershipDiagnosticf(id.At, "island-containing transfer payload '%s' must be a local value", id.Name)
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
			return ownershipDiagnosticf(expr.Pos(), "island-containing enum transfer payload must be a local value")
		}
		if _, ok := locals[id.Name]; !ok {
			return ownershipDiagnosticf(id.At, "island-containing transfer payload '%s' must be a local value", id.Name)
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
	fieldTree := make(map[string]int)
	for _, field := range info.Fields {
		arg, ok := argByLabel[field.Name]
		if !ok {
			return "", regionNone, true, fmt.Errorf("%s: missing field '%s'", frontend.FormatPos(e.At), field.Name)
		}
		if field.FunctionTypeValue {
			markMutableFunctionTypedGlobalSource(arg, globals, analysis)
			if _, err := validateFunctionTypeStructFieldBinding(resolvedType, field, arg, locals, globals, funcs, types, module, imports); err != nil {
				return "", regionNone, true, err
			}
			orderedArgs = append(orderedArgs, arg)
			orderedLabels = append(orderedLabels, field.Name)
			continue
		}
		valType, valRegion, err := checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, true, err
		}
		if !typesCompatibleWithNullPtr(field.TypeName, valType, arg) {
			return "", regionNone, true, fmt.Errorf("%s: type mismatch for field '%s'", frontend.FormatPos(arg.Pos()), field.Name)
		}
		consumeIslandSourceLocals(arg, field.TypeName, locals, types, module, imports, state)
		appendRegionTree(fieldTree, field.Name, field.TypeName, arg, valRegion, types, state)
		orderedArgs = append(orderedArgs, arg)
		orderedLabels = append(orderedLabels, field.Name)
	}

	e.Name = resolvedType
	e.Args = orderedArgs
	e.ArgLabels = orderedLabels
	e.ResolvedType = resolvedType
	state.setExprRegionTree(e, fieldTree)
	return resolvedType, constructorRegionFromTree(fieldTree), true, nil
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
		if c.Default {
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
		if c.Default {
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
	typeName, caseName, ok := resolveBareEnumPatternParts(field, module, imports)
	if !ok || typeName != expectedType {
		return "", false
	}
	return caseName, true
}

func bareEnumPatternTypeAndCase(pattern frontend.Expr, module string, imports map[string]string) (string, string, bool) {
	field, ok := pattern.(*frontend.FieldAccessExpr)
	if !ok {
		return "", "", false
	}
	return resolveBareEnumPatternParts(field, module, imports)
}

func resolveBareEnumPatternParts(field *frontend.FieldAccessExpr, module string, imports map[string]string) (string, string, bool) {
	baseName, fields, pos, ok := splitFieldPath(field.Base)
	if !ok {
		return "", "", false
	}
	parts := append([]string{baseName}, fields...)
	if len(parts) == 0 {
		return "", "", false
	}
	ref := frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed, Name: strings.Join(parts, ".")}
	typeName, err := resolveTypeName(&ref, module, imports)
	if err != nil {
		return "", "", false
	}
	return typeName, field.Field, true
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
		case TypeI32, TypeI64, TypeU8, TypeBool, TypePtr, TypeIsland, TypeCap, TypeActor, TypeEnum:
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
