package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use closed resource 'canceled'",
	)
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use consumed value 'other'",
	)
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use consumed value 'other'",
	)
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use consumed value 'other'",
	)
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use consumed value 'returned.value'",
	)
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use consumed value 'other'",
	)
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
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use consumed value 'other'",
	)
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
