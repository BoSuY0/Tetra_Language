package plir

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/rangeproof"
	"tetra_language/compiler/internal/semantics"
	semanticsresources "tetra_language/compiler/internal/semantics/resources"
)

const moduloRangeProofKeyPrefix = "\x00modulo-range:"
const moduloRangeProofFieldSep = "\x00"
const allocationConstLengthKeyPrefix = "\x00allocation-const-length:"
const nonNegativeWhileProofBase = "\x00non-negative"
const nonNegativeConstLoopProofBase = "\x00non-negative-const-loop"

type moduloRangeProof struct {
	baseName     string
	divisorName  string
	proofID      string
	indexValueID string
	rangeText    string
}

func (b *builder) ensureViewValue(localName string, base string, pos frontend.Position) string {
	if localName == "" {
		localName = base
	}
	id := valueID(ValueView, localName)
	if _, ok := b.values[id]; ok {
		return id
	}
	typeName := ""
	if info, ok := b.fn.Locals[localName]; ok {
		typeName = info.TypeName
	}
	if typeName == "" && base != "" {
		if info, ok := b.fn.Locals[base]; ok {
			typeName = info.TypeName
		}
	}
	value := Value{
		ID:         id,
		Kind:       ValueView,
		Type:       typeName,
		Source:     sourceString(pos),
		Region:     "fn:" + b.fn.Name,
		Provenance: Provenance{Kind: ProvenanceParam, Root: "param:" + base},
		Lifetime:   Lifetime{Birth: sourceString(pos), Death: "loop:end", Owner: base},
		Borrow:     BorrowImm,
		Escape:     EscapeNoEscape,
	}
	b.addValue(value)
	b.addFact(
		Fact{
			Kind:    FactProvenanceKnown,
			ValueID: id,
			Reason:  "for collection iterable view preserves source provenance",
		},
	)
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
	b.addFact(Fact{Kind: FactBorrowedImm, ValueID: id})
	b.addFact(
		Fact{
			Kind:    FactNoEscape,
			ValueID: id,
			Reason:  "for collection iterable view may not escape its owner",
		},
	)
	return id
}

func (b *builder) addLoopIndex(s *frontend.ForRangeStmt) string {
	name := s.IndexLocal
	if name == "" {
		name = s.Name + ":index"
	}
	id := valueID(ValueLoopIndex, name)
	value := Value{
		ID:         id,
		Kind:       ValueLoopIndex,
		Type:       "i32",
		Source:     sourceString(s.At),
		Region:     "fn:" + b.fn.Name,
		Provenance: Provenance{Kind: ProvenanceStack, Root: name},
		Lifetime:   Lifetime{Birth: "loop:start", Death: "loop:end", Owner: name},
		Escape:     EscapeNoEscape,
	}
	b.addValue(value)
	return id
}

func (b *builder) addValue(value Value) {
	if value.ID == "" {
		value.ID = fmt.Sprintf("v%d", b.valueSeq)
		b.valueSeq++
	}
	b.values[value.ID] = value
}

func (b *builder) syntheticTargetName(prefix string, call *frontend.CallExpr) string {
	if path := callResultPath(call); path != "" {
		return path
	}
	if call != nil && (call.At.Line != 0 || call.At.Col != 0) {
		return syntheticCallPath(prefix, call)
	}
	name := fmt.Sprintf("%s_%d", prefix, b.valueSeq)
	b.valueSeq++
	return name
}

func (b *builder) addFact(fact Fact) {
	if fact.ID == "" {
		fact.ID = fmt.Sprintf("f%d", b.factSeq)
		b.factSeq++
	}
	if fact.ValueID != "" {
		if value, ok := b.values[fact.ValueID]; ok && value.Provenance.Kind == ProvenanceIsland {
			root := value.Provenance.Root
			if root == "" {
				root = "unknown"
			}
			if fact.IslandID == "" {
				fact.IslandID = "island:" + root
			}
			if fact.Epoch == 0 {
				fact.Epoch = 1
			}
			if fact.BaseID == "" {
				fact.BaseID = value.ID
			}
		}
	}
	b.facts = append(b.facts, fact)
}

func (b *builder) addOperation(op Operation) Operation {
	if op.ID == "" {
		op.ID = fmt.Sprintf("op%d", b.opSeq)
		b.opSeq++
	}
	if op.Block == "" {
		op.Block = b.current
	}
	b.ops = append(b.ops, op)
	if op.Block != "" {
		b.appendBlockOp(op.Block, op.ID)
	}
	return op
}

func (b *builder) newBlock(kind string, pos frontend.Position, entry bool) string {
	id := kind
	if entry {
		id = "entry"
	} else {
		id = fmt.Sprintf("%s:%d", kind, b.blockSeq)
		b.blockSeq++
	}
	block := BasicBlock{ID: id, Kind: kind, Entry: entry, Source: sourceString(pos)}
	b.blockIndex[id] = len(b.blocks)
	b.blocks = append(b.blocks, block)
	return id
}

func (b *builder) appendBlockOp(blockID string, opID string) {
	idx, ok := b.blockIndex[blockID]
	if !ok {
		return
	}
	for _, existing := range b.blocks[idx].Ops {
		if existing == opID {
			return
		}
	}
	b.blocks[idx].Ops = append(b.blocks[idx].Ops, opID)
}

func (b *builder) addEdge(from string, to string) {
	if from == "" || to == "" || from == to {
		return
	}
	fromIdx, fromOK := b.blockIndex[from]
	toIdx, toOK := b.blockIndex[to]
	if !fromOK || !toOK {
		return
	}
	if !containsString(b.blocks[fromIdx].Succs, to) {
		b.blocks[fromIdx].Succs = append(b.blocks[fromIdx].Succs, to)
		sort.Strings(b.blocks[fromIdx].Succs)
	}
	if !containsString(b.blocks[toIdx].Preds, from) {
		b.blocks[toIdx].Preds = append(b.blocks[toIdx].Preds, from)
		sort.Strings(b.blocks[toIdx].Preds)
	}
}

func (b *builder) markExit(blockID string) {
	if idx, ok := b.blockIndex[blockID]; ok {
		b.blocks[idx].Exit = true
	}
}

func (b *builder) attachProofUses(fn *Function) {
	if len(fn.ProofGuards) == 0 || len(fn.ProofUses) == 0 {
		return
	}
	for i := range fn.ProofGuards {
		for _, use := range fn.ProofUses {
			if use.ProofID == fn.ProofGuards[i].ID {
				fn.ProofGuards[i].Dominates = append(fn.ProofGuards[i].Dominates, use)
			}
		}
	}
}

type localProofState struct {
	zero     map[string]bool
	constInt map[string]int64
	lenBound map[string]string
	external map[string]bool
	invalid  map[string]bool
}

func (b *builder) snapshotLocalProofState() localProofState {
	return localProofState{
		zero:     cloneBoolMap(b.zeroLocals),
		constInt: cloneInt64Map(b.constIntLocals),
		lenBound: cloneStringMap(b.lenBoundLocals),
		external: cloneBoolMap(b.externalLocals),
		invalid:  cloneBoolMap(b.invalidLocals),
	}
}

func (b *builder) restoreLocalProofState(state localProofState) {
	b.zeroLocals = cloneBoolMap(state.zero)
	b.constIntLocals = cloneInt64Map(state.constInt)
	b.lenBoundLocals = cloneStringMap(state.lenBound)
	b.externalLocals = cloneBoolMap(state.external)
	b.invalidLocals = cloneBoolMap(state.invalid)
}

func (b *builder) mergeLocalProofState(thenState localProofState, elseState localProofState) {
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
		b.zeroLocals[key] = thenState.zero[key] && elseState.zero[key]
		if thenValue, thenOK := thenState.constInt[key]; thenOK {
			if elseValue, elseOK := elseState.constInt[key]; elseOK && thenValue == elseValue {
				b.constIntLocals[key] = thenValue
			} else {
				delete(b.constIntLocals, key)
			}
		} else {
			delete(b.constIntLocals, key)
		}
		if thenValue, thenOK := thenState.lenBound[key]; thenOK {
			if elseValue, elseOK := elseState.lenBound[key]; elseOK && thenValue == elseValue {
				b.lenBoundLocals[key] = thenValue
			} else {
				delete(b.lenBoundLocals, key)
			}
		} else {
			delete(b.lenBoundLocals, key)
		}
		b.externalLocals[key] = thenState.external[key] || elseState.external[key]
		b.invalidLocals[key] = thenState.invalid[key] || elseState.invalid[key]
	}
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneInt64Map(in map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (b *builder) pushActiveProof(proof rangeProof) {
	b.activeProof = append(b.activeProof, proof)
}

func (b *builder) popActiveProof() {
	b.activeProof = b.activeProof[:len(b.activeProof)-1]
}

func (b *builder) invalidateActiveProofForLocal(name string) {
	for i := range b.activeProof {
		if proofPathMatchesMutation(b.activeProof[i].IndexName, name) ||
			proofPathMatchesMutation(b.activeProof[i].Base, name) ||
			proofPathMatchesMutation(b.activeProof[i].AffineLeftName, name) ||
			proofPathMatchesMutation(b.activeProof[i].AffineRightName, name) {
			b.activeProof[i].ID = ""
		}
	}
	b.forgetAllocationConstLengthForBase(name)
	b.forgetModuloRangeProofForLocal(name)
}

func (b *builder) invalidateActiveProofsForMutableCallArgs(
	args []frontend.Expr,
	ownership []string,
) {
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
		path := exprPath(args[i])
		if path == "" {
			continue
		}
		b.invalidateActiveProofForLocal(path)
	}
}

func (b *builder) invalidateNoAliasForMutableCallArgs(
	args []frontend.Expr,
	ownership []string,
	reason string,
) {
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
		b.invalidateNoAliasForPath(exprPath(args[i]), reason)
	}
}

func (b *builder) invalidateNoAliasForCallInputs(args []frontend.Expr, reason string) {
	for _, arg := range args {
		b.invalidateNoAliasForPath(exprPath(arg), reason)
	}
}

