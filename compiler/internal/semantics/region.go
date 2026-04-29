package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

const (
	regionNone       = -1
	regionUnknown    = -2
	regionParamStart = -3
)

type branchScopeInfo struct {
	thenID int
	elseID int
}

type resourceFinalization struct {
	state string
	pos   frontend.Position
}

type scopeInfo struct {
	localScopes     map[string]int
	islandScopes    map[string]int
	ifScopes        map[*frontend.IfStmt]branchScopeInfo
	ifLetScopes     map[*frontend.IfLetStmt]branchScopeInfo
	whileScopes     map[*frontend.WhileStmt]int
	forScopes       map[*frontend.ForRangeStmt]int
	matchCaseScopes map[*frontend.MatchStmt][]int
	matchExprScopes map[*frontend.MatchExpr][]int
	catchExprScopes map[*frontend.CatchExpr][]int
	unsafeScopes    map[*frontend.UnsafeStmt]int
	deferScopes     map[*frontend.DeferStmt]int
	scopeStack      []int
	nextScopeID     int
}

func newScopeInfo() *scopeInfo {
	return &scopeInfo{
		localScopes:     make(map[string]int),
		islandScopes:    make(map[string]int),
		ifScopes:        make(map[*frontend.IfStmt]branchScopeInfo),
		ifLetScopes:     make(map[*frontend.IfLetStmt]branchScopeInfo),
		whileScopes:     make(map[*frontend.WhileStmt]int),
		forScopes:       make(map[*frontend.ForRangeStmt]int),
		matchCaseScopes: make(map[*frontend.MatchStmt][]int),
		matchExprScopes: make(map[*frontend.MatchExpr][]int),
		catchExprScopes: make(map[*frontend.CatchExpr][]int),
		unsafeScopes:    make(map[*frontend.UnsafeStmt]int),
		deferScopes:     make(map[*frontend.DeferStmt]int),
	}
}

func (s *scopeInfo) currentScopeID() int {
	if len(s.scopeStack) == 0 {
		return regionNone
	}
	return s.scopeStack[len(s.scopeStack)-1]
}

func (s *scopeInfo) enterScope() int {
	id := s.nextScopeID
	s.nextScopeID++
	s.scopeStack = append(s.scopeStack, id)
	return id
}

func (s *scopeInfo) exitScope() {
	if len(s.scopeStack) == 0 {
		return
	}
	s.scopeStack = s.scopeStack[:len(s.scopeStack)-1]
}

type regionState struct {
	localScopes           map[string]int
	islandScopes          map[string]int
	ifScopes              map[*frontend.IfStmt]branchScopeInfo
	ifLetScopes           map[*frontend.IfLetStmt]branchScopeInfo
	whileScopes           map[*frontend.WhileStmt]int
	forScopes             map[*frontend.ForRangeStmt]int
	matchCaseScopes       map[*frontend.MatchStmt][]int
	matchExprScopes       map[*frontend.MatchExpr][]int
	catchExprScopes       map[*frontend.CatchExpr][]int
	unsafeScopes          map[*frontend.UnsafeStmt]int
	deferScopes           map[*frontend.DeferStmt]int
	islandNameByID        map[int]string
	regionVars            map[string]int
	paramRegionIndex      map[int]int
	resourceParamIndex    map[int]int
	resourceParamPath     map[int]string
	borrowedParamRegion   map[int]string
	paramNames            []string
	unknownVars           map[string]bool
	unknownConflicts      map[string]regionConflict
	consumedVars          map[string]frontend.Position
	consumedResources     map[int]frontend.Position
	resourceVars          map[string]int
	unknownResources      map[int]bool
	finalizedResources    map[int]resourceFinalization
	nextResourceID        int
	deferCaptureFrames    []map[string]frontend.Position
	activeScopes          []int
	activeIndex           map[int]int
	unsafeDepth           int
	loopDepth             int
	throwType             string
	allowThrowDepth       int
	allowThrowCall        *frontend.CallExpr
	allowCatchDepth       int
	allowCatchCall        *frontend.CallExpr
	async                 bool
	allowAwaitDepth       int
	allowAwaitCall        *frontend.CallExpr
	returnRegion          int
	returnRegionSet       bool
	returnResourceParam   int
	returnResourcePath    string
	returnResourceSet     bool
	returnResourceUnknown bool
	actorStateFields      map[string]ActorStateField
}

