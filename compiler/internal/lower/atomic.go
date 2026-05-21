package lower

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/target"
)

type atomicBuiltinSpec struct {
	Op        target.AtomicOp
	Order     target.MemoryOrder
	WidthBits int
	Pointer   bool
	Fence     bool
}

type atomicBuiltinOrder struct {
	Suffix string
	Order  target.MemoryOrder
}

type atomicBuiltinWidth struct {
	Suffix    string
	WidthBits int
	Pointer   bool
}

var atomicBuiltinOrders = []atomicBuiltinOrder{
	{Suffix: "relaxed", Order: target.MemoryOrderRelaxed},
	{Suffix: "acquire", Order: target.MemoryOrderAcquire},
	{Suffix: "release", Order: target.MemoryOrderRelease},
	{Suffix: "acq_rel", Order: target.MemoryOrderAcqRel},
	{Suffix: "seq_cst", Order: target.MemoryOrderSeqCst},
}

var atomicBuiltinWidths = []atomicBuiltinWidth{
	{Suffix: "u8", WidthBits: 8},
	{Suffix: "u16", WidthBits: 16},
	{Suffix: "i32", WidthBits: 32},
	{Suffix: "i64", WidthBits: 64},
	{Suffix: "ptr", Pointer: true},
}

var atomicBuiltinOps = []struct {
	Prefix string
	Op     target.AtomicOp
}{
	{Prefix: "compare_exchange_weak_", Op: target.AtomicCompareExchangeWeak},
	{Prefix: "compare_exchange_", Op: target.AtomicCompareExchange},
	{Prefix: "fetch_add_", Op: target.AtomicFetchAdd},
	{Prefix: "fetch_sub_", Op: target.AtomicFetchSub},
	{Prefix: "fetch_and_", Op: target.AtomicFetchAnd},
	{Prefix: "fetch_or_", Op: target.AtomicFetchOr},
	{Prefix: "fetch_xor_", Op: target.AtomicFetchXor},
	{Prefix: "exchange_", Op: target.AtomicExchange},
	{Prefix: "store_", Op: target.AtomicStore},
	{Prefix: "load_", Op: target.AtomicLoad},
}

func parseAtomicBuiltinName(name string) (atomicBuiltinSpec, bool, error) {
	const prefix = "core.atomic_"
	if !strings.HasPrefix(name, prefix) {
		return atomicBuiltinSpec{}, false, nil
	}
	rest := strings.TrimPrefix(name, prefix)
	if strings.HasPrefix(rest, "fence_") {
		orderSuffix := strings.TrimPrefix(rest, "fence_")
		order, ok := parseAtomicBuiltinOrder(orderSuffix)
		if !ok {
			return atomicBuiltinSpec{}, true, fmt.Errorf("unsupported atomic fence memory order suffix %q", orderSuffix)
		}
		return atomicBuiltinSpec{Op: target.AtomicFence, Order: order, Fence: true}, true, nil
	}
	for _, op := range atomicBuiltinOps {
		if !strings.HasPrefix(rest, op.Prefix) {
			continue
		}
		tail := strings.TrimPrefix(rest, op.Prefix)
		for _, width := range atomicBuiltinWidths {
			widthPrefix := width.Suffix + "_"
			if !strings.HasPrefix(tail, widthPrefix) {
				continue
			}
			orderSuffix := strings.TrimPrefix(tail, widthPrefix)
			order, ok := parseAtomicBuiltinOrder(orderSuffix)
			if !ok {
				return atomicBuiltinSpec{}, true, fmt.Errorf("unsupported atomic memory order suffix %q", orderSuffix)
			}
			if !atomicBuiltinOrderAllowed(op.Op, order) {
				return atomicBuiltinSpec{}, true, fmt.Errorf("atomic %s does not support memory order %s", op.Op, order)
			}
			return atomicBuiltinSpec{
				Op:        op.Op,
				Order:     order,
				WidthBits: width.WidthBits,
				Pointer:   width.Pointer,
			}, true, nil
		}
		return atomicBuiltinSpec{}, true, fmt.Errorf("unsupported atomic value width in builtin %q", name)
	}
	return atomicBuiltinSpec{}, true, fmt.Errorf("unsupported atomic builtin %q", name)
}

func parseAtomicBuiltinOrder(suffix string) (target.MemoryOrder, bool) {
	for _, order := range atomicBuiltinOrders {
		if suffix == order.Suffix {
			return order.Order, true
		}
	}
	return target.MemoryOrderUnknown, false
}

