package lower

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
)

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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf("while unchecked load proof id = %q, want proof:while prefix; instrs=%#v", got, fn.Instrs)
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
		t.Fatalf("compound increment while loop still contains checked i32 index load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf("compound increment unchecked load proof id = %q, want proof:while prefix; instrs=%#v", got, fn.Instrs)
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
		t.Fatalf("commuted increment while loop still contains checked i32 index load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf("commuted increment unchecked load proof id = %q, want proof:while prefix; instrs=%#v", got, fn.Instrs)
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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf("const step-one unchecked load proof id = %q, want proof:while prefix; instrs=%#v", got, fn.Instrs)
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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf("alias while unchecked load proof id = %q, want proof:while prefix; instrs=%#v", got, fn.Instrs)
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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf("while <= unchecked load proof id = %q, want proof:while prefix; instrs=%#v", got, fn.Instrs)
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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf("while != len unchecked load proof id = %q, want proof:while prefix; instrs=%#v", got, fn.Instrs)
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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf("start/end alias unchecked load proof id = %q, want proof:while prefix; instrs=%#v", got, fn.Instrs)
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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf("view end alias unchecked load proof id = %q, want proof:while prefix; instrs=%#v", got, fn.Instrs)
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
		t.Fatalf("callback inout call should keep checked i32 load after unknown mutable boundary: %#v", fn.Instrs)
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
		t.Fatalf("if unchecked load proof id = %q, want proof:if prefix; instrs=%#v", got, fn.Instrs)
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
		t.Fatalf("dominating if guard with explicit lower bound should remove checked load: %#v", fn.Instrs)
	}
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:if:") {
		t.Fatalf("if lower-bound unchecked load proof id = %q, want proof:if prefix; instrs=%#v", got, fn.Instrs)
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
		t.Fatalf("if upper guard without lower bound unexpectedly removed bounds check: %#v", fn.Instrs)
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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:while:") {
		t.Fatalf("nested while unchecked load proof id = %q, want proof:while prefix; instrs=%#v", got, fn.Instrs)
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
		t.Fatalf("invalid String alias while loop unexpectedly removed bounds check: %#v", fn.Instrs)
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
		t.Fatalf("branch-joined invalid alias while loop should keep checked u8 load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf("branch-joined invalid alias while loop unexpectedly removed bounds check: %#v", fn.Instrs)
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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:for-collection") {
		t.Fatalf("view-chain for proof id = %q, want proof:for-collection prefix; instrs=%#v", got, fn.Instrs)
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
		t.Fatalf("invalid intermediate String view chain should keep checked u8 index load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf("invalid intermediate String view chain unexpectedly contains unchecked u8 index load: %#v", fn.Instrs)
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
		t.Fatalf("raw-derived view chain unexpectedly contains unchecked u8 index load: %#v", fn.Instrs)
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
		t.Fatalf("invalid String view alias for-loop should keep checked u8 index load: %#v", fn.Instrs)
	}
	if countInstrKind(fn, ir.IRIndexLoadU8Unchecked) != 0 {
		t.Fatalf("invalid String view alias for-loop unexpectedly contains unchecked u8 index load: %#v", fn.Instrs)
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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:copy-loop:") {
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
	if got := firstProofID(fn, ir.IRIndexLoadI32Unchecked); !strings.HasPrefix(got, "proof:copy-loop:") {
		t.Fatalf("copy_into loop proof id = %q, want proof:copy-loop prefix; instrs=%#v", got, fn.Instrs)
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
		t.Fatalf("invalid String view loop unexpectedly contains unchecked u8 index load: %#v", fn.Instrs)
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
