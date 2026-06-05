package compiler_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/testkit"
)

func TestSafetyDiagnosticCodesForKeyFamilies(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantCode string
		wantText string
		parseErr bool
	}{
		{
			name: "ownership consumed value reuse",
			src: `
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let a: Int = 1
    let b: Int = take(a)
    return a + b
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'a'",
		},
		{
			name: "ownership partial struct field consume whole value",
			src: `
struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func use(pair: Pair) -> Int:
    return pair.left + pair.right

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return use(pair) + moved
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'pair.left'",
		},
		{
			name: "ownership partial struct field consume whole copy",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'pair.left'",
		},
		{
			name: "ownership partial struct field consume enum constructor",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'pair.left'",
		},
		{
			name: "ownership partial enum payload consume whole value",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'msg.$case0.payload0'",
		},
		{
			name: "ownership partial enum payload consume whole copy",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'msg.$case0.payload0'",
		},
		{
			name: "ownership partial enum payload consume enum constructor",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'msg.$case0.payload0'",
		},
		{
			name: "ownership optional payload consume whole value",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'maybe.$elem'",
		},
		{
			name: "lifetime borrowed return escape",
			src: `
func leak(x: borrow []u8) -> []u8:
    return x

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed slice return requires '-> borrow []u8' or '.copy()'",
		},
		{
			name: "lifetime borrowed slice struct literal return escape",
			src: `
struct BufBox:
    buf: []u8

func leak(x: borrow []u8) -> BufBox:
    return BufBox(buf: x)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "lifetime borrowed slice struct alias return escape",
			src: `
struct BufBox:
    buf: []u8

func leak(x: borrow []u8) -> BufBox:
    let box: BufBox = BufBox(buf: x)
    return box

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "lifetime borrowed slice struct inout escape",
			src: `
struct BufBox:
    buf: []u8

func leak(read: borrow []u8, out: inout BufBox) -> Int:
    out = BufBox(buf: read)
    return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
		{
			name: "lifetime borrowed slice enum direct return escape",
			src: `
enum BufMsg:
    case send([]u8)

func leak(x: borrow []u8) -> BufMsg:
    return BufMsg.send(x)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "aggregate 'BufMsg' contains borrowed slice field 'BufMsg.send[1]' that cannot escape through owned return",
		},
		{
			name: "lifetime borrowed slice enum alias return escape",
			src: `
enum BufMsg:
    case send([]u8)

func leak(x: borrow []u8) -> BufMsg:
    let msg: BufMsg = BufMsg.send(x)
    return msg

func main() -> Int:
    return 0
	`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "ownership borrowed slice struct owned call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "ownership borrowed slice struct consume call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "ownership borrowed slice struct inout call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'mutate'",
		},
		{
			name: "ownership borrowed slice enum owned call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "ownership borrowed slice enum consume call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "ownership borrowed slice enum inout call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'mutate'",
		},
		{
			name: "ownership borrowed slice struct function-typed owned call escape",
			src: `
struct BufBox:
    buf: []u8

func caller(cb: fn(BufBox) -> Int, x: borrow []u8) -> Int:
    return cb(BufBox(buf: x))

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of callback 'cb'",
		},
		{
			name: "ownership borrowed slice struct function-typed consume call escape",
			src: `
struct BufBox:
    buf: []u8

func caller(cb: fn(consume BufBox) -> Int, x: borrow []u8) -> Int:
    let box: BufBox = BufBox(buf: x)
    return cb(box)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by callback 'cb'",
		},
		{
			name: "ownership borrowed slice struct function-typed inout call escape",
			src: `
struct BufBox:
    buf: []u8

func caller(cb: fn(inout BufBox) -> Int, x: borrow []u8) -> Int:
    var box: BufBox = BufBox(buf: x)
    return cb(box)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to callback 'cb'",
		},
		{
			name: "ownership borrowed slice enum function-typed owned call escape",
			src: `
enum BufMsg:
    case send([]u8)

func caller(cb: fn(BufMsg) -> Int, x: borrow []u8) -> Int:
    return cb(BufMsg.send(x))

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of callback 'cb'",
		},
		{
			name: "ownership borrowed slice enum function-typed consume call escape",
			src: `
enum BufMsg:
    case send([]u8)

func caller(cb: fn(consume BufMsg) -> Int, x: borrow []u8) -> Int:
    let msg: BufMsg = BufMsg.send(x)
    return cb(msg)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by callback 'cb'",
		},
		{
			name: "ownership borrowed slice enum function-typed inout call escape",
			src: `
enum BufMsg:
    case send([]u8)

func caller(cb: fn(inout BufMsg) -> Int, x: borrow []u8) -> Int:
    var msg: BufMsg = BufMsg.send(x)
    return cb(msg)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to callback 'cb'",
		},
		{
			name: "ownership borrowed slice struct function-typed field owned call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'",
		},
		{
			name: "ownership borrowed slice struct function-typed field consume call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed struct field call 'h.cb'",
		},
		{
			name: "ownership borrowed slice struct function-typed field inout call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed struct field call 'h.cb'",
		},
		{
			name: "ownership borrowed slice enum function-typed field owned call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'",
		},
		{
			name: "ownership borrowed slice enum function-typed field consume call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed struct field call 'h.cb'",
		},
		{
			name: "ownership borrowed slice enum function-typed field inout call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed struct field call 'h.cb'",
		},
		{
			name: "ownership borrowed slice struct function-typed enum-payload owned call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'",
		},
		{
			name: "ownership borrowed slice struct function-typed enum-payload consume call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed enum payload call 'cb'",
		},
		{
			name: "ownership borrowed slice struct function-typed enum-payload inout call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed enum payload call 'cb'",
		},
		{
			name: "ownership borrowed slice enum function-typed enum-payload owned call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'",
		},
		{
			name: "ownership borrowed slice enum function-typed enum-payload consume call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed enum payload call 'cb'",
		},
		{
			name: "ownership borrowed slice enum function-typed enum-payload inout call escape",
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed enum payload call 'cb'",
		},
		{
			name: "lifetime borrowed optional assignment escape",
			src: `
func leak(x: borrow ptr) -> ptr?:
    var maybe: ptr? = none
    maybe = x
    return maybe

func main() -> Int:
    return 0
	`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "lifetime borrowed ptr optional payload return escape",
			src: `
func leak(maybe: borrow ptr?) -> ptr:
    if let raw = maybe:
        return raw
    else:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'maybe' cannot escape via return",
		},
		{
			name: "ownership borrowed ptr optional payload owned call escape",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "ownership borrowed ptr optional payload consume call escape",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'sink'",
		},
		{
			name: "ownership borrowed ptr optional payload inout call escape",
			src: `
func sink(raw: inout ptr) -> Int:
    raw = 0
    return 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'sink'",
		},
		{
			name: "lifetime borrowed ptr enum payload inout escape",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'msg' cannot escape via inout assignment to 'out'",
		},
		{
			name: "lifetime borrowed ptr optional payload inout escape",
			src: `
func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
		},
		{
			name: "lifetime borrowed slice optional assignment escape",
			src: `
func leak(x: borrow []u8) -> []u8?:
    var maybe: []u8? = none
    maybe = x
    return maybe

func main() -> Int:
    return 0
	`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot escape through owned return",
		},
		{
			name: "ownership borrowed slice optional assignment owned escape",
			src: `
func sink(value: []u8?) -> Int:
    return 0

func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "ownership borrowed slice optional assignment consume escape",
			src: `
func sink(value: consume []u8?) -> Int:
    return 0

func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "lifetime borrowed slice optional assignment inout escape",
			src: `
func leak(x: borrow []u8, out: inout []u8?) -> Int:
    var maybe: []u8? = none
    maybe = x
    out = maybe
    return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'x' cannot escape via inout assignment to 'out'",
		},
		{
			name: "lifetime borrowed ptr aggregate alias return escape",
			src: `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    let alias: PtrBox = box
    return alias

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "ownership borrowed ptr aggregate owned call escape",
			src: `
struct PtrBox:
    raw: ptr

func sink(value: PtrBox) -> Int:
    return 0

func leak(box: borrow PtrBox) -> Int:
    return sink(box)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'box' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "ownership borrowed ptr aggregate consume call escape",
			src: `
struct PtrBox:
    raw: ptr

func sink(value: consume PtrBox) -> Int:
    return 0

func leak(box: borrow PtrBox) -> Int:
    return sink(box)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'box' cannot be consumed by 'sink'",
		},
		{
			name: "ownership borrowed ptr index aggregate consume call escape",
			src: `
struct PtrBox:
    raw: ptr

func sink(value: consume PtrBox) -> Int:
    return 0

func leak(boxes: borrow []PtrBox) -> Int:
    return sink(boxes[0])

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from",
		},
		{
			name: "ownership borrowed ptr nested field index aggregate consume call escape",
			src: `
struct PtrBox:
    raw: ptr

struct Container:
    boxes: []PtrBox

func sink(value: consume PtrBox) -> Int:
    return 0

func leak(container: borrow Container) -> Int:
    return sink(container.boxes[0])

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from",
		},
		{
			name: "ownership borrowed ptr index assignment consume call escape",
			src: `
struct PtrBox:
    raw: ptr

func sink(value: consume PtrBox) -> Int:
    return 0

func leak(boxes: borrow []PtrBox) -> Int:
    let first: PtrBox = boxes[0]
    return sink(first)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from",
		},
		{
			name: "ownership borrowed ptr aggregate inout call escape",
			src: `
struct PtrBox:
    raw: ptr

func sink(value: inout PtrBox) -> Int:
    value = PtrBox(raw: 0)
    return 0

func leak(box: borrow PtrBox) -> Int:
    return sink(box)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'box' cannot be passed as inout to 'sink'",
		},
		{
			name: "ownership borrowed ptr nested aggregate owned call escape",
			src: `
struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

func sink(value: OuterBox) -> Int:
    return 0

func leak(outer: borrow OuterBox) -> Int:
    return sink(outer)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'outer' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "ownership borrowed ptr nested aggregate consume call escape",
			src: `
struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

func sink(value: consume OuterBox) -> Int:
    return 0

func leak(outer: borrow OuterBox) -> Int:
    return sink(outer)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'outer' cannot be consumed by 'sink'",
		},
		{
			name: "ownership borrowed ptr nested aggregate inout call escape",
			src: `
struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

func sink(value: inout OuterBox) -> Int:
    value = OuterBox(box: PtrBox(raw: 0))
    return 0

func leak(outer: borrow OuterBox) -> Int:
    return sink(outer)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'outer' cannot be passed as inout to 'sink'",
		},
		{
			name: "lifetime borrowed ptr enum alias return escape",
			src: `
enum PtrMsg:
    case raw(ptr)

func leak(x: borrow ptr) -> PtrMsg:
    let msg: PtrMsg = PtrMsg.raw(x)
    return msg

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "lifetime borrowed ptr enum payload return escape",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'msg' cannot escape via return",
		},
		{
			name: "ownership borrowed ptr enum payload owned call escape",
			src: `
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
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'msg' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "ownership borrowed ptr enum payload consume call escape",
			src: `
enum PtrMsg:
    case raw(ptr)
    case empty

func sink(raw: consume ptr) -> Int:
    return 0

func leak(msg: borrow PtrMsg) -> Int:
    match msg:
    case PtrMsg.raw(raw):
        return sink(raw)
    case PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'msg' cannot be consumed by 'sink'",
		},
		{
			name: "ownership borrowed ptr enum payload inout call escape",
			src: `
enum PtrMsg:
    case raw(ptr)
    case empty

func sink(raw: inout ptr) -> Int:
    raw = 0
    return 0

func leak(msg: borrow PtrMsg) -> Int:
    match msg:
    case PtrMsg.raw(raw):
        return sink(raw)
    case PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'msg' cannot be passed as inout to 'sink'",
		},
		{
			name: "ownership borrowed ptr merge branch alias consume call escape",
			src: `
func sink(raw: consume ptr) -> Int:
    return 0

func leak(left: borrow ptr, right: borrow ptr, n: Int) -> Int:
    var value: ptr = left
    if n > 0:
        value = right
    return sink(value)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'left' cannot be consumed by 'sink'",
		},
		{
			name: "ownership borrowed ptr optional assignment consume escape",
			src: `
func sink(value: consume ptr?) -> Int:
    return 0

func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "lifetime scoped island optional region escape",
			src: `
func make() -> []u8?
uses alloc, islands, mem:
    island(16) as isl:
        var xs: []u8 = core.island_make_u8(isl, 4)
        var maybe: []u8? = none
        maybe = xs
        return maybe
    return none
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "slice from scoped island cannot escape to outer scope",
		},
		{
			name: "function value unsupported escape",
			src: `
func add1(x: Int) -> Int:
    return x + 1

func take_ptr(x: ptr) -> Int:
    return 0

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return take_ptr(f)
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "function value 'f' cannot escape outside the supported fnptr ABI",
		},
		{
			name: "capturing closure raw pointer escape",
			src: `
func choose(p: ptr) -> Int:
    return 0

func main() -> Int:
    let y: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + y
    return choose(f)
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "capturing closure 'f' cannot escape as raw ptr",
		},
		{
			name: "callable resource capture escape",
			src: `
struct PtrBox:
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
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "escaped function value captures local 'box' of type 'PtrBox'",
		},
		{
			name: "callable mutable capture heap escape",
			src: `
func pick() -> fn(Int) -> Int:
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
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "heap-escaped function value captures mutable local 'total'",
		},
		{
			name: "generic closure capture",
			src: `
func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return f(41)
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "generic closure literal captures local 'base'",
		},
		{
			name: "generic callback closure capture",
			src: `
func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    return apply(fn<T>(x: T) -> T:
        let _: Int = base
        return x
    , 41)
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "callback argument 'closure literal' captures local 'base'",
		},
		{
			name: "function typed storage unsupported capture",
			src: `
struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "function-typed storage 'f' captures unsupported local 'box'",
		},
		{
			name: "function typed return unsupported capture",
			src: `
struct PtrBox:
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
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "function-typed return 'closure literal' captures unsupported local 'box'",
		},
		{
			name: "captured closure explicit type args",
			src: `
func main() -> Int:
    let y: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + y
    return f<Int>(41)
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "explicit type arguments are not supported for captured closure 'f'",
		},
		{
			name: "function typed explicit type args",
			src: `
func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f<Int>(41)
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "explicit type arguments are not supported for function-typed callback 'f'",
		},
		{
			name: "unsupported function value call",
			src: `
func main() -> Int:
    let p: ptr = 0
    return p(41)
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "function value 'p' cannot be called through the supported fnptr ABI",
		},
		{
			name: "generic closure pointer escape",
			src: `
func use(p: ptr) -> Int:
    return 0

func main() -> Int:
    let id: ptr = fn<T>(x: T) -> T:
        return x
    return use(id)
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "generic closure 'id' cannot be used as a pointer value",
		},
		{
			name: "generic closure direct call requirement",
			src: `
func main() -> Int:
    var id: ptr = fn<T>(x: T) -> T:
        return x
    return id(1)
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "generic closure 'id' requires the generic direct-call closure ABI",
		},
		{
			name: "resource use after free",
			src: `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        free(isl)
        free(isl)
    }
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use freed resource 'isl'",
		},
		{
			name: "resource struct-field alias use after free",
			src: `
struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let alias: IslandBox = box
        free(box.handle)
        free(alias.handle)
    }
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use freed resource 'alias.handle'",
		},
		{
			name: "resource enum-payload alias use after free",
			src: `
enum MoveMsg:
    case take(island)

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: MoveMsg = MoveMsg.take(core.island_new(16))
        match msg:
        case MoveMsg.take(other):
            let alias: island = other
            free(other)
            free(alias)
    }
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use freed resource 'alias'",
		},
		{
			name: "resource optional payload free whole value",
			src: `
func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        match maybe:
        case some(other):
            free(other)
            return use(maybe)
        case none:
            return 0
    }
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use freed resource 'maybe.$elem'",
		},
		{
			name: "resource double join",
			src: `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(task)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use joined resource 'task'",
		},
		{
			name: "task group use after close",
			src: `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(group)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use closed resource 'group'",
		},
		{
			name: "resource ambiguous provenance",
			src: `
struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let left: island = core.island_new(16)
        let right: island = core.island_new(16)
        var box: IslandBox = IslandBox(handle: left)
        if 1:
            box = IslandBox(handle: right)
        free(box.handle)
    }
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "ambiguous resource provenance for 'box.handle'",
		},
		{
			name: "island transfer non-local payload",
			src: `
enum MoveMsg:
    case take(island)

func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        return core.send_typed(peer, MoveMsg.take(core.island_new(16)))
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "island transfer payload must be a local value",
		},
		{
			name: "actor use after transfer",
			src: `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _: Int = take_actor(peer)
    return core.send(peer, 1)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'peer'",
		},
		{
			name: "actor branch consume reuse",
			src: `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func use(flag: Int) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    if flag:
        let _: Int = take_actor(peer)
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return use(1)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'peer'",
		},
		{
			name: "actor match consume reuse",
			src: `
enum Choice:
    case take
    case keep

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func use(choice: Choice) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    match choice:
    case Choice.take:
        let taken: Int = take_actor(peer)
    case Choice.keep:
        let kept: Int = 0
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return use(Choice.take)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'peer'",
		},
		{
			name: "actor loop consume reuse",
			src: `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func use(limit: Int) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    var i: Int = 0
    while i < limit:
        let _: Int = take_actor(peer)
        i = i + 1
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return use(1)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'peer'",
		},
		{
			name: "task use after transfer",
			src: `
func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = take_task(task)
    return value + core.task_join_i32(task)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "cannot use consumed value 'task'",
		},
		{
			name: "effect missing declaration",
			src: `
func main() -> Int:
    print("missing uses\n")
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyEffect,
			wantText: "uses effect 'io'",
		},
		{
			name: "privacy missing clause",
			src: `
func main() -> Int
uses privacy:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyPrivacy,
			wantText: "uses effect 'privacy' requires semantic clause 'privacy'",
		},
		{
			name: "budget missing clause",
			src: `
func audit() -> Int
uses budget:
    return 1

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyBudget,
			wantText: "uses effect 'budget' requires semantic clause 'budget'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != tt.wantCode || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, tt.wantCode)
			}
			if !tt.parseErr && (diag.Line == 0 || diag.Column == 0) {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
			raw, err := json.Marshal(diag)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(raw), `"code":"`+tt.wantCode+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, tt.wantCode)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForCrossModuleResourceAliasFinalization(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		wantText string
	}{
		{
			name: "struct-field alias use after free",
			files: map[string]string{
				"lib/resources.t4": `module lib.resources

pub struct IslandBox:
    handle: island
`,
				"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: resources.IslandBox = resources.IslandBox(handle: core.island_new(16))
        let alias: resources.IslandBox = box
        free(box.handle)
        free(alias.handle)
    }
    return 0
`,
			},
			wantText: "cannot use freed resource 'alias.handle'",
		},
		{
			name: "enum-payload alias use after free",
			files: map[string]string{
				"lib/resources.t4": `module lib.resources

pub enum MoveMsg:
    case take(island)

pub func unwrap(msg: MoveMsg) -> island:
    match msg:
    case MoveMsg.take(handle):
        return handle
`,
				"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: resources.MoveMsg = resources.MoveMsg.take(core.island_new(16))
        let other: island = resources.unwrap(msg)
        match msg:
        case resources.MoveMsg.take(handle):
            free(handle)
            free(other)
    }
    return 0
`,
			},
			wantText: "cannot use freed resource 'other'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, tt.files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
			raw, err := json.Marshal(diag)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForGenericBorrowReturns(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "same module aggregate",
			src: `
struct PtrBox:
    raw: ptr

func leak<T>(value: borrow T) -> T:
    return value

func caller(x: borrow ptr) -> PtrBox:
    return leak(PtrBox(raw: x))

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
		{
			name: "same module optional ptr",
			src: `
func leak<T>(value: borrow T) -> T:
    return value

func caller(maybe: borrow ptr?) -> ptr?:
    return leak(maybe)

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
			raw, err := json.Marshal(diag)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForCrossModuleGenericBorrowReturns(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		wantText string
	}{
		{
			name: "aggregate",
			files: map[string]string{
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
			},
			wantText: "borrowed local 'value' cannot escape via return",
		},
		{
			name: "optional ptr",
			files: map[string]string{
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
			},
			wantText: "borrowed local 'value' cannot escape via return",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, tt.files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
			raw, err := json.Marshal(diag)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForFunctionTypedOptionalPtrCallbacks(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "value-owned",
			src: `
func caller(cb: fn(ptr?) -> Int, maybe: borrow ptr?) -> Int:
    return cb(maybe)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of callback 'cb'",
		},
		{
			name: "value-consume",
			src: `
func caller(cb: fn(consume ptr?) -> Int, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return cb(alias)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by callback 'cb'",
		},
		{
			name: "value-inout",
			src: `
func caller(cb: fn(inout ptr?) -> Int, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return cb(alias)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to callback 'cb'",
		},
		{
			name: "struct-field-owned",
			src: `
struct Handler:
    cb: fn(ptr?) -> Int

func caller(h: Handler, maybe: borrow ptr?) -> Int:
    return h.cb(maybe)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'",
		},
		{
			name: "struct-field-consume",
			src: `
struct Handler:
    cb: fn(consume ptr?) -> Int

func caller(h: Handler, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return h.cb(alias)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by function-typed struct field call 'h.cb'",
		},
		{
			name: "struct-field-inout",
			src: `
struct Handler:
    cb: fn(inout ptr?) -> Int

func caller(h: Handler, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return h.cb(alias)

func main() -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-payload-owned",
			src: `
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
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'",
		},
		{
			name: "enum-payload-consume",
			src: `
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
`,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by function-typed enum payload call 'cb'",
		},
		{
			name: "enum-payload-inout",
			src: `
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
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to function-typed enum payload call 'cb'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
			raw, err := json.Marshal(diag)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForGenericResourceAliasFinalization(t *testing.T) {
	t.Run("same module task-handle generic struct alias join", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct Box<T>:
    value: T

func worker() -> Int:
    return 7

func pass_task(box: Box<task.i32>) -> Box<task.i32>:
    return box

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: Box<task.i32> = Box<task.i32>{value: task}
    let returned: Box<task.i32> = pass_task(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.value)
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'returned.value'")
	})

	t.Run("cross module task-handle generic struct alias join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_task(box: Box<task.i32>) -> Box<task.i32>:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.Box<task.i32> = resources.Box<task.i32>{value: task}
    let returned: resources.Box<task.i32> = resources.pass_task(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.value)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'returned.value'")
	})

	t.Run("same module task-group generic struct alias close", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct Box<T>:
    value: T

func pass_group(box: Box<task.group>) -> Box<task.group>:
    return box

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: Box<task.group> = Box<task.group>{value: group}
    let returned: Box<task.group> = pass_group(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.value)
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'returned.value'")
	})

	t.Run("cross module task-group generic struct alias close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_group(box: Box<task.group>) -> Box<task.group>:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: resources.Box<task.group> = resources.Box<task.group>{value: group}
    let returned: resources.Box<task.group> = resources.pass_group(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.value)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'returned.value'")
	})

	t.Run("same module island generic struct alias free", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct Box<T>:
    value: T

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: Box<island> = Box<island>{value: core.island_new(16)}
        let alias: Box<island> = box
        free(box.value)
        free(alias.value)
    }
    return 0
`)
		assertResourceAliasFinalizationDiagnostic(t, err, "cannot use freed resource 'alias.value'")
	})

	t.Run("cross module island generic struct alias free", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct Box<T>:
    value: T
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: resources.Box<island> = resources.Box<island>{value: core.island_new(16)}
        let alias: resources.Box<island> = box
        free(box.value)
        free(alias.value)
    }
    return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertResourceAliasFinalizationDiagnostic(t, err, "cannot use freed resource 'alias.value'")
	})
}

func assertResourceAliasFinalizationDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected resource alias finalization diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want resource alias finalization diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForTransitiveResourceAliasFinalization(t *testing.T) {
	t.Run("same module task-handle transitive alias join", func(t *testing.T) {
		err := testkit.CheckProgram(`
func worker() -> Int:
    return 7

func alias_one(task: task.i32) -> task.i32:
    return task

func alias_two(task: task.i32) -> task.i32:
    return alias_one(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = alias_two(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle transitive alias join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func alias_one(task: task.i32) -> task.i32:
    return task

pub func alias_two(task: task.i32) -> task.i32:
    return alias_one(task)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = resources.alias_two(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group transitive alias close", func(t *testing.T) {
		err := testkit.CheckProgram(`
func alias_one(group: task.group) -> task.group:
    return group

func alias_two(group: task.group) -> task.group:
    return alias_one(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let other: task.group = alias_two(group)
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(other)
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group transitive alias close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func alias_one(group: task.group) -> task.group:
    return group

pub func alias_two(group: task.group) -> task.group:
    return alias_one(group)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let other: task.group = resources.alias_two(group)
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(other)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

	t.Run("same module island transitive alias free", func(t *testing.T) {
		err := testkit.CheckProgram(`
func alias_one(isl: island) -> island:
    return isl

func alias_two(isl: island) -> island:
    return alias_one(isl)

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = alias_two(isl)
        free(isl)
        free(other)
    }
    return 0
`)
		assertResourceAliasFinalizationDiagnostic(t, err, "cannot use freed resource 'other'")
	})

	t.Run("cross module island transitive alias free", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func alias_one(isl: island) -> island:
    return isl

pub func alias_two(isl: island) -> island:
    return alias_one(isl)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = resources.alias_two(isl)
        free(isl)
        free(other)
    }
    return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertResourceAliasFinalizationDiagnostic(t, err, "cannot use freed resource 'other'")
	})
}

func TestSafetyDiagnosticCodesForEnumConstructorReturnResourceAliases(t *testing.T) {
	t.Run("same module task-handle enum constructor return alias join", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum TaskMsg:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: TaskMsg = wrap(task)
    match returned:
    case TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle enum constructor return alias join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskMsg = resources.wrap(task)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group enum constructor return alias close", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum GroupMsg:
    case wrap(task.group)

func wrap(group: task.group) -> GroupMsg:
    return GroupMsg.wrap(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: GroupMsg = wrap(group)
    match returned:
    case GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group enum constructor return alias close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum GroupMsg:
    case wrap(task.group)

pub func wrap(group: task.group) -> GroupMsg:
    return GroupMsg.wrap(group)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: resources.GroupMsg = resources.wrap(group)
    match returned:
    case resources.GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

}

func TestSafetyDiagnosticCodesForActorStructFieldEnumPayloadAliasTransfer(t *testing.T) {
	t.Run("same module transitive interprocedural actor alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
func worker() -> Int:
    return 0

func alias_one(peer: actor) -> actor:
    return peer

func alias_two(peer: actor) -> actor:
    return alias_one(peer)

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let other: actor = alias_two(peer)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("cross module transitive interprocedural actor alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func alias_one(peer: actor) -> actor:
    return peer

pub func alias_two(peer: actor) -> actor:
    return alias_one(peer)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let other: actor = resources.alias_two(peer)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("same module struct-field alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct ActorBox:
    handle: actor

func worker() -> Int:
    return 0

func pass(box: ActorBox) -> ActorBox:
    return box

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: ActorBox = ActorBox(handle: peer)
    let returned: ActorBox = pass(box)
    let _: Int = take_actor(peer)
    return core.send(returned.handle, 1)
`)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.handle'")
	})

	t.Run("cross module struct-field alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct ActorBox:
    handle: actor

pub func pass(box: ActorBox) -> ActorBox:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: resources.ActorBox = resources.ActorBox(handle: peer)
    let returned: resources.ActorBox = resources.pass(box)
    let _: Int = take_actor(peer)
    return core.send(returned.handle, 1)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.handle'")
	})

	t.Run("same module generic struct-field alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct Box<T>:
    value: T

func worker() -> Int:
    return 0

func pass_actor(box: Box<actor>) -> Box<actor>:
    return box

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: Box<actor> = Box<actor>{value: peer}
    let returned: Box<actor> = pass_actor(box)
    let _: Int = take_actor(peer)
    return core.send(returned.value, 1)
`)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.value'")
	})

	t.Run("cross module generic struct-field alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_actor(box: Box<actor>) -> Box<actor>:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: resources.Box<actor> = resources.Box<actor>{value: peer}
    let returned: resources.Box<actor> = resources.pass_actor(box)
    let _: Int = take_actor(peer)
    return core.send(returned.value, 1)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.value'")
	})

	t.Run("same module enum-payload alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum ActorMsg:
    case wrap(actor)

func worker() -> Int:
    return 0

func pass(msg: ActorMsg) -> ActorMsg:
    return msg

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let msg: ActorMsg = ActorMsg.wrap(peer)
    let returned: ActorMsg = pass(msg)
    match returned:
    case ActorMsg.wrap(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("cross module enum-payload alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum ActorMsg:
    case wrap(actor)

pub func pass(msg: ActorMsg) -> ActorMsg:
    return msg
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let msg: resources.ActorMsg = resources.ActorMsg.wrap(peer)
    let returned: resources.ActorMsg = resources.pass(msg)
    match returned:
    case resources.ActorMsg.wrap(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})
}

func assertActorAliasTransferDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected actor alias transfer diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want actor alias transfer diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForActorTaskOptionalPayloadAliasTransfer(t *testing.T) {
	t.Run("same module actor if-let optional-payload alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
func worker() -> Int:
    return 0

func pass(maybe: actor?) -> actor?:
    return maybe

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let maybe: actor? = peer
    let returned: actor? = pass(maybe)
    if let other = returned:
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`)
		assertActorTaskOptionalPayloadAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("cross module actor match optional-payload alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func pass(maybe: actor?) -> actor?:
    return maybe
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let maybe: actor? = peer
    let returned: actor? = resources.pass(maybe)
    match returned:
    case some(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    case none:
        return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertActorTaskOptionalPayloadAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("same module task if-let optional-payload alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
func worker() -> Int:
    return 7

func pass(maybe: task.i32?) -> task.i32?:
    return maybe

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    let returned: task.i32? = pass(maybe)
    if let other = returned:
        let first: Int = take_task(task)
        return first + core.task_join_i32(other)
    return 0
`)
		assertActorTaskOptionalPayloadAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("cross module task match optional-payload alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func pass(maybe: task.i32?) -> task.i32?:
    return maybe
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    let returned: task.i32? = resources.pass(maybe)
    match returned:
    case some(other):
        let first: Int = take_task(task)
        return first + core.task_join_i32(other)
    case none:
        return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertActorTaskOptionalPayloadAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})
}

func assertActorTaskOptionalPayloadAliasTransferDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected actor/task optional-payload alias transfer diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want actor/task optional-payload alias transfer diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForTaskHandleStructFieldEnumPayloadAliasTransfer(t *testing.T) {
	t.Run("same module struct-field alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct TaskBox:
    handle: task.i32

func worker() -> Int:
    return 7

func pass(box: TaskBox) -> TaskBox:
    return box

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: TaskBox = TaskBox(handle: task)
    let returned: TaskBox = pass(box)
    let first: Int = take_task(task)
    return first + core.task_join_i32(returned.handle)
`)
		assertTaskHandleAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.handle'")
	})

	t.Run("cross module struct-field alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = take_task(task)
    return first + core.task_join_i32(returned.handle)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.handle'")
	})

	t.Run("same module enum-payload alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum TaskMsg:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func pass(msg: TaskMsg) -> TaskMsg:
    return msg

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let msg: TaskMsg = TaskMsg.wrap(task)
    let returned: TaskMsg = pass(msg)
    match returned:
    case TaskMsg.wrap(other):
        let first: Int = take_task(task)
        return first + core.task_join_i32(other)
    return 0
`)
		assertTaskHandleAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("cross module enum-payload alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func pass(msg: TaskMsg) -> TaskMsg:
    return msg
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let msg: resources.TaskMsg = resources.TaskMsg.wrap(task)
    let returned: resources.TaskMsg = resources.pass(msg)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = take_task(task)
        return first + core.task_join_i32(other)
    return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})
}

func assertTaskHandleAliasTransferDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected task-handle alias transfer diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want task-handle alias transfer diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForTaskHandleStructFieldEnumPayloadAliasJoin(t *testing.T) {
	t.Run("same module struct-field alias join", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct TaskBox:
    handle: task.i32

func worker() -> Int:
    return 7

func pass(box: TaskBox) -> TaskBox:
    return box

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: TaskBox = TaskBox(handle: task)
    let returned: TaskBox = pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'returned.handle'")
	})

	t.Run("cross module struct-field alias join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'returned.handle'")
	})

	t.Run("same module enum-payload alias join", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum TaskMsg:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: TaskMsg = wrap(task)
    match returned:
    case TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("cross module enum-payload alias join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskMsg = resources.wrap(task)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})
}

func assertTaskHandleAliasJoinDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected task-handle alias join diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want task-handle alias join diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForTaskGroupStructFieldEnumPayloadAliasClose(t *testing.T) {
	t.Run("same module struct-field alias close", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct GroupBox:
    handle: task.group

func pass(box: GroupBox) -> GroupBox:
    return box

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: GroupBox = GroupBox(handle: group)
    let returned: GroupBox = pass(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.handle)
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'returned.handle'")
	})

	t.Run("cross module struct-field alias close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct GroupBox:
    handle: task.group

pub func pass(box: GroupBox) -> GroupBox:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: resources.GroupBox = resources.GroupBox(handle: group)
    let returned: resources.GroupBox = resources.pass(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.handle)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'returned.handle'")
	})

	t.Run("same module enum-payload alias close", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum GroupMsg:
    case wrap(task.group)

func pass(msg: GroupMsg) -> GroupMsg:
    return msg

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let msg: GroupMsg = GroupMsg.wrap(group)
    let returned: GroupMsg = pass(msg)
    match returned:
    case GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

	t.Run("cross module enum-payload alias close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum GroupMsg:
    case wrap(task.group)

pub func pass(msg: GroupMsg) -> GroupMsg:
    return msg
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let msg: resources.GroupMsg = resources.GroupMsg.wrap(group)
    let returned: resources.GroupMsg = resources.pass(msg)
    match returned:
    case resources.GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})
}

func assertTaskGroupAliasCloseDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected task-group alias close diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want task-group alias close diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForTaskHandleGroupOptionalPayloadJoinCloseAliases(t *testing.T) {
	t.Run("same module task-handle if-let optional-payload join", func(t *testing.T) {
		err := testkit.CheckProgram(`
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    if let other = maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle match optional-payload join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func pass(maybe: task.i32?) -> task.i32?:
    return maybe
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    let returned: task.i32? = resources.pass(maybe)
    match returned:
    case some(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    case none:
        return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group if-let optional-payload close", func(t *testing.T) {
		err := testkit.CheckProgram(`
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let other = maybe:
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group match optional-payload close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func pass(maybe: task.group?) -> task.group?:
    return maybe
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    let returned: task.group? = resources.pass(maybe)
    match returned:
    case some(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    case none:
        return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})
}

func TestSafetyDiagnosticCodesForTaskGroupCancelReturnProvenance(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		err := testkit.CheckProgram(`
func cancel(group: task.group) -> task.group
uses runtime:
    return core.task_group_cancel(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'canceled'")
	})

	t.Run("cross module", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func cancel(group: task.group) -> task.group
uses runtime:
    return core.task_group_cancel(group)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = resources.cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'canceled'")
	})
}

func TestSafetyDiagnosticCodesForPtrContainingNestedAggregateCallRejections(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		nested   bool
		wantText string
	}{
		{
			name:     "ptr-containing aggregate owned call",
			mode:     "owned",
			wantText: "borrowed value derived from 'box' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name:     "ptr-containing aggregate consume call",
			mode:     "consume",
			wantText: "borrowed value derived from 'box' cannot be consumed by 'sink'",
		},
		{
			name:     "ptr-containing aggregate inout call",
			mode:     "inout",
			wantText: "borrowed value derived from 'box' cannot be passed as inout to 'sink'",
		},
		{
			name:     "nested ptr-containing aggregate owned call",
			mode:     "owned",
			nested:   true,
			wantText: "borrowed value derived from 'outer' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name:     "nested ptr-containing aggregate consume call",
			mode:     "consume",
			nested:   true,
			wantText: "borrowed value derived from 'outer' cannot be consumed by 'sink'",
		},
		{
			name:     "nested ptr-containing aggregate inout call",
			mode:     "inout",
			nested:   true,
			wantText: "borrowed value derived from 'outer' cannot be passed as inout to 'sink'",
		},
	}

	for _, tt := range tests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(ptrAggregateCallEscapeSource(tt.mode, tt.nested, false))
			assertPtrContainingNestedAggregateCallDiagnostic(t, err, tt.wantText)
		})
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox

