package semantics

import (
	"fmt"

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

type scopeInfo struct {
	localScopes     map[string]int
	islandScopes    map[string]int
	ifScopes        map[*frontend.IfStmt]branchScopeInfo
	ifLetScopes     map[*frontend.IfLetStmt]branchScopeInfo
	whileScopes     map[*frontend.WhileStmt]int
	forScopes       map[*frontend.ForRangeStmt]int
	matchCaseScopes map[*frontend.MatchStmt][]int
	unsafeScopes    map[*frontend.UnsafeStmt]int
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
		unsafeScopes:    make(map[*frontend.UnsafeStmt]int),
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
	localScopes         map[string]int
	islandScopes        map[string]int
	ifScopes            map[*frontend.IfStmt]branchScopeInfo
	ifLetScopes         map[*frontend.IfLetStmt]branchScopeInfo
	whileScopes         map[*frontend.WhileStmt]int
	forScopes           map[*frontend.ForRangeStmt]int
	matchCaseScopes     map[*frontend.MatchStmt][]int
	unsafeScopes        map[*frontend.UnsafeStmt]int
	islandNameByID      map[int]string
	regionVars          map[string]int
	paramRegionIndex    map[int]int
	borrowedParamRegion map[int]string
	paramNames          []string
	unknownVars         map[string]bool
	unknownConflicts    map[string]regionConflict
	consumedVars        map[string]frontend.Position
	activeScopes        []int
	activeIndex         map[int]int
	unsafeDepth         int
	loopDepth           int
	throwType           string
	allowThrowDepth     int
	allowThrowCall      *frontend.CallExpr
	async               bool
	allowAwaitDepth     int
	allowAwaitCall      *frontend.CallExpr
	returnRegion        int
	returnRegionSet     bool
}

func newRegionState(scopes *scopeInfo) *regionState {
	localScopes := make(map[string]int)
	islandScopes := make(map[string]int)
	var ifScopes map[*frontend.IfStmt]branchScopeInfo
	var ifLetScopes map[*frontend.IfLetStmt]branchScopeInfo
	var whileScopes map[*frontend.WhileStmt]int
	var forScopes map[*frontend.ForRangeStmt]int
	var matchCaseScopes map[*frontend.MatchStmt][]int
	var unsafeScopes map[*frontend.UnsafeStmt]int
	if scopes != nil {
		localScopes = scopes.localScopes
		islandScopes = scopes.islandScopes
		ifScopes = scopes.ifScopes
		ifLetScopes = scopes.ifLetScopes
		whileScopes = scopes.whileScopes
		forScopes = scopes.forScopes
		matchCaseScopes = scopes.matchCaseScopes
		unsafeScopes = scopes.unsafeScopes
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
		unsafeScopes:        unsafeScopes,
		islandNameByID:      islandNameByID,
		regionVars:          make(map[string]int),
		paramRegionIndex:    make(map[int]int),
		borrowedParamRegion: make(map[int]string),
		unknownConflicts:    make(map[string]regionConflict),
		unknownVars:         make(map[string]bool),
		consumedVars:        make(map[string]frontend.Position),
		activeIndex:         make(map[int]int),
	}
}

func (s *regionState) markConsumed(name string, pos frontend.Position) {
	if s == nil || name == "" {
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
	if consumedAt, ok := s.consumedVars[name]; ok {
		return fmt.Errorf("%s: cannot use consumed value '%s' (consumed at %s)", frontend.FormatPos(pos), name, frontend.FormatPos(consumedAt))
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
