package lower

import (
	"fmt"
	"strings"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowerlets "tetra_language/compiler/internal/lower/lets"
	"tetra_language/compiler/internal/semantics"
)

func (l *lowerer) ensureDiscardLocal() int {
	if l.discardLocal >= 0 {
		return l.discardLocal
	}
	l.discardLocal = l.localSlots
	l.localSlots++
	return l.discardLocal
}

func (l *lowerer) allocScratchSlots(slots int) int {
	base := l.localSlots
	l.localSlots += slots
	return base
}

func (l *lowerer) lowerUnusedCopyLet(name string, info semantics.LocalInfo, expr frontend.Expr, pos frontend.Position) (bool, int, error) {
	if !l.stackAllocationLowering || info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 1 {
		return false, 0, nil
	}
	if _, ok := freshCopyBuiltinElement(call.Name); !ok {
		return false, 0, nil
	}
	alloc, ok := l.allocationPlan[name]
	if !ok || alloc.ActualLoweringStorage != allocplan.StorageEliminated || alloc.LoweringStatus != "eliminated_unused_copy" {
		return false, 0, nil
	}
	sourceSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return false, 0, err
	}
	if sourceSlots != 2 {
		return false, 0, fmt.Errorf("%s: %s expects one view source argument", frontend.FormatPos(pos), call.Name)
	}
	srcLen := l.allocScratchSlots(1)
	srcPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	return true, 2, nil
}

func (l *lowerer) lowerScalarReplacementLet(name string, info semantics.LocalInfo, expr frontend.Expr, pos frontend.Position) (bool, int, error) {
	if !l.stackAllocationLowering || info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 1 {
		return false, 0, nil
	}
	elem, isMake := stackAllocationElementByBuiltin(call.Name)
	if !isMake {
		var ok bool
		elem, ok = freshCopyBuiltinElement(call.Name)
		if !ok {
			return false, 0, nil
		}
	}
	isCopy := !isMake
	alloc, ok := l.allocationPlan[name]
	if !ok || alloc.ActualLoweringStorage != allocplan.StorageEliminated || alloc.LoweringStatus != "scalar_replacement" {
		return false, 0, nil
	}
	if alloc.ElementSize <= 0 || alloc.ByteSize <= 0 || alloc.ByteSize%alloc.ElementSize != 0 {
		return false, 0, nil
	}
	length := int64(alloc.ByteSize / alloc.ElementSize)
	if isMake {
		var known bool
		length, known = evalConstInt64ForAllocation(call.Args[0])
		if !known {
			return false, 0, nil
		}
	}
	if length <= 0 || length > int64(alloc.ByteSize) {
		return false, 0, nil
	}
	if alloc.ElementSize <= 0 || int(length)*alloc.ElementSize != alloc.ByteSize {
		return false, 0, nil
	}
	loadKind, ok := lowerIndexLoadKind(elem, l.types)
	if !ok {
		return false, 0, nil
	}
	srcPtr, srcLen := -1, -1
	if isCopy {
		sourceSlots, err := l.lowerExpr(call.Args[0])
		if err != nil {
			return false, 0, err
		}
		if sourceSlots != 2 {
			return false, 0, fmt.Errorf("%s: %s expects one view source argument", frontend.FormatPos(pos), call.Name)
		}
		srcLen = l.allocScratchSlots(1)
		srcPtr = l.allocScratchSlots(1)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})
	}
	elementBase := l.allocScratchSlots(int(length))
	for i := int64(0); i < length; i++ {
		if isCopy {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcPtr, Pos: pos})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(i), Pos: pos})
			l.emit(ir.IRInstr{Kind: loadKind, Pos: pos})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: elementBase + int(i), Pos: pos})
	}
	l.scalarSlices[name] = scalarSliceLocal{
		elemType:    elem,
		length:      length,
		elementBase: elementBase,
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(length), Pos: pos})
	return true, 2, nil
}

func (l *lowerer) lowerScalarIndexStore(index *frontend.IndexExpr, value frontend.Expr, pos frontend.Position) (bool, error) {
	meta, indexValue, ok, err := l.scalarSliceIndex(index)
	if err != nil || !ok {
		return ok, err
	}
	slots, err := l.lowerExprAs(value, meta.elemType)
	if err != nil {
		return true, err
	}
	if slots != 1 {
		return true, fmt.Errorf("%s: scalar-replaced slice store expects single-slot element", frontend.FormatPos(pos))
	}
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: meta.elementBase + int(indexValue), Pos: pos})
	return true, nil
}

func (l *lowerer) lowerScalarIndexLoad(index *frontend.IndexExpr) (bool, int, error) {
	meta, indexValue, ok, err := l.scalarSliceIndex(index)
	if err != nil || !ok {
		return ok, 0, err
	}
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: meta.elementBase + int(indexValue), Pos: index.At})
	return true, 1, nil
}