` + ptrAggregateSinkSource(tt.mode, tt.nested, false, true),
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(` + ptrAggregateBorrowParam(tt.nested, true) + `) -> Int:
    return sinker.sink(` + ptrAggregateBorrowArg(tt.nested) + `)

func main() -> Int:
    return 0
`,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertPtrContainingNestedAggregateCallDiagnostic(t, err, strings.Replace(tt.wantText, "'sink'", "'lib.sink.sink'", 1))
		})
	}
}

func ptrAggregateCallEscapeSource(mode string, nested bool, qualified bool) string {
	return `
struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

` + ptrAggregateSinkSource(mode, nested, qualified, false) + `
func caller(` + ptrAggregateBorrowParam(nested, qualified) + `) -> Int:
    return sink(` + ptrAggregateBorrowArg(nested) + `)

func main() -> Int:
    return 0
`
}

func ptrAggregateSinkSource(mode string, nested bool, qualified bool, public bool) string {
	typeName := "PtrBox"
	if nested {
		typeName = "OuterBox"
	}
	if qualified {
		typeName = "sinker." + typeName
	}
	param := "value: " + typeName
	switch mode {
	case "consume":
		param = "value: consume " + typeName
	case "inout":
		param = "value: inout " + typeName
	}
	body := "    return 0"
	if mode == "inout" {
		if nested {
			body = "    value = OuterBox(box: PtrBox(raw: 0))\n    return 0"
		} else {
			body = "    value = PtrBox(raw: 0)\n    return 0"
		}
	}
	prefix := ""
	if public {
		prefix = "pub "
	}
	return prefix + "func sink(" + param + ") -> Int:\n" + body + "\n"
}

