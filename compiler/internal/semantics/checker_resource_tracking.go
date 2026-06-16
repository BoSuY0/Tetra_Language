package semantics

import (
	"strings"

	"tetra_language/compiler/internal/frontend"
	semanticsresources "tetra_language/compiler/internal/semantics/resources"
)

type flowSnapshot struct {
	reachable              bool
	consumedVars           map[string]frontend.Position
	maybeConsumedVars      map[string]ownershipJoinConflict
	ownershipAliases       map[string]string
	borrowedPtrAliases     map[string]string
	ownedRegionSliceOwners map[string]string
	awaitInvalidatedBorrow map[int]frontend.Position
	consumedResources      map[int]frontend.Position
	resourceVars           map[string]int
	unknownResources       map[int]bool
	finalizedResources     map[int]resourceFinalization
}

type loopFlowExit struct {
	label string
	vars  map[string]int
	flow  flowSnapshot
	taint map[string]bool
}

type loopFlowFrame struct {
	breaks    []loopFlowExit
	continues []loopFlowExit
}

func snapshotFlow(state *regionState) flowSnapshot {
	return flowSnapshot{
		reachable:              state.reachable,
		consumedVars:           copyConsumedVars(state.consumedVars),
		maybeConsumedVars:      copyOwnershipJoinConflicts(state.maybeConsumedVars),
		ownershipAliases:       copyStringMap(state.ownershipAliases),
		borrowedPtrAliases:     copyStringMap(state.borrowedPtrAliases),
		ownedRegionSliceOwners: copyStringMap(state.ownedRegionSliceOwners),
		awaitInvalidatedBorrow: copyPositionByIntMap(state.awaitInvalidatedBorrow),
		consumedResources:      copyConsumedResources(state.consumedResources),
		resourceVars:           copyResourceVars(state.resourceVars),
		unknownResources:       copyUnknownResources(state.unknownResources),
		finalizedResources:     copyFinalizedResources(state.finalizedResources),
	}
}

func restoreFlow(state *regionState, snap flowSnapshot) {
	state.reachable = snap.reachable
	state.consumedVars = copyConsumedVars(snap.consumedVars)
	state.maybeConsumedVars = copyOwnershipJoinConflicts(snap.maybeConsumedVars)
	state.ownershipAliases = copyStringMap(snap.ownershipAliases)
	state.borrowedPtrAliases = copyStringMap(snap.borrowedPtrAliases)
	state.ownedRegionSliceOwners = copyStringMap(snap.ownedRegionSliceOwners)
	state.awaitInvalidatedBorrow = copyPositionByIntMap(snap.awaitInvalidatedBorrow)
	state.consumedResources = copyConsumedResources(snap.consumedResources)
	state.resourceVars = copyResourceVars(snap.resourceVars)
	state.unknownResources = copyUnknownResources(snap.unknownResources)
	state.finalizedResources = copyFinalizedResources(snap.finalizedResources)
}

func mergeFlow(state *regionState, a, b flowSnapshot) {
	mergeFlowWithLabels(state, a, b, "left", "right")
}

func mergeFlowWithLabels(state *regionState, a, b flowSnapshot, leftLabel, rightLabel string) {
	if !a.reachable && !b.reachable {
		restoreFlow(state, a)
		state.reachable = false
		return
	}
	if !a.reachable {
		restoreFlow(state, b)
		return
	}
	if !b.reachable {
		restoreFlow(state, a)
		return
	}
	consumedResources := mergeConsumedResources(a.consumedResources, b.consumedResources)
	finalizedResources := mergeFinalizedResources(a.finalizedResources, b.finalizedResources, leftLabel, rightLabel)
	unknownResources := mergeUnknownResources(a.unknownResources, b.unknownResources)
	state.reachable = true
	state.consumedVars = mergeConsumedVars(a.consumedVars, b.consumedVars)
	state.maybeConsumedVars = mergeMaybeConsumedVars(a, b, leftLabel, rightLabel)
	state.ownershipAliases = mergeOwnershipAliases(a.ownershipAliases, b.ownershipAliases)
	state.borrowedPtrAliases = mergeBorrowedPtrAliases(a.borrowedPtrAliases, b.borrowedPtrAliases)
	state.ownedRegionSliceOwners = mergeOwnershipAliases(a.ownedRegionSliceOwners, b.ownedRegionSliceOwners)
	state.awaitInvalidatedBorrow = mergeAwaitInvalidatedBorrowRegions(a.awaitInvalidatedBorrow, b.awaitInvalidatedBorrow)
	state.consumedResources = consumedResources
	state.unknownResources = unknownResources
	state.finalizedResources = finalizedResources
	state.resourceVars = mergeResourceVars(state, a.resourceVars, b.resourceVars, consumedResources, finalizedResources, unknownResources, leftLabel, rightLabel)
}

