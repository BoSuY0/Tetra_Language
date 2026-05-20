package main

import (
	"os"
	"path/filepath"
	"testing"

	"tetra_language/compiler"
)

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrEnumPayloadCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(raw: ptr) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'msg' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "consume",
			sinkSrc: `func sink(raw: consume ptr) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'msg' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(raw: inout ptr) -> Int:
    raw = 0
    return 0
`,
			wantText: "borrowed value derived from 'msg' cannot be passed as inout to 'sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_ptr_enum_payload_"+tt.name+"_call.tetra")
			src := `enum PtrMsg:
    case raw(ptr)
    case empty

` + tt.sinkSrc + `
func leak(msg: borrow PtrMsg) -> Int:
    match msg:
    case PtrMsg.raw(raw):
        return sink(raw)
    case PtrMsg.empty:
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

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrEnumPayloadCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "owned",
			sinkSrc: `func sink(raw: ptr) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'msg' cannot be passed to non-borrow parameter 1 of 'app.main.sink'",
		},
		{
			name: "consume",
			sinkSrc: `func sink(raw: consume ptr) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'msg' cannot be consumed by 'app.main.sink'",
		},
		{
			name: "inout",
			sinkSrc: `func sink(raw: inout ptr) -> Int:
    raw = 0
    return 0
`,
			wantText: "borrowed value derived from 'msg' cannot be passed as inout to 'app.main.sink'",
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

pub enum PtrMsg:
    case raw(ptr)
    case empty
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

`+tt.sinkSrc+`
func leak(msg: borrow model.PtrMsg) -> Int:
    match msg:
    case model.PtrMsg.raw(raw):
        return sink(raw)
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalPayloadOwnedCallEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_ownership_ptr_optional_payload_owned_call.tetra")
	src := `func sink(raw: ptr) -> Int:
    return 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'sink'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrOptionalPayloadOwnedCallEscapeCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/sink.t4", `module lib.sink

pub func sink(raw: ptr) -> Int:
    return 0
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sinker.sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'lib.sink.sink'")
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalPayloadConsumeInoutCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "consume",
			src: `func sink(raw: consume ptr) -> Int:
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
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			src: `func sink(raw: inout ptr) -> Int:
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
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'sink'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_ptr_optional_payload_"+tt.name+"_call.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowPtrOptionalPayloadConsumeInoutCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		wantText string
	}{
		{
			name: "consume",
			sinkSrc: `module lib.sink

pub func sink(raw: consume ptr) -> Int:
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'lib.sink.sink'",
		},
		{
			name: "inout",
			sinkSrc: `module lib.sink

pub func sink(raw: inout ptr) -> Int:
    raw = 0
    return 0
`,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'lib.sink.sink'",
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
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", tt.sinkSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        return sinker.sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowSliceOptionalPayloadBindingEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantCode string
		wantText string
	}{
		{
			name: "owned",
			src: `func sink(raw: []u8) -> Int:
    return 0

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        return sink(raw)
    case none:
        return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name: "consume",
			src: `func sink(raw: consume []u8) -> Int:
    return 0

func leak(maybe: borrow []u8?) -> Int:
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
			name: "inout-call",
			src: `func sink(raw: inout []u8) -> Int:
    raw = raw
    return 0

func leak(maybe: borrow []u8?) -> Int:
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
			name: "inout-assignment",
			src: `func leak(maybe: borrow []u8?, out: inout []u8) -> Int:
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
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_ownership_slice_optional_payload_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONDiagnosticForPath(t, srcPath, srcPath, tt.wantCode, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleOwnershipBorrowSliceOptionalPayloadBindingEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		sinkSrc  string
		leakSrc  string
		wantCode string
		wantText string
		wantFile string
	}{
		{
			name: "owned",
			sinkSrc: `module lib.sink

pub func sink(raw: []u8) -> Int:
    return 0
`,
			leakSrc: `module app.main
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'lib.sink.sink'",
			wantFile: "src/app/main.t4",
		},
		{
			name: "consume",
			sinkSrc: `module lib.sink

pub func sink(raw: consume []u8) -> Int:
    return 0
`,
			leakSrc: `module app.main
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'lib.sink.sink'",
			wantFile: "src/app/main.t4",
		},
		{
			name: "inout-call",
			sinkSrc: `module lib.sink

pub func sink(raw: inout []u8) -> Int:
    raw = raw
    return 0
`,
			leakSrc: `module app.main
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
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'lib.sink.sink'",
			wantFile: "src/app/main.t4",
		},
		{
			name:    "inout-assignment",
			sinkSrc: "",
			leakSrc: `module lib.leaks

pub func leak(maybe: borrow []u8?, out: inout []u8) -> Int:
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
			wantFile: "src/lib/leaks.t4",
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
			if tt.sinkSrc != "" {
				writeCLIProjectFile(t, dir, "src/lib/sink.t4", tt.sinkSrc)
				writeCLIProjectFile(t, dir, "src/app/main.t4", tt.leakSrc)
			} else {
				writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.leakSrc)
				writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
			}
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			wantPath := filepath.Join(dir, filepath.FromSlash(tt.wantFile))
			assertCLIJSONDiagnosticForPath(t, srcPath, wantPath, tt.wantCode, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipBorrowPtrOptionalAssignmentConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_ownership_ptr_optional_assignment_consume.tetra")
	src := `func sink(value: consume ptr?) -> Int:
    return 0

func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "borrowed value derived from 'x' cannot be consumed by 'sink'")
}
