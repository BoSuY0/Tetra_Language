package runtimeabi_test

import (
	"math"
	"strings"
	"testing"

	. "tetra_language/compiler/internal/runtimeabi"
)

func TestRawPointerBoundsTracksAllocationBaseAndDerivedOffsets(t *testing.T) {
	root, err := NewRawAllocationBounds("p", 16)
	if err != nil {
		t.Fatalf("NewRawAllocationBounds: %v", err)
	}
	if root.Status != RawPointerBoundsAllocationBase || root.BaseID != "p" ||
		root.BaseBytes != 16 ||
		root.OffsetBytes != 0 {
		t.Fatalf("root metadata = %+v, want allocation-base p/16/0", root)
	}

	derived, diag := DeriveRawPointerBounds(root, 4, 4)
	if diag != nil {
		t.Fatalf("DeriveRawPointerBounds returned diagnostic: %+v", diag)
	}
	if derived.Status != RawPointerBoundsDerivedOffset || derived.BaseID != "p" ||
		derived.BaseBytes != 16 ||
		derived.OffsetBytes != 4 {
		t.Fatalf("derived metadata = %+v, want derived allocation-base offset 4", derived)
	}
	if !derived.VerifiedAllocationRoot {
		t.Fatalf("derived metadata should keep verified allocation root: %+v", derived)
	}
}

func TestRawPointerBoundsDiagnosticsForImpossiblePtrAdd(t *testing.T) {
	root, err := NewRawAllocationBounds("p", 4)
	if err != nil {
		t.Fatalf("NewRawAllocationBounds: %v", err)
	}

	negBounds, neg := DeriveRawPointerBounds(root, -1, 1)
	if neg == nil || neg.Code != RawPointerDiagnosticNegativePtrAdd ||
		!strings.Contains(neg.Message, "negative ptr_add offset") {
		t.Fatalf(
			"negative ptr_add diagnostic = %+v, want %s",
			neg,
			RawPointerDiagnosticNegativePtrAdd,
		)
	}
	if negBounds.Status != RawPointerBoundsRejectedNegativeOffset {
		t.Fatalf(
			"negative ptr_add status = %q, want %q",
			negBounds.Status,
			RawPointerBoundsRejectedNegativeOffset,
		)
	}

	upperBounds, upper := DeriveRawPointerBounds(root, 4, 1)
	if upper == nil || upper.Code != RawPointerDiagnosticAllocationUpperBound ||
		!strings.Contains(upper.Message, "allocation upper bound") {
		t.Fatalf(
			"upper-bound ptr_add diagnostic = %+v, want %s",
			upper,
			RawPointerDiagnosticAllocationUpperBound,
		)
	}
	if upperBounds.Status != RawPointerBoundsRejectedUpperBound {
		t.Fatalf(
			"upper-bound ptr_add status = %q, want %q",
			upperBounds.Status,
			RawPointerBoundsRejectedUpperBound,
		)
	}

	widthBounds, width := DeriveRawPointerBounds(root, 2, 4)
	if width == nil || width.Code != RawPointerDiagnosticAccessWidth {
		t.Fatalf("width diagnostic = %+v, want %s", width, RawPointerDiagnosticAccessWidth)
	}
	if widthBounds.Status != RawPointerBoundsRejectedAccessWidthOverflow {
		t.Fatalf(
			"access-width status = %q, want %q",
			widthBounds.Status,
			RawPointerBoundsRejectedAccessWidthOverflow,
		)
	}
}

func TestRawPointerBoundsRejectsOffsetWidthIntegerOverflow(t *testing.T) {
	root, err := NewRawAllocationBounds("huge", math.MaxInt64)
	if err != nil {
		t.Fatalf("NewRawAllocationBounds: %v", err)
	}

	bounds, diag := DeriveRawPointerBounds(root, math.MaxInt64-1, 4)
	if diag == nil || diag.Code != RawPointerDiagnosticAccessWidth {
		t.Fatalf("overflow diagnostic = %+v, want %s", diag, RawPointerDiagnosticAccessWidth)
	}
	if bounds.Status != RawPointerBoundsRejectedAccessWidthOverflow {
		t.Fatalf(
			"overflow status = %q, want %q",
			bounds.Status,
			RawPointerBoundsRejectedAccessWidthOverflow,
		)
	}
}

