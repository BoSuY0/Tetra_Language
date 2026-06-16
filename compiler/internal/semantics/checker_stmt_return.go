package semantics

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
)

func checkReturnStmt(
	s *frontend.ReturnStmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	returnType string,
	borrowedParams map[string]struct{},
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) error {
	tname := ""
	regionID := regionNone
	handledFunctionReturn := false
	callerSig, callerSigOK := currentCallerSignature(effects, funcs)
	if closure, ok := s.Value.(*frontend.ClosureExpr); ok {
		if !callerSigOK || !callerSig.ReturnFunctionType {
			return unsupportedFunctionValueEscapeError(s.At, callbackArgumentName(s.Value))
		}
		targetInfo := functionReturnLocalInfo(callerSig)
		if err := validateFunctionTypedClosureAssignment("return", targetInfo, closure, locals, funcs, types, module, imports, closure.At); err != nil {
			return err
		}
		closureSymbol := closure.Name
		if _, ok := funcs[closureSymbol]; !ok && module != "" {
			qualified := qualifyName(module, closure.Name)
			if _, ok := funcs[qualified]; ok {
				closureSymbol = qualified
				closure.Name = qualified
			}
		}
		targetSig, ok := funcs[closureSymbol]
		if !ok {
			return fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(s.At), closureSymbol)
		}
		validationSig := targetSig
		if len(closure.Captures) > 0 {
			captureSlots, err := functionCaptureSlotCount(closure.Captures, types)
			if err != nil {
				return err
			}
			if captureSlots < 1 || captureSlots > FnPtrEnvSlotCount {
				if captureSlots < 1 {
					return unsupportedFunctionTypedReturnCaptureError(s.At, "closure literal", captureSlots)
				}
				escapeKind, handleValue, err := classifyCallableEscape(callableBoundaryReturn, closure.Captures, types)
				if err != nil {
					return err
				}
				if analysis != nil {
					analysis.returnFunctionEscapeKind = escapeKind
					analysis.returnFunctionHandleValue = handleValue
				}
			}
			validationSig.ParamTypes = append([]string(nil), callerSig.ReturnFunctionParams...)
			validationSig.ParamOwnership = append([]string(nil), callerSig.ReturnFunctionParamOwnership...)
		}
		if err := validateReturnedFunctionSignature(callerSig, validationSig, s.At, callbackArgumentName(s.Value)); err != nil {
			return err
		}
		if analysis != nil {
			if analysis.returnFunctionSymbol == "" {
				analysis.returnFunctionSymbol = closureSymbol
			}
			recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
			recordReturnFunctionCaptures(analysis, closure.Captures)
		}
		tname = returnType
		regionID = regionNone
		handledFunctionReturn = true
	}
	if id, ok := s.Value.(*frontend.IdentExpr); ok {
		if localInfo, exists := locals[id.Name]; exists && localInfo.FunctionTypeValue {
			if !callerSigOK || !callerSig.ReturnFunctionType {
				return unsupportedFunctionValueEscapeError(s.At, id.Name)
			}
			if localInfo.FunctionValue == "" {
				validationSig := FuncSig{
					ParamTypes:      append([]string(nil), localInfo.FunctionParamTypes...),
					ParamOwnership:  append([]string(nil), localInfo.FunctionParamOwnership...),
					ReturnType:      localInfo.FunctionReturnType,
					ReturnOwnership: localInfo.FunctionReturnOwnership,
					ThrowsType:      localInfo.FunctionThrowsType,
					Effects:         append([]string(nil), localInfo.FunctionEffects...),
				}
				if err := validateReturnedFunctionSignature(callerSig, validationSig, s.At, id.Name); err != nil {
					return err
				}
				if analysis != nil {
					analysis.returnFunctionParamName = localInfo.FunctionParamName
					if analysis.returnFunctionParamName == "" {
						analysis.returnFunctionParamName = id.Name
					}
					if localInfo.FunctionTouchesMutableGlobals {
						analysis.returnFunctionTouchesMutableGlobals = true
					}
				}
				tname = localInfo.TypeName
				regionID = regionNone
				handledFunctionReturn = true
			} else {
				if localInfo.GenericFunctionValue {
					return unsupportedGenericFunctionTypedReturnError(s.At, id.Name)
				}
				targetSig, ok := funcs[localInfo.FunctionValue]
				if !ok {
					return fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(s.At), localInfo.FunctionValue)
				}
				if targetSig.Generic {
					return unsupportedGenericFunctionTypedReturnError(s.At, id.Name)
				}
				validationSig := targetSig
				if localInfo.FunctionReturnType != "" {
					explicitSlots, err := functionParamSlotCount(localInfo.FunctionParamTypes, types)
					if err != nil {
						return err
					}
					hiddenSlots := targetSig.ParamSlots - explicitSlots
					if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !localInfo.FunctionHandleValue) {
						return unsupportedFunctionTypedReturnCaptureError(s.At, id.Name, hiddenSlots)
					}
					if hiddenSlots > FnPtrEnvSlotCount && analysis != nil {
						analysis.returnFunctionEscapeKind = localInfo.FunctionEscapeKind
						analysis.returnFunctionHandleValue = localInfo.FunctionHandleValue
					}
					validationSig.ParamTypes = append([]string(nil), localInfo.FunctionParamTypes...)
					validationSig.ParamOwnership = append([]string(nil), localInfo.FunctionParamOwnership...)
					validationSig.ParamSlots = explicitSlots
					validationSig.ReturnType = localInfo.FunctionReturnType
					validationSig.ReturnOwnership = localInfo.FunctionReturnOwnership
					validationSig.ThrowsType = localInfo.FunctionThrowsType
					validationSig.Effects = append([]string(nil), localInfo.FunctionEffects...)
				}
				if err := validateReturnedFunctionSignature(callerSig, validationSig, s.At, id.Name); err != nil {
					return err
				}
				if analysis != nil {
					if analysis.returnFunctionSymbol == "" {
						analysis.returnFunctionSymbol = localInfo.FunctionValue
					}
					if localInfo.FunctionTouchesMutableGlobals {
						analysis.returnFunctionTouchesMutableGlobals = true
					}
					recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
					recordReturnFunctionCaptures(analysis, localInfo.FunctionCaptures)
					recordReturnFunctionCaptures(analysis, localInfo.FunctionEscapeCaptures)
				}
				tname = localInfo.TypeName
				regionID = regionNone
				handledFunctionReturn = true
			}
		} else if localInfo, exists := locals[id.Name]; exists && localInfo.FunctionValue != "" {
			if !callerSigOK || !callerSig.ReturnFunctionType {
				return unsupportedFunctionValueEscapeError(s.At, id.Name)
			}
			if localInfo.GenericFunctionValue {
				return unsupportedGenericFunctionTypedReturnError(s.At, id.Name)
			}
			targetSig, ok := funcs[localInfo.FunctionValue]
			if !ok {
				return fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(s.At), localInfo.FunctionValue)
			}
			if targetSig.Generic {
				return unsupportedGenericFunctionTypedReturnError(s.At, id.Name)
			}
			validationSig := targetSig
			if len(localInfo.FunctionCaptures) > 0 {
				captureSlots, err := functionCaptureSlotCount(localInfo.FunctionCaptures, types)
				if err != nil {
					return err
				}
				if captureSlots < 1 {
					return unsupportedFunctionTypedReturnCaptureError(s.At, id.Name, captureSlots)
				}
				if captureSlots > FnPtrEnvSlotCount {
					escapeKind, handleValue, err := classifyCallableEscape(callableBoundaryReturn, localInfo.FunctionCaptures, types)
					if err != nil {
						return err
					}
					if analysis != nil {
						analysis.returnFunctionEscapeKind = escapeKind
						analysis.returnFunctionHandleValue = handleValue
					}
				}
				validationSig.ParamTypes = append([]string(nil), callerSig.ReturnFunctionParams...)
				validationSig.ParamOwnership = append([]string(nil), callerSig.ReturnFunctionParamOwnership...)
			}
			if err := validateReturnedFunctionSignature(callerSig, validationSig, s.At, id.Name); err != nil {
				return err
			}
			if analysis != nil {
				if analysis.returnFunctionSymbol == "" {
					analysis.returnFunctionSymbol = localInfo.FunctionValue
				}
				if localInfo.FunctionTouchesMutableGlobals {
					analysis.returnFunctionTouchesMutableGlobals = true
				}
				recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
				recordReturnFunctionCaptures(analysis, localInfo.FunctionCaptures)
				recordReturnFunctionCaptures(analysis, localInfo.FunctionEscapeCaptures)
			}
			tname = returnType
			regionID = regionNone
			handledFunctionReturn = true
		} else if globalInfo, exists := globals[id.Name]; exists && globalInfo.FunctionTypeValue {
			if !callerSigOK || !callerSig.ReturnFunctionType {
				return unsupportedFunctionValueEscapeError(s.At, id.Name)
			}
			markMutableFunctionTypedGlobalSource(s.Value, globals, analysis)
			if globalInfo.FunctionValue == "" {
				if globalInfo.Mutable {
					return unsupportedImportedMutableFunctionTypedGlobalUseError(s.At, id.Name)
				}
				return unsupportedFunctionTypedGlobalTargetError(s.At, id.Name)
			}
			validationSig := FuncSig{
				ParamTypes:      append([]string(nil), globalInfo.FunctionParamTypes...),
				ParamOwnership:  append([]string(nil), globalInfo.FunctionParamOwnership...),
				ReturnType:      globalInfo.FunctionReturnType,
				ReturnOwnership: globalInfo.FunctionReturnOwnership,
				ThrowsType:      globalInfo.FunctionThrowsType,
				Effects:         append([]string(nil), globalInfo.FunctionEffects...),
			}
			if err := validateReturnedFunctionSignature(callerSig, validationSig, s.At, id.Name); err != nil {
				return err
			}
			if analysis != nil && !globalInfo.Mutable && analysis.returnFunctionSymbol == "" {
				analysis.returnFunctionSymbol = globalInfo.FunctionValue
			}
			if analysis != nil && !globalInfo.Mutable {
				if targetSig, ok := funcs[globalInfo.FunctionValue]; ok {
					recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
				}
			}
			tname = globalInfo.TypeName
			regionID = regionNone
			handledFunctionReturn = true
		}
	}
	if !handledFunctionReturn {
		if id, ok := s.Value.(*frontend.IdentExpr); ok && callerSigOK && callerSig.ReturnFunctionType {
			if _, exists := locals[id.Name]; exists {
				// Local function-typed values and local closure pointers are handled by
				// the local return path or by the normal expression checker.
			} else if _, exists := globals[id.Name]; exists {
				// Globals are ordinary values, not direct function symbols.
			} else {
				resolved, err := resolveCheckedCallName(id.Name, funcs, module, imports, id.At)
				if err == nil {
					targetSig, ok := funcs[resolved]
					if !ok {
						return fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(s.At), resolved)
					}
					if err := ensureFuncVisible(resolved, targetSig, module, id.At); err != nil {
						return err
					}
					if targetSig.Generic {
						return unsupportedGenericFunctionTypedReturnError(s.At, id.Name)
					}
					if err := validateReturnedFunctionSignature(callerSig, targetSig, s.At, id.Name); err != nil {
						return err
					}
					if analysis != nil {
						if analysis.returnFunctionSymbol == "" {
							analysis.returnFunctionSymbol = resolved
						}
						recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
					}
					id.Name = resolved
					tname = returnType
					regionID = regionNone
					handledFunctionReturn = true
				}
			}
		}
	}
	if !handledFunctionReturn {
		if fieldInfo, ok, err := resolveFunctionFieldArgument(s.Value, locals); err != nil {
			return err
		} else if ok {
			if !callerSigOK || !callerSig.ReturnFunctionType {
				return unsupportedFunctionValueEscapeError(s.At, callbackArgumentName(s.Value))
			}
			if fieldInfo.FunctionValue == "" {
				if !functionFieldInfoHasTargetSet(fieldInfo) {
					return fmt.Errorf("%s: returning function-typed value '%s' requires a symbol-backed non-capturing function value in this MVP", frontend.FormatPos(s.At), callbackArgumentName(s.Value))
				}
				if err := validateReturnedFunctionSignature(callerSig, functionFieldInfoSig(fieldInfo), s.At, callbackArgumentName(s.Value)); err != nil {
					return err
				}
				if analysis != nil {
					analysis.returnFunctionParamName = fieldInfo.FunctionParamName
					if fieldInfo.FunctionTouchesMutableGlobals {
						analysis.returnFunctionTouchesMutableGlobals = true
					}
					recordReturnFunctionCaptures(analysis, fieldInfo.FunctionCaptures)
					recordReturnFunctionCaptures(analysis, fieldInfo.FunctionEscapeCaptures)
				}
				tname = returnType
				regionID = regionNone
				handledFunctionReturn = true
			} else {
				targetSig, ok := funcs[fieldInfo.FunctionValue]
				if !ok {
					return fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(s.At), fieldInfo.FunctionValue)
				}
				if targetSig.Generic {
					return unsupportedGenericFunctionTypedReturnError(s.At, callbackArgumentName(s.Value))
				}
				validationSig := targetSig
				if fieldInfo.FunctionReturnType != "" {
					explicitSlots, err := functionParamSlotCount(fieldInfo.FunctionParamTypes, types)
					if err != nil {
						return err
					}
					hiddenSlots := targetSig.ParamSlots - explicitSlots
					if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !fieldInfo.FunctionHandleValue) {
						return unsupportedFunctionTypedReturnCaptureError(s.At, callbackArgumentName(s.Value), hiddenSlots)
					}
					if hiddenSlots > FnPtrEnvSlotCount && analysis != nil {
						analysis.returnFunctionEscapeKind = fieldInfo.FunctionEscapeKind
						analysis.returnFunctionHandleValue = fieldInfo.FunctionHandleValue
					}
					validationSig.ParamTypes = append([]string(nil), fieldInfo.FunctionParamTypes...)
					validationSig.ParamOwnership = append([]string(nil), fieldInfo.FunctionParamOwnership...)
					validationSig.ParamSlots = explicitSlots
					validationSig.ReturnType = fieldInfo.FunctionReturnType
					validationSig.ReturnOwnership = fieldInfo.FunctionReturnOwnership
					validationSig.Effects = append([]string(nil), fieldInfo.FunctionEffects...)
				}
				if err := validateReturnedFunctionSignature(callerSig, validationSig, s.At, callbackArgumentName(s.Value)); err != nil {
					return err
				}
				if analysis != nil {
					if analysis.returnFunctionSymbol == "" {
						analysis.returnFunctionSymbol = fieldInfo.FunctionValue
					}
					if fieldInfo.FunctionTouchesMutableGlobals {
						analysis.returnFunctionTouchesMutableGlobals = true
					}
					recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
					recordReturnFunctionCaptures(analysis, fieldInfo.FunctionCaptures)
					recordReturnFunctionCaptures(analysis, fieldInfo.FunctionEscapeCaptures)
				}
				tname = returnType
				regionID = regionNone
				handledFunctionReturn = true
			}
		} else if fieldAccess, fieldOK := s.Value.(*frontend.FieldAccessExpr); fieldOK && callerSigOK && callerSig.ReturnFunctionType {
			if globalInfo, globalSig, globalOK, err := resolveFunctionTypedGlobalFieldAccess(fieldAccess, globals, funcs); err != nil {
				return err
			} else if globalOK {
				if err := validateReturnedFunctionSignature(callerSig, globalSig, s.At, callbackArgumentName(s.Value)); err != nil {
					return err
				}
				if analysis != nil {
					if analysis.returnFunctionSymbol == "" {
						analysis.returnFunctionSymbol = globalInfo.FunctionValue
					}
					recordReturnFunctionTargetMutableGlobalUse(analysis, globalSig)
				}
				tname = returnType
				regionID = regionNone
				handledFunctionReturn = true
			} else if resolved, importedOK := resolveImportedFunctionFieldAccess(fieldAccess, funcs, module, imports); importedOK {
				targetSig := funcs[resolved]
				if targetSig.Generic {
					return unsupportedGenericFunctionTypedReturnError(s.At, callbackArgumentName(s.Value))
				}
				if err := validateReturnedFunctionSignature(callerSig, targetSig, s.At, callbackArgumentName(s.Value)); err != nil {
					return err
				}
				if analysis != nil {
					if analysis.returnFunctionSymbol == "" {
						analysis.returnFunctionSymbol = resolved
					}
					recordReturnFunctionTargetMutableGlobalUse(analysis, targetSig)
				}
				tname = returnType
				regionID = regionNone
				handledFunctionReturn = true
			}
		}
	}
	if !handledFunctionReturn {
		var err error
		tname, regionID, err = checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return err
		}
	}
	if callerSigOK && callerSig.ReturnFunctionType && !handledFunctionReturn {
		return unsupportedFunctionTypedReturnSourceError(s.At)
	}
	if err := checkWholeOwnershipValueAvailable(s.Value, types, module, imports, state); err != nil {
		return err
	}
	if surfaceType, ok := surfaceEphemeralValueType(returnType, types); ok && !surfaceEphemeralReturnAllowed(analysis, surfaceType) {
		return lifetimeDiagnosticf(s.At, "surface value '%s' cannot escape via return; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType)
	}
	if surfaceFramePixelsEscapeExpr(s.Value, locals, globals, types, analysis) {
		return lifetimeDiagnosticf(s.At, "surface frame pixels cannot escape via return; keep Frame.pixels local to the active Surface frame")
	}
	if typeMayContainRegion(tname, types) || typeMayContainPtr(tname, types) {
		if err := checkBorrowedReturnContract(s.Value, returnType, callerSig, callerSigOK, borrowedParams, locals, globals, funcs, types, module, imports, state, effects, analysis, s.At); err != nil {
			return err
		}
	}
	if typeMayContainRegion(tname, types) {
		tree := regionTreeForExpr(tname, s.Value, regionID, types, state)
		if err := checkRegionTreeWithinScope(tree, regionNone, s.At, state); err != nil {
			return err
		}
		if !typeContainsResourceHandle(tname, types) {
			if err := state.recordReturnRegionSummary(tree, s.At); err != nil {
				return err
			}
		}
		regionID = commonRegionFromTree(tree)
		if id, ok := s.Value.(*frontend.IdentExpr); ok {
			if _, borrowed := borrowedParams[id.Name]; borrowed {
				return lifetimeDiagnosticf(s.At, "borrowed local '%s' cannot escape via return", id.Name)
			}
		}
	}
	if tname == "ptr" {
		if borrowedName, borrowed := borrowedPtrOwnerFromExpr(s.Value, state, borrowedParams); borrowed {
			return lifetimeDiagnosticf(s.At, "borrowed local '%s' cannot escape via return", borrowedName)
		}
	}
	resourceReturnType := tname
	if typeContainsResourceHandle(returnType, types) {
		resourceReturnType = returnType
	}
	if typeContainsResourceHandle(resourceReturnType, types) {
		summary, unknown, err := returnResourceSummaryForExpr(s.Value, resourceReturnType, funcs, types, module, imports, state)
		if err != nil {
			return err
		}
		if unknown {
			state.recordUnknownReturnResource()
		} else if err := state.recordReturnResourceSummary(summary, s.At); err != nil {
			return err
		}
	}
	if len(state.returnRegionSummary) == 0 {
		if err := state.recordReturnRegion(regionID, s.At); err != nil {
			return err
		}
	}
	secretTainted, err := exprSecretTainted(s.Value, tname, locals, globals, funcs, types, module, imports, analysis)
	if err != nil {
		return err
	}
	if analysis.underSecretControl() {
		secretTainted = true
	}
	if secretTainted {
		analysis.returnSecretTaint = true
		if analysis.rejectSecretReturn {
			return privacyDiagnosticf(s.At, "secret-tainted value cannot be returned from @export function '%s'", analysis.exportedFuncName)
		}
		if !analysis.allowSecretReturn {
			return privacyDiagnosticf(s.At, "secret-tainted value requires semantic clause 'privacy' before return")
		}
	}
	if !typesCompatibleWithNullPtr(returnType, tname, s.Value) {
		return fmt.Errorf("%s: return type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), returnType, tname)
	}
	if analysis != nil {
		returnFields, err := functionFieldsFromReturnedStructExpr(returnType, s.Value, locals, globals, funcs, types, module, imports)
		if err != nil {
			return err
		}
		if len(returnFields) > 0 || len(analysis.returnFunctionFields) > 0 {
			if len(analysis.returnFunctionFields) == 0 {
				analysis.returnFunctionFields = functionFieldReturnSnapshotMap(returnFields)
			} else {
				for fieldName, field := range returnFields {
					mergeFunctionFieldInfoIntoMap(analysis.returnFunctionFields, fieldName, functionFieldInfoAsReturnSnapshot(field))
				}
			}
		}
		returnPayloadFields, err := enumPayloadFieldsFromReturnedStructExpr(returnType, s.Value, locals, globals, funcs, types, module, imports)
		if err != nil {
			return err
		}
		if len(returnPayloadFields) > 0 || len(analysis.returnEnumPayloadFields) > 0 {
			if len(analysis.returnEnumPayloadFields) == 0 {
				analysis.returnEnumPayloadFields = functionFieldReturnSnapshotMap(returnPayloadFields)
			} else {
				for fieldName, field := range returnPayloadFields {
					mergeFunctionFieldInfoIntoMap(analysis.returnEnumPayloadFields, fieldName, functionFieldInfoAsReturnSnapshot(field))
				}
			}
		}
		returnPayloads, err := enumPayloadFunctionsFromReturnedEnumExpr(returnType, s.Value, locals, globals, funcs, types, module, imports)
		if err != nil {
			return err
		}
		if len(returnPayloads) > 0 || len(analysis.returnEnumPayloadFunctions) > 0 {
			if len(analysis.returnEnumPayloadFunctions) == 0 {
				analysis.returnEnumPayloadFunctions = functionFieldReturnSnapshotMap(returnPayloads)
			} else {
				for payloadKey, payload := range returnPayloads {
					mergeFunctionFieldInfoIntoMap(analysis.returnEnumPayloadFunctions, payloadKey, functionFieldInfoAsReturnSnapshot(payload))
				}
			}
		}
	}
	state.reachable = false
	return nil
}
