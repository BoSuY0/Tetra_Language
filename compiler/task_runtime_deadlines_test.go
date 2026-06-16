package compiler

import (
	"testing"
	"time"
)

func TestTimeRuntimeBuiltinsLowerToRuntimeCalls(t *testing.T) {
	src := []byte(`
func main() -> Int
uses runtime:
    let start: Int = core.time_now_ms()
    let err: Int = core.sleep_ms(5)
    let untilErr: Int = core.sleep_until(core.deadline_ms(6))
    let deadline: Int = core.deadline_ms(7)
    return start + err + untilErr + deadline
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
	for _, name := range []string{"__tetra_time_now_ms", "__tetra_sleep_ms", "__tetra_sleep_until_ms", "__tetra_deadline_ms"} {
		if !hasIRCall(mainFn, name) {
			t.Fatalf("main does not call %s: %#v", name, mainFn.Instrs)
		}
	}
}

func TestDeadlineAwareRuntimeBuiltinsCheckAndLower(t *testing.T) {
	src := []byte(`
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return 42

func main() -> Int
uses actors, runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let _sleepUntil: Int = core.sleep_until(core.deadline_ms(1))
    let joined: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(5))
    let recv: actor.recv_result_i32 = core.recv_until(core.deadline_ms(6))
    return joined.value + joined.error + recv.value + recv.error
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
	for _, name := range []string{"__tetra_sleep_until_ms", "__tetra_task_join_until_i32", "__tetra_actor_recv_until"} {
		if !hasIRCall(mainFn, name) {
			t.Fatalf("main does not call %s: %#v", name, mainFn.Instrs)
		}
	}
}

func TestWaitCompositionRuntimeBuiltinsCheckAndLower(t *testing.T) {
	src := []byte(`
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return 42

func main() -> Int
uses actors, runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let polled: task.result_i32 = core.task_poll_i32(task)
    let yielded: Int = core.yield()
    let ready: Bool = core.timer_ready(core.deadline_ms(0))
    let selected: task.result_i32 = core.select2_i32(task, core.deadline_ms(5))
    let recv: actor.recv_result_i32 = core.recv_poll()
    let msg: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(6))
    if ready:
        return polled.error + yielded + selected.value + recv.error + msg.error
    return 99
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
		"__tetra_task_poll_i32",
		"__tetra_actor_yield_now",
		"__tetra_timer_ready_ms",
		"__tetra_task_join_until_i32",
		"__tetra_actor_recv_poll",
		"__tetra_actor_recv_msg_until",
	} {
		if !hasIRCall(mainFn, name) {
			t.Fatalf("main does not call %s: %#v", name, mainFn.Instrs)
		}
	}
}

func TestTimeRuntimeLogicalClockBuildAndRun(t *testing.T) {
	src := `
func main() -> Int
uses runtime:
    let start: Int = core.time_now_ms()
    let err: Int = core.sleep_ms(5)
    let after: Int = core.time_now_ms()
    let deadline: Int = core.deadline_ms(7)
    return (after - start) + deadline + err
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 17 {
		t.Fatalf("exit code = %d, want logical clock result 17", exitCode)
	}
}

func TestSleepUntilUsesAbsoluteDeadlineBuildAndRun(t *testing.T) {
	src := `
func main() -> Int
uses runtime:
    let start: Int = core.time_now_ms()
    let deadline: Int = core.deadline_ms(7)
    let err: Int = core.sleep_until(deadline)
    let after: Int = core.time_now_ms()
    let immediate: Int = core.sleep_until(deadline)
    let finalTime: Int = core.time_now_ms()
    if err != 0:
        return 20 + err
    if immediate != 0:
        return 30 + immediate
    return (after - start) * 10 + (finalTime - after)
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 70 {
		t.Fatalf("exit code = %d, want sleep_until absolute deadline result 70", exitCode)
	}
}

func TestTaskSleepTimersWakeInDeadlineOrderBuildAndRun(t *testing.T) {
	src := `
func slow() -> Int
uses runtime:
    let _err: Int = core.sleep_ms(5)
    return core.time_now_ms()

func fast() -> Int
uses runtime:
    let _err: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let slowTask: task.i32 = core.task_spawn_i32("slow")
    let fastTask: task.i32 = core.task_spawn_i32("fast")
    let _mainSleep: Int = core.sleep_ms(10)
    let fastValue: Int = core.task_join_i32(fastTask)
    let slowValue: Int = core.task_join_i32(slowTask)
    return fastValue * 10 + slowValue
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 25 {
		t.Fatalf("exit code = %d, want fast wake 2 and slow wake 5", exitCode)
	}
}

func TestTaskJoinUntilTimesOutThenFinalJoinCompletesBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(10)
    return 99

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let early: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(3))
    if early.error != 2:
        return 20 + early.error
    if early.value != 0:
        return 40 + early.value
    let afterTimeout: Int = core.time_now_ms()
    if afterTimeout != 3:
        return 60 + afterTimeout
    let final: task.result_i32 = core.task_join_result_i32(task)
    if final.error != 0:
        return 80 + final.error
    if final.value != 99:
        return final.value
    return core.time_now_ms()
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task_join_until_i32 should wake on timeout deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 10 {
		t.Fatalf("exit code = %d, want final join logical time 10 after timeout", exitCode)
	}
}

