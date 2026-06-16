package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

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
