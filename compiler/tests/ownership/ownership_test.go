package compiler_test

import (
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/testkit"
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
	if params[0].Ownership != "borrow" || params[1].Ownership != "inout" || params[2].Ownership != "consume" || params[3].Ownership != "borrow" {
		t.Fatalf("ownership markers = %q/%q/%q/%q", params[0].Ownership, params[1].Ownership, params[2].Ownership, params[3].Ownership)
	}
	if got := strings.Join(params[3].Type.ParamOwnership, ","); got != "borrow,inout,consume" {
		t.Fatalf("function type ownership markers = %q", got)
	}
	formatted, err := compiler.FormatSource(src, "ownership.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	wantParams := "a: borrow Int, b: inout Int, c: consume Int, cb: borrow fn(borrow Int, inout Int, consume Int) -> Int"
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
	if !strings.Contains(err.Error(), "borrowed slice return requires '-> borrow []u8' or '.copy()'") {
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
	if !strings.Contains(err.Error(), "borrowed slice return requires '-> borrow []u8' or '.copy()'") {
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
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed local 'x' cannot escape via return")
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
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed local 'x' cannot escape via global assignment to 'leaked'")
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
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed local 'x' cannot escape via global assignment to 'leaked'")
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
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed local 'x' cannot escape via inout assignment to 'out'")
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be consumed by 'lib.sink.sink'")
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
`, "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of callback 'cb'")
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
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'")
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
`, "borrowed value derived from 'x' cannot be passed as inout to function-typed struct field call 'h.cb'")
}

func TestOwnershipRejectsBorrowedOptionalPtrPassedToFunctionTypedStructField(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Handler:
    cb: fn(ptr?) -> Int

func caller(h: Handler, maybe: borrow ptr?) -> Int:
    return h.cb(maybe)

func main() -> Int:
    return 0
`, "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'")
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
`, "borrowed value derived from 'maybe' cannot be consumed by function-typed struct field call 'h.cb'")
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
`, "borrowed value derived from 'maybe' cannot be passed as inout to function-typed struct field call 'h.cb'")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrPassedToFunctionTypedStructField(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrConsumedByFunctionTypedStructField(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be consumed by function-typed struct field call 'h.cb'")
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be passed as inout to function-typed struct field call 'h.cb'")
}

func TestOwnershipRejectsBorrowedPtrEnumAggregatePassedToFunctionTypedOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PtrMsg:
    case raw(ptr)

func caller(cb: fn(PtrMsg) -> Int, x: borrow ptr) -> Int:
    return cb(PtrMsg.raw(x))

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of callback 'cb'")
}

func TestOwnershipRejectsBorrowedSliceAggregatePassedToFunctionTypedParameter(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "struct-owned",
			src: `
struct BufBox:
    buf: []u8

func caller(cb: fn(BufBox) -> Int, x: borrow []u8) -> Int:
    return cb(BufBox(buf: x))

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of callback 'cb'",
		},
		{
			name: "struct-consume",
			src: `
struct BufBox:
    buf: []u8

func caller(cb: fn(consume BufBox) -> Int, x: borrow []u8) -> Int:
    let box: BufBox = BufBox(buf: x)
    return cb(box)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be consumed by callback 'cb'",
		},
		{
			name: "struct-inout",
			src: `
struct BufBox:
    buf: []u8

func caller(cb: fn(inout BufBox) -> Int, x: borrow []u8) -> Int:
    var box: BufBox = BufBox(buf: x)
    return cb(box)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to callback 'cb'",
		},
		{
			name: "enum-owned",
			src: `
enum BufMsg:
    case send([]u8)

func caller(cb: fn(BufMsg) -> Int, x: borrow []u8) -> Int:
    return cb(BufMsg.send(x))

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of callback 'cb'",
		},
		{
			name: "enum-consume",
			src: `
enum BufMsg:
    case send([]u8)

func caller(cb: fn(consume BufMsg) -> Int, x: borrow []u8) -> Int:
    let msg: BufMsg = BufMsg.send(x)
    return cb(msg)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be consumed by callback 'cb'",
		},
		{
			name: "enum-inout",
			src: `
enum BufMsg:
    case send([]u8)

func caller(cb: fn(inout BufMsg) -> Int, x: borrow []u8) -> Int:
    var msg: BufMsg = BufMsg.send(x)
    return cb(msg)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to callback 'cb'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.wantText)
		})
	}
}

func TestOwnershipRejectsBorrowedSliceAggregatePassedToFunctionTypedStructField(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "struct-owned",
			src: `
struct BufBox:
    buf: []u8

struct Handler:
    cb: fn(BufBox) -> Int

func caller(h: Handler, x: borrow []u8) -> Int:
    return h.cb(BufBox(buf: x))

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'",
		},
		{
			name: "struct-consume",
			src: `
struct BufBox:
    buf: []u8

struct Handler:
    cb: fn(consume BufBox) -> Int

func caller(h: Handler, x: borrow []u8) -> Int:
    let box: BufBox = BufBox(buf: x)
    return h.cb(box)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed struct field call 'h.cb'",
		},
		{
			name: "struct-inout",
			src: `
struct BufBox:
    buf: []u8

struct Handler:
    cb: fn(inout BufBox) -> Int

func caller(h: Handler, x: borrow []u8) -> Int:
    var box: BufBox = BufBox(buf: x)
    return h.cb(box)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-owned",
			src: `
enum BufMsg:
    case send([]u8)

struct Handler:
    cb: fn(BufMsg) -> Int

func caller(h: Handler, x: borrow []u8) -> Int:
    return h.cb(BufMsg.send(x))

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-consume",
			src: `
enum BufMsg:
    case send([]u8)

struct Handler:
    cb: fn(consume BufMsg) -> Int

func caller(h: Handler, x: borrow []u8) -> Int:
    let msg: BufMsg = BufMsg.send(x)
    return h.cb(msg)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-inout",
			src: `
enum BufMsg:
    case send([]u8)

struct Handler:
    cb: fn(inout BufMsg) -> Int

func caller(h: Handler, x: borrow []u8) -> Int:
    var msg: BufMsg = BufMsg.send(x)
    return h.cb(msg)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed struct field call 'h.cb'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.wantText)
		})
	}
}

func TestOwnershipRejectsCrossModuleBorrowedSliceAggregatePassedToFunctionTypedStructField(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		appCall  string
		wantText string
	}{
		{
			name: "struct-owned",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub struct Handler:
    cb: fn(BufBox) -> Int
`,
			appCall:  "return h.cb(callbacks.BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'",
		},
		{
			name: "struct-consume",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub struct Handler:
    cb: fn(consume BufBox) -> Int
`,
			appCall:  "let box: callbacks.BufBox = callbacks.BufBox(buf: x)\n    return h.cb(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed struct field call 'h.cb'",
		},
		{
			name: "struct-inout",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub struct Handler:
    cb: fn(inout BufBox) -> Int
`,
			appCall:  "var box: callbacks.BufBox = callbacks.BufBox(buf: x)\n    return h.cb(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-owned",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub struct Handler:
    cb: fn(BufMsg) -> Int
`,
			appCall:  "return h.cb(callbacks.BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-consume",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub struct Handler:
    cb: fn(consume BufMsg) -> Int
`,
			appCall:  "let msg: callbacks.BufMsg = callbacks.BufMsg.send(x)\n    return h.cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-inout",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub struct Handler:
    cb: fn(inout BufMsg) -> Int
`,
			appCall:  "var msg: callbacks.BufMsg = callbacks.BufMsg.send(x)\n    return h.cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed struct field call 'h.cb'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/callbacks.t4": tt.libSrc,
				"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller(h: callbacks.Handler, x: borrow []u8) -> Int:
    ` + tt.appCall + `

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.wantText)
		})
	}
}

func TestOwnershipRejectsBorrowedPtrAggregatePassedToFunctionTypedEnumPayload(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

enum Choice:
    case some(fn(PtrBox) -> Int)
    case empty

func caller(choice: Choice, x: borrow ptr) -> Int:
    match choice:
    case Choice.some(cb):
        return cb(PtrBox(raw: x))
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'")
}

func TestOwnershipRejectsBorrowedPtrAggregateConsumedByFunctionTypedEnumPayload(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

enum Choice:
    case some(fn(consume PtrBox) -> Int)
    case empty

func caller(choice: Choice, x: borrow ptr) -> Int:
    let box: PtrBox = PtrBox(raw: x)
    match choice:
    case Choice.some(cb):
        return cb(box)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be consumed by function-typed enum payload call 'cb'")
}

func TestOwnershipRejectsBorrowedPtrAggregateInoutFunctionTypedEnumPayload(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

enum Choice:
    case some(fn(inout PtrBox) -> Int)
    case empty

func caller(choice: Choice, x: borrow ptr) -> Int:
    var box: PtrBox = PtrBox(raw: x)
    match choice:
    case Choice.some(cb):
        return cb(box)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed as inout to function-typed enum payload call 'cb'")
}

func TestOwnershipRejectsBorrowedSliceAggregatePassedToFunctionTypedEnumPayload(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "struct-owned",
			src: `
struct BufBox:
    buf: []u8

enum Choice:
    case some(fn(BufBox) -> Int)
    case empty

func caller(choice: Choice, x: borrow []u8) -> Int:
    match choice:
    case Choice.some(cb):
        return cb(BufBox(buf: x))
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'",
		},
		{
			name: "struct-consume",
			src: `
struct BufBox:
    buf: []u8

enum Choice:
    case some(fn(consume BufBox) -> Int)
    case empty

func caller(choice: Choice, x: borrow []u8) -> Int:
    let box: BufBox = BufBox(buf: x)
    match choice:
    case Choice.some(cb):
        return cb(box)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed enum payload call 'cb'",
		},
		{
			name: "struct-inout",
			src: `
struct BufBox:
    buf: []u8

enum Choice:
    case some(fn(inout BufBox) -> Int)
    case empty

func caller(choice: Choice, x: borrow []u8) -> Int:
    var box: BufBox = BufBox(buf: x)
    match choice:
    case Choice.some(cb):
        return cb(box)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed enum payload call 'cb'",
		},
		{
			name: "enum-owned",
			src: `
enum BufMsg:
    case send([]u8)

enum Choice:
    case some(fn(BufMsg) -> Int)
    case empty

func caller(choice: Choice, x: borrow []u8) -> Int:
    match choice:
    case Choice.some(cb):
        return cb(BufMsg.send(x))
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'",
		},
		{
			name: "enum-consume",
			src: `
enum BufMsg:
    case send([]u8)

enum Choice:
    case some(fn(consume BufMsg) -> Int)
    case empty

func caller(choice: Choice, x: borrow []u8) -> Int:
    let msg: BufMsg = BufMsg.send(x)
    match choice:
    case Choice.some(cb):
        return cb(msg)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed enum payload call 'cb'",
		},
		{
			name: "enum-inout",
			src: `
enum BufMsg:
    case send([]u8)

enum Choice:
    case some(fn(inout BufMsg) -> Int)
    case empty

func caller(choice: Choice, x: borrow []u8) -> Int:
    var msg: BufMsg = BufMsg.send(x)
    match choice:
    case Choice.some(cb):
        return cb(msg)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed enum payload call 'cb'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.wantText)
		})
	}
}

func TestOwnershipRejectsCrossModuleBorrowedSliceAggregatePassedToFunctionTypedEnumPayload(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		setup    string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub enum Choice:
    case some(fn(BufBox) -> Int)
    case empty
`,
			call:     "return cb(callbacks.BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'",
		},
		{
			name: "struct-consume",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub enum Choice:
    case some(fn(consume BufBox) -> Int)
    case empty
`,
			setup:    "let box: callbacks.BufBox = callbacks.BufBox(buf: x)\n    ",
			call:     "return cb(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed enum payload call 'cb'",
		},
		{
			name: "struct-inout",
			libSrc: `module lib.callbacks

pub struct BufBox:
    buf: []u8

pub enum Choice:
    case some(fn(inout BufBox) -> Int)
    case empty
`,
			setup:    "var box: callbacks.BufBox = callbacks.BufBox(buf: x)\n    ",
			call:     "return cb(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed enum payload call 'cb'",
		},
		{
			name: "enum-owned",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub enum Choice:
    case some(fn(BufMsg) -> Int)
    case empty
`,
			call:     "return cb(callbacks.BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'",
		},
		{
			name: "enum-consume",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub enum Choice:
    case some(fn(consume BufMsg) -> Int)
    case empty
`,
			setup:    "let msg: callbacks.BufMsg = callbacks.BufMsg.send(x)\n    ",
			call:     "return cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed enum payload call 'cb'",
		},
		{
			name: "enum-inout",
			libSrc: `module lib.callbacks

pub enum BufMsg:
    case send([]u8)

pub enum Choice:
    case some(fn(inout BufMsg) -> Int)
    case empty
`,
			setup:    "var msg: callbacks.BufMsg = callbacks.BufMsg.send(x)\n    ",
			call:     "return cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed enum payload call 'cb'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/callbacks.t4": tt.libSrc,
				"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller(choice: callbacks.Choice, x: borrow []u8) -> Int:
    ` + tt.setup + `match choice:
    case callbacks.Choice.some(cb):
        ` + tt.call + `
    case callbacks.Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.wantText)
		})
	}
}

func TestOwnershipRejectsBorrowedOptionalPtrPassedToFunctionTypedEnumPayload(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum Choice:
    case some(fn(ptr?) -> Int)
    case empty

func caller(choice: Choice, maybe: borrow ptr?) -> Int:
    match choice:
    case Choice.some(cb):
        return cb(maybe)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`, "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'")
}

