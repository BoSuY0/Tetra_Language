package lower

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

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
		t.Fatalf("function-typed alias should materialize add1 once and copy fnptr slots: %#v", fn.Instrs)
	}
	if countCallableKind(fn.Instrs, ir.IRLoadLocal) < semantics.FnPtrSlotCount || countCallableKind(fn.Instrs, ir.IRStoreLocal) < 2*semantics.FnPtrSlotCount {
		t.Fatalf("function-typed alias did not copy the %d-slot fnptr value: %#v", semantics.FnPtrSlotCount, fn.Instrs)
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
		t.Fatalf("nine-capture callable call should read 9 heap env slots, got %d: %#v", got, fn.Instrs)
	}
	if got := countInstr(fn.Instrs, ir.IRPtrAdd, ""); got != 0 {
		t.Fatalf("nine-capture callable heap env should use base+offset access, got %d ptr_add instructions: %#v", got, fn.Instrs)
	}
	calls := 0
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.ArgSlots == 10 && instr.RetSlots == 1 {
			calls++
		}
	}
	if calls != 1 {
		t.Fatalf("nine-capture callable should call closure with explicit arg plus 9 env slots: %#v", fn.Instrs)
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
		t.Fatalf("direct named callback argument did not lower to IRSymAddr(add1): %#v", mainFn.Instrs)
	}
	if countCall(apply.Instrs, "add1", 1, 1) != 1 {
		t.Fatalf("single-target callback body did not lower to direct IRCall(add1): %#v", apply.Instrs)
	}
	if countInstr(apply.Instrs, ir.IRSymAddr, "add1") != 0 {
		t.Fatalf("single-target callback body should not emit a dynamic branch IRSymAddr: %#v", apply.Instrs)
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

	if countInstr(apply.Instrs, ir.IRSymAddr, "add1") != 1 || countInstr(apply.Instrs, ir.IRSymAddr, "add2") != 1 {
		t.Fatalf("multi-target callback body did not compare against both target symbols: %#v", apply.Instrs)
	}
	if countCallableKind(apply.Instrs, ir.IRCmpEqI32) < 2 || countCallableKind(apply.Instrs, ir.IRJmpIfZero) < 2 {
		t.Fatalf("multi-target callback body lacks symbol compare/branch sequence: %#v", apply.Instrs)
	}
	if countCall(apply.Instrs, "add1", 1, 1) != 1 || countCall(apply.Instrs, "add2", 1, 1) != 1 {
		t.Fatalf("multi-target callback body did not lower both direct target calls: %#v", apply.Instrs)
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

	if countInstr(mainFn.Instrs, ir.IRSymAddr, "add1") < 1 || countInstr(mainFn.Instrs, ir.IRSymAddr, "add2") < 2 {
		t.Fatalf("mutable global callable did not preserve both assignment and dispatch targets: %#v", mainFn.Instrs)
	}
	if countCallableKind(mainFn.Instrs, ir.IRStoreGlobal) < semantics.FnPtrSlotCount || countCallableKind(mainFn.Instrs, ir.IRLoadGlobal) < semantics.FnPtrSlotCount {
		t.Fatalf("mutable global callable did not store/load %d-slot fnptr value: %#v", semantics.FnPtrSlotCount, mainFn.Instrs)
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
		t.Fatalf("mutable global callback argument did not load current fnptr slots: %#v", mainFn.Instrs)
	}
	if countInstr(mainFn.Instrs, ir.IRSymAddr, "add1") > 1 {
		t.Fatalf("mutable global callback argument was rewritten to static initial target: %#v", mainFn.Instrs)
	}
	if countCall(apply.Instrs, "add1", 1, 1) != 1 || countCall(apply.Instrs, "add2", 1, 1) != 1 {
		t.Fatalf("callee callback target set did not include both mutable global targets: %#v", apply.Instrs)
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
		t.Fatalf("global assignment did not store %d fnptr slots: %#v", semantics.FnPtrSlotCount, mainFn.Instrs)
	}
	if countCallableKind(mainFn.Instrs, ir.IRLoadLocal) < semantics.FnPtrSlotCount {
		t.Fatalf("global assignment did not copy fnptr slots from the mutable local: %#v", mainFn.Instrs)
	}
	if countCall(mainFn.Instrs, "identity", 1, 1) != 1 {
		t.Fatalf("mutable global dispatch lost initial identity target: %#v", mainFn.Instrs)
	}
	if countCallableClosureCalls(mainFn.Instrs) != 1 {
		t.Fatalf("mutable global dispatch lost captured return target: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedReturnMutableLocalGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured return target from configure assignment: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedReturnStructFieldGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured struct-field target from configure assignment: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedStructFieldGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured struct-field direct closure target from configure assignment: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedReturnWholeStructGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured whole-struct target from configure assignment: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedWholeStructGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured whole-struct direct closure target from configure assignment: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedReturnWholeNestedStructGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured whole-nested-struct target from configure assignment: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedWholeNestedStructGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured whole-nested-struct direct closure target from configure assignment: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedReturnEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured enum-payload target from configure assignment: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured enum-payload direct closure target from configure assignment: %#v", mainFn.Instrs)
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
		t.Fatalf("mutable global dispatch did not receive captured whole-enum direct closure target from configure assignment: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedReturnedStructEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured returned-struct enum-payload target from configure assignment: %#v", mainFn.Instrs)
	}
}

func TestLowerCallableCapturedReturnedEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR(t *testing.T) {
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
		t.Fatalf("mutable global dispatch did not receive captured returned-enum payload target from configure assignment: %#v", mainFn.Instrs)
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
		t.Fatalf("string-return callback branches did not preserve two return slots: %#v", apply.Instrs)
	}
	if countStoresAfterCalls(apply.Instrs, map[string]bool{"word1": true, "word2": true}) < 4 {
		t.Fatalf("string-return callback branches did not store two result slots per target: %#v", apply.Instrs)
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
		t.Fatalf("struct-return callback branches did not preserve two return slots: %#v", apply.Instrs)
	}
	if countStoresAfterCalls(apply.Instrs, map[string]bool{"pair1": true, "pair2": true}) < 4 {
		t.Fatalf("struct-return callback branches did not store two result slots per target: %#v", apply.Instrs)
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
		t.Fatalf("whole-struct reassignment did not preserve both field-call targets: %#v", mainFn.Instrs)
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
		t.Fatalf("struct-valued field reassignment did not preserve both nested field-call targets: %#v", mainFn.Instrs)
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
		t.Fatalf("whole nested-struct reassignment did not preserve both nested field-call targets: %#v", mainFn.Instrs)
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
	if err == nil || !strings.Contains(err.Error(), "unknown callback target 'missing_callback_target'") {
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
		if instr.Kind == ir.IRCall && instr.Name == name && instr.ArgSlots == argSlots && instr.RetSlots == retSlots {
			count++
		}
	}
	return count
}

func requireContiguousArgumentLoadsBeforeCall(t *testing.T, instrs []ir.IRInstr, name string, argSlots int) {
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
				t.Fatalf("call %s arg %d is loaded by %v, want IRLoadLocal: %#v", name, slot+1, load.Kind, instrs)
			}
			if slot == 0 {
				base = load.Local
				continue
			}
			if load.Local != base+slot {
				t.Fatalf("call %s arg loads are locals %d then %d, want contiguous scratch locals: %#v", name, base, load.Local, instrs)
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
