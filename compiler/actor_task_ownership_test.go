package compiler

import "testing"

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
				requireCheckFileOK(t, tc.src)
				return
			}
			requireCheckFileErrorContains(t, tc.src, tc.wantErr)
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
			requireCheckFileErrorContains(t, tc.src, tc.wantErr)
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
			requireCheckFileErrorContains(t, tc.src, tc.wantErr)
		})
	}
}
