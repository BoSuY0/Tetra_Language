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

func requireCheckWorldFilesOK(t *testing.T, files map[string]string, entry string) {
	t.Helper()

	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, files)

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash(entry)))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestActorSpawnOwnershipMatrix(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "allows_synchronous_i32_target",
			src: `
func worker() -> Int:
    return 7

func main() -> Int
uses actors:
    let a: actor = core.spawn("worker")
    return 0
`,
		},
		{
			name: "rejects_async_target",
			src: `
async func worker() -> Int:
    return 7

func main() -> Int
uses actors:
    let a: actor = core.spawn("worker")
    return 0
`,
			wantErr: "spawn target must be synchronous",
		},
		{
			name: "rejects_invalid_target_shape",
			src: `
func worker(x: Int) -> Int:
    return x

func main() -> Int
uses actors:
    let a: actor = core.spawn("worker")
    return 0
`,
			wantErr: "spawn target must have shape",
		},
		{
			name: "rejects_non_literal_target_name",
			src: `
func worker() -> Int:
    return 7

func main() -> Int
uses actors:
    let name: str = "worker"
    let a: actor = core.spawn(name)
    return 0
`,
			wantErr: "spawn expects a string literal",
		},
		{
			name: "rejects_empty_target_name",
			src: `
func main() -> Int
uses actors:
    let a: actor = core.spawn("")
    return 0
`,
			wantErr: "spawn expects a non-empty name",
		},
		{
			name: "rejects_builtin_target",
			src: `
func main() -> Int
uses actors:
    let a: actor = core.spawn("core.recv")
    return 0
`,
			wantErr: "spawn target must be a user function",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantErr == "" {
				testkit.RequireFileCheckOK(t, tc.src)
				return
			}
			testkit.RequireFileCheckErrorContains(t, tc.src, tc.wantErr)
		})
	}
}

func TestTaskSpawnOwnershipMatrix(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "rejects_throwing_target",
			src: `
enum SpawnErr:
    case boom

func worker() -> Int throws SpawnErr:
    return 0

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			wantErr: "task_spawn_i32 target must not throw",
		},
		{
			name: "rejects_non_literal_target_name",
			src: `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let name: str = "worker"
    let task: task.i32 = core.task_spawn_i32(name)
    return core.task_join_i32(task)
`,
			wantErr: "task_spawn_i32 expects a string literal",
		},
		{
			name: "rejects_empty_target_name",
			src: `
func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("")
    return core.task_join_i32(task)
`,
			wantErr: "task_spawn_i32 expects a non-empty name",
		},
		{
			name: "rejects_builtin_target",
			src: `
func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("core.recv")
    return core.task_join_i32(task)
`,
			wantErr: "task_spawn_i32 target must be a user function",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testkit.RequireFileCheckErrorContains(t, tc.src, tc.wantErr)
		})
	}
}

func TestTaskSpawnGroupOwnershipMatrix(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "rejects_non_literal_target_name",
			src: `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let name: str = "worker"
    let task: task.i32 = core.task_spawn_group_i32(group, name)
    return core.task_join_i32(task)
`,
			wantErr: "task_spawn_group_i32 expects a string literal worker name",
		},
		{
			name: "rejects_empty_target_name",
			src: `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "")
    return core.task_join_i32(task)
`,
			wantErr: "task_spawn_group_i32 expects a non-empty name",
		},
		{
			name: "rejects_builtin_target",
			src: `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "core.recv")
    return core.task_join_i32(task)
`,
			wantErr: "task_spawn_group_i32 target must be a user function",
		},
		{
			name: "rejects_invalid_target_shape",
			src: `
func worker(x: Int) -> Int:
    return x

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			wantErr: "task_spawn_group_i32 target must have shape",
		},
		{
			name: "rejects_async_target",
			src: `
async func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			wantErr: "task_spawn_group_i32 target must be synchronous",
		},
		{
			name: "rejects_throwing_target",
			src: `
enum SpawnErr:
    case boom

func worker() -> Int throws SpawnErr:
    return 0

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			wantErr: "task_spawn_group_i32 target must not throw",
		},
		{
			name: "rejects_target_touching_mutable_global_state",
			src: `
var g: Int

func worker() -> Int:
    g = g + 1
    return g

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			wantErr: "touches mutable global state",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testkit.RequireFileCheckErrorContains(t, tc.src, tc.wantErr)
		})
	}
}

func TestActorAndTaskTransfersCannotBeReusedAfterConsume(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "task",
			src: `
func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = take_task(task)
    return value + core.task_join_i32(task)
`,
			want: "cannot use consumed value 'task'",
		},
		{
			name: "actor",
			src: `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _: Int = take_actor(peer)
    return core.send(peer, 1)
`,
			want: "cannot use consumed value 'peer'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireFileCheckErrorContains(t, tt.src, tt.want)
		})
	}
}

func TestTaskConsumeRejectsOptionalMatchPayloadAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    match maybe:
    case some(other):
        let first: Int = take_task(task)
        return first + core.task_join_i32(other)
    case none:
        return 0
`, "cannot use consumed value 'other'")
}

