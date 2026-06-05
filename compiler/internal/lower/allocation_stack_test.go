package lower

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func lowerStackAllocationProgram(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	out, err := LowerWithOptions(checked, Options{StackAllocationLowering: true})
	if err != nil {
		t.Fatalf("LowerWithOptions: %v", err)
	}
	return out
}

func lowerFunctionTempRegionProgram(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	out, err := LowerWithOptions(checked, Options{StackAllocationLowering: true, FunctionTempRegionLowering: true})
	if err != nil {
		t.Fatalf("LowerWithOptions: %v", err)
	}
	return out
}

func TestLowerStackAllocationForFixedNoEscapeSlice(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 10
    xs[1] = 11
    return xs[0] + xs[1]
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRStackSliceI32) != 1 {
		t.Fatalf("main stack slice count = %d, want 1: %#v", countInstrKind(fn, ir.IRStackSliceI32), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRMakeSliceI32) != 0 {
		t.Fatalf("main still contains heap make slice: %#v", fn.Instrs)
	}
}

func TestLowerKeepsEscapingAllocationOnHeap(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func ret() -> []i32
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    return xs

func main() -> Int
uses alloc, mem:
    var xs: []i32 = ret()
    return xs.len
`)
	fn := findIRFunc(t, prog, "ret")
	if countInstrKind(fn, ir.IRStackSliceI32) != 0 {
		t.Fatalf("escaping allocation stack-lowered: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRMakeSliceI32) != 1 {
		t.Fatalf("escaping allocation make slice count = %d, want 1: %#v", countInstrKind(fn, ir.IRMakeSliceI32), fn.Instrs)
	}
}

func TestLowerBorrowedViewOverStackAllocationHasNoNewAllocation(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[1] = 20
    xs[2] = 22
    let mid: []i32 = xs.window(1, 2).borrow()
    return mid[0] + mid[1]
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRStackSliceI32) != 1 {
		t.Fatalf("main stack slice count = %d, want 1 backing allocation: %#v", countInstrKind(fn, ir.IRStackSliceI32), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRMakeSliceI32) != 0 || countInstrKind(fn, ir.IRAllocBytes) != 0 {
		t.Fatalf("borrowed view introduced allocation IR: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRSliceWindow) != 1 {
		t.Fatalf("borrowed window count = %d, want 1: %#v", countInstrKind(fn, ir.IRSliceWindow), fn.Instrs)
	}
}

func TestLowerNonEscapingCopyOfStackViewUsesStackStorage(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[0] = 20
    xs[1] = 22
    let copied: []u8 = xs.window(0, 2).copy()
    return copied[0] + copied[1]
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRStackSliceU8) != 1 {
		t.Fatalf("main u8 stack slice count = %d, want only source stack slice after copy scalar replacement: %#v", countInstrKind(fn, ir.IRStackSliceU8), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRMakeSliceU8) != 0 {
		t.Fatalf("copy still heap-lowers through make_u8: %#v", fn.Instrs)
	}
}

func TestLowerCopyScalarReplacementRequiresDirectConstantUses(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[0] = 20
    xs[1] = 22
    let copied: []u8 = xs.window(0, 2).copy()
    let alias: []u8 = copied
    return alias[0]
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRStackSliceU8) != 2 {
		t.Fatalf("aliased copy u8 stack slice count = %d, want source plus copied stack storage: %#v", countInstrKind(fn, ir.IRStackSliceU8), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRMakeSliceU8) != 0 {
		t.Fatalf("aliased copy fell back to heap make_u8 instead of stack fallback: %#v", fn.Instrs)
	}
}

func TestLowerUnusedCopyEliminatesFreshAllocation(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[0] = 20
    let unused: []u8 = xs.copy()
    return xs[0]
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRStackSliceU8) != 1 {
		t.Fatalf("main u8 stack slice count = %d, want only source stack slice: %#v", countInstrKind(fn, ir.IRStackSliceU8), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRMakeSliceU8) != 0 || countInstrKind(fn, ir.IRAllocBytes) != 0 {
		t.Fatalf("unused copy still emitted fresh allocation IR: %#v", fn.Instrs)
	}
}

func TestLowerFunctionTempRegionCopyEmitsEnterMakeAndReset(t *testing.T) {
	prog := lowerFunctionTempRegionProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 2
    var xs: []u8 = make_u8(8)
    xs[0] = 20
    let copied: []u8 = xs.window(0, n).copy()
    return copied.len
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRRegionEnter) != 1 {
		t.Fatalf("main region enter count = %d, want 1: %#v", countInstrKind(fn, ir.IRRegionEnter), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRRegionMakeSliceU8) != 1 {
		t.Fatalf("main region make slice count = %d, want 1: %#v", countInstrKind(fn, ir.IRRegionMakeSliceU8), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRRegionReset) != 1 {
		t.Fatalf("main region reset count = %d, want 1: %#v", countInstrKind(fn, ir.IRRegionReset), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRMakeSliceU8) != 0 {
		t.Fatalf("region copy still heap-lowers through make_u8: %#v", fn.Instrs)
	}
	resetAt := firstInstrKind(fn, ir.IRRegionReset)
	returnAt := firstInstrKind(fn, ir.IRReturn)
	if resetAt < 0 || returnAt < 0 || resetAt > returnAt {
		t.Fatalf("region reset must dominate return: reset=%d return=%d instrs=%#v", resetAt, returnAt, fn.Instrs)
	}
}

func TestLowerExplicitIslandMakeSliceCarriesAllocationName(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u16 = core.island_make_u16(isl, 2)
        xs[0] = 7
        return xs[0]
    return 0
`)
	fn := findIRFunc(t, prog, "main")
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRIslandMakeSliceU16 {
			if instr.Name != "xs" {
				t.Fatalf("island make slice allocation name = %q, want xs: %#v", instr.Name, fn.Instrs)
			}
			return
		}
	}
	t.Fatalf("main missing island make slice: %#v", fn.Instrs)
}

