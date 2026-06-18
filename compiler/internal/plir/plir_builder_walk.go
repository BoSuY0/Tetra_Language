package plir

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/rangeproof"
	"tetra_language/compiler/internal/semantics"
)

type rangeProof struct {
	ID              string
	IndexName       string
	IndexValueID    string
	Base            string
	AffineLeftName  string
	AffineRightName string
	AffineStride    int64
	Condition       string
	Operation       string
	Source          string
	RangeText       string
	Lower           Bound
	Upper           Bound
	InclusiveLower  bool
	InclusiveUpper  bool
	Reason          string
	Derivation      []string
}

func (b *builder) walkStmt(stmt frontend.Stmt) {
	switch s := stmt.(type) {
	case *frontend.LetStmt:
		b.walkExpr(s.Value, s.Name)
		if !exprStoresDirectlyIntoTarget(s.Value) {
			b.recordLocalAssignment(s.Name, s.Value, s.At)
		}
		b.rememberLocalProofMetadata(s.Name, s.Value)
		b.rememberAliasMetadata(s.Name, s.Value)
	case *frontend.AssignStmt:
		assignmentWalked := false
		if id, ok := s.Target.(*frontend.IdentExpr); ok {
			if b.isGlobalName(id.Name) {
				b.walkExpr(s.Value, "")
			} else {
				b.clearRawPointerMetadata(id.Name)
				b.walkExpr(s.Value, id.Name)
			}
			assignmentWalked = true
		}
		if !assignmentWalked {
			b.walkExpr(s.Value, "")
		}
		if idx, ok := s.Target.(*frontend.IndexExpr); ok {
			b.walkExpr(idx.Base, "")
			b.walkExpr(idx.Index, "")
			op := b.addOperation(Operation{
				Kind:   OpIndexStore,
				Source: sourceString(s.At),
				Inputs: []string{exprPath(idx.Base), exprPath(idx.Index)},
			})
			if proof, ok := b.activeProofForIndex(idx); ok && proofUseAllowedForIndexStore(proof.ID) {
				b.proofUses = append(b.proofUses, ProofUse{
					ProofID: proof.ID,
					Block:   op.Block,
					OpID:    op.ID,
					UseKind: "bounds_check",
					Source:  sourceString(s.At),
				})
			}
		}
		if id, ok := s.Target.(*frontend.IdentExpr); ok {
			if b.isGlobalName(id.Name) {
				b.recordGlobalStore(id.Name, s.Value, s.At)
			} else {
				b.recordLocalAssignment(id.Name, s.Value, s.At)
				b.rememberLocalProofMetadata(id.Name, s.Value)
				b.rememberAliasMetadata(id.Name, s.Value)
				b.invalidateActiveProofForLocal(id.Name)
			}
		}
	case *frontend.ReturnStmt:
		b.walkExpr(s.Value, "$return")
		b.addOperation(Operation{
			Kind:   OpReturn,
			Source: sourceString(s.At),
			Inputs: []string{exprPath(s.Value)},
		})
	case *frontend.ThrowStmt:
		b.walkExpr(s.Value, "")
	case *frontend.PrintStmt:
		b.walkExpr(s.Value, "")
		b.addOperation(Operation{
			Kind:   OpPrint,
			Source: sourceString(s.At),
			Inputs: []string{exprPath(s.Value)},
		})
	case *frontend.ExprStmt:
		b.walkExpr(s.Expr, "")
	case *frontend.IfStmt:
		b.walkIfStmt(s)
	case *frontend.IfLetStmt:
		b.walkExpr(s.Value, s.ValueLocal)
		b.walkBlock(s.Then)
		b.walkBlock(s.Else)
	case *frontend.WhileStmt:
		b.walkWhileStmt(s)
	case *frontend.ForRangeStmt:
		b.walkForRangeStmt(s)
	case *frontend.MatchStmt:
		b.walkExpr(s.Value, s.ScrutineeLocal)
		for _, c := range s.Cases {
			b.walkExpr(c.Guard, "")
			b.walkBlock(c.Body)
		}
	case *frontend.IslandStmt:
		b.walkExpr(s.Size, "")
		b.walkBlock(s.Body)
	case *frontend.UnsafeStmt:
		b.addOperation(
			Operation{
				Kind:   OpUnsafe,
				Source: sourceString(s.At),
				Note:   "unsafe block requires conservative provenance/escape assumptions",
			},
		)
		b.walkBlock(s.Body)
	case *frontend.DeferStmt:
		b.walkBlock(s.Body)
	}
}