func (b *builder) invalidateNoAliasForPath(path string, reason string) {
	root := rootPath(path)
	if root == "" {
		return
	}
	b.noAliasInvalidatedRoots[root] = reason
}

func rootPath(path string) string {
	if path == "" {
		return ""
	}
	if idx := strings.IndexByte(path, '.'); idx >= 0 {
		return path[:idx]
	}
	return path
}

func callHasInoutArgument(args []frontend.Expr, ownership []string) bool {
	if len(args) == 0 || len(ownership) == 0 {
		return false
	}
	for i, owner := range ownership {
		if i >= len(args) {
			return false
		}
		if owner == "inout" && exprPath(args[i]) != "" {
			return true
		}
	}
	return false
}

func (b *builder) callParamOwnership(name string) []string {
	if name == "" {
		return nil
	}
	if b.funcs != nil {
		if sig, ok := b.funcs[name]; ok {
			return sig.ParamOwnership
		}
	}
	if local, ok := b.fn.Locals[name]; ok && local.FunctionTypeValue {
		return local.FunctionParamOwnership
	}
	if b.globals != nil {
		if global, ok := b.globals[name]; ok && global.FunctionTypeValue {
			return global.FunctionParamOwnership
		}
	}
	return nil
}

func (b *builder) callAliasBoundaryKind(name string) string {
	if name == "" {
		return ""
	}
	if local, ok := b.fn.Locals[name]; ok && local.FunctionTypeValue {
		return "function_typed_inout"
	}
	if b.globals != nil {
		if global, ok := b.globals[name]; ok && global.FunctionTypeValue {
			return "function_typed_inout"
		}
	}
	return ""
}

func (b *builder) callSummaryNote(name string) string {
	if !b.callSummaryUnknown(name) {
		return name
	}
	if name == "" {
		return "unknown external call"
	}
	return name + " unknown external call"
}

func (b *builder) callSummaryUnknown(name string) bool {
	if name == "" {
		return true
	}
	if strings.HasPrefix(name, "core.") {
		return false
	}
	if b.funcs != nil {
		if _, ok := b.funcs[name]; ok {
			return false
		}
	}
	if local, ok := b.fn.Locals[name]; ok && local.FunctionTypeValue {
		return false
	}
	if b.globals != nil {
		if global, ok := b.globals[name]; ok && global.FunctionTypeValue {
			return false
		}
	}
	return true
}

func appendOperationNote(note string, part string) string {
	if strings.TrimSpace(part) == "" {
		return note
	}
	if strings.Contains(note, part) {
		return note
	}
	if strings.TrimSpace(note) == "" {
		return part
	}
	return note + " " + part
}

func proofPathMatchesMutation(proofPath string, mutatedPath string) bool {
	if proofPath == "" || mutatedPath == "" {
		return false
	}
	proof := semanticsresources.Path(proofPath)
	mutated := semanticsresources.Path(mutatedPath)
	return proof == mutated || proof.IsDescendantOf(mutated)
}

func (b *builder) activeProofForIndex(index *frontend.IndexExpr) (rangeProof, bool) {
	base := exprPath(index.Base)
	idx := exprPath(index.Index)
	if base == "" {
		return rangeProof{}, false
	}
	if proof, ok := b.activeHelperOffsetProofForIndex(base, index); ok {
		return proof, true
	}
	if proof, ok := b.activeHelperSummaryProofForIndex(base, index); ok {
		return proof, true
	}
	if proof, ok := b.activeAllocationLiteralZeroProofForIndex(base, index); ok {
		return proof, true
	}
	if proof, ok := b.activeAffineConstExtentProofForIndex(base, index); ok {
		return proof, true
	}
	if idx != "" {
		for i := len(b.activeProof) - 1; i >= 0; i-- {
			proof := b.activeProof[i]
			if proof.ID != "" && proof.Base == base && proof.IndexName == idx {
				return proof, true
			}
		}
		if proof, ok := b.activeModuloRangeProofForIndex(base, idx); ok {
			return proof, true
		}
	}
	if proof, ok := b.activeModuloConstProofForIndex(base, index.Index); ok {
		return proof, true
	}
	return rangeProof{}, false
}

func (b *builder) activeHelperOffsetProofForIndex(
	base string,
	index *frontend.IndexExpr,
) (rangeProof, bool) {
	if base == "" || index == nil || b.helperOffsetProof.Empty() ||
		base != b.helperOffsetProof.ParamName {
		return rangeProof{}, false
	}
	operation := b.currentIndexProofOperation(base, index)
	access, ok := b.helperOffsetProof.AccessForIndex(index.Index, operation)
	if !ok {
		return rangeProof{}, false
	}
	indexName := strconv.FormatInt(access.ActualIndex, 10)
	latticeRange := rangeproof.LessThanLen(indexName, base)
	derivation := append([]string(nil), latticeRange.Derivation...)
	derivation = append(derivation, b.helperOffsetProof.Derivation(access)...)
	proof := rangeProof{
		ID:             rangeproof.HelperOffsetBoundsProofID(base, access.ActualIndex, index.Pos()),
		IndexName:      indexName,
		IndexValueID:   b.ensureProofIndexValue(indexName, index.Pos()),
		Base:           base,
		Condition:      b.helperOffsetProof.Condition(access),
		Operation:      operation,
		Source:         sourceString(index.Pos()),
		RangeText:      fmt.Sprintf("%d in [0, %s.len)", access.ActualIndex, base),
		Lower:          plirBoundFromRangeBound(latticeRange.Lower),
		Upper:          plirBoundFromRangeBound(latticeRange.Upper),
		InclusiveLower: latticeRange.InclusiveLower,
		InclusiveUpper: latticeRange.InclusiveUpper,
		Reason:         "helper-offset local call offset proof",
		Derivation:     derivation,
	}
	b.addRangeProof(proof, b.current, "")
	return proof, true
}

func (b *builder) activeHelperSummaryProofForIndex(
	base string,
	index *frontend.IndexExpr,
) (rangeProof, bool) {
	if base == "" || index == nil || b.helperSummaryProof.Empty() ||
		base != b.helperSummaryProof.ParamName {
		return rangeProof{}, false
	}
	operation := b.currentIndexProofOperation(base, index)
	if operation != "index_store" {
		return rangeProof{}, false
	}
	indexValue, ok := b.proofConstIntValue(index.Index)
	if !ok || indexValue < 0 {
		return rangeProof{}, false
	}
	if _, ok := b.helperSummaryProof.StoreForIndex(indexValue); !ok ||
		indexValue >= b.helperSummaryProof.Length {
		return rangeProof{}, false
	}
	indexName := strconv.FormatInt(indexValue, 10)
	latticeRange := rangeproof.LessThanLen(indexName, base)
	derivation := append([]string(nil), latticeRange.Derivation...)
	derivation = append(derivation, b.helperSummaryProof.Derivation()...)
	proof := rangeProof{
		ID:             rangeproof.HelperSummaryBoundsProofID(base, indexValue, index.Pos()),
		IndexName:      indexName,
		IndexValueID:   b.ensureProofIndexValue(indexName, index.Pos()),
		Base:           base,
		Condition:      b.helperSummaryProof.Condition(indexValue),
		Operation:      operation,
		Source:         sourceString(index.Pos()),
		RangeText:      fmt.Sprintf("%d in [0, %s.len)", indexValue, base),
		Lower:          plirBoundFromRangeBound(latticeRange.Lower),
		Upper:          plirBoundFromRangeBound(latticeRange.Upper),
		InclusiveLower: latticeRange.InclusiveLower,
		InclusiveUpper: latticeRange.InclusiveUpper,
		Reason:         "helper-summary local call length proof",
		Derivation:     derivation,
	}
	b.addRangeProof(proof, b.current, "")
	return proof, true
}

func (b *builder) activeAllocationLiteralZeroProofForIndex(
	base string,
	index *frontend.IndexExpr,
) (rangeProof, bool) {
	if base == "" || index == nil || !isZeroNumber(index.Index) {
		return rangeProof{}, false
	}
	info, ok := b.fn.Locals[base]
	if !ok || !strings.HasPrefix(info.TypeName, "[]") {
		return rangeProof{}, false
	}
	if b.externalLocals[base] || b.invalidLocals[base] {
		return rangeProof{}, false
	}
	length, ok := b.allocationConstLengthForBase(base)
	if !ok || length <= 0 {
		return rangeProof{}, false
	}
	indexName := "0"
	latticeRange := rangeproof.LessThanLen(indexName, base)
	derivation := append([]string(nil), latticeRange.Derivation...)
	derivation = append(derivation, "allocation_literal_zero", "allocation_length_positive")
	operation := b.currentIndexProofOperation(base, index)
	proof := rangeProof{
		ID: proofIDForRange(
			"allocation-zero",
			"literal0",
			base,
			index.Pos(),
		) + ":" + proofNamePart(
			operation,
			"index",
		),
		IndexName:      indexName,
		IndexValueID:   b.ensureProofIndexValue(indexName, index.Pos()),
		Base:           base,
		Condition:      fmt.Sprintf("0 < %s.len && %s.len == %d", base, base, length),
		Operation:      operation,
		Source:         sourceString(index.Pos()),
		RangeText:      rangeTextFromLattice(latticeRange),
		Lower:          plirBoundFromRangeBound(latticeRange.Lower),
		Upper:          plirBoundFromRangeBound(latticeRange.Upper),
		InclusiveLower: latticeRange.InclusiveLower,
		InclusiveUpper: latticeRange.InclusiveUpper,
		Reason:         "allocation literal-zero length proof",
		Derivation:     derivation,
	}
	b.addRangeProof(proof, b.current, "")
	return proof, true
}

func (b *builder) currentIndexProofOperation(base string, index *frontend.IndexExpr) string {
	if index == nil || len(b.ops) == 0 {
		return "index_load"
	}
	last := b.ops[len(b.ops)-1]
	if len(last.Inputs) >= 2 && last.Inputs[0] == base && last.Inputs[1] == exprPath(index.Index) {
		switch last.Kind {
		case OpIndexStore:
			return "index_store"
		case OpIndexLoad:
			return "index_load"
		}
	}
	return "index_load"
}

