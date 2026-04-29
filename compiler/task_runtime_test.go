package compiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"tetra_language/compiler/internal/ir"
)

func TestTaskSpawnI32CollectsRuntimeEntry(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    if task.value != 1:
        return 60 + task.value
    if task.error != 0:
        return 70 + task.error
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
	used, entries, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collectActorEntries: %v", err)
	}
	if !used {
		t.Fatalf("task runtime was not collected")
	}
	if len(entries) != 2 || entries[0] != "main" || entries[1] != "worker" {
		t.Fatalf("entries = %#v, want [main worker]", entries)
	}
}

func TestRequiredTypedTaskRuntimeSymbolsSupportStagedSlotsUpToEight(t *testing.T) {
	base := map[string]struct{}{}
	for _, name := range requiredTypedTaskRuntimeSymbols(4) {
		base[name] = struct{}{}
	}
	for _, slots := range []int{2, 3, 4} {
		name := "__tetra_task_join_typed_" + strconv.Itoa(slots)
		if _, ok := base[name]; !ok {
			t.Fatalf("missing required typed task runtime symbol %q", name)
		}
	}
	if _, ok := base["__tetra_task_result_get"]; ok {
		t.Fatalf("non-staged symbol set should not require __tetra_task_result_get")
	}

	got := map[string]struct{}{}
	for _, name := range requiredTypedTaskRuntimeSymbols(8) {
		got[name] = struct{}{}
	}
	for _, slots := range []int{2, 3, 4, 5, 6, 7, 8} {
		name := "__tetra_task_join_typed_" + strconv.Itoa(slots)
		if _, ok := got[name]; !ok {
			t.Fatalf("missing required typed task runtime symbol %q", name)
		}
	}
	if _, ok := got["__tetra_task_result_get"]; !ok {
		t.Fatalf("missing required staged typed task runtime symbol %q", "__tetra_task_result_get")
	}
}

func TestTaskSpawnI32LowersToRuntimeSpawn(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 41

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
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "__tetra_task_spawn_i32") {
		t.Fatalf("main does not call __tetra_task_spawn_i32: %#v", mainFn.Instrs)
	}
	if !hasIRCall(mainFn, "__tetra_task_join_i32") {
		t.Fatalf("main does not call __tetra_task_join_i32: %#v", mainFn.Instrs)
	}
	if hasIRCall(mainFn, "worker") {
		t.Fatalf("main still calls worker directly during task spawn: %#v", mainFn.Instrs)
	}
}

func TestTaskSpawnI32TypedPayloadLowersToRuntimeWrapper(t *testing.T) {
	src := []byte(`
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
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	used, entries, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collectActorEntries: %v", err)
	}
	if !used {
		t.Fatalf("typed task runtime was not collected")
	}
	if hasString(entries, "worker") {
		t.Fatalf("typed task entries should use a wrapper, got %#v", entries)
	}
	if !hasPrefix(entries, "__tetra_task_typed_") {
		t.Fatalf("typed task wrapper entry missing: %#v", entries)
	}

	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "__tetra_task_spawn_i32") {
		t.Fatalf("main does not call __tetra_task_spawn_i32: %#v", mainFn.Instrs)
	}
	if !hasIRCall(mainFn, "__tetra_task_join_typed_4") {
		t.Fatalf("main does not call __tetra_task_join_typed_4: %#v", mainFn.Instrs)
	}
	if hasIRCall(mainFn, "worker") {
		t.Fatalf("main still calls worker directly during typed task spawn: %#v", mainFn.Instrs)
	}
	if !hasIRFuncPrefix(irProg.Funcs, "__tetra_task_typed_") {
		var names []string
		for _, fn := range irProg.Funcs {
			names = append(names, fn.Name)
		}
		t.Fatalf("typed task wrapper IR function missing; funcs=%#v", names)
	}
}

func TestTaskSpawnI32TypedStagedSlotsFiveLowersToRuntimeStagedPath(t *testing.T) {
	src := []byte(`
enum TaskErr:
    case boom(Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(5, 7)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b):
        a + b
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
	if !hasIRCall(mainFn, "__tetra_task_join_typed_5") {
		t.Fatalf("main does not call __tetra_task_join_typed_5: %#v", mainFn.Instrs)
	}
	if !hasIRCall(mainFn, "__tetra_task_result_get") {
		t.Fatalf("main does not call __tetra_task_result_get in staged path: %#v", mainFn.Instrs)
	}
}

func TestTaskSpawnI32TypedStagedSlotBuildAndRunSmoke(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		exitCode int
	}{
		{
			name: "slots_5",
			src: `
enum TaskErr:
    case boom(Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(9, 12)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b):
        a + b
`,
			exitCode: 21,
		},
		{
			name: "slots_8",
			src: `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
`,
			exitCode: 15,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stdout, exitCode := buildAndRunWithOptions(t, tc.src, BuildOptions{})
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != tc.exitCode {
				t.Fatalf("exit code = %d, want %d", exitCode, tc.exitCode)
			}
		})
	}
}

func TestTaskSpawnI32TypedRejectsExplicitSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "main")
	if err := os.WriteFile(srcPath, []byte(`
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(42)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Runtime: RuntimeSelfHost})
	if err == nil {
		t.Fatalf("expected explicit selfhost typed task rejection")
	}
	if !strings.Contains(err.Error(), "self-host runtime does not support typed task handles") {
		t.Fatalf("error = %v", err)
	}
}

func TestTaskSpawnI32TypedStagedSlotsEightNestedSpawnAutoAndBuiltinRuntimeParity(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)

func child() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func worker() -> Int throws TaskErr
uses runtime:
    let child_task = core.task_spawn_i32_typed<TaskErr>("child")
    return catch core.task_join_i32_typed<TaskErr>(child_task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        90 + a + b + c + d + e
`
	for _, tc := range []struct {
		name string
		rt   RuntimeMode
	}{
		{name: "auto", rt: RuntimeAuto},
		{name: "builtin", rt: RuntimeBuiltin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: tc.rt})
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != 15 {
				t.Fatalf("exit code = %d, want 15", exitCode)
			}
		})
	}
}

