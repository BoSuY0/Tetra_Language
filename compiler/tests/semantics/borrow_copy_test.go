package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func TestSliceAndStringBorrowTypeCheckWithoutAllocEffect(t *testing.T) {
	testkit.RequireCheckOK(t, `
func slice_len(xs: []i32) -> Int:
    let b: []i32 = xs.window(1, 2).borrow()
    return b.len

func string_len() -> Int:
    let b: String = "abcdef".window(1, 3).borrow()
    return b.len

func main() -> Int:
    return string_len()
`)
}

func TestBuildSliceCopyCreatesIndependentOwnedStorage(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 10
    xs[1] = 20
    xs[2] = 30
    xs[3] = 40
    let ys: []i32 = xs.window(1, 2).copy()
    xs[1] = 99
    if ys.len != 2:
        return 1
    if ys[0] != 20:
        return 2
    if ys[1] != 30:
        return 3
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStringCopyCreatesIndependentOwnedStorage(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    let text: String = "abcdef"
    let mid: String = text.window(1, 3).copy()
    if mid.len != 3:
        return 1
    if mid[0] != 98:
        return 2
    if mid[1] != 99:
        return 3
    if mid[2] != 100:
        return 4
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildCopyIntoMutatesDestinationAndReturnsCount(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var src: []u8 = make_u8(2)
    src[0] = 65
    src[1] = 66
    var dst: []u8 = make_u8(2)
    dst[0] = 1
    dst[1] = 2
    let n: Int = src.copy_into(dst)
    if n != 2:
        return 1
    if dst[0] != 65:
        return 2
    if dst[1] != 66:
        return 3
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStringCopyIntoMutatesDestinationAndReturnsCount(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    let text: String = "abcdef".window(1, 3)
    var dst: []u8 = make_u8(3)
    let n: Int = text.copy_into(dst)
    if n != 3:
        return 1
    if dst[0] != 98:
        return 2
    if dst[1] != 99:
        return 3
    if dst[2] != 100:
        return 4
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildCopyIntoRejectsInsufficientDestination(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var src: []u8 = make_u8(2)
    var dst: []u8 = make_u8(1)
    return src.copy_into(dst)
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode == 0 || exitCode == 42 {
		t.Fatalf("insufficient destination exited %d, want trap/non-success", exitCode)
	}
}

func TestBuildCopyIntoZeroLengthSucceedsWithoutTouchingDestination(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var src: []u8 = make_u8(0)
    var dst: []u8 = make_u8(1)
    dst[0] = 77
    let n: Int = src.copy_into(dst)
    if n != 0:
        return 1
    if dst[0] != 77:
        return 2
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBorrowedSliceAndStringEscapeDiagnostics(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func leak(xs: []i32) -> []i32:
    let b: []i32 = xs.borrow()
    return b

func main() -> Int:
    return 0
`, "borrowed slice return requires '-> borrow []i32' or '.copy()'")

	testkit.RequireFileCheckErrorContains(t, `
func leak(text: String) -> String:
    return text.window(1, 2).borrow()

func main() -> Int:
    return 0
`, "borrowed String return requires '-> borrow String' or '.copy()'")
}

func TestBorrowedSliceAndStringBorrowedReturnContracts(t *testing.T) {
	testkit.RequireCheckOK(t, `
func view_bytes(xs: borrow []u8) -> borrow []u8:
    return xs.window(1, 2).borrow()

func view_u16(xs: borrow []u16) -> borrow []u16:
    return xs.window(1, 2).borrow()

func view_i32(xs: borrow []i32) -> borrow []i32:
    return xs.window(1, 2).borrow()

func view_bool(xs: borrow []bool) -> borrow []bool:
    return xs.window(1, 2).borrow()

func view_text(text: borrow String) -> borrow String:
    return text.window(1, 2).borrow()

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckErrorContains(t, `
func leak_owned(xs: borrow []u8) -> []u8:
    return xs.window(0, 1).borrow()

func main() -> Int:
    return 0
`, "borrowed slice return requires '-> borrow []u8' or '.copy()'")

	testkit.RequireFileCheckErrorContains(t, `
func leak_local() -> borrow []u8
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    return xs.window(0, 1).borrow()

func main() -> Int:
    return 0
`, "borrowed slice return derives from local owner 'xs'")
}

func TestFunctionTypedBorrowedReturnOwnershipContract(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Holder:
    cb: fn(borrow []u8) -> borrow []u8

enum Choice:
    case cb(fn(borrow []u8) -> borrow []u8)

func borrowed_view(xs: borrow []u8) -> borrow []u8:
    return xs.borrow()

func apply(cb: fn(borrow []u8) -> borrow []u8, xs: borrow []u8) -> borrow []u8:
    return cb(xs)

func from_local(xs: borrow []u8) -> borrow []u8:
    let cb: fn(borrow []u8) -> borrow []u8 = borrowed_view
    return cb(xs)

func from_field(xs: borrow []u8, holder: Holder) -> borrow []u8:
    return holder.cb(xs)

func from_enum(xs: borrow []u8) -> Choice:
    return Choice.cb(borrowed_view)

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckErrorContains(t, `
func owned_copy(xs: borrow []u8) -> []u8
uses alloc, mem:
    return xs.copy()

func apply(xs: borrow []u8, cb: fn(borrow []u8) -> borrow []u8 uses alloc, mem) -> borrow []u8
uses alloc, mem:
    return cb(xs)

func bad(xs: borrow []u8) -> borrow []u8
uses alloc, mem:
    return apply(xs, owned_copy)

func main() -> Int:
    return 0
`, "callback function symbol 'owned_copy' return ownership mismatch: expected 'borrow', got 'owned'")

	testkit.RequireFileCheckErrorContains(t, `
func owned_copy(xs: borrow []u8) -> []u8
uses alloc, mem:
    return xs.copy()

func main() -> Int
uses alloc, mem:
    let cb: fn(borrow []u8) -> borrow []u8 uses alloc, mem = owned_copy
    return 0
`, "function-typed local 'cb' return ownership mismatch: expected 'borrow', got 'owned'")

	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, mem:
    let cb: fn(borrow []u8) -> borrow []u8 uses alloc, mem = fn(xs: borrow []u8) -> []u8
    uses alloc, mem:
        return xs.copy()
    return 0
`, "function-typed local 'cb' return ownership mismatch: expected 'borrow', got 'owned'")

	testkit.RequireFileCheckErrorContains(t, `
var cb: fn(borrow []u8) -> borrow []u8 uses alloc, mem = fn(xs: borrow []u8) -> []u8
uses alloc, mem:
    return xs.copy()

func main() -> Int:
    return 0
`, "function-typed local 'cb' return ownership mismatch: expected 'borrow', got 'owned'")

	testkit.RequireFileCheckErrorContains(t, `
struct Holder:
    cb: fn(borrow []u8) -> borrow []u8 uses alloc, mem

func main() -> Int
uses alloc, mem:
    let h: Holder = Holder(cb: fn(xs: borrow []u8) -> []u8
    uses alloc, mem:
        return xs.copy()
    )
    return 0
`, "function-typed assignment to 'h.cb' return ownership mismatch: expected 'borrow', got 'owned'")

	testkit.RequireFileCheckErrorContains(t, `
enum Choice:
    case cb(fn(borrow []u8) -> borrow []u8 uses alloc, mem)

func main() -> Int
uses alloc, mem:
    let c: Choice = Choice.cb(fn(xs: borrow []u8) -> []u8
    uses alloc, mem:
        return xs.copy()
    )
    return 0
`, "function-typed assignment to 'Choice.cb[1]' return ownership mismatch: expected 'borrow', got 'owned'")
}

func TestBorrowedReturnForwardingRequiresBorrowReturn(t *testing.T) {
	testkit.RequireCheckOK(t, `
func inner(xs: borrow []u8) -> borrow []u8:
    return xs.window(1, 2).borrow()

func outer(xs: borrow []u8) -> borrow []u8:
    return inner(xs)

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckErrorContains(t, `
func inner(xs: borrow []u8) -> borrow []u8:
    return xs.window(1, 2).borrow()

func outer_bad(xs: borrow []u8) -> []u8:
    return inner(xs)

func main() -> Int:
    return 0
`, "borrowed slice return requires '-> borrow []u8' or '.copy()'")
}

func TestBorrowedReturnBranchOriginConsistency(t *testing.T) {
	testkit.RequireCheckOK(t, `
func choose_same(flag: Bool, xs: borrow []u8) -> borrow []u8:
    if flag:
        return xs.prefix(2).borrow()
    return xs.suffix(1).borrow()

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckErrorContains(t, `
func choose(flag: Bool, a: borrow []u8, b: borrow []u8) -> borrow []u8:
    if flag:
        return a.borrow()
    return b.borrow()

func main() -> Int:
    return 0
`, "borrowed return has multiple possible owner sources ('a', 'b'); named lifetimes are not supported in v1")
}

func TestBorrowedReturnRejectsUnsafeUnknownProvenance(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func bad(xs: []u8) -> borrow []u8
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        return core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem).borrow()

func main() -> Int:
    return 0
`, "borrowed slice return requires caller-visible borrow source")
}

func TestBorrowedAggregateEscapeDiagnostics(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Box:
    bytes: []u8

func bad(xs: borrow []u8) -> Box:
    return Box(bytes: xs.window(1, 2).borrow())

func main() -> Int:
    return 0
`, "aggregate 'Box' contains borrowed slice field 'bytes' that cannot escape through owned return")

	testkit.RequireFileCheckErrorContains(t, `
struct Box:
    bytes: []u8

var saved: Box

func stash(xs: borrow []u8) -> Int:
    saved = Box(bytes: xs.window(1, 2).borrow())
    return 0
`, "aggregate 'Box' contains borrowed slice field 'bytes' that cannot be stored in global")

	testkit.RequireCheckErrorContains(t, `
struct Box:
    bytes: []u8

enum Msg:
    case boxed(Box)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(4)
    return core.send_typed(core.self(), Msg.boxed(Box(bytes: xs.window(1, 2).borrow())))
`, "aggregate 'Box' contains borrowed slice field 'bytes' that cannot cross actor boundary")

	testkit.RequireFileCheckErrorContains(t, `
struct TextBox:
    text: String

func bad_text(text: borrow String) -> TextBox:
    return TextBox(text: text.window(1, 2).borrow())

func main() -> Int:
    return 0
`, "aggregate 'TextBox' contains borrowed String field 'text' that cannot escape through owned return")

	testkit.RequireFileCheckErrorContains(t, `
func bad_optional(xs: borrow []u8) -> []u8?:
    return xs.window(1, 2).borrow()

func main() -> Int:
    return 0
`, "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot escape through owned return")

	testkit.RequireFileCheckErrorContains(t, `
enum MaybeBytes:
    case some([]u8)
    case empty

func bad_enum(xs: borrow []u8) -> MaybeBytes:
    return MaybeBytes.some(xs.window(1, 2).borrow())

func main() -> Int:
    return 0
`, "aggregate 'MaybeBytes' contains borrowed slice field 'MaybeBytes.some[1]' that cannot escape through owned return")

	testkit.RequireFileCheckErrorContains(t, `
struct Box<T>:
    value: T

func bad_generic(xs: borrow []u8) -> Box<[]u8>:
    return Box<[]u8>{value: xs.window(1, 2).borrow()}

func main() -> Int:
    return 0
`, "contains borrowed slice field 'value' that cannot escape through owned return")

	testkit.RequireCheckOK(t, `
struct Box<T>:
    value: T

enum MaybeBytes:
    case some([]u8)
    case empty

func good_optional(xs: borrow []u8) -> []u8?
uses alloc, mem:
    let owned: []u8 = xs.window(1, 2).copy()
    return owned

func good_enum(xs: borrow []u8) -> MaybeBytes
uses alloc, mem:
    return MaybeBytes.some(xs.window(1, 2).copy())

func good_generic(xs: borrow []u8) -> Box<[]u8>
uses alloc, mem:
    return Box<[]u8>{value: xs.window(1, 2).copy()}

func main() -> Int:
    return 0
	`)
}

func TestMemoryIdealV0BorrowStructOptionalLocalAndCopyEscapes(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Box:
    bytes: []u8

func local_struct(xs: borrow []u8) -> Int:
    let box: Box = Box(bytes: xs.window(1, 2).borrow())
    return box.bytes.len

func local_optional(xs: borrow []u8) -> Int:
    let maybe: []u8? = xs.window(1, 2).borrow()
    if let raw = maybe:
        return raw.len
    else:
        return 0

func return_copied_struct(xs: borrow []u8) -> Box
uses alloc, mem:
    return Box(bytes: xs.window(0, 1).copy())

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckOK(t, `
struct Box:
    bytes: []u8

var saved: []u8? = none

func stash_copied_optional(xs: borrow []u8) -> Int
uses alloc, mem:
    saved = xs.window(0, 1).copy()
    return 0

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV1BorrowEnumPayloadAndGenericWrapperLocalUse(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Box<T>:
    value: T

enum MaybeBytes:
    case some([]u8)
    case empty

func local_enum(xs: borrow []u8) -> Int:
    let maybe: MaybeBytes = MaybeBytes.some(xs.window(1, 2).borrow())
    match maybe:
        case MaybeBytes.some(raw):
            return raw.len
        case MaybeBytes.empty:
            return 0

func local_generic(xs: borrow []u8) -> Int:
    let box: Box<[]u8> = Box<[]u8>{value: xs.window(1, 2).borrow()}
    return box.value.len

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV1BorrowEnumPayloadGlobalStorageRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MaybeBytes:
    case some([]u8)
    case empty

var saved: MaybeBytes

func stash(xs: borrow []u8) -> Int:
    saved = MaybeBytes.some(xs.window(1, 2).borrow())
    return 0
`, "aggregate 'MaybeBytes' contains borrowed slice field 'MaybeBytes.some[1]' that cannot be stored in global")
}

func TestMemoryIdealV1BorrowGenericWrapperGlobalStorageRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Box<T>:
    value: T

var saved: Box<[]u8>

func stash(xs: borrow []u8) -> Int:
    saved = Box<[]u8>{value: xs.window(1, 2).borrow()}
    return 0
`, "contains borrowed slice field 'value' that cannot be stored in global")
}

func TestMemoryIdealV1BorrowEnumPayloadAndGenericWrapperCopyEscapes(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
struct Box<T>:
    value: T

enum MaybeBytes:
    case some([]u8)
    case empty

var saved_enum: MaybeBytes
var saved_box: Box<[]u8>

func stash_copied(xs: borrow []u8) -> Int
uses alloc, mem:
    saved_enum = MaybeBytes.some(xs.window(0, 1).copy())
    saved_box = Box<[]u8>{value: xs.window(1, 1).copy()}
    return 0

func return_enum(xs: borrow []u8) -> MaybeBytes
uses alloc, mem:
    return MaybeBytes.some(xs.window(0, 1).copy())

func return_generic(xs: borrow []u8) -> Box<[]u8>
uses alloc, mem:
    return Box<[]u8>{value: xs.window(0, 1).copy()}

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV2KnownCallbackAndFunctionTypedFieldBorrowUse(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Sink:
    cb: fn(borrow []u8) -> Int

func len_borrow(xs: borrow []u8) -> Int:
    return xs.len

func local_callback(xs: borrow []u8) -> Int:
    let cb: fn(borrow []u8) -> Int = len_borrow
    return cb(xs.window(0, 1).borrow())

func field_callback(xs: borrow []u8, sink: Sink) -> Int:
    return sink.cb(xs.window(0, 1).borrow())

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV2CopyBeforeCallbackEscapeAccepted(t *testing.T) {
	testkit.RequireCheckOK(t, `
func takes_owned(xs: []u8) -> Int:
    return xs.len

func call_with_copy(xs: borrow []u8) -> Int
uses alloc, mem:
    let cb: fn([]u8) -> Int = takes_owned
    let owned: []u8 = xs.window(0, 1).copy()
    return cb(owned)

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV2BorrowedCallbackNonBorrowParamRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func takes_owned(xs: []u8) -> Int:
    return xs.len

func bad(xs: borrow []u8) -> Int:
    let cb: fn([]u8) -> Int = takes_owned
    return cb(xs.window(0, 1).borrow())

func main() -> Int:
    return 0
`, "borrowed value derived from 'xs' cannot be passed to non-borrow parameter 1 of callback 'cb'")
}

func TestMemoryIdealV2BorrowedCallbackReturnAsOwnedRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func view(xs: borrow []u8) -> borrow []u8:
    return xs.window(0, 1).borrow()

func bad(xs: borrow []u8) -> []u8:
    let cb: fn(borrow []u8) -> borrow []u8 = view
    return cb(xs)

func main() -> Int:
    return 0
`, "borrowed slice return requires '-> borrow []u8' or '.copy()'")
}

func TestMemoryIdealV2BorrowedCallbackGlobalStorageRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var saved: []u8

func view(xs: borrow []u8) -> borrow []u8:
    return xs.window(0, 1).borrow()

func bad(xs: borrow []u8) -> Int:
    let cb: fn(borrow []u8) -> borrow []u8 = view
    saved = cb(xs)
    return 0
`, "borrowed local 'xs' cannot escape via global assignment to 'saved'")
}

func TestMemoryIdealV2BorrowedCallbackConsumeAndInoutRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func consume_bytes(xs: consume []u8) -> Int:
    return xs.len

func bad(xs: borrow []u8) -> Int:
    let view: []u8 = xs.window(0, 1).borrow()
    let cb: fn(consume []u8) -> Int = consume_bytes
    return cb(view)

func main() -> Int:
    return 0
`, "borrowed value derived from 'xs' cannot be consumed by callback 'cb'")

	testkit.RequireFileCheckErrorContains(t, `
func mutate(xs: inout []u8) -> Int:
    return xs.len

func bad(xs: borrow []u8) -> Int:
    var view: []u8 = xs.window(0, 1).borrow()
    let cb: fn(inout []u8) -> Int = mutate
    return cb(view)

func main() -> Int:
    return 0
`, "borrowed value derived from 'xs' cannot be passed as inout to callback 'cb'")
}

func TestMemoryIdealV2CallbackAliasesInoutArgumentRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func touch(dst: inout []u8, view: borrow []u8) -> Int:
    return dst.len + view.len

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    let cb: fn(inout []u8, borrow []u8) -> Int = touch
    return cb(xs, xs)
`, "borrowed argument 'xs' aliases inout argument in callback 'cb'")
}

func TestMemoryIdealV2UnknownCallbackTargetConservativeRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func apply(cb: fn(borrow []u8) -> Int, xs: borrow []u8) -> Int:
    return cb(xs)

func outer(cb: fn(borrow []u8) -> Int, xs: borrow []u8) -> Int
noalloc:
    return apply(cb, xs)

func main() -> Int:
    return 0
`, "callback argument for 'apply' has no known fnptr target under semantic clause 'noalloc'")
}

func TestMemoryIdealV2CapturingCallbackGlobalEscapeRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var saved: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(cb: fn(Int) -> Int) -> Int:
    saved = cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`, "function-typed parameter 'cb' cannot be stored in global function-typed value 'saved'")
}

func TestMemoryIdealV3KnownStaticProtocolTargetBorrowUse(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct BorrowView:
    value: []u8

protocol ViewLike:
    func len(self: borrow BorrowView) -> Int

extension BorrowView:
    func len(self: borrow BorrowView) -> Int:
        return self.value.len

impl BorrowView: ViewLike

func local_static_protocol(xs: borrow []u8) -> Int:
    let view: BorrowView = BorrowView{value: xs.window(0, 1).borrow()}
    return BorrowView.len(view)

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV3BorrowedInterfaceReturnAsOwnedRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct BorrowView:
    value: []u8

protocol ViewLike:
    func len(self: BorrowView) -> Int

extension BorrowView:
    func len(self: BorrowView) -> Int:
        return self.value.len

impl BorrowView: ViewLike

func bad(xs: borrow []u8) -> BorrowView:
    return BorrowView{value: xs.window(0, 1).borrow()}

func main() -> Int:
    return 0
`, "aggregate 'BorrowView' contains borrowed slice field 'value' that cannot escape through owned return")
}

func TestMemoryIdealV3BorrowedInterfaceGlobalStorageRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct BorrowView:
    value: []u8

protocol ViewLike:
    func len(self: BorrowView) -> Int

extension BorrowView:
    func len(self: BorrowView) -> Int:
        return self.value.len

impl BorrowView: ViewLike

var saved: BorrowView

func bad(xs: borrow []u8) -> Int:
    saved = BorrowView{value: xs.window(0, 1).borrow()}
    return 0
`, "contains borrowed slice field 'value' that cannot be stored in global")
}