func (b *builder) activeAffineConstExtentProofForIndex(
	base string,
	index *frontend.IndexExpr,
) (rangeProof, bool) {
	if base == "" || index == nil || b.externalLocals[base] || b.invalidLocals[base] {
		return rangeProof{}, false
	}
	leftName, stride, rightName, ok := affineConstExtentIndexParts(index.Index)
	if !ok {
		return rangeProof{}, false
	}
	source := sourceString(index.Pos())
	for i := len(b.activeProof) - 1; i >= 0; i-- {
		proof := b.activeProof[i]
		if proof.ID == "" ||
			!strings.HasPrefix(proof.ID, "proof:affine-const:") ||
			proof.Base != base ||
			proof.AffineLeftName != leftName ||
			proof.AffineRightName != rightName ||
			proof.AffineStride != stride ||
			proof.Source != source {
			continue
		}
		length, ok := b.allocationConstLengthForBase(base)
		if !ok || length != 9 {
			return rangeProof{}, false
		}
		return proof, true
	}
	return rangeProof{}, false
}

func (b *builder) activeModuloConstProofForIndex(
	base string,
	index frontend.Expr,
) (rangeProof, bool) {
	if base == "" || b.externalLocals[base] || b.invalidLocals[base] {
		return rangeProof{}, false
	}
	indexName := exprPath(index)
	if indexName == "" {
		return rangeProof{}, false
	}
	numerator, divisorValue, ok := b.moduloConstDivisor(index)
	if !ok || divisorValue <= 0 {
		return rangeProof{}, false
	}
	length, ok := b.allocationConstLengthForBase(base)
	if !ok || length != divisorValue {
		return rangeProof{}, false
	}
	if !b.exprKnownNonNegative(numerator) {
		return rangeProof{}, false
	}
	latticeRange := rangeproof.LessThanLen(indexName, base)
	derivation := append([]string(nil), latticeRange.Derivation...)
	derivation = append(derivation, "modulo_const_allocation_length")
	proof := rangeProof{
		ID:             proofIDForRange("modulo", "modulo_const", base, index.Pos()),
		IndexName:      indexName,
		IndexValueID:   b.ensureProofIndexValue(indexName, index.Pos()),
		Base:           base,
		Condition:      fmt.Sprintf("%s = modulo_const(%d)", indexName, divisorValue),
		Source:         sourceString(index.Pos()),
		RangeText:      rangeTextFromLattice(latticeRange),
		Lower:          plirBoundFromRangeBound(latticeRange.Lower),
		Upper:          plirBoundFromRangeBound(latticeRange.Upper),
		InclusiveLower: latticeRange.InclusiveLower,
		InclusiveUpper: latticeRange.InclusiveUpper,
		Reason:         "modulo const allocation length proof",
		Derivation:     derivation,
	}
	b.addRangeProof(proof, b.current, "")
	return proof, true
}

func (b *builder) activeModuloRangeProofForIndex(base string, indexName string) (rangeProof, bool) {
	if base == "" || indexName == "" {
		return rangeProof{}, false
	}
	proof, ok := b.moduloRangeProofForLocal(indexName)
	if !ok || proof.baseName != base {
		return rangeProof{}, false
	}
	if b.externalLocals[base] || b.invalidLocals[base] {
		return rangeProof{}, false
	}
	if currentBase := b.lenBoundLocals[proof.divisorName]; currentBase != base {
		return rangeProof{}, false
	}
	latticeRange := rangeproof.LessThanLen(indexName, base)
	derivation := append([]string(nil), latticeRange.Derivation...)
	derivation = append(derivation, "modulo_allocation_length_alias")
	return rangeProof{
		ID:             proof.proofID,
		IndexName:      indexName,
		IndexValueID:   proof.indexValueID,
		Base:           base,
		Condition:      indexName + " = modulo(" + proof.divisorName + ")",
		RangeText:      proof.rangeText,
		Lower:          plirBoundFromRangeBound(latticeRange.Lower),
		Upper:          plirBoundFromRangeBound(latticeRange.Upper),
		InclusiveLower: latticeRange.InclusiveLower,
		InclusiveUpper: latticeRange.InclusiveUpper,
		Reason:         "modulo allocation length alias proof",
		Derivation:     derivation,
	}, true
}

func (b *builder) addRangeProof(proof rangeProof, truthBlock string, opID string) {
	if proof.ID == "" {
		return
	}
	b.addFact(Fact{
		Kind:    FactIndexInRange,
		ValueID: proof.IndexValueID,
		Range:   proof.RangeText,
		ProofID: proof.ID,
		Source:  sourceStringFromProofSource(proof),
		Reason:  proof.Reason,
	})
	b.proofGuards = append(b.proofGuards, ProofGuard{
		ID:        proof.ID,
		Kind:      "range",
		Block:     truthBlock,
		OpID:      opID,
		Condition: proof.Condition,
		Reason:    proof.Reason,
	})
	b.rangeFacts = append(b.rangeFacts, RangeFact{
		Value:          proof.IndexValueID,
		Lower:          proof.Lower,
		Upper:          proof.Upper,
		InclusiveLower: proof.InclusiveLower,
		InclusiveUpper: proof.InclusiveUpper,
		Source:         sourceStringFromProofSource(proof),
		ProofID:        proof.ID,
		Reason:         proof.Reason,
		Derivation:     append([]string(nil), proof.Derivation...),
	})
	b.addBoundsProofTerm(proof)
}

func (b *builder) addBoundsProofTerm(proof rangeProof) {
	if proof.ID == "" {
		return
	}
	for _, term := range b.proofTerms {
		if term.ID == proof.ID {
			return
		}
	}
	operation := proof.Operation
	if operation == "" {
		operation = "index_load"
	}
	term := ProofTerm{
		ID:            proof.ID,
		Kind:          "bounds_check",
		SubjectBaseID: proof.Base,
		IndexValueID:  proof.IndexValueID,
		Operation:     operation,
		Range:         proof.RangeText,
		Source:        sourceStringFromProofSource(proof),
		FactsUsed:     append([]string(nil), proof.Derivation...),
	}
	if ref, ok := b.proofMemoryRefForBase(proof.Base); ok {
		term.IslandID = ref.IslandID
		term.Epoch = ref.Epoch
		term.BaseID = ref.BaseID
	}
	b.proofTerms = append(b.proofTerms, term)
}

func (b *builder) proofMemoryRefForBase(base string) (islandTokenState, bool) {
	for _, id := range valueIDsForPath(base) {
		for _, fact := range b.facts {
			if fact.ValueID != id || fact.IslandID == "" || fact.Epoch <= 0 {
				continue
			}
			return islandTokenState{
				IslandID: fact.IslandID,
				Epoch:    fact.Epoch,
				BaseID:   fact.BaseID,
			}, true
		}
	}
	return islandTokenState{}, false
}

func sourceStringFromProofSource(proof rangeProof) string {
	if proof.Source != "" {
		return proof.Source
	}
	return proof.ID
}

func (b *builder) rememberAliasMetadata(name string, expr frontend.Expr) {
	if name == "" {
		return
	}
	b.externalLocals[name] = b.exprHasExternalProvenance(expr)
	b.invalidLocals[name] = b.exprIsInvalidView(expr)
	b.rememberIslandTokenAlias(name, expr)
	b.recordViewAlias(name, expr)
	if b.invalidLocals[name] {
		b.reclassifyMemoryBinding(
			name,
			Provenance{Kind: ProvenanceUnknown},
			"alias source is invalid before construction",
		)
		return
	}
	if b.externalLocals[name] {
		b.reclassifyMemoryBinding(
			name,
			b.conservativeProvenanceFromExpr(expr),
			"alias source has external or unknown provenance",
		)
	}
}

func (b *builder) recordViewAlias(name string, expr frontend.Expr) {
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return
	}
	sourceViewID := valueID(ValueView, id.Name)
	sourceView, ok := b.values[sourceViewID]
	if !ok {
		return
	}
	aliasID := valueID(ValueView, name)
	if _, exists := b.values[aliasID]; !exists {
		alias := sourceView
		alias.ID = aliasID
		alias.Source = sourceString(expr.Pos())
		alias.Lifetime = Lifetime{Birth: sourceString(expr.Pos()), Owner: name}
		b.addValue(alias)
	}
	b.copyDerivedWindowFacts(id.Name, aliasID, "alias preserves derived window range")
	if sourceView.Provenance.Kind == ProvenanceExternal ||
		sourceView.Provenance.Kind == ProvenanceUnknown {
		b.externalLocals[name] = true
		if !b.hasFactForValue(FactProvenanceUnknown, aliasID) {
			b.addFact(
				Fact{
					Kind:    FactProvenanceUnknown,
					ValueID: aliasID,
					Reason:  "alias source has external or unknown provenance",
				},
			)
		}
	}
}

func (b *builder) reclassifyMemoryBinding(name string, provenance Provenance, reason string) {
	if name == "" {
		return
	}
	if provenance.Kind == "" {
		provenance = Provenance{Kind: ProvenanceUnknown}
	}
	if provenance.Kind == ProvenanceExternal && provenance.Root == "" {
		provenance.Root = "external:" + name
	}
	for _, kind := range []ValueKind{ValueLocal, ValueParam} {
		id := valueID(kind, name)
		value, ok := b.values[id]
		if !ok || !isMemoryBackedType(value.Type) {
			continue
		}
		value.Provenance = provenance
		b.values[id] = value
		b.removeFactsForValue(id, FactProvenanceKnown, FactLenStable)
		if !b.hasFactForValue(FactProvenanceUnknown, id) {
			b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: id, Reason: reason})
		}
	}
}

func (b *builder) removeFactsForValue(valueID string, kinds ...FactKind) {
	if valueID == "" || len(kinds) == 0 {
		return
	}
	remove := map[FactKind]bool{}
	for _, kind := range kinds {
		remove[kind] = true
	}
	filtered := b.facts[:0]
	for _, fact := range b.facts {
		if fact.ValueID == valueID && remove[fact.Kind] {
			continue
		}
		filtered = append(filtered, fact)
	}
	b.facts = filtered
}

