package semantics

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
	semanticsregions "tetra_language/compiler/internal/semantics/regions"
)

type regionConflict struct {
	leftLabel  string
	leftRegion int

	rightLabel  string
	rightRegion int
}

func (s *regionState) enterIsland(name string) error {
	id, ok := s.islandScopes[name]
	if !ok {
		return fmt.Errorf("unknown island scope '%s'", name)
	}
	s.activateScope(id)
	s.regionVars[name] = id
	return nil
}

func (s *regionState) exitIsland() {
	if len(s.activeScopes) == 0 {
		return
	}
	s.deactivateScope(s.activeScopes[len(s.activeScopes)-1])
}

func (s *regionState) activateScope(id int) {
	if s == nil || id < 0 {
		return
	}
	if _, exists := s.activeIndex[id]; exists {
		return
	}
	s.activeScopes = append(s.activeScopes, id)
	s.activeIndex[id] = len(s.activeScopes) - 1
}

func (s *regionState) deactivateScope(id int) {
	if s == nil || id < 0 {
		return
	}
	idx, ok := s.activeIndex[id]
	if !ok {
		return
	}
	delete(s.activeIndex, id)
	copy(s.activeScopes[idx:], s.activeScopes[idx+1:])
	s.activeScopes = s.activeScopes[:len(s.activeScopes)-1]
	for i := idx; i < len(s.activeScopes); i++ {
		s.activeIndex[s.activeScopes[i]] = i
	}
}

func (s *regionState) isScopeActive(id int) bool {
	if id < 0 {
		return true
	}
	_, ok := s.activeIndex[id]
	return ok
}

func (s *regionState) scopeIndex(id int) (int, bool) {
	idx, ok := s.activeIndex[id]
	return idx, ok
}

func (s *regionState) isScopeWithin(targetID, regionID int) bool {
	if regionID < 0 {
		return true
	}
	if targetID < 0 {
		return false
	}
	regionIdx, ok := s.scopeIndex(regionID)
	if !ok {
		return false
	}
	targetIdx, ok := s.scopeIndex(targetID)
	if !ok {
		return false
	}
	return targetIdx >= regionIdx
}

func copyRegionVars(src map[string]int) map[string]int {
	return semanticsregions.CopyVars(src)
}

func copyRegionTree(src map[string]int) map[string]int {
	return semanticsregions.CopyTree(src)
}

func mergeRegionVars(a, b map[string]int) map[string]int {
	return semanticsregions.MergeVars(a, b)
}

func joinRegion(a, b int) int {
	return semanticsregions.Join(a, b)
}

func commonRegionFromTree(tree map[string]int) int {
	return semanticsregions.CommonFromTree(tree)
}

func constructorRegionFromTree(tree map[string]int) int {
	return semanticsregions.ConstructorFromTree(tree)
}

func regionTreeForExpr(typeName string, expr frontend.Expr, exprRegion int, types map[string]*TypeInfo, state *regionState) map[string]int {
	tree := make(map[string]int)
	appendRegionTree(tree, "", typeName, expr, exprRegion, types, state)
	return tree
}

func appendRegionTree(out map[string]int, prefix string, typeName string, expr frontend.Expr, exprRegion int, types map[string]*TypeInfo, state *regionState) {
	if !typeMayContainRegion(typeName, types) {
		return
	}
	if state != nil {
		if tree, ok := state.exprRegionTree(expr); ok {
			for leaf, regionID := range tree {
				if regionID != regionNone {
					out[joinResourcePath(prefix, leaf)] = regionID
				}
			}
			return
		}
		if sourcePrefix, ok := resourcePathForExpr(expr); ok {
			copied := false
			for _, leaf := range regionLeafPaths(typeName, types, "") {
				sourceLeaf := joinResourcePath(sourcePrefix, leaf)
				if regionID, ok := state.regionVars[sourceLeaf]; ok {
					out[joinResourcePath(prefix, leaf)] = regionID
					copied = true
				}
			}
			if copied {
				return
			}
		}
	}
	if info, ok := types[typeName]; ok && info.Kind == TypeOptional {
		appendRegionTree(out, resourceFieldPath(prefix, "$elem"), info.ElemType, expr, exprRegion, types, state)
		return
	}
	if exprRegion == regionNone {
		return
	}
	for _, leaf := range regionLeafPaths(typeName, types, "") {
		out[joinResourcePath(prefix, leaf)] = exprRegion
	}
}

