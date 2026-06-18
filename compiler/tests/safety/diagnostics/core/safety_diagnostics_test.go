package compiler_test

import (
	"encoding/json"
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
			wantText: ("aggregate 'BufBox' contains borrowed slice field 'buf' that " +
				"cannot escape through owned return"),
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
			wantText: ("aggregate 'BufMsg' contains borrowed slice field " +
				"'BufMsg.send[1]' that cannot escape through owned return"),
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
			wantText: ("borrowed value derived from 'x' cannot be passed to non-borrow " +
				"parameter 1 of callback 'cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be passed to non-borrow " +
				"parameter 1 of callback 'cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be passed to non-borrow " +
				"parameter 1 of function-typed struct field call 'h.cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be consumed by function-" +
				"typed struct field call 'h.cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be passed as inout to " +
				"function-typed struct field call 'h.cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be passed to non-borrow " +
				"parameter 1 of function-typed struct field call 'h.cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be consumed by function-" +
				"typed struct field call 'h.cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be passed as inout to " +
				"function-typed struct field call 'h.cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be passed to non-borrow " +
				"parameter 1 of function-typed enum payload call 'cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be consumed by function-" +
				"typed enum payload call 'cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be passed as inout to " +
				"function-typed enum payload call 'cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be passed to non-borrow " +
				"parameter 1 of function-typed enum payload call 'cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be consumed by function-" +
				"typed enum payload call 'cb'"),
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
			wantText: ("borrowed value derived from 'x' cannot be passed as inout to " +
				"function-typed enum payload call 'cb'"),
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
			wantText: ("borrowed value derived from 'maybe' cannot be passed to non-" +
				"borrow parameter 1 of 'sink'"),
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
