package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

func matchPatternCompatible(scrutType, patternType string, types map[string]*TypeInfo) bool {
	if scrutType == patternType {
		return true
	}
	if patternType == optionalSomePatternType {
		if scrutInfo, ok := types[scrutType]; ok && scrutInfo.Kind == TypeOptional {
			return true
		}
	}
	if patternType == "none" {
		if scrutInfo, ok := types[scrutType]; ok && scrutInfo.Kind == TypeOptional {
			return true
		}
	}
	if isInt32Like(scrutType) && isInt32Like(patternType) {
		return true
	}
	scrutInfo, scrutOK := types[scrutType]
	patternInfo, patternOK := types[patternType]
	if scrutOK && patternOK && (scrutInfo.Kind == TypeEnum || patternInfo.Kind == TypeEnum) {
		return false
	}
	return false
}

func validateIfLetPattern(
	pattern frontend.Expr,
	valueType string,
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
	valueInfo, ok := types[valueType]
	if !ok {
		return fmt.Errorf("unknown type '%s'", valueType)
	}
	patType := ""
	if some, ok := pattern.(*frontend.SomePatternExpr); ok {
		if valueInfo.Kind != TypeOptional {
			return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(some.At))
		}
		patType = optionalSomePatternType
	} else if enumPat, ok := pattern.(*frontend.EnumCasePatternExpr); ok {
		caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("%s: unknown enum pattern '%s.%s'", frontend.FormatPos(enumPat.At), enumPat.TypeName, enumPat.CaseName)
		}
		if err := validateEnumCasePatternPayload(enumPat, caseType, caseInfo, module); err != nil {
			return err
		}
		patType = caseType
	} else {
		var err error
		patType, _, err = checkExprWithEffects(pattern, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return err
		}
	}
	if valueInfo.Kind == TypeOptional && patType != "none" && patType != optionalSomePatternType {
		return fmt.Errorf("%s: optional if let supports only 'none', 'some(name)', and '_' patterns", frontend.FormatPos(pattern.Pos()))
	}
	if !matchPatternCompatible(valueType, patType, types) {
		return fmt.Errorf("%s: if let pattern type mismatch: expected '%s', got '%s'", frontend.FormatPos(pattern.Pos()), valueType, patType)
	}
	return nil
}

const optionalSomePatternType = "__optional_some_pattern"

func matchPatternKey(pattern frontend.Expr, patternType string) string {
	switch p := pattern.(type) {
	case *frontend.NumberExpr:
		return fmt.Sprintf("i32:%d", p.Value)
	case *frontend.NoneLitExpr:
		return "optional:none"
	case *frontend.SomePatternExpr:
		return "optional:some"
	case *frontend.EnumCasePatternExpr:
		if p.EnumType != "" {
			return "enum:" + p.EnumType + "." + p.CaseName
		}
		return "enum:" + p.TypeName + "." + p.CaseName
	case *frontend.FieldAccessExpr:
		if p.EnumType != "" {
			return "enum:" + p.EnumType + "." + p.Field
		}
		return patternType + ":" + p.Field
	default:
		return ""
	}
}

func uniqueHiddenLocal(prefix string, pos frontend.Position, locals map[string]LocalInfo) string {
	base := fmt.Sprintf("%s_%d_%d", prefix, pos.Line, pos.Col)
	name := base
	for i := 1; ; i++ {
		if _, exists := locals[name]; !exists {
			return name
		}
		name = fmt.Sprintf("%s_%d", base, i)
	}
}

func collectScopedLocal(
	name string,
	info LocalInfo,
	pos frontend.Position,
	locals map[string]LocalInfo,
	slotIndex *int,
	scopes *scopeInfo,
) error {
	if existing, exists := locals[name]; exists {
		if scopes == nil {
			return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(pos), name)
		}
		currentScope := scopes.currentScopeID()
		existingScope := scopes.localScopes[name]
		if currentScope == regionNone || existingScope == regionNone || currentScope == existingScope || !localInfosShareScopedStorage(existing, info) {
			return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(pos), name)
		}
		ids := scopes.localScopeSets[name]
		if ids == nil {
			ids = make(map[int]struct{}, 2)
			scopes.localScopeSets[name] = ids
		}
		ids[existingScope] = struct{}{}
		ids[currentScope] = struct{}{}
		scopes.localScopes[name] = currentScope
		return nil
	}
	locals[name] = info
	if scopes != nil {
		scopes.localScopes[name] = scopes.currentScopeID()
	}
	*slotIndex += info.SlotCount
	return nil
}

func localInfosShareScopedStorage(left, right LocalInfo) bool {
	return left.SlotCount == right.SlotCount &&
		left.TypeName == right.TypeName &&
		left.FunctionTypeValue == right.FunctionTypeValue &&
		left.FunctionReturnType == right.FunctionReturnType &&
		left.FunctionReturnOwnership == right.FunctionReturnOwnership &&
		strings.Join(left.FunctionParamTypes, "\x00") == strings.Join(right.FunctionParamTypes, "\x00")
}