func atomicBuiltinOrderAllowed(op target.AtomicOp, order target.MemoryOrder) bool {
	switch op {
	case target.AtomicLoad:
		return order == target.MemoryOrderRelaxed || order == target.MemoryOrderAcquire || order == target.MemoryOrderSeqCst
	case target.AtomicStore:
		return order == target.MemoryOrderRelaxed || order == target.MemoryOrderRelease || order == target.MemoryOrderSeqCst
	case target.AtomicExchange, target.AtomicCompareExchange, target.AtomicCompareExchangeWeak,
		target.AtomicFetchAdd, target.AtomicFetchSub, target.AtomicFetchAnd, target.AtomicFetchOr, target.AtomicFetchXor:
		return order == target.MemoryOrderRelaxed || order == target.MemoryOrderAcquire ||
			order == target.MemoryOrderRelease || order == target.MemoryOrderAcqRel || order == target.MemoryOrderSeqCst
	default:
		return false
	}
}

func (l *lowerer) lowerAtomicBuiltinCall(e *frontend.CallExpr) (int, bool, error) {
	spec, ok, err := parseAtomicBuiltinName(e.Name)
	if !ok || err != nil {
		return 0, ok, err
	}
	if spec.Fence {
		if len(e.Args) != 1 {
			return 0, true, fmt.Errorf("%s: atomic_fence_%s expects 1 memory capability argument", frontend.FormatPos(e.At), spec.Order)
		}
		slots, err := l.lowerExpr(e.Args[0])
		if err != nil {
			return 0, true, err
		}
		if slots != 1 {
			return 0, true, fmt.Errorf("%s: atomic_fence_%s expects a 1-slot memory capability", frontend.FormatPos(e.Args[0].Pos()), spec.Order)
		}
		discard := l.ensureDiscardLocal()
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
		kind, err := atomicFenceKindForOrder(spec.Order)
		if err != nil {
			return 0, true, err
		}
		l.emit(ir.IRInstr{Kind: kind, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		return 1, true, nil
	}

	expectedArgs := 0
	switch spec.Op {
	case target.AtomicLoad:
		expectedArgs = 2
	case target.AtomicStore, target.AtomicExchange,
		target.AtomicFetchAdd, target.AtomicFetchSub, target.AtomicFetchAnd, target.AtomicFetchOr, target.AtomicFetchXor:
		expectedArgs = 3
	case target.AtomicCompareExchange, target.AtomicCompareExchangeWeak:
		expectedArgs = 4
	default:
		return 0, true, fmt.Errorf("%s: unsupported atomic op %s", frontend.FormatPos(e.At), spec.Op)
	}
	if len(e.Args) != expectedArgs {
		return 0, true, fmt.Errorf("%s: atomic %s expects %d arguments", frontend.FormatPos(e.At), spec.Op, expectedArgs)
	}
	total := 0
	for _, arg := range e.Args {
		slots, err := l.lowerExpr(arg)
		if err != nil {
			return 0, true, err
		}
		total += slots
	}
	if total != expectedArgs {
		return 0, true, fmt.Errorf("%s: atomic %s expects %d argument slots", frontend.FormatPos(e.At), spec.Op, expectedArgs)
	}
	var kind ir.IRInstrKind
	if spec.Pointer {
		kind, err = atomicPointerKindForOp(spec.Op)
	} else {
		kind, err = atomicValueKindForOpWidth(spec.Op, spec.WidthBits)
	}
	if err != nil {
		return 0, true, err
	}
	l.emit(ir.IRInstr{Kind: kind, Pos: e.At})
	return 1, true, nil
}

func atomicValueKindForOpWidth(op target.AtomicOp, widthBits int) (ir.IRInstrKind, error) {
	switch op {
	case target.AtomicFence:
		return 0, fmt.Errorf("atomic fence lowering requires atomicFenceKindForOrder")
	}

	switch widthBits {
	case 8:
		return atomicValueKindForOp(op, widthBits,
			ir.IRAtomicLoadI8,
			ir.IRAtomicStoreI8,
			ir.IRAtomicExchangeI8,
			ir.IRAtomicCompareExchangeI8,
			ir.IRAtomicFetchAddI8,
			ir.IRAtomicFetchSubI8,
			ir.IRAtomicFetchAndI8,
			ir.IRAtomicFetchOrI8,
			ir.IRAtomicFetchXorI8,
		)
	case 16:
		return atomicValueKindForOp(op, widthBits,
			ir.IRAtomicLoadI16,
			ir.IRAtomicStoreI16,
			ir.IRAtomicExchangeI16,
			ir.IRAtomicCompareExchangeI16,
			ir.IRAtomicFetchAddI16,
			ir.IRAtomicFetchSubI16,
			ir.IRAtomicFetchAndI16,
			ir.IRAtomicFetchOrI16,
			ir.IRAtomicFetchXorI16,
		)
	case 32:
		return atomicValueKindForOp(op, widthBits,
			ir.IRAtomicLoadI32,
			ir.IRAtomicStoreI32,
			ir.IRAtomicExchangeI32,
			ir.IRAtomicCompareExchangeI32,
			ir.IRAtomicFetchAddI32,
			ir.IRAtomicFetchSubI32,
			ir.IRAtomicFetchAndI32,
			ir.IRAtomicFetchOrI32,
			ir.IRAtomicFetchXorI32,
		)
	case 64:
		return atomicValueKindForOp(op, widthBits,
			ir.IRAtomicLoadI64,
			ir.IRAtomicStoreI64,
			ir.IRAtomicExchangeI64,
			ir.IRAtomicCompareExchangeI64,
			ir.IRAtomicFetchAddI64,
			ir.IRAtomicFetchSubI64,
			ir.IRAtomicFetchAndI64,
			ir.IRAtomicFetchOrI64,
			ir.IRAtomicFetchXorI64,
		)
	default:
		return 0, fmt.Errorf("unsupported atomic width %d bits", widthBits)
	}
}

func atomicPointerKindForOp(op target.AtomicOp) (ir.IRInstrKind, error) {
	switch op {
	case target.AtomicFence:
		return 0, fmt.Errorf("atomic fence lowering requires atomicFenceKindForOrder")
	case target.AtomicLoad:
		return ir.IRAtomicLoadPtr, nil
	case target.AtomicStore:
		return ir.IRAtomicStorePtr, nil
	case target.AtomicExchange:
		return ir.IRAtomicExchangePtr, nil
	case target.AtomicCompareExchange, target.AtomicCompareExchangeWeak:
		return ir.IRAtomicCompareExchangePtr, nil
	case target.AtomicFetchAdd:
		return ir.IRAtomicFetchAddPtr, nil
	case target.AtomicFetchSub:
		return ir.IRAtomicFetchSubPtr, nil
	case target.AtomicFetchAnd:
		return ir.IRAtomicFetchAndPtr, nil
	case target.AtomicFetchOr:
		return ir.IRAtomicFetchOrPtr, nil
	case target.AtomicFetchXor:
		return ir.IRAtomicFetchXorPtr, nil
	default:
		return 0, fmt.Errorf("unsupported atomic op %s for pointer-sized value", op)
	}
}

func atomicValueKindForOp(
	op target.AtomicOp,
	widthBits int,
	load ir.IRInstrKind,
	store ir.IRInstrKind,
	exchange ir.IRInstrKind,
	compareExchange ir.IRInstrKind,
	fetchAdd ir.IRInstrKind,
	fetchSub ir.IRInstrKind,
	fetchAnd ir.IRInstrKind,
	fetchOr ir.IRInstrKind,
	fetchXor ir.IRInstrKind,
) (ir.IRInstrKind, error) {
	switch op {
	case target.AtomicLoad:
		return load, nil
	case target.AtomicStore:
		return store, nil
	case target.AtomicExchange:
		return exchange, nil
	case target.AtomicCompareExchange, target.AtomicCompareExchangeWeak:
		return compareExchange, nil
	case target.AtomicFetchAdd:
		return fetchAdd, nil
	case target.AtomicFetchSub:
		return fetchSub, nil
	case target.AtomicFetchAnd:
		return fetchAnd, nil
	case target.AtomicFetchOr:
		return fetchOr, nil
	case target.AtomicFetchXor:
		return fetchXor, nil
	default:
		return 0, fmt.Errorf("unsupported atomic op %s for %d-bit value", op, widthBits)
	}
}

func atomicFenceKindForOrder(order target.MemoryOrder) (ir.IRInstrKind, error) {
	switch order {
	case target.MemoryOrderRelaxed:
		return ir.IRAtomicFenceRelaxed, nil
	case target.MemoryOrderAcquire:
		return ir.IRAtomicFenceAcquire, nil
	case target.MemoryOrderRelease:
		return ir.IRAtomicFenceRelease, nil
	case target.MemoryOrderAcqRel:
		return ir.IRAtomicFenceAcqRel, nil
	case target.MemoryOrderSeqCst:
		return ir.IRAtomicFenceSeqCst, nil
	default:
		return 0, fmt.Errorf("unsupported atomic fence memory order %s", order)
	}
}
