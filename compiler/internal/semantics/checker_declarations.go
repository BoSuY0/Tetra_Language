package semantics

import (
	"fmt"
	"reflect"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	semanticsdeclarations "tetra_language/compiler/internal/semantics/declarations"
	semanticsflow "tetra_language/compiler/internal/semantics/flow"
	semanticsstatements "tetra_language/compiler/internal/semantics/statements"
	semanticsworld "tetra_language/compiler/internal/semantics/world"
)

func validateCapsuleDecls(file *frontend.FileAST) error {
	return semanticsdeclarations.ValidateCapsuleDecls(file)
}

func validateEnumPayloadCycles(edges map[string]map[string]frontend.Position, enumModules map[string]string) error {
	const (
		enumVisitNew = iota
		enumVisitActive
		enumVisitDone
	)
	visitState := map[string]int{}
	var dfs func(string) error
	dfs = func(name string) error {
		visitState[name] = enumVisitActive
		for target, at := range edges[name] {
			if _, ok := enumModules[target]; !ok {
				continue
			}
			switch visitState[target] {
			case enumVisitActive:
				moduleName := enumModules[name]
				if tgt, ok := enumModules[target]; ok {
					moduleName = tgt
				}
				return fmt.Errorf("%s: recursive enum payload '%s'", frontend.FormatPos(at), displayTypeName(target, moduleName))
			case enumVisitDone:
				continue
			default:
				if err := dfs(target); err != nil {
					return err
				}
			}
		}
		visitState[name] = enumVisitDone
		return nil
	}
	for name := range edges {
		if visitState[name] == enumVisitNew {
			if err := dfs(name); err != nil {
				return err
			}
		}
	}
	return nil
}

func refreshCompositeSlotLayouts(types map[string]*TypeInfo) error {
	if len(types) == 0 {
		return nil
	}
	maxPasses := len(types) + 1
	for pass := 0; pass < maxPasses; pass++ {
		changed := false
		for _, info := range types {
			if info == nil {
				continue
			}
			switch info.Kind {
			case TypeOptional:
				elemInfo, ok := types[info.ElemType]
				if !ok {
					continue
				}
				slotCount := elemInfo.SlotCount + 1
				if info.SlotCount != slotCount {
					info.SlotCount = slotCount
					changed = true
				}
			case TypeStruct:
				if len(info.Fields) == 0 {
					continue
				}
				offset := 0
				fieldMap := make(map[string]FieldInfo, len(info.Fields))
				fields := make([]FieldInfo, len(info.Fields))
				for i, field := range info.Fields {
					slotCount := field.SlotCount
					if fieldInfo, ok := types[field.TypeName]; ok {
						slotCount = fieldInfo.SlotCount
					}
					if slotCount <= 0 {
						slotCount = 1
					}
					field.Offset = offset
					field.SlotCount = slotCount
					fields[i] = field
					fieldMap[field.Name] = field
					offset += slotCount
				}
				if info.SlotCount != offset || !fieldLayoutsEqual(info.Fields, fields) {
					info.SlotCount = offset
					info.Fields = fields
					info.FieldMap = fieldMap
					changed = true
				}
			case TypeEnum:
				maxPayloadSlots := 0
				for i, caseInfo := range info.EnumCases {
					totalPayloadSlots := 0
					for j, payloadType := range caseInfo.PayloadTypes {
						slotCount := 1
						if payloadInfo, ok := types[payloadType]; ok {
							slotCount = payloadInfo.SlotCount
						}
						if slotCount <= 0 {
							slotCount = 1
						}
						if j < len(caseInfo.PayloadSlots) && caseInfo.PayloadSlots[j] != slotCount {
							caseInfo.PayloadSlots[j] = slotCount
							changed = true
						}
						totalPayloadSlots += slotCount
					}
					if caseInfo.SlotCount != totalPayloadSlots {
						caseInfo.SlotCount = totalPayloadSlots
						changed = true
					}
					info.EnumCases[i] = caseInfo
					info.CaseMap[caseInfo.Name] = caseInfo
					if totalPayloadSlots > maxPayloadSlots {
						maxPayloadSlots = totalPayloadSlots
					}
				}
				slotCount := 1 + maxPayloadSlots
				if info.SlotCount != slotCount {
					info.SlotCount = slotCount
					changed = true
				}
			}
		}
		if !changed {
			return nil
		}
	}
	return fmt.Errorf("recursive composite type layout is not supported")
}

