package semantics

import (
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

const (
	regionNone                = -1
	regionUnknown             = -2
	regionParamStart          = -3
	regionExplicitBorrowStart = -1000000
)

type branchScopeInfo struct {
	thenID int
	elseID int
}

type resourceFinalization struct {
	state          string
	pos            frontend.Position
	maybe          bool
	mayBeAvailable bool
	states         map[string]frontend.Position
}

type ownershipJoinConflict struct {
	leftLabel     string
	leftConsumed  bool
	leftPos       frontend.Position
	rightLabel    string
	rightConsumed bool
	rightPos      frontend.Position
}

type scopeInfo struct {
	localScopes     map[string]int
	localScopeSets  map[string]map[int]struct{}
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
		localScopeSets:  make(map[string]map[int]struct{}),
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
	localScopes            map[string]int
	localScopeSets         map[string]map[int]struct{}
	islandScopes           map[string]int
	ifScopes               map[*frontend.IfStmt]branchScopeInfo
	ifLetScopes            map[*frontend.IfLetStmt]branchScopeInfo
	whileScopes            map[*frontend.WhileStmt]int
	forScopes              map[*frontend.ForRangeStmt]int
	matchCaseScopes        map[*frontend.MatchStmt][]int
	matchExprScopes        map[*frontend.MatchExpr][]int
	catchExprScopes        map[*frontend.CatchExpr][]int
	unsafeScopes           map[*frontend.UnsafeStmt]int
	deferScopes            map[*frontend.DeferStmt]int
	islandNameByID         map[int]string
	regionVars             map[string]int
	exprRegionTrees        map[frontend.Expr]map[string]int
	paramRegionIndex       map[int]int
	resourceParamIndex     map[int]int
	resourceParamPath      map[int]string
	borrowedParamRegion    map[int]string
	nextExplicitBorrow     int
	paramNames             []string
	unknownVars            map[string]bool
	unknownConflicts       map[string]regionConflict
	reachable              bool
	consumedVars           map[string]frontend.Position
	maybeConsumedVars      map[string]ownershipJoinConflict
	ownershipAliases       map[string]string
	borrowedPtrAliases     map[string]string
	ownedRegionSliceOwners map[string]string
	consumedResources      map[int]frontend.Position
	resourceVars           map[string]int
	unknownResources       map[int]bool
	finalizedResources     map[int]resourceFinalization
	nextResourceID         int
	deferCaptureFrames     []map[string]frontend.Position
	activeScopes           []int
	activeIndex            map[int]int
	unsafeDepth            int
	loopDepth              int
	loopFlowFrames         []loopFlowFrame
	throwType              string
	allowThrowDepth        int
	allowThrowCall         *frontend.CallExpr
	allowCatchDepth        int
	allowCatchCall         *frontend.CallExpr
	async                  bool
	allowAwaitDepth        int
	allowAwaitCall         *frontend.CallExpr
	returnRegion           int
	returnRegionSet        bool
	returnRegionSummary    ReturnRegionSummary
	returnResourceParam    int
	returnResourcePath     string
	returnResourceSummary  ReturnResourceSummary
	returnResourceSet      bool
	returnResourceUnknown  bool
	throwResourceSummary   ReturnResourceSummary
	actorStateFields       map[string]ActorStateField
}

func newRegionState(scopes *scopeInfo) *regionState {
	localScopes := make(map[string]int)
	localScopeSets := make(map[string]map[int]struct{})
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
		localScopeSets = scopes.localScopeSets
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
		localScopes:            localScopes,
		localScopeSets:         localScopeSets,
		islandScopes:           islandScopes,
		ifScopes:               ifScopes,
		ifLetScopes:            ifLetScopes,
		whileScopes:            whileScopes,
		forScopes:              forScopes,
		matchCaseScopes:        matchCaseScopes,
		matchExprScopes:        matchExprScopes,
		catchExprScopes:        catchExprScopes,
		unsafeScopes:           unsafeScopes,
		deferScopes:            deferScopes,
		islandNameByID:         islandNameByID,
		regionVars:             make(map[string]int),
		exprRegionTrees:        make(map[frontend.Expr]map[string]int),
		paramRegionIndex:       make(map[int]int),
		resourceParamIndex:     make(map[int]int),
		resourceParamPath:      make(map[int]string),
		borrowedParamRegion:    make(map[int]string),
		nextExplicitBorrow:     regionExplicitBorrowStart,
		unknownConflicts:       make(map[string]regionConflict),
		unknownVars:            make(map[string]bool),
		reachable:              true,
		consumedVars:           make(map[string]frontend.Position),
		maybeConsumedVars:      make(map[string]ownershipJoinConflict),
		ownershipAliases:       make(map[string]string),
		borrowedPtrAliases:     make(map[string]string),
		ownedRegionSliceOwners: make(map[string]string),
		consumedResources:      make(map[int]frontend.Position),
		resourceVars:           make(map[string]int),
		unknownResources:       make(map[int]bool),
		finalizedResources:     make(map[int]resourceFinalization),
		nextResourceID:         1,
		activeIndex:            make(map[int]int),
	}
}

