package semantics

import (
	"fmt"
	"sort"
	"strings"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	semanticsexpressions "tetra_language/compiler/internal/semantics/expressions"
	semanticsgenerics "tetra_language/compiler/internal/semantics/generics"
	semanticspolicy "tetra_language/compiler/internal/semantics/policy"
)

// ---- exprs.go ----

func markMutableFunctionTypedGlobalSource(
	expr frontend.Expr,
	globals map[string]GlobalInfo,
	analysis *functionAnalysisState,
) {
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
	returnInfo, found, err := functionTypedReturnParamRefMetadata(
		callSig,
		callSig.ReturnFunctionParamName,
		call,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
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
		resultType, err := checkMatchExpr(
			e,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return "", regionNone, err
		}
		return resultType, regionNone, nil
	case *frontend.CatchExpr:
		resultType, err := checkCatchExpr(
			e,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
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
			return "", regionNone, ownershipDiagnosticf(e.At, (("ambiguous resource provenance for " +
				"'%s' after control-flow ") +
				"merge"), e.Name)
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
			return "", regionNone, fmt.Errorf(
				"%s: unknown identifier '%s'",
				frontend.FormatPos(e.At),
				e.Name,
			)
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
							("%s: ambiguous region for '%s' after control-flow merge (%s: " +
								"%s, %s: %s); hint: assign to a fresh variable in each " +
								"branch and use it after the merge"),
							frontend.FormatPos(e.At),
							e.Name,
							conflict.leftLabel,
							formatRegionID(state, conflict.leftRegion),
							conflict.rightLabel,
							formatRegionID(state, conflict.rightRegion),
						)
					}
					return "", regionNone, fmt.Errorf(
						("%s: ambiguous region for '%s' after control-flow merge; " +
							"hint: reassign it to a single region before use"),
						frontend.FormatPos(e.At),
						e.Name,
					)
				}
				return "", regionNone, fmt.Errorf(
					"%s: ambiguous region for '%s'",
					frontend.FormatPos(e.At),
					e.Name,
				)
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
		if typeName, _, ok, err := resolveEnumCaseExpr(
			e,
			locals,
			globals,
			types,
			module,
			imports,
		); ok || err != nil {
			if err != nil {
				return "", regionNone, err
			}
			return typeName, regionNone, nil
		}
		targetInfo, targetType, err := ResolveFieldAccessType(e, locals, globals, types)
		if err != nil {
			return "", regionNone, err
		}
		baseType, baseRegion, err := checkExprWithEffects(
			e.Base,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
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
				return "", regionNone, ownershipDiagnosticf(
					e.At,
					"resource expression mixes resource provenance",
				)
			}
			path, _ := resourcePathForExpr(e)
			if source.unknown {
				return "", regionNone, ownershipDiagnosticf(e.At, (("ambiguous resource provenance " +
					"for '%s' after control-flow ") +
					"merge"), path)
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
		baseType, _, err := checkExprWithEffects(
			e.Base,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return "", regionNone, err
		}
		indexType, _, err := checkExprWithEffects(
			e.Index,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
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
		if enumType, caseInfo, ok, err := resolveEnumCaseConstructorCall(
			e,
			types,
			module,
			imports,
		); ok || err != nil {
			if err != nil {
				return "", regionNone, err
			}
			if len(e.ArgLabels) > 0 {
				for _, label := range e.ArgLabels {
					if label != "" {
						return "", regionNone, fmt.Errorf(
							"%s: enum case payload arguments do not use labels",
							frontend.FormatPos(e.At),
						)
					}
				}
			}
			if len(caseInfo.PayloadTypes) == 0 {
				return "", regionNone, fmt.Errorf(
					"%s: enum case '%s.%s' has no payload; use '%s.%s'",
					frontend.FormatPos(e.At),
					displayTypeName(enumType, module),
					caseInfo.Name,
					displayTypeName(enumType, module),
					caseInfo.Name,
				)
			}
			if len(e.Args) != len(caseInfo.PayloadTypes) {
				return "", regionNone, fmt.Errorf(
					"%s: enum case '%s.%s' expects %d payload argument(s), got %d",
					frontend.FormatPos(e.At),
					displayTypeName(enumType, module),
					caseInfo.Name,
					len(caseInfo.PayloadTypes),
					len(e.Args),
				)
			}
			payloadTree := make(map[string]int)
			for i, arg := range e.Args {
				argType := ""
				argRegion := regionNone
				if i < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[i] {
					markMutableFunctionTypedGlobalSource(arg, globals, analysis)
					if _, err := validateFunctionTypeEnumPayloadBinding(
						enumType,
						caseInfo,
						i,
						arg,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
					); err != nil {
						return "", regionNone, err
					}
					argType = caseInfo.PayloadTypes[i]
				} else {
					var err error
					argType, argRegion, err = checkExprWithEffects(
						arg,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						state,
						effects,
						analysis,
					)
					if err != nil {
						return "", regionNone, err
					}
				}
				if !typesCompatibleWithNullPtr(caseInfo.PayloadTypes[i], argType, arg) {
					return "", regionNone, fmt.Errorf(
						"%s: enum case '%s.%s' payload %d expects '%s', got '%s'",
						frontend.FormatPos(arg.Pos()),
						displayTypeName(enumType, module),
						caseInfo.Name,
						i+1,
						caseInfo.PayloadTypes[i],
						argType,
					)
				}
				if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
					consumeIslandSourceLocals(arg, caseInfo.PayloadTypes[i], locals, types, module, imports, state)
					appendRegionTree(
						payloadTree,
						resourceEnumPayloadPath("", caseInfo.Ordinal, i),
						caseInfo.PayloadTypes[i],
						arg,
						argRegion,
						types,
						state,
					)
				}
			}
			e.ResolvedType = enumType
			state.setExprRegionTree(e, payloadTree)
			return enumType, constructorRegionFromTree(payloadTree), nil
		}
		return checkCallExprWithEffects(
			e,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			if len(e.Decl.TypeParams) > 0 {
				if name, pos, ok := firstCapture(collectClosureCaptures(e.Decl, locals)); ok {
					return "", regionNone, unsupportedGenericClosureCaptureError(pos, name)
				}
				return "ptr", regionNone, nil
			}
			if name, pos, ok := firstCapture(collectClosureCaptures(e.Decl, locals)); ok {
				return "", regionNone, fmt.Errorf(("%s: capturing closure literal captures '%s' but is not " +
					"let-bound; %s"), frontend.FormatPos(pos), name, closureLiteralDirectCallCaptureText())
			}
		}
		return "ptr", regionNone, nil
	case *frontend.TryExpr:
		if state.throwType == "" {
			return "", regionNone, fmt.Errorf(
				"%s: try is only allowed in throwing functions",
				frontend.FormatPos(e.At),
			)
		}
		call, ok := e.X.(*frontend.CallExpr)
		isTryAwait := false
		awaitPos := e.At
		if !ok {
			if await, awaitOK := e.X.(*frontend.AwaitExpr); awaitOK {
				if !state.async {
					return "", regionNone, fmt.Errorf(
						"%s: await is only allowed in async functions",
						frontend.FormatPos(await.At),
					)
				}
				call, ok = await.X.(*frontend.CallExpr)
				isTryAwait = ok
				awaitPos = await.At
			}
		}
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: try expects a throwing function call",
				frontend.FormatPos(e.At),
			)
		}
		state.allowThrowDepth++
		state.allowThrowCall = call
		if isTryAwait {
			state.allowAwaitDepth++
			state.allowAwaitCall = call
		}
		tname, regionID, err := checkCallExprWithEffects(
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
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
			return "", regionNone, fmt.Errorf(
				"%s: await is only allowed in async functions",
				frontend.FormatPos(e.At),
			)
		}
		if tryExpr, ok := e.X.(*frontend.TryExpr); ok {
			if call, callOK := tryExpr.X.(*frontend.CallExpr); callOK {
				return "", regionNone, fmt.Errorf(
					"%s: use 'try await %s()' for async typed-error propagation",
					frontend.FormatPos(e.At),
					call.Name,
				)
			}
			return "", regionNone, fmt.Errorf(("%s: use 'try await <call>()' for async typed-error " +
				"propagation"), frontend.FormatPos(e.At))
		}
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: await expects an async function call",
				frontend.FormatPos(e.At),
			)
		}
		state.allowAwaitDepth++
		state.allowAwaitCall = call
		tname, regionID, err := checkCallExprWithEffects(
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
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
		if info.RuntimeOwned && !info.UserConstructible {
			return "", regionNone, runtimeOwnedConstructionError(e.At, resolved)
		}
		if info.Kind != TypeStruct {
			return "", regionNone, fmt.Errorf("%s: '%s' is not a struct", frontend.FormatPos(e.At), resolved)
		}
		seen := make(map[string]frontend.StructFieldInit, len(e.Fields))
		for _, field := range e.Fields {
			if _, exists := info.FieldMap[field.Name]; !exists {
				return "", regionNone, fmt.Errorf(
					"%s: unknown field '%s'",
					frontend.FormatPos(field.At),
					field.Name,
				)
			}
			if _, exists := seen[field.Name]; exists {
				return "", regionNone, fmt.Errorf(
					"%s: duplicate field '%s'",
					frontend.FormatPos(field.At),
					field.Name,
				)
			}
			seen[field.Name] = field
		}
		fieldTree := make(map[string]int)
		for _, field := range info.Fields {
			init, ok := seen[field.Name]
			if !ok {
				return "", regionNone, fmt.Errorf(
					"%s: missing field '%s'",
					frontend.FormatPos(e.At),
					field.Name,
				)
			}
			if field.FunctionTypeValue {
				markMutableFunctionTypedGlobalSource(init.Value, globals, analysis)
				if _, err := validateFunctionTypeStructFieldBinding(
					resolved,
					field,
					init.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
				); err != nil {
					return "", regionNone, err
				}
				continue
			}
			valType, valRegion, err := checkExprWithEffects(
				init.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return "", regionNone, err
			}
			if !typesCompatibleWithNullPtr(field.TypeName, valType, init.Value) {
				return "", regionNone, fmt.Errorf(
					"%s: type mismatch for field '%s'",
					frontend.FormatPos(init.At),
					field.Name,
				)
			}
			consumeIslandSourceLocals(init.Value, field.TypeName, locals, types, module, imports, state)
			appendRegionTree(fieldTree, field.Name, field.TypeName, init.Value, valRegion, types, state)
		}
		state.setExprRegionTree(e, fieldTree)
		return resolved, constructorRegionFromTree(fieldTree), nil
	case *frontend.UnaryExpr:
		xtype, _, err := checkExprWithEffects(
			e.X,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
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
				return "", regionNone, fmt.Errorf(
					"%s: unary '!' expects bool or i32/u8",
					frontend.FormatPos(e.At),
				)
			}
			return "bool", regionNone, nil
		default:
			return "", regionNone, fmt.Errorf("%s: unsupported unary operator", frontend.FormatPos(e.At))
		}
	case *frontend.BinaryExpr:
		ltype, _, err := checkExprWithEffects(
			e.Left,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return "", regionNone, err
		}
		rtype, _, err := checkExprWithEffects(
			e.Right,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return "", regionNone, err
		}
		switch e.Op {
		case frontend.TokenPlus,
			frontend.TokenMinus,
			frontend.TokenStar,
			frontend.TokenSlash,
			frontend.TokenPercent:
			if !isInt32Like(ltype) || !isInt32Like(rtype) {
				return "", regionNone, fmt.Errorf(
					"%s: arithmetic operators require i32/u8",
					frontend.FormatPos(e.At),
				)
			}
			return "i32", regionNone, nil
		case frontend.TokenLess, frontend.TokenGreater, frontend.TokenGreaterEq, frontend.TokenLessEq:
			if !isInt32Like(ltype) || !isInt32Like(rtype) {
				return "", regionNone, fmt.Errorf(
					"%s: relational operators require i32/u8",
					frontend.FormatPos(e.At),
				)
			}
			return "bool", regionNone, nil
		case frontend.TokenEqEq, frontend.TokenBangEq:
			if !comparableTypes(ltype, rtype, types) {
				return "", regionNone, fmt.Errorf(
					"%s: cannot compare '%s' and '%s'",
					frontend.FormatPos(e.At),
					ltype,
					rtype,
				)
			}
			return "bool", regionNone, nil
		case frontend.TokenAmpAmp, frontend.TokenPipePipe:
			if ltype != "bool" || rtype != "bool" {
				return "", regionNone, fmt.Errorf(
					"%s: logical operators require bool",
					frontend.FormatPos(e.At),
				)
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
	scrutType, scrutRegion, err := checkExprWithEffects(
		e.Value,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		state,
		effects,
		analysis,
	)
	if err != nil {
		return "", err
	}
	scrutInfo, scrutInfoOK := types[scrutType]
	if !isInt32Like(scrutType) {
		if !scrutInfoOK || (scrutInfo.Kind != TypeEnum && scrutInfo.Kind != TypeOptional) {
			return "", fmt.Errorf(
				"%s: match value must be enum or i32/u8",
				frontend.FormatPos(e.At),
			)
		}
	}
	if e.ResultType == "" {
		resultType, err := inferMatchExprType(e, locals, globals, funcs, types, module, imports)
		if err != nil {
			return "", fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		e.ResultType = resultType
	}
	if err := reportBarePayloadMatchExprPatterns(
		e,
		scrutType,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	); err != nil {
		return "", err
	}
	if !matchExprHasCompleteOptionalPatterns(e) &&
		!matchExprHasCompleteEnumPatterns(e, locals, globals, funcs, types, module, imports) {
		hasDefault := false
		for _, c := range e.Cases {
			if c.Default && c.Guard == nil {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			return "", fmt.Errorf(
				"%s: match expression must be exhaustive",
				frontend.FormatPos(e.At),
			)
		}
	}
	scrutineeResourcePath := e.ScrutineeLocal
	scrutineeOwnershipPath := scrutineeResourcePath
	if path, ok := resourcePathForExpr(e.Value); ok {
		scrutineeOwnershipPath = path
	}
	if scrutineeResourcePath != "" {
		if err := bindResourceTreeFromExpr(
			scrutineeResourcePath,
			scrutType,
			e.Value,
			funcs,
			types,
			module,
			imports,
			state,
		); err != nil {
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
					return "", fmt.Errorf(
						"%s: some pattern requires optional match value",
						frontend.FormatPos(some.At),
					)
				}
				patType = optionalSomePatternType
			} else if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
				caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
				if err != nil {
					return "", err
				}
				if !found {
					return "", fmt.Errorf(
						"%s: unknown enum pattern '%s.%s'",
						frontend.FormatPos(enumPat.At),
						enumPat.TypeName,
						enumPat.CaseName,
					)
				}
				if err := validateEnumCasePatternPayload(enumPat, caseType, caseInfo, module); err != nil {
					return "", err
				}
				patType = caseType
			} else {
				var err error
				patType, _, err = checkExprWithEffects(
					c.Pattern,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				)
				if err != nil {
					return "", err
				}
			}
			if scrutInfoOK &&
				scrutInfo.Kind == TypeOptional &&
				patType != "none" &&
				patType != optionalSomePatternType {
				return "", fmt.Errorf(("%s: optional match supports only 'none', 'some(name)', and " +
					"'_' patterns"), frontend.FormatPos(c.At))
			}
			if !matchPatternCompatible(scrutType, patType, types) {
				return "", fmt.Errorf(
					"%s: match pattern type mismatch: expected '%s', got '%s'",
					frontend.FormatPos(c.At),
					scrutType,
					patType,
				)
			}
			if c.Guard == nil {
				if key := matchPatternKey(c.Pattern, patType); key != "" {
					if first, exists := seenPatterns[key]; exists {
						return "", fmt.Errorf(
							"%s: duplicate match pattern (first at %s)",
							frontend.FormatPos(c.At),
							frontend.FormatPos(first),
						)
					}
					seenPatterns[key] = c.At
				}
			}
		}
		caseScopeID := regionNone
		if i < len(caseScopes) {
			caseScopeID = caseScopes[i]
		}
		if caseScopeID == regionNone {
			caseScopeID = patternBindingScopeID(c.Pattern, state)
		}
		err := withActiveScope(state, caseScopeID, func() error {
			if err := bindPatternOwnershipAliases(
				c.Pattern,
				"",
				scrutineeOwnershipPath,
				scrutType,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
			if err := bindPatternResourceLocals(
				c.Pattern,
				"",
				scrutineeResourcePath,
				scrutType,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
			if err := bindPatternRegionLocals(
				c.Pattern,
				"",
				scrutineeResourcePath,
				scrutType,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
			if c.Guard != nil {
				guardType, _, err := checkExprWithEffects(
					c.Guard,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				)
				if err != nil {
					return err
				}
				if guardType != "bool" {
					return fmt.Errorf(
						"%s: match guard must be Bool",
						frontend.FormatPos(c.Guard.Pos()),
					)
				}
			}
			armType, armRegion, err := checkExprWithEffects(
				c.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(e.ResultType, armType, c.Value) {
				return fmt.Errorf(
					"%s: match expression case type mismatch: expected '%s', got '%s'",
					frontend.FormatPos(c.At),
					e.ResultType,
					armType,
				)
			}
			if e.ResultLocal != "" {
				if err := bindResourceTreeFromExpr(
					e.ResultLocal,
					e.ResultType,
					c.Value,
					funcs,
					types,
					module,
					imports,
					state,
				); err != nil {
					return err
				}
				bindRegionTreeFromExpr(
					e.ResultLocal,
					e.ResultType,
					c.Value,
					armRegion,
					types,
					state,
				)
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
		return "", fmt.Errorf(
			"%s: catch expects a throwing function call",
			frontend.FormatPos(e.At),
		)
	}
	state.allowCatchDepth++
	state.allowCatchCall = call
	successType, _, err := checkCallExprWithEffects(
		call,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		state,
		effects,
		analysis,
	)
	state.allowCatchDepth--
	state.allowCatchCall = nil
	if err != nil {
		return "", err
	}
	var catchSig FuncSig
	catchSigOK := false
	if call.Name == "core.task_join_i32_typed" || call.Name == "core.task_join_group_i32_typed" {
		if len(call.TypeArgs) != 1 || call.TypeArgs[0].Name == "" {
			return "", fmt.Errorf(
				"%s: task_join_i32_typed missing resolved error type",
				frontend.FormatPos(call.At),
			)
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
		if err := bindCatchErrorResourceSummary(
			e.ErrorLocal,
			call,
			catchSig,
			funcs,
			types,
			module,
			imports,
			state,
		); err != nil {
			return "", err
		}
	}
	if err := reportBarePayloadCatchExprPatterns(
		e,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	); err != nil {
		return "", err
	}
	if !catchExprHasCompleteOptionalPatterns(e, e.ErrorType, types) &&
		!catchExprHasCompleteEnumPatterns(
			e,
			e.ErrorType,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		) {
		hasDefault := false
		for _, c := range e.Cases {
			if c.Default && c.Guard == nil {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			return "", fmt.Errorf(
				"%s: catch expression must be exhaustive",
				frontend.FormatPos(e.At),
			)
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
			patType, err := catchPatternType(
				c.Pattern,
				e.ErrorType,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return "", err
			}
			if !matchPatternCompatible(e.ErrorType, patType, types) {
				return "", fmt.Errorf(
					"%s: catch pattern type mismatch: expected '%s', got '%s'",
					frontend.FormatPos(c.At),
					e.ErrorType,
					patType,
				)
			}
			if c.Guard == nil {
				if key := matchPatternKey(c.Pattern, patType); key != "" {
					if first, exists := seenPatterns[key]; exists {
						return "", fmt.Errorf(
							"%s: duplicate catch pattern (first at %s)",
							frontend.FormatPos(c.At),
							frontend.FormatPos(first),
						)
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
			if err := bindPatternOwnershipAliases(
				c.Pattern,
				"",
				e.ErrorLocal,
				e.ErrorType,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
			if err := bindPatternBorrowedPtrAliases(
				c.Pattern,
				"",
				e.ErrorLocal,
				e.ErrorType,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
			if err := bindPatternResourceLocals(
				c.Pattern,
				"",
				e.ErrorLocal,
				e.ErrorType,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
			if err := bindPatternRegionLocals(
				c.Pattern,
				"",
				e.ErrorLocal,
				e.ErrorType,
				types,
				module,
				imports,
				state,
			); err != nil {
				return err
			}
			if c.Guard != nil {
				guardType, _, err := checkExprWithEffects(
					c.Guard,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				)
				if err != nil {
					return err
				}
				if guardType != "bool" {
					return fmt.Errorf(
						"%s: catch guard must be Bool",
						frontend.FormatPos(c.Guard.Pos()),
					)
				}
			}
			armType, _, err := checkExprWithEffects(
				c.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(e.ResultType, armType, c.Value) {
				return fmt.Errorf(
					"%s: catch expression case type mismatch: expected '%s', got '%s'",
					frontend.FormatPos(c.At),
					e.ResultType,
					armType,
				)
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		caseVars := copyRegionVars(state.regionVars)
		caseFlow := snapshotFlow(state)
		state.regionVars = mergeRegionVars(mergedVars, caseVars)
		mergeFlowWithLabels(
			state,
			mergedFlow,
			caseFlow,
			"previous cases",
			fmt.Sprintf("case %d", i+1),
		)
		mergedVars = copyRegionVars(state.regionVars)
		mergedFlow = snapshotFlow(state)
	}
	state.regionVars = mergedVars
	restoreFlow(state, mergedFlow)
	return e.ResultType, nil
}

// ownershipArgRef records canonical call-argument access paths in source order.

// ---- exprs_callbacks_typed.go ----

var noblockForbiddenCallEffects = semanticspolicy.NoblockForbiddenCallEffects

var realtimeForbiddenCallEffects = semanticspolicy.RealtimeForbiddenCallEffects

func currentCallerSignature(effects *effectContext, funcs map[string]FuncSig) (FuncSig, bool) {
	var zero FuncSig
	if effects == nil || effects.funcName == "" {
		return zero, false
	}
	sig, ok := funcs[effects.funcName]
	if !ok {
		return zero, false
	}
	return sig, true
}

func validateCallAgainstSemanticClauses(
	callerSig FuncSig,
	calleeSig FuncSig,
	calleeName string,
	pos frontend.Position,
) error {
	return validateCallAgainstSemanticClauseTarget(
		callerSig,
		calleeSig,
		fmt.Sprintf("call to '%s'", calleeName),
		pos,
	)
}

func validateCallAgainstSemanticClauseTarget(
	callerSig FuncSig,
	calleeSig FuncSig,
	calleePhrase string,
	pos frontend.Position,
) error {
	if err := validateBudgetedSemanticCallTarget(callerSig, calleeSig, calleePhrase, pos); err != nil {
		return err
	}
	if callerSig.HasRealtime {
		if blocked := firstFuncSigForbiddenEffect(
			calleeSig,
			realtimeForbiddenCallEffects,
		); blocked != "" {
			return fmt.Errorf(
				"%s: semantic clause 'realtime' forbids %s because it is not realtime-safe (effect '%s')",
				frontend.FormatPos(pos),
				calleePhrase,
				blocked,
			)
		}
	}
	if callerSig.HasNoAlloc && funcSigHasEffect(calleeSig, "alloc") {
		return fmt.Errorf(
			"%s: semantic clause 'noalloc' forbids %s because it may allocate",
			frontend.FormatPos(pos),
			calleePhrase,
		)
	}
	if callerSig.HasNoBlock {
		if blocked := firstFuncSigForbiddenEffect(calleeSig, noblockForbiddenCallEffects); blocked != "" {
			return fmt.Errorf(
				"%s: semantic clause 'noblock' forbids %s because it may block (effect '%s')",
				frontend.FormatPos(pos),
				calleePhrase,
				blocked,
			)
		}
	}
	return nil
}

func validateBudgetedSemanticCallTarget(
	callerSig FuncSig,
	calleeSig FuncSig,
	calleePhrase string,
	pos frontend.Position,
) error {
	if !calleeSig.HasBudget {
		return nil
	}
	required := calleeSig.Budget
	if !callerSig.HasBudget {
		return budgetDiagnosticf(
			pos,
			"budget context for %s requires caller budget at least %d",
			calleePhrase,
			required,
		)
	}
	if callerSig.Budget < required {
		return budgetDiagnosticf(
			pos,
			"budget context for %s requires caller budget at least %d, got %d",
			calleePhrase,
			required,
			callerSig.Budget,
		)
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
	if err := validateBudgetedSemanticCallTarget(
		calleeSig,
		callbackSig,
		fmt.Sprintf("callback function symbol '%s' for callee '%s'", callbackName, calleeName),
		pos,
	); err != nil {
		return err
	}
	if calleeSig.HasRealtime {
		if blocked := firstFuncSigForbiddenEffect(
			callbackSig,
			realtimeForbiddenCallEffects,
		); blocked != "" {
			return fmt.Errorf(
				"%s: callback function symbol '%s' is not realtime-safe (effect '%s') for callee '%s'",
				frontend.FormatPos(pos),
				callbackName,
				blocked,
				calleeName,
			)
		}
	}
	if calleeSig.HasNoAlloc && funcSigHasEffect(callbackSig, "alloc") {
		return fmt.Errorf(
			"%s: callback function symbol '%s' may allocate but callee '%s' has semantic clause 'noalloc'",
			frontend.FormatPos(pos),
			callbackName,
			calleeName,
		)
	}
	if calleeSig.HasNoBlock {
		if blocked := firstFuncSigForbiddenEffect(
			callbackSig,
			noblockForbiddenCallEffects,
		); blocked != "" {
			return fmt.Errorf(
				("%s: callback function symbol '%s' may block (effect '%s') " +
					"but callee '%s' has semantic clause 'noblock'"),
				frontend.FormatPos(pos),
				callbackName,
				blocked,
				calleeName,
			)
		}
	}
	return nil
}

func funcSigHasEffect(sig FuncSig, effect string) bool {
	return semanticspolicy.FuncSigHasEffect(sig, effect)
}

func actorTaskWorkerBoundaryEffect(sig FuncSig) string {
	return semanticspolicy.ActorTaskWorkerBoundaryEffect(sig)
}

func firstFuncSigForbiddenEffect(sig FuncSig, forbidden []string) string {
	return semanticspolicy.FirstFuncSigForbiddenEffect(sig, forbidden)
}

func hasStrictSemanticCallClauses(sig FuncSig) bool {
	return semanticspolicy.HasStrictSemanticCallClauses(sig)
}

func firstStrictSemanticCallClause(sig FuncSig) string {
	return semanticspolicy.FirstStrictSemanticCallClause(sig)
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
		if err := validateFunctionTypedClosureAssignment(
			"closure literal",
			targetInfo,
			closure,
			locals,
			funcs,
			types,
			module,
			imports,
			closure.At,
			"callback argument",
		); err != nil {
			return "", "", err
		}
		if len(closure.Captures) > 0 {
			captureSlots, err := functionCaptureSlotCount(closure.Captures, types)
			if err != nil {
				return "", "", err
			}
			if captureSlots > FnPtrEnvSlotCount {
				if _, _, err := classifyCallableEscape(
					callableBoundaryCallback,
					closure.Captures,
					types,
				); err != nil {
					return "", "", err
				}
			}
		}
		callbackSig, ok := funcs[closure.Name]
		if !ok {
			builtSig, err := buildInterfaceFuncSig(callbackArgumentName(arg), funcSigSpec{
				ParamTypes:          append([]string(nil), targetInfo.FunctionParamTypes...),
				ParamOwnership:      append([]string(nil), targetInfo.FunctionParamOwnership...),
				ReturnType:          targetInfo.FunctionReturnType,
				ReturnRegionParam:   regionNone,
				ReturnResourceParam: regionNone,
				Effects:             append([]string(nil), targetInfo.FunctionEffects...),
			}, types)
			if err != nil {
				return "", "", err
			}
			callbackSig = builtSig
		}
		if !deferEffectValidation {
			if err := validateCallbackClauseCompatibility(
				callbackSig,
				calleeSig,
				calleeName,
				closure.At,
				callbackArgumentName(arg),
			); err != nil {
				return "", "", err
			}
		}
		return targetInfo.TypeName, closure.Name, nil
	}
	if call, ok := arg.(*frontend.CallExpr); ok {
		argType, _, err := checkExprWithEffects(
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return "", "", err
		}
		callSig, ok := funcs[call.Name]
		if !ok || !callSig.ReturnFunctionType {
			return "", "", fmt.Errorf(
				"%s: callback argument call '%s' does not return a function type",
				frontend.FormatPos(call.At),
				call.Name,
			)
		}
		if err := markFunctionTypedReturnCallMutableGlobalUse(
			callSig,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			analysis,
		); err != nil {
			return "", "", err
		}
		returnedSig, err := buildInterfaceFuncSig(callbackArgumentName(call), funcSigSpec{
			ParamTypes:          append([]string(nil), callSig.ReturnFunctionParams...),
			ParamOwnership:      append([]string(nil), callSig.ReturnFunctionParamOwnership...),
			ReturnType:          callSig.ReturnFunctionReturn,
			ThrowsType:          callSig.ReturnFunctionThrows,
			ReturnRegionParam:   regionNone,
			ReturnResourceParam: regionNone,
			Effects:             append([]string(nil), callSig.ReturnFunctionEffects...),
		}, types)
		if err != nil {
			return "", "", err
		}
		if err := validateCallbackSignature(
			returnedSig,
			calleeSig,
			paramIndex,
			call.At,
			callbackArgumentName(call),
			deferEffectValidation,
		); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(
			returnedSig,
			calleeSig,
			calleeName,
			call.At,
			callbackArgumentName(call),
		); err != nil {
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
			callbackSig, err := buildInterfaceFuncSig(callbackArgumentName(arg), funcSigSpec{
				ParamTypes:          append([]string(nil), fieldInfo.FunctionParamTypes...),
				ParamOwnership:      append([]string(nil), fieldInfo.FunctionParamOwnership...),
				ParamSlots:          paramSlots,
				ReturnType:          fieldInfo.FunctionReturnType,
				ThrowsType:          fieldInfo.FunctionThrowsType,
				ReturnRegionParam:   regionNone,
				ReturnResourceParam: regionNone,
				Effects:             append([]string(nil), fieldInfo.FunctionEffects...),
			}, types)
			if err != nil {
				return "", "", err
			}
			if err := validateCallbackSignatureForField(
				callbackSig,
				fieldInfo,
				calleeSig,
				paramIndex,
				arg.Pos(),
				callbackArgumentName(arg),
				types,
				deferEffectValidation,
			); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(
				callbackSig,
				calleeSig,
				calleeName,
				arg.Pos(),
				callbackArgumentName(arg),
			); err != nil {
				return "", "", err
			}
			return "fnptr", "", nil
		}
		localSig, ok := funcs[fieldInfo.FunctionValue]
		if !ok {
			return "", "", fmt.Errorf(
				"%s: unknown callback function symbol '%s'",
				frontend.FormatPos(arg.Pos()),
				fieldInfo.FunctionValue,
			)
		}
		if err := validateCallbackSignatureForField(
			localSig,
			fieldInfo,
			calleeSig,
			paramIndex,
			arg.Pos(),
			callbackArgumentName(arg),
			types,
			deferEffectValidation,
		); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(
			localSig,
			calleeSig,
			calleeName,
			arg.Pos(),
			callbackArgumentName(arg),
		); err != nil {
			return "", "", err
		}
		return "fnptr", fieldInfo.FunctionValue, nil
	}
	if fieldAccess, ok := arg.(*frontend.FieldAccessExpr); ok {
		globalInfo, globalSig, globalOK, err := resolveFunctionTypedGlobalFieldAccess(
			fieldAccess,
			globals,
			funcs,
		)
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
			if err := validateCallbackSignatureForLocal(
				globalSig,
				globalLocal,
				calleeSig,
				paramIndex,
				arg.Pos(),
				callbackArgumentName(arg),
				types,
				deferEffectValidation,
			); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(
				globalSig,
				calleeSig,
				calleeName,
				arg.Pos(),
				callbackArgumentName(arg),
			); err != nil {
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
				return "", "", fmt.Errorf(
					"%s: unknown callback function symbol '%s'",
					frontend.FormatPos(arg.Pos()),
					localInfo.FunctionValue,
				)
			}
			if localSig.ThrowsType != "" &&
				callbackExpectedThrowsType(calleeSig, paramIndex) == "" {
				return "", "", unsupportedThrowingCallbackSymbolError(arg.Pos(), id.Name)
			}
			targetInfo := functionParamLocalInfo(calleeSig, paramIndex)
			targetInfo.FunctionValue = localInfo.FunctionValue
			targetInfo.FunctionCaptures = append(
				[]frontend.ClosureCapture(nil),
				localInfo.FunctionCaptures...)
			if len(targetInfo.FunctionCaptures) > 0 {
				captureSlots, err := functionCaptureSlotCount(targetInfo.FunctionCaptures, types)
				if err != nil {
					return "", "", err
				}
				if captureSlots > FnPtrEnvSlotCount {
					escapeKind, handleValue, err := classifyCallableEscape(
						callableBoundaryCallback,
						targetInfo.FunctionCaptures,
						types,
					)
					if err != nil {
						return "", "", err
					}
					targetInfo.FunctionEscapeKind = escapeKind
					targetInfo.FunctionHandleValue = handleValue
				}
			}
			if err := validateCallbackSignatureForLocal(
				localSig,
				targetInfo,
				calleeSig,
				paramIndex,
				arg.Pos(),
				id.Name,
				types,
				deferEffectValidation,
			); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(
				localSig,
				calleeSig,
				calleeName,
				arg.Pos(),
				id.Name,
			); err != nil {
				return "", "", err
			}
			return targetInfo.TypeName, localInfo.FunctionValue, nil
		}
		if localInfo.FunctionValue == "" {
			paramSlots, err := functionParamSlotCount(localInfo.FunctionParamTypes, types)
			if err != nil {
				return "", "", err
			}
			callbackSig, err := buildInterfaceFuncSig(id.Name, funcSigSpec{
				ParamTypes:          append([]string(nil), localInfo.FunctionParamTypes...),
				ParamOwnership:      append([]string(nil), localInfo.FunctionParamOwnership...),
				ParamSlots:          paramSlots,
				ReturnType:          localInfo.FunctionReturnType,
				ThrowsType:          localInfo.FunctionThrowsType,
				ReturnRegionParam:   regionNone,
				ReturnResourceParam: regionNone,
				Effects:             append([]string(nil), localInfo.FunctionEffects...),
			}, types)
			if err != nil {
				return "", "", err
			}
			if err := validateCallbackSignatureForLocal(
				callbackSig,
				localInfo,
				calleeSig,
				paramIndex,
				arg.Pos(),
				id.Name,
				types,
				deferEffectValidation,
			); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(
				callbackSig,
				calleeSig,
				calleeName,
				arg.Pos(),
				id.Name,
			); err != nil {
				return "", "", err
			}
			return "fnptr", "", nil
		}
		if localInfo.GenericFunctionValue {
			return "", "", unsupportedGenericCallbackSymbolError(arg.Pos(), id.Name)
		}
		localSig, ok := funcs[localInfo.FunctionValue]
		if !ok {
			return "", "", fmt.Errorf(
				"%s: unknown callback function symbol '%s'",
				frontend.FormatPos(arg.Pos()),
				localInfo.FunctionValue,
			)
		}
		if err := validateCallbackSignatureForLocal(
			localSig,
			localInfo,
			calleeSig,
			paramIndex,
			arg.Pos(),
			id.Name,
			types,
			deferEffectValidation,
		); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(
			localSig,
			calleeSig,
			calleeName,
			arg.Pos(),
			id.Name,
		); err != nil {
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
			return "", "", fmt.Errorf(
				"%s: unknown callback function symbol '%s'",
				frontend.FormatPos(arg.Pos()),
				globalInfo.FunctionValue,
			)
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
		if err := validateCallbackSignatureForLocal(
			globalSig,
			globalLocal,
			calleeSig,
			paramIndex,
			arg.Pos(),
			id.Name,
			types,
			deferEffectValidation,
		); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(
			globalSig,
			calleeSig,
			calleeName,
			arg.Pos(),
			id.Name,
		); err != nil {
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
	if err := validateCallbackSignature(
		sig,
		calleeSig,
		paramIndex,
		arg.Pos(),
		id.Name,
		deferEffectValidation,
	); err != nil {
		return "", "", err
	}
	if err := validateCallbackClauseCompatibility(
		sig,
		calleeSig,
		calleeName,
		arg.Pos(),
		id.Name,
	); err != nil {
		return "", "", err
	}
	return "fnptr", resolved, nil
}

func unsupportedCallbackArgumentSourceError(pos frontend.Position, calleeName string) error {
	return fmt.Errorf(
		("%s: callback argument for '%s' must be a supported fnptr " +
			"source: closure literal, function-typed local/global/struct " +
			"field, direct named function/closure symbol, or " +
			"function-typed return call"),
		frontend.FormatPos(pos),
		calleeName,
	)
}

func unsupportedCallbackCaptureError(pos frontend.Position, rawName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf(
			("%s: callback argument '%s' captures %d environment slots; " +
				"captured callback arguments support at most %d fnptr " +
				"environment slots within the supported fnptr ABI"),
			frontend.FormatPos(pos),
			rawName,
			envSlots,
			FnPtrEnvSlotCount,
		)
	}
	return fmt.Errorf(
		("%s: callback argument '%s' captures local values; captured " +
			"function values cannot be passed as callback arguments in " +
			"this MVP; closure lifetime/ABI evidence is only available " +
			"for local direct calls"),
		frontend.FormatPos(pos),
		rawName,
	)
}

func unsupportedClosureLiteralCallbackCaptureError(pos frontend.Position, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf(
			("%s: callback argument 'closure literal' captures %d " +
				"environment slots; captured callback arguments support at " +
				"most %d fnptr environment slots within the supported fnptr " +
				"ABI"),
			frontend.FormatPos(pos),
			envSlots,
			FnPtrEnvSlotCount,
		)
	}
	return fmt.Errorf(
		("%s: callback argument 'closure literal' captures local " +
			"values; captured function values cannot be passed as " +
			"callback arguments in this MVP; closure lifetime/ABI " +
			"evidence is only available for local direct calls"),
		frontend.FormatPos(pos),
	)
}

func unsupportedFunctionTypedCallCaptureError(
	pos frontend.Position,
	rawName string,
	envSlots int,
) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf(
			("%s: function-typed callback '%s' captures %d environment " +
				"slots; direct function-typed calls support at most %d fnptr " +
				"environment slots within the supported fnptr ABI"),
			frontend.FormatPos(pos),
			rawName,
			envSlots,
			FnPtrEnvSlotCount,
		)
	}
	return fmt.Errorf(
		"%s: function-typed callback '%s' has unsupported captured environment size",
		frontend.FormatPos(pos),
		rawName,
	)
}

func unsupportedFunctionFieldCallCaptureError(
	pos frontend.Position,
	rawName string,
	envSlots int,
) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf(
			("%s: function-typed struct field call '%s' captures %d " +
				"environment slots; direct struct-field calls support at " +
				"most %d fnptr environment slots within the supported fnptr " +
				"ABI"),
			frontend.FormatPos(pos),
			rawName,
			envSlots,
			FnPtrEnvSlotCount,
		)
	}
	return fmt.Errorf(
		"%s: function-typed struct field call '%s' has unsupported captured environment size",
		frontend.FormatPos(pos),
		rawName,
	)
}

func unsupportedEnumPayloadCallCaptureError(
	pos frontend.Position,
	rawName string,
	envSlots int,
) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf(
			("%s: function-typed enum payload binding '%s' captures %d " +
				"environment slots; direct enum-payload calls support at " +
				"most %d fnptr environment slots within the supported fnptr " +
				"ABI"),
			frontend.FormatPos(pos),
			rawName,
			envSlots,
			FnPtrEnvSlotCount,
		)
	}
	return fmt.Errorf(
		"%s: function-typed enum payload binding '%s' has unsupported captured environment size",
		frontend.FormatPos(pos),
		rawName,
	)
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
		return validateCallbackSignature(
			visibleSig,
			calleeSig,
			paramIndex,
			pos,
			rawName,
			deferEffectValidation,
		)
	}
	if len(localInfo.FunctionCaptures) == 0 {
		return validateCallbackSignature(
			callbackSig,
			calleeSig,
			paramIndex,
			pos,
			rawName,
			deferEffectValidation,
		)
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
	return validateCallbackSignature(
		trimmed,
		calleeSig,
		paramIndex,
		pos,
		rawName,
		deferEffectValidation,
	)
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
		return validateCallbackSignature(
			callbackSig,
			calleeSig,
			paramIndex,
			pos,
			rawName,
			deferEffectValidation,
		)
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
	return validateCallbackSignature(
		visibleSig,
		calleeSig,
		paramIndex,
		pos,
		rawName,
		deferEffectValidation,
	)
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

func resolveFunctionFieldCall(
	name string,
	locals map[string]LocalInfo,
) (FunctionFieldInfo, bool, error) {
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

func resolveFunctionFieldArgument(
	expr frontend.Expr,
	locals map[string]LocalInfo,
) (FunctionFieldInfo, bool, error) {
	name := callbackArgumentName(expr)
	if name == "" {
		return FunctionFieldInfo{}, false, nil
	}
	return resolveFunctionFieldCall(name, locals)
}

func callbackArgumentName(expr frontend.Expr) string {
	return semanticsexpressions.CallbackArgumentName(expr)
}

func validateCallbackSignature(
	callbackSig FuncSig,
	calleeSig FuncSig,
	paramIndex int,
	pos frontend.Position,
	rawName string,
	deferEffectValidation bool,
) error {
	if paramIndex >= len(calleeSig.ParamFunctionParams) ||
		paramIndex >= len(calleeSig.ParamFunctionReturns) {
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
		return fmt.Errorf(
			"%s: callback function symbol '%s' has incompatible parameter count: expected %d, got %d",
			frontend.FormatPos(pos),
			rawName,
			len(wantParams),
			len(callbackSig.ParamTypes),
		)
	}
	wantOwnership := []string(nil)
	if paramIndex < len(calleeSig.ParamFunctionOwnership) {
		wantOwnership = calleeSig.ParamFunctionOwnership[paramIndex]
	}
	if err := validateFunctionTypeParamOwnership(
		wantOwnership,
		callbackSig.ParamOwnership,
		len(wantParams),
		pos,
		"callback function symbol",
		rawName,
	); err != nil {
		return err
	}
	for i := range wantParams {
		if wantParams[i] != callbackSig.ParamTypes[i] {
			return fmt.Errorf(
				"%s: callback function symbol '%s' parameter %d type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(pos),
				rawName,
				i+1,
				wantParams[i],
				callbackSig.ParamTypes[i],
			)
		}
	}
	if wantReturn != "" && wantReturn != callbackSig.ReturnType {
		return fmt.Errorf(
			"%s: callback function symbol '%s' return type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			rawName,
			wantReturn,
			callbackSig.ReturnType,
		)
	}
	if wantReturnOwnership != callbackSig.ReturnOwnership {
		return fmt.Errorf(
			"%s: callback function symbol '%s' return ownership mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			rawName,
			ownershipDisplay(wantReturnOwnership),
			ownershipDisplay(callbackSig.ReturnOwnership),
		)
	}
	if wantThrows != callbackSig.ThrowsType {
		return fmt.Errorf(
			"%s: callback function symbol '%s' throws type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			rawName,
			wantThrows,
			callbackSig.ThrowsType,
		)
	}
	if !deferEffectValidation {
		if err := validateFunctionTypeCallableEffects(
			wantEffects,
			callbackSig.Effects,
			pos,
			"callback function symbol",
			rawName,
		); err != nil {
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

func validateFunctionTypedThrowCall(
	throwsType string,
	e *frontend.CallExpr,
	state *regionState,
) error {
	isTryCall := state != nil && state.allowThrowDepth > 0 && state.allowThrowCall == e
	isCatchCall := state != nil && state.allowCatchDepth > 0 && state.allowCatchCall == e
	if throwsType == "" {
		if isTryCall {
			return fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		if isCatchCall {
			return fmt.Errorf(
				"%s: catch expects a throwing function call",
				frontend.FormatPos(e.At),
			)
		}
		return nil
	}
	if !isTryCall && !isCatchCall {
		return fmt.Errorf(
			"%s: call to throwing function '%s' requires try",
			frontend.FormatPos(e.At),
			e.Name,
		)
	}
	if isTryCall && state.throwType == "" {
		return fmt.Errorf("%s: try is only allowed in throwing functions", frontend.FormatPos(e.At))
	}
	if isTryCall && !typesCompatibleWithNullPtr(state.throwType, throwsType, e) {
		return fmt.Errorf(
			"%s: thrown error type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(e.At),
			state.throwType,
			throwsType,
		)
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
			return "", regionNone, fmt.Errorf(
				"%s: send_typed does not accept explicit type arguments",
				frontend.FormatPos(e.At),
			)
		}
		if len(e.Args) != 2 {
			return "", regionNone, fmt.Errorf(
				"%s: send_typed expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		targetType, _, err := checkExprWithEffects(
			e.Args[0],
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return "", regionNone, err
		}
		if targetType != "actor" {
			return "", regionNone, fmt.Errorf(
				"%s: type mismatch for 'core.send_typed' arg 1",
				frontend.FormatPos(e.Args[0].Pos()),
			)
		}
		msgType, _, err := checkExprWithEffects(
			e.Args[1],
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return "", regionNone, err
		}
		info, ok := types[msgType]
		if !ok || info.Kind != TypeEnum {
			return "", regionNone, fmt.Errorf(
				"%s: send_typed expects an enum message",
				frontend.FormatPos(e.Args[1].Pos()),
			)
		}
		if err := validateTypedActorMessageType(msgType, types, map[string]bool{}); err != nil {
			return "", regionNone, fmt.Errorf("%s: %v", frontend.FormatPos(e.Args[1].Pos()), err)
		}
		transferOwners := actorTransferOwnerPayloads(e.Args[1], msgType, types, module, imports)
		if err := validateActorBoundaryPayloadExpr(
			e.Args[1],
			msgType,
			types,
			module,
			imports,
			state,
			transferOwners,
		); err != nil {
			return "", regionNone, err
		}
		if err := checkBorrowedEscape(
			e.Args[1],
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
			func(borrowedName string) error {
				return ownershipDiagnosticf(e.Args[1].Pos(), (("cannot send borrowed view across actor " +
					"boundary; use .copy()") +
					" (borrowed value derived from '%s' cannot cross actor " +
					"boundary without copy)"), borrowedName)
			},
		); err != nil {
			return "", regionNone, err
		}
		if err := consumeTypedActorTransferPayloads(
			e.Args[1],
			msgType,
			locals,
			types,
			module,
			imports,
			state,
		); err != nil {
			return "", regionNone, err
		}
		e.Name = resolved
		return "i32", regionNone, nil
	case "core.recv_typed":
		if len(e.Args) != 0 {
			return "", regionNone, fmt.Errorf(
				"%s: recv_typed expects 0 arguments",
				frontend.FormatPos(e.At),
			)
		}
		if len(e.TypeArgs) != 1 {
			return "", regionNone, fmt.Errorf(
				"%s: recv_typed expects one explicit type argument",
				frontend.FormatPos(e.At),
			)
		}
		typeName, err := resolveTypeName(&e.TypeArgs[0], module, imports)
		if err != nil {
			return "", regionNone, err
		}
		e.TypeArgs[0].Name = typeName
		info, ok := types[typeName]
		if !ok || info.Kind != TypeEnum {
			return "", regionNone, fmt.Errorf(
				"%s: recv_typed expects an enum type argument",
				frontend.FormatPos(e.TypeArgs[0].At),
			)
		}
		if err := validateTypedActorMessageType(typeName, types, map[string]bool{}); err != nil {
			return "", regionNone, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
		}
		e.Name = resolved
		return typeName, regionNone, nil
	default:
		return "", regionNone, fmt.Errorf(
			"%s: unknown typed actor builtin '%s'",
			frontend.FormatPos(e.At),
			resolved,
		)
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
		return "", regionNone, fmt.Errorf(
			"%s: %s expects one explicit error type argument",
			frontend.FormatPos(e.At),
			resolved,
		)
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
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32_typed expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		if resolved == "core.task_spawn_group_i32_typed" && len(e.Args) != 2 {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_group_i32_typed expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		workerArg := 0
		if resolved == "core.task_spawn_group_i32_typed" {
			groupType, _, err := checkExprWithEffects(
				e.Args[0],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return "", regionNone, err
			}
			if groupType != "task.group" {
				return "", regionNone, fmt.Errorf(
					"%s: type mismatch for 'core.task_spawn_group_i32_typed' arg 1",
					frontend.FormatPos(e.Args[0].Pos()),
				)
			}
			if err := checkResourceCallArg(
				resolved,
				"task.group",
				e.Args[0],
				funcs,
				module,
				imports,
				state,
			); err != nil {
				return "", regionNone, err
			}
			workerArg = 1
		}
		lit, ok := e.Args[workerArg].(*frontend.StringLitExpr)
		if !ok {
			if resolved == "core.task_spawn_group_i32_typed" {
				return "", regionNone, fmt.Errorf(
					"%s: task_spawn_group_i32_typed expects a string literal worker name",
					frontend.FormatPos(e.At),
				)
			}
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32_typed expects a string literal",
				frontend.FormatPos(e.At),
			)
		}
		raw := string(lit.Value)
		if raw == "" {
			if resolved == "core.task_spawn_group_i32_typed" {
				return "", regionNone, fmt.Errorf(
					"%s: task_spawn_group_i32_typed expects a non-empty name",
					frontend.FormatPos(e.At),
				)
			}
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32_typed expects a non-empty name",
				frontend.FormatPos(e.At),
			)
		}
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf(
				"%s: %s target must be a user function, got '%s'",
				frontend.FormatPos(e.At),
				taskTypedSpawnName(resolved),
				target,
			)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: unknown function '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			if handleInfo.SlotCount > 4 {
				return "", regionNone, fmt.Errorf(
					"%s: %s target must have shape func %s() -> i32",
					frontend.FormatPos(e.At),
					taskTypedSpawnName(resolved),
					target,
				)
			}
			return "", regionNone, fmt.Errorf(
				"%s: %s target must have shape func %s() -> i32 throws %s",
				frontend.FormatPos(e.At),
				taskTypedSpawnName(resolved),
				target,
				displayTypeName(errorType, module),
			)
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf(
				"%s: %s target must be synchronous",
				frontend.FormatPos(e.At),
				taskTypedSpawnName(resolved),
			)
		}
		if handleInfo.SlotCount > 4 {
			if targetSig.ThrowsType != "" && targetSig.ThrowsType != errorType {
				return "", regionNone, fmt.Errorf(
					"%s: %s target must throw '%s' (or be non-throwing in staged mode)",
					frontend.FormatPos(e.At),
					taskTypedSpawnName(resolved),
					displayTypeName(errorType, module),
				)
			}
		} else if targetSig.ThrowsType != errorType {
			return "", regionNone, fmt.Errorf(
				"%s: %s target must throw '%s'",
				frontend.FormatPos(e.At),
				taskTypedSpawnName(resolved),
				displayTypeName(errorType, module),
			)
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf(
				"%s: %s target '%s' touches mutable global state and cannot cross task boundary",
				frontend.FormatPos(e.At),
				taskTypedSpawnName(resolved),
				target,
			)
		}
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf(
				"%s: %s target '%s' uses effect '%s' and cannot cross task boundary",
				frontend.FormatPos(e.At),
				taskTypedSpawnName(resolved),
				target,
				blocked,
			)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf(
				"%s: %s target '%s' is not sendable across task boundary",
				frontend.FormatPos(e.At),
				taskTypedSpawnName(resolved),
				target,
			)
		}
		lit.Value = []byte(target)
		e.Name = resolved
		return handleType, regionNone, nil
	case "core.task_join_i32_typed", "core.task_join_group_i32_typed":
		if len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf(
				"%s: task_join_i32_typed expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		argType, _, err := checkExprWithEffects(
			e.Args[0],
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return "", regionNone, err
		}
		if !typesCompatibleWithNullPtr(handleType, argType, e.Args[0]) {
			return "", regionNone, fmt.Errorf(
				"%s: type mismatch for '%s' arg 1: expected '%s', got '%s'",
				frontend.FormatPos(e.Args[0].Pos()),
				resolved,
				handleType,
				argType,
			)
		}
		isTryCall := state != nil && state.allowThrowDepth > 0 && state.allowThrowCall == e
		isCatchCall := state != nil && state.allowCatchDepth > 0 && state.allowCatchCall == e
		if !isTryCall && !isCatchCall {
			return "", regionNone, fmt.Errorf(
				"%s: call to throwing function '%s' requires try",
				frontend.FormatPos(e.At),
				resolved,
			)
		}
		if isTryCall && state.throwType == "" {
			return "", regionNone, fmt.Errorf(
				"%s: try is only allowed in throwing functions",
				frontend.FormatPos(e.At),
			)
		}
		if isTryCall && state.throwType != errorType {
			return "", regionNone, fmt.Errorf(
				"%s: thrown error type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(e.At),
				state.throwType,
				errorType,
			)
		}
		markTaskHandleJoined(e.Args[0], funcs, module, imports, state)
		e.Name = resolved
		return "i32", regionNone, nil
	default:
		return "", regionNone, fmt.Errorf(
			"%s: unknown typed task builtin '%s'",
			frontend.FormatPos(e.At),
			resolved,
		)
	}
}

// ---- exprs_calls.go ----

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
	if rewritten, err := rewriteSliceViewMethodCall(e, locals, globals, types); rewritten ||
		err != nil {
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
			return "", regionNone, unsupportedFunctionTypedExplicitTypeArgsError(
				e.At,
				fmt.Sprintf("function-typed struct field call '%s'", e.Name),
			)
		}
		if len(e.Args) != len(fieldInfo.FunctionParamTypes) {
			return "", regionNone, fmt.Errorf(("%s: wrong argument count for function-typed struct field " +
				"call '%s'"), frontend.FormatPos(e.At), e.Name)
		}
		if err := validateFunctionTypedValueCallLabels(
			e,
			"function-typed struct field call",
			e.Name,
		); err != nil {
			return "", regionNone, err
		}
		if fieldInfo.FunctionValue != "" {
			targetSig, ok := funcs[fieldInfo.FunctionValue]
			if !ok {
				return "", regionNone, fmt.Errorf(
					"%s: unknown function symbol '%s'",
					frontend.FormatPos(e.At),
					fieldInfo.FunctionValue,
				)
			}
			markFunctionTargetMutableGlobalUse(targetSig, analysis)
			if targetSig.Generic {
				return "", regionNone, fmt.Errorf(("%s: generic function symbol '%s' is not supported for " +
					"function-typed struct field call in this MVP"), frontend.FormatPos(e.At), e.Name)
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
				if err := validateCallAgainstSemanticClauseTarget(
					callerSig,
					targetSig,
					fmt.Sprintf("function-typed struct field call '%s'", e.Name),
					e.At,
				); err != nil {
					return "", regionNone, err
				}
			}
		} else if hasCallerSig {
			if err := validateFunctionTypeCallableEffects(
				callerSig.Effects,
				fieldInfo.FunctionEffects,
				e.At,
				"function-typed struct field call",
				e.Name,
			); err != nil {
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
			argType, argRegion, err := checkExprWithEffects(
				arg,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return "", regionNone, err
			}
			argRegions[i] = argRegion
			if !typesCompatibleWithNullPtr(fieldInfo.FunctionParamTypes[i], argType, arg) {
				return "", regionNone, fmt.Errorf((("%s: type mismatch for function-typed struct " +
					"field call '%s' ") +
					"arg %d"), frontend.FormatPos(arg.Pos()), e.Name, i+1)
			}
			paramOwnership := ownershipAt(fieldInfo.FunctionParamOwnership, i)
			if paramOwnership == "" {
				if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed value derived " +
						"from '%s' cannot be passed to ") +
						"non-borrow parameter %d of function-typed struct field call " +
						"'%s'"), borrowedName, i+1, e.Name)
				}
				if argType == "ptr" {
					if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed value derived " +
							"from '%s' cannot be passed to ") +
							"non-borrow parameter %d of function-typed struct field call " +
							"'%s'"), borrowedName, i+1, e.Name)
					}
				}
				if argType != "ptr" && (typeMayContainRegion(
					argType,
					types,
				) || typeMayContainPtr(
					argType,
					types,
				)) {
					if err := checkBorrowedEscape(
						arg,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						state,
						effects,
						analysis,
						func(borrowedName string) error {
							return ownershipDiagnosticf(arg.Pos(), (("borrowed value derived from '%s' cannot " +
								"be passed to ") +
								"non-borrow parameter %d of function-typed struct field call " +
								"'%s'"), borrowedName, i+1, e.Name)
						},
					); err != nil {
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
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed value derived " +
						"from '%s' cannot be consumed by ") +
						"function-typed struct field call '%s'"), borrowedName, e.Name)
				}
				if argType == "ptr" {
					if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed value derived " +
							"from '%s' cannot be consumed by ") +
							"function-typed struct field call '%s'"), borrowedName, e.Name)
					}
				}
				if argType != "ptr" && (typeMayContainRegion(
					argType,
					types,
				) || typeMayContainPtr(
					argType,
					types,
				)) {
					if err := checkBorrowedEscape(
						arg,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						state,
						effects,
						analysis,
						func(borrowedName string) error {
							return ownershipDiagnosticf(arg.Pos(), (("borrowed value derived from '%s' cannot " +
								"be consumed by ") +
								"function-typed struct field call '%s'"), borrowedName, e.Name)
						},
					); err != nil {
						return "", regionNone, err
					}
				}
				path := name
				if first, exists := findOwnershipAlias(inoutArgs, path); exists {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("consumed argument '%s' " +
						"aliases inout argument in ") +
						"function-typed struct field call '%s' (inout at %s)"), path, e.Name, frontend.FormatPos(
						first.pos,
					))
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
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed argument '%s' " +
							"aliases inout argument in ") +
							"function-typed struct field call '%s' (inout at %s)"), path, e.Name, frontend.FormatPos(
							first.pos,
						))
					}
					borrowArgs = append(borrowArgs, ownershipArgRef{path: path, pos: arg.Pos()})
				}
			}
			if paramOwnership == "inout" {
				path, ok := canonicalOwnershipAccessPath(arg)
				if !ok {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("inout argument for " +
						"function-typed struct field call '%s' ") +
						"must be a mutable local value"), e.Name)
				}
				if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed value derived " +
						"from '%s' cannot be passed as inout ") +
						"to function-typed struct field call '%s'"), borrowedName, e.Name)
				}
				if argType == "ptr" {
					if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed value derived " +
							"from '%s' cannot be passed as inout ") +
							"to function-typed struct field call '%s'"), borrowedName, e.Name)
					}
				}
				if argType != "ptr" && (typeMayContainRegion(
					argType,
					types,
				) || typeMayContainPtr(
					argType,
					types,
				)) {
					if err := checkBorrowedEscape(
						arg,
						locals,
						globals,
						funcs,
						types,
						module,
						imports,
						state,
						effects,
						analysis,
						func(borrowedName string) error {
							return ownershipDiagnosticf(arg.Pos(), (("borrowed value derived from '%s' cannot " +
								"be passed as inout ") +
								"to function-typed struct field call '%s'"), borrowedName, e.Name)
						},
					); err != nil {
						return "", regionNone, err
					}
				}
				if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
					return "", regionNone, err
				}
				targetInfo, _, err := resolveAssignTarget(arg, locals, globals, types)
				if err != nil || !targetInfo.Mutable || targetInfo.Global {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("inout argument '%s' for " +
						"function-typed struct field call ") +
						"'%s' must be mutable"), path, e.Name)
				}
				if first, exists := findOwnershipAlias(inoutArgs, path); exists {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("inout argument '%s' used " +
						"more than once in function-typed ") +
						"struct field call '%s' (first at %s)"), path, e.Name, frontend.FormatPos(first.pos))
				}
				if first, exists := findOwnershipAlias(borrowArgs, path); exists {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("inout argument '%s' " +
						"aliases borrowed argument in ") +
						"function-typed struct field call '%s' (borrow at %s)"), path, e.Name, frontend.FormatPos(
						first.pos,
					))
				}
				if first, exists := findOwnershipAlias(consumeArgRefs, path); exists {
					return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("inout argument '%s' " +
						"aliases consumed argument in ") +
						"function-typed struct field call '%s' (consume at %s)"), path, e.Name, frontend.FormatPos(
						first.pos,
					))
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
					return "", regionNone, ownershipDiagnosticf(e.Args[i].Pos(), (("value '%s' consumed " +
						"more than once in function-typed struct ") +
						"field call '%s'"), name, e.Name)
				}
				if resourceValuesAlias(
					consumeArgs[j],
					consumeArgTypes[j],
					name,
					consumeArgTypes[i],
					types,
					state,
				) {
					return "", regionNone, ownershipDiagnosticf(e.Args[i].Pos(), (("value '%s' consumed " +
						"more than once in function-typed struct ") +
						"field call '%s'"), name, e.Name)
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
		return fieldInfo.FunctionReturnType, functionTypedBorrowReturnRegion(
			fieldInfo.FunctionReturnOwnership,
			fieldInfo.FunctionParamOwnership,
			argRegions,
			state,
		), nil
	} else if local, ok := locals[e.Name]; ok {
		if analysis != nil && local.FunctionTouchesMutableGlobals {
			analysis.touchesMutableGlobals = true
		}
		if local.FunctionEnumPayload && local.FunctionValue != "" && local.FunctionTypeValue {
			targetSig, ok := funcs[local.FunctionValue]
			if !ok {
				return "", regionNone, fmt.Errorf(
					"%s: unknown function symbol '%s'",
					frontend.FormatPos(e.At),
					local.FunctionValue,
				)
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
		if local.FunctionValue == "" || (local.FunctionTypeValue && len(
			local.FunctionCaptures,
		) == 0 && local.SlotCount == FnPtrSlotCount) {
			if !local.FunctionTypeValue {
				return "", regionNone, unsupportedFunctionValueCallError(e.At, e.Name)
			}
			if len(local.FunctionCaptures) > 0 {
				return "", regionNone, fmt.Errorf(("%s: function-typed callback '%s' captures local values; " +
					"captured function values cannot be called through function " +
					"type in this MVP"), frontend.FormatPos(e.At), e.Name)
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
				return "", regionNone, fmt.Errorf(
					"%s: wrong argument count for %s",
					frontend.FormatPos(e.At),
					valueCallPhrase,
				)
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
				argType, argRegion, err := checkExprWithEffects(
					arg,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
				)
				if err != nil {
					return "", regionNone, err
				}
				argRegions[i] = argRegion
				if !typesCompatibleWithNullPtr(local.FunctionParamTypes[i], argType, arg) {
					return "", regionNone, fmt.Errorf(
						"%s: type mismatch for %s arg %d",
						frontend.FormatPos(arg.Pos()),
						valueCallPhrase,
						i+1,
					)
				}
				paramOwnership := ownershipAt(local.FunctionParamOwnership, i)
				if paramOwnership == "" {
					if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed value derived " +
							"from '%s' cannot be passed to ") +
							"non-borrow parameter %d of %s"), borrowedName, i+1, valueCallPhrase)
					}
					if argType == "ptr" {
						if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
							return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed value derived " +
								"from '%s' cannot be passed to ") +
								"non-borrow parameter %d of %s"), borrowedName, i+1, valueCallPhrase)
						}
					}
					if argType != "ptr" && (typeMayContainRegion(
						argType,
						types,
					) || typeMayContainPtr(
						argType,
						types,
					)) {
						if err := checkBorrowedEscape(
							arg,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							state,
							effects,
							analysis,
							func(borrowedName string) error {
								return ownershipDiagnosticf(arg.Pos(), (("borrowed value derived from '%s' cannot " +
									"be passed to ") +
									"non-borrow parameter %d of %s"), borrowedName, i+1, valueCallPhrase)
							},
						); err != nil {
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
						return "", regionNone, ownershipDiagnosticf(
							arg.Pos(),
							"borrowed value derived from '%s' cannot be consumed by %s",
							borrowedName,
							valueCallPhrase,
						)
					}
					if argType == "ptr" {
						if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
							return "", regionNone, ownershipDiagnosticf(
								arg.Pos(),
								"borrowed value derived from '%s' cannot be consumed by %s",
								borrowedName,
								valueCallPhrase,
							)
						}
					}
					if argType != "ptr" && (typeMayContainRegion(
						argType,
						types,
					) || typeMayContainPtr(
						argType,
						types,
					)) {
						if err := checkBorrowedEscape(
							arg,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							state,
							effects,
							analysis,
							func(borrowedName string) error {
								return ownershipDiagnosticf(
									arg.Pos(),
									"borrowed value derived from '%s' cannot be consumed by %s",
									borrowedName,
									valueCallPhrase,
								)
							},
						); err != nil {
							return "", regionNone, err
						}
					}
					path := name
					if first, exists := findOwnershipAlias(inoutArgs, path); exists {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("consumed argument '%s' " +
							"aliases inout argument in %s (inout ") +
							"at %s)"), path, valueCallPhrase, frontend.FormatPos(first.pos))
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
							return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed argument '%s' " +
								"aliases inout argument in %s (inout ") +
								"at %s)"), path, valueCallPhrase, frontend.FormatPos(first.pos))
						}
						borrowArgs = append(borrowArgs, ownershipArgRef{path: path, pos: arg.Pos()})
					}
				}
				if paramOwnership == "inout" {
					path, ok := canonicalOwnershipAccessPath(arg)
					if !ok {
						return "", regionNone, ownershipDiagnosticf(
							arg.Pos(),
							"inout argument for %s must be a mutable local value",
							valueCallPhrase,
						)
					}
					if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed value derived " +
							"from '%s' cannot be passed as inout ") +
							"to %s"), borrowedName, valueCallPhrase)
					}
					if argType == "ptr" {
						if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
							return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("borrowed value derived " +
								"from '%s' cannot be passed as inout ") +
								"to %s"), borrowedName, valueCallPhrase)
						}
					}
					if argType != "ptr" && (typeMayContainRegion(
						argType,
						types,
					) || typeMayContainPtr(
						argType,
						types,
					)) {
						if err := checkBorrowedEscape(
							arg,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							state,
							effects,
							analysis,
							func(borrowedName string) error {
								return ownershipDiagnosticf(arg.Pos(), (("borrowed value derived from '%s' cannot " +
									"be passed as inout ") +
									"to %s"), borrowedName, valueCallPhrase)
							},
						); err != nil {
							return "", regionNone, err
						}
					}
					if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
						return "", regionNone, err
					}
					targetInfo, _, err := resolveAssignTarget(arg, locals, globals, types)
					if err != nil || !targetInfo.Mutable || targetInfo.Global {
						return "", regionNone, ownershipDiagnosticf(
							arg.Pos(),
							"inout argument '%s' for %s must be mutable",
							path,
							valueCallPhrase,
						)
					}
					if first, exists := findOwnershipAlias(inoutArgs, path); exists {
						return "", regionNone, ownershipDiagnosticf(
							arg.Pos(),
							"inout argument '%s' used more than once in %s (first at %s)",
							path,
							valueCallPhrase,
							frontend.FormatPos(first.pos),
						)
					}
					if first, exists := findOwnershipAlias(borrowArgs, path); exists {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("inout argument '%s' " +
							"aliases borrowed argument in %s (borrow ") +
							"at %s)"), path, valueCallPhrase, frontend.FormatPos(first.pos))
					}
					if first, exists := findOwnershipAlias(consumeArgRefs, path); exists {
						return "", regionNone, ownershipDiagnosticf(arg.Pos(), (("inout argument '%s' " +
							"aliases consumed argument in %s ") +
							"(consume at %s)"), path, valueCallPhrase, frontend.FormatPos(first.pos))
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
						return "", regionNone, ownershipDiagnosticf(
							e.Args[i].Pos(),
							"value '%s' consumed more than once in %s",
							name,
							valueCallPhrase,
						)
					}
					if resourceValuesAlias(
						consumeArgs[j],
						consumeArgTypes[j],
						name,
						consumeArgTypes[i],
						types,
						state,
					) {
						return "", regionNone, ownershipDiagnosticf(
							e.Args[i].Pos(),
							"value '%s' consumed more than once in %s",
							name,
							valueCallPhrase,
						)
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
					declaredSig, err := buildInterfaceFuncSig(e.Name, funcSigSpec{
						ParamTypes:          append([]string(nil), local.FunctionParamTypes...),
						ParamOwnership:      append([]string(nil), local.FunctionParamOwnership...),
						ParamSlots:          paramSlots,
						ReturnType:          local.FunctionReturnType,
						ReturnRegionParam:   regionNone,
						ReturnResourceParam: regionNone,
						Effects:             append([]string(nil), local.FunctionEffects...),
					}, types)
					if err != nil {
						return "", regionNone, err
					}
					semanticCallPhrase := fmt.Sprintf("call to '%s'", e.Name)
					if local.FunctionEnumPayload {
						semanticCallPhrase = valueCallPhrase
					}
					if err := validateCallAgainstSemanticClauseTarget(
						callerSig,
						declaredSig,
						semanticCallPhrase,
						e.At,
					); err != nil {
						return "", regionNone, err
					}
				} else {
					targetSig, ok := funcs[local.FunctionValue]
					if !ok {
						return "", regionNone, fmt.Errorf(
							"%s: unknown function symbol '%s'",
							frontend.FormatPos(e.At),
							local.FunctionValue,
						)
					}
					semanticCallPhrase := fmt.Sprintf("call to callback '%s'", e.Name)
					if local.FunctionEnumPayload {
						semanticCallPhrase = valueCallPhrase
					}
					if err := validateCallAgainstSemanticClauseTarget(
						callerSig,
						targetSig,
						semanticCallPhrase,
						e.At,
					); err != nil {
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
			return local.FunctionReturnType, functionTypedBorrowReturnRegion(
				local.FunctionReturnOwnership,
				local.FunctionParamOwnership,
				argRegions,
				state,
			), nil
		}
		if local.GenericFunctionValue {
			return "", regionNone, unsupportedGenericClosureDirectCallError(e.At, e.Name)
		}
		if local.FunctionTypeValue && local.FunctionReturnType != "" {
			targetSig, ok := funcs[local.FunctionValue]
			if !ok {
				return "", regionNone, fmt.Errorf(
					"%s: unknown function symbol '%s'",
					frontend.FormatPos(e.At),
					local.FunctionValue,
				)
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
					return "", regionNone, fmt.Errorf(
						"%s: wrong argument count for %s",
						frontend.FormatPos(e.At),
						valueCallPhrase,
					)
				}
				if err := validateFunctionTypedValueCallLabels(
					e,
					"function-typed callback",
					e.Name,
				); err != nil {
					return "", regionNone, err
				}
				if hasCallerSig {
					if err := validateCallAgainstSemanticClauseTarget(
						callerSig,
						targetSig,
						valueCallPhrase,
						e.At,
					); err != nil {
						return "", regionNone, err
					}
				}
				argRegions, err := checkFunctionTypedCallArguments(
					e,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					local.FunctionParamTypes,
					local.FunctionParamOwnership,
					valueCallPhrase,
				)
				if err != nil {
					return "", regionNone, err
				}
				if err := effects.requireAll(e.At, local.FunctionEffects); err != nil {
					return "", regionNone, err
				}
				if err := validateFunctionTypedThrowCall(local.FunctionThrowsType, e, state); err != nil {
					return "", regionNone, err
				}
				return local.FunctionReturnType, functionTypedBorrowReturnRegion(
					local.FunctionReturnOwnership,
					local.FunctionParamOwnership,
					argRegions,
					state,
				), nil
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
			return "", regionNone, fmt.Errorf(
				"%s: unknown function symbol '%s'",
				frontend.FormatPos(e.At),
				global.FunctionValue,
			)
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
		return checkTypedActorBuiltin(
			e,
			resolved,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
	}
	if resolved == "core.task_spawn_i32_typed" || resolved == "core.task_spawn_group_i32_typed" ||
		resolved == "core.task_join_i32_typed" || resolved == "core.task_join_group_i32_typed" {
		return checkTypedTaskBuiltin(
			e,
			resolved,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
	}
	sig, ok := funcs[resolved]
	if !ok {
		if ctorType, ctorRegion, handled, err := checkStructConstructorCallWithEffects(
			e,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		); handled {
			return ctorType, ctorRegion, err
		}
		if diagnostic, ok := atomicBuiltinDiagnostic(resolved); ok {
			return "", regionNone, fmt.Errorf("%s: %s", frontend.FormatPos(e.At), diagnostic)
		}
		return "", regionNone, fmt.Errorf(
			"%s: unknown function '%s'",
			frontend.FormatPos(e.At),
			resolved,
		)
	}
	if err := ensureFuncVisible(resolved, sig, module, e.At); err != nil {
		return "", regionNone, err
	}
	if sig.Generic {
		return "", regionNone, fmt.Errorf(
			"%s: generic function '%s' could not be monomorphized; use inferable value arguments",
			frontend.FormatPos(e.At),
			e.Name,
		)
	}
	if len(e.TypeArgs) > 0 {
		if functionTypedGlobalCallName != "" {
			return "", regionNone, unsupportedFunctionTypedExplicitTypeArgsError(
				e.At,
				fmt.Sprintf("function-typed global call '%s'", functionTypedGlobalCallName),
			)
		}
		return "", regionNone, fmt.Errorf(
			"%s: explicit type arguments are only supported for recv_typed",
			frontend.FormatPos(e.At),
		)
	}
	if hasCallerSig {
		semanticCallPhrase := fmt.Sprintf("call to '%s'", resolved)
		if functionTypedGlobalCallName != "" {
			semanticCallPhrase = fmt.Sprintf(
				"function-typed global call '%s'",
				functionTypedGlobalCallName,
			)
		}
		if err := validateCallAgainstSemanticClauseTarget(
			callerSig,
			sig,
			semanticCallPhrase,
			e.At,
		); err != nil {
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
			return "", regionNone, fmt.Errorf(
				"%s: call to throwing function '%s' requires try",
				frontend.FormatPos(e.At),
				resolved,
			)
		}
		if isTryCall && state.throwType == "" {
			return "", regionNone, fmt.Errorf(
				"%s: try is only allowed in throwing functions",
				frontend.FormatPos(e.At),
			)
		}
		if isTryCall && !typesCompatibleWithNullPtr(state.throwType, sig.ThrowsType, e) {
			return "", regionNone, fmt.Errorf(
				"%s: thrown error type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(e.At),
				state.throwType,
				sig.ThrowsType,
			)
		}
	} else if isTryCall {
		return "", regionNone, fmt.Errorf(
			"%s: try expects a throwing function call",
			frontend.FormatPos(e.At),
		)
	} else if isCatchCall {
		return "", regionNone, fmt.Errorf(
			"%s: catch expects a throwing function call",
			frontend.FormatPos(e.At),
		)
	}
	isAwaitCall := state != nil && state.allowAwaitDepth > 0 && state.allowAwaitCall == e
	if sig.Async {
		if !isAwaitCall {
			return "", regionNone, fmt.Errorf(
				"%s: call to async function '%s' requires await",
				frontend.FormatPos(e.At),
				resolved,
			)
		}
		if !state.async {
			return "", regionNone, fmt.Errorf(
				"%s: await is only allowed in async functions",
				frontend.FormatPos(e.At),
			)
		}
	} else if isAwaitCall {
		return "", regionNone, fmt.Errorf(
			"%s: await expects an async function call",
			frontend.FormatPos(e.At),
		)
	}
	if (resolved == "core.actor_dispatch" || resolved == "core.actor_main_entry_id") &&
		!strings.HasPrefix(module, "__") {
		return "", regionNone, fmt.Errorf(
			"%s: '%s' is reserved for internal runtime modules",
			frontend.FormatPos(e.At),
			resolved,
		)
	}
	callTargetPhrase := fmt.Sprintf("'%s'", resolved)
	callActionPhrase := fmt.Sprintf("call to '%s'", resolved)
	if functionTypedGlobalCallName != "" {
		callTargetPhrase = fmt.Sprintf(
			"function-typed global call '%s'",
			functionTypedGlobalCallName,
		)
		callActionPhrase = callTargetPhrase
	}
	if len(e.Args) != len(sig.ParamTypes) {
		return "", regionNone, fmt.Errorf(
			"%s: wrong argument count for %s",
			frontend.FormatPos(e.At),
			callTargetPhrase,
		)
	}
	if len(e.ArgLabels) > 0 && functionTypedGlobalCallName != "" {
		if err := validateFunctionTypedValueCallLabels(
			e,
			"function-typed global call",
			functionTypedGlobalCallName,
		); err != nil {
			return "", regionNone, err
		}
	} else if len(e.ArgLabels) > 0 {
		if len(e.ArgLabels) != len(e.Args) {
			return "", regionNone, fmt.Errorf(
				"%s: internal error: call argument labels are inconsistent",
				frontend.FormatPos(e.At),
			)
		}
		if len(sig.ParamNames) != len(e.Args) {
			return "", regionNone, fmt.Errorf(
				"%s: argument labels are not supported for '%s'",
				frontend.FormatPos(e.At),
				resolved,
			)
		}
		for i, label := range e.ArgLabels {
			if label == "" {
				return "", regionNone, fmt.Errorf((("%s: cannot mix labeled and unlabeled arguments " +
					"in call to ") +
					"'%s'"), frontend.FormatPos(e.Args[i].Pos()), resolved)
			}
			if sig.ParamNames[i] == "" || label != sig.ParamNames[i] {
				return "", regionNone, fmt.Errorf(("%s: argument label mismatch for '%s': expected '%s', got " +
					"'%s'"), frontend.FormatPos(e.Args[i].Pos()), resolved, sig.ParamNames[i], label)
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
				if _, _, globalOK, err := resolveFunctionTypedGlobalFieldAccess(
					fieldAccess,
					globals,
					funcs,
				); err != nil {
					return "", regionNone, err
				} else if globalOK {
					globalCallbackArg = true
				}
			}
			callbackType, callbackSymbol, err := resolveCallbackArgumentType(
				arg,
				resolved,
				sig,
				i,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
				hasCallerSig && hasStrictSemanticCallClauses(callerSig),
			)
			if err != nil {
				return "", regionNone, err
			}
			if hasCallerSig {
				if callbackSymbol == "" {
					if hasStrictSemanticCallClauses(callerSig) {
						return "", regionNone, unsupportedCallbackUnknownSemanticTargetError(
							arg.Pos(),
							resolved,
							firstStrictSemanticCallClause(callerSig),
						)
					}
				} else if callbackSig, ok := funcs[callbackSymbol]; ok {
					callbackPhrase := fmt.Sprintf("call to '%s'", callbackSymbol)
					if localCallbackArg || globalCallbackArg || fieldCallbackArg {
						if name := callbackArgumentName(arg); name != "" {
							callbackPhrase = fmt.Sprintf("callback argument '%s'", name)
						}
					}
					if err := validateCallAgainstSemanticClauseTarget(
						callerSig,
						callbackSig,
						callbackPhrase,
						arg.Pos(),
					); err != nil {
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
				if id, ok := arg.(*frontend.IdentExpr); ok && !localCallbackArg &&
					!globalCallbackArg &&
					!fieldCallbackArg {
					// Keep lowered target collection deterministic across modules/import aliases.
					id.Name = callbackSymbol
				}
			}
			argType = callbackType
		} else {
			var err error
			argType, argRegion, err = checkExprWithEffects(
				arg,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
			)
			if err != nil {
				return "", regionNone, err
			}
		}
		if !typesCompatibleWithNullPtr(sig.ParamTypes[i], argType, arg) {
			return "", regionNone, fmt.Errorf(
				"%s: type mismatch for %s arg %d",
				frontend.FormatPos(arg.Pos()),
				callTargetPhrase,
				i+1,
			)
		}
		if err := checkResourceCallArg(
			resolved,
			sig.ParamTypes[i],
			arg,
			funcs,
			module,
			imports,
			state,
		); err != nil {
			return "", regionNone, err
		}
		paramOwnership := ""
		if i < len(sig.ParamOwnership) {
			paramOwnership = sig.ParamOwnership[i]
		}
		if paramOwnership == "" {
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return "", regionNone, ownershipDiagnosticf(
					arg.Pos(),
					"borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s",
					borrowedName,
					i+1,
					ownershipTargetPhrase,
				)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return "", regionNone, ownershipDiagnosticf(
						arg.Pos(),
						"borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s",
						borrowedName,
						i+1,
						ownershipTargetPhrase,
					)
				}
			}
			if argType != "ptr" &&
				(typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(
					arg,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					func(borrowedName string) error {
						return ownershipDiagnosticf(arg.Pos(), (("borrowed value derived from '%s' cannot be " +
							"passed to ") +
							"non-borrow parameter %d of %s"), borrowedName, i+1, ownershipTargetPhrase)
					},
				); err != nil {
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
				return "", regionNone, ownershipDiagnosticf(
					arg.Pos(),
					"borrowed value derived from '%s' cannot be consumed by %s",
					borrowedName,
					ownershipTargetPhrase,
				)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return "", regionNone, ownershipDiagnosticf(
						arg.Pos(),
						"borrowed value derived from '%s' cannot be consumed by %s",
						borrowedName,
						ownershipTargetPhrase,
					)
				}
			}
			if argType != "ptr" &&
				(typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(
					arg,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					func(borrowedName string) error {
						return ownershipDiagnosticf(
							arg.Pos(),
							"borrowed value derived from '%s' cannot be consumed by %s",
							borrowedName,
							ownershipTargetPhrase,
						)
					},
				); err != nil {
					return "", regionNone, err
				}
			}
			path := name
			if first, exists := findOwnershipAlias(inoutArgs, path); exists {
				return "", regionNone, ownershipDiagnosticf(
					arg.Pos(),
					"consumed argument '%s' aliases inout argument in %s (inout at %s)",
					path,
					ownershipCallPhrase,
					frontend.FormatPos(first.pos),
				)
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
					return "", regionNone, ownershipDiagnosticf(
						arg.Pos(),
						"borrowed argument '%s' aliases inout argument in %s (inout at %s)",
						path,
						ownershipCallPhrase,
						frontend.FormatPos(first.pos),
					)
				}
				borrowArgs = append(borrowArgs, ownershipArgRef{path: path, pos: arg.Pos()})
			}
		}
		if paramOwnership == "inout" {
			path, ok := canonicalOwnershipAccessPath(arg)
			if !ok {
				return "", regionNone, ownershipDiagnosticf(
					arg.Pos(),
					"inout argument for %s must be a mutable local value",
					ownershipTargetPhrase,
				)
			}
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return "", regionNone, ownershipDiagnosticf(
					arg.Pos(),
					"borrowed value derived from '%s' cannot be passed as inout to %s",
					borrowedName,
					ownershipTargetPhrase,
				)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return "", regionNone, ownershipDiagnosticf(
						arg.Pos(),
						"borrowed value derived from '%s' cannot be passed as inout to %s",
						borrowedName,
						ownershipTargetPhrase,
					)
				}
			}
			if argType != "ptr" &&
				(typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(
					arg,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					func(borrowedName string) error {
						return ownershipDiagnosticf(arg.Pos(), (("borrowed value derived from '%s' cannot be " +
							"passed as inout ") +
							"to %s"), borrowedName, ownershipTargetPhrase)
					},
				); err != nil {
					return "", regionNone, err
				}
			}
			if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
				return "", regionNone, err
			}
			targetInfo, _, err := resolveAssignTarget(arg, locals, globals, types)
			if err != nil || !targetInfo.Mutable || targetInfo.Global {
				return "", regionNone, ownershipDiagnosticf(
					arg.Pos(),
					"inout argument '%s' for %s must be mutable",
					path,
					ownershipTargetPhrase,
				)
			}
			if first, exists := findOwnershipAlias(inoutArgs, path); exists {
				return "", regionNone, ownershipDiagnosticf(
					arg.Pos(),
					"inout argument '%s' used more than once in %s (first at %s)",
					path,
					ownershipCallPhrase,
					frontend.FormatPos(first.pos),
				)
			}
			if first, exists := findOwnershipAlias(borrowArgs, path); exists {
				return "", regionNone, ownershipDiagnosticf(
					arg.Pos(),
					"inout argument '%s' aliases borrowed argument in %s (borrow at %s)",
					path,
					ownershipCallPhrase,
					frontend.FormatPos(first.pos),
				)
			}
			if first, exists := findOwnershipAlias(consumeArgRefs, path); exists {
				return "", regionNone, ownershipDiagnosticf(
					arg.Pos(),
					"inout argument '%s' aliases consumed argument in %s (consume at %s)",
					path,
					ownershipCallPhrase,
					frontend.FormatPos(first.pos),
				)
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
				return "", regionNone, ownershipDiagnosticf(
					e.Args[i].Pos(),
					"value '%s' consumed more than once in %s",
					name,
					ownershipCallPhrase,
				)
			}
			if resourceValuesAlias(
				consumeArgs[j],
				consumeArgTypes[j],
				name,
				consumeArgTypes[i],
				types,
				state,
			) {
				return "", regionNone, ownershipDiagnosticf(
					e.Args[i].Pos(),
					"value '%s' consumed more than once in %s",
					name,
					ownershipCallPhrase,
				)
			}
		}
		if resolved == "core.island_reset" {
			if slicePath, live := state.liveOwnedRegionSliceForOwner(name); live {
				return "", regionNone, ownershipDiagnosticf(
					e.Args[i].Pos(),
					"cannot reset island '%s' while borrowed slice '%s' is alive",
					name,
					slicePath,
				)
			}
		}
		markConsumedResourceValue(name, consumeArgTypes[i], types, state, e.Args[i].Pos())
		if resolved == "core.island_reset" {
			state.markOwnedRegionSlicesConsumedByOwner(name, e.Args[i].Pos())
		}
	}
	if handleArg, ok := surfaceHostABIHandleArgIndex(resolved); ok && handleArg < len(e.Args) {
		if owner, ok := surfaceHandleOwnerPathExprWithAnalysis(
			e.Args[handleArg],
			locals,
			globals,
			types,
			analysis,
		); ok {
			if err := state.checkNotConsumed(owner, e.Args[handleArg].Pos()); err != nil {
				return "", regionNone, err
			}
		}
	}
	if resolved == "core.surface_close" && len(e.Args) > 0 {
		if owner, ok := surfaceHandleOwnerPathExprWithAnalysis(
			e.Args[0],
			locals,
			globals,
			types,
			analysis,
		); ok {
			markConsumedResourceValue(owner, surfaceSurfaceTypeName, types, state, e.Args[0].Pos())
		}
	}
	if resolved == "core.surface_present_rgba" && len(e.Args) > 1 {
		if frameName, ok := surfaceFramePixelsSourceExpr(
			e.Args[1],
			locals,
			globals,
			types,
			analysis,
		); ok &&
			frameName != "" {
			if err := checkSurfacePresentFrameOwnerPath(
				frameName,
				analysis,
				state,
				e.Args[1].Pos(),
			); err != nil {
				return "", regionNone, err
			}
			analysis.markSurfaceFramePresented(frameName, e.Args[1].Pos())
			markConsumedResourceValue(
				frameName,
				surfaceFrameTypeName,
				types,
				state,
				e.Args[1].Pos(),
			)
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
		if err := recordTryCallThrowResourceSummary(
			e,
			sig,
			funcs,
			types,
			module,
			imports,
			state,
		); err != nil {
			return "", regionNone, err
		}
	}
	if resolved == "core.spawn" {
		if len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf(
				"%s: spawn expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: spawn expects a string literal",
				frontend.FormatPos(e.At),
			)
		}
		raw := string(lit.Value)
		if raw == "" {
			return "", regionNone, fmt.Errorf(
				"%s: spawn expects a non-empty name",
				frontend.FormatPos(e.At),
			)
		}
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf(
				"%s: spawn target must be a user function, got '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: unknown function '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			return "", regionNone, fmt.Errorf(
				"%s: spawn target must have shape fun %s(): i32",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf(
				"%s: spawn target must be synchronous",
				frontend.FormatPos(e.At),
			)
		}
		if targetSig.ThrowsType != "" {
			return "", regionNone, fmt.Errorf(
				"%s: spawn target must not throw",
				frontend.FormatPos(e.At),
			)
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf(
				"%s: spawn target '%s' touches mutable global state and cannot cross actor boundary",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf(
				"%s: spawn target '%s' uses effect '%s' and cannot cross actor boundary",
				frontend.FormatPos(e.At),
				target,
				blocked,
			)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf(
				"%s: spawn target '%s' is not sendable across actor boundary",
				frontend.FormatPos(e.At),
				target,
			)
		}
		lit.Value = []byte(target)
	}
	if resolved == "core.spawn_remote" {
		if len(e.Args) != 2 {
			return "", regionNone, fmt.Errorf(
				"%s: spawn_remote expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		lit, ok := e.Args[1].(*frontend.StringLitExpr)
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: spawn_remote expects a string literal",
				frontend.FormatPos(e.At),
			)
		}
		raw := string(lit.Value)
		if raw == "" {
			return "", regionNone, fmt.Errorf(
				"%s: spawn_remote expects a non-empty name",
				frontend.FormatPos(e.At),
			)
		}
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf(
				"%s: spawn_remote target must be a user function, got '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: unknown function '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			return "", regionNone, fmt.Errorf(
				"%s: spawn_remote target must have shape fun %s(): i32",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf(
				"%s: spawn_remote target must be synchronous",
				frontend.FormatPos(e.At),
			)
		}
		if targetSig.ThrowsType != "" {
			return "", regionNone, fmt.Errorf(
				"%s: spawn_remote target must not throw",
				frontend.FormatPos(e.At),
			)
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf(
				"%s: spawn_remote target '%s' touches mutable global state and cannot cross actor boundary",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf(
				"%s: spawn_remote target '%s' uses effect '%s' and cannot cross actor boundary",
				frontend.FormatPos(e.At),
				target,
				blocked,
			)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf(
				"%s: spawn_remote target '%s' is not sendable across actor boundary",
				frontend.FormatPos(e.At),
				target,
			)
		}
		lit.Value = []byte(target)
	}
	if resolved == "core.task_spawn_i32" {
		if len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32 expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32 expects a string literal",
				frontend.FormatPos(e.At),
			)
		}
		raw := string(lit.Value)
		if raw == "" {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32 expects a non-empty name",
				frontend.FormatPos(e.At),
			)
		}
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32 target must be a user function, got '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: unknown function '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32 target must have shape func %s() -> i32",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32 target must be synchronous",
				frontend.FormatPos(e.At),
			)
		}
		if targetSig.ThrowsType != "" {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32 target must not throw",
				frontend.FormatPos(e.At),
			)
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32 target '%s' touches mutable global state and cannot cross task boundary",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32 target '%s' uses effect '%s' and cannot cross task boundary",
				frontend.FormatPos(e.At),
				target,
				blocked,
			)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_i32 target '%s' is not sendable across task boundary",
				frontend.FormatPos(e.At),
				target,
			)
		}
		lit.Value = []byte(target)
	}
	if resolved == "core.task_spawn_group_i32" {
		if len(e.Args) != 2 {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_group_i32 expects 2 arguments",
				frontend.FormatPos(e.At),
			)
		}
		lit, ok := e.Args[1].(*frontend.StringLitExpr)
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_group_i32 expects a string literal worker name",
				frontend.FormatPos(e.At),
			)
		}
		raw := string(lit.Value)
		if raw == "" {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_group_i32 expects a non-empty name",
				frontend.FormatPos(e.At),
			)
		}
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_group_i32 target must be a user function, got '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: unknown function '%s'",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_group_i32 target must have shape func %s() -> i32",
				frontend.FormatPos(e.At),
				target,
			)
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_group_i32 target must be synchronous",
				frontend.FormatPos(e.At),
			)
		}
		if targetSig.ThrowsType != "" {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_group_i32 target must not throw",
				frontend.FormatPos(e.At),
			)
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf(
				("%s: task_spawn_group_i32 target '%s' touches mutable global " +
					"state and cannot cross task boundary"),
				frontend.FormatPos(e.At),
				target,
			)
		}
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_group_i32 target '%s' uses effect '%s' and cannot cross task boundary",
				frontend.FormatPos(e.At),
				target,
				blocked,
			)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf(
				"%s: task_spawn_group_i32 target '%s' is not sendable across task boundary",
				frontend.FormatPos(e.At),
				target,
			)
		}
		lit.Value = []byte(target)
	}
	if resolved == "core.sym_addr" {
		if len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf(
				"%s: sym_addr expects 1 argument",
				frontend.FormatPos(e.At),
			)
		}
		lit, ok := e.Args[0].(*frontend.StringLitExpr)
		if !ok {
			return "", regionNone, fmt.Errorf(
				"%s: sym_addr expects a string literal",
				frontend.FormatPos(e.At),
			)
		}
		if len(lit.Value) == 0 {
			return "", regionNone, fmt.Errorf(
				"%s: sym_addr expects a non-empty symbol name",
				frontend.FormatPos(e.At),
			)
		}
	}
	if (resolved == "core.island_make_u8" ||
		resolved == "core.island_make_u16" ||
		resolved == "core.island_make_i32" ||
		resolved == "core.island_make_bool") &&
		len(argRegions) > 0 &&
		argRegions[0] == regionUnknown {
		return "", regionNone, fmt.Errorf(
			"%s: ambiguous region for '%s' argument",
			frontend.FormatPos(e.At),
			resolved,
		)
	}
	if err := effects.requireAll(e.At, sig.Effects); err != nil {
		return "", regionNone, err
	}
	if builtinNeedsUnsafe(resolved, argRegions) && !state.inUnsafe() {
		return "", regionNone, effectDiagnosticf(
			e.At,
			"'%s' is only allowed in unsafe blocks",
			resolved,
		)
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
				return "", regionNone, fmt.Errorf(
					"%s: invalid region signature for '%s'",
					frontend.FormatPos(e.At),
					resolved,
				)
			}
			leafRegion := argRegions[paramIndex]
			if leafRegion == regionUnknown {
				return "", regionNone, fmt.Errorf(
					"%s: ambiguous region for '%s' return",
					frontend.FormatPos(e.At),
					resolved,
				)
			}
			if leafRegion != regionNone {
				tree[leaf] = leafRegion
			}
		}
		state.setExprRegionTree(e, tree)
		regionID = constructorRegionFromTree(tree)
	} else if sig.ReturnRegionParam >= 0 {
		if sig.ReturnRegionParam >= len(argRegions) {
			return "", regionNone, fmt.Errorf(
				"%s: invalid region signature for '%s'",
				frontend.FormatPos(e.At),
				resolved,
			)
		}
		regionID = argRegions[sig.ReturnRegionParam]
		if regionID == regionUnknown {
			return "", regionNone, fmt.Errorf(
				"%s: ambiguous region for '%s' return",
				frontend.FormatPos(e.At),
				resolved,
			)
		}
	}
	return sig.ReturnType, regionID, nil
}