func structReprOrDefault(repr string) string {
	if repr == "" {
		return frontend.StructReprDefault
	}
	return repr
}

func fieldLayoutsEqual(a, b []FieldInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func collectCapsulePermissionsByModule(world *module.World) map[string]map[string]struct{} {
	out := map[string]map[string]struct{}{}
	if world == nil {
		return out
	}
	for _, file := range world.Files {
		if file == nil {
			continue
		}
		moduleName := file.Module
		if _, ok := out[moduleName]; !ok {
			out[moduleName] = map[string]struct{}{}
		}
		for _, capsule := range file.Capsules {
			if capsule == nil {
				continue
			}
			for _, entry := range capsule.Entries {
				if granted, ok := capsulePermissionEntry(entry); ok && granted {
					out[moduleName][capsulePermissionFromEntryKey(entry.Key)] = struct{}{}
				}
			}
		}
	}
	return out
}

func capsulePermissionEntry(entry frontend.CapsuleEntryDecl) (bool, bool) {
	switch entry.Key {
	case "permissions.io", "permissions.mem":
		b, ok := entry.Value.(*frontend.BoolLitExpr)
		if !ok {
			return false, true
		}
		return b.Value, true
	default:
		return false, false
	}
}

func capsulePermissionFromEntryKey(key string) string {
	switch key {
	case "permissions.io":
		return "capsule.io"
	case "permissions.mem":
		return "capsule.mem"
	default:
		return ""
	}
}

func isCapsuleMetadataLiteral(expr frontend.Expr) bool {
	return semanticsdeclarations.IsCapsuleMetadataLiteral(expr)
}

func isCapsuleMetadataKey(key string) bool {
	return semanticsdeclarations.IsCapsuleMetadataKey(key)
}

func isCapsuleKeySegment(seg string) bool {
	return semanticsdeclarations.IsCapsuleKeySegment(seg)
}

func stmtListEndsWithReturn(stmts []frontend.Stmt) bool {
	return semanticsstatements.ListEndsWithReturn(stmts)
}

func injectActorStateLocals(fields map[string]ActorStateField, locals map[string]LocalInfo, scopes *scopeInfo) error {
	for name, field := range fields {
		if _, exists := locals[name]; exists {
			return fmt.Errorf("duplicate local '%s'", name)
		}
		locals[name] = LocalInfo{
			Base:           -1,
			SlotCount:      1,
			TypeName:       field.TypeName,
			Mutable:        field.Mutable,
			Const:          field.Const,
			ActorField:     true,
			ActorFieldSlot: field.Slot,
			ActorFieldInit: field.Init,
		}
		if scopes != nil && scopes.localScopes != nil {
			scopes.localScopes[name] = regionNone
		}
	}
	return nil
}

func normalizeExtensionMethodNames(world *module.World) error {
	for _, file := range world.Files {
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		for _, fn := range file.Funcs {
			if fn.ExtensionOf == "" {
				continue
			}
			targetRef := frontend.TypeRef{At: fn.Pos, Kind: frontend.TypeRefNamed, Name: fn.ExtensionOf}
			resolvedTarget, err := resolveTypeName(&targetRef, file.Module, imports)
			if err != nil {
				return err
			}
			methodName := extensionMethodNamePart(fn.Name)
			if methodName == "" {
				return fmt.Errorf("%s: invalid extension method name '%s'", frontend.FormatPos(fn.Pos), fn.Name)
			}
			fn.ExtensionOf = resolvedTarget
			fn.Name = resolvedTarget + "." + methodName
		}
	}
	return nil
}

func extensionMethodNamePart(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 && idx+1 < len(name) {
		return name[idx+1:]
	}
	return name
}

func checkedFuncFullName(module string, fn *frontend.FuncDecl) string {
	return semanticsworld.CheckedFuncFullName(module, fn)
}

func validateFunctionParamNames(fn *frontend.FuncDecl) error {
	seen := make(map[string]struct{}, len(fn.Params))
	for _, param := range fn.Params {
		if param.Name == "" {
			return fmt.Errorf("%s: parameter name required", frontend.FormatPos(param.At))
		}
		if _, exists := seen[param.Name]; exists {
			return fmt.Errorf("%s: duplicate parameter '%s'", frontend.FormatPos(param.At), param.Name)
		}
		seen[param.Name] = struct{}{}
	}
	return nil
}

func addPublicImportFunctionAliases(world *module.World, funcs map[string]FuncSig) error {
	for _, file := range world.Files {
		for _, imp := range file.Imports {
			if !imp.Public || len(imp.Items) == 0 {
				continue
			}
			for _, item := range imp.Items {
				target := qualifyName(imp.Path, item)
				sig, ok := funcs[target]
				if !ok {
					continue
				}
				if err := ensureFuncVisible(target, sig, file.Module, imp.At); err != nil {
					return err
				}
				alias := qualifyName(file.Module, item)
				if _, exists := funcs[alias]; exists && alias != target {
					return fmt.Errorf("%s: re-export '%s' conflicts with function '%s'", frontend.FormatPos(imp.At), item, alias)
				}
				sig.Public = true
				funcs[alias] = sig
			}
		}
	}
	return nil
}

func addImportedFunctionTypedGlobalAliases(world *module.World, globalsByModule map[string]map[string]GlobalInfo) error {
	if world == nil {
		return nil
	}
	for _, file := range world.Files {
		if file == nil {
			continue
		}
		globals := globalsByModule[file.Module]
		if globals == nil {
			globals = make(map[string]GlobalInfo)
			globalsByModule[file.Module] = globals
		}
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		for alias, target := range imports {
			if symbol, isSymbol := importSymbolTarget(target); isSymbol {
				dot := strings.LastIndex(symbol, ".")
				if dot < 0 || dot == 0 || dot == len(symbol)-1 {
					continue
				}
				moduleName := symbol[:dot]
				globalName := symbol[dot+1:]
				global, ok := globalsByModule[moduleName][globalName]
				if !ok || !global.Public || !global.FunctionTypeValue || global.FunctionValue == "" {
					continue
				}
				if global.Mutable {
					global.FunctionValue = ""
				}
				globals[alias] = global
				globals[symbol] = global
				continue
			}
			importedGlobals := globalsByModule[target]
			for name, global := range importedGlobals {
				if !global.Public || !global.FunctionTypeValue || global.FunctionValue == "" {
					continue
				}
				if global.Mutable {
					global.FunctionValue = ""
				}
				globals[alias+"."+name] = global
				globals[target+"."+name] = global
			}
		}
	}
	return nil
}

func initialReturnResourceParam(returnType string, types map[string]*TypeInfo) int {
	if typeContainsResourceHandle(returnType, types) {
		return regionUnknown
	}
	return regionNone
}

func cloneReturnRegionSummary(in ReturnRegionSummary) ReturnRegionSummary {
	return semanticsflow.CloneReturnRegionSummary(in)
}

func returnRegionSummariesEqual(a, b ReturnRegionSummary) bool {
	return semanticsflow.ReturnRegionSummariesEqual(a, b)
}

func cloneReturnResourceSummary(in ReturnResourceSummary) ReturnResourceSummary {
	return semanticsflow.CloneReturnResourceSummary(in)
}

func returnResourceSummariesEqual(a, b ReturnResourceSummary) bool {
	return semanticsflow.ReturnResourceSummariesEqual(a, b)
}

func stmtListEndsWithReturnTyped(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	if len(stmts) == 0 {
		return false
	}
	return stmtEndsWithReturnTyped(stmts[len(stmts)-1], locals, globals, funcs, types, module, imports)
}

func stmtEndsWithReturnTyped(
	stmt frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	switch s := stmt.(type) {
	case *frontend.ReturnStmt, *frontend.ThrowStmt:
		return true
	case *frontend.IfStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 &&
			stmtListEndsWithReturnTyped(s.Then, locals, globals, funcs, types, module, imports) &&
			stmtListEndsWithReturnTyped(s.Else, locals, globals, funcs, types, module, imports)
	case *frontend.IfLetStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 &&
			stmtListEndsWithReturnTyped(s.Then, locals, globals, funcs, types, module, imports) &&
			stmtListEndsWithReturnTyped(s.Else, locals, globals, funcs, types, module, imports)
	case *frontend.MatchStmt:
		for _, c := range s.Cases {
			if !stmtListEndsWithReturnTyped(c.Body, locals, globals, funcs, types, module, imports) {
				return false
			}
		}
		if matchHasDefault(s) || matchHasCompleteOptionalPatterns(s) {
			return true
		}
		return matchHasCompleteEnumPatterns(s, locals, globals, funcs, types, module, imports)
	case *frontend.UnsafeStmt:
		return stmtListEndsWithReturnTyped(s.Body, locals, globals, funcs, types, module, imports)
	default:
		return false
	}
}

func matchHasDefault(s *frontend.MatchStmt) bool {
	for _, c := range s.Cases {
		if c.Default && c.Guard == nil {
			return true
		}
	}
	return false
}

func matchHasCompleteOptionalPatterns(s *frontend.MatchStmt) bool {
	hasNone := false
	hasSome := false
	for _, c := range s.Cases {
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
			hasNone = true
		}
		if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			hasSome = true
		}
	}
	return hasNone && hasSome
}

