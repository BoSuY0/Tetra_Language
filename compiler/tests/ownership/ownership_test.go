package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/testkit"
	"tetra_language/compiler/tests/ownership/testhelpers"
)

func TestOwnershipMarkersParseAndFormat(t *testing.T) {
	src := []byte(`
func mix(a: borrow Int, b: inout Int, c: consume Int, cb: borrow fn(borrow Int, inout Int, consume Int) -> Int) -> Int:
    return cb(a) + b + c
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	params := prog.Funcs[0].Params
	if params[0].Ownership != "borrow" || params[1].Ownership != "inout" ||
		params[2].Ownership != "consume" ||
		params[3].Ownership != "borrow" {
		t.Fatalf(
			"ownership markers = %q/%q/%q/%q",
			params[0].Ownership,
			params[1].Ownership,
			params[2].Ownership,
			params[3].Ownership,
		)
	}
	if got := strings.Join(params[3].Type.ParamOwnership, ","); got != "borrow,inout,consume" {
		t.Fatalf("function type ownership markers = %q", got)
	}
	formatted, err := compiler.FormatSource(src, "ownership.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	wantParams := ("a: borrow Int, b: inout Int, c: consume Int, cb: borrow " +
		"fn(borrow Int, inout Int, consume Int) -> Int")
	if !strings.Contains(string(formatted), wantParams) {
		t.Fatalf("formatted source missing markers:\n%s", string(formatted))
	}
	twice, err := compiler.FormatSource(formatted, "ownership.tetra")
	if err != nil {
		t.Fatalf("FormatSource twice: %v", err)
	}
	if string(twice) != string(formatted) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(formatted), string(twice))
	}
}

func TestOwnershipInoutParamIsMutable(t *testing.T) {
	src := []byte(`
func bump(x: inout Int) -> Int:
    x = x + 1
    return x

func main() -> Int:
    var a: Int = 1
    return bump(a)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestMemoryIdealV0SequentialInoutAndCopyThenInout(t *testing.T) {
	testkit.RequireCheckOK(t, `
func fill(xs: inout []u8) -> Int
uses mem:
    xs[0] = xs[0] + 1
    return xs[0]

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(1)
    xs[0] = 1
    let first: Int = fill(xs)
    let second: Int = fill(xs)
    var owned: []u8 = xs.copy()
    return first + second + fill(owned)
`)
}

func TestOwnershipInoutRequiresMutableLocal(t *testing.T) {
	src := []byte(`
func bump(x: inout Int) -> Int:
    x = x + 1
    return x

func main() -> Int:
    let a: Int = 1
    return bump(a)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected inout argument mutability error")
	}
	if !strings.Contains(err.Error(), "inout argument") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipBorrowParamCannotMutate(t *testing.T) {
	src := []byte(`
func bump(x: borrow Int) -> Int:
    x = x + 1
    return x

func main() -> Int:
    return bump(1)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected borrow mutation error")
	}
	if !strings.Contains(err.Error(), "cannot assign to val 'x'") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsReturningBorrowedParam(t *testing.T) {
	src := []byte(`
func leak(x: borrow []u8) -> []u8:
    return x

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected escaping borrowed local error")
	}
	if !strings.Contains(
		err.Error(),
		"borrowed slice return requires '-> borrow []u8' or '.copy()'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsReturningBorrowedPtrParam(t *testing.T) {
	src := []byte(`
func leak(x: borrow ptr) -> ptr:
    return x

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected escaping borrowed ptr error")
	}
	if !strings.Contains(err.Error(), "borrowed local 'x' cannot escape via return") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsReturningBorrowedPtrAlias(t *testing.T) {
	src := []byte(`
func leak(x: borrow ptr) -> ptr:
    let y: ptr = x
    return y

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected escaping borrowed ptr alias error")
	}
	if !strings.Contains(err.Error(), "borrowed local 'x' cannot escape via return") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsBorrowEscapeViaAliasReturn(t *testing.T) {
	src := []byte(`
func leak(x: borrow []u8) -> []u8:
    let y: []u8 = x
    return y

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected escaping borrowed alias error")
	}
	if !strings.Contains(
		err.Error(),
		"borrowed slice return requires '-> borrow []u8' or '.copy()'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsBorrowEscapeViaFixedArrayAliasReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(x: borrow [2]Int) -> [2]Int:
    let y: [2]Int = x
    return y

func main() -> Int:
    return 0
	`, "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsCrossModuleBorrowEscapeViaFixedArrayAliasReturn(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func leak(x: borrow [2]Int) -> [2]Int:
    let y: [2]Int = x
    return y
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"lib/leak.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestOwnershipRejectsBorrowEscapeViaStringAliasReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(x: borrow str) -> str:
    let y: str = x
    return y

func main() -> Int:
    return 0
	`, "borrowed String return requires '-> borrow String' or '.copy()'")
}

func TestOwnershipRejectsBorrowedSliceGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var leaked: []u8

func leak(x: borrow []u8) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedStringGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var leaked: str = ""

func leak(x: borrow str) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedFixedArrayGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct ArrayBox:
    items: [2]Int

var leaked: ArrayBox

func leak(x: borrow [2]Int) -> Int:
    leaked.items = x
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedFixedArrayGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

struct ArrayBox:
    items: [2]Int

var leaked: ArrayBox

pub func leak(x: borrow [2]Int) -> Int:
    leaked.items = x
    return 0
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"lib/leak.t4",
		"borrowed local 'x' cannot escape via global assignment to 'leaked'",
	)
}

func TestOwnershipRejectsBorrowedFixedArrayOptionalGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var leaked: [2]Int? = none

func leak(x: borrow [2]Int) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedFixedArrayOptionalGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

var leaked: [2]Int? = none

pub func leak(x: borrow [2]Int) -> Int:
    leaked = x
    return 0
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"lib/leak.t4",
		"borrowed local 'x' cannot escape via global assignment to 'leaked'",
	)
}

