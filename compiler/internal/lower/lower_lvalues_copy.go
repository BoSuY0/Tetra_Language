package lower

import (
	"fmt"
	"strings"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowerexpressions "tetra_language/compiler/internal/lower/expressions"
	lowerlets "tetra_language/compiler/internal/lower/lets"
	"tetra_language/compiler/internal/semantics"
)

func (l *lowerer) emitGlobalStringLiteralInitIfNeeded(g semantics.GlobalInfo, pos frontend.Position) {
	if g.TypeName != "str" || !g.HasStringLiteralInit {
		return
	}
	readyLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: readyLabel, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStrLit, Str: g.StringLiteralInit, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex + 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: readyLabel, Pos: pos})
}

func (l *lowerer) emitGlobalArrayBackingsInitIfNeeded(g semantics.GlobalInfo, pos frontend.Position) {
	for _, backing := range g.ArrayBackings {
		byteLen := globalArrayBackingByteLen(backing.ElemType, backing.Len, l.types)
		if byteLen <= 0 {
			continue
		}
		ptrSlot := g.DataIndex + backing.HeaderOffset
		lenSlot := ptrSlot + 1
		readyLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: ptrSlot, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: readyLabel, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStrLit, Str: make([]byte, byteLen), Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.ensureDiscardLocal(), Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: ptrSlot, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(backing.Len), Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: lenSlot, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: readyLabel, Pos: pos})
	}
}

func globalArrayBackingByteLen(elemType string, n int, types map[string]*semantics.TypeInfo) int {
	if n <= 0 {
		return 0
	}
	switch elemType {
	case "u8":
		return n
	case "u16":
		return n * 2
	case "i32", "c_int", "c_uint", "bool",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return n * 4
	}
	if info, ok := types[elemType]; ok && info.Kind == semantics.TypeStruct && info.SlotCount == 1 {
		return n * 4
	}
	return 0
}

func (l *lowerer) emitGlobalFunctionValueInitIfNeeded(g semantics.GlobalInfo, pos frontend.Position) {
	if !g.FunctionTypeValue || !g.Mutable || g.FunctionValue == "" {
		return
	}
	readyLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: readyLabel, Pos: pos})
	slots := l.emitFunctionSymbolValue(g.FunctionValue, nil, pos)
	for i := slots - 1; i >= 0; i-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex + i, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: readyLabel, Pos: pos})
}

type lvalueInfo struct {
	Base      int
	SlotCount int
	TypeName  string
	Name      string
	Global    bool
}

func (l *lowerer) resolveLValue(expr frontend.Expr) (lvalueInfo, error) {
	baseName, fields, pos, ok := splitFieldPathLower(expr)
	if !ok {
		return lvalueInfo{}, fmt.Errorf("%s: invalid assignment target", frontend.FormatPos(pos))
	}
	info, ok := l.locals[baseName]
	if !ok {
		if g, ok := l.globals[baseName]; ok {
			targetType, slotCount, offset, err := resolveFieldChainLower(g.TypeName, g.DataIndex, fields, l.types, pos)
			if err != nil {
				return lvalueInfo{}, err
			}
			if _, ok := l.types[targetType]; !ok {
				return lvalueInfo{}, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), targetType)
			}
			return lvalueInfo{Base: offset, SlotCount: slotCount, TypeName: targetType, Name: baseName, Global: true}, nil
		}
		return lvalueInfo{}, fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(pos), baseName)
	}
	targetType, slotCount, offset, err := resolveFieldChainLower(info.TypeName, info.Base, fields, l.types, pos)
	if err != nil {
		return lvalueInfo{}, err
	}
	if _, ok := l.types[targetType]; !ok {
		return lvalueInfo{}, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), targetType)
	}
	return lvalueInfo{Base: offset, SlotCount: slotCount, TypeName: targetType, Name: baseName}, nil
}

func splitFieldPathLower(expr frontend.Expr) (string, []string, frontend.Position, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, nil, e.At, true
	case *frontend.FieldAccessExpr:
		baseName, fields, pos, ok := splitFieldPathLower(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, e.Field)
		return baseName, fields, e.At, true
	default:
		return "", nil, expr.Pos(), false
	}
}