func (b *builder) walkBlock(stmts []frontend.Stmt) {
	for _, stmt := range stmts {
		b.walkStmt(stmt)
	}
}

func (b *builder) walkIfStmt(s *frontend.IfStmt) {
	b.walkExpr(s.Cond, "")
	condOp := b.addOperation(
		Operation{
			Kind:   OpGuard,
			Source: sourceString(s.At),
			Inputs: []string{exprPath(s.Cond)},
			Note:   exprPath(s.Cond),
		},
	)
	condBlock := b.current
	thenBlock := b.newBlock("if_then", s.At, false)
	elseBlock := ""
	joinBlock := b.newBlock("if_join", s.At, false)
	b.addEdge(condBlock, thenBlock)
	if len(s.Else) > 0 {
		elseBlock = b.newBlock("if_else", s.At, false)
		b.addEdge(condBlock, elseBlock)
	} else {
		b.addEdge(condBlock, joinBlock)
	}
	var proof *rangeProof
	if candidate, ok := b.ifRangeProof(s); ok {
		b.addRangeProof(candidate, thenBlock, condOp.ID)
		proof = &candidate
	}
	b.current = thenBlock
	if proof != nil {
		b.pushActiveProof(*proof)
	}
	branchState := b.snapshotLocalProofState()
	b.walkBlock(s.Then)
	if proof != nil {
		b.popActiveProof()
	}
	thenEnd := b.current
	thenState := b.snapshotLocalProofState()
	b.addEdge(thenEnd, joinBlock)
	elseState := branchState
	if elseBlock != "" {
		b.restoreLocalProofState(branchState)
		b.current = elseBlock
		b.walkBlock(s.Else)
		elseEnd := b.current
		elseState = b.snapshotLocalProofState()
		b.addEdge(elseEnd, joinBlock)
	}
	b.current = joinBlock
	b.mergeLocalProofState(thenState, elseState)
}

func (b *builder) walkWhileStmt(s *frontend.WhileStmt) {
	preheader := b.current
	header := b.newBlock("while_header", s.At, false)
	body := b.newBlock("while_body", s.At, false)
	after := b.newBlock("while_after", s.At, false)
	b.addEdge(preheader, header)
	b.current = header
	b.walkExpr(s.Cond, "")
	condOp := b.addOperation(
		Operation{
			Kind:   OpGuard,
			Source: sourceString(s.At),
			Inputs: []string{exprPath(s.Cond)},
			Note:   exprPath(s.Cond),
		},
	)
	b.addEdge(header, body)
	b.addEdge(header, after)

	proofs := b.whileRangeProofs(s)
	for _, proof := range proofs {
		if !isNonNegativeWhileProofBase(proof.Base) {
			b.addRangeProof(proof, body, condOp.ID)
		}
	}
	b.current = body
	for _, proof := range proofs {
		b.pushActiveProof(proof)
	}
	b.walkBlock(s.Body)
	for range proofs {
		b.popActiveProof()
	}
	b.addEdge(b.current, header)
	b.current = after
	for _, proof := range proofs {
		b.zeroLocals[proof.IndexName] = false
	}
}