func (s *regionState) markConsumed(name string, pos frontend.Position) {
	if s == nil || name == "" {
		return
	}
	s.markConsumedDirect(name, pos)
	if source, ok := s.ownershipAliasSource(name); ok {
		s.markConsumedDirect(source, pos)
	}
}

func (s *regionState) markConsumedDirect(name string, pos frontend.Position) {
	if s == nil || name == "" {
		return
	}
	if id, ok := s.resourceID(name); ok {
		s.consumedResources[id] = pos
		return
	}
	delete(s.maybeConsumedVars, name)
	s.consumedVars[name] = pos
}

func (s *regionState) clearConsumed(name string) {
	if s == nil || name == "" {
		return
	}
	delete(s.consumedVars, name)
	delete(s.maybeConsumedVars, name)
	if source, ok := s.ownershipAliasSource(name); ok {
		delete(s.consumedVars, source)
		delete(s.maybeConsumedVars, source)
	}
}

func (s *regionState) clearConsumedTree(name string) {
	if s == nil || name == "" {
		return
	}
	s.clearConsumedTreeDirect(name)
}

func (s *regionState) clearConsumedTreeDirect(name string) {
	if s == nil || name == "" {
		return
	}
	queryName := name
	if source, ok := s.ownershipAliasSource(name); ok {
		queryName = source
	}
	for path := range s.consumedVars {
		target := path
		if source, ok := s.ownershipAliasSource(path); ok {
			target = source
		}
		if target == queryName || ownershipPathPrefix(queryName, target) {
			delete(s.consumedVars, path)
		}
	}
	for path := range s.maybeConsumedVars {
		target := path
		if source, ok := s.ownershipAliasSource(path); ok {
			target = source
		}
		if target == queryName || ownershipPathPrefix(queryName, target) {
			delete(s.maybeConsumedVars, path)
		}
	}
}

func (s *regionState) checkAssignableOwnershipPath(path string, pos frontend.Position) error {
	if s == nil || path == "" {
		return nil
	}
	parent := parentOwnershipPath(path)
	if parent == "" {
		return nil
	}
	return s.checkNotConsumed(parent, pos)
}

func (s *regionState) bindOwnershipAlias(name string, source string) {
	if s == nil || name == "" {
		return
	}
	if source == "" || source == name {
		delete(s.ownershipAliases, name)
		return
	}
	s.ownershipAliases[name] = source
}

func (s *regionState) bindBorrowedPtrAlias(name string, owner string) {
	if s == nil || name == "" {
		return
	}
	if owner == "" || owner == name {
		s.clearBorrowedPtrAliasTree(name)
		return
	}
	s.borrowedPtrAliases[name] = owner
}