func newRegionState(scopes *scopeInfo) *regionState {
	localScopes := make(map[string]int)
	islandScopes := make(map[string]int)
	var ifScopes map[*frontend.IfStmt]branchScopeInfo
	var ifLetScopes map[*frontend.IfLetStmt]branchScopeInfo
	var whileScopes map[*frontend.WhileStmt]int
	var forScopes map[*frontend.ForRangeStmt]int
	var matchCaseScopes map[*frontend.MatchStmt][]int
	var matchExprScopes map[*frontend.MatchExpr][]int
	var catchExprScopes map[*frontend.CatchExpr][]int
	var unsafeScopes map[*frontend.UnsafeStmt]int
	var deferScopes map[*frontend.DeferStmt]int
	if scopes != nil {
		localScopes = scopes.localScopes
		islandScopes = scopes.islandScopes
		ifScopes = scopes.ifScopes
		ifLetScopes = scopes.ifLetScopes
		whileScopes = scopes.whileScopes
		forScopes = scopes.forScopes
		matchCaseScopes = scopes.matchCaseScopes
		matchExprScopes = scopes.matchExprScopes
		catchExprScopes = scopes.catchExprScopes
		unsafeScopes = scopes.unsafeScopes
		deferScopes = scopes.deferScopes
	}
	islandNameByID := make(map[int]string, len(islandScopes))
	for name, id := range islandScopes {
		islandNameByID[id] = name
	}
	return &regionState{
		localScopes:         localScopes,
		islandScopes:        islandScopes,
		ifScopes:            ifScopes,
		ifLetScopes:         ifLetScopes,
		whileScopes:         whileScopes,
		forScopes:           forScopes,
		matchCaseScopes:     matchCaseScopes,
		matchExprScopes:     matchExprScopes,
		catchExprScopes:     catchExprScopes,
		unsafeScopes:        unsafeScopes,
		deferScopes:         deferScopes,
		islandNameByID:      islandNameByID,
		regionVars:          make(map[string]int),
		paramRegionIndex:    make(map[int]int),
		resourceParamIndex:  make(map[int]int),
		resourceParamPath:   make(map[int]string),
		borrowedParamRegion: make(map[int]string),
		unknownConflicts:    make(map[string]regionConflict),
		unknownVars:         make(map[string]bool),
		consumedVars:        make(map[string]frontend.Position),
		consumedResources:   make(map[int]frontend.Position),
		resourceVars:        make(map[string]int),
		unknownResources:    make(map[int]bool),
		finalizedResources:  make(map[int]resourceFinalization),
		nextResourceID:      1,
		activeIndex:         make(map[int]int),
	}
}

func (s *regionState) markConsumed(name string, pos frontend.Position) {
	if s == nil || name == "" {
		return
	}
	if id, ok := s.resourceID(name); ok {
		s.consumedResources[id] = pos
		return
	}
	s.consumedVars[name] = pos
}

func (s *regionState) clearConsumed(name string) {
	if s == nil || name == "" {
		return
	}
	delete(s.consumedVars, name)
}

func (s *regionState) checkNotConsumed(name string, pos frontend.Position) error {
	if s == nil || name == "" {
		return nil
	}
	if consumedAt, ok := s.consumedAt(name); ok {
		return fmt.Errorf("%s: cannot use consumed value '%s' (consumed at %s)", frontend.FormatPos(pos), name, frontend.FormatPos(consumedAt))
	}
	return nil
}

func (s *regionState) markResourceFinalized(name string, state string, pos frontend.Position) {
	if s == nil || name == "" || state == "" {
		return
	}
	id := s.ensureResource(name)
	s.finalizedResources[id] = resourceFinalization{state: state, pos: pos}
}

func (s *regionState) clearResourceFinalized(name string) {
	if s == nil || name == "" {
		return
	}
	if id, ok := s.resourceID(name); ok {
		delete(s.finalizedResources, id)
	}
}

