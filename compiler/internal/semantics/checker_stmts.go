package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

func checkStmts(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	returnType string,
	borrowedParams map[string]struct{},
	inoutParams map[string]struct{},
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) error {
	if state != nil {
		state.pushDeferCaptureFrame()
		defer state.popDeferCaptureFrame()
	}
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			if err := effects.require(s.At, "io"); err != nil {
				return err
			}
			tname, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !isPrintableType(tname, types) {
				return fmt.Errorf("%s: print expects str or []u8", frontend.FormatPos(s.At))
			}
			secretTainted, err := exprSecretTainted(s.Value, tname, locals, globals, funcs, types, module, imports, analysis)
			if err != nil {
				return err
			}
			if analysis.underSecretControl() {
				secretTainted = true
			}
			if secretTainted {
				return privacyDiagnosticf(s.At, "secret-tainted value cannot be printed")
			}
		case *frontend.BreakStmt:
			if state.loopDepth == 0 {
				return fmt.Errorf("%s: break outside loop", frontend.FormatPos(s.At))
			}
			recordLoopFlowExit(state, "break", analysis)
			if err := state.checkPendingDeferCaptures(s.At); err != nil {
				return err
			}
			state.reachable = false
			return nil
		case *frontend.ContinueStmt:
			if state.loopDepth == 0 {
				return fmt.Errorf("%s: continue outside loop", frontend.FormatPos(s.At))
			}
			recordLoopFlowExit(state, "continue", analysis)
			if err := state.checkPendingDeferCaptures(s.At); err != nil {
				return err
			}
			state.reachable = false
			return nil
		case *frontend.FreeStmt:
			if err := effects.requireAll(s.At, []string{"islands", "mem"}); err != nil {
				return err
			}
			tname, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if tname != "island" {
				return fmt.Errorf("%s: free expects island, got '%s'", frontend.FormatPos(s.At), tname)
			}
			if !s.Implicit && !state.inUnsafe() {
				return effectDiagnosticf(s.At, "free is only allowed in unsafe blocks")
			}
			source, err := resourceSourceForExpr(s.Value, funcs, module, imports, state)
			if err != nil {
				return err
			}
			if source.ambiguous {
				return ownershipDiagnosticf(s.Value.Pos(), "resource expression mixes resource provenance")
			}
			if source.unknown {
				name, _ := resourcePathForExpr(s.Value)
				if name == "" {
					name = "<resource>"
				}
				return ownershipDiagnosticf(s.Value.Pos(), "ambiguous resource provenance for '%s' after control-flow merge", name)
			}
			if source.known {
				state.markResourceFinalized(source.name, "freed", s.Value.Pos())
			}
		case *frontend.ReturnStmt:
			if err := checkReturnStmt(s, locals, globals, funcs, types, module, imports, returnType, borrowedParams, state, effects, analysis); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if state.throwType == "" {
				return fmt.Errorf("%s: throw is only allowed in throwing functions", frontend.FormatPos(s.At))
			}
			tname, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(state.throwType, tname, s.Value) {
				return fmt.Errorf("%s: throw type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), state.throwType, tname)
			}
			if surfaceType, ok := surfaceEphemeralValueType(state.throwType, types); ok {
				return lifetimeDiagnosticf(s.At, "surface value '%s' cannot escape via throw; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType)
			}
			if surfaceType, ok := surfaceEphemeralValueType(tname, types); ok {
				return lifetimeDiagnosticf(s.At, "surface value '%s' cannot escape via throw; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType)
			}
			if surfaceFramePixelsEscapeExpr(s.Value, locals, globals, types, analysis) {
				return lifetimeDiagnosticf(s.At, "surface frame pixels cannot escape via throw; keep Frame.pixels local to the active Surface frame")
			}
			if typeMayContainRegion(tname, types) || typeMayContainPtr(tname, types) || typeMayContainRegion(state.throwType, types) || typeMayContainPtr(state.throwType, types) {
				if err := checkBorrowedEscape(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
					return lifetimeDiagnosticf(s.At, "borrowed local '%s' cannot escape via throw", borrowedName)
				}); err != nil {
					return err
				}
			}
			if typeMayContainRegion(tname, types) || typeMayContainRegion(state.throwType, types) {
				if err := checkRegionTreeWithinScope(regionTreeForExpr(tname, s.Value, regionNone, types, state), regionNone, s.At, state); err != nil {
					return err
				}
			}
			if typeContainsResourceHandle(state.throwType, types) {
				summary, unknown, err := returnResourceSummaryForExpr(s.Value, state.throwType, funcs, types, module, imports, state)
				if err != nil {
					return err
				}
				if !unknown {
					if err := state.recordThrowResourceSummary(summary, s.At); err != nil {
						return err
					}
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
				if analysis.rejectSecretReturn {
					return privacyDiagnosticf(s.At, "secret-tainted value cannot be thrown from @export function '%s'", analysis.exportedFuncName)
				}
				if !analysis.allowSecretReturn {
					return privacyDiagnosticf(s.At, "secret-tainted value requires semantic clause 'privacy' before throw")
				}
			}
			state.reachable = false
		case *frontend.DeferStmt:
			if err := validateDeferBodyControl(s.Body, 0); err != nil {
				return err
			}
			scopeID := regionNone
			if state.deferScopes != nil {
				if scoped, ok := state.deferScopes[s]; ok {
					scopeID = scoped
				}
			}
			captures := collectDeferCaptures(s.Body, locals)
			if err := withActiveScope(state, scopeID, func() error {
				return checkDeferBody(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
			}); err != nil {
				return err
			}
			state.registerDeferCaptures(captures)
		case *frontend.IslandStmt:
			if err := effects.requireAll(s.At, []string{"alloc", "islands", "mem"}); err != nil {
				return err
			}
			sizeType, _, err := checkExprWithEffects(s.Size, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !isInt32Like(sizeType) {
				return fmt.Errorf("%s: island size must be i32/u8", frontend.FormatPos(s.At))
			}
			if err := state.enterIsland(s.Name); err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if err := checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis); err != nil {
				state.exitIsland()
				return err
			}
			state.exitIsland()
		case *frontend.LetStmt:
			state.clearConsumed(s.Name)
			resolved, err := resolveTypeName(&s.Type, module, imports)
			if err != nil {
				return err
			}
			s.Type.Name = resolved
			if _, err := ensureTypeInfo(resolved, types); err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			valType := ""
			valRegion := regionNone
			handledFunctionSymbol := false
			if s.Type.Kind == frontend.TypeRefFunction {
				if _, ok := s.Value.(*frontend.ClosureExpr); ok {
					if info, exists := locals[s.Name]; exists && info.FunctionTypeValue {
						valType = resolved
						valRegion = regionNone
						handledFunctionSymbol = true
					}
				} else if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if info, exists := locals[s.Name]; exists && info.FunctionTypeValue {
						if info.FunctionValue == "" {
							if sourceInfo, sourceExists := locals[id.Name]; !sourceExists || !sourceInfo.FunctionTypeValue {
								return unsupportedFunctionTypedLocalInitializerSourceError(s.At, s.Name)
							}
						}
						valType = resolved
						valRegion = regionNone
						handledFunctionSymbol = true
					}
				} else if _, ok := s.Value.(*frontend.FieldAccessExpr); ok {
					if info, exists := locals[s.Name]; exists && info.FunctionTypeValue && info.FunctionValue != "" {
						valType = resolved
						valRegion = regionNone
						handledFunctionSymbol = true
					}
				}
			}
			if !handledFunctionSymbol {
				var checkErr error
				valType, valRegion, checkErr = checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if checkErr != nil {
					return checkErr
				}
			}
			if !typesCompatibleWithNullPtr(resolved, valType, s.Value) {
				return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), resolved, valType)
			}
			if err := checkWholeOwnershipValueAvailable(s.Value, types, module, imports, state); err != nil {
				return err
			}
			secretTainted, err := exprSecretTainted(s.Value, valType, locals, globals, funcs, types, module, imports, analysis)
			if err != nil {
				return err
			}
			if typeUsesSecret(resolved, types) {
				secretTainted = true
			}
			if analysis.underSecretControl() {
				secretTainted = true
			}
			analysis.setLocalSecretTaint(s.Name, secretTainted)
			if source, ok := surfaceFramePixelsSourceExpr(s.Value, locals, globals, types, analysis); ok {
				bindLocalSurfaceFramePixelsSource(locals, analysis, s.Name, source)
			} else {
				bindLocalSurfaceFramePixelsSource(locals, analysis, s.Name, "")
			}
			if owner, ok := surfaceHandleOwnerPathExprWithAnalysis(s.Value, locals, globals, types, analysis); ok {
				analysis.setLocalSurfaceHandleOwner(s.Name, owner)
			} else {
				analysis.setLocalSurfaceHandleOwner(s.Name, "")
			}
			bindSurfaceFrameOwnerForLocal(s.Name, resolved, s.Value, analysis)
			if typeMayContainRegion(resolved, types) {
				scopeID := localScopeID(s.Name, state)
				if err := checkRegionTreeWithinScope(regionTreeForExpr(resolved, s.Value, valRegion, types, state), scopeID, s.At, state); err != nil {
					return err
				}
				bindRegionTreeFromExpr(s.Name, resolved, s.Value, valRegion, types, state)
			}
			state.bindRegion(s.Name, valRegion)
			bindBorrowedPtrAliasFromExpr(s.Name, resolved, s.Value, types, module, imports, state, borrowedParams)
			if err := bindResourceTreeFromExpr(s.Name, resolved, s.Value, funcs, types, module, imports, state); err != nil {
				return err
			}
			if err := bindOwnedRegionSliceOwnerFromExpr(s.Name, resolved, s.Value, types, module, imports, state); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if s.CompoundValue != nil && compoundIndexTargetHasSideEffects(s.Target) {
				return fmt.Errorf("%s: compound index assignment target with side effects is not supported; use an explicit temporary index", frontend.FormatPos(s.At))
			}
			if idx, ok := s.Target.(*frontend.IndexExpr); ok {
				if err := rejectRepresentationMetadataExprAssignment(idx.Base, locals, globals, types); err != nil {
					return err
				}
				indexType, _, err := checkExprWithEffects(idx.Index, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
				if !isInt32Like(indexType) {
					return fmt.Errorf("%s: index must be i32/u8", frontend.FormatPos(idx.At))
				}
				if _, _, err := checkExprWithEffects(idx.Base, locals, globals, funcs, types, module, imports, state, effects, analysis); err != nil {
					return err
				}
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if err := state.checkNotConsumed(id.Name, s.At); err != nil {
					return err
				}
				if g, ok := globals[id.Name]; ok {
					if !g.Mutable {
						if g.Const {
							return fmt.Errorf("%s: cannot assign to const '%s'", frontend.FormatPos(s.At), id.Name)
						}
						return fmt.Errorf("%s: cannot assign to val '%s'", frontend.FormatPos(s.At), id.Name)
					}
					if g.FunctionTypeValue {
						if analysis != nil {
							analysis.touchesMutableGlobals = true
						}
						targetInfo := LocalInfo{
							TypeName:                g.TypeName,
							SlotCount:               FnPtrSlotCount,
							FunctionTypeValue:       true,
							FunctionParamTypes:      append([]string(nil), g.FunctionParamTypes...),
							FunctionParamOwnership:  append([]string(nil), g.FunctionParamOwnership...),
							FunctionReturnType:      g.FunctionReturnType,
							FunctionReturnOwnership: g.FunctionReturnOwnership,
							FunctionThrowsType:      g.FunctionThrowsType,
							FunctionEffects:         append([]string(nil), g.FunctionEffects...),
						}
						allowCapturedGlobalSnapshot, err := allowCapturedGlobalFunctionSnapshot(s.Value, locals, types, state)
						if err != nil {
							return err
						}
						if err := validateFunctionTypedAssignmentValue(id.Name, targetInfo, s.Value, locals, globals, funcs, types, module, imports, s.At, allowCapturedGlobalSnapshot, callableBoundaryGlobal); err != nil {
							return err
						}
						continue
					}
					valType, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
					if err != nil {
						return err
					}
					if !typesCompatibleWithNullPtr(g.TypeName, valType, s.Value) {
						return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), g.TypeName, valType)
					}
					if err := checkWholeOwnershipValueAvailable(s.Value, types, module, imports, state); err != nil {
						return err
					}
					if surfaceType, ok := surfaceEphemeralValueType(g.TypeName, types); ok {
						return lifetimeDiagnosticf(s.At, "surface value '%s' cannot escape via global assignment to '%s'; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType, id.Name)
					}
					if surfaceType, ok := surfaceEphemeralValueType(valType, types); ok {
						return lifetimeDiagnosticf(s.At, "surface value '%s' cannot escape via global assignment to '%s'; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType, id.Name)
					}
					if surfaceFramePixelsEscapeExpr(s.Value, locals, globals, types, analysis) {
						return lifetimeDiagnosticf(s.At, "surface frame pixels cannot escape via global assignment to '%s'; keep Frame.pixels local to the active Surface frame", id.Name)
					}
					if typeMayContainRegion(valType, types) || typeMayContainRegion(g.TypeName, types) ||
						typeMayContainPtr(valType, types) || typeMayContainPtr(g.TypeName, types) {
						if err := checkBorrowedAggregateEscape(s.Value, g.TypeName, "be stored in global", locals, globals, funcs, types, module, imports, state, effects, analysis, s.At); err != nil {
							return err
						}
						if err := checkBorrowedEscape(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
							return lifetimeDiagnosticf(s.At, "borrowed local '%s' cannot escape via global assignment to '%s'", borrowedName, id.Name)
						}); err != nil {
							return err
						}
					}
					if valType == "ptr" {
						if borrowedName, borrowed := borrowedPtrOwnerFromExpr(s.Value, state, borrowedParams); borrowed {
							return lifetimeDiagnosticf(s.At, "borrowed local '%s' cannot escape via global assignment to '%s'", borrowedName, id.Name)
						}
					}
					secretTainted, err := exprSecretTainted(s.Value, valType, locals, globals, funcs, types, module, imports, analysis)
					if err != nil {
						return err
					}
					if analysis.underSecretControl() {
						secretTainted = true
					}
					if secretTainted {
						return privacyDiagnosticf(s.At, "secret-tainted value cannot be stored in global '%s'", id.Name)
					}
					continue
				}
			}
			targetInfo, targetType, err := resolveAssignTarget(s.Target, locals, globals, types)
			if err != nil {
				return err
			}
			if !targetInfo.Global {
				if err := checkLocalScope(targetInfo.Name, state, s.At); err != nil {
					return err
				}
			}
			if !targetInfo.Mutable {
				if targetInfo.Const {
					return fmt.Errorf("%s: cannot assign to const '%s'", frontend.FormatPos(s.At), targetInfo.Name)
				}
				return fmt.Errorf("%s: cannot assign to val '%s'", frontend.FormatPos(s.At), targetInfo.Name)
			}
			targetOwnershipPath := ""
			if path, ok := canonicalOwnershipAccessPath(s.Target); ok {
				if err := state.checkAssignableOwnershipPath(path, s.At); err != nil {
					return err
				}
				targetOwnershipPath = path
			}
			handledFunctionAssignment := false
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if localInfo, exists := locals[id.Name]; exists && localInfo.FunctionTypeValue {
					markMutableFunctionTypedGlobalSource(s.Value, globals, analysis)
					if err := validateFunctionTypedAssignmentValue(id.Name, localInfo, s.Value, locals, globals, funcs, types, module, imports, s.At, true, callableBoundaryLocal); err != nil {
						return err
					}
					if err := updateFunctionTypedLocalAssignmentMetadata(id.Name, s.Value, locals, globals, funcs, types, module, imports); err != nil {
						return err
					}
					handledFunctionAssignment = true
				}
			} else if targetName, fieldInfo, ok, err := functionFieldLocalInfoFromExpr(s.Target, locals, types); err != nil {
				return err
			} else if ok {
				markMutableFunctionTypedGlobalSource(s.Value, globals, analysis)
				if err := validateFunctionTypedAssignmentValue(targetName, fieldInfo, s.Value, locals, globals, funcs, types, module, imports, s.At, true, callableBoundaryStructField); err != nil {
					return err
				}
				if err := updateFunctionTypedFieldAssignmentMetadata(targetName, fieldInfo, s.Value, locals, globals, funcs, types, module, imports); err != nil {
					return err
				}
				handledFunctionAssignment = true
			}
			valType := targetType
			valRegion := regionNone
			if !handledFunctionAssignment {
				var err error
				valType, valRegion, err = checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
			}
			if !typesCompatibleWithNullPtr(targetType, valType, s.Value) {
				return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), targetType, valType)
			}
			if err := checkWholeOwnershipValueAvailable(s.Value, types, module, imports, state); err != nil {
				return err
			}
			secretTainted := false
			if !handledFunctionAssignment {
				var err error
				secretTainted, err = exprSecretTainted(s.Value, valType, locals, globals, funcs, types, module, imports, analysis)
				if err != nil {
					return err
				}
			}
			if analysis.underSecretControl() {
				secretTainted = true
			}
			if secretTainted && targetInfo.ActorField {
				return privacyDiagnosticf(s.At, "secret-tainted value cannot be stored in actor state field '%s'", targetInfo.Name)
			}
			if targetInfo.Global {
				if surfaceType, ok := surfaceEphemeralValueType(targetType, types); ok {
					return lifetimeDiagnosticf(s.At, "surface value '%s' cannot escape via global assignment to '%s'; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType, targetInfo.Name)
				}
				if surfaceType, ok := surfaceEphemeralValueType(valType, types); ok {
					return lifetimeDiagnosticf(s.At, "surface value '%s' cannot escape via global assignment to '%s'; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType, targetInfo.Name)
				}
				if surfaceFramePixelsEscapeExpr(s.Value, locals, globals, types, analysis) {
					return lifetimeDiagnosticf(s.At, "surface frame pixels cannot escape via global assignment to '%s'; keep Frame.pixels local to the active Surface frame", targetInfo.Name)
				}
				if typeMayContainRegion(valType, types) || typeMayContainRegion(targetType, types) ||
					typeMayContainPtr(valType, types) || typeMayContainPtr(targetType, types) {
					if err := checkBorrowedAggregateEscape(s.Value, targetType, "be stored in global", locals, globals, funcs, types, module, imports, state, effects, analysis, s.At); err != nil {
						return err
					}
					if err := checkBorrowedEscape(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
						return lifetimeDiagnosticf(s.At, "borrowed local '%s' cannot escape via global assignment to '%s'", borrowedName, targetInfo.Name)
					}); err != nil {
						return err
					}
				}
				if secretTainted {
					return privacyDiagnosticf(s.At, "secret-tainted value cannot be stored in global '%s'", targetInfo.Name)
				}
				continue
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				analysis.setLocalSecretTaint(id.Name, secretTainted || typeUsesSecret(targetType, types))
				if source, ok := surfaceFramePixelsSourceExpr(s.Value, locals, globals, types, analysis); ok {
					bindLocalSurfaceFramePixelsSource(locals, analysis, id.Name, source)
				} else {
					bindLocalSurfaceFramePixelsSource(locals, analysis, id.Name, "")
				}
				if owner, ok := surfaceHandleOwnerPathExprWithAnalysis(s.Value, locals, globals, types, analysis); ok {
					analysis.setLocalSurfaceHandleOwner(id.Name, owner)
				} else {
					analysis.setLocalSurfaceHandleOwner(id.Name, "")
				}
				if targetType == surfaceFrameTypeName {
					analysis.clearSurfaceFramePresented(id.Name)
				}
				bindSurfaceFrameOwnerForLocal(id.Name, targetType, s.Value, analysis)
				if localInfo, exists := locals[id.Name]; exists && localInfo.Mutable {
					if fields, err := functionFieldsFromReturnedStructExpr(targetType, s.Value, locals, globals, funcs, types, module, imports); err != nil {
						return err
					} else if len(fields) > 0 || len(localInfo.FunctionFields) > 0 {
						localInfo.FunctionFields = cloneFunctionFieldMap(fields)
						locals[id.Name] = localInfo
					}
					if payloadFields, err := enumPayloadFieldsFromReturnedStructExpr(targetType, s.Value, locals, globals, funcs, types, module, imports); err != nil {
						return err
					} else if len(payloadFields) > 0 || len(localInfo.EnumPayloadFields) > 0 {
						localInfo.EnumPayloadFields = cloneFunctionFieldMap(payloadFields)
						locals[id.Name] = localInfo
					}
				}
			} else if info, ok := types[targetType]; ok && info.Kind == TypeStruct {
				if err := updateFunctionTypedStructFieldAssignmentMetadata(s.Target, targetType, s.Value, locals, globals, funcs, types, module, imports); err != nil {
					return err
				}
				if err := updateEnumPayloadStructFieldAssignmentMetadata(s.Target, targetType, s.Value, locals, globals, funcs, types, module, imports); err != nil {
					return err
				}
			} else if info, ok := types[targetType]; ok && info.Kind == TypeEnum {
				if err := updateEnumPayloadStructFieldAssignmentMetadata(s.Target, targetType, s.Value, locals, globals, funcs, types, module, imports); err != nil {
					return err
				}
			} else if secretTainted {
				analysis.setLocalSecretTaint(targetInfo.Name, true)
			}
			if _, outParam := inoutParams[targetInfo.Name]; outParam {
				if surfaceType, ok := surfaceEphemeralValueType(targetType, types); ok {
					return lifetimeDiagnosticf(s.At, "surface value '%s' cannot escape via inout assignment to '%s'; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType, targetInfo.Name)
				}
				if surfaceType, ok := surfaceEphemeralValueType(valType, types); ok {
					return lifetimeDiagnosticf(s.At, "surface value '%s' cannot escape via inout assignment to '%s'; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType, targetInfo.Name)
				}
				if surfaceFramePixelsEscapeExpr(s.Value, locals, globals, types, analysis) {
					return lifetimeDiagnosticf(s.At, "surface frame pixels cannot escape via inout assignment to '%s'; keep Frame.pixels local to the active Surface frame", targetInfo.Name)
				}
				if valRegion < regionNone || typeMayContainPtr(targetType, types) || typeMayContainPtr(valType, types) {
					if err := checkBorrowedInoutEscape(s.Value, targetInfo.Name, s.At, locals, globals, funcs, types, module, imports, state, effects, analysis); err != nil {
						return err
					}
				}
			}
			if _, ok := s.Target.(*frontend.IndexExpr); !ok {
				targetResourceName := targetInfo.Name
				if path, ok := resourcePathForExpr(s.Target); ok {
					targetResourceName = path
				}
				if targetType == surfaceFrameTypeName {
					analysis.clearSurfaceFramePresented(targetResourceName)
				}
				bindSurfaceFrameOwnerForLocal(targetResourceName, targetType, s.Value, analysis)
				if typeMayContainRegion(targetType, types) {
					scopeID := localScopeID(targetInfo.Name, state)
					if err := checkRegionTreeWithinScope(regionTreeForExpr(targetType, s.Value, valRegion, types, state), scopeID, s.At, state); err != nil {
						return err
					}
					bindRegionTreeFromExpr(targetResourceName, targetType, s.Value, valRegion, types, state)
				}
				state.bindRegion(targetResourceName, valRegion)
				bindBorrowedPtrAliasFromExpr(targetResourceName, targetType, s.Value, types, module, imports, state, borrowedParams)
				if err := bindResourceTreeFromExpr(targetResourceName, targetType, s.Value, funcs, types, module, imports, state); err != nil {
					return err
				}
				if err := bindOwnedRegionSliceOwnerFromExpr(targetResourceName, targetType, s.Value, types, module, imports, state); err != nil {
					return err
				}
			}
			if targetOwnershipPath != "" {
				state.clearConsumedTree(targetOwnershipPath)
			}
		case *frontend.IfStmt:
			condType, _, err := checkExprWithEffects(s.Cond, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !isConditionType(condType) {
				return fmt.Errorf("%s: condition must be bool or i32/u8", frontend.FormatPos(s.At))
			}
			condSecretTainted, err := exprSecretTainted(s.Cond, condType, locals, globals, funcs, types, module, imports, analysis)
			if err != nil {
				return err
			}
			scopeIDs := branchScopeInfo{thenID: regionNone, elseID: regionNone}
			if scoped, ok := state.ifScopes[s]; ok {
				scopeIDs = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			beforeTaint := analysis.copySecretTaint()
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			analysis.restoreSecretTaint(beforeTaint)
			if err := withActiveScope(state, scopeIDs.thenID, func() error {
				return analysis.withSecretControl(condSecretTainted, func() error {
					return checkStmts(s.Then, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
				})
			}); err != nil {
				return err
			}
			thenVars := copyRegionVars(state.regionVars)
			thenFlow := snapshotFlow(state)
			thenTaint := analysis.copySecretTaint()
			var elseVars map[string]int
			var elseFlow flowSnapshot
			var elseTaint map[string]bool
			if len(s.Else) > 0 {
				state.regionVars = copyRegionVars(before)
				restoreFlow(state, beforeFlow)
				analysis.restoreSecretTaint(beforeTaint)
				if err := withActiveScope(state, scopeIDs.elseID, func() error {
					return analysis.withSecretControl(condSecretTainted, func() error {
						return checkStmts(s.Else, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
					})
				}); err != nil {
					return err
				}
				elseVars = copyRegionVars(state.regionVars)
				elseFlow = snapshotFlow(state)
				elseTaint = analysis.copySecretTaint()
			} else {
				elseVars = before
				elseFlow = beforeFlow
				elseTaint = beforeTaint
			}
			mergeControlFlowWithLabels(state, analysis, thenVars, thenFlow, thenTaint, elseVars, elseFlow, elseTaint, "then", "else")
			markUnknownRegions(state)
		case *frontend.IfLetStmt:
			valueType, valueRegion, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			valueSecretTainted, err := exprSecretTainted(s.Value, valueType, locals, globals, funcs, types, module, imports, analysis)
			if err != nil {
				return err
			}
			valueInfo, valueInfoOK := types[valueType]
			if s.Pattern == nil {
				if _, ok := optionalElemName(valueType); !ok {
					return fmt.Errorf("%s: if let requires optional value, got '%s'", frontend.FormatPos(s.At), valueType)
				}
			} else if !valueInfoOK || (valueInfo.Kind != TypeOptional && valueInfo.Kind != TypeEnum) {
				return fmt.Errorf("%s: if let pattern requires optional or enum value, got '%s'", frontend.FormatPos(s.At), valueType)
			} else if err := validateIfLetPattern(s.Pattern, valueType, locals, globals, funcs, types, module, imports, state, effects, analysis); err != nil {
				return err
			}
			valueResourcePath := s.ValueLocal
			valueOwnershipPath := valueResourcePath
			if path, ok := resourcePathForExpr(s.Value); ok {
				valueOwnershipPath = path
			}
			if valueResourcePath != "" {
				if err := bindResourceTreeFromExpr(valueResourcePath, valueType, s.Value, funcs, types, module, imports, state); err != nil {
					return err
				}
				bindRegionTreeFromExpr(valueResourcePath, valueType, s.Value, valueRegion, types, state)
			} else if path, ok := resourcePathForExpr(s.Value); ok {
				valueResourcePath = path
			}
			scopeIDs := branchScopeInfo{thenID: regionNone, elseID: regionNone}
			if scoped, ok := state.ifLetScopes[s]; ok {
				scopeIDs = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			beforeTaint := analysis.copySecretTaint()
			analysis.setLocalSecretTaint(s.ValueLocal, valueSecretTainted)
			beforeTaint = mergeSecretTaintMaps(beforeTaint, analysis.copySecretTaint())
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			analysis.restoreSecretTaint(beforeTaint)
			if err := withActiveScope(state, scopeIDs.thenID, func() error {
				if err := bindPatternOwnershipAliases(s.Pattern, s.Name, valueOwnershipPath, valueType, types, module, imports, state); err != nil {
					return err
				}
				if err := bindPatternBorrowedPtrAliases(s.Pattern, s.Name, valueOwnershipPath, valueType, types, module, imports, state); err != nil {
					return err
				}
				if err := bindPatternResourceLocals(s.Pattern, s.Name, valueResourcePath, valueType, types, module, imports, state); err != nil {
					return err
				}
				if err := bindPatternRegionLocals(s.Pattern, s.Name, valueResourcePath, valueType, types, module, imports, state); err != nil {
					return err
				}
				bindPatternSecretTaintLocals(s.Pattern, s.Name, valueSecretTainted, analysis)
				return analysis.withSecretControl(valueSecretTainted, func() error {
					return checkStmts(s.Then, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
				})
			}); err != nil {
				return err
			}
			thenVars := copyRegionVars(state.regionVars)
			thenFlow := snapshotFlow(state)
			thenTaint := analysis.copySecretTaint()
			var elseVars map[string]int
			var elseFlow flowSnapshot
			var elseTaint map[string]bool
			if len(s.Else) > 0 {
				state.regionVars = copyRegionVars(before)
				restoreFlow(state, beforeFlow)
				analysis.restoreSecretTaint(beforeTaint)
				if err := withActiveScope(state, scopeIDs.elseID, func() error {
					return analysis.withSecretControl(valueSecretTainted, func() error {
						return checkStmts(s.Else, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
					})
				}); err != nil {
					return err
				}
				elseVars = copyRegionVars(state.regionVars)
				elseFlow = snapshotFlow(state)
				elseTaint = analysis.copySecretTaint()
			} else {
				elseVars = before
				elseFlow = beforeFlow
				elseTaint = beforeTaint
			}
			mergeControlFlowWithLabels(state, analysis, thenVars, thenFlow, thenTaint, elseVars, elseFlow, elseTaint, "then", "else")
			markUnknownRegions(state)
		case *frontend.WhileStmt:
			condType, _, err := checkExprWithEffects(s.Cond, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !isConditionType(condType) {
				return fmt.Errorf("%s: condition must be bool or i32/u8", frontend.FormatPos(s.At))
			}
			condSecretTainted, err := exprSecretTainted(s.Cond, condType, locals, globals, funcs, types, module, imports, analysis)
			if err != nil {
				return err
			}
			bodyScopeID := regionNone
			if scoped, ok := state.whileScopes[s]; ok {
				bodyScopeID = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			beforeTaint := analysis.copySecretTaint()
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			analysis.restoreSecretTaint(beforeTaint)
			state.loopDepth++
			pushLoopFlowFrame(state)
			if err := withActiveScope(state, bodyScopeID, func() error {
				return analysis.withSecretControl(condSecretTainted, func() error {
					return checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
				})
			}); err != nil {
				popLoopFlowFrame(state)
				state.loopDepth--
				return err
			}
			loopFrame := popLoopFlowFrame(state)
			state.loopDepth--
			bodyVars := copyRegionVars(state.regionVars)
			bodyFlow := snapshotFlow(state)
			bodyTaint := analysis.copySecretTaint()
			exits := []loopFlowExit{{label: "before", vars: before, flow: beforeFlow, taint: beforeTaint}}
			if bodyFlow.reachable {
				exits = append(exits, loopFlowExit{label: "body", vars: bodyVars, flow: bodyFlow, taint: bodyTaint})
			}
			exits = append(exits, loopFrame.continues...)
			exits = append(exits, loopFrame.breaks...)
			mergeLoopFlowExits(state, analysis, exits)
			markUnknownRegions(state)
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				iterType, _, err := checkExprWithEffects(s.Iterable, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
				elemType, err := collectionElementType(iterType, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
				}
				if loopInfo, ok := locals[s.Name]; ok && loopInfo.TypeName != elemType {
					return fmt.Errorf("%s: for collection element type mismatch: local '%s' is %s, iterable yields %s", frontend.FormatPos(s.At), s.Name, loopInfo.TypeName, elemType)
				}
			} else {
				startType, _, err := checkExprWithEffects(s.Start, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
				endType, _, err := checkExprWithEffects(s.End, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
				if !isInt32Like(startType) || !isInt32Like(endType) {
					return fmt.Errorf("%s: for range bounds must be i32/u8", frontend.FormatPos(s.At))
				}
			}
			bodyScopeID := regionNone
			if scoped, ok := state.forScopes[s]; ok {
				bodyScopeID = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			beforeTaint := analysis.copySecretTaint()
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			analysis.restoreSecretTaint(beforeTaint)
			state.loopDepth++
			pushLoopFlowFrame(state)
			if err := withActiveScope(state, bodyScopeID, func() error {
				return checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
			}); err != nil {
				popLoopFlowFrame(state)
				state.loopDepth--
				return err
			}
			loopFrame := popLoopFlowFrame(state)
			state.loopDepth--
			bodyVars := copyRegionVars(state.regionVars)
			bodyFlow := snapshotFlow(state)
			bodyTaint := analysis.copySecretTaint()
			exits := []loopFlowExit{{label: "before", vars: before, flow: beforeFlow, taint: beforeTaint}}
			if bodyFlow.reachable {
				exits = append(exits, loopFlowExit{label: "body", vars: bodyVars, flow: bodyFlow, taint: bodyTaint})
			}
			exits = append(exits, loopFrame.continues...)
			exits = append(exits, loopFrame.breaks...)
			mergeLoopFlowExits(state, analysis, exits)
			markUnknownRegions(state)
		case *frontend.MatchStmt:
			scrutType, scrutRegion, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			scrutSecretTainted, err := exprSecretTainted(s.Value, scrutType, locals, globals, funcs, types, module, imports, analysis)
			if err != nil {
				return err
			}
			scrutInfo, scrutInfoOK := types[scrutType]
			if !isInt32Like(scrutType) {
				info, ok := types[scrutType]
				if !ok || (info.Kind != TypeEnum && info.Kind != TypeOptional) {
					return fmt.Errorf("%s: match value must be enum or i32/u8", frontend.FormatPos(s.At))
				}
			}
			scrutineeResourcePath := s.ScrutineeLocal
			scrutineeOwnershipPath := scrutineeResourcePath
			if path, ok := resourcePathForExpr(s.Value); ok {
				scrutineeOwnershipPath = path
			}
			if scrutineeResourcePath != "" {
				if err := bindResourceTreeFromExpr(scrutineeResourcePath, scrutType, s.Value, funcs, types, module, imports, state); err != nil {
					return err
				}
				bindRegionTreeFromExpr(scrutineeResourcePath, scrutType, s.Value, scrutRegion, types, state)
			} else if path, ok := resourcePathForExpr(s.Value); ok {
				scrutineeResourcePath = path
			}
			seenDefault := false
			seenPatterns := map[string]frontend.Position{}
			scrutineeFunctionPayloads, err := enumPayloadFunctionValuesForMatchExpr(s.Value, locals, globals, funcs, types, module, imports, scrutType)
			if err != nil {
				return err
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			beforeTaint := analysis.copySecretTaint()
			analysis.setLocalSecretTaint(s.ScrutineeLocal, scrutSecretTainted)
			beforeTaint = mergeSecretTaintMaps(beforeTaint, analysis.copySecretTaint())
			merged := copyRegionVars(before)
			mergedFlow := beforeFlow
			mergedTaint := beforeTaint
			labels := []string{"fallthrough"}
			caseScopes := state.matchCaseScopes[s]
			for i, c := range s.Cases {
				if seenDefault {
					return fmt.Errorf("%s: match default must be last", frontend.FormatPos(c.At))
				}
				if c.Default {
					seenDefault = true
				} else {
					patType := ""
					if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
						if !scrutInfoOK || scrutInfo.Kind != TypeOptional {
							return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(some.At))
						}
						patType = optionalSomePatternType
					} else if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
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
						patType, _, err = checkExprWithEffects(c.Pattern, locals, globals, funcs, types, module, imports, state, effects, analysis)
						if err != nil {
							return err
						}
					}
					if scrutInfoOK && scrutInfo.Kind == TypeOptional && patType != "none" && patType != optionalSomePatternType {
						return fmt.Errorf("%s: optional match supports only 'none', 'some(name)', and '_' patterns", frontend.FormatPos(c.At))
					}
					if !matchPatternCompatible(scrutType, patType, types) {
						return fmt.Errorf("%s: match pattern type mismatch: expected '%s', got '%s'", frontend.FormatPos(c.At), scrutType, patType)
					}
					if c.Guard == nil {
						if key := matchPatternKey(c.Pattern, patType); key != "" {
							if first, exists := seenPatterns[key]; exists {
								return fmt.Errorf("%s: duplicate match pattern (first at %s)", frontend.FormatPos(c.At), frontend.FormatPos(first))
							}
							seenPatterns[key] = c.At
						}
					}
				}
				state.regionVars = copyRegionVars(before)
				restoreFlow(state, beforeFlow)
				analysis.restoreSecretTaint(beforeTaint)
				caseScopeID := regionNone
				if i < len(caseScopes) {
					caseScopeID = caseScopes[i]
				}
				if err := withActiveScope(state, caseScopeID, func() error {
					if err := bindEnumPatternFunctionPayloadLocals(c.Pattern, scrutineeFunctionPayloads, locals, types, module, imports); err != nil {
						return err
					}
					if err := bindPatternOwnershipAliases(c.Pattern, "", scrutineeOwnershipPath, scrutType, types, module, imports, state); err != nil {
						return err
					}
					if err := bindPatternBorrowedPtrAliases(c.Pattern, "", scrutineeOwnershipPath, scrutType, types, module, imports, state); err != nil {
						return err
					}
					if err := bindPatternResourceLocals(c.Pattern, "", scrutineeResourcePath, scrutType, types, module, imports, state); err != nil {
						return err
					}
					if err := bindPatternRegionLocals(c.Pattern, "", scrutineeResourcePath, scrutType, types, module, imports, state); err != nil {
						return err
					}
					bindPatternSecretTaintLocals(c.Pattern, "", scrutSecretTainted, analysis)
					caseControlSecretTainted := scrutSecretTainted
					if c.Guard != nil {
						guardType, _, err := checkExprWithEffects(c.Guard, locals, globals, funcs, types, module, imports, state, effects, analysis)
						if err != nil {
							return err
						}
						if guardType != "bool" {
							return fmt.Errorf("%s: match guard must be Bool", frontend.FormatPos(c.Guard.Pos()))
						}
						guardSecretTainted, err := exprSecretTainted(c.Guard, guardType, locals, globals, funcs, types, module, imports, analysis)
						if err != nil {
							return err
						}
						caseControlSecretTainted = caseControlSecretTainted || guardSecretTainted
					}
					return analysis.withSecretControl(caseControlSecretTainted, func() error {
						return checkStmts(c.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
					})
				}); err != nil {
					return err
				}
				caseVars := copyRegionVars(state.regionVars)
				caseFlow := snapshotFlow(state)
				caseTaint := analysis.copySecretTaint()
				mergeControlFlowWithLabels(state, analysis, merged, mergedFlow, mergedTaint, caseVars, caseFlow, caseTaint, strings.Join(labels, "/"), fmt.Sprintf("case %d", i+1))
				merged = copyRegionVars(state.regionVars)
				mergedFlow = snapshotFlow(state)
				mergedTaint = analysis.copySecretTaint()
				labels = append(labels, fmt.Sprintf("case %d", i+1))
			}
			if seenDefault {
				state.regionVars = merged
				restoreFlow(state, mergedFlow)
				analysis.restoreSecretTaint(mergedTaint)
			} else {
				mergeControlFlowWithLabels(state, analysis, before, beforeFlow, beforeTaint, merged, mergedFlow, mergedTaint, "before", "cases")
			}
			markUnknownRegions(state)
		case *frontend.UnsafeStmt:
			scopeID := regionNone
			if scoped, ok := state.unsafeScopes[s]; ok {
				scopeID = scoped
			}
			state.enterUnsafe()
			if err := withActiveScope(state, scopeID, func() error {
				return checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
			}); err != nil {
				state.exitUnsafe()
				return err
			}
			state.exitUnsafe()
		case *frontend.ExprStmt:
			tname, _, err := checkExprWithEffects(s.Expr, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if _, err := exprSecretTainted(s.Expr, tname, locals, globals, funcs, types, module, imports, analysis); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: unsupported statement", frontend.FormatPos(s.Pos()))
		}
		if err := state.checkPendingDeferCaptures(stmt.Pos()); err != nil {
			return err
		}
		if !state.reachable {
			return nil
		}
	}
	return nil
}