func matchHasCompleteEnumPatterns(
	s *frontend.MatchStmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	scrutType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return false
	}
	info, ok := types[scrutType]
	if !ok || info.Kind != TypeEnum || len(info.EnumCases) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(info.EnumCases))
	for i := range s.Cases {
		c := &s.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		switch pat := c.Pattern.(type) {
		case *frontend.FieldAccessExpr:
			caseType, caseName, ok := bareEnumPatternTypeAndCase(pat, module, imports)
			if !ok || caseType != scrutType {
				return false
			}
			caseInfo, ok := info.CaseMap[caseName]
			if !ok || len(caseInfo.PayloadTypes) != 0 {
				return false
			}
			pat.EnumType = scrutType
			pat.EnumOrdinal = caseInfo.Ordinal
			seen[caseName] = struct{}{}
		case *frontend.EnumCasePatternExpr:
			caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
			if err != nil || !found || caseType != scrutType {
				return false
			}
			if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
				return false
			}
			seen[pat.CaseName] = struct{}{}
		default:
			return false
		}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
}

func matchExprHasCompleteOptionalPatterns(e *frontend.MatchExpr) bool {
	seenNone := false
	seenSome := false
	for _, c := range e.Cases {
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
			seenNone = true
		}
		if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			seenSome = true
		}
	}
	return seenNone && seenSome
}