func (l *lowerer) scalarSliceIndex(index *frontend.IndexExpr) (scalarSliceLocal, int64, bool, error) {
	if index == nil {
		return scalarSliceLocal{}, 0, false, nil
	}
	base, ok := index.Base.(*frontend.IdentExpr)
	if !ok || base == nil {
		return scalarSliceLocal{}, 0, false, nil
	}
	meta, ok := l.scalarSlices[base.Name]
	if !ok {
		return scalarSliceLocal{}, 0, false, nil
	}
	indexValue, known := evalConstInt64ForAllocation(index.Index)
	if !known {
		return scalarSliceLocal{}, 0, true, fmt.Errorf("%s: scalar-replaced slice '%s' has non-constant index after allocation planning", frontend.FormatPos(index.At), base.Name)
	}
	if indexValue < 0 || indexValue >= meta.length {
		return scalarSliceLocal{}, 0, true, fmt.Errorf("%s: scalar-replaced slice '%s' has out-of-range constant index %d", frontend.FormatPos(index.At), base.Name, indexValue)
	}
	return meta, indexValue, true, nil
}

func (l *lowerer) lowerFunctionTempRegionCopyLet(name string, info semantics.LocalInfo, expr frontend.Expr, pos frontend.Position) (bool, int, error) {
	if !l.functionTempRegionLowering || info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 1 {
		return false, 0, nil
	}
	elem, ok := copyBuiltinElement(call.Name)
	if !ok {
		return false, 0, nil
	}
	alloc, ok := l.allocationPlan[name]
	if !ok || alloc.ActualLoweringStorage != allocplan.StorageFunctionTempRegion {
		return false, 0, nil
	}
	_, loadKind, storeKind, ok := copyElementIRKinds(elem, l.types)
	if !ok {
		return false, 0, nil
	}
	regionKind, ok := regionSliceKindByElement(elem)
	if !ok {
		return false, 0, nil
	}
	sourceSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return false, 0, err
	}
	if sourceSlots != 2 {
		return false, 0, fmt.Errorf("%s: %s expects one view source argument", frontend.FormatPos(pos), call.Name)
	}
	srcLen := l.allocScratchSlots(1)
	srcPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})

	l.ensureFunctionTempRegion(pos)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: regionKind, Name: name, Pos: pos})
	dstLen := l.allocScratchSlots(1)
	dstPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstPtr, Pos: pos})

	l.emitCopyLoop(srcPtr, srcLen, dstPtr, dstLen, loadKind, storeKind, copyLoopBoundsProofID(call.Name, call.At), pos)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstLen, Pos: pos})
	return true, 2, nil
}

func (l *lowerer) lowerExplicitIslandAllocationLet(name string, info semantics.LocalInfo, expr frontend.Expr, pos frontend.Position) (bool, int, error) {
	if info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 2 {
		return false, 0, nil
	}
	kind, ok := islandSliceKindByBuiltin(call.Name)
	if !ok {
		return false, 0, nil
	}
	alloc, ok := l.allocationPlan[name]
	if !ok || alloc.ActualLoweringStorage != allocplan.StorageExplicitIsland {
		return false, 0, nil
	}
	islandSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return false, 0, err
	}
	if islandSlots != 1 {
		return false, 0, fmt.Errorf("%s: %s expects island handle argument", frontend.FormatPos(pos), call.Name)
	}
	lengthSlots, err := l.lowerExpr(call.Args[1])
	if err != nil {
		return false, 0, err
	}
	if lengthSlots != 1 {
		return false, 0, fmt.Errorf("%s: %s expects length argument", frontend.FormatPos(pos), call.Name)
	}
	l.emit(ir.IRInstr{Kind: kind, Name: name, Pos: pos})
	return true, 2, nil
}

func (l *lowerer) ensureFunctionTempRegion(pos frontend.Position) {
	if l.functionTempRegionEntered {
		return
	}
	l.emit(ir.IRInstr{Kind: ir.IRRegionEnter, Pos: pos})
	l.functionTempRegionEntered = true
}

func (l *lowerer) emitFunctionTempRegionReset(pos frontend.Position) {
	if !l.functionTempRegionEntered {
		return
	}
	l.emit(ir.IRInstr{Kind: ir.IRRegionReset, Pos: pos})
}

