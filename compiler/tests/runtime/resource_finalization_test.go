package compiler_test

import (
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func requireCheckWorldFilesErrorContains(t *testing.T, files map[string]string, entry string, want string) {
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

func TestTaskHandleFinalizationRejectsInterproceduralOptionalMatchPayloadAliasAfterJoin(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use joined resource 'other'")
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use joined resource 'other'")
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use joined resource 'returned.handle'")
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use joined resource 'returned.value'")
}

func TestTaskHandleFinalizationRejectsCrossModuleTransitiveInterproceduralAliasAfterJoin(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use joined resource 'other'")
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

func TestTaskHandleFinalizationRejectsInterproceduralEnumConstructorReturnAliasAfterJoin(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use joined resource 'other'")
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use joined resource 'other'")
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

func TestTaskGroupCloseStillAllowsStatus(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return closeError
    return core.task_group_status(group)
`)
}

func TestTaskGroupFinalizationRejectsSpawnAfterClose(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let _: Int = core.task_group_close(group)
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`, "cannot use closed resource 'group'")
}

func TestTaskGroupFinalizationRejectsUserCallAfterClose(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func use_group(group: task.group) -> Int
uses runtime:
    return core.task_group_status(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let _: Int = core.task_group_close(group)
    return use_group(group)
`, "cannot use closed resource 'group'")
}

func TestTaskGroupFinalizationRejectsDoubleClose(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(group)
`, "cannot use closed resource 'group'")
}

func TestTaskGroupFinalizationRejectsOptionalPayloadAliasAfterClose(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let other = maybe:
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`, "cannot use closed resource 'other'")
}

func TestTaskGroupFinalizationRejectsInterproceduralOptionalPayloadAliasAfterClose(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func pass(maybe: task.group?) -> task.group?:
    return maybe

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    let returned: task.group? = pass(maybe)
    if let other = returned:
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`, "cannot use closed resource 'other'")
}

func TestTaskGroupFinalizationRejectsInterproceduralOptionalMatchPayloadAliasAfterClose(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func pass(maybe: task.group?) -> task.group?:
    return maybe

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    let returned: task.group? = pass(maybe)
    match returned:
    case some(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    case none:
        return 0
`, "cannot use closed resource 'other'")
}

func TestTaskGroupFinalizationRejectsCrossModuleOptionalPayloadAliasAfterClose(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func pass(maybe: task.group?) -> task.group?:
    return maybe
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    let returned: task.group? = resources.pass(maybe)
    if let other = returned:
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use closed resource 'other'")
}

func TestTaskGroupFinalizationRejectsCrossModuleOptionalMatchPayloadAliasAfterClose(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func pass(maybe: task.group?) -> task.group?:
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use closed resource 'other'")
}

func TestTaskGroupFinalizationRejectsInterproceduralStructFieldAliasAfterClose(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use closed resource 'returned.handle'")
}

func TestTaskGroupFinalizationRejectsCrossModuleStructFieldAliasAfterClose(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use closed resource 'returned.handle'")
}

func TestTaskGroupFinalizationRejectsGenericStructFieldAliasAfterClose(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use closed resource 'returned.value'")
}

func TestTaskGroupFinalizationRejectsCrossModuleGenericStructFieldAliasAfterClose(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use closed resource 'returned.value'")
}

func TestTaskGroupFinalizationRejectsCrossModuleTransitiveInterproceduralAliasAfterClose(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use closed resource 'other'")
}

func TestTaskGroupFinalizationRejectsInterproceduralEnumPayloadAliasAfterClose(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use closed resource 'other'")
}

func TestTaskGroupFinalizationRejectsCrossModuleEnumPayloadAliasAfterClose(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use closed resource 'other'")
}

func TestTaskGroupFinalizationAllowsReopenAssignmentAfterClose(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let _: Int = core.task_group_close(group)
    group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`)
}

func TestTaskGroupFinalizationMergesClosedBranch(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    if 1:
        let _: Int = core.task_group_close(group)
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`, "cannot use closed resource 'group'")
}

func TestTaskGroupFinalizationReportsMaybeClosedAfterMerge(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    if 1:
        let _: Int = core.task_group_close(group)
    return core.task_group_close(group)
`, "may have been closed after control-flow merge")
}

func TestTaskGroupFinalizationReportsMaybeClosedAfterLoop(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    var i: Int = 0
    while i < 1:
        let closed: Int = core.task_group_close(group)
        i = i + 1
    return core.task_group_close(group)
`, "may have been closed after control-flow merge")
}

func TestTaskGroupFinalizationReportsMaybeClosedAfterMatch(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum Choice:
    case close
    case keep

func choose() -> Choice:
    return Choice.close

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let choice: Choice = choose()
    match choice:
    case Choice.close:
        let closed: Int = core.task_group_close(group)
    case Choice.keep:
        let kept: Int = 0
    return core.task_group_close(group)
`, "may have been closed after control-flow merge")
}

func TestIslandFinalizationRejectsDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        free(isl)
        free(isl)
    }
    return 0
`, "cannot use freed resource 'isl'")
}

func TestIslandFinalizationRejectsIslandMakeAfterFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        free(isl)
        let xs: []u8 = core.island_make_u8(isl, 1)
    }
    return 0
`, "cannot use freed resource 'isl'")
}

func TestIslandResetConsumesSourceToken(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let next: island = core.island_reset(isl)
        free(isl)
        free(next)
    }
    return 0
`, "cannot use consumed value 'isl'")
}

func TestIslandResetRejectsAliasUseAfterSourceReset(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let alias: island = isl
        let next: island = core.island_reset(isl)
        let xs: []u8 = core.island_make_u8(alias, 1)
        free(next)
        return xs[0]
    }
    return 0
`, "cannot use consumed value 'alias'")
}

func TestIslandResetInvalidatesPreviousSliceBorrow(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let old: []u8 = core.island_make_u8(isl, 1)
        let next: island = core.island_reset(isl)
        free(next)
        return old[0]
    }
    return 0
`, "cannot reset island 'isl' while borrowed slice 'old' is alive")
}

func TestIslandResetRejectsWhilePreviousSliceBorrowAlive(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let old: []u8 = core.island_make_u8(isl, 1)
        let next: island = core.island_reset(isl)
        free(next)
    }
    return 0
`, "cannot reset island 'isl' while borrowed slice 'old' is alive")
}

func TestIslandResetAllowsAfterPreviousSliceOwnerCleared(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        var old: []u8 = core.island_make_u8(isl, 1)
        old = make_u8(1)
        let next: island = core.island_reset(isl)
        let fresh: []u8 = core.island_make_u8(next, 1)
        let value: Int = old[0] + fresh[0]
        free(next)
        return value
    }
    return 0
`)
}

func TestIslandResetReturnedTokenCanAllocateAndFree(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let next: island = core.island_reset(isl)
        let fresh: []u8 = core.island_make_u8(next, 1)
        let value: Int = fresh[0]
        free(next)
        return value
    }
    return 0
`)
}

func TestIslandFinalizationReportsMaybeFreedAfterMerge(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        if 1:
            free(isl)
        free(isl)
    }
    return 0
`, "may have been freed after control-flow merge")
}

func TestIslandFinalizationReportsMaybeFreedAfterLoop(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        var i: Int = 0
        while i < 1:
            free(isl)
            i = i + 1
        free(isl)
    }
    return 0
`, "may have been freed after control-flow merge")
}

func TestIslandFinalizationReportsMaybeFreedAfterMatch(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "may have been freed after control-flow merge")
}

func TestIslandFinalizationRejectsAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let alias: island = isl
        free(isl)
        free(alias)
    }
    return 0
`, "cannot use freed resource 'alias'")
}

func TestIslandFinalizationRejectsStructFieldAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let alias: IslandBox = box
        free(box.handle)
        free(alias.handle)
    }
    return 0
`, "cannot use freed resource 'alias.handle'")
}

func TestIslandFinalizationRejectsGenericStructFieldAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use freed resource 'alias.value'")
}

func TestIslandFinalizationRejectsCrossModuleGenericStructFieldAliasDoubleFree(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'alias.value'")
}

func TestIslandFinalizationRejectsStructFieldFreeThenOriginalFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let alias: island = box.handle
        free(box.handle)
        free(alias)
    }
    return 0