func resolveFieldChainLower(typeName string, baseOffset int, fields []string, types map[string]*semantics.TypeInfo, pos frontend.Position) (string, int, int, error) {
	offset := baseOffset
	current := typeName
	for _, field := range fields {
		info, ok := types[current]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
		}
		if info.Kind != semantics.TypeStruct && info.Kind != semantics.TypeSlice && info.Kind != semantics.TypeArray && info.Kind != semantics.TypeStr {
			return "", 0, 0, fmt.Errorf("%s: '%s' is not a struct", frontend.FormatPos(pos), current)
		}
		fieldInfo, ok := info.FieldMap[field]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(pos), field)
		}
		offset += fieldInfo.Offset
		current = fieldInfo.TypeName
	}
	info, ok := types[current]
	if !ok {
		return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
	}
	return current, info.SlotCount, offset, nil
}

func isNoneExpr(expr frontend.Expr) bool {
	_, ok := expr.(*frontend.NoneLitExpr)
	return ok
}

func (l *lowerer) lowerOptionalTag(expr frontend.Expr) error {
	if isNoneExpr(expr) {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: expr.Pos()})
		return nil
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		info, ok := l.locals[e.Name]
		if !ok {
			return fmt.Errorf("%s: optional comparison to none requires a stored optional value", frontend.FormatPos(e.At))
		}
		typeInfo, ok := l.types[info.TypeName]
		if !ok || typeInfo.Kind != semantics.TypeOptional {
			return fmt.Errorf("%s: optional comparison to none requires optional value", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + typeInfo.SlotCount - 1, Pos: e.At})
		return nil
	case *frontend.FieldAccessExpr:
		target, err := l.resolveLValue(e)
		if err != nil {
			return err
		}
		tname, err := l.inferExprType(e)
		if err != nil {
			return err
		}
		typeInfo, ok := l.types[tname]
		if !ok || typeInfo.Kind != semantics.TypeOptional {
			return fmt.Errorf("%s: optional comparison to none requires optional value", frontend.FormatPos(e.At))
		}
		kind := ir.IRLoadLocal
		if target.Global {
			kind = ir.IRLoadGlobal
		}
		l.emit(ir.IRInstr{Kind: kind, Local: target.Base + typeInfo.SlotCount - 1, Pos: e.At})
		return nil
	default:
		return fmt.Errorf("%s: optional comparison to none requires a stored optional value", frontend.FormatPos(expr.Pos()))
	}
}

func (l *lowerer) indexElemType(base frontend.Expr) (string, error) {
	baseType, err := l.inferExprType(base)
	if err != nil {
		return "", err
	}
	info, ok := l.types[baseType]
	if !ok {
		return "", fmt.Errorf("unknown type '%s'", baseType)
	}
	switch info.Kind {
	case semantics.TypeStr:
		return "u8", nil
	case semantics.TypeSlice:
		return info.ElemType, nil
	case semantics.TypeArray:
		return info.ElemType, nil
	default:
		return "", fmt.Errorf("%s: cannot index '%s'", frontend.FormatPos(base.Pos()), baseType)
	}
}

func lowerIndexLoadKind(elemType string, types map[string]*semantics.TypeInfo) (ir.IRInstrKind, bool) {
	return lowerexpressions.IndexLoadKind(elemType, types)
}

func uncheckedIndexLoadKind(kind ir.IRInstrKind) ir.IRInstrKind {
	return lowerexpressions.UncheckedIndexLoadKind(kind)
}

func sliceViewElementShift(name string) (int32, bool) {
	if strings.HasPrefix(name, "core.string_") {
		return 0, true
	}
	parts := strings.Split(name, "_")
	if len(parts) == 0 {
		return 0, false
	}
	switch parts[len(parts)-1] {
	case "u8":
		return 0, true
	case "u16":
		return 1, true
	case "i32", "bool":
		return 2, true
	default:
		return 0, false
	}
}

