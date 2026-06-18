package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

type functionAnalysisState struct {
	touchesMutableGlobals               bool
	returnFunctionSymbol                string
	returnFunctionParamName             string
	returnFunctionCaptures              []frontend.ClosureCapture
	returnFunctionTouchesMutableGlobals bool
	returnFunctionEscapeKind            CallableEscapeKind
	returnFunctionHandleValue           bool
	returnFunctionFields                map[string]FunctionFieldInfo
	returnEnumPayloadFunctions          map[string]FunctionFieldInfo
	returnEnumPayloadFields             map[string]FunctionFieldInfo
	borrowedReturnOwner                 string
	secretTaint                         map[string]bool
	surfaceFramePixels                  map[string]string
	currentFuncName                     string
	funcReturnSecretTaint               map[string]bool
	funcParamSecretTaint                map[string]map[string]bool
	discoveredParamTaint                bool
	allowSecretReturn                   bool
	rejectSecretReturn                  bool
	exportedFuncName                    string
	returnSecretTaint                   bool
	secretControlDepth                  int
	surfacePresentedFrames              map[string]frontend.Position
	surfaceFrameOwners                  map[string]string
	surfaceHandleOwners                 map[string]string
}

func newFunctionAnalysisState(
	fn *frontend.FuncDecl,
	policy functionClausePolicy,
	fullName string,
	returnSecretTaint map[string]bool,
	paramSecretTaint map[string]map[string]bool,
	types map[string]*TypeInfo,
) *functionAnalysisState {
	analysis := &functionAnalysisState{
		secretTaint:           make(map[string]bool),
		currentFuncName:       fullName,
		funcReturnSecretTaint: returnSecretTaint,
		funcParamSecretTaint:  paramSecretTaint,
		allowSecretReturn:     policy.hasPrivacy,
		rejectSecretReturn:    fn.ExportName != "",
		exportedFuncName:      fn.Name,
	}
	for _, param := range fn.Params {
		if typeUsesSecret(param.Type.Name, types) {
			analysis.secretTaint[param.Name] = true
		}
	}
	if inbound := paramSecretTaint[fullName]; len(inbound) > 0 {
		for name, tainted := range inbound {
			if tainted {
				analysis.secretTaint[name] = true
			}
		}
	}
	return analysis
}

func recordReturnFunctionCaptures(analysis *functionAnalysisState, captures []frontend.ClosureCapture) {
	if analysis == nil || len(captures) == 0 || len(analysis.returnFunctionCaptures) > 0 {
		return
	}
	analysis.returnFunctionCaptures = append([]frontend.ClosureCapture(nil), captures...)
}

func recordReturnFunctionTargetMutableGlobalUse(analysis *functionAnalysisState, sig FuncSig) {
	if analysis != nil && sig.TouchesMutableGlobals {
		analysis.returnFunctionTouchesMutableGlobals = true
	}
}