func pushLoopFlowFrame(state *regionState) {
	if state == nil {
		return
	}
	state.loopFlowFrames = append(state.loopFlowFrames, loopFlowFrame{})
}

func popLoopFlowFrame(state *regionState) loopFlowFrame {
	if state == nil || len(state.loopFlowFrames) == 0 {
		return loopFlowFrame{}
	}
	frame := state.loopFlowFrames[len(state.loopFlowFrames)-1]
	state.loopFlowFrames = state.loopFlowFrames[:len(state.loopFlowFrames)-1]
	return frame
}

func recordLoopFlowExit(state *regionState, label string, analysis *functionAnalysisState) {
	if state == nil || len(state.loopFlowFrames) == 0 {
		return
	}
	exit := loopFlowExit{
		label: label,
		vars:  copyRegionVars(state.regionVars),
		flow:  snapshotFlow(state),
	}
	if analysis != nil {
		exit.taint = analysis.copySecretTaint()
	}
	frame := &state.loopFlowFrames[len(state.loopFlowFrames)-1]
	if label == "break" {
		frame.breaks = append(frame.breaks, exit)
		return
	}
	frame.continues = append(frame.continues, exit)
}

func mergeLoopFlowExits(state *regionState, analysis *functionAnalysisState, exits []loopFlowExit) {
	if len(exits) == 0 {
		return
	}
	mergedVars := copyRegionVars(exits[0].vars)
	mergedFlow := exits[0].flow
	mergedTaint := cloneBoolMap(exits[0].taint)
	labels := []string{exits[0].label}
	for _, exit := range exits[1:] {
		leftLabel := strings.Join(labels, "/")
		mergeControlFlowWithLabels(state, analysis, mergedVars, mergedFlow, mergedTaint, exit.vars, exit.flow, exit.taint, leftLabel, exit.label)
		mergedVars = copyRegionVars(state.regionVars)
		mergedFlow = snapshotFlow(state)
		if analysis != nil {
			mergedTaint = analysis.copySecretTaint()
		}
		labels = append(labels, exit.label)
	}
	state.regionVars = mergedVars
	restoreFlow(state, mergedFlow)
	if analysis != nil {
		analysis.restoreSecretTaint(mergedTaint)
	}
}

func mergeControlFlowWithLabels(
	state *regionState,
	analysis *functionAnalysisState,
	leftVars map[string]int,
	leftFlow flowSnapshot,
	leftTaint map[string]bool,
	rightVars map[string]int,
	rightFlow flowSnapshot,
	rightTaint map[string]bool,
	leftLabel string,
	rightLabel string,
) {
	switch {
	case !leftFlow.reachable && !rightFlow.reachable:
		state.regionVars = copyRegionVars(leftVars)
		restoreFlow(state, leftFlow)
		state.reachable = false
		analysis.restoreSecretTaint(mergeSecretTaintMaps(leftTaint, rightTaint))
	case !leftFlow.reachable:
		state.regionVars = copyRegionVars(rightVars)
		restoreFlow(state, rightFlow)
		analysis.restoreSecretTaint(rightTaint)
	case !rightFlow.reachable:
		state.regionVars = copyRegionVars(leftVars)
		restoreFlow(state, leftFlow)
		analysis.restoreSecretTaint(leftTaint)
	default:
		state.regionVars = mergeRegionVars(leftVars, rightVars)
		mergeFlowWithLabels(state, leftFlow, rightFlow, leftLabel, rightLabel)
		analysis.restoreSecretTaint(mergeSecretTaintMaps(leftTaint, rightTaint))
		recordMergeConflicts(state, leftVars, rightVars, leftLabel, rightLabel)
	}
}

type resourceSourceResult struct {
	name      string
	known     bool
	ambiguous bool
	unknown   bool
}

func bindResourceFromExpr(
	name string,
	typeName string,
	expr frontend.Expr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if !isResourceHandleType(typeName) {
		state.bindResource(name, "", false)
		return nil
	}
	if typeName == surfaceSurfaceTypeName {
		if owner, ok := surfaceConstructedHandleOwnerPathExpr(expr); ok {
			if _, exists := state.resourceID(owner); exists {
				state.bindResource(name, owner, true)
				return nil
			}
		}
	}
	source, err := resourceSourceForExpr(expr, funcs, module, imports, state)
	if err != nil {
		return err
	}
	if source.ambiguous {
		return ownershipDiagnosticf(expr.Pos(), "resource expression mixes resource provenance")
	}
	if source.unknown {
		state.bindUnknownResource(name)
		return nil
	}
	sourceName := ""
	if source.known {
		sourceName = source.name
		if _, consumed := state.consumedAt(sourceName); consumed {
			state.bindTransferredResource(name, sourceName)
			return nil
		}
	}
	state.bindResource(name, sourceName, true)
	return nil
}

