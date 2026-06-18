package semantics

import (
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

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
			return fmt.Errorf("typed actor message payload supports at most 8 value slots, got %d for '%s'", info.SlotCount-1, typeName)
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
