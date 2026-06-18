package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "LifetimeRejectsTaskUseAfterJoin",
			src: `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(task)
`,
			want: "cannot use joined resource 'task'",
		},
		{
			name: "LifetimeRejectsTaskGroupSpawnAfterClose",
			src: `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let _: Int = core.task_group_close(group)
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			want: "cannot use closed resource 'group'",
		},
		{
			name: "RaceSafetyRejectsActorTargetMutableGlobal",
			src: `
var g: Int

func worker() -> Int:
    g = g + 1
    return g

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`,
			want: "touches mutable global state",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableGlobal",
			src: `
var g: Int

func worker() -> Int:
    g = g + 1
    return g

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "touches mutable global state",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobal",
			src: `
var cb: fn() -> Int = one

func one() -> Int:
    return 1

func worker() -> Int:
    return cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobalCallbackArgument",
			src: `
var cb: fn() -> Int = one

func one() -> Int:
    return 1

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(cb)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobalMutableLocalReassignment",
			src: `
var cb: fn() -> Int = one

func one() -> Int:
    return 1

func two() -> Int:
    return 2

func worker() -> Int:
    var f: fn() -> Int = two
    f = cb
    return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobalWrite",
			src: `
var cb: fn() -> Int = one

func one() -> Int:
    return 1

func two() -> Int:
    return 2

func worker() -> Int:
    cb = two
    return cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobalStructField",
			src: `
struct Holder:
    cb: fn() -> Int

var cb: fn() -> Int = one

func one() -> Int:
    return 1

func worker() -> Int:
    let holder: Holder = Holder(cb: cb)
    return holder.cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobalStructFieldReassignment",
			src: `
struct Holder:
    cb: fn() -> Int

var cb: fn() -> Int = one

func one() -> Int:
    return 1

func two() -> Int:
    return 2

func worker() -> Int:
    var holder: Holder = Holder(cb: two)
    holder.cb = cb
    return holder.cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobalEnumPayload",
			src: `
enum Choice:
    case some(fn() -> Int)

var cb: fn() -> Int = one

func one() -> Int:
    return 1

func worker() -> Int:
    let choice: Choice = Choice.some(cb)
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobalEnumPayloadReassignment",
			src: `
enum Choice:
    case some(fn() -> Int)

var cb: fn() -> Int = one

func one() -> Int:
    return 1

func two() -> Int:
    return 2

func worker() -> Int:
    var choice: Choice = Choice.some(two)
    choice = Choice.some(cb)
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobalReturn",
			src: `
var cb: fn() -> Int = one

func one() -> Int:
    return 1

func pick() -> fn() -> Int:
    return cb

func worker() -> Int:
    let f: fn() -> Int = pick()
    return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobalReturnedStructField",
			src: `
struct Holder:
    cb: fn() -> Int

var cb: fn() -> Int = one

func one() -> Int:
    return 1

func makeHolder() -> Holder:
    return Holder(cb: cb)

func worker() -> Int:
    let holder: Holder = makeHolder()
    return holder.cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetMutableFunctionTypedGlobalReturnedEnumPayload",
			src: `
enum Choice:
    case some(fn() -> Int)

var cb: fn() -> Int = one

func one() -> Int:
    return 1

func makeChoice() -> Choice:
    return Choice.some(cb)

func worker() -> Int:
    let choice: Choice = makeChoice()
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetImmutableFunctionTypedGlobalWithMutableTarget",
			src: `
var g: Int
val cb: fn() -> Int = inc

func inc() -> Int:
    g = g + 1
    return g

func worker() -> Int:
    return cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetImmutableFunctionTypedGlobalCallbackWithMutableTarget",
			src: `
var g: Int
val cb: fn() -> Int = inc

func inc() -> Int:
    g = g + 1
    return g

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(cb)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedLocalCallbackWithMutableTarget",
			src: `
var g: Int

func inc() -> Int:
    g = g + 1
    return g

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    let f: fn() -> Int = inc
    return apply(f)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedReturnCallCallbackWithMutableTarget",
			src: `
var g: Int
val cb: fn() -> Int = inc

func inc() -> Int:
    g = g + 1
    return g

func pick() -> fn() -> Int:
    return cb

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(pick())

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedStructFieldReturnCallCallbackWithMutableTarget",
			src: `
struct Holder:
    cb: fn() -> Int

var g: Int

func inc() -> Int:
    g = g + 1
    return g

func pick(holder: Holder) -> fn() -> Int:
    return holder.cb

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    let holder: Holder = Holder(cb: inc)
    return apply(pick(holder))

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedEnumPayloadReturnCallCallbackWithMutableTarget",
			src: `
enum Choice:
    case some(fn() -> Int)

var g: Int

func inc() -> Int:
    g = g + 1
    return g

func pick(choice: Choice) -> fn() -> Int:
    match choice:
        case Choice.some(f):
            return f

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    let choice: Choice = Choice.some(inc)
    return apply(pick(choice))

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedMultiReturnCallCallbackWithMutableTarget",
			src: `
var g: Int

func safe() -> Int:
    return 1

func inc() -> Int:
    g = g + 1
    return g

func pick(flag: Bool) -> fn() -> Int:
    if flag:
        return safe
    return inc

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(pick(false))

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedMultiReturnAliasCallbackWithMutableTarget",
			src: `
var g: Int

func safe() -> Int:
    return 1

func inc() -> Int:
    g = g + 1
    return g

func pick(flag: Bool) -> fn() -> Int:
    if flag:
        return safe
    return inc

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    let f: fn() -> Int = pick(false)
    return apply(f)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedMultiReturnAliasReturnedCallbackWithMutableTarget",
			src: `
var g: Int

func safe() -> Int:
    return 1

func inc() -> Int:
    g = g + 1
    return g

func pick(flag: Bool) -> fn() -> Int:
    if flag:
        return safe
    return inc

func choose() -> fn() -> Int:
    let f: fn() -> Int = pick(false)
    return f

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(choose())

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedMultiReturnStructFieldWithMutableTarget",
			src: `
struct Holder:
    cb: fn() -> Int

var g: Int

func safe() -> Int:
    return 1

func inc() -> Int:
    g = g + 1
    return g

func pick(flag: Bool) -> fn() -> Int:
    if flag:
        return safe
    return inc

func worker() -> Int:
    let holder: Holder = Holder(cb: pick(false))
    return holder.cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedMultiReturnEnumPayloadWithMutableTarget",
			src: `
enum Choice:
    case some(fn() -> Int)

var g: Int

func safe() -> Int:
    return 1

func inc() -> Int:
    g = g + 1
    return g

func pick(flag: Bool) -> fn() -> Int:
    if flag:
        return safe
    return inc

func worker() -> Int:
    let choice: Choice = Choice.some(pick(false))
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetReturnedStructMultiTargetFieldWithMutableTarget",
			src: `
struct Holder:
    cb: fn() -> Int

var g: Int

func safe() -> Int:
    return 1

func inc() -> Int:
    g = g + 1
    return g

func makeHolder(flag: Bool) -> Holder:
    if flag:
        return Holder(cb: safe)
    return Holder(cb: inc)

func worker() -> Int:
    let holder: Holder = makeHolder(false)
    return holder.cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetReturnedEnumMultiTargetPayloadWithMutableTarget",
			src: `
enum Choice:
    case some(fn() -> Int)

var g: Int

func safe() -> Int:
    return 1

func inc() -> Int:
    g = g + 1
    return g

func makeChoice(flag: Bool) -> Choice:
    if flag:
        return Choice.some(safe)
    return Choice.some(inc)

func worker() -> Int:
    let choice: Choice = makeChoice(false)
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetReturnedEnumPayloadDirectClosureWithMutableGlobalTarget",
			src: `
enum Choice:
    case some(fn() -> Int)

var g: Int

func makeChoice() -> Choice:
    return Choice.some(fn() -> Int:
        g = g + 1
        return g
    )

func worker() -> Int:
    let choice: Choice = makeChoice()
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTypedTaskTargetReturnedEnumPayloadDirectClosureWithMutableGlobalTarget",
			src: `
enum TaskErr:
    case failed

enum Choice:
    case some(fn() -> Int)

var g: Int

func makeChoice() -> Choice:
    return Choice.some(fn() -> Int:
        g = g + 1
        return g
    )

func worker() -> Int throws TaskErr:
    let choice: Choice = makeChoice()
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        0
`,
			want: "task_spawn_i32_typed target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskGroupTargetReturnedEnumPayloadDirectClosureWithMutableGlobalTarget",
			src: `
enum Choice:
    case some(fn() -> Int)

var g: Int

func makeChoice() -> Choice:
    return Choice.some(fn() -> Int:
        g = g + 1
        return g
    )

func worker() -> Int:
    let choice: Choice = makeChoice()
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_group_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTypedTaskGroupTargetReturnedEnumPayloadDirectClosureWithMutableGlobalTarget",
			src: `
enum TaskErr:
    case failed

enum Choice:
    case some(fn() -> Int)

var g: Int

func makeChoice() -> Choice:
    return Choice.some(fn() -> Int:
        g = g + 1
        return g
    )

func worker() -> Int throws TaskErr:
    let choice: Choice = makeChoice()
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    return catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        0
`,
			want: "task_spawn_group_i32_typed target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsActorTargetReturnedEnumPayloadDirectClosureWithMutableGlobalTarget",
			src: `
enum Choice:
    case some(fn() -> Int)

var g: Int

func makeChoice() -> Choice:
    return Choice.some(fn() -> Int:
        g = g + 1
        return g
    )

func worker() -> Int:
    let choice: Choice = makeChoice()
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`,
			want: "spawn target 'worker' touches mutable global state and cannot cross actor boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetReturnedStructMultiTargetFieldCallbackWithMutableTarget",
			src: `
struct Holder:
    cb: fn() -> Int

var g: Int

func safe() -> Int:
    return 1

func inc() -> Int:
    g = g + 1
    return g

func makeHolder(flag: Bool) -> Holder:
    if flag:
        return Holder(cb: safe)
    return Holder(cb: inc)

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    let holder: Holder = makeHolder(false)
    return apply(holder.cb)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetReturnedEnumMultiTargetPayloadCallbackWithMutableTarget",
			src: `
enum Choice:
    case some(fn() -> Int)

var g: Int

func safe() -> Int:
    return 1

func inc() -> Int:
    g = g + 1
    return g

func makeChoice(flag: Bool) -> Choice:
    if flag:
        return Choice.some(safe)
    return Choice.some(inc)

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    let choice: Choice = makeChoice(false)
    match choice:
        case Choice.some(f):
            return apply(f)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedMutableLocalDirectCallWithMutableTarget",
			src: `
var g: Int

func inc() -> Int:
    g = g + 1
    return g

func worker() -> Int:
    var f: fn() -> Int = inc
    return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedStructFieldDirectCallWithMutableTarget",
			src: `
struct Holder:
    cb: fn() -> Int

var g: Int

func inc() -> Int:
    g = g + 1
    return g

func worker() -> Int:
    let holder: Holder = Holder(cb: inc)
    return holder.cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskTargetFunctionTypedEnumPayloadDirectCallWithMutableTarget",
			src: `
enum Choice:
    case some(fn() -> Int)

var g: Int

func inc() -> Int:
    g = g + 1
    return g

func worker() -> Int:
    let choice: Choice = Choice.some(inc)
    match choice:
        case Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTypedTaskTargetMutableFunctionTypedGlobal",
			src: `
enum TaskErr:
    case failed

var cb: fn() -> Int = one

func one() -> Int:
    return 1

func worker() -> Int throws TaskErr:
    return cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return core.task_join_i32_typed<TaskErr>(task)
`,
			want: "task_spawn_i32_typed target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsTaskGroupTargetMutableFunctionTypedGlobal",
			src: `
var cb: fn() -> Int = one

func one() -> Int:
    return 1

func worker() -> Int:
    return cb()

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_group_i32 target 'worker' touches mutable global state and cannot cross task boundary",
		},
		{
			name: "RaceSafetyRejectsActorTargetMutableFunctionTypedGlobal",
			src: `
var cb: fn() -> Int = one

func one() -> Int:
    return 1

func worker() -> Int:
    return cb()

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`,
			want: "spawn target 'worker' touches mutable global state and cannot cross actor boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireFileCheckErrorContains(t, tt.src, tt.want)
		})
	}
}
