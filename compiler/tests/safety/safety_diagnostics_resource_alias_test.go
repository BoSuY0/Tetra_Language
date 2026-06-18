package compiler_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/testkit"
)

func TestSafetyDiagnosticCodesForCrossModuleResourceAliasFinalization(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		wantText string
	}{
		{
			name: "struct-field alias use after free",
			files: map[string]string{
				"lib/resources.t4": `module lib.resources

pub struct IslandBox:
    handle: island
`,
				"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: resources.IslandBox = resources.IslandBox(handle: core.island_new(16))
        let alias: resources.IslandBox = box
        free(box.handle)
        free(alias.handle)
    }
    return 0
`,
			},
			wantText: "cannot use freed resource 'alias.handle'",
		},
		{
			name: "enum-payload alias use after free",
			files: map[string]string{
				"lib/resources.t4": `module lib.resources

pub enum MoveMsg:
    case take(island)

pub func unwrap(msg: MoveMsg) -> island:
    match msg:
    case MoveMsg.take(handle):
        return handle
`,
				"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: resources.MoveMsg = resources.MoveMsg.take(core.island_new(16))
        let other: island = resources.unwrap(msg)
        match msg:
        case resources.MoveMsg.take(handle):
            free(handle)
            free(other)
    }
    return 0
`,
			},
			wantText: "cannot use freed resource 'other'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, tt.files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
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

func TestSafetyDiagnosticCodesForGenericBorrowReturns(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "same module aggregate",
			src: `
struct PtrBox:
    raw: ptr

func leak<T>(value: borrow T) -> T:
    return value

func caller(x: borrow ptr) -> PtrBox:
    return leak(PtrBox(raw: x))

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
		{
			name: "same module optional ptr",
			src: `
func leak<T>(value: borrow T) -> T:
    return value

func caller(maybe: borrow ptr?) -> ptr?:
    return leak(maybe)

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
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
			if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForCrossModuleGenericBorrowReturns(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		wantText string
	}{
		{
			name: "aggregate",
			files: map[string]string{
				"lib/leak.t4": `module lib.leak

pub struct PtrBox:
    raw: ptr

pub func leak<T>(value: borrow T) -> T:
    return value
`,
				"app/main.t4": `module app.main
import lib.leak as leaks

func caller(x: borrow ptr) -> leaks.PtrBox:
    return leaks.leak(leaks.PtrBox(raw: x))

func main() -> Int:
    return 0
`,
			},
			wantText: "borrowed local 'value' cannot escape via return",
		},
		{
			name: "optional ptr",
			files: map[string]string{
				"lib/leak.t4": `module lib.leak

pub func leak<T>(value: borrow T) -> T:
    return value
`,
				"app/main.t4": `module app.main
import lib.leak as leaks

func caller(maybe: borrow ptr?) -> ptr?:
    return leaks.leak(maybe)

func main() -> Int:
    return 0
`,
			},
			wantText: "borrowed local 'value' cannot escape via return",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, tt.files)
			world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			_, err = compiler.CheckWorld(world)
			if err == nil {
				t.Fatalf("expected safety diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeSafetyLifetime || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyLifetime)
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
			if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyLifetime+`"`) {
				t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyLifetime)
			}
		})
	}
}

func TestSafetyDiagnosticCodesForFunctionTypedOptionalPtrCallbacks(t *testing.T) {
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
			name: "struct-field-owned",
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
			name: "struct-field-consume",
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
			name: "struct-field-inout",
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

func TestSafetyDiagnosticCodesForGenericResourceAliasFinalization(t *testing.T) {
	t.Run("same module task-handle generic struct alias join", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct Box<T>:
    value: T

func worker() -> Int:
    return 7

func pass_task(box: Box<task.i32>) -> Box<task.i32>:
    return box

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: Box<task.i32> = Box<task.i32>{value: task}
    let returned: Box<task.i32> = pass_task(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.value)
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'returned.value'")
	})

	t.Run("cross module task-handle generic struct alias join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_task(box: Box<task.i32>) -> Box<task.i32>:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.Box<task.i32> = resources.Box<task.i32>{value: task}
    let returned: resources.Box<task.i32> = resources.pass_task(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.value)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'returned.value'")
	})

	t.Run("same module task-group generic struct alias close", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct Box<T>:
    value: T

func pass_group(box: Box<task.group>) -> Box<task.group>:
    return box

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: Box<task.group> = Box<task.group>{value: group}
    let returned: Box<task.group> = pass_group(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.value)
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'returned.value'")
	})

	t.Run("cross module task-group generic struct alias close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_group(box: Box<task.group>) -> Box<task.group>:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: resources.Box<task.group> = resources.Box<task.group>{value: group}
    let returned: resources.Box<task.group> = resources.pass_group(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.value)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'returned.value'")
	})

	t.Run("same module island generic struct alias free", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct Box<T>:
    value: T

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: Box<island> = Box<island>{value: core.island_new(16)}
        let alias: Box<island> = box
        free(box.value)
        free(alias.value)
    }
    return 0
