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
	case *frontend.IdentExpr:
		if err := state.checkNotConsumed(e.Name, e.At); err != nil {
			return "", regionNone, err
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
				return g.TypeName, regionNone, nil
			}
			return "", regionNone, fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(e.At), e.Name)
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
		targetInfo, targetType, err := ResolveFieldAccessType(e, locals, types)
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
		if err := checkLocalScope(targetInfo.Name, state, e.At); err != nil {
			return "", regionNone, err
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
		default:
			return "", regionNone, fmt.Errorf("%s: cannot index '%s'", frontend.FormatPos(e.At), baseType)
		}
	case *frontend.CallExpr:
		return checkCallExprWithEffects(e, locals, globals, funcs, types, module, imports, state, effects, analysis)
	case *frontend.ClosureExpr:
		return "ptr", regionNone, nil
	case *frontend.TryExpr:
		if state.throwType == "" {
			return "", regionNone, fmt.Errorf("%s: try is only allowed in throwing functions", frontend.FormatPos(e.At))
		}
		if _, ok := e.X.(*frontend.AwaitExpr); ok {
			return "", regionNone, fmt.Errorf("%s: async typed-error propagation is not supported in the v1.0 profile", frontend.FormatPos(e.At))
		}
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return "", regionNone, fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		state.allowThrowDepth++
		state.allowThrowCall = call
		tname, regionID, err := checkCallExprWithEffects(call, locals, globals, funcs, types, module, imports, state, effects, analysis)
		state.allowThrowDepth--
		state.allowThrowCall = nil
		return tname, regionID, err
	case *frontend.AwaitExpr:
		if !state.async {
			return "", regionNone, fmt.Errorf("%s: await is only allowed in async functions", frontend.FormatPos(e.At))
		}
		if _, ok := e.X.(*frontend.TryExpr); ok {
			return "", regionNone, fmt.Errorf("%s: async typed-error propagation is not supported in the v1.0 profile", frontend.FormatPos(e.At))
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
	resolved := ""
	isBuiltin := false
	if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
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
	sig, ok := funcs[resolved]
	if !ok {
		if ctorType, ctorRegion, handled, err := checkStructConstructorCallWithEffects(e, locals, globals, funcs, types, module, imports, state, effects, analysis); handled {
			return ctorType, ctorRegion, err
		}
		return "", regionNone, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), resolved)
	}
	if sig.Generic {
		return "", regionNone, fmt.Errorf("%s: generic function '%s' could not be monomorphized; use inferable value arguments", frontend.FormatPos(e.At), e.Name)
	}
	if analysis != nil && !isBuiltin && sig.TouchesMutableGlobals {
		analysis.touchesMutableGlobals = true
	}
	isTryCall := state != nil && state.allowThrowDepth > 0 && state.allowThrowCall == e
	if sig.ThrowsType != "" {
		if !isTryCall {
			return "", regionNone, fmt.Errorf("%s: call to throwing function '%s' requires try", frontend.FormatPos(e.At), resolved)
		}
		if state.throwType == "" {
			return "", regionNone, fmt.Errorf("%s: try is only allowed in throwing functions", frontend.FormatPos(e.At))
		}
		if !typesCompatibleWithNullPtr(state.throwType, sig.ThrowsType, e) {
			return "", regionNone, fmt.Errorf("%s: thrown error type mismatch: expected '%s', got '%s'", frontend.FormatPos(e.At), state.throwType, sig.ThrowsType)
		}
	} else if isTryCall {
		return "", regionNone, fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
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
	consumeArgPositions := make(map[string]frontend.Position)
	borrowArgs := make(map[string]frontend.Position)
	inoutArgs := make(map[string]frontend.Position)
	for i, arg := range e.Args {
		argType, argRegion, err := checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		if !typesCompatibleWithNullPtr(sig.ParamTypes[i], argType, arg) {
			return "", regionNone, fmt.Errorf("%s: type mismatch for '%s' arg %d", frontend.FormatPos(arg.Pos()), resolved, i+1)
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
		}
		state.markConsumed(name, e.Args[i].Pos())
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
	if (resolved == "core.island_make_u8" || resolved == "core.island_make_i32") && len(argRegions) > 0 && argRegions[0] == regionUnknown {
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