func (b *builder) walkForRangeStmt(s *frontend.ForRangeStmt) {
	if s.Iterable == nil {
		b.walkExpr(s.Start, "")
		b.walkExpr(s.End, s.EndLocal)
		b.walkBlock(s.Body)
		return
	}
	b.walkExpr(s.Iterable, s.IterableLocal)
	base := exprPath(s.Iterable)
	if base == "" {
		base = s.IterableLocal
	}
	iterID := b.ensureViewValue(s.IterableLocal, base, s.Iterable.Pos())
	indexID := b.addLoopIndex(s)
	preheader := b.current
	header := b.newBlock("for_header", s.At, false)
	body := b.newBlock("for_body", s.At, false)
	after := b.newBlock("for_after", s.At, false)
	b.addEdge(preheader, header)
	b.current = header
	proofID := forCollectionProofID(s)
	op := b.addOperation(Operation{
		Kind:    OpForSlice,
		Source:  sourceString(s.At),
		Inputs:  []string{iterID, indexID},
		Outputs: []string{valueID(ValueLocal, s.Name)},
		Note:    "range: 0.." + base + ".len",
	})
	b.addEdge(header, body)
	b.addEdge(header, after)
	if b.collectionIterableProofAllowed(s.Iterable) {
		latticeRange := rangeproof.LessThanLen(s.IndexLocal, base)
		b.addFact(Fact{
			Kind:    FactIndexInRange,
			ValueID: indexID,
			Range:   "0.." + base + ".len",
			ProofID: proofID,
			Source:  sourceString(s.At),
			Reason:  "for collection loop index is dominated by index < iterable.len guard",
			Uses:    []string{iterID},
		})
		b.addFact(
			Fact{
				Kind:    FactLenStable,
				ValueID: iterID,
				Reason:  "for collection iterable is copied into hidden slice header",
			},
		)
		b.proofGuards = append(b.proofGuards, ProofGuard{
			ID:        proofID,
			Kind:      "range",
			Block:     body,
			OpID:      op.ID,
			Condition: s.IndexLocal + " < " + base + ".len",
			Reason:    "for loop range proof",
		})
		b.proofUses = append(b.proofUses, ProofUse{
			ProofID: proofID,
			Block:   body,
			OpID:    op.ID,
			UseKind: "bounds_check",
			Source:  sourceString(s.At),
		})
		b.addBoundsProofTerm(rangeProof{
			ID:             proofID,
			IndexName:      s.IndexLocal,
			IndexValueID:   indexID,
			Base:           base,
			Condition:      s.IndexLocal + " < " + base + ".len",
			Source:         sourceString(s.At),
			RangeText:      "0.." + base + ".len",
			Lower:          plirBoundFromRangeBound(latticeRange.Lower),
			Upper:          plirBoundFromRangeBound(latticeRange.Upper),
			InclusiveLower: latticeRange.InclusiveLower,
			InclusiveUpper: latticeRange.InclusiveUpper,
			Reason:         "for collection loop index is dominated by index < iterable.len guard",
			Derivation:     append([]string(nil), latticeRange.Derivation...),
		})
		b.rangeFacts = append(b.rangeFacts, RangeFact{
			Value:          indexID,
			Lower:          plirBoundFromRangeBound(latticeRange.Lower),
			Upper:          plirBoundFromRangeBound(latticeRange.Upper),
			InclusiveLower: latticeRange.InclusiveLower,
			InclusiveUpper: latticeRange.InclusiveUpper,
			Source:         sourceString(s.At),
			ProofID:        proofID,
			Reason:         "for loop range proof",
			Derivation:     append([]string(nil), latticeRange.Derivation...),
		})
	}
	b.current = body
	b.walkBlock(s.Body)
	b.addEdge(b.current, header)
	b.current = after
}

