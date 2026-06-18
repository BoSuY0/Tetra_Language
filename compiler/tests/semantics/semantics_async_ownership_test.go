package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

// ---- async_test.go ----

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

	stdout, exitCode := buildAndRunFile(
		t,
		testkit.RepoPath(t, "examples", "async", "async_smoke.tetra"),
	)
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

	stdout, exitCode := buildAndRunFile(
		t,
		testkit.RepoPath(t, "examples", "tasks", "task_smoke.tetra"),
	)
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
	if !strings.Contains(
		err.Error(),
		"task_spawn_i32 target must have shape func worker() -> i32",
	) {
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

// ---- async_borrow_lifetime_test.go ----

func TestAsyncBorrowLifetimeAllowsPreAwaitLocalUse(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
async func ready() -> Int:
    return 1

async func caller(xs: borrow []u8) -> Int:
    let view: []u8 = xs.borrow()
    let before: Int = view.len
    let after: Int = await ready()
    return before + after

func main() -> Int:
    return 0
`)
}

func TestAsyncBorrowLifetimeRejectsPostAwaitLocalUse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
async func ready() -> Int:
    return 1

async func caller(xs: borrow []u8) -> Int:
    let view: []u8 = xs.borrow()
    let _: Int = await ready()
    return view.len

func main() -> Int:
    return 0
`, "borrowed view 'view' cannot be used after await suspension")
}

func TestAsyncBorrowLifetimeRejectsPostTryAwaitLocalUse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum AsyncErr:
    case failed

async func ready() -> Int throws AsyncErr:
    return 1

async func caller(xs: borrow []u8) -> Int throws AsyncErr:
    let view: []u8 = xs.borrow()
    let _: Int = try await ready()
    return view.len

func main() -> Int:
    return 0
`, "borrowed view 'view' cannot be used after await suspension")
}

// ---- async_inout_ownership_test.go ----

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_inout_base.t4": `module lib.ownership_async_inout_base

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"app/main.t4": `module app.main
import lib.ownership_async_inout_base as base

async func caller(x: borrow ptr, out: inout ptr) -> Int throws base.AsyncErr:
    out = try await base.producer(x)
    return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_inout_base_await.t4": `module lib.ownership_async_inout_base_await

pub async func producer(x: borrow ptr) -> ptr:
    return x
`,
		"app/main.t4": `module app.main
import lib.ownership_async_inout_base_await as base

async func caller(x: borrow ptr, out: inout ptr) -> Int:
    out = await base.producer(x)
    return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_base_await.t4": `module lib.ownership_async_callback_base_await

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, cb: fn(inout ptr) -> Int) -> Int:
    let value: ptr = await producer(x)
    return cb(value)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_base_await as base

