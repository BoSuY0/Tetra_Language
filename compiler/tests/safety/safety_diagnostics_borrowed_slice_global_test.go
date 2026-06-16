package compiler_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/testkit"
)

func TestSafetyDiagnosticCodesForBorrowedSliceOptionalPayloadInoutGlobalEscapes(t *testing.T) {
	type caseDef struct {
		name     string
		body     string
		extra    string
		params   string
		wantText string
	}

	cases := []caseDef{
		{
			name: "if-let inout",
			body: `
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0
`,
			params:   "maybe: borrow []u8?, out: inout []u8",
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
		},
		{
			name: "match inout",
			body: `
    match maybe:
    case some(raw):
        out = raw
        return 0
    case none:
        return 0
`,
			params:   "maybe: borrow []u8?, out: inout []u8",
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
		},
		{
			name: "if-let global",
			extra: `
var leaked: []u8? = none
`,
			body: `
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
			`,
			params:   "maybe: borrow []u8?",
			wantText: "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot be stored in global",
		},
		{
			name: "match global",
			extra: `
var leaked: []u8? = none
`,
			body: `
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
			params:   "maybe: borrow []u8?",
			wantText: "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot be stored in global",
		},
	}

	for _, tt := range cases {
		t.Run("same module "+tt.name, func(t *testing.T) {
			src := tt.extra + `
func leak(` + tt.params + `) -> Int:` + tt.body + `
func main() -> Int:
    return 0
`
			file, err := compiler.ParseFile([]byte(src), "borrowed_slice_optional_payload_"+strings.ReplaceAll(tt.name, " ", "_")+".tetra")
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedSliceOptionalPayloadDiagnostic(t, err, tt.wantText)
		})
	}

	for _, tt := range cases {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": `module lib.leaks
` + tt.extra + `
pub func leak(` + tt.params + `) -> Int:` + tt.body,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedSliceOptionalPayloadDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertBorrowedSliceOptionalPayloadDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed slice optional-payload diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed slice optional-payload escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedNestedSliceStructEscapes(t *testing.T) {
	sameModuleTests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal return",
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
			name: "alias return",
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
			name: "inout assignment",
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
		{
			name: "global assignment",
			src: `
struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox

var leaked: OuterBox

func leak(read: borrow []u8) -> Int:
    leaked = OuterBox(box: BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot be stored in global",
		},
	}

	for _, tt := range sameModuleTests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), "borrowed_nested_slice_struct_escape.tetra")
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedNestedSliceStructDiagnostic(t, err, tt.wantText)
		})
	}

	crossModuleTests := []struct {
		name     string
		files    map[string]string
		wantText string
	}{
		{
			name: "literal return",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(x: borrow []u8) -> OuterBox:
    return OuterBox(box: BufBox(buf: x))
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias return",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(x: borrow []u8) -> OuterBox:
    let outer: OuterBox = OuterBox(box: BufBox(buf: x))
    return outer
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout assignment",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(read: borrow []u8, out: inout OuterBox) -> Int:
    out = OuterBox(box: BufBox(buf: read))
    return 0
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
		{
			name: "global assignment",
			files: map[string]string{
				"lib/model.t4": `module lib.model

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox
`,
				"app/main.t4": `module app.main
import lib.model as model

var leaked: model.OuterBox

func leak(read: borrow []u8) -> Int:
    leaked = model.OuterBox(box: model.BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			},
			wantText: "aggregate 'lib.model.BufBox' contains borrowed slice field 'buf' that cannot be stored in global",
		},
	}

	for _, tt := range crossModuleTests {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, tt.files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedNestedSliceStructDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertBorrowedNestedSliceStructDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed nested slice struct diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed nested slice struct escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedNestedSliceEnumPayloadEscapes(t *testing.T) {
	sameModuleTests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal return",
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
			name: "alias return",
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
			name: "inout assignment",
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
		{
			name: "global assignment",
			src: `
struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty

var leaked: OuterMsg

func leak(read: borrow []u8) -> Int:
    leaked = OuterMsg.wrap(BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot be stored in global",
		},
	}

	for _, tt := range sameModuleTests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), "borrowed_nested_slice_enum_payload_escape.tetra")
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedNestedSliceEnumPayloadDiagnostic(t, err, tt.wantText)
		})
	}

	crossModuleTests := []struct {
		name     string
		files    map[string]string
		wantText string
	}{
		{
			name: "literal return",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(x: borrow []u8) -> OuterMsg:
    return OuterMsg.wrap(BufBox(buf: x))
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias return",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(x: borrow []u8) -> OuterMsg:
    let msg: OuterMsg = OuterMsg.wrap(BufBox(buf: x))
    return msg
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout assignment",
			files: map[string]string{
				"lib/leaks.t4": `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(read: borrow []u8, out: inout OuterMsg) -> Int:
    out = OuterMsg.wrap(BufBox(buf: read))
    return 0
`,
				"app/main.t4": `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`,
			},
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
		{
			name: "global assignment",
			files: map[string]string{
				"lib/model.t4": `module lib.model

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty
`,
				"app/main.t4": `module app.main
import lib.model as model

var leaked: model.OuterMsg

func leak(read: borrow []u8) -> Int:
    leaked = model.OuterMsg.wrap(model.BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			},
			wantText: "aggregate 'lib.model.BufBox' contains borrowed slice field 'buf' that cannot be stored in global",
		},
	}

	for _, tt := range crossModuleTests {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, tt.files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedNestedSliceEnumPayloadDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertBorrowedNestedSliceEnumPayloadDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed nested slice enum-payload diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed nested slice enum-payload escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrOptionalGlobalAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
var leaked: ptr? = none

func leak(x: borrow ptr) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_optional_global_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrOptionalGlobalAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrOptionalGlobalAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedPtrOptionalGlobalAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr optional global assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'x' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed ptr optional global assignment escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrAggregateOptionalGlobalAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
struct PtrBox:
    raw: ptr

var leaked: PtrBox? = none

func leak(box: borrow PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_aggregate_optional_global_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrAggregateOptionalGlobalAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrAggregateOptionalGlobalAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedPtrAggregateOptionalGlobalAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr aggregate optional global assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'box' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed ptr aggregate optional global assignment escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedSliceGlobalAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
var leaked: []u8

func leak(x: borrow []u8) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_slice_global_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedSliceGlobalAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedSliceGlobalAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedSliceGlobalAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed slice global assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'x' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed slice global assignment escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}
