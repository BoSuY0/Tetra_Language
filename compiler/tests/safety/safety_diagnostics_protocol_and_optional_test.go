package compiler_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/testkit"
)

func TestSafetyDiagnosticCodesForGenericProtocolRequirementOwnershipMismatch(t *testing.T) {
	tests := []struct {
		name     string
		cross    bool
		wantText string
	}{
		{
			name:     "same module",
			wantText: "method 'Box.map' does not match protocol 'Mapper' requirement 'map': parameter 2 ownership differs: expected 'consume', got 'owned'",
		},
		{
			name:     "cross module",
			cross:    true,
			wantText: "method 'lib.model.Box.map' does not match protocol 'lib.model.Mapper' requirement 'map': parameter 2 ownership differs: expected 'consume', got 'owned'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.cross {
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
				tmp := t.TempDir()
				testkit.WriteFiles(t, tmp, files)
				world, loadErr := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
				if loadErr != nil {
					t.Fatalf("LoadWorld: %v", loadErr)
				}
				_, err = compiler.CheckWorld(world)
			} else {
				err = testkit.CheckProgram(`
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
`)
			}
			assertGenericProtocolRequirementOwnershipMismatchDiagnostic(t, err, tt.wantText)
		})
	}
}

func TestSafetyDiagnosticCodesForProtocolImplOwnershipMismatch(t *testing.T) {
	tests := []struct {
		name     string
		cross    bool
		wantText string
	}{
		{
			name:     "same module",
			wantText: "method 'Box.sink' does not match protocol 'Sink' requirement 'sink': parameter 1 ownership differs: expected 'consume', got 'owned'",
		},
		{
			name:     "cross module",
			cross:    true,
			wantText: "method 'lib.model.Box.sink' does not match protocol 'lib.model.Sink' requirement 'sink': parameter 1 ownership differs: expected 'consume', got 'owned'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.cross {
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
				tmp := t.TempDir()
				testkit.WriteFiles(t, tmp, files)
				world, loadErr := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
				if loadErr != nil {
					t.Fatalf("LoadWorld: %v", loadErr)
				}
				_, err = compiler.CheckWorld(world)
			} else {
				err = testkit.CheckProgram(`
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
`)
			}
			assertProtocolImplOwnershipMismatchDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertProtocolImplOwnershipMismatchDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected protocol impl ownership mismatch diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSemantic || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSemantic)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want protocol impl ownership mismatch diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSemantic+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSemantic)
	}
}

func assertGenericProtocolRequirementOwnershipMismatchDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected generic protocol requirement ownership mismatch diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSemantic || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSemantic)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want generic protocol ownership mismatch diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSemantic+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSemantic)
	}
}

func TestSafetyDiagnosticCodesForTypedErrorResourceAliasFinalization(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "catch payload alias after join",
			src: `
enum TaskErr:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch fail(task):
    case TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		},
		{
			name: "rethrow through try payload alias after join",
			src: `
enum TaskErr:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)

