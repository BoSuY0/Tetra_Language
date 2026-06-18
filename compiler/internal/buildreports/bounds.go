package buildreports

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
)

func BuildProofReport(plirProg *plir.Program, bounds BoundsReport, target string) ProofReport {
	return ProofReport{
		ReportEnvelope: ReportEnvelope{SchemaVersion: 1, Kind: "proof", Target: target},
		Bounds:         bounds,
		Proofs:         buildProofEvidence(plirProg, bounds),
		PLIR:           plirProg,
	}
}

func buildProofEvidence(prog *plir.Program, bounds BoundsReport) []ProofEvidence {
	if prog == nil {
		return nil
	}
	removed := map[string]bool{}
	for _, fn := range bounds.Functions {
		for _, site := range fn.Sites {
			if site.Removed && site.ProofID != "" {
				removed[site.ProofID] = true
			}
		}
	}
	removedUses := boundsRemovedProofUsesByID(prog, bounds)
	for proofID := range removedUses {
		removed[proofID] = true
	}
	var out []ProofEvidence
	for _, fn := range prog.Funcs {
		facts := proofFactsByID(fn)
		for _, guard := range fn.ProofGuards {
			uses := guard.Dominates
			if len(uses) == 0 {
				for _, use := range fn.ProofUses {
					if use.ProofID == guard.ID {
						uses = append(uses, use)
					}
				}
			}
			uses = appendProofUsesUnique(uses, removedUses[guard.ID]...)
			if len(uses) == 0 {
				out = append(out, ProofEvidence{
					ProofID:            guard.ID,
					Kind:               guard.Kind,
					Guard:              guard.Condition,
					Fact:               facts[guard.ID],
					Reason:             guard.Reason,
					RemovedBoundsCheck: proofIDRemoved(guard.ID, removed),
				})
				continue
			}
			for _, use := range uses {
				out = append(out, ProofEvidence{
					ProofID:            guard.ID,
					Kind:               guard.Kind,
					Guard:              guard.Condition,
					Dominates:          use.UseKind + " " + use.OpID,
					Fact:               facts[guard.ID],
					Reason:             guard.Reason,
					RemovedBoundsCheck: proofIDRemoved(guard.ID, removed),
				})
			}
		}
	}
	return out
}

func proofIDRemoved(proofID string, removed map[string]bool) bool {
	if removed[proofID] {
		return true
	}
	for removedID := range removed {
		if removedID != "" && strings.HasPrefix(proofID, removedID+":") {
			return true
		}
	}
	return false
}

func boundsRemovedProofUsesByID(
	prog *plir.Program,
	bounds BoundsReport,
) map[string][]plir.ProofUse {
	out := map[string][]plir.ProofUse{}
	if prog == nil {
		return out
	}
	funcs := map[string]plir.Function{}
	for _, fn := range prog.Funcs {
		funcs[fn.Name] = fn
	}
	for _, row := range bounds.Functions {
		fn, ok := funcs[row.Function]
		if !ok {
			continue
		}
		for _, site := range row.Sites {
			if !site.Removed || site.ProofID == "" {
				continue
			}
			opKind, ok := proofUseOpKindForBoundsSite(site.Kind)
			if !ok {
				continue
			}
			op, ok := proofUseOpForSite(fn, opKind, site.Site)
			if !ok {
				continue
			}
			guard, proofID, ok := proofGuardForBoundsSite(fn, site.ProofID, opKind)
			if !ok || !proofGuardDominatesBlock(fn, guard, op.Block) {
				continue
			}
			out[proofID] = append(out[proofID], plir.ProofUse{
				ProofID: proofID,
				Block:   op.Block,
				OpID:    op.ID,
				UseKind: "bounds_check",
				Source:  site.Site,
			})
		}
	}
	return out
}

func appendProofUsesUnique(uses []plir.ProofUse, extra ...plir.ProofUse) []plir.ProofUse {
	seen := map[string]bool{}
	for _, use := range uses {
		seen[proofUseKey(use)] = true
	}
	for _, use := range extra {
		key := proofUseKey(use)
		if seen[key] {
			continue
		}
		uses = append(uses, use)
		seen[key] = true
	}
	return uses
}

func proofUseKey(use plir.ProofUse) string {
	return use.ProofID + "\x00" + use.Block + "\x00" + use.OpID + "\x00" + use.UseKind + "\x00" + use.Source
}

func proofUseOpKindForBoundsSite(kind string) (plir.OperationKind, bool) {
	switch kind {
	case "i32.load", "u8.load", "u16.load":
		return plir.OpIndexLoad, true
	case "i32.store", "u8.store", "u16.store":
		return plir.OpIndexStore, true
	default:
		return "", false
	}
}