func TestOwnershipRejectsBorrowedFixedArrayInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(x: borrow [2]Int, out: inout [2]Int) -> Int:
    out = x
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsCrossModuleBorrowedFixedArrayInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func leak(x: borrow [2]Int, out: inout [2]Int) -> Int:
    out = x
    return 0
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"lib/leak.t4",
		"borrowed local 'x' cannot escape via inout assignment to 'out'",
	)
}

func TestOwnershipRejectsConsumeOfBorrowDerivedAlias(t *testing.T) {
	src := []byte(`
func take(x: consume []u8) -> Int:
    return 0

func leak(x: borrow []u8) -> Int:
    let y: []u8 = x
    return take(y)

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected borrowed consume rejection")
	}
	if !strings.Contains(err.Error(), "borrow") || !strings.Contains(err.Error(), "consume") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsIfBlockLocalEscape(t *testing.T) {
	src := []byte(`
func main() -> Int:
    if 1:
        let x: Int = 1
    return x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected block-local scope error")
	}
	if !strings.Contains(err.Error(), "out of scope") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipConsumeArgumentCannotBeReused(t *testing.T) {
	src := []byte(`
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let a: Int = 1
    let b: Int = take(a)
    return a + b
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected consumed reuse error")
	}
	if !strings.Contains(err.Error(), "cannot use consumed value 'a'") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipConsumedValueCannotBeReassigned(t *testing.T) {
	src := []byte(`
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    var a: Int = 1
    let b: Int = take(a)
    a = b
    return b
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected consumed assignment error")
	}
	if !strings.Contains(err.Error(), "cannot use consumed value 'a'") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsBorrowInoutAlias(t *testing.T) {
	src := []byte(`
func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    return mix(a, a)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected borrow/inout aliasing error")
	}
	if !strings.Contains(err.Error(), "alias") && !strings.Contains(err.Error(), "borrow") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsConsumeInoutAlias(t *testing.T) {
	src := []byte(`
func mix(taken: consume Int, write: inout Int) -> Int:
    write = write + taken
    return write

func main() -> Int:
    var a: Int = 1
    return mix(a, a)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected consume/inout aliasing error")
	}
	if !strings.Contains(err.Error(), "alias") && !strings.Contains(err.Error(), "consume") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsCapturedFunctionTypedLocalBorrowInoutAlias(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int:
    var a: Int = 1
    let bias: Int = 0
    let cb: fn(borrow Int, inout Int) -> Int = fn(read: borrow Int, write: inout Int) -> Int:
        write = write + read + bias
        return write
    return cb(a, a)
`, "inout argument 'a' aliases borrowed argument in function-typed callback 'cb'")
}

func TestOwnershipRejectsCapturedFunctionTypedLocalConsumeInoutAlias(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int:
    var a: Int = 1
    let bias: Int = 0
    let cb: fn(consume Int, inout Int) -> Int = fn(taken: consume Int, write: inout Int) -> Int:
        write = write + taken + bias
        return write
    return cb(a, a)
`, "inout argument 'a' aliases consumed argument in function-typed callback 'cb'")
}

func TestOwnershipRejectsCapturedFunctionTypedLocalUseAfterConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int:
    let value: Int = 1
    let bias: Int = 0
    let cb: fn(consume Int) -> Int = fn(taken: consume Int) -> Int:
        return taken + bias
    let moved: Int = cb(value)
    return value + moved
`, "cannot use consumed value 'value'")
}

func TestOwnershipAllowsCapturedFunctionTypedLocalDistinctBorrowInout(t *testing.T) {
	testkit.RequireCheckOK(t, `
func main() -> Int:
    var read: Int = 1
    var write: Int = 2
    let bias: Int = 0
    let cb: fn(borrow Int, inout Int) -> Int = fn(left: borrow Int, right: inout Int) -> Int:
        right = right + left + bias
        return right
    return cb(read, write)
`)
}

func TestOwnershipRejectsPassingBorrowedValueToOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink(x: []u8) -> Int:
    return 0

func caller(x: borrow []u8) -> Int:
    let y: []u8 = x
    return sink(y)

func main() -> Int:
    return 0
`, "borrowed value derived from")
}