func TestOwnershipRejectsBorrowedOptionalPtrConsumedByFunctionTypedEnumPayload(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum Choice:
    case some(fn(consume ptr?) -> Int)
    case empty

func caller(choice: Choice, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    match choice:
    case Choice.some(cb):
        return cb(alias)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`, "borrowed value derived from 'maybe' cannot be consumed by function-typed enum payload call 'cb'")
}

func TestOwnershipRejectsBorrowedOptionalPtrInoutFunctionTypedEnumPayload(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum Choice:
    case some(fn(inout ptr?) -> Int)
    case empty

func caller(choice: Choice, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    match choice:
    case Choice.some(cb):
        return cb(alias)
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`, "borrowed value derived from 'maybe' cannot be passed as inout to function-typed enum payload call 'cb'")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrPassedToFunctionTypedEnumPayload(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum Choice:
    case some(fn(ptr?) -> Int)
    case empty
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller(choice: callbacks.Choice, maybe: borrow ptr?) -> Int:
    match choice:
    case callbacks.Choice.some(cb):
        return cb(maybe)
    case callbacks.Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrConsumedByFunctionTypedEnumPayload(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum Choice:
    case some(fn(consume ptr?) -> Int)
    case empty
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller(choice: callbacks.Choice, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    match choice:
    case callbacks.Choice.some(cb):
        return cb(alias)
    case callbacks.Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be consumed by function-typed enum payload call 'cb'")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrInoutFunctionTypedEnumPayload(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum Choice:
    case some(fn(inout ptr?) -> Int)
    case empty
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller(choice: callbacks.Choice, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    match choice:
    case callbacks.Choice.some(cb):
        return cb(alias)
    case callbacks.Choice.empty:
        return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be passed as inout to function-typed enum payload call 'cb'")
}

func TestOwnershipRejectsBorrowEscapeViaInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(read: borrow []u8, write: inout []u8) -> Int:
    write = read
    return 0

func main() -> Int:
    return 0
`, "cannot escape via inout")
}

func TestOwnershipAllowsBorrowToBorrowForwarding(t *testing.T) {
	testkit.RequireCheckOK(t, `
func read(x: borrow []u8) -> Int:
    return 0

func forward(x: borrow []u8) -> Int:
    return read(x)

func main() -> Int:
    return 0
`)
}

func TestOwnershipRejectsConsumeArgumentThatIsNotLocalValue(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    return take(1)
`, "must be a local value")
}

func TestOwnershipRejectsInoutArgumentThatIsNotLocalValue(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func bump(x: inout Int) -> Int:
    x = x + 1
    return x

func main() -> Int:
    var a: Int = 1
    return bump(a + 1)
`, "must be a mutable local value")
}

func TestOwnershipReportsMaybeConsumedAfterControlFlowJoin(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let value: Int = 1
    if 1:
        let moved: Int = take(value)
    return value
`, "value 'value' may have been consumed after ownership join")
}

func TestOwnershipRejectsValueConsumedTwiceInSingleCall(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func add(a: consume Int, b: consume Int) -> Int:
    return a + b

func main() -> Int:
    let x: Int = 1
    return add(x, x)
`, "consumed more than once")
}

func TestOwnershipRejectsInoutBorrowAliasWithInoutFirst(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func mix(write: inout Int, read: borrow Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    return mix(a, a)
`, "aliases inout argument")
}

func TestOwnershipRejectsOverlappingMutableInoutSliceBorrow(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func mix(left: inout []u8, right: inout []u8) -> Int
uses mem:
    left[0] = 1
    right[0] = 2
    return left[0] + right[0]

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(1)
    return mix(xs, xs)
`, "used more than once")
}

func TestOwnershipRejectsInoutConsumeAliasWithInoutFirst(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func mix(write: inout Int, taken: consume Int) -> Int:
    write = write + taken
    return write

func main() -> Int:
    var a: Int = 1
    return mix(a, a)
`, "aliases inout argument")
}

func TestOwnershipAllowsBorrowInoutWithDistinctLocals(t *testing.T) {
	testkit.RequireCheckOK(t, `
func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    var b: Int = 2
    return mix(a, b)
`)
}

func TestOwnershipAllowsConsumeInoutWithDistinctLocals(t *testing.T) {
	testkit.RequireCheckOK(t, `
func mix(taken: consume Int, write: inout Int) -> Int:
    write = write + taken
    return write

func main() -> Int:
    var a: Int = 1
    var b: Int = 2
    return mix(a, b)
`)
}