func proofUseOpForSite(
	fn plir.Function,
	kind plir.OperationKind,
	site string,
) (plir.Operation, bool) {
	for _, op := range fn.Ops {
		if op.Kind == kind && op.Source == site {
			return op, true
		}
	}
	return plir.Operation{}, false
}

func proofGuardByID(fn plir.Function, proofID string) (plir.ProofGuard, bool) {
	for _, guard := range fn.ProofGuards {
		if guard.ID == proofID {
			return guard, true
		}
	}
	return plir.ProofGuard{}, false
}

func proofGuardForBoundsSite(
	fn plir.Function,
	proofID string,
	opKind plir.OperationKind,
) (plir.ProofGuard, string, bool) {
	if guard, ok := proofGuardByID(fn, proofID); ok {
		return guard, proofID, true
	}
	operation := proofOperationForOpKind(opKind)
	for _, term := range fn.ProofTerms {
		if !strings.HasPrefix(term.ID, proofID+":") {
			continue
		}
		if operation != "" && term.Operation != operation {
			continue
		}
		if guard, ok := proofGuardByID(fn, term.ID); ok {
			return guard, term.ID, true
		}
	}
	for _, guard := range fn.ProofGuards {
		if strings.HasPrefix(guard.ID, proofID+":") {
			return guard, guard.ID, true
		}
	}
	return plir.ProofGuard{}, "", false
}

func proofOperationForOpKind(kind plir.OperationKind) string {
	switch kind {
	case plir.OpIndexLoad:
		return "index_load"
	case plir.OpIndexStore:
		return "index_store"
	default:
		return ""
	}
}

func proofGuardDominatesBlock(fn plir.Function, guard plir.ProofGuard, block string) bool {
	if guard.Block == "" || block == "" {
		return false
	}
	if guard.Block == block {
		return true
	}
	for _, row := range fn.Dominators {
		if row.Block != block {
			continue
		}
		for _, dom := range row.Dominators {
			if dom == guard.Block {
				return true
			}
		}
	}
	return false
}

func proofFactsByID(fn plir.Function) map[string]string {
	out := map[string]string{}
	for _, fact := range fn.Facts {
		if fact.ProofID == "" || fact.Kind != plir.FactIndexInRange {
			continue
		}
		out[fact.ProofID] = fact.ValueID
		if fact.Range != "" {
			out[fact.ProofID] = fmt.Sprintf("%s in [%s]", fact.ValueID, fact.Range)
		}
	}
	for _, fact := range fn.RangeFacts {
		if fact.ProofID == "" {
			continue
		}
		out[fact.ProofID] = formatRangeEvidence(fact)
	}
	return out
}

func formatRangeEvidence(fact plir.RangeFact) string {
	lower := formatReportBound(fact.Lower)
	upper := formatReportBound(fact.Upper)
	open := "["
	close := ")"
	if !fact.InclusiveLower {
		open = "("
	}
	if fact.InclusiveUpper {
		close = "]"
	}
	name := fact.Value
	if strings.HasPrefix(name, "local:") {
		name = strings.TrimPrefix(name, "local:")
	}
	if strings.HasPrefix(name, "loop_index:") {
		name = strings.TrimPrefix(name, "loop_index:")
	}
	text := fmt.Sprintf("%s in %s%s, %s%s", name, open, lower, upper, close)
	if len(fact.Derivation) > 0 {
		text += "; derivation: " + strings.Join(fact.Derivation, ", ")
	}
	return text
}

func formatReportBound(bound plir.Bound) string {
	switch bound.Kind {
	case plir.BoundConst:
		return fmt.Sprintf("%d", bound.Const)
	case plir.BoundSymbol:
		return bound.Symbol
	case plir.BoundSymbolMinus:
		return fmt.Sprintf("%s - %d", bound.Symbol, bound.Const)
	default:
		return string(bound.Kind)
	}
}

func BuildBoundsReport(
	prog *ir.IRProgram,
	checked *semantics.CheckedProgram,
	target string,
) BoundsReport {
	report := BoundsReport{
		ReportEnvelope: ReportEnvelope{SchemaVersion: 1, Kind: "bounds", Target: target},
	}
	if prog == nil {
		return report
	}
	leftReasons := buildBoundsLeftReasonIndex(checked)
	removedReasons := buildBoundsRemovedReasonIndex(checked)
	for _, fn := range prog.Funcs {
		row := BoundsFunctionRow{Function: fn.Name}
		for _, instr := range fn.Instrs {
			switch {
			case isUncheckedIndexLoad(instr.Kind):
				row.Removed++
				report.Totals.Removed++
				row.Sites = append(row.Sites, BoundsCheckSite{
					Site:    reportPos(instr.Pos),
					Kind:    irIndexKind(instr.Kind),
					Removed: true,
					ProofID: instr.ProofID,
					Reason: removedBoundsReasonForSite(
						fn.Name,
						instr.Pos,
						instr.ProofID,
						removedReasons,
					),
				})
			case isProofTaggedIndexStore(instr):
				row.Removed++
				report.Totals.Removed++
				row.Sites = append(row.Sites, BoundsCheckSite{
					Site:    reportPos(instr.Pos),
					Kind:    irIndexKind(instr.Kind),
					Removed: true,
					ProofID: instr.ProofID,
					Reason: removedBoundsReasonForSite(
						fn.Name,
						instr.Pos,
						instr.ProofID,
						removedReasons,
					),
				})
			case isCheckedIndexAccess(instr.Kind):
				row.Left++
				report.Totals.Left++
				row.Sites = append(row.Sites, BoundsCheckSite{
					Site:    reportPos(instr.Pos),
					Kind:    irIndexKind(instr.Kind),
					Removed: false,
					Reason:  leftBoundsReason(fn.Name, instr.Pos, leftReasons),
				})
			}
		}
		if row.Removed > 0 || row.Left > 0 {
			report.Functions = append(report.Functions, row)
		}
	}
	return report
}

