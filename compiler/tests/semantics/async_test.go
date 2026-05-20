package compiler_test

import (
	"runtime"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func requireCheckFileErrorContains(t *testing.T, src string, want string) {
	t.Helper()
	file, err := compiler.ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: file.Module,
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{file.Module: file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func requireCheckFileOK(t *testing.T, src string) {
	t.Helper()
	file, err := compiler.ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: file.Module,
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{file.Module: file},
	}
	if _, err := compiler.CheckWorld(world); err != nil {
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !prog.Funcs[0].Async {
		t.Fatalf("expected async function")
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !checked.FuncSigs["answer"].Async {
		t.Fatalf("expected async signature")
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestAsyncSmokeExampleBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunFile(t, testkit.RepoPath(t, "examples", "async_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want 42", exitCode)
	}
}

func TestAsyncRejectAwaitOutsideAsync(t *testing.T) {
	src := []byte(`
async func answer() -> Int:
    return 42

func main() -> Int:
    return await answer()
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected bare async call error")
	}
	if !strings.Contains(err.Error(), "requires await") {
		t.Fatalf("error = %v", err)
	}
}

func TestAsyncTypedErrorPropagationTryAwaitCheckAndLower(t *testing.T) {
	src := []byte(`
enum AsyncErr:
    case failed

async func worker() -> Int throws AsyncErr:
    return 42

async func caller() -> Int throws AsyncErr:
    return try await worker()

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestAsyncRejectBorrowedEscapeViaTryAwaitReturn(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum AsyncErr:
    case failed

async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

async func caller(x: borrow ptr) -> ptr throws AsyncErr:
    return try await producer(x)

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaTryAwaitInterproceduralReturn(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum AsyncErr:
    case failed

async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

async func relay(x: borrow ptr) -> ptr throws AsyncErr:
    return try await producer(x)

async func caller(x: borrow ptr) -> ptr throws AsyncErr:
    return try await relay(x)

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaAwaitGlobalAssign(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum AsyncErr:
    case failed

var leaked: ptr = 0

async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

async func caller(x: borrow ptr) -> Int throws AsyncErr:
    leaked = await producer(x)
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaTryAwaitGlobalAssign(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum AsyncErr:
    case failed

var leaked: ptr = 0

async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

async func caller(x: borrow ptr) -> Int throws AsyncErr:
    leaked = try await producer(x)
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaTryAwaitAggregateOptionalGlobalAssign(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum AsyncErr:
    case failed

struct Holder:
    value: ptr

var leaked: Holder? = none

async func producer(x: borrow ptr) -> Holder? throws AsyncErr:
    return Holder { value: x }

async func caller(x: borrow ptr) -> Int throws AsyncErr:
    leaked = try await producer(x)
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaTryAwaitGlobalFieldAssign(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum AsyncErr:
    case failed

struct Holder:
    value: ptr

var holder: Holder

async func producer(x: borrow ptr) -> Holder throws AsyncErr:
    return Holder { value: x }

async func caller(x: borrow ptr) -> Int throws AsyncErr:
    let produced: Holder = try await producer(x)
    holder.value = produced.value
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaTryAwaitStructFieldGlobalAssign(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum AsyncErr:
    case failed

struct Holder:
    value: ptr

var holder: Holder

async func producer(x: borrow ptr) -> Holder throws AsyncErr:
    return Holder { value: x }

async func caller(x: borrow ptr) -> Int throws AsyncErr:
    holder = try await producer(x)
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaTryAwaitOptionalGlobalAssign(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum AsyncErr:
    case failed

var leaked: ptr? = none

async func producer(x: borrow ptr) -> ptr? throws AsyncErr:
    return x

async func caller(x: borrow ptr) -> Int throws AsyncErr:
    leaked = try await producer(x)
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaTryAwaitMatchOptionalGlobalAssign(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum AsyncErr:
    case failed

var leaked: ptr = 0

async func producer(x: borrow ptr) -> ptr? throws AsyncErr:
    return x

async func caller(x: borrow ptr) -> Int throws AsyncErr:
    match try await producer(x):
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestAsyncTypedErrorBoundaryRejectsAwaitTryForm(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum AsyncErr:
    case failed

async func worker() -> Int throws AsyncErr:
    throw AsyncErr.failed

async func caller() -> Int throws AsyncErr:
    return await try worker()

func main() -> Int:
    return 0
`, "use 'try await worker()'")
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTaskSmokeExampleBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunFile(t, testkit.RepoPath(t, "examples", "task_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want 42", exitCode)
	}
}

func TestTaskSpawnRequiresRuntimeUse(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 42

func main() -> task.i32:
    return core.task_spawn_i32("worker")
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected throwing actor target rejection")
	}
	if !strings.Contains(err.Error(), "spawn target must not throw") {
		t.Fatalf("error = %v", err)
	}
}

func TestTaskSpawnRejectsThrowingTarget(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum SpawnErr:
    case boom

func worker() -> Int throws SpawnErr:
    return 0

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`, "task_spawn_i32 target must not throw")
}

func TestTaskSpawnJoinTypedErrorCheckAndLower(t *testing.T) {
	src := []byte(`
enum TaskErr:
    case boom

func worker() -> Int throws TaskErr:
    return 42

func caller() -> Int throws TaskErr
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTaskJoinTypedErrorRequiresTry(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum TaskErr:
    case boom

func worker() -> Int throws TaskErr:
    return 42

func caller() -> Int throws TaskErr
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`, "requires try")
}

func TestTaskJoinTypedErrorCatchCheckAndLower(t *testing.T) {
	src := []byte(`
enum TaskErr:
    case boom
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom

func caller() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom:
        7
    case TaskErr.stopped:
        8

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTaskSpawnJoinTypedPayloadErrorTryCheckAndLower(t *testing.T) {
	src := []byte(`
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    return 42

func caller() -> Int throws TaskErr
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTaskSpawnJoinTypedPayloadErrorCatchBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(7)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`
	stdout, exitCode := buildAndRunWithOptions(t, src, compiler.BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want 7", exitCode)
	}
}

func TestTaskSpawnGroupRejectsThrowingTarget(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum SpawnErr:
    case boom

func worker() -> Int throws SpawnErr:
    return 0

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`, "task_spawn_group_i32 target must not throw")
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
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
    if err == err:
        return 0
    return 1
`)
}

func TestTaskSpawnGroupTypedPayloadErrorSuccessBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum GroupErr:
    case stopped
    case boom(Int)

func worker() -> Int throws GroupErr:
    return 42

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task = core.task_spawn_group_i32_typed<GroupErr>(group, "worker")
    let value: Int = catch core.task_join_group_i32_typed<GroupErr>(task):
    case GroupErr.stopped:
        5
    case GroupErr.boom(code):
        code
    let _closed: Int = core.task_group_close(group)
    return value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, compiler.BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want 42", exitCode)
	}
}

func TestTaskSpawnGroupTypedPayloadErrorCatchBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum GroupErr:
    case stopped
    case boom(Int)

func worker() -> Int throws GroupErr:
    throw GroupErr.boom(7)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task = core.task_spawn_group_i32_typed<GroupErr>(group, "worker")
    return catch core.task_join_group_i32_typed<GroupErr>(task):
    case GroupErr.stopped:
        5
    case GroupErr.boom(code):
        code
`
	stdout, exitCode := buildAndRunWithOptions(t, src, compiler.BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want 7", exitCode)
	}
}

func TestTaskSpawnGroupTypedPayloadCancelBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum GroupErr:
    case stopped
    case boom(Int)

func worker() -> Int throws GroupErr:
    return 99

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    group = core.task_group_cancel(group)
    let task = core.task_spawn_group_i32_typed<GroupErr>(group, "worker")
    return catch core.task_join_group_i32_typed<GroupErr>(task):
    case GroupErr.stopped:
        5
    case GroupErr.boom(code):
        code
`
	stdout, exitCode := buildAndRunWithOptions(t, src, compiler.BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code = %d, want 5", exitCode)
	}
}

func TestTaskSpawnGroupTypedRejectsSpawnAfterClose(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum GroupErr:
    case stopped

func worker() -> Int throws GroupErr:
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let _: Int = core.task_group_close(group)
    let task = core.task_spawn_group_i32_typed<GroupErr>(group, "worker")
    return 0
`, "cannot use closed resource 'group'")
}

func TestTaskSpawnGroupTypedRejectsMaybeClosedAfterControlFlowMerge(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum GroupErr:
    case stopped

func worker() -> Int throws GroupErr:
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    if 1:
        let _: Int = core.task_group_close(group)
    let task = core.task_spawn_group_i32_typed<GroupErr>(group, "worker")
    return 0
`, "may have been closed after control-flow merge")
}

func TestTaskSpawnGroupTypedRejectsOptionalPayloadAliasAfterClose(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum GroupErr:
    case stopped

func worker() -> Int throws GroupErr:
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let other = maybe:
        let _: Int = core.task_group_close(group)
        let task = core.task_spawn_group_i32_typed<GroupErr>(other, "worker")
        return 0
    return 1
`, "cannot use closed resource 'other'")
}

func TestTaskSpawnGroupTypedRejectsMutableGlobalTarget(t *testing.T) {
	requireCheckFileErrorContains(t, `
enum GroupErr:
    case stopped

var g: Int

func worker() -> Int throws GroupErr:
    g = g + 1
    return g

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task = core.task_spawn_group_i32_typed<GroupErr>(group, "worker")
    return catch core.task_join_group_i32_typed<GroupErr>(task):
    case GroupErr.stopped:
        0
`, "task_spawn_group_i32_typed target")
}