func (l *lowerer) lowerCopyBuiltinFromStack(name string, total int, pos frontend.Position) (int, error) {
	if total != 2 {
		return 0, fmt.Errorf("%s: %s expects one view source argument", frontend.FormatPos(pos), name)
	}
	elem, ok := copyBuiltinElement(name)
	if !ok {
		return 0, lowerUnsupportedError(pos, "unsupported copy builtin '%s'", name)
	}
	makeKind, loadKind, storeKind, ok := copyElementIRKinds(elem, l.types)
	if !ok {
		return 0, lowerUnsupportedError(pos, "unsupported copy element type '%s'", elem)
	}
	srcLen := l.allocScratchSlots(1)
	srcPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: makeKind, Pos: pos})
	dstLen := l.allocScratchSlots(1)
	dstPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstPtr, Pos: pos})

	l.emitCopyLoop(srcPtr, srcLen, dstPtr, dstLen, loadKind, storeKind, copyLoopBoundsProofID(name, pos), pos)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstLen, Pos: pos})
	return 2, nil
}

func (l *lowerer) lowerCopyIntoBuiltinFromStack(name string, total int, pos frontend.Position) (int, error) {
	if total != 4 {
		return 0, fmt.Errorf("%s: %s expects source and destination view arguments", frontend.FormatPos(pos), name)
	}
	elem, ok := copyBuiltinElement(name)
	if !ok {
		return 0, lowerUnsupportedError(pos, "unsupported copy_into builtin '%s'", name)
	}
	_, loadKind, storeKind, ok := copyElementIRKinds(elem, l.types)
	if !ok {
		return 0, lowerUnsupportedError(pos, "unsupported copy_into element type '%s'", elem)
	}
	shift, ok := copyElementShift(elem)
	if !ok {
		return 0, lowerUnsupportedError(pos, "unsupported copy_into element shift for '%s'", elem)
	}
	dstLen := l.allocScratchSlots(1)
	dstPtr := l.allocScratchSlots(1)
	srcLen := l.allocScratchSlots(1)
	srcPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: srcPtr, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRSlicePrefix, Imm: shift, Pos: pos})
	checkedDstLen := l.allocScratchSlots(1)
	checkedDstPtr := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: checkedDstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: checkedDstPtr, Pos: pos})

	l.emitCopyLoop(srcPtr, srcLen, checkedDstPtr, checkedDstLen, loadKind, storeKind, copyLoopBoundsProofID(name, pos), pos)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	return 1, nil
}

func (l *lowerer) emitCopyLoop(srcPtr, srcLen, dstPtr, dstLen int, loadKind, storeKind ir.IRInstrKind, proofID string, pos frontend.Position) {
	index := l.allocScratchSlots(1)
	value := l.allocScratchSlots(1)
	startLabel := l.newLabel()
	endLabel := l.newLabel()

	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: index, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: index, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: srcLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: index, Pos: pos})
	if proofID != "" {
		l.emit(ir.IRInstr{Kind: uncheckedIndexLoadKind(loadKind), ProofID: proofID, Pos: pos})
	} else {
		l.emit(ir.IRInstr{Kind: loadKind, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: value, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstPtr, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: dstLen, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: index, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: value, Pos: pos})
	l.emit(ir.IRInstr{Kind: storeKind, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: index, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: index, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: pos})
}

func copyBuiltinElement(name string) (string, bool) {
	if name == "core.string_copy" || name == "core.string_copy_into" {
		return "u8", true
	}
	for _, prefix := range []string{"core.slice_copy_into_", "core.slice_copy_"} {
		if strings.HasPrefix(name, prefix) {
			elem := strings.TrimPrefix(name, prefix)
			switch elem {
			case "u8", "u16", "i32", "bool":
				return elem, true
			}
		}
	}
	return "", false
}

func freshCopyBuiltinElement(name string) (string, bool) {
	if name == "core.string_copy_into" || strings.HasPrefix(name, "core.slice_copy_into_") {
		return "", false
	}
	return copyBuiltinElement(name)
}

func copyElementIRKinds(elem string, types map[string]*semantics.TypeInfo) (ir.IRInstrKind, ir.IRInstrKind, ir.IRInstrKind, bool) {
	makeKind := ir.IRMakeSliceI32
	switch elem {
	case "u8":
		makeKind = ir.IRMakeSliceU8
	case "u16":
		makeKind = ir.IRMakeSliceU16
	case "i32", "bool":
		makeKind = ir.IRMakeSliceI32
	default:
		return 0, 0, 0, false
	}
	loadKind, ok := lowerIndexLoadKind(elem, types)
	if !ok {
		return 0, 0, 0, false
	}
	storeKind, ok := lowerIndexStoreKind(elem, types)
	if !ok {
		return 0, 0, 0, false
	}
	return makeKind, loadKind, storeKind, true
}