func (s *regionState) clearBorrowedPtrAliasTree(name string) {
	if s == nil || name == "" {
		return
	}
	for path := range s.borrowedPtrAliases {
		if path == name || ownershipPathPrefix(name, path) {
			delete(s.borrowedPtrAliases, path)
		}
	}
}

func (s *regionState) bindOwnedRegionSliceOwner(name string, owner string) {
	if s == nil || name == "" {
		return
	}
	if owner == "" || owner == name {
		s.clearOwnedRegionSliceOwnerTree(name)
		return
	}
	s.ownedRegionSliceOwners[name] = owner
}

func (s *regionState) clearOwnedRegionSliceOwnerTree(name string) {
	if s == nil || name == "" {
		return
	}
	for path := range s.ownedRegionSliceOwners {
		if path == name || ownershipPathPrefix(name, path) {
			delete(s.ownedRegionSliceOwners, path)
		}
	}
}

func (s *regionState) ownedRegionSliceOwner(path string) (string, bool) {
	if s == nil || path == "" {
		return "", false
	}
	for probe := path; probe != ""; probe = ownershipPathParent(probe) {
		owner, ok := s.ownedRegionSliceOwners[probe]
		if !ok || owner == "" {
			continue
		}
		if probe == path {
			return owner, true
		}
		return owner + path[len(probe):], true
	}
	return "", false
}

func (s *regionState) borrowedPtrAliasOwner(name string) (string, bool) {
	if s == nil || name == "" {
		return "", false
	}
	owner, ok := s.borrowedPtrAliases[name]
	return owner, ok && owner != ""
}

func (s *regionState) borrowedPtrAliasOwnerInTree(name string) (string, bool) {
	if s == nil || name == "" {
		return "", false
	}
	if owner, ok := s.borrowedPtrAliasOwner(name); ok {
		return owner, true
	}
	paths := make([]string, 0, len(s.borrowedPtrAliases))
	for path := range s.borrowedPtrAliases {
		if ownershipPathPrefix(name, path) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		if owner := s.borrowedPtrAliases[path]; owner != "" {
			return owner, true
		}
	}
	return "", false
}

func (s *regionState) checkNotConsumed(name string, pos frontend.Position) error {
	if s == nil || name == "" {
		return nil
	}
	if consumedName, consumedAt, conflict, maybe, ok := s.consumedPath(name); ok {
		reportName := ownershipDiagnosticPath(name, consumedName)
		if maybe {
			return ownershipDiagnosticf(pos, "cannot use consumed value '%s': value '%s' may have been consumed after ownership join (%s: %s, %s: %s)", reportName, reportName, conflict.leftLabel, formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos), conflict.rightLabel, formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos))
		}
		return ownershipDiagnosticf(pos, "cannot use consumed value '%s' (consumed at %s)", reportName, frontend.FormatPos(consumedAt))
	}
	if source, ok := s.ownershipAliasSource(name); ok {
		if consumedName, consumedAt, conflict, maybe, ok := s.consumedPath(source); ok {
			reportName := ownershipDiagnosticPath(name, consumedName)
			if maybe {
				return ownershipDiagnosticf(pos, "cannot use consumed value '%s': value '%s' may have been consumed after ownership join (%s: %s, %s: %s)", reportName, reportName, conflict.leftLabel, formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos), conflict.rightLabel, formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos))
			}
			return ownershipDiagnosticf(pos, "cannot use consumed value '%s' (consumed at %s)", reportName, frontend.FormatPos(consumedAt))
		}
	}
	return nil
}

