package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrAggregateCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(value: PtrBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'box' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "consume",
			sinkSrc: `func sink(value: consume PtrBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'box' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(value: inout PtrBox) -> Int:
    value = PtrBox(raw: 0)
    return 0
`,
			wantText: "borrowed value derived from 'box' cannot be passed as inout to 'sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_ptr_aggregate_"+tt.name+"_call.tetra")
			src := `struct PtrBox:
    raw: ptr

` + tt.sinkSrc + `
func leak(box: borrow PtrBox) -> Int:
    return sink(box)

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

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrAggregateCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(value: model.PtrBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'box' cannot be passed to non-borrow parameter 1 of 'app.main.sink'",
		},
		{
			name: "consume",
			sinkSrc: `func sink(value: consume model.PtrBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'box' cannot be consumed by 'app.main.sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(value: inout model.PtrBox) -> Int:
    value = model.PtrBox(raw: 0)
    return 0
`,
			wantText: "borrowed value derived from 'box' cannot be passed as inout to 'app.main.sink'",
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
			writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct PtrBox:
    raw: ptr
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

`+tt.sinkSrc+`
func leak(box: borrow model.PtrBox) -> Int:
    return sink(box)

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowPtrAggregateCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		mainCall string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `pub func sink(value: PtrBox) -> Int:
    return 0
`,
			mainCall: "return sinker.sink(sinker.PtrBox(raw: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'lib.sink.sink'",
		},
		{
			name: "consume",
			sinkSrc: `pub func take(value: consume PtrBox) -> Int:
    return 0
`,
			mainCall: "let box: sinker.PtrBox = sinker.PtrBox(raw: x)\n    return sinker.take(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.sink.take'",
		},
		{
			name: "inout",
			sinkSrc: `pub func mutate(value: inout PtrBox) -> Int:
    value = value
    return 0
`,
			mainCall: "var box: sinker.PtrBox = sinker.PtrBox(raw: x)\n    return sinker.mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.sink.mutate'",
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
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", `module lib.sink

pub struct PtrBox:
    raw: ptr

`+tt.sinkSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    `+tt.mainCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowPtrNestedAggregateCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		mainCall string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `pub func sink(value: OuterBox) -> Int:
    return 0
`,
			mainCall: "return sinker.sink(sinker.OuterBox(box: sinker.PtrBox(raw: x)))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'lib.sink.sink'",
		},
		{
			name: "consume",
			sinkSrc: `pub func take(value: consume OuterBox) -> Int:
    return 0
`,
			mainCall: "let outer: sinker.OuterBox = sinker.OuterBox(box: sinker.PtrBox(raw: x))\n    return sinker.take(outer)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.sink.take'",
		},
		{
			name: "inout",
			sinkSrc: `pub func mutate(value: inout OuterBox) -> Int:
    value = value
    return 0
`,
			mainCall: "var outer: sinker.OuterBox = sinker.OuterBox(box: sinker.PtrBox(raw: x))\n    return sinker.mutate(outer)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.sink.mutate'",
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
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", `module lib.sink

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox

`+tt.sinkSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    `+tt.mainCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrNestedAggregateCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(value: OuterBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'outer' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "consume",
			sinkSrc: `func sink(value: consume OuterBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'outer' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(value: inout OuterBox) -> Int:
    value = OuterBox(box: PtrBox(raw: 0))
    return 0
`,
			wantText: "borrowed value derived from 'outer' cannot be passed as inout to 'sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_ptr_nested_aggregate_"+tt.name+"_call.tetra")
			src := `struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

` + tt.sinkSrc + `
func leak(outer: borrow OuterBox) -> Int:
    return sink(outer)

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

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrNestedAggregateCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(value: model.OuterBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'outer' cannot be passed to non-borrow parameter 1 of 'app.main.sink'",
		},
		{
			name: "consume",
			sinkSrc: `func sink(value: consume model.OuterBox) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'outer' cannot be consumed by 'app.main.sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(value: inout model.OuterBox) -> Int:
    value = model.OuterBox(box: model.PtrBox(raw: 0))
    return 0
`,
			wantText: "borrowed value derived from 'outer' cannot be passed as inout to 'app.main.sink'",
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
			writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

`+tt.sinkSrc+`
func leak(outer: borrow model.OuterBox) -> Int:
    return sink(outer)

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}