func TestOwnershipRejectsBorrowedPtrPassedToOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink(x: ptr) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    return sink(x)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'")
}

func TestOwnershipRejectsBorrowedPtrAliasPassedToOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink(x: ptr) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    let y: ptr = x
    return sink(y)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'")
}

func TestOwnershipRejectsBorrowedPtrAggregatePassedToOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func sink(box: PtrBox) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    return sink(PtrBox(raw: x))

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'")
}

func TestOwnershipRejectsBorrowedPtrAggregateConsumedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func sink(box: consume PtrBox) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    let box: PtrBox = PtrBox(raw: x)
    return sink(box)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be consumed by 'sink'")
}

func TestOwnershipRejectsBorrowedPtrConsumedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink(raw: consume ptr) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    return sink(x)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be consumed by 'sink'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrConsumedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub func sink(raw: consume ptr) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    return sinker.sink(x)

func main() -> Int:
    return 0
`,
	}
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed value derived from 'x' cannot be consumed by 'lib.sink.sink'",
	)
}

func TestOwnershipRejectsBorrowedPtrAggregateInoutParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func sink(box: inout PtrBox) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    var box: PtrBox = PtrBox(raw: x)
    return sink(box)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed as inout to 'sink'")
}

func TestOwnershipRejectsBorrowedPtrEnumAggregatePassedToOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PtrMsg:
    case raw(ptr)

func sink(msg: PtrMsg) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    return sink(PtrMsg.raw(x))

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'")
}

func TestOwnershipRejectsBorrowedPtrEnumAggregateConsumedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PtrMsg:
    case raw(ptr)

func sink(msg: consume PtrMsg) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    let msg: PtrMsg = PtrMsg.raw(x)
    return sink(msg)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be consumed by 'sink'")
}

func TestOwnershipRejectsBorrowedPtrEnumAggregateInoutParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PtrMsg:
    case raw(ptr)

func sink(msg: inout PtrMsg) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    var msg: PtrMsg = PtrMsg.raw(x)
    return sink(msg)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed as inout to 'sink'")
}

func TestOwnershipRejectsBorrowedPtrPassedToFunctionTypedOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func caller(cb: fn(ptr) -> Int, x: borrow ptr) -> Int:
    return cb(x)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of callback 'cb'")
}

func TestOwnershipRejectsBorrowedPtrAliasPassedToFunctionTypedOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func caller(cb: fn(ptr) -> Int, x: borrow ptr) -> Int:
    let y: ptr = x
    return cb(y)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of callback 'cb'")
}

func TestOwnershipRejectsBorrowedOptionalPtrPassedToFunctionTypedOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func caller(cb: fn(ptr?) -> Int, maybe: borrow ptr?) -> Int:
    return cb(maybe)

func main() -> Int:
    return 0
`, ("borrowed value derived from 'maybe' cannot be passed to non-" +
		"borrow parameter 1 of callback 'cb'"))
}

func TestOwnershipRejectsBorrowedOptionalPtrConsumedByFunctionTypedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func caller(cb: fn(consume ptr?) -> Int, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return cb(alias)

func main() -> Int:
    return 0
`, "borrowed value derived from 'maybe' cannot be consumed by callback 'cb'")
}

func TestOwnershipRejectsBorrowedOptionalPtrInoutFunctionTypedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func caller(cb: fn(inout ptr?) -> Int, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return cb(alias)

func main() -> Int:
    return 0
`, "borrowed value derived from 'maybe' cannot be passed as inout to callback 'cb'")
}