func ptrAggregateBorrowParam(nested bool, qualified bool) string {
	typeName := "PtrBox"
	name := "box"
	if nested {
		typeName = "OuterBox"
		name = "outer"
	}
	if qualified {
		typeName = "sinker." + typeName
	}
	return name + ": borrow " + typeName
}

func ptrAggregateBorrowArg(nested bool) string {
	if nested {
		return "outer"
	}
	return "box"
}

func assertPtrContainingNestedAggregateCallDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected ptr-containing/nested aggregate call diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want ptr-containing/nested aggregate call diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForPtrEnumPayloadCallRejections(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		wantText string
	}{
		{
			name:     "owned call",
			mode:     "owned",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name:     "consume call",
			mode:     "consume",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name:     "inout call",
			mode:     "inout",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'sink'",
		},
	}

	for _, tt := range tests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(ptrEnumPayloadCallEscapeSource(tt.mode, false, false))
			assertPtrEnumPayloadCallDiagnostic(t, err, tt.wantText)
		})
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": `module lib.sink

pub enum PtrMsg:
    case raw(ptr)

` + ptrEnumPayloadSinkSource(tt.mode, false, true),
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    ` + ptrEnumPayloadCallBody(tt.mode, true) + `

func main() -> Int:
    return 0
`,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertPtrEnumPayloadCallDiagnostic(t, err, strings.Replace(tt.wantText, "'sink'", "'lib.sink.sink'", 1))
		})
	}
}

