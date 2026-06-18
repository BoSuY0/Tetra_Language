package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
	"tetra_language/compiler/tests/ownership/testhelpers"
)

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
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"lib/leaks.t4",
		"borrowed local 'read' cannot escape via inout assignment to 'out'",
	)
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
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'read' cannot escape via inout assignment to 'out'",
	)
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
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'box' cannot escape via global assignment to 'leaked'",
	)
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
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'box' cannot escape via global assignment to 'leaked'",
	)
}

func TestOwnershipRejectsCrossModuleBorrowedPtrAggregateParamGlobalFieldTargetAssignment(
	t *testing.T,
) {
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
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'box' cannot escape via global assignment to 'leaked'",
	)
}

func TestOwnershipRejectsCrossModuleBorrowedPtrNestedAggregateParamGlobalFieldAssignment(
	t *testing.T,
) {
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
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'outer' cannot escape via global assignment to 'leaked'",
	)
}

func TestOwnershipRejectsBorrowEscapeViaEnumReturn(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum BufMsg:
    case send([]u8)

func leak(x: borrow []u8) -> BufMsg:
    return BufMsg.send(x)

func main() -> Int:
    return 0
	`, ("aggregate 'BufMsg' contains borrowed slice field " +
		"'BufMsg.send[1]' that cannot escape through owned return"))
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
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"lib/leaks.t4",
		("aggregate 'BufMsg' contains borrowed slice field " +
			"'BufMsg.send[1]' that cannot escape through owned return"),
	)
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
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"lib/leaks.t4",
		"borrowed local 'x' cannot escape via return",
	)
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
			testhelpers.RequireCheckWorldFilesErrorContains(t, files, "lib/leaks.t4", tt.wantText)
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
			appCall: "return sinker.sink(sinker.BufMsg.send(x))",
			wantText: ("borrowed value derived from 'x' cannot be passed to non-borrow " +
				"parameter 1 of 'lib.sink.sink'"),
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
			testhelpers.RequireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.wantText)
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
	testhelpers.RequireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
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