func isProofTaggedIndexStore(instr ir.IRInstr) bool {
	if instr.ProofID == "" {
		return false
	}
	switch instr.Kind {
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		return true
	default:
		return false
	}
}

func isUncheckedIndexLoad(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
		return true
	default:
		return false
	}
}

func isCheckedIndexAccess(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
		ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		return true
	default:
		return false
	}
}

func irIndexKind(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked:
		return "i32.load"
	case ir.IRIndexLoadU8, ir.IRIndexLoadU8Unchecked:
		return "u8.load"
	case ir.IRIndexLoadU16, ir.IRIndexLoadU16Unchecked:
		return "u16.load"
	case ir.IRIndexStoreI32:
		return "i32.store"
	case ir.IRIndexStoreU8:
		return "u8.store"
	case ir.IRIndexStoreU16:
		return "u16.store"
	default:
		return fmt.Sprintf("ir.%d", kind)
	}
}

type boundsLeftReasonKey struct {
	Function string
	File     string
	Line     int
	Col      int
}

type boundsBranchGuard struct {
	Index string
	Base  string
}

type boundsLeftReasonContext struct {
	seenBranchGuards        []boundsBranchGuard
	missingLowerBoundGuards []boundsBranchGuard
	activeProofGuards       []boundsBranchGuard
	mutationInvalidated     []boundsBranchGuard
}

type boundsLeftReasonBuilder struct {
	function string
	funcs    map[string]semantics.FuncSig
	locals   map[string]semantics.LocalInfo
	globals  map[string]semantics.GlobalInfo
	reasons  map[boundsLeftReasonKey]string
}

func buildBoundsLeftReasonIndex(checked *semantics.CheckedProgram) map[boundsLeftReasonKey]string {
	reasons := map[boundsLeftReasonKey]string{}
	if checked == nil {
		return reasons
	}
	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		builder := boundsLeftReasonBuilder{
			function: fn.Name,
			funcs:    checked.FuncSigs,
			locals:   fn.Locals,
			globals:  checked.GlobalsByModule[fn.Module],
			reasons:  reasons,
		}
		builder.walkBoundsReasonStmts(fn.Decl.Body, boundsLeftReasonContext{})
	}
	return reasons
}

func leftBoundsReason(
	function string,
	pos frontend.Position,
	reasons map[boundsLeftReasonKey]string,
) string {
	if reason := reasons[boundsLeftReasonKeyFor(function, pos)]; reason != "" {
		return reason
	}
	return "left_missing_dominance"
}

func removedBoundsReasonForSite(
	function string,
	pos frontend.Position,
	proofID string,
	reasons map[boundsLeftReasonKey]string,
) string {
	if reason := reasons[boundsLeftReasonKeyFor(function, pos)]; reason != "" {
		return reason
	}
	return removedBoundsReason(proofID)
}

func boundsLeftReasonKeyFor(function string, pos frontend.Position) boundsLeftReasonKey {
	return boundsLeftReasonKey{Function: function, File: pos.File, Line: pos.Line, Col: pos.Col}
}

