package lower

import (
	"strconv"
	"strings"
	"testing"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/target"
)

// ---- actor_state_test.go ----

func TestLowerActorStateUsesRuntimeLoadStore(t *testing.T) {
	src := []byte(`
actor Counter:
    var count: Int = 1
    val enabled: Bool = true
    func run() -> Int:
        if enabled:
            count = count + 1
        return count

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Counter.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	run := findIRFuncByName(t, irProg.Funcs, "Counter.run")
	if !hasIRCallName(run, "__tetra_actor_state_load") {
		t.Fatalf("Counter.run is missing __tetra_actor_state_load call: %#v", run.Instrs)
	}
	if !hasIRCallName(run, "__tetra_actor_state_store") {
		t.Fatalf("Counter.run is missing __tetra_actor_state_store call: %#v", run.Instrs)
	}
}

func TestLowerActorStateExtendedScalarsUseRuntimeLoadStore(t *testing.T) {
	src := []byte(`
actor Counter:
    var err: task.error = 0
    var step: UInt8 = 1
    const boost: UInt16 = 2
    func run() -> Int:
        err = err + 1
        step = step + 1
        return err + step + boost

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Counter.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	run := findIRFuncByName(t, irProg.Funcs, "Counter.run")
	if !hasIRCallName(run, "__tetra_actor_state_load") {
		t.Fatalf("Counter.run is missing __tetra_actor_state_load call: %#v", run.Instrs)
	}
	if !hasIRCallName(run, "__tetra_actor_state_store") {
		t.Fatalf("Counter.run is missing __tetra_actor_state_store call: %#v", run.Instrs)
	}
}

func findIRFuncByName(t *testing.T, funcs []ir.IRFunc, name string) ir.IRFunc {
	t.Helper()
	for _, fn := range funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing IR function %q", name)
	return ir.IRFunc{}
}

func hasIRCallName(fn ir.IRFunc, name string) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			return true
		}
	}
	return false
}

// ---- allocation_stack_test.go ----

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
	out, err := LowerWithOptions(
		checked,
		Options{StackAllocationLowering: true, FunctionTempRegionLowering: true},
	)
	if err != nil {
		t.Fatalf("LowerWithOptions: %v", err)
	}
	return out
}

func lowerOwnedAllocDropProgram(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	out, err := LowerWithOptions(checked, Options{OwnedAllocDropLowering: true})
	if err != nil {
		t.Fatalf("LowerWithOptions: %v", err)
	}
	return out
}

func lowerOwnedAllocDropProgramError(t *testing.T, src string) error {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	_, err = LowerWithOptions(checked, Options{OwnedAllocDropLowering: true})
	return err
}

func lowerOwnedAllocDropFileProgram(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "owned_alloc_drop.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	checked, err := semantics.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	out, err := LowerWithOptions(checked, Options{OwnedAllocDropLowering: true})
	if err != nil {
		t.Fatalf("LowerWithOptions: %v", err)
	}
	return out
}

func lowerOwnedAllocDropFileProgramError(t *testing.T, src string) error {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "owned_alloc_drop.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	checked, err := semantics.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	_, err = LowerWithOptions(checked, Options{OwnedAllocDropLowering: true})
	return err
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
		t.Fatalf(
			"main stack slice count = %d, want 1: %#v",
			countInstrKind(fn, ir.IRStackSliceI32),
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRMakeSliceI32) != 0 {
		t.Fatalf("main still contains heap make slice: %#v", fn.Instrs)
	}
}