func wrapper(task: task.i32) -> Int throws TaskErr
uses runtime:
    return try fail(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch wrapper(task):
    case TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, "cannot use joined resource 'other'") {
				t.Fatalf("message = %q, want joined alias diagnostic", diag.Message)
			}
			raw, err := json.Marshal(diag)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForResourceFinalizationMergeDiagnostics(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "branch maybe joined task handle",
			src: `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    if 1:
        let _: Int = core.task_join_i32(task)
    return core.task_join_i32(task)
`,
			wantText: "may have been joined after control-flow merge",
		},
		{
			name: "loop maybe closed task group",
			src: `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    var i: Int = 0
    while i < 1:
        let closed: Int = core.task_group_close(group)
        i = i + 1
    return core.task_group_close(group)
`,
			wantText: "may have been closed after control-flow merge",
		},
		{
			name: "match maybe freed island",
			src: `
enum Choice:
    case freeit
    case keep

func choose() -> Choice:
    return Choice.freeit

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let choice: Choice = choose()
        match choice:
        case Choice.freeit:
            free(isl)
        case Choice.keep:
            let kept: Int = 0
        free(isl)
    }
    return 0
`,
			wantText: "may have been freed after control-flow merge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
			}
			if diag.Line == 0 || diag.Column == 0 {
				t.Fatalf("diagnostic position missing: %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantText) {
				t.Fatalf("message = %q, want substring %q", diag.Message, tt.wantText)
			}
			raw, err := json.Marshal(diag)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForOptionalPayloadWholeValueConsumeAndFree(t *testing.T) {
	t.Run("same module payload consume", func(t *testing.T) {
		src := `
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
`
		err := testkit.CheckProgram(src)
		assertOptionalPayloadWholeValueDiagnostic(t, err, "cannot use consumed value 'maybe.$elem'")
	})

	t.Run("cross module payload consume", func(t *testing.T) {
		files := map[string]string{
			"lib/leaks.t4": `module lib.leaks

pub func take(raw: consume ptr) -> ptr:
    return raw

pub func use(value: ptr?) -> Int:
    return 0

pub func leak(maybe: ptr?) -> Int:
    match maybe:
    case some(raw):
        let moved: ptr = take(raw)
    case none:
        let untouched: Int = 0
    return use(maybe)
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
		assertOptionalPayloadWholeValueDiagnostic(t, err, "cannot use consumed value 'maybe.$elem'")
	})

	t.Run("same module payload free", func(t *testing.T) {
		src := `
func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        match maybe:
        case some(other):
            free(other)
            return use(maybe)
        case none:
            return 0
    }
    return 0
`
		err := testkit.CheckProgram(src)
		assertOptionalPayloadWholeValueDiagnostic(t, err, "cannot use freed resource 'maybe.$elem'")
	})

	t.Run("cross module payload free", func(t *testing.T) {
		files := map[string]string{
			"lib/leaks.t4": `module lib.leaks

pub func use(value: island?) -> Int:
    return 0

pub func leak(maybe: island?) -> Int
uses alloc, islands, mem:
    unsafe {
        match maybe:
        case some(other):
            free(other)
            return use(maybe)
        case none:
            return 0
    }
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
		assertOptionalPayloadWholeValueDiagnostic(t, err, "cannot use freed resource 'maybe.$elem'")
	})
}

func assertOptionalPayloadWholeValueDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected optional-payload whole-value diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want optional-payload whole-value diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodeForCallableMutableCaptureGlobalEscape(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var total: Int = 1
    cb = fn(x: Int) -> Int:
        return total + x
    return 0
`)
	file, err := compiler.ParseFile(src, "callable_mutable_capture_global_escape.tetra")
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
		t.Fatalf("expected callable mutable capture global escape diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "global-escaped function value captures mutable local 'total'") {
		t.Fatalf("message = %q, want callable mutable capture global escape", diag.Message)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrOptionalAssignmentGlobalEscape(t *testing.T) {
	tests := []struct {
		name string
		file string
		src  string
	}{
		{
			name: "if-let",
			file: "borrowed_ptr_optional_assignment_global_escape.tetra",
			src: `
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
`,
		},
		{
			name: "match",
			file: "borrowed_ptr_optional_assignment_match_global_escape.tetra",
			src: `
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
`,
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
			assertBorrowedPtrOptionalAssignmentGlobalEscapeDiagnostic(t, err)
		})
	}

	crossModuleTests := []struct {
		name string
		body string
	}{
		{
			name: "if-let",
			body: `
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
`,
		},
		{
			name: "match",
			body: `
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
		},
	}
	for _, tt := range crossModuleTests {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": `module lib.leaks

var leaked: ptr = 0

pub func leak(x: borrow ptr) -> Int:
    var maybe: ptr? = none
    maybe = x
` + tt.body,
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
			assertBorrowedPtrOptionalAssignmentGlobalEscapeDiagnostic(t, err)
		})
	}
}

func assertBorrowedPtrOptionalAssignmentGlobalEscapeDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr optional assignment global escape diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'x' cannot escape via global assignment to 'leaked'") {
		t.Fatalf("message = %q, want borrowed ptr optional assignment global escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrEnumAliasReturnEscapes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		src := `
enum PtrMsg:
    case raw(ptr)

func leak(x: borrow ptr) -> PtrMsg:
    let msg: PtrMsg = PtrMsg.raw(x)
    return msg

func main() -> Int:
    return 0
`
		file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_enum_alias_return_escape.tetra")
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		_, err = compiler.CheckWorld(world)
		assertBorrowedPtrEnumAliasReturnDiagnostic(t, err)
	})

	t.Run("cross module", func(t *testing.T) {
		files := map[string]string{
			"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
`,
			"app/main.t4": `module app.main
import lib.model as model

func leak(x: borrow ptr) -> model.PtrMsg:
    let msg: model.PtrMsg = model.PtrMsg.raw(x)
    return msg

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
		assertBorrowedPtrEnumAliasReturnDiagnostic(t, err)
	})
}

func assertBorrowedPtrEnumAliasReturnDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr enum alias return diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, "borrowed local 'x' cannot escape via return") {
		t.Fatalf("message = %q, want borrowed ptr enum alias return escape", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrAggregateReturnEscapes(t *testing.T) {
	sameModuleTests := []struct {
		name         string
		file         string
		src          string
		borrowedName string
	}{
		{
			name: "whole aggregate",
			file: "borrowed_ptr_aggregate_return_escape.tetra",
			src: `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    return box

func main() -> Int:
    return 0
`,
			borrowedName: "box",
		},
		{
			name: "field",
			file: "borrowed_ptr_aggregate_field_return_escape.tetra",
			src: `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> ptr:
    return box.raw

func main() -> Int:
    return 0
`,
			borrowedName: "box",
		},
		{
			name: "alias",
			file: "borrowed_ptr_aggregate_alias_return_escape.tetra",
			src: `
struct PtrBox:
    raw: ptr

func leak(box: borrow PtrBox) -> PtrBox:
    let alias: PtrBox = box
    return alias

func main() -> Int:
    return 0
`,
			borrowedName: "box",
		},
		{
			name: "nested field",
			file: "borrowed_ptr_nested_aggregate_field_return_escape.tetra",
			src: `
struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

func leak(outer: borrow OuterBox) -> ptr:
    return outer.box.raw

func main() -> Int:
    return 0
`,
			borrowedName: "outer",
		},
	}
	for _, tt := range sameModuleTests {
		t.Run("same module "+tt.name, func(t *testing.T) {
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
			assertBorrowedPtrAggregateReturnDiagnostic(t, err, tt.borrowedName)
		})
	}

	crossModuleTests := []struct {
		name         string
		body         string
		returnType   string
		paramType    string
		borrowedName string
	}{
		{
			name: "whole aggregate",
			body: `
    return box
`,
			returnType:   "model.PtrBox",
			paramType:    "model.PtrBox",
			borrowedName: "box",
		},
		{
			name: "field",
			body: `
    return box.raw
`,
			returnType:   "ptr",
			paramType:    "model.PtrBox",
			borrowedName: "box",
		},
		{
			name: "alias",
			body: `
    let alias: model.PtrBox = box
    return alias
`,
			returnType:   "model.PtrBox",
			paramType:    "model.PtrBox",
			borrowedName: "box",
		},
		{
			name: "nested field",
			body: `
    return outer.box.raw
`,
			returnType:   "ptr",
			paramType:    "model.OuterBox",
			borrowedName: "outer",
		},
	}
	for _, tt := range crossModuleTests {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			paramName := tt.borrowedName
			files := map[string]string{
				"lib/model.t4": `module lib.model

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox
`,
				"app/main.t4": `module app.main
import lib.model as model

func leak(` + paramName + `: borrow ` + tt.paramType + `) -> ` + tt.returnType + `:` + tt.body + `
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
			assertBorrowedPtrAggregateReturnDiagnostic(t, err, tt.borrowedName)
		})
	}
}

