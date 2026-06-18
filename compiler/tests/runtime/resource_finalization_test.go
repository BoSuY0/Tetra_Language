package compiler_test

import (
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func requireCheckWorldFilesErrorContains(
	t *testing.T,
	files map[string]string,
	entry string,
	want string,
) {
	t.Helper()
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, files)
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash(entry)))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func TestTaskHandleFinalizationRejectsDoubleJoin(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(task)
`, "cannot use joined resource 'task'")
}

func TestTaskHandleFinalizationRejectsUseAfterJoinResult(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let alias: task.i32 = task
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(alias)
`, "cannot use joined resource 'alias'")
}

func TestTaskHandleFinalizationRejectsOptionalPayloadAliasAfterJoin(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use joined resource 'other'")
}

func TestTaskHandleFinalizationRejectsInterproceduralOptionalPayloadAliasAfterJoin(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func pass(maybe: task.i32?) -> task.i32?:
    return maybe

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    let returned: task.i32? = pass(maybe)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`, "cannot use joined resource 'other'")
}

func TestTaskHandleFinalizationRejectsInterproceduralOptionalMatchPayloadAliasAfterJoin(
	t *testing.T,
) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func pass(maybe: task.i32?) -> task.i32?:
    return maybe

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    let returned: task.i32? = pass(maybe)
    match returned:
    case some(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    case none:
        return 0
`, "cannot use joined resource 'other'")
}

func TestTaskHandleFinalizationRejectsCrossModuleOptionalPayloadAliasAfterJoin(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func pass(maybe: task.i32?) -> task.i32?:
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
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use joined resource 'other'",
	)
}

func TestTaskHandleFinalizationRejectsCrossModuleOptionalMatchPayloadAliasAfterJoin(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func pass(maybe: task.i32?) -> task.i32?:
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use joined resource 'other'",
	)
}

func TestTaskHandleFinalizationRejectsInterproceduralStructFieldAliasAfterJoin(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use joined resource 'returned.handle'")
}

func TestTaskHandleFinalizationRejectsCrossModuleStructFieldAliasAfterJoin(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use joined resource 'returned.handle'",
	)
}

func TestTaskHandleFinalizationRejectsGenericStructFieldAliasAfterJoin(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use joined resource 'returned.value'")
}

func TestTaskHandleFinalizationRejectsCrossModuleGenericStructFieldAliasAfterJoin(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use joined resource 'returned.value'",
	)
}

func TestTaskHandleFinalizationRejectsCrossModuleTransitiveInterproceduralAliasAfterJoin(
	t *testing.T,
) {
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use joined resource 'other'",
	)
}

func TestTaskHandleFinalizationRejectsInterproceduralEnumPayloadAliasAfterJoin(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum TaskMsg:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func pass(msg: TaskMsg) -> TaskMsg:
    return msg

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let msg: TaskMsg = TaskMsg.wrap(task)
    let returned: TaskMsg = pass(msg)
    match returned:
    case TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`, "cannot use joined resource 'other'")
}

func TestTaskHandleFinalizationRejectsJoinedHandleInThrowPayload(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum TaskErr:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    let _: Int = core.task_join_i32(task)
    throw TaskErr.wrap(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch fail(task):
    case TaskErr.wrap(other):
        core.task_join_i32(other)
`, "cannot use joined resource 'task'")
}

func TestTaskHandleFinalizationRejectsJoinedHandleDirectThrow(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func fail(task: task.i32) -> Int throws task.i32
uses runtime:
    let _: Int = core.task_join_i32(task)
    throw task

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch fail(task):
    case _:
        0
`, "cannot use joined resource 'task'")
}

func TestTaskHandleFinalizationRejectsThrownCatchPayloadAliasAfterJoin(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use joined resource 'other'")
}

func TestTaskHandleFinalizationRejectsRethrownCatchPayloadAliasAfterJoin(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use joined resource 'other'")
}

func TestTaskHandleFinalizationRejectsInterproceduralEnumConstructorReturnAliasAfterJoin(
	t *testing.T,
) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use joined resource 'other'")
}

func TestTaskHandleFinalizationRejectsCrossModuleEnumConstructorReturnAliasAfterJoin(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use joined resource 'other'",
	)
}

func TestTaskHandleFinalizationRejectsCrossModuleEnumPayloadAliasAfterJoin(t *testing.T) {
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

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let msg: resources.TaskMsg = resources.TaskMsg.wrap(task)
    let returned: resources.TaskMsg = resources.pass(msg)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use joined resource 'other'",
	)
}

func TestTaskHandleFinalizationJoinUntilDoesNotConsumeHandle(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let first: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(5))
    return first.value + core.task_join_i32(task)
`)
}

func TestTaskHandleFinalizationReportsMaybeJoinedAfterMerge(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    if 1:
        let _: Int = core.task_join_i32(task)
    return core.task_join_i32(task)
`, "may have been joined after control-flow merge")
}

func TestTaskHandleFinalizationReportsMaybeJoinedAfterLoop(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    var i: Int = 0
    while i < 1:
        let _: Int = core.task_join_i32(task)
        i = i + 1
    return core.task_join_i32(task)
`, "may have been joined after control-flow merge")
}

func TestTaskHandleFinalizationReportsMaybeJoinedAfterMatch(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum Choice:
    case join
    case skip

func worker() -> Int:
    return 7

func choose() -> Choice:
    return Choice.join

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let choice: Choice = choose()
    match choice:
    case Choice.join:
        let joined: Int = core.task_join_i32(task)
    case Choice.skip:
        let skipped: Int = 0
    return core.task_join_i32(task)
`, "may have been joined after control-flow merge")
}