`, "cannot use freed resource 'alias'")
}

func TestIslandTransferRejectsAggregateAliasFieldReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

struct OuterBox:
    inner: IslandBox

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let alias: IslandBox = box
        let _: OuterBox = OuterBox(inner: box)
        free(alias.handle)
    }
    return 0
`, "cannot use consumed value 'alias.handle'")
}

func TestIslandTransferRejectsFieldAccessAggregateAliasFieldReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

struct HolderBox:
    inner: IslandBox

struct OuterBox:
    inner: IslandBox

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let holder: HolderBox = HolderBox(inner: box)
        let alias: IslandBox = holder.inner
        let _: OuterBox = OuterBox(inner: holder.inner)
        free(alias.handle)
    }
    return 0
`, "cannot use consumed value 'alias.handle'")
}

func TestIslandFinalizationAllowsSingleStructFieldFree(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        free(box.handle)
    }
    return 0
`)
}

func TestIslandFinalizationRejectsStructFieldMergeAmbiguity(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let left: island = core.island_new(16)
        let right: island = core.island_new(16)
        var box: IslandBox = IslandBox(handle: left)
        if 1:
            box = IslandBox(handle: right)
        free(box.handle)
    }
    return 0
`, "ambiguous resource provenance for 'box.handle'")
}

func TestIslandFinalizationRejectsInterproceduralStructFieldAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

func unwrap(box: IslandBox) -> island:
    return box.handle

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let other: island = unwrap(box)
        free(box.handle)
        free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsInterproceduralStructFieldReturnAmbiguity(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Pair:
    left: island
    right: island

func pick(pair: Pair, flag: Int) -> island:
    if flag:
        return pair.left
    return pair.right

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let pair: Pair = Pair(left: core.island_new(16), right: core.island_new(32))
        let picked: island = pick(pair, 1)
        free(picked)
    }
    return 0
`, "return mixes resource provenance")
}

func TestIslandFinalizationRejectsInterproceduralAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func alias(isl: island) -> island:
    return isl

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = alias(isl)
        free(isl)
        free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsTransitiveInterproceduralAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleTransitiveInterproceduralAliasDoubleFree(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsBranchReturnedAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func branch_alias(isl: island, flag: Int) -> island:
    if flag:
        return isl
    return isl

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = branch_alias(isl, 1)
        free(isl)
        free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsAmbiguousResourceReturn(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func choose_island(left: island, right: island, flag: Int) -> island:
    if flag:
        return left
    return right

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let left: island = core.island_new(16)
        let right: island = core.island_new(16)
        let picked: island = choose_island(left, right, 1)
        free(picked)
    }
    return 0
`, "return mixes resource provenance")
}

func TestIslandFinalizationRejectsMergedLocalAmbiguousResourceReturn(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func choose_island(left: island, right: island, flag: Int) -> island:
    var picked: island = left
    if flag:
        picked = right
    return picked

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let left: island = core.island_new(16)
        let right: island = core.island_new(16)
        let picked: island = choose_island(left, right, 1)
        free(picked)
    }
    return 0
`, "ambiguous resource provenance for 'picked'")
}

func TestActorConsumeRejectsMergedLocalAmbiguousResourceReturn(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 0

func choose_actor(left: actor, right: actor, flag: Int) -> actor:
    var picked: actor = left
    if flag:
        picked = right
    return picked

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let left: actor = core.spawn("worker")
    let right: actor = core.spawn("worker")
    let picked: actor = choose_actor(left, right, 1)
    let _: Int = take_actor(left)
    return core.send(picked, 1)
`, "ambiguous resource provenance for 'picked'")
}

func TestIslandFinalizationRejectsUninferredRecursiveResourceReturn(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func recursive_alias(isl: island) -> island:
    let other: island = recursive_alias(isl)
    return other

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = recursive_alias(isl)
        free(isl)
        free(other)
    }
    return 0
`, "ambiguous resource provenance for 'other'")
}

func TestIslandFinalizationRejectsEnumPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MoveMsg:
    case take(island)

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: MoveMsg = MoveMsg.take(core.island_new(16))
        match msg:
        case MoveMsg.take(other):
            let alias: island = other
            free(other)
            free(alias)
    }
    return 0