func bindResourceTreeFromExpr(
	name string,
	typeName string,
	expr frontend.Expr,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || name == "" {
		return nil
	}
	state.clearResourceTree(name)
	if !typeContainsResourceHandle(typeName, types) {
		state.bindResource(name, "", false)
		return nil
	}
	if isResourceHandleType(typeName) {
		return bindResourceFromExpr(name, typeName, expr, funcs, module, imports, state)
	}
	if info, ok := types[typeName]; ok && info.Kind == TypeOptional {
		if sourcePrefix, ok := resourcePathForExpr(expr); ok && resourceTreeHasPath(sourcePrefix, info.ElemType, types, state) {
			copyResourceTreeFromPath(resourceFieldPath(name, "$elem"), sourcePrefix, info.ElemType, types, state)
			return nil
		}
	}
	if sourcePrefix, ok := resourcePathForExpr(expr); ok {
		copyResourceTreeFromPath(name, sourcePrefix, typeName, types, state)
		return nil
	}
	switch e := expr.(type) {
	case *frontend.TryExpr:
		return bindResourceTreeFromExpr(name, typeName, e.X, funcs, types, module, imports, state)
	case *frontend.AwaitExpr:
		return bindResourceTreeFromExpr(name, typeName, e.X, funcs, types, module, imports, state)
	case *frontend.MatchExpr:
		if e.ResultLocal != "" {
			copyResourceTreeFromPath(name, e.ResultLocal, typeName, types, state)
			return nil
		}
	case *frontend.StructLitExpr:
		info, ok := types[typeName]
		if !ok || info.Kind != TypeStruct {
			markResourceTreeUnknown(name, typeName, types, state)
			return nil
		}
		byName := make(map[string]frontend.Expr, len(e.Fields))
		for _, field := range e.Fields {
			byName[field.Name] = field.Value
		}
		for _, field := range info.Fields {
			value := byName[field.Name]
			if value == nil {
				continue
			}
			if err := bindResourceTreeFromExpr(resourceFieldPath(name, field.Name), field.TypeName, value, funcs, types, module, imports, state); err != nil {
				return err
			}
		}
		return nil
	case *frontend.CallExpr:
		if info, ok := types[typeName]; ok && info.Kind == TypeStruct && e.Name == typeName {
			for i, field := range info.Fields {
				if i >= len(e.Args) {
					break
				}
				if err := bindResourceTreeFromExpr(resourceFieldPath(name, field.Name), field.TypeName, e.Args[i], funcs, types, module, imports, state); err != nil {
					return err
				}
			}
			return nil
		}
		resolved, err := resolveCheckedCallName(e.Name, funcs, module, imports, e.At)
		if err != nil {
			return err
		}
		if resolved == "core.recv_typed" {
			bindFreshResourceTree(name, typeName, types, state)
			return nil
		}
		enumType, caseInfo, ok, err := resolveEnumCaseConstructorCall(e, types, module, imports)
		if err != nil {
			return err
		}
		if ok && enumType == typeName {
			for i, arg := range e.Args {
				if i >= len(caseInfo.PayloadTypes) {
					break
				}
				if err := bindResourceTreeFromExpr(resourceEnumPayloadPath(name, caseInfo.Ordinal, i), caseInfo.PayloadTypes[i], arg, funcs, types, module, imports, state); err != nil {
					return err
				}
			}
			return nil
		}
		sig, ok := funcs[resolved]
		if !ok {
			return nil
		}
		if handled, err := bindResourceTreeFromCallSummary(name, typeName, e, sig, funcs, types, module, imports, state); handled || err != nil {
			return err
		}
	}
	markResourceTreeUnknown(name, typeName, types, state)
	return nil
}

