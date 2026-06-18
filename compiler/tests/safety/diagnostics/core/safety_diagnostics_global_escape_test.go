package compiler_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/testkit"
)

func TestSafetyDiagnosticCodesForBorrowedPtrWholeAggregateGlobalAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
struct PtrBox:
    raw: ptr

var leaked: PtrBox

func leak(box: borrow PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile(
			[]byte(src),
			"borrowed_ptr_whole_aggregate_global_escape.tetra",
		)
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrWholeAggregateGlobalAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
		files := map[string]string{
			"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr
`,
			"app/main.t4": `module app.main
import lib.model as model

var leaked: model.PtrBox

func leak(box: borrow model.PtrBox) -> Int:
    leaked = box
    return 0

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
		assertBorrowedPtrWholeAggregateGlobalAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedPtrWholeAggregateGlobalAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr whole-aggregate global assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf(
			"diagnostic identity = %#v, want code %s",
			diag,
			compiler.DiagnosticCodeSafetyLifetime,
		)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(
		diag.Message,
		"borrowed local 'box' cannot escape via global assignment to 'leaked'",
	) {
		t.Fatalf(
			"message = %q, want borrowed ptr whole-aggregate global assignment escape",
			diag.Message,
		)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrEnumWholeValueGlobalAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
enum PtrMsg:
    case raw(ptr)

var leaked: PtrMsg

func leak(msg: borrow PtrMsg) -> Int:
    leaked = msg
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile(
			[]byte(src),
			"borrowed_ptr_enum_whole_value_global_escape.tetra",
		)
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrEnumWholeValueGlobalAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
		files := map[string]string{
			"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
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
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrEnumWholeValueGlobalAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedPtrEnumWholeValueGlobalAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr enum whole-value global assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf(
			"diagnostic identity = %#v, want code %s",
			diag,
			compiler.DiagnosticCodeSafetyLifetime,
		)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(
		diag.Message,
		"borrowed local 'msg' cannot escape via global assignment to 'leaked'",
	) {
		t.Fatalf(
			"message = %q, want borrowed ptr enum whole-value global assignment escape",
			diag.Message,
		)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrGlobalFieldTargetAssignmentEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
struct PtrBox:
    raw: ptr

var leaked: PtrBox

func leak(box: borrow PtrBox) -> Int:
    leaked.raw = box.raw
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile(
			[]byte(src),
			"borrowed_ptr_global_field_target_escape.tetra",
		)
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrGlobalFieldTargetAssignmentDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
		files := map[string]string{
			"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr
`,
			"app/main.t4": `module app.main
import lib.model as model

var leaked: model.PtrBox

func leak(box: borrow model.PtrBox) -> Int:
    leaked.raw = box.raw
    return 0

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
		assertBorrowedPtrGlobalFieldTargetAssignmentDiagnostic(t, err)
	})
}

func assertBorrowedPtrGlobalFieldTargetAssignmentDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr global field target assignment diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf(
			"diagnostic identity = %#v, want code %s",
			diag,
			compiler.DiagnosticCodeSafetyLifetime,
		)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(
		diag.Message,
		"borrowed local 'box' cannot escape via global assignment to 'leaked'",
	) {
		t.Fatalf(
			"message = %q, want borrowed ptr global field target assignment escape",
			diag.Message,
		)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrAggregateAndNestedGlobalFieldEscapes(t *testing.T) {
	t.Run("same module aggregate", func(t *testing.T) {
		src := `
struct PtrBox:
    raw: ptr

var leaked: ptr = 0

func leak(box: borrow PtrBox) -> Int:
    leaked = box.raw
    return 0

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile(
			[]byte(src),
			"borrowed_ptr_aggregate_global_field_escape.tetra",
		)
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrAggregateGlobalFieldDiagnostic(t, err, "box")
	})

	t.Run("same module nested aggregate", func(t *testing.T) {
		src := `
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
`
		file, err := compiler.ParseFile(
			[]byte(src),
			"borrowed_ptr_nested_aggregate_global_field_escape.tetra",
		)
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrAggregateGlobalFieldDiagnostic(t, err, "outer")
	})

	t.Run("cross module aggregate", func(t *testing.T) {
		files := map[string]string{
			"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr
`,
			"app/main.t4": `module app.main
import lib.model as model

var leaked: ptr = 0

func leak(box: borrow model.PtrBox) -> Int:
    leaked = box.raw
    return 0

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
		assertBorrowedPtrAggregateGlobalFieldDiagnostic(t, err, "box")
	})

	t.Run("cross module nested aggregate", func(t *testing.T) {
		files := map[string]string{
			"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox
`,
			"app/main.t4": `module app.main
import lib.model as model

var leaked: ptr = 0

func leak(outer: borrow model.OuterBox) -> Int:
    leaked = outer.box.raw
    return 0

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
		assertBorrowedPtrAggregateGlobalFieldDiagnostic(t, err, "outer")
	})
}

func assertBorrowedPtrAggregateGlobalFieldDiagnostic(t *testing.T, err error, borrowedName string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr aggregate global field diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf(
			"diagnostic identity = %#v, want code %s",
			diag,
			compiler.DiagnosticCodeSafetyLifetime,
		)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	wantText := "borrowed local '" + borrowedName + "' cannot escape via global assignment to 'leaked'"
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf(
			"message = %q, want borrowed ptr aggregate global field escape %q",
			diag.Message,
			wantText,
		)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedFixedArrayEscapes(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		src      string
		wantText string
	}{
		{
			name: "alias return",
			file: "borrowed_fixed_array_alias_return_escape.tetra",
			src: `
func leak(x: borrow [2]Int) -> [2]Int:
    let y: [2]Int = x
    return y

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "direct global assignment",
			file: "borrowed_fixed_array_global_escape.tetra",
			src: `
struct ArrayBox:
    items: [2]Int

var leaked: ArrayBox

func leak(x: borrow [2]Int) -> Int:
    leaked.items = x
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "optional global assignment",
			file: "borrowed_fixed_array_optional_global_escape.tetra",
			src: `
var leaked: [2]Int? = none

func leak(x: borrow [2]Int) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "inout assignment",
			file: "borrowed_fixed_array_inout_escape.tetra",
			src: `
func leak(x: borrow [2]Int, out: inout [2]Int) -> Int:
    out = x
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), tt.file)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected borrowed fixed-array escape diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf(
					"diagnostic identity = %#v, want code %s",
					diag,
					compiler.DiagnosticCodeSafetyLifetime,
				)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrAggregateGlobalEscapes(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		src      string
		wantText string
	}{
		{
			name: "aggregate",
			file: "borrowed_ptr_aggregate_global_escape.tetra",
			src: `
struct PtrBox:
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
			name: "nested aggregate",
			file: "borrowed_ptr_nested_aggregate_global_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'outer' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "enum payload",
			file: "borrowed_ptr_enum_payload_global_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'msg' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "optional payload if-let",
			file: "borrowed_ptr_optional_payload_global_escape.tetra",
			src: `
var leaked: ptr = 0

func leak(maybe: borrow ptr?) -> Int:
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'maybe' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "optional payload match",
			file: "borrowed_ptr_optional_payload_match_global_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'maybe' cannot escape via global assignment to 'leaked'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), tt.file)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected borrowed ptr aggregate global escape diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf(
					"diagnostic identity = %#v, want code %s",
					diag,
					compiler.DiagnosticCodeSafetyLifetime,
				)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForCallableGlobalStorageEscapes(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		src      string
		wantText string
	}{
		{
			name: "captured callable global storage",
			file: "captured_callable_global_storage.tetra",
			src: `
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = Holder(cb: identity(captured))
    cb = holder.cb
    return 0
`,
			wantText: "captured function value cannot be stored in global function-typed value 'cb'",
		},
		{
			name: "function-typed parameter global storage",
			file: "function_typed_parameter_global_storage.tetra",
			src: `
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = f
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
			wantText: "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := compiler.ParseFile([]byte(tt.src), tt.file)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected callable global storage escape diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf(
					"diagnostic identity = %#v, want code %s",
					diag,
					compiler.DiagnosticCodeSafetyLifetime,
				)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForImportedMutableFunctionTypedGlobalBoundary(t *testing.T) {
	tests := []struct {
		name     string
		app      string
		wantText string
	}{
		{
			name: "direct call",
			app: `module app.main
import lib.math as math

func main() -> Int:
    let _: Int = math.select_add2()
    return math.cb(40)
`,
			wantText: ("imported mutable function-typed global 'math.cb' cannot be " +
				"called directly across module boundary"),
		},
		{
			name: "local initializer",
			app: `module app.main
import lib.math as math

func main() -> Int:
    let local: fn(Int) -> Int = math.cb
    return local(40)
`,
			wantText: ("imported mutable function-typed global 'math.cb' cannot be used " +
				"across module boundary"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/math.t4": `module lib.math

pub var cb: fn(Int) -> Int = add1

pub func add1(x: Int) -> Int:
    return x + 1

pub func add2(x: Int) -> Int:
    return x + 2

pub func select_add2() -> Int:
    cb = add2
    return 0
`,
				"app/main.t4": tt.app,
			}
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected imported mutable function-typed global boundary diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf(
					"diagnostic identity = %#v, want code %s",
					diag,
					compiler.DiagnosticCodeSafetyLifetime,
				)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}

func TestSafetyPrivacyConsentDiagnosticsUsePrivacyCode(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "consent token wrong type",
			src: `
func seal(token: Int) -> secret.i32
uses privacy
privacy
consent(token):
    return 0
`,
			wantText: "semantic clause 'consent' parameter 'token' must have type consent.token",
		},
		{
			name: "consent unknown parameter",
			src: `
func seal(token: consent.token) -> secret.i32
uses privacy
privacy
consent(missing):
    return core.secret_seal_i32(1, token)
`,
			wantText: "semantic clause 'consent' references unknown parameter 'missing'",
		},
		{
			name: "consent malformed argument path",
			src: `
func seal(token: consent.token) -> secret.i32
uses privacy
privacy
consent(token.value):
    return core.secret_seal_i32(1, token)
`,
			wantText: "semantic clause 'consent' expects an identifier argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety privacy diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyPrivacy || diag.Severity != "error" {
				t.Fatalf(
					"diagnostic identity = %#v, want code %s",
					diag,
					compiler.DiagnosticCodeSafetyPrivacy,
				)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}

func TestSafetyEffectPolicyConflictDiagnosticsUseEffectCode(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "noalloc conflict with uses alloc",
			src: `
func main() -> Int
uses alloc
noalloc:
    return 0
`,
			wantText: "semantic clause 'noalloc' conflicts with declared effect 'alloc'",
		},
		{
			name: "noblock conflict with uses block (control)",
			src: `
func main() -> Int
uses control
noblock:
    return 0
`,
			wantText: "semantic clause 'noblock' conflicts with declared effect 'control'",
		},
		{
			name: "realtime policy with uses sleep (runtime) maps to effect code",
			src: `
func main() -> Int
uses runtime
realtime
noalloc
noblock:
    let _: Int = core.sleep_ms(1)
    return 0
`,
			wantText: "semantic clause 'noblock' conflicts with declared effect 'runtime'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety effect diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyEffect || diag.Severity != "error" {
				t.Fatalf(
					"diagnostic identity = %#v, want code %s",
					diag,
					compiler.DiagnosticCodeSafetyEffect,
				)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
		})
	}
}