func applyInterfaceFunctionReturnMetadata(
	sig *FuncSig,
	fn *frontend.FuncDecl,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (bool, error) {
	if sig == nil || fn == nil {
		return false, nil
	}
	changed := false
	for _, stmt := range fn.Body {
		if throwStmt, ok := stmt.(*frontend.ThrowStmt); ok {
			if typeContainsResourceHandle(sig.ThrowsType, types) {
				state := newRegionState(nil)
				initParamRegions(fn.Params, state, types)
				summary, unknown, err := returnResourceSummaryForExpr(throwStmt.Value, sig.ThrowsType, funcs, types, module, imports, state)
				if err != nil {
					return false, err
				}
				if !unknown && !returnResourceSummariesEqual(sig.ThrowResourceSummary, summary) {
					sig.ThrowResourceSummary = cloneReturnResourceSummary(summary)
					changed = true
				}
			}
			continue
		}
		if sig.ReturnFunctionType {
			nestedChanged, err := applyInterfaceFunctionReturnParamMetadataFromNestedStmt(sig, stmt, types, module, imports)
			if err != nil {
				return false, err
			}
			if nestedChanged {
				changed = true
			}
		}
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		if typeContainsResourceHandle(sig.ThrowsType, types) {
			if tryExpr, ok := ret.Value.(*frontend.TryExpr); ok {
				if call, ok := tryExpr.X.(*frontend.CallExpr); ok {
					resolved := call.Name
					calleeSig, ok := funcs[resolved]
					if !ok {
						var err error
						resolved, err = resolveKnownCallName(call.Name, funcs, module, imports, call.At)
						if err != nil {
							return false, err
						}
						calleeSig, ok = funcs[resolved]
					}
					if ok && calleeSig.ThrowsType != "" {
						state := newRegionState(nil)
						initParamRegions(fn.Params, state, types)
						if err := recordTryCallThrowResourceSummary(call, calleeSig, funcs, types, module, imports, state); err != nil {
							return false, err
						}
						if len(state.throwResourceSummary) > 0 && !returnResourceSummariesEqual(sig.ThrowResourceSummary, state.throwResourceSummary) {
							sig.ThrowResourceSummary = cloneReturnResourceSummary(state.throwResourceSummary)
							changed = true
						}
					}
				}
			}
		}
		if typeContainsResourceHandle(sig.ReturnType, types) {
			state := newRegionState(nil)
			initParamRegions(fn.Params, state, types)
			summary, unknown, err := returnResourceSummaryForExpr(ret.Value, sig.ReturnType, funcs, types, module, imports, state)
			if err != nil {
				return false, err
			}
			newReturnResourceParam := regionNone
			newReturnResourcePath := ""
			newReturnResourceSummary := ReturnResourceSummary(nil)
			if unknown {
				newReturnResourceParam = regionUnknown
			} else if len(summary) > 0 {
				newReturnResourceSummary = cloneReturnResourceSummary(summary)
				if provenances := summary[""]; len(provenances) == 1 {
					newReturnResourceParam = provenances[0].ParamIndex
					newReturnResourcePath = provenances[0].ParamPath
				}
			}
			if sig.ReturnResourceParam != newReturnResourceParam || sig.ReturnResourcePath != newReturnResourcePath || !returnResourceSummariesEqual(sig.ReturnResourceSummary, newReturnResourceSummary) {
				sig.ReturnResourceParam = newReturnResourceParam
				sig.ReturnResourcePath = newReturnResourcePath
				sig.ReturnResourceSummary = newReturnResourceSummary
				changed = true
			}
		}
		if typeMayContainRegion(sig.ReturnType, types) && !typeContainsResourceHandle(sig.ReturnType, types) {
			summary, err := returnRegionSummaryForInterfaceExpr(ret.Value, sig.ReturnType, sig.Effects, fn, globals, funcs, types, module, imports)
			if err != nil {
				return false, err
			}
			newReturnRegionParam := regionNone
			if len(summary) > 0 {
				commonParam := regionNone
				for _, paramIndex := range summary {
					if commonParam == regionNone {
						commonParam = paramIndex
						continue
					}
					if commonParam != paramIndex {
						commonParam = regionUnknown
						break
					}
				}
				if commonParam >= 0 {
					newReturnRegionParam = commonParam
				}
			}
			if sig.ReturnRegionParam != newReturnRegionParam || !returnRegionSummariesEqual(sig.ReturnRegionSummary, summary) {
				sig.ReturnRegionParam = newReturnRegionParam
				sig.ReturnRegionSummary = cloneReturnRegionSummary(summary)
				changed = true
			}
		}
		if sig.ReturnFunctionType {
			if closure, ok := ret.Value.(*frontend.ClosureExpr); ok {
				locals, err := interfaceFunctionReturnStubLocals(fn.Body, ret, types, module, imports)
				if err != nil {
					return false, err
				}
				if err := configureClosureCaptures(closure, locals, funcs, types, module, true, "interface function-typed return"); err != nil {
					return false, err
				}
				if len(closure.Captures) > 0 {
					target := closureFunctionValueName(closure, funcs, module)
					if sig.ReturnFunctionSymbol != target {
						sig.ReturnFunctionSymbol = target
						changed = true
					}
					if !closureCapturesEqual(sig.ReturnFunctionCaptures, closure.Captures) {
						sig.ReturnFunctionCaptures = append([]frontend.ClosureCapture(nil), closure.Captures...)
						changed = true
					}
					captureSlots, err := functionCaptureSlotCount(closure.Captures, types)
					if err != nil {
						return false, err
					}
					escapeKind := CallableEscapeKind("")
					handleValue := false
					if captureSlots > FnPtrEnvSlotCount {
						escapeKind, handleValue, err = classifyCallableEscape(callableBoundaryReturn, closure.Captures, types)
						if err != nil {
							return false, err
						}
					}
					if sig.ReturnFunctionEscapeKind != escapeKind {
						sig.ReturnFunctionEscapeKind = escapeKind
						changed = true
					}
					if sig.ReturnFunctionHandleValue != handleValue {
						sig.ReturnFunctionHandleValue = handleValue
						changed = true
					}
					desiredReturnSlots := sig.ReturnSlots
					if handleValue {
						desiredReturnSlots = CallableHandleSlotCount
					}
					if sig.ReturnSlots != desiredReturnSlots {
						sig.ReturnSlots = desiredReturnSlots
						changed = true
					}
				}
				continue
			}
			if id, ok := ret.Value.(*frontend.IdentExpr); ok {
				for i, name := range sig.ParamNames {
					if name != id.Name || i >= len(sig.ParamFunctionTypes) || !sig.ParamFunctionTypes[i] {
						continue
					}
					if sig.ReturnFunctionParamName != name {
						sig.ReturnFunctionParamName = name
						changed = true
					}
				}
				continue
			}
			fieldPath := callbackArgumentName(ret.Value)
			if fieldPath != "" {
				for _, name := range sig.ParamNames {
					if !strings.HasPrefix(fieldPath, name+".") {
						continue
					}
					if sig.ReturnFunctionParamName != fieldPath {
						sig.ReturnFunctionParamName = fieldPath
						changed = true
					}
				}
			}
		}
		returnFields, err := functionFieldsFromReturnedStructExpr(sig.ReturnType, ret.Value, nil, globals, funcs, types, module, imports)
		if err != nil {
			return false, err
		}
		if !functionFieldMapsEqual(sig.ReturnFunctionFields, returnFields) {
			sig.ReturnFunctionFields = cloneFunctionFieldMap(returnFields)
			changed = true
		}
		returnPayloadFields, err := enumPayloadFieldsFromReturnedStructExpr(sig.ReturnType, ret.Value, nil, globals, funcs, types, module, imports)
		if err != nil {
			return false, err
		}
		if !functionFieldMapsEqual(sig.ReturnEnumPayloadFields, returnPayloadFields) {
			sig.ReturnEnumPayloadFields = cloneFunctionFieldMap(returnPayloadFields)
			changed = true
		}
		returnPayloads, err := enumPayloadFunctionsFromReturnedEnumExpr(sig.ReturnType, ret.Value, nil, globals, funcs, types, module, imports)
		if err != nil {
			return false, err
		}
		if !functionFieldMapsEqual(sig.ReturnEnumPayloadFunctions, returnPayloads) {
			sig.ReturnEnumPayloadFunctions = cloneFunctionFieldMap(returnPayloads)
			changed = true
		}
	}
	return changed, nil
}

func returnRegionSummaryForInterfaceExpr(
	expr frontend.Expr,
	returnType string,
	effectNames []string,
	fn *frontend.FuncDecl,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (ReturnRegionSummary, error) {
	if expr == nil || fn == nil || !typeMayContainRegion(returnType, types) {
		return nil, nil
	}
	state := newRegionState(nil)
	initParamRegions(fn.Params, state, types)
	locals, err := interfaceParamRegionLocals(fn.Params, types, module, imports)
	if err != nil {
		return nil, err
	}
	effects := newEffectContext(module+"."+fn.Name, effectNames, fn.Uses, strings.HasPrefix(module, "__"))
	tname, regionID, err := checkExprWithEffects(expr, locals, globals, funcs, types, module, imports, state, effects, nil)
	if err != nil {
		return nil, err
	}
	if !typesCompatibleWithNullPtr(returnType, tname, expr) {
		return nil, fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(expr.Pos()), returnType, tname)
	}
	tree := regionTreeForExpr(returnType, expr, regionID, types, state)
	if len(tree) == 0 {
		return nil, nil
	}
	if err := state.recordReturnRegionSummary(tree, expr.Pos()); err != nil {
		return nil, err
	}
	return cloneReturnRegionSummary(state.returnRegionSummary), nil
}

func interfaceParamRegionLocals(params []frontend.ParamDecl, types map[string]*TypeInfo, module string, imports map[string]string) (map[string]LocalInfo, error) {
	locals := make(map[string]LocalInfo, len(params))
	slotIndex := 0
	for _, param := range params {
		paramTypeName, err := resolveTypeName(&param.Type, module, imports)
		if err != nil {
			return nil, err
		}
		info, err := ensureTypeInfo(paramTypeName, types)
		if err != nil {
			return nil, err
		}
		locals[param.Name] = LocalInfo{
			Base:      slotIndex,
			SlotCount: info.SlotCount,
			TypeName:  paramTypeName,
			Mutable:   param.Ownership == "inout",
		}
		slotIndex += info.SlotCount
	}
	return locals, nil
}

func applyInterfaceFunctionReturnParamMetadataFromNestedStmt(
	sig *FuncSig,
	stmt frontend.Stmt,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (bool, error) {
	match, ok := stmt.(*frontend.MatchStmt)
	if !ok {
		return false, nil
	}
	payloadBindings, err := interfaceMatchFunctionPayloadBindings(*sig, match, types, module, imports)
	if err != nil {
		return false, err
	}
	changed := false
	for _, c := range match.Cases {
		for _, caseStmt := range c.Body {
			ret, ok := caseStmt.(*frontend.ReturnStmt)
			if !ok {
				continue
			}
			paramRef := interfaceFunctionReturnParamRef(*sig, ret.Value, payloadBindings)
			if paramRef == "" || sig.ReturnFunctionParamName == paramRef {
				continue
			}
			sig.ReturnFunctionParamName = paramRef
			changed = true
		}
	}
	return changed, nil
}

func interfaceFunctionReturnParamRef(sig FuncSig, expr frontend.Expr, payloadBindings map[string]string) string {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if payloadRef := payloadBindings[id.Name]; payloadRef != "" {
			return payloadRef
		}
		for i, name := range sig.ParamNames {
			if name == id.Name && i < len(sig.ParamFunctionTypes) && sig.ParamFunctionTypes[i] {
				return name
			}
		}
	}
	fieldPath := callbackArgumentName(expr)
	if fieldPath == "" {
		return ""
	}
	for _, name := range sig.ParamNames {
		if strings.HasPrefix(fieldPath, name+".") {
			return fieldPath
		}
	}
	return ""
}

func interfaceMatchFunctionPayloadBindings(
	sig FuncSig,
	match *frontend.MatchStmt,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]string, error) {
	id, ok := match.Value.(*frontend.IdentExpr)
	if !ok {
		return nil, nil
	}
	paramIndex := -1
	for i, name := range sig.ParamNames {
		if name == id.Name {
			paramIndex = i
			break
		}
	}
	if paramIndex < 0 || paramIndex >= len(sig.ParamTypes) {
		return nil, nil
	}
	scrutType := sig.ParamTypes[paramIndex]
	bindings := map[string]string{}
	for _, c := range match.Cases {
		enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr)
		if !ok {
			continue
		}
		caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
		if err != nil {
			return nil, err
		}
		if !found || caseType != scrutType {
			continue
		}
		for i, binding := range enumPat.Bindings {
			if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
				continue
			}
			bindings[binding] = id.Name + "#" + enumPayloadFunctionKey(caseInfo.Ordinal, i)
		}
	}
	return bindings, nil
}