func ptrEnumPayloadCallEscapeSource(mode string, qualified bool, public bool) string {
	return `
enum PtrMsg:
    case raw(ptr)

` + ptrEnumPayloadSinkSource(mode, qualified, public) + `
func caller(x: borrow ptr) -> Int:
    ` + ptrEnumPayloadCallBody(mode, qualified) + `

func main() -> Int:
    return 0
`
}

func ptrEnumPayloadSinkSource(mode string, qualified bool, public bool) string {
	typeName := "PtrMsg"
	if qualified {
		typeName = "sinker." + typeName
	}
	param := "value: " + typeName
	switch mode {
	case "consume":
		param = "value: consume " + typeName
	case "inout":
		param = "value: inout " + typeName
	}
	body := "    return 0"
	if mode == "inout" {
		body = "    value = PtrMsg.raw(0)\n    return 0"
	}
	prefix := ""
	if public {
		prefix = "pub "
	}
	return prefix + "func sink(" + param + ") -> Int:\n" + body + "\n"
}

func ptrEnumPayloadCallBody(mode string, qualified bool) string {
	typeName := "PtrMsg"
	sinkName := "sink"
	if qualified {
		typeName = "sinker." + typeName
		sinkName = "sinker.sink"
	}
	switch mode {
	case "consume":
		return "let msg: " + typeName + " = " + typeName + ".raw(x)\n    return " + sinkName + "(msg)"
	case "inout":
		return "var msg: " + typeName + " = " + typeName + ".raw(x)\n    return " + sinkName + "(msg)"
	default:
		return "return " + sinkName + "(" + typeName + ".raw(x))"
	}
}

func assertPtrEnumPayloadCallDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected ptr enum-payload call diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want ptr enum-payload call diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForPtrOptionalPayloadCallRejections(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		wantText string
	}{
		{
			name:     "owned call",
			mode:     "owned",
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name:     "consume call",
			mode:     "consume",
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'sink'",
		},
		{
			name:     "inout call",
			mode:     "inout",
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'sink'",
		},
	}

	for _, tt := range tests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(ptrOptionalPayloadCallEscapeSource(tt.mode, false, false))
			assertPtrOptionalPayloadCallDiagnostic(t, err, tt.wantText)
		})
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": `module lib.sink

` + ptrOptionalPayloadSinkSource(tt.mode, false, true),
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(maybe: borrow ptr?) -> Int:
    ` + ptrOptionalPayloadCallBody(tt.mode, true) + `

func main() -> Int:
    return 0
`,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertPtrOptionalPayloadCallDiagnostic(t, err, strings.Replace(tt.wantText, "'sink'", "'lib.sink.sink'", 1))
		})
	}
}

func ptrOptionalPayloadCallEscapeSource(mode string, qualified bool, public bool) string {
	return ptrOptionalPayloadSinkSource(mode, qualified, public) + `
func caller(maybe: borrow ptr?) -> Int:
    ` + ptrOptionalPayloadCallBody(mode, qualified) + `

func main() -> Int:
    return 0
`
}

func ptrOptionalPayloadSinkSource(mode string, qualified bool, public bool) string {
	param := "raw: ptr"
	switch mode {
	case "consume":
		param = "raw: consume ptr"
	case "inout":
		param = "raw: inout ptr"
	}
	body := "    return 0"
	if mode == "inout" {
		body = "    raw = 0\n    return 0"
	}
	prefix := ""
	if public {
		prefix = "pub "
	}
	return prefix + "func sink(" + param + ") -> Int:\n" + body + "\n"
}

func ptrOptionalPayloadCallBody(mode string, qualified bool) string {
	sinkName := "sink"
	if qualified {
		sinkName = "sinker.sink"
	}
	return `match maybe:
    case some(raw):
        return ` + sinkName + `(raw)
    case none:
        return 0`
}

func assertPtrOptionalPayloadCallDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected ptr optional-payload call diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want ptr optional-payload call diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForSliceOptionalPayloadCallRejections(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		wantText string
	}{
		{
			name:     "owned call",
			mode:     "owned",
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name:     "consume call",
			mode:     "consume",
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'sink'",
		},
		{
			name:     "inout call",
			mode:     "inout",
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'sink'",
		},
	}

	for _, tt := range tests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(sliceOptionalPayloadCallEscapeSource(tt.mode, false, false))
			assertSliceOptionalPayloadCallDiagnostic(t, err, tt.wantText)
		})
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": `module lib.sink

` + sliceOptionalPayloadSinkSource(tt.mode, true),
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(maybe: borrow []u8?) -> Int:
    ` + sliceOptionalPayloadCallBody(true) + `

func main() -> Int:
    return 0
`,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertSliceOptionalPayloadCallDiagnostic(t, err, strings.Replace(tt.wantText, "'sink'", "'lib.sink.sink'", 1))
		})
	}
}

func sliceOptionalPayloadCallEscapeSource(mode string, qualified bool, public bool) string {
	return sliceOptionalPayloadSinkSource(mode, public) + `
func caller(maybe: borrow []u8?) -> Int:
    ` + sliceOptionalPayloadCallBody(qualified) + `

func main() -> Int:
    return 0
`
}

func sliceOptionalPayloadSinkSource(mode string, public bool) string {
	param := "raw: []u8"
	switch mode {
	case "consume":
		param = "raw: consume []u8"
	case "inout":
		param = "raw: inout []u8"
	}
	body := "    return 0"
	if mode == "inout" {
		body = "    raw = raw\n    return 0"
	}
	prefix := ""
	if public {
		prefix = "pub "
	}
	return prefix + "func sink(" + param + ") -> Int:\n" + body + "\n"
}

func sliceOptionalPayloadCallBody(qualified bool) string {
	sinkName := "sink"
	if qualified {
		sinkName = "sinker.sink"
	}
	return `match maybe:
    case some(raw):
        return ` + sinkName + `(raw)
    case none:
        return 0`
}

func assertSliceOptionalPayloadCallDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected slice optional-payload call diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want slice optional-payload call diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForFunctionTypedSliceAggregateCallbackCallRejections(t *testing.T) {
	type caseDef struct {
		name       string
		target     string
		aggregate  string
		mode       string
		cross      bool
		wantCaller string
	}
	var cases []caseDef
	for _, target := range []string{"value", "field", "payload"} {
		for _, aggregate := range []string{"struct", "enum"} {
			for _, mode := range []string{"owned", "consume", "inout"} {
				cases = append(cases, caseDef{
					name:       target + " " + aggregate + " " + mode,
					target:     target,
					aggregate:  aggregate,
					mode:       mode,
					wantCaller: functionTypedSliceCallbackWantCaller(target),
				})
			}
		}
	}
	for _, target := range []string{"field", "payload"} {
		for _, mode := range []string{"owned", "consume", "inout"} {
			cases = append(cases, caseDef{
				name:       "cross " + target + " " + mode,
				target:     target,
				aggregate:  "struct",
				mode:       mode,
				cross:      true,
				wantCaller: functionTypedSliceCallbackWantCaller(target),
			})
		}
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.cross {
				files := functionTypedSliceCallbackCrossModuleFiles(tt.target, tt.aggregate, tt.mode)
				tmp := t.TempDir()
				testkit.WriteFiles(t, tmp, files)
				world, loadErr := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
				if loadErr != nil {
					t.Fatalf("LoadWorld: %v", loadErr)
				}
				_, err = compiler.CheckWorld(world)
			} else {
				err = testkit.CheckProgram(functionTypedSliceCallbackSource(tt.target, tt.aggregate, tt.mode, false))
			}
			wantText := functionTypedSliceCallbackWantText(tt.mode, tt.wantCaller)
			assertFunctionTypedSliceAggregateCallbackDiagnostic(t, err, wantText)
		})
	}
}

func functionTypedSliceCallbackSource(target, aggregate, mode string, qualified bool) string {
	return functionTypedSliceCallbackTypeDecls(aggregate, false) + "\n" +
		functionTypedSliceCallbackTargetDecl(target, aggregate, mode, false) + "\n" +
		"func caller(" + functionTypedSliceCallbackParam(target, aggregate, mode, qualified) + ", x: borrow []u8) -> Int:\n" +
		functionTypedSliceCallbackBody(target, aggregate, mode, qualified) + "\n\n" +
		"func main() -> Int:\n    return 0\n"
}

func functionTypedSliceCallbackCrossModuleFiles(target, aggregate, mode string) map[string]string {
	return map[string]string{
		"lib/callbacks.t4": "module lib.callbacks\n\n" +
			functionTypedSliceCallbackTypeDecls(aggregate, true) + "\n" +
			functionTypedSliceCallbackTargetDecl(target, aggregate, mode, true),
		"app/main.t4": "module app.main\n" +
			"import lib.callbacks as callbacks\n\n" +
			"func caller(" + functionTypedSliceCallbackParam(target, aggregate, mode, true) + ", x: borrow []u8) -> Int:\n" +
			functionTypedSliceCallbackBody(target, aggregate, mode, true) + "\n\n" +
			"func main() -> Int:\n    return 0\n",
	}
}

func functionTypedSliceCallbackTypeDecls(aggregate string, public bool) string {
	prefix := ""
	if public {
		prefix = "pub "
	}
	if aggregate == "enum" {
		return prefix + "enum BufMsg:\n    case send([]u8)\n"
	}
	return prefix + "struct BufBox:\n    buf: []u8\n"
}

func functionTypedSliceCallbackTargetDecl(target, aggregate, mode string, public bool) string {
	if target == "value" {
		return ""
	}
	prefix := ""
	if public {
		prefix = "pub "
	}
	callbackType := "fn(" + functionTypedSliceCallbackParamType(aggregate, mode, false) + ") -> Int"
	if target == "payload" {
		return prefix + "enum Choice:\n    case some(" + callbackType + ")\n    case empty\n"
	}
	return prefix + "struct Handler:\n    cb: " + callbackType + "\n"
}

func functionTypedSliceCallbackParam(target, aggregate, mode string, qualified bool) string {
	paramType := functionTypedSliceCallbackParamType(aggregate, mode, qualified)
	switch target {
	case "payload":
		typeName := "Choice"
		if qualified {
			typeName = "callbacks.Choice"
		}
		return "choice: " + typeName
	case "field":
		typeName := "Handler"
		if qualified {
			typeName = "callbacks.Handler"
		}
		return "h: " + typeName
	default:
		return "cb: fn(" + paramType + ") -> Int"
	}
}

func functionTypedSliceCallbackParamType(aggregate, mode string, qualified bool) string {
	typeName := "BufBox"
	if aggregate == "enum" {
		typeName = "BufMsg"
	}
	if qualified {
		typeName = "callbacks." + typeName
	}
	switch mode {
	case "consume":
		return "consume " + typeName
	case "inout":
		return "inout " + typeName
	default:
		return typeName
	}
}

