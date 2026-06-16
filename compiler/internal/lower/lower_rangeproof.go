package lower

import (
	"tetra_language/compiler/internal/frontend"
	lowerrangeproof "tetra_language/compiler/internal/lower/rangeproof"
	corerangeproof "tetra_language/compiler/internal/rangeproof"
	"tetra_language/compiler/internal/semantics"
)

func (l *lowerer) whileRangeProof(stmt *frontend.WhileStmt) (whileRangeProof, bool) {
	indexName, baseName, ok := l.whileRangeCondition(stmt.Cond)
	if !ok {
		return whileRangeProof{}, false
	}
	if !l.zeroLocals[indexName] {
		return whileRangeProof{}, false
	}
	if !l.whileBodyHasUnitIncrement(stmt.Body, indexName) {
		return whileRangeProof{}, false
	}
	if l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] {
		return whileRangeProof{}, false
	}
	return whileRangeProof{
		indexName: indexName,
		baseName:  baseName,
		proofID:   whileBoundsProofID(indexName, baseName, stmt.At),
		active:    true,
	}, true
}

func (l *lowerer) ifRangeProof(stmt *frontend.IfStmt) (whileRangeProof, bool) {
	indexName, baseName, ok := branchRangeCondition(stmt.Cond)
	if !ok {
		indexName, baseName, ok = whileRangeCondition(stmt.Cond)
		if !ok || !l.zeroLocals[indexName] {
			return whileRangeProof{}, false
		}
	}
	if l.externalSliceLocals[baseName] || l.invalidSliceLocals[baseName] {
		return whileRangeProof{}, false
	}
	return whileRangeProof{
		indexName: indexName,
		baseName:  baseName,
		proofID:   ifBoundsProofID(indexName, baseName, stmt.At),
		active:    true,
	}, true
}

func (l *lowerer) pushWhileRangeProof(proof whileRangeProof) {
	l.whileRangeProofs = append(l.whileRangeProofs, proof)
}

func (l *lowerer) popWhileRangeProof() {
	l.whileRangeProofs = l.whileRangeProofs[:len(l.whileRangeProofs)-1]
}

func (l *lowerer) invalidateWhileRangeProofForLocal(name string) {
	for i := range l.whileRangeProofs {
		if lowerProofPathMatchesMutation(l.whileRangeProofs[i].indexName, name) || lowerProofPathMatchesMutation(l.whileRangeProofs[i].baseName, name) {
			l.whileRangeProofs[i].active = false
		}
	}
}

func (l *lowerer) invalidateWhileRangeProofsForInoutArgs(args []frontend.Expr, ownership []string) {
	if len(args) == 0 || len(ownership) == 0 {
		return
	}
	for i, owner := range ownership {
		if owner != "inout" {
			continue
		}
		if i >= len(args) {
			break
		}
		path := simpleExprPath(args[i])
		if path == "" {
			continue
		}
		l.invalidateWhileRangeProofForLocal(path)
	}
}

func lowerProofPathMatchesMutation(proofPath string, mutatedPath string) bool {
	return lowerrangeproof.PathMatchesMutation(proofPath, mutatedPath)
}

func (l *lowerer) activeWhileProofForIndex(index *frontend.IndexExpr) (string, bool) {
	baseName := simpleExprPath(index.Base)
	indexName := simpleExprPath(index.Index)
	if baseName == "" || indexName == "" {
		return "", false
	}
	for i := len(l.whileRangeProofs) - 1; i >= 0; i-- {
		proof := l.whileRangeProofs[i]
		if proof.active && proof.baseName == baseName && proof.indexName == indexName {
			return proof.proofID, true
		}
	}
	return "", false
}