func TestLowerSliceSumLargeNoEscapeI32SliceUsesStack(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    var r: Int = 0
    while r < 64:
        i = 0
        while i < n:
            total = total + xs[i]
            i = i + 1
        r = r + 1
    if total > 0:
        return 0
    return 1
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRStackSliceI32) != 1 {
		t.Fatalf(
			"slice_sum stack slice count = %d, want 1: %#v",
			countInstrKind(fn, ir.IRStackSliceI32),
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRMakeSliceI32) != 0 {
		t.Fatalf("slice_sum still contains heap make_i32: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRAllocBytes) != 0 {
		t.Fatalf("slice_sum emitted allocator IR for stack-lowered i32 slice: %#v", fn.Instrs)
	}
}

func TestLowerStackAllocationForImmutableLengthReadOnlyLocalCall(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 4
    var keys: []i32 = make_i32(n)
    var values: []i32 = make_i32(n)
    return lookup(keys, values, n, 2)
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRStackSliceI32) != 2 {
		t.Fatalf(
			"main stack slice count = %d, want 2: %#v",
			countInstrKind(fn, ir.IRStackSliceI32),
			fn.Instrs,
		)
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
		t.Fatalf(
			"escaping allocation make slice count = %d, want 1: %#v",
			countInstrKind(fn, ir.IRMakeSliceI32),
			fn.Instrs,
		)
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
		t.Fatalf(
			"main stack slice count = %d, want 1 backing allocation: %#v",
			countInstrKind(fn, ir.IRStackSliceI32),
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRMakeSliceI32) != 0 || countInstrKind(fn, ir.IRAllocBytes) != 0 {
		t.Fatalf("borrowed view introduced allocation IR: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRSliceWindow) != 1 {
		t.Fatalf(
			"borrowed window count = %d, want 1: %#v",
			countInstrKind(fn, ir.IRSliceWindow),
			fn.Instrs,
		)
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
		t.Fatalf(
			("main u8 stack slice count = %d, want only source stack slice " +
				"after copy scalar replacement: %#v"),
			countInstrKind(fn, ir.IRStackSliceU8),
			fn.Instrs,
		)
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
		t.Fatalf(
			"aliased copy u8 stack slice count = %d, want source plus copied stack storage: %#v",
			countInstrKind(fn, ir.IRStackSliceU8),
			fn.Instrs,
		)
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
		t.Fatalf(
			"main u8 stack slice count = %d, want only source stack slice: %#v",
			countInstrKind(fn, ir.IRStackSliceU8),
			fn.Instrs,
		)
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
		t.Fatalf(
			"main region enter count = %d, want 1: %#v",
			countInstrKind(fn, ir.IRRegionEnter),
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRRegionMakeSliceU8) != 1 {
		t.Fatalf(
			"main region make slice count = %d, want 1: %#v",
			countInstrKind(fn, ir.IRRegionMakeSliceU8),
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRRegionReset) != 1 {
		t.Fatalf(
			"main region reset count = %d, want 1: %#v",
			countInstrKind(fn, ir.IRRegionReset),
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRMakeSliceU8) != 0 {
		t.Fatalf("region copy still heap-lowers through make_u8: %#v", fn.Instrs)
	}
	resetAt := firstInstrKind(fn, ir.IRRegionReset)
	returnAt := firstInstrKind(fn, ir.IRReturn)
	if resetAt < 0 || returnAt < 0 || resetAt > returnAt {
		t.Fatalf(
			"region reset must dominate return: reset=%d return=%d instrs=%#v",
			resetAt,
			returnAt,
			fn.Instrs,
		)
	}
}

func TestLowerFunctionTempRegionCopyResetsBeforeBranchReturns(t *testing.T) {
	prog := lowerFunctionTempRegionProgram(t, `
func branchy(flag: Bool) -> Int
uses alloc, mem:
    let n: Int = 2
    var xs: []u8 = make_u8(8)
    xs[0] = 20
    let copied: []u8 = xs.window(0, n).copy()
    if flag:
        return copied.len
    return copied[0]

func main() -> Int
uses alloc, mem:
    return branchy(true)
`)
	fn := findIRFunc(t, prog, "branchy")
	if countInstrKind(fn, ir.IRRegionEnter) != 1 ||
		countInstrKind(fn, ir.IRRegionMakeSliceU8) != 1 {
		t.Fatalf("main missing function-temp region enter/make evidence: %#v", fn.Instrs)
	}
	assertRegionResetImmediatelyBeforeEveryReturn(t, fn)
}

func TestLowerFunctionTempRegionCopyResetsBeforeThrow(t *testing.T) {
	prog := lowerFunctionTempRegionProgram(t, `
enum E:
    case bad

func fail(flag: Bool) -> Int throws E
uses alloc, mem:
    let n: Int = 2
    var xs: []u8 = make_u8(8)
    xs[0] = 20
    let copied: []u8 = xs.window(0, n).copy()
    if flag:
        throw E.bad
    return copied.len

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "fail")
	if countInstrKind(fn, ir.IRRegionEnter) != 1 ||
		countInstrKind(fn, ir.IRRegionMakeSliceU8) != 1 {
		t.Fatalf("fail missing function-temp region enter/make evidence: %#v", fn.Instrs)
	}
	assertRegionResetImmediatelyBeforeEveryReturn(t, fn)
}

func TestLowerOwnedAllocBytesDropsBeforeReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let _stored: Int = core.store_i32(p, 42, mem)
    return 0
`)
	fn := findIRFunc(t, prog, "main")

	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("main alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 1 || countInstrKind(fn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("main missing owned drop/release before return: %#v", fn.Instrs)
	}
	assertOwnedReleaseBeforeFirstReturnValue(t, fn)
}

func TestLowerOwnedAllocBytesInBranchDropsBeforeJoin(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func maybe(flag: Bool) -> Int
uses alloc, capability, mem:
    if flag:
        unsafe:
            let mem: cap.mem = core.cap_mem()
            let p: ptr = core.alloc_bytes(16)
            let _stored: Int = core.store_i32(p, 42, mem)
    return 0

func main() -> Int
uses alloc, capability, mem:
    return maybe(false)
`)
	fn := findIRFunc(t, prog, "maybe")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 1 || countInstrKind(fn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("maybe missing owned drop/release in branch: %#v", fn.Instrs)
	}
	assertOwnedReleaseBeforeFirstJmpIfZeroTarget(t, fn)
}

func TestLowerOwnedAllocBytesSuppressesDropForReturnedPointer(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func ret() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "ret")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("ret alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned owned pointer must not be dropped in callee: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsAlwaysSomeReturnedOptionalOwnedPayloadInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func maybe() -> ptr?
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func main() -> Int
uses alloc, mem:
    let q: ptr? = maybe()
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned optional owned payload must not be dropped in callee: %#v", maybeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "maybe") != 1 {
		t.Fatalf("main must call maybe once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned optional payload must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsMixedReturnedOptionalOwnedPayloadInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func maybe(flag: Bool) -> ptr?
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            return p
    return none

func main() -> Int
uses alloc, mem:
    let q: ptr? = maybe(true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned optional owned payload must not be dropped in callee: %#v", maybeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "maybe") != 1 {
		t.Fatalf("main must call maybe once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned optional payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocal(t, mainFn, 1)
}

func TestLowerOwnedAllocBytesDropsRelayedMixedReturnedOptionalOwnedPayloadInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func maybe(flag: Bool) -> ptr?
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            return p
    return none

func relay(flag: Bool) -> ptr?
uses alloc, mem:
    return maybe(flag)

func main() -> Int
uses alloc, mem:
    let q: ptr? = relay(true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned optional owned payload must not be dropped in source callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("relayed optional owned payload must not be dropped in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned relayed optional payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocal(t, mainFn, 1)
}

func TestLowerOwnedAllocBytesDropsLocalRelayedMixedReturnedOptionalOwnedPayloadInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func maybe(flag: Bool) -> ptr?
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            return p
    return none

func relay(flag: Bool) -> ptr?
uses alloc, mem:
    let q: ptr? = maybe(flag)
    return q

func main() -> Int
uses alloc, mem:
    let q: ptr? = relay(true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned optional owned payload must not be dropped in source callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("local relayed optional owned payload must not be dropped in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned local relayed optional payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocal(t, mainFn, 1)
}

func TestLowerOwnedAllocBytesSuppressesDropForThrownPointer(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func fail() -> Int throws ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        throw p

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "fail")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("thrown owned pointer must not be dropped in throwing callee: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsBeforeTryPropagationReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum E:
    case bad

func fail(flag: Bool) -> Int throws E:
    if flag:
        throw E.bad
    return 7

func caller(flag: Bool) -> Int throws E
uses alloc, capability, mem:
    var p: ptr = 0
    unsafe:
        let mem: cap.mem = core.cap_mem()
        p = core.alloc_bytes(16)
        let _stored: Int = core.store_i32(p, 42, mem)
    return try fail(flag)

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "caller")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("caller alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 2 || countInstrKind(fn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("try propagation and success returns must both drop/release owned allocation: %#v", fn.Instrs)
	}
	assertOwnedReleaseBeforeEveryReturn(t, fn)
}

func TestLowerOwnedAllocBytesDropsPartialInlineAggregateBeforeTryPropagation(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum E:
    case bad

struct FailableBox:
    raw: ptr
    code: Int

func fail(flag: Bool) -> Int throws E:
    if flag:
        throw E.bad
    return 7

func make_box(flag: Bool) -> FailableBox throws E
uses alloc, mem:
    unsafe:
        return FailableBox(raw: core.alloc_bytes(16), code: try fail(flag))

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "make_box")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 1 || countInstrKind(fn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("partial aggregate try failure must drop/release initialized owned field: %#v", fn.Instrs)
	}
	assertOwnedReleaseBeforeFirstReturn(t, fn)
}

func TestLowerOwnedAllocBytesDropsNestedPartialInlineAggregateBeforeTryPropagation(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum E:
    case bad

struct PtrBox:
    raw: ptr
    code: Int

struct Holder:
    head: Int
    box: PtrBox

func fail(flag: Bool) -> Int throws E:
    if flag:
        throw E.bad
    return 7

func make_holder(flag: Bool) -> Holder throws E
uses alloc, mem:
    unsafe:
        return Holder(head: 1, box: PtrBox(raw: core.alloc_bytes(16), code: try fail(flag)))

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "make_holder")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_holder alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 1 || countInstrKind(fn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("nested partial aggregate try failure must drop/release initialized owned field: %#v", fn.Instrs)
	}
	assertOwnedReleaseBeforeFirstReturn(t, fn)
}

func TestLowerOwnedAllocBytesDropsPartialInlineEnumPayloadBeforeTryPropagation(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum E:
    case bad

enum PtrResult:
    case owned(ptr, Int)

func fail(flag: Bool) -> Int throws E:
    if flag:
        throw E.bad
    return 7

func make_result(flag: Bool) -> PtrResult throws E
uses alloc, mem:
    unsafe:
        return PtrResult.owned(core.alloc_bytes(16), try fail(flag))

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "make_result")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 1 || countInstrKind(fn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("partial enum payload try failure must drop/release initialized owned payload: %#v", fn.Instrs)
	}
	assertOwnedReleaseBeforeFirstReturn(t, fn)
}

func TestLowerOwnedAllocBytesDropsPartialInlineAggregateThrowBeforeTryPropagation(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct FailableBox:
    raw: ptr
    code: Int

func fail_code(flag: Bool) -> Int throws FailableBox:
    if flag:
        throw FailableBox(raw: 0, code: 1)
    return 7

func fail(flag: Bool) -> Int throws FailableBox
uses alloc, mem:
    unsafe:
        throw FailableBox(raw: core.alloc_bytes(16), code: try fail_code(flag))

func main() -> Int
uses alloc, mem:
    return catch fail(true):
    case _:
        9
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 1 || countInstrKind(failFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("partial aggregate throw try failure must drop/release initialized owned field: %#v", failFn.Instrs)
	}
	assertOwnedReleaseBeforeFirstReturn(t, failFn)
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "fail") != 1 {
		t.Fatalf("main must call fail once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caught partial aggregate throw must be dropped in catch path: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsNestedPartialInlineAggregateThrowBeforeTryPropagation(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr
    code: Int

struct Holder:
    head: Int
    box: PtrBox

func fail_box(flag: Bool) -> Int throws Holder:
    if flag:
        throw Holder(head: 0, box: PtrBox(raw: 0, code: 1))
    return 7

func fail(flag: Bool) -> Int throws Holder
uses alloc, mem:
    unsafe:
        throw Holder(head: 1, box: PtrBox(raw: core.alloc_bytes(16), code: try fail_box(flag)))

func main() -> Int
uses alloc, mem:
    return catch fail(true):
    case _:
        9
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 1 || countInstrKind(failFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("nested partial aggregate throw try failure must drop/release initialized owned field: %#v", failFn.Instrs)
	}
	assertOwnedReleaseBeforeFirstReturn(t, failFn)
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "fail") != 1 {
		t.Fatalf("main must call fail once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caught nested partial aggregate throw must be dropped in catch path: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterLocalMoveReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func ret() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        let q: ptr = p
        return q

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "ret")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("ret alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("moved returned owned pointer must not be dropped in callee: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRejectsMovedFromLocalReturn(t *testing.T) {
	src := `
func ret() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        let q: ptr = p
        return p

func main() -> Int:
    return 0
`
	err := lowerOwnedAllocDropProgramError(t, src)
	if err == nil {
		prog := lowerOwnedAllocDropProgram(t, src)
		retFn := findIRFunc(t, prog, "ret")
		t.Fatalf("LowerWithOptions error = <nil>, want use after move; ret IR: %#v", retFn.Instrs)
	}
	if !strings.Contains(err.Error(), "use after move") {
		t.Fatalf("LowerWithOptions error = %v, want use after move", err)
	}
}

func TestLowerOwnedAllocBytesRejectsAggregateUseAfterFieldMove(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
struct PtrBox:
    tag: Int
    raw: ptr

func ret() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(tag: 1, raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
        let q: ptr = box.raw
        return box

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "use after move") {
		t.Fatalf("LowerWithOptions error = %v, want use after move", err)
	}
}

func TestLowerOwnedAllocBytesRejectsMultiOwnedAggregateReturn(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
struct PairBox:
    left: ptr
    right: ptr

func make_pair() -> PairBox
uses alloc, mem:
    var pair: PairBox = PairBox(left: 0, right: 0)
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        pair.left = left
        pair.right = right
    return pair

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned return slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned return slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsOwnedAggregateIndexStore(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
struct PtrBox:
    raw: ptr

func store_index(seed: [2]PtrBox) -> Int
uses alloc, mem:
    var boxes: [2]PtrBox = seed
    unsafe:
        var box: PtrBox = PtrBox(raw: 0)
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
        boxes[0] = box
    return 0

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "index store") {
		t.Fatalf("LowerWithOptions error = %v, want index store diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsCaughtOwnedPayloadIndexStore(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
struct PtrBox:
    raw: ptr

enum PtrError:
    case owned(ptr)

func fail() -> PtrBox throws PtrError
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        throw PtrError.owned(p)

func store_index(seed: [2]PtrBox) -> Int
uses alloc, mem:
    var boxes: [2]PtrBox = seed
    unsafe:
        boxes[0] = catch fail():
        case PtrError.owned(p):
            PtrBox(raw: p)
    return 0

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "index store") {
		t.Fatalf("LowerWithOptions error = %v, want index store diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesDropsInlineOwnedAggregateReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return PtrBox(raw: p)

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("inline owned aggregate return must not drop in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned inline aggregate return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsInlineDirectAllocAggregateReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    unsafe:
        return PtrBox(raw: core.alloc_bytes(16))

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("inline direct allocation aggregate return must not drop in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned inline direct allocation aggregate return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsInlineOwnedReturnCallAggregateReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_raw() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(16)

func make_box() -> PtrBox
uses alloc, mem:
    return PtrBox(raw: make_raw())

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
    return 0
`)
	makeRawFn := findIRFunc(t, prog, "make_raw")
	if countInstrKind(makeRawFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_raw alloc_bytes count = %d, want 1: %#v", countInstrKind(makeRawFn, ir.IRAllocBytes), makeRawFn.Instrs)
	}
	if countInstrKind(makeRawFn, ir.IRDropOwned) != 0 || countInstrKind(makeRawFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("raw factory must not drop returned pointer in callee: %#v", makeRawFn.Instrs)
	}
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countCallsNamed(makeBoxFn.Instrs, "make_raw") != 1 {
		t.Fatalf("make_box must call make_raw once: %#v", makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("inline owned-return call aggregate return must not drop in wrapper: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned inline owned-return call aggregate return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsNestedInlineOwnedAggregateReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

func make_holder() -> Holder
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return Holder(head: 1, box: PtrBox(tag: 2, raw: p))

func main() -> Int
uses alloc, mem:
    unsafe:
        let holder: Holder = make_holder()
    return 0
`)
	makeHolderFn := findIRFunc(t, prog, "make_holder")
	if countInstrKind(makeHolderFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_holder alloc_bytes count = %d, want 1: %#v", countInstrKind(makeHolderFn, ir.IRAllocBytes), makeHolderFn.Instrs)
	}
	if countInstrKind(makeHolderFn, ir.IRDropOwned) != 0 || countInstrKind(makeHolderFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("nested inline owned aggregate return must not drop in callee: %#v", makeHolderFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_holder") != 1 {
		t.Fatalf("main must call make_holder once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned nested inline aggregate return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsNestedInlineDirectAllocAggregateReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

func make_holder() -> Holder
uses alloc, mem:
    unsafe:
        return Holder(head: 1, box: PtrBox(tag: 2, raw: core.alloc_bytes(16)))

func main() -> Int
uses alloc, mem:
    unsafe:
        let holder: Holder = make_holder()
    return 0
`)
	makeHolderFn := findIRFunc(t, prog, "make_holder")
	if countInstrKind(makeHolderFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_holder alloc_bytes count = %d, want 1: %#v", countInstrKind(makeHolderFn, ir.IRAllocBytes), makeHolderFn.Instrs)
	}
	if countInstrKind(makeHolderFn, ir.IRDropOwned) != 0 || countInstrKind(makeHolderFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("nested inline direct allocation aggregate return must not drop in callee: %#v", makeHolderFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_holder") != 1 {
		t.Fatalf("main must call make_holder once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned nested inline direct allocation aggregate return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsNestedInlineOwnedReturnCallAggregateReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

func make_raw() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(16)

func make_holder() -> Holder
uses alloc, mem:
    return Holder(head: 1, box: PtrBox(tag: 2, raw: make_raw()))

func main() -> Int
uses alloc, mem:
    unsafe:
        let holder: Holder = make_holder()
    return 0
`)
	makeRawFn := findIRFunc(t, prog, "make_raw")
	if countInstrKind(makeRawFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_raw alloc_bytes count = %d, want 1: %#v", countInstrKind(makeRawFn, ir.IRAllocBytes), makeRawFn.Instrs)
	}
	if countInstrKind(makeRawFn, ir.IRDropOwned) != 0 || countInstrKind(makeRawFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("raw factory must not drop returned pointer in callee: %#v", makeRawFn.Instrs)
	}
	makeHolderFn := findIRFunc(t, prog, "make_holder")
	if countCallsNamed(makeHolderFn.Instrs, "make_raw") != 1 {
		t.Fatalf("make_holder must call make_raw once: %#v", makeHolderFn.Instrs)
	}
	if countInstrKind(makeHolderFn, ir.IRDropOwned) != 0 || countInstrKind(makeHolderFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("nested inline owned-return call aggregate return must not drop in wrapper: %#v", makeHolderFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_holder") != 1 {
		t.Fatalf("main must call make_holder once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned nested inline owned-return call aggregate return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsInlineOwnedAggregateThrowInCatch(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func fail() -> Int throws PtrBox
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        throw PtrBox(raw: p)

func main() -> Int
uses alloc, mem:
    return catch fail():
    case _:
        7
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("inline owned aggregate throw must not drop in callee: %#v", failFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "fail") != 1 {
		t.Fatalf("main must call fail once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caught inline owned aggregate throw must be dropped in catch path: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsReturnedEnumOwnedPayloadInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case owned(ptr)

func make_result() -> PtrResult
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return PtrResult.owned(p)

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result()
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("enum owned payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_result") != 1 {
		t.Fatalf("main must call make_result once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned enum payload return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsInlineAllocEnumOwnedPayloadReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case owned(ptr)

func make_result() -> PtrResult
uses alloc, mem:
    unsafe:
        return PtrResult.owned(core.alloc_bytes(16))

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result()
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("inline enum owned payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned inline enum payload return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsLocalEnumOwnedPayload(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case owned(ptr)

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = PtrResult.owned(core.alloc_bytes(16))
    return 0
`)
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("main alloc_bytes count = %d, want 1: %#v", countInstrKind(mainFn, ir.IRAllocBytes), mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("local enum owned payload must be dropped in owner scope: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsAssignedLocalEnumOwnedPayload(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case empty
    case owned(ptr)

func main() -> Int
uses alloc, mem:
    var result: PtrResult = PtrResult.empty
    unsafe:
        result = PtrResult.owned(core.alloc_bytes(16))
    return 0
`)
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("main alloc_bytes count = %d, want 1: %#v", countInstrKind(mainFn, ir.IRAllocBytes), mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("assigned local enum owned payload must be dropped in owner scope: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsLocalEnumOwnedPayloadReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case owned(ptr)

func make_result() -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = PtrResult.owned(core.alloc_bytes(16))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result()
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("local enum payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned local enum payload return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsAssignedLocalEnumOwnedPayloadReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case empty
    case owned(ptr)

func make_result() -> PtrResult
uses alloc, mem:
    var result: PtrResult = PtrResult.empty
    unsafe:
        result = PtrResult.owned(core.alloc_bytes(16))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result()
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("assigned local enum payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned assigned local enum payload return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsBranchAssignedLocalEnumOwnedPayloadReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case empty
    case owned(ptr)

func make_result(flag: Bool) -> PtrResult
uses alloc, mem:
    var result: PtrResult = PtrResult.empty
    unsafe:
        if flag:
            result = PtrResult.owned(core.alloc_bytes(16))
        else:
            result = PtrResult.owned(core.alloc_bytes(32))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(true)
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("make_result alloc_bytes count = %d, want 2: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("branch-assigned local enum payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned branch-assigned local enum payload return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsMatchAssignedLocalEnumOwnedPayloadReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Choice:
    case left
    case right

enum PtrResult:
    case empty
    case owned(ptr)

func make_result(choice: Choice) -> PtrResult
uses alloc, mem:
    var result: PtrResult = PtrResult.empty
    unsafe:
        match choice:
        case Choice.left:
            result = PtrResult.owned(core.alloc_bytes(16))
        case _:
            result = PtrResult.owned(core.alloc_bytes(32))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(Choice.left)
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("make_result alloc_bytes count = %d, want 2: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("match-assigned local enum payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned match-assigned local enum payload return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsConditionalBranchAssignedLocalEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case owned(ptr)
    case empty

func make_result(flag: Int) -> PtrResult
uses alloc, mem:
    var result: PtrResult = PtrResult.empty
    unsafe:
        if flag == 1:
            result = PtrResult.owned(core.alloc_bytes(16))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(1)
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("conditional branch-assigned local enum payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned conditional branch-assigned local enum payload return must be dropped in caller when present: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsMatchResultEnumOwnedPayloadReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case owned(ptr)

func make_result(flag: Int) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = match flag:
        case 0:
            PtrResult.owned(core.alloc_bytes(16))
        case _:
            PtrResult.owned(core.alloc_bytes(32))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(1)
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("make_result alloc_bytes count = %d, want 2: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("match-result enum payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned match-result enum payload return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsGuardedMatchResultEnumOwnedPayloadReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case owned(ptr)

func make_result(flag: Int, prefer: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = match flag:
        case 0 if prefer == true:
            PtrResult.owned(core.alloc_bytes(16))
        case _:
            PtrResult.owned(core.alloc_bytes(32))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(0, false)
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("make_result alloc_bytes count = %d, want 2: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("guarded match-result enum payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded match-result enum payload return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsMixedMatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case empty
    case owned(ptr)

func make_result(flag: Int) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = match flag:
        case 0:
            PtrResult.empty
        case _:
            PtrResult.owned(core.alloc_bytes(16))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(1)
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("mixed match-result enum payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned mixed match-result enum payload return must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocal(t, mainFn, 0)
}

func TestLowerOwnedAllocBytesDropsNonZeroFallbackMixedMatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrResult:
    case owned(ptr)
    case empty

func make_result(flag: Int) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = match flag:
        case 0:
            PtrResult.owned(core.alloc_bytes(16))
        case _:
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(0)
    return 0
`)
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("nonzero fallback mixed match-result enum payload return must not drop in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned nonzero fallback mixed match-result enum payload return must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesRejectsMultiOwnedEnumPayloadThrow(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
enum PairErr:
    case pair(ptr, ptr)

func fail() -> Int throws PairErr
uses alloc, mem:
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        throw PairErr.pair(left, right)

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsNestedMultiOwnedEnumPayloadThrow(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
struct PairBox:
    left: ptr
    right: ptr

enum PairErr:
    case pair(PairBox)

func fail() -> Int throws PairErr
uses alloc, mem:
    var pair: PairBox = PairBox(left: 0, right: 0)
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        pair.left = left
        pair.right = right
        throw PairErr.pair(pair)

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsStaticallyMultiOwnedEnumPayloadThrowWithOneTrackedSlot(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
struct PairBox:
    left: ptr
    right: ptr

enum PairErr:
    case pair(PairBox)

func fail() -> Int throws PairErr
uses alloc, mem:
    var pair: PairBox = PairBox(left: 0, right: 0)
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        pair.left = left
        throw PairErr.pair(pair)

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsLocalStagedMultiOwnedEnumPayloadReturn(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
struct PairBox:
    left: ptr
    right: ptr

enum PairResult:
    case pair(PairBox)

func make_result() -> PairResult
uses alloc, mem:
    var pair: PairBox = PairBox(left: 0, right: 0)
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        pair.left = left
        let result: PairResult = PairResult.pair(pair)
        return result

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsAssignedMultiOwnedEnumPayloadReturn(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
struct PairBox:
    left: ptr
    right: ptr

enum PairResult:
    case pair(PairBox)

func make_result() -> PairResult
uses alloc, mem:
    var pair: PairBox = PairBox(left: 0, right: 0)
    var result: PairResult = PairResult.pair(PairBox(left: 0, right: 0))
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        pair.left = left
        result = PairResult.pair(pair)
        return result

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsGlobalAssignedMultiOwnedEnumPayload(t *testing.T) {
	err := lowerOwnedAllocDropFileProgramError(t, `
struct PairBox:
    left: ptr
    right: ptr

enum PairResult:
    case pair(PairBox)

var saved: PairResult

func store() -> Int
uses alloc, mem:
    var pair: PairBox = PairBox(left: 0, right: 0)
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        pair.left = left
        saved = PairResult.pair(pair)
    return 0

func main() -> Int
uses alloc, mem:
    return store()
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsGlobalFieldAssignedMultiOwnedEnumPayload(t *testing.T) {
	err := lowerOwnedAllocDropFileProgramError(t, `
struct PairBox:
    left: ptr
    right: ptr

enum PairResult:
    case pair(PairBox)

struct Saved:
    result: PairResult

var saved: Saved

func store() -> Int
uses alloc, mem:
    var pair: PairBox = PairBox(left: 0, right: 0)
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        pair.left = left
        saved.result = PairResult.pair(pair)
    return 0

func main() -> Int
uses alloc, mem:
    return store()
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsMatchExprMultiOwnedEnumPayload(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
enum PairResult:
    case pair(ptr, ptr)

func take_left() -> ptr
uses alloc, mem:
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        return match PairResult.pair(left, right):
        case PairResult.pair(payload_left, payload_right):
            left

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsMatchStmtMultiOwnedEnumPayload(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
enum PairResult:
    case pair(ptr, ptr)

func take_left() -> ptr
uses alloc, mem:
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        match PairResult.pair(left, right):
        case PairResult.pair(payload_left, payload_right):
            return left

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsCallArgMultiOwnedEnumPayload(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
enum PairResult:
    case pair(ptr, ptr)

func relay(result: PairResult) -> PairResult:
    return result

func make_result() -> PairResult
uses alloc, mem:
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        return relay(PairResult.pair(left, right))

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsStoredCallArgMultiOwnedEnumPayload(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
enum PairResult:
    case pair(ptr, ptr)

func relay(result: PairResult) -> PairResult:
    return result

func make_result() -> PairResult
uses alloc, mem:
    let cb: fn(PairResult) -> PairResult = relay
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        return cb(PairResult.pair(left, right))

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsFunctionParamCallArgMultiOwnedEnumPayload(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
enum PairResult:
    case pair(ptr, ptr)

func relay(result: PairResult) -> PairResult:
    return result

func apply(cb: fn(PairResult) -> PairResult) -> PairResult
uses alloc, mem:
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        return cb(PairResult.pair(left, right))

func make_result() -> PairResult
uses alloc, mem:
    return apply(relay)

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsMatchResultMultiOwnedEnumPayloadLocalReturn(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
enum PairResult:
    case empty
    case pair(ptr, ptr)

func make_result(flag: Int) -> PairResult
uses alloc, mem:
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        let result: PairResult = match flag:
        case 0:
            PairResult.pair(left, right)
        case _:
            PairResult.empty
        return result

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesRejectsCatchResultMultiOwnedEnumPayloadLocalReturn(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
enum Err:
    case bad

enum PairResult:
    case empty
    case pair(ptr, ptr)

func fail() -> PairResult throws Err:
    throw Err.bad

func make_result() -> PairResult
uses alloc, mem:
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        let result: PairResult = catch fail():
        case Err.bad:
            PairResult.pair(left, right)
        return result

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "multiple owned enum payload slots") {
		t.Fatalf("LowerWithOptions error = %v, want multiple owned enum payload slots diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesDropsCatchResultEnumOwnedPayloadReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

enum PtrResult:
    case owned(ptr)

func fail() -> PtrResult throws PtrError
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        throw PtrError.owned(p)

func make_result() -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch fail():
        case PtrError.owned(p):
            PtrResult.owned(core.alloc_bytes(32))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result()
    return 0
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("thrown enum payload must not drop in throwing callee: %#v", failFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 1 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("catch result local return must only drop the original thrown payload in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned catch-result enum payload return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsNonOwnedErrorCatchResultEnumOwnedPayloadReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Err:
    case bad

enum PtrResult:
    case owned(ptr)

func maybe(flag: Bool) -> PtrResult throws Err
uses alloc, mem:
    if flag:
        throw Err.bad
    unsafe:
        return PtrResult.owned(core.alloc_bytes(16))

func make_result(flag: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case _:
            PtrResult.owned(core.alloc_bytes(32))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned success transfer must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned non-owned-error catch result must transfer without callee drop: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned non-owned-error catch-result enum payload must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsGuardedNonOwnedErrorCatchResultEnumOwnedPayloadReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Err:
    case bad

enum PtrResult:
    case owned(ptr)

func maybe(flag: Bool) -> PtrResult throws Err
uses alloc, mem:
    if flag:
        throw Err.bad
    unsafe:
        return PtrResult.owned(core.alloc_bytes(16))

func make_result(flag: Bool, prefer: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case Err.bad if prefer == true:
            PtrResult.owned(core.alloc_bytes(32))
        case _:
            PtrResult.owned(core.alloc_bytes(48))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(true, false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned success transfer must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("make_result alloc_bytes count = %d, want 2: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("guarded non-owned-error catch result must transfer without callee drop: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded non-owned-error catch-result enum payload must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsMixedNonOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Err:
    case bad

enum PtrResult:
    case empty
    case owned(ptr)

func maybe(flag: Bool) -> PtrResult throws Err
uses alloc, mem:
    if flag:
        throw Err.bad
    unsafe:
        return PtrResult.owned(core.alloc_bytes(16))

func make_result(flag: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case _:
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned success transfer must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("mixed non-owned-error catch result must transfer without callee drop: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned mixed non-owned-error catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocal(t, mainFn, 0)
}

func TestLowerOwnedAllocBytesDropsNonZeroFallbackMixedNonOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Err:
    case bad

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws Err
uses alloc, mem:
    if flag:
        throw Err.bad
    unsafe:
        return PtrResult.owned(core.alloc_bytes(16))

func make_result(flag: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case _:
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned success transfer must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("nonzero-fallback mixed non-owned-error catch result must transfer without callee drop: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned nonzero-fallback mixed non-owned-error catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsConditionalOwnedCallRecoveryNonOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Err:
    case bad

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws Err
uses alloc, mem:
    if flag:
        throw Err.bad
    unsafe:
        return PtrResult.owned(core.alloc_bytes(16))

func replacement(prefer: Bool) -> PtrResult
uses alloc, mem:
    var result: PtrResult = PtrResult.empty
    unsafe:
        if prefer == true:
            result = PtrResult.owned(core.alloc_bytes(32))
        return result

func make_result(flag: Bool, prefer: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case Err.bad if prefer == true:
            replacement(prefer)
        case _:
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(true, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned success transfer must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	replacementFn := findIRFunc(t, prog, "replacement")
	if countInstrKind(replacementFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("replacement alloc_bytes count = %d, want 1: %#v", countInstrKind(replacementFn, ir.IRAllocBytes), replacementFn.Instrs)
	}
	if countInstrKind(replacementFn, ir.IRDropOwned) != 0 || countInstrKind(replacementFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("conditional replacement result must transfer to caller, not drop in callee: %#v", replacementFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countCallsNamed(makeFn.Instrs, "replacement") != 1 {
		t.Fatalf("make_result must call replacement once: %#v", makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("conditional owned-call non-owned-error catch result must transfer without callee drop: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned conditional owned-call non-owned-error catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsConditionalBranchAssignedNonOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Err:
    case bad

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws Err
uses alloc, mem:
    if flag:
        throw Err.bad
    unsafe:
        return PtrResult.owned(core.alloc_bytes(16))

func make_result(flag: Bool, catch_flag: Bool) -> PtrResult
uses alloc, mem:
    var result: PtrResult = PtrResult.empty
    unsafe:
        if flag == true:
            result = catch maybe(catch_flag):
            case _:
                PtrResult.owned(core.alloc_bytes(32))
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(true, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("maybe alloc_bytes count = %d, want 1: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned success transfer must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_result recovery alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("branch-assigned non-owned-error catch result must transfer without callee drop: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned branch-assigned non-owned-error catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsConditionalBranchAssignedMixedOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return PtrResult.owned(core.alloc_bytes(32))

func make_result(flag: Bool, catch_flag: Bool) -> PtrResult
uses alloc, mem:
    var result: PtrResult = PtrResult.empty
    unsafe:
        if flag == true:
            result = catch maybe(catch_flag):
            case PtrError.owned(p):
                PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(true, false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned error/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRDropOwned) != 1 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf(
			"branch-assigned mixed owned-error catch result drop/release = %d/%d, want 1/1 for original caught payload recovery only: %#v",
			countInstrKind(makeFn, ir.IRDropOwned),
			countInstrKind(makeFn, ir.IRReleaseAllocation),
			makeFn.Instrs,
		)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned branch-assigned mixed owned-error catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsMatchAssignedMixedOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Choice:
    case left
    case right

enum PtrError:
    case owned(ptr)

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return PtrResult.owned(core.alloc_bytes(32))

func make_result(choice: Choice, catch_flag: Bool) -> PtrResult
uses alloc, mem:
    var result: PtrResult = PtrResult.empty
    unsafe:
        match choice:
        case Choice.left:
            result = catch maybe(catch_flag):
            case PtrError.owned(p):
                PtrResult.empty
        case _:
            result = PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(Choice.left, false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned error/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRDropOwned) != 1 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf(
			"match-assigned mixed owned-error catch result drop/release = %d/%d, want 1/1 for original caught payload recovery only: %#v",
			countInstrKind(makeFn, ir.IRDropOwned),
			countInstrKind(makeFn, ir.IRReleaseAllocation),
			makeFn.Instrs,
		)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned match-assigned mixed owned-error catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsMixedOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return PtrResult.owned(core.alloc_bytes(32))

func make_result(flag: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case PtrError.owned(p):
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned error/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRDropOwned) != 1 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("owned-error mixed catch result must drop original caught payload exactly once: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned mixed owned-error catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsGuardedMixedOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return PtrResult.owned(core.alloc_bytes(32))

func make_result(flag: Bool, prefer: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case PtrError.owned(p) if prefer == true:
            PtrResult.empty
        case PtrError.owned(p):
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(false, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned error/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRDropOwned) != 2 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("guarded owned-error mixed catch result must drop original caught payload on guarded and fallback paths: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded mixed owned-error catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsGuardedOwnedRecoveryMixedOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return PtrResult.owned(core.alloc_bytes(32))

func make_result(flag: Bool, prefer: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case PtrError.owned(p) if prefer == true:
            PtrResult.owned(core.alloc_bytes(64))
        case PtrError.owned(p):
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(false, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned error/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("guarded recovery alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 2 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("guarded owned-result recovery must drop original caught payload on guarded and fallback paths only: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded owned-result mixed catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesRelaysGuardedOwnedErrorPayloadIntoMixedCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return PtrResult.owned(core.alloc_bytes(32))

func make_result(flag: Bool, prefer: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case PtrError.owned(p) if prefer == true:
            PtrResult.owned(p)
        case PtrError.owned(p):
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(false, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned error/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRDropOwned) != 1 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf(
			"guarded payload relay mixed catch-result drop/release = %d/%d, want 1/1; original caught payload must be dropped only on fallback path: %#v",
			countInstrKind(makeFn, ir.IRDropOwned),
			countInstrKind(makeFn, ir.IRReleaseAllocation),
			makeFn.Instrs,
		)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded payload relay mixed catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsGuardedConditionalOwnedCallRecoveryMixedOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return PtrResult.owned(core.alloc_bytes(32))

func replacement(prefer: Bool) -> PtrResult
uses alloc, mem:
    var result: PtrResult = PtrResult.empty
    unsafe:
        if prefer == true:
            result = PtrResult.owned(core.alloc_bytes(64))
        return result

func make_result(flag: Bool, prefer: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case PtrError.owned(p) if prefer == true:
            replacement(prefer)
        case PtrError.owned(p):
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(false, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned error/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	replacementFn := findIRFunc(t, prog, "replacement")
	if countInstrKind(replacementFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("replacement alloc_bytes count = %d, want 1: %#v", countInstrKind(replacementFn, ir.IRAllocBytes), replacementFn.Instrs)
	}
	if countInstrKind(replacementFn, ir.IRDropOwned) != 0 || countInstrKind(replacementFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("conditional replacement result must transfer to caller, not drop in callee: %#v", replacementFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countCallsNamed(makeFn.Instrs, "replacement") != 1 {
		t.Fatalf("make_result must call replacement once: %#v", makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 2 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf(
			"guarded conditional owned-call recovery drop/release = %d/%d, want 2/2; original caught payload must be dropped on guarded and fallback paths: %#v",
			countInstrKind(makeFn, ir.IRDropOwned),
			countInstrKind(makeFn, ir.IRReleaseAllocation),
			makeFn.Instrs,
		)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded conditional owned-call mixed catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsGuardedOwnedCallRecoveryMixedOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return PtrResult.owned(core.alloc_bytes(32))

func replacement() -> PtrResult
uses alloc, mem:
    unsafe:
        return PtrResult.owned(core.alloc_bytes(64))

func make_result(flag: Bool, prefer: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case PtrError.owned(p) if prefer == true:
            replacement()
        case PtrError.owned(p):
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(false, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned error/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	replacementFn := findIRFunc(t, prog, "replacement")
	if countInstrKind(replacementFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("replacement alloc_bytes count = %d, want 1: %#v", countInstrKind(replacementFn, ir.IRAllocBytes), replacementFn.Instrs)
	}
	if countInstrKind(replacementFn, ir.IRDropOwned) != 0 || countInstrKind(replacementFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("replacement owned result must transfer to caller, not drop in callee: %#v", replacementFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countCallsNamed(makeFn.Instrs, "replacement") != 1 {
		t.Fatalf("make_result must call replacement once: %#v", makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 2 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("guarded owned-call recovery must drop original caught payload on guarded and fallback paths only: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded owned-call mixed catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsGuardedOwnedCallRecoveryMultiCaseMixedOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)
    case timeout

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(mode: Int) -> PtrResult throws PtrError
uses alloc, mem:
    if mode == 1:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    if mode == 2:
        throw PtrError.timeout
    unsafe:
        return PtrResult.owned(core.alloc_bytes(32))

func replacement() -> PtrResult
uses alloc, mem:
    unsafe:
        return PtrResult.owned(core.alloc_bytes(64))

func make_result(mode: Int, prefer: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(mode):
        case PtrError.owned(p) if prefer == true:
            replacement()
        case PtrError.owned(p):
            PtrResult.empty
        case PtrError.timeout:
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(0, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned error/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	replacementFn := findIRFunc(t, prog, "replacement")
	if countInstrKind(replacementFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("replacement alloc_bytes count = %d, want 1: %#v", countInstrKind(replacementFn, ir.IRAllocBytes), replacementFn.Instrs)
	}
	if countInstrKind(replacementFn, ir.IRDropOwned) != 0 || countInstrKind(replacementFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("replacement owned result must transfer to caller, not drop in callee: %#v", replacementFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countCallsNamed(makeFn.Instrs, "replacement") != 1 {
		t.Fatalf("make_result must call replacement once: %#v", makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 2 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf(
			"multi-case guarded owned-call recovery drop/release = %d/%d, want 2/2; original caught owned payload must be dropped on guarded and fallback paths only: %#v",
			countInstrKind(makeFn, ir.IRDropOwned),
			countInstrKind(makeFn, ir.IRReleaseAllocation),
			makeFn.Instrs,
		)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned multi-case guarded owned-call mixed catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsMultiCaseMixedOwnedErrorCatchResultEnumOwnedPayloadReturnInCallerWhenPresent(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)
    case timeout

enum PtrResult:
    case owned(ptr)
    case empty

func maybe(flag: Bool) -> PtrResult throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return PtrResult.owned(core.alloc_bytes(32))

func make_result(flag: Bool) -> PtrResult
uses alloc, mem:
    unsafe:
        let result: PtrResult = catch maybe(flag):
        case PtrError.owned(p):
            PtrResult.empty
        case PtrError.timeout:
            PtrResult.empty
        return result

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrResult = make_result(false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned error/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	makeFn := findIRFunc(t, prog, "make_result")
	if countInstrKind(makeFn, ir.IRDropOwned) != 1 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("multi-case owned-error mixed catch result must drop only the original owned caught payload: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned multi-case mixed owned-error catch-result enum payload must be dropped in caller when present: %#v", mainFn.Instrs)
	}
	assertOwnedReleaseGuardedByLocalEq(t, mainFn, 0, 0)
}

func TestLowerOwnedAllocBytesDropsThrownEnumOwnedPayloadInCatch(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func fail() -> Int throws PtrError
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        throw PtrError.owned(p)

func main() -> Int
uses alloc, mem:
    return catch fail():
    case PtrError.owned(p):
        7
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("enum owned payload throw must not drop in callee: %#v", failFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "fail") != 1 {
		t.Fatalf("main must call fail once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 2 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("caught enum owned payload must be dropped on case and unmatched catch paths: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsLocalEnumOwnedPayloadThrowInCatch(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func fail() -> Int throws PtrError
uses alloc, mem:
    unsafe:
        let err: PtrError = PtrError.owned(core.alloc_bytes(16))
        throw err

func main() -> Int
uses alloc, mem:
    return catch fail():
    case PtrError.owned(p):
        7
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("local enum payload throw must not drop in callee: %#v", failFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) != 2 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("caught local enum payload must be dropped on case and unmatched catch paths: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsAssignedLocalEnumOwnedPayloadThrowInCatch(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case empty
    case owned(ptr)

func fail() -> Int throws PtrError
uses alloc, mem:
    var err: PtrError = PtrError.empty
    unsafe:
        err = PtrError.owned(core.alloc_bytes(16))
        throw err

func main() -> Int
uses alloc, mem:
    return catch fail():
    case PtrError.owned(p):
        7
    case PtrError.empty:
        0
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("assigned local enum payload throw must not drop in callee: %#v", failFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countInstrKind(mainFn, ir.IRDropOwned) < 2 || countInstrKind(mainFn, ir.IRReleaseAllocation) < 2 {
		t.Fatalf("caught assigned local enum payload must be dropped on catch paths: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtEnumOwnedPayloadToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func fail() -> ptr throws PtrError
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        throw PtrError.owned(p)

func relay() -> ptr
uses alloc, mem:
    return catch fail():
    case PtrError.owned(p):
        p

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay()
    return 0
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned thrown payload must not be dropped in throwing callee: %#v", failFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "fail") != 1 {
		t.Fatalf("relay must call fail once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("relayed caught payload must not be dropped in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned catch-relayed payload must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysMultiSourceCaughtEnumOwnedPayloadToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func fail(which: Bool) -> ptr throws PtrError
uses alloc, mem:
    if which:
        unsafe:
            let left: ptr = core.alloc_bytes(16)
            throw PtrError.owned(left)
    unsafe:
        let right: ptr = core.alloc_bytes(32)
        throw PtrError.owned(right)

func relay(which: Bool) -> ptr
uses alloc, mem:
    return catch fail(which):
    case PtrError.owned(p):
        p

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true)
    return 0
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("fail alloc_bytes count = %d, want 2: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned thrown payloads must not be dropped in throwing callee: %#v", failFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "fail") != 1 {
		t.Fatalf("relay must call fail once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("multi-source catch relay must not drop owned result in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned multi-source catch-relayed payload must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysPreallocatedLocalCatchRecoveryToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func fail() -> ptr throws PtrError
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        throw PtrError.owned(p)

func relay() -> ptr
uses alloc, mem:
    unsafe:
        let fallback: ptr = core.alloc_bytes(64)
        return catch fail():
        case PtrError.owned(p):
            fallback

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay()
    return 0
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned thrown payload must not be dropped in throwing callee: %#v", failFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countInstrKind(relayFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("relay fallback alloc_bytes count = %d, want 1: %#v", countInstrKind(relayFn, ir.IRAllocBytes), relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 1 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("relay must drop original caught payload only, not transferred fallback: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned preallocated catch recovery must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysGuardedPreallocatedLocalCatchRecoveryToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func fail() -> ptr throws PtrError
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        throw PtrError.owned(p)

func relay(prefer: Bool) -> ptr
uses alloc, mem:
    unsafe:
        let fallback: ptr = core.alloc_bytes(64)
        return catch fail():
        case PtrError.owned(p) if prefer == true:
            fallback
        case PtrError.owned(p):
            fallback

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true)
    return 0
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned thrown payload must not be dropped in throwing callee: %#v", failFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countInstrKind(relayFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("relay fallback alloc_bytes count = %d, want 1: %#v", countInstrKind(relayFn, ir.IRAllocBytes), relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 2 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("guarded relay must drop original caught payload on guarded/fallback paths only, not transferred fallback: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded preallocated catch recovery must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysSuccessOrErrorPreallocatedLocalCatchRecoveryToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func maybe(flag: Bool) -> ptr throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool) -> ptr
uses alloc, mem:
    unsafe:
        let fallback: ptr = core.alloc_bytes(64)
        return catch maybe(flag):
        case PtrError.owned(p):
            fallback

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countInstrKind(relayFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("relay fallback alloc_bytes count = %d, want 1: %#v", countInstrKind(relayFn, ir.IRAllocBytes), relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 2 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("relay must clean unused fallback on success and original caught payload on error only: %#v", relayFn.Instrs)
	}
	assertReleaseLayoutInFirstCatchSuccessBranch(t, relayFn, "layout:core.alloc_bytes:fallback")
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned success-or-error preallocated catch recovery must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysDifferentPreallocatedLocalCatchRecoveryToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case left(ptr)
    case right(ptr)

func fail(which: Bool) -> ptr throws PtrError
uses alloc, mem:
    unsafe:
        let payload: ptr = core.alloc_bytes(16)
        if which:
            throw PtrError.left(payload)
        throw PtrError.right(payload)

func relay(which: Bool) -> ptr
uses alloc, mem:
    unsafe:
        let leftFallback: ptr = core.alloc_bytes(64)
        let rightFallback: ptr = core.alloc_bytes(96)
        return catch fail(which):
        case PtrError.left(p):
            leftFallback
        case PtrError.right(p):
            rightFallback

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true)
    return 0
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned thrown payloads must not be dropped in throwing callee: %#v", failFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countInstrKind(relayFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("relay fallback alloc_bytes count = %d, want 2: %#v", countInstrKind(relayFn, ir.IRAllocBytes), relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 4 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 4 {
		t.Fatalf("relay must clean original caught payload and the unselected fallback on each error path: %#v", relayFn.Instrs)
	}
	assertReleaseLayoutInCatchCaseBranch(t, relayFn, 0, "layout:core.alloc_bytes:rightFallback")
	assertReleaseLayoutInCatchCaseBranch(t, relayFn, 1, "layout:core.alloc_bytes:leftFallback")
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned different-local preallocated catch recovery must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysGuardedDefaultPreallocatedLocalCatchRecoveryToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)
    case timeout

func fail(mode: Int) -> ptr throws PtrError
uses alloc, mem:
    if mode == 1:
        unsafe:
            let payload: ptr = core.alloc_bytes(16)
            throw PtrError.owned(payload)
    throw PtrError.timeout

func relay(mode: Int, prefer: Bool) -> ptr
uses alloc, mem:
    unsafe:
        let guardedFallback: ptr = core.alloc_bytes(64)
        let defaultFallback: ptr = core.alloc_bytes(96)
        return catch fail(mode):
        case PtrError.owned(p) if prefer == true:
            guardedFallback
        case _:
            defaultFallback

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(1, false)
    return 0
`)
	failFn := findIRFunc(t, prog, "fail")
	if countInstrKind(failFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("fail alloc_bytes count = %d, want 1: %#v", countInstrKind(failFn, ir.IRAllocBytes), failFn.Instrs)
	}
	if countInstrKind(failFn, ir.IRDropOwned) != 0 || countInstrKind(failFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned thrown payload must not be dropped in throwing callee: %#v", failFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countInstrKind(relayFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("relay fallback alloc_bytes count = %d, want 2: %#v", countInstrKind(relayFn, ir.IRAllocBytes), relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 4 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 4 {
		t.Fatalf("relay must clean original caught payload only on owned default paths and unused fallbacks per path: %#v", relayFn.Instrs)
	}
	assertReleaseLayoutInCatchCaseBranch(t, relayFn, 0, "layout:core.alloc_bytes:defaultFallback")
	assertReleaseLayoutInAnyCatchCaseBranch(t, relayFn, "layout:core.alloc_bytes:guardedFallback")
	assertReleaseLayoutGuardedByTagEqInAnyBranch(t, relayFn, "layout:throw:fail:payload", 0)
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded default preallocated catch recovery must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtEnumOwnedPayloadOrSuccessToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func maybe(flag: Bool) -> ptr throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case PtrError.owned(p):
        p

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("catch relay must not drop owned throw/success result in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned catch-relayed throw/success result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtMultiCaseEnumOwnedPayloadToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)
    case fallback

func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case PtrError.owned(p):
        p
    case PtrError.fallback:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep exhaustive fallback arm lowering: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("multi-case catch relay must not drop owned result in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned multi-case catch-relayed result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtDefaultEnumOwnedPayloadToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)
    case fallback
    case timeout

func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case PtrError.owned(p):
        p
    case _:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep default fallback arm lowering: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("default catch relay must not drop owned result in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned default catch-relayed result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsCaughtOwnedPayloadInSingleOwnedDefaultFallbackToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)
    case timeout

func make_timeout() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(48)

func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case PtrError.timeout:
        make_timeout()
    case _:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_timeout") != 1 || countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep explicit non-owned and default fallback arms: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 1 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("default fallback must drop the original owned enum payload exactly once: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned single-owned-default fallback result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysGuardedEnumOwnedPayloadOrDefaultFallbackToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)
    case timeout

func make_timeout() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(48)

func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool, prefer: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case PtrError.timeout:
        make_timeout()
    case PtrError.owned(p) if prefer == true:
        p
    case _:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true, false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_timeout") != 1 || countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep explicit non-owned and default fallback arms: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 1 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("default fallback must drop the original guarded owned enum payload exactly once: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded-default fallback result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtGuardedEnumOwnedPayloadToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func maybe(flag: Bool) -> ptr throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool, prefer: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case PtrError.owned(p) if prefer == true:
        p
    case PtrError.owned(p):
        p

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(false, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("guarded catch relay must not drop owned result in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded catch-relayed result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtGuardedEnumOwnedPayloadOrFallbackToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool, prefer: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case PtrError.owned(p) if prefer == true:
        p
    case PtrError.owned(p):
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true, false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep same-case fallback arm lowering: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 1 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("same-case fallback arm must drop the original owned error payload exactly once: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded fallback catch result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtGuardedEnumOwnedPayloadFallbackOrRelayToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)

func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool, prefer: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case PtrError.owned(p) if prefer == true:
        make_fallback()
    case PtrError.owned(p):
        p

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep guarded same-case fallback arm lowering: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 1 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("guarded same-case fallback arm must drop the original owned error payload exactly once: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded fallback-or-relay catch result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtGuardedEnumNonOwnedFallbackToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum PtrError:
    case owned(ptr)
    case fallback

func make_preferred() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(96)

func maybe(flag: Bool) -> ptr throws PtrError
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw PtrError.owned(p)
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool, prefer: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case PtrError.owned(p):
        p
    case PtrError.fallback if prefer == true:
        make_preferred()
    case PtrError.fallback:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(false, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_preferred") != 1 || countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep both guarded and unguarded fallback arms: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("guarded non-owned enum fallback relay must not drop owned result in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded non-owned enum fallback catch result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtOptionalOwnedPayloadToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws ptr?
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw p
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case some(p):
        p
    case none:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned optional throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep optional none fallback arm lowering: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("optional catch relay must not drop owned result in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned optional catch-relayed result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtGuardedOptionalOwnedPayloadToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws ptr?
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw p
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool, prefer: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case some(p) if prefer == true:
        p
    case some(p):
        p
    case none:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(false, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned guarded optional throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep guarded optional none fallback arm lowering: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("guarded optional catch relay must not drop owned result in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded optional catch-relayed result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtGuardedOptionalOwnedPayloadOrFallbackToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws ptr?
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw p
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool, prefer: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case some(p) if prefer == true:
        p
    case some(p):
        make_fallback()
    case none:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true, false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned guarded optional throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_fallback") != 2 {
		t.Fatalf("relay must keep optional some and none fallback arm lowering: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 1 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("same-some fallback arm must drop the original owned optional payload exactly once: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded optional fallback catch result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtGuardedOptionalOwnedPayloadFallbackOrRelayToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws ptr?
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw p
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool, prefer: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case some(p) if prefer == true:
        make_fallback()
    case some(p):
        p
    case none:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned guarded optional throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_fallback") != 2 {
		t.Fatalf("relay must keep guarded optional some and none fallback arm lowering: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 1 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("guarded same-some fallback arm must drop the original owned optional payload exactly once: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded optional fallback-or-relay catch result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysGuardedOptionalOwnedPayloadOrDefaultFallbackToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws ptr?
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw p
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool, prefer: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case some(p) if prefer == true:
        make_fallback()
    case _:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true, false)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned guarded optional throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_fallback") != 2 {
		t.Fatalf("relay must keep guarded optional and default fallback arm lowering: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 2 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("guarded optional default fallback arms must each drop the original owned optional payload: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded optional default fallback catch result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRelaysCaughtGuardedOptionalNoneFallbackToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make_preferred() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(96)

func maybe(flag: Bool) -> ptr throws ptr?
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw p
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool, prefer: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case some(p):
        p
    case none if prefer == true:
        make_preferred()
    case none:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(false, true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned optional throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_preferred") != 1 || countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep both guarded and unguarded none fallback arms: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("guarded optional none fallback relay must not drop owned result in relay callee: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned guarded none fallback catch result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsCaughtOptionalPayloadInDefaultFallbackToCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make_none() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(48)

func make_fallback() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(64)

func maybe(flag: Bool) -> ptr throws ptr?
uses alloc, mem:
    if flag:
        unsafe:
            let p: ptr = core.alloc_bytes(16)
            throw p
    unsafe:
        return core.alloc_bytes(32)

func relay(flag: Bool) -> ptr
uses alloc, mem:
    return catch maybe(flag):
    case none:
        make_none()
    case _:
        make_fallback()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay(true)
    return 0
`)
	maybeFn := findIRFunc(t, prog, "maybe")
	if countInstrKind(maybeFn, ir.IRAllocBytes) != 2 {
		t.Fatalf("maybe alloc_bytes count = %d, want 2: %#v", countInstrKind(maybeFn, ir.IRAllocBytes), maybeFn.Instrs)
	}
	if countInstrKind(maybeFn, ir.IRDropOwned) != 0 || countInstrKind(maybeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned optional throw/success transfers must not be dropped in throwing callee: %#v", maybeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "maybe") != 1 {
		t.Fatalf("relay must call maybe once: %#v", relayFn.Instrs)
	}
	if countCallsNamed(relayFn.Instrs, "make_none") != 1 || countCallsNamed(relayFn.Instrs, "make_fallback") != 1 {
		t.Fatalf("relay must keep explicit none and default fallback arms: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 1 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("default fallback must drop the original owned optional payload exactly once: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned optional default fallback result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRejectsMultiOwnedInlineAggregateThrow(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
struct PairBox:
    left: ptr
    right: ptr

func fail() -> Int throws PairBox
uses alloc, mem:
    unsafe:
        let left: ptr = core.alloc_bytes(16)
        let right: ptr = core.alloc_bytes(32)
        throw PairBox(left: left, right: right)

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "inline owned throw") {
		t.Fatalf("LowerWithOptions error = %v, want inline owned throw diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesClearsMovedFieldMarkerAfterAggregateOverwrite(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

func ret() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(tag: 1, raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
        let q: ptr = box.raw
        box = PtrBox(tag: 2, raw: 0)
        return box

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "ret")
	if countInstrKind(fn, ir.IRDropOwned) != 1 || countInstrKind(fn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("moved field owner must be cleaned up after aggregate overwrite: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterLocalStoreReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func ret() -> ptr
uses alloc, mem:
    var q: ptr = 0
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        q = p
    return q

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "ret")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("ret alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("locally stored returned owned pointer must not be dropped in callee: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsOverwrittenOwnedLocalBeforeBorrowedReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func relay(borrowed: ptr) -> ptr
uses alloc, mem:
    var q: ptr = make()
    q = borrowed
    return q

func main() -> Int
uses alloc, mem:
    unsafe:
        let borrowed: ptr = 0
        let r: ptr = relay(borrowed)
    return 0
`)
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "make") != 1 {
		t.Fatalf("relay must call make once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 1 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("overwritten owned local must be dropped before borrowed overwrite/return: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 0 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("borrowed relay result must not be summarized as caller-owned: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsAssignedReturnedOwnedCallResultInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func relay() -> ptr
uses alloc, mem:
    var q: ptr = 0
    q = make()
    return q

func main() -> Int
uses alloc, mem:
    unsafe:
        let r: ptr = relay()
    return 0
`)
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "make") != 1 {
		t.Fatalf("relay must call make once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("relay must not drop assigned owned pointer it returns: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned assigned relay result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterGlobalStore(t *testing.T) {
	prog := lowerOwnedAllocDropFileProgram(t, `
var leaked: ptr = 0

func store() -> Int
uses alloc, capability, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        leaked = p
    return 0

func main() -> Int
uses alloc, capability, mem:
    return store()
`)
	fn := findIRFunc(t, prog, "store")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("store alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRStoreGlobal) == 0 {
		t.Fatalf("store must write the transferred pointer to global storage: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("global-stored owned pointer must not be dropped in source scope: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterReturnedOwnedCallStoredInGlobal(t *testing.T) {
	prog := lowerOwnedAllocDropFileProgram(t, `
var leaked: ptr = 0

func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func store() -> Int
uses alloc, capability, mem:
    leaked = make()
    return 0

func main() -> Int
uses alloc, capability, mem:
    return store()
`)
	fn := findIRFunc(t, prog, "store")
	if countCallsNamed(fn.Instrs, "make") != 1 {
		t.Fatalf("store must call make once: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRStoreGlobal) == 0 {
		t.Fatalf("store must write the returned owned pointer to global storage: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("global-stored returned owned pointer must not be dropped in source scope: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterGlobalFieldStore(t *testing.T) {
	prog := lowerOwnedAllocDropFileProgram(t, `
struct PtrBox:
    raw: ptr

var box: PtrBox

func store() -> Int
uses alloc, capability, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
    return 0

func main() -> Int
uses alloc, capability, mem:
    return store()
`)
	fn := findIRFunc(t, prog, "store")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("store alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRStoreGlobal) == 0 {
		t.Fatalf("store must write the transferred pointer to global field storage: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("global-field-stored owned pointer must not be dropped in source scope: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterReturnedOwnedCallStoredInGlobalField(t *testing.T) {
	prog := lowerOwnedAllocDropFileProgram(t, `
struct PtrBox:
    raw: ptr

var box: PtrBox

func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func store() -> Int
uses alloc, capability, mem:
    box.raw = make()
    return 0

func main() -> Int
uses alloc, capability, mem:
    return store()
`)
	fn := findIRFunc(t, prog, "store")
	if countCallsNamed(fn.Instrs, "make") != 1 {
		t.Fatalf("store must call make once: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRStoreGlobal) == 0 {
		t.Fatalf("store must write the returned owned pointer to global field storage: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("global-field-stored returned owned pointer must not be dropped in source scope: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterNestedInlineAggregateGlobalStore(t *testing.T) {
	prog := lowerOwnedAllocDropFileProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

var saved: Holder

func store() -> Int
uses alloc, capability, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        saved = Holder(head: 1, box: PtrBox(tag: 2, raw: p))
    return 0

func main() -> Int
uses alloc, capability, mem:
    return store()
`)
	fn := findIRFunc(t, prog, "store")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("store alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRStoreGlobal) == 0 {
		t.Fatalf("store must write the nested aggregate to global storage: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("global-stored nested aggregate owned field must not be dropped in source scope: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterNestedInlineDirectAllocAggregateGlobalStore(t *testing.T) {
	prog := lowerOwnedAllocDropFileProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

var saved: Holder

func store() -> Int
uses alloc, capability, mem:
    unsafe:
        saved = Holder(head: 1, box: PtrBox(tag: 2, raw: core.alloc_bytes(16)))
    return 0

func main() -> Int
uses alloc, capability, mem:
    return store()
`)
	fn := findIRFunc(t, prog, "store")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("store alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRStoreGlobal) == 0 {
		t.Fatalf("store must write the direct nested aggregate to global storage: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("global-stored nested direct aggregate owned field must not be dropped in source scope: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterNestedInlineOwnedReturnCallAggregateGlobalStore(t *testing.T) {
	prog := lowerOwnedAllocDropFileProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

var saved: Holder

func make_raw() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(16)

func store() -> Int
uses alloc, capability, mem:
    saved = Holder(head: 1, box: PtrBox(tag: 2, raw: make_raw()))
    return 0

func main() -> Int
uses alloc, capability, mem:
    return store()
`)
	makeRawFn := findIRFunc(t, prog, "make_raw")
	if countInstrKind(makeRawFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_raw alloc_bytes count = %d, want 1: %#v", countInstrKind(makeRawFn, ir.IRAllocBytes), makeRawFn.Instrs)
	}
	if countInstrKind(makeRawFn, ir.IRDropOwned) != 0 || countInstrKind(makeRawFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("raw factory must not drop returned pointer in callee: %#v", makeRawFn.Instrs)
	}
	fn := findIRFunc(t, prog, "store")
	if countCallsNamed(fn.Instrs, "make_raw") != 1 {
		t.Fatalf("store must call make_raw once: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRStoreGlobal) == 0 {
		t.Fatalf("store must write the owned-return nested aggregate to global storage: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("global-stored nested owned-return aggregate field must not be dropped in source scope: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRejectsNestedInlineLocalLiteralSourceReturn(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

func ret() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        let holder: Holder = Holder(head: 1, box: PtrBox(tag: 2, raw: p))
        return p

func main() -> Int:
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "use after move") {
		t.Fatalf("LowerWithOptions error = %v, want nested local literal source use-after-move diagnostic", err)
	}
}

func TestLowerOwnedAllocBytesDropsNestedInlineDirectAllocLocalLiteralInCallee(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

func store() -> Int
uses alloc, mem:
    unsafe:
        let holder: Holder = Holder(head: 1, box: PtrBox(tag: 2, raw: core.alloc_bytes(16)))
    return 0

func main() -> Int
uses alloc, mem:
    return store()
`)
	fn := findIRFunc(t, prog, "store")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("store alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 1 || countInstrKind(fn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("nested direct local literal owned field must be dropped in callee: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsAssignedNestedInlineOwnedReturnCallLocalLiteralInCallee(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

func make_raw() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(16)

func store() -> Int
uses alloc, mem:
    var holder: Holder = Holder(head: 0, box: PtrBox(tag: 0, raw: 0))
    holder = Holder(head: 1, box: PtrBox(tag: 2, raw: make_raw()))
    return 0

func main() -> Int
uses alloc, mem:
    return store()
`)
	makeRawFn := findIRFunc(t, prog, "make_raw")
	if countInstrKind(makeRawFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_raw alloc_bytes count = %d, want 1: %#v", countInstrKind(makeRawFn, ir.IRAllocBytes), makeRawFn.Instrs)
	}
	if countInstrKind(makeRawFn, ir.IRDropOwned) != 0 || countInstrKind(makeRawFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("raw factory must not drop returned pointer in callee: %#v", makeRawFn.Instrs)
	}
	fn := findIRFunc(t, prog, "store")
	if countCallsNamed(fn.Instrs, "make_raw") != 1 {
		t.Fatalf("store must call make_raw once: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 1 || countInstrKind(fn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("assigned nested owned-return local literal field must be dropped in callee: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterLocalFieldStoreReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func ret() -> ptr
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
    return box.raw

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "ret")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("ret alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countInstrKind(fn, ir.IRStoreLocal) == 0 {
		t.Fatalf("ret must write the transferred pointer to local field storage: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("local-field-returned owned pointer must not be dropped in callee: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsReturnedOwnedLocalFieldResultInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func ret() -> ptr
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
    return box.raw

func main() -> Int
uses alloc, mem:
    unsafe:
        let p: ptr = ret()
    return 0
`)
	retFn := findIRFunc(t, prog, "ret")
	if countInstrKind(retFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("ret alloc_bytes count = %d, want 1: %#v", countInstrKind(retFn, ir.IRAllocBytes), retFn.Instrs)
	}
	if countInstrKind(retFn, ir.IRDropOwned) != 0 || countInstrKind(retFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("local-field-returned owned pointer must not be dropped in callee: %#v", retFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "ret") != 1 {
		t.Fatalf("main must call ret once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned local-field result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsReturnedOwnedCallStoredInLocalFieldInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func ret() -> ptr
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    box.raw = make()
    return box.raw

func main() -> Int
uses alloc, mem:
    unsafe:
        let p: ptr = ret()
    return 0
`)
	retFn := findIRFunc(t, prog, "ret")
	if countCallsNamed(retFn.Instrs, "make") != 1 {
		t.Fatalf("ret must call make once: %#v", retFn.Instrs)
	}
	if countInstrKind(retFn, ir.IRDropOwned) != 0 || countInstrKind(retFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("local-field returned owned call result must not be dropped in callee: %#v", retFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "ret") != 1 {
		t.Fatalf("main must call ret once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned local-field call result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsReturnedOwnedSingleSlotAggregateInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
    return box

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned single-slot aggregate must not be dropped in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned single-slot aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsInlineStructLiteralOwnedAggregateInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        let box: PtrBox = PtrBox(raw: p)
        return box

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("inline literal returned aggregate must not be dropped in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned inline literal aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsAssignedInlineStructLiteralOwnedAggregateInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box = PtrBox(raw: p)
    return box

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("assigned inline literal returned aggregate must not be dropped in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned assigned inline literal aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropForUnsafeNestedAssignedInlineStructLiteralReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box = PtrBox(raw: p)
        return box

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("unsafe nested returned aggregate must not be dropped in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned unsafe nested aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterDefaultMatchReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Choice:
    case left
    case right

struct PtrBox:
    raw: ptr

func make_box(choice: Choice) -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box = PtrBox(raw: p)
    match choice:
    case Choice.left:
        return box
    case _:
        return box

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box(Choice.left)
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("default match returned aggregate must not be dropped in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned default match aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterCompleteEnumMatchReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Choice:
    case left
    case right

struct PtrBox:
    raw: ptr

func make_box(choice: Choice) -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box = PtrBox(raw: p)
    match choice:
    case Choice.left:
        return box
    case Choice.right:
        return box

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box(Choice.left)
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("complete enum match returned aggregate must not be dropped in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned complete enum match aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterCompleteOptionalMatchReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_box(choice: Bool?) -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box = PtrBox(raw: p)
    match choice:
    case some(flag):
        return box
    case none:
        return box

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box(true)
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("complete optional match returned aggregate must not be dropped in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned complete optional match aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsInlineStructLiteralDirectAllocAggregateInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    unsafe:
        let box: PtrBox = PtrBox(raw: core.alloc_bytes(16))
        return box

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("direct inline literal returned aggregate must not be dropped in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned direct inline literal aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsAssignedInlineStructLiteralDirectAllocAggregateInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    unsafe:
        box = PtrBox(raw: core.alloc_bytes(16))
    return box

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("assigned direct inline literal returned aggregate must not be dropped in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned assigned direct inline literal aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsInlineStructLiteralOwnedReturnCallAggregateInCallee(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_raw() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func store() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = PtrBox(raw: make_raw())
    return 0

func main() -> Int
uses alloc, mem:
    return store()
`)
	makeRawFn := findIRFunc(t, prog, "make_raw")
	if countInstrKind(makeRawFn, ir.IRDropOwned) != 0 || countInstrKind(makeRawFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("owned-return field source must not be dropped in producer callee: %#v", makeRawFn.Instrs)
	}
	storeFn := findIRFunc(t, prog, "store")
	if countCallsNamed(storeFn.Instrs, "make_raw") != 1 {
		t.Fatalf("store must call make_raw once: %#v", storeFn.Instrs)
	}
	if countInstrKind(storeFn, ir.IRDropOwned) != 1 || countInstrKind(storeFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("inline literal owned-return field must be dropped in storing callee: %#v", storeFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsAssignedInlineStructLiteralOwnedReturnCallAggregateInCallee(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    raw: ptr

func make_raw() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func store() -> Int
uses alloc, mem:
    var box: PtrBox = PtrBox(raw: 0)
    unsafe:
        box = PtrBox(raw: make_raw())
    return 0

func main() -> Int
uses alloc, mem:
    return store()
`)
	makeRawFn := findIRFunc(t, prog, "make_raw")
	if countInstrKind(makeRawFn, ir.IRDropOwned) != 0 || countInstrKind(makeRawFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("assigned owned-return field source must not be dropped in producer callee: %#v", makeRawFn.Instrs)
	}
	storeFn := findIRFunc(t, prog, "store")
	if countCallsNamed(storeFn.Instrs, "make_raw") != 1 {
		t.Fatalf("store must call make_raw once: %#v", storeFn.Instrs)
	}
	if countInstrKind(storeFn, ir.IRDropOwned) != 1 || countInstrKind(storeFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("assigned inline literal owned-return field must be dropped in storing callee: %#v", storeFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsReturnedOwnedMultiSlotAggregateInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(tag: 1, raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
    return box

func main() -> Int
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make_box alloc_bytes count = %d, want 1: %#v", countInstrKind(makeBoxFn, ir.IRAllocBytes), makeBoxFn.Instrs)
	}
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned multi-slot aggregate must not be dropped in callee: %#v", makeBoxFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_box") != 1 {
		t.Fatalf("main must call make_box once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned multi-slot aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterMultiSlotAggregateLocalMoveReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(tag: 1, raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
    return box

func relay() -> PtrBox
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
        let moved: PtrBox = box
        return moved

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrBox = relay()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned multi-slot aggregate factory must not drop in callee: %#v", makeBoxFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "make_box") != 1 {
		t.Fatalf("relay must call make_box once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("moved multi-slot aggregate returned from relay must not be dropped in relay: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned moved multi-slot aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterMultiSlotAggregateLocalAssignReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

func make_box() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(tag: 1, raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
    return box

func relay() -> PtrBox
uses alloc, mem:
    unsafe:
        let box: PtrBox = make_box()
        var moved: PtrBox = PtrBox(tag: 0, raw: 0)
        moved = box
        return moved

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: PtrBox = relay()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned multi-slot aggregate factory must not drop in callee: %#v", makeBoxFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "make_box") != 1 {
		t.Fatalf("relay must call make_box once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("assigned multi-slot aggregate returned from relay must not be dropped in relay: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned assigned multi-slot aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesDropAfterMultiSlotAggregateGlobalFieldStore(t *testing.T) {
	prog := lowerOwnedAllocDropFileProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

var saved: Holder

func make_box() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(tag: 1, raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
    return box

func store() -> Int
uses alloc, capability, mem:
    unsafe:
        let box: PtrBox = make_box()
        saved.box = box
    return 0

func main() -> Int
uses alloc, capability, mem:
    return store()
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned multi-slot aggregate factory must not drop in callee: %#v", makeBoxFn.Instrs)
	}
	storeFn := findIRFunc(t, prog, "store")
	if countCallsNamed(storeFn.Instrs, "make_box") != 1 {
		t.Fatalf("store must call make_box once: %#v", storeFn.Instrs)
	}
	if countInstrKind(storeFn, ir.IRStoreGlobal) == 0 {
		t.Fatalf("store must write the transferred multi-slot aggregate to global field storage: %#v", storeFn.Instrs)
	}
	if countInstrKind(storeFn, ir.IRDropOwned) != 0 || countInstrKind(storeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("global-field-stored multi-slot aggregate must not be dropped in source scope: %#v", storeFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsReturnedOwnedNestedAggregateInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

func make_holder() -> Holder
uses alloc, mem:
    var holder: Holder = Holder(head: 1, box: PtrBox(tag: 2, raw: 0))
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        holder.box.raw = p
    return holder

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: Holder = make_holder()
    return 0
`)
	makeHolderFn := findIRFunc(t, prog, "make_holder")
	if countInstrKind(makeHolderFn, ir.IRDropOwned) != 0 || countInstrKind(makeHolderFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned nested aggregate factory must not drop in callee: %#v", makeHolderFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_holder") != 1 {
		t.Fatalf("main must call make_holder once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned nested aggregate result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsReturnedOwnedNestedAggregateAssignedFieldInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

func make_box() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(tag: 2, raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
    return box

func make_holder() -> Holder
uses alloc, mem:
    var holder: Holder = Holder(head: 1, box: PtrBox(tag: 0, raw: 0))
    unsafe:
        let box: PtrBox = make_box()
        holder.box = box
    return holder

func main() -> Int
uses alloc, mem:
    unsafe:
        let result: Holder = make_holder()
    return 0
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned nested field source factory must not drop in callee: %#v", makeBoxFn.Instrs)
	}
	makeHolderFn := findIRFunc(t, prog, "make_holder")
	if countCallsNamed(makeHolderFn.Instrs, "make_box") != 1 {
		t.Fatalf("make_holder must call make_box once: %#v", makeHolderFn.Instrs)
	}
	if countInstrKind(makeHolderFn, ir.IRDropOwned) != 0 || countInstrKind(makeHolderFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned nested aggregate assigned field must not drop in callee: %#v", makeHolderFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make_holder") != 1 {
		t.Fatalf("main must call make_holder once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned nested aggregate assigned field result must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsReturnedOwnedAggregateAssignedToNestedFieldBeforeReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
struct PtrBox:
    tag: Int
    raw: ptr

struct Holder:
    head: Int
    box: PtrBox

func make_box() -> PtrBox
uses alloc, mem:
    var box: PtrBox = PtrBox(tag: 2, raw: 0)
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        box.raw = p
    return box

func store() -> Int
uses alloc, mem:
    var holder: Holder = Holder(head: 1, box: PtrBox(tag: 0, raw: 0))
    unsafe:
        holder.box = make_box()
    return 0

func main() -> Int
uses alloc, mem:
    return store()
`)
	makeBoxFn := findIRFunc(t, prog, "make_box")
	if countInstrKind(makeBoxFn, ir.IRDropOwned) != 0 || countInstrKind(makeBoxFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned nested field source factory must not drop in callee: %#v", makeBoxFn.Instrs)
	}
	storeFn := findIRFunc(t, prog, "store")
	if countCallsNamed(storeFn.Instrs, "make_box") != 1 {
		t.Fatalf("store must call make_box once: %#v", storeFn.Instrs)
	}
	if countInstrKind(storeFn, ir.IRDropOwned) != 1 || countInstrKind(storeFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("nested field assigned owned-return aggregate must be dropped before return: %#v", storeFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSuppressesCallerDropAfterConsumeCall(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func sink(raw: consume ptr) -> Int:
    return 0

func main() -> Int
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return sink(p)
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRAllocBytes) != 1 {
		t.Fatalf("main alloc_bytes count = %d, want 1: %#v", countInstrKind(fn, ir.IRAllocBytes), fn.Instrs)
	}
	if countCallsNamed(fn.Instrs, "sink") != 1 {
		t.Fatalf("main must pass the owned pointer to sink: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 0 || countInstrKind(fn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("consume-call transferred owned pointer must not be dropped in caller: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsConsumeParamInCallee(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func sink(raw: consume ptr) -> Int:
    return 0

func main() -> Int
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return sink(p)
`)
	fn := findIRFunc(t, prog, "sink")
	if len(fn.OwnedParams) != 1 {
		t.Fatalf("sink owned params = %#v, want one consume ptr param", fn.OwnedParams)
	}
	if countInstrKind(fn, ir.IRDropOwned) != 1 || countInstrKind(fn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("consume ptr parameter must be dropped/released by callee: %#v", fn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsReturnedOwnedCallResultInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = make()
    return 0
`)
	makeFn := findIRFunc(t, prog, "make")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("returned owned pointer must not be dropped in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make") != 1 {
		t.Fatalf("main must call make once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned returned pointer must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsDirectInlineAllocReturnInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        return core.alloc_bytes(16)

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = make()
    return 0
`)
	makeFn := findIRFunc(t, prog, "make")
	if countInstrKind(makeFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("make alloc_bytes count = %d, want 1: %#v", countInstrKind(makeFn, ir.IRAllocBytes), makeFn.Instrs)
	}
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("direct inline alloc return must not be dropped in callee: %#v", makeFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "make") != 1 {
		t.Fatalf("main must call make once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned direct inline alloc return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsRelayedReturnedOwnedCallResultInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func relay() -> ptr
uses alloc, mem:
    return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let q: ptr = relay()
    return 0
`)
	makeFn := findIRFunc(t, prog, "make")
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("direct factory must not drop returned owned pointer in callee: %#v", makeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "make") != 1 {
		t.Fatalf("relay must call make once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("relay must not drop owned pointer it returns to caller: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned relayed pointer must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsLocalRelayedReturnedOwnedCallResultInCaller(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func relay() -> ptr
uses alloc, mem:
    let q: ptr = make()
    return q

func main() -> Int
uses alloc, mem:
    unsafe:
        let r: ptr = relay()
    return 0
`)
	makeFn := findIRFunc(t, prog, "make")
	if countInstrKind(makeFn, ir.IRDropOwned) != 0 || countInstrKind(makeFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("direct factory must not drop returned owned pointer in callee: %#v", makeFn.Instrs)
	}
	relayFn := findIRFunc(t, prog, "relay")
	if countCallsNamed(relayFn.Instrs, "make") != 1 {
		t.Fatalf("relay must call make once: %#v", relayFn.Instrs)
	}
	if countInstrKind(relayFn, ir.IRDropOwned) != 0 || countInstrKind(relayFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("relay must not drop owned pointer stored in q and returned: %#v", relayFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "relay") != 1 {
		t.Fatalf("main must call relay once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("caller-owned local-relayed pointer must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesRejectsMovedFromLocalReturnInCallerContext(t *testing.T) {
	err := lowerOwnedAllocDropProgramError(t, `
func ret() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        let q: ptr = p
        return p

func main() -> Int
uses alloc, mem:
    unsafe:
        let r: ptr = ret()
    return 0
`)
	if err == nil || !strings.Contains(err.Error(), "use after move") {
		t.Fatalf("LowerWithOptions error = %v, want use after move", err)
	}
}

func TestLowerOwnedAllocBytesDoesNotSummarizeMixedBranchReturnAsOwned(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func choose(flag: Bool, borrowed: ptr) -> ptr
uses alloc, mem:
    if flag:
        return borrowed
    return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let borrowed: ptr = 0
        let r: ptr = choose(true, borrowed)
    return 0
`)
	chooseFn := findIRFunc(t, prog, "choose")
	if countCallsNamed(chooseFn.Instrs, "make") != 1 {
		t.Fatalf("choose must call make on the owned path: %#v", chooseFn.Instrs)
	}
	if countInstrKind(chooseFn, ir.IRDropOwned) != 0 || countInstrKind(chooseFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("choose must not drop either returned path in callee: %#v", chooseFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "choose") != 1 {
		t.Fatalf("main must call choose once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 0 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("mixed borrowed/owned branch return must not be summarized as caller-owned: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDoesNotSummarizeMixedIfLetReturnAsOwned(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func choose(value: Int?, borrowed: ptr) -> ptr
uses alloc, mem:
    if let some(x) = value:
        return borrowed
    return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let borrowed: ptr = 0
        let r: ptr = choose(1, borrowed)
    return 0
`)
	chooseFn := findIRFunc(t, prog, "choose")
	if countCallsNamed(chooseFn.Instrs, "make") != 1 {
		t.Fatalf("choose must call make on the owned if-let fallthrough path: %#v", chooseFn.Instrs)
	}
	if countInstrKind(chooseFn, ir.IRDropOwned) != 0 || countInstrKind(chooseFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("choose must not drop either returned if-let path in callee: %#v", chooseFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "choose") != 1 {
		t.Fatalf("main must call choose once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 0 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("mixed borrowed/owned if-let return must not be summarized as caller-owned: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSummarizesAllOwnedIfLetReturns(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func choose(value: Int?) -> ptr
uses alloc, mem:
    if let some(x) = value:
        return make()
    return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let r: ptr = choose(1)
    return 0
`)
	chooseFn := findIRFunc(t, prog, "choose")
	if countCallsNamed(chooseFn.Instrs, "make") != 2 {
		t.Fatalf("choose must call make on both if-let paths: %#v", chooseFn.Instrs)
	}
	if countInstrKind(chooseFn, ir.IRDropOwned) != 0 || countInstrKind(chooseFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("choose must not drop owned pointers it returns: %#v", chooseFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "choose") != 1 {
		t.Fatalf("main must call choose once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("all-owned if-let return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSummarizesAllOwnedBranchReturns(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func choose(flag: Bool) -> ptr
uses alloc, mem:
    if flag:
        return make()
    return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let r: ptr = choose(false)
    return 0
`)
	chooseFn := findIRFunc(t, prog, "choose")
	if countCallsNamed(chooseFn.Instrs, "make") != 2 {
		t.Fatalf("choose must call make on both paths: %#v", chooseFn.Instrs)
	}
	if countInstrKind(chooseFn, ir.IRDropOwned) != 0 || countInstrKind(chooseFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("choose must not drop owned pointers it returns: %#v", chooseFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "choose") != 1 {
		t.Fatalf("main must call choose once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("all-owned branch return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDoesNotSummarizeMixedMatchReturnAsOwned(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Choice:
    case borrowed
    case owned

func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func choose(choice: Choice, borrowed: ptr) -> ptr
uses alloc, mem:
    match choice:
    case Choice.borrowed:
        return borrowed
    case Choice.owned:
        return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let borrowed: ptr = 0
        let r: ptr = choose(Choice.borrowed, borrowed)
    return 0
`)
	chooseFn := findIRFunc(t, prog, "choose")
	if countCallsNamed(chooseFn.Instrs, "make") != 1 {
		t.Fatalf("choose must call make on the owned match path: %#v", chooseFn.Instrs)
	}
	if countInstrKind(chooseFn, ir.IRDropOwned) != 0 || countInstrKind(chooseFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("choose must not drop either returned match path in callee: %#v", chooseFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "choose") != 1 {
		t.Fatalf("main must call choose once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 0 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("mixed borrowed/owned match return must not be summarized as caller-owned: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSummarizesAllOwnedMatchReturns(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum Choice:
    case left
    case right

func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func choose(choice: Choice) -> ptr
uses alloc, mem:
    match choice:
    case Choice.left:
        return make()
    case Choice.right:
        return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let r: ptr = choose(Choice.left)
    return 0
`)
	chooseFn := findIRFunc(t, prog, "choose")
	if countCallsNamed(chooseFn.Instrs, "make") != 2 {
		t.Fatalf("choose must call make on both match paths: %#v", chooseFn.Instrs)
	}
	if countInstrKind(chooseFn, ir.IRDropOwned) != 0 || countInstrKind(chooseFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("choose must not drop owned pointers it returns: %#v", chooseFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "choose") != 1 {
		t.Fatalf("main must call choose once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("all-owned match return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDoesNotSummarizeMixedWhileReturnAsOwned(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func choose(flag: Bool, borrowed: ptr) -> ptr
uses alloc, mem:
    while flag:
        return borrowed
    return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let borrowed: ptr = 0
        let r: ptr = choose(true, borrowed)
    return 0
`)
	chooseFn := findIRFunc(t, prog, "choose")
	if countCallsNamed(chooseFn.Instrs, "make") != 1 {
		t.Fatalf("choose must call make on the owned fallthrough path: %#v", chooseFn.Instrs)
	}
	if countInstrKind(chooseFn, ir.IRDropOwned) != 0 || countInstrKind(chooseFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("choose must not drop either returned path in callee: %#v", chooseFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "choose") != 1 {
		t.Fatalf("main must call choose once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 0 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("mixed borrowed/owned while return must not be summarized as caller-owned: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSummarizesAllOwnedWhileReturns(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func choose(flag: Bool) -> ptr
uses alloc, mem:
    while flag:
        return make()
    return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let r: ptr = choose(false)
    return 0
`)
	chooseFn := findIRFunc(t, prog, "choose")
	if countCallsNamed(chooseFn.Instrs, "make") != 2 {
		t.Fatalf("choose must call make on loop and fallthrough paths: %#v", chooseFn.Instrs)
	}
	if countInstrKind(chooseFn, ir.IRDropOwned) != 0 || countInstrKind(chooseFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("choose must not drop owned pointers it returns: %#v", chooseFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "choose") != 1 {
		t.Fatalf("main must call choose once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("all-owned while return must be dropped in caller: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDoesNotSummarizeMixedForRangeReturnAsOwned(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func choose(borrowed: ptr) -> ptr
uses alloc, mem:
    for i in 0..<1:
        return borrowed
    return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let borrowed: ptr = 0
        let r: ptr = choose(borrowed)
    return 0
`)
	chooseFn := findIRFunc(t, prog, "choose")
	if countCallsNamed(chooseFn.Instrs, "make") != 1 {
		t.Fatalf("choose must call make on the owned for-range fallthrough path: %#v", chooseFn.Instrs)
	}
	if countInstrKind(chooseFn, ir.IRDropOwned) != 0 || countInstrKind(chooseFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("choose must not drop either returned for-range path in callee: %#v", chooseFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "choose") != 1 {
		t.Fatalf("main must call choose once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 0 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("mixed borrowed/owned for-range return must not be summarized as caller-owned: %#v", mainFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesSummarizesAllOwnedForRangeReturns(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
func make() -> ptr
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        return p

func choose() -> ptr
uses alloc, mem:
    for i in 0..<1:
        return make()
    return make()

func main() -> Int
uses alloc, mem:
    unsafe:
        let r: ptr = choose()
    return 0
`)
	chooseFn := findIRFunc(t, prog, "choose")
	if countCallsNamed(chooseFn.Instrs, "make") != 2 {
		t.Fatalf("choose must call make on loop and fallthrough paths: %#v", chooseFn.Instrs)
	}
	if countInstrKind(chooseFn, ir.IRDropOwned) != 0 || countInstrKind(chooseFn, ir.IRReleaseAllocation) != 0 {
		t.Fatalf("choose must not drop owned pointers it returns: %#v", chooseFn.Instrs)
	}
	mainFn := findIRFunc(t, prog, "main")
	if countCallsNamed(mainFn.Instrs, "choose") != 1 {
		t.Fatalf("main must call choose once: %#v", mainFn.Instrs)
	}
	if countInstrKind(mainFn, ir.IRDropOwned) != 1 || countInstrKind(mainFn, ir.IRReleaseAllocation) != 1 {
		t.Fatalf("all-owned for-range return must be dropped in caller: %#v", mainFn.Instrs)
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
				t.Fatalf(
					"island make slice allocation name = %q, want xs: %#v",
					instr.Name,
					fn.Instrs,
				)
			}
			return
		}
	}
	t.Fatalf("main missing island make slice: %#v", fn.Instrs)
}

func TestLowerInlineExplicitIslandMakeSliceCarriesAllocationNames(t *testing.T) {
	prog := lowerStackAllocationProgram(t, `
struct CachePair:
    hot: []u8
    cold: []u8

func make_pair(hot_region: island, cold_region: island) -> CachePair
uses alloc, islands, mem:
    return CachePair(hot: core.island_make_u8(hot_region, 2), cold: core.island_make_u8(cold_region, 2))

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "make_pair")
	names := map[string]bool{}
	for _, instr := range fn.Instrs {
		if instr.Kind != ir.IRIslandMakeSliceU8 {
			continue
		}
		if instr.Name == "" {
			t.Fatalf("inline island make slice missing allocation name: %#v", fn.Instrs)
		}
		names[instr.Name] = true
	}
	if len(names) != 2 {
		t.Fatalf(
			"inline island allocation names = %#v, want two distinct names: %#v",
			names,
			fn.Instrs,
		)
	}
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
		t.Fatalf(
			"copy_into IR missing prefix guard or store: prefix=%d store=%d instrs=%#v",
			prefixAt,
			storeAt,
			fn.Instrs,
		)
	}
	if prefixAt > storeAt {
		t.Fatalf(
			"copy_into writes before destination length guard: prefix=%d store=%d instrs=%#v",
			prefixAt,
			storeAt,
			fn.Instrs,
		)
	}
}

func TestJsonParseStringifyWriteMessageObjectUsesProofTaggedStores(t *testing.T) {
	prog := lowerProofFileProgram(
		t,
		jsonParseStringifyHelperSummarySource(128, "write_message_object(buf)"),
	)
	fn := findIRFunc(t, prog, "p25.json_parse_stringify.write_message_object")
	if got := countInstrKind(fn, ir.IRIndexStoreU8); got != 27 {
		t.Fatalf("write_message_object u8 stores = %d, want 27; instrs=%#v", got, fn.Instrs)
	}
	if got := countProofTaggedInstrKind(fn, ir.IRIndexStoreU8); got != 27 {
		t.Fatalf(
			"write_message_object proof-tagged stores = %d, want 27; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
	for _, proofID := range proofIDsForKind(fn, ir.IRIndexStoreU8) {
		if !strings.HasPrefix(proofID, "proof:helper-summary:") {
			t.Fatalf(
				"write_message_object store proof id = %q, want proof:helper-summary prefix; instrs=%#v",
				proofID,
				fn.Instrs,
			)
		}
	}
}

func TestJsonParseStringifyShortBufferKeepsOutOfRangeStoreChecked(t *testing.T) {
	prog := lowerProofFileProgram(
		t,
		jsonParseStringifyHelperSummarySource(26, "write_message_object(buf)"),
	)
	fn := findIRFunc(t, prog, "p25.json_parse_stringify.write_message_object")
	if got := countProofTaggedInstrKind(fn, ir.IRIndexStoreU8); got != 0 {
		t.Fatalf(
			"short buffer helper-summary proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstrKind(fn, ir.IRIndexStoreU8); got != 27 {
		t.Fatalf(
			"short buffer checked/proof-tagged u8 stores = %d, want 27 retained store ops; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestJsonParseStringifyDynamicIndexDoesNotUseHelperSummaryProof(t *testing.T) {
	prog := lowerProofFileProgram(t, `
module p25.json_parse_stringify

func write_message_object(dst: inout []u8, i: Int) -> Int
uses mem:
    dst[i] = 125
    return 1

func main() -> Int
uses alloc, mem:
    var buf: []u8 = core.make_u8(128)
    return write_message_object(buf, 0)
`)
	fn := findIRFunc(t, prog, "p25.json_parse_stringify.write_message_object")
	if got := countProofTaggedInstrKind(fn, ir.IRIndexStoreU8); got != 0 {
		t.Fatalf(
			"dynamic index helper-summary proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstrKind(fn, ir.IRIndexStoreU8); got != 1 {
		t.Fatalf("dynamic index checked u8 stores = %d, want 1; instrs=%#v", got, fn.Instrs)
	}
}

func TestJsonParseStringifyUnrelatedPublicHelperDoesNotInheritHelperSummaryProof(t *testing.T) {
	prog := lowerProofFileProgram(t, `
module p25.json_parse_stringify

func write_unrelated(dst: inout []u8) -> Int
uses mem:
    dst[0] = 125
    return 1

func main() -> Int
uses alloc, mem:
    var buf: []u8 = core.make_u8(128)
    return 0
`)
	fn := findIRFunc(t, prog, "p25.json_parse_stringify.write_unrelated")
	if got := countProofTaggedInstrKind(fn, ir.IRIndexStoreU8); got != 0 {
		t.Fatalf(
			"unrelated helper inherited helper-summary proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestJsonParseStringifyUnsafeHelperDoesNotUseHelperSummaryProof(t *testing.T) {
	prog := lowerProofFileProgram(t, `
module p25.json_parse_stringify

func write_message_object(dst: inout []u8) -> Int
uses mem:
    unsafe:
        dst[0] = 125
    return 1

func main() -> Int
uses alloc, mem:
    var buf: []u8 = core.make_u8(128)
    return write_message_object(buf)
`)
	fn := findIRFunc(t, prog, "p25.json_parse_stringify.write_message_object")
	if got := countProofTaggedInstrKind(fn, ir.IRIndexStoreU8); got != 0 {
		t.Fatalf(
			"unsafe helper-summary proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestPostgreSQLHelperOffsetUsesProofTaggedAccesses(t *testing.T) {
	prog := lowerProofFileProgram(t, postgresqlHelperOffsetSource(64, `
        frame[0] = frame_data_row()
        var pos: Int = write_i32_be_at(frame, 1, 12)
        pos = write_i16_be_at(frame, pos, 2)
        total = total + frame_type_at(frame, 0) + frame_payload_start(0)
`))

	frameType := findIRFunc(t, prog, "p25.postgresql_single_multiple_update.frame_type_at")
	if got := countInstrKind(frameType, ir.IRIndexLoadU8); got != 0 {
		t.Fatalf("frame_type_at checked u8 loads = %d, want 0; instrs=%#v", got, frameType.Instrs)
	}
	if got := countInstrKind(frameType, ir.IRIndexLoadU8Unchecked); got != 1 {
		t.Fatalf("frame_type_at unchecked u8 loads = %d, want 1; instrs=%#v", got, frameType.Instrs)
	}
	if got := firstProofID(frameType, ir.IRIndexLoadU8Unchecked); !strings.HasPrefix(
		got,
		"proof:helper-offset:",
	) {
		t.Fatalf(
			"frame_type_at load proof id = %q, want proof:helper-offset prefix; instrs=%#v",
			got,
			frameType.Instrs,
		)
	}

	write32 := findIRFunc(t, prog, "p25.postgresql_single_multiple_update.write_i32_be_at")
	if got := countInstrKind(write32, ir.IRIndexStoreU8); got != 4 {
		t.Fatalf("write_i32_be_at stores = %d, want 4; instrs=%#v", got, write32.Instrs)
	}
	if got := countProofTaggedInstrKind(write32, ir.IRIndexStoreU8); got != 4 {
		t.Fatalf(
			"write_i32_be_at proof-tagged stores = %d, want 4; instrs=%#v",
			got,
			write32.Instrs,
		)
	}
	for _, proofID := range proofIDsForKind(write32, ir.IRIndexStoreU8) {
		if !strings.HasPrefix(proofID, "proof:helper-offset:") {
			t.Fatalf(
				"write_i32_be_at store proof id = %q, want proof:helper-offset prefix; instrs=%#v",
				proofID,
				write32.Instrs,
			)
		}
	}

	write16 := findIRFunc(t, prog, "p25.postgresql_single_multiple_update.write_i16_be_at")
	if got := countInstrKind(write16, ir.IRIndexStoreU8); got != 2 {
		t.Fatalf("write_i16_be_at stores = %d, want 2; instrs=%#v", got, write16.Instrs)
	}
	if got := countProofTaggedInstrKind(write16, ir.IRIndexStoreU8); got != 2 {
		t.Fatalf(
			"write_i16_be_at proof-tagged stores = %d, want 2; instrs=%#v",
			got,
			write16.Instrs,
		)
	}
	for _, proofID := range proofIDsForKind(write16, ir.IRIndexStoreU8) {
		if !strings.HasPrefix(proofID, "proof:helper-offset:") {
			t.Fatalf(
				"write_i16_be_at store proof id = %q, want proof:helper-offset prefix; instrs=%#v",
				proofID,
				write16.Instrs,
			)
		}
	}
}

func TestPostgreSQLHelperOffsetRejectsShortBuffer(t *testing.T) {
	prog := lowerProofFileProgram(t, postgresqlHelperOffsetSource(4, `
        var pos: Int = write_i32_be_at(frame, 1, 12)
        total = total + pos
`))
	fn := findIRFunc(t, prog, "p25.postgresql_single_multiple_update.write_i32_be_at")
	if got := countProofIDPrefixForKind(fn, ir.IRIndexStoreU8, "proof:helper-offset:"); got != 0 {
		t.Fatalf(
			"short buffer helper-offset proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstrKind(fn, ir.IRIndexStoreU8); got != 4 {
		t.Fatalf(
			"short buffer checked stores = %d, want 4 retained store ops; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestPostgreSQLHelperOffsetRejectsUnsafeStart(t *testing.T) {
	prog := lowerProofFileProgram(t, postgresqlHelperOffsetSource(64, `
        var pos: Int = write_i32_be_at(frame, 61, 12)
        total = total + pos
`))
	fn := findIRFunc(t, prog, "p25.postgresql_single_multiple_update.write_i32_be_at")
	if got := countProofIDPrefixForKind(fn, ir.IRIndexStoreU8, "proof:helper-offset:"); got != 0 {
		t.Fatalf(
			"unsafe start helper-offset proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestPostgreSQLHelperOffsetRejectsUnknownStart(t *testing.T) {
	prog := lowerProofFileProgram(t, postgresqlHelperOffsetSource(64, `
        var start: Int = total
        var pos: Int = write_i32_be_at(frame, start, 12)
        total = total + pos
`))
	fn := findIRFunc(t, prog, "p25.postgresql_single_multiple_update.write_i32_be_at")
	if got := countProofIDPrefixForKind(fn, ir.IRIndexStoreU8, "proof:helper-offset:"); got != 0 {
		t.Fatalf(
			"unknown start helper-offset proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestPostgreSQLHelperOffsetRejectsUnsafeLoadOffsets(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "negative",
			body: `
        total = total + frame_type_at(frame, -1)
`,
		},
		{
			name: "unknown",
			body: `
        var offset: Int = total
        total = total + frame_type_at(frame, offset)
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := lowerProofFileProgram(t, postgresqlHelperOffsetSource(64, tt.body))
			fn := findIRFunc(t, prog, "p25.postgresql_single_multiple_update.frame_type_at")
			if got := countInstrKind(fn, ir.IRIndexLoadU8Unchecked); got != 0 {
				t.Fatalf(
					"%s load helper unchecked loads = %d, want 0; instrs=%#v",
					tt.name,
					got,
					fn.Instrs,
				)
			}
			if got := countInstrKind(fn, ir.IRIndexLoadU8); got != 1 {
				t.Fatalf(
					"%s load helper checked loads = %d, want 1; instrs=%#v",
					tt.name,
					got,
					fn.Instrs,
				)
			}
		})
	}
}

func TestPostgreSQLHelperOffsetRejectsMixedSafeUnsafeCallSites(t *testing.T) {
	prog := lowerProofFileProgram(t, postgresqlHelperOffsetSource(64, `
        var safe_pos: Int = write_i32_be_at(frame, 1, 12)
        var unsafe_pos: Int = write_i32_be_at(frame, 61, 12)
        total = total + safe_pos + unsafe_pos
`))
	fn := findIRFunc(t, prog, "p25.postgresql_single_multiple_update.write_i32_be_at")
	if got := countProofIDPrefixForKind(fn, ir.IRIndexStoreU8, "proof:helper-offset:"); got != 0 {
		t.Fatalf(
			"mixed call sites helper-offset proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestHTTPPlaintextJSONDoesNotUseHelperOffsetProofs(t *testing.T) {
	prog := lowerProofFileProgram(t, httpPlaintextJSONHelperOffsetNonTargetSource)
	for _, name := range []string{
		"p25.http_plaintext_json.write_plaintext_response",
		"p25.http_plaintext_json.write_json_response",
	} {
		fn := findIRFunc(t, prog, name)
		if got := countProofIDPrefixForKind(fn, ir.IRIndexStoreU8, "proof:helper-offset:"); got != 0 {
			t.Fatalf(
				"%s helper-offset proof-tagged stores = %d, want 0; instrs=%#v",
				name,
				got,
				fn.Instrs,
			)
		}
	}
}

func TestHTTPPlaintextJSONMultiHelperSummaryUsesProofTaggedStores(t *testing.T) {
	prog := lowerProofFileProgram(
		t,
		httpPlaintextJSONHelperSummarySource(192, 192, "", httpPlaintextJSONDefaultLoopBody),
	)
	tests := []struct {
		fnName string
		want   int
	}{
		{fnName: "p25.http_plaintext_json.write_plaintext_response", want: 24},
		{fnName: "p25.http_plaintext_json.write_json_response", want: 21},
	}
	for _, tt := range tests {
		t.Run(tt.fnName, func(t *testing.T) {
			fn := findIRFunc(t, prog, tt.fnName)
			if got := countInstrKind(fn, ir.IRIndexStoreU8); got != tt.want {
				t.Fatalf(
					"%s u8 stores = %d, want %d; instrs=%#v",
					tt.fnName,
					got,
					tt.want,
					fn.Instrs,
				)
			}
			if got := countProofTaggedInstrKind(fn, ir.IRIndexStoreU8); got != tt.want {
				t.Fatalf(
					"%s proof-tagged stores = %d, want %d; instrs=%#v",
					tt.fnName,
					got,
					tt.want,
					fn.Instrs,
				)
			}
			for _, proofID := range proofIDsForKind(fn, ir.IRIndexStoreU8) {
				if !strings.HasPrefix(proofID, "proof:helper-summary:") {
					t.Fatalf(
						"%s store proof id = %q, want proof:helper-summary prefix; instrs=%#v",
						tt.fnName,
						proofID,
						fn.Instrs,
					)
				}
				if strings.HasPrefix(proofID, "proof:helper-offset:") {
					t.Fatalf(
						"%s accidentally used helper-offset proof id %q; instrs=%#v",
						tt.fnName,
						proofID,
						fn.Instrs,
					)
				}
			}
		})
	}
}

func TestHTTPPlaintextJSONShortPlainRejectsOnlyPlaintextHelperSummary(t *testing.T) {
	prog := lowerProofFileProgram(
		t,
		httpPlaintextJSONHelperSummarySource(23, 192, "", httpPlaintextJSONDefaultLoopBody),
	)
	plain := findIRFunc(t, prog, "p25.http_plaintext_json.write_plaintext_response")
	if got := countProofTaggedInstrKind(plain, ir.IRIndexStoreU8); got != 0 {
		t.Fatalf(
			"short plain helper-summary proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			plain.Instrs,
		)
	}
	if got := countInstrKind(plain, ir.IRIndexStoreU8); got != 24 {
		t.Fatalf("short plain retained stores = %d, want 24; instrs=%#v", got, plain.Instrs)
	}
	json := findIRFunc(t, prog, "p25.http_plaintext_json.write_json_response")
	if got := countProofIDPrefixForKind(json, ir.IRIndexStoreU8, "proof:helper-summary:"); got != 21 {
		t.Fatalf(
			"safe json helper-summary proof-tagged stores = %d, want 21; instrs=%#v",
			got,
			json.Instrs,
		)
	}
}

func TestHTTPPlaintextJSONShortJSONRejectsOnlyJSONHelperSummary(t *testing.T) {
	prog := lowerProofFileProgram(
		t,
		httpPlaintextJSONHelperSummarySource(192, 20, "", httpPlaintextJSONDefaultLoopBody),
	)
	plain := findIRFunc(t, prog, "p25.http_plaintext_json.write_plaintext_response")
	if got := countProofIDPrefixForKind(plain, ir.IRIndexStoreU8, "proof:helper-summary:"); got != 24 {
		t.Fatalf(
			"safe plain helper-summary proof-tagged stores = %d, want 24; instrs=%#v",
			got,
			plain.Instrs,
		)
	}
	json := findIRFunc(t, prog, "p25.http_plaintext_json.write_json_response")
	if got := countProofTaggedInstrKind(json, ir.IRIndexStoreU8); got != 0 {
		t.Fatalf(
			"short json_buf helper-summary proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			json.Instrs,
		)
	}
	if got := countInstrKind(json, ir.IRIndexStoreU8); got != 21 {
		t.Fatalf("short json_buf retained stores = %d, want 21; instrs=%#v", got, json.Instrs)
	}
}

func TestHTTPPlaintextJSONMixedSafeUnsafeCallSitesRejectSameHelper(t *testing.T) {
	prog := lowerProofFileProgram(t, httpPlaintextJSONHelperSummarySource(192, 192, `
    var short_plain: []u8 = core.make_u8(8)
`, `
        total = total + write_plaintext_response(plain)
        total = total + write_plaintext_response(short_plain)
        total = total + write_json_response(json_buf)
`))
	plain := findIRFunc(t, prog, "p25.http_plaintext_json.write_plaintext_response")
	if got := countProofTaggedInstrKind(plain, ir.IRIndexStoreU8); got != 0 {
		t.Fatalf(
			"mixed safe/short plaintext call sites proof-tagged stores = %d, want 0; instrs=%#v",
			got,
			plain.Instrs,
		)
	}
	json := findIRFunc(t, prog, "p25.http_plaintext_json.write_json_response")
	if got := countProofIDPrefixForKind(json, ir.IRIndexStoreU8, "proof:helper-summary:"); got != 21 {
		t.Fatalf(
			"unmixed json helper-summary proof-tagged stores = %d, want 21; instrs=%#v",
			got,
			json.Instrs,
		)
	}
}

func TestHTTPHelperSummaryRejectsDynamicUnsafeAndParameterReadReturn(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "dynamic_index",
			src: `
module p25.http_plaintext_json

func write_plaintext_response(dst: inout []u8, i: Int) -> Int
uses mem:
    dst[i] = 72
    return 1

func main() -> Int
uses alloc, mem:
    var plain: []u8 = core.make_u8(192)
    return write_plaintext_response(plain, 0)
`,
		},
		{
			name: "unsafe_body",
			src: `
module p25.http_plaintext_json

func write_plaintext_response(dst: inout []u8) -> Int
uses mem:
    unsafe:
        dst[0] = 72
    return 1

func main() -> Int
uses alloc, mem:
    var plain: []u8 = core.make_u8(192)
    return write_plaintext_response(plain)
`,
		},
		{
			name: "parameter_read_return",
			src: `
module p25.http_plaintext_json

func write_plaintext_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    return dst[0]

func main() -> Int
uses alloc, mem:
    var plain: []u8 = core.make_u8(192)
    return write_plaintext_response(plain)
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := lowerProofFileProgram(t, tt.src)
			fn := findIRFunc(t, prog, "p25.http_plaintext_json.write_plaintext_response")
			if got := countProofIDPrefixForKind(fn, ir.IRIndexStoreU8, "proof:helper-summary:"); got != 0 {
				t.Fatalf(
					"%s helper-summary proof-tagged stores = %d, want 0; instrs=%#v",
					tt.name,
					got,
					fn.Instrs,
				)
			}
		})
	}
}

func jsonParseStringifyHelperSummarySource(bufLen int, call string) string {
	return strings.NewReplacer(
		"$LEN", strconv.Itoa(bufLen),
		"$CALL", call,
	).Replace(`
module p25.json_parse_stringify

func write_message_object(dst: inout []u8) -> Int
uses mem:
    dst[0] = 123
    dst[1] = 34
    dst[2] = 109
    dst[3] = 101
    dst[4] = 115
    dst[5] = 115
    dst[6] = 97
    dst[7] = 103
    dst[8] = 101
    dst[9] = 34
    dst[10] = 58
    dst[11] = 34
    dst[12] = 72
    dst[13] = 101
    dst[14] = 108
    dst[15] = 108
    dst[16] = 111
    dst[17] = 44
    dst[18] = 32
    dst[19] = 87
    dst[20] = 111
    dst[21] = 114
    dst[22] = 108
    dst[23] = 100
    dst[24] = 33
    dst[25] = 34
    dst[26] = 125
    return 27

func main() -> Int
uses alloc, mem:
    var buf: []u8 = core.make_u8($LEN)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        total = total + $CALL
        i = i + 1
    if total == 55296:
        return 0
    return 1
`)
}

const httpPlaintextJSONDefaultLoopBody = `
        total = total + write_plaintext_response(plain)
        total = total + write_json_response(json_buf)
`

func httpPlaintextJSONHelperSummarySource(
	plainLen int,
	jsonLen int,
	extraLocals string,
	loopBody string,
) string {
	return strings.NewReplacer(
		"$PLAIN_LEN", strconv.Itoa(plainLen),
		"$JSON_LEN", strconv.Itoa(jsonLen),
		"$EXTRA_LOCALS", strings.Trim(extraLocals, "\n"),
		"$LOOP_BODY", strings.Trim(loopBody, "\n"),
	).Replace(`
module p25.http_plaintext_json

func write_plaintext_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 72
    dst[20] = 101
    dst[21] = 108
    dst[22] = 108
    dst[23] = 111
    return 24

func write_json_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 123
    dst[20] = 125
    return 21

func main() -> Int
uses alloc, mem:
    var plain: []u8 = core.make_u8($PLAIN_LEN)
    var json_buf: []u8 = core.make_u8($JSON_LEN)
$EXTRA_LOCALS
    var i: Int = 0
    var total: Int = 0
    while i < 1024:
$LOOP_BODY
        i = i + 1
    if total > 0:
        return 0
    return 1
`)
}

func postgresqlHelperOffsetSource(frameLen int, loopBody string) string {
	return strings.NewReplacer(
		"$LEN", strconv.Itoa(frameLen),
		"$LOOP_BODY", strings.Trim(loopBody, "\n"),
	).Replace(`
module p25.postgresql_single_multiple_update

func frame_data_row() -> Int:
    return 68

func frame_payload_start(offset: Int) -> Int:
    return offset + 5

func frame_type_at(src: []u8, offset: Int) -> Int
uses mem:
    return src[offset]

func write_i32_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 16777216) % 256
    dst[start + 1] = (value / 65536) % 256
    dst[start + 2] = (value / 256) % 256
    dst[start + 3] = value % 256
    return start + 4

func write_i16_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 256) % 256
    dst[start + 1] = value % 256
    return start + 2

func main() -> Int
uses alloc, mem:
    var frame: []u8 = core.make_u8($LEN)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
$LOOP_BODY
        i = i + 1
    if total > 0:
        return 0
    return 1
`)
}

const httpPlaintextJSONHelperOffsetNonTargetSource = `
module p25.http_plaintext_json

func write_plaintext_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 72
    dst[20] = 101
    dst[21] = 108
    dst[22] = 108
    dst[23] = 111
    return 24

func write_json_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 123
    dst[20] = 125
    return 21

func main() -> Int
uses alloc, mem:
    var plain: []u8 = core.make_u8(192)
    var json_buf: []u8 = core.make_u8(192)
    var i: Int = 0
    var total: Int = 0
    while i < 1024:
        total = total + write_plaintext_response(plain)
        total = total + write_json_response(json_buf)
        i = i + 1
    if total > 0:
        return 0
    return 1
`

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
	if countInstrKind(fn, ir.IRStackSliceI32) != 0 || countInstrKind(fn, ir.IRMakeSliceI32) != 0 ||
		countInstrKind(fn, ir.IRAllocBytes) != 0 {
		t.Fatalf("scalar replacement still emitted allocation IR: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 ||
		countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 ||
		countInstrKind(fn, ir.IRIndexStoreI32) != 0 {
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
		t.Fatalf(
			"small struct should occupy scalar local slots, got LocalSlots=%d instrs=%#v",
			fn.LocalSlots,
			fn.Instrs,
		)
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
		t.Fatalf(
			"small fixed array should occupy scalar local slots, got LocalSlots=%d instrs=%#v",
			fn.LocalSlots,
			fn.Instrs,
		)
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
		t.Fatalf(
			"dynamic-index tiny slice stack slice count = %d, want 1: %#v",
			countInstrKind(fn, ir.IRStackSliceI32),
			fn.Instrs,
		)
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

func assertRegionResetImmediatelyBeforeEveryReturn(t *testing.T, fn ir.IRFunc) {
	t.Helper()
	returns := 0
	for i, instr := range fn.Instrs {
		if instr.Kind != ir.IRReturn {
			continue
		}
		returns++
		if i == 0 || fn.Instrs[i-1].Kind != ir.IRRegionReset {
			t.Fatalf(
				"%s return at instruction %d is not immediately preceded by region reset: %#v",
				fn.Name,
				i,
				fn.Instrs,
			)
		}
	}
	if returns == 0 {
		t.Fatalf("%s has no return instructions: %#v", fn.Name, fn.Instrs)
	}
	if got := countInstrKind(fn, ir.IRRegionReset); got != returns {
		t.Fatalf(
			"%s region reset count = %d, want one per return/throw exit (%d): %#v",
			fn.Name,
			got,
			returns,
			fn.Instrs,
		)
	}
}

func assertOwnedReleaseBeforeFirstReturnValue(t *testing.T, fn ir.IRFunc) {
	t.Helper()
	releaseIndex := -1
	returnIndex := -1
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRReleaseAllocation && releaseIndex < 0 {
			releaseIndex = i
		}
		if instr.Kind == ir.IRReturn {
			returnIndex = i
			break
		}
	}
	if releaseIndex < 0 {
		t.Fatalf("%s has no IRReleaseAllocation: %#v", fn.Name, fn.Instrs)
	}
	if returnIndex < 0 {
		t.Fatalf("%s has no IRReturn: %#v", fn.Name, fn.Instrs)
	}
	returnValueIndex := -1
	for i := releaseIndex + 1; i < returnIndex; i++ {
		if fn.Instrs[i].Kind == ir.IRConstI32 && fn.Instrs[i].Imm == 0 {
			returnValueIndex = i
			break
		}
	}
	if returnValueIndex < 0 {
		t.Fatalf(
			"%s release index %d must precede a following return value before return %d: %#v",
			fn.Name,
			releaseIndex,
			returnIndex,
			fn.Instrs,
		)
	}
}

func assertOwnedReleaseBeforeFirstReturn(t *testing.T, fn ir.IRFunc) {
	t.Helper()
	releaseIndex := -1
	returnIndex := -1
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRReleaseAllocation && releaseIndex < 0 {
			releaseIndex = i
		}
		if instr.Kind == ir.IRReturn {
			returnIndex = i
			break
		}
	}
	if releaseIndex < 0 {
		t.Fatalf("%s has no IRReleaseAllocation: %#v", fn.Name, fn.Instrs)
	}
	if returnIndex < 0 {
		t.Fatalf("%s has no IRReturn: %#v", fn.Name, fn.Instrs)
	}
	if releaseIndex > returnIndex {
		t.Fatalf(
			"%s release index %d must precede first return %d: %#v",
			fn.Name,
			releaseIndex,
			returnIndex,
			fn.Instrs,
		)
	}
}

func assertOwnedReleaseBeforeEveryReturn(t *testing.T, fn ir.IRFunc) {
	t.Helper()
	releases := 0
	returns := 0
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRReleaseAllocation {
			releases++
		}
		if instr.Kind != ir.IRReturn {
			continue
		}
		if releases <= returns {
			t.Fatalf(
				"%s return at instruction %d has no preceding owned release for that exit: %#v",
				fn.Name,
				i,
				fn.Instrs,
			)
		}
		returns++
	}
	if returns == 0 {
		t.Fatalf("%s has no IRReturn: %#v", fn.Name, fn.Instrs)
	}
}

func assertOwnedReleaseBeforeFirstJmpIfZeroTarget(t *testing.T, fn ir.IRFunc) {
	t.Helper()
	target := -1
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRJmpIfZero {
			target = instr.Label
			break
		}
	}
	if target < 0 {
		t.Fatalf("%s has no IRJmpIfZero: %#v", fn.Name, fn.Instrs)
	}
	targetIndex := -1
	releaseIndex := -1
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRReleaseAllocation && releaseIndex < 0 {
			releaseIndex = i
		}
		if instr.Kind == ir.IRLabel && instr.Label == target {
			targetIndex = i
		}
	}
	if releaseIndex < 0 {
		t.Fatalf("%s has no IRReleaseAllocation: %#v", fn.Name, fn.Instrs)
	}
	if targetIndex < 0 {
		t.Fatalf("%s has no label %d: %#v", fn.Name, target, fn.Instrs)
	}
	if releaseIndex >= targetIndex {
		t.Fatalf(
			"%s release index %d must be before branch join label index %d: %#v",
			fn.Name,
			releaseIndex,
			targetIndex,
			fn.Instrs,
		)
	}
}

func assertReleaseLayoutInFirstCatchSuccessBranch(t *testing.T, fn ir.IRFunc, layoutID string) {
	t.Helper()
	successLabel := -1
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRJmpIfZero {
			successLabel = instr.Label
			break
		}
	}
	if successLabel < 0 {
		t.Fatalf("%s has no catch success branch jump: %#v", fn.Name, fn.Instrs)
	}
	successIndex := -1
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel && instr.Label == successLabel {
			successIndex = i
			break
		}
	}
	if successIndex < 0 {
		t.Fatalf("%s has no catch success label %d: %#v", fn.Name, successLabel, fn.Instrs)
	}
	successEndJump := -1
	releaseIndex := -1
	for i := successIndex + 1; i < len(fn.Instrs); i++ {
		instr := fn.Instrs[i]
		if instr.Kind == ir.IRReleaseAllocation && instr.LayoutID == layoutID {
			releaseIndex = i
		}
		if instr.Kind == ir.IRJmp {
			successEndJump = i
			break
		}
	}
	if successEndJump < 0 {
		t.Fatalf("%s catch success branch has no end jump after label %d: %#v", fn.Name, successLabel, fn.Instrs)
	}
	if releaseIndex < 0 {
		t.Fatalf(
			"%s catch success branch must release %s before end jump %d: %#v",
			fn.Name,
			layoutID,
			successEndJump,
			fn.Instrs,
		)
	}
}

func assertReleaseLayoutInCatchCaseBranch(t *testing.T, fn ir.IRFunc, caseIndex int, layoutID string) {
	t.Helper()
	caseLabels := catchDispatchCaseLabels(fn)
	if caseIndex < 0 || caseIndex >= len(caseLabels) {
		t.Fatalf("%s has %d catch case labels, need case %d: %#v", fn.Name, len(caseLabels), caseIndex, fn.Instrs)
	}
	caseLabel := caseLabels[caseIndex]
	caseStart := -1
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel && instr.Label == caseLabel {
			caseStart = i
			break
		}
	}
	if caseStart < 0 {
		t.Fatalf("%s has no catch case label %d: %#v", fn.Name, caseLabel, fn.Instrs)
	}
	caseEndJump := -1
	releaseIndex := -1
	for i := caseStart + 1; i < len(fn.Instrs); i++ {
		instr := fn.Instrs[i]
		if instr.Kind == ir.IRReleaseAllocation && instr.LayoutID == layoutID {
			releaseIndex = i
		}
		if instr.Kind == ir.IRJmp {
			caseEndJump = i
			break
		}
		if instr.Kind == ir.IRLabel {
			break
		}
	}
	if caseEndJump < 0 {
		t.Fatalf("%s catch case %d has no end jump after label %d: %#v", fn.Name, caseIndex, caseLabel, fn.Instrs)
	}
	if releaseIndex < 0 {
		t.Fatalf(
			"%s catch case %d must release %s before end jump %d: %#v",
			fn.Name,
			caseIndex,
			layoutID,
			caseEndJump,
			fn.Instrs,
		)
	}
}

func assertReleaseLayoutInAnyCatchCaseBranch(t *testing.T, fn ir.IRFunc, layoutID string) {
	t.Helper()
	for i, instr := range fn.Instrs {
		if instr.Kind != ir.IRReleaseAllocation || instr.LayoutID != layoutID {
			continue
		}
		if releaseIsInsideLabelToJumpBranch(fn.Instrs, i) {
			return
		}
	}
	t.Fatalf("%s must release %s inside a catch case branch: %#v", fn.Name, layoutID, fn.Instrs)
}

func assertReleaseLayoutGuardedByTagEqInAnyBranch(t *testing.T, fn ir.IRFunc, layoutID string, tagValue int32) {
	t.Helper()
	for releaseIndex, instr := range fn.Instrs {
		if instr.Kind != ir.IRReleaseAllocation || instr.LayoutID != layoutID {
			continue
		}
		branchStart := previousLabelIndex(fn.Instrs, releaseIndex)
		if branchStart < 0 {
			continue
		}
		guardIndex := -1
		guardLabel := -1
		for i := branchStart + 1; i+3 < releaseIndex; i++ {
			if fn.Instrs[i].Kind == ir.IRLoadLocal &&
				fn.Instrs[i+1].Kind == ir.IRConstI32 &&
				fn.Instrs[i+1].Imm == tagValue &&
				fn.Instrs[i+2].Kind == ir.IRCmpEqI32 &&
				fn.Instrs[i+3].Kind == ir.IRJmpIfZero {
				guardIndex = i
				guardLabel = fn.Instrs[i+3].Label
			}
		}
		if guardIndex < 0 {
			continue
		}
		for i := releaseIndex + 1; i < len(fn.Instrs); i++ {
			if fn.Instrs[i].Kind == ir.IRLabel && fn.Instrs[i].Label == guardLabel {
				return
			}
			if fn.Instrs[i].Kind == ir.IRJmp {
				break
			}
		}
	}
	t.Fatalf("%s release %s must be guarded by enum tag == %d inside a catch case branch: %#v", fn.Name, layoutID, tagValue, fn.Instrs)
}

func releaseIsInsideLabelToJumpBranch(instrs []ir.IRInstr, releaseIndex int) bool {
	if previousLabelIndex(instrs, releaseIndex) < 0 {
		return false
	}
	for i := releaseIndex + 1; i < len(instrs); i++ {
		switch instrs[i].Kind {
		case ir.IRJmp:
			return true
		case ir.IRLabel:
			return false
		}
	}
	return false
}

func previousLabelIndex(instrs []ir.IRInstr, before int) int {
	for i := before - 1; i >= 0; i-- {
		if instrs[i].Kind == ir.IRLabel {
			return i
		}
	}
	return -1
}

func catchDispatchCaseLabels(fn ir.IRFunc) []int {
	labels := []int(nil)
	for i := 1; i < len(fn.Instrs); i++ {
		instr := fn.Instrs[i]
		if instr.Kind != ir.IRJmp || fn.Instrs[i-1].Kind != ir.IRJmpIfZero {
			continue
		}
		labels = append(labels, instr.Label)
	}
	return labels
}

func assertOwnedReleaseGuardedByLocal(t *testing.T, fn ir.IRFunc, tagLocal int) {
	t.Helper()
	dropIndex := -1
	releaseIndex := -1
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRDropOwned && dropIndex < 0 {
			dropIndex = i
		}
		if instr.Kind == ir.IRReleaseAllocation && releaseIndex < 0 {
			releaseIndex = i
		}
	}
	if dropIndex < 0 || releaseIndex < 0 {
		t.Fatalf("%s has no owned drop/release: %#v", fn.Name, fn.Instrs)
	}
	guardIndex := -1
	guardLabel := -1
	for i := 0; i+1 < dropIndex; i++ {
		if fn.Instrs[i].Kind == ir.IRLoadLocal &&
			fn.Instrs[i].Local == tagLocal &&
			fn.Instrs[i+1].Kind == ir.IRJmpIfZero {
			guardIndex = i
			guardLabel = fn.Instrs[i+1].Label
			break
		}
	}
	if guardIndex < 0 {
		t.Fatalf("%s owned release is not guarded by local%d tag: %#v", fn.Name, tagLocal, fn.Instrs)
	}
	labelIndex := -1
	for i := releaseIndex + 1; i < len(fn.Instrs); i++ {
		if fn.Instrs[i].Kind == ir.IRLabel && fn.Instrs[i].Label == guardLabel {
			labelIndex = i
			break
		}
	}
	if labelIndex < 0 {
		t.Fatalf(
			"%s owned release guard at %d has no post-release label %d: %#v",
			fn.Name,
			guardIndex,
			guardLabel,
			fn.Instrs,
		)
	}
	if !(guardIndex < dropIndex && dropIndex < releaseIndex && releaseIndex < labelIndex) {
		t.Fatalf(
			"%s owned release guard/drop/release/label order is invalid: guard=%d drop=%d release=%d label=%d instrs=%#v",
			fn.Name,
			guardIndex,
			dropIndex,
			releaseIndex,
			labelIndex,
			fn.Instrs,
		)
	}
}

func assertOwnedReleaseGuardedByLocalEq(t *testing.T, fn ir.IRFunc, tagLocal int, tagValue int32) {
	t.Helper()
	dropIndex := -1
	releaseIndex := -1
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRDropOwned && dropIndex < 0 {
			dropIndex = i
		}
		if instr.Kind == ir.IRReleaseAllocation && releaseIndex < 0 {
			releaseIndex = i
		}
	}
	if dropIndex < 0 || releaseIndex < 0 {
		t.Fatalf("%s has no owned drop/release: %#v", fn.Name, fn.Instrs)
	}
	guardIndex := -1
	guardLabel := -1
	for i := 0; i+3 < dropIndex; i++ {
		if fn.Instrs[i].Kind == ir.IRLoadLocal &&
			fn.Instrs[i].Local == tagLocal &&
			fn.Instrs[i+1].Kind == ir.IRConstI32 &&
			fn.Instrs[i+1].Imm == tagValue &&
			fn.Instrs[i+2].Kind == ir.IRCmpEqI32 &&
			fn.Instrs[i+3].Kind == ir.IRJmpIfZero {
			guardIndex = i
			guardLabel = fn.Instrs[i+3].Label
			break
		}
	}
	if guardIndex < 0 {
		t.Fatalf("%s owned release is not guarded by local%d == %d tag: %#v", fn.Name, tagLocal, tagValue, fn.Instrs)
	}
	labelIndex := -1
	for i := releaseIndex + 1; i < len(fn.Instrs); i++ {
		if fn.Instrs[i].Kind == ir.IRLabel && fn.Instrs[i].Label == guardLabel {
			labelIndex = i
			break
		}
	}
	if labelIndex < 0 {
		t.Fatalf(
			"%s owned release eq guard at %d has no post-release label %d: %#v",
			fn.Name,
			guardIndex,
			guardLabel,
			fn.Instrs,
		)
	}
	if !(guardIndex < dropIndex && dropIndex < releaseIndex && releaseIndex < labelIndex) {
		t.Fatalf(
			"%s owned release eq guard/drop/release/label order is invalid: guard=%d drop=%d release=%d label=%d instrs=%#v",
			fn.Name,
			guardIndex,
			dropIndex,
			releaseIndex,
			labelIndex,
			fn.Instrs,
		)
	}
}

// ---- atomic_builtin_test.go ----

func TestLowerCoreAtomicI32BuiltinsToIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        var ignored_store: i32 = core.atomic_store_i32_release(p, 1, mem)
        let loaded: i32 = core.atomic_load_i32_acquire(p, mem)
        let exchanged: i32 = core.atomic_exchange_i32_seq_cst(p, 2, mem)
        let cas: i32 = core.atomic_compare_exchange_i32_acq_rel(p, loaded, exchanged, mem)
        let add: i32 = core.atomic_fetch_add_i32_relaxed(p, 3, mem)
        let sub: i32 = core.atomic_fetch_sub_i32_seq_cst(p, 1, mem)
        let anded: i32 = core.atomic_fetch_and_i32_acquire(p, 7, mem)
        let ored: i32 = core.atomic_fetch_or_i32_release(p, 8, mem)
        let xored: i32 = core.atomic_fetch_xor_i32_acq_rel(p, 9, mem)
        var ignored_fence: i32 = core.atomic_fence_seq_cst(mem)
        return loaded + exchanged + cas + add + sub + anded + ored + xored
    return 0
`, "main")

	for _, tc := range []struct {
		name string
		kind ir.IRInstrKind
	}{
		{"load", ir.IRAtomicLoadI32},
		{"store", ir.IRAtomicStoreI32},
		{"exchange", ir.IRAtomicExchangeI32},
		{"compare_exchange", ir.IRAtomicCompareExchangeI32},
		{"fetch_add", ir.IRAtomicFetchAddI32},
		{"fetch_sub", ir.IRAtomicFetchSubI32},
		{"fetch_and", ir.IRAtomicFetchAndI32},
		{"fetch_or", ir.IRAtomicFetchOrI32},
		{"fetch_xor", ir.IRAtomicFetchXorI32},
		{"fence", ir.IRAtomicFenceSeqCst},
	} {
		if got := countInstr(fn.Instrs, tc.kind, ""); got != 1 {
			t.Fatalf(
				"atomic %s should lower to one %v, got %d: %#v",
				tc.name,
				tc.kind,
				got,
				fn.Instrs,
			)
		}
	}
}

func TestLowerCoreAtomicSmallAndPointerBuiltinsToIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let byte: u8 = 1
        let word: u16 = 2
        let old_byte: u8 = core.atomic_exchange_u8_seq_cst(p, byte, mem)
        let old_word: u16 = core.atomic_exchange_u16_seq_cst(p, word, mem)
        let loaded: ptr = core.atomic_load_ptr_acquire(p, mem)
        var ignored_store: ptr = core.atomic_store_ptr_release(p, loaded, mem)
        let swapped: ptr = core.atomic_exchange_ptr_seq_cst(p, loaded, mem)
        let cas: ptr = core.atomic_compare_exchange_ptr_acq_rel(p, loaded, swapped, mem)
        let add: ptr = core.atomic_fetch_add_ptr_relaxed(p, loaded, mem)
        let sub: ptr = core.atomic_fetch_sub_ptr_seq_cst(p, loaded, mem)
        let anded: ptr = core.atomic_fetch_and_ptr_acquire(p, loaded, mem)
        let ored: ptr = core.atomic_fetch_or_ptr_release(p, loaded, mem)
        let xored: ptr = core.atomic_fetch_xor_ptr_acq_rel(p, loaded, mem)
        return old_byte + old_word
    return 0
`, "main")

	for _, tc := range []struct {
		name string
		kind ir.IRInstrKind
	}{
		{"u8 exchange", ir.IRAtomicExchangeI8},
		{"u16 exchange", ir.IRAtomicExchangeI16},
		{"ptr load", ir.IRAtomicLoadPtr},
		{"ptr store", ir.IRAtomicStorePtr},
		{"ptr exchange", ir.IRAtomicExchangePtr},
		{"ptr compare_exchange", ir.IRAtomicCompareExchangePtr},
		{"ptr fetch_add", ir.IRAtomicFetchAddPtr},
		{"ptr fetch_sub", ir.IRAtomicFetchSubPtr},
		{"ptr fetch_and", ir.IRAtomicFetchAndPtr},
		{"ptr fetch_or", ir.IRAtomicFetchOrPtr},
		{"ptr fetch_xor", ir.IRAtomicFetchXorPtr},
	} {
		if got := countInstr(fn.Instrs, tc.kind, ""); got != 1 {
			t.Fatalf(
				"atomic %s should lower to one %v, got %d: %#v",
				tc.name,
				tc.kind,
				got,
				fn.Instrs,
			)
		}
	}
}

func TestLowerCoreAtomicI64AndWeakCompareExchangeBuiltinsToIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        let exchanged: i64 = core.atomic_exchange_i64_seq_cst(p, loaded, mem)
        let weak_i64: i64 = core.atomic_compare_exchange_weak_i64_seq_cst(p, loaded, exchanged, mem)
        let weak_i32: i32 = core.atomic_compare_exchange_weak_i32_seq_cst(p, 0, 1, mem)
        var ignored_store: i64 = core.atomic_store_i64_release(p, weak_i64, mem)
        return weak_i32
    return 0
`, "main")

	for _, tc := range []struct {
		name string
		kind ir.IRInstrKind
	}{
		{"i64 load", ir.IRAtomicLoadI64},
		{"i64 exchange", ir.IRAtomicExchangeI64},
		{"i64 weak compare_exchange", ir.IRAtomicCompareExchangeI64},
		{"i64 store", ir.IRAtomicStoreI64},
		{"i32 weak compare_exchange", ir.IRAtomicCompareExchangeI32},
	} {
		if got := countInstr(fn.Instrs, tc.kind, ""); got != 1 {
			t.Fatalf(
				"atomic %s should lower to one %v, got %d: %#v",
				tc.name,
				tc.kind,
				got,
				fn.Instrs,
			)
		}
	}
}

// ---- atomic_test.go ----

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
	if err == nil ||
		!strings.Contains(err.Error(), "unsupported atomic fence memory order unknown") {
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
			t.Fatalf(
				"atomicValueKindForOpWidth(%s, %d) = %v, want %v",
				tc.op,
				tc.widthBits,
				got,
				tc.want,
			)
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

// ---- callable_test.go ----

func TestLowerCallableFunctionValueEmitsSymAddrIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return 0
`, "main")

	if countInstr(fn.Instrs, ir.IRSymAddr, "add1") != 1 {
		t.Fatalf("function-typed binding did not emit one IRSymAddr(add1): %#v", fn.Instrs)
	}
}

func TestLowerCallableAliasCopiesFnptrSlotsIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    let g: fn(Int) -> Int = f
    return 0
`, "main")

	if countInstr(fn.Instrs, ir.IRSymAddr, "add1") != 1 {
		t.Fatalf(
			"function-typed alias should materialize add1 once and copy fnptr slots: %#v",
			fn.Instrs,
		)
	}
	if countCallableKind(fn.Instrs, ir.IRLoadLocal) < semantics.FnPtrSlotCount ||
		countCallableKind(fn.Instrs, ir.IRStoreLocal) < 2*semantics.FnPtrSlotCount {
		t.Fatalf(
			"function-typed alias did not copy the %d-slot fnptr value: %#v",
			semantics.FnPtrSlotCount,
			fn.Instrs,
		)
	}
}

func TestLowerCallableNineCaptureHandleAllocatesAndReadsAllEnvSlotsIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    return cb(-3)
`, "main")

	if got := countInstr(fn.Instrs, ir.IRAllocBytes, ""); got != 1 {
		t.Fatalf("nine-capture callable should allocate one heap env, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRMemWritePtrOffset, ""); got != 9 {
		t.Fatalf("nine-capture callable should write 9 heap env slots, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRMemReadPtrOffset, ""); got != 9 {
		t.Fatalf(
			"nine-capture callable call should read 9 heap env slots, got %d: %#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstr(fn.Instrs, ir.IRPtrAdd, ""); got != 0 {
		t.Fatalf(
			"nine-capture callable heap env should use base+offset access, got %d ptr_add instructions: %#v",
			got,
			fn.Instrs,
		)
	}
	calls := 0
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.ArgSlots == 10 && instr.RetSlots == 1 {
			calls++
		}
	}
	if calls != 1 {
		t.Fatalf(
			"nine-capture callable should call closure with explicit arg plus 9 env slots: %#v",
			fn.Instrs,
		)
	}
}

func TestLowerCallableDirectNamedParamUsesKnownTargetIR(t *testing.T) {
	prog := lowerCallableProgram(t, `
func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(add1, 41)
`)
	apply := requireCallableFunc(t, prog, "apply")
	mainFn := requireCallableFunc(t, prog, "main")

	if countInstr(mainFn.Instrs, ir.IRSymAddr, "add1") != 1 {
		t.Fatalf(
			"direct named callback argument did not lower to IRSymAddr(add1): %#v",
			mainFn.Instrs,
		)
	}
	if countCall(apply.Instrs, "add1", 1, 1) != 1 {
		t.Fatalf(
			"single-target callback body did not lower to direct IRCall(add1): %#v",
			apply.Instrs,
		)
	}
	if countInstr(apply.Instrs, ir.IRSymAddr, "add1") != 0 {
		t.Fatalf(
			"single-target callback body should not emit a dynamic branch IRSymAddr: %#v",
			apply.Instrs,
		)
	}
}

func TestLowerCallableParamCallCoercesOptionalArgumentSlotsIR(t *testing.T) {
	prog := lowerCallableProgram(t, `
func unwrap(value: Int?) -> Int:
    if let some(x) = value:
        return x
    else:
        return 0

func apply(cb: fn(Int?) -> Int) -> Int:
    return cb(41)

func main() -> Int:
    return apply(unwrap)
`)
	apply := requireCallableFunc(t, prog, "apply")

	requireContiguousArgumentLoadsBeforeCall(t, apply.Instrs, "unwrap", 2)
}

func TestLowerStoredCallableCallCoercesOptionalArgumentSlotsIR(t *testing.T) {
	mainFn := requireCallableFunc(t, lowerCallableProgram(t, `
struct Holder:
    cb: fn(Int?) -> Int

func unwrap(value: Int?) -> Int:
    if let some(x) = value:
        return x
    else:
        return 0

func main() -> Int:
    let holder: Holder = Holder(cb: unwrap)
    return holder.cb(41)
`), "main")

	requireContiguousArgumentLoadsBeforeCall(t, mainFn.Instrs, "unwrap", 2)
}

func TestLowerCallableMultiTargetParamBranchesOnSymAddrIR(t *testing.T) {
	apply := lowerCallableFunc(t, `
func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let a: Int = apply(add1, 10)
    let b: Int = apply(add2, 20)
    return a + b
`, "apply")

	if countInstr(apply.Instrs, ir.IRSymAddr, "add1") != 1 ||
		countInstr(apply.Instrs, ir.IRSymAddr, "add2") != 1 {
		t.Fatalf(
			"multi-target callback body did not compare against both target symbols: %#v",
			apply.Instrs,
		)
	}
	if countCallableKind(apply.Instrs, ir.IRCmpEqI32) < 2 ||
		countCallableKind(apply.Instrs, ir.IRJmpIfZero) < 2 {
		t.Fatalf(
			"multi-target callback body lacks symbol compare/branch sequence: %#v",
			apply.Instrs,
		)
	}
	if countCall(apply.Instrs, "add1", 1, 1) != 1 || countCall(apply.Instrs, "add2", 1, 1) != 1 {
		t.Fatalf(
			"multi-target callback body did not lower both direct target calls: %#v",
			apply.Instrs,
		)
	}
}

func TestLowerCallableMutableGlobalAssignmentBranchesOnSymAddrIR(t *testing.T) {
	mainFn := requireCallableFunc(t, lowerCallableFileProgram(t, `
var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    cb = add2
    return cb(40)
`), "main")

	if countInstr(mainFn.Instrs, ir.IRSymAddr, "add1") < 1 ||
		countInstr(mainFn.Instrs, ir.IRSymAddr, "add2") < 2 {
		t.Fatalf(
			"mutable global callable did not preserve both assignment and dispatch targets: %#v",
			mainFn.Instrs,
		)
	}
	if countCallableKind(mainFn.Instrs, ir.IRStoreGlobal) < semantics.FnPtrSlotCount ||
		countCallableKind(mainFn.Instrs, ir.IRLoadGlobal) < semantics.FnPtrSlotCount {
		t.Fatalf(
			"mutable global callable did not store/load %d-slot fnptr value: %#v",
			semantics.FnPtrSlotCount,
			mainFn.Instrs,
		)
	}
	if countCall(mainFn.Instrs, "add1", 1, 1) != 1 || countCall(mainFn.Instrs, "add2", 1, 1) != 1 {
		t.Fatalf("mutable global callable did not lower both branch targets: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableMutableGlobalCallbackArgumentLoadsCurrentGlobalIR(t *testing.T) {
	prog := lowerCallableFileProgram(t, `
var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    cb = add2
    return apply(cb, 40)
`)
	mainFn := requireCallableFunc(t, prog, "main")
	apply := requireCallableFunc(t, prog, "apply")

	if countCallableKind(mainFn.Instrs, ir.IRLoadGlobal) < 4 {
		t.Fatalf(
			"mutable global callback argument did not load current fnptr slots: %#v",
			mainFn.Instrs,
		)
	}
	if countInstr(mainFn.Instrs, ir.IRSymAddr, "add1") > 1 {
		t.Fatalf(
			"mutable global callback argument was rewritten to static initial target: %#v",
			mainFn.Instrs,
		)
	}
	if countCall(apply.Instrs, "add1", 1, 1) != 1 || countCall(apply.Instrs, "add2", 1, 1) != 1 {
		t.Fatalf(
			"callee callback target set did not include both mutable global targets: %#v",
			apply.Instrs,
		)
	}
}

func TestLowerCallableCapturedReturnMutableLocalGlobalAssignmentCopiesFnptrIR(t *testing.T) {
	mainFn := requireCallableFunc(t, lowerCallableFileProgram(t, `
var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func main() -> Int:
    var local: fn(Int) -> Int = identity
    local = make()
    cb = local
    return cb(40)
`), "main")

	if countCallableKind(mainFn.Instrs, ir.IRStoreGlobal) < semantics.FnPtrSlotCount {
		t.Fatalf(
			"global assignment did not store %d fnptr slots: %#v",
			semantics.FnPtrSlotCount,
			mainFn.Instrs,
		)
	}
	if countCallableKind(mainFn.Instrs, ir.IRLoadLocal) < semantics.FnPtrSlotCount {
		t.Fatalf(
			"global assignment did not copy fnptr slots from the mutable local: %#v",
			mainFn.Instrs,
		)
	}
	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf("mutable global dispatch lost captured return target: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedReturnMutableLocalGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var local: fn(Int) -> Int = identity
    local = make()
    cb = local
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			"mutable global dispatch did not receive captured return target from configure assignment: %#v",
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedReturnStructFieldGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var holder: Holder = Holder(cb: identity)
    holder.cb = make()
    cb = holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured struct-field " +
				"target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedStructFieldGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        return x + delta
    )
    cb = holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured struct-field " +
				"direct closure target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedReturnWholeStructGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var holder: Holder = Holder(cb: identity)
    holder = Holder(cb: make())
    cb = holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured whole-struct " +
				"target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedWholeStructGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    var holder: Holder = Holder(cb: identity)
    holder = Holder(cb: fn(x: Int) -> Int:
        return x + delta
    )
    cb = holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured whole-struct " +
				"direct closure target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedReturnWholeNestedStructGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var box: Box = Box(holder: Holder(cb: identity))
    box = Box(holder: Holder(cb: make()))
    cb = box.holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured whole-nested-" +
				"struct target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedWholeNestedStructGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    var box: Box = Box(holder: Holder(cb: identity))
    box = Box(holder: Holder(cb: fn(x: Int) -> Int:
        return x + delta
    ))
    cb = box.holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured whole-nested-" +
				"struct direct closure target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedReturnEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(make())
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured enum-payload " +
				"target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        return x + delta
    )
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured enum-payload " +
				"direct closure target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedWholeEnumGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
	prog := lowerCallableFileProgram(t, `
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(fn(x: Int) -> Int:
        return x + delta
    )
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured whole-enum " +
				"direct closure target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedReturnedStructEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func makeBox() -> Box:
    let delta: Int = 2
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int:
        return x + delta
    ))

func configure() -> Int:
    let box: Box = makeBox()
    match box.choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured returned-" +
				"struct enum-payload target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableCapturedReturnedEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR(
	t *testing.T,
) {
	prog := lowerCallableFileProgram(t, `
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func makeChoice() -> MaybeCallback:
    let delta: Int = 2
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + delta
    )

func configure() -> Int:
    let choice: MaybeCallback = makeChoice()
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`)
	mainFn := requireCallableFunc(t, prog, "main")

	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf(
			("mutable global dispatch did not receive captured returned-enum " +
				"payload target from configure assignment: %#v"),
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableMultiTargetStringReturnSlotsIR(t *testing.T) {
	apply := lowerCallableFunc(t, `
func word1(x: Int) -> String:
    return "cat"

func word2(x: Int) -> String:
    return "zebra"

func apply(cb: fn(Int) -> String, x: Int) -> String:
    return cb(x)

func main() -> Int:
    let a: String = apply(word1, 0)
    let b: String = apply(word2, 0)
    return a.len + b.len
`, "apply")

	if apply.ReturnSlots != 2 {
		t.Fatalf("string-return callback apply ReturnSlots = %d, want 2", apply.ReturnSlots)
	}
	if countCall(apply.Instrs, "word1", 1, 2) != 1 || countCall(apply.Instrs, "word2", 1, 2) != 1 {
		t.Fatalf(
			"string-return callback branches did not preserve two return slots: %#v",
			apply.Instrs,
		)
	}
	if countStoresAfterCalls(apply.Instrs, map[string]bool{"word1": true, "word2": true}) < 4 {
		t.Fatalf(
			"string-return callback branches did not store two result slots per target: %#v",
			apply.Instrs,
		)
	}
}

func TestLowerCallableMultiTargetStructReturnSlotsIR(t *testing.T) {
	apply := lowerCallableFunc(t, `
struct Pair:
    x: Int
    y: Int

func pair1(x: Int) -> Pair:
    return Pair(x: x, y: 1)

func pair2(x: Int) -> Pair:
    return Pair(x: x, y: 2)

func apply(cb: fn(Int) -> Pair, x: Int) -> Pair:
    return cb(x)

func main() -> Int:
    let a: Pair = apply(pair1, 10)
    let b: Pair = apply(pair2, 20)
    return a.x + a.y + b.x + b.y
`, "apply")

	if apply.ReturnSlots != 2 {
		t.Fatalf("struct-return callback apply ReturnSlots = %d, want 2", apply.ReturnSlots)
	}
	if countCall(apply.Instrs, "pair1", 1, 2) != 1 || countCall(apply.Instrs, "pair2", 1, 2) != 1 {
		t.Fatalf(
			"struct-return callback branches did not preserve two return slots: %#v",
			apply.Instrs,
		)
	}
	if countStoresAfterCalls(apply.Instrs, map[string]bool{"pair1": true, "pair2": true}) < 4 {
		t.Fatalf(
			"struct-return callback branches did not store two result slots per target: %#v",
			apply.Instrs,
		)
	}
}

func TestLowerCallableWholeStructReassignmentFromReturnPropagatesFieldTargetIR(t *testing.T) {
	mainFn := lowerCallableFunc(t, `
struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func pick() -> fn(Int) -> Int:
    return add2

func main() -> Int:
    var holder: Holder = Holder(cb: add1)
    holder = Holder(cb: pick())
    return holder.cb(40)
`, "main")

	if countCall(mainFn.Instrs, "add1", 1, 1) != 1 || countCall(mainFn.Instrs, "add2", 1, 1) != 1 {
		t.Fatalf(
			"whole-struct reassignment did not preserve both field-call targets: %#v",
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableStructValuedFieldReassignmentFromReturnPropagatesFieldTargetIR(t *testing.T) {
	mainFn := lowerCallableFunc(t, `
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func pick() -> fn(Int) -> Int:
    return add2

func main() -> Int:
    var box: Box = Box(holder: Holder(cb: add1))
    box.holder = Holder(cb: pick())
    return box.holder.cb(40)
`, "main")

	if countCall(mainFn.Instrs, "add1", 1, 1) != 1 || countCall(mainFn.Instrs, "add2", 1, 1) != 1 {
		t.Fatalf(
			"struct-valued field reassignment did not preserve both nested field-call targets: %#v",
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableWholeNestedStructReassignmentFromReturnPropagatesFieldTargetIR(t *testing.T) {
	mainFn := lowerCallableFunc(t, `
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func pick() -> fn(Int) -> Int:
    return add2

func main() -> Int:
    var box: Box = Box(holder: Holder(cb: add1))
    box = Box(holder: Holder(cb: pick()))
    return box.holder.cb(40)
`, "main")

	if countCall(mainFn.Instrs, "add1", 1, 1) != 1 || countCall(mainFn.Instrs, "add2", 1, 1) != 1 {
		t.Fatalf(
			"whole nested-struct reassignment did not preserve both nested field-call targets: %#v",
			mainFn.Instrs,
		)
	}
}

func TestLowerCallableUnknownTargetDiagnostic(t *testing.T) {
	checked := checkCallableProgram(t, `
func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return 0
`)
	var apply semantics.CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "apply" {
			apply = fn
			break
		}
	}
	if apply.Name == "" {
		t.Fatalf("apply function not found")
	}

	_, err := lowerCheckedFunc(
		apply,
		checked.Types,
		checked.FuncSigs,
		checked.GlobalsByModule[apply.Module],
		typedTaskStagedTarget{},
		map[string][]string{"cb": {"missing_callback_target"}},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "unknown callback target 'missing_callback_target'") {
		t.Fatalf("error = %v, want unknown callback target diagnostic", err)
	}
}

func lowerCallableFunc(t *testing.T, src string, name string) ir.IRFunc {
	t.Helper()
	return requireCallableFunc(t, lowerCallableProgram(t, src), name)
}

func lowerCallableProgram(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	checked := checkCallableProgram(t, src)
	prog, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	return prog
}

func checkCallableProgram(t *testing.T, src string) *semantics.CheckedProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return checked
}

func lowerCallableFileProgram(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: file.Module,
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{file.Module: file},
	}
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: true})
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	prog, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	return prog
}

func requireCallableFunc(t *testing.T, prog *ir.IRProgram, name string) ir.IRFunc {
	t.Helper()
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("function %q not found", name)
	return ir.IRFunc{}
}

func countInstr(instrs []ir.IRInstr, kind ir.IRInstrKind, name string) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind != kind {
			continue
		}
		if name != "" && instr.Name != name {
			continue
		}
		count++
	}
	return count
}

func countCallableKind(instrs []ir.IRInstr, kind ir.IRInstrKind) int {
	return countInstr(instrs, kind, "")
}

func countCall(instrs []ir.IRInstr, name string, argSlots int, retSlots int) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind == ir.IRCall && instr.Name == name && instr.ArgSlots == argSlots &&
			instr.RetSlots == retSlots {
			count++
		}
	}
	return count
}

func requireContiguousArgumentLoadsBeforeCall(
	t *testing.T,
	instrs []ir.IRInstr,
	name string,
	argSlots int,
) {
	t.Helper()
	for i, instr := range instrs {
		if instr.Kind != ir.IRCall || instr.Name != name || instr.ArgSlots != argSlots {
			continue
		}
		if i < argSlots {
			t.Fatalf("call %s lacks %d preceding argument loads: %#v", name, argSlots, instrs)
		}
		base := -1
		for slot := 0; slot < argSlots; slot++ {
			load := instrs[i-argSlots+slot]
			if load.Kind != ir.IRLoadLocal {
				t.Fatalf(
					"call %s arg %d is loaded by %v, want IRLoadLocal: %#v",
					name,
					slot+1,
					load.Kind,
					instrs,
				)
			}
			if slot == 0 {
				base = load.Local
				continue
			}
			if load.Local != base+slot {
				t.Fatalf(
					"call %s arg loads are locals %d then %d, want contiguous scratch locals: %#v",
					name,
					base,
					load.Local,
					instrs,
				)
			}
		}
		return
	}
	t.Fatalf("call %s with %d arg slots not found: %#v", name, argSlots, instrs)
}

func countCallableClosureCalls(instrs []ir.IRInstr) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind == ir.IRCall && strings.Contains(instr.Name, "closure") {
			count++
		}
	}
	return count
}

func countStoresAfterCalls(instrs []ir.IRInstr, names map[string]bool) int {
	count := 0
	for i, instr := range instrs {
		if instr.Kind != ir.IRCall || !names[instr.Name] {
			continue
		}
		for j := i + 1; j < len(instrs) && instrs[j].Kind == ir.IRStoreLocal; j++ {
			count++
		}
	}
	return count
}

// ---- catch_typed_task_test.go ----

func TestLowerCatchHandlerCollectsStagedTypedTaskWrapper(t *testing.T) {
	prog := lowerProgramForCatchTest(t, `
enum OuterErr:
    case nope

enum TaskErr:
    case boom(Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    return 11

func fail() -> Int throws OuterErr:
    throw OuterErr.nope

func main() -> Int
uses runtime:
    return catch fail():
    case OuterErr.nope:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.boom(left, right):
            left + right
        case TaskErr.stopped:
            0
`)

	wrapperName := typedTaskWrapperName("worker", "TaskErr")
	if !programHasFunc(prog, wrapperName) {
		t.Fatalf(
			"typed task wrapper %q was not collected from catch handler; funcs=%v",
			wrapperName,
			programFuncNames(prog),
		)
	}

	mainFn := requireIRFunc(t, prog, "main")
	if !hasCall(mainFn.Instrs, "__tetra_task_join_typed_5", 1) {
		t.Fatalf("main IR lacks staged typed-task join call: %#v", mainFn.Instrs)
	}
	if countCallsNamed(mainFn.Instrs, "__tetra_task_result_get") < 4 {
		t.Fatalf("main IR lacks staged result-slot loads: %#v", mainFn.Instrs)
	}
	if !hasInstructionPair(mainFn.Instrs, ir.IRCmpEqI32, ir.IRJmpIfZero) {
		t.Fatalf("main IR lacks catch enum compare/branch checks: %#v", mainFn.Instrs)
	}
}

func TestLowerMatchExprCollectsStagedTypedTaskWrapper(t *testing.T) {
	prog := lowerProgramForCatchTest(t, `
enum Choice:
    case left
    case right

enum TaskErr:
    case boom(Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    return 13

func main() -> Int
uses runtime:
    let choice: Choice = Choice.left
    return match choice:
    case Choice.left:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.boom(left, right):
            left + right
        case TaskErr.stopped:
            0
    case Choice.right:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.boom(otherLeft, otherRight):
            otherLeft + otherRight
        case TaskErr.stopped:
            0
`)

	wrapperName := typedTaskWrapperName("worker", "TaskErr")
	if !programHasFunc(prog, wrapperName) {
		t.Fatalf(
			"typed task wrapper %q was not collected from match expression; funcs=%v",
			wrapperName,
			programFuncNames(prog),
		)
	}
}

func TestLowerTryTypedTaskJoinUsesStagedResultSlots(t *testing.T) {
	prog := lowerProgramForCatchTest(t, `
enum TaskErr:
    case boom(Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    return 17

func caller() -> Int throws TaskErr
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`)

	wrapperName := typedTaskWrapperName("worker", "TaskErr")
	if !programHasFunc(prog, wrapperName) {
		t.Fatalf(
			"typed task wrapper %q was not collected for try join; funcs=%v",
			wrapperName,
			programFuncNames(prog),
		)
	}

	callerFn := requireIRFunc(t, prog, "caller")
	if !hasCall(callerFn.Instrs, "__tetra_task_join_typed_5", 1) {
		t.Fatalf("caller IR lacks staged typed-task try join call: %#v", callerFn.Instrs)
	}
	if countCallsNamed(callerFn.Instrs, "__tetra_task_result_get") < 4 {
		t.Fatalf("caller IR lacks staged typed-task result loads: %#v", callerFn.Instrs)
	}
	if countKind(callerFn.Instrs, ir.IRReturn) < 1 {
		t.Fatalf("caller IR lacks propagation return path: %#v", callerFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsBeforeTryTypedTaskJoinPropagationReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum TaskErr:
    case boom(Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2)

func caller() -> Int throws TaskErr
uses runtime, alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        let task = core.task_spawn_i32_typed<TaskErr>("worker")
        return try core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`)

	callerFn := requireIRFunc(t, prog, "caller")
	if !hasCall(callerFn.Instrs, "__tetra_task_join_typed_5", 1) {
		t.Fatalf("caller IR lacks staged typed-task try join call: %#v", callerFn.Instrs)
	}
	if countInstrKind(callerFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("caller alloc_bytes count = %d, want 1: %#v", countInstrKind(callerFn, ir.IRAllocBytes), callerFn.Instrs)
	}
	if countInstrKind(callerFn, ir.IRDropOwned) != 2 || countInstrKind(callerFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("typed task try propagation and success exits must both drop owned allocation: %#v", callerFn.Instrs)
	}
}

func TestLowerOwnedAllocBytesDropsBeforeCompactPayloadTypedTaskJoinPropagationReturn(t *testing.T) {
	prog := lowerOwnedAllocDropProgram(t, `
enum TaskErr:
    case code(Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.code(7)

func caller() -> Int throws TaskErr
uses runtime, alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(16)
        let task = core.task_spawn_i32_typed<TaskErr>("worker")
        return try core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`)

	callerFn := requireIRFunc(t, prog, "caller")
	if !hasCall(callerFn.Instrs, "__tetra_task_join_typed_4", 4) {
		t.Fatalf("caller IR lacks compact payload typed-task try join call: %#v", callerFn.Instrs)
	}
	if countInstrKind(callerFn, ir.IRAllocBytes) != 1 {
		t.Fatalf("caller alloc_bytes count = %d, want 1: %#v", countInstrKind(callerFn, ir.IRAllocBytes), callerFn.Instrs)
	}
	if countInstrKind(callerFn, ir.IRDropOwned) != 2 || countInstrKind(callerFn, ir.IRReleaseAllocation) != 2 {
		t.Fatalf("compact typed task try propagation and success exits must both drop owned allocation: %#v", callerFn.Instrs)
	}
}

func TestLowerStagedTypedTaskPolicyFailureStagesStatus(t *testing.T) {
	prog := lowerProgramForCatchTest(t, `
enum TaskErr:
    case boom(Int, Int)
    case stopped

func worker() -> Int throws TaskErr
uses budget
budget(4):
    return 17

func main() -> Int
uses runtime, budget
budget(8):
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(left, right):
        left + right
    case TaskErr.stopped:
        0
`)

	workerFn := requireIRFunc(t, prog, "worker")
	if !workerFn.Policy.HasBudget || workerFn.Policy.FailLabel < 0 {
		t.Fatalf("worker IR lacks budget policy failure metadata: %#v", workerFn.Policy)
	}
	if countCallsNamed(workerFn.Instrs, "__tetra_task_result_begin") < 2 {
		t.Fatalf(
			"worker IR did not stage both normal and policy-failure typed task results: %#v",
			workerFn.Instrs,
		)
	}
}

func lowerProgramForCatchTest(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	return irProg
}

func requireIRFunc(t *testing.T, prog *ir.IRProgram, name string) ir.IRFunc {
	t.Helper()
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("function %q not found; funcs=%v", name, programFuncNames(prog))
	return ir.IRFunc{}
}

func programHasFunc(prog *ir.IRProgram, name string) bool {
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return true
		}
	}
	return false
}

func programFuncNames(prog *ir.IRProgram) []string {
	if prog == nil {
		return nil
	}
	names := make([]string, 0, len(prog.Funcs))
	for _, fn := range prog.Funcs {
		names = append(names, fn.Name)
	}
	return names
}

func hasCall(instrs []ir.IRInstr, name string, retSlots int) bool {
	for _, instr := range instrs {
		if instr.Kind == ir.IRCall && instr.Name == name && instr.RetSlots == retSlots {
			return true
		}
	}
	return false
}

func countCallsNamed(instrs []ir.IRInstr, name string) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			count++
		}
	}
	return count
}

// ---- distributed_actor_runtime_test.go ----

func TestLowerDistributedActorRuntimeBuiltins(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let _connected: Int = core.actor_node_connect(2, 5010)
    let peer: actor = core.spawn_remote(2, "worker")
    let _status: Int = core.actor_node_status(2)
    return core.send(peer, 7)
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	mainFn := findIRFuncByName(t, irProg.Funcs, "main")
	for _, name := range []string{
		"__tetra_actor_node_connect",
		"__tetra_actor_spawn_remote",
		"__tetra_actor_node_status",
		"__tetra_actor_send",
	} {
		if !hasIRCallName(mainFn, name) {
			t.Fatalf("main is missing %s call: %#v", name, mainFn.Instrs)
		}
	}
}

func TestLowerActorRefInternalBuiltins(t *testing.T) {
	src := []byte(`
func main() -> Int
uses actors:
    unsafe:
        let peer: actor = core.actor_ref_local(7, 1)
        return core.actor_ref_slot(peer)
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFuncByName(t, irProg.Funcs, "main")
	for _, instr := range mainFn.Instrs {
		if instr.Kind == ir.IRCall {
			t.Fatalf("actor_ref helpers should lower without runtime calls, got %#v", instr)
		}
	}
	hasStoreLocal := false
	for _, instr := range mainFn.Instrs {
		if instr.Kind == ir.IRStoreLocal {
			hasStoreLocal = true
			break
		}
	}
	if !hasStoreLocal {
		t.Fatalf("actor_ref_slot should discard the high actor slot: %#v", mainFn.Instrs)
	}
}

func TestLowerActorLifecycleRuntimeBuiltins(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _status: actor.status = core.actor_status(peer)
    let _raw_status: actor.status_result_raw = core.actor_status_raw(peer)
    let _waited: actor.wait_result = core.actor_wait(peer)
    let _waited_until: actor.wait_result = core.actor_wait_until(peer, 10)
    let reason: actor.exit_reason = core.actor_exit_reason(peer)
    let monitor: actor.monitor = core.actor_monitor(peer)
    let linked: Int = core.actor_link(peer)
    let unlinked: Int = core.actor_unlink(peer)
    let demonitored: Int = core.actor_demonitor(monitor)
    let stopped: Int = core.actor_stop(peer, reason)
    let trapped: Int = core.actor_set_trap_exit(1)
    return linked + unlinked + demonitored + stopped + trapped
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	mainFn := findIRFuncByName(t, irProg.Funcs, "main")
	actorSlots := runtimeabi.ActorHandleABI().RefSlots
	tests := []struct {
		name string
		args int
		rets int
	}{
		{name: "__tetra_actor_status", args: actorSlots, rets: 1},
		{name: "__tetra_actor_status_raw", args: actorSlots, rets: 2},
		{name: "__tetra_actor_wait", args: actorSlots, rets: 2},
		{name: "__tetra_actor_wait_until", args: actorSlots + 1, rets: 2},
		{name: "__tetra_actor_exit_reason", args: actorSlots, rets: 1},
		{name: "__tetra_actor_monitor", args: actorSlots, rets: 1},
		{name: "__tetra_actor_link", args: actorSlots, rets: 1},
		{name: "__tetra_actor_unlink", args: actorSlots, rets: 1},
		{name: "__tetra_actor_demonitor", args: 1, rets: 1},
		{name: "__tetra_actor_stop", args: actorSlots + 1, rets: 1},
		{name: "__tetra_actor_set_trap_exit", args: 1, rets: 1},
	}
	for _, tt := range tests {
		if countCall(mainFn.Instrs, tt.name, tt.args, tt.rets) != 1 {
			t.Fatalf("main missing call %s/%d/%d: %#v", tt.name, tt.args, tt.rets, mainFn.Instrs)
		}
	}
}

func TestLowerActorSystemReceiveRuntimeBuiltins(t *testing.T) {
	src := []byte(`
func blocking() -> actor.system_recv_raw
uses actors, runtime:
    return core.actor_recv_system()

func poll() -> actor.system_recv_raw
uses actors:
    return core.actor_recv_system_poll()

func timed() -> actor.system_recv_raw
uses actors, runtime:
    return core.actor_recv_system_until(10)

func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	allInstrs := append(
		append(
			findIRFuncByName(t, irProg.Funcs, "blocking").Instrs,
			findIRFuncByName(t, irProg.Funcs, "poll").Instrs...,
		),
		findIRFuncByName(t, irProg.Funcs, "timed").Instrs...,
	)
	if got := countCall(allInstrs, "__tetra_actor_recv_system_begin", 2, 1); got != 3 {
		t.Fatalf("system receive begin calls = %d, want 3: %#v", got, allInstrs)
	}
	rawSlots := runtimeabi.ActorHandleABI().RefSlots + 7
	if got, want := countCall(allInstrs, "__tetra_actor_recv_system_slot", 1, 1), 3*(rawSlots-1); got != want {
		t.Fatalf("system receive slot calls = %d, want %d: %#v", got, want, allInstrs)
	}
	if got := countCall(allInstrs, "__tetra_actor_recv_system_count", 0, 1); got != 3 {
		t.Fatalf("system receive count calls = %d, want 3: %#v", got, allInstrs)
	}
	requireConstPairBeforeCall(t, allInstrs, "__tetra_actor_recv_system_begin", 0, 0)
	requireConstPairBeforeCall(t, allInstrs, "__tetra_actor_recv_system_begin", 1, 0)
	requireConstPairBeforeCall(t, allInstrs, "__tetra_actor_recv_system_begin", 2, 10)
}

func requireConstPairBeforeCall(
	t *testing.T,
	instrs []ir.IRInstr,
	name string,
	first int32,
	second int32,
) {
	t.Helper()
	for i, instr := range instrs {
		if instr.Kind != ir.IRCall || instr.Name != name {
			continue
		}
		if i < 2 {
			continue
		}
		a := instrs[i-2]
		b := instrs[i-1]
		if a.Kind == ir.IRConstI32 && a.Imm == first &&
			b.Kind == ir.IRConstI32 && b.Imm == second {
			return
		}
	}
	t.Fatalf("missing const pair (%d,%d) before %s call: %#v", first, second, name, instrs)
}

// ---- enum_payload_test.go ----

func TestLowerEnumPayloadConstructorLayoutIR(t *testing.T) {
	fn := lowerMainForEnumPayloadTest(t, `
enum Pair:
    case both(Int, String)
    case empty

func main() -> Int:
    let pair: Pair = Pair.both(7, "xy")
    return 0
`)

	if !hasEnumConstructorSequence(fn.Instrs, []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRConstI32, Imm: 7},
		{Kind: ir.IRStrLit},
	}, 4) {
		t.Fatalf("constructor IR does not preserve tag + payload slot order: %#v", fn.Instrs)
	}
}

func TestLowerEnumPayloadConstructorZeroPadsWideNoPayloadCaseIR(t *testing.T) {
	fn := lowerMainForEnumPayloadTest(t, `
enum Wide:
    case data(Int, String)
    case empty

func main() -> Int:
    let wide: Wide = Wide.empty
    return 0
`)

	if !hasEnumConstructorSequence(fn.Instrs, []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRConstI32, Imm: 0},
	}, 4) {
		t.Fatalf("no-payload constructor IR does not preserve tag + zero padding: %#v", fn.Instrs)
	}
}

func TestLowerEnumPayloadSlotOrderInMatchIR(t *testing.T) {
	fn := lowerMainForEnumPayloadTest(t, `
enum Pair:
    case both(Int, String)

func main() -> Int:
    let pair: Pair = Pair.both(5, "xy")
    match pair:
    case Pair.both(code, text):
        return code
`)

	firstBindingLoad, secondBindingLoad := findFirstTwoPayloadBindingLoads(t, fn.Instrs)
	if secondBindingLoad.Local != firstBindingLoad.Local+1 {
		t.Fatalf(
			"payload binding loads not contiguous/in declaration order: first=%#v second=%#v",
			firstBindingLoad,
			secondBindingLoad,
		)
	}
}

func TestLowerMatchExpressionEnumPayloadIR(t *testing.T) {
	fn := lowerMainForEnumPayloadTest(t, `
enum Result:
    case ok(Int)
    case err(Int)

func main() -> Int:
    let result: Result = Result.ok(42)
    let score: Int = match result:
    case Result.ok(value):
        value
    case Result.err(code):
        code
    return score
`)

	if !hasInstructionPair(fn.Instrs, ir.IRCmpEqI32, ir.IRJmpIfZero) {
		t.Fatalf("match expression IR lacks compare/branch discriminator checks: %#v", fn.Instrs)
	}
	if countKind(fn.Instrs, ir.IRLabel) < 3 {
		t.Fatalf("match expression IR label count too low: %#v", fn.Instrs)
	}
}

func TestLowerIfLetEnumPayloadPatternIR(t *testing.T) {
	fn := lowerMainForEnumPayloadTest(t, `
enum Result:
    case ok(Int, String)
    case err(Int)

func main() -> Int:
    let result: Result = Result.ok(40, "xy")
    if let Result.ok(code, text) = result:
        return code + text.len
    else:
        return 0
`)

	if !hasInstructionPair(fn.Instrs, ir.IRCmpEqI32, ir.IRJmpIfZero) {
		t.Fatalf("if-let enum pattern IR lacks compare/branch discriminator checks: %#v", fn.Instrs)
	}
	firstBindingLoad, secondBindingLoad := findFirstTwoPayloadBindingLoads(t, fn.Instrs)
	if secondBindingLoad.Local != firstBindingLoad.Local+1 {
		t.Fatalf(
			"if-let payload binding loads not contiguous/in declaration order: first=%#v second=%#v",
			firstBindingLoad,
			secondBindingLoad,
		)
	}
}

func lowerMainForEnumPayloadTest(t *testing.T, src string) ir.IRFunc {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	for _, fn := range irProg.Funcs {
		if fn.Name == "main" {
			return fn
		}
	}
	t.Fatalf("main function not found")
	return ir.IRFunc{}
}

func hasEnumConstructorSequence(instrs []ir.IRInstr, values []ir.IRInstr, storeSlots int) bool {
	for i := 0; i+len(values)+storeSlots <= len(instrs); i++ {
		matched := true
		for j, want := range values {
			got := instrs[i+j]
			if got.Kind != want.Kind {
				matched = false
				break
			}
			if want.Kind == ir.IRConstI32 && got.Imm != want.Imm {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		for j := 0; j < storeSlots; j++ {
			if instrs[i+len(values)+j].Kind != ir.IRStoreLocal {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func findFirstTwoPayloadBindingLoads(t *testing.T, instrs []ir.IRInstr) (ir.IRInstr, ir.IRInstr) {
	t.Helper()
	for i := 0; i+3 < len(instrs); i++ {
		if instrs[i].Kind == ir.IRLoadLocal && instrs[i+1].Kind == ir.IRStoreLocal &&
			instrs[i+2].Kind == ir.IRLoadLocal &&
			instrs[i+3].Kind == ir.IRLoadLocal {
			return instrs[i], instrs[i+2]
		}
	}
	t.Fatalf("payload binding load sequence not found: %#v", instrs)
	return ir.IRInstr{}, ir.IRInstr{}
}

func hasInstructionPair(instrs []ir.IRInstr, first, second ir.IRInstrKind) bool {
	for i := 0; i+1 < len(instrs); i++ {
		if instrs[i].Kind == first && instrs[i+1].Kind == second {
			return true
		}
	}
	return false
}

func countKind(instrs []ir.IRInstr, kind ir.IRInstrKind) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind == kind {
			count++
		}
	}
	return count
}

// ---- filesystem_test.go ----

func TestLowerFilesystemExistsBuiltinUsesRuntimeCall(t *testing.T) {
	prog := lowerCallableProgram(t, `
func probe(cap: cap.io) -> Bool
uses io:
    return core.fs_exists("README.md", cap)

func main() -> Int:
    return 0
`)
	probe := requireCallableFunc(t, prog, "probe")
	if countCall(probe.Instrs, "__tetra_fs_exists", 3, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.fs_exists to __tetra_fs_exists(3 -> 1): %#v",
			probe.Instrs,
		)
	}
}

// ---- global_assignment_test.go ----

func TestLowerGlobalStructFieldAssignmentStoresGlobalSlot(t *testing.T) {
	checked, mainFn := lowerGlobalAssignmentProgram(t, `
struct Box:
    first: Int
    value: Int

var box: Box

func main() -> Int:
    var first: Int = 11
    var second: Int = 22
    box.value = 42
    return first + second + box.value
`)

	box := checked.GlobalsByModule[""]["box"]
	valueSlot := box.DataIndex + 1
	if !hasConstStore(mainFn.Instrs, ir.IRStoreGlobal, valueSlot, 42) {
		t.Fatalf(
			"global field assignment did not store 42 into global slot %d: %#v",
			valueSlot,
			mainFn.Instrs,
		)
	}
	if hasConstStore(mainFn.Instrs, ir.IRStoreLocal, 1, 42) {
		t.Fatalf("global field assignment still stores 42 into local slot 1: %#v", mainFn.Instrs)
	}
}

func TestLowerGlobalStructFieldAssignmentWithoutLocalsVerifies(t *testing.T) {
	_, _ = lowerGlobalAssignmentProgram(t, `
struct Box:
    value: Int

var box: Box

func main() -> Int:
    box.value = 42
    return box.value
`)
}

func lowerGlobalAssignmentProgram(t *testing.T, src string) (*semantics.CheckedProgram, ir.IRFunc) {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "global_assignment.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	checked, err := semantics.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	for _, fn := range irProg.Funcs {
		if fn.Name == "main" {
			return checked, fn
		}
	}
	t.Fatalf("main function not found")
	return nil, ir.IRFunc{}
}

func hasConstStore(instrs []ir.IRInstr, kind ir.IRInstrKind, slot int, value int32) bool {
	for i := 0; i+1 < len(instrs); i++ {
		if instrs[i].Kind == ir.IRConstI32 && instrs[i].Imm == value &&
			instrs[i+1].Kind == kind && instrs[i+1].Local == slot {
			return true
		}
	}
	return false
}

// ---- net_test.go ----

func TestLowerNetBuiltinsUseRuntimeCalls(t *testing.T) {
	prog := lowerCallableProgram(t, `
func probe(cap: cap.io) -> Int
uses alloc, io, mem:
    let fd: Int = core.net_socket_tcp4(cap)
    let bind_status: Int = core.net_bind_tcp4_loopback(fd, 18080, cap)
    let connect_status: Int = core.net_connect_tcp4_loopback(fd, 18080, cap)
    let listen_status: Int = core.net_listen(fd, 8, cap)
    let client: Int = core.net_accept4(fd, 0, cap)
    var buf: []u8 = core.make_u8(8)
    let read_status: Int = core.net_read(client, buf, 0, 8, cap)
    let recv_status: Int = core.net_recv(client, buf, 0, 8, cap)
    let write_status: Int = core.net_write(client, buf, 0, 2, cap)
    let send_status: Int = core.net_send(client, buf, 0, 2, cap)
    let epfd: Int = core.net_epoll_create(cap)
    let epoll_add: Int = core.net_epoll_ctl_add_read(epfd, fd, cap)
    let epoll_add_rw: Int = core.net_epoll_ctl_add_read_write(epfd, fd, cap)
    let epoll_mod_read: Int = core.net_epoll_ctl_mod_read(epfd, fd, cap)
    let epoll_mod_rw: Int = core.net_epoll_ctl_mod_read_write(epfd, fd, cap)
    let epoll_delete: Int = core.net_epoll_ctl_delete(epfd, fd, cap)
    let epoll_ready: Int = core.net_epoll_wait_one(epfd, 0, cap)
    var event: []i32 = core.make_i32(2)
    let epoll_event_ready: Int = core.net_epoll_wait_one_into(epfd, event, 0, cap)
    let nb: Int = core.net_set_nonblocking(fd, cap)
    let reuse: Int = core.net_set_reuseport(fd, cap)
    let nodelay: Int = core.net_set_tcp_nodelay(fd, cap)
    let closed: Int = core.net_close(fd, cap)
    return fd + bind_status + connect_status + listen_status + client + read_status + recv_status + write_status + send_status + epfd + epoll_add + epoll_add_rw + epoll_mod_read + epoll_mod_rw + epoll_delete + epoll_ready + epoll_event_ready + nb + reuse + nodelay + closed

func main() -> Int:
    return 0
`)
	probe := requireCallableFunc(t, prog, "probe")
	if countCall(probe.Instrs, "__tetra_net_socket_tcp4", 1, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_socket_tcp4 to __tetra_net_socket_tcp4(1 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_bind_tcp4_loopback", 3, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_bind_tcp4_loopback to __tetra_net_bind_tcp4_loopback(3 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_connect_tcp4_loopback", 3, 1) != 1 {
		t.Fatalf(
			("probe did not lower core.net_connect_tcp4_loopback to __tetra_" +
				"net_connect_tcp4_loopback(3 -> 1): %#v"),
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_listen", 3, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_listen to __tetra_net_listen(3 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_accept4", 3, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_accept4 to __tetra_net_accept4(3 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_read", 6, 1) != 1 {
		t.Fatalf("probe did not lower core.net_read to __tetra_net_read(6 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_recv", 6, 1) != 1 {
		t.Fatalf("probe did not lower core.net_recv to __tetra_net_recv(6 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_write", 6, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_write to __tetra_net_write(6 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_send", 6, 1) != 1 {
		t.Fatalf("probe did not lower core.net_send to __tetra_net_send(6 -> 1): %#v", probe.Instrs)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_create", 1, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_epoll_create to __tetra_net_epoll_create(1 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_ctl_add_read", 3, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_epoll_ctl_add_read to __tetra_net_epoll_ctl_add_read(3 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_ctl_add_read_write", 3, 1) != 1 {
		t.Fatalf(
			("probe did not lower core.net_epoll_ctl_add_read_write to __" +
				"tetra_net_epoll_ctl_add_read_write(3 -> 1): %#v"),
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_ctl_mod_read", 3, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_epoll_ctl_mod_read to __tetra_net_epoll_ctl_mod_read(3 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_ctl_mod_read_write", 3, 1) != 1 {
		t.Fatalf(
			("probe did not lower core.net_epoll_ctl_mod_read_write to __" +
				"tetra_net_epoll_ctl_mod_read_write(3 -> 1): %#v"),
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_ctl_delete", 3, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_epoll_ctl_delete to __tetra_net_epoll_ctl_delete(3 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_wait_one", 3, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_epoll_wait_one to __tetra_net_epoll_wait_one(3 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_epoll_wait_one_into", 5, 1) != 1 {
		t.Fatalf(
			("probe did not lower core.net_epoll_wait_one_into to __tetra_net_" +
				"epoll_wait_one_into(5 -> 1): %#v"),
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_set_nonblocking", 2, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_set_nonblocking to __tetra_net_set_nonblocking(2 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_set_reuseport", 2, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_set_reuseport to __tetra_net_set_reuseport(2 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_set_tcp_nodelay", 2, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_set_tcp_nodelay to __tetra_net_set_tcp_nodelay(2 -> 1): %#v",
			probe.Instrs,
		)
	}
	if countCall(probe.Instrs, "__tetra_net_close", 2, 1) != 1 {
		t.Fatalf(
			"probe did not lower core.net_close to __tetra_net_close(2 -> 1): %#v",
			probe.Instrs,
		)
	}
}

// ---- privacy_lowering_test.go ----

func TestLowerPrivacySealUnsealI32DeterministicShapeAndNoSideEffects(t *testing.T) {
	src := []byte(`
func seal(token: consent.token, value: Int) -> secret.i32
uses privacy
privacy
consent(token):
    return core.secret_seal_i32(value, token)

func unseal(token: consent.token, value: secret.i32) -> Int
uses privacy
privacy
consent(token):
    return core.secret_unseal_i32(value, token)

func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	seal := findLoweredFuncByName(t, irProg, "seal")
	unseal := findLoweredFuncByName(t, irProg, "unseal")
	wantPattern := []ir.IRInstrKind{ir.IRConstI32, ir.IRMulI32, ir.IRAddI32}

	if got := countKindPattern(seal.Instrs, wantPattern); got != 1 {
		t.Fatalf("seal lowering pattern count = %d, want 1; instrs=%#v", got, seal.Instrs)
	}
	if got := countKindPattern(unseal.Instrs, wantPattern); got != 1 {
		t.Fatalf("unseal lowering pattern count = %d, want 1; instrs=%#v", got, unseal.Instrs)
	}

	for _, fn := range []ir.IRFunc{seal, unseal} {
		assertNoPrivacySideEffects(t, fn)
	}
}

func TestLowerConsentTokenUsesOpaqueRuntimeSentinel(t *testing.T) {
	src := []byte(`
func require_token(token: consent.token) -> Int
uses privacy
privacy
consent(token):
    return 7

func main() -> Int
uses privacy
privacy:
    let token: consent.token = core.consent_token()
    return require_token(token)
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	requireToken := findLoweredFuncByName(t, irProg, "require_token")
	tokenSentinel := assertExactConsentGuard(t, requireToken)
	if tokenSentinel == 0 || tokenSentinel == 1 {
		t.Fatalf("consent token sentinel = %d, want opaque non-zero/non-one value", tokenSentinel)
	}

	mainFn := findLoweredFuncByName(t, irProg, "main")
	if !containsConstI32(mainFn.Instrs, tokenSentinel) {
		t.Fatalf(
			"main does not mint the consent sentinel %d; instrs=%#v",
			tokenSentinel,
			mainFn.Instrs,
		)
	}
	if containsConstI32(mainFn.Instrs, 1) {
		t.Fatalf(
			"main still appears to mint forgeable consent token constant 1; instrs=%#v",
			mainFn.Instrs,
		)
	}
}

func findLoweredFuncByName(t *testing.T, prog *ir.IRProgram, name string) ir.IRFunc {
	t.Helper()
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("lowered function %q not found", name)
	return ir.IRFunc{}
}

func countKindPattern(instrs []ir.IRInstr, pattern []ir.IRInstrKind) int {
	if len(pattern) == 0 {
		return 0
	}
	count := 0
	for i := 0; i+len(pattern) <= len(instrs); i++ {
		ok := true
		for j := range pattern {
			if instrs[i+j].Kind != pattern[j] {
				ok = false
				break
			}
		}
		if ok {
			count++
		}
	}
	return count
}

func assertExactConsentGuard(t *testing.T, fn ir.IRFunc) int32 {
	t.Helper()
	for i := 0; i+3 < len(fn.Instrs); i++ {
		if fn.Instrs[i].Kind == ir.IRLoadLocal &&
			fn.Instrs[i+1].Kind == ir.IRConstI32 &&
			fn.Instrs[i+2].Kind == ir.IRCmpEqI32 &&
			fn.Instrs[i+3].Kind == ir.IRJmpIfZero {
			return fn.Instrs[i+1].Imm
		}
	}
	t.Fatalf("%s missing exact consent guard; instrs=%#v", fn.Name, fn.Instrs)
	return 0
}

func containsConstI32(instrs []ir.IRInstr, imm int32) bool {
	for _, instr := range instrs {
		if instr.Kind == ir.IRConstI32 && instr.Imm == imm {
			return true
		}
	}
	return false
}

func assertNoPrivacySideEffects(t *testing.T, fn ir.IRFunc) {
	t.Helper()
	disallowed := map[ir.IRInstrKind]string{
		ir.IRCall:              "runtime call",
		ir.IRWrite:             "stdout write",
		ir.IRStoreGlobal:       "global storage write",
		ir.IRMemWriteI32:       "memory write i32",
		ir.IRMemWriteU8:        "memory write u8",
		ir.IRMemWritePtr:       "memory write ptr",
		ir.IRMemWriteI32Offset: "memory write i32 offset",
		ir.IRMemWriteU8Offset:  "memory write u8 offset",
		ir.IRMemWritePtrOffset: "memory write ptr offset",
		ir.IRMmioWriteI32:      "mmio write",
		ir.IRCtxSwitch:         "context switch",
	}
	for _, instr := range fn.Instrs {
		if reason, bad := disallowed[instr.Kind]; bad {
			t.Fatalf("%s contains disallowed %s instruction: %#v", fn.Name, reason, instr)
		}
	}
}

// ---- proof_bce_test.go ----

func lowerProofProgram(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		t.Fatalf("PLIR verify: %v", err)
	}
	out, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	return out
}

func lowerProofFileProgram(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "p25/hash_table.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: file.Module,
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{file.Module: file},
	}
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: true})
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		t.Fatalf("PLIR verify: %v", err)
	}
	out, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	return out
}

func proofPLIRFileProgram(t *testing.T, src string) *plir.Program {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "p25/allocation.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: file.Module,
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{file.Module: file},
	}
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: true})
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		t.Fatalf("PLIR verify: %v", err)
	}
	return proofProg
}

func TestForSliceLoopUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    for x in xs:
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("sum still contains checked i32 index load: %#v", fn.Instrs)
	}
	var proofID string
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRIndexLoadI32Unchecked {
			proofID = instr.ProofID
			break
		}
	}
	if proofID == "" {
		t.Fatalf("sum missing proof-tagged unchecked i32 index load: %#v", fn.Instrs)
	}
}

func TestExternalIndexKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func get(xs: []i32, i: Int) -> Int
uses mem:
    return xs[i]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return get(xs, 0)
`)
	fn := findIRFunc(t, prog, "get")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("get should keep one checked i32 index load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("get unexpectedly contains unchecked i32 index load: %#v", fn.Instrs)
	}
}

func TestWhileLessThanLenUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("sum still contains checked i32 index load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"while unchecked load proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestWhileCompoundIncrementUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i += 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf(
			"compound increment while loop still contains checked i32 index load: %#v",
			fn.Instrs,
		)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"compound increment unchecked load proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestWhileCommutedIncrementUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = 1 + i
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf(
			"commuted increment while loop still contains checked i32 index load: %#v",
			fn.Instrs,
		)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"commuted increment unchecked load proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestWhileConstStepOneIncrementUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    let step: Int = 1
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + step
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("const step-one while loop still contains checked i32 index load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"const step-one unchecked load proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestWhileNonUnitIncrementKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + 2
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("non-unit increment should keep one checked load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("non-unit increment unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestWhileAliasLessThanLenUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    let ys: []i32 = xs
    var total = 0
    var i = 0
    while i < ys.len:
        total = total + ys[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("alias while loop still contains checked i32 index load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"alias while unchecked load proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestWhileLessEqualLenMinusOneUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i <= xs.len - 1:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(0)
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("sum still contains checked i32 index load for <= len - 1 proof: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"while <= unchecked load proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestWhileNotEqualLenUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i != xs.len:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("while != len loop still contains checked i32 index load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"while != len unchecked load proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestWhileStartEndAliasesUseProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    let start: Int = 0
    let end: Int = xs.len
    var total = 0
    var i = start
    while i < end:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("start/end alias while loop still contains checked i32 index load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"start/end alias unchecked load proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestWhileViewEndAliasUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    let view: []i32 = xs.prefix(2)
    let end: Int = view.len
    var total = 0
    var i = 0
    while i < end:
        total = total + view[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("view end alias while loop still contains checked i32 index load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"view end alias unchecked load proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestWhileUnsafeEndAliasKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    let end: Int = xs.len + 1
    var total = 0
    var i = 0
    while i < end:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("unsafe end alias should keep one checked load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("unsafe end alias unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestWhileBaseReassignmentInvalidatesRangeProof(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum_reassign(xs: []i32, ys: []i32) -> Int
uses mem:
    var view: []i32 = xs
    var total = 0
    var i = 0
    while i < view.len:
        view = ys
        total = total + view[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    var ys: []i32 = make_i32(1)
    xs[0] = 1
    ys[0] = 2
    return sum_reassign(xs, ys)
`)
	fn := findIRFunc(t, prog, "sum_reassign")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("base reassignment should keep checked i32 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("base reassignment unexpectedly kept stale unchecked load: %#v", fn.Instrs)
	}
}

func TestWhileInoutCallInvalidatesRangeProof(t *testing.T) {
	prog := lowerProofProgram(t, `
func touch(view: inout []i32) -> Int
uses mem:
    return view.len

func sum_inout(view: inout []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < view.len:
        touch(view)
        total = total + view[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return sum_inout(xs)
`)
	fn := findIRFunc(t, prog, "sum_inout")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("inout call should keep checked i32 load after mutation boundary: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("inout call unexpectedly kept stale unchecked load: %#v", fn.Instrs)
	}
}

func TestWhileCallbackInoutCallInvalidatesRangeProof(t *testing.T) {
	prog := lowerProofProgram(t, `
func touch(view: inout []i32) -> Int
uses mem:
    return view.len

func sum_callback(view: inout []i32, cb: fn(inout []i32) -> Int uses mem) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < view.len:
        cb(view)
        total = total + view[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return sum_callback(xs, touch)
`)
	fn := findIRFunc(t, prog, "sum_callback")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf(
			"callback inout call should keep checked i32 load after unknown mutable boundary: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("callback inout call unexpectedly kept stale unchecked load: %#v", fn.Instrs)
	}
}

func TestWhileMissingGuardKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func get(xs: []i32, n: Int) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < n:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return get(xs, 1)
`)
	fn := findIRFunc(t, prog, "get")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("missing len guard should keep one checked load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("missing len guard unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestWhileAllocationLengthAliasUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 4
    var xs: []i32 = make_i32(n)
    var i = 0
    while i < n:
        xs[i] = i
        i = i + 1
    var total = 0
    i = 0
    while i < n:
        total = total + xs[i]
        i = i + 1
    return total
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("allocation length alias loop still contains checked i32 load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"allocation length alias proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("allocation length alias should keep one store instruction: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexStoreI32); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf(
			"allocation length alias store proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestP50HashTableLookupUsesCallBoundaryLengthProof(t *testing.T) {
	prog := lowerProofFileProgram(t, `
module p25.hash_table

func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        keys[i] = i * 2 + 1
        values[i] = i + 7
        i = i + 1
    var checksum: Int = 0
    var q: Int = 0
    while q < n:
        let key: Int = q * 2 + 1
        checksum = checksum + lookup(keys, values, n, key)
        q = q + 1
    if checksum > 0:
        return 0
    return 1
`)
	lookup := findIRFunc(t, prog, "p25.hash_table.lookup")
	if countInstrKind(lookup, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("lookup should remove both call-boundary-proven checked loads: %#v", lookup.Instrs)
	}
	if countInstrKind(lookup, ir.IRIndexLoadI32Unchecked) != 2 {
		t.Fatalf("lookup should contain two proof-tagged unchecked loads: %#v", lookup.Instrs)
	}
	proofIDs := proofIDsForKind(lookup, ir.IRIndexLoadI32Unchecked)
	if !containsProofIDPrefix(proofIDs, "proof:call-boundary:i:keys:") {
		t.Fatalf(
			"lookup keys load missing call-boundary proof in %#v; instrs=%#v",
			proofIDs,
			lookup.Instrs,
		)
	}
	if !containsProofIDPrefix(proofIDs, "proof:call-boundary:i:values:") {
		t.Fatalf(
			"lookup values load missing call-boundary proof in %#v; instrs=%#v",
			proofIDs,
			lookup.Instrs,
		)
	}

	mainFn := findIRFunc(t, prog, "p25.hash_table.main")
	if countInstrKind(mainFn, ir.IRIndexStoreI32) != 2 {
		t.Fatalf("hash_table main should have two i32 stores: %#v", mainFn.Instrs)
	}
	proofs := proofIDsForKind(mainFn, ir.IRIndexStoreI32)
	if len(proofs) != 2 {
		t.Fatalf(
			"hash_table main stores should both have proofs, got %#v; instrs=%#v",
			proofs,
			mainFn.Instrs,
		)
	}
	if !containsProofIDPrefix(proofs, "proof:while-const:i:keys:") {
		t.Fatalf(
			"hash_table main keys store missing local allocation proof in %#v; instrs=%#v",
			proofs,
			mainFn.Instrs,
		)
	}
	if !containsProofIDPrefix(proofs, "proof:while:i:values:") {
		t.Fatalf(
			"hash_table main values store proof regressed in %#v; instrs=%#v",
			proofs,
			mainFn.Instrs,
		)
	}
}

func TestP50HashTableLookupKeepsCheckedLoadsWhenNMayExceedKeysLen(t *testing.T) {
	prog := lowerProofFileProgram(t, `
module p25.hash_table

func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    let short: Int = 128
    var keys: []i32 = core.make_i32(short)
    var values: []i32 = core.make_i32(n)
    return lookup(keys, values, n, 7)
`)
	lookup := findIRFunc(t, prog, "p25.hash_table.lookup")
	if countInstrKind(lookup, ir.IRIndexLoadI32) != 2 {
		t.Fatalf(
			"lookup should keep checked loads when keys.len may be shorter than n: %#v",
			lookup.Instrs,
		)
	}
	if countInstrKind(lookup, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("lookup unexpectedly removed loads for unsafe keys boundary: %#v", lookup.Instrs)
	}
}

func TestP50HashTableLookupKeepsCheckedLoadsWhenValuesLenMayBeShorter(t *testing.T) {
	prog := lowerProofFileProgram(t, `
module p25.hash_table

func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    let short: Int = 128
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(short)
    return lookup(keys, values, n, 7)
`)
	lookup := findIRFunc(t, prog, "p25.hash_table.lookup")
	if countInstrKind(lookup, ir.IRIndexLoadI32) != 2 {
		t.Fatalf(
			"lookup should keep checked loads when values.len may be shorter than n: %#v",
			lookup.Instrs,
		)
	}
	if countInstrKind(lookup, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("lookup unexpectedly removed loads for unsafe values boundary: %#v", lookup.Instrs)
	}
}

func TestP50UnrelatedPublicHelperDoesNotInheritLookupCallBoundaryProof(t *testing.T) {
	prog := lowerProofFileProgram(t, `
module p25.hash_table

pub func probe(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(n)
    return probe(keys, values, n, 7)
`)
	probe := findIRFunc(t, prog, "p25.hash_table.probe")
	if countInstrKind(probe, ir.IRIndexLoadI32) != 2 {
		t.Fatalf("unrelated helper should keep checked loads: %#v", probe.Instrs)
	}
	if countInstrKind(probe, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf(
			"unrelated helper unexpectedly inherited lookup call-boundary proof: %#v",
			probe.Instrs,
		)
	}
}

func TestSliceSumShapeProofTagsStoreAndPreservesLoadBCE(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    var r: Int = 0
    while r < 64:
        i = 0
        while i < n:
            total = total + xs[i]
            i = i + 1
        r = r + 1
    if total > 0:
        return 0
    return 1
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("slice_sum shape should have one i32 store instruction: %#v", fn.Instrs)
	}
	storeProofID := firstProofID(fn, ir.IRIndexStoreI32)
	if !strings.HasPrefix(storeProofID, "proof:while:") {
		t.Fatalf(
			"slice_sum store proof id = %q, want proof:while prefix; instrs=%#v",
			storeProofID,
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("slice_sum load-side BCE regressed to checked load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 1 {
		t.Fatalf("slice_sum shape should have one proof-tagged unchecked i32 load: %#v", fn.Instrs)
	}
	loadProofID := firstProofID(fn, ir.IRIndexLoadI32Unchecked)
	if !strings.HasPrefix(loadProofID, "proof:while:") {
		t.Fatalf(
			"slice_sum load proof id = %q, want proof:while prefix; instrs=%#v",
			loadProofID,
			fn.Instrs,
		)
	}
	if storeProofID == loadProofID {
		t.Fatalf(
			"slice_sum store/load proofs should come from distinct dominating while guards, both were %q",
			storeProofID,
		)
	}
}

func TestBoundsCheckLoopsModuloAllocationLengthAliasUsesProofTaggedUncheckedLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    i = 0
    while i < 200000:
        let idx: Int = (i * 17) % n
        total = total + xs[idx]
        i = i + 1
    if total >= 0:
        return 0
    return 1
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("bounds_check_loops shape should have one i32 store instruction: %#v", fn.Instrs)
	}
	storeProofID := firstProofID(fn, ir.IRIndexStoreI32)
	if !strings.HasPrefix(storeProofID, "proof:while:") {
		t.Fatalf(
			"bounds_check_loops store proof id = %q, want proof:while prefix; instrs=%#v",
			storeProofID,
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("bounds_check_loops modulo load should not keep checked i32 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 1 {
		t.Fatalf(
			"bounds_check_loops modulo load should be one proof-tagged unchecked i32 load: %#v",
			fn.Instrs,
		)
	}
	loadProofID := firstProofID(fn, ir.IRIndexLoadI32Unchecked)
	if !strings.HasPrefix(loadProofID, "proof:modulo:") {
		t.Fatalf(
			"bounds_check_loops modulo load proof id = %q, want proof:modulo prefix; instrs=%#v",
			loadProofID,
			fn.Instrs,
		)
	}
}

func TestMatrixModuloConstInlineUsesProofTaggedUncheckedLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    var c: []i32 = core.make_i32(9)
    var r: Int = 0
    var checksum: Int = 0
    while r < 2000:
        checksum = checksum + c[r % 9]
        r = r + 1
    return checksum
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("matrix modulo const load should not keep checked i32 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 1 {
		t.Fatalf(
			"matrix modulo const load should be one proof-tagged unchecked i32 load: %#v",
			fn.Instrs,
		)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:modulo:",
	) {
		t.Fatalf(
			"matrix modulo const proof id = %q, want proof:modulo prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestMatrixModuloConstImmutableDivisorLocalUsesProofTaggedUncheckedLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    let d: Int = 9
    var c: []i32 = core.make_i32(9)
    var r: Int = 0
    var checksum: Int = 0
    while r < 2000:
        checksum = checksum + c[r % d]
        r = r + 1
    return checksum
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf(
			"matrix modulo immutable divisor load should not keep checked i32 load: %#v",
			fn.Instrs,
		)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:modulo:",
	) {
		t.Fatalf(
			"matrix modulo immutable divisor proof id = %q, want proof:modulo prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestMatrixConstLoopSetupStoresUseProofTaggedStores(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    var b: []i32 = core.make_i32(9)
    var c: []i32 = core.make_i32(9)
    var i: Int = 0
    while i < 9:
        a[i] = i + 1
        b[i] = 9 - i
        c[i] = 0
        i = i + 1
    return 0
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexStoreI32) != 3 {
		t.Fatalf("setup loop should have three i32 stores: %#v", fn.Instrs)
	}
	proofs := proofIDsForKind(fn, ir.IRIndexStoreI32)
	if len(proofs) != 3 {
		t.Fatalf(
			"setup loop should have three store proofs, got %#v; instrs=%#v",
			proofs,
			fn.Instrs,
		)
	}
	seen := map[string]bool{}
	for _, proofID := range proofs {
		if !strings.HasPrefix(proofID, "proof:while-const:i:") {
			t.Fatalf(
				"setup store proof id = %q, want proof:while-const:i prefix; instrs=%#v",
				proofID,
				fn.Instrs,
			)
		}
		if seen[proofID] {
			t.Fatalf(
				"setup stores should have base-specific proof ids, saw duplicate %q in %#v",
				proofID,
				proofs,
			)
		}
		seen[proofID] = true
	}
	for _, base := range []string{"a", "b", "c"} {
		found := false
		for proofID := range seen {
			if strings.Contains(proofID, ":"+base+":") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("setup stores missing proof for base %q in %#v", base, proofs)
		}
	}
}

func TestMatrixConstLoopUpperLargerThanAllocationKeepsCheckedStore(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    var i: Int = 0
    while i < 10:
        a[i] = i
        i = i + 1
    return 0
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("upper-larger loop should keep one i32 store: %#v", fn.Instrs)
	}
	if countProofTaggedInstrKind(fn, ir.IRIndexStoreI32) != 0 {
		t.Fatalf("upper-larger loop unexpectedly proof-tagged store: %#v", fn.Instrs)
	}
}

func TestMatrixConstLoopMutableAllocationLengthKeepsCheckedStore(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    var n: Int = 9
    var a: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < 9:
        a[i] = i
        i = i + 1
    return 0
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("mutable allocation length loop should keep one i32 store: %#v", fn.Instrs)
	}
	if countProofTaggedInstrKind(fn, ir.IRIndexStoreI32) != 0 {
		t.Fatalf("mutable allocation length unexpectedly proof-tagged store: %#v", fn.Instrs)
	}
}

func TestMatrixConstLoopNonUnitIncrementKeepsCheckedStore(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    var i: Int = 0
    while i < 9:
        a[i] = i
        i = i + 2
    return 0
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("non-unit const loop should keep one i32 store: %#v", fn.Instrs)
	}
	if countProofTaggedInstrKind(fn, ir.IRIndexStoreI32) != 0 {
		t.Fatalf("non-unit const loop unexpectedly proof-tagged store: %#v", fn.Instrs)
	}
}

func TestMatrixAffineConstStoreAndLoadsUseProofTags(t *testing.T) {
	prog := lowerProofProgram(t, matrixAffineLoadProgram(
		"var a: []i32 = core.make_i32(9)",
		"var c: []i32 = core.make_i32(9)",
		"row < 3",
		"k < 3",
		"row * 3 + k",
		"col < 3",
		"row * 3 + col",
		"row = row + 1",
		"k = k + 1",
		"col = col + 1",
		"",
	))
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("matrix affine loads should both be proof-tagged unchecked: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 2 {
		t.Fatalf(
			"matrix affine a/b loads should be two proof-tagged unchecked i32 loads: %#v",
			fn.Instrs,
		)
	}
	loadProofIDs := proofIDsForKind(fn, ir.IRIndexLoadI32Unchecked)
	if !containsProofIDPrefix(loadProofIDs, "proof:affine-const:row_k:a:") {
		t.Fatalf(
			"matrix affine a load proof ids = %#v, want base-specific row_k/a proof; instrs=%#v",
			loadProofIDs,
			fn.Instrs,
		)
	}
	if !containsProofIDPrefix(loadProofIDs, "proof:affine-const:k_col:b:") {
		t.Fatalf(
			"matrix affine b load proof ids = %#v, want base-specific k_col/b proof; instrs=%#v",
			loadProofIDs,
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("matrix affine should have one i32 store: %#v", fn.Instrs)
	}
	proofID := firstProofID(fn, ir.IRIndexStoreI32)
	if !strings.HasPrefix(proofID, "proof:affine-const:") {
		t.Fatalf(
			"matrix affine store proof id = %q, want proof:affine-const prefix; instrs=%#v",
			proofID,
			fn.Instrs,
		)
	}
	if !strings.Contains(proofID, ":c:") {
		t.Fatalf(
			"matrix affine store proof id = %q, want base-specific c proof; instrs=%#v",
			proofID,
			fn.Instrs,
		)
	}
}

func TestMatrixAffineConstInvalidALoadShapesKeepCheckedLoad(t *testing.T) {
	tests := []struct {
		name               string
		aDecl              string
		rowGuard           string
		kGuard             string
		aLoadIndex         string
		rowInc             string
		kInc               string
		beforeLoad         string
		wantCheckedLoads   int
		wantUncheckedLoads int
	}{
		{
			name:               "wrong_stride",
			aDecl:              "var a: []i32 = core.make_i32(9)",
			rowGuard:           "row < 3",
			kGuard:             "k < 3",
			aLoadIndex:         "row * 4 + k",
			rowInc:             "row = row + 1",
			kInc:               "k = k + 1",
			wantCheckedLoads:   1,
			wantUncheckedLoads: 1,
		},
		{
			name:               "mutable_allocation_length",
			aDecl:              "var n: Int = 9\n    var a: []i32 = core.make_i32(n)",
			rowGuard:           "row < 3",
			kGuard:             "k < 3",
			aLoadIndex:         "row * 3 + k",
			rowInc:             "row = row + 1",
			kInc:               "k = k + 1",
			wantCheckedLoads:   1,
			wantUncheckedLoads: 1,
		},
		{
			name:               "non_unit_k_increment",
			aDecl:              "var a: []i32 = core.make_i32(9)",
			rowGuard:           "row < 3",
			kGuard:             "k < 3",
			aLoadIndex:         "row * 3 + k",
			rowInc:             "row = row + 1",
			kInc:               "k = k + 2",
			wantCheckedLoads:   2,
			wantUncheckedLoads: 0,
		},
		{
			name:               "non_strict_k_guard",
			aDecl:              "var a: []i32 = core.make_i32(9)",
			rowGuard:           "row < 3",
			kGuard:             "k <= 2",
			aLoadIndex:         "row * 3 + k",
			rowInc:             "row = row + 1",
			kInc:               "k = k + 1",
			wantCheckedLoads:   2,
			wantUncheckedLoads: 0,
		},
		{
			name:               "base_reassignment_before_load",
			aDecl:              "var a: []i32 = core.make_i32(9)",
			rowGuard:           "row < 3",
			kGuard:             "k < 3",
			aLoadIndex:         "row * 3 + k",
			rowInc:             "row = row + 1",
			kInc:               "k = k + 1",
			beforeLoad:         "a = core.make_i32(9)",
			wantCheckedLoads:   1,
			wantUncheckedLoads: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := lowerProofProgram(t, matrixAffineLoadProgram(
				tt.aDecl,
				"var c: []i32 = core.make_i32(9)",
				tt.rowGuard,
				tt.kGuard,
				tt.aLoadIndex,
				"col < 3",
				"row * 3 + col",
				tt.rowInc,
				tt.kInc,
				"col = col + 1",
				tt.beforeLoad,
			))
			fn := findIRFunc(t, prog, "main")
			if countInstrKind(fn, ir.IRIndexLoadI32) != tt.wantCheckedLoads {
				t.Fatalf(
					"%s: checked matrix load count = %d, want %d: %#v",
					tt.name,
					countInstrKind(fn, ir.IRIndexLoadI32),
					tt.wantCheckedLoads,
					fn.Instrs,
				)
			}
			if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != tt.wantUncheckedLoads {
				t.Fatalf(
					"%s: unchecked matrix load count = %d, want %d: %#v",
					tt.name,
					countInstrKind(fn, ir.IRIndexLoadI32Unchecked),
					tt.wantUncheckedLoads,
					fn.Instrs,
				)
			}
			if containsProofIDPrefix(
				proofIDsForKind(fn, ir.IRIndexLoadI32Unchecked),
				"proof:affine-const:row_k:a:",
			) {
				t.Fatalf(
					"%s: invalid a load unexpectedly received row_k/a proof: %#v",
					tt.name,
					fn.Instrs,
				)
			}
			storeProofID := firstProofID(fn, ir.IRIndexStoreI32)
			if !strings.HasPrefix(storeProofID, "proof:affine-const:row_col:c:") {
				t.Fatalf(
					"%s: P38 c store proof should remain intact, got %q; instrs=%#v",
					tt.name,
					storeProofID,
					fn.Instrs,
				)
			}
		})
	}
}

func TestMatrixAffineConstInvalidBLoadShapesKeepCheckedLoad(t *testing.T) {
	tests := []struct {
		name               string
		bDecl              string
		colGuard           string
		kGuard             string
		bLoadIndex         string
		kInc               string
		colInc             string
		beforeLoad         string
		wantCheckedLoads   int
		wantUncheckedLoads int
	}{
		{
			name:               "wrong_stride",
			bDecl:              "var b: []i32 = core.make_i32(9)",
			colGuard:           "col < 3",
			kGuard:             "k < 3",
			bLoadIndex:         "k * 4 + col",
			kInc:               "k = k + 1",
			colInc:             "col = col + 1",
			wantCheckedLoads:   1,
			wantUncheckedLoads: 1,
		},
		{
			name:               "mutable_allocation_length",
			bDecl:              "var n: Int = 9\n    var b: []i32 = core.make_i32(n)",
			colGuard:           "col < 3",
			kGuard:             "k < 3",
			bLoadIndex:         "k * 3 + col",
			kInc:               "k = k + 1",
			colInc:             "col = col + 1",
			wantCheckedLoads:   1,
			wantUncheckedLoads: 1,
		},
		{
			name:               "non_unit_k_increment",
			bDecl:              "var b: []i32 = core.make_i32(9)",
			colGuard:           "col < 3",
			kGuard:             "k < 3",
			bLoadIndex:         "k * 3 + col",
			kInc:               "k = k + 2",
			colInc:             "col = col + 1",
			wantCheckedLoads:   2,
			wantUncheckedLoads: 0,
		},
		{
			name:               "non_unit_col_increment",
			bDecl:              "var b: []i32 = core.make_i32(9)",
			colGuard:           "col < 3",
			kGuard:             "k < 3",
			bLoadIndex:         "k * 3 + col",
			kInc:               "k = k + 1",
			colInc:             "col = col + 2",
			wantCheckedLoads:   1,
			wantUncheckedLoads: 1,
		},
		{
			name:               "non_strict_col_guard",
			bDecl:              "var b: []i32 = core.make_i32(9)",
			colGuard:           "col <= 2",
			kGuard:             "k < 3",
			bLoadIndex:         "k * 3 + col",
			kInc:               "k = k + 1",
			colInc:             "col = col + 1",
			wantCheckedLoads:   1,
			wantUncheckedLoads: 1,
		},
		{
			name:               "base_reassignment_before_load",
			bDecl:              "var b: []i32 = core.make_i32(9)",
			colGuard:           "col < 3",
			kGuard:             "k < 3",
			bLoadIndex:         "k * 3 + col",
			kInc:               "k = k + 1",
			colInc:             "col = col + 1",
			beforeLoad:         "b = core.make_i32(9)",
			wantCheckedLoads:   1,
			wantUncheckedLoads: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := lowerProofProgram(t, matrixAffineBLoadProgram(
				tt.bDecl,
				"row < 3",
				tt.colGuard,
				tt.kGuard,
				tt.bLoadIndex,
				"row = row + 1",
				tt.kInc,
				tt.colInc,
				tt.beforeLoad,
			))
			fn := findIRFunc(t, prog, "main")
			if countInstrKind(fn, ir.IRIndexLoadI32) != tt.wantCheckedLoads {
				t.Fatalf(
					"%s: checked matrix load count = %d, want %d: %#v",
					tt.name,
					countInstrKind(fn, ir.IRIndexLoadI32),
					tt.wantCheckedLoads,
					fn.Instrs,
				)
			}
			if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != tt.wantUncheckedLoads {
				t.Fatalf(
					"%s: unchecked matrix load count = %d, want %d: %#v",
					tt.name,
					countInstrKind(fn, ir.IRIndexLoadI32Unchecked),
					tt.wantUncheckedLoads,
					fn.Instrs,
				)
			}
			if containsProofIDPrefix(
				proofIDsForKind(fn, ir.IRIndexLoadI32Unchecked),
				"proof:affine-const:k_col:b:",
			) {
				t.Fatalf(
					"%s: invalid b load unexpectedly received k_col/b proof: %#v",
					tt.name,
					fn.Instrs,
				)
			}
		})
	}
}

func TestMatrixAffineConstInvalidShapesKeepCheckedStore(t *testing.T) {
	tests := []struct {
		name        string
		cDecl       string
		rowGuard    string
		colGuard    string
		storeIndex  string
		rowInc      string
		colInc      string
		beforeStore string
	}{
		{
			name:       "wrong_stride",
			cDecl:      "var c: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			colGuard:   "col < 3",
			storeIndex: "row * 4 + col",
			rowInc:     "row = row + 1",
			colInc:     "col = col + 1",
		},
		{
			name:       "mutable_allocation_length",
			cDecl:      "var n: Int = 9\n    var c: []i32 = core.make_i32(n)",
			rowGuard:   "row < 3",
			colGuard:   "col < 3",
			storeIndex: "row * 3 + col",
			rowInc:     "row = row + 1",
			colInc:     "col = col + 1",
		},
		{
			name:       "non_unit_row_increment",
			cDecl:      "var c: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			colGuard:   "col < 3",
			storeIndex: "row * 3 + col",
			rowInc:     "row = row + 2",
			colInc:     "col = col + 1",
		},
		{
			name:       "non_unit_col_increment",
			cDecl:      "var c: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			colGuard:   "col < 3",
			storeIndex: "row * 3 + col",
			rowInc:     "row = row + 1",
			colInc:     "col = col + 2",
		},
		{
			name:       "non_strict_col_guard",
			cDecl:      "var c: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			colGuard:   "col <= 2",
			storeIndex: "row * 3 + col",
			rowInc:     "row = row + 1",
			colInc:     "col = col + 1",
		},
		{
			name:        "base_reassignment_before_store",
			cDecl:       "var c: []i32 = core.make_i32(9)",
			rowGuard:    "row < 3",
			colGuard:    "col < 3",
			storeIndex:  "row * 3 + col",
			rowInc:      "row = row + 1",
			colInc:      "col = col + 1",
			beforeStore: "c = core.make_i32(9)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := lowerProofProgram(
				t,
				matrixAffineStoreProgram(
					tt.cDecl,
					tt.rowGuard,
					tt.colGuard,
					tt.storeIndex,
					tt.rowInc,
					tt.colInc,
					tt.beforeStore,
				),
			)
			fn := findIRFunc(t, prog, "main")
			if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
				t.Fatalf("%s: matrix affine should have one i32 store: %#v", tt.name, fn.Instrs)
			}
			if countProofTaggedInstrKind(fn, ir.IRIndexStoreI32) != 0 {
				t.Fatalf(
					"%s: invalid affine shape unexpectedly proof-tagged store: %#v",
					tt.name,
					fn.Instrs,
				)
			}
		})
	}
}

func TestMatrixAffineIndexStillKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func pick(row: Int, col: Int) -> Int
uses alloc, mem:
    var c: []i32 = core.make_i32(9)
    return c[row * 3 + col]

func main() -> Int
uses alloc, mem:
    return pick(0, 0)
`)
	fn := findIRFunc(t, prog, "pick")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("matrix affine index should keep one checked i32 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("matrix affine index unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestMatrixModuloConstDivisorTooLargeKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    var c: []i32 = core.make_i32(9)
    var r: Int = 0
    var checksum: Int = 0
    while r < 2000:
        checksum = checksum + c[r % 10]
        r = r + 1
    return checksum
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf(
			"modulo const divisor larger than length should keep one checked i32 load: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf(
			"modulo const divisor larger than length unexpectedly removed bounds check: %#v",
			fn.Instrs,
		)
	}
}

func TestMatrixModuloConstMutableLengthKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    var n: Int = 9
    var c: []i32 = core.make_i32(n)
    var r: Int = 0
    var checksum: Int = 0
    while r < 2000:
        checksum = checksum + c[r % 9]
        r = r + 1
    return checksum
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("mutable allocation length should keep one checked i32 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf(
			"mutable allocation length unexpectedly removed modulo const bounds check: %#v",
			fn.Instrs,
		)
	}
}

func TestMatrixModuloConstUnprovenNumeratorKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func pick(r: Int) -> Int
uses alloc, mem:
    var c: []i32 = core.make_i32(9)
    return c[r % 9]

func main() -> Int
uses alloc, mem:
    return pick(0)
`)
	fn := findIRFunc(t, prog, "pick")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("unproven modulo numerator should keep one checked i32 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("unproven modulo numerator unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestModuloMutableAllocationLengthAliasKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    var n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    i = 0
    while i < 8:
        let idx: Int = (i * 17) % n
        total = total + xs[idx]
        i = i + 1
    return total
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("mutable modulo divisor should keep one checked i32 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("mutable modulo divisor unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestModuloWrongDivisorKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    let m: Int = 1024
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    i = 0
    while i < 8:
        let idx: Int = (i * 17) % m
        total = total + xs[idx]
        i = i + 1
    return total
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("wrong modulo divisor should keep one checked i32 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("wrong modulo divisor unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestModuloBaseReassignmentBeforeLoadKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    let m: Int = 2
    var xs: []i32 = core.make_i32(n)
    var ys: []i32 = core.make_i32(m)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    i = 0
    while i < 8:
        let idx: Int = (i * 17) % n
        xs = ys
        total = total + xs[idx]
        i = i + 1
    return total
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf(
			"base reassignment before modulo load should keep one checked i32 load: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf(
			"base reassignment before modulo load unexpectedly removed bounds check: %#v",
			fn.Instrs,
		)
	}
}

func TestModuloInoutMutationBeforeLoadKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func touch(view: inout []i32) -> Int
uses mem:
    return view.len

func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    i = 0
    while i < 8:
        let idx: Int = (i * 17) % n
        touch(xs)
        total = total + xs[idx]
        i = i + 1
    return total
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf(
			"inout mutation before modulo load should keep one checked i32 load: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf(
			"inout mutation before modulo load unexpectedly removed bounds check: %#v",
			fn.Instrs,
		)
	}
}

func TestWhileMutableAllocationLengthAliasKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    var n: Int = 4
    var xs: []i32 = make_i32(n)
    var total = 0
    var i = 0
    while i < n:
        total = total + xs[i]
        i = i + 1
    return total
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("mutable allocation length alias should keep one checked i32 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf(
			"mutable allocation length alias unexpectedly removed bounds check: %#v",
			fn.Instrs,
		)
	}
}

func TestP55AllocationTetraLiteralZeroUsesProofTaggedStoreAndUncheckedLoad(t *testing.T) {
	prog := lowerProofFileProgram(t, `
module p25.allocation

func main() -> Int
uses alloc, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 1024:
        var xs: []i32 = core.make_i32(32)
        xs[0] = r
        checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`)
	fn := findIRFunc(t, prog, "p25.allocation.main")
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("allocation_tetra shape should have one i32 store instruction: %#v", fn.Instrs)
	}
	if countProofTaggedInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("allocation_tetra literal-zero store should be proof-tagged: %#v", fn.Instrs)
	}
	storeProofID := firstProofID(fn, ir.IRIndexStoreI32)
	if !strings.HasPrefix(storeProofID, "proof:allocation-zero:literal0:xs:") {
		t.Fatalf(
			"allocation_tetra store proof id = %q, want allocation-zero literal0 proof; instrs=%#v",
			storeProofID,
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf(
			"allocation_tetra literal-zero load should not keep checked i32 load: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 1 {
		t.Fatalf(
			"allocation_tetra literal-zero load should be one proof-tagged unchecked i32 load: %#v",
			fn.Instrs,
		)
	}
	loadProofID := firstProofID(fn, ir.IRIndexLoadI32Unchecked)
	if !strings.HasPrefix(loadProofID, "proof:allocation-zero:literal0:xs:") {
		t.Fatalf(
			"allocation_tetra load proof id = %q, want allocation-zero literal0 proof; instrs=%#v",
			loadProofID,
			fn.Instrs,
		)
	}
}

func TestP55AllocationTetraPLIRRecordsLiteralZeroProofUses(t *testing.T) {
	prog := proofPLIRFileProgram(t, `
module p25.allocation

func main() -> Int
uses alloc, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 1024:
        var xs: []i32 = core.make_i32(32)
        xs[0] = r
        checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`)

	fn := findPLIRFunc(t, prog, "p25.allocation.main")
	termsByOperation := map[string]plir.ProofTerm{}
	for _, term := range fn.ProofTerms {
		if strings.HasPrefix(term.ID, "proof:allocation-zero:literal0:xs:") {
			termsByOperation[term.Operation] = term
		}
	}
	if len(termsByOperation) != 2 {
		t.Fatalf(
			"allocation_tetra should have store/load literal-zero proof terms, got %#v\n%s",
			fn.ProofTerms,
			plir.FormatText(prog),
		)
	}
	for _, operation := range []string{"index_store", "index_load"} {
		term := termsByOperation[operation]
		if term.ID == "" ||
			term.SubjectBaseID != "xs" ||
			term.IndexValueID != "local:0" ||
			term.Range != "0 in [0, xs.len)" ||
			!containsStringValue(term.FactsUsed, "allocation_literal_zero") ||
			!containsStringValue(term.FactsUsed, "allocation_length_positive") {
			t.Fatalf("allocation literal-zero %s proof term = %+v", operation, term)
		}

		guard, ok := plirProofGuardForID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing allocation literal-zero proof guard for %s/%s: %#v",
				operation,
				term.ID,
				fn.ProofGuards,
			)
		}
		if guard.Kind != "range" ||
			!strings.Contains(guard.Condition, "0 < xs.len") ||
			!strings.Contains(guard.Condition, "xs.len == 32") ||
			guard.Reason != "allocation literal-zero length proof" {
			t.Fatalf("allocation literal-zero %s proof guard = %+v", operation, guard)
		}

		use, ok := plirProofUseForID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing allocation literal-zero proof use for %s/%s: %#v",
				operation,
				term.ID,
				fn.ProofUses,
			)
		}
		if !plir.Dominates(fn, guard.Block, use.Block) {
			t.Fatalf(
				"allocation literal-zero guard block %s should dominate use block %s in %+v",
				guard.Block,
				use.Block,
				fn.Dominators,
			)
		}
		op, ok := plirOperationForID(fn, use.OpID)
		if !ok || string(op.Kind) != operation || len(op.Inputs) < 2 || op.Inputs[0] != "xs" ||
			op.Inputs[1] != "0" {
			t.Fatalf(
				"allocation literal-zero proof use should target %s op, use=%+v op=%+v",
				operation,
				use,
				op,
			)
		}

		rangeFact, ok := plirRangeFactForProofID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing allocation literal-zero range fact for %s/%s: %#v",
				operation,
				term.ID,
				fn.RangeFacts,
			)
		}
		if rangeFact.Value != "local:0" ||
			rangeFact.Lower != (plir.Bound{Kind: plir.BoundConst, Const: 0}) ||
			rangeFact.Upper != (plir.Bound{Kind: plir.BoundSymbol, Symbol: "xs.len"}) ||
			!rangeFact.InclusiveLower ||
			rangeFact.InclusiveUpper ||
			!containsStringValue(rangeFact.Derivation, "allocation_literal_zero") ||
			!containsStringValue(rangeFact.Derivation, "allocation_length_positive") {
			t.Fatalf("allocation literal-zero %s range fact = %+v", operation, rangeFact)
		}
	}
}

func TestP58RegionIslandLiteralZeroUsesProofTaggedStoreAndUncheckedLoad(t *testing.T) {
	prog := lowerProofFileProgram(t, `
module p25.region_island_allocation

func main() -> Int
uses alloc, islands, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 256:
        island(256) as isl:
            var xs: []i32 = core.island_make_i32(isl, 16)
            xs[0] = r
            checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`)
	fn := findIRFunc(t, prog, "p25.region_island_allocation.main")
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf(
			"region_island_allocation shape should have one i32 store instruction: %#v",
			fn.Instrs,
		)
	}
	if countProofTaggedInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf(
			"region_island_allocation literal-zero store should be proof-tagged: %#v",
			fn.Instrs,
		)
	}
	storeProofID := firstProofID(fn, ir.IRIndexStoreI32)
	if !strings.HasPrefix(storeProofID, "proof:allocation-zero:literal0:xs:") {
		t.Fatalf(
			"region_island_allocation store proof id = %q, want allocation-zero literal0 proof; instrs=%#v",
			storeProofID,
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf(
			"region_island_allocation literal-zero load should not keep checked i32 load: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 1 {
		t.Fatalf(
			"region_island_allocation literal-zero load should be one proof-tagged unchecked i32 load: %#v",
			fn.Instrs,
		)
	}
	loadProofID := firstProofID(fn, ir.IRIndexLoadI32Unchecked)
	if !strings.HasPrefix(loadProofID, "proof:allocation-zero:literal0:xs:") {
		t.Fatalf(
			"region_island_allocation load proof id = %q, want allocation-zero literal0 proof; instrs=%#v",
			loadProofID,
			fn.Instrs,
		)
	}
}

func TestP58RegionIslandLiteralZeroPLIRRecordsIslandProofUses(t *testing.T) {
	prog := proofPLIRFileProgram(t, `
module p25.region_island_allocation

func main() -> Int
uses alloc, islands, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 256:
        island(256) as isl:
            var xs: []i32 = core.island_make_i32(isl, 16)
            xs[0] = r
            checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`)

	fn := findPLIRFunc(t, prog, "p25.region_island_allocation.main")
	termsByOperation := map[string]plir.ProofTerm{}
	for _, term := range fn.ProofTerms {
		if strings.HasPrefix(term.ID, "proof:allocation-zero:literal0:xs:") {
			termsByOperation[term.Operation] = term
		}
	}
	if len(termsByOperation) != 2 {
		t.Fatalf(
			"region_island_allocation should have store/load literal-zero proof terms, got %#v\n%s",
			fn.ProofTerms,
			plir.FormatText(prog),
		)
	}
	for _, operation := range []string{"index_store", "index_load"} {
		term := termsByOperation[operation]
		if term.ID == "" ||
			term.SubjectBaseID != "xs" ||
			term.IndexValueID != "local:0" ||
			term.Range != "0 in [0, xs.len)" ||
			term.IslandID != "island:isl" ||
			term.Epoch <= 0 ||
			term.BaseID == "" ||
			!containsStringValue(term.FactsUsed, "allocation_literal_zero") ||
			!containsStringValue(term.FactsUsed, "allocation_length_positive") {
			t.Fatalf("region island allocation literal-zero %s proof term = %+v", operation, term)
		}

		guard, ok := plirProofGuardForID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing region island allocation literal-zero proof guard for %s/%s: %#v",
				operation,
				term.ID,
				fn.ProofGuards,
			)
		}
		if guard.Kind != "range" ||
			!strings.Contains(guard.Condition, "0 < xs.len") ||
			!strings.Contains(guard.Condition, "xs.len == 16") ||
			guard.Reason != "allocation literal-zero length proof" {
			t.Fatalf("region island allocation literal-zero %s proof guard = %+v", operation, guard)
		}

		use, ok := plirProofUseForID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing region island allocation literal-zero proof use for %s/%s: %#v",
				operation,
				term.ID,
				fn.ProofUses,
			)
		}
		if !plir.Dominates(fn, guard.Block, use.Block) {
			t.Fatalf(
				"region island allocation literal-zero guard block %s should dominate use block %s in %+v",
				guard.Block,
				use.Block,
				fn.Dominators,
			)
		}
		op, ok := plirOperationForID(fn, use.OpID)
		if !ok || string(op.Kind) != operation || len(op.Inputs) < 2 || op.Inputs[0] != "xs" ||
			op.Inputs[1] != "0" {
			t.Fatalf(
				"region island allocation literal-zero proof use should target %s op, use=%+v op=%+v",
				operation,
				use,
				op,
			)
		}

		rangeFact, ok := plirRangeFactForProofID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing region island allocation literal-zero range fact for %s/%s: %#v",
				operation,
				term.ID,
				fn.RangeFacts,
			)
		}
		if rangeFact.Value != "local:0" ||
			rangeFact.Lower != (plir.Bound{Kind: plir.BoundConst, Const: 0}) ||
			rangeFact.Upper != (plir.Bound{Kind: plir.BoundSymbol, Symbol: "xs.len"}) ||
			!rangeFact.InclusiveLower ||
			rangeFact.InclusiveUpper ||
			!containsStringValue(rangeFact.Derivation, "allocation_literal_zero") ||
			!containsStringValue(rangeFact.Derivation, "allocation_length_positive") {
			t.Fatalf(
				"region island allocation literal-zero %s range fact = %+v",
				operation,
				rangeFact,
			)
		}
	}
}

func TestP55AllocationLiteralZeroMutableLengthKeepsChecks(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, mem:
    var n: Int = 32
    var xs: []i32 = core.make_i32(n)
    xs[0] = 7
    return xs[0]
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 ||
		countProofTaggedInstrKind(fn, ir.IRIndexStoreI32) != 0 {
		t.Fatalf(
			"mutable allocation length literal-zero store should keep checked store: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 ||
		countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf(
			"mutable allocation length literal-zero load should keep checked load: %#v",
			fn.Instrs,
		)
	}
}

func TestP55AllocationLiteralZeroInoutMutationKeepsChecks(t *testing.T) {
	prog := lowerProofProgram(t, `
func touch(view: inout []i32) -> Int
uses mem:
    return view.len

func main() -> Int
uses alloc, mem:
    var xs: []i32 = core.make_i32(32)
    touch(xs)
    xs[0] = 7
    return xs[0]
`)
	fn := findIRFunc(t, prog, "main")
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 ||
		countProofTaggedInstrKind(fn, ir.IRIndexStoreI32) != 0 {
		t.Fatalf(
			"inout mutation before literal-zero store should keep checked store: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 ||
		countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("inout mutation before literal-zero load should keep checked load: %#v", fn.Instrs)
	}
}

func TestP55AllocationNonzeroOrOutOfRangeLiteralKeepsChecks(t *testing.T) {
	tests := []struct {
		name   string
		length string
		index  string
	}{
		{name: "nonzero_literal", length: "32", index: "1"},
		{name: "zero_length_literal_zero", length: "0", index: "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := lowerProofProgram(t, strings.NewReplacer(
				"$LENGTH", tt.length,
				"$INDEX", tt.index,
			).Replace(`
func main() -> Int
uses alloc, mem:
    var xs: []i32 = core.make_i32($LENGTH)
    xs[$INDEX] = 7
    return xs[$INDEX]
`))
			fn := findIRFunc(t, prog, "main")
			if countInstrKind(fn, ir.IRIndexStoreI32) != 1 ||
				countProofTaggedInstrKind(fn, ir.IRIndexStoreI32) != 0 {
				t.Fatalf("%s store should keep checked store: %#v", tt.name, fn.Instrs)
			}
			if countInstrKind(fn, ir.IRIndexLoadI32) != 1 ||
				countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
				t.Fatalf("%s load should keep checked load: %#v", tt.name, fn.Instrs)
			}
		})
	}
}

func TestP58RegionIslandLiteralZeroNegativeCasesKeepChecks(t *testing.T) {
	tests := []struct {
		name       string
		lengthDecl string
		makeLength string
		index      string
	}{
		{name: "zero_length_literal_zero", makeLength: "0", index: "0"},
		{
			name:       "dynamic_length_literal_zero",
			lengthDecl: "var n: Int = 16",
			makeLength: "n",
			index:      "0",
		},
		{name: "nonzero_literal", makeLength: "16", index: "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := lowerProofFileProgram(t, strings.NewReplacer(
				"$LENGTH_DECL", tt.lengthDecl,
				"$MAKE_LENGTH", tt.makeLength,
				"$INDEX", tt.index,
			).Replace(`
module p58.region_island_negative

func main() -> Int
uses alloc, islands, mem:
    island(256) as isl:
        $LENGTH_DECL
        var xs: []i32 = core.island_make_i32(isl, $MAKE_LENGTH)
        xs[$INDEX] = 7
        return xs[$INDEX]
    return 0
`))
			fn := findIRFunc(t, prog, "p58.region_island_negative.main")
			if countInstrKind(fn, ir.IRIndexStoreI32) != 1 ||
				countProofTaggedInstrKind(fn, ir.IRIndexStoreI32) != 0 {
				t.Fatalf("%s store should keep checked store: %#v", tt.name, fn.Instrs)
			}
			if countInstrKind(fn, ir.IRIndexLoadI32) != 1 ||
				countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
				t.Fatalf("%s load should keep checked load: %#v", tt.name, fn.Instrs)
			}
		})
	}
}

func TestAllocationLengthAliasRejectsMutableMakeLengthBuiltins(t *testing.T) {
	for _, name := range []string{
		"make_u8", "make_u16", "make_i32", "make_bool",
		"core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool",
	} {
		t.Run(name, func(t *testing.T) {
			l := &lowerer{locals: map[string]semantics.LocalInfo{
				"n": {Mutable: true},
			}}
			_, ok := l.allocationLengthBoundLocal(&frontend.CallExpr{
				Name: name,
				Args: []frontend.Expr{&frontend.IdentExpr{Name: "n"}},
			})
			if ok {
				t.Fatalf("allocationLengthBoundLocal(%s, mutable n) accepted mutable length", name)
			}
		})
	}
}

func TestNonDominatingIfGuardKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func get(xs: []i32, i: Int) -> Int
uses mem:
    var j = i
    if j < xs.len:
        j = j + 0
    return xs[j]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return get(xs, 0)
`)
	fn := findIRFunc(t, prog, "get")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("non-dominating branch guard should keep checked load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("non-dominating branch guard unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestDominatingIfGuardWithZeroIndexUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func get(xs: []i32) -> Int
uses mem:
    var i = 0
    if i < xs.len:
        return xs[i]
    return 0

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 7
    return get(xs)
`)
	fn := findIRFunc(t, prog, "get")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("dominating if guard with zero index should remove checked load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:if:") {
		t.Fatalf(
			"if unchecked load proof id = %q, want proof:if prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestDominatingIfGuardWithExplicitLowerBoundUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func get(xs: []i32, i: Int) -> Int
uses mem:
    if i >= 0 && i < xs.len:
        return xs[i]
    return 0

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 7
    return get(xs, 0)
`)
	fn := findIRFunc(t, prog, "get")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf(
			"dominating if guard with explicit lower bound should remove checked load: %#v",
			fn.Instrs,
		)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:if:") {
		t.Fatalf(
			"if lower-bound unchecked load proof id = %q, want proof:if prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestIfUpperGuardWithoutLowerBoundKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func get(xs: []i32, i: Int) -> Int
uses mem:
    if i < xs.len:
        return xs[i]
    return 0

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 7
    return get(xs, 0)
`)
	fn := findIRFunc(t, prog, "get")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("if upper guard without lower bound should keep checked load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf(
			"if upper guard without lower bound unexpectedly removed bounds check: %#v",
			fn.Instrs,
		)
	}
}

func TestNestedWhileLoopUsesInnerProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum_nested(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        var j = 0
        while j < xs.len:
            total = total + xs[j]
            j = j + 1
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum_nested(xs)
`)
	fn := findIRFunc(t, prog, "sum_nested")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("nested while loop should remove checked inner load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:while:",
	) {
		t.Fatalf(
			"nested while unchecked load proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestRawSliceWhileLoopKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum_raw(xs: []u8) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let view: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
        var total = 0
        var i = 0
        while i < view.len:
            total = total + view[i]
            i = i + 1
        return total
    return 0

func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(1)
    xs[0] = 1
    return sum_raw(xs)
`)
	fn := findIRFunc(t, prog, "sum_raw")
	if countInstrKind(fn, ir.IRIndexLoadU8) != 1 {
		t.Fatalf("raw slice while loop should keep checked u8 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf("raw slice while loop unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestRawI32SliceWhileLoopKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum_raw(xs: []i32) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let view: []i32 = core.raw_slice_i32_from_parts(xs.ptr, xs.len, mem)
        var total = 0
        var i = 0
        while i < view.len:
            total = total + view[i]
            i = i + 1
        return total
    return 0

func main() -> Int
uses alloc, capability, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return sum_raw(xs)
`)
	fn := findIRFunc(t, prog, "sum_raw")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 1 {
		t.Fatalf("raw i32 slice while loop should keep checked i32 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 0 {
		t.Fatalf("raw i32 slice while loop unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestTextInsertBytesSourceLoopUsesProofTaggedUncheckedLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func insert_shape(buf: inout []u8, idx: Int, bytes: []u8) -> Int
uses mem:
    var i: Int = 0
    while i < bytes.len:
        buf[idx + i] = bytes[i]
        i = i + 1
    return i

func main() -> Int
uses alloc, mem:
    var buf: []u8 = make_u8(4)
    var bytes: []u8 = make_u8(2)
    bytes[0] = 79
    bytes[1] = 75
    return insert_shape(buf, 0, bytes)
`)
	fn := findIRFunc(t, prog, "insert_shape")
	if countInstrKind(fn, ir.IRIndexLoadU8) != 0 {
		t.Fatalf("insert_bytes source loop should not keep checked bytes[i] load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadU8Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf(
			"insert_bytes source loop proof id = %q, want proof:while prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexStoreU8) != 1 {
		t.Fatalf(
			"insert_bytes destination store should remain one checked buf[idx+i] store: %#v",
			fn.Instrs,
		)
	}
}

func TestRawSliceFromPartsLowersElementSizeShift(t *testing.T) {
	prog := lowerProofProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let xs: []i32 = core.raw_slice_i32_from_parts(p, 2, mem)
        return xs.len
    return 0
`)
	fn := findIRFunc(t, prog, "main")
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRRawSliceFromParts {
			if instr.Imm != 2 {
				t.Fatalf("raw_slice_i32_from_parts shift = %d, want 2: %#v", instr.Imm, fn.Instrs)
			}
			return
		}
	}
	t.Fatalf("main missing IRRawSliceFromParts: %#v", fn.Instrs)
}

func TestRawSliceAliasWhileLoopKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum_raw(xs: []u8) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let view: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
        let alias: []u8 = view
        var total = 0
        var i = 0
        while i < alias.len:
            total = total + alias[i]
            i = i + 1
        return total
    return 0

func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(1)
    xs[0] = 1
    return sum_raw(xs)
`)
	fn := findIRFunc(t, prog, "sum_raw")
	if countInstrKind(fn, ir.IRIndexLoadU8) != 1 {
		t.Fatalf("raw slice alias while loop should keep checked u8 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf("raw slice alias while loop unexpectedly removed bounds check: %#v", fn.Instrs)
	}
}

func TestInvalidStringViewAliasLoopKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum_bad() -> Int:
    let view: String = core.string_window("abc", 4, 0)
    let alias: String = view
    var total = 0
    var i = 0
    while i < alias.len:
        total = total + alias[i]
        i = i + 1
    return total

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "sum_bad")
	if countInstrKind(fn, ir.IRIndexLoadU8) != 1 {
		t.Fatalf("invalid String alias while loop should keep checked u8 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf(
			"invalid String alias while loop unexpectedly removed bounds check: %#v",
			fn.Instrs,
		)
	}
}

func TestBranchJoinInvalidAliasWhileLoopKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum_join(flag: Int) -> Int
uses mem:
    var alias: String = "abc"
    if flag:
        alias = core.string_window("abc", 4, 0)
    else:
        alias = "abc"
    var total = 0
    var i = 0
    while i < alias.len:
        total = total + alias[i]
        i = i + 1
    return total

func main() -> Int
uses mem:
    return sum_join(1)
`)
	fn := findIRFunc(t, prog, "sum_join")
	if countInstrKind(fn, ir.IRIndexLoadU8) != 1 {
		t.Fatalf(
			"branch-joined invalid alias while loop should keep checked u8 load: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf(
			"branch-joined invalid alias while loop unexpectedly removed bounds check: %#v",
			fn.Instrs,
		)
	}
}

func TestForSliceWindowLoopUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
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
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("sum still contains checked i32 index load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadI32Unchecked) != 1 {
		t.Fatalf("sum missing unchecked i32 index load over window: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRSliceWindow) != 1 {
		t.Fatalf("sum missing slice window constructor IR: %#v", fn.Instrs)
	}
}

func TestForStringWindowLoopUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
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
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadU8) != 0 {
		t.Fatalf("sum still contains checked u8 index load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 1 {
		t.Fatalf("sum missing unchecked u8 index load over String window: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRSliceWindow) != 1 {
		t.Fatalf("sum missing String window constructor IR: %#v", fn.Instrs)
	}
}

func TestForSlicePrefixSuffixViewChainUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    let view: []i32 = xs.prefix(4).suffix(1)
    var total = 0
    for x in view:
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    xs[3] = 4
    return sum(xs)
`)
	fn := findIRFunc(t, prog, "sum")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("safe prefix/suffix view chain should use unchecked load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:for-collection",
	) {
		t.Fatalf(
			"view-chain for proof id = %q, want proof:for-collection prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
}

func TestForInvalidIntermediateStringViewChainKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum_bad_chain() -> Int:
    let view: String = core.string_suffix(core.string_window("abc", 4, 0), 0)
    var total = 0
    for ch in view:
        total = total + ch
    return total

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "sum_bad_chain")
	if countInstrKind(fn, ir.IRIndexLoadU8) != 1 {
		t.Fatalf(
			"invalid intermediate String view chain should keep checked u8 index load: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf(
			"invalid intermediate String view chain unexpectedly contains unchecked u8 index load: %#v",
			fn.Instrs,
		)
	}
}

func TestForRawSliceViewChainKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
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
    xs[0] = 1
    return sum_raw_chain(xs)
`)
	fn := findIRFunc(t, prog, "sum_raw_chain")
	if countInstrKind(fn, ir.IRIndexLoadU8) != 1 {
		t.Fatalf("raw-derived view chain should keep checked u8 index load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf(
			"raw-derived view chain unexpectedly contains unchecked u8 index load: %#v",
			fn.Instrs,
		)
	}
}

func TestForInvalidStringViewAliasKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum_bad() -> Int:
    let view: String = core.string_window("abc", 4, 0)
    var total = 0
    for ch in view:
        total = total + ch
    return total

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "sum_bad")
	if countInstrKind(fn, ir.IRIndexLoadU8) != 1 {
		t.Fatalf(
			"invalid String view alias for-loop should keep checked u8 index load: %#v",
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf(
			"invalid String view alias for-loop unexpectedly contains unchecked u8 index load: %#v",
			fn.Instrs,
		)
	}
}

func TestCopyLoopSourceLoadUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func copied_len(xs: []i32) -> Int
uses alloc, mem:
    let copied: []i32 = xs.copy()
    return copied.len

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return copied_len(xs)
`)
	fn := findIRFunc(t, prog, "copied_len")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("copy loop source load should be proof-tagged unchecked: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:copy-loop:",
	) {
		t.Fatalf("copy loop proof id = %q, want proof:copy-loop prefix; instrs=%#v", got, fn.Instrs)
	}
}

func TestCopyIntoLoopSourceLoadUsesProofTaggedUncheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func copy_count(src: []i32, dst: inout []i32) -> Int
uses mem:
    return src.copy_into(dst)

func main() -> Int
uses alloc, mem:
    var src: []i32 = make_i32(1)
    var dst: []i32 = make_i32(1)
    src[0] = 1
    return copy_count(src, dst)
`)
	fn := findIRFunc(t, prog, "copy_count")
	if countInstrKind(fn, ir.IRIndexLoadI32) != 0 {
		t.Fatalf("copy_into loop source load should be proof-tagged unchecked: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(
		got,
		"proof:copy-loop:",
	) {
		t.Fatalf(
			"copy_into loop proof id = %q, want proof:copy-loop prefix; instrs=%#v",
			got,
			fn.Instrs,
		)
	}
	if countInstrKind(fn, ir.IRIndexStoreI32) != 1 {
		t.Fatalf("copy_into destination store should remain checked store: %#v", fn.Instrs)
	}
}

func TestInvalidStringViewLoopKeepsCheckedIndexLoad(t *testing.T) {
	prog := lowerProofProgram(t, `
func sum_bad() -> Int:
    var total = 0
    for ch in core.string_window("abc", 4, 0):
        total = total + ch
    return total

func main() -> Int:
    return 0
`)
	fn := findIRFunc(t, prog, "sum_bad")
	if countInstrKind(fn, ir.IRIndexLoadU8) != 1 {
		t.Fatalf("invalid String view loop should keep checked u8 index load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf(
			"invalid String view loop unexpectedly contains unchecked u8 index load: %#v",
			fn.Instrs,
		)
	}
}

func findIRFunc(t *testing.T, prog *ir.IRProgram, name string) ir.IRFunc {
	t.Helper()
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing IR function %s", name)
	return ir.IRFunc{}
}

func findPLIRFunc(t *testing.T, prog *plir.Program, name string) plir.Function {
	t.Helper()
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing PLIR function %s", name)
	return plir.Function{}
}

func countInstrKind(fn ir.IRFunc, kind ir.IRInstrKind) int {
	total := 0
	for _, instr := range fn.Instrs {
		if instr.Kind == kind {
			total++
		}
	}
	return total
}

func firstProofID(fn ir.IRFunc, kind ir.IRInstrKind) string {
	for _, instr := range fn.Instrs {
		if instr.Kind == kind {
			return instr.ProofID
		}
	}
	return ""
}

func proofIDsForKind(fn ir.IRFunc, kind ir.IRInstrKind) []string {
	var out []string
	for _, instr := range fn.Instrs {
		if instr.Kind == kind && instr.ProofID != "" {
			out = append(out, instr.ProofID)
		}
	}
	return out
}

func containsProofIDPrefix(ids []string, prefix string) bool {
	for _, id := range ids {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}

func containsStringValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func plirProofGuardForID(fn plir.Function, id string) (plir.ProofGuard, bool) {
	for _, guard := range fn.ProofGuards {
		if guard.ID == id {
			return guard, true
		}
	}
	return plir.ProofGuard{}, false
}

func plirProofUseForID(fn plir.Function, id string) (plir.ProofUse, bool) {
	for _, use := range fn.ProofUses {
		if use.ProofID == id {
			return use, true
		}
	}
	return plir.ProofUse{}, false
}

func plirOperationForID(fn plir.Function, id string) (plir.Operation, bool) {
	for _, op := range fn.Ops {
		if op.ID == id {
			return op, true
		}
	}
	return plir.Operation{}, false
}

func plirRangeFactForProofID(fn plir.Function, id string) (plir.RangeFact, bool) {
	for _, fact := range fn.RangeFacts {
		if fact.ProofID == id {
			return fact, true
		}
	}
	return plir.RangeFact{}, false
}

func countProofTaggedInstrKind(fn ir.IRFunc, kind ir.IRInstrKind) int {
	total := 0
	for _, instr := range fn.Instrs {
		if instr.Kind == kind && instr.ProofID != "" {
			total++
		}
	}
	return total
}

func countProofIDPrefixForKind(fn ir.IRFunc, kind ir.IRInstrKind, prefix string) int {
	total := 0
	for _, instr := range fn.Instrs {
		if instr.Kind == kind && strings.HasPrefix(instr.ProofID, prefix) {
			total++
		}
	}
	return total
}

func matrixAffineStoreProgram(
	cDecl string,
	rowGuard string,
	colGuard string,
	storeIndex string,
	rowInc string,
	colInc string,
	beforeStore string,
) string {
	if beforeStore != "" {
		beforeStore = "\n            " + beforeStore
	}
	return strings.NewReplacer(
		"$C_DECL", cDecl,
		"$ROW_GUARD", rowGuard,
		"$COL_GUARD", colGuard,
		"$STORE_INDEX", storeIndex,
		"$ROW_INC", rowInc,
		"$COL_INC", colInc,
		"$BEFORE_STORE", beforeStore,
	).Replace(`
func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    var b: []i32 = core.make_i32(9)
    $C_DECL
    var row: Int = 0
    while $ROW_GUARD:
        var col: Int = 0
        while $COL_GUARD:
            var k: Int = 0
            var total: Int = 0
            while k < 3:
                total = total + a[row * 3 + k] * b[k * 3 + col]
                k = k + 1$BEFORE_STORE
            c[$STORE_INDEX] = total
            $COL_INC
        $ROW_INC
    return 0
`)
}

func matrixAffineBLoadProgram(
	bDecl string,
	rowGuard string,
	colGuard string,
	kGuard string,
	bLoadIndex string,
	rowInc string,
	kInc string,
	colInc string,
	beforeLoad string,
) string {
	if beforeLoad != "" {
		beforeLoad = "\n                " + beforeLoad
	}
	return strings.NewReplacer(
		"$B_DECL", bDecl,
		"$ROW_GUARD", rowGuard,
		"$COL_GUARD", colGuard,
		"$K_GUARD", kGuard,
		"$B_LOAD_INDEX", bLoadIndex,
		"$ROW_INC", rowInc,
		"$K_INC", kInc,
		"$COL_INC", colInc,
		"$BEFORE_LOAD", beforeLoad,
	).Replace(`
func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    $B_DECL
    var c: []i32 = core.make_i32(9)
    var row: Int = 0
    while $ROW_GUARD:
        var col: Int = 0
        while $COL_GUARD:
            var k: Int = 0
            var total: Int = 0
            while $K_GUARD:$BEFORE_LOAD
                total = total + a[row * 3 + k] * b[$B_LOAD_INDEX]
                $K_INC
            c[row * 3 + col] = total
            $COL_INC
        $ROW_INC
    return 0
`)
}

func matrixAffineLoadProgram(
	aDecl string,
	cDecl string,
	rowGuard string,
	kGuard string,
	aLoadIndex string,
	colGuard string,
	storeIndex string,
	rowInc string,
	kInc string,
	colInc string,
	beforeLoad string,
) string {
	if beforeLoad != "" {
		beforeLoad = "\n                " + beforeLoad
	}
	return strings.NewReplacer(
		"$A_DECL", aDecl,
		"$C_DECL", cDecl,
		"$ROW_GUARD", rowGuard,
		"$K_GUARD", kGuard,
		"$A_LOAD_INDEX", aLoadIndex,
		"$COL_GUARD", colGuard,
		"$STORE_INDEX", storeIndex,
		"$ROW_INC", rowInc,
		"$K_INC", kInc,
		"$COL_INC", colInc,
		"$BEFORE_LOAD", beforeLoad,
	).Replace(`
func main() -> Int
uses alloc, mem:
    $A_DECL
    var b: []i32 = core.make_i32(9)
    $C_DECL
    var row: Int = 0
    while $ROW_GUARD:
        var col: Int = 0
        while $COL_GUARD:
            var k: Int = 0
            var total: Int = 0
            while $K_GUARD:$BEFORE_LOAD
                total = total + a[$A_LOAD_INDEX] * b[k * 3 + col]
                $K_INC
            c[$STORE_INDEX] = total
            $COL_INC
        $ROW_INC
    return 0
`)
}

// ---- raw_memory_test.go ----

func TestLowerRawPtrAddDirectOffsetMemoryAccessIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let stored: Int = core.store_i32(core.ptr_add(p, 4, mem), 42, mem)
        let value: Int = core.load_i32(core.ptr_add(p, 4, mem), mem)
        let stored_ptr: ptr = core.store_ptr(core.ptr_add(p, 8, mem), p, mem)
        let loaded_ptr: ptr = core.load_ptr(core.ptr_add(p, 8, mem), mem)
        let stored_arch_ptr: ptr = core.store_arch_ptr(core.ptr_add(p, 8, mem), p, mem)
        return value
    return 0
`, "main")

	if got := countInstr(fn.Instrs, ir.IRMemWriteI32Offset, ""); got != 1 {
		t.Fatalf(
			"direct ptr_add store_i32 should lower to one offset write, got %d: %#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstr(fn.Instrs, ir.IRMemReadI32Offset, ""); got != 1 {
		t.Fatalf(
			"direct ptr_add load_i32 should lower to one offset read, got %d: %#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstr(fn.Instrs, ir.IRMemWritePtrOffset, ""); got != 1 {
		t.Fatalf(
			"direct ptr_add store_ptr should lower to one offset write, got %d: %#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstr(fn.Instrs, ir.IRMemReadPtrOffset, ""); got != 1 {
		t.Fatalf(
			"direct ptr_add load_ptr should lower to one offset read, got %d: %#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstr(fn.Instrs, ir.IRMemWriteArchPtrOffset, ""); got != 1 {
		t.Fatalf(
			"direct ptr_add store_arch_ptr should lower to one offset write, got %d: %#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstr(fn.Instrs, ir.IRPtrAdd, ""); got != 0 {
		t.Fatalf(
			"direct ptr_add memory access should fold into offset IR, got %d ptr_add instructions: %#v",
			got,
			fn.Instrs,
		)
	}
}

func TestLowerRawPtrAddLocalOffsetMemoryAccessIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let q: ptr = core.ptr_add(p, 4, mem)
        let stored: Int = core.store_i32(q, 42, mem)
        return core.load_i32(q, mem)
    return 0
`, "main")

	if got := countInstr(fn.Instrs, ir.IRMemWriteI32Offset, ""); got != 1 {
		t.Fatalf(
			"local ptr_add store_i32 should lower to one offset write, got %d: %#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstr(fn.Instrs, ir.IRMemReadI32Offset, ""); got != 1 {
		t.Fatalf(
			"local ptr_add load_i32 should lower to one offset read, got %d: %#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstr(fn.Instrs, ir.IRPtrAdd, ""); got != 1 {
		t.Fatalf(
			("local ptr_add should keep exactly one value-producing ptr_add " +
				"for q initialization, got %d: %#v"),
			got,
			fn.Instrs,
		)
	}
}

func TestLowerRawPtrAddMutableLocalWithDiscardOffsetMemoryAccessIR(t *testing.T) {
	fn := lowerCallableFunc(t, `
func main() -> i32
uses alloc, capability, mem:
    var out: i32 = 1
    unsafe:
        var mem: cap.mem = core.cap_mem()
        var p: ptr = core.alloc_bytes(8)
        var q: ptr = core.ptr_add(p, 4, mem)
        var _: i32 = core.store_i32(q, 123, mem)
        var v: i32 = core.load_i32(q, mem)
        if v == 123:
            out = 77
        else:
            out = 1
    return out
`, "main")

	if got := countInstr(fn.Instrs, ir.IRMemWriteI32Offset, ""); got != 1 {
		t.Fatalf(
			"mutable local ptr_add store_i32 should lower to one offset write, got %d: %#v",
			got,
			fn.Instrs,
		)
	}
	if got := countInstr(fn.Instrs, ir.IRMemReadI32Offset, ""); got != 1 {
		t.Fatalf(
			"mutable local ptr_add load_i32 should lower to one offset read, got %d: %#v",
			got,
			fn.Instrs,
		)
	}
}

// ---- slice_bool_test.go ----

func TestLowerBoolSliceBuiltinsUseI32LayoutIR(t *testing.T) {
	src := []byte(`
func main() -> Int
uses alloc, islands, mem:
    var xs: []bool = make_bool(2)
    xs[0] = true
    xs[1] = false
    island(64) as isl:
        var ys: []bool = core.island_make_bool(isl, 1)
        ys[0] = xs[0]
    if xs[0]:
        return 1
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	var mainFn *ir.IRFunc
	for i := range irProg.Funcs {
		if irProg.Funcs[i].Name == "main" {
			mainFn = &irProg.Funcs[i]
			break
		}
	}
	if mainFn == nil {
		t.Fatalf("main function not found in IR output")
	}

	makeI32Count := 0
	islandMakeI32Count := 0
	for _, instr := range mainFn.Instrs {
		switch instr.Kind {
		case ir.IRMakeSliceI32:
			makeI32Count++
		case ir.IRIslandMakeSliceI32:
			islandMakeI32Count++
		}
	}

	if makeI32Count == 0 {
		t.Fatalf("expected IRMakeSliceI32 for make_bool")
	}
	if islandMakeI32Count == 0 {
		t.Fatalf("expected IRIslandMakeSliceI32 for island_make_bool")
	}
}

// ---- surface_test.go ----

func TestLowerSurfaceHostBuiltinsCallRuntimeABI(t *testing.T) {
	checked := checkCallableProgram(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("demo", 10, 10)
    let event: Int = core.surface_poll_event_kind(handle)
    let event_x: Int = core.surface_poll_event_x(handle)
    let event_y: Int = core.surface_poll_event_y(handle)
    let event_button: Int = core.surface_poll_event_button(handle)
    let event_slots: []i32 = core.make_i32(5)
    let event_copied: Int = core.surface_poll_event_into(handle, event_slots)
    let event_text_len: Int = core.surface_poll_event_text_len(handle)
    let text: []u8 = core.make_u8(4)
    let event_text_copied: Int = core.surface_poll_event_text_into(handle, text)
    let clipboard_write: Int = core.surface_clipboard_write_text(handle, text)
    let clipboard_read: Int = core.surface_clipboard_read_text_into(handle, text)
    let composition_slots: []i32 = core.make_i32(4)
    let composition_copied: Int = core.surface_poll_composition_into(handle, composition_slots)
    let _: Int = core.surface_begin_frame(handle)
    let pixels: []u8 = core.make_u8(4)
    let presented: Int = core.surface_present_rgba(handle, pixels, 1, 1, 4)
    let redraw: Int = core.surface_request_redraw(handle)
    let closed: Int = core.surface_close(handle)
    return handle + event + event_x + event_y + event_button + event_copied + event_text_len + event_text_copied + clipboard_write + clipboard_read + composition_copied + presented + redraw + closed + core.surface_now_ms()
`)

	prog, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFuncByName(t, prog.Funcs, "main")

	for _, tc := range []struct {
		name string
		args int
		rets int
	}{
		{name: "__tetra_surface_open", args: 4, rets: 1},
		{name: "__tetra_surface_poll_event_kind", args: 1, rets: 1},
		{name: "__tetra_surface_poll_event_x", args: 1, rets: 1},
		{name: "__tetra_surface_poll_event_y", args: 1, rets: 1},
		{name: "__tetra_surface_poll_event_button", args: 1, rets: 1},
		{name: "__tetra_surface_poll_event_into", args: 3, rets: 1},
		{name: "__tetra_surface_poll_event_text_len", args: 1, rets: 1},
		{name: "__tetra_surface_poll_event_text_into", args: 3, rets: 1},
		{name: "__tetra_surface_clipboard_write_text", args: 3, rets: 1},
		{name: "__tetra_surface_clipboard_read_text_into", args: 3, rets: 1},
		{name: "__tetra_surface_poll_composition_into", args: 3, rets: 1},
		{name: "__tetra_surface_begin_frame", args: 1, rets: 1},
		{name: "__tetra_surface_present_rgba", args: 6, rets: 1},
		{name: "__tetra_surface_request_redraw", args: 1, rets: 1},
		{name: "__tetra_surface_close", args: 1, rets: 1},
		{name: "__tetra_surface_now_ms", args: 0, rets: 1},
	} {
		if countSurfaceRuntimeCall(mainFn.Instrs, tc.name, tc.args, tc.rets) != 1 {
			t.Fatalf(
				"main missing one %s(%d)->%d call: %#v",
				tc.name,
				tc.args,
				tc.rets,
				mainFn.Instrs,
			)
		}
	}
}

func countSurfaceRuntimeCall(instrs []ir.IRInstr, name string, args int, rets int) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind == ir.IRCall && instr.Name == name && instr.ArgSlots == args &&
			instr.RetSlots == rets {
			count++
		}
	}
	return count
}

// ---- typed_task_slots_test.go ----

func TestLowerTypedTaskWrapperSlotBounds(t *testing.T) {
	tests := []struct {
		name      string
		slotCount int
		ok        bool
	}{
		{name: "slot_1_rejected", slotCount: 1, ok: false},
		{name: "slot_2_allowed", slotCount: 2, ok: true},
		{name: "slot_4_allowed", slotCount: 4, ok: true},
		{name: "slot_5_allowed", slotCount: 5, ok: true},
		{name: "slot_8_allowed", slotCount: 8, ok: true},
		{name: "slot_9_rejected", slotCount: 9, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := typedTaskWrapper{
				Name:              "__tetra_task_typed_test",
				Target:            "worker",
				SlotCount:         tt.slotCount,
				StatusSlot:        tt.slotCount - 1,
				TargetReturnSlots: 1,
			}
			fn, err := lowerTypedTaskWrapper(wrapper)
			if tt.ok {
				if err != nil {
					t.Fatalf("lowerTypedTaskWrapper(%d): %v", tt.slotCount, err)
				}
				if fn.LocalSlots != tt.slotCount+1 {
					t.Fatalf("locals = %d, want %d", fn.LocalSlots, tt.slotCount+1)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for slot count %d", tt.slotCount)
			}
			if !strings.Contains(err.Error(), "unsupported slot count") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestLowerTypedTaskWrapperStagedThrowingTargetPassThroughStatus(t *testing.T) {
	wrapper := typedTaskWrapper{
		Name:              "__tetra_task_typed_throwing",
		Target:            "worker",
		ErrorType:         "TaskErr",
		TargetThrowsType:  "TaskErr",
		SlotCount:         5,
		StatusSlot:        4,
		TargetReturnSlots: 1,
	}
	fn, err := lowerTypedTaskWrapper(wrapper)
	if err != nil {
		t.Fatalf("lowerTypedTaskWrapper: %v", err)
	}
	if fn.LocalSlots != 0 {
		t.Fatalf("locals = %d, want 0", fn.LocalSlots)
	}
	if len(fn.Instrs) != 2 {
		t.Fatalf("instr count = %d, want 2", len(fn.Instrs))
	}
	if fn.Instrs[0].Kind != ir.IRCall || fn.Instrs[0].Name != "worker" ||
		fn.Instrs[0].RetSlots != 1 {
		t.Fatalf("first instr = %#v, want call worker ret1", fn.Instrs[0])
	}
	if fn.Instrs[1].Kind != ir.IRReturn {
		t.Fatalf("second instr = %#v, want return", fn.Instrs[1])
	}
}

// ---- ui_test.go ----

func TestLowerUIBundle(t *testing.T) {
	src := []byte(`
state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "Increment"

func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	bundle, err := LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	if bundle == nil {
		t.Fatalf("bundle = nil")
	}
	if bundle.Schema != "tetra.ui.v0.4.0" {
		t.Fatalf("schema = %q", bundle.Schema)
	}
	if len(bundle.States) != 1 || len(bundle.Views) != 1 {
		t.Fatalf("bundle = %#v", bundle)
	}
	if bundle.Views[0].StateType == "" || len(bundle.Views[0].Commands) != 1 {
		t.Fatalf("view payload = %#v", bundle.Views[0])
	}
	view := bundle.Views[0]
	if len(view.Events) != 1 || view.Events[0].Command != "increment" {
		t.Fatalf("events payload = %#v", view.Events)
	}
	if len(view.Commands) != 1 || len(view.Commands[0].Operations) != 1 {
		t.Fatalf("command operations = %#v", view.Commands)
	}
	op := view.Commands[0].Operations[0]
	if op.Kind != "state_add" || op.Target != "state.count" || op.Value != "1" {
		t.Fatalf("command operation = %#v, want state_add state.count by 1", op)
	}
	if len(view.Styles) != 1 || view.Styles[0].Value != "320" {
		t.Fatalf("styles payload = %#v", view.Styles)
	}
	if len(view.Accessibility) != 1 || view.Accessibility[0].Value != `"Increment"` {
		t.Fatalf("accessibility payload = %#v", view.Accessibility)
	}
}

func TestLowerUIBundleRecognizesStateSubtractCommands(t *testing.T) {
	src := []byte(`
state CounterState:
    var count: Int = 5

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> decrement
    command decrement:
        state.count = state.count - 2

func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	bundle, err := LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	view := bundle.Views[0]
	if len(view.Commands) != 1 || len(view.Commands[0].Operations) != 1 {
		t.Fatalf("command operations = %#v", view.Commands)
	}
	op := view.Commands[0].Operations[0]
	if op.Kind != "state_sub" || op.Target != "state.count" || op.Value != "2" {
		t.Fatalf("command operation = %#v, want state_sub state.count by 2", op)
	}
}

func TestLowerUIBundleRecognizesCompoundStateDeltaCommands(t *testing.T) {
	src := []byte(`
state CounterState:
    var count: Int = 5

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> adjust
    command adjust:
        state.count += 2
        state.count -= 1

func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	bundle, err := LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	view := bundle.Views[0]
	if len(view.Commands) != 1 || len(view.Commands[0].Operations) != 2 {
		t.Fatalf("command operations = %#v", view.Commands)
	}
	wants := []UILoweredCommandOperation{
		{Kind: "state_add", Target: "state.count", Value: "2"},
		{Kind: "state_sub", Target: "state.count", Value: "1"},
	}
	for i, want := range wants {
		if got := view.Commands[0].Operations[i]; got != want {
			t.Fatalf("operation %d = %#v, want %#v", i, got, want)
		}
	}
}

func TestLowerUIBundleRejectsNilCheckedProgram(t *testing.T) {
	if _, err := LowerUI(nil); err == nil {
		t.Fatalf("expected nil checked program error")
	}
}

func TestLowerUIBundleReturnsNilWhenUIDeclsAreMissing(t *testing.T) {
	src := []byte(`
func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	bundle, err := LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	if bundle != nil {
		t.Fatalf("bundle = %#v, want nil", bundle)
	}
}

// ---- ui_toolkit_test.go ----

func TestLowerUIToolkitBundle(t *testing.T) {
	bundle := lowerUIToolkitForTest(t, `
state FormState:
    var name: String = "tetra"
    var saved: Bool = false

view FormView(state: FormState):
    bind nameInput: String = state.name
    bind savedText: Bool = state.saved
    event input -> setName
    event click -> save
    command setName:
        state.name = "toolkit"
    command save:
        state.saved = true
    style width: Int = 640
    accessibility label: String = "Form"

func main() -> Int:
    return 0
`)
	if bundle.Schema != UIToolkitSchema {
		t.Fatalf("schema = %q, want %q", bundle.Schema, UIToolkitSchema)
	}
	if len(bundle.Views) != 1 {
		t.Fatalf("views = %#v", bundle.Views)
	}
	view := bundle.Views[0]
	for _, want := range []string{
		"window",
		"root",
		"panel",
		"text",
		"button",
		"input",
		"list",
		"table",
		"dialog",
		"menu",
	} {
		if !contains(view.WidgetKinds, want) {
			t.Fatalf("widget kinds missing %q: %#v", want, view.WidgetKinds)
		}
	}
	for _, want := range []string{"stack", "row", "column", "grid", "flex"} {
		if !contains(view.LayoutKinds, want) {
			t.Fatalf("layout kinds missing %q: %#v", want, view.LayoutKinds)
		}
	}
	if len(view.Widgets) < 5 {
		t.Fatalf("widgets = %#v", view.Widgets)
	}
	if len(view.Events) != 2 || view.Events[0].Name != "click" || view.Events[1].Name != "input" {
		t.Fatalf("events should be deterministic and sorted: %#v", view.Events)
	}
	if len(view.Commands) != 2 || len(view.Commands[0].Operations) == 0 {
		t.Fatalf("commands = %#v", view.Commands)
	}
}

func TestLowerUIToolkitRejectsUnsupportedCommandOperation(t *testing.T) {
	_, err := LowerUIToolkit(&UILoweredBundle{
		Schema: UIBundleSchema,
		Views: []UILoweredView{{
			Name:      "BadView",
			Module:    "main",
			StateType: "BadState",
			Commands: []UILoweredCommand{{
				Name:           "unsupported",
				StatementCount: 1,
			}},
		}},
	})
	if err == nil {
		t.Fatalf("expected unsupported toolkit operation error")
	}
	if !strings.Contains(err.Error(), "unsupported UI toolkit command operation") {
		t.Fatalf("error = %v", err)
	}
}

func lowerUIToolkitForTest(t *testing.T, src string) *UIToolkitBundle {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	ui, err := LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	bundle, err := LowerUIToolkit(ui)
	if err != nil {
		t.Fatalf("LowerUIToolkit: %v", err)
	}
	if bundle == nil {
		t.Fatalf("bundle = nil")
	}
	return bundle
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// ---- unsupported_test.go ----

func TestLowerUnsupportedStatementNamesFeature(t *testing.T) {
	err := (&lowerer{}).lowerStmt(&frontend.ExpectStmt{
		At: frontend.Position{File: "lower.tetra", Line: 4, Col: 3},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, want := range []string{"lower.tetra:4:3", "unsupported statement kind", "ExpectStmt"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
	diag, ok := frontend.DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != DiagnosticCodeLowerUnsupported || diag.File != "lower.tetra" ||
		diag.Line != 4 ||
		diag.Column != 3 {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if !strings.Contains(diag.Hint, "lowering") {
		t.Fatalf("hint = %q", diag.Hint)
	}
}

func TestLowerUnsupportedExpressionNamesFeature(t *testing.T) {
	errExpr := &frontend.SomePatternExpr{
		At: frontend.Position{File: "lower.tetra", Line: 5, Col: 9},
	}
	_, err := (&lowerer{}).lowerExpr(errExpr)
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, want := range []string{
		"lower.tetra:5:9",
		"unsupported expression kind",
		"SomePatternExpr",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
	diag, ok := frontend.DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != DiagnosticCodeLowerUnsupported || diag.File != "lower.tetra" ||
		diag.Line != 5 ||
		diag.Column != 9 {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestLowerUnsupportedOperatorsNameOperator(t *testing.T) {
	pos := frontend.Position{File: "lower.tetra", Line: 6, Col: 5}
	l := &lowerer{}
	_, err := l.lowerExpr(&frontend.UnaryExpr{
		At: pos,
		Op: frontend.TokenQuestion,
		X:  &frontend.NumberExpr{At: pos, Value: 1},
	})
	if err == nil {
		t.Fatalf("expected unary operator error")
	}
	if !strings.Contains(err.Error(), "unsupported unary operator '?'") {
		t.Fatalf("error = %v", err)
	}

	l = &lowerer{}
	_, err = l.lowerExpr(&frontend.BinaryExpr{
		At:    pos,
		Op:    frontend.TokenQuestion,
		Left:  &frontend.NumberExpr{At: pos, Value: 1},
		Right: &frontend.NumberExpr{At: pos, Value: 2},
	})
	if err == nil {
		t.Fatalf("expected binary operator error")
	}
	if !strings.Contains(err.Error(), "unsupported binary operator '?'") {
		t.Fatalf("error = %v", err)
	}
}

func TestLowerInferUnsupportedExpressionNamesFeature(t *testing.T) {
	errExpr := &frontend.SomePatternExpr{
		At: frontend.Position{File: "infer.tetra", Line: 8, Col: 13},
	}
	_, err := (&lowerer{}).inferExprType(errExpr)
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, want := range []string{
		"infer.tetra:8:13",
		"unsupported expression kind",
		"SomePatternExpr",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}

// ---- verify_test.go ----

func TestVerifyProgramAcceptsLoweredControlFlow(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> Int?:
    if flag:
        return 42
    else:
        return none

func main() -> Int:
    var total: Int = 0
    for i in 0..<6:
        if i == 1:
            continue
        if i == 5:
            break
        total = total + i
    let value: Int? = maybe(true)
    match value:
    case some(x):
        return total + x
    case none:
        return total
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	if err := VerifyProgram(irProg); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
}

func TestLowerImplicitOptionalCallArgumentUsesCalleeParamSlots(t *testing.T) {
	src := []byte(`
func unwrap(value: Int?) -> Int:
    if let some(x) = value:
        return x
    else:
        return 0

func main() -> Int:
    return unwrap(41)
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := irProg.Funcs[irProg.MainIndex]
	for _, instr := range mainFn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == "unwrap" {
			if instr.ArgSlots != 2 || instr.RetSlots != 1 {
				t.Fatalf(
					"unwrap call ABI = args %d rets %d, want args 2 rets 1",
					instr.ArgSlots,
					instr.RetSlots,
				)
			}
			return
		}
	}
	t.Fatalf("main did not lower an unwrap call: %#v", mainFn.Instrs)
}

func TestVerifyFuncRejectsUnbalancedReturnStack(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_return",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "return expects 1 stack slots") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsEmptyReturningFunction(t *testing.T) {
	fn := ir.IRFunc{Name: "empty", ReturnSlots: 1}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "empty body cannot produce 1 return slots") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyProgramRejectsInvalidMainMetadata(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 1,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{Name: "main", Instrs: []ir.IRInstr{{Kind: ir.IRReturn}}},
		},
	}
	err := VerifyProgram(prog)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "main index 1 out of bounds") {
		t.Fatalf("error = %v", err)
	}
	diag, ok := frontend.DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != DiagnosticCodeIRVerifier || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestVerifyProgramRejectsDuplicateFunctionNames(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{Name: "main", Instrs: []ir.IRInstr{{Kind: ir.IRReturn}}},
			{Name: "main", Instrs: []ir.IRInstr{{Kind: ir.IRReturn}}},
		},
	}
	err := VerifyProgram(prog)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), `duplicate function name "main"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyProgramRejectsKnownFunctionCallSignatureMismatch(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRCall, Name: "helper", ArgSlots: 0, RetSlots: 1},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "helper",
				ParamSlots:  1,
				LocalSlots:  1,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadLocal, Local: 0},
					{Kind: ir.IRReturn},
				},
			},
		},
	}
	err := VerifyProgram(prog)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), `call "helper" ABI mismatch`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsInvalidSlotMetadata(t *testing.T) {
	tests := []struct {
		name string
		fn   ir.IRFunc
		want string
	}{
		{
			name: "negative_param_slots",
			fn:   ir.IRFunc{Name: "bad_param", ParamSlots: -1},
			want: "negative slot metadata",
		},
		{
			name: "negative_return_slots",
			fn:   ir.IRFunc{Name: "bad_return_slots", ReturnSlots: -1},
			want: "negative slot metadata",
		},
		{
			name: "params_exceed_locals",
			fn:   ir.IRFunc{Name: "bad_locals", ParamSlots: 2, LocalSlots: 1},
			want: "param slots 2 exceed locals 1",
		},
		{
			name: "owned_param_local_out_of_range",
			fn: ir.IRFunc{
				Name:       "bad_owned_param",
				LocalSlots: 1,
				OwnedParams: []ir.IROwnedParam{
					{
						Local:           1,
						LayoutID:        "layout:consume_param:bad_owned_param:raw",
						OwnershipDomain: ir.IROwnershipDomainHeap,
						ReleaseKind:     ir.IRReleaseKindLinuxMmap,
					},
				},
			},
			want: "owned param local 1 out of range",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyFunc(tt.fn)
			if err == nil {
				t.Fatalf("expected verifier error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestVerifyFuncRejectsUnknownBranchLabel(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_label",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: 99},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "unknown label 99") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsNegativeBranchLabel(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_negative_label",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: -1},
			{Kind: ir.IRLabel, Label: -1},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "negative label -1") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsLocalSlotOutOfBounds(t *testing.T) {
	pos := frontend.Position{File: "bad_lower.t4", Line: 7, Col: 5}
	fn := ir.IRFunc{
		Name:       "bad_local",
		LocalSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 1, Pos: pos},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "local slot 1 out of bounds") {
		t.Fatalf("error = %v", err)
	}
	diag, ok := frontend.DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != DiagnosticCodeIRVerifier || diag.File != "bad_lower.t4" || diag.Line != 7 ||
		diag.Column != 5 {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestVerifyFuncRejectsNegativeGlobalSlotOperands(t *testing.T) {
	tests := []struct {
		name string
		fn   ir.IRFunc
	}{
		{
			name: "load_global",
			fn: ir.IRFunc{
				Name:        "bad_load_global",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRLoadGlobal, Local: -1},
					{Kind: ir.IRReturn},
				},
			},
		},
		{
			name: "store_global",
			fn: ir.IRFunc{
				Name: "bad_store_global",
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 7},
					{Kind: ir.IRStoreGlobal, Local: -1},
					{Kind: ir.IRReturn},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyFunc(tt.fn)
			if err == nil {
				t.Fatalf("expected verifier error")
			}
			if !strings.Contains(err.Error(), "global slot -1 out of bounds") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestLowerRunsProgramLevelVerifier(t *testing.T) {
	prog, err := frontend.Parse([]byte("func main() -> Int:\n    return 0\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	checked.MainIndex = len(checked.Funcs) + 1

	_, err = Lower(checked)
	if err == nil {
		t.Fatalf("expected program verifier error")
	}
	diag, ok := frontend.DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != DiagnosticCodeIRVerifier {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if !strings.Contains(diag.Message, "main index") {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestVerifyFuncRejectsCallStackUnderflow(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_call",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRCall, Name: "callee", ArgSlots: 2, RetSlots: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "stack underflow") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsInconsistentBranchStack(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_join",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "inconsistent stack height") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsUnknownInstructionKind(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_kind",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRInstrKind(999)},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "unknown instruction kind 999") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsNegativeCallABISlots(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_call_abi",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRCall, Name: "callee", ArgSlots: -1, RetSlots: 0},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), `call "callee" has negative ABI slots`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsKnownRuntimeCallSignatureMismatch(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_runtime_call",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32},
			{Kind: ir.IRCall, Name: "__tetra_actor_spawn", ArgSlots: 0, RetSlots: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), `runtime call "__tetra_actor_spawn" ABI mismatch`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsTypedTaskRuntimeCallSignatureMismatch(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_typed_task_runtime_call",
		ReturnSlots: 5,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRConstI32},
			{Kind: ir.IRCall, Name: "__tetra_task_join_typed_5", ArgSlots: 5, RetSlots: 5},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), `runtime call "__tetra_task_join_typed_5" ABI mismatch`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncAcceptsKnownRuntimeCallSignature(t *testing.T) {
	actorSlots := runtimeabi.ActorHandleABI().RefSlots
	fn := ir.IRFunc{
		Name:        "runtime_call",
		ReturnSlots: actorSlots,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32},
			{Kind: ir.IRCall, Name: "__tetra_actor_spawn", ArgSlots: 1, RetSlots: actorSlots},
			{Kind: ir.IRReturn},
		},
	}
	if err := VerifyFunc(fn); err != nil {
		t.Fatalf("VerifyFunc: %v", err)
	}
}

func TestVerifyFuncRejectsMissingNamedOperands(t *testing.T) {
	tests := []struct {
		name string
		fn   ir.IRFunc
		want string
	}{
		{
			name: "call",
			fn: ir.IRFunc{
				Name: "bad_call_name",
				Instrs: []ir.IRInstr{
					{Kind: ir.IRCall, ArgSlots: 0, RetSlots: 0},
					{Kind: ir.IRReturn},
				},
			},
			want: "call is missing target name",
		},
		{
			name: "symbol_address",
			fn: ir.IRFunc{
				Name:        "bad_symbol_name",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRSymAddr},
					{Kind: ir.IRReturn},
				},
			},
			want: "symbol address is missing name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyFunc(tt.fn)
			if err == nil {
				t.Fatalf("expected verifier error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestVerifyFuncRejectsInvalidRawSliceElementShift(t *testing.T) {
	for _, shift := range []int32{-1, 3} {
		fn := ir.IRFunc{
			Name:        "bad_raw_slice_shift",
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRCapMem},
				{Kind: ir.IRRawSliceFromParts, Imm: shift},
				{Kind: ir.IRReturn},
			},
		}
		err := VerifyFunc(fn)
		if err == nil {
			t.Fatalf("shift %d: expected verifier error", shift)
		}
		if !strings.Contains(err.Error(), "raw slice element-size shift") {
			t.Fatalf("shift %d: error = %v, want raw slice element-size shift", shift, err)
		}
	}
}

func TestVerifyFuncAcceptsPolicyGuardShape(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "guarded",
		ParamSlots:  1,
		LocalSlots:  2,
		ReturnSlots: 1,
		Policy: ir.IRPolicy{
			HasBudget:    true,
			Budget:       3,
			BudgetLocal:  1,
			HasConsent:   true,
			ConsentLocal: 0,
			FailLabel:    1,
		},
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: consentTokenRuntimeSentinel},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGeI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRCall, Name: "callee", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
	if err := VerifyFunc(fn); err != nil {
		t.Fatalf("VerifyFunc: %v", err)
	}
}

func TestVerifyFuncRejectsMissingBudgetGuardBeforeChargedInstruction(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_budget_guard",
		LocalSlots:  1,
		ReturnSlots: 1,
		Policy: ir.IRPolicy{
			HasBudget:   true,
			Budget:      1,
			BudgetLocal: 0,
			FailLabel:   1,
		},
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRCall, Name: "callee", ArgSlots: 0, RetSlots: 1},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "missing budget guard") {
		t.Fatalf("error = %v", err)
	}
}

func TestStackEffectCoversEveryIRInstrKind(t *testing.T) {
	for kind := ir.IRWrite; kind < ir.IRInstrKindCount; kind++ {
		_, _, known := stackEffect(ir.IRInstr{Kind: kind, ArgSlots: 1, RetSlots: 1})
		if !known {
			t.Fatalf("missing stack effect for IR kind %d", kind)
		}
	}
}

func TestAtomicPointerExchangeAndFenceStackEffects(t *testing.T) {
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicLoadPtr}); !known || pop != 2 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicLoadPtr stack effect pop=%d push=%d known=%v, want pop2 push1",
			pop,
			push,
			known,
		)
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicStorePtr}); !known || pop != 3 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicStorePtr stack effect pop=%d push=%d known=%v, want pop3 push1",
			pop,
			push,
			known,
		)
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicExchangePtr}); !known ||
		pop != 3 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicExchangePtr stack effect pop=%d push=%d known=%v, want pop3 push1",
			pop,
			push,
			known,
		)
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicFetchAddPtr}); !known ||
		pop != 3 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicFetchAddPtr stack effect pop=%d push=%d known=%v, want pop3 push1",
			pop,
			push,
			known,
		)
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicFetchSubPtr}); !known ||
		pop != 3 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicFetchSubPtr stack effect pop=%d push=%d known=%v, want pop3 push1",
			pop,
			push,
			known,
		)
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicFetchAndPtr}); !known ||
		pop != 3 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicFetchAndPtr stack effect pop=%d push=%d known=%v, want pop3 push1",
			pop,
			push,
			known,
		)
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicFetchOrPtr}); !known ||
		pop != 3 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicFetchOrPtr stack effect pop=%d push=%d known=%v, want pop3 push1",
			pop,
			push,
			known,
		)
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicFetchXorPtr}); !known ||
		pop != 3 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicFetchXorPtr stack effect pop=%d push=%d known=%v, want pop3 push1",
			pop,
			push,
			known,
		)
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicCompareExchangePtr}); !known ||
		pop != 4 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicCompareExchangePtr stack effect pop=%d push=%d known=%v, want pop4 push1",
			pop,
			push,
			known,
		)
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicFenceSeqCst}); !known ||
		pop != 0 ||
		push != 0 {
		t.Fatalf(
			"IRAtomicFenceSeqCst stack effect pop=%d push=%d known=%v, want pop0 push0",
			pop,
			push,
			known,
		)
	}
	for _, kind := range []ir.IRInstrKind{
		ir.IRAtomicFenceRelaxed,
		ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease,
		ir.IRAtomicFenceAcqRel,
	} {
		if pop, push, known := stackEffect(ir.IRInstr{Kind: kind}); !known || pop != 0 ||
			push != 0 {
			t.Fatalf(
				"%v stack effect pop=%d push=%d known=%v, want pop0 push0",
				kind,
				pop,
				push,
				known,
			)
		}
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicLoadI32}); !known || pop != 2 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicLoadI32 stack effect pop=%d push=%d known=%v, want pop2 push1",
			pop,
			push,
			known,
		)
	}
	for _, kind := range []ir.IRInstrKind{
		ir.IRAtomicStoreI32,
		ir.IRAtomicExchangeI32,
		ir.IRAtomicFetchAddI32,
		ir.IRAtomicFetchSubI32,
		ir.IRAtomicFetchAndI32,
		ir.IRAtomicFetchOrI32,
		ir.IRAtomicFetchXorI32,
	} {
		if pop, push, known := stackEffect(ir.IRInstr{Kind: kind}); !known || pop != 3 ||
			push != 1 {
			t.Fatalf(
				"%v stack effect pop=%d push=%d known=%v, want pop3 push1",
				kind,
				pop,
				push,
				known,
			)
		}
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicCompareExchangeI32}); !known ||
		pop != 4 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicCompareExchangeI32 stack effect pop=%d push=%d known=%v, want pop4 push1",
			pop,
			push,
			known,
		)
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicLoadI64}); !known || pop != 2 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicLoadI64 stack effect pop=%d push=%d known=%v, want pop2 push1",
			pop,
			push,
			known,
		)
	}
	for _, kind := range []ir.IRInstrKind{
		ir.IRAtomicStoreI64,
		ir.IRAtomicExchangeI64,
		ir.IRAtomicFetchAddI64,
		ir.IRAtomicFetchSubI64,
		ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64,
		ir.IRAtomicFetchXorI64,
	} {
		if pop, push, known := stackEffect(ir.IRInstr{Kind: kind}); !known || pop != 3 ||
			push != 1 {
			t.Fatalf(
				"%v stack effect pop=%d push=%d known=%v, want pop3 push1",
				kind,
				pop,
				push,
				known,
			)
		}
	}
	if pop, push, known := stackEffect(ir.IRInstr{Kind: ir.IRAtomicCompareExchangeI64}); !known ||
		pop != 4 ||
		push != 1 {
		t.Fatalf(
			"IRAtomicCompareExchangeI64 stack effect pop=%d push=%d known=%v, want pop4 push1",
			pop,
			push,
			known,
		)
	}
	for _, tc := range []struct {
		load ir.IRInstrKind
		cas  ir.IRInstrKind
		ops  []ir.IRInstrKind
	}{
		{
			load: ir.IRAtomicLoadI8,
			cas:  ir.IRAtomicCompareExchangeI8,
			ops: []ir.IRInstrKind{
				ir.IRAtomicStoreI8,
				ir.IRAtomicExchangeI8,
				ir.IRAtomicFetchAddI8,
				ir.IRAtomicFetchSubI8,
				ir.IRAtomicFetchAndI8,
				ir.IRAtomicFetchOrI8,
				ir.IRAtomicFetchXorI8,
			},
		},
		{
			load: ir.IRAtomicLoadI16,
			cas:  ir.IRAtomicCompareExchangeI16,
			ops: []ir.IRInstrKind{
				ir.IRAtomicStoreI16,
				ir.IRAtomicExchangeI16,
				ir.IRAtomicFetchAddI16,
				ir.IRAtomicFetchSubI16,
				ir.IRAtomicFetchAndI16,
				ir.IRAtomicFetchOrI16,
				ir.IRAtomicFetchXorI16,
			},
		},
	} {
		if pop, push, known := stackEffect(ir.IRInstr{Kind: tc.load}); !known || pop != 2 ||
			push != 1 {
			t.Fatalf(
				"%v stack effect pop=%d push=%d known=%v, want pop2 push1",
				tc.load,
				pop,
				push,
				known,
			)
		}
		for _, kind := range tc.ops {
			if pop, push, known := stackEffect(ir.IRInstr{Kind: kind}); !known || pop != 3 ||
				push != 1 {
				t.Fatalf(
					"%v stack effect pop=%d push=%d known=%v, want pop3 push1",
					kind,
					pop,
					push,
					known,
				)
			}
		}
		if pop, push, known := stackEffect(ir.IRInstr{Kind: tc.cas}); !known || pop != 4 ||
			push != 1 {
			t.Fatalf(
				"%v stack effect pop=%d push=%d known=%v, want pop4 push1",
				tc.cas,
				pop,
				push,
				known,
			)
		}
	}
}

func TestBudgetChargeModelIsExplicit(t *testing.T) {
	charged := map[ir.IRInstrKind]int32{
		ir.IRWrite:                    1,
		ir.IRCall:                     1,
		ir.IRAllocBytes:               1,
		ir.IRMakeSliceU8:              1,
		ir.IRMakeSliceU16:             1,
		ir.IRMakeSliceI32:             1,
		ir.IRStackSliceU8:             1,
		ir.IRStackSliceU16:            1,
		ir.IRStackSliceI32:            1,
		ir.IRRegionEnter:              1,
		ir.IRRegionMakeSliceU8:        1,
		ir.IRRegionMakeSliceU16:       1,
		ir.IRRegionMakeSliceI32:       1,
		ir.IRRegionReset:              1,
		ir.IRRawSliceFromParts:        1,
		ir.IRSliceWindow:              1,
		ir.IRSlicePrefix:              1,
		ir.IRSliceSuffix:              1,
		ir.IRIndexLoadI32:             1,
		ir.IRIndexLoadI32Unchecked:    1,
		ir.IRIndexStoreI32:            1,
		ir.IRIndexLoadU8:              1,
		ir.IRIndexLoadU8Unchecked:     1,
		ir.IRIndexStoreU8:             1,
		ir.IRIndexLoadU16:             1,
		ir.IRIndexLoadU16Unchecked:    1,
		ir.IRIndexStoreU16:            1,
		ir.IRIslandNew:                1,
		ir.IRIslandMakeSliceU8:        1,
		ir.IRIslandMakeSliceU16:       1,
		ir.IRIslandMakeSliceI32:       1,
		ir.IRIslandFree:               1,
		ir.IRIslandReset:              1,
		ir.IRDropOwned:                1,
		ir.IRReleaseAllocation:        1,
		ir.IRCapIO:                    1,
		ir.IRCapMem:                   1,
		ir.IRMemReadI32:               1,
		ir.IRMemWriteI32:              1,
		ir.IRMemReadU8:                1,
		ir.IRMemWriteU8:               1,
		ir.IRMemReadPtr:               1,
		ir.IRMemWritePtr:              1,
		ir.IRMemWriteArchPtr:          1,
		ir.IRMemReadI32Offset:         1,
		ir.IRMemWriteI32Offset:        1,
		ir.IRMemReadU8Offset:          1,
		ir.IRMemWriteU8Offset:         1,
		ir.IRMemReadPtrOffset:         1,
		ir.IRMemWritePtrOffset:        1,
		ir.IRMemWriteArchPtrOffset:    1,
		ir.IRPtrAdd:                   1,
		ir.IRMmioReadI32:              1,
		ir.IRMmioWriteI32:             1,
		ir.IRSymAddr:                  1,
		ir.IRCtxSwitch:                1,
		ir.IRAtomicLoadPtr:            1,
		ir.IRAtomicStorePtr:           1,
		ir.IRAtomicExchangePtr:        1,
		ir.IRAtomicFetchAddPtr:        1,
		ir.IRAtomicFetchSubPtr:        1,
		ir.IRAtomicFetchAndPtr:        1,
		ir.IRAtomicFetchOrPtr:         1,
		ir.IRAtomicFetchXorPtr:        1,
		ir.IRAtomicCompareExchangePtr: 1,
		ir.IRAtomicFenceSeqCst:        1,
		ir.IRAtomicFenceRelaxed:       1,
		ir.IRAtomicFenceAcquire:       1,
		ir.IRAtomicFenceRelease:       1,
		ir.IRAtomicFenceAcqRel:        1,
		ir.IRAtomicLoadI32:            1,
		ir.IRAtomicStoreI32:           1,
		ir.IRAtomicExchangeI32:        1,
		ir.IRAtomicCompareExchangeI32: 1,
		ir.IRAtomicFetchAddI32:        1,
		ir.IRAtomicFetchSubI32:        1,
		ir.IRAtomicFetchAndI32:        1,
		ir.IRAtomicFetchOrI32:         1,
		ir.IRAtomicFetchXorI32:        1,
		ir.IRAtomicLoadI64:            1,
		ir.IRAtomicStoreI64:           1,
		ir.IRAtomicExchangeI64:        1,
		ir.IRAtomicCompareExchangeI64: 1,
		ir.IRAtomicFetchAddI64:        1,
		ir.IRAtomicFetchSubI64:        1,
		ir.IRAtomicFetchAndI64:        1,
		ir.IRAtomicFetchOrI64:         1,
		ir.IRAtomicFetchXorI64:        1,
		ir.IRAtomicLoadI8:             1,
		ir.IRAtomicStoreI8:            1,
		ir.IRAtomicExchangeI8:         1,
		ir.IRAtomicCompareExchangeI8:  1,
		ir.IRAtomicFetchAddI8:         1,
		ir.IRAtomicFetchSubI8:         1,
		ir.IRAtomicFetchAndI8:         1,
		ir.IRAtomicFetchOrI8:          1,
		ir.IRAtomicFetchXorI8:         1,
		ir.IRAtomicLoadI16:            1,
		ir.IRAtomicStoreI16:           1,
		ir.IRAtomicExchangeI16:        1,
		ir.IRAtomicCompareExchangeI16: 1,
		ir.IRAtomicFetchAddI16:        1,
		ir.IRAtomicFetchSubI16:        1,
		ir.IRAtomicFetchAndI16:        1,
		ir.IRAtomicFetchOrI16:         1,
		ir.IRAtomicFetchXorI16:        1,
	}
	for kind, want := range charged {
		got, ok := budgetChargeForInstr(kind)
		if !ok {
			t.Fatalf("kind %v missing from budget charge model", kind)
		}
		if got != want {
			t.Fatalf("kind %v cost = %d, want %d", kind, got, want)
		}
	}

	uncharged := []ir.IRInstrKind{
		ir.IRStrLit,
		ir.IRConstI32,
		ir.IRLoadLocal,
		ir.IRStoreLocal,
		ir.IRLoadGlobal,
		ir.IRStoreGlobal,
		ir.IRAddI32,
		ir.IRSubI32,
		ir.IRNegI32,
		ir.IRCmpEqI32,
		ir.IRCmpLtI32,
		ir.IRMulI32,
		ir.IRDivI32,
		ir.IRModI32,
		ir.IRCmpGtI32,
		ir.IRCmpGeI32,
		ir.IRCmpLeI32,
		ir.IRCmpNeI32,
		ir.IRLabel,
		ir.IRJmp,
		ir.IRJmpIfZero,
		ir.IRReturn,
	}
	classified := make(map[ir.IRInstrKind]string, len(charged)+len(uncharged))
	for kind := range charged {
		classified[kind] = "charged"
	}
	for _, kind := range uncharged {
		if previous, exists := classified[kind]; exists {
			t.Fatalf("kind %d classified as both %s and uncharged", kind, previous)
		}
		classified[kind] = "uncharged"
		if got, ok := budgetChargeForInstr(kind); ok {
			t.Fatalf("kind %v unexpectedly charged with cost %d", kind, got)
		}
	}
	for kind := ir.IRWrite; kind < ir.IRInstrKindCount; kind++ {
		if _, ok := classified[kind]; !ok {
			t.Fatalf("missing budget charge classification for IR kind %d", kind)
		}
	}
}

func TestVerifyFuncRejectsMissingBudgetGuardBeforeIndexedAccess(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_index_budget_guard",
		LocalSlots:  1,
		ReturnSlots: 1,
		Policy: ir.IRPolicy{
			HasBudget:   true,
			Budget:      1,
			BudgetLocal: 0,
			FailLabel:   1,
		},
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 100},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadI32},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "missing budget guard") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsMalformedConsentGuardShape(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "bad_consent_guard",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Policy: ir.IRPolicy{
			HasConsent:   true,
			ConsentLocal: 0,
			FailLabel:    1,
		},
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpNeI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "malformed consent guard") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsUnreachableUnknownInstructionKind(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_unreachable_kind",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRInstrKind(999)},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "instr 1: unknown instruction kind 999") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsUnreachableLinearStackUnderflow(t *testing.T) {
	fn := ir.IRFunc{
		Name: "bad_unreachable_stack",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	if !strings.Contains(err.Error(), "linear stack underflow") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyFuncRejectsDuplicateBranchLabelsWithStableDiagnostic(t *testing.T) {
	pos := frontend.Position{File: "duplicate_label.t4", Line: 4, Col: 9}
	fn := ir.IRFunc{
		Name: "bad_duplicate_label",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLabel, Label: 1, Pos: pos},
			{Kind: ir.IRReturn},
		},
	}
	err := VerifyFunc(fn)
	if err == nil {
		t.Fatalf("expected duplicate label verifier error")
	}
	if !strings.Contains(err.Error(), "duplicate label 1") {
		t.Fatalf("error = %v", err)
	}
	diag, ok := frontend.DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != DiagnosticCodeIRVerifier || diag.File != "duplicate_label.t4" ||
		diag.Line != 4 ||
		diag.Column != 9 ||
		diag.Hint == "" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestVerifyProgramAcceptsRepresentativeLoweringFamilies(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "typed_errors",
			src: `
enum E:
    case bad

func fail(flag: Bool) -> Int throws E:
    if flag:
        return 7
    throw E.bad

func caller(flag: Bool) -> Int throws E:
    return try fail(flag)

func main() -> Int:
    return 0
`,
		},
		{
			name: "tasks",
			src: `
func worker() -> Int:
    return 3

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
		},
		{
			name: "actors",
			src: `
func worker() -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _: Int = core.send(peer, 1)
    return 0
`,
		},
		{
			name: "unsafe_budget_guards",
			src: `
func main() -> Int
uses alloc, budget, capability, mem
budget(32):
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 9, mem)
        return core.load_i32(p, mem)
    return 0
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := frontend.Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			checked, err := semantics.Check(prog)
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			irProg, err := Lower(checked)
			if err != nil {
				t.Fatalf("Lower: %v", err)
			}
			if err := VerifyProgram(irProg); err != nil {
				t.Fatalf("VerifyProgram: %v", err)
			}
		})
	}
}