func bindRegionTreeFromExpr(name string, typeName string, expr frontend.Expr, exprRegion int, types map[string]*TypeInfo, state *regionState) {
	if state == nil || name == "" {
		return
	}
	state.clearRegionTree(name)
	if !typeMayContainRegion(typeName, types) {
		return
	}
	for leaf, regionID := range regionTreeForExpr(typeName, expr, exprRegion, types, state) {
		if regionID != regionNone {
			state.bindRegion(joinResourcePath(name, leaf), regionID)
		}
	}
}

func copyRegionTreeFromPath(dst string, src string, typeName string, types map[string]*TypeInfo, state *regionState) {
	if state == nil || dst == "" || src == "" {
		return
	}
	state.clearRegionTree(dst)
	if !typeMayContainRegion(typeName, types) {
		return
	}
	for _, leaf := range regionLeafPaths(typeName, types, "") {
		srcLeaf := joinResourcePath(src, leaf)
		if regionID, ok := state.regionVars[srcLeaf]; ok {
			state.bindRegion(joinResourcePath(dst, leaf), regionID)
		}
	}
}

func checkRegionTreeWithinScope(tree map[string]int, targetScopeID int, pos frontend.Position, state *regionState) error {
	if state == nil {
		return nil
	}
	for _, regionID := range tree {
		if regionID < 0 {
			continue
		}
		if !state.isScopeWithin(targetScopeID, regionID) {
			return lifetimeDiagnosticf(
				pos,
				"slice from scoped island cannot escape to outer scope (value: %s, target: %s)",
				formatRegionID(state, regionID),
				formatScopeID(state, targetScopeID),
			)
		}
	}
	return nil
}

func checkRegionUsable(regionID int, name string, pos frontend.Position, state *regionState) error {
	if state == nil || regionID == regionNone {
		return nil
	}
	if regionID == regionUnknown {
		return fmt.Errorf("%s: ambiguous region for '%s'", frontend.FormatPos(pos), name)
	}
	if err := state.checkBorrowedRegionAfterAwait(regionID, name, pos); err != nil {
		return err
	}
	if !state.isScopeActive(regionID) {
		return lifetimeDiagnosticf(pos, "slice from scoped island is out of scope")
	}
	return nil
}

func regionLeafPaths(typeName string, types map[string]*TypeInfo, prefix string) []string {
	return regionLeafPathsVisiting(typeName, types, prefix, map[string]bool{})
}

func regionLeafPathsVisiting(typeName string, types map[string]*TypeInfo, prefix string, visiting map[string]bool) []string {
	info, ok := types[typeName]
	if !ok {
		return nil
	}
	if visiting[typeName] {
		return nil
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)
	switch info.Kind {
	case TypeSlice, TypeIsland, TypeStr:
		return []string{prefix}
	case TypeStruct:
		out := []string{}
		for _, field := range info.Fields {
			out = append(out, regionLeafPathsVisiting(field.TypeName, types, resourceFieldPath(prefix, field.Name), visiting)...)
		}
		return out
	case TypeEnum:
		out := []string{}
		for _, c := range info.EnumCases {
			for i, payload := range c.PayloadTypes {
				out = append(out, regionLeafPathsVisiting(payload, types, resourceEnumPayloadPath(prefix, c.Ordinal, i), visiting)...)
			}
		}
		return out
	case TypeArray:
		return []string{prefix}
	case TypeOptional:
		return regionLeafPathsVisiting(info.ElemType, types, resourceFieldPath(prefix, "$elem"), visiting)
	default:
		return nil
	}
}

func markUnknownRegions(state *regionState) {
	if state == nil {
		return
	}
	for name := range state.unknownVars {
		if state.regionVars[name] != regionUnknown {
			delete(state.unknownVars, name)
		}
	}
	for name := range state.unknownConflicts {
		if state.regionVars[name] != regionUnknown {
			delete(state.unknownConflicts, name)
		}
	}
	for name, regionID := range state.regionVars {
		if regionID == regionUnknown {
			state.unknownVars[name] = true
			continue
		}
		delete(state.unknownVars, name)
		delete(state.unknownConflicts, name)
	}
}