func (b *builder) hasFactForValue(kind FactKind, valueID string) bool {
	for _, fact := range b.facts {
		if fact.Kind == kind && fact.ValueID == valueID {
			return true
		}
	}
	return false
}

func (b *builder) conservativeProvenanceFromExpr(expr frontend.Expr) Provenance {
	if b.exprIsInvalidView(expr) {
		return Provenance{Kind: ProvenanceUnknown}
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		for _, kind := range []ValueKind{ValueView, ValueLocal, ValueParam, ValueAllocIntent} {
			value, ok := b.values[valueID(kind, e.Name)]
			if !ok {
				continue
			}
			switch value.Provenance.Kind {
			case ProvenanceExternal:
				root := value.Provenance.Root
				if root == "" {
					root = e.Name
				}
				return Provenance{Kind: ProvenanceExternal, Root: "derived:" + root}
			case ProvenanceUnknown, "":
				return Provenance{Kind: ProvenanceUnknown}
			}
		}
		if b.externalLocals[e.Name] {
			return Provenance{Kind: ProvenanceExternal, Root: "alias:" + e.Name}
		}
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = target
		}
		if rawSliceBuiltin(name) {
			return Provenance{Kind: ProvenanceExternal, Root: "raw_parts"}
		}
		if borrowOrViewBuiltin(name) {
			if len(e.Args) == 0 {
				return Provenance{Kind: ProvenanceUnknown}
			}
			return b.conservativeProvenanceFromExpr(e.Args[0])
		}
	}
	return Provenance{Kind: ProvenanceUnknown}
}

func (b *builder) exprHasExternalProvenance(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if b.externalLocals[e.Name] {
			return true
		}
		if value, ok := b.values[valueID(ValueView, e.Name)]; ok {
			return value.Provenance.Kind == ProvenanceExternal || value.Provenance.Kind == ProvenanceUnknown
		}
		return false
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = target
		}
		if rawSliceBuiltin(name) {
			return true
		}
		if copyBuiltin(name) {
			return false
		}
		if borrowOrViewBuiltin(name) {
			return len(e.Args) == 0 || b.exprHasExternalProvenance(e.Args[0])
		}
	}
	return false
}

func (b *builder) exprIsInvalidView(expr frontend.Expr) bool {
	if staticInvalidIterableExpr(expr) {
		return true
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return b.invalidLocals[e.Name]
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = target
		}
		if copyBuiltin(name) {
			return false
		}
		if borrowOrViewBuiltin(name) {
			return len(e.Args) > 0 && b.exprIsInvalidView(e.Args[0])
		}
	}
	return false
}

func (b *builder) collectionIterableProofAllowed(expr frontend.Expr) bool {
	if expr == nil {
		return false
	}
	return !b.exprHasExternalProvenance(expr) && !b.exprIsInvalidView(expr)
}

func (b *builder) whileRangeProof(s *frontend.WhileStmt) (rangeProof, bool) {
	proof, ok := b.rangeProofFromCondition(s.Cond, s.At)
	if !ok {
		return rangeProof{}, false
	}
	if b.externalLocals[proof.Base] || b.invalidLocals[proof.Base] {
		return rangeProof{}, false
	}
	if !b.zeroLocals[proof.IndexName] {
		return rangeProof{}, false
	}
	if !b.bodyHasUnitIncrement(s.Body, proof.IndexName) {
		return rangeProof{}, false
	}
	proof.ID = proofIDForRange("while", proof.IndexName, proof.Base, s.At)
	proof.Reason = "while loop range proof"
	return proof, true
}

func (b *builder) whileRangeProofs(s *frontend.WhileStmt) []rangeProof {
	if proof, ok := b.whileRangeProof(s); ok {
		proofs := b.constLoopAllocationLengthProofsExcept(s, map[string]bool{proof.Base: true})
		proofs = append(proofs, proof)
		return proofs
	}
	if proofs := b.callBoundaryRangeProofs(s); len(proofs) > 0 {
		return proofs
	}
	proofs := b.constLoopAllocationLengthProofs(s)
	if affineProofs := b.affineConstExtentProofsForWhile(s); len(affineProofs) > 0 {
		proofs = append(proofs, affineProofs...)
		return proofs
	}
	if proof, ok := b.nonNegativeWhileRangeProof(s); ok {
		proofs = append(proofs, proof)
	}
	return proofs
}

func (b *builder) callBoundaryRangeProofs(s *frontend.WhileStmt) []rangeProof {
	indexName, upperName, ok := callBoundaryLoopCondition(s.Cond)
	if !ok {
		return nil
	}
	if !b.zeroLocals[indexName] {
		return nil
	}
	if !b.bodyHasExactlyOneUnitIncrement(s.Body, indexName) {
		return nil
	}
	allowed := map[string]bool{}
	for _, base := range b.callBoundaryLenProof.BasesForUpper(upperName) {
		if base != "" {
			allowed[base] = true
		}
	}
	if len(allowed) == 0 {
		return nil
	}
	bases := b.callBoundaryDirectIndexLoadBases(s.Body, indexName)
	if len(bases) == 0 {
		return nil
	}
	proofs := make([]rangeProof, 0, len(bases))
	for _, base := range bases {
		if !allowed[base.name] || b.externalLocals[base.name] || b.invalidLocals[base.name] {
			continue
		}
		latticeRange := rangeproof.LessThanLen(indexName, base.name)
		derivation := append([]string(nil), latticeRange.Derivation...)
		derivation = append(derivation, "call_boundary_length")
		proofs = append(proofs, rangeProof{
			ID:           proofIDForRange("call-boundary", indexName, base.name, base.pos),
			IndexName:    indexName,
			IndexValueID: b.ensureProofIndexValue(indexName, s.At),
			Base:         base.name,
			Condition: fmt.Sprintf(
				"%s < %s && %s <= %s.len",
				indexName,
				upperName,
				upperName,
				base.name,
			),
			Operation:      "index_load",
			Source:         sourceString(base.pos),
			RangeText:      rangeTextFromLattice(latticeRange),
			Lower:          plirBoundFromRangeBound(latticeRange.Lower),
			Upper:          plirBoundFromRangeBound(latticeRange.Upper),
			InclusiveLower: latticeRange.InclusiveLower,
			InclusiveUpper: latticeRange.InclusiveUpper,
			Reason:         "call-boundary length proof",
			Derivation:     derivation,
		})
	}
	return proofs
}