func (s *regionState) checkNoConsumedDescendants(name string, pos frontend.Position) error {
	if s == nil || name == "" {
		return nil
	}
	queryName := name
	if source, ok := s.ownershipAliasSource(name); ok {
		queryName = source
	}
	for consumedName, consumedAt := range s.consumedVars {
		reportName := consumedName
		if source, ok := s.ownershipAliasSource(consumedName); ok {
			reportName = source
		}
		if reportName != queryName && !ownershipPathPrefix(queryName, reportName) {
			continue
		}
		if conflict, maybe := s.maybeConsumedVars[consumedName]; maybe {
			return ownershipDiagnosticf(pos, "cannot use consumed value '%s': value '%s' may have been consumed after ownership join (%s: %s, %s: %s)", reportName, reportName, conflict.leftLabel, formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos), conflict.rightLabel, formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos))
		}
		if conflict, maybe := s.maybeConsumedVars[reportName]; maybe {
			return ownershipDiagnosticf(pos, "cannot use consumed value '%s': value '%s' may have been consumed after ownership join (%s: %s, %s: %s)", reportName, reportName, conflict.leftLabel, formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos), conflict.rightLabel, formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos))
		}
		return ownershipDiagnosticf(pos, "cannot use consumed value '%s' (consumed at %s)", reportName, frontend.FormatPos(consumedAt))
	}
	return nil
}

func (s *regionState) checkNoConsumedProperDescendants(name string, pos frontend.Position) error {
	if s == nil || name == "" {
		return nil
	}
	queryName := name
	if source, ok := s.ownershipAliasSource(name); ok {
		queryName = source
	}
	for consumedName, consumedAt := range s.consumedVars {
		reportName := consumedName
		if source, ok := s.ownershipAliasSource(consumedName); ok {
			reportName = source
		}
		if reportName == queryName || !ownershipPathPrefix(queryName, reportName) {
			continue
		}
		if conflict, maybe := s.maybeConsumedVars[consumedName]; maybe {
			return ownershipDiagnosticf(pos, "cannot use consumed value '%s': value '%s' may have been consumed after ownership join (%s: %s, %s: %s)", reportName, reportName, conflict.leftLabel, formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos), conflict.rightLabel, formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos))
		}
		if conflict, maybe := s.maybeConsumedVars[reportName]; maybe {
			return ownershipDiagnosticf(pos, "cannot use consumed value '%s': value '%s' may have been consumed after ownership join (%s: %s, %s: %s)", reportName, reportName, conflict.leftLabel, formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos), conflict.rightLabel, formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos))
		}
		return ownershipDiagnosticf(pos, "cannot use consumed value '%s' (consumed at %s)", reportName, frontend.FormatPos(consumedAt))
	}
	return nil
}

func (s *regionState) consumedPath(name string) (string, frontend.Position, ownershipJoinConflict, bool, bool) {
	for path := name; path != ""; path = ownershipPathParent(path) {
		if consumedAt, ok := s.consumedAt(path); ok {
			consumedName := path
			if source, alias := s.ownershipAliasSource(path); alias {
				consumedName = source
			}
			conflict, maybe := s.maybeConsumedVars[consumedName]
			if !maybe && consumedName != path {
				conflict, maybe = s.maybeConsumedVars[path]
			}
			return consumedName, consumedAt, conflict, maybe, true
		}
		if probePath, alias := s.ownershipAliasSource(path); alias {
			if consumedAt, ok := s.consumedAt(probePath); ok {
				conflict, maybe := s.maybeConsumedVars[probePath]
				return probePath, consumedAt, conflict, maybe, true
			}
		}
	}
	return "", frontend.Position{}, ownershipJoinConflict{}, false, false
}

func ownershipDiagnosticPath(queryPath string, consumedPath string) string {
	if queryPath != "" && consumedPath != "" && containsSyntheticOwnershipSegment(consumedPath) && !containsSyntheticOwnershipSegment(queryPath) {
		return queryPath
	}
	return consumedPath
}

func containsSyntheticOwnershipSegment(path string) bool {
	for _, segment := range strings.Split(path, ".") {
		if strings.HasPrefix(segment, "$") {
			return true
		}
	}
	return false
}