func initParamRegions(params []frontend.ParamDecl, state *regionState, types map[string]*TypeInfo) {
	if state != nil && (state.paramNames == nil || len(state.paramNames) != len(params)) {
		state.paramNames = make([]string, len(params))
		for i := range params {
			state.paramNames[i] = params[i].Name
		}
	}
	next := regionParamStart
	for i := range params {
		param := params[i]
		if typeContainsResourceHandle(param.Type.Name, types) {
			for _, leaf := range resourceLeafPaths(param.Type.Name, types, "") {
				name := joinResourcePath(param.Name, leaf)
				state.bindResource(name, "", true)
				if id, ok := state.resourceID(name); ok {
					state.resourceParamIndex[id] = i
					state.resourceParamPath[id] = leaf
				}
			}
		}
		if typeMayContainRegion(param.Type.Name, types) {
			state.regionVars[param.Name] = next
			state.paramRegionIndex[next] = i
			if param.Ownership == "borrow" {
				state.borrowedParamRegion[next] = param.Name
			}
			next--
		}
		if param.Ownership == "borrow" && typeMayContainPtr(param.Type.Name, types) {
			for _, leaf := range ptrLeafPaths(param.Type.Name, types, "") {
				state.borrowedPtrAliases[joinResourcePath(param.Name, leaf)] = param.Name
			}
		}
	}
}

func (s *regionState) resourceParamOwner(name string) (int, string, bool) {
	if s == nil || name == "" {
		return 0, "", false
	}
	id, ok := s.resourceID(name)
	if !ok {
		return 0, "", false
	}
	idx, ok := s.resourceParamIndex[id]
	if !ok {
		return 0, "", false
	}
	return idx, s.resourceParamPath[id], true
}

func (s *regionState) borrowedParamOwner(regionID int) (string, bool) {
	if s == nil || regionID >= regionNone {
		return "", false
	}
	name, ok := s.borrowedParamRegion[regionID]
	return name, ok
}

func (s *regionState) bindExplicitBorrow(owner string) int {
	if s == nil {
		return regionNone
	}
	if owner == "" {
		owner = "<borrow>"
	}
	if s.nextExplicitBorrow >= regionNone {
		s.nextExplicitBorrow = regionExplicitBorrowStart
	}
	id := s.nextExplicitBorrow
	s.nextExplicitBorrow--
	s.borrowedParamRegion[id] = owner
	return id
}

func formatRegionID(state *regionState, regionID int) string {
	switch {
	case regionID == regionNone:
		return "none"
	case regionID == regionUnknown:
		return "unknown"
	case regionID >= 0:
		if state != nil {
			if name, ok := state.islandNameByID[regionID]; ok && name != "" {
				return fmt.Sprintf("isl#%d(%s)", regionID, name)
			}
		}
		return fmt.Sprintf("isl#%d", regionID)
	default:
		if state != nil {
			if idx, ok := state.paramRegionIndex[regionID]; ok {
				if idx >= 0 && idx < len(state.paramNames) {
					return fmt.Sprintf("param#%d(%s)", idx, state.paramNames[idx])
				}
				return fmt.Sprintf("param#%d", idx)
			}
		}
		return fmt.Sprintf("param(%d)", regionID)
	}
}

func formatScopeID(state *regionState, scopeID int) string {
	if scopeID == regionNone {
		return "root"
	}
	if scopeID == regionUnknown {
		return "unknown"
	}
	if state != nil {
		if name, ok := state.islandNameByID[scopeID]; ok && name != "" {
			return fmt.Sprintf("scope#%d(%s)", scopeID, name)
		}
	}
	return fmt.Sprintf("scope#%d", scopeID)
}

func recordMergeConflicts(state *regionState, leftVars, rightVars map[string]int, leftLabel, rightLabel string) {
	if state == nil {
		return
	}
	for name, left := range leftVars {
		right, ok := rightVars[name]
		if !ok {
			right = regionNone
		}
		if left == right {
			continue
		}
		if left == regionNone && right == regionNone {
			continue
		}
		state.unknownConflicts[name] = regionConflict{
			leftLabel:   leftLabel,
			leftRegion:  left,
			rightLabel:  rightLabel,
			rightRegion: right,
		}
	}
	for name, right := range rightVars {
		if _, ok := leftVars[name]; ok {
			continue
		}
		if right == regionNone {
			continue
		}
		state.unknownConflicts[name] = regionConflict{
			leftLabel:   leftLabel,
			leftRegion:  regionNone,
			rightLabel:  rightLabel,
			rightRegion: right,
		}
	}
}

func (s *regionState) enterUnsafe() {
	s.unsafeDepth++
}

func (s *regionState) exitUnsafe() {
	if s.unsafeDepth > 0 {
		s.unsafeDepth--
	}
}

func (s *regionState) inUnsafe() bool {
	return s.unsafeDepth > 0
}