func TestOwnershipAllowsWholeStructConsume(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Pair:
    left: Int
    right: Int

func take(pair: consume Pair) -> Int:
    return pair.left + pair.right

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    return take(pair)
`)
}

func TestOwnershipAllowsPartialStructFieldConsumeAndRemainingFieldUse(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return moved + pair.right
`)
}

func TestOwnershipRejectsConsumedStructFieldReuse(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return pair.left + moved
`, "cannot use consumed value 'pair.left'")
}

func TestOwnershipRejectsWholeStructUseAfterPartialFieldConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func use(pair: consume Pair) -> Int:
    return pair.left + pair.right

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return use(pair) + moved
`, "cannot use consumed value 'pair.left'")
}

func TestOwnershipRejectsWholeStructCopyAfterPartialFieldConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func sum(pair: Pair) -> Int:
    return pair.left + pair.right

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return sum(pair) + moved
`, "cannot use consumed value 'pair.left'")
}

func TestOwnershipRejectsWholeStructLetAfterPartialFieldConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let copy: Pair = pair
    return moved + copy.right
`, "cannot use consumed value 'pair.left'")
}

func TestOwnershipRejectsWholeStructReturnAfterPartialFieldConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func make() -> Pair:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return pair

func main() -> Int:
    let pair: Pair = make()
    return pair.right
`, "cannot use consumed value 'pair.left'")
}

func TestOwnershipAllowsConsumedStructFieldReassignment(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    var pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    pair.left = 3
    return pair.left + pair.right + moved
`)
}

func TestOwnershipAllowsWholeStructReassignmentAfterPartialFieldConsume(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    var pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    pair = Pair(left: 3, right: 4)
    return pair.left + pair.right + moved
`)
}

func TestOwnershipRejectsStructFieldAssignmentAfterWholeStructConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Pair:
    left: Int
    right: Int

func take(pair: consume Pair) -> Int:
    return pair.left + pair.right

func main() -> Int:
    var pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair)
    pair.left = 3
    return pair.left + moved
`, "cannot use consumed value 'pair'")
}

func TestOwnershipRejectsEnumConstructorPayloadAfterPartialFieldConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Pair:
    left: Int
    right: Int

enum Wrap:
    case one(Pair)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let wrapped: Wrap = Wrap.one(pair)
    return moved
`, "cannot use consumed value 'pair.left'")
}

func TestOwnershipRejectsCrossModuleEnumConstructorPayloadAfterPartialFieldConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Pair:
    left: Int
    right: Int

pub enum Wrap:
    case one(Pair)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: model.Pair = model.Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let wrapped: model.Wrap = model.Wrap.one(pair)
    return moved
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'pair.left'")
}

func TestOwnershipAllowsCrossModulePartialStructFieldConsumeAndRemainingFieldUse(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Pair:
    left: Int
    right: Int
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: model.Pair = model.Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return moved + pair.right
`,
	}
	requireCheckWorldFilesOK(t, files, "app/main.t4")
}

func TestOwnershipRejectsCrossModuleWholeStructReturnAfterPartialFieldConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Pair:
    left: Int
    right: Int
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func make() -> model.Pair:
    let pair: model.Pair = model.Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return pair

func main() -> Int:
    let pair: model.Pair = make()
    return pair.right
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'pair.left'")
}

func TestOwnershipRejectsCrossModuleWholeStructCallAfterPartialFieldConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Pair:
    left: Int
    right: Int
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func use(pair: model.Pair) -> Int:
    return pair.left + pair.right

func main() -> Int:
    let pair: model.Pair = model.Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return use(pair) + moved
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'pair.left'")
}

func TestOwnershipRejectsCrossModuleWholeStructLetAfterPartialFieldConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Pair:
    left: Int
    right: Int
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: model.Pair = model.Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let copy: model.Pair = pair
    return moved + copy.right
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'pair.left'")
}

func TestOwnershipRejectsCrossModuleWholeStructCopyAfterPartialFieldConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Pair:
    left: Int
    right: Int
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: model.Pair = model.Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let copy: model.Pair = pair
    return moved + copy.right
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'pair.left'")
}

func TestOwnershipAllowsPartialEnumPayloadConsumeAndSiblingPayloadUse(t *testing.T) {
	testkit.RequireCheckOK(t, `
enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        return moved + right
    case PairMsg.empty:
        return 0
`)
}

func TestOwnershipAllowsCrossModulePartialEnumPayloadConsumeAndSiblingPayloadUse(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PairMsg:
    case both(Int, Int)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: model.PairMsg = model.PairMsg.both(1, 2)
    match msg:
    case model.PairMsg.both(left, right):
        let moved: Int = take(left)
        return moved + right
    case model.PairMsg.empty:
        return 0
`,
	}
	requireCheckWorldFilesOK(t, files, "app/main.t4")
}

func TestOwnershipRejectsWholeEnumUseAfterPartialPayloadConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func use(msg: PairMsg) -> Int:
    match msg:
    case PairMsg.both(left, right):
        return left + right
    case PairMsg.empty:
        return 0

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        return use(msg) + moved
    case PairMsg.empty:
        return 0
`, "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsWholeEnumUseAfterPartialPayloadConsumeWithAliasBinding(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum Msg:
    case some(Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func use(msg: Msg) -> Int:
    return 0

func main() -> Int:
    let msg: Msg = Msg.some(1)
    match msg:
    case Msg.some(raw):
        let moved: Int = take(raw)
        return use(msg)
    case Msg.empty:
        return 0
`, "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsWholeEnumUseAfterPartialPayloadConsumeWithAliasBindingCrossModule(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum Msg:
    case some(Int)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func use(msg: model.Msg) -> Int:
    return 0

func main() -> Int:
    let msg: model.Msg = model.Msg.some(1)
    match msg:
    case model.Msg.some(raw):
        let moved: Int = take(raw)
        return use(msg)
    case model.Msg.empty:
        return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsWholeEnumLetAfterPartialPayloadConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        let copy: PairMsg = msg
        return moved + right
    case PairMsg.empty:
        return 0
`, "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsWholeEnumReturnAfterPartialPayloadConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func make() -> PairMsg:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        return msg
    case PairMsg.empty:
        return PairMsg.empty

func main() -> Int:
    let msg: PairMsg = make()
    return 0
`, "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsCrossModuleWholeEnumReturnAfterPartialPayloadConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PairMsg:
    case both(Int, Int)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func make() -> model.PairMsg:
    let msg: model.PairMsg = model.PairMsg.both(1, 2)
    match msg:
    case model.PairMsg.both(left, right):
        let moved: Int = take(left)
        return msg
    case model.PairMsg.empty:
        return model.PairMsg.empty

func main() -> Int:
    let msg: model.PairMsg = make()
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsCrossModuleWholeEnumCallAfterPartialPayloadConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PairMsg:
    case both(Int, Int)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func use(msg: model.PairMsg) -> Int:
    match msg:
    case model.PairMsg.both(left, right):
        return left + right
    case model.PairMsg.empty:
        return 0

func main() -> Int:
    let msg: model.PairMsg = model.PairMsg.both(1, 2)
    match msg:
    case model.PairMsg.both(left, right):
        let moved: Int = take(left)
        return use(msg) + moved
    case model.PairMsg.empty:
        return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsCrossModuleWholeEnumLetAfterPartialPayloadConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PairMsg:
    case both(Int, Int)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: model.PairMsg = model.PairMsg.both(1, 2)
    match msg:
    case model.PairMsg.both(left, right):
        let moved: Int = take(left)
        let copy: model.PairMsg = msg
        return moved + right
    case model.PairMsg.empty:
        return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsCrossModuleWholeEnumCopyAfterPartialPayloadConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PairMsg:
    case both(Int, Int)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: model.PairMsg = model.PairMsg.both(1, 2)
    match msg:
    case model.PairMsg.both(left, right):
        let moved: Int = take(left)
        let copy: model.PairMsg = msg
        return moved + right
    case model.PairMsg.empty:
        return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsEnumConstructorPayloadAfterPartialPayloadConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PairMsg:
    case both(Int, Int)
    case empty

enum Wrap:
    case one(PairMsg)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        let wrapped: Wrap = Wrap.one(msg)
        return moved + right
    case PairMsg.empty:
        return 0
`, "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsCrossModuleEnumConstructorPayloadAfterPartialPayloadConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PairMsg:
    case both(Int, Int)
    case empty

pub enum Wrap:
    case one(PairMsg)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: model.PairMsg = model.PairMsg.both(1, 2)
    match msg:
    case model.PairMsg.both(left, right):
        let moved: Int = take(left)
        let wrapped: model.Wrap = model.Wrap.one(msg)
        return moved + right
    case model.PairMsg.empty:
        return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'msg.$case0.payload0'")
}

func TestOwnershipRejectsConsumedEnumPayloadBindingReuse(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        return left + moved + right
    case PairMsg.empty:
        return 0
`, "cannot use consumed value 'left'")
}

func TestOwnershipAllowsWholeEnumReassignmentAfterPartialPayloadConsume(t *testing.T) {
	testkit.RequireCheckOK(t, `
enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func use(msg: PairMsg) -> Int:
    match msg:
    case PairMsg.both(left, right):
        return left + right
    case PairMsg.empty:
        return 0

func main() -> Int:
    var msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        msg = PairMsg.both(3, 4)
        return use(msg) + moved + right
    case PairMsg.empty:
        return 0
`)
}

func TestOwnershipGenericFunctionTypedGlobalPreservesConsumeMarker(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
val cb: fn(consume Int) -> Int = keep

func keep<T>(value: consume T) -> T:
    return value

func main() -> Int:
    let value: Int = 1
    let moved: Int = cb(value)
    return value + moved
`, "cannot use consumed value 'value'")
}