func callBoundaryLoopCondition(cond frontend.Expr) (string, string, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenLess {
		return "", "", false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil || left.Name == "" {
		return "", "", false
	}
	right, ok := bin.Right.(*frontend.IdentExpr)
	if !ok || right == nil || right.Name == "" {
		return "", "", false
	}
	return left.Name, right.Name, true
}

type callBoundaryIndexLoadBase struct {
	name string
	pos  frontend.Position
}

func (b *builder) callBoundaryDirectIndexLoadBases(
	stmts []frontend.Stmt,
	indexName string,
) []callBoundaryIndexLoadBase {
	seen := map[string]frontend.Position{}
	var walkStmt func(frontend.Stmt)
	var walkExpr func(frontend.Expr)
	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.IndexExpr:
			if e != nil && exprPath(e.Index) == indexName {
				base := exprPath(e.Base)
				if base != "" {
					if _, ok := seen[base]; !ok {
						seen[base] = e.Pos()
					}
				}
			}
			if e != nil {
				walkExpr(e.Base)
				walkExpr(e.Index)
			}
		case *frontend.BinaryExpr:
			if e != nil {
				walkExpr(e.Left)
				walkExpr(e.Right)
			}
		case *frontend.UnaryExpr:
			if e != nil {
				walkExpr(e.X)
			}
		case *frontend.CallExpr:
			if e != nil {
				for _, arg := range e.Args {
					walkExpr(arg)
				}
			}
		case *frontend.FieldAccessExpr:
			if e != nil {
				walkExpr(e.Base)
			}
		case *frontend.StructLitExpr:
			if e != nil {
				for _, field := range e.Fields {
					walkExpr(field.Value)
				}
			}
		case *frontend.MatchExpr:
			if e != nil {
				walkExpr(e.Value)
				for _, c := range e.Cases {
					walkExpr(c.Guard)
					walkExpr(c.Value)
				}
			}
		case *frontend.CatchExpr:
			if e != nil {
				walkExpr(e.Call)
				for _, c := range e.Cases {
					walkExpr(c.Guard)
					walkExpr(c.Value)
				}
			}
		case *frontend.TryExpr:
			if e != nil {
				walkExpr(e.X)
			}
		case *frontend.AwaitExpr:
			if e != nil {
				walkExpr(e.X)
			}
		}
	}
	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ExprStmt:
			walkExpr(s.Expr)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, nested := range s.Then {
				walkStmt(nested)
			}
			for _, nested := range s.Else {
				walkStmt(nested)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value)
			for _, nested := range s.Then {
				walkStmt(nested)
			}
			for _, nested := range s.Else {
				walkStmt(nested)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, nested := range s.Body {
				walkStmt(nested)
			}
		case *frontend.ForRangeStmt:
			walkExpr(s.Start)
			walkExpr(s.End)
			walkExpr(s.Iterable)
			for _, nested := range s.Body {
				walkStmt(nested)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				walkExpr(c.Guard)
				for _, nested := range c.Body {
					walkStmt(nested)
				}
			}
		case *frontend.UnsafeStmt:
			for _, nested := range s.Body {
				walkStmt(nested)
			}
		case *frontend.DeferStmt:
			for _, nested := range s.Body {
				walkStmt(nested)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, nested := range s.Body {
				walkStmt(nested)
			}
		}
	}
	for _, stmt := range stmts {
		walkStmt(stmt)
	}
	out := make([]callBoundaryIndexLoadBase, 0, len(seen))
	for base, pos := range seen {
		out = append(out, callBoundaryIndexLoadBase{name: base, pos: pos})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func (b *builder) constLoopAllocationLengthProofs(s *frontend.WhileStmt) []rangeProof {
	return b.constLoopAllocationLengthProofsExcept(s, nil)
}

func (b *builder) constLoopAllocationLengthProofsExcept(
	s *frontend.WhileStmt,
	exclude map[string]bool,
) []rangeProof {
	indexName, upper, ok := b.constLoopCondition(s.Cond)
	if !ok || upper <= 0 {
		return nil
	}
	if !b.zeroLocals[indexName] {
		return nil
	}
	if !b.bodyHasExactlyOneUnitIncrement(s.Body, indexName) {
		return nil
	}
	bases := b.constLoopDirectIndexStoreBases(s.Body, indexName)
	if len(bases) == 0 {
		return nil
	}
	proofs := make([]rangeProof, 0, len(bases))
	for _, base := range bases {
		if exclude != nil && exclude[base.name] {
			continue
		}
		if b.externalLocals[base.name] || b.invalidLocals[base.name] {
			continue
		}
		length, ok := b.allocationConstLengthForBase(base.name)
		if !ok || length != upper {
			continue
		}
		latticeRange := rangeproof.LessThanLen(indexName, base.name)
		derivation := append([]string(nil), latticeRange.Derivation...)
		derivation = append(derivation, "const_loop_allocation_length")
		proofs = append(proofs, rangeProof{
			ID:           proofIDForRange("while-const", indexName, base.name, base.pos),
			IndexName:    indexName,
			IndexValueID: b.valueIDForName(indexName),
			Base:         base.name,
			Condition: fmt.Sprintf(
				"%s < %d && %s.len == %d",
				indexName,
				upper,
				base.name,
				upper,
			),
			Operation:      "index_store",
			Source:         sourceString(base.pos),
			RangeText:      rangeTextFromLattice(latticeRange),
			Lower:          plirBoundFromRangeBound(latticeRange.Lower),
			Upper:          plirBoundFromRangeBound(latticeRange.Upper),
			InclusiveLower: latticeRange.InclusiveLower,
			InclusiveUpper: latticeRange.InclusiveUpper,
			Reason:         "const loop allocation length proof",
			Derivation:     derivation,
		})
	}
	return proofs
}

func (b *builder) constLoopCondition(cond frontend.Expr) (string, int64, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenLess {
		return "", 0, false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil || left.Name == "" {
		return "", 0, false
	}
	upper, ok := b.proofConstIntValue(bin.Right)
	if !ok {
		return "", 0, false
	}
	return left.Name, upper, true
}

type constLoopDirectIndexStoreBase struct {
	name string
	pos  frontend.Position
}

func (b *builder) constLoopDirectIndexStoreBases(
	stmts []frontend.Stmt,
	indexName string,
) []constLoopDirectIndexStoreBase {
	seen := map[string]frontend.Position{}
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		index, ok := assign.Target.(*frontend.IndexExpr)
		if !ok || index == nil {
			continue
		}
		if exprPath(index.Index) != indexName {
			continue
		}
		base := exprPath(index.Base)
		if base != "" {
			if _, ok := seen[base]; !ok {
				seen[base] = index.Pos()
			}
		}
	}
	out := make([]constLoopDirectIndexStoreBase, 0, len(seen))
	for base, pos := range seen {
		out = append(out, constLoopDirectIndexStoreBase{name: base, pos: pos})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func (b *builder) nonNegativeWhileRangeProof(s *frontend.WhileStmt) (rangeProof, bool) {
	bin, ok := s.Cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenLess {
		return rangeProof{}, false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil || left.Name == "" {
		return rangeProof{}, false
	}
	if !b.zeroLocals[left.Name] {
		return rangeProof{}, false
	}
	upper, ok := b.proofConstIntValue(bin.Right)
	if !ok || upper <= 0 {
		return rangeProof{}, false
	}
	if b.bodyHasExactlyOneUnitIncrement(s.Body, left.Name) {
		if proofs := b.affineConstExtentProofs(s, left.Name, upper); len(proofs) == 1 {
			return proofs[0], true
		}
		return rangeProof{
			ID:        proofIDForRange("while-nonnegative", left.Name, "nonnegative", s.At),
			IndexName: left.Name,
			Base:      encodeNonNegativeConstLoopProofBase(upper),
			Condition: exprPath(s.Cond),
			Source:    sourceString(s.At),
			Reason:    "while loop nonnegative induction proof",
		}, true
	}
	if !b.bodyOnlyMutatesIndexByUnitIncrement(s.Body, left.Name) {
		return rangeProof{}, false
	}
	return rangeProof{
		ID:        proofIDForRange("while-nonnegative", left.Name, "nonnegative", s.At),
		IndexName: left.Name,
		Base:      nonNegativeWhileProofBase,
		Condition: exprPath(s.Cond),
		Source:    sourceString(s.At),
		Reason:    "while loop nonnegative induction proof",
	}, true
}

func (b *builder) affineConstExtentProofsForWhile(s *frontend.WhileStmt) []rangeProof {
	bin, ok := s.Cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenLess {
		return nil
	}
	right, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || right == nil || right.Name == "" {
		return nil
	}
	if !b.zeroLocals[right.Name] {
		return nil
	}
	upper, ok := b.proofConstIntValue(bin.Right)
	if !ok || upper <= 0 {
		return nil
	}
	if !b.bodyHasExactlyOneUnitIncrement(s.Body, right.Name) {
		return nil
	}
	return b.affineConstExtentProofs(s, right.Name, upper)
}

func (b *builder) affineConstExtentProofs(
	s *frontend.WhileStmt,
	rightName string,
	rightUpper int64,
) []rangeProof {
	if rightUpper != 3 {
		return nil
	}
	var proofs []rangeProof
	switch rightName {
	case "col":
		if proof, ok := b.affineConstExtentStoreProof(s.Body, rightName, rightUpper); ok {
			proofs = append(proofs, proof)
		}
		if proof, ok := b.affineConstExtentBLoadProof(s.Body, rightName, rightUpper); ok {
			proofs = append(proofs, proof)
		}
	case "k":
		if proof, ok := b.affineConstExtentLoadProof(s.Body, rightName, rightUpper); ok {
			proofs = append(proofs, proof)
		}
	default:
		return nil
	}
	return proofs
}

func (b *builder) affineConstExtentStoreProof(
	stmts []frontend.Stmt,
	rightName string,
	rightUpper int64,
) (rangeProof, bool) {
	var out rangeProof
	found := false
	for idx, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		index, ok := assign.Target.(*frontend.IndexExpr)
		if !ok || index == nil {
			continue
		}
		base := exprPath(index.Base)
		if base != "c" {
			continue
		}
		leftName, stride, matchedRight, ok := affineConstExtentIndexParts(index.Index)
		if !ok || leftName != "row" || matchedRight != rightName || stride != 3 || rightUpper != 3 {
			continue
		}
		leftUpper, ok := b.activeExactConstLoopUpperForLocal(leftName)
		if !ok || leftUpper != 3 {
			return rangeProof{}, false
		}
		length, ok := b.allocationConstLengthForBase(base)
		if !ok || length != 9 || leftUpper*stride != length || rightUpper != stride {
			return rangeProof{}, false
		}
		if affineConstExtentPrefixInvalidated(stmts[:idx], base, leftName, rightName) {
			return rangeProof{}, false
		}
		if found {
			return rangeProof{}, false
		}
		indexName := exprPath(index.Index)
		latticeRange := rangeproof.LessThanLen(indexName, base)
		derivation := append([]string(nil), latticeRange.Derivation...)
		derivation = append(derivation, "affine_const_extent")
		out = rangeProof{
			ID: proofIDForRange(
				"affine-const",
				leftName+"_"+rightName,
				base,
				index.Pos(),
			),
			IndexName:       rightName,
			IndexValueID:    b.ensureProofIndexValue(indexName, index.Pos()),
			Base:            base,
			AffineLeftName:  leftName,
			AffineRightName: rightName,
			AffineStride:    stride,
			Condition: fmt.Sprintf(
				"%s < %d && %s < %d && %s.len == %d && %s in [0, %s.len)",
				leftName,
				leftUpper,
				rightName,
				rightUpper,
				base,
				length,
				indexName,
				base,
			),
			Operation:      "index_store",
			Source:         sourceString(index.Pos()),
			RangeText:      rangeTextFromLattice(latticeRange),
			Lower:          plirBoundFromRangeBound(latticeRange.Lower),
			Upper:          plirBoundFromRangeBound(latticeRange.Upper),
			InclusiveLower: latticeRange.InclusiveLower,
			InclusiveUpper: latticeRange.InclusiveUpper,
			Reason:         "affine const extent proof",
			Derivation:     derivation,
		}
		found = true
	}
	return out, found
}

func (b *builder) affineConstExtentLoadProof(
	stmts []frontend.Stmt,
	rightName string,
	rightUpper int64,
) (rangeProof, bool) {
	var out rangeProof
	found := false
	for idx, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		loads := affineConstExtentLoadIndexes(assign.Value)
		for _, index := range loads {
			base := exprPath(index.Base)
			if base != "a" {
				continue
			}
			leftName, stride, matchedRight, ok := affineConstExtentIndexParts(index.Index)
			if !ok || leftName != "row" || matchedRight != rightName || stride != 3 ||
				rightUpper != 3 {
				continue
			}
			leftUpper, ok := b.activeExactConstLoopUpperForLocal(leftName)
			if !ok || leftUpper != 3 {
				return rangeProof{}, false
			}
			length, ok := b.allocationConstLengthForBase(base)
			if !ok || length != 9 || leftUpper*stride != length || rightUpper != stride {
				return rangeProof{}, false
			}
			if affineConstExtentPrefixInvalidated(stmts[:idx], base, leftName, rightName) {
				return rangeProof{}, false
			}
			if found {
				return rangeProof{}, false
			}
			indexName := exprPath(index.Index)
			latticeRange := rangeproof.LessThanLen(indexName, base)
			derivation := append([]string(nil), latticeRange.Derivation...)
			derivation = append(derivation, "affine_const_extent")
			out = rangeProof{
				ID: proofIDForRange(
					"affine-const",
					leftName+"_"+rightName,
					base,
					index.Pos(),
				),
				IndexName:       rightName,
				IndexValueID:    b.ensureProofIndexValue(indexName, index.Pos()),
				Base:            base,
				AffineLeftName:  leftName,
				AffineRightName: rightName,
				AffineStride:    stride,
				Condition: fmt.Sprintf(
					"%s < %d && %s < %d && %s.len == %d && %s in [0, %s.len)",
					leftName,
					leftUpper,
					rightName,
					rightUpper,
					base,
					length,
					indexName,
					base,
				),
				Operation:      "index_load",
				Source:         sourceString(index.Pos()),
				RangeText:      rangeTextFromLattice(latticeRange),
				Lower:          plirBoundFromRangeBound(latticeRange.Lower),
				Upper:          plirBoundFromRangeBound(latticeRange.Upper),
				InclusiveLower: latticeRange.InclusiveLower,
				InclusiveUpper: latticeRange.InclusiveUpper,
				Reason:         "affine const extent proof",
				Derivation:     derivation,
			}
			found = true
		}
	}
	return out, found
}

func (b *builder) affineConstExtentBLoadProof(
	stmts []frontend.Stmt,
	rightName string,
	rightUpper int64,
) (rangeProof, bool) {
	if rightName != "col" || rightUpper != 3 {
		return rangeProof{}, false
	}
	var out rangeProof
	found := false
	for outerIdx, stmt := range stmts {
		nested, ok := stmt.(*frontend.WhileStmt)
		if !ok || nested == nil {
			continue
		}
		leftName, leftUpper, ok := b.constLoopCondition(nested.Cond)
		if !ok || leftName != "k" || leftUpper != 3 {
			continue
		}
		if !affineConstExtentPrefixDeclaresZeroLocal(stmts[:outerIdx], leftName) {
			return rangeProof{}, false
		}
		if !b.bodyHasExactlyOneUnitIncrement(nested.Body, leftName) {
			return rangeProof{}, false
		}
		for innerIdx, nestedStmt := range nested.Body {
			assign, ok := nestedStmt.(*frontend.AssignStmt)
			if !ok || assign == nil {
				continue
			}
			for _, index := range affineConstExtentLoadIndexes(assign.Value) {
				base := exprPath(index.Base)
				if base != "b" {
					continue
				}
				matchedLeft, stride, matchedRight, ok := affineConstExtentIndexParts(index.Index)
				if !ok || matchedLeft != leftName || matchedRight != rightName || stride != 3 {
					continue
				}
				length, ok := b.allocationConstLengthForBase(base)
				if !ok || length != 9 || leftUpper*stride != length || rightUpper != stride {
					return rangeProof{}, false
				}
				if affineConstExtentPrefixInvalidated(
					stmts[:outerIdx],
					base,
					leftName,
					rightName,
				) ||
					affineConstExtentPrefixInvalidated(
						nested.Body[:innerIdx],
						base,
						leftName,
						rightName,
					) {
					return rangeProof{}, false
				}
				if found {
					return rangeProof{}, false
				}
				indexName := exprPath(index.Index)
				latticeRange := rangeproof.LessThanLen(indexName, base)
				derivation := append([]string(nil), latticeRange.Derivation...)
				derivation = append(derivation, "affine_const_extent")
				out = rangeProof{
					ID: proofIDForRange(
						"affine-const",
						leftName+"_"+rightName,
						base,
						index.Pos(),
					),
					IndexName:       rightName,
					IndexValueID:    b.ensureProofIndexValue(indexName, index.Pos()),
					Base:            base,
					AffineLeftName:  leftName,
					AffineRightName: rightName,
					AffineStride:    stride,
					Condition: fmt.Sprintf(
						"%s < %d && %s < %d && %s.len == %d && %s in [0, %s.len)",
						leftName,
						leftUpper,
						rightName,
						rightUpper,
						base,
						length,
						indexName,
						base,
					),
					Operation:      "index_load",
					Source:         sourceString(index.Pos()),
					RangeText:      rangeTextFromLattice(latticeRange),
					Lower:          plirBoundFromRangeBound(latticeRange.Lower),
					Upper:          plirBoundFromRangeBound(latticeRange.Upper),
					InclusiveLower: latticeRange.InclusiveLower,
					InclusiveUpper: latticeRange.InclusiveUpper,
					Reason:         "affine const extent proof",
					Derivation:     derivation,
				}
				found = true
			}
		}
	}
	return out, found
}

func (b *builder) activeExactConstLoopUpperForLocal(name string) (int64, bool) {
	for i := len(b.activeProof) - 1; i >= 0; i-- {
		proof := b.activeProof[i]
		if proof.ID == "" || proof.IndexName != name {
			continue
		}
		if upper, ok := decodeNonNegativeConstLoopProofBase(proof.Base); ok {
			return upper, true
		}
	}
	return 0, false
}

func affineConstExtentIndexParts(expr frontend.Expr) (string, int64, string, bool) {
	add, ok := expr.(*frontend.BinaryExpr)
	if !ok || add == nil || add.Op != frontend.TokenPlus {
		return "", 0, "", false
	}
	mul, ok := add.Left.(*frontend.BinaryExpr)
	if !ok || mul == nil || mul.Op != frontend.TokenStar {
		return "", 0, "", false
	}
	left, ok := mul.Left.(*frontend.IdentExpr)
	if !ok || left == nil || left.Name == "" {
		return "", 0, "", false
	}
	stride, ok := mul.Right.(*frontend.NumberExpr)
	if !ok || stride == nil || stride.Value <= 0 {
		return "", 0, "", false
	}
	right, ok := add.Right.(*frontend.IdentExpr)
	if !ok || right == nil || right.Name == "" {
		return "", 0, "", false
	}
	return left.Name, int64(stride.Value), right.Name, true
}

func affineConstExtentLoadIndexes(expr frontend.Expr) []*frontend.IndexExpr {
	var out []*frontend.IndexExpr
	var walk func(frontend.Expr)
	walk = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.IndexExpr:
			if e != nil {
				out = append(out, e)
			}
		case *frontend.BinaryExpr:
			if e != nil {
				walk(e.Left)
				walk(e.Right)
			}
		case *frontend.UnaryExpr:
			if e != nil {
				walk(e.X)
			}
		case *frontend.CallExpr:
			if e != nil {
				for _, arg := range e.Args {
					walk(arg)
				}
			}
		case *frontend.FieldAccessExpr:
			if e != nil {
				walk(e.Base)
			}
		}
	}
	walk(expr)
	return out
}