func (s *regionState) ownershipAliasSource(path string) (string, bool) {
	if s == nil || path == "" {
		return "", false
	}
	for probe := path; probe != ""; probe = ownershipPathParent(probe) {
		source, ok := s.ownershipAliases[probe]
		if !ok || source == "" {
			continue
		}
		if probe == path {
			return source, true
		}
		return source + path[len(probe):], true
	}
	return "", false
}

func parentOwnershipPath(path string) string {
	return ownershipPathParent(path)
}

func formatOwnershipJoinState(consumed bool, pos frontend.Position) string {
	if !consumed {
		return "available"
	}
	return fmt.Sprintf("consumed at %s", frontend.FormatPos(pos))
}

func (s *regionState) markResourceFinalized(name string, state string, pos frontend.Position) {
	if s == nil || name == "" || state == "" {
		return
	}
	id := s.ensureResource(name)
	s.finalizedResources[id] = resourceFinalization{state: state, pos: pos}
}

func (s *regionState) markResourceFinalizedAliases(name string, state string, pos frontend.Position) {
	if s == nil || name == "" || state == "" {
		return
	}
	id, ok := s.resourceID(name)
	if !ok {
		id = s.ensureResource(name)
	}
	for _, aliasID := range s.resourceVars {
		if aliasID == id {
			s.finalizedResources[aliasID] = resourceFinalization{state: state, pos: pos}
		}
	}
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
	if !ok || resourceFinalizationAllows(final, "closed") {
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
	if resourceFinalizationAllows(final, allowed...) {
		return nil
	}
	return s.resourceFinalizationError(name, final, pos)
}

func (s *regionState) resourceFinalizationError(name string, final resourceFinalization, pos frontend.Position) error {
	if final.maybe {
		states := resourceFinalizationStates(final)
		if len(states) == 1 {
			state := states[0]
			return ownershipDiagnosticf(
				pos,
				"cannot use %s resource '%s': resource may have been %s after control-flow merge (%s)",
				state,
				name,
				state,
				formatResourceFinalizationPossibilities(final),
			)
		}
		return ownershipDiagnosticf(
			pos,
			"cannot use finalized resource '%s': ambiguous finalization state after control-flow merge (%s)",
			name,
			formatResourceFinalizationPossibilities(final),
		)
	}
	return ownershipDiagnosticf(
		pos,
		"cannot use %s resource '%s' (%s at %s)",
		final.state,
		name,
		final.state,
		frontend.FormatPos(final.pos),
	)
}

func resourceFinalizationAllows(final resourceFinalization, allowed ...string) bool {
	allowedStates := make(map[string]bool, len(allowed))
	for _, state := range allowed {
		allowedStates[state] = true
	}
	for state := range resourceFinalizationStatePositions(final) {
		if !allowedStates[state] {
			return false
		}
	}
	return true
}

func resourceFinalizationStates(final resourceFinalization) []string {
	statePositions := resourceFinalizationStatePositions(final)
	states := make([]string, 0, len(statePositions))
	for state := range statePositions {
		states = append(states, state)
	}
	sort.Strings(states)
	return states
}

func resourceFinalizationStatePositions(final resourceFinalization) map[string]frontend.Position {
	states := make(map[string]frontend.Position)
	if final.state != "" {
		states[final.state] = final.pos
	}
	for state, pos := range final.states {
		if existing, ok := states[state]; ok {
			states[state] = earliestPosition(existing, pos)
			continue
		}
		states[state] = pos
	}
	return states
}

func formatResourceFinalizationPossibilities(final resourceFinalization) string {
	parts := []string{}
	if final.mayBeAvailable {
		parts = append(parts, "available")
	}
	for _, state := range resourceFinalizationStates(final) {
		pos := resourceFinalizationStatePositions(final)[state]
		parts = append(parts, fmt.Sprintf("%s at %s", state, frontend.FormatPos(pos)))
	}
	return strings.Join(parts, ", ")
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
	case "actor", "island", "task.group", "task.i32", surfaceSurfaceTypeName:
		return true
	default:
		return strings.HasPrefix(typeName, "task.i32.throws.")
	}
}