func TestTaskSpawnI32TypedStagedSlotsRejectsExplicitSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "main")
	if err := os.WriteFile(srcPath, []byte(`
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Runtime: RuntimeSelfHost})
	if err == nil {
		t.Fatalf("expected explicit selfhost staged typed task rejection")
	}
	if !strings.Contains(err.Error(), "self-host runtime does not support typed task handles") {
		t.Fatalf("error = %v", err)
	}
}

func TestTaskSpawnI32TypedRejectsHandleSlotsAboveEightEarly(t *testing.T) {
	requireCheckErrorContains(t, `
enum TaskErr:
    case huge(Int, Int, Int, Int, Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.huge(1, 2, 3, 4, 5, 6)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.huge(a, b, c, d, e, f):
        a + b + c + d + e + f
`, "typed task supports at most 8 slots")
}

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

func buildAndRunWithOptionsTimeout(t *testing.T, src string, opt BuildOptions, timeout time.Duration) (string, int, bool) {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, outPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return out.String(), -1, true
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out.String(), exitErr.ProcessState.ExitCode(), false
		}
		t.Fatalf("run binary: %v", err)
	}
	return out.String(), cmd.ProcessState.ExitCode(), false
}

func TestTaskCancellationCheckpointUngroupedTaskBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let canceled: Int = core.task_is_canceled()
    if canceled != 0:
        return 40 + canceled
    let checkpoint: task.error = core.task_checkpoint()
    if checkpoint != 0:
        return 50 + checkpoint
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    if result.error != 0:
        return 80 + result.error
    return result.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want ungrouped checkpoint result 7", exitCode)
	}
}

func TestTaskCancellationCheckpointSeesSelfCanceledGroupBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    let _canceledGroup: task.group = core.task_group_cancel(group)
    let canceled: Int = core.task_is_canceled()
    let checkpoint: task.error = core.task_checkpoint()
    return canceled * 10 + checkpoint

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
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 11 {
		t.Fatalf("exit code = %d, want canceled checkpoint result 11", exitCode)
	}
}

func TestTaskCancellationCheckpointInheritedByNestedChildBuildAndRun(t *testing.T) {
	src := `
func child() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    let _canceledGroup: task.group = core.task_group_cancel(group)
    let canceled: Int = core.task_is_canceled()
    let checkpoint: task.error = core.task_checkpoint()
    return canceled * 10 + checkpoint

func worker() -> Int
uses runtime:
    let childTask: task.i32 = core.task_spawn_i32("child")
    let childResult: task.result_i32 = core.task_join_result_i32(childTask)
    if childResult.error != 0:
        return 70 + childResult.error
    return childResult.value

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
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 11 {
		t.Fatalf("exit code = %d, want nested canceled checkpoint result 11", exitCode)
	}
}

func TestTaskGroupCurrentInheritedByChildTaskBuildAndRun(t *testing.T) {
	src := `
func leaf() -> Int:
    return 1

func child() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    return core.task_group_status(group)

func worker() -> Int
uses runtime:
    let ownGroup: task.group = core.task_group_current()
    let ownStatus: Int = core.task_group_status(ownGroup)
    if ownStatus != 1:
        return 70 + ownStatus
    let childTask: task.i32 = core.task_spawn_i32("child")
    let childResult: task.result_i32 = core.task_join_result_i32(childTask)
    if childResult.error != 0:
        return 90 + childResult.error
    return childResult.value

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
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want inherited open group status 1", exitCode)
	}
}

func TestTaskGroupCurrentVisibleInGroupTaskBuildAndRun(t *testing.T) {
	src := `
func leaf() -> Int:
    return 1

func worker() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    return core.task_group_status(group)

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
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want visible open group status 1", exitCode)
	}
}

func TestTaskGroupCloseMarksOpenGroupClosedBuildAndRun(t *testing.T) {
	src := `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 70 + closeError
    return core.task_group_status(group)
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code = %d, want closed group status 3", exitCode)
	}
}

func TestTaskGroupClosePreservesCanceledStatusBuildAndRun(t *testing.T) {
	src := `
func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    group = core.task_group_cancel(group)
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 70 + closeError
    return core.task_group_status(group)
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want canceled group status 2", exitCode)
	}
}

func TestNestedTaskSpawnI32BuildAndRun(t *testing.T) {
	src := `
func child() -> Int:
    return 7

func worker() -> Int
uses runtime:
    let childTask: task.i32 = core.task_spawn_i32("child")
    let childResult: task.result_i32 = core.task_join_result_i32(childTask)
    if childResult.error != 0:
        return 90 + childResult.error
    return childResult.value

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    if result.error != 0:
        return 80 + result.error
    return result.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want nested child result 7", exitCode)
	}
}

func findIRFunc(t *testing.T, funcs []IRFunc, name string) IRFunc {
	t.Helper()
	for _, fn := range funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing IR function %q", name)
	return IRFunc{}
}

func hasIRCall(fn IRFunc, name string) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			return true
		}
	}
	return false
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func hasIRFuncPrefix(funcs []IRFunc, prefix string) bool {
	for _, fn := range funcs {
		if strings.HasPrefix(fn.Name, prefix) {
			return true
		}
	}
	return false
}