func interfaceFunctionReturnStubLocals(
	body []frontend.Stmt,
	stop *frontend.ReturnStmt,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]LocalInfo, error) {
	locals := map[string]LocalInfo{}
	slot := 0
	for _, stmt := range body {
		if stmt == stop {
			break
		}
		let, ok := stmt.(*frontend.LetStmt)
		if !ok {
			continue
		}
		typeName, err := resolveTypeName(&let.Type, module, imports)
		if err != nil {
			return nil, err
		}
		info, ok := types[typeName]
		if !ok {
			return nil, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(let.At), typeName)
		}
		locals[let.Name] = LocalInfo{
			Base:      slot,
			SlotCount: info.SlotCount,
			TypeName:  typeName,
			Mutable:   let.Mutable,
			Const:     let.Const,
		}
		slot += info.SlotCount
	}
	return locals, nil
}

func (analysis *functionAnalysisState) localSecretTainted(name string) bool {
	return analysis != nil && analysis.secretTaint != nil && analysis.secretTaint[name]
}

func (analysis *functionAnalysisState) setLocalSecretTaint(name string, tainted bool) {
	if analysis == nil || name == "" {
		return
	}
	if analysis.secretTaint == nil {
		analysis.secretTaint = make(map[string]bool)
	}
	if tainted {
		analysis.secretTaint[name] = true
		return
	}
	delete(analysis.secretTaint, name)
}