func functionTypedSliceCallbackBody(target, aggregate, mode string, qualified bool) string {
	callable := "cb"
	if target == "field" {
		callable = "h.cb"
	}
	arg := functionTypedSliceCallbackAggregateValue(aggregate, "x", qualified)
	if mode == "consume" {
		local := functionTypedSliceCallbackLocalName(aggregate)
		prefix := "    let " + local + ": " + functionTypedSliceCallbackParamType(aggregate, "owned", qualified) + " = " + arg + "\n"
		return prefix + functionTypedSliceCallbackCall(target, callable, local, qualified)
	}
	if mode == "inout" {
		local := functionTypedSliceCallbackLocalName(aggregate)
		prefix := "    var " + local + ": " + functionTypedSliceCallbackParamType(aggregate, "owned", qualified) + " = " + arg + "\n"
		return prefix + functionTypedSliceCallbackCall(target, callable, local, qualified)
	}
	return functionTypedSliceCallbackCall(target, callable, arg, qualified)
}

func functionTypedSliceCallbackCall(target, callable, arg string, qualified bool) string {
	if target != "payload" {
		return "    return " + callable + "(" + arg + ")"
	}
	casePrefix := "Choice"
	if qualified {
		casePrefix = "callbacks.Choice"
	}
	return "    match choice:\n" +
		"    case " + casePrefix + ".some(cb):\n" +
		"        return " + callable + "(" + arg + ")\n" +
		"    case " + casePrefix + ".empty:\n" +
		"        return 0"
}

func functionTypedSliceCallbackAggregateValue(aggregate, source string, qualified bool) string {
	if aggregate == "enum" {
		typeName := "BufMsg"
		if qualified {
			typeName = "callbacks.BufMsg"
		}
		return typeName + ".send(" + source + ")"
	}
	typeName := "BufBox"
	if qualified {
		typeName = "callbacks.BufBox"
	}
	return typeName + "(buf: " + source + ")"
}

func functionTypedSliceCallbackLocalName(aggregate string) string {
	if aggregate == "enum" {
		return "msg"
	}
	return "box"
}

func functionTypedSliceCallbackWantCaller(target string) string {
	switch target {
	case "field":
		return "function-typed struct field call 'h.cb'"
	case "payload":
		return "function-typed enum payload call 'cb'"
	default:
		return "callback 'cb'"
	}
}

func functionTypedSliceCallbackWantText(mode, caller string) string {
	switch mode {
	case "consume":
		return "borrowed value derived from 'x' cannot be consumed by " + caller
	case "inout":
		return "borrowed value derived from 'x' cannot be passed as inout to " + caller
	default:
		return "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of " + caller
	}
}

func assertFunctionTypedSliceAggregateCallbackDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected function-typed slice aggregate callback diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want function-typed slice aggregate callback diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForGenericProtocolRequirementOwnershipMismatch(t *testing.T) {
	tests := []struct {
		name     string
		cross    bool
		wantText string
	}{
		{
			name:     "same module",
			wantText: "method 'Box.map' does not match protocol 'Mapper' requirement 'map': parameter 2 ownership differs: expected 'consume', got 'owned'",
		},
		{
			name:     "cross module",
			cross:    true,
			wantText: "method 'lib.model.Box.map' does not match protocol 'lib.model.Mapper' requirement 'map': parameter 2 ownership differs: expected 'consume', got 'owned'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.cross {
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
				tmp := t.TempDir()
				testkit.WriteFiles(t, tmp, files)
				world, loadErr := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
				if loadErr != nil {
					t.Fatalf("LoadWorld: %v", loadErr)
				}
				_, err = compiler.CheckWorld(world)
			} else {
				err = testkit.CheckProgram(`
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
`)
			}
			assertGenericProtocolRequirementOwnershipMismatchDiagnostic(t, err, tt.wantText)
		})
	}
}

func TestSafetyDiagnosticCodesForProtocolImplOwnershipMismatch(t *testing.T) {
	tests := []struct {
		name     string
		cross    bool
		wantText string
	}{
		{
			name:     "same module",
			wantText: "method 'Box.sink' does not match protocol 'Sink' requirement 'sink': parameter 1 ownership differs: expected 'consume', got 'owned'",
		},
		{
			name:     "cross module",
			cross:    true,
			wantText: "method 'lib.model.Box.sink' does not match protocol 'lib.model.Sink' requirement 'sink': parameter 1 ownership differs: expected 'consume', got 'owned'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.cross {
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
				tmp := t.TempDir()
				testkit.WriteFiles(t, tmp, files)
				world, loadErr := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
				if loadErr != nil {
					t.Fatalf("LoadWorld: %v", loadErr)
				}
				_, err = compiler.CheckWorld(world)
			} else {
				err = testkit.CheckProgram(`
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
`)
			}
			assertProtocolImplOwnershipMismatchDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertProtocolImplOwnershipMismatchDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected protocol impl ownership mismatch diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSemantic || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSemantic)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want protocol impl ownership mismatch diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSemantic+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSemantic)
	}
}

func assertGenericProtocolRequirementOwnershipMismatchDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected generic protocol requirement ownership mismatch diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSemantic || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSemantic)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want generic protocol ownership mismatch diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSemantic+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSemantic)
	}
}

func TestSafetyDiagnosticCodesForTypedErrorResourceAliasFinalization(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "catch payload alias after join",
			src: `
enum TaskErr:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch fail(task):
    case TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		},
		{
			name: "rethrow through try payload alias after join",
			src: `
enum TaskErr:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)

func wrapper(task: task.i32) -> Int throws TaskErr
uses runtime:
    return try fail(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch wrapper(task):
    case TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, "cannot use joined resource 'other'") {
				t.Fatalf("message = %q, want joined alias diagnostic", diag.Message)
			}
			raw, err := json.Marshal(diag)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForResourceFinalizationMergeDiagnostics(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "branch maybe joined task handle",
			src: `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    if 1:
        let _: Int = core.task_join_i32(task)
    return core.task_join_i32(task)
`,
			wantText: "may have been joined after control-flow merge",
		},
		{
			name: "loop maybe closed task group",
			src: `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    var i: Int = 0
    while i < 1:
        let closed: Int = core.task_group_close(group)
        i = i + 1
    return core.task_group_close(group)
`,
			wantText: "may have been closed after control-flow merge",
		},
		{
			name: "match maybe freed island",
			src: `
enum Choice:
    case freeit
    case keep

func choose() -> Choice:
    return Choice.freeit

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let choice: Choice = choose()
        match choice:
        case Choice.freeit:
            free(isl)
        case Choice.keep:
            let kept: Int = 0
        free(isl)
    }
    return 0
`,
			wantText: "may have been freed after control-flow merge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
			raw, err := json.Marshal(diag)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForOptionalPayloadWholeValueConsumeAndFree(t *testing.T) {
	t.Run("same module payload consume", func(t *testing.T) {
		src := `
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
`
		err := testkit.CheckProgram(src)
		assertOptionalPayloadWholeValueDiagnostic(t, err, "cannot use consumed value 'maybe.$elem'")
	})

	t.Run("cross module payload consume", func(t *testing.T) {
		files := map[string]string{
			"lib/leaks.t4": `module lib.leaks

pub func take(raw: consume ptr) -> ptr:
    return raw

pub func use(value: ptr?) -> Int:
    return 0

pub func leak(maybe: ptr?) -> Int:
    match maybe:
    case some(raw):
        let moved: ptr = take(raw)
    case none:
        let untouched: Int = 0
    return use(maybe)
`,
			"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertOptionalPayloadWholeValueDiagnostic(t, err, "cannot use consumed value 'maybe.$elem'")
	})

	t.Run("same module payload free", func(t *testing.T) {
		src := `
func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        match maybe:
        case some(other):
            free(other)
            return use(maybe)
        case none:
            return 0
    }
    return 0
`
		err := testkit.CheckProgram(src)
		assertOptionalPayloadWholeValueDiagnostic(t, err, "cannot use freed resource 'maybe.$elem'")
	})

	t.Run("cross module payload free", func(t *testing.T) {
		files := map[string]string{
			"lib/leaks.t4": `module lib.leaks

pub func use(value: island?) -> Int:
    return 0

pub func leak(maybe: island?) -> Int
uses alloc, islands, mem:
    unsafe {
        match maybe:
        case some(other):
            free(other)
            return use(maybe)
        case none:
            return 0
    }
    return 0
`,
			"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertOptionalPayloadWholeValueDiagnostic(t, err, "cannot use freed resource 'maybe.$elem'")
	})
}

func assertOptionalPayloadWholeValueDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected optional-payload whole-value diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want optional-payload whole-value diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodeForCallableMutableCaptureGlobalEscape(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var total: Int = 1
    cb = fn(x: Int) -> Int:
        return total + x
    return 0
`)
	file, err := compiler.ParseFile(src, "callable_mutable_capture_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected callable mutable capture global escape diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "global-escaped function value captures mutable local 'total'") {
		t.Fatalf("message = %q, want callable mutable capture global escape", diag.Message)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrOptionalAssignmentGlobalEscape(t *testing.T) {
	tests := []struct {
		name string
		file string
		src  string
	}{
		{
			name: "if-let",
			file: "borrowed_ptr_optional_assignment_global_escape.tetra",
			src: `
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
`,
		},
		{
			name: "match",
			file: "borrowed_ptr_optional_assignment_match_global_escape.tetra",
			src: `
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
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), tt.file)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedPtrOptionalAssignmentGlobalEscapeDiagnostic(t, err)
		})
	}

	crossModuleTests := []struct {
		name string
		body string
	}{
		{
			name: "if-let",
			body: `
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
`,
		},
		{
			name: "match",
			body: `
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
		},
	}
	for _, tt := range crossModuleTests {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": `module lib.leaks

var leaked: ptr = 0

pub func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
` + tt.body,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedPtrOptionalAssignmentGlobalEscapeDiagnostic(t, err)
		})
	}
}

func assertBorrowedPtrOptionalAssignmentGlobalEscapeDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr optional assignment global escape diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'x' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed ptr optional assignment global escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrEnumAliasReturnEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
enum PtrMsg:
    case raw(ptr)

func leak(x: borrow ptr) -> PtrMsg:
    let msg: PtrMsg = PtrMsg.raw(x)
    return msg

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_enum_alias_return_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrEnumAliasReturnDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrEnumAliasReturnDiagnostic(t, err)
	})
}

func assertBorrowedPtrEnumAliasReturnDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr enum alias return diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'x' cannot escape via return") {
		t.Fatalf("message = %q, want borrowed ptr enum alias return escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrAggregateReturnEscapes(t *testing.T) {
	sameModuleTests := []struct {
		name         string
		file         string
		src          string
		borrowedName string
	}{
		{
			name: "whole aggregate",
			file: "borrowed_ptr_aggregate_return_escape.tetra",
			src: `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    return box

func main() -> Int:
    return 0
`,
			borrowedName: "box",
		},
		{
			name: "field",
			file: "borrowed_ptr_aggregate_field_return_escape.tetra",
			src: `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> ptr:
    return box.raw

func main() -> Int:
    return 0
`,
			borrowedName: "box",
		},
		{
			name: "alias",
			file: "borrowed_ptr_aggregate_alias_return_escape.tetra",
			src: `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    let alias: PtrBox = box
    return alias

func main() -> Int:
    return 0
`,
			borrowedName: "box",
		},
		{
			name: "nested field",
			file: "borrowed_ptr_nested_aggregate_field_return_escape.tetra",
			src: `
struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

func leak(outer: borrow OuterBox) -> ptr:
    return outer.box.raw

func main() -> Int:
    return 0
`,
			borrowedName: "outer",
		},
	}
	for _, tt := range sameModuleTests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), tt.file)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedPtrAggregateReturnDiagnostic(t, err, tt.borrowedName)
		})
	}

	crossModuleTests := []struct {
		name         string
		body         string
		returnType   string
		paramType    string
		borrowedName string
	}{
		{
			name: "whole aggregate",
			body: `
    return box
`,
			returnType:   "model.PtrBox",
			paramType:    "model.PtrBox",
			borrowedName: "box",
		},
		{
			name: "field",
			body: `
    return box.raw
`,
			returnType:   "ptr",
			paramType:    "model.PtrBox",
			borrowedName: "box",
		},
		{
			name: "alias",
			body: `
    let alias: model.PtrBox = box
    return alias
`,
			returnType:   "model.PtrBox",
			paramType:    "model.PtrBox",
			borrowedName: "box",
		},
		{
			name: "nested field",
			body: `
    return outer.box.raw
`,
			returnType:   "ptr",
			paramType:    "model.OuterBox",
			borrowedName: "outer",
		},
	}
	for _, tt := range crossModuleTests {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			paramName := tt.borrowedName
			files := map[string]string{
				"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox
`,
				"app/main.t4": `module app.main
import lib.model as model

func leak(` + paramName + `: borrow ` + tt.paramType + `) -> ` + tt.returnType + `:` + tt.body + `
func main() -> Int:
    return 0
`,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedPtrAggregateReturnDiagnostic(t, err, tt.borrowedName)
		})
	}
}