func TestOwnershipRejectsBorrowedPtrAggregatePassedToGenericOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func sink<T>(value: T) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    return sink(PtrBox(raw: x))

func main() -> Int:
    return 0
`, "borrowed value derived from 'x'")
}

func TestOwnershipRejectsBorrowedPtrAggregateConsumedByGenericParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func sink<T>(value: consume T) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    let box: PtrBox = PtrBox(raw: x)
    return sink(box)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x'")
}

func TestOwnershipRejectsBorrowedPtrAggregateInoutGenericParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func sink<T>(value: inout T) -> Int:
    return 0

func caller(x: borrow ptr) -> Int:
    var box: PtrBox = PtrBox(raw: x)
    return sink(box)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x'")
}

func TestOwnershipRejectsBorrowedSliceAggregatePassedToGenericParameter(t *testing.T) {
	tests := []struct {
		name     string
		typeSrc  string
		callee   string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callee: `func sink<T>(value: T) -> Int:
    return 0
`,
			call:     "return sink(BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "struct-consume",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callee: `func take<T>(value: consume T) -> Int:
    return 0
`,
			call:     "let box: BufBox = BufBox(buf: x)\n    return take(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed",
		},
		{
			name: "struct-inout",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callee: `func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			call:     "var box: BufBox = BufBox(buf: x)\n    return mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout",
		},
		{
			name: "enum-owned",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func sink<T>(value: T) -> Int:
    return 0
`,
			call:     "return sink(BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "enum-consume",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func take<T>(value: consume T) -> Int:
    return 0
`,
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return take(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed",
		},
		{
			name: "enum-inout",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			call:     "var msg: BufMsg = BufMsg.send(x)\n    return mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.typeSrc+"\n"+tt.callee+`
func caller(x: borrow []u8) -> Int:
    `+tt.call+`

func main() -> Int:
    return 0
`, tt.wantText)
		})
	}
}

func TestOwnershipRejectsGenericBorrowAggregateReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func leak<T>(value: borrow T) -> T:
    return value

func caller(x: borrow ptr) -> PtrBox:
    return leak(PtrBox(raw: x))

func main() -> Int:
    return 0
`, "borrowed")
}

func TestOwnershipRejectsGenericBorrowOptionalPtrReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak<T>(value: borrow T) -> T:
    return value

func caller(maybe: borrow ptr?) -> ptr?:
    return leak(maybe)

func main() -> Int:
    return 0
`, "borrowed")
}

func TestOwnershipRejectsCrossModuleGenericBorrowAggregateReturn(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub struct PtrBox:
    raw: ptr

pub func leak<T>(value: borrow T) -> T:
    return value
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func caller(x: borrow ptr) -> leaks.PtrBox:
    return leaks.leak(leaks.PtrBox(raw: x))

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed")
}

func TestOwnershipRejectsCrossModuleGenericBorrowOptionalPtrReturn(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func leak<T>(value: borrow T) -> T:
    return value
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func caller(maybe: borrow ptr?) -> ptr?:
    return leaks.leak(maybe)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEscapeViaOptionalAssignmentReturn(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func leak(x: borrow ptr) -> ptr?:
    var maybe: ptr? = none
    maybe = x
    return maybe
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsCrossModuleBorrowedSliceEscapeViaOptionalAssignmentReturn(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func leak(x: borrow []u8) -> []u8?:
    var maybe: []u8? = none
    maybe = x
    return maybe
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot escape through owned return")
}

func TestOwnershipRejectsCrossModuleBorrowedSliceEscapeViaOptionalAssignmentOwnedCall(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func sink(value: []u8?) -> Int:
    return 0

pub func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1")
}

func TestOwnershipRejectsCrossModuleBorrowedSliceEscapeViaOptionalAssignmentConsumeCall(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func sink(value: consume []u8?) -> Int:
    return 0

pub func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed value derived from 'x' cannot be consumed")
}

func TestOwnershipRejectsCrossModuleBorrowedSliceEscapeViaOptionalAssignmentInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func leak(x: borrow []u8, out: inout []u8?) -> Int:
    var maybe: []u8? = none
    maybe = x
    out = maybe
    return 0
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed local 'x' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEscapeViaOptionalAssignmentOwnedCall(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func sink(value: ptr?) -> Int:
    return 0

pub func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    return sink(maybe)
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEscapeViaOptionalAssignmentConsumeCall(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func sink(value: consume ptr?) -> Int:
    return 0

pub func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    return sink(maybe)
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed value derived from 'x' cannot be consumed")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEscapeViaOptionalAssignmentInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leak.t4": `module lib.leak

pub func leak(x: borrow ptr, out: inout ptr?) -> Int:
    var maybe: ptr? = none
    maybe = x
    out = maybe
    return 0
`,
		"app/main.t4": `module app.main
import lib.leak as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leak.t4", "borrowed local 'x' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowedOptionalPtrPassedToGenericOwnedParameter(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink<T>(value: T) -> Int:
    return 0

func caller(maybe: borrow ptr?) -> Int:
    return sink(maybe)

func main() -> Int:
    return 0
`, "borrowed value derived from 'maybe'")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrPassedToOwnedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

func sink(value: ptr?) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(maybe: borrow ptr?) -> Int:
    return sinker.sink(maybe)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrConsumedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

func take(value: consume ptr?) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return sinker.take(alias)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be consumed")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrInoutParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

func mutate(value: inout ptr?) -> Int:
    value = value
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return sinker.mutate(alias)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be passed as inout")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrPassedToGenericOwnedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub func sink<T>(value: T) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(maybe: borrow ptr?) -> Int:
    return sinker.sink(maybe)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrConsumedByGenericParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub func take<T>(value: consume T) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return sinker.take(alias)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be consumed")
}

func TestOwnershipRejectsCrossModuleBorrowedOptionalPtrInoutGenericParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return sinker.mutate(alias)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be passed as inout")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregatePassedToOwnedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub func sink(value: PtrBox) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    return sinker.sink(sinker.PtrBox(raw: x))

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregateConsumedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub func take(value: consume PtrBox) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    let box: sinker.PtrBox = sinker.PtrBox(raw: x)
    return sinker.take(box)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be consumed")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregateInoutParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub func mutate(value: inout PtrBox) -> Int:
    value = value
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    var box: sinker.PtrBox = sinker.PtrBox(raw: x)
    return sinker.mutate(box)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be passed as inout")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregatePassedToGenericOwnedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub func sink<T>(value: T) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    return sinker.sink(sinker.PtrBox(raw: x))

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregateConsumedByGenericParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub func take<T>(value: consume T) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    let box: sinker.PtrBox = sinker.PtrBox(raw: x)
    return sinker.take(box)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be consumed")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregateInoutGenericParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    var box: sinker.PtrBox = sinker.PtrBox(raw: x)
    return sinker.mutate(box)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be passed as inout")
}

func TestOwnershipRejectsCrossModuleBorrowedSliceAggregatePassedToGenericParameter(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		appCall  string
		wantText string
	}{
		{
			name: "struct-owned",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func sink<T>(value: T) -> Int:
    return 0
`,
			appCall:  "return sinker.sink(sinker.BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "struct-consume",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func take<T>(value: consume T) -> Int:
    return 0
`,
			appCall:  "let box: sinker.BufBox = sinker.BufBox(buf: x)\n    return sinker.take(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed",
		},
		{
			name: "struct-inout",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			appCall:  "var box: sinker.BufBox = sinker.BufBox(buf: x)\n    return sinker.mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout",
		},
		{
			name: "enum-owned",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func sink<T>(value: T) -> Int:
    return 0
`,
			appCall:  "return sinker.sink(sinker.BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "enum-consume",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func take<T>(value: consume T) -> Int:
    return 0
`,
			appCall:  "let msg: sinker.BufMsg = sinker.BufMsg.send(x)\n    return sinker.take(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed",
		},
		{
			name: "enum-inout",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			appCall:  "var msg: sinker.BufMsg = sinker.BufMsg.send(x)\n    return sinker.mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": tt.libSrc,
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow []u8) -> Int:
    ` + tt.appCall + `

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.wantText)
		})
	}
}

func TestOwnershipRejectsCrossModuleBorrowedPtrNestedAggregatePassedToOwnedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox

pub func sink(value: OuterBox) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    return sinker.sink(sinker.OuterBox(box: sinker.PtrBox(raw: x)))

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrNestedAggregateConsumedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox

pub func take(value: consume OuterBox) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    let outer: sinker.OuterBox = sinker.OuterBox(box: sinker.PtrBox(raw: x))
    return sinker.take(outer)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be consumed")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrNestedAggregateInoutParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox

pub func mutate(value: inout OuterBox) -> Int:
    value = value
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    var outer: sinker.OuterBox = sinker.OuterBox(box: sinker.PtrBox(raw: x))
    return sinker.mutate(outer)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be passed as inout")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEnumAggregatePassedToOwnedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub enum PtrMsg:
    case raw(ptr)

pub func sink(value: PtrMsg) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    return sinker.sink(sinker.PtrMsg.raw(x))

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEnumAggregateConsumedParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub enum PtrMsg:
    case raw(ptr)

pub func take(value: consume PtrMsg) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    let msg: sinker.PtrMsg = sinker.PtrMsg.raw(x)
    return sinker.take(msg)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be consumed")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEnumAggregateInoutParameter(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub enum PtrMsg:
    case raw(ptr)

pub func mutate(value: inout PtrMsg) -> Int:
    value = value
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    var msg: sinker.PtrMsg = sinker.PtrMsg.raw(x)
    return sinker.mutate(msg)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'x' cannot be passed as inout")
}

func TestOwnershipRejectsGenericFunctionTypedGlobalOwnershipMismatch(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
val cb: fn(Int) -> Int = keep

func keep<T>(value: consume T) -> T:
    return value

func main() -> Int:
    return cb(1)
`, "function-typed assignment 'cb' parameter 1 ownership mismatch: expected 'owned', got 'consume'")
}

