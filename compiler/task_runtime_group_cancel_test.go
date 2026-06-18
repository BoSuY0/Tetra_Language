package compiler

import (
	"strings"
	"testing"
	"time"
)

func TestTaskGroupLowersToRuntimeCalls(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    group = core.task_group_cancel(group)
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    return result.error
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	for _, name := range []string{
		"__tetra_task_group_open",
		"__tetra_task_spawn_group_i32",
		"__tetra_task_group_cancel",
		"__tetra_task_group_close",
	} {
		if !hasIRCall(mainFn, name) {
			t.Fatalf("main does not call %s: %#v", name, mainFn.Instrs)
		}
	}
}

func TestTaskGroupCancelAfterSpawnBeforeJoinBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int:
    return 77

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    group = core.task_group_cancel(group)
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.value != 0:
        return result.value
    return result.error
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want canceled status 1", exitCode)
	}
}

func TestTaskGroupCancelWakesJoinUntilBeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(10)
    return 99

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let _delay: Int = core.sleep_ms(2)
    group = core.task_group_cancel(group)
    let result: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(20))
    let _closed: Int = core.task_group_close(group)
    if result.value != 0:
        return result.value
    if result.error != 1:
        return 30 + result.error
    let now: Int = core.time_now_ms()
    if now != 2:
        return 50 + now
    return result.error
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; canceled grouped task should wake join_until before deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want canceled task error 1 at logical time 2", exitCode)
	}
}

func TestTaskGroupCancelWhileActorWaitsOnJoinReturnsCanceledBuildAndRun(t *testing.T) {
	src := `
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 99

func canceller() -> Int
uses runtime:
    let _delay: Int = core.sleep_ms(2)
    let group: task.group = core.task_group_current()
    let _canceled: task.group = core.task_group_cancel(group)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let slow_task: task.i32 = core.task_spawn_group_i32(group, "slow")
    let _cancel_task: task.i32 = core.task_spawn_group_i32(group, "canceller")
    let result: task.result_i32 = core.task_join_result_i32(slow_task)
    let _closed: Int = core.task_group_close(group)
    if result.value != 0:
        return result.value
    if result.error != 1:
        return 30 + result.error
    let now: Int = core.time_now_ms()
    if now != 2:
        return 50 + now
    return result.error
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task group cancel should wake actor waiting on task_join_result_i32")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want join_result_i32 canceled error 1 while caller was already waiting", exitCode)
	}
}

func TestTaskGroupCancelWhileActorWaitsOnJoinI32WakesWithZeroValueBuildAndRun(t *testing.T) {
	src := `
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 99

func canceller() -> Int
uses runtime:
    let _delay: Int = core.sleep_ms(2)
    let group: task.group = core.task_group_current()
    let _canceled: task.group = core.task_group_cancel(group)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let slow_task: task.i32 = core.task_spawn_group_i32(group, "slow")
    let _cancel_task: task.i32 = core.task_spawn_group_i32(group, "canceller")
    let joined: Int = core.task_join_i32(slow_task)
    let _closed: Int = core.task_group_close(group)
    if joined != 0:
        return joined
    let now: Int = core.time_now_ms()
    if now != 2:
        return 50 + now
    return 1
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task group cancel should wake actor waiting on task_join_i32")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want task_join_i32 to wake at cancellation time with raw zero value", exitCode)
	}
}

func TestTaskGroupCancelWakesSelect2BeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 99

func canceller() -> Int
uses runtime:
    let _delay: Int = core.sleep_ms(2)
    let group: task.group = core.task_group_current()
    let _canceled: task.group = core.task_group_cancel(group)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let slow_task: task.i32 = core.task_spawn_group_i32(group, "slow")
    let _cancel_task: task.i32 = core.task_spawn_group_i32(group, "canceller")
    let selected: task.result_i32 = core.select2_i32(slow_task, core.deadline_ms(100))
    let _closed: Int = core.task_group_close(group)
    if selected.value != 0:
        return selected.value
    if selected.error != 1:
        return 30 + selected.error
    let now: Int = core.time_now_ms()
    if now != 2:
        return 50 + now
    return selected.error
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task group cancel should wake select2_i32 before deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want select2_i32 canceled error 1 before deadline", exitCode)
	}
}

func TestTaskGroupJoinUntilTimeoutThenCancelFinalJoinBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(10)
    return 99

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let early: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(3))
    if early.error != 2:
        return 20 + early.error
    if early.value != 0:
        return 40 + early.value
    let atTimeout: Int = core.time_now_ms()
    if atTimeout != 3:
        return 60 + atTimeout
    group = core.task_group_cancel(group)
    let final: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if final.value != 0:
        return final.value
    if final.error != 1:
        return 80 + final.error
    return final.error
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; final join should observe cancellation after prior timeout")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want canceled final join after timeout", exitCode)
	}
}

func TestTaskGroupCurrentLowersToRuntimeCall(t *testing.T) {
	src := []byte(`
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    if group == group:
        return 0
    return 1
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "__tetra_task_group_current") {
		t.Fatalf("main does not call __tetra_task_group_current: %#v", mainFn.Instrs)
	}
}

func TestTaskCancellationCheckpointLowersToRuntimeCalls(t *testing.T) {
	src := []byte(`
func main() -> Int
uses runtime:
    let canceled: Int = core.task_is_canceled()
    let checkpoint: task.error = core.task_checkpoint()
    return canceled + checkpoint
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	for _, name := range []string{"__tetra_task_is_canceled", "__tetra_task_checkpoint"} {
		if !hasIRCall(mainFn, name) {
			t.Fatalf("main does not call %s: %#v", name, mainFn.Instrs)
		}
	}
}

func TestTimeRuntimeBuiltinsRequireRuntimeUse(t *testing.T) {
	src := []byte(`
func main() -> Int:
    return core.time_now_ms()
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