`, "cannot use freed resource 'alias'")
}

func TestIslandFinalizationRejectsInterproceduralEnumPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MoveMsg:
    case take(island)

func unwrap(msg: MoveMsg) -> island:
    match msg:
    case MoveMsg.take(handle):
        return handle

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: MoveMsg = MoveMsg.take(core.island_new(16))
        let other: island = unwrap(msg)
        match msg:
        case MoveMsg.take(handle):
            free(handle)
            free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleEnumPayloadAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

enum MoveMsg:
    case take(island)

func unwrap(msg: MoveMsg) -> island:
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
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsWholeOptionalUseAfterPayloadFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let maybe: island? = core.island_new(16)
        if let handle = maybe:
            free(handle)
        return use(maybe)
    }
`, "ambiguous resource provenance for 'maybe.$elem' after control-flow merge")
}

func TestIslandFinalizationRejectsCrossModuleWholeOptionalUseAfterPayloadFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub func pass(maybe: island?) -> island?:
    return maybe
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = resources.pass(maybe)
        match returned:
        case some(other):
            free(other)
            return use(returned)
        case none:
            return 0
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'returned.$elem'")
}

func TestIslandFinalizationRejectsCrossModuleWholeOptionalIfLetUseAfterPayloadFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub func pass(maybe: island?) -> island?:
    return maybe
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = resources.pass(maybe)
        if let other = returned:
            free(other)
        return use(returned)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'returned.$elem'")
}

func TestIslandFinalizationRejectsClosureCaptureBeforeFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let cb: fn(Int) -> Int = fn(x: Int) -> Int:
            let alias: island = isl
            return x
        free(isl)
    }
    return 0
`, "function-typed storage 'cb' captures unsupported local 'isl' of type 'island'")
}

func TestIslandFinalizationRejectsInterproceduralOptionalPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func pass(maybe: island?) -> island?:
    return maybe

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = pass(maybe)
        if let other = returned:
            free(isl)
            free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsInterproceduralOptionalMatchPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func pass(maybe: island?) -> island?:
    return maybe

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = pass(maybe)
        match returned:
        case some(other):
            free(isl)
            free(other)
        case none:
            return 0
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleOptionalPayloadAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func pass(maybe: island?) -> island?:
    return maybe
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = resources.pass(maybe)
        if let other = returned:
            free(isl)
            free(other)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleOptionalMatchPayloadAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func pass(maybe: island?) -> island?:
    return maybe
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = resources.pass(maybe)
        match returned:
        case some(other):
            free(isl)
            free(other)
        case none:
            return 0
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsInterproceduralOptionalWrappedReturnAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func wrap(isl: island) -> island?:
    return isl

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let returned: island? = wrap(isl)
        if let other = returned:
            free(isl)
            free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleOptionalWrappedReturnAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func wrap(isl: island) -> island?:
    return isl
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let returned: island? = resources.wrap(isl)
        if let other = returned:
            free(isl)
            free(other)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsInterproceduralStructOptionalPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct MaybeBox:
    maybe: island?

func pass(box: MaybeBox) -> MaybeBox:
    return box

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let box: MaybeBox = MaybeBox(maybe: isl)
        let returned: MaybeBox = pass(box)
        if let other = returned.maybe:
            free(isl)
            free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleStructOptionalPayloadAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub struct MaybeBox:
    maybe: island?

pub func pass(box: MaybeBox) -> MaybeBox:
    return box
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let box: resources.MaybeBox = resources.MaybeBox(maybe: isl)
        let returned: resources.MaybeBox = resources.pass(box)
        if let other = returned.maybe:
            free(isl)
            free(other)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsInterproceduralEnumOptionalPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MaybeEnvelope:
    case wrap(island?)
    case empty

func pass(msg: MaybeEnvelope) -> MaybeEnvelope:
    return msg

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let msg: MaybeEnvelope = MaybeEnvelope.wrap(isl)
        let returned: MaybeEnvelope = pass(msg)
        match returned:
        case MaybeEnvelope.wrap(maybe):
            if let other = maybe:
                free(isl)
                free(other)
        case MaybeEnvelope.empty:
            return 0
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleEnumOptionalPayloadAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub enum MaybeEnvelope:
    case wrap(island?)
    case empty

pub func pass(msg: MaybeEnvelope) -> MaybeEnvelope:
    return msg
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let msg: resources.MaybeEnvelope = resources.MaybeEnvelope.wrap(isl)
        let returned: resources.MaybeEnvelope = resources.pass(msg)
        match returned:
        case resources.MaybeEnvelope.wrap(maybe):
            if let other = maybe:
                free(isl)
                free(other)
        case resources.MaybeEnvelope.empty:
            return 0
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestTaskGroupCancelReturnKeepsResourceProvenance(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = core.task_group_cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`, "cannot use closed resource 'canceled'")
}