func (s *regionState) recordReturnRegion(regionID int, pos frontend.Position) error {
	if regionID == regionUnknown {
		return fmt.Errorf("%s: ambiguous region for return", frontend.FormatPos(pos))
	}
	if regionID >= 0 {
		return lifetimeDiagnosticf(pos, "return from scoped island is not allowed")
	}
	if !s.returnRegionSet {
		s.returnRegion = regionID
		s.returnRegionSet = true
		return nil
	}
	if s.returnRegion != regionID {
		return fmt.Errorf(
			"%s: return mixes values from different regions (first: %s, now: %s)",
			frontend.FormatPos(pos),
			formatRegionID(s, s.returnRegion),
			formatRegionID(s, regionID),
		)
	}
	return nil
}

func (s *regionState) recordReturnRegionSummary(tree map[string]int, pos frontend.Position) error {
	if s == nil || len(tree) == 0 {
		return nil
	}
	for returnPath, regionID := range tree {
		if regionID == regionUnknown {
			return fmt.Errorf("%s: ambiguous region for return", frontend.FormatPos(pos))
		}
		if regionID >= 0 {
			return lifetimeDiagnosticf(pos, "return from scoped island is not allowed")
		}
		idx, ok := s.paramRegionIndex[regionID]
		if !ok {
			return fmt.Errorf("%s: return region does not match parameter", frontend.FormatPos(pos))
		}
		if s.returnRegionSummary == nil {
			s.returnRegionSummary = ReturnRegionSummary{}
		}
		if existing, exists := s.returnRegionSummary[returnPath]; exists {
			if existing != idx {
				return fmt.Errorf(
					"%s: return mixes region provenance for return%s (first: param#%d, now: param#%d)",
					frontend.FormatPos(pos),
					formatResourceParamPath(returnPath),
					existing,
					idx,
				)
			}
			continue
		}
		s.returnRegionSummary[returnPath] = idx
	}
	return nil
}

func (s *regionState) recordReturnResourceParam(paramIndex int, path string, pos frontend.Position) error {
	if paramIndex < 0 {
		return nil
	}
	if !s.returnResourceSet {
		s.returnResourceParam = paramIndex
		s.returnResourcePath = path
		s.returnResourceSet = true
		return nil
	}
	if s.returnResourceParam != paramIndex || s.returnResourcePath != path {
		return fmt.Errorf(
			"%s: return mixes resource provenance (first: param#%d%s, now: param#%d%s)",
			frontend.FormatPos(pos),
			s.returnResourceParam,
			formatResourceParamPath(s.returnResourcePath),
			paramIndex,
			formatResourceParamPath(path),
		)
	}
	return nil
}

func (s *regionState) recordReturnResourceSummary(summary ReturnResourceSummary, pos frontend.Position) error {
	if s == nil {
		return nil
	}
	for returnPath, provenances := range summary {
		for _, provenance := range provenances {
			if provenance.ParamIndex < 0 {
				continue
			}
			if s.returnResourceSummary == nil {
				s.returnResourceSummary = ReturnResourceSummary{}
			}
			existing := s.returnResourceSummary[returnPath]
			if len(existing) == 0 {
				s.returnResourceSummary[returnPath] = []ResourceProvenance{provenance}
				continue
			}
			if len(existing) == 1 && existing[0] == provenance {
				continue
			}
			first := existing[0]
			return fmt.Errorf(
				"%s: return mixes resource provenance (first: param#%d%s -> return%s, now: param#%d%s -> return%s)",
				frontend.FormatPos(pos),
				first.ParamIndex,
				formatResourceParamPath(first.ParamPath),
				formatResourceParamPath(returnPath),
				provenance.ParamIndex,
				formatResourceParamPath(provenance.ParamPath),
				formatResourceParamPath(returnPath),
			)
		}
	}
	if len(s.returnResourceSummary) > 0 {
		s.returnResourceSet = true
		if provenances := s.returnResourceSummary[""]; len(provenances) == 1 {
			s.returnResourceParam = provenances[0].ParamIndex
			s.returnResourcePath = provenances[0].ParamPath
		}
	}
	return nil
}