async func caller(x: borrow ptr, cb: fn(inout ptr) -> Int) -> Int:
    return await base.relay(x, cb)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayAwaitEnumPayloadCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_relay_base_await.t4": `module lib.ownership_async_relay_base_await

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let value: ptr = await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_base_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitEnumPayloadCallbackInoutWithThrows(
	t *testing.T,
) {
	files := map[string]string{
		"lib/ownership_async_relay_base_throws_await.t4": `module lib.ownership_async_relay_base_throws_await

pub enum AsyncErr:
    case failed

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_base_throws_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitOptionalEnumPayloadCallbackInout(
	t *testing.T,
) {
	files := map[string]string{
		"lib/ownership_async_relay_optional_payload_await.t4": `module lib.ownership_async_relay_optional_payload_await

pub enum AsyncErr:
    case failed

pub struct Box:
    value: ptr?

pub enum Holder:
    case some(fn(inout ptr?) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> Box throws AsyncErr:
    return Box { value: x }

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let produced: Box = try await producer(x)
    let value: ptr? = produced.value
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_optional_payload_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitEnumPayloadCallbackInoutChain(
	t *testing.T,
) {
	files := map[string]string{
		"lib/ownership_async_relay_chain_base.t4": `module lib.ownership_async_relay_chain_base

pub enum AsyncErr:
    case failed

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"lib/ownership_async_relay_chain_mid.t4": `module lib.ownership_async_relay_chain_mid
import lib.ownership_async_relay_chain_base as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_chain_base as base
import lib.ownership_async_relay_chain_mid as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitEnumPayloadCallbackInoutChain(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_relay_chain_base_no_throw.t4": `module lib.ownership_async_relay_chain_base_no_throw

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let value: ptr = await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"lib/ownership_async_relay_chain_mid_no_throw.t4": `module lib.ownership_async_relay_chain_mid_no_throw
import lib.ownership_async_relay_chain_base_no_throw as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_chain_base_no_throw as base
import lib.ownership_async_relay_chain_mid_no_throw as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitOptionalEnumPayloadCallbackInoutChain(
	t *testing.T,
) {
	files := map[string]string{
		"lib/ownership_async_relay_chain_optional_base.t4": `module lib.ownership_async_relay_chain_optional_base

pub enum AsyncErr:
    case failed

pub struct Box:
    value: ptr?

pub enum Holder:
    case some(fn(inout ptr?) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> Box throws AsyncErr:
    return Box { value: x }

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let produced: Box = try await producer(x)
    let value: ptr? = produced.value
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"lib/ownership_async_relay_chain_optional_mid.t4": `module lib.ownership_async_relay_chain_optional_mid
import lib.ownership_async_relay_chain_optional_base as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_chain_optional_base as base
import lib.ownership_async_relay_chain_optional_mid as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitStructFieldCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_struct_await.t4": `module lib.ownership_async_callback_struct_await

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let value: ptr = await producer(x)
    return h.cb(value)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_struct_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitOptionalStructFieldCallbackInout(
	t *testing.T,
) {
	files := map[string]string{
		"lib/ownership_async_callback_struct_optional_await.t4": `module lib.ownership_async_callback_struct_optional_await

pub struct Box:
    value: ptr?

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub async func producer(x: borrow ptr) -> Box:
    return Box { value: x }

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let produced: Box = await producer(x)
    let value: ptr? = produced.value
    match value:
    case some(raw):
        return h.cb(raw)
    case none:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_struct_optional_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitEnumPayloadCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_enum_await.t4": `module lib.ownership_async_callback_enum_await

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let value: ptr = await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_enum_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitOptionalEnumPayloadCallbackInoutChainNoThrow(
	t *testing.T,
) {
	files := map[string]string{
		"lib/ownership_async_relay_chain_optional_base_no_throw.t4": `module lib.ownership_async_relay_chain_optional_base_no_throw

pub struct Box:
    value: ptr?

pub enum Holder:
    case some(fn(inout ptr?) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> Box:
    return Box { value: x }

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let produced: Box = await producer(x)
    let value: ptr? = produced.value
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"lib/ownership_async_relay_chain_optional_mid_no_throw.t4": `module lib.ownership_async_relay_chain_optional_mid_no_throw
import lib.ownership_async_relay_chain_optional_base_no_throw as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_chain_optional_base_no_throw as base
import lib.ownership_async_relay_chain_optional_mid_no_throw as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitStructFieldCallbackInoutChain(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_struct_chain_base.t4": `module lib.ownership_async_callback_struct_chain_base

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let value: ptr = await producer(x)
    return h.cb(value)
`,
		"lib/ownership_async_callback_struct_chain_mid.t4": `module lib.ownership_async_callback_struct_chain_mid
import lib.ownership_async_callback_struct_chain_base as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_struct_chain_base as base
import lib.ownership_async_callback_struct_chain_mid as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_inout_base.t4": `module lib.ownership_async_inout_base

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"lib/ownership_async_inout_relay.t4": `module lib.ownership_async_inout_relay
import lib.ownership_async_inout_base as base

pub async func relay(x: borrow ptr, out: inout ptr) -> Int throws base.AsyncErr:
    out = try await base.producer(x)
    return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_inout_base as base
import lib.ownership_async_inout_relay as relay

async func caller(x: borrow ptr, out: inout ptr) -> Int throws base.AsyncErr:
    return try await relay.relay(x, out)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitStructFieldCallbackInoutChain(
	t *testing.T,
) {
	files := map[string]string{
		"lib/ownership_async_relay_struct_chain_base.t4": `module lib.ownership_async_relay_struct_chain_base

pub enum AsyncErr:
    case failed

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    return h.cb(value)
`,
		"lib/ownership_async_relay_struct_chain_mid.t4": `module lib.ownership_async_relay_struct_chain_mid
import lib.ownership_async_relay_struct_chain_base as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_struct_chain_base as base
import lib.ownership_async_relay_struct_chain_mid as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_base.t4": `module lib.ownership_async_callback_base

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, cb: fn(inout ptr) -> Int) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    return cb(value)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_base as base

async func caller(x: borrow ptr, cb: fn(inout ptr) -> Int) -> Int throws base.AsyncErr:
    return try await base.relay(x, cb)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitStructFieldCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_struct.t4": `module lib.ownership_async_callback_struct

pub enum AsyncErr:
    case failed

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    return h.cb(value)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_struct as base

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitOptionalStructFieldCallbackInout(
	t *testing.T,
) {
	files := map[string]string{
		"lib/ownership_async_callback_struct_optional.t4": `module lib.ownership_async_callback_struct_optional

pub enum AsyncErr:
    case failed

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub struct Box:
    value: ptr?

pub async func producer(x: borrow ptr) -> Box throws AsyncErr:
    return Box { value: x }

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: Box = try await producer(x)
    match value.value:
    case some(raw):
        return h.cb(raw)
    case none:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_struct_optional as base

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitEnumPayloadCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_enum.t4": `module lib.ownership_async_callback_enum

pub enum AsyncErr:
    case failed

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_enum as base

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

// ---- async_ownership_test.go ----

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async.t4": `module lib.ownership_async

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"app/main.t4": `module app.main
import lib.ownership_async as ownership

async func caller(x: borrow ptr) -> ptr throws ownership.AsyncErr:
    return try await ownership.producer(x)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async.t4": `module lib.ownership_async

pub async func producer(x: borrow ptr) -> ptr:
    return x
`,
		"app/main.t4": `module app.main
import lib.ownership_async as ownership

async func caller(x: borrow ptr) -> ptr:
    return await ownership.producer(x)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitGlobalAssign(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async.t4": `module lib.ownership_async

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"app/main.t4": `module app.main
import lib.ownership_async as ownership

var leaked: ptr = 0

async func caller(x: borrow ptr) -> Int throws ownership.AsyncErr:
    leaked = try await ownership.producer(x)
    return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> ptr throws base.AsyncErr:
    return try await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

async func caller(x: borrow ptr) -> ptr throws base.AsyncErr:
    return try await relay.relay(x)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayAwaitReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub async func producer(x: borrow ptr) -> ptr:
    return x
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> ptr:
    return await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

async func caller(x: borrow ptr) -> ptr:
    return await relay.relay(x)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitGlobalAssign(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> ptr throws base.AsyncErr:
    return try await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

var leaked: ptr = 0

async func caller(x: borrow ptr) -> Int throws base.AsyncErr:
    leaked = try await relay.relay(x)
    return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayAwaitOptionalReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub struct Holder:
    value: ptr

pub async func producer(x: borrow ptr) -> Holder?:
    return Holder { value: x }
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> base.Holder?:
    return await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

async func caller(x: borrow ptr) -> base.Holder?:
    return await relay.relay(x)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitOptionalGlobalAssign(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub struct Holder:
    value: ptr

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> Holder? throws AsyncErr:
    return Holder { value: x }
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> base.Holder? throws base.AsyncErr:
    return try await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

var leaked: base.Holder? = none

async func caller(x: borrow ptr) -> Int throws base.AsyncErr:
    leaked = try await relay.relay(x)
    return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayMatchOptionalReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub struct Holder:
    value: ptr

pub async func producer(x: borrow ptr) -> Holder?:
    return Holder { value: x }
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> base.Holder?:
    return await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

async func caller(x: borrow ptr) -> ptr:
    match await relay.relay(x):
    case some(value):
        return value.value
    case none:
        return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitMatchOptionalGlobalAssign(
	t *testing.T,
) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub struct Holder:
    value: ptr

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> Holder? throws AsyncErr:
    return Holder { value: x }
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> base.Holder? throws base.AsyncErr:
    return try await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

var leaked: ptr = 0

async func caller(x: borrow ptr) -> Int throws base.AsyncErr:
    match try await relay.relay(x):
    case some(value):
        leaked = value.value
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"borrowed local 'x' cannot escape via return",
	)
}

// ---- atomic_builtin_test.go ----

func TestAtomicBuiltinInvalidFormsReportExplicitDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		call string
		want string
	}{
		{
			name: "load release",
			call: "core.atomic_load_i32_release(p, mem)",
			want: "atomic load does not support memory order release",
		},
		{
			name: "store acquire",
			call: "core.atomic_store_i32_acquire(p, 1, mem)",
			want: "atomic store does not support memory order acquire",
		},
		{
			name: "unknown order",
			call: "core.atomic_fetch_add_i32_consume(p, 1, mem)",
			want: "unsupported atomic memory order 'consume'",
		},
		{
			name: "unknown op",
			call: "core.atomic_nand_i32_relaxed(p, 1, mem)",
			want: "unsupported atomic operation 'nand'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        return `+tt.call+`
    return 0
`, tt.want)
		})
	}
}

func TestAtomicBuiltinI64AndWeakCompareExchangeSurfaceChecks(t *testing.T) {
	testkit.RequireCheckOK(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        let exchanged: i64 = core.atomic_exchange_i64_seq_cst(p, loaded, mem)
        let weak: i64 = core.atomic_compare_exchange_weak_i64_seq_cst(p, loaded, exchanged, mem)
        var ignored_store: i64 = core.atomic_store_i64_release(p, weak, mem)
        return 0
    return 0
`)
}

// ---- borrow_copy_test.go ----

func TestSliceAndStringBorrowTypeCheckWithoutAllocEffect(t *testing.T) {
	testkit.RequireCheckOK(t, `
func slice_len(xs: []i32) -> Int:
    let b: []i32 = xs.window(1, 2).borrow()
    return b.len

func string_len() -> Int:
    let b: String = "abcdef".window(1, 3).borrow()
    return b.len

func main() -> Int:
    return string_len()
`)
}

func TestBuildSliceCopyCreatesIndependentOwnedStorage(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 10
    xs[1] = 20
    xs[2] = 30
    xs[3] = 40
    let ys: []i32 = xs.window(1, 2).copy()
    xs[1] = 99
    if ys.len != 2:
        return 1
    if ys[0] != 20:
        return 2
    if ys[1] != 30:
        return 3
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStringCopyCreatesIndependentOwnedStorage(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    let text: String = "abcdef"
    let mid: String = text.window(1, 3).copy()
    if mid.len != 3:
        return 1
    if mid[0] != 98:
        return 2
    if mid[1] != 99:
        return 3
    if mid[2] != 100:
        return 4
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildCopyIntoMutatesDestinationAndReturnsCount(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var src: []u8 = make_u8(2)
    src[0] = 65
    src[1] = 66
    var dst: []u8 = make_u8(2)
    dst[0] = 1
    dst[1] = 2
    let n: Int = src.copy_into(dst)
    if n != 2:
        return 1
    if dst[0] != 65:
        return 2
    if dst[1] != 66:
        return 3
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStringCopyIntoMutatesDestinationAndReturnsCount(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    let text: String = "abcdef".window(1, 3)
    var dst: []u8 = make_u8(3)
    let n: Int = text.copy_into(dst)
    if n != 3:
        return 1
    if dst[0] != 98:
        return 2
    if dst[1] != 99:
        return 3
    if dst[2] != 100:
        return 4
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildCopyIntoRejectsInsufficientDestination(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var src: []u8 = make_u8(2)
    var dst: []u8 = make_u8(1)
    return src.copy_into(dst)
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode == 0 || exitCode == 42 {
		t.Fatalf("insufficient destination exited %d, want trap/non-success", exitCode)
	}
}

func TestBuildCopyIntoZeroLengthSucceedsWithoutTouchingDestination(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var src: []u8 = make_u8(0)
    var dst: []u8 = make_u8(1)
    dst[0] = 77
    let n: Int = src.copy_into(dst)
    if n != 0:
        return 1
    if dst[0] != 77:
        return 2
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBorrowedSliceAndStringEscapeDiagnostics(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func leak(xs: []i32) -> []i32:
    let b: []i32 = xs.borrow()
    return b

func main() -> Int:
    return 0
`, "borrowed slice return requires '-> borrow []i32' or '.copy()'")

	testkit.RequireFileCheckErrorContains(t, `
func leak(text: String) -> String:
    return text.window(1, 2).borrow()

func main() -> Int:
    return 0
`, "borrowed String return requires '-> borrow String' or '.copy()'")
}

func TestBorrowedSliceAndStringBorrowedReturnContracts(t *testing.T) {
	testkit.RequireCheckOK(t, `
func view_bytes(xs: borrow []u8) -> borrow []u8:
    return xs.window(1, 2).borrow()

func view_u16(xs: borrow []u16) -> borrow []u16:
    return xs.window(1, 2).borrow()

func view_i32(xs: borrow []i32) -> borrow []i32:
    return xs.window(1, 2).borrow()

func view_bool(xs: borrow []bool) -> borrow []bool:
    return xs.window(1, 2).borrow()

func view_text(text: borrow String) -> borrow String:
    return text.window(1, 2).borrow()

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckErrorContains(t, `
func leak_owned(xs: borrow []u8) -> []u8:
    return xs.window(0, 1).borrow()

func main() -> Int:
    return 0
`, "borrowed slice return requires '-> borrow []u8' or '.copy()'")

	testkit.RequireFileCheckErrorContains(t, `
func leak_local() -> borrow []u8
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    return xs.window(0, 1).borrow()

func main() -> Int:
    return 0
`, "borrowed slice return derives from local owner 'xs'")
}

func TestFunctionTypedBorrowedReturnOwnershipContract(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Holder:
    cb: fn(borrow []u8) -> borrow []u8

enum Choice:
    case cb(fn(borrow []u8) -> borrow []u8)

func borrowed_view(xs: borrow []u8) -> borrow []u8:
    return xs.borrow()

func apply(cb: fn(borrow []u8) -> borrow []u8, xs: borrow []u8) -> borrow []u8:
    return cb(xs)

func from_local(xs: borrow []u8) -> borrow []u8:
    let cb: fn(borrow []u8) -> borrow []u8 = borrowed_view
    return cb(xs)

func from_field(xs: borrow []u8, holder: Holder) -> borrow []u8:
    return holder.cb(xs)

func from_enum(xs: borrow []u8) -> Choice:
    return Choice.cb(borrowed_view)

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckErrorContains(t, `
func owned_copy(xs: borrow []u8) -> []u8
uses alloc, mem:
    return xs.copy()

func apply(xs: borrow []u8, cb: fn(borrow []u8) -> borrow []u8 uses alloc, mem) -> borrow []u8
uses alloc, mem:
    return cb(xs)

func bad(xs: borrow []u8) -> borrow []u8
uses alloc, mem:
    return apply(xs, owned_copy)

func main() -> Int:
    return 0
`, ("callback function symbol 'owned_copy' return ownership " +
		"mismatch: expected 'borrow', got 'owned'"))

	testkit.RequireFileCheckErrorContains(t, `
func owned_copy(xs: borrow []u8) -> []u8
uses alloc, mem:
    return xs.copy()

func main() -> Int
uses alloc, mem:
    let cb: fn(borrow []u8) -> borrow []u8 uses alloc, mem = owned_copy
    return 0
`, "function-typed local 'cb' return ownership mismatch: expected 'borrow', got 'owned'")

	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, mem:
    let cb: fn(borrow []u8) -> borrow []u8 uses alloc, mem = fn(xs: borrow []u8) -> []u8
    uses alloc, mem:
        return xs.copy()
    return 0
`, "function-typed local 'cb' return ownership mismatch: expected 'borrow', got 'owned'")

	testkit.RequireFileCheckErrorContains(t, `
var cb: fn(borrow []u8) -> borrow []u8 uses alloc, mem = fn(xs: borrow []u8) -> []u8
uses alloc, mem:
    return xs.copy()

func main() -> Int:
    return 0
`, "function-typed local 'cb' return ownership mismatch: expected 'borrow', got 'owned'")

	testkit.RequireFileCheckErrorContains(t, `
struct Holder:
    cb: fn(borrow []u8) -> borrow []u8 uses alloc, mem

func main() -> Int
uses alloc, mem:
    let h: Holder = Holder(cb: fn(xs: borrow []u8) -> []u8
    uses alloc, mem:
        return xs.copy()
    )
    return 0
`, "function-typed assignment to 'h.cb' return ownership mismatch: expected 'borrow', got 'owned'")

	testkit.RequireFileCheckErrorContains(t, `
enum Choice:
    case cb(fn(borrow []u8) -> borrow []u8 uses alloc, mem)

func main() -> Int
uses alloc, mem:
    let c: Choice = Choice.cb(fn(xs: borrow []u8) -> []u8
    uses alloc, mem:
        return xs.copy()
    )
    return 0
`, ("function-typed assignment to 'Choice.cb[1]' return ownership " +
		"mismatch: expected 'borrow', got 'owned'"))
}

func TestBorrowedReturnForwardingRequiresBorrowReturn(t *testing.T) {
	testkit.RequireCheckOK(t, `
func inner(xs: borrow []u8) -> borrow []u8:
    return xs.window(1, 2).borrow()

func outer(xs: borrow []u8) -> borrow []u8:
    return inner(xs)

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckErrorContains(t, `
func inner(xs: borrow []u8) -> borrow []u8:
    return xs.window(1, 2).borrow()

func outer_bad(xs: borrow []u8) -> []u8:
    return inner(xs)

func main() -> Int:
    return 0
`, "borrowed slice return requires '-> borrow []u8' or '.copy()'")
}

func TestBorrowedReturnBranchOriginConsistency(t *testing.T) {
	testkit.RequireCheckOK(t, `
func choose_same(flag: Bool, xs: borrow []u8) -> borrow []u8:
    if flag:
        return xs.prefix(2).borrow()
    return xs.suffix(1).borrow()

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckErrorContains(t, `
func choose(flag: Bool, a: borrow []u8, b: borrow []u8) -> borrow []u8:
    if flag:
        return a.borrow()
    return b.borrow()

func main() -> Int:
    return 0
`, ("borrowed return has multiple possible owner sources ('a', 'b'); " +
		"named lifetimes are not supported in v1"))
}

func TestBorrowedReturnRejectsUnsafeUnknownProvenance(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func bad(xs: []u8) -> borrow []u8
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        return core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem).borrow()

func main() -> Int:
    return 0
`, "borrowed slice return requires caller-visible borrow source")
}

func TestBorrowedAggregateEscapeDiagnostics(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Box:
    bytes: []u8

func bad(xs: borrow []u8) -> Box:
    return Box(bytes: xs.window(1, 2).borrow())

func main() -> Int:
    return 0
`, "aggregate 'Box' contains borrowed slice field 'bytes' that cannot escape through owned return")

	testkit.RequireFileCheckErrorContains(t, `
struct Box:
    bytes: []u8

var saved: Box

func stash(xs: borrow []u8) -> Int:
    saved = Box(bytes: xs.window(1, 2).borrow())
    return 0
`, "aggregate 'Box' contains borrowed slice field 'bytes' that cannot be stored in global")

	testkit.RequireCheckErrorContains(t, `
struct Box:
    bytes: []u8

enum Msg:
    case boxed(Box)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(4)
    return core.send_typed(core.self(), Msg.boxed(Box(bytes: xs.window(1, 2).borrow())))
`, "aggregate 'Box' contains borrowed slice field 'bytes' that cannot cross actor boundary")

	testkit.RequireFileCheckErrorContains(t, `
struct TextBox:
    text: String

func bad_text(text: borrow String) -> TextBox:
    return TextBox(text: text.window(1, 2).borrow())

func main() -> Int:
    return 0
`, ("aggregate 'TextBox' contains borrowed String field 'text' that " +
		"cannot escape through owned return"))

	testkit.RequireFileCheckErrorContains(t, `
func bad_optional(xs: borrow []u8) -> []u8?:
    return xs.window(1, 2).borrow()

func main() -> Int:
    return 0
`, ("aggregate '[]u8?' contains borrowed slice field '$elem' that " +
		"cannot escape through owned return"))

	testkit.RequireFileCheckErrorContains(t, `
enum MaybeBytes:
    case some([]u8)
    case empty

func bad_enum(xs: borrow []u8) -> MaybeBytes:
    return MaybeBytes.some(xs.window(1, 2).borrow())

func main() -> Int:
    return 0
`, ("aggregate 'MaybeBytes' contains borrowed slice field " +
		"'MaybeBytes.some[1]' that cannot escape through owned return"))

	testkit.RequireFileCheckErrorContains(t, `
struct Box<T>:
    value: T

func bad_generic(xs: borrow []u8) -> Box<[]u8>:
    return Box<[]u8>{value: xs.window(1, 2).borrow()}

func main() -> Int:
    return 0
`, "contains borrowed slice field 'value' that cannot escape through owned return")

	testkit.RequireCheckOK(t, `
struct Box<T>:
    value: T

enum MaybeBytes:
    case some([]u8)
    case empty

func good_optional(xs: borrow []u8) -> []u8?
uses alloc, mem:
    let owned: []u8 = xs.window(1, 2).copy()
    return owned

func good_enum(xs: borrow []u8) -> MaybeBytes
uses alloc, mem:
    return MaybeBytes.some(xs.window(1, 2).copy())

func good_generic(xs: borrow []u8) -> Box<[]u8>
uses alloc, mem:
    return Box<[]u8>{value: xs.window(1, 2).copy()}

func main() -> Int:
    return 0
	`)
}

func TestMemoryIdealV0BorrowStructOptionalLocalAndCopyEscapes(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Box:
    bytes: []u8

func local_struct(xs: borrow []u8) -> Int:
    let box: Box = Box(bytes: xs.window(1, 2).borrow())
    return box.bytes.len

func local_optional(xs: borrow []u8) -> Int:
    let maybe: []u8? = xs.window(1, 2).borrow()
    if let raw = maybe:
        return raw.len
    else:
        return 0

func return_copied_struct(xs: borrow []u8) -> Box
uses alloc, mem:
    return Box(bytes: xs.window(0, 1).copy())

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckOK(t, `
struct Box:
    bytes: []u8

var saved: []u8? = none

func stash_copied_optional(xs: borrow []u8) -> Int
uses alloc, mem:
    saved = xs.window(0, 1).copy()
    return 0

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV1BorrowEnumPayloadAndGenericWrapperLocalUse(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Box<T>:
    value: T

enum MaybeBytes:
    case some([]u8)
    case empty

func local_enum(xs: borrow []u8) -> Int:
    let maybe: MaybeBytes = MaybeBytes.some(xs.window(1, 2).borrow())
    match maybe:
        case MaybeBytes.some(raw):
            return raw.len
        case MaybeBytes.empty:
            return 0

func local_generic(xs: borrow []u8) -> Int:
    let box: Box<[]u8> = Box<[]u8>{value: xs.window(1, 2).borrow()}
    return box.value.len

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV1BorrowEnumPayloadGlobalStorageRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MaybeBytes:
    case some([]u8)
    case empty

var saved: MaybeBytes

func stash(xs: borrow []u8) -> Int:
    saved = MaybeBytes.some(xs.window(1, 2).borrow())
    return 0
`, ("aggregate 'MaybeBytes' contains borrowed slice field " +
		"'MaybeBytes.some[1]' that cannot be stored in global"))
}

func TestMemoryIdealV1BorrowGenericWrapperGlobalStorageRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Box<T>:
    value: T

var saved: Box<[]u8>

func stash(xs: borrow []u8) -> Int:
    saved = Box<[]u8>{value: xs.window(1, 2).borrow()}
    return 0
`, "contains borrowed slice field 'value' that cannot be stored in global")
}

func TestMemoryIdealV1BorrowEnumPayloadAndGenericWrapperCopyEscapes(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
struct Box<T>:
    value: T

enum MaybeBytes:
    case some([]u8)
    case empty

var saved_enum: MaybeBytes
var saved_box: Box<[]u8>

func stash_copied(xs: borrow []u8) -> Int
uses alloc, mem:
    saved_enum = MaybeBytes.some(xs.window(0, 1).copy())
    saved_box = Box<[]u8>{value: xs.window(1, 1).copy()}
    return 0

func return_enum(xs: borrow []u8) -> MaybeBytes
uses alloc, mem:
    return MaybeBytes.some(xs.window(0, 1).copy())

func return_generic(xs: borrow []u8) -> Box<[]u8>
uses alloc, mem:
    return Box<[]u8>{value: xs.window(0, 1).copy()}

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV2KnownCallbackAndFunctionTypedFieldBorrowUse(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct Sink:
    cb: fn(borrow []u8) -> Int

func len_borrow(xs: borrow []u8) -> Int:
    return xs.len

func local_callback(xs: borrow []u8) -> Int:
    let cb: fn(borrow []u8) -> Int = len_borrow
    return cb(xs.window(0, 1).borrow())

func field_callback(xs: borrow []u8, sink: Sink) -> Int:
    return sink.cb(xs.window(0, 1).borrow())

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV2CopyBeforeCallbackEscapeAccepted(t *testing.T) {
	testkit.RequireCheckOK(t, `
func takes_owned(xs: []u8) -> Int:
    return xs.len

func call_with_copy(xs: borrow []u8) -> Int
uses alloc, mem:
    let cb: fn([]u8) -> Int = takes_owned
    let owned: []u8 = xs.window(0, 1).copy()
    return cb(owned)

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV2BorrowedCallbackNonBorrowParamRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func takes_owned(xs: []u8) -> Int:
    return xs.len

func bad(xs: borrow []u8) -> Int:
    let cb: fn([]u8) -> Int = takes_owned
    return cb(xs.window(0, 1).borrow())

func main() -> Int:
    return 0
`, "borrowed value derived from 'xs' cannot be passed to non-borrow parameter 1 of callback 'cb'")
}

func TestMemoryIdealV2BorrowedCallbackReturnAsOwnedRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func view(xs: borrow []u8) -> borrow []u8:
    return xs.window(0, 1).borrow()

func bad(xs: borrow []u8) -> []u8:
    let cb: fn(borrow []u8) -> borrow []u8 = view
    return cb(xs)

func main() -> Int:
    return 0
`, "borrowed slice return requires '-> borrow []u8' or '.copy()'")
}

func TestMemoryIdealV2BorrowedCallbackGlobalStorageRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var saved: []u8

func view(xs: borrow []u8) -> borrow []u8:
    return xs.window(0, 1).borrow()

func bad(xs: borrow []u8) -> Int:
    let cb: fn(borrow []u8) -> borrow []u8 = view
    saved = cb(xs)
    return 0
`, "borrowed local 'xs' cannot escape via global assignment to 'saved'")
}

func TestMemoryIdealV2BorrowedCallbackConsumeAndInoutRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func consume_bytes(xs: consume []u8) -> Int:
    return xs.len

func bad(xs: borrow []u8) -> Int:
    let view: []u8 = xs.window(0, 1).borrow()
    let cb: fn(consume []u8) -> Int = consume_bytes
    return cb(view)

func main() -> Int:
    return 0
`, "borrowed value derived from 'xs' cannot be consumed by callback 'cb'")

	testkit.RequireFileCheckErrorContains(t, `
func mutate(xs: inout []u8) -> Int:
    return xs.len

func bad(xs: borrow []u8) -> Int:
    var view: []u8 = xs.window(0, 1).borrow()
    let cb: fn(inout []u8) -> Int = mutate
    return cb(view)

func main() -> Int:
    return 0
`, "borrowed value derived from 'xs' cannot be passed as inout to callback 'cb'")
}

func TestMemoryIdealV2CallbackAliasesInoutArgumentRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func touch(dst: inout []u8, view: borrow []u8) -> Int:
    return dst.len + view.len

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    let cb: fn(inout []u8, borrow []u8) -> Int = touch
    return cb(xs, xs)
`, "borrowed argument 'xs' aliases inout argument in callback 'cb'")
}

func TestMemoryIdealV2UnknownCallbackTargetConservativeRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func apply(cb: fn(borrow []u8) -> Int, xs: borrow []u8) -> Int:
    return cb(xs)

func outer(cb: fn(borrow []u8) -> Int, xs: borrow []u8) -> Int
noalloc:
    return apply(cb, xs)

func main() -> Int:
    return 0
`, "callback argument for 'apply' has no known fnptr target under semantic clause 'noalloc'")
}

func TestMemoryIdealV2CapturingCallbackGlobalEscapeRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var saved: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(cb: fn(Int) -> Int) -> Int:
    saved = cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`, "function-typed parameter 'cb' cannot be stored in global function-typed value 'saved'")
}

func TestMemoryIdealV3KnownStaticProtocolTargetBorrowUse(t *testing.T) {
	testkit.RequireCheckOK(t, `
struct BorrowView:
    value: []u8

protocol ViewLike:
    func len(self: borrow BorrowView) -> Int

extension BorrowView:
    func len(self: borrow BorrowView) -> Int:
        return self.value.len

impl BorrowView: ViewLike

func local_static_protocol(xs: borrow []u8) -> Int:
    let view: BorrowView = BorrowView{value: xs.window(0, 1).borrow()}
    return BorrowView.len(view)

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV3BorrowedInterfaceReturnAsOwnedRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct BorrowView:
    value: []u8

protocol ViewLike:
    func len(self: BorrowView) -> Int

extension BorrowView:
    func len(self: BorrowView) -> Int:
        return self.value.len

impl BorrowView: ViewLike

func bad(xs: borrow []u8) -> BorrowView:
    return BorrowView{value: xs.window(0, 1).borrow()}

func main() -> Int:
    return 0
`, ("aggregate 'BorrowView' contains borrowed slice field 'value' " +
		"that cannot escape through owned return"))
}

func TestMemoryIdealV3BorrowedInterfaceGlobalStorageRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct BorrowView:
    value: []u8

protocol ViewLike:
    func len(self: BorrowView) -> Int

extension BorrowView:
    func len(self: BorrowView) -> Int:
        return self.value.len

impl BorrowView: ViewLike

var saved: BorrowView

func bad(xs: borrow []u8) -> Int:
    saved = BorrowView{value: xs.window(0, 1).borrow()}
    return 0
`, "contains borrowed slice field 'value' that cannot be stored in global")
}