func TestTaskJoinUntilReturnsCompletedTaskBeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(5))
    if result.error != 0:
        return 20 + result.error
    return result.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task_join_until_i32 should wake when task completes")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want joined worker value before deadline 2", exitCode)
	}
}

func TestTaskPollReturnsTimeoutUntilTaskCompletesBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(4)
    return 77

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let early: task.result_i32 = core.task_poll_i32(task)
    if early.error != 2:
        return 20 + early.error
    if early.value != 0:
        return 40 + early.value
    let _sleep: Int = core.sleep_ms(5)
    let late: task.result_i32 = core.task_poll_i32(task)
    if late.error != 0:
        return 60 + late.error
    return late.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task_poll_i32 must not block")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 77 {
		t.Fatalf("exit code = %d, want completed poll value 77", exitCode)
	}
}

func TestYieldAllowsReadyTaskToRunBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses actors:
    let _sent: Int = core.send(core.sender(), 9)
    return 0

func main() -> Int
uses actors, runtime:
    let _task: task.i32 = core.task_spawn_i32("worker")
    let before: actor.recv_result_i32 = core.recv_poll()
    if before.error != 2:
        return 20 + before.error
    let yielded: Int = core.yield()
    if yielded != 0:
        return 40 + yielded
    let after: actor.recv_result_i32 = core.recv_poll()
    if after.error != 0:
        return 60 + after.error
    return after.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; yield should resume after another ready actor runs")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 9 {
		t.Fatalf("exit code = %d, want recv_poll value after yield", exitCode)
	}
}

func TestTimerReadyAndSelect2TaskTimerBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return 33

func main() -> Int
uses runtime:
    let deadline: Int = core.deadline_ms(2)
    if core.timer_ready(deadline):
        return 10
    let task: task.i32 = core.task_spawn_i32("worker")
    let selected: task.result_i32 = core.select2_i32(task, deadline)
    if selected.error != 2:
        return 20 + selected.error
    if !core.timer_ready(deadline):
        return 40
    let final: task.result_i32 = core.task_join_result_i32(task)
    if final.error != 0:
        return 60 + final.error
    return final.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; select2_i32 should wake on timer deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 33 {
		t.Fatalf("exit code = %d, want final task value after select timeout", exitCode)
	}
}

func TestDocumentedTaskTimeBuiltinSelfHostParity(t *testing.T) {
	tests := []struct {
		name string
		src  string
		exit int
	}{
		{
			name: "sleep",
			src: `
func main() -> Int
uses runtime:
    let start: Int = core.time_now_ms()
    let err: Int = core.sleep_ms(3)
    if err != 0:
        return 20 + err
    let after: Int = core.time_now_ms()
    let untilErr: Int = core.sleep_until(core.deadline_ms(2))
    if untilErr != 0:
        return 40 + untilErr
    let finalTime: Int = core.time_now_ms()
    return (after - start) * 10 + (finalTime - after)
`,
			exit: 32,
		},
		{
			name: "deadline_join",
			src: `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(5))
    if result.error != 0:
        return 20 + result.error
    return result.value
`,
			exit: 2,
		},
		{
			name: "poll",
			src: `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return 31

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let early: task.result_i32 = core.task_poll_i32(task)
    if early.error != 2:
        return 20 + early.error
    if early.value != 0:
        return 40 + early.value
    let _sleep: Int = core.sleep_ms(3)
    let late: task.result_i32 = core.task_poll_i32(task)
    if late.error != 0:
        return 60 + late.error
    return late.value
`,
			exit: 31,
		},
		{
			name: "select2_timer",
			src: `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return 33

func main() -> Int
uses runtime:
    let deadline: Int = core.deadline_ms(2)
    let task: task.i32 = core.task_spawn_i32("worker")
    let selected: task.result_i32 = core.select2_i32(task, deadline)
    if selected.error != 2:
        return 20 + selected.error
    if !core.timer_ready(deadline):
        return 40
    let final: task.result_i32 = core.task_join_result_i32(task)
    if final.error != 0:
        return 60 + final.error
    return final.value
`,
			exit: 33,
		},
	}

	runtimes := []struct {
		name string
		mode RuntimeMode
	}{
		{name: "builtin", mode: RuntimeBuiltin},
		{name: "selfhost", mode: RuntimeSelfHost},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var want *struct {
				stdout string
				exit   int
			}
			for _, rt := range runtimes {
				rt := rt
				t.Run(rt.name, func(t *testing.T) {
					stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, tt.src, BuildOptions{Runtime: rt.mode}, 250*time.Millisecond)
					if timedOut {
						t.Fatalf("program timed out for runtime %s", rt.name)
					}
					got := struct {
						stdout string
						exit   int
					}{stdout: stdout, exit: exitCode}
					if want == nil {
						want = &got
					} else if got != *want {
						t.Fatalf("runtime parity mismatch: got=%#v want=%#v", got, *want)
					}
					if stdout != "" {
						t.Fatalf("stdout mismatch: %q", stdout)
					}
					if exitCode != tt.exit {
						t.Fatalf("exit code = %d, want %d", exitCode, tt.exit)
					}
				})
			}
		})
	}
}

func TestWaitCompositionRuntimeSelfHostBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(1)
    let _sent: Int = core.send_msg(core.sender(), 13, 5)
    return 21

func main() -> Int
uses actors, runtime:
    let deadline: Int = core.deadline_ms(1)
    if core.timer_ready(deadline):
        return 10
    let task: task.i32 = core.task_spawn_i32("worker")
    let early: task.result_i32 = core.task_poll_i32(task)
    if early.error != 2:
        return 20 + early.error
    let yielded: Int = core.yield()
    if yielded != 0:
        return 40 + yielded
    let empty: actor.recv_result_i32 = core.recv_poll()
    if empty.error != 2:
        return 60 + empty.error
    let msg: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(3))
    if msg.error != 0:
        return 80 + msg.error
    if msg.value != 13:
        return 100 + msg.value
    if msg.tag != 5:
        return 120 + msg.tag
    if !core.timer_ready(deadline):
        return 140
    let selected: task.result_i32 = core.select2_i32(task, core.deadline_ms(5))
    if selected.error != 0:
        return 160 + selected.error
    return selected.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{Runtime: RuntimeSelfHost}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; selfhost wait composition should wake on message/task events")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 21 {
		t.Fatalf("exit code = %d, want selected task value 21", exitCode)
	}
}

func TestDeadlineAwareRuntimeSelfHostBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let _sleepUntil: Int = core.sleep_until(core.deadline_ms(1))
    let task: task.i32 = core.task_spawn_i32("worker")
    let early: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(2))
    if early.error != 2:
        return 20 + early.error
    let final: task.result_i32 = core.task_join_result_i32(task)
    if final.error != 0:
        return 40 + final.error
    return final.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{Runtime: RuntimeSelfHost}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; selfhost deadline-aware waits should advance logical time")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 6 {
		t.Fatalf("exit code = %d, want selfhost final wake time 6", exitCode)
	}
}

func TestTaskJoinWaitStateAllowsSleepingTaskDeadlineBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task_join_i32 should wait without starving sleeping task deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code = %d, want joined sleeping task wake time 5", exitCode)
	}
}

func TestTaskJoinWaitStateSelfHostRuntimeBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{Runtime: RuntimeSelfHost}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; selfhost task_join_i32 should park while child sleeps")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code = %d, want selfhost joined sleeping task wake time 5", exitCode)
	}
}

func TestTaskDeadlineBuiltinSelfHostParityBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(5))
    if result.error != 0:
        return 20 + result.error
    return result.value
`
	var want *struct {
		stdout string
		exit   int
	}
	for _, tc := range []struct {
		name string
		rt   RuntimeMode
	}{
		{name: "builtin", rt: RuntimeBuiltin},
		{name: "selfhost", rt: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{Runtime: tc.rt}, 250*time.Millisecond)
			if timedOut {
				t.Fatalf("program timed out; deadline join should complete under runtime %d", tc.rt)
			}
			got := struct {
				stdout string
				exit   int
			}{stdout: stdout, exit: exitCode}
			if want == nil {
				want = &got
			} else if got != *want {
				t.Fatalf("runtime parity mismatch: got=%#v want=%#v", got, *want)
			}
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != 2 {
				t.Fatalf("exit code = %d, want deadline join value 2", exitCode)
			}
		})
	}
}

func TestTaskJoinResultWaitStateAllowsSleepingTaskDeadlineBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(3)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    if result.error != 0:
        return 20 + result.error
    return result.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task_join_result_i32 should wait without starving sleeping task deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code = %d, want joined result wake time 3", exitCode)
	}
}

func TestTaskJoinTypedWaitStateAllowsSleepingThrowBuildAndRun(t *testing.T) {
	src := `
enum WaitErr:
    case boom(Int)
    case stopped

func worker() -> Int throws WaitErr
uses runtime:
    let _sleep: Int = core.sleep_ms(4)
    throw WaitErr.boom(core.time_now_ms())

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<WaitErr>("worker")
    return catch core.task_join_i32_typed<WaitErr>(task):
    case WaitErr.boom(code):
        code
    case WaitErr.stopped:
        99
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; typed task join should wait without starving sleeping task deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 4 {
		t.Fatalf("exit code = %d, want typed join throw payload 4", exitCode)
	}
}

func TestRuntimeSchedulerCanceledSleepingTaskReturnsCancelBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(10)
    let checkpoint: task.error = core.task_checkpoint()
    if checkpoint != 0:
        return core.time_now_ms()
    return 99

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let _delay: Int = core.sleep_ms(2)
    group = core.task_group_cancel(group)
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.value != 0:
        return result.value
    if result.error != 1:
        return 20 + result.error
    let now: Int = core.time_now_ms()
    if now != 2:
        return 40 + now
    return result.error
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want canceled task error 1 at logical time 2", exitCode)
	}
}