func collectionElementType(typeName string, types map[string]*TypeInfo) (string, error) {
	info, err := ensureTypeInfo(typeName, types)
	if err != nil {
		return "", err
	}
	switch info.Kind {
	case TypeStr:
		return "u8", nil
	case TypeSlice:
		return info.ElemType, nil
	case TypeArray:
		return info.ElemType, nil
	default:
		return "", fmt.Errorf("for collection requires array, slice, or string, got '%s'", typeName)
	}
}

func collectExprLocals(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	slotIndex *int,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scopes *scopeInfo,
	globals map[string]GlobalInfo,
) error {
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		if err := collectExprLocals(e.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
			return err
		}
		scrutType, err := inferExprTypeForDecl(e.Value, locals, globals, funcs, types, module, imports)
		if err != nil {
			return fmt.Errorf("%s: cannot infer match value type: %v", frontend.FormatPos(e.At), err)
		}
		scrutInfo, err := ensureTypeInfo(scrutType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		if scrutInfo.SlotCount != 1 && scrutInfo.Kind != TypeOptional && scrutInfo.Kind != TypeEnum {
			return fmt.Errorf("%s: match value must be single-slot", frontend.FormatPos(e.At))
		}
		resultType, err := inferMatchExprType(e, locals, globals, funcs, types, module, imports)
		if err != nil {
			return fmt.Errorf("%s: cannot infer match expression type: %v", frontend.FormatPos(e.At), err)
		}
		resultInfo, err := ensureTypeInfo(resultType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		e.ScrutineeLocal = uniqueHiddenLocal("__match_expr_value", e.At, locals)
		locals[e.ScrutineeLocal] = LocalInfo{Base: *slotIndex, SlotCount: scrutInfo.SlotCount, TypeName: scrutType, Mutable: false}
		if scopes != nil {
			scopes.localScopes[e.ScrutineeLocal] = scopes.currentScopeID()
		}
		*slotIndex += scrutInfo.SlotCount
		e.ResultLocal = uniqueHiddenLocal("__match_expr_result", e.At, locals)
		e.ResultType = resultType
		locals[e.ResultLocal] = LocalInfo{Base: *slotIndex, SlotCount: resultInfo.SlotCount, TypeName: resultType, Mutable: false}
		if scopes != nil {
			scopes.localScopes[e.ResultLocal] = scopes.currentScopeID()
		}
		*slotIndex += resultInfo.SlotCount
		caseScopeIDs := make([]int, len(e.Cases))
		for i, c := range e.Cases {
			if scopes != nil {
				caseScopeIDs[i] = scopes.enterScope()
			} else {
				caseScopeIDs[i] = regionNone
			}
			if !c.Default {
				if err := collectPatternLocals(c.Pattern, scrutType, locals, slotIndex, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
			if c.Guard != nil {
				if err := collectExprLocals(c.Guard, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
			if err := collectExprLocals(c.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		}
		if scopes != nil {
			scopes.matchExprScopes[e] = caseScopeIDs
		}
	case *frontend.CatchExpr:
		if err := collectExprLocals(e.Call, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
			return err
		}
		resultType, err := inferCatchExprType(e, locals, globals, funcs, types, module, imports)
		if err != nil {
			return fmt.Errorf("%s: cannot infer catch expression type: %v", frontend.FormatPos(e.At), err)
		}
		resultInfo, err := ensureTypeInfo(resultType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		errorInfo, err := ensureTypeInfo(e.ErrorType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		e.ErrorLocal = uniqueHiddenLocal("__catch_error", e.At, locals)
		locals[e.ErrorLocal] = LocalInfo{Base: *slotIndex, SlotCount: errorInfo.SlotCount, TypeName: e.ErrorType, Mutable: false}
		if scopes != nil {
			scopes.localScopes[e.ErrorLocal] = scopes.currentScopeID()
		}
		*slotIndex += errorInfo.SlotCount
		e.ResultLocal = uniqueHiddenLocal("__catch_result", e.At, locals)
		e.ResultType = resultType
		locals[e.ResultLocal] = LocalInfo{Base: *slotIndex, SlotCount: resultInfo.SlotCount, TypeName: resultType, Mutable: false}
		if scopes != nil {
			scopes.localScopes[e.ResultLocal] = scopes.currentScopeID()
		}
		*slotIndex += resultInfo.SlotCount
		caseScopeIDs := make([]int, len(e.Cases))
		for i, c := range e.Cases {
			if scopes != nil {
				caseScopeIDs[i] = scopes.enterScope()
			} else {
				caseScopeIDs[i] = regionNone
			}
			if !c.Default {
				if err := collectPatternLocals(c.Pattern, e.ErrorType, locals, slotIndex, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
			if c.Guard != nil {
				if err := collectExprLocals(c.Guard, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
			if err := collectExprLocals(c.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		}
		if scopes != nil {
			scopes.catchExprScopes[e] = caseScopeIDs
		}
	case *frontend.UnaryExpr:
		return collectExprLocals(e.X, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	case *frontend.BinaryExpr:
		if err := collectExprLocals(e.Left, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
			return err
		}
		return collectExprLocals(e.Right, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	case *frontend.CallExpr:
		if enumType, caseInfo, ok, err := resolveEnumCaseConstructorCall(e, types, module, imports); err != nil {
			return err
		} else if ok {
			for i, arg := range e.Args {
				if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
					continue
				}
				if closure, ok := arg.(*frontend.ClosureExpr); ok {
					label := fmt.Sprintf("%s.%s[%d]", displayTypeName(enumType, module), caseInfo.Name, i+1)
					if err := configureClosureCaptures(closure, locals, funcs, types, module, true, functionTypedClosureCaptureBoundaryPhrase("function-typed assignment", label)); err != nil {
						return err
					}
				}
			}
		}
		if len(e.ArgLabels) == len(e.Args) {
			allLabeled := len(e.Args) > 0
			byLabel := make(map[string]frontend.Expr, len(e.Args))
			for i, label := range e.ArgLabels {
				if label == "" {
					allLabeled = false
					break
				}
				byLabel[label] = e.Args[i]
			}
			if allLabeled {
				typeRef := frontend.TypeRef{At: e.At, Kind: frontend.TypeRefNamed, Name: e.Name}
				if typeName, err := resolveTypeName(&typeRef, module, imports); err == nil {
					if info, ok := types[typeName]; ok && info.Kind == TypeStruct {
						for _, field := range info.Fields {
							if !field.FunctionTypeValue {
								continue
							}
							arg, ok := byLabel[field.Name]
							if !ok {
								continue
							}
							if closure, ok := arg.(*frontend.ClosureExpr); ok {
								label := fmt.Sprintf("%s.%s", displayTypeName(typeName, module), field.Name)
								if err := configureClosureCaptures(closure, locals, funcs, types, module, true, functionTypedClosureCaptureBoundaryPhrase("function-typed assignment", label)); err != nil {
									return err
								}
							}
						}
					}
				}
			}
		}
		for _, arg := range e.Args {
			if err := collectExprLocals(arg, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		}
	case *frontend.StructLitExpr:
		typeName, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return err
		}
		if info, ok := types[typeName]; ok && info.Kind == TypeStruct {
			fieldsByName := map[string]FieldInfo{}
			for _, field := range info.Fields {
				fieldsByName[field.Name] = field
			}
			for _, field := range e.Fields {
				fieldInfo, ok := fieldsByName[field.Name]
				if !ok || !fieldInfo.FunctionTypeValue {
					continue
				}
				if closure, ok := field.Value.(*frontend.ClosureExpr); ok {
					label := fmt.Sprintf("%s.%s", displayTypeName(typeName, module), fieldInfo.Name)
					if err := configureClosureCaptures(closure, locals, funcs, types, module, true, functionTypedClosureCaptureBoundaryPhrase("function-typed assignment", label)); err != nil {
						return err
					}
				}
			}
		}
		for _, field := range e.Fields {
			if err := collectExprLocals(field.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		}
	case *frontend.FieldAccessExpr:
		if e.Base != nil {
			return collectExprLocals(e.Base, locals, slotIndex, funcs, types, module, imports, scopes, globals)
		}
	case *frontend.IndexExpr:
		if err := collectExprLocals(e.Base, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
			return err
		}
		return collectExprLocals(e.Index, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	case *frontend.TryExpr:
		return collectExprLocals(e.X, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	case *frontend.AwaitExpr:
		return collectExprLocals(e.X, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	}
	return nil
}

func collectPatternLocals(
	pattern frontend.Expr,
	scrutType string,
	locals map[string]LocalInfo,
	slotIndex *int,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scopes *scopeInfo,
	globals map[string]GlobalInfo,
) error {
	info, err := ensureTypeInfo(scrutType, types)
	if err != nil {
		return err
	}
	switch pat := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(pat.At))
		}
		if _, exists := globals[pat.Name]; exists {
			return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(pat.At), pat.Name, pat.Name)
		}
		elemInfo, err := ensureTypeInfo(info.ElemType, types)
		if err != nil {
			return err
		}
		if err := collectScopedLocal(pat.Name, LocalInfo{Base: *slotIndex, SlotCount: elemInfo.SlotCount, TypeName: info.ElemType, Mutable: false}, pat.At, locals, slotIndex, scopes); err != nil {
			return err
		}
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return fmt.Errorf("%s: enum pattern type mismatch", frontend.FormatPos(pat.At))
		}
		if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
			return err
		}
		for i, binding := range pat.Bindings {
			if _, exists := globals[binding]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(pat.At), binding, binding)
			}
			localInfo := LocalInfo{Base: *slotIndex, SlotCount: caseInfo.PayloadSlots[i], TypeName: caseInfo.PayloadTypes[i], Mutable: false}
			if i < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[i] {
				localInfo = functionLocalInfoForEnumPayload(caseInfo, i, FunctionFieldInfo{})
				localInfo.Base = *slotIndex
				localInfo.SlotCount = caseInfo.PayloadSlots[i]
				localInfo.TypeName = caseInfo.PayloadTypes[i]
			}
			if err := collectScopedLocal(binding, localInfo, pat.At, locals, slotIndex, scopes); err != nil {
				return err
			}
		}
	}
	return nil
}

func collectLocals(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	slotIndex *int,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scopes *scopeInfo,
	globals map[string]GlobalInfo,
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			resolved := ""
			if s.Type.Kind == frontend.TypeRefNamed && s.Type.Name == "" {
				inferred, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
				if err != nil {
					return fmt.Errorf("%s: cannot infer type for '%s': %v", frontend.FormatPos(s.At), s.Name, err)
				}
				resolved = inferred
				s.Type = frontend.TypeRef{At: s.At, Kind: frontend.TypeRefNamed, Name: inferred}
			} else {
				var err error
				resolved, err = resolveTypeName(&s.Type, module, imports)
				if err != nil {
					return err
				}
				s.Type.Name = resolved
			}
			info, err := ensureTypeInfo(resolved, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			functionValue := ""
			genericFunctionValue := false
			var functionCaptures []frontend.ClosureCapture
			var functionEscapeCaptures []frontend.ClosureCapture
			functionTouchesMutableGlobals := false
			functionReturnSnapshotAlias := false
			functionDirectSnapshotAlias := false
			functionEscapeKind := CallableEscapeKind("")
			functionHandleValue := false
			functionParamName := ""
			functionTypeValue := s.Type.Kind == frontend.TypeRefFunction
			functionParamTypes := []string(nil)
			functionParamOwnership := []string(nil)
			functionReturnType := ""
			functionReturnOwnership := ""
			functionThrowsType := ""
			functionEffects := []string(nil)
			if functionTypeValue {
				functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(s.Type, module, imports)
				if err != nil {
					return err
				}
				functionParamOwnership = functionTypeRefParamOwnership(s.Type)
				functionReturnOwnership = functionTypeRefReturnOwnership(s.Type)
				functionThrowsType, err = functionTypeRefThrowsType(s.Type, module, imports)
				if err != nil {
					return err
				}
			}
			functionFields, err := functionFieldsFromStructLiteral(s.Name, info, s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return err
			}
			if len(functionFields) == 0 {
				functionFields = functionFieldsFromStructAlias(s.Value, locals)
			}
			if len(functionFields) == 0 {
				functionFields, err = functionFieldsFromReturnCall(s.Value, locals, globals, funcs, types, module, imports, resolved)
				if err != nil {
					return err
				}
			}
			if len(functionFields) == 0 {
				functionFields = declaredFunctionFieldsForStructType(resolved, types)
			}
			enumPayloadFields, err := enumPayloadFieldsFromStructLiteral(info, s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return err
			}
			if len(enumPayloadFields) == 0 {
				enumPayloadFields = enumPayloadFieldsFromStructAlias(s.Value, locals)
			}
			if len(enumPayloadFields) == 0 {
				enumPayloadFields, err = enumPayloadFieldsFromReturnCall(s.Value, locals, globals, funcs, types, module, imports, resolved)
				if err != nil {
					return err
				}
			}
			enumPayloadFunctions, err := enumPayloadFunctionsFromConstructor(info, s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return err
			}
			if len(enumPayloadFunctions) == 0 {
				enumPayloadFunctions = enumPayloadFunctionsFromAlias(s.Value, locals)
			}
			if len(enumPayloadFunctions) == 0 {
				enumPayloadFunctions, err = enumPayloadFunctionsFromReturnCall(s.Value, locals, globals, funcs, types, module, imports, resolved)
				if err != nil {
					return err
				}
			}
			if s.Mutable {
				enumPayloadFunctions = nil
			}
			if closure, ok := s.Value.(*frontend.ClosureExpr); ok {
				if functionTypeValue {
					if err := validateFunctionTypeLiteralBinding(s.Name, s.Type, closure, locals, module, imports); err != nil {
						return err
					}
				}
				captureBoundary := ""
				if functionTypeValue {
					captureBoundary = functionTypedClosureCaptureBoundaryPhrase("function-typed assignment", s.Name)
				}
				if err := configureClosureCaptures(closure, locals, funcs, types, module, functionTypeValue, captureBoundary); err != nil {
					return err
				}
				if functionTypeValue && len(closure.Captures) > 0 {
					captureSlots, err := functionCaptureSlotCount(closure.Captures, types)
					if err != nil {
						return err
					}
					if captureSlots > FnPtrEnvSlotCount {
						escapeKind, handleValue, err := classifyCallableEscape(callableBoundaryLocal, closure.Captures, types)
						if err != nil {
							return err
						}
						functionEscapeKind = escapeKind
						functionHandleValue = handleValue
					}
				}
				functionValue = qualifyName(module, closure.Name)
				genericFunctionValue = len(closure.Decl.TypeParams) > 0
				functionCaptures = append([]frontend.ClosureCapture(nil), closure.Captures...)
				functionDirectSnapshotAlias = len(closure.Captures) > 0
			} else if functionTypeValue {
				switch init := s.Value.(type) {
				case *frontend.IdentExpr:
					resolved, err := validateFunctionTypeNamedSymbolBinding(s.Name, s.Type, init, locals, globals, funcs, types, module, imports, true)
					if err != nil {
						return err
					}
					functionValue = resolved
					if source, ok := locals[init.Name]; ok {
						if source.FunctionParamName != "" {
							functionParamName = source.FunctionParamName
						} else if source.FunctionTypeValue && source.FunctionValue == "" {
							functionParamName = init.Name
						}
						if len(source.FunctionCaptures) > 0 || len(source.FunctionEscapeCaptures) > 0 {
							functionCaptures = append([]frontend.ClosureCapture(nil), source.FunctionCaptures...)
							functionEscapeCaptures = append([]frontend.ClosureCapture(nil), source.FunctionEscapeCaptures...)
						}
						functionDirectSnapshotAlias = source.FunctionDirectSnapshotAlias
						functionEscapeKind = source.FunctionEscapeKind
						functionHandleValue = source.FunctionHandleValue
						if len(functionCaptures) > 0 && !functionHandleValue {
							captureSlots, err := functionCaptureSlotCount(functionCaptures, types)
							if err != nil {
								return err
							}
							if captureSlots > FnPtrEnvSlotCount {
								escapeKind, handleValue, err := classifyCallableEscape(callableBoundaryLocal, functionCaptures, types)
								if err != nil {
									return err
								}
								functionEscapeKind = escapeKind
								functionHandleValue = handleValue
							}
						}
					}
				case *frontend.FieldAccessExpr:
					fieldInfo, ok, err := resolveFunctionFieldArgument(init, locals)
					if err != nil {
						return err
					}
					if ok && fieldInfo.FunctionValue == "" && functionFieldInfoHasTargetSet(fieldInfo) {
						targetInfo := LocalInfo{
							FunctionTypeValue:       true,
							FunctionParamTypes:      append([]string(nil), functionParamTypes...),
							FunctionParamOwnership:  append([]string(nil), functionParamOwnership...),
							FunctionReturnType:      functionReturnType,
							FunctionReturnOwnership: functionReturnOwnership,
							FunctionThrowsType:      functionThrowsType,
							FunctionEffects:         append([]string(nil), functionEffects...),
						}
						if err := validateFunctionInfoAssignable(s.Name, targetInfo, functionFieldInfoSig(fieldInfo), init.At); err != nil {
							return err
						}
						functionParamName = fieldInfo.FunctionParamName
						functionCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...)
						functionEscapeCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionEscapeCaptures...)
						functionDirectSnapshotAlias = fieldInfo.FunctionDirectSnapshotAlias
						break
					}
					if !ok || fieldInfo.FunctionValue == "" {
						if globalInfo, globalSig, globalOK, err := resolveFunctionTypedGlobalFieldAccess(init, globals, funcs); err != nil {
							return err
						} else if globalOK {
							if globalSig.Generic {
								return unsupportedGenericFunctionTypedLocalInitializerError(init.At, callbackArgumentName(init), s.Name)
							}
							if err := validateFunctionTypeSymbolSignature(s.Name, s.Type, globalSig, module, imports, init.At); err != nil {
								return err
							}
							functionValue = globalInfo.FunctionValue
							break
						}
						return unsupportedFunctionTypedLocalInitializerSourceError(init.At, s.Name)
					}
					targetSig, ok := funcs[fieldInfo.FunctionValue]
					if !ok {
						return fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(init.At), fieldInfo.FunctionValue)
					}
					if targetSig.Generic {
						return unsupportedGenericFunctionTypedLocalInitializerError(init.At, callbackArgumentName(init), s.Name)
					}
					if err := validateFunctionTypeSymbolSignature(s.Name, s.Type, functionFieldInfoSig(fieldInfo), module, imports, init.At); err != nil {
						return err
					}
					functionValue = fieldInfo.FunctionValue
					functionParamName = fieldInfo.FunctionParamName
					functionCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...)
					functionEscapeCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionEscapeCaptures...)
					functionDirectSnapshotAlias = fieldInfo.FunctionDirectSnapshotAlias
					functionEscapeKind = fieldInfo.FunctionEscapeKind
					functionHandleValue = fieldInfo.FunctionHandleValue
				case *frontend.CallExpr:
					resolved, err := validateFunctionTypeReturnCallBinding(s.Name, s.Type, init, funcs, module, imports)
					if err != nil {
						return err
					}
					functionValue = resolved
					metadataValue, metadataCaptures, metadataEscapeCaptures, metadataParamName, err := functionAssignmentMetadataWithReturnParamRefs(init, locals, globals, funcs, types, module, imports)
					if err != nil {
						return err
					}
					if metadataValue != "" {
						functionValue = metadataValue
					}
					functionParamName = metadataParamName
					functionCaptures = append([]frontend.ClosureCapture(nil), metadataCaptures...)
					functionEscapeCaptures = append([]frontend.ClosureCapture(nil), metadataEscapeCaptures...)
					functionReturnSnapshotAlias = isFunctionReturnSnapshotAlias(init, funcs, metadataCaptures, metadataEscapeCaptures, metadataParamName)
					functionDirectSnapshotAlias = false
					if callSig, ok := funcs[init.Name]; ok && callSig.ReturnFunctionHandleValue {
						functionEscapeKind = callSig.ReturnFunctionEscapeKind
						functionHandleValue = callSig.ReturnFunctionHandleValue
					}
				default:
					return unsupportedFunctionTypedLocalInitializerSourceError(s.At, s.Name)
				}
				functionTouchesMutableGlobals, err = functionAssignmentValueTouchesMutableGlobals(s.Value, locals, globals, funcs, types, module, imports)
				if err != nil {
					return err
				}
			}
			localSlotCount := info.SlotCount
			if functionTypeValue && functionHandleValue {
				localSlotCount = CallableHandleSlotCount
			}
			surfaceFramePixelsSource := ""
			if source, ok := surfaceFramePixelsSourceExpr(s.Value, locals, globals, types, nil); ok {
				surfaceFramePixelsSource = source
			}
			locals[s.Name] = LocalInfo{
				Base:                          *slotIndex,
				SlotCount:                     localSlotCount,
				TypeName:                      resolved,
				Mutable:                       s.Mutable,
				Const:                         s.Const,
				FunctionValue:                 functionValue,
				FunctionParamName:             functionParamName,
				GenericFunctionValue:          genericFunctionValue,
				FunctionCaptures:              functionCaptures,
				FunctionEscapeCaptures:        functionEscapeCaptures,
				FunctionTouchesMutableGlobals: functionTouchesMutableGlobals,
				FunctionReturnSnapshotAlias:   functionReturnSnapshotAlias,
				FunctionDirectSnapshotAlias:   functionDirectSnapshotAlias,
				FunctionEscapeKind:            functionEscapeKind,
				FunctionHandleValue:           functionHandleValue,
				FunctionTypeValue:             functionTypeValue,
				FunctionParamTypes:            functionParamTypes,
				FunctionParamOwnership:        functionParamOwnership,
				FunctionReturnType:            functionReturnType,
				FunctionReturnOwnership:       functionReturnOwnership,
				FunctionThrowsType:            functionThrowsType,
				FunctionEffects:               functionEffects,
				FunctionFields:                functionFields,
				EnumPayloadFunctions:          enumPayloadFunctions,
				EnumPayloadFields:             enumPayloadFields,
				SurfaceFramePixelsSource:      surfaceFramePixelsSource,
			}
			if scopes != nil {
				scopes.localScopes[s.Name] = scopes.currentScopeID()
			}
			*slotIndex += localSlotCount
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			islandInfo, err := ensureTypeInfo("island", types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
				scopes.localScopes[s.Name] = scopeID
				scopes.islandScopes[s.Name] = scopeID
			}
			locals[s.Name] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: islandInfo.SlotCount,
				TypeName:  "island",
				Mutable:   false,
			}
			*slotIndex += islandInfo.SlotCount
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		case *frontend.IfStmt:
			if err := collectExprLocals(s.Cond, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			thenScopeID := regionNone
			elseScopeID := regionNone
			if scopes != nil {
				thenScopeID = scopes.enterScope()
			}
			if err := collectLocals(s.Then, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
			if len(s.Else) > 0 {
				if scopes != nil {
					elseScopeID = scopes.enterScope()
				}
				if err := collectLocals(s.Else, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
				if scopes != nil {
					scopes.exitScope()
				}
			}
			if scopes != nil {
				scopes.ifScopes[s] = branchScopeInfo{thenID: thenScopeID, elseID: elseScopeID}
			}
		case *frontend.IfLetStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			valueType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return fmt.Errorf("%s: cannot infer if-let value type: %v", frontend.FormatPos(s.At), err)
			}
			valueInfo, err := ensureTypeInfo(valueType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if s.Pattern == nil && valueInfo.Kind != TypeOptional {
				return fmt.Errorf("%s: if let requires optional value, got '%s'", frontend.FormatPos(s.At), valueType)
			}
			if s.Pattern != nil && valueInfo.Kind != TypeOptional && valueInfo.Kind != TypeEnum {
				return fmt.Errorf("%s: if let pattern requires optional or enum value, got '%s'", frontend.FormatPos(s.At), valueType)
			}
			if s.Pattern == nil {
				if _, exists := globals[s.Name]; exists {
					return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
				}
			}
			elemInfo, err := ensureTypeInfo(valueInfo.ElemType, types)
			if s.Pattern == nil && err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			s.ValueLocal = uniqueHiddenLocal("__iflet_value", s.At, locals)
			locals[s.ValueLocal] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: valueInfo.SlotCount,
				TypeName:  valueType,
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.ValueLocal] = scopes.currentScopeID()
			}
			*slotIndex += valueInfo.SlotCount
			thenScopeID := regionNone
			elseScopeID := regionNone
			if scopes != nil {
				thenScopeID = scopes.enterScope()
			}
			if s.Pattern == nil {
				if err := collectScopedLocal(s.Name, LocalInfo{
					Base:      *slotIndex,
					SlotCount: elemInfo.SlotCount,
					TypeName:  valueInfo.ElemType,
					Mutable:   false,
				}, s.At, locals, slotIndex, scopes); err != nil {
					return err
				}
			} else if err := collectPatternLocals(s.Pattern, valueType, locals, slotIndex, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if err := collectLocals(s.Then, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
			if len(s.Else) > 0 {
				if scopes != nil {
					elseScopeID = scopes.enterScope()
				}
				if err := collectLocals(s.Else, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
				if scopes != nil {
					scopes.exitScope()
				}
			}
			if scopes != nil {
				scopes.ifLetScopes[s] = branchScopeInfo{thenID: thenScopeID, elseID: elseScopeID}
			}
		case *frontend.WhileStmt:
			if err := collectExprLocals(s.Cond, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.whileScopes[s] = scopeID
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				if err := collectExprLocals(s.Iterable, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
			} else {
				if err := collectExprLocals(s.Start, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
				if err := collectExprLocals(s.End, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			loopType := "i32"
			var iterableInfo *TypeInfo
			if s.Iterable != nil {
				iterType, err := inferExprTypeForDecl(s.Iterable, locals, globals, funcs, types, module, imports)
				if err != nil {
					return fmt.Errorf("%s: cannot infer for collection type: %v", frontend.FormatPos(s.At), err)
				}
				elemType, err := collectionElementType(iterType, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
				}
				iterableInfo, err = ensureTypeInfo(iterType, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
				}
				loopType = elemType
			}
			info, err := ensureTypeInfo(loopType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			locals[s.Name] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: info.SlotCount,
				TypeName:  loopType,
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.Name] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
			if s.Iterable != nil {
				s.IterableLocal = uniqueHiddenLocal("__for_iter", s.At, locals)
				locals[s.IterableLocal] = LocalInfo{
					Base:      *slotIndex,
					SlotCount: iterableInfo.SlotCount,
					TypeName:  iterableInfo.Name,
					Mutable:   false,
				}
				if scopes != nil {
					scopes.localScopes[s.IterableLocal] = scopes.currentScopeID()
				}
				*slotIndex += iterableInfo.SlotCount
				s.IndexLocal = uniqueHiddenLocal("__for_index", s.At, locals)
				indexInfo := types["i32"]
				locals[s.IndexLocal] = LocalInfo{
					Base:      *slotIndex,
					SlotCount: indexInfo.SlotCount,
					TypeName:  "i32",
					Mutable:   false,
				}
				if scopes != nil {
					scopes.localScopes[s.IndexLocal] = scopes.currentScopeID()
				}
				*slotIndex += indexInfo.SlotCount
			}
			s.EndLocal = uniqueHiddenLocal("__for_end", s.At, locals)
			locals[s.EndLocal] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: 1,
				TypeName:  "i32",
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.EndLocal] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.forScopes[s] = scopeID
			}
		case *frontend.MatchStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			scrutType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return fmt.Errorf("%s: cannot infer match value type: %v", frontend.FormatPos(s.At), err)
			}
			info, err := ensureTypeInfo(scrutType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if info.SlotCount != 1 && info.Kind != TypeOptional && info.Kind != TypeEnum {
				return fmt.Errorf("%s: match value must be single-slot", frontend.FormatPos(s.At))
			}
			s.ScrutineeLocal = uniqueHiddenLocal("__match_value", s.At, locals)
			locals[s.ScrutineeLocal] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: info.SlotCount,
				TypeName:  scrutType,
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.ScrutineeLocal] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
			caseScopeIDs := make([]int, len(s.Cases))
			scrutineeFunctionPayloads, err := enumPayloadFunctionValuesForMatchExpr(s.Value, locals, globals, funcs, types, module, imports, scrutType)
			if err != nil {
				return err
			}
			for i, c := range s.Cases {
				if scopes != nil {
					caseScopeIDs[i] = scopes.enterScope()
				} else {
					caseScopeIDs[i] = regionNone
				}
				if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
					if info.Kind != TypeOptional {
						return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(some.At))
					}
					if _, exists := globals[some.Name]; exists {
						return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(some.At), some.Name, some.Name)
					}
					elemInfo, err := ensureTypeInfo(info.ElemType, types)
					if err != nil {
						return fmt.Errorf("%s: %v", frontend.FormatPos(some.At), err)
					}
					if err := collectScopedLocal(some.Name, LocalInfo{
						Base:      *slotIndex,
						SlotCount: elemInfo.SlotCount,
						TypeName:  info.ElemType,
						Mutable:   false,
					}, some.At, locals, slotIndex, scopes); err != nil {
						return err
					}
				}
				if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
					caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
					if err != nil {
						return err
					}
					if !found || caseType != scrutType {
						return fmt.Errorf("%s: enum pattern type mismatch", frontend.FormatPos(enumPat.At))
					}
					if err := validateEnumCasePatternPayload(enumPat, caseType, caseInfo, module); err != nil {
						return err
					}
					for j, binding := range enumPat.Bindings {
						if _, exists := globals[binding]; exists {
							return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(enumPat.At), binding, binding)
						}
						slots := 1
						if j < len(caseInfo.PayloadSlots) {
							slots = caseInfo.PayloadSlots[j]
						}
						localInfo := LocalInfo{
							Base:      *slotIndex,
							SlotCount: slots,
							TypeName:  caseInfo.PayloadTypes[j],
							Mutable:   false,
						}
						if j < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[j] {
							localInfo = functionLocalInfoForEnumPayload(caseInfo, j, FunctionFieldInfo{})
							localInfo.Base = *slotIndex
							localInfo.SlotCount = slots
							localInfo.TypeName = caseInfo.PayloadTypes[j]
						}
						if err := collectScopedLocal(binding, localInfo, enumPat.At, locals, slotIndex, scopes); err != nil {
							return err
						}
					}
					if err := bindEnumPatternFunctionPayloadLocals(enumPat, scrutineeFunctionPayloads, locals, types, module, imports); err != nil {
						return err
					}
				}
				if c.Guard != nil {
					if err := collectExprLocals(c.Guard, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
						return err
					}
				}
				if err := collectLocals(c.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
				if scopes != nil {
					scopes.exitScope()
				}
			}
			if scopes != nil {
				scopes.matchCaseScopes[s] = caseScopeIDs
			}
		case *frontend.UnsafeStmt:
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.unsafeScopes[s] = scopeID
			}
		case *frontend.DeferStmt:
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.deferScopes[s] = scopeID
			}
		case *frontend.ExprStmt:
			if err := collectExprLocals(s.Expr, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.ReturnStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.PrintStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if err := rejectRepresentationMetadataExprAssignment(s.Target, locals, globals, types); err != nil {
				return err
			}
			if err := collectExprLocals(s.Target, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if local, exists := locals[id.Name]; exists && local.Mutable {
					if local.FunctionTypeValue {
						if _, ok := s.Value.(*frontend.ClosureExpr); ok {
							_ = updateFunctionTypedLocalAssignmentMetadata(id.Name, s.Value, locals, globals, funcs, types, module, imports)
							local = locals[id.Name]
						}
					}
					info, infoOK := types[local.TypeName]
					if infoOK && info.Kind == TypeEnum {
						payloads, err := enumPayloadFunctionsFromConstructor(info, s.Value, locals, globals, funcs, types, module, imports)
						if err != nil {
							return err
						}
						if len(payloads) == 0 {
							payloads = enumPayloadFunctionsFromAlias(s.Value, locals)
						}
						if len(payloads) == 0 {
							payloads, err = enumPayloadFunctionsFromReturnCall(s.Value, locals, globals, funcs, types, module, imports, local.TypeName)
							if err != nil {
								return err
							}
						}
						local.EnumPayloadFunctions = payloads
						locals[id.Name] = local
					}
					if infoOK && info.Kind == TypeStruct {
						payloadFields, err := enumPayloadFieldsFromReturnedStructExpr(local.TypeName, s.Value, locals, globals, funcs, types, module, imports)
						if err != nil {
							return err
						}
						if len(payloadFields) > 0 || len(local.EnumPayloadFields) > 0 {
							local.EnumPayloadFields = cloneFunctionFieldMap(payloadFields)
							locals[id.Name] = local
						}
					}
				}
			} else {
				if targetName, fieldInfo, ok, err := functionFieldLocalInfoFromExpr(s.Target, locals, types); err != nil {
					return err
				} else if ok {
					if _, closure := s.Value.(*frontend.ClosureExpr); closure {
						_ = updateFunctionTypedFieldAssignmentMetadata(targetName, fieldInfo, s.Value, locals, globals, funcs, types, module, imports)
					}
				}
				targetType, err := inferExprTypeForDecl(s.Target, locals, globals, funcs, types, module, imports)
				if err != nil {
					return err
				}
				if info, ok := types[targetType]; ok && (info.Kind == TypeEnum || info.Kind == TypeStruct) {
					if err := updateEnumPayloadStructFieldAssignmentMetadata(s.Target, targetType, s.Value, locals, globals, funcs, types, module, imports); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