func (l *lowerer) rememberRangeMetadataForLocal(name string, expr frontend.Expr) {
	l.forgetLenBoundsForBase(name)
	if value, ok := l.proofConstIntValue(expr); ok {
		l.zeroLocals[name] = value == 0
		l.constIntLocals[name] = value
	} else {
		l.zeroLocals[name] = isZeroLiteral(expr)
		delete(l.constIntLocals, name)
	}
	if base := lenFieldBaseName(expr); base != "" {
		l.lenBoundLocals[name] = base
	} else {
		delete(l.lenBoundLocals, name)
	}
	if lengthName, ok := l.allocationLengthBoundLocal(expr); ok {
		l.lenBoundLocals[lengthName] = name
	}
	l.externalSliceLocals[name] = l.exprHasExternalSliceProvenance(expr)
	l.invalidSliceLocals[name] = l.exprIsInvalidSliceView(expr)
}

func (l *lowerer) forgetLenBoundsForBase(baseName string) {
	if baseName == "" {
		return
	}
	for name, base := range l.lenBoundLocals {
		if base == baseName {
			delete(l.lenBoundLocals, name)
		}
	}
}

func (l *lowerer) allocationLengthBoundLocal(expr frontend.Expr) (string, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil || len(call.Args) != 1 {
		return "", false
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	switch name {
	case "core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool":
	default:
		return "", false
	}
	id, ok := call.Args[0].(*frontend.IdentExpr)
	if !ok || id == nil || id.Name == "" {
		return "", false
	}
	info, ok := l.locals[id.Name]
	if !ok || info.Mutable {
		return "", false
	}
	return id.Name, true
}

type rangeMetadataState struct {
	zero     map[string]bool
	constInt map[string]int64
	lenBound map[string]string
	external map[string]bool
	invalid  map[string]bool
}

func (l *lowerer) snapshotRangeMetadata() rangeMetadataState {
	return rangeMetadataState{
		zero:     cloneLowerBoolMap(l.zeroLocals),
		constInt: cloneLowerInt64Map(l.constIntLocals),
		lenBound: cloneLowerStringMap(l.lenBoundLocals),
		external: cloneLowerBoolMap(l.externalSliceLocals),
		invalid:  cloneLowerBoolMap(l.invalidSliceLocals),
	}
}

func (l *lowerer) restoreRangeMetadata(state rangeMetadataState) {
	l.zeroLocals = cloneLowerBoolMap(state.zero)
	l.constIntLocals = cloneLowerInt64Map(state.constInt)
	l.lenBoundLocals = cloneLowerStringMap(state.lenBound)
	l.externalSliceLocals = cloneLowerBoolMap(state.external)
	l.invalidSliceLocals = cloneLowerBoolMap(state.invalid)
}

func (l *lowerer) mergeRangeMetadata(thenState rangeMetadataState, elseState rangeMetadataState) {
	keys := map[string]bool{}
	for key := range thenState.zero {
		keys[key] = true
	}
	for key := range elseState.zero {
		keys[key] = true
	}
	for key := range thenState.constInt {
		keys[key] = true
	}
	for key := range elseState.constInt {
		keys[key] = true
	}
	for key := range thenState.lenBound {
		keys[key] = true
	}
	for key := range elseState.lenBound {
		keys[key] = true
	}
	for key := range thenState.external {
		keys[key] = true
	}
	for key := range elseState.external {
		keys[key] = true
	}
	for key := range thenState.invalid {
		keys[key] = true
	}
	for key := range elseState.invalid {
		keys[key] = true
	}
	for key := range keys {
		l.zeroLocals[key] = thenState.zero[key] && elseState.zero[key]
		if thenValue, thenOK := thenState.constInt[key]; thenOK {
			if elseValue, elseOK := elseState.constInt[key]; elseOK && thenValue == elseValue {
				l.constIntLocals[key] = thenValue
			} else {
				delete(l.constIntLocals, key)
			}
		} else {
			delete(l.constIntLocals, key)
		}
		if thenValue, thenOK := thenState.lenBound[key]; thenOK {
			if elseValue, elseOK := elseState.lenBound[key]; elseOK && thenValue == elseValue {
				l.lenBoundLocals[key] = thenValue
			} else {
				delete(l.lenBoundLocals, key)
			}
		} else {
			delete(l.lenBoundLocals, key)
		}
		l.externalSliceLocals[key] = thenState.external[key] || elseState.external[key]
		l.invalidSliceLocals[key] = thenState.invalid[key] || elseState.invalid[key]
	}
}

func cloneLowerBoolMap(in map[string]bool) map[string]bool {
	return lowerrangeproof.CloneBoolMap(in)
}

func cloneLowerInt64Map(in map[string]int64) map[string]int64 {
	return lowerrangeproof.CloneInt64Map(in)
}

func cloneLowerStringMap(in map[string]string) map[string]string {
	return lowerrangeproof.CloneStringMap(in)
}

func (l *lowerer) whileRangeCondition(cond frontend.Expr) (string, string, bool) {
	indexName, baseName, _, ok := l.rangeFromCondition(cond)
	return indexName, baseName, ok
}

func (l *lowerer) rangeFromCondition(cond frontend.Expr) (string, string, corerangeproof.Range, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return "", "", corerangeproof.Range{}, false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil {
		return "", "", corerangeproof.Range{}, false
	}
	switch bin.Op {
	case frontend.TokenLess, frontend.TokenBangEq:
		base := l.lenBoundBaseName(bin.Right)
		if base == "" {
			return "", "", corerangeproof.Range{}, false
		}
		return left.Name, base, corerangeproof.LessThanLen(left.Name, base), true
	case frontend.TokenLessEq:
		base := lenMinusOneBaseName(bin.Right)
		if base == "" {
			return "", "", corerangeproof.Range{}, false
		}
		return left.Name, base, corerangeproof.LessEqualLenMinusOne(left.Name, base), true
	default:
		return "", "", corerangeproof.Range{}, false
	}
}

func staticRangeFromCondition(cond frontend.Expr) (string, string, corerangeproof.Range, bool) {
	return lowerrangeproof.StaticRangeFromCondition(cond)
}

func staticRangeCondition(cond frontend.Expr) (string, string, bool) {
	return lowerrangeproof.StaticRangeCondition(cond)
}

func whileRangeCondition(cond frontend.Expr) (string, string, bool) {
	return lowerrangeproof.WhileRangeCondition(cond)
}

func staticWhileRangeCondition(cond frontend.Expr) (string, string, bool) {
	return lowerrangeproof.StaticWhileRangeCondition(cond)
}

func branchRangeCondition(cond frontend.Expr) (string, string, bool) {
	return lowerrangeproof.BranchRangeCondition(cond)
}

func (l *lowerer) lenBoundBaseName(expr frontend.Expr) string {
	if base := lenFieldBaseName(expr); base != "" {
		return base
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return ""
	}
	return l.lenBoundLocals[id.Name]
}

func branchRangeConditionParts(lower frontend.Expr, upper frontend.Expr) (string, string, bool) {
	return lowerrangeproof.BranchRangeConditionParts(lower, upper)
}

func nonNegativeGuardIndex(expr frontend.Expr) (string, bool) {
	return lowerrangeproof.NonNegativeGuardIndex(expr)
}

func isZeroNumber(expr frontend.Expr) bool {
	return lowerrangeproof.IsZeroNumber(expr)
}

func lenFieldBaseName(expr frontend.Expr) string {
	return lowerrangeproof.LenFieldBaseName(expr)
}

func lenMinusOneBaseName(expr frontend.Expr) string {
	return lowerrangeproof.LenMinusOneBaseName(expr)
}

func (l *lowerer) whileBodyHasUnitIncrement(stmts []frontend.Stmt, indexName string) bool {
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if !ok || target.Name != indexName {
			continue
		}
		if l.isUnitIncrementExpr(assign.Value, indexName) {
			return true
		}
	}
	return false
}