func (b *boundsLeftReasonBuilder) walkBoundsReasonStmts(
	stmts []frontend.Stmt,
	ctx boundsLeftReasonContext,
) boundsLeftReasonContext {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
			ctx = b.markCallMutationsInExpr(ctx, s.Value)
		case *frontend.ReturnStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
		case *frontend.ThrowStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
		case *frontend.DeferStmt:
			b.walkBoundsReasonStmts(s.Body, ctx)
		case *frontend.LetStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
			ctx = b.markCallMutationsInExpr(ctx, s.Value)
		case *frontend.AssignStmt:
			if idx, ok := s.Target.(*frontend.IndexExpr); ok && idx != nil {
				b.markBoundsIndexReason(idx, ctx, s.At)
				b.walkBoundsReasonExpr(idx.Base, ctx, frontend.Position{})
				b.walkBoundsReasonExpr(idx.Index, ctx, frontend.Position{})
			} else {
				b.walkBoundsReasonExpr(s.Target, ctx, frontend.Position{})
			}
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(s.CompoundValue, ctx, frontend.Position{})
			ctx = b.markCallMutationsInExpr(ctx, s.Value)
			ctx = b.markCallMutationsInExpr(ctx, s.CompoundValue)
			if id, ok := s.Target.(*frontend.IdentExpr); ok && id != nil {
				ctx = b.markMutationInvalidated(ctx, id.Name)
			}
		case *frontend.IfStmt:
			b.walkBoundsReasonExpr(s.Cond, ctx, frontend.Position{})
			if guard, ok := reportMissingLowerBranchGuard(s.Cond); ok {
				thenCtx := ctx
				thenCtx.missingLowerBoundGuards = appendBoundsBranchGuard(ctx.missingLowerBoundGuards, guard)
				b.walkBoundsReasonStmts(s.Then, thenCtx)
				b.walkBoundsReasonStmts(s.Else, ctx)
				continue
			}
			if guard, ok := reportFullBranchGuard(s.Cond); ok {
				b.walkBoundsReasonStmts(s.Then, ctx)
				b.walkBoundsReasonStmts(s.Else, ctx)
				ctx.seenBranchGuards = append(ctx.seenBranchGuards, guard)
				continue
			}
			b.walkBoundsReasonStmts(s.Then, ctx)
			b.walkBoundsReasonStmts(s.Else, ctx)
		case *frontend.IfLetStmt:
			b.walkBoundsReasonExpr(s.Pattern, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
			b.walkBoundsReasonStmts(s.Then, ctx)
			b.walkBoundsReasonStmts(s.Else, ctx)
		case *frontend.WhileStmt:
			b.walkBoundsReasonExpr(s.Cond, ctx, frontend.Position{})
			bodyCtx := ctx
			if guard, ok := reportUpperBranchGuard(s.Cond); ok {
				bodyCtx.activeProofGuards = appendBoundsBranchGuard(bodyCtx.activeProofGuards, guard)
			}
			b.walkBoundsReasonStmts(s.Body, bodyCtx)
		case *frontend.ForRangeStmt:
			b.walkBoundsReasonExpr(s.Start, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(s.End, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(s.Iterable, ctx, frontend.Position{})
			b.walkBoundsReasonStmts(s.Body, ctx)
		case *frontend.MatchStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
			for _, c := range s.Cases {
				b.walkBoundsReasonExpr(c.Pattern, ctx, frontend.Position{})
				b.walkBoundsReasonExpr(c.Guard, ctx, frontend.Position{})
				b.walkBoundsReasonStmts(c.Body, ctx)
			}
		case *frontend.FreeStmt:
			b.walkBoundsReasonExpr(s.Value, ctx, frontend.Position{})
		case *frontend.UnsafeStmt:
			b.walkBoundsReasonStmts(s.Body, ctx)
		case *frontend.IslandStmt:
			b.walkBoundsReasonExpr(s.Size, ctx, frontend.Position{})
			b.walkBoundsReasonStmts(s.Body, ctx)
		case *frontend.ExprStmt:
			b.walkBoundsReasonExpr(s.Expr, ctx, frontend.Position{})
			ctx = b.markCallMutationsInExpr(ctx, s.Expr)
		case *frontend.ExpectStmt:
			b.walkBoundsReasonExpr(s.Cond, ctx, frontend.Position{})
		}
	}
	return ctx
}

func (b *boundsLeftReasonBuilder) walkBoundsReasonExpr(
	expr frontend.Expr,
	ctx boundsLeftReasonContext,
	siteOverride frontend.Position,
) {
	switch e := expr.(type) {
	case *frontend.BinaryExpr:
		b.walkBoundsReasonExpr(e.Left, ctx, frontend.Position{})
		b.walkBoundsReasonExpr(e.Right, ctx, frontend.Position{})
	case *frontend.UnaryExpr:
		b.walkBoundsReasonExpr(e.X, ctx, frontend.Position{})
	case *frontend.TryExpr:
		b.walkBoundsReasonExpr(e.X, ctx, frontend.Position{})
	case *frontend.AwaitExpr:
		b.walkBoundsReasonExpr(e.X, ctx, frontend.Position{})
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			b.walkBoundsReasonExpr(arg, ctx, frontend.Position{})
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			b.walkBoundsReasonExpr(field.Value, ctx, frontend.Position{})
		}
	case *frontend.FieldAccessExpr:
		b.walkBoundsReasonExpr(e.Base, ctx, frontend.Position{})
	case *frontend.IndexExpr:
		b.markBoundsIndexReason(e, ctx, siteOverride)
		b.walkBoundsReasonExpr(e.Base, ctx, frontend.Position{})
		b.walkBoundsReasonExpr(e.Index, ctx, frontend.Position{})
	case *frontend.MatchExpr:
		b.walkBoundsReasonExpr(e.Value, ctx, frontend.Position{})
		for _, c := range e.Cases {
			b.walkBoundsReasonExpr(c.Pattern, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(c.Guard, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(c.Value, ctx, frontend.Position{})
		}
	case *frontend.CatchExpr:
		b.walkBoundsReasonExpr(e.Call, ctx, frontend.Position{})
		for _, c := range e.Cases {
			b.walkBoundsReasonExpr(c.Pattern, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(c.Guard, ctx, frontend.Position{})
			b.walkBoundsReasonExpr(c.Value, ctx, frontend.Position{})
		}
	}
}

func (b *boundsLeftReasonBuilder) markBoundsIndexReason(
	index *frontend.IndexExpr,
	ctx boundsLeftReasonContext,
	siteOverride frontend.Position,
) {
	if index == nil {
		return
	}
	guard := boundsBranchGuard{Base: reportExprPath(index.Base), Index: reportExprPath(index.Index)}
	if guard.Base == "" || guard.Index == "" {
		return
	}
	reason := ""
	if boundsGuardListContains(ctx.mutationInvalidated, guard) {
		reason = "left_proof_invalidated_by_mutation"
	} else if boundsGuardListContains(ctx.missingLowerBoundGuards, guard) {
		reason = "left_missing_non_negative_lower_bound"
	} else if boundsGuardListContains(ctx.seenBranchGuards, guard) {
		reason = "left_guard_not_dominating"
	}
	if reason == "" {
		return
	}
	pos := index.At
	if siteOverride.Line != 0 || siteOverride.Col != 0 || siteOverride.File != "" {
		pos = siteOverride
	}
	b.setBoundsLeftReason(pos, reason)
}

func (b *boundsLeftReasonBuilder) markMutationInvalidated(
	ctx boundsLeftReasonContext,
	name string,
) boundsLeftReasonContext {
	if name == "" {
		return ctx
	}
	for _, guard := range ctx.activeProofGuards {
		if reportProofPathMatchesMutation(guard.Index, name) ||
			reportProofPathMatchesMutation(guard.Base, name) {
			ctx.mutationInvalidated = appendBoundsBranchGuard(ctx.mutationInvalidated, guard)
		}
	}
	return ctx
}

func (b *boundsLeftReasonBuilder) markCallMutationsInExpr(
	ctx boundsLeftReasonContext,
	expr frontend.Expr,
) boundsLeftReasonContext {
	switch e := expr.(type) {
	case *frontend.BinaryExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.Left)
		ctx = b.markCallMutationsInExpr(ctx, e.Right)
	case *frontend.UnaryExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.X)
	case *frontend.TryExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.X)
	case *frontend.AwaitExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.X)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			ctx = b.markCallMutationsInExpr(ctx, arg)
		}
		ctx = b.markCallMutationInvalidated(ctx, e)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			ctx = b.markCallMutationsInExpr(ctx, field.Value)
		}
	case *frontend.FieldAccessExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.Base)
	case *frontend.IndexExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.Base)
		ctx = b.markCallMutationsInExpr(ctx, e.Index)
	case *frontend.MatchExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.Value)
		for _, c := range e.Cases {
			ctx = b.markCallMutationsInExpr(ctx, c.Pattern)
			ctx = b.markCallMutationsInExpr(ctx, c.Guard)
			ctx = b.markCallMutationsInExpr(ctx, c.Value)
		}
	case *frontend.CatchExpr:
		ctx = b.markCallMutationsInExpr(ctx, e.Call)
		for _, c := range e.Cases {
			ctx = b.markCallMutationsInExpr(ctx, c.Pattern)
			ctx = b.markCallMutationsInExpr(ctx, c.Guard)
			ctx = b.markCallMutationsInExpr(ctx, c.Value)
		}
	}
	return ctx
}