func TestOwnershipRejectsProtocolImplParameterOwnershipMismatch(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box:
    value: Int

protocol Sink:
    func sink(self: consume Box) -> Int

extension Box:
    func sink(self: Box) -> Int:
        return self.value

impl Box: Sink

func main() -> Int:
    return 0
`, "ownership")
}

func TestOwnershipAllowsProtocolImplMatchingParameterOwnership(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Box:
    value: Int

protocol Sink:
    func sink(self: consume Box) -> Int

extension Box:
    func sink(self: consume Box) -> Int:
        return self.value

impl Box: Sink

func main() -> Int:
    return 0
`)
}

func TestOwnershipRejectsCrossModuleProtocolImplParameterOwnershipMismatch(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Box:
    value: Int

pub protocol Sink:
    func sink(self: consume Box) -> Int

extension Box:
    func sink(self: Box) -> Int:
        return self.value

impl Box: Sink
`,
		"app/main.t4": `module app.main
import lib.model as model

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "ownership")
}

func TestOwnershipAllowsCrossModuleProtocolImplMatchingParameterOwnership(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Box:
    value: Int

pub protocol Sink:
    func sink(self: consume Box) -> Int

extension Box:
    func sink(self: consume Box) -> Int:
        return self.value

impl Box: Sink
`,
		"app/main.t4": `module app.main
import lib.model as model

func main() -> Int:
    let box: model.Box = model.Box(value: 1)
    return box.value
`,
	}
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, files)
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestOwnershipRejectsGenericProtocolRequirementParameterOwnershipMismatch(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box:
    value: Int

protocol Mapper:
    func map<T>(self: Box, value: consume T) -> T

extension Box:
    func map<T>(self: Box, value: T) -> T:
        return value

impl Box: Mapper

func main() -> Int:
    return 0
`, "ownership")
}

func TestOwnershipRejectsCrossModuleGenericProtocolRequirementParameterOwnershipMismatch(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Box:
    value: Int

pub protocol Mapper:
    func map<T>(self: Box, value: consume T) -> T
`,
		"app/main.t4": `module app.main
import lib.model as model

extension model.Box:
    func map<T>(self: model.Box, value: T) -> T:
        return value

impl model.Box: model.Mapper

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "parameter 2 ownership differs: expected 'consume', got 'owned'")
}

func TestOwnershipRejectsBorrowDerivedValueAsInoutArgument(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func mutate(x: inout []u8) -> Int:
    x[0] = 1
    return 0

func caller(x: borrow []u8) -> Int:
    var y: []u8 = x
    return mutate(y)

func main() -> Int:
    return 0
`, "cannot be passed as inout")
}

func TestOwnershipRejectsBorrowEscapeViaStructLiteralReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct BufBox:
    buf: []u8

func leak(x: borrow []u8) -> BufBox:
    return BufBox(buf: x)

func main() -> Int:
    return 0
	`, "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return")
}

func TestOwnershipRejectsBorrowEscapeViaStructAliasReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct BufBox:
    buf: []u8

func leak(x: borrow []u8) -> BufBox:
    let box: BufBox = BufBox(buf: x)
    return box

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsCrossModuleBorrowEscapeViaStructLiteralReturn(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func leak(x: borrow []u8) -> BufBox:
    return BufBox(buf: x)
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return")
}

func TestOwnershipRejectsCrossModuleBorrowEscapeViaStructAliasReturn(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func leak(x: borrow []u8) -> BufBox:
    let box: BufBox = BufBox(buf: x)
    return box
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaStructLiteralReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func leak(x: borrow ptr) -> PtrBox:
    return PtrBox(raw: x)

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaStructAliasReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func leak(x: borrow ptr) -> PtrBox:
    let box: PtrBox = PtrBox(raw: x)
    return box

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaStructFieldAssignmentReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func leak(x: borrow ptr) -> PtrBox:
    var box: PtrBox = PtrBox(raw: 0)
    box.raw = x
    return box

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaOptionalAssignmentReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(x: borrow ptr) -> ptr?:
    var maybe: ptr? = none
    maybe = x
    return maybe

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrOptionalGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var leaked: ptr? = none

func leak(x: borrow ptr) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrAggregateOptionalGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

var leaked: PtrBox? = none

func leak(box: borrow PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'box' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedSliceEscapeViaOptionalAssignmentReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(x: borrow []u8) -> []u8?:
    var maybe: []u8? = none
    maybe = x
    return maybe

func main() -> Int:
    return 0
`, "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot escape through owned return")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaOptionalAssignmentIfLetGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var leaked: ptr = 0

func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaOptionalAssignmentMatchGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var leaked: ptr = 0

func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaOptionalAssignmentOwnedCall(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink(value: ptr?) -> Int:
    return 0

func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaOptionalAssignmentConsumeCall(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink(value: consume ptr?) -> Int:
    return 0

func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be consumed by 'sink'")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaOptionalAssignmentInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(x: borrow ptr, out: inout ptr?) -> Int:
    var maybe: ptr? = none
    maybe = x
    out = maybe
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowedSliceEscapeViaOptionalAssignmentOwnedCall(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink(value: []u8?) -> Int:
    return 0

func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'")
}

func TestOwnershipRejectsBorrowedSliceEscapeViaOptionalAssignmentConsumeCall(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink(value: consume []u8?) -> Int:
    return 0

func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`, "borrowed value derived from 'x' cannot be consumed by 'sink'")
}

func TestOwnershipRejectsBorrowedSliceEscapeViaOptionalAssignmentInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(x: borrow []u8, out: inout []u8?) -> Int:
    var maybe: []u8? = none
    maybe = x
    out = maybe
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowedPtrAggregateParamReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    return box

func main() -> Int:
    return 0
`, "borrowed local 'box' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrAggregateParamFieldReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> ptr:
    return box.raw

func main() -> Int:
    return 0
`, "borrowed local 'box' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrAggregateParamAliasReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    let alias: PtrBox = box
    return alias

func main() -> Int:
    return 0
`, "borrowed local 'box' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrAggregateParamInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox, out: inout PtrBox) -> Int:
    out = box
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'box' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowedPtrNestedAggregateParamInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

func leak(outer: borrow OuterBox, out: inout OuterBox) -> Int:
    out = outer
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'outer' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowedPtrAggregateParamGlobalFieldAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

var leaked: ptr = 0

func leak(box: borrow PtrBox) -> Int:
    leaked = box.raw
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'box' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrAggregateParamGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

var leaked: PtrBox

func leak(box: borrow PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'box' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrAggregateParamGlobalFieldTargetAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

var leaked: PtrBox

func leak(box: borrow PtrBox) -> Int:
    leaked.raw = box.raw
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'box' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrNestedAggregateParamGlobalFieldAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

var leaked: ptr = 0

func leak(outer: borrow OuterBox) -> Int:
    leaked = outer.box.raw
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'outer' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrEnumParamPayloadBindingReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PtrMsg:
    case raw(ptr)
    case empty

func leak(msg: borrow PtrMsg) -> ptr:
    match msg:
    case PtrMsg.raw(raw):
        return raw
    case PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'msg' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrEnumParamPayloadBindingOwnedCall(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PtrMsg:
    case raw(ptr)
    case empty

func sink(raw: ptr) -> Int:
    return 0

func leak(msg: borrow PtrMsg) -> Int:
    match msg:
    case PtrMsg.raw(raw):
        return sink(raw)
    case PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`, "borrowed value derived from 'msg' cannot be passed to non-borrow parameter 1 of 'sink'")
}

func TestOwnershipRejectsBorrowedPtrEnumParamPayloadBindingGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum PtrMsg:
    case raw(ptr)
    case empty

var leaked: ptr = 0