func functionTypedBorrowReturnRegion(
	returnOwnership string,
	paramOwnership []string,
	argRegions []int,
	state *regionState,
) int {
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
	return strings.HasPrefix(name, "core.slice_copy_") &&
		!strings.HasPrefix(name, "core.slice_copy_into_")
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
		return fmt.Errorf(
			"%s: internal error: call argument labels are inconsistent",
			frontend.FormatPos(e.At),
		)
	}
	for i, label := range e.ArgLabels {
		if label == "" {
			return fmt.Errorf(
				"%s: cannot mix labeled and unlabeled arguments in %s '%s'",
				frontend.FormatPos(e.Args[i].Pos()),
				kind,
				name,
			)
		}
	}
	return nil
}

// ---- exprs_ownership.go ----

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

func consumeLocalArgumentName(
	expr frontend.Expr,
	callee string,
	callback bool,
	phraseOverride ...string,
) (string, error) {
	targetPhrase := fmt.Sprintf("'%s'", callee)
	if callback {
		targetPhrase = fmt.Sprintf("callback '%s'", callee)
	}
	if len(phraseOverride) > 0 && phraseOverride[0] != "" {
		targetPhrase = phraseOverride[0]
	}
	path, ok := canonicalOwnershipAccessPath(expr)
	if !ok {
		return "", ownershipDiagnosticf(
			expr.Pos(),
			"consume argument for %s must be a local value",
			targetPhrase,
		)
	}
	return path, nil
}

