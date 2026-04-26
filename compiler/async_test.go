package compiler

import (
	"strings"
	"testing"
)

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
    let task: Int = core.task_spawn_i32("worker")
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

func main() -> Int:
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

func main() -> Int
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