func TestTaskGroupCancelWrapperReturnKeepsResourceProvenance(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func cancel(group: task.group) -> task.group
uses runtime:
    return core.task_group_cancel(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`, "cannot use closed resource 'canceled'")
}

func TestTaskGroupCancelCrossModuleWrapperReturnKeepsResourceProvenance(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func cancel(group: task.group) -> task.group
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use closed resource 'canceled'")
}

func TestTaskHandleFinalizationRejectsInterproceduralAliasJoin(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func alias_task(task: task.i32) -> task.i32:
    return task

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = alias_task(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`, "cannot use joined resource 'other'")
}

func TestActorConsumeRejectsInterproceduralAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 0

func alias_actor(peer: actor) -> actor:
    return peer

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let other: actor = alias_actor(peer)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`, "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsCrossModuleTransitiveInterproceduralAliasReuse(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsReuseAfterBranchConsume(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func flow(flag: Int) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    if flag:
        let taken: Int = take_actor(peer)
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return flow(1)
`, "cannot use consumed value 'peer'")
}

func TestActorConsumeRejectsReuseAfterLoopConsume(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func flow(limit: Int) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    var i: Int = 0
    while i < limit:
        let taken: Int = take_actor(peer)
        i = i + 1
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return flow(1)
`, "cannot use consumed value 'peer'")
}

func TestActorConsumeRejectsReuseAfterMatchConsume(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum Choice:
    case take
    case keep

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func flow(choice: Choice) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    match choice:
    case Choice.take:
        let taken: Int = take_actor(peer)
    case Choice.keep:
        let kept: Int = 0
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return flow(Choice.take)
`, "cannot use consumed value 'peer'")
}

func TestActorConsumeRejectsOptionalPayloadAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let maybe: actor? = peer
    if let other = maybe:
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`, "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsInterproceduralOptionalPayloadAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsOptionalMatchPayloadAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let maybe: actor? = peer
    match maybe:
    case some(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    case none:
        return 0
`, "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsInterproceduralOptionalMatchPayloadAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
    match returned:
    case some(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    case none:
        return 0
`, "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsCrossModuleOptionalPayloadAliasReuse(t *testing.T) {
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
    if let other = returned:
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsCrossModuleOptionalMatchPayloadAliasReuse(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsStructFieldAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct ActorBox:
    peer: actor

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: ActorBox = ActorBox(peer: peer)
    let _: Int = take_actor(peer)
    return core.send(box.peer, 1)
`, "cannot use consumed value 'box.peer'")
}

func TestActorConsumeRejectsInterproceduralStructFieldAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct ActorBox:
    peer: actor

func worker() -> Int:
    return 0

func unwrap(box: ActorBox) -> actor:
    return box.peer

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: ActorBox = ActorBox(peer: peer)
    let other: actor = unwrap(box)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`, "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsGenericStructFieldAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
`, "cannot use consumed value 'returned.value'")
}

func TestActorConsumeRejectsCrossModuleGenericStructFieldAliasReuse(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'returned.value'")
}