func TestMemoryIdealV3UnknownDynamicDispatchConservativeRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Vec2:
    x: Int

protocol Drawable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Drawable

func render<T: Drawable>(value: T) -> Int:
    return Drawable.draw(value)

func main() -> Int:
    return render(Vec2(x: 1))
`, "unknown function 'Drawable.draw'")

	testkit.RequireFileCheckErrorContains(t, `
struct Vec2:
    x: Int

protocol Drawable:
    func draw(self: Vec2) -> Int

func main() -> Int:
    let value: Drawable = Vec2(x: 1)
    return 0
`, "unknown type 'Drawable'")
}

func TestBorrowedViewGlobalStorageRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var saved: []i32

func stash(xs: []i32) -> Int:
    saved = xs.window(0, 1).borrow()
    return 0
`, "borrowed local 'xs' cannot escape via global assignment to 'saved'")
}

func TestBorrowedActorSendRejectedUnlessCopied(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum Msg:
    case bytes([]u8)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(1)
    return core.send_typed(core.self(), Msg.bytes(xs.borrow()))
`, "cannot send borrowed view across actor boundary; use .copy()")

	testkit.RequireCheckOK(t, `
enum Msg:
    case bytes([]u8)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(1)
    return core.send_typed(core.self(), Msg.bytes(xs.copy()))
`)

	testkit.RequireCheckErrorContains(t, `
enum Msg:
    case text(String)

func main() -> Int
uses actors, alloc, mem:
    let text: String = "remote".copy()
    return core.send_typed(core.self(), Msg.text(text.borrow()))
`, "cannot send borrowed view across actor boundary; use .copy()")

	testkit.RequireCheckOK(t, `
enum Msg:
    case text(String)

func main() -> Int
uses actors, alloc, mem:
    let text: String = "remote".copy()
    return core.send_typed(core.self(), Msg.text(text.borrow().copy()))
`)

	testkit.RequireCheckOK(t, `
enum Msg:
    case text(String)

func main() -> Int
uses actors, alloc, mem:
    return core.send_typed(core.self(), Msg.text("remote".copy()))
`)
}

func TestBorrowedTaskBoundaryTypedErrorPayloadRejected(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum TaskErr:
    case bytes([]u8)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)
`, "typed task error payload must be sendable across task boundary")

	testkit.RequireCheckErrorContains(t, `
enum TaskErr:
    case text(String)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)
`, "typed task error payload must be sendable across task boundary")

	testkit.RequireCheckOK(t, `
enum TaskErr:
    case code(Int)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses alloc, mem, runtime:
    var xs: []u8 = make_u8(1)
    xs[0] = 42
    let copied: []u8 = xs.borrow().copy()
    if copied[0] != 42:
        return 1
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    let status: Int = catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.code(value):
        value
    return status
`)
}