func (analysis *functionAnalysisState) localSurfaceFramePixels(name string) bool {
	_, ok := analysis.localSurfaceFramePixelsSource(name)
	return ok
}

func (analysis *functionAnalysisState) localSurfaceFramePixelsSource(name string) (string, bool) {
	if analysis == nil || analysis.surfaceFramePixels == nil {
		return "", false
	}
	source, ok := analysis.surfaceFramePixels[name]
	return source, ok
}

func (analysis *functionAnalysisState) setLocalSurfaceFramePixelsSource(name string, frameName string) {
	if analysis == nil || name == "" {
		return
	}
	if analysis.surfaceFramePixels == nil {
		analysis.surfaceFramePixels = make(map[string]string)
	}
	if frameName != "" {
		analysis.surfaceFramePixels[name] = frameName
		return
	}
	delete(analysis.surfaceFramePixels, name)
}

func bindLocalSurfaceFramePixelsSource(locals map[string]LocalInfo, analysis *functionAnalysisState, name string, frameName string) {
	if analysis != nil {
		analysis.setLocalSurfaceFramePixelsSource(name, frameName)
	}
	if info, ok := locals[name]; ok {
		info.SurfaceFramePixelsSource = frameName
		locals[name] = info
	}
}

func (analysis *functionAnalysisState) markSurfaceFramePresented(name string, pos frontend.Position) {
	if analysis == nil || name == "" {
		return
	}
	if analysis.surfacePresentedFrames == nil {
		analysis.surfacePresentedFrames = make(map[string]frontend.Position)
	}
	analysis.surfacePresentedFrames[name] = pos
}

func (analysis *functionAnalysisState) clearSurfaceFramePresented(name string) {
	if analysis == nil || name == "" {
		return
	}
	delete(analysis.surfacePresentedFrames, name)
}

func (analysis *functionAnalysisState) localSurfaceFrameOwner(name string) (string, bool) {
	if analysis == nil || analysis.surfaceFrameOwners == nil {
		return "", false
	}
	owner, ok := analysis.surfaceFrameOwners[name]
	return owner, ok
}

func (analysis *functionAnalysisState) setLocalSurfaceFrameOwner(name string, owner string) {
	if analysis == nil || name == "" {
		return
	}
	if analysis.surfaceFrameOwners == nil {
		analysis.surfaceFrameOwners = make(map[string]string)
	}
	if owner != "" {
		analysis.surfaceFrameOwners[name] = owner
		return
	}
	delete(analysis.surfaceFrameOwners, name)
}

func (analysis *functionAnalysisState) localSurfaceHandleOwner(name string) (string, bool) {
	if analysis == nil || analysis.surfaceHandleOwners == nil {
		return "", false
	}
	owner, ok := analysis.surfaceHandleOwners[name]
	return owner, ok
}