`)
		assertResourceAliasFinalizationDiagnostic(t, err, "cannot use freed resource 'alias.value'")
	})

	t.Run("cross module island generic struct alias free", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct Box<T>:
    value: T
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: resources.Box<island> = resources.Box<island>{value: core.island_new(16)}
        let alias: resources.Box<island> = box
        free(box.value)
        free(alias.value)
    }
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
		assertResourceAliasFinalizationDiagnostic(t, err, "cannot use freed resource 'alias.value'")
	})
}

func assertResourceAliasFinalizationDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected resource alias finalization diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want resource alias finalization diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForTransitiveResourceAliasFinalization(t *testing.T) {
	t.Run("same module task-handle transitive alias join", func(t *testing.T) {
		err := testkit.CheckProgram(`
func worker() -> Int:
    return 7

func alias_one(task: task.i32) -> task.i32:
    return task

func alias_two(task: task.i32) -> task.i32:
    return alias_one(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = alias_two(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle transitive alias join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func alias_one(task: task.i32) -> task.i32:
    return task

pub func alias_two(task: task.i32) -> task.i32:
    return alias_one(task)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = resources.alias_two(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group transitive alias close", func(t *testing.T) {
		err := testkit.CheckProgram(`
func alias_one(group: task.group) -> task.group:
    return group

func alias_two(group: task.group) -> task.group:
    return alias_one(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let other: task.group = alias_two(group)
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(other)
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group transitive alias close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func alias_one(group: task.group) -> task.group:
    return group

pub func alias_two(group: task.group) -> task.group:
    return alias_one(group)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let other: task.group = resources.alias_two(group)
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(other)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

	t.Run("same module island transitive alias free", func(t *testing.T) {
		err := testkit.CheckProgram(`
func alias_one(isl: island) -> island:
    return isl

func alias_two(isl: island) -> island:
    return alias_one(isl)

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = alias_two(isl)
        free(isl)
        free(other)
    }
    return 0
`)
		assertResourceAliasFinalizationDiagnostic(t, err, "cannot use freed resource 'other'")
	})

	t.Run("cross module island transitive alias free", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func alias_one(isl: island) -> island:
    return isl

pub func alias_two(isl: island) -> island:
    return alias_one(isl)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = resources.alias_two(isl)
        free(isl)
        free(other)
    }
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
		assertResourceAliasFinalizationDiagnostic(t, err, "cannot use freed resource 'other'")
	})
}

func TestSafetyDiagnosticCodesForEnumConstructorReturnResourceAliases(t *testing.T) {
	t.Run("same module task-handle enum constructor return alias join", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum TaskMsg:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: TaskMsg = wrap(task)
    match returned:
    case TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle enum constructor return alias join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskMsg = resources.wrap(task)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
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
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group enum constructor return alias close", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum GroupMsg:
    case wrap(task.group)

func wrap(group: task.group) -> GroupMsg:
    return GroupMsg.wrap(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: GroupMsg = wrap(group)
    match returned:
    case GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group enum constructor return alias close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum GroupMsg:
    case wrap(task.group)

pub func wrap(group: task.group) -> GroupMsg:
    return GroupMsg.wrap(group)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: resources.GroupMsg = resources.wrap(group)
    match returned:
    case resources.GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
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
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

}

func TestSafetyDiagnosticCodesForActorStructFieldEnumPayloadAliasTransfer(t *testing.T) {
	t.Run("same module transitive interprocedural actor alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
func worker() -> Int:
    return 0

func alias_one(peer: actor) -> actor:
    return peer

func alias_two(peer: actor) -> actor:
    return alias_one(peer)

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let other: actor = alias_two(peer)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("cross module transitive interprocedural actor alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func alias_one(peer: actor) -> actor:
    return peer

pub func alias_two(peer: actor) -> actor:
    return alias_one(peer)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let other: actor = resources.alias_two(peer)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("same module struct-field alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct ActorBox:
    handle: actor

func worker() -> Int:
    return 0

func pass(box: ActorBox) -> ActorBox:
    return box

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: ActorBox = ActorBox(handle: peer)
    let returned: ActorBox = pass(box)
    let _: Int = take_actor(peer)
    return core.send(returned.handle, 1)
`)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.handle'")
	})

	t.Run("cross module struct-field alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct ActorBox:
    handle: actor

pub func pass(box: ActorBox) -> ActorBox:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: resources.ActorBox = resources.ActorBox(handle: peer)
    let returned: resources.ActorBox = resources.pass(box)
    let _: Int = take_actor(peer)
    return core.send(returned.handle, 1)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.handle'")
	})

	t.Run("same module generic struct-field alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct Box<T>:
    value: T

func worker() -> Int:
    return 0

func pass_actor(box: Box<actor>) -> Box<actor>:
    return box

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: Box<actor> = Box<actor>{value: peer}
    let returned: Box<actor> = pass_actor(box)
    let _: Int = take_actor(peer)
    return core.send(returned.value, 1)
`)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.value'")
	})

	t.Run("cross module generic struct-field alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_actor(box: Box<actor>) -> Box<actor>:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: resources.Box<actor> = resources.Box<actor>{value: peer}
    let returned: resources.Box<actor> = resources.pass_actor(box)
    let _: Int = take_actor(peer)
    return core.send(returned.value, 1)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.value'")
	})

	t.Run("same module enum-payload alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum ActorMsg:
    case wrap(actor)

func worker() -> Int:
    return 0

func pass(msg: ActorMsg) -> ActorMsg:
    return msg

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let msg: ActorMsg = ActorMsg.wrap(peer)
    let returned: ActorMsg = pass(msg)
    match returned:
    case ActorMsg.wrap(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`)
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("cross module enum-payload alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum ActorMsg:
    case wrap(actor)

pub func pass(msg: ActorMsg) -> ActorMsg:
    return msg
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let msg: resources.ActorMsg = resources.ActorMsg.wrap(peer)
    let returned: resources.ActorMsg = resources.pass(msg)
    match returned:
    case resources.ActorMsg.wrap(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
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
		assertActorAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})
}

func assertActorAliasTransferDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected actor alias transfer diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want actor alias transfer diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForActorTaskOptionalPayloadAliasTransfer(t *testing.T) {
	t.Run("same module actor if-let optional-payload alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
func worker() -> Int:
    return 0

func pass(maybe: actor?) -> actor?:
    return maybe

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let maybe: actor? = peer
    let returned: actor? = pass(maybe)
    if let other = returned:
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`)
		assertActorTaskOptionalPayloadAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("cross module actor match optional-payload alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func pass(maybe: actor?) -> actor?:
    return maybe
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let maybe: actor? = peer
    let returned: actor? = resources.pass(maybe)
    match returned:
    case some(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    case none:
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
		assertActorTaskOptionalPayloadAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("same module task if-let optional-payload alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
func worker() -> Int:
    return 7

func pass(maybe: task.i32?) -> task.i32?:
    return maybe

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    let returned: task.i32? = pass(maybe)
    if let other = returned:
        let first: Int = take_task(task)
        return first + core.task_join_i32(other)
    return 0
`)
		assertActorTaskOptionalPayloadAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("cross module task match optional-payload alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func pass(maybe: task.i32?) -> task.i32?:
    return maybe
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    let returned: task.i32? = resources.pass(maybe)
    match returned:
    case some(other):
        let first: Int = take_task(task)
        return first + core.task_join_i32(other)
    case none:
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
		assertActorTaskOptionalPayloadAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})
}

func assertActorTaskOptionalPayloadAliasTransferDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected actor/task optional-payload alias transfer diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want actor/task optional-payload alias transfer diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}