func (l *lowerer) lowerStackCopyLet(name string, info semantics.LocalInfo, expr frontend.Expr, pos frontend.Position) (bool, int, error) {
	if !l.stackAllocationLowering || info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 1 {
		return false, 0, nil
	}
	elem, ok := copyBuiltinElement(call.Name)
	if !ok {
		return false, 0, nil
	}
	alloc, ok := l.allocationPlan[name]
	if !ok || alloc.ActualLoweringStorage != allocplan.StorageStack || alloc.ByteSize <= 0 || alloc.ElementSize <= 0 {
		return false, 0, nil
	}
	_, loadKind, storeKind, ok := copyElementIRKinds(elem, l.types)
	if !ok {
		return false, 0, nil
	}
	stackKind, ok := stackSliceKindByElement(elem)
	if !ok {
		return false, 0, nil
	}
	sourceSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return false, 0, err
	}
	if sourceSlots != 2 {
		return false, 0, fmt.Errorf("%s: %s expects one view source argument", frontend.FormatPos(pos), call.Name)
	}
	srcLen := l.allocScratchSlots(1)
	srcPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})

	backingSlots := (alloc.ByteSize + 7) / 8
	backingBase := l.allocScratchSlots(backingSlots)
	logicalLen := alloc.ByteSize / alloc.ElementSize
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: stackKind, Local: backingBase, ArgSlots: backingSlots, Imm: int32(logicalLen), Name: name, Pos: pos})
	dstLen := l.allocScratchSlots(1)
	dstPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstPtr, Pos: pos})

	l.emitCopyLoop(srcPtr, srcLen, dstPtr, dstLen, loadKind, storeKind, copyLoopBoundsProofID(name, pos), pos)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstLen, Pos: pos})
	return true, 2, nil
}

func (l *lowerer) lowerStackAllocationLet(name string, info semantics.LocalInfo, expr frontend.Expr, pos frontend.Position) (bool, int, error) {
	if !l.stackAllocationLowering || info.SlotCount != 2 {
		return false, 0, nil
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false, 0, nil
	}
	call = lowerCallExprWithBuiltinAlias(call)
	if len(call.Args) != 1 {
		return false, 0, nil
	}
	alloc, ok := l.allocationPlan[name]
	if !ok {
		return false, 0, nil
	}
	if alloc.ActualLoweringStorage != allocplan.StorageStack && alloc.ActualLoweringStorage != allocplan.StorageEliminated {
		return false, 0, nil
	}
	length, known := l.proofConstIntValue(call.Args[0])
	if !known {
		return false, 0, nil
	}
	kind, ok := stackSliceKindByBuiltin(call.Name)
	if !ok {
		return false, 0, nil
	}
	lengthSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return false, 0, err
	}
	if lengthSlots != 1 {
		return false, 0, fmt.Errorf("%s: allocation length must be i32", frontend.FormatPos(pos))
	}
	if alloc.ActualLoweringStorage == allocplan.StorageEliminated {
		if length != 0 {
			return false, 0, fmt.Errorf("%s: eliminated allocation %q has non-zero length %d", frontend.FormatPos(pos), name, length)
		}
		l.emit(ir.IRInstr{Kind: kind, Local: -1, ArgSlots: 0, Imm: 0, Name: name, Pos: pos})
		return true, 2, nil
	}
	if length <= 0 || alloc.ByteSize <= 0 {
		return false, 0, nil
	}
	backingSlots := (alloc.ByteSize + 7) / 8
	backingBase := l.allocScratchSlots(backingSlots)
	l.emit(ir.IRInstr{Kind: kind, Local: backingBase, ArgSlots: backingSlots, Imm: int32(length), Name: name, Pos: pos})
	return true, 2, nil
}

func stackSliceKindByBuiltin(name string) (ir.IRInstrKind, bool) {
	return lowerlets.StackSliceKindByBuiltin(name)
}

func stackAllocationElementByBuiltin(name string) (string, bool) {
	return lowerlets.StackAllocationElementByBuiltin(name)
}

func stackSliceKindByElement(elem string) (ir.IRInstrKind, bool) {
	return lowerlets.StackSliceKindByElement(elem)
}

func regionSliceKindByElement(elem string) (ir.IRInstrKind, bool) {
	return lowerlets.RegionSliceKindByElement(elem)
}

func islandSliceKindByBuiltin(name string) (ir.IRInstrKind, bool) {
	return lowerlets.IslandSliceKindByBuiltin(name)
}

func (l *lowerer) allocationNameForBuiltinCall(name string, pos frontend.Position, storage allocplan.StorageClass) string {
	if len(l.allocationPlan) == 0 {
		return ""
	}
	source := frontend.FormatPos(pos)
	for id, alloc := range l.allocationPlan {
		if alloc.Builtin == name && alloc.Source == source && alloc.ActualLoweringStorage == storage {
			return id
		}
	}
	return ""
}

