package lower

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/target"
)

func TestAtomicFenceKindForOrderMapsEveryMemoryOrder(t *testing.T) {
	cases := []struct {
		order target.MemoryOrder
		want  ir.IRInstrKind
	}{
		{target.MemoryOrderRelaxed, ir.IRAtomicFenceRelaxed},
		{target.MemoryOrderAcquire, ir.IRAtomicFenceAcquire},
		{target.MemoryOrderRelease, ir.IRAtomicFenceRelease},
		{target.MemoryOrderAcqRel, ir.IRAtomicFenceAcqRel},
		{target.MemoryOrderSeqCst, ir.IRAtomicFenceSeqCst},
	}

	for _, tc := range cases {
		got, err := atomicFenceKindForOrder(tc.order)
		if err != nil {
			t.Fatalf("atomicFenceKindForOrder(%s): %v", tc.order, err)
		}
		if got != tc.want {
			t.Fatalf("atomicFenceKindForOrder(%s) = %v, want %v", tc.order, got, tc.want)
		}
	}
}

func TestAtomicFenceKindForOrderRejectsUnknownOrder(t *testing.T) {
	_, err := atomicFenceKindForOrder(target.MemoryOrderUnknown)
	if err == nil || !strings.Contains(err.Error(), "unsupported atomic fence memory order unknown") {
		t.Fatalf("expected unsupported memory order diagnostic, got %v", err)
	}
}

func TestAtomicValueKindForOpWidthMapsFixedWidths(t *testing.T) {
	cases := []struct {
		op        target.AtomicOp
		widthBits int
		want      ir.IRInstrKind
	}{
		{target.AtomicLoad, 8, ir.IRAtomicLoadI8},
		{target.AtomicStore, 8, ir.IRAtomicStoreI8},
		{target.AtomicExchange, 8, ir.IRAtomicExchangeI8},
		{target.AtomicCompareExchange, 8, ir.IRAtomicCompareExchangeI8},
		{target.AtomicCompareExchangeWeak, 8, ir.IRAtomicCompareExchangeI8},
		{target.AtomicFetchAdd, 8, ir.IRAtomicFetchAddI8},
		{target.AtomicFetchSub, 8, ir.IRAtomicFetchSubI8},
		{target.AtomicFetchAnd, 8, ir.IRAtomicFetchAndI8},
		{target.AtomicFetchOr, 8, ir.IRAtomicFetchOrI8},
		{target.AtomicFetchXor, 8, ir.IRAtomicFetchXorI8},

		{target.AtomicLoad, 16, ir.IRAtomicLoadI16},
		{target.AtomicStore, 16, ir.IRAtomicStoreI16},
		{target.AtomicExchange, 16, ir.IRAtomicExchangeI16},
		{target.AtomicCompareExchange, 16, ir.IRAtomicCompareExchangeI16},
		{target.AtomicCompareExchangeWeak, 16, ir.IRAtomicCompareExchangeI16},
		{target.AtomicFetchAdd, 16, ir.IRAtomicFetchAddI16},
		{target.AtomicFetchSub, 16, ir.IRAtomicFetchSubI16},
		{target.AtomicFetchAnd, 16, ir.IRAtomicFetchAndI16},
		{target.AtomicFetchOr, 16, ir.IRAtomicFetchOrI16},
		{target.AtomicFetchXor, 16, ir.IRAtomicFetchXorI16},

		{target.AtomicLoad, 32, ir.IRAtomicLoadI32},
		{target.AtomicStore, 32, ir.IRAtomicStoreI32},
		{target.AtomicExchange, 32, ir.IRAtomicExchangeI32},
		{target.AtomicCompareExchange, 32, ir.IRAtomicCompareExchangeI32},
		{target.AtomicCompareExchangeWeak, 32, ir.IRAtomicCompareExchangeI32},
		{target.AtomicFetchAdd, 32, ir.IRAtomicFetchAddI32},
		{target.AtomicFetchSub, 32, ir.IRAtomicFetchSubI32},
		{target.AtomicFetchAnd, 32, ir.IRAtomicFetchAndI32},
		{target.AtomicFetchOr, 32, ir.IRAtomicFetchOrI32},
		{target.AtomicFetchXor, 32, ir.IRAtomicFetchXorI32},

		{target.AtomicLoad, 64, ir.IRAtomicLoadI64},
		{target.AtomicStore, 64, ir.IRAtomicStoreI64},
		{target.AtomicExchange, 64, ir.IRAtomicExchangeI64},
		{target.AtomicCompareExchange, 64, ir.IRAtomicCompareExchangeI64},
		{target.AtomicCompareExchangeWeak, 64, ir.IRAtomicCompareExchangeI64},
		{target.AtomicFetchAdd, 64, ir.IRAtomicFetchAddI64},
		{target.AtomicFetchSub, 64, ir.IRAtomicFetchSubI64},
		{target.AtomicFetchAnd, 64, ir.IRAtomicFetchAndI64},
		{target.AtomicFetchOr, 64, ir.IRAtomicFetchOrI64},
		{target.AtomicFetchXor, 64, ir.IRAtomicFetchXorI64},
	}

	for _, tc := range cases {
		got, err := atomicValueKindForOpWidth(tc.op, tc.widthBits)
		if err != nil {
			t.Fatalf("atomicValueKindForOpWidth(%s, %d): %v", tc.op, tc.widthBits, err)
		}
		if got != tc.want {
			t.Fatalf("atomicValueKindForOpWidth(%s, %d) = %v, want %v", tc.op, tc.widthBits, got, tc.want)
		}
	}
}