func (s *regionState) bindResource(name string, source string, isResource bool) {
	if s == nil || name == "" {
		return
	}
	if !isResource {
		delete(s.resourceVars, name)
		return
	}
	if source != "" {
		if id, ok := s.resourceID(source); ok {
			s.resourceVars[name] = id
			return
		}
	}
	s.resourceVars[name] = s.allocateResourceID()
}

func (s *regionState) bindTransferredResource(name string, source string) {
	if s == nil || name == "" {
		return
	}
	id := s.allocateResourceID()
	s.resourceVars[name] = id
	if sourceID, ok := s.resourceID(source); ok {
		if idx, idxOK := s.resourceParamIndex[sourceID]; idxOK {
			s.resourceParamIndex[id] = idx
		}
		if path, pathOK := s.resourceParamPath[sourceID]; pathOK {
			s.resourceParamPath[id] = path
		}
	}
}

func (s *regionState) bindUnknownResource(name string) {
	if s == nil || name == "" {
		return
	}
	id := s.allocateResourceID()
	s.resourceVars[name] = id
	s.unknownResources[id] = true
}

func (s *regionState) resourceUnknown(name string) bool {
	if s == nil || name == "" {
		return false
	}
	id, ok := s.resourceID(name)
	if !ok {
		return false
	}
	return s.unknownResources[id]
}

func (s *regionState) resourceFinalization(name string) (resourceFinalization, bool) {
	if s == nil || name == "" {
		return resourceFinalization{}, false
	}
	id, ok := s.resourceID(name)
	if !ok {
		return resourceFinalization{}, false
	}
	final, ok := s.finalizedResources[id]
	return final, ok
}

func (s *regionState) checkResourceNotFinalized(name string, pos frontend.Position) error {
	if s == nil || name == "" {
		return nil
	}
	final, ok := s.resourceFinalization(name)
	if !ok || final.state == "closed" {
		return nil
	}
	return s.resourceFinalizationError(name, final, pos)
}

func (s *regionState) checkResourceFinalizationAllowed(name string, pos frontend.Position, allowed ...string) error {
	if s == nil || name == "" {
		return nil
	}
	final, ok := s.resourceFinalization(name)
	if !ok {
		return nil
	}
	for _, state := range allowed {
		if final.state == state {
			return nil
		}
	}
	return s.resourceFinalizationError(name, final, pos)
}

func (s *regionState) resourceFinalizationError(name string, final resourceFinalization, pos frontend.Position) error {
	return fmt.Errorf(
		"%s: cannot use %s resource '%s' (%s at %s)",
		frontend.FormatPos(pos),
		final.state,
		name,
		final.state,
		frontend.FormatPos(final.pos),
	)
}

func (s *regionState) resourceID(name string) (int, bool) {
	if s == nil || name == "" {
		return 0, false
	}
	id, ok := s.resourceVars[name]
	return id, ok
}

func (s *regionState) ensureResource(name string) int {
	if id, ok := s.resourceID(name); ok {
		return id
	}
	id := s.allocateResourceID()
	s.resourceVars[name] = id
	return id
}

func (s *regionState) allocateResourceID() int {
	if s.nextResourceID <= 0 {
		s.nextResourceID = 1
	}
	id := s.nextResourceID
	s.nextResourceID++
	return id
}

func (s *regionState) consumedAt(name string) (frontend.Position, bool) {
	if s == nil || name == "" {
		return frontend.Position{}, false
	}
	if consumedAt, ok := s.consumedVars[name]; ok {
		return consumedAt, true
	}
	if id, ok := s.resourceID(name); ok {
		consumedAt, consumed := s.consumedResources[id]
		return consumedAt, consumed
	}
	return frontend.Position{}, false
}

func isResourceHandleType(typeName string) bool {
	switch typeName {
	case "actor", "island", "task.group", "task.i32":
		return true
	default:
		return strings.HasPrefix(typeName, "task.i32.throws.")
	}
}

func typeContainsResourceHandle(typeName string, types map[string]*TypeInfo) bool {
	return typeContainsResourceHandleVisiting(typeName, types, map[string]bool{})
}