func (b *builder) walkExpr(expr frontend.Expr, targetName string) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			b.walkExpr(arg, "")
		}
		name := e.Name
		if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = builtin
		}
		ownership := b.callParamOwnership(name)
		b.invalidateActiveProofsForMutableCallArgs(e.Args, ownership)
		note := b.callSummaryNote(name)
		if boundary := b.callAliasBoundaryKind(
			name,
		); boundary != "" && callHasInoutArgument(
			e.Args,
			ownership,
		) {
			b.invalidateNoAliasForMutableCallArgs(e.Args, ownership, boundary)
			note = appendOperationNote(note, "alias_boundary:"+boundary)
		}
		if b.callSummaryUnknown(name) {
			b.invalidateNoAliasForCallInputs(e.Args, "unknown_external_call")
			note = appendOperationNote(note, "alias_boundary:unknown_external_call")
		}
		if name != e.Name {
			if b.recordBuiltinCall(name, e, targetName) {
				return
			}
		} else if b.recordBuiltinCall(name, e, targetName) {
			return
		}
		b.addOperation(Operation{
			Kind:   OpCall,
			Source: sourceString(e.At),
			Inputs: callInputs(e.Args),
			Note:   note,
		})
	case *frontend.BinaryExpr:
		b.walkExpr(e.Left, "")
		b.walkExpr(e.Right, "")
	case *frontend.UnaryExpr:
		b.walkExpr(e.X, "")
	case *frontend.TryExpr:
		b.walkExpr(e.X, "")
	case *frontend.AwaitExpr:
		b.walkExpr(e.X, "")
	case *frontend.StructLitExpr:
		inputs := make([]string, 0, len(e.Fields))
		for _, field := range e.Fields {
			b.walkExpr(field.Value, "")
			if input := exprPath(field.Value); input != "" {
				inputs = append(inputs, input)
			}
		}
		if targetName != "" && len(inputs) > 0 {
			b.addOperation(
				Operation{
					Kind:    OpAggregate,
					Source:  sourceString(e.At),
					Inputs:  inputs,
					Outputs: []string{targetName},
					Note:    "struct aggregate",
				},
			)
		}
	case *frontend.IndexExpr:
		b.walkExpr(e.Base, "")
		b.walkExpr(e.Index, "")
		op := b.addOperation(Operation{
			Kind:   OpIndexLoad,
			Source: sourceString(e.At),
			Inputs: []string{exprPath(e.Base), exprPath(e.Index)},
		})
		if proof, ok := b.activeProofForIndex(e); ok {
			b.proofUses = append(b.proofUses, ProofUse{
				ProofID: proof.ID,
				Block:   op.Block,
				OpID:    op.ID,
				UseKind: "bounds_check",
				Source:  sourceString(e.At),
			})
		}
	case *frontend.FieldAccessExpr:
		b.walkExpr(e.Base, "")
	case *frontend.MatchExpr:
		b.walkExpr(e.Value, e.ScrutineeLocal)
		for _, c := range e.Cases {
			b.walkExpr(c.Guard, "")
			b.walkExpr(c.Value, "")
		}
	case *frontend.CatchExpr:
		b.walkExpr(e.Call, "")
		for _, c := range e.Cases {
			b.walkExpr(c.Guard, "")
			b.walkExpr(c.Value, "")
		}
	case *frontend.ClosureExpr:
		inputs := make([]string, 0, len(e.Captures))
		for _, capture := range e.Captures {
			if capture.Name != "" {
				inputs = append(inputs, capture.Name)
			}
		}
		if len(inputs) > 0 {
			outputs := []string(nil)
			if targetName != "" {
				outputs = []string{targetName}
			}
			b.addOperation(
				Operation{
					Kind:    OpClosure,
					Source:  sourceString(e.At),
					Inputs:  inputs,
					Outputs: outputs,
					Note:    "closure captures environment",
				},
			)
		}
	}
}

func (b *builder) recordLocalAssignment(name string, expr frontend.Expr, pos frontend.Position) {
	if name == "" {
		return
	}
	input := exprPath(expr)
	if input == "" || input == name {
		return
	}
	b.addOperation(
		Operation{
			Kind:    OpAssign,
			Source:  sourceString(pos),
			Inputs:  []string{input},
			Outputs: []string{name},
			Note:    "local assignment",
		},
	)
}