func checkWholeOwnershipValueAvailable(
	expr frontend.Expr,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	return checkWholeOwnershipValueAvailableForType(expr, "", types, module, imports, state)
}

func checkWholeOwnershipValueAvailableForType(
	expr frontend.Expr,
	expectedType string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
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
			if err := checkWholeOwnershipValueAvailableForType(
				field.Value,
				fieldInfo.TypeName,
				types,
				module,
				imports,
				state,
			); err != nil {
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
				if err := checkWholeOwnershipValueAvailableForType(
					e.Args[i],
					field.TypeName,
					types,
					module,
					imports,
					state,
				); err != nil {
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
				if err := checkWholeOwnershipValueAvailableForType(
					arg,
					caseInfo.PayloadTypes[i],
					types,
					module,
					imports,
					state,
				); err != nil {
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

func checkFunctionTypedCallArguments(
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
	paramTypes []string,
	paramOwnerships []string,
	valueCallPhrase string,
) ([]int, error) {
	consumeArgs := make([]string, len(e.Args))
	consumeArgTypes := make([]string, len(e.Args))
	consumeArgRefs := make([]ownershipArgRef, 0, len(e.Args))
	borrowArgs := make([]ownershipArgRef, 0, len(e.Args))
	inoutArgs := make([]ownershipArgRef, 0, len(e.Args))
	argRegions := make([]int, len(e.Args))
	for i, arg := range e.Args {
		argType, argRegion, err := checkExprWithEffects(
			arg,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return nil, err
		}
		argRegions[i] = argRegion
		if !typesCompatibleWithNullPtr(paramTypes[i], argType, arg) {
			return nil, fmt.Errorf(
				"%s: type mismatch for %s arg %d",
				frontend.FormatPos(arg.Pos()),
				valueCallPhrase,
				i+1,
			)
		}
		paramOwnership := ownershipAt(paramOwnerships, i)
		if paramOwnership == "" {
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return nil, ownershipDiagnosticf(
					arg.Pos(),
					"borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s",
					borrowedName,
					i+1,
					valueCallPhrase,
				)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return nil, ownershipDiagnosticf(
						arg.Pos(),
						"borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s",
						borrowedName,
						i+1,
						valueCallPhrase,
					)
				}
			}
			if argType != "ptr" &&
				(typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(
					arg,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					func(borrowedName string) error {
						return ownershipDiagnosticf(arg.Pos(), (("borrowed value derived from '%s' cannot be " +
							"passed to ") +
							"non-borrow parameter %d of %s"), borrowedName, i+1, valueCallPhrase)
					},
				); err != nil {
					return nil, err
				}
			}
			if path, ok := canonicalOwnershipAccessPath(arg); ok {
				if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
					return nil, err
				}
			}
		}
		if paramOwnership == "consume" {
			name, err := consumeLocalArgumentName(arg, e.Name, true)
			if err != nil {
				return nil, err
			}
			if err := state.checkNoConsumedDescendants(name, arg.Pos()); err != nil {
				return nil, err
			}
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return nil, ownershipDiagnosticf(
					arg.Pos(),
					"borrowed value derived from '%s' cannot be consumed by %s",
					borrowedName,
					valueCallPhrase,
				)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return nil, ownershipDiagnosticf(
						arg.Pos(),
						"borrowed value derived from '%s' cannot be consumed by %s",
						borrowedName,
						valueCallPhrase,
					)
				}
			}
			if argType != "ptr" &&
				(typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(
					arg,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					func(borrowedName string) error {
						return ownershipDiagnosticf(
							arg.Pos(),
							"borrowed value derived from '%s' cannot be consumed by %s",
							borrowedName,
							valueCallPhrase,
						)
					},
				); err != nil {
					return nil, err
				}
			}
			path := name
			if first, exists := findOwnershipAlias(inoutArgs, path); exists {
				return nil, ownershipDiagnosticf(
					arg.Pos(),
					"consumed argument '%s' aliases inout argument in %s (inout at %s)",
					path,
					valueCallPhrase,
					frontend.FormatPos(first.pos),
				)
			}
			consumeArgs[i] = name
			consumeArgTypes[i] = argType
			consumeArgRefs = append(consumeArgRefs, ownershipArgRef{path: path, pos: arg.Pos()})
		}
		if paramOwnership == "borrow" {
			path, ok := canonicalOwnershipAccessPath(arg)
			if ok {
				if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
					return nil, err
				}
				if first, exists := findOwnershipAlias(inoutArgs, path); exists {
					return nil, ownershipDiagnosticf(
						arg.Pos(),
						"borrowed argument '%s' aliases inout argument in %s (inout at %s)",
						path,
						valueCallPhrase,
						frontend.FormatPos(first.pos),
					)
				}
				borrowArgs = append(borrowArgs, ownershipArgRef{path: path, pos: arg.Pos()})
			}
		}
		if paramOwnership == "inout" {
			path, ok := canonicalOwnershipAccessPath(arg)
			if !ok {
				return nil, ownershipDiagnosticf(
					arg.Pos(),
					"inout argument for %s must be a mutable local value",
					valueCallPhrase,
				)
			}
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return nil, ownershipDiagnosticf(
					arg.Pos(),
					"borrowed value derived from '%s' cannot be passed as inout to %s",
					borrowedName,
					valueCallPhrase,
				)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return nil, ownershipDiagnosticf(
						arg.Pos(),
						"borrowed value derived from '%s' cannot be passed as inout to %s",
						borrowedName,
						valueCallPhrase,
					)
				}
			}
			if argType != "ptr" &&
				(typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(
					arg,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					func(borrowedName string) error {
						return ownershipDiagnosticf(arg.Pos(), (("borrowed value derived from '%s' cannot be " +
							"passed as inout ") +
							"to %s"), borrowedName, valueCallPhrase)
					},
				); err != nil {
					return nil, err
				}
			}
			if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
				return nil, err
			}
			targetInfo, _, err := resolveAssignTarget(arg, locals, globals, types)
			if err != nil || !targetInfo.Mutable || targetInfo.Global {
				return nil, ownershipDiagnosticf(
					arg.Pos(),
					"inout argument '%s' for %s must be mutable",
					path,
					valueCallPhrase,
				)
			}
			if first, exists := findOwnershipAlias(inoutArgs, path); exists {
				return nil, ownershipDiagnosticf(
					arg.Pos(),
					"inout argument '%s' used more than once in %s (first at %s)",
					path,
					valueCallPhrase,
					frontend.FormatPos(first.pos),
				)
			}
			if first, exists := findOwnershipAlias(borrowArgs, path); exists {
				return nil, ownershipDiagnosticf(
					arg.Pos(),
					"inout argument '%s' aliases borrowed argument in %s (borrow at %s)",
					path,
					valueCallPhrase,
					frontend.FormatPos(first.pos),
				)
			}
			if first, exists := findOwnershipAlias(consumeArgRefs, path); exists {
				return nil, ownershipDiagnosticf(
					arg.Pos(),
					"inout argument '%s' aliases consumed argument in %s (consume at %s)",
					path,
					valueCallPhrase,
					frontend.FormatPos(first.pos),
				)
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
				return nil, ownershipDiagnosticf(
					e.Args[i].Pos(),
					"value '%s' consumed more than once in %s",
					name,
					valueCallPhrase,
				)
			}
			if resourceValuesAlias(
				consumeArgs[j],
				consumeArgTypes[j],
				name,
				consumeArgTypes[i],
				types,
				state,
			) {
				return nil, ownershipDiagnosticf(
					e.Args[i].Pos(),
					"value '%s' consumed more than once in %s",
					name,
					valueCallPhrase,
				)
			}
		}
		markConsumedResourceValue(name, consumeArgTypes[i], types, state, e.Args[i].Pos())
	}
	return argRegions, nil
}