func bindOwnedRegionSliceOwnerFromExpr(
	name string,
	typeName string,
	expr frontend.Expr,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || name == "" {
		return nil
	}
	info, ok := types[typeName]
	if !ok {
		state.clearOwnedRegionSliceOwnerTree(name)
		return nil
	}
	switch info.Kind {
	case TypeSlice:
		owner := ownedRegionSliceOwnerForExpr(expr, state)
		if owner == "" {
			state.clearOwnedRegionSliceOwnerTree(name)
			return nil
		}
		state.bindOwnedRegionSliceOwner(name, owner)
		return nil
	case TypeStruct:
		state.clearOwnedRegionSliceOwnerTree(name)
		if sourcePrefix, ok := resourcePathForExpr(expr); ok {
			copyOwnedRegionSliceOwnerTreeFromPath(name, sourcePrefix, typeName, types, state)
			return nil
		}
		if lit, ok := expr.(*frontend.StructLitExpr); ok {
			byName := make(map[string]frontend.Expr, len(lit.Fields))
			for _, field := range lit.Fields {
				byName[field.Name] = field.Value
			}
			for _, field := range info.Fields {
				value := byName[field.Name]
				if value == nil {
					continue
				}
				if err := bindOwnedRegionSliceOwnerFromExpr(resourceFieldPath(name, field.Name), field.TypeName, value, types, module, imports, state); err != nil {
					return err
				}
			}
			return nil
		}
		if call, ok := expr.(*frontend.CallExpr); ok && call.Name == typeName {
			for i, field := range info.Fields {
				if i >= len(call.Args) {
					break
				}
				if err := bindOwnedRegionSliceOwnerFromExpr(resourceFieldPath(name, field.Name), field.TypeName, call.Args[i], types, module, imports, state); err != nil {
					return err
				}
			}
		}
	case TypeEnum:
		state.clearOwnedRegionSliceOwnerTree(name)
		if sourcePrefix, ok := resourcePathForExpr(expr); ok {
			copyOwnedRegionSliceOwnerTreeFromPath(name, sourcePrefix, typeName, types, state)
			return nil
		}
		call, ok := expr.(*frontend.CallExpr)
		if !ok {
			return nil
		}
		enumType, caseInfo, found, err := resolveEnumCaseConstructorCall(call, types, module, imports)
		if err != nil {
			return err
		}
		if !found || enumType != typeName {
			return nil
		}
		for i, arg := range call.Args {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			if err := bindOwnedRegionSliceOwnerFromExpr(resourceEnumPayloadPath(name, caseInfo.Ordinal, i), caseInfo.PayloadTypes[i], arg, types, module, imports, state); err != nil {
				return err
			}
		}
	case TypeOptional:
		state.clearOwnedRegionSliceOwnerTree(name)
		call, ok := expr.(*frontend.CallExpr)
		if !ok || len(call.Args) == 0 {
			return nil
		}
		return bindOwnedRegionSliceOwnerFromExpr(resourceFieldPath(name, "$elem"), info.ElemType, call.Args[0], types, module, imports, state)
	default:
		state.clearOwnedRegionSliceOwnerTree(name)
	}
	return nil
}

func copyOwnedRegionSliceOwnerTreeFromPath(dst string, src string, typeName string, types map[string]*TypeInfo, state *regionState) {
	if state == nil || dst == "" || src == "" {
		return
	}
	for _, leaf := range regionLeafPaths(typeName, types, "") {
		srcLeaf := joinResourcePath(src, leaf)
		if owner, ok := state.ownedRegionSliceOwner(srcLeaf); ok {
			state.bindOwnedRegionSliceOwner(joinResourcePath(dst, leaf), owner)
		}
	}
}

func ownedRegionSliceOwnerForExpr(expr frontend.Expr, state *regionState) string {
	if state == nil || expr == nil || isExplicitCopyExpr(expr) {
		return ""
	}
	if path, ok := resourcePathForExpr(expr); ok {
		if owner, found := state.ownedRegionSliceOwner(path); found {
			return owner
		}
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || len(call.Args) == 0 {
		return ""
	}
	name := call.Name
	if target, ok := ResolveBuiltinAlias(name); ok {
		name = target
	}
	switch name {
	case "core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool":
		if owner, ok := resourcePathForExpr(call.Args[0]); ok {
			return owner
		}
	}
	return ""
}

func bindResourceTreeFromCallSummary(
	name string,
	typeName string,
	call *frontend.CallExpr,
	sig FuncSig,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (bool, error) {
	if len(sig.ReturnResourceSummary) == 0 {
		return false, nil
	}
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		dstLeaf := joinResourcePath(name, leaf)
		provenances := sig.ReturnResourceSummary[leaf]
		if len(provenances) == 0 {
			state.bindResource(dstLeaf, "", true)
			continue
		}
		if len(provenances) > 1 {
			state.bindUnknownResource(dstLeaf)
			continue
		}
		source, err := resourceSourceForCallProvenance(call.Args, sig, provenances[0], funcs, types, module, imports, state, call.At)
		if err != nil {
			return true, err
		}
		if source.ambiguous || source.unknown || !source.known {
			state.bindUnknownResource(dstLeaf)
			continue
		}
		if _, consumed := state.consumedAt(source.name); consumed {
			state.bindTransferredResource(dstLeaf, source.name)
			continue
		}
		state.bindResource(dstLeaf, source.name, true)
	}
	return true, nil
}

func bindResourceTreeFromPathOrUnknown(dst string, src string, typeName string, types map[string]*TypeInfo, state *regionState) {
	if !typeContainsResourceHandle(typeName, types) {
		state.bindResource(dst, "", false)
		return
	}
	copyResourceTreeFromPath(dst, src, typeName, types, state)
}

func bindPatternResourceLocals(
	pattern frontend.Expr,
	fallbackName string,
	scrutineePath string,
	scrutType string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || scrutineePath == "" || !typeContainsResourceHandle(scrutType, types) {
		return nil
	}
	info, ok := types[scrutType]
	if !ok {
		return nil
	}
	if pattern == nil {
		if fallbackName == "" || info.Kind != TypeOptional {
			return nil
		}
		bindResourceTreeFromPathOrUnknown(fallbackName, resourceFieldPath(scrutineePath, "$elem"), info.ElemType, types, state)
		return nil
	}
	switch p := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return nil
		}
		bindResourceTreeFromPathOrUnknown(p.Name, resourceFieldPath(scrutineePath, "$elem"), info.ElemType, types, state)
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(p, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return nil
		}
		for i, binding := range p.Bindings {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			bindResourceTreeFromPathOrUnknown(binding, resourceEnumPayloadPath(scrutineePath, caseInfo.Ordinal, i), caseInfo.PayloadTypes[i], types, state)
		}
	}
	return nil
}