func matchExprHasCompleteEnumPatterns(
	e *frontend.MatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	scrutType, err := inferExprTypeForDecl(e.Value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return false
	}
	info, ok := types[scrutType]
	if !ok || info.Kind != TypeEnum || len(info.EnumCases) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(info.EnumCases))
	for i := range e.Cases {
		c := &e.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		switch pat := c.Pattern.(type) {
		case *frontend.FieldAccessExpr:
			caseType, caseName, ok := bareEnumPatternTypeAndCase(pat, module, imports)
			if !ok || caseType != scrutType {
				return false
			}
			caseInfo, ok := info.CaseMap[caseName]
			if !ok || len(caseInfo.PayloadTypes) != 0 {
				return false
			}
			pat.EnumType = scrutType
			pat.EnumOrdinal = caseInfo.Ordinal
			seen[caseName] = struct{}{}
		case *frontend.EnumCasePatternExpr:
			caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
			if err != nil || !found || caseType != scrutType {
				return false
			}
			if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
				return false
			}
			seen[pat.CaseName] = struct{}{}
		default:
			return false
		}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
}

func inferMatchExprType(
	e *frontend.MatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, error) {
	scrutType, err := inferExprTypeForDecl(e.Value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return "", err
	}
	resultType := ""
	for _, c := range e.Cases {
		armLocals := cloneLocalMap(locals)
		if !c.Default {
			if err := bindMatchPatternLocalsForInference(c.Pattern, scrutType, armLocals, types, module, imports); err != nil {
				return "", err
			}
		}
		armType, err := inferExprTypeForDecl(c.Value, armLocals, globals, funcs, types, module, imports)
		if err != nil {
			return "", err
		}
		if resultType == "" {
			resultType = armType
			continue
		}
		if !typesCompatibleWithNullPtr(resultType, armType, c.Value) {
			return "", fmt.Errorf("match expression case type mismatch: expected '%s', got '%s'", resultType, armType)
		}
	}
	if resultType == "" {
		return "", fmt.Errorf("match expression requires at least one case")
	}
	e.ResultType = resultType
	return resultType, nil
}