func (l *lowerer) lowerMatchExpr(e *frontend.MatchExpr) (int, error) {
	info, ok := l.locals[e.ScrutineeLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown match expression scrutinee local", frontend.FormatPos(e.At))
	}
	resultInfo, ok := l.locals[e.ResultLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown match expression result local", frontend.FormatPos(e.At))
	}
	valueSlots, err := l.lowerExpr(e.Value)
	if err != nil {
		return 0, err
	}
	if valueSlots != info.SlotCount {
		return 0, fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(e.At))
	}
	for i := info.SlotCount - 1; i >= 0; i-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: e.At})
	}
	endLabel := l.newLabel()
	defaultLabel := -1
	caseLabels := make([]int, len(e.Cases))
	guardFailLabels := make([]int, len(e.Cases))
	scrutTypeInfo, scrutTypeOK := l.types[info.TypeName]
	for i, c := range e.Cases {
		guardFailLabels[i] = endLabel
		caseLabels[i] = l.newLabel()
		if c.Default {
			defaultLabel = caseLabels[i]
			continue
		}
		nextLabel := l.newLabel()
		guardFailLabels[i] = nextLabel
		if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeOptional {
			if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
				continue
			}
			if !isNoneExpr(c.Pattern) {
				return 0, fmt.Errorf("%s: optional match supports only 'none', 'some(name)', and '_' patterns", frontend.FormatPos(c.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: c.At})
		} else if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeEnum {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
			switch pat := c.Pattern.(type) {
			case *frontend.FieldAccessExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum match pattern was not resolved", frontend.FormatPos(c.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			case *frontend.EnumCasePatternExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum match pattern was not resolved", frontend.FormatPos(c.At))
				}
				if err := l.validateEnumPatternLayout(pat, info); err != nil {
					return 0, err
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			default:
				return 0, fmt.Errorf("%s: enum match supports enum case patterns and '_'", frontend.FormatPos(c.At))
			}
		} else {
			if info.SlotCount != 1 {
				return 0, fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
			patSlots, err := l.lowerExpr(c.Pattern)
			if err != nil {
				return 0, err
			}
			if patSlots != 1 {
				return 0, fmt.Errorf("%s: match pattern slot mismatch", frontend.FormatPos(c.At))
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
	}
	if defaultLabel >= 0 {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: defaultLabel, Pos: e.At})
	} else {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
	}
	for i, c := range e.Cases {
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: caseLabels[i], Pos: c.At})
		if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			bindInfo, ok := l.locals[some.Name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown some binding '%s'", frontend.FormatPos(some.At), some.Name)
			}
			if bindInfo.SlotCount != info.SlotCount-1 {
				return 0, fmt.Errorf("%s: optional some binding slot mismatch", frontend.FormatPos(some.At))
			}
			for slot := 0; slot < bindInfo.SlotCount; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + slot, Pos: some.At})
			}
			for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: some.At})
			}
		}
		if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
			if err := l.emitIfLetPatternBindings(enumPat, info); err != nil {
				return 0, err
			}
		}
		if c.Guard != nil {
			slots, err := l.lowerExpr(c.Guard)
			if err != nil {
				return 0, err
			}
			if slots != 1 {
				return 0, fmt.Errorf("%s: match guard must be single-slot", frontend.FormatPos(c.Guard.Pos()))
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: guardFailLabels[i], Pos: c.Guard.Pos()})
		}
		slots, err := l.lowerExprAs(c.Value, e.ResultType)
		if err != nil {
			return 0, err
		}
		if slots != resultInfo.SlotCount {
			return 0, fmt.Errorf("%s: match expression result slot mismatch", frontend.FormatPos(c.At))
		}
		for slot := resultInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultInfo.Base + slot, Pos: c.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: c.At})
	}
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
	for slot := 0; slot < resultInfo.SlotCount; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultInfo.Base + slot, Pos: e.At})
	}
	return resultInfo.SlotCount, nil
}