func bindPatternOwnershipAliases(
	pattern frontend.Expr,
	fallbackName string,
	scrutineePath string,
	scrutType string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || scrutineePath == "" {
		return nil
	}
	info, ok := types[scrutType]
	if !ok {
		return nil
	}
	if pattern == nil {
		if fallbackName == "" || info.Kind != TypeOptional {
			return nil
		}
		state.bindOwnershipAlias(fallbackName, resourceFieldPath(scrutineePath, "$elem"))
		return nil
	}
	switch p := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind == TypeOptional {
			state.bindOwnershipAlias(p.Name, resourceFieldPath(scrutineePath, "$elem"))
		}
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(p, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return nil
		}
		for i, binding := range p.Bindings {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			state.bindOwnershipAlias(binding, resourceEnumPayloadPath(scrutineePath, caseInfo.Ordinal, i))
		}
	}
	return nil
}

func bindPatternBorrowedPtrAliases(
	pattern frontend.Expr,
	fallbackName string,
	scrutineePath string,
	scrutType string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || scrutineePath == "" {
		return nil
	}
	info, ok := types[scrutType]
	if !ok {
		return nil
	}
	if pattern == nil {
		if fallbackName == "" || info.Kind != TypeOptional {
			return nil
		}
		copyBorrowedPtrAliasesFromPath(fallbackName, resourceFieldPath(scrutineePath, "$elem"), info.ElemType, types, state)
		return nil
	}
	switch p := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind == TypeOptional {
			copyBorrowedPtrAliasesFromPath(p.Name, resourceFieldPath(scrutineePath, "$elem"), info.ElemType, types, state)
		}
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(p, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return nil
		}
		for i, binding := range p.Bindings {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			copyBorrowedPtrAliasesFromPath(binding, resourceEnumPayloadPath(scrutineePath, caseInfo.Ordinal, i), caseInfo.PayloadTypes[i], types, state)
		}
	}
	return nil
}

func copyBorrowedPtrAliasesFromPath(dst string, src string, typeName string, types map[string]*TypeInfo, state *regionState) {
	if state == nil || dst == "" || src == "" || !typeMayContainPtr(typeName, types) {
		return
	}
	state.clearBorrowedPtrAliasTree(dst)
	for _, leaf := range ptrLeafPaths(typeName, types, "") {
		srcLeaf := joinResourcePath(src, leaf)
		if owner, borrowed := state.borrowedPtrAliasOwner(srcLeaf); borrowed {
			state.bindBorrowedPtrAlias(joinResourcePath(dst, leaf), owner)
		}
	}
}

func bindPatternRegionLocals(
	pattern frontend.Expr,
	fallbackName string,
	scrutineePath string,
	scrutType string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || scrutineePath == "" || !typeMayContainRegion(scrutType, types) {
		return nil
	}
	info, ok := types[scrutType]
	if !ok {
		return nil
	}
	if pattern == nil {
		if fallbackName == "" || info.Kind != TypeOptional {
			return nil
		}
		copyRegionTreeFromPath(fallbackName, resourceFieldPath(scrutineePath, "$elem"), info.ElemType, types, state)
		return nil
	}
	switch p := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return nil
		}
		copyRegionTreeFromPath(p.Name, resourceFieldPath(scrutineePath, "$elem"), info.ElemType, types, state)
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(p, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return nil
		}
		for i, binding := range p.Bindings {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			copyRegionTreeFromPath(binding, resourceEnumPayloadPath(scrutineePath, caseInfo.Ordinal, i), caseInfo.PayloadTypes[i], types, state)
		}
	}
	return nil
}

