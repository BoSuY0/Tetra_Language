package compiler_test

import (
	"strings"
	"testing"

	"tetra_language/compiler/tests/callables/testkit"
)

var (
	buildAndRun      = testkit.BuildAndRun
	buildAndRunFiles = testkit.BuildAndRunFiles
	buildOnly        = testkit.BuildOnly
	buildOnlyFiles   = testkit.BuildOnlyFiles
)

func TestBuildFunctionTypedCallableMVPRejectsUnsupportedForms(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "generic literal binding with capture",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return f(41)
`,
			want: ("generic closure captures are not supported by the production " +
				"fnptr ABI; use a non-generic closure or pass captured state explicitly"),
		},
		{
			name: "generic direct callback literal with capture",
			src: `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    return apply(fn<T>(x: T) -> T:
        let _: Int = base
        return x
    , 41)
`,
			want: ("callback argument 'closure literal' captures local 'base'; " +
				"generic closure captures are not supported by the production fnptr ABI; " +
				"use a non-generic closure or pass captured state explicitly"),
		},
		{
			name: "generic struct field literal with capture",
			src: `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let base: Int = 1
    let holder: Holder = Holder(cb: fn<T>(x: T) -> T:
        let _: Int = base
        return x
    )
    return holder.cb(41)
`,
			want: ("generic closure captures are not supported by the production " +
				"fnptr ABI; use a non-generic closure or pass captured state explicitly"),
		},
		{
			name: "generic nested struct initializer literal with capture",
			src: `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func main() -> Int:
    let base: Int = 1
    let box: Box = Box(holder: Holder(cb: fn<T>(x: T) -> T:
        let _: Int = base
        return x
    ))
    return box.holder.cb(41)
`,
			want: ("generic closure captures are not supported by the production " +
				"fnptr ABI; use a non-generic closure or pass captured state explicitly"),
		},
		{
			name: "generic enum payload literal with capture",
			src: `enum Choice:
    case some(fn(Int) -> Int)

func main() -> Int:
    let base: Int = 1
    let choice: Choice = Choice.some(fn<T>(x: T) -> T:
        let _: Int = base
        return x
    )
    match choice:
        case Choice.some(cb):
            return cb(41)
`,
			want: ("generic closure captures are not supported by the production " +
				"fnptr ABI; use a non-generic closure or pass captured state explicitly"),
		},
		{
			name: "generic return literal with capture",
			src: `func pick() -> fn(Int) -> Int:
    let base: Int = 1
    return fn<T>(x: T) -> T:
        let _: Int = base
        return x

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return cb(41)
`,
			want: ("generic closure captures are not supported by the production " +
				"fnptr ABI; use a non-generic closure or pass captured state explicitly"),
		},
		{
			name: "generic mutable local reassignment literal with capture",
			src: `func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 1
    var f: fn(Int) -> Int = add1
    f = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return f(41)
`,
			want: ("generic closure captures are not supported by the production " +
				"fnptr ABI; use a non-generic closure or pass captured state explicitly"),
		},
		{
			name: "generic struct field reassignment literal with capture",
			src: `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 1
    var holder: Holder = Holder(cb: add1)
    holder.cb = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return holder.cb(41)
`,
			want: ("generic closure captures are not supported by the production " +
				"fnptr ABI; use a non-generic closure or pass captured state explicitly"),
		},
		{
			name: "generic nested struct field reassignment literal with capture",
			src: `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 1
    var box: Box = Box(holder: Holder(cb: add1))
    box.holder.cb = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return box.holder.cb(41)
`,
			want: ("generic closure captures are not supported by the production " +
				"fnptr ABI; use a non-generic closure or pass captured state explicitly"),
		},
		{
			name: "generic enum payload reassignment literal with capture",
			src: `enum Choice:
    case some(fn(Int) -> Int)

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 1
    var choice: Choice = Choice.some(add1)
    choice = Choice.some(fn<T>(x: T) -> T:
        let _: Int = base
        return x
    )
    match choice:
        case Choice.some(cb):
            return cb(41)
`,
			want: ("generic closure captures are not supported by the production " +
				"fnptr ABI; use a non-generic closure or pass captured state explicitly"),
		},
		{
			name: "throwing literal binding",
			src: `enum Boom:
    case bad

func main() -> Int:
    let f: fn(Int) -> Int = fn(x: Int) -> Int throws Boom:
        throw Boom.bad
    return f(41)
	`,
			want: "function-typed local 'f' throws type mismatch: expected '', got 'Boom'",
		},
		{
			name: "function typed return closure literal ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

func pick() -> fn(Int) -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`,
			want: ("function-typed return 'closure literal' captures unsupported " +
				"local 'box' of type 'PtrBox'; only immutable local Int/Bool/String, " +
				"simple struct, enum, and optional captures without ptr/resource fields " +
				"are supported within the supported fnptr ABI"),
		},
		{
			name: "function typed return oversized ptr field capture reports heap escape diagnostic",
			src: `struct PtrBox:
    p: ptr

func pick() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let box: PtrBox = PtrBox(p: 0)
    return fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x + one + two + three + four + five + six + seven + eight

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`,
			want: ("escaped function value captures local 'box' of type 'PtrBox'; " +
				"pointer or resource captures require an explicit ownership transfer " +
				"model"),
		},
		{
			name: "function typed return oversized mutable capture reports heap escape diagnostic",
			src: `func pick() -> fn(Int) -> Int:
    var total: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + total + two + three + four + five + six + seven + eight + nine

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`,
			want: ("heap-escaped function value captures mutable local 'total'; " +
				"mutable by-reference captures require a proven lifetime and " +
				"synchronization model"),
		},
		{
			name: "function typed global oversized ptr field capture reports resource escape diagnostic",
			src: `struct PtrBox:
    p: ptr

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let box: PtrBox = PtrBox(p: 0)
    cb = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x + one + two + three + four + five + six + seven + eight
    return 0
`,
			want: ("escaped function value captures local 'box' of type 'PtrBox'; " +
				"pointer or resource captures require an explicit ownership transfer " +
				"model"),
		},
		{
			name: "function typed local closure literal rejects extra parameter",
			src: `func main() -> Int:
    let f: fn(Int) -> Int = fn(x: Int, y: Int) -> Int:
        return x + y
    return f(41)
`,
			want: "function-typed local 'f' parameter count mismatch: expected 1, got 2",
		},
		{
			name: "throwing direct closure literal callback argument",
			src: `enum Boom:
    case bad

func apply(x: Int, cb: fn(Int) -> Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(41, fn(x: Int) -> Int throws Boom:
        throw Boom.bad
    )
`,
			want: "callback argument 'closure literal' throws type mismatch: expected '', got 'Boom'",
		},
		{
			name: "direct closure literal callback argument parameter count mismatch",
			src: `func apply(x: Int, cb: fn(Int) -> Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(41, fn() -> Int:
        return 1
    )
`,
			want: "callback argument 'closure literal' parameter count mismatch: expected 1, got 0",
		},
		{
			name: "direct closure literal callback argument rejects extra parameter",
			src: `func apply(x: Int, cb: fn(Int) -> Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(41, fn(x: Int, y: Int) -> Int:
        return x + y
    )
`,
			want: "callback argument 'closure literal' parameter count mismatch: expected 1, got 2",
		},
		{
			name: "generic named symbol with uninferable type parameter",
			src: `func keep<T>(x: Int) -> Int:
    return x

func main() -> Int:
    let f: fn(Int) -> Int = keep
    return f(41)
`,
			want: "cannot infer generic argument 'T' for function-typed local 'f'",
		},
		{
			name: "throwing named symbol binding",
			src: `enum Boom:
    case bad

func risky(x: Int) -> Int throws Boom:
    throw Boom.bad

func main() -> Int:
    let f: fn(Int) -> Int = risky
    return f(41)
`,
			want: ("throwing function symbol 'risky' cannot initialize function-" +
				"typed local 'f'; local fnptr ABI requires the declared throws type to " +
				"match"),
		},
		{
			name: "throwing returned symbol without declared throws",
			src: `enum Boom:
    case bad

func risky(x: Int) -> Int throws Boom:
    throw Boom.bad

func pick() -> fn(Int) -> Int:
    return risky

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`,
			want: "returned function symbol 'risky' throws type mismatch: expected '', got 'Boom'",
		},
		{
			name: "throwing struct field call without try",
			src: `enum Boom:
    case bad

struct Holder:
    cb: fn(Int) -> Int throws Boom

func risky(x: Int) -> Int throws Boom:
    throw Boom.bad

func main() -> Int:
    let holder: Holder = Holder(cb: risky)
    return holder.cb(41)
`,
			want: "call to throwing function 'holder.cb' requires try",
		},
		{
			name: "throwing enum payload call without try",
			src: `enum Boom:
    case bad

enum Choice:
    case some(fn(Int) -> Int throws Boom)

func risky(x: Int) -> Int throws Boom:
    throw Boom.bad

func main() -> Int:
    let choice: Choice = Choice.some(risky)
    match choice:
        case Choice.some(cb):
            return cb(41)
`,
			want: "call to throwing function 'cb' requires try",
		},
		{
			name: "throwing global call without try",
			src: `enum Boom:
    case bad

val cb: fn(Int) -> Int throws Boom = risky

func risky(x: Int) -> Int throws Boom:
    throw Boom.bad

func main() -> Int:
    return cb(41)
`,
			want: "call to throwing function 'risky' requires try",
		},
		{
			name: "throwing mutable global reassignment throws mismatch",
			src: `enum Boom:
    case bad

enum Crash:
    case bad

var cb: fn(Int) -> Int throws Boom = risky

func risky(x: Int) -> Int throws Boom:
    return x + 1

func crash(x: Int) -> Int throws Crash:
    throw Crash.bad

func main() -> Int:
    cb = crash
    return 0
`,
			want: "function-typed assignment to 'cb' throws type mismatch: expected 'Boom', got 'Crash'",
		},
		{
			name: "symbol backed value escape",
			src: `func add1(x: Int) -> Int:
    return x + 1

func take_ptr(x: ptr) -> Int:
    return 0

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return take_ptr(f)
`,
			want: "function value 'f' cannot escape outside the supported fnptr ABI",
		},
		{
			name: "callback wrong argument count",
			src: `func call(cb: fn(Int) -> Int) -> Int:
    return cb()

func main() -> Int:
    return 0
`,
			want: "wrong argument count for callback 'cb'",
		},
		{
			name: "callback argument type mismatch",
			src: `func call(cb: fn(Int) -> Int) -> Int:
    return cb(true)

func main() -> Int:
    return 0
`,
			want: "type mismatch for callback 'cb' arg 1",
		},
		{
			name: "callback argument literal source rejected",
			src: `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(1, 41)
`,
			want: ("callback argument for 'apply' must be a supported fnptr source: " +
				"closure literal, function-typed local/global/struct field, direct named " +
				"function/closure symbol, or function-typed return call"),
		},
		{
			name: "callback argument local non function source rejected",
			src: `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let f: Int = 1
    return apply(f, 41)
`,
			want: ("callback argument for 'apply' must be a supported fnptr source: " +
				"closure literal, function-typed local/global/struct field, direct named " +
				"function/closure symbol, or function-typed return call"),
		},
		{
			name: "function typed parameter call without target set rejected by lowering",
			src: `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return 0
`,
			want: ("function-typed parameter 'cb' cannot be lowered as a direct " +
				"fnptr call without a known target set; pass a direct named function/" +
				"closure symbol at each call site or use supported function-typed " +
				"storage before dispatch"),
		},
		{
			name: "callback labeled argument type mismatch",
			src: `func call(cb: fn(Int) -> Int) -> Int:
    return cb(x: true)

func main() -> Int:
    return 0
`,
			want: "type mismatch for callback 'cb' arg 1",
		},
		{
			name: "callback mixed labeled and unlabeled arguments",
			src: `func call(cb: fn(Int, Int) -> Int) -> Int:
    return cb(left: 1, 2)

func main() -> Int:
    return 0
`,
			want: "cannot mix labeled and unlabeled arguments in callback 'cb'",
		},
		{
			name: "function typed struct field mixed labeled and unlabeled arguments",
			src: `struct Holder:
    cb: fn(Int, Int) -> Int

func add(x: Int, y: Int) -> Int:
    return x + y

func main() -> Int:
    let holder: Holder = Holder(cb: add)
    return holder.cb(left: 1, 2)
`,
			want: ("cannot mix labeled and unlabeled arguments in function-typed " +
				"struct field call 'holder.cb'"),
		},
		{
			name: "function typed struct field wrong argument count",
			src: `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    return holder.cb()
`,
			want: "wrong argument count for function-typed struct field call 'holder.cb'",
		},
		{
			name: "function typed struct field literal initializer source rejected",
			src: `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let holder: Holder = Holder(cb: 1)
    return 0
`,
			want: ("function-typed struct field 'holder.cb' initializer must be a " +
				"supported fnptr source: closure literal, function-typed local/global/" +
				"struct field, direct named function/closure symbol, or function-typed " +
				"return call"),
		},
		{
			name: "function typed struct field generic ptr closure initializer rejected",
			src: `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let id: ptr = fn<T>(x: T) -> T:
        return x
    let holder: Holder = Holder(cb: id)
    return 0
`,
			want: ("generic function symbol 'id' cannot initialize function-typed " +
				"struct field 'holder.cb'; struct-field fnptr ABI requires a monomorphic " +
				"target"),
		},
		{
			name: "function typed enum payload literal initializer source rejected",
			src: `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(1)
    return 0
`,
			want: ("function-typed enum payload 'MaybeCallback.some[1]' initializer " +
				"must be a supported fnptr source: closure literal, function-typed local/" +
				"global/struct field, direct named function/closure symbol, or function-" +
				"typed return call"),
		},
		{
			name: "function typed enum payload generic ptr closure initializer rejected",
			src: `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let id: ptr = fn<T>(x: T) -> T:
        return x
    let choice: MaybeCallback = MaybeCallback.some(id)
    return 0
`,
			want: ("generic function symbol 'id' cannot initialize function-typed " +
				"enum payload 'MaybeCallback.some[1]'; enum-payload fnptr ABI requires a " +
				"monomorphic target"),
		},
		{
			name: "function typed local literal initializer source rejected",
			src: `func main() -> Int:
    let f: fn(Int) -> Int = 1
    return 0
`,
			want: ("function-typed local 'f' initializer must be a symbol-backed " +
				"function value, target-set-backed function value, direct named function " +
				"symbol, or closure literal for the supported fnptr ABI"),
		},
		{
			name: "function typed local unknown return call initializer source rejected",
			src: `func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return 0
`,
			want: ("function-typed local 'f' initializer call 'pick' must resolve " +
				"to a function-typed return for the supported fnptr ABI"),
		},
		{
			name: "function typed local global non function initializer source rejected",
			src: `val value: Int = 1

func main() -> Int:
    let f: fn(Int) -> Int = value
    return 0
`,
			want: ("function-typed local 'f' initializer must be a symbol-backed " +
				"function value, target-set-backed function value, direct named function " +
				"symbol, or closure literal for the supported fnptr ABI"),
		},
		{
			name: "function typed local field non function initializer source rejected",
			src: `struct Box:
    value: Int

func main() -> Int:
    let box: Box = Box(value: 1)
    let f: fn(Int) -> Int = box.value
    return 0
`,
			want: ("function-typed local 'f' initializer must be a symbol-backed " +
				"function value, target-set-backed function value, direct named function " +
				"symbol, or closure literal for the supported fnptr ABI"),
		},
		{
			name: "function typed struct field explicit type arguments rejected",
			src: `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    return holder.cb<Int>(41)
`,
			want: ("explicit type arguments are not supported for function-typed " +
				"struct field call 'holder.cb'; function-typed dispatch uses a " +
				"monomorphic fnptr ABI, so remove explicit type arguments"),
		},
		{
			name: "function typed struct field borrow inout alias rejected",
			src: `struct Holder:
    cb: fn(borrow Int, inout Int) -> Int

func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    let holder: Holder = Holder(cb: mix)
    return holder.cb(a, a)
`,
			want: ("inout argument 'a' aliases borrowed argument in function-typed " +
				"struct field call 'holder.cb'"),
		},
		{
			name: "function typed enum payload borrow inout alias rejected",
			src: `enum MaybeCallback:
    case some(fn(borrow Int, inout Int) -> Int)
    case empty

func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(mix)
    match choice:
    case MaybeCallback.some(cb):
        return cb(a, a)
    case MaybeCallback.empty:
        return 0
`,
			want: "inout argument 'a' aliases borrowed argument in function-typed enum payload call 'cb'",
		},
		{
			name: "function typed enum payload explicit type arguments rejected",
			src: `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb<Int>(41)
    case MaybeCallback.empty:
        return 0
`,
			want: ("explicit type arguments are not supported for function-typed " +
				"enum payload call 'cb'; function-typed dispatch uses a monomorphic " +
				"fnptr ABI, so remove explicit type arguments"),
		},
		{
			name: "function typed enum payload wrong argument count",
			src: `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb()
    case MaybeCallback.empty:
        return 0
`,
			want: "wrong argument count for function-typed enum payload call 'cb'",
		},
		{
			name: "function typed global borrow inout alias rejected",
			src: `val cb: fn(borrow Int, inout Int) -> Int = mix

func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    return cb(a, a)
`,
			want: "inout argument 'a' aliases borrowed argument in function-typed global call 'cb'",
		},
		{
			name: "function typed global consume non local rejected",
			src: `val cb: fn(consume Int) -> Int = take

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    return cb(1)
`,
			want: "consume argument for function-typed global call 'cb' must be a local value",
		},
		{
			name: "function typed global wrong argument count rejected",
			src: `val cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return cb()
`,
			want: "wrong argument count for function-typed global call 'cb'",
		},
		{
			name: "function typed global type mismatch rejected",
			src: `val cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return cb(true)
`,
			want: "type mismatch for function-typed global call 'cb' arg 1",
		},
		{
			name: "function typed global explicit type arguments rejected",
			src: `val cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return cb<Int>(41)
`,
			want: ("explicit type arguments are not supported for function-typed " +
				"global call 'cb'; function-typed dispatch uses a monomorphic fnptr ABI, " +
				"so remove explicit type arguments"),
		},
		{
			name: "captured function typed local explicit type arguments rejected",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f<Int>(41)
`,
			want: ("explicit type arguments are not supported for function-typed " +
				"callback 'f'; function-typed dispatch uses a monomorphic fnptr ABI, so " +
				"remove explicit type arguments"),
		},
		{
			name: "captured function typed local wrong argument count rejected",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f()
`,
			want: "wrong argument count for function-typed callback 'f'",
		},
		{
			name: "captured function typed local type mismatch rejected",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f(true)
`,
			want: "type mismatch for function-typed callback 'f' arg 1",
		},
		{
			name: "captured function typed local mixed labeled arguments rejected",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int, Int) -> Int = fn(x: Int, y: Int) -> Int:
        return x + y + base
    return f(left: 1, 2)
`,
			want: "cannot mix labeled and unlabeled arguments in function-typed callback 'f'",
		},
		{
			name: "direct closure literal callback ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return apply(fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    , 41)
`,
			want: ("callback argument 'closure literal' captures unsupported local " +
				"'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple " +
				"struct, enum, and optional captures without ptr/resource fields are " +
				"supported within the supported fnptr ABI"),
		},
		{
			name: "function typed local initializer ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    return 0
`,
			want: ("function-typed storage 'f' captures unsupported local 'box' of " +
				"type 'PtrBox'; only immutable local Int/Bool/String, simple struct, " +
				"enum, and optional captures without ptr/resource fields are supported " +
				"within the supported fnptr ABI"),
		},
		{
			name: "function typed struct field initializer ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    )
    return 0
`,
			want: ("function-typed storage 'holder.cb' captures unsupported local " +
				"'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple " +
				"struct, enum, and optional captures without ptr/resource fields are " +
				"supported within the supported fnptr ABI"),
		},
		{
			name: "function typed enum payload initializer ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    )
    return 0
`,
			want: ("function-typed storage 'MaybeCallback.some[1]' captures " +
				"unsupported local 'box' of type 'PtrBox'; only immutable local Int/Bool/" +
				"String, simple struct, enum, and optional captures without ptr/resource " +
				"fields are supported within the supported fnptr ABI"),
		},
		{
			name: "function typed mutable local reassignment ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    var f: fn(Int) -> Int = add0
    f = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    return 0
`,
			want: ("function-typed storage 'f' captures unsupported local 'box' of " +
				"type 'PtrBox'; only immutable local Int/Bool/String, simple struct, " +
				"enum, and optional captures without ptr/resource fields are supported " +
				"within the supported fnptr ABI"),
		},
		{
			name: "function typed mutable local reassignment literal source rejected",
			src: `func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var f: fn(Int) -> Int = add0
    f = 1
    return 0
`,
			want: ("function-typed assignment to 'f' must use a supported fnptr " +
				"source: closure literal, function-typed local/global/struct field, " +
				"direct named function/closure symbol, or function-typed return call"),
		},
		{
			name: "function typed mutable local reassignment local non function source rejected",
			src: `func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let value: Int = 1
    var f: fn(Int) -> Int = add0
    f = value
    return 0
`,
			want: ("function-typed assignment to 'f' must use a supported fnptr " +
				"source: closure literal, function-typed local/global/struct field, " +
				"direct named function/closure symbol, or function-typed return call"),
		},
		{
			name: "function typed mutable local reassignment generic ptr closure alias rejected",
			src: `func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var f: fn(Int) -> Int = add0
    let id: ptr = fn<T>(x: T) -> T:
        return x
    f = id
    return 0
`,
			want: ("generic function symbol 'id' cannot be assigned to function-" +
				"typed target 'f'; assignment fnptr ABI requires a monomorphic target"),
		},
		{
			name: "function typed mutable local reassignment throwing ptr closure alias rejected",
			src: `enum Boom:
    case bad

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var f: fn(Int) -> Int = add0
    let risky: ptr = fn(x: Int) -> Int throws Boom:
        throw Boom.bad
    f = risky
    return 0
`,
			want: ("throwing function symbol 'risky' cannot be assigned to function-" +
				"typed target 'f'; assignment fnptr ABI requires the target's declared " +
				"throws type to match"),
		},
		{
			name: "function typed mutable local reassignment unknown return call source rejected",
			src: `func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var f: fn(Int) -> Int = add0
    f = pick()
    return 0
`,
			want: ("function-typed assignment to 'f' initializer call 'pick' must " +
				"resolve to a function-typed return for the supported fnptr ABI"),
		},
		{
			name: "function typed struct field reassignment ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

struct Holder:
    cb: fn(Int) -> Int

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    var holder: Holder = Holder(cb: add0)
    holder.cb = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    return 0
`,
			want: ("function-typed storage 'holder.cb' captures unsupported local " +
				"'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple " +
				"struct, enum, and optional captures without ptr/resource fields are " +
				"supported within the supported fnptr ABI"),
		},
		{
			name: "function typed enum payload reassignment ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    )
    return 0
`,
			want: ("function-typed storage 'MaybeCallback.some[1]' captures " +
				"unsupported local 'box' of type 'PtrBox'; only immutable local Int/Bool/" +
				"String, simple struct, enum, and optional captures without ptr/resource " +
				"fields are supported within the supported fnptr ABI"),
		},
		{
			name: "direct callback signature parameter mismatch",
			src: `func as_bool(flag: Bool) -> Int:
    if flag:
        return 1
    return 0

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(as_bool, 1)
`,
			want: "callback function symbol 'as_bool' parameter 1 type mismatch: expected 'i32', got 'bool'",
		},
		{
			name: "direct callback signature return mismatch",
			src: `func truthy(x: Int) -> Bool:
    return true

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(truthy, 1)
`,
			want: "callback function symbol 'truthy' return type mismatch: expected 'i32', got 'bool'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := buildOnly(t, tc.src)
			if err == nil {
				t.Fatalf("expected diagnostic containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestBuildFunctionTypedCrossModuleUnsupportedCaptureDiagnostics(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name: "imported direct closure literal callback ptr field capture rejected",
			files: map[string]string{
				"lib/callbacks.t4": `module lib.callbacks

pub func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
				"app/main.t4": `module app.main
import lib.callbacks as callbacks

struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return callbacks.apply(fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    , 41)
`,
			},
			want: ("callback argument 'closure literal' captures unsupported local " +
				"'box' of type 'app.main.PtrBox'; only immutable local Int/Bool/String, " +
				"simple struct, enum, and optional captures without ptr/resource fields " +
				"are supported within the supported fnptr ABI"),
		},
		{
			name: "imported enum payload constructor ptr field capture rejected",
			files: map[string]string{
				"lib/types.t4": `module lib.types

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty
`,
				"app/main.t4": `module app.main
import lib.types as types

struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let choice: types.MaybeCallback = types.MaybeCallback.some(fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    )
    return 0
`,
			},
			want: ("function-typed storage 'lib.types.MaybeCallback.some[1]' " +
				"captures unsupported local 'box' of type 'app.main.PtrBox'; only " +
				"immutable local Int/Bool/String, simple struct, enum, and optional " +
				"captures without ptr/resource fields are supported within the supported " +
				"fnptr ABI"),
		},
		{
			name: "imported struct field constructor ptr field capture rejected",
			files: map[string]string{
				"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int
`,
				"app/main.t4": `module app.main
import lib.types as types

struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let holder: types.Holder = types.Holder(cb: fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    )
    return holder.cb(41)
`,
			},
			want: ("function-typed storage 'holder.cb' captures unsupported local " +
				"'box' of type 'app.main.PtrBox'; only immutable local Int/Bool/String, " +
				"simple struct, enum, and optional captures without ptr/resource fields " +
				"are supported within the supported fnptr ABI"),
		},
		{
			name: "imported struct field constructor argument ptr field capture rejected",
			files: map[string]string{
				"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int

pub func call(holder: Holder, x: Int) -> Int:
    return holder.cb(x)
`,
				"app/main.t4": `module app.main
import lib.types as types

struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return types.call(types.Holder(cb: fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    ), 41)
`,
			},
			want: ("function-typed storage 'lib.types.Holder.cb' captures " +
				"unsupported local 'box' of type 'app.main.PtrBox'; only immutable local " +
				"Int/Bool/String, simple struct, enum, and optional captures without ptr/" +
				"resource fields are supported within the supported fnptr ABI"),
		},
		{
			name: "imported function typed return ptr field capture rejected",
			files: map[string]string{
				"lib/maker.t4": `module lib.maker

struct PtrBox:
    p: ptr

pub func make() -> fn(Int) -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
`,
				"app/main.t4": `module app.main
import lib.maker as maker

func main() -> Int:
    let cb: fn(Int) -> Int = maker.make()
    return cb(41)
`,
			},
			want: ("function-typed return 'closure literal' captures unsupported " +
				"local 'box' of type 'lib.maker.PtrBox'; only immutable local Int/Bool/" +
				"String, simple struct, enum, and optional captures without ptr/resource " +
				"fields are supported within the supported fnptr ABI"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := buildOnlyFiles(t, tc.files, "app/main.t4")
			if err == nil {
				t.Fatalf("expected diagnostic containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}
