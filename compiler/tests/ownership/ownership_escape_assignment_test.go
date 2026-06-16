package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

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