func copyResourceTreeFromPath(dst string, src string, typeName string, types map[string]*TypeInfo, state *regionState) {
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		dstLeaf := joinResourcePath(dst, leaf)
		srcLeaf := joinResourcePath(src, leaf)
		if id, ok := state.resourceID(srcLeaf); ok && !state.resourceUnknown(srcLeaf) {
			if _, consumed := state.consumedAt(srcLeaf); consumed {
				state.bindTransferredResource(dstLeaf, srcLeaf)
				continue
			}
			state.resourceVars[dstLeaf] = id
			continue
		}
		state.bindUnknownResource(dstLeaf)
	}
}

func resourceTreeHasPath(prefix string, typeName string, types map[string]*TypeInfo, state *regionState) bool {
	if state == nil || prefix == "" {
		return false
	}
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		if _, ok := state.resourceID(joinResourcePath(prefix, leaf)); ok {
			return true
		}
	}
	return false
}

func markResourceTreeUnknown(prefix string, typeName string, types map[string]*TypeInfo, state *regionState) {
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		state.bindUnknownResource(joinResourcePath(prefix, leaf))
	}
}

func bindFreshResourceTree(prefix string, typeName string, types map[string]*TypeInfo, state *regionState) {
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		state.bindResource(joinResourcePath(prefix, leaf), "", true)
	}
}

func resourceLeafPaths(typeName string, types map[string]*TypeInfo, prefix string) []string {
	return resourceLeafPathsVisiting(typeName, types, prefix, map[string]bool{})
}

func ptrLeafPaths(typeName string, types map[string]*TypeInfo, prefix string) []string {
	return ptrLeafPathsVisiting(typeName, types, prefix, map[string]bool{})
}

func resourceLeafPathsVisiting(typeName string, types map[string]*TypeInfo, prefix string, visiting map[string]bool) []string {
	if typeName == surfaceFrameTypeName {
		return nil
	}
	if isResourceHandleType(typeName) {
		return []string{prefix}
	}
	info, ok := types[typeName]
	if !ok {
		return nil
	}
	var out []string
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			out = append(out, resourceLeafPathsVisiting(field.TypeName, types, resourceFieldPath(prefix, field.Name), visiting)...)
		}
	case TypeEnum:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, c := range info.EnumCases {
			for i, payload := range c.PayloadTypes {
				out = append(out, resourceLeafPathsVisiting(payload, types, resourceEnumPayloadPath(prefix, c.Ordinal, i), visiting)...)
			}
		}
	case TypeArray, TypeOptional:
		out = append(out, resourceLeafPathsVisiting(info.ElemType, types, resourceFieldPath(prefix, "$elem"), visiting)...)
	}
	return out
}

func ptrLeafPathsVisiting(typeName string, types map[string]*TypeInfo, prefix string, visiting map[string]bool) []string {
	if typeName == "ptr" {
		return []string{prefix}
	}
	info, ok := types[typeName]
	if !ok {
		return nil
	}
	var out []string
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			out = append(out, ptrLeafPathsVisiting(field.TypeName, types, resourceFieldPath(prefix, field.Name), visiting)...)
		}
	case TypeEnum:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, c := range info.EnumCases {
			for i, payload := range c.PayloadTypes {
				out = append(out, ptrLeafPathsVisiting(payload, types, resourceEnumPayloadPath(prefix, c.Ordinal, i), visiting)...)
			}
		}
	case TypeArray, TypeOptional:
		out = append(out, ptrLeafPathsVisiting(info.ElemType, types, resourceFieldPath(prefix, "$elem"), visiting)...)
	}
	return out
}

func resourcePathForExpr(expr frontend.Expr) (string, bool) {
	return semanticsresources.PathForExpr(expr)
}

func resourceFieldPath(prefix string, field string) string {
	return semanticsresources.FieldPath(prefix, field)
}

func resourceEnumPayloadPath(prefix string, ordinal int32, index int) string {
	return semanticsresources.EnumPayloadPath(prefix, ordinal, index)
}

func joinResourcePath(prefix string, leaf string) string {
	return semanticsresources.JoinPath(prefix, leaf)
}

func resourceSourceForPath(path string, state *regionState) resourceSourceResult {
	if state == nil || path == "" {
		return resourceSourceResult{}
	}
	if _, ok := state.resourceID(path); !ok {
		return resourceSourceResult{}
	}
	if state.resourceUnknown(path) {
		return resourceSourceResult{unknown: true}
	}
	return resourceSourceResult{name: path, known: true}
}

