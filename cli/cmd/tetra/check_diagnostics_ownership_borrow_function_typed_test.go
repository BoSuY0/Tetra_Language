package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateFunctionTypedParameterCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		typeSrc  string
		callback string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callback: "cb: fn(BufBox) -> Int",
			call:     "return cb(BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of callback 'cb'",
		},
		{
			name: "struct-consume",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callback: "cb: fn(consume BufBox) -> Int",
			call:     "let box: BufBox = BufBox(buf: x)\n    return cb(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by callback 'cb'",
		},
		{
			name: "struct-inout",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callback: "cb: fn(inout BufBox) -> Int",
			call:     "var box: BufBox = BufBox(buf: x)\n    return cb(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to callback 'cb'",
		},
		{
			name: "enum-owned",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callback: "cb: fn(BufMsg) -> Int",
			call:     "return cb(BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of callback 'cb'",
		},
		{
			name: "enum-consume",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callback: "cb: fn(consume BufMsg) -> Int",
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by callback 'cb'",
		},
		{
			name: "enum-inout",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callback: "cb: fn(inout BufMsg) -> Int",
			call:     "var msg: BufMsg = BufMsg.send(x)\n    return cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to callback 'cb'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_slice_aggregate_function_typed_"+tt.name+".tetra")
			src := tt.typeSrc + `
func caller(` + tt.callback + `, x: borrow []u8) -> Int:
    ` + tt.call + `

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateFunctionTypedStructFieldCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		typeSrc  string
		field    string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			field:    "cb: fn(BufBox) -> Int",
			call:     "return h.cb(BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'",
		},
		{
			name: "struct-consume",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			field:    "cb: fn(consume BufBox) -> Int",
			call:     "let box: BufBox = BufBox(buf: x)\n    return h.cb(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed struct field call 'h.cb'",
		},
		{
			name: "struct-inout",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			field:    "cb: fn(inout BufBox) -> Int",
			call:     "var box: BufBox = BufBox(buf: x)\n    return h.cb(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-owned",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			field:    "cb: fn(BufMsg) -> Int",
			call:     "return h.cb(BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-consume",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			field:    "cb: fn(consume BufMsg) -> Int",
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return h.cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-inout",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			field:    "cb: fn(inout BufMsg) -> Int",
			call:     "var msg: BufMsg = BufMsg.send(x)\n    return h.cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed struct field call 'h.cb'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_slice_aggregate_function_typed_field_"+tt.name+".tetra")
			src := tt.typeSrc + `
struct Handler:
    ` + tt.field + `

