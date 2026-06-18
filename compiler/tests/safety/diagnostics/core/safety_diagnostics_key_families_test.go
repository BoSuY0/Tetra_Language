package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/testkit"
)

func TestSafetyDiagnosticCodesForBorrowedAggregateAndResourceKeyFamilies(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantCode string
		wantText string
		parseErr bool
	}{
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
			wantText: ("aggregate '[]u8?' contains borrowed slice field '$elem' that " +
				"cannot escape through owned return"),
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
			wantText: ("borrowed value derived from 'box' cannot be passed to non-" +
				"borrow parameter 1 of 'sink'"),
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
			wantText: ("borrowed value derived from 'outer' cannot be passed to non-" +
				"borrow parameter 1 of 'sink'"),
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
			wantText: ("borrowed value derived from 'msg' cannot be passed to non-" +
				"borrow parameter 1 of 'sink'"),
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