func TestUnknownRawPointerBoundsStayCheckedAndRawSliceExternal(t *testing.T) {
	unknown := UnknownRawPointerBounds("ffi pointer")
	derived, diag := DeriveRawPointerBounds(unknown, 8, 1)
	if diag != nil {
		t.Fatalf(
			"unknown ptr_add should stay checked rather than claim allocation-base rejection: %+v",
			diag,
		)
	}
	if derived.Status != RawPointerBoundsCheckedExternalUnknown || derived.VerifiedAllocationRoot {
		t.Fatalf("unknown derived metadata = %+v, want checked external unknown", derived)
	}

	rawSlice := RawSliceBoundsFromParts(derived, 4, 1)
	if rawSlice.Status != RawSliceBoundsExternalUnknown || rawSlice.VerifiedAllocationRoot {
		t.Fatalf("raw slice metadata = %+v, want external unknown", rawSlice)
	}

	base, err := NewRawAllocationBounds("owned", 8)
	if err != nil {
		t.Fatalf("NewRawAllocationBounds: %v", err)
	}
	verifiedSlice := RawSliceBoundsFromParts(base, 8, 1)
	if verifiedSlice.Status != RawSliceBoundsVerifiedAllocationRoot ||
		!verifiedSlice.VerifiedAllocationRoot {
		t.Fatalf("verified raw slice metadata = %+v, want verified allocation root", verifiedSlice)
	}
}

func TestRawSliceBoundsDoNotVerifyRejectedRawPointer(t *testing.T) {
	root, err := NewRawAllocationBounds("p", 8)
	if err != nil {
		t.Fatalf("NewRawAllocationBounds: %v", err)
	}
	rejected, diag := DeriveRawPointerBounds(root, -1, 1)
	if diag == nil || rejected.Status != RawPointerBoundsRejectedNegativeOffset {
		t.Fatalf("rejected pointer = %+v diag=%+v, want negative rejection", rejected, diag)
	}
	rawSlice := RawSliceBoundsFromParts(rejected, 1, 1)
	if rawSlice.Status != RawSliceBoundsExternalUnknown || rawSlice.VerifiedAllocationRoot {
		t.Fatalf("raw slice from rejected pointer = %+v, want external unknown", rawSlice)
	}
}

func TestRawSliceBoundsRejectsLengthArithmeticOverflow(t *testing.T) {
	root, err := NewRawAllocationBounds("huge", math.MaxInt64)
	if err != nil {
		t.Fatalf("NewRawAllocationBounds: %v", err)
	}
	rawSlice := RawSliceBoundsFromParts(root, math.MaxInt64/2+2, 5)
	if rawSlice.Status != RawSliceBoundsRejectedLengthOverflow || rawSlice.VerifiedAllocationRoot {
		t.Fatalf(
			"raw slice with overflowing length bytes = %+v, want rejected length overflow",
			rawSlice,
		)
	}

	derived, diag := DeriveRawPointerBounds(root, math.MaxInt64-1, 1)
	if diag != nil {
		t.Fatalf("DeriveRawPointerBounds: %+v", diag)
	}
	rawSlice = RawSliceBoundsFromParts(derived, 4, 1)
	if rawSlice.Status != RawSliceBoundsExternalUnknown || rawSlice.VerifiedAllocationRoot {
		t.Fatalf("raw slice with overflowing offset+length = %+v, want external unknown", rawSlice)
	}
}

func TestRawSliceBoundsRejectsI32ByteLengthOverflow(t *testing.T) {
	root, err := NewRawAllocationBounds("p", math.MaxInt64)
	if err != nil {
		t.Fatalf("NewRawAllocationBounds: %v", err)
	}

	rawSlice := RawSliceBoundsFromParts(root, 536870912, 4)
	if rawSlice.Status != RawSliceBoundsRejectedLengthOverflow || rawSlice.VerifiedAllocationRoot {
		t.Fatalf(
			"raw slice i32 byte overflow = %+v, want rejected length overflow without verified root",
			rawSlice,
		)
	}
}

func TestRawSliceBoundsRejectsInvalidElementWidth(t *testing.T) {
	root, err := NewRawAllocationBounds("p", 16)
	if err != nil {
		t.Fatalf("NewRawAllocationBounds: %v", err)
	}

	rawSlice := RawSliceBoundsFromParts(root, 4, 0)
	if rawSlice.Status != RawSliceBoundsStatus("rejected_invalid_element_width") ||
		rawSlice.VerifiedAllocationRoot {
		t.Fatalf(
			("raw slice invalid element width = %+v, want rejected_invalid_" +
				"element_width without verified root"),
			rawSlice,
		)
	}
}

func TestRawSliceBoundsRejectsNegativeLengthForVerifiedRoot(t *testing.T) {
	root, err := NewRawAllocationBounds("p", 8)
	if err != nil {
		t.Fatalf("NewRawAllocationBounds: %v", err)
	}

	rawSlice := RawSliceBoundsFromParts(root, -1, 1)
	if string(rawSlice.Status) != "rejected_negative_length" || rawSlice.VerifiedAllocationRoot {
		t.Fatalf(
			"raw slice with negative length = %+v, want rejected_negative_length without verified root",
			rawSlice,
		)
	}
}