func typeContainsResourceHandleVisiting(typeName string, types map[string]*TypeInfo, visiting map[string]bool) bool {
	if isResourceHandleType(typeName) {
		return true
	}
	info, ok := types[typeName]
	if !ok {
		return false
	}
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if typeContainsResourceHandleVisiting(field.TypeName, types, visiting) {
				return true
			}
		}
	case TypeEnum:
		if visiting[typeName] {
			return false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if typeContainsResourceHandleVisiting(payload, types, visiting) {
					return true
				}
			}
		}
	case TypeArray, TypeOptional:
		return typeContainsResourceHandleVisiting(info.ElemType, types, visiting)
	}
	return false
}

func (s *regionState) clearResourceTree(prefix string) {
	if s == nil || prefix == "" {
		return
	}
	delete(s.resourceVars, prefix)
	prefixDot := prefix + "."
	for name := range s.resourceVars {
		if strings.HasPrefix(name, prefixDot) {
			delete(s.resourceVars, name)
		}
	}
}

func (s *regionState) pushDeferCaptureFrame() {
	if s == nil {
		return
	}
	s.deferCaptureFrames = append(s.deferCaptureFrames, make(map[string]frontend.Position))
}

func (s *regionState) popDeferCaptureFrame() {
	if s == nil || len(s.deferCaptureFrames) == 0 {
		return
	}
	s.deferCaptureFrames = s.deferCaptureFrames[:len(s.deferCaptureFrames)-1]
}

func (s *regionState) registerDeferCaptures(captures map[string]frontend.Position) {
	if s == nil || len(captures) == 0 || len(s.deferCaptureFrames) == 0 {
		return
	}
	frame := s.deferCaptureFrames[len(s.deferCaptureFrames)-1]
	for name, pos := range captures {
		if _, exists := frame[name]; !exists {
			frame[name] = pos
		}
	}
}

func (s *regionState) checkPendingDeferCaptures(pos frontend.Position) error {
	if s == nil || (len(s.consumedVars) == 0 && len(s.consumedResources) == 0) || len(s.deferCaptureFrames) == 0 {
		return nil
	}
	for i := len(s.deferCaptureFrames) - 1; i >= 0; i-- {
		for name, capturedAt := range s.deferCaptureFrames[i] {
			consumedAt, consumed := s.consumedAt(name)
			if !consumed {
				continue
			}
			if pos.Line == 0 {
				pos = consumedAt
			}
			return fmt.Errorf(
				"%s: defer cleanup captures value '%s' at %s, but it was consumed at %s before cleanup ran",
				frontend.FormatPos(pos),
				name,
				frontend.FormatPos(capturedAt),
				frontend.FormatPos(consumedAt),
			)
		}
	}
	return nil
}

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
	if len(src) == 0 {
		return make(map[string]int)
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeRegionVars(a, b map[string]int) map[string]int {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]int)
	}
	merged := make(map[string]int)
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			vb = regionNone
		}
		if va == vb {
			if va != regionNone {
				merged[k] = va
			}
			continue
		}
		merged[k] = regionUnknown
	}
	for k, vb := range b {
		if _, ok := a[k]; ok {
			continue
		}
		if vb != regionNone {
			merged[k] = regionUnknown
		}
	}
	return merged
}

func joinRegion(a, b int) int {
	if a == regionNone {
		return b
	}
	if b == regionNone {
		return a
	}
	if a == b {
		return a
	}
	return regionUnknown
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
		return fmt.Errorf("%s: return from scoped island is not allowed", frontend.FormatPos(pos))
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
	info, ok := types[typeName]
	if !ok {
		return false
	}
	switch info.Kind {
	case TypeSlice:
		return true
	case TypeIsland:
		return true
	case TypeStruct:
		for _, field := range info.Fields {
			if typeMayContainRegion(field.TypeName, types) {
				return true
			}
		}
		return false
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if typeMayContainRegion(payload, types) {
					return true
				}
			}
		}
		return false
	case TypeArray:
		return typeMayContainRegion(info.ElemType, types)
	default:
		return false
	}
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