func TestActorConsumeRejectsCrossModuleStructFieldAliasReuse(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub struct ActorBox:
    peer: actor

pub func unwrap(box: ActorBox) -> actor:
    return box.peer
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
    let box: resources.ActorBox = resources.ActorBox(peer: peer)
    let other: actor = resources.unwrap(box)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsAggregateAliasFieldReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct ActorBox:
    peer: actor

func worker() -> Int:
    return 0

func take_box(box: consume ActorBox) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: ActorBox = ActorBox(peer: peer)
    let alias: ActorBox = box
    let _: Int = take_box(box)
    return core.send(alias.peer, 1)
`, "cannot use consumed value 'alias.peer'")
}

func TestActorConsumeRejectsAggregateAliasConsumedTwiceInSingleCall(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct ActorBox:
    peer: actor

func worker() -> Int:
    return 0

func take_two_boxes(first: consume ActorBox, second: consume ActorBox) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: ActorBox = ActorBox(peer: peer)
    let alias: ActorBox = box
    return take_two_boxes(box, alias)
`, "consumed more than once")
}

func TestTypedActorTransferRejectsFieldAccessEnumPayloadAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MoveMsg:
    case take(island)

struct Envelope:
    msg: MoveMsg

func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe {
        let msg: MoveMsg = MoveMsg.take(core.island_new(16))
        let envelope: Envelope = Envelope(msg: msg)
        let alias: MoveMsg = envelope.msg
        let _: Int = core.send_typed(peer, envelope.msg)
        match alias:
        case MoveMsg.take(isl):
            free(isl)
    }
    return 0
`, "cannot use consumed value")
}

func TestActorConsumeRejectsEnumPayloadAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MoveMsg:
    case handoff(actor)

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let msg: MoveMsg = MoveMsg.handoff(peer)
    match msg:
    case MoveMsg.handoff(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`, "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsCrossModuleEnumPayloadAliasReuse(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub enum MoveMsg:
    case handoff(actor)

pub func pass(msg: MoveMsg) -> MoveMsg:
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
    let msg: resources.MoveMsg = resources.MoveMsg.handoff(peer)
    let returned: resources.MoveMsg = resources.pass(msg)
    match returned:
    case resources.MoveMsg.handoff(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'other'")
}

func TestActorConsumeRejectsAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let alias: actor = peer
    let _: Int = take_actor(peer)
    return core.send(alias, 1)
`, "cannot use consumed value 'alias'")
}

func TestActorConsumeRejectsAliasConsumedTwiceInSingleCall(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 0

func take_two(first: consume actor, second: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let alias: actor = peer
    return take_two(peer, alias)
`, "consumed more than once")
}

func TestConsumeInThenBranchDoesNotPoisonElseBranch(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let value: Int = 1
    if 1:
        let _: Int = take(value)
    else:
        return value
    return 0
`)
}

func TestConsumeInBranchRejectsUseAfterMerge(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let value: Int = 1
    if 1:
        let _: Int = take(value)
    return value
`, "cannot use consumed value 'value'")
}

func TestConsumeInMatchExprArmDoesNotPoisonOtherArm(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
enum Choice:
    case left
    case right

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let value: Int = 1
    let choice: Choice = Choice.left
    let result: Int = match choice:
    case Choice.left:
        take(value)
    case Choice.right:
        value
    return result
`)
}

func TestConsumeInMatchExprRejectsUseAfterMerge(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum Choice:
    case left
    case right

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let value: Int = 1
    let choice: Choice = Choice.left
    let result: Int = match choice:
    case Choice.left:
        take(value)
    case Choice.right:
        value
    return result + value
`, "cannot use consumed value 'value'")
}

func TestMemoryBoundaryHandoffRejectsStaleIslandAfterResetAcrossActorBoundary(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MoveMsg:
    case take(island)

func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe {
        let isl: island = core.island_new(16)
        let next: island = core.island_reset(isl)
        let _: Int = core.send_typed(peer, MoveMsg.take(isl))
        free(next)
    }
    return 0
`, "cannot use consumed value 'isl'")
}

func TestMemoryBoundaryHandoffRejectsUnsafePointerAsSafeActorMessage(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum Msg:
    case raw(ptr)

func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, mem:
    let peer: actor = core.spawn("worker")
    unsafe {
        let raw: ptr = core.alloc_bytes(4)
        return core.send_typed(peer, Msg.raw(raw))
    }
`, "typed actor message payload must be value-only")
}