func (l *lowerer) isUnitIncrementExpr(expr frontend.Expr, indexName string) bool {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenPlus {
		return false
	}
	if left, ok := bin.Left.(*frontend.IdentExpr); ok && left.Name == indexName {
		return l.isUnitStepExpr(bin.Right)
	}
	if right, ok := bin.Right.(*frontend.IdentExpr); ok && right.Name == indexName {
		return l.isUnitStepExpr(bin.Left)
	}
	return false
}

func (l *lowerer) isUnitStepExpr(expr frontend.Expr) bool {
	if num, ok := expr.(*frontend.NumberExpr); ok && num != nil {
		return num.Value == 1
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return false
	}
	info, ok := l.locals[id.Name]
	if !ok || info.Mutable {
		return false
	}
	value, ok := l.constIntLocals[id.Name]
	return ok && value == 1
}

func (l *lowerer) proofConstIntValue(expr frontend.Expr) (int64, bool) {
	if value, ok := evalConstInt64ForAllocation(expr); ok {
		return value, true
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return 0, false
	}
	info, ok := l.locals[id.Name]
	if !ok || info.Mutable {
		return 0, false
	}
	value, ok := l.constIntLocals[id.Name]
	return value, ok
}

func isZeroLiteral(expr frontend.Expr) bool {
	return lowerrangeproof.IsZeroLiteral(expr)
}