func (l *lowerer) lowerCatchExpr(e *frontend.CatchExpr) (int, error) {
	call, ok := e.Call.(*frontend.CallExpr)
	if !ok {
		return 0, fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
	}
	errorInfo, ok := l.locals[e.ErrorLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown catch error local", frontend.FormatPos(e.At))
	}
	resultInfo, ok := l.locals[e.ResultLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown catch result local", frontend.FormatPos(e.At))
	}
	call = lowerCallExprWithBuiltinAlias(call)
	var callSuccessSlots int
	var callErrorSlots int
	var callCompact bool
	var expectedReturnSlots int
	if isTypedTaskJoinCall(call.Name) {
		if len(call.TypeArgs) != 1 || call.TypeArgs[0].Name == "" {
			return 0, fmt.Errorf("%s: task_join_i32_typed missing resolved error type", frontend.FormatPos(call.At))
		}
		errorInfo, ok := l.types[call.TypeArgs[0].Name]
		if !ok || errorInfo.Kind != semantics.TypeEnum {
			return 0, fmt.Errorf("%s: typed task error argument must be an enum", frontend.FormatPos(call.TypeArgs[0].At))
		}
		_, handleInfo, err := semantics.EnsureTypedTaskHandleType(call.TypeArgs[0].Name, l.types)
		if err != nil {
			return 0, fmt.Errorf("%s: %v", frontend.FormatPos(call.TypeArgs[0].At), err)
		}
		callSuccessSlots = 1
		callErrorSlots = errorInfo.SlotCount
		callCompact = errorInfo.SlotCount == 1
		expectedReturnSlots = handleInfo.SlotCount
	} else {
		sig, ok := l.funcs[call.Name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
		}
		if sig.ThrowsType == "" {
			return 0, fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
		}
		var err error
		callSuccessSlots, callErrorSlots, callCompact, err = throwingLayout(sig.ReturnType, sig.ThrowsType, l.types)
		if err != nil {
			return 0, err
		}
		expectedReturnSlots = sig.ReturnSlots
	}
	if callSuccessSlots != resultInfo.SlotCount || callErrorSlots != errorInfo.SlotCount {
		return 0, fmt.Errorf("%s: catch slot mismatch", frontend.FormatPos(e.At))
	}
	var slots int
	var err error
	if isTypedTaskJoinCall(call.Name) {
		slots, err = l.lowerTypedTaskJoinForCatch(call, e.At)
	} else {
		slots, err = l.lowerExpr(call)
	}
	if err != nil {
		return 0, err
	}
	if slots != expectedReturnSlots {
		return 0, fmt.Errorf("%s: catch call result slot mismatch", frontend.FormatPos(e.At))
	}

	successLabel := l.newLabel()
	endLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: successLabel, Pos: e.At})

	if callCompact {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: errorInfo.Base, Pos: e.At})
	} else {
		for slot := callErrorSlots - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: errorInfo.Base + slot, Pos: e.At})
		}
		discard := l.ensureDiscardLocal()
		for slot := 0; slot < callSuccessSlots; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
		}
	}

	defaultLabel := -1
	caseLabels := make([]int, len(e.Cases))
	guardFailLabels := make([]int, len(e.Cases))
	errorTypeInfo, errorTypeOK := l.types[errorInfo.TypeName]
	for i, c := range e.Cases {
		guardFailLabels[i] = endLabel
		caseLabels[i] = l.newLabel()
		if c.Default {
			defaultLabel = caseLabels[i]
			continue
		}
		nextLabel := l.newLabel()
		guardFailLabels[i] = nextLabel
		if errorTypeOK && errorTypeInfo.Kind == semantics.TypeOptional {
			if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errorInfo.Base + errorInfo.SlotCount - 1, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
				continue
			}
			if !isNoneExpr(c.Pattern) {
				return 0, fmt.Errorf("%s: optional catch supports only 'none', 'some(name)', and '_' patterns", frontend.FormatPos(c.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errorInfo.Base + errorInfo.SlotCount - 1, Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: c.At})
		} else if errorTypeOK && errorTypeInfo.Kind == semantics.TypeEnum {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errorInfo.Base, Pos: c.At})
			switch pat := c.Pattern.(type) {
			case *frontend.FieldAccessExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum catch pattern was not resolved", frontend.FormatPos(c.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			case *frontend.EnumCasePatternExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum catch pattern was not resolved", frontend.FormatPos(c.At))
				}
				if err := l.validateEnumPatternLayout(pat, errorInfo); err != nil {
					return 0, err
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			default:
				return 0, fmt.Errorf("%s: enum catch supports enum case patterns and '_'", frontend.FormatPos(c.At))
			}
		} else {
			if errorInfo.SlotCount != 1 {
				return 0, fmt.Errorf("%s: catch error slot mismatch", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errorInfo.Base, Pos: c.At})
			patSlots, err := l.lowerExpr(c.Pattern)
			if err != nil {
				return 0, err
			}
			if patSlots != 1 {
				return 0, fmt.Errorf("%s: catch pattern slot mismatch", frontend.FormatPos(c.At))
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
	}
	if defaultLabel >= 0 {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: defaultLabel, Pos: e.At})
	} else {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
	}
	for i, c := range e.Cases {
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: caseLabels[i], Pos: c.At})
		if err := l.emitIfLetPatternBindings(c.Pattern, errorInfo); err != nil {
			return 0, err
		}
		if c.Guard != nil {
			slots, err := l.lowerExpr(c.Guard)
			if err != nil {
				return 0, err
			}
			if slots != 1 {
				return 0, fmt.Errorf("%s: catch guard must be single-slot", frontend.FormatPos(c.Guard.Pos()))
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: guardFailLabels[i], Pos: c.Guard.Pos()})
		}
		slots, err := l.lowerExprAs(c.Value, e.ResultType)
		if err != nil {
			return 0, err
		}
		if slots != resultInfo.SlotCount {
			return 0, fmt.Errorf("%s: catch expression result slot mismatch", frontend.FormatPos(c.At))
		}
		for slot := resultInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultInfo.Base + slot, Pos: c.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: c.At})
	}

	successEntrySlots := callSuccessSlots
	if !callCompact {
		successEntrySlots += callErrorSlots
	}
	l.emitZeroSlots(successEntrySlots, e.At)
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: successLabel, Pos: e.At})
	if !callCompact {
		discard := l.ensureDiscardLocal()
		for slot := 0; slot < callErrorSlots; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
		}
	}
	for slot := resultInfo.SlotCount - 1; slot >= 0; slot-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultInfo.Base + slot, Pos: e.At})
	}
	l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
	for slot := 0; slot < resultInfo.SlotCount; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultInfo.Base + slot, Pos: e.At})
	}
	return resultInfo.SlotCount, nil
}

