package compiler

import (
	"strings"
	"testing"
)

func TestDeferRunsLIFOAndPreservesReturnValue(t *testing.T) {
	src := `func main() -> Int
uses io:
    defer:
        print("a")
    defer:
        print("b")
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "ba" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want 42", exitCode)
	}
}

func TestDeferRunsOnNestedReturnBeforeOuterCleanup(t *testing.T) {
	src := `func main() -> Int
uses io:
    defer:
        print("outer")
    if true:
        defer:
            print("inner")
        return 7
    return 1
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "innerouter" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want 7", exitCode)
	}
}

func TestDeferRunsWhenLoopScopeExitsByBreak(t *testing.T) {
	src := `func main() -> Int
uses io:
    while true:
        defer:
            print("loop")
        break
    print("after")
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "loopafter" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
}

func TestDeferRunsWhenLoopScopeExitsByContinue(t *testing.T) {
	src := `func main() -> Int
uses io:
    var i: Int = 0
    while i < 2:
        i = i + 1
        defer:
            print("tick")
        continue
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "ticktick" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
}

func TestDeferRunsBeforeThrowReturn(t *testing.T) {
	src := `enum E:
    case bad

func fail() -> Int throws E
uses io:
    defer:
        print("cleanup")
    throw E.bad

func main() -> Int
uses io:
    return catch fail():
    case E.bad:
        3
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "cleanup" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code = %d, want 3", exitCode)
	}
}

func TestDeferRunsBeforeScopedIslandAutoFree(t *testing.T) {
	src := `func main() -> Int
uses alloc, islands, io, mem:
    island(64) as isl:
        var msg: []u8 = core.island_make_u8(isl, 2)
        msg[0] = 79
        msg[1] = 10
        defer:
            print(msg)
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "O\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
}

func TestDeferRunsWhenCanceledTaskCheckpoints(t *testing.T) {
	src := `func worker() -> Int
uses io, runtime:
    defer:
        print("cleanup")
    let group: task.group = core.task_group_current()
    let _canceledGroup: task.group = core.task_group_cancel(group)
    let checkpoint: task.error = core.task_checkpoint()
    if checkpoint != 0:
        return 5
    return 9

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.error != 0:
        return 80 + result.error
    return result.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "cleanup" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code = %d, want 5", exitCode)
	}
}

func TestDeferRejectsReturnInsideCleanup(t *testing.T) {
	src := []byte(`func main() -> Int:
    defer:
        return 1
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "return is not allowed in defer") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeferRejectsBreakToOuterLoopInsideCleanup(t *testing.T) {
	src := []byte(`func main() -> Int:
    while true:
        defer:
            break
        return 0
    return 1
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "break is not allowed in defer") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeferRejectsLaterConsumeOfCapturedValue(t *testing.T) {
	src := []byte(`func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let a: Int = 1
    defer:
        let _captured: Int = a
    let b: Int = take(a)
    return b
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "defer cleanup captures value 'a'") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeferRejectsLaterActorTransferOfCapturedIsland(t *testing.T) {
	src := []byte(`enum MoveMsg:
    case take(island)

func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        var isl: island = core.island_new(16)
        defer:
            let _buf: []u8 = core.island_make_u8(isl, 1)
        let _sent: Int = core.send_typed(peer, MoveMsg.take(isl))
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "defer cleanup captures value 'isl'") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeferBodyConsumeDoesNotPoisonPreCleanupReturn(t *testing.T) {
	src := []byte(`func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let a: Int = 1
    defer:
        let _done: Int = take(a)
    return a
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestDeferRejectsThrowInsideCleanup(t *testing.T) {
	src := []byte(`enum E:
    case bad

func main() -> Int throws E:
    defer:
        throw E.bad
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "throw is not allowed in defer") {
		t.Fatalf("error = %v", err)
	}
}