func (analysis *functionAnalysisState) setLocalSurfaceHandleOwner(name string, owner string) {
	if analysis == nil || name == "" {
		return
	}
	if analysis.surfaceHandleOwners == nil {
		analysis.surfaceHandleOwners = make(map[string]string)
	}
	if owner != "" {
		analysis.surfaceHandleOwners[name] = owner
		return
	}
	delete(analysis.surfaceHandleOwners, name)
}

func (analysis *functionAnalysisState) checkSurfaceFramePixelsUsable(name string, pos frontend.Position) error {
	frameName, ok := analysis.localSurfaceFramePixelsSource(name)
	if !ok || frameName == "" || analysis.surfacePresentedFrames == nil {
		return nil
	}
	if _, presented := analysis.surfacePresentedFrames[frameName]; !presented {
		return nil
	}
	return lifetimeDiagnosticf(pos, "surface frame pixels alias '%s' cannot be used after frame '%s' was presented; keep Frame.pixels local to the active Surface frame", name, frameName)
}

func (analysis *functionAnalysisState) underSecretControl() bool {
	return analysis != nil && analysis.secretControlDepth > 0
}

func (analysis *functionAnalysisState) withSecretControl(tainted bool, fn func() error) error {
	if analysis == nil || !tainted {
		return fn()
	}
	analysis.secretControlDepth++
	defer func() {
		analysis.secretControlDepth--
	}()
	return fn()
}

func (analysis *functionAnalysisState) markFunctionParamSecretTaint(funcName, paramName string) {
	if analysis == nil || funcName == "" || paramName == "" || analysis.funcParamSecretTaint == nil {
		return
	}
	params := analysis.funcParamSecretTaint[funcName]
	if params == nil {
		params = make(map[string]bool)
		analysis.funcParamSecretTaint[funcName] = params
	}
	if !params[paramName] {
		params[paramName] = true
		analysis.discoveredParamTaint = true
	}
}

func (analysis *functionAnalysisState) copySecretTaint() map[string]bool {
	if analysis == nil || len(analysis.secretTaint) == 0 {
		return make(map[string]bool)
	}
	out := make(map[string]bool, len(analysis.secretTaint))
	for name, tainted := range analysis.secretTaint {
		if tainted {
			out[name] = true
		}
	}
	return out
}

func (analysis *functionAnalysisState) restoreSecretTaint(snapshot map[string]bool) {
	if analysis == nil {
		return
	}
	analysis.secretTaint = copySecretTaintMap(snapshot)
}

func copySecretTaintMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return make(map[string]bool)
	}
	dst := make(map[string]bool, len(src))
	for name, tainted := range src {
		if tainted {
			dst[name] = true
		}
	}
	return dst
}

func mergeSecretTaintMaps(a, b map[string]bool) map[string]bool {
	merged := copySecretTaintMap(a)
	for name, tainted := range b {
		if tainted {
			merged[name] = true
		}
	}
	return merged
}

