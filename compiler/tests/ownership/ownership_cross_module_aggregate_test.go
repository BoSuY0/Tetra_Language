package compiler_test

import (
	"path/filepath"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

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
