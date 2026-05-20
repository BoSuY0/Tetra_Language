package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumAliasReturnEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_enum_alias_return.tetra")
	src := `enum PtrMsg:
    case raw(ptr)

func leak(x: borrow ptr) -> PtrMsg:
    let msg: PtrMsg = PtrMsg.raw(x)
    return msg

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumAliasReturnEscapeCode(t *testing.T) {
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
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

func leak(x: borrow ptr) -> model.PtrMsg:
    let msg: model.PtrMsg = model.PtrMsg.raw(x)
    return msg

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrAggregateReturnEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "whole",
			src: `struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    return box

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "field",
			src: `struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> ptr:
    return box.raw

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "alias",
			src: `struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    let alias: PtrBox = box
    return alias

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "nested-field",
			src: `struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

func leak(outer: borrow OuterBox) -> ptr:
    return outer.box.raw

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'outer' cannot escape via return",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_aggregate_"+tt.name+"_return.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrAggregateReturnEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "whole",
			src: `module app.main
import lib.model as model

func leak(box: borrow model.PtrBox) -> model.PtrBox:
    return box

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "field",
			src: `module app.main
import lib.model as model

func leak(box: borrow model.PtrBox) -> ptr:
    return box.raw

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "alias",
			src: `module app.main
import lib.model as model

