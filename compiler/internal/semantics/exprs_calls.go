package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

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
				argRegions, err := checkFunctionTypedCallArguments(e, locals, globals, funcs, types, module, imports, state, effects, analysis, local.FunctionParamTypes, local.FunctionParamOwnership, valueCallPhrase)
				if err != nil {
					return "", regionNone, err
				}
				if err := effects.requireAll(e.At, local.FunctionEffects); err != nil {
					return "", regionNone, err
				}
				if err := validateFunctionTypedThrowCall(local.FunctionThrowsType, e, state); err != nil {
					return "", regionNone, err
				}
				return local.FunctionReturnType, functionTypedBorrowReturnRegion(local.FunctionReturnOwnership, local.FunctionParamOwnership, argRegions, state), nil
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
		if resolved == "core.island_reset" {
			if slicePath, live := state.liveOwnedRegionSliceForOwner(name); live {
				return "", regionNone, ownershipDiagnosticf(e.Args[i].Pos(), "cannot reset island '%s' while borrowed slice '%s' is alive", name, slicePath)
			}
		}
		markConsumedResourceValue(name, consumeArgTypes[i], types, state, e.Args[i].Pos())
		if resolved == "core.island_reset" {
			state.markOwnedRegionSlicesConsumedByOwner(name, e.Args[i].Pos())
		}
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