func (b *builder) recordGlobalStore(name string, expr frontend.Expr, pos frontend.Position) {
	if name == "" {
		return
	}
	input := exprPath(expr)
	if input == "" {
		return
	}
	b.addOperation(
		Operation{
			Kind:    OpGlobalStore,
			Source:  sourceString(pos),
			Inputs:  []string{input},
			Outputs: []string{name},
			Note:    "global store",
		},
	)
}

func (b *builder) isGlobalName(name string) bool {
	if name == "" || b.globals == nil {
		return false
	}
	_, ok := b.globals[name]
	return ok
}

func (b *builder) recordBuiltinCall(name string, call *frontend.CallExpr, targetName string) bool {
	if name == "core.alloc_bytes" {
		b.recordAllocBytesCall(name, call, targetName)
		return true
	}
	if name == "core.ptr_add" {
		b.recordRawPtrAddCall(name, call, targetName)
		return true
	}
	if rawMemoryAccessBuiltin(name) {
		b.recordRawMemoryAccessCall(name, call, targetName)
		return true
	}
	elem, ok := makeSliceElem(name)
	if ok {
		b.recordMakeSliceCall(name, elem, call, targetName)
		return true
	}
	elem, ok = rawSliceElem(name)
	if ok {
		b.recordRawSliceCall(name, elem, call, targetName)
		return true
	}
	elem, ok = sliceBorrowElem(name)
	if ok {
		b.recordBorrowCall(name, "[]"+elem, call, targetName)
		return true
	}
	if stringBorrowBuiltin(name) {
		b.recordBorrowCall(name, "str", call, targetName)
		return true
	}
	elem, ok = sliceCopyElem(name)
	if ok {
		b.recordCopyCall(name, "[]"+elem, elem, call, targetName)
		return true
	}
	if stringCopyBuiltin(name) {
		b.recordCopyCall(name, "str", "u8", call, targetName)
		return true
	}
	if name == "core.send_typed" {
		b.recordActorSendCall(name, call, targetName)
		return true
	}
	if name == "core.island_reset" {
		b.recordIslandResetCall(name, call, targetName)
		return true
	}
	if sliceCopyIntoBuiltin(name) || stringCopyIntoBuiltin(name) {
		b.recordCopyIntoCall(name, call)
		return true
	}
	elem, method, ok := sliceViewElem(name)
	if ok {
		b.recordSliceViewCall(name, "[]"+elem, method, call, targetName)
		return true
	}
	valueType, method, ok := stringViewBuiltin(name)
	if ok {
		b.recordSliceViewCall(name, valueType, method, call, targetName)
		return true
	}
	return false
}

func (b *builder) recordCopyIntoCall(name string, call *frontend.CallExpr) {
	source := callArgPath(call, 0)
	destination := callArgPath(call, 1)
	overlap := b.copyIntoOverlapStatus(source, destination)
	note := fmt.Sprintf(
		("%s copies into caller-owned destination without allocation " +
			"source:%s destination:%s dest_capacity_check:normal_build overlap:%s"),
		name,
		source,
		destination,
		overlap,
	)
	op := b.addOperation(
		Operation{
			Kind:   OpCall,
			Source: sourceString(call.At),
			Inputs: callInputs(call.Args),
			Note:   note,
		},
	)
	b.addCopyLoopRangeProof(name, call, op)
}

func (b *builder) recordActorSendCall(name string, call *frontend.CallExpr, targetName string) {
	op := Operation{
		Kind:   OpActorSend,
		Source: sourceString(call.At),
		Inputs: callInputs(call.Args),
		Note:   name + " typed actor ownership transfer",
	}
	if targetName != "" {
		op.Outputs = []string{targetName}
	}
	b.addOperation(op)
	if len(call.Args) < 2 {
		return
	}
	b.recordTypedActorMovedFacts(call.Args[1], call.At)
}