func TestAtomicValueKindForOpWidthRejectsUnsupportedCases(t *testing.T) {
	cases := []struct {
		name      string
		op        target.AtomicOp
		widthBits int
		want      string
	}{
		{
			name:      "unsupported-width",
			op:        target.AtomicLoad,
			widthBits: 24,
			want:      "unsupported atomic width 24 bits",
		},
		{
			name:      "fence-uses-order-helper",
			op:        target.AtomicFence,
			widthBits: 32,
			want:      "atomic fence lowering requires atomicFenceKindForOrder",
		},
		{
			name:      "unknown-op",
			op:        target.AtomicOpUnknown,
			widthBits: 32,
			want:      "unsupported atomic op unknown for 32-bit value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := atomicValueKindForOpWidth(tc.op, tc.widthBits)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected diagnostic containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestAtomicPointerKindForOpMapsPointerOps(t *testing.T) {
	cases := []struct {
		op   target.AtomicOp
		want ir.IRInstrKind
	}{
		{target.AtomicLoad, ir.IRAtomicLoadPtr},
		{target.AtomicStore, ir.IRAtomicStorePtr},
		{target.AtomicExchange, ir.IRAtomicExchangePtr},
		{target.AtomicCompareExchange, ir.IRAtomicCompareExchangePtr},
		{target.AtomicCompareExchangeWeak, ir.IRAtomicCompareExchangePtr},
		{target.AtomicFetchAdd, ir.IRAtomicFetchAddPtr},
		{target.AtomicFetchSub, ir.IRAtomicFetchSubPtr},
		{target.AtomicFetchAnd, ir.IRAtomicFetchAndPtr},
		{target.AtomicFetchOr, ir.IRAtomicFetchOrPtr},
		{target.AtomicFetchXor, ir.IRAtomicFetchXorPtr},
	}

	for _, tc := range cases {
		got, err := atomicPointerKindForOp(tc.op)
		if err != nil {
			t.Fatalf("atomicPointerKindForOp(%s): %v", tc.op, err)
		}
		if got != tc.want {
			t.Fatalf("atomicPointerKindForOp(%s) = %v, want %v", tc.op, got, tc.want)
		}
	}
}

func TestAtomicPointerKindForOpRejectsUnsupportedCases(t *testing.T) {
	cases := []struct {
		name string
		op   target.AtomicOp
		want string
	}{
		{
			name: "fence-uses-order-helper",
			op:   target.AtomicFence,
			want: "atomic fence lowering requires atomicFenceKindForOrder",
		},
		{
			name: "unknown-op",
			op:   target.AtomicOpUnknown,
			want: "unsupported atomic op unknown for pointer-sized value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := atomicPointerKindForOp(tc.op)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected diagnostic containing %q, got %v", tc.want, err)
			}
		})
	}
}