func (b *boundsLeftReasonBuilder) markCallMutationInvalidated(
	ctx boundsLeftReasonContext,
	call *frontend.CallExpr,
) boundsLeftReasonContext {
	if call == nil {
		return ctx
	}
	ownership := b.callParamOwnership(call.Name)
	for i, owner := range ownership {
		if owner != "inout" {
			continue
		}
		if i >= len(call.Args) {
			break
		}
		path := reportExprPath(call.Args[i])
		if path == "" {
			continue
		}
		ctx = b.markMutationInvalidated(ctx, path)
	}
	return ctx
}

func (b *boundsLeftReasonBuilder) callParamOwnership(name string) []string {
	if name == "" {
		return nil
	}
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	if b.funcs != nil {
		if sig, ok := b.funcs[name]; ok {
			return sig.ParamOwnership
		}
	}
	if local, ok := b.locals[name]; ok && local.FunctionTypeValue {
		return local.FunctionParamOwnership
	}
	if b.globals != nil {
		if global, ok := b.globals[name]; ok && global.FunctionTypeValue {
			return global.FunctionParamOwnership
		}
	}
	return nil
}

func reportProofPathMatchesMutation(proofPath string, mutatedPath string) bool {
	if proofPath == "" || mutatedPath == "" {
		return false
	}
	return proofPath == mutatedPath || strings.HasPrefix(proofPath, mutatedPath+".")
}

