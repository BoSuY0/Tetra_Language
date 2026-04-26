package compiler

import (
	"strings"
	"testing"
)

func requireCheckFileErrorContains(t *testing.T, src string, want string) {
	t.Helper()
	file, err := ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: file.Module,
		Files:       []*FileAST{file},
		ByModule:    map[string]*FileAST{file.Module: file},
	}
	_, err = CheckWorld(world)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func requireCheckFileOK(t *testing.T, src string) {
	t.Helper()
	file, err := ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: file.Module,
		Files:       []*FileAST{file},
		ByModule:    map[string]*FileAST{file.Module: file},
	}
	if _, err := CheckWorld(world); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestAsyncParseCheckAndLower(t *testing.T) {
	src := []byte(`
async func answer() -> Int:
    return 42

async func caller() -> Int:
    let value: Int = await answer()
    return value

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !prog.Funcs[0].Async {
		t.Fatalf("expected async function")
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !checked.FuncSigs["answer"].Async {
		t.Fatalf("expected async signature")
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestAsyncRejectAwaitOutsideAsync(t *testing.T) {
	src := []byte(`
async func answer() -> Int:
    return 42

func main() -> Int:
    return await answer()
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected await context error")
	}
	if !strings.Contains(err.Error(), "await is only allowed in async functions") {
		t.Fatalf("error = %v", err)
	}
}

func TestAsyncRejectBareAsyncCall(t *testing.T) {
	src := []byte(`
async func answer() -> Int:
    return 42

async func caller() -> Int:
    return answer()

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected bare async call error")
	}
	if !strings.Contains(err.Error(), "requires await") {
		t.Fatalf("error = %v", err)
	}
}

func TestTaskSpawnJoinCheckAndLower(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTaskSpawnRequiresRuntimeUse(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 42

func main() -> task.i32:
    return core.task_spawn_i32("worker")
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected missing runtime effect error")
	}
	if !strings.Contains(err.Error(), "uses effect 'runtime'") {
		t.Fatalf("error = %v", err)
	}
}

func TestTaskSpawnRejectsInvalidTargetShape(t *testing.T) {
	src := []byte(`
func worker(x: Int) -> Int:
    return x

func main() -> task.i32
uses runtime:
    return core.task_spawn_i32("worker")
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected invalid task target shape error")
	}
	if !strings.Contains(err.Error(), "task_spawn_i32 target must have shape func worker() -> i32") {
		t.Fatalf("error = %v", err)
	}
}

func TestTaskSpawnRejectsAsyncTarget(t *testing.T) {
	src := []byte(`
async func worker() -> Int:
    return 42

func main() -> task.i32
uses runtime:
    return core.task_spawn_i32("worker")
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected async task target rejection")
	}
	if !strings.Contains(err.Error(), "task_spawn_i32 target must be synchronous") {
		t.Fatalf("error = %v", err)
	}
}

func TestActorSpawnRejectsThrowingTarget(t *testing.T) {
	src := []byte(`
enum SpawnErr:
    case boom

func worker() -> Int throws SpawnErr:
    return 0

func main() -> Int
uses actors:
    let a: actor = core.spawn("worker")
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected throwing actor target rejection")
	}
	if !strings.Contains(err.Error(), "spawn target must not throw") {
		t.Fatalf("error = %v", err)
	}
}

func TestTaskSpawnRejectsMutableGlobalTarget(t *testing.T) {
	requireCheckFileErrorContains(t, `
var g: Int

func worker() -> Int:
    g = g + 1
    return g

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`, "task_spawn_i32 target")
}

func TestActorSpawnRejectsMutableGlobalTarget(t *testing.T) {
	requireCheckFileErrorContains(t, `
var g: Int

func worker() -> Int
uses actors:
    g = g + 1
    return g

func main() -> Int
uses actors:
    let a: actor = core.spawn("worker")
    return 0
`, "spawn target")
}

func TestTaskSpawnAllowsImmutableGlobalTarget(t *testing.T) {
	requireCheckFileOK(t, `
val g: Int = 41

func worker() -> Int:
    return g + 1

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`)
}

func TestTaskGroupsTypedHandlesAndJoinResultCheckAndLower(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    return result.value
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTaskGroupCancelAllowsTypedErrorResult(t *testing.T) {
	requireCheckFileOK(t, `
func worker() -> Int:
    return 99

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    group = core.task_group_cancel(group)
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let err: task.error = result.error
    if core.task_join_i32(task) != 0:
        return 1
    if err == err:
        return 0
    return 1
`)
}
