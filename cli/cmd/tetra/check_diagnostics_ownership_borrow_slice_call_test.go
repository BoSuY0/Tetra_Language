package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateCallEscapeCodes(t *testing.T) {
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
			callee: `func sink(value: BufBox) -> Int:
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
			callee: `func sink(value: consume BufBox) -> Int:
    return 0
`,
			call:     "let box: BufBox = BufBox(buf: x)\n    return sink(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "struct-inout",
			typeSrc: `struct BufBox:
    buf: []u8
`,
			callee: `func mutate(value: inout BufBox) -> Int:
    value = value
    return 0
`,
			call:     "var box: BufBox = BufBox(buf: x)\n    return mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'mutate'",
		},
		{
			name: "enum-owned",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func sink(value: BufMsg) -> Int:
    return 0
`,
			call:     "return sink(BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "enum-consume",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func sink(value: consume BufMsg) -> Int:
    return 0
`,
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return sink(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "enum-inout",
			typeSrc: `enum BufMsg:
    case send([]u8)
`,
			callee: `func mutate(value: inout BufMsg) -> Int:
    value = value
    return 0
`,
			call:     "var msg: BufMsg = BufMsg.send(x)\n    return mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'mutate'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_slice_aggregate_call_"+tt.name+".tetra")
			src := tt.typeSrc + "\n" + tt.callee + `
func caller(x: borrow []u8) -> Int:
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

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceAggregateCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		typeSrc  string
		callee   string
		call     string
		wantText string
	}{
		{
			name: "struct-owned",
			typeSrc: `pub struct BufBox:
    buf: []u8
`,
			callee: `pub func sink(value: BufBox) -> Int:
    return 0
`,
			call:     "return sink(BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'lib.leaks.sink'",
		},
		{
			name: "struct-consume",
			typeSrc: `pub struct BufBox:
    buf: []u8
`,
			callee: `pub func sink(value: consume BufBox) -> Int:
    return 0
`,
			call:     "let box: BufBox = BufBox(buf: x)\n    return sink(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.leaks.sink'",
		},
		{
			name: "struct-inout",
			typeSrc: `pub struct BufBox:
    buf: []u8
`,
			callee: `pub func mutate(value: inout BufBox) -> Int:
    value = value
    return 0
`,
			call:     "var box: BufBox = BufBox(buf: x)\n    return mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.leaks.mutate'",
		},
		{
			name: "enum-owned",
			typeSrc: `pub enum BufMsg:
    case send([]u8)
`,
			callee: `pub func sink(value: BufMsg) -> Int:
    return 0
`,
			call:     "return sink(BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'lib.leaks.sink'",
		},
		{
			name: "enum-consume",
			typeSrc: `pub enum BufMsg:
    case send([]u8)
`,
			callee: `pub func sink(value: consume BufMsg) -> Int:
    return 0
`,
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return sink(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.leaks.sink'",
		},
		{
			name: "enum-inout",
			typeSrc: `pub enum BufMsg:
    case send([]u8)
`,
			callee: `pub func mutate(value: inout BufMsg) -> Int:
    value = value
    return 0
`,
			call:     "var msg: BufMsg = BufMsg.send(x)\n    return mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.leaks.mutate'",
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
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", `module lib.leaks

`+tt.typeSrc+`
`+tt.callee+`
pub func caller(x: borrow []u8) -> Int:
    `+tt.call+`
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceAggregateGenericCallEscapeCodes(t *testing.T) {
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
			call:     "let msg: BufMsg = BufMsg.send(x)\n    return sink(msg)",
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
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_slice_aggregate_generic_call_"+tt.name+".tetra")
			src := tt.typeSrc + "\n" + tt.callee + `
func caller(x: borrow []u8) -> Int:
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

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceAggregateGenericCallEscapeCodes(t *testing.T) {
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
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func caller(x: borrow []u8) -> Int:
    `+tt.appCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowOptionalPtrGenericCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		callee   string
		call     string
		wantText string
	}{
		{
			name: "owned",
			callee: `func sink<T>(value: T) -> Int:
    return 0
`,
			call:     "return sink(maybe)",
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "consume",
			callee: `func take<T>(value: consume T) -> Int:
    return 0
`,
			call:     "let alias: ptr? = maybe\n    return take(alias)",
			wantText: "borrowed value derived from 'maybe' cannot be consumed",
		},
		{
			name: "inout",
			callee: `func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			call:     "var alias: ptr? = maybe\n    return mutate(alias)",
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_optional_ptr_generic_call_"+tt.name+".tetra")
			src := tt.callee + `
func caller(maybe: borrow ptr?) -> Int:
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

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowOptionalPtrGenericCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		appCall  string
		wantText string
	}{
		{
			name: "owned",
			libSrc: `module lib.sink

pub func sink<T>(value: T) -> Int:
    return 0
`,
			appCall:  "return sinker.sink(maybe)",
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "consume",
			libSrc: `module lib.sink

pub func take<T>(value: consume T) -> Int:
    return 0
`,
			appCall:  "let alias: ptr? = maybe\n    return sinker.take(alias)",
			wantText: "borrowed value derived from 'maybe' cannot be consumed",
		},
		{
			name: "inout",
			libSrc: `module lib.sink

pub func mutate<T>(value: inout T) -> Int:
    value = value
    return 0
`,
			appCall:  "var alias: ptr? = maybe\n    return sinker.mutate(alias)",
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout",
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
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func caller(maybe: borrow ptr?) -> Int:
    `+tt.appCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}