func TestOwnershipRejectsBorrowedPtrAggregatePassedToFunctionTypedOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func caller(cb: fn(PtrBox) -> Int, x: borrow ptr) -> Int:
    return cb(PtrBox(raw: x))

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of callback 'cb'")
}

func TestOwnershipRejectsBorrowedPtrAggregateConsumedByFunctionTypedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func caller(cb: fn(consume PtrBox) -> Int, x: borrow ptr) -> Int:
    let box: PtrBox = PtrBox(raw: x)
    return cb(box)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be consumed by callback 'cb'")
}

func TestOwnershipRejectsBorrowedPtrAggregateInoutFunctionTypedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func caller(cb: fn(inout PtrBox) -> Int, x: borrow ptr) -> Int:
    var box: PtrBox = PtrBox(raw: x)
    return cb(box)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed as inout to callback 'cb'")
}

func TestOwnershipRejectsBorrowedPtrAggregatePassedToFunctionTypedStructField(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

struct Handler:
    cb: fn(PtrBox) -> Int

func caller(h: Handler, x: borrow ptr) -> Int:
    return h.cb(PtrBox(raw: x))

func main() -> Int:
    return 0
`, ("borrowed value derived from 'x' cannot be passed to non-borrow " +
		"parameter 1 of function-typed struct field call 'h.cb'"))
}

func TestOwnershipRejectsBorrowedPtrAggregateConsumedByFunctionTypedStructField(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

struct Handler:
    cb: fn(consume PtrBox) -> Int

func caller(h: Handler, x: borrow ptr) -> Int:
    let box: PtrBox = PtrBox(raw: x)
    return h.cb(box)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be consumed by function-typed struct field call 'h.cb'")
}

func TestOwnershipRejectsBorrowedPtrAggregateInoutFunctionTypedStructField(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

struct Handler:
    cb: fn(inout PtrBox) -> Int

func caller(h: Handler, x: borrow ptr) -> Int:
    var box: PtrBox = PtrBox(raw: x)
    return h.cb(box)

func main() -> Int:
    return 0
`, ("borrowed value derived from 'x' cannot be passed as inout to " +
		"function-typed struct field call 'h.cb'"))
}

func TestOwnershipRejectsBorrowedOptionalPtrPassedToFunctionTypedStructField(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Handler:
    cb: fn(ptr?) -> Int

func caller(h: Handler, maybe: borrow ptr?) -> Int:
    return h.cb(maybe)

func main() -> Int:
    return 0
`, ("borrowed value derived from 'maybe' cannot be passed to non-" +
		"borrow parameter 1 of function-typed struct field call 'h.cb'"))
}

func TestOwnershipRejectsBorrowedOptionalPtrConsumedByFunctionTypedStructField(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Handler:
    cb: fn(consume ptr?) -> Int

func caller(h: Handler, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return h.cb(alias)

func main() -> Int:
    return 0
`, ("borrowed value derived from 'maybe' cannot be consumed by " +
		"function-typed struct field call 'h.cb'"))
}

func TestOwnershipRejectsBorrowedOptionalPtrInoutFunctionTypedStructField(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Handler:
    cb: fn(inout ptr?) -> Int

func caller(h: Handler, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return h.cb(alias)

func main() -> Int:
    return 0
`, ("borrowed value derived from 'maybe' cannot be passed as inout " +
		"to function-typed struct field call 'h.cb'"))
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrPassedToFunctionTypedStructField(
	t *testing.T,
) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub struct Handler:
    cb: fn(ptr?) -> Int
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller(h: callbacks.Handler, maybe: borrow ptr?) -> Int:
    return h.cb(maybe)

func main() -> Int:
    return 0
`,
	}
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		("borrowed value derived from 'maybe' cannot be passed to non-" +
			"borrow parameter 1 of function-typed struct field call 'h.cb'"),
	)
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrConsumedByFunctionTypedStructField(
	t *testing.T,
) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub struct Handler:
    cb: fn(consume ptr?) -> Int
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller(h: callbacks.Handler, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return h.cb(alias)

func main() -> Int:
    return 0
`,
	}
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		("borrowed value derived from 'maybe' cannot be consumed by " +
			"function-typed struct field call 'h.cb'"),
	)
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrInoutFunctionTypedStructField(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub struct Handler:
    cb: fn(inout ptr?) -> Int
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller(h: callbacks.Handler, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return h.cb(alias)

func main() -> Int:
    return 0
`,
	}
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		("borrowed value derived from 'maybe' cannot be passed as inout " +
			"to function-typed struct field call 'h.cb'"),
	)
}