func validateDeferBodyControl(stmts []frontend.Stmt, loopDepth int) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			return fmt.Errorf("%s: return is not allowed in defer", frontend.FormatPos(s.At))
		case *frontend.ThrowStmt:
			return fmt.Errorf("%s: throw is not allowed in defer", frontend.FormatPos(s.At))
		case *frontend.DeferStmt:
			return fmt.Errorf("%s: nested defer is not allowed in defer", frontend.FormatPos(s.At))
		case *frontend.BreakStmt:
			if loopDepth == 0 {
				return fmt.Errorf("%s: break is not allowed in defer outside a cleanup-local loop", frontend.FormatPos(s.At))
			}
		case *frontend.ContinueStmt:
			if loopDepth == 0 {
				return fmt.Errorf("%s: continue is not allowed in defer outside a cleanup-local loop", frontend.FormatPos(s.At))
			}
		case *frontend.IfStmt:
			if err := validateDeferBodyControl(s.Then, loopDepth); err != nil {
				return err
			}
			if err := validateDeferBodyControl(s.Else, loopDepth); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := validateDeferBodyControl(s.Then, loopDepth); err != nil {
				return err
			}
			if err := validateDeferBodyControl(s.Else, loopDepth); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth+1); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth+1); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			for _, c := range s.Cases {
				if err := validateDeferBodyControl(c.Body, loopDepth); err != nil {
					return err
				}
			}
		case *frontend.IslandStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth); err != nil {
				return err
			}
		case *frontend.UnsafeStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkDeferBody(
	body []frontend.Stmt,
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
	savedRegionVars := copyRegionVars(state.regionVars)
	savedUnknownVars := copyBoolMap(state.unknownVars)
	savedUnknownConflicts := copyRegionConflictMap(state.unknownConflicts)
	savedReachable := state.reachable
	savedConsumedVars := copyConsumedVars(state.consumedVars)
	savedMaybeConsumedVars := copyOwnershipJoinConflicts(state.maybeConsumedVars)
	savedOwnershipAliases := copyStringMap(state.ownershipAliases)
	savedBorrowedPtrAliases := copyStringMap(state.borrowedPtrAliases)
	savedOwnedRegionSliceOwners := copyStringMap(state.ownedRegionSliceOwners)
	savedAwaitInvalidatedBorrow := copyPositionByIntMap(state.awaitInvalidatedBorrow)
	savedConsumedResources := copyConsumedResources(state.consumedResources)
	savedResourceVars := copyResourceVars(state.resourceVars)
	savedUnknownResources := copyUnknownResources(state.unknownResources)
	savedFinalizedResources := copyFinalizedResources(state.finalizedResources)
	savedSecretTaint := analysis.copySecretTaint()
	savedNextResourceID := state.nextResourceID
	savedReturnRegion := state.returnRegion
	savedReturnRegionSet := state.returnRegionSet
	savedReturnRegionSummary := cloneReturnRegionSummary(state.returnRegionSummary)
	savedLoopDepth := state.loopDepth
	savedUnsafeDepth := state.unsafeDepth
	savedAllowThrowDepth := state.allowThrowDepth
	savedAllowThrowCall := state.allowThrowCall
	savedAllowCatchDepth := state.allowCatchDepth
	savedAllowCatchCall := state.allowCatchCall
	savedAllowAwaitDepth := state.allowAwaitDepth
	savedAllowAwaitCall := state.allowAwaitCall
	defer func() {
		state.regionVars = savedRegionVars
		state.unknownVars = savedUnknownVars
		state.unknownConflicts = savedUnknownConflicts
		state.reachable = savedReachable
		state.consumedVars = savedConsumedVars
		state.maybeConsumedVars = savedMaybeConsumedVars
		state.ownershipAliases = savedOwnershipAliases
		state.borrowedPtrAliases = savedBorrowedPtrAliases
		state.ownedRegionSliceOwners = savedOwnedRegionSliceOwners
		state.awaitInvalidatedBorrow = savedAwaitInvalidatedBorrow
		state.consumedResources = savedConsumedResources
		state.resourceVars = savedResourceVars
		state.unknownResources = savedUnknownResources
		state.finalizedResources = savedFinalizedResources
		analysis.restoreSecretTaint(savedSecretTaint)
		state.nextResourceID = savedNextResourceID
		state.returnRegion = savedReturnRegion
		state.returnRegionSet = savedReturnRegionSet
		state.returnRegionSummary = savedReturnRegionSummary
		state.loopDepth = savedLoopDepth
		state.unsafeDepth = savedUnsafeDepth
		state.allowThrowDepth = savedAllowThrowDepth
		state.allowThrowCall = savedAllowThrowCall
		state.allowCatchDepth = savedAllowCatchDepth
		state.allowCatchCall = savedAllowCatchCall
		state.allowAwaitDepth = savedAllowAwaitDepth
		state.allowAwaitCall = savedAllowAwaitCall
	}()
	state.loopDepth = 0
	return checkStmts(body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
}

func copyBoolMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return make(map[string]bool)
	}
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return make(map[string]string)
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeBorrowedPtrAliases(a, b map[string]string) map[string]string {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]string)
	}
	merged := make(map[string]string)
	for name, owner := range a {
		if owner == "" {
			continue
		}
		merged[name] = owner
	}
	for name, owner := range b {
		if owner == "" {
			continue
		}
		if existing, exists := merged[name]; exists {
			if owner < existing {
				merged[name] = owner
			}
			continue
		}
		merged[name] = owner
	}
	return merged
}

func mergeOwnershipAliases(a, b map[string]string) map[string]string {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]string)
	}
	merged := make(map[string]string)
	for name, source := range a {
		if source == "" {
			continue
		}
		if rightSource, ok := b[name]; ok && rightSource == source {
			merged[name] = source
		}
	}
	return merged
}