func assertBorrowedPtrAggregateReturnDiagnostic(t *testing.T, err error, borrowedName string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr aggregate return diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	wantText := "borrowed local '" + borrowedName + "' cannot escape via return"
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed ptr aggregate return escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrEnumPayloadEscapes(t *testing.T) {
	sameModuleTests := []struct {
		name     string
		file     string
		src      string
		wantText string
	}{
		{
			name: "return",
			file: "borrowed_ptr_enum_payload_return_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'msg' cannot escape via return",
		},
		{
			name: "global",
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
			name: "inout",
			file: "borrowed_ptr_enum_payload_inout_escape.tetra",
			src: `
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
`,
			wantText: "borrowed local 'msg' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range sameModuleTests {
		t.Run("same module "+tt.name, func(t *testing.T) {
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
			assertBorrowedPtrEnumPayloadDiagnostic(t, err, tt.wantText)
		})
	}

	crossModuleTests := []struct {
		name     string
		body     string
		extra    string
		params   string
		retType  string
		wantText string
	}{
		{
			name: "return",
			body: `
    match msg:
    case model.PtrMsg.raw(raw):
        return raw
    case model.PtrMsg.empty:
        return 0
`,
			params:   "msg: borrow model.PtrMsg",
			retType:  "ptr",
			wantText: "borrowed local 'msg' cannot escape via return",
		},
		{
			name: "global",
			extra: `
var leaked: ptr = 0
`,
			body: `
    match msg:
    case model.PtrMsg.raw(raw):
        leaked = raw
        return 0
    case model.PtrMsg.empty:
        return 0
`,
			params:   "msg: borrow model.PtrMsg",
			retType:  "Int",
			wantText: "borrowed local 'msg' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "inout",
			body: `
    match msg:
    case model.PtrMsg.raw(raw):
        out = raw
        return 0
    case model.PtrMsg.empty:
        return 0
`,
			params:   "msg: borrow model.PtrMsg, out: inout ptr",
			retType:  "Int",
			wantText: "borrowed local 'msg' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range crossModuleTests {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/model.t4": `module lib.model

pub enum PtrMsg:
    case raw(ptr)
    case empty
`,
				"app/main.t4": `module app.main
import lib.model as model
` + tt.extra + `
func leak(` + tt.params + `) -> ` + tt.retType + `:` + tt.body + `
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
			assertBorrowedPtrEnumPayloadDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertBorrowedPtrEnumPayloadDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr enum-payload diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed ptr enum-payload escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}

func TestSafetyDiagnosticCodesForBorrowedPtrOptionalPayloadEscapes(t *testing.T) {
	type caseDef struct {
		name     string
		body     string
		extra    string
		params   string
		retType  string
		wantText string
	}

	cases := []caseDef{
		{
			name: "if-let return",
			body: `
    if let raw = maybe:
        return raw
    else:
        return 0
`,
			params:   "maybe: borrow ptr?",
			retType:  "ptr",
			wantText: "borrowed local 'maybe' cannot escape via return",
		},
		{
			name: "match return",
			body: `
    match maybe:
    case some(raw):
        return raw
    case none:
        return 0
`,
			params:   "maybe: borrow ptr?",
			retType:  "ptr",
			wantText: "borrowed local 'maybe' cannot escape via return",
		},
		{
			name: "if-let global",
			extra: `
var leaked: ptr = 0
`,
			body: `
    if let raw = maybe:
        leaked = raw
        return 0
    else:
        return 0
`,
			params:   "maybe: borrow ptr?",
			retType:  "Int",
			wantText: "borrowed local 'maybe' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "match global",
			extra: `
var leaked: ptr = 0
`,
			body: `
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`,
			params:   "maybe: borrow ptr?",
			retType:  "Int",
			wantText: "borrowed local 'maybe' cannot escape via global assignment to 'leaked'",
		},
		{
			name: "if-let inout",
			body: `
    if let raw = maybe:
        out = raw
        return 0
    else:
        return 0
`,
			params:   "maybe: borrow ptr?, out: inout ptr",
			retType:  "Int",
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
			params:   "maybe: borrow ptr?, out: inout ptr",
			retType:  "Int",
			wantText: "borrowed local 'maybe' cannot escape via inout assignment to 'out'",
		},
	}

	for _, tt := range cases {
		t.Run("same module "+tt.name, func(t *testing.T) {
			src := tt.extra + `
func leak(` + tt.params + `) -> ` + tt.retType + `:` + tt.body + `
func main() -> Int:
    return 0
`
			file, err := compiler.ParseFile([]byte(src), "borrowed_ptr_optional_payload_"+strings.ReplaceAll(tt.name, " ", "_")+".tetra")
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			world := &compiler.World{
				EntryModule: "",
				Files:       []*compiler.FileAST{file},
				ByModule:    map[string]*compiler.FileAST{"": file},
			}
			_, err = compiler.CheckWorld(world)
			assertBorrowedPtrOptionalPayloadDiagnostic(t, err, tt.wantText)
		})
	}

	for _, tt := range cases {
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/leaks.t4": `module lib.leaks
` + tt.extra + `
pub func leak(` + tt.params + `) -> ` + tt.retType + `:` + tt.body,
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
			assertBorrowedPtrOptionalPayloadDiagnostic(t, err, tt.wantText)
		})
	}
}

func assertBorrowedPtrOptionalPayloadDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected borrowed ptr optional-payload diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want borrowed ptr optional-payload escape %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
	}
}