func leak(box: borrow model.PtrBox) -> model.PtrBox:
    let alias: model.PtrBox = box
    return alias

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via return",
		},
		{
			name: "nested-field",
			src: `module app.main
import lib.model as model

func leak(outer: borrow model.OuterBox) -> ptr:
    return outer.box.raw

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'outer' cannot escape via return",
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
			writeCLIProjectFile(t, dir, "src/app/main.t4", tt.src)

			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalAssignmentGlobalEscapeCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `var leaked: ptr = 0

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
`,
		},
		{
			name: "match",
			src: `var leaked: ptr = 0

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
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_optional_assignment_"+tt.name+"_global.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalAssignmentGlobalEscapeCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `module lib.leaks

var leaked: ptr = 0

pub func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
`,
		},
		{
			name: "match",
			src: `module lib.leaks

var leaked: ptr = 0

pub func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
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
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.src)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)

			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrAggregateGlobalEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "aggregate",
			src: `struct PtrBox:
    raw: ptr

var leaked: ptr = 0

func leak(box: borrow PtrBox) -> Int:
    leaked = box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "aggregate whole global",
			src: `struct PtrBox:
    raw: ptr

var leaked: PtrBox

func leak(box: borrow PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "aggregate global field target",
			src: `struct PtrBox:
    raw: ptr

var leaked: PtrBox

func leak(box: borrow PtrBox) -> Int:
    leaked.raw = box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "nested aggregate",
			src: `struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

var leaked: ptr = 0

func leak(outer: borrow OuterBox) -> Int:
    leaked = outer.box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'outer' cannot escape via global assignment to 'leaked'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_aggregate_global.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrAggregateGlobalEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		modelSrc string
		mainSrc  string
		wantText string
	}{
		{
			name: "aggregate",
			modelSrc: `module lib.model

pub struct PtrBox:
    raw: ptr
`,
			mainSrc: `module app.main
import lib.model as model

var leaked: ptr = 0

func leak(box: borrow model.PtrBox) -> Int:
    leaked = box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "aggregate whole global",
			modelSrc: `module lib.model

pub struct PtrBox:
    raw: ptr
`,
			mainSrc: `module app.main
import lib.model as model

var leaked: model.PtrBox

func leak(box: borrow model.PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "aggregate global field target",
			modelSrc: `module lib.model

pub struct PtrBox:
    raw: ptr
`,
			mainSrc: `module app.main
import lib.model as model

var leaked: model.PtrBox

func leak(box: borrow model.PtrBox) -> Int:
    leaked.raw = box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'box' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "nested aggregate",
			modelSrc: `module lib.model

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox
`,
			mainSrc: `module app.main
import lib.model as model

var leaked: ptr = 0

func leak(outer: borrow model.OuterBox) -> Int:
    leaked = outer.box.raw
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'outer' cannot escape via global assignment to 'leaked'",
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
			writeCLIProjectFile(t, dir, "src/lib/model.t4", tt.modelSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", tt.mainSrc)

			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadReturnEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_enum_payload_return.tetra")
	src := `enum PtrMsg:
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
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'msg' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadReturnEscapeCode(t *testing.T) {
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

func leak(msg: borrow model.PtrMsg) -> ptr:
    match msg:
    case model.PtrMsg.raw(raw):
        return raw
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'msg' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadGlobalEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_enum_payload_global.tetra")
	src := `enum PtrMsg:
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
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'msg' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumGlobalEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_enum_global.tetra")
	src := `enum PtrMsg:
    case raw(ptr)
    case empty

var leaked: PtrMsg

func leak(msg: borrow PtrMsg) -> Int:
    leaked = msg
    return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'msg' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadGlobalEscapeCode(t *testing.T) {
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
`)
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'msg' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumGlobalEscapeCode(t *testing.T) {
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

var leaked: model.PtrMsg

func leak(msg: borrow model.PtrMsg) -> Int:
    leaked = msg
    return 0

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'msg' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrEnumPayloadInoutEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_enum_payload_inout.tetra")
	src := `enum PtrMsg:
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
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'msg' cannot escape via inout assignment to 'out'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrEnumPayloadInoutEscapeCode(t *testing.T) {
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

func leak(msg: borrow model.PtrMsg, out: inout ptr) -> Int:
    match msg:
    case model.PtrMsg.raw(raw):
        out = raw
        return 0
    case model.PtrMsg.empty:
        return 0

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'msg' cannot escape via inout assignment to 'out'")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadInoutEscapeCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`,
		},
		{
			name: "match",
			src: `func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_optional_payload_inout.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'maybe' cannot escape via inout assignment to 'out'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadInoutEscapeCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `module lib.leaks

pub func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0
`,
		},
		{
			name: "match",
			src: `module lib.leaks

pub func leak(maybe: borrow ptr?, out: inout ptr) -> Int:
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
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.src)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, "borrowed local 'maybe' cannot escape via inout assignment to 'out'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadGlobalEscapeCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `var leaked: ptr = 0

func leak(maybe: borrow ptr?) -> Int:
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`,
		},
		{
			name: "match",
			src: `var leaked: ptr = 0

func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_optional_payload_global.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'maybe' cannot escape via global assignment to 'leaked'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadGlobalEscapeCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `module lib.leaks

var leaked: ptr = 0

pub func leak(maybe: borrow ptr?) -> Int:
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
`,
		},
		{
			name: "match",
			src: `module lib.leaks

var leaked: ptr = 0

pub func leak(maybe: borrow ptr?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
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
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.src)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, "borrowed local 'maybe' cannot escape via global assignment to 'leaked'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowPtrOptionalPayloadReturnEscapeCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `func leak(maybe: borrow ptr?) -> ptr:
    if let raw = maybe:
        return raw
    else:
        return 0

func main() -> Int:
    return 0
`,
		},
		{
			name: "match",
			src: `func leak(maybe: borrow ptr?) -> ptr:
    match maybe:
    case some(raw):
        return raw
    case none:
        return 0

func main() -> Int:
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_ptr_optional_payload_return.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'maybe' cannot escape via return")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowPtrOptionalPayloadReturnEscapeCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if-let",
			src: `module lib.leaks

pub func leak(maybe: borrow ptr?) -> ptr:
    if let raw = maybe:
        return raw
    else:
        return 0
`,
		},
		{
			name: "match",
			src: `module lib.leaks

pub func leak(maybe: borrow ptr?) -> ptr:
    match maybe:
    case some(raw):
        return raw
    case none:
        return 0
`,
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
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.src)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, "borrowed local 'maybe' cannot escape via return")
		})
	}
}
