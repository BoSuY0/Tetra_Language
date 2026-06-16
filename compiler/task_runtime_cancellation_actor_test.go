package compiler

import (
	"testing"
	"time"
)

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

func TestTaskSpawnsActorAndReceivesMailboxReplyBuildAndRun(t *testing.T) {
	src := `
func actor_worker() -> Int
uses actors:
    let _sent: Int = core.send(core.sender(), 6)
    return 0

func worker() -> Int
uses actors:
    let _actor: actor = core.spawn("actor_worker")
    return 1

func main() -> Int
uses actors, runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let reply: actor.recv_result_i32 = core.recv_until(core.deadline_ms(3))
    if reply.error != 0:
        return 40 + reply.error
    let result: task.result_i32 = core.task_join_result_i32(task)
    if result.error != 0:
        return 80 + result.error
    return result.value + reply.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task-spawned actor should reply to parent mailbox")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want task/actor mailbox reply result 7", exitCode)
	}
}

func TestTaskGroupCancelWakesActorRecvUntilBeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func actor_waiter() -> Int
uses actors, runtime:
    let result: actor.recv_result_i32 = core.recv_until(core.deadline_ms(100))
    let _sent: Int = core.send(core.sender(), result.error)
    return result.error

func worker() -> Int
uses actors:
    let _actor: actor = core.spawn("actor_waiter")
    return 1

func main() -> Int
uses actors, runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let launched: task.result_i32 = core.task_join_result_i32(task)
    if launched.error != 0:
        return 20 + launched.error
    if launched.value != 1:
        return 30 + launched.value
    let _delay: Int = core.sleep_ms(2)
    group = core.task_group_cancel(group)
    let reply: actor.recv_result_i32 = core.recv_until(core.deadline_ms(5))
    let _closed: Int = core.task_group_close(group)
    if reply.error != 0:
        return 40 + reply.error
    if reply.value != 1:
        return 60 + reply.value
    let now: Int = core.time_now_ms()
    if now != 2:
        return 80 + now
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task group cancel should wake actor recv_until before the original deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want actor recv_until canceled wake at logical time 2", exitCode)
	}
}

func TestTaskGroupCancelWakesActorRecvMsgUntilBeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func actor_waiter() -> Int
uses actors, runtime:
    let result: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(100))
    let _sent: Int = core.send_msg(core.sender(), result.error, result.tag)
    return result.error

func worker() -> Int
uses actors:
    let _actor: actor = core.spawn("actor_waiter")
    return 1

func main() -> Int
uses actors, runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let launched: task.result_i32 = core.task_join_result_i32(task)
    if launched.error != 0:
        return 20 + launched.error
    if launched.value != 1:
        return 30 + launched.value
    let _delay: Int = core.sleep_ms(2)
    group = core.task_group_cancel(group)
    let reply: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(5))
    let _closed: Int = core.task_group_close(group)
    if reply.error != 0:
        return 40 + reply.error
    if reply.value != 1:
        return 60 + reply.value
    if reply.tag != 0:
        return 70 + reply.tag
    let now: Int = core.time_now_ms()
    if now != 2:
        return 80 + now
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; task group cancel should wake actor recv_msg_until before the original deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want actor recv_msg_until canceled wake at logical time 2", exitCode)
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