func copyRegionConflictMap(src map[string]regionConflict) map[string]regionConflict {
	if len(src) == 0 {
		return make(map[string]regionConflict)
	}
	dst := make(map[string]regionConflict, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyConsumedVars(src map[string]frontend.Position) map[string]frontend.Position {
	if len(src) == 0 {
		return make(map[string]frontend.Position)
	}
	dst := make(map[string]frontend.Position, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyOwnershipJoinConflicts(src map[string]ownershipJoinConflict) map[string]ownershipJoinConflict {
	if len(src) == 0 {
		return make(map[string]ownershipJoinConflict)
	}
	dst := make(map[string]ownershipJoinConflict, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyConsumedResources(src map[int]frontend.Position) map[int]frontend.Position {
	if len(src) == 0 {
		return make(map[int]frontend.Position)
	}
	dst := make(map[int]frontend.Position, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyPositionByIntMap(src map[int]frontend.Position) map[int]frontend.Position {
	if len(src) == 0 {
		return make(map[int]frontend.Position)
	}
	dst := make(map[int]frontend.Position, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyResourceVars(src map[string]int) map[string]int {
	if len(src) == 0 {
		return make(map[string]int)
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyUnknownResources(src map[int]bool) map[int]bool {
	if len(src) == 0 {
		return make(map[int]bool)
	}
	dst := make(map[int]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyFinalizedResources(src map[int]resourceFinalization) map[int]resourceFinalization {
	if len(src) == 0 {
		return make(map[int]resourceFinalization)
	}
	dst := make(map[int]resourceFinalization, len(src))
	for k, v := range src {
		dst[k] = copyResourceFinalization(v)
	}
	return dst
}

func copyResourceFinalization(src resourceFinalization) resourceFinalization {
	dst := src
	if len(src.states) > 0 {
		dst.states = make(map[string]frontend.Position, len(src.states))
		for state, pos := range src.states {
			dst.states[state] = pos
		}
	}
	return dst
}

func mergeConsumedVars(a, b map[string]frontend.Position) map[string]frontend.Position {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]frontend.Position)
	}
	merged := make(map[string]frontend.Position)
	for name, left := range a {
		if right, ok := b[name]; ok {
			merged[name] = earliestPosition(left, right)
			continue
		}
		merged[name] = left
	}
	for name, right := range b {
		if _, exists := merged[name]; !exists {
			merged[name] = right
		}
	}
	return merged
}

func mergeMaybeConsumedVars(a, b flowSnapshot, leftLabel, rightLabel string) map[string]ownershipJoinConflict {
	// SAFE-003 incremental subset: model local consume states as SSA-like edge
	// joins. A value consumed on only some incoming edges remains unusable, but
	// diagnostics now distinguish maybe-consumed joins from linear local flow.
	merged := make(map[string]ownershipJoinConflict)
	names := make(map[string]struct{})
	for name := range a.consumedVars {
		names[name] = struct{}{}
	}
	for name := range b.consumedVars {
		names[name] = struct{}{}
	}
	for name := range a.maybeConsumedVars {
		names[name] = struct{}{}
	}
	for name := range b.maybeConsumedVars {
		names[name] = struct{}{}
	}
	for name := range names {
		leftConsumed, leftMaybe, leftPos, leftConflict := ownershipSnapshotConsumed(a, name)
		rightConsumed, rightMaybe, rightPos, rightConflict := ownershipSnapshotConsumed(b, name)
		if !leftConsumed && !rightConsumed {
			continue
		}
		if leftConsumed && rightConsumed && !leftMaybe && !rightMaybe {
			continue
		}
		conflict := ownershipJoinConflict{
			leftLabel:     leftLabel,
			leftConsumed:  leftConsumed,
			leftPos:       leftPos,
			rightLabel:    rightLabel,
			rightConsumed: rightConsumed,
			rightPos:      rightPos,
		}
		if leftMaybe {
			conflict.leftConsumed = true
			conflict.leftPos = ownershipJoinConflictPosition(leftConflict)
		}
		if rightMaybe {
			conflict.rightConsumed = true
			conflict.rightPos = ownershipJoinConflictPosition(rightConflict)
		}
		merged[name] = conflict
	}
	return merged
}

func ownershipSnapshotConsumed(snap flowSnapshot, name string) (bool, bool, frontend.Position, ownershipJoinConflict) {
	if conflict, ok := snap.maybeConsumedVars[name]; ok {
		return true, true, ownershipJoinConflictPosition(conflict), conflict
	}
	pos, ok := snap.consumedVars[name]
	return ok, false, pos, ownershipJoinConflict{}
}

func ownershipJoinConflictPosition(conflict ownershipJoinConflict) frontend.Position {
	switch {
	case conflict.leftConsumed && conflict.rightConsumed:
		return earliestPosition(conflict.leftPos, conflict.rightPos)
	case conflict.leftConsumed:
		return conflict.leftPos
	case conflict.rightConsumed:
		return conflict.rightPos
	default:
		return frontend.Position{}
	}
}

func mergeConsumedResources(a, b map[int]frontend.Position) map[int]frontend.Position {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]frontend.Position)
	}
	merged := make(map[int]frontend.Position)
	for id, left := range a {
		if right, ok := b[id]; ok {
			merged[id] = earliestPosition(left, right)
			continue
		}
		merged[id] = left
	}
	for id, right := range b {
		if _, exists := merged[id]; !exists {
			merged[id] = right
		}
	}
	return merged
}

func mergeFinalizedResources(a, b map[int]resourceFinalization, leftLabel, rightLabel string) map[int]resourceFinalization {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]resourceFinalization)
	}
	merged := make(map[int]resourceFinalization)
	ids := make(map[int]struct{})
	for id := range a {
		ids[id] = struct{}{}
	}
	for id := range b {
		ids[id] = struct{}{}
	}
	for id := range ids {
		left, leftOK := a[id]
		right, rightOK := b[id]
		if final, ok := mergeResourceFinalizationValues(left, leftOK, right, rightOK, leftLabel, rightLabel); ok {
			merged[id] = final
		}
	}
	return merged
}

func mergeResourceFinalizationValues(left resourceFinalization, leftOK bool, right resourceFinalization, rightOK bool, leftLabel, rightLabel string) (resourceFinalization, bool) {
	if !leftOK && !rightOK {
		return resourceFinalization{}, false
	}
	states := make(map[string]frontend.Position)
	mayBeAvailable := !leftOK || !rightOK
	addFinalizationStates(states, left)
	addFinalizationStates(states, right)
	if leftOK && left.mayBeAvailable {
		mayBeAvailable = true
	}
	if rightOK && right.mayBeAvailable {
		mayBeAvailable = true
	}
	if len(states) == 0 {
		return resourceFinalization{}, false
	}
	if len(states) == 1 && !mayBeAvailable && !left.maybe && !right.maybe {
		for state, pos := range states {
			return resourceFinalization{state: state, pos: pos}, true
		}
	}
	return resourceFinalization{
		state:          firstResourceFinalizationState(states),
		pos:            earliestResourceFinalizationPosition(states),
		maybe:          true,
		mayBeAvailable: mayBeAvailable,
		states:         states,
	}, true
}

func addFinalizationStates(dst map[string]frontend.Position, final resourceFinalization) {
	for state, pos := range resourceFinalizationStatePositions(final) {
		if existing, ok := dst[state]; ok {
			dst[state] = earliestPosition(existing, pos)
			continue
		}
		dst[state] = pos
	}
}

func firstResourceFinalizationState(states map[string]frontend.Position) string {
	first := ""
	for state := range states {
		if first == "" || state < first {
			first = state
		}
	}
	return first
}

func earliestResourceFinalizationPosition(states map[string]frontend.Position) frontend.Position {
	var earliest frontend.Position
	for _, pos := range states {
		earliest = earliestPosition(earliest, pos)
	}
	return earliest
}

func mergeUnknownResources(a, b map[int]bool) map[int]bool {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]bool)
	}
	merged := make(map[int]bool)
	for id, unknown := range a {
		if unknown {
			merged[id] = true
		}
	}
	for id, unknown := range b {
		if unknown {
			merged[id] = true
		}
	}
	return merged
}

func mergeAwaitInvalidatedBorrowRegions(a, b map[int]frontend.Position) map[int]frontend.Position {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]frontend.Position)
	}
	merged := copyPositionByIntMap(a)
	for regionID, pos := range b {
		if existing, exists := merged[regionID]; exists {
			merged[regionID] = earliestPosition(existing, pos)
			continue
		}
		merged[regionID] = pos
	}
	return merged
}