func (l *lowerer) emitIfLetPatternCheck(pattern frontend.Expr, valueInfo semantics.LocalInfo, elseLabel int, pos frontend.Position) error {
	scrutTypeInfo, scrutTypeOK := l.types[valueInfo.TypeName]
	if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeOptional {
		if _, ok := pattern.(*frontend.SomePatternExpr); ok {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + valueInfo.SlotCount - 1, Pos: pos})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: pos})
			return nil
		}
		if !isNoneExpr(pattern) {
			return fmt.Errorf("%s: optional if let supports only 'none' and 'some(name)' patterns", frontend.FormatPos(pos))
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + valueInfo.SlotCount - 1, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: pos})
		return nil
	}
	if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeEnum {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base, Pos: pos})
		switch pat := pattern.(type) {
		case *frontend.FieldAccessExpr:
			if pat.EnumType == "" {
				return fmt.Errorf("%s: enum if-let pattern was not resolved", frontend.FormatPos(pos))
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: pos})
		case *frontend.EnumCasePatternExpr:
			if pat.EnumType == "" {
				return fmt.Errorf("%s: enum if-let pattern was not resolved", frontend.FormatPos(pos))
			}
			if err := l.validateEnumPatternLayout(pat, valueInfo); err != nil {
				return err
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: pos})
		default:
			return fmt.Errorf("%s: enum if let supports enum case patterns", frontend.FormatPos(pos))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: pos})
		return nil
	}
	return fmt.Errorf("%s: if let pattern requires optional or enum value", frontend.FormatPos(pos))
}

func enumPayloadSlotCount(pat *frontend.EnumCasePatternExpr, fallbackBindings map[string]semantics.LocalInfo) (int, error) {
	if pat == nil {
		return 0, nil
	}
	if len(pat.PayloadSlots) > 0 {
		if len(pat.PayloadSlots) != len(pat.Bindings) {
			return 0, fmt.Errorf("%s: enum payload pattern slot metadata mismatch", frontend.FormatPos(pat.At))
		}
		total := 0
		for _, slots := range pat.PayloadSlots {
			if slots <= 0 {
				return 0, fmt.Errorf("%s: enum payload pattern slot metadata mismatch", frontend.FormatPos(pat.At))
			}
			total += slots
		}
		return total, nil
	}
	total := 0
	for _, binding := range pat.Bindings {
		bindInfo, ok := fallbackBindings[binding]
		if !ok {
			return 0, fmt.Errorf("%s: unknown enum payload binding '%s'", frontend.FormatPos(pat.At), binding)
		}
		if bindInfo.SlotCount <= 0 {
			return 0, fmt.Errorf("%s: enum payload binding '%s' slot mismatch", frontend.FormatPos(pat.At), binding)
		}
		total += bindInfo.SlotCount
	}
	return total, nil
}

func (l *lowerer) validateEnumPatternLayout(pattern frontend.Expr, valueInfo semantics.LocalInfo) error {
	enumPat, ok := pattern.(*frontend.EnumCasePatternExpr)
	if !ok {
		return nil
	}
	payloadSlots, err := enumPayloadSlotCount(enumPat, l.locals)
	if err != nil {
		return err
	}
	if payloadSlots > valueInfo.SlotCount-1 {
		return fmt.Errorf("%s: enum payload pattern exceeds value layout", frontend.FormatPos(enumPat.At))
	}
	return nil
}

func (l *lowerer) emitIfLetPatternBindings(pattern frontend.Expr, valueInfo semantics.LocalInfo) error {
	if some, ok := pattern.(*frontend.SomePatternExpr); ok {
		bindInfo, ok := l.locals[some.Name]
		if !ok {
			return fmt.Errorf("%s: unknown some binding '%s'", frontend.FormatPos(some.At), some.Name)
		}
		for slot := 0; slot < bindInfo.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + slot, Pos: some.At})
		}
		for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: some.At})
		}
	}
	if enumPat, ok := pattern.(*frontend.EnumCasePatternExpr); ok {
		payloadOffset := 1
		for i, binding := range enumPat.Bindings {
			bindInfo, ok := l.locals[binding]
			if !ok {
				return fmt.Errorf("%s: unknown enum payload binding '%s'", frontend.FormatPos(enumPat.At), binding)
			}
			wantSlots := bindInfo.SlotCount
			if i < len(enumPat.PayloadSlots) {
				wantSlots = enumPat.PayloadSlots[i]
			}
			if bindInfo.SlotCount != wantSlots {
				return fmt.Errorf("%s: enum payload binding '%s' slot mismatch", frontend.FormatPos(enumPat.At), binding)
			}
			for slot := 0; slot < bindInfo.SlotCount; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + payloadOffset + slot, Pos: enumPat.At})
			}
			for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: enumPat.At})
			}
			payloadOffset += wantSlots
		}
	}
	return nil
}

