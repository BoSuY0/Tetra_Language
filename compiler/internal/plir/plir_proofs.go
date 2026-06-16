package plir

import (
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/rangeproof"
	"tetra_language/compiler/internal/semantics"
)

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
	b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "for collection iterable view preserves source provenance"})
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
	b.addFact(Fact{Kind: FactBorrowedImm, ValueID: id})
	b.addFact(Fact{Kind: FactNoEscape, ValueID: id, Reason: "for collection iterable view may not escape its owner"})
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
		if proofPathMatchesMutation(b.activeProof[i].IndexName, name) || proofPathMatchesMutation(b.activeProof[i].Base, name) {
			b.activeProof[i].ID = ""
		}
	}
}

func (b *builder) invalidateActiveProofsForMutableCallArgs(args []frontend.Expr, ownership []string) {
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

func (b *builder) invalidateNoAliasForMutableCallArgs(args []frontend.Expr, ownership []string, reason string) {
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
	return proofPath == mutatedPath || strings.HasPrefix(proofPath, mutatedPath+".")
}

func (b *builder) activeProofForIndex(index *frontend.IndexExpr) (rangeProof, bool) {
	base := exprPath(index.Base)
	idx := exprPath(index.Index)
	if base == "" || idx == "" {
		return rangeProof{}, false
	}
	for i := len(b.activeProof) - 1; i >= 0; i-- {
		proof := b.activeProof[i]
		if proof.ID != "" && proof.Base == base && proof.IndexName == idx {
			return proof, true
		}
	}
	return rangeProof{}, false
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
	term := ProofTerm{
		ID:            proof.ID,
		Kind:          "bounds_check",
		SubjectBaseID: proof.Base,
		IndexValueID:  proof.IndexValueID,
		Operation:     "index_load",
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
			return islandTokenState{IslandID: fact.IslandID, Epoch: fact.Epoch, BaseID: fact.BaseID}, true
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
		b.reclassifyMemoryBinding(name, Provenance{Kind: ProvenanceUnknown}, "alias source is invalid before construction")
		return
	}
	if b.externalLocals[name] {
		b.reclassifyMemoryBinding(name, b.conservativeProvenanceFromExpr(expr), "alias source has external or unknown provenance")
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
	if sourceView.Provenance.Kind == ProvenanceExternal || sourceView.Provenance.Kind == ProvenanceUnknown {
		b.externalLocals[name] = true
		if !b.hasFactForValue(FactProvenanceUnknown, aliasID) {
			b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: aliasID, Reason: "alias source has external or unknown provenance"})
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

func (b *builder) rangeProofFromCondition(cond frontend.Expr, pos frontend.Position) (rangeProof, bool) {
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

func (b *builder) branchRangeProofFromCondition(cond frontend.Expr, pos frontend.Position) (rangeProof, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenAmpAmp {
		return rangeProof{}, false
	}
	if proof, ok := b.branchRangeProofParts(bin.Left, bin.Right, pos); ok {
		return proof, true
	}
	return b.branchRangeProofParts(bin.Right, bin.Left, pos)
}

func (b *builder) branchRangeProofParts(lower frontend.Expr, upper frontend.Expr, pos frontend.Position) (rangeProof, bool) {
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
	if left, ok := bin.Left.(*frontend.IdentExpr); ok && left != nil && bin.Op == frontend.TokenGreaterEq && isZeroNumber(bin.Right) {
		return left.Name, true
	}
	if right, ok := bin.Right.(*frontend.IdentExpr); ok && right != nil && bin.Op == frontend.TokenLessEq && isZeroNumber(bin.Left) {
		return right.Name, true
	}
	return "", false
}

func isZeroNumber(expr frontend.Expr) bool {
	num, ok := expr.(*frontend.NumberExpr)
	return ok && num != nil && num.Value == 0
}

func (b *builder) valueIDForName(name string) string {
	for _, kind := range []ValueKind{ValueLocal, ValueLoopIndex, ValueParam, ValueView, ValueAllocIntent} {
		id := valueID(kind, name)
		if _, ok := b.values[id]; ok {
			return id
		}
	}
	return valueID(ValueLocal, name)
}

func (b *builder) rangeUpperFromCondition(op frontend.TokenType, right frontend.Expr) (string, Bound, bool, bool) {
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

func (b *builder) rangeFromCondition(indexName string, op frontend.TokenType, right frontend.Expr) (string, rangeproof.Range, bool) {
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

func rangeUpperFromCondition(op frontend.TokenType, right frontend.Expr) (string, Bound, bool, bool) {
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
}

func (b *builder) forgetLenBoundsForBase(baseName string) {
	if baseName == "" {
		return
	}
	for name, base := range b.lenBoundLocals {
		if base == baseName {
			delete(b.lenBoundLocals, name)
		}
	}
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