func TestCopyResultMayEscapeSafely(t *testing.T) {
	testkit.RequireCheckOK(t, `
func owned_copy(xs: []i32) -> []i32
uses alloc, mem:
    return xs.window(0, 1).copy()

func main() -> Int:
    return 0
`)
}

func TestBorrowCopyBuildOnlyTargets(t *testing.T) {
	src := `func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    xs[0] = 1
    xs[1] = 2
    let copied: []u8 = xs.window(0, 2).copy()
    var dst: []u8 = make_u8(2)
    return copied.copy_into(dst)
`
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(tmp, "app-"+target+".wasm")
		if _, err := compiler.BuildFileWithStatsOpt(
			srcPath,
			outPath,
			target,
			compiler.BuildOptions{Jobs: 1},
		); err != nil {
			t.Fatalf("build %s: %v", target, err)
		}
	}
	for _, target := range []string{"linux-x86"} {
		outPath := filepath.Join(tmp, "app-"+target+".tobj")
		if _, err := compiler.BuildFileWithStatsOpt(
			srcPath,
			outPath,
			target,
			compiler.BuildOptions{Jobs: 1, Emit: compiler.EmitLibrary},
		); err != nil {
			t.Fatalf("build %s library: %v", target, err)
		}
	}
}

// ---- borrow_escape_matrix_test.go ----