func TestTaskConsumeRejectsInterproceduralOptionalMatchPayloadAliasReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
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
    match returned:
    case some(other):
        let first: Int = take_task(task)
        return first + core.task_join_i32(other)
    case none:
        return 0
`, "cannot use consumed value 'other'")
}

func TestTaskConsumeRejectsCrossModuleOptionalMatchPayloadAliasReuse(t *testing.T) {
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
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use consumed value 'other'")
}

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

func TestReleaseTraceabilityCrossModuleCallableMutableTargetBoundary(t *testing.T) {
	lib := `
module lib.callbacks

var g: Int

pub func safe() -> Int:
    return 1

pub func inc() -> Int:
    g = g + 1
    return g

pub func pick(flag: Bool) -> fn() -> Int:
    if flag:
        return safe
    return inc

pub func choose() -> fn() -> Int:
    let f: fn() -> Int = pick(false)
    return f
`

	tests := []struct {
		name string
		app  string
		want string
	}{
		{
			name: "task_return_call_callback_argument",
			app: `
module app.main
import lib.callbacks as callbacks

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(callbacks.choose())

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "typed_task_return_call_callback_argument",
			app: `
module app.main
import lib.callbacks as callbacks

enum TaskErr:
    case failed

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int throws TaskErr:
    return apply(callbacks.choose())

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        0
`,
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "task_group_return_call_callback_argument",
			app: `
module app.main
import lib.callbacks as callbacks

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(callbacks.choose())

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "typed_task_group_return_call_callback_argument",
			app: `
module app.main
import lib.callbacks as callbacks

enum TaskErr:
    case failed

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int throws TaskErr:
    return apply(callbacks.choose())

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    return catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        0
`,
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "actor_return_call_callback_argument",
			app: `
module app.main
import lib.callbacks as callbacks

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(callbacks.choose())

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`,
			want: "touches mutable global state and cannot cross actor boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/callbacks.t4": lib,
				"app/main.t4":      tt.app,
			}
			requireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.want)
		})
	}
}

func TestReleaseTraceabilityCrossModuleReturnedAggregateCallableMutableTargetBoundary(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name: "task_struct_field_direct_call",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub struct Holder:
    cb: fn() -> Int

var g: Int

pub func safe() -> Int:
    return 1

pub func inc() -> Int:
    g = g + 1
    return g

pub func makeHolder(flag: Bool) -> Holder:
    if flag:
        return Holder(cb: safe)
    return Holder(cb: inc)
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

func worker() -> Int:
    let holder: callbacks.Holder = callbacks.makeHolder(false)
    return holder.cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			},
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "task_enum_payload_callback_argument",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub enum Choice:
    case some(fn() -> Int)

var g: Int

pub func safe() -> Int:
    return 1

pub func inc() -> Int:
    g = g + 1
    return g

pub func makeChoice(flag: Bool) -> Choice:
    if flag:
        return Choice.some(safe)
    return Choice.some(inc)
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    let choice: callbacks.Choice = callbacks.makeChoice(false)
    match choice:
        case callbacks.Choice.some(f):
            return apply(f)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			},
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "task_returned_enum_payload_direct_closure",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub enum Choice:
    case some(fn() -> Int)

var g: Int

pub func makeChoice() -> Choice:
    return Choice.some(fn() -> Int:
        g = g + 1
        return g
    )
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

func worker() -> Int:
    let choice: callbacks.Choice = callbacks.makeChoice()
    match choice:
        case callbacks.Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			},
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "typed_task_returned_enum_payload_direct_closure",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub enum Choice:
    case some(fn() -> Int)

var g: Int

pub func makeChoice() -> Choice:
    return Choice.some(fn() -> Int:
        g = g + 1
        return g
    )
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

enum TaskErr:
    case failed

func worker() -> Int throws TaskErr:
    let choice: callbacks.Choice = callbacks.makeChoice()
    match choice:
        case callbacks.Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        0
`,
			},
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "task_group_returned_enum_payload_direct_closure",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub enum Choice:
    case some(fn() -> Int)

var g: Int

pub func makeChoice() -> Choice:
    return Choice.some(fn() -> Int:
        g = g + 1
        return g
    )
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

func worker() -> Int:
    let choice: callbacks.Choice = callbacks.makeChoice()
    match choice:
        case callbacks.Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			},
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "typed_task_group_returned_enum_payload_direct_closure",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub enum Choice:
    case some(fn() -> Int)

var g: Int

pub func makeChoice() -> Choice:
    return Choice.some(fn() -> Int:
        g = g + 1
        return g
    )
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

enum TaskErr:
    case failed