func TestMemoryIdealV3UnknownDynamicDispatchConservativeRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Vec2:
    x: Int

protocol Drawable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Drawable

func render<T: Drawable>(value: T) -> Int:
    return Drawable.draw(value)

func main() -> Int:
    return render(Vec2(x: 1))
`, "unknown function 'Drawable.draw'")

	testkit.RequireFileCheckErrorContains(t, `
struct Vec2:
    x: Int

protocol Drawable:
    func draw(self: Vec2) -> Int

func main() -> Int:
    let value: Drawable = Vec2(x: 1)
    return 0
`, "unknown type 'Drawable'")
}

func TestBorrowedViewGlobalStorageRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var saved: []i32

func stash(xs: []i32) -> Int:
    saved = xs.window(0, 1).borrow()
    return 0
`, "borrowed local 'xs' cannot escape via global assignment to 'saved'")
}

func TestBorrowedActorSendRejectedUnlessCopied(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum Msg:
    case bytes([]u8)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(1)
    return core.send_typed(core.self(), Msg.bytes(xs.borrow()))
`, "cannot send borrowed view across actor boundary; use .copy()")

	testkit.RequireCheckOK(t, `
enum Msg:
    case bytes([]u8)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(1)
    return core.send_typed(core.self(), Msg.bytes(xs.copy()))
`)

	testkit.RequireCheckErrorContains(t, `
enum Msg:
    case text(String)

func main() -> Int
uses actors, alloc, mem:
    let text: String = "remote".copy()
    return core.send_typed(core.self(), Msg.text(text.borrow()))
`, "cannot send borrowed view across actor boundary; use .copy()")

	testkit.RequireCheckOK(t, `
enum Msg:
    case text(String)

func main() -> Int
uses actors, alloc, mem:
    let text: String = "remote".copy()
    return core.send_typed(core.self(), Msg.text(text.borrow().copy()))
`)

	testkit.RequireCheckOK(t, `
enum Msg:
    case text(String)

func main() -> Int
uses actors, alloc, mem:
    return core.send_typed(core.self(), Msg.text("remote".copy()))
`)
}

func TestBorrowedTaskBoundaryTypedErrorPayloadRejected(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum TaskErr:
    case bytes([]u8)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)
`, "typed task error payload must be sendable across task boundary")

	testkit.RequireCheckErrorContains(t, `
enum TaskErr:
    case text(String)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)
`, "typed task error payload must be sendable across task boundary")

	testkit.RequireCheckOK(t, `
enum TaskErr:
    case code(Int)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses alloc, mem, runtime:
    var xs: []u8 = make_u8(1)
    xs[0] = 42
    let copied: []u8 = xs.borrow().copy()
    if copied[0] != 42:
        return 1
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    let status: Int = catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.code(value):
        value
    return status
`)
}

func TestCopyResultMayEscapeSafely(t *testing.T) {
	testkit.RequireCheckOK(t, `
func owned_copy(xs: []i32) -> []i32
uses alloc, mem:
    return xs.window(0, 1).copy()

func main() -> Int:
    return 0
`)
}

func TestBorrowCopyBuildOnlyTargets(t *testing.T) {
	src := `func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    xs[0] = 1
    xs[1] = 2
    let copied: []u8 = xs.window(0, 2).copy()
    var dst: []u8 = make_u8(2)
    return copied.copy_into(dst)
`
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(tmp, "app-"+target+".wasm")
		if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target, compiler.BuildOptions{Jobs: 1}); err != nil {
			t.Fatalf("build %s: %v", target, err)
		}
	}
	for _, target := range []string{"linux-x86"} {
		outPath := filepath.Join(tmp, "app-"+target+".tobj")
		if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target, compiler.BuildOptions{Jobs: 1, Emit: compiler.EmitLibrary}); err != nil {
			t.Fatalf("build %s library: %v", target, err)
		}
	}
}