func (l *lowerer) collectionIterableProofAllowed(expr frontend.Expr) bool {
	if expr == nil {
		return false
	}
	return !l.exprHasExternalSliceProvenance(expr) && !l.exprIsInvalidSliceView(expr)
}

func isRawSliceConstructor(expr frontend.Expr) bool {
	return lowerrangeproof.IsRawSliceConstructor(expr)
}

func rawSliceElementShift(name string) int32 {
	return lowerrangeproof.RawSliceElementShift(name)
}

func (l *lowerer) exprHasExternalSliceProvenance(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return l.externalSliceLocals[e.Name]
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = target
		}
		if isRawSliceConstructor(&frontend.CallExpr{Name: name}) {
			return true
		}
		if isSliceCopyBuiltinName(name) {
			return false
		}
		if isBorrowOrViewBuiltinName(name) {
			return len(e.Args) == 0 || l.exprHasExternalSliceProvenance(e.Args[0])
		}
	}
	return false
}

func (l *lowerer) exprIsInvalidSliceView(expr frontend.Expr) bool {
	if staticInvalidCollectionIterable(expr) {
		return true
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return l.invalidSliceLocals[e.Name]
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = target
		}
		if isSliceCopyBuiltinName(name) {
			return false
		}
		if isBorrowOrViewBuiltinName(name) {
			return len(e.Args) > 0 && l.exprIsInvalidSliceView(e.Args[0])
		}
	}
	return false
}

func isSliceCopyBuiltinName(name string) bool {
	return lowerrangeproof.IsSliceCopyBuiltinName(name)
}

func isBorrowOrViewBuiltinName(name string) bool {
	return lowerrangeproof.IsBorrowOrViewBuiltinName(name)
}

func simpleExprPath(expr frontend.Expr) string {
	return lowerrangeproof.SimpleExprPath(expr)
}

func whileBoundsProofID(indexName string, baseName string, pos frontend.Position) string {
	return lowerrangeproof.WhileBoundsProofID(indexName, baseName, pos)
}

func ifBoundsProofID(indexName string, baseName string, pos frontend.Position) string {
	return lowerrangeproof.IfBoundsProofID(indexName, baseName, pos)
}

func rangeBoundsProofID(kind string, indexName string, baseName string, pos frontend.Position) string {
	return lowerrangeproof.RangeBoundsProofID(kind, indexName, baseName, pos)
}

func copyLoopBoundsProofID(name string, pos frontend.Position) string {
	return lowerrangeproof.CopyLoopBoundsProofID(name, pos)
}

func forCollectionBoundsProofID(stmt *frontend.ForRangeStmt) string {
	return lowerrangeproof.ForCollectionBoundsProofID(stmt)
}

func isViewCollectionIterable(expr frontend.Expr) bool {
	return lowerrangeproof.IsViewCollectionIterable(expr)
}