func leak(msg: borrow PtrMsg) -> Int:
    match msg:
    case PtrMsg.raw(raw):
        leaked = raw
        return 0
    case PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'msg' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrEnumParamGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum PtrMsg:
    case raw(ptr)
    case empty

var leaked: PtrMsg

func leak(msg: borrow PtrMsg) -> Int:
    leaked = msg
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'msg' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrEnumParamPayloadBindingInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PtrMsg:
    case raw(ptr)
    case empty

func leak(msg: borrow PtrMsg, out: inout ptr) -> Int:
    match msg:
    case PtrMsg.raw(raw):
        out = raw
        return 0
    case PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'msg' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEnumParamPayloadBindingReturn(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func leak(msg: borrow model.PtrMsg) -> ptr:
    match msg:
    case model.PtrMsg.raw(raw):
        return raw
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'msg' cannot escape via return")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEnumParamPayloadBindingOwnedCall(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func sink(raw: ptr) -> Int:
    return 0

func leak(msg: borrow model.PtrMsg) -> Int:
    match msg:
    case model.PtrMsg.raw(raw):
        return sink(raw)
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'msg' cannot be passed to non-borrow parameter 1 of 'app.main.sink'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEnumParamPayloadBindingGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

var leaked: ptr = 0

func leak(msg: borrow model.PtrMsg) -> Int:
    match msg:
    case model.PtrMsg.raw(raw):
        leaked = raw
        return 0
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'msg' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEnumParamGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

var leaked: model.PtrMsg

func leak(msg: borrow model.PtrMsg) -> Int:
    leaked = msg
    return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'msg' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEnumParamPayloadBindingInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`,
		"app/main.t4": `module app.main
import lib.model as model

func leak(msg: borrow model.PtrMsg, out: inout ptr) -> Int:
    match msg:
    case model.PtrMsg.raw(raw):
        out = raw
        return 0
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'msg' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowedPtrOptionalIfLetBindingReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(maybe: borrow ptr?) -> ptr:
    if let raw = maybe:
        return raw
    else:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'maybe' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrOptionalMatchBindingReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(maybe: borrow ptr?) -> ptr:
    match maybe:
    case some(raw):
        return raw
    case none:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'maybe' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrOptionalMatchBindingOwnedCall(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink(raw: ptr) -> Int:
    return 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`, "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'sink'")
}

func TestOwnershipRejectsBorrowedPtrOptionalMatchBindingConsumedCall(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func sink(raw: consume ptr) -> Int:
    return 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`, "borrowed value derived from 'maybe' cannot be consumed by 'sink'")
}

func TestOwnershipRejectsBorrowedPtrOptionalMatchBindingInoutCall(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func mutate(raw: inout ptr) -> Int:
    raw = raw
    return 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        var alias: ptr = raw
        return mutate(alias)
    case none:
        return 0

func main() -> Int:
    return 0
`, "borrowed value derived from 'maybe' cannot be passed as inout to 'mutate'")
}

func TestOwnershipRejectsBorrowedPtrOptionalIfLetBindingGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var leaked: ptr = 0

func leak(maybe: borrow ptr?) -> Int:
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'maybe' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrOptionalMatchBindingGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var leaked: ptr = 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'maybe' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrOptionalIfLetBindingInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'maybe' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowedPtrOptionalMatchBindingInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'maybe' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrOptionalIfLetBindingReturn(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

pub func leak(maybe: borrow ptr?) -> ptr:
    if let raw = maybe:
        return raw
    else:
        return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'maybe' cannot escape via return")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrOptionalMatchBindingReturn(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

pub func leak(maybe: borrow ptr?) -> ptr:
    match maybe:
    case some(raw):
        return raw
    case none:
        return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'maybe' cannot escape via return")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrOptionalMatchBindingImportedOwnedCall(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub func sink(raw: ptr) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sinker.sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'lib.sink.sink'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrOptionalMatchBindingImportedConsumedCall(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub func sink(raw: consume ptr) -> Int:
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sinker.sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be consumed by 'lib.sink.sink'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrOptionalMatchBindingImportedInoutCall(t *testing.T) {
	files := map[string]string{
		"lib/sink.t4": `module lib.sink

pub func mutate(raw: inout ptr) -> Int:
    raw = raw
    return 0
`,
		"app/main.t4": `module app.main
import lib.sink as sinker

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        var alias: ptr = raw
        return sinker.mutate(alias)
    case none:
        return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed value derived from 'maybe' cannot be passed as inout to 'lib.sink.mutate'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrOptionalIfLetBindingGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

var leaked: ptr = 0

pub func leak(maybe: borrow ptr?) -> Int:
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'maybe' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrOptionalMatchBindingGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

var leaked: ptr = 0

pub func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'maybe' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrOptionalIfLetBindingInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

pub func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'maybe' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrOptionalMatchBindingInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

pub func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'maybe' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowedSliceOptionalPayloadBindingCallEscapes(t *testing.T) {
	tests := []struct {
		name     string
		callee   string
		call     string
		wantText string
	}{
		{
			name: "owned",
			callee: `func sink(raw: []u8) -> Int:
    return 0
`,
			call:     "return sink(raw)",
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "consume",
			callee: `func sink(raw: consume []u8) -> Int:
    return 0
`,
			call:     "return sink(raw)",
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			callee: `func sink(raw: inout []u8) -> Int:
    raw = raw
    return 0
`,
			call:     "return sink(raw)",
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.callee+`
func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        `+tt.call+`
    case none:
        return 0

func main() -> Int:
    return 0
`, tt.wantText)
		})
	}
}

func TestOwnershipRejectsBorrowedSliceOptionalPayloadBindingInoutAssignment(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "if-let",
			body: `func leak(maybe: borrow []u8?, out: inout []u8) -> Int:
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0
`,
		},
		{
			name: "match",
			body: `func leak(maybe: borrow []u8?, out: inout []u8) -> Int:
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.body+`
func main() -> Int:
    return 0
`, "borrowed local 'maybe' cannot escape via inout assignment to 'out'")
		})
	}
}

func TestOwnershipRejectsBorrowedSliceOptionalPayloadGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `var leaked: []u8? = none

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`, "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot be stored in global")
}

func TestOwnershipRejectsCrossModuleBorrowedSliceOptionalPayloadGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

var leaked: []u8? = none

pub func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot be stored in global")
}

func TestOwnershipRejectsCrossModuleBorrowedSliceGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

var leaked: []u8

pub func leak(x: borrow []u8) -> Int:
    leaked = x
    return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrOptionalGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

var leaked: ptr? = none

pub func leak(x: borrow ptr) -> Int:
    leaked = x
    return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregateOptionalGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr
`,
		"lib/leaks.t4": `module lib.leaks
import lib.model as model

var leaked: model.PtrBox? = none

pub func leak(box: borrow model.PtrBox) -> Int:
    leaked = box
    return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'box' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedSliceOptionalPayloadBindingImportedCallEscapes(t *testing.T) {
	tests := []struct {
		name     string
		callee   string
		wantText string
	}{
		{
			name: "owned",
			callee: `pub func sink(raw: []u8) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'lib.sink.sink'",
		},
		{
			name: "consume",
			callee: `pub func sink(raw: consume []u8) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'lib.sink.sink'",
		},
		{
			name: "inout",
			callee: `pub func sink(raw: inout []u8) -> Int:
    raw = raw
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'lib.sink.sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": `module lib.sink

` + tt.callee,
				"app/main.t4": `module app.main
import lib.sink as sinker

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        return sinker.sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.wantText)
		})
	}
}

func TestOwnershipRejectsCrossModuleBorrowedSliceOptionalPayloadBindingInoutAssignment(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "if-let",
			body: `pub func leak(maybe: borrow []u8?, out: inout []u8) -> Int:
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0
`,
		},
		{
			name: "match",
			body: `pub func leak(maybe: borrow []u8?, out: inout []u8) -> Int:
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": `module lib.leaks

` + tt.body,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'maybe' cannot escape via inout assignment to 'out'")
		})
	}
}

func TestOwnershipRejectsWholeOptionalUseAfterPayloadConsume(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func take(raw: consume ptr) -> ptr:
    return raw

func use(value: ptr?) -> Int:
    return 0

func leak(maybe: ptr?) -> Int:
    match maybe:
    case some(raw):
        let moved: ptr = take(raw)
    case none:
        let untouched: Int = 0
    return use(maybe)

func main() -> Int:
    return 0
`, "cannot use consumed value 'maybe.$elem'")
}

func TestOwnershipRejectsCrossModuleWholeOptionalUseAfterPayloadConsume(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub func pass(maybe: ptr?) -> ptr?:
    return maybe
`,
		"app/main.t4": `module app.main
import lib.model as model

func take(raw: consume ptr) -> ptr:
    return raw

func use(value: ptr?) -> Int:
    return 0

func leak(maybe: ptr?) -> Int:
    let returned: ptr? = model.pass(maybe)
    match returned:
    case some(raw):
        let moved: ptr = take(raw)
    case none:
        let untouched: Int = 0
    return use(returned)

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'returned.$elem'")
}

func TestOwnershipRejectsBorrowEscapeViaStructInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct BufBox:
    buf: []u8