func TestLowerCopyIntoChecksDestinationBeforeWriting(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func main() -> Int
uses alloc, mem:
    var src: []u8 = make_u8(4)
    var dst: []u8 = make_u8(2)
    return src.copy_into(dst)
`)
	fn := findIRFunc(t, prog, "main")
	prefixAt := -1
	storeAt := -1
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRSlicePrefix && prefixAt < 0 {
			prefixAt = i
		}
		if instr.Kind == ir.IRIndexStoreU8 && storeAt < 0 {
			storeAt = i
		}
	}
	if prefixAt < 0 || storeAt < 0 {
		t.Fatalf("copy_into IR missing prefix guard or store: prefix=%d store=%d instrs=%#v", prefixAt, storeAt, fn.Instrs)
	}
	if prefixAt > storeAt {
		t.Fatalf("copy_into writes before destination length guard: prefix=%d store=%d instrs=%#v", prefixAt, storeAt, fn.Instrs)
	}
}

func TestLowerScalarReplacementEliminatesTinyConstantIndexSlice(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 20
    xs[1] = 22
    return xs[0] + xs[1]
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRStackSliceI32) != 0 || countInstrKind(fn, ir.IRMakeSliceI32) != 0 || countInstrKind(fn, ir.IRAllocBytes) != 0 {
		t.Fatalf("scalar replacement still emitted allocation IR: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 || countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 || countInstrKind(fn, ir.IRIndexStoreI32) != 0 {
		t.Fatalf("scalar replacement still emitted indexed memory IR: %#v", fn.Instrs)
	}
}

func TestLowerSmallStructUsesScalarSlotsWithoutAllocatorIR(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
struct Pair:
    left: Int
    right: Int

func main() -> Int:
    let p: Pair = Pair(left: 20, right: 22)
    return p.left + p.right
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRAllocBytes) != 0 ||
		countInstrKind(fn, ir.IRMakeSliceU8) != 0 ||
		countInstrKind(fn, ir.IRMakeSliceU16) != 0 ||
		countInstrKind(fn, ir.IRMakeSliceI32) != 0 {
		t.Fatalf("small struct scalar-slot lowering emitted allocator IR: %#v", fn.Instrs)
	}
	if fn.LocalSlots < 2 {
		t.Fatalf("small struct should occupy scalar local slots, got LocalSlots=%d instrs=%#v", fn.LocalSlots, fn.Instrs)
	}
}

func TestLowerSmallFixedArrayUsesScalarSlotsWithoutAllocatorIR(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func touch(seed: [2]Int) -> Int:
    var xs: [2]Int = seed
    xs[0] = 20
    xs[1] = 22
    return xs[0] + xs[1]

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "touch")
	if countInstrKind(fn, ir.IRAllocBytes) != 0 ||
		countInstrKind(fn, ir.IRMakeSliceU8) != 0 ||
		countInstrKind(fn, ir.IRMakeSliceU16) != 0 ||
		countInstrKind(fn, ir.IRMakeSliceI32) != 0 {
		t.Fatalf("small fixed-array scalar-slot lowering emitted allocator IR: %#v", fn.Instrs)
	}
	if fn.LocalSlots < 2 {
		t.Fatalf("small fixed array should occupy scalar local slots, got LocalSlots=%d instrs=%#v", fn.LocalSlots, fn.Instrs)
	}
}

func TestLowerDynamicIndexKeepsTinySliceChecked(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func main() -> Int
uses alloc, mem:
    var i = 0
    var xs: []i32 = make_i32(2)
    xs[i] = 42
    return xs[i]
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRStackSliceI32) != 1 {
		t.Fatalf("dynamic-index tiny slice stack slice count = %d, want 1: %#v", countInstrKind(fn, ir.IRStackSliceI32), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 || countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("dynamic-index tiny slice should keep checked index IR: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("dynamic-index tiny slice received unchecked load without proof: %#v", fn.Instrs)
	}
}

func firstInstrKind(fn ir.IRFunc, kind ir.IRInstrKind) int {
	for i, instr := range fn.Instrs {
		if instr.Kind == kind {
			return i
		}
	}
	return -1
}