func affineConstExtentPrefixInvalidated(stmts []frontend.Stmt, names ...string) bool {
	seen := map[string]bool{}
	for _, name := range names {
		if name != "" {
			seen[name] = true
		}
	}
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if ok && target != nil && seen[target.Name] {
			return true
		}
	}
	return false
}

func affineConstExtentPrefixDeclaresZeroLocal(stmts []frontend.Stmt, name string) bool {
	for _, stmt := range stmts {
		let, ok := stmt.(*frontend.LetStmt)
		if !ok || let == nil || let.Name != name {
			continue
		}
		return isZeroNumber(let.Value)
	}
	return false
}

func (b *builder) ifRangeProof(s *frontend.IfStmt) (rangeProof, bool) {
	proof, ok := b.branchRangeProofFromCondition(s.Cond, s.At)
	if !ok {
		proof, ok = b.rangeProofFromCondition(s.Cond, s.At)
		if !ok || !b.zeroLocals[proof.IndexName] {
			return rangeProof{}, false
		}
	}
	if b.externalLocals[proof.Base] || b.invalidLocals[proof.Base] {
		return rangeProof{}, false
	}
	proof.ID = proofIDForRange("if", proof.IndexName, proof.Base, s.At)
	proof.Reason = "if branch range proof"
	return proof, true
}

func (b *builder) rangeProofFromCondition(
	cond frontend.Expr,
	pos frontend.Position,
) (rangeProof, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return rangeProof{}, false
	}
	index, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || index == nil {
		return rangeProof{}, false
	}
	base, latticeRange, ok := b.rangeFromCondition(index.Name, bin.Op, bin.Right)
	if !ok || base == "" || !latticeRange.Known {
		return rangeProof{}, false
	}
	indexID := b.valueIDForName(index.Name)
	return rangeProof{
		IndexName:      index.Name,
		IndexValueID:   indexID,
		Base:           base,
		Condition:      exprPath(cond),
		Source:         sourceString(pos),
		RangeText:      rangeTextFromLattice(latticeRange),
		Lower:          plirBoundFromRangeBound(latticeRange.Lower),
		Upper:          plirBoundFromRangeBound(latticeRange.Upper),
		InclusiveLower: latticeRange.InclusiveLower,
		InclusiveUpper: latticeRange.InclusiveUpper,
		Reason:         "range guard proof",
		Derivation:     append([]string(nil), latticeRange.Derivation...),
	}, true
}

func (b *builder) branchRangeProofFromCondition(
	cond frontend.Expr,
	pos frontend.Position,
) (rangeProof, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenAmpAmp {
		return rangeProof{}, false
	}
	if proof, ok := b.branchRangeProofParts(bin.Left, bin.Right, pos); ok {
		return proof, true
	}
	return b.branchRangeProofParts(bin.Right, bin.Left, pos)
}

func (b *builder) branchRangeProofParts(
	lower frontend.Expr,
	upper frontend.Expr,
	pos frontend.Position,
) (rangeProof, bool) {
	lowerIndex, ok := nonNegativeGuardIndex(lower)
	if !ok {
		return rangeProof{}, false
	}
	proof, ok := b.rangeProofFromCondition(upper, pos)
	if !ok || proof.IndexName != lowerIndex {
		return rangeProof{}, false
	}
	proof.Condition = exprPath(lower) + " && " + exprPath(upper)
	return proof, true
}