func (b *builder) recordIslandResetCall(name string, call *frontend.CallExpr, targetName string) {
	inputs := callInputs(call.Args)
	outputs := []string(nil)
	if targetName != "" {
		outputs = []string{valueID(ValueLocal, targetName)}
	}
	b.addOperation(Operation{
		Kind:    OpCall,
		Source:  sourceString(call.At),
		Inputs:  inputs,
		Outputs: outputs,
		Note:    name + " advances island token epoch and consumes the source token",
	})
	if len(call.Args) == 0 {
		return
	}
	source := callArgPath(call, 0)
	if source == "" || source == "?" {
		source = "island"
	}
	sourceToken := b.islandTokenForPath(source)
	nextToken := sourceToken
	nextToken.Epoch++
	if nextToken.Epoch <= 1 {
		nextToken.Epoch = 2
	}
	if targetName != "" {
		b.rememberIslandToken(targetName, nextToken)
	}
	if sourceID, ok := b.localOrParamValueIDForExpr(call.Args[0]); ok {
		b.addFact(Fact{
			Kind:    FactMoved,
			ValueID: sourceID,
			Region:  sourceToken.IslandID,
			Source:  name + " " + sourceString(call.At),
			Reason:  "island reset consumes the source token",
		})
	}
	b.addFact(Fact{
		Kind:     FactIslandEpochAdvanced,
		IslandID: sourceToken.IslandID,
		Epoch:    nextToken.Epoch,
		BaseID:   sourceToken.BaseID,
		Source:   name + " " + sourceString(call.At),
		Reason:   "island reset advances epoch and invalidates previous references",
	})
}

func (b *builder) rememberIslandToken(name string, token islandTokenState) {
	if name == "" || token.IslandID == "" {
		return
	}
	if token.Epoch <= 0 {
		token.Epoch = 1
	}
	if token.BaseID == "" {
		token.BaseID = "token:" + islandTokenRoot(token.IslandID)
	}
	b.islandTokens[name] = token
}

func (b *builder) rememberIslandTokenAlias(name string, expr frontend.Expr) {
	if name == "" {
		return
	}
	path := exprPath(expr)
	if path == "" {
		return
	}
	token, ok := b.islandTokens[path]
	if !ok {
		return
	}
	b.rememberIslandToken(name, token)
}

func (b *builder) islandTokenForPath(path string) islandTokenState {
	if token, ok := b.islandTokens[path]; ok && token.IslandID != "" {
		return token
	}
	if path == "" || path == "?" {
		path = "island"
	}
	return islandTokenState{
		IslandID: "island:" + path,
		Epoch:    1,
		BaseID:   "token:" + path,
	}
}

func islandTokenRoot(islandID string) string {
	return strings.TrimPrefix(islandID, "island:")
}

func (b *builder) recordTypedActorMovedFacts(expr frontend.Expr, pos frontend.Position) {
	msgCall, ok := expr.(*frontend.CallExpr)
	if !ok || msgCall == nil {
		return
	}
	_, caseInfo, ok := plirEnumCaseConstructor(msgCall, b.types)
	if !ok {
		return
	}
	ownerIDs := b.actorTransferOwnerValueIDs(msgCall, caseInfo)
	for i, payloadType := range caseInfo.PayloadTypes {
		if i >= len(msgCall.Args) {
			break
		}
		b.recordTypedActorMovedFactsForPayload(msgCall.Args[i], payloadType, ownerIDs, pos)
	}
}

func (b *builder) actorTransferOwnerValueIDs(
	call *frontend.CallExpr,
	caseInfo semantics.EnumCaseInfo,
) []string {
	var owners []string
	for i, payloadType := range caseInfo.PayloadTypes {
		if i >= len(call.Args) {
			break
		}
		if plirTypeKind(payloadType, b.types) != semantics.TypeIsland {
			continue
		}
		if id, ok := b.localOrParamValueIDForExpr(call.Args[i]); ok {
			owners = append(owners, id)
		}
	}
	return owners
}