func mergeResourceVars(state *regionState, a, b map[string]int, consumed map[int]frontend.Position, finalized map[int]resourceFinalization, unknown map[int]bool, leftLabel, rightLabel string) map[string]int {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]int)
	}
	merged := make(map[string]int)
	for name, left := range a {
		right, ok := b[name]
		if !ok {
			merged[name] = left
			continue
		}
		if left == right {
			merged[name] = left
			continue
		}
		merged[name] = mergeResourceIDs(state, left, right, consumed, finalized, unknown, leftLabel, rightLabel)
	}
	for name, right := range b {
		if _, exists := merged[name]; !exists {
			merged[name] = right
		}
	}
	return merged
}

func mergeResourceIDs(state *regionState, left int, right int, consumed map[int]frontend.Position, finalized map[int]resourceFinalization, unknown map[int]bool, leftLabel, rightLabel string) int {
	if state == nil {
		return left
	}
	merged := state.allocateResourceID()
	leftParam, leftParamOK := state.resourceParamIndex[left]
	rightParam, rightParamOK := state.resourceParamIndex[right]
	leftPath := state.resourceParamPath[left]
	rightPath := state.resourceParamPath[right]
	if unknown[left] || unknown[right] {
		unknown[merged] = true
	} else if leftParamOK && rightParamOK && leftParam == rightParam && leftPath == rightPath {
		state.resourceParamIndex[merged] = leftParam
		state.resourceParamPath[merged] = leftPath
	} else {
		unknown[merged] = true
	}
	leftConsumed, leftConsumedOK := consumed[left]
	rightConsumed, rightConsumedOK := consumed[right]
	switch {
	case leftConsumedOK && rightConsumedOK:
		consumed[merged] = earliestPosition(leftConsumed, rightConsumed)
	case leftConsumedOK:
		consumed[merged] = leftConsumed
	case rightConsumedOK:
		consumed[merged] = rightConsumed
	}
	leftFinal, leftFinalOK := finalized[left]
	rightFinal, rightFinalOK := finalized[right]
	if final, ok := mergeResourceFinalizationValues(leftFinal, leftFinalOK, rightFinal, rightFinalOK, leftLabel, rightLabel); ok {
		finalized[merged] = final
	}
	return merged
}

func earliestPosition(a, b frontend.Position) frontend.Position {
	if a.Line == 0 {
		return b
	}
	if b.Line == 0 {
		return a
	}
	if a.Line < b.Line || (a.Line == b.Line && a.Col <= b.Col) {
		return a
	}
	return b
}

func earliestFinalization(a, b resourceFinalization) resourceFinalization {
	if earliestPosition(a.pos, b.pos) == a.pos {
		return a
	}
	return b
}
