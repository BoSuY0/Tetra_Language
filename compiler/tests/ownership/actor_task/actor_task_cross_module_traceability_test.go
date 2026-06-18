package compiler_test

import (
	"testing"

	"tetra_language/compiler/tests/ownership/testhelpers"
)

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
			testhelpers.RequireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.want)
		})
	}
}

func TestReleaseTraceabilityCrossModuleReturnedAggregateCallableMutableTargetBoundary(
	t *testing.T,
) {
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
			testhelpers.RequireCheckWorldFilesErrorContains(t, tt.files, "app/main.t4", tt.want)
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

			testhelpers.RequireCheckWorldFilesErrorContains(t, files, "app/main.t4", tt.want)
		})
	}
}