func returnResourceSummaryForExpr(
	expr frontend.Expr,
	typeName string,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (ReturnResourceSummary, bool, error) {
	if state == nil || !typeContainsResourceHandle(typeName, types) {
		return nil, false, nil
	}
	summary := ReturnResourceSummary{}
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		source, err := resourceSourceForExprLeaf(expr, typeName, leaf, funcs, types, module, imports, state)
		if err != nil {
			return nil, false, err
		}
		if source.ambiguous {
			return nil, false, ownershipDiagnosticf(expr.Pos(), "resource expression mixes resource provenance")
		}
		if source.unknown {
			return nil, true, nil
		}
		if !source.known {
			continue
		}
		paramIndex, paramPath, ok := state.resourceParamOwner(source.name)
		if !ok {
			continue
		}
		summary[leaf] = appendResourceProvenance(summary[leaf], ResourceProvenance{
			ParamIndex: paramIndex,
			ParamPath:  paramPath,
		})
	}
	if len(summary) == 0 {
		return nil, false, nil
	}
	return summary, false, nil
}

func appendResourceProvenance(in []ResourceProvenance, provenance ResourceProvenance) []ResourceProvenance {
	for _, existing := range in {
		if existing == provenance {
			return in
		}
	}
	return append(in, provenance)
}