func worker() -> Int throws TaskErr:
    let choice: callbacks.Choice = callbacks.makeChoice()
    match choice:
        case callbacks.Choice.some(f):
            return f()

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    return catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        0
`,
			},
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "actor_returned_enum_payload_direct_closure",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub enum Choice:
    case some(fn() -> Int)

var g: Int

pub func makeChoice() -> Choice:
    return Choice.some(fn() -> Int:
        g = g + 1
        return g
    )
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

func worker() -> Int:
    let choice: callbacks.Choice = callbacks.makeChoice()
    match choice:
        case callbacks.Choice.some(f):
            return f()

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`,
			},
			want: "touches mutable global state and cannot cross actor boundary",
		},
		{
			name: "actor_struct_field_direct_call",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub struct Holder:
    cb: fn() -> Int

var g: Int

pub func safe() -> Int:
    return 1

pub func inc() -> Int:
    g = g + 1
    return g

pub func makeHolder(flag: Bool) -> Holder:
    if flag:
        return Holder(cb: safe)
    return Holder(cb: inc)
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

func worker() -> Int:
    let holder: callbacks.Holder = callbacks.makeHolder(false)
    return holder.cb()

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`,
			},
			want: "touches mutable global state and cannot cross actor boundary",
		},
		{
			name: "actor_enum_payload_callback_argument",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub enum Choice:
    case some(fn() -> Int)

var g: Int

pub func safe() -> Int:
    return 1

pub func inc() -> Int:
    g = g + 1
    return g

pub func makeChoice(flag: Bool) -> Choice:
    if flag:
        return Choice.some(safe)
    return Choice.some(inc)
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    let choice: callbacks.Choice = callbacks.makeChoice(false)
    match choice:
        case callbacks.Choice.some(f):
            return apply(f)

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`,
			},
			want: "touches mutable global state and cannot cross actor boundary",
		},
		{
			name: "typed_task_struct_field_direct_call",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub struct Holder:
    cb: fn() -> Int

var g: Int

pub func safe() -> Int:
    return 1

pub func inc() -> Int:
    g = g + 1
    return g

pub func makeHolder(flag: Bool) -> Holder:
    if flag:
        return Holder(cb: safe)
    return Holder(cb: inc)
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

enum TaskErr:
    case failed

func worker() -> Int throws TaskErr:
    let holder: callbacks.Holder = callbacks.makeHolder(false)
    return holder.cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        0
`,
			},
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "typed_task_group_struct_field_direct_call",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub struct Holder:
    cb: fn() -> Int

var g: Int

pub func safe() -> Int:
    return 1

pub func inc() -> Int:
    g = g + 1
    return g

pub func makeHolder(flag: Bool) -> Holder:
    if flag:
        return Holder(cb: safe)
    return Holder(cb: inc)
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

enum TaskErr:
    case failed

func worker() -> Int throws TaskErr:
    let holder: callbacks.Holder = callbacks.makeHolder(false)
    return holder.cb()

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    return catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        0
`,
			},
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "task_group_enum_payload_callback_argument",
			files: map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

pub enum Choice:
    case some(fn() -> Int)

var g: Int

pub func safe() -> Int:
    return 1

pub func inc() -> Int:
    g = g + 1
    return g

pub func makeChoice(flag: Bool) -> Choice:
    if flag:
        return Choice.some(safe)
    return Choice.some(inc)
`,
				"app/main.t4": `
module app.main
import lib.callbacks as callbacks

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    let choice: callbacks.Choice = callbacks.makeChoice(false)
    match choice:
        case callbacks.Choice.some(f):
            return apply(f)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			},
			want: "touches mutable global state and cannot cross task boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireCheckWorldFilesErrorContains(t, tt.files, "app/main.t4", tt.want)
		})
	}
}

func TestReleaseTraceabilityCrossModuleImmutableCallableGlobalMutableTargetBoundary(t *testing.T) {
	tests := []struct {
		name string
		app  string
		want string
	}{
		{
			name: "task_direct_call",
			app: `
module app.main
import lib.callbacks as callbacks

func worker() -> Int:
    return callbacks.cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "task_callback_argument",
			app: `
module app.main
import lib.callbacks as callbacks

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(callbacks.cb)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "typed_task_direct_call",
			app: `
module app.main
import lib.callbacks as callbacks

enum TaskErr:
    case failed

func worker() -> Int throws TaskErr:
    return callbacks.cb()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        0
`,
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "task_group_callback_argument",
			app: `
module app.main
import lib.callbacks as callbacks

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(callbacks.cb)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`,
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "typed_task_group_callback_argument",
			app: `
module app.main
import lib.callbacks as callbacks

enum TaskErr:
    case failed

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int throws TaskErr:
    return apply(callbacks.cb)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    return catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        0
`,
			want: "touches mutable global state and cannot cross task boundary",
		},
		{
			name: "actor_direct_call",
			app: `
module app.main
import lib.callbacks as callbacks

func worker() -> Int:
    return callbacks.cb()

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`,
			want: "touches mutable global state and cannot cross actor boundary",
		},
		{
			name: "actor_callback_argument",
			app: `
module app.main
import lib.callbacks as callbacks

func apply(f: fn() -> Int) -> Int:
    return f()

func worker() -> Int:
    return apply(callbacks.cb)

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`,
			want: "touches mutable global state and cannot cross actor boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/callbacks.t4": `
module lib.callbacks

var g: Int
pub val cb: fn() -> Int = inc

pub func inc() -> Int:
    g = g + 1
    return g
`,
				"app/main.t4": tt.app,
			}

			requireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.want)
		})
	}
}