func assertBorrowedPtrAggregateReturnDiagnostic(t *testing.T, err error, borrowedName string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr aggregate return diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	wantText := "borrowed local '" + borrowedName + "' cannot escape via return"
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed ptr aggregate return escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrEnumPayloadEscapes(t *testing.T) {
	sameModuleTests := []struct {
		name     string
		file     string
		src      string
		wantText string
	}{
		{
			name: "return",
			file: "borrowed_ptr_enum_payload_return_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'msg' cannot escape via return",
		},
		{
			name: "global",
			file: "borrowed_ptr_enum_payload_global_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'msg' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "inout",
			file: "borrowed_ptr_enum_payload_inout_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'msg' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range sameModuleTests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), tt.file)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedPtrEnumPayloadDiagnostic(t, err, tt.wantText)
		})
	}

	crossModuleTests := []struct {
		name     string
		body     string
		extra    string
		params   string
		retType  string
		wantText string
	}{
		{
			name: "return",
			body: `
    match msg:
    case model.PtrMsg.raw(raw):
        return raw
    case model.PtrMsg.empty:
        return 0
`,
			params:   "msg: borrow model.PtrMsg",
			retType:  "ptr",
			wantText: "borrowed local 'msg' cannot escape via return",
		},
		{
			name: "global",
			extra: `
var leaked: ptr = 0
`,
			body: `
    match msg:
    case model.PtrMsg.raw(raw):
        leaked = raw
        return 0
    case model.PtrMsg.empty:
        return 0
`,
			params:   "msg: borrow model.PtrMsg",
			retType:  "Int",
			wantText: "borrowed local 'msg' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "inout",
			body: `
    match msg:
    case model.PtrMsg.raw(raw):
        out = raw
        return 0
    case model.PtrMsg.empty:
        return 0
`,
			params:   "msg: borrow model.PtrMsg, out: inout ptr",
			retType:  "Int",
			wantText: "borrowed local 'msg' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range crossModuleTests {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`,
				"app/main.t4": `module app.main
import lib.model as model
` + tt.extra + `
func leak(` + tt.params + `) -> ` + tt.retType + `:` + tt.body + `
func main() -> Int:
    return 0
`,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedPtrEnumPayloadDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertBorrowedPtrEnumPayloadDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr enum-payload diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed ptr enum-payload escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrOptionalPayloadEscapes(t *testing.T) {
	type caseDef struct {
		name     string
		body     string
		extra    string
		params   string
		retType  string
		wantText string
	}

	cases := []caseDef{
		{
			name: "if-let return",
			body: `
    if let raw = maybe:
        return raw
    else:
        return 0
`,
			params:   "maybe: borrow ptr?",
			retType:  "ptr",
			wantText: "borrowed local 'maybe' cannot escape via return",
		},
		{
			name: "match return",
			body: `
    match maybe:
    case some(raw):
        return raw
    case none:
        return 0
`,
			params:   "maybe: borrow ptr?",
			retType:  "ptr",
			wantText: "borrowed local 'maybe' cannot escape via return",
		},
		{
			name: "if-let global",
			extra: `
var leaked: ptr = 0
`,
			body: `
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
`,
			params:   "maybe: borrow ptr?",
			retType:  "Int",
			wantText: "borrowed local 'maybe' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "match global",
			extra: `
var leaked: ptr = 0
`,
			body: `
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
			params:   "maybe: borrow ptr?",
			retType:  "Int",
			wantText: "borrowed local 'maybe' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "if-let inout",
			body: `
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0
`,
			params:   "maybe: borrow ptr?, out: inout ptr",
			retType:  "Int",
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
		},
		{
			name: "match inout",
			body: `
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0
`,
			params:   "maybe: borrow ptr?, out: inout ptr",
			retType:  "Int",
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
		},
	}

	for _, tt := range cases {
		t.Run("same module "+tt.name, func(t *testing.T) {
			src := tt.extra + `
func leak(` + tt.params + `) -> ` + tt.retType + `:` + tt.body + `
func main() -> Int:
    return 0
`
			file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_optional_payload_"+strings.ReplaceAll(tt.name, " ", "_")+".tetra")
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedPtrOptionalPayloadDiagnostic(t, err, tt.wantText)
		})
	}

	for _, tt := range cases {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": `module lib.leaks
` + tt.extra + `
pub func leak(` + tt.params + `) -> ` + tt.retType + `:` + tt.body,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedPtrOptionalPayloadDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertBorrowedPtrOptionalPayloadDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr optional-payload diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed ptr optional-payload escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedSliceOptionalPayloadInoutGlobalEscapes(t *testing.T) {
	type caseDef struct {
		name     string
		body     string
		extra    string
		params   string
		wantText string
	}

	cases := []caseDef{
		{
			name: "if-let inout",
			body: `
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0
`,
			params:   "maybe: borrow []u8?, out: inout []u8",
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
		},
		{
			name: "match inout",
			body: `
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0
`,
			params:   "maybe: borrow []u8?, out: inout []u8",
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
		},
		{
			name: "if-let global",
			extra: `
var leaked: []u8? = none
`,
			body: `
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
			`,
			params:   "maybe: borrow []u8?",
			wantText: "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot be stored in global",
		},
		{
			name: "match global",
			extra: `
var leaked: []u8? = none
`,
			body: `
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
			params:   "maybe: borrow []u8?",
			wantText: "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot be stored in global",
		},
	}

	for _, tt := range cases {
		t.Run("same module "+tt.name, func(t *testing.T) {
			src := tt.extra + `
func leak(` + tt.params + `) -> Int:` + tt.body + `
func main() -> Int:
    return 0
`
			file, err := compiler.ParseFile([]byte(src), "borrowed_slice_optional_payload_"+strings.ReplaceAll(tt.name, " ", "_")+".tetra")
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedSliceOptionalPayloadDiagnostic(t, err, tt.wantText)
		})
	}

	for _, tt := range cases {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": `module lib.leaks
` + tt.extra + `
pub func leak(` + tt.params + `) -> Int:` + tt.body,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedSliceOptionalPayloadDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertBorrowedSliceOptionalPayloadDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed slice optional-payload diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed slice optional-payload escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedNestedSliceStructEscapes(t *testing.T) {
	sameModuleTests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal return",
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
			name: "alias return",
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
			name: "inout assignment",
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
		{
			name: "global assignment",
			src: `
struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox

var leaked: OuterBox

func leak(read: borrow []u8) -> Int:
    leaked = OuterBox(box: BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot be stored in global",
		},
	}

	for _, tt := range sameModuleTests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), "borrowed_nested_slice_struct_escape.tetra")
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedNestedSliceStructDiagnostic(t, err, tt.wantText)
		})
	}

	crossModuleTests := []struct {
		name     string
		files    map[string]string
		wantText string
	}{
		{
			name: "literal return",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(x: borrow []u8) -> OuterBox:
    return OuterBox(box: BufBox(buf: x))
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias return",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(x: borrow []u8) -> OuterBox:
    let outer: OuterBox = OuterBox(box: BufBox(buf: x))
    return outer
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout assignment",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(read: borrow []u8, out: inout OuterBox) -> Int:
    out = OuterBox(box: BufBox(buf: read))
    return 0
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
		{
			name: "global assignment",
			files: map[string]string{
				"lib/model.t4": `module lib.model

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox
`,
				"app/main.t4": `module app.main
import lib.model as model

var leaked: model.OuterBox

func leak(read: borrow []u8) -> Int:
    leaked = model.OuterBox(box: model.BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			},
			wantText: "aggregate 'lib.model.BufBox' contains borrowed slice field 'buf' that cannot be stored in global",
		},
	}

	for _, tt := range crossModuleTests {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, tt.files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedNestedSliceStructDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertBorrowedNestedSliceStructDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed nested slice struct diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed nested slice struct escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedNestedSliceEnumPayloadEscapes(t *testing.T) {
	sameModuleTests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal return",
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
			name: "alias return",
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
			name: "inout assignment",
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
		{
			name: "global assignment",
			src: `
struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty

var leaked: OuterMsg

func leak(read: borrow []u8) -> Int:
    leaked = OuterMsg.wrap(BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot be stored in global",
		},
	}

	for _, tt := range sameModuleTests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), "borrowed_nested_slice_enum_payload_escape.tetra")
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedNestedSliceEnumPayloadDiagnostic(t, err, tt.wantText)
		})
	}

	crossModuleTests := []struct {
		name     string
		files    map[string]string
		wantText string
	}{
		{
			name: "literal return",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(x: borrow []u8) -> OuterMsg:
    return OuterMsg.wrap(BufBox(buf: x))
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias return",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(x: borrow []u8) -> OuterMsg:
    let msg: OuterMsg = OuterMsg.wrap(BufBox(buf: x))
    return msg
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout assignment",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(read: borrow []u8, out: inout OuterMsg) -> Int:
    out = OuterMsg.wrap(BufBox(buf: read))
    return 0
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
		{
			name: "global assignment",
			files: map[string]string{
				"lib/model.t4": `module lib.model

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty
`,
				"app/main.t4": `module app.main
import lib.model as model

var leaked: model.OuterMsg

func leak(read: borrow []u8) -> Int:
    leaked = model.OuterMsg.wrap(model.BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			},
			wantText: "aggregate 'lib.model.BufBox' contains borrowed slice field 'buf' that cannot be stored in global",
		},
	}

	for _, tt := range crossModuleTests {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, tt.files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedNestedSliceEnumPayloadDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertBorrowedNestedSliceEnumPayloadDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed nested slice enum-payload diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed nested slice enum-payload escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrOptionalGlobalAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
var leaked: ptr? = none

func leak(x: borrow ptr) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_optional_global_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrOptionalGlobalAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrOptionalGlobalAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedPtrOptionalGlobalAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr optional global assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'x' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed ptr optional global assignment escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrAggregateOptionalGlobalAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
struct PtrBox:
    raw: ptr

var leaked: PtrBox? = none

func leak(box: borrow PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_aggregate_optional_global_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrAggregateOptionalGlobalAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrAggregateOptionalGlobalAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedPtrAggregateOptionalGlobalAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr aggregate optional global assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'box' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed ptr aggregate optional global assignment escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedSliceGlobalAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
var leaked: []u8

func leak(x: borrow []u8) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_slice_global_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedSliceGlobalAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedSliceGlobalAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedSliceGlobalAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed slice global assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'x' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed slice global assignment escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrWholeAggregateGlobalAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
struct PtrBox:
    raw: ptr

var leaked: PtrBox

func leak(box: borrow PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_whole_aggregate_global_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrWholeAggregateGlobalAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrWholeAggregateGlobalAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedPtrWholeAggregateGlobalAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr whole-aggregate global assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'box' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed ptr whole-aggregate global assignment escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrEnumWholeValueGlobalAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
enum PtrMsg:
    case raw(ptr)

var leaked: PtrMsg

func leak(msg: borrow PtrMsg) -> Int:
    leaked = msg
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_enum_whole_value_global_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrEnumWholeValueGlobalAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
		files := map[string]string{
			"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrEnumWholeValueGlobalAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedPtrEnumWholeValueGlobalAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr enum whole-value global assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'msg' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed ptr enum whole-value global assignment escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrGlobalFieldTargetAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
struct PtrBox:
    raw: ptr

var leaked: PtrBox

func leak(box: borrow PtrBox) -> Int:
    leaked.raw = box.raw
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_global_field_target_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrGlobalFieldTargetAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrGlobalFieldTargetAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedPtrGlobalFieldTargetAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr global field target assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'box' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed ptr global field target assignment escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrAggregateAndNestedGlobalFieldEscapes(t *testing.T) {
	t.Run("same module aggregate", func(t *testing.T) {
		src := `
struct PtrBox:
    raw: ptr

var leaked: ptr = 0

func leak(box: borrow PtrBox) -> Int:
    leaked = box.raw
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_aggregate_global_field_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrAggregateGlobalFieldDiagnostic(t, err, "box")
	})

	t.Run("same module nested aggregate", func(t *testing.T) {
		src := `
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
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_nested_aggregate_global_field_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrAggregateGlobalFieldDiagnostic(t, err, "outer")
	})

	t.Run("cross module aggregate", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrAggregateGlobalFieldDiagnostic(t, err, "box")
	})

	t.Run("cross module nested aggregate", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrAggregateGlobalFieldDiagnostic(t, err, "outer")
	})
}

func assertBorrowedPtrAggregateGlobalFieldDiagnostic(t *testing.T, err error, borrowedName string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr aggregate global field diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	wantText := "borrowed local '" + borrowedName + "' cannot escape via global assignment to 'leaked'"
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed ptr aggregate global field escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedFixedArrayEscapes(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		src      string
		wantText string
	}{
		{
			name: "alias return",
			file: "borrowed_fixed_array_alias_return_escape.tetra",
			src: `
func leak(x: borrow [2]Int) -> [2]Int:
    let y: [2]Int = x
    return y

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "direct global assignment",
			file: "borrowed_fixed_array_global_escape.tetra",
			src: `
struct ArrayBox:
    items: [2]Int

var leaked: ArrayBox

func leak(x: borrow [2]Int) -> Int:
    leaked.items = x
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "optional global assignment",
			file: "borrowed_fixed_array_optional_global_escape.tetra",
			src: `
var leaked: [2]Int? = none

func leak(x: borrow [2]Int) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "inout assignment",
			file: "borrowed_fixed_array_inout_escape.tetra",
			src: `
func leak(x: borrow [2]Int, out: inout [2]Int) -> Int:
    out = x
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), tt.file)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected borrowed fixed-array escape diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrAggregateGlobalEscapes(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		src      string
		wantText string
	}{
		{
			name: "aggregate",
			file: "borrowed_ptr_aggregate_global_escape.tetra",
			src: `
struct PtrBox:
    raw: ptr

var leaked: ptr = 0

func leak(box: borrow PtrBox) -> Int:
    leaked = box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "nested aggregate",
			file: "borrowed_ptr_nested_aggregate_global_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'outer' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "enum payload",
			file: "borrowed_ptr_enum_payload_global_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'msg' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "optional payload if-let",
			file: "borrowed_ptr_optional_payload_global_escape.tetra",
			src: `
var leaked: ptr = 0

func leak(maybe: borrow ptr?) -> Int:
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'maybe' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "optional payload match",
			file: "borrowed_ptr_optional_payload_match_global_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'maybe' cannot escape via global assignment to 'leaked'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), tt.file)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected borrowed ptr aggregate global escape diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForCallableGlobalStorageEscapes(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		src      string
		wantText string
	}{
		{
			name: "captured callable global storage",
			file: "captured_callable_global_storage.tetra",
			src: `
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = Holder(cb: identity(captured))
    cb = holder.cb
    return 0
`,
			wantText: "captured function value cannot be stored in global function-typed value 'cb'",
		},
		{
			name: "function-typed parameter global storage",
			file: "function_typed_parameter_global_storage.tetra",
			src: `
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = f
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
			wantText: "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), tt.file)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected callable global storage escape diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForImportedMutableFunctionTypedGlobalBoundary(t *testing.T) {
	tests := []struct {
		name     string
		app      string
		wantText string
	}{
		{
			name: "direct call",
			app: `module app.main
import lib.math as math

func main() -> Int:
    let _: Int = math.select_add2()
    return math.cb(40)
`,
			wantText: "imported mutable function-typed global 'math.cb' cannot be called directly across module boundary",
		},
		{
			name: "local initializer",
			app: `module app.main
import lib.math as math

func main() -> Int:
    let local: fn(Int) -> Int = math.cb
    return local(40)
`,
			wantText: "imported mutable function-typed global 'math.cb' cannot be used across module boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/math.t4": `module lib.math

pub var cb: fn(Int) -> Int = add1

pub func add1(x: Int) -> Int:
    return x + 1

pub func add2(x: Int) -> Int:
    return x + 2

pub func select_add2() -> Int:
    cb = add2
    return 0
`,
				"app/main.t4": tt.app,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected imported mutable function-typed global boundary diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}

func TestSafetyPrivacyConsentDiagnosticsUsePrivacyCode(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "consent token wrong type",
			src: `
func seal(token: Int) -> secret.i32
uses privacy
privacy
consent(token):
    return 0
`,
			wantText: "semantic clause 'consent' parameter 'token' must have type consent.token",
		},
		{
			name: "consent unknown parameter",
			src: `
func seal(token: consent.token) -> secret.i32
uses privacy
privacy
consent(missing):
    return core.secret_seal_i32(1, token)
`,
			wantText: "semantic clause 'consent' references unknown parameter 'missing'",
		},
		{
			name: "consent malformed argument path",
			src: `
func seal(token: consent.token) -> secret.i32
uses privacy
privacy
consent(token.value):
    return core.secret_seal_i32(1, token)
`,
			wantText: "semantic clause 'consent' expects an identifier argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety privacy diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyPrivacy || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyPrivacy)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}

func TestSafetyEffectPolicyConflictDiagnosticsUseEffectCode(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "noalloc conflict with uses alloc",
			src: `
func main() -> Int
uses alloc
noalloc:
    return 0
`,
			wantText: "semantic clause 'noalloc' conflicts with declared effect 'alloc'",
		},
		{
			name: "noblock conflict with uses block (control)",
			src: `
func main() -> Int
uses control
noblock:
    return 0
`,
			wantText: "semantic clause 'noblock' conflicts with declared effect 'control'",
		},
		{
			name: "realtime policy with uses sleep (runtime) maps to effect code",
			src: `
func main() -> Int
uses runtime
realtime
noalloc
noblock:
    let _: Int = core.sleep_ms(1)
    return 0
`,
			wantText: "semantic clause 'noblock' conflicts with declared effect 'runtime'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety effect diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyEffect || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyEffect)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}