func typeContainsResourceHandle(typeName string, types map[string]*TypeInfo) bool {
	return typeContainsResourceHandleVisiting(typeName, types, map[string]bool{})
}

func typeContainsResourceHandleVisiting(typeName string, types map[string]*TypeInfo, visiting map[string]bool) bool {
	if typeName == surfaceFrameTypeName {
		return false
	}
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

func (s *regionState) clearRegionTree(prefix string) {
	if s == nil || prefix == "" {
		return
	}
	delete(s.regionVars, prefix)
	delete(s.unknownVars, prefix)
	delete(s.unknownConflicts, prefix)
	s.clearOwnedRegionSliceOwnerTree(prefix)
	prefixDot := prefix + "."
	for name := range s.regionVars {
		if strings.HasPrefix(name, prefixDot) {
			delete(s.regionVars, name)
			delete(s.unknownVars, name)
			delete(s.unknownConflicts, name)
		}
	}
}

func (s *regionState) bindRegion(name string, regionID int) {
	if s == nil || name == "" {
		return
	}
	if regionID == regionNone {
		delete(s.regionVars, name)
		delete(s.unknownVars, name)
		delete(s.unknownConflicts, name)
		return
	}
	s.regionVars[name] = regionID
	delete(s.unknownVars, name)
	delete(s.unknownConflicts, name)
}

func (s *regionState) setExprRegionTree(expr frontend.Expr, tree map[string]int) {
	if s == nil || expr == nil {
		return
	}
	if len(tree) == 0 {
		delete(s.exprRegionTrees, expr)
		return
	}
	s.exprRegionTrees[expr] = copyRegionTree(tree)
}

func (s *regionState) exprRegionTree(expr frontend.Expr) (map[string]int, bool) {
	if s == nil || expr == nil {
		return nil, false
	}
	tree, ok := s.exprRegionTrees[expr]
	if !ok {
		return nil, false
	}
	return copyRegionTree(tree), true
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
			consumedAt, consumed := s.deferredCaptureConsumedAt(name)
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

func (s *regionState) deferredCaptureConsumedAt(name string) (frontend.Position, bool) {
	if s == nil || name == "" {
		return frontend.Position{}, false
	}
	queryName := name
	if source, ok := s.ownershipAliasSource(name); ok {
		queryName = source
	}
	if consumedAt, consumed := s.consumedAt(queryName); consumed {
		return consumedAt, true
	}
	for consumedName, consumedAt := range s.consumedVars {
		reportName := consumedName
		if source, ok := s.ownershipAliasSource(consumedName); ok {
			reportName = source
		}
		if reportName == queryName || ownershipPathPrefix(queryName, reportName) {
			return consumedAt, true
		}
	}
	for resourceName, resourceID := range s.resourceVars {
		consumedAt, consumed := s.consumedResources[resourceID]
		if !consumed {
			continue
		}
		reportName := resourceName
		if source, ok := s.ownershipAliasSource(resourceName); ok {
			reportName = source
		}
		if reportName == queryName || ownershipPathPrefix(queryName, reportName) {
			return consumedAt, true
		}
	}
	return frontend.Position{}, false
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

func copyRegionTree(src map[string]int) map[string]int {
	return copyRegionVars(src)
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

func commonRegionFromTree(tree map[string]int) int {
	regionID := regionNone
	for _, leafRegion := range tree {
		regionID = joinRegion(regionID, leafRegion)
	}
	return regionID
}

func constructorRegionFromTree(tree map[string]int) int {
	regionID := commonRegionFromTree(tree)
	if regionID == regionUnknown {
		return regionNone
	}
	return regionID
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
