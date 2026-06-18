package plir_test

import (
	"strings"
	"testing"

	. "tetra_language/compiler/internal/plir"
)

func TestVerifierRejectsContradictoryBorrowAndOwnershipFacts(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "v0",
			Type:       "[]u8",
			Borrow:     BorrowImm,
			Provenance: Provenance{Kind: ProvenanceParam, Root: "xs"},
		}},
		Facts: []Fact{{
			ID:      "f0",
			Kind:    FactOwned,
			ValueID: "v0",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "owned contradicts borrowed value") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsNoEscapeOnEscapingValue(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "v0",
			Type:       "[]u8",
			Escape:     EscapeReturn,
			Provenance: Provenance{Kind: ProvenanceParam, Root: "xs"},
		}},
		Facts: []Fact{{
			ID:      "f0",
			Kind:    FactNoEscape,
			ValueID: "v0",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "no_escape contradicts escaping value") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsBorrowedFactWithoutNoEscape(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "v0",
			Type:       "[]u8",
			Borrow:     BorrowImm,
			Provenance: Provenance{Kind: ProvenanceParam, Root: "xs"},
		}},
		Facts: []Fact{{
			ID:      "borrowed",
			Kind:    FactBorrowedImm,
			ValueID: "v0",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "borrowed_imm requires no_escape fact") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsDerivedWindowWithoutSource(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "view",
			Kind:       ValueView,
			Type:       "[]u8",
			Borrow:     BorrowImm,
			Escape:     EscapeNoEscape,
			Provenance: Provenance{Kind: ProvenanceParam, Root: "xs"},
		}},
		Facts: []Fact{
			{ID: "window", Kind: FactDerivedWindow, ValueID: "view", Range: "0..1"},
			{ID: "borrowed", Kind: FactBorrowedImm, ValueID: "view"},
			{ID: "no_escape", Kind: FactNoEscape, ValueID: "view"},
		},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "derived_window requires source") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsContradictoryProvenanceFacts(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "v0",
			Type:       "[]u8",
			Provenance: Provenance{Kind: ProvenanceExternal, Root: "ffi"},
		}},
		Facts: []Fact{
			{ID: "known", Kind: FactProvenanceKnown, ValueID: "v0"},
			{ID: "unknown", Kind: FactProvenanceUnknown, ValueID: "v0"},
		},
	}}}
	err := VerifyProgram(prog)
	if err == nil ||
		!strings.Contains(err.Error(), "provenance_known contradicts provenance_unknown") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsNoHeapAllocationOnAllocationIntent(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:   "alloc",
			Kind: ValueAllocIntent,
			Type: "[]u8",
			Alloc: &AllocIntent{
				ElementType:         "u8",
				ElementSize:         1,
				LengthExpr:          "n",
				ZeroGuardStatus:     "checked",
				NegativeGuardStatus: "checked",
				OverflowGuardStatus: "checked",
			},
			Provenance: Provenance{Kind: ProvenanceAllocation, Root: "alloc"},
		}},
		Facts: []Fact{{
			ID:      "f0",
			Kind:    FactNoHeapAllocation,
			ValueID: "alloc",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil ||
		!strings.Contains(err.Error(), "no_heap_allocation contradicts allocation intent") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsCopyAllocationWithoutOwnedFact(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:   "copy",
			Kind: ValueAllocIntent,
			Type: "[]u8",
			Alloc: &AllocIntent{
				ElementType:         "u8",
				ElementSize:         1,
				LengthExpr:          "xs.len",
				ZeroGuardStatus:     "checked",
				NegativeGuardStatus: "checked",
				OverflowGuardStatus: "checked",
				Builtin:             "core.slice_copy_u8",
			},
			Provenance: Provenance{Kind: ProvenanceAllocation, Root: "copy"},
		}},
		Facts: []Fact{{ID: "known", Kind: FactProvenanceKnown, ValueID: "copy"}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "copy allocation intent requires owned fact") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestFromCheckedProgramRecordsRawSliceExternalProvenance(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(1)
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let view: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
        return view.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	var mainFn Function
	for _, candidate := range prog.Funcs {
		if candidate.Name == "main" {
			mainFn = candidate
			break
		}
	}
	if mainFn.Name == "" {
		t.Fatalf("missing PLIR function main")
	}
	var sawRawView bool
	for _, value := range mainFn.Values {
		if value.ID == "view:view" {
			sawRawView = true
			if value.Provenance.Kind != ProvenanceExternal {
				t.Fatalf("raw view provenance = %s, want external", value.Provenance.Kind)
			}
			if value.UnsafeClass != UnsafeUnknown {
				t.Fatalf("raw view unsafe class = %q, want %q", value.UnsafeClass, UnsafeUnknown)
			}
		}
	}
	if !sawRawView {
		t.Fatalf("missing raw view in PLIR values: %#v", mainFn.Values)
	}
	var sawRawSliceOp bool
	for _, op := range mainFn.Ops {
		if op.Kind == OpUnsafe && strings.Contains(op.Note, "external-provenance view") {
			sawRawSliceOp = true
			if op.UnsafeClass != UnsafeUnknown {
				t.Fatalf(
					"raw slice op unsafe class = %q, want %q: %+v",
					op.UnsafeClass,
					UnsafeUnknown,
					op,
				)
			}
		}
	}
	if !sawRawSliceOp {
		t.Fatalf("missing raw slice unsafe operation:\n%s", FormatText(prog))
	}
	if !mainFn.HasFact(FactProvenanceUnknown) {
		t.Fatalf("raw view should record conservative unknown provenance fact: %#v", mainFn.Facts)
	}
	if mainFn.HasFact(FactLenStable) {
		for _, fact := range mainFn.Facts {
			if fact.ValueID == "view:view" && fact.Kind == FactLenStable {
				t.Fatalf("raw view unexpectedly received len_stable fact: %#v", fact)
			}
		}
	}
}

func TestFromCheckedProgramRecordsVerifiedRootRawSliceBoundsEvidence(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let view: []u8 = core.raw_slice_u8_from_parts(p, 4, mem)
        return view.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	view := findPLIRValue(t, fn, "view:view")
	if view.Provenance.Kind == ProvenanceAllocation || view.Provenance.Kind == ProvenanceParam ||
		view.Provenance.Kind == ProvenanceStack {
		t.Fatalf("verified-root raw slice must not become safe provenance: %+v", view)
	}
	if view.UnsafeClass != UnsafeChecked {
		t.Fatalf(
			"verified-root raw slice unsafe class = %q, want %q\n%s",
			view.UnsafeClass,
			UnsafeChecked,
			FormatText(prog),
		)
	}
	assertUnsafeNoteContains(
		t,
		fn,
		"core.raw_slice_u8_from_parts",
		"raw_slice_bounds",
		"verified_allocation_root",
		"base:p",
		"length_bytes:4",
	)
	for _, fact := range fn.Facts {
		if fact.ValueID == "view:view" &&
			(fact.Kind == FactProvenanceKnown || fact.Kind == FactLenStable || fact.Kind == FactIndexInRange || fact.Kind == FactNoAlias) {
			t.Fatalf(
				"verified-root raw slice gained safe proof fact: %+v\n%s",
				fact,
				FormatText(prog),
			)
		}
	}
}

func TestFromCheckedProgramRejectsVerifiedRootRawSliceNegativeLength(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let view: []u8 = core.raw_slice_u8_from_parts(p, 0 - 1, mem)
        return view.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	assertUnsafeNoteContains(t, fn, "core.raw_slice_u8_from_parts", "rejected_negative_length")
	view := findPLIRValue(t, fn, "view:view")
	if view.UnsafeClass != UnsafeChecked {
		t.Fatalf(
			"rejected verified-root raw slice unsafe class = %q, want %q",
			view.UnsafeClass,
			UnsafeChecked,
		)
	}
}

func TestFromCheckedProgramRecordsRawSliceElementWidthAndOverflowEvidence(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(64)
        let bytes: []u8 = core.raw_slice_u8_from_parts(p, 8, mem)
        let words: []u16 = core.raw_slice_u16_from_parts(p, 8, mem)
        let ints: []i32 = core.raw_slice_i32_from_parts(p, 8, mem)
        let flags: []bool = core.raw_slice_bool_from_parts(p, 8, mem)
        let overflow: []i32 = core.raw_slice_i32_from_parts(p, 536870912, mem)
        return bytes.len + words.len + ints.len + flags.len + overflow.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	for _, tc := range []struct {
		name        string
		elemSize    string
		lengthBytes string
	}{
		{name: "core.raw_slice_u8_from_parts", elemSize: "elem_size:1", lengthBytes: "length_bytes:8"},
		{name: "core.raw_slice_u16_from_parts", elemSize: "elem_size:2", lengthBytes: "length_bytes:16"},
		{name: "core.raw_slice_i32_from_parts", elemSize: "elem_size:4", lengthBytes: "length_bytes:32"},
		{name: "core.raw_slice_bool_from_parts", elemSize: "elem_size:4", lengthBytes: "length_bytes:32"},
	} {
		assertUnsafeNoteContains(
			t,
			fn,
			tc.name,
			"raw_slice_bounds",
			"verified_allocation_root",
			tc.elemSize,
			tc.lengthBytes,
		)
	}
	assertUnsafeNoteContains(
		t,
		fn,
		"overflow",
		"raw_slice_bounds",
		"rejected_length_overflow",
		"elem_size:4",
	)
}

func TestFromCheckedProgramRecordsAllocBytesRawBoundsMetadata(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let q: ptr = core.ptr_add(p, 4, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return core.load_u8(q, mem)
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	var mainFn Function
	for _, candidate := range prog.Funcs {
		if candidate.Name == "main" {
			mainFn = candidate
			break
		}
	}
	if mainFn.Name == "" {
		t.Fatalf("missing PLIR function main")
	}
	var rawAlloc Value
	for _, value := range mainFn.Values {
		if value.ID == "alloc_intent:p" {
			rawAlloc = value
			break
		}
	}
	if rawAlloc.ID == "" || rawAlloc.Alloc == nil {
		t.Fatalf("missing raw alloc_bytes allocation intent: %#v", mainFn.Values)
	}
	if rawAlloc.Alloc.Builtin != "core.alloc_bytes" || rawAlloc.Alloc.ElementType != "raw_bytes" ||
		rawAlloc.Alloc.RawPointerBoundsStatus != "allocation_base_metadata" {
		t.Fatalf(
			"raw allocation intent = %+v, want alloc_bytes raw allocation-base metadata",
			rawAlloc.Alloc,
		)
	}
	if rawAlloc.UnsafeClass != UnsafeVerifiedRoot {
		t.Fatalf(
			"raw allocation unsafe class = %q, want %q",
			rawAlloc.UnsafeClass,
			UnsafeVerifiedRoot,
		)
	}
	var sawDerivedOffset bool
	var sawRawStore bool
	var sawRawLoad bool
	for _, op := range mainFn.Ops {
		if len(op.Outputs) == 1 && op.Outputs[0] == "q" &&
			strings.Contains(op.Note, "derived_allocation_offset") &&
			strings.Contains(op.Note, "base:p") {
			if op.UnsafeClass != UnsafeChecked {
				t.Fatalf(
					"ptr_add unsafe class = %q, want %q: %+v",
					op.UnsafeClass,
					UnsafeChecked,
					op,
				)
			}
			sawDerivedOffset = true
		}
		if op.Kind == OpUnsafe && strings.Contains(op.Note, "core.store_u8 raw memory gateway") {
			if op.UnsafeClass != UnsafeChecked {
				t.Fatalf(
					"store_u8 unsafe class = %q, want %q: %+v",
					op.UnsafeClass,
					UnsafeChecked,
					op,
				)
			}
			sawRawStore = true
		}
		if op.Kind == OpUnsafe && strings.Contains(op.Note, "core.load_u8 raw memory gateway") {
			if op.UnsafeClass != UnsafeChecked {
				t.Fatalf(
					"load_u8 unsafe class = %q, want %q: %+v",
					op.UnsafeClass,
					UnsafeChecked,
					op,
				)
			}
			sawRawLoad = true
		}
	}
	if !sawDerivedOffset {
		t.Fatalf("missing derived raw pointer offset operation:\n%s", FormatText(prog))
	}
	if !sawRawStore || !sawRawLoad {
		t.Fatalf(
			"missing raw load/store unsafe gateway operations store=%v load=%v:\n%s",
			sawRawStore,
			sawRawLoad,
			FormatText(prog),
		)
	}
}

func TestFromCheckedProgramRecordsVerifiedRootRawBoundsRejections(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let neg_base: ptr = core.alloc_bytes(8)
        let neg: ptr = core.ptr_add(neg_base, 0 - 1, mem)
        let neg_read: UInt8 = core.load_u8(neg, mem)
        let upper_base: ptr = core.alloc_bytes(8)
        let upper: ptr = core.ptr_add(upper_base, 8, mem)
        let upper_read: UInt8 = core.load_u8(upper, mem)
        let i32_base: ptr = core.alloc_bytes(8)
        let i32_ptr: ptr = core.ptr_add(i32_base, 5, mem)
        let i32_read: Int = core.load_i32(i32_ptr, mem)
        let ptr_base: ptr = core.alloc_bytes(4)
        let ptr_ptr: ptr = core.ptr_add(ptr_base, 1, mem)
        let ptr_write: ptr = core.store_ptr(ptr_ptr, ptr_base, mem)
        return 0
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	assertUnsafeNoteContains(t, fn, "neg", "rejected_negative_offset")
	assertUnsafeNoteContains(t, fn, "upper", "rejected_upper_bound")
	assertUnsafeNoteContains(
		t,
		fn,
		"core.load_i32 raw memory gateway",
		"rejected_access_width_overflow",
	)
	assertUnsafeNoteContains(
		t,
		fn,
		"core.store_ptr raw memory gateway",
		"rejected_access_width_overflow",
	)
}

func TestFromCheckedProgramKeepsUnknownRawPointerNegativeOffsetConservative(t *testing.T) {
	checked := checkedProgram(t, `
func external(raw: ptr) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let q: ptr = core.ptr_add(raw, 0 - 1, mem)
        let xs: []u8 = core.raw_slice_u8_from_parts(q, 1, mem)
        return xs.len
    return 0

func main() -> Int:
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "external")
	assertUnsafeNoteContains(t, fn, "core.ptr_add", "checked_external_unknown", "offset:0 - 1")
	assertUnsafeNoteContains(t, fn, "external-provenance view")
	for _, op := range fn.Ops {
		if op.Kind != OpUnsafe {
			continue
		}
		if strings.Contains(op.Note, "rejected_") ||
			strings.Contains(op.Note, "derived_allocation_offset") {
			t.Fatalf(
				"unknown raw pointer op gained checked-root bounds claim: %+v\n%s",
				op,
				FormatText(prog),
			)
		}
	}
}

func TestFromCheckedProgramRejectsNestedNegativePtrAddDelta(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let q: ptr = core.ptr_add(p, 8, mem)
        let r: ptr = core.ptr_add(q, 0 - 1, mem)
        let read: UInt8 = core.load_u8(r, mem)
        return 0
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	assertUnsafeNoteContains(t, fn, "r", "rejected_negative_offset")
	for _, op := range fn.Ops {
		if op.Kind == OpUnsafe && containsString(op.Outputs, "r") &&
			strings.Contains(op.Note, "derived_allocation_offset") {
			t.Fatalf(
				"nested negative ptr_add delta was accepted as derived offset: %+v\n%s",
				op,
				FormatText(prog),
			)
		}
	}
}

func TestFromCheckedProgramClearsRawPointerMetadataOnAssignment(t *testing.T) {
	checked := checkedProgram(t, `
func external(raw: ptr) -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        var q: ptr = core.ptr_add(p, 4, mem)
        q = core.ptr_add(p, 0 - 1, mem)
        let neg_read: UInt8 = core.load_u8(q, mem)
        q = core.ptr_add(raw, 0, mem)
        let unknown_read: UInt8 = core.load_u8(q, mem)
        return 0
    return 0

func main() -> Int:
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "external")
	assertUnsafeNoteContains(t, fn, "q", "rejected_negative_offset")
	assertUnsafeNoteContains(t, fn, "core.ptr_add", "checked_external_unknown", "base:raw")
	for _, op := range fn.Ops {
		if op.Kind != OpUnsafe || !strings.Contains(op.Note, "core.load_u8 raw memory gateway") {
			continue
		}
		if strings.Contains(op.Note, "derived_allocation_offset") {
			t.Fatalf(
				"load after reassignment retained stale verified-root metadata: %+v\n%s",
				op,
				FormatText(prog),
			)
		}
	}
}

func TestFromCheckedProgramKeepsDynamicRawPointerOffsetConservative(t *testing.T) {
	checked := checkedProgram(t, `
func read_at(n: Int) -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let q: ptr = core.ptr_add(p, n, mem)
        let read: UInt8 = core.load_u8(q, mem)
        return 0
    return 0

func main() -> Int:
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "read_at")
	assertUnsafeNoteContains(t, fn, "core.ptr_add", "checked_external_unknown", "offset:n")
	for _, op := range fn.Ops {
		if op.Kind != OpUnsafe {
			continue
		}
		if strings.Contains(op.Note, "derived_allocation_offset") ||
			strings.Contains(op.Note, "rejected_") {
			t.Fatalf(
				"dynamic raw offset received static bounds claim: %+v\n%s",
				op,
				FormatText(prog),
			)
		}
	}
}

func TestFromCheckedProgramRecordsAllocationLengthContract(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, islands, mem:
    var bytes: []u8 = make_u8(0)
    var flags: []bool = make_bool(536870912)
    island(64) as isl:
        var words: []u16 = core.island_make_u16(isl, 3)
        return bytes.len + flags.len + words.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	mainFn := findPLIRFunction(t, prog, "main")
	bytes := findPLIRAllocValue(t, mainFn, "bytes")
	if bytes.Alloc.ElementType != "u8" || bytes.Alloc.ElementSize != 1 ||
		bytes.Alloc.LengthExpr != "0" {
		t.Fatalf("bytes allocation intent = %+v", bytes.Alloc)
	}
	if bytes.Alloc.ZeroGuardStatus != "valid_empty_no_allocator" ||
		bytes.Alloc.NegativeGuardStatus != "reject_before_allocation" ||
		bytes.Alloc.OverflowGuardStatus != "reject_before_allocation" {
		t.Fatalf("bytes allocation guards = %+v", bytes.Alloc)
	}

	flags := findPLIRAllocValue(t, mainFn, "flags")
	if flags.Alloc.ElementType != "bool" || flags.Alloc.ElementSize != 4 ||
		flags.Alloc.LengthExpr != "536870912" {
		t.Fatalf("flags allocation intent = %+v", flags.Alloc)
	}
	if !flags.Alloc.LengthConstKnown || flags.Alloc.LengthConst != 536870912 {
		t.Fatalf(
			"flags length const = known:%v value:%d",
			flags.Alloc.LengthConstKnown,
			flags.Alloc.LengthConst,
		)
	}

	words := findPLIRAllocValue(t, mainFn, "words")
	if words.Alloc.ElementType != "u16" || words.Alloc.ElementSize != 2 ||
		words.Alloc.LengthExpr != "3" {
		t.Fatalf("island words allocation intent = %+v", words.Alloc)
	}
	if words.Provenance.Kind != ProvenanceIsland {
		t.Fatalf("island words provenance = %s, want island", words.Provenance.Kind)
	}
}

func TestFromCheckedProgramRecordsSliceWindowProvenanceAndRange(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    for x in xs.window(1, 2):
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(3)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"func sum",
		"fact derived_window",
		"range: xs[1..3]",
		"fact len_stable",
		"fact index_in_range",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramRecordsStringWindowProvenanceAndRange(t *testing.T) {
	checked := checkedProgram(t, `
func sum(text: String) -> Int
uses mem:
    var total = 0
    for ch in text.window(1, 3):
        total = total + ch
    return total

func main() -> Int
uses mem:
    let text: String = "abcdef"
    return sum(text)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"func sum",
		"value view:",
		": str",
		"fact derived_window",
		"range: text[1..4]",
		"fact len_stable",
		"fact index_in_range",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramRecordsSliceViewByteWidthAndNormalBuildBoundsChecks(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var bytes: []u8 = make_u8(4)
    var words: []u16 = make_u16(4)
    var nums: []i32 = make_i32(4)
    var flags: []bool = make_bool(4)
    let b: []u8 = bytes.window(1, 2)
    let w: []u16 = words.prefix(2)
    let n: []i32 = nums.suffix(1)
    let f: []bool = flags.window(0, 1)
    let s: String = "abcdef".window(1, 3)
    let sp: String = s.prefix(2)
    return b.len + w.len + n.len + f.len + sp.len
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	tests := []struct {
		output string
		want   []string
	}{
		{
			output: "view:b",
			want: []string{
				"core.slice_window_u8",
				"elem_width:1",
				"elem_shift:0",
				"bounds_check:normal_build",
			},
		},
		{
			output: "view:w",
			want: []string{
				"core.slice_prefix_u16",
				"elem_width:2",
				"elem_shift:1",
				"bounds_check:normal_build",
			},
		},
		{
			output: "view:n",
			want: []string{
				"core.slice_suffix_i32",
				"elem_width:4",
				"elem_shift:2",
				"bounds_check:normal_build",
			},
		},
		{
			output: "view:f",
			want: []string{
				"core.slice_window_bool",
				"elem_width:4",
				"elem_shift:2",
				"bounds_check:normal_build",
			},
		},
		{
			output: "view:s",
			want: []string{
				"core.string_window",
				"elem_width:1",
				"elem_shift:0",
				"bounds_check:normal_build",
			},
		},
		{
			output: "view:sp",
			want: []string{
				"core.string_prefix",
				"elem_width:1",
				"elem_shift:0",
				"bounds_check:normal_build",
			},
		},
	}
	for _, tc := range tests {
		op, ok := findOperationForOutput(fn, tc.output)
		if !ok {
			t.Fatalf("missing slice view operation for %s:\n%s", tc.output, FormatText(prog))
		}
		for _, want := range tc.want {
			if !strings.Contains(op.Note, want) {
				t.Fatalf("operation for %s note %q missing %q", tc.output, op.Note, want)
			}
		}
	}
}

func TestFromCheckedProgramRecordsCopyIntoOverlapAndCapacityContract(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    let src: []u8 = xs.window(0, 3)
    var dst: []u8 = xs.window(1, 3)
    return src.copy_into(dst)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	op, ok := findOperationNoteContaining(fn, "core.slice_copy_into_u8")
	if !ok {
		t.Fatalf("missing copy_into operation:\n%s", FormatText(prog))
	}
	for _, want := range []string{
		"source:src",
		"destination:dst",
		"dest_capacity_check:normal_build",
		"overlap:known_overlap",
	} {
		if !strings.Contains(op.Note, want) {
			t.Fatalf("copy_into note %q missing %q\n%s", op.Note, want, FormatText(prog))
		}
	}
	for _, valueID := range []string{"view:src", "view:dst"} {
		if hasFactForValue(fn, FactNoAlias, valueID) {
			t.Fatalf(
				"copy_into overlap must not create no_alias for %s:\n%s",
				valueID,
				FormatText(prog),
			)
		}
	}
}

func TestFromCheckedProgramRecordsBorrowCopyFacts(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(3)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    let view: []i32 = xs.window(1, 2)
    let borrowed: []i32 = view.borrow()
    let copied: []i32 = borrowed.copy()
    return copied.len
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findPLIRFunction(t, prog, "main")
	borrowed := findPLIRValue(t, fn, "view:borrowed")
	if borrowed.Borrow != BorrowImm || borrowed.Escape != EscapeNoEscape {
		t.Fatalf("borrowed value = %+v, want immutable no_escape borrow", borrowed)
	}
	copied := findPLIRAllocValue(t, fn, "copied")
	if copied.Borrow != BorrowNone || copied.Provenance.Kind != ProvenanceAllocation {
		t.Fatalf("copied value = %+v, want owned allocation provenance", copied)
	}
	for _, op := range fn.Ops {
		if op.Kind == OpCall &&
			(op.Note == "core.slice_window_i32" || op.Note == "core.slice_copy_i32") {
			t.Fatalf(
				"known slice builtin was also recorded as unknown call: %#v\n%s",
				op,
				FormatText(prog),
			)
		}
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"fact borrowed_imm value: view:borrowed",
		"fact no_escape value: view:borrowed",
		"fact owned value: alloc_intent:copied",
		"fact provenance_known value: alloc_intent:copied",
		"fact derived_window value: view:borrowed",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramPreservesIslandViewAndOwnedCopyFacts(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 4)
        let view: []u8 = xs.window(0, 2)
        let borrowed: []u8 = view.borrow()
        let copied: []u8 = borrowed.copy()
        return copied.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findPLIRFunction(t, prog, "main")
	xs := findPLIRAllocValue(t, fn, "xs")
	if xs.Provenance.Kind != ProvenanceIsland {
		t.Fatalf("island allocation provenance = %+v, want island", xs.Provenance)
	}
	view := findPLIRValue(t, fn, "view:view")
	if view.Provenance.Kind != ProvenanceIsland || view.Escape != EscapeNoEscape {
		t.Fatalf("island view = %+v, want island no_escape view", view)
	}
	borrowed := findPLIRValue(t, fn, "view:borrowed")
	if borrowed.Provenance.Kind != ProvenanceIsland || borrowed.Borrow != BorrowImm ||
		borrowed.Escape != EscapeNoEscape {
		t.Fatalf("borrowed island view = %+v, want borrowed island no_escape view", borrowed)
	}
	copied := findPLIRAllocValue(t, fn, "copied")
	if copied.Provenance.Kind != ProvenanceAllocation || copied.Borrow != BorrowNone {
		t.Fatalf("copy from island = %+v, want owned allocation provenance", copied)
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"fact provenance_known value: alloc_intent:xs",
		"fact borrowed_imm value: view:view",
		"fact borrowed_imm value: view:borrowed",
		"fact owned value: alloc_intent:copied",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramRecordsIndexStoreFacts(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, io, mem:
    var xs: []u8 = make_u8(2)
    xs[1] = 42
    print(xs)
    return xs[1]
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findPLIRFunction(t, prog, "main")
	var sawStore bool
	var sawPrint bool
	for _, op := range fn.Ops {
		if op.Kind == OpIndexStore {
			sawStore = true
			if len(op.Inputs) != 2 || op.Inputs[0] != "xs" || op.Inputs[1] != "1" {
				t.Fatalf("index store inputs = %#v, want xs/1\n%s", op.Inputs, FormatText(prog))
			}
		}
		if op.Kind == OpPrint {
			sawPrint = true
			if len(op.Inputs) != 1 || op.Inputs[0] != "xs" {
				t.Fatalf("print inputs = %#v, want xs\n%s", op.Inputs, FormatText(prog))
			}
		}
	}
	if !sawStore {
		t.Fatalf("PLIR dump missing index_store:\n%s", FormatText(prog))
	}
	if !sawPrint {
		t.Fatalf("PLIR dump missing print:\n%s", FormatText(prog))
	}
}

func TestFromCheckedProgramRecordsEffectOptimizationFacts(t *testing.T) {
	checked := checkedProgram(t, `
func add(x: Int, y: Int) -> Int:
    return x + y

func log(xs: []u8) -> Int
uses io:
    print(xs)
    return 0

func main() -> Int
uses alloc, io, mem:
    var xs: []u8 = make_u8(1)
    log(xs)
    return add(20, 22)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	add := findPLIRFunction(t, prog, "add")
	for _, want := range []FactKind{
		FactPureCall,
		FactNoHeapAllocation,
		FactNoMemWrite,
		FactNoActorSend,
		FactNoUnknownEscape,
	} {
		if !add.HasFact(want) {
			t.Fatalf("add facts missing %s: %#v\n%s", want, add.Facts, FormatText(prog))
		}
	}
	log := findPLIRFunction(t, prog, "log")
	for _, want := range []FactKind{FactNoHeapAllocation, FactNoMemWrite, FactNoActorSend} {
		if !log.HasFact(want) {
			t.Fatalf("log facts missing %s: %#v\n%s", want, log.Facts, FormatText(prog))
		}
	}
	for _, forbidden := range []FactKind{FactPureCall, FactNoUnknownEscape} {
		if log.HasFact(forbidden) {
			t.Fatalf("log unexpectedly has %s: %#v\n%s", forbidden, log.Facts, FormatText(prog))
		}
	}
	mainFn := findPLIRFunction(t, prog, "main")
	for _, forbidden := range []FactKind{FactNoHeapAllocation, FactNoMemWrite, FactPureCall} {
		if mainFn.HasFact(forbidden) {
			t.Fatalf("main unexpectedly has %s: %#v\n%s", forbidden, mainFn.Facts, FormatText(prog))
		}
	}
}

func TestBorrowFromSimpleAliasPreservesDerivedWindowFact(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(3)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    let view: []i32 = xs.window(1, 2)
    let alias: []i32 = view
    let borrowed: []i32 = alias.borrow()
    return borrowed.len
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"fact derived_window value: view:alias",
		"fact derived_window value: view:borrowed",
		"fact no_escape value: view:borrowed",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramRecordsLocalViewChainForRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func sum_chain(xs: []i32) -> Int
uses mem:
    let view: []i32 = xs.prefix(4).suffix(1)
    var total = 0
    for x in view:
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    return sum_chain(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "sum_chain")
	dump := FormatText(prog)
	for _, want := range []string{
		"fact derived_window value: view:view range: xs[1..4]",
		"fact index_in_range",
		"proof proof:for-collection:x:",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
	if len(fn.ProofGuards) != 1 || len(fn.ProofUses) != 1 {
		t.Fatalf(
			"view-chain loop proof guards/uses = %d/%d, want 1/1\n%s",
			len(fn.ProofGuards),
			len(fn.ProofUses),
			dump,
		)
	}
	if fn.ProofGuards[0].ID != fn.ProofUses[0].ProofID {
		t.Fatalf(
			"view-chain proof guard/use mismatch: %#v vs %#v",
			fn.ProofGuards[0],
			fn.ProofUses[0],
		)
	}
}

func TestFromCheckedProgramComposesViewChainDerivedWindowRange(t *testing.T) {
	checked := checkedProgram(t, `
func chain_range(xs: []i32) -> Int
uses mem:
    let a: []i32 = xs.window(1, 5)
    let b: []i32 = a.prefix(4)
    let c: []i32 = b.suffix(1)
    return c.len

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(8)
    return chain_range(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"fact derived_window value: view:a range: xs[1..6]",
		"fact derived_window value: view:b range: xs[1..5]",
		"fact derived_window value: view:c range: xs[2..5]",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing composed range %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramDoesNotProveInvalidStringViewLoop(t *testing.T) {
	checked := checkedProgram(t, `
func sum_bad() -> Int:
    var total = 0
    for ch in core.string_window("abc", 4, 0):
        total = total + ch
    return total

func main() -> Int:
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findPLIRFunction(t, prog, "sum_bad")
	for _, fact := range fn.Facts {
		if fact.Kind == FactIndexInRange || fact.Kind == FactDerivedWindow ||
			fact.Kind == FactLenStable {
			t.Fatalf(
				"invalid String view loop received false fact: %#v\n%s",
				fact,
				FormatText(prog),
			)
		}
	}
}

func TestFromCheckedProgramDoesNotProveInvalidIntermediateViewChain(t *testing.T) {
	checked := checkedProgram(t, `
func sum_bad_chain() -> Int:
    let view: String = core.string_suffix(core.string_window("abc", 4, 0), 0)
    var total = 0
    for ch in view:
        total = total + ch
    return total

func main() -> Int:
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "sum_bad_chain")
	for _, fact := range fn.Facts {
		if fact.Kind == FactIndexInRange || fact.Kind == FactDerivedWindow ||
			fact.Kind == FactLenStable {
			t.Fatalf(
				"invalid intermediate view chain received false fact: %#v\n%s",
				fact,
				FormatText(prog),
			)
		}
	}
	if len(fn.ProofGuards) != 0 || len(fn.ProofUses) != 0 {
		t.Fatalf(
			"invalid intermediate view chain received proof guards/uses: %#v %#v\n%s",
			fn.ProofGuards,
			fn.ProofUses,
			FormatText(prog),
		)
	}
}

func TestFromCheckedProgramDoesNotProveRawDerivedViewChain(t *testing.T) {
	checked := checkedProgram(t, `
func sum_raw_chain(xs: []u8) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let raw: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
        let view: []u8 = raw.prefix(1).suffix(0)
        var total = 0
        for x in view:
            total = total + x
        return total
    return 0

func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(1)
    return sum_raw_chain(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "sum_raw_chain")
	for _, fact := range fn.Facts {
		if fact.Kind == FactIndexInRange {
			t.Fatalf(
				"raw-derived view chain received range/len proof fact: %#v\n%s",
				fact,
				FormatText(prog),
			)
		}
		if fact.Kind == FactLenStable &&
			(strings.HasPrefix(
				fact.ValueID,
				"view:",
			) || fact.ValueID == "local:view" || fact.ValueID == "local:raw") {
			t.Fatalf(
				"raw-derived view chain received view len_stable fact: %#v\n%s",
				fact,
				FormatText(prog),
			)
		}
	}
	if len(fn.ProofGuards) != 0 || len(fn.ProofUses) != 0 {
		t.Fatalf(
			"raw-derived view chain received proof guards/uses: %#v %#v\n%s",
			fn.ProofGuards,
			fn.ProofUses,
			FormatText(prog),
		)
	}
}

func findPLIRFunction(t *testing.T, prog *Program, name string) Function {
	t.Helper()
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing PLIR function %s: %#v", name, prog.Funcs)
	return Function{}
}

func findPLIRAllocValue(t *testing.T, fn Function, name string) Value {
	t.Helper()
	want := valueID(ValueAllocIntent, name)
	for _, value := range fn.Values {
		if value.ID == want {
			if value.Alloc == nil {
				t.Fatalf("%s has nil allocation intent", want)
			}
			return value
		}
	}
	t.Fatalf("missing PLIR alloc value %s in %s: %#v", want, fn.Name, fn.Values)
	return Value{}
}

func findPLIRValue(t *testing.T, fn Function, id string) Value {
	t.Helper()
	for _, value := range fn.Values {
		if value.ID == id {
			return value
		}
	}
	t.Fatalf("missing PLIR value %s in %s: %#v", id, fn.Name, fn.Values)
	return Value{}
}

func findOperationForOutput(fn Function, output string) (Operation, bool) {
	for _, op := range fn.Ops {
		if containsString(op.Outputs, output) {
			return op, true
		}
	}
	return Operation{}, false
}

func findOperationNoteContaining(fn Function, needle string) (Operation, bool) {
	for _, op := range fn.Ops {
		if strings.Contains(op.Note, needle) {
			return op, true
		}
	}
	return Operation{}, false
}

func assertUnsafeNoteContains(t *testing.T, fn Function, needles ...string) {
	t.Helper()
	for _, op := range fn.Ops {
		if op.Kind != OpUnsafe {
			continue
		}
		haystack := op.Note + " " + strings.Join(
			op.Inputs,
			" ",
		) + " " + strings.Join(
			op.Outputs,
			" ",
		)
		matches := true
		for _, needle := range needles {
			if !strings.Contains(haystack, needle) {
				matches = false
				break
			}
		}
		if matches {
			return
		}
	}
	t.Fatalf(
		"missing unsafe op note containing %v:\n%s",
		needles,
		FormatText(&Program{Funcs: []Function{fn}}),
	)
}
