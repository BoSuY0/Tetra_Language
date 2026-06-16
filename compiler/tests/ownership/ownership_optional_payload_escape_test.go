package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

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