func (b *boundsLeftReasonBuilder) setBoundsLeftReason(pos frontend.Position, reason string) {
	if pos.Line == 0 && pos.Col == 0 && pos.File == "" {
		return
	}
	key := boundsLeftReasonKeyFor(b.function, pos)
	if existing := b.reasons[key]; existing == "left_missing_non_negative_lower_bound" {
		return
	}
	b.reasons[key] = reason
}

func boundsGuardListContains(guards []boundsBranchGuard, want boundsBranchGuard) bool {
	for _, guard := range guards {
		if guard == want {
			return true
		}
	}
	return false
}

func appendBoundsBranchGuard(
	guards []boundsBranchGuard,
	guard boundsBranchGuard,
) []boundsBranchGuard {
	out := make([]boundsBranchGuard, 0, len(guards)+1)
	out = append(out, guards...)
	out = append(out, guard)
	return out
}

func reportMissingLowerBranchGuard(cond frontend.Expr) (boundsBranchGuard, bool) {
	if _, ok := reportFullBranchGuard(cond); ok {
		return boundsBranchGuard{}, false
	}
	return reportUpperBranchGuard(cond)
}

func reportFullBranchGuard(cond frontend.Expr) (boundsBranchGuard, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenAmpAmp {
		return boundsBranchGuard{}, false
	}
	if guard, ok := reportFullBranchGuardParts(bin.Left, bin.Right); ok {
		return guard, true
	}
	return reportFullBranchGuardParts(bin.Right, bin.Left)
}

func reportFullBranchGuardParts(
	lower frontend.Expr,
	upper frontend.Expr,
) (boundsBranchGuard, bool) {
	lowerIndex, ok := reportNonNegativeGuardIndex(lower)
	if !ok {
		return boundsBranchGuard{}, false
	}
	upperGuard, ok := reportUpperBranchGuard(upper)
	if !ok || upperGuard.Index != lowerIndex {
		return boundsBranchGuard{}, false
	}
	return upperGuard, true
}

func reportUpperBranchGuard(cond frontend.Expr) (boundsBranchGuard, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return boundsBranchGuard{}, false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil {
		return boundsBranchGuard{}, false
	}
	var base string
	switch bin.Op {
	case frontend.TokenLess:
		base = reportLenFieldBaseName(bin.Right)
	case frontend.TokenLessEq:
		base = reportLenMinusOneBaseName(bin.Right)
	}
	if base == "" {
		return boundsBranchGuard{}, false
	}
	return boundsBranchGuard{Index: left.Name, Base: base}, true
}

func reportNonNegativeGuardIndex(expr frontend.Expr) (string, bool) {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return "", false
	}
	if left, ok := bin.Left.(*frontend.IdentExpr); ok && left != nil &&
		bin.Op == frontend.TokenGreaterEq &&
		reportIsZeroNumber(bin.Right) {
		return left.Name, true
	}
	if right, ok := bin.Right.(*frontend.IdentExpr); ok && right != nil &&
		bin.Op == frontend.TokenLessEq &&
		reportIsZeroNumber(bin.Left) {
		return right.Name, true
	}
	return "", false
}

func reportLenFieldBaseName(expr frontend.Expr) string {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok || field == nil || field.Field != "len" {
		return ""
	}
	return reportExprPath(field.Base)
}

func reportLenMinusOneBaseName(expr frontend.Expr) string {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenMinus {
		return ""
	}
	right, ok := bin.Right.(*frontend.NumberExpr)
	if !ok || right == nil || right.Value != 1 {
		return ""
	}
	return reportLenFieldBaseName(bin.Left)
}

func reportIsZeroNumber(expr frontend.Expr) bool {
	num, ok := expr.(*frontend.NumberExpr)
	return ok && num != nil && num.Value == 0
}

type boundsReportViewInfo struct {
	isView   bool
	composed bool
	unsafe   bool
}

type boundsRemovedReasonBuilder struct {
	function string
	reasons  map[boundsLeftReasonKey]string
	locals   map[string]boundsReportViewInfo
}

func buildBoundsRemovedReasonIndex(
	checked *semantics.CheckedProgram,
) map[boundsLeftReasonKey]string {
	reasons := map[boundsLeftReasonKey]string{}
	if checked == nil {
		return reasons
	}
	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		builder := boundsRemovedReasonBuilder{
			function: fn.Name,
			reasons:  reasons,
			locals:   map[string]boundsReportViewInfo{},
		}
		builder.walkRemovedReasonStmts(fn.Decl.Body)
	}
	return reasons
}