func nonNegativeGuardIndex(expr frontend.Expr) (string, bool) {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return "", false
	}
	if left, ok := bin.Left.(*frontend.IdentExpr); ok && left != nil &&
		bin.Op == frontend.TokenGreaterEq &&
		isZeroNumber(bin.Right) {
		return left.Name, true
	}
	if right, ok := bin.Right.(*frontend.IdentExpr); ok && right != nil &&
		bin.Op == frontend.TokenLessEq &&
		isZeroNumber(bin.Left) {
		return right.Name, true
	}
	return "", false
}

func isZeroNumber(expr frontend.Expr) bool {
	num, ok := expr.(*frontend.NumberExpr)
	return ok && num != nil && num.Value == 0
}

func (b *builder) valueIDForName(name string) string {
	for _, kind := range []ValueKind{
		ValueLocal,
		ValueLoopIndex,
		ValueParam,
		ValueView,
		ValueAllocIntent,
	} {
		id := valueID(kind, name)
		if _, ok := b.values[id]; ok {
			return id
		}
	}
	return valueID(ValueLocal, name)
}

func (b *builder) ensureProofIndexValue(name string, pos frontend.Position) string {
	id := valueID(ValueLocal, name)
	if _, ok := b.values[id]; ok {
		return id
	}
	b.addValue(Value{
		ID:         id,
		Kind:       ValueLocal,
		Type:       "i32",
		Source:     sourceString(pos),
		Region:     "fn:" + b.fn.Name,
		Provenance: Provenance{Kind: ProvenanceStack, Root: name},
		Lifetime:   Lifetime{Birth: sourceString(pos), Owner: name},
		Escape:     EscapeNoEscape,
	})
	return id
}

func (b *builder) rangeUpperFromCondition(
	op frontend.TokenType,
	right frontend.Expr,
) (string, Bound, bool, bool) {
	switch op {
	case frontend.TokenLess, frontend.TokenBangEq:
		base := b.lenBoundBase(right)
		if base == "" {
			return "", Bound{}, false, false
		}
		return base, Bound{Kind: BoundSymbol, Symbol: base + ".len"}, false, true
	case frontend.TokenLessEq:
		base := lenMinusOneBase(right)
		if base == "" {
			return "", Bound{}, false, false
		}
		return base, Bound{Kind: BoundSymbolMinus, Symbol: base + ".len", Const: 1}, true, true
	default:
		return "", Bound{}, false, false
	}
}

func (b *builder) rangeFromCondition(
	indexName string,
	op frontend.TokenType,
	right frontend.Expr,
) (string, rangeproof.Range, bool) {
	switch op {
	case frontend.TokenLess, frontend.TokenBangEq:
		base := b.lenBoundBase(right)
		if base == "" {
			return "", rangeproof.Range{}, false
		}
		return base, rangeproof.LessThanLen(indexName, base), true
	case frontend.TokenLessEq:
		base := lenMinusOneBase(right)
		if base == "" {
			return "", rangeproof.Range{}, false
		}
		return base, rangeproof.LessEqualLenMinusOne(indexName, base), true
	default:
		return "", rangeproof.Range{}, false
	}
}

func (b *builder) lenBoundBase(expr frontend.Expr) string {
	if base := lenFieldBase(expr); base != "" {
		return base
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return ""
	}
	return b.lenBoundLocals[id.Name]
}

func rangeUpperFromCondition(
	op frontend.TokenType,
	right frontend.Expr,
) (string, Bound, bool, bool) {
	switch op {
	case frontend.TokenLess, frontend.TokenBangEq:
		base := lenFieldBase(right)
		if base == "" {
			return "", Bound{}, false, false
		}
		return base, Bound{Kind: BoundSymbol, Symbol: base + ".len"}, false, true
	case frontend.TokenLessEq:
		base := lenMinusOneBase(right)
		if base == "" {
			return "", Bound{}, false, false
		}
		return base, Bound{Kind: BoundSymbolMinus, Symbol: base + ".len", Const: 1}, true, true
	default:
		return "", Bound{}, false, false
	}
}

func lenFieldBase(expr frontend.Expr) string {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok || field == nil || field.Field != "len" {
		return ""
	}
	return exprPath(field.Base)
}

func lenMinusOneBase(expr frontend.Expr) string {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenMinus {
		return ""
	}
	num, ok := bin.Right.(*frontend.NumberExpr)
	if !ok || num == nil || num.Value != 1 {
		return ""
	}
	return lenFieldBase(bin.Left)
}

func (b *builder) rememberLocalProofMetadata(name string, expr frontend.Expr) {
	b.forgetLenBoundsForBase(name)
	b.forgetAllocationConstLengthForBase(name)
	b.forgetModuloRangeProofForLocal(name)
	if value, ok := b.proofConstIntValue(expr); ok {
		b.zeroLocals[name] = value == 0
		b.constIntLocals[name] = value
	} else {
		b.zeroLocals[name] = isZeroExpr(expr)
		delete(b.constIntLocals, name)
	}
	if base := lenFieldBase(expr); base != "" {
		b.lenBoundLocals[name] = base
	} else {
		delete(b.lenBoundLocals, name)
	}
	if lengthName, ok := b.allocationLengthBoundLocal(expr); ok {
		b.lenBoundLocals[lengthName] = name
	}
	if length, ok := b.allocationConstLength(expr); ok {
		b.constIntLocals[allocationConstLengthKey(name)] = length
	}
	b.rememberModuloRangeProofForLocal(name, expr)
}

func (b *builder) forgetLenBoundsForBase(baseName string) {
	if baseName == "" {
		return
	}
	b.forgetAllocationConstLengthForBase(baseName)
	for name, base := range b.lenBoundLocals {
		if base == baseName {
			delete(b.lenBoundLocals, name)
			continue
		}
		if proof, ok := decodeModuloRangeProof(base); ok && proof.baseName == baseName {
			delete(b.lenBoundLocals, name)
		}
	}
}

func (b *builder) rememberModuloRangeProofForLocal(name string, expr frontend.Expr) {
	delete(b.lenBoundLocals, moduloRangeProofKey(name))
	info, ok := b.fn.Locals[name]
	if !ok || info.Mutable || info.SlotCount != 1 {
		return
	}
	numerator, divisorName, ok := moduloDivisorName(expr)
	if !ok {
		return
	}
	divisorInfo, ok := b.fn.Locals[divisorName]
	if !ok || divisorInfo.Mutable {
		return
	}
	divisorValue, ok := b.proofConstIntValue(&frontend.IdentExpr{Name: divisorName})
	if !ok || divisorValue <= 0 {
		return
	}
	baseName := b.lenBoundLocals[divisorName]
	if baseName == "" || b.externalLocals[baseName] || b.invalidLocals[baseName] {
		return
	}
	if !b.exprKnownNonNegative(numerator) {
		return
	}
	latticeRange := rangeproof.LessThanLen(name, baseName)
	derivation := append([]string(nil), latticeRange.Derivation...)
	derivation = append(derivation, "modulo_allocation_length_alias")
	proof := rangeProof{
		ID:             proofIDForRange("modulo", name, baseName, expr.Pos()),
		IndexName:      name,
		IndexValueID:   b.valueIDForName(name),
		Base:           baseName,
		Condition:      name + " = " + exprPath(expr),
		Source:         sourceString(expr.Pos()),
		RangeText:      rangeTextFromLattice(latticeRange),
		Lower:          plirBoundFromRangeBound(latticeRange.Lower),
		Upper:          plirBoundFromRangeBound(latticeRange.Upper),
		InclusiveLower: latticeRange.InclusiveLower,
		InclusiveUpper: latticeRange.InclusiveUpper,
		Reason:         "modulo allocation length alias proof",
		Derivation:     derivation,
	}
	b.addRangeProof(proof, b.current, "")
	b.lenBoundLocals[moduloRangeProofKey(name)] = encodeModuloRangeProof(moduloRangeProof{
		baseName:     baseName,
		divisorName:  divisorName,
		proofID:      proof.ID,
		indexValueID: proof.IndexValueID,
		rangeText:    proof.RangeText,
	})
}

func (b *builder) moduloRangeProofForLocal(name string) (moduloRangeProof, bool) {
	encoded, ok := b.lenBoundLocals[moduloRangeProofKey(name)]
	if !ok {
		return moduloRangeProof{}, false
	}
	return decodeModuloRangeProof(encoded)
}

func (b *builder) forgetModuloRangeProofForLocal(name string) {
	if name == "" {
		return
	}
	delete(b.lenBoundLocals, moduloRangeProofKey(name))
	for key, encoded := range b.lenBoundLocals {
		proof, ok := decodeModuloRangeProof(encoded)
		if !ok {
			continue
		}
		if proofPathMatchesMutation(proof.baseName, name) ||
			proofPathMatchesMutation(proof.divisorName, name) {
			delete(b.lenBoundLocals, key)
		}
	}
}

func (b *builder) exprKnownNonNegative(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return e != nil && e.Value >= 0
	case *frontend.IdentExpr:
		if e == nil || e.Name == "" {
			return false
		}
		if b.activeNonNegativeWhileProofForLocal(e.Name) {
			return true
		}
		if value, ok := b.proofConstIntValue(e); ok {
			return value >= 0
		}
		return false
	case *frontend.BinaryExpr:
		if e == nil {
			return false
		}
		switch e.Op {
		case frontend.TokenPlus, frontend.TokenStar:
			return b.exprKnownNonNegative(e.Left) && b.exprKnownNonNegative(e.Right)
		default:
			return false
		}
	default:
		return false
	}
}

func (b *builder) activeNonNegativeWhileProofForLocal(name string) bool {
	for i := len(b.activeProof) - 1; i >= 0; i-- {
		proof := b.activeProof[i]
		if proof.ID != "" && isNonNegativeWhileProofBase(proof.Base) && proof.IndexName == name {
			return true
		}
		if proof.ID != "" && (proof.AffineLeftName == name || proof.AffineRightName == name) {
			return true
		}
	}
	return false
}