func rawPtrAddCall(expr frontend.Expr) (*frontend.CallExpr, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok {
		return nil, false
	}
	name := call.Name
	if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = builtin
	}
	if name != "core.ptr_add" {
		return nil, false
	}
	return call, true
}

func (l *lowerer) rawPtrOffsetAliasFromExpr(expr frontend.Expr) (rawPtrOffsetLocal, bool) {
	call, ok := rawPtrAddCall(expr)
	if !ok || len(call.Args) != 3 {
		return rawPtrOffsetLocal{}, false
	}
	base, ok := call.Args[0].(*frontend.IdentExpr)
	if !ok {
		return rawPtrOffsetLocal{}, false
	}
	baseInfo, ok := l.locals[base.Name]
	if !ok || baseInfo.SlotCount != 1 {
		return rawPtrOffsetLocal{}, false
	}
	alias := rawPtrOffsetLocal{BaseLocal: baseInfo.Base, OffsetLocal: -1}
	if prior, ok := l.rawPtrOffsetLocals[baseInfo.Base]; ok {
		alias = prior
	}
	switch offset := call.Args[1].(type) {
	case *frontend.NumberExpr:
		if alias.HasOffsetImm {
			alias.OffsetImm += offset.Value
		} else if alias.OffsetLocal < 0 {
			alias.OffsetImm = offset.Value
			alias.HasOffsetImm = true
		} else {
			return rawPtrOffsetLocal{}, false
		}
	case *frontend.IdentExpr:
		if alias.HasOffsetImm || alias.OffsetLocal >= 0 {
			return rawPtrOffsetLocal{}, false
		}
		offsetInfo, ok := l.locals[offset.Name]
		if !ok || offsetInfo.SlotCount != 1 {
			return rawPtrOffsetLocal{}, false
		}
		alias.OffsetLocal = offsetInfo.Base
	default:
		return rawPtrOffsetLocal{}, false
	}
	return alias, true
}

func (l *lowerer) rememberRawPtrOffsetAlias(local int, expr frontend.Expr) {
	l.clearRawPtrOffsetAliasesForLocal(local)
	alias, ok := l.rawPtrOffsetAliasFromExpr(expr)
	if !ok {
		delete(l.rawPtrOffsetLocals, local)
		return
	}
	l.rawPtrOffsetLocals[local] = alias
}

func (l *lowerer) clearRawPtrOffsetAliasesForLocal(local int) {
	delete(l.rawPtrOffsetLocals, local)
	for aliasLocal, alias := range l.rawPtrOffsetLocals {
		if alias.BaseLocal == local || (!alias.HasOffsetImm && alias.OffsetLocal == local) {
			delete(l.rawPtrOffsetLocals, aliasLocal)
		}
	}
}

func (l *lowerer) lowerRawOffsetAlias(alias rawPtrOffsetLocal, pos frontend.Position) {
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: alias.BaseLocal, Pos: pos})
	if alias.HasOffsetImm {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: alias.OffsetImm, Pos: pos})
		return
	}
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: alias.OffsetLocal, Pos: pos})
}

func (l *lowerer) lowerRawOffsetAddress(expr frontend.Expr, pos frontend.Position) (bool, error) {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if info, ok := l.locals[id.Name]; ok {
			if alias, ok := l.rawPtrOffsetLocals[info.Base]; ok {
				l.lowerRawOffsetAlias(alias, pos)
				return true, nil
			}
		}
	}
	call, ok := rawPtrAddCall(expr)
	if !ok {
		return false, nil
	}
	if len(call.Args) != 3 {
		return true, fmt.Errorf("%s: ptr_add expects 3 arguments", frontend.FormatPos(call.At))
	}
	baseSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return true, err
	}
	if baseSlots != 1 {
		return true, fmt.Errorf("%s: ptr_add expects a 1-slot base pointer", frontend.FormatPos(call.Args[0].Pos()))
	}
	offsetSlots, err := l.lowerExpr(call.Args[1])
	if err != nil {
		return true, err
	}
	if offsetSlots != 1 {
		return true, fmt.Errorf("%s: ptr_add expects a 1-slot offset", frontend.FormatPos(call.Args[1].Pos()))
	}
	memSlots, err := l.lowerExpr(call.Args[2])
	if err != nil {
		return true, err
	}
	if memSlots != 1 {
		return true, fmt.Errorf("%s: ptr_add expects a 1-slot memory capability", frontend.FormatPos(call.Args[2].Pos()))
	}
	discard := l.ensureDiscardLocal()
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	return true, nil
}