func (s *regionState) recordThrowResourceSummary(summary ReturnResourceSummary, pos frontend.Position) error {
	if s == nil {
		return nil
	}
	for throwPath, provenances := range summary {
		for _, provenance := range provenances {
			if provenance.ParamIndex < 0 {
				continue
			}
			if s.throwResourceSummary == nil {
				s.throwResourceSummary = ReturnResourceSummary{}
			}
			existing := s.throwResourceSummary[throwPath]
			if len(existing) == 0 {
				s.throwResourceSummary[throwPath] = []ResourceProvenance{provenance}
				continue
			}
			if len(existing) == 1 && existing[0] == provenance {
				continue
			}
			first := existing[0]
			return fmt.Errorf(
				"%s: throw mixes resource provenance (first: param#%d%s -> throw%s, now: param#%d%s -> throw%s)",
				frontend.FormatPos(pos),
				first.ParamIndex,
				formatResourceParamPath(first.ParamPath),
				formatResourceParamPath(throwPath),
				provenance.ParamIndex,
				formatResourceParamPath(provenance.ParamPath),
				formatResourceParamPath(throwPath),
			)
		}
	}
	return nil
}

func formatResourceParamPath(path string) string {
	if path == "" {
		return ""
	}
	return "." + path
}

func (s *regionState) recordUnknownReturnResource() {
	if s == nil {
		return
	}
	s.returnResourceUnknown = true
}

func typeMayContainRegion(typeName string, types map[string]*TypeInfo) bool {
	return typeMayContainRegionVisiting(typeName, types, map[string]bool{}, map[string]bool{})
}

func typeMayContainPtr(typeName string, types map[string]*TypeInfo) bool {
	return typeMayContainPtrVisiting(typeName, types, map[string]bool{}, map[string]bool{})
}

func typeMayContainPtrVisiting(typeName string, types map[string]*TypeInfo, visiting map[string]bool, memo map[string]bool) bool {
	if resolved, ok := memo[typeName]; ok {
		return resolved
	}
	if typeName == "ptr" {
		memo[typeName] = true
		return true
	}
	if typeName == "fnptr" {
		memo[typeName] = false
		return false
	}
	if visiting[typeName] {
		return false
	}
	info, ok := types[typeName]
	if !ok {
		return false
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)

	result := false
	switch info.Kind {
	case TypeStruct:
		for _, field := range info.Fields {
			if typeMayContainPtrVisiting(field.TypeName, types, visiting, memo) {
				result = true
				break
			}
		}
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if typeMayContainPtrVisiting(payload, types, visiting, memo) {
					result = true
					break
				}
			}
			if result {
				break
			}
		}
	case TypeArray, TypeOptional:
		result = typeMayContainPtrVisiting(info.ElemType, types, visiting, memo)
	default:
		result = false
	}
	memo[typeName] = result
	return result
}

func typeMayContainRegionVisiting(typeName string, types map[string]*TypeInfo, visiting map[string]bool, memo map[string]bool) bool {
	if resolved, ok := memo[typeName]; ok {
		return resolved
	}
	if visiting[typeName] {
		return false
	}
	info, ok := types[typeName]
	if !ok {
		return false
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)

	result := false
	switch info.Kind {
	case TypeSlice:
		result = true
	case TypeIsland:
		result = true
	case TypeStr:
		result = true
	case TypeStruct:
		for _, field := range info.Fields {
			if typeMayContainRegionVisiting(field.TypeName, types, visiting, memo) {
				result = true
				break
			}
		}
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if typeMayContainRegionVisiting(payload, types, visiting, memo) {
					result = true
					break
				}
			}
			if result {
				break
			}
		}
	case TypeArray:
		result = true
	case TypeOptional:
		result = typeMayContainRegionVisiting(info.ElemType, types, visiting, memo)
	default:
		result = false
	}
	memo[typeName] = result
	return result
}

func localScopeID(name string, state *regionState) int {
	if state == nil {
		return regionNone
	}
	if id, ok := state.localScopes[name]; ok {
		return id
	}
	return regionNone
}

func checkLocalScope(name string, state *regionState, pos frontend.Position) error {
	if state != nil {
		if ids := state.localScopeSets[name]; len(ids) > 0 {
			for id := range ids {
				if state.isScopeActive(id) {
					return nil
				}
			}
			return fmt.Errorf("%s: identifier '%s' is out of scope", frontend.FormatPos(pos), name)
		}
	}
	scopeID := localScopeID(name, state)
	if scopeID == regionNone {
		return nil
	}
	if !state.isScopeActive(scopeID) {
		return fmt.Errorf("%s: identifier '%s' is out of scope", frontend.FormatPos(pos), name)
	}
	return nil
}

func withActiveScope(state *regionState, scopeID int, run func() error) error {
	if state == nil || scopeID == regionNone {
		return run()
	}
	state.activateScope(scopeID)
	defer state.deactivateScope(scopeID)
	return run()
}
