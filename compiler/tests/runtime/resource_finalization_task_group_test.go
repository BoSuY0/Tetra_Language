package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

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