func leak(read: borrow []u8, out: inout BufBox) -> Int:
    out = BufBox(buf: read)
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'read' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsCrossModuleBorrowEscapeViaStructInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func leak(read: borrow []u8, out: inout BufBox) -> Int:
    out = BufBox(buf: read)
    return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'read' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowEscapeViaNestedSliceStruct(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal-return",
			src: `
struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox

func leak(x: borrow []u8) -> OuterBox:
    return OuterBox(box: BufBox(buf: x))

func main() -> Int:
    return 0
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			src: `
struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox

func leak(x: borrow []u8) -> OuterBox:
    let outer: OuterBox = OuterBox(box: BufBox(buf: x))
    return outer

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			src: `
struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox

func leak(read: borrow []u8, out: inout OuterBox) -> Int:
    out = OuterBox(box: BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.wantText)
		})
	}
}

func TestOwnershipRejectsCrossModuleBorrowEscapeViaNestedSliceStruct(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "literal-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(x: borrow []u8) -> OuterBox:
    return OuterBox(box: BufBox(buf: x))
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(x: borrow []u8) -> OuterBox:
    let outer: OuterBox = OuterBox(box: BufBox(buf: x))
    return outer
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(read: borrow []u8, out: inout OuterBox) -> Int:
    out = OuterBox(box: BufBox(buf: read))
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": tt.libSrc,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", tt.wantText)
		})
	}
}

func TestOwnershipRejectsBorrowEscapeViaNestedSliceEnumPayload(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal-return",
			src: `
struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty

func leak(x: borrow []u8) -> OuterMsg:
    return OuterMsg.wrap(BufBox(buf: x))

func main() -> Int:
    return 0
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			src: `
struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty

func leak(x: borrow []u8) -> OuterMsg:
    let msg: OuterMsg = OuterMsg.wrap(BufBox(buf: x))
    return msg

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			src: `
struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty

func leak(read: borrow []u8, out: inout OuterMsg) -> Int:
    out = OuterMsg.wrap(BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.wantText)
		})
	}
}

func TestOwnershipRejectsCrossModuleBorrowEscapeViaNestedSliceEnumPayload(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "literal-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(x: borrow []u8) -> OuterMsg:
    return OuterMsg.wrap(BufBox(buf: x))
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(x: borrow []u8) -> OuterMsg:
    let msg: OuterMsg = OuterMsg.wrap(BufBox(buf: x))
    return msg
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(read: borrow []u8, out: inout OuterMsg) -> Int:
    out = OuterMsg.wrap(BufBox(buf: read))
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": tt.libSrc,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", tt.wantText)
		})
	}
}

func TestOwnershipRejectsBorrowedSliceStructOwnedConsumeInoutCallEscape(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "owned",
			src: `
struct BufBox:
    buf: []u8

func sink(value: BufBox) -> Int:
    return 0

func caller(x: borrow []u8) -> Int:
    return sink(BufBox(buf: x))

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "consume",
			src: `
struct BufBox:
    buf: []u8

func sink(value: consume BufBox) -> Int:
    return 0

func caller(x: borrow []u8) -> Int:
    let box: BufBox = BufBox(buf: x)
    return sink(box)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			src: `
struct BufBox:
    buf: []u8

func mutate(value: inout BufBox) -> Int:
    value = value
    return 0

func caller(x: borrow []u8) -> Int:
    var box: BufBox = BufBox(buf: x)
    return mutate(box)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'mutate'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.wantText)
		})
	}
}

func TestOwnershipRejectsCrossModuleBorrowedSliceStructOwnedConsumeInoutCallEscape(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "owned",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func sink(value: BufBox) -> Int:
    return 0

pub func caller(x: borrow []u8) -> Int:
    return sink(BufBox(buf: x))
`,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "consume",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func sink(value: consume BufBox) -> Int:
    return 0

pub func caller(x: borrow []u8) -> Int:
    let box: BufBox = BufBox(buf: x)
    return sink(box)
`,
			wantText: "borrowed value derived from 'x' cannot be consumed",
		},
		{
			name: "inout",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func mutate(value: inout BufBox) -> Int:
    value = value
    return 0

pub func caller(x: borrow []u8) -> Int:
    var box: BufBox = BufBox(buf: x)
    return mutate(box)
`,
			wantText: "borrowed value derived from 'x' cannot be passed as inout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": tt.libSrc,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", tt.wantText)
		})
	}
}

func TestOwnershipRejectsImportedBorrowedSliceStructOwnedConsumeInoutCallEscape(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		appCall  string
		wantText string
	}{
		{
			name: "owned",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func sink(value: BufBox) -> Int:
    return 0
`,
			appCall:  "return sinker.sink(sinker.BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'lib.sink.sink'",
		},
		{
			name: "consume",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func sink(value: consume BufBox) -> Int:
    return 0
`,
			appCall:  "let box: sinker.BufBox = sinker.BufBox(buf: x)\n    return sinker.sink(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.sink.sink'",
		},
		{
			name: "inout",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func mutate(value: inout BufBox) -> Int:
    value = value
    return 0
`,
			appCall:  "var box: sinker.BufBox = sinker.BufBox(buf: x)\n    return sinker.mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.sink.mutate'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": tt.libSrc,
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow []u8) -> Int:
    ` + tt.appCall + `

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.wantText)
		})
	}
}

func TestOwnershipRejectsBorrowedPtrEscapeViaScalarInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(read: borrow ptr, out: inout ptr) -> Int:
    out = read
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'read' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEscapeViaScalarInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

pub func leak(read: borrow ptr, out: inout ptr) -> Int:
    out = read
    return 0
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'read' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaStructInoutAssignment(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct PtrBox:
    raw: ptr

func leak(read: borrow ptr, out: inout PtrBox) -> Int:
    out = PtrBox(raw: read)
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'read' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEscapeViaStructInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr
`,
		"app/main.t4": `module app.main
import lib.model as model

func leak(read: borrow ptr, out: inout model.PtrBox) -> Int:
    out = model.PtrBox(raw: read)
    return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'read' cannot escape via inout assignment to 'out'")
}

func TestOwnershipRejectsBorrowEscapeViaGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var leaked: ptr = 0

func leak(x: borrow ptr) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowedPtrAliasEscapeViaGlobalAssignment(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var leaked: ptr = 0

func leak(x: borrow ptr) -> Int:
    let y: ptr = x
    leaked = y
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregateParamGlobalFieldAssignment(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr
`,
		"app/main.t4": `module app.main
import lib.model as model

var leaked: ptr = 0

func leak(box: borrow model.PtrBox) -> Int:
    leaked = box.raw
    return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'box' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregateParamGlobalAssignment(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr
`,
		"app/main.t4": `module app.main
import lib.model as model

var leaked: model.PtrBox

func leak(box: borrow model.PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'box' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregateParamGlobalFieldTargetAssignment(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr
`,
		"app/main.t4": `module app.main
import lib.model as model

var leaked: model.PtrBox

func leak(box: borrow model.PtrBox) -> Int:
    leaked.raw = box.raw
    return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'box' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrNestedAggregateParamGlobalFieldAssignment(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox
`,
		"app/main.t4": `module app.main
import lib.model as model

var leaked: ptr = 0

func leak(outer: borrow model.OuterBox) -> Int:
    leaked = outer.box.raw
    return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'outer' cannot escape via global assignment to 'leaked'")
}

func TestOwnershipRejectsBorrowEscapeViaEnumReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum BufMsg:
    case send([]u8)

func leak(x: borrow []u8) -> BufMsg:
    return BufMsg.send(x)

func main() -> Int:
    return 0
	`, "aggregate 'BufMsg' contains borrowed slice field 'BufMsg.send[1]' that cannot escape through owned return")
}

func TestOwnershipRejectsBorrowEscapeViaEnumAliasReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum BufMsg:
    case send([]u8)

func leak(x: borrow []u8) -> BufMsg:
    let msg: BufMsg = BufMsg.send(x)
    return msg

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaMatchExpressionReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func leak(x: borrow ptr, flag: Int) -> ptr:
    return match flag:
    case 1:
        x
    case _:
        0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaCatchExpressionReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum ReadErr:
    case miss

func read() -> ptr throws ReadErr:
    throw ReadErr.miss

func leak(x: borrow ptr) -> ptr:
    return catch read():
    case ReadErr.miss:
        x

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsBorrowedPtrEscapeViaThrowPayload(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum LeakErr:
    case raw(ptr)

func leak(x: borrow ptr) -> Int throws LeakErr:
    throw LeakErr.raw(x)

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via throw")
}

func TestOwnershipRejectsScopedIslandSliceEscapeViaThrowPayload(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum LeakErr:
    case buf([]u8)

func leak() -> Int throws LeakErr
uses alloc, islands, mem:
    island(16) as isl:
        let buf: []u8 = core.island_make_u8(isl, 4)
        throw LeakErr.buf(buf)
    return 0

func main() -> Int:
    return 0
`, "slice from scoped island cannot escape")
}

func TestOwnershipRejectsUseAfterConsumeViaThrowPayload(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum LeakErr:
    case value(Int)

func take(x: consume Int) -> Int:
    return x

func fail() -> Int throws LeakErr:
    let value: Int = 1
    let moved: Int = take(value)
    throw LeakErr.value(value)

func main() -> Int:
    return 0
`, "cannot use consumed value 'value'")
}

