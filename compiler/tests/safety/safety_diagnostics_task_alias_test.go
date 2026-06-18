package compiler_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/testkit"
)

func TestSafetyDiagnosticCodesForTaskHandleStructFieldEnumPayloadAliasTransfer(t *testing.T) {
	t.Run("same module struct-field alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct TaskBox:
    handle: task.i32

func worker() -> Int:
    return 7

func pass(box: TaskBox) -> TaskBox:
    return box

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: TaskBox = TaskBox(handle: task)
    let returned: TaskBox = pass(box)
    let first: Int = take_task(task)
    return first + core.task_join_i32(returned.handle)
`)
		assertTaskHandleAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.handle'")
	})

	t.Run("cross module struct-field alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    return box
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
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = take_task(task)
    return first + core.task_join_i32(returned.handle)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasTransferDiagnostic(t, err, "cannot use consumed value 'returned.handle'")
	})

	t.Run("same module enum-payload alias transfer", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum TaskMsg:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func pass(msg: TaskMsg) -> TaskMsg:
    return msg

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let msg: TaskMsg = TaskMsg.wrap(task)
    let returned: TaskMsg = pass(msg)
    match returned:
    case TaskMsg.wrap(other):
        let first: Int = take_task(task)
        return first + core.task_join_i32(other)
    return 0
`)
		assertTaskHandleAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})

	t.Run("cross module enum-payload alias transfer", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func pass(msg: TaskMsg) -> TaskMsg:
    return msg
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
    let msg: resources.TaskMsg = resources.TaskMsg.wrap(task)
    let returned: resources.TaskMsg = resources.pass(msg)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = take_task(task)
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
		assertTaskHandleAliasTransferDiagnostic(t, err, "cannot use consumed value 'other'")
	})
}

func assertTaskHandleAliasTransferDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected task-handle alias transfer diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want task-handle alias transfer diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForTaskHandleStructFieldEnumPayloadAliasJoin(t *testing.T) {
	t.Run("same module struct-field alias join", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct TaskBox:
    handle: task.i32

func worker() -> Int:
    return 7

func pass(box: TaskBox) -> TaskBox:
    return box

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: TaskBox = TaskBox(handle: task)
    let returned: TaskBox = pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'returned.handle'")
	})

	t.Run("cross module struct-field alias join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'returned.handle'")
	})

	t.Run("same module enum-payload alias join", func(t *testing.T) {
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

	t.Run("cross module enum-payload alias join", func(t *testing.T) {
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
}

func assertTaskHandleAliasJoinDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected task-handle alias join diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want task-handle alias join diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForTaskGroupStructFieldEnumPayloadAliasClose(t *testing.T) {
	t.Run("same module struct-field alias close", func(t *testing.T) {
		err := testkit.CheckProgram(`
struct GroupBox:
    handle: task.group

func pass(box: GroupBox) -> GroupBox:
    return box

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: GroupBox = GroupBox(handle: group)
    let returned: GroupBox = pass(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.handle)
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'returned.handle'")
	})

	t.Run("cross module struct-field alias close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub struct GroupBox:
    handle: task.group

pub func pass(box: GroupBox) -> GroupBox:
    return box
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: resources.GroupBox = resources.GroupBox(handle: group)
    let returned: resources.GroupBox = resources.pass(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.handle)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'returned.handle'")
	})

	t.Run("same module enum-payload alias close", func(t *testing.T) {
		err := testkit.CheckProgram(`
enum GroupMsg:
    case wrap(task.group)

func pass(msg: GroupMsg) -> GroupMsg:
    return msg

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let msg: GroupMsg = GroupMsg.wrap(group)
    let returned: GroupMsg = pass(msg)
    match returned:
    case GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

	t.Run("cross module enum-payload alias close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub enum GroupMsg:
    case wrap(task.group)

pub func pass(msg: GroupMsg) -> GroupMsg:
    return msg
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let msg: resources.GroupMsg = resources.GroupMsg.wrap(group)
    let returned: resources.GroupMsg = resources.pass(msg)
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

func assertTaskGroupAliasCloseDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected task-group alias close diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want task-group alias close diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForTaskHandleGroupOptionalPayloadJoinCloseAliases(t *testing.T) {
	t.Run("same module task-handle if-let optional-payload join", func(t *testing.T) {
		err := testkit.CheckProgram(`
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    if let other = maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`)
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle match optional-payload join", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func pass(maybe: task.i32?) -> task.i32?:
    return maybe
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    let returned: task.i32? = resources.pass(maybe)
    match returned:
    case some(other):
        let first: Int = core.task_join_i32(task)
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
		assertTaskHandleAliasJoinDiagnostic(t, err, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group if-let optional-payload close", func(t *testing.T) {
		err := testkit.CheckProgram(`
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let other = maybe:
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group match optional-payload close", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func pass(maybe: task.group?) -> task.group?:
    return maybe
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    let returned: task.group? = resources.pass(maybe)
    match returned:
    case some(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
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
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'other'")
	})
}

func TestSafetyDiagnosticCodesForTaskGroupCancelReturnProvenance(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		err := testkit.CheckProgram(`
func cancel(group: task.group) -> task.group
uses runtime:
    return core.task_group_cancel(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'canceled'")
	})

	t.Run("cross module", func(t *testing.T) {
		files := map[string]string{
			"lib/resources.t4": `module lib.resources

pub func cancel(group: task.group) -> task.group
uses runtime:
    return core.task_group_cancel(group)
`,
			"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = resources.cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`,
		}
		tmp := t.TempDir()
		testkit.WriteFiles(t, tmp, files)
		world, err := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
		if err != nil {
			t.Fatalf("LoadWorld: %v", err)
		}
		_, err = compiler.CheckWorld(world)
		assertTaskGroupAliasCloseDiagnostic(t, err, "cannot use closed resource 'canceled'")
	})
}

func TestSafetyDiagnosticCodesForPtrContainingNestedAggregateCallRejections(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		nested   bool
		wantText string
	}{
		{
			name:     "ptr-containing aggregate owned call",
			mode:     "owned",
			wantText: "borrowed value derived from 'box' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name:     "ptr-containing aggregate consume call",
			mode:     "consume",
			wantText: "borrowed value derived from 'box' cannot be consumed by 'sink'",
		},
		{
			name:     "ptr-containing aggregate inout call",
			mode:     "inout",
			wantText: "borrowed value derived from 'box' cannot be passed as inout to 'sink'",
		},
		{
			name:     "nested ptr-containing aggregate owned call",
			mode:     "owned",
			nested:   true,
			wantText: "borrowed value derived from 'outer' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name:     "nested ptr-containing aggregate consume call",
			mode:     "consume",
			nested:   true,
			wantText: "borrowed value derived from 'outer' cannot be consumed by 'sink'",
		},
		{
			name:     "nested ptr-containing aggregate inout call",
			mode:     "inout",
			nested:   true,
			wantText: "borrowed value derived from 'outer' cannot be passed as inout to 'sink'",
		},
	}

	for _, tt := range tests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(ptrAggregateCallEscapeSource(tt.mode, tt.nested, false))
			assertPtrContainingNestedAggregateCallDiagnostic(t, err, tt.wantText)
		})
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": `module lib.sink

pub struct PtrBox:
    raw: ptr

pub struct OuterBox:
    box: PtrBox

` + ptrAggregateSinkSource(tt.mode, tt.nested, false, true),
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(` + ptrAggregateBorrowParam(tt.nested, true) + `) -> Int:
    return sinker.sink(` + ptrAggregateBorrowArg(tt.nested) + `)

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
			assertPtrContainingNestedAggregateCallDiagnostic(t, err, strings.Replace(tt.wantText, "'sink'", "'lib.sink.sink'", 1))
		})
	}
}

func ptrAggregateCallEscapeSource(mode string, nested bool, qualified bool) string {
	return `
struct PtrBox:
    raw: ptr

struct OuterBox:
    box: PtrBox

` + ptrAggregateSinkSource(mode, nested, qualified, false) + `
func caller(` + ptrAggregateBorrowParam(nested, qualified) + `) -> Int:
    return sink(` + ptrAggregateBorrowArg(nested) + `)

func main() -> Int:
    return 0
`
}

func ptrAggregateSinkSource(mode string, nested bool, qualified bool, public bool) string {
	typeName := "PtrBox"
	if nested {
		typeName = "OuterBox"
	}
	if qualified {
		typeName = "sinker." + typeName
	}
	param := "value: " + typeName
	switch mode {
	case "consume":
		param = "value: consume " + typeName
	case "inout":
		param = "value: inout " + typeName
	}
	body := "    return 0"
	if mode == "inout" {
		if nested {
			body = "    value = OuterBox(box: PtrBox(raw: 0))\n    return 0"
		} else {
			body = "    value = PtrBox(raw: 0)\n    return 0"
		}
	}
	prefix := ""
	if public {
		prefix = "pub "
	}
	return prefix + "func sink(" + param + ") -> Int:\n" + body + "\n"
}

func ptrAggregateBorrowParam(nested bool, qualified bool) string {
	typeName := "PtrBox"
	name := "box"
	if nested {
		typeName = "OuterBox"
		name = "outer"
	}
	if qualified {
		typeName = "sinker." + typeName
	}
	return name + ": borrow " + typeName
}

func ptrAggregateBorrowArg(nested bool) string {
	if nested {
		return "outer"
	}
	return "box"
}

func assertPtrContainingNestedAggregateCallDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected ptr-containing/nested aggregate call diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want ptr-containing/nested aggregate call diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForPtrEnumPayloadCallRejections(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		wantText string
	}{
		{
			name:     "owned call",
			mode:     "owned",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name:     "consume call",
			mode:     "consume",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name:     "inout call",
			mode:     "inout",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'sink'",
		},
	}

	for _, tt := range tests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(ptrEnumPayloadCallEscapeSource(tt.mode, false, false))
			assertPtrEnumPayloadCallDiagnostic(t, err, tt.wantText)
		})
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": `module lib.sink

pub enum PtrMsg:
    case raw(ptr)

` + ptrEnumPayloadSinkSource(tt.mode, false, true),
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(x: borrow ptr) -> Int:
    ` + ptrEnumPayloadCallBody(tt.mode, true) + `

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
			assertPtrEnumPayloadCallDiagnostic(t, err, strings.Replace(tt.wantText, "'sink'", "'lib.sink.sink'", 1))
		})
	}
}

func ptrEnumPayloadCallEscapeSource(mode string, qualified bool, public bool) string {
	return `
enum PtrMsg:
    case raw(ptr)

` + ptrEnumPayloadSinkSource(mode, qualified, public) + `
func caller(x: borrow ptr) -> Int:
    ` + ptrEnumPayloadCallBody(mode, qualified) + `

func main() -> Int:
    return 0
`
}

func ptrEnumPayloadSinkSource(mode string, qualified bool, public bool) string {
	typeName := "PtrMsg"
	if qualified {
		typeName = "sinker." + typeName
	}
	param := "value: " + typeName
	switch mode {
	case "consume":
		param = "value: consume " + typeName
	case "inout":
		param = "value: inout " + typeName
	}
	body := "    return 0"
	if mode == "inout" {
		body = "    value = PtrMsg.raw(0)\n    return 0"
	}
	prefix := ""
	if public {
		prefix = "pub "
	}
	return prefix + "func sink(" + param + ") -> Int:\n" + body + "\n"
}

func ptrEnumPayloadCallBody(mode string, qualified bool) string {
	typeName := "PtrMsg"
	sinkName := "sink"
	if qualified {
		typeName = "sinker." + typeName
		sinkName = "sinker.sink"
	}
	switch mode {
	case "consume":
		return "let msg: " + typeName + " = " + typeName + ".raw(x)\n    return " + sinkName + "(msg)"
	case "inout":
		return "var msg: " + typeName + " = " + typeName + ".raw(x)\n    return " + sinkName + "(msg)"
	default:
		return "return " + sinkName + "(" + typeName + ".raw(x))"
	}
}

func assertPtrEnumPayloadCallDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected ptr enum-payload call diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want ptr enum-payload call diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForPtrOptionalPayloadCallRejections(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		wantText string
	}{
		{
			name:     "owned call",
			mode:     "owned",
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name:     "consume call",
			mode:     "consume",
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'sink'",
		},
		{
			name:     "inout call",
			mode:     "inout",
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'sink'",
		},
	}

	for _, tt := range tests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(ptrOptionalPayloadCallEscapeSource(tt.mode, false, false))
			assertPtrOptionalPayloadCallDiagnostic(t, err, tt.wantText)
		})
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": `module lib.sink

` + ptrOptionalPayloadSinkSource(tt.mode, false, true),
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(maybe: borrow ptr?) -> Int:
    ` + ptrOptionalPayloadCallBody(tt.mode, true) + `

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
			assertPtrOptionalPayloadCallDiagnostic(t, err, strings.Replace(tt.wantText, "'sink'", "'lib.sink.sink'", 1))
		})
	}
}

func ptrOptionalPayloadCallEscapeSource(mode string, qualified bool, public bool) string {
	return ptrOptionalPayloadSinkSource(mode, qualified, public) + `
func caller(maybe: borrow ptr?) -> Int:
    ` + ptrOptionalPayloadCallBody(mode, qualified) + `

func main() -> Int:
    return 0
`
}

func ptrOptionalPayloadSinkSource(mode string, qualified bool, public bool) string {
	param := "raw: ptr"
	switch mode {
	case "consume":
		param = "raw: consume ptr"
	case "inout":
		param = "raw: inout ptr"
	}
	body := "    return 0"
	if mode == "inout" {
		body = "    raw = 0\n    return 0"
	}
	prefix := ""
	if public {
		prefix = "pub "
	}
	return prefix + "func sink(" + param + ") -> Int:\n" + body + "\n"
}

func ptrOptionalPayloadCallBody(mode string, qualified bool) string {
	sinkName := "sink"
	if qualified {
		sinkName = "sinker.sink"
	}
	return `match maybe:
    case some(raw):
        return ` + sinkName + `(raw)
    case none:
        return 0`
}

func assertPtrOptionalPayloadCallDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected ptr optional-payload call diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want ptr optional-payload call diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForSliceOptionalPayloadCallRejections(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		wantText string
	}{
		{
			name:     "owned call",
			mode:     "owned",
			wantText: "borrowed value derived from 'maybe' cannot be passed to non-borrow parameter 1 of 'sink'",
		},
		{
			name:     "consume call",
			mode:     "consume",
			wantText: "borrowed value derived from 'maybe' cannot be consumed by 'sink'",
		},
		{
			name:     "inout call",
			mode:     "inout",
			wantText: "borrowed value derived from 'maybe' cannot be passed as inout to 'sink'",
		},
	}

	for _, tt := range tests {
		t.Run("same module "+tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(sliceOptionalPayloadCallEscapeSource(tt.mode, false, false))
			assertSliceOptionalPayloadCallDiagnostic(t, err, tt.wantText)
		})
		t.Run("cross module "+tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/sink.t4": `module lib.sink

` + sliceOptionalPayloadSinkSource(tt.mode, true),
				"app/main.t4": `module app.main
import lib.sink as sinker

func caller(maybe: borrow []u8?) -> Int:
    ` + sliceOptionalPayloadCallBody(true) + `

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
			assertSliceOptionalPayloadCallDiagnostic(t, err, strings.Replace(tt.wantText, "'sink'", "'lib.sink.sink'", 1))
		})
	}
}

func sliceOptionalPayloadCallEscapeSource(mode string, qualified bool, public bool) string {
	return sliceOptionalPayloadSinkSource(mode, public) + `
func caller(maybe: borrow []u8?) -> Int:
    ` + sliceOptionalPayloadCallBody(qualified) + `

func main() -> Int:
    return 0
`
}

func sliceOptionalPayloadSinkSource(mode string, public bool) string {
	param := "raw: []u8"
	switch mode {
	case "consume":
		param = "raw: consume []u8"
	case "inout":
		param = "raw: inout []u8"
	}
	body := "    return 0"
	if mode == "inout" {
		body = "    raw = raw\n    return 0"
	}
	prefix := ""
	if public {
		prefix = "pub "
	}
	return prefix + "func sink(" + param + ") -> Int:\n" + body + "\n"
}

func sliceOptionalPayloadCallBody(qualified bool) string {
	sinkName := "sink"
	if qualified {
		sinkName = "sinker.sink"
	}
	return `match maybe:
    case some(raw):
        return ` + sinkName + `(raw)
    case none:
        return 0`
}

func assertSliceOptionalPayloadCallDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected slice optional-payload call diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want slice optional-payload call diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}

func TestSafetyDiagnosticCodesForFunctionTypedSliceAggregateCallbackCallRejections(t *testing.T) {
	type caseDef struct {
		name       string
		target     string
		aggregate  string
		mode       string
		cross      bool
		wantCaller string
	}
	var cases []caseDef
	for _, target := range []string{"value", "field", "payload"} {
		for _, aggregate := range []string{"struct", "enum"} {
			for _, mode := range []string{"owned", "consume", "inout"} {
				cases = append(cases, caseDef{
					name:       target + " " + aggregate + " " + mode,
					target:     target,
					aggregate:  aggregate,
					mode:       mode,
					wantCaller: functionTypedSliceCallbackWantCaller(target),
				})
			}
		}
	}
	for _, target := range []string{"field", "payload"} {
		for _, mode := range []string{"owned", "consume", "inout"} {
			cases = append(cases, caseDef{
				name:       "cross " + target + " " + mode,
				target:     target,
				aggregate:  "struct",
				mode:       mode,
				cross:      true,
				wantCaller: functionTypedSliceCallbackWantCaller(target),
			})
		}
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.cross {
				files := functionTypedSliceCallbackCrossModuleFiles(tt.target, tt.aggregate, tt.mode)
				tmp := t.TempDir()
				testkit.WriteFiles(t, tmp, files)
				world, loadErr := compiler.LoadWorld(filepath.Join(tmp, "app", "main.t4"))
				if loadErr != nil {
					t.Fatalf("LoadWorld: %v", loadErr)
				}
				_, err = compiler.CheckWorld(world)
			} else {
				err = testkit.CheckProgram(functionTypedSliceCallbackSource(tt.target, tt.aggregate, tt.mode, false))
			}
			wantText := functionTypedSliceCallbackWantText(tt.mode, tt.wantCaller)
			assertFunctionTypedSliceAggregateCallbackDiagnostic(t, err, wantText)
		})
	}
}

func functionTypedSliceCallbackSource(target, aggregate, mode string, qualified bool) string {
	return functionTypedSliceCallbackTypeDecls(aggregate, false) + "\n" +
		functionTypedSliceCallbackTargetDecl(target, aggregate, mode, false) + "\n" +
		"func caller(" + functionTypedSliceCallbackParam(target, aggregate, mode, qualified) + ", x: borrow []u8) -> Int:\n" +
		functionTypedSliceCallbackBody(target, aggregate, mode, qualified) + "\n\n" +
		"func main() -> Int:\n    return 0\n"
}

func functionTypedSliceCallbackCrossModuleFiles(target, aggregate, mode string) map[string]string {
	return map[string]string{
		"lib/callbacks.t4": "module lib.callbacks\n\n" +
			functionTypedSliceCallbackTypeDecls(aggregate, true) + "\n" +
			functionTypedSliceCallbackTargetDecl(target, aggregate, mode, true),
		"app/main.t4": "module app.main\n" +
			"import lib.callbacks as callbacks\n\n" +
			"func caller(" + functionTypedSliceCallbackParam(target, aggregate, mode, true) + ", x: borrow []u8) -> Int:\n" +
			functionTypedSliceCallbackBody(target, aggregate, mode, true) + "\n\n" +
			"func main() -> Int:\n    return 0\n",
	}
}

func functionTypedSliceCallbackTypeDecls(aggregate string, public bool) string {
	prefix := ""
	if public {
		prefix = "pub "
	}
	if aggregate == "enum" {
		return prefix + "enum BufMsg:\n    case send([]u8)\n"
	}
	return prefix + "struct BufBox:\n    buf: []u8\n"
}

func functionTypedSliceCallbackTargetDecl(target, aggregate, mode string, public bool) string {
	if target == "value" {
		return ""
	}
	prefix := ""
	if public {
		prefix = "pub "
	}
	callbackType := "fn(" + functionTypedSliceCallbackParamType(aggregate, mode, false) + ") -> Int"
	if target == "payload" {
		return prefix + "enum Choice:\n    case some(" + callbackType + ")\n    case empty\n"
	}
	return prefix + "struct Handler:\n    cb: " + callbackType + "\n"
}

func functionTypedSliceCallbackParam(target, aggregate, mode string, qualified bool) string {
	paramType := functionTypedSliceCallbackParamType(aggregate, mode, qualified)
	switch target {
	case "payload":
		typeName := "Choice"
		if qualified {
			typeName = "callbacks.Choice"
		}
		return "choice: " + typeName
	case "field":
		typeName := "Handler"
		if qualified {
			typeName = "callbacks.Handler"
		}
		return "h: " + typeName
	default:
		return "cb: fn(" + paramType + ") -> Int"
	}
}

func functionTypedSliceCallbackParamType(aggregate, mode string, qualified bool) string {
	typeName := "BufBox"
	if aggregate == "enum" {
		typeName = "BufMsg"
	}
	if qualified {
		typeName = "callbacks." + typeName
	}
	switch mode {
	case "consume":
		return "consume " + typeName
	case "inout":
		return "inout " + typeName
	default:
		return typeName
	}
}

func functionTypedSliceCallbackBody(target, aggregate, mode string, qualified bool) string {
	callable := "cb"
	if target == "field" {
		callable = "h.cb"
	}
	arg := functionTypedSliceCallbackAggregateValue(aggregate, "x", qualified)
	if mode == "consume" {
		local := functionTypedSliceCallbackLocalName(aggregate)
		prefix := "    let " + local + ": " + functionTypedSliceCallbackParamType(aggregate, "owned", qualified) + " = " + arg + "\n"
		return prefix + functionTypedSliceCallbackCall(target, callable, local, qualified)
	}
	if mode == "inout" {
		local := functionTypedSliceCallbackLocalName(aggregate)
		prefix := "    var " + local + ": " + functionTypedSliceCallbackParamType(aggregate, "owned", qualified) + " = " + arg + "\n"
		return prefix + functionTypedSliceCallbackCall(target, callable, local, qualified)
	}
	return functionTypedSliceCallbackCall(target, callable, arg, qualified)
}

func functionTypedSliceCallbackCall(target, callable, arg string, qualified bool) string {
	if target != "payload" {
		return "    return " + callable + "(" + arg + ")"
	}
	casePrefix := "Choice"
	if qualified {
		casePrefix = "callbacks.Choice"
	}
	return "    match choice:\n" +
		"    case " + casePrefix + ".some(cb):\n" +
		"        return " + callable + "(" + arg + ")\n" +
		"    case " + casePrefix + ".empty:\n" +
		"        return 0"
}

func functionTypedSliceCallbackAggregateValue(aggregate, source string, qualified bool) string {
	if aggregate == "enum" {
		typeName := "BufMsg"
		if qualified {
			typeName = "callbacks.BufMsg"
		}
		return typeName + ".send(" + source + ")"
	}
	typeName := "BufBox"
	if qualified {
		typeName = "callbacks.BufBox"
	}
	return typeName + "(buf: " + source + ")"
}

func functionTypedSliceCallbackLocalName(aggregate string) string {
	if aggregate == "enum" {
		return "msg"
	}
	return "box"
}

func functionTypedSliceCallbackWantCaller(target string) string {
	switch target {
	case "field":
		return "function-typed struct field call 'h.cb'"
	case "payload":
		return "function-typed enum payload call 'cb'"
	default:
		return "callback 'cb'"
	}
}

func functionTypedSliceCallbackWantText(mode, caller string) string {
	switch mode {
	case "consume":
		return "borrowed value derived from 'x' cannot be consumed by " + caller
	case "inout":
		return "borrowed value derived from 'x' cannot be passed as inout to " + caller
	default:
		return "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of " + caller
	}
}

func assertFunctionTypedSliceAggregateCallbackDiagnostic(t *testing.T, err error, wantText string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected function-typed slice aggregate callback diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyOwnership || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v, want code %s", diag, compiler.DiagnosticCodeSafetyOwnership)
	}
	if diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic position missing: %#v", diag)
	}
	if !strings.Contains(diag.Message, wantText) {
		t.Fatalf("message = %q, want function-typed slice aggregate callback diagnostic %q", diag.Message, wantText)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"code":"`+compiler.DiagnosticCodeSafetyOwnership+`"`) {
		t.Fatalf("json = %s, missing code %s", raw, compiler.DiagnosticCodeSafetyOwnership)
	}
}