func moduloDivisorName(expr frontend.Expr) (frontend.Expr, string, bool) {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenPercent {
		return nil, "", false
	}
	divisor, ok := bin.Right.(*frontend.IdentExpr)
	if !ok || divisor == nil || divisor.Name == "" {
		return nil, "", false
	}
	return bin.Left, divisor.Name, true
}

func (b *builder) moduloConstDivisor(expr frontend.Expr) (frontend.Expr, int64, bool) {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenPercent {
		return nil, 0, false
	}
	switch divisor := bin.Right.(type) {
	case *frontend.NumberExpr:
		if divisor == nil {
			return nil, 0, false
		}
		return bin.Left, int64(divisor.Value), true
	case *frontend.IdentExpr:
		if divisor == nil || divisor.Name == "" {
			return nil, 0, false
		}
		value, ok := b.proofConstIntValue(divisor)
		if !ok {
			return nil, 0, false
		}
		return bin.Left, value, true
	default:
		return nil, 0, false
	}
}

func moduloRangeProofKey(name string) string {
	return moduloRangeProofKeyPrefix + name
}

func encodeModuloRangeProof(proof moduloRangeProof) string {
	return strings.Join(
		[]string{
			proof.baseName,
			proof.divisorName,
			proof.proofID,
			proof.indexValueID,
			proof.rangeText,
		},
		moduloRangeProofFieldSep,
	)
}

func decodeModuloRangeProof(encoded string) (moduloRangeProof, bool) {
	parts := strings.Split(encoded, moduloRangeProofFieldSep)
	if len(parts) != 5 {
		return moduloRangeProof{}, false
	}
	proof := moduloRangeProof{
		baseName:     parts[0],
		divisorName:  parts[1],
		proofID:      parts[2],
		indexValueID: parts[3],
		rangeText:    parts[4],
	}
	if proof.baseName == "" || proof.divisorName == "" || proof.proofID == "" ||
		proof.indexValueID == "" ||
		proof.rangeText == "" {
		return moduloRangeProof{}, false
	}
	return proof, true
}

func (b *builder) allocationLengthBoundLocal(expr frontend.Expr) (string, bool) {
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
	info, ok := b.fn.Locals[id.Name]
	if !ok || info.Mutable {
		return "", false
	}
	return id.Name, true
}

func (b *builder) allocationConstLength(expr frontend.Expr) (int64, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return 0, false
	}
	lengthExpr, ok := allocationConstLengthExpr(call)
	if !ok {
		return 0, false
	}
	length, ok := b.proofConstIntValue(lengthExpr)
	if !ok || length < 0 {
		return 0, false
	}
	return length, true
}

func (b *builder) allocationConstLengthForBase(baseName string) (int64, bool) {
	if baseName == "" {
		return 0, false
	}
	length, ok := b.constIntLocals[allocationConstLengthKey(baseName)]
	return length, ok
}

func (b *builder) forgetAllocationConstLengthForBase(baseName string) {
	if baseName == "" {
		return
	}
	delete(b.constIntLocals, allocationConstLengthKey(baseName))
}

func allocationConstLengthKey(baseName string) string {
	return allocationConstLengthKeyPrefix + baseName
}

func encodeNonNegativeConstLoopProofBase(upper int64) string {
	return nonNegativeConstLoopProofBase + moduloRangeProofFieldSep + strconv.FormatInt(upper, 10)
}

func decodeNonNegativeConstLoopProofBase(encoded string) (int64, bool) {
	prefix := nonNegativeConstLoopProofBase + moduloRangeProofFieldSep
	if !strings.HasPrefix(encoded, prefix) {
		return 0, false
	}
	upper, err := strconv.ParseInt(strings.TrimPrefix(encoded, prefix), 10, 64)
	if err != nil || upper <= 0 {
		return 0, false
	}
	return upper, true
}

func isNonNegativeWhileProofBase(encoded string) bool {
	if encoded == nonNegativeWhileProofBase {
		return true
	}
	_, ok := decodeNonNegativeConstLoopProofBase(encoded)
	return ok
}

func proofUseAllowedForIndexStore(proofID string) bool {
	return strings.HasPrefix(proofID, "proof:while-const:") ||
		strings.HasPrefix(proofID, "proof:affine-const:") ||
		strings.HasPrefix(proofID, "proof:allocation-zero:") ||
		strings.HasPrefix(proofID, "proof:helper-offset:") ||
		strings.HasPrefix(proofID, "proof:helper-summary:")
}

func isMakeSliceCallName(name string) bool {
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	_, ok := makeSliceElem(name)
	return ok
}

func allocationConstLengthExpr(call *frontend.CallExpr) (frontend.Expr, bool) {
	if call == nil {
		return nil, false
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	switch name {
	case "core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool":
		if len(call.Args) != 1 {
			return nil, false
		}
		return call.Args[0], true
	case "core.island_make_u8",
		"core.island_make_u16",
		"core.island_make_i32",
		"core.island_make_bool":
		if len(call.Args) < 2 {
			return nil, false
		}
		return call.Args[1], true
	default:
		return nil, false
	}
}

func (b *builder) bodyHasUnitIncrement(stmts []frontend.Stmt, indexName string) bool {
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if !ok || target.Name != indexName {
			continue
		}
		if b.isUnitIncrement(assign.Value, indexName) {
			return true
		}
	}
	return false
}

func (b *builder) bodyOnlyMutatesIndexByUnitIncrement(
	stmts []frontend.Stmt,
	indexName string,
) bool {
	found := false
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if !ok || target.Name != indexName {
			continue
		}
		if !b.isUnitIncrement(assign.Value, indexName) {
			return false
		}
		found = true
	}
	return found
}

func (b *builder) bodyHasExactlyOneUnitIncrement(stmts []frontend.Stmt, indexName string) bool {
	found := false
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if !ok || target.Name != indexName {
			continue
		}
		if found || !b.isUnitIncrement(assign.Value, indexName) {
			return false
		}
		found = true
	}
	return found
}

func (b *builder) isUnitIncrement(expr frontend.Expr, indexName string) bool {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenPlus {
		return false
	}
	if left, ok := bin.Left.(*frontend.IdentExpr); ok && left.Name == indexName {
		return b.isUnitStepExpr(bin.Right)
	}
	if right, ok := bin.Right.(*frontend.IdentExpr); ok && right.Name == indexName {
		return b.isUnitStepExpr(bin.Left)
	}
	return false
}

func (b *builder) isUnitStepExpr(expr frontend.Expr) bool {
	if num, ok := expr.(*frontend.NumberExpr); ok && num != nil {
		return num.Value == 1
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return false
	}
	info, ok := b.fn.Locals[id.Name]
	if !ok || info.Mutable {
		return false
	}
	value, ok := b.constIntLocals[id.Name]
	return ok && value == 1
}

func (b *builder) proofConstIntValue(expr frontend.Expr) (int64, bool) {
	return b.constIntValue(expr)
}

func (b *builder) constIntValue(expr frontend.Expr) (int64, bool) {
	if value, ok := evalConstInt64(expr); ok {
		return value, true
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return 0, false
	}
	info, ok := b.fn.Locals[id.Name]
	if !ok || info.Mutable {
		return 0, false
	}
	value, ok := b.constIntLocals[id.Name]
	return value, ok
}

func isZeroExpr(expr frontend.Expr) bool {
	num, ok := expr.(*frontend.NumberExpr)
	return ok && num != nil && num.Value == 0
}

func rangeText(indexName string, base string, upper Bound, inclusiveUpper bool) string {
	if upper.Kind == BoundSymbolMinus {
		return fmt.Sprintf("%s in [0, %s - %d]", indexName, upper.Symbol, upper.Const)
	}
	if inclusiveUpper {
		return fmt.Sprintf("%s in [0, %s]", indexName, base+".len")
	}
	return fmt.Sprintf("%s in [0, %s)", indexName, base+".len")
}

func rangeTextFromLattice(r rangeproof.Range) string {
	upper := plirBoundFromRangeBound(r.Upper)
	base := ""
	switch r.Upper.Kind {
	case rangeproof.BoundSymbol, rangeproof.BoundSymbolMinus, rangeproof.BoundSymbolPlus:
		base = strings.TrimSuffix(r.Upper.Symbol, ".len")
	}
	return rangeText(r.Value, base, upper, r.InclusiveUpper)
}

func plirBoundFromRangeBound(bound rangeproof.Bound) Bound {
	switch bound.Kind {
	case rangeproof.BoundConst:
		return Bound{Kind: BoundConst, Const: bound.Const}
	case rangeproof.BoundSymbol:
		return Bound{Kind: BoundSymbol, Symbol: bound.Symbol}
	case rangeproof.BoundSymbolMinus:
		return Bound{Kind: BoundSymbolMinus, Symbol: bound.Symbol, Const: bound.Const}
	default:
		return Bound{Kind: BoundUnknown}
	}
}

func proofIDForRange(kind string, indexName string, base string, pos frontend.Position) string {
	base = proofNamePart(base, "value")
	return fmt.Sprintf("proof:%s:%s:%s:%d:%d", kind, indexName, base, pos.Line, pos.Col)
}

func copyLoopProofID(name string, pos frontend.Position) string {
	return fmt.Sprintf("proof:copy-loop:%s:%d:%d", proofNamePart(name, "copy"), pos.Line, pos.Col)
}

func proofNamePart(name string, fallback string) string {
	name = strings.NewReplacer(".", "_", " ", "_").Replace(name)
	if name == "" {
		return fallback
	}
	return name
}

func forCollectionProofID(stmt *frontend.ForRangeStmt) string {
	kind := "for-collection"
	if isViewIterable(stmt.Iterable) {
		kind = "for-collection-view"
	}
	return fmt.Sprintf("proof:%s:%s:%d:%d", kind, stmt.Name, stmt.At.Line, stmt.At.Col)
}

func isViewIterable(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = builtin
	}
	return strings.HasPrefix(name, "core.slice_window_") ||
		strings.HasPrefix(name, "core.slice_prefix_") ||
		strings.HasPrefix(name, "core.slice_suffix_") ||
		name == "core.string_window" ||
		name == "core.string_prefix" ||
		name == "core.string_suffix"
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
