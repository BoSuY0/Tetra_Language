package abisuite

func CheckX32SingleTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x32", "x32-task-runtime", "x32 task runtime", singleTaskRuntimeSmokeSource(), 0x3e, deps)
}

func CheckX32TypedTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x32", "x32-typed-task-runtime", "x32 typed-task runtime", typedTaskRuntimeSmokeSource(), 0x3e, deps)
}

func CheckX32StagedTypedTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x32", "x32-staged-typed-task-runtime", "x32 staged typed-task runtime", stagedTypedTaskRuntimeSmokeSource(), 0x3e, deps)
}

func CheckX32TaskGroupSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x32", "x32-task-group-runtime", "x32 task-group runtime", taskGroupRuntimeSmokeSource(), 0x3e, deps)
}

func CheckX32TypedTaskGroupSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x32", "x32-typed-task-group-runtime", "x32 typed-task-group runtime", typedTaskGroupRuntimeSmokeSource(), 0x3e, deps)
}

func CheckX32SingleActorSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x32", "x32-actor-runtime", "x32 actor runtime", singleActorRuntimeSmokeSource(), 0x3e, deps)
}

func CheckX32ActorStateSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x32", "x32-actor-state-runtime", "x32 actor-state runtime", actorStateRuntimeSmokeSource(), 0x3e, deps)
}

func CheckX86SingleTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x86", "x86-task-runtime", "x86 task runtime", singleTaskRuntimeSmokeSource(), 0x03, deps)
}

func CheckX86TypedTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x86", "x86-typed-task-runtime", "x86 typed-task runtime", typedTaskRuntimeSmokeSource(), 0x03, deps)
}

func CheckX86StagedTypedTaskSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x86", "x86-staged-typed-task-runtime", "x86 staged typed-task runtime", stagedTypedTaskRuntimeSmokeSource(), 0x03, deps)
}

func CheckX86TaskGroupSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x86", "x86-task-group-runtime", "x86 task-group runtime", taskGroupRuntimeSmokeSource(), 0x03, deps)
}

func CheckX86TypedTaskGroupSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x86", "x86-typed-task-group-runtime", "x86 typed-task-group runtime", typedTaskGroupRuntimeSmokeSource(), 0x03, deps)
}

func CheckX86SingleActorSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x86", "x86-actor-runtime", "x86 actor runtime", singleActorRuntimeSmokeSource(), 0x03, deps)
}

func CheckX86ActorStateSelfHostRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkSelfHostRuntimeSmoke("linux-x86", "x86-actor-state-runtime", "x86 actor-state runtime", actorStateRuntimeSmokeSource(), 0x03, deps)
}

func checkSelfHostRuntimeSmoke(target string, stem string, label string, src string, wantMachine uint16, deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      target,
		stem:        stem,
		label:       label,
		src:         src,
		wantClass:   1,
		wantMachine: wantMachine,
	}, deps)
}

func singleTaskRuntimeSmokeSource() string {
	return `
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`
}

func typedTaskRuntimeSmokeSource() string {
	return `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`
}

func stagedTypedTaskRuntimeSmokeSource() string {
	return `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
    case TaskErr.stopped:
        99
`
}

func taskGroupRuntimeSmokeSource() string {
	return `
func worker() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        return 60 + status
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    if result.error != 0:
        return 100 + result.error
    return result.value
`
}

func typedTaskGroupRuntimeSmokeSource() string {
	return `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        throw TaskErr.boom(60 + status)
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    let result: Int = catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    return result
`
}

func singleActorRuntimeSmokeSource() string {
	return `
func worker() -> Int
uses actors:
    let value: Int = core.recv()
    if value == 41:
        let _sent: Int = core.send(core.sender(), 42)
        return 0
    return 1

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _sent: Int = core.send(peer, 41)
    let reply: Int = core.recv()
    if reply == 42:
        return 0
    return reply
`
}

func actorStateRuntimeSmokeSource() string {
	return `
actor Counter:
    var count: Int = 0
    const enabled: Bool = true
    func run() -> Int
    uses actors:
        let delta: Int = core.recv()
        if enabled:
            count = count + delta + 1
        let _sent: Int = core.send(core.sender(), count)
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Counter.run")
    let _sent: Int = core.send(peer, 41)
    return core.recv()
`
}