func bindCatchErrorResourceSummary(
	errorLocal string,
	call *frontend.CallExpr,
	sig FuncSig,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || errorLocal == "" || call == nil || len(sig.ThrowResourceSummary) == 0 {
		return nil
	}
	state.clearResourceTree(errorLocal)
	for leaf, provenances := range sig.ThrowResourceSummary {
		var merged resourceSourceResult
		set := false
		for _, provenance := range provenances {
			source, err := resourceSourceForCallProvenance(call.Args, sig, provenance, funcs, types, module, imports, state, call.At)
			if err != nil {
				return err
			}
			if !set {
				merged = source
				set = true
				continue
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		dst := joinResourcePath(errorLocal, leaf)
		if !set || merged.unknown {
			state.bindUnknownResource(dst)
			continue
		}
		if merged.ambiguous {
			return ownershipDiagnosticf(call.At, "resource expression mixes resource provenance")
		}
		if !merged.known {
			continue
		}
		if _, consumed := state.consumedAt(merged.name); consumed {
			state.bindTransferredResource(dst, merged.name)
			continue
		}
		state.bindResource(dst, merged.name, true)
	}
	return nil
}

func recordTryCallThrowResourceSummary(
	call *frontend.CallExpr,
	sig FuncSig,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || call == nil || len(sig.ThrowResourceSummary) == 0 || !typeContainsResourceHandle(sig.ThrowsType, types) {
		return nil
	}
	summary := ReturnResourceSummary{}
	for leaf, provenances := range sig.ThrowResourceSummary {
		for _, provenance := range provenances {
			source, err := resourceSourceForCallProvenance(call.Args, sig, provenance, funcs, types, module, imports, state, call.At)
			if err != nil {
				return err
			}
			if source.ambiguous {
				return ownershipDiagnosticf(call.At, "resource expression mixes resource provenance")
			}
			if source.unknown || !source.known {
				continue
			}
			paramIndex, paramPath, ok := state.resourceParamOwner(source.name)
			if !ok {
				continue
			}
			summary[leaf] = appendResourceProvenance(summary[leaf], ResourceProvenance{
				ParamIndex: paramIndex,
				ParamPath:  paramPath,
			})
		}
	}
	if len(summary) == 0 {
		return nil
	}
	return state.recordThrowResourceSummary(summary, call.At)
}

func resourceSourceForExprLeaf(
	expr frontend.Expr,
	typeName string,
	leaf string,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (resourceSourceResult, error) {
	if leaf == "" {
		return resourceSourceForExpr(expr, funcs, module, imports, state)
	}
	if sourcePrefix, ok := resourcePathForExpr(expr); ok {
		source := resourceSourceForPath(joinResourcePath(sourcePrefix, leaf), state)
		if !source.known && !source.unknown {
			if wrapped, handled, err := resourceSourceForOptionalWrappedLeaf(expr, typeName, leaf, funcs, types, module, imports, state); handled || err != nil {
				return wrapped, err
			}
			return resourceSourceResult{unknown: true}, nil
		}
		return source, nil
	}
	switch e := expr.(type) {
	case *frontend.TryExpr:
		return resourceSourceForExprLeaf(e.X, typeName, leaf, funcs, types, module, imports, state)
	case *frontend.AwaitExpr:
		return resourceSourceForExprLeaf(e.X, typeName, leaf, funcs, types, module, imports, state)
	case *frontend.StructLitExpr:
		return resourceSourceForStructFieldLeaf(e.Fields, typeName, leaf, funcs, types, module, imports, state)
	case *frontend.CallExpr:
		if info, ok := types[typeName]; ok && info.Kind == TypeStruct && e.Name == typeName {
			fields := make([]frontend.StructFieldInit, 0, len(info.Fields))
			for i, field := range info.Fields {
				if i >= len(e.Args) {
					break
				}
				fields = append(fields, frontend.StructFieldInit{Name: field.Name, Value: e.Args[i]})
			}
			return resourceSourceForStructFieldLeaf(fields, typeName, leaf, funcs, types, module, imports, state)
		}
		if source, handled, err := resourceSourceForEnumConstructorLeaf(e, typeName, leaf, funcs, types, module, imports, state); handled || err != nil {
			return source, err
		}
		resolved, err := resolveCheckedCallName(e.Name, funcs, module, imports, e.At)
		if err != nil {
			return resourceSourceResult{}, err
		}
		sig, ok := funcs[resolved]
		if !ok {
			return resourceSourceResult{}, nil
		}
		if sig.ReturnResourceParam == regionUnknown && len(sig.ReturnResourceSummary) == 0 {
			return resourceSourceResult{unknown: true}, nil
		}
		provenances := sig.ReturnResourceSummary[leaf]
		if len(provenances) == 0 {
			return resourceSourceResult{}, nil
		}
		var merged resourceSourceResult
		for i, provenance := range provenances {
			source, err := resourceSourceForCallProvenance(e.Args, sig, provenance, funcs, types, module, imports, state, e.At)
			if err != nil {
				return resourceSourceResult{}, err
			}
			if i == 0 {
				merged = source
				continue
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		return merged, nil
	case *frontend.MatchExpr:
		var merged resourceSourceResult
		set := false
		for _, c := range e.Cases {
			source, err := resourceSourceForExprLeaf(c.Value, typeName, leaf, funcs, types, module, imports, state)
			if err != nil {
				return resourceSourceResult{}, err
			}
			if !set {
				merged = source
				set = true
				continue
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		return merged, nil
	case *frontend.CatchExpr:
		merged, err := resourceSourceForExprLeaf(e.Call, typeName, leaf, funcs, types, module, imports, state)
		if err != nil {
			return resourceSourceResult{}, err
		}
		for _, c := range e.Cases {
			source, err := resourceSourceForExprLeaf(c.Value, typeName, leaf, funcs, types, module, imports, state)
			if err != nil {
				return resourceSourceResult{}, err
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		return merged, nil
	default:
		return resourceSourceResult{}, nil
	}
}

func resourceSourceForOptionalWrappedLeaf(
	expr frontend.Expr,
	typeName string,
	leaf string,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (resourceSourceResult, bool, error) {
	info, ok := types[typeName]
	if !ok || info.Kind != TypeOptional {
		return resourceSourceResult{}, false, nil
	}
	tail, ok := resourceLeafTail(leaf, "$elem")
	if !ok {
		return resourceSourceResult{}, false, nil
	}
	source, err := resourceSourceForExprLeaf(expr, info.ElemType, tail, funcs, types, module, imports, state)
	if err != nil {
		return resourceSourceResult{}, true, err
	}
	if source.known || source.unknown || source.ambiguous {
		return source, true, nil
	}
	return resourceSourceResult{}, false, nil
}

func resourceSourceForStructFieldLeaf(
	fields []frontend.StructFieldInit,
	typeName string,
	leaf string,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (resourceSourceResult, error) {
	info, ok := types[typeName]
	if !ok || info.Kind != TypeStruct {
		return resourceSourceResult{}, nil
	}
	byName := make(map[string]frontend.Expr, len(fields))
	for _, field := range fields {
		byName[field.Name] = field.Value
	}
	for _, field := range info.Fields {
		tail, ok := resourceLeafTail(leaf, field.Name)
		if !ok {
			continue
		}
		value := byName[field.Name]
		if value == nil {
			return resourceSourceResult{}, nil
		}
		return resourceSourceForExprLeaf(value, field.TypeName, tail, funcs, types, module, imports, state)
	}
	return resourceSourceResult{}, nil
}

func resourceSourceForEnumConstructorLeaf(
	call *frontend.CallExpr,
	typeName string,
	leaf string,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (resourceSourceResult, bool, error) {
	info, ok := types[typeName]
	if !ok || info.Kind != TypeEnum {
		return resourceSourceResult{}, false, nil
	}
	caseType, caseInfo, found, err := resolveEnumCaseConstructorCall(call, types, module, imports)
	if err != nil {
		return resourceSourceResult{}, true, err
	}
	if !found || caseType != typeName {
		return resourceSourceResult{}, false, nil
	}
	for i, payloadType := range caseInfo.PayloadTypes {
		if i >= len(call.Args) {
			break
		}
		tail, ok := resourceLeafTail(leaf, resourceEnumPayloadPath("", caseInfo.Ordinal, i))
		if !ok {
			continue
		}
		source, err := resourceSourceForExprLeaf(call.Args[i], payloadType, tail, funcs, types, module, imports, state)
		return source, true, err
	}
	return resourceSourceResult{}, true, nil
}

func resourceLeafTail(leaf string, head string) (string, bool) {
	return semanticsresources.LeafTail(leaf, head)
}
