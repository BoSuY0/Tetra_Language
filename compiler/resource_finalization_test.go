package compiler

import "testing"

func TestTaskHandleFinalizationRejectsDoubleJoin(t *testing.T) {
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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

func TestTaskHandleFinalizationJoinUntilDoesNotConsumeHandle(t *testing.T) {
	requireCheckFileOK(t, `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let first: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(5))
    return first.value + core.task_join_i32(task)
`)
}

func TestTaskGroupCloseStillAllowsStatus(t *testing.T) {
	requireCheckFileOK(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(group)
`, "cannot use closed resource 'group'")
}

func TestTaskGroupFinalizationAllowsReopenAssignmentAfterClose(t *testing.T) {
	requireCheckFileOK(t, `
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
	requireCheckFileErrorContains(t, `
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

func TestIslandFinalizationRejectsDoubleFree(t *testing.T) {
	requireCheckFileErrorContains(t, `
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

func TestIslandFinalizationRejectsAliasDoubleFree(t *testing.T) {
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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

func TestIslandFinalizationRejectsStructFieldFreeThenOriginalFree(t *testing.T) {
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileOK(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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

func TestIslandFinalizationRejectsBranchReturnedAliasDoubleFree(t *testing.T) {
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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

func TestTaskGroupCancelReturnKeepsResourceProvenance(t *testing.T) {
	requireCheckFileErrorContains(t, `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = core.task_group_cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`, "cannot use closed resource 'canceled'")
}

func TestTaskHandleFinalizationRejectsInterproceduralAliasJoin(t *testing.T) {
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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

func TestActorConsumeRejectsStructFieldAliasReuse(t *testing.T) {
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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

func TestActorConsumeRejectsAggregateAliasFieldReuse(t *testing.T) {
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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

func TestActorConsumeRejectsAliasReuse(t *testing.T) {
	requireCheckFileErrorContains(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileOK(t, `
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
	requireCheckFileErrorContains(t, `
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
	requireCheckFileOK(t, `
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
	requireCheckFileErrorContains(t, `
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