func caller(h: Handler, x: borrow []u8) -> Int:
    ` + tt.call + `

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceAggregateFunctionTypedStructFieldCallEscapeCodes(t *testing.T) {
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
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/callbacks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.callbacks as callbacks

func caller(h: callbacks.Handler, x: borrow []u8) -> Int:
    `+tt.appCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateFunctionTypedEnumPayloadCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		typeSrc  string
		payload  string
		setup    string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			payload:  "case some(fn(BufBox) -> Int)",
			call:     "return cb(BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'",
		},
		{
			name: "struct-consume",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			payload:  "case some(fn(consume BufBox) -> Int)",
			setup:    "let box: BufBox = BufBox(buf: x)\n    ",
			call:     "return cb(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed enum payload call 'cb'",
		},
		{
			name: "struct-inout",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			payload:  "case some(fn(inout BufBox) -> Int)",
			setup:    "var box: BufBox = BufBox(buf: x)\n    ",
			call:     "return cb(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed enum payload call 'cb'",
		},
		{
			name: "enum-owned",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			payload:  "case some(fn(BufMsg) -> Int)",
			call:     "return cb(BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'",
		},
		{
			name: "enum-consume",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			payload:  "case some(fn(consume BufMsg) -> Int)",
			setup:    "let msg: BufMsg = BufMsg.send(x)\n    ",
			call:     "return cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by function-typed enum payload call 'cb'",
		},
		{
			name: "enum-inout",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			payload:  "case some(fn(inout BufMsg) -> Int)",
			setup:    "var msg: BufMsg = BufMsg.send(x)\n    ",
			call:     "return cb(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to function-typed enum payload call 'cb'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_slice_aggregate_function_typed_enum_payload_"+tt.name+".tetra")
			src := tt.typeSrc + `
enum Choice:
    ` + tt.payload + `
    case empty

func caller(choice: Choice, x: borrow []u8) -> Int:
    ` + tt.setup + `match choice:
    case Choice.some(cb):
        ` + tt.call + `
    case Choice.empty:
        return 0

func main() -> Int:
    return 0
`
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceAggregateFunctionTypedEnumPayloadCallEscapeCodes(t *testing.T) {
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
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/callbacks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.callbacks as callbacks

func caller(choice: callbacks.Choice, x: borrow []u8) -> Int:
    `+tt.setup+`match choice:
    case callbacks.Choice.some(cb):
        `+tt.call+`
    case callbacks.Choice.empty:
        return 0

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowOptionalPtrFunctionTypedCallbackCodes(t *testing.T) {
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
			name: "field-owned",
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
			name: "field-consume",
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
			name: "field-inout",
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
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_optional_ptr_callback_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowOptionalPtrFunctionTypedCallbackCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		appSrc   string
		wantText string
	}{
		{
			name: "field-owned",
			libSrc: `module lib.callbacks

pub struct Handler:
    cb: fn(ptr?) -> Int
`,
			appSrc: `func caller(h: callbacks.Handler, maybe: borrow ptr?) -> Int:
    return h.cb(maybe)
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of function-typed struct field call 'h.cb'",
		},
		{
			name: "field-consume",
			libSrc: `module lib.callbacks

pub struct Handler:
    cb: fn(consume ptr?) -> Int
`,
			appSrc: `func caller(h: callbacks.Handler, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    return h.cb(alias)
`,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by function-typed struct field call 'h.cb'",
		},
		{
			name: "field-inout",
			libSrc: `module lib.callbacks

pub struct Handler:
    cb: fn(inout ptr?) -> Int
`,
			appSrc: `func caller(h: callbacks.Handler, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    return h.cb(alias)
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to function-typed struct field call 'h.cb'",
		},
		{
			name: "enum-payload-owned",
			libSrc: `module lib.callbacks

pub enum Choice:
    case some(fn(ptr?) -> Int)
    case empty
`,
			appSrc: `func caller(choice: callbacks.Choice, maybe: borrow ptr?) -> Int:
    match choice:
    case callbacks.Choice.some(cb):
        return cb(maybe)
    case callbacks.Choice.empty:
        return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of function-typed enum payload call 'cb'",
		},
		{
			name: "enum-payload-consume",
			libSrc: `module lib.callbacks

pub enum Choice:
    case some(fn(consume ptr?) -> Int)
    case empty
`,
			appSrc: `func caller(choice: callbacks.Choice, maybe: borrow ptr?) -> Int:
    let alias: ptr? = maybe
    match choice:
    case callbacks.Choice.some(cb):
        return cb(alias)
    case callbacks.Choice.empty:
        return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by function-typed enum payload call 'cb'",
		},
		{
			name: "enum-payload-inout",
			libSrc: `module lib.callbacks

pub enum Choice:
    case some(fn(inout ptr?) -> Int)
    case empty
`,
			appSrc: `func caller(choice: callbacks.Choice, maybe: borrow ptr?) -> Int:
    var alias: ptr? = maybe
    match choice:
    case callbacks.Choice.some(cb):
        return cb(alias)
    case callbacks.Choice.empty:
        return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to function-typed enum payload call 'cb'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/callbacks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.callbacks as callbacks

`+tt.appSrc+`
func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}