func (b *boundsRemovedReasonBuilder) walkRemovedReasonStmts(stmts []frontend.Stmt) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			b.rememberViewLocal(s.Name, s.Value)
		case *frontend.AssignStmt:
			if id, ok := s.Target.(*frontend.IdentExpr); ok && id != nil {
				b.rememberViewLocal(id.Name, s.Value)
			}
		case *frontend.IfStmt:
			thenLocals := cloneBoundsReportViewInfoMap(b.locals)
			elseLocals := cloneBoundsReportViewInfoMap(b.locals)
			thenBuilder := boundsRemovedReasonBuilder{
				function: b.function,
				reasons:  b.reasons,
				locals:   thenLocals,
			}
			elseBuilder := boundsRemovedReasonBuilder{
				function: b.function,
				reasons:  b.reasons,
				locals:   elseLocals,
			}
			thenBuilder.walkRemovedReasonStmts(s.Then)
			elseBuilder.walkRemovedReasonStmts(s.Else)
			b.locals = mergeBoundsReportViewInfoMaps(thenBuilder.locals, elseBuilder.locals)
		case *frontend.IfLetStmt:
			b.walkRemovedReasonStmts(s.Then)
			b.walkRemovedReasonStmts(s.Else)
		case *frontend.WhileStmt:
			b.walkRemovedReasonStmts(s.Body)
		case *frontend.ForRangeStmt:
			if info := b.viewChainInfo(s.Iterable); info.composed && !info.unsafe {
				b.setRemovedReason(s.At, "removed_by_view_chain")
			}
			b.walkRemovedReasonStmts(s.Body)
		case *frontend.MatchStmt:
			for _, c := range s.Cases {
				b.walkRemovedReasonStmts(c.Body)
			}
		case *frontend.DeferStmt:
			b.walkRemovedReasonStmts(s.Body)
		case *frontend.UnsafeStmt:
			b.walkRemovedReasonStmts(s.Body)
		case *frontend.IslandStmt:
			b.walkRemovedReasonStmts(s.Body)
		}
	}
}

func (b *boundsRemovedReasonBuilder) rememberViewLocal(name string, expr frontend.Expr) {
	if name == "" {
		return
	}
	info := b.viewChainInfo(expr)
	if !info.isView && !info.unsafe {
		delete(b.locals, name)
		return
	}
	b.locals[name] = info
}

func (b *boundsRemovedReasonBuilder) viewChainInfo(expr frontend.Expr) boundsReportViewInfo {
	switch e := expr.(type) {
	case nil:
		return boundsReportViewInfo{}
	case *frontend.IdentExpr:
		if e == nil {
			return boundsReportViewInfo{}
		}
		return b.locals[e.Name]
	case *frontend.CallExpr:
		if e == nil {
			return boundsReportViewInfo{}
		}
		name := reportResolvedBuiltinName(e.Name)
		if reportRawSliceBuiltinName(name) {
			return boundsReportViewInfo{isView: true, unsafe: true}
		}
		if reportCopyResultBuiltinName(name) {
			return boundsReportViewInfo{}
		}
		if reportBorrowBuiltinName(name) {
			source := b.viewChainInfo(reportCallArg(e, 0))
			return boundsReportViewInfo{isView: true, composed: source.composed, unsafe: source.unsafe}
		}
		if reportViewBuiltinName(name) {
			source := b.viewChainInfo(reportCallArg(e, 0))
			return boundsReportViewInfo{
				isView:   true,
				composed: source.isView || source.composed,
				unsafe:   source.unsafe || reportStaticInvalidStringViewCall(name, e),
			}
		}
	}
	return boundsReportViewInfo{}
}

func (b *boundsRemovedReasonBuilder) setRemovedReason(pos frontend.Position, reason string) {
	if pos.Line == 0 && pos.Col == 0 && pos.File == "" {
		return
	}
	b.reasons[boundsLeftReasonKeyFor(b.function, pos)] = reason
}