func (b *builder) recordTypedActorMovedFactsForPayload(
	expr frontend.Expr,
	typeName string,
	ownerIDs []string,
	pos frontend.Position,
) {
	switch plirTypeKind(typeName, b.types) {
	case semantics.TypeIsland:
		if id, ok := b.localOrParamValueIDForExpr(expr); ok {
			b.addTypedActorMovedFact(id, nil, pos)
		}
	case semantics.TypeSlice:
		if len(ownerIDs) == 0 || plirExprIsExplicitCopy(expr) {
			return
		}
		if id, ok := b.localOrParamValueIDForExpr(expr); ok {
			b.addTypedActorMovedFact(id, ownerIDs, pos)
		}
	case semantics.TypeStruct:
		lit, ok := expr.(*frontend.StructLitExpr)
		if !ok {
			return
		}
		info, ok := b.types[typeName]
		if !ok {
			return
		}
		byName := make(map[string]frontend.Expr, len(lit.Fields))
		for _, field := range lit.Fields {
			byName[field.Name] = field.Value
		}
		for _, field := range info.Fields {
			if value := byName[field.Name]; value != nil {
				b.recordTypedActorMovedFactsForPayload(value, field.TypeName, ownerIDs, pos)
			}
		}
	case semantics.TypeEnum:
		call, ok := expr.(*frontend.CallExpr)
		if !ok || call == nil {
			return
		}
		_, caseInfo, ok := plirEnumCaseConstructor(call, b.types)
		if !ok {
			return
		}
		for i, payloadType := range caseInfo.PayloadTypes {
			if i >= len(call.Args) {
				break
			}
			b.recordTypedActorMovedFactsForPayload(call.Args[i], payloadType, ownerIDs, pos)
		}
	}
}

func (b *builder) localOrParamValueIDForExpr(expr frontend.Expr) (string, bool) {
	path := exprPath(expr)
	if path == "" {
		return "", false
	}
	for _, kind := range []ValueKind{ValueLocal, ValueParam} {
		id := valueID(kind, path)
		if _, ok := b.values[id]; ok {
			return id, true
		}
	}
	return "", false
}

func (b *builder) addTypedActorMovedFact(valueID string, uses []string, pos frontend.Position) {
	if valueID == "" || b.hasFactForValue(FactMoved, valueID) {
		return
	}
	b.addFact(Fact{
		Kind:    FactMoved,
		ValueID: valueID,
		Region:  "actor_transfer",
		Source:  "core.send_typed " + sourceString(pos),
		Reason:  "typed actor ownership transfer moved payload",
		Uses:    append([]string(nil), uses...),
	})
}

func plirEnumCaseConstructor(
	call *frontend.CallExpr,
	types map[string]*semantics.TypeInfo,
) (string, semantics.EnumCaseInfo, bool) {
	if call == nil {
		return "", semantics.EnumCaseInfo{}, false
	}
	caseName := plirCallCaseName(call.Name)
	if call.ResolvedType != "" {
		if info, ok := types[call.ResolvedType]; ok && info.Kind == semantics.TypeEnum {
			if caseInfo, ok := info.CaseMap[caseName]; ok {
				return call.ResolvedType, caseInfo, true
			}
		}
	}
	for typeName, info := range types {
		if info == nil || info.Kind != semantics.TypeEnum {
			continue
		}
		if caseInfo, ok := info.CaseMap[caseName]; ok {
			return typeName, caseInfo, true
		}
	}
	return "", semantics.EnumCaseInfo{}, false
}

func plirCallCaseName(name string) string {
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		return name[dot+1:]
	}
	return name
}

func plirTypeKind(typeName string, types map[string]*semantics.TypeInfo) semantics.TypeKind {
	if info, ok := types[typeName]; ok && info != nil {
		return info.Kind
	}
	return semantics.TypeKind(-1)
}

func plirExprIsExplicitCopy(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	return copyBuiltin(name)
}