func TestOwnershipRejectsUseAfterConsumeViaMatchExpressionReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func take(x: consume Int) -> Int:
    return x

func leak(flag: Int) -> Int:
    let value: Int = 1
    let moved: Int = take(value)
    return match flag:
    case 1:
        value
    case _:
        0

func main() -> Int:
    return 0
`, "cannot use consumed value 'value'")
}

func TestOwnershipRejectsCrossModuleBorrowEscapeViaEnumReturn(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

pub enum BufMsg:
    case send([]u8)

pub func leak(x: borrow []u8) -> BufMsg:
    return BufMsg.send(x)
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "aggregate 'BufMsg' contains borrowed slice field 'BufMsg.send[1]' that cannot escape through owned return")
}

func TestOwnershipRejectsCrossModuleBorrowEscapeViaEnumAliasReturn(t *testing.T) {
	files := map[string]string{
		"lib/leaks.t4": `module lib.leaks

pub enum BufMsg:
    case send([]u8)

pub func leak(x: borrow []u8) -> BufMsg:
    let msg: BufMsg = BufMsg.send(x)
    return msg
`,
		"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsBorrowedSliceEnumOwnedConsumeInoutCallEscape(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "owned",
			src: `
enum BufMsg:
    case send([]u8)

func sink(value: BufMsg) -> Int:
    return 0

func caller(x: borrow []u8) -> Int:
    return sink(BufMsg.send(x))

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "consume",
			src: `
enum BufMsg:
    case send([]u8)

func sink(value: consume BufMsg) -> Int:
    return 0

func caller(x: borrow []u8) -> Int:
    let msg: BufMsg = BufMsg.send(x)
    return sink(msg)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			src: `
enum BufMsg:
    case send([]u8)

func mutate(value: inout BufMsg) -> Int:
    value = value
    return 0

func caller(x: borrow []u8) -> Int:
    var msg: BufMsg = BufMsg.send(x)
    return mutate(msg)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'mutate'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.wantText)
		})
	}
}

func TestOwnershipRejectsCrossModuleBorrowedSliceEnumOwnedConsumeInoutCallEscape(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "owned",
			libSrc: `module lib.leaks

pub enum BufMsg:
    case send([]u8)

pub func sink(value: BufMsg) -> Int:
    return 0

pub func caller(x: borrow []u8) -> Int:
    return sink(BufMsg.send(x))
`,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "consume",
			libSrc: `module lib.leaks

pub enum BufMsg:
    case send([]u8)

pub func sink(value: consume BufMsg) -> Int:
    return 0

pub func caller(x: borrow []u8) -> Int:
    let msg: BufMsg = BufMsg.send(x)
    return sink(msg)
`,
			wantText: "borrowed value derived from 'x' cannot be consumed",
		},
		{
			name: "inout",
			libSrc: `module lib.leaks

pub enum BufMsg:
    case send([]u8)

pub func mutate(value: inout BufMsg) -> Int:
    value = value
    return 0

pub func caller(x: borrow []u8) -> Int:
    var msg: BufMsg = BufMsg.send(x)
    return mutate(msg)
`,
			wantText: "borrowed value derived from 'x' cannot be passed as inout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": tt.libSrc,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", tt.wantText)
		})
	}
}

func TestOwnershipRejectsImportedBorrowedSliceEnumOwnedConsumeInoutCallEscape(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		appCall  string
		wantText string
	}{
		{
			name: "owned",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func sink(value: BufMsg) -> Int:
    return 0
`,
			appCall:  "return sinker.sink(sinker.BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'lib.sink.sink'",
		},
		{
			name: "consume",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func sink(value: consume BufMsg) -> Int:
    return 0
`,
			appCall:  "let msg: sinker.BufMsg = sinker.BufMsg.send(x)\n    return sinker.sink(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.sink.sink'",
		},
		{
			name: "inout",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func mutate(value: inout BufMsg) -> Int:
    value = value
    return 0
`,
			appCall:  "var msg: sinker.BufMsg = sinker.BufMsg.send(x)\n    return sinker.mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.sink.mutate'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": tt.libSrc,
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow []u8) -> Int:
    ` + tt.appCall + `

func main() -> Int:
    return 0
`,
			}
			requireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.wantText)
		})
	}
}

func TestOwnershipRejectsBorrowedPtrEscapeViaEnumAliasReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PtrMsg:
    case raw(ptr)

func leak(x: borrow ptr) -> PtrMsg:
    let msg: PtrMsg = PtrMsg.raw(x)
    return msg

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestOwnershipRejectsCrossModuleBorrowedPtrEscapeViaEnumAliasReturn(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
`,
		"app/main.t4": `module app.main
import lib.model as model

func leak(x: borrow ptr) -> model.PtrMsg:
    let msg: model.PtrMsg = model.PtrMsg.raw(x)
    return msg

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestOwnershipAllowsStructFieldsFromDistinctBorrowRegions(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct PairBuf:
    left: []u8
    right: []u8

func read(buf: borrow []u8) -> Int:
    return 0

func inspect(left: borrow []u8, right: borrow []u8) -> Int:
    let pair: PairBuf = PairBuf(left: left, right: right)
    return read(pair.left) + read(pair.right)

func main() -> Int:
    return 0
`)
}

func TestOwnershipRejectsBorrowEscapeViaEnumPayloadBinding(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum PairMsg:
    case both([]u8, []u8)

func sink(buf: []u8) -> Int:
    return 0

func leak(left: borrow []u8, right: borrow []u8) -> Int:
    let msg: PairMsg = PairMsg.both(left, right)
    match msg:
    case PairMsg.both(a, b):
        return sink(a)

func main() -> Int:
    return 0
`, "borrowed value derived from 'left'")
}

func TestOwnershipRejectsBorrowInoutAliasThroughFieldRoot(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Cell:
    value: Int

func mix(read: borrow Cell, write: inout Int) -> Int:
    write = write + read.value
    return write

func main() -> Int:
    var cell: Cell = Cell(value: 1)
    return mix(cell, cell.value)
`, "aliases borrowed argument")
}

func TestOwnershipAllowsBorrowInoutDistinctStructFields(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Pair:
    left: Int
    right: Int

func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var pair: Pair = Pair(left: 1, right: 2)
    return mix(pair.left, pair.right)
`)
}

func TestOwnershipRejectsBorrowedProjectionAsInoutArgument(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct BufBox:
    buf: []u8

func mutate(buf: inout []u8) -> Int:
    buf[0] = 1
    return 0

func caller(box: borrow BufBox) -> Int:
    return mutate(box.buf)

func main() -> Int:
    return 0
`, "borrowed value derived from 'box' cannot be passed as inout to 'mutate'")
}

func TestOwnershipTracksInterproceduralStructReturnResourceLeaves(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
struct IslandPair:
    left: island
    right: island

func forward_pair(pair: IslandPair) -> IslandPair:
    return pair

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let pair: IslandPair = IslandPair(left: core.island_new(16), right: core.island_new(32))
        let alias: IslandPair = forward_pair(pair)
        free(alias.right)
    }
    return 0
`)
}

func TestOwnershipRejectsInterproceduralStructReturnResourceLeafDoubleFree(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
struct IslandPair:
    left: island
    right: island

func forward_pair(pair: IslandPair) -> IslandPair:
    return pair

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let pair: IslandPair = IslandPair(left: core.island_new(16), right: core.island_new(32))
        let alias: IslandPair = forward_pair(pair)
        free(pair.left)
        free(alias.left)
    }
    return 0
`, "cannot use freed resource 'alias.left'")
}

func TestOwnershipRejectsIndirectRecursiveEnumPayloadCycle(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
enum A:
    case b(B)

enum B:
    case a(A)

func id(a: A) -> A:
    return a

func main() -> Int:
    return 0
`, "recursive enum payload")
}

func TestOwnershipReturnThrowTerminalBranchesDoNotPoisonFallthroughFlow(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
enum FlowErr:
    case done

func take(x: consume Int) -> Int:
    return x

func return_branch(flag: Int) -> Int:
    let value: Int = 1
    if flag:
        return take(value)
    return value

func throw_branch(flag: Int) -> Int throws FlowErr:
    let value: Int = 1
    if flag:
        let _: Int = take(value)
        throw FlowErr.done
    return value

func main() -> Int:
    return return_branch(0)
`)
}

func TestOwnershipLoopBreakExitReportsMaybeConsumedWithBreakLabel(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let value: Int = 1
    while 1:
        let moved: Int = take(value)
        break
    return value
`, "break: consumed")
}

func TestOwnershipLoopContinueExitReportsMaybeConsumedWithContinueLabel(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let value: Int = 1
    while 1:
        let moved: Int = take(value)
        continue
    return value
`, "continue: consumed")
}

func TestOwnershipLoopBreakMakesFollowingBodyStatementsUnreachable(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let value: Int = 1
    while 1:
        break
        let moved: Int = take(value)
    return value
`)
}