func TestBorrowEscapeMatrixRejectsGenericBorrowAggregateGlobalFieldTarget(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Box<T>:
    value: T

struct Slot:
    box: Box<[]u8>

var saved: Slot

func stash(xs: borrow []u8) -> Int:
    saved.box = Box<[]u8>{value: xs.window(1, 2).borrow()}
    return 0
`, "contains borrowed slice field 'value' that cannot be stored in global")
}

func TestBorrowEscapeMatrixRejectsCrossModuleGenericBorrowAggregateGlobalFieldTarget(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Box<T>:
    value: T

pub struct Slot:
    box: Box<[]u8>
`,
		"app/main.t4": `module app.main
import lib.model as model

var saved: model.Slot

func stash(xs: borrow []u8) -> Int:
    saved.box = model.Box<[]u8>{value: xs.window(1, 2).borrow()}
    return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"contains borrowed slice field 'value' that cannot be stored in global",
	)
}

func TestBorrowEscapeMatrixAllowsCopiedGenericAggregateGlobalFieldTarget(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
struct Box<T>:
    value: T

struct Slot:
    box: Box<[]u8>

var saved: Slot

func stash(xs: borrow []u8) -> Int
uses alloc, mem:
    saved.box = Box<[]u8>{value: xs.window(1, 2).copy()}
    return 0

func main() -> Int:
    return 0
`)
}

// ---- callable_borrow_escape_test.go ----

func TestCallableBorrowEscapeRejectsBorrowedStringCaptureGlobalStore(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func install(text: borrow String) -> Int:
    let view: String = text.window(0, 1).borrow()
    let captured: ptr = fn(x: Int) -> Int:
        return x + view.len
    cb = captured
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'view' cannot escape via function capture")
}