func cloneBoundsReportViewInfoMap(
	in map[string]boundsReportViewInfo,
) map[string]boundsReportViewInfo {
	out := make(map[string]boundsReportViewInfo, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func mergeBoundsReportViewInfoMaps(
	left, right map[string]boundsReportViewInfo,
) map[string]boundsReportViewInfo {
	out := map[string]boundsReportViewInfo{}
	keys := map[string]bool{}
	for key := range left {
		keys[key] = true
	}
	for key := range right {
		keys[key] = true
	}
	for key := range keys {
		l, lok := left[key]
		r, rok := right[key]
		if !lok || !rok {
			continue
		}
		info := boundsReportViewInfo{
			isView:   l.isView && r.isView,
			composed: l.composed && r.composed,
			unsafe:   l.unsafe || r.unsafe,
		}
		if info.isView || info.unsafe {
			out[key] = info
		}
	}
	return out
}

func reportResolvedBuiltinName(name string) string {
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		return target
	}
	return name
}

func reportRawSliceBuiltinName(name string) bool {
	switch name {
	case "core.raw_slice_u8_from_parts",
		"core.raw_slice_u16_from_parts",
		"core.raw_slice_i32_from_parts",
		"core.raw_slice_bool_from_parts":
		return true
	default:
		return false
	}
}

func reportCopyResultBuiltinName(name string) bool {
	if name == "core.string_copy" {
		return true
	}
	if !strings.HasPrefix(name, "core.slice_copy_") ||
		strings.HasPrefix(name, "core.slice_copy_into_") {
		return false
	}
	switch strings.TrimPrefix(name, "core.slice_copy_") {
	case "u8", "u16", "i32", "bool":
		return true
	default:
		return false
	}
}

func reportBorrowBuiltinName(name string) bool {
	if name == "core.string_borrow" {
		return true
	}
	if !strings.HasPrefix(name, "core.slice_borrow_") {
		return false
	}
	switch strings.TrimPrefix(name, "core.slice_borrow_") {
	case "u8", "u16", "i32", "bool":
		return true
	default:
		return false
	}
}

func reportViewBuiltinName(name string) bool {
	if name == "core.string_window" || name == "core.string_prefix" ||
		name == "core.string_suffix" {
		return true
	}
	for _, prefix := range []string{"core.slice_window_", "core.slice_prefix_", "core.slice_suffix_"} {
		if strings.HasPrefix(name, prefix) {
			switch strings.TrimPrefix(name, prefix) {
			case "u8", "u16", "i32", "bool":
				return true
			}
		}
	}
	return false
}

func reportStaticInvalidStringViewCall(name string, call *frontend.CallExpr) bool {
	if call == nil || !strings.HasPrefix(name, "core.string_") {
		return false
	}
	sourceLen, knownLen := reportStaticStringByteLen(reportCallArg(call, 0))
	if !knownLen {
		return false
	}
	switch name {
	case "core.string_window":
		start, startKnown := reportEvalConstInt64(reportCallArg(call, 1))
		count, countKnown := reportEvalConstInt64(reportCallArg(call, 2))
		if !startKnown || !countKnown {
			return false
		}
		return start < 0 || count < 0 || start > sourceLen || count > sourceLen-start
	case "core.string_prefix":
		count, known := reportEvalConstInt64(reportCallArg(call, 1))
		return known && (count < 0 || count > sourceLen)
	case "core.string_suffix":
		start, known := reportEvalConstInt64(reportCallArg(call, 1))
		return known && (start < 0 || start > sourceLen)
	default:
		return false
	}
}

func reportCallArg(call *frontend.CallExpr, index int) frontend.Expr {
	if call == nil || index < 0 || index >= len(call.Args) {
		return nil
	}
	return call.Args[index]
}

func reportStaticStringByteLen(expr frontend.Expr) (int64, bool) {
	lit, ok := expr.(*frontend.StringLitExpr)
	if !ok || lit == nil {
		return 0, false
	}
	return int64(len(lit.Value)), true
}

func reportEvalConstInt64(expr frontend.Expr) (int64, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		if e == nil {
			return 0, false
		}
		return int64(e.Value), true
	case *frontend.UnaryExpr:
		if e == nil || e.Op != frontend.TokenMinus {
			return 0, false
		}
		value, ok := reportEvalConstInt64(e.X)
		if !ok {
			return 0, false
		}
		return -value, true
	default:
		return 0, false
	}
}

func removedBoundsReason(proofID string) string {
	switch {
	case strings.HasPrefix(proofID, "proof:while:"):
		return "removed_by_while_range"
	case strings.HasPrefix(proofID, "proof:if:"):
		return "removed_by_branch_guard"
	case strings.HasPrefix(proofID, "proof:copy-loop:"):
		return "removed_by_copy_loop_range"
	case strings.HasPrefix(proofID, "proof:call-boundary:"):
		return "removed_by_call_boundary_range"
	case strings.HasPrefix(proofID, "proof:helper-summary:"):
		return "removed_by_helper_summary_range"
	case strings.HasPrefix(proofID, "proof:helper-offset:"):
		return "removed_by_helper_offset_range"
	case strings.HasPrefix(proofID, "proof:allocation-zero:"):
		return "removed_by_allocation_literal_zero_length"
	case strings.HasPrefix(proofID, "proof:for-collection-view:"):
		return "removed_by_view_constructor"
	case strings.HasPrefix(proofID, "proof:for-collection:"):
		return "removed_by_for_loop_range"
	default:
		return "removed_by_for_loop_range"
	}
}