func (l *lowerer) lowerSurfaceRuntimeCall(e *frontend.CallExpr, runtimeName string, expectedArgSlots int) (int, error) {
	total := 0
	for _, arg := range e.Args {
		slots, err := l.lowerExpr(arg)
		if err != nil {
			return 0, err
		}
		total += slots
	}
	if total != expectedArgSlots {
		return 0, fmt.Errorf("%s: %s lowered %d argument slots, want %d", frontend.FormatPos(e.At), e.Name, total, expectedArgSlots)
	}
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: runtimeName, ArgSlots: total, RetSlots: 1, Pos: e.At})
	return 1, nil
}

func (l *lowerer) lowerRawOffsetCall(e *frontend.CallExpr) (int, bool, error) {
	switch e.Name {
	case "core.load_i32", "core.load_u8", "core.load_ptr":
		if len(e.Args) != 2 {
			return 0, true, fmt.Errorf("%s: %s expects 2 arguments", frontend.FormatPos(e.At), strings.TrimPrefix(e.Name, "core."))
		}
		ok, err := l.lowerRawOffsetAddress(e.Args[0], e.At)
		if err != nil || !ok {
			return 0, ok, err
		}
		memSlots, err := l.lowerExpr(e.Args[1])
		if err != nil {
			return 0, true, err
		}
		if memSlots != 1 {
			return 0, true, fmt.Errorf("%s: %s expects a 1-slot memory capability", frontend.FormatPos(e.Args[1].Pos()), strings.TrimPrefix(e.Name, "core."))
		}
		switch e.Name {
		case "core.load_i32":
			l.emit(ir.IRInstr{Kind: ir.IRMemReadI32Offset, Pos: e.At})
		case "core.load_u8":
			l.emit(ir.IRInstr{Kind: ir.IRMemReadU8Offset, Pos: e.At})
		default:
			l.emit(ir.IRInstr{Kind: ir.IRMemReadPtrOffset, Pos: e.At})
		}
		return 1, true, nil
	case "core.store_i32", "core.store_u8", "core.store_ptr", "core.store_arch_ptr":
		if len(e.Args) != 3 {
			return 0, true, fmt.Errorf("%s: %s expects 3 arguments", frontend.FormatPos(e.At), strings.TrimPrefix(e.Name, "core."))
		}
		ok, err := l.lowerRawOffsetAddress(e.Args[0], e.At)
		if err != nil || !ok {
			return 0, ok, err
		}
		valueSlots, err := l.lowerExpr(e.Args[1])
		if err != nil {
			return 0, true, err
		}
		if valueSlots != 1 {
			return 0, true, fmt.Errorf("%s: %s expects a 1-slot value", frontend.FormatPos(e.Args[1].Pos()), strings.TrimPrefix(e.Name, "core."))
		}
		memSlots, err := l.lowerExpr(e.Args[2])
		if err != nil {
			return 0, true, err
		}
		if memSlots != 1 {
			return 0, true, fmt.Errorf("%s: %s expects a 1-slot memory capability", frontend.FormatPos(e.Args[2].Pos()), strings.TrimPrefix(e.Name, "core."))
		}
		switch e.Name {
		case "core.store_i32":
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteI32Offset, Pos: e.At})
		case "core.store_u8":
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteU8Offset, Pos: e.At})
		case "core.store_arch_ptr":
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteArchPtrOffset, Pos: e.At})
		default:
			l.emit(ir.IRInstr{Kind: ir.IRMemWritePtrOffset, Pos: e.At})
		}
		return 1, true, nil
	default:
		return 0, false, nil
	}
}

func (l *lowerer) lowerPtrAddValueCall(e *frontend.CallExpr) (int, bool, error) {
	if e.Name != "core.ptr_add" {
		return 0, false, nil
	}
	if len(e.Args) != 3 {
		return 0, true, fmt.Errorf("%s: ptr_add expects 3 arguments", frontend.FormatPos(e.At))
	}
	alias, ok := l.rawPtrOffsetAliasFromExpr(e)
	if !ok {
		return 0, false, nil
	}
	l.lowerRawOffsetAlias(alias, e.At)
	memSlots, err := l.lowerExpr(e.Args[2])
	if err != nil {
		return 0, true, err
	}
	if memSlots != 1 {
		return 0, true, fmt.Errorf("%s: ptr_add expects a 1-slot memory capability", frontend.FormatPos(e.Args[2].Pos()))
	}
	l.emit(ir.IRInstr{Kind: ir.IRPtrAdd, Pos: e.At})
	return 1, true, nil
}