func copyElementShift(elem string) (int32, bool) {
	switch elem {
	case "u8":
		return 0, true
	case "u16":
		return 1, true
	case "i32", "bool":
		return 2, true
	default:
		return 0, false
	}
}

func staticInvalidCollectionIterable(expr frontend.Expr) bool {
	return staticInvalidAllocationIterable(expr) || staticInvalidStringViewIterable(expr)
}

func staticInvalidAllocationIterable(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	elemSize, ok := allocationElementSizeByBuiltin(name)
	if !ok {
		if target, aliasOK := semantics.ResolveBuiltinAlias(name); aliasOK {
			name = target
			elemSize, ok = allocationElementSizeByBuiltin(name)
		}
	}
	if !ok {
		return false
	}
	lengthArgIndex := 0
	if strings.HasPrefix(name, "core.island_make_") {
		lengthArgIndex = 1
	}
	if lengthArgIndex >= len(call.Args) {
		return false
	}
	length, known := evalConstInt64ForAllocation(call.Args[lengthArgIndex])
	if !known {
		return false
	}
	if length < 0 {
		return true
	}
	return elemSize > 0 && length*int64(elemSize) > 2147483647
}

func staticInvalidStringViewIterable(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	if target, aliasOK := semantics.ResolveBuiltinAlias(name); aliasOK {
		name = target
	}
	if !strings.HasPrefix(name, "core.string_") {
		return false
	}
	sourceLen, knownLen := staticStringByteLen(callArg(call, 0))
	if !knownLen {
		return false
	}
	switch name {
	case "core.string_window":
		if len(call.Args) != 3 {
			return false
		}
		start, startKnown := evalConstInt64ForAllocation(call.Args[1])
		count, countKnown := evalConstInt64ForAllocation(call.Args[2])
		if !startKnown || !countKnown {
			return false
		}
		return start < 0 || count < 0 || start > sourceLen || count > sourceLen-start
	case "core.string_prefix":
		if len(call.Args) != 2 {
			return false
		}
		count, known := evalConstInt64ForAllocation(call.Args[1])
		if !known {
			return false
		}
		return count < 0 || count > sourceLen
	case "core.string_suffix":
		if len(call.Args) != 2 {
			return false
		}
		start, known := evalConstInt64ForAllocation(call.Args[1])
		if !known {
			return false
		}
		return start < 0 || start > sourceLen
	default:
		return false
	}
}

func callArg(call *frontend.CallExpr, index int) frontend.Expr {
	if call == nil || index < 0 || index >= len(call.Args) {
		return nil
	}
	return call.Args[index]
}

func staticStringByteLen(expr frontend.Expr) (int64, bool) {
	lit, ok := expr.(*frontend.StringLitExpr)
	if !ok || lit == nil {
		return 0, false
	}
	return int64(len(lit.Value)), true
}

func allocationElementSizeByBuiltin(name string) (int, bool) {
	return lowerlets.AllocationElementSizeByBuiltin(name)
}

func evalConstInt64ForAllocation(expr frontend.Expr) (int64, bool) {
	switch e := expr.(type) {
	case nil:
		return 0, false
	case *frontend.NumberExpr:
		return int64(e.Value), true
	case *frontend.UnaryExpr:
		v, ok := evalConstInt64ForAllocation(e.X)
		if !ok {
			return 0, false
		}
		if e.Op == frontend.TokenMinus {
			return -v, true
		}
		return 0, false
	case *frontend.BinaryExpr:
		left, ok := evalConstInt64ForAllocation(e.Left)
		if !ok {
			return 0, false
		}
		right, ok := evalConstInt64ForAllocation(e.Right)
		if !ok {
			return 0, false
		}
		switch e.Op {
		case frontend.TokenPlus:
			return left + right, true
		case frontend.TokenMinus:
			return left - right, true
		case frontend.TokenStar:
			return left * right, true
		case frontend.TokenSlash:
			if right == 0 {
				return 0, false
			}
			return left / right, true
		case frontend.TokenPercent:
			if right == 0 {
				return 0, false
			}
			return left % right, true
		default:
			return 0, false
		}
	default:
		return 0, false
	}
}