func inferCatchExprType(
	e *frontend.CatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, error) {
	call, ok := e.Call.(*frontend.CallExpr)
	if !ok {
		return "", fmt.Errorf("catch expects a throwing function call")
	}
	if builtin, ok := ResolveBuiltinAlias(call.Name); ok && (builtin == "core.task_join_i32_typed" || builtin == "core.task_join_group_i32_typed") {
		if len(call.TypeArgs) != 1 {
			return "", fmt.Errorf("task_join_i32_typed expects one explicit error type argument")
		}
		errorType, err := resolveTypeName(&call.TypeArgs[0], module, imports)
		if err != nil {
			return "", err
		}
		if err := validateTypedTaskErrorType(errorType, types, call.TypeArgs[0].At); err != nil {
			return "", err
		}
		call.TypeArgs[0].Name = errorType
		e.ErrorType = errorType
		e.ResultType = "i32"
		for _, c := range e.Cases {
			armLocals := cloneLocalMap(locals)
			if !c.Default {
				if err := bindMatchPatternLocalsForInference(c.Pattern, errorType, armLocals, types, module, imports); err != nil {
					return "", err
				}
			}
			armType, err := inferExprTypeForDecl(c.Value, armLocals, globals, funcs, types, module, imports)
			if err != nil {
				return "", err
			}
			if !typesCompatibleWithNullPtr("i32", armType, c.Value) {
				return "", fmt.Errorf("catch expression case type mismatch: expected 'i32', got '%s'", armType)
			}
		}
		return "i32", nil
	}
	sig, err := resolveCallSigForInference(call, funcs, module, imports)
	if err != nil {
		return "", err
	}
	if sig.ThrowsType == "" {
		return "", fmt.Errorf("catch expects a throwing function call")
	}
	e.ErrorType = sig.ThrowsType
	e.ResultType = sig.ReturnType
	for _, c := range e.Cases {
		armLocals := cloneLocalMap(locals)
		if !c.Default {
			if err := bindMatchPatternLocalsForInference(c.Pattern, sig.ThrowsType, armLocals, types, module, imports); err != nil {
				return "", err
			}
		}
		armType, err := inferExprTypeForDecl(c.Value, armLocals, globals, funcs, types, module, imports)
		if err != nil {
			return "", err
		}
		if !typesCompatibleWithNullPtr(sig.ReturnType, armType, c.Value) {
			return "", fmt.Errorf("catch expression case type mismatch: expected '%s', got '%s'", sig.ReturnType, armType)
		}
	}
	return sig.ReturnType, nil
}

func resolveCallSigForInference(call *frontend.CallExpr, funcs map[string]FuncSig, module string, imports map[string]string) (FuncSig, error) {
	resolved := ""
	if builtin, ok := ResolveBuiltinAlias(call.Name); ok {
		resolved = builtin
	} else if _, ok := funcs[call.Name]; ok {
		resolved = call.Name
	} else {
		name, err := resolveKnownCallName(call.Name, funcs, module, imports, call.At)
		if err != nil {
			return FuncSig{}, err
		}
		resolved = name
	}
	sig, ok := funcs[resolved]
	if !ok {
		return FuncSig{}, fmt.Errorf("unknown function '%s'", resolved)
	}
	if sig.Generic {
		return FuncSig{}, fmt.Errorf("generic function '%s' could not be monomorphized; use inferable value arguments", call.Name)
	}
	return sig, nil
}

func catchPatternType(
	pattern frontend.Expr,
	errorType string,
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
	info, ok := types[errorType]
	if !ok {
		return "", fmt.Errorf("unknown type '%s'", errorType)
	}
	switch pat := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return "", fmt.Errorf("%s: some pattern requires optional catch value", frontend.FormatPos(pat.At))
		}
		return optionalSomePatternType, nil
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
		if err != nil {
			return "", err
		}
		if !found {
			return "", fmt.Errorf("%s: unknown enum pattern '%s.%s'", frontend.FormatPos(pat.At), pat.TypeName, pat.CaseName)
		}
		if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
			return "", err
		}
		return caseType, nil
	default:
		patType, _, err := checkExprWithEffects(pattern, locals, globals, funcs, types, module, imports, state, effects, analysis)
		return patType, err
	}
}