// checkCallExprWithEffects intentionally keeps call validation in one ordered
// path: resolve local/builtin/imported targets, enforce semantic clauses and
// effects, validate async/throw context, then check arguments, ownership, and
// resource provenance before returning type and region metadata.

// ---- exprs_resources_actors.go ----

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
		return ownershipDiagnosticf(
			arg.Pos(),
			"ambiguous resource provenance for '%s' after control-flow merge",
			name,
		)
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
		if source, err := resourceSourceForExpr(
			call.Args[0],
			funcs,
			module,
			imports,
			state,
		); err == nil &&
			source.known {
			state.markResourceFinalizedAliases(source.name, "closed", call.Args[0].Pos())
		}
	}
}

func markTaskHandleJoined(
	arg frontend.Expr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	state *regionState,
) {
	if state == nil {
		return
	}
	if source, err := resourceSourceForExpr(arg, funcs, module, imports, state); err == nil &&
		source.known {
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

func borrowedOwnerFromExpr(
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
) (string, bool, error) {
	if explicitCopyResultExpr(expr) {
		return "", false, nil
	}
	tname, regionID, err := checkExprWithEffects(
		expr,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		state,
		effects,
		analysis,
	)
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

func checkBorrowedEscape(
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
	format func(string) error,
) error {
	if expr == nil {
		return nil
	}
	if handled, err := checkBorrowedEscapeAggregateConstructor(
		expr,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		state,
		effects,
		analysis,
		format,
	); handled ||
		err != nil {
		return err
	}
	if borrowedName, borrowed, err := borrowedOwnerFromExpr(
		expr,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		state,
		effects,
		analysis,
	); err != nil {
		return err
	} else if borrowed {
		return format(borrowedName)
	}
	if lit, ok := expr.(*frontend.StructLitExpr); ok {
		for _, field := range lit.Fields {
			if err := checkBorrowedEscape(
				field.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
				format,
			); err != nil {
				return err
			}
		}
	}
	if call, ok := expr.(*frontend.CallExpr); ok {
		if _, caseInfo, found, err := resolveEnumCaseConstructorCall(
			call,
			types,
			module,
			imports,
		); err != nil {
			return err
		} else if found {
			for i, arg := range call.Args {
				if i >= len(caseInfo.PayloadTypes) {
					break
				}
				if err := checkBorrowedEscape(
					arg,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					format,
				); err != nil {
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
						if err := checkBorrowedEscape(
							arg,
							locals,
							globals,
							funcs,
							types,
							module,
							imports,
							state,
							effects,
							analysis,
							format,
						); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	if fieldAccess, ok := expr.(*frontend.FieldAccessExpr); ok {
		if _, _, enumCase, err := resolveEnumCaseExpr(
			fieldAccess,
			locals,
			globals,
			types,
			module,
			imports,
		); err != nil {
			return err
		} else if enumCase {
			return nil
		}
		if err := checkBorrowedEscape(
			fieldAccess.Base,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
			format,
		); err != nil {
			return err
		}
	}
	if idx, ok := expr.(*frontend.IndexExpr); ok {
		if err := checkBorrowedEscape(
			idx.Base,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
			format,
		); err != nil {
			return err
		}
	}
	if match, ok := expr.(*frontend.MatchExpr); ok {
		for _, c := range match.Cases {
			if err := checkBorrowedEscape(
				c.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
				format,
			); err != nil {
				return err
			}
		}
	}
	if catch, ok := expr.(*frontend.CatchExpr); ok {
		caseScopes := state.catchExprScopes[catch]
		for i, c := range catch.Cases {
			caseScopeID := regionNone
			if i < len(caseScopes) {
				caseScopeID = caseScopes[i]
			}
			if caseScopeID == regionNone {
				caseScopeID = patternBindingScopeID(c.Pattern, state)
			}
			if err := withActiveScope(state, caseScopeID, func() error {
				return checkBorrowedEscape(
					c.Value,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					format,
				)
			}); err != nil {
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
		return checkBorrowedEscape(
			unary.X,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
			format,
		)
	}
	if binary, ok := expr.(*frontend.BinaryExpr); ok {
		if err := checkBorrowedEscape(
			binary.Left,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
			format,
		); err != nil {
			return err
		}
		return checkBorrowedEscape(
			binary.Right,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
			format,
		)
	}
	return nil
}

func checkBorrowedEscapeAggregateConstructor(
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
	format func(string) error,
) (bool, error) {
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
			if err := checkBorrowedEscape(
				field.Value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				state,
				effects,
				analysis,
				format,
			); err != nil {
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
				if err := checkBorrowedEscape(
					e.Args[i],
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					format,
				); err != nil {
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
				if i >= len(
					caseInfo.PayloadTypes,
				) || !borrowedEscapeShouldInspect(
					caseInfo.PayloadTypes[i],
					types,
				) {
					continue
				}
				if err := checkBorrowedEscape(
					arg,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					state,
					effects,
					analysis,
					format,
				); err != nil {
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

func checkBorrowedInoutEscape(
	expr frontend.Expr,
	targetName string,
	pos frontend.Position,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) error {
	return checkBorrowedEscape(
		expr,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		state,
		effects,
		analysis,
		func(borrowedName string) error {
			return lifetimeDiagnosticf(
				pos,
				"borrowed local '%s' cannot escape via inout assignment to '%s'",
				borrowedName,
				targetName,
			)
		},
	)
}

func markConsumedResourceValue(
	name string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
	pos frontend.Position,
) {
	if state == nil || name == "" {
		return
	}
	if !typeContainsResourceHandle(typeName, types) {
		state.markConsumed(name, pos)
		return
	}
	markConsumedResourcePath(name, typeName, types, state, pos)
}

func markConsumedResourcePath(
	prefix string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
	pos frontend.Position,
) {
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

func resourceValuesAlias(
	leftName string,
	leftType string,
	rightName string,
	rightType string,
	types map[string]*TypeInfo,
	state *regionState,
) bool {
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

func resourceIDsForValue(
	name string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
) map[int]bool {
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

func checkResourceTreeUsable(
	name string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
	pos frontend.Position,
) error {
	if state == nil || name == "" || !typeContainsResourceHandle(typeName, types) {
		return nil
	}
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		path := joinResourcePath(name, leaf)
		source := resourceSourceForPath(path, state)
		if source.unknown {
			return ownershipDiagnosticf(
				pos,
				"ambiguous resource provenance for '%s' after control-flow merge",
				path,
			)
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

func validateTypedTaskErrorType(
	typeName string,
	types map[string]*TypeInfo,
	pos frontend.Position,
) error {
	info, ok := types[typeName]
	if !ok {
		return fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), typeName)
	}
	if info.Kind != TypeEnum {
		return fmt.Errorf("%s: typed task error argument must be an enum", frontend.FormatPos(pos))
	}
	if reason := typeActorTaskSendabilityUnsafeReason(
		typeName,
		types,
		map[string]bool{},
	); reason != "" {
		return fmt.Errorf(
			"%s: typed task error payload must be sendable across task boundary: %s",
			frontend.FormatPos(pos),
			reason,
		)
	}
	return nil
}

func taskTypedSpawnName(resolved string) string {
	if resolved == "core.task_spawn_group_i32_typed" {
		return "task_spawn_group_i32_typed"
	}
	return "task_spawn_i32_typed"
}

func validateTypedActorMessageType(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
) error {
	info, ok := types[typeName]
	if !ok {
		return fmt.Errorf("unknown type '%s'", typeName)
	}
	if isRuntimeSystemMessageSurfaceType(typeName) || (info.RuntimeOwned && !info.ActorSendable) {
		return fmt.Errorf(
			"runtime system messages cannot be sent through the ordinary actor mailbox; use actor lifecycle, link, monitor, or cluster APIs",
		)
	}
	if surfaceType, ok := surfaceActorTaskBoundaryValueType(typeName, types); ok {
		return fmt.Errorf("surface value '%s' cannot cross actor/task boundary", surfaceType)
	}
	switch info.Kind {
	case TypeI32, TypeU8, TypeBool, TypeIsland:
		return nil
	case TypeEnum:
		if len(info.EnumCases) > 255 {
			return fmt.Errorf(
				"typed actor message enum supports at most 255 cases, got %d for '%s'",
				len(info.EnumCases),
				typeName,
			)
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
		if info.SlotCount-1 > 8 {
			return fmt.Errorf(
				"typed actor message payload supports at most 8 value slots, got %d for '%s'",
				info.SlotCount-1,
				typeName,
			)
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

func isRuntimeSystemMessageSurfaceType(typeName string) bool {
	switch typeName {
	case "lib.core.actors.SystemMessage", "lib.core.actors.SystemReceiveResult":
		return true
	default:
		return false
	}
}

func validateActorBoundaryPayloadExpr(
	expr frontend.Expr,
	typeName string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	transferOwners map[string]bool,
) error {
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
			if owner := ownedRegionSliceOwnerForExpr(expr, state); owner != "" &&
				transferOwners[owner] {
				return nil
			}
		}
		return ownershipDiagnosticf(
			expr.Pos(),
			("cannot send borrowed view across actor boundary; use .copy()" +
				" (borrowed value derived from '%s' cannot cross actor " +
				"boundary without copy)"),
			borrowOwnerNameFromExpr(expr),
		)
	case TypeStruct:
		for _, field := range structFieldExprs(expr, info) {
			if kind, _, directView := borrowedReturnDirectViewLabels(field.typeName, types); directView &&
				!isExplicitCopyExpr(field.value) {
				return ownershipDiagnosticf(
					expr.Pos(),
					"aggregate '%s' contains borrowed %s field '%s' that cannot cross actor boundary",
					displayTypeName(typeName, module),
					kind,
					field.name,
				)
			}
			if err := validateActorBoundaryPayloadExpr(
				field.value,
				field.typeName,
				types,
				module,
				imports,
				state,
				transferOwners,
			); err != nil {
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
					if err := validateActorBoundaryPayloadExpr(
						arg,
						caseInfo.PayloadTypes[i],
						types,
						module,
						imports,
						state,
						transferOwners,
					); err != nil {
						return err
					}
				}
			}
		}
	case TypeOptional:
		if call, ok := expr.(*frontend.CallExpr); ok {
			for _, arg := range call.Args {
				if err := validateActorBoundaryPayloadExpr(
					arg,
					info.ElemType,
					types,
					module,
					imports,
					state,
					transferOwners,
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func actorTransferOwnerPayloads(
	expr frontend.Expr,
	typeName string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) map[string]bool {
	owners := make(map[string]bool)
	collectActorTransferOwnerPayloads(expr, typeName, types, module, imports, owners)
	return owners
}

func collectActorTransferOwnerPayloads(
	expr frontend.Expr,
	typeName string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	owners map[string]bool,
) {
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
			collectActorTransferOwnerPayloads(
				field.value,
				field.typeName,
				types,
				module,
				imports,
				owners,
			)
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
			collectActorTransferOwnerPayloads(
				arg,
				caseInfo.PayloadTypes[i],
				types,
				module,
				imports,
				owners,
			)
		}
	case TypeOptional:
		if call, ok := expr.(*frontend.CallExpr); ok {
			for _, arg := range call.Args {
				collectActorTransferOwnerPayloads(
					arg,
					info.ElemType,
					types,
					module,
					imports,
					owners,
				)
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
	return fmt.Errorf(
		"typed actor message payload must be value-only, got %s '%s'",
		category,
		typeName,
	)
}

func typedActorTypeContainsIsland(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
) bool {
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
				consumeIslandSourceLocals(
					field.Value,
					fieldInfo.TypeName,
					locals,
					types,
					module,
					imports,
					state,
				)
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
					consumeIslandSourceLocals(
						arg,
						caseInfo.PayloadTypes[i],
						locals,
						types,
						module,
						imports,
						state,
					)
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
			return ownershipDiagnosticf(
				id.At,
				"island transfer payload '%s' must be a local value",
				id.Name,
			)
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
				if err := consumeTypedActorTransferPayloads(
					field.Value,
					fieldInfo.TypeName,
					locals,
					types,
					module,
					imports,
					state,
				); err != nil {
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
			return ownershipDiagnosticf(
				expr.Pos(),
				"island-containing struct transfer payload must be a local value",
			)
		}
		if _, ok := locals[id.Name]; !ok {
			return ownershipDiagnosticf(
				id.At,
				"island-containing transfer payload '%s' must be a local value",
				id.Name,
			)
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
					if err := consumeTypedActorTransferPayloads(
						arg,
						caseInfo.PayloadTypes[i],
						locals,
						types,
						module,
						imports,
						state,
					); err != nil {
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
			return ownershipDiagnosticf(
				expr.Pos(),
				"island-containing enum transfer payload must be a local value",
			)
		}
		if _, ok := locals[id.Name]; !ok {
			return ownershipDiagnosticf(
				id.At,
				"island-containing transfer payload '%s' must be a local value",
				id.Name,
			)
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
	if info.RuntimeOwned && !info.UserConstructible {
		return "", regionNone, true, runtimeOwnedConstructionError(e.At, resolvedType)
	}
	if len(e.Args) != len(info.Fields) {
		return "", regionNone, true, fmt.Errorf(
			"%s: wrong field count for '%s'",
			frontend.FormatPos(e.At),
			resolvedType,
		)
	}

	argByLabel := make(map[string]frontend.Expr, len(e.Args))
	for i, label := range e.ArgLabels {
		if _, exists := argByLabel[label]; exists {
			return "", regionNone, true, fmt.Errorf(
				"%s: duplicate field '%s'",
				frontend.FormatPos(e.Args[i].Pos()),
				label,
			)
		}
		argByLabel[label] = e.Args[i]
	}
	for label, expr := range argByLabel {
		if _, ok := info.FieldMap[label]; !ok {
			return "", regionNone, true, fmt.Errorf(
				"%s: unknown field '%s'",
				frontend.FormatPos(expr.Pos()),
				label,
			)
		}
	}

	orderedArgs := make([]frontend.Expr, 0, len(info.Fields))
	orderedLabels := make([]string, 0, len(info.Fields))
	fieldTree := make(map[string]int)
	for _, field := range info.Fields {
		arg, ok := argByLabel[field.Name]
		if !ok {
			return "", regionNone, true, fmt.Errorf(
				"%s: missing field '%s'",
				frontend.FormatPos(e.At),
				field.Name,
			)
		}
		if field.FunctionTypeValue {
			markMutableFunctionTypedGlobalSource(arg, globals, analysis)
			if _, err := validateFunctionTypeStructFieldBinding(
				resolvedType,
				field,
				arg,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			); err != nil {
				return "", regionNone, true, err
			}
			orderedArgs = append(orderedArgs, arg)
			orderedLabels = append(orderedLabels, field.Name)
			continue
		}
		valType, valRegion, err := checkExprWithEffects(
			arg,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			state,
			effects,
			analysis,
		)
		if err != nil {
			return "", regionNone, true, err
		}
		if !typesCompatibleWithNullPtr(field.TypeName, valType, arg) {
			return "", regionNone, true, fmt.Errorf(
				"%s: type mismatch for field '%s'",
				frontend.FormatPos(arg.Pos()),
				field.Name,
			)
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

func runtimeOwnedConstructionError(pos frontend.Position, typeName string) error {
	return fmt.Errorf(
		"%s: runtime-owned actor handle '%s' cannot be constructed",
		frontend.FormatPos(pos),
		typeName,
	)
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
		return barePayloadRequiredDiagnostic(
			c.At,
			scrutType,
			caseName,
			len(caseInfo.PayloadTypes),
			module,
		)
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
		return barePayloadRequiredDiagnostic(
			c.At,
			e.ErrorType,
			caseName,
			len(caseInfo.PayloadTypes),
			module,
		)
	}
	return nil
}

func barePayloadRequiredDiagnostic(
	pos frontend.Position,
	typeName string,
	caseName string,
	arity int,
	module string,
) error {
	if arity <= 0 {
		arity = 1
	}
	return fmt.Errorf(
		"%s: enum case '%s.%s' carries %d payload value(s); use '%s.%s(%s)'",
		frontend.FormatPos(pos),
		displayTypeName(typeName, module),
		caseName,
		arity,
		displayTypeName(typeName, module),
		caseName,
		placeholderBindingList(arity),
	)
}

func bareEnumPatternCaseName(
	pattern frontend.Expr,
	expectedType string,
	module string,
	imports map[string]string,
) (string, bool) {
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

func bareEnumPatternTypeAndCase(
	pattern frontend.Expr,
	module string,
	imports map[string]string,
) (string, string, bool) {
	field, ok := pattern.(*frontend.FieldAccessExpr)
	if !ok {
		return "", "", false
	}
	return resolveBareEnumPatternParts(field, module, imports)
}

func resolveBareEnumPatternParts(
	field *frontend.FieldAccessExpr,
	module string,
	imports map[string]string,
) (string, string, bool) {
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

// ---- function_types.go ----

func validateFunctionTypeLiteralBinding(
	name string,
	declared frontend.TypeRef,
	closure *frontend.ClosureExpr,
	locals map[string]LocalInfo,
	module string,
	imports map[string]string,
) error {
	if declared.Kind != frontend.TypeRefFunction {
		return nil
	}
	if closure == nil || closure.Decl == nil {
		return fmt.Errorf(
			("%s: function-typed local '%s' must be initialized with a " +
				"non-capturing closure literal in this MVP"),
			frontend.FormatPos(declared.At),
			name,
		)
	}
	if len(closure.Decl.TypeParams) > 0 {
		return fmt.Errorf(
			"%s: generic closure literals are not supported for function-typed local '%s' in this MVP",
			frontend.FormatPos(closure.At),
			name,
		)
	}
	declaredEffects, err := functionTypeRefEffects(declared, declared.At)
	if err != nil {
		return err
	}
	closureEffects, err := normalizeEffects(closure.Decl.Uses, closure.Decl.Pos)
	if err != nil {
		return err
	}
	if err := validateFunctionTypeCallableEffects(
		declaredEffects,
		closureEffects,
		closure.At,
		"function-typed local",
		name,
	); err != nil {
		return err
	}
	explicitParams := explicitClosureParams(closure)
	if len(explicitParams) != len(declared.Params) {
		return fmt.Errorf(
			"%s: function-typed local '%s' parameter count mismatch: expected %d, got %d",
			frontend.FormatPos(closure.At),
			name,
			len(declared.Params),
			len(explicitParams),
		)
	}
	if err := validateFunctionTypeParamOwnership(
		functionTypeRefParamOwnership(declared),
		paramDeclOwnership(explicitParams),
		len(declared.Params),
		closure.At,
		"function-typed local",
		name,
	); err != nil {
		return err
	}
	for i := range declared.Params {
		want, err := resolveTypeName(&declared.Params[i], module, imports)
		if err != nil {
			return err
		}
		got, err := resolveTypeName(&explicitParams[i].Type, module, imports)
		if err != nil {
			return err
		}
		if want != got {
			return fmt.Errorf(
				"%s: function-typed local '%s' parameter %d type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(closure.Decl.Params[i].At),
				name,
				i+1,
				want,
				got,
			)
		}
	}
	if declared.Return == nil {
		return fmt.Errorf("%s: missing function return type", frontend.FormatPos(declared.At))
	}
	wantRet, err := resolveTypeName(declared.Return, module, imports)
	if err != nil {
		return err
	}
	gotRet, err := resolveTypeName(&closure.Decl.ReturnType, module, imports)
	if err != nil {
		return err
	}
	if wantRet != gotRet {
		return fmt.Errorf(
			"%s: function-typed local '%s' return type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(closure.At),
			name,
			wantRet,
			gotRet,
		)
	}
	if declared.ReturnOwnership != closure.Decl.ReturnOwnership {
		return fmt.Errorf(
			"%s: function-typed local '%s' return ownership mismatch: expected '%s', got '%s'",
			frontend.FormatPos(closure.At),
			name,
			ownershipDisplay(declared.ReturnOwnership),
			ownershipDisplay(closure.Decl.ReturnOwnership),
		)
	}
	wantThrows, err := functionTypeRefThrowsType(declared, module, imports)
	if err != nil {
		return err
	}
	gotThrows := ""
	if closure.Decl.HasThrows {
		gotThrows, err = resolveTypeName(&closure.Decl.Throws, module, imports)
		if err != nil {
			return err
		}
	}
	if wantThrows != gotThrows {
		return fmt.Errorf(
			"%s: function-typed local '%s' throws type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(closure.At),
			name,
			wantThrows,
			gotThrows,
		)
	}
	return nil
}

func explicitClosureParams(closure *frontend.ClosureExpr) []frontend.ParamDecl {
	if closure == nil || closure.Decl == nil {
		return nil
	}
	params := closure.Decl.Params
	if captureCount := len(closure.Captures); captureCount > 0 && captureCount <= len(params) {
		return params[:len(params)-captureCount]
	}
	return params
}

func validateFunctionTypeStructFieldBinding(
	localName string,
	field FieldInfo,
	init frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (FunctionFieldInfo, error) {
	if !field.FunctionTypeValue {
		return FunctionFieldInfo{}, nil
	}
	resolved := ""
	paramName := ""
	captures := []frontend.ClosureCapture(nil)
	escapeCaptures := []frontend.ClosureCapture(nil)
	directSnapshotAlias := false
	escapeKind := CallableEscapeKind("")
	handleValue := false
	switch value := init.(type) {
	case *frontend.IdentExpr:
		source, localIdent := locals[value.Name]
		var err error
		resolved, err = validateFunctionTypeNamedSymbolBinding(
			localName+"."+field.Name,
			field.FunctionTypeRef,
			value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			true,
			unsupportedGenericFunctionTypedStructFieldInitializerError,
		)
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		captures = append([]frontend.ClosureCapture(nil), source.FunctionCaptures...)
		escapeCaptures = append([]frontend.ClosureCapture(nil), source.FunctionEscapeCaptures...)
		directSnapshotAlias = source.FunctionDirectSnapshotAlias
		escapeKind = source.FunctionEscapeKind
		handleValue = source.FunctionHandleValue
		if source.FunctionTypeValue && source.FunctionValue == "" {
			paramName = source.FunctionParamName
			if paramName == "" {
				paramName = value.Name
			}
		}
		if resolved != "" && !localIdent {
			value.Name = resolved
		}
	case *frontend.FieldAccessExpr:
		fieldInfo, ok, err := resolveFunctionFieldArgument(value, locals)
		target := ""
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		if ok && fieldInfo.FunctionValue != "" {
			target = fieldInfo.FunctionValue
			captures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...)
			escapeCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionEscapeCaptures...)
			directSnapshotAlias = fieldInfo.FunctionDirectSnapshotAlias
			escapeKind = fieldInfo.FunctionEscapeKind
			handleValue = fieldInfo.FunctionHandleValue
		} else if ok && functionFieldInfoHasTargetSet(fieldInfo) {
			paramName = fieldInfo.FunctionParamName
			captures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...)
			escapeCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionEscapeCaptures...)
			directSnapshotAlias = fieldInfo.FunctionDirectSnapshotAlias
			escapeKind = fieldInfo.FunctionEscapeKind
			handleValue = fieldInfo.FunctionHandleValue
			fieldSig, err := functionFieldInfoSig(fieldInfo)
			if err != nil {
				return FunctionFieldInfo{}, err
			}
			if err := validateFunctionInfoAssignable(
				localName+"."+field.Name,
				functionFieldLocalInfo(field),
				fieldSig,
				value.At,
			); err != nil {
				return FunctionFieldInfo{}, err
			}
		} else if globalInfo, _, globalOK, err := resolveFunctionTypedGlobalFieldAccess(
			value,
			globals,
			funcs,
		); err != nil {
			return FunctionFieldInfo{}, err
		} else if globalOK {
			target = globalInfo.FunctionValue
		} else if imported, importedOK := resolveImportedFunctionFieldAccess(
			value,
			funcs,
			module,
			imports,
		); importedOK {
			target = imported
		} else {
			return FunctionFieldInfo{}, unsupportedFunctionTypedStructFieldInitializerSourceError(
				value.At,
				localName,
				field.Name,
			)
		}
		if target != "" {
			targetSig, ok := funcs[target]
			if !ok {
				return FunctionFieldInfo{}, fmt.Errorf(
					"%s: unknown function symbol '%s'",
					frontend.FormatPos(value.At),
					target,
				)
			}
			if targetSig.Generic {
				return FunctionFieldInfo{}, unsupportedGenericFunctionTypedStructFieldInitializerError(
					value.At,
					callbackArgumentName(value),
					localName+"."+field.Name,
				)
			}
			if err := validateFunctionTypeSymbolSignature(
				localName+"."+field.Name,
				field.FunctionTypeRef,
				targetSig,
				module,
				imports,
				value.At,
			); err != nil {
				return FunctionFieldInfo{}, err
			}
			resolved = target
		}
	case *frontend.CallExpr:
		var err error
		resolved, err = validateFunctionTypeReturnCallBinding(
			localName+"."+field.Name,
			field.FunctionTypeRef,
			value,
			funcs,
			module,
			imports,
		)
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		metadataValue, metadataCaptures, metadataEscapeCaptures, metadataParamName, err :=
			functionAssignmentMetadataWithReturnParamRefs(
				value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		if metadataValue != "" {
			resolved = metadataValue
		}
		paramName = metadataParamName
		captures = append([]frontend.ClosureCapture(nil), metadataCaptures...)
		escapeCaptures = append([]frontend.ClosureCapture(nil), metadataEscapeCaptures...)
		if callSig, ok := funcs[value.Name]; ok && callSig.ReturnFunctionHandleValue {
			escapeKind = callSig.ReturnFunctionEscapeKind
			handleValue = callSig.ReturnFunctionHandleValue
		}
	case *frontend.ClosureExpr:
		target := functionFieldLocalInfo(field)
		if err := validateFunctionTypedClosureAssignment(
			localName+"."+field.Name,
			target,
			value,
			locals,
			funcs,
			types,
			module,
			imports,
			value.At,
		); err != nil {
			return FunctionFieldInfo{}, err
		}
		if len(value.Captures) > 0 {
			captureSlots, err := functionCaptureSlotCount(value.Captures, types)
			if err != nil {
				return FunctionFieldInfo{}, err
			}
			if captureSlots > FnPtrEnvSlotCount {
				var err error
				escapeKind, handleValue, err = classifyCallableEscape(
					callableBoundaryStructField,
					value.Captures,
					types,
				)
				if err != nil {
					return FunctionFieldInfo{}, err
				}
			}
		}
		resolved = closureFunctionValueName(value, funcs, module)
		captures = append([]frontend.ClosureCapture(nil), value.Captures...)
		directSnapshotAlias = len(value.Captures) > 0
	default:
		return FunctionFieldInfo{}, unsupportedFunctionTypedStructFieldInitializerSourceError(
			init.Pos(),
			localName,
			field.Name,
		)
	}
	touchesMutableGlobals, err := functionAssignmentValueTouchesMutableGlobals(
		init,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return FunctionFieldInfo{}, err
	}
	return FunctionFieldInfo{
		FunctionValue:                 resolved,
		FunctionParamName:             paramName,
		FunctionCaptures:              captures,
		FunctionEscapeCaptures:        escapeCaptures,
		FunctionTouchesMutableGlobals: touchesMutableGlobals,
		FunctionReturnSnapshotAlias: isFunctionReturnSnapshotAlias(
			init,
			funcs,
			captures,
			escapeCaptures,
			paramName,
		),
		FunctionDirectSnapshotAlias: directSnapshotAlias,
		FunctionEscapeKind:          escapeKind,
		FunctionHandleValue:         handleValue,
		FunctionParamTypes:          append([]string(nil), field.FunctionParamTypes...),
		FunctionParamOwnership:      append([]string(nil), field.FunctionParamOwnership...),
		FunctionReturnType:          field.FunctionReturnType,
		FunctionReturnOwnership:     field.FunctionReturnOwnership,
		FunctionThrowsType:          field.FunctionThrowsType,
		FunctionEffects:             append([]string(nil), field.FunctionEffects...),
	}, nil
}

func unsupportedFunctionTypedStructFieldInitializerSourceError(
	pos frontend.Position,
	localName, fieldName string,
) error {
	return fmt.Errorf(
		("%s: function-typed struct field '%s.%s' initializer must be " +
			"a supported fnptr source: closure literal, function-typed " +
			"local/global/struct field, direct named function/closure " +
			"symbol, or function-typed return call"),
		frontend.FormatPos(pos),
		localName,
		fieldName,
	)
}

func functionFieldLocalInfo(field FieldInfo) LocalInfo {
	return LocalInfo{
		SlotCount:               field.SlotCount,
		TypeName:                field.TypeName,
		FunctionTypeValue:       field.FunctionTypeValue,
		FunctionParamTypes:      append([]string(nil), field.FunctionParamTypes...),
		FunctionParamOwnership:  append([]string(nil), field.FunctionParamOwnership...),
		FunctionReturnType:      field.FunctionReturnType,
		FunctionReturnOwnership: field.FunctionReturnOwnership,
		FunctionThrowsType:      field.FunctionThrowsType,
		FunctionEffects:         append([]string(nil), field.FunctionEffects...),
	}
}

func functionFieldInfoFromField(field FieldInfo) FunctionFieldInfo {
	return FunctionFieldInfo{
		FunctionParamTypes:      append([]string(nil), field.FunctionParamTypes...),
		FunctionParamOwnership:  append([]string(nil), field.FunctionParamOwnership...),
		FunctionReturnType:      field.FunctionReturnType,
		FunctionReturnOwnership: field.FunctionReturnOwnership,
		FunctionThrowsType:      field.FunctionThrowsType,
		FunctionEffects:         append([]string(nil), field.FunctionEffects...),
	}
}

func declaredFunctionFieldsForStructType(
	typeName string,
	types map[string]*TypeInfo,
) map[string]FunctionFieldInfo {
	info, ok := types[typeName]
	if !ok || info.Kind != TypeStruct {
		return nil
	}
	out := map[string]FunctionFieldInfo{}
	for _, field := range info.Fields {
		if field.FunctionTypeValue {
			out[field.Name] = functionFieldInfoFromField(field)
			continue
		}
		nested := declaredFunctionFieldsForStructType(field.TypeName, types)
		for nestedName, nestedInfo := range nested {
			out[field.Name+"."+nestedName] = nestedInfo
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func functionFieldsForStructParameter(
	paramName, typeName string,
	types map[string]*TypeInfo,
) map[string]FunctionFieldInfo {
	fields := declaredFunctionFieldsForStructType(typeName, types)
	for fieldName, field := range fields {
		field.FunctionParamName = paramName + "." + fieldName
		fields[fieldName] = field
	}
	return fields
}

func declaredEnumPayloadFunctionsForType(
	typeName string,
	types map[string]*TypeInfo,
) map[string]FunctionFieldInfo {
	info, ok := types[typeName]
	if !ok || info.Kind != TypeEnum {
		return nil
	}
	out := map[string]FunctionFieldInfo{}
	for _, enumCase := range info.EnumCases {
		for i, isFunction := range enumCase.PayloadFunctionTypes {
			if !isFunction {
				continue
			}
			out[enumPayloadFunctionKey(enumCase.Ordinal, i)] = enumPayloadFunctionInfo(
				enumCase,
				i,
				"",
			)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func enumPayloadFunctionsForEnumParameter(
	paramName, typeName string,
	types map[string]*TypeInfo,
) map[string]FunctionFieldInfo {
	payloads := declaredEnumPayloadFunctionsForType(typeName, types)
	for payloadKey, payload := range payloads {
		payload.FunctionParamName = paramName + "#" + payloadKey
		payloads[payloadKey] = payload
	}
	return payloads
}

func enumPayloadFieldKey(fieldPath, payloadKey string) string {
	if fieldPath == "" {
		return payloadKey
	}
	return fieldPath + "#" + payloadKey
}

func enumPayloadFieldMatchesPrefix(fieldKey, prefix string) bool {
	return strings.HasPrefix(fieldKey, prefix+"#") || strings.HasPrefix(fieldKey, prefix+".")
}

func declaredEnumPayloadFieldsForStructType(
	typeName string,
	types map[string]*TypeInfo,
) map[string]FunctionFieldInfo {
	info, ok := types[typeName]
	if !ok || info.Kind != TypeStruct {
		return nil
	}
	out := map[string]FunctionFieldInfo{}
	for _, field := range info.Fields {
		if fieldInfo, ok := types[field.TypeName]; ok && fieldInfo.Kind == TypeEnum {
			for payloadKey, payload := range declaredEnumPayloadFunctionsForType(field.TypeName, types) {
				out[enumPayloadFieldKey(field.Name, payloadKey)] = payload
			}
			continue
		}
		nested := declaredEnumPayloadFieldsForStructType(field.TypeName, types)
		for nestedName, nestedInfo := range nested {
			out[field.Name+"."+nestedName] = nestedInfo
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func enumPayloadFieldsForStructParameter(
	paramName, typeName string,
	types map[string]*TypeInfo,
) map[string]FunctionFieldInfo {
	fields := declaredEnumPayloadFieldsForStructType(typeName, types)
	for fieldName, field := range fields {
		field.FunctionParamName = paramName + "." + fieldName
		fields[fieldName] = field
	}
	return fields
}

func closureFunctionValueName(
	closure *frontend.ClosureExpr,
	funcs map[string]FuncSig,
	module string,
) string {
	if closure == nil || closure.Name == "" {
		return ""
	}
	if _, ok := funcs[closure.Name]; ok {
		return closure.Name
	}
	if module != "" {
		qualified := qualifyName(module, closure.Name)
		if _, ok := funcs[qualified]; ok {
			return qualified
		}
	}
	return closure.Name
}

func functionFieldLocalInfoFromExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	types map[string]*TypeInfo,
) (string, LocalInfo, bool, error) {
	name := callbackArgumentName(expr)
	baseName, fields, pos, ok := splitFieldPath(expr)
	if !ok {
		return "", LocalInfo{}, false, nil
	}
	local, ok := locals[baseName]
	if !ok {
		return "", LocalInfo{}, false, nil
	}
	if len(fields) == 0 {
		return "", LocalInfo{}, false, nil
	}
	current := local.TypeName
	var field FieldInfo
	for _, fieldName := range fields {
		typeInfo, ok := types[current]
		if !ok || typeInfo.Kind != TypeStruct {
			return "", LocalInfo{}, false, nil
		}
		var fieldOK bool
		field, fieldOK = typeInfo.FieldMap[fieldName]
		if !fieldOK {
			return "", LocalInfo{}, false, nil
		}
		current = field.TypeName
	}
	if !field.FunctionTypeValue {
		return "", LocalInfo{}, false, nil
	}
	if _, _, _, err := resolveFieldChain(local.TypeName, local.Base, fields, types, pos); err != nil {
		return "", LocalInfo{}, false, err
	}
	return name, functionFieldLocalInfo(field), true, nil
}

func functionParamLocalInfo(sig FuncSig, index int) LocalInfo {
	info := LocalInfo{
		SlotCount: FnPtrSlotCount,
		TypeName:  "fnptr",
		FunctionTypeValue: index >= 0 && index < len(sig.ParamFunctionTypes) &&
			sig.ParamFunctionTypes[index],
	}
	if info.FunctionTypeValue && index >= 0 && index < len(sig.ParamNames) {
		info.FunctionParamName = sig.ParamNames[index]
	}
	if index >= 0 && index < len(sig.ParamFunctionParams) {
		info.FunctionParamTypes = append([]string(nil), sig.ParamFunctionParams[index]...)
	}
	if index >= 0 && index < len(sig.ParamFunctionOwnership) {
		info.FunctionParamOwnership = append([]string(nil), sig.ParamFunctionOwnership[index]...)
	}
	if index >= 0 && index < len(sig.ParamFunctionReturns) {
		info.FunctionReturnType = sig.ParamFunctionReturns[index]
	}
	if index >= 0 && index < len(sig.ParamFunctionReturnOwnership) {
		info.FunctionReturnOwnership = sig.ParamFunctionReturnOwnership[index]
	}
	if index >= 0 && index < len(sig.ParamFunctionThrows) {
		info.FunctionThrowsType = sig.ParamFunctionThrows[index]
	}
	if index >= 0 && index < len(sig.ParamFunctionEffects) {
		info.FunctionEffects = append([]string(nil), sig.ParamFunctionEffects[index]...)
	}
	return info
}

func functionReturnLocalInfo(sig FuncSig) LocalInfo {
	return LocalInfo{
		SlotCount:               FnPtrSlotCount,
		TypeName:                "fnptr",
		FunctionTypeValue:       sig.ReturnFunctionType,
		FunctionParamName:       sig.ReturnFunctionParamName,
		FunctionParamTypes:      append([]string(nil), sig.ReturnFunctionParams...),
		FunctionParamOwnership:  append([]string(nil), sig.ReturnFunctionParamOwnership...),
		FunctionReturnType:      sig.ReturnFunctionReturn,
		FunctionReturnOwnership: sig.ReturnFunctionReturnOwnership,
		FunctionThrowsType:      sig.ReturnFunctionThrows,
		FunctionEffects:         append([]string(nil), sig.ReturnFunctionEffects...),
	}
}

func functionFieldsFromStructLiteral(
	localName string,
	info *TypeInfo,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	if info == nil || info.Kind != TypeStruct {
		return nil, nil
	}
	byName := map[string]frontend.Expr{}
	if lit, ok := value.(*frontend.StructLitExpr); ok {
		byName = make(map[string]frontend.Expr, len(lit.Fields))
		for _, field := range lit.Fields {
			byName[field.Name] = field.Value
		}
	} else if call, ok := value.(*frontend.CallExpr); ok && call.Name == info.Name && len(
		call.ArgLabels,
	) == len(
		call.Args,
	) {
		byName = make(map[string]frontend.Expr, len(call.Args))
		for i, label := range call.ArgLabels {
			if label == "" {
				return nil, nil
			}
			byName[label] = call.Args[i]
		}
	} else {
		return nil, nil
	}
	out := map[string]FunctionFieldInfo{}
	for _, field := range info.Fields {
		init, ok := byName[field.Name]
		if !ok {
			continue
		}
		if field.FunctionTypeValue {
			fieldInfo, err := validateFunctionTypeStructFieldBinding(
				localName,
				field,
				init,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return nil, err
			}
			out[field.Name] = fieldInfo
			continue
		}
		nestedType, ok := types[field.TypeName]
		if !ok || nestedType.Kind != TypeStruct {
			continue
		}
		nested, err := functionFieldsFromStructLiteral(
			localName+"."+field.Name,
			nestedType,
			init,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		if len(nested) == 0 {
			nested = functionFieldsFromStructAlias(init, locals)
		}
		if len(nested) == 0 {
			nested, err = functionFieldsFromReturnCall(
				init,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				field.TypeName,
			)
			if err != nil {
				return nil, err
			}
		}
		for nestedName, nestedInfo := range nested {
			out[field.Name+"."+nestedName] = nestedInfo
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func functionFieldsFromStructAlias(
	value frontend.Expr,
	locals map[string]LocalInfo,
) map[string]FunctionFieldInfo {
	name := callbackArgumentName(value)
	if name == "" {
		return nil
	}
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return nil
	}
	local, ok := locals[parts[0]]
	if !ok || len(local.FunctionFields) == 0 {
		return nil
	}
	prefix := strings.Join(parts[1:], ".")
	out := map[string]FunctionFieldInfo{}
	for fieldName, fieldInfo := range local.FunctionFields {
		projected := fieldName
		if prefix != "" {
			if fieldName == prefix {
				continue
			}
			prefixWithDot := prefix + "."
			if !strings.HasPrefix(fieldName, prefixWithDot) {
				continue
			}
			projected = strings.TrimPrefix(fieldName, prefixWithDot)
		}
		if projected == "" {
			continue
		}
		out[projected] = cloneFunctionFieldInfo(fieldInfo)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func functionFieldsFromReturnedStructExpr(
	returnType string,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	info, ok := types[returnType]
	if !ok || info.Kind != TypeStruct {
		return nil, nil
	}
	fields, err := functionFieldsFromStructLiteral(
		"<return>",
		info,
		value,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		fields = functionFieldsFromStructAlias(value, locals)
	}
	if len(fields) == 0 {
		fields, err = functionFieldsFromReturnCall(
			value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			returnType,
		)
		if err != nil {
			return nil, err
		}
	}
	return fields, nil
}

func enumPayloadFieldsFromStructLiteral(
	info *TypeInfo,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	if info == nil || info.Kind != TypeStruct {
		return nil, nil
	}
	byName := map[string]frontend.Expr{}
	if lit, ok := value.(*frontend.StructLitExpr); ok {
		byName = make(map[string]frontend.Expr, len(lit.Fields))
		for _, field := range lit.Fields {
			byName[field.Name] = field.Value
		}
	} else if call, ok := value.(*frontend.CallExpr); ok && call.Name == info.Name && len(
		call.ArgLabels,
	) == len(
		call.Args,
	) {
		byName = make(map[string]frontend.Expr, len(call.Args))
		for i, label := range call.ArgLabels {
			if label == "" {
				return nil, nil
			}
			byName[label] = call.Args[i]
		}
	} else {
		return nil, nil
	}
	out := map[string]FunctionFieldInfo{}
	for _, field := range info.Fields {
		init, ok := byName[field.Name]
		if !ok {
			continue
		}
		fieldInfo, ok := types[field.TypeName]
		if !ok {
			continue
		}
		if fieldInfo.Kind == TypeEnum {
			payloads, err := enumPayloadFunctionsFromConstructor(
				fieldInfo,
				init,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return nil, err
			}
			if len(payloads) == 0 {
				payloads = enumPayloadFunctionsFromAlias(init, locals)
			}
			if len(payloads) == 0 {
				payloads, err = enumPayloadFunctionsFromReturnCall(
					init,
					locals,
					globals,
					funcs,
					types,
					module,
					imports,
					field.TypeName,
				)
				if err != nil {
					return nil, err
				}
			}
			for payloadKey, payload := range payloads {
				out[enumPayloadFieldKey(field.Name, payloadKey)] = payload
			}
			continue
		}
		if fieldInfo.Kind != TypeStruct {
			continue
		}
		nested, err := enumPayloadFieldsFromStructLiteral(
			fieldInfo,
			init,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		if len(nested) == 0 {
			nested = enumPayloadFieldsFromStructAlias(init, locals)
		}
		if len(nested) == 0 {
			nested, err = enumPayloadFieldsFromReturnCall(
				init,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				field.TypeName,
			)
			if err != nil {
				return nil, err
			}
		}
		for nestedName, nestedInfo := range nested {
			out[field.Name+"."+nestedName] = nestedInfo
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func enumPayloadFieldsFromStructAlias(
	value frontend.Expr,
	locals map[string]LocalInfo,
) map[string]FunctionFieldInfo {
	name := callbackArgumentName(value)
	if name == "" {
		return nil
	}
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return nil
	}
	local, ok := locals[parts[0]]
	if !ok || len(local.EnumPayloadFields) == 0 {
		return nil
	}
	prefix := strings.Join(parts[1:], ".")
	out := map[string]FunctionFieldInfo{}
	for fieldName, fieldInfo := range local.EnumPayloadFields {
		projected := fieldName
		if prefix != "" {
			prefixWithDot := prefix + "."
			if !strings.HasPrefix(fieldName, prefixWithDot) {
				continue
			}
			projected = strings.TrimPrefix(fieldName, prefixWithDot)
		}
		if projected == "" {
			continue
		}
		out[projected] = cloneFunctionFieldInfo(fieldInfo)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func enumPayloadFieldsFromReturnedStructExpr(
	returnType string,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	info, ok := types[returnType]
	if !ok || info.Kind != TypeStruct {
		return nil, nil
	}
	fields, err := enumPayloadFieldsFromStructLiteral(
		info,
		value,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		fields = enumPayloadFieldsFromStructAlias(value, locals)
	}
	if len(fields) == 0 {
		fields, err = enumPayloadFieldsFromReturnCall(
			value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			returnType,
		)
		if err != nil {
			return nil, err
		}
	}
	return fields, nil
}

func enumPayloadFieldsFromReturnCall(
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	expectedType string,
) (map[string]FunctionFieldInfo, error) {
	call, ok := value.(*frontend.CallExpr)
	if !ok {
		return nil, nil
	}
	resolved, err := resolveCheckedCallName(call.Name, funcs, module, imports, call.At)
	if err != nil {
		return nil, nil
	}
	sig, ok := funcs[resolved]
	if !ok || sig.ReturnType != expectedType {
		return nil, nil
	}
	if len(sig.ReturnEnumPayloadFields) == 0 {
		argumentCaptures, err := capturedFunctionTypedCallArgumentMetadata(
			sig,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		structArgumentCaptures, err := capturedFunctionTypedStructCallArgumentMetadata(
			sig,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		enumArgumentCaptures, err := capturedFunctionTypedEnumCallArgumentMetadata(
			sig,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		argumentCaptures = append(argumentCaptures, structArgumentCaptures...)
		argumentCaptures = append(argumentCaptures, enumArgumentCaptures...)
		if len(argumentCaptures) == 0 {
			return nil, nil
		}
		fields := declaredEnumPayloadFieldsForStructType(expectedType, types)
		for fieldName, field := range fields {
			field.FunctionEscapeCaptures = append(
				[]frontend.ClosureCapture(nil),
				argumentCaptures...)
			fields[fieldName] = field
		}
		return fields, nil
	}
	fields := cloneFunctionFieldMap(sig.ReturnEnumPayloadFields)
	for fieldName, field := range fields {
		if field.FunctionParamName == "" {
			fields[fieldName] = functionFieldInfoAsReturnSnapshot(field)
			continue
		}
		resolvedField, found, err := functionTypedReturnParamRefMetadata(
			sig,
			field.FunctionParamName,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil || !found {
			return nil, err
		}
		if resolvedField.FunctionValue != "" {
			field.FunctionValue = resolvedField.FunctionValue
		}
		if resolvedField.FunctionParamName != "" {
			field.FunctionParamName = resolvedField.FunctionParamName
		}
		field.FunctionEscapeCaptures = append(
			[]frontend.ClosureCapture(nil),
			resolvedField.FunctionCaptures...)
		field.FunctionEscapeCaptures = append(
			field.FunctionEscapeCaptures,
			resolvedField.FunctionEscapeCaptures...)
		fields[fieldName] = functionFieldInfoAsReturnSnapshot(field)
	}
	return fields, nil
}

func functionFieldsFromReturnCall(
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	expectedType string,
) (map[string]FunctionFieldInfo, error) {
	call, ok := value.(*frontend.CallExpr)
	if !ok {
		return nil, nil
	}
	resolved, err := resolveCheckedCallName(call.Name, funcs, module, imports, call.At)
	if err != nil {
		return nil, nil
	}
	sig, ok := funcs[resolved]
	if !ok || sig.ReturnType != expectedType {
		return nil, nil
	}
	if len(sig.ReturnFunctionFields) == 0 {
		argumentCaptures, err := capturedFunctionTypedCallArgumentMetadata(
			sig,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		structArgumentCaptures, err := capturedFunctionTypedStructCallArgumentMetadata(
			sig,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		argumentCaptures = append(argumentCaptures, structArgumentCaptures...)
		structArgumentFields, err := functionTypedStructCallArgumentFieldMetadata(
			sig,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		argumentParamName := ""
		for i, functionParam := range sig.ParamFunctionTypes {
			if !functionParam || i >= len(call.Args) {
				continue
			}
			_, _, _, paramName, err := functionAssignmentMetadataWithReturnParamRefs(
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return nil, err
			}
			if paramName != "" {
				argumentParamName = paramName
				break
			}
		}
		if len(argumentCaptures) == 0 && argumentParamName == "" && len(structArgumentFields) == 0 {
			return nil, nil
		}
		fields := declaredFunctionFieldsForStructType(expectedType, types)
		for fieldName, field := range fields {
			field.FunctionParamName = argumentParamName
			field.FunctionEscapeCaptures = append(
				[]frontend.ClosureCapture(nil),
				argumentCaptures...)
			fields[fieldName] = field
		}
		if fields == nil && len(structArgumentFields) > 0 {
			fields = map[string]FunctionFieldInfo{}
		}
		for fieldName, field := range structArgumentFields {
			mergeFunctionFieldInfoIntoMap(fields, fieldName, field)
		}
		return fields, nil
	}
	fields := cloneFunctionFieldMap(sig.ReturnFunctionFields)
	for fieldName, field := range fields {
		if field.FunctionParamName == "" {
			fields[fieldName] = functionFieldInfoAsReturnSnapshot(field)
			continue
		}
		resolvedField, found, err := functionTypedReturnParamRefMetadata(
			sig,
			field.FunctionParamName,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil || !found {
			return nil, err
		}
		if resolvedField.FunctionValue != "" {
			field.FunctionValue = resolvedField.FunctionValue
		}
		if resolvedField.FunctionParamName != "" {
			field.FunctionParamName = resolvedField.FunctionParamName
		}
		field.FunctionEscapeCaptures = append(
			[]frontend.ClosureCapture(nil),
			resolvedField.FunctionCaptures...)
		field.FunctionEscapeCaptures = append(
			field.FunctionEscapeCaptures,
			resolvedField.FunctionEscapeCaptures...)
		fields[fieldName] = functionFieldInfoAsReturnSnapshot(field)
	}
	return fields, nil
}

func functionFieldInfoAsReturnSnapshot(info FunctionFieldInfo) FunctionFieldInfo {
	if info.FunctionParamName != "" || info.FunctionValue == "" {
		return info
	}
	if len(info.FunctionCaptures) == 0 && len(info.FunctionEscapeCaptures) == 0 {
		return info
	}
	out := cloneFunctionFieldInfo(info)
	if len(out.FunctionCaptures) > 0 {
		out.FunctionEscapeCaptures = append(out.FunctionEscapeCaptures, out.FunctionCaptures...)
		out.FunctionCaptures = nil
	}
	out.FunctionReturnSnapshotAlias = true
	return out
}

func functionFieldReturnSnapshotMap(in map[string]FunctionFieldInfo) map[string]FunctionFieldInfo {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]FunctionFieldInfo, len(in))
	for name, info := range in {
		out[name] = functionFieldInfoAsReturnSnapshot(info)
	}
	return out
}

func functionTypedStructCallArgumentFieldMetadata(
	sig FuncSig,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	out := map[string]FunctionFieldInfo{}
	for i, typeName := range sig.ParamTypes {
		if i >= len(call.Args) {
			continue
		}
		info, ok := types[typeName]
		if !ok || info.Kind != TypeStruct {
			continue
		}
		fields := functionFieldsFromStructAlias(call.Args[i], locals)
		if len(fields) == 0 {
			var err error
			fields, err = functionFieldsFromStructLiteral(
				"<argument>",
				info,
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return nil, err
			}
		}
		if len(fields) == 0 {
			var err error
			fields, err = functionFieldsFromReturnCall(
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				typeName,
			)
			if err != nil {
				return nil, err
			}
		}
		for fieldName, field := range fields {
			if !functionFieldInfoHasTargetSet(field) {
				continue
			}
			mergeFunctionFieldInfoIntoMap(out, fieldName, field)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func capturedFunctionTypedStructCallArgumentMetadata(
	sig FuncSig,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) ([]frontend.ClosureCapture, error) {
	var out []frontend.ClosureCapture
	for i, typeName := range sig.ParamTypes {
		if i >= len(call.Args) {
			continue
		}
		info, ok := types[typeName]
		if !ok || info.Kind != TypeStruct {
			continue
		}
		fields := functionFieldsFromStructAlias(call.Args[i], locals)
		if len(fields) == 0 {
			var err error
			fields, err = functionFieldsFromStructLiteral(
				"<argument>",
				info,
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return nil, err
			}
		}
		if len(fields) == 0 {
			var err error
			fields, err = functionFieldsFromReturnCall(
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				typeName,
			)
			if err != nil {
				return nil, err
			}
		}
		for _, field := range fields {
			out = append(out, field.FunctionCaptures...)
			out = append(out, field.FunctionEscapeCaptures...)
		}
	}
	return out, nil
}

func capturedFunctionTypedCallArgumentMetadata(
	sig FuncSig,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) ([]frontend.ClosureCapture, error) {
	var out []frontend.ClosureCapture
	for i, functionParam := range sig.ParamFunctionTypes {
		if !functionParam || i >= len(call.Args) {
			continue
		}
		captures, escapeCaptures, err := functionTypedCallArgumentCaptureMetadata(
			sig,
			i,
			call.Args[i],
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, captures...)
		out = append(out, escapeCaptures...)
	}
	return out, nil
}

func cloneFunctionFieldMap(in map[string]FunctionFieldInfo) map[string]FunctionFieldInfo {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]FunctionFieldInfo, len(in))
	for name, info := range in {
		out[name] = cloneFunctionFieldInfo(info)
	}
	return out
}

func cloneFunctionFieldInfo(info FunctionFieldInfo) FunctionFieldInfo {
	return FunctionFieldInfo{
		FunctionValue:     info.FunctionValue,
		FunctionParamName: info.FunctionParamName,
		FunctionCaptures: append(
			[]frontend.ClosureCapture(nil),
			info.FunctionCaptures...),
		FunctionEscapeCaptures: append(
			[]frontend.ClosureCapture(nil),
			info.FunctionEscapeCaptures...),
		FunctionTouchesMutableGlobals: info.FunctionTouchesMutableGlobals,
		FunctionReturnSnapshotAlias:   info.FunctionReturnSnapshotAlias,
		FunctionDirectSnapshotAlias:   info.FunctionDirectSnapshotAlias,
		FunctionParamTypes:            append([]string(nil), info.FunctionParamTypes...),
		FunctionParamOwnership:        append([]string(nil), info.FunctionParamOwnership...),
		FunctionReturnType:            info.FunctionReturnType,
		FunctionReturnOwnership:       info.FunctionReturnOwnership,
		FunctionThrowsType:            info.FunctionThrowsType,
		FunctionEffects:               append([]string(nil), info.FunctionEffects...),
	}
}

func mergeFunctionFieldInfo(left, right FunctionFieldInfo) FunctionFieldInfo {
	out := cloneFunctionFieldInfo(left)
	if out.FunctionValue == "" {
		out.FunctionValue = right.FunctionValue
	}
	if out.FunctionParamName == "" {
		out.FunctionParamName = right.FunctionParamName
	}
	if len(out.FunctionCaptures) == 0 {
		out.FunctionCaptures = append([]frontend.ClosureCapture(nil), right.FunctionCaptures...)
	}
	if len(out.FunctionEscapeCaptures) == 0 {
		out.FunctionEscapeCaptures = append(
			[]frontend.ClosureCapture(nil),
			right.FunctionEscapeCaptures...)
	}
	out.FunctionTouchesMutableGlobals = out.FunctionTouchesMutableGlobals ||
		right.FunctionTouchesMutableGlobals
	out.FunctionDirectSnapshotAlias = out.FunctionDirectSnapshotAlias ||
		right.FunctionDirectSnapshotAlias
	if len(out.FunctionParamTypes) == 0 {
		out.FunctionParamTypes = append([]string(nil), right.FunctionParamTypes...)
	}
	if len(out.FunctionParamOwnership) == 0 {
		out.FunctionParamOwnership = append([]string(nil), right.FunctionParamOwnership...)
	}
	if out.FunctionReturnType == "" {
		out.FunctionReturnType = right.FunctionReturnType
	}
	if out.FunctionReturnOwnership == "" {
		out.FunctionReturnOwnership = right.FunctionReturnOwnership
	}
	if out.FunctionThrowsType == "" {
		out.FunctionThrowsType = right.FunctionThrowsType
	}
	if len(out.FunctionEffects) == 0 {
		out.FunctionEffects = append([]string(nil), right.FunctionEffects...)
	}
	return out
}

func mergeFunctionFieldInfoIntoMap(
	fields map[string]FunctionFieldInfo,
	name string,
	info FunctionFieldInfo,
) {
	if existing, exists := fields[name]; exists {
		fields[name] = mergeFunctionFieldInfo(existing, info)
		return
	}
	fields[name] = cloneFunctionFieldInfo(info)
}

func functionFieldInfoHasTargetSet(info FunctionFieldInfo) bool {
	return info.FunctionParamName != "" ||
		len(info.FunctionCaptures) > 0 ||
		len(info.FunctionEscapeCaptures) > 0 ||
		info.FunctionTouchesMutableGlobals
}

func functionFieldInfoSig(info FunctionFieldInfo) (FuncSig, error) {
	return buildInterfaceFuncSig("function-field", funcSigSpec{
		ParamTypes:          append([]string(nil), info.FunctionParamTypes...),
		ParamOwnership:      append([]string(nil), info.FunctionParamOwnership...),
		ReturnType:          info.FunctionReturnType,
		ReturnOwnership:     info.FunctionReturnOwnership,
		ThrowsType:          info.FunctionThrowsType,
		ReturnRegionParam:   regionNone,
		ReturnResourceParam: regionNone,
		Effects:             append([]string(nil), info.FunctionEffects...),
	}, nil)
}

func functionFieldMapsEqual(a, b map[string]FunctionFieldInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for name, left := range a {
		right, ok := b[name]
		if !ok || !functionFieldInfoEqual(left, right) {
			return false
		}
	}
	return true
}

func functionFieldInfoEqual(a, b FunctionFieldInfo) bool {
	return a.FunctionValue == b.FunctionValue &&
		a.FunctionParamName == b.FunctionParamName &&
		a.FunctionReturnType == b.FunctionReturnType &&
		a.FunctionReturnOwnership == b.FunctionReturnOwnership &&
		a.FunctionTouchesMutableGlobals == b.FunctionTouchesMutableGlobals &&
		closureCapturesEqual(a.FunctionCaptures, b.FunctionCaptures) &&
		closureCapturesEqual(a.FunctionEscapeCaptures, b.FunctionEscapeCaptures) &&
		stringSlicesEqual(a.FunctionParamTypes, b.FunctionParamTypes) &&
		stringSlicesEqual(a.FunctionParamOwnership, b.FunctionParamOwnership) &&
		stringSlicesEqual(a.FunctionEffects, b.FunctionEffects)
}

func closureCapturesEqual(a, b []frontend.ClosureCapture) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name || a[i].Type.Name != b[i].Type.Name ||
			a[i].Type.Kind != b[i].Type.Kind {
			return false
		}
	}
	return true
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---- function_types_enum_payload.go ----

func validateFunctionTypeReturnCallBinding(
	name string,
	declared frontend.TypeRef,
	init *frontend.CallExpr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) (string, error) {
	resolvedCall, err := resolveCheckedCallName(init.Name, funcs, module, imports, init.At)
	if err != nil {
		return "", unsupportedFunctionTypedLocalInitializerReturnCallSourceError(
			init.At,
			name,
			init.Name,
		)
	}
	init.Name = resolvedCall
	callSig, ok := funcs[resolvedCall]
	if !ok {
		return "", unsupportedFunctionTypedLocalInitializerReturnCallSourceError(
			init.At,
			name,
			init.Name,
		)
	}
	if !callSig.ReturnFunctionType {
		return "", unsupportedFunctionTypedLocalInitializerReturnCallSourceError(
			init.At,
			name,
			init.Name,
		)
	}
	if callSig.ReturnFunctionSymbol != "" {
		targetSig, ok := funcs[callSig.ReturnFunctionSymbol]
		if !ok {
			return "", fmt.Errorf(
				"%s: unknown returned function symbol '%s'",
				frontend.FormatPos(init.At),
				callSig.ReturnFunctionSymbol,
			)
		}
		if targetSig.Generic {
			return "", unsupportedGenericFunctionTypedLocalInitializerError(
				init.At,
				callSig.ReturnFunctionSymbol,
				name,
			)
		}
	}
	returnedSig, err := buildInterfaceFuncSig(name, funcSigSpec{
		ParamTypes:          append([]string(nil), callSig.ReturnFunctionParams...),
		ParamOwnership:      append([]string(nil), callSig.ReturnFunctionParamOwnership...),
		ReturnType:          callSig.ReturnFunctionReturn,
		ReturnOwnership:     callSig.ReturnFunctionReturnOwnership,
		ThrowsType:          callSig.ReturnFunctionThrows,
		ReturnRegionParam:   regionNone,
		ReturnResourceParam: regionNone,
		Effects:             append([]string(nil), callSig.ReturnFunctionEffects...),
	}, nil)
	if err != nil {
		return "", err
	}
	if err := validateFunctionTypeSymbolSignature(
		name,
		declared,
		returnedSig,
		module,
		imports,
		init.At,
	); err != nil {
		return "", err
	}
	return callSig.ReturnFunctionSymbol, nil
}

func resolveImportedFunctionFieldAccess(
	value *frontend.FieldAccessExpr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) (string, bool) {
	base, ok := value.Base.(*frontend.IdentExpr)
	if !ok {
		return "", false
	}
	importedModule, ok := imports[base.Name]
	if !ok || importedModule == "" {
		return "", false
	}
	name := importedModule + "." + value.Field
	sig, ok := funcs[name]
	if !ok {
		return "", false
	}
	if err := ensureFuncVisible(name, sig, module, value.At); err != nil {
		return "", false
	}
	return name, true
}

func resolveFunctionTypedGlobalFieldAccess(
	value *frontend.FieldAccessExpr,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
) (GlobalInfo, FuncSig, bool, error) {
	var zero FuncSig
	name := callbackArgumentName(value)
	global, ok := globals[name]
	if !ok {
		return GlobalInfo{}, zero, false, nil
	}
	if !global.FunctionTypeValue || global.FunctionValue == "" {
		if global.Mutable {
			return GlobalInfo{}, zero, true, unsupportedImportedMutableFunctionTypedGlobalUseError(
				value.At,
				name,
			)
		}
		return GlobalInfo{}, zero, true, unsupportedFunctionTypedGlobalTargetError(
			value.At,
			name,
		)
	}
	sig, ok := funcs[global.FunctionValue]
	if !ok {
		return GlobalInfo{}, zero, true, fmt.Errorf(
			"%s: unknown function symbol '%s'",
			frontend.FormatPos(value.At),
			global.FunctionValue,
		)
	}
	return global, sig, true, nil
}

func resolveImportedFunctionGlobalInitializer(
	value *frontend.FieldAccessExpr,
	world *module.World,
	currentModule string,
	imports map[string]string,
	types map[string]*TypeInfo,
) (string, FuncSig, bool, error) {
	var zero FuncSig
	base, ok := value.Base.(*frontend.IdentExpr)
	if !ok {
		return "", zero, false, nil
	}
	importedModule, ok := imports[base.Name]
	if !ok || importedModule == "" {
		return "", zero, false, nil
	}
	file := world.ByModule[importedModule]
	if file == nil {
		return "", zero, false, nil
	}
	var target *frontend.FuncDecl
	for _, fn := range file.Funcs {
		if fn != nil && fn.Name == value.Field {
			target = fn
			break
		}
	}
	if target == nil {
		return "", zero, false, nil
	}
	name := importedModule + "." + target.Name
	sig, err := funcSigFromDeclForGlobalInitializer(file, target, importedModule, types)
	if err != nil {
		return "", zero, false, err
	}
	if err := ensureFuncVisible(name, sig, currentModule, value.At); err != nil {
		return "", zero, false, err
	}
	return name, sig, true, nil
}

func funcSigFromDeclForGlobalInitializer(
	file *frontend.FileAST,
	fn *frontend.FuncDecl,
	currentModule string,
	types map[string]*TypeInfo,
) (FuncSig, error) {
	var zero FuncSig
	if err := validateFunctionParamNames(fn); err != nil {
		return zero, err
	}
	imports, err := collectImportAliases(file)
	if err != nil {
		return zero, err
	}
	effects, err := normalizeEffects(fn.Uses, fn.Pos)
	if err != nil {
		return zero, err
	}
	retName, err := resolveTypeName(&fn.ReturnType, currentModule, imports)
	if err != nil {
		return zero, err
	}
	throwsType := ""
	if fn.HasThrows {
		throwsType, err = resolveTypeName(&fn.Throws, currentModule, imports)
		if err != nil {
			return zero, err
		}
	}
	paramTypes := make([]string, 0, len(fn.Params))
	paramOwnership := make([]string, 0, len(fn.Params))
	for i := range fn.Params {
		param := &fn.Params[i]
		resolved, err := resolveTypeName(&param.Type, currentModule, imports)
		if err != nil {
			return zero, err
		}
		if _, err := ensureTypeInfo(resolved, types); err != nil {
			return zero, fmt.Errorf("%s: %v", frontend.FormatPos(param.At), err)
		}
		paramTypes = append(paramTypes, resolved)
		paramOwnership = append(paramOwnership, param.Ownership)
	}
	spec := funcSigSpec{
		Generic:             len(fn.TypeParams) > 0,
		Public:              declarationIsPublic(file, fn.Public),
		ParamTypes:          paramTypes,
		ParamOwnership:      paramOwnership,
		ReturnType:          retName,
		ReturnOwnership:     fn.ReturnOwnership,
		ThrowsType:          throwsType,
		ReturnRegionParam:   regionNone,
		ReturnResourceParam: initialReturnResourceParam(retName, types),
		Effects:             effects,
	}
	name := currentModule + "." + fn.Name
	if spec.Generic {
		return buildGenericFuncSig(name, spec, types)
	}
	return buildDeclaredFuncSig(name, spec, types)
}

func enumPayloadFunctionKey(ordinal int32, index int) string {
	return fmt.Sprintf("%d:%d", ordinal, index)
}

func enumPayloadFunctionInfo(
	caseInfo EnumCaseInfo,
	index int,
	functionValue string,
) FunctionFieldInfo {
	field := FunctionFieldInfo{FunctionValue: functionValue}
	if index >= 0 && index < len(caseInfo.PayloadFunctionParams) {
		field.FunctionParamTypes = append([]string(nil), caseInfo.PayloadFunctionParams[index]...)
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionOwns) {
		field.FunctionParamOwnership = append([]string(nil), caseInfo.PayloadFunctionOwns[index]...)
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionReturns) {
		field.FunctionReturnType = caseInfo.PayloadFunctionReturns[index]
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionReturnOwns) {
		field.FunctionReturnOwnership = caseInfo.PayloadFunctionReturnOwns[index]
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionThrows) {
		field.FunctionThrowsType = caseInfo.PayloadFunctionThrows[index]
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionEffects) {
		field.FunctionEffects = append([]string(nil), caseInfo.PayloadFunctionEffects[index]...)
	}
	return field
}

func validateFunctionTypeEnumPayloadBinding(
	enumType string,
	caseInfo EnumCaseInfo,
	index int,
	init frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (FunctionFieldInfo, error) {
	if index < 0 || index >= len(caseInfo.PayloadFunctionTypes) ||
		!caseInfo.PayloadFunctionTypes[index] {
		return FunctionFieldInfo{}, nil
	}
	label := fmt.Sprintf("%s.%s[%d]", displayTypeName(enumType, module), caseInfo.Name, index+1)
	resolved := ""
	paramName := ""
	captures := []frontend.ClosureCapture(nil)
	escapeCaptures := []frontend.ClosureCapture(nil)
	directSnapshotAlias := false
	escapeKind := CallableEscapeKind("")
	handleValue := false
	switch value := init.(type) {
	case *frontend.IdentExpr:
		source, localIdent := locals[value.Name]
		var err error
		resolved, err = validateFunctionTypeNamedSymbolBinding(
			label,
			caseInfo.PayloadFunctionRefs[index],
			value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			true,
			unsupportedGenericFunctionTypedEnumPayloadInitializerError,
		)
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		captures = append([]frontend.ClosureCapture(nil), source.FunctionCaptures...)
		escapeCaptures = append([]frontend.ClosureCapture(nil), source.FunctionEscapeCaptures...)
		directSnapshotAlias = source.FunctionDirectSnapshotAlias
		escapeKind = source.FunctionEscapeKind
		handleValue = source.FunctionHandleValue
		if source.FunctionTypeValue && source.FunctionValue == "" {
			paramName = source.FunctionParamName
			if paramName == "" {
				paramName = value.Name
			}
		}
		if resolved != "" && !localIdent {
			value.Name = resolved
		}
	case *frontend.FieldAccessExpr:
		fieldInfo, ok, err := resolveFunctionFieldArgument(value, locals)
		target := ""
		fieldTargetInfo := FunctionFieldInfo{}
		fieldTargetInfoOK := false
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		if ok && fieldInfo.FunctionValue != "" {
			target = fieldInfo.FunctionValue
			fieldTargetInfo = fieldInfo
			fieldTargetInfoOK = true
			captures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...)
			escapeCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionEscapeCaptures...)
			directSnapshotAlias = fieldInfo.FunctionDirectSnapshotAlias
			escapeKind = fieldInfo.FunctionEscapeKind
			handleValue = fieldInfo.FunctionHandleValue
		} else if ok && functionFieldInfoHasTargetSet(fieldInfo) {
			paramName = fieldInfo.FunctionParamName
			captures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...)
			escapeCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionEscapeCaptures...)
			directSnapshotAlias = fieldInfo.FunctionDirectSnapshotAlias
			escapeKind = fieldInfo.FunctionEscapeKind
			handleValue = fieldInfo.FunctionHandleValue
			fieldSig, err := functionFieldInfoSig(fieldInfo)
			if err != nil {
				return FunctionFieldInfo{}, err
			}
			if err := validateFunctionInfoAssignable(
				label,
				enumPayloadLocalInfo(caseInfo, index),
				fieldSig,
				value.At,
			); err != nil {
				return FunctionFieldInfo{}, err
			}
		} else if globalInfo, _, globalOK, err := resolveFunctionTypedGlobalFieldAccess(
			value,
			globals,
			funcs,
		); err != nil {
			return FunctionFieldInfo{}, err
		} else if globalOK {
			target = globalInfo.FunctionValue
		} else if imported, importedOK := resolveImportedFunctionFieldAccess(
			value,
			funcs,
			module,
			imports,
		); importedOK {
			target = imported
		} else {
			return FunctionFieldInfo{}, unsupportedFunctionTypedEnumPayloadInitializerSourceError(
				value.At,
				label,
			)
		}
		if target != "" {
			targetSig, ok := funcs[target]
			if !ok {
				return FunctionFieldInfo{}, fmt.Errorf(
					"%s: unknown function symbol '%s'",
					frontend.FormatPos(value.At),
					target,
				)
			}
			if targetSig.Generic {
				return FunctionFieldInfo{}, unsupportedGenericFunctionTypedEnumPayloadInitializerError(
					value.At,
					callbackArgumentName(value),
					label,
				)
			}
			if fieldTargetInfoOK {
				fieldSig, err := functionFieldInfoSig(fieldTargetInfo)
				if err != nil {
					return FunctionFieldInfo{}, err
				}
				if err := validateFunctionInfoAssignable(
					label,
					enumPayloadLocalInfo(caseInfo, index),
					fieldSig,
					value.At,
				); err != nil {
					return FunctionFieldInfo{}, err
				}
			} else {
				if err := validateFunctionTypeSymbolSignature(
					label,
					caseInfo.PayloadFunctionRefs[index],
					targetSig,
					module,
					imports,
					value.At,
				); err != nil {
					return FunctionFieldInfo{}, err
				}
			}
			resolved = target
		}
	case *frontend.CallExpr:
		var err error
		resolved, err = validateFunctionTypeReturnCallBinding(
			label,
			caseInfo.PayloadFunctionRefs[index],
			value,
			funcs,
			module,
			imports,
		)
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		metadataValue, metadataCaptures, metadataEscapeCaptures, metadataParamName, err :=
			functionAssignmentMetadataWithReturnParamRefs(
				value,
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		if metadataValue != "" {
			resolved = metadataValue
		}
		paramName = metadataParamName
		captures = append([]frontend.ClosureCapture(nil), metadataCaptures...)
		escapeCaptures = append([]frontend.ClosureCapture(nil), metadataEscapeCaptures...)
		if callSig, ok := funcs[value.Name]; ok && callSig.ReturnFunctionHandleValue {
			escapeKind = callSig.ReturnFunctionEscapeKind
			handleValue = callSig.ReturnFunctionHandleValue
		}
	case *frontend.ClosureExpr:
		target := enumPayloadLocalInfo(caseInfo, index)
		if err := validateFunctionTypedClosureAssignment(
			label,
			target,
			value,
			locals,
			funcs,
			types,
			module,
			imports,
			value.At,
		); err != nil {
			return FunctionFieldInfo{}, err
		}
		if len(value.Captures) > 0 {
			captureSlots, err := functionCaptureSlotCount(value.Captures, types)
			if err != nil {
				return FunctionFieldInfo{}, err
			}
			if captureSlots > FnPtrEnvSlotCount {
				var err error
				escapeKind, handleValue, err = classifyCallableEscape(
					callableBoundaryEnumPayload,
					value.Captures,
					types,
				)
				if err != nil {
					return FunctionFieldInfo{}, err
				}
			}
		}
		resolved = closureFunctionValueName(value, funcs, module)
		captures = append([]frontend.ClosureCapture(nil), value.Captures...)
		directSnapshotAlias = len(value.Captures) > 0
	default:
		return FunctionFieldInfo{}, unsupportedFunctionTypedEnumPayloadInitializerSourceError(
			init.Pos(),
			label,
		)
	}
	if index >= len(caseInfo.PayloadFunctionRefs) {
		return FunctionFieldInfo{}, fmt.Errorf(
			"%s: function-typed enum payload '%s' is missing function type metadata",
			frontend.FormatPos(init.Pos()),
			label,
		)
	}
	info := enumPayloadFunctionInfo(caseInfo, index, resolved)
	info.FunctionParamName = paramName
	info.FunctionCaptures = captures
	info.FunctionEscapeCaptures = escapeCaptures
	info.FunctionReturnSnapshotAlias = isFunctionReturnSnapshotAlias(
		init,
		funcs,
		captures,
		escapeCaptures,
		paramName,
	)
	info.FunctionDirectSnapshotAlias = directSnapshotAlias
	info.FunctionEscapeKind = escapeKind
	info.FunctionHandleValue = handleValue
	touchesMutableGlobals, err := functionAssignmentValueTouchesMutableGlobals(
		init,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return FunctionFieldInfo{}, err
	}
	info.FunctionTouchesMutableGlobals = touchesMutableGlobals
	return info, nil
}

func unsupportedFunctionTypedEnumPayloadInitializerSourceError(
	pos frontend.Position,
	label string,
) error {
	return fmt.Errorf(
		("%s: function-typed enum payload '%s' initializer must be a " +
			"supported fnptr source: closure literal, function-typed " +
			"local/global/struct field, direct named function/closure " +
			"symbol, or function-typed return call"),
		frontend.FormatPos(pos),
		label,
	)
}

func enumPayloadLocalInfo(caseInfo EnumCaseInfo, index int) LocalInfo {
	info := enumPayloadFunctionInfo(caseInfo, index, "")
	return LocalInfo{
		SlotCount:               FnPtrSlotCount,
		TypeName:                "fnptr",
		FunctionTypeValue:       true,
		FunctionParamTypes:      append([]string(nil), info.FunctionParamTypes...),
		FunctionParamOwnership:  append([]string(nil), info.FunctionParamOwnership...),
		FunctionReturnType:      info.FunctionReturnType,
		FunctionReturnOwnership: info.FunctionReturnOwnership,
		FunctionThrowsType:      info.FunctionThrowsType,
		FunctionEffects:         append([]string(nil), info.FunctionEffects...),
	}
}

func enumPayloadFunctionsFromConstructor(
	info *TypeInfo,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	if info == nil || info.Kind != TypeEnum {
		return nil, nil
	}
	call, ok := value.(*frontend.CallExpr)
	if !ok {
		return nil, nil
	}
	enumType, caseInfo, ok, err := resolveEnumCaseConstructorCall(call, types, module, imports)
	if err != nil {
		return nil, err
	}
	if !ok || enumType != info.Name {
		return nil, nil
	}
	out := map[string]FunctionFieldInfo{}
	for i, arg := range call.Args {
		if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
			continue
		}
		payloadInfo, err := validateFunctionTypeEnumPayloadBinding(
			enumType,
			caseInfo,
			i,
			arg,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		out[enumPayloadFunctionKey(caseInfo.Ordinal, i)] = payloadInfo
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func enumPayloadFunctionsFromReturnedEnumExpr(
	returnType string,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	info, ok := types[returnType]
	if !ok || info.Kind != TypeEnum {
		return nil, nil
	}
	payloads, err := enumPayloadFunctionsFromConstructor(
		info,
		value,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
	)
	if err != nil {
		return nil, err
	}
	if len(payloads) == 0 {
		payloads = enumPayloadFunctionValuesForExpr(value, locals)
	}
	if len(payloads) == 0 {
		payloads, err = enumPayloadFunctionsFromReturnCall(
			value,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
			returnType,
		)
		if err != nil {
			return nil, err
		}
	}
	return payloads, nil
}

func enumPayloadFunctionsFromReturnCall(
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	expectedType string,
) (map[string]FunctionFieldInfo, error) {
	call, ok := value.(*frontend.CallExpr)
	if !ok {
		return nil, nil
	}
	resolved, err := resolveCheckedCallName(call.Name, funcs, module, imports, call.At)
	if err != nil {
		return nil, nil
	}
	sig, ok := funcs[resolved]
	if !ok || sig.ReturnType != expectedType {
		return nil, nil
	}
	if len(sig.ReturnEnumPayloadFunctions) == 0 {
		argumentCaptures, err := capturedFunctionTypedCallArgumentMetadata(
			sig,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		enumArgumentCaptures, err := capturedFunctionTypedEnumCallArgumentMetadata(
			sig,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		argumentCaptures = append(argumentCaptures, enumArgumentCaptures...)
		enumArgumentPayloads, err := functionTypedEnumCallArgumentPayloadMetadata(
			sig,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil {
			return nil, err
		}
		argumentParamName := ""
		for i, functionParam := range sig.ParamFunctionTypes {
			if !functionParam || i >= len(call.Args) {
				continue
			}
			_, _, _, paramName, err := functionAssignmentMetadataWithReturnParamRefs(
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return nil, err
			}
			if paramName != "" {
				argumentParamName = paramName
				break
			}
		}
		if len(argumentCaptures) == 0 && argumentParamName == "" && len(enumArgumentPayloads) == 0 {
			return nil, nil
		}
		payloads := declaredEnumPayloadFunctionsForType(expectedType, types)
		for key, payload := range payloads {
			payload.FunctionParamName = argumentParamName
			payload.FunctionEscapeCaptures = append(
				[]frontend.ClosureCapture(nil),
				argumentCaptures...)
			payloads[key] = payload
		}
		if payloads == nil && len(enumArgumentPayloads) > 0 {
			payloads = map[string]FunctionFieldInfo{}
		}
		for key, payload := range enumArgumentPayloads {
			mergeFunctionFieldInfoIntoMap(payloads, key, payload)
		}
		return payloads, nil
	}
	payloads := cloneFunctionFieldMap(sig.ReturnEnumPayloadFunctions)
	for payloadKey, payload := range payloads {
		if payload.FunctionParamName == "" {
			payloads[payloadKey] = functionFieldInfoAsReturnSnapshot(payload)
			continue
		}
		resolvedPayload, found, err := functionTypedReturnParamRefMetadata(
			sig,
			payload.FunctionParamName,
			call,
			locals,
			globals,
			funcs,
			types,
			module,
			imports,
		)
		if err != nil || !found {
			return nil, err
		}
		if resolvedPayload.FunctionValue != "" {
			payload.FunctionValue = resolvedPayload.FunctionValue
		}
		if resolvedPayload.FunctionParamName != "" {
			payload.FunctionParamName = resolvedPayload.FunctionParamName
		}
		payload.FunctionEscapeCaptures = append(
			[]frontend.ClosureCapture(nil),
			resolvedPayload.FunctionCaptures...)
		payload.FunctionEscapeCaptures = append(
			payload.FunctionEscapeCaptures,
			resolvedPayload.FunctionEscapeCaptures...)
		payloads[payloadKey] = functionFieldInfoAsReturnSnapshot(payload)
	}
	return payloads, nil
}

func functionTypedEnumCallArgumentPayloadMetadata(
	sig FuncSig,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	out := map[string]FunctionFieldInfo{}
	for i, typeName := range sig.ParamTypes {
		if i >= len(call.Args) {
			continue
		}
		info, ok := types[typeName]
		if !ok || info.Kind != TypeEnum {
			continue
		}
		payloads := enumPayloadFunctionValuesForExpr(call.Args[i], locals)
		if len(payloads) == 0 {
			var err error
			payloads, err = enumPayloadFunctionsFromConstructor(
				info,
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return nil, err
			}
		}
		if len(payloads) == 0 {
			var err error
			payloads, err = enumPayloadFunctionsFromReturnCall(
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				typeName,
			)
			if err != nil {
				return nil, err
			}
		}
		for payloadKey, payload := range payloads {
			if !functionFieldInfoHasTargetSet(payload) {
				continue
			}
			mergeFunctionFieldInfoIntoMap(out, payloadKey, payload)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func capturedFunctionTypedEnumCallArgumentMetadata(
	sig FuncSig,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) ([]frontend.ClosureCapture, error) {
	var out []frontend.ClosureCapture
	for i, typeName := range sig.ParamTypes {
		if i >= len(call.Args) {
			continue
		}
		info, ok := types[typeName]
		if !ok || info.Kind != TypeEnum {
			continue
		}
		payloads := enumPayloadFunctionValuesForExpr(call.Args[i], locals)
		if len(payloads) == 0 {
			var err error
			payloads, err = enumPayloadFunctionsFromConstructor(
				info,
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
			)
			if err != nil {
				return nil, err
			}
		}
		if len(payloads) == 0 {
			var err error
			payloads, err = enumPayloadFunctionsFromReturnCall(
				call.Args[i],
				locals,
				globals,
				funcs,
				types,
				module,
				imports,
				typeName,
			)
			if err != nil {
				return nil, err
			}
		}
		for _, payload := range payloads {
			out = append(out, payload.FunctionCaptures...)
			out = append(out, payload.FunctionEscapeCaptures...)
		}
	}
	return out, nil
}

func enumPayloadFunctionsFromAlias(
	value frontend.Expr,
	locals map[string]LocalInfo,
) map[string]FunctionFieldInfo {
	if id, ok := value.(*frontend.IdentExpr); ok {
		local, ok := locals[id.Name]
		if !ok || len(local.EnumPayloadFunctions) == 0 {
			return nil
		}
		return cloneFunctionFieldMap(local.EnumPayloadFunctions)
	}
	return enumPayloadFunctionsFromStructFieldExpr(value, locals)
}

func functionLocalInfoForEnumPayload(
	caseInfo EnumCaseInfo,
	index int,
	value FunctionFieldInfo,
) LocalInfo {
	info := LocalInfo{
		SlotCount:         1,
		TypeName:          "ptr",
		Mutable:           false,
		FunctionValue:     value.FunctionValue,
		FunctionParamName: value.FunctionParamName,
		FunctionCaptures: append(
			[]frontend.ClosureCapture(nil),
			value.FunctionCaptures...),
		FunctionEscapeCaptures: append(
			[]frontend.ClosureCapture(nil),
			value.FunctionEscapeCaptures...),
		FunctionTouchesMutableGlobals: value.FunctionTouchesMutableGlobals,
		FunctionReturnSnapshotAlias:   value.FunctionReturnSnapshotAlias,
		FunctionDirectSnapshotAlias:   value.FunctionDirectSnapshotAlias,
		FunctionEscapeKind:            value.FunctionEscapeKind,
		FunctionHandleValue:           value.FunctionHandleValue,
		FunctionEnumPayload:           true,
		FunctionTypeValue:             true,
		FunctionParamTypes:            append([]string(nil), value.FunctionParamTypes...),
		FunctionParamOwnership:        append([]string(nil), value.FunctionParamOwnership...),
		FunctionReturnType:            value.FunctionReturnType,
		FunctionReturnOwnership:       value.FunctionReturnOwnership,
		FunctionThrowsType:            value.FunctionThrowsType,
		FunctionEffects:               append([]string(nil), value.FunctionEffects...),
	}
	if value.FunctionValue == "" {
		info.FunctionValue = ""
		info.FunctionParamTypes = append([]string(nil), caseInfo.PayloadFunctionParams[index]...)
		info.FunctionParamOwnership = append([]string(nil), caseInfo.PayloadFunctionOwns[index]...)
		info.FunctionReturnType = caseInfo.PayloadFunctionReturns[index]
		if index < len(caseInfo.PayloadFunctionReturnOwns) {
			info.FunctionReturnOwnership = caseInfo.PayloadFunctionReturnOwns[index]
		}
		if index < len(caseInfo.PayloadFunctionThrows) {
			info.FunctionThrowsType = caseInfo.PayloadFunctionThrows[index]
		}
		info.FunctionEffects = append([]string(nil), caseInfo.PayloadFunctionEffects[index]...)
	}
	return info
}

func enumPayloadFunctionValuesForExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
) map[string]FunctionFieldInfo {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		local, ok := locals[id.Name]
		if !ok || len(local.EnumPayloadFunctions) == 0 {
			return nil
		}
		return local.EnumPayloadFunctions
	}
	return enumPayloadFunctionsFromStructFieldExpr(expr, locals)
}

func enumPayloadFunctionsFromStructFieldExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
) map[string]FunctionFieldInfo {
	name := callbackArgumentName(expr)
	if name == "" {
		return nil
	}
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return nil
	}
	local, ok := locals[parts[0]]
	if !ok || len(local.EnumPayloadFields) == 0 {
		return nil
	}
	prefix := strings.Join(parts[1:], ".") + "#"
	out := map[string]FunctionFieldInfo{}
	for fieldName, fieldInfo := range local.EnumPayloadFields {
		if !strings.HasPrefix(fieldName, prefix) {
			continue
		}
		payloadKey := strings.TrimPrefix(fieldName, prefix)
		if payloadKey == "" {
			continue
		}
		out[payloadKey] = cloneFunctionFieldInfo(fieldInfo)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func enumPayloadFunctionValuesForMatchExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scrutType string,
) (map[string]FunctionFieldInfo, error) {
	if payloads := enumPayloadFunctionValuesForExpr(expr, locals); len(payloads) > 0 {
		return payloads, nil
	}
	return enumPayloadFunctionsFromReturnCall(
		expr,
		locals,
		globals,
		funcs,
		types,
		module,
		imports,
		scrutType,
	)
}

func bindEnumPatternFunctionPayloadLocals(
	pattern frontend.Expr,
	payloads map[string]FunctionFieldInfo,
	locals map[string]LocalInfo,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	if len(payloads) == 0 {
		return nil
	}
	enumPat, ok := pattern.(*frontend.EnumCasePatternExpr)
	if !ok {
		return nil
	}
	_, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	for i, binding := range enumPat.Bindings {
		if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
			continue
		}
		value, ok := payloads[enumPayloadFunctionKey(caseInfo.Ordinal, i)]
		if !ok {
			continue
		}
		hasMetadata := value.FunctionValue != "" ||
			value.FunctionParamName != "" ||
			len(value.FunctionCaptures) > 0 ||
			len(value.FunctionEscapeCaptures) > 0
		if !hasMetadata {
			continue
		}
		localInfo := functionLocalInfoForEnumPayload(caseInfo, i, value)
		if existing, exists := locals[binding]; exists {
			localInfo.Base = existing.Base
			localInfo.SlotCount = existing.SlotCount
			localInfo.TypeName = existing.TypeName
		}
		locals[binding] = localInfo
	}
	return nil
}

// ---- generics.go ----

type genericDef struct {
	module       string
	file         *frontend.FileAST
	imports      map[string]string
	decl         *frontend.FuncDecl
	conformances map[protocolConformanceKey]frontend.Position
	protocols    map[string]genericProtocolInfo
	knownTypes   map[string]struct{}
}

type genericStructDef struct {
	module string
	file   *frontend.FileAST
	decl   *frontend.StructDecl
}

type genericWorkItem struct {
	fn      *frontend.FuncDecl
	module  string
	imports map[string]string
}

type protocolConformanceKey struct {
	typeName string
	protocol string
}

type genericProtocolInfo struct {
	module string
	public bool
}

func monomorphizeFuncDeclFullName(module string, fn *frontend.FuncDecl) string {
	if fn != nil && fn.ExtensionOf != "" {
		return fn.Name
	}
	if fn == nil {
		return qualifyName(module, "")
	}
	return qualifyName(module, fn.Name)
}

func genericInstanceFullName(generic genericDef, name string) string {
	if generic.decl != nil && generic.decl.ExtensionOf != "" {
		return name
	}
	return qualifyName(generic.module, name)
}

func monomorphizeGenerics(world *module.World) error {
	if err := normalizeExtensionMethodNames(world); err != nil {
		return err
	}
	fileImports := make(map[*frontend.FileAST]map[string]string, len(world.Files))
	generics := map[string]genericDef{}
	for _, file := range world.Files {
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		fileImports[file] = imports
		for _, fn := range file.Funcs {
			if len(fn.TypeParams) > 0 {
				fullName := monomorphizeFuncDeclFullName(file.Module, fn)
				generics[fullName] = genericDef{
					module:  file.Module,
					file:    file,
					imports: imports,
					decl:    fn,
				}
			}
		}
	}
	conformances, err := collectProtocolConformances(world, fileImports)
	if err != nil {
		return err
	}
	protocols := collectGenericProtocolInfos(world)
	knownTypes := collectGenericKnownTypes(world)
	for name, def := range generics {
		def.conformances = conformances
		def.protocols = protocols
		def.knownTypes = knownTypes
		generics[name] = def
	}
	structCtx := newGenericStructContext(world, fileImports)
	if structCtx != nil {
		if err := structCtx.rewriteWorld(world); err != nil {
			return err
		}
	}
	if len(generics) == 0 {
		if structCtx != nil {
			structCtx.finalize(world)
		}
		return nil
	}

	funcDecls := map[string]*frontend.FuncDecl{}
	structDecls := map[string]*frontend.StructDecl{}
	enumDecls := map[string]*frontend.EnumDecl{}
	for _, file := range world.Files {
		for _, enum := range file.Enums {
			enumDecls[qualifyName(file.Module, enum.Name)] = enum
		}
		for _, st := range file.Structs {
			if len(st.TypeParams) == 0 {
				structDecls[qualifyName(file.Module, st.Name)] = st
			}
		}
		for _, fn := range file.Funcs {
			funcDecls[monomorphizeFuncDeclFullName(file.Module, fn)] = fn
		}
	}
	if structCtx != nil {
		for name, st := range structCtx.created {
			structDecls[name] = st
		}
	}
	created := map[string]*frontend.FuncDecl{}
	createdByFile := map[*frontend.FileAST]map[string]*frontend.FuncDecl{}
	var work []genericWorkItem
	for _, file := range world.Files {
		imports := fileImports[file]
		for _, fn := range file.Funcs {
			if len(fn.TypeParams) > 0 {
				continue
			}
			work = append(work, genericWorkItem{fn: fn, module: file.Module, imports: imports})
		}
	}
	for _, file := range world.Files {
		imports := fileImports[file]
		for _, glob := range file.Globals {
			if err := monomorphizeFunctionTypedGlobalInitializer(
				glob,
				generics,
				created,
				createdByFile,
				&work,
				fileImports,
				file.Module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		}
	}

	for i := 0; i < len(work); i++ {
		item := work[i]
		env := map[string]string{}
		for _, param := range item.fn.Params {
			resolved, err := resolveTypeName(&param.Type, item.module, item.imports)
			if err != nil {
				return err
			}
			env[param.Name] = resolved
		}
		if err := monomorphizeStmts(
			item.fn.Body,
			env,
			map[string]frontend.TypeRef{},
			item.fn.ReturnType,
			funcDecls,
			structDecls,
			enumDecls,
			generics,
			created,
			createdByFile,
			&work,
			fileImports,
			item.module,
			item.imports,
			structCtx,
		); err != nil {
			return err
		}
	}

	for _, file := range world.Files {
		perFile := createdByFile[file]
		if len(perFile) == 0 {
			continue
		}
		names := make([]string, 0, len(perFile))
		for name := range perFile {
			names = append(names, name)
		}
		sort.Strings(names)
		generated := make([]*frontend.FuncDecl, 0, len(names))
		for _, name := range names {
			generated = append(generated, perFile[name])
		}
		file.Funcs = append(generated, file.Funcs...)
	}
	if structCtx != nil {
		structCtx.finalize(world)
	}
	return nil
}

func monomorphizeFunctionTypedGlobalInitializer(
	glob *frontend.GlobalDecl,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	if glob == nil || glob.Type.Kind != frontend.TypeRefFunction || glob.Init == nil {
		return nil
	}
	original := glob.Init
	replacement, specialized, err := monomorphizeGenericFunctionValueExpr(
		glob.Init,
		glob.Type,
		fmt.Sprintf("function-typed global '%s'", glob.Name),
		generics,
		created,
		createdByFile,
		work,
		fileImports,
		module,
		imports,
		structCtx,
	)
	if err != nil || !specialized {
		return err
	}
	if id, ok := replacement.(*frontend.IdentExpr); ok {
		switch init := original.(type) {
		case *frontend.FieldAccessExpr:
			glob.Init = &frontend.FieldAccessExpr{At: init.At, Base: init.Base, Field: lastNameSegment(
				id.Name,
			)}
			return nil
		case *frontend.IdentExpr:
			name := id.Name
			if module != "" {
				name = strings.TrimPrefix(name, module+".")
			}
			glob.Init = &frontend.IdentExpr{At: init.At, Name: name}
			return nil
		}
	}
	glob.Init = replacement
	return nil
}

func lastNameSegment(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func collectProtocolConformances(
	world *module.World,
	fileImports map[*frontend.FileAST]map[string]string,
) (map[protocolConformanceKey]frontend.Position, error) {
	conformances := map[protocolConformanceKey]frontend.Position{}
	for _, file := range world.Files {
		imports := fileImports[file]
		for _, impl := range file.Impls {
			typeName, err := resolveTypeName(&impl.Type, file.Module, imports)
			if err != nil {
				return nil, err
			}
			protoName, err := resolveTypeName(&impl.Protocol, file.Module, imports)
			if err != nil {
				return nil, err
			}
			conformances[protocolConformanceKey{typeName: typeName, protocol: protoName}] = impl.At
		}
	}
	return conformances, nil
}

func collectGenericProtocolInfos(world *module.World) map[string]genericProtocolInfo {
	protocols := map[string]genericProtocolInfo{}
	for _, file := range world.Files {
		for _, proto := range file.Protocols {
			fullName := qualifyName(file.Module, proto.Name)
			protocols[fullName] = genericProtocolInfo{
				module: file.Module,
				public: declarationIsPublic(file, proto.Public),
			}
		}
	}
	return protocols
}

func collectGenericKnownTypes(world *module.World) map[string]struct{} {
	known := map[string]struct{}{}
	for name := range baseTypes() {
		known[name] = struct{}{}
	}
	for _, file := range world.Files {
		for _, st := range file.Structs {
			known[qualifyName(file.Module, st.Name)] = struct{}{}
		}
		for _, st := range file.States {
			known[qualifyName(file.Module, st.Name)] = struct{}{}
		}
		for _, en := range file.Enums {
			known[qualifyName(file.Module, en.Name)] = struct{}{}
		}
	}
	return known
}

func monomorphizeGenericStructs(
	world *module.World,
	fileImports map[*frontend.FileAST]map[string]string,
) error {
	ctx := newGenericStructContext(world, fileImports)
	if ctx == nil {
		return nil
	}
	if err := ctx.rewriteWorld(world); err != nil {
		return err
	}
	ctx.finalize(world)
	return nil
}

func newGenericStructContext(
	world *module.World,
	fileImports map[*frontend.FileAST]map[string]string,
) *genericStructContext {
	templates := map[string]genericStructDef{}
	for _, file := range world.Files {
		for _, st := range file.Structs {
			if len(st.TypeParams) == 0 {
				continue
			}
			fullName := qualifyName(file.Module, st.Name)
			templates[fullName] = genericStructDef{module: file.Module, file: file, decl: st}
		}
	}
	if len(templates) == 0 {
		return nil
	}

	created := map[string]*frontend.StructDecl{}
	createdByFile := map[*frontend.FileAST]map[string]*frontend.StructDecl{}
	return &genericStructContext{
		templates:     templates,
		created:       created,
		createdByFile: createdByFile,
		fileImports:   fileImports,
	}
}

func (ctx *genericStructContext) rewriteWorld(world *module.World) error {
	for _, file := range world.Files {
		imports := ctx.fileImports[file]
		for _, st := range file.Structs {
			if len(st.TypeParams) > 0 {
				continue
			}
			for i := range st.Fields {
				if err := ctx.rewriteTypeRef(&st.Fields[i].Type, file.Module, imports); err != nil {
					return err
				}
			}
		}
		for _, enum := range file.Enums {
			for i := range enum.Cases {
				for j := range enum.Cases[i].Payload {
					if err := ctx.rewriteTypeRef(&enum.Cases[i].Payload[j], file.Module, imports); err != nil {
						return err
					}
				}
			}
		}
		for _, glob := range file.Globals {
			if err := ctx.rewriteTypeRef(&glob.Type, file.Module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(glob.Init, file.Module, imports); err != nil {
				return err
			}
		}
		for _, fn := range file.Funcs {
			if len(fn.TypeParams) > 0 {
				continue
			}
			if err := ctx.rewriteTypeRef(&fn.ReturnType, file.Module, imports); err != nil {
				return err
			}
			if fn.HasThrows {
				if err := ctx.rewriteTypeRef(&fn.Throws, file.Module, imports); err != nil {
					return err
				}
			}
			for i := range fn.Params {
				if err := ctx.rewriteTypeRef(&fn.Params[i].Type, file.Module, imports); err != nil {
					return err
				}
			}
			if err := ctx.rewriteStmts(fn.Body, file.Module, imports); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ctx *genericStructContext) finalize(world *module.World) {
	for _, file := range world.Files {
		kept := file.Structs[:0]
		for _, st := range file.Structs {
			if len(st.TypeParams) == 0 {
				kept = append(kept, st)
			}
		}
		perFile := ctx.createdByFile[file]
		if len(perFile) > 0 {
			names := make([]string, 0, len(perFile))
			for name := range perFile {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				kept = append(kept, perFile[name])
			}
		}
		file.Structs = kept
	}
}

type genericStructContext struct {
	templates     map[string]genericStructDef
	created       map[string]*frontend.StructDecl
	createdByFile map[*frontend.FileAST]map[string]*frontend.StructDecl
	fileImports   map[*frontend.FileAST]map[string]string
}

func (ctx *genericStructContext) rewriteTypeRef(
	ref *frontend.TypeRef,
	module string,
	imports map[string]string,
) error {
	if ref == nil {
		return nil
	}
	switch ref.Kind {
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		return ctx.rewriteTypeRef(ref.Elem, module, imports)
	case frontend.TypeRefFunction:
		for i := range ref.Params {
			if err := ctx.rewriteTypeRef(&ref.Params[i], module, imports); err != nil {
				return err
			}
		}
		return ctx.rewriteTypeRef(ref.Return, module, imports)
	case frontend.TypeRefNamed:
		for i := range ref.TypeArgs {
			if err := ctx.rewriteTypeRef(&ref.TypeArgs[i], module, imports); err != nil {
				return err
			}
		}
		base := *ref
		base.TypeArgs = nil
		resolved, err := resolveTypeName(&base, module, imports)
		if err != nil {
			return err
		}
		generic, ok := ctx.templates[resolved]
		if !ok {
			if len(ref.TypeArgs) > 0 {
				return fmt.Errorf(
					"%s: type '%s' does not accept type arguments",
					frontend.FormatPos(ref.At),
					ref.Name,
				)
			}
			return nil
		}
		if len(ref.TypeArgs) == 0 {
			return fmt.Errorf(
				"%s: generic struct '%s' requires %d type argument(s)",
				frontend.FormatPos(ref.At),
				ref.Name,
				len(generic.decl.TypeParams),
			)
		}
		if len(ref.TypeArgs) != len(generic.decl.TypeParams) {
			return fmt.Errorf(
				"%s: generic struct '%s' expects %d type argument, got %d",
				frontend.FormatPos(ref.At),
				ref.Name,
				len(generic.decl.TypeParams),
				len(ref.TypeArgs),
			)
		}
		subst := map[string]string{}
		for i, tp := range generic.decl.TypeParams {
			if containsFunctionTypeRef(ref.TypeArgs[i]) {
				return fmt.Errorf(
					("%s: generic struct '%s' type argument '%s' uses function " +
						"type; generic struct instantiation cannot carry " +
						"function-typed values under the supported fnptr ABI"),
					frontend.FormatPos(ref.TypeArgs[i].At),
					ref.Name,
					tp,
				)
			}
			argName, err := resolveTypeName(&ref.TypeArgs[i], module, imports)
			if err != nil {
				return err
			}
			subst[tp] = argName
		}
		name, err := ctx.instantiate(generic, subst, ref.At)
		if err != nil {
			return err
		}
		ref.Name = qualifyName(generic.module, name)
		ref.TypeArgs = nil
		return nil
	default:
		return nil
	}
}

func (ctx *genericStructContext) instantiate(
	generic genericStructDef,
	subst map[string]string,
	at frontend.Position,
) (string, error) {
	name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
	fullName := qualifyName(generic.module, name)
	if existing, exists := ctx.created[fullName]; exists {
		if hasGenericStructTemplateRefs(existing) {
			return "", fmt.Errorf(
				"%s: nested generic struct instantiation for '%s' is not supported in this MVP",
				frontend.FormatPos(at),
				displayTypeName(fullName, generic.module),
			)
		}
		return name, nil
	}
	clone := *generic.decl
	clone.Name = name
	clone.TypeParams = nil
	clone.Fields = make([]frontend.FieldDecl, len(generic.decl.Fields))
	ctx.created[fullName] = &clone
	if _, ok := ctx.createdByFile[generic.file]; !ok {
		ctx.createdByFile[generic.file] = map[string]*frontend.StructDecl{}
	}
	ctx.createdByFile[generic.file][name] = &clone
	imports := ctx.fileImports[generic.file]
	for i, field := range generic.decl.Fields {
		clone.Fields[i] = field
		clone.Fields[i].Type = substituteTypeRef(field.Type, subst)
		if hasGenericTypeArgs(clone.Fields[i].Type) {
			return "", fmt.Errorf(
				"%s: nested generic struct instantiation for '%s.%s' is not supported in this MVP",
				frontend.FormatPos(clone.Fields[i].At),
				displayTypeName(fullName, generic.module),
				clone.Fields[i].Name,
			)
		}
		if err := ctx.rewriteTypeRef(&clone.Fields[i].Type, generic.module, imports); err != nil {
			return "", err
		}
	}
	if hasGenericStructTemplateRefs(&clone) {
		return "", fmt.Errorf(
			"%s: nested generic struct instantiation for '%s' is not supported in this MVP",
			frontend.FormatPos(at),
			displayTypeName(fullName, generic.module),
		)
	}
	return name, nil
}

func hasGenericStructTemplateRefs(st *frontend.StructDecl) bool {
	if st == nil {
		return false
	}
	for i := range st.Fields {
		if hasGenericTypeArgs(st.Fields[i].Type) {
			return true
		}
	}
	return false
}

func hasGenericTypeArgs(ref frontend.TypeRef) bool {
	if len(ref.TypeArgs) > 0 {
		return true
	}
	if ref.Elem != nil && hasGenericTypeArgs(*ref.Elem) {
		return true
	}
	if ref.Return != nil && hasGenericTypeArgs(*ref.Return) {
		return true
	}
	for i := range ref.Params {
		if hasGenericTypeArgs(ref.Params[i]) {
			return true
		}
	}
	return false
}

func hasDirectFunctionTypeRef(ref frontend.TypeRef) bool {
	return ref.Kind == frontend.TypeRefFunction
}

func containsFunctionTypeRef(ref frontend.TypeRef) bool {
	if ref.Kind == frontend.TypeRefFunction {
		return true
	}
	if ref.Elem != nil && containsFunctionTypeRef(*ref.Elem) {
		return true
	}
	if ref.Return != nil && containsFunctionTypeRef(*ref.Return) {
		return true
	}
	for i := range ref.Params {
		if containsFunctionTypeRef(ref.Params[i]) {
			return true
		}
	}
	for i := range ref.TypeArgs {
		if containsFunctionTypeRef(ref.TypeArgs[i]) {
			return true
		}
	}
	return false
}

func (ctx *genericStructContext) rewriteStmts(
	stmts []frontend.Stmt,
	module string,
	imports map[string]string,
) error {
	for _, stmt := range stmts {
		if err := ctx.rewriteStmt(stmt, module, imports); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *genericStructContext) rewriteStmt(
	stmt frontend.Stmt,
	module string,
	imports map[string]string,
) error {
	switch s := stmt.(type) {
	case *frontend.ReturnStmt:
		return ctx.rewriteExpr(s.Value, module, imports)
	case *frontend.ThrowStmt:
		return ctx.rewriteExpr(s.Value, module, imports)
	case *frontend.DeferStmt:
		return ctx.rewriteStmts(s.Body, module, imports)
	case *frontend.PrintStmt:
		return ctx.rewriteExpr(s.Value, module, imports)
	case *frontend.ExpectStmt:
		return ctx.rewriteExpr(s.Cond, module, imports)
	case *frontend.FreeStmt:
		return ctx.rewriteExpr(s.Value, module, imports)
	case *frontend.LetStmt:
		if s.Type.Name != "" || s.Type.Elem != nil {
			if err := ctx.rewriteTypeRef(&s.Type, module, imports); err != nil {
				return err
			}
		}
		return ctx.rewriteExpr(s.Value, module, imports)
	case *frontend.AssignStmt:
		if err := ctx.rewriteExpr(s.Target, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteExpr(s.Value, module, imports); err != nil {
			return err
		}
		return ctx.rewriteExpr(s.CompoundValue, module, imports)
	case *frontend.IfStmt:
		if err := ctx.rewriteExpr(s.Cond, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteStmts(s.Then, module, imports); err != nil {
			return err
		}
		return ctx.rewriteStmts(s.Else, module, imports)
	case *frontend.IfLetStmt:
		if err := ctx.rewriteExpr(s.Pattern, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteExpr(s.Value, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteStmts(s.Then, module, imports); err != nil {
			return err
		}
		return ctx.rewriteStmts(s.Else, module, imports)
	case *frontend.WhileStmt:
		if err := ctx.rewriteExpr(s.Cond, module, imports); err != nil {
			return err
		}
		return ctx.rewriteStmts(s.Body, module, imports)
	case *frontend.ForRangeStmt:
		if err := ctx.rewriteExpr(s.Start, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteExpr(s.End, module, imports); err != nil {
			return err
		}
		if err := ctx.rewriteExpr(s.Iterable, module, imports); err != nil {
			return err
		}
		return ctx.rewriteStmts(s.Body, module, imports)
	case *frontend.MatchStmt:
		if err := ctx.rewriteExpr(s.Value, module, imports); err != nil {
			return err
		}
		for i := range s.Cases {
			if err := ctx.rewriteExpr(s.Cases[i].Pattern, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(s.Cases[i].Guard, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteStmts(s.Cases[i].Body, module, imports); err != nil {
				return err
			}
		}
		return nil
	case *frontend.UnsafeStmt:
		return ctx.rewriteStmts(s.Body, module, imports)
	case *frontend.IslandStmt:
		if err := ctx.rewriteExpr(s.Size, module, imports); err != nil {
			return err
		}
		return ctx.rewriteStmts(s.Body, module, imports)
	case *frontend.ExprStmt:
		return ctx.rewriteExpr(s.Expr, module, imports)
	default:
		return nil
	}
}

func (ctx *genericStructContext) rewriteExpr(
	expr frontend.Expr,
	module string,
	imports map[string]string,
) error {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *frontend.BinaryExpr:
		if err := ctx.rewriteExpr(e.Left, module, imports); err != nil {
			return err
		}
		return ctx.rewriteExpr(e.Right, module, imports)
	case *frontend.UnaryExpr:
		return ctx.rewriteExpr(e.X, module, imports)
	case *frontend.TryExpr:
		return ctx.rewriteExpr(e.X, module, imports)
	case *frontend.AwaitExpr:
		return ctx.rewriteExpr(e.X, module, imports)
	case *frontend.CatchExpr:
		if err := ctx.rewriteExpr(e.Call, module, imports); err != nil {
			return err
		}
		for i := range e.Cases {
			if err := ctx.rewriteExpr(e.Cases[i].Pattern, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(e.Cases[i].Guard, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(e.Cases[i].Value, module, imports); err != nil {
				return err
			}
		}
		return nil
	case *frontend.MatchExpr:
		if err := ctx.rewriteExpr(e.Value, module, imports); err != nil {
			return err
		}
		for i := range e.Cases {
			if err := ctx.rewriteExpr(e.Cases[i].Pattern, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(e.Cases[i].Guard, module, imports); err != nil {
				return err
			}
			if err := ctx.rewriteExpr(e.Cases[i].Value, module, imports); err != nil {
				return err
			}
		}
		return nil
	case *frontend.CallExpr:
		for i := range e.TypeArgs {
			if err := ctx.rewriteTypeRef(&e.TypeArgs[i], module, imports); err != nil {
				return err
			}
		}
		for _, arg := range e.Args {
			if err := ctx.rewriteExpr(arg, module, imports); err != nil {
				return err
			}
		}
		return nil
	case *frontend.StructLitExpr:
		if err := ctx.rewriteTypeRef(&e.Type, module, imports); err != nil {
			return err
		}
		for _, field := range e.Fields {
			if err := ctx.rewriteExpr(field.Value, module, imports); err != nil {
				return err
			}
		}
		return nil
	case *frontend.FieldAccessExpr:
		return ctx.rewriteExpr(e.Base, module, imports)
	case *frontend.IndexExpr:
		if err := ctx.rewriteExpr(e.Base, module, imports); err != nil {
			return err
		}
		return ctx.rewriteExpr(e.Index, module, imports)
	default:
		return nil
	}
}

// ---- generics_clone_types.go ----

func monomorphizeExpr(
	expr frontend.Expr,
	env map[string]string,
	funcDecls map[string]*frontend.FuncDecl,
	structDecls map[string]*frontend.StructDecl,
	enumDecls map[string]*frontend.EnumDecl,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (string, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", nil
	case *frontend.BoolLitExpr:
		return "bool", nil
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.NoneLitExpr:
		return "none", nil
	case *frontend.SomePatternExpr:
		return "", nil
	case *frontend.EnumCasePatternExpr:
		return "", nil
	case *frontend.IdentExpr:
		if tname, ok := env[e.Name]; ok {
			return tname, nil
		}
		resolved, err := resolveMonomorphizeCallName(e.Name, module, imports, funcDecls, generics, e.At)
		if err == nil {
			if fn, ok := funcDecls[resolved]; ok && len(fn.TypeParams) == 0 {
				return genericTypeName(functionTypeRefFromFuncDecl(fn)), nil
			}
		}
		return "", nil
	case *frontend.FieldAccessExpr:
		baseType, err := monomorphizeExpr(
			e.Base,
			env,
			funcDecls,
			structDecls,
			enumDecls,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		)
		if err != nil {
			return "", err
		}
		if decl, ok := structDecls[baseType]; ok {
			for _, field := range decl.Fields {
				if field.Name == e.Field {
					return genericTypeName(qualifyFieldAssignmentPathType(field.Type, baseType)), nil
				}
			}
		}
		return "", nil
	case *frontend.IndexExpr:
		if _, err := monomorphizeExpr(
			e.Base,
			env,
			funcDecls,
			structDecls,
			enumDecls,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		); err != nil {
			return "", err
		}
		if _, err := monomorphizeExpr(
			e.Index,
			env,
			funcDecls,
			structDecls,
			enumDecls,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		); err != nil {
			return "", err
		}
		return "", nil
	case *frontend.UnaryExpr:
		_, err := monomorphizeExpr(
			e.X,
			env,
			funcDecls,
			structDecls,
			enumDecls,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		)
		return "i32", err
	case *frontend.BinaryExpr:
		if _, err := monomorphizeExpr(
			e.Left,
			env,
			funcDecls,
			structDecls,
			enumDecls,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		); err != nil {
			return "", err
		}
		if _, err := monomorphizeExpr(
			e.Right,
			env,
			funcDecls,
			structDecls,
			enumDecls,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		); err != nil {
			return "", err
		}
		switch e.Op {
		case frontend.TokenEqEq,
			frontend.TokenBangEq,
			frontend.TokenLess,
			frontend.TokenLessEq,
			frontend.TokenGreater,
			frontend.TokenGreaterEq,
			frontend.TokenAmpAmp, frontend.TokenPipePipe:
			return "bool", nil
		default:
			return "i32", nil
		}
	case *frontend.TryExpr:
		return monomorphizeExpr(
			e.X,
			env,
			funcDecls,
			structDecls,
			enumDecls,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		)
	case *frontend.CatchExpr:
		resultType, err := monomorphizeExpr(
			e.Call,
			env,
			funcDecls,
			structDecls,
			enumDecls,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		)
		if err != nil {
			return "", err
		}
		for _, c := range e.Cases {
			caseEnv := cloneStringMap(env)
			if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				caseEnv[some.Name] = "i32"
			}
			if !c.Default {
				if _, err := monomorphizeExpr(
					c.Pattern,
					caseEnv,
					funcDecls,
					structDecls,
					enumDecls,
					generics,
					created,
					createdByFile,
					work,
					fileImports,
					module,
					imports,
					structCtx,
				); err != nil {
					return "", err
				}
			}
			if c.Guard != nil {
				if _, err := monomorphizeExpr(
					c.Guard,
					caseEnv,
					funcDecls,
					structDecls,
					enumDecls,
					generics,
					created,
					createdByFile,
					work,
					fileImports,
					module,
					imports,
					structCtx,
				); err != nil {
					return "", err
				}
			}
			if _, err := monomorphizeExpr(
				c.Value,
				caseEnv,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return "", err
			}
		}
		return resultType, nil
	case *frontend.AwaitExpr:
		return monomorphizeExpr(
			e.X,
			env,
			funcDecls,
			structDecls,
			enumDecls,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		)
	case *frontend.StructLitExpr:
		resolved, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return "", err
		}
		if decl, ok := structDecls[resolved]; ok {
			fields := make(map[string]frontend.TypeRef, len(decl.Fields))
			for _, field := range decl.Fields {
				fields[field.Name] = field.Type
			}
			for i := range e.Fields {
				field := &e.Fields[i]
				declared := fields[field.Name]
				if declared.Kind != frontend.TypeRefFunction {
					continue
				}
				if closure, ok := field.Value.(*frontend.ClosureExpr); ok && isGenericClosureLiteral(closure) {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return "", unsupportedGenericClosureCaptureError(pos, name)
					}
					replacement, specialized, err := monomorphizeGenericClosureLiteralValue(
						closure,
						declared,
						fmt.Sprintf("struct field '%s.%s'", e.Type.Name, field.Name),
						generics,
						created,
						createdByFile,
						work,
						fileImports,
						module,
						imports,
						structCtx,
					)
					if err != nil {
						return "", err
					}
					if specialized {
						field.Value = replacement
						continue
					}
				}
				replacement, specialized, err := monomorphizeGenericFunctionValueExpr(
					field.Value,
					declared,
					fmt.Sprintf("struct field '%s.%s'", e.Type.Name, field.Name),
					generics,
					created,
					createdByFile,
					work,
					fileImports,
					module,
					imports,
					structCtx,
				)
				if err != nil {
					return "", err
				}
				if specialized {
					field.Value = replacement
				}
			}
		}
		for _, field := range e.Fields {
			if _, err := monomorphizeExpr(
				field.Value,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return "", err
			}
		}
		e.Type.Name = resolved
		return resolved, nil
	case *frontend.CallExpr:
		if err := monomorphizeFunctionTypedEnumPayloadArgs(
			e,
			enumDecls,
			env,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		); err != nil {
			return "", err
		}
		argTypes := make([]string, 0, len(e.Args))
		for _, arg := range e.Args {
			tname, err := monomorphizeExpr(
				arg,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			)
			if err != nil {
				return "", err
			}
			argTypes = append(argTypes, tname)
		}
		if enumTypeName, ok := monomorphizeEnumConstructorTypeName(e, enumDecls, module, imports); ok {
			return enumTypeName, nil
		}

		resolved := e.Name
		if localGenericClosure, ok := env[genericClosureBindingKey(e.Name)]; ok {
			if _, isGeneric := generics[localGenericClosure]; isGeneric {
				resolved = localGenericClosure
			}
		}
		if resolved == e.Name {
			if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
				resolved = builtin
			} else {
				var err error
				resolved, err = resolveMonomorphizeCallName(e.Name, module, imports, funcDecls, generics, e.At)
				if err != nil {
					return "", err
				}
			}
		}
		generic, ok := generics[resolved]
		if !ok {
			if callee, exists := funcDecls[resolved]; exists {
				if err := monomorphizeFunctionTypedCallArgs(
					e,
					callee,
					env,
					generics,
					created,
					createdByFile,
					work,
					fileImports,
					module,
					imports,
					structCtx,
				); err != nil {
					return "", err
				}
				return monomorphizeConcreteReturnType(callee, resolved, module, imports)
			}
			return "", nil
		}
		if len(e.Args) != len(generic.decl.Params) {
			return "", fmt.Errorf(
				"%s: wrong argument count for generic function '%s'",
				frontend.FormatPos(e.At),
				e.Name,
			)
		}
		subst := map[string]string{}
		for i, param := range generic.decl.Params {
			if argTypes[i] == "" {
				return "", fmt.Errorf(
					"%s: cannot infer generic argument for '%s' arg %d",
					frontend.FormatPos(e.Args[i].Pos()),
					e.Name,
					i+1,
				)
			}
			if err := bindGenericType(param.Type, argTypes[i], generic.decl.TypeParams, subst); err != nil {
				return "", fmt.Errorf("%s: %v", frontend.FormatPos(e.Args[i].Pos()), err)
			}
		}
		for _, tp := range generic.decl.TypeParams {
			if subst[tp] == "" {
				return "", fmt.Errorf(
					"%s: cannot infer generic argument '%s' for '%s'",
					frontend.FormatPos(e.At),
					tp,
					e.Name,
				)
			}
		}
		if err := checkGenericProtocolBounds(e.At, e.Name, generic, subst, module); err != nil {
			return "", err
		}
		name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
		fullName := genericInstanceFullName(generic, name)
		var returnType frontend.TypeRef
		if existing, exists := created[fullName]; exists {
			returnType = existing.ReturnType
		} else {
			clone := cloneGenericFunc(generic.decl, name, subst)
			cloneImports := fileImports[generic.file]
			if structCtx != nil {
				if err := rewriteGenericFuncStructRefs(
					structCtx,
					clone,
					generic.module,
					cloneImports,
				); err != nil {
					return "", err
				}
			}
			returnType = clone.ReturnType
			created[fullName] = clone
			if _, ok := createdByFile[generic.file]; !ok {
				createdByFile[generic.file] = map[string]*frontend.FuncDecl{}
			}
			createdByFile[generic.file][name] = clone
			*work = append(
				*work,
				genericWorkItem{fn: clone, module: generic.module, imports: fileImports[generic.file]},
			)
		}
		e.Name = fullName
		return genericTypeName(returnType), nil
	case *frontend.ClosureExpr:
		return "ptr", nil
	default:
		return "", nil
	}
}

func checkGenericProtocolBounds(
	callPos frontend.Position,
	callName string,
	generic genericDef,
	subst map[string]string,
	callerModule string,
) error {
	for _, bound := range generic.decl.TypeParamBounds {
		actual := subst[bound.Name]
		if actual == "" {
			continue
		}
		boundRef := bound.Bound
		protoName, err := resolveTypeName(&boundRef, generic.module, generic.imports)
		if err != nil {
			return err
		}
		protoInfo, ok := generic.protocols[protoName]
		if !ok {
			if _, isType := generic.knownTypes[protoName]; isType {
				return fmt.Errorf(
					"%s: generic bound '%s' for '%s' must name a protocol, got non-protocol type '%s'",
					frontend.FormatPos(bound.Bound.At),
					displayTypeName(protoName, generic.module),
					bound.Name,
					displayTypeName(protoName, generic.module),
				)
			}
			return fmt.Errorf("%s: unknown protocol bound '%s' for generic parameter '%s'",
				frontend.FormatPos(bound.Bound.At),
				displayTypeName(protoName, generic.module),
				bound.Name,
			)
		}
		if !symbolBelongsToModule(protoName, callerModule) && !protoInfo.public {
			return fmt.Errorf("%s: private protocol '%s' is not visible from module '%s'",
				frontend.FormatPos(callPos),
				protoName,
				callerModule,
			)
		}
		if _, ok := generic.conformances[protocolConformanceKey{
			typeName: actual,
			protocol: protoName,
		}]; ok {
			continue
		}
		return fmt.Errorf(
			("%s: generic argument '%s' does not satisfy bound '%s' for " +
				"'%s' in call to '%s' (missing impl %s: %s)"),
			frontend.FormatPos(callPos),
			displayTypeName(actual, generic.module),
			displayTypeName(protoName, generic.module),
			bound.Name,
			callName,
			displayTypeName(actual, generic.module),
			displayTypeName(protoName, generic.module),
		)
	}
	return nil
}

func rewriteGenericFuncStructRefs(
	ctx *genericStructContext,
	fn *frontend.FuncDecl,
	module string,
	imports map[string]string,
) error {
	if err := ctx.rewriteTypeRef(&fn.ReturnType, module, imports); err != nil {
		return err
	}
	if fn.HasThrows {
		if err := ctx.rewriteTypeRef(&fn.Throws, module, imports); err != nil {
			return err
		}
	}
	for i := range fn.Params {
		if err := ctx.rewriteTypeRef(&fn.Params[i].Type, module, imports); err != nil {
			return err
		}
	}
	return ctx.rewriteStmts(fn.Body, module, imports)
}

func monomorphizeIterableElemType(typeName string) (string, bool) {
	if strings.HasPrefix(typeName, "[]") {
		return strings.TrimPrefix(typeName, "[]"), true
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return elem, true
	}
	if typeName == "str" {
		return "u8", true
	}
	return "", false
}

func bindGenericType(
	param frontend.TypeRef,
	actual string,
	typeParams []string,
	subst map[string]string,
) error {
	if param.Kind == frontend.TypeRefNamed && contains(typeParams, param.Name) {
		if existing := subst[param.Name]; existing != "" && existing != actual {
			return fmt.Errorf(
				"conflicting generic argument for '%s': %s vs %s",
				param.Name,
				existing,
				actual,
			)
		}
		subst[param.Name] = actual
		return nil
	}
	switch param.Kind {
	case frontend.TypeRefNamed:
		if len(param.TypeArgs) == 0 {
			return nil
		}
		if matched, err := bindGenericNamedTypeArgs(param, actual, typeParams, subst); err != nil ||
			matched {
			return err
		}
	case frontend.TypeRefFunction:
		actualParams, actualReturn, ok := parseGenericFunctionTypeName(actual)
		if !ok {
			return nil
		}
		if len(param.Params) != len(actualParams) {
			return nil
		}
		for i := range param.Params {
			if err := bindGenericType(param.Params[i], actualParams[i], typeParams, subst); err != nil {
				return err
			}
		}
		if param.Return != nil {
			return bindGenericType(*param.Return, actualReturn, typeParams, subst)
		}
	case frontend.TypeRefOptional:
		if param.Elem == nil {
			return nil
		}
		elemActual := actual
		if elem, ok := optionalElemName(actual); ok {
			elemActual = elem
		}
		return bindGenericType(*param.Elem, elemActual, typeParams, subst)
	case frontend.TypeRefSlice:
		if param.Elem == nil {
			return nil
		}
		if elemActual, ok := sliceElemName(actual); ok {
			return bindGenericType(*param.Elem, elemActual, typeParams, subst)
		}
	case frontend.TypeRefArray:
		if param.Elem == nil {
			return nil
		}
		if elemActual, ok := arrayElemName(actual, param.Len); ok {
			return bindGenericType(*param.Elem, elemActual, typeParams, subst)
		}
	}
	return nil
}

func bindGenericNamedTypeArgs(
	param frontend.TypeRef,
	actual string,
	typeParams []string,
	subst map[string]string,
) (bool, error) {
	paramBase := lastNameSegment(param.Name)
	actualBase := lastNameSegment(actual)
	prefix := paramBase + "__"
	if paramBase == "" || !strings.HasPrefix(actualBase, prefix) {
		return false, nil
	}
	rest := strings.TrimPrefix(actualBase, prefix)
	for i, arg := range param.TypeArgs {
		argName := genericNamedTypeArgSegmentName(arg)
		if argName == "" {
			return false, nil
		}
		marker := argName + "_"
		if !strings.HasPrefix(rest, marker) {
			return false, nil
		}
		rest = strings.TrimPrefix(rest, marker)
		var encoded string
		if i+1 < len(param.TypeArgs) {
			nextArgName := genericNamedTypeArgSegmentName(param.TypeArgs[i+1])
			if nextArgName == "" {
				return false, nil
			}
			nextMarker := "__" + nextArgName + "_"
			idx := strings.Index(rest, nextMarker)
			if idx < 0 {
				return false, nil
			}
			encoded = rest[:idx]
			rest = rest[idx+len("__"):]
		} else {
			encoded = rest
			rest = ""
		}
		concrete, err := unsanitizeGenericType(encoded)
		if err != nil {
			return true, err
		}
		if canonical, ok := canonicalBuiltinType(concrete); ok {
			concrete = canonical
		}
		if err := bindGenericType(arg, concrete, typeParams, subst); err != nil {
			return true, err
		}
	}
	if rest != "" {
		return false, nil
	}
	return true, nil
}

func genericNamedTypeArgSegmentName(ref frontend.TypeRef) string {
	if ref.Kind == frontend.TypeRefNamed && ref.Name != "" {
		return ref.Name
	}
	return genericTypeName(ref)
}

func arrayElemName(name string, wantLen int) (string, bool) {
	gotLen, elem, ok := parseArrayTypeName(name)
	if !ok || gotLen != wantLen {
		return "", false
	}
	return elem, true
}

func cloneGenericFunc(
	fn *frontend.FuncDecl,
	name string,
	subst map[string]string,
) *frontend.FuncDecl {
	out := *fn
	out.Name = name
	out.ExportName = ""
	out.TypeParams = nil
	out.TypeParamBounds = nil
	out.Params = cloneParams(fn.Params, subst)
	out.ReturnType = substituteTypeRef(fn.ReturnType, subst)
	out.Throws = substituteTypeRef(fn.Throws, subst)
	out.Uses = append([]string(nil), fn.Uses...)
	out.SemanticClauses = append([]frontend.SemanticClause(nil), fn.SemanticClauses...)
	out.Body = cloneStmts(fn.Body, subst)
	return &out
}

func cloneParams(params []frontend.ParamDecl, subst map[string]string) []frontend.ParamDecl {
	out := make([]frontend.ParamDecl, len(params))
	for i, p := range params {
		out[i] = p
		out[i].Type = substituteTypeRef(p.Type, subst)
	}
	return out
}

func cloneStmts(stmts []frontend.Stmt, subst map[string]string) []frontend.Stmt {
	out := make([]frontend.Stmt, 0, len(stmts))
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			out = append(out, &frontend.ReturnStmt{At: s.At, Value: cloneExpr(s.Value, subst)})
		case *frontend.ThrowStmt:
			out = append(out, &frontend.ThrowStmt{At: s.At, Value: cloneExpr(s.Value, subst)})
		case *frontend.DeferStmt:
			out = append(out, &frontend.DeferStmt{At: s.At, Body: cloneStmts(s.Body, subst)})
		case *frontend.BreakStmt:
			out = append(out, &frontend.BreakStmt{At: s.At})
		case *frontend.ContinueStmt:
			out = append(out, &frontend.ContinueStmt{At: s.At})
		case *frontend.PrintStmt:
			out = append(out, &frontend.PrintStmt{At: s.At, Value: cloneExpr(s.Value, subst)})
		case *frontend.ExpectStmt:
			out = append(out, &frontend.ExpectStmt{At: s.At, Cond: cloneExpr(s.Cond, subst)})
		case *frontend.FreeStmt:
			out = append(
				out,
				&frontend.FreeStmt{At: s.At, Value: cloneExpr(s.Value, subst), Implicit: s.Implicit},
			)
		case *frontend.LetStmt:
			out = append(
				out,
				&frontend.LetStmt{At: s.At, Name: s.Name, Type: substituteTypeRef(
					s.Type,
					subst,
				), Mutable: s.Mutable, Const: s.Const, Value: cloneExpr(
					s.Value,
					subst,
				)},
			)
		case *frontend.AssignStmt:
			var compoundValue frontend.Expr
			if s.CompoundValue != nil {
				compoundValue = cloneExpr(s.CompoundValue, subst)
			}
			out = append(
				out,
				&frontend.AssignStmt{At: s.At, Target: cloneExpr(
					s.Target,
					subst,
				), Value: cloneExpr(
					s.Value,
					subst,
				), Op: s.Op, CompoundValue: compoundValue},
			)
		case *frontend.IfStmt:
			out = append(
				out,
				&frontend.IfStmt{At: s.At, Cond: cloneExpr(
					s.Cond,
					subst,
				), Then: cloneStmts(
					s.Then,
					subst,
				), Else: cloneStmts(
					s.Else,
					subst,
				)},
			)
		case *frontend.IfLetStmt:
			out = append(
				out,
				&frontend.IfLetStmt{At: s.At, Name: s.Name, Pattern: cloneExpr(
					s.Pattern,
					subst,
				), Value: cloneExpr(
					s.Value,
					subst,
				), ValueLocal: s.ValueLocal, Then: cloneStmts(
					s.Then,
					subst,
				), Else: cloneStmts(
					s.Else,
					subst,
				)},
			)
		case *frontend.WhileStmt:
			out = append(
				out,
				&frontend.WhileStmt{At: s.At, Cond: cloneExpr(s.Cond, subst), Body: cloneStmts(s.Body, subst)},
			)
		case *frontend.ForRangeStmt:
			var start, end, iterable frontend.Expr
			if s.Start != nil {
				start = cloneExpr(s.Start, subst)
			}
			if s.End != nil {
				end = cloneExpr(s.End, subst)
			}
			if s.Iterable != nil {
				iterable = cloneExpr(s.Iterable, subst)
			}
			out = append(
				out,
				&frontend.ForRangeStmt{
					At:            s.At,
					Name:          s.Name,
					Start:         start,
					End:           end,
					Iterable:      iterable,
					IterableLocal: s.IterableLocal,
					IndexLocal:    s.IndexLocal,
					EndLocal:      s.EndLocal,
					Body:          cloneStmts(s.Body, subst),
				},
			)
		case *frontend.MatchStmt:
			cases := make([]frontend.MatchCase, 0, len(s.Cases))
			for _, c := range s.Cases {
				cases = append(
					cases,
					frontend.MatchCase{At: c.At, Pattern: cloneExpr(
						c.Pattern,
						subst,
					), Guard: cloneExpr(
						c.Guard,
						subst,
					), Default: c.Default, Body: cloneStmts(
						c.Body,
						subst,
					)},
				)
			}
			out = append(
				out,
				&frontend.MatchStmt{At: s.At, Value: cloneExpr(
					s.Value,
					subst,
				), ScrutineeLocal: s.ScrutineeLocal, Cases: cases},
			)
		case *frontend.UnsafeStmt:
			out = append(out, &frontend.UnsafeStmt{At: s.At, Body: cloneStmts(s.Body, subst)})
		case *frontend.IslandStmt:
			out = append(
				out,
				&frontend.IslandStmt{At: s.At, Size: cloneExpr(
					s.Size,
					subst,
				), Name: s.Name, Body: cloneStmts(
					s.Body,
					subst,
				)},
			)
		case *frontend.ExprStmt:
			out = append(out, &frontend.ExprStmt{At: s.At, Expr: cloneExpr(s.Expr, subst)})
		default:
			out = append(out, stmt)
		}
	}
	return out
}

func cloneExpr(expr frontend.Expr, subst map[string]string) frontend.Expr {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return &frontend.NumberExpr{At: e.At, Value: e.Value}
	case *frontend.BoolLitExpr:
		return &frontend.BoolLitExpr{At: e.At, Value: e.Value}
	case *frontend.NoneLitExpr:
		return &frontend.NoneLitExpr{At: e.At}
	case *frontend.SomePatternExpr:
		return &frontend.SomePatternExpr{At: e.At, Name: e.Name}
	case *frontend.EnumCasePatternExpr:
		return &frontend.EnumCasePatternExpr{
			At:           e.At,
			TypeName:     e.TypeName,
			CaseName:     e.CaseName,
			Bindings:     append([]string(nil), e.Bindings...),
			HasPayload:   e.HasPayload,
			EnumType:     e.EnumType,
			EnumOrdinal:  e.EnumOrdinal,
			PayloadSlots: append([]int(nil), e.PayloadSlots...),
		}
	case *frontend.StringLitExpr:
		return &frontend.StringLitExpr{At: e.At, Value: append([]byte(nil), e.Value...)}
	case *frontend.IdentExpr:
		return &frontend.IdentExpr{At: e.At, Name: e.Name}
	case *frontend.FieldAccessExpr:
		return &frontend.FieldAccessExpr{At: e.At, Base: cloneExpr(
			e.Base,
			subst,
		), Field: e.Field, EnumType: e.EnumType, EnumOrdinal: e.EnumOrdinal}
	case *frontend.IndexExpr:
		return &frontend.IndexExpr{At: e.At, Base: cloneExpr(
			e.Base,
			subst,
		), Index: cloneExpr(
			e.Index,
			subst,
		)}
	case *frontend.UnaryExpr:
		return &frontend.UnaryExpr{At: e.At, Op: e.Op, X: cloneExpr(e.X, subst)}
	case *frontend.BinaryExpr:
		return &frontend.BinaryExpr{At: e.At, Op: e.Op, Left: cloneExpr(
			e.Left,
			subst,
		), Right: cloneExpr(
			e.Right,
			subst,
		)}
	case *frontend.TryExpr:
		return &frontend.TryExpr{At: e.At, X: cloneExpr(e.X, subst)}
	case *frontend.CatchExpr:
		cases := make([]frontend.CatchExprCase, 0, len(e.Cases))
		for _, c := range e.Cases {
			cases = append(
				cases,
				frontend.CatchExprCase{At: c.At, Pattern: cloneExpr(
					c.Pattern,
					subst,
				), Guard: cloneExpr(
					c.Guard,
					subst,
				), Default: c.Default, Value: cloneExpr(
					c.Value,
					subst,
				)},
			)
		}
		return &frontend.CatchExpr{
			At:          e.At,
			Call:        cloneExpr(e.Call, subst),
			ErrorLocal:  e.ErrorLocal,
			ResultLocal: e.ResultLocal,
			ErrorType:   e.ErrorType,
			ResultType:  e.ResultType,
			Cases:       cases,
		}
	case *frontend.AwaitExpr:
		return &frontend.AwaitExpr{At: e.At, X: cloneExpr(e.X, subst)}
	case *frontend.CallExpr:
		args := make([]frontend.Expr, 0, len(e.Args))
		for _, arg := range e.Args {
			args = append(args, cloneExpr(arg, subst))
		}
		labels := append([]string(nil), e.ArgLabels...)
		typeArgs := make([]frontend.TypeRef, len(e.TypeArgs))
		for i := range e.TypeArgs {
			typeArgs[i] = substituteTypeRef(e.TypeArgs[i], subst)
		}
		return &frontend.CallExpr{
			At:           e.At,
			Name:         e.Name,
			TypeArgs:     typeArgs,
			Args:         args,
			ArgLabels:    labels,
			ResolvedType: e.ResolvedType,
		}
	case *frontend.StructLitExpr:
		fields := make([]frontend.StructFieldInit, 0, len(e.Fields))
		for _, field := range e.Fields {
			fields = append(
				fields,
				frontend.StructFieldInit{At: field.At, Name: field.Name, Value: cloneExpr(field.Value, subst)},
			)
		}
		return &frontend.StructLitExpr{At: e.At, Type: substituteTypeRef(e.Type, subst), Fields: fields}
	case *frontend.ClosureExpr:
		return &frontend.ClosureExpr{At: e.At, Name: e.Name, Decl: e.Decl, Captures: cloneClosureCaptures(
			e.Captures,
			subst,
		)}
	default:
		return expr
	}
}

func cloneClosureCaptures(
	captures []frontend.ClosureCapture,
	subst map[string]string,
) []frontend.ClosureCapture {
	if len(captures) == 0 {
		return nil
	}
	out := make([]frontend.ClosureCapture, len(captures))
	for i, capture := range captures {
		out[i] = frontend.ClosureCapture{
			At:      capture.At,
			Name:    capture.Name,
			Type:    substituteTypeRef(capture.Type, subst),
			Mutable: capture.Mutable,
		}
	}
	return out
}

func substituteTypeRef(ref frontend.TypeRef, subst map[string]string) frontend.TypeRef {
	out := ref
	if len(ref.TypeArgs) > 0 {
		out.TypeArgs = make([]frontend.TypeRef, len(ref.TypeArgs))
		for i := range ref.TypeArgs {
			out.TypeArgs[i] = substituteTypeRef(ref.TypeArgs[i], subst)
		}
	}
	if len(ref.Params) > 0 {
		out.Params = make([]frontend.TypeRef, len(ref.Params))
		for i := range ref.Params {
			out.Params[i] = substituteTypeRef(ref.Params[i], subst)
		}
	}
	if ref.Kind == frontend.TypeRefNamed {
		if concrete := subst[ref.Name]; concrete != "" {
			return typeRefFromGenericTypeName(ref.At, concrete)
		}
		return out
	}
	if ref.Return != nil {
		ret := substituteTypeRef(*ref.Return, subst)
		out.Return = &ret
	}
	if ref.Elem != nil {
		elem := substituteTypeRef(*ref.Elem, subst)
		out.Elem = &elem
	}
	return out
}

func typeRefFromGenericTypeName(at frontend.Position, name string) frontend.TypeRef {
	if params, ret, ok := parseGenericFunctionTypeName(name); ok {
		paramRefs := make([]frontend.TypeRef, 0, len(params))
		for _, param := range params {
			paramRefs = append(paramRefs, typeRefFromGenericTypeName(at, param))
		}
		retRef := typeRefFromGenericTypeName(at, ret)
		return frontend.TypeRef{
			At:     at,
			Kind:   frontend.TypeRefFunction,
			Params: paramRefs,
			Return: &retRef,
		}
	}
	if elem, ok := optionalElemName(name); ok {
		elemRef := typeRefFromGenericTypeName(at, elem)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefOptional, Elem: &elemRef}
	}
	if elem, ok := sliceElemName(name); ok {
		elemRef := typeRefFromGenericTypeName(at, elem)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefSlice, Elem: &elemRef}
	}
	if n, elem, ok := parseArrayTypeName(name); ok {
		elemRef := typeRefFromGenericTypeName(at, elem)
		return frontend.TypeRef{At: at, Kind: frontend.TypeRefArray, Elem: &elemRef, Len: n}
	}
	return frontend.TypeRef{At: at, Kind: frontend.TypeRefNamed, Name: name}
}

func parseGenericFunctionTypeName(name string) ([]string, string, bool) {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "fn(") {
		return nil, "", false
	}
	depth := 1
	closeIdx := -1
	for i := len("fn("); i < len(name); i++ {
		switch name[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				closeIdx = i
				i = len(name)
			}
		}
	}
	if closeIdx < 0 {
		return nil, "", false
	}
	rest := strings.TrimSpace(name[closeIdx+1:])
	if !strings.HasPrefix(rest, "->") {
		return nil, "", false
	}
	paramsText := name[len("fn("):closeIdx]
	params := []string{}
	if strings.TrimSpace(paramsText) != "" {
		params = splitGenericTypeList(paramsText)
	}
	ret := strings.TrimSpace(rest[len("->"):])
	if idx := strings.Index(ret, " uses "); idx >= 0 {
		ret = strings.TrimSpace(ret[:idx])
	}
	if idx := strings.Index(ret, " throws "); idx >= 0 {
		ret = strings.TrimSpace(ret[:idx])
	}
	if ret == "" {
		return nil, "", false
	}
	return params, ret, true
}

func splitGenericTypeList(text string) []string {
	var out []string
	start := 0
	parenDepth := 0
	angleDepth := 0
	bracketDepth := 0
	for i, r := range text {
		switch r {
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case ',':
			if parenDepth == 0 && angleDepth == 0 && bracketDepth == 0 {
				out = append(out, strings.TrimSpace(text[start:i]))
				start = i + len(string(r))
			}
		}
	}
	out = append(out, strings.TrimSpace(text[start:]))
	return out
}

func substituteGenericTypeName(ref frontend.TypeRef, subst map[string]string) string {
	return genericTypeName(substituteTypeRef(ref, subst))
}

func mangleGenericName(base string, order []string, subst map[string]string) string {
	return semanticsgenerics.MangleName(base, order, subst)
}

func sanitizeGenericType(tname string) string {
	return semanticsgenerics.SanitizeType(tname)
}

func unsanitizeGenericType(tname string) (string, error) {
	return semanticsgenerics.UnsanitizeType(tname)
}

func genericTypeName(ref frontend.TypeRef) string {
	return semanticsgenerics.TypeName(ref, canonicalBuiltinType)
}

const genericClosureBindingPrefix = semanticsgenerics.ClosureBindingPrefix

func genericClosureBindingKey(name string) string {
	return semanticsgenerics.ClosureBindingKey(name)
}

func isGenericClosureLiteral(closure *frontend.ClosureExpr) bool {
	return closure != nil &&
		closure.Decl != nil &&
		len(closure.Decl.TypeParams) > 0
}

func monomorphizeEnvLocals(env map[string]string) map[string]LocalInfo {
	locals := make(map[string]LocalInfo, len(env))
	for name, typeName := range env {
		if strings.HasPrefix(name, genericClosureBindingPrefix) || typeName == "" {
			continue
		}
		locals[name] = LocalInfo{TypeName: typeName}
	}
	return locals
}

func cloneStringMap(src map[string]string) map[string]string {
	return semanticsgenerics.CloneStringMap(src)
}

func cloneFunctionTypeMap(src map[string]frontend.TypeRef) map[string]frontend.TypeRef {
	return semanticsgenerics.CloneFunctionTypeMap(src)
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// ---- generics_monomorphize.go ----

func monomorphizeStmts(
	stmts []frontend.Stmt,
	env map[string]string,
	functionLocals map[string]frontend.TypeRef,
	returnType frontend.TypeRef,
	funcDecls map[string]*frontend.FuncDecl,
	structDecls map[string]*frontend.StructDecl,
	enumDecls map[string]*frontend.EnumDecl,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			if returnType.Kind == frontend.TypeRefFunction {
				if closure, ok := s.Value.(*frontend.ClosureExpr); ok && isGenericClosureLiteral(closure) {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return unsupportedGenericClosureCaptureError(pos, name)
					}
					replacement, specialized, err := monomorphizeGenericClosureLiteralValue(
						closure,
						returnType,
						"function return",
						generics,
						created,
						createdByFile,
						work,
						fileImports,
						module,
						imports,
						structCtx,
					)
					if err != nil {
						return err
					}
					if specialized {
						s.Value = replacement
					}
				}
				replacement, specialized, err := monomorphizeGenericFunctionValueExpr(
					s.Value,
					returnType,
					"function return",
					generics,
					created,
					createdByFile,
					work,
					fileImports,
					module,
					imports,
					structCtx,
				)
				if err != nil {
					return err
				}
				if specialized {
					s.Value = replacement
				}
			}
			if _, err := monomorphizeExpr(
				s.Value,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if _, err := monomorphizeExpr(
				s.Value,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.DeferStmt:
			if err := monomorphizeStmts(
				s.Body,
				cloneStringMap(env),
				cloneFunctionTypeMap(functionLocals),
				returnType,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.BreakStmt, *frontend.ContinueStmt:
		case *frontend.PrintStmt:
			if _, err := monomorphizeExpr(
				s.Value,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.ExpectStmt:
			if _, err := monomorphizeExpr(
				s.Cond,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if _, err := monomorphizeExpr(
				s.Value,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.LetStmt:
			if err := monomorphizeFunctionTypedBinding(
				s,
				env,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
			valType, err := monomorphizeExpr(
				s.Value,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			)
			if err != nil {
				return err
			}
			if s.Type.Name != "" || s.Type.Elem != nil {
				resolved, err := resolveTypeName(&s.Type, module, imports)
				if err != nil {
					return err
				}
				env[s.Name] = resolved
			} else if s.Type.Kind == frontend.TypeRefFunction {
				functionLocals[s.Name] = s.Type
				env[s.Name] = genericTypeName(s.Type)
			} else {
				env[s.Name] = valType
			}
			closureBindingKey := genericClosureBindingKey(s.Name)
			delete(env, closureBindingKey)
			if !s.Mutable {
				if closure, ok := s.Value.(*frontend.ClosureExpr); ok && isGenericClosureLiteral(closure) {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return unsupportedGenericClosureCaptureError(pos, name)
					}
					env[closureBindingKey] = qualifyName(module, closure.Name)
				}
			}
		case *frontend.AssignStmt:
			if _, err := monomorphizeExpr(
				s.Target,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				declared, exists := functionLocals[id.Name]
				if exists && declared.Kind == frontend.TypeRefFunction {
					if closure, ok := s.Value.(*frontend.ClosureExpr); ok && isGenericClosureLiteral(closure) {
						outerLocals := monomorphizeEnvLocals(env)
						if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
							return unsupportedGenericClosureCaptureError(pos, name)
						}
						replacement, specialized, err := monomorphizeGenericClosureLiteralValue(
							closure,
							declared,
							fmt.Sprintf("function-typed assignment to '%s'", id.Name),
							generics,
							created,
							createdByFile,
							work,
							fileImports,
							module,
							imports,
							structCtx,
						)
						if err != nil {
							return err
						}
						if specialized {
							s.Value = replacement
						}
					}
					replacement, specialized, err := monomorphizeGenericFunctionValueExpr(
						s.Value,
						declared,
						fmt.Sprintf("function-typed assignment to '%s'", id.Name),
						generics,
						created,
						createdByFile,
						work,
						fileImports,
						module,
						imports,
						structCtx,
					)
					if err != nil {
						return err
					}
					if specialized {
						s.Value = replacement
					}
				}
			} else if declared, targetName, ok := functionTypeForFieldAssignmentTarget(
				s.Target,
				env,
				structDecls,
			); ok && declared.Kind == frontend.TypeRefFunction {
				if closure, ok := s.Value.(*frontend.ClosureExpr); ok && isGenericClosureLiteral(closure) {
					outerLocals := monomorphizeEnvLocals(env)
					if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
						return unsupportedGenericClosureCaptureError(pos, name)
					}
					replacement, specialized, err := monomorphizeGenericClosureLiteralValue(
						closure,
						declared,
						fmt.Sprintf("function-typed assignment to '%s'", targetName),
						generics,
						created,
						createdByFile,
						work,
						fileImports,
						module,
						imports,
						structCtx,
					)
					if err != nil {
						return err
					}
					if specialized {
						s.Value = replacement
					}
				}
				replacement, specialized, err := monomorphizeGenericFunctionValueExpr(
					s.Value,
					declared,
					fmt.Sprintf("function-typed assignment to '%s'", targetName),
					generics,
					created,
					createdByFile,
					work,
					fileImports,
					module,
					imports,
					structCtx,
				)
				if err != nil {
					return err
				}
				if specialized {
					s.Value = replacement
				}
			}
			if _, err := monomorphizeExpr(
				s.Value,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.IfStmt:
			if _, err := monomorphizeExpr(
				s.Cond,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
			if err := monomorphizeStmts(
				s.Then,
				cloneStringMap(env),
				cloneFunctionTypeMap(functionLocals),
				returnType,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
			if err := monomorphizeStmts(
				s.Else,
				cloneStringMap(env),
				cloneFunctionTypeMap(functionLocals),
				returnType,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			valueType, err := monomorphizeExpr(
				s.Value,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			)
			if err != nil {
				return err
			}
			thenEnv := cloneStringMap(env)
			if elem, ok := optionalElemName(valueType); ok {
				thenEnv[s.Name] = elem
			} else {
				thenEnv[s.Name] = "i32"
			}
			if err := monomorphizeStmts(
				s.Then,
				thenEnv,
				cloneFunctionTypeMap(functionLocals),
				returnType,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
			if err := monomorphizeStmts(
				s.Else,
				cloneStringMap(env),
				cloneFunctionTypeMap(functionLocals),
				returnType,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if _, err := monomorphizeExpr(
				s.Cond,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
			if err := monomorphizeStmts(
				s.Body,
				cloneStringMap(env),
				cloneFunctionTypeMap(functionLocals),
				returnType,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			bodyEnv := cloneStringMap(env)
			if s.Iterable != nil {
				iterType, err := monomorphizeExpr(
					s.Iterable,
					env,
					funcDecls,
					structDecls,
					enumDecls,
					generics,
					created,
					createdByFile,
					work,
					fileImports,
					module,
					imports,
					structCtx,
				)
				if err != nil {
					return err
				}
				if elem, ok := monomorphizeIterableElemType(iterType); ok {
					bodyEnv[s.Name] = elem
				} else {
					bodyEnv[s.Name] = "i32"
				}
			} else {
				if _, err := monomorphizeExpr(
					s.Start,
					env,
					funcDecls,
					structDecls,
					enumDecls,
					generics,
					created,
					createdByFile,
					work,
					fileImports,
					module,
					imports,
					structCtx,
				); err != nil {
					return err
				}
				if _, err := monomorphizeExpr(
					s.End,
					env,
					funcDecls,
					structDecls,
					enumDecls,
					generics,
					created,
					createdByFile,
					work,
					fileImports,
					module,
					imports,
					structCtx,
				); err != nil {
					return err
				}
				bodyEnv[s.Name] = "i32"
			}
			if err := monomorphizeStmts(
				s.Body,
				bodyEnv,
				cloneFunctionTypeMap(functionLocals),
				returnType,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			scrutType, err := monomorphizeExpr(
				s.Value,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			)
			if err != nil {
				return err
			}
			someElemType, hasSomeElem := optionalElemName(scrutType)
			for _, c := range s.Cases {
				caseEnv := cloneStringMap(env)
				if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok && hasSomeElem {
					caseEnv[some.Name] = someElemType
				}
				if !c.Default {
					if _, err := monomorphizeExpr(
						c.Pattern,
						caseEnv,
						funcDecls,
						structDecls,
						enumDecls,
						generics,
						created,
						createdByFile,
						work,
						fileImports,
						module,
						imports,
						structCtx,
					); err != nil {
						return err
					}
				}
				if c.Guard != nil {
					if _, err := monomorphizeExpr(
						c.Guard,
						caseEnv,
						funcDecls,
						structDecls,
						enumDecls,
						generics,
						created,
						createdByFile,
						work,
						fileImports,
						module,
						imports,
						structCtx,
					); err != nil {
						return err
					}
				}
				if err := monomorphizeStmts(
					c.Body,
					caseEnv,
					cloneFunctionTypeMap(functionLocals),
					returnType,
					funcDecls,
					structDecls,
					enumDecls,
					generics,
					created,
					createdByFile,
					work,
					fileImports,
					module,
					imports,
					structCtx,
				); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := monomorphizeStmts(
				s.Body,
				env,
				functionLocals,
				returnType,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if _, err := monomorphizeExpr(
				s.Size,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
			bodyEnv := cloneStringMap(env)
			bodyEnv[s.Name] = "island"
			if err := monomorphizeStmts(
				s.Body,
				bodyEnv,
				cloneFunctionTypeMap(functionLocals),
				returnType,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		case *frontend.ExprStmt:
			if _, err := monomorphizeExpr(
				s.Expr,
				env,
				funcDecls,
				structDecls,
				enumDecls,
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func monomorphizeFunctionTypedBinding(
	stmt *frontend.LetStmt,
	env map[string]string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	if stmt.Type.Kind != frontend.TypeRefFunction {
		return nil
	}
	if closure, ok := stmt.Value.(*frontend.ClosureExpr); ok && closure != nil &&
		closure.Decl != nil &&
		len(closure.Decl.TypeParams) > 0 {
		outerLocals := monomorphizeEnvLocals(env)
		if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
			return unsupportedGenericClosureCaptureError(pos, name)
		}
		replacement, specialized, err := monomorphizeGenericClosureLiteralValue(
			closure,
			stmt.Type,
			fmt.Sprintf("function-typed local '%s'", stmt.Name),
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		)
		if err != nil || !specialized {
			return err
		}
		stmt.Value = replacement
		return nil
	}
	replacement, specialized, err := monomorphizeGenericFunctionValueExpr(
		stmt.Value,
		stmt.Type,
		fmt.Sprintf("function-typed local '%s'", stmt.Name),
		generics,
		created,
		createdByFile,
		work,
		fileImports,
		module,
		imports,
		structCtx,
	)
	if err != nil || !specialized {
		return err
	}
	stmt.Value = replacement
	return nil
}

func monomorphizeGenericClosureLiteralValue(
	closure *frontend.ClosureExpr,
	declared frontend.TypeRef,
	context string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (frontend.Expr, bool, error) {
	if declared.Kind != frontend.TypeRefFunction || closure == nil || closure.Decl == nil ||
		len(closure.Decl.TypeParams) == 0 {
		return closure, false, nil
	}
	fullOriginal := qualifyName(module, closure.Name)
	generic, ok := generics[fullOriginal]
	if !ok {
		return closure, false, nil
	}
	declaredParams, declaredReturn, _, err := functionTypeRefSignatureAndEffects(
		declared,
		module,
		imports,
	)
	if err != nil {
		return nil, false, err
	}
	if len(declaredParams) != len(generic.decl.Params) {
		return nil, false, fmt.Errorf(
			"%s: %s generic closure literal parameter count mismatch: expected %d, got %d",
			frontend.FormatPos(closure.At),
			context,
			len(declaredParams),
			len(generic.decl.Params),
		)
	}
	subst := map[string]string{}
	for i, param := range generic.decl.Params {
		if err := bindGenericType(
			param.Type,
			declaredParams[i],
			generic.decl.TypeParams,
			subst,
		); err != nil {
			return nil, false, fmt.Errorf("%s: %v", frontend.FormatPos(closure.At), err)
		}
	}
	if err := bindGenericType(
		generic.decl.ReturnType,
		declaredReturn,
		generic.decl.TypeParams,
		subst,
	); err != nil {
		return nil, false, fmt.Errorf("%s: %v", frontend.FormatPos(closure.At), err)
	}
	for _, tp := range generic.decl.TypeParams {
		if subst[tp] == "" {
			return nil, false, fmt.Errorf(
				"%s: cannot infer generic argument '%s' for %s",
				frontend.FormatPos(closure.At),
				tp,
				context,
			)
		}
	}
	if err := checkGenericProtocolBounds(
		closure.At,
		closure.Name,
		generic,
		subst,
		module,
	); err != nil {
		return nil, false, err
	}
	name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
	fullName := genericInstanceFullName(generic, name)
	clone, exists := created[fullName]
	if !exists {
		clone = cloneGenericFunc(generic.decl, name, subst)
		cloneImports := fileImports[generic.file]
		if structCtx != nil {
			if err := rewriteGenericFuncStructRefs(
				structCtx,
				clone,
				generic.module,
				cloneImports,
			); err != nil {
				return nil, false, err
			}
		}
		created[fullName] = clone
		if _, ok := createdByFile[generic.file]; !ok {
			createdByFile[generic.file] = map[string]*frontend.FuncDecl{}
		}
		createdByFile[generic.file][name] = clone
		*work = append(
			*work,
			genericWorkItem{fn: clone, module: generic.module, imports: fileImports[generic.file]},
		)
	}
	return &frontend.ClosureExpr{At: closure.At, Name: name, Decl: clone}, true, nil
}

func monomorphizeGenericFunctionValueExpr(
	expr frontend.Expr,
	declared frontend.TypeRef,
	context string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (frontend.Expr, bool, error) {
	switch init := expr.(type) {
	case *frontend.IdentExpr:
		fullName, specialized, err := monomorphizeGenericFunctionValue(
			init,
			declared,
			context,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		)
		if err != nil || !specialized {
			return expr, specialized, err
		}
		init.Name = fullName
		return init, true, nil
	case *frontend.FieldAccessExpr:
		name := callbackArgumentName(init)
		if name == "" {
			return expr, false, nil
		}
		asIdent := &frontend.IdentExpr{At: init.At, Name: name}
		fullName, specialized, err := monomorphizeGenericFunctionValue(
			asIdent,
			declared,
			context,
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		)
		if err != nil || !specialized {
			return expr, specialized, err
		}
		return &frontend.IdentExpr{At: init.At, Name: fullName}, true, nil
	default:
		return expr, false, nil
	}
}

func monomorphizeGenericFunctionValue(
	init *frontend.IdentExpr,
	declared frontend.TypeRef,
	context string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) (string, bool, error) {
	if declared.Kind != frontend.TypeRefFunction {
		return "", false, nil
	}
	resolved, err := resolveCallName(init.Name, module, imports, init.At)
	if err != nil {
		return "", false, nil
	}
	generic, ok := generics[resolved]
	if !ok {
		return "", false, nil
	}
	declaredParams, declaredReturn, _, err := functionTypeRefSignatureAndEffects(
		declared,
		module,
		imports,
	)
	if err != nil {
		return "", false, err
	}
	if len(declaredParams) != len(generic.decl.Params) {
		return "", false, fmt.Errorf(
			"%s: %s generic function symbol '%s' parameter count mismatch: expected %d, got %d",
			frontend.FormatPos(init.At),
			context,
			init.Name,
			len(declaredParams),
			len(generic.decl.Params),
		)
	}
	subst := map[string]string{}
	for i, param := range generic.decl.Params {
		if err := bindGenericType(
			param.Type,
			declaredParams[i],
			generic.decl.TypeParams,
			subst,
		); err != nil {
			return "", false, fmt.Errorf("%s: %v", frontend.FormatPos(init.At), err)
		}
	}
	if err := bindGenericType(
		generic.decl.ReturnType,
		declaredReturn,
		generic.decl.TypeParams,
		subst,
	); err != nil {
		return "", false, fmt.Errorf("%s: %v", frontend.FormatPos(init.At), err)
	}
	for _, tp := range generic.decl.TypeParams {
		if subst[tp] == "" {
			return "", false, fmt.Errorf(
				"%s: cannot infer generic argument '%s' for %s",
				frontend.FormatPos(init.At),
				tp,
				context,
			)
		}
	}
	if err := checkGenericProtocolBounds(init.At, init.Name, generic, subst, module); err != nil {
		return "", false, err
	}
	name := mangleGenericName(generic.decl.Name, generic.decl.TypeParams, subst)
	fullName := genericInstanceFullName(generic, name)
	if _, exists := created[fullName]; !exists {
		clone := cloneGenericFunc(generic.decl, name, subst)
		cloneImports := fileImports[generic.file]
		if structCtx != nil {
			if err := rewriteGenericFuncStructRefs(
				structCtx,
				clone,
				generic.module,
				cloneImports,
			); err != nil {
				return "", false, err
			}
		}
		created[fullName] = clone
		if _, ok := createdByFile[generic.file]; !ok {
			createdByFile[generic.file] = map[string]*frontend.FuncDecl{}
		}
		createdByFile[generic.file][name] = clone
		*work = append(
			*work,
			genericWorkItem{fn: clone, module: generic.module, imports: fileImports[generic.file]},
		)
	}
	return fullName, true, nil
}

func monomorphizeFunctionTypedCallArgs(
	call *frontend.CallExpr,
	callee *frontend.FuncDecl,
	env map[string]string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	limit := len(call.Args)
	if len(callee.Params) < limit {
		limit = len(callee.Params)
	}
	for i := 0; i < limit; i++ {
		if callee.Params[i].Type.Kind != frontend.TypeRefFunction {
			continue
		}
		if closure, ok := call.Args[i].(*frontend.ClosureExpr); ok && closure != nil &&
			closure.Decl != nil &&
			len(closure.Decl.TypeParams) > 0 {
			outerLocals := monomorphizeEnvLocals(env)
			if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
				return unsupportedGenericClosureCallbackCaptureError(pos, name)
			}
			replacement, specialized, err := monomorphizeGenericClosureLiteralValue(
				closure,
				callee.Params[i].Type,
				fmt.Sprintf("callback argument for '%s'", call.Name),
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			)
			if err != nil {
				return err
			}
			if specialized {
				call.Args[i] = replacement
				continue
			}
		}
		replacement, specialized, err := monomorphizeGenericFunctionValueExpr(
			call.Args[i],
			callee.Params[i].Type,
			fmt.Sprintf("callback argument for '%s'", call.Name),
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		)
		if err != nil {
			return err
		}
		if specialized {
			call.Args[i] = replacement
		}
	}
	return nil
}

func resolveMonomorphizeCallName(
	name string,
	module string,
	imports map[string]string,
	funcDecls map[string]*frontend.FuncDecl,
	generics map[string]genericDef,
	pos frontend.Position,
) (string, error) {
	if _, ok := funcDecls[name]; ok {
		return name, nil
	}
	if _, ok := generics[name]; ok {
		return name, nil
	}
	resolved, err := resolveCallName(name, module, imports, pos)
	if err != nil {
		return "", err
	}
	if _, ok := funcDecls[resolved]; ok {
		return resolved, nil
	}
	if _, ok := generics[resolved]; ok {
		return resolved, nil
	}
	if module != "" && strings.Contains(name, ".") {
		moduleLocal := qualifyName(module, name)
		if _, ok := funcDecls[moduleLocal]; ok {
			return moduleLocal, nil
		}
		if _, ok := generics[moduleLocal]; ok {
			return moduleLocal, nil
		}
	}
	return resolved, nil
}

func monomorphizeConcreteReturnType(
	callee *frontend.FuncDecl,
	resolvedName string,
	module string,
	imports map[string]string,
) (string, error) {
	if callee == nil {
		return "", nil
	}
	calleeModule := module
	if callee.Name != "" {
		suffix := "." + callee.Name
		if resolvedName == callee.Name {
			calleeModule = ""
		} else if strings.HasSuffix(resolvedName, suffix) {
			calleeModule = strings.TrimSuffix(resolvedName, suffix)
		}
	}
	ret := substituteTypeRef(callee.ReturnType, nil)
	resolved, err := resolveTypeName(&ret, calleeModule, imports)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func monomorphizeFunctionTypedEnumPayloadArgs(
	call *frontend.CallExpr,
	enumDecls map[string]*frontend.EnumDecl,
	env map[string]string,
	generics map[string]genericDef,
	created map[string]*frontend.FuncDecl,
	createdByFile map[*frontend.FileAST]map[string]*frontend.FuncDecl,
	work *[]genericWorkItem,
	fileImports map[*frontend.FileAST]map[string]string,
	module string,
	imports map[string]string,
	structCtx *genericStructContext,
) error {
	parts := strings.Split(call.Name, ".")
	if len(parts) < 2 {
		return nil
	}
	caseName := parts[len(parts)-1]
	typeRef := frontend.TypeRef{
		At:   call.At,
		Kind: frontend.TypeRefNamed,
		Name: strings.Join(parts[:len(parts)-1], "."),
	}
	typeName, err := resolveTypeName(&typeRef, module, imports)
	if err != nil {
		return nil
	}
	enumDecl, ok := enumDecls[typeName]
	if !ok {
		return nil
	}
	var enumCase *frontend.EnumCaseDecl
	for i := range enumDecl.Cases {
		if enumDecl.Cases[i].Name == caseName {
			enumCase = &enumDecl.Cases[i]
			break
		}
	}
	if enumCase == nil {
		return nil
	}
	limit := len(call.Args)
	if len(enumCase.Payload) < limit {
		limit = len(enumCase.Payload)
	}
	for i := 0; i < limit; i++ {
		declared := enumCase.Payload[i]
		if declared.Kind != frontend.TypeRefFunction {
			continue
		}
		if closure, ok := call.Args[i].(*frontend.ClosureExpr); ok && closure != nil &&
			closure.Decl != nil &&
			len(closure.Decl.TypeParams) > 0 {
			outerLocals := monomorphizeEnvLocals(env)
			if name, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, outerLocals)); ok {
				return unsupportedGenericClosureCaptureError(pos, name)
			}
			replacement, specialized, err := monomorphizeGenericClosureLiteralValue(
				closure,
				declared,
				fmt.Sprintf("enum payload '%s.%s[%d]'", typeRef.Name, caseName, i+1),
				generics,
				created,
				createdByFile,
				work,
				fileImports,
				module,
				imports,
				structCtx,
			)
			if err != nil {
				return err
			}
			if specialized {
				call.Args[i] = replacement
				continue
			}
		}
		replacement, specialized, err := monomorphizeGenericFunctionValueExpr(
			call.Args[i],
			declared,
			fmt.Sprintf("enum payload '%s.%s[%d]'", typeRef.Name, caseName, i+1),
			generics,
			created,
			createdByFile,
			work,
			fileImports,
			module,
			imports,
			structCtx,
		)
		if err != nil {
			return err
		}
		if specialized {
			call.Args[i] = replacement
		}
	}
	return nil
}

func monomorphizeEnumConstructorTypeName(
	call *frontend.CallExpr,
	enumDecls map[string]*frontend.EnumDecl,
	module string,
	imports map[string]string,
) (string, bool) {
	parts := strings.Split(call.Name, ".")
	if len(parts) < 2 {
		return "", false
	}
	caseName := parts[len(parts)-1]
	typeRef := frontend.TypeRef{
		At:   call.At,
		Kind: frontend.TypeRefNamed,
		Name: strings.Join(parts[:len(parts)-1], "."),
	}
	typeName, err := resolveTypeName(&typeRef, module, imports)
	if err != nil {
		return "", false
	}
	enumDecl, ok := enumDecls[typeName]
	if !ok {
		return "", false
	}
	for i := range enumDecl.Cases {
		if enumDecl.Cases[i].Name == caseName {
			return typeName, true
		}
	}
	return "", false
}

func functionTypeForFieldAssignmentTarget(
	expr frontend.Expr,
	env map[string]string,
	structDecls map[string]*frontend.StructDecl,
) (frontend.TypeRef, string, bool) {
	return functionTypeForFieldAssignmentTargetFrom(expr, "", env, structDecls)
}

func functionTypeRefFromFuncDecl(fn *frontend.FuncDecl) frontend.TypeRef {
	params := make([]frontend.TypeRef, 0, len(fn.Params))
	paramOwnership := make([]string, 0, len(fn.Params))
	for _, param := range fn.Params {
		params = append(params, param.Type)
		paramOwnership = append(paramOwnership, param.Ownership)
	}
	ret := fn.ReturnType
	out := frontend.TypeRef{
		At:             fn.Pos,
		Kind:           frontend.TypeRefFunction,
		Params:         params,
		ParamOwnership: paramOwnership,
		Return:         &ret,
		Uses:           append([]string(nil), fn.Uses...),
	}
	if fn.HasThrows {
		throws := fn.Throws
		out.Throws = &throws
	}
	return out
}

func functionTypeForFieldAssignmentTargetFrom(
	expr frontend.Expr,
	path string,
	env map[string]string,
	structDecls map[string]*frontend.StructDecl,
) (frontend.TypeRef, string, bool) {
	switch target := expr.(type) {
	case *frontend.IdentExpr:
		typeName := env[target.Name]
		if typeName == "" {
			return frontend.TypeRef{}, "", false
		}
		if path == "" {
			path = target.Name
		}
		return frontend.TypeRef{Kind: frontend.TypeRefNamed, Name: typeName}, path, true
	case *frontend.FieldAccessExpr:
		baseType, basePath, ok := functionTypeForFieldAssignmentTargetFrom(
			target.Base,
			path,
			env,
			structDecls,
		)
		if !ok || baseType.Name == "" {
			return frontend.TypeRef{}, "", false
		}
		decl, ok := structDecls[baseType.Name]
		if !ok {
			return frontend.TypeRef{}, "", false
		}
		for _, field := range decl.Fields {
			if field.Name != target.Field {
				continue
			}
			if basePath == "" {
				basePath = target.Field
			} else {
				basePath += "." + target.Field
			}
			return qualifyFieldAssignmentPathType(field.Type, baseType.Name), basePath, true
		}
	}
	return frontend.TypeRef{}, "", false
}

func qualifyFieldAssignmentPathType(ref frontend.TypeRef, ownerType string) frontend.TypeRef {
	if ref.Kind != frontend.TypeRefNamed || ref.Name == "" {
		return ref
	}
	if _, ok := canonicalBuiltinType(ref.Name); ok {
		return ref
	}
	if strings.Contains(ref.Name, ".") {
		return ref
	}
	if idx := strings.LastIndex(ownerType, "."); idx >= 0 {
		ref.Name = qualifyName(ownerType[:idx], ref.Name)
	}
	return ref
}