func catchExprHasCompleteOptionalPatterns(e *frontend.CatchExpr, errorType string, types map[string]*TypeInfo) bool {
	info, ok := types[errorType]
	if !ok || info.Kind != TypeOptional {
		return false
	}
	seenSome := false
	seenNone := false
	for _, c := range e.Cases {
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			seenSome = true
			continue
		}
		if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
			seenNone = true
			continue
		}
		return false
	}
	return seenSome && seenNone
}

func catchExprHasCompleteEnumPatterns(
	e *frontend.CatchExpr,
	errorType string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	info, ok := types[errorType]
	if !ok || info.Kind != TypeEnum || len(info.EnumCases) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(info.EnumCases))
	for i := range e.Cases {
		c := &e.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		switch pat := c.Pattern.(type) {
		case *frontend.FieldAccessExpr:
			caseType, caseName, ok := bareEnumPatternTypeAndCase(pat, module, imports)
			if !ok || caseType != errorType {
				return false
			}
			caseInfo, ok := info.CaseMap[caseName]
			if !ok || len(caseInfo.PayloadTypes) != 0 {
				return false
			}
			pat.EnumType = errorType
			pat.EnumOrdinal = caseInfo.Ordinal
			seen[caseName] = struct{}{}
		case *frontend.EnumCasePatternExpr:
			caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
			if err != nil || !found || caseType != errorType {
				return false
			}
			if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
				return false
			}
			seen[pat.CaseName] = struct{}{}
		default:
			return false
		}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
}

func cloneLocalMap(locals map[string]LocalInfo) map[string]LocalInfo {
	out := make(map[string]LocalInfo, len(locals))
	for name, info := range locals {
		out[name] = info
	}
	return out
}

func bindMatchPatternLocalsForInference(
	pattern frontend.Expr,
	scrutType string,
	locals map[string]LocalInfo,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	info, ok := types[scrutType]
	if !ok {
		return fmt.Errorf("unknown type '%s'", scrutType)
	}
	switch pat := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(pat.At))
		}
		elemInfo, err := ensureTypeInfo(info.ElemType, types)
		if err != nil {
			return err
		}
		locals[pat.Name] = LocalInfo{SlotCount: elemInfo.SlotCount, TypeName: info.ElemType}
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
			locals[binding] = LocalInfo{SlotCount: caseInfo.PayloadSlots[i], TypeName: caseInfo.PayloadTypes[i]}
		}
	}
	return nil
}

func validateEnumCasePatternPayload(pattern *frontend.EnumCasePatternExpr, caseType string, caseInfo EnumCaseInfo, module string) error {
	want := len(caseInfo.PayloadTypes)
	got := len(pattern.Bindings)
	if got > 0 && !pattern.HasPayload {
		pattern.HasPayload = true
	}
	if want == 0 {
		if pattern.HasPayload {
			return fmt.Errorf("%s: enum case '%s.%s' has no payload; use '%s.%s'", frontend.FormatPos(pattern.At), displayTypeName(caseType, module), pattern.CaseName, displayTypeName(caseType, module), pattern.CaseName)
		}
		if got != 0 {
			return fmt.Errorf("%s: enum case '%s.%s' pattern expects 0 binding(s), got %d", frontend.FormatPos(pattern.At), displayTypeName(caseType, module), pattern.CaseName, got)
		}
		return nil
	}
	if !pattern.HasPayload {
		return fmt.Errorf("%s: enum case '%s.%s' carries %d payload value(s); use '%s.%s(%s)'", frontend.FormatPos(pattern.At), displayTypeName(caseType, module), pattern.CaseName, want, displayTypeName(caseType, module), pattern.CaseName, placeholderBindingList(want))
	}
	if got != want {
		return fmt.Errorf("%s: enum case '%s.%s' pattern expects %d binding(s), got %d", frontend.FormatPos(pattern.At), displayTypeName(caseType, module), pattern.CaseName, want, got)
	}
	return nil
}

func placeholderBindingList(n int) string {
	if n <= 0 {
		return ""
	}
	bindings := make([]string, n)
	for i := range bindings {
		bindings[i] = fmt.Sprintf("value%d", i+1)
	}
	return strings.Join(bindings, ", ")
}

func stmtEndsWithReturn(stmt frontend.Stmt) bool {
	return semanticsstatements.EndsWithReturn(stmt)
}
